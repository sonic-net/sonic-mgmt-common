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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/go-redis/redis/v7"
)

type Search = common.Search

func expandResult(r interface{}) (val interface{}, err error) {
	f := reflect.ValueOf(r).MethodByName("Result")
	if !f.IsValid() {
		val, err = r, nil
	} else if ret := f.Call(nil); ret[1].IsNil() {
		val, err = ret[0].Interface(), nil
	} else {
		val, err = ret[0].Interface(), ret[1].Interface().(error)
	}
	if arr, ok := val.([]string); ok {
		sort.Strings(arr)
	}
	if s, ok := val.(string); ok {
		var j map[string]interface{}
		if err := json.Unmarshal([]byte(s), &j); err == nil {
			jj, _ := json.Marshal(j)
			val = string(jj) // reformat json data for comparison
		}
	}
	return
}

func verifyResult(t testing.TB, res, exp interface{}) {
	aVal, aErr := expandResult(res)
	eVal, eErr := expandResult(exp)
	if !reflect.DeepEqual(aVal, eVal) || !errors.Is(aErr, eErr) {
		t.Errorf("Verification failed at %s", t.Name())
		t.Errorf("Exp: val=%#v, err=%v", eVal, eErr)
		t.Errorf("Got: val=%#v, err=%v", aVal, aErr)
	}
}

