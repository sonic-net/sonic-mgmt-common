////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Dell, Inc.                                                 //
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

//go:build xfmrtest
// +build xfmrtest

package transformer_test

import (
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

type verifyDbResultData struct {
	test      string
	db_key    string
	db_result map[string]interface{}
}

func Test_node_exercising_subtree_xfmr_and_virtual_table(t *testing.T) {
	var pre_req_map, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing Set on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/interfaces"
	url_body_json = "{\"openconfig-test-xfmr:interface\": [ { \"id\": \"Eth_0\", \"config\": { \"id\": \"Eth_0\" }, \"ingress-test-sets\": { \"ingress-test-set\": [ { \"set-name\": \"TestSet_01\", \"type\": \"TEST_SET_IPV4\", \"config\": { \"set-name\": \"TestSet_01\", \"type\": \"TEST_SET_IPV4\" } } ] } } ]}"
	expected_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{"ports@": "Eth_0", "type": "IPV4"}}}
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": ""}}
	t.Run("Test set on node exercising subtree-xfmr and virtual table.", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on node exercising subtree-xfmr and virtual table.", verifyDbResult(rclient, "TEST_SET_TABLE|TestSet_01_TEST_SET_IPV4", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Set on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{
		"ports@": "Eth_0,Eth_1,Eth_3"},
		"TestSet_02_TEST_SET_IPV4": map[string]interface{}{
			"ports@": "Eth_1,Eth_4"}}}
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": "", "TestSet_02_TEST_SET_IPV4": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-test-xfmr:test-xfmr/interfaces/interface[id=Eth_1]"
	t.Run("Test delete on node exercising subtree-xfmr and virtual table.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{
		"ports@": "Eth_0,Eth_3"}}}
	t.Run("Verify delete on node exercising subtree-xfmr and virtual table (TestSet_01).", verifyDbResult(rclient, "TEST_SET_TABLE|TestSet_01_TEST_SET_IPV4", expected_map, false))
	expected_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_02_TEST_SET_IPV4": map[string]interface{}{
		"ports@": "Eth_4"}}}
	t.Run("Verify delete on node exercising subtree-xfmr and virtual table (TestSet_02).", verifyDbResult(rclient, "TEST_SET_TABLE|TestSet_02_TEST_SET_IPV4", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_03_TEST_SET_IPV6": map[string]interface{}{
		"ports@": "Eth_1"}}}

	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-test-xfmr:ingress-test-set\":[{\"config\":{\"set-name\":\"TestSet_03\",\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"},\"set-name\":\"TestSet_03\",\"state\":{\"set-name\":\"TestSet_03\",\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"},\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"}]}"
	url = "/openconfig-test-xfmr:test-xfmr/interfaces/interface[id=Eth_1]/ingress-test-sets/ingress-test-set[set-name=TestSet_03][type=TEST_SET_IPV6]"
	t.Run("Test get on node exercising subtree-xfmr and virtual table.", processGetRequest(url, nil, expected_get_json, false))
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_03_TEST_SET_IPV6": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")

}

