package transformer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	NODE_CFG_TBL    = "NODE_CFG"
	EEPROM_INFO_TBL = "EEPROM_INFO"

	IC_NAME_PREFIX  = "integrated_circuit"
	CHASSIS_PREFIX  = "chassis"
	SYS_EEPROM_NAME = "System Eeprom"

	/** Upper-level URIs **/
	COMP    = "/openconfig-platform:components/component"
	COMP_ST = "/openconfig-platform:components/component/state"

	/** Supported oc-platform component state URIs **/
	COMP_STATE_EMPTY             = "/openconfig-platform:components/component/state/empty"
	COMP_STATE_FIRM_VER          = "/openconfig-platform:components/component/state/firmware-version"
	COMP_STATE_HW_VER            = "/openconfig-platform:components/component/state/hardware-version"
	COMP_STATE_INSTALL_POSITION  = "/openconfig-platform:components/component/state/install-position"
	COMP_STATE_INSTALL_COMPONENT = "/openconfig-platform:components/component/state/install-component"
	COMP_STATE_MFG_DATE          = "/openconfig-platform:components/component/state/mfg-date"
	COMP_STATE_MFG_NAME          = "/openconfig-platform:components/component/state/mfg-name"
	COMP_STATE_NAME              = "/openconfig-platform:components/component/state/name"
	COMP_STATE_PART_NO           = "/openconfig-platform:components/component/state/part-no"
	COMP_STATE_SERIAL_NO         = "/openconfig-platform:components/component/state/serial-no"
	COMP_STATE_TYPE              = "/openconfig-platform:components/component/state/type"
	COMP_STATE_PARENT            = "/openconfig-platform:components/component/state/parent"
	COMP_STATE_TEMP_CTR          = "/openconfig-platform:components/component/state/temperature"
)

type componentType int64

const (
	CompTypeInvalid componentType = iota
	CompTypeIC
	CompTypeSysEeprom
)

/* Structures to read syseeprom from redis-db */
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

type PathType int

const (
	/* Represents all paths under /components/component */
	AllPaths PathType = iota
	/* Represents all paths under /components/component/state */
	StatePaths
)

func (pt PathType) String() string {
	switch pt {
	case AllPaths:
		return "AllPaths"
	case StatePaths:
		return "StatePaths"
	}
	return fmt.Sprintf("%s", pt)
}

func (ct componentType) String() string {
	switch ct {
	case CompTypeInvalid:
		return "CompTypeInvalid"
	case CompTypeIC:
		return "CompTypeIC"
	case CompTypeSysEeprom:
		return "CompTypeSysEeprom"
	}
	return fmt.Sprintf("%s", ct)
}

var compTblMap = map[componentType][]string{
	CompTypeIC:        {NODE_CFG_TBL, IC_NAME_PREFIX + "*"},
	CompTypeSysEeprom: {EEPROM_INFO_TBL, "*"},
}

func init() {
	XlateFuncBind("Subscribe_pfm_components_xfmr", Subscribe_pfm_components_xfmr)
	XlateFuncBind("DbToYang_pfm_components_xfmr", DbToYang_pfm_components_xfmr)
}

var compTypeCache sync.Map

func validICName(name *string) bool {
	if name == nil || *name == "" {
		return false
	}
	// Expect node name of form integrated_circuitX, where X is an integer
	if !strings.HasPrefix(*name, IC_NAME_PREFIX) {
		return false
	}

	sp := strings.SplitAfter(*name, IC_NAME_PREFIX)
	if len(sp) < 2 {
		return false
	}

	if _, err := strconv.Atoi(sp[1]); err != nil {
		return false
	}
	return true
}

func validSysEepromName(name string) bool {
	return name == SYS_EEPROM_NAME || name == "eeprom" || name == CHASSIS_PREFIX
}

func getCompTypeByName(compName string) (componentType, error) {
	switch {
	case validICName(&compName):
		return CompTypeIC, nil
	case validSysEepromName(compName):
		return CompTypeSysEeprom, nil
	default:
		return CompTypeInvalid, fmt.Errorf("component name %s did not match with supported types.", compName)
	}
}

func keyInDbTable(tableName, key string, d *db.DB) bool {
	if d == nil {
		return false
	}
	log.V(3).Infof("keyInDbTable: tableName=%s, key=%s, db=%s", tableName, key, d.Name())
	keys, err := d.GetKeysPattern(&db.TableSpec{Name: tableName}, db.Key{Comp: []string{key}})
	if err != nil {
		return false
	}
	return len(keys) > 0
}

