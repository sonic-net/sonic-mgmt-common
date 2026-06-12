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
	TRANSCEIVER_TBL     = "TRANSCEIVER_INFO"
	TRANSCEIVER_STATUS  = "TRANSCEIVER_STATUS"
	TRANSCEIVER_DOM     = "TRANSCEIVER_DOM_SENSOR"

	XCVR_LANE_LIMIT = 8

	XCVR_KEY_PREFIX = "Ethernet"
	IC_NAME_PREFIX  = "integrated_circuit"
	CHASSIS_PREFIX  = "chassis"

	/** Transceiver status values **/
	SFP_STATUS_REMOVED  = "0"
	SFP_STATUS_INSERTED = "1"

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
	COMP_STATE_REMOVABLE         = "/openconfig-platform:components/component/state/removable"
	COMP_STATE_SERIAL_NO         = "/openconfig-platform:components/component/state/serial-no"
	COMP_STATE_TYPE              = "/openconfig-platform:components/component/state/type"
	COMP_STATE_PARENT            = "/openconfig-platform:components/component/state/parent"
	COMP_STATE_TEMP_CTR          = "/openconfig-platform:components/component/state/temperature"
	COMP_STATE_TEMP              = "/openconfig-platform:components/component/state/temperature/instant"

	/** Supported Xcvr URIs **/
	XCVR_BASE_PREFIX = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver"
	XCVR_BASE_STATE  = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state"
	XCVR_FORM_FACTOR = "/openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state/form-factor"
)

type componentType int64

const (
	CompTypeInvalid componentType = iota
	CompTypeXcvr
	CompTypeIC
)

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

type PathType int

const (
	/* Represents all paths under /components/component */
	AllPaths PathType = iota
	/* Represents all paths under /components/component/config */
	ConfigPaths
	/* Represents all paths under /components/component/state */
	StatePaths
	/* Represents a path to a specific leaf */
	SingularPath
)

func (pt PathType) String() string {
	switch pt {
	case AllPaths:
		return "AllPaths"
	case ConfigPaths:
		return "ConfigPaths"
	case StatePaths:
		return "StatePaths"
	case SingularPath:
		return "SingularPath"
	}
	return fmt.Sprintf("%d", int(pt))
}

func (ct componentType) String() string {
	switch ct {
	case CompTypeInvalid:
		return "CompTypeInvalid"
	case CompTypeXcvr:
		return "CompTypeXcvr"
	case CompTypeIC:
		return "CompTypeIC"
	}
	return fmt.Sprintf("%d", int(ct))
}

var compTblMap = map[componentType][]string{
	CompTypeXcvr: {TRANSCEIVER_STATUS, XCVR_KEY_PREFIX + "*"},
	CompTypeIC:   {NODE_CFG_TBL, IC_NAME_PREFIX + "*"},
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
	case validXcvrName(&compName):
		return CompTypeXcvr, nil
	case validICName(&compName):
		return CompTypeIC, nil

	default:
		return CompTypeInvalid, fmt.Errorf("component name %s did not match with supported types.", compName)
	}
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
	if strings.HasPrefix(uri, "/openconfig-platform:components/component/oc-transceiver:transceiver") {
		cTypes = []componentType{CompTypeXcvr}
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
	case CompTypeXcvr:
		return fillXcvrInfo(pfComp, compName, pType != SingularPath, "", targetUriPath, dbs)
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

	if pf_cpts == nil {
		return nil
	}

	log.V(3).Infof("Preparing dB for system components")

	uri := inParams.uri
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
		if compType == CompTypeXcvr {
			ygot.BuildEmptyTree(pf_comp.State)
		}
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
			return fmt.Errorf("invalid input component name: %s", compName)
		}
		ygot.BuildEmptyTree(pf_comp)
		switch compType {
		case CompTypeXcvr:
			ygot.BuildEmptyTree(pf_comp.Transceiver)
			ygot.BuildEmptyTree(pf_comp.Transceiver.State)
			ygot.BuildEmptyTree(pf_comp.Transceiver.Config)

			laneIdx := NewPathInfo(uri).Var("index")
			switch targetUriPath {
			case XCVR_BASE_PREFIX /*XCVR_BASE_CONFIG,*/, XCVR_BASE_STATE:
				return fillXcvrInfo(pf_comp, compName, true, laneIdx, targetUriPath, dbs)
			default:
				/* For individual components*/
				return fillXcvrInfo(pf_comp, compName, false, laneIdx, targetUriPath, dbs)
			}
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
	/*subkey is second level key set to null which is not used for now*/
	subKey := ""
	return getSysComponents(getPfmRootObject(inParams.ygRoot), targetUriPath, inParams, compName, subKey)
}

func validXcvrName(name *string) bool {
	if name == nil || *name == "" {
		return false
	}

	/* Expect tranceiver name of form EthernetX, where X is an integer */
	if !strings.HasPrefix(*name, XCVR_KEY_PREFIX) {
		return false
	}

	sp := strings.SplitAfter(*name, "Ethernet")
	if len(sp) < 2 {
		return false
	}

	if _, err := strconv.Atoi(sp[1]); err != nil {
		return false
	}
	return true
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
		fillXcvrLaneInfo(xcvrCom, uint16(idx), xcvrInfo, name, maxLanes)
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
					fillXcvrLaneInfo(xcvrCom, uint16(i), xcvrInfo, name, maxLanes)
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
		xcvrInfo.Presence = false
		return xcvrInfo, err
	}

	for i := 0; i < XCVR_LANE_LIMIT; i++ {
		xcvrInfo.Lanes[i].RxPowerLane = xcvrDOMEntry.Get(fmt.Sprintf("rx%dpower", i+1))
		xcvrInfo.Lanes[i].TxBiasLane = xcvrDOMEntry.Get(fmt.Sprintf("tx%dbias", i+1))
		xcvrInfo.Lanes[i].TxPowerLane = xcvrDOMEntry.Get(fmt.Sprintf("tx%dpower", i+1))
		xcvrInfo.Lanes[i].TxDisable = xcvrDOMEntry.Get(fmt.Sprintf("tx%ddisable", i+1))
	}

	xcvrInfo.Temperature = xcvrDOMEntry.Get("temperature")

	return xcvrInfo, err
}

var sfpTypeToMaxLanesMap = map[string]int{
	"SFP/SFP+/SFP28":                1,
	"QSFP":                          4,
	"QSFP+ or later":                4,
	"QSFP28 or later":               4,
	"OSFP 8X Pluggable Transceiver": 8,
	"QSFP-DD Double Density 8X Pluggable Transceiver": 8,
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
