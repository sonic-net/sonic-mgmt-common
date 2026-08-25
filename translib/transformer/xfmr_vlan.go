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
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/openconfig/ygot/ygot"
)

type ocVlanStatus = ocbinds.E_OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_Vlans_Vlan_Config_Status

const (
	ocVlansPath       = "/openconfig-vlan:vlans"
	ocVlanConfigPath  = "/openconfig-vlan:vlans/vlan/config"
	ocVlanStatePath   = "/openconfig-vlan:vlans/vlan/state"
	ocVlanMembersPath = "/openconfig-vlan:vlans/vlan/members"

	minSonicVlanId = 2
	maxSonicVlanId = 4094

	ocVlanStatusUnset     ocVlanStatus = ocbinds.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_Vlans_Vlan_Config_Status_UNSET
	ocVlanStatusActive    ocVlanStatus = ocbinds.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_Vlans_Vlan_Config_Status_ACTIVE
	ocVlanStatusSuspended ocVlanStatus = ocbinds.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_Vlans_Vlan_Config_Status_SUSPENDED
)

func init() {
	XlateFuncBind("YangToDb_oc_vlans_xfmr", YangToDb_oc_vlans_xfmr)
	XlateFuncBind("DbToYang_oc_vlans_xfmr", DbToYang_oc_vlans_xfmr)
	XlateFuncBind("Subscribe_oc_vlans_xfmr", Subscribe_oc_vlans_xfmr)
}

func sonicVlanName(vlanId uint16) string {
	return "Vlan" + strconv.Itoa(int(vlanId))
}

func validSonicVlanId(vlanId uint16) bool {
	return vlanId >= minSonicVlanId && vlanId <= maxSonicVlanId
}

func parseVlanIdFromName(vlanName string) (uint16, bool) {
	if !strings.HasPrefix(vlanName, "Vlan") {
		return 0, false
	}
	id, err := strconv.Atoi(vlanName[len("Vlan"):])
	if err != nil || id < 0 || id > 4095 {
		return 0, false
	}
	return uint16(id), true
}

func vlanIdFromEntry(vlanName string, entry db.Value) (uint16, bool) {
	if entry.Field != nil {
		if raw, ok := entry.Field["vlanid"]; ok && raw != "" {
			id, err := strconv.Atoi(raw)
			if err == nil && id >= 0 && id <= 4095 {
				return uint16(id), true
			}
		}
	}
	return parseVlanIdFromName(vlanName)
}

func validateSonicVlanId(vlanId uint16) error {
	if !validSonicVlanId(vlanId) {
		return tlerr.InvalidArgsError{Format: "Invalid VLAN id " + strconv.Itoa(int(vlanId)) + ": SONiC supports VLAN ids 2-4094"}
	}
	return nil
}

func validateOcVlanName(vlanId uint16, name *string) error {
	if name == nil || *name == "" {
		return nil
	}
	expected := sonicVlanName(vlanId)
	if *name != expected {
		return tlerr.InvalidArgsError{Format: "VLAN name must be " + expected}
	}
	return nil
}

func ocVlansCfgDb(inParams XfmrParams) *db.DB {
	if inParams.dbs[db.ConfigDB] != nil {
		return inParams.dbs[db.ConfigDB]
	}
	return inParams.d
}

func ocVlansTargetPath(inParams XfmrParams) string {
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	if targetUriPath == "" {
		targetUriPath, _, _ = XfmrRemoveXPATHPredicates(inParams.uri)
	}
	return targetUriPath
}

func ocVlanIdFromParams(inParams XfmrParams) (uint16, bool, error) {
	reqVidStr := NewPathInfo(inParams.uri).Var("vlan-id")
	if reqVidStr == "" {
		reqVidStr = NewPathInfo(inParams.requestUri).Var("vlan-id")
	}
	if reqVidStr == "" {
		return 0, false, nil
	}
	id, err := strconv.Atoi(reqVidStr)
	if err != nil || id < 0 {
		return 0, false, tlerr.InvalidArgsError{Format: "Invalid VLAN id"}
	}
	return uint16(id), true, nil
}

type ocVlanFill struct {
	config     bool
	state      bool
	members    bool
	configLeaf string
	stateLeaf  string
}

