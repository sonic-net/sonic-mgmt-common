//////////////////////////////////////////////////////////////////////////
//
// Copyright 2026 Cisco Systems, Inc.
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

// Package transformer implements OpenConfig VRRP transformers for IPv6.
//
// OpenConfig path:
//   interfaces/interface/subinterfaces/subinterface/ipv6/addresses/address/vrrp/vrrp-group
//
// CONFIG_DB mapping:
//   VRRP6|{interface}|{vrid}
//   VRRP6_TRACK|{interface}|{vrid}|{track_if}

package transformer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/openconfig/ygot/ygot"
)

const (
	VRRP6_TABLE       = "VRRP6"
	VRRP6_TRACK_TABLE = "VRRP6_TRACK"

	VRRP6_FIELD_PRIORITY = "priority"
	VRRP6_FIELD_PREEMPT  = "preempt"
	VRRP6_FIELD_VID      = "vid"
	VRRP6_FIELD_VIP      = "vip"
	VRRP6_FIELD_VIP_LIST = "vip@"

	VRRP6_TRACK_FIELD_PRIORITY_INCREMENT = "priority_increment"

	VRRP6_PREEMPT_ENABLED  = "enabled"
	VRRP6_PREEMPT_DISABLED = "disabled"
)

func init() {
	XlateFuncBind("YangToDb_vrrp_group_key_xfmr", YangToDb_vrrp_group_key_xfmr)
	XlateFuncBind("DbToYang_vrrp_group_key_xfmr", DbToYang_vrrp_group_key_xfmr)
	XlateFuncBind("vrrp_table_xfmr", vrrp_table_xfmr)

	XlateFuncBind("YangToDb_vrrp_priority_xfmr", YangToDb_vrrp_priority_xfmr)
	XlateFuncBind("DbToYang_vrrp_priority_xfmr", DbToYang_vrrp_priority_xfmr)

	XlateFuncBind("YangToDb_vrrp_preempt_xfmr", YangToDb_vrrp_preempt_xfmr)
	XlateFuncBind("DbToYang_vrrp_preempt_xfmr", DbToYang_vrrp_preempt_xfmr)

	XlateFuncBind("YangToDb_vrrp_virtual_address_xfmr", YangToDb_vrrp_virtual_address_xfmr)
	XlateFuncBind("DbToYang_vrrp_virtual_address_xfmr", DbToYang_vrrp_virtual_address_xfmr)

	XlateFuncBind("YangToDb_vrrp_virtual_link_local_xfmr", YangToDb_vrrp_virtual_link_local_xfmr)
	XlateFuncBind("DbToYang_vrrp_virtual_link_local_xfmr", DbToYang_vrrp_virtual_link_local_xfmr)

	XlateFuncBind("YangToDb_vrrp_interface_tracking_xfmr", YangToDb_vrrp_interface_tracking_xfmr)
	XlateFuncBind("DbToYang_vrrp_interface_tracking_xfmr", DbToYang_vrrp_interface_tracking_xfmr)
	XlateFuncBind("Subscribe_vrrp_interface_tracking_xfmr", Subscribe_vrrp_interface_tracking_xfmr)

	XlateFuncBind("YangToDb_vrrp_track_interface_xfmr", YangToDb_vrrp_track_interface_xfmr)
	XlateFuncBind("DbToYang_vrrp_track_interface_xfmr", DbToYang_vrrp_track_interface_xfmr)

	XlateFuncBind("YangToDb_vrrp_priority_decrement_xfmr", YangToDb_vrrp_priority_decrement_xfmr)
	XlateFuncBind("DbToYang_vrrp_priority_decrement_xfmr", DbToYang_vrrp_priority_decrement_xfmr)
}

type vrrpContext struct {
	intfName string
	vrid     string
}

func extractVrrpContext(uri string) (vrrpContext, error) {
	pathInfo := NewPathInfo(uri)
	vrid := pathInfo.Var("virtual-router-id")
	if vrid == "" {
		// Accept legacy path key name [vrid=N] in addition to [virtual-router-id=N].
		vrid = pathInfo.Var("vrid")
	}
	ctx := vrrpContext{
		intfName: pathInfo.Var("name"),
		vrid:     vrid,
	}
	if ctx.intfName == "" {
		return ctx, fmt.Errorf("interface name not found in URI: %s", uri)
	}
	if ctx.vrid == "" {
		return ctx, fmt.Errorf("virtual-router-id not found in URI: %s", uri)
	}
	return ctx, nil
}

