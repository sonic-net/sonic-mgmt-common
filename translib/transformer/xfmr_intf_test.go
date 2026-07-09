package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
	"reflect"
	"strings"
	"testing"
)

func TestInvalidInterfaceType_FieldXfmrDbtoYang(t *testing.T) {
	name := "bogusinterfacename"
	dummyDbDataMap := make(map[db.DBNum]map[string]map[string]db.Value)
	inParams := XfmrParams{
		key:       name,
		uri:       "/interfaces/interface[name=" + name + "]/",
		dbDataMap: &dummyDbDataMap,
		curDb:     0,
	}
	tests := []struct {
		f    FieldXfmrDbtoYang
		name string
	}{
		{DbToYang_intf_hardware_port_xfmr, "DbToYang_intf_hardware_port_xfmr"},
		{DbToYang_intf_transceiver_xfmr, "DbToYang_intf_transceiver_xfmr"},
		{DbToYang_intf_physical_channel_xfmr, "DbToYang_intf_physical_channel_xfmr"},
		{DbToYang_pins_if_id_xfmr, "DbToYang_pins_if_id_xfmr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.f(inParams); err == nil {
				t.Fatalf("Expected an error when passing invalid interface name")
			}
		})
	}

	t.Run("DbToYangPath_intf_get_counters_path_xfmr", func(t *testing.T) {
		pathParams := XfmrDbToYgPathParams{
			tblName:    "COUNTERS_PORT_NAME_MAP",
			tblKeyComp: []string{name},
			ygPathKeys: make(map[string]string),
		}

		if err := DbToYangPath_intf_get_counters_path_xfmr(pathParams); err != nil {
			t.Fatalf("Path transformer failed with error: %v", err)
		}
	})
}

func TestInvalidInterfaceType_SubTreeXfmrDbToYang(t *testing.T) {
	name := "bogusinterfacename"
	var rootObj ygot.GoStruct = &ocbinds.Device{}
	inParams := XfmrParams{
		key:    name,
		uri:    "/openconfig-interfaces:interfaces/interface[name=" + name + "]/state/counters",
		ygRoot: &rootObj,
	}
	tests := []struct {
		f    SubTreeXfmrDbToYang
		name string
	}{
		{DbToYang_intf_get_counters_xfmr, "DbToYang_intf_get_counters_xfmr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f(inParams); err == nil {
				t.Fatalf("Expected an error when passing invalid interface name")
			}
		})
	}
}

func TestInvalidInterfaceType_YangToDb(t *testing.T) {
	name := "bogusinterfacename"
	inParams := XfmrParams{
		key:   name,
		uri:   "/interfaces/interface[name=" + name + "]/",
		param: "not nothing",
	}
	tests := []struct {
		f    FieldXfmrYangToDb
		name string
	}{
		{YangToDb_pins_if_id_xfmr, "YangToDb_pins_if_id_xfmr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.f(inParams); err == nil {
				t.Fatalf("Expected an error when passing invalid interface name")
			}
		})
	}
}

func TestDbToYang_intf_hardware_port_xfmr(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer configDb.DeleteDB()

	configDb.SetEntry(&db.TableSpec{Name: "PORT"},
		db.Key{Comp: []string{"Ethernet55"}},
		db.Value{Field: map[string]string{"index": "55"}})

	dbs := [db.MaxDB]*db.DB{
		db.ConfigDB: configDb,
	}

	tests := []struct {
		name          string
		dbArray       [db.MaxDB]*db.DB
		expectError   bool
		expectedValue string
	}{
		{
			name:          "Success_Path",
			dbArray:       dbs,
			expectError:   false,
			expectedValue: "1/55",
		},
		{
			name:          "Error_Path_Trigger_Missing_DB",
			dbArray:       [db.MaxDB]*db.DB{},
			expectError:   true,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				key:   "Ethernet55",
				curDb: db.ConfigDB,
				dbs:   tt.dbArray,
				uri:   "/interfaces/interface[name=Ethernet55]/state/hardware-port",
			}

			result, err := DbToYang_intf_hardware_port_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result["hardware-port"] != tt.expectedValue {
				t.Errorf("Expected %s, but got %v", tt.expectedValue, result["hardware-port"])
			}
		})
	}
}

func TestDbToYang_intf_transceiver_xfmr(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer configDb.DeleteDB()

	configDb.SetEntry(&db.TableSpec{Name: "PORT"},
		db.Key{Comp: []string{"Ethernet1/0/1"}},
		db.Value{Field: map[string]string{"index": "5"}})

	dbs := [db.MaxDB]*db.DB{
		db.ConfigDB: configDb,
	}

	tests := []struct {
		name          string
		dbArray       [db.MaxDB]*db.DB
		expectError   bool
		expectedValue string
	}{
		{
			name:          "Success_Path",
			dbArray:       dbs,
			expectError:   false,
			expectedValue: "Ethernet5",
		},
		{
			name:          "Error_Path_Trigger_Missing_DB",
			dbArray:       [db.MaxDB]*db.DB{},
			expectError:   true,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				key:   "Ethernet1/0/1",
				curDb: db.ConfigDB,
				dbs:   tt.dbArray,
				uri:   "/interfaces/interface[name=Ethernet1/0/1]/state/transceiver",
			}

			result, err := DbToYang_intf_transceiver_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				if result != nil {
					t.Errorf("Expected result map to be nil on failure, but got: %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected Error: %v", err)
			}
			if result["transceiver"] != tt.expectedValue {
				t.Errorf("Expected %s, but got %v", tt.expectedValue, result["transceiver"])
			}
		})
	}
}

