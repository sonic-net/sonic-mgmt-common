////////////////////////////////////////////////////////////////////////////////
//                                                                            //
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
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	ocbinds "github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	VRRP_TBL                                    = "VRRP"
	VRRP6_TBL                                   = "VRRP6"
	VRRP_TRACK_TBL                              = "VRRP_TRACK"
	VRRP6_TRACK_TBL                             = "VRRP6_TRACK"
	VRRP_STATE_TBL                              = "VRRP_STATE_TABLE"
	VRRP6_STATE_TBL                             = "VRRP6_STATE_TABLE"
	VRRP_FIELD_VRID                             = "vrid"
	VRRP_FIELD_VIP                              = "vip@"
	VRRP_FIELD_PRIORITY                         = "priority"
	VRRP_FIELD_PREEMPT                          = "preempt"
	VRRP_FIELD_PREEMPT_DELAY                    = "preempt_delay"
	VRRP_FIELD_ADV_INTERVAL                     = "adv_interval"
	VRRP_FIELD_TRACK_PRIO_INCR                  = "priority_increment"
	VRRP_STATE_FIELD_EFF_PRIORITY               = "effective_priority"
	VRRP_STATE_FIELD_STATE                      = "state"
	VRRP_STATE_FIELD_ADVERTISEMENTS_TX          = "advertisements_tx"
	VRRP_STATE_FIELD_ADVERTISEMENTS_RX          = "advertisements_rx"
	VRRP_STATE_FIELD_GRATUITOUS_ARP_TX          = "gratuitous_arp_tx"
	VRRP_STATE_FIELD_NEIGHBOR_ADVERTISEMENTS_TX = "neighbor_advertisements_tx"
	VRRP_STATE_FIELD_TRANSITIONS                = "state_transitions"
)

type vrrpGroupData struct {
	vrid                       uint8
	virtualAddresses           []string
	priority                   uint8
	preempt                    bool
	preemptDelay               uint16
	advertisementCenti         uint16
	trackInterfaces            []string
	priorityDecrement          uint8
	currentPriority            uint8
	hasCurrentPriority         bool
	vrrpState                  ocbinds.E_OpenconfigIfIpVrrpExt_VrrpStateType
	hasVrrpState               bool
	advertisementsSent         *uint64
	advertisementsReceived     *uint64
	gratuitousArpSent          *uint64
	neighborAdvertisementsSent *uint64
	stateTransitions           *uint64
}

func parseVrrpState(value string) (ocbinds.E_OpenconfigIfIpVrrpExt_VrrpStateType, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "INITIALIZE":
		return ocbinds.OpenconfigIfIpVrrpExt_VrrpStateType_INITIALIZE, true
	case "BACKUP":
		return ocbinds.OpenconfigIfIpVrrpExt_VrrpStateType_BACKUP, true
	case "MASTER":
		return ocbinds.OpenconfigIfIpVrrpExt_VrrpStateType_MASTER, true
	default:
		return ocbinds.OpenconfigIfIpVrrpExt_VrrpStateType_UNSET, false
	}
}

func parseVrrpCounter(entry db.Value, field, stateTbl string, keyComp []string) *uint64 {
	value := strings.TrimSpace(entry.Get(field))
	if value == "" {
		return nil
	}
	counter, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		log.Warningf("VRRP state table=%s key=%v has unusable %s=%q",
			stateTbl, keyComp, field, value)
		return nil
	}
	return &counter
}

func init() {
	XlateFuncBind("DbToYang_routed_vlan_ipv4_vrrp_xfmr", DbToYang_routed_vlan_ipv4_vrrp_xfmr)
	XlateFuncBind("DbToYang_routed_vlan_ipv6_vrrp_xfmr", DbToYang_routed_vlan_ipv6_vrrp_xfmr)
	XlateFuncBind("YangToDb_routed_vlan_ipv4_vrrp_xfmr", YangToDb_routed_vlan_ipv4_vrrp_xfmr)
	XlateFuncBind("YangToDb_routed_vlan_ipv6_vrrp_xfmr", YangToDb_routed_vlan_ipv6_vrrp_xfmr)
	XlateFuncBind("Subscribe_routed_vlan_ipv4_vrrp_xfmr", Subscribe_routed_vlan_ipv4_vrrp_xfmr)
	XlateFuncBind("Subscribe_routed_vlan_ipv6_vrrp_xfmr", Subscribe_routed_vlan_ipv6_vrrp_xfmr)
	XlateFuncBind("DbToYang_subintf_ipv4_vrrp_xfmr", DbToYang_subintf_ipv4_vrrp_xfmr)
	XlateFuncBind("DbToYang_subintf_ipv6_vrrp_xfmr", DbToYang_subintf_ipv6_vrrp_xfmr)
	XlateFuncBind("YangToDb_subintf_ipv4_vrrp_xfmr", YangToDb_subintf_ipv4_vrrp_xfmr)
	XlateFuncBind("YangToDb_subintf_ipv6_vrrp_xfmr", YangToDb_subintf_ipv6_vrrp_xfmr)
	XlateFuncBind("Subscribe_subintf_ipv4_vrrp_xfmr", Subscribe_subintf_ipv4_vrrp_xfmr)
	XlateFuncBind("Subscribe_subintf_ipv6_vrrp_xfmr", Subscribe_subintf_ipv6_vrrp_xfmr)
	XlateFuncBind("YangToDb_vrrp_group_key_xfmr", YangToDb_vrrp_group_key_xfmr)
	XlateFuncBind("DbToYang_vrrp_group_key_xfmr", DbToYang_vrrp_group_key_xfmr)
}

var YangToDb_vrrp_group_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := getVrrpKeyPathInfo(inParams)
	ifName := strings.TrimSpace(pathInfo.Var("name"))
	log.Warningf("VRRP_TRACE key_xfmr Y2D: uri=%s requestUri=%s keyIn=%s name=%s vrid=%s",
		inParams.uri, inParams.requestUri, inParams.key, ifName, strings.TrimSpace(pathInfo.Var("virtual-router-id")))
	if vrid := strings.TrimSpace(pathInfo.Var("virtual-router-id")); vrid != "" {
		if ifName != "" {
			log.Warningf("VRRP_TRACE key_xfmr Y2D: return=%s|%s source=path", ifName, vrid)
			return ifName + "|" + vrid, nil
		}
		if keyIfName := extractIfNameFromDbKey(inParams.key); keyIfName != "" {
			log.Warningf("VRRP_TRACE key_xfmr Y2D: return=%s|%s source=keyFallback", keyIfName, vrid)
			return keyIfName + "|" + vrid, nil
		}
		log.Warningf("VRRP_TRACE key_xfmr Y2D: return=%s source=vridOnly", vrid)
		return vrid, nil
	}
	if ifName == "" {
		ifName = extractIfNameFromDbKey(inParams.key)
	}
	if ifName != "" {
		if vrid := extractVridFromDbKey(inParams.key); vrid != "" {
			log.Warningf("VRRP_TRACE key_xfmr Y2D: return=%s|%s source=keySplit", ifName, vrid)
			return ifName + "|" + vrid, nil
		}
	}
	log.Warningf("VRRP_TRACE key_xfmr Y2D: return=%s source=finalVridOnly", extractVridFromDbKey(inParams.key))
	return extractVridFromDbKey(inParams.key), nil
}

var DbToYang_vrrp_group_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	pathInfo := getVrrpKeyPathInfo(inParams)
	vrid := pathInfo.Var("virtual-router-id")
	if vrid == "" {
		vrid = extractVridFromDbKey(inParams.key)
	}
	log.Warningf("VRRP_TRACE key_xfmr D2Y: uri=%s requestUri=%s keyIn=%s resolvedVrid=%s", inParams.uri, inParams.requestUri, inParams.key, vrid)
	if vrid == "" {
		return res, nil
	}
	if val, err := strconv.ParseUint(vrid, 10, 8); err == nil {
		res["virtual-router-id"] = uint8(val)
	} else {
		res["virtual-router-id"] = vrid
	}
	return res, nil
}

func extractVridFromDbKey(dbKey string) string {
	if dbKey == "" {
		return ""
	}
	parts := strings.Split(dbKey, "|")
	if len(parts) < 1 {
		return ""
	}
	return parts[len(parts)-1]
}

func extractIfNameFromDbKey(dbKey string) string {
	if dbKey == "" {
		return ""
	}
	parts := strings.Split(dbKey, "|")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func getVrrpKeyPathInfo(inParams XfmrParams) *PathInfo {
	pathInfo := NewPathInfo(inParams.uri)
	if inParams.requestUri == "" || inParams.requestUri == inParams.uri {
		return pathInfo
	}
	reqPathInfo := NewPathInfo(inParams.requestUri)
	if reqPathInfo.Var("name") == "" {
		return pathInfo
	}
	if pathInfo.Var("name") == "" {
		return reqPathInfo
	}
	// Prefer requestUri when it carries a more specific keyed target than callback uri.
	if (pathInfo.Var("ip") == "" && reqPathInfo.Var("ip") != "") ||
		(pathInfo.Var("index") == "" && reqPathInfo.Var("index") != "") ||
		(pathInfo.Var("virtual-router-id") == "" && reqPathInfo.Var("virtual-router-id") != "") {
		return reqPathInfo
	}
	if isMoreSpecificPath(reqPathInfo.YangPath, pathInfo.YangPath) {
		return reqPathInfo
	}
	return pathInfo
}

func getTargetVridForRead(pathInfo *PathInfo, dbKey string) string {
	if pathInfo != nil {
		if vrid := strings.TrimSpace(pathInfo.Var("virtual-router-id")); vrid != "" {
			return vrid
		}
	}
	return extractVridFromDbKey(dbKey)
}

var DbToYang_routed_vlan_ipv4_vrrp_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	return dbToYangIPv4Vrrp(inParams)
}

var DbToYang_routed_vlan_ipv6_vrrp_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	return dbToYangIPv6Vrrp(inParams)
}

var YangToDb_routed_vlan_ipv4_vrrp_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	return yangToDbRoutedVlanVrrp(inParams, false)
}

var YangToDb_routed_vlan_ipv6_vrrp_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	return yangToDbRoutedVlanVrrp(inParams, true)
}

var DbToYang_subintf_ipv4_vrrp_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	return dbToYangSubIntfVrrp(inParams, false)
}

var DbToYang_subintf_ipv6_vrrp_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	return dbToYangSubIntfVrrp(inParams, true)
}

var YangToDb_subintf_ipv4_vrrp_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	return yangToDbSubIntfVrrp(inParams, false)
}

var YangToDb_subintf_ipv6_vrrp_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	return yangToDbSubIntfVrrp(inParams, true)
}

var Subscribe_routed_vlan_ipv4_vrrp_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return subscribeVrrp(inParams, false)
}

var Subscribe_routed_vlan_ipv6_vrrp_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return subscribeVrrp(inParams, true)
}

var Subscribe_subintf_ipv4_vrrp_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return subscribeVrrp(inParams, false)
}

var Subscribe_subintf_ipv6_vrrp_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return subscribeVrrp(inParams, true)
}

func subscribeVrrp(inParams XfmrSubscInParams, isIPv6 bool) (XfmrSubscOutParams, error) {
	result := XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{},
	}

	pathInfo := NewPathInfo(inParams.uri)
	ifName := strings.TrimSpace(pathInfo.StringVar("name", "*"))
	vrid := strings.TrimSpace(pathInfo.Var("virtual-router-id"))
	if vrid == "" {
		vrid = "*"
	}

	tbl := VRRP_TBL
	stateTbl := VRRP_STATE_TBL
	if isIPv6 {
		tbl = VRRP6_TBL
		stateTbl = VRRP6_STATE_TBL
	}
	key := ifName + "|" + vrid

	// For RFC parent-table checks in GET/CRUD flows, provide concrete table/key mapping.
	// For subscription flows, wildcard key still allows expansion downstream.
	// current-priority to be watched for failover cases
	result.dbDataMap = RedisDbSubscribeMap{
		db.ConfigDB: {
			tbl: {
				key: {},
			},
		},
		db.StateDB: {
			stateTbl: {
				key: {},
			},
		},
	}

	log.Warningf("VRRP_TRACE subscribeVrrp: subscProc=%v uri=%s table=%s stateTable=%s key=%s isIPv6=%v", inParams.subscProc, inParams.uri, tbl, stateTbl, key, isIPv6)
	return result, nil
}

