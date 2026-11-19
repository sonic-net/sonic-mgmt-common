////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Nokia.                                                     //
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
	"testing"
)

func TestSystemAppRouting(t *testing.T) {
	sysApp := reflect.TypeOf(SysApp{})
	commonApp := reflect.TypeOf(CommonApp{})

	tests := []struct {
		path string
		want reflect.Type
	}{
		{"/openconfig-system:system/processes", sysApp},
		{"/openconfig-system:system/processes/process", sysApp},
		{"/openconfig-system:system/processes/process[pid=1]/state/name", sysApp},
		{"/openconfig-system:system", commonApp},
		{"/openconfig-system:system/state", commonApp},
		{"/openconfig-system:system/ssh-server/state", commonApp},
		{"/openconfig-system:system/openconfig-system-grpc:grpc-servers", commonApp},
	}

	for _, tt := range tests {
		info, err := getAppModuleInfo(tt.path)
		if err != nil {
			t.Errorf("getAppModuleInfo(%q) failed: %v", tt.path, err)
			continue
		}
		if info.appType != tt.want {
			t.Errorf("getAppModuleInfo(%q) = %v, want %v", tt.path, info.appType, tt.want)
		}
	}
}
