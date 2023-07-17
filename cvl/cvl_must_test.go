////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package cvl_test

import (
	"testing"
	"github.com/Azure/sonic-mgmt-common/cvl"
)

func TestValidateEditConfig_Delete_Must_Check_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"PORT" : map[string]interface{} {
			"Ethernet3" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
			},
			"Ethernet5" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
				"ports@": "Ethernet3,Ethernet5",
			},
			"TestACL2": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
		"ACL_RULE" : map[string]interface{} {
			"TestACL1|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"L4_SRC_PORT": "1909",
				"IP_PROTOCOL": "103",
				"DST_IP": "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
			"TestACL2|Rule2": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"L4_SRC_PORT": "1909",
				"IP_PROTOCOL": "103",
				"DST_IP": "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataAclRule :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_RULE|TestACL2|Rule2",
			map[string]string {
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataAclRule, Success)
}

func TestValidateEditConfig_Delete_Must_Check_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"PORT" : map[string]interface{} {
			"Ethernet3" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
				//"mtu": "9100",
			},
			"Ethernet5" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "85,86,87,89",
				//"mtu": "9100",
			},
		},
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
				"ports@": "Ethernet3,Ethernet5",
			},
		},
		"ACL_RULE" : map[string]interface{} {
			"TestACL1|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"L4_SRC_PORT": "1909",
				"IP_PROTOCOL": "103",
				"DST_IP": "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataAclRule :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string {
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataAclRule, CVLErrorInfo{
		ErrCode:          CVL_SEMANTIC_ERROR,
		TableName:        "ACL_RULE",
		Keys:             []string{"TestACL1", "Rule1"},
		Field:            "aclname",
		Value:            "TestACL1",
		Msg:              mustExpressionErrMessage,
		ConstraintErrMsg: "Ports are already bound to this rule.",
	})
}

func TestValidateEditConfig_Create_ErrAppTag_In_Must_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "102",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "VLAN",
		//Keys:      []string{"Vlan1001"},   <<<< BUG: key is not filled if must expr is defined on list
		Msg:       mustExpressionErrMessage,
		ErrAppTag: "vlan-invalid",
	})
}

func TestValidateEditConfig_MustExp_With_Default_Value_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan2001": map[string] interface{} {
				"vlanid":   "2001",
			},
		},
	}


	//Try to create er interface collding with vlan interface IP prefix
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "CFG_L2MC_TABLE|Vlan2001",
			map[string]string{
				"enabled":   "true",
				"query-max-response-time": "25", //default query-interval = 125
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_MustExp_With_Default_Value_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan2002": map[string] interface{} {
				"vlanid":   "2002",
			},
		},
	}


	//Try to create er interface collding with vlan interface IP prefix
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "CFG_L2MC_TABLE|Vlan2002",
			map[string]string{
				"enabled":   "true",
				"query-interval": "9", //default query-max-response-time = 10
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	//Try to add second element
	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData)

	// Both query-interval and query-max-response-time have must expressions checking each other..
	// Order of evaluation is random
	expField, expValue := "query-interval", "9"
	if cvlErrInfo.Field == "query-max-response-time" {
		expField, expValue = "query-max-response-time", "10"
	}

	verifyErr(t, cvlErrInfo, CVLErrorInfo{
		ErrCode:          CVL_SEMANTIC_ERROR,
		TableName:        "CFG_L2MC_TABLE",
		Keys:             []string{"Vlan2002"},
		Field:            expField, // "query-interval" or "query-max-response-time"
		Value:            expValue, // "9" or "10"
		Msg:              mustExpressionErrMessage,
		ConstraintErrMsg: "Invalid IGMP Snooping query interval value.",
	})
}

func TestValidateEditConfig_MustExp_Chained_Predicate_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan701": map[string] interface{} {
				"vlanid":   "701",
				"members@": "Ethernet20",
			},
			"Vlan702": map[string] interface{} {
				"vlanid":   "702",
				"members@": "Ethernet20,Ethernet24,Ethernet28",
			},
			"Vlan703": map[string] interface{} {
				"vlanid":   "703",
				"members@": "Ethernet20",
			},
		},
		"VLAN_MEMBER" : map[string]interface{} {
			"Vlan701|Ethernet20": map[string] interface{} {
				"tagging_mode": "tagged",
			},
			"Vlan702|Ethernet20": map[string] interface{} {
				"tagging_mode": "tagged",
			},
			"Vlan702|Ethernet24": map[string] interface{} {
				"tagging_mode": "tagged",
			},
			"Vlan702|Ethernet28": map[string] interface{} {
				"tagging_mode": "tagged",
			},
			"Vlan703|Ethernet20": map[string] interface{} {
				"tagging_mode": "tagged",
			},
		},
		"INTERFACE" : map[string]interface{} {
			"Ethernet20|1.1.1.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Ethernet24|1.1.2.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Ethernet28|1.1.2.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Ethernet20|1.1.3.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
		},
		"VLAN_INTERFACE" : map[string]interface{} {
			"Vlan701|2.2.2.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Vlan701|2.2.3.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Vlan702|2.2.4.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Vlan702|2.2.5.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
			"Vlan703|2.2.6.0/32": map[string] interface{} {
				"NULL": "NULL",
			},
		},
	}


	//Try to create er interface collding with vlan interface IP prefix
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VLAN_INTERFACE|Vlan702|1.1.2.0/32",
			map[string]string{
				"NULL": "NULL",
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SEMANTIC_ERROR,
		//TableName:        "VLAN_INTERFACE",  <<< BUG: cvl returns VLAN_INTERFACE_IPADDR
		Keys:             []string{"Vlan702", "1.1.2.0/32"},
		Field:            "vlanName",
		Value:            "Vlan702",
		Msg:              mustExpressionErrMessage,
		ConstraintErrMsg: "Vlan and port being member of same vlan can't have same IP prefix.",
	})
}

