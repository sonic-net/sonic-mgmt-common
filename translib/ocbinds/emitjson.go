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

package ocbinds

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

// EmitJSON serializes a GoStruct s into RFC7951 json.
func EmitJSON(s ygot.ValidatedGoStruct, opts *EmitJSONOptions) ([]byte, error) {
	return getEmitJSONImpl()(s, opts)
}

func newEmitJSON(s ygot.ValidatedGoStruct, opts *EmitJSONOptions) ([]byte, error) {
	glog.V(3).Infof("Render %T using internal EmitJSON", s)
	if s == nil {
		return nil, nil
	}

	var y2j ygot2json
	y2j.buff = new(bytes.Buffer)
	if opts != nil {
		y2j.EmitJSONOptions = *opts
	}

	y2j.renderStruct(reflect.ValueOf(s))

	// renderStruct() does not generate anything for empty struct
	if y2j.buff.Len() == 0 {
		y2j.buff.WriteString("{}")
	}

	return y2j.buff.Bytes(), nil
}

type EmitJSONOptions struct {
	NoPrefix bool // Render node & enum names without module prefix
	SortList bool // Sort lists by keys
}

type ygot2json struct {
	EmitJSONOptions
	buff      *bytes.Buffer
	parentMod string        // Module of parent node
	encoder   *json.Encoder // JSON encoder for rendering strings}
}

func (y2j *ygot2json) renderStruct(s reflect.Value) {
	sv := s.Elem()
	st := sv.Type()
	empty := true
	origParentMod := y2j.parentMod

	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		if y2j.isNil(&fv) {
			continue
		}

		if empty {
			y2j.buff.WriteByte('{')
			empty = false
		} else {
			y2j.buff.WriteByte(',')
		}

		ft := st.Field(i)
		mod, name := y2j.getNodeName(&ft)
		if len(mod) != 0 && mod != y2j.parentMod {
			fmt.Fprintf(y2j.buff, "\"%s:%s\":", mod, name)
			y2j.parentMod = mod
		} else {
			fmt.Fprintf(y2j.buff, "\"%s\":", name)
		}

		switch ft.Type.Kind() {
		case reflect.Ptr:
			if ft.Type.Elem().Kind() == reflect.Struct {
				y2j.renderStruct(fv)
			} else {
				y2j.renderLeaf(&fv)
			}
		case reflect.Map: // list
			y2j.renderList(&fv)
		case reflect.Slice:
			if ft.Type.Elem().Kind() == reflect.Ptr { // keyless list
				y2j.renderKeylessList(&fv)
			} else if ft.Type.Name() == ygot.BinaryTypeName { // binary
				y2j.renderBinary(fv.Bytes())
			} else { // leaf-list
				y2j.renderKeylessList(&fv)
			}
		case reflect.Int64: // enum
			y2j.renderEnum(&fv)
		case reflect.Interface: // union
			unboxed := fv.Elem().Elem().Field(0)
			y2j.renderLeaf(&unboxed)
		case reflect.Bool: // empty leaf
			y2j.buff.WriteString("[null]")
		default:
			panic(fmt.Sprintf("Unknown type %s at %s.%s",
				ft.Type.Name(), st.Name(), ft.Name))
		}

		y2j.parentMod = origParentMod
	}

	if !empty {
		y2j.buff.WriteByte('}')
	}
}

func (y2j *ygot2json) isNil(v *reflect.Value) bool {
	if !v.IsValid() || v.IsZero() {
		return true
	}
	vt := v.Type()
	if vt.Kind() == reflect.Map {
		return v.Len() == 0
	}
	if vt.Kind() != reflect.Ptr || vt.Elem().Kind() != reflect.Struct {
		return false
	}
	sv := v.Elem()
	for i := sv.NumField() - 1; i >= 0; i-- {
		mfv := sv.Field(i)
		if !y2j.isNil(&mfv) {
			return false
		}
	}
	return true
}

func (y2j *ygot2json) getNodeName(f *reflect.StructField) (mod string, elem string) {
	elem = f.Tag.Get("path")
	if len(elem) == 0 {
		panic(f.Name + " has no 'path' tag")
	}
	if !y2j.NoPrefix {
		mod = f.Tag.Get("module")
	}
	return
}