func TestDbToYangPath_intf_path_xfmr(t *testing.T) {
	rootPath := "/openconfig-interfaces:interfaces/interface"

	tests := []struct {
		name        string
		tblKeyComp  []string
		expectError bool
		errorMsg    string
		expectKeys  map[string]string
	}{
		{
			name:        "Success - Valid Single Component Key (Ethernet202)",
			tblKeyComp:  []string{"Ethernet202"},
			expectError: false,
			expectKeys: map[string]string{
				rootPath + "/name": "Ethernet202",
			},
		},
		{
			name:        "Success - Valid Single Component Key (PortChannel10)",
			tblKeyComp:  []string{"PortChannel10"},
			expectError: false,
			expectKeys: map[string]string{
				rootPath + "/name": "PortChannel10",
			},
		},
		{
			name:        "Failure - Empty Table Key Components List",
			tblKeyComp:  []string{},
			expectError: true,
			errorMsg:    "Invalid tblKeyCom for intf path xmfr:",
		},
		{
			name:        "Failure - Multi Component Composite Key",
			tblKeyComp:  []string{"Ethernet202", "Subinterface1"},
			expectError: true,
			errorMsg:    "Invalid tblKeyCom for intf path xmfr:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ygPathKeysMap := make(map[string]string)

			inParams := XfmrDbToYgPathParams{
				tblKeyComp: tt.tblKeyComp,
				ygPathKeys: ygPathKeysMap,
			}

			err := DbToYangPath_intf_path_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error from the path transformer but got success")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message verification failed.\nExpected containing: %q\nGot actual error: %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error returned from path transformer: %v", err)
			}

			if len(inParams.ygPathKeys) != len(tt.expectKeys) {
				t.Fatalf("Output path map size mismatch. Expected %d entries, got %d", len(tt.expectKeys), len(inParams.ygPathKeys))
			}

			for expectedKey, expectedVal := range tt.expectKeys {
				actualVal, exists := inParams.ygPathKeys[expectedKey]
				if !exists {
					t.Errorf("Expected path key %q was not found in the output ygPathKeys map", expectedKey)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("Value mismatch for path key %q.\nExpected mapped value: %q\nGot actual value: %q", expectedKey, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestDbToYang_pins_if_id_xfmr(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer configDb.DeleteDB()

	dbs := [db.MaxDB]*db.DB{
		db.ConfigDB: configDb,
	}

	applDb, _ := db.NewDB(db.Options{
		DBNo:               db.ApplDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: ":",
		KeySeparator:       ":",
	})
	defer applDb.DeleteDB()

	dbs2 := [db.MaxDB]*db.DB{
		db.ApplDB: applDb,
	}

	stateDb, _ := db.NewDB(db.Options{
		DBNo:               db.StateDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: ":",
		KeySeparator:       ":",
	})
	defer stateDb.DeleteDB()

	dbs3 := [db.MaxDB]*db.DB{
		db.StateDB: stateDb,
	}

	tests := []struct {
		name        string
		setupMock   func()
		inParams    XfmrParams
		expectError bool
		errorMsg    string
		expectMap   map[string]interface{}
	}{
		{
			name:      "Failure - Invalid Interface Identifier",
			setupMock: func() {},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=InvalidIntf0]/config/id",
				curDb: db.ConfigDB,
				dbs:   dbs,
			},
			expectError: true,
			errorMsg:    "Invalid interface:",
		},
		{
			name: "Failure - Invalid ID Value String in DB",
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "P4RT_PORT_ID_TABLE"},
					db.Key{Comp: []string{"Vlan100"}},
					db.Value{Field: map[string]string{"id": "200"}})
			},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Vlan100]/config/id",
				curDb: db.ConfigDB,
				dbs:   dbs,
			},
			expectError: true,
			errorMsg:    "not supported for Config Id",
		},
		{
			name: "Failure - ID Field Missing from DB Entry",
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet405"}},
					db.Value{Field: map[string]string{"id": ""}})
			},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Ethernet405]/config/id",
				curDb: db.ConfigDB,
				dbs:   dbs,
			},
			expectError: true,
			errorMsg:    "config id field not found in DB.",
		},
		{
			name:      "Failure - DB Connection Error",
			setupMock: func() {},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Ethernet4]/config/id",
				curDb: db.ErrorDB,
				dbs:   dbs,
			},
			expectError: true,
			errorMsg:    "connection closed",
		},
		{
			name: "Success - Valid ID in ConfigDB",
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet201"}},
					db.Value{Field: map[string]string{"id": "201"}})
			},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Ethernet201]/config/id",
				curDb: db.ConfigDB,
				dbs:   dbs,
			},
			expectError: false,
			expectMap:   map[string]interface{}{"id": uint32(201)},
		},
		{
			name: "Success - Valid ID in ApplDB",
			setupMock: func() {
				applDb.SetEntry(&db.TableSpec{Name: "P4RT_PORT_ID_TABLE"},
					db.Key{Comp: []string{"Ethernet202"}},
					db.Value{Field: map[string]string{"id": "202"}})
			},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Ethernet202]/config/id",
				curDb: db.ApplDB,
				dbs:   dbs2,
			},
			expectError: false,
			expectMap:   map[string]interface{}{"id": uint32(202)},
		},
		{
			name: "Success - Valid ID in StateDB",
			setupMock: func() {
				stateDb.SetEntry(&db.TableSpec{Name: "PORT_TABLE"},
					db.Key{Comp: []string{"Ethernet203"}},
					db.Value{Field: map[string]string{"id": "203"}})
			},
			inParams: XfmrParams{
				uri:   "/interfaces/interface[name=Ethernet203]/state/id",
				curDb: db.StateDB,
				dbs:   dbs3,
			},
			expectError: false,
			expectMap:   map[string]interface{}{"id": uint32(203)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			resMap, err := DbToYang_pins_if_id_xfmr(tt.inParams)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error but function returned success")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error string mismatch.\nExpected containing: %q\nGot: %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error returned: %v", err)
			}

			if len(resMap) != len(tt.expectMap) {
				t.Fatalf("Result map size mismatch. Expected %d keys, got %d", len(tt.expectMap), len(resMap))
			}

			for k, expectedVal := range tt.expectMap {
				actualVal, exists := resMap[k]
				if !exists {
					t.Errorf("Expected key %q missing from output map", k)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("Value mismatch for key %q.\nExpected (%T): %v\nGot (%T): %v", k, expectedVal, expectedVal, actualVal, actualVal)
				}
			}
		})
	}

	configDb.DeleteEntry(&db.TableSpec{Name: "P4RT_PORT_ID_TABLE"}, db.Key{Comp: []string{"Ethernet0"}})
	configDb.DeleteEntry(&db.TableSpec{Name: "P4RT_PORT_ID_TABLE"}, db.Key{Comp: []string{"Vlan100"}})
	configDb.DeleteEntry(&db.TableSpec{Name: "PORT"}, db.Key{Comp: []string{"Ethernet405"}})
	configDb.DeleteEntry(&db.TableSpec{Name: "PORT"}, db.Key{Comp: []string{"Ethernet201"}})
	applDb.DeleteEntry(&db.TableSpec{Name: "P4RT_PORT_ID_TABLE"}, db.Key{Comp: []string{"Ethernet202"}})
	stateDb.DeleteEntry(&db.TableSpec{Name: "PORT_TABLE"}, db.Key{Comp: []string{"Ethernet203"}})
}

