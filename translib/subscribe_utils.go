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
	"reflect"
	"sync/atomic"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/utils"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

var objPrinter = &pretty.Config{
	Compact:        true,
	SkipZeroFields: true,
	Formatter:      pretty.DefaultFormatter,
}

// Bits is a set of 8 bit flags.
type Bits = utils.Bits

// Counter is a monotonically increasing unsigned integer.
type Counter uint64

// Next increments the counter and returns the new value
func (c *Counter) Next() uint64 {
	return atomic.AddUint64((*uint64)(c), 1)
}

// mergeYgotAtPathStr merges a ygot.ValidatedGoStruct at a given path in ygot tree ygRoot.
// Returns the target node from the ygot tree.
func mergeYgotAtPathStr(ygRoot *ocbinds.Device, pathStr string, value ygot.ValidatedGoStruct) (ygot.ValidatedGoStruct, error) {
	p, err := ygot.StringToStructuredPath(pathStr)
	if err != nil {
		return nil, err
	}

	return mergeYgotAtPath(ygRoot, p, value)
}

// mergeYgotAtPathStr merges a ygot.ValidatedGoStruct at a given path in ygot tree ygRoot.
// Returns the target node from the ygot tree.
// This function may update the input path!
func mergeYgotAtPath(ygRoot *ocbinds.Device, p *gnmi.Path, value ygot.ValidatedGoStruct) (ygot.ValidatedGoStruct, error) {
	var pNode ygot.ValidatedGoStruct
	path.RemoveModulePrefix(p)

	ygNode, _, err := ytypes.GetOrCreateNode(ygSchema.RootSchema(), ygRoot, p)
	if err != nil {
		return nil, err
	}
	if n, ok := ygNode.(ygot.ValidatedGoStruct); ok {
		pNode = n
	} else {
		return nil, fmt.Errorf("GetOrCreateNode returned %T; is not a ValidatedGoStruct", ygNode)
	}
	if value != nil {
		err = ygot.MergeStructInto(pNode, value)
	}
	return pNode, err
}

// clearListKeys sets the list key fields of a ygot struct to their zero values.
// Noop if the ygot obj is not a yang list (i.e, not a ygot.KeyHelperGoStruct).
func clearListKeys(y ygot.ValidatedGoStruct) error {
	yList, ok := y.(ygot.KeyHelperGoStruct)
	if !ok { // not a list node
		return nil
	}

	keyMap, err := yList.ΛListKeyMap()
	if err != nil {
		return err
	}

	yv := reflect.ValueOf(y).Elem()
	yt := yv.Type()
	for i := 0; i < yv.NumField(); i++ {
		ft := yt.Field(i)
		pathTag, _ := ft.Tag.Lookup("path")
		if _, ok := keyMap[pathTag]; ok {
			yv.Field(i).Set(reflect.Zero(ft.Type))
		}
	}

	return nil
}

// getYgotAtPath looks up the child ygot struct at a childPath in a parent struct.
// Absolute path to the parent struct from root should also be passed.
// Returns error if childPath does not point to a struct.
// Warning: this function may update the childPath by removing module prefixes.
func getYgotAtPath(parent ygot.ValidatedGoStruct, childPath *gnmi.Path) (ygot.ValidatedGoStruct, error) {
	structName := reflect.TypeOf(parent).Elem().Name()
	schema, ok := ocbinds.SchemaTree[structName]
	if !ok {
		return nil, fmt.Errorf("Could not find schema for %T", parent)
	}

	path.RemoveModulePrefix(childPath)
	nodes, err := ytypes.GetNode(schema, parent, childPath)
	if err != nil {
		return nil, err
	}

	if yv, ok := nodes[0].Data.(ygot.ValidatedGoStruct); ok {
		return yv, nil
	}

	return nil, fmt.Errorf("Not a ValidatedGoStruct")
}

// isEmptyYgotStruct returns true if none of the fields of ygot struct y are set.
func isEmptyYgotStruct(y ygot.ValidatedGoStruct) bool {
	yv := reflect.ValueOf(y).Elem()
	return isEmptyYgotStructVal(yv)
}

// isEmptyYgotStructVal returns true if the reflect.Value yv represents
// a nil or empty ygot struct. Can panic if it is not a ygot struct value.
func isEmptyYgotStructVal(yv reflect.Value) bool {
	for i := yv.NumField() - 1; i >= 0; i-- {
		fv := yv.Field(i)
		// ygot struct's fields will always be one of ptr, map, slice or an int (for enum & identity).
		// Value.IsZero() should handle all cases.
		if fv.IsValid() && !fv.IsZero() {
			ft := fv.Type()
			if ft.Kind() != reflect.Ptr || ft.Elem().Kind() != reflect.Struct {
				return false
			}
			if !isEmptyYgotStructVal(fv.Elem()) {
				return false
			}
		}
	}
	return true
}
