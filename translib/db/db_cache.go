////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	// "fmt"
	// "strconv"

	// "errors"
	"reflect"
	"strings"
	"sync"
	// "github.com/Azure/sonic-mgmt-common/cvl"
	// "github.com/go-redis/redis/v7"
	// "github.com/golang/glog"
	// "github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

type dbCache struct {
	Tables map[string]Table
	Maps   map[string]MAP
}

type DBGlobalCache struct {
	Databases [MaxDB]dbCache
}

type DBCacheConfig struct {
	PerConnection bool            // Enable per DB conn cache
	Global        bool            // Enable global cache (TBD)
	CacheTables   map[string]bool // Only cache these tables.
	// Empty == Cache all tables
	NoCacheTables map[string]bool // Do not cache these tables.
	// "all" == Do not cache any tables
	CacheMaps map[string]bool // Only cache these maps.
	// Empty == Cache all maps
	NoCacheMaps map[string]bool // Do not cache these maps
	// "all" == Do not cache any maps
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func ReconfigureCache() error {
	return dbCacheConfig.reconfigure()
}

func ClearCache() error {
	return nil // TBD for Global Cache
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

var dbCacheConfig *DBCacheConfig
var defaultDBCacheConfig DBCacheConfig = DBCacheConfig{
	PerConnection: false,
	Global:        false,
}
var reconfigureCacheConfig bool
var mutexCacheConfig sync.Mutex

// var zeroDBCache = &DBCache{}

func init() {
	dbCacheConfig = &DBCacheConfig{}
	dbCacheConfig.handleReconfigureSignal()
	dbCacheConfig.reconfigure()
}

////////////////////////////////////////////////////////////////////////////////
//  Configure DB Cache                                                        //
////////////////////////////////////////////////////////////////////////////////

func getDBCacheConfig() DBCacheConfig {

	dbCacheConfig.reconfigure()

	mutexCacheConfig.Lock()

	cacheConfig := DBCacheConfig{
		CacheTables:   make(map[string]bool, len(dbCacheConfig.CacheTables)),
		NoCacheTables: make(map[string]bool, len(dbCacheConfig.NoCacheTables)),
		CacheMaps:     make(map[string]bool, len(dbCacheConfig.CacheMaps)),
		NoCacheMaps:   make(map[string]bool, len(dbCacheConfig.NoCacheMaps)),
	}

	cacheConfig.PerConnection = dbCacheConfig.PerConnection
	cacheConfig.Global = dbCacheConfig.Global

	for k, v := range dbCacheConfig.CacheTables {
		cacheConfig.CacheTables[k] = v
	}

	for k, v := range dbCacheConfig.NoCacheTables {
		cacheConfig.NoCacheTables[k] = v
	}

	for k, v := range dbCacheConfig.CacheMaps {
		cacheConfig.CacheMaps[k] = v
	}

	for k, v := range dbCacheConfig.NoCacheMaps {
		cacheConfig.NoCacheMaps[k] = v
	}

	mutexCacheConfig.Unlock()

	return cacheConfig
}

func (config *DBCacheConfig) reconfigure() error {
	mutexCacheConfig.Lock()
	var doReconfigure bool = reconfigureCacheConfig
	if reconfigureCacheConfig {
		reconfigureCacheConfig = false
	}
	mutexCacheConfig.Unlock()

	if doReconfigure {
		var readDBCacheConfig DBCacheConfig
		readDBCacheConfig.readFromDB()

		mutexCacheConfig.Lock()
		configChanged := !reflect.DeepEqual(*dbCacheConfig, readDBCacheConfig)
		mutexCacheConfig.Unlock()

		if configChanged {
			ClearCache()
		}

		mutexCacheConfig.Lock()
		dbCacheConfig = &readDBCacheConfig
		mutexCacheConfig.Unlock()
	}
	return nil
}

func (config *DBCacheConfig) handleReconfigureSignal() error {
	mutexCacheConfig.Lock()
	reconfigureCacheConfig = true
	mutexCacheConfig.Unlock()
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//  Read DB Cache Configuration                                               //
////////////////////////////////////////////////////////////////////////////////

func (config *DBCacheConfig) readFromDB() error {
	fields, e := readRedis("TRANSLIB_DB|default")
	if e != nil {

		config.PerConnection = defaultDBCacheConfig.PerConnection
		config.Global = defaultDBCacheConfig.Global
		config.CacheTables = make(map[string]bool,
			len(defaultDBCacheConfig.CacheTables))
		for k, v := range defaultDBCacheConfig.CacheTables {
			config.CacheTables[k] = v
		}

		config.NoCacheTables = make(map[string]bool,
			len(defaultDBCacheConfig.NoCacheTables))
		for k, v := range defaultDBCacheConfig.NoCacheTables {
			config.NoCacheTables[k] = v
		}

		config.CacheMaps = make(map[string]bool,
			len(defaultDBCacheConfig.CacheMaps))
		for k, v := range defaultDBCacheConfig.CacheMaps {
			config.CacheMaps[k] = v
		}

		config.NoCacheMaps = make(map[string]bool,
			len(defaultDBCacheConfig.NoCacheMaps))
		for k, v := range defaultDBCacheConfig.NoCacheMaps {
			config.NoCacheMaps[k] = v
		}

	} else {
		for k, v := range fields {
			switch {
			case k == "per_connection_cache" && v == "True":
				config.PerConnection = true
			case k == "per_connection_cache" && v == "False":
				config.PerConnection = false
			case k == "global_cache" && v == "True":
				config.Global = true
			case k == "global_cache" && v == "False":
				config.Global = false
			case k == "@tables_cache":
				l := strings.Split(v, ",")
				config.CacheTables = make(map[string]bool, len(l))
				for _, t := range l {
					config.CacheTables[t] = true
				}
			case k == "@no_tables_cache":
				l := strings.Split(v, ",")
				config.NoCacheTables = make(map[string]bool, len(l))
				for _, t := range l {
					config.NoCacheTables[t] = true
				}
			case k == "@maps_cache":
				l := strings.Split(v, ",")
				config.CacheMaps = make(map[string]bool, len(l))
				for _, t := range l {
					config.CacheMaps[t] = true
				}
			case k == "@no_maps_cache":
				l := strings.Split(v, ",")
				config.NoCacheMaps = make(map[string]bool, len(l))
				for _, t := range l {
					config.NoCacheMaps[t] = true
				}
			}
		}
	}
	return e
}

func (config *DBCacheConfig) isCacheTable(name string) bool {
	if (config.CacheTables[name] || (len(config.CacheTables) == 0)) &&
		!config.NoCacheTables["all"] &&
		!config.NoCacheTables[name] {
		return true
	}
	return false
}

func (config *DBCacheConfig) isCacheMap(name string) bool {
	if (config.CacheMaps[name] || (len(config.CacheMaps) == 0)) &&
		!config.NoCacheMaps["all"] &&
		!config.NoCacheMaps[name] {
		return true
	}
	return false
}
