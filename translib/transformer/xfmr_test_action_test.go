//go:build xfmrtest
// +build xfmrtest

package transformer_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/Azure/sonic-mgmt-common/translib"
)

func sensorResetURL(id string) string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/test-sensor-groups/test-sensor-group[id=" + id + "]/test-sensor-reset"
}

func sensorStatusURL(id string) string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/test-sensor-groups/test-sensor-group[id=" + id + "]/test-sensor-status"
}

func sensorClearURL(id string) string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/test-sensor-groups/test-sensor-group[id=" + id + "]/test-sensor-clear"
}

func sensorConfigureURL(id string) string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/test-sensor-groups/test-sensor-group[id=" + id + "]/test-sensor-configure"
}

func globalResetURL() string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/global-sensor/test-global-reset"
}

func sensorCalibrateURL(id string) string {
	return "/openconfig-test-xfmr-action:test-xfmr-action/test-sensor-groups/test-sensor-group[id=" + id + "]/openconfig-test-xfmr-action-ext:test-sensor-calibrate"
}

// Test_OC_Action_SensorReset validates the test-sensor-reset action callback:
//   - key name (id) is correctly extracted from the URI path variables
//   - input body is accepted without error
//   - output JSON contains the expected "openconfig-test-xfmr-action:output" wrapper
//   - output fields (status, message) are populated
func Test_OC_Action_SensorReset(t *testing.T) {
	const inputBody = `{"openconfig-test-xfmr-action:input":{"level":1}}`

	// --- success: valid key, with input ---
	t.Run("success with input", processActionRequest(
		sensorResetURL("GROUP1"), inputBody, "POST",
		"admin", "admin", false, false,
	))

	// --- output validation: key name appears in message ---
	t.Run("output contains key name", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    sensorResetURL("GROUP1"),
			Payload: []byte(inputBody),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		// Verify output wrapper key
		var out map[string]map[string]string
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields, ok := out["openconfig-test-xfmr-action:output"]
		if !ok {
			t.Fatalf("missing 'openconfig-test-xfmr-action:output' in response: %s", resp.Payload)
		}

		// Verify status field
		if fields["status"] != "success" {
			t.Errorf("status: got %q, want %q", fields["status"], "success")
		}

		// Verify key name (GROUP1) is reflected in message
		if !strings.Contains(fields["message"], "GROUP1") {
			t.Errorf("message %q does not contain key name GROUP1", fields["message"])
		}
	})

	// --- different key: verify key is read per-request, not cached ---
	t.Run("different key GROUP2", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    sensorResetURL("GROUP2"),
			Payload: []byte(`{"openconfig-test-xfmr-action:input":{"level":0}}`),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}
		if !strings.Contains(string(resp.Payload), "GROUP2") {
			t.Errorf("response missing key GROUP2: %s", resp.Payload)
		}
	})
}

// Test_OC_Action_NoInput validates an action that has no input definition.
// The callback should succeed with an empty body and return output.
func Test_OC_Action_NoInput(t *testing.T) {
	t.Run("empty body succeeds", processActionRequest(
		sensorStatusURL("GROUP1"), "", "POST",
		"admin", "admin", false, false,
	))

	t.Run("output contains status", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path: sensorStatusURL("GROUP1"),
			User: UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		var out map[string]map[string]string
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields, ok := out["openconfig-test-xfmr-action:output"]
		if !ok {
			t.Fatalf("missing 'openconfig-test-xfmr-action:output' in response: %s", resp.Payload)
		}
		if !strings.Contains(fields["status"], "GROUP1") {
			t.Errorf("status %q does not contain key name GROUP1", fields["status"])
		}
	})

	t.Run("different key GROUP3", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path: sensorStatusURL("GROUP3"),
			User: UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}
		if !strings.Contains(string(resp.Payload), "GROUP3") {
			t.Errorf("response missing key GROUP3: %s", resp.Payload)
		}
	})
}

// Test_OC_Action_NoOutput validates an action that has input but no output.
// The callback should succeed and return an empty (nil) payload.
func Test_OC_Action_NoOutput(t *testing.T) {
	const inputBody = `{"openconfig-test-xfmr-action:input":{"reason":"maintenance"}}`

	t.Run("succeeds with input", processActionRequest(
		sensorClearURL("GROUP1"), inputBody, "POST",
		"admin", "admin", false, false,
	))

	t.Run("response payload is empty", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    sensorClearURL("GROUP1"),
			Payload: []byte(inputBody),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}
		if len(resp.Payload) > 0 {
			t.Errorf("expected empty payload, got: %s", resp.Payload)
		}
	})

	t.Run("different key GROUP2", func(t *testing.T) {
		_, err := Action(ActionRequest{
			Path:    sensorClearURL("GROUP2"),
			Payload: []byte(`{"openconfig-test-xfmr-action:input":{"reason":"cleanup"}}`),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed for GROUP2: %v", err)
		}
	})
}