func Test_node_exercising_tableName_key_and_field_xfmr(t *testing.T) {
	var pre_req_map, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing Set on Yang Node Exercising Table-Name, Key-Xfmr and Field-Xfmr ++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sets"
	url_body_json = "{ \"openconfig-test-xfmr:test-set\": [ { \"name\": \"TestSet_01\", \"type\": \"TEST_SET_IPV4\", \"config\": { \"name\": \"TestSet_01\", \"type\": \"TEST_SET_IPV4\", \"description\": \"TestSet_01Description\" } } ]}"
	expected_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{"type": "IPV4", "description": "Description : TestSet_01Description"}}}
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": ""}}
	t.Run("Test set on node exercising Table-Name, Key-Xfmr and Field-Xfmr", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on node exercising Table-Name, Key-Xfmr and Field-Xfmr", verifyDbResult(rclient, "TEST_SET_TABLE|TestSet_01_TEST_SET_IPV4", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Set on Yang Node Exercising Table-Name, Key-Xfmr and Field-Xfmr ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Yang Node Exercising Table-Name ,Key-Xfmr and Field-Xfmr ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{
		"type":        "IPV4",
		"description": "Description : TestSet_01_description",
		"ports@":      "Eth_0"}}}
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-test-xfmr:test-xfmr/test-sets/test-set[name=TestSet_01][type=TEST_SET_IPV4]/config/description"
	t.Run("Test delete on node exercising Table-Name, Key-Xfmr and Field-Xfmr", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_01_TEST_SET_IPV4": map[string]interface{}{
		"type":   "IPV4",
		"ports@": "Eth_0"}}}
	t.Run("Verify delete on node exercising Table-Name, Key-Xfmr and Field-Xfmr", verifyDbResult(rclient, "TEST_SET_TABLE|TestSet_01_TEST_SET_IPV4", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on Yang Node Exercising Subtree-Xfmr and Virtual Table ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Yang Node Exercising Table-Name, Key-Xfmr and Field-Xfmr ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_03_TEST_SET_IPV6": map[string]interface{}{
		"type":        "IPV6",
		"description": "Description : TestSet_03Description",
		"ports@":      "Eth_3"}}}

	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-test-xfmr:test-sets\":{\"test-set\":[{\"config\":{\"description\":\"TestSet_03Description\",\"name\":\"TestSet_03\",\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"},\"name\":\"TestSet_03\",\"state\":{\"description\":\"TestSet_03Description\",\"name\":\"TestSet_03\",\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"},\"type\":\"openconfig-test-xfmr:TEST_SET_IPV6\"}]}}"
	url = "/openconfig-test-xfmr:test-xfmr/test-sets"
	t.Run("Test get on node exercising Table-Name, Key-Xfmr and Field-Xfmr.", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"TEST_SET_TABLE": map[string]interface{}{"TestSet_03_TEST_SET_IPV6": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on Yang Node Exercising  Table-Name, Key-Xfmr and Field-Xfmr++++++++++++")
}

func Test_node_exercising_pre_xfmr_node(t *testing.T) {
	t.Log("\n\n+++++++++++++ Performing set on node exercising pre-xfmr ++++++++++++")
	err_str := "REPLACE not supported at this node."
	expected_err := tlerr.NotSupportedError{Format: err_str}
	//expected_err := tlerr.NotSupported("REPLACE not supported at this node.")
	url := "/openconfig-test-xfmr:test-xfmr/test-sets"
	url_body_json := "{ \"openconfig-test-xfmr:test-sets\": { \"test-set\": [ { \"name\": \"TestSet_03\", \"type\": \"TEST_SET_IPV4\", \"config\": { \"name\": \"TestSet_03\", \"type\": \"TEST_SET_IPV4\", \"description\": \"testSet_03 description\" } } ] }}"
	t.Run("Test set on node exercising pre-xfmr.", processSetRequest(url, url_body_json, "PUT", true, expected_err))
	t.Log("\n\n+++++++++++++ Done Performing set on node exercising pre-xfmr ++++++++++++")
}

func Test_node_with_child_tableXfmr_keyXfmr_fieldNameXfmrs_nonConfigDB_data(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": ""}}
	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"

	t.Log("++++++++++++++  Test_set_on_node_with_child_table_key_field_xfmrs  +++++++++++++")

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)

	// Payload
	post_payload := "{\"openconfig-test-xfmr:test-sensor-group\":[ { \"id\" : \"test_group_1\", \"config\": { \"id\": \"test_group_1\", \"group-colors\": [ \"red,blue,green\" ] }, \"test-sensor-types\": { \"test-sensor-type\": [ { \"type\": \"sensora_testA\", \"config\": { \"type\": \"sensora_testA\", \"exclude-filter\": \"filterB\" } } ] } } ]}"
	post_sensor_group_expected := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "10"}}}
	post_sensor_table_expected := map[string]interface{}{"TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"exclude_filter": "filter_filterB"}}}

	t.Run("Set on Node having child table and field transformer mapping", processSetRequest(url, post_payload, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on node with child table and field transformer", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", post_sensor_group_expected, false))
	t.Run("Verify set on node with child table and field transformer", verifyDbResult(rclient, "TEST_SENSOR_A_TABLE|test_group_1|sensor_type_a_testA", post_sensor_table_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_get_on_node_with_table_key_field_xfmrs_nonConfigDB_data  +++++++++++++")

	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "10"}}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"exclude_filter": "filter_filterB"}}}
	nonconfig_prereq := map[string]interface{}{"TEST_SENSOR_GROUP_COUNTERS": map[string]interface{}{"test_group_1": map[string]interface{}{"frame-in": "12345", "frame-out": "678910"}}}

	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	loadDB(db.CountersDB, nonconfig_prereq)

	get_expected := "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":10,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":10,\"counters\":{\"frame-in\":12345,\"frame-out\":678910},\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"test-sensor-types\":{\"test-sensor-type\":[{\"config\":{\"exclude-filter\":\"filterB\",\"type\":\"sensora_testA\"},\"state\":{\"exclude-filter\":\"filterB\",\"type\":\"sensora_testA\"},\"type\":\"sensora_testA\"}]}}]}"

	t.Run("Verify_get_on_node_with_child_table_key_field_xfmrs", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
	unloadDB(db.CountersDB, nonconfig_prereq)

	t.Log("++++++++++++++  Test_delete_on_node_with_child_table_key_field_xfmrs  +++++++++++++")

	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"

	// Setup - Prerequisite - None
	unloadDB(db.ConfigDB, cleanuptbl)
	loadDB(db.ConfigDB, prereq)

	delete_expected := make(map[string]interface{})

	t.Run("Delete on node with child table, key and field xfmrs", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on node with child table, key and field xfmrs", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", delete_expected, false))
	t.Run("Verify delete on node with child table, key and field xfmrs", verifyDbResult(rclient, "TEST_SENSOR_A_TABLE|test_group_1|sensor_type_a_testA", delete_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_node_exercising_tableXfmr_virtual_table_and_validate_handler(t *testing.T) {
	/* verify if get like traversal happens correctly when deleting a node having table-xfmr, virtual-table and validate handler annotation in child yang hierachy */
	t.Log("+++++++++++++++++ Test delete in yang hierachy involing table-xfmr, virtual-table and validate handler annotations +++++++++++++++")
	prereq_ni_instance := map[string]interface{}{"TEST_VRF": map[string]interface{}{"default": map[string]interface{}{"enabled": "true"},
		"Vrf_01": map[string]interface{}{"enabled": "false"}, "Vrf_02": map[string]interface{}{"enabled": "true"}}}
	prereq_bgp := map[string]interface{}{"TEST_BGP_NETWORK_CFG": map[string]interface{}{"Vrf_01|22": map[string]interface{}{"backdoor": "true", "policy-name": "abcd"},
		"Vrf_01|33": map[string]interface{}{"policy-name": "fgh"}, "Vrf_02|55": map[string]interface{}{"backdoor": "false"}}}
	prereq_ospfv2_router := map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true", "write-multiplier": "2"},
		"Vrf_02": map[string]interface{}{"enabled": "true"}}}
	prereq_ospfv2_router_distribution := map[string]interface{}{"TEST_OSPFV2_ROUTER_DISTRIBUTION": map[string]interface{}{"Vrf_01|66": map[string]interface{}{"priority": "6"},
		"Vrf_01|98": map[string]interface{}{"table-id": "4"}, "Vrf_02|81": map[string]interface{}{"priority": "9", "table-id": "67"}}}
	cleanuptbl := map[string]interface{}{"TEST_VRF": map[string]interface{}{"default": "", "Vrf_01": "", "Vrf_02": ""},
		"TEST_BGP_NETWORK_CFG":            map[string]interface{}{"Vrf_01|22": "", "Vrf_01|33": "", "Vrf_02|55": ""},
		"TEST_OSPFV2_ROUTER":              map[string]interface{}{"Vrf_01": "", "Vrf_02": ""},
		"TEST_OSPFV2_ROUTER_DISTRIBUTION": map[string]interface{}{"Vrf_01|66": "", "Vrf_01|98": "", "Vrf_02|81": ""}}
	//Setup
	loadDB(db.ConfigDB, prereq_ni_instance)
	loadDB(db.ConfigDB, prereq_bgp)
	loadDB(db.ConfigDB, prereq_ospfv2_router)
	loadDB(db.ConfigDB, prereq_ospfv2_router_distribution)
	url := "/openconfig-test-xfmr:test-xfmr/test-ni-instances/test-ni-instance[ni-name=vrf-01]/test-protocols"
	expected_ni_instance_vrf_01 := map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "false"}}}
	expected_ni_instance_default := map[string]interface{}{"TEST_VRF": map[string]interface{}{"default": map[string]interface{}{"enabled": "true"}}}
	expected_ni_instance_vrf_02 := map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_02": map[string]interface{}{"enabled": "true"}}}
	expected_bgp := map[string]interface{}{"TEST_BGP_NETWORK_CFG": map[string]interface{}{"Vrf_02|55": map[string]interface{}{"backdoor": "false"}}}
	expected_ospfv2_router := map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_02": map[string]interface{}{"enabled": "true"}}}
	expected_ospfv2_router_distribution := map[string]interface{}{"TEST_OSPFV2_ROUTER_DISTRIBUTION": map[string]interface{}{"Vrf_02|81": map[string]interface{}{"priority": "9", "table-id": "67"}}}
	empty_expected := make(map[string]interface{})
	t.Run("Delete in yang hierachy involing table-xfmr, virtual-table and validate handler annotations.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	verifyDbResultList := [11]verifyDbResultData{
		{test: "Verify ni-instance in request URL(vrf-01) is not deleted from db since URL points to child node.", db_key: "TEST_VRF|Vrf_01", db_result: expected_ni_instance_vrf_01},
		{test: "Verify default ni-instance not assocaited with ni-instance in request URL is retained in db.", db_key: "TEST_VRF|default", db_result: expected_ni_instance_default},
		{test: "Verify Vrf_02 ni-instance not assocaited with ni-instance in request URL is retained in db.", db_key: "TEST_VRF|Vrf_02", db_result: expected_ni_instance_vrf_02},
		{test: "Verify delete of bgp instance(vrf-01|22) assocaited with ni-instance in request URL is deleted from db.", db_key: "TEST_BGP_NETWORK_CFG|Vrf_01|22", db_result: empty_expected},
		{test: "Verify delete of bgp instance(vrf-01|33) assocaited with ni-instance in request URL is deleted from db.", db_key: "TEST_BGP_NETWORK_CFG|Vrf_01|33", db_result: empty_expected},
		{test: "Verify bgp instance(vrf-02|55) not assocaited with ni-instance in request URL is retained in db.", db_key: "TEST_BGP_NETWORK_CFG|Vrf_02|55", db_result: expected_bgp},
		{test: "Verify ospfv2-global/router instance(vrf-01) assocaited with ni-instance in request URL is deleted from db.", db_key: "TEST_OSPFV2_ROUTER|Vrf_01", db_result: empty_expected},
		{test: "Verify ospfv2-global/router instance(vrf-02) not assocaited with ni-instance in request URL is retained in db.", db_key: "TEST_OSPFV2_ROUTER|Vrf_02",
			db_result: expected_ospfv2_router},
		{test: "Verify ospfv2-router-distribution instance(vrf-01|66) assocaited with ni-instance in request URL is deleted from db.", db_key: "TEST_OSPFV2_ROUTER_DISTRIBUTION|Vrf_01|66",
			db_result: empty_expected},
		{test: "Verify ospfv2-router-distribution instance(vrf-01|98) assocaited with ni-instance in request URL is deleted from db.", db_key: "TEST_OSPFV2_ROUTER_DISTRIBUTION|Vrf_01|98",
			db_result: empty_expected},
		{test: "Verify ospfv2-router-distribution instance(vrf-02|81) not assocaited with ni-instance in request URL is retained in db.", db_key: "TEST_OSPFV2_ROUTER_DISTRIBUTION|Vrf_02|81",
			db_result: expected_ospfv2_router_distribution},
	}
	for _, data := range verifyDbResultList {
		t.Run(data.test, verifyDbResult(rclient, data.db_key, data.db_result, false))
	}
	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_node_exercising_non_table_owner_annotation(t *testing.T) {

	prereq := map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": map[string]interface{}{"metric": "32", "hwsku": "testhwsku", "deployment-id": "834"}}}
	cleanuptbl := map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": ""}}
	expected_map := map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": map[string]interface{}{"hwsku": "testhwsku", "deployment-id": "834"}}}
	url := "/openconfig-test-xfmr:test-xfmr/test-sets/system-zone-device-data"
	t.Log("++++++++++++++  Test_delete_on_node_exercising_non_table_owner_annotation  +++++++++++++")
	prereq = map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": map[string]interface{}{"metric": "32", "hwsku": "testhwsku", "deployment-id": "834"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": ""}}
	expected_map = map[string]interface{}{"DEVICE_ZONE_METADATA": map[string]interface{}{"local-zonehost": map[string]interface{}{"hwsku": "testhwsku", "deployment-id": "834"}}}
	// Setup
	loadDB(db.ConfigDB, prereq)
	t.Run("Delete on node exercising non table owner annotation.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on node exercising non table owner annotation.", verifyDbResult(rclient, "DEVICE_ZONE_METADATA|local-zonehost", expected_map, false))
	// TearDown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_node_exercising_subset_of_fields_in_mapped_table(t *testing.T) {

	prereq_ni_instance := map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true"}}}
	prereq_ospfv2_router := map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true", "write-multiplier": "2",
		"initial-delay": "40", "max-delay": "100"}}}
	cleanuptbl := map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_01": ""},
		"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": ""}}
	expected_ospfv2_router := map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true", "write-multiplier": "2",
		"initial-delay": "50"}}}
	url := "/openconfig-test-xfmr:test-xfmr/test-ni-instances/ni-instance[ni-name=vrf-01]/test-protocols/test-protocol[name=ospfv2]/ospfv2/" +
		"global/timers/config"

	t.Log("++++++++++++++  Test_delete_on_node_exercising_subset_of_fields_in_mapped_table  +++++++++++++")
	prereq_ni_instance = map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true"}}}
	prereq_ospfv2_router = map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true", "write-multiplier": "2",
		"initial-delay": "50"}}}
	cleanuptbl = map[string]interface{}{"TEST_VRF": map[string]interface{}{"Vrf_01": ""},
		"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": ""}}
	expected_ospfv2_router = map[string]interface{}{"TEST_OSPFV2_ROUTER": map[string]interface{}{"Vrf_01": map[string]interface{}{"enabled": "true", "write-multiplier": "2"}}}
	url = "/openconfig-test-xfmr:test-xfmr/test-ni-instances/test-ni-instance[ni-name=vrf-01]/test-protocols/test-protocol[name=ospfv2]/ospfv2/" +
		"global/timers/config"
	// Setup
	loadDB(db.ConfigDB, prereq_ni_instance)
	loadDB(db.ConfigDB, prereq_ospfv2_router)
	t.Run("Delete on node exercising subset of fields in mapped table.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on node exercising subset of fields in mapped table.", verifyDbResult(rclient, "TEST_OSPFV2_ROUTER|Vrf_01", expected_ospfv2_router, false))
	// TearDown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func test_node_exercising_db_parent_child_nonkey_leafref_relationship(t *testing.T) {
	/* oc yang hierarchy parent node's db table mapping has nonkey leafref db child relationship to a
	   node's db table mapping in the child hierachy & siblings yang nodes having similar relationship */
	prereq := map[string]interface{}{"TEST_NTP": map[string]interface{}{"global": map[string]interface{}{"trusted-key@": "68", "auth-enabled": "true"}},
		"TEST_NTP_AUTHENTICATION_KEY": map[string]interface{}{"68": map[string]interface{}{"key-type": "MD5", "key-value": "0x635352e91dd9ddf2ed9542db848d3b31"}}}
	cleanuptbl := map[string]interface{}{"TEST_NTP": map[string]interface{}{"global": ""}, "TEST_NTP_AUTHENTICATION_KEY": map[string]interface{}{"68": ""}}
	url := "/openconfig-test-xfmr:test-xfmr/test-ntp/test-ntp-keys"
	experr := tlerr.TranslibCVLFailure{Code: 1002, CVLErrorInfo: cvl.CVLErrorInfo{ErrCode: 1002, CVLErrDetails: "Config Validation Semantic Error",
		Msg: "Validation failed for Delete operation, given instance is in use", ConstraintErrMsg: "Validation failed for Delete operation, given instance is in use",
		TableName: "TEST_NTP_AUTHENTICATION_KEY", Keys: []string{"68"}, Field: "", Value: "", ErrAppTag: ""}}
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	t.Log("++++++++++++++  Test_delete_on_node_exercising_db_parent_child_nonkey_leafref_relationship  +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-ntp/test-ntp-keys"
	t.Run("Test delete of child yang node having parent-child non-key leafref relationship with parent yang node(error-case).", processDeleteRequest(url, true, experr))
	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_leaf_node(t *testing.T) {

	/* Test delete on leaf node that has a yang default */
	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}}

	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/config/color-hold-time"
	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "30"}}}

	t.Log("++++++++++++++  Test_delete_on_leaf_node_with_default_value_when_mapped_instance_exists_in_db  +++++++++++++")
	// Setup
	unloadDB(db.ConfigDB, cleanuptbl)
	loadDB(db.ConfigDB, prereq)
	del_sensor_group_expected := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "10"}}}
	t.Run("Delete on leaf node having default value when mapped instance exists in db.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on leaf node with default value resets to default", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", del_sensor_group_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_delete_on_leaf_node_with_default_value_when_mapped_instance_does_not_exist_db  +++++++++++++") //table mapping to container
	cleanuptbl = map[string]interface{}{"TRANSPORT_ZONE": map[string]interface{}{"transport-host": ""}}
	empty_expected := make(map[string]interface{})
	// Setup
	unloadDB(db.ConfigDB, cleanuptbl)
	url = "/openconfig-test-xfmr:test-xfmr/test-sets/transport-zone"
	t.Run("Delete on leaf node having default value when mapped instance doesn't exists in db and is annotated to a container.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on leaf node with default value doesn't reset to default creating an instance in DB when mapped instance is annotated to a container.",
		verifyDbResult(rclient, "TRANSPORT_ZONE|transport-host", empty_expected, false))

	t.Log("++++++++++++++  Test_delete_on_leaf_node_without_default_value  +++++++++++++")
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": ""}}
	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "testdescription"}}}
	expected_map := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode"}}}
	url = "/openconfig-test-xfmr:test-xfmr/global-sensor/description"
	//Setup
	loadDB(db.ConfigDB, prereq)
	t.Run("Delete on leaf node without default value", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on leaf node without default value", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", expected_map, false))
	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_post_xfmr(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}}
	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/config/color-hold-time"
	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "30"}}}

	t.Log("++++++++++++++  Test_post_xfmr  +++++++++++++")

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)
	loadDB(db.ConfigDB, prereq)

	patch_payload := "{ \"openconfig-test-xfmr:color-hold-time\": 50}"
	patch_expected := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "50"}}}
	post_expected := map[string]interface{}{"TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_post50": map[string]interface{}{"description_a": "Added instance in post xfmr"}}}

	t.Run("Test_post_xfmr", processSetRequest(url, patch_payload, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Test_post_xfmr", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", patch_expected, false))
	t.Run("Verify Test_post_xfmr", verifyDbResult(rclient, "TEST_SENSOR_A_TABLE|test_group_1|sensor_type_a_post50", post_expected, false))

	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_sonic_yang_node_operations(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_id_123": ""}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"sensor_id_123|sensor_type_a_123": ""}}
	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_id_123": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "25"}}}
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_A_TABLE"

	t.Log("++++++++++++++  Test_set_on_sonic_table_yang_node +++++++++++++")

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)
	loadDB(db.ConfigDB, prereq)

	// Payload
	post_payload := "{ \"sonic-test-xfmr:TEST_SENSOR_A_TABLE_LIST\": [ { \"id\": \"sensor_id_123\", \"type\": \"sensor_type_a_123\", \"exclude_filter\": \"filter_123\", \"description_a\": \"description test field for sensor A table\" } ]}"
	post_sensor_table_expected := map[string]interface{}{"TEST_SENSOR_A_TABLE": map[string]interface{}{"sensor_id_123|sensor_type_a_123": map[string]interface{}{"exclude_filter": "filter_123", "description_a": "description test field for sensor A table"}}}

	t.Run("Set on sonic table yang node", processSetRequest(url, post_payload, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on sonic table yang node", verifyDbResult(rclient, "TEST_SENSOR_A_TABLE|sensor_id_123|sensor_type_a_123", post_sensor_table_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_delete_on_sonic_module  +++++++++++++")

	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": ""}, "TEST_SENSOR_B_TABLE": map[string]interface{}{"test_group_1|sensor_type_b_testB": ""}, "TEST_SET_TABLE": map[string]interface{}{"test_set_1": ""}}

	url = "/sonic-test-xfmr:sonic-test-xfmr"
	prereq = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "30"}}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"exclude_filter": "filter_filterB"}}}, "TEST_SENSOR_B_TABLE": map[string]interface{}{"test_group_1|sensor_type_b_testB": map[string]interface{}{"exclude_filter": "filter_filterB"}}, "TEST_SET_TABLE": map[string]interface{}{"quert_TEST_SET_IPV4": map[string]interface{}{"type": "IPV4"}}}

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)
	loadDB(db.ConfigDB, prereq)

	delete_expected := make(map[string]interface{})

	t.Run("Delete on sonic module", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on sonic module table1", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", delete_expected, false))
	t.Run("Verify delete on sonic module table2", verifyDbResult(rclient, "TEST_SENSOR_A_TABLE|test_group_1|sensor_type_a_testA", delete_expected, false))
	t.Run("Verify delete on sonic module table3", verifyDbResult(rclient, "TEST_SENSOR_B_TABLE|test_group_1|sensor_type_b_testB", delete_expected, false))
	t.Run("Verify delete on sonic module table4", verifyDbResult(rclient, "TEST_SET_TABLE|quert_TEST_SET_IPV4", delete_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_get_on_sonic_table_with_key_xfmr  +++++++++++++")

	cleanuptbl = map[string]interface{}{"TEST_SENSOR_MODE_TABLE": map[string]interface{}{"mode:testsensor123:3543": ""}}
	prereq = map[string]interface{}{"TEST_SENSOR_MODE_TABLE": map[string]interface{}{"mode:testsensor123:3543": map[string]interface{}{"description": "Test sensor mode"}}}
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_MODE_TABLE"

	// Setup - Prerequisite
	loadDB(db.CountersDB, prereq)

	get_expected := "{\"sonic-test-xfmr:TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"Test sensor mode\",\"id\":3543,\"mode\":\"mode:testsensor123\"}]}}"
	t.Run("Get on Sonic table with key xfmr", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.CountersDB, cleanuptbl)
}

func Test_leaflist_node(t *testing.T) {
	var pre_req_map, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_01]/config/group-colors"
	url_body_json = "{ \"openconfig-test-xfmr:group-colors\": [ \"red\",\"black\" ]}"
	pre_req_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{
		"colors@": "red,green"}}}
	expected_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{"colors@": "red,green,black"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test patch on leaf-list.", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify patch on leaf-list.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_01", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Patch/Update on Yang leaf-list Node demonstrating leaf-list contents merge ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Put/Replace on Yang leaf-list Node demonstrating leaf-list contents swap ++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_01]/config/group-colors"
	url_body_json = "{ \"openconfig-test-xfmr:group-colors\": [ \"blue\",\"yellow\" ]}"
	pre_req_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{
		"colors@": "red,green"}}}
	expected_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{"colors@": "blue,yellow"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test replace on leaf-list.", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on leaf-list.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_01", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Put/Replace on Yang leaf-list Node demonstrating leaf-list contents swap ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Yang leaf-list Node - demonstrating specific leaf-list instance deletion & complete leaf-list deletion ++++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_01]/config/group-colors[group-colors=blue]"
	pre_req_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{
		"colors@": "red,blue,green", "color-hold-time": "30"}}}
	expected_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{"colors@": "red,green", "color-hold-time": "30"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test specific leaf-list instance deletion.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify specific leaf-list instance deletion.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_01", expected_map, false))
	time.Sleep(1 * time.Second)
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_01]/config/group-colors"
	expected_map = map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_01": map[string]interface{}{"color-hold-time": "30"}}}
	t.Run("Test complete leaf-list attribute deletion.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify complete leaf-list attribute deletion.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_01", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Performing Delete on Yang leaf-list Node instance demonstrating specific leaf-list instance deletion ++++++++++++")
}

