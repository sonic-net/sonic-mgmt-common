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
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func BenchmarkGetConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, e := db.GetConfig(nil, nil); e != nil {
			b.Errorf("GetConfig() returns err: %v", e)
		}
	}
}

func TestGetConfig(t *testing.T) {
	if _, e := db.GetConfig(nil, nil); e != nil {
		t.Errorf("GetConfig() returns err: %v", e)
	}
}

func TestGetConfigSingular(t *testing.T) {
	if _, e := db.GetConfig([]*TableSpec{&ts}, nil); e != nil {
		t.Errorf("GetConfig() returns err: %v", e)
	}
}

func TestGetConfigAllTables(t *testing.T) {
	verifyGetConfigAllTables(t, db, nil)
}

func verifyGetConfigAllTables(t *testing.T, db *DB, opts *GetConfigOptions) {
	tablesM, e := db.GetConfig([]*TableSpec{}, opts)
	if e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}

	table, e := db.GetTable(&ts)
	if e != nil {
		t.Errorf("GetTable() returns err: %v", e)
	}

	table.patterns = nil // because GetConfig() does not populate patterns
	if !reflect.DeepEqual(tablesM[ts], table) {
		fmt.Println("\ntable: \n", table)
		fmt.Println("\ntablesM[ts]: \n", tablesM[ts])
		t.Errorf("GetTable(ts) != GetConfig()[ts]")
	}

	// Count the keys in all the tables
	tsM := make(map[TableSpec]int, 10)
	redisKeys, e := db.client.Keys("*").Result()
	if e != nil {
		t.Errorf("client.Keys() returns err: %v", e)
	}

	for _, redisKey := range redisKeys {

		// Does it have a Separator?
		if strings.IndexAny(redisKey, db.Opts.TableNameSeparator+
			db.Opts.KeySeparator) == -1 {

			continue
		}

		ts, _ := db.redis2ts_key(redisKey)
		tsM[ts]++
	}

	if len(tsM) != len(tablesM) {
		fmt.Println("\n#tsM: \n", len(tsM))
		fmt.Println("\n#tablesM: \n", len(tablesM))
		t.Errorf("#GetConfig() != #Tables")
	}

	for ts, table := range tablesM {
		tableComp, e := db.GetTable(&ts)
		if e != nil {
			t.Errorf("GetTable(%v) returns err: %v", ts, e)
		}

		tableComp.patterns = nil // because GetConfig() does not populate patterns
		if !reflect.DeepEqual(table, tableComp) {
			fmt.Println("\ntable: \n", table)
			fmt.Println("\ntableComp: \n", tableComp)
			t.Errorf("Detail: GetTable(%q) != GetConfig()[%q]", ts.Name, ts.Name)
		}
	}
}

func TestGetConfig_writable(t *testing.T) {
	d, err := newDB(ConfigDB)
	if err != nil {
		t.Fatal("newDB() failed;", err)
	}
	defer d.DeleteDB()

	// GetConfig() should fail with nil options
	t.Run("nilOpts", func(tt *testing.T) {
		_, err = d.GetConfig([]*TableSpec{}, nil)
		if err == nil {
			tt.Errorf("GetConfig() with nil options should have failed on writable DB")
		}
	})

	// GetConfig() should fail with AllowWritable=false
	t.Run("AllowWritable=false", func(tt *testing.T) {
		_, err = d.GetConfig([]*TableSpec{}, &GetConfigOptions{AllowWritable: false})
		if err == nil {
			tt.Errorf("GetConfig() with AllowWritable=false should have failed on writable DB")
		}
	})

	// GetConfig() should work with AllowWritable=true
	t.Run("AllowWritable=true", func(tt *testing.T) {
		verifyGetConfigAllTables(tt, d, &GetConfigOptions{AllowWritable: true})
	})
}

