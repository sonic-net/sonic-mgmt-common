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

// Config Session

package cs

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

type configSessionState int

const (
	cs_STATE_None configSessionState = iota // 0
	cs_STATE_SUSPENDED
	cs_STATE_ACTIVE
	cs_STATE_CONFIRM_TIMER
	cs_STATE_ROLLBACK_REPLACE
)

const (
	ccDbTxCmdsLim int = 10000
)

type configSession struct {
	// Config Session Identifying attributes
	name  string
	token string

	// Config Session State
	state configSessionState

	// Config Session User information
	username string
	roles    []string

	// Config Session KLISH CLI session information
	pid int32

	// Candidate Configuration DB
	ccDB *db.DB

	// Times
	startTime  time.Time
	resumeTime time.Time
	exitTime   time.Time

	// lastActiveTime could be used for (Future) ConfigSession Idle Timeout
	lastActiveTime time.Time

	//Commit timer
	commitTimer *time.Timer
	// commitTime could be used for (Future) Confirm Commit
	commitTime time.Time

	// Config Session commit state
	commitState configSessionState

	// Current cfg file name for rollback
	rollbackCfg string

	// channel to end timer routine.
	commitCh chan<- bool
}

var csMutex sync.Mutex

// uCS is the unnamed Config Session (the only one supported in first cut)
var uCS *configSession

var csTokenCtr uint32

func (cs configSession) String() string {
	return fmt.Sprintf("{ name: %v, token: %v, state: %v,\n"+
		"  username: %v, roles: %v, pid: %v, ccDB: %v,\n"+
		"  startTime: %v, resumeTime: %v, exitTime: %v, lastActiveTime: %v,\n"+
		"  commitTime, %v }",
		cs.name, cs.token, cs.state,
		cs.username, cs.roles, cs.pid, cs.ccDB,
		cs.startTime, cs.resumeTime, cs.exitTime, cs.lastActiveTime,
		cs.commitTime)
}

func (cs *configSession) IsPidActive() bool {
	if cs == nil {
		return false
	}

	pid := cs.pid
	if pid == 0 {
		return false
	} else if process, err := os.FindProcess(int(pid)); err != nil {
		glog.Errorf("cs.IsPidActive[%s]: FindProcess() err %s", cs.token, err)
		return false
	} else if err = process.Signal(syscall.Signal(0)); err != nil {
		glog.Infof("cs.IsPidActive[%s]: Inactive pid %v", cs.token, pid)
		return false
	}

	return true
}

func (cs *configSession) UpdateLastActiveTime() {
	csMutex.Lock()
	defer csMutex.Unlock()

	if !(cs == nil || uCS == nil) {
		uCS.lastActiveTime = time.Now()
	}
}

func (cs *configSession) StartCommitTimer(timeout int) *time.Timer {
	csMutex.Lock()
	defer csMutex.Unlock()
	if !(cs == nil || uCS == nil) {
		uCS.commitTimer = time.NewTimer(time.Duration(timeout) * time.Second)
		return uCS.commitTimer
	}
	return nil
}

func (cs *configSession) SetRollbackCfg(cfg string) {
	csMutex.Lock()
	defer csMutex.Unlock()
	if !(cs == nil || uCS == nil) {
		uCS.rollbackCfg = cfg
	}
}

func (cs *configSession) SetCommitState(state configSessionState) {
	csMutex.Lock()
	defer csMutex.Unlock()
	if !(cs == nil || uCS == nil) {
		uCS.commitState = state
	}
}

func (cs *configSession) SetCommitCh(ch chan<- bool) {
	csMutex.Lock()
	defer csMutex.Unlock()
	if !(cs == nil || uCS == nil) {
		uCS.commitCh = ch
	}
}

func newCS(name string, username string, roles []string, pid int32) (*configSession, error) {
	glog.Infof("newCS: %s %s %v %d", name, username, roles, pid)

	csMutex.Lock()
	defer csMutex.Unlock()

	// Does the uCS exist already?
	if uCS != nil {
		return nil, tlerr.TranslibBusy{}
	}

	ccDB, err := db.NewDB(db.Options{DBNo: db.ConfigDB,
		IsSession: true,
		TxCmdsLim: ccDbTxCmdsLim})
	if err != nil {
		glog.Errorf("newCS: db.NewDB err %s", err)
		return nil, err
	}

	if err = csStartTx(ccDB); err != nil {
		ccDB.DeleteDB()
		return nil, err
	}

	token := fmt.Sprintf("%d-%d", time.Now().Unix(),
		atomic.AddUint32(&csTokenCtr, 1))

	if err = db.ConfigDBTryLock(token); err != nil {
		glog.Errorf("newCS: db.ConfigDBTryLock err %s", err)
		csAbortTx(ccDB)
		ccDB.DeleteDB()
		return nil, err
	}

	uCS = &configSession{
		name:     name,
		token:    token,
		state:    cs_STATE_ACTIVE,
		username: username,
		roles:    roles,
		pid:      pid,
		ccDB:     ccDB,
	}

	uCS.startTime = time.Now()
	uCS.lastActiveTime = uCS.startTime

	glog.Infof("newCS[%s]: %s %s %v %d", token, name, username, roles, pid)
	return uCS, nil
}

