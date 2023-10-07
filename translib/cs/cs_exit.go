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

// Config Session Exit

package cs

import (
	"github.com/golang/glog"
)

func (sess *Session) Exit() (bool, CsStatus) {
	glog.Infof("Exit:[%s]:Begin:", sess.token)

	var success bool
	var status CsStatus

	if sess.IsConfigSession() {

		switch sess.configSession.state {
		case cs_STATE_ACTIVE:

			suspendCS(sess.name)
			success = true
			status = CsStatusSuccess{}

		default:

			glog.Infof("Exit: Session In Use")
			success = false
			status = CsStatusInvalidSession{Tag: ErrTagNotActive}

		}

	} else {

		glog.Errorf("Exit: Invalid Session")
		success = false
		status = CsStatusInvalidSession{}
	}

	glog.Infof("Exit:[%s]:End:", sess.token)
	return success, status
}
