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
	"testing"
)

func BenchmarkExistKeysPattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if exists, e := db.ExistKeysPattern(&ts, Key{Comp: []string{"*"}}); e != nil || !exists {
			b.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
		}
	}
}

func BenchmarkGetKeysPattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, e := db.GetKeysPattern(&ts, Key{Comp: []string{"*"}}); e != nil {
			b.Errorf("GetKeysPattern() returns err: %v", e)
		}
	}
}

func TestGetKeysPattern(t *testing.T) {
	if keys, e := db.GetKeysPattern(&ts, Key{Comp: []string{"*"}}); e != nil || len(keys) == 0 {
		t.Errorf("GetKeysPattern() returns len(keys) == 0 || err: %v", e)
	}
}

func TestExistKeysPattern(t *testing.T) {
	if exists, e := db.ExistKeysPattern(&ts, Key{Comp: []string{"*"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternSinglular(t *testing.T) {
	if exists, e := db.ExistKeysPattern(&ts, Key{Comp: []string{"KEY1"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternGeneric(t *testing.T) {
	if exists, e := db.ExistKeysPattern(&ts, Key{Comp: []string{"KEY*"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternEmpty(t *testing.T) {
	if exists, e := db.ExistKeysPattern(&TableSpec{Name: "UNLIKELY_23"},
		Key{Comp: []string{"*"}}); e != nil || exists {
		t.Errorf("ExistKeysPattern() returns exists || err: %v", e)
	}

	if exists, e := db.ExistKeysPattern(&ts, Key{Comp: []string{"UNKNOWN"}}); e != nil || exists {
		t.Errorf("ExistKeysPattern() returns exists || err: %v", e)
	}
}

func TestExistKeysPatternRW(t *testing.T) {
	dbRW, err := newDB(ConfigDB)
	if err != nil {
		t.Fatalf("TestExistKeysPatternRW: newDB() for RW fails err = %v\n", err)
	}
	t.Cleanup(func() { dbRW.DeleteDB() })

	if exists, e := dbRW.ExistKeysPattern(&ts, Key{Comp: []string{"*"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternSinglularRW(t *testing.T) {
	dbRW, err := newDB(ConfigDB)
	if err != nil {
		t.Fatalf("TestExistKeysPatternSinglularRW: newDB() for RW fails err = %v\n", err)
	}
	t.Cleanup(func() { dbRW.DeleteDB() })

	if exists, e := dbRW.ExistKeysPattern(&ts, Key{Comp: []string{"KEY1"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternGenericRW(t *testing.T) {
	dbRW, err := newDB(ConfigDB)
	if err != nil {
		t.Fatalf("TestExistKeysPatternGenericRW: newDB() for RW fails err = %v\n", err)
	}
	t.Cleanup(func() { dbRW.DeleteDB() })

	if exists, e := dbRW.ExistKeysPattern(&ts, Key{Comp: []string{"KEY*"}}); e != nil || !exists {
		t.Errorf("ExistKeysPattern() returns !exists || err: %v", e)
	}
}

func TestExistKeysPatternEmptyRW(t *testing.T) {
	dbRW, err := newDB(ConfigDB)
	if err != nil {
		t.Fatalf("TestExistKeysPatternEmptyRW: newDB() for RW fails err = %v\n", err)
	}
	t.Cleanup(func() { dbRW.DeleteDB() })

	if exists, e := dbRW.ExistKeysPattern(&TableSpec{Name: "UNLIKELY_23"},
		Key{Comp: []string{"*"}}); e != nil || exists {
		t.Errorf("ExistKeysPattern() returns exists || err: %v", e)
	}

	if exists, e := dbRW.ExistKeysPattern(&ts, Key{Comp: []string{"UNKNOWN"}}); e != nil || exists {
		t.Errorf("ExistKeysPattern() returns exists || err: %v", e)
	}
}
