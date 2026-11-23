//////////////////////////////////////////////////////////////////////////
//
// Copyright 2024 Dell, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//////////////////////////////////////////////////////////////////////////

package transformer

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	flexCounterTablePortKey = "PORT"
	flexCounterPollInterval = "POLL_INTERVAL"
	ratesTableName          = "RATES"
	ratesFieldRxBPS         = "RX_BPS"
	ratesFieldTxBPS         = "TX_BPS"
	ratesFieldRxPPS         = "RX_PPS"
	ratesFieldTxPPS         = "TX_PPS"
)

func init() {
	XlateFuncBind("DbToYang_intf_get_rates_xfmr", DbToYang_intf_get_rates_xfmr)
	XlateFuncBind("Subscribe_intf_get_rates_xfmr", Subscribe_intf_get_rates_xfmr)
	XlateFuncBind("DbToYangPath_intf_get_rates_path_xfmr", DbToYangPath_intf_get_rates_path_xfmr)
}

const intfRatesStateXpathPrefix = "/openconfig-interfaces:interfaces/interface/rates/state"

// isIntfRatesStateURI matches gNMI/translib URIs for openconfig-if-rates (augmented
// module prefix on "rates", e.g. openconfig-if-rates:rates/state). Same approach as
// intf counters: normalize with XfmrRemoveXPATHPredicates, not raw "/rates/state".
func isIntfRatesStateURI(uri string) bool {
	if uri == "" {
		return false
	}
	xpath, _, err := XfmrRemoveXPATHPredicates(uri)
	if err != nil {
		return false
	}
	return strings.Contains(xpath, intfRatesStateXpathPrefix)
}

func getPortFlexPollLoadIntervalSec(configDB *db.DB) uint16 {
	const defaultSec uint16 = 300
	if configDB == nil {
		return defaultSec
	}
	entry, err := configDB.GetEntry(&db.TableSpec{Name: "FLEX_COUNTER_TABLE"}, db.Key{Comp: []string{flexCounterTablePortKey}})
	if err != nil {
		log.Infof("getPortFlexPollLoadIntervalSec: FLEX_COUNTER_TABLE PORT read failed: %v", err)
		return defaultSec
	}
	pollMsStr, ok := entry.Field[flexCounterPollInterval]
	if !ok || pollMsStr == "" {
		return defaultSec
	}
	pollMs, err := strconv.ParseUint(pollMsStr, 10, 64)
	if err != nil {
		return defaultSec
	}
	sec := pollMs / 1000
	if sec < 1 {
		sec = 1
	}
	if sec > 600 {
		sec = 600
	}
	return uint16(sec)
}

func parseRatesCounterUint(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		if f < 0 {
			return 0, nil
		}
		return uint64(math.Round(f)), nil
	}
	return strconv.ParseUint(val, 10, 64)
}

func getRatesEntry(countersDB *db.DB, portOID string) (db.Value, error) {
	return countersDB.GetEntry(&db.TableSpec{Name: ratesTableName}, db.Key{Comp: []string{portOID}})
}

func ratesGetTargetURI(inParams XfmrParams) string {
	target := inParams.uri
	if inParams.requestUri != "" && inParams.requestUri != inParams.uri && isIntfRatesStateURI(inParams.requestUri) {
		target = inParams.requestUri
	}
	yangPath, err := getYangPathFromUri(target)
	if err != nil {
		return NewPathInfo(target).YangPath
	}
	return yangPath
}

func ratesStateTargetAllLeaves(targetURI string) bool {
	xpath, _, err := XfmrRemoveXPATHPredicates(targetURI)
	if err == nil && xpath != "" {
		return strings.HasSuffix(xpath, "/rates/state")
	}
	yangPath, _ := getYangPathFromUri(targetURI)
	return strings.HasSuffix(yangPath, "/rates/state") ||
		strings.HasSuffix(yangPath, "openconfig-if-rates:rates/state")
}

func ratesStateTargetLeaf(targetURI, leaf string) bool {
	if ratesStateTargetAllLeaves(targetURI) {
		return true
	}
	yangPath, err := getYangPathFromUri(targetURI)
	if err != nil {
		return strings.HasSuffix(targetURI, "/"+leaf)
	}
	return strings.HasSuffix(yangPath, "/"+leaf)
}

