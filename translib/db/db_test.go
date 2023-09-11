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

package db

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

var dbConfig = `
{
    "INSTANCES": {
        "redis":{
            "hostname" : "127.0.0.1",
            "port" : 6379,
            "unix_socket_path" : "/var/run/redis/redis.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "redis2":{
            "hostname" : "127.0.0.1",
            "port" : 63792,
            "unix_socket_path" : "/var/run/redis/redis2.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "redis3":{
           "hostname" : "127.0.0.1",
            "port" : 63793,
            "unix_socket_path" : "/var/run/redis/redis3.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "rediswb":{
            "hostname" : "127.0.0.1",
            "port" : 63970,
            "unix_socket_path" : "/var/run/redis/rediswb.sock",
            "persistence_for_warm_boot" : "yes"
        }
    },
    "DATABASES" : {
        "APPL_DB" : {
            "id" : 0,
            "separator": ":",
            "instance" : "redis2"
        },
        "ASIC_DB" : {
            "id" : 1,
            "separator": ":",
            "instance" : "redis3"
        },
        "COUNTERS_DB" : {
            "id" : 2,
            "separator": ":",
            "instance" : "redis"
        },
        "CONFIG_DB" : {
            "id" : 4,
            "separator": "|",
            "instance" : "redis"
        },
        "PFC_WD_DB" : {
            "id" : 5,
            "separator": ":",
            "instance" : "redis"
        },
        "FLEX_COUNTER_DB" : {
            "id" : 5,
            "separator": ":",
            "instance" : "redis"
        },
        "STATE_DB" : {
            "id" : 6,
            "separator": "|",
            "instance" : "redis"
        },
        "SNMP_OVERLAY_DB" : {
            "id" : 7,
            "separator": "|",
            "instance" : "redis"
        }
    },
    "VERSION" : "1.0"
}
`

// "TEST_" prefix is used by a lot of DB Tests. Avoid it.
const DBPAT_TST_PREFIX string = "DBPAT_TST"

var ts TableSpec = TableSpec{
	Name: DBPAT_TST_PREFIX + strconv.FormatInt(int64(os.Getpid()), 10),
}
var db *DB
var dbOnC *DB

func newReadOnlyDB(dBNum DBNum) (*DB, error) {
	d, e := NewDB(Options{
		DBNo:               dBNum,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
		IsWriteDisabled:    true,
	})
	return d, e
}

func newOnCDB(dBNum DBNum) (*DB, error) {
	d, e := NewDB(Options{
		DBNo:               dBNum,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
		IsWriteDisabled:    true,
		IsOnChangeEnabled:  true,
	})
	return d, e
}

// setupTestData populates given test entries in db and deletes all those keys
// whne the test case ends.
func setupTestData(t *testing.T, redis *redis.Client, data map[string]map[string]interface{}) {
	keys := make([]string, 0, len(data))
	t.Cleanup(func() { redis.Del(keys...) })
	for k, v := range data {
		keys = append(keys, k)
		if _, err := redis.HMSet(k, v).Result(); err != nil {
			t.Fatalf("HMSET %s failed; err=%v", k, err)
		}
	}
}

func testTableSetup(tableEntries int) {
	var err error
	db, err = newDB(ConfigDB)
	if err != nil {
		fmt.Printf("newDB() fails err = %v\n", err)
		return
	}

	for i := 0; i < tableEntries; i++ {
		e := db.SetEntry(&ts,
			Key{Comp: []string{"KEY" + strconv.FormatInt(int64(i), 10)}},
			Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}})
		if e != nil {
			fmt.Printf("SetEntry() fails e = %v\n", e)
			return
		}
	}

	db.DeleteDB()
	db, err = newReadOnlyDB(ConfigDB)
	if err != nil {
		fmt.Printf("newReadOnlyDB() fails err = %v\n", err)
		return
	}

	dbOnC, err = newOnCDB(ConfigDB)
	if err != nil {
		fmt.Printf("newDB() for OnC fails err = %v\n", err)
		return
	}

}

func testTableTearDown(tableEntries int) {
	var err error
	if db != nil {
		db.DeleteDB()
	}
	db, err = newDB(ConfigDB)
	if err != nil {
		fmt.Printf("newDB() fails err = %v\n", err)
		return
	}

	for i := 0; i < tableEntries; i++ {
		e := db.DeleteEntry(&ts,
			Key{Comp: []string{"KEY" + strconv.FormatInt(int64(i), 10)}})
		if e != nil {
			fmt.Printf("DeleteEntry() fails e = %v", e)
			return
		}
	}

	db.DeleteDB()

	dbOnC.DeleteDB()

}

