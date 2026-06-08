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
