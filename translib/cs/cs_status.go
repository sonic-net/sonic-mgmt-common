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

// Config Session Status

package cs

import (
	"fmt"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

type CsStatus interface {
	Status() string
}

func Err2CsStatus(err error) CsStatus {
	var status CsStatus
	if err == nil {
		status = CsStatusSuccess{}
	} else {
		switch err := err.(type) {
		case CsStatus:
			return err
		case tlerr.TranslibInvalidSession:
			status = CsStatusInvalidSession{Tag: ErrTagUnknown}
		default:
			status = CsStatusInternalError{Err: err}
		}
	}

	return status
}

type ErrTag string

const (
	ErrTagNameNotFound    ErrTag = "cs-name-not-found"  // Lookup by name failed
	ErrTagTokenNotFound   ErrTag = "cs-token-not-found" // Lookup by token failed
	ErrTagInvalidUser     ErrTag = "cs-invalid-user"    // CS owned by someone else
	ErrTagInvalidTerminal ErrTag = "cs-invalid-term"    // CS active on different terminal
	ErrTagActive          ErrTag = "cs-active"          // Op not allowed on active CS
	ErrTagNotActive       ErrTag = "cs-inactive"        // Op not allowed on suspended CS
	ErrTagInvalidState    ErrTag = "cs-invalid-state"   // Op not allowed in current CS state
	ErrTagUnknown         ErrTag = "cs-unknown"         // Any unexpected CS context (mostly code error)
)

type CsStatusCreatedSession struct {
}

func (s CsStatusCreatedSession) Status() string {
	return "Session Created"
}

type CsStatusResumedSession struct {
}

func (s CsStatusResumedSession) Status() string {
	return "Session Resumed"
}

type CsStatusSuccess struct {
}

func (s CsStatusSuccess) Status() string {
	return "Success"
}

// CsStatusInvalidSession indicates session lookup failures
// or bad state. ErrTag holds the actual cause.
type CsStatusInvalidSession struct {
	Tag ErrTag
}

func (s CsStatusInvalidSession) Status() string {
	return "Invalid Session"
}

func (s CsStatusInvalidSession) Error() string {
	return fmt.Sprintf("Invalid Session: %v", s.Tag)
}

// CsStatusNotAllowed indicates user authorization error
type CsStatusNotAllowed struct {
}

func (s CsStatusNotAllowed) Status() string {
	return "Not Allowed"
}

func (s CsStatusNotAllowed) Error() string {
	return s.Status()
}

type CsStatusCommitFailure struct {
	Err error
}

func (s CsStatusCommitFailure) Status() string {
	return fmt.Sprintf("Commit Failure: %s", s.Err)
}

func (s CsStatusCommitFailure) Error() string {
	return s.Status()
}

// CsStatusCommitWarning indicates completion of CS Commit operation with warnings.
type CsStatusCommitWarning struct {
	UnlockFailure     error // ConfigDB unlock failed
	CheckpointFailure error // Checkpoint creation failed
}

func (s CsStatusCommitWarning) Status() string {
	return fmt.Sprintf("Commit Warning: %+v", s)
}

type CsStatusCommitSuccess struct {
}

func (s CsStatusCommitSuccess) Status() string {
	return "Commit Success"
}

// CsStatusAbortWarning indicates completion of CS Abort operation with warnings.
type CsStatusAbortWarning struct {
	UnlockFailure error // ConfigDB unlock failed
}

func (s CsStatusAbortWarning) Status() string {
	return "Aborted with warning"
}

type CsStatusInternalError struct {
	Err error
}

func (s CsStatusInternalError) Status() string {
	return fmt.Sprintf("Internal Error: %s", s.Err)
}

func (s CsStatusInternalError) Error() string {
	return s.Status()
}
