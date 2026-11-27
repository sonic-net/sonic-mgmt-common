////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Nokia.                                                     //
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

package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
)

const (
	DNS_OPTIONS_TABLE      = "DNS_OPTIONS"
	DNS_OPTIONS_GLOBAL_KEY = "GLOBAL"

	dnsSearchLeaf  = "search"
	dnsSearchField = "search@"
)

func init() {
	XlateFuncBind("YangToDb_dns_config_xfmr", YangToDb_dns_config_xfmr)
	XlateFuncBind("DbToYang_dns_config_xfmr", DbToYang_dns_config_xfmr)
	XlateFuncBind("Subscribe_dns_config_xfmr", Subscribe_dns_options_xfmr)
	XlateFuncBind("DbToYang_dns_state_xfmr", DbToYang_dns_state_xfmr)
	XlateFuncBind("Subscribe_dns_state_xfmr", Subscribe_dns_options_xfmr)
}

// dnsSearch returns the search domains in DNS_OPTIONS|GLOBAL, or nil if
// the row or its search@ field is not configured.
func dnsSearch(d *db.DB) []string {
	entry, err := d.GetEntry(&db.TableSpec{Name: DNS_OPTIONS_TABLE}, db.Key{Comp: []string{DNS_OPTIONS_GLOBAL_KEY}})
	if err != nil {
		return nil
	}
	search := entry.GetList(dnsSearchField)
	if len(search) == 0 {
		return nil
	}
	return search
}

// dnsSearchConfigured reports whether CONFIG_DB holds a search list. Without
// a DB handle it answers true, so the caller still emits its delete.
func dnsSearchConfigured(inParams XfmrParams) bool {
	d := inParams.d
	if d == nil {
		d = inParams.dbs[db.ConfigDB]
	}
	if d == nil {
		return true
	}
	return dnsSearch(d) != nil
}

// dnsOptionsSubOp queues a DNS_OPTIONS|GLOBAL change alongside the request.
func dnsOptionsSubOp(inParams XfmrParams, oper Operation, val db.Value) {
	if inParams.subOpDataMap[oper] == nil {
		subOpMap := make(RedisDbMap)
		inParams.subOpDataMap[oper] = &subOpMap
	}
	subOpMap := *inParams.subOpDataMap[oper]
	if subOpMap[db.ConfigDB] == nil {
		subOpMap[db.ConfigDB] = make(map[string]map[string]db.Value)
	}
	if subOpMap[db.ConfigDB][DNS_OPTIONS_TABLE] == nil {
		subOpMap[db.ConfigDB][DNS_OPTIONS_TABLE] = make(map[string]db.Value)
	}
	subOpMap[db.ConfigDB][DNS_OPTIONS_TABLE][DNS_OPTIONS_GLOBAL_KEY] = val
}

// YangToDb_dns_config_xfmr maps /system/dns/config/search to the CONFIG_DB
// DNS_OPTIONS|GLOBAL search@ leaf-list, which hostcfgd renders as the
// resolv.conf search line. The row's other fields (ndots, timeout, attempts)
// have no OpenConfig leaf and are never written here.
//
// The common app merges a search@ update into the existing list and swaps it
// on a north-bound replace, so this only has to emit the requested domains.
var YangToDb_dns_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	res := make(map[string]map[string]db.Value)
	log.V(3).Infof("YangToDb_dns_config_xfmr: oper %v, request %s", inParams.oper, inParams.requestUri)

	if inParams.oper == DELETE {
		// The common app queues nothing for a leaf-list field missing from the
		// row, and rejects a request that queues nothing, so a delete with no
		// list to remove has to produce no row at all.
		if !dnsSearchConfigured(inParams) {
			return res, nil
		}
		// An empty value clears the whole list; a value names the one domain
		// a leaf-list instance delete removes.
		val := db.Value{Field: map[string]string{
			dnsSearchField: NewPathInfo(inParams.requestUri).Var(dnsSearchLeaf),
		}}
		res[DNS_OPTIONS_TABLE] = map[string]db.Value{DNS_OPTIONS_GLOBAL_KEY: val}
		return res, nil
	}

	var search []string
	if sys := getAppRootObject(inParams); sys != nil && sys.Dns != nil && sys.Dns.Config != nil {
		search = sys.Dns.Config.Search
	}

	if inParams.oper == REPLACE {
		// A replace result row is written with SetEntry when the target is the
		// config container, which would drop the row's other fields. An update
		// sub-operation is written with ModEntry and, under a north-bound
		// replace, still swaps the leaf-list instead of merging it.
		if len(search) == 0 {
			if dnsSearchConfigured(inParams) {
				dnsOptionsSubOp(inParams, DELETE, db.Value{Field: map[string]string{dnsSearchField: ""}})
			}
		} else {
			val := db.Value{Field: map[string]string{}}
			val.SetList(dnsSearchField, search)
			dnsOptionsSubOp(inParams, UPDATE, val)
		}
		return res, nil
	}

	if len(search) == 0 {
		return res, nil
	}
	val := db.Value{Field: map[string]string{}}
	val.SetList(dnsSearchField, search)
	res[DNS_OPTIONS_TABLE] = map[string]db.Value{DNS_OPTIONS_GLOBAL_KEY: val}
	return res, nil
}

func dnsRootObject(inParams XfmrParams) *ocbinds.OpenconfigSystem_System_Dns {
	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	if deviceObj.System == nil {
		deviceObj.System = &ocbinds.OpenconfigSystem_System{}
	}
	if deviceObj.System.Dns == nil {
		deviceObj.System.Dns = &ocbinds.OpenconfigSystem_System_Dns{}
	}
	return deviceObj.System.Dns
}

var DbToYang_dns_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	d := inParams.dbs[db.ConfigDB]
	if d == nil {
		return tlerr.InternalError{Format: "CONFIG_DB is not available"}
	}
	dns := dnsRootObject(inParams)
	if dns.Config == nil {
		dns.Config = &ocbinds.OpenconfigSystem_System_Dns_Config{}
	}
	dns.Config.Search = dnsSearch(d)
	return nil
}

// DbToYang_dns_state_xfmr fills /system/dns/state. SONiC keeps no separate
// operational copy of the search list, so state mirrors config.
var DbToYang_dns_state_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	d := inParams.dbs[db.ConfigDB]
	if d == nil {
		return tlerr.InternalError{Format: "CONFIG_DB is not available"}
	}
	dns := dnsRootObject(inParams)
	if dns.State == nil {
		dns.State = &ocbinds.OpenconfigSystem_System_Dns_State{}
	}
	dns.State.Search = dnsSearch(d)
	return nil
}

var Subscribe_dns_options_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.ConfigDB: {DNS_OPTIONS_TABLE: {DNS_OPTIONS_GLOBAL_KEY: {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}
