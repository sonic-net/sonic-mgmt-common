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

func Test_openconfig_subintf(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING AND REMOVING IPv4 ADDRESS AT SUBINTERFACES ++++++++++++")
	t.Log("\n\n--- TC 1: Delete/Clear existing IPv4 address on Ethernet0 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("Test Delete/Clear IPv4 on subinterfaces", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get/Verify IPv4 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json := "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\", \"state\":{ \"family\":\"ipv4\", \"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	t.Run("Test Delete IPv4 address at subinterfaces level", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test/Verify Get at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//-------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Get at subinterfaces/subinterface[index=0] level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\", \"state\":{ \"family\":\"ipv4\", \"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterface[index=0]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
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
	expected_get_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}, \"ip\": \"4.4.4.4\", \"state\":{ \"family\":\"ipv4\", \"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"4.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv4 address at addresses/address/config level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=4.4.4.4]/config"
	expected_get_json = "{\"openconfig-if-ip:config\": {\"ip\": \"4.4.4.4\", \"prefix-length\": 24}}"
	t.Run("Test Get IPv4 address at subinterfaces ipv4/config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Verify Get at interfaces/interface[name=Ethernet0] ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]"
	expected_get_json = "{\"openconfig-interfaces:interface\":[{\"config\":{\"enabled\":true,\"mtu\":9100,\"name\":\"Ethernet0\"},\"openconfig-if-ethernet:ethernet\":{\"config\":{\"auto-negotiate\":false,\"port-speed\":\"openconfig-if-ethernet:SPEED_10GB\"},\"state\":{\"auto-negotiate\":false,\"port-speed\":\"openconfig-if-ethernet:SPEED_10GB\"}},\"name\":\"Ethernet0\",\"state\":{\"admin-status\":\"UP\",\"description\":\"\",\"enabled\":true,\"mtu\":9100,\"name\":\"Ethernet0\"},\"subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv4\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"4.4.4.4\",\"prefix-length\":24},\"ip\":\"4.4.4.4\",\"state\":{\"family\":\"ipv4\",\"ip\":\"4.4.4.4\",\"prefix-length\":24}}]}},\"openconfig-if-ip:ipv6\":{\"config\":{\"enabled\":false}},\"state\":{\"index\":0}}]}}]}"
	t.Run("Test Get at interfaces/interface[name=Ethernet0]", processGetRequest(url, nil, expected_get_json, false))
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
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ DONE CONFIGURING AND REMOVING IPV4 ADDRESSES ON SUBINTERFACES  ++++++++++++")

	t.Log("\n\n+++++++++++++ CONFIGURING AND REMOVING IPv6 ADDRESS AT SUBINTERFACES ++++++++++++")
	t.Log("\n\n--- Delete/Clear IPv6 address ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete/Clear IPv6 address on subinterfaces addresses", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\",\"state\":{\"family\":\"ipv6\",\"ip\":\"a::e\",\"prefix-length\":64}}]},\"config\":{\"enabled\":false}},\"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	t.Run("Test Delete IPv6 address at subinterfaces", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
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
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\",\"state\":{\"family\":\"ipv6\",\"ip\":\"a::e\",\"prefix-length\":64}}]},\"config\":{\"enabled\":false}},\"state\":{\"index\":0}}]}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface"
	t.Run("Test Delete IPv6 address at subinterface", processDeleteRequest(url, true))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
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
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\",\"state\":{\"family\":\"ipv6\",\"ip\":\"a::e\",\"prefix-length\":64}}]}}"
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

	t.Log("\n\n--- PATCH IPv6 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"a::e\", \"openconfig-if-ip:config\": {\"ip\": \"a::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 address at subinterfaces ipv6/config level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=a::e]/config"
	expected_get_json = "{\"openconfig-if-ip:config\": {\"ip\": \"a::e\", \"prefix-length\": 64}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces ipv6/config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Verify Get at interfaces/interface[name=Ethernet0] ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]"
	expected_get_json = "{\"openconfig-interfaces:interface\":[{\"config\":{\"enabled\":true,\"mtu\":9100,\"name\":\"Ethernet0\"},\"openconfig-if-ethernet:ethernet\":{\"config\":{\"auto-negotiate\":false,\"port-speed\":\"openconfig-if-ethernet:SPEED_10GB\"},\"state\":{\"auto-negotiate\":false,\"port-speed\":\"openconfig-if-ethernet:SPEED_10GB\"}},\"name\":\"Ethernet0\",\"state\":{\"admin-status\":\"UP\",\"description\":\"\",\"enabled\":true,\"mtu\":9100,\"name\":\"Ethernet0\"},\"subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"a::e\",\"prefix-length\":64},\"ip\":\"a::e\",\"state\":{\"family\":\"ipv6\",\"ip\":\"a::e\",\"prefix-length\":64}}]},\"config\":{\"enabled\":false}},\"state\":{\"index\":0}}]}}]}"

	t.Run("Test Get at interfaces/interface[name=Ethernet0]", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- Negative test: Delete IPv6 config container ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/config"
	err_str = "Delete not allowed at this container"
	expected_err_2 = tlerr.NotSupportedError{Format: err_str}
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
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
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
