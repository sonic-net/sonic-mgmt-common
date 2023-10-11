////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
//go:build prune_xfmrtest
// +build prune_xfmrtest

package transformer

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

var ygSchema *ytypes.Schema

// Modelled (largely taken from) request_binder.go
// request_binder.go is in translib which is a parent package of transformer
// so, to avoid circular dependencies for unit testing code, we use a copy
// here.
type rB struct {
	uri                *string
	payload            *[]byte
	appRootNodeType    *reflect.Type
	pathParent         *gnmi.Path
	targetNodePath     *gnmi.Path
	targetNodeSchema   *yang.Entry
	targetNodeListInst bool
	isSonicModel       bool
}

func getRB(uri *string, payload *[]byte, appRootNodeType *reflect.Type) *rB {
	return &rB{uri, payload, appRootNodeType, nil, nil, nil, false, false}
}

// Modelled from request_binder.go for testing
func (rb *rB) unMarshallUri(t *testing.T, deviceObj *ocbinds.Device) (*interface{}, error) {

	var path *gnmi.Path
	var err error

	path, err = ygot.StringToPath(*rb.uri, ygot.StructuredPath,
		ygot.StringSlicePath)
	if err != nil {
		t.Error("Error in uri to path conversion: ", err)
		return nil, err
	}

	for _, p := range path.Elem {
		pathSlice := strings.Split(p.Name, ":")
		p.Name = pathSlice[len(pathSlice)-1]
	}

	targetPath := path
	var pathList []*gnmi.PathElem = path.Elem
	pathLen := len(pathList)

	if len(pathList[pathLen-1].Key) > 0 {
		rb.targetNodeListInst = true
	}

	gpath := &gnmi.Path{}
	for i := 0; i < (pathLen - 1); i++ {
		gpath.Elem = append(gpath.Elem, pathList[i])
	}

	rb.targetNodePath = &gnmi.Path{}
	rb.targetNodePath.Elem = append(rb.targetNodePath.Elem, pathList[(pathLen-1)])
	t.Logf("unMarshallUri: modified path is: %v\n", gpath)

	rb.pathParent = gpath

	if rb.targetNodeListInst {
		targetPath = rb.pathParent
	}

	ygNode, ygEntry, errYg := ytypes.GetOrCreateNode(ygSchema.RootSchema(),
		deviceObj, targetPath)
	if errYg != nil {
		t.Error("Error in creating the target object: ", errYg)
		return nil, errYg
	} else {
		rb.targetNodeSchema = ygEntry
	}

	return &ygNode, nil
}

func (rb *rB) unMarshallPayload(t *testing.T, workObj *interface{}) error {
	targetObj, ok := (*workObj).(ygot.GoStruct)
	if !ok {
		t.Error("Error in casting the target object")
		return errors.New("Error in casting the target object")
	}

	if err := ocbinds.Unmarshal(*(rb.payload), targetObj); err != nil {
		t.Error("ocbinds.Unmarshal: ", err)
		return err
	}

	return nil
}

