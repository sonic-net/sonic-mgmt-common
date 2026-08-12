////////////////////////////////////////////////////////////////////////////////
//
// Copyright 2026 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
////////////////////////////////////////////////////////////////////////////////

//go:build testapp
// +build testapp

package transformer_test

import (
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

func Test_oc_vrrp6_group_operations(t *testing.T) {
	t.Log("\n\n++++++++++ Test IPv6 VRRP group operations ++++++++++++")

	preReq := map[string]interface{}{
		"PORT": map[string]interface{}{
			"Ethernet64": map[string]interface{}{"admin_status": "up"},
			"Ethernet72": map[string]interface{}{"admin_status": "up"},
		},
		"INTERFACE": map[string]interface{}{
			"Ethernet64|2001:db8::1/64": map[string]interface{}{"NULL": "NULL"},
		},
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": map[string]interface{}{
				"vid":      "1",
				"vip@":     "2001:db8::fe/128",
				"priority": "100",
			},
		},
	}
	loadDB(db.ConfigDB, preReq)

	base := "/openconfig-interfaces:interfaces/interface[name=Ethernet64]/subinterfaces/subinterface[index=0]/ipv6/addresses/address[ip=2001:db8::1]/vrrp/vrrp-group[virtual-router-id=1]"

	t.Run("PUT VRRP6 priority", processSetRequest(
		base+"/config/priority",
		`{"openconfig-if-ip:priority": 120}`,
		"PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify VRRP6 priority", verifyDbResult(rclient, "VRRP6|Ethernet64|1", map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": map[string]interface{}{
				"priority": "120",
			},
		},
	}, false))

	t.Run("PUT VRRP6 preempt disabled", processSetRequest(
		base+"/config/preempt",
		`{"openconfig-if-ip:preempt": false}`,
		"PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify VRRP6 preempt", verifyDbResult(rclient, "VRRP6|Ethernet64|1", map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": map[string]interface{}{
				"preempt": "disabled",
			},
		},
	}, false))

	t.Run("PUT VRRP6 virtual-address", processSetRequest(
		base+"/config/virtual-address",
		`{"openconfig-if-ip:virtual-address": ["2001:db8::10/128", "2001:db8::11/128"]}`,
		"PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify VRRP6 virtual-address", verifyDbResult(rclient, "VRRP6|Ethernet64|1", map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": map[string]interface{}{
				"vip@": "2001:db8::10/128,2001:db8::11/128,2001:db8::fe/128",
			},
		},
	}, false))

	t.Run("PUT VRRP6 virtual-link-local", processSetRequest(
		base+"/config/virtual-link-local",
		`{"openconfig-if-ip:virtual-link-local": "fe80::1/64"}`,
		"PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify VRRP6 virtual-link-local", verifyDbResult(rclient, "VRRP6|Ethernet64|1", map[string]interface{}{
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": map[string]interface{}{
				"vip@": "2001:db8::10/128,2001:db8::11/128,2001:db8::fe/128,fe80::1/64",
			},
		},
	}, false))

	t.Run("PUT VRRP6 interface-tracking", processSetRequest(
		base+"/interface-tracking/config",
		`{
  "openconfig-if-ip:config": {
    "track-interface": ["Ethernet72"],
    "priority-decrement": 20
  }
}`,
		"PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify VRRP6_TRACK entry", verifyDbResult(rclient, "VRRP6_TRACK|Ethernet64|1|Ethernet72", map[string]interface{}{
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet64|1|Ethernet72": map[string]interface{}{
				"priority_increment": "20",
			},
		},
	}, false))

	cleanup := map[string]interface{}{
		"VRRP6_TRACK": map[string]interface{}{
			"Ethernet64|1|Ethernet72": "",
		},
		"VRRP6": map[string]interface{}{
			"Ethernet64|1": "",
		},
		"INTERFACE": map[string]interface{}{
			"Ethernet64|2001:db8::1/64": "",
		},
		"PORT": map[string]interface{}{
			"Ethernet64": "",
			"Ethernet72": "",
		},
	}
	unloadDB(db.ConfigDB, cleanup)
}
