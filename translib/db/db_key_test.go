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

import "testing"

func TestIsPattern(t *testing.T) {
	t.Run("none1", testNotPattern("aaa"))
	t.Run("none5", testNotPattern("aaa", "bbb", "ccc", "ddd", "eee"))
	t.Run("* frst", testPattern("*aa", "bbb"))
	t.Run("* last", testPattern("aa*", "bbb"))
	t.Run("* midl", testPattern("a*a", "bbb"))
	t.Run("* frst", testPattern("aaa", "*bb"))
	t.Run("* last", testPattern("aaa", "bb*"))
	t.Run("* midl", testPattern("aaa", "b*b"))
	t.Run("? frst", testPattern("aaa", "?bb"))
	t.Run("? last", testPattern("aaa", "bb?"))
	t.Run("? midl", testPattern("a?a", "bbb"))
	t.Run("\\* frst", testNotPattern("\\*aa", "bbb"))
	t.Run("\\* last", testNotPattern("aaa", "bb\\*"))
	t.Run("\\* midl", testNotPattern("a\\*a", "bbb"))
	t.Run("\\? frst", testNotPattern("aaa", "\\?bb"))
	t.Run("\\? last", testNotPattern("aa\\?", "bbb"))
	t.Run("\\? midl", testNotPattern("aaa", "b\\?b"))
	t.Run("**", testPattern("aaa", "b**b"))
	t.Run("??", testPattern("a**a", "bbb"))
	t.Run("\\**", testPattern("aa\\**", "bbb"))
	t.Run("\\??", testPattern("aaa", "b\\??b"))
	t.Run("class", testNotPattern("a[bcd]e"))
	t.Run("range", testNotPattern("a[b-d]e"))
	// TODO have * and ? inside character class :)
}

func testPattern(comp ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		k := NewKey(comp...)
		if !k.IsPattern() {
			t.Fatalf("IsPattern() did not detect pattern in %v", k)
		}
	}
}

func testNotPattern(comp ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		k := NewKey(comp...)
		if k.IsPattern() {
			t.Fatalf("IsPattern() wrongly detected pattern in %v", k)
		}
	}
}

func TestKeyEquals(t *testing.T) {
	t.Run("empty", keyEq(NewKey(), NewKey(), true))
	t.Run("1comp", keyEq(NewKey("aa"), NewKey("aa"), true))
	t.Run("2comps", keyEq(NewKey("aa", "bb"), NewKey("aa", "bb"), true))
	t.Run("diff", keyEq(NewKey("aa", "bb"), NewKey("aa", "b"), false))
	t.Run("bigger", keyEq(NewKey("AA", "BB"), NewKey("AA", "BB", "CC"), false))
	t.Run("smallr", keyEq(NewKey("AA", "BB"), NewKey("AA"), false))
}

func keyEq(k1, k2 *Key, exp bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		if k1.Equals(*k2) != exp {
			t.Fatalf("Equals() failed for k1=%v, k2=%v", k1, k2)
		}
	}
}

func TestKeyMatches(t *testing.T) {
	t.Run("empty", keyMatch(NewKey(), NewKey(), true))
	t.Run("bigger", keyMatch(NewKey("AA"), NewKey("AA", "BB"), false))
	t.Run("smallr", keyMatch(NewKey("AA", "BB"), NewKey("AA"), false))
	t.Run("equals", keyMatch(NewKey("AA", "BB"), NewKey("AA", "BB"), true))
	t.Run("nequal", keyMatch(NewKey("AA", "BB"), NewKey("AA", "BBc"), false))
	t.Run("AA|*", keyMatch(NewKey("AA", "BB"), NewKey("AA", "*"), true))
	t.Run("*|*", keyMatch(NewKey("AA", "BB"), NewKey("*", "*"), true))
	t.Run("*A|B*", keyMatch(NewKey("xyzA", "Bcd"), NewKey("*A", "B*"), true))
	t.Run("neg1:*A|B*", keyMatch(NewKey("xyzABC", "Bcd"), NewKey("*A", "B*"), false))
	t.Run("neg2:*A|B*", keyMatch(NewKey("xyzA", "bcd"), NewKey("*A", "B*"), false))
	t.Run("AA|B*C", keyMatch(NewKey("AA", "BxyzC"), NewKey("A*A", "B*C"), true))
	t.Run("AA|B\\*C", keyMatch(NewKey("AA", "B*C"), NewKey("AA", "B\\*C"), true))
	t.Run("neg1:AA|B\\*C", keyMatch(NewKey("AA", "BxyzC"), NewKey("AA", "B\\*C"), false))
	t.Run("AA|B?", keyMatch(NewKey("AA", "BB"), NewKey("AA", "B?"), true))
	t.Run("??|?B", keyMatch(NewKey("AA", "BB"), NewKey("??", "?B"), true))
	t.Run("?\\?|?B", keyMatch(NewKey("A?", "bB"), NewKey("?\\?", "?B"), true))
	t.Run("*:aa/bb", keyMatch(NewKey("aa/bb"), NewKey("*"), true))
	t.Run("*/*:aa/bb", keyMatch(NewKey("aa/bb"), NewKey("*/*"), true))
	t.Run("*ab*:aabb", keyMatch(NewKey("aabb"), NewKey("*ab*"), true))
	t.Run("*ab*:aab", keyMatch(NewKey("aab"), NewKey("*ab*"), true))
	t.Run("*ab*:abb", keyMatch(NewKey("abb"), NewKey("*ab*"), true))
	t.Run("ab*:abb", keyMatch(NewKey("abb"), NewKey("ab*"), true))
	t.Run("ab\\*:ab*", keyMatch(NewKey("ab*"), NewKey("ab\\*"), true))
	t.Run("ab\\*:abb", keyMatch(NewKey("ab*"), NewKey("abb"), false))
	t.Run("ab\\:abb", keyMatch(NewKey("ab\\"), NewKey("abb"), false))
	t.Run("abb:ab", keyMatch(NewKey("ab"), NewKey("abb"), false))
	t.Run("aa:bb", keyMatch(NewKey("bb"), NewKey("aa"), false))
	t.Run("a*b:aa/bb", keyMatch(NewKey("aa/bb"), NewKey("a*b"), true))
	t.Run("a**b:ab", keyMatch(NewKey("ab"), NewKey("a******b"), true))
	t.Run("a**b:axyb", keyMatch(NewKey("axyb"), NewKey("a******b"), true))
	t.Run("**b:axyb", keyMatch(NewKey("axyb"), NewKey("******b"), true))
	t.Run("a**:axyb", keyMatch(NewKey("axyb"), NewKey("a******"), true))
	t.Run("ipaddr", keyMatch(NewKey("10.1.2.3/24"), NewKey("10.*"), true))
}

func keyMatch(k, p *Key, exp bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		if k.Matches(*p) == exp {
		} else if exp {
			t.Fatalf("Key %v did not match pattern %v", k, p)
		} else {
			t.Fatalf("Key %v should not have matched pattern %v", k, p)
		}
	}
}