func TestYangToDb_pins_if_id_xfmr(t *testing.T) {
	validID := uint32(100)
	zeroID := uint32(0)
	invalidTypeParam := "not-a-uint32-pointer"

	tests := []struct {
		name        string
		uri         string
		param       interface{}
		setupMock   func()
		expectError bool
		errorMsg    string
		expectMap   map[string]string
	}{
		{
			name:        "Success - Valid Ethernet ID Parsed to String",
			uri:         "/interfaces/interface[name=Ethernet202]/config/id",
			param:       &validID,
			setupMock:   func() {},
			expectError: false,
			expectMap:   map[string]string{"id": "100"},
		},
		{
			name:        "Success - Edge Case Zero ID Parsed to String",
			uri:         "/interfaces/interface[name=Ethernet4]/config/id",
			param:       &zeroID,
			setupMock:   func() {},
			expectError: false,
			expectMap:   map[string]string{"id": "0"},
		},
		{
			name:        "Failure - Interface KEY Not Present in URI",
			uri:         "/interfaces/interface/config/id",
			param:       &validID,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Interface KEY not present",
		},
		{
			name:        "Failure - Invalid Interface Identifier Syntax",
			uri:         "/interfaces/interface[name=InvalidIntfName0]/config/id",
			param:       &validID,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Invalid interface:",
		},
		{
			name:        "Failure - Unsupported Interface Type (Vlan)",
			uri:         "/interfaces/interface[name=Vlan100]/config/id",
			param:       &validID,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "not supported for Config Id",
		},
		{
			name:        "Failure - Param is Nil Pointer",
			uri:         "/interfaces/interface[name=Ethernet202]/config/id",
			param:       nil,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Config Id doesn't exist",
		},
		{
			name:        "Failure - Param Type Type-Assertion Mismatch",
			uri:         "/interfaces/interface[name=Ethernet202]/config/id",
			param:       &invalidTypeParam,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Config Id doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			inParams := XfmrParams{
				uri:   tt.uri,
				param: tt.param,
			}

			resMap, err := YangToDb_pins_if_id_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error but function returned execution success")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message text mismatch.\nExpected containing: %q\nGot actual: %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error returned: %v", err)
			}

			if len(resMap) != len(tt.expectMap) {
				t.Fatalf("Result map size mismatch. Expected %d keys, got %d", len(tt.expectMap), len(resMap))
			}

			for k, expectedVal := range tt.expectMap {
				actualVal, exists := resMap[k]
				if !exists {
					t.Errorf("Expected database map key %q missing from transformer output", k)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("Value mismatch for key %q.\nExpected: %q\nGot: %q", k, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestGetPortIndex(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer configDb.DeleteDB()

	dbs := [db.MaxDB]*db.DB{
		db.ConfigDB: configDb,
	}
	errDb, _ := db.NewDB(db.Options{
		DBNo:               db.ErrorDB,
		InitIndicator:      "ERROR_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer errDb.DeleteDB()

	dbs1 := [db.MaxDB]*db.DB{
		db.ErrorDB: errDb,
	}
	funcName := "DbToYang_intf_hardware_port_xfmr"

	t.Run("Success - Valid Path", func(t *testing.T) {
		configDb.SetEntry(&db.TableSpec{Name: "PORT"},
			db.Key{Comp: []string{"Ethernet202"}},
			db.Value{Field: map[string]string{"index": "0"}})

		params := XfmrParams{
			uri:   "/interfaces/interface[name=Ethernet202]",
			curDb: db.ConfigDB,
			dbs:   dbs,
		}

		index, err := getPortIndex(params, funcName)

		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if index != "0" {
			t.Errorf("Expected index '0', got '%s'", index)
		}
	})

	t.Run("Error 1 - Invalid Interface Name Syntax", func(t *testing.T) {
		params := XfmrParams{
			uri:   "/interfaces/interface[name=InvalidName123]",
			curDb: db.ConfigDB,
			dbs:   dbs,
		}

		_, err := getPortIndex(params, funcName)
		if err == nil || !strings.Contains(err.Error(), "Invalid interface:") {
			t.Errorf("Expected 'Invalid interface' error, got: %v", err)
		}
	})

	t.Run("Error 2 - Not IntfTypeEthernet", func(t *testing.T) {
		params := XfmrParams{uri: "/interfaces/interface[name=PortChannel1]", curDb: db.ConfigDB, dbs: dbs}
		_, err := getPortIndex(params, funcName)

		if err == nil || !strings.Contains(err.Error(), "interface type is not IntfTypeEthernet") {
			t.Errorf("Expected type mismatch error, got: %v", err)
		}
	})

	t.Run("Error 3 - Entry does not Exist", func(t *testing.T) {
		params := XfmrParams{uri: "/interfaces/interface[name=Ethernet101]", curDb: db.ErrorDB, dbs: dbs1}
		_, err := getPortIndex(params, funcName)

		if err == nil || !strings.Contains(err.Error(), "Entry does not exist") {
			t.Errorf("Expected type map missing error, got: %v", err)
		}
	})

	t.Run("Error 4 - DB Read Error or Entry Missing", func(t *testing.T) {

		params := XfmrParams{uri: "/interfaces/interface[name=Ethernet888]", curDb: db.ConfigDB, dbs: dbs}
		_, err := getPortIndex(params, funcName)

		if err == nil {
			t.Error("Expected DB read error, got nil")
		}
	})

	t.Run("Error 5 - Index Field Missing in DB Entry", func(t *testing.T) {
		configDb.SetEntry(&db.TableSpec{Name: "PORT"},
			db.Key{Comp: []string{"Ethernet66"}},
			db.Value{Field: map[string]string{"lanes": "16"}})

		params := XfmrParams{uri: "/interfaces/interface[name=Ethernet66]", curDb: db.ConfigDB, dbs: dbs}
		_, err := getPortIndex(params, funcName)

		expectedMsg := funcName + " index not found in DB"
		if err == nil || !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Expected %q error, got: %v", expectedMsg, err)
		}
	})
}

func TestDbToYang_intf_get_counters_xfmr(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{DBNo: db.ConfigDB, TableNameSeparator: "|", KeySeparator: "|"})
	defer configDb.DeleteDB()
	dbs := [db.MaxDB]*db.DB{db.ConfigDB: configDb}

	tests := []struct {
		name        string
		uri         string
		getDevice   func() *ocbinds.Device
		setupMock   func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "Success - Populate Counters Core Callback Execution",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				return d
			},
			setupMock: func() {
				entry := IntfTypeTblMap[IntfTypeEthernet]
				targetField := reflect.ValueOf(&entry.CountersHdl).Elem().FieldByName("PopulateCounters")
				if targetField.IsValid() {
					mockFunc := reflect.MakeFunc(targetField.Type(), func(args []reflect.Value) []reflect.Value {
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					})
					targetField.Set(mockFunc)
				}
				IntfTypeTblMap[IntfTypeEthernet] = entry
			},
			expectError: false,
		},
		{
			name: "Success - Pre-existing Interfaces Subtree Match Branch",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				d.Interfaces.NewInterface("Ethernet202")
				return d
			},
			setupMock: func() {
				entry := IntfTypeTblMap[IntfTypeEthernet]
				targetField := reflect.ValueOf(&entry.CountersHdl).Elem().FieldByName("PopulateCounters")
				if targetField.IsValid() {
					mockFunc := reflect.MakeFunc(targetField.Type(), func(args []reflect.Value) []reflect.Value {
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					})
					targetField.Set(mockFunc)
				}
				IntfTypeTblMap[IntfTypeEthernet] = entry
			},
			expectError: false,
		},
		{
			name: "Success - Redundant Target URI Path Branch Coverage",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/hardware-port",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				return d
			},
			setupMock:   func() {},
			expectError: false,
		},
		{
			name: "Failure - Invalid Interface Type Branch Coverage",
			uri:  "/openconfig-interfaces:interfaces/interface[name=InvalidIntf99]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				return d
			},
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Invalid interface type IntfTypeUnset",
		},
		{
			name: "Success - Counters Callback Not Supported Branch Coverage",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				return d
			},
			setupMock: func() {
				entry := IntfTypeTblMap[IntfTypeEthernet]
				entry.CountersHdl.PopulateCounters = nil
				IntfTypeTblMap[IntfTypeEthernet] = entry
			},
			expectError: false,
		},
		{
			name: "Success - Interface Not Found In Existing Map Coverage",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}
				ygot.BuildEmptyTree(d)
				d.Interfaces.NewInterface("Ethernet4")
				return d
			},
			setupMock: func() {
				entry := IntfTypeTblMap[IntfTypeEthernet]
				targetField := reflect.ValueOf(&entry.CountersHdl).Elem().FieldByName("PopulateCounters")
				if targetField.IsValid() {
					mockFunc := reflect.MakeFunc(targetField.Type(), func(args []reflect.Value) []reflect.Value {
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					})
					targetField.Set(mockFunc)
				}
				IntfTypeTblMap[IntfTypeEthernet] = entry
			},
			expectError: false,
		},
		{
			name: "Success - Nil State Component Verification Branch Coverage",
			uri:  "/openconfig-interfaces:interfaces/interface[name=Ethernet202]/state/counters",
			getDevice: func() *ocbinds.Device {
				d := &ocbinds.Device{}

				d.Interfaces = &ocbinds.OpenconfigInterfaces_Interfaces{}
				d.Interfaces.Interface = make(map[string]*ocbinds.OpenconfigInterfaces_Interfaces_Interface)

				intfObj := &ocbinds.OpenconfigInterfaces_Interfaces_Interface{
					Name:  ygot.String("Ethernet202"),
					State: &ocbinds.OpenconfigInterfaces_Interfaces_Interface_State{},
				}

				d.Interfaces.Interface["Ethernet202"] = intfObj
				return d
			},
			setupMock: func() {
				entry := IntfTypeTblMap[IntfTypeEthernet]
				targetField := reflect.ValueOf(&entry.CountersHdl).Elem().FieldByName("PopulateCounters")
				if targetField.IsValid() {
					mockFunc := reflect.MakeFunc(targetField.Type(), func(args []reflect.Value) []reflect.Value {
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					})
					targetField.Set(mockFunc)
				}
				IntfTypeTblMap[IntfTypeEthernet] = entry
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ethernetOrigVal, ethExists := IntfTypeTblMap[IntfTypeEthernet]

			tt.setupMock()

			t.Cleanup(func() {
				if ethExists {
					IntfTypeTblMap[IntfTypeEthernet] = ethernetOrigVal
				}
			})

			devRoot := tt.getDevice()
			var goStructInterface ygot.GoStruct = devRoot

			var inParams XfmrParams
			inParams.uri = tt.uri
			inParams.curDb = db.ConfigDB
			inParams.dbs = dbs
			inParams.ygRoot = &goStructInterface

			err := DbToYang_intf_get_counters_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error but function returned success")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error string mismatch.\nExpected containing: %q\nGot: %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error returned on positive execution path: %v", err)
			}

			if devRoot == nil || devRoot.Interfaces == nil {
				t.Errorf("Expected populated Openconfig output tree layout, but tree reference components are unassigned")
			}
		})
	}
}

