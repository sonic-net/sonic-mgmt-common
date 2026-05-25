//////////////////////////////////////////////////////////////////////////
//
// Copyright 2019 Dell, Inc.
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

// pfm_app_transceiver.go holds the per-component OpenConfig Platform
// Transceiver getters and their STATE_DB-backed Db structs. The code was
// moved out of pfm_app.go to keep that file focused on the PlatformApp
// lifecycle (init / translate* / process* / doGetPlatformInfo) and the
// non-transceiver Component-state path. doGetPlatformInfo continues to be
// the single entry point that dispatches GETs to these getters.

package translib

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
)

type CompTransceiverStateDb struct {
	Connector    string
	Manufacturer string
	VendorOui    string
	VendorRev    string
	Serial       string
	VendorDate   string
}

func (app *PlatformApp) getCompTransceiverStateDbObj(ifName string) CompTransceiverStateDb {
	log.Infof("parseCompTransceiverStateDb Enter ifName=%s", ifName)

	var compTransceiverStateDbObj CompTransceiverStateDb

	transceiverInfoTable, ok := app.getTransceiverInfoEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverStateDbObj: TRANSCEIVER_INFO entry missing for ifName=%s", ifName)
		return compTransceiverStateDbObj
	}

	compTransceiverStateDbObj.Connector = transceiverInfoTable.Get("connector")
	compTransceiverStateDbObj.Manufacturer = transceiverInfoTable.Get("manufacturer")
	compTransceiverStateDbObj.VendorOui = transceiverInfoTable.Get("vendor_oui")
	compTransceiverStateDbObj.VendorRev = transceiverInfoTable.Get("vendor_rev")
	compTransceiverStateDbObj.Serial = transceiverInfoTable.Get("serial")
	compTransceiverStateDbObj.VendorDate = transceiverInfoTable.Get("vendor_date")

	return compTransceiverStateDbObj
}

func (app *PlatformApp) getCompTransceiverStateFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_State, all bool, compName string) error {
	log.Infof("getCompTransceiverStateFromDb Enter compName=%s", compName)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverStateDb := app.getCompTransceiverStateDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/connector-type" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("connector") {
			if strings.HasPrefix(compTransceiverStateDb.Connector, "AOC") {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_AOC_CONNECTOR
			} else if strings.HasPrefix(compTransceiverStateDb.Connector, "DAC") {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_DAC_CONNECTOR
			} else if strings.HasPrefix(compTransceiverStateDb.Connector, "LC") {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_LC_CONNECTOR
			} else if strings.HasPrefix(compTransceiverStateDb.Connector, "MPO") {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_MPO_CONNECTOR
			} else if strings.HasPrefix(compTransceiverStateDb.Connector, "SC") {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_SC_CONNECTOR
			} else {
				oc_val.ConnectorType = ocbinds.OpenconfigTransportTypes_FIBER_CONNECTOR_TYPE_UNSET
			}
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/vendor" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("manufacturer") {
			oc_val.Vendor = &compTransceiverStateDb.Manufacturer
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/vendor-part" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("vendor_oui") {
			// Per HLD (sonic-net/SONiC PR #1858, "Mapping between Openconfig
			// YANG and Redis DB" Table 2): vendor-part is sourced from
			// STATE_DB TRANSCEIVER_INFO.vendor_oui.
			oc_val.VendorPart = &compTransceiverStateDb.VendorOui
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/vendor-rev" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("vendor_rev") {
			oc_val.VendorRev = &compTransceiverStateDb.VendorRev
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/serial-no" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("serial") {
			oc_val.SerialNo = &compTransceiverStateDb.Serial
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/date-code" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("vendor_date") {
			rex := regexp.MustCompile("[0-9]+")
			subMatchString := rex.FindAllString(compTransceiverStateDb.VendorDate, -1)
			if len(subMatchString) >= 3 {
				if len(subMatchString[0]) == 4 && len(subMatchString[1]) == 2 && len(subMatchString[2]) == 2 {
					vendorDate := fmt.Sprintf("%s-%s-%sT00:00:00.000Z", subMatchString[0], subMatchString[1], subMatchString[2])
					formatMatch, _ := regexp.MatchString("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])[Tt]00:00:00\\.000Z", vendorDate)
					if formatMatch {
						oc_val.DateCode = &vendorDate
					}
				}
			}
		}
	}

	return nil
}

type CompTransceiverStateSupplyVoltageDb struct {
	voltage float64
}

