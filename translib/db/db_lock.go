////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//     http://www.apache.org/licenses/LICENSE-2.0                             //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package db

// DB Layer Lock

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"

	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
)

const (
	configDBLock   string = "config_db_lock"
	lockTable      string = "LOCK"
	lockKey        string = "translib"
	lockTableKey   string = lockTable + "|" + lockKey
	noSessionToken string = "0-0"

	tryLockAttempt int           = 4
	tryLockPause   time.Duration = 200
)

var execName string

type lockStruct struct {
	comm   string // Basename of the executable
	locked bool
}

type LockStruct struct {
	Name string // Lockname
	Id   string // ID Unique to the executable (Eg: Session-Token, "0-0")

	lockStruct
}

func (lt *LockStruct) tryLock() error {
	var err error
	var client *redis.Client
	var reply interface{}

	if (lt == nil) || lt.locked {
		err = tlerr.TranslibDBNotSupported{}
		glog.Errorf("tryLock: %v: %v", lt, err)
		return err
	}

	// Create The State DB Connection.
	if client, err = getStateDB(); err != nil {
		return err
	}
	defer client.Close()

	// HSETNX: Set Hash Field if Not Exist
	args := []interface{}{"HSETNX", lockTableKey, lt.Name,
		lt.comm + ":" + lt.Id}
	glog.Info("tryLock: RedisCmd: STATE_DB: ", args)
	if reply, err = client.Do(args...).Result(); err == nil {
		if intReply, ok := reply.(int64); !ok {
			glog.Errorf("tryLock: Reply %v Not int64: %v Type: %v",
				args, reply, reflect.TypeOf(reply))
			err = tlerr.TranslibDBScriptFail{Description: "Unexpected response"}
		} else if intReply == 0 {
			err = lt.dbLockedError(client)
		} else {
			lt.locked = true
			glog.Infof("tryLock: Locked: %s:%s", lt.Name, lt.Id)
		}
	}

	if err != nil {
		glog.Error("tryLock: Error", lt, err)
	}
	return err
}

func (lt *LockStruct) unlock() error {
	var err error
	var client *redis.Client
	var reply interface{}

	if (lt == nil) || !lt.locked {
		err = tlerr.TranslibDBNotSupported{}
		glog.Errorf("unlock: %v: %v", lt, err)
		return err
	}

	// Create The State DB Connection.
	if client, err = getStateDB(); err != nil {
		return err
	}
	defer client.Close()

	// Run the LUA Script to HDEL if we set the key
	if reply, err = luaScriptUnlock.Run(client,
		[]string{lockTableKey},
		[]string{lt.Name, lt.comm, lt.Id}).Result(); err == nil {

		if intReply, ok := reply.(int64); !ok {
			glog.Errorf("unlock: Reply %v Not int64: %v Type: %v",
				lt, reply, reflect.TypeOf(reply))
			err = tlerr.TranslibDBScriptFail{Description: "Unexpected response"}
		} else if intReply == 1 {
			lt.locked = false
			glog.Infof("unlock: Unlocked: %s:%s", lt.Name, lt.Id)
		} else {
			glog.Info("unlock: Already Unlocked")
			err = tlerr.TranslibDBLock{}
		}
	}

	if err != nil {
		// If this is an attempt to ClearLock(), do not log an error
		if _, ok := err.(tlerr.TranslibDBLock); !ok || (lt.Id != "*") {
			glog.Error("unlock: Error", lt, err)
		}
	}
	return err
}

func (lt *LockStruct) dbLockedError(c *redis.Client) error {
	var lockId string

	if lt.locked { // locked by self
		lockId = lt.Id
	} else if c == nil {
		glog.Warningf("dbLockedError: nil db connection; assuming generic lock")
	} else if v, err := c.HGet(lockTableKey, lt.Name).Result(); err != nil {
		glog.Warningf("dbLockedError: 'HGET %s %s' failed; err=%v", lockTableKey, lt.Name, err)
	} else if parts := strings.SplitN(v, ":", 2); len(parts) != 2 {
		glog.Warningf("dbLockedError: 'HGET %s %s' returned unknown value %q", lockTableKey, lt.Name, v)
	} else {
		lockId = parts[1]
	}

	lockType := tlerr.DBLockGeneric
	if len(lockId) != 0 && lockId != noSessionToken {
		lockType = tlerr.DBLockConfigSession
	}

	return tlerr.TranslibDBLock{Type: lockType}
}

var cdbLock *LockStruct

