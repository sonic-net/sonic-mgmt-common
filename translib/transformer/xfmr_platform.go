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
	NODE_CFG_TBL  = "NODE_CFG"
	SW_COMP_TBL   = "SW_COMP_INFO"
	BOOTL_TYPE    = "BOOT_LOADER"
	OS_TYPE       = "OPERATING_SYSTEM"
	NW_STACK_TYPE = "SOFTWARE_MODULE"

	IC_NAME_PREFIX  = "integrated_circuit"
	CHASSIS_PREFIX  = "chassis"
	NW_STACK_PREFIX = "network_stack"
	OS_PREFIX       = "os"
	BOOTL_PREFIX    = "boot_loader"

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
	COMP_STATE_OPER_STATUS       = "/openconfig-platform:components/component/state/oper-status"
	COMP_STATE_PART_NO           = "/openconfig-platform:components/component/state/part-no"
	COMP_STATE_SERIAL_NO         = "/openconfig-platform:components/component/state/serial-no"
	COMP_STATE_TYPE              = "/openconfig-platform:components/component/state/type"
	COMP_STATE_PARENT            = "/openconfig-platform:components/component/state/parent"
	COMP_STATE_TEMP_CTR          = "/openconfig-platform:components/component/state/temperature"
	COMP_STATE_SW_VER            = "/openconfig-platform:components/component/state/software-version"

	/** Supported Software Module URIs **/
	COMP_SW_MOD                 = "/openconfig-platform:components/component/software-module"
	COMP_SW_MOD_ST              = "/openconfig-platform:components/component/software-module/state"
	SW_MODULE_STATE_MODULE_TYPE = "/openconfig-platform:components/component/software-module/state/openconfig-platform-software:module-type"
	SW_BOOT_LOADER_STATE_TYPE   = "/openconfig-platform:components/component/boot-loader/state/openconfig-platform-boot-loader:type"
)

type componentType int64

const (
	CompTypeInvalid componentType = iota
	CompTypeIC
	CompTypeNWStack
	CompTypeOS
	CompTypeBootLoader
)

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
	case AllCompPaths:
		return "AllComponentPaths"
	case StatePaths:
		return "StatePaths"
	case SinglePath:
		return "SinglePath"
	}
	return fmt.Sprintf("%s", pt)
}

func (ct componentType) String() string {
	switch ct {
	case CompTypeInvalid:
		return "CompTypeInvalid"
	case CompTypeIC:
		return "CompTypeIC"
	case CompTypeNWStack:
		return "CompTypeNWStack"
	case CompTypeOS:
		return "CompTypeOS"
	case CompTypeBootLoader:
		return "CompTypeBootLoader"
	}
	return fmt.Sprintf("%s", ct)
}

var compTblMap = map[componentType][]string{
	CompTypeIC:         {NODE_CFG_TBL, "*"},
	CompTypeNWStack:    {SW_COMP_TBL, "*"},
	CompTypeOS:         {SW_COMP_TBL, "*"},
	CompTypeBootLoader: {SW_COMP_TBL, "*"},
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

func getCompTypeByName(compName string) (componentType, error) {
	switch {
	case validICName(&compName):
		return CompTypeIC, nil
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
				// Need to check if actuall NW_STACK_TYPE or older boot loader
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
	compType, err := getCompTypeByName(name)
	if err == nil {
		compTypeCache.Store(name, compType)
		return compType
	}
	log.V(3).Infof("Invalid comp type")
	return CompTypeInvalid
}

var Subscribe_pfm_components_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	key := NewPathInfo(inParams.uri).Var("name")

	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr uri (%v) key(%s) mode(%v) +++", inParams.uri, key, inParams.subscProc)
	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr requestUri (%v) +++", inParams.requestURI)

	pathInfo := NewPathInfo(inParams.requestURI)
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return result, err
	}

	if key == "" || strings.Contains(key, "_sensor") {
		/* no need to verify dB data if we are requesting ALL
		   components or if request is for sensor */
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
	if strings.HasPrefix(uri, "/openconfig-platform:components/component/software-module") {
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
	cType := getCompType(key, inParams.dbs[db.StateDB], inParams.dbs[db.ConfigDB])
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
	case CompTypeIC:
		return fillICInfo(pfComp, compName, targetUriPath, dbs, ygRoot)
	case CompTypeNWStack, CompTypeOS, CompTypeBootLoader:
		return fillSWCompInfo(pfComp, compName, pType, targetUriPath, d, cType)
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
		derivedCompType := getCompType(comp, d, cfgdb)
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
	cfgdb := dbs[db.ConfigDB]
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
			compType := getCompType(compName, d, cfgdb)
			if compType == CompTypeInvalid {
				return nil
			}
			pf_comp, ok := pf_cpts.Component[compName]
			if !ok || pf_comp == nil {
				return fmt.Errorf("invalid input component name: %s", compName)
			}
			ygot.BuildEmptyTree(pf_comp)
			if err = compTypeToFuncCall(compType, compName, subCompName, pf_comp, targetUriPath, dbs, AllPaths, ygRoot); err != nil {
				log.V(3).Info(err)
			}
		}
	case COMP_ST:
		compType := getCompType(compName, d, cfgdb)
		if compType == CompTypeInvalid {
			return nil
		}
		pf_comp, ok := pf_cpts.Component[compName]
		if !ok || pf_comp == nil {
			return fmt.Errorf("invalid input component name for state path: %s", compName)
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
		compType := getCompType(compName, d, cfgdb)
		if compType == CompTypeInvalid {
			return nil
		}
		pf_comp, ok := pf_cpts.Component[compName]
		if !ok || pf_comp == nil {
			return fmt.Errorf("invalid input component name: %s", compName)
		}
		ygot.BuildEmptyTree(pf_comp)
		switch compType {
		case CompTypeIC:
			return fillICInfo(pf_comp, compName, targetUriPath, inParams.dbs, inParams.ygRoot)
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
				return fillSWCompInfo(pf_comp, compName, AllCompPaths, targetUriPath, d, compType)
			default:
				return fillSWCompInfo(pf_comp, compName, SinglePath, targetUriPath, d, compType)
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
	/*subkey is second level key set to null which is not used for now*/
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
	compState := comp.State
	swModuleState := comp.SoftwareModule.State
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
			swModuleState.ModuleType = ocbinds.OpenconfigPlatformSoftware_SOFTWARE_MODULE_TYPE_USERSPACE_PACKAGE_BUNDLE
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
		swModuleState.ModuleType = ocbinds.OpenconfigPlatformSoftware_SOFTWARE_MODULE_TYPE_USERSPACE_PACKAGE_BUNDLE
	case SW_BOOT_LOADER_STATE_TYPE:
		fillBootLoaderContainer(swcInfo, comp)
	}
	return nil
}
