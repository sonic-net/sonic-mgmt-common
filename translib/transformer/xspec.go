////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Dell, Inc.                                                 //
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

package transformer

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/utils"
	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
)

// subscription flags
const (
	subsPrefSample utils.Bits = 1 << iota
	subsOnChangeEnable
	subsOnChangeDisable
	subsDelAsUpdate
)

/* Data needed to construct lookup table from yang */
type yangXpathInfo struct {
	tableName          *string
	xfmrTbl            *string
	childTable         []string
	yangEntry          *yang.Entry
	keyXpath           map[int]*[]string
	delim              string
	fieldName          string
	xfmrFunc           string
	xfmrField          string
	validateFunc       string
	xfmrKey            string
	keyName            *string
	dbIndex            db.DBNum
	keyLevel           uint8
	isKey              bool
	defVal             string
	tblOwner           *bool
	hasChildSubTree    bool
	hasNonTerminalNode bool
	subscribeMinIntvl  int
	virtualTbl         *bool
	nameWithMod        *string
	operationalQP      bool
	hasChildOpertnlNd  bool
	yangType           yangElementType
	xfmrPath           string
	compositeFields    []string
	dbKeyCompCnt       int
	subscriptionFlags  utils.Bits
	isDataSrcDynamic   *bool
	isRefByKey         bool
}

type dbInfo struct {
	dbIndex     db.DBNum
	keyName     *string
	dbEntry     *yang.Entry
	yangXpath   []string
	module      string
	delim       string
	leafRefPath []string
	listName    []string
	keyList     []string
	xfmrKey     string
	xfmrValue   *string
	hasXfmrFn   bool
	cascadeDel  int8
	yangType    yangElementType
	isKey       bool
}

type moduleAnnotInfo struct {
	xfmrPre  string
	xfmrPost string
}

type depTblData struct {
	/* list of dependent tables within same sonic yang for a given table,
	   as provided by CVL in child first order */
	DepTblWithinMdl []string
	/* list of dependent tables across sonic yangs for a given table,
	   as provided by CVL in child first order */
	DepTblAcrossMdl []string
}

type sonicTblSeqnInfo struct {
	OrdTbl []string //all tables within sonic yang, as provided by CVL in child first order
	DepTbl map[string]depTblData
}

type mdlInfo struct {
	Org string
	Ver string
}

var dbConfigMap = make(map[string]interface{})
var xYangSpecMap map[string]*yangXpathInfo
var xDbSpecMap map[string]*dbInfo
var xYangModSpecMap map[string]*moduleAnnotInfo
var xYangRpcSpecMap map[string]string
var xDbSpecTblSeqnMap map[string]*sonicTblSeqnInfo
var xDbRpcSpecMap map[string]string
var xMdlCpbltMap map[string]*mdlInfo
var sonicOrdTblListMap map[string][]string
var sonicLeafRefMap map[string][]string

func init() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	dbConfigMap = db.GetDbConfigMap()

	go func() {
		for {
			<-sigs

			mapPrint("/tmp/fullSpec.txt")
			dbMapPrint("/tmp/dbSpecMap.txt")
			xDbSpecTblSeqnMapPrint("/tmp/dbSpecTblSeqnMap.txt")

			debug.FreeOSMemory()
		}
	}()
	log.Info("xspec init done ...")
}

/* Add module name to map storing model info for model capabilities */
func addMdlCpbltEntry(yangMdlNm string) {
	if xMdlCpbltMap == nil {
		xMdlCpbltMap = make(map[string]*mdlInfo)
	}
	mdlInfoEntry := new(mdlInfo)
	if mdlInfoEntry == nil {
		log.Warningf("Memory allocation failure for storing model info for gnmi - module %v", yangMdlNm)
		return
	}
	mdlInfoEntry.Org = ""
	mdlInfoEntry.Ver = ""
	xMdlCpbltMap[yangMdlNm] = mdlInfoEntry
}

/* Add version and organization info for model capabilities into map */
func addMdlCpbltData(yangMdlNm string, version string, organization string) {
	if xMdlCpbltMap == nil {
		xMdlCpbltMap = make(map[string]*mdlInfo)
	}
	mdlInfoEntry, ok := xMdlCpbltMap[yangMdlNm]
	if (!ok) || (mdlInfoEntry == nil) {
		mdlInfoEntry = new(mdlInfo)
		if mdlInfoEntry == nil {
			log.Warningf("Memory allocation failure for storing model info for gnmi - module %v", yangMdlNm)
			return
		}
	}
	mdlInfoEntry.Ver = version
	mdlInfoEntry.Org = organization
	xMdlCpbltMap[yangMdlNm] = mdlInfoEntry
}

/* update transformer spec path in db spec node */
func updateDbTableData(xpath string, tableName string) {
	if len(tableName) == 0 {
		return
	}
	_, ok := xDbSpecMap[tableName]
	if ok {
		xDbSpecMap[tableName].yangXpath = append(xDbSpecMap[tableName].yangXpath, xpath)
	}
}

func childContainerListPresenceFlagSet(xpath string) {
	parXpath := parentXpathGet(xpath)
	for {
		if parXpath == "" {
			break
		}
		if parXpathData, ok := xYangSpecMap[parXpath]; ok {
			parXpathData.hasNonTerminalNode = true
		}
		parXpath = parentXpathGet(parXpath)
	}
}

func childSubTreePresenceFlagSet(xpath string) {
	parXpath := parentXpathGet(xpath)
	for {
		if parXpath == "" {
			break
		}
		if parXpathData, ok := xYangSpecMap[parXpath]; ok {
			parXpathData.hasChildSubTree = true
		}
		parXpath = parentXpathGet(parXpath)
	}
}

func childOperationalNodeFlagSet(xpath string) {
	if xpathData, ok := xYangSpecMap[xpath]; ok {
		yangType := xpathData.yangType
		if (yangType != YANG_LEAF) && (yangType != YANG_LEAF_LIST) {
			xpathData.hasChildOpertnlNd = true
		}
	}

	parXpath := parentXpathGet(xpath)
	for {
		if parXpath == "" {
			break
		}
		if parXpathData, ok := xYangSpecMap[parXpath]; ok {
			if parXpathData.hasChildOpertnlNd {
				break
			}
			parXpathData.hasChildOpertnlNd = true
		}
		parXpath = parentXpathGet(parXpath)
	}
}

