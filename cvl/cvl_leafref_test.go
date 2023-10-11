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
	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
)

// EditConfig(Create) with chained leafref from redis
func TestValidateEditConfig_Create_Chained_Leafref_DepData_Positive(t *testing.T) {
	setupTestData(t, map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan100": map[string]interface{}{
				"members@": "Ethernet1",
				"vlanid":   "100",
			},
		},
		"PORT": map[string]interface{}{
			"Ethernet1": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "81,82,83,84",
				"mtu":   "9100",
			},
			"Ethernet2": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
				"mtu":   "9100",
			},
		},
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"stage":  "INGRESS",
				"type":   "L3",
				"ports@": "Ethernet2",
			},
		},
	})

	cvSess, _ := NewCvlSession()

	cfgDataVlan := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"VLAN_MEMBER|Vlan100|Ethernet1",
			map[string]string{
				"tagging_mode": "tagged",
			},
			false,
		},
	}

	_, err := cvSess.ValidateEditConfig(cfgDataVlan)

	if err != cvl.CVL_SUCCESS { //should succeed
		t.Errorf("Config Validation failed.")
		return
	}

	cfgDataAclRule := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":           "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
			false,
		},
	}

	_, err = cvSess.ValidateEditConfig(cfgDataAclRule)

	cvl.ValidationSessClose(cvSess)

	if err != cvl.CVL_SUCCESS { //should succeed
		t.Errorf("Config Validation failed.")
	}
}

