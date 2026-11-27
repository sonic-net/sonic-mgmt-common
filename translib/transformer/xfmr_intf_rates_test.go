//////////////////////////////////////////////////////////////////////////
//
// Copyright 2024 Dell, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//////////////////////////////////////////////////////////////////////////

package transformer

import (
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func TestDbToYang_intf_get_rates_xfmr_ethernetOnly(t *testing.T) {
	countersDb, err := db.NewDB(db.Options{DBNo: db.CountersDB, TableNameSeparator: "|", KeySeparator: "|"})
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer countersDb.DeleteDB()

	configDb, err := db.NewDB(db.Options{DBNo: db.ConfigDB, TableNameSeparator: "|", KeySeparator: "|"})
	if err != nil {
		t.Fatalf("NewDB config: %v", err)
	}
	defer configDb.DeleteDB()

	dbs := [db.MaxDB]*db.DB{}
	dbs[db.CountersDB] = countersDb
	dbs[db.ConfigDB] = configDb

	t.Run("non-Ethernet returns empty rates state", func(t *testing.T) {
		device := &ocbinds.Device{}
		ygot.BuildEmptyTree(device)
		var root ygot.GoStruct = device
		inParams := XfmrParams{
			uri:    "/openconfig-interfaces:interfaces/interface[name=Vlan100]/rates/state",
			ygRoot: &root,
			dbs:    dbs,
		}
		if err := DbToYang_intf_get_rates_xfmr(inParams); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		intf, ok := device.Interfaces.Interface["Vlan100"]
		if !ok || intf.Rates == nil || intf.Rates.State == nil {
			return
		}
		if intf.Rates.State.InBitsRate != nil {
			t.Fatalf("expected no in-bits-rate for Vlan100")
		}
	})

	t.Run("invalid interface name", func(t *testing.T) {
		device := &ocbinds.Device{}
		ygot.BuildEmptyTree(device)
		var root ygot.GoStruct = device
		inParams := XfmrParams{
			uri:    "/openconfig-interfaces:interfaces/interface[name=Invalid99]/rates/state",
			ygRoot: &root,
			dbs:    dbs,
		}
		if err := DbToYang_intf_get_rates_xfmr(inParams); err == nil {
			t.Fatalf("expected error for invalid interface type")
		}
	})
}

func TestDbToYangPath_intf_get_rates_path_xfmr(t *testing.T) {
	pathParams := XfmrDbToYgPathParams{
		tblName:    "COUNTERS_PORT_NAME_MAP",
		tblKeyComp: []string{"Ethernet0"},
		ygPathKeys: map[string]string{},
	}
	if err := DbToYangPath_intf_get_rates_path_xfmr(pathParams); err != nil {
		t.Fatalf("path xfmr: %v", err)
	}
	if pathParams.ygPathKeys["/openconfig-interfaces:interfaces/interface/name"] != "Ethernet0" {
		t.Fatalf("expected Ethernet0 in path keys, got %v", pathParams.ygPathKeys)
	}
}

func TestIsIntfRatesStateURI(t *testing.T) {
	cases := []struct {
		uri    string
		expect bool
	}{
		{
			"/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-rates:rates/state",
			true,
		},
		{
			"/openconfig-interfaces:interfaces/interface[name=Ethernet0]/openconfig-if-rates:rates/state/in-bits-rate",
			true,
		},
		{
			"/openconfig-interfaces:interfaces/interface[name=Ethernet0]/rates/state",
			true,
		},
		{
			"/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state/counters",
			false,
		},
		{
			"/openconfig-interfaces:interfaces/interface[name=Ethernet0]/state",
			false,
		},
	}
	for _, tc := range cases {
		if got := isIntfRatesStateURI(tc.uri); got != tc.expect {
			t.Fatalf("isIntfRatesStateURI(%q) = %v, want %v", tc.uri, got, tc.expect)
		}
	}
}

func TestGetPortFlexPollLoadIntervalSec(t *testing.T) {
	configDb, err := db.NewDB(db.Options{DBNo: db.ConfigDB, TableNameSeparator: "|", KeySeparator: "|"})
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer configDb.DeleteDB()

	sec := getPortFlexPollLoadIntervalSec(configDb)
	if sec < 1 || sec > 600 {
		t.Fatalf("load-interval out of range: %d", sec)
	}
}
