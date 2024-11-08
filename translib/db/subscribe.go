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

/*
Package db implements a wrapper over the go-redis/redis.
*/
package db

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

// SKey is (TableSpec, Key, []SEvent) 3-tuples to be watched in a Transaction.
type SKey struct {
	Ts     *TableSpec
	Key    *Key
	SEMap  map[SEvent]bool // nil map indicates subscribe to all
	Opaque interface{}     // opaque data set by the caller
}

type SEvent int

const (
	SEventNone  SEvent = iota // No Op
	SEventHSet                // HSET, HMSET, and its variants
	SEventHDel                // HDEL, also SEventDel generated, if HASH is becomes empty
	SEventDel                 // DEL, & also if key gets deleted (empty HASH, expire,..)
	SEventOther               // Some other command not covered above.

	// The below two are always sent regardless of SEMap.
	SEventClose // Close requested due to Unsubscribe() called.
	SEventErr   // Error condition. Call Unsubscribe, after return.
)

var redisPayload2sEventMap map[string]SEvent = map[string]SEvent{
	"":     SEventNone,
	"hset": SEventHSet,
	"hdel": SEventHDel,
	"del":  SEventDel,
}

var txOp2sEventMap map[_txOp]SEvent = map[_txOp]SEvent{
	txOpNone:  SEventNone,
	txOpHMSet: SEventHSet,
	txOpHDel:  SEventHDel,
	txOpDel:   SEventDel,
}

type SessNotif uint8

const (
	RunningConfigNotif    SessNotif = iota // 0
	CandidateConfigNotif                   // 1
	CandidateConfigOpened                  // 2
	CandidateConfigClosed                  // 3
)

var sN2StringMap map[SessNotif]string = map[SessNotif]string{
	RunningConfigNotif:    "RunningConfigNotif",
	CandidateConfigNotif:  "CandidateConfigNotif",
	CandidateConfigOpened: "CandidateConfigOpened",
	CandidateConfigClosed: "CandidateConfigClosed",
}

func (sN SessNotif) String() string {
	return sN2StringMap[sN]
}

func init() {
	// Optimization: Start the goroutine that is scanning the SubscribeDB
	// channels. Instead of one goroutine per Subscribe.
}

// HFunc gives the name of the table, and other per-table customizations.
type HFunc func(*DB, *SKey, *Key, SEvent) error

// SubscribeDB is the factory method to create a subscription to the DB.
// The returned instance can only be used for Subscription.
func SubscribeDB(opt Options, skeys []*SKey, handler HFunc) (*DB, error) {
	return iSubscribeDB(opt, skeys, handler)
}

// HFuncSA is Session Aware HFunc
type HFuncSA func(*DB, SessNotif, string, *SKey, *Key, SEvent)

// SubscribeDBSA is Session Aware factory method to create a subscription to DB
// The returned instance can only be used for Subscription.
func SubscribeDBSA(opt Options, skeys []*SKey, handler HFuncSA) (*DB, error) {
	return iSubscribeDB(opt, skeys, handler)
}

