////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package cvl

import (
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

// StoreHint saves hints which are passed to  the Custom Validation Callbacks
// Caller guarantees that no other CRUD ops are in progress
// key == nil: Clear all Hints
func (c *CVL) StoreHint(key string, value interface{}) CVLRetCode {

	CVL_LOG(INFO_DEBUG, "StoreHint() %v: %v", key, value)

	if c == nil {
		return CVL_SUCCESS
	}

	if len(key) == 0 {
		for k := range c.custvCache.Hint {
			delete(c.custvCache.Hint, k)
		}
	} else if value == nil {
		delete(c.custvCache.Hint, key)
	} else {
		c.custvCache.Hint[key] = value
	}

	return CVL_SUCCESS
}

// LoadHint is used only for Go UT
func (c *CVL) LoadHint(key string) (interface{}, bool) {
	if c == nil {
		return nil, false
	}

	value, ok := c.custvCache.Hint[key]

	CVL_LOG(INFO_DEBUG, "LoadHint() %v: %v", key, value)

	return value, ok
}
