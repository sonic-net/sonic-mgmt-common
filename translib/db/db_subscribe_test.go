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
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

var SUB_TST string = "SUB_TST_" + strconv.FormatInt(int64(os.Getpid()), 10)
var sTs *TableSpec = &TableSpec{Name: SUB_TST}
var sK = SUB_TST + "|key1"
var sE0 Value = Value{Field: map[string]string{"f1": "v1", "f2": "v2"}}
var sEr Value = Value{Field: map[string]string{"f2": "v2"}}
var sE1 Value = Value{Field: map[string]string{"f1": "v1"}}

var recvHSet bool
var recvHDel bool
var recvDel bool
var recvCCOpened bool
var recvCCClosed bool
var recvCCNotif bool
var recvRCNotif bool
var unknownCCNotif bool

func newCCDB(dBNum DBNum) (*DB, error) {
	d, e := NewDB(Options{
		DBNo:               dBNum,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		DisableCVLCheck:    true,
		IsSession:          true,
	})
	return d, e
}

func cleanupSub(t *testing.T, d *DB, deleteDB bool) {
	if d == nil {
		return
	}

	if deleteDB {
		defer d.DeleteDB()
	}

	recvHSet = false
	recvHDel = false
	recvDel = false
	recvCCOpened = false
	recvCCClosed = false
	recvCCNotif = false
	recvRCNotif = false
	unknownCCNotif = false

	d.DeleteEntry(sTs, d.redis2key(sTs, sK))
}

func subHdlr(d *DB, skey *SKey, key *Key, event SEvent) error {
	fmt.Println("subHdlr: d: ", d, " skey: ", *skey, " key: ", *key,
		" event: ", event)
	switch event {
	case SEventHSet:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err == nil {
				if reflect.DeepEqual(entry, sE0) {
					recvHSet = true
				} else {
					fmt.Printf("subHdlr: recvHSet %v != %v\n", entry, sE0)
				}
			} else {
				fmt.Printf("subHdlr: recvHSet: GetEntry err = %v\n", err)
			}
		} else {
			fmt.Printf("subHdlr: recvHSet: key: %v != redisKey %v\n", key, sK)
		}
	case SEventHDel:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err == nil {
				if reflect.DeepEqual(entry, sE1) {
					recvHDel = true
				} else {
					fmt.Printf("subHdlr: recvHDel %v != %v\n", entry, sE1)
				}
			} else {
				fmt.Printf("subHdlr: recvHDel: GetEntry err = %v\n", err)
			}
		} else {
			fmt.Printf("subHdlr: recvHDel: key: %v != redisKey %v\n", key, sK)
		}
	case SEventDel:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err != nil {
				if _, ok := err.(tlerr.TranslibRedisClientEntryNotExist); ok {
					recvDel = true
				} else {
					fmt.Printf("subHdlr: recvHDel error mismatch %v\n", err)
				}
			} else {
				fmt.Printf("subHdlr: recvDel: GetEntry success = %v\n", entry)
			}
		} else {
			fmt.Printf("subHdlr: recvDel: key: %v != redisKey %v\n", key, sK)
		}
	}

	return nil
}