func getCompType(name string, d *db.DB) componentType {
	if name == "*" {
		return CompTypeInvalid
	}
	if val, ok := compTypeCache.Load(name); ok {
		return val.(componentType)
	}
	compType, err := getCompTypeByName(name)
	if err == nil {
		compTypeCache.Store(name, compType)
		return compType
	}
	if d != nil {
		if keyInDbTable(EEPROM_INFO_TBL, name, d) {
			compTypeCache.Store(name, CompTypeSysEeprom)
			return CompTypeSysEeprom
		}
	}
	return CompTypeInvalid
}

var Subscribe_pfm_components_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams

	// Try extracting key from uri first; fallback to requestURI if empty
	reqPathInfo := NewPathInfo(inParams.uri)
	key := reqPathInfo.Var("name")
	if key == "" {
		reqPathInfo = NewPathInfo(inParams.requestURI)
		key = reqPathInfo.Var("name")
	}

	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr uri (%v) key(%s) mode(%v) +++", inParams.uri, key, inParams.subscProc)
	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr requestUri (%v) +++", inParams.requestURI)

	targetUriPath, err := getYangPathFromUri(reqPathInfo.Path)
	if err != nil {
		return result, err
	}

	if key == "" || validSysEepromName(key) || strings.Contains(key, "_sensor") {
		/* no need to verify dB data if we are requesting ALL
		   components, System Eeprom, or if request is for sensor */
		result.isVirtualTbl = true
		return result, err
	}

	if inParams.subscProc == TRANSLATE_EXISTS {
		return translateExists(inParams, key)
	}
	if inParams.subscProc == TRANSLATE_SUBSCRIBE {
		return translateSubscribe(inParams, key, targetUriPath)
	}

	return result, err
}

func getPfmRootObject(s *ygot.GoStruct) *ocbinds.OpenconfigPlatform_Components {
	if s == nil {
		return nil
	}
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.Components
}

/* Helper for the main subscribe transformer handling the TRANSLATE_EXISTS case. */
func translateExists(inParams XfmrSubscInParams, key string) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	dbNum := db.StateDB
	d := inParams.dbs[dbNum]
	if d == nil {
		return result, fmt.Errorf("translateExists: No usable DB client in inParams (checked %v and %v)", db.StateDB.Name(), db.ConfigDB.Name())
	}
	compType := getCompType(key, d)
	if compType == CompTypeInvalid {
		return result, nil
	}
	tblInfo, ok := compTblMap[compType]
	if !ok || len(tblInfo) == 0 {
		return result, errors.New("table not found or is empty.")
	}
	tblName := tblInfo[0]
	tblKey := key
	result.dbDataMap = RedisDbSubscribeMap{dbNum: {tblName: {tblKey: {}}}}
	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr result: %v %v %v +++", dbNum, tblName, tblKey)
	return result, nil
}

/* Helper for the main subscribe transformer handling the TRANSLATE_SUBSCRIBE case. */
func translateSubscribe(inParams XfmrSubscInParams, key, targetUriPath string) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	/* Handle TRANSLATE_SUBSCRIBE by expanding the wildcard yang key to a set of
	 * DB tables and keys.  If the key is not a wildcard then identify the set
	 * of DB tables and keys which apply to it. */
	result.isVirtualTbl = false
	result.needCache = true
	result.onChange = OnchangeEnable
	result.nOpts = &notificationOpts{mInterval: 0, pType: OnChange}

	/* Use the requested path to create a positive filter of component types to
	 * process.  Note that a completely empty filter means no filtering is
	 * required. */
	compTypeFilter := []componentType{}
	cType := getCompType(key, inParams.dbs[db.StateDB])
	if cType == CompTypeInvalid {
		return result, nil
	}
	compTypeFilter = []componentType{cType}
	for cType, tblNames := range compTblMap {
		if len(tblNames) < 2 {
			continue
		}
		/* An empty filter means no filtering is required. */
		if len(compTypeFilter) > 0 {
			/* Filtering is required, skip all component types not present in the
			 * filter. */
			filteredOut := true
			for _, ct := range compTypeFilter {
				if ct == cType {
					filteredOut = false
					break
				}
			}
			if filteredOut {
				continue
			}
		}
		tblName := tblNames[0]
		tblKey := tblNames[1]
		tblDb := db.StateDB
		tblKey = key
		if result.dbDataMap[tblDb] == nil {
			result.dbDataMap[tblDb] = make(map[string]map[string]map[string]string)
		}
		if result.dbDataMap[tblDb][tblName] == nil {
			result.dbDataMap[tblDb][tblName] = make(map[string]map[string]string)
		}
		if result.dbDataMap[tblDb][tblName][tblKey] == nil {
			result.dbDataMap[tblDb][tblName][tblKey] = map[string]string{}
		}
	}
	if log.V(3) {
		for db, _ := range result.dbDataMap {
			for tbl, _ := range result.dbDataMap[db] {
				for k, v := range result.dbDataMap[db][tbl] {
					log.Infof("+++ Subscribe_pfm_components_xfmr result: DB=%d, Table=%s, Key=%s, Flds=%v +++", db, tbl, k, v)
				}
			}
		}
	}
	return result, nil
}

