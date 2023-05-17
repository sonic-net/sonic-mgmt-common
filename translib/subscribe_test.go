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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/openconfig/ygot/ygot"
)

// Messages is a utility to collect list of
// formatted messages and log it later.
type Messages struct {
	list []string
}

func (m *Messages) Add(format string, args ...interface{}) {
	m.list = append(m.list, fmt.Sprintf(format, args...))
}

func (m *Messages) Empty() bool {
	return len(m.list) == 0
}

func (m *Messages) LogTo(t *testing.T) {
	for _, s := range m.list {
		t.Log(s)
	}
}

///////////////////
// Utilities to test translateSubscribe

var translErr = -1

type translateSubscribeVerifier struct {
	t           *testing.T
	path        string
	mode        NotificationType
	targetInfos []*notificationAppInfo
	childInfos  []*notificationAppInfo
	appError    error
}

func testTranslateSubscribe(t *testing.T, path string) *translateSubscribeVerifier {
	return testTranslateSubscribeForMode(t, path, OnChange)
}

func testTranslateSubscribeForMode(t *testing.T, path string, mode NotificationType) *translateSubscribeVerifier {
	tv := &translateSubscribeVerifier{
		t:    t,
		path: path,
		mode: mode,
	}
	app, _, err := getAppModule(path, Version{})
	if err != nil {
		tv.appError = err
		return tv
	}

	resp, err := (*app).translateSubscribe(
		translateSubRequest{
			ctxID:   t.Name(),
			path:    path,
			mode:    mode,
			recurse: true,
			dbs:     [db.MaxDB]*db.DB{},
		})

	if err != nil {
		tv.appError = err
	} else {
		tv.targetInfos = resp.ntfAppInfoTrgt
		tv.childInfos = resp.ntfAppInfoTrgtChlds
	}

	return tv
}

// VerifyCount validates if translateSubscribe returned expected number of
// target and child notificationAppInfo objects. Pass any of the count as translErr
// to validate for error response.
func (tv *translateSubscribeVerifier) VerifyCount(targetCount, childCount int) {
	if targetCount == translErr || childCount == translErr {
		if tv.appError == nil {
			tv.t.Fatalf("tanslateSubscribe(%s, %v) should have failed", tv.path, tv.mode)
		}
		return
	}
	if tv.appError != nil {
		tv.t.Fatalf("tanslateSubscribe(%s, %v) failed; err=%v", tv.path, tv.mode, tv.appError)
	}
	if len(tv.targetInfos) != targetCount || len(tv.childInfos) != childCount {
		tv.t.Fatalf("translateSubscribe(%s, %v) failed; Expected %v target and %d child infos. Found %d and %d",
			tv.path, tv.mode, targetCount, childCount, len(tv.targetInfos), len(tv.childInfos))
	}
}

// VerifyTarget checks if target notificationAppInfo list has a matching entry
func (tv *translateSubscribeVerifier) VerifyTarget(path string, expInfo *notificationAppInfo) {
	tv.t.Helper()
	tv.findAndCompare("targetInfo", path, expInfo)
}

// VerifyChild checks if child notificationAppInfo list has a matching entry
func (tv *translateSubscribeVerifier) VerifyChild(path string, nAppInfo *notificationAppInfo) {
	tv.t.Helper()
	tv.findAndCompare("childInfo", path, nAppInfo)
}

func (tv *translateSubscribeVerifier) findAndCompare(kind, path string, expInfo *notificationAppInfo) {
	tv.t.Helper()
	var paths []string
	list := tv.targetInfos
	if kind == "childInfo" {
		list = tv.childInfos
	}
	for _, nInfo := range list {
		if p, err := ygot.PathToString(nInfo.path); err != nil {
			tv.t.Errorf("translateSubscribe(%s, %v) returned invalid %s path: %v; err=%v",
				tv.path, tv.mode, kind, nInfo.path, err)
		} else if p == path {
			tv.compare(nInfo, expInfo)
			return
		} else {
			paths = append(paths, p)
		}
	}
	// Path not found
	tv.t.Logf("Did not find %s for: %v", kind, path)
	tv.t.Logf("Available %s paths : %v", kind, paths)
	tv.t.FailNow()
}

