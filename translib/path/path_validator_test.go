////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package path

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/ygot/ygot"
)

func TestNewPathValidator(t *testing.T) {
	pathValdtor := NewPathValidator(&(AppendModulePrefix{}), &(AddWildcardKeys{}))
	if pathValdtor != nil {
		t.Log("reflect.ValueOf(pathValdtor.rootObj).Type().Name() ==> ", reflect.ValueOf(pathValdtor.rootObj).Elem().Type().Name())
		if reflect.ValueOf(pathValdtor.rootObj).Elem().Type().Name() != "Device" || pathValdtor.hasIgnoreKeyValidationOption() ||
			!pathValdtor.hasAppendModulePrefixOption() || !pathValdtor.hasAddWildcardKeyOption() {
			t.Error("Error in creating the NewPathValidator")
		}
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		tid       int
		path      string
		resPath   string
		errPrefix string
	}{{
		tid:     1, // validate path and key value
		path:    "/openconfig-acl:acl/acl-sets/acl-set[name=Sample][type=ACL_IPV4]/state/description",
		resPath: "/openconfig-acl:acl/acl-sets/acl-set[name=Sample][type=ACL_IPV4]/state/description",
	}, {
		tid: 2, // fill key name and value as wild card for missing keys
		path: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviation:igmp-snooping/interfaces/interface/config",
		resPath: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviation:igmp-snooping/interfaces/interface[name=*]/config",
	}, {
		tid: 3, // append module prefix
		path: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"igmp-snooping/interfaces",
		resPath: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviation:igmp-snooping/interfaces",
	}, {
		tid:     4, // validate path and key value
		path:    "/openconfig-interfaces:interfaces/interface[name=Ethernet8]/ethernet/switched-vlan/config/trunk-vlans",
		resPath: "/openconfig-interfaces:interfaces/interface[name=Ethernet8]/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans",
	}, {
		tid:     5, // validate path and key value
		path:    "/acl/acl-sets/acl-set[name=Sample][type=ACL_IPV4]/config/type",
		resPath: "/openconfig-acl:acl/acl-sets/acl-set[name=Sample][type=ACL_IPV4]/config/type",
	}, {
		tid: 6, // negative - invalid module prefix
		path: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviationxx:igmp-snooping/interfaces",
		errPrefix: "Invalid yang module prefix in the path node openconfig-network-instance-deviationxx:igmp-snooping",
	}, {
		tid: 7, // negative - invalid path
		path: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviation:igmp-snooping/interfacesxx",
		errPrefix: "Node interfacesxx not found in the given gnmi path elem",
	}, {
		tid: 8, // negative - invalid key name
		path: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=IGMP_SNOOPING][name=IGMP-SNOOPING]/" +
			"openconfig-network-instance-deviation:igmp-snooping/interfaces/interface[namexx=*]/config",
		errPrefix: "Invalid key name: map[namexx:*] in the list node path: interface",
	}, {
		tid:       9, // negative - invalid path
		path:      "/openconfig-system:system/config/hostname/invalid",
		errPrefix: "Node invalid not found in the given gnmi path elem",
	}, { // verify module prefix at list node path
		tid:     10,
		path:    "/openconfig-network-instance:network-instances/network-instance[name=*]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/global/inter-area-propagation-policies/openconfig-ospfv2-ext:inter-area-policy[src-area=0.0.0.1]",
		resPath: "/openconfig-network-instance:network-instances/network-instance[name=*]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/global/inter-area-propagation-policies/openconfig-ospfv2-ext:inter-area-policy[src-area=0.0.0.1]",
	}, { // append module prefix at list node path and verify
		tid:     11,
		path:    "/openconfig-network-instance:network-instances/network-instance[name=*]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/global/inter-area-propagation-policies/inter-area-policy[src-area=0.0.0.1]",
		resPath: "/openconfig-network-instance:network-instances/network-instance[name=*]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/global/inter-area-propagation-policies/openconfig-ospfv2-ext:inter-area-policy[src-area=0.0.0.1]",
	}, {
		tid:     12,
		path:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp/rib/afi-safis/afi-safi[afi-safi-name=L2VPN_EVPN]/openconfig-bgp-evpn-ext:l2vpn-evpn/loc-rib/routes/route[origin=6.6.6.2][path-id=0][prefix=[5\\]:[0\\]:[96\\]:[6601::\\]][route-distinguisher=120.1.1.1:5096]",
		resPath: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp/rib/afi-safis/afi-safi[afi-safi-name=L2VPN_EVPN]/openconfig-bgp-evpn-ext:l2vpn-evpn/loc-rib/routes/route[origin=6.6.6.2][path-id=0][prefix=[5\\]:[0\\]:[96\\]:[6601::\\]][route-distinguisher=120.1.1.1:5096]",
	}}

	pathValdtor := NewPathValidator(&(AppendModulePrefix{}), &(AddWildcardKeys{}))
	for _, tt := range tests {
		gPath, err := ygot.StringToPath(tt.path, ygot.StructuredPath)
		if err != nil {
			t.Error("Error in uri to path conversion: ", err)
			break
		}
		pathValdtor.init(gPath)
		err = pathValdtor.validatePath()
		if err != nil {
			if !strings.HasPrefix(err.Error(), tt.errPrefix) {
				t.Errorf("Testcase %v failed; error: %v", tt.tid, err)
			}
			return
		}
		if resPath, err := ygot.PathToString(gPath); resPath != tt.resPath {
			t.Errorf("Testcase %v failed; error: %v; result path: %v", tt.tid, err, resPath)
		}
	}
}

func BenchmarkValidatePath1(b *testing.B) {
	pathValdtor := NewPathValidator(&(AppendModulePrefix{}), &(AddWildcardKeys{}))
	gPath, err := ygot.StringToPath("/openconfig-tam:tam/flowgroups/flowgroup[name=*]/config/priority", ygot.StructuredPath)
	if err != nil {
		b.Error("Error in uri to path conversion: ", err)
		return
	}
	for i := 0; i < b.N; i++ {
		pathValdtor.Validate(gPath)
	}
}

func BenchmarkValidatePath2(b *testing.B) {
	pathValdtor := NewPathValidator(&(AppendModulePrefix{}), &(AddWildcardKeys{}))
	gPath, err := ygot.StringToPath("/openconfig-interfaces:interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/openconfig-if-ip:ipv6/addresses/address[ip=*]/state", ygot.StructuredPath)
	if err != nil {
		b.Error("Error in uri to path conversion: ", err)
		return
	}
	for i := 0; i < b.N; i++ {
		pathValdtor.Validate(gPath)
	}
}

func BenchmarkValidatePath3(b *testing.B) {
	pathValdtor := NewPathValidator(&(AppendModulePrefix{}), &(AddWildcardKeys{}))
	gPath, err := ygot.StringToPath("/openconfig-network-instance:network-instances/network-instance[name=*]/protocols/protocol[identifier=*][name=*]"+
		"/ospfv2/areas/area[identifier=*]/lsdb/lsa-types/lsa-type[type=*]/lsas/lsa-ext[link-state-id=*][advertising-router=*]"+
		"/opaque-lsa/extended-prefix/tlvs/tlv/sid-label-binding/tlvs/tlv/ero-path/segments/segment/unnumbered-hop/state/router-id", ygot.StructuredPath)
	if err != nil {
		b.Error("Error in uri to path conversion: ", err)
		return
	}
	for i := 0; i < b.N; i++ {
		pathValdtor.Validate(gPath)
	}
}
