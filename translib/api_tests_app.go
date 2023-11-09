////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

//go:build test
// +build test

package translib

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

// apiTests is an app module for testing translib APIs.
// Implements dummy handlers for paths starting with "/api-tests:".
// Returns error if path contains "/error/"; see getError function.
type apiTests struct {
	path string
	body []byte
	appOptions

	echoMsg string
	echoErr string
}

func init() {
	err := register("/api-tests:",
		&appInfo{
			appType:       reflect.TypeOf(apiTests{}),
			isNative:      true,
			tablesToWatch: nil})

	if err != nil {
		glog.Fatalf("Failed to register ApiTest app; %v", err)
	}
}

func (app *apiTests) initialize(inp appData) {
	app.path = inp.path
	app.body = inp.payload
	app.appOptions = inp.appOptions
}

func (app *apiTests) translateCreate(d *db.DB) ([]db.WatchKeys, error) {
	return nil, app.translatePath()
}

func (app *apiTests) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	return nil, app.translatePath()
}

func (app *apiTests) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	return nil, app.translatePath()
}

func (app *apiTests) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	return nil, app.translatePath()
}

func (app *apiTests) translateGet(dbs [db.MaxDB]*db.DB) error {
	return app.translatePath()
}

func (app *apiTests) translateAction(dbs [db.MaxDB]*db.DB) error {
	var req struct {
		Input struct {
			Message string `json:"message"`
			ErrType string `json:"error-type"`
		} `json:"api-tests:input"`
	}

	err := json.Unmarshal(app.body, &req)
	if err != nil {
		glog.Errorf("Failed to parse rpc input; err=%v", err)
		return tlerr.InvalidArgs("Invalid rpc input")
	}

	app.echoMsg = req.Input.Message
	app.echoErr = req.Input.ErrType

	return nil
}

func (app *apiTests) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	return emptySubscribeResponse(req.path)
}

func (app *apiTests) processSubscribe(req processSubRequest) (processSubResponse, error) {
	return processSubResponse{}, tlerr.New("not implemented")
}

func (app *apiTests) processCreate(d *db.DB) (SetResponse, error) {
	return app.processSet()
}

func (app *apiTests) processUpdate(d *db.DB) (SetResponse, error) {
	return app.processSet()
}

func (app *apiTests) processReplace(d *db.DB) (SetResponse, error) {
	return app.processSet()
}

func (app *apiTests) processDelete(d *db.DB) (SetResponse, error) {
	return app.processSet()
}

func (app *apiTests) processGet(dbs [db.MaxDB]*db.DB, fmtType TranslibFmtType) (GetResponse, error) {
	var gr GetResponse
	err := app.getError()
	if err != nil {
		return gr, err
	}

	resp := make(map[string]interface{})
	resp["message"] = app.echoMsg
	resp["path"] = app.path
	resp["depth"] = app.depth
	resp["content"] = app.content
	resp["fields"] = app.fields

	gr.Payload, err = json.Marshal(&resp)
	return gr, err
}

func (app *apiTests) processAction(dbs [db.MaxDB]*db.DB) (ActionResponse, error) {
	var ar ActionResponse

	err := app.getError()
	if err == nil {
		var respData struct {
			Output struct {
				Message string `json:"message"`
			} `json:"api-tests:output"`
		}

		respData.Output.Message = app.echoMsg
		ar.Payload, err = json.Marshal(&respData)
	}

	return ar, err
}

func (app *apiTests) translatePath() error {
	app.echoMsg = "Hello, world!"
	k := strings.Index(app.path, "error/")
	if k >= 0 {
		app.echoErr = app.path[k+6:]
	}
	return nil
}

func (app *apiTests) processSet() (SetResponse, error) {
	var sr SetResponse
	err := app.getError()
	return sr, err
}

func (app *apiTests) getError() error {
	switch strings.ToLower(app.echoErr) {
	case "invalid-args", "invalidargs":
		return tlerr.InvalidArgs(app.echoMsg)
	case "exists":
		return tlerr.AlreadyExists(app.echoMsg)
	case "not-found", "notfound":
		return tlerr.NotFound(app.echoMsg)
	case "not-supported", "notsupported", "unsupported":
		return tlerr.NotSupported(app.echoMsg)
	case "", "no", "none", "false":
		return nil
	default:
		return tlerr.New(app.echoMsg)
	}
}