func TestGetSpecificCounterAttr_InFcsErrors(t *testing.T) {
	tests := []struct {
		name          string
		targetUriPath string
		entry         *db.Value
		getCounter    func() interface{}
		expectHandled bool
		expectError   bool
	}{
		{
			name:          "Success - Hit InFcsErrors Target Case Branch",
			targetUriPath: "/openconfig-interfaces:interfaces/interface/state/counters/in-fcs-errors",
			entry: &db.Value{
				Field: map[string]string{
					"SAI_PORT_STAT_ETHER_STATS_CRC_ALIGN_ERRORS": "42",
				},
			},
			getCounter: func() interface{} {
				c := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_Counters{}
				return c
			},
			expectHandled: true,
			expectError:   false,
		},
		{
			name:          "Success - Fallthrough to Default Path Unhandled",
			targetUriPath: "/openconfig-interfaces:interfaces/interface/state/counters/unsupported-attribute",
			entry:         &db.Value{},
			getCounter: func() interface{} {
				return &ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_Counters{}
			},
			expectHandled: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counterContainer := tt.getCounter()

			handled, err := getSpecificCounterAttr(tt.targetUriPath, tt.entry, counterContainer)

			if tt.expectError && err == nil {
				t.Fatalf("Expected execution error but function returned success status")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected processing error encountered: %v", err)
			}

			if handled != tt.expectHandled {
				t.Errorf("Handled boolean indicator status mismatch.\nExpected: %t\nGot: %t", tt.expectHandled, handled)
			}

			if tt.targetUriPath == "/openconfig-interfaces:interfaces/interface/state/counters/in-fcs-errors" && err == nil {
				typedCounter := counterContainer.(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_Counters)
				if typedCounter.InFcsErrors == nil {
					t.Errorf("Target field InFcsErrors pointer remained nil; value was not parsed into object")
				}
			}
		})
	}
}

func TestDbToYang_intf_physical_channel_xfmr(t *testing.T) {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		InitIndicator:      "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer configDb.DeleteDB()

	stateDb, _ := db.NewDB(db.Options{
		DBNo:               db.StateDB,
		InitIndicator:      "STATE_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})
	defer stateDb.DeleteDB()

	dbs := [db.MaxDB]*db.DB{
		db.ConfigDB: configDb,
		db.StateDB:  stateDb,
	}

	originalSfpMap := sfpTypeToMaxLanesMap
	t.Cleanup(func() {
		sfpTypeToMaxLanesMap = originalSfpMap
	})
	sfpTypeToMaxLanesMap = map[string]int{
		"QSFP28": 4,
	}

	tests := []struct {
		name        string
		uri         string
		curDb       db.DBNum
		setupMock   func()
		expectError bool
		errorMsg    string
		expectMap   map[string]interface{}
	}{
		{
			name:        "Failure - Invalid Interface Name Syntax",
			uri:         "/interfaces/interface[name=InvalidIntf0]/state/physical-channel",
			curDb:       db.ConfigDB,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "Invalid interface:",
		},
		{
			name:        "Failure - Interface Type is Not Ethernet (PortChannel)",
			uri:         "/interfaces/interface[name=PortChannel1]/state/physical-channel",
			curDb:       db.ConfigDB,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "interface type is not IntfTypeEthernet",
		},
		{
			name:  "Failure - Lanes Field Missing from DB Entry",
			uri:   "/interfaces/interface[name=Ethernet4]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet4"}},
					db.Value{Field: map[string]string{"index": "4"}})
			},
			expectError: true,
			errorMsg:    "lanes not found in DB",
		},
		{
			name:  "Failure - Transceiver Type Field Missing in StateDB",
			uri:   "/interfaces/interface[name=Ethernet8]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet8"}},
					db.Value{Field: map[string]string{"lanes": "8", "index": "8"}})

				stateDb.SetEntry(&db.TableSpec{Name: "TRANSCEIVER_INFO"},
					db.Key{Comp: []string{"Ethernet8"}},
					db.Value{Field: map[string]string{"type": ""}})
			},
			expectError: true,
			errorMsg:    "empty transceiver type for physical-channel",
		},
		{
			name:  "Failure - Unknown Transceiver Type Map Lookup Match",
			uri:   "/interfaces/interface[name=Ethernet12]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet12"}},
					db.Value{Field: map[string]string{"lanes": "12", "index": "12"}})

				stateDb.SetEntry(&db.TableSpec{Name: "TRANSCEIVER_INFO"},
					db.Key{Comp: []string{"Ethernet12"}},
					db.Value{Field: map[string]string{"type": "UNKNOWN_SFP"}})
			},
			expectError: true,
			errorMsg:    "could not find the max number of lanes",
		},
		{
			name:        "Failure - DB Connection Error",
			uri:         "/interfaces/interface[name=Ethernet16]/state/physical-channel",
			curDb:       db.ErrorDB,
			setupMock:   func() {},
			expectError: true,
			errorMsg:    "DB error",
		},
		{
			name:  "Failure - Port Index Field Missing from DB",
			uri:   "/interfaces/interface[name=Ethernet100]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet100"}},
					db.Value{Field: map[string]string{"lanes": "100"}})
			},
			expectError: true,
			errorMsg:    "index not found in DB",
		},
		{
			name:  "Failure - Transceiver Entry Missing from StateDB",
			uri:   "/interfaces/interface[name=Ethernet20]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {
				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet20"}},
					db.Value{Field: map[string]string{"lanes": "20", "index": "20"}})
			},
			expectError: true,
			errorMsg:    "Entry does not exist",
		},
		{
			name:  "Failure - Platform ChannelOffset Error",
			uri:   "/interfaces/interface[name=Ethernet999]/state/physical-channel",
			curDb: db.ConfigDB,
			setupMock: func() {

				configDb.SetEntry(&db.TableSpec{Name: "PORT"},
					db.Key{Comp: []string{"Ethernet999"}},
					db.Value{Field: map[string]string{"lanes": "999", "index": "999"}})

				stateDb.SetEntry(&db.TableSpec{Name: "TRANSCEIVER_INFO"},
					db.Key{Comp: []string{"Ethernet999"}},
					db.Value{Field: map[string]string{"type": "QSFP28"}})
			},
			expectError: true,
			errorMsg:    "could not find the channel offset for Ethernet999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			inParams := XfmrParams{
				uri:   tt.uri,
				curDb: tt.curDb,
				dbs:   dbs,
			}

			resMap, err := DbToYang_intf_physical_channel_xfmr(inParams)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected function execution to fail but got success")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error string signature mismatch.\nExpected containing: %q\nGot actual: %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error returned during logic execution: %v", err)
			}

			actualChannels, ok := resMap["physical-channel"].([]uint16)
			if !ok {
				t.Fatalf("Returned key 'physical-channel' type mismatch. Expected []uint16, got: %T", resMap["physical-channel"])
			}

			expectedChannels := tt.expectMap["physical-channel"].([]uint16)
			if len(actualChannels) != len(expectedChannels) {
				t.Fatalf("Channel array size mismatch. Expected length %d, got %d", len(expectedChannels), len(actualChannels))
			}

			for idx, targetVal := range expectedChannels {
				if actualChannels[idx] != targetVal {
					t.Errorf("Channel assignment value mismatch at position index %d. Expected %d, got %d", idx, targetVal, actualChannels[idx])
				}
			}
		})
	}
}

