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
	"unsafe"
)

/*
#cgo LDFLAGS: -lyang
#include <libyang/libyang.h>
#include <libyang/tree_data.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

const char *golys_module_from_prefix(const struct lys_module *module, const char *prefix)
{
	const struct lysp_module *mod;
	LY_ARRAY_COUNT_TYPE u;

	if (module == NULL || prefix == NULL) {
			return NULL;
	}

	mod = module->parsed;
	LY_ARRAY_FOR(mod->imports, u) {
		if (strcmp(mod->imports[u].prefix, prefix) == 0) {
			return mod->imports[u].name;
		}
	}

	return NULL;
}
*/
import "C"

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

/* Extracting error details requires knowledge of text strings within libyang.
 * Enumeration of known text strings (extracted from libyang 3.7.8) below:
 *
 * apptag:
 *  - missing-choice
 *  - too-few-elements
 *  - too-many-elements
 *  - data-not-unique
 *  - must-violation
 *  - not-left
 *  - instance-required
 *
 * error messages:
 *  NOTE: Mappings not listed here are in ly_common.h as defines prefixed with LY_VCODE_
 *  - "Invalid non-empty-encoded %s value \"%.*s\"."
 *  - "Invalid non-boolean-encoded %s value \"%.*s\"."
 *  - "Invalid non-string-encoded %s value \"%.*s\"."
 *  - "Invalid non-num64-encoded %s value \"%.*s\"."
 *  - "Invalid non-number-encoded %s value \"%.*s\"."
 *  - "Invalid %zu. character of decimal64 value \"%.*s\"."
 *  - "Invalid empty decimal64 value."
 *  - "Invalid empty value length %zu."
 *  - "Invalid type %s empty value."
 *  - "Invalid type %s value \"%.*s\"."
 *  - "Invalid %" PRIu64 ". character of decimal64 value \"%.*s\"."
 *  - "Invalid type %s value \"%.*s\"."
 *  - "Invalid enumeration value \"%.*s\"."
 *  - "Invalid enumeration value % " PRIi32 "."
 *  - "Invalid boolean value \"%.*s\"."
 *  - "Invalid bit \"%.*s\"."
 *  - "Invalid Base64 character 0x%x."
 *  - "Invalid Base64 character '%c'."
 *  - "Invalid character 0x%hhx."
 *  - "Invalid node-instance-identifier \"%.*s\" value - semantic error."
 *  - "Invalid node-instance-identifier \"%.*s\" value - syntax error."
 *  - "Invalid first character '%c', list key predicates expected."
 *  - "Invalid LYB boolean value size %zu (expected 1)."
 *  - "Invalid LYB date-and-time character '%c' (expected a digit)."
 *  - "Invalid LYB date-and-time value size %zu (expected at least 8)."
 *  - "Invalid LYB bits value size %zu (expected %zu)."
 *  - "Invalid LYB union value size %zu (expected at least 4)."
 *  - "Invalid LYB union type index %" PRIu64 " (type count %" LY_PRI_ARRAY_COUNT_TYPE ")."
 *  - "Invalid LYB signed integer value size %zu (expected %zu)."
 *  - "Invalid LYB unsigned integer value size %zu (expected %zu)."
 *  - "Invalid LYB enumeration value size %zu (expected 4)."
 *  - "Invalid LYB decimal64 value size %zu (expected 8)."
 *  - "Invalid LYB ipv6-address-no-zone value size %zu (expected 16)."
 *  - "Invalid LYB ipv6-address zone character 0x%x."
 *  - "Invalid LYB ipv6-address value size %zu (expected at least 16)."
 *  - "Invalid LYB ipv6-prefix value size %zu (expected %d)."
 *  - "Invalid LYB ipv6-prefix prefix length %" PRIu8 "."
 *  - "Invalid LYB ipv4-address value size %zu (expected at least 4)."
 *  - "Invalid LYB ipv4-address zone character 0x%x."
 *  - "Invalid LYB ipv4-address-no-zone value size %zu (expected 4)."
 *  - "Invalid LYB ipv4-prefix value size %zu (expected %d)."
 *  - "Invalid identityref \"%.*s\" value - identity is disabled by if-feature."
 *  - "Invalid identityref \"%.*s\" value - identity found in non-implemented module \"%s\"."
 *  - "Invalid identityref \"%.*s\" value - identity not derived from all the bases %s."
 *  - "Invalid identityref \"%.*s\" value - identity not derived from the base %s."
 *  - "Invalid identityref \"%.*s\" value - identity not found in module \"%s\"."
 *  - "Invalid identityref \"%.*s\" value - unable to map prefix to YANG schema."
 *  - "Invalid empty identityref value."
 *  - "Unsatisfied range - value \"%.*s\" is out of the allowed range."
 *  - "Unsatisfied length - string \"%.*s\" length is not allowed."
 *  - "Unsatisfied pattern - \"%.*s\" does not conform to %s\"%s\"."
 *  - "Value \"%.*s\" of decimal64 type exceeds defined number (%u) of fraction digits."
 *  - "Value \"%.*s\" is out of type %s min/max bounds."
 *  - "A %s definition \"%s\" is not allowed to reference %s value \"%s\"."
 *  - "Failed to convert IPv6 address \"%s\"."
 *  - "Failed to convert IPv4 address \"%s\"."
 *  - "Failed to resolve prefix \"%.*s\"."
 *  - "Failed to convert IPv4 address \"%s\"."
 *  - "Duplicate bit \"%s\"."
 *  - "Newlines are expected every 64 Base64 characters."
 *  - "Base64 encoded value length must be divisible by 4."
 *  - "No memory."
 */