func Test_node_exercising_singleton_container_and_keyname_mapping(t *testing.T) {
	var pre_req_map, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing Set on Yang Node Exercising Mapping to Sonic-Yang Singleton Container and Key-name  ++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/global-sensor"
	url_body_json = "{ \"openconfig-test-xfmr:mode\": \"testmode\", \"openconfig-test-xfmr:description\": \"testdescription\"}"
	expected_map = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "testdescription"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test set on yang node exercising mapping to sonic singleton conatiner and key-name.", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on yang node exercising mapping to sonic-yang singleton conatiner and key-name.", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Set on Yang Node Exercising Mapping to Sonic-Yang Singleton Container and Key-name  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Yang Node Exercising Mapping to Sonic-Yang Singleton Container and Key-name  ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{
		"mode":        "testmode",
		"description": "testdescription"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-test-xfmr:test-xfmr/global-sensor/description"
	t.Run("Test delete on node exercising mapping to sonic-yang singleton conatiner and key-name.", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{
		"mode": "testmode"}}}
	t.Run("Verify delete on node exercising mapping to sonic-yang singleton conatiner and key-name.", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", expected_map, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on Yang Node Exercising Mapping to Sonic-Yang Singleton Container and Key-name  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Yang Node Exercising  Mapping to Sonic-Yang Singleton Container and Key-name ++++++++++++")
	pre_req_map = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{
		"mode":        "testmode",
		"description": "testdescription"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-test-xfmr:global-sensor\": {\"description\": \"testdescription\",\"mode\": \"testmode\"}}"
	url = "/openconfig-test-xfmr:test-xfmr/global-sensor"
	t.Run("Test get on node exercising mapping to sonic-yang singleton conatiner and key-name.", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on Yang Node Exercising  mapping to sonic-yang singleton conatiner and key-name ++++++++++++")
}

func Test_singleton_sonic_yang_node_operations(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": "", "global_sensor_timer": "", "test_device|32": ""}}

	t.Log("++++++++++++++  Test_create_on_sonic_singleton_container_yang_node +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)
	// Payload
	post_payload := "{ \"sonic-test-xfmr:global_sensor\": { \"mode\": \"testmode\", \"description\": \"testdescp\" }}"
	post_sensor_global_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "testdescp", "reset_time": 5}}}

	t.Run("Create on singleton sonic table yang node", processSetRequest(url, post_payload, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Create on singleton sonic table yang node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", post_sensor_global_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_create_on_sonic_node_having_singleton_container_sibling_list +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)
	// Payload
	post_payload = "{ \"sonic-test-xfmr:global_sensor\": { \"mode\": \"testmode\", \"description\": \"testdescp\" }, \"sonic-test-xfmr:TEST_SENSOR_GLOBAL_LIST\": [{\"device_name\": \"test_device\", \"device_id\": 32,\"device_status\": \"ON\"}]}"
	post_sensor_global_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "testdescp", "reset_time": 5}}}
	post_sensor_device_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"test_device|32": map[string]interface{}{"device_status": "ON"}}}

	t.Run("Create on sonic table yang node having singleton container and sibling list", processSetRequest(url, post_payload, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Create on sonic table yang node with singleton container and sibling list", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", post_sensor_device_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_patch_on_sonic_singleton_container_node +++++++++++++")

	prereq := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}, "global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}}}

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor"

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	// Payload
	patch_payload := "{ \"sonic-test-xfmr:global_sensor\": { \"mode\": \"testmode\", \"description\": \"test description\", \"reset_time\": 20 }}"
	patch_sensor_global_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description", "reset_time": 20}}}
	patch_sensor_timer_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}}}

	t.Run("Patch on singleton sonic container yang node", processSetRequest(url, patch_payload, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify patch on singleton sonic container yang node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", patch_sensor_global_expected, false))
	t.Run("Verify patch on singleton sonic container yang node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor_timer", patch_sensor_timer_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_create_and_modify_on_sonic_sibling_singleton_container_yang_node +++++++++++++")

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description"}}}

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	// Payload
	patch_payload = "{\"sonic-test-xfmr:TEST_SENSOR_GLOBAL\":{\"global_sensor\": {\"mode\": \"testmode\", \"description\": \"test description for testmode\", \"reset_time\": 25 }, \"sonic-test-xfmr:global_sensor_timer\": { \"timer_mode\": \"sample\", \"timer_description\": \"test sample timer mode\" }}}"
	patch_sensor_global_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}}}
	patch_sensor_timer_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 5}}}

	t.Run("Create and modify on sibling singleton containers sonic yang node", processSetRequest(url, patch_payload, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Create and modify on sibling singleton containers in sonic yang table node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", patch_sensor_global_expected, false))
	t.Run("Verify Create and modify on sibling singleton containers in sonic yang table node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor_timer", patch_sensor_timer_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_patch_on_sonic_yang_with_singleton_container_sibling_list +++++++++++++")

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}, "global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}, "test_device|32": map[string]interface{}{"device_status": "ON"}}}

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	// Payload
	patch_payload = "{\"sonic-test-xfmr:TEST_SENSOR_GLOBAL\":{ \"global_sensor\": { \"mode\": \"testmode\", \"description\": \"testdescp\" }, \"TEST_SENSOR_GLOBAL_LIST\": [{\"device_name\": \"test_device\",\"device_id\": 32, \"device_status\": \"OFF\"}]}}"

	patch_sensor_global_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "testdescp", "reset_time": 25}}}
	patch_sensor_timer_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}}}
	patch_sensor_device_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"test_device|32": map[string]interface{}{"device_status": "OFF"}}}

	t.Run("Patch on sonic node having singleton container and sibling list yang nodes", processSetRequest(url, patch_payload, "PATCH", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify patch on sonic yang node with sibling list and singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", patch_sensor_global_expected, false))
	t.Run("Verify patch on sonic yang node with sibling list and singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor_timer", patch_sensor_timer_expected, false))
	t.Run("Verify patch on sonic yang node with sibling list and singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", patch_sensor_device_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_replace_on_sonic_singleton_container_leaf +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor/mode"

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description"}}}
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	// Payload
	put_payload := "{ \"sonic-test-xfmr:mode\": \"test_mode_1\"}"
	put_sensor_global_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "test_mode_1", "description": "test description"}}}

	t.Run("Put on singleton sonic yang leaf node", processSetRequest(url, put_payload, "PUT", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify put on singleton sonic yang leaf node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", put_sensor_global_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_replace_on_sonic_sibling_singleton_container_yang_node +++++++++++++")

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description", "reset_time": 40}, "test_device|32": map[string]interface{}{"device_status": "ON"}}}

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	// Payload
	put_payload = "{\"sonic-test-xfmr:TEST_SENSOR_GLOBAL\":{\"global_sensor\": {\"mode\": \"testmode\", \"description\": \"test description for testmode\"}, \"sonic-test-xfmr:global_sensor_timer\": { \"timer_mode\": \"sample\", \"timer_description\": \"test sample timer mode\" },\"TEST_SENSOR_GLOBAL_LIST\": [{\"device_name\": \"test_device\",\"device_id\": 32, \"device_status\": \"OFF\"}]}}"
	put_sensor_global_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 5}}}
	put_sensor_timer_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 5}}}
	put_sensor_device_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"test_device|32": map[string]interface{}{"device_status": "OFF"}}}

	t.Run("Create and replace on sibling singleton containers sonic yang node", processSetRequest(url, put_payload, "PUT", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Create and replace on sibling singleton containers in sonic yang table node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", put_sensor_global_expected, false))
	t.Run("Verify Create and replace on sibling singleton containers in sonic yang table node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor_timer", put_sensor_timer_expected, false))
	t.Run("Verify replace on sibling singleton containers to list in sonic yang table node", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", put_sensor_device_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_replace_on_sonic_list_instance_sibling_to_singleton_container +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/TEST_SENSOR_GLOBAL_LIST[device_name=test_device][device_id=32]"

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description"}}}
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	// Payload
	put_payload = "{\"sonic-test-xfmr:TEST_SENSOR_GLOBAL_LIST\": [{\"device_name\": \"test_device\",\"device_id\": 32, \"device_status\": \"OFF\"}]}"
	put_sensor_device_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"test_device|32": map[string]interface{}{"device_status": "OFF"}}}

	t.Run("Put on sonic yang node list, sibling to singleton container", processSetRequest(url, put_payload, "PUT", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify put/create on  sonic yang node list, sibling to singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", put_sensor_device_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_delete_on_singleton_sonic_container  +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor"

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	delete_expected := make(map[string]interface{})

	t.Run("Delete on singleton sonic container", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on sonic singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", delete_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_delete_on_whole_sibling_list_to_singleton_sonic_container  +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/TEST_SENSOR_GLOBAL_LIST"
	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}, "global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}, "test_device|32": map[string]interface{}{"device_status": "OFF"}, "test_device|54": map[string]interface{}{"device_status": "ON"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": "", "test_device|32": "", "test_device|54": ""}}
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	delete_expected = make(map[string]interface{})
	delete_expected_global_sensor := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}}}

	t.Run("Delete on singleton sonic container", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on whole sibling list to singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", delete_expected, false))
	t.Run("Verify delete on whole sibling list to singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|54", delete_expected, false))
	t.Run("Verify delete on whole sibling list to singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", delete_expected_global_sensor, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_delete_on_table_with_mutiple_sibling_singleton_sonic_containers_and_sibling_list  +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}, "global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}, "test_device|32": map[string]interface{}{"device_status": "OFF"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": "", "global_sensor_timer": "", "test_device|32": ""}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	delete_expected = make(map[string]interface{})

	t.Run("Delete on table with mutiple sibling singleton sonic containers and sibling list", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on table with mutiple sonic singleton container and sibling list(global_sensor)", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", delete_expected, false))
	t.Run("Verify delete on table with mutiple sonic singleton container and sbling list(global_sensor_timer)", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor_timer", delete_expected, false))
	t.Run("Verify delete on table with mutiple sonic singleton containerand sibling list(test_device|32)", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|test_device|32", delete_expected, false))

	t.Log("++++++++++++++  Test_get_on_sonic_singleton_container  +++++++++++++")

	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "mode_test", "description": "test description for single container"}}}
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor"

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	get_expected := "{\"sonic-test-xfmr:global_sensor\": { \"mode\": \"mode_test\", \"description\": \"test description for single container\" }}"
	t.Run("Get on Sonic singleton container", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_get_on_table_with_mutiple_sibling_singleton_sonic_containers_and_sibling_list  +++++++++++++")

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	prereq = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "description": "test description for testmode", "reset_time": 25}, "global_sensor_timer": map[string]interface{}{"timer_mode": "sample", "timer_description": "test sample timer mode", "reset_time": 30}, "test_device|32": map[string]interface{}{"device_status": "OFF"}}}
	cleanuptbl = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": "", "global_sensor_timer": "", "test_device|32": ""}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	get_expected = "{\"sonic-test-xfmr:TEST_SENSOR_GLOBAL\":{ \"global_sensor\": { \"mode\": \"testmode\", \"description\": \"test description for testmode\", \"reset_time\":25 },\"global_sensor_timer\": { \"timer_mode\": \"sample\", \"timer_description\": \"test sample timer mode\", \"reset_time\":30}, \"TEST_SENSOR_GLOBAL_LIST\": [{\"device_name\": \"test_device\",\"device_id\": 32, \"device_status\": \"OFF\"}]}}"
	t.Run("Get on Sonic table with mutiple sonic singleton containers and sibling list", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

}

// Query parameter UT cases

func Test_Query_Params_OC_Yang_Get(t *testing.T) {

	var qp queryParamsUT
	qp.depth = 3

	cleanuptbl := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}, "TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": ""}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": ""}}
	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"colors@": "red,blue,green", "color-hold-time": "30"}}}
	prereq_cntr := map[string]interface{}{"TEST_SENSOR_GROUP_COUNTERS": map[string]interface{}{"test_group_1": map[string]interface{}{"frame-in": "3435", "frame-out": "3452"}}}

	// Setup - Prerequisite - None
	unloadDB(db.ConfigDB, cleanuptbl)
	unloadDB(db.CountersDB, prereq_cntr)
	loadDB(db.ConfigDB, prereq)
	loadDB(db.CountersDB, prereq_cntr)

	t.Log("++++++++++++++  Test_Query_Depth3_Container_Get  +++++++++++++")
	url := "/openconfig-test-xfmr:test-xfmr"
	get_expected := "{}"
	t.Run("Test_Query_Depth3_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth4_Container_Get  +++++++++++++")
	qp.depth = 4
	get_expected = "{\"openconfig-test-xfmr:test-xfmr\":{\"test-sensor-groups\":{\"test-sensor-group\":[{\"id\":\"test_group_1\"}]}}}"
	t.Run("Test_Query_Depth4_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth8_Container_Get  +++++++++++++")
	qp.depth = 8
	get_expected = "{\"openconfig-test-xfmr:test-xfmr\":{\"test-sensor-groups\":{\"test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"counters\":{\"frame-in\":3435,\"frame-out\":3452},\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}}}"
	t.Run("Test_Query_Depth8_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth0_Container_Get  +++++++++++++")
	qp.depth = 0
	t.Run("Test_Query_Depth0_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth3_List_Get  +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group"
	qp.depth = 3
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}"
	t.Run("Test_Query_Depth3_List_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth3_List_Instance_Get  +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"
	t.Run("Test_Query_Depth3_List_Instance_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth2_Leaf_Get  +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/config/color-hold-time"
	qp.depth = 2
	get_expected = "{\"openconfig-test-xfmr:color-hold-time\":30}"
	t.Run("Test_Query_Depth2_Leaf_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Content_All_Get +++++++++++++")
	// Reset Depth
	qp.depth = 0
	qp.content = "all"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"counters\":{\"frame-in\":3435,\"frame-out\":3452},\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}"
	t.Run("Test_Query_Content_All_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Content_Config_Get +++++++++++++")
	qp.content = "config"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\"}]}"
	t.Run("Test_Query_Content_Config_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Content_NonConfig_Get +++++++++++++")
	qp.content = "nonconfig"
	url = "/openconfig-test-xfmr:test-xfmr"
	get_expected = "{\"openconfig-test-xfmr:test-xfmr\":{\"test-sensor-groups\":{\"test-sensor-group\":[{\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"counters\":{\"frame-in\":3435,\"frame-out\":3452},\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}}}"
	t.Run("Test_Query_Content_NonConfig_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Content_Operational_Get +++++++++++++")
	qp.content = "operational"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-groups\":{\"test-sensor-group\":[{\"id\":\"test_group_1\",\"state\":{\"counters\":{\"frame-in\":3435,\"frame-out\":3452}}}]}}"
	t.Run("Test_Query_Content_Operational_Get", processGetRequest(url, &qp, get_expected, false))
	t.Log("++++++++++++++  Test_Query_Content_Mismatch_Leaf_Get +++++++++++++")
	qp.content = "config"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id==test_group_1]/state/counters/frame-in"
	get_expected = "{}"
	expected_err := tlerr.InvalidArgsError{Format: "Bad Request - requested content type doesn't match content type of terminal node uri."}
	t.Run("Test_Query_Content_Mismatch_Leaf_Get", processGetRequest(url, &qp, get_expected, true, expected_err))

	t.Log("++++++++++++++  Test_Query_Content_Mismatch_Container_Get +++++++++++++")
	qp.content = "nonconfig"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id==test_group_1]/config"
	get_expected = "{}"
	t.Run("Test_Query_Content_Mismatch_Container_Get1", processGetRequest(url, &qp, get_expected, false))
	qp.content = "config"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/state"
	t.Run("Test_Query_Content_Mismatch_Container_Get2", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth3_Content_Config_List_Get +++++++++++++")
	qp.depth = 3
	qp.content = "config"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\"}]}"
	t.Run("Test_Query_Depth3_Content_Config_List_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth4_Content_All_Container_Get +++++++++++++")
	qp.depth = 4
	qp.content = "all"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-groups\":{\"test-sensor-group\":[{\"config\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}}"
	t.Run("Test_Query_Depth4_Content_All_Container_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Depth3_Content_Nonconfig_ListInstance_Get +++++++++++++")
	qp.depth = 3
	qp.content = "nonconfig"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"group-colors\":[\"red\",\"blue\",\"green\"],\"id\":\"test_group_1\"}}]}"
	t.Run("Test_Query_Depth3_Content_NonConfig_ListInstance_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Fields_Leaf_Get +++++++++++++")
	// Reset Depth and Content
	qp.depth = 0
	qp.content = ""
	qp.fields = []string{"config/color-hold-time"}
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":30},\"id\":\"test_group_1\"}]}"
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"
	t.Run("Test_Query_Fields_Leaf_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Fields_MultiLeaf_Get +++++++++++++")
	qp.fields = []string{"state/color-hold-time", "state/counters/frame-in"}
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group"
	get_expected = "{\"openconfig-test-xfmr:test-sensor-group\":[{\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":30,\"counters\":{\"frame-in\":3435}}}]}"
	t.Run("Test_Query_Fields_MultiLeaf_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Fields_Container_Get +++++++++++++")
	qp.fields = []string{"counters"}
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/state"
	get_expected = "{\"openconfig-test-xfmr:state\":{\"counters\":{\"frame-in\":3435,\"frame-out\":3452}}}"
	t.Run("Test_Query_Fields_Container_Get", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Query_Fields_Error_IncorrectField_Get +++++++++++++")
	qp.fields = []string{"state/color-hold-times"}
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]"
	get_expected = "{}"
	expected_err = tlerr.InvalidArgsError{Format: "Invalid field name/path: state/color-hold-times"}
	t.Run("Test_Query_Fields_Error_IncorrectField_Get", processGetRequest(url, &qp, get_expected, true, expected_err))

	t.Log("++++++++++++++  Test_Query_Fields_Error_Leaf_Get +++++++++++++")
	qp.fields = []string{"color-hold-time"}
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/config/color-hold-time"
	get_expected = "{}"
	expected_err = tlerr.InvalidArgsError{Format: "Bad Request - fields query parameter specified on a terminal node uri."}
	t.Run("Test_Query_Fields_Error_Leaf_Get", processGetRequest(url, &qp, get_expected, true, expected_err))

	// Teardown
	unloadDB(db.ConfigDB, prereq)
	unloadDB(db.CountersDB, prereq_cntr)
}

