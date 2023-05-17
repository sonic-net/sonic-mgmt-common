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
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/golang/glog"
)

/* This file includes common utilities for app module test code. */

// CleanupFunc is the callback function used for cleaning config_db
// entries before and after tests. Should be registered via addCleanupFunc.
type CleanupFunc func() error

var cleanupFuncs map[string]CleanupFunc

// addCleanupFunc registers a cleanup function. Should be called from
// init() of individual test files if they need pre/post cleanup.
func addCleanupFunc(name string, f CleanupFunc) {
	if cleanupFuncs == nil {
		cleanupFuncs = map[string]CleanupFunc{}
	}

	cleanupFuncs[name] = f
}

func TestMain(m *testing.M) {
	// cleanup before tests
	for name, f := range cleanupFuncs {
		if err := f(); err != nil {
			glog.Errorf("%s cleanup failed; err=%v", name, err)
			os.Exit(-1)
		} else {
			glog.Infof("Cleanup %s before tests", name)
		}
	}

	ret := m.Run()

	// cleanup after tests
	for name, f := range cleanupFuncs {
		if err := f(); err != nil {
			glog.Warningf("Cleanup %s failed; err=%v", name, err)
		} else {
			glog.Infof("Cleanup %s after tests", name)
		}
	}

	os.Exit(ret)
}

func processGetRequest(url string, expectedRespJson string, errorCase bool) func(*testing.T) {
	return func(t *testing.T) {
		response, err := Get(GetRequest{Path: url})
		switch {
		case err != nil && !errorCase:
			t.Fatalf("Error %v received for Url: %s", err, url)
		case err == nil && errorCase:
			t.Fatalf("GET %s did not return an error", url)
		case errorCase:
			return
		}

		respJson := response.Payload

		var jResponse, jExpected map[string]interface{}
		if err := json.Unmarshal(respJson, &jResponse); err != nil {
			t.Fatalf("invalid response json; err = %v\npayload = %s", err, respJson)
		}
		if err := json.Unmarshal([]byte(expectedRespJson), &jExpected); err != nil {
			t.Fatalf("invalid expected json; err = %v", err)
		}
		if !reflect.DeepEqual(jResponse, jExpected) {
			t.Errorf("GET %s returned invalid response", url)
			t.Errorf("Expected: %s", expectedRespJson)
			t.Fatalf("Received: %s", respJson)
		}
	}
}

func processSetRequest(url string, jsonPayload string, oper string, errorCase bool) func(*testing.T) {
	return func(t *testing.T) {
		var err error
		switch oper {
		case "POST":
			_, err = Create(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PATCH":
			_, err = Update(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PUT":
			_, err = Replace(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "DELETE":
			_, err = Delete(SetRequest{Path: url})
		default:
			t.Errorf("Operation not supported")
		}
		if err != nil && !errorCase {
			t.Errorf("Error %v received for Url: %s", err, url)
		}
	}
}

func processDeleteRequest(url string) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Delete(SetRequest{Path: url})
		if err != nil {
			t.Errorf("Error %v received for Url: %s", err, url)
		}
	}
}

func getConfigDb() *db.DB {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})

	return configDb
}
