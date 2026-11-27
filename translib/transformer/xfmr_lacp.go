////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Dell, Inc.                                                 //
//  Copyright 2026 Nokia.                                                     //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package transformer

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

var (
	/* The set of fields in the PORTCHANNEL table controlled by openconfig-lacp */
	PortChanFlds = [...]string{"system_mac", "fast_rate", "fallback"}
)

func init() {
	XlateFuncBind("YangToDb_lacp_config_xfmr", YangToDb_lacp_config_xfmr)
	XlateFuncBind("DbToYang_lacp_config_xfmr", DbToYang_lacp_config_xfmr)
	XlateFuncBind("Subscribe_lacp_config_xfmr", Subscribe_lacp_config_xfmr)
	XlateFuncBind("DbToYang_lacp_state_xfmr", DbToYang_lacp_state_xfmr)
	XlateFuncBind("Subscribe_lacp_state_xfmr", Subscribe_lacp_state_xfmr)
	XlateFuncBind("DbToYang_lacp_intfs_xfmr", DbToYang_lacp_intfs_xfmr)
	XlateFuncBind("YangToDb_lacp_intfs_xfmr", YangToDb_lacp_intfs_xfmr)
	XlateFuncBind("Subscribe_lacp_intfs_xfmr", Subscribe_lacp_intfs_xfmr)
}

func getLacpRoot(s *ygot.GoStruct) *ocbinds.OpenconfigLacp_Lacp {
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.Lacp
}

func hasBit(n int, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}

func String2Uint(b string, dflt uint64) *uint64 {
	v, err := strconv.ParseUint(b, 10, 64)
	if err != nil {
		if b != "" {
			log.V(3).Infof("string conversion failure for %v; err = %v", b, err)
		}
		return &dflt
	}
	return &v
}

func updateSubOpDataMap(subOpMap map[db.DBNum]map[string]map[string]db.Value, oper Operation, inParams XfmrParams) {
	if len(subOpMap[db.ConfigDB]) == 0 {
		return
	}
	if inParams.subOpDataMap[oper] == nil {
		inParams.subOpDataMap[oper] = &subOpMap
		return
	}
	if (*inParams.subOpDataMap[oper])[db.ConfigDB] == nil {
		(*inParams.subOpDataMap[oper])[db.ConfigDB] = make(map[string]map[string]db.Value)
	}
	mapCopy((*inParams.subOpDataMap[oper])[db.ConfigDB], subOpMap[db.ConfigDB])
}

func getLacpInterfaceConfigInfo(inParams XfmrParams, ifKey string) (map[string]db.Value, error) {
	infoMap := make(map[string]db.Value)

	prtInst := (*inParams.dbDataMap)[db.ConfigDB][PORTCHANNEL_TN][ifKey]
	infoMap[ifKey] = db.Value{
		Field: map[string]string{
			"interval":      prtInst.Field["fast_rate"],
			"system-id-mac": prtInst.Field["system_mac"],
			"fallback":      prtInst.Field["fallback"],
		},
	}
	return infoMap, nil
}

func getLacpInterfaceStateInfo(inParams XfmrParams, ifKey string) (map[string]db.Value, error) {
	infoMap := make(map[string]db.Value)

	prtInst := (*inParams.dbDataMap)[db.StateDB][LAG_TABLE_TN][ifKey]

	infoMap[ifKey] = db.Value{
		Field: map[string]string{
			"interval":        prtInst.Field["runner.fast_rate"],
			"lacp-mode":       prtInst.Field["runner.active"],
			"system-id-mac":   prtInst.Field["team_device.ifinfo.dev_addr"],
			"system-priority": prtInst.Field["runner.sys_prio"],
			"fallback":        prtInst.Field["runner.fallback"],
		},
	}
	return infoMap, nil
}

func prepareFallbackVal(infoMap map[string]db.Value, ifKey string) *bool {
	defaultFbVal := false
	if fb, ok := infoMap[ifKey].Field["fallback"]; ok {
		if fbVal, err := strconv.ParseBool(fb); err != nil {
			log.V(3).Infof("Error parsing fallback mode for %v. Use default value false. Err: %v", ifKey, err)
			return &defaultFbVal
		} else {
			return &fbVal
		}
	} else {
		log.V(3).Infof("Return default fallback mode value false for %v. ", ifKey)
		return &defaultFbVal
	}
}

/* lacpMemberCounterField returns the teamd LACP counter from STATE_DB. Legacy teamd
 * serviceability (tlm_teamd) publishes runner.statistics.*; older paths used
 * runner.counters.*. Prefer the new keys when present. */
func lacpMemberCounterField(fields map[string]string, newKey, oldKey string) string {
	if v, ok := fields[newKey]; ok && v != "" {
		return v
	}
	return fields[oldKey]
}

