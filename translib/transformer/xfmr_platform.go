package transformer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	CONTROLLER_CARD_TBL = "CONTROLLER_CARD_INFO"
	NODE_CFG_TBL        = "NODE_CFG"
	EEPROM_INFO_TBL     = "EEPROM_INFO"
	TRANSCEIVER_TBL     = "TRANSCEIVER_INFO"
	TRANSCEIVER_STATUS  = "TRANSCEIVER_STATUS"
	TRANSCEIVER_DOM     = "TRANSCEIVER_DOM_SENSOR"

	XCVR_LANE_LIMIT = 8

	XCVR_KEY_PREFIX = "Ethernet"
	IC_NAME_PREFIX  = "integrated_circuit"
	CHASSIS_PREFIX  = "chassis"
	SYS_EEPROM_NAME = "System Eeprom"
	SW_COMP_TBL     = "SW_COMP_INFO"
	BOOTL_TYPE      = "BOOT_LOADER"
	OS_TYPE         = "OPERATING_SYSTEM"
	NW_STACK_TYPE   = "SOFTWARE_MODULE"

	NW_STACK_PREFIX = "network_stack"
	OS_PREFIX       = "os"
	BOOTL_PREFIX    = "boot_loader"

	/** Transceiver status values **/
	SFP_STATUS_REMOVED  = "0"
	SFP_STATUS_INSERTED = "1"

	/** Upper-level URIs **/
	COMP     = "/openconfig-platform:components/component"
	COMP_ST  = "/openconfig-platform:components/component/state"
	COMP_CFG = "/openconfig-platform:components/component/config"

	/** Supported oc-platform component state URIs **/
	COMP_STATE_EMPTY             = "/openconfig-platform:components/component/state/empty"
	COMP_STATE_FIRM_VER          = "/openconfig-platform:components/component/state/firmware-version"
	COMP_STATE_HW_VER            = "/openconfig-platform:components/component/state/hardware-version"
	COMP_STATE_INSTALL_POSITION  = "/openconfig-platform:components/component/state/install-position"
	COMP_STATE_INSTALL_COMPONENT = "/openconfig-platform:components/component/state/install-component"
	COMP_STATE_MFG_DATE          = "/openconfig-platform:components/component/state/mfg-date"
	COMP_STATE_MFG_NAME          = "/openconfig-platform:components/component/state/mfg-name"
	COMP_STATE_NAME              = "/openconfig-platform:components/component/state/name"
	COMP_STATE_OPER_STATUS       = "/openconfig-platform:components/component/state/oper-status"
	COMP_STATE_PART_NO           = "/openconfig-platform:components/component/state/part-no"
	COMP_STATE_REMOVABLE         = "/openconfig-platform:components/component/state/removable"
	COMP_STATE_SERIAL_NO         = "/openconfig-platform:components/component/state/serial-no"
	COMP_STATE_TYPE              = "/openconfig-platform:components/component/state/type"
	COMP_STATE_PARENT            = "/openconfig-platform:components/component/state/parent"
	COMP_STATE_TEMP_CTR          = "/openconfig-platform:components/component/state/temperature"
	COMP_STATE_TEMP              = "/openconfig-platform:components/component/state/temperature/instant"

	/** Supported Integrated Circuit URIs **/
	COMP_IC     = "/openconfig-platform:components/component/integrated-circuit"
	COMP_IC_CFG = "/openconfig-platform:components/component/integrated-circuit/config"
	COMP_IC_ST  = "/openconfig-platform:components/component/integrated-circuit/state"

	/** Supported Xcvr URIs **/
	XCVR_BASE_PREFIX  = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver"
	XCVR_BASE_STATE   = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state"
	XCVR_FORM_FACTOR  = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/form-factor"
	COMP_STATE_SW_VER = "/openconfig-platform:components/component/state/software-version"

	/** Supported Software Module URIs **/
	COMP_SW_MOD                 = "/openconfig-platform:components/component/software-module"
	COMP_SW_MOD_ST              = "/openconfig-platform:components/component/software-module/state"
	SW_MODULE_STATE_MODULE_TYPE = "/openconfig-platform:components/component/software-module/state/openconfig-platform-software:module-type"
	SW_BOOT_LOADER_STATE_TYPE   = "/openconfig-platform:components/component/boot-loader/state/openconfig-platform-boot-loader:type"
)

type componentType int64

