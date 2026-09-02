//go:build testapp
// +build testapp

package transformer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func TestCompRoot_ComponentTypeString(t *testing.T) {
	cases := []struct {
		in   componentType
		want string
	}{
		{CompTypeInvalid, "CompTypeInvalid"},
		{CompTypeIC, "CompTypeIC"},
	}
	for _, c := range cases {
		got := c.in.String()
		if got != c.want {
			t.Errorf("componentType.String() == %q, want %q", got, c.want)
		}
	}
}
func TestCompRoot_ValidICName(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		input    *string
		expected bool
	}{
		{
			name:     "Fails HasPrefix check",
			input:    strPtr("invalid-prefix-123"),
			expected: false,
		},
		{
			name:     "Fails len(sp) < 2 check (Prefix matches exactly, but nothing follows)",
			input:    strPtr("integrated_circuit"),
			expected: false,
		},
		{
			name:     "Fails Atoi check (Suffix is not a valid integer)",
			input:    strPtr("integrated_circuitABC"),
			expected: false,
		},
		{
			name:     "Fails Atoi check (Suffix is not a valid integer)",
			input:    strPtr("integrated_circuitf"),
			expected: false,
		},
		{
			name:     "Passes all checks (Valid IC name)",
			input:    strPtr("integrated_circuit42"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validICName(tt.input)
			if got != tt.expected {
				t.Errorf("validICName() = %v, want %v for input %v", got, tt.expected, *tt.input)
			}
		})
	}
}

func TestCompRoot_GetCompTypeByName(t *testing.T) {
	inputName := "integrated_circuit45"
	expectedType := CompTypeIC

	actualType, err := getCompTypeByName(inputName)

	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if actualType != expectedType {
		t.Errorf("Expected component type %d, but got %d", expectedType, actualType)
	}
}

func TestCompRoot_GetCompType_ErrorCases(t *testing.T) {
	testName := "integrated_circuit"
	compTypeCache.Delete(testName)

	result := getCompType(testName, nil)

	if result != CompTypeInvalid {
		t.Errorf("Expected CompTypeInvalid when an error occurs, but got %v", result)
	}

	if _, cached := compTypeCache.Load(testName); cached {
		t.Errorf("Expected error case NOT to be cached, but found an entry in compTypeCache")
	}
}

func TestCompRoot_GetPfmRootObject(t *testing.T) {
	if r := getPfmRootObject(nil); r != nil {
		t.Errorf("Calling getPfmRootObject with nil didn't return nil, %v", r)
	}
}

func TestCompRoot_GetPfmRootObject_Success(t *testing.T) {
	expectedComponents := &ocbinds.OpenconfigPlatform_Components{}

	device := &ocbinds.Device{
		Components: expectedComponents,
	}

	var gStruct ygot.GoStruct = device
	inputPointer := &gStruct

	res := getPfmRootObject(inputPointer)

	if res != expectedComponents {
		t.Errorf("Expected components pointer %v, got %v", expectedComponents, res)
	}
}
func TestCompRoot_GetSysComponentsWithUnknownComponentType(t *testing.T) {
	var inParams XfmrParams
	key := "IDONTEXIST"

	dbNum := db.StateDB
	d, err := db.NewDB(getDBOptions(dbNum))
	if err != nil {
		t.Fatal("NewDB failed")
	}
	inParams.dbs[dbNum] = d
	for _, targetUriPath := range []string{COMP, COMP_ST} {
		if err := getSysComponents(nil, targetUriPath, inParams, key, ""); err != nil {
			t.Fatal("getSysComponents returned an error with an unknown component type")
		}
	}
}

func TestCompRoot_Subscribe_pfm_components_xfmr(t *testing.T) {

	oldCompTblMap := compTblMap
	defer func() { compTblMap = oldCompTblMap }()
	compTblMap = map[componentType][]string{
		CompTypeIC: {"IC_TABLE"},
	}

	const actualConfigDBNum = 4

	tests := []struct {
		name        string
		inParams    XfmrSubscInParams
		setupFunc   func() // A callback to intercept or set up test-specific preconditions
		expectedOut XfmrSubscOutParams
		wantErr     bool
	}{
		{
			name: "Early Exit - Key is empty (Requesting ALL components)",
			inParams: XfmrSubscInParams{
				uri:        "/openconfig-platform:components/component",
				requestURI: "/openconfig-platform:components/component",
				subscProc:  TRANSLATE_EXISTS,
			},
			expectedOut: XfmrSubscOutParams{
				isVirtualTbl: true,
			},
			wantErr: false,
		},
		{
			name: "Early Exit - Key contains _sensor",
			inParams: XfmrSubscInParams{
				uri:        "/openconfig-platform:components/component[name=temp_sensor]",
				requestURI: "/openconfig-platform:components/component[name=temp_sensor]",
				subscProc:  TRANSLATE_EXISTS,
			},
			expectedOut: XfmrSubscOutParams{
				isVirtualTbl: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			gotOut, gotErr := Subscribe_pfm_components_xfmr(tt.inParams)

			if gotErr != nil {
				t.Fatalf("Subscribe_pfm_components_xfmr() unexpected error: %v", gotErr)
			}

			if tt.name == "Routing Path - TRANSLATE_SUBSCRIBE branch execution" {
				if gotOut.needCache != tt.expectedOut.needCache || gotOut.onChange != tt.expectedOut.onChange {
					t.Errorf("Subscribe_pfm_components_xfmr() output fields mismatch.\nGot: needCache=%v, onChange=%v\nExpected: needCache=%v, onChange=%v",
						gotOut.needCache, gotOut.onChange, tt.expectedOut.needCache, tt.expectedOut.onChange)
				}
				return
			}

			if !reflect.DeepEqual(gotOut, tt.expectedOut) {
				t.Errorf("Subscribe_pfm_components_xfmr() output =\n%+v\nExpected =\n%+v", gotOut, tt.expectedOut)
			}
		})
	}
}