func TestCPU_DbToYang_intf_cpu_xfmr(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		expectMap map[string]interface{}
		wantErr   bool
	}{
		{"Success - CPU", "/interfaces/interface[name=CPU]/state/cpu", map[string]interface{}{"cpu": true}, false},
		{"Success - Non CPU", "/interfaces/interface[name=Ethernet999]/state/cpu", map[string]interface{}{}, false},
		{"Failure - No Key", "/interfaces/interface/state/cpu", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{uri: tt.uri}
			res, err := DbToYang_intf_cpu_xfmr(inParams)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && !reflect.DeepEqual(res, tt.expectMap) {
				t.Errorf("Expected %v, got %v", tt.expectMap, res)
			}
		})
	}
}

func TestCPU_DbToYang_intf_description_xfmr(t *testing.T) {
	d, err := db.NewDB(getDBOptions(db.ConfigDB))
	if err != nil {
		t.Fatalf("Failed to create ConfigDB: %v", err)
	}
	defer d.DeleteDB()

	ts := &db.TableSpec{Name: "PORT"}
	d.SetEntry(ts, db.Key{Comp: []string{"Ethernet998"}}, db.Value{Field: map[string]string{"description": "Test Description"}})
	d.SetEntry(ts, db.Key{Comp: []string{"Ethernet999"}}, db.Value{Field: map[string]string{"other": "value"}})

	defer func() {
		d.DeleteEntry(ts, db.Key{Comp: []string{"Ethernet998"}})
		d.DeleteEntry(ts, db.Key{Comp: []string{"Ethernet999"}})
	}()

	tests := []struct {
		name      string
		ifName    string
		expectVal string
		expectErr bool
	}{
		{
			name:      "Successful Retrieval",
			ifName:    "Ethernet998",
			expectVal: "Test Description",
			expectErr: false,
		},
		{
			name:      "Missing Description Field",
			ifName:    "Ethernet999",
			expectErr: true,
		},
		{
			name:      "Invalid Interface Name",
			ifName:    "Bogus0",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				key:   tt.ifName,
				uri:   "/openconfig-interfaces:interfaces/interface[name=" + tt.ifName + "]/config/description",
				curDb: db.ConfigDB,
				dbs:   [db.MaxDB]*db.DB{db.ConfigDB: d},
			}
			got, err := DbToYang_intf_description_xfmr(inParams)
			if (err != nil) != tt.expectErr {
				t.Fatalf("DbToYang_intf_description_xfmr() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr {
				if val, ok := got["description"]; !ok || val != tt.expectVal {
					t.Errorf("Expected description %q, got %v", tt.expectVal, got)
				}
			}
		})
	}
}

func TestCPU_DbToYang_pins_ifindex_xfmr(t *testing.T) {
	d, err := db.NewDB(getDBOptions(db.ConfigDB))
	if err != nil {
		t.Fatalf("Failed to create ConfigDB: %v", err)
	}
	defer d.DeleteDB()

	ts := &db.TableSpec{Name: "P4RT_PORT_ID_TABLE"}
	d.SetEntry(ts, db.Key{Comp: []string{"Ethernet998"}}, db.Value{Field: map[string]string{"id": "12345"}})
	d.SetEntry(ts, db.Key{Comp: []string{"Ethernet999"}}, db.Value{Field: map[string]string{"id": "not-a-number"}})

	defer func() {
		d.DeleteEntry(ts, db.Key{Comp: []string{"Ethernet998"}})
		d.DeleteEntry(ts, db.Key{Comp: []string{"Ethernet999"}})
	}()

	tests := []struct {
		name      string
		ifName    string
		expectVal uint32
		expectErr bool
	}{
		{
			name:      "Successful ID Parsing",
			ifName:    "Ethernet998",
			expectVal: 12345,
			expectErr: false,
		},
		{
			name:      "Non-numeric ID",
			ifName:    "Ethernet999",
			expectErr: true,
		},
		{
			name:      "Missing Entry",
			ifName:    "Ethernet997",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				key:   tt.ifName,
				uri:   "/openconfig-interfaces:interfaces/interface[name=" + tt.ifName + "]/state/ifindex",
				curDb: db.ConfigDB,
				dbs:   [db.MaxDB]*db.DB{db.ConfigDB: d},
			}
			got, err := DbToYang_pins_ifindex_xfmr(inParams)
			if (err != nil) != tt.expectErr {
				t.Fatalf("DbToYang_pins_ifindex_xfmr() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr {
				if val, ok := got["ifindex"]; !ok || val != tt.expectVal {
					t.Errorf("Expected ifindex %d, got %v", tt.expectVal, got)
				}
			}
		})
	}
}

func mockPopulateCounters(inParams XfmrParams, itfName string, counters interface{}) error {
	switch c := counters.(type) {
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters:
		var val uint64
		switch itfName {
		case "Ethernet998":
			val = 100
		case "Ethernet999":
			val = 50
		default:
			val = 0 // Return 0 for parent aggregate/unhandled interfaces to prevent unwanted additions
		}
		c.InPkts = &val
		c.OutPkts = &val
	case *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters:
		var val uint64 = 200
		c.InPkts = &val
		c.OutPkts = &val
	}
	return nil
}

