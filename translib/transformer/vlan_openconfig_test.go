////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2025 Dell, Inc.                                                 //
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

//go:build testapp
// +build testapp

package transformer_test

import (
	"testing"
	"time"
)

func Test_openconfig_vlan(t *testing.T) {
	var url, url_input_body_json string

	t.Log("\n\n+++++++++++++ CONFIGURING VLAN ++++++++++++")

	t.Log("\n\n--- PATCH to Create VLAN 10 ---")
	url = "/openconfig-interfaces:interfaces"
	url_input_body_json = "{\"openconfig-interfaces:interfaces\":{\"interface\":[{\"name\":\"Vlan10\",\"config\":{\"name\":\"Vlan10\",\"mtu\":9000,\"enabled\":true}}]}}"
	t.Run("Test Create VLAN 10", processSetRequest(url, url_input_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
}
