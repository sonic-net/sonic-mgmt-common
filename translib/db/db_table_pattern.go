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
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// GetTablePattern is similar to GetTable, except, it gets a subset of
// the table restricted by the key pattern "pat". It is expected that there
// will be a single call to the underlying go-redis layer, perhaps through
// the use of a script. However, the caller is not advised to rely on this,
// and encouraged to use this instead of the current GetTable() for reasonably
// sized subsets of tables ("reasonably sized" to be determined by experiments)
// to obtain a more performant behavior.
// Note: The time/hit statistics do not cover the error cases.
func (d *DB) GetTablePattern(ts *TableSpec, pat Key) (Table, error) {

	var err error
	var ok bool
	var keys []Key
	var redisKey string
	var redisValue []interface{}
	var tkNv []interface{}
	var luaTable interface{}

	// GetTablePatternHits
	// Time Start
	var cacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats

	if (d == nil) || (d.client == nil) {
		return Table{}, tlerr.TranslibDBConnectionReset{}
	}

	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	if glog.V(3) {
		glog.Info("GetTablePattern: Begin: ts: ", ts, " pat: ", pat)
	}

	defer func() {
		if err != nil {
			glog.Error("GetTablePattern: ts: ", ts, " err: ", err)
		}
		if glog.V(3) {
			glog.Info("GetTablePattern: End: ts: ", ts)
		}
	}()

	isComplete := false
	if pat.IsAllKeyPattern() {
		isComplete = true
	}
	// Create Table
	table := Table{
		ts:       ts,
		entry:    make(map[string]Value, InitialTableEntryCount),
		complete: isComplete,
		patterns: make(map[string][]Key, InitialTablePatternCount),
		db:       d,
	}

	// Check Per Connection Cache first.
	if (d.dbCacheConfig.PerConnection &&
		d.dbCacheConfig.isCacheTable(ts.Name)) ||
		(d.Opts.IsOnChangeEnabled && d.onCReg.isCacheTable(ts.Name)) {

		if cTable, ok := d.cache.Tables[ts.Name]; ok && cTable.complete {

			// Copy relevent keys of cTable to table, and set keys
			// Be aware of cache poisoning
			for redisKey, value := range cTable.entry {
				key := d.redis2key(ts, redisKey)
				// {Comp: []string{"*"}} matches {Comp: []string{"1", "2"}}
				// However, Key.Matches() goes strictly by len(Comp)
				if isComplete || key.Matches(pat) {
					table.entry[redisKey] = value.Copy()
					keys = append(keys, key)
				}
			}
			table.patterns[d.key2redis(ts, pat)] = keys
			cacheHit = true

			goto GetTablePatternFoundCache
		}
	}

	// Run the Lua script
	luaTable, err = luaScriptGetTable.Run(d.client,
		[]string{d.key2redis(ts, pat)}).Result()
	if err != nil {
		return table, err
	}

	// Walk through the results
	//     Initialize table.patterns
	//     Set table.entry entry

	tkNv, ok = luaTable.([]interface{})

	if !ok {
		err = tlerr.TranslibDBScriptFail{Description: "Unexpected list"}
		return table, err
	}

	keys = make([]Key, 0, len(tkNv)/2)
	for i, v := range tkNv {
		if i%2 == 0 {
			if redisKey, ok = v.(string); !ok {
				err = tlerr.TranslibDBScriptFail{Description: "Unexpected key"}
				return table, err
			}
		} else {
			if redisValue, ok = v.([]interface{}); !ok {
				err = tlerr.TranslibDBScriptFail{Description: "Unexpected hash"}
				return table, err
			}
			value := Value{Field: make(map[string]string, len(redisValue)/2)}
			var fstr, fn string
			for j, f := range redisValue {
				if fstr, ok = f.(string); !ok {
					err = tlerr.TranslibDBScriptFail{Description: "Unexpected field"}
					return table, err
				}
				if j%2 == 0 {
					fn = fstr
				} else {
					value.Field[fn] = fstr
				}
			}
			table.entry[redisKey] = value
			keys = append(keys, d.redis2key(ts, redisKey))
		}
	}

GetTablePatternFoundCache:

	// Populate the PerConnection cache, if enabled, allKeyPat, and cacheMiss
	if !cacheHit && isComplete &&
		((d.dbCacheConfig.PerConnection &&
			d.dbCacheConfig.isCacheTable(ts.Name)) ||
			(d.Opts.IsOnChangeEnabled && d.onCReg.isCacheTable(ts.Name))) {

		if _, ok := d.cache.Tables[ts.Name]; !ok {
			d.cache.Tables[ts.Name] = Table{
				ts:       ts,
				entry:    make(map[string]Value, InitialTableEntryCount),
				complete: table.complete,
				patterns: make(map[string][]Key, InitialTablePatternCount),
				db:       d,
			}
		}

		// Note: keys (and thus keysCopy stored in the PerConnection cache)
		// should be *before* adjusting with Redis CAS Tx Cache
		keysCopy := make([]Key, len(keys))
		for i, key := range keys {
			keysCopy[i] = key.Copy()
		}
		d.cache.Tables[ts.Name].patterns[d.key2redis(ts, pat)] = keysCopy

		for redisKey, value := range table.entry {
			d.cache.Tables[ts.Name].entry[redisKey] = value.Copy()
		}
	}

	// Reconcile with Redis CAS Transaction cache
	for k := range d.txTsEntryMap[ts.Name] {
		var present bool
		var index int
		key := d.redis2key(ts, k)

		if !isComplete && !key.Matches(pat) {
			continue
		}

		for i := 0; i < len(keys); i++ {
			if key.Equals(keys[i]) {
				index = i
				present = true
				break
			}
		}

		if !present {
			if len(d.txTsEntryMap[ts.Name][k].Field) > 0 {
				keys = append(keys, key)
				table.entry[k] = (d.txTsEntryMap[ts.Name][k]).Copy()
			}
		} else {
			if len(d.txTsEntryMap[ts.Name][k].Field) == 0 {
				keys = append(keys[:index], keys[index+1:]...)
				delete(table.entry, k)
			} else {
				table.entry[k] = (d.txTsEntryMap[ts.Name][k]).Copy()
			}
		}
	}

	table.patterns[d.key2redis(ts, pat)] = keys

	// Time End, Time, Peak
	if d.dbStatsConfig.TableStats {
		stats = d.stats.Tables[ts.Name]
	} else {
		stats = d.stats.AllTables
	}

	stats.Hits++
	stats.GetTablePatternHits++
	if cacheHit {
		stats.GetTablePatternCacheHits++
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetTablePatternPeak {
			stats.GetTablePatternPeak = dur
		}
		stats.GetTablePatternTime += dur
	}

	if d.dbStatsConfig.TableStats {
		d.stats.Tables[ts.Name] = stats
	} else {
		d.stats.AllTables = stats
	}

	return table, err
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

var luaScriptGetTable *redis.Script

func init() {
	// Lua Script:
	//
	// Key1  Value1              Key2  Value2      ...
	//       f11 v11 f12 v12 ...       f21 v21 ...

	luaScriptGetTable = redis.NewScript(`
		local tkNv = {}
		for i,k in pairs(redis.call('KEYS', KEYS[1])) do
			tkNv[2*i - 1], tkNv[2*i] = k, redis.call('HGETALL', k)
		end
		return tkNv
	`)

}
