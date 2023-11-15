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

package ocbinds

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/maruel/natural"
)

// Values represents a sequence of reflect.Value objects.
type Values interface {
	sort.Interface
	At(i int) reflect.Value
}

// SortedMap returns map values sorted by its key. Integer keys (both signed and unsigned)
// are sorted by numerical order. String keys are sorted by natural order. Other keys are
// converted to string using fmt.Sprintf("%v", k) and sorted.
func SortedMap(v *reflect.Value) Values {
	var sv Values
	switch v.Type().Key().Kind() {
	case reflect.String:
		sMap := &strMap{}
		sMap.initFromStringMap(v)
		sv = sMap
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uMap := &uintMap{}
		uMap.initFromMap(v)
		sv = uMap
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iMap := &intMap{}
		iMap.initFromMap(v)
		sv = iMap
	default:
		sMap := &strMap{}
		sMap.initFromMap(v)
		sv = sMap
	}

	sort.Sort(sv)
	return sv
}

// strMap represents a map with string keys.
// Implements the Values interface.
type strMap struct {
	entries []strToValue
}

type strToValue struct {
	key string
	val reflect.Value
}

func (m *strMap) Len() int {
	return len(m.entries)
}

func (m *strMap) Less(i, j int) bool {
	return natural.Less(m.entries[i].key, m.entries[j].key)
}

func (m *strMap) Swap(i, j int) {
	m.entries[i], m.entries[j] = m.entries[j], m.entries[i]
}

func (m *strMap) At(i int) reflect.Value {
	return m.entries[i].val
}

func (m *strMap) initFromMap(v *reflect.Value) {
	m.entries = make([]strToValue, v.Len())
	for i, itr := 0, v.MapRange(); itr.Next(); i++ {
		m.entries[i].key = fmt.Sprintf("%v", itr.Key().Interface())
		m.entries[i].val = itr.Value()
	}
}

func (m *strMap) initFromStringMap(v *reflect.Value) {
	m.entries = make([]strToValue, v.Len())
	for i, itr := 0, v.MapRange(); itr.Next(); i++ {
		m.entries[i].key = itr.Key().String()
		m.entries[i].val = itr.Value()
	}
}

// uintMap represents a map with unsigned int keys (any size).
// Implements the Values interface.
type uintMap struct {
	entries []uintToValue
}

type uintToValue struct {
	key uint64
	val reflect.Value
}

func (m *uintMap) Len() int {
	return len(m.entries)
}

func (m *uintMap) Less(i, j int) bool {
	return m.entries[i].key < m.entries[j].key
}

func (m *uintMap) Swap(i, j int) {
	m.entries[i], m.entries[j] = m.entries[j], m.entries[i]
}

func (m *uintMap) At(i int) reflect.Value {
	return m.entries[i].val
}

func (m *uintMap) initFromMap(v *reflect.Value) {
	m.entries = make([]uintToValue, v.Len())
	for i, itr := 0, v.MapRange(); itr.Next(); i++ {
		m.entries[i].key = itr.Key().Uint()
		m.entries[i].val = itr.Value()
	}
}

// intMap represents a map with signed int keys (any size).
// Implements the Values interface.
type intMap struct {
	entries []intToValue
}

type intToValue struct {
	key int64
	val reflect.Value
}

func (m *intMap) Len() int {
	return len(m.entries)
}

func (m *intMap) Less(i, j int) bool {
	return m.entries[i].key < m.entries[j].key
}

func (m *intMap) Swap(i, j int) {
	m.entries[i], m.entries[j] = m.entries[j], m.entries[i]
}

func (m *intMap) At(i int) reflect.Value {
	return m.entries[i].val
}

func (m *intMap) initFromMap(v *reflect.Value) {
	m.entries = make([]intToValue, v.Len())
	for i, itr := 0, v.MapRange(); itr.Next(); i++ {
		m.entries[i].key = itr.Key().Int()
		m.entries[i].val = itr.Value()
	}
}