/* sonic yang GET operation query-parameter tests */
func Test_sonic_yang_content_query_parameter_operations(t *testing.T) {
	var qp queryParamsUT
	prereq_config_db := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"description": "testdescription"}},
		"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"eth0": "10m"}}}
	prereq_nonconfig_db := map[string]interface{}{"TEST_SENSOR_MODE_TABLE": map[string]interface{}{"mode:testsensor123:3543": map[string]interface{}{"description": "Test sensor mode"}}}
	//Setup
	loadDB(db.ConfigDB, prereq_config_db)
	loadDB(db.CountersDB, prereq_nonconfig_db)
	t.Log("++++++++++++++  Test_content_all_query_parameter_on_sonic_yang  +++++++++++++")
	//covers singleton container and nested list
	url := "/sonic-test-xfmr:sonic-test-xfmr"
	qp.content = "all"
	get_expected := "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"}]}]},\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}},\"TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"Test sensor mode\",\"id\":3543,\"mode\":\"mode:testsensor123\"}]}}}"
	t.Run("Sonic yang query parameter content=all", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_content_config_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is top-level container
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"}]}]},\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}}}}"
	t.Run("Sonic yang query parameter content=config(target is top-level container)", processGetRequest(url, &qp, get_expected, false))

	//URI target is configurable nested whole list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH"
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"}]}"
	t.Run("Sonic yang query parameter content=config(target is whole nested child list)", processGetRequest(url, &qp, get_expected, false))

	//URI target is table level container
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH"
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\":[{\"TEST_CABLE_LENGTH\":[{\"length\":\"10m\",\"port\":\"eth0\"}],\"name\":\"testcable_01\"}]}}"
	t.Run("Sonic yang query parameter content=config(target is table level container)", processGetRequest(url, &qp, get_expected, false))

	//URI target is immediate child list of table
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST"
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"}]}]}"
	t.Run("Sonic yang query parameter content=config(target is immediate child list of table)", processGetRequest(url, &qp, get_expected, false))

	//URI target is nested list leaf
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]/length"
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:length\":\"10m\"}"
	t.Run("Sonic yang query parameter content=config(target is nested list leaf)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_content_nonconfig_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is top-level container
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.content = "nonconfig"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"Test sensor mode\",\"id\":3543,\"mode\":\"mode:testsensor123\"}]}}}"
	t.Run("Sonic yang query parameter content=nonconfig", processGetRequest(url, &qp, get_expected, false))

	//URI target is configutable nested whole list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH"
	qp.content = "nonconfig"
	get_expected = "{}"
	t.Run("Sonic yang query parameter content=nonconfig(target is whole nested child list)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_content_mismatch_error_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is nonconfigurable leaf
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_MODE_TABLE/TEST_SENSOR_MODE_TABLE_LIST[id=3543][mode=testsensor123]/description"
	qp.content = "config"
	get_expected = "{}"
	exp_err := tlerr.InvalidArgsError{Format: "Bad Request - requested content type doesn't match content type of terminal node uri."}
	t.Run("Sonic yang query parameter simple terminal node content mismatch error.", processGetRequest(url, &qp, get_expected, true, exp_err))

	//URI target is nested list leaf
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]/length"
	qp.content = "nonconfig"
	get_expected = "{}"
	t.Run("Sonic yang query parameter nested list leaf content mismatch error", processGetRequest(url, &qp, get_expected, true, exp_err))

	// Teardown
	unloadDB(db.ConfigDB, prereq_config_db)
	unloadDB(db.CountersDB, prereq_nonconfig_db)

}

