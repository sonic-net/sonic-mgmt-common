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
	// "fmt"
	// "errors"
	// "flag"
	// "github.com/golang/glog"
	// "time"
	// "github.com/Azure/sonic-mgmt-common/translib/tlerr"
	// "os/exec"
	"os"
	"strconv"
	"testing"
	// "reflect"
)

func testSCAddDelKeys(t *testing.T, d *DB, ts *TableSpec, prefix string, count int, delete bool) {
	var e error
	var op string
	for i := 0; i < count; i++ {
		akey := Key{Comp: []string{prefix + strconv.Itoa(i)}}
		avalue := Value{map[string]string{"k1": "v1", "k2": "v2"}}
		if delete {
			op = "delete"
			e = d.DeleteEntry(ts, akey)
		} else {
			op = "create"
			e = d.SetEntry(ts, akey, avalue)
		}
		if e != nil {
			t.Fatalf("%v fails e = %v", op, e)
		}
	}
}

func testSCGetNextKeys(d *DB, ts *TableSpec, pattern string, expected int) func(*testing.T) {
	return func(t *testing.T) {

		patKey := Key{Comp: []string{pattern}}
		scOpts := ScanCursorOpts{CountHint: 10}

		sc, e := d.NewScanCursor(ts, patKey, &scOpts)

		if (sc == nil) || (e != nil) {
			t.Fatalf("testSCGetNextKeys() fails e = %v", e)
			return
		}

		var keys []Key
		var count int
		for scanComplete := false; !scanComplete; {
			keys, scanComplete, e = sc.GetNextKeys(&scOpts)
			if e != nil {
				t.Fatalf("sc.GetNextKeys() fails e = %v", e)
				return
			}
			count += len(keys)
		}

		if count != expected {
			t.Fatalf("testSCGetNextKeys() count: %v != expected: %v", count, expected)
		}

		if e = sc.DeleteScanCursor(); e != nil {
			t.Fatalf("DeleteScanCursor() fails e = %v", e)
		}
	}

}

func TestNewScanCursor(t *testing.T) {

	var pid int = os.Getpid()

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    false,
	})

	if (d == nil) || (e != nil) {
		t.Fatalf("NewDB() fails e = %v", e)
		return
	}
	defer d.DeleteDB()

	ts := TableSpec{Name: "TESTSC_" + strconv.FormatInt(int64(pid), 10)}

	prefix := "SCKEY_"
	testSCAddDelKeys(t, d, &ts, prefix, 100, false)
	defer testSCAddDelKeys(t, d, &ts, prefix, 100, true)
	d.Opts.IsWriteDisabled = true //disabling the write for the scan cursor to work
	t.Run("pattern=*", testSCGetNextKeys(d, &ts, "*", 100))
	t.Run("pattern=SCKEY_0", testSCGetNextKeys(d, &ts, "SCKEY_0", 1))
	t.Run("pattern=SCKEY_1*", testSCGetNextKeys(d, &ts, "SCKEY_1*", 11))
	t.Run("pattern=NOTALIKELYKEY", testSCGetNextKeys(d, &ts, "NOTALIKELYKEY", 0))
	d.Opts.IsWriteDisabled = false
}
