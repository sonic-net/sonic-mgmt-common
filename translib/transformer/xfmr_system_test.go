//////////////////////////////////////////////////////////////////////////
//
// Copyright (c) 2024 Cisco Systems, Inc.
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
//////////////////////////////////////////////////////////////////////////

//go:build testapp
// +build testapp

package transformer_test

import (
	"errors"

	"github.com/Azure/sonic-mgmt-common/translib/db"

	//"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"os"
	"testing"
	"time"
)

var not_implemented_err error = errors.New("Not implemented")

func Test_oc_system_config(t *testing.T) {
	var pre_req_map, pre_req_map_empty, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/config container nodes  ++++++++++++")

	/* hostname */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/config/hostname node  ++++++++++++")
	url = "/openconfig-system:system/config"
	url_body_json = "{ \"openconfig-system:hostname\": \"unittest-hostname\"}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/config/hostname node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/config/hostname node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/config/hostname node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/config/hostname node ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:hostname\":\"unittest-hostname-2\"}"
	url = "/openconfig-system:system/config/hostname"
	t.Run("Test get on system/config/hostname node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/config/hostname node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/config/hostname node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-3"}}}
	url = "/openconfig-system:system/config/hostname"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"NULL": "NULL"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test delete on system/config/hostname node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on system/config/hostname node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/config/hostname node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/config/hostname node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/config/hostname"
	url_body_json = "{ \"openconfig-system:hostname\": \"unittest-hostname-new\"}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-new"}}}
	t.Run("Test PATCH(Update) on system/config/hostname node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/config/hostname node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/config/hostname node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/config/hostname node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/config/hostname"
	url_body_json = "{ \"openconfig-system:hostname\": \"unittest-hostname-new\"}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-new"}}}
	t.Run("Test PUT(Replace) on system/config/hostname node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/config/hostname node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/config/hostname node  ++++++++++++")

	/* login-banner */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/config/login-banner node  ++++++++++++")
	url = "/openconfig-system:system/config"
	url_body_json = "{ \"openconfig-system:login-banner\": \"unittest login banner\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/config/login-banner node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/config/login-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/config/login-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/config/login-banner node ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:login-banner\":\"unittest login banner 2\"}"
	url = "/openconfig-system:system/config/login-banner"
	t.Run("Test get on system/config/login-banner node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{}}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/config/login-banner node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/config/login-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/login-banner"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"NULL": "NULL"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test Delete on system/config/login-banner node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Delete on system/config/login-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/config/login-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/config/login-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/login-banner"
	url_body_json = "{ \"openconfig-system:login-banner\": \"unittest login banner new\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner new"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/config/login-banner node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/config/login-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/config/login-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/config/login-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/login-banner"
	url_body_json = "{ \"openconfig-system:login-banner\": \"unittest login banner new\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner new"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/config/login-banner node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/config/login-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/config/login-banner node  ++++++++++++")

	/* motd-banner */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/config/motd-banner node  ++++++++++++")
	url = "/openconfig-system:system/config"
	url_body_json = "{ \"openconfig-system:motd-banner\": \"unittest motd banner\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/config/motd-banner node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/config/motd-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/config/motd-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/config/motd-banner node ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner 2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:motd-banner\":\"unittest motd banner 2\"}"
	url = "/openconfig-system:system/config/motd-banner"
	t.Run("Test get on system/config/motd-banner node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/config/motd-banner node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/config/motd-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner 2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/motd-banner"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"NULL": "NULL"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test Delete on system/config/motd-banner node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Delete on system/config/motd-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/config/motd-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/config/motd-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/motd-banner"
	url_body_json = "{ \"openconfig-system:motd-banner\": \"unittest motd banner new\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner new"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/config/motd-banner node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/config/motd-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/config/motd-banner node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/config/motd-banner node  ++++++++++++")
	pre_req_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner old"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/config/motd-banner"
	url_body_json = "{ \"openconfig-system:motd-banner\": \"unittest motd banner new\"}"
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"motd": "unittest motd banner new"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/config/motd-banner node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/config/motd-banner node", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/config/motd-banner node  ++++++++++++")

	/* config container */
	t.Log("\n\n+++++++++++++ Performing Get on system/config node ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-3"}}, "BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 3", "motd": "unittest motd banner 3"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:config\":{\"hostname\":\"unittest-hostname-3\", \"login-banner\":\"unittest login banner 3\", \"motd-banner\":\"unittest motd banner 3\"}}"
	url = "/openconfig-system:system/config"
	t.Run("Test get on system/config node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	/*
		expected_get_json = "{\"openconfig-system:system\": {\"config\":{\"hostname\":\"unittest-hostname-3\", \"login-banner\":\"unittest login banner 3\", \"motd-banner\":\"unittest motd banner 3\"}}}"
		url = "/openconfig-system:system"
		t.Run("Test get on system node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
	*/
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}, "BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/config node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/config node  ++++++++++++")
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system"
	url_body_json = "{ \"openconfig-system:config\": {\"hostname\": \"unittest-hostname-4\", \"login-banner\":\"unittest login banner 4\", \"motd-banner\":\"unittest motd banner 4\"}}"
	t.Run("Test POST(Create) on system/config node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-4"}}}
	t.Run("Verify POST(Create) on system/config node DEV_META", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 4", "motd": "unittest motd banner 4"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	t.Run("Verify POST(Create) on system/config node BANNERS", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-3"}}, "BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 3", "motd": "unittest motd banner 3"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/config"
	t.Run("Test Delete on system/config node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify Delete on system/config node DEV_META", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"NULL": "NULL"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	t.Run("Verify Delete on system/config node BANNERS", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-3"}}, "BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 3", "motd": "unittest motd banner 3"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/config"
	url_body_json = "{ \"openconfig-system:config\": {\"hostname\": \"unittest-hostname-4\", \"login-banner\":\"unittest login banner 4\", \"motd-banner\":\"unittest motd banner 4\"}}"
	t.Run("Test PATCH(Update) on system/config node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-4"}}}
	t.Run("Verify PATCH(Update) on system/config node DEV_META", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 4", "motd": "unittest motd banner 4"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	t.Run("Verify PATCH(Update) on system/config node BANNERS", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-3"}}, "BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 3", "motd": "unittest motd banner 3"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/config"
	url_body_json = "{ \"openconfig-system:config\": {\"hostname\": \"unittest-hostname-4\", \"login-banner\":\"unittest login banner 4\", \"motd-banner\":\"unittest motd banner 4\"}}"
	t.Run("Test PUT(Replace) on system/config node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"hostname": "unittest-hostname-4"}}}
	t.Run("Verify PUT(Replace) on system/config node DEV_META", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": map[string]interface{}{"login": "unittest login banner 4", "motd": "unittest motd banner 4"}}}
	cleanuptbl = map[string]interface{}{"BANNER_MESSAGE": map[string]interface{}{"global": ""}}
	t.Run("Verify PUT(Replace) on system/config node BANNERS", verifyDbResult(rclient, "BANNER_MESSAGE|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/config container nodes  ++++++++++++")
}

func Test_oc_system_state(t *testing.T) {
	var url, expected_get_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/state container nodes  ++++++++++++")

	/* hostname */
	t.Log("\n\n+++++++++++++ Performing Get on system/state/hostname node ++++++++++++")
	url = "/openconfig-system:system/state/hostname"
	t.Run("Test get on system/state/hostname node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/state/hostname node ++++++++++++")

	/* login-banner */
	t.Log("\n\n+++++++++++++ Performing Get on system/state/login-banner node ++++++++++++")
	url = "/openconfig-system:system/state/login-banner"
	t.Run("Test get on system/state/login-banner node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/state/login-banner node ++++++++++++")

	/* motd-banner */
	t.Log("\n\n+++++++++++++ Performing Get on system/state/motd-banner node ++++++++++++")
	url = "/openconfig-system:system/state/motd-banner"
	t.Run("Test get on system/state/motd-banner node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/state/motd-banner node ++++++++++++")

	/* current-datetime */
	/*
		t.Log("\n\n+++++++++++++ Performing Get on system/state/current-datetime node ++++++++++++")
		expected_get_json = "{\"openconfig-system:current-datetime\":\"2024-08-15T10:11:41Z+00:00\"}"
		url = "/openconfig-system:system/state/current-datetime"
		t.Run("Test get on system/state/current-datetime node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Performing Get on system/state/current-datetime node ++++++++++++")
	*/

	/* up-time */
	/*
		t.Log("\n\n+++++++++++++ Performing Get on system/state/up-time node ++++++++++++")
		expected_get_json = "{\"openconfig-system:up-time\":\"13627000000000\"}"
		url = "/openconfig-system:system/state/up-time"
		t.Run("Test get on system/state/up-time node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing Get on system/state/up-time node ++++++++++++")
	*/

	/* boot-time */
	/*
		t.Log("\n\n+++++++++++++ Performing Get on system/state/boot-time node ++++++++++++")
		expected_get_json = "{\"openconfig-system:boot-time\":\"1698025723767651793\"}"
		url = "/openconfig-system:system/state/boot-time"
		t.Run("Test get on system/state/boot-time node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing Get on system/state/boot-time node ++++++++++++")
	*/

	/* software-version */
	t.Log("\n\n+++++++++++++ Performing Get on system/state/software-version node ++++++++++++")
	os.MkdirAll("/etc/sonic", 0755)
	yamlContent := []byte("build_version: sonic-version\n")
	if err := os.WriteFile("/etc/sonic/sonic_version.yml", yamlContent, 0644); err != nil {
		t.Fatalf("failed to write sonic_version.yml: %v", err)
	}
	expected_get_json = "{\"openconfig-system:software-version\":\"sonic-version\"}"
	url = "/openconfig-system:system/state/software-version"
	t.Run("Test get on system/state/software-version node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	os.Remove("/etc/sonic/sonic_version.yml")
	t.Log("\n\n+++++++++++++ Done Performing Get on system/state/software-version node ++++++++++++")

	/* state container */
	/*
		t.Log("\n\n+++++++++++++ Performing Get on system/state node ++++++++++++")
		pre_req_map = map[string]interface{}{"VERSIONS": map[string]interface{}{"SOFTWARE": map[string]interface{}{"VERSION": "unittest sonic version"}, "DATABASE": map[string]interface{}{"VERSION": "unittest database version"}}}
		loadDB(db.ConfigDB, pre_req_map)
		expected_get_json = "{\"openconfig-system:state\":{\"boot-time\":\"1698025723717492667\", \"current-datetime\":\"2024-08-15T11:10:46Z+00:00\", \"up-time\":\"25694523000000000\", \"software-version\":\"unittest sonic version\"}}"
		url = "/openconfig-system:system/state"
		t.Run("Test get on system/state node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		expected_get_json = "{\"openconfig-system:system\":{\"state\":{\"boot-time\":\"1698025723717492667\", \"current-datetime\":\"2024-08-15T11:10:46Z+00:00\", \"up-time\":\"25694523000000000\", \"software-version\":\"unittest sonic version\"}}}"
		url = "/openconfig-system:system"
		t.Run("Test get on system node", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		cleanuptbl = map[string]interface{}{"VERSIONS": map[string]interface{}{"SOFTWARE": "", "DATABASE": ""}}
		unloadDB(db.ConfigDB, cleanuptbl)
		t.Log("\n\n+++++++++++++ Done Performing Get on system/state node ++++++++++++")
	*/

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/state container nodes  ++++++++++++")
}

func Test_oc_system_processes(t *testing.T) {
	var pre_req_map, cleanuptbl map[string]interface{}
	var url string

	t.Log("\n\n+++++++++++++ Performing Get on system/processes node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/processes supported nodes ++++++++++++")
	pre_req_map = map[string]interface{}{"PROCESS_STATS": map[string]interface{}{"60000": map[string]interface{}{"%CPU": "5.3", "%MEM": "20.4", "CMD": "ut-proc-name --ut-arg1 --ut-arg2"}}}
	loadDB(db.StateDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:process\":[{\"pid\":\"60000\",\"state\":{\"args\":[\"--ut-arg1\",\"--ut-arg2\"],\"cpu-utilization\":5,\"memory-utilization\":20,\"name\":\"ut-proc-name\",\"pid\":\"60000\"}}]}"
	url = "/openconfig-system:system/processes/process"
	t.Run("Test get on system/processes node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/processes supported nodes ++++++++++++")

	t.Log("\n\n+++++++++++++ Done Performing Get on system/processes node ++++++++++++")
}

func Test_oc_system_clock(t *testing.T) {
	var pre_req_map, pre_req_map_empty, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/clock container nodes  ++++++++++++")

	/* timezone-name */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/clock/config node  ++++++++++++")
	url = "/openconfig-system:system/clock/config"
	url_body_json = "{ \"openconfig-system:timezone-name\": \"Cuba\"}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Cuba"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/clock/config/timezone-name node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/clock/config/timezone-name node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/clock/config/timezone-name node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/clock/config node (Wrong input)  ++++++++++++")
	url = "/openconfig-system:system/clock/config"
	url_body_json = "{ \"openconfig-system:timezone-name\": \"India\"}"
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	expected_err := errors.New("Timezone India does not conform format")
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/clock/config/timezone-name node", processSetRequest(url, url_body_json, "POST", true, expected_err))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/clock/config/timezone-name node (Wrong input)  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/clock/config/timezone-name node ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Indian/Mauritius"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/clock/config/timezone-name"
	t.Run("Test delete on system/clock/config/timezone-name node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify POST(Create) on system/clock/config/timezone-name node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))

	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/clock/config/timezone-name node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/clock/config/timezone-name node ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Indian/Mauritius"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:timezone-name\":\"Indian/Mauritius\"}"
	url = "/openconfig-system:system/clock/config/timezone-name"
	t.Run("Test get on system/clock/config/timezone-name node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	expected_get_json = "{\"openconfig-system:clock\":{\"config\":{\"timezone-name\":\"Indian/Mauritius\"}}}"
	url = "/openconfig-system:system/clock"
	t.Run("Test get on system/clock node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/clock/config/timezone-name node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/clock/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Indian/Mauritius"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/clock/config"
	url_body_json = "{ \"openconfig-system:config\": { \"timezone-name\": \"Cuba\"}}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Cuba"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/clock/config/timezone-name node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/clock/config/timezone-name node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/clock/config/timezone-name node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/clock/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Indian/Mauritius"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system"
	url_body_json = "{ \"openconfig-system:system\": {\"clock\": {\"config\": { \"timezone-name\": \"Cuba\"}}}}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "Cuba"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/clock/config/timezone-name node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/clock/config/timezone-name node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/clock/config/timezone-name node  ++++++++++++")

	// config post
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/clock/config node  ++++++++++++")
	url = "/openconfig-system:system/clock/config"
	url_body_json = "{ \"openconfig-system:timezone-name\": \"UTC\"}"
	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "UTC"}}}
	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/clock/config node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/clock/config node", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/clock/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/clock/config ++++++++++++")
	pre_req_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"timezone": "UTC"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/clock/config"
	t.Run("Test delete on system/clock/config node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	expected_map = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify POST(Create) on system/clock/config", verifyDbResult(rclient, "DEVICE_METADATA|localhost", expected_map, false))

	cleanuptbl = map[string]interface{}{"DEVICE_METADATA": map[string]interface{}{"localhost": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/clock/config node ++++++++++++")

	/* clock/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/clock/state node ++++++++++++")
	url = "/openconfig-system:system/clock/state"
	t.Run("Test get on system/clock/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/clock/state node ++++++++++++")
}

func Test_oc_system_ssh_server(t *testing.T) {
	var pre_req_map, pre_req_map_empty, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/ssh-server container nodes  ++++++++++++")

	/* timeout */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ssh-server/config node  ++++++++++++")
	url = "/openconfig-system:system/ssh-server/config"
	url_body_json = "{ \"openconfig-system:timeout\": 500}"
	expected_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "500"}}}
	cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/ssh-server/config/timeout node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/ssh-server/config/timeout node", verifyDbResult(rclient, "SSH_SERVER|POLICIES", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ssh-server/config/timeout node  ++++++++++++")

	/*
		t.Log("\n\n+++++++++++++ Performing Invalid Set on system/ssh-server/config node (Wrong input)  ++++++++++++")
		url = "/openconfig-system:system/ssh-server/config"
		url_body_json = "{ \"openconfig-system:timeout\": 700}"
		cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
		loadDB(db.ConfigDB, pre_req_map_empty)
		expected_err = tlerr.InvalidArgsError{Format: "Field \"login_timeout\" has invalid value \"700\""}
		time.Sleep(1 * time.Second)
		t.Run("Test invalid set on system/ssh-server/config/timeout node", processSetRequest(url, url_body_json, "POST", true, expected_err))
		time.Sleep(1 * time.Second)
		unloadDB(db.ConfigDB, cleanuptbl)
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing Invalid Set on system/ssh-server/config/timeout node (Wrong input)  ++++++++++++")
	*/

	t.Log("\n\n+++++++++++++ Performing Get on system/ssh-server/config/timeout node ++++++++++++")
	pre_req_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "300"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:timeout\":300}"
	url = "/openconfig-system:system/ssh-server/config/timeout"
	t.Run("Test get on system/ssh-server/config/timeout node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ssh-server/config/timeout node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/ssh-server/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "300"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system"
	expected_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{}}}
	cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test DELETE on system/ssh-server/config/timeout node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify DELETE on system/ssh-server/config/timeout node", verifyDbResult(rclient, "SSH_SERVER|POLICIES", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing DELETE on system/ssh-server/config/timeout node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/ssh-server/config top level node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "300"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ssh-server/config"
	expected_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"NULL": "NULL"}}}
	cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test DELETE on system/ssh-server/config/ node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify DELETE on system/ssh-server/config/timeout node", verifyDbResult(rclient, "SSH_SERVER|POLICIES", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing DELETE on system/ssh-server/config/ top level node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/ssh-server/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "300"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ssh-server/config/timeout"
	url_body_json = "{ \"openconfig-system:timeout\": 500}"
	expected_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "500"}}}
	cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/ssh-server/config/timeout node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/ssh-server/config/timeout node", verifyDbResult(rclient, "SSH_SERVER|POLICIES", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/ssh-server/config/timeout node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/ssh-server/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "300"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ssh-server/config/timeout"
	url_body_json = "{ \"openconfig-system:timeout\": 500}"
	expected_map = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": map[string]interface{}{"login_timeout": "500"}}}
	cleanuptbl = map[string]interface{}{"SSH_SERVER": map[string]interface{}{"POLICIES": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/ssh-server/config/timeout node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/ssh-server/config/timeout node", verifyDbResult(rclient, "SSH_SERVER|POLICIES", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/ssh-server/config/timeout node  ++++++++++++")

	/* ssh-server/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/ssh-server/state node ++++++++++++")
	url = "/openconfig-system:system/ssh-server/state"
	t.Run("Test get on system/ssh-server/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ssh-server/state node ++++++++++++")

	/* ssh-server/state/timeout */
	t.Log("\n\n+++++++++++++ Performing Get on system/ssh-server/state/timeout node ++++++++++++")
	url = "/openconfig-system:system/ssh-server/state/timeout"
	t.Run("Test get on system/ssh-server/state/timeout node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ssh-server/state/timeout node ++++++++++++")

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/ssh-server container nodes  ++++++++++++")
}