// Test_OC_Action_MultipleInputs validates an action with multiple input leaves.
// All input values (mode, interval, enabled) must appear in the output.
func Test_OC_Action_MultipleInputs(t *testing.T) {
	const inputBody = `{"openconfig-test-xfmr-action:input":{"mode":"high-precision","interval":30,"enabled":true}}`

	t.Run("succeeds with all inputs", processActionRequest(
		sensorConfigureURL("GROUP1"), inputBody, "POST",
		"admin", "admin", false, false,
	))

	t.Run("all input values echoed in output", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    sensorConfigureURL("GROUP1"),
			Payload: []byte(inputBody),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		var out map[string]map[string]interface{}
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields, ok := out["openconfig-test-xfmr-action:output"]
		if !ok {
			t.Fatalf("missing 'openconfig-test-xfmr-action:output' in response: %s", resp.Payload)
		}

		if mode, _ := fields["applied-mode"].(string); mode != "high-precision" {
			t.Errorf("applied-mode: got %q, want %q", mode, "high-precision")
		}
		if interval, _ := fields["applied-interval"].(float64); int(interval) != 30 {
			t.Errorf("applied-interval: got %v, want 30", interval)
		}
		if result, _ := fields["result"].(string); !strings.Contains(result, "GROUP1") {
			t.Errorf("result %q does not contain key GROUP1", result)
		}
	})

	t.Run("different key and values", func(t *testing.T) {
		body := `{"openconfig-test-xfmr-action:input":{"mode":"low-power","interval":60,"enabled":false}}`
		resp, err := Action(ActionRequest{
			Path:    sensorConfigureURL("GROUP2"),
			Payload: []byte(body),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		var out map[string]map[string]interface{}
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields := out["openconfig-test-xfmr-action:output"]
		if mode, _ := fields["applied-mode"].(string); mode != "low-power" {
			t.Errorf("applied-mode: got %q, want %q", mode, "low-power")
		}
		if interval, _ := fields["applied-interval"].(float64); int(interval) != 60 {
			t.Errorf("applied-interval: got %v, want 60", interval)
		}
		if !strings.Contains(string(resp.Payload), "GROUP2") {
			t.Errorf("response missing key GROUP2: %s", resp.Payload)
		}
	})
}

// Test_OC_Action_ContainerLevel validates an action defined on a container
// (not inside a list), so there are no list keys in the URI path.
func Test_OC_Action_ContainerLevel(t *testing.T) {
	const inputBody = `{"openconfig-test-xfmr-action:input":{"force":true}}`

	t.Run("succeeds without list key", processActionRequest(
		globalResetURL(), inputBody, "POST",
		"admin", "admin", false, false,
	))

	t.Run("output contains expected fields", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    globalResetURL(),
			Payload: []byte(inputBody),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		var out map[string]map[string]string
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields, ok := out["openconfig-test-xfmr-action:output"]
		if !ok {
			t.Fatalf("missing 'openconfig-test-xfmr-action:output' in response: %s", resp.Payload)
		}
		if fields["status"] != "success" {
			t.Errorf("status: got %q, want %q", fields["status"], "success")
		}
		if fields["message"] != "global sensor reset" {
			t.Errorf("message: got %q, want %q", fields["message"], "global sensor reset")
		}
	})

	t.Run("empty body succeeds", func(t *testing.T) {
		_, err := Action(ActionRequest{
			Path: globalResetURL(),
			User: UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action with empty body failed: %v", err)
		}
	})
}

// Test_OC_Action_Augmented validates an action that another module adds to a list entry with augment.
// The URI carries the augmenting module's prefix on the action name, and input/output use that module's name.
func Test_OC_Action_Augmented(t *testing.T) {
	const inputBody = `{"openconfig-test-xfmr-action-ext:input":{"offset":5}}`

	t.Run("succeeds with input", processActionRequest(
		sensorCalibrateURL("GROUP1"), inputBody, "POST",
		"admin", "admin", false, false,
	))

	t.Run("output contains key and input value", func(t *testing.T) {
		resp, err := Action(ActionRequest{
			Path:    sensorCalibrateURL("GROUP2"),
			Payload: []byte(inputBody),
			User:    UserRoles{Name: "admin", Roles: []string{"admin"}},
		})
		if err != nil {
			t.Fatalf("Action failed: %v", err)
		}

		var out map[string]map[string]string
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("unmarshal failed: %v\nraw: %s", err, resp.Payload)
		}
		fields, ok := out["openconfig-test-xfmr-action-ext:output"]
		if !ok {
			t.Fatalf("missing 'openconfig-test-xfmr-action-ext:output' in response: %s", resp.Payload)
		}
		if fields["status"] != "success" {
			t.Errorf("status: got %q, want %q", fields["status"], "success")
		}
		if want := "calibrated sensor group GROUP2 with offset 5"; fields["message"] != want {
			t.Errorf("message: got %q, want %q", fields["message"], want)
		}
	})
}
