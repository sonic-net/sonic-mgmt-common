package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var platformCfgBackup platformConfig

func savePlatformConfig() {
	platformCfgBackup = platformCfg
}

func restorePlatformConfig() {
	platformCfg = platformCfgBackup
}

func resetState() {
	once = sync.Once{}
	initErr = nil
	platformCfg = platformConfig{
		intfs: make(map[string]*platformIntf),
	}
}

func loadTestJson(t *testing.T, jsonStr string) error {
	file, err := os.CreateTemp("", "platform_test_tmp-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(jsonStr)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	return doParsePlatformJson(file.Name())
}

func TestMissingJsonFile(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()
	err := doParsePlatformJson("/path/to/nonexistent/file")
	if err == nil {
		t.Errorf("No error generated for missing platform.json file")
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
		`{"interfaces": {"ethernet1":{ "index":"" } }}`,
		`{"interfaces": {"ethernet1":{ "xedni":"a" } }}`,
		// Index should be an integer
		`{"interfaces": {"ethernet1":{ "index":"NotAnInteger" } }}`,
		// Index should match the name
		`{"interfaces": {"ethernet1":{ "index":"12345" } }}`,
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
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"a,b,c" } }}`,
		// Lanes should be have empty entries
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"1,2,,3" } }}`,
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G", "lanes":"1,2,3," } }}`,
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G", "lanes":",1,2,3" } }}`,
		// Should have enough lanes for all children
		`{"interfaces": {"ethernet1":{ "index":"7,7,7", "breakout_modes":"1x400G", "lanes": "12,13", "alias_at_lanes": "foo,bar,baz"},
		                 "ethernet1":{ "index":"7", "breakout_modes":"1x400G"},
		                 "ethernet1":{ "index":"7", "breakout_modes":"1x400G"} }}`,
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
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G" } }}`,
		// Primary intf isn't a primary
		`{"interfaces": {"ethernet1":{ "index":"7", "breakout_modes":"1x400G" }, "ethernet1":{ "index":"7", "breakout_modes":"1x400G" }}}`,
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
		`{"interfaces": {"ethernet100":{ "index":"7", "breakout_modes":"1x400G"  } }}`,
	}

	for _, badJson := range badJsons {
		if err := loadTestJson(t, badJson); err == nil {
			t.Errorf("No error generated for malformed interface case: \"%s\"", badJson)
		}
	}
}