func Test_sonic_yang_depth_query_parameter_operations(t *testing.T) {
	var qp queryParamsUT

	prereq := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"description": "testdescription"}},
		"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"eth0": "10m", "eth1": "11m"},
			"testcable_02": map[string]interface{}{"eth2": "22m"}}}

	t.Log("++++++++++++++  Test_depth_level_1_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is singleton container leaf
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor/description"
	qp.depth = 1
	//Setup
	loadDB(db.ConfigDB, prereq)
	get_expected := "{\"sonic-test-xfmr:description\":\"testdescription\"}"
	t.Run("Sonic yang query parameter depth=1(target is singleton container leaf)", processGetRequest(url, &qp, get_expected, false))

	//URI target is nested child  list-instance of table-list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]"
	qp.depth = 1
	get_expected = "{}"
	t.Run("Sonic yang query parameter depth=1(target is nested child list-instance of table-list)", processGetRequest(url, &qp, get_expected, false))

	//URI target is nested list leaf
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]/length"
	qp.depth = 1
	get_expected = "{\"sonic-test-xfmr:length\":\"10m\"}"
	t.Run("Sonic yang query parameter depth=1(target is nested list leaf)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_depth_level_2_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is singleton container
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor"
	qp.depth = 2
	get_expected = "{\"sonic-test-xfmr:global_sensor\":{\"description\":\"testdescription\"}}"
	t.Run("Sonic yang query parameter depth=2(target is singleton container)", processGetRequest(url, &qp, get_expected, false))

	//URI target is table-list having a nested child list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST"
	qp.depth = 2
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\"},{\"name\": \"testcable_02\"}]}"
	t.Run("Sonic yang query parameter depth=2(target is table-list having a nested child list)", processGetRequest(url, &qp, get_expected, false))

	//URI target is nested child whole-list of table-list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH"
	qp.depth = 2
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"},{\"length\": \"11m\",\"port\": \"eth1\"}]}"
	t.Run("Sonic yang query parameter depth=2(target is nested child list of table-list)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_depth_level_3_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is table-list having a nested child list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST"
	qp.depth = 3
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"},{\"length\": \"11m\",\"port\": \"eth1\"}]},{\"name\": \"testcable_02\", \"TEST_CABLE_LENGTH\": [{\"length\": \"22m\",\"port\": \"eth2\"}]}]}"
	t.Run("Sonic yang query parameter depth=3(target is table-list having a nested child list)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_depth_level_4_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is top-level container and covers singleton and nested list
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.depth = 4
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\": {\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}},\"TEST_CABLE_LENGTH\": {\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\"},{\"name\": \"testcable_02\"}]}}}"
	t.Run("Sonic yang query parameter depth=4(target is nested child list of table-list)", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_depth_level_5_query_parameter_on_sonic_yang  +++++++++++++")
	//URI target is top-level container and covers singleton and nested list
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.depth = 5
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\": {\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}},\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"},{\"length\": \"11m\",\"port\": \"eth1\"}]},{\"name\": \"testcable_02\", \"TEST_CABLE_LENGTH\": [{\"length\": \"22m\",\"port\": \"eth2\"}]}]}}}"
	t.Run("Sonic yang query parameter depth=5", processGetRequest(url, &qp, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, prereq)
}

func Test_sonic_yang_content_plus_depth_query_parameter_operations(t *testing.T) {
	var qp queryParamsUT

	prereq_config_db := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"description": "testdescription"}},
		"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"eth0": "10m"}}}
	prereq_nonconfig_db := map[string]interface{}{"TEST_SENSOR_MODE_TABLE": map[string]interface{}{"mode:testsensor123:3543": map[string]interface{}{"description": "Test sensor mode"}}}
	//Setup
	loadDB(db.ConfigDB, prereq_config_db)
	loadDB(db.CountersDB, prereq_nonconfig_db)

	t.Log("++++++++++++++  Test_content_all_depth_level_4_query_parameter_on_sonic_yang  +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr"
	qp.depth = 4
	qp.content = "all"
	get_expected := "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\"}]},\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}},\"TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"Test sensor mode\",\"id\":3543,\"mode\":\"mode:testsensor123\"}]}}}"
	t.Run("Sonic yang query parameter content=all depth=4", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_content_config_depth_level_5_query_parameter_on_sonic_yang  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.depth = 5
	qp.content = "config"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\": [{\"name\": \"testcable_01\",\"TEST_CABLE_LENGTH\": [{\"length\": \"10m\",\"port\": \"eth0\"}]}]},\"TEST_SENSOR_GLOBAL\":{\"global_sensor\":{\"description\":\"testdescription\"}}}}"
	t.Run("Sonic yang query parameter content=config depth=5", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_content_nonconfig_depth_level_4_query_parameter_on_sonic_yang  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.depth = 4
	qp.content = "nonconfig"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"Test sensor mode\",\"id\":3543,\"mode\":\"mode:testsensor123\"}]}}}"
	t.Run("Sonic yang query parameter content=nonconfig depth=4", processGetRequest(url, &qp, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, prereq_config_db)
	unloadDB(db.CountersDB, prereq_nonconfig_db)
}

func Test_sonic_yang_fields_query_parameter_operations(t *testing.T) {
	var qp queryParamsUT
	qp.fields = make([]string, 0)

	t.Log("++++++++++++++  Test_fields(single_field)_query_parameter_on_sonic_yang  +++++++++++++")
	prereq := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"description": "testdescription"}}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"exclude_filter": "filter_filterB", "description_a": "test group1 sensor type a descriptionXYZ"}, "test_group_2|sensor_type_a_testB": map[string]interface{}{"exclude_filter": "filter_filterA", "description_a": "test group2 sensor type a descriptionB"}}}
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_A_TABLE/TEST_SENSOR_A_TABLE_LIST"
	qp.fields = append(qp.fields, "exclude_filter")
	loadDB(db.ConfigDB, prereq)
	get_expected := "{\"sonic-test-xfmr:TEST_SENSOR_A_TABLE_LIST\":[{\"exclude_filter\": \"filter_filterB\",\"id\": \"test_group_1\",\"type\": \"sensor_type_a_testA\" },{\"exclude_filter\": \"filter_filterA\",\"id\": \"test_group_2\",\"type\": \"sensor_type_a_testB\"}]}"
	t.Run("Sonic yang query parameter fields(single field) ", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_fields(multiple_fields)_query_parameter_on_sonic_yang  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	qp.fields = make([]string, 0)
	qp.fields = append(qp.fields, "TEST_SENSOR_GLOBAL/global_sensor/description", "TEST_SENSOR_A_TABLE")
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\": {\"TEST_SENSOR_A_TABLE\": {\"TEST_SENSOR_A_TABLE_LIST\": [{\"description_a\": \"test group1 sensor type a descriptionXYZ\",\"exclude_filter\": \"filter_filterB\",\"id\": \"test_group_1\",\"type\": \"sensor_type_a_testA\"},{\"description_a\": \"test group2 sensor type a descriptionB\",\"exclude_filter\": \"filter_filterA\",\"id\": \"test_group_2\",\"type\": \"sensor_type_a_testB\"}]},\"TEST_SENSOR_GLOBAL\": {\"global_sensor\": {\"description\": \"testdescription\"}}}}"
	t.Run("Sonic yang query parameter fields(multiple field) ", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_invalid_fields_query_parameter_on_sonic_yang  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL"
	qp.fields = make([]string, 0)
	qp.fields = append(qp.fields, "global_sensor/desc")
	get_expected = "{}"
	exp_err := tlerr.InvalidArgsError{Format: "Invalid field name/path: global_sensor/desc"}
	t.Run("Sonic yang query parameter invalid fields", processGetRequest(url, &qp, get_expected, true, exp_err))

	t.Log("++++++++++++++  Test_invalid_fields_query_parameter_target_on_sonic_yang  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_A_TABLE/TEST_SENSOR_A_TABLE_LIST[id=test_group_1][type=sensor_type_a_testA]/exclude_filter"
	qp.fields = make([]string, 0)
	qp.fields = append(qp.fields, "exclude_filter")
	get_expected = "{}"
	exp_err = tlerr.InvalidArgsError{Format: "Bad Request - fields query parameter specified on a terminal node uri."}
	t.Run("Sonic yang invalid fields query parameter request target", processGetRequest(url, &qp, get_expected, true, exp_err))
	// Teardown
	unloadDB(db.ConfigDB, prereq)
}

func Test_OC_Sonic_OneOnOne_Composite_KeyMapping(t *testing.T) {

	parent_prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"NULL": "NULL"}}}
	prereq := map[string]interface{}{"TEST_SENSOR_COMPONENT_TABLE": map[string]interface{}{"FAN|TYPE1|14.31": map[string]interface{}{"description": "Test fan sensor type1 v14.31"}}}

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, prereq)
	loadDB(db.ConfigDB, parent_prereq)

	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/test-sensor-components"

	t.Log("++++++++++++++  Test_Set_OC_Sonic_OneOnOne_Composite_KeyMapping  +++++++++++++")

	url_body_json := "{\"openconfig-test-xfmr:test-sensor-component\":[{\"config\":{\"name\":\"FAN\",\"type\":\"TYPE1\",\"version\":\"14.31\",\"description\":\"Test fan sensor type1 v14.31\"},\"name\":\"FAN\",\"type\":\"TYPE1\",\"version\":\"14.31\"}]}"

	expected_map := map[string]interface{}{"TEST_SENSOR_COMPONENT_TABLE": map[string]interface{}{"FAN|TYPE1|14.31": map[string]interface{}{"description": "Test fan sensor type1 v14.31"}}}

	t.Run("SET on OC_Sonic_OneOnOne_Composite_KeyMapping", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test OC-Sonic one-one composite key mapping", verifyDbResult(rclient, "TEST_SENSOR_COMPONENT_TABLE|FAN|TYPE1|14.31", expected_map, false))

	// Teardown
	unloadDB(db.ConfigDB, prereq)
	unloadDB(db.ConfigDB, parent_prereq)

	t.Log("++++++++++++++  Test_Get_OC_Sonic_OneOnOne_Composite_KeyMapping  +++++++++++++")

	loadDB(db.ConfigDB, parent_prereq)
	loadDB(db.ConfigDB, prereq)

	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/test-sensor-components"

	get_expected := "{\"openconfig-test-xfmr:test-sensor-components\":{\"test-sensor-component\":[{\"config\":{\"description\":\"Test fan sensor type1 v14.31\",\"name\":\"FAN\",\"type\":\"TYPE1\",\"version\":\"14.31\"},\"name\":\"FAN\",\"state\":{\"description\":\"Test fan sensor type1 v14.31\",\"name\":\"FAN\",\"type\":\"TYPE1\",\"version\":\"14.31\"},\"type\":\"TYPE1\",\"version\":\"14.31\"}]}}"
	t.Run("GET on List_OC_Sonic_OneOnOne_Composite_KeyMapping", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, prereq)
	unloadDB(db.ConfigDB, parent_prereq)
}

/*
Test OC List having config container with leaves, that are referenced by list key-leafs and have no annotation.

	Also covers the list's state container that have leaves same as list keys
*/
func Test_NodeWithListHavingConfigLeafRefByKey_OC_Yang(t *testing.T) {

	t.Log("++++++++++++++  Test_set_on_OC_yang_node_with_list_having_config_leaf_referenced_by_list_key  +++++++++++++")
	pre_req := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"color-hold-time": "10"}}}
	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"
	// Payload
	post_payload := "{\"openconfig-test-xfmr:test-sensor-group\":[ { \"id\" : \"test_group_1\", \"config\": { \"id\": \"test_group_1\"} } ]}"
	post_sensor_group_expected := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"color-hold-time": "10"}}}
	t.Run("Set on OC-Yang node with list having config leaf referenced by list key.", processSetRequest(url, post_payload, "POST", false))
	time.Sleep(1 * time.Second)
	t.Run("Verify set on OC-Yang node with list having config leaf referenced by list key.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", post_sensor_group_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, pre_req)

	t.Log("++++++++++++++  Test get on OC yang node with list having config leaf referenced by list key and state leaf same as list key  +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group"
	// Setup - Prerequisite
	loadDB(db.ConfigDB, pre_req)
	// Payload
	get_expected := "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"color-hold-time\":10,\"id\":\"test_group_1\"},\"id\":\"test_group_1\",\"state\":{\"color-hold-time\":10,\"id\":\"test_group_1\"}}]}"
	t.Run("Verify get on OC yang node with list having config leaf referenced by list key and state leaf same list key", processGetRequest(url, nil, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, pre_req)

	t.Log("++++++++++++++ GET on OC YANG config container leaf that is referenced by immediate parent list's key and has no app annotations +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/state/id"
	// Setup - Prerequisite
	loadDB(db.ConfigDB, pre_req)
	// Payload
	get_expected = "{\"openconfig-test-xfmr:id\":\"test_group_1\"}"
	t.Run("Get on leaf in OC config container, with no app annotation, and is referenced by immediate parent list's key leaf", processGetRequest(url, nil, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, pre_req)

	t.Log("++++++++++++++ GET on OC YANG State container leaf that is same as immediate parent list's key and has no app annotations +++++++++++++")
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/state/id"
	// Setup - Prerequisite
	loadDB(db.ConfigDB, pre_req)
	// Payload
	get_expected = "{\"openconfig-test-xfmr:id\":\"test_group_1\"}"
	t.Run("Get on leaf in OC state container, with no app annotation, and is same as immediate parent list's key leaf", processGetRequest(url, nil, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, pre_req)
}

func Test_OC_Sonic_SingleKey_Mapping(t *testing.T) {

	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"group1\\|1": map[string]interface{}{"color-hold-time": "1354"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Get_OC_Sonic_SingleKey_Mapping  +++++++++++++")

	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups"

	get_expected := "{\"openconfig-test-xfmr:test-sensor-groups\":{\"test-sensor-group\":[{\"config\":{\"color-hold-time\":1354,\"id\":\"group1\\\\|1\"},\"id\":\"group1\\\\|1\",\"state\":{\"color-hold-time\":1354,\"id\":\"group1\\\\|1\"}}]}}"
	t.Run("GET on OC_Sonic_Single_KeyMapping_OC_request", processGetRequest(url, nil, get_expected, false))

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GROUP"

	get_expected = "{\"sonic-test-xfmr:TEST_SENSOR_GROUP\":{\"TEST_SENSOR_GROUP_LIST\":[{\"color-hold-time\":1354,\"id\":\"group1\\\\|1\"}]}}"
	t.Run("GET on OC_Sonic_Single_KeyMapping_SonicRequest", processGetRequest(url, nil, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Get_Sonic_SingleKey_Mapping  +++++++++++++")

	prereq = map[string]interface{}{"TEST_INTERFACE_MODE_TABLE": map[string]interface{}{"testname": map[string]interface{}{"description": "single key list entry"}, "testname|testmode": map[string]interface{}{"description": "double key list entry"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_INTERFACE_MODE_TABLE"

	get_expected = "{\"sonic-test-xfmr:TEST_INTERFACE_MODE_TABLE\":{\"TEST_INTERFACE_MODE_TABLE_IPADDR_LIST\":[{\"description\":\"double key list entry\",\"mode\":\"testmode\",\"name\":\"testname\"}],\"TEST_INTERFACE_MODE_TABLE_LIST\":[{\"description\":\"single key list entry\",\"name\":\"testname\"}]}}"
	t.Run("GET on Sonic_Single_KeyMapping", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, prereq)
}

func Test_sonic_yang_default_value_handling(t *testing.T) {
	t.Log("++++++++++++++  Test_set_on_sonic_yang_where_default_value_for_a_node_not_present_in_payload  +++++++++++++")
	pre_req := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": ""}, "TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": ""}}
	//setup
	unloadDB(db.ConfigDB, pre_req)
	url := "/sonic-test-xfmr:sonic-test-xfmr"
	post_payload := "{ \"sonic-test-xfmr:TEST_SENSOR_GROUP\": { \"TEST_SENSOR_GROUP_LIST\": [ { \"id\": \"test_group_1\" } ] }, \"sonic-test-xfmr:TEST_SENSOR_GLOBAL\": { \"global_sensor\": { \"mode\": \"testmode\", \"description\": \"testdescription\"} }}"
	sensor_global_expected := map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"description": "testdescription", "mode": "testmode", "reset_time": 5}}}
	sensor_group_expected := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"color-hold-time": 10}}}
	t.Run("Test set on sonic yang where default value for a node not present in payload.", processSetRequest(url, post_payload, "POST", false, nil))
	t.Run("Verify set on sonic yang where default value for a node not present in payload for list node", verifyDbResult(rclient, "TEST_SENSOR_GROUP|test_group_1", sensor_group_expected, false))
	t.Run("Verify set on sonic yang where default value for a node not present in payload for singleton container", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", sensor_global_expected, false))
	//Teardown
	unloadDB(db.ConfigDB, pre_req)

	t.Log("++++++++++++++  Test_delete_reseting_sonic_yang_leaf_node_to_default  +++++++++++++")
	pre_req = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "reset_time": 19}}}
	//setup
	loadDB(db.ConfigDB, pre_req)
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_SENSOR_GLOBAL/global_sensor/reset_time"
	sensor_global_expected = map[string]interface{}{"TEST_SENSOR_GLOBAL": map[string]interface{}{"global_sensor": map[string]interface{}{"mode": "testmode", "reset_time": 5}}}
	t.Run("Test delete reseting sonic yang leaf node to default", processDeleteRequest(url, false))
	t.Run("Verify delete reseting sonic yang leaf node to default", verifyDbResult(rclient, "TEST_SENSOR_GLOBAL|global_sensor", sensor_global_expected, false))
	//Teardown
	unloadDB(db.ConfigDB, pre_req)
}

// Test partial key at whole list level
func Test_WholeList_PartialKey(t *testing.T) {

	/* Delete And Get at a parent yang list node should rerieve/process only those child nodes which are relevant to the parent list key */
	parent_prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_1": map[string]interface{}{"NULL": "NULL"}, "sensor_group_2": map[string]interface{}{"NULL": "NULL"}}}
	prereq := map[string]interface{}{"TEST_SENSOR_ZONE_TABLE": map[string]interface{}{"sensor_group_1|zoneA": map[string]interface{}{"description": "sensor_group_1 zoneA instance"}, "sensor_group_2|zoneA": map[string]interface{}{"description": "sensor_group_2 zoneA instance"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, parent_prereq)
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  GET Test_PartialKey_Get +++++++++++++")
	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_1]"
	get_expected := "{\"openconfig-test-xfmr:test-sensor-group\":[{\"config\":{\"id\":\"sensor_group_1\"},\"id\":\"sensor_group_1\",\"state\":{\"id\":\"sensor_group_1\"},\"test-sensor-zones\":{\"test-sensor-zone\":[{\"config\":{\"description\":\"sensor_group_1 zoneA instance\",\"zone\":\"zoneA\"},\"state\":{\"zone\":\"zoneA\"},\"zone\":\"zoneA\"}]}}]}"

	t.Run("GET on Test_WholeList_PartialKey_Get", processGetRequest(url, nil, get_expected, false))

	t.Log("++++++++++++++  DELETE Test_WholeList_PartialKey_Delete +++++++++++++")

	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_1]/test-sensor-zones/test-sensor-zone"

	expected_del := map[string]interface{}{}
	expected_map := map[string]interface{}{"TEST_SENSOR_ZONE_TABLE": map[string]interface{}{"sensor_group_2|zoneA": map[string]interface{}{"description": "sensor_group_2 zoneA instance"}}}

	t.Run("DELETE on whole list with partial/ parent key, ", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("DELETE on whole list with partial/parent key - verify child table instance with parent key gets deleted", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_1|zoneA", expected_del, false))
	t.Run("DELETE on whole list with partial/parent key - verify child table instance not having parent key still exists", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_2|zoneA", expected_map, false))

	/* Verify if get like traversal is happenning for delete when a yang list node in child hierarchy has partial key of parent */
	t.Log("++++++++++++++  DELETE On Container with Child Node List With PartialKey of Parent List ++++++++++++++")
	// Setup - Prerequisite
	loadDB(db.ConfigDB, parent_prereq)
	loadDB(db.ConfigDB, prereq)
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_1]/test-sensor-zones"
	t.Run("Test_Delete_On_Container_with_Child_Node_List_With_PartialKey_of_Parent_List", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on container with child node list with partial key of parent - releavant child instances get deleted.", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_1|zoneA", expected_del, false))
	t.Run("Verify delete on container with child node list with partial key of parent - non-releavant child instances not deleted.", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_2|zoneA", expected_map, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++  DELETE On List Instance With Child Node List With PartialKey Of Parent List ++++++++++++++")
	// Setup - Prerequisite
	loadDB(db.ConfigDB, parent_prereq)
	loadDB(db.ConfigDB, prereq)
	url = "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=sensor_group_1]"
	expected_parent_map := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"sensor_group_2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test_Delete_On_List_Instance_with_Child_Node_List_With_PartialKey_of_Parent_List", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on list instance with child node list with partial key of parent - parent instance specified in request gets deleted.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_1", expected_del, false))
	t.Run("Verify delete on list instance with child node list with partial key of parent - other parent list instances stay intact.", verifyDbResult(rclient, "TEST_SENSOR_GROUP|sensor_group_2", expected_parent_map, false))
	t.Run("Verify delete on list instance with child node list with partial key of parent - releavant child instances get deleted.", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_1|zoneA", expected_del, false))
	t.Run("Verify delete on list instance with child node list with partial key of parent - non-releavant child instances not deleted.s", verifyDbResult(rclient, "TEST_SENSOR_ZONE_TABLE|sensor_group_2|zoneA", expected_map, false))

	t.Log("++++++++++++++  Done Test_GET_And_Delete_Involving_Node_With_PartialKey_Of_Parent +++++++++++++")

	// Teardown
	unloadDB(db.ConfigDB, parent_prereq)
	unloadDB(db.ConfigDB, prereq)
}

func Test_Validate_Handler_Get(t *testing.T) {
	prereq := map[string]interface{}{"TEST_SENSOR_GROUP": map[string]interface{}{"test_group_1": map[string]interface{}{"color-hold-time": "10"}}, "TEST_SENSOR_A_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA": map[string]interface{}{"exclude_filter": "filter_filterA1"}}, "TEST_SENSOR_B_TABLE": map[string]interface{}{"test_group_1|sensor_type_b_testB": map[string]interface{}{"exclude_filter": "filter_filterB"}}, "TEST_SENSOR_A_LIGHT_SENSOR_TABLE": map[string]interface{}{"test_group_1|sensor_type_a_testA|light_sensor_1": map[string]interface{}{"light-intensity-measure": 6}}}
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)
	t.Log("++++++++++++++  Test_Validate_Handler_Get +++++++++++++")
	url := "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group[id=test_group_1]/test-sensor-types/test-sensor-type"
	get_expected := "{\"openconfig-test-xfmr:test-sensor-type\":[{\"config\":{\"exclude-filter\":\"filterA1\",\"type\":\"sensora_testA\"},\"sensor-a-light-sensors\":{\"sensor-a-light-sensor\":[{\"config\":{\"light-intensity-measure\":6,\"tag\":\"lightsensor_1\"},\"state\":{\"light-intensity-measure\":6,\"tag\":\"lightsensor_1\"},\"tag\":\"lightsensor_1\"}]},\"state\":{\"exclude-filter\":\"filterA1\",\"type\":\"sensora_testA\"},\"type\":\"sensora_testA\"},{\"config\":{\"exclude-filter\":\"filterB\",\"type\":\"sensorb_testB\"},\"state\":{\"exclude-filter\":\"filterB\",\"type\":\"sensorb_testB\"},\"type\":\"sensorb_testB\"}]}"
	//light sensor child nodes are valid/filled only for sensora type of parent
	t.Run("Test Get on list with children having validate handler", processGetRequest(url, nil, get_expected, false))
	// Teardown
	unloadDB(db.ConfigDB, prereq)
}

//Sonic Nested list GET test cases

func Test_Sonic_NestedList_Get(t *testing.T) {

	prereq := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "23m", "eth1": "32m"}, "testCable_02": map[string]interface{}{"eth0": "20m", "eth1": "34m"}},
		"TEST_SENSOR_GROUP": map[string]interface{}{"testgroup": map[string]interface{}{"color-hold-time": 20, "colors@": "red,blue"}, "testGroup": map[string]interface{}{"color-hold-time": 30, "colors@": "red,blue"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at module level +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr"
	get_expected := "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\":[{\"TEST_CABLE_LENGTH\":[{\"length\":\"23m\",\"port\":\"eth0\"},{\"length\":\"32m\",\"port\":\"eth1\"}],\"name\":\"testCable\"},{\"TEST_CABLE_LENGTH\":[{\"length\":\"20m\",\"port\":\"eth0\"},{\"length\":\"34m\",\"port\":\"eth1\"}],\"name\":\"testCable_02\"}]},\"TEST_SENSOR_GROUP\":{\"TEST_SENSOR_GROUP_LIST\":[{\"color-hold-time\":30,\"colors\":[\"red\",\"blue\"],\"id\":\"testGroup\"},{\"color-hold-time\":20,\"colors\":[\"red\",\"blue\"],\"id\":\"testgroup\"}]}}}"
	t.Run("GET on Sonic_NestedList at module level", processGetRequest(url, nil, get_expected, false))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at outer list level  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH_LIST\":[{\"TEST_CABLE_LENGTH\":[{\"length\":\"23m\",\"port\":\"eth0\"},{\"length\":\"32m\",\"port\":\"eth1\"}],\"name\":\"testCable\"},{\"TEST_CABLE_LENGTH\":[{\"length\":\"20m\",\"port\":\"eth0\"},{\"length\":\"34m\",\"port\":\"eth1\"}],\"name\":\"testCable_02\"}]}"
	t.Run("GET on Sonic_NestedList at outer list level", processGetRequest(url, nil, get_expected, false))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at outer list level non existent instance  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable2]"
	expected_err := tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("GET on Sonic_NestedList at outer list level non existent instance", processGetRequest(url, nil, get_expected, true, expected_err))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at inner list level non existent instance  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth3]"
	expected_err = tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("GET on Sonic_NestedList at inner list level non existent instance ", processGetRequest(url, nil, get_expected, true, expected_err))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at inner list level instance  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth1]"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\": [{\"length\": \"32m\",\"port\": \"eth1\"}]}"
	t.Run("GET on Sonic_NestedList at inner list level instance", processGetRequest(url, nil, get_expected, false))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at inner whole list level  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable_02]/TEST_CABLE_LENGTH"
	get_expected = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\": [{\"length\":\"20m\",\"port\":\"eth0\"},{\"length\": \"34m\",\"port\": \"eth1\"}]}"
	t.Run("GET on Sonic_NestedList at inner whole list level", processGetRequest(url, nil, get_expected, false))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at inner list instance non key non existent +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth3]/length"
	expected_err = tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("GET on Sonic_NestedList at inner list instance non key non existent", processGetRequest(url, nil, get_expected, true, expected_err))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get at inner list instance non key existent case +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable_02]/TEST_CABLE_LENGTH[port=eth1]/length"
	get_expected = "{\"sonic-test-xfmr:length\": \"34m\"}"
	t.Run("GET on Sonic_NestedList at inner list instance non key existent case", processGetRequest(url, nil, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, prereq)

}

