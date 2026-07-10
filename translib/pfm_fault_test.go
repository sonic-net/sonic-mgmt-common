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
	"math"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func TestDecimalEpochToNanoseconds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  uint64
	}{
		{name: "seconds", input: "1745614206", want: 1745614206000000000},
		{name: "fraction", input: "1745614206.0123456", want: 1745614206012345600},
		{name: "nanoseconds", input: "1.000000001", want: 1000000001},
		{name: "exact trailing precision", input: "1.1234567890", want: 1123456789},
		{name: "zero", input: "0.0000000000", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decimalEpochToNanoseconds(test.input)
			if err != nil {
				t.Fatalf("decimalEpochToNanoseconds(%q) failed: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("decimalEpochToNanoseconds(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}

	invalid := []string{"", "-1", "+1", ".1", "1.", "1.2.3", "1e3", "1.1234567891", "18446744074"}
	for _, input := range invalid {
		if got, err := decimalEpochToNanoseconds(input); err == nil {
			t.Errorf("decimalEpochToNanoseconds(%q) = %d, want error", input, got)
		}
	}

	maxSeconds := uint64(math.MaxUint64 / 1_000_000_000)
	if _, err := decimalEpochToNanoseconds("18446744073.709551616"); err == nil || maxSeconds != 18446744073 {
		t.Errorf("overflowing fractional timestamp was accepted")
	}
}

func TestParseAndBuildPlatformFault(t *testing.T) {
	entry := validPlatformFaultEntry()
	entry.Field["repair_actions"] = `[
		{"action":"ACTION_RESEAT"},
		{"action":"openconfig-platform-healthz-fault:ACTION_COLD_REBOOT"},
		{"action":"ACTION_POWER_CYCLE"},
		{"action":"ACTION_FACTORY_RESET"},
		{"action":"ACTION_REPLACE"},
		{"action":"ACTION_WARM_REBOOT"}
	]`

	fault, err := parsePlatformFault(db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}, entry)
	if err != nil {
		t.Fatalf("parsePlatformFault failed: %v", err)
	}
	if fault.originTime != 1745614206012345600 || fault.lastDetectionTime != 1745614206987654321 {
		t.Fatalf("timestamps were not converted exactly: %#v", fault)
	}

	components := &ocbinds.OpenconfigPlatform_Components{}
	if err := addPlatformFault(components, fault); err != nil {
		t.Fatalf("addPlatformFault failed: %v", err)
	}
	component := components.Component["PSU0"]
	faultNode := component.Healthz.Faults.Fault[fault.symptom]
	if faultNode.State.Status != ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_ACTIVE {
		t.Errorf("status = %v, want ACTIVE", faultNode.State.Status)
	}
	if got := *faultNode.State.Counters.Occurrences; got != 2 {
		t.Errorf("occurrences = %d, want 2", got)
	}
	if got := len(faultNode.Remediations.Remediation); got != 6 {
		t.Fatalf("remediation count = %d, want 6", got)
	}
	for i := uint64(0); i < 6; i++ {
		remediation := faultNode.Remediations.Remediation[i]
		if remediation == nil || *remediation.State.Index != i {
			t.Fatalf("remediation %d does not use its zero-based list position", i)
		}
		if got := *remediation.State.Target; got != "PSU0" {
			t.Errorf("remediation %d target = %q, want PSU0", i, got)
		}
	}
}

func TestParseInactivePlatformFaultWithNoRemediations(t *testing.T) {
	entry := validPlatformFaultEntry()
	entry.Field["status"] = "INACTIVE"
	entry.Field["repair_actions"] = "[]"

	fault, err := parsePlatformFault(
		db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}, entry)
	if err != nil {
		t.Fatalf("parsePlatformFault rejected inactive fault without remediations: %v", err)
	}
	if len(fault.repairActions) != 0 {
		t.Fatalf("inactive fault remediation count = %d, want 0", len(fault.repairActions))
	}
	if fault.status != ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_INACTIVE {
		t.Fatalf("inactive fault status = %v, want INACTIVE", fault.status)
	}
	components := &ocbinds.OpenconfigPlatform_Components{}
	if err := addPlatformFault(components, fault); err != nil {
		t.Fatalf("addPlatformFault failed: %v", err)
	}
	faultNode := components.Component["PSU0"].Healthz.Faults.Fault[fault.symptom]
	if faultNode.Remediations != nil {
		t.Fatal("inactive fault with empty repair_actions exported a remediations container")
	}
}

func TestAddPlatformFaultRowsIsolatesMalformedRows(t *testing.T) {
	bad := validPlatformFaultEntry()
	bad.Field["component_name"] = ""

	rows := []platformFaultRow{
		{key: db.Key{Comp: []string{"BROKEN", "SYMPTOM_OVER_THRESHOLD"}}, entry: bad},
		{key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}, entry: validPlatformFaultEntry()},
	}
	components := &ocbinds.OpenconfigPlatform_Components{}
	if got := addPlatformFaultRows(components, rows, "", ""); got != 1 {
		t.Fatalf("added row count = %d, want 1", got)
	}
	if _, ok := components.Component["PSU0"]; !ok {
		t.Fatal("valid row after malformed row was not exported")
	}
	if _, ok := components.Component["BROKEN"]; ok {
		t.Fatal("malformed row was exported")
	}
}

