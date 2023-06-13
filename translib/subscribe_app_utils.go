////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

// notificationInfoBuilder provides utility APIs to build notificationAppInfo
// data.
type notificationInfoBuilder struct {
	pathInfo *PathInfo
	yangMap  yangMapTree

	primaryInfos []*notificationAppInfo
	subtreeInfos []*notificationAppInfo

	requestPath *gnmi.Path
	currentPath *gnmi.Path // Path cone to be used in notificationAppInfo
	currentIndx int        // Current yangMap node's element in currentPath
	fieldPrefix string     // Yang prefix to be used in Field()
	fieldFilter string     // Field name filter
	treeDepth   int        // depth of current call tree wrt requestPath
	currentInfo *notificationAppInfo
	subtreePath []string // subpath to current node relative to currentInfo.path
}

type yangMapFunc func(nb *notificationInfoBuilder) error

type yangMapTree struct {
	mapFunc yangMapFunc
	subtree map[string]*yangMapTree
}

func (nb *notificationInfoBuilder) Build() (translateSubResponse, error) {
	var err error
	nb.requestPath, err = ygot.StringToStructuredPath(nb.pathInfo.Path)
	if err != nil {
		log.Warningf("Error parsing path \"%s\"; err=%v", nb.pathInfo.Path, err)
		return translateSubResponse{},
			tlerr.InvalidArgs("invalid subscribe path: %s", nb.pathInfo.Path)
	}

	// Find matching yangMapTree node
	index, ymap, recurse := nb.yangMap.match(nb.requestPath, 1)

	if index < 0 {
		return translateSubResponse{},
			tlerr.InvalidArgs("unknown path: %s", nb.pathInfo.Path)
	}

	nb.currentIndx = index
	nb.currentPath = nb.requestPath
	if err := ymap.collect(nb, recurse); err != nil {
		log.Warningf("translateSubscribe failed for path: \"%s\"; err=%s", nb.pathInfo.Path, err)
		return translateSubResponse{}, tlerr.New("Internal error")
	}

	return translateSubResponse{
		ntfAppInfoTrgt:      nb.primaryInfos,
		ntfAppInfoTrgtChlds: nb.subtreeInfos,
	}, nil
}

func (nb *notificationInfoBuilder) New() *notificationInfoBuilder {
	nb.currentInfo = &notificationAppInfo{
		path:                nb.currentPath,
		dbno:                db.MaxDB,
		isOnChangeSupported: true,
		pType:               OnChange,
	}
	nb.subtreePath = nil // nb.currentPath already has the subpath
	if nb.treeDepth == 0 {
		nb.primaryInfos = append(nb.primaryInfos, nb.currentInfo)
	} else {
		nb.subtreeInfos = append(nb.subtreeInfos, nb.currentInfo)
	}
	return nb
}

func (nb *notificationInfoBuilder) PathKey(name, value string) *notificationInfoBuilder {
	path.SetKeyAt(nb.currentPath, nb.currentIndx, name, value)
	return nb
}

func (nb *notificationInfoBuilder) Table(dbno db.DBNum, tableName string) *notificationInfoBuilder {
	nb.currentInfo.dbno = dbno
	nb.currentInfo.table = &db.TableSpec{Name: tableName}
	if dbno == db.CountersDB {
		nb.OnChange(false)
	}
	return nb
}

func (nb *notificationInfoBuilder) Key(keyComp ...string) *notificationInfoBuilder {
	nb.currentInfo.key = &db.Key{Comp: keyComp}
	return nb
}

func (nb *notificationInfoBuilder) FieldScan(fieldPattern string) *notificationInfoBuilder {
	nb.currentInfo.fieldScanPattern = fieldPattern
	if nb.currentInfo.key == nil {
		nb.currentInfo.key = new(db.Key) // non-nul key is required to mark it as a db mapping
	}
	return nb
}

func (nb *notificationInfoBuilder) Field(yangAttr, dbField string) *notificationInfoBuilder {
	// Ignore unwanted fields
	if len(nb.fieldFilter) != 0 {
		if yangAttr != nb.fieldFilter {
			return nb
		}

		// When request path points to a leaf, we do not want the
		// yang leaf name in the fields map!!
		yangAttr = ""
	}

	isAdded := false
	for _, dbFldYgPath := range nb.currentInfo.dbFldYgPathInfoList {
		if dbFldYgPath.rltvPath == nb.fieldPrefix {
			if v, exists := dbFldYgPath.dbFldYgPathMap[dbField]; exists {
				dbFldYgPath.dbFldYgPathMap[dbField] = v + "," + yangAttr
			} else {
				dbFldYgPath.dbFldYgPathMap[dbField] = yangAttr
			}
			isAdded = true
			break
		}
	}

	if !isAdded {
		dbFldInfo := dbFldYgPathInfo{nb.fieldPrefix, make(map[string]string)}
		dbFldInfo.dbFldYgPathMap[dbField] = yangAttr
		nb.currentInfo.dbFldYgPathInfoList = append(nb.currentInfo.dbFldYgPathInfoList, &dbFldInfo)
	}

	return nb
}

