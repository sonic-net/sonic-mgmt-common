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
	"os"
	"strconv"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

var testSTok string = "1001-1"
var fTs *TableSpec = &TableSpec{Name: lockTable}
var fKey Key = Key{Comp: []string{lockKey}}

var stateDB *DB

func setupKey(t *testing.T, ts *TableSpec, key Key, val Value) {
	var err error
	if stateDB, err = NewDB(Options{DBNo: StateDB}); err != nil {
		t.Errorf("setupKey: NewDB(StateDB) fails: %v", err)
	}
	stateDB.DeleteEntry(ts, key)
	t.Cleanup(func() { stateDB.DeleteEntry(ts, key); cdbLock = nil })

	if err = stateDB.ModEntry(ts, key, val); err != nil {
		t.Errorf("setupKey: ModEntry() fails: %v", err)
	}
}

// TestLock: Somebody else's lock should be honored.
func TestLock(t *testing.T) {
	t.Run("writeLock", testLock(execName+":"+noSessionToken, tlerr.DBLockGeneric))
	t.Run("sessionLock", testLock(execName+":123456789-100", tlerr.DBLockConfigSession))
	t.Run("emptyToken", testLock(execName+":", tlerr.DBLockGeneric))
	t.Run("badToken", testLock(execName, tlerr.DBLockGeneric))
	t.Run("nilToken", testLock("", tlerr.DBLockGeneric))
}

// testLock simulates a lock with given lockToken and verifies whether tryLock()
// returns a TranslibDBLock with given DBLockType code
func testLock(lockToken string, expError tlerr.DBLockType) func(*testing.T) {
	return func(t *testing.T) {
		fVal := Value{Field: map[string]string{configDBLock: lockToken}}
		setupKey(t, fTs, fKey, fVal)

		err := (&LockStruct{Name: configDBLock, Id: testSTok,
			lockStruct: lockStruct{comm: execName}}).tryLock()

		if e, ok := err.(tlerr.TranslibDBLock); !ok || e.Type != expError {
			exp := tlerr.TranslibDBLock{Type: expError}
			t.Errorf("Expecting %#v: Received %#v", exp, err)
		}
	}
}

// TestLockUnlock: Unlocking our own lock.
func TestLockUnlock(t *testing.T) {

	var err error

	// Clean it up.
	if err = stateDB.DeleteEntry(fTs, fKey); err != nil {
		t.Errorf("DeleteEntry: Expecting nil: Received %v", err)
	}
	t.Cleanup(func() { stateDB.DeleteEntry(fTs, fKey); cdbLock = nil })

	// Lock it
	ls := &LockStruct{Name: configDBLock, Id: testSTok,
		lockStruct: lockStruct{comm: execName}}
	err = ls.tryLock()
	if err != nil {
		t.Errorf("tryLock: Expecting nil: Received %v", err)
	}

	// Lock it Again! -- Should fail with Not Supported
	err = ls.tryLock()
	if _, ok := err.(tlerr.TranslibDBNotSupported); !ok {
		t.Errorf("Expecting %v: Received %v", tlerr.TranslibDBNotSupported{},
			err)
	}

	// Unlock it.
	err = ls.unlock()
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}

	// Unlock it Again! -- Should fail with Not Supported
	err = ls.unlock()
	if _, ok := err.(tlerr.TranslibDBNotSupported); !ok {
		t.Errorf("unlock: Expecting %v: Received %v",
			tlerr.TranslibDBNotSupported{}, err)
	}

	// Lock it Yet Again! -- Should succeed
	err = ls.tryLock()
	if err != nil {
		t.Errorf("tryLock 2: Expecting nil: Received %v", err)
	}

	// Let's be nice, and clean it up.
	// Unlock it.
	err = ls.unlock()
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}
}

// TestLockUnlockNonExisting: Unlock a non-existing lock.
func TestLockUnlockNonExisting(t *testing.T) {

	var err error

	// Clean it up.
	if err = stateDB.DeleteEntry(fTs, fKey); err != nil {
		t.Errorf("DeleteEntry: Expecting nil: Received %v", err)
	}
	t.Cleanup(func() { stateDB.DeleteEntry(fTs, fKey); cdbLock = nil })

	// Unlock it. -- Should Fail.
	ls := &LockStruct{Name: configDBLock, Id: testSTok,
		lockStruct: lockStruct{comm: execName}}
	err = ls.unlock()
	if _, ok := err.(tlerr.TranslibDBNotSupported); !ok {
		t.Errorf("Expecting %v: Received %v", tlerr.TranslibDBNotSupported{},
			err)
	}
}

// TestLockNotOurLockUnlock: Try unlocking somebody else's lock
func TestLockNotOurLockUnlock(t *testing.T) {

	var err error

	fVal := Value{Field: map[string]string{
		configDBLock: execName + ":" + testSTok + "0"}}
	//                                            ^^^ Somebody else's lock
	setupKey(t, fTs, fKey, fVal)

	// Unlock it. -- Should Fail.
	ls := &LockStruct{Name: configDBLock, Id: testSTok,
		lockStruct: lockStruct{comm: execName}}
	err = ls.unlock()
	if _, ok := err.(tlerr.TranslibDBNotSupported); !ok {
		t.Errorf("Expecting %v: Received %v", tlerr.TranslibDBNotSupported{},
			err)
	}

	// Unlock it faking locked field in LockStruct{} -- Fail with different err.
	ls.locked = true
	err = ls.unlock()
	if _, ok := err.(tlerr.TranslibDBLock); !ok {
		t.Errorf("Expecting %v: Received %v", tlerr.TranslibDBLock{},
			err)
	}
}