func (y2j *ygot2json) renderLeaf(v *reflect.Value) {
	switch v.Type().Kind() {
	case reflect.Ptr:
		elem := v.Elem()
		y2j.renderLeaf(&elem)
	case reflect.Int64:
		if _, ok := v.Interface().(ygot.GoEnum); ok {
			y2j.renderEnum(v)
		} else {
			fmt.Fprintf(y2j.buff, "\"%d\"", v.Int())
		}
	case reflect.Uint64:
		fmt.Fprintf(y2j.buff, "\"%d\"", v.Uint())
	case reflect.Float64:
		v := strconv.FormatFloat(v.Float(), 'f', -1, 64)
		fmt.Fprintf(y2j.buff, "\"%s\"", v)
	case reflect.String:
		y2j.encodeValue(v.String())
	case reflect.Interface: // union
		unboxed := v.Elem().Elem().Field(0)
		y2j.renderLeaf(&unboxed)
	default:
		//y2j.encodeValue(fv.Interface())
		fmt.Fprintf(y2j.buff, "%v", v.Interface())
	}
}

func (y2j *ygot2json) renderList(v *reflect.Value) {
	y2j.buff.WriteByte('[')
	if y2j.SortList && v.Len() > 1 {
		y2j.renderListSorted(v)
	} else {
		y2j.renderListUnsorted(v)
	}
	y2j.buff.WriteByte(']')
}

func (y2j *ygot2json) renderListSorted(v *reflect.Value) {
	sm := SortedMap(v)
	for i, n := 0, sm.Len(); i < n; i++ {
		if i != 0 {
			y2j.buff.WriteByte(',')
		}
		y2j.renderStruct(sm.At(i))
	}
}

func (y2j *ygot2json) renderListUnsorted(v *reflect.Value) {
	var comma bool
	for iter := v.MapRange(); iter.Next(); {
		if comma {
			y2j.buff.WriteByte(',')
		} else {
			comma = true
		}
		y2j.renderStruct(iter.Value())
	}
}

func (y2j *ygot2json) renderKeylessList(v *reflect.Value) {
	size := v.Len()
	y2j.buff.WriteByte('[')
	for i := 0; i < size; i++ {
		if i != 0 {
			y2j.buff.WriteByte(',')
		}
		iv := v.Index(i)
		if iv.Type().Kind() == reflect.Ptr {
			y2j.renderStruct(iv)
		} else {
			y2j.renderLeaf(&iv)
		}
	}
	y2j.buff.WriteByte(']')
}

func (y2j *ygot2json) encodeValue(v interface{}) {
	if y2j.encoder == nil {
		y2j.encoder = json.NewEncoder(y2j.buff)
		y2j.encoder.SetEscapeHTML(false)
	}
	y2j.encoder.Encode(v)
	y2j.buff.Truncate(y2j.buff.Len() - 1) // remove newline added by encoder
}

func (y2j *ygot2json) renderBinary(v []byte) {
	y2j.buff.WriteByte('"')
	b64 := base64.NewEncoder(base64.StdEncoding, y2j.buff)
	b64.Write(v)
	b64.Close()
	y2j.buff.WriteByte('"')
}

func (y2j *ygot2json) renderEnum(v *reflect.Value) {
	enum, ok := v.Interface().(ygot.GoEnum)
	if !ok {
		panic(fmt.Sprintf("%s is not a GoEnum", v.Type().Name()))
	}
	ev, ok := enum.ΛMap()[v.Type().Name()][v.Int()]
	if !ok {
		panic(fmt.Sprintf("Invalid value for %s: %v", v.Type().Name(), enum))
	}
	if y2j.NoPrefix || len(ev.DefiningModule) == 0 {
		fmt.Fprintf(y2j.buff, "\"%s\"", ev.Name)
	} else {
		fmt.Fprintf(y2j.buff, "\"%s:%s\"", ev.DefiningModule, ev.Name)
	}
}
