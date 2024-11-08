////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

// Package path defines utilities to operate on translib path.
// TRanslib uses gnmi path syntax.
package path

import (
	"strings"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// HasWildcardKey checks if a gnmi.Path contains any wildcard key value ("*").
func HasWildcardKey(p *gnmi.Path) bool {
	if p == nil {
		return false
	}
	for _, e := range p.Elem {
		for _, v := range e.Key {
			if v == "*" {
				return true
			}
		}
	}

	return false
}

// HasWildcardAtKey checks if a gnmi.Path contains any wildcard key value ("*") at the given Elem index
func HasWildcardAtKey(p *gnmi.Path, index int) bool {
	if index >= Len(p) {
		return false
	}
	for _, kv := range p.Elem[index].Key {
		if kv == "*" {
			return true
		}
	}
	return false
}

// StrHasWildcardKey checks if gnmi path string has any wildcard key value.
// Equivalent of HasWildcardKey(ygot.StringToStructuredPath(p)); but
// checks for the wildcard without actually constructing a gnmi.Path.
func StrHasWildcardKey(p string) bool {
	var inKey, inValue, inEscape bool
	for i, c := range p {
		if inEscape {
			inEscape = false
			continue
		}
		switch c {
		case '\\':
			inEscape = true
		case '[':
			if !inValue {
				inKey = true
			}
		case '=':
			if inKey {
				if hasWildcardKeyAt(p, i+1) {
					return true
				}
				inValue = true
				inKey = false
			}
		case ']':
			if inValue {
				inValue = false
			}
		}
	}
	return false
}

func hasWildcardKeyAt(p string, index int) bool {
	len := len(p) - index
	if len > 2 && p[index] == '\\' {
		// As per gNMI spec, only \ and ] can be escaped.
		// Still handle escape of * to match ygot's parsig logic..
		len--
		index++
	}
	return len > 1 && p[index] == '*' && p[index+1] == ']'
}

// Len returns number of elements in a gnmi.Path
func Len(path *gnmi.Path) int {
	if path == nil {
		return 0
	}
	return len(path.Elem)
}

// Equals returns true if path elements and they key/values are equal.
func Equals(p1, p2 *gnmi.Path) bool {
	if n1 := Len(p1); n1 != Len(p2) {
		return false
	} else if n1 == 0 {
		return true // both are empty paths
	}
	for i, e1 := range p1.Elem {
		if !util.PathElemsEqual(e1, p2.Elem[i]) {
			return false
		}
	}
	return true
}

// New returns gnmi.Path object for a path string. Returns nil with an error
// info for bad path value.
func New(pathStr string) (*gnmi.Path, error) {
	if len(pathStr) == 0 || pathStr == "/" {
		return new(gnmi.Path), nil
	}
	return ygot.StringToStructuredPath(pathStr)
}

// Copy returns a new gnmi.Path using the elements of input gnmi.Path.
// Adding or removing elements from new path does not affect old one.
// But changes the element's contents will affect old path.
func Copy(path *gnmi.Path) *gnmi.Path {
	if path == nil {
		return nil
	}
	newElem := make([]*gnmi.PathElem, len(path.Elem))
	copy(newElem, path.Elem)
	return &gnmi.Path{Elem: newElem}
}

// Clone returns a clone of given gnmi.Path.
func Clone(path *gnmi.Path) *gnmi.Path {
	if path == nil {
		return nil
	}
	newElem := make([]*gnmi.PathElem, len(path.Elem))
	for i, ele := range path.Elem {
		newElem[i] = cloneElem(ele)
	}
	return &gnmi.Path{Elem: newElem}
}

func cloneElem(pe *gnmi.PathElem) *gnmi.PathElem {
	clone := &gnmi.PathElem{Name: pe.Name}
	if pe.Key != nil {
		clone.Key = make(map[string]string)
		for k, v := range pe.Key {
			clone.Key[k] = v
		}
	}
	return clone
}

// String returns gnmi.Path as a string. Returns empty string
// if the path is not valid.
func String(path *gnmi.Path) string {
	if path == nil {
		return "<nil>"
	}
	if s, err := ygot.PathToString(path); err == nil {
		return s
	}
	return "<invalid>"
}

// SubPath creates a new gnmi.Path having path elements from
// given start and end indices.
func SubPath(path *gnmi.Path, startIndex, endIndex int) *gnmi.Path {
	if n := Len(path); n == 0 || startIndex < 0 || endIndex > n {
		return &gnmi.Path{}
	}
	newElem := make([]*gnmi.PathElem, endIndex-startIndex)
	copy(newElem, path.Elem[startIndex:endIndex])
	return &gnmi.Path{Elem: newElem}
}

// AppendElems appends one or more path elements to a gnmi.Path
func AppendElems(path *gnmi.Path, elems ...string) *gnmi.Path {
	if path == nil {
		path = &gnmi.Path{}
	}
	for _, ele := range elems {
		pe := &gnmi.PathElem{Name: ele}
		path.Elem = append(path.Elem, pe)
	}
	return path
}

// AppendPathStr parses and appends a path string to the gnmi.Path object.
func AppendPathStr(path *gnmi.Path, suffix string) (*gnmi.Path, error) {
	if path == nil {
		path = &gnmi.Path{}
	}
	p, err := ygot.StringToStructuredPath(suffix)
	if err == nil {
		path.Elem = append(path.Elem, p.Elem...)
	}
	return path, err
}

// MergeElemsAt merges new path elements at given path position.
// Returns number of elements merged.
func MergeElemsAt(path *gnmi.Path, index int, elems ...string) int {
	if path == nil {
		return 0
	}
	size := len(path.Elem)
	for i, ele := range elems {
		if index >= size {
			newElem := &gnmi.PathElem{Name: ele}
			path.Elem = append(path.Elem, newElem)
		} else if elems[i] != path.Elem[index].Name {
			return i
		} else {
			index++
		}
	}

	return len(elems)
}

// GetElemAt returns the path element name at given index.
// Returns empty string if index is not valid.
func GetElemAt(path *gnmi.Path, index int) string {
	if n := Len(path); index < n {
		return path.Elem[index].Name
	}
	return ""
}

// HasElemAt checks if the path element at the given index has the string
// Returns true if string is found else false
func HasElemAt(path *gnmi.Path, index int, elem string) bool {
	if n := Len(path); index < n {
		return path.Elem[index].Name == elem
	}
	return false
}

// SetKeyAt adds/updates a key value to the path element at given index.
func SetKeyAt(path *gnmi.Path, index int, name, value string) {
	if n := Len(path); index < n {
		SetKey(path.Elem[index], name, value)
	}
}

// SetKey adds/updates a key value to the path element.
func SetKey(elem *gnmi.PathElem, name, value string) {
	if elem == nil {
		return
	}
	if elem.Key == nil {
		elem.Key = map[string]string{name: value}
	} else {
		elem.Key[name] = value
	}
}

// Matches checks if the path matches a template. Path must satisfy
// following conditions for a match:
//  1. Should be of equal length or longer than template.
//  2. Element names at each position should match.
//  3. Keys at each postion should match -- should have same set of key
//     names with same values. Wildcard value in the template matches
//     any value of corresponding key in the path. But wildcard value
//     in the path can only match with a wildcard value in template.
//
// Examples:
// "AA/BB/CC" matches "AA/BB"
// "AA/BB[x=1][y=1]" matches "AA/BB[x=1][y=*]"
// "AA/BB[x=1][y=*]" matches "AA/BB[x=1][y=*]"
// "AA/BB[x=1]" does not match "AA/BB[x=1][y=*]"
// "AA/BB[x=*]" does not match "AA/BB[x=1]"
func Matches(path *gnmi.Path, template *gnmi.Path) bool {
	if n := Len(path); n == 0 || n < Len(template) {
		return false
	}

	for i, t := range template.Elem {
		p := path.Elem[i]
		if t.Name != p.Name {
			return false
		}
		if len(t.Key) != len(p.Key) {
			return false
		}
		for k, tv := range t.Key {
			if pv, ok := p.Key[k]; !ok || (tv != "*" && tv != pv) {
				return false
			}
		}
	}

	return true
}

// RemoveModulePrefix strips out module prefixes from each element of a path.
func RemoveModulePrefix(path *gnmi.Path) {
	if Len(path) == 0 {
		return
	}
	for _, ele := range path.Elem {
		if k := strings.IndexByte(ele.Name, ':'); k != -1 {
			ele.Name = ele.Name[k+1:]
		}
	}
}

// SplitLastElem splits a string path into prefix & last path element
func SplitLastElem(p string) (prefix, last string) {
	var lastSlash int
	var inEscape, inKey bool
	p = strings.TrimSuffix(p, "/")
	for i, c := range p {
		switch {
		case inEscape:
			inEscape = false
		case c == '/' && !inKey:
			lastSlash = i
		case c == '[':
			inKey = true
		case c == ']':
			inKey = false
		case c == '\\':
			inEscape = true
		}
	}
	return p[:lastSlash], p[lastSlash:]
}
