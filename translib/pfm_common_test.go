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
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/transformer"
	"github.com/Workiva/go-datastructures/queue"
)

func TestPlatformCommonAppOwnership(t *testing.T) {
	for _, uri := range []string{
		"/openconfig-platform:components",
		"/openconfig-platform:components/component[name=Ethernet0]/transceiver/state",
		"/openconfig-platform:components/component[name=PSU0]/healthz/state",
	} {
		info, err := getAppModuleInfo(uri)
		if err != nil || info.appType != reflect.TypeOf(CommonApp{}) {
			t.Fatalf("platform request %q is not owned by CommonApp: %+v, %v", uri, info, err)
		}
	}
	for _, name := range []string{"openconfig-platform-healthz", "openconfig-platform-healthz-fault"} {
		count := 0
		for _, model := range getModels() {
			if model.Name == name {
				count++
			}
		}
		if count != 1 {
			t.Errorf("model %s capability count = %d, want 1", name, count)
		}
	}
}

func TestPlatformCommonAppQualifiedSubscriptions(t *testing.T) {
	app := &CommonApp{}
	for _, target := range []struct{ uri, table string }{
		{"/openconfig-platform:components/openconfig-platform:component[name=PSU0]/openconfig-platform-healthz:healthz/openconfig-platform-healthz:state", "COMPONENT_HEALTH_INFO"},
		{"/openconfig-platform:components/component[name=PSU0]/healthz/faults/fault[symptom=SYMPTOM_UNKNOWN]", "FAULT_INFO"},
	} {
		response, err := app.translateSubscribe(translateSubRequest{path: target.uri})
		if err != nil || len(response.ntfAppInfoTrgt) != 1 {
			t.Fatalf("translate %q: %+v, %v", target.uri, response, err)
		}
		info := response.ntfAppInfoTrgt[0]
		if info.table == nil || info.table.Name != target.table || !info.isOnChangeSupported || info.handlerFunc == nil {
			t.Fatalf("lost direct Healthz ON_CHANGE mapping: %+v", info)
		}
		key := db.Key{Comp: []string{"PSU0"}}
		if target.table == "FAULT_INFO" {
			key.Comp = append(key.Comp, "SYMPTOM_UNKNOWN")
		}
		resolved, err := app.processSubscribe(processSubRequest{path: info.path, table: info.table, key: &key})
		if err != nil || resolved.path == nil {
			t.Fatalf("Healthz notification dispatch: %+v, %v", resolved, err)
		}
	}
}

func TestPlatformCommonAppPreservesTransformerErrors(t *testing.T) {
	app := &CommonApp{pathInfo: NewPathInfo("/openconfig-platform:components")}
	qParams, _ := transformer.NewQueryParams(0, "", nil)
	var want error = errors.New("transformer DB failure")
	empty, err := app.composePlatformHealthz(nil, [db.MaxDB]*db.DB{}, qParams, true, want)
	if !empty || err != want {
		t.Fatalf("transformer failure was hidden: empty=%t, err=%v", empty, err)
	}
	app.pathInfo = NewPathInfo("/openconfig-platform:components/component[name=PSU0]/state")
	want = tlerr.NotFound("Resource not found")
	_, err = app.composePlatformHealthz(nil, [db.MaxDB]*db.DB{}, qParams, true, want)
	if !tlerr.IsNotFound(err) {
		t.Fatalf("unrelated component state error was hidden: %v", err)
	}
}

func TestPlatformCommonAppHealthzComposition(t *testing.T) {
	stateDB := getStateDB()
	if stateDB == nil {
		t.Fatal("STATE_DB unavailable")
	}
	t.Cleanup(func() { stateDB.DeleteDB() })
	const component = "DLDD_merge_healthz_test"
	const software = "DLDD_merge_software_test"
	for _, row := range []struct {
		table db.TableSpec
		key   db.Key
		value db.Value
	}{
		{platformHealthzTableSpec, db.Key{Comp: []string{component}}, validPlatformHealthzEntry()},
		{db.TableSpec{Name: "SW_COMP_INFO"}, db.Key{Comp: []string{software}}, db.Value{Field: map[string]string{
			"name": software, "type": "SOFTWARE_MODULE", "software-version": "dldd-merge-test", "oper-status": "active",
		}}},
	} {
		row := row
		if err := stateDB.SetEntry(&row.table, row.key, row.value); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { stateDB.DeleteEntry(&row.table, row.key) })
	}
	for _, uri := range []string{
		"/openconfig-platform:components/component[name=" + component + "]",
		"/openconfig-platform:components/component[name=" + component + "]/openconfig-platform-healthz:healthz/state",
		"/openconfig-platform:components",
	} {
		for _, format := range []TranslibFmtType{TRANSLIB_FMT_IETF_JSON, TRANSLIB_FMT_YGOT} {
			response, err := Get(GetRequest{Path: uri, FmtType: format})
			if err != nil {
				t.Fatalf("GET %s: %v", uri, err)
			}
			payload := response.Payload
			if format == TRANSLIB_FMT_YGOT {
				payload, err = dumpIetfJson(response.ValueTree)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(string(payload), "UNHEALTHY") {
				t.Fatalf("GET %s lost projected Healthz state: %s", uri, payload)
			}
			if uri == "/openconfig-platform:components" && !strings.Contains(string(payload), "dldd-merge-test") {
				t.Fatalf("parent GET lost transformer software data: %s", payload)
			}
		}
	}
	response, err := Get(GetRequest{Path: "/openconfig-platform:components", QueryParams: QueryParameters{Content: "config"}})
	if err != nil || strings.Contains(string(response.Payload), "UNHEALTHY") {
		t.Fatalf("config query included operational Healthz state: %s, %v", response.Payload, err)
	}

	var dbs [db.MaxDB]*db.DB
	dbs[db.StateDB] = stateDB
	configDB, err := db.NewDB(db.Options{DBNo: db.ConfigDB, TableNameSeparator: "|", KeySeparator: "|"})
	if err != nil {
		t.Fatal(err)
	}
	defer configDB.DeleteDB()
	dbs[db.ConfigDB] = configDB
	app := &CommonApp{}
	for _, parent := range []string{
		"/openconfig-platform:components/component[name=" + component + "]",
		"/openconfig-platform:components/component[name=*]",
	} {
		mapping, err := app.translateSubscribe(translateSubRequest{path: parent, mode: OnChange, recurse: true, dbs: dbs})
		if err != nil || len(mapping.ntfAppInfoTrgt) == 0 {
			t.Fatalf("DLDD parent ON_CHANGE has no target mapping: %+v, %v", mapping, err)
		}
		for _, info := range append(mapping.ntfAppInfoTrgt, mapping.ntfAppInfoTrgtChlds...) {
			if err := (&subscribeContext{}).validateAppInfo(info, OnChange); err != nil {
				t.Fatalf("invalid ON_CHANGE mapping for %s: %v", parent, err)
			}
		}
	}
	mapping, err := app.translateSubscribe(translateSubRequest{
		path: "/openconfig-platform:components/component[name=" + software + "]", recurse: true, dbs: dbs,
	})
	if err != nil {
		t.Fatal(err)
	}
	tables := make(map[string]bool)
	for _, info := range append(mapping.ntfAppInfoTrgt, mapping.ntfAppInfoTrgtChlds...) {
		if info.table != nil {
			tables[info.table.Name] = true
		}
	}
	for _, table := range []string{"SW_COMP_INFO", "FAULT_INFO", "COMPONENT_HEALTH_INFO"} {
		if !tables[table] {
			t.Errorf("parent subscription lost %s: %v", table, tables)
		}
	}
}