func Test_Sonic_NestedList_Get_Fields_QueryParams(t *testing.T) {

	prereq := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "23m", "eth1": "32m"}, "testCable_02": map[string]interface{}{"eth0": "20m", "eth1": "34m"}},
		"TEST_SENSOR_GROUP": map[string]interface{}{"testgroup": map[string]interface{}{"color-hold-time": 20, "colors@": "red,blue"}, "testGroup": map[string]interface{}{"color-hold-time": 30, "colors@": "red,blue"}}}
	var qp queryParamsUT
	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get fields Query parameter having inner list in query +++++++++++++")
	qp.fields = []string{"TEST_CABLE_LENGTH/port"}
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST"
	get_expected := "{}"
	exp_err := tlerr.NotSupportedError{Format: "Yang node type list not supported in fields query parameter(TEST_CABLE_LENGTH/port)."}
	t.Run("GET on Sonic_NestedList fields Query parameter having inner list in query", processGetRequest(url, &qp, get_expected, true, exp_err))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get fields Query parameter having outer list in query +++++++++++++")
	qp.fields = []string{"TEST_CABLE_LENGTH_LIST/TEST_CABLE_LENGTH/port"}
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH"
	get_expected = "{}"
	exp_err = tlerr.NotSupportedError{Format: "Yang node type list not supported in fields query parameter(TEST_CABLE_LENGTH_LIST/TEST_CABLE_LENGTH/port)."}
	t.Run("GET on Sonic_NestedList fields Query parameter having outer list in query", processGetRequest(url, &qp, get_expected, true, exp_err))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get fields Query parameter for only one table  +++++++++++++")
	qp.fields = []string{"TEST_CABLE_LENGTH"}
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\":[{\"TEST_CABLE_LENGTH\":[{\"length\":\"23m\",\"port\":\"eth0\"},{\"length\":\"32m\",\"port\":\"eth1\"}],\"name\":\"testCable\"},{\"TEST_CABLE_LENGTH\":[{\"length\":\"20m\",\"port\":\"eth0\"},{\"length\":\"34m\",\"port\":\"eth1\"}],\"name\":\"testCable_02\"}]}}}"
	t.Run("GET on Sonic_NestedList fields Query parameter for only one table", processGetRequest(url, &qp, get_expected, false))

	t.Log("++++++++++++++  Test_Sonic_NestedList_Get fields Query parameter for multiple tables +++++++++++++")
	prereq2 := map[string]interface{}{"TEST_SENSOR_MODE_TABLE": map[string]interface{}{"mode:testsensor:23": map[string]interface{}{"description": "testDesc"}, "testsensor:23": map[string]interface{}{"description": "testDesc"}}}
	loadDB(db.CountersDB, prereq2)
	time.Sleep(1 * time.Second)
	qp.fields = make([]string, 0)
	qp.fields = append(qp.fields, "TEST_CABLE_LENGTH", "TEST_SENSOR_MODE_TABLE")
	url = "/sonic-test-xfmr:sonic-test-xfmr"
	get_expected = "{\"sonic-test-xfmr:sonic-test-xfmr\":{\"TEST_CABLE_LENGTH\":{\"TEST_CABLE_LENGTH_LIST\":[{\"TEST_CABLE_LENGTH\":[{\"length\":\"23m\",\"port\":\"eth0\"},{\"length\":\"32m\",\"port\":\"eth1\"}],\"name\":\"testCable\"},{\"TEST_CABLE_LENGTH\":[{\"length\":\"20m\",\"port\":\"eth0\"},{\"length\":\"34m\",\"port\":\"eth1\"}],\"name\":\"testCable_02\"}]},\"TEST_SENSOR_MODE_TABLE\":{\"TEST_SENSOR_MODE_TABLE_LIST\":[{\"description\":\"testDesc\",\"id\":23,\"mode\":\"mode:testsensor\"},{\"description\":\"testDesc\",\"id\":23,\"mode\":\"testsensor\"}]}}}"
	t.Run("GET on Sonic_NestedList fields Query parameter for multiple tables", processGetRequest(url, &qp, get_expected, false))

	// Teardown
	unloadDB(db.ConfigDB, prereq)
	unloadDB(db.CountersDB, prereq2)

}