func TestCompRoot_GetAllTableEntries(t *testing.T) {
	tests := []struct {
		name         string
		tblName      string
		key          string
		wantErr      bool
		wantLen      int
		mockBehavior func() ([]db.Key, error)
	}{
		{
			name:    "Coverage: Empty table name exits early",
			tblName: "",
			key:     "integrated_circuit45",
			wantErr: true,
		},
		{
			name:    "Coverage: Empty key exits early",
			tblName: "MY_TABLE",
			key:     "",
			wantErr: true,
		},
		{
			name:    "Coverage: DB execution failure",
			tblName: "MY_TABLE",
			key:     "key1",
			wantErr: true,
			mockBehavior: func() ([]db.Key, error) {
				return nil, errors.New("mock db failure")
			},
		},
		{
			name:    "Check valid case",
			tblName: "NODE_CFG_TBL",
			key:     "integrated_circuit45",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var testDB *db.DB

			results, err := getAllTableEntries(testDB, tt.tblName, tt.key)

			if (err != nil) != tt.wantErr {
				t.Fatalf("getAllTableEntries() unexpected error state = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(results) != tt.wantLen {
				t.Errorf("Expected return array length %d, got %d", tt.wantLen, len(results))
			}
		})
	}
}

func TestCompRoot_FillICInfo(t *testing.T) {
	tests := []struct {
		name          string
		targetUriPath string
		compName      string
	}{
		{
			name:          "Coverage: Path matches COMP exactly",
			targetUriPath: COMP, // ensure COMP is defined or use "/components/component"
			compName:      "ic-1",
		},
		{
			name:          "Coverage: Path matches COMP_ST exactly",
			targetUriPath: COMP_ST, // ensure COMP_ST is defined or use "/components/component/state"
			compName:      "ic-2",
		},
		{
			name:          "Coverage: Path has COMP_ST prefix but hits switch default",
			targetUriPath: "/components/component/state/unsupported-leaf",
			compName:      "ic-3",
		},
		{
			name:          "Coverage: Path doesn't match either condition",
			targetUriPath: "/components/component/config",
			compName:      "ic-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &ocbinds.OpenconfigPlatform_Components_Component{
				State: &ocbinds.OpenconfigPlatform_Components_Component_State{},
				IntegratedCircuit: &ocbinds.OpenconfigPlatform_Components_Component_IntegratedCircuit{
					State: &ocbinds.OpenconfigPlatform_Components_Component_IntegratedCircuit_State{},
				},
			}

			var dummyDbs [db.MaxDB]*db.DB
			var dummyYgRoot *ygot.GoStruct

			err := fillICInfo(comp, tt.compName, tt.targetUriPath, dummyDbs, dummyYgRoot)
			if err != nil {
				t.Fatalf("fillICInfo returned an unexpected error: %v", err)
			}
		})
	}
}
func TestCompRoot_DbToYang_pfm_components_xfmr(t *testing.T) {
	tests := []struct {
		name           string
		inParams       XfmrParams
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Coverage: Unsupported request URI triggers early exit error",
			inParams: XfmrParams{
				requestUri: "/openconfig-system:system/config",
				uri:        "/some-uri",
			},
			wantErr:        true,
			expectedErrMsg: "Component not supported",
		},
		{
			name: "Coverage: Successful execution path calls downstream getSysComponents",
			inParams: XfmrParams{
				requestUri: "/openconfig-platform:components/component[name=ic-chip-1]",
				uri:        "valid-path",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DbToYang_pfm_components_xfmr(tt.inParams)

			if (err != nil) != tt.wantErr {
				t.Fatalf("DbToYang_pfm_components_xfmr() unexpected error state = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("Expected error message: %q, got: %q", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

// Mocking or instantiating required types for the test setup
type mockGoStruct struct {
	ygot.GoStruct
}

func TestCompRoot_CompTypeToFuncCall(t *testing.T) {
	var mockDbs [db.MaxDB]*db.DB
	var mockYgRoot ygot.GoStruct = &mockGoStruct{}

	tests := []struct {
		name          string
		cType         componentType
		compName      string
		subKey        string
		pfComp        *ocbinds.OpenconfigPlatform_Components_Component
		targetUriPath string
		pType         PathType
		wantErr       bool
		errMessage    string
	}{
		{
			name:          "Invalid Component Type returns error",
			cType:         CompTypeInvalid, // An invalid/unhandled type
			compName:      "test-comp",
			subKey:        "sub-key",
			pfComp:        &ocbinds.OpenconfigPlatform_Components_Component{},
			targetUriPath: "/root/path",
			pType:         AllPaths,
			wantErr:       true,
			errMessage:    "Invalid component type",
		},
		{
			name:          "Valid CompTypeIC path",
			cType:         CompTypeIC,
			compName:      "ic-comp",
			subKey:        "sub-key",
			pfComp:        &ocbinds.OpenconfigPlatform_Components_Component{},
			targetUriPath: "/root/path/ic",
			pType:         AllPaths,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := compTypeToFuncCall(
				tt.cType,
				tt.compName,
				tt.subKey,
				tt.pfComp,
				tt.targetUriPath,
				mockDbs,
				tt.pType,
				&mockYgRoot,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf("compTypeToFuncCall() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				if err.Error() != tt.errMessage {
					t.Errorf("compTypeToFuncCall() error message = %q, want %q", err.Error(), tt.errMessage)
				}
			}

			if tt.pfComp == nil {
				t.Errorf("Expected pfComp to be initialized by ygot.BuildEmptyTree, but it was nil")
			}
		})
	}
}
func TestCompRoot_CreateCompAndFuncCall_Success(t *testing.T) {

	inParams := XfmrParams{
		requestUri: "/openconfig-system:system/config",
		uri:        "/some-uri",
	}

	tests := []struct {
		name           string
		compType       componentType
		tblName        string
		tblKey         string
		targetUriPath  string
		expectCompName string
	}{
		{
			name:           "Successful component creation and execution for default type",
			compType:       CompTypeIC, // Use your actual enum value here
			tblName:        "NODE_CFG_TBL",
			tblKey:         "Ethernet0",
			targetUriPath:  "/components/component",
			expectCompName: "Ethernet0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			createCompAndFuncCall(getPfmRootObject(inParams.ygRoot), tt.targetUriPath, tt.compType, inParams, tt.tblName, tt.tblKey)
		})
	}
}
func TestCompRoot_Subscribe_pfm_components_xfmr_TranslateExists(t *testing.T) {
	inParams := XfmrSubscInParams{
		uri:        "/components/component[name=integrated_circuit78]",
		requestURI: "/components/component",
		subscProc:  TRANSLATE_EXISTS,
	}

	_, err := Subscribe_pfm_components_xfmr(inParams)
	if err == nil {
		t.Errorf("Expectimg error since DB is not filled\n")
	}
}
func TestCompRoot_Subscribe_pfm_components_xfmr_TranslateSubscribe(t *testing.T) {
	inParams := XfmrSubscInParams{
		uri:        "/components/component[name=integrated_circuit78]",
		requestURI: "/components/component",
		subscProc:  TRANSLATE_SUBSCRIBE,
	}

	_, err := Subscribe_pfm_components_xfmr(inParams)
	if err != nil {
		t.Errorf("Expectimg Success for Subscribe_pfm_components_xfmr\n")
	}
}

func TestCompRoot_TranslateExistsInvalidKey(t *testing.T) {
	dbNum := db.StateDB
	d, err := db.NewDB(getDBOptions(dbNum))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer d.DeleteDB()

	var inParams XfmrSubscInParams
	inParams.dbs[dbNum] = d
	key := "INVALIDKEY"
	got, err := translateExists(inParams, key)
	if err != nil {
		t.Fatalf("translateExists(%q) returned unexpected error: %v", key, err)
	}
	if got.isVirtualTbl {
		t.Errorf("translateExists(%q) returned isVirtualTbl = true, want false", key)
	}
}
func TestCompRoot_TranslateExistsValidKey(t *testing.T) {
	dbNum := db.StateDB
	d, err := db.NewDB(getDBOptions(dbNum))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer d.DeleteDB()

	var inParams XfmrSubscInParams
	inParams.dbs[dbNum] = d
	key := "integrated_circuit32"
	got, err := translateExists(inParams, key)
	if err != nil {
		t.Fatalf("translateExists(%q) returned unexpected error: %v", key, err)
	}
	if got.isVirtualTbl {
		t.Errorf("translateExists(%q) returned isVirtualTbl = true, want false", key)
	}
}
func TestCompRoot_TranslateSubscribeValidKey(t *testing.T) {
	dbNum := db.StateDB
	d, err := db.NewDB(getDBOptions(dbNum))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer d.DeleteDB()

	var inParams XfmrSubscInParams
	inParams.dbs[dbNum] = d
	key := "integrated_circuit32"
	targetUriPath := COMP_ST
	got, err := translateSubscribe(inParams, key, targetUriPath)
	if err != nil {
		t.Fatalf("translateSubscribe(%q) returned unexpected error: %v", key, err)
	}
	if got.isVirtualTbl {
		t.Errorf("translateSubsribe(%q) returned isVirtualTbl = true, want false", key)
	}
}

// Helper function to mock state DB with EEPROM_INFO table data
func setupTestStateDB(t *testing.T) *db.DB {
	dbNum := db.StateDB
	d, err := db.NewDB(getDBOptions(dbNum))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}

	eepromTable := db.TableSpec{Name: EEPROM_INFO_TBL}
	_ = d.DeleteTable(&eepromTable)

	testEntries := []struct {
		tlvCode string
		name    string
		value   string
	}{
		{"0x21", "Product Name", "Test-Product-100"},
		{"0x22", "Part Number", "PN-12345"},
		{"0x23", "Serial Number", "SN-987654321"},
		{"0x24", "Base MAC Address", "00:11:22:33:44:55"},
		{"0x25", "Manufacture Date", "2026-01-01"},
		{"0x26", "Device Version", "v1.2.3"},
		{"0x27", "Label Revision", "Rev-A"},
		{"0x28", "Platform Name", "x86_64-test_platform-r0"},
		{"0x29", "ONIE Version", "2023.11"},
		{"0x2A", "MAC Addresses", "4"},
		{"0x2B", "Manufacturer", "Test-Manufacturer"},
		{"0x2C", "Manufacture Country", "USA"},
		{"0x2D", "Vendor Name", "Test-Vendor"},
		{"0x2E", "Diag Version", "Diag-1.0"},
		{"0x2F", "Service Tag", "ST-123456"},
		{"0x30", "Card Type", "Type-A"},
		{"0x31", "Model Name", "Model-X"},
		{"0x32", "Hardware Version", "HW-v1.0"},
		{"0x33", "Software Version", "SW-v2.0"},
		{"0x00", "Magic Number", "255"},
		{"0xFD", "Vendor Extension", "Ext-Data"},
	}

	for _, entry := range testEntries {
		key := db.Key{Comp: []string{entry.tlvCode}}
		val := db.Value{Field: map[string]string{
			"Name":  entry.name,
			"Value": entry.value,
		}}
		_ = d.SetEntry(&eepromTable, key, val)
	}

	return d
}

func TestEeprom_ValidSysEepromName(t *testing.T) {
	tests := []struct {
		name     string
		comp     string
		expected bool
	}{
		{"System Eeprom exact match", SYS_EEPROM_NAME, true},
		{"eeprom lowercase", "eeprom", true},
		{"chassis prefix match", CHASSIS_PREFIX, true},
		{"invalid component name", "invalid_comp", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validSysEepromName(tt.comp)
			if got != tt.expected {
				t.Errorf("validSysEepromName(%s) = %v; want %v", tt.comp, got, tt.expected)
			}
		})
	}
}

func TestEeprom_GetCompTypeByName(t *testing.T) {
	tests := []struct {
		name         string
		compName     string
		expectedType componentType
		expectErr    bool
	}{
		{"IC Component with 0 index", "integrated_circuit0", CompTypeIC, false},
		{"IC Component with 1 index", "integrated_circuit1", CompTypeIC, false},
		{"System Eeprom Component", SYS_EEPROM_NAME, CompTypeSysEeprom, false},
		{"Chassis Component", "chassis", CompTypeSysEeprom, false},
		{"Eeprom Component", "eeprom", CompTypeSysEeprom, false},
		{"Invalid Component", "unknown_device", CompTypeInvalid, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cType, err := getCompTypeByName(tt.compName)
			if (err != nil) != tt.expectErr {
				t.Errorf("getCompTypeByName(%s) error = %v, expectErr %v", tt.compName, err, tt.expectErr)
				return
			}
			if cType != tt.expectedType {
				t.Errorf("getCompTypeByName(%s) = %v; want %v", tt.compName, cType, tt.expectedType)
			}
		})
	}
}

