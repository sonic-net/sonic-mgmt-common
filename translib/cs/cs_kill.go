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

// Config Session Kill

package cs

import (
	"errors"

	"github.com/golang/glog"
)

func (sess *Session) Kill(killActive bool, user string) CsStatus {
	glog.Infof("Kill:[%s]:Begin: killActive=%t", sess.token, killActive)

	var status CsStatus
	var notify bool

	if sess.IsConfigSession() {

		switch sess.configSession.state {
		case cs_STATE_ACTIVE:

			if !killActive && sess.IsPidActive() {
				glog.Infof("Kill[%s]: Session is active on terminal %d", sess.token, sess.pid)
				status = CsStatusInvalidSession{Tag: ErrTagActive}
				break
			}

			notify = true
			glog.Infof("Kill:[%s]: KillActive Option", sess.token)
			fallthrough

		case cs_STATE_SUSPENDED:
			var err, errU error

			//rollback if commit timer is running.
			if sess.configSession.commitState == cs_STATE_CONFIRM_TIMER {
				if err = abortSessionCommit(sess); err != nil {
					status = CsStatusInternalError{Err: errors.New("Config reload failure on session clear")}
					break
				}
			}

			if sess.configSession.commitState == cs_STATE_ROLLBACK_REPLACE {
				status = CsStatusSuccess{}
				break
			}

			err, errU = deleteCS(sess.name)
			if err != nil {
				status = CsStatusInternalError{Err: err}
				notify = false
			} else if errU != nil {
				status = CsStatusAbortWarning{UnlockFailure: errU}
			} else {
				status = CsStatusSuccess{}
			}

		default:
			status = CsStatusInvalidSession{Tag: ErrTagInvalidState}
		}

	} else {

		status = CsStatusInvalidSession{}
	}

	if notify {
		sess.SendMesg("The configure session was cleared", user)
	}

	glog.Infof("Kill:[%s]:End:", sess.token)
	return status
}
