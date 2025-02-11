////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2025 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"strconv"
	"testing"

	"github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

func mockStpFeatureEnabled(t *testing.T, v bool) {
	t.Helper()
	applDb := util.NewDbClient("APPL_DB")
	if applDb == nil {
		t.Fatal("Failed to open APPL_DB client")
	}
	defer applDb.Close()
	res := applDb.HSet("SWITCH_TABLE:switch", "stp_supported", strconv.FormatBool(v))
	if res.Err() != nil {
		t.Fatalf("Failed to set stp_supported=%v in APPL_DB, err=%v", v, res.Err())
	}
}

func TestCustomValidation_success(t *testing.T) {
	mockStpFeatureEnabled(t, true)

	cfgData := []CVLEditConfigData{
		{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "STP|GLOBAL",
			Data:  map[string]string{"mode": "pvst"},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestCustomValidation_error(t *testing.T) {
	mockStpFeatureEnabled(t, false)
	defer mockStpFeatureEnabled(t, true)

	cfgData := []CVLEditConfigData{
		{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "STP|GLOBAL",
			Data:  map[string]string{"mode": "pvst"},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SEMANTIC_ERROR,
		TableName:        "STP",
		Keys:             []string{"STP", "GLOBAL"},
		ConstraintErrMsg: "Spanning-tree feature not enabled",
	})
}