/* Recursive api to fill the map with yang details */
func yangToDbMapFill(keyLevel uint8, xYangSpecMap map[string]*yangXpathInfo, entry *yang.Entry, xpathPrefix string, xpathFull string) {
	xpath := ""
	curKeyLevel := uint8(0)
	curXpathFull := ""

	if entry != nil && (entry.Kind == yang.CaseEntry || entry.Kind == yang.ChoiceEntry) {
		curXpathFull = xpathFull + "/" + entry.Name
		if _, ok := xYangSpecMap[xpathPrefix]; ok {
			curKeyLevel = xYangSpecMap[xpathPrefix].keyLevel
		}
		curXpathData, ok := xYangSpecMap[curXpathFull]
		if !ok {
			curXpathData = new(yangXpathInfo)
			curXpathData.subscribeMinIntvl = XFMR_INVALID
			curXpathData.dbIndex = db.ConfigDB // default value
			xYangSpecMap[curXpathFull] = curXpathData
		}
		if entry.Kind == yang.CaseEntry {
			curXpathData.yangType = YANG_CASE
		} else {
			curXpathData.yangType = YANG_CHOICE
		}
		curXpathData.yangEntry = entry
		if xYangSpecMap[xpathPrefix].subscriptionFlags.Has(subsPrefSample) {
			curXpathData.subscriptionFlags.Set(subsPrefSample)
		}
		if xYangSpecMap[xpathPrefix].subscriptionFlags.Has(subsOnChangeDisable) {
			curXpathData.subscriptionFlags.Set(subsOnChangeDisable)
		} else if xYangSpecMap[xpathPrefix].subscriptionFlags.Has(subsOnChangeEnable) {
			curXpathData.subscriptionFlags.Set(subsOnChangeEnable)
		}
		curXpathData.subscribeMinIntvl = xYangSpecMap[xpathPrefix].subscribeMinIntvl
		xpath = xpathPrefix
	} else {
		if entry == nil {
			return
		}

		yangType := getYangTypeIntId(entry)

		if entry.Parent.Annotation["schemapath"].(string) == "/" {
			xpath = entry.Annotation["schemapath"].(string)
			/* module name is delimetered from the rest of schema path with ":" */
			xpath = string('/') + strings.Replace(xpath[1:], "/", ":", 1)
		} else {
			xpath = xpathPrefix + "/" + entry.Name
		}

		updateChoiceCaseXpath := false
		curXpathFull = xpath
		if xpathPrefix != xpathFull {
			curXpathFull = xpathFull + "/" + entry.Name
			if annotNode, ok := xYangSpecMap[curXpathFull]; ok {
				xpathData := new(yangXpathInfo)
				xpathData.subscribeMinIntvl = XFMR_INVALID
				xpathData.dbIndex = db.ConfigDB // default value
				xYangSpecMap[xpath] = xpathData
				copyYangXpathSpecData(xYangSpecMap[xpath], annotNode)
				updateChoiceCaseXpath = true
			}
		}

		xpathData, ok := xYangSpecMap[xpath]
		if !ok {
			xpathData = new(yangXpathInfo)
			xYangSpecMap[xpath] = xpathData
			xpathData.dbIndex = db.ConfigDB // default value
			xpathData.subscribeMinIntvl = XFMR_INVALID
		} else {
			if len(xpathData.xfmrFunc) > 0 {
				childSubTreePresenceFlagSet(xpath)
			}
		}

		if xpathData.tableName != nil && *xpathData.tableName != "" {
			childToUpdateParent(xpath, *xpathData.tableName)
		}

		parentXpathData, ok := xYangSpecMap[xpathPrefix]
		if ok && parentXpathData == nil {
			ok = false
		}
		/* Inherit the tableOwner status from the parent until a new table annotation is encountered at the current node */
		if xpathData.tblOwner == nil && ok && parentXpathData.tblOwner != nil {
			if xpathData.xfmrTbl == nil && xpathData.tableName == nil {
				xpathData.tblOwner = new(bool)
				*xpathData.tblOwner = *parentXpathData.tblOwner
			}
		}

		/* init current xpath table data with its parent data, change only if needed. */
		if ok && xpathData.tableName == nil {
			if xpathData.tableName == nil && parentXpathData.tableName != nil && xpathData.xfmrTbl == nil {
				xpathData.tableName = parentXpathData.tableName
				if xpathData.dbKeyCompCnt == 0 {
					xpathData.dbKeyCompCnt = parentXpathData.dbKeyCompCnt
				}
			} else if xpathData.xfmrTbl == nil && parentXpathData.xfmrTbl != nil {
				xpathData.xfmrTbl = parentXpathData.xfmrTbl
			}
		}

		if ok && xpathData.dbIndex == db.ConfigDB && parentXpathData.dbIndex != db.ConfigDB {
			// If DB Index is not annotated and parent DB index is annotated inherit the DB Index of the parent
			xpathData.dbIndex = parentXpathData.dbIndex
		}

		if ok && (len(xpathData.validateFunc) == 0) && (len(parentXpathData.validateFunc) > 0) {
			xpathData.validateFunc = parentXpathData.validateFunc
		}

		if ok && len(parentXpathData.xfmrFunc) > 0 && len(xpathData.xfmrFunc) == 0 {
			xpathData.xfmrFunc = parentXpathData.xfmrFunc
		}

		if ok && len(parentXpathData.xfmrPath) > 0 && len(xpathData.xfmrPath) == 0 {
			xpathData.xfmrPath = parentXpathData.xfmrPath
		}

		if ok && (parentXpathData.subscribeMinIntvl == XFMR_INVALID) {
			log.Warningf("Susbscribe MinInterval/OnChange flag is set to invalid for(%v) \r\n", xpathPrefix)
			return
		}

		if ok {
			if !xpathData.subscriptionFlags.Has(subsOnChangeDisable) && parentXpathData.subscriptionFlags.Has(subsOnChangeDisable) {
				xpathData.subscriptionFlags.Set(subsOnChangeDisable)
			} else if !xpathData.subscriptionFlags.Has(subsOnChangeEnable) && parentXpathData.subscriptionFlags.Has(subsOnChangeEnable) {
				xpathData.subscriptionFlags.Set(subsOnChangeEnable)
			}

			if xpathData.subscribeMinIntvl == XFMR_INVALID {
				xpathData.subscribeMinIntvl = parentXpathData.subscribeMinIntvl
			}

			if !xpathData.subscriptionFlags.Has(subsPrefSample) && parentXpathData.subscriptionFlags.Has(subsPrefSample) {
				xpathData.subscriptionFlags.Set(subsPrefSample)
			}

			if entry.Prefix.Name != parentXpathData.yangEntry.Prefix.Name {
				if _, ok := entry.Annotation["modulename"]; ok {
					xpathData.nameWithMod = new(string)
					*xpathData.nameWithMod = entry.Annotation["modulename"].(string) + ":" + entry.Name
				}
			}

		}

		if ((yangType == YANG_LEAF) || (yangType == YANG_LEAF_LIST)) && (len(xpathData.fieldName) == 0) {

			if xpathData.tableName != nil && xDbSpecMap[*xpathData.tableName] != nil {
				if _, ok := xDbSpecMap[*xpathData.tableName+"/"+entry.Name]; ok {
					xpathData.fieldName = entry.Name
				} else {
					if _, ok := xDbSpecMap[*xpathData.tableName+"/"+strings.ToUpper(entry.Name)]; ok {
						xpathData.fieldName = strings.ToUpper(entry.Name)
					}
					if _, ok := xDbSpecMap[*xpathData.tableName+"/"+entry.Name]; ok {
						xpathData.fieldName = entry.Name
					}
				}
			} else if xpathData.xfmrTbl != nil {
				/* table transformer present */
				xpathData.fieldName = entry.Name
			}
		}
		if (yangType == YANG_LEAF) && (len(entry.Default) > 0) {
			xpathData.defVal = entry.Default
		}

		if ((yangType == YANG_LEAF) || (yangType == YANG_LEAF_LIST)) && (len(xpathData.fieldName) > 0) && (xpathData.tableName != nil) {
			dbPath := *xpathData.tableName + "/" + xpathData.fieldName
			_, ok := xDbSpecMap[dbPath]
			if ok && xDbSpecMap[dbPath] != nil {
				xDbSpecMap[dbPath].yangXpath = append(xDbSpecMap[dbPath].yangXpath, xpath)
				if xDbSpecMap[dbPath].isKey {
					xpathData.fieldName = ""
				}
			}
		}

		/* fill table with key data. */
		curKeyLevel := keyLevel
		if len(entry.Key) != 0 {
			parentKeyLen := 0

			/* create list with current keys */
			keyXpath := make([]string, len(strings.Split(entry.Key, " ")))
			isOcMdl := strings.HasPrefix(xpath, "/"+OC_MDL_PFX)
			for id, keyName := range strings.Split(entry.Key, " ") {
				keyXpath[id] = xpath + "/" + keyName
				if _, ok := xYangSpecMap[xpath+"/"+keyName]; !ok {
					keyXpathData := new(yangXpathInfo)
					keyXpathData.subscribeMinIntvl = XFMR_INVALID
					keyXpathData.dbIndex = db.ConfigDB // default value
					xYangSpecMap[xpath+"/"+keyName] = keyXpathData
				}
				xYangSpecMap[xpath+"/"+keyName].isKey = true
				if isOcMdl {
					var keyLfsInContainerXpaths []string
					if configContEntry, ok := entry.Dir["config"]; ok && configContEntry != nil { //OC Mdl list has config container
						if _, keyLfOk := configContEntry.Dir[keyName]; keyLfOk {
							keyLfsInContainerXpaths = append(keyLfsInContainerXpaths, xpath+CONFIG_CNT_WITHIN_XPATH+keyName)
						}
					}

					/* Mark OC Model list state-container leaves that are also list key-leaves,as isRefByKey,even though there
					is no yang leaf-reference from list-key leaves.This will enable xfmr infra to fill them and elliminate
					app annotation
					*/
					if stateContEntry, ok := entry.Dir["state"]; ok && stateContEntry != nil { //OC Mdl list has state container
						if _, keyLfOk := stateContEntry.Dir[keyName]; keyLfOk {
							keyLfsInContainerXpaths = append(keyLfsInContainerXpaths, xpath+STATE_CNT_WITHIN_XPATH+keyName)
						}
					}

					for _, keyLfInContXpath := range keyLfsInContainerXpaths {
						if _, ok := xYangSpecMap[keyLfInContXpath]; !ok {
							xYangSpecMap[keyLfInContXpath] = new(yangXpathInfo)
							xYangSpecMap[keyLfInContXpath].subscribeMinIntvl = XFMR_INVALID
							xYangSpecMap[keyLfInContXpath].dbIndex = db.ConfigDB // default value
						}
						xYangSpecMap[keyLfInContXpath].isRefByKey = true
					}

				}
			}

			xpathData.keyXpath = make(map[int]*[]string, (parentKeyLen + 1))
			k := 0
			if parentXpathData != nil {
				for ; k < parentKeyLen; k++ {
					/* copy parent key-list to child key-list*/
					xpathData.keyXpath[k] = parentXpathData.keyXpath[k]
				}
			}
			xpathData.keyXpath[k] = &keyXpath
			xpathData.keyLevel = curKeyLevel
			curKeyLevel++
		} else if parentXpathData != nil && parentXpathData.keyXpath != nil {
			xpathData.keyXpath = parentXpathData.keyXpath
		}
		xpathData.yangEntry = entry
		xpathData.yangType = yangType

		if xpathData.subscribeMinIntvl == XFMR_INVALID {
			xpathData.subscribeMinIntvl = 0
		}

		if !xpathData.subscriptionFlags.Has(subsPrefSample) && xpathData.subscriptionFlags.Has(subsOnChangeDisable) {
			if log.V(5) {
				log.Infof("subscribe OnChange is disabled so setting subscribe preference to default/sample from onchange for xpath - %v", xpath)
			}
			xpathData.subscriptionFlags.Set(subsPrefSample)
		}

		if updateChoiceCaseXpath {
			copyYangXpathSpecData(xYangSpecMap[curXpathFull], xYangSpecMap[xpath])
		}

		if (yangType == YANG_CONTAINER) || (yangType == YANG_LIST) {
			childContainerListPresenceFlagSet(xpath)
		}

	}

	/* get current obj's children */
	var childList []string
	for k := range entry.Dir {
		childList = append(childList, k)
	}

	/* now recurse, filling the map with current node's children info */
	for _, child := range childList {
		yangToDbMapFill(curKeyLevel, xYangSpecMap, entry.Dir[child], xpath, curXpathFull)
	}
}

