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
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"github.com/kylelemons/godebug/pretty"
)

type GetConfigOptions struct {
	ScanCountHint int64 // Hint of redis work required for Scan
	AllowWritable bool  // Allow on writable enabled DB object too
}

// GetConfig API to get some tables. Very expensive operation (time/resources)
// If len(tables) == 0, return all tables in the CONFIG_DB
// Notes:
//   - Only supported on the CONFIG_DB
//   - The keys of the table should not contain TableSeparator, or KeySeparator
//     (CONFIG_DB table keys should not contain "|" (defaults for TableSeparator,
//     or KeySeparator))
//   - Only supported when write/set is disabled [IsWriteDisabled == true ]
//   - OnChange not supported [IsEnableOnChange == false]
//   - PCC (per_connection_cache) is not supported, and it will log an error/
//     warning.
func (d *DB) GetConfig(tables []*TableSpec, opt *GetConfigOptions) (map[TableSpec]Table, error) {

	if glog.V(3) {
		glog.Infof("GetConfig: Begin: tables: %v, opt: %+v", tables, opt)
	}

	if d.Opts.DBNo != ConfigDB {
		err := SupportsCfgDBOnly
		glog.Error("GetConfig: error: ", err)
		return nil, err
	}

	allowWritable := opt != nil && opt.AllowWritable
	if !d.Opts.IsWriteDisabled && !allowWritable {
		err := SupportsReadOnly
		glog.Error("GetConfig: error: ", err)
		return nil, err
	}

	if d.Opts.IsOnChangeEnabled {
		err := OnChangeNoSupport
		glog.Error("GetConfig: error: ", err)
		return nil, err
	}

	if d.dbCacheConfig.PerConnection {
		glog.Warning("GetConfig: Per Connection Cache not supported")
	}

	// Filter on tables: This is optimized for 1 table. Filtering on multiple
	// tables can be optimized, however, it needs some glob pattern
	// manufacturing feasibility. All tables is the only requirement currently,
	// therefore not considering further optimization.
	pattern := Key{Comp: []string{"*"}}
	var ts *TableSpec
	var tsM map[TableSpec]bool
	if len(tables) == 1 {

		ts = tables[0]
		tsM = map[TableSpec]bool{*ts: true}

	} else {

		ts = &(TableSpec{Name: "*"})

		// Create a map of requested tables, for faster comparision
		if len(tables) > 1 {
			tsM = make(map[TableSpec]bool, len(tables))
			for _, ts := range tables {
				tsM[*ts] = true
			}
		}
	}

	scanCountHint := int64(defaultGetTablesSCCountHint)
	if (opt != nil) && (opt.ScanCountHint != 0) {
		scanCountHint = opt.ScanCountHint
	}
	scOpts := ScanCursorOpts{
		AllowWritable:   allowWritable,
		CountHint:       scanCountHint,
		AllowDuplicates: true,
		ScanType:        KeyScanType, // Default
	}

	sc, err := d.NewScanCursor(ts, pattern, &scOpts)
	if err != nil {
		return nil, err
	}
	defer sc.DeleteScanCursor()

	tblM := make(map[TableSpec]Table, InitialTablesCount)

	for scanComplete := false; !scanComplete; {
		var redisKeys []string
		redisKeys, scanComplete, err = sc.GetNextRedisKeys(&scOpts)
		if err != nil {
			// GetNextRedisKeys already logged the error
			return nil, err
		}

		if glog.V(4) {
			glog.Infof("GetConfig: %v #redisKeys, scanComplete %v",
				len(redisKeys), scanComplete)
		}

		// Initialize the pipeline
		pipe := d.client.Pipeline()

		tss := make([]*TableSpec, 0, len(redisKeys))
		presults := make([]*redis.StringStringMapCmd, 0, len(redisKeys))
		keys := make([]Key, 0, len(redisKeys))

		for index, redisKey := range redisKeys {
			if glog.V(6) {
				glog.Infof("GetConfig: redisKeys[%d]: %v", index, redisKey)
			}

			// Keys with no (Table|Key)Separator in them would not be selected
			// due to the "*|*" pattern search. So, no need to select them.

			rKts, key := d.redis2ts_key(redisKey)

			// Do the table filtering here, since redis glob style patterns
			// cannot handle multiple tables matching.
			if len(tables) > 1 {
				if present, ok := tsM[rKts]; !ok || !present {
					tss = append(tss, nil)
					keys = append(keys, Key{})
					presults = append(presults, nil)
					continue
				}
			}

			tss = append(tss, &rKts)
			keys = append(keys, key)
			presults = append(presults, pipe.HGetAll(redisKey))
		}

		if glog.V(3) {
			glog.Info("GetConfig: #tss: ", len(tss), ", #presults: ",
				len(presults), ", #keys: ", len(keys))
		}

		// Execute the Pipeline
		if glog.V(3) {
			glog.Info("GetConfig: RedisCmd: ", d.Name(), ": ", "pipe.Exec")
		}
		_, err = pipe.Exec() // Ignore returned Cmds. If any err, log it.

		// Close the Pipeline
		pipe.Close()

		if err != nil {
			glog.Error("GetConfig: pipe.Exec() err: ", err)
			return nil, err
		}

		// Iterate the returned array of Values to create tblM[]
		for index, redisKey := range redisKeys {
			if glog.V(6) {
				glog.Infof("GetConfig: tblM[] redisKeys[%d]: %v", index,
					redisKey)
			}

			result := presults[index]

			if tss == nil {
				continue
			}

			if result == nil {
				glog.Warningf("GetConfig: redisKeys[%d]: %v nil", index,
					redisKey)
				continue
			}

			field, err := result.Result()
			if err != nil {
				glog.Warningf("GetConfig: redisKeys[%d]: %v err", index, err)
				continue
			}

			dbValue := Value{Field: field}

			// Create Table in map if not created
			ts := *(tss[index])
			if _, ok := tblM[ts]; !ok {
				tblM[ts] = Table{
					ts:       &ts,
					entry:    make(map[string]Value, InitialTableEntryCount),
					complete: true,
					db:       d,
				}
			}

			tblM[ts].entry[redisKey] = dbValue
		}
	}

	if allowWritable {
		err = d.applyTxCache(tblM, tsM)
		if err != nil {
			glog.V(2).Info("GetConfig: applyTxCache failed: ", err)
			return nil, err
		}
	}

	if glog.V(3) {
		glog.Infof("GetConfig: End: #tblM: %v", len(tblM))
	}
	if glog.V(6) {
		for ts, table := range tblM {
			glog.Infof("GetConfig: #entry in tblM[%v] = %v", ts.Name,
				len(table.entry))
			if glog.V(8) {
				glog.Infof("GetConfig: pretty entry in tblM[%v] = \n%v",
					ts.Name, pretty.Sprint(table.entry))
			}
		}
	}

	return tblM, nil
}

