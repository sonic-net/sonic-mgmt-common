////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Nokia.                                                     //
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

//go:build testapp
// +build testapp

package transformer

import (
	"reflect"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/openconfig/ygot/ygot"
)

func TestSubscribe_dns_servers_xfmr(t *testing.T) {
	const serverUri = "/openconfig-system:system/dns/servers/server[address=192.0.2.1]"
	tests := []struct {
		name        string
		inParams    XfmrSubscInParams
		wantKey     string
		wantVirtual bool
	}{
		{
			name:     "subscribe to all servers",
			inParams: XfmrSubscInParams{uri: dnsServerPath + "[address=*]", requestURI: dnsServersPath, subscProc: TRANSLATE_SUBSCRIBE},
			wantKey:  "*",
		},
		{
			name:     "subscribe to one server",
			inParams: XfmrSubscInParams{uri: serverUri, requestURI: serverUri + "/state", subscProc: TRANSLATE_SUBSCRIBE},
			wantKey:  "192.0.2.1",
		},
		{
			name:        "resource check for one server",
			inParams:    XfmrSubscInParams{uri: serverUri, subscProc: TRANSLATE_SUBSCRIBE},
			wantVirtual: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Subscribe_dns_servers_xfmr(tt.inParams)
			if err != nil {
				t.Fatalf("Subscribe_dns_servers_xfmr() error = %v", err)
			}
			if got.isVirtualTbl != tt.wantVirtual {
				t.Fatalf("isVirtualTbl = %v, want %v", got.isVirtualTbl, tt.wantVirtual)
			}
			if tt.wantVirtual {
				return
			}
			want := RedisDbSubscribeMap{db.ConfigDB: {DNS_NAMESERVER_TABLE: {tt.wantKey: {}}}}
			if !reflect.DeepEqual(got.dbDataMap, want) {
				t.Errorf("dbDataMap = %v, want %v", got.dbDataMap, want)
			}
			if got.onChange != OnchangeEnable {
				t.Errorf("onChange = %v, want OnchangeEnable", got.onChange)
			}
		})
	}
}

func TestDbToYangPath_dns_servers_path_xfmr(t *testing.T) {
	for _, sub := range []string{"", "/state", "/config/address"} {
		t.Run("server[address=*]"+sub, func(t *testing.T) {
			gPath, err := ygot.StringToPath(dnsServerPath+"[address=*]"+sub, ygot.StructuredPath)
			if err != nil {
				t.Fatal(err)
			}
			xlator, err := NewSubscribeNotfRespXlator("dns", gPath, db.ConfigDB,
				&db.TableSpec{Name: DNS_NAMESERVER_TABLE}, &db.Key{Comp: []string{"2001:db8::53"}},
				nil, [db.MaxDB]*db.DB{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := xlator.Translate()
			if err != nil {
				t.Fatalf("Translate() error = %v", err)
			}
			want := dnsServerPath + "[address=2001:db8::53]" + sub
			if s, _ := ygot.PathToString(got); s != want {
				t.Errorf("notification path = %s, want %s", s, want)
			}
		})
	}

	err := DbToYangPath_dns_servers_path_xfmr(XfmrDbToYgPathParams{
		tblName: "DNS_OPTIONS", tblKeyComp: []string{"GLOBAL"}, ygPathKeys: map[string]string{}})
	if err == nil {
		t.Error("DbToYangPath_dns_servers_path_xfmr() accepted a DNS_OPTIONS key")
	}
}