func getWorkObj(t *testing.T, uri *string, payload *[]byte,
	appRootNodeType *reflect.Type) (*interface{}, *interface{}) {

	var deviceObj ocbinds.Device = ocbinds.Device{}

	rb := getRB(uri, payload, appRootNodeType)
	workObj, _ := rb.unMarshallUri(t, &deviceObj)

	rootIntf := reflect.ValueOf(&deviceObj).Interface()
	// ygotObj := rootIntf.(ygot.GoStruct)
	// var ygotRootObj *ygot.GoStruct = &ygotObj

	var tmpTargetNode *interface{}
	var ygEntry *yang.Entry

	if rb.pathParent != nil && !rb.targetNodeListInst {
		treeNodeList, err := ytypes.GetNode(ygSchema.RootSchema(), &deviceObj, rb.pathParent)
		if err != nil {
			t.Error("getWorkObj: ytype.GetNode() err: ", err)
			return nil, nil
		}

		if len(treeNodeList) == 0 {
			t.Error("getWorkObj: treeNodeList: empty")
			return nil, nil
		}

		tmpTargetNode = &(treeNodeList[0].Data)
		ygEntry = treeNodeList[0].Schema
	} else {
		tmpTargetNode = workObj
		ygEntry = rb.targetNodeSchema
	}

	if err := rb.unMarshallPayload(t, tmpTargetNode); err != nil {
		return nil, nil
	}

	if ygEntry != nil {
		var workObjIntf interface{}
		if ygEntry.IsContainer() && !rb.targetNodeListInst {
			v := reflect.ValueOf(*tmpTargetNode).Elem()
			for i := 0; i < v.NumField(); i++ {
				ft := v.Type().Field(i)
				tagVal, _ := ft.Tag.Lookup("path")
				if len(rb.targetNodePath.Elem) > 0 && tagVal == rb.targetNodePath.Elem[0].Name {
					fv := v.Field(i)
					workObjIntf = fv.Interface()
					break
				}
			}
		} else if ygEntry.IsList() || rb.targetNodeListInst {
			if treeNodeList, err := ytypes.GetNode(ygEntry, *tmpTargetNode, rb.targetNodePath); err != nil {
				t.Error("getWorkObj: targetNodeList, treeNodeList: err: ", err)
				return nil, nil
			} else if len(treeNodeList) == 0 {
				t.Error("getWorkObj: targetNodeList, treeNodeList: empty")
				return nil, nil
			} else {
				workObjIntf = treeNodeList[0].Data
			}
		}

		if workObjIntf != nil {
			workObj = &workObjIntf
		} else {
			t.Error("getWorkObj: Target node not found.")
		}
	}

	return workObj, &rootIntf
}

// areEqual is taken from common_app.go for testing purposes
func areEqual(a, b interface{}) bool {
	if util.IsValueNil(a) && util.IsValueNil(b) {
		return true
	}
	if util.IsValueNil(a) || util.IsValueNil(b) {
		return false
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr && vb.Kind() == reflect.Ptr {
		return reflect.DeepEqual(va.Elem().Interface(), vb.Elem().Interface())
	}

	return reflect.DeepEqual(a, b)
}

// areEqual is taken from common_app.go for testing purposes
func areEqualSprint(a, b interface{}) bool {
	if util.IsValueNil(a) && util.IsValueNil(b) {
		return true
	}
	if util.IsValueNil(a) || util.IsValueNil(b) {
		return false
	}
	aSprint := pretty.Sprint(a)
	bSprint := pretty.Sprint(b)
	return aSprint == bSprint
}

type xpTests struct {
	tid           string
	uri           string
	requestUri    string
	payload       []byte
	appRootType   reflect.Type
	queryParams   QueryParams
	prunedPayload []byte
}

var xp_tests []xpTests

func TestXfmrPruneQP(t *testing.T) {

	var err error
	if ygSchema, err = ocbinds.GetSchema(); err != nil {
		t.Error("Error in ocbinds.GetSchema(): ", err)
	}

	for _, tt := range xp_tests {

		t.Logf("TestXfmrPruneQP: Test Case %s: Start\n", tt.tid)

		workObj, rootObj := getWorkObj(t, &tt.requestUri, &tt.payload, &tt.appRootType)
		if workObj == nil || rootObj == nil {
			t.Error("Cannot retrieve workObj/rootObj for the request")
			break
		}
		ygRootNode, ok := (*rootObj).(ygot.GoStruct)
		if !ok {
			t.Error("Cannot convert workObj to ygot.GoStruct")
		}

		if pruneQPErr := xfmrPruneQP(&ygRootNode, tt.queryParams, tt.uri,
			tt.requestUri); pruneQPErr != nil {
			t.Error("xfmrPruneQP: pruneQPErr: ", pruneQPErr)
		}
		prunedWorkObj, _ := getWorkObj(t, &tt.requestUri,
			&tt.prunedPayload, &tt.appRootType)

		// compare prunedWorkObj with workObj and print a diff?
		if !areEqualSprint(workObj, prunedWorkObj) {
			t.Logf("\nTestXfmrPruneQP: QP:", tt.queryParams)
			t.Logf("\nTestXfmrPruneQP: Got:\n", pretty.Sprint(workObj))
			t.Logf("\nTestXfmrPruneQP: Exp:\n", pretty.Sprint(prunedWorkObj))
			t.Errorf("TestXfmrPruneQP: Test Case %s: Fail\n", tt.tid)
		}
		t.Logf("TestXfmrPruneQP: Test Case %s: End\n", tt.tid)
	}
}
