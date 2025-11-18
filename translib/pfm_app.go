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

package translib

import (
	"errors"
	"fmt"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type PlatformApp struct {
	path                         *PathInfo
	reqData                      []byte
	ygotRoot                     *ygot.GoStruct
	ygotTarget                   *interface{}
	eepromTs                     *db.TableSpec
	transceiverInfoTs            *db.TableSpec
	transceiverDomSensorTs       *db.TableSpec
	transceiverDomThresholdTs    *db.TableSpec
	applPortTs                   *db.TableSpec
	eepromTable                  map[string]dbEntry
	transceiverInfoTable         map[string]dbEntry
	transceiverDomSensorTable    map[string]dbEntry
	transceiverDomThresholdTable map[string]dbEntry
	applPortTable                map[string]dbEntry
}

const (
	fractionDigits1  = 10
	fractionDigits2  = 100
	fractionDigits3  = 1000
	fractionDigits18 = 1000000000000000000
)

func init() {
	log.Info("Init called for Platform module")
	err := register("/openconfig-platform:components",
		&appInfo{appType: reflect.TypeOf(PlatformApp{}),
			ygotRootType: reflect.TypeOf(ocbinds.OpenconfigPlatform_Components{}),
			isNative:     false})
	if err != nil {
		log.Fatal("Register Platform app module with App Interface failed with error=", err)
	}

	err = addModel(&ModelData{Name: "openconfig-platform",
		Org: "OpenConfig working group",
		Ver: "1.0.2"})
	if err != nil {
		log.Fatal("Adding model data to appinterface failed with error=", err)
	}
}

func (app *PlatformApp) initialize(data appData) {
	log.Info("initialize:if:path =", data.path)

	app.path = NewPathInfo(data.path)
	app.reqData = data.payload
	app.ygotRoot = data.ygotRoot
	app.ygotTarget = data.ygotTarget
	app.eepromTs = &db.TableSpec{Name: "EEPROM_INFO"}
	app.transceiverInfoTs = &db.TableSpec{Name: "TRANSCEIVER_INFO"}
	app.transceiverDomSensorTs = &db.TableSpec{Name: "TRANSCEIVER_DOM_SENSOR"}
	app.transceiverDomThresholdTs = &db.TableSpec{Name: "TRANSCEIVER_DOM_THRESHOLD"}
	app.applPortTs = &db.TableSpec{Name: "PORT_TABLE"}

}

func (app *PlatformApp) getAppRootObject() *ocbinds.OpenconfigPlatform_Components {
	deviceObj := (*app.ygotRoot).(*ocbinds.Device)
	return deviceObj.Components
}

func (app *PlatformApp) translateAction(dbs [db.MaxDB]*db.DB) error {
	err := errors.New("Not supported")
	return err
}

func (app *PlatformApp) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	return emptySubscribeResponse(req.path)
}

func (app *PlatformApp) processSubscribe(req processSubRequest) (processSubResponse, error) {
	return processSubResponse{}, tlerr.New("not implemented")
}

func (app *PlatformApp) translateCreate(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys

	err = errors.New("PlatformApp Not implemented, translateCreate")
	return keys, err
}

func (app *PlatformApp) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	err = errors.New("PlatformApp Not implemented, translateUpdate")
	return keys, err
}

func (app *PlatformApp) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys

	err = errors.New("Not implemented PlatformApp translateReplace")
	return keys, err
}

func (app *PlatformApp) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys

	err = errors.New("Not implemented PlatformApp translateDelete")
	return keys, err
}

func (app *PlatformApp) translateGet(dbs [db.MaxDB]*db.DB) error {
	var err error
	log.Info("PlatformApp: translateGet - path: ", app.path.Path)
	return err
}

func (app *PlatformApp) processCreate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	err = errors.New("Not implemented PlatformApp processCreate")
	return resp, err
}

func (app *PlatformApp) processUpdate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	err = errors.New("Not implemented PlatformApp processUpdate")
	return resp, err
}

func (app *PlatformApp) processReplace(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse
	log.Info("processReplace:intf:path =", app.path)
	err = errors.New("Not implemented, PlatformApp processReplace")
	return resp, err
}

func (app *PlatformApp) processDelete(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	err = errors.New("Not implemented PlatformApp processDelete")
	return resp, err
}