const (
	CompTypeInvalid componentType = iota
	CompTypeXcvr
	CompTypeIC
	CompTypeSysEeprom
	CompTypeNWStack
	CompTypeOS
	CompTypeBootLoader
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

/* ICInfo structure read from State DB */
type ICInfo struct {
	NodeID string
}

type XcvrLane struct {
	RxPowerLane string
	TxBiasLane  string
	TxPowerLane string
	TxDisable   string
}

type XcvrInfo struct {
	/* Most are strings since media sends 'N/A' when data is not available
	   Conversion will be done before sending along */
	Presence    bool
	Lanes       [XCVR_LANE_LIMIT]XcvrLane
	Temperature string
	Parent      string
	MfgName     string
	MfgDate     string
	PartNo      string
	SerialNo    string
	HardwareRev string
	Type        string
}

/*SWCompInfo structure read from State DB*/
type SWCompInfo struct {
	Name            string
	SoftwareVersion string
	Parent          string
	OperStatus      string
	Type            string
	BootLoaderType  string
}

var dbToYangBootLoaderTypeMap = map[string]ocbinds.E_OpenconfigPlatformBootLoader_BOOT_LOADER_BASE{
	"GRUB":         ocbinds.OpenconfigPlatformBootLoader_BOOT_LOADER_BASE_BOOT_LOADER_GRUB,
	"ONIE":         ocbinds.OpenconfigPlatformBootLoader_BOOT_LOADER_BASE_BOOT_LOADER_ONIE,
	"UBOOT":        ocbinds.OpenconfigPlatformBootLoader_BOOT_LOADER_BASE_BOOT_LOADER_UBOOT,
	"SYSTEMD_BOOT": ocbinds.OpenconfigPlatformBootLoader_BOOT_LOADER_BASE_BOOT_LOADER_SYSTEMD_BOOT,
	"LINUXBOOT":    ocbinds.OpenconfigPlatformBootLoader_BOOT_LOADER_BASE_BOOT_LOADER_LINUXBOOT,
}

func operStatusFromString(status string) (ocbinds.E_OpenconfigPlatformTypes_COMPONENT_OPER_STATUS, error) {
	switch strings.ToLower(status) {
	case "active":
		return ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_ACTIVE, nil
	case "inactive":
		return ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_INACTIVE, nil
	case "disabled":
		return ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_DISABLED, nil
	default:
		return ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_DISABLED, fmt.Errorf("unknown oper-status: %s", status)
	}
}

type PathType int

const (
	/* Represents all paths under /components/component */
	AllPaths PathType = iota
	/* Represents all paths under /components/component/config */
	ConfigPaths
	/* Represents all paths under /components/component/state */
	StatePaths
	/* Represents all paths under a component subtree, e.g.
	 * /components/component/port or /components/component/fan */
	AllCompPaths
	/* Represents a path to a specific leaf */
	SinglePath
)

func (pt PathType) String() string {
	switch pt {
	case AllPaths:
		return "AllPaths"
	case ConfigPaths:
		return "ConfigPaths"
	case AllCompPaths:
		return "AllComponentPaths"
	case StatePaths:
		return "StatePaths"
	case SinglePath:
		return "SinglePath"
	}
	return strconv.Itoa(int(pt))
}

func (ct componentType) String() string {
	switch ct {
	case CompTypeInvalid:
		return "CompTypeInvalid"
	case CompTypeXcvr:
		return "CompTypeXcvr"
	case CompTypeIC:
		return "CompTypeIC"
	case CompTypeSysEeprom:
		return "CompTypeSysEeprom"
	case CompTypeNWStack:
		return "CompTypeNWStack"
	case CompTypeOS:
		return "CompTypeOS"
	case CompTypeBootLoader:
		return "CompTypeBootLoader"
	}
	return strconv.Itoa(int(ct))
}

var compTblMap = map[componentType][]string{
	CompTypeXcvr:       {TRANSCEIVER_STATUS, XCVR_KEY_PREFIX + "*"},
	CompTypeIC:         {NODE_CFG_TBL, "*"},
	CompTypeSysEeprom:  {EEPROM_INFO_TBL, "*"},
	CompTypeNWStack:    {SW_COMP_TBL, "*"},
	CompTypeOS:         {SW_COMP_TBL, "*"},
	CompTypeBootLoader: {SW_COMP_TBL, "*"},
}

func init() {
	XlateFuncBind("DbToYangPath_pfm_components_path_xfmr", DbToYangPath_pfm_components_path_xfmr)
	XlateFuncBind("Subscribe_pfm_components_xfmr", Subscribe_pfm_components_xfmr)
	XlateFuncBind("DbToYang_pfm_components_xfmr", DbToYang_pfm_components_xfmr)
	XlateFuncBind("YangToDb_pfm_components_xfmr", YangToDb_pfm_components_xfmr)
}

var compTypeCache sync.Map

func validICName(name string) bool {
	suffix := strings.TrimPrefix(name, IC_NAME_PREFIX)
	if suffix == name {
		return false
	}
	_, err := strconv.Atoi(suffix)
	return err == nil
}

func validXcvrName(name *string) bool {
	if name == nil || *name == "" {
		return false
	}

	/* Expect transceiver name of form EthernetX, where X is an integer */
	if !strings.HasPrefix(*name, XCVR_KEY_PREFIX) {
		return false
	}

	sp := strings.SplitAfter(*name, XCVR_KEY_PREFIX)
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
	case validXcvrName(&compName):
		return CompTypeXcvr, nil
	case validICName(compName):
		return CompTypeIC, nil
	case validSysEepromName(compName):
		return CompTypeSysEeprom, nil
	case validSWCompName(&compName, NW_STACK_PREFIX):
		return CompTypeNWStack, nil
	case validSWCompName(&compName, OS_PREFIX):
		return CompTypeOS, nil
	case strings.HasPrefix(compName, BOOTL_PREFIX):
		return CompTypeBootLoader, nil
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

func getCompType(name string, stdb, cfgdb *db.DB) componentType {
	if name == "*" {
		log.V(3).Infof("Invalid comp type for name as *")
		return CompTypeInvalid
	}
	if val, ok := compTypeCache.Load(name); ok {
		return val.(componentType)
	}
	if stdb != nil {
		if swcEntry, err := stdb.GetEntry(&db.TableSpec{Name: SW_COMP_TBL}, db.Key{Comp: []string{name}}); err == nil {
			switch swcEntry.Get("type") {
			case BOOTL_TYPE:
				compTypeCache.Store(name, CompTypeBootLoader)
				return CompTypeBootLoader
			case NW_STACK_TYPE:
				// Need to check if actually NW_STACK_TYPE or older boot loader
				if swcEntry.Get("module-type") == BOOTL_TYPE {
					compTypeCache.Store(name, CompTypeBootLoader)
					return CompTypeBootLoader
				}
				compTypeCache.Store(name, CompTypeNWStack)
				return CompTypeNWStack
			case OS_TYPE:
				compTypeCache.Store(name, CompTypeOS)
				return CompTypeOS
			}
		}
	}
	/* SysEEPROM and Transceiver - StateDB */
	if stdb != nil {
		if keyInDbTable(EEPROM_INFO_TBL, name, stdb) {
			compTypeCache.Store(name, CompTypeSysEeprom)
			return CompTypeSysEeprom
		}
		if keyInDbTable(TRANSCEIVER_STATUS, name, stdb) {
			compTypeCache.Store(name, CompTypeXcvr)
			return CompTypeXcvr
		}
	}
	/* IC - ConfigDB */
	if cfgdb != nil {
		if keyInDbTable(NODE_CFG_TBL, name, cfgdb) {
			compTypeCache.Store(name, CompTypeIC)
			return CompTypeIC
		}
	}
	compType, err := getCompTypeByName(name)
	if err == nil {
		compTypeCache.Store(name, compType)
		return compType
	}
	return CompTypeInvalid
}

var DbToYangPath_pfm_components_path_xfmr PathXfmrDbToYangFunc = func(inParams XfmrDbToYgPathParams) error {
	rootPath := COMP

	log.V(3).Infof("DbToYangPath_pfm_path_xfmr: inParams: %#v", inParams)

	if len(inParams.tblKeyComp) == 0 {
		return fmt.Errorf("Invalid tblKeyComp for pfm path xmfr:%v", inParams.tblKeyComp)
	}

	tblKey := inParams.tblKeyComp[0]
	switch inParams.tblName {
	default:
		inParams.ygPathKeys[rootPath+"/name"] = tblKey
	}

	log.V(3).Info("DbToYangPath_pfm_path_xfmr:- params.ygPathKeys: ", inParams.ygPathKeys)
	return nil
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

/* Given a URI for a subscription, return a list of component types which apply
 * to it.  For example a URI of "/components/component/port" would return
 * [CompTypePort] while a URI of "/components/component/state/software-version"
 * would return a list of all component types which report software version. */

func compTypesForSubscriptionUri(uri string) []componentType {
	cTypes := []componentType{}
	if strings.HasPrefix(uri, "/openconfig-platform:components/component/integrated-circuit") {
		cTypes = []componentType{CompTypeIC}
	} else if strings.HasPrefix(uri, "/openconfig-platform:components/component/oc-transceiver:transceiver") ||
		strings.HasPrefix(uri, "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver") {
		cTypes = []componentType{CompTypeXcvr}
	} else if strings.HasPrefix(uri, "/openconfig-platform:components/component/software-module") {
		cTypes = []componentType{CompTypeNWStack, CompTypeOS}
	} else if strings.HasPrefix(uri, "/openconfig-platform:components/component/state/oper-status") {
		cTypes = []componentType{CompTypeNWStack, CompTypeOS}
	} else if strings.HasPrefix(uri, "/openconfig-platform:components/component/state/boot-loader") {
		cTypes = []componentType{CompTypeBootLoader}
	} else if strings.HasPrefix(uri, COMP_STATE_PARENT) {
		cTypes = []componentType{CompTypeNWStack, CompTypeOS, CompTypeBootLoader}
	} else if strings.HasPrefix(uri, "/openconfig-platform:components/component/state/software-version") {
		cTypes = []componentType{CompTypeNWStack, CompTypeOS, CompTypeBootLoader}
	}
	return cTypes
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
	if key == "*" {
		result.isVirtualTbl = true
		return result, nil
	}
	dbNum := db.StateDB
	d := inParams.dbs[dbNum]
	if d == nil {
		dbNum = db.ConfigDB
		d = inParams.dbs[dbNum]
	}
	if d == nil {
		return result, fmt.Errorf("translateExists: No usable DB client in inParams (checked %v and %v)", db.StateDB.Name(), db.ConfigDB.Name())
	}
	compType := getCompType(key, inParams.dbs[db.StateDB], inParams.dbs[db.ConfigDB])
	if compType == CompTypeInvalid {
		return result, nil
	}
	tblInfo, ok := compTblMap[compType]
	if !ok {
		return result, errors.New("table not found.")
	}
	if len(tblInfo) == 0 {
		return result, errors.New("table is empty.")
	}
	tblName := tblInfo[0]
	tblKey := key
	switch compType {
	case CompTypeIC:
		dbNum = db.ConfigDB
	case CompTypeSysEeprom, CompTypeXcvr:
		dbNum = db.StateDB
	}
	result.dbDataMap = RedisDbSubscribeMap{dbNum: {tblName: {tblKey: {}}}}
	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr result: %v %v %v +++", dbNum, tblName, tblKey)
	return result, nil
}

/* Helper for the main subscribe transformer handling the TRANSLATE_SUBSCRIBE case. */
func translateSubscribe(inParams XfmrSubscInParams, key, targetUriPath string) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	/* Handle TRANSLATE_SUBSCRIBE by expanding the wildcard yang key to a set of
	 * DB tables and keys. If the key is not a wildcard then identify the set
	 * of DB tables and keys which apply to it. */
	result.isVirtualTbl = false
	result.needCache = true
	result.onChange = OnchangeEnable
	result.nOpts = &notificationOpts{mInterval: 0, pType: OnChange}

	/* Use the requested path to create a positive filter of component types to process. */
	compTypeFilter := []componentType{}
	cType := getCompType(key, inParams.dbs[db.StateDB], inParams.dbs[db.ConfigDB])
	if key == "*" {
		compTypeFilter = compTypesForSubscriptionUri(targetUriPath)
	} else {
		compTypeFilter = []componentType{cType}
		if cType == CompTypeInvalid {
			return result, nil
		}

	}
	for cType, tblNames := range compTblMap {
		if len(tblNames) < 2 {
			continue
		}
		/* Filtering is required, skip all component types not present in the filter. */
		if len(compTypeFilter) > 0 {
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
		if key != "*" {
			tblKey = key
		}

		if cType == CompTypeIC {
			tblDb = db.ConfigDB
		}

		if result.dbDataMap[tblDb] == nil {
			result.dbDataMap[tblDb] = make(map[string]map[string]map[string]string)
		}
		if result.dbDataMap[tblDb][tblName] == nil {
			result.dbDataMap[tblDb][tblName] = make(map[string]map[string]string)
		}

		/* Add the DB table and key to the result. */
		if result.dbDataMap[tblDb][tblName][tblKey] == nil {
			result.dbDataMap[tblDb][tblName][tblKey] = map[string]string{}
		}
	}
	if log.V(3) {
		for dbIdx, _ := range result.dbDataMap {
			for tbl, _ := range result.dbDataMap[dbIdx] {
				for k, v := range result.dbDataMap[dbIdx][tbl] {
					log.Infof("+++ Subscribe_pfm_components_xfmr result: DB=%d, Table=%s, Key=%s, Flds=%v +++", dbIdx, tbl, k, v)
				}
			}
		}
	}
	return result, nil
}

var YangToDb_pfm_components_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	pathInfo := NewPathInfo(inParams.uri)
	key := pathInfo.Var("name")
	if key == "" {
		return nil, nil
	}

	log.V(3).Infof("YangToDb_pfm_components_xfmr: uri %s, name %s, requestURI %s, op %v", inParams.uri, key, inParams.requestUri, inParams.oper)
	pfmObj := getPfmRootObject(inParams.ygRoot)
	if pfmObj == nil || pfmObj.Component == nil || len(pfmObj.Component) < 1 {
		return nil, tlerr.NotSupported("YangToDb_pfm_components_xfmr: Empty component.")
	}

	comp, ok := pfmObj.Component[key]
	if !ok || comp == nil {
		return nil, fmt.Errorf("YangToDb_pfm_components_xfmr: Invalid component name: %s", key)
	}

	inParams.key = key
	var tblName string
	cType := CompTypeInvalid
	if validICName(key) {
		tblName = NODE_CFG_TBL
		cType = CompTypeIC
	} else {
		return nil, fmt.Errorf("YangToDb_pfm_components_xfmr: Unable to identify component type for key: %s", key)
	}
	inParams.table = tblName

	memMap := make(map[string]map[string]db.Value)
	if inParams.oper == DELETE {
		switch cType {
		case CompTypeIC:
			/* We only support deletion on the following path:
			 * /components/component/integrated-circuit/config/node-id */
			memMap[NODE_CFG_TBL] = map[string]db.Value{key: db.Value{Field: map[string]string{"node-id": ""}}}
		}
	} else {
		if comp.Config != nil {
			fields := db.Value{Field: make(map[string]string)}
			if comp.Config.Name != nil {
				if inParams.key != *comp.Config.Name {
					return nil, fmt.Errorf("Mismatch between component name key: (%s) and name to be configured: (%s)", inParams.key, *comp.Config.Name)
				}
				fields.Set("name", *comp.Config.Name)
			}
			memMap[tblName] = map[string]db.Value{key: fields}
		}
		if comp.IntegratedCircuit != nil && comp.IntegratedCircuit.Config != nil && comp.IntegratedCircuit.Config.NodeId != nil {
			if cType != CompTypeIC {
				return nil, fmt.Errorf("Component name \"%s\" not identified as an Integrated Circuit but contains an integrated-circuit subtree..", key)
			}
			dbVal := db.Value{Field: make(map[string]string)}
			if _, ok := memMap[NODE_CFG_TBL]; !ok {
				memMap[NODE_CFG_TBL] = make(map[string]db.Value)
			}
			if _, ok := memMap[NODE_CFG_TBL][key]; !ok {
				memMap[NODE_CFG_TBL][key] = dbVal
			} else {
				dbVal = memMap[NODE_CFG_TBL][key]
			}
			nodeID := *comp.IntegratedCircuit.Config.NodeId
			dbVal.Set("node-id", strconv.FormatUint(nodeID, 10))
		}
	}

	log.V(3).Infof("YangToDb_pfm_components_xfmr: result %v", memMap)
	return memMap, nil
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

func getICInfoFromDb(name string, d *db.DB, tblName string) ICInfo {
	var nodeInfo ICInfo

	// Only proceed if the table is the one we are interested in
	if d == nil || tblName != NODE_CFG_TBL {
		return nodeInfo
	}

	nodeEntry, err := d.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{name}})
	if err != nil {
		log.V(3).Infof("Cant get entry: %s; Error: %v", name, err)
		return nodeInfo
	}

	nodeInfo.NodeID = nodeEntry.Get("node-id")

	return nodeInfo
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
	var all, compSt, icCfg, icSt bool
	if targetUriPath == COMP {
		all = true
	} else if strings.HasPrefix(targetUriPath, COMP_ST) {
		compSt = true
	} else if targetUriPath == COMP_IC {
		icCfg = true
		icSt = true
	} else if strings.HasPrefix(targetUriPath, COMP_IC_CFG) {
		icCfg = true
	} else if strings.HasPrefix(targetUriPath, COMP_IC_ST) {
		icSt = true
	}
	log.V(3).Infof("dbToYangIC: name %s targetUriPath %s", name, targetUriPath)

	// Build the integrated-circuit container and sub-containers when explicitly requested or during a full get
	if all || icCfg || icSt {
		ygot.BuildEmptyTree(comp.IntegratedCircuit)
		if all || icCfg {
			ygot.BuildEmptyTree(comp.IntegratedCircuit.Config)
		}
		if all || icSt {
			ygot.BuildEmptyTree(comp.IntegratedCircuit.State)
		}
	}
	cfgDb := dbs[db.ConfigDB]
	nodeCfg := getICInfoFromDb(name, cfgDb, NODE_CFG_TBL)

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

	/* Handle component integrated-circuit config and state paths */
	if nodeCfg.NodeID != "" {
		nodeID, err := strconv.ParseUint(nodeCfg.NodeID, 10, 64)
		if err != nil {
			log.Warningf("string conversion failed for IC %s node-id \"%s\": %v", name, nodeCfg.NodeID, err)
		} else {
			/* Handle component integrated-circuit config paths */
			if all || icCfg {
				comp.IntegratedCircuit.Config.NodeId = &nodeID
			}

			/* Handle component integrated-circuit state paths */
			if all || icSt {
				comp.IntegratedCircuit.State.NodeId = &nodeID
			}
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
			if eeprom.HardwareVersion == nil {
				eeprom.HardwareVersion = &eepromDb.Hardware_Version
			}
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

func test_if_available(s string) bool {
	return ((s != "") && (s != "N/A") && (s != "n/a"))
}

func fillXcvrLaneInfo(xcvrCom *ocbinds.OpenconfigPlatform_Components_Component, laneIdx uint16, xcvrInfo XcvrInfo, name string, maxLanes int) (err error) {
	channel, ok := xcvrCom.Transceiver.PhysicalChannels.Channel[laneIdx]
	if !ok || channel == nil {
		channel, err = xcvrCom.Transceiver.PhysicalChannels.NewChannel(laneIdx)
		if err != nil {
			return fmt.Errorf("cannot create channel object: %w", err)
		}
	}
	ygot.BuildEmptyTree(channel)
	ygot.BuildEmptyTree(channel.Config)
	ygot.BuildEmptyTree(channel.State)
	channel.Config.Index = &laneIdx
	channel.State.Index = &laneIdx

	if laneIdx < XCVR_LANE_LIMIT {
		lane := &xcvrInfo.Lanes[laneIdx]
		convAndFillDBValues(lane.RxPowerLane, lane.TxPowerLane, lane.TxBiasLane, lane.TxDisable, channel)
	} else {
		return errors.New("lane index is invalid.")
	}
	return nil
}

func fillXcvrInfo(xcvrCom *ocbinds.OpenconfigPlatform_Components_Component,
	name string, all bool, laneIdx string, targetUriPath string, dbs [db.MaxDB]*db.DB) error {
	var err error

	ygot.BuildEmptyTree(xcvrCom)
	ygot.BuildEmptyTree(xcvrCom.Transceiver)
	ygot.BuildEmptyTree(xcvrCom.Transceiver.State)

	log.V(3).Infof("fillXcvrInfo: name %s, all %v laneIdx %s targetUriPath %s", name, all, laneIdx, targetUriPath)

	d := dbs[db.StateDB]
	if d == nil {
		d, err = db.NewDB(getDBOptions(db.StateDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer d.DeleteDB()
	}
	cfgdb := dbs[db.ConfigDB]
	if cfgdb == nil {
		cfgdb, err = db.NewDB(getDBOptions(db.ConfigDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer cfgdb.DeleteDB()
	}

	xcvrStatusState, err := getXcvrStatusInfoFromDb(name, d)

	nm := name
	xcvrEEPROMState := xcvrCom.State
	xcvrEEPROMState.Name = &nm
	var xcvrInfo XcvrInfo
	if !xcvrStatusState.Presence {
		p := !xcvrStatusState.Presence
		xcvrEEPROMState.Empty = &p
	} else {
		xcvrInfo, err = getXcvrInfoFromDb(name, d)
		if err != nil {
			log.V(3).Info("Error Getting transceiver info from dB")
			return err
		}
	}

	if xcvrInfo.Type != "" && laneIdx != "" {
		maxLanes, ok := sfpTypeToMaxLanesMap[xcvrInfo.Type]
		if !ok {
			return errors.New("could not find the max number of lanes for transceiver.")
		}
		idx, err := strconv.ParseUint(laneIdx, 10, 16)
		if err != nil {
			return err
		}
		if idx >= uint64(maxLanes) {
			return errors.New("lane index greater than the max number of lanes for transceiver.")
		}
		if err := fillXcvrLaneInfo(xcvrCom, uint16(idx), xcvrInfo, name, maxLanes); err != nil {
			return err
		}
	}

	xcvrState := xcvrCom.Transceiver.State

	if all {
		/* Top level */
		xcvrEEPROMState.Name = &nm
		xcvrCom.Config.Name = &nm

		/* Present state */
		p := !xcvrInfo.Presence
		xcvrEEPROMState.Empty = &p

		q := true
		xcvrEEPROMState.Removable = &q

		xcvrEEPROMState.Type, _ = xcvrCom.State.To_OpenconfigPlatform_Components_Component_State_Type_Union(
			ocbinds.OpenconfigPlatformTypes_OPENCONFIG_HARDWARE_COMPONENT_TRANSCEIVER)

		if test_if_available(xcvrInfo.Parent) {
			xcvrEEPROMState.Parent = &xcvrInfo.Parent
		}

		/* Vendor info */
		if xcvrInfo.SerialNo != "" {
			xcvrEEPROMState.SerialNo = &xcvrInfo.SerialNo
		}
		if xcvrInfo.PartNo != "" {
			xcvrEEPROMState.PartNo = &xcvrInfo.PartNo
		}
		if xcvrInfo.MfgName != "" {
			xcvrEEPROMState.MfgName = &xcvrInfo.MfgName
		}
		if xcvrInfo.HardwareRev != "" {
			xcvrEEPROMState.HardwareVersion = &xcvrInfo.HardwareRev
			// Using the 'hardware_rev' field to also populate the firmware-version path.
			xcvrEEPROMState.FirmwareVersion = &xcvrInfo.HardwareRev
		}
		if xcvrInfo.MfgDate != "" {
			xcvrEEPROMState.MfgDate = &xcvrInfo.MfgDate
		}
		/* Not present */
		if xcvrInfo.Presence {
			xcvrEEPROMState.OperStatus = ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_ACTIVE
		}

		if xcvrInfo.Temperature != "" {
			if float64val, err := strconv.ParseFloat(xcvrInfo.Temperature, 64); err == nil {
				xcvrEEPROMState.Temperature.Instant = &float64val
			}
		}

		/* Physical-Channels level */
		if xcvrInfo.Type != "" && laneIdx == "" {
			if maxLanes, ok := sfpTypeToMaxLanesMap[xcvrInfo.Type]; ok {
				for i := 0; i < maxLanes; i++ {
					if err := fillXcvrLaneInfo(xcvrCom, uint16(i), xcvrInfo, name, maxLanes); err != nil {
						return err
					}
				}
			} else {
				log.V(3).Info("Could not find the max number of lanes for transceiver.")
			}
		}
		return err
	}

	switch targetUriPath {
	case COMP_STATE_EMPTY:
		q := !xcvrInfo.Presence
		xcvrEEPROMState.Empty = &q
	case COMP_STATE_NAME:
		nm := name
		xcvrEEPROMState.Name = &nm
	case COMP_STATE_TYPE:
		xcvrEEPROMState.Type, _ = xcvrCom.State.To_OpenconfigPlatform_Components_Component_State_Type_Union(
			ocbinds.OpenconfigPlatformTypes_OPENCONFIG_HARDWARE_COMPONENT_TRANSCEIVER)
	case COMP_STATE_PARENT:
		if test_if_available(xcvrInfo.Parent) {
			xcvrEEPROMState.Parent = &xcvrInfo.Parent
		}
	case COMP_STATE_SERIAL_NO:
		if xcvrInfo.SerialNo != "" {
			xcvrEEPROMState.SerialNo = &xcvrInfo.SerialNo
		}
	case COMP_STATE_PART_NO:
		if xcvrInfo.PartNo != "" {
			xcvrEEPROMState.PartNo = &xcvrInfo.PartNo
		}
	case COMP_STATE_MFG_NAME:
		if xcvrInfo.MfgName != "" {
			xcvrEEPROMState.MfgName = &xcvrInfo.MfgName
		}
	case COMP_STATE_HW_VER:
		if xcvrInfo.HardwareRev != "" {
			xcvrEEPROMState.HardwareVersion = &xcvrInfo.HardwareRev
		}
	// Using the 'hardware_rev' field to also populate the firmware-version path.
	case COMP_STATE_FIRM_VER:
		if xcvrInfo.HardwareRev != "" {
			xcvrEEPROMState.FirmwareVersion = &xcvrInfo.HardwareRev
		}
	case COMP_STATE_MFG_DATE:
		if xcvrInfo.MfgDate != "" {
			xcvrEEPROMState.MfgDate = &xcvrInfo.MfgDate
		}
	case COMP_STATE_OPER_STATUS:
		xcvrEEPROMState.OperStatus = ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_ACTIVE
	case COMP_STATE_TEMP:
		if xcvrInfo.Temperature != "" {
			float64val, err := strconv.ParseFloat(xcvrInfo.Temperature, 64)
			if err != nil {
				return err
			}
			xcvrEEPROMState.Temperature.Instant = &float64val
		}
	case COMP_STATE_REMOVABLE:
		q := true
		xcvrEEPROMState.Removable = &q
	case XCVR_FORM_FACTOR:
		if xcvrInfo.Type != "" {
			xcvrState.FormFactor = formFactorTypeFromString(xcvrInfo.Type)
		}
	}
	return err
}

func formFactorTypeFromString(ft string) ocbinds.E_OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE {
	switch {
	case ft == "N/A" || ft == "":
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_UNSET
	case strings.HasPrefix(ft, "Unknown"):
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_OTHER
	case strings.HasPrefix(ft, "SFP"):
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_SFP
	case ft == "QSFP":
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_QSFP
	case strings.HasPrefix(ft, "QSFP+"):
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_QSFP_PLUS
	case strings.HasPrefix(ft, "QSFP28"):
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_QSFP28
	case strings.HasPrefix(ft, "OSFP") || strings.HasPrefix(ft, "QSFP-DD"):
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_OSFP
	default:
		return ocbinds.OpenconfigTransportTypes_TRANSCEIVER_FORM_FACTOR_TYPE_UNSET
	}
}

func getXcvrStatusInfoFromDb(name string, d *db.DB) (XcvrInfo, error) {
	xcvrStatusEntry, err := d.GetEntry(&db.TableSpec{Name: TRANSCEIVER_STATUS}, db.Key{Comp: []string{name}})
	if err != nil {
		return XcvrInfo{}, err
	}

	var xcvrStatus XcvrInfo

	status := xcvrStatusEntry.Get("status")
	switch status {
	case SFP_STATUS_INSERTED:
		xcvrStatus.Presence = true
	case SFP_STATUS_REMOVED, "":
		xcvrStatus.Presence = false
	default:
		return XcvrInfo{}, fmt.Errorf("Unknown status for transceiver %s: %s", name, status)
	}
	return xcvrStatus, nil
}

func getXcvrInfoFromDb(name string, d *db.DB) (XcvrInfo, error) {
	var xcvrInfo XcvrInfo
	var err error

	xcvrEntry, err := d.GetEntry(&db.TableSpec{Name: TRANSCEIVER_TBL}, db.Key{Comp: []string{name}})

	if err != nil {
		xcvrInfo.Presence = false
		return xcvrInfo, err
	}

	/* Existence of entry implies presence */
	xcvrInfo.Presence = true
	xcvrInfo.Parent = xcvrEntry.Get("parent")
	xcvrInfo.MfgName = xcvrEntry.Get("manufacturer")
	xcvrInfo.MfgDate = xcvrEntry.Get("vendor_date")
	xcvrInfo.PartNo = xcvrEntry.Get("model")
	xcvrInfo.SerialNo = xcvrEntry.Get("serial")
	xcvrInfo.HardwareRev = xcvrEntry.Get("hardware_rev")
	xcvrInfo.Type = xcvrEntry.Get("type")

	xcvrDOMEntry, err := d.GetEntry(&db.TableSpec{Name: TRANSCEIVER_DOM}, db.Key{Comp: []string{name}})
	if err != nil {
		log.V(3).Infof("DOM sensor info not available for transceiver %s: %v", name, err)
	} else {
		for i := 0; i < XCVR_LANE_LIMIT; i++ {
			xcvrInfo.Lanes[i].RxPowerLane = xcvrDOMEntry.Get(fmt.Sprintf("rx%dpower", i+1))
			xcvrInfo.Lanes[i].TxBiasLane = xcvrDOMEntry.Get(fmt.Sprintf("tx%dbias", i+1))
			xcvrInfo.Lanes[i].TxPowerLane = xcvrDOMEntry.Get(fmt.Sprintf("tx%dpower", i+1))
			xcvrInfo.Lanes[i].TxDisable = xcvrDOMEntry.Get(fmt.Sprintf("tx%ddisable", i+1))
		}
		xcvrInfo.Temperature = xcvrDOMEntry.Get("temperature")
	}

	return xcvrInfo, nil
}

func convAndFillDBValues(rxpField, txpField, txbField, txdisableField string, channel *ocbinds.OpenconfigPlatform_Components_Component_Transceiver_PhysicalChannels_Channel) {
	ygot.BuildEmptyTree(channel)
	ygot.BuildEmptyTree(channel.State)
	ygot.BuildEmptyTree(channel.State.InputPower)
	ygot.BuildEmptyTree(channel.State.OutputPower)
	ygot.BuildEmptyTree(channel.State.LaserBiasCurrent)

	if rxpField != "" {
		if rxPower, err := strconv.ParseFloat(rxpField, 64); err == nil {
			channel.State.InputPower.Instant = &rxPower
		} else {
			log.V(3).Infof("Error converting rxPower (\"%s\") from string to float64", rxpField)
		}
	}
	if txpField != "" {
		if txPower, err := strconv.ParseFloat(txpField, 64); err == nil {
			channel.State.OutputPower.Instant = &txPower
		} else {
			log.V(3).Infof("Error converting txPower (\"%s\") from string to float64", txpField)
		}
	}
	if txbField != "" {
		if txBias, err := strconv.ParseFloat(txbField, 64); err == nil {
			channel.State.LaserBiasCurrent.Instant = &txBias
		} else {
			log.V(3).Infof("Error converting txBias (\"%s\") from string to float64", txbField)
		}
	}
	txLaserEnable := false
	if txdisableField == "False" || txdisableField == "false" {
		txLaserEnable = true
	}
	channel.State.TxLaser = &txLaserEnable
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
	d := dbs[db.StateDB]
	ygot.BuildEmptyTree(pfComp)
	switch cType {
	case CompTypeXcvr:
		return fillXcvrInfo(pfComp, compName, pType != SinglePath, "", targetUriPath, dbs)
	case CompTypeIC:
		return fillICInfo(pfComp, compName, targetUriPath, dbs, ygRoot)
	case CompTypeSysEeprom:
		return fillSysEepromInfo(pfComp, compName, targetUriPath, dbs, ygRoot)
	case CompTypeNWStack, CompTypeOS, CompTypeBootLoader:
		return fillSWCompInfo(pfComp, compName, pType, targetUriPath, d, cType)
	}
	return errors.New("Invalid component type")
}

func createCompAndFuncCall(pfCpts *ocbinds.OpenconfigPlatform_Components, targetUriPath string, compType componentType, inParams XfmrParams, tblName string, tblKey string) {
	var compNames []string
	var err error
	dbs := inParams.dbs
	stdb := dbs[db.StateDB]
	cfgdb := dbs[db.ConfigDB]

	var keySep string
	switch compType {
	case CompTypeIC:
		compNames, err = getAllTableEntries(cfgdb, tblName, tblKey)
		if cfgdb != nil {
			keySep = cfgdb.Opts.KeySeparator
		}
	case CompTypeSysEeprom:
		compNames = []string{SYS_EEPROM_NAME}
		if stdb != nil {
			keySep = stdb.Opts.KeySeparator
		}
	default:
		compNames, err = getAllTableEntries(stdb, tblName, tblKey)
		if stdb != nil {
			keySep = stdb.Opts.KeySeparator
		}
	}
	if err != nil {
		log.V(3).Info(err)
	}

	if keySep == "" {
		keySep = "|"
	}

	for _, compAndKey := range compNames {
		compKeys := strings.Split(compAndKey, keySep)
		if len(compKeys) == 0 {
			continue
		}
		comp := compKeys[0]
		derivedCompType := getCompType(comp, stdb, cfgdb)
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

/* Main workhorse of the DbToYang transformer. The get is either for the entire
 * component list (/components/component) or for a specific component name; no
 * wildcards are handled here. */
func getSysComponents(pf_cpts *ocbinds.OpenconfigPlatform_Components, targetUriPath string, inParams XfmrParams, compName, subKey string) error {

	if pf_cpts == nil {
		return nil
	}

	log.V(3).Infof("Preparing dB for system components")

	uri := inParams.uri
	dbs := inParams.dbs
	ygRoot := inParams.ygRoot

	var err error
	stdb := dbs[db.StateDB]
	cfgdb := dbs[db.ConfigDB]
	log.V(3).Infof("getSysComponents: compName: %s targetUriPath: %s", compName, targetUriPath)
	switch targetUriPath {
	case COMP:
		log.V(3).Infof("compName: %v", compName)
		subCompName := "" /* Get all subcomponents */
		if compName == "" {
			/* Handle all component types */
			for cType, tbl := range compTblMap {
				tblName := tbl[0]
				createCompAndFuncCall(pf_cpts, targetUriPath, cType, inParams, tblName, tbl[1])
			}
		} else {
			compType := getCompType(compName, stdb, cfgdb)
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
			/* The third argument in compTypeToFuncCall() is passed as empty as subcomp is not supported */
			if err = compTypeToFuncCall(compType, compName, subCompName, pf_comp, targetUriPath, dbs, AllPaths, ygRoot); err != nil {
				log.V(3).Info(err)
			}
		}
	case COMP_ST:
		compType := getCompType(compName, stdb, cfgdb)
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
	case COMP_CFG:
		compType := getCompType(compName, stdb, cfgdb)
		if compType == CompTypeInvalid {
			return nil
		}
		if compType == CompTypeIC {
			pf_comp, ok := pf_cpts.Component[compName]
			if !ok || pf_comp == nil {
				var errNew error
				pf_comp, errNew = pf_cpts.NewComponent(compName)
				if errNew != nil {
					return fmt.Errorf("Invalid component name: %s", compName)
				}
			}
			ygot.BuildEmptyTree(pf_comp)
			if err = compTypeToFuncCall(compType, compName, subKey, pf_comp, targetUriPath, dbs, ConfigPaths, ygRoot); err != nil {
				log.V(3).Info(err)
			}
			break
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
		compType := getCompType(compName, stdb, cfgdb)
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
		case CompTypeXcvr:
			ygot.BuildEmptyTree(pf_comp.Transceiver)
			ygot.BuildEmptyTree(pf_comp.Transceiver.State)
			ygot.BuildEmptyTree(pf_comp.Transceiver.Config)

			laneIdx := NewPathInfo(uri).Var("index")
			switch targetUriPath {
			case XCVR_BASE_PREFIX, XCVR_BASE_STATE:
				return fillXcvrInfo(pf_comp, compName, true, laneIdx, targetUriPath, dbs)
			default:
				/* For individual components */
				return fillXcvrInfo(pf_comp, compName, false, laneIdx, targetUriPath, dbs)
			}
		case CompTypeIC:
			return fillICInfo(pf_comp, compName, targetUriPath, inParams.dbs, inParams.ygRoot)
		case CompTypeSysEeprom:
			return fillSysEepromInfo(pf_comp, compName, targetUriPath, inParams.dbs, inParams.ygRoot)
		case CompTypeNWStack:
			fallthrough
		case CompTypeOS:
			fallthrough
		case CompTypeBootLoader:
			ygot.BuildEmptyTree(pf_comp.SoftwareModule)
			ygot.BuildEmptyTree(pf_comp.SoftwareModule.State)
			switch targetUriPath {
			case COMP_SW_MOD:
				fallthrough
			case COMP_SW_MOD_ST:
				return fillSWCompInfo(pf_comp, compName, AllCompPaths, targetUriPath, stdb, compType)
			default:
				return fillSWCompInfo(pf_comp, compName, SinglePath, targetUriPath, stdb, compType)
			}
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
	/* subkey is second level key set to null which is not used for now */
	subKey := ""

	return getSysComponents(getPfmRootObject(inParams.ygRoot), targetUriPath, inParams, compName, subKey)
}

func validSWCompName(name *string, prefix string) bool {
	if name == nil || *name == "" {
		return false
	}
	// Expect node name of form network_stackX or osX, where X is an integer (either 0 or 1)
	if !strings.HasPrefix(*name, prefix) {
		return false
	}

	/* Currently restricted to indices 0 or 1. Update this upper-bound
	check if future hardware configurations require support for more instances. */
	sp := strings.SplitAfter(*name, prefix)
	if len(sp) < 2 {
		return false
	}

	if val, err := strconv.Atoi(sp[1]); err != nil || val > 1 {
		return false
	}
	return true
}

func getSWCompInfoFromDb(name string, d *db.DB, tblName string) (SWCompInfo, error) {
	if d == nil {
		return SWCompInfo{}, errors.New("DB instance is nil")
	}

	swcEntry, err := d.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{name}})
	if err != nil {
		log.Info("Cannot get entry: ", name, "; Error: ", err)
		return SWCompInfo{}, err
	}

	swcInfo := SWCompInfo{
		Name:            swcEntry.Get("name"),
		SoftwareVersion: swcEntry.Get("software-version"),
		Parent:          swcEntry.Get("parent"),
		OperStatus:      swcEntry.Get("oper-status"),
		Type:            swcEntry.Get("type"),
		BootLoaderType:  swcEntry.Get("boot-loader-type"),
	}

	return swcInfo, nil
}

func fillBootLoaderContainer(info SWCompInfo, comp *ocbinds.OpenconfigPlatform_Components_Component) {
	if info.BootLoaderType == "" {
		return
	}
	ygot.BuildEmptyTree(comp.BootLoader)
	ygot.BuildEmptyTree(comp.BootLoader.State)
	if et, ok := dbToYangBootLoaderTypeMap[info.BootLoaderType]; ok {
		comp.BootLoader.State.Type = et
	}
}

/* Filling in the state info for software components available in Redis DB */
func fillSWCompInfo(comp *ocbinds.OpenconfigPlatform_Components_Component,
	name string, pType PathType, targetUriPath string, stdb *db.DB, cType componentType) error {
	swcInfo, err := getSWCompInfoFromDb(name, stdb, SW_COMP_TBL)
	if err != nil {
		log.V(3).Info("Error Getting SW Comp info from State DB: ", err.Error())
		return err
	}
	ygot.BuildEmptyTree(comp)
	compState := comp.State

	defaultVal := ""
	/*getChassis function is not used here . Parent name used as chassis*/
	defaultParentVal := CHASSIS_PREFIX

	if pType == AllPaths || pType == AllCompPaths || pType == StatePaths {
		// Filling in state values
		// State Name
		compState.Name = &name
		// State Software Version
		compState.SoftwareVersion = &defaultVal
		if swcInfo.SoftwareVersion != "" {
			compState.SoftwareVersion = &swcInfo.SoftwareVersion
		}
		// State Parent
		compState.Parent = &defaultParentVal
		if swcInfo.Parent != "" {
			compState.Parent = &swcInfo.Parent
		}
		// State Type
		switch cType {
		case CompTypeOS:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_OPERATING_SYSTEM)
		case CompTypeBootLoader:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_BOOT_LOADER)
			fillBootLoaderContainer(swcInfo, comp)
			return nil
		case CompTypeNWStack:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_SOFTWARE_MODULE)
		}
		// State Oper Status
		if operStatus, err := operStatusFromString(swcInfo.OperStatus); err == nil {
			compState.OperStatus = operStatus
		} else {
			compState.OperStatus = ocbinds.OpenconfigPlatformTypes_COMPONENT_OPER_STATUS_DISABLED
		}
		if pType == StatePaths {
			return nil
		}
		// SW Module State Module Type
		if cType == CompTypeNWStack {
			ygot.BuildEmptyTree(comp.SoftwareModule)
			ygot.BuildEmptyTree(comp.SoftwareModule.State)
			if comp.SoftwareModule != nil && comp.SoftwareModule.State != nil {
				comp.SoftwareModule.State.ModuleType = ocbinds.OpenconfigPlatformSoftware_SOFTWARE_MODULE_TYPE_USERSPACE_PACKAGE_BUNDLE
			}
		}
		return nil
	}

	switch targetUriPath {
	case COMP_STATE_NAME:
		compState.Name = &name
	case COMP_STATE_SW_VER:
		if swcInfo.SoftwareVersion == "" {
			return errors.New("software_version field not present in State DB")
		}
		compState.SoftwareVersion = &swcInfo.SoftwareVersion
	case COMP_STATE_PARENT:
		compState.Parent = &defaultParentVal
		if swcInfo.Parent != "" {
			compState.Parent = &swcInfo.Parent
		}
	case COMP_STATE_TYPE:
		switch cType {
		case CompTypeOS:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_OPERATING_SYSTEM)
		case CompTypeBootLoader:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_BOOT_LOADER)
		case CompTypeNWStack:
			compState.Type, _ = compState.To_OpenconfigPlatform_Components_Component_State_Type_Union(
				ocbinds.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_SOFTWARE_MODULE)
		default:
			return errors.New("invalid component type for software component")
		}
	case COMP_STATE_OPER_STATUS:
		if cType == CompTypeBootLoader {
			return errors.New("invalid path for this component type.")
		}
		if operStatus, err := operStatusFromString(swcInfo.OperStatus); err == nil {
			compState.OperStatus = operStatus
		} else {
			return errors.New("oper_status is missing/invalid field value in State DB: " + swcInfo.OperStatus)
		}
	case SW_MODULE_STATE_MODULE_TYPE:
		if cType != CompTypeNWStack {
			return errors.New("invalid component for software-module/state/module-type path.")
		}
		ygot.BuildEmptyTree(comp.SoftwareModule)
		ygot.BuildEmptyTree(comp.SoftwareModule.State)
		if comp.SoftwareModule != nil && comp.SoftwareModule.State != nil {
			comp.SoftwareModule.State.ModuleType = ocbinds.OpenconfigPlatformSoftware_SOFTWARE_MODULE_TYPE_USERSPACE_PACKAGE_BUNDLE
		}
	case SW_BOOT_LOADER_STATE_TYPE:
		fillBootLoaderContainer(swcInfo, comp)
	}
	return nil
}