func Test_Sonic_NestedList_Create(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": ""}}

	// Setup - Prerequisite
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Create_instance_err  +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]"
	url_body_json := "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":[{\"port\":\"eth0\",\"length\":\"12m\"}]}"
	expected_err := tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("Test_Sonic_Nested_list_Create_instance_err", processSetRequest(url, url_body_json, "POST", true, expected_err))

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Create_instance  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH"
	url_body_json = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH_LIST\":[{\"name\":\"testCable\",\"TEST_CABLE_LENGTH\":[{\"port\":\"eth0\",\"length\":\"12m\"}]}]}"
	expected_map := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "12m"}}}
	t.Run("Test_Sonic_Nested_list_Create_instance", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test_Sonic_Nested_list_Create_instance Verify DB", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testCable", expected_map, false))

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Create_modify_instance  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]"
	url_body_json = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":[{\"port\":\"eth1\",\"length\":\"22m\"}]}"
	expected_map = map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "12m", "eth1": "22m"}}}
	t.Run("Test_Sonic_Nested_list_Create_modify_instance", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test_Sonic_Nested_list_Create_modify_instance Verify DB", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testCable", expected_map, false))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)

}

func Test_Sonic_NestedList_Update(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": ""}}
	prereq := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "12m", "eth1": "22m"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Update_inner_list_instance_existance_case  +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth0]"
	url_body_json := "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":[{\"port\":\"eth0\",\"length\":\"77m\"}]}"
	expected_map := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "77m", "eth1": "22m"}}}
	t.Run("Test_Sonic_Nested_list_Update_inner_list_instance_existance_case", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test_Sonic_Nested_list_Update_inner_list_instance_existance_case", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testCable", expected_map, false))

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Update_inner_list_instance_non_existance_case  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth3]"
	url_body_json = "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":[{\"port\":\"eth3\",\"length\":\"57m\"}]}"
	expected_err := tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("Test_Sonic_Nested_list_Update_inner_list_instance_non_existance_case", processSetRequest(url, url_body_json, "PATCH", true, expected_err))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_Sonic_NestedList_Replace(t *testing.T) {

	cleanuptbl := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": ""}}
	prereq := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "77m", "eth1": "22m"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Replace_inner_list_instance_non_existance_case  +++++++++++++")
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth3]"
	url_body_json := "{\"sonic-test-xfmr:TEST_CABLE_LENGTH\":[{\"port\":\"eth3\",\"length\":\"66m\"}]}"
	expected_map := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "77m", "eth1": "22m", "eth3": "66m"}}}
	t.Run("Test_Sonic_Nested_list_Replace_inner_list_instance_non_existance_case", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test_Sonic_Nested_list_Replace_inner_list_instance_non_existance_case Verify", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testCable", expected_map, false))

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Replace_inner_list_non_key_leaf_existance_case  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth1]/length"
	url_body_json = "{\"sonic-test-xfmr:length\":\"11m\"}"
	expected_map = map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testCable": map[string]interface{}{"eth0": "77m", "eth1": "11m", "eth3": "66m"}}}
	t.Run("Test_Sonic_Nested_list_Replace_inner_list_non_key_leaf_existance_case", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Test_Sonic_Nested_list_Replace_inner_list_non_key_leaf_existance_case Verify", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testCable", expected_map, false))

	t.Log("++++++++++++++  Test_Sonic_Nested_list_Replace_inner_list_non_key_leaf_non_existance_case  +++++++++++++")
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testCable]/TEST_CABLE_LENGTH[port=eth8]/length"
	url_body_json = "{\"sonic-test-xfmr:length\":\"88m\"}"
	expected_err := tlerr.NotFoundError{Format: "Resource not found"}
	t.Run("Test_Sonic_Nested_list_Replace_inner_list_non_key_leaf_non_existance_case", processSetRequest(url, url_body_json, "PUT", true, expected_err))

	// Teardown
	unloadDB(db.ConfigDB, cleanuptbl)
}

