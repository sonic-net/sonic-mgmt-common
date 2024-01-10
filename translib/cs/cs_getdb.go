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

// Config Session Get ConfigDB

package cs

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/golang/glog"
)

func (sess *Session) GetConfigDB(opts *db.Options) (*db.DB, bool, func(),
	error) {

	glog.Infof("GetConfigDB: %v", opts)

	var err error
	var d *db.DB
	var isCS bool
	var cleanup func()

	if sess.state == cs_STATE_None {

		cleanup = func() { d.DeleteDB() }
		var dopts *db.Options
		if opts != nil {
			dopts = opts
		} else {
			dopts = &(db.Options{DBNo: db.ConfigDB})
		}
		dopts.DBNo = db.ConfigDB
		d, err = db.NewDB(*dopts)

	} else {

		glog.Infof("GetConfigDB: Session DB")
		d = sess.ccDB
		isCS = true
		cleanup = func() {
			sess.configSession.UpdateLastActiveTime()

			// Rollback the stale savepoint if exists (happens when the app module panics)
			if d.HasSP() {
				glog.Infof("Attempting to rollback the stale savepoint...")
				if rbErr := d.Rollback2SP(); rbErr != nil {
					glog.Errorf("Failed to rollback the stale savepoint: %v", rbErr)
				}
			}
		}

	}

	return d, isCS, cleanup, err
}