/* Get a list of all table entries available */
func getAllTableEntries(d *db.DB, tblName string, key string) ([]string, error) {
	if tblName == "" || key == "" {
		return nil, errors.New("getAllTableEntries: empty table name or key.")
	}
	keyList, err := d.GetKeysPattern(&(db.TableSpec{Name: tblName}), db.Key{Comp: []string{key}})
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, v := range keyList {
		if len(v.Comp) == 0 {
			continue
		}
		ret = append(ret, strings.Join(v.Comp, d.Opts.KeySeparator))
	}
	return ret, nil
}

/* Filling in the config and state info for integrated circuits available in Redis DB */
func fillICInfo(comp *ocbinds.OpenconfigPlatform_Components_Component,
	name string, targetUriPath string, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct) error {
	/* Integrated-circuits have the following subtrees to populate:
	 *   ...component/config
	 *   ...component/state
	 *   ...component/integrated-circuit
	 *   ...component/integrated-circuit/config
	 *   ...component/integrated-circuit/state
	 * Decide now which subtrees to fill based on the request. */
	var all, compSt bool
	if targetUriPath == COMP {
		all = true
	} else if strings.HasPrefix(targetUriPath, COMP_ST) {
		compSt = true
	}
	log.V(3).Infof("dbToYangIC: name %s targetUriPath %s", name, targetUriPath)
	ygot.BuildEmptyTree(comp.IntegratedCircuit)
	ygot.BuildEmptyTree(comp.IntegratedCircuit.State)
	/* Handle component state paths: name, type, parent, fully-qualified-name */
	if all || compSt {
		var stName, stType, stParent bool
		switch targetUriPath {
		case COMP_ST:
			stName, stType = true, true
		case COMP_STATE_NAME:
			stName = true
		case COMP_STATE_TYPE:
			stType = true
		case COMP_STATE_PARENT:
			stParent = true
		default:
			/* Unsupported path or /components/component */
		}
		if all || stName {
			comp.State.Name = &name
		}
		if all || stType {
			comp.State.Type, _ = comp.State.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_HARDWARE_COMPONENT_INTEGRATED_CIRCUIT)
		}
		if all || stParent {
			parentChassis := CHASSIS_PREFIX
			comp.State.Parent = &parentChassis
		}
	}
	return nil
}

