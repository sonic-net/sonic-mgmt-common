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

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

func TestSetGoRedisOpts(t *testing.T) {

	compareRedisOptsString2Struct(t, "ReadTimeout=10s",
		&redis.Options{ReadTimeout: 10 * time.Second})
	compareRedisOptsString2Struct(t, "ReadTimeout=10s,WriteTimeout=11s",
		&redis.Options{ReadTimeout: 10 * time.Second, WriteTimeout: 11 * time.Second})

}

func TestReadFromDBRedisOpts(t *testing.T) {

	compareRedisOptsDBRead2Struct(t, "ReadTimeout=10s",
		&redis.Options{ReadTimeout: 10 * time.Second})
	compareRedisOptsDBRead2Struct(t, "ReadTimeout=10s,WriteTimeout=11s",
		&redis.Options{ReadTimeout: 10 * time.Second, WriteTimeout: 11 * time.Second})

}

func compareRedisOptsString2Struct(t *testing.T, optsS string, opts *redis.Options) {
	setGoRedisOpts(optsS)
	if !reflect.DeepEqual(dbRedisOptsConfig.opts, *opts) {
		t.Errorf("SetGoRedisOpts() mismatch (%s) != %+v", optsS, opts)
		t.Errorf("New dbRedisOptsConfig.opts: %+v", dbRedisOptsConfig.opts)
	}
}

func compareRedisOptsDBRead2Struct(t *testing.T, optsS string, opts *redis.Options) {
	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Fatalf("NewDB() fails e = %v", e)
	}

	defer d.DeleteDB()

	// Do SetEntry
	ts := TableSpec{Name: "TRANSLIB_DB"}

	key := make([]string, 1, 1)
	key[0] = "default"
	akey := Key{Comp: key}

	if oldValue, ge := d.GetEntry(&ts, akey); ge == nil {
		defer d.SetEntry(&ts, akey, oldValue)
	}

	value := make(map[string]string, 1)
	value["go_redis_opts"] = optsS
	avalue := Value{Field: value}

	if e = d.SetEntry(&ts, akey, avalue); e != nil {
		t.Fatalf("SetEntry() fails e = %v", e)
	}

	t.Logf("TestReadFromDBRedisOpts: handleReconfigureSignal()")
	dbRedisOptsConfig.handleReconfigureSignal()
	dbRedisOptsConfig.reconfigure()
	if !reflect.DeepEqual(dbRedisOptsConfig.opts, *opts) {
		t.Errorf("reconfigure() mismatch (%s) != %+v", optsS, opts)
		t.Errorf("New dbRedisOptsConfig.opts: %+v", dbRedisOptsConfig.opts)
	}
}
