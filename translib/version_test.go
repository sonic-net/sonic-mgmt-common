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

package translib

import (
	"fmt"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Workiva/go-datastructures/queue"
)

func ver(major, minor, patch uint32) Version {
	return Version{Major: major, Minor: minor, Patch: patch}
}

func TestVersionParseStr(t *testing.T) {
	t.Run("empty", testVerSet("", ver(0, 0, 0), false))
	t.Run("0.0.0", testVerSet("0.0.0", ver(0, 0, 0), false))
	t.Run("1.0.0", testVerSet("1.0.0", ver(1, 0, 0), true))
	t.Run("1.2.3", testVerSet("1.2.3", ver(1, 2, 3), true))
	t.Run("1.-.-", testVerSet("1", Version{}, false))
	t.Run("1.2.-", testVerSet("1.2", Version{}, false))
	t.Run("1.-.3", testVerSet("1..2", Version{}, false))
	t.Run("neg_majr", testVerSet("-1.0.0", Version{}, false))
	t.Run("bad_majr", testVerSet("A.2.3", Version{}, false))
	t.Run("neg_minr", testVerSet("1.-2.0", Version{}, false))
	t.Run("bad_minr", testVerSet("1.B.3", Version{}, false))
	t.Run("neg_pat", testVerSet("1.2.-3", Version{}, false))
	t.Run("bad_pat", testVerSet("1.2.C", Version{}, false))
	t.Run("invalid", testVerSet("invalid", Version{}, false))
}

func testVerSet(vStr string, exp Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		v, err := NewVersion(vStr)
		if err != nil {
			if expSuccess == true {
				t.Fatalf("Failed to parse \"%s\"; err=%v. Expected %v", vStr, err, exp)
			}
			return
		}

		if !expSuccess {
			t.Fatalf("Version string \"%s\" was expected to fail; but got %v", vStr, v)
		}

		if v != exp {
			t.Fatalf("Failed to parse \"%s\"; expected %v", vStr, v)
		}
	}
}

func TestVersionGetBase(t *testing.T) {
	t.Run("0.0.0", testGetBase(ver(0, 0, 0), ver(0, 0, 0)))
	t.Run("0.0.1", testGetBase(ver(0, 0, 1), ver(0, 0, 1)))
	t.Run("0.1.2", testGetBase(ver(0, 1, 2), ver(0, 1, 2)))
	t.Run("1.0.0", testGetBase(ver(1, 0, 0), ver(1, 0, 0)))
	t.Run("1.0.1", testGetBase(ver(1, 0, 1), ver(1, 0, 0)))
	t.Run("1.2.1", testGetBase(ver(1, 2, 1), ver(1, 0, 0)))
	t.Run("2.0.0", testGetBase(ver(2, 0, 0), ver(1, 0, 0)))
	t.Run("2.3.4", testGetBase(ver(2, 3, 4), ver(1, 0, 0)))
	t.Run("3.0.0", testGetBase(ver(5, 6, 7), ver(4, 0, 0)))
}

func testGetBase(v, expBase Version) func(*testing.T) {
	return func(t *testing.T) {
		base := v.GetCompatibleBaseVersion()
		if base != expBase {
			t.Fatalf("Got base of %s as %s; expected %s", v, base, expBase)
		}
	}
}

func setYangBundleVersion(v Version) {
	theYangBundleVersion = v
	theYangBaseVersion = v.GetCompatibleBaseVersion()
}

func TestVersionCheck(t *testing.T) {
	// Set yang bundle version tp 2.3.4 and try various ClientVersion
	testVer, _ := NewVersion("2.3.4")
	origVer := theYangBundleVersion
	setYangBundleVersion(testVer)
	defer setYangBundleVersion(origVer) // restore original version

	testVCheck(t, "", true)
	testVCheck(t, "0.0.9", false)
	testVCheck(t, "0.9.9", false)
	testVCheck(t, "1.0.0", true)
	testVCheck(t, "1.2.3", true)
	testVCheck(t, "2.0.0", true)
	testVCheck(t, "2.1.9", true)
	testVCheck(t, "2.3.2", true)
	testVCheck(t, "2.3.4", true)
	testVCheck(t, "2.3.9", false)
	testVCheck(t, "2.4.0", false)
	testVCheck(t, "3.0.0", false)
}