func TestCPU_DbToYang_intf_ipv4_counters_xfmr_PortChannel(t *testing.T) {
	stateDb, err := db.NewDB(getDBOptions(db.StateDB))
	if err != nil {
		t.Fatalf("Failed to create StateDB: %v", err)
	}
	defer stateDb.DeleteDB()

	sep := stateDb.Opts.KeySeparator
	lagName := "PortChannelTestUnique"

	ts := &db.TableSpec{Name: LAG_MEMBER_TABLE_TN + sep + lagName}
	stateDb.DeleteMapAll(ts)
	defer stateDb.DeleteMapAll(ts)

	stateDb.SetEntry(ts, db.Key{Comp: []string{"Ethernet998"}}, db.Value{Field: map[string]string{"status": "up"}})
	stateDb.SetEntry(ts, db.Key{Comp: []string{"Ethernet999"}}, db.Value{Field: map[string]string{"status": "up"}})

	origPopulate := populatePortCounters
	populatePortCounters = mockPopulateCounters
	defer func() { populatePortCounters = origPopulate }()

	device := &ocbinds.Device{
		Interfaces: &ocbinds.OpenconfigInterfaces_Interfaces{},
	}
	ygot.BuildEmptyTree(device)
	var root ygot.GoStruct = device

	inParams := XfmrParams{
		ygRoot: &root,
		uri:    "/openconfig-interfaces:interfaces/interface[name=" + lagName + "]/subinterfaces/subinterface[index=0]/ipv4/state/counters",
		dbs:    [db.MaxDB]*db.DB{db.StateDB: stateDb},
	}

	err = DbToYang_intf_ipv4_counters_xfmr(inParams)
	if err != nil {
		t.Fatalf("IPv4 Transformer failed: %v", err)
	}

	intf := device.Interfaces.Interface[lagName]
	if intf == nil || intf.Subinterfaces == nil || intf.Subinterfaces.Subinterface[0] == nil {
		t.Fatal("Subinterface[0] was not created in device tree")
	}

	v4Counters := intf.Subinterfaces.Subinterface[0].Ipv4.State.Counters
	if v4Counters == nil || v4Counters.InPkts == nil {
		t.Fatal("InPkts was not populated")
	}

	if *v4Counters.InPkts != 150 {
		t.Errorf("Aggregation error: Expected 150 (100+50), got %v", *v4Counters.InPkts)
	}
}

func TestCPU_DbToYang_intf_ipv6_counters_xfmr_Ethernet(t *testing.T) {
	stateDb, _ := db.NewDB(getDBOptions(db.StateDB))
	defer stateDb.DeleteDB()

	origPopulate := populatePortCounters
	populatePortCounters = mockPopulateCounters
	defer func() { populatePortCounters = origPopulate }()

	device := &ocbinds.Device{
		Interfaces: &ocbinds.OpenconfigInterfaces_Interfaces{},
	}
	ygot.BuildEmptyTree(device)
	var root ygot.GoStruct = device

	inParams := XfmrParams{
		ygRoot: &root,
		uri:    "/openconfig-interfaces:interfaces/interface[name=Ethernet999]/subinterfaces/subinterface[index=0]/ipv6/state/counters",
		dbs:    [db.MaxDB]*db.DB{},
	}
	inParams.dbs[db.StateDB] = stateDb

	err := DbToYang_intf_ipv6_counters_xfmr(inParams)
	if err != nil {
		t.Fatalf("IPv6 Transformer failed: %v", err)
	}

	intf := device.Interfaces.Interface["Ethernet999"]
	if intf == nil || intf.Subinterfaces == nil || intf.Subinterfaces.Subinterface[0] == nil {
		t.Fatal("Subinterface[0] was not created in device tree")
	}

	v6Counters := intf.Subinterfaces.Subinterface[0].Ipv6.State.Counters
	if v6Counters == nil || v6Counters.InPkts == nil || *v6Counters.InPkts != 200 {
		t.Errorf("Expected 200, got %v", v6Counters)
	}
}