func ConfigDBTryLock(token string) error {
	var err error
	glog.Info("ConfigDBTryLock:")
	if bool(glog.V(3)) || !flag.Parsed() {
		dumpStack(7, 13) // Skip the stack frames upto NewDB()
	} else {
		dumpStack(9, 10)
	}

	// If len(token) == 0, this is not a configure session. (Eg: exec mode
	// configure replace)
	if cdbLock != nil {
		err = cdbLock.dbLockedError(nil)
	} else {
		ls := LockStruct{Name: configDBLock, Id: token,
			lockStruct: lockStruct{comm: execName}}
		for attempts := 0; attempts < tryLockAttempt; attempts++ {
			if err = ls.tryLock(); err == nil {
				cdbLock = &ls
				break
			} else if lErr, ok := err.(tlerr.TranslibDBLock); ok && lErr.Type == tlerr.DBLockConfigSession {
				break
			} else if (attempts + 1) == tryLockAttempt {
				break
			}
			glog.Infof("ConfigDBTryLock: Pausing %d ms", tryLockPause)
			time.Sleep(tryLockPause * time.Millisecond)
			glog.Infof("ConfigDBTryLock: Retrying Attempt %d", attempts)
		}
	}

	if err != nil {
		glog.Error("ConfigDBTryLock: Error", err)
	}
	return err
}

func ConfigDBUnlock(token string) error {
	var err error
	glog.Info("ConfigDBUnlock:")
	if bool(glog.V(3)) || !flag.Parsed() {
		dumpStack(7, 13) // Skip the stack frames upto DeleteDB()
	} else {
		dumpStack(9, 10)
	}

	if cdbLock == nil {
		err = tlerr.TranslibDBLock{}
	} else if cdbLock != nil {
		err = cdbLock.unlock()
		cdbLock = nil
	}

	if err != nil {
		glog.Error("ConfigDBUnlock: Error", err)
	}
	return err
}

func ConfigDBClearLock() error {
	var err error
	glog.Info("ConfigDBClearLock:")

	err = (&LockStruct{Name: configDBLock, Id: "*",
		lockStruct: lockStruct{comm: execName, locked: true}}).unlock()
	cdbLock = nil

	// Clearing an absent lock is ok.
	if _, ok := err.(tlerr.TranslibDBLock); ok {
		glog.Info("ConfigDBClearLock: Lock Absent")
		err = nil
	}
	return err
}

///////////////////////////////////////////////////////////////////////////////
// Internal Functions                                                        //
///////////////////////////////////////////////////////////////////////////////

func dumpStack(begin, end int) {
	stackL := strings.SplitN(string(debug.Stack()), "\n", end+1)
	for idx, line := range stackL {
		if idx < begin {
			continue
		}
		if idx == end {
			break
		}
		glog.Infof("dS:[%d]:[%s]", idx, line)
	}
}

func getStateDB() (*redis.Client, error) {
	var client *redis.Client
	var err error
	if client = redis.NewClient(adjustRedisOpts(&Options{
		DBNo: StateDB})); client == nil {

		glog.Error("getStateDB: Could not create redis client: STATE_DB")
		err = tlerr.TranslibDBCannotOpen{}
	}
	return client, err
}

func getExecName() string {
	return filepath.Base(os.Args[0])
}

var luaScriptUnlock *redis.Script

func init() {

	// Executable Name
	execName = getExecName()

	// Register the Lua Script. Only Unlock if the Hash Field Value matches
	// i.e. if HGET KEYS[1] ARGV[1] == ARGV[2]:ARGV[3], ARGV[2],[3] could be *
	luaScriptUnlock = redis.NewScript(`
		local fieldVal = redis.call("HGET", KEYS[1], ARGV[1])
		if fieldVal then
			local slen = string.len(fieldVal)
			local colon = string.find(fieldVal, ':', 1, true)
			if not colon then
				colon = slen + 1
			end
			local comm = string.sub(fieldVal, 1, colon - 1)
			local id = string.sub(fieldVal, colon + 1, slen)
			if ((ARGV[2] == '*') or (ARGV[2] == comm)) and
					((ARGV[3] == '*') or (ARGV[3] == id)) then
				return redis.call("HDEL", KEYS[1], ARGV[1])
			end
		end
		return 0
	`)

	// Clears the ConfigDB Lock, (if the current executable placed it, i.e.
	// RESTCONF/rest-server clears it's lock, and gNMI/telemetry clears
	// it's lock).
	ConfigDBClearLock()
}