func buildVrrp6Key(intfName, vrid string) string {
	return fmt.Sprintf("%s|%s", intfName, vrid)
}

func buildVrrp6TrackKey(intfName, vrid, trackIf string) string {
	return fmt.Sprintf("%s|%s|%s", intfName, vrid, trackIf)
}

func isLinkLocalV6(addr string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(addr)), "fe80:")
}

func parseVipField(entry db.Value) []string {
	vipStr, ok := entry.Field[VRRP6_FIELD_VIP_LIST]
	if !ok || vipStr == "" {
		vipStr, ok = entry.Field[VRRP6_FIELD_VIP]
	}
	if !ok || vipStr == "" {
		return nil
	}
	parts := strings.Split(vipStr, ",")
	var addrs []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			addrs = append(addrs, trimmed)
		}
	}
	return addrs
}

func formatVipField(addrs []string) string {
	return strings.Join(addrs, ",")
}

func getVrrp6EntryFromDb(inParams XfmrParams, key string) (db.Value, bool) {
	if inParams.dbDataMap == nil {
		return db.Value{}, false
	}
	data := (*inParams.dbDataMap)[db.ConfigDB]
	if data == nil {
		return db.Value{}, false
	}
	vrrpTable := data[VRRP6_TABLE]
	if vrrpTable == nil {
		return db.Value{}, false
	}
	entry, ok := vrrpTable[key]
	return entry, ok
}

func mergeVipForDb(inParams XfmrParams, key string, mutate func([]string) []string) (map[string]string, error) {
	current := parseVipField(db.Value{})
	if entry, ok := getVrrp6EntryFromDb(inParams, key); ok {
		current = parseVipField(entry)
	}
	updated := mutate(current)
	res := make(map[string]string)
	if len(updated) == 0 {
		res[VRRP6_FIELD_VIP] = ""
		return res, nil
	}
	res[VRRP6_FIELD_VIP] = formatVipField(updated)
	return res, nil
}

func getVrrpIpv6GroupFromYgRoot(inParams XfmrParams, create bool) (*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address_Vrrp_VrrpGroup, vrrpContext, error) {
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, ctx, err
	}

	pathInfo := NewPathInfo(inParams.uri)
	ipAddr := pathInfo.Var("ip")
	if ipAddr == "" {
		return nil, ctx, fmt.Errorf("IPv6 address not found in URI: %s", inParams.uri)
	}

	subifIdx := pathInfo.Var("index")
	if subifIdx == "" {
		subifIdx = "0"
	}
	idx64, err := strconv.ParseUint(subifIdx, 10, 32)
	if err != nil {
		return nil, ctx, fmt.Errorf("invalid subinterface index %s: %v", subifIdx, err)
	}
	subifIndex := uint32(idx64)

	vrid64, err := strconv.ParseUint(ctx.vrid, 10, 8)
	if err != nil {
		return nil, ctx, fmt.Errorf("invalid virtual-router-id %s: %v", ctx.vrid, err)
	}
	vrid := uint8(vrid64)

	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil {
		return nil, ctx, fmt.Errorf("interfaces container missing in payload")
	}

	intfObj, ok := intfsObj.Interface[ctx.intfName]
	if !ok {
		if !create {
			return nil, ctx, nil
		}
		intfObj, _ = intfsObj.NewInterface(ctx.intfName)
	}
	ygot.BuildEmptyTree(intfObj)
	if intfObj.Subinterfaces == nil {
		ygot.BuildEmptyTree(intfObj.Subinterfaces)
	}
	subifObj, ok := intfObj.Subinterfaces.Subinterface[subifIndex]
	if !ok {
		if !create {
			return nil, ctx, nil
		}
		subifObj, _ = intfObj.Subinterfaces.NewSubinterface(subifIndex)
	}
	ygot.BuildEmptyTree(subifObj)
	if subifObj.Ipv6 == nil {
		ygot.BuildEmptyTree(subifObj.Ipv6)
	}
	if subifObj.Ipv6.Addresses == nil {
		ygot.BuildEmptyTree(subifObj.Ipv6.Addresses)
	}
	addrObj, ok := subifObj.Ipv6.Addresses.Address[ipAddr]
	if !ok {
		if !create {
			return nil, ctx, nil
		}
		addrObj, _ = subifObj.Ipv6.Addresses.NewAddress(ipAddr)
	}
	ygot.BuildEmptyTree(addrObj)
	if addrObj.Vrrp == nil {
		ygot.BuildEmptyTree(addrObj.Vrrp)
	}
	vrrpGroup, ok := addrObj.Vrrp.VrrpGroup[vrid]
	if !ok {
		if !create {
			return nil, ctx, nil
		}
		vrrpGroup, err = addrObj.Vrrp.NewVrrpGroup(vrid)
		if err != nil {
			return nil, ctx, err
		}
	}
	ygot.BuildEmptyTree(vrrpGroup)
	return vrrpGroup, ctx, nil
}

