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

// Config Session StartOrResume

package cs

import (
	"github.com/golang/glog"
)

func (sess *Session) StartOrResume(pid int32) (string, bool, CsStatus) {
	glog.Infof("StartOrResume:[%s]:Begin: %s %d", sess.name, sess.token,
		sess.pid)

	var token string
	var success bool
	var status CsStatus

	if sess.IsConfigSession() {

		switch sess.configSession.state {
		case cs_STATE_ACTIVE:

			glog.Infof("StartOrResume: Session In Use. Check if PID is stale")
			if sess.IsPidActive() {
				success = false
				status = CsStatusNotAllowed{}
				break
			}
			glog.Infof("StartOrResume: PID is stale. Suspend Session. Retrying")
			suspendCS(sess.name)
			fallthrough

		case cs_STATE_SUSPENDED:

			if cs, err := resumeCS(sess.name, sess.roles,
				pid); (cs != nil) && (err == nil) {

				glog.Infof("StartOrResume: Resumed Session")
				sess.configSession = *cs
				token = cs.token
				success = true
				status = CsStatusResumedSession{}

			} else {

				glog.Errorf("StartOrResume: Resume Session: err: %v", err)
				success = false
				status = CsStatusInternalError{Err: err}

			}

		default:

			glog.Errorf("StartOrResume: Unknown State")
			success = false
			status = CsStatusInvalidSession{}

		}

	} else {

		if cs, err := newCS(sess.name, sess.username, sess.roles,
			pid); (cs != nil) && (err == nil) {

			glog.Infof("StartOrResume: New Session")
			sess.configSession = *cs
			token = cs.token
			success = true
			status = CsStatusCreatedSession{}

		} else {

			glog.Errorf("StartOrResume: New Session: err: %v", err)
			success = false
			status = CsStatusInternalError{Err: err}
		}

	}

	glog.Infof("StartOrResume:[%s]:End:", sess.token)
	return token, success, status
}
