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

//go:build test
// +build test

package cvl_test

import (
	"testing"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
)

func extraChecksHelper(t *testing.T, dbData map[string]interface{}, data []CVLEditConfigData, hintKey string, hintExpectedVal bool) bool {
	if dbData != nil {
		setupTestData(t, dbData)
	}
	c := NewTestSession(t)
	c.StoreHint(hintKey, false)
	res, _ := c.ValidateEditConfig(data)
	verifyErr(t, res, Success)
	v, _ := c.LoadHint(hintKey)
	if bv, _ := v.(bool); bv != hintExpectedVal {
		t.Errorf("Test extra validation failed. expected: %v actual :%v", hintExpectedVal, bv)
		return true
	}
	return false

}

func TestValidateExtraChecks_Update_other_with_field_not_present_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet301": map[string]interface{}{
				"alias":        "hundredGigE1",
				"admin_status": "up",
			},
		},
	}
	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"PORT|Ethernet301",
			map[string]string{
				"admin_status": "down",
			},
			false,
		},
	}
	if extraChecksHelper(t, depDataMap, cfgData, "ExtraFieldValidationCalled", false) {
		t.Fatalf("Custom validation called when field is not present")
	}
}

func TestValidateExtraChecks_Update_other_with_field_present_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet300": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet301": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"PORT|Ethernet300",
			map[string]string{
				"admin_status": "down",
				"index":        "0",
			},
			false,
		},
	}
	if extraChecksHelper(t, depDataMap, cfgData, "ExtraFieldValidationCalled", true) {
		t.Fatalf("Custom validation is not called when field is present")
	}

}

func TestValidateExtraChecks_Update_field_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet300": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet301": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"PORT|Ethernet300",
			map[string]string{
				"diag_mode": "update test data",
			},
			false,
		},
	}
	if extraChecksHelper(t, depDataMap, cfgData, "ExtraFieldValidationCalled", true) {
		t.Fatalf("Custom validation  not called when updating field")
	}

}

func TestValidateExtraChecks_Create_field_Positive(t *testing.T) {

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_CREATE,
			"PORT|Ethernet302",
			map[string]string{
				"diag_mode": "test",
			},
			false,
		},
	}
	if extraChecksHelper(t, nil, cfgData, "ExtraFieldValidationCalled", true) {
		t.Fatalf("Custom validation  not called when creating field")
	}

}

func TestValidateExtraChecks_Delete_field_Positive(t *testing.T) {

	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet303": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet304": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_DELETE,
			"PORT|Ethernet303",
			map[string]string{},
			false,
		},
	}

	if extraChecksHelper(t, depDataMap, cfgData, "ExtraFieldValidationCalled", true) {
		t.Fatalf("Custom validation  not called when deleting parent with field present")
	}
}

func TestValidateExtraChecks_Delete_no_field_Positive(t *testing.T) {

	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet303": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet304": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}
	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_DELETE,
			"PORT|Ethernet304",
			map[string]string{},
			false,
		},
	}

	if extraChecksHelper(t, depDataMap, cfgData, "ExtraFieldValidationCalled", false) {
		t.Fatalf("Custom validation  called when deleting parent with field not present")
	}
}

func TestValidateExtraChecks_List_update_Positive(t *testing.T) {

	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet300": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet304": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"PORT|Ethernet300",
			map[string]string{
				"admin_status": "down",
				"index":        "0",
			},
			false,
		},
	}

	if extraChecksHelper(t, depDataMap, cfgData, "ListLevelValidationCalled", true) {
		t.Fatalf("List level Custom validation  not called when field present")
	}
}

func TestValidateExtraChecks_List_update_no_field_Positive(t *testing.T) {

	depDataMap := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet300": map[string]interface{}{
				"alias":        "hundredGigE1",
				"lanes":        "81,82,83,84",
				"diag_mode":    "test data",
				"admin_status": "up",
			},
			"Ethernet304": map[string]interface{}{
				"alias": "hundredGigE1",
				"lanes": "85,86,87,89",
			},
		},
	}

	cfgData := []cmn.CVLEditConfigData{
		cmn.CVLEditConfigData{
			cmn.VALIDATE_ALL,
			cmn.OP_UPDATE,
			"PORT|Ethernet304",
			map[string]string{
				"admin_status": "down",
				"index":        "0",
			},
			false,
		},
	}

	if extraChecksHelper(t, depDataMap, cfgData, "ListLevelValidationCalled", true) {
		t.Fatalf("List level Custom validation  not called when field not present")
	}
}