func getLacpMemberInfo(inParams XfmrParams, ifKey string, ifMemKey string) (map[string]db.Value, error) {
	memberInfoMap := make(map[string]db.Value)

	prtInst := (*inParams.dbDataMap)[db.StateDB][LAG_MEMBER_TABLE_TN][ifKey+"|"+ifMemKey]
	fld := map[string]string{
		"system-id":       prtInst.Field["runner.actor_lacpdu_info.system"],
		"oper-key":        prtInst.Field["runner.actor_lacpdu_info.key"],
		"partner-id":      prtInst.Field["runner.partner_lacpdu_info.system"],
		"partner-key":     prtInst.Field["runner.partner_lacpdu_info.key"],
		"lacp-in-pkts":    lacpMemberCounterField(prtInst.Field, "runner.statistics.lacpdu_rx_stats", "runner.counters.lacp-in-packets"),
		"lacp-out-pkts":   lacpMemberCounterField(prtInst.Field, "runner.statistics.lacpdu_tx_stats", "runner.counters.lacp-out-packets"),
		"lacp-rx-errors":  lacpMemberCounterField(prtInst.Field, "runner.statistics.lacpdu_illegal_pkts", "runner.counters.lacp-rx-errors"),
		"activity":        "lacp-activity-type:PASSIVE",
		"timeout":         "lacp-timeout-type:LONG",
		"aggregatable":    "false",
		"synchronization": "lacp-synchronization-type:OUT_SYNC",
		"collecting":      "false",
		"distributing":    "false",
	}
	memberInfoMap[ifMemKey] = db.Value{Field: fld}

	port, ok := prtInst.Field["runner.actor_lacpdu_info.port"]
	if ok {
		fld["port"] = port
	}
	partner_port, ok := prtInst.Field["runner.partner_lacpdu_info.port"]
	if ok {
		fld["partner-port"] = partner_port
	}
	actorStateMask := 0
	actorStateMaskStr, ok := prtInst.Field["runner.actor_lacpdu_info.state"]
	if ok {
		var err error
		actorStateMask, err = strconv.Atoi(actorStateMaskStr)
		if err != nil {
			log.V(3).Infof("ifKey=%s ifMemKey=%s, error %v converting actor-state-mask (%s) to int", ifKey, ifMemKey, err, actorStateMaskStr)
			return memberInfoMap, err
		}
	}
	if hasBit(actorStateMask, 0) {
		fld["activity"] = "lacp-activity-type:ACTIVE"
	}
	/* IEEE 802.1AX LACP_Timeout: 1 = Short Timeout, 0 = Long Timeout */
	if hasBit(actorStateMask, 1) {
		fld["timeout"] = "lacp-timeout-type:SHORT"
	}
	if hasBit(actorStateMask, 2) {
		fld["aggregatable"] = "true"
	}
	if hasBit(actorStateMask, 3) {
		fld["synchronization"] = "lacp-synchronization-type:IN_SYNC"
	}
	if hasBit(actorStateMask, 4) {
		fld["collecting"] = "true"
	}
	if hasBit(actorStateMask, 5) {
		fld["distributing"] = "true"
	}

	return memberInfoMap, nil
}

func _fillLacpInterfaceConfig(ifKey string, infoMap map[string]db.Value,
	configObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Config) error {

	configObj.Name = &ifKey
	configObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_SLOW
	if infoMap[ifKey].Field["interval"] == "true" {
		configObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_FAST
	}

	sim, ok := infoMap[ifKey].Field["system-id-mac"]
	if ok && sim != "" {
		configObj.SystemIdMac = &sim
	}

	configObj.Fallback = prepareFallbackVal(infoMap, ifKey)

	return nil
}

func _fillLacpInterfaceState(ifKey string, infoMap map[string]db.Value, targetUriPath string,
	stateObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_State,
	membersObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members) error {

	switch targetUriPath {
	case "/openconfig-lacp:lacp/interfaces":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface/state":
		stateObj.Name = &ifKey
		stateObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_SLOW
		if infoMap[ifKey].Field["interval"] == "true" {
			stateObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_FAST
		}
		stateObj.LacpMode = ocbinds.OpenconfigLacp_LacpActivityType_ACTIVE
		if infoMap[ifKey].Field["lacp-mode"] == "false" {
			stateObj.LacpMode = ocbinds.OpenconfigLacp_LacpActivityType_PASSIVE
		}
		sim := infoMap[ifKey].Field["system-id-mac"]
		if sim != "" {
			stateObj.SystemIdMac = &sim
		}
		sp, err := strconv.Atoi(infoMap[ifKey].Field["system-priority"])
		if err == nil {
			usp := uint16(sp)
			stateObj.SystemPriority = &usp
		}
		stateObj.Fallback = prepareFallbackVal(infoMap, ifKey)
	case "/openconfig-lacp:lacp/interfaces/interface/state/name":
		stateObj.Name = &ifKey
	case "/openconfig-lacp:lacp/interfaces/interface/state/interval":
		stateObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_SLOW
		if infoMap[ifKey].Field["interval"] == "true" {
			stateObj.Interval = ocbinds.OpenconfigLacp_LacpPeriodType_FAST
		}
	case "/openconfig-lacp:lacp/interfaces/interface/state/lacp-mode":
		stateObj.LacpMode = ocbinds.OpenconfigLacp_LacpActivityType_ACTIVE
		if infoMap[ifKey].Field["lacp-mode"] == "false" {
			stateObj.LacpMode = ocbinds.OpenconfigLacp_LacpActivityType_PASSIVE
		}
	case "/openconfig-lacp:lacp/interfaces/interface/state/system-id-mac":
		sim := infoMap[ifKey].Field["system-id-mac"]
		stateObj.SystemIdMac = &sim
	case "/openconfig-lacp:lacp/interfaces/interface/state/system-priority":
		sp, err := strconv.Atoi(infoMap[ifKey].Field["system-priority"])
		if err != nil {
			return err
		}
		usp := uint16(sp)
		stateObj.SystemPriority = &usp
	case "/openconfig-lacp:lacp/interfaces/interface/state/fallback":
		stateObj.Fallback = prepareFallbackVal(infoMap, ifKey)
	default:
		log.V(3).Infof("Unsupported path; name=%s path=%s", ifKey, targetUriPath)
		return tlerr.InvalidArgsError{Format: "unsupported path"}
	}

	return nil
}

