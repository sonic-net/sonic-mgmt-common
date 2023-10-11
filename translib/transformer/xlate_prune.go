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

package transformer

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	ygutil "github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func getPruneObjNode(ygRoot *ygot.GoStruct, uri string, requestUri string,
	queryParams *QueryParams) ([]reflect.Value, int, string, error) {

	var err error
	var curValList []reflect.Value

	log.V(3).Infof("getPruneObjNode: URI %s, requestUri %s, queryParams %s",
		uri, requestUri, queryParams)

	xpath, _, _ := XfmrRemoveXPATHPredicates(uri)

	// Split (predicate-less)xpath, requestUri, and uri into element arrays
	xpathList := strings.Split(xpath, "/")
	ruList := strings.Split(requestUri, "/")

	// The fields are w.r.t the requestUri. Thus while indexing into
	// QueryParams.fields array, need to adjust the curDepth by depthDiff
	depthDiff := len(xpathList) - len(ruList)
	pruneXpath := xpath

	path, err := ygot.StringToPath(uri, ygot.StructuredPath,
		ygot.StringSlicePath)
	if err != nil {
		log.Errorf("getPruneObjNode: StringToPath err: %s", err)
		return curValList, depthDiff, pruneXpath, err
	}

	for _, p := range path.Elem {
		pathSlice := strings.Split(p.Name, ":")
		p.Name = pathSlice[len(pathSlice)-1]
	}

	ygSchema, err := ocbinds.GetSchema()
	if err != nil {
		log.Errorf("getPruneObjNode: GetSchema err: %s", err)
		return curValList, depthDiff, pruneXpath, err
	}
	treeNodeList, err := ytypes.GetNode(ygSchema.RootSchema(),
		*ygRoot, path, &ytypes.GetPartialKeyMatch{})
	if err != nil {
		log.Errorf("getPruneObjNode: GetNode err: %s", err)
		return curValList, depthDiff, pruneXpath, err
	}

	curValList = make([]reflect.Value, len(treeNodeList))
	for i, tn := range treeNodeList {
		curValList[i] = reflect.ValueOf(tn.Data)
	}

	if log.V(5) {
		log.Infof("getPruneObjNode: depthDiff: %d pruneXpath: %s err: %s returns:\n%s",
			depthDiff, pruneXpath, err, pretty.Sprint(curValList))
	}

	return curValList, depthDiff, pruneXpath, err
}

// Prune the ygRoot after running a subtree transformer.
// ygRoot: as returned from GET subtree callback (I/O)
// qp: QueryParams structure that was passed to GET subtree callback (I)
// uri: uri path/point where GET subtree callback was called during GET traversal. (I)
// requestUri: uri with which the North Bound Client side request was made (I)
func xfmrPruneQP(ygRoot *ygot.GoStruct, queryParams QueryParams, uri string,
	requestUri string) error {

	ts := time.Now()

	pruneObjs, depthDiff, pruneXpath, err := getPruneObjNode(
		ygRoot, uri, requestUri, &queryParams)

	if err != nil {
		log.Errorf("xfmrPruneQP: Couldn't get prune Obj: uri(\"%v\")"+
			"requestUri(\"%v\") error(\"%v\").", uri, requestUri, err)
		return err
	}

	for _, val := range pruneObjs {
		if log.V(8) {
			log.Infof("xfmrPruneQP: B4: pruneObj:\n%s",
				pretty.Sprint(val.Interface()))
		}

		err = pruneYGObj(val, pruneXpath, &queryParams, 1, depthDiff, nil)
		if err != nil {
			break
		}

		if log.V(8) {
			log.Infof("xfmrPruneQP: After: pruneObj:\n%s",
				pretty.Sprint(val.Interface()))
		}
	}

	tt := time.Since(ts)
	GetPruneQPStats().add(tt, uri)
	log.Infof("xfmrPruneQP: URI %v, requestUri %v, TimeTaken %s",
		uri, requestUri, tt)
	log.Infof("xfmrPruneQP: Totals: %s", GetPruneQPStats())

	return err
}