func testVCheck(t *testing.T, ver string, expSuccess bool) {
	v, err := NewVersion(ver)
	if ver != "" && err != nil {
		t.Fatalf("Bad version \"%s\"", ver)
	}

	t.Run(fmt.Sprintf("get_%s", ver), vGet(v, expSuccess))
	t.Run(fmt.Sprintf("create_%s", ver), vCreate(v, expSuccess))
	t.Run(fmt.Sprintf("update_%s", ver), vUpdate(v, expSuccess))
	t.Run(fmt.Sprintf("delete_%s", ver), vDelete(v, expSuccess))
	t.Run(fmt.Sprintf("replace_%s", ver), vReplace(v, expSuccess))
	t.Run(fmt.Sprintf("action_%s", ver), vAction(v, expSuccess))
	t.Run(fmt.Sprintf("subs_%s", ver), vSubscribe(v, expSuccess))
	t.Run(fmt.Sprintf("is_subs_%s", ver), vIsSubscribe(v, expSuccess))
}

var (
	tPath = "/openconfig-acl:acl"
	tBody = []byte("{}")
)

func vCreate(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Create(SetRequest{Path: tPath, Payload: tBody, ClientVersion: v})
		checkErr(t, err, expSuccess)
	}
}

func vUpdate(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Update(SetRequest{Path: tPath, Payload: tBody, ClientVersion: v})
		checkErr(t, err, expSuccess)
	}
}

func vReplace(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Replace(SetRequest{Path: tPath, Payload: tBody, ClientVersion: v})
		checkErr(t, err, expSuccess)
	}
}

func vDelete(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Delete(SetRequest{Path: tPath, ClientVersion: v})
		checkErr(t, err, expSuccess)
	}
}

func vGet(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Get(GetRequest{Path: tPath, ClientVersion: v})
		checkErr(t, err, expSuccess)
	}
}

func vAction(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Action(ActionRequest{Path: tPath, ClientVersion: v})
		checkErr(t, ignoreNotImpl(err), expSuccess)
	}
}

func vSubscribe(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		stopChan := make(chan struct{})
		defer close(stopChan)
		req := SubscribeRequest{
			Paths:         []string{tPath},
			ClientVersion: v,
			Q:             queue.NewPriorityQueue(10, true),
			Stop:          stopChan,
		}
		err := Subscribe(req)
		checkErr(t, ignoreNotImpl(err), expSuccess)
	}
}

func vIsSubscribe(v Version, expSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		p := IsSubscribePath{Path: tPath}
		req := IsSubscribeRequest{Paths: []IsSubscribePath{p}, ClientVersion: v}
		resp, err := IsSubscribeSupported(req)
		if err == nil && len(resp) == 1 && resp[0].Err != nil {
			err = resp[0].Err
		}
		checkErr(t, ignoreNotImpl(err), expSuccess)
	}
}

func isVersionError(err error) bool {
	if _, ok := err.(tlerr.TranslibUnsupportedClientVersion); ok {
		return true
	}
	return false
}

func checkErr(t *testing.T, err error, expSuccess bool) {
	if expSuccess && err != nil {
		t.Fatalf("Unexpected %T %v", err, err)
	}
	if !expSuccess && !isVersionError(err) {
		t.Fatalf("Unexpected %T %v; expected TranslibUnsupportedClientVersion", err, err)
	}
}

func ignoreNotImpl(err error) error {
	switch err.(type) {
	case nil:
		return nil
	case tlerr.NotSupportedError:
		return nil
	default:
		e := err.Error()
		if e == "Not supported" || e == "Not implemented" {
			return nil
		}
	}
	return err
}
