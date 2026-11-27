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

//go:build testapp
// +build testapp

package transformer_test

import (
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

const dnsServersUrl = "/openconfig-system:system/dns/servers"

func dnsServerUrl(addr string) string {
	return dnsServersUrl + "/server[address=" + addr + "]"
}

func dnsNameServerRows(addrs ...string) map[string]interface{} {
	rows := map[string]interface{}{}
	for _, a := range addrs {
		rows[a] = map[string]interface{}{"NULL": "NULL"}
	}
	return map[string]interface{}{"DNS_NAMESERVER": rows}
}

func dnsServerJson(addr string) string {
	return `{"address":"` + addr + `","config":{"address":"` + addr + `"},"state":{"address":"` + addr + `","port":53}}`
}

var dnsNameServerAbsent = map[string]interface{}{}

func Test_openconfig_system_dns_servers(t *testing.T) {
	const v4, v6, other, fourth = "192.0.2.1", "2001:db8::1", "198.51.100.1", "203.0.113.1"
	cleanup := dnsNameServerRows(v4, v6, other, fourth)

	t.Run("Get without name servers", processGetRequest(dnsServersUrl, nil, `{}`, false))

	t.Run("Patch servers", processSetRequest(dnsServersUrl,
		`{"openconfig-system:servers":{"server":[{"address":"`+v4+`","config":{"address":"`+v4+`"}},{"address":"`+v6+`","config":{"address":"`+v6+`"}}]}}`,
		"PATCH", false))
	t.Run("Verify IPv4 server", verifyDbResult(rclient, "DNS_NAMESERVER|"+v4, dnsNameServerRows(v4), false))
	t.Run("Verify IPv6 server", verifyDbResult(rclient, "DNS_NAMESERVER|"+v6, dnsNameServerRows(v6), false))

	t.Run("Get servers", processGetRequest(dnsServersUrl, nil,
		`{"openconfig-system:servers":{"server":[`+dnsServerJson(v4)+`,`+dnsServerJson(v6)+`]}}`, false))
	t.Run("Get server state", processGetRequest(dnsServerUrl(v4)+"/state", nil,
		`{"openconfig-system:state":{"address":"`+v4+`","port":53}}`, false))
	t.Run("Get unknown server", processGetRequest(dnsServerUrl(other), nil, "", true))

	t.Run("Patch one server config", processSetRequest(dnsServerUrl(other)+"/config",
		`{"openconfig-system:config":{"address":"`+other+`"}}`, "PATCH", false))
	t.Run("Verify added server", verifyDbResult(rclient, "DNS_NAMESERVER|"+other, dnsNameServerRows(other), false))

	t.Run("Reject a fourth server", processSetRequest(dnsServerUrl(fourth)+"/config/address",
		`{"openconfig-system:address":"`+fourth+`"}`, "PATCH", true))
	t.Run("Verify fourth server is not written", verifyDbResult(rclient, "DNS_NAMESERVER|"+fourth, dnsNameServerAbsent, false))

	t.Run("Reject a host name", processSetRequest(dnsServersUrl,
		`{"openconfig-system:servers":{"server":[{"address":"ns.example.com","config":{"address":"ns.example.com"}}]}}`,
		"PATCH", true))

	t.Run("Replace servers", processSetRequest(dnsServersUrl,
		`{"openconfig-system:servers":{"server":[{"address":"`+v6+`","config":{"address":"`+v6+`"}}]}}`,
		"PUT", false))
	t.Run("Verify replace removes IPv4 server", verifyDbResult(rclient, "DNS_NAMESERVER|"+v4, dnsNameServerAbsent, false))
	t.Run("Verify replace removes added server", verifyDbResult(rclient, "DNS_NAMESERVER|"+other, dnsNameServerAbsent, false))
	t.Run("Verify replace keeps IPv6 server", verifyDbResult(rclient, "DNS_NAMESERVER|"+v6, dnsNameServerRows(v6), false))

	t.Run("Delete one server", processDeleteRequest(dnsServerUrl(v6), false))
	t.Run("Verify server deleted", verifyDbResult(rclient, "DNS_NAMESERVER|"+v6, dnsNameServerAbsent, false))

	t.Run("Patch a server address leaf", processSetRequest(dnsServerUrl(v4)+"/config/address",
		`{"openconfig-system:address":"`+v4+`"}`, "PATCH", false))
	t.Run("Verify server added through its leaf", verifyDbResult(rclient, "DNS_NAMESERVER|"+v4, dnsNameServerRows(v4), false))
	unloadDB(db.ConfigDB, cleanup)

	loadDB(db.ConfigDB, dnsNameServerRows(v4, v6))
	t.Run("Delete all servers", processDeleteRequest(dnsServersUrl, false))
	t.Run("Verify IPv4 server deleted", verifyDbResult(rclient, "DNS_NAMESERVER|"+v4, dnsNameServerAbsent, false))
	t.Run("Verify IPv6 server deleted", verifyDbResult(rclient, "DNS_NAMESERVER|"+v6, dnsNameServerAbsent, false))
	unloadDB(db.ConfigDB, cleanup)
}

const dnsConfigUrl = "/openconfig-system:system/dns/config"

func dnsOptionsRow(fields map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"DNS_OPTIONS": map[string]interface{}{"GLOBAL": fields}}
}

