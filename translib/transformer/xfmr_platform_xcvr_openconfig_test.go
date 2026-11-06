////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 NTT Corporation.                                           //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package transformer_test

import (
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

const xcvrTestPrefix = "/openconfig-platform:components/component[name=Ethernet0]/openconfig-platform-transceiver:transceiver"

/* STATE_DB contents modeled on an xcvrd-populated QSFP28 module. String
 * fields carry the trailing padding xcvrd copies from the EEPROM. */
var xcvrTestStateDb = map[string]interface{}{
	"TRANSCEIVER_STATUS": map[string]interface{}{
		"Ethernet0": map[string]interface{}{"status": "1"},
	},
	"TRANSCEIVER_INFO": map[string]interface{}{
		"Ethernet0": map[string]interface{}{
			"type":         "QSFP28 or later",
			"manufacturer": "FINISAR CORP    ",
			"model":        "FTLC9558REPM    ",
			"serial":       "XAPB6M8         ",
			"vendor_oui":   "00-90-65",
			"vendor_rev":   "A1",
			"vendor_date":  "2023-12-26   ",
			"connector":    "MPO 1x12",
		},
	},
	"TRANSCEIVER_DOM_SENSOR": map[string]interface{}{
		"Ethernet0": map[string]interface{}{
			"temperature": "33.73",
			"voltage":     "3.321",
			"tx1power":    "-1.073", "tx2power": "-1.1", "tx3power": "-1.2", "tx4power": "-1.3",
			"rx1power": "-0.074", "rx2power": "-0.1", "rx3power": "-0.2", "rx4power": "-0.3",
			"tx1bias": "7.292", "tx2bias": "7.1", "tx3bias": "7.2", "tx4bias": "7.3",
			"tx1disable": "False", "tx2disable": "False", "tx3disable": "False", "tx4disable": "False",
		},
	},
	"TRANSCEIVER_DOM_THRESHOLD": map[string]interface{}{
		"Ethernet0": map[string]interface{}{
			"temphighalarm": "75.0", "templowalarm": "-5.0", "temphighwarning": "70.0", "templowwarning": "0.0",
			"vcchighalarm": "3.63", "vcclowalarm": "2.97", "vcchighwarning": "3.465", "vcclowwarning": "3.135",
			"rxpowerhighalarm": "3.4", "rxpowerlowalarm": "-14.001", "rxpowerhighwarning": "2.4", "rxpowerlowwarning": "-11.002",
			"txpowerhighalarm": "5.0", "txpowerlowalarm": "-10.0", "txpowerhighwarning": "3.0", "txpowerlowwarning": "-8.0",
			"txbiashighalarm": "13.0", "txbiaslowalarm": "3.0", "txbiashighwarning": "11.0", "txbiaslowwarning": "5.0",
		},
	},
}

func Test_openconfig_platform_transceiver(t *testing.T) {
	loadDB(db.StateDB, xcvrTestStateDb)
	defer unloadDB(db.StateDB, xcvrTestStateDb)
	time.Sleep(1 * time.Second)

	t.Run("transceiver state", processGetRequest(xcvrTestPrefix+"/state", nil,
		`{"state":{
			"form-factor":"openconfig-transport-types:QSFP28",
			"connector-type":"openconfig-transport-types:MPO_CONNECTOR",
			"vendor":"FINISAR CORP",
			"vendor-part":"00-90-65",
			"vendor-rev":"A1",
			"serial-no":"XAPB6M8",
			"date-code":"2023-12-26T00:00:00.000Z",
			"supply-voltage":{"instant":"3.32"}}}`, false))

	t.Run("transceiver state vendor-part", processGetRequest(xcvrTestPrefix+"/state/vendor-part", nil,
		`{"vendor-part":"00-90-65"}`, false))

	t.Run("transceiver state date-code", processGetRequest(xcvrTestPrefix+"/state/date-code", nil,
		`{"date-code":"2023-12-26T00:00:00.000Z"}`, false))

	t.Run("transceiver state connector-type", processGetRequest(xcvrTestPrefix+"/state/connector-type", nil,
		`{"connector-type":"openconfig-transport-types:MPO_CONNECTOR"}`, false))

	t.Run("transceiver supply-voltage", processGetRequest(xcvrTestPrefix+"/state/supply-voltage", nil,
		`{"supply-voltage":{"instant":"3.32"}}`, false))

	t.Run("component mfg-date", processGetRequest("/openconfig-platform:components/component[name=Ethernet0]/state/mfg-date", nil,
		`{"openconfig-platform:mfg-date":"2023-12-26"}`, false))

	t.Run("channel laser-temperature", processGetRequest(xcvrTestPrefix+"/physical-channels/channel[index=2]/state/laser-temperature", nil,
		`{"laser-temperature":{"instant":"33.7"}}`, false))

	t.Run("threshold critical", processGetRequest(xcvrTestPrefix+"/thresholds/threshold[severity=CRITICAL]/state", nil,
		`{"state":{
			"severity":"openconfig-alarm-types:CRITICAL",
			"laser-temperature-upper":"75","laser-temperature-lower":"-5",
			"supply-voltage-upper":"3.63","supply-voltage-lower":"2.97",
			"input-power-upper":"3.4","input-power-lower":"-14",
			"output-power-upper":"5","output-power-lower":"-10",
			"laser-bias-current-upper":"13","laser-bias-current-lower":"3"}}`, false))

	t.Run("threshold warning", processGetRequest(xcvrTestPrefix+"/thresholds/threshold[severity=WARNING]/state", nil,
		`{"state":{
			"severity":"openconfig-alarm-types:WARNING",
			"laser-temperature-upper":"70","laser-temperature-lower":"0",
			"supply-voltage-upper":"3.46","supply-voltage-lower":"3.13",
			"input-power-upper":"2.4","input-power-lower":"-11",
			"output-power-upper":"3","output-power-lower":"-8",
			"laser-bias-current-upper":"11","laser-bias-current-lower":"5"}}`, false))
}

/* A module whose vendor_date is not a calendar date (seen on real hardware
 * as "2020-23-12 08") must not leak into mfg-date / date-code, which are
 * typed as oc-yang:date / oc-yang:date-and-time. */
func Test_openconfig_platform_transceiver_bad_vendor_date(t *testing.T) {
	stateDb := map[string]interface{}{
		"TRANSCEIVER_STATUS": map[string]interface{}{
			"Ethernet0": map[string]interface{}{"status": "1"},
		},
		"TRANSCEIVER_INFO": map[string]interface{}{
			"Ethernet0": map[string]interface{}{
				"type":        "QSFP28 or later",
				"vendor_rev":  "A1",
				"vendor_date": "2020-23-12 08",
			},
		},
	}
	loadDB(db.StateDB, stateDb)
	defer unloadDB(db.StateDB, stateDb)
	time.Sleep(1 * time.Second)

	t.Run("transceiver state without date-code", processGetRequest(xcvrTestPrefix+"/state", nil,
		`{"state":{
			"form-factor":"openconfig-transport-types:QSFP28",
			"vendor-rev":"A1"}}`, false))
}