func TestCvlDB_RedisCompatibility(t *testing.T) {
	d1 := newTestDB(t, ConfigDB)
	setupTestData(t, d1.client, map[string]map[string]interface{}{
		"CVLDB|event|1": {"id": 1, "message": "hello, world!"},
		"CVLDB|event|2": {"id": 2, "message": "foo bar"},
	})

	c1 := &cvlDBAccess{d1}

	// Exists()

	t.Run("Exists", func(tt *testing.T) {
		cVal := c1.Exists("CVLDB|event|1")
		rVal := d1.client.Exists("CVLDB|event|1")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("Exists_unknownKey", func(tt *testing.T) {
		cVal := c1.Exists("CVLDB|event|unknown")
		rVal := d1.client.Exists("CVLDB|event|unknown")
		verifyResult(tt, cVal, rVal)
	})

	// Keys()

	t.Run("Keys_all", func(tt *testing.T) {
		cVal := c1.Keys("CVLDB|*")
		rVal := d1.client.Keys("CVLDB|*")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("Keys_WC", func(tt *testing.T) {
		cVal := c1.Keys("CVLDB|*|1")
		rVal := d1.client.Keys("CVLDB|*|1")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("Keys_noWC", func(tt *testing.T) {
		cVal := c1.Keys("CVLDB|event|2")
		rVal := d1.client.Keys("CVLDB|event|2")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("Keys_unknownKey", func(tt *testing.T) {
		cVal := c1.Keys("CVLDB|*|unknown")
		rVal := d1.client.Keys("CVLDB|*|unknown")
		verifyResult(tt, cVal, rVal)
	})

	// HGet()

	t.Run("HGet", func(tt *testing.T) {
		cVal := c1.HGet("CVLDB|event|1", "message")
		rVal := d1.client.HGet("CVLDB|event|1", "message")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HGet_unknownField", func(tt *testing.T) {
		cVal := c1.HGet("CVLDB|event|1", "xyz")
		rVal := d1.client.HGet("CVLDB|event|1", "xyz")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HGet_unknownKey", func(tt *testing.T) {
		cVal := c1.HGet("CVLDB|event|unknown", "message")
		rVal := d1.client.HGet("CVLDB|event|unknown", "message")
		verifyResult(tt, cVal, rVal)
	})

	// HMGet()

	t.Run("HMGet", func(tt *testing.T) {
		cVal := c1.HMGet("CVLDB|event|1", "message", "id")
		rVal := d1.client.HMGet("CVLDB|event|1", "message", "id")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HMGet_unknownField", func(tt *testing.T) {
		cVal := c1.HMGet("CVLDB|event|1", "xyz", "abc", "id")
		rVal := d1.client.HMGet("CVLDB|event|1", "xyz", "abc", "id")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HMGet_unknownKey", func(tt *testing.T) {
		cVal := c1.HMGet("CVLDB|event|unknown", "id")
		rVal := d1.client.HMGet("CVLDB|event|unknown", "id")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HMGet_unknownKey_2", func(tt *testing.T) {
		cVal := c1.HMGet("CVLDB|event|unknown", "message", "id")
		rVal := d1.client.HMGet("CVLDB|event|unknown", "message", "id")
		verifyResult(tt, cVal, rVal)
	})

	// HGetAll()

	t.Run("HGetAll", func(tt *testing.T) {
		cVal := c1.HGetAll("CVLDB|event|1")
		rVal := d1.client.HGetAll("CVLDB|event|1")
		verifyResult(tt, cVal, rVal)
	})
	t.Run("HGetAll_unknownKey", func(tt *testing.T) {
		cVal := c1.HGetAll("CVLDB|event|unknown")
		rVal := d1.client.HGetAll("CVLDB|event|unknown")
		verifyResult(tt, cVal, rVal)
	})

	// Pipe Keys No Tx
	t.Run("Pipe_Keys_No_Tx", func(tt *testing.T) {
		pipe := c1.Pipeline()
		cVal1 := pipe.Keys("CVLDB|*")
		cVal2 := pipe.Keys("CVLDB|*|1")
		cVal3 := pipe.Keys("CVLDB|event|2")
		cVal4 := pipe.Keys("CVLDB|*|unknown")
		pipe.Exec()
		rPipe := c1.Db.client.Pipeline()
		rVal1 := rPipe.Keys("CVLDB|*")
		rVal2 := rPipe.Keys("CVLDB|*|1")
		rVal3 := rPipe.Keys("CVLDB|event|2")
		rVal4 := rPipe.Keys("CVLDB|*|unknown")
		rPipe.Exec()
		verifyResult(tt, cVal1, rVal1)
		verifyResult(tt, cVal2, rVal2)
		verifyResult(tt, cVal3, rVal3)
		verifyResult(tt, cVal4, rVal4)
		pipe.Close()
		rPipe.Close()
	})

	// Pipe Keys, HMGet No Tx
	t.Run("Pipe_Keys_HMGet_No_Tx", func(tt *testing.T) {
		pipe := c1.Pipeline()
		cVal1 := pipe.Keys("CVLDB|*")
		cVal2 := pipe.HMGet("CVLDB|event|1", "message", "id")
		pipe.Exec()
		rPipe := c1.Db.client.Pipeline()
		rVal1 := rPipe.Keys("CVLDB|*")
		rVal2 := rPipe.HMGet("CVLDB|event|1", "message", "id")
		rPipe.Exec()
		verifyResult(tt, cVal1, rVal1)
		verifyResult(tt, cVal2, rVal2)
		pipe.Close()
		rPipe.Close()
	})

	// Pipe HGet() No Tx
	t.Run("Pipe_HGet_No_Tx", func(tt *testing.T) {
		pipe := c1.Pipeline()
		cVal1 := pipe.HGet("CVLDB|event|1", "message")
		cVal2 := pipe.HGet("CVLDB|event|1", "xyz")
		cVal3 := pipe.HGet("CVLDB|event|unknown", "message")
		pipe.Exec()
		rPipe := c1.Db.client.Pipeline()
		rVal1 := rPipe.HGet("CVLDB|event|1", "message")
		rVal2 := rPipe.HGet("CVLDB|event|1", "xyz")
		rVal3 := rPipe.HGet("CVLDB|event|unknown", "message")
		rPipe.Exec()
		verifyResult(tt, cVal1, rVal1)
		verifyResult(tt, cVal2, rVal2)
		verifyResult(tt, cVal3, rVal3)
		pipe.Close()
		rPipe.Close()
	})

	// Pipe HGetAll() No Tx
	t.Run("HGetAll", func(tt *testing.T) {
		pipe := c1.Pipeline()
		cVal1 := pipe.HGetAll("CVLDB|event|1")
		cVal2 := pipe.HGetAll("CVLDB|event|unknown")
		pipe.Exec()
		rPipe := c1.Db.client.Pipeline()
		rVal1 := rPipe.HGetAll("CVLDB|event|1")
		rVal2 := rPipe.HGetAll("CVLDB|event|unknown")
		rPipe.Exec()
		verifyResult(tt, cVal1, rVal1)
		verifyResult(tt, cVal2, rVal2)
		pipe.Close()
		rPipe.Close()
	})

	// Pipe Keys, HMGet, HGet,HGeAll No Tx
	t.Run("Pipe_HGet_HMGet_No_Tx", func(tt *testing.T) {
		pipe := c1.Pipeline()
		cVal1 := pipe.Keys("CVLDB|*")
		cVal2 := pipe.HMGet("CVLDB|event|1", "message", "id")
		cVal3 := pipe.HGet("CVLDB|event|1", "message")
		cVal4 := pipe.HGetAll("CVLDB|event|1")
		pipe.Exec()
		rPipe := c1.Db.client.Pipeline()
		rVal1 := rPipe.Keys("CVLDB|*")
		rVal2 := rPipe.HMGet("CVLDB|event|1", "message", "id")
		rVal3 := rPipe.HGet("CVLDB|event|1", "message")
		rVal4 := rPipe.HGetAll("CVLDB|event|1")
		rPipe.Exec()
		verifyResult(tt, cVal1, rVal1)
		verifyResult(tt, cVal2, rVal2)
		verifyResult(tt, cVal3, rVal3)
		verifyResult(tt, cVal4, rVal4)
		pipe.Close()
		rPipe.Close()
	})
}

func TestCvlDB_Search(t *testing.T) {
	d := newTestDB(t, ConfigDB)
	setupTestData(t, d.client, map[string]map[string]interface{}{
		"CVLDB_PORT|Eth1": {"mtu": "6789", "admin": "up"},
		"CVLDB_PORT|Po1":  {"mtu": "1444", "admin": "up"},
		"CVLDB_PORT|Po2":  {"mtu": "2444", "admin": "down"},
		"CVLDB_VLAN|1001": {"NULL": "NULL"},
		"CVLDB_VLAN|1002": {"members@": "Eth1,Po1"},
		"CVLDB_VLAN|1003": {"members@": "Eth1"},
		"CVLDB_VLAN|1004": {"members@": "Po2"},
		"CVLDB_VLAN|1005": {"NULL": "NULL"},
	})

	c := &cvlDBAccess{d}
	s := Search{}

	// Match entire table using a wildcard key Pattern
	s = Search{Pattern: "CVLDB_VLAN|*"}
	t.Run("Lookup_table", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{
			"1001": {"NULL": "NULL"},
			"1002": {"members@": "Eth1,Po1"},
			"1003": {"members@": "Eth1"},
			"1004": {"members@": "Po2"},
			"1005": {"NULL": "NULL"} }}`)
	})
	t.Run("Count_table", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(5))
	})

	// Match using a key Parttern only
	s = Search{Pattern: "CVLDB_PORT|Eth*"}
	t.Run("Lookup_keys", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT":{"Eth1": {"mtu": "6789", "admin": "up"}}}`)
	})
	t.Run("Count_keys", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(1))
	})

	// Match using a Predicate that checks k map only
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		KeyNames:  []string{"name"},
		Predicate: "string.find(k.name, 'Eth') == 1",
	}
	t.Run("Lookup_predicate_k", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT":{"Eth1": {"mtu": "6789", "admin": "up"}}}`)
	})
	t.Run("Count_predicate_k", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(1))
	})

	// Match using a Predicate that checks h map only
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		Predicate: "h['members@'] ~= nil and string.find(h['members@']..',', 'Eth1,') ~= nil",
	}
	t.Run("Lookup_predicate_h", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{
			"1002": {"members@": "Eth1,Po1"},
			"1003": {"members@": "Eth1"} }}`)
	})
	t.Run("Count_predicate_h", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(2))
	})

	// Match using a Predicate that checks both k and h maps
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		KeyNames:  []string{"id"},
		Predicate: "tonumber(k.id) > 1003 and h['NULL'] ~= nil",
	}
	t.Run("Lookup_predicate_kh", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{"1005": {"NULL": "NULL"}}}`)
	})
	t.Run("Count_predicate_kh", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(1))
	})

	// Match using a Predicate with return statement (existing cvl & custom validations)
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		Predicate: "return (h.admin == 'down')",
	}
	t.Run("Lookup_predicate_return", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT": {"Po2":  {"mtu": "2444", "admin": "down"}}}`)
	})
	t.Run("Count_predicate_return", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(1))
	})

	// Match using WithFields
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		WithField: "members@",
	}
	t.Run("Count_withField", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(3))
	})

	// Key not found case
	s = Search{Pattern: "CVLDB_VLAN|xxxx"}
	t.Run("Lookup_key_not_matched", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), strResult{"", redis.Nil})
	})
	t.Run("Count_key_not_matched", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(0))
	})

	// Predicate not matched
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		Predicate: "h.admin == 'xxx'",
	}
	t.Run("Lookup_predicate_not_matched", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), strResult{"", redis.Nil})
	})
	t.Run("Count_predicate_not_matched", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(0))
	})

}

