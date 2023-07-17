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
	"fmt"
	"testing"
	"github.com/Azure/sonic-mgmt-common/cvl"
)

// EditConfig(Create) with chained leafref from redis
func TestValidateEditConfig_Create_Chained_Leafref_DepData_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan100": map[string]interface{} {
				"members@": "Ethernet1",
				"vlanid": "100",
			},
		},
		"PORT" : map[string]interface{} {
			"Ethernet1" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
				"mtu": "9100",
			},
			"Ethernet2" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "85,86,87,89",
				"mtu": "9100",
			},
		},
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
				"ports@":"Ethernet2",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	cfgDataVlan := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN_MEMBER|Vlan100|Ethernet1",
			map[string]string {
				"tagging_mode" : "tagged",
			},
		},
	}

	errInfo, _  := cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, errInfo, Success)


	cfgDataAclRule :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string {
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

	errInfo, _  = cvSess.ValidateEditConfig(cfgDataAclRule)
	verifyErr(t, errInfo, Success)
}

func TestValidateEditConfig_Create_Leafref_To_NonKey_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"BGP_GLOBALS" : map[string]interface{} {
			"default": map[string] interface{} {
				"router_id": "1.1.1.1",
				"local_asn": "12338",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"DEVICE_METADATA|localhost",
			map[string]string {
				"vrf_name": "default",
				"bgp_asn": "12338",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Update_Leafref_To_NonKey_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"BGP_GLOBALS" : map[string]interface{} {
			"default": map[string] interface{} {
				"router_id": "1.1.1.1",
				"local_asn": "12338",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"DEVICE_METADATA|localhost",
			map[string]string {
				"vrf_name": "default",
				"bgp_asn": "17698",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
		TableName: "DEVICE_METADATA",
		Keys:      []string{"localhost"},
		//Field:     "bgp_asn", /* BUG: cvl does not fill field & value */
		//Value:     "17698",
		ConstraintErrMsg: "No instance found for '17698'",
		ErrAppTag:        "instance-required",
	})
}

func TestValidateEditConfig_Create_Leafref_Multi_Key_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL901": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
			"TestACL902": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
		"ACL_RULE" : map[string]interface{} {
			"TestACL901|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"DST_IP": "20.2.2.2/32",
			},
			"TestACL902|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.2/32",
				"DST_IP": "20.2.2.4/32",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"TAM_INT_IFA_FLOW_TABLE|Flow_1",
			map[string]string {
				"acl-table-name": "TestACL901",
				"acl-rule-name": "Rule1",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Leafref_Multi_Key_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL901": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
			"TestACL902": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
		"ACL_RULE" : map[string]interface{} {
			"TestACL902|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.2/32",
				"DST_IP": "20.2.2.4/32",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"TAM_INT_IFA_FLOW_TABLE|Flow_1",
			map[string]string {
				"acl-table-name": "TestACL901",
				"acl-rule-name": "Rule1", //This is not there in above depDataMap
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
		TableName: "TAM_INT_IFA_FLOW_TABLE",
		Keys:      []string{"Flow_1"},
		// Field:            "acl-rule-name", /* BUG: cvl does not fill field & value */
		// Value:            "Rule1",
		ConstraintErrMsg: "No instance found for 'Rule1'",
		ErrAppTag:        "instance-required",
	})
}

func TestValidateEditConfig_Create_Leafref_With_Other_DataType_In_Union_Positive(t *testing.T) {

	depDataMap := map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"STP_PORT|StpIntf10", //Non-leafref
			map[string]string{
				"enabled": "true",
				"edge_port": "true",
				"link_type": "shared",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Leafref_With_Other_DataType_In_Union_Negative(t *testing.T) {

	depDataMap := map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"STP_PORT|Test12", //Non-leafref
			map[string]string{
				"enabled": "true",
				"edge_port": "true",
				"link_type": "shared",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "STP_PORT",
		Keys:      []string{"Test12"},
		Field:     "ifname",
		Value:     "Test12",
		Msg:       invalidValueErrMessage,
	})
}

func TestValidateEditConfig_Create_Leafref_With_Other_DataType_In_Union_Non_Existing_Negative(t *testing.T) {

	depDataMap := map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"STP_PORT|Ethernet3999", //Correct PORT format but not existing
			map[string]string{
				"enabled": "true",
				"edge_port": "true",
				"link_type": "shared",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
		TableName:        "STP_PORT",
		Keys:             []string{"Ethernet3999"},
		ConstraintErrMsg: "No instance found for 'Ethernet3999'",
		ErrAppTag:        "instance-required",
	})
}

func TestValidateEditConfig_Delete_Leafref(t *testing.T) {
	depDataMap := map[string]interface{}{
		"PORTCHANNEL": map[string]interface{}{
			"PortChannel1": map[string]interface{}{
				"NULL": "NULL",
			},
			"PortChannel2": map[string]interface{}{
				"NULL": "NULL",
			},
			"PortChannel3": map[string]interface{}{
				"NULL": "NULL",
			},
			"PortChannel4": map[string]interface{}{
				"NULL": "NULL",
			},
		},
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"type":	  "L3",
				"stage":  "INGRESS",
				"ports@": "PortChannel1",
			},
			"TestACL2": map[string]interface{}{
				"type":	  "L3",
				"stage":  "INGRESS",
				"ports@": "PortChannel3,PortChannel4",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	t.Run("positive", deletePO(2, true))
	t.Run("negative", deletePO(1, false))
	t.Run("with_dep", deleteACLAndPO("TestACL1", "nil", 1, false, true))
	t.Run("with_dep_field", deleteACLAndPO("TestACL1", "", 1, false, true))
	t.Run("with_dep_update", deleteACLAndPO("TestACL2", "PortChannel4", 3, false, true))
	//t.Run("with_dep_bulk", deleteACLAndPO("TestACL1", 1, true, true))
}

func deletePO(poId int, expSuccess bool) func(*testing.T) {
	return func (t *testing.T) {
		session, _ := cvl.ValidationSessOpen()
		defer cvl.ValidationSessClose(session)
		validateDeletePO(t, session, nil, poId, expSuccess)
	}
}

func deleteACLAndPO(aclName, ports string, poId int, bulk, expSuccess bool) func(*testing.T) {
	return func (t *testing.T) {
		session, _ := cvl.ValidationSessOpen()
		defer cvl.ValidationSessClose(session)
		var cfgData []cvl.CVLEditConfigData

		cfgData = append(cfgData, cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			fmt.Sprintf("ACL_TABLE|%s", aclName),
			map[string]string{ },
		})

		if ports != "nil" {
			cfgData[0].Data["ports@"] = ports
			if ports != "" {
				cfgData[0].VOp = cvl.OP_UPDATE
			}
		}

		if !bulk {
			errInfo, status := session.ValidateEditConfig(cfgData)
			if status != cvl.CVL_SUCCESS {
				t.Errorf("ACL \"%s\" delete validation failed; %v", aclName, errInfo)
				return
			}

			cfgData[0].VType = cvl.VALIDATE_NONE
		}

		validateDeletePO(t, session, cfgData, poId, expSuccess)
	}
}

func validateDeletePO(t *testing.T, session *cvl.CVL, cfgData []cvl.CVLEditConfigData, poId int, expSuccess bool) {
	cfgData = append(cfgData, cvl.CVLEditConfigData{
		cvl.VALIDATE_ALL,
		cvl.OP_DELETE,
		fmt.Sprintf("PORTCHANNEL|PortChannel%d", poId),
		map[string]string{ },
	})

	errInfo, status := session.ValidateEditConfig(cfgData)
	if expSuccess && status != cvl.CVL_SUCCESS {
		t.Errorf("po%d delete validation failed; %v", poId, errInfo)
	}
	if !expSuccess && status == cvl.CVL_SUCCESS {
		t.Errorf("po%d delete validation should have failed", poId)
	}
}

