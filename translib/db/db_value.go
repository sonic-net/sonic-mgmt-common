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

package db

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

// Value gives the fields as a map.
// (Eg: { Field: map[string]string { "type" : "l3v6", "ports" : "eth0" } } ).
type Value struct {
	Field map[string]string
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func (v Value) String() string {
	var str string
	for k, v1 := range v.Field {
		str = str + fmt.Sprintf("\"%s\": \"%s\"\n", k, v1)
	}

	return str
}

func (v Value) Copy() (rV Value) {
	rV = Value{Field: make(map[string]string, len(v.Field))}
	for k, v1 := range v.Field {
		rV.Field[k] = v1
	}
	return
}

// Compare4TxOps gives the Redis Ops/Notifs that are to be generated whilst
// going from a v Value to dst Value. Returns if HSet, and/or HDel needs
// to be performed
func (v Value) Compare4TxOps(dst Value) (isHSet, isHDel bool) {
	for _, fs := range v.Field {
		if fd, fdOk := dst.Field[fs]; !fdOk {
			isHDel = true
		} else if fd != fs {
			isHSet = true
		}
		if isHDel && isHSet {
			return
		}
	}

	for _, fd := range dst.Field {
		if _, fsOk := v.Field[fd]; !fsOk {
			isHSet = true
		}
		if isHSet {
			return
		}
	}
	return
}

//===== Functions for db.Value =====

func (v *Value) IsPopulated() bool {
	return len(v.Field) > 0
}

// Has function checks if a field exists.
func (v *Value) Has(name string) bool {
	_, flag := v.Field[name]
	return flag
}

// Get returns the value of a field. Returns empty string if the field
// does not exists. Use Has() function to check existance of field.
func (v *Value) Get(name string) string {
	return v.Field[name]
}

// Set function sets a string value for a field.
func (v *Value) Set(name, value string) {
	v.Field[name] = value
}

// GetInt returns value of a field as int. Returns 0 if the field does
// not exists. Returns an error if the field value is not a number.
func (v *Value) GetInt(name string) (int, error) {
	data, ok := v.Field[name]
	if ok {
		return strconv.Atoi(data)
	}
	return 0, nil
}

// SetInt sets an integer value for a field.
func (v *Value) SetInt(name string, value int) {
	v.Set(name, strconv.Itoa(value))
}

// GetList returns the value of a an array field. A "@" suffix is
// automatically appended to the field name if not present (as per
// swsssdk convention). Field value is split by comma and resulting
// slice is returned. Empty slice is returned if field not exists.
func (v *Value) GetList(name string) []string {
	var data string
	if strings.HasSuffix(name, "@") {
		data = v.Get(name)
	} else {
		data = v.Get(name + "@")
	}

	if len(data) == 0 {
		return []string{}
	}

	return strings.Split(data, ",")
}

// SetList function sets an list value to a field. Field name and
// value are formatted as per swsssdk conventions:
// - A "@" suffix is appended to key name
// - Field value is the comma separated string of list items
func (v *Value) SetList(name string, items []string) {
	if !strings.HasSuffix(name, "@") {
		name += "@"
	}

	if len(items) != 0 {
		data := strings.Join(items, ",")
		v.Set(name, data)
	} else {
		v.Remove(name)
	}
}

// Remove function removes a field from this Value.
func (v *Value) Remove(name string) {
	delete(v.Field, name)
}

// ContainsAll returns true if this value is a superset of the other value.
func (v *Value) ContainsAll(other *Value) bool {
	if len(v.Field) < len(other.Field) {
		return false
	}
	for oName, oVal := range other.Field {
		switch fVal, ok := v.Field[oName]; {
		case !ok: // field not present
			return false
		case fVal == oVal: // field values match
			continue
		case oName[len(oName)-1] != '@': // non leaf-list value mismatch
			return false
		case !leaflistEquals(fVal, oVal): // leaf-list value mismatch, ignoring order
			return false
		}
	}
	return true
}

// Equals returns true if this value contains same set of attributes as the other value
func (v *Value) Equals(other *Value) bool {
	return len(v.Field) == len(other.Field) && v.ContainsAll(other)
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// leaflistEquals compares two leaf-list values (comma separated instances)
// are same. Ignores the order of instances
func leaflistEquals(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	listA := strings.Split(a, ",")
	listB := strings.Split(b, ",")
	if len(listA) != len(listB) {
		return false
	}

	sort.Strings(listA)
	for _, s := range listB {
		if k := sort.SearchStrings(listA, s); k == len(listA) || listA[k] != s {
			return false
		}
	}

	return true
}
