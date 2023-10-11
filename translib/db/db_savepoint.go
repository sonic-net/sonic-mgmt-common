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

// DB Layer Savepoint
// Support for nested transactions. i.e. a Savepoint is a point to which the
// transaction can be rolled back to without affecting any operations
// performed before the savepoint.
//

import (
	"github.com/Azure/sonic-mgmt-common/cvl"
	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"

	"github.com/golang/glog"
)

// SavePoint type currently assumes it is singleton (only one object is
// instantiated). However, it could be extended to be a stack of SavePoint
// objects.
// Note: Any change to the underlying datastructures it is trying to save,
// can result in a change being required to savePoint as well.
type _savePoint struct {
	// CAS Transaction Operations (txCmds)
	txCmdsLen int

	// CVL Edit Operations (cvlEditConfigData)
	// When appending to cvlEditConfigData,
	// there is always one op, unless the ReplaceOp is true, in which case
	// it is two. (If this statement is not valid, re-write this)
	// Ensure the cvlEditConfigData entries values are not pointers
	cECDLen int

	// Record the CAS Transaction(Tx) Cache prior values, at every change.
	// CAS Tx Cache is only changed in doWrite().
	// CAS Tx Cache indicates an entry deletion by using a zero valued
	// Value structure in the map (i.e. len(Value.Field) == 0)
	// So, to distinguish a indication of a deletion vs. the entry not being
	// present in the cache, we need 2 fields. One to store the prior value
	// (even though it may have been zero valued Value to indicate deletion),
	// and the field to indicate if it was present (or absent) from
	// the cache.
	txTsOrigEntryMap map[string]map[string]origEntry

	// Record the CVL Hints (with the length of cECDLen at the time they were
	// invoked)
	cHints map[int]map[string]interface{}
}

type origEntry struct {
	value  Value
	absent bool
}

var savePoint *_savePoint

func (d *DB) HasSP() bool {
	if d == nil || !d.Opts.IsSession {
		return false
	}
	return savePoint != nil // TODO keep savePoint in DB struct
}

func (d *DB) DeclareSP() error {
	glog.Infof("DeclareSP: Begin")

	if (d == nil) || !d.Opts.IsSession {
		glog.Error("DeclareSP: Invalid Session")
		return tlerr.TranslibInvalidSession{}
	}

	if savePoint != nil {
		glog.Error("DeclareSP: Only one SavePoint Supported")
		return tlerr.TranslibDBNotSupported{}
	}

	savePoint = &_savePoint{txCmdsLen: len(d.txCmds), // Record CAS Tx Ops
		cECDLen:          len(d.cvlEditConfigData), // Record CVL Edit Ops
		txTsOrigEntryMap: make(map[string]map[string]origEntry),
	}

	glog.Infof("DeclareSP: End: %# v", savePoint)
	return nil
}

func (d *DB) ReleaseSP() error {
	glog.Infof("ReleaseSP: Begin")

	if (d == nil) || !d.Opts.IsSession {
		glog.Error("ReleaseSP: Invalid Session")
		return tlerr.TranslibInvalidSession{}
	}

	if savePoint == nil {
		glog.Error("DeclareSP: SavePoint Absent")
		return tlerr.TranslibDBNotSupported{}
	}

	if glog.V(3) {
		glog.Infof("ReleaseSP: End: Releasing %# v", savePoint)
	} else {
		glog.Infof("ReleaseSP: End:")
	}

	savePoint = nil

	return nil
}

