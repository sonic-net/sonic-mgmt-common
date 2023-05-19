////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

func newTestDB(t *testing.T, opts Options) *DB {
	t.Helper()
	d, err := NewDB(opts)
	if err != nil {
		t.Fatalf("NewDB() failed: %v", err)
	}
	t.Cleanup(func() { d.DeleteDB() })
	return d
}

func TestOnChangeCacheReg(t *testing.T) {
	ts := &TableSpec{Name: "PORT"}
	t.Run("occDisable", func(t *testing.T) {
		d := newTestDB(t, Options{
			DBNo:            ConfigDB,
			IsWriteDisabled: true,
		})
		if err := d.RegisterTableForOnChangeCaching(ts); err == nil {
			t.Fatal("RegisterTableForOnChangeCaching should have failed when IsEnableOnChange=false")
		}
		if d.onCReg.isCacheTable(ts.Name) {
			t.Fatalf("isCacheTable(%q) returned true", ts.Name)
		}
	})

	t.Run("occEnable", func(t *testing.T) {
		d := newTestDB(t, Options{
			DBNo:              ConfigDB,
			IsWriteDisabled:   true,
			IsOnChangeEnabled: true,
		})
		if err := d.RegisterTableForOnChangeCaching(ts); err != nil {
			t.Fatal("RegisterTableForOnChangeCaching failed; ", err)
		}
		if !d.onCReg.isCacheTable(ts.Name) {
			t.Fatalf("isCacheTable(%q) returned false", ts.Name)
		}
		if name := "XYZ"; d.onCReg.isCacheTable(name) {
			t.Fatalf("isCacheTable(%q) returned true", name)
		}
	})

	t.Run("writeEnable", func(t *testing.T) {
		_, err := NewDB(Options{
			DBNo:              ConfigDB,
			IsOnChangeEnabled: true,
		})
		if err == nil {
			t.Error("NewDB should have failed")
		}
	})
}

func TestOnChangeCache(t *testing.T) {
	// OnChange cache enabled db
	d := newTestDB(t, Options{DBNo: ConfigDB, IsWriteDisabled: true, IsOnChangeEnabled: true})
	// Writale db to write test keys
	dw := newTestDB(t, Options{DBNo: ConfigDB, DisableCVLCheck: true})

	tsA := &TableSpec{Name: "TESTOCC_A"}
	tsB := &TableSpec{Name: "TESTOCC_B"}
	key1 := Key{Comp: []string{"001"}}
	key2 := Key{Comp: []string{"002"}}
	key3 := Key{Comp: []string{"003"}}
	val1 := Value{Field: map[string]string{"msg": "hello, world!"}}
	val2 := Value{Field: map[string]string{"msg": "foo bar"}}
	val3 := Value{Field: map[string]string{"NULL": "NULL"}}

	// Create test entries in tables TESTOCC_A and TESTOCC_B
	t.Cleanup(func() {
		dw.DeleteEntry(tsA, key1)
		dw.DeleteEntry(tsA, key2)
		dw.DeleteEntry(tsA, key3)
		dw.DeleteEntry(tsB, key1)
	})
	dw.CreateEntry(tsA, key1, val1)
	dw.CreateEntry(tsA, key2, val2)
	dw.CreateEntry(tsB, key1, val3)

	// Enable OnChange cache for TESTOCC_A only
	if err := d.RegisterTableForOnChangeCaching(tsA); err != nil {
		t.Fatalf("RegisterTableForOnChangeCaching(%q) failed: %v", tsA.Name, err)
	}

	// Call GetEntry() for one key from table A and check OnChangecache contains only that entry
	verifyGetEntry(t, d, tsA, key1, val1)
	verifyOnChangeCache(t, d, tsA, map[string]Value{
		d.key2redis(tsA, key1): val1,
	})

	// Call GetEntry() for other keys from both tables
	verifyGetEntry(t, d, tsA, key1, val1)
	verifyGetEntry(t, d, tsA, key2, val2)
	verifyGetEntry(t, d, tsA, key3, tlerr.TranslibRedisClientEntryNotExist{})
	verifyGetEntry(t, d, tsB, key1, val3)
	verifyGetEntry(t, d, tsB, key3, tlerr.TranslibRedisClientEntryNotExist{})

	// Verify OnChange cache contains entries from table A only
	verifyOnChangeCache(t, d, tsA, map[string]Value{
		d.key2redis(tsA, key1): val1,
		d.key2redis(tsA, key2): val2,
	})
	verifyOnChangeCache(t, d, tsB, nil)

	// Make few changes to both tables
	dw.SetEntry(tsA, key1, val3)
	dw.DeleteEntry(tsA, key2)
	dw.CreateEntry(tsA, key3, val2)
	dw.SetEntry(tsB, key1, val1)

	// Verify that GetEntry() does not return updated values for table A (due to cache)
	verifyGetEntry(t, d, tsA, key1, val1)
	verifyGetEntry(t, d, tsA, key2, val2)

	// But GetEntry() should return new values for table B immediately
	verifyGetEntry(t, d, tsB, key1, val1)

	// Update OnChange cache
	verifyOnChangeCacheUpdate(t, d, tsA, key1, val1, val3)
	verifyOnChangeCacheUpdate(t, d, tsA, key3, Value{}, val2)
	verifyOnChangeCacheDelete(t, d, tsA, key2, val2)

	// Verify cache contents
	verifyOnChangeCache(t, d, tsA, map[string]Value{
		d.key2redis(tsA, key1): val3,
		d.key2redis(tsA, key3): val2,
	})

	// Verify GetEntry() calls
	verifyGetEntry(t, d, tsA, key1, val3)
	verifyGetEntry(t, d, tsA, key2, tlerr.TranslibRedisClientEntryNotExist{})
	verifyGetEntry(t, d, tsA, key3, val2)

}