// Prunes Ygot.GoStruct
func pruneYGObj(val reflect.Value, xpath string, queryParams *QueryParams,
	pruneDepth uint, depthDiff int, ctxt context.Context) error {

	// isReqContextCancelled(ctxt) is in a different PR
	if ctxt != nil {
		log.Warningf("pruneYGObj: Replace with isReqContextCancelled()")
		return nil
	}

	if (pruneDepth == 1) || log.V(4) {
		log.Infof("pruneYGObj: Pruning: xpath %v, pruneDepth %d depthDiff %d",
			xpath, pruneDepth, depthDiff)
	}

	if ygutil.IsValueNil(val) {
		log.Infof("pruneYGObj: Bypassing nil node: %s, %v", xpath, val)
		return nil
	}

	if !val.IsValid() {
		log.Infof("pruneYGObj: Bypassing !Valid node:%s", xpath)
		return nil
	}

	if !ygutil.IsValueStructPtr(val) {
		log.V(3).Infof("pruneYGObj: Keeping leaf(y) node: %s type: %v val: %v",
			xpath, val.Kind(), val)
		return nil
	}

	fvals := val.Elem()
	ftypes := fvals.Type()

	// Filters each member(child node) of the struct.
	for i := 0; i < fvals.NumField(); i++ {
		ft := ftypes.Field(i)

		if ygutil.IsYgotAnnotation(ft) {
			continue
		}

		pname, ok := ft.Tag.Lookup("path")
		if !ok {
			continue
		}

		fv := fvals.Field(i)
		if !fv.IsValid() || fv.IsZero() {
			log.V(6).Infof("pruneYGObj: Skipping zero value node:",
				xpath, "/", pname)
			continue
		}

		// Root node is special. All children are modules.
		if xpath == "" {
			mname, ok := ft.Tag.Lookup("module")
			if !ok {
				continue
			}
			// For root, the first level xpath is module-name:path.
			mcPath := "/" + mname + ":" + pname
			err := pruneYGObj(fv, mcPath, queryParams, pruneDepth+1, depthDiff, ctxt)
			if err != nil {
				return err
			}
			continue
		}

		chldXpath := xpath + "/" + pname

		if keep, keepSubtree := matchQueryParametersByXpath(chldXpath,
			queryParams, pruneDepth, depthDiff); !keep || keepSubtree {

			if !keep {
				log.V(3).Infof("pruneYGObj: Removing node: %s type: %v",
					chldXpath, fv.Kind())
				fv.Set(reflect.Zero(fv.Type()))
			}

			if keepSubtree {
				log.V(3).Infof("pruneYGObj: Keeping subtree: %s type: %v",
					chldXpath, fv.Kind())
			}

			continue
		}

		switch fv.Kind() {
		case reflect.Map:
			// If the depth pruning causes inclusion of just the key,
			// but not the values, then the resultant map element will be
			// nil, and not even include the key in the attributes.
			// Eg(.../subinterfaces/subinterface[name=0]/, but
			//    no .../subinterfaces/subinterface[name=0]/name)
			// To avoid this situation, check the depth at the Map level
			// itself, and delete all the MapIndex entirely.
			// Empty Containers/Lists are not allowed in any case.
			var deleteMapElements bool
			if queryParams.isDepthEnabled() &&
				(pruneDepth+1 >= queryParams.curDepth) {
				log.V(3).Infof("pruneYGObj: Trim chldXpath: %s pruneDepth: %d",
					chldXpath, pruneDepth)
				deleteMapElements = true
			}
			for _, k := range fv.MapKeys() {
				if deleteMapElements {
					fv.SetMapIndex(k, reflect.Value{})
					continue
				}
				v := fv.MapIndex(k)
				err := pruneYGObj(v, chldXpath, queryParams, pruneDepth+1, depthDiff, ctxt)
				if err != nil {
					return err
				}
			}
		case reflect.Slice:
			for i, fvLen := 0, fv.Len(); i < fvLen; i++ {
				v := fv.Index(i)
				err := pruneYGObj(v, chldXpath, queryParams, pruneDepth+1, depthDiff, ctxt)
				if err != nil {
					return err
				}
			}
		case reflect.Ptr:
			err := pruneYGObj(fv, chldXpath, queryParams, pruneDepth+1, depthDiff, ctxt)
			if err != nil {
				return err
			}
		default:
			log.V(3).Infof("pruneYGObj: Keeping node: %s type: %v",
				chldXpath, fv.Kind())
		}
	}

	if (pruneDepth == 1) || log.V(4) {
		log.Infof("pruneYGObj: Done Pruning: xpath %v", xpath)
	}
	return nil
}

// matchQueryParametersByXpath is operating under the simplyfying
// assumptions of transformer QP implementation: The only simultaneous
// combination of query parameters allowed is Depth, and Content.
// Note: Content=all with Fields is also allowed (because there is no
// filtering when Content=all)
// Return: keep, keepSubtree
//
//	keep        : bool : keep the node, or prune it.
//	keepSubtree : bool : keep the entire subtree, i.e. stop pruning
//	                     on this xpath.
func matchQueryParametersByXpath(xpath string, queryParams *QueryParams,
	pruneDepth uint, depthDiff int) (bool, bool) {

	log.V(6).Infof("matchQueryParametersByXpath: xpath %v, qP %v,"+
		"pruneDepth %d, depthDiff %d",
		xpath, queryParams, pruneDepth, depthDiff)

	yangNode, ok := xYangSpecMap[xpath]
	if ok && yangNode.yangType == YANG_MODULE {
		log.V(6).Infof("matchQueryParametersByXpath: Module: xpath %v", xpath)
		return true, false
	}

	yangEntry := getYangEntryForXPath(xpath)
	if yangEntry == nil {

		// If the Yang Specification is not found, we are in uncharted
		// water. Best to stop pruning.
		log.Warningf("matchQueryParametersByXpath: not found: %s", xpath)
		return true, true
	}

	if queryParams.isDepthEnabled() &&
		!matchDepth(yangNode, yangEntry, queryParams.curDepth, pruneDepth) {

		return false, false
	}

	if queryParams.isContentEnabled() &&
		!matchContent(yangNode, yangEntry, queryParams.content) {

		return false, false
	}

	if queryParams.isFieldsEnabled() {
		if keep, keepSubtree := matchFields(yangNode, yangEntry, queryParams.fields,
			pruneDepth, depthDiff); !keep || keepSubtree {
			return keep, keepSubtree
		}
	}

	return true, false
}

