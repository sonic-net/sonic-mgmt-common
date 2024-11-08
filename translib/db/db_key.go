////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
)

// Key is the db key components without table name prefix.
// (Eg: { Comp : [] string { "acl1", "rule1" } } ).
type Key struct {
	Comp []string
}

// NewKey returns a Key object with given key components
func NewKey(comps ...string) *Key {
	return &Key{Comp: comps}
}

// Copy returns a (deep) copy of the given Key
func (k Key) Copy() (rK Key) {
	rK = Key{Comp: make([]string, len(k.Comp))}
	copy(rK.Comp, k.Comp)
	return
}

func (k Key) String() string {
	return fmt.Sprintf("{Comp: %v}", k.Comp)
}

// Len returns number of components in the Key
func (k Key) Len() int {
	return len(k.Comp)
}

// Get returns the key component at given index
func (k Key) Get(index int) string {
	return k.Comp[index]
}

// IsPattern checks if the key has redis glob-style pattern.
// Supports only '*' and '?' wildcards.
func (k Key) IsPattern() bool {
	for _, s := range k.Comp {
		n := len(s)
		for i := 0; i < n; i++ {
			switch s[i] {
			case '\\':
				i++
			case '*', '?':
				return true
			}
		}
	}
	return false
}

// Equals checks if db key k equals to the other key.
func (k Key) Equals(other Key) bool {
	if k.Len() != other.Len() {
		return false
	}
	for i, c := range k.Comp {
		if c != other.Comp[i] {
			return false
		}
	}
	return true
}

// Matches checks if db key k matches a key pattern.
func (k Key) Matches(pattern Key) bool {
	if k.Len() != pattern.Len() {
		return false
	}
	for i, c := range k.Comp {
		if pattern.Comp[i] == "*" {
			continue
		}
		if !patternMatch(c, 0, pattern.Comp[i], 0) {
			return false
		}
	}
	return true
}

// IsAllKeyPattern returns true if it is an all key wildcard pattern.
// (i.e. A key with a single component "*")
func (k *Key) IsAllKeyPattern() bool {
	if (len(k.Comp) == 1) && (k.Comp[0] == "*") {
		return true
	}
	return false
}

// patternMatch checks if the value matches a key pattern.
// vIndex and pIndex are start positions of value and pattern strings to match.
// Mimics redis pattern matcher - i.e, glob like pattern matcher which
// matches '/' against wildcard.
// Supports '*' and '?' wildcards with '\' as the escape character.
// '*' matches any char sequence or none; '?' matches exactly one char.
// Character classes are not supported (redis supports it).
func patternMatch(value string, vIndex int, pattern string, pIndex int) bool {
	for pIndex < len(pattern) {
		switch pattern[pIndex] {
		case '*':
			// Skip successive *'s in the pattern
			pIndex++
			for pIndex < len(pattern) && pattern[pIndex] == '*' {
				pIndex++
			}
			// Pattern ends with *. Its a match always
			if pIndex == len(pattern) {
				return true
			}
			// Try to match remaining pattern with every value substring
			for ; vIndex < len(value); vIndex++ {
				if patternMatch(value, vIndex, pattern, pIndex) {
					return true
				}
			}
			// No match for remaining pattern
			return false

		case '?':
			// Accept any char.. there should be at least one
			if vIndex >= len(value) {
				return false
			}
			vIndex++
			pIndex++

		case '\\':
			// Do not treat \ as escape char if it is the last pattern char.
			// Redis commands behave this way.
			if pIndex+1 < len(pattern) {
				pIndex++
			}
			fallthrough

		default:
			if vIndex >= len(value) || pattern[pIndex] != value[vIndex] {
				return false
			}
			vIndex++
			pIndex++
		}
	}

	// All pattern chars have been compared.
	// It is a match if all value chars have been exhausted too.
	return (vIndex == len(value))
}