func TestPlatIntfAlias(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()

	platJson := `{
		"interfaces": {
			"Ethernet1": {
				"index": "1,1,1,1,1,1,1,1",
				"lanes": "9,10,11,12,13,14,15,16",
				"breakout_modes": "8x100G",
				"alias_at_lanes": "Eth1/1, Eth1/2, Eth1/3, Eth1/4, Eth1/5, Eth1/6, Eth1/7, Eth1/8"
			}
		}
	}`

	if err := loadTestJson(t, platJson); err != nil {
		t.Fatalf("Error loading test json: %v", err)
	}

	for i := 1; i <= 1; i++ {
		intf := fmt.Sprintf("Ethernet%d", i)
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

func TestCalcChannelOffset(t *testing.T) {
	platformCfg.intfs = make(map[string]*platformIntf)

	intfName := "Ethernet3333"
	platformCfg.intfs[intfName] = &platformIntf{
		name:          intfName,
		isPrimary:     true,
		lanes:         []int{},
		channelOffset: 99,
	}

	calcChannelOffset()

	actualOffset := platformCfg.intfs[intfName].channelOffset
	if actualOffset != 99 {
		t.Errorf("Expected channelOffset to remain 99 for zero-lane interface, but got %d", actualOffset)
	}
}

func resetGlobalState() {
	once = sync.Once{}
	initErr = nil
	platformCfg = platformConfig{
		intfs:              make(map[string]*platformIntf),
		intfNameToPortName: make(map[string]string),
		portNameToIntfName: make(map[string]string),
	}
}

func TestEnsureLoaded(t *testing.T) {
	//Create a temporary platform.json file
	tmpFile := filepath.Join(t.TempDir(), "platform.json")

	// Ethernet0: lanes 25,26,27,28,29,30,31,32. Offset = 25 % 4 = 1
	// Ethernet4: lanes 33,34,35,36,37,38,39,40. Offset = 29 % 4 = 1
	mockData := `{
		"interfaces": {
			"Ethernet0": {
				"index": "0,0,0,0,0,0,0,0",
				"lanes": "25,26,27,28,29,30,31,32",
				"alias_at_lanes": "Eth0"
			},
			"Ethernet4": {
				"index": "1,1,1,1,1,1,1,1",
				"lanes": "33,34,35,36,37,38,39,40",
				"alias_at_lanes": "Eth4"
			}
		}
	}`

	if err := os.WriteFile(tmpFile, []byte(mockData), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	//Override the global path variable
	oldPath := PLATFORM_JSON_PATH
	PLATFORM_JSON_PATH = tmpFile
	defer func() { PLATFORM_JSON_PATH = oldPath }() // Restore after test

	// --- Test Case: ensureLoaded ---
	t.Run("TestEnsureLoaded", func(t *testing.T) {
		resetGlobalState() // Crucial: forces a fresh execution of once.Do

		err := ensureLoaded()
		if err != nil {
			t.Fatalf("ensureLoaded returned unexpected error: %v", err)
		}

		if len(platformCfg.intfs) != 2 {
			t.Errorf("Expected 2 interfaces in map, got %d", len(platformCfg.intfs))
		}
	})

	// --- Test Case: ChannelOffset ---
	t.Run("TestChannelOffset", func(t *testing.T) {
		resetGlobalState() // Ensure clean state for this calculation

		// Test for Ethernet0
		// Math: lanes[0] (25) % len(lanes) (4) = 1
		expected := uint16(1)
		val, err := ChannelOffset("Ethernet0")

		if err != nil {
			t.Fatalf("ChannelOffset failed: %v", err)
		}
		if val != expected {
			t.Errorf("Ethernet0 offset mismatch: expected %d, got %d", expected, val)
		}

		// Test for Ethernet4
		// Math: lanes[0] (29) % len(lanes) (4) = 1
		val4, err := ChannelOffset("Ethernet4")
		if err != nil {
			t.Fatalf("ChannelOffset failed for Ethernet4: %v", err)
		}
		if val4 != expected {
			t.Errorf("Ethernet4 offset mismatch: expected %d, got %d", expected, val4)
		}
	})
}

func TestAliasFallbackLogic(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()

	// Scenario 1: Field is missing entirely
	t.Run("MissingAliasField", func(t *testing.T) {
		platJson := `{
			"interfaces": {
				"Ethernet100": {
					"index": "100",
					"lanes": "1"
				}
			}
		}`
		if err := loadTestJson(t, platJson); err != nil {
			t.Fatalf("Error loading test json: %v", err)
		}

		pIntf, err := platIntfByName("Ethernet100")
		if err != nil {
			t.Fatalf("Interface not found: %v", err)
		}

		// Validation: Alias should match the Name
		if pIntf.alias != "Ethernet100" {
			t.Errorf("Expected alias to fallback to name 'Ethernet100', but got '%s'", pIntf.alias)
		}
	})

	// Scenario 2: Field exists but is an empty string
	t.Run("EmptyAliasString", func(t *testing.T) {
		platJson := `{
			"interfaces": {
				"Ethernet200": {
					"index": "200",
					"lanes": "1",
					"alias_at_lanes": ""
				}
			}
		}`
		if err := loadTestJson(t, platJson); err != nil {
			t.Fatalf("Error loading test json: %v", err)
		}

		pIntf, _ := platIntfByName("Ethernet200")

		// Validation: Alias should match the Name
		if pIntf.alias != "Ethernet200" {
			t.Errorf("Expected alias to fallback to name 'Ethernet200' for empty string, but got '%s'", pIntf.alias)
		}
	})
}

func TestPrimaryLinking(t *testing.T) {
	savePlatformConfig()
	defer restorePlatformConfig()

	// Ethernet0 is Primary (has lanes)
	// Ethernet0_Child is Secondary (missing lanes, but same index "0")
	platJson := `{
		"interfaces": {
			"Ethernet0": {
				"index": "0",
				"lanes": "1,2,3,4"
			},
			"Ethernet0_Child": {
				"index": "0"
			}
		}
	}`

	if err := loadTestJson(t, platJson); err != nil {
		t.Fatalf("Error loading test json: %v", err)
	}

	// 1. Verify Ethernet0_Child was parsed
	child, err := platIntfByName("Ethernet0_Child")
	if err != nil {
		t.Fatalf("Secondary interface not found: %v", err)
	}

	// 2. Verify it is NOT a primary
	if child.isPrimary {
		t.Error("Ethernet0_Child should not be a primary interface")
	}

	// 3. Verify the linking logic (This covers the requested lines)
	if child.primary == nil {
		t.Fatal("Linking failed: Ethernet0_Child.primary is nil")
	}

	if child.primary.name != "Ethernet0" {
		t.Errorf("Linking mismatch: Expected primary to be 'Ethernet0', got '%s'", child.primary.name)
	}

	t.Logf("SUCCESS: Ethernet0_Child correctly linked to primary %s", child.primary.name)
}
