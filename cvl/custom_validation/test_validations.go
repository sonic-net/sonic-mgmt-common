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

package custom_validation

import (
	"context"
	"strings"

	"github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

func (t *CustomValidation) ValidateIfExtraFieldValidationCalled(
	vc *CustValidationCtxt) CVLErrorInfo {
	vc.SessCache.Hint["ExtraFieldValidationCalled"] = true
	return CVLErrorInfo{ErrCode: CVL_SUCCESS}

}

func (t *CustomValidation) ValidateIfListLevelValidationCalled(
	vc *CustValidationCtxt) CVLErrorInfo {
	vc.SessCache.Hint["ListLevelValidationCalled"] = true
	return CVLErrorInfo{ErrCode: CVL_SUCCESS}

}

func (t *CustomValidation) ValidateStpFeatureEnabled(vc *CustValidationCtxt) CVLErrorInfo {
	if vc.CurCfg.VOp == OP_DELETE {
		return CVLErrorInfo{}
	}
	applDb := util.NewDbClient("APPL_DB")
	if applDb == nil {
		return CVLErrorInfo{
			ErrCode:       CVL_INTERNAL_UNKNOWN,
			CVLErrDetails: "Database access failure",
		}
	}
	if enabled, _ := applDb.HGet(context.Background(), "SWITCH_TABLE:switch", "stp_supported").Result(); enabled != "true" {
		keys := strings.Split(vc.CurCfg.Key, "|")
		return CVLErrorInfo{
			ErrCode:          CVL_SEMANTIC_ERROR,
			TableName:        keys[0],
			Keys:             keys,
			ConstraintErrMsg: "Spanning-tree feature not enabled",
		}
	}
	return CVLErrorInfo{}
}