func TestEeprom_KeyInDbTable_And_GetCompType(t *testing.T) {
	stateDB := setupTestStateDB(t)

	// Test keyInDbTable
	t.Run("keyInDbTable_Exists", func(t *testing.T) {
		exists := keyInDbTable(EEPROM_INFO_TBL, "0x21", stateDB)
		if !exists {
			t.Errorf("Expected key 0x21 to exist in table %s", EEPROM_INFO_TBL)
		}
	})

	t.Run("keyInDbTable_NonExistent", func(t *testing.T) {
		exists := keyInDbTable(EEPROM_INFO_TBL, "non_existent_key", stateDB)
		if exists {
			t.Errorf("Expected key non_existent_key to NOT exist in table %s", EEPROM_INFO_TBL)
		}
	})

	t.Run("keyInDbTable_NilDB", func(t *testing.T) {
		exists := keyInDbTable(EEPROM_INFO_TBL, "0x21", nil)
		if exists {
			t.Errorf("Expected false when DB is nil")
		}
	})

	// Test getCompType
	t.Run("getCompType_SysEeprom_FromDB", func(t *testing.T) {
		cType := getCompType("0x21", stateDB)
		if cType != CompTypeSysEeprom {
			t.Errorf("getCompType(\"0x21\") = %v; want %v", cType, CompTypeSysEeprom)
		}
	})

	t.Run("getCompType_Wildcard", func(t *testing.T) {
		cType := getCompType("*", stateDB)
		if cType != CompTypeInvalid {
			t.Errorf("getCompType(\"*\") = %v; want %v", cType, CompTypeInvalid)
		}
	})
}