func TestValidateEditConfig_Create_Leafref_To_NonKey_Positive(t *testing.T) {
	setupTestData(t, map[string]interface{}{
		"BGP_GLOBALS": map[string]interface{}{
			"default": map[string]interface{}{
				"router_id": "1.1.1.1",
				"local_asn": "12338",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"DEVICE_METADATA|localhost",
			map[string]string{
				"vrf_name": "default",
				"bgp_asn":  "12338",
			},
			false,
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Update_Leafref_To_NonKey_Negative(t *testing.T) {
	setupTestData(t, map[string]interface{}{
		"BGP_GLOBALS": map[string]interface{}{
			"default": map[string]interface{}{
				"router_id": "1.1.1.1",
				"local_asn": "12338",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"DEVICE_METADATA|localhost",
			map[string]string{
				"vrf_name": "default",
				"bgp_asn":  "17698",
			},
			false,
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
	setupTestData(t, map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL901": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "L3",
			},
			"TestACL902": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "L3",
			},
		},
		"ACL_RULE": map[string]interface{}{
			"TestACL901|Rule1": map[string]interface{}{
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE":       "IPV4",
				"SRC_IP":        "10.1.1.1/32",
				"DST_IP":        "20.2.2.2/32",
			},
			"TestACL902|Rule1": map[string]interface{}{
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE":       "IPV4",
				"SRC_IP":        "10.1.1.2/32",
				"DST_IP":        "20.2.2.4/32",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"TAM_INT_IFA_FLOW_TABLE|Flow_1",
			map[string]string{
				"acl-table-name": "TestACL901",
				"acl-rule-name":  "Rule1",
			},
			false,
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Leafref_Multi_Key_Negative(t *testing.T) {
	setupTestData(t, map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL901": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "L3",
			},
			"TestACL902": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "L3",
			},
		},
		"ACL_RULE": map[string]interface{}{
			"TestACL902|Rule1": map[string]interface{}{
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE":       "IPV4",
				"SRC_IP":        "10.1.1.2/32",
				"DST_IP":        "20.2.2.4/32",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"TAM_INT_IFA_FLOW_TABLE|Flow_1",
			map[string]string{
				"acl-table-name": "TestACL901",
				"acl-rule-name":  "Rule1", //This is not there in above depDataMap
			},
			false,
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

	setupTestData(t, map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"STP_PORT|StpIntf10", //Non-leafref
			map[string]string{
				"enabled":   "true",
				"edge_port": "true",
				"link_type": "shared",
			},
			false,
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Leafref_With_Other_DataType_In_Union_Negative(t *testing.T) {

	setupTestData(t, map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"STP_PORT|Test12", //Non-leafref
			map[string]string{
				"enabled":   "true",
				"edge_port": "true",
				"link_type": "shared",
			},
			false,
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

	setupTestData(t, map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "rpvst",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"STP_PORT|Ethernet3999", //Correct PORT format but not existing
			map[string]string{
				"enabled":   "true",
				"edge_port": "true",
				"link_type": "shared",
			},
			false,
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
	setupTestData(t, map[string]interface{}{
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
				"type":   "L3",
				"stage":  "INGRESS",
				"ports@": "PortChannel1",
			},
			"TestACL2": map[string]interface{}{
				"type":   "L3",
				"stage":  "INGRESS",
				"ports@": "PortChannel3,PortChannel4",
			},
		},
	})

	t.Run("positive", deletePO(2, true))
	t.Run("negative", deletePO(1, false))
	t.Run("with_dep", deleteACLAndPO("TestACL1", "nil", 1, false, true))
	t.Run("with_dep_field", deleteACLAndPO("TestACL1", "", 1, false, true))
	t.Run("with_dep_update", deleteACLAndPO("TestACL2", "PortChannel4", 3, false, true))
	//t.Run("with_dep_bulk", deleteACLAndPO("TestACL1", 1, true, true))
}

func deletePO(poId int, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		session, _ := NewCvlSession()
		defer cvl.ValidationSessClose(session)
		validateDeletePO(t, session, nil, poId, expSuccess)
	}
}

func deleteACLAndPO(aclName, ports string, poId int, bulk, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		session, _ := NewCvlSession()
		defer cvl.ValidationSessClose(session)
		var cfgData []cmn.CVLEditConfigData

		cfgData = append(cfgData, cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_DELETE,
			fmt.Sprintf("ACL_TABLE|%s", aclName),
			map[string]string{},
			false,
		})

		if ports != "nil" {
			cfgData[0].Data["ports@"] = ports
			if ports != "" {
				cfgData[0].VOp = cmn.OP_UPDATE
			}
		}

		if !bulk {
			errInfo, status := session.ValidateEditConfig(cfgData)
			if status != cvl.CVL_SUCCESS {
				t.Errorf("ACL \"%s\" delete validation failed; %v", aclName, errInfo)
				return
			}

			cfgData[0].VType = cmn.VALIDATE_NONE
		}

		validateDeletePO(t, session, cfgData, poId, expSuccess)
	}
}

func validateDeletePO(t *testing.T, session *cvl.CVL, cfgData []cmn.CVLEditConfigData, poId int, expSuccess bool) {
	cfgData = append(cfgData, cmn.CVLEditConfigData{
		cmn.VALIDATE_ALL,
		cmn.OP_DELETE,
		fmt.Sprintf("PORTCHANNEL|PortChannel%d", poId),
		map[string]string{},
		false,
	})

	errInfo, status := session.ValidateEditConfig(cfgData)
	if expSuccess && status != cvl.CVL_SUCCESS {
		t.Errorf("po%d delete validation failed; %v", poId, errInfo)
	}
	if !expSuccess && status == cvl.CVL_SUCCESS {
		t.Errorf("po%d delete validation should have failed", poId)
	}
}

func TestValidateEditConfig_Update_Leafref_Bulk(t *testing.T) {
	setupTestData(t, map[string]interface{}{
		"VRF": map[string]interface{}{
			"Vrf1": map[string]interface{}{
				"fallback": "false",
			},
		},
		"PORTCHANNEL": map[string]interface{}{
			"PortChannel1": map[string]interface{}{
				"mtu": "9100",
			},
			"PortChannel2": map[string]interface{}{
				"mtu": "9100",
			},
		},
		"PORTCHANNEL_INTERFACE": map[string]interface{}{
			"PortChannel1": map[string]interface{}{
				"vrf_name": "Vrf1",
			},
		},
	})

	deleteVrf1 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_DELETE,
		Key:   "VRF|Vrf1",
	}
	createVrf2 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_CREATE,
		Key:   "VRF|Vrf2",
		Data:  map[string]string{"fallback": "false"},
	}
	updateIntf1 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_UPDATE,
		Key:   "PORTCHANNEL_INTERFACE|PortChannel1",
		Data:  map[string]string{"vrf_name": "Vrf2"},
	}
	deleteIntf1 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_DELETE,
		Key:   "PORTCHANNEL_INTERFACE|PortChannel1",
	}
	deletePo1 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_DELETE,
		Key:   "PORTCHANNEL|PortChannel1",
	}
	createIntf2 := cmn.CVLEditConfigData{
		VType: cmn.VALIDATE_ALL,
		VOp:   cmn.OP_CREATE,
		Key:   "PORTCHANNEL_INTERFACE|PortChannel2",
		Data:  map[string]string{"vrf_name": "Vrf1"},
	}

	t.Run("cre_upd", validateEdit(createVrf2, updateIntf1))
	t.Run("upd_cre", validateEdit(updateIntf1, createVrf2))
	t.Run("del_cre_upd", validateEdit(deleteVrf1, createVrf2, updateIntf1))
	t.Run("cre_upd_del", validateEdit(createVrf2, updateIntf1, deleteVrf1))
	t.Run("upd_del_cre", validateEdit(updateIntf1, deleteVrf1, createVrf2))
	t.Run("del_all", validateEdit(deleteIntf1, deleteVrf1))
	t.Run("del_all_reverse", validateEdit(deleteVrf1, deletePo1, deleteIntf1))
	t.Run("del_add_neg", validateEditErr(cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING, deleteVrf1, deleteIntf1, createIntf2))
}

func validateEdit(data ...cmn.CVLEditConfigData) func(*testing.T) {
	return func(t *testing.T) {
		verifyValidateEditConfig(t, data, Success)
	}
}

func validateEditErr(exp cvl.CVLRetCode, data ...cmn.CVLEditConfigData) func(*testing.T) {
	return func(t *testing.T) {
		verifyValidateEditConfig(t, data, CVLErrorInfo{ErrCode: exp})
	}
}
