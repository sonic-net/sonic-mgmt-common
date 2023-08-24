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

//go:build testapp
// +build testapp

package transformer_test

import (
	"errors"
        "testing"
        "time"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
        "github.com/Azure/sonic-mgmt-common/translib/db"
)



func Test_node_on_openconfig_sflow(t *testing.T) {
        var pre_req_map, expected_map, cleanuptbl map[string]interface{}
        var url, url_body_json string

        t.Log("\n\n+++++++++++++ Performing Set on sflow node  ++++++++++++")
        url = "/openconfig-sampling-sflow:sampling/sflow/config"
	url_body_json = "{ \"openconfig-sampling-sflow:enabled\": true, \"openconfig-sampling-sflow:polling-interval\": 100, \"openconfig-sampling-sflow:agent\": \"Ethernet0\"}"
	expected_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet0" , "polling_interval":"100"}}}
        cleanuptbl = map[string]interface{}{"SFLOW": map[string]interface{}{"global": ""}}
        loadDB(db.ConfigDB, pre_req_map)
        time.Sleep(1 * time.Second)
        t.Run("Test set on sflow node", processSetRequest(url, url_body_json, "POST", false, nil))
        time.Sleep(1 * time.Second)
        t.Run("Verify set on sflow node", verifyDbResult(rclient, "SFLOW|global", expected_map, false))
        time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(1 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Set on sflow node  ++++++++++++")

        t.Log("\n\n+++++++++++++ Performing Delete on sflow node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet4" , "polling_interval":"200"}}}
        loadDB(db.ConfigDB, pre_req_map)
        time.Sleep(1 * time.Second)
        url = "/openconfig-sampling-sflow:sampling/sflow/config"
	expected_err := errors.New("DELETE not supported on attribute")
        t.Run("Test delete on sflow node", processDeleteRequest(url, true, expected_err ))
        time.Sleep(1 * time.Second)
        cleanuptbl = map[string]interface{}{"SFLOW": map[string]interface{}{"global": ""}}
        unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(1 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Delete on sflow node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Sflow node ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet8" , "polling_interval":"300"}}}
        loadDB(db.ConfigDB, pre_req_map)
        expected_get_json := "{\"openconfig-sampling-sflow:state\":{\"agent\":\"Ethernet8\",\"enabled\":true,\"polling-interval\":300}}"
        url = "/openconfig-sampling-sflow:sampling/sflow/state"
        t.Run("Test get on sflow node", processGetRequest(url, nil, expected_get_json, false))
        time.Sleep(1 * time.Second)
        unloadDB(db.ConfigDB, cleanuptbl)
        t.Log("\n\n+++++++++++++ Done Performing Get on Sflow node ++++++++++++")

        t.Log("\n\n+++++++++++++ Performing Put/Replace on sflow leaf node  ++++++++++++")
        url = "/openconfig-sampling-sflow:sampling/sflow/config"
	url_body_json = "{ \"openconfig-sampling-sflow:enabled\": true, \"openconfig-sampling-sflow:polling-interval\": 100, \"openconfig-sampling-sflow:agent\": \"Ethernet0\"}"
	pre_req_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet0" , "polling_interval":"100"}}}
	expected_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet0" , "polling_interval":"300"}}}
        cleanuptbl = map[string]interface{}{"SFLOW": map[string]interface{}{"global": ""}}
        loadDB(db.ConfigDB, pre_req_map)
        time.Sleep(1 * time.Second)
        url = "/openconfig-sampling-sflow:sampling/sflow/config/polling-interval"
	url_body_json = "{ \"openconfig-sampling-sflow:polling-interval\": 300}"
        t.Run("Update polling-interval on sflow node", processSetRequest(url, url_body_json, "PUT", false, nil))
        time.Sleep(1 * time.Second)
        t.Run("Verify polling-interval on sflow node", verifyDbResult(rclient, "SFLOW|global", expected_map, false))
        time.Sleep(1 * time.Second)
        unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(1 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Put/Replace on sflow leaf node  ++++++++++++")

        t.Log("\n\n+++++++++++++ Performing Patch on sflow leaf node  ++++++++++++")
        url = "/openconfig-sampling-sflow:sampling/sflow/config"
	url_body_json = "{ \"openconfig-sampling-sflow:enabled\": true, \"openconfig-sampling-sflow:polling-interval\": 100, \"openconfig-sampling-sflow:agent\": \"Ethernet0\"}"
	pre_req_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet0" , "polling_interval":"100"}}}
	expected_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet4" , "polling_interval":"100"}}}
        cleanuptbl = map[string]interface{}{"SFLOW": map[string]interface{}{"global": ""}}
        loadDB(db.ConfigDB, pre_req_map)
        time.Sleep(1 * time.Second)
        url = "/openconfig-sampling-sflow:sampling/sflow/config/agent"
	url_body_json = "{ \"openconfig-sampling-sflow:agent\": \"Ethernet4\"}"
        t.Run("Update Agent on sflow node", processSetRequest(url, url_body_json, "PATCH", false, nil))
        time.Sleep(1 * time.Second)
        t.Run("Verify Agent on sflow node", verifyDbResult(rclient, "SFLOW|global", expected_map, false))
        time.Sleep(1 * time.Second)
        unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(1 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Patch on sflow leaf node  ++++++++++++")
}

func Test_node_openconfig_sflow_collector(t *testing.T) {
        var pre_req_map, expected_map, cleanuptbl map[string]interface{}
        var url, url_body_json string

        t.Log("\n\n+++++++++++++ Performing Set on Collector ++++++++++++")
        url = "/openconfig-sampling-sflow:sampling/sflow/collectors"
	url_body_json = "{ \"openconfig-sampling-sflow:collector\": [ { \"address\": \"1.1.1.1\", \"port\": 6343, \"network-instance\": \"default\", \"config\": { \"address\": \"1.1.1.1\", \"port\": 6343, \"network-instance\": \"default\" } } ]}"
	expected_map = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"1.1.1.1_6343_default": map[string]interface{}{
		"collector_ip": "1.1.1.1",
		"collector_port": "6343",
		"collector_vrf": "default"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"1.1.1.1_6343_default": ""}}
        t.Run("Test set on collector node for sflow", processSetRequest(url, url_body_json, "POST", false, nil))
        time.Sleep(1 * time.Second)
        t.Run("Verify set on collector node for sflow", verifyDbResult(rclient, "SFLOW_COLLECTOR|1.1.1.1_6343_default", expected_map, false))
        unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(1 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Set on Collector ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Collector ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"2.2.2.2_4444_default": map[string]interface{}{
		"collector_ip": "2.2.2.2",
		"collector_port": "4444",
		"collector_vrf": "default"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"2.2.2.2_4444_default": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-sampling-sflow:sampling/sflow/collectors/collector[address=2.2.2.2][port=4444][network-instance=default]"
	t.Run("Test delete on collector for sflow", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	delete_expected := make(map[string]interface{})
        t.Run("Verify delete on collector node for sflow", verifyDbResult(rclient, "SFLOW_COLLECTOR|2.2.2.2_4444_default", delete_expected, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on Collector ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Collector ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"3.3.3.3_6666_default": map[string]interface{}{
		"collector_ip": "3.3.3.3",
		"collector_port": "6666",
		"collector_vrf": "default"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{ \"openconfig-sampling-sflow:collectors\":{\"collector\":[{\"address\":\"3.3.3.3\",\"config\":{\"address\":\"3.3.3.3\",\"network-instance\":\"default\",\"port\":6666},\"network-instance\":\"default\",\"port\":6666,\"state\":{\"address\":\"3.3.3.3\",\"network-instance\":\"default\",\"port\":6666}}]}}"
        url = "/openconfig-sampling-sflow:sampling/sflow/collectors"
	t.Run("Test get on collector node for sflow", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"SFLOW_COLLECTOR": map[string]interface{}{"3.3.3.3_6666_default": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on Collector ++++++++++++")
}


func Test_node_openconfig_sflow_interface(t *testing.T) {
	var pre_req_map, non_pre_req_map, expected_map, cleanuptbl , cleanuptbl_sflow map[string]interface{}
        var url, url_body_json string

	//Sflow needs to be enabled to configure for an sflow interface
        t.Log("\n\n+++++++++++++ Performing Set on Sflow Interface ++++++++++++")
        url = "/openconfig-sampling-sflow:sampling/sflow/interfaces"
	url_body_json = "{ \"openconfig-sampling-sflow:interface\": [ { \"name\": \"Ethernet0\", \"config\": { \"name\": \"Ethernet0\", \"enabled\": true, \"sampling-rate\": 10000 } } ]}"
	pre_req_map = map[string]interface{}{"SFLOW": map[string]interface{}{"global": map[string]interface{}{"admin_state": "up", "agent_id":"Ethernet0" , "polling_interval":"100"}}}
        loadDB(db.ConfigDB, pre_req_map)
	expected_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "10000"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": ""}}
        t.Run("Test set on sflow interface", processSetRequest(url, url_body_json, "POST", false, nil))
        time.Sleep(2 * time.Second)
        t.Run("Verify set on sflow interface", verifyDbResult(rclient, "SFLOW_SESSION|Ethernet0", expected_map, false))
        unloadDB(db.ConfigDB, cleanuptbl)
        time.Sleep(2 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Set on Sflow interface ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Sflow interface ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet4": map[string]interface{}{
		"admin_state": "down",
		"sample_rate": "20000"}}}
	non_pre_req_map = map[string]interface{}{"SFLOW_SESSION_TABLE": map[string]interface{}{"Ethernet4": map[string]interface{}{
		"admin_state": "down",
		"sample_rate": "20000"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet4": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	loadDB(db.ApplDB, non_pre_req_map)
	time.Sleep(2 * time.Second)
	url = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface[name=Ethernet4]"
	t.Run("Test delete on sflow interface", processDeleteRequest(url, false))
	time.Sleep(2 * time.Second)
	delete_expected := make(map[string]interface{})
        t.Run("Verify delete on sflow interface", verifyDbResult(rclient, "SFLOW_SESSION|Ethernet4", delete_expected, false))
	unloadDB(db.ConfigDB, cleanuptbl)
	unloadDB(db.ApplDB, non_pre_req_map)
	time.Sleep(2 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on Sflow Interface ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on Sflow Interface ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet8": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "30000"}}}
	non_pre_req_map = map[string]interface{}{"SFLOW_SESSION_TABLE": map[string]interface{}{"Ethernet8": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "30000"}}}
	loadDB(db.ConfigDB, pre_req_map)
	loadDB(db.ApplDB, non_pre_req_map)
	expected_get_json := "{\"openconfig-sampling-sflow:state\":{\"enabled\":true,\"name\":\"Ethernet8\",\"sampling-rate\":30000}}"
        url = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface[name=Ethernet8]/state"
	t.Run("Test get on sflow interface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(2 * time.Second)
	cleanuptbl = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet8": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	unloadDB(db.ApplDB, non_pre_req_map)
	t.Log("\n\n+++++++++++++ Done Performing Get on SFlow Interface ++++++++++++")

        t.Log("\n\n+++++++++++++ Performing Put/Replace on Sflow Interface ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "10000"}}}
	non_pre_req_map = map[string]interface{}{"SFLOW_SESSION_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "10000"}}}
	expected_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "down",
		"sample_rate": "10000"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": ""}}
        loadDB(db.ConfigDB, pre_req_map)
        loadDB(db.ApplDB, non_pre_req_map)
        url = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface[name=Ethernet0]/config/enabled"
	url_body_json = "{ \"openconfig-sampling-sflow:enabled\": false}"
        t.Run("Update admin-state for sflow interface", processSetRequest(url, url_body_json, "PUT", false, nil))
        time.Sleep(2 * time.Second)
        t.Run("Verify admin-state for sflow interface", verifyDbResult(rclient, "SFLOW_SESSION|Ethernet0", expected_map, false))
        unloadDB(db.ConfigDB, cleanuptbl)
        unloadDB(db.ApplDB, non_pre_req_map)
        time.Sleep(2 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Put/Replace on Sflow interface ++++++++++++")

        t.Log("\n\n+++++++++++++ Performing Patch on Sflow Interface ++++++++++++")
	pre_req_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "10000"}}}
	non_pre_req_map = map[string]interface{}{"SFLOW_SESSION_TABLE": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "10000"}}}
	expected_map = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": map[string]interface{}{
		"admin_state": "up",
		"sample_rate": "20000"}}}
        cleanuptbl = map[string]interface{}{"SFLOW_SESSION": map[string]interface{}{"Ethernet0": ""}}
        loadDB(db.ConfigDB, pre_req_map)
        loadDB(db.ApplDB, non_pre_req_map)
        cleanuptbl_sflow = map[string]interface{}{"SFLOW": map[string]interface{}{"global": ""}}
        url = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface[name=Ethernet0]/config/sampling-rate"
	url_body_json = "{ \"openconfig-sampling-sflow:sampling-rate\": 20000}"
        t.Run("Update sampling-rate on sflow interface", processSetRequest(url, url_body_json, "PATCH", false, nil))
        time.Sleep(2 * time.Second)
        t.Run("Verify sampling-rate on sflow interface", verifyDbResult(rclient, "SFLOW_SESSION|Ethernet0", expected_map, false))
        unloadDB(db.ConfigDB, cleanuptbl)
	unloadDB(db.ConfigDB, cleanuptbl_sflow)
        unloadDB(db.ApplDB, non_pre_req_map)
        time.Sleep(2 * time.Second)
        t.Log("\n\n+++++++++++++ Done Performing Patch on Sflow interface ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on Interfaces when sflow is disabled ++++++++++++")
	//Sflow global interfaces deletion is not supported
	url = "/openconfig-sampling-sflow:sampling/sflow/interfaces"
	err_str := "DELETE not supported"
	expected_err := tlerr.NotSupportedError{Format: err_str}
        t.Run("Test delete on sflow interfaces", processDeleteRequest(url, true, expected_err ))
	t.Log("\n\n+++++++++++++ Done Delete on Interfaces when sflow is disabled ++++++++++++")
}