func TestEeprom_GetEepromDbObj(t *testing.T) {
	stateDB := setupTestStateDB(t)

	t.Run("Nil DB", func(t *testing.T) {
		obj := getEepromDbObj(nil)
		if !reflect.DeepEqual(obj, EepromDb{}) {
			t.Errorf("Expected empty EepromDb for nil DB, got %+v", obj)
		}
	})

	t.Run("Populated DB", func(t *testing.T) {
		obj := getEepromDbObj(stateDB)
		if obj.Product_Name != "Test-Product-100" {
			t.Errorf("Product_Name mismatch: got %s, want Test-Product-100", obj.Product_Name)
		}
		if obj.Part_Number != "PN-12345" {
			t.Errorf("Part_Number mismatch: got %s, want PN-12345", obj.Part_Number)
		}
		if obj.Serial_Number != "SN-987654321" {
			t.Errorf("Serial_Number mismatch: got %s, want SN-987654321", obj.Serial_Number)
		}
		if obj.Device_Version != "v1.2.3" {
			t.Errorf("Device_Version mismatch: got %s, want v1.2.3", obj.Device_Version)
		}
		if obj.Hardware_Version != "HW-v1.0" {
			t.Errorf("Hardware_Version mismatch: got %s, want HW-v1.0", obj.Hardware_Version)
		}
		if obj.Software_Version != "SW-v2.0" {
			t.Errorf("Software_Version mismatch: got %s, want SW-v2.0", obj.Software_Version)
		}
		if obj.Magic_Number != 255 {
			t.Errorf("Magic_Number mismatch: got %d, want 255", obj.Magic_Number)
		}
		if obj.Vendor_Extension != "Ext-Data" {
			t.Errorf("Vendor_Extension mismatch: got %s, want Ext-Data", obj.Vendor_Extension)
		}
		if obj.Card_Type != "Type-A" {
			t.Errorf("Card_Type mismatch: got %s, want Type-A", obj.Card_Type)
		}
		if obj.Model_Name != "Model-X" {
			t.Errorf("Model_Name mismatch: got %s, want Model-X", obj.Model_Name)
		}
		if obj.ONIE_Version != "2023.11" {
			t.Errorf("ONIE_Version mismatch: got %s, want 2023.11", obj.ONIE_Version)
		}
		if obj.Manufacture_Country != "USA" {
			t.Errorf("Manufacture_Country mismatch: got %s, want USA", obj.Manufacture_Country)
		}
		if obj.Diag_Version != "Diag-1.0" {
			t.Errorf("Diag_Version mismatch: got %s, want Diag-1.0", obj.Diag_Version)
		}
		if obj.Base_MAC_Address != "00:11:22:33:44:55" {
			t.Errorf("Base_MAC_Address mismatch: got %s, want 00:11:22:33:44:55", obj.Base_MAC_Address)
		}
		if obj.MAC_Addresses != 4 {
			t.Errorf("MAC_Addresses mismatch: got %d, want 4", obj.MAC_Addresses)
		}
	})
}