/* find and set operation query parameter nodes */
var xlateUpdateQueryParamInfo = func() {

	for xpath := range xYangSpecMap {
		if xYangSpecMap[xpath].yangEntry != nil {
			readOnly := xYangSpecMap[xpath].yangEntry.ReadOnly()
			if xYangSpecMap[xpath].yangType == YANG_LEAF || xYangSpecMap[xpath].yangType == YANG_LEAF_LIST {
				xYangSpecMap[xpath].yangEntry = nil //memory optimization - don't cache for leafy nodes
			}
			if !readOnly {
				continue
			}
		}

		if strings.Contains(xpath, STATE_CNT_WITHIN_XPATH) {
			cfgXpath := strings.Replace(xpath, STATE_CNT_WITHIN_XPATH, CONFIG_CNT_WITHIN_XPATH, -1)
			if strings.HasSuffix(cfgXpath, STATE_CNT_SUFFIXED_XPATH) {
				suffix_idx := strings.LastIndex(cfgXpath, STATE_CNT_SUFFIXED_XPATH)
				cfgXpath = cfgXpath[:suffix_idx] + CONFIG_CNT_SUFFIXED_XPATH
			}
			if _, ok := xYangSpecMap[cfgXpath]; !ok {
				xYangSpecMap[xpath].operationalQP = true
				childOperationalNodeFlagSet(xpath)
			}
		} else if strings.HasSuffix(xpath, STATE_CNT_SUFFIXED_XPATH) {
			suffix_idx := strings.LastIndex(xpath, STATE_CNT_SUFFIXED_XPATH)
			cfgXpath := xpath[:suffix_idx] + CONFIG_CNT_SUFFIXED_XPATH
			if _, ok := xYangSpecMap[cfgXpath]; !ok {
				xYangSpecMap[xpath].operationalQP = true
				childOperationalNodeFlagSet(xpath)
			}
		} else {
			xYangSpecMap[xpath].operationalQP = true
			childOperationalNodeFlagSet(xpath)
		}
	}
}