func yangToDbRoutedVlanVrrp(inParams XfmrParams, isIPv6 bool) (map[string]map[string]db.Value, error) {
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}

	pathInfo := getVrrpPathInfo(inParams)
	ifName := pathInfo.Var("name")
	log.Warningf("VRRP_TRACE yangToDbRoutedVlanVrrp: op=%v uri=%s requestUri=%s yangPath=%s ifName=%s isIPv6=%v paramType=%T",
		inParams.oper, inParams.uri, inParams.requestUri, pathInfo.YangPath, ifName, isIPv6, inParams.param)
	if ifName == "" {
		return map[string]map[string]db.Value{}, nil
	}
	if err := validateDenyVrrpReplaceVirtualRouterIdLeaf(inParams, pathInfo); err != nil {
		return nil, err
	}
	uriIP := pathInfo.Var("ip")
	if err := validateVrrpAddressKey(inParams.dbs[db.ConfigDB], ifName, uriIP, isIPv6); err != nil {
		return nil, err
	}

	vrrpTbl := VRRP_TBL
	vrrpTrackTbl := VRRP_TRACK_TBL
	if isIPv6 {
		vrrpTbl = VRRP6_TBL
		vrrpTrackTbl = VRRP6_TRACK_TBL
	}

	resMap := map[string]map[string]db.Value{
		vrrpTbl: {},
	}

	if inParams.oper == DELETE {
		vrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
		if vrid == 0 {
			return resMap, nil
		}
		if handled, leafDeleteMap := buildVrrpLeafDeleteMap(pathInfo, ifName, vrid, isIPv6, vrrpTbl, vrrpTrackTbl, inParams.dbs[db.ConfigDB]); handled {
			return leafDeleteMap, nil
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
		resMap[vrrpTbl][vrrpKey] = db.Value{}
		if cfgDb := inParams.dbs[db.ConfigDB]; cfgDb != nil {
			trackMap := getTrackDeleteMap(cfgDb, vrrpTrackTbl, ifName, vrid)
			if len(trackMap) > 0 {
				resMap[vrrpTrackTbl] = trackMap
			}
		}
		return resMap, nil
	}

	if isIPv6 {
		targetVrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
		grpObj, grpVrid, ok := getRoutedVlanIPv6VrrpGroupFromParam(inParams.param, targetVrid)
		if !ok {
			grpObj, grpVrid, ok = getRoutedVlanIPv6VrrpGroupFromRoot(inParams, ifName, uriIP, targetVrid)
		}
		if !ok {
			leafVrid, leafFields, leafTrackEntries, leafHandled := buildVrrpLeafSetMap(pathInfo, inParams.param, ifName, targetVrid, true, vrrpTrackTbl, inParams.dbs[db.ConfigDB])
			if !leafHandled || leafVrid == 0 {
				return resMap, nil
			}
			if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], ifName, leafFields); err != nil {
				return nil, err
			}
			vrrpKey := fmt.Sprintf("%s|%d", ifName, leafVrid)
			if len(leafFields) > 0 {
				leafFields[VRRP_FIELD_VRID] = strconv.Itoa(int(leafVrid))
				resMap[vrrpTbl][vrrpKey] = db.Value{Field: leafFields}
			}
			if len(leafTrackEntries) > 0 {
				resMap[vrrpTrackTbl] = leafTrackEntries
			}
			return resMap, nil
		}
		var cfgVrid *uint8
		if grpObj.Config != nil {
			cfgVrid = grpObj.Config.VirtualRouterId
		}
		if err := validateVrrpPayloadVrid(inParams.oper, targetVrid, grpObj.VirtualRouterId, cfgVrid); err != nil {
			return nil, err
		}
		vrid, fields, trackEntries := mapRoutedVlanIPv6VrrpGroupToDbFields(grpObj, grpVrid, ifName, vrrpTrackTbl, inParams.dbs[db.ConfigDB], pathInfo)
		if vrid == 0 {
			return resMap, nil
		}
		if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], ifName, fields); err != nil {
			return nil, err
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
		if len(fields) > 0 {
			fields[VRRP_FIELD_VRID] = strconv.Itoa(int(vrid))
			resMap[vrrpTbl][vrrpKey] = db.Value{Field: fields}
		}
		if len(trackEntries) > 0 {
			resMap[vrrpTrackTbl] = trackEntries
		}
		return resMap, nil
	}

	targetVrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
	grpObj, grpVrid, ok := getRoutedVlanIPv4VrrpGroupFromParam(inParams.param, targetVrid)
	if !ok {
		grpObj, grpVrid, ok = getRoutedVlanIPv4VrrpGroupFromRoot(inParams, ifName, uriIP, targetVrid)
	}
	if !ok {
		leafVrid, leafFields, leafTrackEntries, leafHandled := buildVrrpLeafSetMap(pathInfo, inParams.param, ifName, targetVrid, false, vrrpTrackTbl, inParams.dbs[db.ConfigDB])
		if !leafHandled || leafVrid == 0 {
			return resMap, nil
		}
		if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], ifName, leafFields); err != nil {
			return nil, err
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, leafVrid)
		if len(leafFields) > 0 {
			leafFields[VRRP_FIELD_VRID] = strconv.Itoa(int(leafVrid))
			resMap[vrrpTbl][vrrpKey] = db.Value{Field: leafFields}
		}
		if len(leafTrackEntries) > 0 {
			resMap[vrrpTrackTbl] = leafTrackEntries
		}
		return resMap, nil
	}
	var cfgVrid *uint8
	if grpObj.Config != nil {
		cfgVrid = grpObj.Config.VirtualRouterId
	}
	if err := validateVrrpPayloadVrid(inParams.oper, targetVrid, grpObj.VirtualRouterId, cfgVrid); err != nil {
		return nil, err
	}
	vrid, fields, trackEntries := mapRoutedVlanIPv4VrrpGroupToDbFields(grpObj, grpVrid, ifName, vrrpTrackTbl, inParams.dbs[db.ConfigDB], pathInfo)
	if vrid == 0 {
		return resMap, nil
	}
	if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], ifName, fields); err != nil {
		return nil, err
	}
	vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
	if len(fields) > 0 {
		fields[VRRP_FIELD_VRID] = strconv.Itoa(int(vrid))
		resMap[vrrpTbl][vrrpKey] = db.Value{Field: fields}
	}
	if len(trackEntries) > 0 {
		resMap[vrrpTrackTbl] = trackEntries
	}
	return resMap, nil
}

func yangToDbSubIntfVrrp(inParams XfmrParams, isIPv6 bool) (map[string]map[string]db.Value, error) {
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}

	pathInfo := getVrrpPathInfo(inParams)
	ifName := pathInfo.Var("name")
	log.Warningf("VRRP_TRACE yangToDbSubIntfVrrp: op=%v uri=%s requestUri=%s yangPath=%s ifName=%s index=%s isIPv6=%v paramType=%T",
		inParams.oper, inParams.uri, inParams.requestUri, pathInfo.YangPath, ifName, pathInfo.Var("index"), isIPv6, inParams.param)
	if ifName == "" {
		return map[string]map[string]db.Value{}, nil
	}
	if err := validateDenyVrrpReplaceVirtualRouterIdLeaf(inParams, pathInfo); err != nil {
		return nil, err
	}
	subIntfIdx := parseSubIntfIndex(pathInfo.Var("index"))
	parentIfName := ifName
	if subIntfIdx != 0 {
		parentIfName = fmt.Sprintf("%s.%d", ifName, subIntfIdx)
	}
	uriIP := pathInfo.Var("ip")
	if err := validateVrrpAddressKey(inParams.dbs[db.ConfigDB], parentIfName, uriIP, isIPv6); err != nil {
		return nil, err
	}

	vrrpTbl := VRRP_TBL
	vrrpTrackTbl := VRRP_TRACK_TBL
	if isIPv6 {
		vrrpTbl = VRRP6_TBL
		vrrpTrackTbl = VRRP6_TRACK_TBL
	}

	resMap := map[string]map[string]db.Value{
		vrrpTbl: {},
	}

	// Handle delete by removing matching key(s) under the requested VRID.
	if inParams.oper == DELETE {
		vrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
		if vrid == 0 {
			return resMap, nil
		}
		if handled, leafDeleteMap := buildVrrpLeafDeleteMap(pathInfo, ifName, vrid, isIPv6, vrrpTbl, vrrpTrackTbl, inParams.dbs[db.ConfigDB]); handled {
			return leafDeleteMap, nil
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
		resMap[vrrpTbl][vrrpKey] = db.Value{}
		if cfgDb := inParams.dbs[db.ConfigDB]; cfgDb != nil {
			trackMap := getTrackDeleteMap(cfgDb, vrrpTrackTbl, ifName, vrid)
			if len(trackMap) > 0 {
				resMap[vrrpTrackTbl] = trackMap
			}
		}
		return resMap, nil
	}

	if isIPv6 {
		targetVrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
		grpObj, grpVrid, ok := getSubIntfIPv6VrrpGroupFromParam(inParams.param, targetVrid)
		if !ok {
			grpObj, grpVrid, ok = getSubIntfIPv6VrrpGroupFromRoot(inParams, ifName, subIntfIdx, uriIP, targetVrid)
		}
		if !ok {
			leafVrid, leafFields, leafTrackEntries, leafHandled := buildVrrpLeafSetMap(pathInfo, inParams.param, ifName, targetVrid, true, vrrpTrackTbl, inParams.dbs[db.ConfigDB])
			if !leafHandled || leafVrid == 0 {
				return resMap, nil
			}
			if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], parentIfName, leafFields); err != nil {
				return nil, err
			}
			vrrpKey := fmt.Sprintf("%s|%d", ifName, leafVrid)
			if len(leafFields) > 0 {
				leafFields[VRRP_FIELD_VRID] = strconv.Itoa(int(leafVrid))
				resMap[vrrpTbl][vrrpKey] = db.Value{Field: leafFields}
			}
			if len(leafTrackEntries) > 0 {
				resMap[vrrpTrackTbl] = leafTrackEntries
			}
			return resMap, nil
		}
		var cfgVrid *uint8
		if grpObj.Config != nil {
			cfgVrid = grpObj.Config.VirtualRouterId
		}
		if err := validateVrrpPayloadVrid(inParams.oper, targetVrid, grpObj.VirtualRouterId, cfgVrid); err != nil {
			return nil, err
		}
		vrid, fields, trackEntries := mapSubIntfIPv6VrrpGroupToDbFields(grpObj, grpVrid, ifName, vrrpTrackTbl, inParams.dbs[db.ConfigDB], pathInfo)
		if vrid == 0 {
			return resMap, nil
		}
		if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], parentIfName, fields); err != nil {
			return nil, err
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
		if len(fields) > 0 {
			fields[VRRP_FIELD_VRID] = strconv.Itoa(int(vrid))
			resMap[vrrpTbl][vrrpKey] = db.Value{Field: fields}
		}
		if len(trackEntries) > 0 {
			resMap[vrrpTrackTbl] = trackEntries
		}
		return resMap, nil
	}

	targetVrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
	grpObj, grpVrid, ok := getSubIntfIPv4VrrpGroupFromParam(inParams.param, targetVrid)
	if !ok {
		grpObj, grpVrid, ok = getSubIntfIPv4VrrpGroupFromRoot(inParams, ifName, subIntfIdx, uriIP, targetVrid)
	}
	if !ok {
		leafVrid, leafFields, leafTrackEntries, leafHandled := buildVrrpLeafSetMap(pathInfo, inParams.param, ifName, targetVrid, false, vrrpTrackTbl, inParams.dbs[db.ConfigDB])
		if !leafHandled || leafVrid == 0 {
			return resMap, nil
		}
		if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], parentIfName, leafFields); err != nil {
			return nil, err
		}
		vrrpKey := fmt.Sprintf("%s|%d", ifName, leafVrid)
		if len(leafFields) > 0 {
			leafFields[VRRP_FIELD_VRID] = strconv.Itoa(int(leafVrid))
			resMap[vrrpTbl][vrrpKey] = db.Value{Field: leafFields}
		}
		if len(leafTrackEntries) > 0 {
			resMap[vrrpTrackTbl] = leafTrackEntries
		}
		return resMap, nil
	}
	var cfgVrid *uint8
	if grpObj.Config != nil {
		cfgVrid = grpObj.Config.VirtualRouterId
	}
	if err := validateVrrpPayloadVrid(inParams.oper, targetVrid, grpObj.VirtualRouterId, cfgVrid); err != nil {
		return nil, err
	}
	vrid, fields, trackEntries := mapSubIntfIPv4VrrpGroupToDbFields(grpObj, grpVrid, ifName, vrrpTrackTbl, inParams.dbs[db.ConfigDB], pathInfo)
	if vrid == 0 {
		return resMap, nil
	}
	if err := validateVrrpFieldsVipSubnet(inParams.dbs[db.ConfigDB], parentIfName, fields); err != nil {
		return nil, err
	}
	vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
	if len(fields) > 0 {
		fields[VRRP_FIELD_VRID] = strconv.Itoa(int(vrid))
		resMap[vrrpTbl][vrrpKey] = db.Value{Field: fields}
	}
	if len(trackEntries) > 0 {
		resMap[vrrpTrackTbl] = trackEntries
	}
	return resMap, nil
}