func TestValidateEditConfig_MustExp_Within_Same_Table_Negative(t *testing.T) {
	//Try to create 
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "TAM_COLLECTOR_TABLE|Col10",
			map[string]string{
				"ipaddress-type": "ipv6", //Invalid ip address type
				"ipaddress":   "10.101.1.2",
			},
                },
        }

		verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
			ErrCode:          CVL_SEMANTIC_ERROR,
			TableName:        "TAM_COLLECTOR_TABLE",
			Keys:             []string{"Col10"},
			Field:            "ipaddress-type",
			Value:            "ipv6",
			Msg:              mustExpressionErrMessage,
			ConstraintErrMsg: "IP address and IP address type does not match.",
			ErrAppTag:        "ipaddres-type-mismatch",
		})
}

//Check if all data is fetched for xpath without predicate
func TestValidateEditConfig_MustExp_Without_Predicate_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan201": map[string] interface{} {
				"vlanid":   "201",
				"members@": "Ethernet4,Ethernet8,Ethernet12,Ethernet16",
			},
			"Vlan202": map[string] interface{} {
				"vlanid":   "202",
				"members@": "Ethernet4",
			},
		},
	}

	//Try to create 
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VLAN_INTERFACE|Vlan201",
			map[string]string{
				"NULL": "NULL",
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_MustExp_Non_Key_As_Predicate_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan201": map[string] interface{} {
				"vlanid":   "201",
			},
			"Vlan202": map[string] interface{} {
				"vlanid":   "202",
			},
		},
		"VXLAN_TUNNEL" : map[string]interface{} {
			"tun1": map[string] interface{} {
				"src_ip":   "10.10.1.2",
			},
		},
		"VXLAN_TUNNEL_MAP" : map[string]interface{} {
			"tun1|vmap1": map[string] interface{} {
				"vlan": "Vlan201",
				"vni": "300",
			},
		},
	}

	//Try to create 
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VXLAN_TUNNEL_MAP|tun1|vmap2",
			map[string]string{
				"vlan": "Vlan202",
				"vni": "300", //same VNI is not valid
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "VXLAN_TUNNEL_MAP",
		Keys:      []string{"tun1", "vmap2"},
		Field:     "vni",
		Value:     "300",
		Msg:       mustExpressionErrMessage,
		ErrAppTag: "not-unique-vni",
	})
}

func TestValidateEditConfig_MustExp_Non_Key_As_Predicate_In_External_Table_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan201": map[string] interface{} {
				"vlanid":   "201",
			},
			"Vlan202": map[string] interface{} {
				"vlanid":   "202",
			},
			"Vlan203": map[string] interface{} {
				"vlanid":   "203",
			},
		},
		"VXLAN_TUNNEL" : map[string]interface{} {
			"tun1": map[string] interface{} {
				"src_ip":   "10.10.1.2",
			},
		},
		"VXLAN_TUNNEL_MAP" : map[string]interface{} {
			"tun1|vmap1": map[string] interface{} {
				"vlan": "Vlan201",
				"vni": "301",
			},
			"tun1|vmap2": map[string] interface{} {
				"vlan": "Vlan202",
				"vni": "302",
			},
			"tun1|vmap3": map[string] interface{} {
				"vlan": "Vlan203",
				"vni": "303",
			},
		},
	}

	//Try to create 
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VRF|vrf101",
			map[string]string{
				"vni": "302",
			},
                },
        }

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_MustExp_Update_Leaf_List_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan202": map[string] interface{} {
				"vlanid":   "202",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"VLAN|Vlan202",
			map[string]string{
				"members@": "Ethernet4,Ethernet8",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_MustExp_Add_NULL(t *testing.T) {
	depDataMap := map[string]interface{} {
		"INTERFACE": map[string]interface{} {
			"Ethernet20": map[string] interface{} {
				"unnumbered": "Loopback1",
			},
		},
		"LOOPBACK_INTERFACE": map[string]interface{} {
			"Loopback1": map[string] interface{} {
				"NULL": "NULL",
			},
			"Loopback1|1.2.3.4/32": map[string] interface{} {
				"NULL": "NULL",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	delUnnumber := cvl.CVLEditConfigData{
		VType: cvl.VALIDATE_ALL,
		VOp:   cvl.OP_DELETE,
		Key:   "INTERFACE|Ethernet20",
		Data:  map[string]string{ "unnumbered": "" },
	}

	addNull := cvl.CVLEditConfigData{
		VType: cvl.VALIDATE_ALL,
		VOp:   cvl.OP_UPDATE,
		Key:   "INTERFACE|Ethernet20",
		Data:  map[string]string{ "NULL": "NULL" },
	}

	t.Run("before", testNullAdd(addNull, delUnnumber))
	t.Run("after", testNullAdd(delUnnumber, addNull))
}

func testNullAdd(data ...cvl.CVLEditConfigData) func(*testing.T) {
	return func(t *testing.T) {
		session, _ := cvl.ValidationSessOpen()
		defer cvl.ValidationSessClose(session)

		var cfgData []cvl.CVLEditConfigData
		for i, d := range data {
			cfgData = append(cfgData, d)
			errInfo, _ := session.ValidateEditConfig(cfgData)
			verifyErr(t, errInfo, Success)

			cfgData[i].VType = cvl.VALIDATE_NONE // dont validate for next op
		}
	}
}