func (d *DB) applyTxCache(data map[TableSpec]Table, tableFilter map[TableSpec]bool) error {
	for _, cmd := range d.txCmds {
		cmdTs := TableSpec{Name: cmd.ts.Name} // to match the TableSpec created from redis key
		if len(tableFilter) != 0 && !tableFilter[cmdTs] {
			continue
		}

		if cmd.op != txOpHMSet {
			if _, tableFound := data[cmdTs]; !tableFound {
				continue
			}
		}

		keyStr := d.key2redis(&cmdTs, *cmd.key)
		keyDelete := false

		switch cmd.op {
		case txOpHMSet:
			if table, tableFound := data[cmdTs]; !tableFound {
				data[cmdTs] = Table{
					ts:       &cmdTs,
					entry:    map[string]Value{keyStr: cmd.value.Copy()},
					complete: true,
					db:       d,
				}
			} else if entry, keyFound := table.entry[keyStr]; keyFound {
				for fName, fVal := range cmd.value.Field {
					entry.Set(fName, fVal)
				}
			} else {
				table.entry[keyStr] = cmd.value.Copy()
			}
		case txOpHDel:
			if entry, keyFound := data[cmdTs].entry[keyStr]; keyFound {
				for fName := range cmd.value.Field {
					entry.Remove(fName)
				}
				keyDelete = !entry.IsPopulated()
			}
		case txOpDel:
			keyDelete = true
		}

		if keyDelete {
			table := data[cmdTs]
			delete(table.entry, keyStr)
			if len(table.entry) == 0 {
				delete(data, cmdTs)
			}
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Constants                                                        //
////////////////////////////////////////////////////////////////////////////////

const (
	defaultGetTablesSCCountHint = 100
)
