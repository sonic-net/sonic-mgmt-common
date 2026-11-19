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