func ocVlanFillFromPath(targetUriPath string) ocVlanFill {
	switch {
	case targetUriPath == ocVlanConfigPath+"/vlan-id":
		return ocVlanFill{config: true, configLeaf: "vlan-id"}
	case targetUriPath == ocVlanConfigPath+"/name":
		return ocVlanFill{config: true, configLeaf: "name"}
	case targetUriPath == ocVlanConfigPath+"/status":
		return ocVlanFill{config: true, configLeaf: "status"}
	case targetUriPath == ocVlanStatePath+"/vlan-id":
		return ocVlanFill{state: true, stateLeaf: "vlan-id"}
	case targetUriPath == ocVlanStatePath+"/name":
		return ocVlanFill{state: true, stateLeaf: "name"}
	case targetUriPath == ocVlanStatePath+"/status":
		return ocVlanFill{state: true, stateLeaf: "status"}
	case targetUriPath == ocVlanConfigPath:
		return ocVlanFill{config: true}
	case targetUriPath == ocVlanStatePath:
		return ocVlanFill{state: true}
	case strings.HasPrefix(targetUriPath, ocVlanMembersPath):
		return ocVlanFill{members: true}
	default:
		return ocVlanFill{config: true, state: true, members: true}
	}
}

func ocVlansReadOnlyPath(targetUriPath string) bool {
	return strings.HasPrefix(targetUriPath, ocVlanStatePath) || strings.HasPrefix(targetUriPath, ocVlanMembersPath)
}

func shouldFillLeaf(requested, leaf string) bool {
	return requested == "" || requested == leaf
}

func listConfigDbVlanKeys(cfgDb *db.DB) ([]string, error) {
	if cfgDb == nil {
		return nil, nil
	}
	keys, err := cfgDb.GetKeys(&db.TableSpec{Name: VLAN_TN})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key.Comp) == 0 {
			continue
		}
		name := key.Get(0)
		if _, ok := parseVlanIdFromName(name); ok {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		idi, _ := parseVlanIdFromName(names[i])
		idj, _ := parseVlanIdFromName(names[j])
		return idi < idj
	})
	return names, nil
}

func listVlanMembers(cfgDb *db.DB, vlanName string) ([]string, error) {
	if cfgDb == nil {
		return nil, nil
	}
	keys, err := cfgDb.GetKeys(&db.TableSpec{Name: VLAN_MEMBER_TN})
	if err != nil {
		return nil, err
	}
	members := make([]string, 0)
	seen := make(map[string]struct{})
	for _, key := range keys {
		if len(key.Comp) < 2 {
			continue
		}
		if key.Get(0) != vlanName {
			continue
		}
		ifName := key.Get(1)
		if ifName == "" {
			continue
		}
		if _, ok := seen[ifName]; ok {
			continue
		}
		seen[ifName] = struct{}{}
		members = append(members, ifName)
	}
	sort.Strings(members)
	return members, nil
}

func vlanEntryExists(cfgDb *db.DB, vlanName string) bool {
	if cfgDb == nil {
		return false
	}
	entry, err := cfgDb.GetEntry(&db.TableSpec{Name: VLAN_TN}, db.Key{Comp: []string{vlanName}})
	return err == nil && entry.IsPopulated()
}

func vlanStatusFromAdmin(admin string) ocVlanStatus {
	if strings.EqualFold(admin, "down") {
		return ocVlanStatusSuspended
	}
	return ocVlanStatusActive
}

func adminStatusFromOc(status ocVlanStatus) string {
	if status == ocVlanStatusSuspended {
		return "down"
	}
	return "up"
}

func ocVlanDbFields(vlanId uint16, vlanObj *ocbinds.OpenconfigVlan_Vlans_Vlan, oper Operation, targetUriPath string, cfgDb *db.DB) (map[string]string, error) {
	if err := validateSonicVlanId(vlanId); err != nil {
		return nil, err
	}

	var cfg *ocbinds.OpenconfigVlan_Vlans_Vlan_Config
	if vlanObj != nil {
		cfg = vlanObj.Config
	}
	if cfg != nil {
		if cfg.VlanId != nil && *cfg.VlanId != vlanId {
			return nil, tlerr.InvalidArgsError{Format: "VLAN id in payload does not match " + strconv.Itoa(int(vlanId))}
		}
		if err := validateOcVlanName(vlanId, cfg.Name); err != nil {
			return nil, err
		}
	}

	fields := map[string]string{
		"vlanid": strconv.Itoa(int(vlanId)),
	}

	statusSpecified := cfg != nil && cfg.Status != ocVlanStatusUnset
	statusLeaf := targetUriPath == ocVlanConfigPath+"/status"
	exists := vlanEntryExists(cfgDb, sonicVlanName(vlanId))
	if oper == REPLACE || !exists || statusSpecified || statusLeaf {
		admin := "up"
		if statusSpecified {
			admin = adminStatusFromOc(cfg.Status)
		}
		fields["admin_status"] = admin
	}
	return fields, nil
}

