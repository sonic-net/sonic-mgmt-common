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
	"github.com/golang/glog"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

// Table gives the entire table as a map.
// (Eg: { ts: &TableSpec{ Name: "ACL_TABLE" },
//        entry: map[string]Value {
//            "ACL_TABLE|acl1|rule1_1":  Value {
//                            Field: map[string]string {
//                              "type" : "l3v6", "ports" : "Ethernet0",
//                            }
//                          },
//            "ACL_TABLE|acl1|rule1_2":  Value {
//                            Field: map[string]string {
//                              "type" : "l3v6", "ports" : "eth0",
//                            }
//                          },
//                          }
//        })

type Table struct {
	ts       *TableSpec
	entry    map[string]Value
	complete bool
	patterns map[string][]Key
	db       *DB
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// GetTable gets the entire table.
func (d *DB) GetTable(ts *TableSpec) (Table, error) {
	if glog.V(3) {
		glog.Info("GetTable: Begin: ts: ", ts)
	}

	if (d == nil) || (d.client == nil) {
		return Table{}, tlerr.TranslibDBConnectionReset{}
	}

	/*
		table := Table{
			ts: ts,
			entry: map[string]Value{
				"table1|k0.0|k0.1": Value{
					map[string]string{
						"f0.0": "v0.0",
						"f0.1": "v0.1",
						"f0.2": "v0.2",
					},
				},
				"table1|k1.0|k1.1": Value{
					map[string]string{
						"f1.0": "v1.0",
						"f1.1": "v1.1",
						"f1.2": "v1.2",
					},
				},
			},
		        db: d,
		}
	*/

	// Create Table
	table := Table{
		ts:       ts,
		entry:    make(map[string]Value, InitialTableEntryCount),
		complete: true,
		patterns: make(map[string][]Key, InitialTablePatternCount),
		db:       d,
	}

	// This can be done via a LUA script as well. For now do this. TBD
	// Read Keys
	keys, e := d.GetKeys(ts)
	if e != nil {
		glog.Error("GetTable: GetKeys: " + e.Error())
		table = Table{}
		goto GetTableExit
	}

	table.patterns[d.key2redis(ts, Key{Comp: []string{"*"}})] = keys

	// For each key in Keys
	// 	Add Value into table.entry[key)]
	for i := 0; i < len(keys); i++ {
		value, e := d.GetEntry(ts, keys[i])
		if e != nil {
			glog.Warning("GetTable: GetKeys: ", d.Name(),
				": ", ts.Name, ": ", e.Error())
			value = Value{}
			e = nil
		}
		table.entry[d.key2redis(ts, keys[i])] = value
	}

	// Mark Per Connection Cache table as complete.
	if (d.dbCacheConfig.PerConnection &&
		d.dbCacheConfig.isCacheTable(ts.Name)) ||
		(d.Opts.IsOnChangeEnabled && d.onCReg.isCacheTable(ts.Name)) {
		if cTable, ok := d.cache.Tables[ts.Name]; ok {
			cTable.complete = true
		}
	}

GetTableExit:

	if glog.V(3) {
		glog.Info("GetTable: End: table: ", table)
	}
	return table, e
}

// GetKeys method retrieves all entry/row keys from a previously read table.
func (t *Table) GetKeys() ([]Key, error) {
	if glog.V(3) {
		glog.Info("Table.GetKeys: Begin: t: ", t)
	}

	keys := make([]Key, 0, len(t.entry))
	for k := range t.entry {
		keys = append(keys, t.db.redis2key(t.ts, k))
	}

	if glog.V(3) {
		glog.Info("Table.GetKeys: End: keys: ", keys)
	}
	return keys, nil
}

// GetEntry method retrieves an entry/row from a previously read table.
func (t *Table) GetEntry(key Key) (Value, error) {
	/*
		return Value{map[string]string{
			"f0.0": "v0.0",
			"f0.1": "v0.1",
			"f0.2": "v0.2",
		},
		}, nil
	*/
	if glog.V(3) {
		glog.Info("Table.GetEntry: Begin: t: ", t, " key: ", key)
	}

	v := t.entry[t.db.key2redis(t.ts, key)]

	if glog.V(3) {
		glog.Info("Table.GetEntry: End: entry: ", v)
	}
	return v, nil
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////
