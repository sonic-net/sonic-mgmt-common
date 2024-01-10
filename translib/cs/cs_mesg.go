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

// Config Session Send Message

package cs

import (
	"fmt"
	"os"

	"github.com/golang/glog"
)

func (sess *Session) SendMesg(mesg string, userName string) error {
	glog.Infof("SendMesg:[%s]:Begin: %s: from %s", sess.token, mesg, userName)

	var err error
	var ttyName string
	var n int
	var tty *os.File

	fMesg := fmt.Sprintf("\n\nMessage from %s: %s\n\n", userName, mesg)
	if sess.IsConfigSession() && sess.IsPidActive() {
		link := fmt.Sprintf("/proc/%d/fd/0", sess.TerminalPID())
		if ttyName, err = os.Readlink(link); err != nil {
			glog.Errorf("SendMesg: Readlink(%s) err %v", link, err)
		} else if tty, err = os.OpenFile(ttyName, os.O_RDWR, 0); err != nil {
			glog.Errorf("SendMesg: Open(%s) err %v", ttyName, err)
		} else {
			defer tty.Close()
			if n, err = tty.WriteString(fMesg); err != nil {
				glog.Infof("SendMesg: WriteString(%s) n %d err %v", fMesg, n,
					err)
				glog.Errorf("SendMesg: WriteString() fails, file: %s", ttyName)
			}
		}
	}

	glog.Infof("SendMesg:[%s]:End:", sess.token)
	return err
}
