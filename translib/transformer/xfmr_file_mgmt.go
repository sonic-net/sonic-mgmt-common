////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Dell, Inc.                                                 //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
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
	"encoding/json"
	"errors"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/golang/glog"
)

var copyMutex = &sync.Mutex{}

func init() {
	XlateFuncBind("rpc_dir_cb", rpc_dir_cb)
	XlateFuncBind("rpc_copy_cb", rpc_copy_cb)
	XlateFuncBind("rpc_delete_cb", rpc_delete_cb)
	XlateFuncBind("YangToDb_config_reload_key_xfmr", YangToDb_config_reload_key_xfmr)
}

var rpc_copy_cb RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	copyMutex.Lock()
	defer copyMutex.Unlock()
	return copy_action(body, dbs)
}

func copy_action(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	var err error
	var result []byte
	var options []string
	var query_result HostResult
	var source, destination, option string

	var operand struct {
		Input struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
			Copy_option string `json:"copy-config-option"`
		} `json:"openconfig-file-mgmt-private:input"`
	}

	var sum struct {
		Output struct {
			Status        int32  `json:"status"`
			Status_detail string `json:"status-detail"`
		} `json:"openconfig-file-mgmt-private:output"`
	}

	err = json.Unmarshal(body, &operand)

	if err != nil {
		/* Unmarshall failed, no input provided.
		 * set to default */
		glog.Error("Copy input not provided.")
		err = errors.New("Input parameters missing.")
	} else {
		source = operand.Input.Source
		destination = operand.Input.Destination
		option = operand.Input.Copy_option

		if source != "running-configuration" || destination != "startup-configuration" || option == "REPLACE" {
			return nil, errors.New("rpc_copy_cb: Only supports running-configuration -> startup-configuration")
		}

		options = append(options, source)
		options = append(options, destination)
		glog.Infof("Invoke cfg_mgmt.save ")
		query_result = HostQuery("cfg_mgmt.save", options)
	}

	sum.Output.Status = 1
	if err != nil {
		sum.Output.Status_detail = err.Error()
		glog.Infof("Error: File management host Query error : err=%v", err.Error())
	} else if query_result.Body[0].(int32) != 0 {
		glog.Infof("Error: File management host Query error")
		sum.Output.Status_detail = query_result.Body[1].(string)
	} else if query_result.Err != nil {
		glog.Infof("Error: File management host Query failed for copy: err=%v", query_result.Err)
		sum.Output.Status_detail = query_result.Err.Error()
	} else {
		sum.Output.Status = 0
		sum.Output.Status_detail = "SUCCESS."
	}

	result, err = json.Marshal(&sum)

	return result, err
}

var rpc_dir_cb RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	return nil, errors.New("rpc_dir_cb: Not Implemented")
}

var rpc_delete_cb RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	return nil, errors.New("rpc_delete_cb: Not Implemented")
}

var YangToDb_config_reload_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	return "state", nil
}