func (tv *translateSubscribeVerifier) compare(nInfo, expInfo *notificationAppInfo) {
	tv.t.Helper()
	var errors Messages
	if nInfo.dbno != expInfo.dbno {
		errors.Add("dbno mismatch; expected=%v, found=%v", expInfo.dbno, nInfo.dbno)
	}
	if tableInfo(nInfo.table) != tableInfo(expInfo.table) {
		errors.Add("table mismatch; expected=%v, found=%v", tableInfo(expInfo.table), tableInfo(nInfo.table))
	}
	if expInfo.key != nil && (nInfo.key == nil || !expInfo.key.Equals(*nInfo.key)) {
		errors.Add("key mismatch; expected=%v, found=%v", keyInfo(expInfo.key), keyInfo(nInfo.key))
	}
	if !listEquals(expInfo.keyGroupComps, nInfo.keyGroupComps) {
		errors.Add("keyGroup mismatch; expected=%v, found=%v", expInfo.keyGroupComps, nInfo.keyGroupComps)
	}
	if expInfo.fieldScanPattern != nInfo.fieldScanPattern {
		errors.Add("fieldScanPattern mismatch; expected=%q, found=%q", expInfo.fieldScanPattern, nInfo.fieldScanPattern)
	}
	if expInfo.handlerFunc.String() != nInfo.handlerFunc.String() {
		errors.Add("handlerFunc mismatch; expected=%v, found=%v", expInfo.handlerFunc, nInfo.handlerFunc)
	}
	dbFields := toFieldsJSON(nInfo)
	expFields := toFieldsJSON(expInfo)
	if expInfo.dbFldYgPathInfoList != nil && !reflect.DeepEqual(dbFields, expFields) {
		val, _ := json.Marshal(dbFields)
		exp, _ := json.Marshal(expFields)
		errors.Add("dbFldYgPathInfoList mismatch;")
		errors.Add("  expected=%v", string(exp))
		errors.Add("  found=%v", string(val))
	}
	if nInfo.deleteAction != expInfo.deleteAction {
		errors.Add("deleteAction mismatch; expected=%v, found=%v", expInfo.deleteAction, nInfo.deleteAction)
	}
	if tv.mode != Sample && nInfo.isOnChangeSupported != expInfo.isOnChangeSupported {
		errors.Add("isOnChangeSupported mismatch; expected=%v, found=%v", expInfo.isOnChangeSupported, nInfo.isOnChangeSupported)
	}
	if tv.mode != OnChange && nInfo.mInterval != expInfo.mInterval {
		errors.Add("minInterval mismatch; expected=%v, found=%v", expInfo.mInterval, nInfo.mInterval)
	}
	if tv.mode == TargetDefined && nInfo.pType != expInfo.pType {
		errors.Add("preferredMode mismatch; expected=%v, found=%v", expInfo.pType, nInfo.pType)
	}
	if !errors.Empty() {
		p, _ := ygot.PathToString(nInfo.path)
		tv.t.Errorf("translateSubscribe(%s, %v) failed", tv.path, tv.mode)
		tv.t.Errorf("notificationAppInfo for '%s' does not match expected values", p)
		errors.LogTo(tv.t)
	}
}

func listEquals(x, y interface{}) bool {
	if reflect.ValueOf(x).Len() == 0 { // treat nil and empty list as equal
		return reflect.ValueOf(y).Len() == 0
	}
	return reflect.DeepEqual(x, y)
}

// subscribeFieldsJSON represents dbFldYgPathInfo objects in JSON format.
// Syntax: {"prefix1": {"db_field1": "yang_field1", ...}, "prefix2": {...}}
type subscribeFieldsJSON map[string]map[string]string

// toFieldsJSON returns ni.dbFldYgPathInfoList as a subscribeFieldsJSON object.
func toFieldsJSON(ni *notificationAppInfo) subscribeFieldsJSON {
	jsonData := make(subscribeFieldsJSON)
	for _, entry := range ni.dbFldYgPathInfoList {
		if _, ok := jsonData[entry.rltvPath]; !ok {
			jsonData[entry.rltvPath] = make(map[string]string)
		}
		for k, v := range entry.dbFldYgPathMap {
			jsonData[entry.rltvPath][k] = v
		}
	}
	return jsonData
}

// parseFieldsJSON parses a JSON string in subscribeFieldsJSON syntax into
// an array of dbFldYgPathInfo objects.
func parseFieldsJSON(mappingJSON string) []*dbFldYgPathInfo {
	if len(mappingJSON) == 0 {
		return nil
	}
	jsonData := make(subscribeFieldsJSON)
	err := json.Unmarshal([]byte(mappingJSON), &jsonData)
	if err != nil {
		panic(fmt.Sprintf("json.Unmarshal failed; err=%v; json=%v", err, mappingJSON))
	}
	var mappings []*dbFldYgPathInfo
	for prefix, fields := range jsonData {
		mappings = append(mappings, &dbFldYgPathInfo{rltvPath: prefix, dbFldYgPathMap: fields})
	}
	return mappings
}