func _fillLacpMember(ifMemKey string, memberInfoMap map[string]db.Value,
	lacpMemberObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member) error {
	memEntries, ok := memberInfoMap[ifMemKey]
	if !ok {
		errStr := "LACP Member Information not available"
		return tlerr.InvalidArgsError{Format: errStr}
	}

	lacpMemberObj.State.Interface = &ifMemKey

	lacpMemberObj.State.Activity = ocbinds.OpenconfigLacp_LacpActivityType_ACTIVE
	if memEntries.Field["activity"] == "lacp-activity-type:PASSIVE" {
		lacpMemberObj.State.Activity = ocbinds.OpenconfigLacp_LacpActivityType_PASSIVE
	}

	lacpMemberObj.State.Timeout = ocbinds.OpenconfigLacp_LacpTimeoutType_LONG
	if memEntries.Field["timeout"] == "lacp-timeout-type:SHORT" {
		lacpMemberObj.State.Timeout = ocbinds.OpenconfigLacp_LacpTimeoutType_SHORT
	}

	aggregatable := true
	if memEntries.Field["aggregatable"] == "false" {
		aggregatable = false
	}
	lacpMemberObj.State.Aggregatable = &aggregatable

	lacpMemberObj.State.Synchronization = ocbinds.OpenconfigLacp_LacpSynchronizationType_IN_SYNC
	if memEntries.Field["synchronization"] == "lacp-synchronization-type:OUT_SYNC" {
		lacpMemberObj.State.Synchronization = ocbinds.OpenconfigLacp_LacpSynchronizationType_OUT_SYNC
	}

	collecting := true
	if memEntries.Field["collecting"] == "false" {
		collecting = false
	}
	lacpMemberObj.State.Collecting = &collecting

	distributing := true
	if memEntries.Field["distributing"] == "false" {
		distributing = false
	}
	lacpMemberObj.State.Distributing = &distributing

	system_id := memEntries.Field["system-id"]
	lacpMemberObj.State.SystemId = &system_id

	oper_key := *String2Uint(memEntries.Field["oper-key"], 0)
	uoper_key := uint16(oper_key)
	lacpMemberObj.State.OperKey = &uoper_key

	partner_id := memEntries.Field["partner-id"]
	lacpMemberObj.State.PartnerId = &partner_id

	partner_key := *String2Uint(memEntries.Field["partner-key"], 0)
	upartner_key := uint16(partner_key)
	lacpMemberObj.State.PartnerKey = &upartner_key

	pport_str, ok := memEntries.Field["partner-port"]
	if ok {
		partner_port := *String2Uint(pport_str, 0)
		upartner_port := uint16(partner_port)
		lacpMemberObj.State.PartnerPortNum = &upartner_port
	}

	port_str, ok := memEntries.Field["port"]
	if ok {
		port := *String2Uint(port_str, 0)
		uport := uint16(port)
		lacpMemberObj.State.PortNum = &uport
	}

	lacpMemberObj.State.Counters.LacpInPkts = String2Uint(memEntries.Field["lacp-in-pkts"], 0)

	lacpMemberObj.State.Counters.LacpOutPkts = String2Uint(memEntries.Field["lacp-out-pkts"], 0)

	lacpMemberObj.State.Counters.LacpRxErrors = String2Uint(memEntries.Field["lacp-rx-errors"], 0)

	return nil
}

