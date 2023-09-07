////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

/*
Package db implements a wrapper over the go-redis/redis.
*/
package db

import (
	"time"

	"github.com/golang/glog"
	// "github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

func init() {
}

func (d *DB) GetMap(ts *TableSpec, mapKey string) (string, error) {

	if glog.V(3) {
		glog.Info("GetMap: Begin: ", "ts: ", ts, " mapKey: ", mapKey)
	}

	if (d == nil) || (d.client == nil) {
		return "", tlerr.TranslibDBConnectionReset{}
	}

	// GetMapHits
	// Time Start
	var cacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats
	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	var mAP MAP
	var e error
	var v string

	// If pseudoDB then do custom. TBD.

	// If cache GetFromCache (CacheHit?)
	if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheMap(ts.Name) {
		var ok bool
		if mAP, ok = d.cache.Maps[ts.Name]; ok {
			if v, ok = mAP.mapMap[mapKey]; ok {
				cacheHit = true
			}
		}
	}

	if !cacheHit {

		glog.Info("GetMap: RedisCmd: ", d.Name(), ": ", "HGET ", ts.Name,
			mapKey)
		v, e = d.client.HGet(ts.Name, mapKey).Result()

		// If cache SetCache (i.e. a cache miss)
		if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheMap(ts.Name) {
			if _, ok := d.cache.Maps[ts.Name]; !ok {
				d.cache.Maps[ts.Name] = MAP{
					ts:       ts,
					complete: false,
					mapMap:   make(map[string]string, InitialMapKeyCount),
					db:       d,
				}
			}
			d.cache.Maps[ts.Name].mapMap[mapKey] = v
		}

	}

	// Time End, Time, Peak
	if d.dbStatsConfig.MapStats {
		stats = d.stats.Maps[ts.Name]
	} else {
		stats = d.stats.AllMaps
	}

	stats.Hits++
	stats.GetMapHits++
	if cacheHit {
		stats.GetMapCacheHits++
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetMapPeak {
			stats.GetMapPeak = dur
		}
		stats.GetMapTime += dur
	}

	if d.dbStatsConfig.MapStats {
		d.stats.Maps[ts.Name] = stats
	} else {
		d.stats.AllMaps = stats
	}

	if glog.V(3) {
		glog.Info("GetMap: End: ", "v: ", v, " e: ", e)
	}

	return v, e
}

func (d *DB) GetMapAll(ts *TableSpec) (Value, error) {

	if glog.V(3) {
		glog.Info("GetMapAll: Begin: ", "ts: ", ts)
	}

	if (d == nil) || (d.client == nil) {
		return Value{}, tlerr.TranslibDBConnectionReset{}
	}

	// GetMapAllHits
	// Time Start
	var cacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats
	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	var mAP MAP
	var e error
	var value Value
	var v map[string]string

	// If pseudoDB then do custom. TBD.

	// If cache GetFromCache (CacheHit?)
	if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheMap(ts.Name) {
		var ok bool
		if mAP, ok = d.cache.Maps[ts.Name]; ok {
			if mAP.complete {
				cacheHit = true
				value = Value{Field: mAP.mapMap}
			}
		}
	}

	if !cacheHit {

		glog.Info("GetMapAll: RedisCmd: ", d.Name(), ": ", "HGETALL ", ts.Name)
		v, e = d.client.HGetAll(ts.Name).Result()

		if len(v) != 0 {

			value = Value{Field: v}

			// If cache SetCache (i.e. a cache miss)
			if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheMap(ts.Name) {
				d.cache.Maps[ts.Name] = MAP{
					ts:       ts,
					complete: true,
					mapMap:   v,
					db:       d,
				}
			}

		} else {
			if glog.V(1) {
				glog.Info("GetMapAll: HGetAll(): empty map")
			}

			if e != nil {
				glog.Error("GetMapAll: ", d.Name(),
					": HGetAll(", ts.Name, "): error: ", e.Error())
			} else {
				e = tlerr.TranslibRedisClientEntryNotExist{Entry: ts.Name}
			}
		}

	}

	// Time End, Time, Peak
	if d.dbStatsConfig.MapStats {
		stats = d.stats.Maps[ts.Name]
	} else {
		stats = d.stats.AllMaps
	}

	stats.Hits++
	stats.GetMapAllHits++
	if cacheHit {
		stats.GetMapAllCacheHits++
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetMapAllPeak {
			stats.GetMapAllPeak = dur
		}
		stats.GetMapAllTime += dur
	}

	if d.dbStatsConfig.MapStats {
		d.stats.Maps[ts.Name] = stats
	} else {
		d.stats.AllMaps = stats
	}

	if glog.V(3) {
		glog.Info("GetMapAll: End: ", "value: ", value, " e: ", e)
	}

	return value, e
}

// For Testing only. Do Not Use!!! ==============================

// SetMap - There is no transaction support on these.
func (d *DB) SetMap(ts *TableSpec, mapKey string, mapValue string) error {

	if glog.V(3) {
		glog.Info("SetMap: Begin: ", "ts: ", ts, " ", mapKey,
			":", mapValue)
	}

	b, e := d.client.HSet(ts.Name, mapKey, mapValue).Result()

	if glog.V(3) {
		glog.Info("GetMap: End: ", "b: ", b, " e: ", e)
	}

	return e
}

// For Testing only. Do Not Use!!! ==============================

// DeleteMapAll - There is no transaction support on these.
// TBD: Unexport this. : Lower case it, and reference too
func (d *DB) DeleteMapAll(ts *TableSpec) error {

	if glog.V(3) {
		glog.Info("DeleteMapAll: Begin: ", "ts: ", ts)
	}

	e := d.client.Del(ts.Name).Err()

	if glog.V(3) {
		glog.Info("DeleteMapAll: End: ", " e: ", e)
	}

	return e
}

// For Testing only. Do Not Use!!! ==============================