var vrrp_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	if strings.Contains(inParams.uri, "interface-tracking") {
		return []string{VRRP6_TRACK_TABLE}, nil
	}
	return []string{VRRP6_TABLE}, nil
}

var YangToDb_vrrp_group_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		if inParams.oper == GET {
			return "", nil
		}
		if inParams.oper == SUBSCRIBE {
			return "*|*", nil
		}
		return "", err
	}
	if inParams.oper == SUBSCRIBE && (ctx.intfName == "*" || ctx.vrid == "*") {
		return fmt.Sprintf("%s|%s", ctx.intfName, ctx.vrid), nil
	}
	return buildVrrp6Key(ctx.intfName, ctx.vrid), nil
}

var DbToYang_vrrp_group_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	keys := strings.Split(inParams.key, "|")
	if len(keys) < 2 {
		return nil, fmt.Errorf("invalid VRRP6 key format: %s", inParams.key)
	}

	pathInfo := NewPathInfo(inParams.uri)
	requestedIntf := pathInfo.Var("name")
	if requestedIntf != "" && requestedIntf != "*" && requestedIntf != keys[0] {
		return nil, fmt.Errorf("VRRP6 entry belongs to interface %s, not requested %s", keys[0], requestedIntf)
	}

	vrid, err := strconv.ParseUint(keys[1], 10, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid virtual-router-id: %s", keys[1])
	}
	rmap["virtual-router-id"] = uint8(vrid)
	return rmap, nil
}

var YangToDb_vrrp_priority_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res := make(map[string]string)
	if inParams.oper == DELETE {
		res[VRRP6_FIELD_PRIORITY] = ""
		return res, nil
	}
	priority, ok := inParams.param.(*uint8)
	if !ok || priority == nil {
		return nil, tlerr.InvalidArgsError{Format: "Invalid priority value"}
	}
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, err
	}
	res[VRRP6_FIELD_PRIORITY] = strconv.FormatUint(uint64(*priority), 10)
	res[VRRP6_FIELD_VID] = ctx.vrid
	return res, nil
}

var DbToYang_vrrp_priority_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	entry, ok := getVrrp6EntryFromDb(inParams, inParams.key)
	if !ok {
		rmap["priority"] = uint8(100)
		return rmap, nil
	}
	priorityStr, ok := entry.Field[VRRP6_FIELD_PRIORITY]
	if !ok || priorityStr == "" {
		rmap["priority"] = uint8(100)
		return rmap, nil
	}
	priority, err := strconv.ParseUint(priorityStr, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid priority value in DB: %s", priorityStr)
	}
	rmap["priority"] = uint8(priority)
	return rmap, nil
}

var YangToDb_vrrp_preempt_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res := make(map[string]string)
	if inParams.oper == DELETE {
		res[VRRP6_FIELD_PREEMPT] = ""
		return res, nil
	}
	preempt, ok := inParams.param.(*bool)
	if !ok || preempt == nil {
		return nil, tlerr.InvalidArgsError{Format: "Invalid preempt value"}
	}
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, err
	}
	if *preempt {
		res[VRRP6_FIELD_PREEMPT] = VRRP6_PREEMPT_ENABLED
	} else {
		res[VRRP6_FIELD_PREEMPT] = VRRP6_PREEMPT_DISABLED
	}
	res[VRRP6_FIELD_VID] = ctx.vrid
	return res, nil
}

var DbToYang_vrrp_preempt_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	entry, ok := getVrrp6EntryFromDb(inParams, inParams.key)
	if !ok {
		rmap["preempt"] = true
		return rmap, nil
	}
	preemptStr, ok := entry.Field[VRRP6_FIELD_PREEMPT]
	if !ok || preemptStr == "" {
		rmap["preempt"] = true
		return rmap, nil
	}
	switch strings.ToLower(preemptStr) {
	case VRRP6_PREEMPT_ENABLED, "true":
		rmap["preempt"] = true
	case VRRP6_PREEMPT_DISABLED, "false":
		rmap["preempt"] = false
	default:
		rmap["preempt"] = true
	}
	return rmap, nil
}

