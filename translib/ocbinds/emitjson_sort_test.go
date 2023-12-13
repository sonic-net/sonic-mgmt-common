////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
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

package ocbinds

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/openconfig/ygot/ygot"
)

func TestEmitJSON_sort_ifName(t *testing.T) {
	intfs := new(OpenconfigInterfaces_Interfaces)
	intfs.NewInterface("Ethernet1")
	intfs.NewInterface("Ethernet10")
	intfs.NewInterface("Vlan1")
	intfs.NewInterface("Ethernet2")

	compareSortedJSON(t, intfs, `{"interface":[
		{"name": "Ethernet1"}, {"name": "Ethernet2"}, 
		{"name": "Ethernet10"}, {"name": "Vlan1"} ]}`)
}

func TestEmitJSON_sort_ipAddr4(t *testing.T) {
	addrs := new(OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv4_Addresses)
	addrs.NewAddress("192.168.9.151")
	addrs.NewAddress("10.24.199.62")
	addrs.NewAddress("192.168.9.5")
	addrs.NewAddress("192.168.19.5")
	addrs.NewAddress("25.205.0.0")

	compareSortedJSON(t, addrs, `{"address":[
		{"ip": "10.24.199.62"}, {"ip": "25.205.0.0"}, {"ip": "192.168.9.5"},
		{"ip": "192.168.9.151"}, {"ip": "192.168.19.5"} ]}`)
}

func TestEmitJSON_sort_ipAddr6(t *testing.T) {
	addrs := new(OpenconfigInterfaces_Interfaces_Interface_RoutedVlan_Ipv6_Addresses)
	addrs.NewAddress("0123:deaf::0010")
	addrs.NewAddress("0123:deaf::00a0")
	addrs.NewAddress("0123:deaf::0001")
	addrs.NewAddress("0123:deaf::000a")
	addrs.NewAddress("01c0:9000::0010")

	//Note: natsort does not handle ipv6 addresses.
	//They are treated as regular strings containing numbers.
	compareSortedJSON(t, addrs, `{"address":[
		{"ip": "01c0:9000::0010"}, {"ip": "0123:deaf::000a"}, {"ip": "0123:deaf::00a0"},
		{"ip": "0123:deaf::0001"}, {"ip": "0123:deaf::0010"} ]}`)
}

func TestEmitJSON_sort_uint(t *testing.T) {
	aclRules := new(OpenconfigAcl_Acl_AclSets_AclSet_AclEntries)
	aclRules.NewAclEntry(101)
	aclRules.NewAclEntry(11)
	aclRules.NewAclEntry(5)
	aclRules.NewAclEntry(200)

	compareSortedJSON(t, aclRules, `{"acl-entry":[
		{"sequence-id": 5}, {"sequence-id": 11},
		{"sequence-id": 101}, {"sequence-id": 200} ]}`)
}

// func TestEmitJSON_sort_enum(t *testing.T) {
// 	wr := new(SonicWarmRestart_SonicWarmRestart_WARM_RESTART)
// 	wr.NewWARM_RESTART_LIST(SonicWarmRestart_ModuleName_swss)
// 	wr.NewWARM_RESTART_LIST(SonicWarmRestart_ModuleName_bgp)
// 	wr.NewWARM_RESTART_LIST(SonicWarmRestart_ModuleName_system)
// 	wr.NewWARM_RESTART_LIST(SonicWarmRestart_ModuleName_teamd)

// 	// Note: enums are sorted by their int values. Hence "teamd" < "swss"
// 	compareSortedJSON(t, wr, `{"WARM_RESTART_LIST":[
// 		{"module": "bgp"}, {"module": "teamd"}, {"module": "swss"}, {"module": "system"} ]}`)
// }

func TestEmitJSON_sort_compositeKey(t *testing.T) {
	acls := new(OpenconfigAcl_Acl_AclSets)
	acls.NewAclSet("TEST1", OpenconfigAcl_ACL_TYPE_ACL_IPV4)
	acls.NewAclSet("TEST2", OpenconfigAcl_ACL_TYPE_ACL_IPV4)
	acls.NewAclSet("TEST10", OpenconfigAcl_ACL_TYPE_ACL_IPV4)
	acls.NewAclSet("TEST1", OpenconfigAcl_ACL_TYPE_ACL_IPV6)
	acls.NewAclSet("TEST3", OpenconfigAcl_ACL_TYPE_ACL_L2)

	compareSortedJSON(t, acls, `{"acl-set":[
		{"name": "TEST1", "type": "ACL_IPV4"}, {"name": "TEST1", "type": "ACL_IPV6"},
		{"name": "TEST2", "type": "ACL_IPV4"}, {"name": "TEST3", "type": "ACL_L2"},
		{"name": "TEST10", "type": "ACL_IPV4"} ]}`)
}

//func TestEmitJSON_sort_overflow(t *testing.T) {
//	intfs := new(OpenconfigInterfaces_Interfaces)
//	intfs.NewInterface("111222333444555666777888999000")
//	intfs.NewInterface("18446744073709551616")
//	intfs.NewInterface("22222222222")
//	intfs.NewInterface("18446744073709551615")
//	intfs.NewInterface("1844674407370955161234")
//
//	compareSortedJSON(t, intfs, `{"interface":[
//		{"name": "22222222222"}, {"name": "18446744073709551615"},
//		{"name": "18446744073709551616"}, {"name": "1844674407370955161234"},
//		{"name": "111222333444555666777888999000"} ]}`)
//}

func compareSortedJSON(t *testing.T, y ygot.ValidatedGoStruct, exp string) {
	data, err := EmitJSON(y, &EmitJSONOptions{NoPrefix: true, SortList: true})
	if err != nil {
		t.Fatal(err)
	}

	var expBuff bytes.Buffer
	if err := json.Compact(&expBuff, []byte(exp)); err != nil {
		t.Error("Invalid exp json: ", exp)
		t.Error("Error: ", err)
		return
	}
	if string(data) != expBuff.String() {
		t.Error("EmitJSON returned: ", string(data))
		t.Error("Expected: ", expBuff.String())
	}
}