func TestGetConfig_txCache(t *testing.T) {
	d, err := newDB(ConfigDB)
	if err != nil {
		t.Fatal("newDB() failed;", err)
	}
	defer d.DeleteDB()

	if err = d.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx() failed; ", err)
	}
	defer d.AbortTx()

	// We'll perform few operations on 'ts' table to populate transaction cache.
	// testTableSetup() would have already populated test entries during init
	newKey := Key{Comp: []string{"__A_NEW_KEY__"}}
	modKey := Key{Comp: []string{"KEY1"}}
	delKey := Key{Comp: []string{"KEY2"}}

	// Value for HMSET; used by both key create and modify steps
	testValue := Value{Field: map[string]string{"GetConfigTestField": "foo"}}

	// Load the existing value of modKey; pick a random field for HDEL
	oldVal, _ := d.GetEntry(&ts, modKey)
	delFields := Value{Field: map[string]string{}}
	for field := range oldVal.Field {
		delFields.Set(field, "")
		break
	}

	// Perform db operations
	d.SetEntry(&ts, newKey, testValue)
	d.ModEntry(&ts, modKey, testValue)
	d.DeleteEntryFields(&ts, modKey, delFields)
	d.DeleteEntry(&ts, delKey)

	// Run GetConfig() for 'ts' table only
	opts := &GetConfigOptions{AllowWritable: true}
	tsConfig, err := d.GetConfig([]*TableSpec{&ts}, opts)
	if err != nil {
		t.Fatalf("GetConfig([%q]) failed; %v", ts.Name, err)
	}
	tsTable := tsConfig[ts]
	if tsTable.ts == nil || len(tsConfig) != 1 {
		t.Fatalf("GetConfig([%q]) returned incorrect data; %v", ts.Name, tsConfig)
	}

	// Check value for newly created key
	t.Run("newKey", func(tt *testing.T) {
		val, _ := tsTable.GetEntry(newKey)
		if !reflect.DeepEqual(val, testValue) {
			tt.Errorf("tsTable contains wrong value for new key %q", newKey.Comp[0])
			tt.Errorf("Expected: %v", testValue)
			tt.Errorf("Received: %v", val)
		}
	})

	// Check value for modified key
	t.Run("modKey", func(tt *testing.T) {
		expValue := oldVal.Copy()
		for f := range delFields.Field {
			expValue.Remove(f)
		}
		for f, v := range testValue.Field {
			expValue.Set(f, v)
		}

		val, _ := tsTable.GetEntry(modKey)
		if !reflect.DeepEqual(val, expValue) {
			tt.Errorf("tsTable contains wrong value for the modified key %q", modKey.Comp[0])
			tt.Errorf("Expected: %v", expValue)
			tt.Errorf("Received: %v", val)
		}
	})

	// Check deleted key is not present in the response
	t.Run("delKey", func(tt *testing.T) {
		val, _ := tsTable.GetEntry(delKey)
		if val.IsPopulated() {
			tt.Errorf("tsTable contains value for deleted key %s!", delKey.Comp[0])
		}
	})

	// Try GetConfig() for all tables..
	t.Run("allTables", func(tt *testing.T) {
		verifyGetConfigAllTables(tt, d, opts)
	})

	// Check adding and deleting same key multiple times.
	t.Run("addDelTable", func(tt *testing.T) {
		newTable := TableSpec{Name: "__A_NEW_TABLE__"}
		for i := 0; i < 10; i++ {
			d.SetEntry(&newTable, newKey, testValue)
			d.DeleteEntry(&newTable, newKey)
		}
		verifyGetConfigTableNotFound(tt, d, newTable)
	})

	// Check deleting all keys of an existing table
	t.Run("delTable", func(tt *testing.T) {
		d.DeleteTable(&ts)
		verifyGetConfigTableNotFound(tt, d, ts)
	})
}

func verifyGetConfigTableNotFound(t *testing.T, d *DB, ts TableSpec) {
	opts := &GetConfigOptions{AllowWritable: true}
	allTables, err := d.GetConfig([]*TableSpec{}, opts)
	if err != nil {
		t.Fatalf("GetConfig() failed; err=%v", err)
	}
	if table, ok := allTables[ts]; ok {
		t.Fatalf("GetConfig() returned stale entries for the deleted table %q;\n%v", ts.Name, table)
	}
}