func iSubscribeDB(opt Options, skeys []*SKey, handler interface{}) (*DB, error) {
	var isSA bool
	var hFunc HFunc
	var hFuncSA HFuncSA
	switch handler := handler.(type) {
	case HFunc:
		hFunc = handler
	case HFuncSA:
		isSA = true
		hFuncSA = handler
	default:
		glog.Errorf("SubscribeDB: Invalid handler type")
		return nil, tlerr.TranslibDBNotSupported{}
	}

	if glog.V(3) {
		glog.Info("SubscribeDB: Begin: opt: ", opt,
			" skeys: ", skeys, " handler: ", handler)
	}

	patterns := make([]string, 0, len(skeys))
	patMap := make(map[string]([]int), len(skeys))
	var s string

	if !opt.IsWriteDisabled {
		glog.Info("SubscribeDB: Setting IsWriteDisabled")
		opt.IsWriteDisabled = true
	}

	opt.IsSubscribeDB = true

	// NewDB
	d, e := NewDB(opt)

	if d.client == nil {
		goto SubscribeDBExit
	}

	// Make sure that the DB is configured for key space notifications
	// Optimize with LUA scripts to atomically add "Kgshxe".
	s, e = d.client.ConfigSet("notify-keyspace-events", "AKE").Result()

	if e != nil {
		glog.Error("SubscribeDB: ConfigSet(): e: ", e, " s: ", s)
		goto SubscribeDBExit
	}

	for i := 0; i < len(skeys); i++ {
		pattern := d.key2redisChannel(skeys[i].Ts, *(skeys[i].Key))
		if _, present := patMap[pattern]; !present {
			patMap[pattern] = make([]int, 0, 5)
			patterns = append(patterns, pattern)
		}
		patMap[pattern] = append(patMap[pattern], i)

	}

	glog.Info("SubscribeDB: patterns: ", patterns)

	d.sPubSub = d.client.PSubscribe(patterns[:]...)

	if d.sPubSub == nil {
		glog.Error("SubscribeDB: PSubscribe() nil: pats: ", patterns)
		e = tlerr.TranslibDBSubscribeFail{}
		goto SubscribeDBExit
	}

	d.sOnCCacheDB = d.Opts.SDB

	// Wait for confirmation, of channel creation
	_, e = d.sPubSub.Receive()

	if e != nil {
		glog.Error("SubscribeDB: Receive() fails: e: ", e)
		e = tlerr.TranslibDBSubscribeFail{}
		goto SubscribeDBExit
	}

	// Register
	d.registerSubscribeDB(isSA, skeys, handler)

	// Start a goroutine to read messages and call handler.
	go func() {
		for msg := range d.sPubSub.Channel() {
			if glog.V(4) {
				glog.Info("SubscribeDB: msg: ", msg)
			}

			// Should this be a goroutine, in case each notification CB
			// takes a long time to run ?
			for _, skeyIndex := range patMap[msg.Pattern] {
				skey := skeys[skeyIndex]
				key := d.redisChannel2key(skey.Ts, msg.Channel)
				sevent := d.redisPayload2sEvent(msg.Payload)

				if len(skey.SEMap) == 0 || skey.SEMap[sevent] {

					if glog.V(2) {
						glog.Info("SubscribeDB: handler( ",
							&d, ", ", skey, ", ", key, ", ", sevent, " )")
					}

					if isSA {
						hFuncSA(d, RunningConfigNotif, "", skey, &key, sevent)
					} else {
						hFunc(d, skey, &key, sevent)
					}
				}
			}
		}

		// Send the Close|Err notification.
		var sEvent = SEventClose
		if !d.sCIP {
			sEvent = SEventErr
		}
		glog.Info("SubscribeDB: SEventClose|Err: ", sEvent)
		if isSA {
			hFuncSA(d, RunningConfigNotif, "", &SKey{}, &Key{}, sEvent)
		} else {
			hFunc(d, &SKey{}, &Key{}, sEvent)
		}
	}()

SubscribeDBExit:

	if e != nil {
		if d.sPubSub != nil {
			d.sPubSub.Close()
		}

		if d.client != nil {
			d.DeleteDB()
			d.client = nil
		}
		d = nil
	}

	if glog.V(3) {
		glog.Info("SubscribeDB: End: d: ", d, " e: ", e)
	}

	return d, e
}

// UnsubscribeDB is used to close a DB subscription
func (d *DB) UnsubscribeDB() error {

	var e error = nil

	if glog.V(3) {
		glog.Info("UnsubscribeDB: d:", d)
	}

	if d.sCIP {
		glog.Error("UnsubscribeDB: Close in Progress")
		e = errors.New("UnsubscribeDB: Close in Progress")
		goto UnsubscribeDBExit
	}

	// Mark close in progress.
	d.sCIP = true

	// Do the close, ch gets closed too.
	d.sPubSub.Close()

	// Wait for the goroutine to complete ? TBD
	// Should not this happen because of the range statement on ch?

	// Close the DB
	d.DeleteDB()

	// UnRegister
	d.unRegisterSubscribeDB()

UnsubscribeDBExit:

	if glog.V(3) {
		glog.Info("UnsubscribeDB: End: d: ", d, " e: ", e)
	}

	return e
}

func (d *DB) key2redisChannel(ts *TableSpec, key Key) string {

	if glog.V(5) {
		glog.Info("key2redisChannel: ", *ts, " key: "+key.String())
	}

	dbId := strconv.Itoa(d.Opts.DBNo.ID())
	return "__keyspace@" + dbId + "__:" + d.key2redis(ts, key)
}

func (d *DB) redisChannel2key(ts *TableSpec, redisChannel string) Key {

	if glog.V(5) {
		glog.Info("redisChannel2key: ", *ts, " redisChannel: "+redisChannel)
	}

	splitRedisKey := strings.SplitN(redisChannel, ":", 2)

	if len(splitRedisKey) > 1 {
		return d.redis2key(ts, splitRedisKey[1])
	}

	glog.Warning("redisChannel2key: Missing key: redisChannel: ", redisChannel)

	return Key{}
}

func (d *DB) redisPayload2sEvent(redisPayload string) SEvent {

	if glog.V(5) {
		glog.Info("redisPayload2sEvent: ", redisPayload)
	}

	sEvent := redisPayload2sEventMap[redisPayload]

	if sEvent == 0 {
		sEvent = SEventOther
	}

	if glog.V(3) {
		glog.Info("redisPayload2sEvent: ", sEvent)
	}

	return sEvent
}

func (d *DB) txOp2sEvent(op _txOp) SEvent {
	return txOp2sEventMap[op]
}