func (app *PlatformApp) processGet(dbs [db.MaxDB]*db.DB, fmtType TranslibFmtType) (GetResponse, error) {
	pathInfo := app.path
	log.Infof("Received GET for PlatformApp Template: %s ,path: %s, vars: %v",
		pathInfo.Template, pathInfo.Path, pathInfo.Vars)

	stateDb := dbs[db.StateDB]
	applDb := dbs[db.ApplDB]

	var payload []byte

	// Read eeprom info from STATE_DB
	app.eepromTable = make(map[string]dbEntry)

	tbl, derr := stateDb.GetTable(app.eepromTs)
	if derr != nil {
		log.Error("EEPROM_INFO table get failed!")
		return GetResponse{Payload: payload}, derr
	}

	keys, _ := tbl.GetKeys()
	for _, key := range keys {
		e, kerr := tbl.GetEntry(key)
		if kerr != nil {
			log.Error("EEPROM_INFO entry get failed!")
			return GetResponse{Payload: payload}, kerr
		}

		app.eepromTable[key.Get(0)] = dbEntry{entry: e}
	}

	// Read transceiver info from STATE_DB
	app.transceiverInfoTable = make(map[string]dbEntry)

	transceiverInfoTbl, derr := stateDb.GetTable(app.transceiverInfoTs)
	if derr != nil {
		log.Error("TRANSCEIVER_INFO table get failed!")
		return GetResponse{Payload: payload}, derr
	}

	transceiverInfoTblKeys, _ := transceiverInfoTbl.GetKeys()
	for _, key := range transceiverInfoTblKeys {
		e, kerr := transceiverInfoTbl.GetEntry(key)
		if kerr != nil {
			log.Error("TRANSCEIVER_INFO entry get failed!")
			return GetResponse{Payload: payload}, kerr
		}

		app.transceiverInfoTable[key.Get(0)] = dbEntry{entry: e}
	}

	// Read transceiver dom sensor from STATE_DB
	app.transceiverDomSensorTable = make(map[string]dbEntry)

	transceiverDomSensorTbl, derr := stateDb.GetTable(app.transceiverDomSensorTs)
	if derr != nil {
		log.Error("TRANSCEIVER_DOM_SENSOR table get failed!")
		return GetResponse{Payload: payload}, derr
	}

	transceiverDomSensorTblKeys, _ := transceiverDomSensorTbl.GetKeys()
	for _, key := range transceiverDomSensorTblKeys {
		e, kerr := transceiverDomSensorTbl.GetEntry(key)
		if kerr != nil {
			log.Error("TRANSCEIVER_DOM_SENSOR entry get failed!")
			return GetResponse{Payload: payload}, kerr
		}

		app.transceiverDomSensorTable[key.Get(0)] = dbEntry{entry: e}
	}

	// Read transceiver dom threshold from STATE_DB
	app.transceiverDomThresholdTable = make(map[string]dbEntry)

	transceiverDomThresholdTbl, derr := stateDb.GetTable(app.transceiverDomThresholdTs)
	if derr != nil {
		log.Error("TRANSCEIVER_DOM_THRESHOLD table get failed!")
		return GetResponse{Payload: payload}, derr
	}

	transceiverDomThresholdTblKeys, _ := transceiverDomThresholdTbl.GetKeys()
	for _, key := range transceiverDomThresholdTblKeys {
		e, kerr := transceiverDomThresholdTbl.GetEntry(key)
		if kerr != nil {
			log.Error("TRANSCEIVER_DOM_THRESHOLD entry get failed!")
			return GetResponse{Payload: payload}, kerr
		}

		app.transceiverDomThresholdTable[key.Get(0)] = dbEntry{entry: e}
	}

	// Read port from APPL_DB
	app.applPortTable = make(map[string]dbEntry)

	applPortTbl, derr := applDb.GetTable(app.applPortTs)
	if derr != nil {
		log.Error("APPL PORT table get failed!")
		return GetResponse{Payload: payload}, derr
	}

	applPortTblKeys, _ := applPortTbl.GetKeys()
	for _, key := range applPortTblKeys {
		e, kerr := applPortTbl.GetEntry(key)
		if kerr != nil {
			log.Error("PORT entry get failed!")
			return GetResponse{Payload: payload}, kerr
		}

		app.applPortTable[key.Get(0)] = dbEntry{entry: e}
	}

	targetUriPath, perr := getYangPathFromUri(app.path.Path)
	if perr != nil {
		log.Infof("getYangPathFromUri failed.")
		return GetResponse{Payload: payload}, perr
	}

	var err error

	if isSubtreeRequest(targetUriPath, "/openconfig-platform:components") {
		err = app.doGetPlatformInfo()
	} else {
		err = errors.New("Not supported component")
	}

	if err == nil {
		return generateGetResponse(pathInfo.Path, app.ygotRoot, fmtType)
	}
	return GetResponse{Payload: payload}, err
}

func (app *PlatformApp) processAction(dbs [db.MaxDB]*db.DB) (ActionResponse, error) {
	var resp ActionResponse
	err := errors.New("Not implemented")

	return resp, err
}

///////////////////////////

/*
*
Structures to read syseeprom from redis-db
*/
type EepromDb struct {
	Product_Name        string
	Part_Number         string
	Serial_Number       string
	Base_MAC_Address    string
	Manufacture_Date    string
	Device_Version      string
	Label_Revision      string
	Platform_Name       string
	ONIE_Version        string
	MAC_Addresses       int
	Manufacturer        string
	Manufacture_Country string
	Vendor_Name         string
	Diag_Version        string
	Service_Tag         string
	Vendor_Extension    string
	Magic_Number        int
	Card_Type           string
	Hardware_Version    string
	Software_Version    string
	Model_Name          string
}

func (app *PlatformApp) getEepromDbObj() EepromDb {
	log.Infof("parseEepromDb Enter")

	var eepromDbObj EepromDb

	for epItem, _ := range app.eepromTable {
		e := app.eepromTable[epItem].entry
		name := e.Get("Name")

		switch name {
		case "Device Version":
			eepromDbObj.Device_Version = e.Get("Value")
		case "Service Tag":
			eepromDbObj.Service_Tag = e.Get("Value")
		case "Vendor Extension":
			eepromDbObj.Vendor_Extension = e.Get("Value")
		case "Magic Number":
			mag, _ := strconv.ParseInt(e.Get("Value"), 10, 64)
			eepromDbObj.Magic_Number = int(mag)
		case "Card Type":
			eepromDbObj.Card_Type = e.Get("Value")
		case "Hardware Version":
			eepromDbObj.Hardware_Version = e.Get("Value")
		case "Software Version":
			eepromDbObj.Software_Version = e.Get("Value")
		case "Model Name":
			eepromDbObj.Model_Name = e.Get("Value")
		case "ONIE Version":
			eepromDbObj.ONIE_Version = e.Get("Value")
		case "Serial Number":
			eepromDbObj.Serial_Number = e.Get("Value")
		case "Vendor Name":
			eepromDbObj.Vendor_Name = e.Get("Value")
		case "Manufacturer":
			eepromDbObj.Manufacturer = e.Get("Value")
		case "Manufacture Country":
			eepromDbObj.Manufacture_Country = e.Get("Value")
		case "Platform Name":
			eepromDbObj.Platform_Name = e.Get("Value")
		case "Diag Version":
			eepromDbObj.Diag_Version = e.Get("Value")
		case "Label Revision":
			eepromDbObj.Label_Revision = e.Get("Value")
		case "Part Number":
			eepromDbObj.Part_Number = e.Get("Value")
		case "Product Name":
			eepromDbObj.Product_Name = e.Get("Value")
		case "Base MAC Address":
			eepromDbObj.Base_MAC_Address = e.Get("Value")
		case "Manufacture Date":
			eepromDbObj.Manufacture_Date = e.Get("Value")
		case "MAC Addresses":
			mac, _ := strconv.ParseInt(e.Get("Value"), 10, 16)
			eepromDbObj.MAC_Addresses = int(mac)
		}
	}

	return eepromDbObj
}

