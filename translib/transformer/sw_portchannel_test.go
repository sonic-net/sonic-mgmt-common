package transformer

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func backupIntfTypeTblMap() map[E_InterfaceType]IntfTblData {
	backup := make(map[E_InterfaceType]IntfTblData)
	for k, v := range IntfTypeTblMap {
		backup[k] = v
	}
	return backup
}

func restoreIntfTypeTblMap(original map[E_InterfaceType]IntfTblData) {
	for k, v := range original {
		IntfTypeTblMap[k] = v
	}
}

func setupLagTest() *db.DB {
	data := IntfTypeTblMap[IntfTypePortChannel]
	data.cfgDb = TblData{portTN: "PORTCHANNEL", intfTN: "PORTCHANNEL_INTERFACE", memberTN: "PORTCHANNEL_MEMBER", keySep: "|"}
	data.appDb = TblData{portTN: "LAG_TABLE", intfTN: "INTF_TABLE", keySep: ":", memberTN: "LAG_MEMBER_TABLE"}
	IntfTypeTblMap[IntfTypePortChannel] = data

	d, _ := db.NewDB(db.Options{
		DBNo: db.ConfigDB, InitIndicator: "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|", KeySeparator: "|",
	})
	return d
}

func loadDataToExistingDb(d *db.DB, jsonStr string) {
	var data map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &data)
	for table, tableData := range data {
		ts := &db.TableSpec{Name: table}
		entries, _ := tableData.(map[string]interface{})
		for key, fields := range entries {
			fieldMap, _ := fields.(map[string]interface{})
			fMap := make(map[string]string)
			for f, v := range fieldMap {
				fMap[f] = fmt.Sprintf("%v", v)
			}
			d.SetEntry(ts, db.Key{Comp: []string{key}}, db.Value{Field: fMap})
		}
	}
}

func Test_Portchannel_GetLagState_new(t *testing.T) {
	ifName := "PortChannel_UT_99"
	ocVal := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_State{}
	lagInfoMap := map[string]db.Value{
		ifName: {Field: map[string]string{"min-links": "1", "lag-type": "LACP", "lag-speed": "10000"}},
	}
	err := getLagState(XfmrParams{}, nil, &ifName, lagInfoMap, ocVal)
	if err != nil {
		t.Errorf("getLagState failed: %v", err)
	}
	if ocVal.LagSpeed == nil || *ocVal.LagSpeed != 10000 {
		t.Error("Speed mismatch")
	}
}

func Test_Portchannel_Uint16Conv(t *testing.T) {
	if val, _ := uint16Conv("12"); val != 12 {
		t.Error("Fail")
	}
	if _, err := uint16Conv("string"); err == nil {
		t.Error("Expected error")
	}
}

func Test_Portchannel_DeleteLagIntfAndMembers(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()
	uniquePC := "PortChannel_UT_Delete"
	d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}}, db.Value{Field: map[string]string{"admin_status": "up"}})
	defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}})

	rMap := make(map[db.DBNum]map[string]map[string]db.Value)
	var redisMap RedisDbMap = rMap
	params := XfmrParams{d: d, dbs: [db.MaxDB]*db.DB{db.ConfigDB: d}, subOpDataMap: map[Operation]*RedisDbMap{DELETE: &redisMap}, oper: DELETE}
	deleteLagIntfAndMembers(&params, &uniquePC)
}

func Test_Portchannel_ValidatePortChannel(t *testing.T) {
	d := &db.DB{}
	t.Run("Invalid_Interface_Type", func(t *testing.T) {
		if err := validatePortChannel(d, "Ethernet_UT_1011"); err == nil {
			t.Error("Fail")
		}
	})
	t.Run("Garbage_Name", func(t *testing.T) {
		if err := validatePortChannel(d, "Invalid99!!"); err == nil {
			t.Error("Fail")
		}
	})
}

