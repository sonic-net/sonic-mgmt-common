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
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

func TestCSGetSession(t *testing.T) {

	t.Logf("Creating CS")
	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	t.Cleanup(func() { deleteCS(sName) })

	sess, e := GetSession(sName, u.token, user, uR, pid, GSOstrict{})
	if e != nil || sess.name != sName {
		t.Errorf("GetSession() GSOstrict{} fails e: %v", e)
	}

	if !sess.IsPidActive() {
		t.Errorf("GetSession().IsPidActive(): false")
	}

	if !sess.IsConfigSession() {
		t.Errorf("GetSession().IsConfigSession(): false")
	}

	sess, e = GetSession(sName, "", user, uR, pid, GSOname{})
	if e != nil || sess.token != u.token {
		t.Errorf("GetSession() GSOname{} fails e: %v", e)
	}

	sess, e = GetSession("", "", user, uR, pid, GSOname{})
	if e != nil || sess.username != user {
		t.Errorf("GetSession() GSOname{} fails e: %v", e)
	}

}

func TestCSGetSessionWithSuspend(t *testing.T) {

	t.Logf("Creating CS")
	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	t.Cleanup(func() { deleteCS(sName) })

	t.Logf("Suspending CS")
	u, e = suspendCS(sName)
	if (u == nil) || (e != nil) {
		t.Fatalf("suspendCS() fails e: %v", e)
	}

	sess, e := GetSession(sName, "", user, uR, pid, GSOname{})
	if e != nil || sess.token != u.token {
		t.Fatalf("GetSession() GSOname{} fails e: %v", e)
	}

	if sess.IsPidActive() {
		t.Errorf("GetSession().IsPidActive(): true")
	}

	if !sess.IsConfigSession() {
		t.Errorf("GetSession().IsConfigSession(): false")
	}

	sess, e = GetSession("", "", user, uR, pid, GSOname{})
	if e != nil || sess.username != user {
		t.Errorf("GetSession() GSOname{} fails e: %v", e)
	}

}

func TestCSGetSessionAll(t *testing.T) {

	sessA, e := GetAllSessions()
	if len(sessA) != 0 {
		t.Errorf("GetAllSessions() fails: %v", sessA)
	}

	t.Logf("Creating CS")
	u, e := newCS(sName, user, uR, pid)
	if (u == nil) || (e != nil) {
		t.Fatalf("newCS() fails e: %v", e)
	}

	t.Cleanup(func() { deleteCS(sName) })

	sessA, e = GetAllSessions()
	if len(sessA) == 0 || e != nil {
		t.Errorf("GetAllSessions() fails: %v e %v", sessA, e)
	}

	t.Logf("Suspending CS")
	u, e = suspendCS(sName)
	if (u == nil) || (e != nil) {
		t.Fatalf("suspendCS() fails e: %v", e)
	}

	sessA, e = GetAllSessions()
	if len(sessA) == 0 || e != nil {
		t.Errorf("GetAllSessions() fails: %v e %v", sessA, e)
	}

}