/* Build lookup table based of yang xpath */
func yangToDbMapBuild(entries map[string]*yang.Entry) {
	if entries == nil {
		return
	}

	if xYangSpecMap == nil {
		xYangSpecMap = make(map[string]*yangXpathInfo)
	}

	for _, e := range entries {
		if e == nil || len(e.Dir) == 0 {
			continue
		}

		/* Start to fill xpath based map with yang data */
		keyLevel := uint8(0)
		yangToDbMapFill(keyLevel, xYangSpecMap, e, "", "")
	}

	sonicOrdTblListMap = make(map[string][]string)
	jsonfile := YangPath + TblInfoJsonFile
	xlateJsonTblInfoLoad(sonicOrdTblListMap, jsonfile)

	xlateUpdateQueryParamInfo()
	sonicLeafRefMap = nil
}

func dbSpecXpathGet(inPath string) (string, error) {
	/* This api currently handles only cointainer inside a list for sonic-yang.
	   Should be enhanced to support nested list in future. */
	var err error
	specPath := ""

	xpath, _, err := XfmrRemoveXPATHPredicates(inPath)
	if err != nil {
		log.Warningf("xpath conversion failed for(%v) \r\n", inPath)
		return specPath, err
	}

	pathList := strings.Split(xpath, "/")
	if len(pathList) < 3 {
		log.Warningf("Leaf-ref path not valid(%v) \r\n", inPath)
		return specPath, err
	}

	tableName := pathList[2]
	fieldName := pathList[len(pathList)-1]
	specPath = tableName + "/" + fieldName
	return specPath, err
}

/* Convert a relative path to an absolute path */
func relativeToActualPathGet(relPath string, entry *yang.Entry) string {
	actDbSpecPath := ""
	xpath, _, err := XfmrRemoveXPATHPredicates(relPath)
	if err != nil {
		return actDbSpecPath
	}

	pathList := strings.Split(xpath[1:], "/")
	if len(pathList) > 3 && (pathList[len(pathList)-3] != "..") {
		tableName := pathList[len(pathList)-3]
		fieldName := pathList[len(pathList)-1]
		actDbSpecPath = tableName + "/" + fieldName
	}
	return actDbSpecPath
}

