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
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
)

func Test_openconfig_interfaces(t *testing.T) {
	var url, url_input_body_json string
	var pre_req_map map[string]interface{}

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
	pre_req_map = map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{"admin_status": "up", "mtu": "9000"}}}
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

	t.Log("\n\n+++++++++++++ Performing Delete on interfaces/interface[name=Ethernet88]/config node ++++++++++++")
	pre_req_map = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet88": map[string]interface{}{"mtu": "9100"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet88]/config"
	del_not_supported_msg := "Delete operation not supported for this path - /openconfig-interfaces:interfaces/interface/config"
	del_not_supported := tlerr.InvalidArgsError{Format: del_not_supported_msg}
	t.Run("Test delete on interfaces/interface[name=Ethernet88]/config node", processDeleteRequest(url, true, del_not_supported))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet88": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
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