func getOcVlansRoot(s *ygot.GoStruct) *ocbinds.OpenconfigVlan_Vlans {
	deviceObj := (*s).(*ocbinds.Device)
	if deviceObj.Vlans == nil {
		ygot.BuildEmptyTree(deviceObj)
	}
	if deviceObj.Vlans == nil {
		deviceObj.Vlans = &ocbinds.OpenconfigVlan_Vlans{}
	}
	return deviceObj.Vlans
}

func fillOcVlanConfig(cfg *ocbinds.OpenconfigVlan_Vlans_Vlan_Config, vlanName string, vlanId uint16, entry db.Value, leaf string) {
	if shouldFillLeaf(leaf, "vlan-id") {
		id := vlanId
		cfg.VlanId = &id
	}
	if shouldFillLeaf(leaf, "name") {
		name := vlanName
		cfg.Name = &name
	}
	if shouldFillLeaf(leaf, "status") {
		admin := ""
		if entry.Field != nil {
			admin = entry.Field["admin_status"]
		}
		cfg.Status = vlanStatusFromAdmin(admin)
	}
}

func fillOcVlanState(st *ocbinds.OpenconfigVlan_Vlans_Vlan_State, vlanName string, vlanId uint16, entry db.Value, leaf string) {
	if shouldFillLeaf(leaf, "vlan-id") {
		id := vlanId
		st.VlanId = &id
	}
	if shouldFillLeaf(leaf, "name") {
		name := vlanName
		st.Name = &name
	}
	if shouldFillLeaf(leaf, "status") {
		admin := ""
		if entry.Field != nil {
			admin = entry.Field["admin_status"]
		}
		st.Status = vlanStatusFromAdmin(admin)
	}
}

func fillOcVlanMembers(vlanObj *ocbinds.OpenconfigVlan_Vlans_Vlan, members []string) {
	if vlanObj.Members == nil {
		vlanObj.Members = &ocbinds.OpenconfigVlan_Vlans_Vlan_Members{}
	}
	for _, ifName := range members {
		ifNameCopy := ifName
		member := &ocbinds.OpenconfigVlan_Vlans_Vlan_Members_Member{
			State: &ocbinds.OpenconfigVlan_Vlans_Vlan_Members_Member_State{
				Interface: &ifNameCopy,
			},
		}
		vlanObj.Members.Member = append(vlanObj.Members.Member, member)
	}
}

func populateOcVlan(vlans *ocbinds.OpenconfigVlan_Vlans, vlanName string, vlanId uint16, entry db.Value, members []string, fill ocVlanFill) error {
	vlanObj, ok := vlans.Vlan[vlanId]
	if !ok {
		var err error
		vlanObj, err = vlans.NewVlan(vlanId)
		if err != nil {
			return err
		}
	}
	if fill.config {
		if vlanObj.Config == nil {
			vlanObj.Config = &ocbinds.OpenconfigVlan_Vlans_Vlan_Config{}
		}
		fillOcVlanConfig(vlanObj.Config, vlanName, vlanId, entry, fill.configLeaf)
	}
	if fill.state {
		if vlanObj.State == nil {
			vlanObj.State = &ocbinds.OpenconfigVlan_Vlans_Vlan_State{}
		}
		fillOcVlanState(vlanObj.State, vlanName, vlanId, entry, fill.stateLeaf)
	}
	if fill.members {
		fillOcVlanMembers(vlanObj, members)
	}
	return nil
}

