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
	"strings"
	"testing"
)

func llistMinElemErr(key, field string) CVLErrorInfo {
	return CVLErrorInfo{
		ErrCode:       CVL_SYNTAX_MINIMUM_INVALID,
		TableName:     strings.Split(key, "|")[0],
		Keys:          strings.Split(key, "|")[1:],
		Field:         strings.TrimSuffix(field, "@"),
		CVLErrDetails: "min-elements constraint not honored",
	}
}

func llistMaxElemErr(key, field string) CVLErrorInfo {
	return CVLErrorInfo{
		ErrCode:       CVL_SYNTAX_MAXIMUM_INVALID,
		TableName:     strings.Split(key, "|")[0],
		Keys:          strings.Split(key, "|")[1:],
		Field:         strings.TrimSuffix(field, "@"),
		CVLErrDetails: "max-elements constraint not honored",
	}
}

// Test min-elements and max-elements constraint validations on leaf-list fields
func TestValidateEditConfig_Leaflist_MinMax(t *testing.T) {

	setupTestData(t, map[string]interface{}{
		"TEST_LEAFLIST": map[string]interface{}{
			"entry1": map[string]interface{}{
				"without-minmax@": "aaa,bbb,ccc", // no constraints
				"with-min0@":      "xxx,yyy",     // min-elements 0
				"with-min1-max2@": "foo,bar",     // min-elements 1, max-elements 2
				"with-min4@":      "1,2,3,4,5,6", // min-elements 4
			},
		}})

	newKey := "TEST_LEAFLIST|new"
	oldKey := "TEST_LEAFLIST|entry1"

	// Create test cases

	t.Run("create_all", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"without-minmax@": "hello,world",
				"with-min0@":      "none",
				"with-min1-max2@": "foo,bar",
				"with-min4@":      "one,two,three,four,five",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("create_without_min0", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"with-min1-max2@": "foo",
				"with-min4@":      "one,two,three,four",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("create_without_min1", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"with-min4@": "one,two,three,four",
			}}})
		verifyErr(tt, res, llistMinElemErr(newKey, "with-min1-max2"))
	})

	t.Run("create_without_min4", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"with-min1-max2@": "foo",
			}}})
		verifyErr(tt, res, llistMinElemErr(newKey, "with-min4"))
	})

	t.Run("create_more_than_max", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"with-min1-max2@": "hello,world,!!",
				"with-min4@":      "one,two,three,four",
			}}})
		verifyErr(tt, res, llistMaxElemErr(newKey, "with-min1-max2"))
	})

	t.Run("create_less_than_min", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   newKey,
			Data: map[string]string{
				"with-min1-max2@": "hello,world",
				"with-min4@":      "one,two",
			}}})
		verifyErr(tt, res, llistMinElemErr(newKey, "with-min4"))
	})

	// update cases

	t.Run("update_without_minmax", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"without-minmax@": "00,11,22,33,44,55,66,77,88,99",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("update_min0", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"without-minmax@": "00,11,22,33,44,55,66,77,88,99",
				"with-min0@":      "foo,bar",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("update_to_min1", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"without-minmax@": "00,11,22,33,44,55,66,77,88,99",
				"with-min1-max2@": "oneinstance",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("update_to_min4", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"with-min4@": "i1,i2,i3,i4",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("update_to_max", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"without-minmax@": "00,11,22,33,44,55,66,77,88,99",
				"with-min1-max2@": "two,instances",
			}}})
		verifyErr(tt, res, Success)
	})

	t.Run("update_more_than_max", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"without-minmax@": "00,11,22,33,44,55,66,77,88,99",
				"with-min1-max2@": "more,than,two,instances",
			}}})
		verifyErr(tt, res, llistMaxElemErr(oldKey, "with-min1-max2"))
	})

	t.Run("update_less_than_min", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data: map[string]string{
				"with-min1-max2@": "i1,i2",
				"with-min4@":      "j1,j2",
			}}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min4"))
	})

	t.Run("update_to_empty1", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data:  map[string]string{"with-min1-max2@": ""},
		}})
		// NOTE: cvl treats empty string as a single leaf-list instance.
		// Hence, this test case would succeed (min-elem 1 is honored).
		verifyErr(tt, res, Success)
	})

	t.Run("update_to_empty4", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_UPDATE,
			Key:   oldKey,
			Data:  map[string]string{"with-min4@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min4"))
	})

	// delete cases

	t.Run("delete_without_minmax", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_DELETE,
			Key:   oldKey,
			Data:  map[string]string{"without-minmax@": ""},
		}})
		verifyErr(tt, res, Success)
	})

	t.Run("delete_min0", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_DELETE,
			Key:   oldKey,
			Data:  map[string]string{"with-min0@": ""},
		}})
		verifyErr(tt, res, Success)
	})

	t.Run("delete_min1", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_DELETE,
			Key:   oldKey,
			Data:  map[string]string{"with-min1-max2@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min1-max2"))
	})

	t.Run("delete_min4", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_DELETE,
			Key:   oldKey,
			Data:  map[string]string{"with-min0@": "", "with-min4@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min4"))
	})

	// replace cases

	t.Run("replace_no_constraints2", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"without-minmax@": ""},
		}})
		verifyErr(tt, res, Success)
	})

	t.Run("replace_remove_min0", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{"without-minmax@": ""},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"with-min0@": ""},
		}})
		verifyErr(tt, res, Success)
	})

	t.Run("replace_remove_min1", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"with-min1-max2@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min1-max2"))
	})

	t.Run("replace_remove_min4", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{"with-min1-max2@": "11,22"},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"with-min4@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min4"))
	})

	t.Run("replace_set_more_than_max", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{"with-min1-max2@": "11,22,33"},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"with-min0@": ""},
		}})
		verifyErr(tt, res, llistMaxElemErr(oldKey, "with-min1-max2"))
	})

	t.Run("replace_set_less_than_min", func(tt *testing.T) {
		c := NewTestSession(tt)
		res, _ := c.ValidateEditConfig([]CVLEditConfigData{{
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_UPDATE,
			Key:       oldKey,
			Data:      map[string]string{"with-min4@": "11,22,33"},
		}, {
			ReplaceOp: true,
			VType:     VALIDATE_ALL,
			VOp:       OP_DELETE,
			Key:       oldKey,
			Data:      map[string]string{"with-min0@": ""},
		}})
		verifyErr(tt, res, llistMinElemErr(oldKey, "with-min4"))
	})

}