func TestFaultKeyComponentCodecMatchesProducer(t *testing.T) {
	tests := []struct {
		plain   string
		encoded string
	}{
		{plain: "PSU0", encoded: "PSU0"},
		{plain: "a-b_c.d~e", encoded: "a-b_c.d~e"},
		{plain: "*", encoded: "%2A"},
		{plain: "a+b:c@d&e=f|g/h i%j", encoded: "a%2Bb%3Ac%40d%26e%3Df%7Cg%2Fh%20i%25j"},
		{plain: "PSU-é", encoded: "PSU-%C3%A9"},
	}
	if got := faultKeyPattern("*"); got != "*" {
		t.Errorf("faultKeyPattern(*) = %q, want wildcard", got)
	}
	for _, test := range tests {
		if got := encodeFaultKeyComponent(test.plain); got != test.encoded {
			t.Errorf("encodeFaultKeyComponent(%q) = %q, want %q", test.plain, got, test.encoded)
		}
		if got, err := decodeFaultKeyComponent(test.encoded); err != nil || got != test.plain {
			t.Errorf("decodeFaultKeyComponent(%q) = %q, %v; want %q", test.encoded, got, err, test.plain)
		}
	}

	entry := validPlatformFaultEntry()
	entry.Field["component_name"] = "PSU+0:A"
	fault, err := parsePlatformFault(
		db.Key{Comp: []string{"PSU%2B0%3AA", "SYMPTOM_OVER_THRESHOLD"}}, entry)
	if err != nil {
		t.Fatalf("parsePlatformFault rejected producer-compatible escaped key: %v", err)
	}
	if fault.component != "PSU+0:A" {
		t.Errorf("decoded component = %q, want PSU+0:A", fault.component)
	}
}

func TestParsePlatformFaultRejectsInvalidMappings(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
		key   db.Key
	}{
		{name: "empty component type", field: "component_type", value: "", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "empty component name", field: "component_name", value: "", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "key mismatch", field: "component_name", value: "PSU1", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "unknown symptom", field: "symptom", value: "VENDOR_SYMPTOM", key: db.Key{Comp: []string{"PSU0", "VENDOR_SYMPTOM"}}},
		{name: "wrong symptom namespace", field: "symptom", value: "vendor:SYMPTOM_OVER_THRESHOLD", key: db.Key{Comp: []string{"PSU0", "vendor%3ASYMPTOM_OVER_THRESHOLD"}}},
		{name: "unknown status", field: "status", value: "BROKEN", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "prefixed status", field: "status", value: "vendor:ACTIVE", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "zero occurrences", field: "occurrences", value: "0", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "empty description", field: "description", value: " ", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "nested action json", field: "repair_actions", value: `[{"action":`, key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "unknown action", field: "repair_actions", value: `[{"action":"ACTION_VENDOR"}]`, key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "wrong action namespace", field: "repair_actions", value: `[{"action":"vendor:ACTION_RESEAT"}]`, key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "missing action list", field: "repair_actions", value: "", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "null action list", field: "repair_actions", value: "null", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
		{name: "active empty action list", field: "repair_actions", value: "[]", key: db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := validPlatformFaultEntry()
			entry.Field[test.field] = test.value
			if _, err := parsePlatformFault(test.key, entry); err == nil {
				t.Fatal("parsePlatformFault succeeded, want error")
			}
		})
	}
}

