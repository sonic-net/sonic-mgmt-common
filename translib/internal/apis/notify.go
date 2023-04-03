////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package apis

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

// ProcessOnChange is a callback function to generate notification messages
// from a DB entry change data. Apps can implement their own diff & notify logic
// and plugin into the translib onchange subscribe infra.
// This callback receives a NotificationContext object containing change details
// of one DB entry and a NotificationSender interface to push translated messages.
// NotificationContext.Path is the subscribed path, which indicates the yang
// attributes that can be included in the notification. Implementations should
// inspect the provided subscribe path and DB changes; and translate them into
// zero, one or more notifications and push them out.
type ProcessOnChange func(*NotificationContext, NotificationSender)

func (p ProcessOnChange) String() string {
	if p == nil {
		return ""
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(p).Pointer()).Name()
	return strings.TrimPrefix(fullName, "github.com/Azure/sonic-mgmt-common/translib/")
}

// NotificationContext contains the subscribed path and details of a DB entry
// change that may result in a notification message.
type NotificationContext struct {
	Path      *gnmi.Path    // subscribe path, can include wildcards
	Db        *db.DB        // db in which the entry was modified
	Table     *db.TableSpec // table for the modified entry
	Key       *db.Key       // key for modified entry
	EntryDiff               // diff info for modified entry
	AllDb     [db.MaxDB]*db.DB
	Opaque    interface{} // app specific opaque data
}

func (nc *NotificationContext) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "{Path='%s'", PathToString(nc.Path))
	fmt.Fprintf(b, ", dbno=%d, table=%v, key=%v", nc.Db.Opts.DBNo, nc.Table, nc.Key)
	fmt.Fprintf(b, ", oldValue=%v, newValue=%v", nc.OldValue.Field, nc.NewValue.Field)
	fmt.Fprintf(b, ", diff=%v}", &nc.EntryDiff)
	return b.String()
}

// NotificationSender provides methods to send notification message to
// the clients. Translib subscribe infra implements this interface.
type NotificationSender interface {
	Send(*Notification) // Send a notification message to clients
}

// Notification is a message containing deleted and updated values
// for a yang path.
type Notification struct {
	// Path is an absolute gnmi path of the changed yang container
	// or list instance. MUST NOT be a leaf path.
	Path string
	// Delete is the list of deleted subpaths (relative to Path).
	// Should contain one empty string if the Path itself was deleted.
	// Can be a nil or empty list if there are no delete paths.
	Delete []string
	// Update holds all the updated values (new+modified) within the Path.
	// MUST be the YGOT struct corresponding to the Path.
	// Can be nil if there are no updated values; or specified as UpdatePaths.
	Update ygot.ValidatedGoStruct
	// UpdatePaths holds the list of updated subpaths (relative to Path).
	// Nil/empty if there are no updates or specified as Update ygot value.
	// Update and UpdatePaths MUST NOT overlap to prevent duplicate notifications.
	UpdatePaths []string
}

// Minimum Interval for sample notification
const (
	SAMPLE_NOTIFICATION_MIN_INTERVAL = 20
)

// DeleteActionType indicates how db delete be handled w.r.t a path.
// By default, db delete will be treated as delete of mapped path.
type DeleteActionType uint8

const (
	// InspectPathOnDelete action attempts Get of mapped path
	// on db delete; notifies delete/update accordingly
	InspectPathOnDelete DeleteActionType = iota + 1
	// InspectLeafOnDelete action attempts Get of each mapped leaf
	// on db delete; and notifies delete/update for each
	InspectLeafOnDelete
)

// PathToString returns a string representation of gnmi path
func PathToString(p *gnmi.Path) string {
	if p == nil {
		return "<nil>"
	}
	if s, err := ygot.PathToString(p); err == nil {
		return s
	}
	return "<invalid>"
}