func verifyGetEntry(t *testing.T, d *DB, ts *TableSpec, k Key, exp interface{}) {
	t.Helper()
	v, err := d.GetEntry(ts, k)
	switch exp := exp.(type) {
	case tlerr.TranslibRedisClientEntryNotExist:
		if _, ok := err.(tlerr.TranslibRedisClientEntryNotExist); !ok {
			t.Errorf("GetEntry(%q) did not return %T", d.key2redis(ts, k), exp)
			t.Fatalf("Received: v=%v, err=%T(%v)", v.Field, err, err)
		}
	case Value:
		if err != nil {
			t.Fatalf("GetEntry(%q) failed: %v", d.key2redis(ts, k), err)
		} else if !reflect.DeepEqual(v, exp) {
			t.Errorf("GetEntry(%q) returned unexpected values", d.key2redis(ts, k))
			t.Errorf("Expected: %v", exp.Field)
			t.Fatalf("Received: %v", v.Field)
		}
	default:
		t.Fatalf("Invalid expValue type: %T", exp)
	}
}

func verifyOnChangeCache(t *testing.T, d *DB, ts *TableSpec, exp map[string]Value) {
	t.Helper()
	if cached := d.cache.Tables[ts.Name]; !reflect.DeepEqual(cached.entry, exp) {
		t.Errorf("OnChange cache comparson failed for table %s", ts.Name)
		t.Errorf("Expected: %v", exp)
		t.Fatalf("Found:    %v", cached.entry)
	}
}

func verifyOnChangeCacheUpdate(t *testing.T, d *DB, ts *TableSpec, k Key, expOld, expNew Value) {
	t.Helper()
	old, new, err := d.OnChangeCacheUpdate(ts, k)
	if err != nil || !reflect.DeepEqual(old, expOld) || !reflect.DeepEqual(new, expNew) {
		t.Errorf("OnChangeCacheUpdate(%q) failed", d.key2redis(ts, k))
		t.Errorf("Expected: old=%v, new=%v", expOld.Field, expNew.Field)
		t.Fatalf("Received: old=%v, new=%v, err=%v", old.Field, new.Field, err)
	}
}

func verifyOnChangeCacheDelete(t *testing.T, d *DB, ts *TableSpec, k Key, expOld Value) {
	t.Helper()
	old, err := d.OnChangeCacheDelete(ts, k)
	if err != nil || !reflect.DeepEqual(old, expOld) {
		t.Errorf("OnChangeCacheUpdate(%q) failed", d.key2redis(ts, k))
		t.Errorf("Expected: %v", expOld.Field)
		t.Fatalf("Received: %v, err=%v", old.Field, err)
	}
}