func TestCPU_DbToYang_intf_mgmt_xfmr(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		want      map[string]interface{}
		expectErr bool
	}{
		{
			name:      "CPU Interface",
			uri:       "/openconfig-interfaces:interfaces/interface[name=CPU]/state/management",
			want:      map[string]interface{}{"management": false},
			expectErr: false,
		},
		{
			name:      "Management Interface - eth prefix",
			uri:       "/openconfig-interfaces:interfaces/interface[name=eth0]/state/management",
			want:      map[string]interface{}{"management": true},
			expectErr: false,
		},
		{
			name:      "Management Interface - mgmt prefix",
			uri:       "/openconfig-interfaces:interfaces/interface[name=mgmt0]/state/management",
			want:      map[string]interface{}{"management": true},
			expectErr: false,
		},
		{
			name:      "Standard Interface",
			uri:       "/openconfig-interfaces:interfaces/interface[name=Ethernet999]/state/management",
			want:      map[string]interface{}{},
			expectErr: false,
		},
		{
			name:      "Missing Name Key",
			uri:       "/openconfig-interfaces:interfaces/interface/state/management",
			want:      map[string]interface{}{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{uri: tt.uri}
			got, err := DbToYang_intf_mgmt_xfmr(inParams)

			if (err != nil) != tt.expectErr {
				t.Fatalf("DbToYang_intf_mgmt_xfmr() error = %v, expectErr %v", err, tt.expectErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DbToYang_intf_mgmt_xfmr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCPU_DbToYang_intf_eth_mac_address_xfmr(t *testing.T) {
	configDb, err := db.NewDB(getDBOptions(db.ConfigDB))
	if err != nil {
		t.Fatalf("Failed to open ConfigDB: %v", err)
	}
	defer configDb.DeleteDB()

	dbs := [db.MaxDB]*db.DB{db.ConfigDB: configDb}

	intfExplicit := "Ethernet998"
	intfFallback := "Ethernet999"

	tsPort := &db.TableSpec{Name: "PORT"}
	tsMeta := &db.TableSpec{Name: "DEVICE_METADATA"}

	// 1. Explicit MAC on Ethernet998
	configDb.SetEntry(tsPort, db.Key{Comp: []string{intfExplicit}}, db.Value{Field: map[string]string{"mac-address": "00:11:22:33:44:55"}})

	// 2. Seed Ethernet999
	configDb.SetEntry(tsPort, db.Key{Comp: []string{intfFallback}}, db.Value{Field: map[string]string{"admin_status": "up"}})

	// 3. System-assigned MAC fallback entry
	configDb.SetEntry(tsMeta, db.Key{Comp: []string{"localhost"}}, db.Value{Field: map[string]string{"mac": "AA:BB:CC:DD:EE:FF"}})

	defer func() {
		configDb.DeleteEntry(tsPort, db.Key{Comp: []string{intfExplicit}})
		configDb.DeleteEntry(tsPort, db.Key{Comp: []string{intfFallback}})
		configDb.DeleteEntry(tsMeta, db.Key{Comp: []string{"localhost"}})
	}()

	tests := []struct {
		name   string
		ifName string
		want   string
	}{
		{"Explicit", intfExplicit, "00:11:22:33:44:55"},
		{"Fallback", intfFallback, "AA:BB:CC:DD:EE:FF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				uri:   "/interfaces/interface[name=" + tt.ifName + "]/ethernet/state/mac-address",
				curDb: db.ConfigDB,
				dbs:   dbs,
			}
			res, err := DbToYang_intf_eth_mac_address_xfmr(inParams)
			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.name, err)
			}
			if res["mac-address"] != tt.want {
				t.Errorf("Expected %s, got %v", tt.want, res["mac-address"])
			}
		})
	}
}

func TestCPU_SumV4Counters_AllFields(t *testing.T) {
	u64 := func(v uint64) *uint64 { return &v }

	// Test Case A: Parent already has non-nil values
	parent1 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters{
		InDiscardedPkts:    u64(10),
		InErrorPkts:        u64(10),
		InForwardedOctets:  u64(10),
		InForwardedPkts:    u64(10),
		InOctets:           u64(10),
		InPkts:             u64(10),
		OutDiscardedPkts:   u64(10),
		OutErrorPkts:       u64(10),
		OutForwardedOctets: u64(10),
		OutForwardedPkts:   u64(10),
		OutOctets:          u64(10),
		OutPkts:            u64(10),
	}

	member1 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters{
		InDiscardedPkts:    u64(5),
		InErrorPkts:        u64(5),
		InForwardedOctets:  u64(5),
		InForwardedPkts:    u64(5),
		InOctets:           u64(5),
		InPkts:             u64(5),
		OutDiscardedPkts:   u64(5),
		OutErrorPkts:       u64(5),
		OutForwardedOctets: u64(5),
		OutForwardedPkts:   u64(5),
		OutOctets:          u64(5),
		OutPkts:            u64(5),
	}

	sumV4Counters(parent1, member1)

	if *parent1.InDiscardedPkts != 15 || *parent1.OutOctets != 15 {
		t.Errorf("sumV4Counters failed addition: got InDiscardedPkts=%d, OutOctets=%d", *parent1.InDiscardedPkts, *parent1.OutOctets)
	}

	// Test Case B: Parent has nil values
	parent2 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters{}

	sumV4Counters(parent2, member1)

	if *parent2.InDiscardedPkts != 5 || *parent2.OutOctets != 5 {
		t.Errorf("sumV4Counters failed assignment: got InDiscardedPkts=%d, OutOctets=%d", *parent2.InDiscardedPkts, *parent2.OutOctets)
	}
}

func TestCPU_SumV6Counters_AllFields(t *testing.T) {
	u64 := func(v uint64) *uint64 { return &v }

	// Test Case A: Parent already has non-nil values
	parent1 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters{
		InDiscardedPkts:    u64(20),
		InErrorPkts:        u64(20),
		InForwardedOctets:  u64(20),
		InForwardedPkts:    u64(20),
		InOctets:           u64(20),
		InPkts:             u64(20),
		OutDiscardedPkts:   u64(20),
		OutErrorPkts:       u64(20),
		OutForwardedOctets: u64(20),
		OutForwardedPkts:   u64(20),
		OutOctets:          u64(20),
		OutPkts:            u64(20),
	}

	member1 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters{
		InDiscardedPkts:    u64(10),
		InErrorPkts:        u64(10),
		InForwardedOctets:  u64(10),
		InForwardedPkts:    u64(10),
		InOctets:           u64(10),
		InPkts:             u64(10),
		OutDiscardedPkts:   u64(10),
		OutErrorPkts:       u64(10),
		OutForwardedOctets: u64(10),
		OutForwardedPkts:   u64(10),
		OutOctets:          u64(10),
		OutPkts:            u64(10),
	}

	sumV6Counters(parent1, member1)

	if *parent1.InErrorPkts != 30 || *parent1.OutForwardedPkts != 30 {
		t.Errorf("sumV6Counters failed addition: got InErrorPkts=%d, OutForwardedPkts=%d", *parent1.InErrorPkts, *parent1.OutForwardedPkts)
	}

	// Test Case B: Parent has nil values
	parent2 := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters{}

	sumV6Counters(parent2, member1)

	if *parent2.InErrorPkts != 10 || *parent2.OutForwardedPkts != 10 {
		t.Errorf("sumV6Counters failed assignment: got InErrorPkts=%d, OutForwardedPkts=%d", *parent2.InErrorPkts, *parent2.OutForwardedPkts)
	}
}

// 3. Test getSpecificCounterAttr for all subinterface IPv4/IPv6 counter path cases and invalid type assertions
func TestCPU_GetSpecificCounterAttr_Subinterface(t *testing.T) {
	entry := &db.Value{
		Field: map[string]string{
			"SAI_PORT_STAT_IP_IN_RECEIVES":          "100",
			"SAI_PORT_STAT_IP_OUT_UCAST_PKTS":       "200",
			"SAI_PORT_STAT_IP_OUT_NON_UCAST_PKTS":   "50",
			"SAI_PORT_STAT_IPV6_IN_RECEIVES":        "300",
			"SAI_PORT_STAT_IPV6_OUT_UCAST_PKTS":     "400",
			"SAI_PORT_STAT_IPV6_OUT_NON_UCAST_PKTS": "50",
			"SAI_PORT_STAT_IPV6_IN_DISCARDS":        "10",
			"SAI_PORT_STAT_IPV6_OUT_DISCARDS":       "20",
		},
	}

	v4Counter := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters{}
	v6Counter := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters{}

	pathsV4 := []string{
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/state/counters/out-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/state/counters/out-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/state/counters/in-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/state/counters/in-pkts",
	}

	for _, p := range pathsV4 {
		status, err := getSpecificCounterAttr(p, entry, v4Counter)
		if !status || err != nil {
			t.Errorf("getSpecificCounterAttr failed for path %s: err=%v", p, err)
		}
	}

	pathsV6 := []string{
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/state/counters/out-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state/counters/out-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/state/counters/in-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state/counters/in-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/state/counters/in-discarded-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state/counters/in-discarded-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/state/counters/out-discarded-pkts",
		"/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state/counters/out-discarded-pkts",
	}

	for _, p := range pathsV6 {
		status, err := getSpecificCounterAttr(p, entry, v6Counter)
		if !status || err != nil {
			t.Errorf("getSpecificCounterAttr failed for path %s: err=%v", p, err)
		}
	}

	// Default fallback path
	status, err := getSpecificCounterAttr("/openconfig-interfaces:interfaces/invalid/path", entry, v4Counter)
	if !status || err != nil {
		t.Errorf("getSpecificCounterAttr expected true, nil for invalid path, got status=%v, err=%v", status, err)
	}

	// Invalid type assertion branches in first switch
	getSpecificCounterAttr("/openconfig-interfaces:interfaces/interface/state/counters", entry, "invalid_type")
	getSpecificCounterAttr("/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/state/counters", entry, "invalid_type")
	getSpecificCounterAttr("/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state/counters", entry, "invalid_type")
	getSpecificCounterAttr("/openconfig-interfaces:interfaces/interface/ethernet/state/counters", entry, "invalid_type")
}

func TestCPU_DbToYang_intf_oper_status_xfmr(t *testing.T) {
	applDb, err := db.NewDB(getDBOptions(db.ApplDB))
	if err != nil {
		t.Fatalf("Failed to create ApplDB: %v", err)
	}
	defer applDb.DeleteDB()

	tests := []struct {
		name       string
		ifName     string
		tblName    string
		dbFields   map[string]string
		wantStatus string
		expectErr  bool
	}{
		{
			name:       "Oper_Status_Up",
			ifName:     "Ethernet0",
			tblName:    "PORT_TABLE",
			dbFields:   map[string]string{"oper_status": "up"},
			wantStatus: "UP",
		},
		{
			name:       "Oper_Status_Down_Branch",
			ifName:     "Ethernet0",
			tblName:    "PORT_TABLE",
			dbFields:   map[string]string{"oper_status": "down", "presence": "1"},
			wantStatus: "DOWN",
		},
		{
			name:       "Oper_Status_Unknown_Default_Branch",
			ifName:     "Ethernet0",
			tblName:    "PORT_TABLE",
			dbFields:   map[string]string{"oper_status": "some_invalid_state", "presence": "1"},
			wantStatus: "UNKNOWN",
		},
		{
			name:       "CPU_Interface_Always_Up",
			ifName:     "CPU",
			wantStatus: "UP",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tblName != "" {
				ts := &db.TableSpec{Name: tt.tblName}
				applDb.DeleteEntry(ts, db.Key{Comp: []string{tt.ifName}})
				if tt.dbFields != nil {
					applDb.SetEntry(ts, db.Key{Comp: []string{tt.ifName}}, db.Value{Field: tt.dbFields})
				}
				defer applDb.DeleteEntry(ts, db.Key{Comp: []string{tt.ifName}})
			}

			inParams := XfmrParams{
				key:   tt.ifName,
				uri:   "/openconfig-interfaces:interfaces/interface[name=" + tt.ifName + "]/state/oper-status",
				curDb: db.ApplDB,
				dbs:   [db.MaxDB]*db.DB{db.ApplDB: applDb},
			}

			got, err := DbToYang_intf_oper_status_xfmr(inParams)

			if (err != nil) != tt.expectErr {
				t.Fatalf("DbToYang_intf_oper_status_xfmr() error = %v, expectErr %v", err, tt.expectErr)
			}

			if !tt.expectErr {
				if status, ok := got["oper-status"]; !ok || status != tt.wantStatus {
					t.Errorf("got status %v, want %v", status, tt.wantStatus)
				}
			}
		})
	}
}

