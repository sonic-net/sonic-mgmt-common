////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2024 Dell, Inc.                                                 //
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
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/Azure/sonic-mgmt-common/translib"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

func Test_openconfig_interfaces(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING INTERFACES ATTRIBUTES ++++++++++++")
	t.Log("\n\n--- PATCH interfaces config---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"mtu\": 8900, \"description\": \"UT_Interface\", \"enabled\": false}}"
	t.Run("Test PATCH on interface config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interfaces config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"description\": \"UT_Interface\", \"enabled\": false, \"mtu\": 8900, \"name\": \"Ethernet0\"}}"
	t.Run("Test GET on interface config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interface leaf nodes---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/enabled"
	url_input_body_json = "{\"openconfig-interfaces:enabled\": true}"
	t.Run("Test PATCH on interface enabled", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/mtu"
	url_input_body_json = "{\"openconfig-interfaces:mtu\": 9000}"
	t.Run("Test PATCH on interface mtu", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/description"
	url_input_body_json = "{\"openconfig-interfaces:description\": \"\"}"
	t.Run("Test PATCH on interface description", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	cleanuptbl := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{"admin_status": "up", "mtu": "9000"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify PATCH interface leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": { \"admin-status\": \"UP\", \"counters\": {\"in-broadcast-pkts\": \"0\", \"in-discards\": \"0\", \"in-errors\": \"0\", \"in-multicast-pkts\": \"0\", \"in-octets\": \"0\", \"in-pkts\": \"0\", \"in-unicast-pkts\": \"0\", \"out-broadcast-pkts\": \"0\", \"out-discards\": \"0\", \"out-errors\": \"0\", \"out-multicast-pkts\": \"0\", \"out-octets\": \"0\", \"out-pkts\": \"0\", \"out-unicast-pkts\": \"0\"}, \"enabled\": true, \"mtu\": 9000, \"name\": \"Ethernet0\"}}"
	t.Run("Test GET on interface state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interface enabled  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/enabled"
	t.Run("Test DELETE on interface enabled", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at interface enabled  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/enabled"
	expected_get_json = "{\"openconfig-interfaces:enabled\": false}"
	t.Run("Test GET on interface config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interface mtu  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/mtu"
	t.Run("Test DELETE on interface mtu", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at interface mtu  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/mtu"
	expected_get_json = "{\"openconfig-interfaces:mtu\": 9100}"
	t.Run("Test GET on interface config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interfaces container  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]"
	err_str := "Physical Interface: Ethernet0 cannot be deleted"
	expected_err := tlerr.InvalidArgsError{Format: err_str}
	t.Run("Test DELETE on interface container", processDeleteRequest(url, true, expected_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interface description ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/description"
	t.Run("Test DELETE on interface description", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at interface description ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/config/description"
	err_str = "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted description", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Ethernet0\",\"config\":{\"name\":\"Ethernet0\",\"mtu\":9100,\"enabled\":true}}]}"
	t.Run("Test PATCH on interface", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	cleanuptbl = map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{"admin_status": "up", "mtu": "9100"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify PATCH interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": { \"admin-status\": \"UP\", \"counters\": {\"in-broadcast-pkts\": \"0\", \"in-discards\": \"0\", \"in-errors\": \"0\", \"in-multicast-pkts\": \"0\", \"in-octets\": \"0\", \"in-pkts\": \"0\", \"in-unicast-pkts\": \"0\", \"out-broadcast-pkts\": \"0\", \"out-discards\": \"0\", \"out-errors\": \"0\", \"out-multicast-pkts\": \"0\", \"out-octets\": \"0\", \"out-pkts\": \"0\", \"out-unicast-pkts\": \"0\"}, \"enabled\": true, \"mtu\": 9100, \"name\": \"Ethernet0\"}}"
	t.Run("Test GET on interface state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ApplDB, cleanuptbl)
}

func Test_openconfig_ethernet(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING ETHERNET ATTRIBUTES ++++++++++++")
	t.Log("\n\n--- PATCH ethernet auto-neg and port-speed ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/port-speed"
	url_input_body_json = "{\"openconfig-if-ethernet:port-speed\":\"SPEED_40GB\"}"
	t.Run("Test PATCH on ethernet port-speed", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/auto-negotiate"
	url_input_body_json = "{\"openconfig-if-ethernet:auto-negotiate\":true}"
	t.Run("Test PATCH on ethernet auto-neg", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	cleanuptbl := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{"admin_status": "up", "autoneg": "on", "mtu": "9100", "speed": "40000", "counters": map[string]interface{}{"in-broadcast-pkts": "0", "in-discards": "0", "in-errors": "0", "in-multicast-pkts": "0", "in-octets": "0", "in-pkts": "0", "in-unicast-pkts": "0", "out-broadcast-pkts": "0", "out-discards": "0", "out-errors": "0", "out-multicast-pkts": "0", "out-octets": "0", "out-pkts": "0", "out-unicast-pkts": "0"}}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify PATCH ethernet ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config"
	expected_get_json := "{\"openconfig-if-ethernet:config\": {\"auto-negotiate\": true,\"port-speed\": \"openconfig-if-ethernet:SPEED_40GB\"}}"
	t.Run("Test GET on ethernet", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at ethernet port-speed---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/port-speed"
	err_str := "DELETE request not allowed for port-speed"
	expected_err := tlerr.NotSupportedError{Format: err_str}
	t.Run("Test DELETE on ethernet port-speed", processDeleteRequest(url, true, expected_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at ethernet container  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet"
	t.Run("Test DELETE on ethernet", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at ethernet container  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config"
	expected_get_json = "{\"openconfig-if-ethernet:config\": {\"auto-negotiate\": true,\"port-speed\": \"openconfig-if-ethernet:SPEED_40GB\"}}"
	t.Run("Test GET on ethernet", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at ethernet auto-negotiate ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/auto-negotiate"
	t.Run("Test DELETE on ethernet auto-negotiate", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at ethernet auto-negotiate ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/auto-negotiate"
	err_str = "auto-negotiate not set"
	expected_err_invalid := tlerr.InvalidArgsError{Format: err_str}
	t.Run("Test GET on deleted auto-negotiate", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH port-speed to set auto-neg ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/port-speed"
	url_input_body_json = "{\"openconfig-if-ethernet:port-speed\":\"SPEED_10GB\"}"
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH on ethernet port-speed", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH on port-speed to set auto-neg ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config"
	expected_get_json = "{\"openconfig-if-ethernet:config\": {\"auto-negotiate\": false,\"port-speed\": \"openconfig-if-ethernet:SPEED_10GB\"}}"
	t.Run("Test GET on ethernet config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at ethernet config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config"
	t.Run("Test DELETE on ethernet config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at ethernet config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/auto-negotiate"
	expected_get_json = "{\"openconfig-if-ethernet:auto-negotiate\": false}"
	t.Run("Test GET on auto-negotiate", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at ethernet config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/port-speed"
	expected_get_json = "{\"openconfig-if-ethernet:port-speed\": \"openconfig-if-ethernet:SPEED_10GB\"}"
	t.Run("Test GET on port-speed", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

}

func Test_openconfig_subintf_ipv4(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING AND REMOVING IPv4 ADDRESS AT SUBINTERFACES ++++++++++++")
	t.Log("\n\n--- TC 1: Delete/Clear existing IPv4 address on Ethernet0 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("Test Delete/Clear IPv4 on subinterfaces", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get/Verify IPv4 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json := "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get IPv4 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"4.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\"}]}}, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	t.Run("Test Delete IPv4 address at subinterfaces level", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test/Verify Get at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//-------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Get at subinterfaces/subinterface[index=0] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get at subinterface[index=0]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"4.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address at subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces/subinterface[index=0] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\"}]}}, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterface[index=0]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Negative test: Verify IPv4 address at incorrect subinterfaces/subinterface[index=1] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=1]"
	expected_get_json = "{}"
	t.Run("Negative test: Test Get IPv4 address at incorrect subinterface[index=1]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv4 address at subinterfaces/subinterface[index=0] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]"
	t.Run("Test Delete IPv4 address at subinterface[index=0] level", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces/subinterface[index=0] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterface[index=0]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"4.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\"}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces ipv4/addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv4 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("Test Delete IPv4 address on subinterfaces addresses", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	expected_get_json = "{}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Duplicate IP test: PATCH existing IPv4 address on another interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet8]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"4.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
	err_str := "IP 4.4.4.4/24 overlaps with IP or IP Anycast 4.4.4.4/24 of Interface Ethernet0"
	expected_err := tlerr.InvalidArgsError{Format: err_str}
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set duplicate IPv4 address on another interface", processSetRequest(url, url_input_body_json, "PATCH", true, expected_err))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Negative test: Delete IPv4 container ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv4"
	err_str = "DELETE operation not allowed on this container"
	expected_err_2 := tlerr.NotSupportedError{Format: err_str}
	time.Sleep(1 * time.Second)
	t.Run("Test Delete IPv4 container not allowed", processDeleteRequest(url, true, expected_err_2))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Delete IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=4.4.4.4]"
	t.Run("Test Delete IPv4 address on subinterfaces address", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ DONE CONFIGURING AND REMOVING IPV4 ADDRESSES ON SUBINTERFACES  ++++++++++++")
}

func Test_openconfig_subintf_ipv6(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING AND REMOVING IPv6 ADDRESS AT SUBINTERFACES ++++++++++++")
	t.Log("\n\n--- Delete/Clear IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete/Clear IPv6 address on subinterfaces addresses", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json := "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get IPv6 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"a::e\", \"openconfig-if-ip:config\": {\"ip\": \"a::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\"}]},\"config\":{\"enabled\":false}, \"state\": {\"enabled\": false}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	t.Run("Test Delete IPv6 address at subinterfaces", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv6 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"a::e\", \"openconfig-if-ip:config\": {\"ip\": \"a::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 address at subinterface level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\"}]},\"config\":{\"enabled\":false}, \"state\": {\"enabled\": false}}, \"state\":{\"index\":0}}]}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface"
	t.Run("Test Delete IPv6 address at subinterface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv6 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"a::e\", \"openconfig-if-ip:config\": {\"ip\": \"a::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\"}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces addresses level---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete IPv6 address at subinterfaces addresses level", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/addresses"
	expected_get_json = "{}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Negative test: Delete IPv6 config container ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/config"
	err_str := "Delete not allowed at this container"
	expected_err_2 := tlerr.NotSupportedError{Format: err_str}
	time.Sleep(1 * time.Second)
	t.Run("Test Delete IPv6 config", processDeleteRequest(url, true, expected_err_2))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Delete IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=a::e]"
	t.Run("Test Delete/Clear IPv6 address on subinterfaces address", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Verify Get IPv6 address after Delete at subinterfaces ipv6/config level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=a::e]/config"
	err_str = "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	time.Sleep(1 * time.Second)
	t.Run("Test Get/Verify Patch IPv6 address after Delete at subinterfaces ipv6/config", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get IPv6 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ DONE CONFIGURING AND REMOVING IPV6 ADDRESSES ON SUBINTERFACES  ++++++++++++")

	t.Log("\n\n+++++++++++++ ENABLE AND DISABLE IPV6 LINK LOCAL ON SUBINTERFACES  ++++++++++++")
	t.Log("\n\n--- Get IPv6 link local value (enabled) at config level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/config"
	expected_get_json = "{\"openconfig-if-ip:config\": {\"enabled\": false}}"
	t.Run("Test Get IPv6 link local at subinterfaces config level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH/Enable ipv6 link local at config/enabled level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/config/enabled"
	url_input_body_json = "{\"openconfig-if-ip:enabled\": true}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set/Enable IPv6 link local on subinterfaces config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 link local at config/enabled level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/config/enabled"
	expected_get_json = "{\"openconfig-if-ip:enabled\": true}"
	time.Sleep(1 * time.Second)
	t.Run("Test Get/Verify Patch IPv6 link local at subinterfaces config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete/Disable IPv6 link local ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/config/enabled"
	t.Run("Test Delete/Disable IPv6 link local on subinterfaces config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 link local ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/ipv6/config"
	expected_get_json = "{\"openconfig-if-ip:config\": {\"enabled\": false}}"
	t.Run("Test Get/Verify Delete IPv6 link local at subinterfaces config level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ DONE ENABLING AND DISABLING IPV6 LINK LOCAL ON SUBINTERFACES  ++++++++++++")
}

func loadVrrpIntfIPs(tbl, ifName string, prefixes ...string) func() {
	load := map[string]interface{}{}
	clean := map[string]interface{}{}
	for _, p := range prefixes {
		load[ifName+"|"+p] = map[string]interface{}{}
		clean[ifName+"|"+p] = ""
	}
	loadDB(db.ConfigDB, map[string]interface{}{tbl: load})
	return func() {
		unloadDB(db.ConfigDB, map[string]interface{}{tbl: clean})
	}
}

func Test_openconfig_routed_vlan_ipv4_vrrp_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY IPv4 VRRP OPENCONFIG LEAVES ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":          "25",
				"vip@":         "50.0.0.254/24",
				"preempt":      "true",
				"priority":     "100",
				"adv_interval": "1",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":25,\"virtual-address\":[\"50.0.0.254\"],\"priority\":100,\"preempt\":true,\"advertisement-interval\":100}}"
	t.Run("Test GET vrrp-group config leaves", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/state"
	expectedStateJSON := "{\"openconfig-if-ip:state\":{\"virtual-router-id\":25,\"virtual-address\":[\"50.0.0.254\"],\"priority\":100,\"preempt\":true,\"advertisement-interval\":100,\"current-priority\":100}}"
	t.Run("Test GET vrrp-group state leaves", processGetRequest(url, nil, expectedStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_fsm_state_get(t *testing.T) {
	config := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid": "25", "vip@": "50.0.0.254",
				"priority": "100", "preempt": "true",
				"adv_interval": "1",
			},
		},
	}
	state := map[string]interface{}{
		"VRRP_STATE_TABLE": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"effective_priority": "90",
				"state":              "Master",
				"last_changed":       "1791180000",
				"advertisements_tx":  "171",
				"advertisements_rx":  "0",
				"gratuitous_arp_tx":  "1",
				"state_transitions":  "2",
			},
		},
	}
	loadDB(db.ConfigDB, config)
	loadDB(db.StateDB, state)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/" +
		"openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/" +
		"address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/state"
	expected := "{\"openconfig-if-ip:state\":{" +
		"\"virtual-router-id\":25," +
		"\"virtual-address\":[\"50.0.0.254\"]," +
		"\"priority\":100,\"preempt\":true," +
		"\"advertisement-interval\":100,\"current-priority\":90," +
		"\"openconfig-if-ip-vrrp-ext:counters\":{" +
		"\"advertisements-received\":\"0\",\"advertisements-sent\":\"171\"," +
		"\"gratuitous-arp-sent\":\"1\",\"state-transitions\":\"2\"}," +
		"\"openconfig-if-ip-vrrp-ext:vrrp-state\":\"MASTER\"}}"
	t.Run("GET VRRP FSM state from existing STATE_DB field",
		processGetRequest(url, nil, expected, false))

	leafURL := url + "/openconfig-if-ip-vrrp-ext:vrrp-state"
	leafExpected := "{\"openconfig-if-ip-vrrp-ext:vrrp-state\":\"MASTER\"}"
	t.Run("GET VRRP FSM extension leaf",
		processGetRequest(leafURL, nil, leafExpected, false))

	countersURL := url + "/openconfig-if-ip-vrrp-ext:counters"
	countersExpected := "{\"openconfig-if-ip-vrrp-ext:counters\":{" +
		"\"advertisements-received\":\"0\",\"advertisements-sent\":\"171\"," +
		"\"gratuitous-arp-sent\":\"1\",\"state-transitions\":\"2\"}}"
	t.Run("GET IPv4 VRRP counters",
		processGetRequest(countersURL, nil, countersExpected, false))

	unloadDB(db.StateDB, state)
	unloadDB(db.ConfigDB, config)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_counters_get(t *testing.T) {
	config := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid": "25", "vip@": "2001:db8::ff",
				"priority": "100", "preempt": "true",
				"adv_interval": "1",
			},
		},
	}
	state := map[string]interface{}{
		"VRRP6_STATE_TABLE": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"effective_priority":         "90",
				"state":                      "Backup",
				"last_changed":               "1791180000",
				"advertisements_tx":          "1806",
				"advertisements_rx":          "1",
				"neighbor_advertisements_tx": "1",
				"state_transitions":          "2",
			},
		},
	}
	loadDB(db.ConfigDB, config)
	loadDB(db.StateDB, state)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/" +
		"openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/" +
		"address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=25]/state/" +
		"openconfig-if-ip-vrrp-ext:counters"
	expected := "{\"openconfig-if-ip-vrrp-ext:counters\":{" +
		"\"advertisements-received\":\"1\",\"advertisements-sent\":\"1806\"," +
		"\"neighbor-advertisements-sent\":\"1\",\"state-transitions\":\"2\"}}"
	t.Run("GET IPv6 VRRP counters",
		processGetRequest(url, nil, expected, false))

	unloadDB(db.StateDB, state)
	unloadDB(db.ConfigDB, config)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_tracking_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY IPv4 VRRP TRACKING OPENCONFIG LEAVES ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":          "25",
				"vip@":         "50.0.0.254/24",
				"preempt":      "true",
				"priority":     "100",
				"adv_interval": "1",
			},
		},
		"VRRP_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet16": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/interface-tracking/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"priority-decrement\":20,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Test GET vrrp-group interface-tracking config", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/interface-tracking/state"
	expectedStateJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":20,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Test GET vrrp-group interface-tracking state", processGetRequest(url, nil, expectedStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
		"VRRP_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet16": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY IPv6 VRRP OPENCONFIG LEAVES ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":          "25",
				"vip@":         "2001:db8::ff/64",
				"preempt":      "true",
				"priority":     "120",
				"adv_interval": "2",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet20": map[string]interface{}{
				"priority_increment": "15",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=25]/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":25,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":120,\"preempt\":true,\"advertisement-interval\":200}}"
	t.Run("Test GET ipv6 vrrp-group config leaves", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=25]/interface-tracking/state"
	expectedTrackingStateJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":15,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Test GET ipv6 vrrp-group interface-tracking state", processGetRequest(url, nil, expectedTrackingStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": "",
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|25": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_patch(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH ROUTED VLAN IPv4 VRRP OPENCONFIG LEAVES ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":           "25",
				"vip@":          "50.0.0.254/24",
				"preempt":       "true",
				"priority":      "100",
				"adv_interval":  "1",
				"preempt_delay": "0",
			},
		},
		"VRRP_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet16": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]"
	t.Run("Patch preempt", processSetRequest(baseURL+"/config/preempt", "{\"preempt\":false}", "PATCH", false, nil))
	t.Run("Patch priority", processSetRequest(baseURL+"/config/priority", "{\"priority\":105}", "PATCH", false, nil))
	t.Run("Patch advertisement-interval", processSetRequest(baseURL+"/config/advertisement-interval", "{\"advertisement-interval\":200}", "PATCH", false, nil))
	t.Run("Patch preempt-delay", processSetRequest(baseURL+"/config/preempt-delay", "{\"preempt-delay\":5}", "PATCH", false, nil))
	t.Run("Patch tracking priority-decrement", processSetRequest(baseURL+"/interface-tracking/config/priority-decrement", "{\"priority-decrement\":30}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":25,\"virtual-address\":[\"50.0.0.254\"],\"priority\":105,\"preempt\":false,\"preempt-delay\":5,\"advertisement-interval\":200}}"
	t.Run("Verify GET patched routed-vlan ipv4 vrrp-group config", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	expectedTrackingJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":30,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Verify GET patched routed-vlan ipv4 tracking state", processGetRequest(baseURL+"/interface-tracking/state", nil, expectedTrackingJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
		"VRRP_TRACK": map[string]interface{}{
			"Vlan100|25|Ethernet16": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_patch(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH ROUTED VLAN IPv6 VRRP OPENCONFIG LEAVES ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": map[string]interface{}{
				"vid":           "29",
				"vip@":          "2001:db8::ff/64",
				"preempt":       "true",
				"priority":      "120",
				"adv_interval":  "2",
				"preempt_delay": "0",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Vlan100|29|Ethernet20": map[string]interface{}{
				"priority_increment": "15",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"
	t.Run("Patch preempt", processSetRequest(baseURL+"/config/preempt", "{\"preempt\":false}", "PATCH", false, nil))
	t.Run("Patch priority", processSetRequest(baseURL+"/config/priority", "{\"priority\":115}", "PATCH", false, nil))
	t.Run("Patch advertisement-interval", processSetRequest(baseURL+"/config/advertisement-interval", "{\"advertisement-interval\":300}", "PATCH", false, nil))
	t.Run("Patch preempt-delay", processSetRequest(baseURL+"/config/preempt-delay", "{\"preempt-delay\":7}", "PATCH", false, nil))
	t.Run("Patch tracking priority-decrement", processSetRequest(baseURL+"/interface-tracking/config/priority-decrement", "{\"priority-decrement\":25}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":29,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":115,\"preempt\":false,\"preempt-delay\":7,\"advertisement-interval\":300}}"
	t.Run("Verify GET patched routed-vlan ipv6 vrrp-group config", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	expectedTrackingJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":25,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Verify GET patched routed-vlan ipv6 tracking state", processGetRequest(baseURL+"/interface-tracking/state", nil, expectedTrackingJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": "",
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Vlan100|29|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY SUBINTERFACE IPv4 VRRP OPENCONFIG LEAVES ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "20.0.0.1/24")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":          "19",
				"vip@":         "10.0.0.254/24",
				"preempt":      "true",
				"priority":     "110",
				"adv_interval": "1",
			},
		},
		"VRRP_TRACK": map[string]interface{}{
			"Ethernet0|19|Ethernet16": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":19,\"virtual-address\":[\"10.0.0.254\"],\"priority\":110,\"preempt\":true,\"advertisement-interval\":100}}"
	t.Run("Test GET subinterface ipv4 vrrp-group config", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]/interface-tracking/state"
	expectedTrackingStateJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":20,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Test GET subinterface ipv4 vrrp-group interface-tracking state", processGetRequest(url, nil, expectedTrackingStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
		"VRRP_TRACK": map[string]interface{}{
			"Ethernet0|19|Ethernet16": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv6_vrrp_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY SUBINTERFACE IPv6 VRRP OPENCONFIG LEAVES ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": map[string]interface{}{
				"vid":          "29",
				"vip@":         "2001:db8::ff/64",
				"preempt":      "true",
				"priority":     "120",
				"adv_interval": "2",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": map[string]interface{}{
				"priority_increment": "15",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":29,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":120,\"preempt\":true,\"advertisement-interval\":200}}"
	t.Run("Test GET subinterface ipv6 vrrp-group config", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]/interface-tracking/state"
	expectedTrackingStateJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":15,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Test GET subinterface ipv6 vrrp-group interface-tracking state", processGetRequest(url, nil, expectedTrackingStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_portchannel_subinterface_vrrp_get(t *testing.T) {
	t.Log("\n\n+++++++++++++ QUERY PORTCHANNEL SUBINTERFACE VRRP OPENCONFIG LEAVES ++++++++++++")
	defer loadVrrpIntfIPs("PORTCHANNEL_INTERFACE", "PortChannel001", "20.0.0.1/24", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"PortChannel001|19": map[string]interface{}{
				"vid":          "19",
				"vip@":         "10.0.0.254/24",
				"preempt":      "true",
				"priority":     "110",
				"adv_interval": "1",
			},
		},
		"VRRP_TRACK": map[string]interface{}{
			"PortChannel001|19|Ethernet16": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=PortChannel001]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]/config"
	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":19,\"virtual-address\":[\"10.0.0.254\"],\"priority\":110,\"preempt\":true,\"advertisement-interval\":100}}"
	t.Run("Test GET PortChannel subinterface ipv4 vrrp-group config", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel001]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]/interface-tracking/state"
	expectedTrackingStateJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":20,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Test GET PortChannel subinterface ipv4 vrrp-group interface-tracking state", processGetRequest(url, nil, expectedTrackingStateJSON, false))
	time.Sleep(1 * time.Second)

	preReqMap = map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"PortChannel001|29": map[string]interface{}{
				"vid":          "29",
				"vip@":         "2001:db8::ff/64",
				"preempt":      "true",
				"priority":     "120",
				"adv_interval": "2",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"PortChannel001|29|Ethernet20": map[string]interface{}{
				"priority_increment": "15",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel001]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]/config"
	expectedConfigJSON = "{\"openconfig-if-ip:config\":{\"virtual-router-id\":29,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":120,\"preempt\":true,\"advertisement-interval\":200}}"
	t.Run("Test GET PortChannel subinterface ipv6 vrrp-group config", processGetRequest(url, nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel001]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]/interface-tracking/state"
	expectedTrackingStateJSON = "{\"openconfig-if-ip:state\":{\"priority-decrement\":15,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Test GET PortChannel subinterface ipv6 vrrp-group interface-tracking state", processGetRequest(url, nil, expectedTrackingStateJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"PortChannel001|19": "",
		},
		"VRRP_TRACK": map[string]interface{}{
			"PortChannel001|19|Ethernet16": "",
		},
		"VRRP6": map[string]interface{}{
			"PortChannel001|29": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"PortChannel001|29|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_patch(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH SUBINTERFACE IPv4 VRRP OPENCONFIG LEAVES ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "20.0.0.1/24")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":           "19",
				"vip@":          "10.0.0.254/24",
				"preempt":       "true",
				"priority":      "110",
				"adv_interval":  "1",
				"preempt_delay": "0",
			},
		},
		"VRRP_TRACK": map[string]interface{}{
			"Ethernet0|19|Ethernet16": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]"
	t.Run("Patch preempt", processSetRequest(baseURL+"/config/preempt", "{\"preempt\":false}", "PATCH", false, nil))
	t.Run("Patch priority", processSetRequest(baseURL+"/config/priority", "{\"priority\":105}", "PATCH", false, nil))
	t.Run("Patch advertisement-interval", processSetRequest(baseURL+"/config/advertisement-interval", "{\"advertisement-interval\":200}", "PATCH", false, nil))
	t.Run("Patch preempt-delay", processSetRequest(baseURL+"/config/preempt-delay", "{\"preempt-delay\":5}", "PATCH", false, nil))
	t.Run("Patch tracking priority-decrement", processSetRequest(baseURL+"/interface-tracking/config/priority-decrement", "{\"priority-decrement\":30}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":19,\"virtual-address\":[\"10.0.0.254\"],\"priority\":105,\"preempt\":false,\"preempt-delay\":5,\"advertisement-interval\":200}}"
	t.Run("Verify GET patched ipv4 vrrp-group config", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	expectedTrackingJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":30,\"track-interface\":[\"Ethernet16\"]}}"
	t.Run("Verify GET patched ipv4 tracking state", processGetRequest(baseURL+"/interface-tracking/state", nil, expectedTrackingJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
		"VRRP_TRACK": map[string]interface{}{
			"Ethernet0|19|Ethernet16": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv6_vrrp_patch(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH SUBINTERFACE IPv6 VRRP OPENCONFIG LEAVES ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": map[string]interface{}{
				"vid":           "29",
				"vip@":          "2001:db8::ff/64",
				"preempt":       "true",
				"priority":      "120",
				"adv_interval":  "2",
				"preempt_delay": "0",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": map[string]interface{}{
				"priority_increment": "15",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"
	t.Run("Patch preempt", processSetRequest(baseURL+"/config/preempt", "{\"preempt\":false}", "PATCH", false, nil))
	t.Run("Patch priority", processSetRequest(baseURL+"/config/priority", "{\"priority\":115}", "PATCH", false, nil))
	t.Run("Patch advertisement-interval", processSetRequest(baseURL+"/config/advertisement-interval", "{\"advertisement-interval\":300}", "PATCH", false, nil))
	t.Run("Patch tracking priority-decrement", processSetRequest(baseURL+"/interface-tracking/config/priority-decrement", "{\"priority-decrement\":25}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":29,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":115,\"preempt\":false,\"advertisement-interval\":300}}"
	t.Run("Verify GET patched ipv6 vrrp-group config", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))
	time.Sleep(1 * time.Second)

	expectedTrackingJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":25,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Verify GET patched ipv6 tracking state", processGetRequest(baseURL+"/interface-tracking/state", nil, expectedTrackingJSON, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_delete_leaf_defaults(t *testing.T) {
	t.Log("\n\n+++++++++++++ DELETE SUBINTERFACE IPv4 VRRP LEAF DEFAULTS ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "20.0.0.1/24")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":           "19",
				"vip@":          "10.0.0.254/24",
				"preempt":       "false",
				"priority":      "110",
				"adv_interval":  "2",
				"preempt_delay": "9",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]"
	t.Run("Delete priority leaf", processDeleteRequest(baseURL+"/config/priority", true))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":19,\"virtual-address\":[\"10.0.0.254\"],\"priority\":100,\"preempt\":false,\"preempt-delay\":9,\"advertisement-interval\":200}}"
	t.Run("Verify GET config after deleting priority leaf", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))

	cleanupTbl := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv6_vrrp_delete_leaf_defaults(t *testing.T) {
	t.Log("\n\n+++++++++++++ DELETE SUBINTERFACE IPv6 VRRP LEAF DEFAULTS ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": map[string]interface{}{
				"vid":           "29",
				"vip@":          "2001:db8::ff/64",
				"preempt":       "false",
				"priority":      "120",
				"adv_interval":  "3",
				"preempt_delay": "7",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"
	t.Run("Delete priority leaf", processDeleteRequest(baseURL+"/config/priority", true))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":29,\"virtual-address\":[\"2001:db8::ff\"],\"priority\":100,\"preempt\":false,\"preempt-delay\":7,\"advertisement-interval\":300}}"
	t.Run("Verify GET config after deleting priority leaf", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))

	cleanupTbl := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_patch_config_container(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH SUBINTERFACE IPv4 VRRP CONFIG CONTAINER ++++++++++++")

	preReqMap := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":           "19",
				"vip@":          "10.0.0.254/24",
				"preempt":       "true",
				"priority":      "110",
				"adv_interval":  "1",
				"preempt_delay": "0",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]"
	payload := "{\"config\":{\"priority\":107,\"preempt\":false,\"preempt-delay\":4,\"advertisement-interval\":300,\"virtual-address\":[\"20.0.0.250\"]}}"
	t.Run("Patch config container", processSetRequest(baseURL+"/config", payload, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":19,\"virtual-address\":[\"20.0.0.250\"],\"priority\":107,\"preempt\":false,\"preempt-delay\":4,\"advertisement-interval\":300}}"
	t.Run("Verify GET config after config container patch", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))

	cleanupTbl := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv6_vrrp_patch_tracking_config_container(t *testing.T) {
	t.Log("\n\n+++++++++++++ PATCH SUBINTERFACE IPv6 VRRP TRACKING CONFIG CONTAINER ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": map[string]interface{}{
				"vid":           "29",
				"vip@":          "2001:db8::ff/64",
				"preempt":       "true",
				"priority":      "120",
				"adv_interval":  "2",
				"preempt_delay": "0",
			},
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": map[string]interface{}{
				"priority_increment": "10",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"
	payload := "{\"config\":{\"priority-decrement\":25,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Patch tracking config container", processSetRequest(baseURL+"/interface-tracking/config", payload, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedTrackingJSON := "{\"openconfig-if-ip:state\":{\"priority-decrement\":25,\"track-interface\":[\"Ethernet20\"]}}"
	t.Run("Verify GET tracking state after config container patch", processGetRequest(baseURL+"/interface-tracking/state", nil, expectedTrackingJSON, false))

	cleanupTbl := map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": "",
		},
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet0|29|Ethernet20": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

// accept-mode and virtual-link-local are not-supported
func Test_openconfig_vrrp_unsupported_leaves(t *testing.T) {
	t.Log("\n\n+++++++++++++ VRRP UNSUPPORTED LEAVES accept-mode / virtual-link-local ++++++++++++")
	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "20.0.0.1/24", "2001:db8::1/64")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":          "19",
				"vip@":         "10.0.0.254/24",
				"preempt":      "true",
				"priority":     "110",
				"adv_interval": "1",
			},
		},
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": map[string]interface{}{
				"vid":          "29",
				"vip@":         "2001:db8::ff/64",
				"preempt":      "true",
				"priority":     "120",
				"adv_interval": "2",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	v4Group := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]"
	v6Group := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"

	t.Run("GET supported priority leaf ipv4", processGetRequest(v4Group+"/config/priority", nil, "{\"openconfig-if-ip:priority\":110}", false))
	t.Run("GET supported priority leaf ipv6", processGetRequest(v6Group+"/config/priority", nil, "{\"openconfig-if-ip:priority\":120}", false))

	type unsupportedLeaf struct {
		group   string
		leaf    string
		payload string
	}
	leaves := []unsupportedLeaf{
		{v4Group, "accept-mode", "{\"accept-mode\":true}"},
		{v6Group, "accept-mode", "{\"accept-mode\":true}"},
		{v6Group, "virtual-link-local", "{\"virtual-link-local\":\"fe80::beef\"}"},
	}
	user := UserRoles{Name: "admin", Roles: []string{"admin"}}
	for _, l := range leaves {
		for _, container := range []string{"config", "state"} {
			url := l.group + "/" + container + "/" + l.leaf
			t.Run("GET "+container+"/"+l.leaf, func(t *testing.T) {
				if _, err := Get(GetRequest{Path: url, User: user}); err == nil {
					t.Fatalf("expected GET to fail for unsupported leaf %s", url)
				}
			})
		}
		url := l.group + "/config/" + l.leaf
		t.Run("PATCH "+l.leaf, func(t *testing.T) {
			if _, err := Update(SetRequest{Path: url, Payload: []byte(l.payload)}); err == nil {
				t.Fatalf("expected PATCH to fail for unsupported leaf %s", url)
			}
		})
		t.Run("DELETE "+l.leaf, func(t *testing.T) {
			if _, err := Delete(SetRequest{Path: url}); err == nil {
				t.Fatalf("expected DELETE to fail for unsupported leaf %s", url)
			}
		})
	}

	for _, group := range []string{v4Group, v6Group} {
		for _, container := range []string{"config", "state"} {
			url := group + "/" + container
			t.Run("GET "+container+" omits unsupported leaves", func(t *testing.T) {
				resp, err := Get(GetRequest{Path: url, User: user})
				if err != nil {
					t.Fatalf("unexpected error for %s: %v", url, err)
				}
				payload := string(resp.Payload)
				for _, leaf := range []string{"accept-mode", "virtual-link-local"} {
					if strings.Contains(payload, leaf) {
						t.Fatalf("%s still returns unsupported leaf %s: %s", url, leaf, payload)
					}
				}
			})
		}
	}

	cleanupTbl := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
		"VRRP6": map[string]interface{}{
			"Ethernet0|29": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_vip_parent_subnet(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv4 VRRP VIP PARENT SUBNET VALIDATION ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":          "25",
				"vip@":         "50.0.0.254",
				"preempt":      "true",
				"priority":     "100",
				"adv_interval": "1",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]"
	t.Run("Patch VIP in parent subnet", processSetRequest(baseURL+"/config/virtual-address", "{\"virtual-address\":[\"50.0.0.200\"]}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	expectedConfigJSON := "{\"openconfig-if-ip:config\":{\"virtual-router-id\":25,\"virtual-address\":[\"50.0.0.200\"],\"priority\":100,\"preempt\":true,\"advertisement-interval\":100}}"
	t.Run("Verify GET after VIP in parent subnet", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))

	expErr := tlerr.InvalidArgsError{Format: "VIP 8.8.8.8 is not in the same subnet of parent interface Vlan100"}
	t.Run("Patch VIP outside parent subnet rejected", processSetRequest(baseURL+"/config/virtual-address", "{\"virtual-address\":[\"8.8.8.8\"]}", "PATCH", true, expErr))
	time.Sleep(1 * time.Second)

	t.Run("Verify GET unchanged after rejected VIP", processGetRequest(baseURL+"/config", nil, expectedConfigJSON, false))

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_vip_parent_subnet_rejected(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv6 VRRP VIP PARENT SUBNET VALIDATION ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": map[string]interface{}{
				"vid":          "29",
				"vip@":         "2001:db8::ff",
				"preempt":      "true",
				"priority":     "120",
				"adv_interval": "2",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]"
	expErr := tlerr.InvalidArgsError{Format: "VIP 2001:db9::1 is not in the same subnet of parent interface Vlan100"}
	t.Run("Patch IPv6 VIP outside parent subnet rejected", processSetRequest(baseURL+"/config/virtual-address", "{\"virtual-address\":[\"2001:db9::1\"]}", "PATCH", true, expErr))

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": "",
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_owned_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv4 REMOVE PREFIX OWNING A VRRP VIP ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
			"Vlan100|60.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":  "25",
				"vip@": "50.0.0.254",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"

	// Prefix owning the VIP cannot be deleted.
	expErr := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 50.0.0.1/24 on Vlan100: VRRP VIP 50.0.0.254 (VR Id: 25) is associated with this prefix."}
	t.Run("Delete prefix owning VRRP VIP rejected", processDeleteRequest(baseURL+"/address[ip=50.0.0.1]", true, expErr))
	// Bulk delete of all addresses includes the owning prefix.
	t.Run("Delete all IPv4 addresses rejected", processDeleteRequest(baseURL, true, expErr))
	time.Sleep(1 * time.Second)

	// The owning prefix is untouched after the rejected requests.
	t.Run("Verify owning prefix still configured", processGetRequest(baseURL+"/address[ip=50.0.0.1]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"50.0.0.1\",\"prefix-length\":24}}", false))

	// A prefix that does not own the VIP can be deleted.
	t.Run("Delete prefix not owning VRRP VIP allowed", processDeleteRequest(baseURL+"/address[ip=60.0.0.1]", false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100":             "",
			"Vlan100|50.0.0.1/24": "",
			"Vlan100|60.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_owned_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv6 REMOVE PREFIX OWNING A VRRP VIP ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
			"Vlan100|50.0.0.1/24":    map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": map[string]interface{}{
				"vid":  "29",
				"vip@": "2001:db8::ff",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	v6URL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]"
	v4URL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]"

	expErr := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 2001:db8::1/64 on Vlan100: VRRP6 VIP 2001:db8::ff (VR Id: 29) is associated with this prefix."}
	t.Run("Delete IPv6 prefix owning VRRP6 VIP rejected", processDeleteRequest(v6URL, true, expErr))

	// IPv4 prefix is unaffected by IPv6-only VRRP.
	t.Run("Delete IPv4 prefix with only VRRP6 VIPs allowed", processDeleteRequest(v4URL, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100":                "",
			"Vlan100|2001:db8::1/64": "",
			"Vlan100|50.0.0.1/24":    "",
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv4_vrrp_same_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv4 REMOVE ANY ADDRESS ON A VRRP-OWNED PREFIX ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
			"Vlan100|50.0.0.2/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":  "25",
				"vip@": "50.0.0.254",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"

	// Either address on the VIP's prefix is protected, even though the other one would remain.
	expErr1 := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 50.0.0.1/24 on Vlan100: VRRP VIP 50.0.0.254 (VR Id: 25) is associated with this prefix."}
	t.Run("Delete first address on VRRP-owned prefix rejected", processDeleteRequest(baseURL+"/address[ip=50.0.0.1]", true, expErr1))
	expErr2 := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 50.0.0.2/24 on Vlan100: VRRP VIP 50.0.0.254 (VR Id: 25) is associated with this prefix."}
	t.Run("Delete second address on VRRP-owned prefix rejected", processDeleteRequest(baseURL+"/address[ip=50.0.0.2]", true, expErr2))
	expErrAll := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 50.0.0.1/24, 50.0.0.2/24 on Vlan100: VRRP VIP 50.0.0.254 (VR Id: 25) is associated with this prefix."}
	t.Run("Delete all IPv4 addresses rejected", processDeleteRequest(baseURL, true, expErrAll))

	// Replacing the list so that one of the addresses is dropped is rejected too.
	listPayload := "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"50.0.0.2\",\"config\":{\"ip\":\"50.0.0.2\",\"prefix-length\":24}}]}}"
	t.Run("Replace address list dropping an address on the VRRP-owned prefix rejected", processSetRequest(baseURL, listPayload, "PUT", true))
	time.Sleep(1 * time.Second)

	t.Run("Verify first address still configured", processGetRequest(baseURL+"/address[ip=50.0.0.1]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"50.0.0.1\",\"prefix-length\":24}}", false))
	t.Run("Verify second address still configured", processGetRequest(baseURL+"/address[ip=50.0.0.2]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"50.0.0.2\",\"prefix-length\":24}}", false))

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100":             "",
			"Vlan100|50.0.0.1/24": "",
			"Vlan100|50.0.0.2/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_ipv6_vrrp_same_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN IPv6 REMOVE ANY ADDRESS ON A VRRP-OWNED PREFIX ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|2001:db8::1/64": map[string]interface{}{},
			"Vlan100|2001:db8::2/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": map[string]interface{}{
				"vid":  "29",
				"vip@": "2001:db8::ff",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"

	expErr1 := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 2001:db8::1/64 on Vlan100: VRRP6 VIP 2001:db8::ff (VR Id: 29) is associated with this prefix."}
	t.Run("Delete first IPv6 address on VRRP6-owned prefix rejected", processDeleteRequest(baseURL+"/address[ip=2001:db8::1]", true, expErr1))
	expErr2 := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 2001:db8::2/64 on Vlan100: VRRP6 VIP 2001:db8::ff (VR Id: 29) is associated with this prefix."}
	t.Run("Delete second IPv6 address on VRRP6-owned prefix rejected", processDeleteRequest(baseURL+"/address[ip=2001:db8::2]", true, expErr2))
	expErrAll := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 2001:db8::1/64, 2001:db8::2/64 on Vlan100: VRRP6 VIP 2001:db8::ff (VR Id: 29) is associated with this prefix."}
	t.Run("Delete all IPv6 addresses rejected", processDeleteRequest(baseURL, true, expErrAll))

	listPayload := "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2001:db8::2\",\"config\":{\"ip\":\"2001:db8::2\",\"prefix-length\":64}}]}}"
	t.Run("Replace IPv6 address list dropping an address on the VRRP6-owned prefix rejected", processSetRequest(baseURL, listPayload, "PUT", true))
	time.Sleep(1 * time.Second)

	t.Run("Verify first IPv6 address still configured", processGetRequest(baseURL+"/address[ip=2001:db8::1]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"2001:db8::1\",\"prefix-length\":64}}", false))
	t.Run("Verify second IPv6 address still configured", processGetRequest(baseURL+"/address[ip=2001:db8::2]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"2001:db8::2\",\"prefix-length\":64}}", false))

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100":                "",
			"Vlan100|2001:db8::1/64": "",
			"Vlan100|2001:db8::2/64": "",
		},
		"VRRP6": map[string]interface{}{
			"Vlan100|29": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_routed_vlan_vrrp_without_vip_prefix_remove_allowed(t *testing.T) {
	t.Log("\n\n+++++++++++++ ROUTED VLAN REMOVE PREFIX WITH VIP-LESS VRRP ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid": "25",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]"
	t.Run("Delete prefix with VIP-less VRRP allowed", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100":             "",
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_owned_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ SUBINTERFACE IPv4 REMOVE PREFIX OWNING A VRRP VIP ++++++++++++")

	// 1.0.0.10 is owned by the most specific prefix 1.0.0.1/24, not by the covering 1.0.0.2/16.
	preReqMap := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|1.0.0.1/24": map[string]interface{}{},
			"Ethernet0|1.0.0.2/16": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|2": map[string]interface{}{
				"vid":  "2",
				"vip@": "1.0.0.10",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"

	// Covering prefix does not own the VIP.
	t.Run("Delete covering prefix not owning VRRP VIP allowed", processDeleteRequest(baseURL+"/address[ip=1.0.0.2]", false))
	time.Sleep(1 * time.Second)

	// The remaining prefix owns the VIP.
	expErr := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 1.0.0.1/24 on Ethernet0: VRRP VIP 1.0.0.10 (VR Id: 2) is associated with this prefix."}
	t.Run("Delete prefix owning VRRP VIP rejected", processDeleteRequest(baseURL+"/address[ip=1.0.0.1]", true, expErr))
	t.Run("Delete all IPv4 addresses rejected", processDeleteRequest(baseURL, true, expErr))
	time.Sleep(1 * time.Second)

	t.Run("Verify owning prefix still configured", processGetRequest(baseURL+"/address[ip=1.0.0.1]/config", nil,
		"{\"openconfig-if-ip:config\":{\"ip\":\"1.0.0.1\",\"prefix-length\":24}}", false))

	cleanupTbl := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0":            "",
			"Ethernet0|1.0.0.1/24": "",
			"Ethernet0|1.0.0.2/16": "",
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|2": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv6_vrrp_owned_prefix_remove(t *testing.T) {
	t.Log("\n\n+++++++++++++ SUBINTERFACE IPv6 REMOVE PREFIX OWNING A VRRP VIP ++++++++++++")

	preReqMap := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|2001:db8::1/64": map[string]interface{}{},
		},
		"VRRP6": map[string]interface{}{
			"Ethernet0|2": map[string]interface{}{
				"vid":  "2",
				"vip@": "2001:db8::10",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]"
	expErr := tlerr.InvalidArgsError{Format: "Cannot alter interface IP 2001:db8::1/64 on Ethernet0: VRRP6 VIP 2001:db8::10 (VR Id: 2) is associated with this prefix."}
	t.Run("Delete IPv6 prefix owning VRRP6 VIP rejected", processDeleteRequest(url, true, expErr))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0":                "",
			"Ethernet0|2001:db8::1/64": "",
		},
		"VRRP6": map[string]interface{}{
			"Ethernet0|2": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_without_vip_prefix_remove_allowed(t *testing.T) {
	t.Log("\n\n+++++++++++++ SUBINTERFACE REMOVE PREFIX WITH VIP-LESS VRRP ++++++++++++")

	preReqMap := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid": "19",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	url := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]"
	t.Run("Delete prefix with VIP-less VRRP allowed", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	cleanupTbl := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0":             "",
			"Ethernet0|20.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_subinterface_ipv4_vrrp_vip_parent_subnet_rejected(t *testing.T) {
	t.Log("\n\n+++++++++++++ SUBINTERFACE IPv4 VRRP VIP PARENT SUBNET VALIDATION ++++++++++++")

	preReqMap := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": map[string]interface{}{
				"vid":          "19",
				"vip@":         "20.0.0.254",
				"preempt":      "true",
				"priority":     "110",
				"adv_interval": "1",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	baseURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]"
	expErr := tlerr.InvalidArgsError{Format: "VIP 10.0.0.250 is not in the same subnet of parent interface Ethernet0"}
	t.Run("Patch subinterface VIP outside parent subnet rejected", processSetRequest(baseURL+"/config/virtual-address", "{\"virtual-address\":[\"10.0.0.250\"]}", "PATCH", true, expErr))

	cleanupTbl := map[string]interface{}{
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Ethernet0|19": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}

func Test_openconfig_vrrp_address_key_validation(t *testing.T) {
	t.Log("\n\n+++++++++++++ VRRP address[ip] KEY VALIDATION AGAINST CONFIG_DB INTERFACE IPs ++++++++++++")

	defer loadVrrpIntfIPs("INTERFACE", "Ethernet0", "11.0.0.1/24", "11.0.0.5/24", "11::1/64")()
	defer loadVrrpIntfIPs("VLAN_INTERFACE", "Vlan1000", "192.168.0.2/24")()

	preReqMap := map[string]interface{}{
		"VRRP": map[string]interface{}{
			"Ethernet0|2": map[string]interface{}{
				"vid": "2", "vip@": "11.0.0.10", "priority": "150",
				"preempt": "true", "adv_interval": "1",
			},
			"Vlan1000|10": map[string]interface{}{
				"vid": "10", "vip@": "192.168.0.1", "priority": "120",
				"preempt": "true", "adv_interval": "1",
			},
		},
		"VRRP6": map[string]interface{}{
			"Ethernet0|3": map[string]interface{}{
				"vid": "3", "vip@": "11::10", "priority": "130",
				"preempt": "true", "adv_interval": "1",
			},
		},
	}
	loadDB(db.ConfigDB, preReqMap)
	defer unloadDB(db.ConfigDB, map[string]interface{}{
		"VRRP":  map[string]interface{}{"Ethernet0|2": "", "Vlan1000|10": ""},
		"VRRP6": map[string]interface{}{"Ethernet0|3": ""},
	})

	subV4 := func(ip string) string {
		return "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=" + ip + "]/vrrp/vrrp-group[virtual-router-id=2]"
	}
	subV6 := func(ip string) string {
		return "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=" + ip + "]/vrrp/vrrp-group[virtual-router-id=3]"
	}
	vlanV4 := func(ip string) string {
		return "/openconfig-interfaces:interfaces/interface[name=Vlan1000]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=" + ip + "]/vrrp/vrrp-group[virtual-router-id=10]"
	}
	prio := func(p int) string { return "{\"openconfig-if-ip:priority\":" + strconv.Itoa(p) + "}" }

	user := UserRoles{Name: "admin", Roles: []string{"admin"}}
	const notConfigured = "is not configured on interface"
	mustReject := func(t *testing.T, op string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected rejection for unconfigured address, got success", op)
		}
		if !strings.Contains(err.Error(), notConfigured) {
			t.Fatalf("%s: unexpected error %q, want it to contain %q", op, err.Error(), notConfigured)
		}
	}
	rejectGet := func(url string) func(*testing.T) {
		return func(t *testing.T) {
			_, err := Get(GetRequest{Path: url, User: user})
			mustReject(t, "GET "+url, err)
		}
	}
	rejectPatch := func(url, payload string) func(*testing.T) {
		return func(t *testing.T) {
			_, err := Update(SetRequest{Path: url, Payload: []byte(payload)})
			mustReject(t, "PATCH "+url, err)
		}
	}
	rejectDelete := func(url string) func(*testing.T) {
		return func(t *testing.T) {
			_, err := Delete(SetRequest{Path: url})
			mustReject(t, "DELETE "+url, err)
		}
	}

	// Configured addresses succeed
	t.Run("GET subintf ipv4 via configured 11.0.0.1", processGetRequest(subV4("11.0.0.1")+"/config/priority", nil, prio(150), false))
	t.Run("GET subintf ipv6 via configured 11::1", processGetRequest(subV6("11::1")+"/config/priority", nil, prio(130), false))
	t.Run("GET routed-vlan via configured 192.168.0.2", processGetRequest(vlanV4("192.168.0.2")+"/config/priority", nil, prio(120), false))

	// Unconfigured host, are rejected
	t.Run("GET subintf ipv4 unconfigured 11.0.0.42", rejectGet(subV4("11.0.0.42")+"/config/priority"))
	t.Run("GET subintf ipv4 VIP 11.0.0.10", rejectGet(subV4("11.0.0.10")+"/config/priority"))
	t.Run("GET subintf ipv4 group unconfigured 11.0.0.42", rejectGet(subV4("11.0.0.42")))
	t.Run("GET subintf ipv6 unconfigured 11::42", rejectGet(subV6("11::42")+"/config/priority"))
	t.Run("GET routed-vlan unconfigured 192.168.0.7", rejectGet(vlanV4("192.168.0.7")+"/state"))
	t.Run("GET routed-vlan VIP 192.168.0.1", rejectGet(vlanV4("192.168.0.1")+"/state"))
	t.Run("PATCH subintf ipv4 unconfigured", rejectPatch(subV4("11.0.0.42")+"/config/priority", "{\"priority\":105}"))
	t.Run("PATCH subintf ipv6 unconfigured", rejectPatch(subV6("11::42")+"/config/priority", "{\"priority\":105}"))
	t.Run("PATCH routed-vlan unconfigured", rejectPatch(vlanV4("192.168.0.7")+"/config/priority", "{\"priority\":105}"))
	t.Run("DELETE subintf ipv4 group unconfigured", rejectDelete(subV4("11.0.0.42")))
	t.Run("DELETE routed-vlan group unconfigured", rejectDelete(vlanV4("192.168.0.7")))
	time.Sleep(1 * time.Second)

	t.Run("Verify subintf ipv4 unchanged", processGetRequest(subV4("11.0.0.1")+"/config/priority", nil, prio(150), false))
	t.Run("Verify subintf ipv6 unchanged", processGetRequest(subV6("11::1")+"/config/priority", nil, prio(130), false))
	t.Run("Verify routed-vlan unchanged", processGetRequest(vlanV4("192.168.0.2")+"/config/priority", nil, prio(120), false))

	t.Run("GET subintf ipv4 via second configured 11.0.0.5", processGetRequest(subV4("11.0.0.5")+"/config/priority", nil, prio(150), false))
	t.Run("PATCH subintf ipv4 via second configured 11.0.0.5", processSetRequest(subV4("11.0.0.5")+"/config/priority", "{\"priority\":140}", "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("GET subintf ipv4 via first configured sees patch", processGetRequest(subV4("11.0.0.1")+"/config/priority", nil, prio(140), false))
}

func Test_openconfig_vrrp_replace_virtual_router_id_rejected(t *testing.T) {
	t.Log("\n\n+++++++++++++ VRRP REPLACE virtual-router-id LEAF REJECTED ++++++++++++")

	preReqMap := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": map[string]interface{}{},
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": map[string]interface{}{
				"vid":          "25",
				"vip@":         "50.0.0.254/24",
				"preempt":      "true",
				"priority":     "100",
				"adv_interval": "1",
			},
		},
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": map[string]interface{}{},
		},
	}
	loadDB(db.ConfigDB, preReqMap)

	replaceErr := tlerr.NotSupportedError{Format: "Replace is not supported for virtual-router-id. Delete the VRRP group and create it under the new virtual-router-id."}

	v4RoutedURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=50.0.0.1]/vrrp/vrrp-group[virtual-router-id=25]/config/virtual-router-id"
	t.Run("Replace routed-vlan ipv4 virtual-router-id leaf rejected", processSetRequest(v4RoutedURL, "25", "PUT", true, replaceErr))
	t.Run("Replace routed-vlan ipv4 virtual-router-id leaf rejected mismatched value", processSetRequest(v4RoutedURL, "11", "PUT", true, replaceErr))

	v6RoutedURL := "/openconfig-interfaces:interfaces/interface[name=Vlan100]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=29]/config/virtual-router-id"
	t.Run("Replace routed-vlan ipv6 virtual-router-id leaf rejected", processSetRequest(v6RoutedURL, "29", "PUT", true, replaceErr))

	subIntfURL := "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=20.0.0.1]/vrrp/vrrp-group[virtual-router-id=19]/config/virtual-router-id"
	t.Run("Replace subinterface ipv4 virtual-router-id leaf rejected", processSetRequest(subIntfURL, "19", "PUT", true, replaceErr))

	cleanupTbl := map[string]interface{}{
		"VLAN_INTERFACE": map[string]interface{}{
			"Vlan100|50.0.0.1/24": "",
		},
		"VRRP": map[string]interface{}{
			"Vlan100|25": "",
		},
		"INTERFACE": map[string]interface{}{
			"Ethernet0|20.0.0.1/24": "",
		},
	}
	unloadDB(db.ConfigDB, cleanupTbl)
}
