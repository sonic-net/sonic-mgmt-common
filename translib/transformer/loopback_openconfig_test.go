////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2025 Cisco.                                                 //
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
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"testing"
	"time"
)

func Test_openconfig_loopback_interfaces(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING LOOPBACK ++++++++++++")

	t.Log("\n\n--- POST to Create Loopback11 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback11\", \"config\": {\"name\": \"Loopback11\"}}]}"
	t.Run("Test Create Loopback11", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"name\": \"Loopback11\",  \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Create Loopback22 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback22\", \"config\": {\"name\": \"Loopback22\"}}]}"
	t.Run("Test Create Loopback22", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"name\": \"Loopback22\", \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ CONFIGURING INTERFACES ATTRIBUTES ++++++++++++")
	t.Log("\n\n--- PATCH interface leaf nodes---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/config/mtu"
	url_input_body_json = "{\"openconfig-interfaces:mtu\": 9000}"
	err_mtu_str := "Configuration for MTU is not supported for Loopback interface "
	var expected_mtu_err error = errors.New(err_mtu_str)
	t.Run("Test PATCH on interface mtu", processSetRequest(url, url_input_body_json, "PATCH", true, expected_mtu_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Create Loopback100 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback100\", \"config\": {\"name\": \"Loopback100\"}}]}"
	t.Run("Test PUT with Loopback100", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	pre_req_map := map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Loopback100": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify Loopback Creation Loockback100 --")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]"
	expected_get_json = "{\"openconfig-interfaces:interface\":[{ \"config\": {\"name\": \"Loopback100\", \"enabled\": true}, \"state\": {\"name\": \"Loopback100\"}, \"name\": \"Loopback100\", \"subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"state\": {\"index\": 0}}]}}]}"
	t.Run("Test GET Loopback100 interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interface leaf enabled node ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/enabled"
	url_input_body_json = "{\"openconfig-interfaces:enabled\": true}"
	t.Run("Test PATCH on interface enabled", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interface leaf enabled node  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/enabled"
	expected_get_json = "{\"openconfig-interfaces:enabled\": true}"
	t.Run("Test GET on interface PATCH enabled config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interface enabled  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/enabled"
	disable_err := errors.New("Disabling Loopback port is not supported")
	t.Run("Test DELETE on interface enabled", processDeleteRequest(url, true, disable_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at interface enabled  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/enabled"
	expected_get_json = "{\"openconfig-interfaces:enabled\": true}"
	t.Run("Test GET on interface enabled delete config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interface leaf enabled node false ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/enabled"
	url_input_body_json = "{\"openconfig-interfaces:enabled\": false}"
	t.Run("Test PATCH on interface enabled", processSetRequest(url, url_input_body_json, "PATCH", true, disable_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- GET interface desc before patch ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/description"
	err_str := "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	expected_get_json = "{}"
	t.Run("Test GET on interface config", processGetRequest(url, nil, expected_get_json, true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interfaces desc ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"description\": \"UT_Loopback_Interface\", \"enabled\": true }}"
	t.Run("Test PATCH on interface description config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interfaces desc config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"UT_Loopback_Interface\", \"enabled\": true, \"name\": \"Loopback100\"}}"
	t.Run("Test GET on interface desc config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interface desc  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/description"
	t.Run("Test DELETE on interface desc", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at interface desc  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/config/description"
	err_str = "Resource not found"
	expected_err_invalid = tlerr.NotFoundError{Format: err_str}
	expected_get_json = "{}"
	t.Run("Test GET on interface config", processGetRequest(url, nil, expected_get_json, true, expected_err_invalid))
	time.Sleep(1 * time.Second)

	cleanuptbl := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": ""}}
	unloadDB(db.ApplDB, cleanuptbl)

	//-----------

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"24.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"24.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"24.4.4.4\", \"prefix-length\": 24}, \"ip\": \"24.4.4.4\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//patch to update existing value (changing prefix length of ip)
	t.Log("\n\n--- PATCH IPv4 Prefix length address at addresses level  from 24 to 32 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"24.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"24.4.4.4\", \"prefix-length\": 32}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"24.4.4.4\", \"prefix-length\": 32}, \"ip\": \"24.4.4.4\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//patch to add new ip || for Ipv4 currently only one IP is allowed per interface
	//Primary IP config already happened and replacing it with new one

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"56.40.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"56.40.4.4\", \"prefix-length\": 32}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"56.40.4.4\", \"prefix-length\": 32}, \"ip\": \"56.40.4.4\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Put IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"44.40.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"44.40.4.4\", \"prefix-length\": 32}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Put/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"44.40.4.4\", \"prefix-length\": 32}, \"ip\": \"44.40.4.4\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run(" Verify Put IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//Test conflicting IP (same subnet)
	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"44.40.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"44.40.4.4\", \"prefix-length\": 32}}]}}"

	err_str = "IP 44.40.4.4/32 overlaps with IP or IP Anycast 44.40.4.4/32 of Interface Loopback11"
	expected_err1 := tlerr.InvalidArgsError{Format: err_str}

	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on another interface", processSetRequest(url, url_input_body_json, "PATCH", true, expected_err1))
	time.Sleep(1 * time.Second)

	//--------------
	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"34.4.4.4\", \"openconfig-if-ip:config\": {\"ip\": \"34.4.4.4\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"34.4.4.4\", \"prefix-length\": 24}, \"ip\": \"34.4.4.4\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interfaces/interface container  Loopback22 interface---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]"
	t.Run("Test DELETE on interface container", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at Loopback22 Interface  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback22]"
	err_str = "Resource not found"
	expected_get_json = "{}"
	expected_err := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted Loopback interface", processGetRequest(url, nil, expected_get_json, true, expected_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Test ethernet container ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/openconfig-if-ethernet:ethernet/config/port-speed"
	url_input_body_json = "{\"openconfig-if-ethernet:port-speed\":\"SPEED_40GB\"}"
	err_str = "Error: Unsupported Interface: Loopback11"
	invalid_port_err := errors.New(err_str)
	t.Run("Test ethernet container", processSetRequest(url, url_input_body_json, "PATCH", true, invalid_port_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Test aggregate container ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]/openconfig-if-aggregate:aggregation/config/min-links"
	url_input_body_json = "{\"openconfig-if-aggregator:min-links\": 2}"
	err_str = "Container not supported for given interface type"
	invalid_port_err = errors.New(err_str)
	t.Run("Test aggregator container", processSetRequest(url, url_input_body_json, "PATCH", true, invalid_port_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interfaces/interface container Loopback11 interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]"
	t.Run("Test DELETE on interface container", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at Loopback11 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback11]"
	err_str = "Resource not found"
	expected_err = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted Loopback interface", processGetRequest(url, nil, "", true, expected_err))
	time.Sleep(1 * time.Second)
}

func Test_openconfig_loopback_ipv6_ipv4_addresses(t *testing.T) {
	t.Log("\n\n+++++++++++++ CONFIGURING LOOPBACK ++++++++++++")

	t.Log("\n\n--- PUT to Create Loopback14 ---")
	url := "/openconfig-interfaces:interfaces/interface[name=Loopback14]"
	url_input_body_json := "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback14\", \"config\": {\"name\": \"Loopback14\"}}]}"
	t.Run("Test Create Loopback14", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"name\": \"Loopback14\", \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Create Loopback15 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback15\", \"config\": {\"name\": \"Loopback15\"}}]}"
	t.Run("Test Create Loopback15", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"name\": \"Loopback15\", \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Create Loopback21 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback21\", \"config\": {\"name\": \"Loopback21\"}}]}"
	t.Run("Test Create Loopback21", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"name\": \"Loopback21\", \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n-- PATCH IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2000::24\", \"openconfig-if-ip:config\": {\"ip\": \"2000::24\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n---  Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2000::24\",\"prefix-length\":64},\"ip\":\"2000::24\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//Test conflicting IP (same subnet)
	t.Log("\n\n---PATCH IPv6 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2000::24\", \"openconfig-if-ip:config\": {\"ip\": \"2000::24\", \"prefix-length\": 64}}]}}"
	err_str := "IP 2000::24/64 overlaps with IP or IP Anycast 2000::24/64 of Interface Loopback21"
	expected_err1 := tlerr.InvalidArgsError{Format: err_str}
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on another interface", processSetRequest(url, url_input_body_json, "PATCH", true, expected_err1))
	time.Sleep(1 * time.Second)

	// patch with same ip (updating existing address)
	t.Log("\n\n--- PATCH IPv6 address at subinterfaces addresses  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2000::24\", \"openconfig-if-ip:config\": {\"ip\": \"2000::24\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n---  Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2000::24\",\"prefix-length\":64},\"ip\":\"2000::24\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run(" Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	// patch with new ip (adding new address)
	t.Log("\n\n--- PATCH IPv6 address at subinterfaces addresses  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"8000::42\", \"openconfig-if-ip:config\": {\"ip\": \"8000::42\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2000::24\",\"prefix-length\":64},\"ip\":\"2000::24\"},{\"config\":{\"ip\":\"8000::42\",\"prefix-length\":64},\"ip\":\"8000::42\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run("  Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//-----------

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"74.74.74.74\", \"openconfig-if-ip:config\": {\"ip\": \"74.74.74.74\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"74.74.74.74\", \"prefix-length\": 24}, \"ip\": \"74.74.74.74\"}]}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2004::2\", \"openconfig-if-ip:config\": {\"ip\": \"2004::2\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2004::2\",\"prefix-length\":64},\"ip\":\"2004::2\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))

	time.Sleep(1 * time.Second)

	//***currently for ipv6, replace operation doesn't remove Old Data, it just Appends the new data
	//curr [a], put [b], ideally we should have [b], we have [a,b]
	t.Log("\n\n--- PUT IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2006::8\", \"openconfig-if-ip:config\": {\"ip\": \"2006::8\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Put/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n---Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2004::2\",\"prefix-length\":64},\"ip\":\"2004::2\"},{\"config\":{\"ip\":\"2006::8\",\"prefix-length\":64},\"ip\":\"2006::8\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify put IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//we have [a,b], PUT [a_update], ideally we should have only [a_update]
	//but we have [a_update, b]
	t.Log("\n\n--- PUT IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2004::2\", \"openconfig-if-ip:config\": {\"ip\": \"2004::2\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Put/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2004::2\",\"prefix-length\":64},\"ip\":\"2004::2\"},{\"config\":{\"ip\":\"2006::8\",\"prefix-length\":64},\"ip\":\"2006::8\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify put IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	//------------------------------------------------------------------------------------------------------------------------------------
	err_str = "Resource not found"
	expected_err := tlerr.NotFoundError{Format: err_str}

	t.Log("\n\n+++++++++++++ REMOVING IPv4 ADDRESS AT SUBINTERFACES  LOOPBACK INTERFACE ++++++++++++")
	t.Log("\n\n---  Delete/Clear existing IPv4 address on Loopback Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("Test Delete/Clear IPv4 on subinterfaces", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get/Verify IPv4 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback14]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4"
	expected_get_json = "{}"
	t.Run("Test Get/Verify Delete IPv4 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces addresses level---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete IPv6 address at subinterfaces addresses level", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]/subinterfaces/subinterface[index=0]/ipv6/addresses"
	expected_get_json = "{}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n---Delete existing IPv6 address on Loopback Interface at subinterfaces addresses address level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=8000::42]"
	t.Run("Test Delete IPv6 on subinterfaces addresses address[x]", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify delete IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"2000::24\",\"prefix-length\":64},\"ip\":\"2000::24\"}]}}, \"state\":{\"index\":0}}]}}"
	t.Run(" Test Get/Verify delete IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces addresses level---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete IPv6 address at subinterfaces addresses level", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]/subinterfaces/subinterface[index=0]/ipv6/addresses"
	expected_get_json = "{}"
	t.Run(" Test Get/Verify Delete IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)

	//------

	t.Log("\n\n--- DELETE at Loopback21 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]"
	t.Run(" Test DELETE loopback interface", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at Loopback21 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback21]"
	err_str = "Resource not found"
	expected_err = tlerr.NotFoundError{Format: err_str}
	expected_get_json = "{}"
	t.Run("Test GET on deleted Loopback interface", processGetRequest(url, nil, expected_get_json, true, expected_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at Loopback15 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]"
	t.Run("Test DELETE Loopback interface", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at Loopback15 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback15]"
	err_str = "Resource not found"
	expected_err = tlerr.NotFoundError{Format: err_str}
	expected_get_json = "{}"
	t.Run("Test GET on deleted Loopback interface", processGetRequest(url, nil, expected_get_json, true, expected_err))
	time.Sleep(1 * time.Second)
}

func Test_openconfig_loopback_state_params(t *testing.T) {
	t.Log("\n\n--- PUT to Create Loopback16 ---")
	url := "/openconfig-interfaces:interfaces/interface[name=Loopback16]"
	url_input_body_json := "{\"openconfig-interfaces:interface\": [{\"name\":\"Loopback16\", \"config\": {\"name\": \"Loopback16\"}}]}"
	t.Run("Test Create Loopback16", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback16]/config"
	expected_get_json := "{\"openconfig-interfaces:config\": {\"name\": \"Loopback16\", \"enabled\": true}}"
	t.Run("Test GET Loopback interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback interface state leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback16]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": {\"name\": \"Loopback16\"}}"
	t.Run("Test GET on interface state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE at interfaces/interface container Loopback16 interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback16]"
	t.Run("Test DELETE on interface container", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at Loopback16 Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback16]"
	err_str := "Resource not found"
	expected_err := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted Loopback interface", processGetRequest(url, nil, "", true, expected_err))
	time.Sleep(1 * time.Second)

	pre_req_map := map[string]interface{}{"INTF_TABLE": map[string]interface{}{"Loopback100": map[string]interface{}{"description": "UT-Lo-Port", "admin_status": "up"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify Loopback interface all state leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": { \"name\": \"Loopback100\", \"description\": \"UT-Lo-Port\", \"enabled\": true, \"oper-status\": \"UP\", \"admin-status\": \"UP\" }}"
	t.Run("Test GET on interface state all nodes", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Loopback interface state/mtu  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Loopback100]/state/mtu"
	expected_get_json = "{}"
	err_str = "Resource not found"
	expected_err = tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on interface state/mtu", processGetRequest(url, nil, expected_get_json, true, expected_err))
	time.Sleep(1 * time.Second)

	cleanuptbl := map[string]interface{}{"PORT_TABLE": map[string]interface{}{"Ethernet0": ""}}
	unloadDB(db.ApplDB, cleanuptbl)
}
