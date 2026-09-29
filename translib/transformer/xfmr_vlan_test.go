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
	"strconv"
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/transformer"
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

	t.Log("\n\n--- GET VLAN (interface level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]"
	expected_get_json = "{\"openconfig-interfaces:interface\":[{\"config\":{\"enabled\":true,\"mtu\":9000,\"name\":\"Vlan10\"},\"name\":\"Vlan10\",\"openconfig-vlan:routed-vlan\":{\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}}},\"state\":{\"name\":\"Vlan10\"},\"subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}},\"state\":{\"index\":0}}]}}]}"
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

	t.Log("\n\n++++++++++++ CHECK STATE ATTRIBUTES ++++++++++++")

	cleanuptbl := map[string]interface{}{"VLAN_TABLE": map[string]interface{}{"Vlan10": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map := map[string]interface{}{"VLAN_TABLE": map[string]interface{}{"Vlan10": map[string]interface{}{"admin_status": "up", "mtu": "9000", "enabled": "true"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- GET VLAN (state level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/state"
	expected_get_json = "{\"openconfig-interfaces:state\":{\"admin-status\":\"UP\",\"enabled\":true,\"mtu\":9000,\"name\":\"Vlan10\"}}"
	t.Run("Test GET VLAN interface state config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- GET VLAN (leaf level admin status) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/state/admin-status"
	expected_get_json = "{\"openconfig-interfaces:admin-status\":\"UP\"}"
	t.Run("Test GET VLAN interface state config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ApplDB, cleanuptbl)
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

	t.Log("\n\n--- PUT to create PortChannel interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel12]"
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"PortChannel12\",\"config\":{\"name\":\"PortChannel12\",\"mtu\":9000,\"description\":\"tst_pc\",\"enabled\":true},\"openconfig-if-aggregate:aggregation\":{\"config\":{\"min-links\":3}}}]}"
	t.Run("Test create PC interface", processSetRequest(url, url_input_body_json, "PUT", false, nil))
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

	// Verify that changing to ACCESS mode with trunk VLANs configured does nothing
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

	// Verify that changing to TRUNK mode with only access VLAN configured does nothing
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

	// Reset VLAN config
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

func Test_openconfig_vlan_interface_ip(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n++++++++++++ CONFIGURE IPv4 VLAN INTERFACE ++++++++++++")

	t.Log("\n\n--- PATCH VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2.2.2.2\",\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24}}]}}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=2.2.2.2]"
	expected_get_json := "{\"openconfig-if-ip:address\":[{\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24},\"ip\":\"2.2.2.2\"}]}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"4.4.5.1\",\"config\":{\"ip\":\"4.4.5.1\",\"prefix-length\":24}}]}}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]"
	expected_get_json = "{\"openconfig-if-ip:address\":[{\"config\":{\"ip\":\"4.4.5.1\",\"prefix-length\":24},\"ip\":\"4.4.5.1\"}]}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:address\":[{\"ip\":\"2.2.2.2\",\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24}}]}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24},\"ip\":\"2.2.2.2\"}]}}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 address (routed-vlan container) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan"
	url_input_body_json = "{\"openconfig-vlan:routed-vlan\": {\"config\": {\"vlan\": \"Vlan20\" },\"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"ip\": \"16.16.16.16\", \"config\": {\"ip\": \"16.16.16.16\", \"prefix-length\": 24}}]}}}}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan"
	expected_get_json = "{\"openconfig-vlan:routed-vlan\":{\"config\":{\"vlan\":\"Vlan20\"},\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"16.16.16.16\",\"prefix-length\":24},\"ip\":\"16.16.16.16\"}]}},\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}},\"state\":{\"vlan\":\"Vlan20\"}}}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface ID (config)---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/config"
	expected_get_json = "{\"openconfig-vlan:config\":{\"vlan\":\"Vlan20\"}}"
	t.Run("Test GET VLAN interface ID (config) ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 address (IPv4 container) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4"
	url_input_body_json = "{\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"ip\":\"8.8.8.8\",\"config\":{\"ip\":\"8.8.8.8\",\"prefix-length\":24}}]}}}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4"
	expected_get_json = "{\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"8.8.8.8\",\"prefix-length\":24},\"ip\":\"8.8.8.8\"}]}}}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"4.4.5.1\",\"config\":{\"ip\":\"4.4.5.1\",\"prefix-length\":32}}]}}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 address (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]/config/prefix-length"
	url_input_body_json = "{\"openconfig-if-ip:prefix-length\":24}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]/config/prefix-length"
	expected_get_json = "{\"openconfig-if-ip:prefix-length\":24}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST VLAN interface IPv4 address (address config level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]/config"
	url_input_body_json = "{\"openconfig-if-ip:prefix-length\":32}"
	t.Run("Test configure VLAN IPv4", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 address (address config level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]/config"
	expected_get_json = "{\"openconfig-if-ip:config\":{\"ip\":\"4.4.5.1\",\"prefix-length\":32}}"
	t.Run("Test GET VLAN interface IPv4 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ CONFIGURE IPv6 VLAN INTERFACE ++++++++++++")

	t.Log("\n\n--- PATCH VLAN interface IPv6 enabled ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config/enabled"
	url_input_body_json = "{\"openconfig-if-ip:enabled\":true}"
	t.Run("Test configure VLAN IPv6", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 enabled ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config/enabled"
	expected_get_json = "{\"openconfig-if-ip:enabled\":true}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN interface IPv6 enabled ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config/enabled"
	url_input_body_json = "{\"openconfig-if-ip:enabled\":false}"
	t.Run("Test configure VLAN IPv6", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 enabled ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config/enabled"
	expected_get_json = "{\"openconfig-if-ip:enabled\":false}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST VLAN interface IPv6 enabled (config level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config"
	url_input_body_json = "{\"openconfig-if-ip:enabled\":true}"
	t.Run("Test configure VLAN IPv6", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 enabled ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/config"
	expected_get_json = "{\"openconfig-if-ip:config\":{\"enabled\":true}}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2001:4860:4860::8888\",\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64}}]}}"
	t.Run("Test configure VLAN IPv6 address", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:4860:4860::8888]"
	expected_get_json = "{\"openconfig-if-ip:address\":[{\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64},\"ip\":\"2001:4860:4860::8888\"}]}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2001:4860:4860::8844\",\"config\":{\"ip\":\"2001:4860:4860::8844\",\"prefix-length\":64}}]}}"
	t.Run("Test configure VLAN IPv6 address", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:4860:4860::8844]"
	expected_get_json = "{\"openconfig-if-ip:address\":[{\"config\":{\"ip\":\"2001:4860:4860::8844\",\"prefix-length\":64},\"ip\":\"2001:4860:4860::8844\"}]}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- POST VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:address\":[{\"ip\":\"2001:4860:4860::8888\",\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64}}]}"
	t.Run("Test configure VLAN IPv6 address", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"2001:4860:4860::8844\",\"prefix-length\":64},\"ip\":\"2001:4860:4860::8844\"},{\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64},\"ip\":\"2001:4860:4860::8888\"}]}}"
	t.Run("Test GET VLAN interface IPv6 config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 address (routed-vlan container) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan"
	url_input_body_json = "{\"openconfig-vlan:routed-vlan\":{\"config\":{\"vlan\":\"Vlan30\"},\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"ip\":\"2606:4700:4700::1111\",\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64}}]}}}}"
	t.Run("Test configure VLAN IPv6 address", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 address (routed-vlan container)---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan"
	expected_get_json = "{\"openconfig-vlan:routed-vlan\":{\"config\":{\"vlan\":\"Vlan30\"},\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"8.8.8.8\",\"prefix-length\":24},\"ip\":\"8.8.8.8\"}]}},\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1111\"}]},\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}},\"state\":{\"vlan\":\"Vlan30\"}}}"
	t.Run("Test GET VLAN interface IPv6 config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 address (IPv6 container) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6"
	url_input_body_json = "{\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"ip\":\"2606:4700:4700::1001\",\"config\":{\"ip\":\"2606:4700:4700::1001\",\"prefix-length\":64}}]}}}"
	t.Run("Test configure VLAN IPv6 address", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 address (IPv6 container)---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6"
	expected_get_json = "{\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2606:4700:4700::1001\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1001\"},{\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1111\"}]},\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}}}"
	t.Run("Test GET VLAN interface IPv6 config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n++++++++++++ CHECK STATE ATTRIBUTES ++++++++++++")

	cleanuptbl := map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map := map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10": map[string]interface{}{"vlan": "Vlan10"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- GET VLAN (state level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/state"
	expected_get_json = "{\"openconfig-vlan:state\":{\"vlan\":\"Vlan10\"}}"
	t.Run("Test GET VLAN interface state config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ApplDB, cleanuptbl)

	cleanuptbl = map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10:4.4.5.1/32": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10:4.4.5.1/32": map[string]interface{}{"ip": "4.4.5.1", "prefix-length": 32}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- GET VLAN (state level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.5.1]/state"
	expected_get_json = "{\"openconfig-if-ip:state\":{\"ip\":\"4.4.5.1\",\"prefix-length\":32}}"
	t.Run("Test GET VLAN interface IPv4 state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ApplDB, cleanuptbl)

	cleanuptbl = map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10:2001:4860:4860::8844/64": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Vlan10:2001:4860:4860::8844/64": map[string]interface{}{"ip": "2001:4860:4860::8844", "prefix-length": 64}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- GET VLAN (state level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2001:4860:4860::8844]/state"
	expected_get_json = "{\"openconfig-if-ip:state\":{\"ip\":\"2001:4860:4860::8844\",\"prefix-length\":64}}"
	t.Run("Test GET VLAN interface IPv6 state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ApplDB, cleanuptbl)

	t.Log("\n\n++++++++++++ CLEAN UP VLAN INTERFACES IP CONFIG ++++++++++++")

	t.Log("\n\n--- DELETE VLAN interface IPv4 (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=8.8.8.8]/config/prefix-length"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IP (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=8.8.8.8]/config/prefix-length"
	expected_get_json = "{\"openconfig-if-ip:prefix-length\":0}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv4 (specify address) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.4.4]"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IP (specify address) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=4.4.4.4]"
	expected_get_json = "{\"openconfig-if-ip:address\":[{\"ip\":\"4.4.4.4\"}]}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv4 (all addresses level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IP (all addresses level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan20]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	expected_get_json = "{}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2.2.2.2\",\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24}}]}}"
	t.Run("Test configure VLAN IPv4 address", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv4 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4/addresses/address[ip=2.2.2.2]"
	expected_get_json = "{\"openconfig-if-ip:address\":[{\"config\":{\"ip\":\"2.2.2.2\",\"prefix-length\":24},\"ip\":\"2.2.2.2\"}]}"
	t.Run("Test GET VLAN interface IPv4", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv4 (IPv4 level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IPv4 (IPv4 level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv4"
	expected_get_json = "{}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv6 (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2606:4700:4700::1111]/config/prefix-length"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IPv6 (prefix-length leaf) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2606:4700:4700::1111]/config/prefix-length"
	expected_get_json = "{\"openconfig-if-ip:prefix-length\":0}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2606:4700:4700::1111\",\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64}}]}}"
	t.Run("Test configure VLAN IPv6 address (temp)", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"2606:4700:4700::1001\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1001\"},{\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1111\"}]}}"
	t.Run("Test GET VLAN interface IPv6", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv6 (specify address) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses/address[ip=2606:4700:4700::1111]"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IPv6 (specify address) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"2606:4700:4700::1001\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1001\"}]}}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv6 (all addresses level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IPv6 (all addresses level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan30]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface IPv6 (IPv6 level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface IPv6 (IPv6 level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6"
	expected_get_json = "{\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}}}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2001:4860:4860::8888\",\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64}}]}}"
	t.Run("Test configure VLAN IPv6 address (temp)", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv6 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"ip\":\"2606:4700:4700::1111\",\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64}}]}}"
	t.Run("Test configure VLAN IPv6 address (temp)", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH VLAN interface IPv4 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan"
	url_input_body_json = "{\"openconfig-vlan:routed-vlan\": {\"config\": {\"vlan\": \"Vlan20\" },\"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"ip\": \"16.16.16.16\", \"config\": {\"ip\": \"16.16.16.16\", \"prefix-length\": 32}}]}}}}"
	t.Run("Test configure VLAN IPv4 address (temp)", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify VLAN interface IPv6 (temp) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan"
	expected_get_json = "{\"openconfig-vlan:routed-vlan\":{\"config\":{\"vlan\":\"Vlan10\"},\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"16.16.16.16\",\"prefix-length\":32},\"ip\":\"16.16.16.16\"}]}},\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2001:4860:4860::8888\",\"prefix-length\":64},\"ip\":\"2001:4860:4860::8888\"},{\"config\":{\"ip\":\"2606:4700:4700::1111\",\"prefix-length\":64},\"ip\":\"2606:4700:4700::1111\"}]},\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}},\"state\":{\"vlan\":\"Vlan10\"}}}"
	t.Run("Test GET VLAN interface IP (temp)", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE VLAN interface all IP (routed-vlan level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan"
	t.Run("Test delete VLAN interface IP config", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify deleted VLAN interface all IP (routed-vlan level) ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Vlan10]/openconfig-vlan:routed-vlan"
	expected_get_json = "{\"openconfig-vlan:routed-vlan\":{\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false},\"state\":{\"enabled\":false}}}}"
	t.Run("Test GET deleted VLAN interface IP config ", processGetRequest(url, nil, expected_get_json, false))
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

func vlanCatalogPath(vlanID int, extra string) string {
	path := "/openconfig-vlan:vlans/vlan[vlan-id=" + strconv.Itoa(vlanID) + "]"
	if extra != "" {
		path += extra
	}
	return path
}

func vlanCatalogDb(vlanID int, admin string) map[string]interface{} {
	name := "Vlan" + strconv.Itoa(vlanID)
	fields := map[string]interface{}{"vlanid": strconv.Itoa(vlanID)}
	if admin != "" {
		fields["admin_status"] = admin
	}
	return map[string]interface{}{
		"VLAN": map[string]interface{}{name: fields},
	}
}

func cleanupVlanCatalog(t *testing.T, vlanIDs ...int) {
	t.Helper()
	for _, id := range vlanIDs {
		processDeleteRequest(vlanCatalogPath(id, ""), true)(t)
	}
	time.Sleep(1 * time.Second)
}

func vlanCatalogGetFixture() map[string]interface{} {
	return map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan100": map[string]interface{}{"vlanid": "100", "admin_status": "up"},
			"Vlan200": map[string]interface{}{"vlanid": "200", "admin_status": "down"},
			"Vlan250": map[string]interface{}{"vlanid": "250"},
			"Vlan251": map[string]interface{}{"vlanid": "251", "admin_status": "DOWN"},
		},
		"VLAN_MEMBER": map[string]interface{}{
			"Vlan100|Ethernet0": map[string]interface{}{"tagging_mode": "untagged"},
			"Vlan100|Ethernet4": map[string]interface{}{"tagging_mode": "tagged"},
		},
	}
}

func vlanCatalogApplFixture() map[string]interface{} {
	return map[string]interface{}{
		"VLAN_TABLE": map[string]interface{}{
			"Vlan100": map[string]interface{}{"admin_status": "up", "mtu": "9100"},
			"Vlan200": map[string]interface{}{"admin_status": "down", "mtu": "9100"},
			"Vlan251": map[string]interface{}{"admin_status": "DOWN", "mtu": "9100"},
		},
		"VLAN_MEMBER_TABLE": map[string]interface{}{
			"Vlan100:Ethernet0": map[string]interface{}{"tagging_mode": "untagged"},
			"Vlan100:Ethernet4": map[string]interface{}{"tagging_mode": "tagged"},
		},
	}
}

func Test_openconfig_vlan_catalog_set(t *testing.T) {
	setIDs := []int{2, 100, 200, 201, 4094}
	cleanupVlanCatalog(t, setIDs...)

	nameErr := tlerr.InvalidArgsError{Format: "VLAN name must be Vlan100"}
	vlan1Err := tlerr.InvalidArgsError{Format: "Invalid VLAN id 1: SONiC supports VLAN ids 2-4094"}
	mismatchIdErr := tlerr.InvalidArgsError{Format: "VLAN id in payload does not match 100"}
	stateErr := tlerr.NotSupportedError{Format: "SET not supported on /openconfig-vlan:vlans/vlan/state"}
	stateLeafErr := tlerr.NotSupportedError{Format: "SET not supported on /openconfig-vlan:vlans/vlan/state/status"}
	membersErr := tlerr.NotSupportedError{Format: "SET not supported on /openconfig-vlan:vlans/vlan/members"}

	t.Log("\n\n++++++++++++ SET CREATE ++++++++++++")
	url := vlanCatalogPath(100, "/config")
	body := `{"openconfig-vlan:config":{"vlan-id":100,"name":"Vlan100","status":"ACTIVE"}}`
	t.Run("PATCH create VLAN 100", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", vlanCatalogDb(100, "up"), false))
	t.Run("GET VLAN 100 config after create", processGetRequest(url, nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"ACTIVE","vlan-id":100}}`, false))

	url = vlanCatalogPath(200, "/config")
	body = `{"openconfig-vlan:config":{"vlan-id":200}}`
	t.Run("PATCH create VLAN 200 without name or status", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 200 defaults in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan200", vlanCatalogDb(200, "up"), false))
	t.Run("GET VLAN 200 config after create", processGetRequest(url, nil, `{"openconfig-vlan:config":{"name":"Vlan200","status":"ACTIVE","vlan-id":200}}`, false))

	url = vlanCatalogPath(2, "/config")
	body = `{"openconfig-vlan:config":{"vlan-id":2,"name":"Vlan2","status":"ACTIVE"}}`
	t.Run("PATCH create VLAN 2", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 2 in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan2", vlanCatalogDb(2, "up"), false))
	t.Run("GET VLAN 2 config after create", processGetRequest(url, nil, `{"openconfig-vlan:config":{"name":"Vlan2","status":"ACTIVE","vlan-id":2}}`, false))

	url = vlanCatalogPath(4094, "/config")
	body = `{"openconfig-vlan:config":{"vlan-id":4094,"name":"Vlan4094","status":"SUSPENDED"}}`
	t.Run("PATCH create VLAN 4094", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 4094 in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan4094", vlanCatalogDb(4094, "down"), false))
	t.Run("GET VLAN 4094 config after create", processGetRequest(url, nil, `{"openconfig-vlan:config":{"name":"Vlan4094","status":"SUSPENDED","vlan-id":4094}}`, false))

	url = "/openconfig-vlan:vlans"
	body = `{"openconfig-vlan:vlan":[{"vlan-id":201,"config":{"vlan-id":201,"name":"Vlan201","status":"SUSPENDED"}}]}`
	t.Run("POST create VLAN 201", processSetRequest(url, body, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 201 in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan201", vlanCatalogDb(201, "down"), false))
	t.Run("GET VLAN 201 config after create", processGetRequest(vlanCatalogPath(201, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan201","status":"SUSPENDED","vlan-id":201}}`, false))

	t.Log("\n\n++++++++++++ SET UPDATE AND REPLACE ++++++++++++")
	url = vlanCatalogPath(100, "/config/status")
	body = `{"openconfig-vlan:status":"SUSPENDED"}`
	t.Run("PATCH VLAN 100 status SUSPENDED", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 suspended in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", vlanCatalogDb(100, "down"), false))
	t.Run("GET VLAN 100 config after suspend", processGetRequest(vlanCatalogPath(100, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"SUSPENDED","vlan-id":100}}`, false))

	url = vlanCatalogPath(100, "/config")
	body = `{"openconfig-vlan:config":{"vlan-id":100,"name":"Vlan100"}}`
	t.Run("PATCH VLAN 100 without status", processSetRequest(url, body, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 still suspended in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", vlanCatalogDb(100, "down"), false))
	t.Run("GET VLAN 100 still SUSPENDED", processGetRequest(url, nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"SUSPENDED","vlan-id":100}}`, false))

	url = vlanCatalogPath(100, "")
	body = `{"openconfig-vlan:vlan":[{"vlan-id":100,"config":{"vlan-id":100,"name":"Vlan100"}}]}`
	t.Run("PUT replace VLAN 100 without status", processSetRequest(url, body, "PUT", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 replaced ACTIVE in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", vlanCatalogDb(100, "up"), false))
	t.Run("GET VLAN 100 config after PUT", processGetRequest(vlanCatalogPath(100, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"ACTIVE","vlan-id":100}}`, false))

	t.Log("\n\n++++++++++++ SET REJECTS ++++++++++++")
	t.Run("PATCH mismatched VLAN name rejected", processSetRequest(vlanCatalogPath(100, "/config"), `{"openconfig-vlan:config":{"name":"engineering"}}`, "PATCH", true, nameErr))
	t.Run("PATCH mismatched VLAN id rejected", processSetRequest(vlanCatalogPath(100, "/config"), `{"openconfig-vlan:config":{"vlan-id":201}}`, "PATCH", true, mismatchIdErr))
	t.Run("PATCH VLAN id 1 rejected", processSetRequest(vlanCatalogPath(1, "/config"), `{"openconfig-vlan:config":{"vlan-id":1}}`, "PATCH", true, vlan1Err))
	t.Run("PATCH VLAN state rejected", processSetRequest(vlanCatalogPath(100, "/state"), `{"openconfig-vlan:state":{"vlan-id":100,"name":"Vlan100","status":"ACTIVE"}}`, "PATCH", true, stateErr))
	t.Run("PATCH VLAN state status rejected", processSetRequest(vlanCatalogPath(100, "/state/status"), `{"openconfig-vlan:status":"SUSPENDED"}`, "PATCH", true, stateLeafErr))
	t.Run("PATCH VLAN members rejected", processSetRequest(vlanCatalogPath(100, "/members"), `{"openconfig-vlan:members":{"member":[{"state":{"interface":"Ethernet0"}}]}}`, "PATCH", true, membersErr))

	cleanupVlanCatalog(t, setIDs...)
	t.Run("Verify VLAN 100 deleted from CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", map[string]interface{}{}, false))
}

func Test_openconfig_vlan_catalog_delete_with_members(t *testing.T) {
	fixture := map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan100": map[string]interface{}{"vlanid": "100", "admin_status": "up"},
		},
		"VLAN_MEMBER": map[string]interface{}{
			"Vlan100|Ethernet0": map[string]interface{}{"tagging_mode": "untagged"},
		},
	}
	memberOnly := map[string]interface{}{
		"VLAN_MEMBER": fixture["VLAN_MEMBER"],
	}
	cleanupVlanCatalog(t, 100)
	unloadDB(db.ConfigDB, fixture)
	loadDB(db.ConfigDB, fixture)
	time.Sleep(1 * time.Second)
	defer func() {
		unloadDB(db.ConfigDB, fixture)
		cleanupVlanCatalog(t, 100)
	}()

	membersErr := tlerr.InvalidArgsError{Format: "VLAN ID 100 can not be removed. First remove all members assigned to this VLAN."}
	url := vlanCatalogPath(100, "")
	t.Run("DELETE VLAN 100 with members rejected", processDeleteRequest(url, true, membersErr))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 still in CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", vlanCatalogDb(100, "up"), false))
	t.Run("Verify VLAN member still in CONFIG_DB", verifyDbResult(rclient, "VLAN_MEMBER|Vlan100|Ethernet0", map[string]interface{}{
		"VLAN_MEMBER": map[string]interface{}{
			"Vlan100|Ethernet0": map[string]interface{}{"tagging_mode": "untagged"},
		},
	}, false))

	unloadDB(db.ConfigDB, memberOnly)
	time.Sleep(1 * time.Second)
	t.Run("DELETE VLAN 100 after members removed", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify VLAN 100 deleted from CONFIG_DB", verifyDbResult(rclient, "VLAN|Vlan100", map[string]interface{}{}, false))
}

func Test_openconfig_vlan_catalog_get(t *testing.T) {
	getIDs := []int{100, 200, 250, 251}
	fixture := vlanCatalogGetFixture()
	appl := vlanCatalogApplFixture()
	cleanupVlanCatalog(t, getIDs...)
	unloadDB(db.ConfigDB, fixture)
	unloadDB(db.ApplDB, appl)
	loadDB(db.ConfigDB, fixture)
	loadDB(db.ApplDB, appl)
	time.Sleep(1 * time.Second)
	defer func() {
		unloadDB(db.ApplDB, appl)
		unloadDB(db.ConfigDB, fixture)
		cleanupVlanCatalog(t, getIDs...)
	}()

	notFound := tlerr.NotFoundError{Format: "Resource not found"}

	t.Log("\n\n++++++++++++ GET CONFIG ++++++++++++")
	t.Run("GET VLAN 100 config", processGetRequest(vlanCatalogPath(100, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"ACTIVE","vlan-id":100}}`, false))
	t.Run("GET VLAN 100 config vlan-id", processGetRequest(vlanCatalogPath(100, "/config/vlan-id"), nil, `{"openconfig-vlan:vlan-id":100}`, false))
	t.Run("GET VLAN 100 config name", processGetRequest(vlanCatalogPath(100, "/config/name"), nil, `{"openconfig-vlan:name":"Vlan100"}`, false))
	t.Run("GET VLAN 100 config status", processGetRequest(vlanCatalogPath(100, "/config/status"), nil, `{"openconfig-vlan:status":"ACTIVE"}`, false))
	t.Run("GET VLAN 200 config", processGetRequest(vlanCatalogPath(200, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan200","status":"SUSPENDED","vlan-id":200}}`, false))
	t.Run("GET VLAN 250 config missing admin_status", processGetRequest(vlanCatalogPath(250, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan250","status":"ACTIVE","vlan-id":250}}`, false))
	t.Run("GET VLAN 251 config DOWN", processGetRequest(vlanCatalogPath(251, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan251","status":"SUSPENDED","vlan-id":251}}`, false))

	t.Log("\n\n++++++++++++ GET STATE ++++++++++++")
	t.Run("GET VLAN 100 state", processGetRequest(vlanCatalogPath(100, "/state"), nil, `{"openconfig-vlan:state":{"name":"Vlan100","status":"ACTIVE","vlan-id":100}}`, false))
	t.Run("GET VLAN 200 state vlan-id", processGetRequest(vlanCatalogPath(200, "/state/vlan-id"), nil, `{"openconfig-vlan:vlan-id":200}`, false))
	t.Run("GET VLAN 200 state name", processGetRequest(vlanCatalogPath(200, "/state/name"), nil, `{"openconfig-vlan:name":"Vlan200"}`, false))
	t.Run("GET VLAN 200 state status", processGetRequest(vlanCatalogPath(200, "/state/status"), nil, `{"openconfig-vlan:status":"SUSPENDED"}`, false))
	t.Run("GET VLAN 251 state DOWN", processGetRequest(vlanCatalogPath(251, "/state"), nil, `{"openconfig-vlan:state":{"name":"Vlan251","status":"SUSPENDED","vlan-id":251}}`, false))

	t.Log("\n\n++++++++++++ GET MEMBERS AND INSTANCE ++++++++++++")
	t.Run("GET VLAN 100 members", processGetRequest(vlanCatalogPath(100, "/members"), nil, `{"openconfig-vlan:members":{"member":[{"state":{"interface":"Ethernet0"}},{"state":{"interface":"Ethernet4"}}]}}`, false))
	t.Run("GET VLAN 200 members empty", processGetRequest(vlanCatalogPath(200, "/members"), nil, `{"openconfig-vlan:members":{}}`, false))
	t.Run("GET VLAN 100 instance", processGetRequest(vlanCatalogPath(100, ""), nil, `{"openconfig-vlan:vlan":[{"config":{"name":"Vlan100","status":"ACTIVE","vlan-id":100},"members":{"member":[{"state":{"interface":"Ethernet0"}},{"state":{"interface":"Ethernet4"}}]},"state":{"name":"Vlan100","status":"ACTIVE","vlan-id":100},"vlan-id":100}]}`, false))
	t.Run("GET missing VLAN 300", processGetRequest(vlanCatalogPath(300, ""), nil, "", true, notFound))

	t.Log("\n\n++++++++++++ GET WITHOUT APPL_DB ++++++++++++")
	t.Run("GET VLAN 250 state missing APPL_DB", processGetRequest(vlanCatalogPath(250, "/state"), nil, "", true, notFound))
	t.Run("GET VLAN 250 members missing APPL_DB", processGetRequest(vlanCatalogPath(250, "/members"), nil, "", true, notFound))
	t.Run("GET VLAN 250 instance config only", processGetRequest(vlanCatalogPath(250, ""), nil, `{"openconfig-vlan:vlan":[{"config":{"name":"Vlan250","status":"ACTIVE","vlan-id":250},"vlan-id":250}]}`, false))
}

func Test_openconfig_vlan_catalog_get_appl_divergence(t *testing.T) {
	getIDs := []int{100}
	cfg := map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan100": map[string]interface{}{"vlanid": "100", "admin_status": "up"},
		},
	}
	appl := map[string]interface{}{
		"VLAN_TABLE": map[string]interface{}{
			"Vlan100": map[string]interface{}{"admin_status": "down", "mtu": "9100"},
		},
	}
	cleanupVlanCatalog(t, getIDs...)
	unloadDB(db.ConfigDB, cfg)
	unloadDB(db.ApplDB, appl)
	loadDB(db.ConfigDB, cfg)
	loadDB(db.ApplDB, appl)
	time.Sleep(1 * time.Second)
	defer func() {
		unloadDB(db.ApplDB, appl)
		unloadDB(db.ConfigDB, cfg)
		cleanupVlanCatalog(t, getIDs...)
	}()

	t.Run("GET VLAN 100 config ACTIVE from CONFIG_DB", processGetRequest(vlanCatalogPath(100, "/config"), nil, `{"openconfig-vlan:config":{"name":"Vlan100","status":"ACTIVE","vlan-id":100}}`, false))
	t.Run("GET VLAN 100 state SUSPENDED from APPL_DB", processGetRequest(vlanCatalogPath(100, "/state"), nil, `{"openconfig-vlan:state":{"name":"Vlan100","status":"SUSPENDED","vlan-id":100}}`, false))
}

func Test_openconfig_vlan_catalog_subscribe_exists(t *testing.T) {
	dbMap, virtual, _, err := transformer.SubscribeOcVlans("/openconfig-vlan:vlans/vlan[vlan-id=100]/state", true)
	if err != nil {
		t.Fatalf("SubscribeOcVlans failed: %v", err)
	}
	if !virtual {
		t.Fatalf("TRANSLATE_EXISTS should be virtual-table")
	}
	if len(dbMap) != 0 {
		t.Fatalf("TRANSLATE_EXISTS should not emit DB maps, got %v", dbMap)
	}
}

func Test_openconfig_vlan_catalog_subscribe_parent_check(t *testing.T) {
	dbMap, virtual, _, err := transformer.SubscribeOcVlansParentCheck("/openconfig-vlan:vlans/vlan[vlan-id=100]")
	if err != nil {
		t.Fatalf("SubscribeOcVlansParentCheck failed: %v", err)
	}
	if !virtual {
		t.Fatalf("parent-existence check should be virtual-table so UPDATE can create a missing VLAN")
	}
	if len(dbMap) != 0 {
		t.Fatalf("parent-existence check should not emit DB maps, got %v", dbMap)
	}
}

func Test_openconfig_vlan_catalog_subscribe_paths(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		wantCfgKey     string
		wantApplKey    string
		wantMemberKey  string
		wantMemberComp bool
	}{
		{
			name:           "wildcard parent",
			uri:            "/openconfig-vlan:vlans",
			wantCfgKey:     "*",
			wantApplKey:    "*",
			wantMemberKey:  "*",
			wantMemberComp: true,
		},
		{
			name:           "keyed parent",
			uri:            "/openconfig-vlan:vlans/vlan[vlan-id=100]",
			wantCfgKey:     "Vlan100",
			wantApplKey:    "Vlan100",
			wantMemberKey:  "Vlan100:*",
			wantMemberComp: true,
		},
		{
			name:       "wildcard config",
			uri:        "/openconfig-vlan:vlans/vlan/config",
			wantCfgKey: "*",
		},
		{
			name:       "keyed config",
			uri:        "/openconfig-vlan:vlans/vlan[vlan-id=100]/config",
			wantCfgKey: "Vlan100",
		},
		{
			name:       "keyed config status",
			uri:        "/openconfig-vlan:vlans/vlan[vlan-id=100]/config/status",
			wantCfgKey: "Vlan100",
		},
		{
			name:        "wildcard state",
			uri:         "/openconfig-vlan:vlans/vlan/state",
			wantApplKey: "*",
		},
		{
			name:        "keyed state",
			uri:         "/openconfig-vlan:vlans/vlan[vlan-id=100]/state",
			wantApplKey: "Vlan100",
		},
		{
			name:        "keyed state status",
			uri:         "/openconfig-vlan:vlans/vlan[vlan-id=200]/state/status",
			wantApplKey: "Vlan200",
		},
		{
			name:           "wildcard members",
			uri:            "/openconfig-vlan:vlans/vlan/members",
			wantMemberKey:  "*",
			wantMemberComp: true,
		},
		{
			name:           "keyed members",
			uri:            "/openconfig-vlan:vlans/vlan[vlan-id=100]/members",
			wantMemberKey:  "Vlan100:*",
			wantMemberComp: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dbMap, virtual, onChange, err := transformer.SubscribeOcVlans(tc.uri, false)
			if err != nil {
				t.Fatalf("SubscribeOcVlans failed: %v", err)
			}
			if virtual {
				t.Fatalf("isVirtualTbl=true, want false so translib keeps Redis watches (map=%v)", dbMap)
			}
			if !onChange {
				t.Fatalf("onChange=false, want OnchangeEnable+OnChange")
			}

			if tc.wantCfgKey != "" {
				if _, ok := dbMap[db.ConfigDB][transformer.VLAN_TN][tc.wantCfgKey]; !ok {
					t.Fatalf("missing CONFIG_DB VLAN[%s]: %v", tc.wantCfgKey, dbMap)
				}
			} else if cfg, ok := dbMap[db.ConfigDB][transformer.VLAN_TN]; ok {
				t.Fatalf("unexpected CONFIG_DB VLAN watch: %v", cfg)
			}

			if tc.wantApplKey != "" {
				if _, ok := dbMap[db.ApplDB][transformer.VLAN_APP_TBL][tc.wantApplKey]; !ok {
					t.Fatalf("missing APPL_DB VLAN_TABLE[%s]: %v", tc.wantApplKey, dbMap)
				}
			} else if app, ok := dbMap[db.ApplDB][transformer.VLAN_APP_TBL]; ok {
				t.Fatalf("unexpected APPL_DB VLAN_TABLE watch: %v", app)
			}

			if tc.wantMemberKey != "" {
				fields, ok := dbMap[db.ApplDB][transformer.VLAN_MEMBER_APP_TBL][tc.wantMemberKey]
				if !ok {
					t.Fatalf("missing APPL_DB VLAN_MEMBER_TABLE[%s]: %v", tc.wantMemberKey, dbMap)
				}
				if tc.wantMemberComp && fields[transformer.KEY_COMP_CNT] != "2" {
					t.Fatalf("KEY_COMP_CNT=%q, want 2 for members key %s", fields[transformer.KEY_COMP_CNT], tc.wantMemberKey)
				}
			} else if mem, ok := dbMap[db.ApplDB][transformer.VLAN_MEMBER_APP_TBL]; ok {
				t.Fatalf("unexpected APPL_DB VLAN_MEMBER_TABLE watch: %v", mem)
			}
		})
	}
}