func matchDepth(yangNode *yangXpathInfo, yangEntry *yang.Entry, depth uint,
	pruneDepth uint) bool {

	log.V(6).Infof("matchDepth: Name %v, depth %d, pruneDepth %d",
		yangEntry.Name, depth, pruneDepth)

	return pruneDepth < depth
}

func isStateEntry(yangEntry *yang.Entry) bool {

	if yangEntry == nil {
		return false
	}

	if yangEntry.ReadOnly() {
		return true
	}

	// leaf is writable, therefore not STATE entry.
	if yangEntry.IsLeaf() {
		return false
	}

	// writable "config" container is CONFIG.
	if yangEntry.IsContainer() && yangEntry.Name == "config" {
		return false
	}

	// Non-Leaf writable node could be STATE entry.
	return true
}

func isOperationalEntry(yangEntry *yang.Entry) bool {

	if !isStateEntry(yangEntry) {
		log.V(6).Infof("isOperationalEntry: OPERATIONAL: Omit CONFIG path:")
		return false
	}

	// Non-leaf writable could be OPERATIONAL
	if !yangEntry.IsLeaf() {
		return true
	}

	// Corresponding config path exists: it is not an operational node.
	cfgEntry := yangEntry.Find("../../config/" + yangEntry.Name)
	if cfgEntry != nil && cfgEntry.IsLeaf() && !cfgEntry.ReadOnly() {
		log.V(6).Infof("matchContent: OPERATIONAL: Omit non-operational STATE path:")
		return false
	}

	return true
}

// matchContent based on ContentType/GNMI Get (type) filter
// TBD: It may be possible to optimise similar to matchFields()
//
//	using keep, keepSubtree prune control returns.
func matchContent(yangNode *yangXpathInfo, yangEntry *yang.Entry,
	content ContentType) bool {

	if (yangNode != nil) && yangNode.isKey {
		log.V(3).Infof("matchContent: path %v, isKey %v", yangNode.fieldName,
			yangNode.isKey)
		return true
	}

	path := yangEntry.Path()
	log.V(6).Infof("matchContent path %v content %v", path, content)

	switch content {
	case QUERY_CONTENT_CONFIG:
		if yangEntry.ReadOnly() {
			log.V(6).Infof("matchContent: CONFIG: Omit STATE path: %v", path)
			return false
		}
	case QUERY_CONTENT_NONCONFIG:
		if !isStateEntry(yangEntry) {
			log.V(6).Infof("matchContent: STATE: Omit CONFIG path: %v", path)
			return false
		}
	case QUERY_CONTENT_OPERATIONAL:
		if !isOperationalEntry(yangEntry) {
			log.V(6).Infof("matchContent: OPERATIONAL: Omit CONFIG/STATE path: %v",
				path)
			return false
		}
	}

	return true
}

// matchFields is operating under the simplyfying assumptions of transformer QP
// implementation: FIELDS having list(with or without key) is not supported
// Return: keep, keepSubtree
//
//	keep        : bool : keep the node(true), or prune it(false).
//	keepSubtree : bool : keep the entire subtree, i.e. stop pruning(true)
func matchFields(yangNode *yangXpathInfo, yangEntry *yang.Entry, fields []string,
	pruneDepth uint, depthDiff int) (bool, bool) {

	log.V(6).Infof("matchFields: yangEntry.Name %v, fields %v, pruneDepth %d, depthDiff %d",
		yangEntry.Name, fields, pruneDepth, depthDiff)

	var keep, keepSubtree bool

	// Adjust depth into fields.
	depth := int(pruneDepth) + depthDiff - 1

	if (yangNode != nil) && yangNode.isKey {
		log.V(3).Infof("matchFields: Name %v key %v", yangNode.fieldName, true)
		return true, true
	}

	log.V(6).Infof("matchFields: Name %v", yangEntry.Name)

	for _, field := range fields {

		log.V(8).Infof("matchFields: field %v", field)
		fieldList := strings.Split(field, "/")
		if len(fieldList) <= depth {
			continue
		}

		if yangEntry.Name == fieldList[depth] {

			keep = true
			keepSubtree = (depth == (len(fieldList) - 1))

			log.V(8).Infof("matchFields: Match %v keepSubtree %v",
				yangEntry.Name, keepSubtree)

			if keepSubtree {
				// Our pruning task is done for this path
				break
			}

			// Evaluate whether pruning has ended after looking at all fields
		}
	}

	log.V(6).Infof("matchFields: Name %v keep %v keepSubtree %v",
		yangEntry.Name, keep, keepSubtree)

	return keep, keepSubtree
}
