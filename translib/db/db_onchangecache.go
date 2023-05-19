////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Internal Types                                                            //
////////////////////////////////////////////////////////////////////////////////

type dbOnChangeReg struct {
	CacheTables map[string]bool // Only cache these tables.
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func (d *DB) RegisterTableForOnChangeCaching(ts *TableSpec) error {
	if glog.V(3) {
		glog.Info("RegisterTableForOnChange: ts:", ts)
	}
	if !d.IsOpen() {
		return ConnectionClosed
	}
	if !d.Opts.IsOnChangeEnabled {
		return OnChangeDisabled
	}

	d.onCReg.CacheTables[ts.Name] = true
	return nil
}

// OnChangeCacheUpdate reads a db entry from redis and updates the on_change cache.
// Returns both previously cached Value and current Value. Previous Value will be
// empty if there was no such cache entry earlier. Returns an error if DB entry
// does not exists or could not be read.
func (d *DB) OnChangeCacheUpdate(ts *TableSpec, key Key) (Value, Value, error) {
	if glog.V(3) {
		glog.Info("OnChangeCacheUpdate: Begin: ", "ts: ", ts, " key: ", key)
	}
	if !d.IsOpen() {
		return Value{}, Value{}, ConnectionClosed
	}
	if !d.Opts.IsOnChangeEnabled {
		return Value{}, Value{}, OnChangeDisabled
	}

	var valueOrig Value
	if _, ok := d.cache.Tables[ts.Name]; ok {
		valueOrig = d.cache.Tables[ts.Name].entry[d.key2redis(ts, key)]
	}

	// Get New Value from the DB
	value, e := d.getEntry(ts, key, true)

	return valueOrig, value, e
}

// OnChangeCacheDelete deletes an entry from the on_change cache.
// Returns the previously cached Value object; or an empty Value if there was
// no such cache entry.
func (d *DB) OnChangeCacheDelete(ts *TableSpec, key Key) (Value, error) {
	if glog.V(3) {
		glog.Info("OnChangeCacheDelete: Begin: ", "ts:", ts, " key:", key)
	}
	if !d.IsOpen() {
		return Value{}, ConnectionClosed
	}
	if !d.Opts.IsOnChangeEnabled {
		return Value{}, OnChangeDisabled
	}

	redisKey := d.key2redis(ts, key)
	var valueOrig Value
	_, ok := d.cache.Tables[ts.Name]
	if ok {
		valueOrig, ok = d.cache.Tables[ts.Name].entry[redisKey]
	}

	if ok {
		glog.V(2).Info("OnChangeCacheDelete: Delete ts:", ts, " key:", key)
		delete(d.cache.Tables[ts.Name].entry, redisKey)
	} else {
		glog.V(2).Info("OnChangeCacheDelete: Not found; ts:", ts, " key:", key)
	}

	return valueOrig, nil
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func init() {
}

func (reg *dbOnChangeReg) isCacheTable(name string) bool {
	return reg.CacheTables[name]
}

func (reg *dbOnChangeReg) size() int {
	return len(reg.CacheTables)
}
