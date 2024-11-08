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

package cs

// Config Session Transaction
// (Start|Commit|Abort)Tx are to be used by the apps that need to invoke them

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/golang/glog"
)

func (sess *Session) StartTx(d *db.DB, w []db.WatchKeys,
	tss []*db.TableSpec) error {

	glog.Infof("cs.StartTx:[%s]: Begin", sess.token)
	var e error
	if (sess.state == cs_STATE_None) && (d == sess.ccDB) {
		e = d.DeclareSP()
	} else {
		e = d.StartTx(w, tss)
	}
	return e
}

func (sess *Session) CommitTx(d *db.DB) error {
	glog.Infof("cs.CommitTx:[%s]: Begin", sess.token)
	var e error
	if (uCS != nil) && (d == uCS.ccDB) {
		e = d.ReleaseSP()
	} else {
		e = d.CommitTx()
	}
	return e
}

func (sess *Session) AbortTx(d *db.DB) error {
	glog.Infof("cs.AbortTx:[%s]: Begin", sess.token)
	var e error
	if (uCS != nil) && (d == uCS.ccDB) {
		e = d.Rollback2SP()
	} else {
		e = d.AbortTx()
	}
	return e
}

func csStartTx(d *db.DB) error {
	glog.Infof("csStartTx: Begin")
	return d.StartSessTx(nil, []*db.TableSpec{&(db.TableSpec{Name: "*"})})
}

func csCommitTx(d *db.DB) error {
	glog.Infof("csCommitTx: Begin")
	return d.CommitSessTx()
}

func csAbortTx(d *db.DB) error {
	glog.Infof("csAbortTx: Begin")
	return d.AbortSessTx()
}
