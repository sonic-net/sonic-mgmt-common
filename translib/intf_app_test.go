////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
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
	"fmt"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

func TestIntfApp_translateSubscribe(t *testing.T) {

	for _, ifName := range []string{"", "*", "Ethernet123"} {
		t.Run(fmt.Sprintf("top[name=%s]", ifName), func(t *testing.T) {
			reqPath := "/openconfig-interfaces:interfaces"
			if len(ifName) != 0 {
				reqPath += fmt.Sprintf("/interface[name=%s]", ifName)
			} else {
				ifName = "*"
			}

			ifPath := fmt.Sprintf("/openconfig-interfaces:interfaces/interface[name=%s]", ifName)
			tv := testTranslateSubscribe(t, reqPath)
			tv.VerifyCount(1, 4)
			tv.VerifyTarget(
				ifPath, portConfigNInfo(ifName, portConfigAllFields))
			tv.VerifyChild(
				ifPath+"/state", portStateNInfo(ifName, portStateAllFields))
			tv.VerifyChild(
				ifPath+"/state/counters", portCountersNInfo(ifName))
			tv.VerifyChild(
				ifPath+"/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address[ip=*]",
				portIpAddrNInfo(ifName, portIpv4KeyPattern))
			tv.VerifyChild(
				ifPath+"/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address[ip=*]",
				portIpAddrNInfo(ifName, portIpv6KeyPattern))
		})
	}

	for _, ifName := range []string{"*", "Ethernet123"} {
		tcPrefix := fmt.Sprintf("name=%s|", ifName)
		ifPath := fmt.Sprintf("/openconfig-interfaces:interfaces/interface[name=%s]", ifName)

		t.Run(tcPrefix+"ifName", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/name")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/name", portConfigNInfo(ifName, `{}`))
		})

		t.Run(tcPrefix+"config_container", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/config")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/config", portConfigNInfo(ifName, `{"": `+portConfigFields+`}`))
		})

		t.Run(tcPrefix+"config_attr", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/config/mtu")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/config/mtu", portConfigNInfo(ifName, `{"": {"mtu": ""}}`))
		})

		t.Run(tcPrefix+"eth_all", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/openconfig-if-ethernet:ethernet")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/openconfig-if-ethernet:ethernet",
				portConfigNInfo(ifName, `{"config":`+portEthConfigFields+`, "state":`+portEthConfigFields+`}`))
		})

		t.Run(tcPrefix+"eth_config", func(t *testing.T) {
			p := ifPath + "/openconfig-if-ethernet:ethernet/config"
			tv := testTranslateSubscribe(t, p)
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(p, portConfigNInfo(ifName, `{"":`+portEthConfigFields+`}`))
		})

		t.Run(tcPrefix+"eth_speed", func(t *testing.T) {
			p := ifPath + "/openconfig-if-ethernet:ethernet/state/port-speed"
			tv := testTranslateSubscribe(t, p)
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(p, portConfigNInfo(ifName, `{"":{"speed": ""}}`))
		})

		t.Run(tcPrefix+"state_container", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/state")
			tv.VerifyCount(1, 1)
			tv.VerifyTarget(ifPath+"/state", portStateNInfo(ifName, portStateAllFields))
			tv.VerifyChild(ifPath+"/state/counters", portCountersNInfo(ifName))
		})

		t.Run(tcPrefix+"state_admin_status", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/state/admin-status")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/state/admin-status", portStateNInfo(ifName, `{"": {"admin_status": ""}}`))
		})

		t.Run(tcPrefix+"state_counters", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/state/counters")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/state/counters", portCountersNInfo(ifName))
		})

		t.Run(tcPrefix+"state_counters_attr", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/state/counters/in-octets")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ifPath+"/state/counters/in-octets", portCountersNInfo(ifName))
		})

		ipv4List := ifPath + "/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4/addresses/address"
		ipv6List := ifPath + "/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6/addresses/address"

		for _, p := range []string{"/subinterfaces", "/subinterfaces/subinterface[index=*]", "/subinterfaces/subinterface[index=0]"} {
			t.Run(tcPrefix+"subif="+NewPathInfo(p).Var("index"), func(t *testing.T) {
				tv := testTranslateSubscribe(t, ifPath+p)
				tv.VerifyCount(2, 0)
				tv.VerifyTarget(ipv4List+"[ip=*]", portIpAddrNInfo(ifName, portIpv4KeyPattern))
				tv.VerifyTarget(ipv6List+"[ip=*]", portIpAddrNInfo(ifName, portIpv6KeyPattern))
			})
		}

		t.Run(tcPrefix+"invalid_subif", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ifPath+"/subinterfaces/subinterface[index=2]")
			tv.VerifyCount(translErr, 0)
		})

		for _, p := range []string{"", "/addresses", "/addresses/address[ip=*]"} {
			t.Run(tcPrefix+"ipv4", func(t *testing.T) {
				tv := testTranslateSubscribe(t, ifPath+"/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv4"+p)
				tv.VerifyCount(1, 0)
				tv.VerifyTarget(ipv4List+"[ip=*]", portIpAddrNInfo(ifName, portIpv4KeyPattern))
			})
		}

		t.Run(tcPrefix+"ipv4_specific", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ipv4List+"[ip=1.2.3.4]")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ipv4List+"[ip=1.2.3.4]", portIpAddrNInfo(ifName, "1.2.3.4/*"))
		})

		t.Run(tcPrefix+"ipv4_invalid", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ipv4List+"[ip=1234]")
			tv.VerifyCount(translErr, 0)
		})

		for _, p := range []string{"", "/addresses", "/addresses/address[ip=*]"} {
			t.Run(tcPrefix+"ipv6", func(t *testing.T) {
				tv := testTranslateSubscribe(t, ifPath+"/subinterfaces/subinterface[index=0]/openconfig-if-ip:ipv6"+p)
				tv.VerifyCount(1, 0)
				tv.VerifyTarget(ipv6List+"[ip=*]", portIpAddrNInfo(ifName, portIpv6KeyPattern))
			})
		}

		t.Run(tcPrefix+"ipv6_specific", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ipv6List+"[ip=1001:80::1]")
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(ipv6List+"[ip=1001:80::1]", portIpAddrNInfo(ifName, "1001:80::1/*"))
		})

		t.Run(tcPrefix+"ipv6_invalid", func(t *testing.T) {
			tv := testTranslateSubscribe(t, ipv6List+"[ip=1234abcd]")
			tv.VerifyCount(translErr, 0)
		})
	}
}

