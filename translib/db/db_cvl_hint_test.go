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

var hKey = "TESTKEY"
var hValue = map[string]string{"a": "1", "b": "2"}

func newEnableCVLDB(dBNum DBNum) (*DB, error) {
	d, e := NewDB(Options{
		DBNo:               dBNum,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	return d, e
}

func cleanupCH(t *testing.T, d *DB) {
	if d == nil {
		return
	}

	d.DeleteDB()
}

func TestCVLHint(t *testing.T) {
	d, e := newEnableCVLDB(ConfigDB)
	if e != nil {
		t.Fatalf("newDB() fails e: %v", e)
	}

	d.clearCVLHint("")

	// Register CleanUp Function
	t.Cleanup(func() { cleanupCH(t, d) })

	if e := d.StoreCVLHint(hKey, hValue); e != nil {
		t.Errorf("d.StoreCVLHint(%v, %v) fails e: %v", hKey, hValue, e)
	}

	d.clearCVLHint(hKey)

	// Negative Test
	if e := d.StoreCVLHint("", hValue); e == nil {
		t.Errorf("d.StoreCVLHint(%v, %v) succeeds", hValue, hValue)
	}
}