func getEepromDbObj(d *db.DB) EepromDb {
	var eepromDbObj EepromDb
	if d == nil {
		return eepromDbObj
	}

	tbl, err := d.GetTable(&db.TableSpec{Name: EEPROM_INFO_TBL})
	if err != nil {
		log.Error("EEPROM_INFO table get failed!")
		return eepromDbObj
	}

	keys, _ := tbl.GetKeys()
	for _, key := range keys {
		e, kerr := tbl.GetEntry(key)
		if kerr != nil {
			continue
		}
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

func fillSysEepromInfo(comp *ocbinds.OpenconfigPlatform_Components_Component,
	name string, targetUriPath string, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct) error {

	var all bool
	if targetUriPath == COMP || targetUriPath == COMP_ST {
		all = true
	}

	if comp.Config == nil {
		comp.Config = &ocbinds.OpenconfigPlatform_Components_Component_Config{}
	}
	if comp.State == nil {
		comp.State = &ocbinds.OpenconfigPlatform_Components_Component_State{}
	}

	stateDb := dbs[db.StateDB]
	eepromDb := getEepromDbObj(stateDb)

	empty := false
	removable := false
	sysName := SYS_EEPROM_NAME
	location := "Slot 1"

	comp.Name = &sysName
	comp.Config.Name = &sysName

	eeprom := comp.State

	if all {
		eeprom.Empty = &empty
		eeprom.Removable = &removable
		eeprom.Name = &sysName
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
		if eepromDb.Manufacture_Date != "" {
			eeprom.MfgDate = &eepromDb.Manufacture_Date
		}
		if eepromDb.Label_Revision != "" {
			eeprom.HardwareVersion = &eepromDb.Label_Revision
		}
		if eepromDb.Platform_Name != "" {
			eeprom.Description = &eepromDb.Platform_Name
		}
		if eepromDb.Manufacturer != "" {
			eeprom.MfgName = &eepromDb.Manufacturer
		}
		if eepromDb.Vendor_Name != "" {
			if eeprom.MfgName == nil {
				eeprom.MfgName = &eepromDb.Vendor_Name
			}
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
		switch targetUriPath {
		case "/openconfig-platform:components/component/state/name":
			eeprom.Name = &sysName
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

/* Helper to go from a component type to the type specific helper which reads
 * the DB data and populates the ocbinds structs.
 * createCompAndFuncCall - when fetching /components/component
 * getSysComponents - when fetching the following paths:
 *   /components/component[name=<compName>]
 *   /components/component[name=<compName>]/config
 *   /components/component[name=<compName>]/state
 */
func compTypeToFuncCall(cType componentType, compName, subKey string, pfComp *ocbinds.OpenconfigPlatform_Components_Component, targetUriPath string, dbs [db.MaxDB]*db.DB, pType PathType, ygRoot *ygot.GoStruct) error {
	log.V(3).Infof("compTypeToFuncCall with name=%s type=%v pType=%v", compName, cType, pType)
	ygot.BuildEmptyTree(pfComp)
	switch cType {
	case CompTypeIC:
		return fillICInfo(pfComp, compName, targetUriPath, dbs, ygRoot)
	case CompTypeSysEeprom:
		return fillSysEepromInfo(pfComp, compName, targetUriPath, dbs, ygRoot)
	}
	return errors.New("Invalid component type")
}

func createCompAndFuncCall(pfCpts *ocbinds.OpenconfigPlatform_Components, targetUriPath string, compType componentType, inParams XfmrParams, tblName string, tblKey string) {
	var compNames []string
	var err error
	dbs := inParams.dbs
	d := dbs[db.StateDB]
	cfgdb := dbs[db.ConfigDB]
	switch compType {
	case CompTypeIC:
		compNames, err = getAllTableEntries(cfgdb, tblName, tblKey)
	case CompTypeSysEeprom:
		compNames = []string{SYS_EEPROM_NAME}
	default:
		compNames, err = getAllTableEntries(d, tblName, tblKey)
	}
	if err != nil {
		log.V(3).Info(err)
	}

	for _, compAndKey := range compNames {
		compKeys := strings.Split(compAndKey, d.Opts.KeySeparator)
		if len(compKeys) == 0 {
			continue
		}
		comp := compKeys[0]
		derivedCompType := getCompType(comp, d)
		if derivedCompType != compType {
			continue
		}
		pfComp := pfCpts.Component[comp]
		if pfComp == nil {
			pfComp, err = pfCpts.NewComponent(comp)
			if err != nil {
				log.V(3).Infof("Component creation failed with NewComponent for comp %v; err = %v", comp, err)
				continue
			}
			ygot.BuildEmptyTree(pfComp)
		}

		if err = compTypeToFuncCall(compType, comp, "", pfComp, targetUriPath, dbs, AllPaths, inParams.ygRoot); err != nil {
			log.V(3).Info(err)
		}
	}
}

/* Main workhorse of the DbToYang transformer.  The get is either for the entire
 * component list (/components/component) or for a specific component name; no
 * wildcards are handled here. */
func getSysComponents(pf_cpts *ocbinds.OpenconfigPlatform_Components, targetUriPath string, inParams XfmrParams, compName, subKey string) error {

	log.V(3).Infof("Preparing dB for system components")

	dbs := inParams.dbs
	ygRoot := inParams.ygRoot

	var err error
	d := dbs[db.StateDB]
	log.V(3).Infof("getSysComponents: compName: %s targetUriPath: %s", compName, targetUriPath)
	switch targetUriPath {
	case COMP:
		log.V(3).Infof("compName: %v", compName)
		subCompName := "" /* Get all subcomponents */
		if compName == "" {
			/* Handle all component types except for ports, they will be handled just below. */
			for cType, tbl := range compTblMap {
				tblName := tbl[0]
				createCompAndFuncCall(pf_cpts, targetUriPath, cType, inParams, tblName, tbl[1])
			}
		} else {
			compType := getCompType(compName, d)
			if compType == CompTypeInvalid {
				return nil
			}
			pf_comp, ok := pf_cpts.Component[compName]
			if !ok || pf_comp == nil {
				var errNew error
				pf_comp, errNew = pf_cpts.NewComponent(compName)
				if errNew != nil {
					return fmt.Errorf("invalid input component name: %s", compName)
				}
			}
			ygot.BuildEmptyTree(pf_comp)
			if err = compTypeToFuncCall(compType, compName, subCompName, pf_comp, targetUriPath, dbs, AllPaths, ygRoot); err != nil {
				log.V(3).Info(err)
			}
		}
	case COMP_ST:
		compType := getCompType(compName, d)
		if compType == CompTypeInvalid {
			return nil
		}
		pf_comp, ok := pf_cpts.Component[compName]
		if !ok || pf_comp == nil {
			var errNew error
			pf_comp, errNew = pf_cpts.NewComponent(compName)
			if errNew != nil {
				return fmt.Errorf("invalid input component name for state path: %s", compName)
			}
		}
		ygot.BuildEmptyTree(pf_comp)
		ygot.BuildEmptyTree(pf_comp.State)
		if err = compTypeToFuncCall(compType, compName, subKey, pf_comp, targetUriPath, dbs, StatePaths, ygRoot); err != nil {
			log.V(3).Info(err)
		}
	default:
		/* The following cases are handled above:
		 *   /components/component
		 *   /components/component[name=<component_name>]
		 *   /components/component[name=<component_name>]/config
		 *   /components/component[name=<component_name>]/state
		 * so the request must be for a specific component's leaf or subtree,
		 * e.g. /components/component[name=integrated_circuit]/integrated-circuit */
		// TODO - Can we de-dup this code with compTypeToFuncCall?  No good way to set pathType...
		compType := getCompType(compName, d)
		if compType == CompTypeInvalid {
			return nil
		}
		pf_comp, ok := pf_cpts.Component[compName]
		if !ok || pf_comp == nil {
			var errNew error
			pf_comp, errNew = pf_cpts.NewComponent(compName)
			if errNew != nil {
				return fmt.Errorf("invalid input component name: %s", compName)
			}
		}
		ygot.BuildEmptyTree(pf_comp)
		switch compType {
		case CompTypeIC:
			return fillICInfo(pf_comp, compName, targetUriPath, inParams.dbs, inParams.ygRoot)
		case CompTypeSysEeprom:
			return fillSysEepromInfo(pf_comp, compName, targetUriPath, inParams.dbs, inParams.ygRoot)
		default:
			return fmt.Errorf("Unhandled Component: %s", compName)
		}
	}
	return err
}

var DbToYang_pfm_components_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	log.V(3).Infof("DbToYang_pfm_components_xfmr: %s, path: %s, vars: %v",
		pathInfo.Template, pathInfo.Path, pathInfo.Vars)

	if !strings.Contains(inParams.requestUri, "/openconfig-platform:components") {
		return errors.New("Component not supported")
	}
	log.V(3).Info("inParams.Uri:", inParams.requestUri)
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return err
	}

	/* Extract the component name (key), it may be empty ("") if the get is for
	 * the entire list/container (/components/component) */
	compName := pathInfo.Var("name")
	if compName == "" {
		reqPathInfo := NewPathInfo(inParams.requestUri)
		compName = reqPathInfo.Var("name")
	}
	/*subkey is second level key set to null which is not used for now*/
	subKey := ""

	if compName != "" {
		d := inParams.dbs[db.StateDB]
		cType := getCompType(compName, d)

		if cType == CompTypeSysEeprom {
			return getSysComponents(getPfmRootObject(inParams.ygRoot), targetUriPath, inParams, compName, subKey)
		}
	}

	return getSysComponents(getPfmRootObject(inParams.ygRoot), targetUriPath, inParams, compName, subKey)
}
