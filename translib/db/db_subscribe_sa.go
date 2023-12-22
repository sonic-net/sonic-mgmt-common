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

// DB Layer Session Aware (SA) Subscription for (Config) Session DB.
//

import (
	"sync"

	"github.com/golang/glog"
)

const (
	InitTable2sDBSize int = 20
	InitSDBArraySize  int = 5
)

// subscribeDB struct to store (SA aware) SubscribeDBSA registration info
type subscribeDB struct {
	d       *DB
	handler interface{}
	skeys   []*SKey

	openSent bool // Has the SessionOpened Notification been sent?
}

type subscriberInfo struct {
	sDBs       []*subscribeDB
	table2sDBs map[string][]*subscribeDB
}

var sInfo subscriberInfo
var sInfoMutex sync.Mutex

// registerSubscribeDB registers a subscription for Session DB notifications
// Only info. on SA aware subscriptions are registered for now.
func (d *DB) registerSubscribeDB(isSA bool, skeys []*SKey, handler interface{}) {

	glog.V(3).Infof("registerSubscribeDB: isSA %t skeys %v", isSA, skeys)

	if !isSA {
		return
	}

	sInfoMutex.Lock()
	defer sInfoMutex.Unlock()

	sDB := &subscribeDB{d: d, skeys: skeys, handler: handler}
	sInfo.sDBs = append(sInfo.sDBs, sDB)

	if sInfo.table2sDBs == nil {
		sInfo.table2sDBs = make(map[string][]*subscribeDB, InitTable2sDBSize)
	}
	for _, skey := range skeys {
		ts := *skey.Ts
		if sInfo.table2sDBs[ts.Name] == nil {
			sInfo.table2sDBs[ts.Name] = make([]*subscribeDB, 0, InitSDBArraySize)
		}

		// For sending notifications we iterate over all the skeys in an
		// sDB, so skip adding sDB if is already on the table2sDBs list
		found := false
		for _, iSDB := range sInfo.table2sDBs[ts.Name] {
			if iSDB == sDB {
				found = true
			}
		}
		if !found {
			sInfo.table2sDBs[ts.Name] = append(sInfo.table2sDBs[ts.Name], sDB)
		}
	}

	// Wait for an actual Session DB change before sending SessionOpened, to
	// get the Session *DB. The Session DB Config DB lock should have been
	// held when the Session DB is being modified
	// Note: The assumption here is that long-lived Subscriptions are
	// going to opened before any Config Session is opened. Thus any
	// short-lived Subscriptions are not going to be interested in Config
	// Sessions.
}

// unRegisterSubscribeDB unregisters a subscription from Session DB notifs.
func (d *DB) unRegisterSubscribeDB() {

	glog.V(3).Info("unRegisterSubscribeDB:")

	sInfoMutex.Lock()
	defer sInfoMutex.Unlock()

	found := false
	for i, sDB := range sInfo.sDBs {
		if sDB.d == d {
			sInfo.sDBs = append(sInfo.sDBs[:i], sInfo.sDBs[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		glog.Info("unRegisterSubscribeDB: SubscribeDB likely not SA")
		return
	}

	for tsName, sDBs := range sInfo.table2sDBs {
		for i, sDB := range sDBs {
			if sDB.d == d {
				sInfo.table2sDBs[tsName] = append(sInfo.table2sDBs[tsName][:i],
					sInfo.table2sDBs[tsName][i+1:]...)
				break
			}
		}
	}
}

func (d *DB) sendSessionOpened() {
	sInfoMutex.Lock()
	defer sInfoMutex.Unlock()

	oIsWriteDisabled := d.Opts.IsWriteDisabled
	d.Opts.IsWriteDisabled = true
	defer func() { d.Opts.IsWriteDisabled = oIsWriteDisabled }()

	for _, sDB := range sInfo.sDBs {
		if !sDB.openSent {
			if hFuncSA, ok := sDB.handler.(HFuncSA); ok {
				callHFuncSA(hFuncSA, d, CandidateConfigOpened, "", nil, nil,
					SEventNone)
				sDB.openSent = true
			}
		}
	}
}

func (d *DB) sendSessionClosed() {
	sInfoMutex.Lock()
	defer sInfoMutex.Unlock()

	oIsWriteDisabled := d.Opts.IsWriteDisabled
	d.Opts.IsWriteDisabled = true
	defer func() { d.Opts.IsWriteDisabled = oIsWriteDisabled }()

	for _, sDB := range sInfo.sDBs {
		if sDB.openSent {
			if hFuncSA, ok := sDB.handler.(HFuncSA); ok {
				callHFuncSA(hFuncSA, d, CandidateConfigClosed, "", nil, nil,
					SEventNone)
				sDB.openSent = false
			}
		}
	}
}

func (d *DB) sendSessionNotification(ts *TableSpec, key *Key, op1, op2 _txOp) {
	sInfoMutex.Lock()
	defer sInfoMutex.Unlock()

	oIsWriteDisabled := d.Opts.IsWriteDisabled
	d.Opts.IsWriteDisabled = true
	defer func() { d.Opts.IsWriteDisabled = oIsWriteDisabled }()

	// First Send the open notification, if it wasn't sent earlier
	for _, sDB := range sInfo.sDBs {

		if !sDB.openSent {
			if hFuncSA, ok := sDB.handler.(HFuncSA); ok {
				callHFuncSA(hFuncSA, d, CandidateConfigOpened, "", nil, nil,
					SEventNone)
				sDB.openSent = true
			}
		}
	}

	// Send the actual notification(s)
	for _, sDB := range sInfo.table2sDBs[ts.Name] {
		for _, skey := range sDB.skeys {
			if skey.Key.IsAllKeyPattern() || key.Matches(*skey.Key) {
				if hFuncSA, ok := sDB.handler.(HFuncSA); ok {
					if sEvent := d.txOp2sEvent(op1); sEvent != SEventNone {
						callHFuncSA(hFuncSA, d, CandidateConfigNotif, "",
							skey, key, sEvent)
					}
					if sEvent := d.txOp2sEvent(op2); sEvent != SEventNone {
						callHFuncSA(hFuncSA, d, CandidateConfigNotif, "",
							skey, key, sEvent)
					}
				}
			}
		}
	}
}

// callHFuncSA invokes hFuncSA safely (i.e. catches and reports any panic),
// in a synchronous manner (i.e. wait for the completion of hFuncSA before
// returning).
func callHFuncSA(hFuncSA HFuncSA, d *DB, sN SessNotif, cN string, sKey *SKey, key *Key, sE SEvent) {
	done := make(chan int)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Errorf("callHFuncSA: panic() key: %v, err: %v", key, err)
			}
			done <- 1
		}()
		hFuncSA(d, sN, cN, sKey, key, sE)
	}()
	<-done
}
