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

package common

// DBAccess is used by cvl and custom validation functions to access the ConfigDB.
// This allows cvl clients to plugin additional data source, like transaction cache,
// into cvl. Most of the interface methods mimic the go-redis APIs. It also defines
// Lookup and Count methods to perform advanced search by matching hash field values.
type DBAccess interface {
	Exists(key string) IntResult
	Keys(pattern string) StrSliceResult
	HGet(key, field string) StrResult
	HMGet(key string, fields ...string) SliceResult
	HGetAll(key string) StrMapResult
	Pipeline() PipeResult

	// Lookup entries using a Search criteria and return them in sonic db json format.
	// E.g, {"INTERFACE": {"Ethernet0": {"vrf", "Vrf1"}, "Ethernet0|1.2.3.4": {"NULL": "NULL"}}}
	// TODO fix the return value for not found case
	Lookup(s Search) JsonResult
	// Count entries using a Search criteria. Returns 0 if there are no matches.
	Count(s Search) IntResult
}

type IntResult interface {
	Result() (int64, error)
}

type StrResult interface {
	Result() (string, error)
}

type StrSliceResult interface {
	Result() ([]string, error)
}

type SliceResult interface {
	Result() ([]interface{}, error)
}

type StrMapResult interface {
	Result() (map[string]string, error)
}

type JsonResult interface {
	Result() (string, error) //TODO have it as map instead of string
}

type PipeResult interface {
	Keys(pattern string) StrSliceResult
	HGet(key, field string) StrResult
	HMGet(key string, fields ...string) SliceResult
	HGetAll(key string) StrMapResult
	Exec() error
	Close()
}

// Search criteria for advanced lookup. Initial filtering is done by matching the key Pattern.
// Results are further refined by applying Predicate, WithField and Limit constraints (optional)
type Search struct {
	// Pattern to match the keys from a redis table. Must contain a table name prefix.
	// E.g, `INTERFACE|Ethernet0` `INTERFACE|*` "INTERFACE|*|*"
	Pattern string
	// Predicate is a lua condition statement to inspect an entry's key and hash attributes.
	// It can use map variables 'k' and 'h' to access key & hash attributes.
	// E.g, `k['type'] == 'L3' and h['enabled'] == true`
	Predicate string
	// KeyNames must contain the key component names in order. Required only if Predicate uses 'k'.
	// E.g, if ["name","type"], a key "ACL|TEST|L3" will expand to lua map {name="TEST", type="L3"}
	KeyNames []string
	// WithField selects a entry only if it contains this hash field
	WithField string
	// Limit the results to maximum these number of entries
	Limit int
}