func (app *PlatformApp) getSysEepromFromDb(eeprom *ocbinds.OpenconfigPlatform_Components_Component_State, all bool) error {

	log.Infof("getSysEepromFromDb Enter")

	eepromDb := app.getEepromDbObj()

	empty := false
	removable := false
	name := "System Eeprom"
	location := "Slot 1"

	if all == true {
		eeprom.Empty = &empty
		eeprom.Removable = &removable
		eeprom.Name = &name
		eeprom.OperStatus = ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_ACTIVE
		eeprom.Location = &location

		if eepromDb.Product_Name != "" {
			eeprom.Id = &eepromDb.Product_Name
		}
		if eepromDb.Part_Number != "" {
			eeprom.PartNo = &eepromDb.Part_Number
		}
		if eepromDb.Serial_Number != "" {
			eeprom.SerialNo = &eepromDb.Serial_Number
		}
		if eepromDb.Base_MAC_Address != "" {
		}
		if eepromDb.Manufacture_Date != "" {
			eeprom.MfgDate = &eepromDb.Manufacture_Date
		}
		if eepromDb.Label_Revision != "" {
			eeprom.HardwareVersion = &eepromDb.Label_Revision
		}
		if eepromDb.Platform_Name != "" {
			eeprom.Description = &eepromDb.Platform_Name
		}
		if eepromDb.ONIE_Version != "" {
		}
		if eepromDb.MAC_Addresses != 0 {
		}
		if eepromDb.Manufacturer != "" {
			eeprom.MfgName = &eepromDb.Manufacturer
		}
		if eepromDb.Manufacture_Country != "" {
		}
		if eepromDb.Vendor_Name != "" {
			if eeprom.MfgName == nil {
				eeprom.MfgName = &eepromDb.Vendor_Name
			}
		}
		if eepromDb.Diag_Version != "" {
		}
		if eepromDb.Service_Tag != "" {
			if eeprom.SerialNo == nil {
				eeprom.SerialNo = &eepromDb.Service_Tag
			}
		}
		if eepromDb.Hardware_Version != "" {
			eeprom.HardwareVersion = &eepromDb.Hardware_Version
		}
		if eepromDb.Software_Version != "" {
			eeprom.SoftwareVersion = &eepromDb.Software_Version
		}
	} else {
		targetUriPath, _ := getYangPathFromUri(app.path.Path)
		switch targetUriPath {
		case "/openconfig-platform:components/component/state/name":
			eeprom.Name = &name
		case "/openconfig-platform:components/component/state/location":
			eeprom.Location = &location
		case "/openconfig-platform:components/component/state/empty":
			eeprom.Empty = &empty
		case "/openconfig-platform:components/component/state/removable":
			eeprom.Removable = &removable
		case "/openconfig-platform:components/component/state/oper-status":
			eeprom.OperStatus = ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_ACTIVE
		case "/openconfig-platform:components/component/state/id":
			if eepromDb.Product_Name != "" {
				eeprom.Id = &eepromDb.Product_Name
			}
		case "/openconfig-platform:components/component/state/part-no":
			if eepromDb.Part_Number != "" {
				eeprom.PartNo = &eepromDb.Part_Number
			}
		case "/openconfig-platform:components/component/state/serial-no":
			if eepromDb.Serial_Number != "" {
				eeprom.SerialNo = &eepromDb.Serial_Number
			}
			if eepromDb.Service_Tag != "" {
				if eeprom.SerialNo == nil {
					eeprom.SerialNo = &eepromDb.Service_Tag
				}
			}
		case "/openconfig-platform:components/component/state/mfg-date":
			if eepromDb.Manufacture_Date != "" {
				eeprom.MfgDate = &eepromDb.Manufacture_Date
			}
		case "/openconfig-platform:components/component/state/hardware-version":
			if eepromDb.Label_Revision != "" {
				eeprom.HardwareVersion = &eepromDb.Label_Revision
			}
			if eepromDb.Hardware_Version != "" {
				if eeprom.HardwareVersion == nil {
					eeprom.HardwareVersion = &eepromDb.Hardware_Version
				}
			}
		case "/openconfig-platform:components/component/state/description":
			if eepromDb.Platform_Name != "" {
				eeprom.Description = &eepromDb.Platform_Name
			}
		case "/openconfig-platform:components/component/state/mfg-name":
			if eepromDb.Manufacturer != "" {
				eeprom.MfgName = &eepromDb.Manufacturer
			}
			if eepromDb.Vendor_Name != "" {
				if eeprom.MfgName == nil {
					eeprom.MfgName = &eepromDb.Vendor_Name
				}
			}
		case "/openconfig-platform:components/component/state/software-version":
			if eepromDb.Software_Version != "" {
				eeprom.SoftwareVersion = &eepromDb.Software_Version
			}
		}
	}
	return nil
}

type CompStateDb struct {
	Serial string
	Model  string
}

func (app *PlatformApp) getCompStateDbObj(ifName string) CompStateDb {
	log.Infof("parseCompStateDb Enter ifName=%s", ifName)

	var compStateDbObj CompStateDb

	transceiverInfoTable := app.transceiverInfoTable[ifName].entry

	compStateDbObj.Serial = transceiverInfoTable.Get("serial")
	compStateDbObj.Model = transceiverInfoTable.Get("model")

	return compStateDbObj
}