func Test_Portchannel_YangToDb_lag_min_links_xfmr(t *testing.T) {
	u16ptr := func(v uint16) *uint16 { return &v }
	uri := "/interfaces/interface[name=PortChannel_UT_Min]/aggregation/config/min-links"
	t.Run("Success_-Valid_MinLinks", func(t *testing.T) {
		res, _ := YangToDb_lag_min_links_xfmr(XfmrParams{uri: uri, param: u16ptr(5)})
		if res["min_links"] != "5" {
			t.Error("Fail")
		}
	})
	t.Run("Success-Boundary_0", func(t *testing.T) {
		res, _ := YangToDb_lag_min_links_xfmr(XfmrParams{uri: uri, param: u16ptr(0)})
		if res["min_links"] != "0" {
			t.Error("Fail")
		}
	})
	t.Run("Success-Boundary_32", func(t *testing.T) {
		res, _ := YangToDb_lag_min_links_xfmr(XfmrParams{uri: uri, param: u16ptr(32)})
		if res["min_links"] != "32" {
			t.Error("Fail")
		}
	})
	t.Run("Error-Param_is_Nil", func(t *testing.T) {
		YangToDb_lag_min_links_xfmr(XfmrParams{uri: uri, param: nil})
	})
	t.Run("Error-Value_Out_of_Range(33)", func(t *testing.T) {
		if _, err := YangToDb_lag_min_links_xfmr(XfmrParams{uri: uri, param: u16ptr(33)}); err == nil {
			t.Error("Fail")
		}
	})
}

func Test_Portchannel_DbToYang_lag_min_links_xfmr(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()

	t.Run("Success_-Value_from_DB", func(t *testing.T) {
		key := "PortChannel_UT_DB1"
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}}, db.Value{Field: map[string]string{"min_links": "8"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}})
		dbMap := map[db.DBNum]map[string]map[string]db.Value{db.ConfigDB: {"PORTCHANNEL": {key: {Field: map[string]string{"min_links": "8"}}}}}
		res, _ := DbToYang_lag_min_links_xfmr(XfmrParams{d: d, key: key, curDb: db.ConfigDB, dbDataMap: &dbMap})
		if res["min-links"] != uint16(8) {
			t.Error("Fail")
		}
	})
	t.Run("Success-Default_Value_Path", func(t *testing.T) {
		key := "PortChannel_UT_DB2"
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}}, db.Value{Field: map[string]string{"admin": "up"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}})
		dbMap := map[db.DBNum]map[string]map[string]db.Value{db.ConfigDB: {"PORTCHANNEL": {key: {Field: map[string]string{"admin": "up"}}}}}
		res, _ := DbToYang_lag_min_links_xfmr(XfmrParams{d: d, key: key, curDb: db.ConfigDB, dbDataMap: &dbMap})
		if res["min-links"] != uint16(1) {
			t.Error("Fail")
		}
	})
	t.Run("Error-Conversion_Failure", func(t *testing.T) {
		key := "PortChannel_UT_DB3"
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}}, db.Value{Field: map[string]string{"min_links": "abc"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{key}})
		dbMap := map[db.DBNum]map[string]map[string]db.Value{db.ConfigDB: {"PORTCHANNEL": {key: {Field: map[string]string{"min_links": "abc"}}}}}
		if _, err := DbToYang_lag_min_links_xfmr(XfmrParams{d: d, key: key, curDb: db.ConfigDB, dbDataMap: &dbMap}); err == nil {
			t.Error("Fail")
		}
	})
	t.Run("Error-Validation_Failure(Not_in_DB)", func(t *testing.T) {
		if _, err := DbToYang_lag_min_links_xfmr(XfmrParams{d: d, key: "PortChannel_Invalid", curDb: db.ConfigDB}); err == nil {
			t.Error("Fail")
		}
	})
}