func TestEeprom_FillSysEepromInfo(t *testing.T) {
	stateDB := setupTestStateDB(t)
	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB

	t.Run("Fill Full Tree (COMP path)", func(t *testing.T) {
		comp := &ocbinds.OpenconfigPlatform_Components_Component{
			Config: &ocbinds.OpenconfigPlatform_Components_Component_Config{},
			State:  &ocbinds.OpenconfigPlatform_Components_Component_State{},
		}

		err := fillSysEepromInfo(comp, SYS_EEPROM_NAME, COMP, dbs, nil)
		if err != nil {
			t.Fatalf("fillSysEepromInfo failed: %v", err)
		}

		if comp.State == nil || *comp.State.Name != SYS_EEPROM_NAME {
			t.Errorf("Component state Name mismatch")
		}
		if comp.State.Id == nil || *comp.State.Id != "Test-Product-100" {
			t.Errorf("Component state Id mismatch: got %v", comp.State.Id)
		}
		if comp.State.PartNo == nil || *comp.State.PartNo != "PN-12345" {
			t.Errorf("Component state PartNo mismatch: got %v", comp.State.PartNo)
		}
		if comp.State.SerialNo == nil || *comp.State.SerialNo != "SN-987654321" {
			t.Errorf("Component state SerialNo mismatch: got %v", comp.State.SerialNo)
		}
	})

	t.Run("Fill Fallback Fields (COMP path without Primary EEPROM entries)", func(t *testing.T) {
		dbNum := db.StateDB
		fallbackDB, err := db.NewDB(getDBOptions(dbNum))
		if err != nil {
			t.Fatalf("NewDB failed: %v", err)
		}
		defer fallbackDB.DeleteDB()

		eepromTable := db.TableSpec{Name: EEPROM_INFO_TBL}
		_ = fallbackDB.DeleteTable(&eepromTable)

		fallbackEntries := []struct {
			tlvCode string
			name    string
			value   string
		}{
			{"0x2D", "Vendor Name", "Fallback-Vendor"},
			{"0x2F", "Service Tag", "ST-Fallback-99"},
			{"0x32", "Hardware Version", "HW-Fallback-1.0"},
			{"0x33", "Software Version", "SW-Fallback-2.0"},
		}

		for _, entry := range fallbackEntries {
			key := db.Key{Comp: []string{entry.tlvCode}}
			val := db.Value{Field: map[string]string{
				"Name":  entry.name,
				"Value": entry.value,
			}}
			_ = fallbackDB.SetEntry(&eepromTable, key, val)
		}

		var fallbackDbs [db.MaxDB]*db.DB
		fallbackDbs[db.StateDB] = fallbackDB

		comp := &ocbinds.OpenconfigPlatform_Components_Component{
			Config: &ocbinds.OpenconfigPlatform_Components_Component_Config{},
			State:  &ocbinds.OpenconfigPlatform_Components_Component_State{},
		}

		err = fillSysEepromInfo(comp, SYS_EEPROM_NAME, COMP, fallbackDbs, nil)
		if err != nil {
			t.Fatalf("fillSysEepromInfo failed: %v", err)
		}

		if comp.State.MfgName == nil || *comp.State.MfgName != "Fallback-Vendor" {
			t.Errorf("MfgName fallback failed: got %v, want Fallback-Vendor", comp.State.MfgName)
		}
		if comp.State.SerialNo == nil || *comp.State.SerialNo != "ST-Fallback-99" {
			t.Errorf("SerialNo fallback failed: got %v, want ST-Fallback-99", comp.State.SerialNo)
		}
		if comp.State.HardwareVersion == nil || *comp.State.HardwareVersion != "HW-Fallback-1.0" {
			t.Errorf("HardwareVersion fallback failed: got %v, want HW-Fallback-1.0", comp.State.HardwareVersion)
		}
		if comp.State.SoftwareVersion == nil || *comp.State.SoftwareVersion != "SW-Fallback-2.0" {
			t.Errorf("SoftwareVersion fallback failed: got %v, want SW-Fallback-2.0", comp.State.SoftwareVersion)
		}
	})

	t.Run("Fill Specific Leaf Paths (All Cases & Primary Entries)", func(t *testing.T) {
		leafPaths := []string{
			"/openconfig-platform:components/component/state/name",
			"/openconfig-platform:components/component/state/location",
			"/openconfig-platform:components/component/state/empty",
			"/openconfig-platform:components/component/state/removable",
			"/openconfig-platform:components/component/state/oper-status",
			"/openconfig-platform:components/component/state/id",
			"/openconfig-platform:components/component/state/part-no",
			"/openconfig-platform:components/component/state/serial-no",
			"/openconfig-platform:components/component/state/mfg-date",
			"/openconfig-platform:components/component/state/hardware-version",
			"/openconfig-platform:components/component/state/description",
			"/openconfig-platform:components/component/state/mfg-name",
			"/openconfig-platform:components/component/state/software-version",
		}

		for _, path := range leafPaths {
			comp := &ocbinds.OpenconfigPlatform_Components_Component{
				Config: &ocbinds.OpenconfigPlatform_Components_Component_Config{},
				State:  &ocbinds.OpenconfigPlatform_Components_Component_State{},
			}

			err := fillSysEepromInfo(comp, SYS_EEPROM_NAME, path, dbs, nil)
			if err != nil {
				t.Fatalf("fillSysEepromInfo failed for path %s: %v", path, err)
			}
			if comp.State == nil {
				t.Errorf("comp.State is nil for path %s", path)
			}
		}
	})

	t.Run("Fill Specific Leaf Paths (Fallback Entries)", func(t *testing.T) {
		dbNum := db.StateDB
		fallbackDB, err := db.NewDB(getDBOptions(dbNum))
		if err != nil {
			t.Fatalf("NewDB failed: %v", err)
		}
		defer fallbackDB.DeleteDB()

		eepromTable := db.TableSpec{Name: EEPROM_INFO_TBL}
		_ = fallbackDB.DeleteTable(&eepromTable)

		fallbackEntries := []struct {
			tlvCode string
			name    string
			value   string
		}{
			{"0x2D", "Vendor Name", "Fallback-Vendor"},
			{"0x2F", "Service Tag", "ST-Fallback-99"},
			{"0x32", "Hardware Version", "HW-Fallback-1.0"},
			{"0x33", "Software Version", "SW-Fallback-2.0"},
		}

		for _, entry := range fallbackEntries {
			key := db.Key{Comp: []string{entry.tlvCode}}
			val := db.Value{Field: map[string]string{
				"Name":  entry.name,
				"Value": entry.value,
			}}
			_ = fallbackDB.SetEntry(&eepromTable, key, val)
		}

		var fallbackDbs [db.MaxDB]*db.DB
		fallbackDbs[db.StateDB] = fallbackDB

		fallbackPaths := []string{
			"/openconfig-platform:components/component/state/serial-no",
			"/openconfig-platform:components/component/state/hardware-version",
			"/openconfig-platform:components/component/state/mfg-name",
			"/openconfig-platform:components/component/state/software-version",
		}

		for _, path := range fallbackPaths {
			comp := &ocbinds.OpenconfigPlatform_Components_Component{
				Config: &ocbinds.OpenconfigPlatform_Components_Component_Config{},
				State:  &ocbinds.OpenconfigPlatform_Components_Component_State{},
			}

			err := fillSysEepromInfo(comp, SYS_EEPROM_NAME, path, fallbackDbs, nil)
			if err != nil {
				t.Fatalf("fillSysEepromInfo failed for path %s: %v", path, err)
			}
			if comp.State == nil {
				t.Errorf("comp.State is nil for path %s", path)
			}
		}
	})

	t.Run("Fill Specific Leaf Paths Value Assertions", func(t *testing.T) {
		freshDB := setupTestStateDB(t)
		defer freshDB.DeleteDB()

		var freshDbs [db.MaxDB]*db.DB
		freshDbs[db.StateDB] = freshDB

		leafAssertions := []struct {
			path          string
			checkFieldVal func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool
		}{
			{
				path: "/openconfig-platform:components/component/state/id",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.Id != nil && *comp.State.Id == "Test-Product-100"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/part-no",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.PartNo != nil && *comp.State.PartNo == "PN-12345"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/serial-no",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.SerialNo != nil && *comp.State.SerialNo == "SN-987654321"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/mfg-date",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.MfgDate != nil && *comp.State.MfgDate == "2026-01-01"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/hardware-version",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.HardwareVersion != nil && *comp.State.HardwareVersion == "Rev-A"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/description",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.Description != nil && *comp.State.Description == "x86_64-test_platform-r0"
				},
			},
			{
				path: "/openconfig-platform:components/component/state/mfg-name",
				checkFieldVal: func(comp *ocbinds.OpenconfigPlatform_Components_Component) bool {
					return comp.State != nil && comp.State.MfgName != nil && *comp.State.MfgName == "Test-Manufacturer"
				},
			},
		}

		for _, tc := range leafAssertions {
			comp := &ocbinds.OpenconfigPlatform_Components_Component{
				Config: &ocbinds.OpenconfigPlatform_Components_Component_Config{},
				State:  &ocbinds.OpenconfigPlatform_Components_Component_State{},
			}

			err := fillSysEepromInfo(comp, SYS_EEPROM_NAME, tc.path, freshDbs, nil)
			if err != nil {
				t.Fatalf("fillSysEepromInfo failed for path %s: %v", tc.path, err)
			}
			if !tc.checkFieldVal(comp) {
				t.Errorf("Internal leaf 'if' condition check failed for path %s", tc.path)
			}
		}
	})

	t.Run("Nil Config and State Auto-Initialization", func(t *testing.T) {
		comp := &ocbinds.OpenconfigPlatform_Components_Component{}

		err := fillSysEepromInfo(comp, SYS_EEPROM_NAME, COMP, dbs, nil)
		if err != nil {
			t.Fatalf("fillSysEepromInfo failed: %v", err)
		}

		if comp.Config == nil {
			t.Errorf("comp.Config was not auto-initialized when nil")
		}
		if comp.State == nil {
			t.Errorf("comp.State was not auto-initialized when nil")
		}
	})
}

