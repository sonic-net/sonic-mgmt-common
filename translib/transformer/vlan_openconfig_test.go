////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2025 Dell, Inc.                                                 //
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
	// "github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
)

func Test_openconfig_vlan_interface(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n++++++++++++ CREATING VLAN INTERFACE ++++++++++++")

	t.Log("\n\n--- PATCH to Create VLAN 10 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interfaces\":{\"interface\":[{\"name\":\"Vlan10\",\"config\":{\"name\":\"Vlan10\",\"mtu\":9000,\"enabled\":true}}]}}"
	t.Run("Test Create VLAN 10", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creation (PATCH) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9000, \"name\": \"Vlan10\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to Create VLAN 20, 30, and 40 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interfaces\":{\"interface\":[{\"name\":\"Vlan20\",\"config\":{\"name\":\"Vlan20\",\"mtu\":9000,\"enabled\":true}}, {\"name\":\"Vlan30\",\"config\":{\"name\":\"Vlan30\",\"mtu\":9100,\"enabled\":true}}, {\"name\":\"Vlan40\",\"config\":{\"name\":\"Vlan40\",\"mtu\":9100,\"enabled\":true}}]}}"
	t.Run("Test Create VLAN 20,30,40", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (PATCH) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9000, \"name\": \"Vlan20\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (PATCH) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9100, \"name\": \"Vlan30\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (PATCH) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan40]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9100, \"name\": \"Vlan40\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST to Create VLAN 50 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Vlan50\",\"config\":{\"name\":\"Vlan50\",\"mtu\":9000,\"enabled\":true}}]}"
	t.Run("Test Create VLAN 50", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creation (POST) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan50]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9000, \"name\": \"Vlan50\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST to Create VLAN 60, 70, and 80 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Vlan60\",\"config\":{\"name\":\"Vlan60\",\"mtu\":9000,\"enabled\":true}}, {\"name\":\"Vlan70\",\"config\":{\"name\":\"Vlan70\",\"mtu\":9100,\"enabled\":true}}, {\"name\":\"Vlan80\",\"config\":{\"name\":\"Vlan80\",\"mtu\":9100,\"enabled\":true}}]}"
	t.Run("Test Create VLAN 60,70,80", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (POST) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan60]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9000, \"name\": \"Vlan60\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (POST) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan70]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9100, \"name\": \"Vlan70\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN Creations (POST) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan80]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9100, \"name\": \"Vlan80\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ GET LEVELS ON VLAN INTERFACE ++++++++++++")

	// TODO - UPDATE WITH ROUTED-VLAN STRUCTURE
	t.Log("\n\n--- GET VLAN (interface level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	expected_get_json = "{\"openconfig-interfaces:interface\":[{\"config\":{\"enabled\":true,\"mtu\":9000,\"name\":\"Vlan10\"},\"name\":\"Vlan10\",\"state\":{\"admin-status\":\"UP\",\"enabled\":true,\"mtu\":9000,\"name\":\"Vlan10\"},\"subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}},\"state\":{\"index\":0}}]}}]}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- GET VLAN (leaf level mtu) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config/mtu"
	expected_get_json = "{\"openconfig-interfaces:mtu\":9000}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- GET VLAN (leaf level enabled) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config/enabled"
	expected_get_json = "{\"openconfig-interfaces:enabled\":true}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ UPDATE VLAN INTERFACE ++++++++++++")

	t.Log("\n\n--- PATCH VLAN interface (leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config/mtu"
	url_input_body_json = "{\"openconfig-interfaces:mtu\":9100}"
	t.Run("Test modify VLAN 10", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN modification ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9100, \"name\": \"Vlan10\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface (config) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\":{\"name\":\"Vlan10\",\"mtu\":9000,\"enabled\":false}}"
	t.Run("Test modify VLAN 10", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN modification ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": false, \"mtu\": 9000, \"name\": \"Vlan10\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	url_input_body_json = "{\"openconfig-interfaces:config\":{\"name\":\"Vlan10\",\"mtu\":9100,\"enabled\":false}}"
	t.Run("Test modify VLAN 10", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN modification ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": false, \"mtu\": 9100, \"name\": \"Vlan10\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Vlan10\",\"config\":{\"name\":\"Vlan10\",\"mtu\":9000,\"enabled\":true}}]}"
	t.Run("Test replace VLAN 10", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN modification ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"enabled\": true, \"mtu\": 9000, \"name\": \"Vlan10\"}}"
	t.Run("Test GET VLAN interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	// t.Log("\n\n++++++++++++ CHECK STATE ATTRIBUTES ++++++++++++")

	// cleanuptbl := map[string]interface{}{"VLAN_TABLE": map[string]interface{}{"Vlan10": ""}}
	// unloadDB(db.ApplDB, cleanuptbl)
	// pre_req_map := map[string]interface{}{"VLAN_TABLE": map[string]interface{}{"Vlan10": map[string]interface{}{"admin_status": "up", "mtu": "9000", "enabled": "true"}}}
	// loadDB(db.ApplDB, pre_req_map)

	// t.Log("\n\n--- GET VLAN (state level) ---")
	// url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/state"
	// expected_get_json = "{\"openconfig-interfaces:state\":{\"admin-status\":\"UP\",\"enabled\":true,\"mtu\":9000,\"name\":\"Vlan10\"}}"
	// t.Run("Test GET VLAN interface state config ", processGetRequest(url, nil, expected_get_json, false))
	// time.Sleep(1 * time.Second)

	// t.Log("\n\n--- GET VLAN (leaf level admin status) ---")
	// url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/state/admin-status"
	// expected_get_json = "{\"openconfig-interfaces:admin-status\":\"UP\"}"
	// t.Run("Test GET VLAN interface state config ", processGetRequest(url, nil, expected_get_json, false))
	// time.Sleep(1 * time.Second)

	// unloadDB(db.ApplDB, cleanuptbl)
}

func Test_openconfig_vlan_member(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n++++++++++++ ADD VLAN MEMBERS ++++++++++++")

	t.Log("\n\n--- PATCH to add VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config"
	url_input_body_json = "{\"openconfig-vlan:config\":{\"interface-mode\":\"ACCESS\",\"access-vlan\":10}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json := "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"access-vlan\":10,\"interface-mode\":\"ACCESS\"},\"state\":{\"access-vlan\":10,\"interface-mode\":\"ACCESS\"}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to add VLAN member (Eth, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config"
	url_input_body_json = "{\"openconfig-vlan:config\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[30,40]}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"access-vlan\":10,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[30,40]},\"state\":{\"access-vlan\":10,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[30,40]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to create PortChannel interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config"
	url_input_body_json = "{\"openconfig-vlan:config\":{\"interface-mode\":\"ACCESS\",\"access-vlan\":10}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to add VLAN member (PC, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config"
	url_input_body_json = "{\"openconfig-vlan:config\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[10,30]}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config"
	expected_get_json = "{\"openconfig-vlan:config\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[10,30]}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to add VLAN member (PC, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config"
	url_input_body_json = "{\"openconfig-vlan:config\":{\"access-vlan\":20}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config"
	expected_get_json = "{\"openconfig-vlan:config\":{\"access-vlan\":20,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[10,30]}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ UPDATE VLAN MEMBERS ++++++++++++")

	t.Log("\n\n--- PATCH VLAN member (Eth, access-vlan leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	url_input_body_json = "{\"openconfig-vlan:access-vlan\":20}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, access-vlan leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	expected_get_json = "{\"openconfig-vlan:access-vlan\":20}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN member (Eth, access-vlan leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	url_input_body_json = "{\"openconfig-vlan:access-vlan\":10}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, access-vlan leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	expected_get_json = "{\"openconfig-vlan:access-vlan\":10}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN member (Eth, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	url_input_body_json = "{\"openconfig-vlan:trunk-vlans\":[20,30]}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{\"openconfig-vlan:trunk-vlans\":[20,30]}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN member (Eth, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	url_input_body_json = "{\"openconfig-vlan:trunk-vlans\":[40]}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{\"openconfig-vlan:trunk-vlans\":[20,30,40]}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	url_input_body_json = "{\"openconfig-vlan:trunk-vlans\":[40]}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{\"openconfig-vlan:trunk-vlans\":[10,30,40]}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	url_input_body_json = "{\"openconfig-vlan:trunk-vlans\":[10,30]}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{\"openconfig-vlan:trunk-vlans\":[10,30]}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	url_input_body_json = "{\"openconfig-vlan:trunk-vlans\":[40]}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, trunk-vlans leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{\"openconfig-vlan:trunk-vlans\":[10,30,40]}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	// Changing to ACCESS mode with trunk VLANs configured does nothing
	t.Log("\n\n--- PATCH VLAN member (Eth, interface-mode leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/interface-mode"
	url_input_body_json = "{\"openconfig-vlan:interface-mode\":\"ACCESS\"}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, interface-mode leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/interface-mode"
	expected_get_json = "{\"openconfig-vlan:interface-mode\":\"TRUNK\"}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	// Changing to TRUNK mode with only access VLAN configured does nothing
	t.Log("\n\n--- PUT to replace VLAN member (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"ACCESS\",\"access-vlan\":10}}}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN member (Eth, interface-mode leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/interface-mode"
	url_input_body_json = "{\"openconfig-vlan:interface-mode\":\"TRUNK\"}"
	t.Run("Test update VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, interface-mode leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/interface-mode"
	expected_get_json = "{\"openconfig-vlan:interface-mode\":\"ACCESS\"}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ DELETE VLAN MEMBERS ++++++++++++")

	// Reset VLAn config
	t.Log("\n\n--- PUT to replace VLAN member (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":10, \"trunk-vlans\":[20,30,40]}}}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,30,40]},\"state\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,30,40]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted resource at VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan"
	err_str := "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN member", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (Eth, one trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans[trunk-vlans=40]"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (Eth, one trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,30]},\"state\":{\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,30]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (Eth, all trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (Eth, all trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted resource at VLAN member (Eth, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{}"
	t.Run("Test GET on deleted VLAN member", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (PC, one trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans[trunk-vlans=10]"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (PC, one trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":20,\"trunk-vlans\":[30,40]},\"state\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":20,\"trunk-vlans\":[30,40]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (PC, all trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (PC, all trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"access-vlan\":20,\"interface-mode\":\"ACCESS\"},\"state\":{\"access-vlan\":20,\"interface-mode\":\"ACCESS\"}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted resource at VLAN member (PC, trunk) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"
	expected_get_json = "{}"
	t.Run("Test GET on deleted VLAN member", processGetRequest(url, nil, expected_get_json, false))
	// err_str = "Resource not found"
	// expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	// t.Run("Test GET on deleted VLAN member", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN member (PC, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/access-vlan"
	t.Run("Test delete VLAN member", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted resource at VLAN member (PC, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/access-vlan"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN member", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN member (Eth, access) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	expected_get_json = "{}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ DELETE VLAN MEMBERS (container level) ++++++++++++")

	t.Log("\n\n--- PATCH to add VLAN member (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":10,\"trunk-vlans\":[20,30,40]}}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to add VLAN member (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":10,\"trunk-vlans\":[20,40]}}}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"access-vlan\":10,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,40]},\"state\":{\"access-vlan\":10,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[20,40]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE all VLAN members (Eth, config) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config"
	t.Run("Test delete all VLAN members", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN members (Eth, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan"
	expected_get_json = "{}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to add VLAN member (PC, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":30,\"trunk-vlans\":[10,20]}}}"
	t.Run("Test add VLAN member", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to add VLAN member (PC, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	url_input_body_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"interface-mode\":\"TRUNK\",\"access-vlan\":30,\"trunk-vlans\":[10,20,40]}}}"
	t.Run("Test replace VLAN member", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN member (PC, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	expected_get_json = "{\"openconfig-vlan:switched-vlan\":{\"config\":{\"access-vlan\":30,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[10,20,40]},\"state\":{\"access-vlan\":30,\"interface-mode\":\"TRUNK\",\"trunk-vlans\":[10,20,40]}}}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE all VLAN members (PC, switched-vlan) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan"
	t.Run("Test delete all VLAN members", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN members (PC, switched-vlan config) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config"
	expected_get_json = "{}"
	t.Run("Test GET VLAN member config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

}

func Test_openconfig_vlan_interface_delete(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n++++++++++++ DELETE VLAN INTERFACE ++++++++++++")

	t.Log("\n\n--- POST to Update Vlan 10 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Vlan10\",\"config\":{\"name\":\"Vlan10\",\"mtu\":9000,\"enabled\":true, \"description\":\"test_vlan\"}}]}"
	t.Run("Test Update VLAN 10", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface attribute (description) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config/description"
	t.Run("Test delete VLAN interface attribute", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface attribute (mtu) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/config/description"
	err_str := "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface attribute", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 10 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 20 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 30 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 40 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan40]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan40]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 50 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan50]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan50]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 60 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan60]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan60]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 70 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan70]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan70]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete Vlan 80 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan80]"
	t.Run("Test delete VLAN interface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan80]"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted VLAN interface", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)

}