func Test_Portchannel_DbToYang_intf_lag_state_xfmr(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	if d == nil {
		t.Fatal("Failed to initialize DB")
	}
	defer d.DeleteDB()

	uniquePC := "PortChannel_UT_Subtree"

	// Ensure a clean start for this specific PortChannel
	d.DeleteTable(&db.TableSpec{Name: "PORTCHANNEL_MEMBER|" + uniquePC})

	setupData := fmt.Sprintf(`{
		"PORTCHANNEL": {"%s": {"min_links": "1", "setup.runner_name": "LACP"}},
		"PORTCHANNEL_MEMBER|%s": {"Ethernet_UT_S1": {"runner.selected": "true", "link.speed": "1000"}}
	}`, uniquePC, uniquePC)
	loadDataToExistingDb(d, setupData)

	tests := []struct {
		name    string
		uri     string
		isNil   bool // If true, we trigger the "Failed to Get root object" branch safely
		wantErr bool
	}{
		{"Container", "/openconfig-interfaces:interfaces/interface[name=" + uniquePC + "]/openconfig-if-aggregate:aggregation/state", false, false},
		{"MinLinks", "/openconfig-interfaces:interfaces/interface[name=" + uniquePC + "]/openconfig-if-aggregate:aggregation/state/min-links", false, false},
		{"LagType", "/openconfig-interfaces:interfaces/interface[name=" + uniquePC + "]/openconfig-if-aggregate:aggregation/state/lag-type", false, false},
		{"Member", "/openconfig-interfaces:interfaces/interface[name=" + uniquePC + "]/openconfig-if-aggregate:aggregation/state/member", false, false},
		{"LagSpeed", "/openconfig-interfaces:interfaces/interface[name=" + uniquePC + "]/openconfig-if-aggregate:aggregation/state/lag-speed", false, false},
		{"Error_NilRoot", "/aggregation/state", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceRoot := &ocbinds.Device{}

			// If we are NOT testing the nil error, initialize the sub-structures
			if !tt.isNil {
				deviceRoot.Interfaces = &ocbinds.OpenconfigInterfaces_Interfaces{}
				deviceRoot.Interfaces.Interface = make(map[string]*ocbinds.OpenconfigInterfaces_Interfaces_Interface)
			}

			var ygRootIntf ygot.GoStruct = deviceRoot
			params := XfmrParams{
				d:      d,
				uri:    tt.uri,
				ygRoot: &ygRootIntf,
				dbs:    [db.MaxDB]*db.DB{db.ConfigDB: d},
			}

			err := DbToYang_intf_lag_state_xfmr(params)

			if (err != nil) != tt.wantErr {
				t.Fatalf("%s: Expected error = %v, got %v", tt.name, tt.wantErr, err)
			}

			// Verification for the Success Container case
			if tt.name == "Container" && !tt.wantErr {
				intf := deviceRoot.Interfaces.Interface[uniquePC]
				if intf == nil || intf.Aggregation == nil || intf.Aggregation.State == nil {
					t.Errorf("Aggregation state was not built correctly")
					return
				}
				if intf.Aggregation.State.LagSpeed == nil || *intf.Aggregation.State.LagSpeed != 1000 {
					t.Errorf("LagSpeed expected 1000, got %v", intf.Aggregation.State.LagSpeed)
				}
			}
		})
	}
}

func Test_Portchannel_DbToYangPath_intf_lag_state_path_xfmr(t *testing.T) {
	tests := []struct {
		name       string
		tblName    string
		tblKeyComp []string
	}{
		{"Valid_PortChannel_Table", "PORTCHANNEL", []string{"PortChannel_UT_P1", "dummy"}},
		{"Valid_PortChannel_Member_Table", "PORTCHANNEL_MEMBER", []string{"PortChannel_UT_P2", "Ethernet_UT_P2"}},
		{"Branch_-Wrong_Table_Name", "VLAN_TABLE", []string{"Vlan10"}},
		{"Branch-PortChannel_with_No_Key_Components", "PORTCHANNEL", []string{}},
		{"Branch-PortChannel_Member_with_Insufficient_Keys", "PORTCHANNEL_MEMBER", []string{"PortChannel_UT_P3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := XfmrDbToYgPathParams{tblName: tt.tblName, tblKeyComp: tt.tblKeyComp, ygPathKeys: make(map[string]string)}
			DbToYangPath_intf_lag_state_path_xfmr(params)
		})
	}
}