func (app *PlatformApp) getCompTransceiverStateSupplyVoltageDbObj(ifName string) CompTransceiverStateSupplyVoltageDb {
	log.Infof("parseCompTransceiverStateSupplyVoltageDb Enter ifName=%s", ifName)

	var compTransceiverStateSupplyVoltageDbObj CompTransceiverStateSupplyVoltageDb

	transceiverDomSensorTable, ok := app.getTransceiverDomSensorEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverStateSupplyVoltageDbObj: TRANSCEIVER_DOM_SENSOR entry missing for ifName=%s", ifName)
		compTransceiverStateSupplyVoltageDbObj.voltage = math.NaN()
		return compTransceiverStateSupplyVoltageDbObj
	}

	if transceiverDomSensorTable.Get("voltage") != "N/A" {
		compTransceiverStateSupplyVoltageDbObj.voltage, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("voltage"), 64)
	} else {
		compTransceiverStateSupplyVoltageDbObj.voltage = math.NaN()
	}

	return compTransceiverStateSupplyVoltageDbObj
}

func (app *PlatformApp) getCompTransceiverStateSupplyVoltageFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_State_SupplyVoltage, all bool, compName string) error {
	log.Infof("getCompTransceiverStateSupplyVoltageFromDb Enter compName=%s", compName)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverStateSupplyVoltageDb := app.getCompTransceiverStateSupplyVoltageDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/supply-voltage/instant" {
		transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
		if transceiverDomSensorTable.Has("voltage") {
			voltage := math.Floor(compTransceiverStateSupplyVoltageDb.voltage*fractionDigits2) / fractionDigits2
			oc_val.Instant = &voltage
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateLaserTemperatureDb struct {
	temperature float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserTemperatureDbObj(ifName string) CompTransceiverPhysicalChannelStateLaserTemperatureDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateLaserTemperatureDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateLaserTemperatureDbObj CompTransceiverPhysicalChannelStateLaserTemperatureDb

	transceiverDomSensorTable, ok := app.getTransceiverDomSensorEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverPhysicalChannelStateLaserTemperatureDbObj: TRANSCEIVER_DOM_SENSOR entry missing for ifName=%s", ifName)
		compTransceiverPhysicalChannelStateLaserTemperatureDbObj.temperature = math.NaN()
		return compTransceiverPhysicalChannelStateLaserTemperatureDbObj
	}

	if transceiverDomSensorTable.Get("temperature") != "N/A" {
		compTransceiverPhysicalChannelStateLaserTemperatureDbObj.temperature, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("temperature"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserTemperatureDbObj.temperature = math.NaN()
	}

	return compTransceiverPhysicalChannelStateLaserTemperatureDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_LaserTemperature, all bool, compName string) error {
	log.Infof("getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb Enter compName=%s", compName)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateLaserTemperatureDb := app.getCompTransceiverPhysicalChannelStateLaserTemperatureDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-temperature/instant" {
		transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
		if transceiverDomSensorTable.Has("temperature") {
			temperature := math.Floor(compTransceiverPhysicalChannelStateLaserTemperatureDb.temperature*fractionDigits1) / fractionDigits1
			oc_val.Instant = &temperature
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateOutputPowerDb struct {
	TxPower [8]float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateOutputPowerDbObj(ifName string) CompTransceiverPhysicalChannelStateOutputPowerDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateOutputPowerDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateOutputPowerDbObj CompTransceiverPhysicalChannelStateOutputPowerDb

	transceiverDomSensorTable, ok := app.getTransceiverDomSensorEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverPhysicalChannelStateOutputPowerDbObj: TRANSCEIVER_DOM_SENSOR entry missing for ifName=%s", ifName)
		return compTransceiverPhysicalChannelStateOutputPowerDbObj
	}

	for i := 0; i < 8; i++ {
		compTransceiverPhysicalChannelStateOutputPowerDbObj.TxPower[i], _ = strconv.ParseFloat(transceiverDomSensorTable.Get(fmt.Sprintf("tx%dpower", i+1)), 64)
	}

	return compTransceiverPhysicalChannelStateOutputPowerDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateOutputPowerFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_OutputPower, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateOutputPowerFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateOutputPowerDb := app.getCompTransceiverPhysicalChannelStateOutputPowerDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/output-power/instant" {
		if int(laneIndex) < 8 {
			fieldName := fmt.Sprintf("tx%dpower", laneIndex+1)
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has(fieldName) {
				txpower := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.TxPower[laneIndex]*fractionDigits2) / fractionDigits2
				oc_val.Instant = &txpower
			}
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateInputPowerDb struct {
	RxPower [8]float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateInputPowerDbObj(ifName string) CompTransceiverPhysicalChannelStateInputPowerDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateInputPowerDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateInputPowerDbObj CompTransceiverPhysicalChannelStateInputPowerDb

	transceiverDomSensorTable, ok := app.getTransceiverDomSensorEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverPhysicalChannelStateInputPowerDbObj: TRANSCEIVER_DOM_SENSOR entry missing for ifName=%s", ifName)
		return compTransceiverPhysicalChannelStateInputPowerDbObj
	}

	for i := 0; i < 8; i++ {
		compTransceiverPhysicalChannelStateInputPowerDbObj.RxPower[i], _ = strconv.ParseFloat(transceiverDomSensorTable.Get(fmt.Sprintf("rx%dpower", i+1)), 64)
	}

	return compTransceiverPhysicalChannelStateInputPowerDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateInputPowerFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_InputPower, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateInputPowerFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateInputPowerDb := app.getCompTransceiverPhysicalChannelStateInputPowerDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/input-power/instant" {
		if int(laneIndex) < 8 {
			fieldName := fmt.Sprintf("rx%dpower", laneIndex+1)
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has(fieldName) {
				rxpower := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.RxPower[laneIndex]*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rxpower
			}
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateLaserBiasCurrentDb struct {
	TxBias [8]float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserBiasCurrentDbObj(ifName string) CompTransceiverPhysicalChannelStateLaserBiasCurrentDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateLaserBiasCurrentDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj CompTransceiverPhysicalChannelStateLaserBiasCurrentDb

	transceiverDomSensorTable, ok := app.getTransceiverDomSensorEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverPhysicalChannelStateLaserBiasCurrentDbObj: TRANSCEIVER_DOM_SENSOR entry missing for ifName=%s", ifName)
		for i := 0; i < 8; i++ {
			compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.TxBias[i] = math.NaN()
		}
		return compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj
	}

	for i := 0; i < 8; i++ {
		field := fmt.Sprintf("tx%dbias", i+1)
		raw := transceiverDomSensorTable.Get(field)
		if raw != "N/A" {
			compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.TxBias[i], _ = strconv.ParseFloat(raw, 64)
		} else {
			compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.TxBias[i] = math.NaN()
		}
	}

	return compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_LaserBiasCurrent, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateLaserBiasCurrentDb := app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-bias-current/instant" {
		if int(laneIndex) < 8 {
			fieldName := fmt.Sprintf("tx%dbias", laneIndex+1)
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has(fieldName) {
				txbias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.TxBias[laneIndex]*fractionDigits2) / fractionDigits2
				oc_val.Instant = &txbias
			}
		}
	}

	return nil
}

type CompTransceiverThresholdStateDb struct {
	TempHighAlarm      float64
	TempLowAlarm       float64
	VccHighAlarm       float64
	VccLowAlarm        float64
	TempHighWarning    float64
	TempLowWarning     float64
	VccHighWarning     float64
	VccLowWarning      float64
	TxPowerHighAlarm   float64
	TxPowerLowAlarm    float64
	RxPowerHighAlarm   float64
	RxPowerLowAlarm    float64
	TxBiasHighAlarm    float64
	TxBiasLowAlarm     float64
	TxPowerHighWarning float64
	TxPowerLowWarning  float64
	RxPowerHighWarning float64
	RxPowerLowWarning  float64
	TxBiasHighWarning  float64
	TxBiasLowWarning   float64
}

func (app *PlatformApp) getCompTransceiverThresholdStateDbObj(ifName string) CompTransceiverThresholdStateDb {
	log.Infof("parseCompTransceiverThresholdStateDb Enter ifName=%s", ifName)

	var compTransceiverThresholdStateDbObj CompTransceiverThresholdStateDb

	transceiverDomThresholdTable, ok := app.getTransceiverDomThresholdEntry(ifName)
	if !ok {
		log.Warningf("getCompTransceiverThresholdStateDbObj: TRANSCEIVER_DOM_THRESHOLD entry missing for ifName=%s", ifName)
		return compTransceiverThresholdStateDbObj
	}

	compTransceiverThresholdStateDbObj.TempHighAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("temphighalarm"), 64)
	compTransceiverThresholdStateDbObj.TempLowAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("templowalarm"), 64)
	compTransceiverThresholdStateDbObj.VccHighAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcchighalarm"), 64)
	compTransceiverThresholdStateDbObj.VccLowAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcclowalarm"), 64)
	compTransceiverThresholdStateDbObj.TempHighWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("temphighwarning"), 64)
	compTransceiverThresholdStateDbObj.TempLowWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("templowwarning"), 64)
	compTransceiverThresholdStateDbObj.VccHighWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcchighwarning"), 64)
	compTransceiverThresholdStateDbObj.VccLowWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcclowwarning"), 64)
	compTransceiverThresholdStateDbObj.TxPowerHighAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerhighalarm"), 64)
	compTransceiverThresholdStateDbObj.TxPowerLowAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerlowalarm"), 64)
	compTransceiverThresholdStateDbObj.RxPowerHighAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerhighalarm"), 64)
	compTransceiverThresholdStateDbObj.RxPowerLowAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerlowalarm"), 64)
	compTransceiverThresholdStateDbObj.TxBiasHighAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiashighalarm"), 64)
	compTransceiverThresholdStateDbObj.TxBiasLowAlarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiaslowalarm"), 64)
	compTransceiverThresholdStateDbObj.TxPowerHighWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerhighwarning"), 64)
	compTransceiverThresholdStateDbObj.TxPowerLowWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerlowwarning"), 64)
	compTransceiverThresholdStateDbObj.RxPowerHighWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerhighwarning"), 64)
	compTransceiverThresholdStateDbObj.RxPowerLowWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerlowwarning"), 64)
	compTransceiverThresholdStateDbObj.TxBiasHighWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiashighwarning"), 64)
	compTransceiverThresholdStateDbObj.TxBiasLowWarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiaslowwarning"), 64)

	return compTransceiverThresholdStateDbObj
}

