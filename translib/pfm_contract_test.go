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
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/openconfig/ygot/ygot"
)

func seedPlatformHealthzContractRows(t *testing.T) string {
	t.Helper()
	const component = "DLDD_healthz_contract_test"
	stateDB := getStateDB()
	if stateDB == nil {
		t.Fatal("STATE_DB unavailable")
	}
	t.Cleanup(func() { stateDB.DeleteDB() })
	fault := validPlatformFaultEntry()
	fault.Field["component_name"] = component
	fault.Field["repair_actions"] = `[{"action":"ACTION_RESEAT"},{"action":"ACTION_REPLACE"}]`
	for _, row := range []struct {
		table db.TableSpec
		key   db.Key
		value db.Value
	}{
		{platformHealthzTableSpec, db.Key{Comp: []string{component}}, validPlatformHealthzEntry()},
		{platformFaultTableSpec, db.Key{Comp: []string{component, "SYMPTOM_OVER_THRESHOLD"}}, fault},
	} {
		row := row
		if err := stateDB.SetEntry(&row.table, row.key, row.value); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { stateDB.DeleteEntry(&row.table, row.key) })
	}
	return "/openconfig-platform:components/component[name=" + component + "]/healthz"
}

func TestPlatformDirectHealthzGetQueryContract(t *testing.T) {
	base := seedPlatformHealthzContractRows(t)
	for _, uri := range []string{base, strings.Replace(base, "/healthz", "/openconfig-platform-healthz:healthz", 1)} {
		for _, format := range []TranslibFmtType{TRANSLIB_FMT_IETF_JSON, TRANSLIB_FMT_YGOT} {
			for _, test := range []struct {
				name       string
				query      QueryParameters
				wantStatus bool
			}{
				{name: "all", wantStatus: true},
				{name: "config", query: QueryParameters{Content: "config"}},
				{name: "depth", query: QueryParameters{Depth: 1}},
				{name: "fields", query: QueryParameters{Fields: []string{"state/status"}}, wantStatus: true},
				{name: "qualified state field", query: QueryParameters{Fields: []string{"openconfig-platform-healthz:state/status"}}, wantStatus: true},
				{name: "qualified status field", query: QueryParameters{Fields: []string{"state/openconfig-platform-healthz:status"}}, wantStatus: true},
				{name: "qualified field path", query: QueryParameters{Fields: []string{"openconfig-platform-healthz:state/openconfig-platform-healthz:status"}}, wantStatus: true},
			} {
				t.Run(fmt.Sprintf("%s/%d", test.name, format), func(t *testing.T) {
					response, err := Get(GetRequest{Path: uri, QueryParams: test.query, FmtType: format})
					if err != nil {
						t.Fatal(err)
					}
					payload := response.Payload
					if format == TRANSLIB_FMT_YGOT && response.ValueTree != nil {
						payload, err = dumpIetfJson(response.ValueTree)
						if err != nil {
							t.Fatal(err)
						}
					}
					if strings.Contains(string(payload), "UNHEALTHY") != test.wantStatus {
						t.Fatalf("status selection: %s", payload)
					}
					if test.name != "all" && (strings.Contains(string(payload), "SYMPTOM_OVER_THRESHOLD") || strings.Contains(string(payload), "unhealthy-count")) {
						t.Fatalf("query retained unselected Healthz data: %s", payload)
					}
				})
			}
		}
		for _, field := range []string{"state/unknown", "faults/fault", "faults/fault[symptom=SYMPTOM_UNKNOWN]"} {
			if _, err := Get(GetRequest{Path: uri, QueryParams: QueryParameters{Fields: []string{field}}}); err == nil {
				t.Errorf("invalid fields selection %q succeeded", field)
			}
		}
		for _, query := range []QueryParameters{{Content: "config"}, {Fields: []string{"status"}}} {
			if _, err := Get(GetRequest{Path: uri + "/state/status", QueryParams: query}); err == nil {
				t.Errorf("terminal-node query %+v succeeded", query)
			}
		}
		for _, format := range []TranslibFmtType{TRANSLIB_FMT_IETF_JSON, TRANSLIB_FMT_YGOT} {
			response, err := Get(GetRequest{
				Path:        uri + "/faults/fault[symptom=SYMPTOM_OVER_THRESHOLD]/remediations/remediation[index=0]",
				QueryParams: QueryParameters{Content: "config"}, FmtType: format,
			})
			if err != nil || string(response.Payload) != "{}" || response.ValueTree != nil {
				t.Fatalf("content-mismatched list response: %+v, %v", response, err)
			}
		}
	}
}

