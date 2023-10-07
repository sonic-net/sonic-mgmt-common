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

// Dump the DB contents

const (
	InitialDebugDBSize int = 20
)

func (d *DB) GetDebugDB() interface{} {
	if d == nil {
		return nil
	}
	dbd := make(map[string]interface{}, InitialDebugDBSize)
	dbd["Opts"] = d.Opts
	dbd["txState"] = d.txState
	dbd["txCmds"] = d.GetDebugTxCmds()
	dbd["txTsEntryMap"] = d.txTsEntryMap
	dbd["cvlEditConfigData"] = d.GetDebugCvlECD()
	dbd["cvlHintsB4Open"] = d.cvlHintsB4Open

	dbd["err"] = d.err

	dbd["sCIP"] = d.sCIP
	// Recursive Call?
	// dbd["sOnCCacheDB"] = d.sOnCCacheDB.GetDebugDB()

	dbd["dbStatsConfig"] = d.dbStatsConfig
	dbd["dbCacheConfig"] = d.dbCacheConfig

	dbd["stats"] = d.stats
	dbd["cache"] = d.cache

	dbd["onCReg"] = d.onCReg

	dbd["configDBLocked"] = d.configDBLocked

	return dbd
}

func (d *DB) GetDebugTxCmds() interface{} {
	if d == nil {
		return nil
	}
	txCmds := make([](map[string]interface{}), len(d.txCmds))
	for i := range d.txCmds {
		txCmds[i] = make(map[string]interface{})
		txCmds[i]["ts"] = d.txCmds[i].ts
		txCmds[i]["op"] = d.txCmds[i].op
		txCmds[i]["key"] = d.txCmds[i].key
		txCmds[i]["value"] = d.txCmds[i].value
	}
	return txCmds
}

func (d *DB) GetDebugCvlECD() interface{} {
	if d == nil {
		return nil
	}
	return d.cvlEditConfigData
}