func TestPlatformFaultIdentityValueSupportsCompiledVendorIdentity(t *testing.T) {
	identities := map[int64]ygot.EnumDefinition{
		1: {Name: "ACTION_RESEAT", DefiningModule: platformFaultModule},
		7: {Name: "ACTION_REPAIR_FABRIC", DefiningModule: "vendor-healthz"},
	}
	tests := []struct {
		value string
		want  int64
	}{
		{value: "ACTION_RESEAT", want: 1},
		{value: platformFaultModule + ":ACTION_RESEAT", want: 1},
		{value: platformFaultModulePrefix + ":ACTION_RESEAT", want: 1},
		{value: "vendor-healthz:ACTION_REPAIR_FABRIC", want: 7},
	}
	for _, test := range tests {
		got, err := platformFaultIdentityValue(test.value, identities)
		if err != nil {
			t.Errorf("platformFaultIdentityValue(%q) failed: %v", test.value, err)
			continue
		}
		if got != test.want {
			t.Errorf("platformFaultIdentityValue(%q) = %d, want %d", test.value, got, test.want)
		}
	}

	for _, value := range []string{
		"ACTION_REPAIR_FABRIC",
		"vendor-healthz:ACTION_RESEAT",
		"vendor-healthz:",
		"vendor:extra:ACTION_RESEAT",
	} {
		if _, err := platformFaultIdentityValue(value, identities); err == nil {
			t.Errorf("platformFaultIdentityValue(%q) succeeded, want error", value)
		}
	}
}

func TestPlatformFaultSymptomUsesGeneratedIdentities(t *testing.T) {
	tests := []struct {
		value string
		want  ocbinds.E_OpenconfigPlatformHealthzFault_SYMPTOM_BASE
	}{
		{"SYMPTOM_OVER_THRESHOLD", ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_SYMPTOM_OVER_THRESHOLD},
		{"oc-platform-healthz-fault:SYMPTOM_UNDER_THRESHOLD", ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_SYMPTOM_UNDER_THRESHOLD},
		{"openconfig-platform-healthz-fault:SYMPTOM_UNKNOWN", ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_SYMPTOM_UNKNOWN},
	}
	for _, test := range tests {
		got, err := platformFaultSymptom(test.value)
		if err != nil {
			t.Errorf("platformFaultSymptom(%q) failed: %v", test.value, err)
			continue
		}
		if got != test.want {
			t.Errorf("platformFaultSymptom(%q) = %v, want %v", test.value, got, test.want)
		}
	}
	if _, err := platformFaultSymptom("vendor-healthz:SYMPTOM_UNKNOWN"); err == nil {
		t.Error("platformFaultSymptom accepted an identity from an unavailable module")
	}
}