/* Fill the map with db details */
func dbMapFill(tableName string, curPath string, moduleNm string, xDbSpecMap map[string]*dbInfo, entry *yang.Entry) {
	if entry == nil {
		return
	}

	entryType := getYangTypeIntId(entry)
	tblDbIndex := db.ConfigDB
	var tblSpecInfo *dbInfo
	tblOk := false

	if entry.Name != moduleNm {
		dbXpath := ""
		tableContainer := false
		if tableName == "" && entryType == YANG_CONTAINER {
			tableContainer = true
		}
		if tableContainer {
			tableName = entry.Name
			dbXpath = tableName
		} else {
			// This includes all nodes which are not module or table containers
			dbXpath = tableName + "/" + entry.Name
			if entry.IsList() && entry.Parent != nil && entry.Parent.IsList() { // nested/child list of list under table-level container
				for siblingNm := range entry.Parent.Dir {
					if (siblingNm == entry.Name) || strings.Contains(entry.Parent.Key, siblingNm) {
						continue
					}
					log.Warningf("Nested list %v can have only key-leaf siblings under parent list %v for table %v in module %v",
						entry.Name, entry.Parent.Name, tableName, moduleNm)
					cleanupNestedListSpecInfo(tableName, entry.Parent)
					return
				}

				if nestedListProcessingErr := sonicYangNestedListValidateElements(tableName, entry); nestedListProcessingErr != nil {
					cleanupNestedListSpecInfo(tableName, entry.Parent)
					return
				}
				dbXpath = tableName + "/" + entry.Parent.Name + "/" + entry.Name
			} else if entry.IsList() && entry.Parent != nil && entry.Parent.Name != tableName {
				log.Warningf("Nested list %v not supported under a non-table level container yang node %v in module %v", entry.Name, entry.Parent.Name, moduleNm)
				return
			}
			if tblSpecInfo, tblOk = xDbSpecMap[tableName]; tblOk {
				tblDbIndex = xDbSpecMap[tableName].dbIndex
			}
		}
		if _, ok := xDbSpecMap[dbXpath]; !ok {
			xDbSpecMap[dbXpath] = new(dbInfo)
		}

		xDbSpecMap[dbXpath].dbIndex = tblDbIndex
		xDbSpecMap[dbXpath].yangType = entryType
		xDbSpecMap[dbXpath].dbEntry = entry
		xDbSpecMap[dbXpath].module = moduleNm
		xDbSpecMap[dbXpath].cascadeDel = XFMR_INVALID
		if tableContainer {
			xDbSpecMap[dbXpath].dbIndex = tblDbIndex
			if entry.Exts != nil && len(entry.Exts) > 0 {
				for _, ext := range entry.Exts {
					dataTagArr := strings.Split(ext.Keyword, ":")
					tagType := dataTagArr[len(dataTagArr)-1]
					switch tagType {
					case "key-name":
						if xDbSpecMap[dbXpath].keyName == nil {
							xDbSpecMap[dbXpath].keyName = new(string)
						}
						*xDbSpecMap[dbXpath].keyName = ext.NName()
					case "db-name":
						xDbSpecMap[dbXpath].dbIndex = db.GetdbNameToIndex(ext.NName())
					case "key-delim":
						xDbSpecMap[dbXpath].delim = ext.NName()
					default:
						log.Infof("Unsupported ext type(%v) for xpath(%v).", tagType, dbXpath)
					}
				}
			}
		} else if tblOk && (entryType == YANG_LIST && len(entry.Key) != 0) {
			if entry.Parent.IsList() { // nested/child list of list under table-level container
				if parentListSpecInfo, parentListOk := xDbSpecMap[tableName+"/"+entry.Parent.Name]; parentListOk && parentListSpecInfo != nil {
					parentListSpecInfo.listName = append(parentListSpecInfo.listName, entry.Name)
				}
			} else {
				tblSpecInfo.listName = append(tblSpecInfo.listName, entry.Name)
			}
			xDbSpecMap[dbXpath].keyList = append(xDbSpecMap[dbXpath].keyList, strings.Split(entry.Key, " ")...)
			for _, keyVal := range xDbSpecMap[dbXpath].keyList {
				dbXpathForKeyLeaf := tableName + "/" + keyVal
				if _, ok := xDbSpecMap[dbXpathForKeyLeaf]; !ok {
					xDbSpecMap[dbXpathForKeyLeaf] = new(dbInfo)
				}
				xDbSpecMap[dbXpathForKeyLeaf].isKey = true
			}
		} else if entryType == YANG_LEAF || entryType == YANG_LEAF_LIST {
			xDbSpecMap[dbXpath].dbEntry = nil //memory optimization - don't cache for leafy nodes
			if entry.Type.Kind == yang.Yleafref {
				var lerr error
				lrefpath := entry.Type.Path
				if strings.Contains(lrefpath, "..") {
					lrefpath = relativeToActualPathGet(lrefpath, entry)
				} else {
					lrefpath, lerr = dbSpecXpathGet(lrefpath)
					if lerr != nil {
						log.Warningf("Failed to add leaf-ref for(%v) \r\n", entry.Type.Path)
						return
					}

				}
				xDbSpecMap[dbXpath].leafRefPath = append(xDbSpecMap[dbXpath].leafRefPath, lrefpath)
				sonicLeafRefMap[lrefpath] = append(sonicLeafRefMap[lrefpath], dbXpath)
			} else if entry.Type.Kind == yang.Yunion && len(entry.Type.Type) > 0 {
				for _, ltype := range entry.Type.Type {
					if ltype.Kind == yang.Yleafref {
						var lerr error
						lrefpath := ltype.Path
						if strings.Contains(lrefpath, "..") {
							lrefpath = relativeToActualPathGet(lrefpath, entry)
						} else {
							lrefpath, lerr = dbSpecXpathGet(lrefpath)
							if lerr != nil {
								log.Warningf("Failed to add leaf-ref for(%v) \r\n", ltype.Path)
								return
							}

						}
						xDbSpecMap[dbXpath].leafRefPath = append(xDbSpecMap[dbXpath].leafRefPath, lrefpath)
						sonicLeafRefMap[lrefpath] = append(sonicLeafRefMap[lrefpath], dbXpath)
					}
				}
			}
		}
	} else {
		moduleXpath := "/" + moduleNm + ":" + entry.Name
		xDbSpecMap[moduleXpath] = new(dbInfo)
		xDbSpecMap[moduleXpath].dbEntry = entry
		xDbSpecMap[moduleXpath].yangType = entryType
		xDbSpecMap[moduleXpath].module = moduleNm
		xDbSpecMap[moduleXpath].cascadeDel = XFMR_INVALID
		xDbSpecMap[moduleXpath].dbIndex = db.MaxDB
		for {
			done := true
			sncTblInfo := new(sonicTblSeqnInfo)
			if sncTblInfo == nil {
				log.Warningf("Memory allocation failure for storing Tbl order and dependency info for sonic module %v", moduleNm)
				break
			}
			cvlSess, cvlRetSess := db.NewValidationSession()
			if cvlRetSess != nil {
				log.Warningf("Failure in creating CVL validation session object required to use CVl API to get Tbl info for module %v - %v", moduleNm, cvlRetSess)
				break
			}
			var cvlRetOrdTbl cvl.CVLRetCode
			sncTblInfo.OrdTbl, cvlRetOrdTbl = cvlSess.GetOrderedTables(moduleNm)
			if cvlRetOrdTbl != cvl.CVL_SUCCESS {
				log.Warningf("Failure in cvlSess.GetOrderedTables(%v) - %v", moduleNm, cvlRetOrdTbl)

			}
			sncTblInfo.DepTbl = make(map[string]depTblData)
			if sncTblInfo.DepTbl == nil {
				log.Warningf("sncTblInfo.DepTbl is nill , no space to store dependency table list for sonic module %v", moduleNm)
				cvl.ValidationSessClose(cvlSess)
				break
			}
			for _, tbl := range sncTblInfo.OrdTbl {
				var cvlRetDepTbl cvl.CVLRetCode
				depTblInfo := depTblData{DepTblWithinMdl: []string{}, DepTblAcrossMdl: []string{}}
				depTblInfo.DepTblWithinMdl, cvlRetDepTbl = cvlSess.GetOrderedDepTables(moduleNm, tbl)
				if cvlRetDepTbl != cvl.CVL_SUCCESS {
					log.Warningf("Failure in cvlSess.GetOrderedDepTables(%v, %v) - %v", moduleNm, tbl, cvlRetDepTbl)
				}
				depTblInfo.DepTblAcrossMdl, cvlRetDepTbl = cvlSess.GetDepTables(moduleNm, tbl)

				if cvlRetDepTbl != cvl.CVL_SUCCESS {
					log.Warningf("Failure in cvlSess.GetDepTables(%v, %v) - %v", moduleNm, tbl, cvlRetDepTbl)
				}
				sncTblInfo.DepTbl[tbl] = depTblInfo
			}
			xDbSpecTblSeqnMap[moduleNm] = sncTblInfo
			cvl.ValidationSessClose(cvlSess)
			if done {
				break
			}
		}

	}

	for childNm, childEntry := range entry.Dir {
		childPath := tableName + "/" + childNm
		dbMapFill(tableName, childPath, moduleNm, xDbSpecMap, childEntry)
		if entry.IsList() && childEntry.IsList() {
			/* If structure is not like current community-sonic yangs with nested lists, that
			   have only key leaves in parent list and only one nested list with only one
			   key and one non-key leaf, then its not supported case so don't traverse the parent list anymore.
			*/
			if _, nestedListOk := xDbSpecMap[tableName+"/"+entry.Name+"/"+childNm]; !nestedListOk {
				return
			}
		}
	}

}

/* Build redis db lookup map */
func dbMapBuild(entries []*yang.Entry) {
	if entries == nil {
		return
	}
	xDbSpecMap = make(map[string]*dbInfo)
	xDbSpecTblSeqnMap = make(map[string]*sonicTblSeqnInfo)
	sonicLeafRefMap = make(map[string][]string)
	xDbRpcSpecMap = make(map[string]string)

	for _, e := range entries {
		if e == nil || len(e.Dir) == 0 {
			continue
		}
		moduleNm := e.Name
		dbMapFill("", "", moduleNm, xDbSpecMap, e)
	}
}

func childToUpdateParent(xpath string, tableName string) {
	var xpathData *yangXpathInfo
	parent := parentXpathGet(xpath)
	if len(parent) == 0 || parent == "/" {
		return
	}

	_, ok := xYangSpecMap[parent]
	if !ok {
		xpathData = new(yangXpathInfo)
		xpathData.dbIndex = db.ConfigDB // default value
		xYangSpecMap[parent] = xpathData
	}

	parentXpathData := xYangSpecMap[parent]
	if !contains(parentXpathData.childTable, tableName) {
		parentXpathData.childTable = append(parentXpathData.childTable, tableName)
	}

	if parentXpathData.yangEntry != nil && parentXpathData.yangType == YANG_LIST &&
		(parentXpathData.tableName != nil || parentXpathData.xfmrTbl != nil) {
		return
	}
	childToUpdateParent(parent, tableName)
}

