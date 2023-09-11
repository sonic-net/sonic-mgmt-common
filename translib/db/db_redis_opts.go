///////////////////////////////////////////////////////////////////////////////
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

import (
	"errors"
	"flag"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Internal Types                                                            //
////////////////////////////////////////////////////////////////////////////////

type _DBRedisOptsConfig struct {
	opts redis.Options
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

var dbRedisOptsConfig *_DBRedisOptsConfig = &_DBRedisOptsConfig{}

var reconfigureRedisOptsConfig = true // Signal Received, or initialization

var mutexRedisOptsConfig sync.Mutex

var goRedisOpts string // Command line options

var goRedisOptsOnce sync.Once // Command line options handled only once

func setGoRedisOpts(optsString string) {
	// Command Line Options have higher priority on startup. After that, on
	// receipt of a signal (SIGUSR2?), the TRANSLIB_DB|default
	// "go_redis_opts" will have higher priority.
	if optsString != "" {
		glog.Infof("setGoRedisOpts: optsString: %s", optsString)
		// On startup, command-line has priority. Skip reconfigure from DB
		reconfigureRedisOptsConfig = false
		dbRedisOptsConfig.parseRedisOptsConfig(optsString)
	}
}

// adjustRedisOpts() gets the redis.Options to be set based on command line
// options, values passed via db Options, and TRANSLIB_DB|default settings.
// Additionally it also adjusts the passed dbOpts for separator.
func adjustRedisOpts(dbOpt *Options) *redis.Options {
	dbRedisOptsConfig.reconfigure()
	mutexRedisOptsConfig.Lock()
	redisOpts := dbRedisOptsConfig.opts
	mutexRedisOptsConfig.Unlock()

	var dbSock string
	var dbNetwork string
	addr := DefaultRedisLocalTCPEP
	dbId := int(dbOpt.DBNo)
	dbPassword := ""
	if dbInstName := getDBInstName(dbOpt.DBNo); dbInstName != "" {
		if isDbInstPresent(dbInstName) {
			if dbSock = getDbSock(dbInstName); dbSock != "" {
				dbNetwork = DefaultRedisUNIXNetwork
				addr = dbSock
			} else {
				dbNetwork = DefaultRedisTCPNetwork
				addr = getDbTcpAddr(dbInstName)
			}
			dbId = getDbId(dbInstName)
			dbSepStr := getDbSeparator(dbInstName)
			dbPassword = getDbPassword(dbInstName)
			if len(dbSepStr) > 0 {
				if len(dbOpt.TableNameSeparator) > 0 &&
					dbOpt.TableNameSeparator != dbSepStr {
					glog.Warningf("TableNameSeparator '%v' in"+
						" the Options is different from the"+
						" one configured in the Db config. file for the"+
						" Db name %v", dbOpt.TableNameSeparator, dbInstName)
				}
				dbOpt.KeySeparator = dbSepStr
				dbOpt.TableNameSeparator = dbSepStr
			} else {
				glog.Warning("Database Separator not present for the Db name: ",
					dbInstName)
			}
		} else {
			glog.Warning("Database instance not present for the Db name: ",
				dbInstName)
		}
	} else {
		glog.Errorf("Invalid database number %d", dbId)
	}

	redisOpts.Network = dbNetwork
	redisOpts.Addr = addr
	redisOpts.Password = dbPassword
	redisOpts.DB = dbId

	// redisOpts.DialTimeout = 0 // Default

	// Default 3secs read & write timeout was not sufficient in high CPU load
	// on certain platforms. Hence increasing to 10secs. (via command-line
	// options). Setting read-timeout is sufficient; internally go-redis
	// updates the same for write-timeout as well.
	// redisOpts.ReadTimeout = 10 * time.Second // Done via command-line

	// For Transactions, limit the pool, if the options haven't over-ridden it.
	if redisOpts.PoolSize == 0 {
		redisOpts.PoolSize = 1
	}
	// Each DB gets it own (single) connection.

	return &redisOpts
}

func init() {
	flag.StringVar(&goRedisOpts, "go_redis_opts", "", "Options for go-redis")
}

////////////////////////////////////////////////////////////////////////////////
//  Configure DB Redis Opts                                                   //
////////////////////////////////////////////////////////////////////////////////

func (config *_DBRedisOptsConfig) reconfigure() error {

	mutexRedisOptsConfig.Lock()
	// Handle command line options after they are parsed.
	if flag.Parsed() {
		goRedisOptsOnce.Do(func() {
			setGoRedisOpts(goRedisOpts)
		})
	} else {
		glog.Warningf("_DBRedisOptsConfig:reconfigure: flags not parsed!")
	}

	var doReconfigure bool = reconfigureRedisOptsConfig
	if reconfigureRedisOptsConfig {
		reconfigureRedisOptsConfig = false
	}
	mutexRedisOptsConfig.Unlock()

	if doReconfigure {
		glog.Infof("_DBRedisOptsConfig:reconfigure: Handling signal.")
		var readDBRedisOptsConfig _DBRedisOptsConfig
		readDBRedisOptsConfig.readFromDB()

		mutexRedisOptsConfig.Lock()
		if !reflect.DeepEqual(*config, readDBRedisOptsConfig) {
			glog.Infof("_DBRedisOptsConfig:reconfigure: Change Detected.")
			dbRedisOptsConfig = &readDBRedisOptsConfig
		}
		mutexRedisOptsConfig.Unlock()
	}
	return nil
}

func (config *_DBRedisOptsConfig) handleReconfigureSignal() error {
	mutexRedisOptsConfig.Lock()
	reconfigureRedisOptsConfig = true
	mutexRedisOptsConfig.Unlock()
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//  Read DB Redis Options Configuration                                       //
////////////////////////////////////////////////////////////////////////////////

func (config *_DBRedisOptsConfig) readFromDB() error {
	fields, e := readRedis("TRANSLIB_DB|default")
	if e == nil {
		if optsString, ok := fields["go_redis_opts"]; ok {
			// Parse optsString into config.opts
			config.parseRedisOptsConfig(optsString)
		}
	}
	return e
}

func (config *_DBRedisOptsConfig) parseRedisOptsConfig(optsString string) error {
	var e, optSAErr error
	var intVal int64
	var eS string

	glog.Infof("parseRedisOptsConfig: optsString: %s", optsString)

	// First zero the config redis.Options, in case there is any existing
	// stale configuration.
	config.opts = redis.Options{}

	// This could be optimized using reflection, if the # of options grows
	for optI, optS := range strings.Split(optsString, ",") {
		glog.Infof("parseRedisOptsConfig: optI: %d optS: %s", optI, optS)
		if optSA := strings.Split(optS, "="); len(optSA) > 1 {
			switch optSA[0] {
			case "MaxRetries":
				if intVal, optSAErr = strconv.ParseInt(optSA[1], 0, 64); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				} else {
					config.opts.MaxRetries = int(intVal)
				}
			case "MinRetryBackoff":
				if config.opts.MinRetryBackoff, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "MaxRetryBackoff":
				if config.opts.MaxRetryBackoff, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "DialTimeout":
				if config.opts.DialTimeout, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "ReadTimeout":
				if config.opts.ReadTimeout, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "WriteTimeout":
				if config.opts.WriteTimeout, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "PoolSize":
				if intVal, optSAErr = strconv.ParseInt(optSA[1], 0, 64); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				} else {
					config.opts.PoolSize = int(intVal)
				}
			case "MinIdleConns":
				if intVal, optSAErr = strconv.ParseInt(optSA[1], 0, 64); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				} else {
					config.opts.MinIdleConns = int(intVal)
				}
			case "MaxConnAge":
				if config.opts.MaxConnAge, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "PoolTimeout":
				if config.opts.PoolTimeout, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "IdleTimeout":
				if config.opts.IdleTimeout, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			case "IdleCheckFrequency":
				if config.opts.IdleCheckFrequency, optSAErr =
					time.ParseDuration(optSA[1]); optSAErr != nil {
					eS += ("Parse Error: " + optSA[0] + " :" + optSAErr.Error())
				}
			default:
				eS += ("Unknown Redis Option: " + optSA[0] + " ")
			}
		}
	}

	if len(eS) != 0 {
		glog.Errorf("parseRedisOptsConfig: Unknown: %s", eS)
		e = errors.New(eS)
	}

	return e
}
