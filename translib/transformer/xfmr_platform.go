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
	NODE_CFG_TBL = "NODE_CFG"

	IC_NAME_PREFIX = "integrated_circuit"

	/** Upper-level URIs **/
	COMP                 = "/openconfig-platform:components/component"
	COMP_ST              = "/openconfig-platform:components/component/state"
	COMP_STATE_MFG_NAME  = "/openconfig-platform:components/component/state/mfg-name"
	COMP_STATE_NAME      = "/openconfig-platform:components/component/state/name"
	COMP_STATE_FIRM_VER  = "/openconfig-platform:components/component/state/firmware-version"
	COMP_STATE_TEMP_CTR  = "/openconfig-platform:components/component/state/temperature"
	COMP_STATE_PART_NO   = "/openconfig-platform:components/component/state/part-no"
	COMP_STATE_SERIAL_NO = "/openconfig-platform:components/component/state/serial-no"
	COMP_STATE_HW_VER    = "/openconfig-platform:components/component/state/hardware-version"

	COMP_STATE_PARENT            = "/openconfig-platform:components/component/state/parent"
	COMP_STATE_INSTALL_POSITION  = "/openconfig-platform:components/component/state/install-position"
	COMP_STATE_INSTALL_COMPONENT = "/openconfig-platform:components/component/state/install-component"
)

type componentType int64

const (
	CompTypeInvalid componentType = iota
	CompTypeIC
)

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
	}
	return fmt.Sprintf("%s", ct)
}

var compTblMap = map[componentType][]string{
	CompTypeIC: {NODE_CFG_TBL, IC_NAME_PREFIX + "*"},
}

func init() {
	XlateFuncBind("DbToYangPath_pfm_components_path_xfmr", DbToYangPath_pfm_components_path_xfmr)
	XlateFuncBind("Subscribe_pfm_components_xfmr", Subscribe_pfm_components_xfmr)
	XlateFuncBind("YangToDb_pfm_components_xfmr", YangToDb_pfm_components_xfmr)
	XlateFuncBind("DbToYang_pfm_components_xfmr", DbToYang_pfm_components_xfmr)
}

var compTypeCache sync.Map

func validICName(name *string) bool {
	if name == nil || *name == "" {
		return false
	}
	// Expect node name of form integrated-circuitX, where X is an integer
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
	default:
		return CompTypeInvalid, fmt.Errorf("component name %s did not match with supported types.", compName)
	}
}
func getCompType(name string, d *db.DB) componentType {
	if val, ok := compTypeCache.Load(name); ok {
		return val.(componentType)
	}
	compType, err := getCompTypeByName(name)
	if err == nil {
		compTypeCache.Store(name, compType)
		return compType
	}
	return CompTypeInvalid
}

/* Helper for the main subscribe transformer handling the TRANSLATE_EXISTS case. */
func translateExists(inParams XfmrSubscInParams, key string) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	dbNum := db.StateDB
	d := inParams.dbs[dbNum]
	if d == nil {
		dbNum := db.ConfigDB
		d = inParams.dbs[dbNum]
	}
	if d == nil {
		return result, fmt.Errorf("translateExists: No usable DB client in inParams (checked %v and %v)", db.StateDB.Name(), db.ConfigDB.Name())
	}
	compType := getCompType(key, d)
	if compType == CompTypeInvalid {
		return result, nil
	}
	tblInfo, ok := compTblMap[compType]
	if !ok {
		return result, errors.New("table not found.")
	}
	tblName := tblInfo[0]
	tblKey := key
	switch compType {
	case CompTypeIC:
		dbNum = db.ConfigDB
	}
	result.dbDataMap = RedisDbSubscribeMap{dbNum: {tblName: {tblKey: {}}}}
	log.V(3).Infof("+++ Subscribe_pfm_components_xfmr result: %v %v %v +++", dbNum, tblName, tblKey)
	return result, nil
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