func Test_Portchannel_YangToDb_lag_type_xfmr(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()
	uniquePC := "PortChannel_UT_Type"
	t.Run("Branch-Nil_Param", func(t *testing.T) {
		YangToDb_lag_type_xfmr(XfmrParams{d: d, param: nil})
	})
	t.Run("Branch-New_PortChannel(Not_in_DB)", func(t *testing.T) {
		YangToDb_lag_type_xfmr(XfmrParams{d: d, uri: "/interfaces/interface[name=PortChannel_New]/aggregation/config/lag-type", param: ocbinds.OpenconfigIfAggregate_AggregationType_LACP})
	})
	t.Run("Branch_-Existing_PortChannel_Match", func(t *testing.T) {
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}}, db.Value{Field: map[string]string{"lag_type": "LACP"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}})
		YangToDb_lag_type_xfmr(XfmrParams{d: d, uri: "/interfaces/interface[name=" + uniquePC + "]/aggregation/config/lag-type", param: ocbinds.OpenconfigIfAggregate_AggregationType_LACP})
	})
	t.Run("Branch-Conflict_Error(LACP_to_STATIC)", func(t *testing.T) {
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}}, db.Value{Field: map[string]string{"lag_type": "LACP"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}})
		if _, err := YangToDb_lag_type_xfmr(XfmrParams{d: d, uri: "/interfaces/interface[name=" + uniquePC + "]/aggregation/config/lag-type", param: ocbinds.OpenconfigIfAggregate_AggregationType_STATIC}); err == nil {
			t.Error("Fail")
		}
	})
	t.Run("Branch_-Conflict_Error(STATIC_to_LACP)", func(t *testing.T) {
		d.SetEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}}, db.Value{Field: map[string]string{"lag_type": "STATIC"}})
		defer d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}})
		if _, err := YangToDb_lag_type_xfmr(XfmrParams{d: d, uri: "/interfaces/interface[name=" + uniquePC + "]/aggregation/config/lag-type", param: ocbinds.OpenconfigIfAggregate_AggregationType_LACP}); err == nil {
			t.Error("Fail")
		}
	})
}

func Test_Portchannel_DbToYang_lag_type_xfmr(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()
	dbMap := map[db.DBNum]map[string]map[string]db.Value{db.ConfigDB: {"PORTCHANNEL": {
		"PortChannel_T1": {Field: map[string]string{"lag_type": "STATIC"}},
		"PortChannel_T2": {Field: map[string]string{"lag_type": "LACP"}},
		"PortChannel_T3": {Field: map[string]string{"admin": "up"}},
	}}}
	loadDataToExistingDb(d, `{"PORTCHANNEL": {"PortChannel_T1": {"lag_type": "STATIC"}, "PortChannel_T2": {"lag_type": "LACP"}, "PortChannel_T3": {"admin": "up"}}}`)
	defer func() {
		d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{"PortChannel_T1"}})
		d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{"PortChannel_T2"}})
		d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{"PortChannel_T3"}})
	}()
	t.Run("Success_-STATIC_Mode", func(t *testing.T) {
		DbToYang_lag_type_xfmr(XfmrParams{d: d, key: "PortChannel_T1", curDb: db.ConfigDB, dbDataMap: &dbMap})
	})
	t.Run("Success-LACP_Mode", func(t *testing.T) {
		DbToYang_lag_type_xfmr(XfmrParams{d: d, key: "PortChannel_T2", curDb: db.ConfigDB, dbDataMap: &dbMap})
	})
	t.Run("Success-Missing_Field_Defaults_to_LACP", func(t *testing.T) {
		DbToYang_lag_type_xfmr(XfmrParams{d: d, key: "PortChannel_T3", curDb: db.ConfigDB, dbDataMap: &dbMap})
	})
	t.Run("Error-Validation_Failure(Not_in_DB)", func(t *testing.T) {
		DbToYang_lag_type_xfmr(XfmrParams{d: d, key: "PortChannel_Invalid", curDb: db.ConfigDB, dbDataMap: &dbMap})
	})
}