func TestMain(m *testing.M) {

	exitCode := 0

	testTableSetup(100)
	if exitCode == 0 {
		exitCode = m.Run()
	}
	testTableTearDown(100)

	os.Exit(exitCode)

}

/*

1.  Create, and close a DB connection. (NewDB(), DeleteDB())

*/

func TestNewDB(t *testing.T) {

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
	} else if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

/*

2.  Get an entry (GetEntry())
3.  Set an entry without Transaction (SetEntry())
4.  Delete an entry without Transaction (DeleteEntry())

20. NT: GetEntry() EntryNotExist.

*/

func TestNoTransaction(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	v, e := d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() fails e = %v", e)
		return
	}

	e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteEntry() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

/*

5.  Get a Table (GetTable())

9.  Get multiple keys (GetKeys())
10. Delete multiple keys (DeleteKeys())
11. Delete Table (DeleteTable())

*/

func TestTable(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

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

	v, e := d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteKeys() fails e = %v", e)
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

	tab, e := d.GetTable(&ts)

	if e != nil {
		t.Errorf("GetTable() fails e = %v", e)
		return
	}

	v, e = tab.GetEntry(akey)

	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("Table.GetEntry() fails e = %v", e)
		return
	}

	e = d.DeleteTable(&ts)

	if e != nil {
		t.Errorf("DeleteTable() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteTable() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

/* Tests for

6.  Set an entry with Transaction (StartTx(), SetEntry(), CommitTx())
7.  Delete an entry with Transaction (StartTx(), DeleteEntry(), CommitTx())
8.  Abort Transaction. (StartTx(), DeleteEntry(), AbortTx())

12. Set an entry with Transaction using WatchKeys Check-And-Set(CAS)
13. Set an entry with Transaction using Table CAS
14. Set an entry with Transaction using WatchKeys, and Table CAS

15. Set an entry with Transaction with empty WatchKeys, and Table CAS
16. Negative Test(NT): Fail a Transaction using WatchKeys CAS
17. NT: Fail a Transaction using Table CAS
18. NT: Abort an Transaction with empty WatchKeys/Table CAS

Cannot Automate 19 for now
19. NT: Check V logs, Error logs

*/

func TestTransaction(t *testing.T) {
	for transRun := TransRunBasic; transRun < TransRunEnd; transRun++ {
		testTransaction(t, transRun)
	}
}

type TransRun int

const (
	TransRunBasic                  TransRun = iota // 0
	TransRunWatchKeys                              // 1
	TransRunTable                                  // 2
	TransRunWatchKeysAndTable                      // 3
	TransRunEmptyWatchKeysAndTable                 // 4
	TransRunFailWatchKeys                          // 5
	TransRunFailTable                              // 6

	// Nothing after this.
	TransRunEnd
)

const (
	TransCacheRunGetAfterCreate            TransRun = iota // 0
	TransCacheRunGetAfterSingleSet                         // 1
	TransCacheRunGetAfterMultiSet                          // 2
	TransCacheRunGetAfterMod                               // 3
	TransCacheRunGetAfterDelEntry                          // 4
	TransCacheRunGetAfterDelField                          // 5
	TransCacheRunGetWithInvalidKey                         // 6
	TransCacheGetKeysAfterSetAndDeleteKeys                 // 7
	TransCacheGetKeysWithoutSet                            // 8
	TransCacheDelEntryEmpty                                // 9
	TransCacheDelFieldsEmpty                               // 10

	// Nothing after this.
	TransCacheRunEnd
)

func TestTransactionCache(t *testing.T) {
	// Tests without any data pre-existing in DB
	for transRun := TransCacheRunGetAfterCreate; transRun <= TransCacheRunEnd; transRun++ {
		testTransactionCache(t, transRun)
	}
}

//TestTransactionCacheWithDBContentKeysPattern
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetKeysPattern checks for number of required required
2. DeleteEntry and then GetKeysPattern, checks for number of required required
*/
func TestTransactionCacheWithDBContentKeysPattern(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "DUMMY_ACL_1"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	keys, e := d.GetKeysPattern(&ts, Key{Comp: []string{"DUMMY_ACL_*"}})

	if (e != nil) || (len(keys) != 1) || (!keys[0].Equals(akey)) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}
	ca[0] = "DUMMY_ACL_2"
	akey = Key{Comp: ca}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	keys, e = d.GetKeysPattern(&ts, Key{Comp: []string{"DUMMY_ACL_*"}})

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey)
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	keys, e = d.GetKeysPattern(&ts, Key{Comp: []string{"DUMMY_ACL_*"}})

	if (e != nil) || (len(keys) != 1) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}
	ca[0] = "DUMMY_ACL_1"
	akey = Key{Comp: ca}
	e = d.DeleteEntry(&ts, akey)
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheMultiKeysPattern
/*
1. Sets a Table entry with multikey
2. Performs GetEntry, GetKeysPattern and GetKeysByPattern
3. Deletes an entry
4. Re-Performs GetEntry and GetKeysPattern
*/
func TestTransactionCacheMultiKeysPattern(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 2, 2)
	ca[0] = "Vlan10"
	ca[1] = "Ethernet0"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}

	e = d.StartTx(nil, nil)
	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}

	keys, e := d.GetKeysPattern(&ts, Key{Comp: []string{"*", "*Ethernet0"}})

	if (e != nil) || (len(keys) != 1) || (!keys[0].Equals(akey)) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}

	keys, e = d.GetKeysByPattern(&ts, "*Ethernet0")

	if (e != nil) || (len(keys) != 1) || (!keys[0].Equals(akey)) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}

	e = d.DeleteEntry(&ts, akey)
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	v, e = d.GetEntry(&ts, akey)
	if e == nil {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}

	keys, e = d.GetKeysPattern(&ts, Key{Comp: []string{"*", "*Ethernet0"}})
	if (e != nil) || (len(keys) != 0) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}

	keys, e = d.GetKeysByPattern(&ts, "*Ethernet0")

	if (e != nil) || (len(keys) != 0) {
		t.Errorf("GetKeysPattern() fails e = %v", e)
		return
	}

	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheWithDBContentKeys
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetKeys checks for number of required required
2. DeleteEntry and then GetKeys, checks for number of required required
*/
func TestTransactionCacheWithDBContentKeys(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	keys, e := d.GetKeys(&ts) //DB get verify

	if (e != nil) || (len(keys) != 1) || (!keys[0].Equals(akey)) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey)
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	keys, e = d.GetKeys(&ts) //Cache get verify

	if (e != nil) || (len(keys) != 0) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheWithDBContentDel
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetEntry
2. DeleteEntry and then GetEntry
*/
func TestTransactionCacheWithDBContentDel(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	v, e = d.GetEntry(&ts, akey) //DB get verify
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey)
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	_, e = d.GetEntry(&ts, akey) //verify from cache
	if e == nil {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheWithDBContentDelFields
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetEntry
2. DeleteEntryFields and then GetEntry
*/
func TestTransactionCacheWithDBContentDelFields(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "changed desc"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	v, e = d.GetEntry(&ts, akey) //DB get verify
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	avalue2 := Value{map[string]string{"policy_desc": "changed desc"}}
	avalue3 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.DeleteEntryFields(&ts, akey, avalue2)
	if e != nil {
		t.Errorf("DeleteEntryFields() fails e = %v", e)
		return
	}
	v, e = d.GetEntry(&ts, akey) //verify from cache
	if (e != nil) || (!reflect.DeepEqual(v, avalue3)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey) //verify from cache
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheWithDBContentMod
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetEntry
2. ModEntry and then GetEntry
*/
func TestTransactionCacheWithDBContentMod(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	v, e = d.GetEntry(&ts, akey) //DB get verify
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	avalue2 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "changed desc"}}
	e = d.ModEntry(&ts, akey, avalue2)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e = d.GetEntry(&ts, akey) //verify from cache
	if (e != nil) || (!reflect.DeepEqual(v, avalue2)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey) //verify from cache
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

//TestTransactionCacheWithDBContentSet
/*
Add a new entry for a table who has already has one entry pre-exisint in DB and performs below checks.
1. GetEntry
2. SetEntry and then GetEntry
*/
func TestTransactionCacheWithDBContentSet(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	e = d.StartTx(nil, nil)

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
	e = d.SetEntry(&ts, akey, avalue)
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e := d.GetEntry(&ts, akey)
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}
	e = d.StartTx(nil, nil)
	v, e = d.GetEntry(&ts, akey) //DB get verify
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.SetEntry(&ts, akey, avalue) //SET tx cache
	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}
	v, e = d.GetEntry(&ts, akey) //verify from cache
	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}
	e = d.DeleteEntry(&ts, akey) //verify from cache
	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func testTransactionCache(t *testing.T, transRun TransRun) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v, transRun = %v", e, transRun)
		return
	}

	e = d.StartTx(nil, nil)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

	switch transRun {
	case TransCacheRunGetAfterCreate:
		//Performs GetEntry after Create
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
		e = d.CreateEntry(&ts, akey, avalue)
		if e != nil {
			t.Errorf("CreateEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)

		if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheRunGetAfterSingleSet:
		//Performs GetEntry after single SetEntry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
		e = d.SetEntry(&ts, akey, avalue)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheRunGetAfterMultiSet:
		//Performs GetEntry after multiple SetEntry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue1 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "some desc"}}
		e = d.SetEntry(&ts, akey, avalue1)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue1)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
		avalue2 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
		e = d.SetEntry(&ts, akey, avalue2)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e = d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue2)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheRunGetAfterMod:
		//Performs GetEntry after ModEntry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue1 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "some desc"}}
		e = d.SetEntry(&ts, akey, avalue1)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue1)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
		avalue2 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "changed desc"}}
		e = d.ModEntry(&ts, akey, avalue2)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e = d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue2)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheRunGetWithInvalidKey:
		//Performs GetEntry for invalid Entry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		_, e := d.GetEntry(&ts, akey)
		if e == nil {
			t.Errorf("GetEntry() should report error")
			return
		}
	case TransCacheRunGetAfterDelEntry:
		//Performs GetEntrys After DelEntry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue1 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "some desc"}}
		e = d.SetEntry(&ts, akey, avalue1)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue1)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
		e = d.DeleteEntry(&ts, akey)
		if e != nil {
			t.Errorf("DeleteEntry() fails e = %v", e)
			return
		}
		v, e = d.GetEntry(&ts, akey)
		if (e == nil) || (reflect.DeepEqual(v, avalue1)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheRunGetAfterDelField:
		//Performs GetEntrys After DelEntryFields
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue1 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "some desc"}}
		e = d.SetEntry(&ts, akey, avalue1)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}
		v, e := d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue1)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
		avalue2 := Value{map[string]string{"policy_desc": "some desc"}}
		e = d.DeleteEntryFields(&ts, akey, avalue2)
		if e != nil {
			t.Errorf("DeleteEntryFields() fails e = %v", e)
			return
		}
		avalue3 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}
		v, e = d.GetEntry(&ts, akey)
		if (e != nil) || (!reflect.DeepEqual(v, avalue3)) {
			t.Errorf("GetEntry() after Tx fails e = %v", e)
			return
		}
	case TransCacheGetKeysAfterSetAndDeleteKeys:
		//Performs GetKeys After Set and Delete of Keys
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue1 := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR", "policy_desc": "some desc"}}
		e = d.SetEntry(&ts, akey, avalue1)
		if e != nil {
			t.Errorf("SetEntry() fails e = %v", e)
			return
		}

		keys, e := d.GetKeys(&ts)

		if (e != nil) || (len(keys) != 1) || (!keys[0].Equals(akey)) {
			t.Errorf("GetKeys() fails e = %v", e)
			return
		}

		e = d.DeleteKeys(&ts, akey)

		if e != nil {
			t.Errorf("DeleteKeys() fails e = %v", e)
			return
		}

		keys, e = d.GetKeys(&ts)

		if (e != nil) || (len(keys) != 0) {
			t.Errorf("GetKeys() fails e = %v", e)
			return
		}
	case TransCacheGetKeysWithoutSet:
		//Performs GetKeys on non-existing table spec
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		keys, e := d.GetKeys(&ts)

		if (e != nil) || (len(keys) != 0) {
			t.Errorf("GetKeys() fails e = %v", e)
			return
		}
	case TransCacheDelEntryEmpty:
		//Performs DelEntry on non-existing entry
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		e = d.DeleteEntry(&ts, akey)
		if e != nil {
			t.Errorf("DeleteEntry() fails e = %v", e)
			return
		}
	case TransCacheDelFieldsEmpty:
		//performs deleteEntryFields on non-existing entry field
		ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

		ca := make([]string, 1, 1)
		ca[0] = "MyACL1_ACL_IPVNOTEXIST"
		akey := Key{Comp: ca}
		avalue := Value{map[string]string{"policy_desc": "some desc"}}
		e = d.DeleteEntryFields(&ts, akey, avalue)
		if e != nil {
			t.Errorf("DeleteEntryFields() fails e = %v", e)
			return
		}
	}

	e = d.AbortTx()

	if e != nil {
		t.Errorf("AbortTx() fails e = %v", e)
		return
	}

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func testTransaction(t *testing.T, transRun TransRun) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v, transRun = %v", e, transRun)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}

	var watchKeys []WatchKeys
	var table []*TableSpec

	switch transRun {
	case TransRunBasic, TransRunWatchKeysAndTable:
		watchKeys = []WatchKeys{{Ts: &ts, Key: &akey}}
		table = []*TableSpec{&ts}
	case TransRunWatchKeys, TransRunFailWatchKeys:
		watchKeys = []WatchKeys{{Ts: &ts, Key: &akey}}
		table = []*TableSpec{}
	case TransRunTable, TransRunFailTable:
		watchKeys = []WatchKeys{}
		table = []*TableSpec{&ts}
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

	e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	v, e := d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

	e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	e = d.AbortTx()

	if e != nil {
		t.Errorf("AbortTx() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v, avalue)) {
		t.Errorf("GetEntry() after Abort Tx fails e = %v", e)
		return
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

	e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	var lockFail bool
	switch transRun {
	case TransRunFailWatchKeys, TransRunFailTable:
		d2, e2 := NewDB(Options{
			DBNo:               ConfigDB,
			InitIndicator:      "",
			TableNameSeparator: "|",
			KeySeparator:       "|",
			DisableCVLCheck:    true,
		})

		if e2 != nil {
			lockFail = true
			break
		}

		d2.StartTx(watchKeys, table)
		d2.DeleteEntry(&ts, akey)
		d2.CommitTx()
		d2.DeleteDB()
	default:
	}

	e = d.CommitTx()

	switch transRun {
	case TransRunFailWatchKeys, TransRunFailTable:
		if !lockFail && e == nil {
			t.Errorf("NT CommitTx() tr: %v fails e = %v",
				transRun, e)
			return
		}
	default:
		if e != nil {
			t.Errorf("CommitTx() fails e = %v", e)
			return
		}
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after Tx DeleteEntry() fails e = %v", e)
		return
	}

	d.DeleteMapAll(&ts)

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func TestMap(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TESTMAP_" + strconv.FormatInt(int64(pid), 10)}

	d.SetMap(&ts, "k1", "v1")
	d.SetMap(&ts, "k2", "v2")

	if v, e := d.GetMap(&ts, "k1"); v != "v1" {
		t.Errorf("GetMap() fails e = %v", e)
		return
	}

	if v, e := d.GetMapAll(&ts); (e != nil) ||
		(!reflect.DeepEqual(v,
			Value{Field: map[string]string{
				"k1": "v1", "k2": "v2"}})) {
		t.Errorf("GetMapAll() fails e = %v", e)
		return
	}

	d.DeleteMapAll(&ts)

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func TestSubscribe(t *testing.T) {

	var pid int = os.Getpid()

	var hSetCalled, hDelCalled, delCalled bool

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	})

	if (d == nil) || (e != nil) {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}

	var skeys []*SKey = make([]*SKey, 1)
	skeys[0] = &(SKey{Ts: &ts, Key: &akey,
		SEMap: map[SEvent]bool{
			SEventHSet: true,
			SEventHDel: true,
			SEventDel:  true,
		}})

	s, e := SubscribeDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
	}, skeys, func(s *DB,
		skey *SKey, key *Key,
		event SEvent) error {
		switch event {
		case SEventHSet:
			hSetCalled = true
		case SEventHDel:
			hDelCalled = true
		case SEventDel:
			delCalled = true
		default:
		}
		return nil
	})

	if (s == nil) || (e != nil) {
		t.Errorf("Subscribe() returns error e: %v", e)
		return
	}

	d.SetEntry(&ts, akey, avalue)
	d.DeleteEntryFields(&ts, akey, avalue)

	time.Sleep(5 * time.Second)

	if !hSetCalled || !hDelCalled || !delCalled {
		t.Errorf("Subscribe() callbacks missed: %v %v %v", hSetCalled,
			hDelCalled, delCalled)
		return
	}

	s.UnsubscribeDB()

	time.Sleep(2 * time.Second)

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}