func _fillLacpMemberAttr(path string, ifMemKey string, memberInfoMap map[string]db.Value,
	lacpMemberObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member) error {
	memEntries, ok := memberInfoMap[ifMemKey]
	if !ok {
		return tlerr.InvalidArgsError{Format: "LACP Member Information not available"}
	}

	switch path {
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/interface":
		lacpMemberObj.Interface = &ifMemKey
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/interface":
		lacpMemberObj.State.Interface = &ifMemKey
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/activity":
		lacpMemberObj.State.Activity = ocbinds.OpenconfigLacp_LacpActivityType_ACTIVE
		if memEntries.Field["activity"] == "lacp-activity-type:PASSIVE" {
			lacpMemberObj.State.Activity = ocbinds.OpenconfigLacp_LacpActivityType_PASSIVE
		}
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/timeout":
		lacpMemberObj.State.Timeout = ocbinds.OpenconfigLacp_LacpTimeoutType_LONG
		if memEntries.Field["timeout"] == "lacp-timeout-type:SHORT" {
			lacpMemberObj.State.Timeout = ocbinds.OpenconfigLacp_LacpTimeoutType_SHORT
		}
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/aggregatable":
		aggregatable := true
		if memEntries.Field["aggregatable"] == "false" {
			aggregatable = false
		}
		lacpMemberObj.State.Aggregatable = &aggregatable
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/synchronization":
		lacpMemberObj.State.Synchronization = ocbinds.OpenconfigLacp_LacpSynchronizationType_IN_SYNC
		if memEntries.Field["synchronization"] == "lacp-synchronization-type:OUT_SYNC" {
			lacpMemberObj.State.Synchronization = ocbinds.OpenconfigLacp_LacpSynchronizationType_OUT_SYNC
		}
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/collecting":
		collecting := true
		if memEntries.Field["collecting"] == "false" {
			collecting = false
		}
		lacpMemberObj.State.Collecting = &collecting
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/distributing":
		distributing := true
		if memEntries.Field["distributing"] == "false" {
			distributing = false
		}
		lacpMemberObj.State.Distributing = &distributing
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/system-id":
		system_id := memEntries.Field["system-id"]
		lacpMemberObj.State.SystemId = &system_id
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/oper-key":
		oper_key := *String2Uint(memEntries.Field["oper-key"], 0)
		uoper_key := uint16(oper_key)
		lacpMemberObj.State.OperKey = &uoper_key
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/partner-id":
		partner_id := memEntries.Field["partner-id"]
		lacpMemberObj.State.PartnerId = &partner_id
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/partner-key":
		partner_key := *String2Uint(memEntries.Field["partner-key"], 0)
		upartner_key := uint16(partner_key)
		lacpMemberObj.State.PartnerKey = &upartner_key
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/port-num":
		port_num := *String2Uint(memEntries.Field["port"], 0)
		port_num_u16 := uint16(port_num)
		lacpMemberObj.State.PortNum = &port_num_u16
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/partner-port-num":
		partner_port_num := *String2Uint(memEntries.Field["partner-port"], 0)
		partner_port_num_u16 := uint16(partner_port_num)
		lacpMemberObj.State.PartnerPortNum = &partner_port_num_u16
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/counters":
		ulacp_in_pkts := *String2Uint(memEntries.Field["lacp-in-pkts"], 0)
		lacpMemberObj.State.Counters.LacpInPkts = &ulacp_in_pkts

		ulacp_out_pkts := *String2Uint(memEntries.Field["lacp-out-pkts"], 0)
		lacpMemberObj.State.Counters.LacpOutPkts = &ulacp_out_pkts

		ulacp_rx_errors := *String2Uint(memEntries.Field["lacp-rx-errors"], 0)
		lacpMemberObj.State.Counters.LacpRxErrors = &ulacp_rx_errors
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/counters/lacp-in-pkts":
		ulacp_in_pkts := *String2Uint(memEntries.Field["lacp-in-pkts"], 0)
		lacpMemberObj.State.Counters.LacpInPkts = &ulacp_in_pkts
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/counters/lacp-out-pkts":
		ulacp_out_pkts := *String2Uint(memEntries.Field["lacp-out-pkts"], 0)
		lacpMemberObj.State.Counters.LacpOutPkts = &ulacp_out_pkts
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state/counters/lacp-rx-errors":
		ulacp_rx_errors := *String2Uint(memEntries.Field["lacp-rx-errors"], 0)
		lacpMemberObj.State.Counters.LacpRxErrors = &ulacp_rx_errors
	default:
		log.V(3).Infof("Unsupported path filling member %s: %s", ifMemKey, path)
		return tlerr.InvalidArgsError{Format: "unsupported path"}
	}

	return nil
}

func populateLacpInterfaceState(inParams XfmrParams, ifKey string, targetUriPath string, itf *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface) error {
	stateInfoMap, err := getLacpInterfaceStateInfo(inParams, ifKey)
	if err != nil {
		return fmt.Errorf("%w; getLacpInterfaceStateInfo failed for %s", err, ifKey)
	}

	// interface/state/lacp-key is derived from member/state/oper-key,
	// so members are populated to support populating interface/state
	if err = populateLacpMembers(inParams, ifKey, itf.Members); err != nil {
		return fmt.Errorf("%w; populateLacpMembers failed for %s", err, ifKey)
	}

	if err = _fillLacpInterfaceState(ifKey, stateInfoMap, targetUriPath, itf.State, itf.Members); err != nil {
		return fmt.Errorf("%w; _fillLacpInterfaceState failed for %s", err, ifKey)
	}
	return nil
}

func populateLacpData(inParams XfmrParams, ifKey string, targetUriPath string, itf *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface) error {
	configInfoMap, err := getLacpInterfaceConfigInfo(inParams, ifKey)
	if err != nil {
		return fmt.Errorf("%w; getLacpInterfaceConfigInfo failed for %s", err, ifKey)
	}
	if err = _fillLacpInterfaceConfig(ifKey, configInfoMap, itf.Config); err != nil {
		return fmt.Errorf("%w; _fillLacpInterfaceConfig failed for %s", err, ifKey)
	}
	if err = populateLacpInterfaceState(inParams, ifKey, targetUriPath, itf); err != nil {
		return fmt.Errorf("%w; populateLacpInterfaceState failed for %s", err, ifKey)
	}
	return nil
}