func (app *PlatformApp) getCompTransceiverThresholdStateFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_Thresholds_Threshold_State, all bool, compName string, severityName string) error {
	log.Infof("getCompTransceiverThresholdStateFromDb Enter compName=%s severityName=%s", compName, severityName)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverThresholdStateDb := app.getCompTransceiverThresholdStateDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if severityName == "CRITICAL" {
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("temphighalarm") {
				temphighalarm := math.Floor(compTransceiverThresholdStateDb.TempHighAlarm*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureUpper = &temphighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("templowalarm") {
				templowalarm := math.Floor(compTransceiverThresholdStateDb.TempLowAlarm*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureLower = &templowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcchighalarm") {
				vcchighalarm := math.Floor(compTransceiverThresholdStateDb.VccHighAlarm*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageUpper = &vcchighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcclowalarm") {
				vcclowalarm := math.Floor(compTransceiverThresholdStateDb.VccLowAlarm*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageLower = &vcclowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerhighalarm") {
				txpowerhighalarm := math.Floor(compTransceiverThresholdStateDb.TxPowerHighAlarm*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerUpper = &txpowerhighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerlowalarm") {
				txpowerlowalarm := math.Floor(compTransceiverThresholdStateDb.TxPowerLowAlarm*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerLower = &txpowerlowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerhighalarm") {
				rxpowerhighalarm := math.Floor(compTransceiverThresholdStateDb.RxPowerHighAlarm*fractionDigits2) / fractionDigits2
				oc_val.InputPowerUpper = &rxpowerhighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerlowalarm") {
				rxpowerlowalarm := math.Floor(compTransceiverThresholdStateDb.RxPowerLowAlarm*fractionDigits2) / fractionDigits2
				oc_val.InputPowerLower = &rxpowerlowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiashighalarm") {
				txbiashighalarm := math.Floor(compTransceiverThresholdStateDb.TxBiasHighAlarm*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentUpper = &txbiashighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiaslowalarm") {
				txbiaslowalarm := math.Floor(compTransceiverThresholdStateDb.TxBiasLowAlarm*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentLower = &txbiaslowalarm
			}
		}
	}

	if severityName == "WARNING" {
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("temphighwarning") {
				temphighwarning := math.Floor(compTransceiverThresholdStateDb.TempHighWarning*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureUpper = &temphighwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("templowwarning") {
				templowwarning := math.Floor(compTransceiverThresholdStateDb.TempLowWarning*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureLower = &templowwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcchighwarning") {
				vcchighwarning := math.Floor(compTransceiverThresholdStateDb.VccHighWarning*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageUpper = &vcchighwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcclowwarning") {
				vcclowwarning := math.Floor(compTransceiverThresholdStateDb.VccLowWarning*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageLower = &vcclowwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerhighwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.TxPowerHighWarning*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerUpper = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerlowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.TxPowerLowWarning*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerLower = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerhighwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.RxPowerHighWarning*fractionDigits2) / fractionDigits2
				oc_val.InputPowerUpper = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerlowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.RxPowerLowWarning*fractionDigits2) / fractionDigits2
				oc_val.InputPowerLower = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiashighwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.TxBiasHighWarning*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentUpper = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiaslowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.TxBiasLowWarning*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentLower = &v
			}
		}
	}

	return nil
}
