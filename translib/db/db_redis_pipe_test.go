////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"os"
	"reflect"
	"strconv"
	"testing"
)

//func init() {
//	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
//	var logLevel string
//	flag.StringVar(&logLevel, "logLevel", "4", "test")
//	flag.Lookup("v").Value.Set(logLevel)
//}

func newDB(dBNum DBNum) (*DB, error) {
	d, e := NewDB(Options{
		DBNo:               dBNum,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})
	return d, e
}

func deleteTableAndDb(d *DB, ts *TableSpec, t *testing.T) {
	e := d.DeleteTable(ts)

	if e != nil {
		t.Errorf("DeleteTable() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func TestGetEntries1(t *testing.T) {

	var pid int = os.Getpid()

	d, e := newDB(ConfigDB)

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key{Comp: ca2}

	// Add the Entries for Get|DeleteKeys

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v, keys = %v", e, keys)
		return
	}

	e = d.DeleteKeys(&ts, Key{Comp: []string{"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	// Add the Entries again for Table

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e = d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	values, errors := d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	for _, value := range values {
		if reflect.DeepEqual(value, avalue) {
			continue
		} else {
			t.FailNow()
		}
	}

	if errors != nil {
		t.FailNow()
	}

	deleteTableAndDb(d, &ts, t)
}

// to test by giving the duplicate key
func TestGetEntries2(t *testing.T) {

	var pid int = os.Getpid()

	d, e := newDB(ConfigDB)

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key{Comp: ca2}

	// Add the Entries for Get|DeleteKeys

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	e = d.DeleteKeys(&ts, Key{Comp: []string{"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	// Add the Entries again for Table

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys = make([]Key, 0)
	keys = append(keys, akey)
	keys = append(keys, akey)

	values, errors := d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	if len(values) != 2 {
		t.FailNow()
	}

	t.Log("avalue ==> : ", avalue)

	for idx, value := range values {
		if reflect.DeepEqual(value, avalue) {
			continue
		} else {
			t.Log("value not matching for the key: ", keys[idx])
			t.FailNow()
		}
	}

	deleteTableAndDb(d, &ts, t)
}

// to test the errors slice by giving one of the invalid key
func TestGetEntries3(t *testing.T) {

	var pid int = os.Getpid()

	d, e := newDB(ConfigDB)

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key{Comp: ca2}

	// Add the Entries for Get|DeleteKeys

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	e = d.DeleteKeys(&ts, Key{Comp: []string{"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	// Add the Entries again for Table

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	ca3 := make([]string, 1, 1)
	ca3[0] = "KEY_NOT_EXIST"
	akey3 := Key{Comp: ca3}

	keys = make([]Key, 0)
	keys = append(keys, akey)
	keys = append(keys, akey)
	keys = append(keys, akey3)

	values, errors := d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	if errors != nil && errors[2] != nil {
		t.Log("Error received correctly for the key ", keys[2], ";  error: ", errors[2])
	} else {
		t.Log("Error not getting received for the key: ", keys[2])
		t.FailNow()
	}

	deleteTableAndDb(d, &ts, t)
}

// To test cache hit by enabling the PerConnection, CacheTables
func TestGetEntries4(t *testing.T) {

	var pid int = os.Getpid()

	d, e := newDB(ConfigDB)

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	d.dbCacheConfig.PerConnection = true
	d.dbCacheConfig.CacheTables[ts.Name] = true

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key{Comp: ca2}

	// Add the Entries for Get|DeleteKeys

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	avalue2 := Value{map[string]string{"ports@": "Ethernet1", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey2, avalue2)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	e = d.DeleteKeys(&ts, Key{Comp: []string{"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	// Add the Entries again for Table

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue2)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	d.dbStatsConfig.TableStats = true
	d.dbStatsConfig.TimeStats = true

	keys = make([]Key, 0)
	keys = append(keys, akey)
	keys = append(keys, akey2)

	values, errors := d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	stats := d.stats.Tables[ts.Name]
	t.Log("stats.GetEntryCacheHits: ", stats.GetEntryCacheHits)
	t.Log("stats.GetEntriesHits: ", stats.GetEntriesHits)
	t.Log("stats.GetEntriesPeak: ", stats.GetEntriesPeak)
	t.Log("stats.GetEntriesTime: ", stats.GetEntriesTime)

	values, errors = d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	stats = d.stats.Tables[ts.Name]
	t.Log("stats.GetEntryCacheHits: ", stats.GetEntryCacheHits)
	t.Log("stats.GetEntriesHits: ", stats.GetEntriesHits)
	t.Log("stats.GetEntriesPeak: ", stats.GetEntriesPeak)
	t.Log("stats.GetEntriesTime: ", stats.GetEntriesTime)

	if stats.GetEntryCacheHits != 2 || stats.GetEntriesHits != 2 {
		t.FailNow()
	}

	if stats.GetEntriesPeak == 0 || stats.GetEntriesTime == 0 {
		t.FailNow()
	}

	deleteTableAndDb(d, &ts, t)
}

// To test cache hit by populating the one of the entry in the transaction map
func TestGetEntries5(t *testing.T) {

	var pid int = os.Getpid()

	d, e := newDB(ConfigDB)

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	d.dbCacheConfig.PerConnection = true
	d.dbCacheConfig.CacheTables[ts.Name] = true

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key{Comp: ca2}

	// Add the Entries for Get|DeleteKeys

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	avalue2 := Value{map[string]string{"ports@": "Ethernet1", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey2, avalue2)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	e = d.DeleteKeys(&ts, Key{Comp: []string{"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	// Add the Entries again for Table

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey2, avalue2)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	d.dbStatsConfig.TableStats = true
	d.dbStatsConfig.TimeStats = true

	keys = make([]Key, 0)
	keys = append(keys, akey)
	keys = append(keys, akey2)

	values, errors := d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	stats := d.stats.Tables[ts.Name]
	t.Log("stats.GetEntryCacheHits: ", stats.GetEntryCacheHits)
	t.Log("stats.GetEntriesHits: ", stats.GetEntriesHits)
	t.Log("stats.GetEntriesPeak: ", stats.GetEntriesPeak)
	t.Log("stats.GetEntriesTime: ", stats.GetEntriesTime)

	d.txTsEntryMap = make(map[string]map[string]Value)
	d.txTsEntryMap[ts.Name] = make(map[string]Value)
	d.txTsEntryMap[ts.Name][d.key2redis(&ts, akey)] = avalue

	values, errors = d.GetEntries(&ts, keys)
	t.Log("values: ", values)
	t.Log("errors: ", errors)

	stats = d.stats.Tables[ts.Name]
	t.Log("stats.GetEntryCacheHits: ", stats.GetEntryCacheHits)
	t.Log("stats.GetEntriesHits: ", stats.GetEntriesHits)
	t.Log("stats.GetEntriesPeak: ", stats.GetEntriesPeak)
	t.Log("stats.GetEntriesTime: ", stats.GetEntriesTime)

	if stats.GetEntryCacheHits != 1 || stats.GetEntriesHits != 2 {
		t.FailNow()
	}

	if stats.GetEntriesPeak == 0 || stats.GetEntriesTime == 0 {
		t.FailNow()
	}

	deleteTableAndDb(d, &ts, t)
}
