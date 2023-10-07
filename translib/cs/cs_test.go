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

// Config Session Test

package cs

import (
	"os"
	"testing"
)

var sName = ""
var user = "sonicbld"
var uR = []string{"sonicbld"}
var pid = int32(os.Getpid())
var commitLabel = "TestCommitLabel"

func TestCSNew(t *testing.T) {

	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	t.Cleanup(func() { deleteCS(sName) })

	u, e = suspendCS(sName)
	if (u == nil) || (e != nil) {
		t.Fatalf("suspendCS() fails e: %v", e)
	}

	u, e = resumeCS(sName, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("resumeCS() fails e: %v", e)
	}

	e, _, _ = commitCS(sName, "", false)
	if e != nil {
		t.Fatalf("commitCS() fails e: %v", e)
	}
}

func TestCSDelete(t *testing.T) {

	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	e, _ = deleteCS(sName)
	if e != nil {
		t.Fatalf("deleteCS() fails e: %v", e)
	}
}

func TestCSPidActive(t *testing.T) {
	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	t.Cleanup(func() { deleteCS(sName) })

	u.UpdateLastActiveTime()

	if !u.IsPidActive() {
		t.Fatalf("IsPidActive() fails e: %v", e)
	}

	e, _ = deleteCS(sName)
	if e != nil {
		t.Fatalf("deleteCS() fails e: %v", e)
	}
}