func getPfmRootObject(s *ygot.GoStruct) *ocbinds.OpenconfigPlatform_Components {
	if s == nil {
		return nil
	}
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.Components
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
	if validICName(&key) {
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
		if comp.IntegratedCircuit != nil && comp.IntegratedCircuit.Config != nil {
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
		}
	}
	log.V(3).Infof("YangToDb_pfm_components_xfmr: result %v", memMap)
	return memMap, nil
}
var DbToYangPath_pfm_components_path_xfmr PathXfmrDbToYangFunc = func(inParams XfmrDbToYgPathParams) error {
	rootPath := COMP

	log.V(3).Infof("DbToYangPath_pfm_path_xfmr: inParams: %#v", inParams)

	if len(inParams.tblKeyComp) == 0 {
		return fmt.Errorf("Invalid tblKeyCom for pfm path xmfr:%v", inParams.tblKeyComp)
	}

	tblKey := inParams.tblKeyComp[0]
	inParams.ygPathKeys[rootPath+"/name"] = tblKey
	log.V(3).Info("DbToYangPath_pfm_path_xfmr:- params.ygPathKeys: ", inParams.ygPathKeys)

	return nil
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
		var subCompTblName, subCompTblKey string
		var subCompDb db.DBNum
		if cType == CompTypeIC {
			/* The integrated-circuit subtree is backed by multiple DB tables but
			 * our compTblMap only captures one.  The special handling here is to
			 * cover all tables.
			 * For wildcard expansion we use the NODE_CFG_TBL in ConfigDB but for
			 * sample cases on the specific counter paths we need the counter
			 * tables. */
			tblDb = db.ConfigDB
		}
		if result.dbDataMap[tblDb] == nil {
			result.dbDataMap[tblDb] = make(map[string]map[string]map[string]string)
		}
		if result.dbDataMap[tblDb][tblName] == nil {
			result.dbDataMap[tblDb][tblName] = make(map[string]map[string]string)
		}
		if subCompTblName != "" {
			if result.dbDataMap[subCompDb] == nil {
				result.dbDataMap[subCompDb] = make(map[string]map[string]map[string]string)
			}
			if result.dbDataMap[subCompDb][subCompTblName] == nil {
				result.dbDataMap[subCompDb][subCompTblName] = make(map[string]map[string]string)
			}
			result.dbDataMap[subCompDb][subCompTblName][subCompTblKey] = map[string]string{}
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
		var stName, stType bool
		switch targetUriPath {
		case COMP_ST:
			stName, stType = true, true
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
				return fmt.Errorf("invalid input component name: %s", compName)
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
			return fmt.Errorf("invalid input component name for state path: %s", compName)
		}
		ygot.BuildEmptyTree(pf_comp)
		ygot.BuildEmptyTree(pf_comp.State)
		ygot.BuildEmptyTree(pf_comp.State.Temperature)
		if err = compTypeToFuncCall(compType, compName, subKey, pf_comp, targetUriPath, dbs, StatePaths, ygRoot); err != nil {
			log.V(3).Info(err)
		}
	default:
		/* The following cases are handled above:
		 *   /components/component
		 *   /components/component[name=<component_name>]
		 *   /components/component[name=<component_name>]/config
		 *   /components/component[name=<component_name>]/state
		 *   /components/component[name=<component_name>]/healthz
		 *   /components/component[name=<component_name>]/healthz/faults
		 * so the request must be for a specific component's leaf or subtree,
		 * e.g. /components/component[name=integrated_circuit]/integrated-circuit */
		// TODO - Can we de-dup this code with compTypeToFuncCall?  No good way to set pathType...
		compType := getCompType(compName, d)
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
	/* Rails and subcomponents may have a second level key */
	subKey := pathInfo.Var("name#2")
	return getSysComponents(getPfmRootObject(inParams.ygRoot), targetUriPath, inParams, compName, subKey)
}
