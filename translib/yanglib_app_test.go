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

package translib

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

func TestYanglibGetAll(t *testing.T) {
	data := getYanglibDataT(t, "", "", "")
	v := data["ietf-yang-library:modules-state"]
	if v1, ok := v.(map[string]interface{}); ok {
		validateMSetID(t, v1["module-set-id"])
		v = v1["module"]
	}
	if v1, ok := v.([]interface{}); !ok || len(v1) == 0 {
		t.Fatalf("App returned incorrect info.. %v", data)
	}
}

func TestYanglibGetMsetID(t *testing.T) {
	data := getYanglibDataT(t, "", "", "module-set-id")
	if len(data) != 1 {
		t.Fatalf("App returned incorrect info.. %v", data)
	}
	validateMSetID(t, data["ietf-yang-library:module-set-id"])
}

func validateMSetID(t *testing.T, msetID interface{}) {
	if m, ok := msetID.(string); !ok || m != GetYangModuleSetID() {
		t.Fatalf("App returned incorrect module-set-id \"%s\"; expected \"%s\"",
			msetID, GetYangModuleSetID())
	}
}

func TestYanglibGetOne(t *testing.T) {
	data := getYanglibDataT(t, "ietf-yang-library", "2016-06-21", "")

	var m map[string]interface{}
	if v, ok := data["ietf-yang-library:module"].([]interface{}); ok && len(v) == 1 {
		m, ok = v[0].(map[string]interface{})
	}

	if m["name"] != "ietf-yang-library" ||
		m["revision"] != "2016-06-21" ||
		m["namespace"] != "urn:ietf:params:xml:ns:yang:ietf-yang-library" ||
		m["conformance-type"] != "implement" {
		t.Fatalf("App returned incorrect info.. %v", data)
	}
}

func TestYanglibGetOneAttr(t *testing.T) {
	data := getYanglibDataT(t, "ietf-yang-library", "2016-06-21", "namespace")
	if data["ietf-yang-library:namespace"] != "urn:ietf:params:xml:ns:yang:ietf-yang-library" {
		t.Fatalf("App returned incorrect info.. %v", data)
	}
}

func TestYanglibSchemaURL(t *testing.T) {
	defer SetSchemaRootURL("")

	t.Run("default", testYlibSchema(nil))

	SetSchemaRootURL("https://localhost/schema1")
	t.Run("no_slash", testYlibSchema("https://localhost/schema1/ietf-yang-library.yang"))

	SetSchemaRootURL("https://localhost/schema2/")
	t.Run("with_slash", testYlibSchema("https://localhost/schema2/ietf-yang-library.yang"))

	SetSchemaRootURL("")
	t.Run("reset", testYlibSchema(nil))
}

func testYlibSchema(expURL interface{}) func(*testing.T) {
	return func(t *testing.T) {
		data := getYanglibDataT(t, "ietf-yang-library", "2016-06-21", "schema")
		if data["ietf-yang-library:schema"] != expURL {
			t.Fatalf("Expected schema url '%s', found '%s'",
				expURL, data["ietf-yang-library:schema"])
		}
	}
}

func TestYanglibConformance(t *testing.T) {
	t.Run("ietf-yang-library", testConfType("ietf-yang-library", "2016-06-21", "implement"))
	t.Run("ietf-yang-types", testConfType("ietf-yang-types", "2013-07-15", "import"))
	t.Run("ietf-inet-types", testConfType("ietf-inet-types", "2013-07-15", "import"))
}

func testConfType(mod, rev, exp string) func(*testing.T) {
	return func(t *testing.T) {
		data := getYanglibDataT(t, mod, rev, "conformance-type")
		if data["ietf-yang-library:conformance-type"] != exp {
			t.Fatalf("App returned unexpected conformance-type for %s@%s; found=%s, exp=%s",
				mod, rev, data["ietf-yang-library:conformance-type"], exp)
		}
	}
}

func TestYanglibGetUnknown(t *testing.T) {
	_, err := getYanglibData("unknown", "0000-00-00", "")
	if _, ok := err.(tlerr.NotFoundError); !ok {
		t.Fatalf("Expected NotFoundError, got %T", err)
	}
}

func getYanglibData(name, rev, attr string) (map[string]interface{}, error) {
	u := "/ietf-yang-library:modules-state"
	if name != "" || rev != "" {
		u += fmt.Sprintf("/module[name=%s][revision=%s]", name, rev)
	}
	if attr != "" {
		u += ("/" + attr)
	}

	data := make(map[string]interface{})
	response, err := Get(GetRequest{Path: u})
	if err == nil {
		err = json.Unmarshal(response.Payload, &data)
	}

	return data, err
}

func getYanglibDataT(t *testing.T, name, rev, attr string) map[string]interface{} {
	data, err := getYanglibData(name, rev, attr)
	if err != nil {
		t.Fatalf("Unexpected erorr: %v", err)
	}
	return data
}