// TestLockClearLock: Try clearing somebody else's lock
func TestLockClearLock(t *testing.T) {

	var err error

	fVal := Value{Field: map[string]string{
		configDBLock: execName + ":" + testSTok + "0"}}
	//                                            ^^^ Somebody else's lock
	setupKey(t, fTs, fKey, fVal)

	// Clear it. -- Should Work.
	ls := &LockStruct{Name: configDBLock, Id: "*",
		lockStruct: lockStruct{comm: execName, locked: true}}
	err = ls.unlock()
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}

	// Lock it -- Should succeed
	ls.Id = testSTok
	err = ls.tryLock()
	if err != nil {
		t.Errorf("tryLock: Expecting nil: Received %v", err)
	}

	// Let's be nice, and clean it up.
	// Unlock it.
	err = ls.unlock()
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}
}

// TestLockConfigDB: Lock/Unlock ConfigDB
func TestLockConfigDB(t *testing.T) {

	var err error

	// Clean it up.
	if err = stateDB.DeleteEntry(fTs, fKey); err != nil {
		t.Errorf("DeleteEntry: Expecting nil: Received %v", err)
	}
	t.Cleanup(func() { stateDB.DeleteEntry(fTs, fKey); cdbLock = nil })

	// Lock it
	err = ConfigDBTryLock(testSTok)
	if err != nil {
		t.Errorf("ConfigDBTryLock: Expecting nil: Received %v", err)
	}

	// Lock it Again! -- Should fail with Lock Error
	err = ConfigDBTryLock(testSTok)
	if _, ok := err.(tlerr.TranslibDBLock); !ok {
		t.Errorf("Expecting %v: Received %v", tlerr.TranslibDBLock{},
			err)
	}

	// Unlock it.
	err = ConfigDBUnlock(testSTok)
	if err != nil {
		t.Errorf("ConfigDBUnlock: Expecting nil: Received %v", err)
	}

	// Unlock it Again! -- Should fail with Lock Error
	err = ConfigDBUnlock(testSTok)
	if _, ok := err.(tlerr.TranslibDBLock); !ok {
		t.Errorf("ConfigDBUnlock: Expecting %v: Received %v",
			tlerr.TranslibDBLock{}, err)
	}

	// Lock it Yet Again! -- Should succeed
	err = ConfigDBTryLock(testSTok)
	if err != nil {
		t.Errorf("ConfigDBTryLock 2: Expecting nil: Received %v", err)
	}

	// Let's be nice, and clean it up.
	// Unlock it.
	err = ConfigDBUnlock(testSTok)
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}
}

// ConfigDBClearLock
// TestLockConfigDBClearLock: Try clearing somebody else's lock
func TestLockConfigDBClearLock(t *testing.T) {

	var err error

	fVal := Value{Field: map[string]string{
		configDBLock: execName + ":" + testSTok + "0"}}
	//                                            ^^^ Somebody else's lock
	setupKey(t, fTs, fKey, fVal)

	// Clear it. -- Should Work.
	err = ConfigDBClearLock()
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}

	// Lock it
	err = ConfigDBTryLock(testSTok)
	if err != nil {
		t.Errorf("ConfigDBTryLock: Expecting nil: Received %v", err)
	}

	// Let's be nice, and clean it up.
	// Unlock it.
	err = ConfigDBUnlock(testSTok)
	if err != nil {
		t.Errorf("unlock: Expecting nil: Received %v", err)
	}
}

// TestLockConfigDBDeleteEntryFields: Verify ConfigDBLocked error when
// calling DeleteEntryFields() on a locked DB
func TestLockConfigDBDeleteEntryFields(t *testing.T) {
	var err error

	// Clean it up.
	if err = stateDB.DeleteEntry(fTs, fKey); err != nil {
		t.Fatalf("DeleteEntry: Expecting nil: Received %v", err)
	}
	t.Cleanup(func() { stateDB.DeleteEntry(fTs, fKey); cdbLock = nil })

	// Lock it
	err = ConfigDBTryLock(testSTok)
	if err != nil {
		t.Fatalf("ConfigDBTryLock: Expecting nil: Received %v", err)
	}

	t.Cleanup(func() {
		err = ConfigDBClearLock()
		if err != nil {
			t.Errorf("Clearing: Expecting nil: Received %v", err)
		}
	})

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		ConfigDBLazyLock:   true,
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Fatalf("NewDB() fails e = %v", e)
	}

	t.Cleanup(func() {
		if e = d.DeleteDB(); e != nil {
			t.Errorf("DeleteDB() fails e = %v", e)
		}
	})

	e = d.StartTx(nil, nil)

	if e != nil {
		t.Fatalf("StartTx() fails e = %v", e)
	}

	t.Cleanup(func() {
		if e = d.AbortTx(); e != nil {
			t.Errorf("DeleteDB() fails e = %v", e)
		}
	})

	ts := TableSpec{Name: "TEST_" + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}
	avalue := Value{map[string]string{"ports@": "Ethernet0", "type": "MIRROR"}}

	e = d.DeleteEntryFields(&ts, akey, avalue)
	if _, ok := e.(tlerr.TranslibDBLock); !ok {
		t.Errorf("ConfigDBUnlock: Expecting %v: Received %v",
			tlerr.TranslibDBLock{}, err)
	}
}