/* Build lookup map based on yang xpath */
func annotEntryFill(xYangSpecMap map[string]*yangXpathInfo, xpath string, entry *yang.Entry) {
	xpathData := new(yangXpathInfo)

	xpathData.dbIndex = db.ConfigDB // default value
	xpathData.subscribeMinIntvl = XFMR_INVALID
	tableOwnerAnnotated := false
	isTableOwner := true
	/* fill table with yang extension data. */
	if entry != nil && len(entry.Exts) > 0 {
		for _, ext := range entry.Exts {
			dataTagArr := strings.Split(ext.Keyword, ":")
			tagType := dataTagArr[len(dataTagArr)-1]
			switch tagType {
			case "table-name":
				if xpathData.tableName == nil {
					xpathData.tableName = new(string)
				}
				*xpathData.tableName = ext.NName()
				updateDbTableData(xpath, *xpathData.tableName)
			case "key-name":
				if xpathData.keyName == nil {
					xpathData.keyName = new(string)
				}
				*xpathData.keyName = ext.NName()
			case "table-transformer":
				if xpathData.xfmrTbl == nil {
					xpathData.xfmrTbl = new(string)
				}
				*xpathData.xfmrTbl = ext.NName()
			case "field-name":
				xpathData.fieldName = ext.NName()
			case "composite-field-names":
				xpathData.compositeFields = strings.Split(ext.NName(), ",")
			case "subtree-transformer":
				xpathData.xfmrFunc = ext.NName()
			case "key-transformer":
				xpathData.xfmrKey = ext.NName()
			case "key-delimiter":
				xpathData.delim = ext.NName()
			case "field-transformer":
				xpathData.xfmrField = ext.NName()
			case "post-transformer":
				if xYangModSpecMap == nil {
					xYangModSpecMap = make(map[string]*moduleAnnotInfo)
				}
				if _, ok := xYangModSpecMap[xpath]; !ok {
					var modInfo = new(moduleAnnotInfo)
					xYangModSpecMap[xpath] = modInfo
				}
				xYangModSpecMap[xpath].xfmrPost = ext.NName()
			case "pre-transformer":
				if xYangModSpecMap == nil {
					xYangModSpecMap = make(map[string]*moduleAnnotInfo)
				}
				if _, ok := xYangModSpecMap[xpath]; !ok {
					var modInfo = new(moduleAnnotInfo)
					xYangModSpecMap[xpath] = modInfo
				}
				xYangModSpecMap[xpath].xfmrPre = ext.NName()
			case "validate-xfmr":
				xpathData.validateFunc = ext.NName()
			case "rpc-callback":
				xYangRpcSpecMap[xpath] = ext.NName()
				xpathData.yangType = YANG_RPC
			case "path-transformer":
				xpathData.xfmrPath = ext.NName()
			case "use-self-key":
				xpathData.keyXpath = nil
			case "db-name":
				xpathData.dbIndex = db.GetdbNameToIndex(ext.NName())
			case "table-owner":
				tableOwnerAnnotated = true
				if strings.EqualFold(ext.NName(), "False") {
					isTableOwner = false
				}
			case "subscribe-preference":
				if ext.NName() == "sample" {
					xpathData.subscriptionFlags.Set(subsPrefSample)
				}
			case "subscribe-on-change":
				if strings.EqualFold(ext.NName(), "disable") {
					xpathData.subscriptionFlags.Set(subsOnChangeDisable)
				} else if strings.EqualFold(ext.NName(), "enable") {
					xpathData.subscriptionFlags.Set(subsOnChangeEnable)
				} else {
					log.Warningf("Invalid subscribe-on-change value: %v defined in the path %v\r\n", ext.NName(), xpath)
				}
			case "subscribe-min-interval":
				if ext.NName() == "NONE" {
					xpathData.subscribeMinIntvl = 0
				} else {
					minIntvl, err := strconv.Atoi(ext.NName())
					if err != nil {
						log.Warningf("Invalid subscribe min interval time(%v).\r\n", ext.NName())
						return
					}
					xpathData.subscribeMinIntvl = minIntvl
				}
			case "virtual-table":
				if xpathData.virtualTbl == nil {
					xpathData.virtualTbl = new(bool)
					*xpathData.virtualTbl = false
				}
				if strings.EqualFold(ext.NName(), "True") {
					*xpathData.virtualTbl = true
				}
			case "db-key-count":
				var err error
				if xpathData.dbKeyCompCnt, err = strconv.Atoi(ext.NName()); err != nil {
					log.Warningf("Invalid db-key-count value (%v) in the yang path %v.\r\n", ext.NName(), xpath)
					return
				}
			case "subscribe-delete-as-update":
				if strings.EqualFold(ext.NName(), "true") {
					xpathData.subscriptionFlags.Set(subsDelAsUpdate)
				}
			case "data-source":
				if strings.EqualFold(ext.NName(), "dynamic") {
					xpathData.isDataSrcDynamic = new(bool)
					*xpathData.isDataSrcDynamic = true
				}
			}
		}
		/* table owner annotation is valid only when it is annotated with a table-name/table-transformer annotation at the node */
		if tableOwnerAnnotated {
			if xpathData.tableName != nil || xpathData.xfmrTbl != nil {
				if xpathData.tblOwner == nil {
					xpathData.tblOwner = new(bool)
				}
				*xpathData.tblOwner = isTableOwner
			} else {
				log.Warningf("table-owner annotation is found without table annotation at xpath %v.\r\n", xpath)
			}
		}
	}
	xYangSpecMap[xpath] = xpathData
}

/* Build xpath from yang-annotation */
func xpathFromDevCreate(path string) string {
	p := strings.Split(path, "/")
	for i, k := range p {
		if len(k) > 0 {
			p[i] = strings.Split(k, ":")[1]
		}
	}
	return strings.Join(p[1:], "/")
}

/* Build lookup map based on yang xpath */
func annotToDbMapBuild(annotEntries []*yang.Entry) {
	if annotEntries == nil {
		return
	}
	if xYangSpecMap == nil {
		xYangSpecMap = make(map[string]*yangXpathInfo)
	}
	if xYangRpcSpecMap == nil {
		xYangRpcSpecMap = make(map[string]string)
	}

	for _, e := range annotEntries {
		if e != nil && len(e.Deviations) > 0 {
			for _, d := range e.Deviations {
				xpath := xpathFromDevCreate(d.Name)
				xpath = "/" + strings.Replace(e.Name, "-annot", "", -1) + ":" + xpath
				for i, deviate := range d.Deviate {
					if i == 2 {
						for _, ye := range deviate {
							annotEntryFill(xYangSpecMap, xpath, ye)
						}
					}
				}
			}
		}
	}
}