func (d *DB) Rollback2SP() error {
	if glog.V(3) {
		glog.Infof("Rollback2SP: Begin: %# v", savePoint)
	} else {
		glog.Infof("Rollback2SP: Begin:")
	}

	if (d == nil) || !d.Opts.IsSession {
		glog.Error("Rollback2SP: Invalid Session")
		return tlerr.TranslibInvalidSession{}
	}

	if savePoint == nil {
		glog.Error("Rollback2SP: SavePoint Absent")
		return tlerr.TranslibDBNotSupported{}
	}

	// Collect the CandidateConfigNotifs to be sent.
	notifOps := make([]_txCmd, 0, len(savePoint.txTsOrigEntryMap))
	for otn, otbl := range savePoint.txTsOrigEntryMap {
		for oRedisKey, oEntry := range otbl {
			if tbl, ok := d.txTsEntryMap[otn]; ok {

				// Note: We are cooking up a TableSpec here because TxCache
				// does not save the original TableSpec.
				ts := &TableSpec{Name: otn}
				key := d.redis2key(ts, oRedisKey)

				if oEntry.absent {

					notifOps = append(notifOps, _txCmd{
						ts: ts, op: txOpDel, key: &key})

				} else {

					hSet, hDel := false, false
					if _, okKey := tbl[oRedisKey]; !okKey {
						glog.Warningf("Rollback2SP: Key missing: %s key: %v",
							otn, oRedisKey)
						hSet = true
					} else {
						hSet, hDel = tbl[oRedisKey].Compare4TxOps(oEntry.value)
					}

					if hSet {
						notifOps = append(notifOps, _txCmd{
							ts: ts, op: txOpHMSet, key: &key})
					}

					if hDel {
						notifOps = append(notifOps, _txCmd{
							ts: ts, op: txOpHDel, key: &key})
					}
				}
			} else {
				glog.Warningf("Rollback2SP: Table missing: %s val: %v", otn,
					oEntry.value)
			}
		}
	}

	// Rollback CAS Tx Operations
	d.txCmds = d.txCmds[0:savePoint.txCmdsLen]

	// The redis CAS Tx cache needs to be rebuilt from scratch, because
	// while reopening (and recreating) the CVL Session, there might be
	// callbacks into the DB Layer (through the CVL DBAccess interface). Thus
	// the redis CAS Tx cache needs to move in lock-step with Validations of
	// the CVL edit ops.
	// Initialize the txTsEntryMap with the values retrieved by HGetAll during
	// doWrite() CAS Tx cache update. If there was no value found by HGetAll
	// a zero Value (i.e. len(Value.Field) == 0) should be there to indicate
	// the key was absent in redis.
	for tn, tb := range d.txTsEntryMap {
		for k := range tb {
			delete(d.txTsEntryMap[tn], k)
		}
	}
	for tn, tb := range d.txTsEntryHGetAll {
		if _, ok := d.txTsEntryMap[tn]; !ok {
			d.txTsEntryHGetAll[tn] = make(map[string]Value)
		}
		for k := range tb {
			d.txTsEntryMap[tn][k] = tb[k].Copy()
		}
	}

	// Rollback CVL Edit Ops Array
	// Reopen the Validation Session to (hopefully) clear the CVL Cache
	var err error
	if d.cv == nil {
		glog.Warningf("Rollback2SP: CVL Session Not Opened")
	} else if ret := cvl.ValidationSessClose(d.cv); ret != cvl.CVL_SUCCESS {
		glog.Warningf("Rollback2SP: Error closing CVL session: ret: %s",
			cvl.GetErrorString(ret))
		err = tlerr.TranslibCVLFailure{Code: int(ret)}
	} else if d.cv, err = d.NewValidationSession(); err != nil {
		glog.Warningf("Rollback2SP: Error opening CVL session: err: %s", err)
	} else {
		// Unless CVL allows multiple "Ops" in one invocation,
		// we have to run through the Validation again step by step.
		// Ops are of length 1, unless it is a ReplaceOp (where it is 2)
		// After Each of the Ops (either 2 for ReplaceOp, or 1 for OtherOp),
		// adjust the CAS Tx cache (in case the next CVL Ops's validations
		// require data from the DB Layer back again.)
		for ix := 0; ix < savePoint.cECDLen; ix++ {

			glog.V(3).Infof("Rollback2SP: Playback %d", ix)

			// Replay the hints at this cECDLen first
			if (savePoint.cHints != nil) && (savePoint.cHints[ix] != nil) {
				for hKey, hValue := range savePoint.cHints[ix] {
					glog.V(3).Infof("Rollback2SP: Playback Hint %s:%v", hKey,
						hValue)
					// TBD Wait for CVL PR
					// hRet := d.cv.StoreHint(hKey, hValue)
					var hRet cvl.CVLRetCode
					if cvl.CVL_SUCCESS != hRet {
						glog.Warningf("Rollback2SP:%d: Hint CVL Failure: %d",
							ix, hRet)
						err = tlerr.TranslibCVLFailure{Code: int(hRet)}
						break
					}
				}
			}
			if err != nil {
				break
			}

			cECD := d.cvlEditConfigData[0 : ix+1]
			cECD[ix].VType = cmn.VALIDATE_ALL
			lenCvlOps := 1
			// TBD Wait for CVL PR
			// if cECD[ix].ReplaceOp {
			// 	cECD = d.cvlEditConfigData[0 : ix+2]
			// 	cECD[ix+1].VType = cmn.VALIDATE_ALL
			// 	lenCvlOps = 2
			// }

			cei, ret := d.cv.ValidateEditConfig(cECD)

			if cvl.CVL_SUCCESS != ret {
				glog.Warning("Rollback2SP: CVL Failure: ", ret)
				glog.Warning("Rollback2SP: ", len(cECD), " ", lenCvlOps)
				err = tlerr.TranslibCVLFailure{Code: int(ret),
					CVLErrorInfo: cei}
				break
			}

			// Fixup the CAS Tx Cache for the current Ops in Lock Step
			ts, key := d.redis2ts_key(cECD[ix].Key)
			rk := d.key2redis(&ts, key)
			if _, ok := d.txTsEntryMap[ts.Name][rk]; !ok {
				d.txTsEntryMap[ts.Name][rk] = Value{
					Field: make(map[string]string)}
			}
			switch cECD[ix].VOp {
			case cmn.OP_CREATE, cmn.OP_UPDATE:
				for fk, fv := range cECD[ix].Data {
					d.txTsEntryMap[ts.Name][rk].Field[fk] = fv
				}
			case cmn.OP_DELETE:
				if len(cECD[ix].Data) == 0 {
					d.txTsEntryMap[ts.Name][rk] = Value{
						Field: make(map[string]string)}
				}
				for fk := range cECD[ix].Data {
					delete(d.txTsEntryMap[ts.Name][rk].Field, fk)
				}
			}
			// TBD Wait for CVL PR
			// if cECD[ix].ReplaceOp {
			// 	for fk := range cECD[ix+1].Data {
			// 		delete(d.txTsEntryMap[ts.Name][rk].Field, fk)
			// 	}
			// }

			cECD[ix].VType = cmn.VALIDATE_NONE
			// TBD Wait for CVL PR
			// if cECD[ix].ReplaceOp {
			// 	cECD[ix+1].VType = cmn.VALIDATE_NONE
			// 	ix++
			// }
		}

		// Zeroise the remaining Hints
		if savePoint.cHints != nil {
			for ix := savePoint.cECDLen; ix < len(d.cvlEditConfigData); ix++ {
				delete(savePoint.cHints, ix)
			}
		}

		// Reset the cvlEditConfigData
		if err == nil {
			d.cvlEditConfigData = d.cvlEditConfigData[0:savePoint.cECDLen]
		}
	}

	if err != nil {
		glog.Warning("Rollback2SP: Setting DB in error flag")
		d.err = err
	}

	// Send the Session Notifications for Subscribers to ConfigDB.
	for _, txCmd := range notifOps {
		// This should be a Config Session, because Rollback2SP
		// is only invoked if d.Opts.IsSession is true.
		d.sendSessionNotification(txCmd.ts, txCmd.key, txCmd.op, txOpNone)
	}
	notifOps = nil

	// Clear the DB Cache
	d.cache = dbCache{Tables: make(map[string]Table, InitialTablesCount),
		Maps: make(map[string]MAP, InitialMapsCount),
	}

	savePoint = nil

	glog.Infof("Rollback2SP: End:")
	return err
}