func TestCPU_PopulatePortCounters(t *testing.T) {
	countersDb, err := db.NewDB(getDBOptions(db.CountersDB))
	if err != nil {
		t.Fatalf("Failed to create CountersDB: %v", err)
	}
	defer countersDb.DeleteDB()

	tsPortMap := &db.TableSpec{Name: "COUNTERS_PORT_NAME_MAP"}
	mapKey := db.Key{Comp: []string{""}}

	// 1. Read and preserve existing mappings (e.g. Ethernet0) so other tests are not broken
	origMapEntry, _ := countersDb.GetEntry(tsPortMap, mapKey)
	mergedFields := make(map[string]string)
	if origMapEntry.IsPopulated() {
		for k, v := range origMapEntry.Field {
			mergedFields[k] = v
		}
	}
	// Add our test interface
	mergedFields["Ethernet998"] = "oid:0x1000000000001"
	countersDb.SetEntry(tsPortMap, mapKey, db.Value{Field: mergedFields})

	// 2. Set test counters entry
	tsCounters := &db.TableSpec{Name: "COUNTERS"}
	testCounterKey := db.Key{Comp: []string{"oid:0x1000000000001"}}
	countersDb.SetEntry(tsCounters, testCounterKey, db.Value{
		Field: map[string]string{
			"SAI_PORT_STAT_IP_IN_RECEIVES":          "150",
			"SAI_PORT_STAT_IP_OUT_UCAST_PKTS":       "200",
			"SAI_PORT_STAT_IP_OUT_NON_UCAST_PKTS":   "50",
			"SAI_PORT_STAT_IPV6_IN_RECEIVES":        "300",
			"SAI_PORT_STAT_IPV6_OUT_UCAST_PKTS":     "400",
			"SAI_PORT_STAT_IPV6_OUT_NON_UCAST_PKTS": "100",
			"SAI_PORT_STAT_IPV6_IN_DISCARDS":        "20",
			"SAI_PORT_STAT_IPV6_OUT_DISCARDS":       "30",
		},
	})

	// 3. Precise cleanup: restore original map and only delete the specific counter entry
	t.Cleanup(func() {
		countersDb.DeleteEntry(tsCounters, testCounterKey)
		if origMapEntry.IsPopulated() {
			countersDb.SetEntry(tsPortMap, mapKey, origMapEntry)
		} else {
			countersDb.DeleteEntry(tsPortMap, mapKey)
		}
	})

	dbs := [db.MaxDB]*db.DB{db.CountersDB: countersDb}

	tests := []struct {
		name        string
		ifNameArg   string
		uri         string
		getCounter  func() interface{}
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, counter interface{})
	}{
		{
			name:      "Success - IPv4 Subinterface Counters with Explicit ifName",
			ifNameArg: "Ethernet998",
			uri:       "/openconfig-interfaces:interfaces/interface[name=Ethernet998]/subinterfaces/subinterface[index=0]/ipv4/state/counters",
			getCounter: func() interface{} {
				c := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters{}
				ygot.BuildEmptyTree(c)
				return c
			},
			expectError: false,
			validate: func(t *testing.T, counter interface{}) {
				v4 := counter.(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_State_Counters)
				if v4.InPkts == nil || *v4.InPkts != 150 {
					t.Errorf("Expected InPkts 150, got %v", v4.InPkts)
				}
				if v4.OutPkts == nil || *v4.OutPkts != 250 {
					t.Errorf("Expected OutPkts 250, got %v", v4.OutPkts)
				}
			},
		},
		{
			name:      "Success - IPv6 Subinterface Counters with URI Fallback ifName",
			ifNameArg: "",
			uri:       "/openconfig-interfaces:interfaces/interface[name=Ethernet998]/subinterfaces/subinterface[index=0]/ipv6/state/counters",
			getCounter: func() interface{} {
				c := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters{}
				ygot.BuildEmptyTree(c)
				return c
			},
			expectError: false,
			validate: func(t *testing.T, counter interface{}) {
				v6 := counter.(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_State_Counters)
				if v6.InPkts == nil || *v6.InPkts != 300 {
					t.Errorf("Expected InPkts 300, got %v", v6.InPkts)
				}
				if v6.OutPkts == nil || *v6.OutPkts != 500 {
					t.Errorf("Expected OutPkts 500, got %v", v6.OutPkts)
				}
				if v6.InDiscardedPkts == nil || *v6.InDiscardedPkts != 20 {
					t.Errorf("Expected InDiscardedPkts 20, got %v", v6.InDiscardedPkts)
				}
				if v6.OutDiscardedPkts == nil || *v6.OutDiscardedPkts != 30 {
					t.Errorf("Expected OutDiscardedPkts 30, got %v", v6.OutDiscardedPkts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrParams{
				uri:   tt.uri,
				curDb: db.CountersDB,
				dbs:   dbs,
			}

			counterObj := tt.getCounter()
			err := populatePortCounters(inParams, tt.ifNameArg, counterObj)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, counterObj)
			}
		})
	}
}
