////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/cvl"
)

const (
	CVL_SUCCESS                = cvl.CVL_SUCCESS
	CVL_SYNTAX_ERROR           = cvl.CVL_SYNTAX_ERROR
	CVL_SEMANTIC_ERROR         = cvl.CVL_SEMANTIC_ERROR
	CVL_SYNTAX_MAXIMUM_INVALID = cvl.CVL_SYNTAX_MAXIMUM_INVALID
	CVL_SYNTAX_MINIMUM_INVALID = cvl.CVL_SYNTAX_MINIMUM_INVALID
)

var Success = CVLErrorInfo{ErrCode: CVL_SUCCESS}

func compareErr(val, exp CVLErrorInfo) bool {
	return (val.ErrCode == exp.ErrCode) &&
		(len(exp.TableName) == 0 || val.TableName == exp.TableName) &&
		(len(exp.Keys) == 0 || reflect.DeepEqual(val.Keys, exp.Keys)) &&
		(len(exp.Field) == 0 || val.Field == exp.Field) &&
		(len(exp.Value) == 0 || val.Value == exp.Value) &&
		(len(exp.Msg) == 0 || val.Msg == exp.Msg) &&
		(len(exp.CVLErrDetails) == 0 || val.CVLErrDetails == exp.CVLErrDetails) &&
		(len(exp.ConstraintErrMsg) == 0 || val.ConstraintErrMsg == exp.ConstraintErrMsg) &&
		(len(exp.ErrAppTag) == 0 || val.ErrAppTag == exp.ErrAppTag)
}

func verifyErr(t *testing.T, res, exp CVLErrorInfo) {
	t.Helper()
	expandMessagePatterns(&exp)
	if !compareErr(res, exp) {
		t.Fatalf("CVLErrorInfo verification failed\nExpected: %#v\nReceived: %#v", exp, res)
	}
}

func verifyValidateEditConfig(t *testing.T, data []CVLEditConfigData, exp CVLErrorInfo) {
	t.Helper()
	c := NewTestSession(t)
	res, _ := c.ValidateEditConfig(data)
	verifyErr(t, res, exp)
}

func expandMessagePatterns(ex *CVLErrorInfo) {
	switch ex.Msg {
	case invalidValueErrMessage:
		ex.Msg = strings.ReplaceAll(ex.Msg, "{{field}}", ex.Field)
		ex.Msg = strings.ReplaceAll(ex.Msg, "{{value}}", ex.Value)
		ex.Msg = strings.TrimSuffix(ex.Msg, " \"\"") // if value is empty
	case unknownFieldErrMessage:
		ex.Msg = strings.ReplaceAll(ex.Msg, "{{field}}", ex.Field)
	}
}

const (
	invalidValueErrMessage   = "Field \"{{field}}\" has invalid value \"{{value}}\""
	unknownFieldErrMessage   = "Unknown field \"{{field}}\""
	genericValueErrMessage   = "Data validation failed"
	mustExpressionErrMessage = "Must expression validation failed"
	whenExpressionErrMessage = "When expression validation failed"
	instanceInUseErrMessage  = "Validation failed for Delete operation, given instance is in use"
)
