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
	//"errors"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
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
	url_input_body_json = "{\"openconfig-interfaces:description\": \"UT_LEAF\"}"
	t.Run("Test PATCH on interface description", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interface leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": { \"admin-status\": \"UP\", \"description\": \"UT_LEAF\", \"enabled\": true, \"mtu\": 9000, \"name\": \"Ethernet0\"}}"
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
	url_input_body_json = "{\"openconfig-interfaces:interface\":[{\"name\":\"Ethernet0\",\"config\":{\"name\":\"Ethernet0\",\"mtu\":9000,\"description\":\"UT_TOP\",\"enabled\":true}}]}"
	t.Run("Test PATCH on interface", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": { \"admin-status\": \"UP\", \"description\": \"UT_TOP\", \"enabled\": true, \"mtu\": 9000, \"name\": \"Ethernet0\"}}"
	t.Run("Test GET on interface state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
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

	t.Log("\n\n--- Verify PATCH ethernet ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet"
	expected_get_json := "{\"openconfig-if-ethernet:ethernet\": {\"config\": {\"auto-negotiate\": true,\"port-speed\": \"openconfig-if-ethernet:SPEED_40GB\"},\"state\": {\"auto-negotiate\": true,\"port-speed\": \"openconfig-if-ethernet:SPEED_40GB\"}}}"
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
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/state"
	expected_get_json = "{\"openconfig-if-ethernet:state\": {\"auto-negotiate\": true,\"port-speed\": \"openconfig-if-ethernet:SPEED_40GB\"}}"
	t.Run("Test GET on ethernet state", processGetRequest(url, nil, expected_get_json, false))
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
