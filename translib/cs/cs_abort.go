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

// Config Session Abort

package cs

import (
	"errors"

	"github.com/golang/glog"
)

func (sess *Session) Abort() (bool, CsStatus) {
	glog.Infof("Abort:[%s]:Begin:", sess.token)

	var success bool
	var status CsStatus

	if sess.IsConfigSession() {

		switch sess.configSession.state {
		case cs_STATE_ACTIVE:
			var err, errU error

			//rollback if commit timer is running.
			if sess.configSession.commitState == cs_STATE_CONFIRM_TIMER {
				if err = abortSessionCommit(sess); err != nil {
					success = false
					status = CsStatusInternalError{Err: errors.New("Config reload failure on commit abort")}
					break
				}
			}

			err, errU = deleteCS(sess.name)
			if err != nil {
				status = CsStatusInternalError{Err: err}
			} else if errU != nil {
				success = true
				status = CsStatusAbortWarning{UnlockFailure: errU}
			} else {
				success = true
				status = CsStatusSuccess{}
			}

		default:

			glog.Infof("Abort: Session In Use")
			success = false
			status = CsStatusInvalidSession{Tag: ErrTagNotActive}

		}

	} else {

		success = false
		status = CsStatusInvalidSession{}
	}

	glog.Infof("Abort:[%s]:End:", sess.token)
	return success, status
}
