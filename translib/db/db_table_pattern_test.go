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
	"testing"
)

var allKeysPat = Key{Comp: []string{"*"}}
var singleKeyPat = Key{Comp: []string{"KEY1"}}
var genericKeyPat = Key{Comp: []string{"KEY*"}}

func BenchmarkGetTablePattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, e := db.GetTablePattern(&ts, allKeysPat); e != nil {
			b.Errorf("GetTablePattern() returns err: %v", e)
		}
	}
}

func BenchmarkGetTableOrig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, e := db.GetTable(&ts); e != nil {
			b.Errorf("GetTable() returns err: %v", e)
		}
	}
}

func TestGetTablePattern(t *testing.T) {
	if _, e := db.GetTablePattern(&ts, allKeysPat); e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}
}

func TestGetTablePatternSingular(t *testing.T) {
	if _, e := db.GetTablePattern(&ts, singleKeyPat); e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}
}

func TestGetTablePatternGeneric(t *testing.T) {
	if _, e := db.GetTablePattern(&ts, genericKeyPat); e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}
}

func TestGetTablePatternEmpty(t *testing.T) {
	if _, e := db.GetTablePattern(&TableSpec{Name: "UNLIKELY_23"}, allKeysPat); e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}
}

func TestGetTableOrig(t *testing.T) {
	if _, e := db.GetTable(&ts); e != nil {
		t.Errorf("GetTable() returns err: %v", e)
	}
}

func TestGetTableOrigEmpty(t *testing.T) {
	if _, e := db.GetTable(&TableSpec{Name: "UNLIKELY_23"}); e != nil {
		t.Errorf("GetTable() returns err: %v", e)
	}
}

func TestGetTablePatternCompOrig(t *testing.T) {
	tPat, e := db.GetTablePattern(&ts, allKeysPat)
	if e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}

	tOrig, e := db.GetTable(&ts)
	if e != nil {
		t.Errorf("GetTable() returns err: %v", e)
	}

	// The ordering of the arrays in patterns map values matters
	delete(tPat.patterns, db.key2redis(&ts, Key{Comp: []string{"*"}}))
	delete(tOrig.patterns, db.key2redis(&ts, Key{Comp: []string{"*"}}))

	if !reflect.DeepEqual(tPat, tOrig) {
		fmt.Println("\ntPat: \n", tPat)
		fmt.Println("\ntOrig: \n", tOrig)
		t.Errorf("GetTable() != GetTablePattern")
	}
}

func TestGetTablePatternCompOrigEmpty(t *testing.T) {
	tsEmpty := TableSpec{Name: "UNLIKELY_23"}
	tPat, e := db.GetTablePattern(&tsEmpty, allKeysPat)
	if e != nil {
		t.Errorf("GetTablePattern() returns err: %v", e)
	}

	tOrig, e := db.GetTable(&tsEmpty)
	if e != nil {
		t.Errorf("GetTable() returns err: %v", e)
	}

	if !reflect.DeepEqual(tPat, tOrig) {
		fmt.Println("\ntPat: \n", tPat)
		fmt.Println("\ntOrig: \n", tOrig)
		t.Errorf("Empty GetTable() != GetTablePattern")
	}
}

func TestGetTablePattern_txCache(t *testing.T) {
	d := newTestDB(t, Options{
			DBNo:            ConfigDB,
			DisableCVLCheck: true,
	})
	setupTestData(t, d.client, map[string]map[string]interface{}{
		"TEST_INTERFACE|Ethernet0":              {"vrf": "Vrf1"},
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": {"NULL": "NULL"},
		"TEST_INTERFACE|Ethernet1":              {"NULL": "NULL"},
		"TEST_INTERFACE|Ethernet1|101.0.0.1/24": {"NULL": "NULL"},
	})

	testTable := &TableSpec{Name: "TEST_INTERFACE"}
	nullValue := Value{map[string]string{"NULL": "NULL"}}
	vrfValue1 := Value{map[string]string{"vrf": "Vrf1"}}

	// Run few tests before transaction changes

	t.Run("T|*/beforeWrite", testGetTablePattern(d, testTable, NewKey("*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0":              vrfValue1,
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet1":              nullValue,
		"TEST_INTERFACE|Ethernet1|101.0.0.1/24": nullValue,
	}))

	t.Run("T|*|*/beforeWrite", testGetTablePattern(d, testTable, NewKey("*", "*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet1|101.0.0.1/24": nullValue,
	}))

	t.Run("T|x|*/beforeWrite", testGetTablePattern(d, testTable, NewKey("Ethernet0", "*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
	}))

	t.Run("T|x|*/unknown,beforeWrite", testGetTablePattern(d, testTable, NewKey("Ethernet9", "*"), map[string]Value{}))

	// Perform few changes in a transaction

	if err := d.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx() failed:", err)
	}
	d.SetEntry(testTable, *NewKey("Ethernet1"), vrfValue1)
	d.DeleteEntry(testTable, *NewKey("Ethernet1", "101.0.0.1/24"))
	d.CreateEntry(testTable, *NewKey("Ethernet2"), nullValue)
	d.CreateEntry(testTable, *NewKey("Ethernet2", "102.0.0.1/24"), nullValue)
	d.CreateEntry(testTable, *NewKey("Ethernet2", "102.0.0.2/32"), nullValue)

	// Rerun the tests

	t.Run("T|*", testGetTablePattern(d, testTable, NewKey("*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0":              vrfValue1,
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet1":              vrfValue1,
		"TEST_INTERFACE|Ethernet2":              nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.2/32": nullValue,
	}))

	t.Run("T|*|*", testGetTablePattern(d, testTable, NewKey("*", "*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.2/32": nullValue,
	}))

	t.Run("T|*|*x", testGetTablePattern(d, testTable, NewKey("*", "*/24"), map[string]Value{
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.1/24": nullValue,
	}))

	t.Run("T|x|*/created", testGetTablePattern(d, testTable, NewKey("Ethernet2", "*"), map[string]Value{
		"TEST_INTERFACE|Ethernet2|102.0.0.1/24": nullValue,
		"TEST_INTERFACE|Ethernet2|102.0.0.2/32": nullValue,
	}))

	t.Run("T|x|*/deleted", testGetTablePattern(d, testTable, NewKey("Ethernet1", "*"), map[string]Value{}))

	t.Run("T|x|*/unknown", testGetTablePattern(d, testTable, NewKey("Ethernet9", "*"), map[string]Value{}))

	t.Run("T|x|*/unmodified", testGetTablePattern(d, testTable, NewKey("Ethernet0", "*"), map[string]Value{
		"TEST_INTERFACE|Ethernet0|100.0.0.1/24": nullValue,
	}))
}

func testGetTablePattern(d *DB, ts *TableSpec, pat *Key, exp map[string]Value) func(*testing.T) {
	return func(t *testing.T) {
		table, err := d.GetTablePattern(ts, *pat)
		if err != nil {
			t.Fatalf("GetTablePattern(\"%s\") returned error: %v", d.key2redis(ts, *pat), err)
		}
		if table.entry == nil {
			table.entry = make(map[string]Value)
		}
		if !reflect.DeepEqual(table.entry, exp) {
			t.Errorf("GetTablePattern(\"%s\") returned expected values", d.key2redis(ts, *pat))
			t.Errorf("Expecting: %v", exp)
			t.Errorf("Received : %v", table.entry)
		}
	}
}