func Test_Sonic_NestedList_Delete(t *testing.T) {
	prereq := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"eth0": "10m", "eth1": "20m"},
		"testcable_02": map[string]interface{}{"eth0": "30m"}}}

	// Setup - Prerequisite
	loadDB(db.ConfigDB, prereq)

	//Delete targeted on nested non-key leaf when nested list instance doesn't exist in DB
	url := "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth4]/length"
	exp_err_res_not_found := tlerr.NotFoundError{Format: "Resource not found"}
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on non-key leaf when nested list instance doesn't exist in DB +++++++++++++")
	t.Run("DELETE on nested list non-key leaf when nested list instance doesn't exist in DB", processDeleteRequest(url, true, exp_err_res_not_found))

	//Delete targeted on nested non-key leaf when nested list instance exists in DB
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]/length"
	exp_err_not_supp := tlerr.NotSupportedError{Format: "DELETE not supported"}
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on non-key leaf when nested list instance exists in DB +++++++++++++")
	t.Run("DELETE on nested list non-key leaf when nested list instance exists in DB", processDeleteRequest(url, true, exp_err_not_supp))

	//Delete targeted on nested list instance and that instance doesn't exist in DB
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth4]"
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on nested list instance and that instance doesn't exist in DB +++++++++++++")
	t.Run("DELETE on nested list instance and that instance doesn't exist in DB", processDeleteRequest(url, true, exp_err_res_not_found))

	//Delete targeted on nested list instance and that instance exists in DB
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH[port=eth0]"
	expected_map := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"eth1": "20m"}}}
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on nested list instance and that instance exists in DB +++++++++++++")
	t.Run("DELETE on nested list instance and that instance exists in DB", processDeleteRequest(url, false))
	t.Run("DELETE on nested list instance and that instance exists in DB - verify other instance of nested list still exists and the one deleted doesn't exist", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testcable_01", expected_map, false))

	//Delete targeted on whole nested list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]/TEST_CABLE_LENGTH"
	expected_map = map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_01": map[string]interface{}{"NULL": "NULL"}}}
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on whole nested list +++++++++++++")
	t.Run("DELETE on whole nested list", processDeleteRequest(url, false))
	t.Run("DELETE on whole nested list - verify that all nested list instances are replaced by NULL/NULL preserving parent list instance", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testcable_01", expected_map, true))

	//Delete targetted at list instance that has nested child list
	url = "/sonic-test-xfmr:sonic-test-xfmr/TEST_CABLE_LENGTH/TEST_CABLE_LENGTH_LIST[name=testcable_01]"
	expected_map_testcable_01 := map[string]interface{}{}
	expected_map_testcable_02 := map[string]interface{}{"TEST_CABLE_LENGTH": map[string]interface{}{"testcable_02": map[string]interface{}{"eth0": "30m"}}}
	t.Log("++++++++++++++  Test_Sonic_NestedList_Delete on list instance that has nested child list +++++++++++++")
	t.Run("DELETE on list instance that has nested child list", processDeleteRequest(url, false))
	t.Run("DELETE on list instance that has nested child list - verify that only that parent list instance is deleted and other instnaces are instact(testcable_01 is deleted)", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testcable_01", expected_map_testcable_01, false))
	t.Run("DELETE on list instance that has nested child list - verify that only that parent list instance is deleted and other instnaces are instact(testcable_02 is instact)", verifyDbResult(rclient, "TEST_CABLE_LENGTH|testcable_02", expected_map_testcable_02, false))

	//Teardown
	unloadDB(db.ConfigDB, prereq)

}