var YangToDb_vrrp_virtual_address_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, err
	}
	dbKey := buildVrrp6Key(ctx.intfName, ctx.vrid)

	if inParams.oper == DELETE {
		return mergeVipForDb(inParams, dbKey, func(current []string) []string {
			var kept []string
			for _, addr := range current {
				if isLinkLocalV6(addr) {
					kept = append(kept, addr)
				}
			}
			return kept
		})
	}

	var globalAddrs []string
	switch v := inParams.param.(type) {
	case *[]string:
		if v != nil {
			for _, addr := range *v {
				if !isLinkLocalV6(addr) {
					globalAddrs = append(globalAddrs, addr)
				}
			}
		}
	case *string:
		if v != nil && *v != "" && !isLinkLocalV6(*v) {
			globalAddrs = append(globalAddrs, *v)
		}
	default:
		return nil, tlerr.InvalidArgsError{Format: "Invalid virtual-address value"}
	}

	return mergeVipForDb(inParams, dbKey, func(current []string) []string {
		var linkLocal []string
		for _, addr := range current {
			if isLinkLocalV6(addr) {
				linkLocal = append(linkLocal, addr)
			}
		}
		merged := append(globalAddrs, linkLocal...)
		return merged
	})
}

var DbToYang_vrrp_virtual_address_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	entry, ok := getVrrp6EntryFromDb(inParams, inParams.key)
	if !ok {
		return rmap, nil
	}
	var globalAddrs []string
	for _, addr := range parseVipField(entry) {
		if !isLinkLocalV6(addr) {
			globalAddrs = append(globalAddrs, addr)
		}
	}
	if len(globalAddrs) > 0 {
		rmap["virtual-address"] = globalAddrs
	}
	return rmap, nil
}

var YangToDb_vrrp_virtual_link_local_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, err
	}
	dbKey := buildVrrp6Key(ctx.intfName, ctx.vrid)

	if inParams.oper == DELETE {
		return mergeVipForDb(inParams, dbKey, func(current []string) []string {
			var kept []string
			for _, addr := range current {
				if !isLinkLocalV6(addr) {
					kept = append(kept, addr)
				}
			}
			return kept
		})
	}

	linkLocal, ok := inParams.param.(*string)
	if !ok || linkLocal == nil || *linkLocal == "" {
		return nil, tlerr.InvalidArgsError{Format: "Invalid virtual-link-local value"}
	}

	return mergeVipForDb(inParams, dbKey, func(current []string) []string {
		var globalAddrs []string
		for _, addr := range current {
			if !isLinkLocalV6(addr) {
				globalAddrs = append(globalAddrs, addr)
			}
		}
		merged := append(globalAddrs, *linkLocal)
		return merged
	})
}

var DbToYang_vrrp_virtual_link_local_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	entry, ok := getVrrp6EntryFromDb(inParams, inParams.key)
	if !ok {
		return rmap, nil
	}
	for _, addr := range parseVipField(entry) {
		if isLinkLocalV6(addr) {
			rmap["virtual-link-local"] = addr
			break
		}
	}
	return rmap, nil
}

func collectVrrpTrackEntries(inParams XfmrParams, ctx vrrpContext) ([]string, uint8) {
	var trackInterfaces []string
	var priorityDecrement uint8

	if inParams.dbDataMap == nil {
		return trackInterfaces, priorityDecrement
	}
	data := (*inParams.dbDataMap)[db.ConfigDB]
	if data == nil {
		return trackInterfaces, priorityDecrement
	}
	trackTable := data[VRRP6_TRACK_TABLE]
	if trackTable == nil {
		return trackInterfaces, priorityDecrement
	}

	keyPrefix := buildVrrp6TrackKey(ctx.intfName, ctx.vrid, "")
	for key, entry := range trackTable {
		if !strings.HasPrefix(key, keyPrefix) {
			continue
		}
		parts := strings.Split(key, "|")
		if len(parts) < 3 {
			continue
		}
		trackInterfaces = append(trackInterfaces, parts[2])
		if decrementStr, ok := entry.Field[VRRP6_TRACK_FIELD_PRIORITY_INCREMENT]; ok && decrementStr != "" {
			if dec, err := strconv.ParseUint(decrementStr, 10, 8); err == nil {
				priorityDecrement = uint8(dec)
			}
		}
	}
	return trackInterfaces, priorityDecrement
}

