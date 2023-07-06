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
)

var GM_PF string = "DBGM_TST_" + strconv.FormatInt(int64(os.Getpid()), 10)
var gmTs *TableSpec = &TableSpec{Name: GM_PF + "RADIUS"}
var gmRK = GM_PF + "RADIUS|global_key"
var gmEntry Value = Value{Field: map[string]string{"auth_type": "pap"}}
var cdbUpdated string = "CONFIG_DB_UPDATED"

func cleanupGM(t *testing.T, d *DB, deleteDB bool) {
	if d == nil {
		return
	}

	if deleteDB {
		defer d.DeleteDB()
	}

	d.DeleteEntry(gmTs, d.redis2key(gmTs, gmRK))
}

func TestGetMeta(t *testing.T) {

	d, e := newDB(ConfigDB)
	if e != nil {
		t.Errorf("newDB() fails e: %v", e)
	}

	gmKey := d.redis2key(gmTs, gmRK)

	// Cleanup before starting
	cleanupGM(t, d, false)

	// Register CleanUp Function
	t.Cleanup(func() { cleanupGM(t, d, true) })

	e = d.StartTx(nil, nil)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
	}

	if e = d.SetEntry(gmTs, gmKey, gmEntry); e != nil {
		t.Fatalf("d.SetEntry(%v,%v,%v) fails e: %v", gmTs, gmKey, gmEntry, e)
	}

	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
	}

	if s, e := d.Get(cdbUpdated); e != nil {
		t.Errorf("d.Get(%s) fails e: %v", cdbUpdated, e)
	} else if len(s) <= 1 {
		t.Errorf("d.Get(%s) returns: %s", cdbUpdated, s)
	}

	if _, e := d.Get("RANDOM_KEY"); e == nil {
		t.Errorf("d.Get(%s) succeeds!", "RANDOM_KEY")
	}

	if _, e := d.Get("CONFIG_DB_TABLE|global"); e == nil {
		t.Errorf("d.Get(%s) succeeds!", "CONFIG_DB_TABLE|global")
	}
}