func populateLacpMembers(inParams XfmrParams, ifKey string, members *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members) error {
	if members == nil {
		ygot.BuildEmptyTree(members)
	}
	var lagMbrTbl map[string]db.Value = nil
	if _, ok := (*inParams.dbDataMap)[db.StateDB]; ok {
		lagMbrTbl, _ = (*inParams.dbDataMap)[db.StateDB][LAG_MEMBER_TABLE_TN]
	}
	if lagMbrTbl == nil {
		// Usually the framework would have populated the member table in dbDataMap, however it
		// does not seem to find it as a child table from the PORTCHANNEL or LAG_TABLE tables.
		// Also, if the request was for lacp/interfaces/interface/config|state then the framework
		// may not fill it in since that table isn't related to the requested paths. However, the
		// lacp-key is actually taken from a random member so we do need the member table but
		// since this is a non-standard mapping we do not have a way to tell the framework to
		// pre-fetch the member table so we read it here.
		(*inParams.dbDataMap)[db.StateDB][LAG_MEMBER_TABLE_TN] = map[string]db.Value{}
		stateDb := inParams.dbs[db.StateDB]
		for k, _ := range (*inParams.dbDataMap)[db.StateDB][LAG_TABLE_TN] {
			mbrTblName := LAG_MEMBER_TABLE_TN + stateDb.Opts.KeySeparator + k
			mbrTbl := db.TableSpec{Name: mbrTblName}
			memTblKeys, err := stateDb.GetKeys(&mbrTbl)
			if err != nil {
				return fmt.Errorf("%w; Unable to retrieve member keys for %s", err, ifKey)
			}
			for i := range memTblKeys {
				mbr := memTblKeys[i].Get(1)
				prtInst, err := stateDb.GetEntry(&mbrTbl, db.Key{Comp: []string{mbr}})
				if err != nil {
					log.V(3).Infof("Error retrieving member: %s %s, err=%v", mbrTblName, mbr, err)
					return err
				}
				mbrKey := k + stateDb.Opts.KeySeparator + mbr
				(*inParams.dbDataMap)[db.StateDB][LAG_MEMBER_TABLE_TN][mbrKey] = prtInst
			}
		}
		lagMbrTbl = (*inParams.dbDataMap)[db.StateDB][LAG_MEMBER_TABLE_TN]
	}

	for k, _ := range lagMbrTbl {
		pcName, memName, ok := strings.Cut(k, "|")
		if !ok {
			continue
		}
		if pcName != ifKey {
			continue
		}
		lacpMemberObj, ok := members.Member[memName]
		if !ok {
			var err error
			lacpMemberObj, err = members.NewMember(memName)
			if err != nil {
				return fmt.Errorf("%w; Creation of portchannel member subtree failed", err)
			}
			ygot.BuildEmptyTree(lacpMemberObj)
		}
		if err := populateLacpMember(inParams, ifKey, memName, lacpMemberObj); err != nil {
			return fmt.Errorf("%w; populateLacpMember failed for %s %s", err, ifKey, memName)
		}
	}

	return nil
}

func populateLacpMember(inParams XfmrParams, ifKey string, ifMemKey string, lacpMemberObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member) error {

	memberInfo, err := getLacpMemberInfo(inParams, ifKey, ifMemKey)
	if err != nil {
		return fmt.Errorf("%w; populateLacpMember: getLacpMemberInfo() failed", err)
	}

	e := _fillLacpMember(ifMemKey, memberInfo, lacpMemberObj)
	if e != nil {
		return fmt.Errorf("%w; _fillLacpMember() failed", e)
	}

	return nil
}

func populateLacpMemberAttr(inParams XfmrParams, ifKey string, ifMemKey string, targetUriPath string, lacpMemberObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member) error {

	memberInfo, err := getLacpMemberInfo(inParams, ifKey, ifMemKey)
	if err != nil {
		return fmt.Errorf("%w; populateLacpMemberAttr: getLacpMemberInfo() failed", err)
	}

	if e := _fillLacpMemberAttr(targetUriPath, ifMemKey, memberInfo, lacpMemberObj); e != nil {
		return fmt.Errorf("%w; Failure in filling LACP member attr data for path=%s", e, targetUriPath)
	}

	return nil
}

func prepareMemberObj(lacpIntfsObj *ocbinds.OpenconfigLacp_Lacp, ifKey string, ifMemKey string) (*ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member, error) {
	lacpintfObj, ok := lacpIntfsObj.Interfaces.Interface[ifKey]
	if !ok {
		return nil, errors.New("prepareMemberObj: PortChannel Instance doesn't exist")
	}
	members := lacpintfObj.Members
	if members != nil && ifMemKey != "" {
		member, ok := members.Member[ifMemKey]
		if !ok {
			return nil, errors.New("prepareMemberObj: PortChannel Member Instance doesn't exist")
		}
		ygot.BuildEmptyTree(member)
		/* It is possible the framework partially initialized "member", so State was
		 * allocated but not any children under State.  Handle that here by creating
		 * the "Counters" struct (the only struct two levels under "member"). */
		if member.State.Counters == nil {
			ygot.BuildEmptyTree(member.State)
		}
		return member, nil
	}
	if ifMemKey == "" {
		return nil, errors.New("prepareMemberObj: ifMemKey key is empty, ifKey:" + ifKey)
	}
	return nil, errors.New("prepareMemberObj: PortChannel Members doesn't exist: " + ifKey + "|" + ifMemKey)
}

var YangToDb_lacp_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	// Base openconfig-lacp config only has system-priority at the global level.
	// No DB mapping needed for system-priority at the global level in SONiC.
	return nil, nil
}

var Subscribe_lacp_config_xfmr = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	result.isVirtualTbl = true
	return result, nil
}

var Subscribe_lacp_state_xfmr = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	result.isVirtualTbl = true
	return result, nil
}

var DbToYang_lacp_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	lacpObj := getLacpRoot(inParams.ygRoot)
	ygot.BuildEmptyTree(lacpObj)
	return nil
}

var DbToYang_lacp_state_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	lacpObj := getLacpRoot(inParams.ygRoot)
	ygot.BuildEmptyTree(lacpObj)
	return nil
}