func (app *PlatformApp) getCompStateFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_State, all bool, compName string) error {
	log.Infof("getCompStateFromDb Enter compName=%s", compName)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compStateDb := app.getCompStateDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/state/serial-no" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("serial") {
			oc_val.SerialNo = &compStateDb.Serial
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/state/part-no" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("model") {
			oc_val.PartNo = &compStateDb.Model
		}
	}

	return nil
}

type CompTransceiverStateDb struct {
	Connector    string
	Manufacturer string
	Vendor_Oui   string
	Vendor_Rev   string
	Serial       string
	Vendor_Date  string
}

func (app *PlatformApp) getCompTransceiverStateDbObj(ifName string) CompTransceiverStateDb {
	log.Infof("parseCompTransceiverStateDb Enter ifName=%s", ifName)

	var compTransceiverStateDbObj CompTransceiverStateDb

	transceiverInfoTable := app.transceiverInfoTable[ifName].entry

	compTransceiverStateDbObj.Connector = transceiverInfoTable.Get("connector")
	compTransceiverStateDbObj.Manufacturer = transceiverInfoTable.Get("manufacturer")
	compTransceiverStateDbObj.Vendor_Oui = transceiverInfoTable.Get("vendor_oui")
	compTransceiverStateDbObj.Vendor_Rev = transceiverInfoTable.Get("vendor_rev")
	compTransceiverStateDbObj.Serial = transceiverInfoTable.Get("serial")
	compTransceiverStateDbObj.Vendor_Date = transceiverInfoTable.Get("vendor_date")

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
			oc_val.VendorPart = &compTransceiverStateDb.Vendor_Oui
		}
	}
	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/vendor-rev" {
		transceiverInfoTable := app.transceiverInfoTable[ifName].entry
		if transceiverInfoTable.Has("vendor_rev") {
			oc_val.VendorRev = &compTransceiverStateDb.Vendor_Rev
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
			subMatchString := rex.FindAllString(compTransceiverStateDb.Vendor_Date, -1)
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

	transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry

	if transceiverDomSensorTable.Get("voltage") != "N/A" {
		compTransceiverStateSupplyVoltageDbObj.voltage, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("voltage"), 64)
	} else {
		compTransceiverStateSupplyVoltageDbObj.voltage, _ = strconv.ParseFloat("NaN", 64)
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

	transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry

	if transceiverDomSensorTable.Get("temperature") != "N/A" {
		compTransceiverPhysicalChannelStateLaserTemperatureDbObj.temperature, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("temperature"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserTemperatureDbObj.temperature, _ = strconv.ParseFloat("NaN", 64)
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
	tx1power float64
	tx2power float64
	tx3power float64
	tx4power float64
	tx5power float64
	tx6power float64
	tx7power float64
	tx8power float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateOutputPowerDbObj(ifName string) CompTransceiverPhysicalChannelStateOutputPowerDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateOutputPowerDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateOutputPowerDbObj CompTransceiverPhysicalChannelStateOutputPowerDb

	transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry

	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx1power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx1power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx2power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx2power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx3power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx3power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx4power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx4power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx5power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx5power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx6power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx6power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx7power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx7power"), 64)
	compTransceiverPhysicalChannelStateOutputPowerDbObj.tx8power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx8power"), 64)

	return compTransceiverPhysicalChannelStateOutputPowerDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateOutputPowerFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_OutputPower, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateOutputPowerFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateOutputPowerDb := app.getCompTransceiverPhysicalChannelStateOutputPowerDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/output-power/instant" {
		switch laneIndex {
		case 0:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx1power") {
				tx1power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx1power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx1power
			}
		case 1:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx2power") {
				tx2power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx2power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx2power
			}
		case 2:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx3power") {
				tx3power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx3power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx3power
			}
		case 3:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx4power") {
				tx4power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx4power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx4power
			}
		case 4:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx5power") {
				tx5power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx5power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx5power
			}
		case 5:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx6power") {
				tx6power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx6power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx6power
			}
		case 6:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx7power") {
				tx7power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx7power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx7power
			}
		case 7:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx8power") {
				tx8power := math.Floor(compTransceiverPhysicalChannelStateOutputPowerDb.tx8power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx8power
			}
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateInputPowerDb struct {
	rx1power float64
	rx2power float64
	rx3power float64
	rx4power float64
	rx5power float64
	rx6power float64
	rx7power float64
	rx8power float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateInputPowerDbObj(ifName string) CompTransceiverPhysicalChannelStateInputPowerDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateInputPowerDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateInputPowerDbObj CompTransceiverPhysicalChannelStateInputPowerDb

	transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry

	compTransceiverPhysicalChannelStateInputPowerDbObj.rx1power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx1power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx2power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx2power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx3power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx3power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx4power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx4power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx5power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx5power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx6power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx6power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx7power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx7power"), 64)
	compTransceiverPhysicalChannelStateInputPowerDbObj.rx8power, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("rx8power"), 64)

	return compTransceiverPhysicalChannelStateInputPowerDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateInputPowerFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_InputPower, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateInputPowerFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateInputPowerDb := app.getCompTransceiverPhysicalChannelStateInputPowerDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/input-power/instant" {
		switch laneIndex {
		case 0:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx1power") {
				rx1power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx1power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx1power
			}
		case 1:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx2power") {
				rx2power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx2power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx2power
			}
		case 2:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx3power") {
				rx3power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx3power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx3power
			}
		case 3:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx4power") {
				rx4power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx4power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx4power
			}
		case 4:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx5power") {
				rx5power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx5power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx5power
			}
		case 5:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx6power") {
				rx6power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx6power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx6power
			}
		case 6:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx7power") {
				rx7power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx7power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx7power
			}
		case 7:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("rx8power") {
				rx8power := math.Floor(compTransceiverPhysicalChannelStateInputPowerDb.rx8power*fractionDigits2) / fractionDigits2
				oc_val.Instant = &rx8power
			}
		}
	}

	return nil
}