func ocVlansYangToDbWrite(inParams XfmrParams, cfgDb *db.DB, targetUriPath string) (map[string]map[string]db.Value, error) {
	vlans := getOcVlansRoot(inParams.ygRoot)
	payload := map[uint16]*ocbinds.OpenconfigVlan_Vlans_Vlan{}
	if vlans != nil && vlans.Vlan != nil {
		payload = vlans.Vlan
	}

	reqVid, haveReqVid, err := ocVlanIdFromParams(inParams)
	if err != nil {
		return nil, err
	}

	vlanMap := make(map[string]db.Value)
	if haveReqVid {
		if err = validateSonicVlanId(reqVid); err != nil {
			return nil, err
		}
		fields, fieldErr := ocVlanDbFields(reqVid, payload[reqVid], inParams.oper, targetUriPath, cfgDb)
		if fieldErr != nil {
			return nil, fieldErr
		}
		vlanMap[sonicVlanName(reqVid)] = db.Value{Field: fields}
		return map[string]map[string]db.Value{VLAN_TN: vlanMap}, nil
	}

	if len(payload) == 0 {
		if inParams.oper == REPLACE {
			return ocVlansYangToDbDelete(inParams, cfgDb, targetUriPath)
		}
		return map[string]map[string]db.Value{}, nil
	}

	for vlanId, vlanObj := range payload {
		if err = validateSonicVlanId(vlanId); err != nil {
			return nil, err
		}
		fields, fieldErr := ocVlanDbFields(vlanId, vlanObj, inParams.oper, targetUriPath, cfgDb)
		if fieldErr != nil {
			return nil, fieldErr
		}
		vlanMap[sonicVlanName(vlanId)] = db.Value{Field: fields}
	}

	if inParams.oper == REPLACE && inParams.subOpDataMap != nil {
		existing, listErr := listConfigDbVlanKeys(cfgDb)
		if listErr != nil {
			return nil, listErr
		}
		delVlanMap := make(map[string]db.Value)
		delMemberMap := make(map[string]db.Value)
		for _, vlanName := range existing {
			vlanId, ok := parseVlanIdFromName(vlanName)
			if !ok {
				continue
			}
			if _, keep := payload[vlanId]; keep {
				continue
			}
			addVlanDeleteKeys(cfgDb, vlanName, delVlanMap, delMemberMap)
		}
		if len(delVlanMap) > 0 || len(delMemberMap) > 0 {
			updateSubOpDataMap(map[db.DBNum]map[string]map[string]db.Value{
				db.ConfigDB: ocVlansDeleteResult(delVlanMap, delMemberMap),
			}, DELETE, inParams)
		}
	}

	return map[string]map[string]db.Value{VLAN_TN: vlanMap}, nil
}

func addVlanDeleteKeys(cfgDb *db.DB, vlanName string, vlanMap map[string]db.Value, memberMap map[string]db.Value) {
	vlanMap[vlanName] = db.Value{}
	members, err := listVlanMembers(cfgDb, vlanName)
	if err != nil {
		return
	}
	for _, ifName := range members {
		memberMap[vlanName+"|"+ifName] = db.Value{}
	}
}

func ocVlansDeleteResult(vlanMap map[string]db.Value, memberMap map[string]db.Value) map[string]map[string]db.Value {
	result := make(map[string]map[string]db.Value)
	if len(vlanMap) > 0 {
		result[VLAN_TN] = vlanMap
	}
	if len(memberMap) > 0 {
		result[VLAN_MEMBER_TN] = memberMap
	}
	return result
}

func ocVlansYangToDbDelete(inParams XfmrParams, cfgDb *db.DB, targetUriPath string) (map[string]map[string]db.Value, error) {
	switch targetUriPath {
	case ocVlanConfigPath + "/vlan-id", ocVlanConfigPath + "/name":
		return nil, tlerr.NotSupported("DELETE not supported on " + targetUriPath)
	}

	reqVid, haveReqVid, err := ocVlanIdFromParams(inParams)
	if err != nil {
		return nil, err
	}

	if targetUriPath == ocVlanConfigPath+"/status" {
		if !haveReqVid {
			return nil, tlerr.InvalidArgsError{Format: "VLAN id is required to delete status"}
		}
		if err = validateSonicVlanId(reqVid); err != nil {
			return nil, err
		}
		if inParams.subOpDataMap != nil {
			subOpMap := map[db.DBNum]map[string]map[string]db.Value{
				db.ConfigDB: {
					VLAN_TN: {
						sonicVlanName(reqVid): {Field: map[string]string{"admin_status": "up"}},
					},
				},
			}
			updateSubOpDataMap(subOpMap, UPDATE, inParams)
		}
		return map[string]map[string]db.Value{}, nil
	}

	vlanMap := make(map[string]db.Value)
	memberMap := make(map[string]db.Value)
	if haveReqVid {
		if err = validateSonicVlanId(reqVid); err != nil {
			return nil, err
		}
		addVlanDeleteKeys(cfgDb, sonicVlanName(reqVid), vlanMap, memberMap)
		return ocVlansDeleteResult(vlanMap, memberMap), nil
	}

	vlanNames, listErr := listConfigDbVlanKeys(cfgDb)
	if listErr != nil {
		return nil, listErr
	}
	for _, vlanName := range vlanNames {
		addVlanDeleteKeys(cfgDb, vlanName, vlanMap, memberMap)
	}
	return ocVlansDeleteResult(vlanMap, memberMap), nil
}