var DbToYang_lacp_intfs_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	lacpIntfsObj := getLacpRoot(inParams.ygRoot)
	var members *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members
	var member *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface_Members_Member
	var lacpintfObj *ocbinds.OpenconfigLacp_Lacp_Interfaces_Interface
	var ok bool

	pathInfo := NewPathInfo(inParams.uri)
	ifKey := pathInfo.Var("name")
	ifMemKey := pathInfo.Var("interface")

	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return fmt.Errorf("%w; DbToYang_lacp_intfs_xfmr: getYangPathFromUri() failed with %s", err, pathInfo.Path)
	}

	log.V(3).Infof("Received %v for path: %s; template: %s vars: %v targetUriPath: %s ifKey: %v ifMemKey: %v lacpRoot: %#v", inParams.oper, pathInfo.Path, pathInfo.Template, pathInfo.Vars, targetUriPath, ifKey, ifMemKey, lacpIntfsObj)
	log.V(3).Infof("DbToYang_lacp_intfs_xfmr: inParams=%#v", inParams)
	log.V(3).Infof("DbToYang_lacp_intfs_xfmr: dbDataMap[4]=%#v", (*inParams.dbDataMap)[db.ConfigDB])
	log.V(3).Infof("DbToYang_lacp_intfs_xfmr: dbDataMap[6]=%#v", (*inParams.dbDataMap)[db.StateDB])

	switch targetUriPath {
	case "/openconfig-lacp:lacp/interfaces":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface":
		ygot.BuildEmptyTree(lacpIntfsObj)
		keys := []db.Key{}
		if ifKey == "" {
			for k, _ := range (*inParams.dbDataMap)[db.ConfigDB][PORTCHANNEL_TN] {
				request_key := db.NewKey(k)
				keys = append(keys, *request_key)
			}
		} else {
			request_key := db.NewKey(ifKey)
			keys = []db.Key{*request_key}
		}
		for _, key := range keys {
			k := key.Get(0)
			lacpintfObj, ok = lacpIntfsObj.Interfaces.Interface[k]
			if !ok {
				lacpintfObj, _ = lacpIntfsObj.Interfaces.NewInterface(k)
			}
			ygot.BuildEmptyTree(lacpintfObj)

			populateLacpData(inParams, k, targetUriPath, lacpintfObj)
		}
	case "/openconfig-lacp:lacp/interfaces/interface/name":
		ygot.BuildEmptyTree(lacpIntfsObj)
		lacpintfObj, ok = lacpIntfsObj.Interfaces.Interface[ifKey]
		if !ok {
			lacpintfObj, _ = lacpIntfsObj.Interfaces.NewInterface(ifKey)
		}
		ygot.BuildEmptyTree(lacpintfObj)
		_, ok := (*inParams.dbDataMap)[db.ConfigDB][PORTCHANNEL_TN][ifKey]
		if ok {
			lacpintfObj.Name = &ifKey
			return nil
		} else {
			return fmt.Errorf("DbToYang_lacp_intfs_xfmr: PortChannel Instance doesn't exist: %s", ifKey)
		}
	case "/openconfig-lacp:lacp/interfaces/interface/members":
		if lacpintfObj, ok = lacpIntfsObj.Interfaces.Interface[ifKey]; !ok {
			return fmt.Errorf("DbToYang_lacp_intfs_xfmr: PortChannel Instance doesn't exist: %s", ifKey)
		}
		members = lacpintfObj.Members
		if members != nil && ifKey != "" {
			return populateLacpMembers(inParams, ifKey, members)
		}
	case "/openconfig-lacp:lacp/interfaces/interface/members/member":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state":
		if ifMemKey == "" || ifMemKey == "*" {
			if lacpintfObj, ok = lacpIntfsObj.Interfaces.Interface[ifKey]; !ok {
				return fmt.Errorf("DbToYang_lacp_intfs_xfmr: PortChannel Instance doesn't exist: %s", ifKey)
			}
			members = lacpintfObj.Members
			if members != nil && ifKey != "" {
				return populateLacpMembers(inParams, ifKey, members)
			}
		} else {
			member, err = prepareMemberObj(lacpIntfsObj, ifKey, ifMemKey)
			if err != nil {
				return err
			}
			return populateLacpMember(inParams, ifKey, ifMemKey, member)
		}
	default:
	}
	if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/members/member") {
		// Assume member attribute
		member, err = prepareMemberObj(lacpIntfsObj, ifKey, ifMemKey)
		if err != nil {
			return err
		}
		return populateLacpMemberAttr(inParams, ifKey, ifMemKey, targetUriPath, member)
	}
	if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/config") {
		lacpConfig, err := getLacpInterfaceConfigInfo(inParams, ifKey)
		if err != nil {
			return err
		}

		lacpIntfObj, ok := lacpIntfsObj.Interfaces.Interface[ifKey]
		if !ok {
			lacpIntfObj, _ = lacpIntfsObj.Interfaces.NewInterface(ifKey)
		}
		ygot.BuildEmptyTree(lacpIntfObj)

		log.V(3).Infof("DbToYang_lacp_intfs_xfmr: key=%s, data=%#v", ifKey, lacpConfig)
		return _fillLacpInterfaceConfig(ifKey, lacpConfig, lacpIntfObj.Config)
	}
	if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/state") {
		lacpintfObj, ok = lacpIntfsObj.Interfaces.Interface[ifKey]
		if !ok {
			lacpintfObj, _ = lacpIntfsObj.Interfaces.NewInterface(ifKey)
		}
		ygot.BuildEmptyTree(lacpintfObj)
		return populateLacpInterfaceState(inParams, ifKey, targetUriPath, lacpintfObj)
	}

	return nil
}