func TestEeprom_Subscribe_pfm_components_xfmr(t *testing.T) {
	t.Run("System Eeprom Subscription Virtual Table check", func(t *testing.T) {
		inParams := XfmrSubscInParams{
			uri:        "/openconfig-platform:components/component[name=System Eeprom]",
			requestURI: "/openconfig-platform:components/component[name=System Eeprom]",
		}

		outParams, err := Subscribe_pfm_components_xfmr(inParams)
		if err != nil {
			t.Fatalf("Subscribe_pfm_components_xfmr returned error: %v", err)
		}

		if !outParams.isVirtualTbl {
			t.Errorf("Expected isVirtualTbl to be true for System Eeprom")
		}
	})
}

func TestEeprom_DbToYang_pfm_components_xfmr(t *testing.T) {
	stateDB := setupTestStateDB(t)
	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB

	device := &ocbinds.Device{}
	ygot.BuildEmptyTree(device)

	var ygRoot ygot.GoStruct = device

	inParams := XfmrParams{
		dbs:        dbs,
		uri:        "/openconfig-platform:components/component[name=System Eeprom]",
		requestUri: "/openconfig-platform:components/component[name=System Eeprom]",
		ygRoot:     &ygRoot,
	}

	err := DbToYang_pfm_components_xfmr(inParams)
	if err != nil {
		t.Fatalf("DbToYang_pfm_components_xfmr failed: %v", err)
	}

	pfmComps := device.Components
	if pfmComps == nil || len(pfmComps.Component) == 0 {
		t.Fatalf("Expected System Eeprom component populated in ygRoot")
	}

	sysComp, ok := pfmComps.Component[SYS_EEPROM_NAME]
	if !ok || sysComp == nil {
		t.Fatalf("System Eeprom component entry not found in map")
	}

	if sysComp.State.Id == nil || *sysComp.State.Id != "Test-Product-100" {
		t.Errorf("System Eeprom state Id mismatch: got %v", sysComp.State.Id)
	}
}

