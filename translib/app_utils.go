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
	"reflect"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func getYangPathFromUri(uri string) (string, error) {
	var path *gnmi.Path
	var err error

	path, err = ygot.StringToPath(uri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		log.Errorf("Error in uri to path conversion: %v", err)
		return "", err
	}

	yangPath, yperr := ygot.PathToSchemaPath(path)
	if yperr != nil {
		log.Errorf("Error in Gnmi path to Yang path conversion: %v", yperr)
		return "", yperr
	}

	return yangPath, err
}

func getYangPathFromYgotStruct(s ygot.GoStruct, yangPathPrefix string, appModuleName string) string {
	tn := reflect.TypeOf(s).Elem().Name()
	schema, ok := ocbinds.SchemaTree[tn]
	if !ok {
		log.Errorf("could not find schema for type %s", tn)
		return ""
	} else if schema != nil {
		yPath := schema.Path()
		//yPath = strings.Replace(yPath, "/device/acl", "/openconfig-acl:acl", 1)
		yPath = strings.Replace(yPath, yangPathPrefix, appModuleName, 1)
		return yPath
	}
	return ""
}

func generateGetResponse(targetUri string, root *ygot.GoStruct, fmtType TranslibFmtType) (GetResponse, error) {
	var err error
	var resp GetResponse

	if len(targetUri) == 0 {
		return resp, tlerr.InvalidArgs("generateGetResponse failed as target Uri is not valid")
	}
	path, err := ygot.StringToPath(targetUri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		return resp, tlerr.InvalidArgs("URI to path conversion failed: %v", err)
	}

	// Get current node (corresponds to ygotTarget) and its parent node
	var pathList []*gnmi.PathElem = path.Elem
	parentPath := &gnmi.Path{}
	for i := 0; i < len(pathList); i++ {
		if log.V(3) {
			log.Infof("pathList[%d]: %s\n", i, pathList[i])
		}
		pathSlice := strings.Split(pathList[i].Name, ":")
		pathList[i].Name = pathSlice[len(pathSlice)-1]
		if i < (len(pathList) - 1) {
			parentPath.Elem = append(parentPath.Elem, pathList[i])
		}
	}
	parentNodeList, err := ytypes.GetNode(ygSchema.RootSchema(), *root, parentPath)
	if err != nil {
		return resp, err
	}
	if len(parentNodeList) == 0 {
		return resp, tlerr.InvalidArgs("Invalid URI: %s", targetUri)
	}
	parentNode := parentNodeList[0].Data

	currentNodeList, ygerr := ytypes.GetNode(ygSchema.RootSchema(), *root, path, &(ytypes.GetPartialKeyMatch{}))
	if ygerr != nil {
		log.Errorf("Error from ytypes.GetNode: %v", ygerr)
		if status.Convert(ygerr).Code() == codes.NotFound {
			return resp, tlerr.NotFound("Resource not found")
		} else {
			return resp, ygerr
		}
	}
	if len(currentNodeList) == 0 {
		return resp, tlerr.NotFound("Resource not found")
	}
	//currentNode := currentNodeList[0].Data
	currentNodeYangName := currentNodeList[0].Schema.Name

	// Create empty clone of parent node
	parentNodeClone := reflect.New(reflect.TypeOf(parentNode).Elem())
	var parentCloneObj ygot.ValidatedGoStruct
	var ok bool
	if parentCloneObj, ok = (parentNodeClone.Interface()).(ygot.ValidatedGoStruct); ok {
		ygot.BuildEmptyTree(parentCloneObj)
		pcType := reflect.TypeOf(parentCloneObj).Elem()
		pcValue := reflect.ValueOf(parentCloneObj).Elem()

		var currentNodeOCFieldName string
		for i := 0; i < pcValue.NumField(); i++ {
			fld := pcValue.Field(i)
			fldType := pcType.Field(i)
			if fldType.Tag.Get("path") == currentNodeYangName {
				currentNodeOCFieldName = fldType.Name
				// Take value from original parent and set in parent clone
				valueFromParent := reflect.ValueOf(parentNode).Elem().FieldByName(currentNodeOCFieldName)
				fld.Set(valueFromParent)
				break
			}
		}
		if log.V(3) {
			log.Infof("Target yang name: %s  OC Field name: %s\n", currentNodeYangName, currentNodeOCFieldName)
		}
	}
	if fmtType == TRANSLIB_FMT_YGOT {
		resp.ValueTree = parentCloneObj
		return resp, err
	}

	resp.Payload, err = dumpIetfJson(parentCloneObj)

	return resp, err
}

func getTargetNodeYangSchema(targetUri string, deviceObj *ocbinds.Device) (*yang.Entry, error) {
	if len(targetUri) == 0 {
		return nil, tlerr.InvalidArgs("GetResponse failed as target Uri is not valid")
	}
	path, err := ygot.StringToPath(targetUri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		return nil, tlerr.InvalidArgs("URI to path conversion failed: %v", err)
	}
	// Get current node (corresponds to ygotTarget)
	var pathList []*gnmi.PathElem = path.Elem
	for i := 0; i < len(pathList); i++ {
		if log.V(3) {
			log.Infof("pathList[%d]: %s\n", i, pathList[i])
		}
		pathSlice := strings.Split(pathList[i].Name, ":")
		pathList[i].Name = pathSlice[len(pathSlice)-1]
	}
	targetNodeList, err := ytypes.GetNode(ygSchema.RootSchema(), deviceObj, path, &(ytypes.GetPartialKeyMatch{}))
	if err != nil {
		return nil, tlerr.InvalidArgs("Getting node information failed: %v", err)
	}
	if len(targetNodeList) == 0 {
		return nil, tlerr.NotFound("Resource not found")
	}
	targetNodeSchema := targetNodeList[0].Schema
	//targetNode := targetNodeList[0].Data
	if log.V(3) {
		log.Infof("Target node yang name: %s\n", targetNodeSchema.Name)
	}
	return targetNodeSchema, nil
}

func dumpIetfJson(s ygot.ValidatedGoStruct) ([]byte, error) {
	cfg := ocbinds.EmitJSONOptions{
		SortList: true,
	}
	return ocbinds.EmitJSON(s, &cfg)
}

func contains(sl []string, str string) bool {
	for _, v := range sl {
		if v == str {
			return true
		}
	}
	return false
}

func removeElement(sl []string, str string) []string {
	for i := 0; i < len(sl); i++ {
		if sl[i] == str {
			sl = append(sl[:i], sl[i+1:]...)
			i--
			sl = sl[:len(sl)]
			break
		}
	}
	return sl
}

// isNotFoundError return true if the error is a 'not found' error
func isNotFoundError(err error) bool {
	switch err.(type) {
	case tlerr.TranslibRedisClientEntryNotExist, tlerr.NotFoundError:
		return true
	default:
		return false
	}
}

// asKey cretaes a db.Key from given key components
func asKey(parts ...string) db.Key {
	return db.Key{Comp: parts}
}

func createEmptyDbValue(fieldName string) db.Value {
	return db.Value{Field: map[string]string{fieldName: ""}}
}
