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

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
)

func TestValidateEditConfig_When_Exp_In_Choice_Negative(t *testing.T) {

	setupTestData(t, map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	})

	cfgDataRule := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":           "IPV6",
				"SRC_IP":            "10.1.1.1/32", //Invalid field
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
			false,
		},
	}

	verifyValidateEditConfig(t, cfgDataRule, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "ACL_RULE",
		//Keys:      []string{"TestACL1", "Rule1"},  <<< BUG: cvl is not populating the key
		Field: "SRC_IP",
		Value: "10.1.1.1/32",
		Msg:   whenExpressionErrMessage,
	})
}

func TestValidateEditConfig_When_Exp_In_Leaf_Positive(t *testing.T) {

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
			"STP_PORT|Ethernet100",
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

func TestValidateEditConfig_When_Exp_In_Leaf_Negative(t *testing.T) {

	setupTestData(t, map[string]interface{}{
		"STP": map[string]interface{}{
			"GLOBAL": map[string]interface{}{
				"mode": "mstp",
			},
		},
	})

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"STP_PORT|Ethernet4",
			map[string]string{
				"enabled":   "true",
				"link_type": "shared",
			},
			false,
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "STP_PORT",
		Keys:      []string{"Ethernet4"},
		Field:     "link_type",
		Value:     "shared",
		Msg:       whenExpressionErrMessage,
	})
}
