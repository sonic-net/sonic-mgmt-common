////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 SONiC contributors.                                       //
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
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func TestPlatformHealthzStateGetProjection(t *testing.T) {
	components := &ocbinds.OpenconfigPlatform_Components{}
	faultEntry := validPlatformFaultEntry()
	faultEntry.Field["component_name"] = "PSU+0:A"
	fault, err := parsePlatformFault(
		db.Key{Comp: []string{"PSU%2B0%3AA", "SYMPTOM_OVER_THRESHOLD"}},
		faultEntry,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := addPlatformFault(components, fault); err != nil {
		t.Fatal(err)
	}
	if components.Component["PSU+0:A"].Healthz.State != nil {
		t.Fatal("a fault row alone was treated as a health assessment")
	}

	rows := []platformHealthzRow{
		{key: db.Key{Comp: []string{"%"}}, entry: validPlatformHealthzEntry()},
		{key: db.Key{Comp: []string{"PSU%2B0%3AA"}}, entry: validPlatformHealthzEntry()},
	}
	if got := addPlatformHealthzRows(components, rows, "PSU+0:A"); got != 1 {
		t.Fatalf("mapped states = %d, want 1", got)
	}
	component := components.Component["PSU+0:A"]
	if component == nil || component.Healthz == nil || component.Healthz.State == nil {
		t.Fatal("known component health was not mapped")
	}
	if component.Healthz.Faults == nil || len(component.Healthz.Faults.Fault) != 1 {
		t.Fatal("existing FAULT_INFO mapping was lost")
	}
	state := component.Healthz.State
	if state.Status != ocbinds.OpenconfigPlatform_Components_Component_Healthz_State_Status_UNHEALTHY ||
		state.LastUnhealthy == nil || *state.LastUnhealthy != 1745614206987654321 ||
		state.UnhealthyCount == nil || *state.UnhealthyCount != 2 {
		t.Fatalf("unexpected Healthz state: %+v", state)
	}

	clear := validPlatformHealthzEntry()
	clear.Field["status"] = "HEALTHY"
	clear.Field["unhealthy_count"] = "0"
	delete(clear.Field, "last_unhealthy")
	stateRow, err := parsePlatformHealthzState(db.Key{Comp: []string{"PSU0"}}, clear)
	if err != nil {
		t.Fatalf("inactive-only recovery projection rejected: %v", err)
	}
	if stateRow.lastUnhealthy != nil || stateRow.unhealthyCount != 0 {
		t.Fatalf("unknown last-unhealthy was fabricated: %+v", stateRow)
	}
}

func TestPlatformHealthzStateRejectsUnknownRows(t *testing.T) {
	bad := []struct {
		name  string
		key   db.Key
		field string
		value string
	}{
		{name: "missing status", key: db.Key{Comp: []string{"PSU0"}}, field: "status", value: ""},
		{name: "unspecified status", key: db.Key{Comp: []string{"PSU0"}}, field: "status", value: "UNSPECIFIED"},
		{name: "bad count", key: db.Key{Comp: []string{"PSU0"}}, field: "unhealthy_count", value: "-1"},
		{name: "bad time", key: db.Key{Comp: []string{"PSU0"}}, field: "last_unhealthy", value: "1.2"},
		{name: "bad key", key: db.Key{Comp: []string{"%"}}},
	}
	for _, test := range bad {
		t.Run(test.name, func(t *testing.T) {
			entry := validPlatformHealthzEntry()
			if test.field != "" {
				entry.Field[test.field] = test.value
			}
			if _, err := parsePlatformHealthzState(test.key, entry); err == nil {
				t.Fatal("malformed or unassessed row was accepted")
			}
		})
	}
}

func TestPlatformHealthzSubscribePaths(t *testing.T) {
	app := &PlatformApp{}
	for _, target := range []string{
		"/openconfig-platform:components/component[name=PSU+0:A]/healthz/state",
		"/openconfig-platform:components/component[name=PSU+0:A]/openconfig-platform-healthz:healthz/state/status",
		"/openconfig-platform:components/component[name=PSU+0:A]/healthz/state/last-unhealthy",
		"/openconfig-platform:components/component[name=PSU+0:A]/healthz/state/unhealthy-count",
	} {
		response, err := app.translateSubscribe(translateSubRequest{path: target})
		if err != nil {
			t.Fatalf("translateSubscribe(%q): %v", target, err)
		}
		if len(response.ntfAppInfoTrgt) != 1 || len(response.ntfAppInfoTrgtChlds) != 0 {
			t.Fatalf("health state target mapping for %q: %+v", target, response)
		}
		info := response.ntfAppInfoTrgt[0]
		if info.table == nil || info.table.Name != "COMPONENT_HEALTH_INFO" ||
			info.dbno != db.StateDB || info.key == nil || info.key.Get(0) != "PSU%2B0%3AA" ||
			!info.isOnChangeSupported || info.pType != OnChange {
			t.Fatalf("unexpected health state subscription mapping: %+v", info)
		}
	}

	response, err := app.translateSubscribe(translateSubRequest{path: "/openconfig-platform:components/component[name=PSU0]/healthz"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.ntfAppInfoTrgt) != 1 || response.ntfAppInfoTrgt[0].dbno != db.MaxDB ||
		len(response.ntfAppInfoTrgtChlds) != 2 {
		t.Fatalf("healthz ancestor does not include state and faults: %+v", response)
	}
	if response.ntfAppInfoTrgtChlds[0].table.Name != "FAULT_INFO" ||
		response.ntfAppInfoTrgtChlds[1].table.Name != "COMPONENT_HEALTH_INFO" {
		t.Fatalf("healthz ancestor table mappings: %+v", response.ntfAppInfoTrgtChlds)
	}
	response, err = app.translateSubscribe(translateSubRequest{path: "/openconfig-platform:components"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.ntfAppInfoTrgtChlds) != 2 {
		t.Fatalf("components ancestor has %d mapped children, want 2", len(response.ntfAppInfoTrgtChlds))
	}
}

func TestPlatformHealthzOnChange(t *testing.T) {
	key := db.Key{Comp: []string{"PSU%2B0%3AA"}}
	ancestor, err := ygot.StringToStructuredPath("/openconfig-platform:components")
	if err != nil {
		t.Fatal(err)
	}
	sender := &recordingNotificationSender{}
	processPlatformHealthzOnChange(&apis.NotificationContext{
		Path: ancestor,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			NewValue:     validPlatformHealthzEntry(),
			EntryCreated: true,
		},
	}, sender)
	if len(sender.notifications) != 1 ||
		sender.notifications[0].Path != "/openconfig-platform:components/component[name=PSU+0:A]/healthz" ||
		len(sender.notifications[0].UpdatePaths) != 1 || sender.notifications[0].UpdatePaths[0] != "/state" {
		t.Fatalf("ancestor health update: %+v", sender.notifications)
	}

	leaf, err := ygot.StringToStructuredPath(
		"/openconfig-platform:components/component[name=PSU+0:A]/healthz/state/status")
	if err != nil {
		t.Fatal(err)
	}
	sender.notifications = nil
	updatedCount := validPlatformHealthzEntry()
	updatedCount.Field["unhealthy_count"] = "3"
	processPlatformHealthzOnChange(&apis.NotificationContext{
		Path: leaf,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			OldValue: validPlatformHealthzEntry(),
			NewValue: updatedCount,
		},
	}, sender)
	if len(sender.notifications) != 0 {
		t.Fatalf("unmodified status leaf was reported as changed: %+v", sender.notifications)
	}

	processPlatformHealthzOnChange(&apis.NotificationContext{
		Path: leaf,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			OldValue:     validPlatformHealthzEntry(),
			EntryDeleted: true,
		},
	}, sender)
	if len(sender.notifications) != 1 ||
		sender.notifications[0].Path != "/openconfig-platform:components/component[name=PSU+0:A]/healthz/state" ||
		len(sender.notifications[0].Delete) != 1 || sender.notifications[0].Delete[0] != "/status" {
		t.Fatalf("health leaf deletion: %+v", sender.notifications)
	}
}

func validPlatformHealthzEntry() db.Value {
	return db.Value{Field: map[string]string{
		"status":          "UNHEALTHY",
		"last_unhealthy":  "1745614206987654321",
		"unhealthy_count": "2",
	}}
}