// doTxSPsave should be called before every change to the CAS Tx Cache.
func (d *DB) doTxSPsave(ts *TableSpec, key Key) {
	if (d == nil) || (savePoint == nil) || !d.Opts.IsSession {
		return
	}

	tsName := ts.Name
	redisKey := d.key2redis(ts, key)
	glog.V(4).Infof("doTxSPsave: Begin: Table: %s redisKey: %s",
		tsName, redisKey)

	if _, ok := savePoint.txTsOrigEntryMap[tsName]; !ok {
		savePoint.txTsOrigEntryMap[tsName] = make(map[string]origEntry)
	}

	// Only record, if we have never recorded the original entry.
	// (On rollback, we don't need to traverse the intermediate entries. The
	// original entry will suffice)
	if _, ok := savePoint.txTsOrigEntryMap[tsName][redisKey]; !ok {
		value, vok := d.txTsEntryMap[tsName][redisKey]
		glog.V(3).Infof("doTxSPsave:Record:T: %s redisKey: %s val: %#v vok: %t",
			tsName, redisKey, value, vok)

		savePoint.txTsOrigEntryMap[tsName][redisKey] = origEntry{
			value: value.Copy(), absent: !vok}
	}
}

// doTxSPsaveHGetAll is a sister func of doTxSPsave, and saves HGetAll() made
// just prior to the time of change to CAS Tx Cache for the first time.
func (d *DB) doTxSPsaveHGetAll(ts *TableSpec, key Key, value Value) {
	if (d == nil) || (savePoint == nil) || !d.Opts.IsSession {
		return
	}

	tsName := ts.Name
	redisKey := d.key2redis(ts, key)
	glog.V(4).Infof("doTxSPsaveHGetAll: Begin: Table: %s redisKey: %s v: %s",
		tsName, redisKey, value)

	if _, ok := d.txTsEntryHGetAll[tsName]; !ok {
		glog.V(4).Infof("doTxSPsaveHGetAll:Missing T: %s redisKey: %s val: %#v",
			tsName, redisKey, value)
		d.txTsEntryHGetAll[tsName] = make(map[string]Value)
	}

	d.txTsEntryHGetAll[tsName][redisKey] = value.Copy()
}

// doCHintSave should be called on successfully Storing a Hint to CVL
func (d *DB) doCHintSave(key string, value interface{}) {
	if (d == nil) || (savePoint == nil) || !d.Opts.IsSession {
		return
	}

	if savePoint.cHints == nil {
		savePoint.cHints = make(map[int]map[string]interface{})
	}

	cECDLen := len(d.cvlEditConfigData)
	if savePoint.cHints[cECDLen] == nil {
		savePoint.cHints[cECDLen] = make(map[string]interface{})
	}

	savePoint.cHints[cECDLen][key] = value
}
