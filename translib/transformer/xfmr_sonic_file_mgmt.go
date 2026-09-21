////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2025 Cisco.                                                 //
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

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

func init() {
	XlateFuncBind("rpc_file_remove_cb", rpc_file_remove_cb)
}

var rpc_file_remove_cb RpcCallpoint = func(body []byte, _ [db.MaxDB]*db.DB) ([]byte, error) {
	var err error
	var output struct{}
	var operand struct {
		RemoteFile string `json:"remote_file"`
	}
	glog.Info("File Remove request - RPC callback")

	err = json.Unmarshal(body, &operand)
	if err != nil {
		glog.Errorf("Error: Failed to parse rpc input; err=%v", err)
		return nil, tlerr.InvalidArgs("Invalid rpc input")
	}

	result, _ := json.Marshal(&output)

	host_output := HostQuery("file.remove", operand.RemoteFile)
	if host_output.Err != nil {
		glog.Errorf("Error: File Remove host Query failed: err=%v", host_output.Err)
		return nil, host_output.Err
	}
	if host_output.Body[0].(int32) != 0 {
		err = tlerr.New(host_output.Body[1].(string))
		glog.Errorf("Error: File Remove host Query failed: err=%v", err)
		return nil, err
	}

	return result, nil
}