func populateIntfRatesState(inParams XfmrParams, state *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Rates_State) error {
	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")
	targetURI := ratesGetTargetURI(inParams)

	countersDB := inParams.dbs[db.CountersDB]
	if countersDB == nil && inParams.curDb == db.CountersDB {
		countersDB = inParams.d
	}
	if countersDB == nil {
		return errors.New("Counters DB not available")
	}

	oid, err := getIntfCountersTblKey(countersDB, ifName)
	if err != nil {
		log.Info(err)
		return err
	}

	loadSec := getPortFlexPollLoadIntervalSec(inParams.dbs[db.ConfigDB])
	if ratesStateTargetLeaf(targetURI, "load-interval") {
		v := loadSec
		state.LoadInterval = &v
	}

	ratesEntry, ratesErr := getRatesEntry(countersDB, oid)
	if ratesErr != nil {
		log.Infof("populateIntfRatesState: no RATES entry for %s oid %s: %v", ifName, oid, ratesErr)
		return nil
	}

	rxBps, _ := parseRatesCounterUint(ratesEntry.Field[ratesFieldRxBPS])
	txBps, _ := parseRatesCounterUint(ratesEntry.Field[ratesFieldTxBPS])
	rxPps, _ := parseRatesCounterUint(ratesEntry.Field[ratesFieldRxPPS])
	txPps, _ := parseRatesCounterUint(ratesEntry.Field[ratesFieldTxPPS])

	assignRateLeaf := func(leaf string, value uint64, set func(*uint64)) {
		if !ratesStateTargetLeaf(targetURI, leaf) {
			return
		}
		v := value
		set(&v)
	}

	assignRateLeaf("in-bits-rate", rxBps*8, func(p *uint64) { state.InBitsRate = p })
	assignRateLeaf("out-bits-rate", txBps*8, func(p *uint64) { state.OutBitsRate = p })
	assignRateLeaf("in-pkts-rate", rxPps, func(p *uint64) { state.InPktsRate = p })
	assignRateLeaf("out-pkts-rate", txPps, func(p *uint64) { state.OutPktsRate = p })

	return nil
}

var DbToYang_intf_get_rates_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")
	targetUriPath, _ := getYangPathFromUri(inParams.uri)
	log.Info("DbToYang_intf_get_rates_xfmr targetUriPath=", targetUriPath, " uri=", inParams.uri, " requestUri=", inParams.requestUri)

	if !isIntfRatesStateURI(inParams.uri) && !isIntfRatesStateURI(inParams.requestUri) {
		log.Infof("%s is redundant for rates xfmr", targetUriPath)
		return nil
	}

	intfType, _, ierr := getIntfTypeByName(ifName)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_get_rates_xfmr - Invalid interface type IntfTypeUnset")
		return errors.New("Invalid interface type IntfTypeUnset")
	}
	if intfType != IntfTypeEthernet {
		log.Infof("Rates for interface %s not supported (Ethernet only)", ifName)
		return nil
	}

	intfsObj := getIntfsRoot(inParams.ygRoot)
	var state *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Rates_State

	if intfsObj != nil && intfsObj.Interface != nil && len(intfsObj.Interface) > 0 {
		intfObj, ok := intfsObj.Interface[ifName]
		if !ok {
			intfObj, _ = intfsObj.NewInterface(ifName)
			ygot.BuildEmptyTree(intfObj)
		}
		ygot.BuildEmptyTree(intfObj)
		if intfObj.Rates == nil {
			ygot.BuildEmptyTree(intfObj)
		}
		if intfObj.Rates.State == nil {
			ygot.BuildEmptyTree(intfObj.Rates)
		}
		state = intfObj.Rates.State
	} else {
		ygot.BuildEmptyTree(intfsObj)
		intfObj, _ := intfsObj.NewInterface(ifName)
		ygot.BuildEmptyTree(intfObj)
		if intfObj.Rates == nil {
			ygot.BuildEmptyTree(intfObj)
		}
		if intfObj.Rates.State == nil {
			ygot.BuildEmptyTree(intfObj.Rates)
		}
		state = intfObj.Rates.State
	}

	err := populateIntfRatesState(inParams, state)
	if log.V(3) {
		log.Info("DbToYang_intf_get_rates_xfmr state=", state)
	}
	return err
}

var Subscribe_intf_get_rates_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams

	if inParams.subscProc != TRANSLATE_SUBSCRIBE {
		return result, nil
	}

	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")
	if ifName == "" || ifName == "*" {
		ifName = "*"
	}

	result.nOpts = new(notificationOpts)
	result.nOpts.pType = Sample
	result.nOpts.mInterval = 30
	result.isVirtualTbl = false
	result.needCache = true

	result.dbDataMap = RedisDbSubscribeMap{
		db.CountersDB: {
			"COUNTERS_PORT_NAME_MAP": {"": {FIELD_CURSOR: ifName}},
			ratesTableName:           {"*": {}},
		},
		db.ConfigDB: {
			"FLEX_COUNTER_TABLE": {flexCounterTablePortKey: {}},
		},
	}
	return result, nil
}

var DbToYangPath_intf_get_rates_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	intfRoot := "/openconfig-interfaces:interfaces/interface"

	switch params.tblName {
	case "COUNTERS_PORT_NAME_MAP":
		if len(params.tblKeyComp) > 0 {
			params.ygPathKeys[intfRoot+"/name"] = params.tblKeyComp[0]
		}
	case ratesTableName:
		if len(params.tblKeyComp) == 0 {
			return nil
		}
		oid := params.tblKeyComp[0]
		countersDB := params.dbs[db.CountersDB]
		if countersDB == nil {
			return nil
		}
		portMap, err := countersDB.GetMapAll(&db.TableSpec{Name: "COUNTERS_PORT_NAME_MAP"})
		if err != nil || !portMap.IsPopulated() {
			return nil
		}
		for ifName, ifOID := range portMap.Field {
			if ifOID == oid {
				params.ygPathKeys[intfRoot+"/name"] = ifName
				break
			}
		}
	}
	return nil
}