func resumeCS(name string, roles []string, pid int32) (*configSession, error) {
	glog.Infof("resumeCS: %s %s %v", name, roles, pid)

	csMutex.Lock()
	defer csMutex.Unlock()

	if uCS == nil || uCS.state != cs_STATE_SUSPENDED {
		return uCS, tlerr.TranslibBusy{}
	}

	uCS.state = cs_STATE_ACTIVE

	// Update the Pid, and Roles
	uCS.roles = roles
	uCS.pid = pid
	uCS.resumeTime = time.Now()
	uCS.lastActiveTime = uCS.resumeTime

	glog.Infof("resumeCS[%s]: %s %s %v", uCS.token, name, roles, pid)
	return uCS, nil
}

func suspendCS(name string) (*configSession, error) {
	glog.Infof("suspendCS: %s", name)

	csMutex.Lock()
	defer csMutex.Unlock()

	if uCS == nil || uCS.state != cs_STATE_ACTIVE {
		return uCS, tlerr.TranslibBusy{}
	}

	uCS.state = cs_STATE_SUSPENDED
	uCS.pid = 0
	uCS.exitTime = time.Now()
	uCS.lastActiveTime = uCS.exitTime

	glog.Infof("suspendCS[%s]: %s", uCS.token, name)
	return uCS, nil
}

// commitCS commits the Transaction. A primary error, and a list of secondary
// (warning?) errors are returned. In the absence of a primary error,
// secondary errors indicate that the commit was successful, but a
// secondary operation (Eg: Unlocking the DB) was not successful)
func commitCS(name string, label string, isConfirmNeeded bool) (error, error, error) {
	glog.Infof("commitCS: name: %s, label: %s", name, label)

	csMutex.Lock()
	defer csMutex.Unlock()

	var err, errSc, errSh error
	var token string

	if uCS == nil || uCS.state != cs_STATE_ACTIVE {
		err = tlerr.TranslibBusy{}
	} else if err = csCommitTx(uCS.ccDB); err != nil {
		// On Commit Failure, keep the Session active so the admin can
		// review their changes. They can abort the session after review.
		glog.Errorf("commitCS: csCommitTx err %s", err)
	} else {

		uCS.commitTime = time.Now()
		token = uCS.token
		if isConfirmNeeded {
			uCS.commitState = cs_STATE_CONFIRM_TIMER
			uCS.ccDB.Opts.IsCommitted = true
		} else {
			// Cp History
			errSh = createCpHistory(label)
			uCS.ccDB.DeleteDB()
			uCS.state = cs_STATE_None
			// Db Unlock
			if errSc = db.ConfigDBUnlock(uCS.token); errSc != nil {
				glog.Warningf("commitCS: db.ConfigDBUnlock errSc %+v", errSc)
			}
			uCS = nil
		}
	}

	if err == nil && !isConfirmNeeded {
		uCS = nil
	}

	glog.Infof("commitCS[%s]: end", token)

	return err, errSc, errSh
}

// deleteCS aborts the Transaction. (similar to commitCS(), it returns two
// errors.
func deleteCS(name string) (error, error) {
	glog.Infof("deleteCS: %s", name)

	csMutex.Lock()
	defer csMutex.Unlock()

	if uCS == nil {
		return tlerr.TranslibBusy{}, nil
	}
	switch uCS.state {
	case cs_STATE_ACTIVE, cs_STATE_SUSPENDED:
		break
	default:
		return tlerr.TranslibBusy{}, nil
	}

	//Skip AbortTx when commit is in commit timer state.
	//CommitTx is done while moving to commit timer state.
	var err error
	if uCS.commitState != cs_STATE_CONFIRM_TIMER {
		err = csAbortTx(uCS.ccDB)
		if err != nil {
			glog.Errorf("deleteCS: csAbortTx err %s", err)
		}
	}

	errU := db.ConfigDBUnlock(uCS.token)
	if errU != nil {
		glog.Errorf("deleteCS: db.ConfigDBUnlock err %s", errU)
	}

	uCS.ccDB.DeleteDB()
	uCS.state = cs_STATE_None

	glog.Infof("deleteCS[%s]: %s err %s errU %s", uCS.token, name, err, errU)

	uCS = nil

	return err, errU
}

func createCpHistory(label string) error {
	// Cp History
	var errSh error
	if errSh = CreateCpHistEntry(label, uCS.token, uCS.username,
		"commit", uCS.commitTime.UnixNano()); errSh != nil {
		glog.Warningf("commitCS: CreateCpHistEntry errSh %+v", errSh)
	}
	return errSh
}

func cleanCS() error {

	csMutex.Lock()
	defer csMutex.Unlock()

	uCS.ccDB.DeleteDB()
	uCS.state = cs_STATE_None
	// Db Unlock
	var errSc error
	if errSc = db.ConfigDBUnlock(uCS.token); errSc != nil {
		glog.Warningf("commitCS: db.ConfigDBUnlock errSc %+v", errSc)
	}
	uCS = nil
	return errSc
}

func DebugGetCSDB() *db.DB {
	csMutex.Lock()
	defer csMutex.Unlock()

	if uCS != nil {
		return uCS.ccDB
	}

	return nil
}
