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

package utils

import "fmt"

// Bits is a set of 8 bit flags.
type Bits uint8

// Set operations sets given bit flags into b
func (b *Bits) Set(flags Bits) {
	*b |= flags
}

// Unset operations clears given bit flags from b
func (b *Bits) Unset(flags Bits) {
	*b &^= flags
}

// Reset operations clears all bits from b
func (b *Bits) Reset() {
	*b = 0
}

// Empty returns true if no bits are set in b
func (b Bits) Empty() bool {
	return b != 0
}

// Has returns true if all flags are present in b
func (b Bits) Has(flags Bits) bool {
	return (b & flags) == flags
}

// HasAny returns true if any of the flags is present in b
func (b Bits) HasAny(flags Bits) bool {
	return (b & flags) != 0
}

func (b Bits) String() string {
	return fmt.Sprintf("%08b", b)
}