func Test_Portchannel_Subscribe_intf_lag_state_xfmr(t *testing.T) {
	const (
		SpecificPC = "PortChannel_UT_Sub" // Unique Name
		ConfigDB   = db.ConfigDB
		StateDB    = db.StateDB
	)

	tests := []struct {
		name      string
		uri       string
		subscProc SubscProcType
		expectDB  db.DBNum
		expectTbl string
	}{
		// 1. SPECIFIC NAME SCENARIOS
		{"specific_config_container", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/config", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_config_lag_type", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/config/lag-type", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_config_min_links", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/config/min-links", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_state_container", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/state", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_state_lag_type", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/state/lag-type", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_state_min_links", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/state/min-links", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"specific_state_lag_speed", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/state/lag-speed", TRANSLATE_SUBSCRIBE, StateDB, "LAG_MEMBER_TABLE"},
		{"specific_state_member", "/openconfig-interfaces:interfaces/interface[name=" + SpecificPC + "]/openconfig-if-aggregate:aggregation/state/member", TRANSLATE_SUBSCRIBE, StateDB, "LAG_MEMBER_TABLE"},

		// 2. WILDCARD SCENARIOS ([name=*])
		{"wildcard_config_container", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/config", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_config_lag_type", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/config/lag-type", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_config_min_links", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/config/min-links", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_state_container", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/state", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_state_lag_type", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/state/lag-type", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_state_min_links", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/state/min-links", TRANSLATE_SUBSCRIBE, ConfigDB, "PORTCHANNEL"},
		{"wildcard_state_lag_speed", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/state/lag-speed", TRANSLATE_SUBSCRIBE, StateDB, "LAG_MEMBER_TABLE"},
		{"wildcard_state_member", "/openconfig-interfaces:interfaces/interface[name=*]/openconfig-if-aggregate:aggregation/state/member", TRANSLATE_SUBSCRIBE, StateDB, "LAG_MEMBER_TABLE"},
		{"wildcard_state_member_empty_ifname", "/openconfig-interfaces:interfaces/interface[name=]/openconfig-if-aggregate:aggregation/state/member", TRANSLATE_SUBSCRIBE, StateDB, "LAG_MEMBER_TABLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inParams := XfmrSubscInParams{uri: tt.uri, subscProc: tt.subscProc}
			result, err := Subscribe_intf_lag_state_xfmr(inParams)
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

			// 1. Verify DB
			dbMap, ok := result.secDbDataMap[tt.expectDB]
			if !ok {
				t.Fatalf("Expected DB %v not found", tt.expectDB)
			}

			// 2. Verify Table
			if _, ok := dbMap[tt.expectTbl]; !ok {
				t.Errorf("Table %s not found in DB %v", tt.expectTbl, tt.expectDB)
			}

			// 3. Verify Key Pattern logic
			if strings.Contains(tt.name, "wildcard") {
				if tt.expectDB == ConfigDB {
					if _, ok := dbMap[tt.expectTbl]["*"]; !ok {
						t.Error("Wildcard '*' missing")
					}
				} else {
					if _, ok := dbMap[tt.expectTbl]["*|*"]; !ok {
						t.Error("Wildcard '*|*' missing")
					}
				}
			} else {
				if tt.expectDB == ConfigDB {
					if _, ok := dbMap[tt.expectTbl][SpecificPC]; !ok {
						t.Errorf("Key %s missing", SpecificPC)
					}
				}
			}
		})
	}
}

func Test_Portchannel_fillLagInfoForIntf(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()

	tests := []struct {
		name      string
		ifName    string
		setupJSON string
		wantErr   bool
		errStr    string
	}{
		{
			name:   "Success_Full_Active_Members",
			ifName: "PortChannel_UT_Fill1",
			setupJSON: `{
				"PORTCHANNEL": {"PortChannel_UT_Fill1": {"min_links": "2", "setup.runner_name": "STATIC"}},
				"PORTCHANNEL_MEMBER|PortChannel_UT_Fill1": {"Ethernet_UT_F1": {"runner.selected": "true", "link.speed": "1000"}}
			}`,
			wantErr: false,
		},
		{
			name:   "Success_Defaults_And_Unselected_Member",
			ifName: "PortChannel_UT_Fill2",
			setupJSON: `{
				"PORTCHANNEL": {"PortChannel_UT_Fill2": {"admin_status": "up"}},
				"PORTCHANNEL_MEMBER|PortChannel_UT_Fill2": {"Ethernet_UT_F2": {"runner.selected": "false", "link.speed": "1000"}}
			}`,
			wantErr: false,
		},
		{
			name:   "Error_Member_Speed_Conversion",
			ifName: "PortChannel_UT_Fill3",
			setupJSON: `{
				"PORTCHANNEL": {"PortChannel_UT_Fill3": {}},
				"PORTCHANNEL_MEMBER|PortChannel_UT_Fill3": {"Ethernet_UT_F3": {"runner.selected": "true", "link.speed": "invalid"}}
			}`,
			wantErr: true,
		},
		{
			name:   "Error_MinLinks_Conversion",
			ifName: "PortChannel_UT_Fill4",
			setupJSON: `{
				"PORTCHANNEL": {"PortChannel_UT_Fill4": {"min_links": "abc"}}
			}`,
			wantErr: true,
		},
		{
			name:      "Error_PortChannel_Entry_Missing",
			ifName:    "PortChannel_UT_Missing",
			setupJSON: `{}`,
			wantErr:   true,
			errStr:    "Failed to Get PortChannel details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadDataToExistingDb(d, tt.setupJSON)
			// Targeted Cleanup
			defer func() {
				d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{tt.ifName}})
				d.DeleteTable(&db.TableSpec{Name: "PORTCHANNEL_MEMBER|" + tt.ifName})
			}()

			lagMap := make(map[string]db.Value)
			err := fillLagInfoForIntf(XfmrParams{d: d, dbs: [db.MaxDB]*db.DB{db.ConfigDB: d}}, d, &tt.ifName, lagMap, nil)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Expected wantErr %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr && tt.errStr != "" && !strings.Contains(err.Error(), tt.errStr) {
				t.Errorf("Expected error %s, got %v", tt.errStr, err)
			}
		})
	}
}