var Subscribe_lacp_intfs_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(3).Infof("Subscribe_lacp_intfs_xfmr:%s; template:%s targetUriPath:%s", pathInfo.Path, pathInfo.Template, targetUriPath)
	log.V(3).Infof("Subscribe_lacp_intfs_xfmr: inParams=%#v", inParams)

	ifName := pathInfo.Var("name")
	ifMemName := pathInfo.Var("interface")
	log.V(3).Infof("ifName %v, ifMemName %v", ifName, ifMemName)
	if ifName == "" {
		ifName = "*"
	}
	if ifMemName == "" {
		ifMemName = "*"
	}

	cfgDbMap := map[string]map[string]map[string]string{}
	sttDbMap := map[string]map[string]map[string]string{}
	result := XfmrSubscOutParams{
		dbDataMap:    RedisDbSubscribeMap{},
		needCache:    true,
		onChange:     OnchangeDisable,
		nOpts:        &notificationOpts{mInterval: 0, pType: Sample},
		isVirtualTbl: false}

	if inParams.subscProc == TRANSLATE_EXISTS {
		/* The parent table for all paths under openconfig-lacp/lacp/interfaces is the
		 * ConfigDB PORTCHANNEL table.  An entry in this table must exist to query any
		 * of the paths under interfaces. */
		if strings.Contains(targetUriPath, "config") {
			cfgDbMap[PORTCHANNEL_TN] = map[string]map[string]string{ifName: map[string]string{}}
			result.dbDataMap[db.ConfigDB] = cfgDbMap
		} else {
			log.V(3).Infof("Subscribe_lacp_intfs_xfmr: Unexpected path, inParams %#v", inParams)
		}
		return result, nil
		/* Optional: If we instead want to skip the resource-exists check done
		 * by the framework we can return "virtual table" and instead implement
		 * the resource-exists checks in our DbToYang and YangToDb transformers.
		 *return XfmrSubscOutParams{isVirtualTbl: true}, nil
		 */
	}

	/* verifyParentTblSubtree() invokes this callback with only the ancestor uri
	 * populated to check parent existence for GET/SET/DELETE, whereas a real
	 * gNMI Subscribe always carries the requestURI.  The only true parent for
	 * every path under /lacp/interfaces is the ConfigDB PORTCHANNEL entry;
	 * LAG_TABLE and LAG_MEMBER_TABLE are populated by teamd only once the LAG is
	 * up and has members, so demanding them here fails config requests on a
	 * memberless PortChannel. */
	if inParams.requestURI == "" {
		cfgDbMap[PORTCHANNEL_TN] = map[string]map[string]string{ifName: {}}
		result.dbDataMap[db.ConfigDB] = cfgDbMap
		return result, nil
	}

	var needPcCfg, needPcStt, needMbr bool

	/* Note that two leaves (interfaces/interface/state/system-priority and
	 * interface/members/member/partner-id) require on-change support. */
	fieldMapLagTbl := map[string]string{}
	fieldMapMbrTbl := map[string]string{}
	if strings.Contains(inParams.requestURI, "system-priority") {
		fieldMapLagTbl["runner.sys_prio"] = "system-priority"
		result.onChange = OnchangeEnable
		result.nOpts.pType = OnChange
		needPcStt = true
	}
	if strings.Contains(inParams.requestURI, "partner-id") {
		fieldMapMbrTbl["runner.partner_lacpdu_info.system"] = "partner-id"
		result.onChange = OnchangeEnable
		result.nOpts.pType = OnChange
		needMbr = true
	}

	/* Also, based on the container-level being queried include the required tables. */
	switch targetUriPath {
	case "/openconfig-lacp:lacp/interfaces":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface":
		needPcCfg = true
		needPcStt = true
		needMbr = true
	case "/openconfig-lacp:lacp/interfaces/interface/config":
		needPcCfg = true
	case "/openconfig-lacp:lacp/interfaces/interface/state":
		needPcStt = true
	case "/openconfig-lacp:lacp/interfaces/interface/members":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface/members/member":
		fallthrough
	case "/openconfig-lacp:lacp/interfaces/interface/members/member/state":
		needMbr = true
	default:
		if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/config") {
			needPcCfg = true
		} else if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/state") {
			needPcStt = true
		} else if strings.HasPrefix(targetUriPath, "/openconfig-lacp:lacp/interfaces/interface/members") {
			needMbr = true
		}
	}

	/* Populate the data returned through dbDataMap. */
	if needPcCfg {
		cfgDbMap[PORTCHANNEL_TN] = map[string]map[string]string{ifName: map[string]string{}}
		result.dbDataMap[db.ConfigDB] = cfgDbMap
	}
	if needPcStt {
		sttDbMap[LAG_TABLE_TN] = map[string]map[string]string{ifName: fieldMapLagTbl}
		result.dbDataMap[db.StateDB] = sttDbMap
	}
	if needMbr {
		sep := "|"
		if inParams.dbs[db.StateDB] != nil {
			sep = inParams.dbs[db.StateDB].Opts.KeySeparator
		}
		entryKey := ifName + sep + ifMemName
		sttDbMap[LAG_MEMBER_TABLE_TN] = map[string]map[string]string{entryKey: fieldMapMbrTbl}
		result.dbDataMap[db.StateDB] = sttDbMap
	}

	log.V(3).Infof("Subscribe_lacp_intfs_xfmr result: %#v", result)
	return result, nil
}

