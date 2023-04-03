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

package translib

import (
	"fmt"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/openconfig/gnmi/proto/gnmi"
)

/**
 *This file contains type definitions to be used by app modules to
 * handle subscribe requests.
 */

// translateSubRequest is the input for translateSubscribe callback
type translateSubRequest struct {
	ctxID   interface{}      // request id for logging
	path    string           // subscribe path
	mode    NotificationType // requested notification type
	recurse bool             // whether mappings for child paths are required
	dbs     [db.MaxDB]*db.DB // DB objects for querying, if needed
}

// translateSubResponse is the output returned by app modules
// from translateSubscribe callback.
type translateSubResponse struct {
	// ntfAppInfoTrgt includes the notificationAppInfo mappings for top
	// level tables corresponding to the subscribe path. At least one
	// such mapping should be present.
	ntfAppInfoTrgt []*notificationAppInfo

	// ntfAppInfoTrgtChlds includes notificationAppInfo mappings for the
	// dependent tables of the entries in ntfAppInfoTrgt. Should be nil
	// if there are no dependent tables.
	ntfAppInfoTrgtChlds []*notificationAppInfo
}

// notificationAppInfo contains the details for monitoring db notifications
// for a given path. App modules provide these details for each subscribe
// path. One notificationAppInfo object must inclue details for one db table.
// One subscribe path can map to multiple notificationAppInfo.
type notificationAppInfo struct {
	// database index for the DB key represented by this notificationAppInfo.
	// Should be db.MaxDB for non-DB data provider cases.
	dbno db.DBNum

	// table name. Should be nil for non-DB case.
	table *db.TableSpec

	// key components without table name prefix. Can include wildcards.
	// Should be nil for non-DB case.
	key *db.Key

	// keyGroupComps holds component indices that uniquely identify the path.
	// Required only when the db entry represents leaf-list instances.
	keyGroupComps []int

	// path to which the key maps to. Can include wildcard keys.
	// Should match request path -- should not point to any node outside
	// the yang segment of request path.
	path *gnmi.Path

	// handlerFunc is the custom on_change event handler callback.
	// Apps can implement their own diff & translate logic in this callback.
	handlerFunc apis.ProcessOnChange

	// dbFieldYangPathMap is the mapping of db entry field to the yang
	// field (leaf/leaf-list) for the input path.
	dbFldYgPathInfoList []*dbFldYgPathInfo

	// deleteAction indicates how entry delete be handled for this path.
	// Required only when db entry represents partial data for the path,
	// or to workaround out of order deletes due to backend limitations.
	deleteAction apis.DeleteActionType

	//fldScanPattern indicates the scan type is based on field names and
	// also the pattern to match the field names in the given table
	fieldScanPattern string

	// isOnChangeSupported indicates if on-change notification is
	// supported for the input path. Table and key mappings should
	// be filled even if on-change is not supported.
	isOnChangeSupported bool

	// mInterval indicates the minimum sample interval supported for
	// the input path. Can be set to 0 (default value) to indicate
	// system default interval.
	mInterval int

	// pType indicates the preferred notification type for the input
	// path. Used when gNMI client subscribes with "TARGET_DEFINED" mode.
	pType NotificationType

	// opaque data can be used to store context information to assist
	// future key-to-path translations. This is an optional data item.
	// Apps can store any context information based on their logic.
	// Translib passes this back to the processSubscribe function when
	// it detects changes to the DB entry for current key or key pattern.
	opaque interface{}

	// isDataSrcDynamic can be used to identify the type of the data source
	isDataSrcDynamic bool
}

type dbFldYgPathInfo struct {
	rltvPath       string
	dbFldYgPathMap map[string]string //db field to leaf / rel. path to leaf
}