func TestProcessPlatformFaultOnChange(t *testing.T) {
	subscribePath, err := ygot.StringToStructuredPath(
		"/openconfig-platform:components/component[name=PSU0]/healthz/faults")
	if err != nil {
		t.Fatal(err)
	}
	key := db.Key{Comp: []string{"PSU0", "SYMPTOM_OVER_THRESHOLD"}}
	sender := &recordingNotificationSender{}

	processPlatformFaultOnChange(&apis.NotificationContext{
		Path: subscribePath,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			NewValue:     validPlatformFaultEntry(),
			EntryCreated: true,
		},
	}, sender)
	if len(sender.notifications) != 1 {
		t.Fatalf("create notifications = %d, want 1", len(sender.notifications))
	}
	wantPath := "/openconfig-platform:components/component[name=PSU0]/healthz/faults/fault[symptom=SYMPTOM_OVER_THRESHOLD]"
	wantParent := "/openconfig-platform:components/component[name=PSU0]/healthz/faults"
	if got := sender.notifications[0].Path; got != wantParent {
		t.Errorf("create notification path = %q, want %q", got, wantParent)
	}
	if len(sender.notifications[0].UpdatePaths) != 1 {
		t.Error("create notification does not request a fresh GET")
	} else if got := sender.notifications[0].UpdatePaths[0]; got != "/fault[symptom=SYMPTOM_OVER_THRESHOLD]" {
		t.Errorf("create relative update path = %q, want fault list entry", got)
	}

	sender.notifications = nil
	processPlatformFaultOnChange(&apis.NotificationContext{
		Path: subscribePath,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			OldValue:     validPlatformFaultEntry(),
			EntryDeleted: true,
		},
	}, sender)
	if len(sender.notifications) != 1 || len(sender.notifications[0].Delete) != 1 {
		t.Fatal("delete did not produce one fault-path deletion")
	}
	wantDeleteParent := "/openconfig-platform:components/component[name=PSU0]/healthz/faults"
	if got := sender.notifications[0].Path; got != wantDeleteParent {
		t.Errorf("delete notification path = %q, want %q", got, wantDeleteParent)
	}
	if got := sender.notifications[0].Delete[0]; got != "/fault[symptom=SYMPTOM_OVER_THRESHOLD]" {
		t.Errorf("delete relative path = %q, want fault list entry", got)
	}

	leafPath, err := ygot.StringToStructuredPath(wantPath + "/state/status")
	if err != nil {
		t.Fatal(err)
	}
	sender.notifications = nil
	processPlatformFaultOnChange(&apis.NotificationContext{
		Path: leafPath,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			NewValue:     validPlatformFaultEntry(),
			EntryCreated: true,
		},
	}, sender)
	if len(sender.notifications) != 1 {
		t.Fatalf("leaf update notifications = %d, want 1", len(sender.notifications))
	}
	if got := sender.notifications[0].Path; got != wantPath+"/state" {
		t.Errorf("leaf update notification path = %q, want state container", got)
	}
	if len(sender.notifications[0].UpdatePaths) != 1 {
		t.Fatalf("leaf update paths = %d, want 1", len(sender.notifications[0].UpdatePaths))
	}
	if got := sender.notifications[0].UpdatePaths[0]; got != "/status" {
		t.Errorf("leaf update relative path = %q, want /status", got)
	}

	sender.notifications = nil
	processPlatformFaultOnChange(&apis.NotificationContext{
		Path: leafPath,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			OldValue:     validPlatformFaultEntry(),
			EntryDeleted: true,
		},
	}, sender)
	if len(sender.notifications) != 1 {
		t.Fatalf("leaf delete notifications = %d, want 1", len(sender.notifications))
	}
	if got := sender.notifications[0].Path; got != wantPath+"/state" {
		t.Errorf("leaf delete notification path = %q, want state container", got)
	}
	if len(sender.notifications[0].Delete) != 1 {
		t.Fatalf("leaf delete paths = %d, want 1", len(sender.notifications[0].Delete))
	}
	if got := sender.notifications[0].Delete[0]; got != "/status" {
		t.Errorf("leaf delete relative path = %q, want /status", got)
	}

	sender.notifications = nil
	bad := validPlatformFaultEntry()
	bad.Field["component_name"] = ""
	processPlatformFaultOnChange(&apis.NotificationContext{
		Path: subscribePath,
		Key:  &key,
		EntryDiff: apis.EntryDiff{
			NewValue:     bad,
			EntryCreated: true,
		},
	}, sender)
	if len(sender.notifications) != 0 {
		t.Fatal("malformed new row produced an update")
	}
}

func TestTranslateFaultSubscribe(t *testing.T) {
	app := &PlatformApp{}
	response, err := app.translateFaultSubscribe(translateSubRequest{
		path: "/openconfig-platform:components/component[name=PSU+0:A]/healthz/faults/fault[symptom=SYMPTOM_OVER_THRESHOLD]",
	})
	if err != nil {
		t.Fatalf("translateFaultSubscribe failed: %v", err)
	}
	if len(response.ntfAppInfoTrgt) != 1 {
		t.Fatalf("target mapping count = %d, want 1", len(response.ntfAppInfoTrgt))
	}
	info := response.ntfAppInfoTrgt[0]
	if info.table == nil || info.table.Name != "FAULT_INFO" || info.table.CompCt != 2 || info.dbno != db.StateDB {
		t.Fatalf("unexpected subscription mapping: %+v", info)
	}
	if info.key == nil || info.key.Len() != 2 ||
		info.key.Get(0) != "PSU%2B0%3AA" || info.key.Get(1) != "SYMPTOM_OVER_THRESHOLD" {
		t.Fatalf("unexpected subscription key mapping: %+v", info.key)
	}
	if !info.isOnChangeSupported || info.pType != OnChange || info.handlerFunc == nil {
		t.Fatalf("subscription is not configured for on-change: %+v", info)
	}
}

type recordingNotificationSender struct {
	notifications []*apis.Notification
}

func (s *recordingNotificationSender) Send(notification *apis.Notification) {
	s.notifications = append(s.notifications, notification)
}

func validPlatformFaultEntry() db.Value {
	return db.Value{Field: map[string]string{
		"component_type":          "PSU",
		"component_name":          "PSU0",
		"component_serial_number": "serial",
		"symptom":                 "openconfig-platform-healthz-fault:SYMPTOM_OVER_THRESHOLD",
		"status":                  "ACTIVE",
		"origin_time":             "1745614206.0123456",
		"last_detection_time":     "1745614206.987654321",
		"occurrences":             "2",
		"description":             "PSU output voltage is above threshold",
		"repair_actions":          `[{"action":"ACTION_RESEAT"}]`,
	}}
}