func Test_oc_system_messages(t *testing.T) {
	var pre_req_map, pre_req_map_empty, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/messages container nodes  ++++++++++++")

	/* severity */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/messages/config/severity node  ++++++++++++")
	url = "/openconfig-system:system/messages/config"
	url_body_json = "{ \"openconfig-system:severity\": \"ERROR\"}"
	expected_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "error"}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/messages/config/severity node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/messages/config/severity node", verifyDbResult(rclient, "SYSLOG_CONFIG|GLOBAL", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/messages/config/severity node  ++++++++++++")
	// Final cleanup for SYSLOG_CONFIG
	cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	/*
		t.Log("\n\n+++++++++++++ Performing POST(Create) on system/messages/config/severity node (Wrong input)  ++++++++++++")
		url = "/openconfig-system:system/messages/config"
		url_body_json = "{ \"openconfig-system:severity\": \"WRONG-SEV\"}"
		cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
		loadDB(db.ConfigDB, pre_req_map_empty)
		expected_err = errors.New("Invalid input")
		time.Sleep(1 * time.Second)
		t.Run("Test POST(Create) on system/messages/config/severity node", processSetRequest(url, url_body_json, "POST", true, expected_err))
		time.Sleep(1 * time.Second)
		unloadDB(db.ConfigDB, cleanuptbl)
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/messages/config/severity node (Wrong input)  ++++++++++++")
	*/

	t.Log("\n\n+++++++++++++ Performing Get on system/messages/config/severity node ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "crit"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:severity\":\"CRITICAL\"}"
	url = "/openconfig-system:system/messages/config/severity"
	t.Run("Test get on system/messages/config/severity node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	expected_get_json = "{\"openconfig-system:messages\":{\"config\":{\"severity\":\"CRITICAL\"}}}"
	url = "/openconfig-system:system/messages"
	t.Run("Test get on system/messages node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/messages/config/severity node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Delete on system/messages/config/severity node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "crit"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/messages/config"
	expected_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test Delete on system/messages/config/severity node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify Delete on system/messages/config/severity node", verifyDbResult(rclient, "SYSLOG_CONFIG|GLOBAL", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Delete on system/messages/config/severity node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/messages/config/severity node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "crit"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/messages/config/severity"
	url_body_json = "{ \"openconfig-system:severity\": \"ERROR\"}"
	expected_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "error"}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/messages/config/severity node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/messages/config/severity node", verifyDbResult(rclient, "SYSLOG_CONFIG|GLOBAL", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/messages/config/severity node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/messages/config/severity node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "crit"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/messages/config/severity"
	url_body_json = "{ \"openconfig-system:severity\": \"ERROR\"}"
	expected_map = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": map[string]interface{}{"severity": "error"}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_CONFIG": map[string]interface{}{"GLOBAL": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/messages/config/severity node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/messages/config/severity node", verifyDbResult(rclient, "SYSLOG_CONFIG|GLOBAL", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/messages/config/severity node  ++++++++++++")

	/* messages/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/messages/state node ++++++++++++")
	url = "/openconfig-system:system/messages/state"
	t.Run("Test get on system/messages/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/messages/state node ++++++++++++")

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/messages container nodes  ++++++++++++")
}

func Test_oc_system_logging(t *testing.T) {
	var pre_req_map, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json, expected_get_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/logging container nodes  ++++++++++++")

	/* remote-servers/remote-server */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/logging/remote-servers/remote-server node  ++++++++++++")
	url = "/openconfig-system:system/logging/remote-servers"
	url_body_json = "{\"openconfig-system:remote-server\": [{\"host\": \"1.1.1.1\", \"config\": { \"host\": \"1.1.1.1\", \"source-address\": \"10.10.10.10\", \"network-instance\": \"Vrf1\", \"remote-port\": 10000 } } ]}"
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"source": "10.10.10.10", "vrf": "Vrf1", "port": "10000"}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": ""}}
	pre_req_map = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": map[string]interface{}{"vni": 100}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/logging/remote-servers/remote-server node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/logging/remote-servers/remote-server node", verifyDbResult(rclient, "SYSLOG_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/logging/remote-servers/remote-server node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/logging/remote-servers/remote-server node ++++++++++++")
	pre_req_map = map[string]interface{}{"VRF": map[string]interface{}{"Vrf2": map[string]interface{}{"vni": 200}}, "SYSLOG_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"source": "20.20.20.20", "vrf": "Vrf2", "port": "20000"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:config\": { \"host\": \"2.2.2.2\", \"source-address\": \"20.20.20.20\", \"network-instance\": \"Vrf2\", \"remote-port\": 20000 } }"
	url = "/openconfig-system:system/logging/remote-servers/remote-server[host=2.2.2.2]/config"
	t.Run("Test get on system/logging/remote-servers/remote-server node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging/remote-servers/remote-server node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/logging node ++++++++++++")
	expected_get_json = "{\"openconfig-system:logging\": {\"remote-servers\": { \"remote-server\": [{\"host\": \"2.2.2.2\", \"config\": { \"host\": \"2.2.2.2\", \"source-address\": \"20.20.20.20\", \"network-instance\": \"Vrf2\", \"remote-port\": 20000 } } ]}}}"
	url = "/openconfig-system:system/logging"
	t.Run("Test get on system/logging node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"2.2.2.2": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging node ++++++++++++")

	/* logging/remote-servers/remote-server/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/logging/remote-servers/remote-server/state node ++++++++++++")
	url = "/openconfig-system:system/logging/remote-servers/remote-server/state"
	t.Run("Test get on system/logging/remote-servers/remote-server/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging/remote-servers/remote-server/state node ++++++++++++")

	/* remote-servers/remote-server/selectors/selector */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/logging/remote-servers/remote-server/selectors/selector node  ++++++++++++")
	pre_req_map = map[string]interface{}{"VRF": map[string]interface{}{"Vrf3": map[string]interface{}{"vni": 300}, "Vrf4": map[string]interface{}{"vni": 400}}, "SYSLOG_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"source": "30.30.30.30", "vrf": "Vrf3", "port": "30000"}}}
	url = "/openconfig-system:system/logging/remote-servers/remote-server[host=3.3.3.3]/selectors"
	url_body_json = "{\"openconfig-system:selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}"
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"source": "30.30.30.30", "vrf": "Vrf3", "port": "30000", "severity": "debug"}}}
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"3.3.3.3": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/logging/remote-servers/remote-server/selectors/selector node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/logging/remote-servers/remote-server/selectors/selector node", verifyDbResult(rclient, "SYSLOG_SERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/logging/remote-servers/remote-server/selectors/selector node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/logging/remote-servers/remote-server/selectors/selector node ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "info"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:selector\": [{\"facility\": \"openconfig-system-logging:ALL\", \"severity\": \"INFORMATIONAL\", \"config\": { \"facility\": \"openconfig-system-logging:ALL\", \"severity\": \"INFORMATIONAL\" } } ]}"
	url = "/openconfig-system:system/logging/remote-servers/remote-server[host=4.4.4.4]/selectors/selector"
	t.Run("Test get on system/logging/remote-servers/remote-server/selectors/selector node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging/remote-servers/remote-server/selectors/selector node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/logging node ++++++++++++")
	expected_get_json = "{\"openconfig-system:logging\": {\"remote-servers\": { \"remote-server\": [{\"host\": \"4.4.4.4\", \"config\": { \"host\": \"4.4.4.4\", \"source-address\": \"40.40.40.40\", \"network-instance\": \"Vrf4\", \"remote-port\": 40000 }, \"selectors\": { \"selector\": [{\"facility\": \"openconfig-system-logging:ALL\", \"severity\": \"INFORMATIONAL\", \"config\": { \"facility\": \"openconfig-system-logging:ALL\", \"severity\": \"INFORMATIONAL\" } } ] }}]}}}"
	url = "/openconfig-system:system/logging"
	t.Run("Test get on system/logging node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/logging/remote-servers/remote-server node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "info"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/logging/remote-servers/remote-server[host=4.4.4.4]"
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{}}
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test DELETE on system/logging/remote-servers/remote-server[host=4.4.4.4] node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify DELETE on system/logging/remote-servers/remote-server[host=4.4.4.4] node", verifyDbResult(rclient, "SYSLOG_SERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing DELETE on system/logging/remote-servers/remote-server node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/logging/remote-servers node  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "info"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/logging/remote-servers"
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{}}
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": ""}}
	time.Sleep(1 * time.Second)
	t.Run("Test DELETE on system/logging/remote-servers node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify DELETE on system/logging/remote-servers node", verifyDbResult(rclient, "SYSLOG_SERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing DELETE on system/logging/remote-servers node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/logging/remote-servers/remote-server  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "info"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/logging/remote-servers/remote-server[host=4.4.4.4]"
	url_body_json = "{\"openconfig-system:remote-server\": [{ \"host\": \"4.4.4.4\", \"config\": { \"host\": \"4.4.4.4\", \"source-address\": \"10.10.10.10\", \"network-instance\": \"Vrf1\", \"remote-port\": 10000}, \"selectors\": {  \"selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}}]}"
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"source": "10.10.10.10", "vrf": "Vrf1", "port": "10000", "severity": "debug"}}}
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/logging/remote-servers/remote-server node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/logging/remote-servers/remote-server node", verifyDbResult(rclient, "SYSLOG_SERVER|4.4.4.4", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"4.4.4.4": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/logging/remote-servers/remote-server node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/logging  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"source": "10.10.10.10", "vrf": "Vrf1", "port": "10000", "severity": "info"}, "2.2.2.2": map[string]interface{}{"source": "20.20.20.20", "vrf": "Vrf2", "port": "20000", "severity": "debug"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/logging"
	url_body_json = "{\"openconfig-system:logging\": { \"remote-servers\": { \"remote-server\": [{ \"host\": \"3.3.3.3\", \"config\": { \"host\": \"3.3.3.3\", \"source-address\": \"30.30.30.30\", \"network-instance\": \"Vrf3\", \"remote-port\": 30000}, \"selectors\": { \"selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}}, { \"host\": \"1.1.1.1\", \"config\": { \"host\": \"1.1.1.1\", \"source-address\": \"40.40.40.40\", \"network-instance\": \"Vrf4\", \"remote-port\": 40000}, \"selectors\": { \"selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}}]}}}"
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/logging node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"source": "30.30.30.30", "vrf": "Vrf3", "port": "30000", "severity": "debug"}}}
	t.Run("Verify PUT(Replace) on system/logging node [create]", verifyDbResult(rclient, "SYSLOG_SERVER|3.3.3.3", expected_map, false))
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "debug"}}}
	t.Run("Verify PUT(Replace) on system/logging node [update]", verifyDbResult(rclient, "SYSLOG_SERVER|1.1.1.1", expected_map, false))
	/*expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{}}}
	t.Run("Verify PUT(Replace) on system/logging node [delete]", verifyDbResult(rclient, "SYSLOG_SERVER|2.2.2.2", expected_map, false))*/
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": "", "3.3.3.3": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/logging node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/logging/remote-servers  ++++++++++++")
	pre_req_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"source": "10.10.10.10", "vrf": "Vrf1", "port": "10000", "severity": "info"}, "2.2.2.2": map[string]interface{}{"source": "20.20.20.20", "vrf": "Vrf2", "port": "20000", "severity": "debug"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/logging/remote-servers"
	url_body_json = "{\"openconfig-system:remote-servers\": { \"remote-server\": [{ \"host\": \"3.3.3.3\", \"config\": { \"host\": \"3.3.3.3\", \"source-address\": \"30.30.30.30\", \"network-instance\": \"Vrf3\", \"remote-port\": 30000}, \"selectors\": { \"selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}}, { \"host\": \"1.1.1.1\", \"config\": { \"host\": \"1.1.1.1\", \"source-address\": \"40.40.40.40\", \"network-instance\": \"Vrf4\", \"remote-port\": 40000}, \"selectors\": { \"selector\": [{\"facility\": \"ALL\", \"severity\": \"DEBUG\", \"config\": { \"facility\": \"ALL\", \"severity\": \"DEBUG\" } } ]}}]}}"
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/logging/remote-servers node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"source": "30.30.30.30", "vrf": "Vrf3", "port": "30000", "severity": "debug"}}}
	t.Run("Verify PUT(Replace) on system/logging/remote-servers node [create]", verifyDbResult(rclient, "SYSLOG_SERVER|3.3.3.3", expected_map, false))
	expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"source": "40.40.40.40", "vrf": "Vrf4", "port": "40000", "severity": "debug"}}}
	t.Run("Verify PUT(Replace) on system/logging/remote-servers node [update]", verifyDbResult(rclient, "SYSLOG_SERVER|1.1.1.1", expected_map, false))
	/*expected_map = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{}}}
	t.Run("Verify PUT(Replace) on system/logging/remote-servers node [delete]", verifyDbResult(rclient, "SYSLOG_SERVER|2.2.2.2", expected_map, false))*/
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"SYSLOG_SERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": "", "3.3.3.3": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/logging/remote-servers node  ++++++++++++")

	/* logging/remote-servers/remote-server/selectors/selector/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/logging/remote-servers/remote-server/selectors/selector/state node ++++++++++++")
	url = "/openconfig-system:system/logging/remote-servers/remote-server/selectors/selector/state"
	t.Run("Test get on system/logging/remote-servers/remote-server/selectors/selector/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/logging/remote-servers/remote-server/selectors/selector/state node ++++++++++++")

	cleanuptbl = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": "", "Vrf2": "", "Vrf3": "", "Vrf4": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/logging container nodes  ++++++++++++")
}

func Test_oc_system_ntp(t *testing.T) {
	var pre_req_map, pre_req_map_empty, expected_map, cleanuptbl map[string]interface{}
	var url, url_body_json string

	t.Log("\n\n+++++++++++++ Performing unit tests on system/ntp/config container nodes  ++++++++++++")

	/* enabled */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ntp/config/enabled node  ++++++++++++")
	url = "/openconfig-system:system/ntp/config"
	url_body_json = "{ \"openconfig-system:enabled\": true}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "authentication": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/ntp/config/enabled node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/ntp/config/enabled node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ntp/config/enabled node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/config/enabled node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "disabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json := "{\"openconfig-system:enabled\":false}"
	url = "/openconfig-system:system/ntp/config/enabled"
	t.Run("Test get on system/ntp/config/enabled node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/config/enabled node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing delete on system/ntp/config/enabled node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	url = "/openconfig-system:system/ntp/config/enabled"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test delete on system/ntp/config/enabled node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on system/ntp/config/enabled node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/ntp/config/enabled node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/ntp/config/enabled node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "disabled"}}}
	url = "/openconfig-system:system/ntp/config/enabled"
	url_body_json = "{ \"openconfig-system:enabled\": true}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/ntp/config/enabled node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/ntp/config/enabled node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/ntp/config/enabled node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/ntp/config/enabled node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	url = "/openconfig-system:system/ntp/config/enabled"
	url_body_json = "{ \"openconfig-system:enabled\": false}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/ntp/config/enabled node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/ntp/config/enabled node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/ntp/config/enabled node  ++++++++++++")

	/* enable-ntp-auth */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ntp/config/enable-ntp-auth node  ++++++++++++")
	url = "/openconfig-system:system/ntp/config"
	url_body_json = "{ \"openconfig-system:enable-ntp-auth\": true}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "enabled", "admin_state": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/ntp/config/enable-ntp-auth node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/ntp/config/enable-ntp-auth node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ntp/config/enable-ntp-auth node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/config/enable-ntp-auth node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "disabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:enable-ntp-auth\":false}"
	url = "/openconfig-system:system/ntp/config/enable-ntp-auth"
	t.Run("Test get on system/ntp/config/enable-ntp-auth node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/config/enable-ntp-auth node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/ntp/config/enable-ntp-auth node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "disabled"}}}
	url = "/openconfig-system:system/ntp/config/enable-ntp-auth"
	url_body_json = "{ \"openconfig-system:enable-ntp-auth\": true}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "enabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/ntp/config/enable-ntp-auth node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/ntp/config/enable-ntp-auth node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/ntp/config/enable-ntp-auth node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/ntp/config/enable-ntp-auth node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "enabled"}}}
	url = "/openconfig-system:system/ntp/config/enable-ntp-auth"
	url_body_json = "{ \"openconfig-system:enable-ntp-auth\": false}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/ntp/config/enable-ntp-auth node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/ntp/config/enable-ntp-auth node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/ntp/config/enable-ntp-auth node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing delete on system/ntp/config/enable-ntp-auth node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "enabled"}}}
	url = "/openconfig-system:system/ntp/config/enable-ntp-auth"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"authentication": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test delete on system/ntp/config/enable-ntp-auth node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on system/ntp/config/enable-ntp-auth node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/ntp/config/enable-ntp-auth node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ntp/config node  ++++++++++++")
	url = "/openconfig-system:system/ntp/config"
	url_body_json = "{ \"openconfig-system:enabled\": true, \"openconfig-system:enable-ntp-auth\": false}"
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "authentication": "disabled"}}}
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/ntp/config node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/ntp/config node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ntp/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/config node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "disabled", "authentication": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:config\":{\"enabled\":false, \"enable-ntp-auth\":true}}"
	url = "/openconfig-system:system/ntp/config"
	t.Run("Test get on system/ntp/config node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/ntp/config container nodes  ++++++++++++")

	/* ntp/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/state node ++++++++++++")
	url = "/openconfig-system:system/ntp/state"
	t.Run("Test get on system/ntp/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/state node ++++++++++++")

	/* ntp-keys/ntp-key */
	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")
	url = "/openconfig-system:system/ntp/ntp-keys"
	url_body_json = "{\"openconfig-system:ntp-key\": [{\"key-id\": 100, \"config\": { \"key-id\": 100, \"key-type\": \"NTP_AUTH_MD5\", \"key-value\": \"key-value-1\"} } ]}"
	expected_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-1"}}}
	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": ""}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test POST(Create) on system/ntp/ntp-keys/ntp-key node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST(Create) on system/ntp/ntp-keys/ntp-key node", verifyDbResult(rclient, "NTP_KEY|100", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/ntp-keys/ntp-key node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"200": map[string]interface{}{"type": "md5", "value": "key-value-2"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:config\": { \"key-id\": 200, \"key-type\": \"openconfig-system:NTP_AUTH_MD5\", \"key-value\": \"key-value-2\"} }"
	url = "/openconfig-system:system/ntp/ntp-keys/ntp-key[key-id=200]/config"
	t.Run("Test get on system/ntp/ntp-keys/ntp-key node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/ntp-keys/ntp-key node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PUT(Replace) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-2"}}}
	url = "/openconfig-system:system/ntp/ntp-keys/ntp-key[key-id=100]"
	url_body_json = "{\"openconfig-system:ntp-key\": [{\"key-id\": 100, \"config\": { \"key-id\": 100, \"key-type\": \"NTP_AUTH_MD5\", \"key-value\": \"key-value-3\"} } ]}"
	expected_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-3"}}}
	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PUT(Replace) on system/ntp/ntp-keys/ntp-key node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT(Replace) on system/ntp/ntp-keys/ntp-key node", verifyDbResult(rclient, "NTP_KEY|100", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PUT(Replace) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing PATCH(Update) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-1"}}}
	url = "/openconfig-system:system/ntp/ntp-keys/ntp-key[key-id=100]"
	url_body_json = "{\"openconfig-system:ntp-key\": [{\"key-id\": 100, \"config\": { \"key-id\": 100, \"key-type\": \"NTP_AUTH_MD5\", \"key-value\": \"key-value-2\"} } ]}"
	expected_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-2"}}}
	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test PATCH(Update) on system/ntp/ntp-keys/ntp-key node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH(Update) on system/ntp/ntp-keys/ntp-key node", verifyDbResult(rclient, "NTP_KEY|100", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing PATCH(Update) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DEL(Delete) on system/ntp/ntp-keys/ntp-key node  ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value-1"}}}
	url = "/openconfig-system:system/ntp/ntp-keys/ntp-key[key-id=100]"
	expected_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{}}
	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Run("Test DELETE on system/ntp/ntp-keys/ntp-key node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify DELETE on system/ntp/ntp-keys/ntp-key node", verifyDbResult(rclient, "NTP_KEY|100", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing DELETE on system/ntp/ntp-keys/ntp-key node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp node ++++++++++++")
	expected_get_json = "{\"openconfig-system:ntp\": {\"ntp-keys\": { \"ntp-key\": [{\"key-id\": 200, \"config\": { \"key-id\": 200, \"key-type\": \"openconfig-system:NTP_AUTH_MD5\", \"key-value\": \"key-value-2\"} } ]}}}"
	url = "/openconfig-system:system/ntp"
	t.Run("Test get on system/ntp node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"200": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp node ++++++++++++")

	/* ntp/ntp-keys/ntp-key/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/ntp-keys/ntp-key/state node ++++++++++++")
	url = "/openconfig-system:system/ntp/ntp-keys/ntp-key/state"
	t.Run("Test get on system/ntp/ntp-keys/ntp-key/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/ntp-keys/ntp-key/state node ++++++++++++")

	/* servers/server */
	// Pre required configs
	// Load mgmt vrf config
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load mgmt port
	pre_req_map = map[string]interface{}{"MGMT_PORT": map[string]interface{}{"eth0": map[string]interface{}{"NULL": "NULL"}, "eth1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load mgmt interface prefix
	pre_req_map = map[string]interface{}{"MGMT_INTERFACE": map[string]interface{}{"eth0|11.11.11.11/24": map[string]interface{}{"gwaddr": "11.11.11.1"}, "eth1|22.22.22.22/32": map[string]interface{}{"gwaddr": "22.22.22.1"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load VRF
	pre_req_map = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load Port
	pre_req_map = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet4": map[string]interface{}{"NULL": "NULL"}, "Ethernet8": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load Interface prefix
	pre_req_map = map[string]interface{}{"INTERFACE": map[string]interface{}{"Ethernet4": map[string]interface{}{"vrf_name": "Vrf1"}, "Ethernet4|33.33.33.33": map[string]interface{}{"NULL": "NULL"}, "Ethernet8|33.33.33.33": map[string]interface{}{"NULL": "NULL"}, "Ethernet8|44.44.44.44/24": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load NTP Global
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load NTP Key
	pre_req_map = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": map[string]interface{}{"type": "md5", "value": "key-value"}}}
	loadDB(db.ConfigDB, pre_req_map)

	t.Log("\n\n+++++++++++++ Performing POST(Create) on system/ntp/servers/server node  ++++++++++++")
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"default\", \"source-address\": \"44.44.44.44\", \"key-id\": 100}}]}"
	t.Run("Test POST(Create) on system/ntp/servers/server node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"association_type": "server", "version": "4", "iburst": "on", "key": "100"}}}
	t.Run("Verify POST(Create) on system/ntp/servers/server node", verifyDbResult(rclient, "NTP_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "default", "src_intf": "Ethernet8"}}}
	t.Run("Verify POST(Create) on system/ntp/server vrf-src_intf node", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing POST(Create) on system/ntp/servers/server node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/servers/server node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"association_type": "pool", "version": "3", "iburst": "on", "key": "100"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"src_intf": "eth1", "vrf": "mgmt"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:config\": { \"address\": \"2.2.2.2\", \"association-type\": \"POOL\", \"version\": 3, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"22.22.22.22\", \"key-id\": 100}}"
	url = "/openconfig-system:system/ntp/servers/server[address=2.2.2.2]/config"
	t.Run("Test get on system/ntp/servers/server node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	cleanuptbl = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"association_type": "pool", "version": "3", "iburst": "on", "key": "100"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"src_intf": "eth0", "vrf": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	cleanuptbl = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	expected_get_json = "{\"openconfig-system:config\": { \"address\": \"2.2.2.2\", \"association-type\": \"POOL\", \"version\": 3, \"iburst\": true, \"source-address\": \"11.11.11.11\", \"key-id\": 100}}"
	url = "/openconfig-system:system/ntp/servers/server[address=2.2.2.2]/config"
	t.Run("Test get on system/ntp/servers/server default vrf node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/servers/server node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/ntp node ++++++++++++")
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"src_intf": "eth0", "vrf": "mgmt"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:ntp\": {\"servers\": { \"server\": [{\"address\": \"2.2.2.2\", \"config\": { \"address\": \"2.2.2.2\", \"association-type\": \"POOL\", \"version\": 3, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"11.11.11.11\", \"key-id\": 100} } ]}, \"ntp-keys\": { \"ntp-key\": [{\"key-id\": 100, \"config\": { \"key-id\": 100, \"key-type\": \"openconfig-system:NTP_AUTH_MD5\", \"key-value\": \"key-value\"} } ]}, \"config\": { \"enabled\": true}}}"
	url = "/openconfig-system:system/ntp"
	t.Run("Test get on system/ntp node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp node ++++++++++++")

	/*
		t.Log("\n\n+++++++++++++ Test mgmt vrf when mgmt_vrf_config not enabled  ++++++++++++")
		cleanuptbl = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": ""}}
		unloadDB(db.ConfigDB, cleanuptbl)
		cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
		unloadDB(db.ConfigDB, cleanuptbl)
		url = "/openconfig-system:system/ntp/servers"
		url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"key-id\": 100}}]}"
		mgmt_vrf_err := errors.New("Must condition not satisfied. Try enable Management VRF.")
		t.Run("Test ntp/servers/server mgmt vrf when mgmt_vrf_config not enabled set", processSetRequest(url, url_body_json, "POST", true, mgmt_vrf_err))
		time.Sleep(1 * time.Second)
	*/

	t.Log("\n\n+++++++++++++ Test multiple vrf case  ++++++++++++")
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "default", "src_intf": "eth1"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"key-id\": 100}}]}"
	vrf_err := errors.New("Given network-instance name is different from already configured one for this/any other server")
	t.Run("Test ntp/servers/server multiple vrf case", processSetRequest(url, url_body_json, "POST", true, vrf_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Test multiple interface case  ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "mgmt", "src_intf": "eth0"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"22.22.22.22\", \"key-id\": 100}}]}"
	intf_err := errors.New("Given source address's port doesn't match with already configured src_intf")
	t.Run("Test ntp/servers/server multiple src_intf case", processSetRequest(url, url_body_json, "POST", true, intf_err))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Test NTP global table cleanup on last server deletion  ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"22.22.22.22\", \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server vrf_intf_cleanup server-1 case", processSetRequest(url, url_body_json, "POST", false, nil))
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"2.2.2.2\", \"config\": { \"address\": \"2.2.2.2\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"22.22.22.22\", \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server vrf_intf_cleanup server-2 case", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/ntp/servers/server[address=1.1.1.1]"
	t.Run("Test delete ntp/servers/server vrf_intf_cleanup server-1 case", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup server-1 case", verifyDbResult(rclient, "NTP_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "src_intf": "eth1", "vrf": "mgmt"}}}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup server-1 global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/ntp/servers/server[address=2.2.2.2]"
	t.Run("Test delete ntp/servers/server vrf_intf_cleanup server-2 case", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup server-2 case", verifyDbResult(rclient, "NTP_SERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup server-2 global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Test NTP global table cleanup on last server deletion - when only vrf is configured ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server vrf_intf_cleanup no-intf case", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "mgmt"}}}
	t.Run("Verify create on ntp/servers/server vrf_intf_cleanup no-intf global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/ntp/servers/server[address=1.1.1.1]"
	t.Run("Test delete ntp/servers/server vrf_intf_cleanup no-intf case", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-intf case", verifyDbResult(rclient, "NTP_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-intf global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Test NTP global table cleanup on last server deletion - when vrf & src_intf not configured ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server vrf_intf_cleanup no-vrf-no-intf case", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	t.Run("Verify create on ntp/servers/server vrf_intf_cleanup no-vrf-no-intf global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/ntp/servers/server[address=1.1.1.1]"
	t.Run("Test delete ntp/servers/server vrf_intf_cleanup no-vrf-no-intf case", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-vrf-no-intf case", verifyDbResult(rclient, "NTP_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-vrf-no-intf global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Test NTP global table cleanup on last server deletion - when NTP|global table is empty ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server vrf_intf_cleanup no-ntp-global case", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{}}}
	t.Run("Verify create on ntp/servers/server vrf_intf_cleanup no-ntp-global global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/ntp/servers/server[address=1.1.1.1]"
	t.Run("Test delete ntp/servers/server vrf_intf_cleanup no-ntp-global case", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-ntp-global case", verifyDbResult(rclient, "NTP_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{}}}
	t.Run("Verify delete on ntp/servers/server vrf_intf_cleanup no-ntp-global global case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)

	/*
		t.Log("\n\n+++++++++++++ Test NTP invalid source address - wrong syntax ++++++++++++")
		cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
		unloadDB(db.ConfigDB, cleanuptbl)
		pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
		loadDB(db.ConfigDB, pre_req_map)
		pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
		loadDB(db.ConfigDB, pre_req_map)
		url = "/openconfig-system:system/ntp/servers"
		url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"22.22.2.a\", \"key-id\": 100}}]}"
		wrong_ip_err := errors.New("Invalid input")
		t.Run("Test ntp/servers/server src_addr wrong syntax case", processSetRequest(url, url_body_json, "POST", true, wrong_ip_err))
	*/

	t.Log("\n\n+++++++++++++ Test NTP invalid source address - not configured on any port ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"192.1.2.3\", \"key-id\": 100}}]}"
	not_conf_ip_err := errors.New("Failed to get source interface for given source address")
	t.Run("Test ntp/servers/server src_addr not configured on vrf any case", processSetRequest(url, url_body_json, "POST", true, not_conf_ip_err))

	t.Log("\n\n+++++++++++++ Test NTP invalid source address - not configured on vrf port ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"network-instance\": \"mgmt\", \"source-address\": \"33.33.33.33\", \"key-id\": 100}}]}"
	not_conf_vrf_ip_err := errors.New("Failed to get source interface for given source address")
	t.Run("Test ntp/servers/server src_addr not configured on vrf port case", processSetRequest(url, url_body_json, "POST", true, not_conf_vrf_ip_err))

	t.Log("\n\n+++++++++++++ Test NTP server src_intf is fetched correctly based on vrf already configured ++++++++++++")
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	pre_req_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "default"}}}
	loadDB(db.ConfigDB, pre_req_map)
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)
	cleanuptbl = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	url = "/openconfig-system:system/ntp/servers"
	url_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.1.1.1\", \"config\": { \"address\": \"1.1.1.1\", \"association-type\": \"SERVER\", \"version\": 4, \"iburst\": true, \"source-address\": \"44.44.44.44\", \"key-id\": 100}}]}"
	t.Run("Test ntp/servers/server src_intf fetch case", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"NTP": map[string]interface{}{"global": map[string]interface{}{"admin_state": "enabled", "vrf": "default", "src_intf": "Ethernet8"}}}
	t.Run("Verify create on ntp/servers/server vrf_intf_cleanup src_intf fetch case", verifyDbResult(rclient, "NTP|global", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean interface prefix
	cleanuptbl = map[string]interface{}{"INTERFACE": map[string]interface{}{"Ethernet4": "", "Ethernet4|33.33.33.33": "", "Ethernet8|33.33.33.33": "", "Ethernet8|44.44.44.44/24": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt interface prefix
	cleanuptbl = map[string]interface{}{"MGMT_INTERFACE": map[string]interface{}{"eth0|11.11.11.11/24": "", "eth1|22.22.22.22/32": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean Port
	cleanuptbl = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet4": "", "Ethernet8": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt port
	cleanuptbl = map[string]interface{}{"MGMT_PORT": map[string]interface{}{"eth0": "", "eth1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean VRF
	cleanuptbl = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt vrf config
	cleanuptbl = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean NTP Keys
	cleanuptbl = map[string]interface{}{"NTP_KEY": map[string]interface{}{"100": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean NTP global
	cleanuptbl = map[string]interface{}{"NTP": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Test delete iburst
	t.Log("\n\n+++++++++++++ Performing delete on system/ntp/servers/server/config/iburst node ++++++++++++")
	pre_req_map = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"association_type": "pool", "version": "3", "iburst": "on"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/ntp/servers/server[address=2.2.2.2]/config/iburst"
	t.Run("Test delete on system/ntp/servers/server/config/iburst node", processDeleteRequest(url, false))
	expected_map = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"association_type": "pool", "version": "3", "iburst": "off"}}}
	t.Run("Verify delete on system/ntp/servers/server/config/iburst node", verifyDbResult(rclient, "NTP_SERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"NTP_SERVER": map[string]interface{}{"2.2.2.2": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/ntp/servers/server/config/iburst node ++++++++++++")

	/* ntp/servers/server/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/ntp/servers/server/state node ++++++++++++")
	url = "/openconfig-system:system/ntp/servers/server/state"
	t.Run("Test get on system/ntp/servers/server/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/ntp/servers/server/state node ++++++++++++")
}

func Test_oc_system_dns(t *testing.T) {
	var url, expected_get_json, url_body_json string
	var pre_req_map, pre_req_map_empty, cleanuptbl, expected_map map[string]interface{}

	t.Log("\n\n+++++++++++++ Performing unit tests on system/dns container nodes  ++++++++++++")

	/* dns/config/search */

	t.Log("\n\n+++++++++++++ Performing create on system/dns/config/search node  ++++++++++++")
	url = "/openconfig-system:system/dns/config"
	url_body_json = "{ \"openconfig-system:search\": [\"1.1.1.1\"]}"
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test create on system/dns/config/search node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on system/dns/config/search node", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing create on system/dns/config/search node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing create multiple nodes on system/dns/config/search node  ++++++++++++")
	url = "/openconfig-system:system/dns/config"
	url_body_json = "{ \"openconfig-system:search\": [\"1.1.1.1\", \"2.2.2.2\"]}"
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	t.Run("Test create multiple nodes on system/dns/config/search node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify create multiple nodes on system/dns/config/search node 1", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify create multiple nodes on system/dns/config/search node 2", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing create multiple nodes on system/dns/config/search node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing update on system/dns/config/search node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/dns/config/search"
	url_body_json = "{ \"openconfig-system:search\": [\"2.2.2.2\"]}"
	time.Sleep(1 * time.Second)
	t.Run("Test update on system/dns/config/search node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify update on system/dns/config/search node old", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify update on system/dns/config/search node new", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing update on system/dns/config/search node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/dns/config/search node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns/config/search"
	t.Run("Test delete on system/dns/config/search node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify delete on system/dns/config/search node", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/dns/config/search node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/dns/config/search node 3 entries ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}, "2.2.2.2": map[string]interface{}{"NULL": "NULL"}, "3.3.3.3": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns/config/search"
	t.Run("Test delete on system/dns/config/search node 3 entries", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify delete on system/dns/config/search node 3 entries", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	t.Run("Verify delete on system/dns/config/search node 3 entries", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	t.Run("Verify delete on system/dns/config/search node 3 entries", verifyDbResult(rclient, "DNS_NAMESERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": "", "3.3.3.3": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/dns/config/search node 3 entries ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/dns/config node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns/config"
	t.Run("Test delete on system/dns/config node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify delete on system/dns/config node", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/dns/config node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/dns node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns"
	t.Run("Test delete on system/dns node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify delete on system/dns node", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system/dns node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on dns - system node  ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system"
	t.Run("Test delete on system node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify delete on system node", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("\n\n+++++++++++++ Done Performing delete on system node  ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing DELETE on system/dns/config/search no DB entry case  ++++++++++++")
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns/config/search"
	t.Run("Test delete on system/dns/config/search no db entry case", processDeleteRequest(url, false))
	t.Log("\n\n+++++++++++++ Done Performing delete on system/dns/config/search  no DB entry case ++++++++++++")

	// Replace operation
	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 0to1 ++++++++++++")
	loadDB(db.ConfigDB, pre_req_map_empty)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/dns/config/search"
	url_body_json = "{\"openconfig-system:search\": [\"1.1.1.1\"]}"
	t.Run("Test Replace on system/dns/config/search node 0to1 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 0to1 ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 0to1 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 1to2 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"2.2.2.2\"]}"
	t.Run("Test Replace on system/dns/config/search node 1to2 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 1to2 del ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 1to2 new ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 1to2 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 2to1-2 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"1.1.1.1\", \"2.2.2.2\"]}"
	t.Run("Test Replace on system/dns/config/search node 2to1-2 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 2to1-2 new ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 2to1-2 unchanged ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 2to1-2 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 1-2to3-4 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"3.3.3.3\", \"4.4.4.4\"]}"
	t.Run("Test Replace on system/dns/config/search node 1-2to3-4 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 1-2to3-4 old-1 ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 1-2to3-4 old-2 ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 1-2to3-4 new-1 ", verifyDbResult(rclient, "DNS_NAMESERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 1-2to3-4 new-2 ", verifyDbResult(rclient, "DNS_NAMESERVER|4.4.4.4", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 1-2to3-4 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 3-4to4 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"4.4.4.4\"]}"
	t.Run("Test Replace on system/dns/config/search node 3-4to4 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 3-4to4 del ", verifyDbResult(rclient, "DNS_NAMESERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 3-4to4 unchanged ", verifyDbResult(rclient, "DNS_NAMESERVER|4.4.4.4", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 3-4to4 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 4to1-2-4 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"1.1.1.1\", \"2.2.2.2\", \"4.4.4.4\"]}"
	t.Run("Test Replace on system/dns/config/search node 4to1-2-4 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 4to1-2-4 new ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 4to1-2-4 new ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"4.4.4.4": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 4to1-2-4 unchanged ", verifyDbResult(rclient, "DNS_NAMESERVER|4.4.4.4", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 4to1-2-4 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node 1-2-4to3-2 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"3.3.3.3\", \"2.2.2.2\"]}"
	t.Run("Test Replace on system/dns/config/search node 1-2-4to3-2 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 1-2-4to3-2 del-1 ", verifyDbResult(rclient, "DNS_NAMESERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 1-2-4to3-2 unchanged ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node 1-2-4to3-2 new ", verifyDbResult(rclient, "DNS_NAMESERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{}}
	t.Run("Verify replace on system/dns/config/search node 1-2-4to3-2 del-2 ", verifyDbResult(rclient, "DNS_NAMESERVER|4.4.4.4", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node 1-2-4to3-2 ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Replace on system/dns/config/search node unchanged-2 ++++++++++++")
	url_body_json = "{\"openconfig-system:search\": [\"3.3.3.3\", \"2.2.2.2\"]}"
	t.Run("Test Replace on system/dns/config/search node unchanged-2 ", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node unchanged-2 unchanged ", verifyDbResult(rclient, "DNS_NAMESERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify replace on system/dns/config/search node unchanged-2 unchanged-2 ", verifyDbResult(rclient, "DNS_NAMESERVER|3.3.3.3", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Replace on system/dns/config/search node unchanged-2 ++++++++++++")

	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": "", "3.3.3.3": "", "4.4.4.4": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	// Get
	t.Log("\n\n+++++++++++++ Performing Get on system/dns/config/search node ++++++++++++")
	pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)
	expected_get_json = "{\"openconfig-system:search\":[\"1.1.1.1\"]}"
	url = "/openconfig-system:system/dns/config/search"
	t.Run("Test get on system/dns/config/search node", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/dns/config/search node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/dns/config node ++++++++++++")
	expected_get_json = "{\"openconfig-system:config\": {\"search\":[\"1.1.1.1\"]}}"
	url = "/openconfig-system:system/dns/config"
	t.Run("Test get on system/dns/config node", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/dns/config node ++++++++++++")

	t.Log("\n\n+++++++++++++ Performing Get on system/dns node ++++++++++++")
	expected_get_json = "{\"openconfig-system:dns\": {\"config\": {\"search\":[\"1.1.1.1\"]}}}"
	url = "/openconfig-system:system/dns"
	t.Run("Test get on system/dns node", processGetRequest(url, nil, expected_get_json, false, nil))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/dns node ++++++++++++")

	/*
		// boot-time & current-datetime also recvd
		t.Log("\n\n+++++++++++++ Performing Get on system node ++++++++++++")
		expected_get_json = "{\"openconfig-system:system\": {\"dns\": {\"config\": {\"search\":[\"1.1.1.1\"]}}}}"
		url = "/openconfig-system:system"
		t.Run("Test get on system node", processGetRequest(url, nil, expected_get_json, false, nil))
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing Get on system node ++++++++++++")

		// Order is changing everytime
		t.Log("\n\n+++++++++++++ Performing Get on system/dns/config/search multiple entries ++++++++++++")
		pre_req_map = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"NULL": "NULL"}, "3.3.3.3": map[string]interface{}{"NULL": "NULL"}}}
		loadDB(db.ConfigDB, pre_req_map)
		expected_get_json = "{\"openconfig-system:search\":[\"1.1.1.1\", \"3.3.3.3\", \"2.2.2.2\"]}"
		url = "/openconfig-system:system/dns/config/search"
		t.Run("Test get on system/dns/config/search multiple entries", processGetRequest(url, nil, expected_get_json, false, nil))
		time.Sleep(1 * time.Second)
		t.Log("\n\n+++++++++++++ Done Performing Get on system/dns/config/search multiple entries ++++++++++++")
	*/

	cleanuptbl = map[string]interface{}{"DNS_NAMESERVER": map[string]interface{}{"1.1.1.1": "", "2.2.2.2": "", "3.3.3.3": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	/* dns/state */
	t.Log("\n\n+++++++++++++ Performing Get on system/dns/state node ++++++++++++")
	url = "/openconfig-system:system/dns/state"
	t.Run("Test get on system/dns/state node", processGetRequest(url, nil, "", true, not_implemented_err))
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Done Performing Get on system/dns/state node ++++++++++++")

	t.Log("\n\n+++++++++++++ Done Performing unit tests on system/dns container nodes  ++++++++++++")
}

func Test_openconfig_system_aaa_authentication(t *testing.T) {
	var url string
	var url_body_json string
	var cleanuptbl map[string]interface{}
	var expected_map map[string]interface{}
	var pre_req_map map[string]interface{}

	t.Log("++++++++++++++ Starting Test: Authentication Method SET Transformations ++++++++++++++")
	t.Log("\n\n+++++++++++++ Performing CREATE on authentication-method node ++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap"}}}
	t.Run("Test create on authentication-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE Authentication Method Transformations ++++++++++++++")

	t.Log("++++++++++++++ Start Get Authentication Method Transformations ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	expected_get_json := "{\"openconfig-system:authentication-method\":[\"openconfig-aaa-types:TACACS_ALL\"]}"

	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authentication-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	t.Log("++++++++++++++ Start Get Authentication Method Transformations ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:authentication-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\"]}"

	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authentication-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	t.Log("++++++++++++++ Start Get Authentication Method Transformations ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,radius,local,default,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:authentication-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\",\"openconfig-aaa-types:LOCAL\",\"default\",\"ldap\"]}"

	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authentication-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	t.Log("++++++++++++++ Start Get Authentication Method Transformations ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,radius,local,invalid"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "invalid"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}

	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	t.Log("++++++++++++++ Start Get Authentication Method Transformations ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "invalid,tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	t.Log("++++++++++++++ Starting Test: Authentication performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authentication-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local"}}}
	t.Run("Test replace on authentication-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node with one same node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"TACACS_ALL\", \"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,local"}}}
	t.Run("Test replace on authentication-method node with one same node", func(t *testing.T) {
		processSetRequest(url, url_body_json, "PUT", false, nil)
		time.Sleep(1 * time.Second)
		verifyDbResult(rclient, "AAA|authentication", expected_map, false)
	})

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node with multiple values ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\", \"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local,default"}}}
	t.Run("Test replace on authentication-method node with multiple values", func(t *testing.T) {
		processSetRequest(url, url_body_json, "PUT", false, nil)
		time.Sleep(1 * time.Second)
		verifyDbResult(rclient, "AAA|authentication", expected_map, false)
	})
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node with empty db ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\", \"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local,default"}}}
	t.Run("Test replace on authentication-method node with multiple values", func(t *testing.T) {
		processSetRequest(url, url_body_json, "PUT", false, nil)
		time.Sleep(1 * time.Second)
		verifyDbResult(rclient, "AAA|authentication", expected_map, false)
	})

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method method type to db testt ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local"}}}
	t.Run("Test replace on authentication-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method default string and invalid payload ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node with default string and invalid payload ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"invalid\"]}"
	t.Run("Test replace on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method default string and invalid and valid payload ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authentication-method node with multiple values ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"default\",\"invalid\"]}"
	t.Run("Test replace on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for MULTI CREATE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 1 . Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"ldap\", \"RADIUS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local,ldap,radius"}}}
	t.Run("Test MULTI CREATE on authentication-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 2. Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\", \"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default,radius,local,ldap"}}}
	t.Run("Test MULTI CREATE on authentication-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius,local,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 3 .Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\",\"RADIUS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius,local,ldap"}}}
	t.Run("Test MULTI CREATE on authentication-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ 4. Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius,local,default"}}}
	t.Run("Test MULTI CREATE on authentication-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 5 .Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"invalid\"]}"
	t.Run("Test MULTI CREATE on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 6. Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"default\",\"invalid\"]}"
	t.Run("Test MULTI CREATE on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "radius,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 7. Performing MULTI CREATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"RADIUS_ALL\",\"LOCAL\",\"invalid\",\"ldap\"]}"
	t.Run("Test MULTI CREATE on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for MULTI CREATE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for UPDATE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing UPDATE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap,local"}}}
	t.Run("Test update on authentication-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))

	// Test Case 2: UPDATE with multiple values when no initial values are present
	t.Log("\n\n+++++++++++++ Performing UPDATE on authentication-method node with no initial values ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"LOCAL\", \"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "local,default"}}}
	t.Run("Test update on authentication-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify update on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on authentication-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"invalid\"]}"
	t.Run("Test replace on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on authentication-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"default\",\"invalid\"]}"
	t.Run("Test replace on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on authentication-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	url_body_json = "{\"openconfig-system:authentication-method\": [\"TACACS_ALL\",\"RADIUS_ALL\",\"invalid\",\"local\"]}"
	t.Run("Test replace on authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for UPDATE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for DELETE oper with tacacs+++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authentication-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "tacacs+,radius,default,ldap,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authentication-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": "ldap,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authentication-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authentication Method for DELETE oper with empty db ++++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authentication-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authentication-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authentication-method node", verifyDbResult(rclient, "AAA|authentication", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authentication Method Transformations for DELETE ++++++++++++++")

	t.Log("\n\n+++++++++++++Testing invalid string on authentication-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authentication": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authentication/config/authentication-method"
	exp_json := "{\"openconfig-system:authentication-method\": [\"invalidMethod\"]}"

	t.Run("Test invalid authentication-method node", func(t *testing.T) {
		err := processSetRequest(url, exp_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Strings apart from local,radius,ldap,default and tacacs+ are not allowed: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing invalid accounting Method Transformations for UPDATE ++++++++++++++")

}

func Test_openconfig_system_aaa_authorization(t *testing.T) {
	var url string
	var url_body_json string
	var cleanuptbl map[string]interface{}
	var expected_map map[string]interface{}
	var pre_req_map map[string]interface{}

	t.Log("++++++++++++++ Starting Test: authorization Method SET Transformations ++++++++++++++")
	t.Log("\n\n+++++++++++++ Performing CREATE on authorization-method node ++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "ldap"}}}
	t.Run("Test create on authorization-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE authorization Method Transformations ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	expected_get_json := "{\"openconfig-system:authorization-method\":[\"openconfig-aaa-types:TACACS_ALL\"]}"

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authorization-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:authorization-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\"]}"

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authorization-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius,local,default,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:authorization-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\",\"openconfig-aaa-types:LOCAL\",\"default\",\"ldap\"]}"

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on authorization-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "invalid"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs,radius,invalid,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")
	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "invalid,tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}

	t.Log("++++++++++++++ Starting Test: authorization performing GET ++++++++++++++")
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	t.Run("Test get on authentication-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on authorization-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: authorization Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"TACACS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+"}}}
	t.Run("Test replace on authorization-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method method type to db testt ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "local"}}}
	t.Run("Test replace on authorization-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method default string and invalid payload ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method node with default string and invalid payload ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"invalid\"]}"
	t.Run("Test replace on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method default string and invalid and valid payload ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing REPLACE on authorization-method node with multiple values ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"default\",\"invalid\"]}"
	t.Run("Test replace on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Done Performing Authorization Method Transformations for REPLACE++++++++++++++")

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius,local,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 3 .Performing MULTI CREATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"LOCAL\",\"RADIUS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius,local,ldap"}}}
	t.Run("Test MULTI CREATE on authorization-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "radius,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 4. Performing MULTI CREATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "radius,local,default"}}}
	t.Run("Test MULTI CREATE on authorization-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify MULTI CREATE on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 5 .Performing MULTI CREATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"invalid\"]}"
	t.Run("Test MULTI CREATE on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 5 .Performing MULTI CREATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"default\",\"invalid\"]}"
	t.Run("Test MULTI CREATE on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ 5 .Performing MULTI CREATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"TACACS_ALL\",\"RADIUS_ALL\",\"default\",\"invalid\"]}"
	t.Run("Test MULTI CREATE on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Starting Test: Authorization Method for UPDATE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing UPDATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "ldap,local"}}}
	t.Run("Test update on authorization-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))

	t.Log("++++++++++++++ Performing Authorization Method for UPDATE oper with identityref ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing UPDATE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"RADIUS_ALL\",\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "radius,local"}}}
	t.Run("Test update on authorization-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))

	// Test Case 2: UPDATE with multiple values when no initial values are present
	t.Log("\n\n+++++++++++++ Performing UPDATE on authorization-method node with no initial values ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"LOCAL\", \"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "local,default"}}}
	t.Run("Test update on authorization-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify update on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Authorization Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on authorization-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"invalid\"]}"
	t.Run("Test replace on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on authorization-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	url_body_json = "{\"openconfig-system:authorization-method\": [\"default\",\"invalid\"]}"
	t.Run("Test replace on authorization-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authorization method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authorization method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for UPDATE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authorization Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "ldap,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authorization-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authorization Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authorization-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authorization Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": "tacacs+,radius,ldap,local,default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authorization-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Authorization Method for DELETE oper with empty db ++++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on authorization-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/authorization/config/authorization-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"authorization": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on authorization-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on authorization-method node", verifyDbResult(rclient, "AAA|authorization", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing authorization Method Transformations for DELETE ++++++++++++++")
}

func Test_openconfig_system_aaa_accounting(t *testing.T) {
	var url string
	var url_body_json string
	var cleanuptbl map[string]interface{}
	var expected_map map[string]interface{}
	var pre_req_map map[string]interface{}

	t.Log("++++++++++++++ Starting Test: accounting Method SET Transformations ++++++++++++++")
	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "ldap"}}}
	t.Run("Test create on accounting-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"TACACS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "ldap,tacacs+"}}}
	t.Run("Test create on accounting-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"TACACS_ALL\",\"RADIUS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius"}}}
	t.Run("Test create on accounting-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"LOCAL\",\"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,local,default"}}}
	t.Run("Test create on accounting-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"TACACS_ALL\",\"RADIUS_ALL\",\"LOCAL\",\"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local,ldap,tacacs+,radius"}}}
	t.Run("Test create on accounting-method node", processSetRequest(url, url_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify create on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"invalid\"]}"
	t.Run("Test MULTI CREATE on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"invalid\"]}"
	t.Run("Test MULTI CREATE on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid authentication method, but got none")
		} else {
			t.Logf("Correctly received error for invalid authentication method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing CREATE on accounting-method node ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"default\",\"invalid\"]}"
	t.Run("Test MULTI CREATE on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "POST", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing CREATE accounting Method Transformations ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	expected_get_json := "{\"openconfig-system:accounting-method\":[\"openconfig-aaa-types:TACACS_ALL\"]}"
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on accounting-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on accounting-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:accounting-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\"]}"
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on accounting-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on accounting-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,local,default,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	expected_get_json = "{\"openconfig-system:accounting-method\":[\"openconfig-aaa-types:TACACS_ALL\",\"openconfig-aaa-types:RADIUS_ALL\",\"openconfig-aaa-types:LOCAL\",\"default\",\"ldap\"]}"
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	t.Logf("Performing GET request on URL: %s", url)
	t.Run("Test get on accounting-method node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on accounting-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "invalid"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	t.Run("Test get on accounting-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on accounting-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting performing GET ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,invalid,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	t.Run("Test get on accounting-method node", func(t *testing.T) {
		// Instead of expecting a valid response, we expect an error
		err := processGetRequest(url, nil, "", false) // Assuming that an error is returned when invalid
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	t.Log("++++++++++++++ Done Performing Get on accounting-method node ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"TACACS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+"}}}
	t.Run("Test replace on accounting-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Accounting Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"LOCAL\",\"ldap\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local,ldap"}}}
	t.Run("Test replace on accounting-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Accounting Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,local,ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"LOCAL\",\"TACACS_ALL\",\"RADIUS_ALL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local,tacacs+,radius"}}}
	t.Run("Test replace on accounting-method node", processSetRequest(url, url_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify replace on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Accounting Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"invalid\"]}"
	t.Run("Test replace on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Accounting Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: accounting Method for REPLACE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing REPLACE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"invalid\",\"default\"]}"
	t.Run("Test replace on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PUT", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing Accounting Method Transformations for REPLACE++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Accounting Method for UPDATE oper++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "ldap"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing UPDATE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"LOCAL\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "ldap,local"}}}
	t.Run("Test update on accounting-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))

	// Test Case 2: UPDATE with multiple values when no initial values are present
	t.Log("\n\n+++++++++++++ Performing UPDATE on accounting-method node with no initial values ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"LOCAL\", \"default\"]}"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "local,default"}}}
	t.Run("Test update on accounting-method node", processSetRequest(url, url_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify update on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting  Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on accounting-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"invalid\"]}"
	t.Run("Test replace on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for UPDATE ++++++++++++++")

	t.Log("\n\n+++++++++++++ Performing UPDATE on accounting-method node with default string ++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	url_body_json = "{\"openconfig-system:accounting-method\": [\"default\",\"invalid\"]}"
	t.Run("Test replace on accounting-method node", func(t *testing.T) {
		err := processSetRequest(url, url_body_json, "PATCH", false, nil)
		if err == nil {
			t.Error("Expected an error for invalid accounting method, but got none")
		} else {
			t.Logf("Correctly received error for invalid accounting method: %v", err)
		}
	})
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for UPDATE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Accounting Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "ldap,local"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on accounting-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Accounting Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on accounting-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Accounting Method for DELETE oper ++++++++++++++")
	pre_req_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": "tacacs+,radius,ldap,local,default"}}}
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on accounting-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for DELETE ++++++++++++++")

	t.Log("++++++++++++++ Starting Test: Accounting Method for DELETE oper when db is null++++++++++++++")
	cleanuptbl = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"login": ""}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++++++++ Performing DELETE on accounting-method node ++++++++++++")
	url = "/openconfig-system:system/aaa/accounting/config/accounting-method"
	expected_map = map[string]interface{}{"AAA": map[string]interface{}{"accounting": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Test delete on accounting-method node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify delete on accounting-method node", verifyDbResult(rclient, "AAA|accounting", expected_map, false))

	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done Performing accounting Method Transformations for DELETE ++++++++++++++")
}

func Test_Aaa_ServerGroupFunctions(t *testing.T) {
	t.Log("++++++++++++++ POST (CREATE)operation for Server Group tacacs Functions ++++++++++++++")
	t.Log("+++++POST for XPATH: /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config ++++++++++")
	pre_req_map := map[string]interface{}{}
	cleanuptbl := map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	create_url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	create_body_json := "{\"openconfig-aaa-tacacs:secret-key\":\"bngss\", \"openconfig-aaa-tacacs:port\":22}"
	expected_map := map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 22}}}

	t.Run("Test POST(CREATE) on server-group node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("Verify POST (CREATE) on server-group tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done POST operation on Server Group tacacs Functions ++++++++++++++")

	t.Log("++++++++++++++ GET operation for Server Group tacacs Functions ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 22}}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	expected_get_json := "{\"openconfig-system:config\":{\"secret-key\":\"bngss\", \"port\":22}}"

	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done GET operation on Server Group tacacs Functions ++++++++++++++")

	t.Log("++++++++++++++ PATCH(Update) operation for Server Group tacacs Functions ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 22}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	patch_url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	patch_body_json := "{\"openconfig-system:config\": {\"openconfig-aaa-tacacs:secret-key\":\"bxb2345\", \"openconfig-aaa-tacacs:port\":34}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bxb2345", "tcp_port": 34}}}

	t.Run("Test PATCH (UPDATE) on server-group node", processSetRequest(patch_url, patch_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify update (PATCH) on server-group tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map, false))
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PATCH (Update) operation on Server Group tacacs Functions ++++++++++++++")
	t.Log("++++++++++++++ PUT(Replace) operation for Server Group tacacs Functions ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 22}}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	put_url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	put_body_json := "{\"openconfig-system:config\": {\"openconfig-aaa-tacacs:secret-key\":\"bxb2345\", \"openconfig-aaa-tacacs:port\":34}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bxb2345", "tcp_port": 34}}}

	t.Run("Test PUT (REPLACE) on server-group node", processSetRequest(put_url, put_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify update (PUT) on server-group tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map, false))
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PUT (replace) operation on Server Group  tacacs Functions ++++++++++++++")

	/* ------------------------------------------------------------------------------------------------- */
	// Pre required configs
	// Load mgmt vrf config
	pre_req_map = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": map[string]interface{}{"mgmtVrfEnabled": "true"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load mgmt port
	pre_req_map = map[string]interface{}{"MGMT_PORT": map[string]interface{}{"eth0": map[string]interface{}{"NULL": "NULL"}, "eth1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load mgmt interface prefix
	pre_req_map = map[string]interface{}{"MGMT_INTERFACE": map[string]interface{}{"eth0|11.11.11.11/24": map[string]interface{}{"gwaddr": "11.11.11.1"}, "eth1|22.22.22.22/32": map[string]interface{}{"gwaddr": "22.22.22.1"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load VRF
	pre_req_map = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load Port
	pre_req_map = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet4": map[string]interface{}{"NULL": "NULL"}, "Ethernet8": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	// Load Interface prefix
	pre_req_map = map[string]interface{}{"INTERFACE": map[string]interface{}{"Ethernet4": map[string]interface{}{"vrf_name": "Vrf1"}, "Ethernet4|33.33.33.33": map[string]interface{}{"NULL": "NULL"}, "Ethernet8|33.33.33.33": map[string]interface{}{"NULL": "NULL"}, "Ethernet8|44.44.44.44/24": map[string]interface{}{"NULL": "NULL"}}}
	loadDB(db.ConfigDB, pre_req_map)

	t.Log("++++++++++++++ POST (CREATE)operation for Server Group tacacs source-address Functions ++++++++++++++")
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""}}
	cleanuptbl2 := map[string]interface{}{"TACPLUS": map[string]interface{}{"global": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	create_body_json = "{\"openconfig-aaa-tacacs:source-address\":\"11.11.11.11\", \"openconfig-aaa-tacacs:port\":22}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"vrf": "mgmt", "tcp_port": 22}}}

	t.Run("Test POST(CREATE) on server-group node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("Verify POST (CREATE) on server-group tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map, false))
	time.Sleep(1 * time.Second)
	expected_map2 := map[string]interface{}{"TACPLUS": map[string]interface{}{"global": map[string]interface{}{"src_intf": "eth0"}}}
	time.Sleep(1 * time.Second)
	t.Run("Verify POST (CREATE) on server-group tacacs node", verifyDbResult(rclient, "TACPLUS|global", expected_map2, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	unloadDB(db.ConfigDB, cleanuptbl2)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Done POST operation on Server Group tacacs Functions ++++++++++++++")

	t.Log("++++++++++++++ GET operation for Server Group tacacs source-addressFunctions ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"vrf": "mgmt", "tcp_port": 22}}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	pre_req_map2 := map[string]interface{}{"TACPLUS": map[string]interface{}{"global": map[string]interface{}{"src_intf": "eth0"}}}

	loadDB(db.ConfigDB, pre_req_map2)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.1.1.1]/tacacs/config"
	expected_get_json = "{\"openconfig-system:config\":{\"source-address\":\"11.11.11.11\", \"port\":22}}"

	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ GET operation for Server Groups root level ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups"
	expected_get_json = "{\"openconfig-system:server-groups\":{\"server-group\":[{\"name\":\"tacacs+\",\"config\":{\"name\":\"tacacs+\"}, \"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\"}, \"tacacs\":{\"config\":{\"port\":22, \"source-address\": \"11.11.11.11\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}}"

	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	t.Log("++++++++++++++ GET operation for Server Group servers root level ++++++++++++++")
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	expected_get_json = "{\"openconfig-system:servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\"}, \"tacacs\":{\"config\":{\"port\":22, \"source-address\": \"11.11.11.11\"}}}]}}"
	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ GET operation for Server Group key level ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	expected_get_json = "{\"openconfig-system:server-group\": [{\"name\": \"tacacs+\",\"config\":{\"name\":\"tacacs+\"}, \"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\"}, \"tacacs\":{\"config\":{\"port\":22, \"source-address\": \"11.11.11.11\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}"
	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)

	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	cleanuptbl2 = map[string]interface{}{"TACPLUS": map[string]interface{}{"global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl2)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done GET operation on Server Group tacacs  source-address Functions ++++++++++++++")

	t.Log("++++++++++++++ POST (CREATE)operation for Server Group radius source-address Functions ++++++++++++++")
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=2.2.2.2]/radius/config"
	create_body_json = "{\"openconfig-aaa:source-address\":\"11.11.11.11\", \"openconfig-aaa:auth-port\":1812}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"auth_port": 1812, "vrf": "mgmt", "src_intf": "eth0"}}}

	t.Run("Test POST(CREATE) on server-group node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("Verify POST (CREATE) on server-group radius address node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Done POST operation on Server Group radius Functions ++++++++++++++")
	t.Log("++++++++++++++ GET operation for Server Group radius source-addressFunctions ++++++++++++++")
	pre_req_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"vrf": "mgmt", "src_intf": "eth0"}}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=2.2.2.2]/radius/config"
	expected_get_json = "{\"openconfig-system:config\":{\"source-address\":\"11.11.11.11\"}}"

	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done GET operation on Server Group radius  source-address Functions ++++++++++++++")
	// Clean interface prefix
	cleanuptbl = map[string]interface{}{"INTERFACE": map[string]interface{}{"Ethernet4": "", "Ethernet4|33.33.33.33": "", "Ethernet8|33.33.33.33": "", "Ethernet8|44.44.44.44/24": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt interface prefix
	cleanuptbl = map[string]interface{}{"MGMT_INTERFACE": map[string]interface{}{"eth0|11.11.11.11/24": "", "eth1|22.22.22.22/32": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean Port
	cleanuptbl = map[string]interface{}{"PORT": map[string]interface{}{"Ethernet4": "", "Ethernet8": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt port
	cleanuptbl = map[string]interface{}{"MGMT_PORT": map[string]interface{}{"eth0": "", "eth1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean VRF
	cleanuptbl = map[string]interface{}{"VRF": map[string]interface{}{"Vrf1": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

	// Clean mgmt vrf config
	cleanuptbl = map[string]interface{}{"MGMT_VRF_CONFIG": map[string]interface{}{"vrf_global": ""}}
	unloadDB(db.ConfigDB, cleanuptbl)

}

func Test_Aaa_ServerGroupFunctions_at_root_levels(t *testing.T) {
	/* POST /openconfig-system:system/aaa/server-groups
	 * PUT /openconfig-system:system/aaa/server-groups
	 * PATCH /openconfig-system:system/aaa/server-groups
	 * GET /openconfig-system:system/aaa/server-groups
	 * DELETE /openconfig-system:system/aaa/server-groups
	 */

	t.Log("++++++++++++++ Performing POST (CREATE) operation for Server Group address /server-groups++++++++++++++")
	pre_req_map := map[string]interface{}{}
	cleanuptbl := map[string]interface{}{
		"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""},
		"RADIUS_SERVER":  map[string]interface{}{"2.2.2.2": ""}, // Clean up for RADIUS server
	}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	create_url1 := "/openconfig-system:system/aaa/server-groups"
	create_body_json1 := "{\"openconfig-system:server-group\":[{\"name\":\"tacacs+\",\"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\"}}}]}},{\"name\":\"radius\",\"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\"}}}]}}]}"
	t.Run("Test POST(CREATE) on server-groups node", processSetRequest(create_url1, create_body_json1, "POST", false, nil))
	time.Sleep(1 * time.Second)
	expected_map1 := map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 49}}}
	expected_map2 := map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"passkey": "radiuskey", "auth_port": 1812}}}
	t.Run("Verify POST (CREATE) on server-group tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST (CREATE) on server-group radius node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map2, false))
	time.Sleep(1 * time.Second)
	//	t.Log("Unloading data from the database...")
	//	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done POST operation on Server Group address recent server-groups ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (REPLACE) operation for Server Group address /server-groups++++++++++++++")
	create_url1 = "/openconfig-system:system/aaa/server-groups"
	create_body_json1 = "{\"openconfig-system:server-groups\": {\"openconfig-system:server-group\":[{\"name\":\"tacacs+\",\"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"sonicss\"}}}]}},{\"name\":\"radius\",\"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey_changed\"}}}]}}]}}"
	t.Run("Test PUT(REPLACE) on server-groups node", processSetRequest(create_url1, create_body_json1, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	expected_map1 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "sonicss", "tcp_port": 49}}}
	expected_map2 = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"passkey": "radiuskey_changed", "auth_port": 1812}}}
	t.Run("Verify PUT (REPLACE) on server-group tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT (REPLACE) on server-group radius node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map2, false))
	time.Sleep(1 * time.Second)
	//	t.Log("Unloading data from the database...")
	//	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on Server Group address recent server-groups ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (UPDATE) operation for Server Group address /server-groups++++++++++++++")
	create_url1 = "/openconfig-system:system/aaa/server-groups"
	create_body_json1 = "{\"openconfig-system:server-groups\": {\"openconfig-system:server-group\":[{\"name\":\"tacacs+\",\"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"port\":50}}}]}},{\"name\":\"radius\",\"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"auth-port\":1815}}}]}}]}}"
	t.Run("Test PATCH(UPDATE) on server-groups node", processSetRequest(create_url1, create_body_json1, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	expected_map1 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "sonicss", "tcp_port": 50}}}
	expected_map2 = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"passkey": "radiuskey_changed", "auth_port": 1815}}}
	t.Run("Verify PATCH (REPLACE) on server-group tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH (REPLACE) on server-group radius node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map2, false))
	time.Sleep(1 * time.Second)
	//	t.Log("Unloading data from the database...")
	//	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PATCH operation on Server Group address recent server-groups ++++++++++++++")

	/* GET Operation on /openconfig-system:system/aaa/server-groups */
	t.Log("++++++++++++++ Performing GET operation for Server Groups root level ++++++++++++++")
	url := "/openconfig-system:system/aaa/server-groups"
	expected_get_json := "{\"openconfig-system:server-groups\":{\"server-group\":[{\"name\":\"radius\",\"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey_changed\", \"auth-port\":1815}}}]},\"state\":{\"name\":\"radius\"}}, {\"name\":\"tacacs+\",\"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\"}, \"tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"sonicss\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}}"

	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))

	/* DELETE Operation on /openconfig-system:system/aaa/server-groups */
	t.Log("++++++++++++++ DELETE  operation for Server Groups root level ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups"
	t.Log(" /openconfig-system:system/aaa/server-groups Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del := map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map_del, false))
	expected_map_del2 := map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map_del2, false))
	time.Sleep(1 * time.Second)

	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)

	/* POST /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]
	 * POST /openconfig-system:system/aaa/server-groups/server-group[name=radius]
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=radius]
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=radius]
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=radius]
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=radius]
	 */

	t.Log("++++++++++++++ Performing POST (CREATE) operation for TACACS+ Server Groups with Multiple Servers ++++++++++++++")
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": "", "1.1.1.2": ""}, "RADIUS_SERVER": map[string]interface{}{"2.2.2.2": "", "2.2.2.3": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	tacacs_url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	tacacs_post_body_json := "{\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\"}}},{\"address\":\"1.1.1.2\",\"config\":{\"address\":\"1.1.1.2\"},\"tacacs\":{\"config\":{\"secret-key\":\"tacacssecret\"}}}]}}"
	expected_map1 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 49}}}
	expected_map2 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{"passkey": "tacacssecret", "tcp_port": 49}}}
	t.Run("Test POST(CREATE) on TACACS+ server-group with multiple servers", processSetRequest(tacacs_url, tacacs_post_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST (CREATE) on server-group tacacs+ node at system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST (CREATE) on server-group tacacs+ node system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.2", expected_map2, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing POST (CREATE) operation for RADIUS Server Groups with Multiple Servers ++++++++++++++")
	radius_url := "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	radius_post_body_json := "{\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\"}}},{\"address\":\"2.2.2.3\",\"config\":{\"address\":\"2.2.2.3\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\"}}}]}}"
	expected_map3 := map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"auth_port": 1812, "passkey": "radiuskey"}}}
	t.Run("Test POST(CREATE) on RADIUS server-group with multiple servers at system/aaa/server-groups/server-group[name=radius]", processSetRequest(radius_url, radius_post_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify POST (CREATE) on server-group radius node at at system/aaa/server-groups/server-group[name=radius]", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map3, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done POST operation on TACACS+ and RADIUS Server Groups with Multiple Servers  /server-groups/server-group++++++++++++++")

	t.Log("++++++++++++++ Perfroming PUT (REPLACE) operation for TACACS+ Server Groups with Multiple Servers ++++++++++++++")
	tacacs_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	tacacs_post_body_json = "{\"openconfig-system:server-group\": [{\"name\":\"tacacs+\", \"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"bngss2\", \"port\":50}}},{\"address\":\"1.1.1.2\",\"config\":{\"address\":\"1.1.1.2\"},\"tacacs\":{\"config\":{\"secret-key\":\"tacacssecret2\", \"port\":50}}}]}}]}"
	expected_map1 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss2", "tcp_port": 50}}}
	expected_map2 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{"passkey": "tacacssecret2", "tcp_port": 50}}}
	t.Run("Test PUT(REPLACE) on TACACS+ server-group with multiple servers", processSetRequest(tacacs_url, tacacs_post_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT (REPLACE) on server-group tacacs+ node at system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT (REPLACE) on server-group tacacs+ node system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.2", expected_map2, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing PUT (REPLACE) operation for RADIUS Server Groups with Multiple Servers ++++++++++++++")
	radius_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	radius_put_body_json := "{\"openconfig-system:server-group\":[{\"name\":\"radius\", \"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey2\", \"auth-port\":1921}}},{\"address\":\"2.2.2.3\",\"config\":{\"address\":\"2.2.2.3\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey2\",\"auth-port\":1852}}}]}}]}"
	expected_map3 = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"auth_port": 1921, "passkey": "radiuskey2"}}}
	expected_map4 := map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.3": map[string]interface{}{"auth_port": 1852, "passkey": "radiuskey2"}}}
	t.Run("Test PUT(REPLACE) on RADIUS server-group with multiple servers at system/aaa/server-groups/server-group[name=radius]", processSetRequest(radius_url, radius_put_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT (REPLACE) on server-group radius node at at system/aaa/server-groups/server-group[name=radius]", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map3, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PUT (REPLACE) on server-group radius node at at system/aaa/server-groups/server-group[name=radius]", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.3", expected_map4, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on TACACS+ and RADIUS Server Groups with Multiple Servers  /server-groups/server-group++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (UPDATE) operation for TACACS+ Server Groups with Multiple Servers ++++++++++++++")
	tacacs_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	tacacs_post_body_json = "{\"openconfig-system:server-group\": [{\"name\":\"tacacs+\", \"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\", \"port\":50}}},{\"address\":\"1.1.1.2\",\"config\":{\"address\":\"1.1.1.2\"},\"tacacs\":{\"config\":{\"secret-key\":\"tacacssecret\", \"port\":50}}}]}}]}"
	expected_map1 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 50}}}
	expected_map2 = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{"passkey": "tacacssecret", "tcp_port": 50}}}
	t.Run("Test PUT(REPLACE) on TACACS+ server-group with multiple servers", processSetRequest(tacacs_url, tacacs_post_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH (UPDATE) on server-group tacacs+ node at system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map1, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH (UPDATE) on server-group tacacs+ node system/aaa/server-groups/server-group[name=tacacs+]", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.2", expected_map2, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing PATCH (UPDATE) operation for RADIUS Server Groups with Multiple Servers ++++++++++++++")
	radius_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	radius_patch_body_json := "{\"openconfig-system:server-group\":[{\"name\":\"radius\", \"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\", \"auth-port\":1921}}},{\"address\":\"2.2.2.3\",\"config\":{\"address\":\"2.2.2.3\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\",\"auth-port\":1852}}}]}}]}"
	expected_map3 = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{"auth_port": 1921, "passkey": "radiuskey"}}}
	expected_map4 = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.3": map[string]interface{}{"auth_port": 1852, "passkey": "radiuskey"}}}
	t.Run("Test PATCH(UPDATE) on RADIUS server-group with multiple servers at system/aaa/server-groups/server-group[name=radius]", processSetRequest(radius_url, radius_patch_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH (UPDATE) on server-group radius node at at system/aaa/server-groups/server-group[name=radius]", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map3, false))
	time.Sleep(1 * time.Second)
	t.Run("Verify PATCH (UPDATE) on server-group radius node at at system/aaa/server-groups/server-group[name=radius]", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.3", expected_map4, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PATCH operation on TACACS+ and RADIUS Server Groups with Multiple Servers  /server-groups/server-group++++++++++++++")

	/* GET Operation on /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+] */
	t.Log("++++++++++++++ Performing GET operation for TACACS+ Server Groups with Multiple Servers ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	expected_get_json = "{\"openconfig-system:server-group\": [{\"name\":\"tacacs+\", \"config\":{\"name\":\"tacacs+\"},\"servers\":{\"server\":[{\"address\":\"1.1.1.1\",\"config\":{\"address\":\"1.1.1.1\"},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\", \"port\":50}}},{\"address\":\"1.1.1.2\",\"config\":{\"address\":\"1.1.1.2\"},\"tacacs\":{\"config\":{\"secret-key\":\"tacacssecret\", \"port\":50}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}"
	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))

	/* GET Operation on /openconfig-system:system/aaa/server-groups/server-group[name=radius] */
	t.Log("++++++++++++++ Performing GET operation for radius Server Groups with Multiple Servers ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	expected_get_json = "{\"openconfig-system:server-group\":[{\"name\":\"radius\", \"config\":{\"name\":\"radius\"},\"servers\":{\"server\":[{\"address\":\"2.2.2.2\",\"config\":{\"address\":\"2.2.2.2\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\", \"auth-port\":1921}}},{\"address\":\"2.2.2.3\",\"config\":{\"address\":\"2.2.2.3\"},\"radius\":{\"config\":{\"secret-key\":\"radiuskey\",\"auth-port\":1852}}}]}, \"state\":{\"name\":\"radius\"}}]}"
	t.Run("Test GET on server-group node", processGetRequest(url, nil, expected_get_json, false))

	t.Log("++++++++++++++ Done GET operation on TACACS+ and RADIUS Server Groups with Multiple Servers  /server-groups/server-group++++++++++++++")

	/* DELETE Operation on /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+] */
	t.Log("++++++++++++++ Performing DELETE operation for radius Server Groups with Multiple Servers ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)
	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.1", expected_map_del, false))
	expected_map_del = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.1.1.2", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done DELETE operation for tacplus Server Groups with Multiple Servers ++++++++++++++")

	/* DELETE Operation on /openconfig-system:system/aaa/server-groups/server-group[name=radius] */
	t.Log("++++++++++++++ Performing DELETE operation for radius Server Groups with Multiple Servers ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.2", expected_map_del, false))
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"2.2.2.3": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|2.2.2.3", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done DELETE operation for radius Server Groups with Multiple Servers ++++++++++++++")

	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	/* POST /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers
	 * POST /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers
	 */
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": ""}, "RADIUS_SERVER": map[string]interface{}{"1.2.2.2": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing POST (create)operation for servergroup tacacs+ servers node ++++++++++++++")
	create_url := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	create_body_json := "{\"openconfig-system:server\": [{\"address\": \"1.2.2.1\", \"config\": { \"address\": \"1.2.2.1\",\"timeout\": 5}, \"tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"bngss\"}}}]}"
	expected_map := map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 5, "passkey": "bngss", "tcp_port": 50}}}

	t.Run("test post(create) on server-group/servers node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify post (create) on server-group/servers tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done post operation on server group/servers tacacs+ functions ++++++++++++++")

	t.Log("++++++++++++++ Performing POST (create)operation for servergroup radius servers node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers"
	create_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.2.2.2\", \"config\": { \"address\": \"1.2.2.2\",\"timeout\": 5}, \"radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":2}}}]}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 5, "passkey": "radius-key", "auth_port": 1813, "retransmit": 2}}}

	t.Run("test post(create) on server-group/servers node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify post (create) on server-group tacacs+ node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup tacacs+ servers node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	create_body_json = "{\"openconfig-system:servers\": {\"server\": [{\"address\": \"1.2.2.1\", \"config\": { \"address\": \"1.2.2.1\",\"timeout\": 6}, \"tacacs\":{\"config\":{\"port\":55, \"secret-key\":\"bngss2\"}}}]}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 6, "passkey": "bngss2", "tcp_port": 55}}}

	t.Run("test PUT(REPLACE) on server-group tacacs+ servers node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify PUT (REPLACE) on server-group tacacs+ servers node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on server group tacacs+  servers node ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup radius servers node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers"
	create_body_json = "{\"openconfig-system:servers\": {\"server\": [{\"address\": \"1.2.2.2\", \"config\": { \"address\": \"1.2.2.2\",\"timeout\": 6}, \"radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key2\", \"retransmit-attempts\":4}}}]}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 6, "passkey": "radius-key2", "auth_port": 1813, "retransmit": 4}}}

	t.Run("test PUT(REPLACE) on server-group radius servers node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify PUT (REPLACE) on server-group radius servers node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on server group radius  servers node ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup tacacs+ servers node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	create_body_json = "{\"openconfig-system:servers\": {\"server\": [{\"address\": \"1.2.2.1\", \"config\": { \"address\": \"1.2.2.1\",\"timeout\": 1}, \"tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"bngss\"}}}]}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 1, "passkey": "bngss", "tcp_port": 50}}}

	t.Run("test PATCH(Update) on server-group tacacs+ servers node", processSetRequest(create_url, create_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify PATCH (UPDATE) on server-group tacacs+ servers node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PATCH operation on server group tacacs+  servers node ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup radius servers node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers"
	create_body_json = "{\"openconfig-system:servers\": {\"server\": [{\"address\": \"1.2.2.2\", \"config\": { \"address\": \"1.2.2.2\",\"timeout\": 1}, \"radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":2}}}]}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 1, "passkey": "radius-key", "auth_port": 1813, "retransmit": 2}}}

	t.Run("test PATCH(UPDATE) on server-group radius servers node", processSetRequest(create_url, create_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify PUT (REPLACE) on server-group radius servers node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on server group radius  servers node ++++++++++++++")

	t.Log("++++++++++++++ Performing GET operation for Server Group tacacs+/servers   ++++++++++++++")
	get_url1 := "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	expected_get_json1 := "{\"openconfig-system:servers\":{\"server\":[{\"address\":\"1.2.2.1\",\"config\":{\"address\":\"1.2.2.1\", \"timeout\":1},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\", \"port\":50}}}]}}"
	t.Run("Test GET on server-group tacacs+ servers node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group tacacs+/servers   ++++++++++++++")

	t.Log("++++++++++++++  Performing GET operation for Server Group radius/servers   ++++++++++++++")
	get_url1 = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers"
	expected_get_json1 = "{\"openconfig-system:servers\":{\"server\":[{\"address\":\"1.2.2.2\",\"config\":{\"address\":\"1.2.2.2\", \"timeout\":1},\"radius\":{\"config\":{\"secret-key\":\"radius-key\", \"auth-port\":1813, \"retransmit-attempts\":2}}}]}}"
	t.Run("Test GET on server-group radius servers node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group radius/servers   ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group tacacs+/servers   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group radius/servers   ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group radius/servers   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group radius/servers   ++++++++++++++")
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	/* POST /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]
	 * POST /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[adress=1.2.2.2]
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]
	 */
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": ""}, "RADIUS_SERVER": map[string]interface{}{"1.2.2.2": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing POST (create)operation for servergroup tacacs+ servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]"
	create_body_json = "{\"config\": {\"address\": \"1.2.2.1\",\"timeout\": 8}, \"tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"bngss\"}}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 8, "passkey": "bngss", "tcp_port": 50}}}

	t.Run("test post(create) on server-group/servers/server node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify post (create) on server-group/servers tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done post operation on server group tacacs+/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing POST (create)operation for servergroup radius servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]"
	create_body_json = "{\"config\": { \"address\": \"1.2.2.2\",\"timeout\": 9}, \"radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":3}}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 9, "passkey": "radius-key", "auth_port": 1813, "retransmit": 3}}}

	t.Run("test post(create) on server-group radius /servers/server node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify post (create) on server-group radius/servers/server  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done post operation on server group radius/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup tacacs+ servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]"
	create_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.2.2.1\", \"config\": { \"address\": \"1.2.2.1\",\"timeout\": 1}, \"tacacs\":{\"config\":{\"port\":60, \"secret-key\":\"bngss2\"}}}]}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 1, "passkey": "bngss2", "tcp_port": 60}}}

	t.Run("test put(replace) on server-group/servers/server node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify put (replace) on server-group/servers tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PUT operation on server group tacacs+/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup radius servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]"
	create_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.2.2.2\", \"config\": { \"address\": \"1.2.2.2\",\"timeout\": 5}, \"radius\":{\"config\":{\"auth-port\":1812, \"secret-key\":\"radius-key2\", \"retransmit-attempts\":1}}}]}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 5, "passkey": "radius-key2", "auth_port": 1812, "retransmit": 1}}}

	t.Run("test PUT(Replace) on server-group radius /servers/server node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify put (replace) on server-group radius/servers/server  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on server group radius/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup tacacs+ servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]"
	create_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.2.2.1\", \"config\": { \"address\": \"1.2.2.1\",\"timeout\": 10}, \"tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"bngss\"}}}]}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"timeout": 10, "passkey": "bngss", "tcp_port": 50}}}

	t.Run("test PATCH(update) on server-group/servers/server node", processSetRequest(create_url, create_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify patch (update) on server-group/servers tacacs+ node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PATCH operation on server group tacacs+/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup radius servers/server node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]"
	create_body_json = "{\"openconfig-system:server\": [{\"address\": \"1.2.2.2\", \"config\": { \"address\": \"1.2.2.2\",\"timeout\": 10}, \"radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":2}}}]}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"timeout": 10, "passkey": "radius-key", "auth_port": 1813, "retransmit": 2}}}

	t.Run("test PATCH(Update) on server-group radius /servers/server node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify patch (update) on server-group radius/servers/server  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PATCH operation on server group radius/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++  Performing GET operation for Server Group tacacs+/servers/server   ++++++++++++++")
	get_url1 = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]"
	expected_get_json1 = "{\"openconfig-system:server\":[{\"address\":\"1.2.2.1\",\"config\":{\"address\":\"1.2.2.1\",\"timeout\":10},\"tacacs\":{\"config\":{\"secret-key\":\"bngss\", \"port\":50}}}]}"
	t.Run("Test GET on server-group tacacs+ servers node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group tacacs+/servers/server ++++++++++++++")

	t.Log("++++++++++++++  Performing GET operation for Server Group radius/servers   ++++++++++++++")
	get_url1 = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]"
	expected_get_json1 = "{\"openconfig-system:server\":[{\"address\":\"1.2.2.2\",\"config\":{\"address\":\"1.2.2.2\",\"timeout\":10},\"radius\":{\"config\":{\"secret-key\":\"radius-key\", \"auth-port\":1813, \"retransmit-attempts\":2}}}]}"
	t.Run("Test GET on server-group radius servers/server node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group radius/servers/server node ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group tacacs+/servers/server   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group tacacs+/servers   ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group radius/servers   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius servers/server radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group radius/servers   ++++++++++++++")
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	/* POST /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs
	 * POST /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs
	 * PUT /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[adress=1.2.2.2]/radius
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs
	 * PATCH /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs
	 * GET /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs
	 * DELETE /openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius
	 */
	//pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"ipaddress": "1.2.2.1"}}, "RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"ipaddress": "1.2.2.2"}}}
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": ""}, "RADIUS_SERVER": map[string]interface{}{"1.2.2.2": ""}}

	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing POST (create)operation for servergroup tacacs+ servers/server/tacacs node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs"
	create_body_json = "{\"openconfig-system:config\":{\"port\":50, \"secret-key\":\"bngss\"}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 50}}}

	t.Run("test post(create) on server-group/servers/server/tacacs node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify post (create) on server-group tacacs+/servers/tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done post operation on server group tacacs+/servers/server/tacacs  functions ++++++++++++++")
	t.Log("++++++++++++++ Performing POST (create)operation for servergroup radius servers/server/radius node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius"
	create_body_json = "{\"openconfig-system:config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":5}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"passkey": "radius-key", "auth_port": 1813, "retransmit": 5}}}

	t.Run("test post(create) on server-group radius /servers/server/radius node", processSetRequest(create_url, create_body_json, "POST", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify post (create) on server-group radius/servers/server  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done post operation on server group radius/servers/server/radius  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup tacacs+ servers/server/tacacs node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs"
	create_body_json = "{\"openconfig-system:tacacs\":{\"config\":{\"port\":60, \"secret-key\":\"bngss2\"}}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"passkey": "bngss2", "tcp_port": 60}}}

	t.Run("test PUT(Replace) on server-group/servers/server/tacacs node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify PUT (Replace) on server-group/servers/tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PUT operation on server group tacacs+/servers/server  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PUT (Replace)operation for servergroup radius servers/server/radius node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius"
	create_body_json = "{\"openconfig-system:radius\":{\"config\":{\"auth-port\":1812, \"secret-key\":\"radius-key2\", \"retransmit-attempts\":6}}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"passkey": "radius-key2", "auth_port": 1812, "retransmit": 6}}}

	t.Run("test PUT(Replace) on server-group radius /servers/server/radius node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify post (create) on server-group radius/servers/server/radius  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PUT operation on server group radius/servers/server/radius  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup tacacs+ servers/server/tacacs node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs"
	create_body_json = "{\"openconfig-system:tacacs\":{\"config\":{\"port\":50, \"secret-key\":\"bngss\"}}}"
	expected_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"passkey": "bngss", "tcp_port": 50}}}

	t.Run("test PATCH(update) on server-group/servers/server/tacacs node", processSetRequest(create_url, create_body_json, "PATCH", false, nil))
	time.Sleep(1 * time.Second)

	t.Run("verify patch (update) on server-group/servers/tacacs node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done  PATCH operation on server group tacacs+/servers/server/tacacs  functions ++++++++++++++")

	t.Log("++++++++++++++ Performing PATCH (Update)operation for servergroup radius servers/server/radius node ++++++++++++++")
	create_url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius"
	create_body_json = "{\"openconfig-system:radius\":{\"config\":{\"auth-port\":1813, \"secret-key\":\"radius-key\", \"retransmit-attempts\":7}}}"
	expected_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"passkey": "radius-key", "auth_port": 1813, "retransmit": 7}}}

	t.Run("test PATCH(Update) on server-group radius /servers/server/radius node", processSetRequest(create_url, create_body_json, "PUT", false, nil))
	time.Sleep(1 * time.Second)
	t.Run("verify patch (update) on server-group radius/servers/server/radius  node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done PATCH operation on server group radius/servers/server/radius  functions ++++++++++++++")

	t.Log("++++++++++++++  Performing GET operation for Server Group tacacs+/servers/server/tacacs   ++++++++++++++")
	get_url1 = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs"
	expected_get_json1 = "{\"openconfig-system:tacacs\":{\"config\":{\"secret-key\":\"bngss\", \"port\":50}}}"
	t.Run("Test GET on server-group tacacs+ servers node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group tacacs+/servers/server/tacacs ++++++++++++++")

	t.Log("++++++++++++++  Performing GET operation for Server Group radius/servers/radius   ++++++++++++++")
	get_url1 = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius"
	expected_get_json1 = "{\"openconfig-system:radius\":{\"config\":{\"secret-key\":\"radius-key\", \"auth-port\":1813, \"retransmit-attempts\":7}}}"
	t.Run("Test GET on server-group radius servers/server/radius node", processGetRequest(get_url1, nil, expected_get_json1, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done GET operation for Server Group radius/servers/server node ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group tacacs+/servers/server/tacacs   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=tacacs+]/servers/server[address=1.2.2.1]/tacacs"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.2.2.1": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "TACPLUS_SERVER|1.2.2.1", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group tacacs+/servers/server/tacacs ++++++++++++++")

	t.Log("++++++++++++++  Performing DELETE operation for Server Group radius/servers   ++++++++++++++")
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.2.2.2]/radius"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius servers/server radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.2.2.2": map[string]interface{}{"NULL": "NULL"}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|1.2.2.2", expected_map_del, false))
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++  Done DELETE operation for Server Group radius/servers/server/radius   ++++++++++++++")
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)

	t.Log("++++++++++++++ Performing POST (CREATE) operation for TACACS+ and RADIUS Server Groups  at server-group container level with Multiple Servers ++++++++++++++")
	pre_req_map = map[string]interface{}{}
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": "", "1.1.1.2": ""}, "RADIUS_SERVER": map[string]interface{}{"2.2.2.2": "", "2.2.2.3": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	t.Log("\n\n+++++++++++++ Performing Delete on server-group radius configuration node ++++++++++++")
	pre_req_map = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{"passkey": "test123"}}}
	loadDB(db.ConfigDB, pre_req_map) // Ensure data is loaded correctly into the DB
	time.Sleep(1 * time.Second)
	//	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]/servers/server[address=1.1.1.2]/radius/config/secret-key"
	url = "/openconfig-system:system/aaa/server-groups/server-group[name=radius]"
	t.Log("Before delete: ", db.ConfigDB) // Log before delete
	t.Run("Test delete on server-group radius configuration node", processDeleteRequest(url, false))
	time.Sleep(1 * time.Second)

	t.Log("After delete: ", db.ConfigDB) // Log after delete
	expected_map_del = map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.1.1.2": map[string]interface{}{}}}
	t.Run("Verify delete on server-group radius configuration node", verifyDbResult(rclient, "RADIUS_SERVER|1.1.1.2", expected_map_del, false))
	time.Sleep(1 * time.Second)

	unloadDB(db.ConfigDB, map[string]interface{}{"RADIUS_SERVER": map[string]interface{}{"1.1.1.2": ""}})
	time.Sleep(1 * time.Second)

	t.Log("\n\n+++++++")

	t.Log("++++++++++++++ Performing GET operation for /system/aaa ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "secret1", "timeout": 22}}, "RADIUS_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"passkey": "secret3", "timeout": 25}}}
	t.Log("Pre-requisite map for /system/aaa:", pre_req_map)
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""}, "RADIUS_SERVER": map[string]interface{}{"3.3.3.3": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa"
	expected_get_json = "{\"openconfig-system:aaa\":{\"server-groups\":{\"server-group\":[{\"config\":{\"name\":\"radius\"}, \"name\":\"radius\", \"servers\":{\"server\":[{\"address\":\"3.3.3.3\", \"config\":{\"address\":\"3.3.3.3\", \"timeout\":25}, \"radius\":{\"config\":{\"secret-key\":\"secret3\"}}}]}, \"state\":{\"name\":\"radius\"}}, {\"config\":{\"name\":\"tacacs+\"}, \"name\":\"tacacs+\", \"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\", \"timeout\":22}, \"tacacs\":{\"config\":{\"secret-key\":\"secret1\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}}}"
	t.Run("Test GET on /system/aaa", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done GET operation for /system/aaa ++++++++++++++")

	t.Log("++++++++++++++ Performing GET operation for /system/aaa ++++++++++++++")
	pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "secret1", "timeout": 22}}}
	t.Log("Pre-requisite map for /system/aaa:", pre_req_map)
	cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""}}
	loadDB(db.ConfigDB, pre_req_map)
	time.Sleep(1 * time.Second)
	url = "/openconfig-system:system/aaa"
	expected_get_json = "{\"openconfig-system:aaa\":{\"server-groups\":{\"server-group\":[{\"config\":{\"name\":\"tacacs+\"}, \"name\":\"tacacs+\", \"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\", \"timeout\":22}, \"tacacs\":{\"config\":{\"secret-key\":\"secret1\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}}}"
	t.Run("Test GET on /system/aaa", processGetRequest(url, nil, expected_get_json, false))
	time.Sleep(1 * time.Second)
	t.Log("Unloading data from the database...")
	unloadDB(db.ConfigDB, cleanuptbl)
	time.Sleep(1 * time.Second)
	t.Log("++++++++++++++ Done GET operation for /system/aaa tacacs ++++++++++++++")
	/*
			* Commenting  out the test case as the current-datetime, up-time, boot-time are varaible

		t.Log("++++++++++++++ GET operation for /system ++++++++++++++")
		pre_req_map = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": map[string]interface{}{"passkey": "secret1", "timeout": 22}}, "RADIUS_SERVER": map[string]interface{}{"3.3.3.3": map[string]interface{}{"passkey": "secret3", "timeout": 25}}}
		t.Log("Pre-requisite map for /system:", pre_req_map)
		cleanuptbl = map[string]interface{}{"TACPLUS_SERVER": map[string]interface{}{"1.1.1.1": ""}, "RADIUS_SERVER": map[string]interface{}{"3.3.3.3": ""}}
		loadDB(db.ConfigDB, pre_req_map)
		time.Sleep(1 * time.Second)
		url = "/openconfig-system:system"
		expected_get_json = "{\"openconfig-system:system\":{\"aaa\":{\"server-groups\":{\"server-group\":[{\"config\":{\"name\":\"radius\"}, \"name\":\"radius\", \"servers\":{\"server\":[{\"address\":\"3.3.3.3\", \"config\":{\"address\":\"3.3.3.3\", \"timeout\":25}, \"radius\":{\"config\":{\"secret-key\":\"secret3\"}}}]}, \"state\":{\"name\":\"radius\"}}, {\"config\":{\"name\":\"tacacs+\"}, \"name\":\"tacacs+\", \"servers\":{\"server\":[{\"address\":\"1.1.1.1\", \"config\":{\"address\":\"1.1.1.1\", \"timeout\":22}, \"tacacs\":{\"config\":{\"secret-key\":\"secret1\"}}}]}, \"state\":{\"name\":\"tacacs+\"}}]}}, \"state\":{\"boot-time\":1750101238566211541, \"current-datetime\":\"2025-06-25T14:07:18Z+00:00\", \"up-time\":759200000000000}}}"
		t.Run("Test GET on /system", processGetRequest(url, nil, expected_get_json, false))
		time.Sleep(1 * time.Second)
		t.Log("Unloading data from the database...")
		unloadDB(db.ConfigDB, cleanuptbl)
		time.Sleep(1 * time.Second)
		t.Log("++++++++++++++ Done GET operation for /system ++++++++++++++")

		****/
}