var YangToDb_lacp_intfs_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	log.V(3).Infof("YangToDb_lacp_intfs_xfmr: inParams=%#v", inParams)
	lacpObj := getLacpRoot(inParams.ygRoot)
	lacpIntfsObj := lacpObj.Interfaces

	if inParams.oper == DELETE {
		toDel := map[string]db.Value{}
		delSubOpMap := map[db.DBNum]map[string]map[string]db.Value{
			db.ConfigDB: {
				PORTCHANNEL_TN: toDel,
			},
		}
		for ifName, _ := range lacpIntfsObj.Interface {
			log.V(3).Infof("YangToDb_lacp_intfs_xfmr: Found key for removal: %s", ifName)
			cfgDb := inParams.dbs[db.ConfigDB]
			if cfgDb == nil {
				return nil, tlerr.InvalidArgsError{Format: "YangToDb_lacp_intfs_xfmr - ConfigDB is nil"}
			}
			pcEntry, err := cfgDb.GetEntry(&db.TableSpec{Name: PORTCHANNEL_TN}, db.Key{Comp: []string{ifName}})
			if err != nil {
				if tlerr.IsTranslibRedisClientEntryNotExist(err) {
					log.V(3).Infof("YangToDb_lacp_intfs_xfmr: key %s for removal not found in table %s", ifName, PORTCHANNEL_TN)
				} else {
					log.Errorf("YangToDb_lacp_intfs_xfmr: Error %v reading %s|%s", err, PORTCHANNEL_TN, ifName)
				}
				continue
			}
			val := db.Value{Field: map[string]string{}}
			for k, _ := range pcEntry.Field {
				if slices.Contains(PortChanFlds[:], k) {
					val.Field[k] = ""
				}
			}
			if len(val.Field) == len(pcEntry.Field) {
				/* Only openconfig-lacp managed fields in the entry, delete the entire key */
				toDel[ifName] = db.Value{}
			} else if len(val.Field) != 0 {
				/* Mix of field ownership, only delete the fields owned by openconfig-lacp */
				toDel[ifName] = val
			} else {
				/* No fields owned by openconfig-lacp, ignore this key */
				continue
			}
			log.V(3).Infof("YangToDb_lacp_intfs_xfmr: %s: %v", ifName, toDel[ifName])
		}
		if len(toDel) != 0 {
			updateSubOpDataMap(delSubOpMap, DELETE, inParams)
		}
		return nil, nil
	}
	/* Treating UPDATE and REPLACE the same here. */
	for ifName, lacpIntfObj := range lacpIntfsObj.Interface {
		log.V(3).Infof("YangToDb_lacp_intfs_xfmr: interface[%s]=%#v", ifName, lacpIntfObj)

		intfType, _, err := getIntfTypeByName(ifName)
		if err != nil || intfType == IntfTypeUnset {
			return nil, tlerr.InvalidArgsError{Format: "Invalid interface: " + ifName}
		}
		if intfType != IntfTypePortChannel {
			return nil, errors.New("YangToDb_lacp_intfs_xfmr, Error: Expected IntfTypePortChannel interface type: " + ifName)
		}

		resMap := make(map[string]string)
		var interval ocbinds.E_OpenconfigLacp_LacpPeriodType
		var systemIdMac *string
		var fallback *bool

		if lacpIntfObj.Config != nil {
			log.V(3).Infof("SET data: %#v", lacpIntfObj.Config)
			interval = lacpIntfObj.Config.Interval
			systemIdMac = lacpIntfObj.Config.SystemIdMac
			fallback = lacpIntfObj.Config.Fallback
		}

		switch {
		case strings.HasSuffix(inParams.requestUri, "interval"):
			fast_rate := "true"
			if interval == ocbinds.OpenconfigLacp_LacpPeriodType_SLOW {
				fast_rate = "false"
			}
			resMap["fast_rate"] = fast_rate

		case strings.HasSuffix(inParams.requestUri, "system-id-mac"):
			if systemIdMac == nil {
				return nil, tlerr.InvalidArgsError{Format: "Config SystemIdMac doesn't exist"}
			}
			resMap["system_mac"] = *systemIdMac

		case strings.HasSuffix(inParams.requestUri, "fallback"):
			if fallback != nil {
				resMap["fallback"] = strconv.FormatBool(*fallback)
			}
		default:
			use_fast_rate := true
			var fast_rate string
			switch interval {
			case ocbinds.OpenconfigLacp_LacpPeriodType_UNSET:
				use_fast_rate = (inParams.oper == REPLACE)
				fallthrough
			case ocbinds.OpenconfigLacp_LacpPeriodType_SLOW:
				fast_rate = "false"
			case ocbinds.OpenconfigLacp_LacpPeriodType_FAST:
				fast_rate = "true"
			}
			if use_fast_rate {
				resMap["fast_rate"] = fast_rate
			}

			if systemIdMac != nil {
				resMap["system_mac"] = *systemIdMac
			}
			if fallback != nil {
				resMap["fallback"] = strconv.FormatBool(*fallback)
			}
		}

		intTbl, ok := IntfTypeTblMap[intfType]
		if !ok {
			log.V(3).Infof("YangToDb_lacp_intfs_xfmr intTbl not found : %v", intfType)
			return nil, errors.New("intTbl not found.")
		}
		subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
		memMap := map[string]map[string]db.Value{
			intTbl.cfgDb.portTN: map[string]db.Value{
				ifName: db.Value{Field: resMap},
			},
		}
		log.V(3).Infof("YangToDb_lacp_intfs_xfmr: %s|%s resMap=%v", intTbl.cfgDb.portTN, ifName, resMap)
		subOpMap[db.ConfigDB] = memMap
		updateSubOpDataMap(subOpMap, UPDATE, inParams)
	}
	return nil, nil
}
