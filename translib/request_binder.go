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

package translib

import (
	"errors"
	"reflect"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

const (
	GET = 1 + iota
	CREATE
	REPLACE
	UPDATE
	DELETE
)

var ygSchema *ytypes.Schema

func init() {
	initSchema()
}

func initSchema() {
	log.Flush()
	var err error
	if ygSchema, err = ocbinds.GetSchema(); err != nil {
		panic("Error in getting the schema: " + err.Error())
	}
}

type requestBinder struct {
	uri                *string
	payload            *[]byte
	opcode             int
	appRootNodeType    *reflect.Type
	pathParent         *gnmi.Path
	targetNodePath     *gnmi.Path
	targetNodeSchema   *yang.Entry
	targetNodeListInst bool
	isSonicModel       bool
}

func getRequestBinder(uri *string, payload *[]byte, opcode int, appRootNodeType *reflect.Type) *requestBinder {
	return &requestBinder{uri, payload, opcode, appRootNodeType, nil, nil, nil, false, false}
}

func (binder *requestBinder) unMarshallPayload(workObj *interface{}) error {
	targetObj, ok := (*workObj).(ygot.GoStruct)
	if !ok {
		err := errors.New("Error in casting the target object")
		log.Error(err)
		return tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
	}

	if len(*binder.payload) == 0 {
		err := errors.New("Request payload is empty")
		log.Error(err)
		return tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
	}
	if log.V(5) {
		log.Info("Ygot target object:")
		pretty.Print(targetObj)
	}
	err := ocbinds.Unmarshal(*binder.payload, targetObj)
	if err != nil {
		log.Error(err)
		return tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
	}

	return nil
}

func (binder *requestBinder) validateObjectType(errObj error) error {

	if errObj == nil {
		return nil
	}

	errStr := errObj.Error()

	if binder.opcode == GET || binder.isSonicModel {
		tmpStr := strings.Replace(errStr, "ERROR_READONLY_OBJECT_FOUND", "", -1)
		if len(tmpStr) > 0 {
			log.Info("validateObjectType ==> GET == return err string ==> ", tmpStr)
			return errors.New(tmpStr)
		} else {
			return nil
		}
	} else {
		if strings.Contains(errStr, "ERROR_READONLY_OBJECT_FOUND") {
			log.Info("validateObjectType ==> WRITE == return err string")
			return errors.New("SET operation not allowed on the read-only object")
		} else {
			log.Info("validateObjectType ==> WRITE == return err string")
			return errors.New(errStr)
		}
	}
}

func (binder *requestBinder) validateRequest(deviceObj *ocbinds.Device) error {

	// Skipping the validation for the sonic yang model
	if binder.isSonicModel {
		log.Warning("Translib: RequestBinder: Skipping the vaidatiion of the given sonic yang model request..")
		return nil
	}

	if binder.pathParent == nil || len(binder.pathParent.Elem) == 0 {
		if binder.opcode == UPDATE || binder.opcode == REPLACE {
			err := deviceObj.Validate(&ytypes.LeafrefOptions{IgnoreMissingData: true})
			err = binder.validateObjectType(err)
			if err != nil {
				return err
			}
			return nil
		} else {
			return errors.New("Path is empty")
		}
	}

	path, err := ygot.StringToPath(binder.pathParent.Elem[0].Name, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		return err
	} else {
		baseTreeNode, err := ytypes.GetNode(ygSchema.RootSchema(), deviceObj, path)
		if err != nil {
			return err
		} else if len(baseTreeNode) == 0 {
			return errors.New("Invalid base URI node")
		} else {
			basePathObj, ok := (baseTreeNode[0].Data).(ygot.ValidatedGoStruct)
			if ok {
				err := basePathObj.Validate(&ytypes.LeafrefOptions{IgnoreMissingData: true})
				err = binder.validateObjectType(err)
				if err != nil {
					return err
				}
			} else {
				return errors.New("Invalid Object in the binding: Not able to cast to type ValidatedGoStruct")
			}
		}
	}

	return nil
}

func (binder *requestBinder) unMarshall() (*ygot.GoStruct, *interface{}, error) {
	var deviceObj ocbinds.Device = ocbinds.Device{}

	workObj, err := binder.unMarshallUri(&deviceObj)
	if err != nil {
		log.Error("Error in creating the target object : ", err)
		return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 404, ErrorStr: err}
	}

	rootIntf := reflect.ValueOf(&deviceObj).Interface()
	ygotObj := rootIntf.(ygot.GoStruct)
	var ygotRootObj *ygot.GoStruct = &ygotObj

	if binder.opcode == GET || binder.opcode == DELETE {
		return ygotRootObj, workObj, nil
	}

	switch binder.opcode {
	case CREATE:
		if reflect.ValueOf(*workObj).Kind() == reflect.Map {
			return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: errors.New("URI doesn't have keys in the request")}
		} else {
			err = binder.unMarshallPayload(workObj)
			if err != nil {
				return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
			}
		}

	case UPDATE, REPLACE:
		var tmpTargetNode *interface{}
		var ygEntry *yang.Entry
		if binder.pathParent != nil && !binder.targetNodeListInst {
			treeNodeList, err2 := ytypes.GetNode(ygSchema.RootSchema(), &deviceObj, binder.pathParent)
			if err2 != nil {
				return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err2}
			}

			if len(treeNodeList) == 0 {
				return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: errors.New("Invalid URI")}
			}

			tmpTargetNode = &(treeNodeList[0].Data)
			ygEntry = treeNodeList[0].Schema
		} else {
			tmpTargetNode = workObj
			ygEntry = binder.targetNodeSchema
		}
		err = binder.unMarshallPayload(tmpTargetNode)
		if err != nil {
			return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
		} else if ygEntry != nil {
			var workObjIntf interface{}
			if ygEntry.IsContainer() && !binder.targetNodeListInst {
				v := reflect.ValueOf(*tmpTargetNode).Elem()
				for i := 0; i < v.NumField(); i++ {
					ft := v.Type().Field(i)
					tagVal, _ := ft.Tag.Lookup("path")
					if len(binder.targetNodePath.Elem) > 0 && tagVal == binder.targetNodePath.Elem[0].Name {
						fv := v.Field(i)
						workObjIntf = fv.Interface()
						break
					}
				}
			} else if ygEntry.IsList() || binder.targetNodeListInst {
				if treeNodeList, err2 := ytypes.GetNode(ygEntry, *tmpTargetNode, binder.targetNodePath); err2 != nil {
					return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err2}
				} else if len(treeNodeList) == 0 {
					return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: errors.New("Invalid URI")}
				} else {
					workObjIntf = treeNodeList[0].Data
				}
			}

			if workObjIntf != nil {
				workObj = &workObjIntf
			} else {
				return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: errors.New("Target node not found.")}
			}
		}

		targetObj, ok := (*tmpTargetNode).(ygot.ValidatedGoStruct)
		if ok {
			if !binder.isSonicModel {
				err := targetObj.Validate(&ytypes.LeafrefOptions{IgnoreMissingData: true})
				err = binder.validateObjectType(err)
				if err != nil {
					return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
				}
			} else {
				log.Warning("Translib: Request binder: Valdation skipping for sonic yang model..")
			}
		}

	default:
		if binder.opcode != GET && binder.opcode != DELETE {
			return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: errors.New("Unknown HTTP METHOD in the request")}
		}
	}

	if binder.opcode != UPDATE && binder.opcode != REPLACE {
		if err = binder.validateRequest(&deviceObj); err != nil {
			return nil, nil, tlerr.TranslibSyntaxValidationError{StatusCode: 400, ErrorStr: err}
		}
	}

	if log.V(5) {
		log.Info("Ygot root object:")
		pretty.Print(ygotRootObj)
		log.Info("Ygot work object:")
		pretty.Print(workObj)
	}

	return ygotRootObj, workObj, nil
}