func TestSubscribeHFunc(t *testing.T) {

	// Create RC DB
	d, e := newDB(ConfigDB)
	if e != nil {
		t.Errorf("newDB() fails e: %v", e)
	}

	// Cleanup
	cleanupSub(t, d, false)

	// Register CleanUp
	t.Cleanup(func() { cleanupSub(t, d, true) })

	// Create SubscribeDB
	subKey := d.redis2key(sTs, sK)
	var sKeys []*SKey = make([]*SKey, 1)
	sKeys[0] = &(SKey{Ts: sTs, Key: &subKey,
		SEMap: map[SEvent]bool{
			SEventHSet: true,
			SEventHDel: true,
			SEventDel:  true,
		}})

	subD, e := SubscribeDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	}, sKeys, subHdlr)

	if e != nil {
		t.Errorf("SubscribeDB() returns error e: %v", e)
	}
	t.Cleanup(func() { subD.UnsubscribeDB() })

	// Write to RC DB
	if e = d.SetEntry(sTs, subKey, sE0); e != nil {
		t.Fatalf("d.SetEntry(%v,%v,%v) fails e: %v", sTs, subKey, sE0, e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the change was received by the handler func
	if !recvHSet {
		t.Errorf("HSet Notification Not Received")
	}

	// Delete Field from RC DB
	if e = d.DeleteEntryFields(sTs, subKey, sEr); e != nil {
		t.Errorf("d.DeleteEntryFields(%v,%v,%v) fails e: %v", sTs, subKey, sEr,
			e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the change was received by the handler func
	if !recvHDel {
		t.Errorf("HDel Notification Not Received")
	}

	// Delete Key from RC DB
	if e = d.DeleteEntry(sTs, subKey); e != nil {
		t.Errorf("d.DeleteEntry(%v,%v) fails e: %v", sTs, subKey, e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the change was received by the handler func
	if !recvDel {
		t.Errorf("Del Notification Not Received")
	}
}

func subHdlrSA(d *DB, sN SessNotif, cN string, skey *SKey, key *Key,
	event SEvent) {

	fmt.Println("subHdlrSA: d: ", d, " sN: ", sN, " cN: ", cN)
	if skey != nil {
		fmt.Println(" skey: ", *skey)
	}
	if key != nil {
		fmt.Println(" key: ", *key)
	}
	fmt.Println(" event: ", event)

	switch sN {
	case CandidateConfigOpened:
		recvCCOpened = true
	case CandidateConfigClosed:
		recvCCClosed = true
	case CandidateConfigNotif:
		recvCCNotif = true
	case RunningConfigNotif:
		recvRCNotif = true
	default:
		unknownCCNotif = true
	}

	switch event {
	case SEventHSet:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err == nil {
				if reflect.DeepEqual(entry, sE0) {
					recvHSet = true
				} else {
					fmt.Printf("subHdlr: recvHSet %v != %v\n", entry, sE0)
				}
			} else {
				fmt.Printf("subHdlr: recvHSet: GetEntry err = %v\n", err)
			}
		} else {
			fmt.Printf("subHdlr: recvHSet: key: %v != redisKey %v\n", key, sK)
		}
	case SEventHDel:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err == nil {
				if reflect.DeepEqual(entry, sE1) {
					recvHDel = true
				} else {
					fmt.Printf("subHdlr: recvHDel %v != %v\n", entry, sE1)
				}
			} else {
				fmt.Printf("subHdlr: recvHDel: GetEntry err = %v\n", err)
			}
		} else {
			fmt.Printf("subHdlr: recvHDel: key: %v != redisKey %v\n", key, sK)
		}
	case SEventDel:
		if d.key2redis(skey.Ts, *key) == sK {
			if entry, err := d.GetEntry(skey.Ts, *key); err != nil {
				if _, ok := err.(tlerr.TranslibRedisClientEntryNotExist); ok {
					recvDel = true
				} else {
					fmt.Printf("subHdlr: recvHDel error mismatch %v\n", err)
				}
			} else {
				fmt.Printf("subHdlr: recvDel: GetEntry success = %v\n", entry)
			}
		} else {
			fmt.Printf("subHdlr: recvDel: key: %v != redisKey %v\n", key, sK)
		}
	}

	return
}

func TestSubscribeHFuncSA(t *testing.T) {

	// Create CC DB
	d, e := newCCDB(ConfigDB)
	if e != nil {
		t.Errorf("newCCDB() fails e: %v", e)
	}

	// Cleanup
	cleanupSub(t, d, false)

	// Start the Session Transaction
	e = d.StartSessTx(nil, []*TableSpec{&(TableSpec{Name: "*"})})
	if e != nil {
		t.Errorf("StartSessTx() fails e: %v", e)
	}

	// Register CleanUp
	t.Cleanup(func() { cleanupSub(t, d, true) })

	// Create SubscribeDBSA
	subKey := d.redis2key(sTs, sK)
	var sKeys []*SKey = make([]*SKey, 1)
	sKeys[0] = &(SKey{Ts: sTs, Key: &subKey,
		SEMap: map[SEvent]bool{
			SEventHSet: true,
			SEventHDel: true,
			SEventDel:  true,
		}})

	subD, e := SubscribeDBSA(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	}, sKeys, subHdlrSA)

	if e != nil {
		t.Errorf("SubscribeDB() returns error e: %v", e)
	}
	t.Cleanup(func() { subD.UnsubscribeDB() })

	// Write to CC DB
	if e = d.SetEntry(sTs, subKey, sE0); e != nil {
		t.Fatalf("d.SetEntry(%v,%v,%v) fails e: %v", sTs, subKey, sE0, e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the CandidateConfigOpened was received by the handler func
	if !recvCCOpened {
		t.Errorf("CandidateConfig Notification Not Received")
	}
	// Reset the recvCCOpened
	recvCCOpened = false

	// Verify that the change was received by the handler func
	if !recvHSet {
		t.Errorf("HSet Notification Not Received")
	}

	// Verify that no other Notif was received by the handler func
	if recvRCNotif || unknownCCNotif {
		t.Errorf("Unknown Notification Received")
	}

	// Delete Field from CC DB
	if e = d.DeleteEntryFields(sTs, subKey, sEr); e != nil {
		t.Errorf("d.DeleteEntryFields(%v,%v,%v) fails e: %v", sTs, subKey, sEr,
			e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the CandidateConfigOpened was NOT received by the handler fn
	if recvCCOpened {
		t.Errorf("Unexpected CandidateConfigOpened Notification Received")
	}

	// Verify that no other Notif was received by the handler func
	if recvRCNotif || unknownCCNotif {
		t.Errorf("Unknown Notification Received")
	}

	// Verify that the CandidateConfigOpened was NOT received by the handler fn
	if recvCCOpened {
		t.Errorf("Unexpected CandidateConfigOpened Notification Received")
	}

	// Verify that the change was received by the handler func
	if !recvHDel {
		t.Errorf("HDel Notification Not Received")
	}

	// Delete Key from CC DB
	if e = d.DeleteEntry(sTs, subKey); e != nil {
		t.Errorf("d.DeleteEntry(%v,%v) fails e: %v", sTs, subKey, e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify that the CandidateConfigOpened was NOT received by the handler fn
	if recvCCOpened {
		t.Errorf("Unexpected CandidateConfigOpened Notification Received")
	}

	// Verify that the change was received by the handler func
	if !recvDel {
		t.Errorf("Del Notification Not Received")
	}

	// Verify that no other Notif was received by the handler func
	if recvRCNotif || unknownCCNotif {
		t.Errorf("Unknown Notification Received")
	}
}

func TestSubscribeNoCCNotif2HFuncSA(t *testing.T) {

	// Confirm that RC Notifs are not sent as CC Notifs to the HFuncSA

	// Create RC DB
	d, e := newDB(ConfigDB)
	if e != nil {
		t.Errorf("newDB() fails e: %v", e)
	}

	// Cleanup
	cleanupSub(t, d, false)

	// Start the Session Transaction
	e = d.StartTx(nil, []*TableSpec{&(TableSpec{Name: "*"})})
	if e != nil {
		t.Errorf("StartTx() fails e: %v", e)
	}

	// Register CleanUp
	t.Cleanup(func() { cleanupSub(t, d, true) })

	// Create SubscribeDBSA
	subKey := d.redis2key(sTs, sK)
	var sKeys []*SKey = make([]*SKey, 1)
	sKeys[0] = &(SKey{Ts: sTs, Key: &subKey,
		SEMap: map[SEvent]bool{
			SEventHSet: true,
			SEventHDel: true,
			SEventDel:  true,
		}})

	subD, e := SubscribeDBSA(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	}, sKeys, subHdlrSA)

	if e != nil {
		t.Errorf("SubscribeDB() returns error e: %v", e)
	}
	t.Cleanup(func() { subD.UnsubscribeDB() })

	// Write to RC DB
	if e = d.SetEntry(sTs, subKey, sE0); e != nil {
		t.Fatalf("d.SetEntry(%v,%v,%v) fails e: %v", sTs, subKey, sE0, e)
	}

	// Commit the Session Transaction
	e = d.CommitTx()
	if e != nil {
		t.Errorf("CommitTx() fails e: %v", e)
	}

	// Wait for the handler func to complete
	t.Log("Sleeping 2 ==============")
	time.Sleep(2 * time.Second)

	// Verify Only the RC Notif was received by the handler func
	if recvCCOpened {
		t.Errorf("CandidateConfig Notification Received: %v", recvCCOpened)
	}

	// Verify that the change was received by the handler func
	if !recvRCNotif || !recvHSet {
		t.Errorf("HSet Notification Not Received: %v %v", recvRCNotif, recvHSet)
	}

	// Verify that no other Notif was received by the handler func
	if recvCCNotif || unknownCCNotif {
		t.Errorf("Unknown Notification Received")
	}
}
