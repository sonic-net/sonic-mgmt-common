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

// Config Session Commit

package cs

import (
	"errors"
	"strconv"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/transformer"
	"github.com/golang/glog"
)

const (
	cs_COMMIT_CONFIRM_TIMEOUT_MIN = 30
)

func saveRunningConfig(username string) (string, error) {

	now := time.Now()
	dest := "temp://rollback.cfg." + strconv.FormatInt(int64(now.UnixNano()), 10) + ".json"

	options := [...]string{"running-configuration", dest, "", username}
	queryResult := transformer.HostQuery("cfg_mgmt.save", options)
	if queryResult.Err != nil {
		glog.Infof("Error: %s", queryResult.Err.Error())
		return "", errors.New("Failed to save current config")
	}
	return dest, nil
}

func processCommitConfirm(sess *Session, label string) (bool, CsStatus) {

	var success bool
	var status CsStatus

	if sess.configSession.commitState != cs_STATE_CONFIRM_TIMER {
		glog.Infof("Commit:[%s] Invalid confirm before commit.", sess.token)
		err := tlerr.New("Confirm is only applicable after commit with timeout.")
		return false, CsStatusCommitFailure{Err: err}
	}

	if sess.configSession.commitTimer != nil {
		glog.Infof("Confirm :[%s]:Stop commit timer.", sess.token)
		sess.configSession.commitTimer.Stop()
		sess.configSession.commitCh <- true
	}
	if errSh := createCpHistory(label); errSh != nil {
		status = CsStatusCommitFailure{Err: errSh}
	} else {
		status = CsStatusCommitSuccess{}
		success = true
	}

	//Clean up to unlock db
	cleanCS()

	return success, status
}

func processCommitTimer(sess *Session, timeout int, rollbackCfg string) (bool, CsStatus) {

	commitTimer := sess.StartCommitTimer(timeout)
	if commitTimer == nil {
		glog.Errorf("Commit:[%s]: Timer start failure. ", sess.token)
		err := tlerr.New("Failed to start commit confirm timer")
		return false, CsStatusCommitFailure{Err: err}
	}
	commitCh := make(chan bool)
	sess.SetCommitCh(commitCh)

	go func() {

		glog.Infof("Commit:[%s]: Timer goroutine. ", sess.token)

		select {
		case <-commitCh:
			glog.Infof("Commit:[%s]: goroutine completed. ", sess.token)
			return

		case timeouttime := <-commitTimer.C:
			//Start rollback process

			sess.SetCommitState(cs_STATE_ROLLBACK_REPLACE)
			glog.Infof("Commit:[%s]:Timer timeout %s, Reload initiated.", sess.token, timeouttime)
			if err := configReload(sess, rollbackCfg); err != nil {
				glog.Errorf("Commit[%s]: Reload failure on commit timer expiry: %s",
					sess.token, err.Error())
			}

			//Clean up
			cleanCS()
		}

	}()
	return true, CsStatusCommitSuccess{}
}

func configReload(sess *Session, rollbackCfg string) error {

	msg := [...]string{"Configure session commit aborted. Configuration 'reload' in progress."}
	queryResult := transformer.HostQuery("infra_host.broadcast_msg", msg)
	if queryResult.Err != nil {
		glog.Errorf("Commit:[%s] Broadcast message failure", sess.token)
	}

	options := [...]string{rollbackCfg, "running-configuration", "OVERWRITE", sess.username}
	queryResult = transformer.HostQuery("cfg_mgmt.save", options)
	if queryResult.Err != nil {
		return errors.New(queryResult.Err.Error())
	}
	return nil
}

func abortSessionCommit(sess *Session) error {
	//rollback if commit timer is running.
	sess.configSession.commitTimer.Stop()
	sess.configSession.commitCh <- true
	if err := configReload(sess, sess.rollbackCfg); err != nil {
		glog.Errorf("Commit[%s]: Configuration reload failure on commit abort : %s",
			sess.token, err.Error())
		return errors.New("Configuration reload failure on commit timeout rollback")
	}
	return nil
}

func (sess *Session) Commit(label string,
	timeout int, confirm bool) (bool, CsStatus) {
	glog.Infof("Commit:[%s]:Begin: label: %s, timeout: %d", sess.token, label, timeout)

	var success bool
	var status CsStatus

	if sess.IsConfigSession() {
		if isCpLabelExist(label) {
			errSh := tlerr.InvalidArgs("label: %s already exist. Choose another label", label)
			status = CsStatusCommitFailure{Err: errSh}
			success = false
			return success, status
		}
		switch sess.configSession.state {

		case cs_STATE_ACTIVE:

			var commitConfirmTimer bool
			var rollbackCfg string
			var err error

			if confirm {
				success, status = processCommitConfirm(sess, label)
				break
			}

			if sess.configSession.commitState == cs_STATE_CONFIRM_TIMER {
				glog.Infof("Commit:[%s] Commit timer running.", sess.token)
				err := tlerr.New("Commit timer running")
				success = false
				status = CsStatusCommitFailure{Err: err}
				break
			}

			if timeout >= cs_COMMIT_CONFIRM_TIMEOUT_MIN {
				//read current config db for rollback
				commitConfirmTimer = true
				//save current running config for rollback.
				if rollbackCfg, err = saveRunningConfig(sess.username); err != nil {
					status = CsStatusCommitFailure{Err: err}
					break
				}
				sess.SetRollbackCfg(rollbackCfg)
			}
			var errSc, errSh error
			if err, errSc, errSh = commitCS(sess.name, label, commitConfirmTimer); err != nil {
				success = false
				status = CsStatusCommitFailure{Err: err}
				break
			}
			success = true

			// If session timer given start the timer and return success.

			if commitConfirmTimer {
				success, status = processCommitTimer(sess, timeout, rollbackCfg)

			} else {
				if (errSc == nil) && (errSh == nil) {
					status = CsStatusCommitSuccess{}
				} else {
					status = CsStatusCommitWarning{
						UnlockFailure:     errSc,
						CheckpointFailure: errSh,
					}
				}
			}
		default:

			glog.Infof("Commit: Session In Use. State %v", sess.configSession.state)
			success = false
			status = CsStatusInvalidSession{Tag: ErrTagNotActive}

		}

	} else {

		glog.Errorf("Commit: Invalid Session")
		success = false
		status = CsStatusInvalidSession{}
	}

	glog.Infof("Commit:[%s]:End:", sess.token)
	return success, status
}