func TestCvlDB_txn(t *testing.T) {
	d1 := newTestDB(t, ConfigDB)
	setupTestData(t, d1.client, map[string]map[string]interface{}{
		"CVLDB|TXN|aaa": {"id": 1, "sev": "hi", "message": "hello, world!"},
		"CVLDB|TXN|bbb": {"id": 2, "sev": "hi", "message": "foo bar"},
		"CVLDB|TXN|ccc": {"id": 3, "sev": "lo", "message": "nothing"},
		"OTHERTABLE|01": {"NULL": "NULL"},
		"OTHERTABLE|02": {"NULL": "NULL"},
	})

	if err := d1.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx failed;", err)
	}
	d1.ModEntry(
		&TableSpec{Name: "CVLDB"},
		Key{[]string{"TXN", "bbb"}},
		Value{map[string]string{"sev": "lo"}},
	)
	d1.DeleteEntryFields(
		&TableSpec{Name: "CVLDB"},
		Key{[]string{"TXN", "bbb"}},
		Value{map[string]string{"message": ""}},
	)
	d1.DeleteEntry(
		&TableSpec{Name: "CVLDB"},
		Key{[]string{"TXN", "ccc"}},
	)
	d1.ModEntry(
		&TableSpec{Name: "CVLDB"},
		Key{[]string{"TXN", "eee"}},
		Value{map[string]string{"id": "5", "sev": "md", "message": "new"}},
	)
	d1.ModEntry(
		&TableSpec{Name: "CVLDB"},
		Key{[]string{"NEW", "aaa"}},
		Value{map[string]string{"field1": "a1", "field2": "a2"}},
	)

	c1 := &cvlDBAccess{d1}

	// Exists()

	t.Run("Exists_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|TXN|eee"), int64(1))
	})
	t.Run("Exists_createdKey_newTable", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|NEW|aaa"), int64(1))
	})
	t.Run("Exists_updatedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|TXN|bbb"), int64(1))
	})
	t.Run("Exists_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|TXN|ccc"), int64(0))
	})
	t.Run("Exists_untouchKey", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|TXN|aaa"), int64(1))
	})
	t.Run("Exists_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Exists("CVLDB|TXN|xxx"), int64(0))
	})

	// Keys()

	t.Run("Keys_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|eee"),
			[]string{"CVLDB|TXN|eee"})
	})
	t.Run("Keys_createdKey_newTable", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|NEW|aaa"),
			[]string{"CVLDB|NEW|aaa"})
	})
	t.Run("Keys_updatedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|bbb"),
			[]string{"CVLDB|TXN|bbb"})
	})
	t.Run("Keys_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|ccc"), []string{})
	})
	t.Run("Keys_untouchKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|aaa"),
			[]string{"CVLDB|TXN|aaa"})
	})
	t.Run("Keys_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|xxx"), []string{})
	})
	t.Run("Keys_wc_all", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|*"),
			[]string{"CVLDB|TXN|aaa", "CVLDB|TXN|bbb", "CVLDB|TXN|eee", "CVLDB|NEW|aaa"})
	})
	t.Run("Keys_wc_middle", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|*|aaa"),
			[]string{"CVLDB|TXN|aaa", "CVLDB|NEW|aaa"})
	})
	t.Run("Keys_wc_suffix", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|TXN|*"),
			[]string{"CVLDB|TXN|aaa", "CVLDB|TXN|bbb", "CVLDB|TXN|eee"})
	})
	t.Run("Keys_wc_suffix_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|NEW|*"),
			[]string{"CVLDB|NEW|aaa"})
	})
	t.Run("Keys_wc_unknownTable", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("UNKNOWN_TABLE|*|aaa"), []string{})
	})
	t.Run("Keys_wc_deletedTable", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|*|ccc"), []string{})
	})
	t.Run("Keys_wc_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Keys("CVLDB|*|xxx"), []string{})
	})

	// HGet()

	hgetNoKey := d1.client.HGet("CVLDB|TXN|xxx", "xxx")
	hgetNoFld := d1.client.HGet("CVLDB|TXN|aaa", "xxx")

	t.Run("HGet_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|NEW|aaa", "field1"), "a1")
	})
	t.Run("HGet_createdKey_unknownField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|NEW|aaa", "fieldX"), hgetNoFld)
	})
	t.Run("HGet_updatedKey_untouchField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|bbb", "id"), "2")
	})
	t.Run("HGet_updatedKey_updatedField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|bbb", "sev"), "lo")
	})
	t.Run("HGet_updatedKey_deletedField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|bbb", "message"), hgetNoFld)
	})
	t.Run("HGet_updatedKey_unknownField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|bbb", "unknown"), hgetNoFld)
	})
	t.Run("HGet_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|ccc", "id"), hgetNoKey)
	})
	t.Run("HGet_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|xxx", "id"), hgetNoKey)
	})
	t.Run("HGet_untouchedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|aaa", "sev"), "hi")
	})
	t.Run("HGet_untouchedKey_unknownField", func(tt *testing.T) {
		verifyResult(tt, c1.HGet("CVLDB|TXN|aaa", "unknown"), hgetNoFld)
	})

	// HMGet()

	fields := []string{"id", "sev", "message", "unknown_field_xyz"}
	hmgetNoKey := d1.client.HMGet("CVLDB|TXN|xxx", fields...)

	t.Run("HMGet_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.HMGet("CVLDB|TXN|eee", fields...),
			[]interface{}{"5", "md", "new", nil})
	})
	t.Run("HMGet_updatedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HMGet("CVLDB|TXN|bbb", fields...),
			[]interface{}{"2", "lo", nil, nil})
	})
	t.Run("HMGet_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HMGet("CVLDB|TXN|ccc", fields...), hmgetNoKey)
	})
	t.Run("HMGet_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.HMGet("CVLDB|TXN|xxx", fields...), hmgetNoKey)
	})
	t.Run("HMGet_untouchedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HMGet("CVLDB|TXN|aaa", fields...),
			[]interface{}{"1", "hi", "hello, world!", nil})
	})

	// HGetAll()

	hgetallNoKey := d1.client.HGetAll("CVLDB|TXN|xxx")

	t.Run("HGetAll_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGetAll("CVLDB|TXN|eee"),
			map[string]string{"id": "5", "sev": "md", "message": "new"})
	})
	t.Run("HGetAll_updatedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGetAll("CVLDB|TXN|bbb"),
			map[string]string{"id": "2", "sev": "lo"})
	})
	t.Run("HGetAll_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGetAll("CVLDB|TXN|ccc"), hgetallNoKey)
	})
	t.Run("HGetAll_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGetAll("CVLDB|TXN|xxx"), hgetallNoKey)
	})
	t.Run("HGetAll_untouchedKey", func(tt *testing.T) {
		verifyResult(tt, c1.HGetAll("CVLDB|TXN|aaa"),
			map[string]string{"id": "1", "sev": "hi", "message": "hello, world!"})
	})

	// Count() by key patterns (no predicate)

	t.Run("CountKeys_createdKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|eee"}), int64(1))
	})
	t.Run("CountKeys_updatedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|bbb"}), int64(1))
	})
	t.Run("CountKeys_deletedKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|ccc"}), int64(0))
	})
	t.Run("CountKeys_untouchKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|aaa"}), int64(1))
	})
	t.Run("CountKeys_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|xxx"}), int64(0))
	})
	t.Run("CountKeys_unknownTable", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "UNKNOWNTABLE|xxx"}), int64(0))
	})
	t.Run("CountKeys_T|*", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|*"}), int64(4))
	})
	t.Run("CountKeys_T|*|*", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|*|*"}), int64(4))
	})
	t.Run("CountKeys_T|TXN|*", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|TXN|*"}), int64(3))
	})
	t.Run("CountKeys_T|NEW|*", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|NEW|*"}), int64(1))
	})
	t.Run("CountKeys_T|*|aaa", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "CVLDB|*|aaa"}), int64(2))
	})
	t.Run("CountKeys_otherTable", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "OTHERTABLE|*"}), int64(2))
	})
	t.Run("CountKeys_otherTable_knownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "OTHERTABLE|01"}), int64(1))
	})
	t.Run("CountKeys_otherTable_unknownKey", func(tt *testing.T) {
		verifyResult(tt, c1.Count(Search{Pattern: "OTHERTABLE|xxx"}), int64(0))
	})

	// Pipe Keys()
	t.Run("Pipe_Keys_1", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|TXN|eee")
		pr2 := pipe.Keys("CVLDB|NEW|aaa")
		pr3 := pipe.Keys("CVLDB|TXN|bbb")
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|eee"})
		verifyResult(tt, pr2,
			[]string{"CVLDB|NEW|aaa"})
		verifyResult(tt, pr3,
			[]string{"CVLDB|TXN|bbb"})
		pipe.Close()
	})
	t.Run("Pipe_Keys_2", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|TXN|bbb")
		pr2 := pipe.Keys("CVLDB|TXN|ccc")
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|bbb"})
		verifyResult(tt, pr2, []string{})
		pipe.Close()
	})
	t.Run("Pipe_Keys_3", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|TXN|aaa")
		pr2 := pipe.Keys("CVLDB|TXN|xxx")
		pr3 := pipe.Keys("CVLDB|*")
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|aaa"})
		verifyResult(tt, pr2, []string{})
		verifyResult(tt, pr3,
			[]string{"CVLDB|TXN|aaa", "CVLDB|TXN|bbb", "CVLDB|TXN|eee", "CVLDB|NEW|aaa"})
		pipe.Close()
	})
	t.Run("Pipe_Keys_4", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|*|aaa")
		pr2 := pipe.Keys("CVLDB|TXN|*")
		pr3 := pipe.Keys("CVLDB|NEW|*")
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|aaa", "CVLDB|NEW|aaa"})
		verifyResult(tt, pr2,
			[]string{"CVLDB|TXN|aaa", "CVLDB|TXN|bbb", "CVLDB|TXN|eee"})
		verifyResult(tt, pr3,
			[]string{"CVLDB|NEW|aaa"})
		pipe.Close()
	})
	t.Run("Pipe_Keys_5", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("UNKNOWN_TABLE|*|aaa")
		pr2 := pipe.Keys("CVLDB|*|ccc")
		pr3 := pipe.Keys("CVLDB|*|xxx")
		pipe.Exec()
		verifyResult(tt, pr1, []string{})
		verifyResult(tt, pr2, []string{})
		verifyResult(tt, pr3, []string{})
		pipe.Close()
	})

	// Pipe HMGet()
	fields = []string{"id", "sev", "message", "unknown_field_xyz"}
	hmgetNoKey = d1.client.HMGet("CVLDB|TXN|xxx", fields...)

	t.Run("Pipe_HMGet_1", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.HMGet("CVLDB|TXN|eee", fields...)
		pr2 := pipe.HMGet("CVLDB|TXN|bbb", fields...)
		pipe.Exec()
		verifyResult(tt, pr1,
			[]interface{}{"5", "md", "new", nil})
		verifyResult(tt, pr2,
			[]interface{}{"2", "lo", nil, nil})
		pipe.Close()
	})

	t.Run("Pipe_HMGet_2", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.HMGet("CVLDB|TXN|ccc", fields...)
		pr2 := pipe.HMGet("CVLDB|TXN|xxx", fields...)
		pr3 := pipe.HMGet("CVLDB|TXN|aaa", fields...)
		pipe.Exec()
		verifyResult(tt, pr1, hmgetNoKey)
		verifyResult(tt, pr2, hmgetNoKey)
		verifyResult(tt, pr3,
			[]interface{}{"1", "hi", "hello, world!", nil})
		pipe.Close()
	})

	//Pipe Keys, HMGet
	t.Run("Pipe_Keys_HMGet", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|TXN|eee")
		pr2 := pipe.Keys("CVLDB|NEW|aaa")
		pr3 := pipe.HMGet("CVLDB|TXN|eee", fields...)
		pr4 := pipe.HMGet("CVLDB|TXN|bbb", fields...)
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|eee"})
		verifyResult(tt, pr2,
			[]string{"CVLDB|NEW|aaa"})
		verifyResult(tt, pr3,
			[]interface{}{"5", "md", "new", nil})
		verifyResult(tt, pr4,
			[]interface{}{"2", "lo", nil, nil})
		pipe.Close()
	})

	// Pipe HGet()

	rPipe := d1.client.Pipeline()
	hgetNoKey = rPipe.HGet("CVLDB|TXN|xxx", "xxx")
	hgetNoFld = rPipe.HGet("CVLDB|TXN|aaa", "xxx")
	rPipe.Exec()
	rPipe.Close()

	t.Run("Pipe_HGet_1", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.HGet("CVLDB|NEW|aaa", "field1")
		pr2 := pipe.HGet("CVLDB|NEW|aaa", "fieldX")
		pr3 := pipe.HGet("CVLDB|TXN|bbb", "id")
		pr4 := pipe.HGet("CVLDB|TXN|bbb", "sev")
		pipe.Exec()
		verifyResult(tt, pr1, "a1")
		verifyResult(tt, pr2, hgetNoFld)
		verifyResult(tt, pr3, "2")
		verifyResult(tt, pr4, "lo")
		pipe.Close()
	})

	t.Run("Pipe_HGet_2", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.HGet("CVLDB|TXN|bbb", "message")
		pr2 := pipe.HGet("CVLDB|TXN|bbb", "unknown")
		pr3 := pipe.HGet("CVLDB|TXN|ccc", "id")
		pr4 := pipe.HGet("CVLDB|TXN|xxx", "id")
		pr5 := pipe.HGet("CVLDB|TXN|aaa", "sev")
		pr6 := pipe.HGet("CVLDB|TXN|aaa", "unknown")
		pipe.Exec()
		verifyResult(tt, pr1, hgetNoFld)
		verifyResult(tt, pr2, hgetNoFld)
		verifyResult(tt, pr3, hgetNoKey)
		verifyResult(tt, pr4, hgetNoKey)
		verifyResult(tt, pr5, "hi")
		verifyResult(tt, pr6, hgetNoFld)
		pipe.Close()
	})

	// Pipe HGetAll()
	rPipe = d1.client.Pipeline()
	hgetallNoKey = rPipe.HGetAll("CVLDB|TXN|xxx")
	rPipe.Exec()
	rPipe.Close()

	t.Run("Pipe_HGetAll", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.HGetAll("CVLDB|TXN|eee")
		pr2 := pipe.HGetAll("CVLDB|TXN|bbb")
		pr3 := pipe.HGetAll("CVLDB|TXN|ccc")
		pr4 := pipe.HGetAll("CVLDB|TXN|xxx")
		pr5 := pipe.HGetAll("CVLDB|TXN|aaa")
		pipe.Exec()
		verifyResult(tt, pr1,
			map[string]string{"id": "5", "sev": "md", "message": "new"})
		verifyResult(tt, pr2,
			map[string]string{"id": "2", "sev": "lo"})
		verifyResult(tt, pr3, hgetallNoKey)
		verifyResult(tt, pr4, hgetallNoKey)
		verifyResult(tt, pr5,
			map[string]string{"id": "1", "sev": "hi", "message": "hello, world!"})
		pipe.Close()
	})

	//Pipe Keys, HMGet, HGet, HGetAll
	t.Run("Pipe_Methods", func(tt *testing.T) {
		pipe := c1.Pipeline()
		pr1 := pipe.Keys("CVLDB|TXN|eee")
		pr2 := pipe.Keys("CVLDB|NEW|aaa")
		pr3 := pipe.HMGet("CVLDB|TXN|eee", fields...)
		pr4 := pipe.HMGet("CVLDB|TXN|bbb", fields...)
		pr5 := pipe.HGetAll("CVLDB|TXN|eee")
		pr6 := pipe.HGetAll("CVLDB|TXN|bbb")
		pr7 := pipe.HGet("CVLDB|TXN|bbb", "message")
		pr8 := pipe.HGet("CVLDB|TXN|bbb", "unknown")
		pipe.Exec()
		verifyResult(tt, pr1,
			[]string{"CVLDB|TXN|eee"})
		verifyResult(tt, pr2,
			[]string{"CVLDB|NEW|aaa"})
		verifyResult(tt, pr3,
			[]interface{}{"5", "md", "new", nil})
		verifyResult(tt, pr4,
			[]interface{}{"2", "lo", nil, nil})
		verifyResult(tt, pr5,
			map[string]string{"id": "5", "sev": "md", "message": "new"})
		verifyResult(tt, pr6,
			map[string]string{"id": "2", "sev": "lo"})
		verifyResult(tt, pr7, hgetNoFld)
		verifyResult(tt, pr8, hgetNoFld)
		pipe.Close()
	})
}

