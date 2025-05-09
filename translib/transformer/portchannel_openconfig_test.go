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
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
)

func Test_openconfig_portchannel(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING PORTCHANNEL ++++++++++++")

	t.Log("\n\n--- POST to Create PortChannel 111 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"PortChannel111\", \"config\": {\"name\": \"PortChannel111\", \"mtu\": 9100, \"description\": \"put_pc\", \"enabled\": true}}]}"
	t.Run("Test Create PortChannel111", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PortChannel Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"description\": \"put_pc\", \"enabled\": true, \"mtu\": 9100, \"name\": \"PortChannel111\"}}"
	t.Run("Test GET PortChannel interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Replace/Create PortChannel 123 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel123]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\": \"PortChannel123\", \"config\": {\"name\": \"PortChannel123\", \"mtu\": 9200, \"description\": \"put_pc_updated\", \"enabled\": true}}]}"
	t.Run("Test PUT PortChannel123", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PortChannel Replacement/Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel123]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"put_pc_updated\", \"enabled\": true, \"mtu\": 9200, \"name\": \"PortChannel123\"}}"
	t.Run("Test GET PortChannel interface after PUT", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Initialize PortChannel Member ---")
	t.Log("\n\n--- DELETE interface IP Addr ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("DELETE on interface IP Addr", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to Add PortChannel Member ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/openconfig-if-aggregate:aggregate-id"
	url_input_body_json = "{\"openconfig-if-aggregate:aggregate-id\":\"PortChannel111\"}"
	t.Run("Test PATCH on Ethernet aggregate-id", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify the added PortChannel Member ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/openconfig-if-aggregate:aggregate-id"
	expected_get_json = "{\"openconfig-if-aggregate:aggregate-id\": \"PortChannel111\"}"
	t.Run("Test GET on portchannel agg-id", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH PortChannel min-links ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config/min-links"
	url_input_body_json = "{\"openconfig-if-aggregate:min-links\":3}"
	t.Run("Test PATCH min-links on portchannel", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH PortChannel config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config"
	expected_get_json = "{\"openconfig-if-aggregate:config\": {\"min-links\": 3}}"
	t.Run("Test GET on portchannel config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE PortChannel min-links ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config/min-links"
	t.Run("Verify DELETE on PortChannel min-links", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE PortChannel min-links ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config"
	expected_get_json = "{\"openconfig-if-aggregate:config\": {\"min-links\": 3}}"
	t.Run("Test GET on portchannel min-links after DELETE", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH PortChannel interface Config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"mtu\": 8900, \"description\": \"agg_intf_conf\", \"enabled\": false}}"
	t.Run("Test PATCH PortChannel interface Config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interfaces config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"agg_intf_conf\", \"enabled\": false, \"mtu\": 8900, \"name\": \"PortChannel111\"}}"
	t.Run("Test GET PortChannel interface Config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE PortChannel interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]"
	t.Run("Test DELETE on PortChannel", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at PortChannel Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]"
	err_str := "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted PortChannel", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)
}