// processSubRequest is the input for app module's processSubscribe function.
// It includes a path template (with wildcards) and one db key that needs to
// be mapped to the path.
type processSubRequest struct {
	ctxID interface{} // context id for logging
	path  *gnmi.Path  // path template to be filled -- contains wildcards

	// DB entry info to be used for filling the path template
	dbno  db.DBNum
	table *db.TableSpec
	key   *db.Key
	entry *db.Value // updated or deleted db entry. DO NOT MODIFY

	// List of all DB objects. Apps should only use these DB objects
	// to query db if they need additional data for translation.
	dbs [db.MaxDB]*db.DB

	// App specific opaque data -- can be used to pass context data
	// between translateSubscribe and processSubscribe.
	opaque interface{}
}

// processSubResponse is the output data structure of processSubscribe
// function. Includes the path with wildcards resolved. Translib validates
// if this path matches the template in processSubRequest.
type processSubResponse struct {
	// path with wildcards resolved
	path *gnmi.Path
}

func (ni *notificationAppInfo) String() string {
	if ni == nil {
		return "<nil>"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "{path='%s'", apis.PathToString(ni.path))
	fmt.Fprintf(&b, ", db=%s, ts=%v, key=%v", ni.dbno, tableInfo(ni.table), keyInfo(ni.key))
	if len(ni.keyGroupComps) != 0 {
		fmt.Fprintf(&b, ", keyGrp=%v", ni.keyGroupComps)
	}
	if len(ni.fieldScanPattern) != 0 {
		fmt.Fprintf(&b, ", fieldScanPattern=%v", ni.fieldScanPattern)
	}
	fmt.Fprintf(&b, ", dynamic=%v", ni.isDataSrcDynamic)
	fmt.Fprintf(&b, ", fields={")
	for i, fi := range ni.dbFldYgPathInfoList {
		if i != 0 {
			fmt.Fprintf(&b, ", ")
		}
		fmt.Fprintf(&b, "%s=%v", fi.rltvPath, fi.dbFldYgPathMap)
	}
	fmt.Fprintf(&b, "}, delAction=%v", ni.deleteAction)
	if ni.handlerFunc != nil {
		fmt.Fprintf(&b, ", handlerFunc=%s", ni.handlerFunc)
	}
	fmt.Fprintf(&b, ", onchange=%v, preferred=%v, m_int=%d", ni.isOnChangeSupported, ni.pType, ni.mInterval)
	fmt.Fprintf(&b, "}")
	return b.String()
}

// isNonDB returns true if the notificationAppInfo ni is a non-DB mapping.
func (ni *notificationAppInfo) isNonDB() bool {
	return ni.dbno < 0 || ni.dbno >= db.MaxDB || ni.table == nil || ni.key == nil
}

// isLeafPath returns true if the notificationAppInfo has a leaf path.
func (ni *notificationAppInfo) isLeafPath() bool {
	// when notificationAppInfo.path is a leaf path, following conditions
	// MUST be true.
	//  - ni.dbFldYgPathInfoList) has only 1 entry
	//	- ni.dbFldYgPathInfoList[0].rltvPath == ""
	//	- ni.dbFldYgPathInfoList[0].dbFldYgPathMap has only 1 entry
	//		with empty yang field (map value)
	for _, pmap := range ni.dbFldYgPathInfoList {
		if len(pmap.rltvPath) != 0 {
			return false
		}
		for _, yfield := range pmap.dbFldYgPathMap {
			if len(yfield) != 0 && yfield[0] != '{' {
				return false
			}
		}
	}
	return true
}

func (r processSubResponse) String() string {
	return fmt.Sprintf("{path=\"%s\"}", apis.PathToString(r.path))
}

// dbInfo returns display information for a db object
func dbInfo(d *db.DB) interface{} {
	if d != nil {
		return d.Opts.DBNo
	}
	return nil
}

// keyInfo returns display information for a db key object
func keyInfo(k *db.Key) interface{} {
	if k != nil {
		return k.Comp
	}
	return nil
}

// tableInfo returns display information for a db table object
func tableInfo(t *db.TableSpec) interface{} {
	switch {
	case t == nil:
		return nil
	case t.CompCt == 0:
		return t.Name
	default:
		return fmt.Sprintf("%s.%d", t.Name, t.CompCt)
	}
}