const (
	portConfigFields    = `{"description": "description", "admin_status": "enabled", "mtu": "mtu"}`
	portEthConfigFields = `{"speed": "port-speed"}`
	portStateAllFields  = `{"": {"index": "ifindex", "admin_status": "enabled,admin-status", "oper_status": "oper-status", "description": "description", "mtu": "mtu"}}`
	portConfigAllFields = `{"config": ` + portConfigFields +
		`, "openconfig-if-ethernet:ethernet/config": ` + portEthConfigFields +
		`, "openconfig-if-ethernet:ethernet/state": ` + portEthConfigFields + `}`

	portIpv4KeyPattern = "*.*.*.*/*"
	portIpv6KeyPattern = "*:*/*"
)

func portConfigNInfo(ifName, fieldsJson string) *notificationAppInfo {
	return &notificationAppInfo{
		dbno:                db.ConfigDB,
		table:               &db.TableSpec{Name: "PORT"},
		key:                 db.NewKey(ifName),
		dbFldYgPathInfoList: parseFieldsJSON(fieldsJson),
		isOnChangeSupported: true,
		pType:               OnChange,
	}
}

func portStateNInfo(ifName, fieldsJson string) *notificationAppInfo {
	return &notificationAppInfo{
		dbno:                db.ApplDB,
		table:               &db.TableSpec{Name: "PORT_TABLE"},
		key:                 db.NewKey(ifName),
		dbFldYgPathInfoList: parseFieldsJSON(fieldsJson),
		isOnChangeSupported: true,
		pType:               OnChange,
	}
}

func portCountersNInfo(ifName string) *notificationAppInfo {
	fieldPattern := ifName
	if ifName == "*" {
		fieldPattern = countersMapFieldPattern
	}
	return &notificationAppInfo{
		dbno:                db.CountersDB,
		table:               &db.TableSpec{Name: "COUNTERS_PORT_NAME_MAP"},
		key:                 db.NewKey(),
		fieldScanPattern:    fieldPattern,
		isOnChangeSupported: false,
		pType:               Sample,
	}
}

func portIpAddrNInfo(ifName, ipAddr string) *notificationAppInfo {
	return &notificationAppInfo{
		dbno:                db.ConfigDB,
		table:               &db.TableSpec{Name: "INTERFACE"},
		key:                 db.NewKey(ifName, ipAddr),
		isOnChangeSupported: true,
		pType:               OnChange,
	}
}
