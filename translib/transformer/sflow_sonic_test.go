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
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
)

func Test_node_sonic_sflow(t *testing.T) {
	var url, url_body_json string

	//Add sflow node
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	url_body_json = "{ \"sonic-sflow:global\": { \"admin_state\": \"down\", \"polling_interval\": 0, \"agent_id\": \"Ethernet0\" }}"
	t.Run("Add sFlow collector", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	//Set admin state
	url = "/sonic-sflow:sonic-sflow/SFLOW/global/admin_state"
	url_body_json = "{ \"sonic-sflow:admin_state\": \"up\"}"
	t.Run("Enable sFlow", processSetRequest(url, url_body_json, "PATCH", false))
	time.Sleep(1 * time.Second)

	//Set polling interval
	url = "/sonic-sflow:sonic-sflow/SFLOW/global/polling_interval"
	url_body_json = "{ \"sonic-sflow:polling_interval\": 100}"
	t.Run("Set sFlow polling interval", processSetRequest(url, url_body_json, "PUT", false))
	time.Sleep(1 * time.Second)

	//Set AgentID
	url = "/sonic-sflow:sonic-sflow/SFLOW/global/agent_id"
	url_body_json = "{ \"sonic-sflow:agent_id\": \"Ethernet4\"}"
	t.Run("Set sFlow agent ID", processSetRequest(url, url_body_json, "PATCH", false))
	time.Sleep(1 * time.Second)

	// Verify global configurations
	url = "/sonic-sflow:sonic-sflow/SFLOW/global"
	url_body_json = "{\"sonic-sflow:global\":{\"admin_state\":\"up\",\"agent_id\":\"Ethernet4\",\"polling_interval\":100,\"sample_direction\":\"rx\"}}"
	t.Run("Verify sFlow global configurations", processGetRequest(url, nil, url_body_json, false))

	//Delete sflow global configurations
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	t.Run("Test delete on sflow node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	//Verify deleted sflow global configuration
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	url_body_json = "{}"
	t.Run("Verify delete on sflow node", processGetRequest(url, nil, url_body_json, false))
}

func Test_node_sonic_sflow_collector(t *testing.T) {
	var url, url_body_json string

	//Add sFlow collector
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR"
	url_body_json = "{ \"sonic-sflow:SFLOW_COLLECTOR_LIST\": [ { \"name\": \"1.1.1.1_6343_default\", \"collector_ip\": \"1.1.1.1\", \"collector_port\": 6343, \"collector_vrf\": \"default\" } ]}"
	t.Run("Add sFlow collector", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	// Verify sFlow collector configurations
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR/SFLOW_COLLECTOR_LIST[name=1.1.1.1_6343_default]"
	t.Run("Verify sFlow collector", processGetRequest(url, nil, url_body_json, false))

	// Set collector ip
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR/SFLOW_COLLECTOR_LIST[name=1.1.1.1_6343_default]/collector_ip"
	url_body_json = "{ \"sonic-sflow:collector_ip\": \"1.1.1.2\"}"
	t.Run("Set sFlow collector ip", processSetRequest(url, url_body_json, "PATCH", false))
	time.Sleep(1 * time.Second)

	// Set collector port
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR/SFLOW_COLLECTOR_LIST[name=1.1.1.1_6343_default]/collector_port"
	url_body_json = "{ \"sonic-sflow:collector_port\": 1234}"
	t.Run("Set sFlow collector port", processSetRequest(url, url_body_json, "PATCH", false))
	time.Sleep(1 * time.Second)

	//Delete collector port
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR/SFLOW_COLLECTOR_LIST"
	t.Run("Test delete on Collector", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	//Verify collector port
	url = "/sonic-sflow:sonic-sflow/SFLOW_COLLECTOR/SFLOW_COLLECTOR_LIST"
	url_body_json = "{}"
	t.Run("Verify delete on sFlow collector", processGetRequest(url, nil, url_body_json, false))
}

func Test_node_sonic_sflow_interface(t *testing.T) {
	var url, url_body_json string

	//Sflow needs to be enabled to configure for the interface
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	url_body_json = "{ \"sonic-sflow:global\": { \"admin_state\": \"up\", \"polling_interval\": 0, \"agent_id\": \"Ethernet0\" }}"
	t.Run("Enable sFlow", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	//Configure for the sflow interface
	url = "/sonic-sflow:sonic-sflow/SFLOW_SESSION"
	url_body_json = "{ \"sonic-sflow:SFLOW_SESSION_LIST\": [ { \"port\": \"Ethernet0\", \"admin_state\": \"up\", \"sample_rate\": 10000 } ]}"
	t.Run("Configure sFlow interface", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	//Changing the sampling-rate
	url = "/sonic-sflow:sonic-sflow/SFLOW_SESSION/SFLOW_SESSION_LIST[port=Ethernet0]/sample_rate"
	url_body_json = "{ \"sonic-sflow:sample_rate\": 20000}"
	t.Run("Configuring the sampling_rate in sflow interface", processSetRequest(url, url_body_json, "PATCH", false))
	time.Sleep(1 * time.Second)

	//Deleting the configured interface
	url = "/sonic-sflow:sonic-sflow/SFLOW_SESSION/SFLOW_SESSION_LIST[port=Ethernet0]"
	t.Run("Delete on configured sflow interface", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	//Verify the deleted sflow interface
	url_body_json = "{}"
	err_str := "Resource not found"
	expected_err := tlerr.NotFoundError{Format: err_str}
	t.Run("Verify delete on sFlow Interface", processGetRequest(url, nil, url_body_json, true, expected_err))

	//Delete sflow global configurations
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	t.Run("Test delete on sflow node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	//Verify deleted sflow global configuration
	url = "/sonic-sflow:sonic-sflow/SFLOW"
	url_body_json = "{}"
	t.Run("Verify delete on sFlow collector", processGetRequest(url, nil, url_body_json, false))
}