var YangToDb_oc_vlans_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}

	targetUriPath := ocVlansTargetPath(inParams)
	if ocVlansReadOnlyPath(targetUriPath) {
		return nil, tlerr.NotSupported("SET not supported on " + targetUriPath)
	}

	cfgDb := ocVlansCfgDb(inParams)
	switch inParams.oper {
	case DELETE:
		return ocVlansYangToDbDelete(inParams, cfgDb, targetUriPath)
	case CREATE, UPDATE, REPLACE:
		return ocVlansYangToDbWrite(inParams, cfgDb, targetUriPath)
	default:
		return nil, tlerr.NotSupported("Unsupported operation on " + ocVlansPath)
	}
}

var DbToYang_oc_vlans_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	cfgDb := ocVlansCfgDb(inParams)
	if cfgDb == nil {
		return tlerr.InternalError{Format: "CONFIG_DB handle is not available for VLAN catalog GET"}
	}

	vlans := getOcVlansRoot(inParams.ygRoot)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.uri)
	if targetUriPath == "" {
		targetUriPath, _, _ = XfmrRemoveXPATHPredicates(inParams.requestUri)
	}
	fill := ocVlanFillFromPath(targetUriPath)
	reqVidStr := NewPathInfo(inParams.uri).Var("vlan-id")
	if reqVidStr == "" {
		reqVidStr = NewPathInfo(inParams.requestUri).Var("vlan-id")
	}

	vlanNames, err := listConfigDbVlanKeys(cfgDb)
	if err != nil {
		return err
	}

	var reqVid uint16
	var haveReqVid bool
	if reqVidStr != "" {
		id, convErr := strconv.Atoi(reqVidStr)
		if convErr != nil {
			return tlerr.NotFoundError{Format: "Resource not found"}
		}
		reqVid = uint16(id)
		haveReqVid = true
	}

	matched := false
	for _, vlanName := range vlanNames {
		entry, getErr := cfgDb.GetEntry(&db.TableSpec{Name: VLAN_TN}, db.Key{Comp: []string{vlanName}})
		if getErr != nil {
			if tlerr.IsTranslibRedisClientEntryNotExist(getErr) {
				continue
			}
			return getErr
		}
		vlanId, ok := vlanIdFromEntry(vlanName, entry)
		if !ok {
			continue
		}
		if haveReqVid && vlanId != reqVid {
			continue
		}
		members := []string{}
		if fill.members {
			members, err = listVlanMembers(cfgDb, vlanName)
			if err != nil {
				return err
			}
		}
		if err = populateOcVlan(vlans, vlanName, vlanId, entry, members, fill); err != nil {
			return err
		}
		matched = true
	}

	if haveReqVid && !matched {
		return tlerr.NotFoundError{Format: "Resource not found"}
	}
	return nil
}

var Subscribe_oc_vlans_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)

	pathInfo := NewPathInfo(inParams.uri)
	vlanId := pathInfo.Var("vlan-id")
	if vlanId == "" {
		vlanId = "*"
	}

	if inParams.subscProc == TRANSLATE_EXISTS {
		result.isVirtualTbl = true
		return result, nil
	}

	result.onChange = OnchangeEnable
	result.nOpts = &notificationOpts{pType: OnChange}

	if vlanId != "*" {
		result.isVirtualTbl = true
		return result, nil
	}

	result.dbDataMap[db.ConfigDB] = map[string]map[string]map[string]string{
		VLAN_TN: {vlanId: {}},
	}
	return result, nil
}