func TestPlatformWildcardComponentInitialSync(t *testing.T) {
	stateDB := getStateDB()
	if stateDB == nil {
		t.Fatal("STATE_DB unavailable")
	}
	defer stateDB.DeleteDB()

	const faultOnly = "DLDD_sync_fault_only"
	const healthOnly = "DLDD_sync_health_only"
	const mixed = "DLDD_sync_inventory_and_health"
	const symptom = "SYMPTOM_OVER_THRESHOLD"
	fault := validPlatformFaultEntry()
	fault.Field["component_name"] = faultOnly
	for _, row := range []struct {
		table db.TableSpec
		key   db.Key
		value db.Value
	}{
		{platformFaultTableSpec, db.Key{Comp: []string{faultOnly, symptom}}, fault},
		{platformHealthzTableSpec, db.Key{Comp: []string{healthOnly}}, validPlatformHealthzEntry()},
		{platformHealthzTableSpec, db.Key{Comp: []string{mixed}}, validPlatformHealthzEntry()},
		{db.TableSpec{Name: "SW_COMP_INFO"}, db.Key{Comp: []string{mixed}}, db.Value{Field: map[string]string{
			"name": mixed, "type": "SOFTWARE_MODULE", "software-version": "dldd-sync-test", "oper-status": "active",
		}}},
	} {
		if err := stateDB.SetEntry(&row.table, row.key, row.value); err != nil {
			t.Fatal(err)
		}
		defer stateDB.DeleteEntry(&row.table, row.key)
	}

	const uri = "/openconfig-platform:components/component[name=*]"
	for _, mode := range []NotificationType{OnChange, TargetDefined} {
		t.Run(mode.String(), func(t *testing.T) {
			session := NewSubscribeSession()
			defer session.Close()
			if mode == TargetDefined {
				prefs, err := IsSubscribeSupported(IsSubscribeRequest{
					Paths: []IsSubscribePath{{Path: uri, Mode: mode}}, Session: session,
				})
				if err != nil || len(prefs) == 0 || prefs[0].Err != nil ||
					!prefs[0].IsOnChangeSupported || prefs[0].PreferredType != OnChange {
					t.Fatalf("wildcard subscription preferences: %+v, %v", prefs, err)
				}
			}
			q := queue.NewPriorityQueue(1, false)
			stop := make(chan struct{})
			defer close(stop)
			if err := Subscribe(SubscribeRequest{Paths: []string{uri}, Q: q, Stop: stop, Session: session}); err != nil {
				t.Fatal(err)
			}
			items, err := q.Get(q.Len())
			if err != nil {
				t.Fatal(err)
			}
			var faultSeen, healthSeen, inventorySeen, mixedHealthSeen bool
			for _, item := range items {
				resp := item.(*SubscribeResponse)
				if resp.SyncComplete {
					if !faultSeen || !healthSeen || !inventorySeen || !mixedHealthSeen {
						t.Fatalf("initial sync missing data: fault=%t health=%t inventory=%t mixed health=%t",
							faultSeen, healthSeen, inventorySeen, mixedHealthSeen)
					}
					return
				}
				if resp.Update == nil {
					continue
				}
				payload, err := dumpIetfJson(resp.Update)
				if err != nil {
					t.Fatal(err)
				}
				update := resp.Path + string(payload)
				faultSeen = faultSeen || (strings.Contains(update, faultOnly) && strings.Contains(update, symptom))
				healthSeen = healthSeen || (strings.Contains(update, healthOnly) && strings.Contains(update, "UNHEALTHY"))
				inventorySeen = inventorySeen || (strings.Contains(update, mixed) && strings.Contains(update, "dldd-sync-test"))
				mixedHealthSeen = mixedHealthSeen || (strings.Contains(update, mixed) && strings.Contains(update, "UNHEALTHY"))
			}
			t.Fatal("missing sync_complete")
		})
	}
}
