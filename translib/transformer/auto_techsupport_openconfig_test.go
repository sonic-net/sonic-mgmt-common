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

//go:build testapp
// +build testapp

package transformer_test

import (
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

const coreFileCleanupUrl = "/openconfig-system:system/openconfig-system-ext:core-file-cleanup"

func autoTsGlobalRow(fields map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"AUTO_TECHSUPPORT": map[string]interface{}{"GLOBAL": fields}}
}

var autoTsGlobalCleanup = map[string]interface{}{"AUTO_TECHSUPPORT": map[string]interface{}{"GLOBAL": ""}}

func Test_openconfig_system_core_file_cleanup(t *testing.T) {
	defaults := map[string]interface{}{
		"state":                 "enabled",
		"rate_limit_interval":   "180",
		"max_techsupport_limit": "10.0",
		"max_core_limit":        "5.0",
	}
	withCoreLimit := func(v string) map[string]interface{} {
		m := map[string]interface{}{}
		for f, fv := range defaults {
			m[f] = fv
		}
		m["max_core_limit"] = v
		return autoTsGlobalRow(m)
	}

	loadDB(db.ConfigDB, autoTsGlobalRow(defaults))
	t.Run("Get core-file-cleanup", processGetRequest(coreFileCleanupUrl, nil,
		`{"openconfig-system-ext:core-file-cleanup":{"config":{"max-core-limit":"5"},"state":{"max-core-limit":"5","enabled":true}}}`, false))
	t.Run("Get core-file-cleanup state", processGetRequest(coreFileCleanupUrl+"/state", nil,
		`{"openconfig-system-ext:state":{"max-core-limit":"5","enabled":true}}`, false))

	t.Run("Patch max-core-limit", processSetRequest(coreFileCleanupUrl+"/config/max-core-limit",
		`{"openconfig-system-ext:max-core-limit":"8.25"}`, "PATCH", false))
	t.Run("Verify patch keeps the rest of the row", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", withCoreLimit("8.25"), false))

	t.Run("Replace config", processSetRequest(coreFileCleanupUrl+"/config",
		`{"openconfig-system-ext:config":{"max-core-limit":"12"}}`, "PUT", false))
	t.Run("Verify replace", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", withCoreLimit("12.0"), false))

	t.Run("Reject out-of-range max-core-limit", processSetRequest(coreFileCleanupUrl+"/config/max-core-limit",
		`{"openconfig-system-ext:max-core-limit":"100"}`, "PATCH", true))
	t.Run("Verify rejected value is not written", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", withCoreLimit("12.0"), false))

	t.Run("Delete max-core-limit", processDeleteRequest(coreFileCleanupUrl+"/config/max-core-limit", false))
	t.Run("Verify delete writes 0.0", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", withCoreLimit("0.0"), false))
	unloadDB(db.ConfigDB, autoTsGlobalCleanup)

	loadDB(db.ConfigDB, autoTsGlobalRow(map[string]interface{}{"state": "disabled", "max_core_limit": "5.0"}))
	t.Run("Get disabled state", processGetRequest(coreFileCleanupUrl+"/state/enabled", nil,
		`{"openconfig-system-ext:enabled":false}`, false))
	unloadDB(db.ConfigDB, autoTsGlobalCleanup)

	t.Run("Get without AUTO_TECHSUPPORT|GLOBAL", processGetRequest(coreFileCleanupUrl+"/config", nil, `{}`, false))
}

const techSupportCleanupUrl = "/openconfig-system:system/openconfig-system-ext:tech-support-cleanup"

func Test_openconfig_system_tech_support_cleanup(t *testing.T) {
	row := func(techsupportLimit, coreLimit string) map[string]interface{} {
		return autoTsGlobalRow(map[string]interface{}{
			"state":                 "enabled",
			"max_techsupport_limit": techsupportLimit,
			"max_core_limit":        coreLimit,
		})
	}

	loadDB(db.ConfigDB, row("10.0", "5.0"))
	t.Run("Get tech-support-cleanup", processGetRequest(techSupportCleanupUrl, nil,
		`{"openconfig-system-ext:tech-support-cleanup":{"config":{"max-techsupport-limit":"10"},"state":{"max-techsupport-limit":"10","enabled":true}}}`, false))

	t.Run("Patch max-techsupport-limit", processSetRequest(techSupportCleanupUrl+"/config/max-techsupport-limit",
		`{"openconfig-system-ext:max-techsupport-limit":"0.5"}`, "PATCH", false))
	t.Run("Verify patch keeps max_core_limit", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", row("0.5", "5.0"), false))

	t.Run("Reject out-of-range max-techsupport-limit", processSetRequest(techSupportCleanupUrl+"/config/max-techsupport-limit",
		`{"openconfig-system-ext:max-techsupport-limit":"-1"}`, "PATCH", true))

	t.Run("Delete max-techsupport-limit", processDeleteRequest(techSupportCleanupUrl+"/config/max-techsupport-limit", false))
	t.Run("Verify delete writes 0.0 and keeps max_core_limit", verifyDbResult(rclient, "AUTO_TECHSUPPORT|GLOBAL", row("0.0", "5.0"), false))
	unloadDB(db.ConfigDB, autoTsGlobalCleanup)
}
