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

// ExistKeysPattern checks if a key pattern exists in a table
// Note:
//  1. Statistics do not capture when the CAS Tx Cache results in a quicker
//     response. This is to avoid mis-interpreting per-connection statistics.
func (d *DB) ExistKeysPattern(ts *TableSpec, pat Key) (bool, error) {

	var err error
	var exists bool

	// ExistsKeysPatternHits
	// Time Start
	var cacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats

	if (d == nil) || (d.client == nil) {
		return exists, tlerr.TranslibDBConnectionReset{}
	}

	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	if glog.V(3) {
		glog.Info("ExistKeysPattern: Begin: ", "ts: ", ts, "pat: ", pat)
	}

	defer func() {
		if err != nil {
			glog.Error("ExistKeysPattern: ts: ", ts, " err: ", err)
		}
		if glog.V(3) {
			glog.Info("ExistKeysPattern: End: ts: ", ts, " exists: ", exists)
		}
	}()

	// If pseudoDB then follow the path of !IsWriteDisabled with Tx Cache. TBD.

	if !d.Opts.IsWriteDisabled {

		// If Write is enabled, then just call GetKeysPattern() and check
		// for now.

		// Optimization: Check the DBL CAS Tx Cache for added entries.
		for k := range d.txTsEntryMap[ts.Name] {
			key := d.redis2key(ts, k)
			if key.Matches(pat) {
				if len(d.txTsEntryMap[ts.Name][k].Field) > 0 {
					exists = true
					break
				}
				// Removed entries/fields, can't help much, since we'd have
				// to compare against the DB retrieved keys anyway
			}
		}

		if !exists {
			var getKeys []Key
			if getKeys, err = d.GetKeysPattern(ts, pat); (err == nil) && len(getKeys) > 0 {

				exists = true
			}
		}

	} else if d.dbCacheConfig.PerConnection &&
		d.dbCacheConfig.isCacheTable(ts.Name) {

		// Check PerConnection cache first, [Found = SUCCESS return]
		var keys []Key
		if table, ok := d.cache.Tables[ts.Name]; ok {
			if keys, ok = table.patterns[d.key2redis(ts, pat)]; ok && len(keys) > 0 {

				exists = true
				cacheHit = true

			}
		}
	}

	// Run Lua script [Found = SUCCESS return]
	if d.Opts.IsWriteDisabled && !exists {

		//glog.Info("ExistKeysPattern: B4= ", luaScriptExistsKeysPatterns.Hash())

		var luaExists interface{}
		if luaExists, err = luaScriptExistsKeysPatterns.Run(d.client,
			[]string{d.key2redis(ts, pat)}).Result(); err == nil {

			if existsString, ok := luaExists.(string); !ok {
				err = tlerr.TranslibDBScriptFail{
					Description: "Unexpected response"}
			} else if existsString == "true" {
				exists = true
			} else if existsString != "false" {
				err = tlerr.TranslibDBScriptFail{Description: existsString}
			}
		}

		//glog.Info("ExistKeysPattern: AF= ", luaScriptExistsKeysPatterns.Hash())
	}

	// Time End, Time, Peak
	if d.dbStatsConfig.TableStats {
		stats = d.stats.Tables[ts.Name]
	} else {
		stats = d.stats.AllTables
	}

	stats.Hits++
	stats.ExistsKeyPatternHits++
	if cacheHit {
		stats.ExistsKeyPatternCacheHits++
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.ExistsKeyPatternPeak {
			stats.ExistsKeyPatternPeak = dur
		}
		stats.ExistsKeyPatternTime += dur
	}

	if d.dbStatsConfig.TableStats {
		d.stats.Tables[ts.Name] = stats
	} else {
		d.stats.AllTables = stats
	}

	return exists, err
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

var luaScriptExistsKeysPatterns *redis.Script

func init() {
	// Register the Lua Script
	luaScriptExistsKeysPatterns = redis.NewScript(`
		for i,k in pairs(redis.call('KEYS', KEYS[1])) do
			return 'true'
		end
		return 'false'
	`)

	// Alternate Lua Script
	// luaScriptExistsKeysPatterns = redis.NewScript(`
	// 	if #redis.call('KEYS', KEYS[1]) > 0 then
	// 		return 'true'
	// 	else
	// 		return 'false'
	// 	end
	// `)

}
