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
	"errors"
	"github.com/Azure/sonic-mgmt-common/translib/db"
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
	expected_get_json := "{\"openconfig-interfaces:config\": {\"description\": \"put_pc\", \"enabled\": true, \"mtu\": 9100, \"name\": \"PortChannel111\", \"type\": \"iana-if-type:ieee8023adLag\"}}"
	t.Run("Test GET PortChannel interface creation config ", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv4 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"64.64.64.64\", \"openconfig-if-ip:config\": {\"ip\": \"64.64.64.64\", \"prefix-length\": 24}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv4 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv4 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv4\": {\"addresses\": {\"address\": [{\"config\": {\"ip\": \"64.64.64.64\", \"prefix-length\": 24}, \"ip\": \"64.64.64.64\"}]}}, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Patch IPv4 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PUT to Replace/Create PortChannel 123 ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel123]"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\": \"PortChannel123\", \"config\": {\"name\": \"PortChannel123\", \"mtu\": 9200, \"description\": \"put_pc_updated\", \"enabled\": true}}]}"
	t.Run("Test PUT PortChannel123", processSetRequest(url, url_input_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PortChannel Replacement/Creation ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel123]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"put_pc_updated\", \"enabled\": true, \"mtu\": 9200, \"name\": \"PortChannel123\", \"type\": \"iana-if-type:ieee8023adLag\"}}"
	t.Run("Test GET PortChannel interface after PUT", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Initialize PortChannel Member ---")
	t.Log("\n\n--- DELETE interface IP Addr ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("DELETE on interface IP Addr", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ CONFIGURING AND REMOVING IPv4 ADDRESS AT SUBINTERFACES  PORTCHANNEL INTERFACE ++++++++++++")
	t.Log("\n\n---  Delete/Clear existing IPv4 address on PortChannel Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses"
	t.Run("Test Delete/Clear IPv4 on subinterfaces", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Get/Verify IPv4 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get IPv4 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
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

	//Verify adding new agg-id on same port
	t.Log("\n\n--- Verify adding new agg-id on same port ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interface\": [{\"name\":\"PortChannel222\", \"config\": {\"name\": \"PortChannel222\", \"mtu\": 9100, \"description\": \"put_pc\", \"enabled\": true}}]}"
	t.Run("Test Create PortChannel222", processSetRequest(url, url_input_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/openconfig-if-aggregate:aggregate-id"
	url_input_body_json = "{\"openconfig-if-aggregate:aggregate-id\":\"PortChannel222\"}"
	agg_exist_err := errors.New("Ethernet0 Interface is already member of PortChannel111")
	t.Run("Test PATCH on Ethernet aggregate-id error case", processSetRequest(url, url_input_body_json, "PATCH", true, agg_exist_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH to re-Add PortChannel Member ---")
	url = "/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-ethernet:ethernet/config/openconfig-if-aggregate:aggregate-id"
	url_input_body_json = "{\"openconfig-if-aggregate:aggregate-id\":\"PortChannel111\"}"
	t.Run("Test PATCH on Ethernet aggregate-id re-add", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interfaces type ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"description\": \"UT_Interface\", \"enabled\": false, \"type\": \"iana-if-type:ieee8023adLag\"}}"
	t.Run("Test PATCH on interface type config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interfaces type config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"UT_Interface\", \"enabled\": false, \"name\": \"PortChannel111\", \"type\": \"iana-if-type:ieee8023adLag\", \"mtu\": 9100}}"
	t.Run("Test GET on interface type config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interfaces wrong type ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"description\": \"UT_Interface\", \"enabled\": false, \"type\": \"iana-if-type:softwareLoopback\"}}"
	wrong_type_err := errors.New("Invalid interface type received")
	t.Run("Test PATCH on interface wrong type config", processSetRequest(url, url_input_body_json, "PATCH", true, wrong_type_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH interfaces loopback-mode ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"description\": \"UT_Interface\", \"enabled\": false, \"type\": \"iana-if-type:ieee8023adLag\", \"loopback-mode\": \"NONE\"}}"
	lo_mode_not_supported_msg := "Invalid interface type for loopback-mode config"
	lo_mode_not_supported := errors.New(lo_mode_not_supported_msg)
	t.Run("Test PATCH on interface loopback-mode config", processSetRequest(url, url_input_body_json, "PATCH", true, lo_mode_not_supported))
	time.Sleep(1 * time.Second)

	pre_req_map := map[string]interface{}{"LAG_TABLE": map[string]interface{}{"PortChannel111": map[string]interface{}{"description": "UT-Po-Port", "admin_status": "up", "index": "100001", "oper_status": "up", "last_up_time": "Sat Feb 08 11:53:34 2025", "last_down_time": "Sat Feb 08 11:53:37 2025", "mtu": "8888"}}}
	loadDB(db.ApplDB, pre_req_map)

	t.Log("\n\n--- Verify interface state leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/state"
	expected_get_json = "{\"openconfig-interfaces:state\": {\"logical\": true, \"management\": false, \"cpu\": false, \"type\": \"iana-if-type:ieee8023adLag\", \"description\": \"UT-Po-Port\", \"ifindex\": 100001, \"oper-status\": \"UP\", \"last-change\": \"173901561700\", \"name\": \"PortChannel111\", \"mtu\": 8888, \"admin-status\": \"UP\", \"enabled\": true}}"
	t.Run("Test GET on interface state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	cleanuptbl := map[string]interface{}{"LAG_TABLE": map[string]interface{}{"PortChannel111": ""}}
	unloadDB(db.ApplDB, cleanuptbl)

	t.Log("\n\n+++++++++++++ CONFIGURING ETHERNET ATTRIBUTES ++++++++++++")
	t.Log("\n\n--- PATCH ethernet auto-neg and port-speed ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-ethernet:ethernet/config/port-speed"
	url_input_body_json = "{\"openconfig-if-ethernet:port-speed\":\"SPEED_40GB\"}"
	speed_err := errors.New("Speed config not supported for given Interface type")
	t.Run("Test PATCH on ethernet port-speed", processSetRequest(url, url_input_body_json, "PATCH", true, speed_err))
	time.Sleep(1 * time.Second)

	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-ethernet:ethernet/config/auto-negotiate"
	url_input_body_json = "{\"openconfig-if-ethernet:auto-negotiate\":true}"
	auto_neg_err := errors.New("AutoNegotiate config not supported for given Interface type")
	t.Run("Test PATCH on ethernet auto-neg", processSetRequest(url, url_input_body_json, "PATCH", true, auto_neg_err))
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
	t.Run("Verify DELETE on PortChannel min-links", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE PortChannel min-links ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config"
	expected_get_json = "{\"openconfig-if-aggregate:config\": {\"min-links\": 1}}"
	t.Run("Test GET on portchannel min-links after DELETE", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH PortChannel lag-type ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config/lag-type"
	url_input_body_json = "{\"openconfig-if-aggregate:lag-type\":\"LACP\"}"
	t.Run("Test PATCH lag-type on portchannel", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH PortChannel config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config"
	expected_get_json = "{\"openconfig-if-aggregate:config\": {\"lag-type\": \"LACP\", \"min-links\": 1}}"
	t.Run("Test GET on portchannel lag-type config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify interface state leaf nodes  ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/state"
	expected_get_json = "{\"openconfig-if-aggregate:state\":{\"min-links\":1 ,\"lag-type\":\"LACP\"}}"
	t.Run("Test GET on interface aggregation state", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- DELETE PortChannel lag-type ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config/lag-type"
	t.Run("Verify DELETE on PortChannel lag-type", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE PortChannel lag-type ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config"
	expected_get_json = "{\"openconfig-if-aggregate:config\": {\"min-links\": 1}}"
	t.Run("Test GET on portchannel lag-type after DELETE", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH PortChannel lag-type wrong value ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/openconfig-if-aggregate:aggregation/config/lag-type"
	url_input_body_json = "{\"openconfig-if-aggregate:lag-type\":\"STATIC\"}"
	lacp_type_err := errors.New("Invalid lag-type config, Only LACP mode supported")
	t.Run("Test PATCH lag-type on portchannel wrong value", processSetRequest(url, url_input_body_json, "PATCH", true, lacp_type_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH PortChannel interface Config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	url_input_body_json = "{\"openconfig-interfaces:config\": { \"mtu\": 8900, \"description\": \"agg_intf_conf\", \"enabled\": false}}"
	t.Run("Test PATCH PortChannel interface Config", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH interfaces config ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/config"
	expected_get_json = "{\"openconfig-interfaces:config\": {\"description\": \"agg_intf_conf\", \"enabled\": false, \"mtu\": 8900, \"name\": \"PortChannel111\", \"type\": \"iana-if-type:ieee8023adLag\"}}"
	t.Run("Test GET PortChannel interface Config", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- PATCH IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"f::f\", \"openconfig-if-ip:config\": {\"ip\": \"f::f\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify IPv6 address at subinterfaces level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\":{\"subinterface\":[{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"f::f\",\"prefix-length\":64},\"ip\":\"f::f\"}]},\"config\":{\"enabled\":false}, \"state\": {\"enabled\": false}}, \"state\":{\"index\":0}}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces level", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces"
	t.Run("Test Delete IPv6 address at subinterfaces", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces"
	expected_get_json = "{\"openconfig-interfaces:subinterfaces\": {\"subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterfaces", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv6 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"b::e\", \"openconfig-if-ip:config\": {\"ip\": \"b::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 address at subinterface level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\":{\"index\":0},\"index\":0,\"openconfig-if-ip:ipv6\":{\"addresses\":{\"address\":[{\"config\":{\"ip\":\"b::e\",\"prefix-length\":64},\"ip\":\"b::e\"}]},\"config\":{\"enabled\":false}, \"state\": {\"enabled\": false}}, \"state\":{\"index\":0}}]}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface"
	t.Run("Test Delete IPv6 address at subinterface", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface"
	expected_get_json = "{\"openconfig-interfaces:subinterface\": [{\"config\": {\"index\": 0}, \"index\": 0, \"openconfig-if-ip:ipv6\": {\"config\": {\"enabled\": false}, \"state\": {\"enabled\": false}}, \"state\": {\"index\": 0}}]}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterface", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- PATCH IPv6 address at addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	url_input_body_json = "{\"openconfig-if-ip:addresses\": {\"address\": [{\"ip\": \"2001::e\", \"openconfig-if-ip:config\": {\"ip\": \"2001::e\", \"prefix-length\": 64}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test Patch/Set IPv6 address on subinterfaces addresses", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify PATCH IPv6 address at addresses level ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	expected_get_json = "{\"openconfig-if-ip:addresses\":{\"address\":[{\"config\":{\"ip\":\"2001::e\",\"prefix-length\":64},\"ip\":\"2001::e\"}]}}"
	t.Run("Test Get/Verify Patch IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Delete IPv6 address at subinterfaces addresses level---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses"
	t.Run("Test Delete IPv6 address at subinterfaces addresses level", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify Delete IPv6 address at subinterfaces addresses ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]/subinterfaces/subinterface[index=0]/ipv6/addresses"
	expected_get_json = "{}"
	t.Run("Test Get/Verify Delete IPv6 address at subinterfaces addresses", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	//------------------------------------------------------------------------------------------------------------------------------------

	t.Log("\n\n--- DELETE PortChannel interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]"
	t.Run("Test DELETE on PortChannel", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n--- Verify DELETE at PortChannel Interface ---")
	url = "/openconfig-interfaces:interfaces/interface[name=PortChannel111]"
	err_str := "Resource not found"
	expected_err_invalid := tlerr.NotFoundError{Format: err_str}
	t.Run("Test GET on deleted PortChannel", processGetRequest(url, nil, "", true, expected_err_invalid))
	time.Sleep(1 * time.Second)
}