func (nb *notificationInfoBuilder) SetFieldPrefix(prefix string) bool {
	i := nb.currentIndx + 1
	n := path.Len(nb.currentPath)

	// SetFieldPrefix("") indicates terminal container
	if len(prefix) == 0 {
		if i < n {
			nb.fieldFilter = nb.currentPath.Elem[i].Name
		}
		nb.fieldPrefix = strings.Join(nb.subtreePath, "/")
		return true
	}

	if len(nb.subtreePath) > 0 {
		prefix = strings.Join(nb.subtreePath, "/") + "/" + prefix
	}
	if i >= n {
		// Request does not contain any additional elements beyond
		// current path. Accept all sub containers & fields
		nb.fieldPrefix = prefix
		nb.fieldFilter = ""
		return true
	}

	pparts := strings.Split(prefix, "/")
	for j, p := range pparts {
		if p != nb.currentPath.Elem[i].Name {
			return false
		}

		i++
		if i >= n {
			if j == len(pparts) { // exact match
				nb.fieldPrefix = ""
			} else { // partial match
				nb.fieldPrefix = strings.Join(pparts[j+1:], "/")
			}
			nb.fieldFilter = ""
			return true
		}
	}

	// Current path is still longer than given prefix. Must be
	// field name filter
	nb.fieldPrefix = ""
	nb.fieldFilter = nb.currentPath.Elem[i].Name
	return true
}

func (nb *notificationInfoBuilder) OnChange(flag bool) *notificationInfoBuilder {
	nb.currentInfo.isOnChangeSupported = flag
	return nb
}

func (nb *notificationInfoBuilder) MinInterval(secs int) *notificationInfoBuilder {
	nb.currentInfo.mInterval = secs
	return nb
}

func (nb *notificationInfoBuilder) Preferred(mode NotificationType) *notificationInfoBuilder {
	nb.currentInfo.pType = mode
	return nb
}

func (nb *notificationInfoBuilder) HandlerFunc(f apis.ProcessOnChange) *notificationInfoBuilder {
	nb.currentInfo.handlerFunc = f
	return nb
}

func (nb *notificationInfoBuilder) Opaque(o interface{}) *notificationInfoBuilder {
	nb.currentInfo.opaque = o
	return nb
}

func (y *yangMapTree) match(reqPath *gnmi.Path, index int) (int, *yangMapTree, bool) {
	size := path.Len(reqPath)
	if len(y.subtree) == 0 || index >= size {
		return index - 1, y, true
	}

	next := reqPath.Elem[index].Name

	for segment, submap := range y.subtree {
		parts := strings.Split(segment, "/")
		if parts[0] != next {
			continue
		}

		// Reuse current handler func if subtree map is nil.
		if submap == nil {
			temp := yangMapTree{mapFunc: y.mapFunc}
			return temp.match(reqPath, index)
		}

		nparts := len(parts)
		if path.MergeElemsAt(reqPath, index, parts...) == nparts {
			return submap.match(reqPath, index+nparts)
		}
		break // no match
	}

	// There are no subtree mappings matching the request path.
	// Use the current node's func, but do not recurse into subtree.
	if y.mapFunc != nil {
		return index - 1, y, false
	}

	return -1, nil, false
}

func (y *yangMapTree) collect(nb *notificationInfoBuilder, recurse bool) error {
	// Reset previous states
	nb.fieldPrefix = ""
	nb.fieldFilter = ""
	bakupIndx := nb.currentIndx
	bakupPath := nb.currentPath
	bakupDepth := nb.treeDepth

	// Invoke yangMapFunc to collect notificationAppInfo
	if y.mapFunc != nil {
		if err := y.mapFunc(nb); err != nil {
			return err
		}
		if nb.currentInfo != nil {
			nb.treeDepth++
		}
	}
	if !recurse {
		return nil
	}

	baseNInfo := nb.currentInfo

	// Recursively collect from subtree
	for subpath, subnode := range y.subtree {
		if subnode == nil {
			continue
		}

		parts := strings.Split(subpath, "/")
		nb.currentIndx = bakupIndx + len(parts)
		nb.currentPath = path.SubPath(bakupPath, 0, bakupIndx+1)
		path.AppendElems(nb.currentPath, parts...)

		if nb.currentInfo == baseNInfo {
			nb.subtreePath = append(nb.subtreePath, subpath)
		}

		if err := subnode.collect(nb, true); err != nil {
			return err
		}

		if len(nb.subtreePath) != 0 {
			nb.subtreePath = nb.subtreePath[:len(nb.subtreePath)-1]
		}
	}

	nb.treeDepth = bakupDepth
	return nil
}

func wildcardMatch(v1, v2 string) bool {
	return v1 == v2 || v1 == "*"
}

// emptySubscribeResponse returns a translateSubResponse containing a non-db mapping
// for the given path
func emptySubscribeResponse(reqPath string) (translateSubResponse, error) {
	p, err := ygot.StringToStructuredPath(reqPath)
	if err != nil {
		return translateSubResponse{}, err
	}
	appInfo := &notificationAppInfo{
		path:                p,
		dbno:                db.MaxDB, // non-DB
		isOnChangeSupported: false,
	}
	return translateSubResponse{
		ntfAppInfoTrgt: []*notificationAppInfo{appInfo},
	}, nil
}