var dnsOptionsCleanup = map[string]interface{}{"DNS_OPTIONS": map[string]interface{}{"GLOBAL": ""}}

func Test_openconfig_system_dns_search(t *testing.T) {
	const ns = "192.0.2.1"
	loadDB(db.ConfigDB, dnsNameServerRows(ns))

	t.Run("Get without search domains", processGetRequest(dnsConfigUrl, nil, `{}`, false))

	t.Run("Patch search domains", processSetRequest(dnsConfigUrl,
		`{"openconfig-system:config":{"search":["example.com","corp.example.com"]}}`, "PATCH", false))
	t.Run("Verify search domains", verifyDbResult(rclient, "DNS_OPTIONS|GLOBAL",
		dnsOptionsRow(map[string]interface{}{"search@": "example.com,corp.example.com"}), false))
	t.Run("Get config", processGetRequest(dnsConfigUrl, nil,
		`{"openconfig-system:config":{"search":["example.com","corp.example.com"]}}`, false))
	t.Run("Get state", processGetRequest("/openconfig-system:system/dns/state", nil,
		`{"openconfig-system:state":{"search":["example.com","corp.example.com"]}}`, false))

	t.Run("Patch adds a domain", processSetRequest(dnsConfigUrl+"/search",
		`{"openconfig-system:search":["lab.example.com"]}`, "PATCH", false))
	t.Run("Verify domain appended", verifyDbResult(rclient, "DNS_OPTIONS|GLOBAL",
		dnsOptionsRow(map[string]interface{}{"search@": "example.com,corp.example.com,lab.example.com"}), false))

	t.Run("Delete one domain", processDeleteRequest(dnsConfigUrl+"/search[search=corp.example.com]", false))
	t.Run("Verify one domain removed", verifyDbResult(rclient, "DNS_OPTIONS|GLOBAL",
		dnsOptionsRow(map[string]interface{}{"search@": "example.com,lab.example.com"}), false))

	t.Run("Reject an invalid domain", processSetRequest(dnsConfigUrl+"/search",
		`{"openconfig-system:search":["bad domain"]}`, "PATCH", true))
	unloadDB(db.ConfigDB, dnsOptionsCleanup)

	loadDB(db.ConfigDB, dnsOptionsRow(map[string]interface{}{"search@": "example.com", "ndots": "2"}))
	t.Run("Replace config", processSetRequest(dnsConfigUrl,
		`{"openconfig-system:config":{"search":["example.net"]}}`, "PUT", false))
	t.Run("Verify replace keeps ndots", verifyDbResult(rclient, "DNS_OPTIONS|GLOBAL",
		dnsOptionsRow(map[string]interface{}{"search@": "example.net", "ndots": "2"}), false))

	t.Run("Delete search domains", processDeleteRequest(dnsConfigUrl+"/search", false))
	t.Run("Verify delete keeps ndots", verifyDbResult(rclient, "DNS_OPTIONS|GLOBAL",
		dnsOptionsRow(map[string]interface{}{"ndots": "2"}), false))
	t.Run("Delete search domains again", processDeleteRequest(dnsConfigUrl+"/search", false))
	unloadDB(db.ConfigDB, dnsOptionsCleanup)
	unloadDB(db.ConfigDB, dnsNameServerRows(ns))
}