func getVrrpPathInfo(inParams XfmrParams) *PathInfo {
	pathInfo := NewPathInfo(inParams.uri)
	if inParams.requestUri == "" || inParams.requestUri == inParams.uri {
		return pathInfo
	}
	reqPathInfo := NewPathInfo(inParams.requestUri)
	if reqPathInfo.Var("name") != "" && pathInfo.Var("name") != "" && isMoreSpecificPath(reqPathInfo.YangPath, pathInfo.YangPath) {
		log.Warningf("VRRP_TRACE getVrrpPathInfo: prefer requestUri path=%s over uri path=%s", reqPathInfo.YangPath, pathInfo.YangPath)
		return reqPathInfo
	}
	if pathInfo.Var("name") != "" {
		return pathInfo
	}
	if reqPathInfo.Var("name") != "" {
		return reqPathInfo
	}
	return pathInfo
}

func isVrrpVirtualRouterIdLeafPath(pathInfo *PathInfo) bool {
	if pathInfo == nil {
		return false
	}
	return strings.HasSuffix(normalizedYangPath(pathInfo.YangPath), "/vrrp-group/config/virtual-router-id")
}

func validateVrrpPayloadVrid(oper Operation, uriVrid uint8, payloadVrids ...*uint8) error {
	if oper == DELETE || uriVrid == 0 {
		return nil
	}
	for _, p := range payloadVrids {
		if p != nil && *p != uriVrid {
			return tlerr.InvalidArgs("virtual-router-id %d in the payload does not match the VRRP group key %d.", *p, uriVrid)
		}
	}
	return nil
}

func validateDenyVrrpReplaceVirtualRouterIdLeaf(inParams XfmrParams, pathInfo *PathInfo) error {
	switch inParams.oper {
	case REPLACE:
		if isVrrpVirtualRouterIdLeafPath(pathInfo) {
			return tlerr.NotSupportedError{Format: "The virtual-router-id cannot be modified directly, delete the existing VRRP group and recreate it with the new virtual-router-id."}
		}
		if inParams.requestUri != "" && inParams.requestUri != inParams.uri {
			reqPathInfo := NewPathInfo(inParams.requestUri)
			if isVrrpVirtualRouterIdLeafPath(reqPathInfo) {
				return tlerr.NotSupportedError{Format: "The virtual-router-id cannot be modified directly, delete the existing VRRP group and recreate it with the new virtual-router-id."}
			}
		}
	case UPDATE, CREATE:
		if !isVrrpVirtualRouterIdLeafPath(pathInfo) {
			return nil
		}
		uriVrid := parseUint8(pathInfo.Var("virtual-router-id"), 0)
		if val, ok := toUint64Param(inParams.param); ok && uriVrid != 0 && val != uint64(uriVrid) {
			return tlerr.InvalidArgs("virtual-router-id %d in the payload does not match the VRRP group key %d.", val, uriVrid)
		}
	}
	return nil
}

func getRoutedVlanIPv4VrrpGroupFromRoot(inParams XfmrParams, ifName string, uriIP string, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || intfsObj.Interface == nil {
		return nil, 0, false
	}
	intfObj, ok := intfsObj.Interface[ifName]
	if !ok || intfObj == nil || intfObj.RoutedVlan == nil || intfObj.RoutedVlan.Ipv4 == nil ||
		intfObj.RoutedVlan.Ipv4.Addresses == nil || intfObj.RoutedVlan.Ipv4.Addresses.Address == nil {
		return nil, 0, false
	}
	addrObj, ok := intfObj.RoutedVlan.Ipv4.Addresses.Address[uriIP]
	if uriIP != "" && uriIP != "*" && (!ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil) {
		return nil, 0, false
	}
	if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
		for _, cand := range intfObj.RoutedVlan.Ipv4.Addresses.Address {
			if cand != nil && cand.Vrrp != nil && cand.Vrrp.VrrpGroup != nil {
				addrObj = cand
				ok = true
				break
			}
		}
		if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
			return nil, 0, false
		}
	}
	if targetVrid != 0 {
		grpObj, ok := addrObj.Vrrp.VrrpGroup[targetVrid]
		return grpObj, targetVrid, ok && grpObj != nil
	}
	for vrid, grpObj := range addrObj.Vrrp.VrrpGroup {
		if grpObj != nil {
			return grpObj, vrid, true
		}
	}
	return nil, 0, false
}

func getRoutedVlanIPv6VrrpGroupFromRoot(inParams XfmrParams, ifName string, uriIP string, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || intfsObj.Interface == nil {
		return nil, 0, false
	}
	intfObj, ok := intfsObj.Interface[ifName]
	if !ok || intfObj == nil || intfObj.RoutedVlan == nil || intfObj.RoutedVlan.Ipv6 == nil ||
		intfObj.RoutedVlan.Ipv6.Addresses == nil || intfObj.RoutedVlan.Ipv6.Addresses.Address == nil {
		return nil, 0, false
	}
	addrObj, ok := intfObj.RoutedVlan.Ipv6.Addresses.Address[uriIP]
	if uriIP != "" && uriIP != "*" && (!ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil) {
		return nil, 0, false
	}
	if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
		for _, cand := range intfObj.RoutedVlan.Ipv6.Addresses.Address {
			if cand != nil && cand.Vrrp != nil && cand.Vrrp.VrrpGroup != nil {
				addrObj = cand
				ok = true
				break
			}
		}
		if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
			return nil, 0, false
		}
	}
	if targetVrid != 0 {
		grpObj, ok := addrObj.Vrrp.VrrpGroup[targetVrid]
		return grpObj, targetVrid, ok && grpObj != nil
	}
	for vrid, grpObj := range addrObj.Vrrp.VrrpGroup {
		if grpObj != nil {
			return grpObj, vrid, true
		}
	}
	return nil, 0, false
}

