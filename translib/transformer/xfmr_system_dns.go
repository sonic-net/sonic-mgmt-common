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
	"github.com/openconfig/ygot/ygot"
)

const (
	DNS_NAMESERVER_TABLE = "DNS_NAMESERVER"

	dnsServersPath    = "/openconfig-system:system/dns/servers"
	dnsServerPath     = "/openconfig-system:system/dns/servers/server"
	dnsServerKeyField = "address"

	// resolv.conf has no port option, so every name server is queried on 53.
	dnsServerPort = 53
)

func init() {
	XlateFuncBind("YangToDb_dns_servers_xfmr", YangToDb_dns_servers_xfmr)
	XlateFuncBind("DbToYang_dns_servers_xfmr", DbToYang_dns_servers_xfmr)
	XlateFuncBind("Subscribe_dns_servers_xfmr", Subscribe_dns_servers_xfmr)
	XlateFuncBind("DbToYangPath_dns_servers_path_xfmr", DbToYangPath_dns_servers_path_xfmr)
}

// dnsNameServers returns the addresses in DNS_NAMESERVER.
func dnsNameServers(d *db.DB) ([]string, error) {
	keys, err := d.GetKeys(&db.TableSpec{Name: DNS_NAMESERVER_TABLE})
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(keys))
	for _, k := range keys {
		if len(k.Comp) > 0 {
			addrs = append(addrs, k.Comp[0])
		}
	}
	return addrs, nil
}

// dnsServersSubOpDelete queues the removal of name servers alongside the
// request, keeping deletes already queued for other tables.
func dnsServersSubOpDelete(inParams XfmrParams, addrs []string) {
	if inParams.subOpDataMap[DELETE] == nil {
		subOpMap := make(RedisDbMap)
		inParams.subOpDataMap[DELETE] = &subOpMap
	}
	subOpMap := *inParams.subOpDataMap[DELETE]
	if subOpMap[db.ConfigDB] == nil {
		subOpMap[db.ConfigDB] = make(map[string]map[string]db.Value)
	}
	tbl := subOpMap[db.ConfigDB][DNS_NAMESERVER_TABLE]
	if tbl == nil {
		tbl = make(map[string]db.Value)
		subOpMap[db.ConfigDB][DNS_NAMESERVER_TABLE] = tbl
	}
	for _, a := range addrs {
		tbl[a] = db.Value{}
	}
}

// YangToDb_dns_servers_xfmr maps /system/dns/servers/server[address] to the
// key-only CONFIG_DB DNS_NAMESERVER table that hostcfgd renders into
// resolv.conf. The port leaf has no CONFIG_DB field and is not written.
var YangToDb_dns_servers_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	tbl := make(map[string]db.Value)
	res := map[string]map[string]db.Value{DNS_NAMESERVER_TABLE: tbl}

	reqPath, err := getYangPathFromUri(inParams.requestUri)
	if err != nil {
		return nil, err
	}
	addr := keyFromInParamsOrUri(inParams, dnsServerKeyField)
	onList := reqPath == dnsServersPath || (reqPath == dnsServerPath && addr == "")
	log.V(3).Infof("YangToDb_dns_servers_xfmr: oper %v, request %s, address %q", inParams.oper, inParams.requestUri, addr)

	if inParams.oper == DELETE {
		switch {
		case onList:
			addrs, err := dnsNameServers(inParams.d)
			if err != nil {
				return nil, err
			}
			for _, a := range addrs {
				tbl[a] = db.Value{}
			}
		case reqPath == dnsServerPath:
			tbl[addr] = db.Value{}
		default:
			return nil, tlerr.NotSupported("Delete is supported on %s and %s only", dnsServersPath, dnsServerPath)
		}
		return res, nil
	}

	var servers map[string]*ocbinds.OpenconfigSystem_System_Dns_Servers_Server
	if sys := getAppRootObject(inParams); sys != nil && sys.Dns != nil && sys.Dns.Servers != nil {
		servers = sys.Dns.Servers.Server
	}
	if !onList && addr != "" {
		tbl[addr] = db.Value{Field: map[string]string{"NULL": "NULL"}}
	}
	for a := range servers {
		tbl[a] = db.Value{Field: map[string]string{"NULL": "NULL"}}
	}

	// A replace of the whole list removes the name servers it leaves out.
	if inParams.oper == REPLACE && onList {
		addrs, err := dnsNameServers(inParams.d)
		if err != nil {
			return nil, err
		}
		var stale []string
		for _, a := range addrs {
			if _, ok := servers[a]; !ok {
				stale = append(stale, a)
			}
		}
		if len(stale) > 0 {
			dnsServersSubOpDelete(inParams, stale)
		}
	}
	return res, nil
}

var DbToYang_dns_servers_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	d := inParams.dbs[db.ConfigDB]
	if d == nil {
		return tlerr.InternalError{Format: "CONFIG_DB is not available"}
	}

	var addrs []string
	if addr := keyFromInParamsOrUri(inParams, dnsServerKeyField); addr != "" {
		if _, err := d.GetEntry(&db.TableSpec{Name: DNS_NAMESERVER_TABLE}, db.Key{Comp: []string{addr}}); err != nil {
			return tlerr.NotFound("DNS server %s is not configured", addr)
		}
		addrs = []string{addr}
	} else {
		var err error
		if addrs, err = dnsNameServers(d); err != nil {
			log.V(3).Infof("DNS_NAMESERVER not read: %v", err)
			return nil
		}
	}

	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	if deviceObj.System == nil {
		deviceObj.System = &ocbinds.OpenconfigSystem_System{}
	}
	if deviceObj.System.Dns == nil {
		deviceObj.System.Dns = &ocbinds.OpenconfigSystem_System_Dns{}
	}
	servers := &ocbinds.OpenconfigSystem_System_Dns_Servers{
		Server: make(map[string]*ocbinds.OpenconfigSystem_System_Dns_Servers_Server),
	}
	for _, a := range addrs {
		servers.Server[a] = &ocbinds.OpenconfigSystem_System_Dns_Servers_Server{
			Address: ygot.String(a),
			Config:  &ocbinds.OpenconfigSystem_System_Dns_Servers_Server_Config{Address: ygot.String(a)},
			State: &ocbinds.OpenconfigSystem_System_Dns_Servers_Server_State{
				Address: ygot.String(a),
				Port:    ygot.Uint16(dnsServerPort),
			},
		}
	}
	deviceObj.System.Dns.Servers = servers
	return nil
}

var Subscribe_dns_servers_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	addr := NewPathInfo(inParams.uri).Var(dnsServerKeyField)
	if addr == "" {
		addr = "*"
	} else if inParams.requestURI == "" {
		// The get and set resource check calls this without a request URI.
		// It must pass for a server that is not configured yet, so that an
		// update below server[address] can create it; DbToYang reports a
		// missing server itself.
		return XfmrSubscOutParams{isVirtualTbl: true}, nil
	}
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.ConfigDB: {DNS_NAMESERVER_TABLE: {addr: {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

// DbToYangPath_dns_servers_path_xfmr fills server[address] in on-change
// notifications from the DNS_NAMESERVER key.
var DbToYangPath_dns_servers_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	if params.tblName != DNS_NAMESERVER_TABLE || len(params.tblKeyComp) != 1 {
		return tlerr.InternalError{Format: "unexpected DNS name server key", Path: params.tblName}
	}
	params.ygPathKeys[dnsServerPath+"/"+dnsServerKeyField] = params.tblKeyComp[0]
	return nil
}