func (binder *requestBinder) getUriPath() (*gnmi.Path, error) {
	var path *gnmi.Path
	var err error

	path, err = ygot.StringToPath(*binder.uri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		log.Error("Error in uri to path conversion: ", err)
		return nil, err
	}

	return path, nil
}

func (binder *requestBinder) unMarshallUri(deviceObj *ocbinds.Device) (*interface{}, error) {
	if len(*binder.uri) == 0 {
		errMsg := errors.New("Error: URI is empty")
		log.Error(errMsg)
		return nil, errMsg
	}

	path, err := binder.getUriPath()
	if err != nil {
		return nil, err
	} else {
		binder.pathParent = path
	}

	for idx, p := range path.Elem {
		pathSlice := strings.Split(p.Name, ":")
		if idx == 0 && len(pathSlice) > 0 && strings.HasPrefix(pathSlice[0], "sonic-") {
			binder.isSonicModel = true
		}
		p.Name = pathSlice[len(pathSlice)-1]
	}

	targetPath := path

	switch binder.opcode {
	case UPDATE, REPLACE:
		var pathList []*gnmi.PathElem = path.Elem
		pathLen := len(pathList)

		if len(pathList[pathLen-1].Key) > 0 {
			binder.targetNodeListInst = true
		}

		gpath := &gnmi.Path{}
		for i := 0; i < (pathLen - 1); i++ {
			gpath.Elem = append(gpath.Elem, pathList[i])
		}

		binder.targetNodePath = &gnmi.Path{}
		binder.targetNodePath.Elem = append(binder.targetNodePath.Elem, pathList[(pathLen-1)])
		log.Info("requestBinder: modified path is: ", gpath)

		binder.pathParent = gpath

		if binder.targetNodeListInst {
			targetPath = binder.pathParent
		}
	}

	ygNode, ygEntry, errYg := ytypes.GetOrCreateNode(ygSchema.RootSchema(), deviceObj, targetPath)
	if errYg != nil {
		log.Error("Error in creating the target object: ", errYg)
		return nil, errYg
	} else {
		binder.targetNodeSchema = ygEntry
	}

	if (binder.opcode == GET || binder.opcode == DELETE) && (!ygEntry.IsLeaf() && !ygEntry.IsLeafList()) {
		if err = binder.validateRequest(deviceObj); err != nil {
			return nil, err
		}
	}

	return &ygNode, nil
}
