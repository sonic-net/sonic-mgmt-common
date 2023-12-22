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

// Config Session External Container

package cs

import (
	"fmt"
	"time"

	"github.com/golang/glog"
)

type Session struct {
	configSession
}

type GetSessionOpts interface {
}

// GSOstrict option requires session pid and username to match.
type GSOstrict struct {
}

// GSOname option requires name to match.
type GSOname struct {
}

func GetSession(name string, token string, username string, roles []string,
	pid int32, opts ...GetSessionOpts) (Session, error) {

	glog.Infof("GetSession:[%s]:[%s]: %s %v %d %#v", name, token,
		username, roles, pid, opts)

	var err error
	var session Session
	var gsoStrict bool
	var gsoName bool

	for _, opt := range opts {
		switch opt.(type) {
		case GSOstrict:
			gsoStrict = true
		case GSOname:
			gsoName = true
		default:
			glog.Warningf("GetSession: Invalid Option. %v", opt)
		}
	}

	csMutex.Lock()
	defer csMutex.Unlock()
	ucs := uCS

	session = Session{configSession: configSession{name: name, token: token,
		username: username, roles: roles, pid: pid}}

	if len(token) == 0 && !gsoName {
		// Match by token is requested without a token! We land here when a read/write
		// is triggered from outside session mode. Translib expects an empty Session object
		// so that GetConfigDB() can point to running config
		glog.V(2).Info("GetSession: client not in session context")

	} else if ucs != nil {

		if gsoName && ucs.name != name {
			glog.Warningf("GetSession: name mismatch %s != %s", name, ucs.name)
			err = CsStatusInvalidSession{Tag: ErrTagNameNotFound}

		} else if (!gsoName) && (ucs.token != token) {
			glog.Warningf("GetSession: token mismatch %s != %s", token, ucs.token)
			err = CsStatusInvalidSession{Tag: ErrTagTokenNotFound}

		} else if gsoStrict && (ucs.username != username) {
			glog.Warningf("GetSession: user mismatch %s != %s", username,
				ucs.username)
			err = CsStatusInvalidSession{Tag: ErrTagInvalidUser}

		} else if gsoStrict && ucs.IsPidActive() && (ucs.pid != pid) {
			glog.Warningf("GetSession: pid mismatch %d != %d", pid, ucs.pid)
			err = CsStatusInvalidSession{Tag: ErrTagInvalidTerminal}

		} else {
			session = Session{configSession: *ucs}
			// If this is an attempt to resume a session, update
			// our copy of the pid.
			// But, don't update a stale(i.e. inactive) pid.
			if gsoStrict && gsoName && (session.pid == 0) {
				session.pid = pid
			}
			glog.Infof("GetSession:[%s]:[%s]: Found Session %s", name, token,
				session.token)
		}

	} else if len(token) != 0 {
		glog.Warningf("GetSession[%s][%s]: Token. No Session", name, token)
		err = CsStatusInvalidSession{Tag: ErrTagTokenNotFound}
	}

	return session, err
}

// GetAllSessions returns all session info from cache
func GetAllSessions() ([]Session, error) {
	csMutex.Lock()
	defer csMutex.Unlock()

	var allSess []Session
	if uCS != nil && uCS.state != cs_STATE_None {
		sess := Session{configSession: *uCS}
		allSess = append(allSess, sess)
	}

	return allSess, nil
}

func (sess *Session) IsConfigSession() bool {
	// For now, any state other than cs_STATE_None means this is an actual
	// Config Session instead of just a container for a cs_irpc.go(CsXYZReq)
	// fields, or a stale session.

	return sess.configSession.state != cs_STATE_None
}

func (sess *Session) IsPidActive() bool {
	if sess == nil {
		return false
	}

	return sess.configSession.IsPidActive()
}

func (sess *Session) String() string {
	return fmt.Sprintf("Session: %s", sess.configSession.String())
}

// Token returns the unique id for this session
func (sess *Session) Token() string {
	return sess.token
}

// Name returns session name; or empty string for unnamed session
func (sess *Session) Name() string {
	return sess.name
}

// IsActive returns true if this session is in active state
func (sess *Session) IsActive() bool {
	return sess.state == cs_STATE_ACTIVE
}

func (sess *Session) GetState() string {
	var state string
	if sess.commitState == cs_STATE_CONFIRM_TIMER {
		state = "PENDING CONFIRM"
	} else if sess.commitState == cs_STATE_ROLLBACK_REPLACE {
		state = "ROLLBACK IN PROGRESS"
	} else if sess.IsActive() {
		state = "ACTIVE"
	} else {
		state = "INACTIVE"
	}
	return state
}

// Username returns the user who created this session
func (sess *Session) Username() string {
	return sess.username
}

// TerminalPID returns the active CLI terminal's pid; 0 if inactive
func (sess *Session) TerminalPID() int32 {
	return sess.pid
}

// StartTime returns session creation timestamp.
func (sess *Session) StartTime() time.Time {
	return sess.startTime
}

// LastResumeTime returns timestamp of last successful session resume event
func (sess *Session) LastResumeTime() time.Time {
	return sess.resumeTime
}

// LastExitTime returns timestamp of last successful session exit event
func (sess *Session) LastExitTime() time.Time {
	return sess.exitTime
}

// LastActiveTime returns timestamp of last activity on this session
func (sess *Session) LastActiveTime() time.Time {
	return sess.lastActiveTime
}

// CommitTime returns session commit timestamp
func (sess *Session) CommitTime() time.Time {
	return sess.commitTime
}

// TxLen returns the number of redis operations cached in this session
func (sess *Session) TxLen() int {
	if sess == nil || sess.ccDB == nil {
		return -1
	}
	return int(sess.ccDB.GetStats().AllTables.TxCmdsLen)
}