func TestPlatformHealthzSubscribeContract(t *testing.T) {
	for _, component := range []string{"PSU0", "*"} {
		for _, healthz := range []string{"healthz", "openconfig-platform-healthz:healthz"} {
			uri := "/openconfig-platform:components/component[name=" + component + "]/" + healthz
			for _, mode := range []NotificationType{TargetDefined, OnChange} {
				translated, err := (&subscribeContext{}).translateSubscribe(uri, mode)
				if err != nil {
					t.Fatalf("translate %s (%s): %v", uri, mode, err)
				}
				response := translated.response
				if len(response.ntfAppInfoTrgt) != 2 || len(response.ntfAppInfoTrgtChlds) != 0 {
					t.Fatalf("Healthz target mappings: %+v", response)
				}
				preferences := newIsSubscribeResponse(0, uri)
				collectNotificationPreferences(response.ntfAppInfoTrgt, preferences)
				collectNotificationPreferences(response.ntfAppInfoTrgtChlds, preferences)
				if !preferences.IsOnChangeSupported || !preferences.IsWildcardSupported || preferences.PreferredType != OnChange {
					t.Fatalf("Healthz notification preferences: %+v", preferences)
				}
			}
		}
	}
}

func TestPlatformRemediationGetContract(t *testing.T) {
	uri := seedPlatformHealthzContractRows(t) + "/faults/fault[symptom=SYMPTOM_OVER_THRESHOLD]/remediations/remediation"
	for _, format := range []TranslibFmtType{TRANSLIB_FMT_IETF_JSON, TRANSLIB_FMT_YGOT} {
		for _, test := range []struct {
			key     string
			indices []uint64
		}{
			{indices: []uint64{0, 1}},
			{key: "[index=0]", indices: []uint64{0}},
			{key: "[index=1]", indices: []uint64{1}},
			{key: "[index=2]"},
		} {
			response, err := Get(GetRequest{Path: uri + test.key, FmtType: format})
			if test.indices == nil {
				if !tlerr.IsNotFound(err) {
					t.Fatalf("missing index: %v", err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			var remediations *ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_Remediations
			if format == TRANSLIB_FMT_YGOT {
				remediations = response.ValueTree.(*ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_Remediations)
			} else {
				remediations = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_Remediations{}
				if err := ocbinds.Unmarshal(response.Payload, remediations); err != nil {
					t.Fatal(err)
				}
			}
			if len(remediations.Remediation) != len(test.indices) {
				t.Fatalf("GET %s returned siblings: %+v", test.key, remediations.Remediation)
			}
			for _, index := range test.indices {
				if remediations.Remediation[index] == nil {
					t.Errorf("GET %s omitted index %d", test.key, index)
				}
			}
		}
		response, err := Get(GetRequest{Path: uri + "[index=0]/state/action", FmtType: format})
		if err != nil {
			t.Fatal(err)
		}
		payload := response.Payload
		if format == TRANSLIB_FMT_YGOT {
			payload, err = dumpIetfJson(response.ValueTree)
			if err != nil {
				t.Fatal(err)
			}
		}
		if !strings.Contains(string(payload), "ACTION_RESEAT") || strings.Contains(string(payload), "ACTION_REPLACE") {
			t.Fatalf("keyed action leaf: %s", payload)
		}
	}
}

func TestPlatformWildcardRemediationNotificationContract(t *testing.T) {
	two := validPlatformFaultEntry()
	two.Field["repair_actions"] = `[{"action":"ACTION_RESEAT"},{"action":"ACTION_REPLACE"}]`
	one := validPlatformFaultEntry()
	one.Field["repair_actions"] = `[{"action":"ACTION_REPLACE"}]`
	empty := validPlatformFaultEntry()
	empty.Field["repair_actions"] = "[]"
	for _, remediation := range []string{"remediation", "openconfig-platform-healthz-fault:remediation"} {
		for _, suffix := range []string{"", "/state/action"} {
			uri := "/openconfig-platform:components/component[name=*]/healthz/faults/fault[symptom=*]/remediations/" + remediation + "[index=*]" + suffix
			subscribePath, err := ygot.StringToStructuredPath(uri)
			if err != nil {
				t.Fatal(err)
			}
			for _, event := range []struct {
				name             string
				diff             apis.EntryDiff
				updates, deletes int
			}{
				{name: "initial", diff: apis.EntryDiff{NewValue: two, EntryCreated: true}, updates: 2},
				{name: "changed", diff: apis.EntryDiff{OldValue: one, NewValue: two}, updates: 2},
				{name: "shorter", diff: apis.EntryDiff{OldValue: two, NewValue: one}, updates: 1, deletes: 1},
				{name: "empty", diff: apis.EntryDiff{OldValue: two, NewValue: empty}, deletes: 2},
				{name: "deleted", diff: apis.EntryDiff{OldValue: two, EntryDeleted: true}, deletes: 2},
			} {
				sender := &recordingNotificationSender{}
				key := db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}
				processPlatformFaultOnChange(&apis.NotificationContext{Path: subscribePath, Key: &key, EntryDiff: event.diff}, sender)
				updates, deletes := 0, 0
				for _, notification := range sender.notifications {
					updates += len(notification.UpdatePaths)
					deletes += len(notification.Delete)
					for _, relative := range append(append([]string{}, notification.UpdatePaths...), notification.Delete...) {
						target := notification.Path + relative
						resolved, err := ygot.StringToStructuredPath(target)
						if err != nil || path.HasWildcardKey(resolved) {
							t.Fatalf("%s unresolved notification: %s (%v)", event.name, target, err)
						}
						rootType := reflect.TypeOf(ocbinds.OpenconfigPlatform_Components{})
						if _, _, err := getRequestBinder(&target, nil, GET, &rootType).unMarshall(); err != nil {
							t.Fatalf("%s GET bind %s: %v", event.name, target, err)
						}
					}
				}
				if updates != event.updates || deletes != event.deletes {
					t.Fatalf("%s %s: updates/deletes=%d/%d, want %d/%d", uri, event.name, updates, deletes, event.updates, event.deletes)
				}
			}
		}
	}
}

func TestPlatformRemediationRemovalContract(t *testing.T) {
	two := validPlatformFaultEntry()
	two.Field["repair_actions"] = `[{"action":"ACTION_RESEAT"},{"action":"ACTION_REPLACE"}]`
	one := validPlatformFaultEntry()
	one.Field["repair_actions"] = `[{"action":"ACTION_REPLACE"}]`
	const faultPath = "/openconfig-platform:components/component[name=PSU0]/healthz/faults/fault[symptom=SYMPTOM_OVER_THRESHOLD]"
	for _, test := range []struct {
		suffix           string
		updates, deletes int
	}{
		{suffix: "/remediations/remediation[index=1]", deletes: 1},
		{suffix: "/remediations/remediation[index=1]/state/action", deletes: 1},
		{suffix: "/remediations", updates: 1, deletes: 1},
		{updates: 1, deletes: 1},
		{suffix: "/state", updates: 1},
	} {
		subscribePath, err := ygot.StringToStructuredPath(faultPath + test.suffix)
		if err != nil {
			t.Fatal(err)
		}
		key := db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}
		sender := &recordingNotificationSender{}
		processPlatformFaultOnChange(&apis.NotificationContext{
			Path: subscribePath, Key: &key,
			EntryDiff: apis.EntryDiff{OldValue: two, NewValue: one},
		}, sender)
		updates, deletes := 0, 0
		for _, notification := range sender.notifications {
			updates += len(notification.UpdatePaths)
			deletes += len(notification.Delete)
			for _, relative := range notification.Delete {
				want := faultPath + "/remediations/remediation[index=1]"
				if strings.HasSuffix(test.suffix, "/state/action") {
					want += "/state/action"
				}
				if got := notification.Path + relative; got != want {
					t.Fatalf("removed index path = %s, want %s", got, want)
				}
			}
		}
		if updates != test.updates || deletes != test.deletes {
			t.Fatalf("%s updates/deletes = %d/%d, want %d/%d", test.suffix, updates, deletes, test.updates, test.deletes)
		}
	}
}
