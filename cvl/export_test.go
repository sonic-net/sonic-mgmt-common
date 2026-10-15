////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Dennis Lanov                                               //
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

// RequestCacheYangDataState returns the number of requestCache entries for a
// Redis table/key and how many of them still point to a YANG data node.
// Test-only: exported to the cvl_test package via this _test.go file.
func (c *CVL) RequestCacheYangDataState(redisTable, key string) (entries, withYangData int) {
	for _, r := range c.requestCache[redisTable][key] {
		entries++
		if r.YangData != nil {
			withYangData++
		}
	}
	return
}