type CompTransceiverPhysicalChannelStateLaserBiasCurrentDb struct {
	tx1bias float64
	tx2bias float64
	tx3bias float64
	tx4bias float64
	tx5bias float64
	tx6bias float64
	tx7bias float64
	tx8bias float64
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserBiasCurrentDbObj(ifName string) CompTransceiverPhysicalChannelStateLaserBiasCurrentDb {
	log.Infof("parseCompTransceiverPhysicalChannelStateLaserBiasCurrentDb Enter ifName=%s", ifName)

	var compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj CompTransceiverPhysicalChannelStateLaserBiasCurrentDb

	transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry

	if transceiverDomSensorTable.Get("tx1bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx1bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx1bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx1bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx2bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx2bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx2bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx2bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx3bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx3bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx3bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx3bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx4bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx4bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx4bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx4bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx5bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx5bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx5bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx5bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx6bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx6bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx6bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx6bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx7bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx7bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx7bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx7bias, _ = strconv.ParseFloat("NaN", 64)
	}
	if transceiverDomSensorTable.Get("tx8bias") != "N/A" {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx8bias, _ = strconv.ParseFloat(transceiverDomSensorTable.Get("tx8bias"), 64)
	} else {
		compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj.tx8bias, _ = strconv.ParseFloat("NaN", 64)
	}

	return compTransceiverPhysicalChannelStateLaserBiasCurrentDbObj
}

func (app *PlatformApp) getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(oc_val *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel_State_LaserBiasCurrent, all bool, compName string, laneIndex uint16) error {
	log.Infof("getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb Enter compName=%s laneIndex=%d", compName, laneIndex)

	ifName := strings.Replace(compName, "transceiver_", "", -1)
	compTransceiverPhysicalChannelStateLaserBiasCurrentDb := app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentDbObj(ifName)

	targetUriPath, _ := getYangPathFromUri(app.path.Path)

	if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-bias-current/instant" {
		switch laneIndex {
		case 0:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx1bias") {
				tx1bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx1bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx1bias
			}
		case 1:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx2bias") {
				tx2bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx2bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx2bias
			}
		case 2:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx3bias") {
				tx3bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx3bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx3bias
			}
		case 3:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx4bias") {
				tx4bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx4bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx4bias
			}
		case 4:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx5bias") {
				tx5bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx5bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx5bias
			}
		case 5:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx6bias") {
				tx6bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx6bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx6bias
			}
		case 6:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx7bias") {
				tx7bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx7bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx7bias
			}
		case 7:
			transceiverDomSensorTable := app.transceiverDomSensorTable[ifName].entry
			if transceiverDomSensorTable.Has("tx8bias") {
				tx8bias := math.Floor(compTransceiverPhysicalChannelStateLaserBiasCurrentDb.tx8bias*fractionDigits2) / fractionDigits2
				oc_val.Instant = &tx8bias
			}
		}
	}

	return nil
}

type CompTransceiverThresholdStateDb struct {
	temphighalarm   float64
	templowalarm    float64
	vcchighalarm    float64
	vcclowalarm     float64
	temphighwarning float64
	templowwarning  float64
	vcchighwarning  float64
	vcclowwarning   float64
	txpowerhighalarm float64
    txpowerlowalarm  float64
    rxpowerhighalarm float64
    rxpowerlowalarm  float64
    txbiashighalarm  float64
    txbiaslowalarm   float64
    txpowerhighwarning float64
    txpowerlowwarning  float64
    rxpowerhighwarning float64
    rxpowerlowwarning  float64
    txbiashighwarning  float64
    txbiaslowwarning   float64
}

