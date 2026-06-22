package platform

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var platformCfgBackup platformConfig
var intfToDefaultModeBackup map[string]string

func savePlatformConfig() {
	platformCfgBackup = platformCfg
}

func restorePlatformConfig() {
	platformCfg = platformCfgBackup
}

func saveHwskuConfig() {
	intfToDefaultModeBackup = intfToDefaultMode
}

func restoreHwskuConfig() {
	intfToDefaultMode = intfToDefaultModeBackup
}

func loadTestJson(t *testing.T, jsonStr string) error {
	file, err := os.CreateTemp("", "platform_test_tmp-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(jsonStr)

	return doParsePlatformJson(file.Name())
}

func loadTestHwskuJson(t *testing.T, jsonStr string) error {
	file, err := os.CreateTemp("", "platform_test_tmp-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(jsonStr)

	return doParseHwskuJson(file.Name())
}

func TestMissingJsonFile(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	err := doParsePlatformJson("/path/to/nonexistent/file")
	if err == nil {
		t.Errorf("No error generated for missing platform.json file")
	}
}

func TestParseThisThenThatFallback(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	platJson :=
		`{
			"interfaces": {
				"test1/12/1": {
					"lanes": "80,81,82,83,84,85,86,87",
					"index": "12,12,12,12,12,12,12,12",
					"default_brkout_mode": "2x800G",
					"breakout_modes": "2x800G, 4x400G, 8x200G[100G,50G], 1x800G(4)+2x400G(4), 1x800G(4)+4x200G[100G,50G](4), 4x200G[100G,50G](4)+2x400G(4)",
					"alias_at_lanes": "Tst12/1, Tst12/2, Tst12/3, Tst12/4, Tst12/5, Tst12/6, Tst12/7, Tst12/8"
				}
			}
		}`
	file, err := os.CreateTemp("", "platform_test_tmp-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(platJson)

	if err := parseThisThenThat("/path/to/nonexistent/file", file.Name()); err != nil {
		t.Errorf("Unexpected error parsing platform json: %v", err)
	}
}

func TestMalformedHwskuJsonFile(t *testing.T) {
	saveHwskuConfig()
	defer restoreHwskuConfig()
	// Missing comma creates an invalid json
	if err := loadTestHwskuJson(t, `{"interfaces": {"eth1":{} "eth2":{} }}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}

	// interfaces should be a dictionary
	if err := loadTestHwskuJson(t, `{"interfaces": 1}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}

	// interface should be a dictionary
	if err := loadTestHwskuJson(t, `{"interfaces": { "eth1": 1 }}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}

	// interface dictionary key should be a string
	if err := loadTestHwskuJson(t, `{"interfaces": { "eth1": { 1: 2} }}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}
}

func TestMalformedJsonFile(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	// Missing comma creates an invalid json
	if err := loadTestJson(t, `{"interfaces": {"eth1":{} "eth2":{} }}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}

	// interfaces should be a dictionary
	if err := loadTestJson(t, `{"interfaces": {1: 2}}`); err == nil {
		t.Errorf("No error generated for malformed platform.json file")
	}
}

func TestMalformedIntfName(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Interface naming is required to be <prefix>1/<index> or <prefix>1/<index>/<subindex>
		`{"interfaces": {"eth1":{}, "eth2":{} }}`,
		// Interface index must be an integer
		`{"interfaces": {"ethernetA/B/C":{} }}`,
		// Interface subindex must be an integer
		`{"interfaces": {"ethernet1/2/C":{} }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestMalformedIndex(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Index should not be empty
		`{"interfaces": {"ethernet1/7/1":{ "index":"" } }}`,
		`{"interfaces": {"ethernet1/7/1":{ "xedni":"a" } }}`,
		// Index should be an integer
		`{"interfaces": {"ethernet1/7/1":{ "index":"NotAnInteger" } }}`,
		// Index should match the name
		`{"interfaces": {"ethernet1/7/1":{ "index":"12345" } }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestMalformedBrkoutMode(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Breakout mode should not be empty
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"" } }}`,
		`{"interfaces": {"ethernet1/7/1":{ "index":"7" } }}`,
		// Default mode should be sane if it is present
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G", "default_brkout_mode":"nonsense mode" } }}`,
		// Default mode should be a single speed if it is present
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G[200G,100G]", "default_brkout_mode":"1x400G[200G]" } }}`,
		// Modes should be sane
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G, 2x400G, 2xOneHundredG" } }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestMalformedLanes(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Lanes should be integers
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"a,b,c" } }}`,
		// Lanes should be have empty entries
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"1,2,,3" } }}`,
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"1,2,3," } }}`,
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G", "lanes":",1,2,3" } }}`,
		// Should have enough lanes for all children
		`{"interfaces": {"ethernet1/7/1":{ "index":"7,7,7", "breakout_modes":"1x400G", "lanes": "12,13", "alias_at_lanes": "foo,bar,baz"},
		                 "ethernet1/7/2":{ "index":"7", "breakout_modes":"1x400G"},
		                 "ethernet1/7/3":{ "index":"7", "breakout_modes":"1x400G"} }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestMalformedPrimary(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Primary intf missing
		`{"interfaces": {"ethernet1/7/2":{ "index":"7", "breakout_modes":"1x400G" } }}`,
		// Primary intf isn't a primary
		`{"interfaces": {"ethernet1/7/1":{ "index":"7", "breakout_modes":"1x400G" }, "ethernet1/7/2":{ "index":"7", "breakout_modes":"1x400G" }}}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestMalformedAlias(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	badJsons := []string{
		// Subindex references bad alias
		`{"interfaces": {"ethernet1/7/1":{ "index":"7,7", "breakout_modes":"1x400G", "lanes": "12,13", "alias_at_lanes": "foo, bar" },
		                 "ethernet1/7/100":{ "index":"7", "breakout_modes":"1x400G"  } }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestSanitizeBreakoutMode(t *testing.T) {
	good := []string{"1x400G", "2x800G[400G]", "8x200G[100G,50G]", "4x800G[100G,50G,40G]", "2x100G(2)+6x200G(6)", "2x200G[100G,50G](2)+6x200G(6)", "2x200G[100G,50G](2)+6x200G[100G](6)", "3x200G(6)+1x800G[400G](2)"}
	bad_single := []string{"1X400G", "1x400g", "400G", "ax400G", "x400G", "1x4O0G", "1x[400G,100G]", "4x800G[100G, 50G, 40G]", "1x400G[]", "1x400G[", "1x400G]", "1x400G(100G,200G)", "1x400G[200G 100G]", "1x400G[200, 100]", "1x400G+1x200G"}
	bad_mixed := []string{"1x400G+1x200G", "2x400G[200G]+4x100G", "2x400G+4x200G[100G]", "2x400G[200G,100G]+4x200G[100G]", "1x400G(a)+1x200G(4)", "1x400G(2)+1x200G)4("}
	bad := append(bad_single, bad_mixed...)

	for _, mode := range good {
		if _, ok := sanitizeBreakoutMode(mode); !ok {
			t.Errorf("Error generated for valid mode: \"%s\"", mode)
		}
	}
	for _, mode := range bad {
		if _, ok := sanitizeBreakoutMode(mode); ok {
			t.Errorf("No error generated for invalid mode: \"%s\"", mode)
		}
	}
}

func TestBrkoutModeToSpeeds(t *testing.T) {
	var cases = []struct {
		mode   string
		speeds []int
	}{
		{mode: "1x400G", speeds: []int{400}},
		{mode: "2x400G[200G,100G]", speeds: []int{100, 200, 400}},
		{mode: "4x200G[100G]", speeds: []int{100, 200}},
		{mode: "1x400G(4)+2x200G(4)", speeds: []int{200, 400}},
		{mode: "2x200G(4)+1x400G(4)", speeds: []int{200, 400}},
		{mode: "8x100G", speeds: []int{100}},
		{mode: "2x800G[400G,200G](4)+1x400G(4)", speeds: []int{200, 400, 800}},
	}
	for _, c := range cases {
		speeds, err := brkoutModeToSpeeds(c.mode)
		if err != nil {
			t.Errorf("Unexpected error %v for mode %s", err, c.mode)
		}
		if len(speeds) != len(c.speeds) {
			t.Errorf("Unexpected speeds %v for mode %s", speeds, c.mode)
		}
		for i, _ := range speeds {
			if speeds[i] != c.speeds[i] {
				t.Errorf("Unexpected speeds %v for mode %s", speeds, c.mode)
			}
		}
	}
}

func TestPlatIntfAlias(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()

	platJson := `{
		"interfaces": {
			"Ethernet1/1/1": {
				"index": "1,1,1,1,1,1,1,1",
				"lanes": "9,10,11,12,13,14,15,16",
				"breakout_modes": "8x100G",
				"alias_at_lanes": "Eth1/1, Eth1/2, Eth1/3, Eth1/4, Eth1/5, Eth1/6, Eth1/7, Eth1/8"
			},
			"Ethernet1/1/2": { "index": "1" },
			"Ethernet1/1/3": { "index": "1" },
			"Ethernet1/1/4": { "index": "1" },
			"Ethernet1/1/5": { "index": "1" },
			"Ethernet1/1/6": { "index": "1" },
			"Ethernet1/1/7": { "index": "1" },
			"Ethernet1/1/8": { "index": "1" }
		}
	}`

	if err := loadTestJson(t, platJson); err != nil {
		t.Fatalf("Error loading test json: %v", err)
	}

	for i := 1; i <= 8; i++ {
		intf := fmt.Sprintf("Ethernet1/1/%d", i)
		expectedAlias := fmt.Sprintf("Eth1/%d", i)
		pIntf, err := platIntfByName(intf)
		if err != nil {
			t.Errorf("Unexpected error %v for interface %s", err, intf)
		}
		if pIntf.alias != expectedAlias {
			t.Errorf("Unexpected alias for interface %s (%#v)", intf, pIntf)
		}
	}
}

func laneSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDoParsePlatformJsonSonic_Success(t *testing.T) {
	platformCfg.intfs = make(map[string]*platformIntf)
	intfToDefaultMode = map[string]string{
		"Ethernet1/1/1": "8x100G",
	}

	inputJson := map[string]interface{}{
		"Ethernet1/1/1": map[string]interface{}{
			"index": "1,1,1,1,1,1,1,1",
			"lanes": "9,10,11,12,13,14,15,16",
			"breakout_modes": map[string]interface{}{
				"8x100G": []interface{}{
					"Ethernet1/1/1", "Ethernet1/1/2", "Ethernet1/1/3", "Ethernet1/1/4",
					"Ethernet1/1/5", "Ethernet1/1/6", "Ethernet1/1/7", "Ethernet1/1/8",
				},
			},
		},
	}

	err := doParsePlatformJsonSonic(inputJson)
	if err != nil {
		t.Fatalf("Expected doParsePlatformJsonSonic to succeed, got error: %v", err)
	}

	intf1, ok := platformCfg.intfs["Ethernet1/1/1"]
	if !ok {
		t.Fatalf("Expected interface 'Ethernet1/1/1' to be populated in platformCfg")
	}
	if !intf1.isPrimary {
		t.Errorf("Expected Ethernet1/1/1 to be marked as primary")
	}
	if intf1.dfltMode != "8x100G" {
		t.Errorf("Expected default mode to be '8x100G', got '%s'", intf1.dfltMode)
	}

	intf2, ok := platformCfg.intfs["Ethernet1/1/2"]
	if !ok {
		t.Fatalf("Expected child interface 'Ethernet1/1/2' to be populated")
	}
	if intf2.isPrimary {
		t.Errorf("Expected Ethernet1/1/2 to NOT be marked as primary")
	}
	if intf2.primary != intf1 {
		t.Errorf("Expected child interface primary reference to link back to parent entry block")
	}

	portName, ok := platformCfg.intfNameToPortName["Ethernet1/1/1"]
	if !ok || portName != "1/1" {
		t.Errorf("Expected map translation registration to match '1/1', got '%s'", portName)
	}
}

func TestDoParsePlatformJsonSonic_MalformedName(t *testing.T) {
	platformCfg.intfs = make(map[string]*platformIntf)
	inputJson := map[string]interface{}{
		"InvalidNameOnly": map[string]interface{}{
			"index": "1",
			"lanes": "1",
			"breakout_modes": map[string]interface{}{
				"1x100G": []interface{}{"InvalidNameOnly"},
			},
		},
	}

	err := doParsePlatformJsonSonic(inputJson)
	if err == nil {
		t.Errorf("Expected failure error on an interface string with an invalid token forward-slash layout pattern")
	}
}

func TestChannelOffset(t *testing.T) {
	platformCfg.intfs = make(map[string]*platformIntf)

	targetIntf := "Ethernet1/1/1"
	var expectedOffset uint16 = 3

	platformCfg.intfs[targetIntf] = &platformIntf{
		name:          targetIntf,
		channelOffset: expectedOffset,
	}

	t.Run("ValidInterface", func(t *testing.T) {
		offset, err := ChannelOffset(targetIntf)
		if err != nil {
			t.Errorf("Unexpected error for %s: %v", targetIntf, err)
		}
		if offset != expectedOffset {
			t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
		}
	})

	t.Run("InvalidInterface", func(t *testing.T) {
		missingIntf := "Ethernet1/99/1"
		_, err := ChannelOffset(missingIntf)
		if err == nil {
			t.Errorf("Expected an error for non-existent interface %s, but got nil", missingIntf)
		}
	})
}

func TestHwskuParsing(t *testing.T) {
	saveHwskuConfig()
	defer restoreHwskuConfig()
	hwskuJson :=
		`{
			"interfaces": {
				"Ethernet1/1/1": {
					"default_brkout_mode": "1x800G"
				},
				"Ethernet1/2/1": {
					"default_brkout_mode": "2x800G"
				},
				"Ethernet1/3/1": {
					"default_brkout_mode": "3x800G"
				},
				"Ethernet1/4/1": {
					"default_brkout_mode": "4x800G"
				}
			}
		}`
	if err := loadTestHwskuJson(t, hwskuJson); err != nil {
		t.Fatalf("Error %v loading hwsku json", err)
	}

	expected := map[string]string{
		"Ethernet1/1/1": "1x800G",
		"Ethernet1/2/1": "2x800G",
		"Ethernet1/3/1": "3x800G",
		"Ethernet1/4/1": "4x800G",
	}
	if diff := cmp.Diff(expected, intfToDefaultMode); diff != "" {
		t.Errorf("unexpected diff (-want +got):\n%s", diff)
	}
}