func annotDbSpecMapFill(xDbSpecMap map[string]*dbInfo, dbXpath string, entry *yang.Entry) error {
	var err error
	var dbXpathData *dbInfo
	var ok bool

	pname := strings.Split(dbXpath, "/")
	if len(pname) < 3 {
		// check rpc?
		if entry != nil && len(entry.Exts) > 0 {
			for _, ext := range entry.Exts {
				dataTagArr := strings.Split(ext.Keyword, ":")
				tagType := dataTagArr[len(dataTagArr)-1]
				switch tagType {
				case "rpc-callback":
					xDbRpcSpecMap[dbXpath] = ext.NName()
				default:
				}
			}
		} else {
			log.Warningf("DB Rpc spec-map doesn't contain rpc entry(%v) \r\n", dbXpath)
		}
		return err
	}

	tableName := pname[2]
	// container(redis tablename)
	dbXpathData, ok = xDbSpecMap[tableName]
	if !ok {
		log.Warningf("DB spec-map data not found(%v) \r\n", dbXpath)
		return err
	}

	if dbXpathData.dbIndex >= db.MaxDB {
		dbXpathData.dbIndex = db.ConfigDB // default value
	}

	/* fill table with cvl yang extension data. */
	if entry != nil && len(entry.Exts) > 0 {
		for _, ext := range entry.Exts {
			dataTagArr := strings.Split(ext.Keyword, ":")
			tagType := dataTagArr[len(dataTagArr)-1]
			switch tagType {
			case "key-name":
				if dbXpathData.keyName == nil {
					dbXpathData.keyName = new(string)
				}
				*dbXpathData.keyName = ext.NName()
			case "value-transformer":
				fieldName := pname[len(pname)-1]
				fieldXpath := tableName + "/" + fieldName
				if fldXpathData, ok := xDbSpecMap[fieldXpath]; ok {
					fldXpathData.xfmrValue = new(string)
					*fldXpathData.xfmrValue = ext.NName()
					dbXpathData.hasXfmrFn = true
					if xpathList, ok := sonicLeafRefMap[fieldXpath]; ok {
						for _, curpath := range xpathList {
							if curSpecData, ok := xDbSpecMap[curpath]; ok && curSpecData.xfmrValue == nil {
								curSpecData.xfmrValue = fldXpathData.xfmrValue
								curTableName := strings.Split(curpath, "/")[0]
								if curTblSpecInfo, ok := xDbSpecMap[curTableName]; ok {
									curTblSpecInfo.hasXfmrFn = true
								}
							}
						}
					}
				}
			case "cascade-delete":
				fieldName := pname[len(pname)-1]
				fieldXpath := tableName + "/" + fieldName
				if fldXpathData, ok := xDbSpecMap[fieldXpath]; ok {
					if fldXpathData.isKey {
						if ext.NName() == "ENABLE" || ext.NName() == "enable" {
							dbXpathData.cascadeDel = XFMR_ENABLE
						} else {
							dbXpathData.cascadeDel = XFMR_DISABLE
						}
					} else {
						log.Warningf("cascade-delete annotation is supported for sonic key leaf only. Ignoring the incorrect annotation")
					}
				}
			case "key-transformer":
				listName := pname[SONIC_TBL_CHILD_INDEX]
				listXpath := tableName + "/" + listName
				if listXpathData, ok := xDbSpecMap[listXpath]; ok {
					listXpathData.xfmrKey = ext.NName()
				}

			default:
			}
		}
	}

	return err
}

func annotDbSpecMap(annotEntries []*yang.Entry) {
	if annotEntries == nil || xDbSpecMap == nil {
		return
	}
	for _, e := range annotEntries {
		if e != nil && len(e.Deviations) > 0 {
			for _, d := range e.Deviations {
				xpath := xpathFromDevCreate(d.Name)
				xpath = "/" + strings.Replace(e.Name, "-annot", "", -1) + ":" + xpath
				for i, deviate := range d.Deviate {
					if i == 2 {
						for _, ye := range deviate {
							annotDbSpecMapFill(xDbSpecMap, xpath, ye)
						}
					}
				}
			}
		}
	}
}

/* Debug function to print the yang xpath lookup map */
func mapPrint(fileName string) {
	fp, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer fp.Close()

	fmt.Fprintf(fp, "-------------  Module level Annotations ---------------\r\n")
	for mod, spec := range xYangModSpecMap {
		fmt.Fprintf(fp, "\n%v:\r\n", mod)
		fmt.Fprintf(fp, "\r    preXfmr  : %v", spec.xfmrPre)
		fmt.Fprintf(fp, "\r    postXfmr : %v", spec.xfmrPost)
	}

	fmt.Fprintf(fp, "-------------  RPC Annotations ---------------\r\n")
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	for rpcXpath, rpcFunc := range xYangRpcSpecMap {
		fmt.Fprintf(fp, "\r\n  %v : %v", rpcXpath, rpcFunc)
	}

	var sortedXpath []string
	for k := range xYangSpecMap {
		sortedXpath = append(sortedXpath, k)
	}
	sort.Strings(sortedXpath)
	for _, xpath := range sortedXpath {
		k := xpath
		d := xYangSpecMap[k]

		fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
		fmt.Fprintf(fp, "%v:\r\n", k)
		fmt.Fprintf(fp, "    yangType: %v\r\n", getYangTypeStrId(d.yangType))
		if d.nameWithMod != nil {
			fmt.Fprintf(fp, "    nameWithMod : %v\r\n", *d.nameWithMod)
		}
		fmt.Fprintf(fp, "    hasChildSubTree : %v\r\n", d.hasChildSubTree)
		fmt.Fprintf(fp, "    hasNonTerminalNode : %v\r\n", d.hasNonTerminalNode)
		fmt.Fprintf(fp, "    subscribeOnChg disbale flag: %v\r\n", d.subscriptionFlags.Has(subsOnChangeDisable))
		fmt.Fprintf(fp, "    subscribeOnChg enable flag: %v\r\n", d.subscriptionFlags.Has(subsOnChangeEnable))
		fmt.Fprintf(fp, "    subscribeMinIntvl  : %v\r\n", d.subscribeMinIntvl)
		fmt.Fprintf(fp, "    subscribePref Sample     : %v\r\n", d.subscriptionFlags.Has(subsPrefSample))
		fmt.Fprintf(fp, "    tableName: ")
		if d.tableName != nil {
			fmt.Fprintf(fp, "%v", *d.tableName)
		}
		fmt.Fprintf(fp, "\r\n    tblOwner: ")
		if d.tblOwner != nil {
			fmt.Fprintf(fp, "%v", *d.tblOwner)
		}
		fmt.Fprintf(fp, "\r\n    virtualTbl: ")
		if d.virtualTbl != nil {
			fmt.Fprintf(fp, "%v", *d.virtualTbl)
		}
		fmt.Fprintf(fp, "\r\n    xfmrTbl  : ")
		if d.xfmrTbl != nil {
			fmt.Fprintf(fp, "%v", *d.xfmrTbl)
		}
		fmt.Fprintf(fp, "\r\n    keyName  : ")
		if d.keyName != nil {
			fmt.Fprintf(fp, "%v", *d.keyName)
		}
		fmt.Fprintf(fp, "\r\n    childTbl : %v", d.childTable)
		fmt.Fprintf(fp, "\r\n    FieldName: %v", d.fieldName)
		fmt.Fprintf(fp, "\r\n    defVal   : %v", d.defVal)
		fmt.Fprintf(fp, "\r\n    keyLevel : %v", d.keyLevel)
		fmt.Fprintf(fp, "\r\n    xfmrKeyFn: %v", d.xfmrKey)
		fmt.Fprintf(fp, "\r\n    xfmrFunc : %v", d.xfmrFunc)
		fmt.Fprintf(fp, "\r\n    xfmrField :%v", d.xfmrField)
		fmt.Fprintf(fp, "\r\n    dbIndex  : %v", d.dbIndex)
		fmt.Fprintf(fp, "\r\n    validateFunc  : %v", d.validateFunc)
		fmt.Fprintf(fp, "\r\n    yangEntry: ")
		if d.yangEntry != nil {
			fmt.Fprintf(fp, "%v", *d.yangEntry)
		}
		fmt.Fprintf(fp, "\r\n    keyXpath: %d\r\n", d.keyXpath)
		for i, kd := range d.keyXpath {
			fmt.Fprintf(fp, "        %d. %#v\r\n", i, kd)
		}
		fmt.Fprintf(fp, "\r\n    isKey   : %v\r\n", d.isKey)
		fmt.Fprintf(fp, "\r\n isRefByKey : %v\r\n", d.isRefByKey)
		fmt.Fprintf(fp, "\r\n    operQP  : %v\r\n", d.operationalQP)
		fmt.Fprintf(fp, "\r\n    hasChildOperQP  : %v\r\n", d.hasChildOpertnlNd)
		fmt.Fprintf(fp, "\r\n    isDataSrcDynamic: ")
		if d.isDataSrcDynamic != nil {
			fmt.Fprintf(fp, "%v", *d.isDataSrcDynamic)
		}
	}
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")

}