func (app *PlatformApp) getCompTransceiverThresholdStateDbObj(ifName string) CompTransceiverThresholdStateDb {
	log.Infof("parseCompTransceiverThresholdStateDb Enter ifName=%s", ifName)

	var compTransceiverThresholdStateDbObj CompTransceiverThresholdStateDb

	transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry

	compTransceiverThresholdStateDbObj.temphighalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("temphighalarm"), 64)
	compTransceiverThresholdStateDbObj.templowalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("templowalarm"), 64)
	compTransceiverThresholdStateDbObj.vcchighalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcchighalarm"), 64)
	compTransceiverThresholdStateDbObj.vcclowalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcclowalarm"), 64)
	compTransceiverThresholdStateDbObj.temphighwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("temphighwarning"), 64)
	compTransceiverThresholdStateDbObj.templowwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("templowwarning"), 64)
	compTransceiverThresholdStateDbObj.vcchighwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcchighwarning"), 64)
	compTransceiverThresholdStateDbObj.vcclowwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("vcclowwarning"), 64)
    compTransceiverThresholdStateDbObj.txpowerhighalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerhighalarm"), 64)
    compTransceiverThresholdStateDbObj.txpowerlowalarm, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerlowalarm"), 64)
    compTransceiverThresholdStateDbObj.rxpowerhighalarm, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerhighalarm"), 64)
    compTransceiverThresholdStateDbObj.rxpowerlowalarm, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerlowalarm"), 64)
    compTransceiverThresholdStateDbObj.txbiashighalarm, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiashighalarm"), 64)
    compTransceiverThresholdStateDbObj.txbiaslowalarm, _   = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiaslowalarm"), 64)
    compTransceiverThresholdStateDbObj.txpowerhighwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerhighwarning"), 64)
    compTransceiverThresholdStateDbObj.txpowerlowwarning, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("txpowerlowwarning"), 64)
    compTransceiverThresholdStateDbObj.rxpowerhighwarning, _ = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerhighwarning"), 64)
    compTransceiverThresholdStateDbObj.rxpowerlowwarning, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("rxpowerlowwarning"), 64)
    compTransceiverThresholdStateDbObj.txbiashighwarning, _  = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiashighwarning"), 64)
    compTransceiverThresholdStateDbObj.txbiaslowwarning, _   = strconv.ParseFloat(transceiverDomThresholdTable.Get("txbiaslowwarning"), 64)

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
				temphighalarm := math.Floor(compTransceiverThresholdStateDb.temphighalarm*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureUpper = &temphighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("templowalarm") {
				templowalarm := math.Floor(compTransceiverThresholdStateDb.templowalarm*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureLower = &templowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcchighalarm") {
				vcchighalarm := math.Floor(compTransceiverThresholdStateDb.vcchighalarm*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageUpper = &vcchighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcclowalarm") {
				vcclowalarm := math.Floor(compTransceiverThresholdStateDb.vcclowalarm*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageLower = &vcclowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerhighalarm") {
				txpowerhighalarm := math.Floor(compTransceiverThresholdStateDb.txpowerhighalarm*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerUpper = &txpowerhighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerlowalarm") {
				txpowerlowalarm := math.Floor(compTransceiverThresholdStateDb.txpowerlowalarm*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerLower = &txpowerlowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerhighalarm") {
				rxpowerhighalarm := math.Floor(compTransceiverThresholdStateDb.rxpowerhighalarm*fractionDigits2) / fractionDigits2
				oc_val.InputPowerUpper = &rxpowerhighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerlowalarm") {
				rxpowerlowalarm := math.Floor(compTransceiverThresholdStateDb.rxpowerlowalarm*fractionDigits2) / fractionDigits2
				oc_val.InputPowerLower = &rxpowerlowalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiashighalarm") {
				txbiashighalarm := math.Floor(compTransceiverThresholdStateDb.txbiashighalarm*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentUpper = &txbiashighalarm
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiaslowalarm") {
				txbiaslowalarm := math.Floor(compTransceiverThresholdStateDb.txbiaslowalarm*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentLower = &txbiaslowalarm
			}
		}
	}

	if severityName == "WARNING" {
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("temphighwarning") {
				temphighwarning := math.Floor(compTransceiverThresholdStateDb.temphighwarning*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureUpper = &temphighwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-temperature-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("templowwarning") {
				templowwarning := math.Floor(compTransceiverThresholdStateDb.templowwarning*fractionDigits1) / fractionDigits1
				oc_val.LaserTemperatureLower = &templowwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcchighwarning") {
				vcchighwarning := math.Floor(compTransceiverThresholdStateDb.vcchighwarning*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageUpper = &vcchighwarning
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/supply-voltage-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("vcclowwarning") {
				vcclowwarning := math.Floor(compTransceiverThresholdStateDb.vcclowwarning*fractionDigits2) / fractionDigits2
				oc_val.SupplyVoltageLower = &vcclowwarning
			}
		}
	   if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-upper" {
        transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
        if transceiverDomThresholdTable.Has("txpowerhighwarning") {
            v := math.Floor(compTransceiverThresholdStateDb.txpowerhighwarning*fractionDigits2) / fractionDigits2
            oc_val.OutputPowerUpper = &v
        }
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/output-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txpowerlowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.txpowerlowwarning*fractionDigits2) / fractionDigits2
				oc_val.OutputPowerLower = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerhighwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.rxpowerhighwarning*fractionDigits2) / fractionDigits2
				oc_val.InputPowerUpper = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/input-power-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("rxpowerlowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.rxpowerlowwarning*fractionDigits2) / fractionDigits2
				oc_val.InputPowerLower = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-upper" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiashighwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.txbiashighwarning*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentUpper = &v
			}
		}
		if all || targetUriPath == "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state/laser-bias-current-lower" {
			transceiverDomThresholdTable := app.transceiverDomThresholdTable[ifName].entry
			if transceiverDomThresholdTable.Has("txbiaslowwarning") {
				v := math.Floor(compTransceiverThresholdStateDb.txbiaslowwarning*fractionDigits2) / fractionDigits2
				oc_val.LaserBiasCurrentLower = &v
			}
		}	
	}

	return nil
}

func (app *PlatformApp) doGetPlatformInfo() error {
	log.Infof("Preparing collection for platform info")

	var err error
	pf_cpts := app.getAppRootObject()
	var compName string
	var severityName string

	targetUriPath, _ := getYangPathFromUri(app.path.Path)
	switch targetUriPath {
	case "/openconfig-platform:components":
		log.Info("case /openconfig-platform:components root")
		pf_comp, _ := pf_cpts.NewComponent("System Eeprom")
		ygot.BuildEmptyTree(pf_comp)
		err = app.getSysEepromFromDb(pf_comp.State, true)
		if err != nil {
			break
		}

		for epItem, _ := range app.transceiverInfoTable {
			compName = "transceiver_" + epItem
			pf_comp, _ := pf_cpts.NewComponent(compName)
			ygot.BuildEmptyTree(pf_comp)

			err = app.getCompStateFromDb(pf_comp.State, true, compName)
			if err != nil {
				break
			}
			err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, true, compName)
			if err != nil {
				break
			}
			err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
			if err != nil {
				break
			}
			ifName := strings.Replace(compName, "transceiver_", "", -1)
			applPortTable := app.applPortTable[ifName].entry

			pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
			if pf_channel_0 != nil {
				ygot.BuildEmptyTree(pf_channel_0)
				err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
				if err != nil {
					break
				}
			}

			for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
				laneNum, _ := strconv.ParseUint(lane, 10, 16)
				pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
				if pf_channel != nil {
					ygot.BuildEmptyTree(pf_channel)
					err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
					if err != nil {
						break
					}
					err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
					if err != nil {
						break
					}
					err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
				}
			}
		}

	case "/openconfig-platform:components/component":
		log.Info("case /openconfig-platform:components/component root")
		compName = app.path.Var("name")
		if compName == "" {
			pf_comp, _ := pf_cpts.NewComponent("System Eeprom")
			ygot.BuildEmptyTree(pf_comp)
			err = app.getSysEepromFromDb(pf_comp.State, true)
			if err != nil {
				break
			}

			for epItem, _ := range app.transceiverInfoTable {
				compName = "transceiver_" + epItem
				pf_comp, _ := pf_cpts.NewComponent(compName)
				ygot.BuildEmptyTree(pf_comp)

				err = app.getCompStateFromDb(pf_comp.State, true, compName)
				if err != nil {
					break
				}
				err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, true, compName)
				if err != nil {
					break
				}
				err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
				if err != nil {
					break
				}
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
				if pf_channel_0 != nil {
					ygot.BuildEmptyTree(pf_channel_0)
					err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
					if err != nil {
						break
					}
				}

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
					if pf_channel != nil {
						ygot.BuildEmptyTree(pf_channel)
						err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
					}
				}
			}
		} else {
			if compName != "System Eeprom" && !strings.Contains(compName, "transceiver_Ethernet") {
				err = errors.New("Invalid component name")
				break
			}
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)

				if compName == "System Eeprom" {
					err = app.getSysEepromFromDb(pf_comp.State, true)
				}

				if strings.Contains(compName, "transceiver_Ethernet") {
					err = app.getCompStateFromDb(pf_comp.State, true, compName)
					if err != nil {
						break
					}
					err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, true, compName)
					if err != nil {
						break
					}
					err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
					if err != nil {
						break
					}
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
					if pf_channel_0 != nil {
						ygot.BuildEmptyTree(pf_channel_0)
						err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
						if err != nil {
							break
						}
					}

					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel)
							err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
							if err != nil {
								break
							}
							err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
							if err != nil {
								break
							}
							err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
						}
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		}
	case "/openconfig-platform:components/component/state":
		log.Info("case /openconfig-platform:components/component/state root")
		compName = app.path.Var("name")
		if compName == "System Eeprom" {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				err = app.getSysEepromFromDb(pf_comp.State, true)
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				err = app.getCompStateFromDb(pf_comp.State, true, compName)
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp.Transceiver)
				err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, true, compName)
				if err != nil {
					break
				}
				err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
				if err != nil {
					break
				}
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
				if pf_channel_0 != nil {
					ygot.BuildEmptyTree(pf_channel_0)
					err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
					if err != nil {
						break
					}
				}

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
					if pf_channel != nil {
						ygot.BuildEmptyTree(pf_channel)
						err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp.Transceiver.State)
				err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, true, compName)
				if err != nil {
					break
				}
				err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/supply-voltage":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/supply-voltage root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, true, compName)
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
				if pf_channel_0 != nil {
					ygot.BuildEmptyTree(pf_channel_0)
					err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
					if err != nil {
						break
					}
				}

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
					if pf_channel != nil {
						ygot.BuildEmptyTree(pf_channel)
						err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
						if err != nil {
							break
						}
						err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				indexName := app.path.Var("index")
				if indexName == "" {
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					pf_channel_0, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(0)
					if pf_channel_0 != nil {
						ygot.BuildEmptyTree(pf_channel_0)
						err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel_0.State.LaserTemperature, true, compName)
						if err != nil {
							break
						}
					}

					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						pf_channel, _ := pf_comp.Transceiver.PhysicalChannels.NewChannel(uint16(laneNum))
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel)
							err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
							if err != nil {
								break
							}
							err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
							if err != nil {
								break
							}
							err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
						}
					}
				} else {
					compIndex, _ := strconv.ParseUint(indexName, 10, 16)
					log.Info("compIndex =", compIndex)
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					if compIndex == 0 {
						pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel)
							err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel.State.LaserTemperature, true, compName)
						} else {
							err = errors.New("Invalid input component index")
						}
					} else {
						for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
							laneNum, _ := strconv.ParseUint(lane, 10, 16)
							if uint16(laneNum) == uint16(compIndex) {
								pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
								if pf_channel != nil {
									ygot.BuildEmptyTree(pf_channel)
									err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
									if err != nil {
										break
									}
									err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
									if err != nil {
										break
									}
									err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
								} else {
									err = errors.New("Invalid input component index")
								}
								break
							} else {
								err = errors.New("Invalid input component index")
							}
						}
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
				log.Info("compIndex =", compIndex)
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				if compIndex == 0 {
					pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
					if pf_channel != nil {
						ygot.BuildEmptyTree(pf_channel.State)
						err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel.State.LaserTemperature, true, compName)
					} else {
						err = errors.New("Invalid input component index")
					}
				} else {
					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						if uint16(laneNum) == uint16(compIndex) {
							pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
							if pf_channel != nil {
								ygot.BuildEmptyTree(pf_channel.State)
								err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
								if err != nil {
									break
								}
								err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
								if err != nil {
									break
								}
								err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
							} else {
								err = errors.New("Invalid input component index")
							}
							break
						} else {
							err = errors.New("Invalid input component index")
						}
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-temperature":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-temperature root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
				log.Info("compIndex =", compIndex)

				if compIndex == 0 {
					pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
					if pf_channel != nil {
						ygot.BuildEmptyTree(pf_channel.State)
						err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel.State.LaserTemperature, true, compName)
					} else {
						err = errors.New("Invalid input component index")
					}
				} else {
					err = errors.New("Invalid input component index")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/output-power":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/output-power root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
				log.Info("compIndex =", compIndex)
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					if uint16(laneNum) == uint16(compIndex) {
						pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel.State)
							err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, true, compName, uint16(index))
						} else {
							err = errors.New("Invalid input component index")
						}
						break
					} else {
						err = errors.New("Invalid input component index")
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/input-power":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/input-power root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
				log.Info("compIndex =", compIndex)
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					if uint16(laneNum) == uint16(compIndex) {
						pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel.State)
							err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, true, compName, uint16(index))
						} else {
							err = errors.New("Invalid input component index")
						}
						break
					} else {
						err = errors.New("Invalid input component index")
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-bias-current":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-bias-current root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
				log.Info("compIndex =", compIndex)
				ifName := strings.Replace(compName, "transceiver_", "", -1)
				applPortTable := app.applPortTable[ifName].entry

				for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
					laneNum, _ := strconv.ParseUint(lane, 10, 16)
					if uint16(laneNum) == uint16(compIndex) {
						pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel.State)
							err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, true, compName, uint16(index))
						} else {
							err = errors.New("Invalid input component index")
						}
						break
					} else {
						err = errors.New("Invalid input component index")
					}
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}
	case "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state":
		log.Info("case /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state root")
		compName = app.path.Var("name")
		if strings.Contains(compName, "transceiver_Ethernet") {
			pf_comp := pf_cpts.Component[compName]
			if pf_comp != nil {
				ygot.BuildEmptyTree(pf_comp)
				severityName = app.path.Var("severity")
				if strings.Contains(severityName, "CRITICAL") {
					pf_threshold := pf_comp.Transceiver.Thresholds.Threshold[ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_CRITICAL]
					if pf_threshold != nil {
						ygot.BuildEmptyTree(pf_threshold.State)
						err = app.getCompTransceiverThresholdStateFromDb(pf_threshold.State, true, compName, severityName)
					} else {
						err = errors.New("Invalid input severity name")
					}
				} else if strings.Contains(severityName, "WARNING") {
					pf_threshold := pf_comp.Transceiver.Thresholds.Threshold[ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_WARNING]
					if pf_threshold != nil {
						ygot.BuildEmptyTree(pf_threshold.State)
						err = app.getCompTransceiverThresholdStateFromDb(pf_threshold.State, true, compName, severityName)
					} else {
						err = errors.New("Invalid input severity name")
					}
				} else {
					err = errors.New("Invalid input severity name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid component name ")
		}

	default:
		if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/state") {
			compName = app.path.Var("name")
			if compName == "System Eeprom" {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					err = app.getSysEepromFromDb(pf_comp.State, false)
				} else {
					err = errors.New("Invalid input component name")
				}
			} else if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					err = app.getCompStateFromDb(pf_comp.State, false, compName)
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/supply-voltage") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					err = app.getCompTransceiverStateSupplyVoltageFromDb(pf_comp.Transceiver.State.SupplyVoltage, false, compName)
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					err = app.getCompTransceiverStateFromDb(pf_comp.Transceiver.State, false, compName)
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-temperature") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)

					if compIndex == 0 {
						pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
						if pf_channel != nil {
							ygot.BuildEmptyTree(pf_channel.State)
							err = app.getCompTransceiverPhysicalChannelStateLaserTemperatureFromDb(pf_channel.State.LaserTemperature, false, compName)
						} else {
							err = errors.New("Invalid input component index")
						}
					} else {
						err = errors.New("Invalid input component index")
					}
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/output-power") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						if uint16(laneNum) == uint16(compIndex) {
							pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
							if pf_channel != nil {
								ygot.BuildEmptyTree(pf_channel.State)
								err = app.getCompTransceiverPhysicalChannelStateOutputPowerFromDb(pf_channel.State.OutputPower, false, compName, uint16(index))
							} else {
								err = errors.New("Invalid input component index")
							}
							break
						} else {
							err = errors.New("Invalid input component index")
						}
					}
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/input-power") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						if uint16(laneNum) == uint16(compIndex) {
							pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
							if pf_channel != nil {
								ygot.BuildEmptyTree(pf_channel.State)
								err = app.getCompTransceiverPhysicalChannelStateInputPowerFromDb(pf_channel.State.InputPower, false, compName, uint16(index))
							} else {
								err = errors.New("Invalid input component index")
							}
							break
						} else {
							err = errors.New("Invalid input component index")
						}
					}
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state/laser-bias-current") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					compIndex, _ := strconv.ParseUint(app.path.Var("index"), 10, 16)
					ifName := strings.Replace(compName, "transceiver_", "", -1)
					applPortTable := app.applPortTable[ifName].entry

					for index, lane := range strings.Split(applPortTable.Get("lanes"), ",") {
						laneNum, _ := strconv.ParseUint(lane, 10, 16)
						if uint16(laneNum) == uint16(compIndex) {
							pf_channel := pf_comp.Transceiver.PhysicalChannels.Channel[uint16(compIndex)]
							if pf_channel != nil {
								ygot.BuildEmptyTree(pf_channel.State)
								err = app.getCompTransceiverPhysicalChannelStateLaserBiasCurrentFromDb(pf_channel.State.LaserBiasCurrent, false, compName, uint16(index))
							} else {
								err = errors.New("Invalid input component index")
							}
							break
						} else {
							err = errors.New("Invalid input component index")
						}
					}
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else if isSubtreeRequest(targetUriPath, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/thresholds/threshold/state") {
			compName = app.path.Var("name")
			if strings.Contains(compName, "transceiver_Ethernet") {
				pf_comp := pf_cpts.Component[compName]
				if pf_comp != nil {
					ygot.BuildEmptyTree(pf_comp)
					severityName = app.path.Var("severity")
					if strings.Contains(severityName, "CRITICAL") {
						pf_threshold := pf_comp.Transceiver.Thresholds.Threshold[ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_CRITICAL]
						if pf_threshold != nil {
							ygot.BuildEmptyTree(pf_threshold.State)
							err = app.getCompTransceiverThresholdStateFromDb(pf_threshold.State, false, compName, severityName)
						} else {
							err = errors.New("Invalid input severity name")
						}
					} else if strings.Contains(severityName, "WARNING") {
						pf_threshold := pf_comp.Transceiver.Thresholds.Threshold[ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_WARNING]
						if pf_threshold != nil {
							ygot.BuildEmptyTree(pf_threshold.State)
							err = app.getCompTransceiverThresholdStateFromDb(pf_threshold.State, false, compName, severityName)
						} else {
							err = errors.New("Invalid input severity name")
						}
					} else {
						err = errors.New("Invalid input severity name")
					}
				} else {
					err = errors.New("Invalid input component name")
				}
			} else {
				err = errors.New("Invalid input component name")
			}
		} else {
			err = errors.New("Invalid Path")
		}
	}
	return err
}