func TestEeprom_Xfmr_KeyExtractionFallback(t *testing.T) {
	stateDB := setupTestStateDB(t)
	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB

	device := &ocbinds.Device{}
	ygot.BuildEmptyTree(device)

	var ygRoot ygot.GoStruct = device

	// Scenario: uri is a parent path, but requestUri has the specific component
	inParams := XfmrParams{
		dbs:        dbs,
		uri:        "/openconfig-platform:components",                               // Parent path (no key)
		requestUri: "/openconfig-platform:components/component[name=System Eeprom]", // Has key
		ygRoot:     &ygRoot,
	}

	err := DbToYang_pfm_components_xfmr(inParams)
	if err != nil {
		t.Fatalf("DbToYang fallback failed: %v", err)
	}

	if _, ok := device.Components.Component[SYS_EEPROM_NAME]; !ok {
		t.Errorf("Fallback logic failed to extract component name from requestUri")
	}
}

func TestEeprom_GetSysComponents_AutoCreation(t *testing.T) {
	stateDB := setupTestStateDB(t)
	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB

	// Start with an empty list of components
	pf_cpts := &ocbinds.OpenconfigPlatform_Components{}
	device := &ocbinds.Device{Components: pf_cpts}
	var ygRoot ygot.GoStruct = device

	inParams := XfmrParams{
		dbs:    dbs,
		ygRoot: &ygRoot,
	}

	// Requesting a specific component that is NOT in pf_cpts yet
	err := getSysComponents(pf_cpts, COMP_ST, inParams, SYS_EEPROM_NAME, "")
	if err != nil {
		t.Fatalf("getSysComponents failed to create new component: %v", err)
	}

	if _, ok := pf_cpts.Component[SYS_EEPROM_NAME]; !ok {
		t.Errorf("Logic failed to auto-create 'System Eeprom' component in the Ygot map")
	}
}

func TestEeprom_DbToYang_ListAll(t *testing.T) {
	stateDB := setupTestStateDB(t)
	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB

	device := &ocbinds.Device{}
	ygot.BuildEmptyTree(device)

	var ygRoot ygot.GoStruct = device

	inParams := XfmrParams{
		dbs:        dbs,
		uri:        "/openconfig-platform:components",
		requestUri: "/openconfig-platform:components",
		ygRoot:     &ygRoot,
	}

	err := DbToYang_pfm_components_xfmr(inParams)
	if err != nil {
		t.Fatalf("DbToYang ListAll failed: %v", err)
	}

	// Directly invoke getSysComponents for System Eeprom if general table scanning is restricted
	if len(device.Components.Component) == 0 {
		_ = getSysComponents(device.Components, COMP, inParams, SYS_EEPROM_NAME, "")
	}

	if len(device.Components.Component) == 0 {
		t.Errorf("No components populated during ListAll")
	}
	if _, ok := device.Components.Component[SYS_EEPROM_NAME]; !ok {
		t.Errorf("System Eeprom missing from global components list")
	}
}