/* Debug function to print redis db lookup map */
func dbMapPrint(fname string) {
	fp, err := os.Create(fname)
	if err != nil {
		return
	}
	defer fp.Close()
	fmt.Fprintf(fp, "-------------  RPC Annotations ---------------\r\n")
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	for rpcXpath, rpcFunc := range xDbRpcSpecMap {
		fmt.Fprintf(fp, "\r\n  %v : %v", rpcXpath, rpcFunc)
	}

	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	for k, v := range xDbSpecMap {
		fmt.Fprintf(fp, "     field:%v: \r\n", k)
		fmt.Fprintf(fp, "     type     :%v \r\n", getYangTypeStrId(v.yangType))
		fmt.Fprintf(fp, " isKey :%v \r\n", v.isKey)
		fmt.Fprintf(fp, "     db-type  :%v \r\n", v.dbIndex)
		fmt.Fprintf(fp, "     hasXfmrFn:%v \r\n", v.hasXfmrFn)
		fmt.Fprintf(fp, "     module   :%v \r\n", v.module)
		fmt.Fprintf(fp, "     listName :%v \r\n", v.listName)
		fmt.Fprintf(fp, "     keyList  :%v \r\n", v.keyList)
		fmt.Fprintf(fp, "     xfmrKey  :%v \r\n", v.xfmrKey)
		fmt.Fprintf(fp, "     cascadeDel :%v \r\n", v.cascadeDel)
		if v.xfmrValue != nil {
			fmt.Fprintf(fp, "     xfmrValue:%v \r\n", *v.xfmrValue)
		}
		fmt.Fprintf(fp, "     leafRefPath:%v \r\n", v.leafRefPath)
		fmt.Fprintf(fp, "     KeyName: ")
		if v.keyName != nil {
			fmt.Fprintf(fp, "%v\r\n", *v.keyName)
		}
		for _, yxpath := range v.yangXpath {
			fmt.Fprintf(fp, "\r\n     oc-yang  :%v ", yxpath)
		}
		fmt.Fprintf(fp, "\r\n     cvl-yang :%v \r\n", v.dbEntry)
		fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")

	}
}

func xDbSpecTblSeqnMapPrint(fname string) {
	fp, err := os.Create(fname)
	if err != nil {
		return
	}
	defer fp.Close()
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	if xDbSpecTblSeqnMap == nil {
		return
	}
	for mdlNm, mdlTblSeqnDt := range xDbSpecTblSeqnMap {
		fmt.Fprintf(fp, "%v : { \r\n", mdlNm)
		if mdlTblSeqnDt == nil {
			continue
		}
		fmt.Fprintf(fp, "        OrderedTableList : %v\r\n", mdlTblSeqnDt.OrdTbl)
		fmt.Fprintf(fp, "        Dependent table list  : {\r\n")
		if mdlTblSeqnDt.DepTbl == nil {
			fmt.Fprintf(fp, "                        }\r\n")
			fmt.Fprintf(fp, "}\r\n")
			continue
		}
		for tblNm, DepTblInfo := range mdlTblSeqnDt.DepTbl {
			fmt.Fprintf(fp, "                                        %v : Within module  : %v\r\n", tblNm, DepTblInfo.DepTblWithinMdl)
			tblNmSpc := " "
			for cnt := 0; cnt < len(tblNm)-1; cnt++ {
				tblNmSpc = tblNmSpc + " "
			}
			fmt.Fprintf(fp, "                                        %v : Across modules : %v\r\n", tblNmSpc, DepTblInfo.DepTblAcrossMdl)
		}
		fmt.Fprintf(fp, "                                }\r\n")
		fmt.Fprintf(fp, "}\r\n")
	}
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	fmt.Fprintf(fp, "OrderedTableList from json: \r\n")

	for tbl, tlist := range sonicOrdTblListMap {
		fmt.Fprintf(fp, "    %v : %v\r\n", tbl, tlist)
	}
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")

}

func sonicYangNestedListValidateElements(tableName string, entry *yang.Entry) error {
	/* All current sommunity sonic yangs have only one key and one non-key leaf in the nested list
	   to support dynamic field-name and value in DB table.The key-leaf becomes dynamic field-name
	   and the non-key-leaf becomes the value of the dynamic field.If the nested list does not conform
	   to this structure do not load it and even its parent list.
	*/
	if (len(strings.Fields(entry.Key)) == 1) && (len(entry.Dir) == 2) {
		return nil
	}

	errStr := fmt.Sprintf("Sonic yang nested list %v with more than one key or non-key leaf not supported.", tableName+"/"+entry.Parent.Name+"/"+entry.Name)
	log.Warningf(errStr)
	return fmt.Errorf("%v", errStr)
}

func cleanupNestedListSpecInfo(tableName string, parentEntry *yang.Entry) {
	for childNm := range parentEntry.Parent.Dir {
		delete(xDbSpecMap, tableName+"/"+childNm)
	}
	delete(xDbSpecMap, tableName+"/"+parentEntry.Name)
}
