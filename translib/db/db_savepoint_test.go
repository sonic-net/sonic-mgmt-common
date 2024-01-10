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
	"reflect"
	"strconv"

	"testing"
)

type testOp int

const (
	testOpNone testOp = iota // No Op
	testOpCreate
	testOpSet
	testOpMod
	testOpDelete
)

type spTCop struct {
	op  testOp
	tbl Table
}

type spTC struct {
	tid       string
	origTable []Table
	modOps    []spTCop
}

var spTests []spTC

var SP_PF string = "DBSP_TST_" + strconv.FormatInt(int64(os.Getpid()), 10)

func cleanupSP(t *testing.T, d *DB, tc *spTC, deleteDB bool) {
	if d == nil {
		return
	}

	t.Logf("cleanupSP: Test Case %s:\n", tc.tid)

	if deleteDB {
		defer d.DeleteDB()
	}

	for _, ta := range tc.origTable {
		d.DeleteTable(ta.ts)
	}

	for _, op := range tc.modOps {
		d.DeleteTable(op.tbl.ts)
	}
}

func runSP(t *testing.T, tc *spTC) {

	t.Logf("runSP: Test Case %s: Begin\n", tc.tid)

	d, e := newDB(ConfigDB)
	if e != nil {
		t.Errorf("newDB() fails e: %v", e)
	}

	// Cleanup before starting
	cleanupSP(t, d, tc, false)

	// Register CleanUp Function
	t.Cleanup(func() { cleanupSP(t, d, tc, true) })

	// Add origTable
	for _, tbl := range tc.origTable {
		for rk, v := range tbl.entry {
			if e = d.SetEntry(tbl.ts, d.redis2key(tbl.ts, rk), v); e != nil {
				t.Errorf("d.SetEntry(%v,%v,%v) fails e: %v", tbl.ts, rk, v, e)
			}
		}
	}

	ccd, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		IsSession:          true,
		DisableCVLCheck:    true,
	})

	if e != nil {
		t.Fatalf("Session NewDB() fails e: %v", e)
	}

	t.Cleanup(func() { ccd.DeleteDB() })

	if e = ccd.StartSessTx(nil, []*TableSpec{&(TableSpec{Name: "*"})}); e != nil {

		t.Errorf("Session StartTx() fails e: %v", e)
	}

	t.Cleanup(func() { ccd.AbortSessTx() })

	if e = ccd.DeclareSP(); e != nil {
		t.Errorf("DeclareSP() fails e: %v", e)
	}

	// Perform Ops
	for _, mOps := range tc.modOps {
		switch mOps.op {
		case testOpDelete:
			for rk, v := range mOps.tbl.entry {
				if len(v.Field) == 0 {
					if e = ccd.DeleteEntry(mOps.tbl.ts, ccd.redis2key(
						mOps.tbl.ts, rk)); e != nil {

						t.Errorf("ccd.DeleteEntry(%v,%v) fails e: %v",
							mOps.tbl.ts, rk, e)
					}
				} else {
					if e = ccd.DeleteEntryFields(mOps.tbl.ts, ccd.redis2key(
						mOps.tbl.ts, rk), v); e != nil {

						t.Errorf("ccd.DeleteEntryFields(%v,%v,%v) fails e: %v",
							mOps.tbl.ts, rk, v, e)
					}
				}
			}

		case testOpCreate:
			for rk, v := range mOps.tbl.entry {
				if e = ccd.CreateEntry(mOps.tbl.ts, ccd.redis2key(mOps.tbl.ts, rk),
					v); e != nil {

					t.Errorf("ccd.CreateEntry(%v,%v,%v) fails e: %v", mOps.tbl.ts,
						rk, v, e)
				}
			}

		case testOpSet:
			for rk, v := range mOps.tbl.entry {
				if e = ccd.SetEntry(mOps.tbl.ts, ccd.redis2key(mOps.tbl.ts, rk),
					v); e != nil {

					t.Errorf("ccd.SetEntry(%v,%v,%v) fails e: %v", mOps.tbl.ts,
						rk, v, e)
				}
			}

		case testOpMod:
			for rk, v := range mOps.tbl.entry {
				if e = ccd.ModEntry(mOps.tbl.ts, ccd.redis2key(mOps.tbl.ts, rk),
					v); e != nil {

					t.Errorf("ccd.ModEntry(%v,%v,%v) fails e: %v", mOps.tbl.ts,
						rk, v, e)
				}
			}
		}
	}

	if e = ccd.Rollback2SP(); e != nil {
		t.Errorf("Rollback2SP() fails e: %v", e)
	}

	// Confirm the ccd only contains origTable.
	for _, tbl := range tc.origTable {
		ccdTbl, e := ccd.GetTable(tbl.ts)
		if e != nil {
			t.Errorf("ccd.GetTable() fails e: %v", e)
		}

		// The ordering of the arrays in patterns map values matters
		delete(tbl.patterns, db.key2redis(tbl.ts, Key{Comp: []string{"*"}}))
		delete(ccdTbl.patterns, db.key2redis(tbl.ts, Key{Comp: []string{"*"}}))

		tbl.db = nil
		ccdTbl.db = nil

		if !reflect.DeepEqual(tbl.entry, ccdTbl.entry) {
			t.Log("\ntbl: \n", tbl.entry)
			t.Log("\nccdTbl: \n", ccdTbl.entry)
			t.Errorf("tbl != ccdTbl")
		}
	}

	t.Logf("runSP: Test Case %s: End\n", tc.tid)
}

// TestDeclareSP tests DeclareSP(), and ReleaseSP()
func TestSPDeclareSP(t *testing.T) {

	ccd, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		IsSession:          true,
	})

	if e != nil {
		t.Fatalf("Session NewDB() fails e: %v", e)
	}

	t.Cleanup(func() {
		t.Log("TestSPDeclareSP: Cleanup\n")
		ccd.DeleteDB()
	})

	if e = ccd.DeclareSP(); e != nil {
		t.Errorf("DeclareSP() fails e: %v", e)
	}

	if e = ccd.Rollback2SP(); e != nil {
		t.Errorf("Rollback2SP() fails e: %v", e)
	}

}

// TestRollback2SP
func TestSPRollback2SP(t *testing.T) {
	for _, tc := range spTests {
		runSP(t, &tc)
	}
}

func init() {
	spTests = append(spTests, basicTests...)
}

var basicTests = []spTC{
	{
		tid: "Basic Test Case",
		origTable: []Table{
			Table{
				ts: &TableSpec{Name: SP_PF + "RADIUS"},
				entry: map[string]Value{
					SP_PF + "RADIUS|global_key": Value{
						Field: map[string]string{
							"auth_type": "pap",
						},
					},
				},
				complete: true,
			},
		},
		modOps: []spTCop{
			spTCop{
				op: testOpMod,
				tbl: Table{
					ts: &TableSpec{Name: SP_PF + "RADIUS"},
					entry: map[string]Value{
						SP_PF + "RADIUS|global_key": Value{
							Field: map[string]string{
								"auth_type": "pap",
							},
						},
					},
					complete: true,
				},
			},
		},
	},
}