func Test_Portchannel_updateMemberPortsMtu(t *testing.T) {
	orig := backupIntfTypeTblMap()
	defer restoreIntfTypeTblMap(orig)
	d := setupLagTest()
	defer d.DeleteDB()
	IntfTypeTblMap[IntfTypeEthernet] = IntfTblData{cfgDb: TblData{portTN: "PORT"}}

	uniquePC := "PortChannel_UT_MTU"
	uniqueEth := "Ethernet_UT_MTUM1"
	loadDataToExistingDb(d, fmt.Sprintf(`{"PORTCHANNEL": {"%s": {"admin_status": "up"}}, "PORTCHANNEL_MEMBER|%s": {"%s": {}}}`, uniquePC, uniquePC, uniqueEth))

	t.Run("Success_PropagateMTU", func(t *testing.T) {
		rMap := make(map[db.DBNum]map[string]map[string]db.Value)
		var redisMap RedisDbMap = rMap
		m := "9100"
		params := XfmrParams{
			d:            d,
			subOpDataMap: map[Operation]*RedisDbMap{UPDATE: &redisMap},
			dbs:          [db.MaxDB]*db.DB{db.ConfigDB: d},
		}
		updateMemberPortsMtu(&params, &uniquePC, &m)
	})

	t.Run("Error_ValidationFailed", func(t *testing.T) {
		invalidPC := "PortChannel_Invalid_99"
		m := "9100"
		params := XfmrParams{
			d:            d,
			subOpDataMap: make(map[Operation]*RedisDbMap),
		}
		err := updateMemberPortsMtu(&params, &invalidPC, &m)
		if err == nil {
			t.Errorf("Expected error for non-existent PortChannel")
		}
	})

	d.DeleteEntry(&db.TableSpec{Name: "PORTCHANNEL"}, db.Key{Comp: []string{uniquePC}})
	d.DeleteTable(&db.TableSpec{Name: "PORTCHANNEL_MEMBER|" + uniquePC})
	d.DeleteEntry(&db.TableSpec{Name: "PORT"}, db.Key{Comp: []string{uniqueEth}})
}