func TestCvlDB_Tx_Search(t *testing.T) {
	d := newTestDB(t, ConfigDB)
	setupTestData(t, d.client, map[string]map[string]interface{}{
		"CVLDB_PORT|Eth1": {"mtu": "6789", "admin": "up"},
		"CVLDB_PORT|Po1":  {"mtu": "1444", "admin": "up"},
		"CVLDB_PORT|Po2":  {"mtu": "2444", "admin": "down"},
		"CVLDB_VLAN|1001": {"NULL": "NULL"},
		"CVLDB_VLAN|1002": {"members@": "Eth1,Po1"},
		"CVLDB_VLAN|1003": {"members@": "Eth1"},
		"CVLDB_VLAN|1004": {"members@": "Po2"},
		"CVLDB_VLAN|1005": {"NULL": "NULL"},
	})

	if err := d.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx failed;", err)
	}

	d.DeleteEntry(
		&TableSpec{Name: "CVLDB_VLAN"},
		Key{[]string{"1001"}},
	)

	d.CreateEntry(
		&TableSpec{Name: "CVLDB_PORT"},
		Key{[]string{"Eth2"}},
		Value{map[string]string{"mtu": "6000", "admin": "up"}},
	)

	d.ModEntry(
		&TableSpec{Name: "CVLDB_PORT"},
		Key{[]string{"Po3"}},
		Value{map[string]string{"mtu": "2444", "admin": "down"}},
	)

	d.ModEntry(
		&TableSpec{Name: "CVLDB_VLAN"},
		Key{[]string{"22"}},
		Value{map[string]string{"members@": "Po1"}},
	)

	c := &cvlDBAccess{d}
	s := Search{}

	// Match entire table using a wildcard key Pattern
	s = Search{Pattern: "CVLDB_VLAN|*"}
	t.Run("Lookup_table_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{
			"1002": {"members@": "Eth1,Po1"},
			"1003": {"members@": "Eth1"},
			"22": {"members@": "Po1"},
			"1004": {"members@": "Po2"},
			"1005": {"NULL": "NULL"} }}`)
	})

	t.Run("Count_table_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(5))
	})

	// Match using a key Parttern only
	s = Search{Pattern: "CVLDB_PORT|Eth*"}
	t.Run("Lookup_keys_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT":{"Eth1": {"admin": "up", "mtu": "6789"},"Eth2": {"admin": "up", "mtu": "6000"}}}`)
	})
	t.Run("Count_keys_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(2))
	})

	// Match using a Predicate that checks k map only
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		KeyNames:  []string{"name"},
		Predicate: "string.find(k.name, 'Eth') == 1",
	}
	t.Run("Lookup_predicate_k_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT":{"Eth1": {"admin": "up", "mtu": "6789"},"Eth2": {"admin": "up", "mtu": "6000"}}}`)
	})

	t.Run("Count_predicate_k_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(2))
	})

	// Match using a Predicate that checks h map only
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		Predicate: "h['members@'] ~= nil and string.find(h['members@']..',', 'Po1,') ~= nil",
	}
	t.Run("Lookup_predicate_h_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{
			"1002": {"members@": "Eth1,Po1"},
			"22": {"members@": "Po1"} }}`)
	})

	t.Run("Count_predicate_h_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(2))
	})

	// Match using a Predicate that checks both k and h maps
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		KeyNames:  []string{"id"},
		Predicate: "tonumber(k.id) > 1000 and h['NULL'] ~= nil",
	}
	t.Run("Lookup_predicate_kh_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_VLAN":{"1005": {"NULL": "NULL"}}}`)
	})
	t.Run("Count_predicate_kh_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(1))
	})

	// Match using a Predicate with return statement (existing cvl & custom validations)
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		Predicate: "return (h.admin == 'down')",
	}

	t.Run("Lookup_predicate_return_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), `{"CVLDB_PORT": {
			"Po2":  {"mtu": "2444", "admin": "down"},
			"Po3":  {"mtu": "2444", "admin": "down"}}}`)
	})

	t.Run("Count_predicate_return_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(2))
	})

	// Match using WithFields
	s = Search{
		Pattern:   "CVLDB_VLAN|*",
		WithField: "members@",
	}
	t.Run("Count_withField_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(4))
	})

	// Key not found case
	s = Search{Pattern: "CVLDB_VLAN|xxxx"}
	t.Run("Lookup_key_not_matched_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), strResult{"", redis.Nil})
	})
	t.Run("Count_key_not_matched_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(0))
	})

	// Predicate not matched
	s = Search{
		Pattern:   "CVLDB_PORT|*",
		Predicate: "h.admin == 'xxx'",
	}
	t.Run("Lookup_predicate_not_matched_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Lookup(s), strResult{"", redis.Nil})
	})
	t.Run("Count_predicate_not_matched_Tx", func(tt *testing.T) {
		verifyResult(tt, c.Count(s), int64(0))
	})
}

func TestScaleCvlDb_Tx_Count(t *testing.T) {
	d := newTestDB(t, ConfigDB)
	if err := d.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx failed;", err)
	}
	for i := 1; i <= 10000; i++ {
		d.CreateEntry(
			&TableSpec{Name: "VLAN"},
			Key{[]string{fmt.Sprint("Vlan", i)}},
			Value{map[string]string{"admin_status": "up", "vlanid": fmt.Sprint(i)}},
		)
	}

	c := &cvlDBAccess{d}
	s := Search{Pattern: "VLAN|*", Predicate: "h.admin_status == 'up'"}
	verifyResult(t, c.Count(s), int64(10000))
}

func TestScaleCvlDb_Tx_Lookup(t *testing.T) {
	d := newTestDB(t, ConfigDB)
	if err := d.StartTx(nil, nil); err != nil {
		t.Fatal("StartTx failed;", err)
	}
	d.CreateEntry(
		&TableSpec{Name: "VLAN"},
		Key{[]string{"Vlan1"}},
		Value{map[string]string{"admin_status": "down", "vlanid": "1"}},
	)
	d.CreateEntry(
		&TableSpec{Name: "VLAN"},
		Key{[]string{"Vlan2"}},
		Value{map[string]string{"admin_status": "down", "vlanid": "2"}},
	)
	for i := 3; i <= 10000; i++ {
		d.CreateEntry(
			&TableSpec{Name: "VLAN"},
			Key{[]string{fmt.Sprint("Vlan", i)}},
			Value{map[string]string{"admin_status": "up", "vlanid": fmt.Sprint(i)}},
		)
	}

	c := &cvlDBAccess{d}
	s := Search{Pattern: "VLAN|*", Predicate: "h.admin_status == 'down'"}
	exp := `{"VLAN":{"Vlan1":{"admin_status":"down","vlanid":"1"},"Vlan2":{"admin_status":"down","vlanid":"2"}}}`
	verifyResult(t, c.Lookup(s), exp)
}

func BenchmarkCvlDb_Tx_Count(b *testing.B) {
	b.StopTimer()
	d := newTestDB(b, ConfigDB)

	if err := d.StartTx(nil, nil); err != nil {
		b.Fatal("StartTx failed;", err)
	}
	for i := 1; i <= 10000; i++ {
		d.CreateEntry(
			&TableSpec{Name: "VLAN"},
			Key{[]string{fmt.Sprint("Vlan", i)}},
			Value{map[string]string{"admin_status": "up", "vlanid": fmt.Sprint(i)}},
		)
	}

	c := &cvlDBAccess{d}
	s := Search{Pattern: "VLAN|*", Predicate: "h.admin_status == 'up'"}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if _, err := c.Count(s).Result(); err != nil {
			b.Fatal("Count api failed;", err)
		}
	}
}

func BenchmarkCvlDb_Tx_Lookup(b *testing.B) {
	b.StopTimer()
	d := newTestDB(b, ConfigDB)

	if err := d.StartTx(nil, nil); err != nil {
		b.Fatal("StartTx failed;", err)
	}
	d.CreateEntry(
		&TableSpec{Name: "VLAN"},
		Key{[]string{"Vlan1"}},
		Value{map[string]string{"admin_status": "down", "vlanid": "1"}},
	)
	d.CreateEntry(
		&TableSpec{Name: "VLAN"},
		Key{[]string{"Vlan2"}},
		Value{map[string]string{"admin_status": "down", "vlanid": "2"}},
	)
	for i := 3; i <= 10000; i++ {
		d.CreateEntry(
			&TableSpec{Name: "VLAN"},
			Key{[]string{fmt.Sprint("Vlan", i)}},
			Value{map[string]string{"admin_status": "up", "vlanid": fmt.Sprint(i)}},
		)
	}

	c := &cvlDBAccess{d}
	s := Search{Pattern: "VLAN|*", Predicate: "h.admin_status == 'down'"}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if _, err := c.Lookup(s).Result(); err != nil {
			b.Fatal("Lookup api failed;", err)
		}
	}
}