// Regex patterns to extract target node name and value from libyang error message.
var (
	lyBadValue    = regexp.MustCompile(`[Vv]alue "([^"]*)"[ \.]`)
	lyUnsatisfied = regexp.MustCompile(`Unsatisfied (?:pattern|range|length) - (?:value |string |)"([^"]*)" `)
	lyElemPrefix  = regexp.MustCompile(`[Ee]lement "([^"]*)"`)
	lyElemSuffix  = regexp.MustCompile(`"([^"]*)" element`)
	lyUnknownElem = regexp.MustCompile(`Term node "([^"]*)" not found\.`)
	lyMandatory   = regexp.MustCompile(`Mandatory node "([^"]*)" instance does not exist\.`)
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

// This function takes a when, must, or leafref path in its original form as
// written in the YANG schema files, and converts it into its fully qualified
// format.  The format in the YANG schema uses import prefixes, so we have to
// replace each import prefix with the fully qualified module name.
//
// Libyang1 would do this for us automatically, but in Libyang3 we have to do
// this conversion ourselves.
//
// In this implementation we are cheating a bit.  It would be quite a bit of
// effort to tokenize everything, especially when and must clauses, so we rely
// on the fact that the namespace can only contain alphanumeric or hypen
// characters and will always begin with either "/" or "[" and end with ":".
// So we extract each prefix using a regex, perform a lookup to determine the
// module name then replace each instance of the prefix with the module name,
// in a loop until all matched prefixes have been processed.
//
// Examples:
//
//	Leafref:
//	  /po:sonic-portchannel/po:PORTCHANNEL/po:PORTCHANNEL_LIST/po:name
//	     ->
//	  /sonic-portchannel:sonic-portchannel/sonic-portchannel:PORTCHANNEL/sonic-portchannel:PORTCHANNEL_LIST/sonic-portchannel:name
//
//	Must:
//	  (/cmn:operation/cmn:operation != 'CREATE') or
//	  (count(/si:sonic-interface/si:INTERFACE/si:INTERFACE_IPADDR_LIST[si:ip_prefix=current()/../ip_prefix] [si:portname=(/svl:sonic-vlan/svl:VLAN_MEMBER/svl:VLAN_MEMBER_LIST[svl:name=current()]/svl:ifname)]) = 0)
//	     ->
//	  (/sonic-common:operation/sonic-common:operation != 'CREATE') or
//	  (count(/sonic-interface:sonic-interface/sonic-interface:INTERFACE/sonic-interface:INTERFACE_IPADDR_LIST[sonic-interface:ip_prefix=current()/../ip_prefix] [sonic-interface:portname=(/sonic-vlan:sonic-vlan/sonic-vlan:VLAN_MEMBER/sonic-vlan:VLAN_MEMBER_LIST[sonic-vlan:name=current()]/sonic-vlan:ifname)]) = 0)
var lyXPathPrefix = regexp.MustCompile("[[/][A-Za-z0-9_-]+:")

func rewriteXPathPrefix(module *YParserModule, xpath string) string {
	hasPrefix := make(map[string]bool)
	prefixes := lyXPathPrefix.FindAllString(xpath, -1)

	for _, prefix := range prefixes {
		if _, ok := hasPrefix[prefix]; ok {
			continue
		}
		hasPrefix[prefix] = true

		// strip / and : surrounding prefix
		prefix = prefix[1 : len(prefix)-1]

		// Dereference it
		Cprefix := C.CString(prefix)
		defer C.free(unsafe.Pointer(Cprefix))
		module_name := C.golys_module_from_prefix((*C.struct_lys_module)(module), Cprefix)
		if module_name == nil {
			continue
		}
		xpath = strings.ReplaceAll(xpath, "/"+prefix+":", "/"+C.GoString(module_name)+":")
		xpath = strings.ReplaceAll(xpath, "["+prefix+":", "["+C.GoString(module_name)+":")
	}

	return xpath
}