func populateVrrpTrackingYang(inParams XfmrParams, ctx vrrpContext, trackInterfaces []string, priorityDecrement uint8) error {
	vrrpGroup, _, err := getVrrpIpv6GroupFromYgRoot(inParams, true)
	if err != nil || vrrpGroup == nil {
		return err
	}
	if vrrpGroup.InterfaceTracking == nil {
		ygot.BuildEmptyTree(vrrpGroup.InterfaceTracking)
	}
	if vrrpGroup.InterfaceTracking.Config == nil {
		ygot.BuildEmptyTree(vrrpGroup.InterfaceTracking.Config)
	}
	if vrrpGroup.InterfaceTracking.State == nil {
		ygot.BuildEmptyTree(vrrpGroup.InterfaceTracking.State)
	}

	if len(trackInterfaces) > 0 {
		vrrpGroup.InterfaceTracking.Config.TrackInterface = trackInterfaces
		vrrpGroup.InterfaceTracking.State.TrackInterface = trackInterfaces
	}
	if priorityDecrement > 0 {
		vrrpGroup.InterfaceTracking.Config.PriorityDecrement = &priorityDecrement
		vrrpGroup.InterfaceTracking.State.PriorityDecrement = &priorityDecrement
	}
	return nil
}

var YangToDb_vrrp_interface_tracking_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	res := make(map[string]map[string]db.Value)
	res[VRRP6_TRACK_TABLE] = make(map[string]db.Value)

	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil, err
	}

	if inParams.oper == DELETE {
		return res, nil
	}

	vrrpGroup, _, err := getVrrpIpv6GroupFromYgRoot(inParams, false)
	if err != nil {
		return nil, err
	}
	if vrrpGroup == nil || vrrpGroup.InterfaceTracking == nil || vrrpGroup.InterfaceTracking.Config == nil {
		return res, nil
	}

	cfg := vrrpGroup.InterfaceTracking.Config
	priorityDec := uint8(0)
	if cfg.PriorityDecrement != nil {
		priorityDec = *cfg.PriorityDecrement
	}
	for _, trackIf := range cfg.TrackInterface {
		trackKey := buildVrrp6TrackKey(ctx.intfName, ctx.vrid, trackIf)
		res[VRRP6_TRACK_TABLE][trackKey] = db.Value{Field: map[string]string{
			VRRP6_TRACK_FIELD_PRIORITY_INCREMENT: strconv.FormatUint(uint64(priorityDec), 10),
		}}
	}
	return res, nil
}

var DbToYang_vrrp_interface_tracking_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return nil
	}
	trackInterfaces, priorityDecrement := collectVrrpTrackEntries(inParams, ctx)
	if len(trackInterfaces) == 0 && priorityDecrement == 0 {
		return nil
	}
	return populateVrrpTrackingYang(inParams, ctx, trackInterfaces, priorityDecrement)
}

var YangToDb_vrrp_track_interface_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	// track-interface is mapped via the interface-tracking subtree transformer.
	return map[string]string{}, nil
}

var DbToYang_vrrp_track_interface_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return rmap, nil
	}
	trackInterfaces, _ := collectVrrpTrackEntries(inParams, ctx)
	if len(trackInterfaces) > 0 {
		rmap["track-interface"] = trackInterfaces
	}
	return rmap, nil
}

var YangToDb_vrrp_priority_decrement_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	// priority-decrement is mapped via the interface-tracking subtree transformer.
	return map[string]string{}, nil
}

var DbToYang_vrrp_priority_decrement_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		return rmap, nil
	}
	_, priorityDecrement := collectVrrpTrackEntries(inParams, ctx)
	rmap["priority-decrement"] = priorityDecrement
	return rmap, nil
}

var Subscribe_vrrp_interface_tracking_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	result.dbDataMap[db.ConfigDB] = make(map[string]map[string]map[string]string)
	result.dbDataMap[db.ConfigDB][VRRP6_TRACK_TABLE] = make(map[string]map[string]string)

	ctx, err := extractVrrpContext(inParams.uri)
	if err != nil {
		result.dbDataMap[db.ConfigDB][VRRP6_TRACK_TABLE]["*"] = map[string]string{
			VRRP6_TRACK_FIELD_PRIORITY_INCREMENT: VRRP6_TRACK_FIELD_PRIORITY_INCREMENT,
		}
	} else {
		keyPattern := fmt.Sprintf("%s|%s|*", ctx.intfName, ctx.vrid)
		result.dbDataMap[db.ConfigDB][VRRP6_TRACK_TABLE][keyPattern] = map[string]string{
			VRRP6_TRACK_FIELD_PRIORITY_INCREMENT: VRRP6_TRACK_FIELD_PRIORITY_INCREMENT,
		}
	}

	result.needCache = false
	result.onChange = OnchangeEnable
	return result, nil
}