func getRoutedVlanIPv4VrrpGroupFromParam(param interface{}, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	switch p := param.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address:
		if p != nil && p.Vrrp != nil {
			return getRoutedVlanIPv4VrrpGroupFromParam(p.Vrrp, targetVrid)
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp:
		if p == nil || p.VrrpGroup == nil {
			return nil, 0, false
		}
		if targetVrid != 0 {
			grpObj, ok := p.VrrpGroup[targetVrid]
			return grpObj, targetVrid, ok && grpObj != nil
		}
		for vrid, grpObj := range p.VrrpGroup {
			if grpObj != nil {
				return grpObj, vrid, true
			}
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup:
		if p != nil {
			return p, resolveVrid("", p.VirtualRouterId, p.Config, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup{Config: p}, resolveVrid("", nil, p, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup{InterfaceTracking: p}, targetVrid, true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup{
				InterfaceTracking: &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking{Config: p},
			}, targetVrid, true
		}
	}
	return nil, 0, false
}

func getRoutedVlanIPv6VrrpGroupFromParam(param interface{}, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	switch p := param.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address:
		if p != nil && p.Vrrp != nil {
			return getRoutedVlanIPv6VrrpGroupFromParam(p.Vrrp, targetVrid)
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp:
		if p == nil || p.VrrpGroup == nil {
			return nil, 0, false
		}
		if targetVrid != 0 {
			grpObj, ok := p.VrrpGroup[targetVrid]
			return grpObj, targetVrid, ok && grpObj != nil
		}
		for vrid, grpObj := range p.VrrpGroup {
			if grpObj != nil {
				return grpObj, vrid, true
			}
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup:
		if p != nil {
			return p, resolveVrid("", p.VirtualRouterId, p.Config, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup{Config: p}, resolveVrid("", nil, p, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup{InterfaceTracking: p}, targetVrid, true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup{
				InterfaceTracking: &ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking{Config: p},
			}, targetVrid, true
		}
	}
	return nil, 0, false
}

func getSubIntfIPv4VrrpGroupFromRoot(inParams XfmrParams, ifName string, subIntfIdx uint32, uriIP string, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || intfsObj.Interface == nil {
		return nil, 0, false
	}
	intfObj, ok := intfsObj.Interface[ifName]
	if !ok || intfObj == nil || intfObj.Subinterfaces == nil || intfObj.Subinterfaces.Subinterface == nil {
		return nil, 0, false
	}
	subIntfObj, ok := intfObj.Subinterfaces.Subinterface[subIntfIdx]
	if !ok || subIntfObj == nil || subIntfObj.Ipv4 == nil || subIntfObj.Ipv4.Addresses == nil || subIntfObj.Ipv4.Addresses.Address == nil {
		return nil, 0, false
	}
	addrObj, ok := subIntfObj.Ipv4.Addresses.Address[uriIP]
	if uriIP != "" && uriIP != "*" && (!ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil) {
		return nil, 0, false
	}
	if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
		for _, cand := range subIntfObj.Ipv4.Addresses.Address {
			if cand != nil && cand.Vrrp != nil && cand.Vrrp.VrrpGroup != nil {
				addrObj = cand
				ok = true
				break
			}
		}
		if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
			return nil, 0, false
		}
	}
	if targetVrid != 0 {
		grpObj, ok := addrObj.Vrrp.VrrpGroup[targetVrid]
		return grpObj, targetVrid, ok && grpObj != nil
	}
	for vrid, grpObj := range addrObj.Vrrp.VrrpGroup {
		if grpObj != nil {
			return grpObj, vrid, true
		}
	}
	return nil, 0, false
}

func getSubIntfIPv6VrrpGroupFromRoot(inParams XfmrParams, ifName string, subIntfIdx uint32, uriIP string, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || intfsObj.Interface == nil {
		return nil, 0, false
	}
	intfObj, ok := intfsObj.Interface[ifName]
	if !ok || intfObj == nil || intfObj.Subinterfaces == nil || intfObj.Subinterfaces.Subinterface == nil {
		return nil, 0, false
	}
	subIntfObj, ok := intfObj.Subinterfaces.Subinterface[subIntfIdx]
	if !ok || subIntfObj == nil || subIntfObj.Ipv6 == nil || subIntfObj.Ipv6.Addresses == nil || subIntfObj.Ipv6.Addresses.Address == nil {
		return nil, 0, false
	}
	addrObj, ok := subIntfObj.Ipv6.Addresses.Address[uriIP]
	if uriIP != "" && uriIP != "*" && (!ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil) {
		return nil, 0, false
	}
	if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
		for _, cand := range subIntfObj.Ipv6.Addresses.Address {
			if cand != nil && cand.Vrrp != nil && cand.Vrrp.VrrpGroup != nil {
				addrObj = cand
				ok = true
				break
			}
		}
		if !ok || addrObj == nil || addrObj.Vrrp == nil || addrObj.Vrrp.VrrpGroup == nil {
			return nil, 0, false
		}
	}
	if targetVrid != 0 {
		grpObj, ok := addrObj.Vrrp.VrrpGroup[targetVrid]
		return grpObj, targetVrid, ok && grpObj != nil
	}
	for vrid, grpObj := range addrObj.Vrrp.VrrpGroup {
		if grpObj != nil {
			return grpObj, vrid, true
		}
	}
	return nil, 0, false
}

func getSubIntfIPv4VrrpGroupFromParam(param interface{}, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	switch p := param.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address:
		if p != nil && p.Vrrp != nil {
			return getSubIntfIPv4VrrpGroupFromParam(p.Vrrp, targetVrid)
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp:
		if p == nil || p.VrrpGroup == nil {
			return nil, 0, false
		}
		if targetVrid != 0 {
			grpObj, ok := p.VrrpGroup[targetVrid]
			return grpObj, targetVrid, ok && grpObj != nil
		}
		for vrid, grpObj := range p.VrrpGroup {
			if grpObj != nil {
				return grpObj, vrid, true
			}
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup:
		if p != nil {
			return p, resolveVrid("", p.VirtualRouterId, p.Config, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup{Config: p}, resolveVrid("", nil, p, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup{InterfaceTracking: p}, targetVrid, true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup{
				InterfaceTracking: &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking{Config: p},
			}, targetVrid, true
		}
	}
	return nil, 0, false
}

func getSubIntfIPv6VrrpGroupFromParam(param interface{}, targetVrid uint8) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup, uint8, bool) {
	switch p := param.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address:
		if p != nil && p.Vrrp != nil {
			return getSubIntfIPv6VrrpGroupFromParam(p.Vrrp, targetVrid)
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp:
		if p == nil || p.VrrpGroup == nil {
			return nil, 0, false
		}
		if targetVrid != 0 {
			grpObj, ok := p.VrrpGroup[targetVrid]
			return grpObj, targetVrid, ok && grpObj != nil
		}
		for vrid, grpObj := range p.VrrpGroup {
			if grpObj != nil {
				return grpObj, vrid, true
			}
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup:
		if p != nil {
			return p, resolveVrid("", p.VirtualRouterId, p.Config, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup{Config: p}, resolveVrid("", nil, p, targetVrid), true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup{InterfaceTracking: p}, targetVrid, true
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking_Config:
		if p != nil {
			return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup{
				InterfaceTracking: &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking{Config: p},
			}, targetVrid, true
		}
	}
	return nil, 0, false
}

func mapRoutedVlanIPv4VrrpGroupToDbFields(
	grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup,
	grpVrid uint8,
	ifName string,
	vrrpTrackTbl string,
	cfgDb *db.DB,
	pathInfo *PathInfo,
) (uint8, map[string]string, map[string]db.Value) {
	fields := map[string]string{}
	vrid := resolveVrid(pathInfo.Var("virtual-router-id"), grpObj.VirtualRouterId, grpObj.Config, grpVrid)
	if vrid == 0 {
		return 0, fields, nil
	}
	if grpObj.Config != nil {
		if grpObj.Config.Priority != nil {
			fields[VRRP_FIELD_PRIORITY] = strconv.Itoa(int(*grpObj.Config.Priority))
		}
		if grpObj.Config.Preempt != nil {
			fields[VRRP_FIELD_PREEMPT] = boolToAdminMode(*grpObj.Config.Preempt)
		}
		if grpObj.Config.PreemptDelay != nil {
			fields[VRRP_FIELD_PREEMPT_DELAY] = strconv.Itoa(int(*grpObj.Config.PreemptDelay))
		}
		if grpObj.Config.AdvertisementInterval != nil {
			fields[VRRP_FIELD_ADV_INTERVAL] = strconv.Itoa(centiToSec(*grpObj.Config.AdvertisementInterval))
		}
		if len(grpObj.Config.VirtualAddress) > 0 {
			fields[VRRP_FIELD_VIP] = strings.Join(normalizeVirtualAddressesList(grpObj.Config.VirtualAddress, false), ",")
		}
	}
	trackMap := mapRoutedVlanIPv4TrackToDb(ifName, vrid, grpObj.InterfaceTracking, cfgDb, vrrpTrackTbl)
	return vrid, fields, trackMap
}

func mapRoutedVlanIPv6VrrpGroupToDbFields(
	grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup,
	grpVrid uint8,
	ifName string,
	vrrpTrackTbl string,
	cfgDb *db.DB,
	pathInfo *PathInfo,
) (uint8, map[string]string, map[string]db.Value) {
	fields := map[string]string{}
	vrid := resolveVrid(pathInfo.Var("virtual-router-id"), grpObj.VirtualRouterId, grpObj.Config, grpVrid)
	if vrid == 0 {
		return 0, fields, nil
	}
	if grpObj.Config != nil {
		if grpObj.Config.Priority != nil {
			fields[VRRP_FIELD_PRIORITY] = strconv.Itoa(int(*grpObj.Config.Priority))
		}
		if grpObj.Config.Preempt != nil {
			fields[VRRP_FIELD_PREEMPT] = boolToAdminMode(*grpObj.Config.Preempt)
		}
		if grpObj.Config.PreemptDelay != nil {
			fields[VRRP_FIELD_PREEMPT_DELAY] = strconv.Itoa(int(*grpObj.Config.PreemptDelay))
		}
		if grpObj.Config.AdvertisementInterval != nil {
			fields[VRRP_FIELD_ADV_INTERVAL] = strconv.Itoa(centiToSec(*grpObj.Config.AdvertisementInterval))
		}
		if len(grpObj.Config.VirtualAddress) > 0 {
			fields[VRRP_FIELD_VIP] = strings.Join(normalizeVirtualAddressesList(grpObj.Config.VirtualAddress, true), ",")
		}
	}
	trackMap := mapRoutedVlanIPv6TrackToDb(ifName, vrid, grpObj.InterfaceTracking, cfgDb, vrrpTrackTbl)
	return vrid, fields, trackMap
}

func mapSubIntfIPv4VrrpGroupToDbFields(
	grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup,
	grpVrid uint8,
	ifName string,
	vrrpTrackTbl string,
	cfgDb *db.DB,
	pathInfo *PathInfo,
) (uint8, map[string]string, map[string]db.Value) {
	fields := map[string]string{}
	vrid := resolveVrid(pathInfo.Var("virtual-router-id"), grpObj.VirtualRouterId, grpObj.Config, grpVrid)
	if vrid == 0 {
		return 0, fields, nil
	}
	if grpObj.Config != nil {
		if grpObj.Config.Priority != nil {
			fields[VRRP_FIELD_PRIORITY] = strconv.Itoa(int(*grpObj.Config.Priority))
		}
		if grpObj.Config.Preempt != nil {
			fields[VRRP_FIELD_PREEMPT] = boolToAdminMode(*grpObj.Config.Preempt)
		}
		if grpObj.Config.PreemptDelay != nil {
			fields[VRRP_FIELD_PREEMPT_DELAY] = strconv.Itoa(int(*grpObj.Config.PreemptDelay))
		}
		if grpObj.Config.AdvertisementInterval != nil {
			fields[VRRP_FIELD_ADV_INTERVAL] = strconv.Itoa(centiToSec(*grpObj.Config.AdvertisementInterval))
		}
		if len(grpObj.Config.VirtualAddress) > 0 {
			fields[VRRP_FIELD_VIP] = strings.Join(normalizeVirtualAddressesList(grpObj.Config.VirtualAddress, false), ",")
		}
	}
	trackMap := mapSubIntfTrackToDb(ifName, vrid, grpObj.InterfaceTracking, cfgDb, vrrpTrackTbl)
	return vrid, fields, trackMap
}

func mapSubIntfIPv6VrrpGroupToDbFields(
	grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup,
	grpVrid uint8,
	ifName string,
	vrrpTrackTbl string,
	cfgDb *db.DB,
	pathInfo *PathInfo,
) (uint8, map[string]string, map[string]db.Value) {
	fields := map[string]string{}
	vrid := resolveVrid(pathInfo.Var("virtual-router-id"), grpObj.VirtualRouterId, grpObj.Config, grpVrid)
	if vrid == 0 {
		return 0, fields, nil
	}
	if grpObj.Config != nil {
		if grpObj.Config.Priority != nil {
			fields[VRRP_FIELD_PRIORITY] = strconv.Itoa(int(*grpObj.Config.Priority))
		}
		if grpObj.Config.Preempt != nil {
			fields[VRRP_FIELD_PREEMPT] = boolToAdminMode(*grpObj.Config.Preempt)
		}
		if grpObj.Config.PreemptDelay != nil {
			fields[VRRP_FIELD_PREEMPT_DELAY] = strconv.Itoa(int(*grpObj.Config.PreemptDelay))
		}
		if grpObj.Config.AdvertisementInterval != nil {
			fields[VRRP_FIELD_ADV_INTERVAL] = strconv.Itoa(centiToSec(*grpObj.Config.AdvertisementInterval))
		}
		if len(grpObj.Config.VirtualAddress) > 0 {
			fields[VRRP_FIELD_VIP] = strings.Join(normalizeVirtualAddressesList(grpObj.Config.VirtualAddress, true), ",")
		}
	}
	trackMap := mapSubIntfIPv6TrackToDb(ifName, vrid, grpObj.InterfaceTracking, cfgDb, vrrpTrackTbl)
	return vrid, fields, trackMap
}

func resolveVrid(uriVrid string, vridPtr *uint8, cfg interface{}, fallbackVrid ...uint8) uint8 {
	if uriVrid != "" {
		return parseUint8(uriVrid, 0)
	}
	if vridPtr != nil {
		return *vridPtr
	}
	switch c := cfg.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_Config:
		if c != nil && c.VirtualRouterId != nil {
			return *c.VirtualRouterId
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_Config:
		if c != nil && c.VirtualRouterId != nil {
			return *c.VirtualRouterId
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_Config:
		if c != nil && c.VirtualRouterId != nil {
			return *c.VirtualRouterId
		}
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_Config:
		if c != nil && c.VirtualRouterId != nil {
			return *c.VirtualRouterId
		}
	}
	if len(fallbackVrid) > 0 && fallbackVrid[0] != 0 {
		return fallbackVrid[0]
	}
	return 0
}

func mapSubIntfTrackToDb(
	ifName string,
	vrid uint8,
	trackObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking,
	cfgDb *db.DB,
	trackTbl string,
) map[string]db.Value {
	res := make(map[string]db.Value)
	if trackObj == nil || trackObj.Config == nil {
		return res
	}
	var prioDecStr string
	if trackObj.Config.PriorityDecrement != nil {
		prioDecStr = strconv.Itoa(int(*trackObj.Config.PriorityDecrement))
	}
	trackList := sanitizeTrackList(trackObj.Config.TrackInterface)
	if len(trackList) == 0 && prioDecStr != "" {
		trackList = getExistingTrackInterfaces(cfgDb, trackTbl, ifName, vrid)
	}
	for _, trackIf := range trackList {
		key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
		fieldMap := map[string]string{}
		if prioDecStr != "" {
			fieldMap[VRRP_FIELD_TRACK_PRIO_INCR] = prioDecStr
		}
		if len(fieldMap) > 0 {
			res[key] = db.Value{Field: fieldMap}
		}
	}
	return res
}

func mapSubIntfIPv6TrackToDb(
	ifName string,
	vrid uint8,
	trackObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking,
	cfgDb *db.DB,
	trackTbl string,
) map[string]db.Value {
	res := make(map[string]db.Value)
	if trackObj == nil || trackObj.Config == nil {
		return res
	}
	var prioDecStr string
	if trackObj.Config.PriorityDecrement != nil {
		prioDecStr = strconv.Itoa(int(*trackObj.Config.PriorityDecrement))
	}
	trackList := sanitizeTrackList(trackObj.Config.TrackInterface)
	if len(trackList) == 0 && prioDecStr != "" {
		trackList = getExistingTrackInterfaces(cfgDb, trackTbl, ifName, vrid)
	}
	for _, trackIf := range trackList {
		key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
		fieldMap := map[string]string{}
		if prioDecStr != "" {
			fieldMap[VRRP_FIELD_TRACK_PRIO_INCR] = prioDecStr
		}
		if len(fieldMap) > 0 {
			res[key] = db.Value{Field: fieldMap}
		}
	}
	return res
}

func mapRoutedVlanIPv4TrackToDb(
	ifName string,
	vrid uint8,
	trackObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking,
	cfgDb *db.DB,
	trackTbl string,
) map[string]db.Value {
	res := make(map[string]db.Value)
	if trackObj == nil || trackObj.Config == nil {
		return res
	}
	var prioDecStr string
	if trackObj.Config.PriorityDecrement != nil {
		prioDecStr = strconv.Itoa(int(*trackObj.Config.PriorityDecrement))
	}
	trackList := sanitizeTrackList(trackObj.Config.TrackInterface)
	if len(trackList) == 0 && prioDecStr != "" {
		trackList = getExistingTrackInterfaces(cfgDb, trackTbl, ifName, vrid)
	}
	for _, trackIf := range trackList {
		key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
		fieldMap := map[string]string{}
		if prioDecStr != "" {
			fieldMap[VRRP_FIELD_TRACK_PRIO_INCR] = prioDecStr
		}
		if len(fieldMap) > 0 {
			res[key] = db.Value{Field: fieldMap}
		}
	}
	return res
}

func mapRoutedVlanIPv6TrackToDb(
	ifName string,
	vrid uint8,
	trackObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup_InterfaceTracking,
	cfgDb *db.DB,
	trackTbl string,
) map[string]db.Value {
	res := make(map[string]db.Value)
	if trackObj == nil || trackObj.Config == nil {
		return res
	}
	var prioDecStr string
	if trackObj.Config.PriorityDecrement != nil {
		prioDecStr = strconv.Itoa(int(*trackObj.Config.PriorityDecrement))
	}
	trackList := sanitizeTrackList(trackObj.Config.TrackInterface)
	if len(trackList) == 0 && prioDecStr != "" {
		trackList = getExistingTrackInterfaces(cfgDb, trackTbl, ifName, vrid)
	}
	for _, trackIf := range trackList {
		key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
		fieldMap := map[string]string{}
		if prioDecStr != "" {
			fieldMap[VRRP_FIELD_TRACK_PRIO_INCR] = prioDecStr
		}
		if len(fieldMap) > 0 {
			res[key] = db.Value{Field: fieldMap}
		}
	}
	return res
}

func sanitizeTrackList(trackList []string) []string {
	if len(trackList) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(trackList))
	for _, trackIf := range trackList {
		trackIf = strings.TrimSpace(trackIf)
		if trackIf != "" {
			out = append(out, trackIf)
		}
	}
	if len(out) < 2 {
		return out
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func getExistingTrackInterfaces(cfgDb *db.DB, trackTbl string, ifName string, vrid uint8) []string {
	if cfgDb == nil {
		return []string{}
	}
	ts := db.TableSpec{Name: trackTbl}
	keyPattern := fmt.Sprintf("%s|%d|*", ifName, vrid)
	keys, err := cfgDb.GetKeysByPattern(&ts, keyPattern)
	if err != nil {
		return []string{}
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key.Comp) < 3 {
			continue
		}
		trackIf := strings.TrimSpace(key.Comp[2])
		if trackIf != "" {
			out = append(out, trackIf)
		}
	}
	if len(out) < 2 {
		return out
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func getTrackDeleteMap(cfgDb *db.DB, trackTbl string, ifName string, vrid uint8) map[string]db.Value {
	out := map[string]db.Value{}
	if cfgDb == nil {
		return out
	}
	ts := db.TableSpec{Name: trackTbl}
	keyPattern := fmt.Sprintf("%s|%d|*", ifName, vrid)
	keys, err := cfgDb.GetKeysByPattern(&ts, keyPattern)
	if err != nil {
		return out
	}
	for _, key := range keys {
		if len(key.Comp) < 3 {
			continue
		}
		out[key.Comp[0]+"|"+key.Comp[1]+"|"+key.Comp[2]] = db.Value{}
	}
	return out
}

func centiToSec(centi uint16) int {
	if centi == 0 {
		return 0
	}
	sec := int(centi) / 100
	if sec == 0 {
		sec = 1
	}
	return sec
}

func dbToYangIPv4Vrrp(inParams XfmrParams) error {
	return dbToYangVrrp(inParams, false)
}

func dbToYangIPv6Vrrp(inParams XfmrParams) error {
	return dbToYangVrrp(inParams, true)
}

func dbToYangVrrp(inParams XfmrParams, isIPv6 bool) error {
	pathInfo := getVrrpReadPathInfo(inParams)
	ifName := strings.TrimSpace(pathInfo.Var("name"))
	targetVrid := getTargetVridForRead(pathInfo, inParams.key)

	if ifName == "" {
		ifName = extractIfNameFromDbKey(inParams.key)
	}
	log.Warningf("VRRP_TRACE dbToYangVrrp: uri=%s requestUri=%s keyIn=%s name=%s targetVrid=%s isIPv6=%v",
		inParams.uri, inParams.requestUri, inParams.key, ifName, targetVrid, isIPv6)

	if ifName == "" {
		return nil
	}

	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		return nil
	}

	if err := validateVrrpAddressKey(cfgDb, ifName, pathInfo.Var("ip"), isIPv6); err != nil {
		return err
	}

	vrrpTbl := VRRP_TBL
	vrrpTrackTbl := VRRP_TRACK_TBL
	vrrpStateTbl := VRRP_STATE_TBL
	if isIPv6 {
		vrrpTbl = VRRP6_TBL
		vrrpTrackTbl = VRRP6_TRACK_TBL
		vrrpStateTbl = VRRP6_STATE_TBL
	}

	vrrpData, err := getIntfVrrpData(cfgDb, ifName, targetVrid, vrrpTbl, isIPv6)
	if err != nil || len(vrrpData) == 0 {
		return err
	}

	if err = mergeIntfVrrpTrackingData(cfgDb, ifName, targetVrid, vrrpTrackTbl, vrrpData); err != nil {
		return err
	}

	if err = mergeIntfVrrpStateData(inParams.dbs[db.StateDB], ifName, targetVrid, vrrpStateTbl, vrrpData); err != nil {
		return err
	}

	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil {
		return nil
	}

	intfObj := getIntfRoutedVlanObject(intfsObj, ifName)
	if isIPv6 {
		return fillRoutedVlanIPv6Vrrp(pathInfo.Var("ip"), intfObj, vrrpData)
	}
	return fillRoutedVlanIPv4Vrrp(pathInfo.Var("ip"), intfObj, vrrpData)
}

func dbToYangSubIntfVrrp(inParams XfmrParams, isIPv6 bool) error {
	pathInfo := getVrrpReadPathInfo(inParams)
	ifName := strings.TrimSpace(pathInfo.Var("name"))
	targetVrid := getTargetVridForRead(pathInfo, inParams.key)
	ifNameFromKey := extractIfNameFromDbKey(inParams.key)
	if ifName == "" {
		ifName = ifNameFromKey
	}
	log.Warningf("VRRP_TRACE dbToYangSubIntfVrrp: uri=%s requestUri=%s keyIn=%s name=%s keyName=%s index=%s targetVrid=%s isIPv6=%v",
		inParams.uri, inParams.requestUri, inParams.key, ifName, ifNameFromKey, pathInfo.Var("index"), targetVrid, isIPv6)
	if ifName == "" {
		return nil
	}

	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		return nil
	}

	addrIfName := ifName
	if subIdx := parseSubIntfIndex(pathInfo.Var("index")); subIdx != 0 {
		addrIfName = fmt.Sprintf("%s.%d", ifName, subIdx)
	}
	if err := validateVrrpAddressKey(cfgDb, addrIfName, pathInfo.Var("ip"), isIPv6); err != nil {
		return err
	}

	vrrpTbl := VRRP_TBL
	vrrpTrackTbl := VRRP_TRACK_TBL
	vrrpStateTbl := VRRP_STATE_TBL
	if isIPv6 {
		vrrpTbl = VRRP6_TBL
		vrrpTrackTbl = VRRP6_TRACK_TBL
		vrrpStateTbl = VRRP6_STATE_TBL
	}

	vrrpData, err := getIntfVrrpData(cfgDb, ifName, targetVrid, vrrpTbl, isIPv6)
	if err != nil || len(vrrpData) == 0 {
		return err
	}

	if err = mergeIntfVrrpTrackingData(cfgDb, ifName, targetVrid, vrrpTrackTbl, vrrpData); err != nil {
		return err
	}

	if err = mergeIntfVrrpStateData(inParams.dbs[db.StateDB], ifName, targetVrid, vrrpStateTbl, vrrpData); err != nil {
		return err
	}

	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil {
		return nil
	}

	intfObj := getIntfObject(intfsObj, ifName)
	if intfObj == nil {
		return nil
	}

	subIntfObj, err := getSubinterfaceObject(intfObj, parseSubIntfIndex(pathInfo.Var("index")))
	if err != nil {
		return err
	}

	if isIPv6 {
		return fillSubIntfIPv6Vrrp(pathInfo.Var("ip"), subIntfObj, vrrpData)
	}
	return fillSubIntfIPv4Vrrp(pathInfo.Var("ip"), subIntfObj, vrrpData)
}

func getVrrpReadPathInfo(inParams XfmrParams) *PathInfo {
	pathInfo := NewPathInfo(inParams.uri)
	if inParams.requestUri == "" || inParams.requestUri == inParams.uri {
		return pathInfo
	}
	reqPathInfo := NewPathInfo(inParams.requestUri)
	if reqPathInfo.Var("name") == "" {
		return pathInfo
	}
	if pathInfo.Var("name") != "" && isMoreSpecificPath(reqPathInfo.YangPath, pathInfo.YangPath) {
		return reqPathInfo
	}
	// Prefer requestUri when it carries a more specific keyed target
	// (ip/index/virtual-router-id) than callback uri.
	if (pathInfo.Var("ip") == "" && reqPathInfo.Var("ip") != "") ||
		(pathInfo.Var("index") == "" && reqPathInfo.Var("index") != "") ||
		(pathInfo.Var("virtual-router-id") == "" && reqPathInfo.Var("virtual-router-id") != "") {
		return reqPathInfo
	}
	if pathInfo.Var("name") == "" {
		return reqPathInfo
	}
	return pathInfo
}

func isMoreSpecificPath(requestYangPath string, callbackYangPath string) bool {
	if requestYangPath == "" || callbackYangPath == "" {
		return false
	}
	return strings.HasPrefix(requestYangPath, callbackYangPath) && len(requestYangPath) > len(callbackYangPath)
}

func normalizedYangPath(yangPath string) string {
	if yangPath == "" {
		return ""
	}
	parts := strings.Split(yangPath, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if idx := strings.IndexByte(part, ':'); idx >= 0 {
			part = part[idx+1:]
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return ""
	}
	return "/" + strings.Join(out, "/")
}

func buildVrrpLeafDeleteMap(
	pathInfo *PathInfo,
	ifName string,
	vrid uint8,
	isIPv6 bool,
	vrrpTbl string,
	vrrpTrackTbl string,
	cfgDb *db.DB,
) (bool, map[string]map[string]db.Value) {
	resMap := map[string]map[string]db.Value{
		vrrpTbl: {},
	}
	normPath := normalizedYangPath(pathInfo.YangPath)
	if normPath == "" {
		return false, resMap
	}
	vrrpKey := fmt.Sprintf("%s|%d", ifName, vrid)
	fields := map[string]string{}

	switch {
	case strings.HasSuffix(normPath, "/vrrp-group/config/priority"):
		fields[VRRP_FIELD_PRIORITY] = "100"
	case strings.HasSuffix(normPath, "/vrrp-group/config/preempt"):
		fields[VRRP_FIELD_PREEMPT] = "true"
	case strings.HasSuffix(normPath, "/vrrp-group/config/preempt-delay"):
		fields[VRRP_FIELD_PREEMPT_DELAY] = "0"
	case strings.HasSuffix(normPath, "/vrrp-group/config/advertisement-interval"):
		fields[VRRP_FIELD_ADV_INTERVAL] = "1"
	case strings.HasSuffix(normPath, "/vrrp-group/config/virtual-address"):
		fields[VRRP_FIELD_VIP] = ""
	case strings.HasSuffix(normPath, "/vrrp-group/interface-tracking/config/priority-decrement"):
		trackIfList := getExistingTrackInterfaces(cfgDb, vrrpTrackTbl, ifName, vrid)
		if len(trackIfList) == 0 {
			return true, resMap
		}
		resMap[vrrpTrackTbl] = map[string]db.Value{}
		for _, trackIf := range trackIfList {
			key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
			resMap[vrrpTrackTbl][key] = db.Value{Field: map[string]string{VRRP_FIELD_TRACK_PRIO_INCR: "0"}}
		}
		return true, resMap
	case strings.HasSuffix(normPath, "/vrrp-group/interface-tracking/config/track-interface"):
		resMap[vrrpTrackTbl] = getTrackDeleteMap(cfgDb, vrrpTrackTbl, ifName, vrid)
		return true, resMap
	default:
		return false, resMap
	}

	fields[VRRP_FIELD_VRID] = strconv.Itoa(int(vrid))
	resMap[vrrpTbl][vrrpKey] = db.Value{Field: fields}
	return true, resMap
}

func buildVrrpLeafSetMap(
	pathInfo *PathInfo,
	param interface{},
	ifName string,
	targetVrid uint8,
	isIPv6 bool,
	vrrpTrackTbl string,
	cfgDb *db.DB,
) (uint8, map[string]string, map[string]db.Value, bool) {
	fields := map[string]string{}
	trackEntries := map[string]db.Value{}
	vrid := targetVrid
	if vrid == 0 {
		vrid = parseUint8(pathInfo.Var("virtual-router-id"), 0)
	}
	if vrid == 0 {
		return 0, fields, trackEntries, false
	}
	normPath := normalizedYangPath(pathInfo.YangPath)
	log.Warningf("VRRP_TRACE buildVrrpLeafSetMap: yangPath=%s normalized=%s targetVrid=%d paramType=%T",
		pathInfo.YangPath, normPath, vrid, param)
	if normPath == "" {
		return 0, fields, trackEntries, false
	}

	switch {
	case strings.HasSuffix(normPath, "/vrrp-group/config/priority"):
		val, ok := toUint64Param(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		fields[VRRP_FIELD_PRIORITY] = strconv.Itoa(int(uint8(val)))
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/config/preempt"):
		val, ok := toBoolParam(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		fields[VRRP_FIELD_PREEMPT] = boolToAdminMode(val)
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/config/preempt-delay"):
		val, ok := toUint64Param(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		fields[VRRP_FIELD_PREEMPT_DELAY] = strconv.Itoa(int(uint16(val)))
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/config/advertisement-interval"):
		val, ok := toUint64Param(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		fields[VRRP_FIELD_ADV_INTERVAL] = strconv.Itoa(centiToSec(uint16(val)))
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/config/virtual-address"):
		addrs, ok := toStringSliceParam(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		normAddrs := normalizeVirtualAddressesList(addrs, isIPv6)
		if len(normAddrs) == 0 {
			return 0, fields, trackEntries, false
		}
		fields[VRRP_FIELD_VIP] = strings.Join(normAddrs, ",")
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/interface-tracking/config/priority-decrement"):
		val, ok := toUint64Param(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		prioIncr := strconv.Itoa(int(uint8(val)))
		trackIfList := getExistingTrackInterfaces(cfgDb, vrrpTrackTbl, ifName, vrid)
		for _, trackIf := range trackIfList {
			key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
			trackEntries[key] = db.Value{Field: map[string]string{VRRP_FIELD_TRACK_PRIO_INCR: prioIncr}}
		}
		return vrid, fields, trackEntries, true
	case strings.HasSuffix(normPath, "/vrrp-group/interface-tracking/config/track-interface"):
		trackIfList, ok := toStringSliceParam(param)
		if !ok {
			return 0, fields, trackEntries, false
		}
		for _, trackIf := range sanitizeTrackList(trackIfList) {
			key := fmt.Sprintf("%s|%d|%s", ifName, vrid, trackIf)
			fieldMap := map[string]string{}
			if prio, exists := getTrackPriorityIncrement(cfgDb, vrrpTrackTbl, ifName, vrid, trackIf); exists {
				fieldMap[VRRP_FIELD_TRACK_PRIO_INCR] = prio
			}
			trackEntries[key] = db.Value{Field: fieldMap}
		}
		return vrid, fields, trackEntries, true
	}
	return 0, fields, trackEntries, false
}

func toBoolParam(param interface{}) (bool, bool) {
	switch v := param.(type) {
	case bool:
		return v, true
	case *bool:
		if v != nil {
			return *v, true
		}
	case string:
		p, err := strconv.ParseBool(strings.TrimSpace(v))
		return p, err == nil
	case *string:
		if v != nil {
			p, err := strconv.ParseBool(strings.TrimSpace(*v))
			return p, err == nil
		}
	}
	return false, false
}

func toUint64Param(param interface{}) (uint64, bool) {
	switch v := param.(type) {
	case uint8:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	case int:
		if v >= 0 {
			return uint64(v), true
		}
	case int32:
		if v >= 0 {
			return uint64(v), true
		}
	case int64:
		if v >= 0 {
			return uint64(v), true
		}
	case *uint8:
		if v != nil {
			return uint64(*v), true
		}
	case *uint16:
		if v != nil {
			return uint64(*v), true
		}
	case *uint32:
		if v != nil {
			return uint64(*v), true
		}
	case *uint64:
		if v != nil {
			return *v, true
		}
	case *int:
		if v != nil && *v >= 0 {
			return uint64(*v), true
		}
	case *int32:
		if v != nil && *v >= 0 {
			return uint64(*v), true
		}
	case *int64:
		if v != nil && *v >= 0 {
			return uint64(*v), true
		}
	case string:
		p, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		return p, err == nil
	case *string:
		if v != nil {
			p, err := strconv.ParseUint(strings.TrimSpace(*v), 10, 64)
			return p, err == nil
		}
	}
	return 0, false
}

func toStringSliceParam(param interface{}) ([]string, bool) {
	switch v := param.(type) {
	case []string:
		return v, true
	case *[]string:
		if v != nil {
			return *v, true
		}
	case string:
		return []string{v}, true
	case *string:
		if v != nil {
			return []string{*v}, true
		}
	}
	return nil, false
}

func getTrackPriorityIncrement(cfgDb *db.DB, trackTbl string, ifName string, vrid uint8, trackIf string) (string, bool) {
	if cfgDb == nil {
		return "", false
	}
	ts := db.TableSpec{Name: trackTbl}
	key := db.Key{Comp: []string{ifName, strconv.Itoa(int(vrid)), trackIf}}
	entry, err := cfgDb.GetEntry(&ts, key)
	if err != nil {
		return "", false
	}
	prio := strings.TrimSpace(entry.Get(VRRP_FIELD_TRACK_PRIO_INCR))
	return prio, prio != ""
}

func fillRoutedVlanIPv4Vrrp(ip string, intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface, vrrpData map[uint8]vrrpGroupData) error {
	ygot.BuildEmptyTree(intfObj)
	ygot.BuildEmptyTree(intfObj.RoutedVlan)
	ygot.BuildEmptyTree(intfObj.RoutedVlan.Ipv4)

	v4Addr, err := getRoutedVlanIPv4AddressObj(intfObj.RoutedVlan, ip)
	if err != nil {
		return err
	}

	ygot.BuildEmptyTree(v4Addr)
	ygot.BuildEmptyTree(v4Addr.Vrrp)

	for vrid, grp := range vrrpData {
		if _, ok := v4Addr.Vrrp.VrrpGroup[vrid]; !ok {
			if _, err = v4Addr.Vrrp.NewVrrpGroup(vrid); err != nil {
				return err
			}
		}

		grpObj := v4Addr.Vrrp.VrrpGroup[vrid]
		populateRoutedVlanVrrpGroup(grpObj, grp)
	}

	return nil
}

func fillSubIntfIPv4Vrrp(ip string, subIntfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface, vrrpData map[uint8]vrrpGroupData) error {
	ygot.BuildEmptyTree(subIntfObj)
	ygot.BuildEmptyTree(subIntfObj.Ipv4)

	v4Addr, err := getSubIntfIPv4AddressObj(subIntfObj, ip)
	if err != nil {
		return err
	}

	ygot.BuildEmptyTree(v4Addr)
	ygot.BuildEmptyTree(v4Addr.Vrrp)

	for vrid, grp := range vrrpData {
		if _, ok := v4Addr.Vrrp.VrrpGroup[vrid]; !ok {
			if _, err = v4Addr.Vrrp.NewVrrpGroup(vrid); err != nil {
				return err
			}
		}

		grpObj := v4Addr.Vrrp.VrrpGroup[vrid]
		populateSubIntfIPv4VrrpGroup(grpObj, grp)
	}

	return nil
}

func fillSubIntfIPv6Vrrp(ip string, subIntfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface, vrrpData map[uint8]vrrpGroupData) error {
	ygot.BuildEmptyTree(subIntfObj)
	ygot.BuildEmptyTree(subIntfObj.Ipv6)

	v6Addr, err := getSubIntfIPv6AddressObj(subIntfObj, ip)
	if err != nil {
		return err
	}

	ygot.BuildEmptyTree(v6Addr)
	ygot.BuildEmptyTree(v6Addr.Vrrp)

	for vrid, grp := range vrrpData {
		if _, ok := v6Addr.Vrrp.VrrpGroup[vrid]; !ok {
			if _, err = v6Addr.Vrrp.NewVrrpGroup(vrid); err != nil {
				return err
			}
		}

		grpObj := v6Addr.Vrrp.VrrpGroup[vrid]
		populateSubIntfIPv6VrrpGroup(grpObj, grp)
	}

	return nil
}

func fillRoutedVlanIPv6Vrrp(ip string, intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface, vrrpData map[uint8]vrrpGroupData) error {
	ygot.BuildEmptyTree(intfObj)
	ygot.BuildEmptyTree(intfObj.RoutedVlan)
	ygot.BuildEmptyTree(intfObj.RoutedVlan.Ipv6)

	v6Addr, err := getRoutedVlanIPv6AddressObj(intfObj.RoutedVlan, ip)
	if err != nil {
		return err
	}

	ygot.BuildEmptyTree(v6Addr)
	ygot.BuildEmptyTree(v6Addr.Vrrp)

	for vrid, grp := range vrrpData {
		if _, ok := v6Addr.Vrrp.VrrpGroup[vrid]; !ok {
			if _, err = v6Addr.Vrrp.NewVrrpGroup(vrid); err != nil {
				return err
			}
		}

		grpObj := v6Addr.Vrrp.VrrpGroup[vrid]
		populateRoutedVlanIPv6VrrpGroup(grpObj, grp)
	}

	return nil
}

func getRoutedVlanIPv4AddressObj(routedVlanObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan, uriIP string) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address, error) {
	addressIp := uriIP
	if addressIp == "" {
		addressIp = "0.0.0.0"
	}

	if _, ok := routedVlanObj.Ipv4.Addresses.Address[addressIp]; !ok {
		if _, err := routedVlanObj.Ipv4.Addresses.NewAddress(addressIp); err != nil {
			return nil, err
		}
	}

	return routedVlanObj.Ipv4.Addresses.Address[addressIp], nil
}

func getRoutedVlanIPv6AddressObj(routedVlanObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan, uriIP string) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address, error) {
	addressIp := uriIP
	if addressIp == "" {
		addressIp = "::"
	}

	if _, ok := routedVlanObj.Ipv6.Addresses.Address[addressIp]; !ok {
		if _, err := routedVlanObj.Ipv6.Addresses.NewAddress(addressIp); err != nil {
			return nil, err
		}
	}

	return routedVlanObj.Ipv6.Addresses.Address[addressIp], nil
}

func getIntfObject(intfsObj *ocbinds.OpenconfigInterfaces_Interfaces, intfName string) *ocbinds.OpenconfigInterfaces_Interfaces_Interface {
	if intfsObj == nil {
		return nil
	}
	ygot.BuildEmptyTree(intfsObj)

	var intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface
	if intfsObj.Interface != nil {
		intfObj = intfsObj.Interface[intfName]
	}
	if intfObj == nil {
		intfObj, _ = intfsObj.NewInterface(intfName)
	}
	ygot.BuildEmptyTree(intfObj)
	return intfObj
}

func parseSubIntfIndex(subIntfStr string) uint32 {
	if subIntfStr == "" {
		return 0
	}
	idx, err := strconv.ParseUint(strings.TrimSpace(subIntfStr), 10, 32)
	if err != nil {
		return 0
	}
	return uint32(idx)
}

func getSubinterfaceObject(intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface, subIntfIdx uint32) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface, error) {
	ygot.BuildEmptyTree(intfObj.Subinterfaces)

	if _, ok := intfObj.Subinterfaces.Subinterface[subIntfIdx]; !ok {
		if _, err := intfObj.Subinterfaces.NewSubinterface(subIntfIdx); err != nil {
			return nil, err
		}
	}

	subIntfObj := intfObj.Subinterfaces.Subinterface[subIntfIdx]
	ygot.BuildEmptyTree(subIntfObj)
	return subIntfObj, nil
}

func getSubIntfIPv4AddressObj(subIntfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface, uriIP string) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address, error) {
	addressIp := uriIP
	if addressIp == "" {
		addressIp = "0.0.0.0"
	}

	if _, ok := subIntfObj.Ipv4.Addresses.Address[addressIp]; !ok {
		if _, err := subIntfObj.Ipv4.Addresses.NewAddress(addressIp); err != nil {
			return nil, err
		}
	}
	return subIntfObj.Ipv4.Addresses.Address[addressIp], nil
}

func getSubIntfIPv6AddressObj(subIntfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface, uriIP string) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address, error) {
	addressIp := uriIP
	if addressIp == "" {
		addressIp = "::"
	}

	if _, ok := subIntfObj.Ipv6.Addresses.Address[addressIp]; !ok {
		if _, err := subIntfObj.Ipv6.Addresses.NewAddress(addressIp); err != nil {
			return nil, err
		}
	}
	return subIntfObj.Ipv6.Addresses.Address[addressIp], nil
}

func populateRoutedVlanVrrpGroup(grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses_Address_Vrrp_VrrpGroup, grp vrrpGroupData) {
	ygot.BuildEmptyTree(grpObj)
	ygot.BuildEmptyTree(grpObj.State)

	vrid := grp.vrid
	grpObj.VirtualRouterId = &vrid

	grpObj.Config.VirtualRouterId = &vrid
	grpObj.Config.Priority = &grp.priority
	grpObj.Config.Preempt = &grp.preempt
	grpObj.Config.AdvertisementInterval = &grp.advertisementCenti
	grpObj.Config.VirtualAddress = append(grpObj.Config.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.State.VirtualRouterId = &vrid
	grpObj.State.Priority = &grp.priority
	grpObj.State.Preempt = &grp.preempt
	grpObj.State.PreemptDelay = &grp.preemptDelay
	grpObj.State.AdvertisementInterval = &grp.advertisementCenti
	// Left unset when vrrpd's value is unavailable; see mergeIntfVrrpStateData.
	if grp.hasCurrentPriority {
		grpObj.State.CurrentPriority = &grp.currentPriority
	}
	if grp.hasVrrpState {
		grpObj.State.VrrpState = grp.vrrpState
	}
	grpObj.State.Counters.AdvertisementsSent = grp.advertisementsSent
	grpObj.State.Counters.AdvertisementsReceived = grp.advertisementsReceived
	grpObj.State.Counters.GratuitousArpSent = grp.gratuitousArpSent
	grpObj.State.Counters.StateTransitions = grp.stateTransitions
	grpObj.State.VirtualAddress = append(grpObj.State.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.Config.PreemptDelay = &grp.preemptDelay

	if len(grp.trackInterfaces) > 0 || grp.priorityDecrement > 0 {
		ygot.BuildEmptyTree(grpObj.InterfaceTracking)
		grpObj.InterfaceTracking.Config.TrackInterface = append(grpObj.InterfaceTracking.Config.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.State.TrackInterface = append(grpObj.InterfaceTracking.State.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.Config.PriorityDecrement = &grp.priorityDecrement
		grpObj.InterfaceTracking.State.PriorityDecrement = &grp.priorityDecrement
	}
}

func populateRoutedVlanIPv6VrrpGroup(grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses_Address_Vrrp_VrrpGroup, grp vrrpGroupData) {
	ygot.BuildEmptyTree(grpObj)
	ygot.BuildEmptyTree(grpObj.State)

	vrid := grp.vrid
	grpObj.VirtualRouterId = &vrid

	grpObj.Config.VirtualRouterId = &vrid
	grpObj.Config.Priority = &grp.priority
	grpObj.Config.Preempt = &grp.preempt
	grpObj.Config.PreemptDelay = &grp.preemptDelay
	grpObj.Config.AdvertisementInterval = &grp.advertisementCenti
	grpObj.Config.VirtualAddress = append(grpObj.Config.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.State.VirtualRouterId = &vrid
	grpObj.State.Priority = &grp.priority
	grpObj.State.Preempt = &grp.preempt
	grpObj.State.PreemptDelay = &grp.preemptDelay
	grpObj.State.AdvertisementInterval = &grp.advertisementCenti
	// Left unset when vrrpd's value is unavailable; see mergeIntfVrrpStateData.
	if grp.hasCurrentPriority {
		grpObj.State.CurrentPriority = &grp.currentPriority
	}
	if grp.hasVrrpState {
		grpObj.State.VrrpState = grp.vrrpState
	}
	grpObj.State.Counters.AdvertisementsSent = grp.advertisementsSent
	grpObj.State.Counters.AdvertisementsReceived = grp.advertisementsReceived
	grpObj.State.Counters.NeighborAdvertisementsSent = grp.neighborAdvertisementsSent
	grpObj.State.Counters.StateTransitions = grp.stateTransitions
	grpObj.State.VirtualAddress = append(grpObj.State.VirtualAddress[:0], grp.virtualAddresses...)

	if len(grp.trackInterfaces) > 0 || grp.priorityDecrement > 0 {
		ygot.BuildEmptyTree(grpObj.InterfaceTracking)
		grpObj.InterfaceTracking.Config.TrackInterface = append(grpObj.InterfaceTracking.Config.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.State.TrackInterface = append(grpObj.InterfaceTracking.State.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.Config.PriorityDecrement = &grp.priorityDecrement
		grpObj.InterfaceTracking.State.PriorityDecrement = &grp.priorityDecrement
	}
}

func populateSubIntfIPv4VrrpGroup(grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Vrrp_VrrpGroup, grp vrrpGroupData) {
	ygot.BuildEmptyTree(grpObj)
	ygot.BuildEmptyTree(grpObj.State)

	vrid := grp.vrid
	grpObj.VirtualRouterId = &vrid

	grpObj.Config.VirtualRouterId = &vrid
	grpObj.Config.Priority = &grp.priority
	grpObj.Config.Preempt = &grp.preempt
	grpObj.Config.AdvertisementInterval = &grp.advertisementCenti
	grpObj.Config.VirtualAddress = append(grpObj.Config.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.State.VirtualRouterId = &vrid
	grpObj.State.Priority = &grp.priority
	grpObj.State.Preempt = &grp.preempt
	grpObj.State.PreemptDelay = &grp.preemptDelay
	grpObj.State.AdvertisementInterval = &grp.advertisementCenti
	// Left unset when vrrpd's value is unavailable; see mergeIntfVrrpStateData.
	if grp.hasCurrentPriority {
		grpObj.State.CurrentPriority = &grp.currentPriority
	}
	if grp.hasVrrpState {
		grpObj.State.VrrpState = grp.vrrpState
	}
	grpObj.State.Counters.AdvertisementsSent = grp.advertisementsSent
	grpObj.State.Counters.AdvertisementsReceived = grp.advertisementsReceived
	grpObj.State.Counters.GratuitousArpSent = grp.gratuitousArpSent
	grpObj.State.Counters.StateTransitions = grp.stateTransitions
	grpObj.State.VirtualAddress = append(grpObj.State.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.Config.PreemptDelay = &grp.preemptDelay

	if len(grp.trackInterfaces) > 0 || grp.priorityDecrement > 0 {
		ygot.BuildEmptyTree(grpObj.InterfaceTracking)
		grpObj.InterfaceTracking.Config.TrackInterface = append(grpObj.InterfaceTracking.Config.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.State.TrackInterface = append(grpObj.InterfaceTracking.State.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.Config.PriorityDecrement = &grp.priorityDecrement
		grpObj.InterfaceTracking.State.PriorityDecrement = &grp.priorityDecrement
	}
}

func populateSubIntfIPv6VrrpGroup(grpObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup, grp vrrpGroupData) {
	ygot.BuildEmptyTree(grpObj)
	ygot.BuildEmptyTree(grpObj.State)

	vrid := grp.vrid
	grpObj.VirtualRouterId = &vrid

	grpObj.Config.VirtualRouterId = &vrid
	grpObj.Config.Priority = &grp.priority
	grpObj.Config.Preempt = &grp.preempt
	grpObj.Config.PreemptDelay = &grp.preemptDelay
	grpObj.Config.AdvertisementInterval = &grp.advertisementCenti
	grpObj.Config.VirtualAddress = append(grpObj.Config.VirtualAddress[:0], grp.virtualAddresses...)

	grpObj.State.VirtualRouterId = &vrid
	grpObj.State.Priority = &grp.priority
	grpObj.State.Preempt = &grp.preempt
	grpObj.State.PreemptDelay = &grp.preemptDelay
	grpObj.State.AdvertisementInterval = &grp.advertisementCenti
	// Left unset when vrrpd's value is unavailable; see mergeIntfVrrpStateData.
	if grp.hasCurrentPriority {
		grpObj.State.CurrentPriority = &grp.currentPriority
	}
	if grp.hasVrrpState {
		grpObj.State.VrrpState = grp.vrrpState
	}
	grpObj.State.Counters.AdvertisementsSent = grp.advertisementsSent
	grpObj.State.Counters.AdvertisementsReceived = grp.advertisementsReceived
	grpObj.State.Counters.NeighborAdvertisementsSent = grp.neighborAdvertisementsSent
	grpObj.State.Counters.StateTransitions = grp.stateTransitions
	grpObj.State.VirtualAddress = append(grpObj.State.VirtualAddress[:0], grp.virtualAddresses...)

	if len(grp.trackInterfaces) > 0 || grp.priorityDecrement > 0 {
		ygot.BuildEmptyTree(grpObj.InterfaceTracking)
		grpObj.InterfaceTracking.Config.TrackInterface = append(grpObj.InterfaceTracking.Config.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.State.TrackInterface = append(grpObj.InterfaceTracking.State.TrackInterface[:0], grp.trackInterfaces...)
		grpObj.InterfaceTracking.Config.PriorityDecrement = &grp.priorityDecrement
		grpObj.InterfaceTracking.State.PriorityDecrement = &grp.priorityDecrement
	}
}

func getIntfVrrpData(cfgDb *db.DB, ifName string, targetVrid string, vrrpTbl string, isIPv6 bool) (map[uint8]vrrpGroupData, error) {
	data := make(map[uint8]vrrpGroupData)
	ts := db.TableSpec{Name: vrrpTbl}
	keyPattern := ifName + "|*"
	if targetVrid != "" {
		keyPattern = ifName + "|" + targetVrid
	}

	keys, err := cfgDb.GetKeysByPattern(&ts, keyPattern)
	if err != nil {
		return data, nil
	}

	for _, key := range keys {
		entry, err := cfgDb.GetEntry(&ts, key)
		if err != nil {
			continue
		}

		vrid, ok := getVrrpVrid(entry, key)
		if !ok {
			continue
		}

		data[vrid] = vrrpGroupData{
			vrid:               vrid,
			virtualAddresses:   parseVirtualAddresses(getVrrpVip(entry), isIPv6),
			priority:           parseUint8(entry.Get(VRRP_FIELD_PRIORITY), 100),
			preempt:            getVrrpPreempt(entry),
			preemptDelay:       parseUint16(entry.Get(VRRP_FIELD_PREEMPT_DELAY), 0),
			advertisementCenti: uint16(parseUint8(entry.Get(VRRP_FIELD_ADV_INTERVAL), 1)) * 100,
		}
	}

	return data, nil
}

func mergeIntfVrrpTrackingData(cfgDb *db.DB, ifName string, targetVrid string, trackTbl string, vrrpData map[uint8]vrrpGroupData) error {
	ts := db.TableSpec{Name: trackTbl}
	keyPattern := ifName + "|*"
	if targetVrid != "" {
		keyPattern = ifName + "|" + targetVrid + "|*"
	}

	keys, err := cfgDb.GetKeysByPattern(&ts, keyPattern)
	if err != nil {
		return nil
	}

	for _, key := range keys {
		keyComp := key.Comp
		if len(keyComp) < 3 {
			continue
		}

		vrid, err := strconv.ParseUint(keyComp[1], 10, 8)
		if err != nil {
			continue
		}
		grp, ok := vrrpData[uint8(vrid)]
		if !ok {
			continue
		}

		trackIf := strings.TrimSpace(keyComp[2])
		if trackIf != "" {
			grp.trackInterfaces = append(grp.trackInterfaces, trackIf)
		}

		entry, err := cfgDb.GetEntry(&ts, key)
		if err == nil {
			grp.priorityDecrement = parseUint8(entry.Get(VRRP_FIELD_TRACK_PRIO_INCR), grp.priorityDecrement)
		}

		vrrpData[uint8(vrid)] = grp
	}

	for vrid, grp := range vrrpData {
		if len(grp.trackInterfaces) > 1 {
			sort.Strings(grp.trackInterfaces)
			grp.trackInterfaces = uniqueStrings(grp.trackInterfaces)
		}
		vrrpData[vrid] = grp
	}

	return nil
}

// mergeIntfVrrpStateData overlays the operational state that frrcfgd publishes
// from FRR vrrpd onto the CONFIG_DB-derived groups.
//
// read current-priority when hasCurrentPriority is set. else its invalid current-prio
func mergeIntfVrrpStateData(stateDb *db.DB, ifName string, targetVrid string, stateTbl string, vrrpData map[uint8]vrrpGroupData) error {
	if stateDb == nil {
		return nil
	}

	ts := db.TableSpec{Name: stateTbl}
	keyPattern := ifName + "|*"
	if targetVrid != "" {
		keyPattern = ifName + "|" + targetVrid
	}

	keys, err := stateDb.GetKeysByPattern(&ts, keyPattern)
	if err != nil {
		// The table is absent until the producer publishes for the first time.
		return nil
	}

	for _, key := range keys {
		keyComp := key.Comp
		if len(keyComp) < 2 {
			continue
		}

		vrid, err := strconv.ParseUint(keyComp[1], 10, 8)
		if err != nil {
			continue
		}
		// vrrpd can know about groups CONFIG_DB does not, e.g. mid-delete.
		// The configured set decides what the model exposes.
		grp, ok := vrrpData[uint8(vrid)]
		if !ok {
			continue
		}

		entry, err := stateDb.GetEntry(&ts, key)
		if err != nil {
			continue
		}

		prioStr := strings.TrimSpace(entry.Get(VRRP_STATE_FIELD_EFF_PRIORITY))
		if prioStr == "" {
			continue
		}
		prio, err := strconv.ParseUint(prioStr, 10, 8)
		if err != nil {
			log.Warningf("VRRP_TRACE mergeIntfVrrpStateData: table=%s key=%v has unusable %s=%q",
				stateTbl, keyComp, VRRP_STATE_FIELD_EFF_PRIORITY, prioStr)
			continue
		}

		grp.currentPriority = uint8(prio)
		grp.hasCurrentPriority = true

		stateStr := entry.Get(VRRP_STATE_FIELD_STATE)
		if state, ok := parseVrrpState(stateStr); ok {
			grp.vrrpState = state
			grp.hasVrrpState = true
		} else if strings.TrimSpace(stateStr) != "" {
			log.Warningf(
				"VRRP state table=%s key=%v has unusable %s=%q",
				stateTbl, keyComp, VRRP_STATE_FIELD_STATE, stateStr)
		}

		grp.advertisementsSent = parseVrrpCounter(
			entry, VRRP_STATE_FIELD_ADVERTISEMENTS_TX, stateTbl, keyComp)
		grp.advertisementsReceived = parseVrrpCounter(
			entry, VRRP_STATE_FIELD_ADVERTISEMENTS_RX, stateTbl, keyComp)
		grp.stateTransitions = parseVrrpCounter(
			entry, VRRP_STATE_FIELD_TRANSITIONS, stateTbl, keyComp)
		if stateTbl == VRRP_STATE_TBL {
			grp.gratuitousArpSent = parseVrrpCounter(
				entry, VRRP_STATE_FIELD_GRATUITOUS_ARP_TX, stateTbl, keyComp)
		} else {
			grp.neighborAdvertisementsSent = parseVrrpCounter(
				entry, VRRP_STATE_FIELD_NEIGHBOR_ADVERTISEMENTS_TX, stateTbl, keyComp)
		}

		vrrpData[uint8(vrid)] = grp
	}

	return nil
}

func getVrrpVrid(entry db.Value, dbKey db.Key) (uint8, bool) {
	vridStr := entry.Get(VRRP_FIELD_VRID)
	if vridStr == "" {
		keyComp := dbKey.Comp
		if len(keyComp) < 2 {
			return 0, false
		}
		vridStr = keyComp[1]
	}

	vrid, err := strconv.ParseUint(vridStr, 10, 8)
	if err != nil {
		return 0, false
	}

	return uint8(vrid), true
}

func getVrrpVip(entry db.Value) string {
	vip := entry.Get(VRRP_FIELD_VIP)
	if vip != "" {
		return vip
	}
	// Schema-aligned fallback for environments persisting plain "vip".
	return entry.Get("vip")
}

func getVrrpPreempt(entry db.Value) bool {
	val := strings.TrimSpace(entry.Get(VRRP_FIELD_PREEMPT))
	if val == "" {
		return true
	}
	return parseBool(val, true)
}

func parseVirtualAddresses(vipList string, isIPv6 bool) []string {
	if vipList == "" {
		return []string{}
	}

	parts := strings.Split(vipList, ",")
	addrs := make([]string, 0, len(parts))
	for _, part := range parts {
		if vip, ok := normalizeVirtualAddress(part, isIPv6); ok {
			addrs = append(addrs, vip)
		}
	}

	return uniqueStringsInOrder(addrs)
}

func normalizeVirtualAddressesList(addrs []string, isIPv6 bool) []string {
	if len(addrs) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if vip, ok := normalizeVirtualAddress(addr, isIPv6); ok {
			out = append(out, vip)
		}
	}
	return uniqueStringsInOrder(out)
}

func normalizeVirtualAddress(vip string, isIPv6 bool) (string, bool) {
	vip = strings.TrimSpace(vip)
	if vip == "" {
		return "", false
	}
	if slashIdx := strings.IndexByte(vip, '/'); slashIdx > 0 {
		vip = vip[:slashIdx]
	}
	ip := net.ParseIP(vip)
	if ip == nil {
		return "", false
	}
	if isIPv6 {
		if ip.To4() != nil {
			return "", false
		}
		return vip, true
	}
	if ip.To4() == nil {
		return "", false
	}
	return vip, true
}

func uniqueStrings(list []string) []string {
	if len(list) < 2 {
		return list
	}
	out := make([]string, 0, len(list))
	prev := ""
	for _, item := range list {
		if item == "" || item == prev {
			continue
		}
		out = append(out, item)
		prev = item
	}
	return out
}

func uniqueStringsInOrder(list []string) []string {
	if len(list) < 2 {
		return list
	}
	out := make([]string, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func validateVrrpFieldsVipSubnet(cfgDb *db.DB, ifName string, fields map[string]string) error {
	if fields == nil {
		return nil
	}
	vipList := strings.TrimSpace(fields[VRRP_FIELD_VIP])
	if vipList == "" {
		return nil
	}
	for _, vip := range strings.Split(vipList, ",") {
		vip = strings.TrimSpace(vip)
		if vip == "" {
			continue
		}
		if !checkVipParentSubnet(cfgDb, vip, ifName) {
			return tlerr.InvalidArgs("VIP %s is not in the same subnet of parent interface %s", vip, ifName)
		}
	}
	return nil
}

func checkVipParentSubnet(cfgDb *db.DB, vip, ifName string) bool {
	vip = strings.TrimSpace(vip)
	if slashIdx := strings.IndexByte(vip, '/'); slashIdx > 0 {
		vip = vip[:slashIdx]
	}
	vipAddr := net.ParseIP(vip)
	if vipAddr == nil {
		return false
	}
	vipIsIPv4 := vipAddr.To4() != nil

	for _, parentPref := range getParentIntfIPPrefixes(cfgDb, ifName) {
		parentIP, network, err := net.ParseCIDR(parentPref)
		if err != nil {
			continue
		}
		if (parentIP.To4() != nil) != vipIsIPv4 {
			continue
		}
		if network.Contains(vipAddr) {
			return true
		}
	}
	return false
}

func vrrpIntfTableName(ifName string) string {
	switch {
	case strings.HasPrefix(ifName, "Ethernet"):
		if strings.Contains(ifName, ".") {
			return "VLAN_SUB_INTERFACE"
		}
		return "INTERFACE"
	case strings.HasPrefix(ifName, "PortChannel"):
		if strings.Contains(ifName, ".") {
			return "VLAN_SUB_INTERFACE"
		}
		return "PORTCHANNEL_INTERFACE"
	case strings.HasPrefix(ifName, "Vlan"):
		return "VLAN_INTERFACE"
	case strings.Contains(ifName, "."):
		if strings.HasPrefix(ifName, "Eth") || strings.HasPrefix(ifName, "Po") {
			return "VLAN_SUB_INTERFACE"
		}
	}
	return ""
}

func getParentIntfIPPrefixes(cfgDb *db.DB, ifName string) []string {
	if cfgDb == nil {
		return nil
	}
	tblName := vrrpIntfTableName(ifName)
	if tblName == "" {
		return nil
	}
	ts := db.TableSpec{Name: tblName}
	keys, err := cfgDb.GetKeysByPattern(&ts, ifName+"|*")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key.Comp) < 2 || strings.TrimSpace(key.Comp[0]) != ifName {
			continue
		}
		pref := strings.Join(key.Comp[1:], ":")
		if strings.TrimSpace(pref) != "" {
			out = append(out, pref)
		}
	}
	return out
}

func vrrpAddressConfigured(cfgDb *db.DB, ifName, ip string, isIPv6 bool) bool {
	want := net.ParseIP(strings.TrimSpace(ip))
	if want == nil || (want.To4() == nil) != isIPv6 {
		return false
	}
	for _, pref := range getParentIntfIPPrefixes(cfgDb, ifName) {
		hostIP, _, err := net.ParseCIDR(strings.TrimSpace(pref))
		if err != nil {
			continue
		}
		if hostIP.Equal(want) {
			return true
		}
	}
	return false
}

func validateVrrpAddressKey(cfgDb *db.DB, ifName, ip string, isIPv6 bool) error {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "*" {
		return nil
	}
	if !vrrpAddressConfigured(cfgDb, ifName, ip, isIPv6) {
		return tlerr.InvalidArgs("IP address %s is not configured on interface %s", ip, ifName)
	}
	return nil
}

func vrrpCanonPrefix(pref string) (canon string, network *net.IPNet, isIPv4 bool, ok bool) {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(pref))
	if err != nil {
		return "", nil, false, false
	}
	ones, _ := ipNet.Mask.Size()
	return ip.String() + "/" + strconv.Itoa(ones), ipNet, ip.To4() != nil, true
}

func vrrpOwnedParentPrefixNetwork(vip string, parentPrefixes []string) (*net.IPNet, bool) {
	vipAddr := net.ParseIP(strings.TrimSpace(vip))
	if vipAddr == nil {
		return nil, false
	}
	vipIsIPv4 := vipAddr.To4() != nil

	var best *net.IPNet
	bestLen := -1
	for _, pref := range parentPrefixes {
		_, network, isIPv4, ok := vrrpCanonPrefix(pref)
		if !ok || isIPv4 != vipIsIPv4 {
			continue
		}
		if !network.Contains(vipAddr) {
			continue
		}
		if ones, _ := network.Mask.Size(); ones > bestLen {
			best = network
			bestLen = ones
		}
	}
	return best, best != nil
}

func validateIntfPrefixRemoveBlockedByVrrp(cfgDb *db.DB, ifName, prefix string) error {
	return validateIntfPrefixesRemoveBlockedByVrrp(cfgDb, ifName, []string{prefix})
}

func validateIntfPrefixesRemoveBlockedByVrrp(cfgDb *db.DB, ifName string, prefixes []string) error {
	if cfgDb == nil || ifName == "" || len(prefixes) == 0 {
		return nil
	}

	removed := make(map[string]bool, len(prefixes))
	for _, pref := range prefixes {
		if canon, _, _, ok := vrrpCanonPrefix(pref); ok {
			removed[canon] = true
		}
	}
	if len(removed) == 0 {
		return nil
	}

	parents := getParentIntfIPPrefixes(cfgDb, ifName)
	if len(parents) == 0 {
		return nil
	}

	removing := make(map[string][]string)
	for _, pref := range parents {
		canon, network, _, ok := vrrpCanonPrefix(pref)
		if !ok {
			continue
		}
		if removed[canon] {
			removing[network.String()] = append(removing[network.String()], canon)
		}
	}
	if len(removing) == 0 {
		return nil
	}

	var details []string
	blocking := make(map[string]bool)
	for _, isIPv6 := range []bool{false, true} {
		vrrpTbl, label := VRRP_TBL, "VRRP"
		if isIPv6 {
			vrrpTbl, label = VRRP6_TBL, "VRRP6"
		}
		data, _ := getIntfVrrpData(cfgDb, ifName, "", vrrpTbl, isIPv6)
		vrids := make([]int, 0, len(data))
		for vrid := range data {
			vrids = append(vrids, int(vrid))
		}
		sort.Ints(vrids)

		for _, vrid := range vrids {
			for _, vip := range data[uint8(vrid)].virtualAddresses {
				owner, ok := vrrpOwnedParentPrefixNetwork(vip, parents)
				if !ok {
					continue
				}
				key := owner.String()
				if len(removing[key]) > 0 {
					details = append(details, fmt.Sprintf("%s VIP %s (VR Id: %d)", label, vip, vrid))
					for _, canon := range removing[key] {
						blocking[canon] = true
					}
				}
			}
		}
	}

	if len(details) == 0 {
		return nil
	}
	verb := "is"
	if len(details) > 1 {
		verb = "are"
	}
	blockingPrefixes := make([]string, 0, len(blocking))
	for canon := range blocking {
		blockingPrefixes = append(blockingPrefixes, canon)
	}
	sort.Strings(blockingPrefixes)
	return tlerr.InvalidArgs("Cannot alter interface IP %s on %s: %s %s associated with this prefix.",
		strings.Join(blockingPrefixes, ", "), ifName, strings.Join(details, ", "), verb)
}

func validateIntfIpRemoveBlockedByVrrp(cfgDb *db.DB, ifName string, ipMap map[string]db.Value) error {
	if len(ipMap) == 0 {
		return nil
	}
	prefixes := make([]string, 0, len(ipMap))
	for pref := range ipMap {
		prefixes = append(prefixes, pref)
	}
	sort.Strings(prefixes)
	return validateIntfPrefixesRemoveBlockedByVrrp(cfgDb, ifName, prefixes)
}

func parseUint8(val string, def uint8) uint8 {
	if val == "" {
		return def
	}
	n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 8)
	if err != nil {
		return def
	}
	return uint8(n)
}

func parseUint16(val string, def uint16) uint16 {
	if val == "" {
		return def
	}
	n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 16)
	if err != nil {
		return def
	}
	return uint16(n)
}

func parseBool(val string, def bool) bool {
	if val == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "true", "yes", "enabled", "on":
		return true
	case "false", "no", "disabled", "off":
		return false
	default:
		return def
	}
}

func boolToAdminMode(val bool) string {
	if val {
		return "enabled"
	}
	return "disabled"
}
