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

package yparser

import (
	"regexp"
	"strings"
)

// parseLyPath parses a libyang formatted path; extracts table name, key components
// and field name from it. Path should represent a sonic yang node. Path elements
// are interpreted as follows:
//
//	/sonic-port:sonic-port/PORT/PORT_LIST[name='Ethernet0']/mtu
//	                       ↑                    ↑           ↑
//	                    table name          key comp     field name
//
// Special case :- single elem path is considered as field name.
// Libyang does not report the full path when a leaf creation fails. So, 1 elem path
// can either represent a top container or a leaf node. But it cannot be the former
// due to our usage.
func parseLyPath(p string) (table string, keys []string, field string) {
	var elem []lyPathElem
	for i := 0; i < len(p); {
		pe, n := parseFirstElem(p[i:])
		elem = append(elem, pe)
		i += n
	}
	// elem[1] is the container representing table name
	if len(elem) > 1 {
		table = elem[1].name
	}
	// elem[2] is the list node, with optional key predicates
	if len(elem) > 2 {
		keys = elem[2].vals
	}
	// Last elem represents db field. Single elem path is also considered as db field!
	if len(elem) == 1 || len(elem) > 3 {
		field = elem[len(elem)-1].name
	}
	return
}

// parseFirstElem parses the first element from a libyang path.
func parseFirstElem(p string) (elem lyPathElem, n int) {
	i, size := 0, len(p)
	if n < size && p[n] == '/' {
		i, n = 1, 1 // skip the optional '/' prefix
	}
	for n < size && p[n] != '[' && p[n] != '/' {
		n++ // locate next predicate or path sep, whichever comes first
	}
	// Collect the element name..
	elem.name = p[i:n]
	if k := strings.IndexByte(elem.name, ':'); k >= 0 {
		elem.name = elem.name[k+1:]
	}
	// Parse each predicate section `[name='value']` and collect the values
	for n < size && p[n] == '[' {
		m := lyPredicatePattern.FindStringSubmatch(p[n:])
		if len(m) != 3 {
			elem.vals = nil
			return elem, size // invalid path, skip everything
		}
		if v := m[2]; len(v) >= 2 { // remove surrounding quotes
			elem.vals = append(elem.vals, v[1:len(v)-1])
		}
		n += len(m[0])
	}
	// Next char must be a path separator
	if n < size && p[n] != '/' {
		n = size // invalid path, skip everything
	}
	return
}

// lyPathElem represents a path element
type lyPathElem struct {
	name string   // element name
	vals []string // key values
}

// lyPredicatePattern is the regex to match a libyang formatted predicate section.
// Expects either `[name='value']` or `[name="value"]` syntax.
// Match group 1 will be the name and group 2 will be the quoted value.
// Libyang uses single quotes if the value does not contain any single quotes;
// otherwise uses double quotes. But, libyang does not insert escape chars when
// value contains both single and double quotes, resulting in a vague format.
// Such predicates cannot be parsed.
var lyPredicatePattern = regexp.MustCompile(`^\[([^=]*)=('[^']*'|"[^"]*")]`)

// Regex patterns to extract target node name and value from libyang error message.
// Example messages:
// - Invalid value "9999999" in "vlanid" element
// - Missing required element "vlanid" in "VLAN_LIST"
// - Value "xyz" does not satisfy the constraint "Ethernet([1-3][0-9]{3}|[1-9][0-9]{2}|[1-9][0-9]|[0-9])" (range, length, or pattern)
// - Leafref "/sonic-port:sonic-port/sonic-port:PORT/sonic-port:ifname" of value "Ethernet668" points to a non-existing leaf
// - Failed to find "extra" as a sibling to "sonic-acl:aclname"
var (
	lyBadValue    = regexp.MustCompile(`[Vv]alue "([^"]*)" `)
	lyElemPrefix  = regexp.MustCompile(`[Ee]lement "([^"]*)"`)
	lyElemSuffix  = regexp.MustCompile(`"([^"]*)" element`)
	lyUnknownElem = regexp.MustCompile(`Failed to find "([^"]*)" `)
)

// parseLyMessage matches a libyang returned log message using given
// regex patterns and returns the first matched group.
func parseLyMessage(s string, regex ...*regexp.Regexp) string {
	for _, ex := range regex {
		if m := ex.FindStringSubmatch(s); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