func TestCSGetConfigDB(t *testing.T) {

	sess, e := GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Startingh CS")
	token, success, status := sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusCreatedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Cleanup(func() { deleteCS(sName) })

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOname{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Ensure GetConfigDB gets Candidate Config DB.")
	dOpts := db.Options{DBNo: db.ConfigDB}
	d, isCS, cleanup, err := sess.GetConfigDB(&dOpts)

	if d == nil || !isCS || err != nil {
		t.Fatalf("GetConfigDB() fails isCS: %v, err: %v", isCS, err)
	}

	cleanup()

	t.Logf("Suspending CS")
	success, status = sess.Exit()
	if !success {
		t.Fatalf("Exit() fails success: %v", success)
	}
	if _, ok := status.(CsStatusSuccess); !ok {
		t.Fatalf("Exit() fails status: %v", status)
	}

	t.Logf("Refresh the Exited Session")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Resume CS")
	token, success, status = sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusResumedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Abort CS")
	success, status = sess.Abort()
	if !success {
		t.Fatalf("Abort() fails success: %v", success)
	}
	if _, ok := status.(CsStatusSuccess); !ok {
		t.Fatalf("Abort() fails status: %v", status)
	}
}

func TestCSCommit(t *testing.T) {
	sess, e := GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Starting CS")
	token, success, status := sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusCreatedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Cleanup(func() { deleteCS(sName) })

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOname{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Commit CS")
	success, status = sess.Commit(commitLabel, 0, false)
	if !success {
		t.Fatalf("Commit() fails success: %v", success)
	}
}

func TestCSKill(t *testing.T) {
	sess, e := GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Starting CS")
	token, success, status := sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusCreatedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Cleanup(func() { deleteCS(sName) })

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Don't Kill Active. Should Fail")
	status = sess.Kill(false, user)
	if _, ok := status.(CsStatusInvalidSession); !ok {
		t.Fatalf("Kill() succeeds status: %v", status)
	}

	t.Logf("Now Kill Active")
	status = sess.Kill(true, user)
	if _, ok := status.(CsStatusSuccess); !ok {
		t.Fatalf("Kill() fails status: %v", status)
	}

	t.Logf("Try INACTIVE session")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Starting CS")
	token, success, status = sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusCreatedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Logf("Exit to make it INACTIVE")
	success, status = sess.Exit()
	if !success {
		t.Fatalf("Exit() fails success: %v", success)
	}
	if _, ok := status.(CsStatusSuccess); !ok {
		t.Fatalf("Exit() fails status: %v", status)
	}

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Kill InActive. Should Succeed")
	status = sess.Kill(false, user)
	if _, ok := status.(CsStatusSuccess); !ok {
		t.Fatalf("Kill() fails status: %v", status)
	}
}

func TestCSTxWithSession(t *testing.T) {
	sess, e := GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v", e)
	}

	t.Logf("Starting CS")
	token, success, status := sess.StartOrResume(pid)
	if token == "" || !success {
		t.Fatalf("StartOrResume() fails token: %v success: %v", token, success)
	}
	if _, ok := status.(CsStatusCreatedSession); !ok {
		t.Fatalf("StartOrResume() fails status: %v", status)
	}

	t.Cleanup(func() { deleteCS(sName) })

	t.Logf("Get the just created Config Session sess.")
	sess, e = GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil || sess.token != token {
		t.Fatalf("GetSession() GSOstrict|name{} fails e: %v token: (%v,%v)", e,
			sess.token, token)
	}

	t.Logf("Ensure GetConfigDB gets Candidate Config DB.")
	dOpts := db.Options{DBNo: db.ConfigDB}
	d, isCS, cleanup, err := sess.GetConfigDB(&dOpts)

	defer cleanup()

	if d == nil || !isCS || err != nil {
		t.Fatalf("GetConfigDB() fails isCS: %v, err: %v", isCS, err)
	}

	t.Logf("StartTx")
	e = sess.StartTx(d, nil, nil)
	if e != nil {
		t.Errorf("StartTx() fails e: %v", e)
	}

	t.Logf("AbortTx")
	e = sess.AbortTx(d)
	if e != nil {
		t.Errorf("AbortTx() fails e: %v", e)
	}

	t.Logf("StartTx")
	e = sess.StartTx(d, nil, nil)
	if e != nil {
		t.Errorf("StartTx() fails e: %v", e)
	}

	t.Logf("CommitTx")
	e = sess.CommitTx(d)
	if e != nil {
		t.Errorf("CommitTx() fails e: %v", e)
	}
}

func TestCSTxNoSession(t *testing.T) {
	sess, e := GetSession(sName, "", user, uR, pid, GSOstrict{}, GSOname{})
	if e != nil {
		t.Fatalf("GetSession() GSOstrict{}, GSOname{} fails e: %v", e)
	}

	t.Logf("Ensure GetConfigDB gets Candidate Config DB.")
	dOpts := db.Options{DBNo: db.ConfigDB}
	d, isCS, cleanup, err := sess.GetConfigDB(&dOpts)

	defer cleanup()

	if d == nil || isCS || err != nil {
		t.Fatalf("GetConfigDB() fails isCS: %v, err: %v", isCS, err)
	}

	t.Logf("StartTx")
	e = sess.StartTx(d, nil, nil)
	if e != nil {
		t.Errorf("StartTx() fails e: %v", e)
	}

	t.Logf("AbortTx")
	e = sess.AbortTx(d)
	if e != nil {
		t.Errorf("AbortTx() fails e: %v", e)
	}

	t.Logf("StartTx")
	e = sess.StartTx(d, nil, nil)
	if e != nil {
		t.Errorf("StartTx() fails e: %v", e)
	}

	t.Logf("CommitTx")
	e = sess.CommitTx(d)
	if e != nil {
		t.Errorf("CommitTx() fails e: %v", e)
	}
}
