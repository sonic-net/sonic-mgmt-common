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
    "strings"
    log "github.com/golang/glog"
    "github.com/Azure/sonic-mgmt-common/cvl"
    "github.com/Azure/sonic-mgmt-common/translib/db"
    "strconv"
    "github.com/openconfig/goyang/pkg/yang"
)

/* Data needed to construct lookup table from yang */
type yangXpathInfo  struct {
    yangDataType   string
    tableName      *string
    xfmrTbl        *string
    childTable      []string
    dbEntry        *yang.Entry
    yangEntry      *yang.Entry
    keyXpath       map[int]*[]string
    delim          string
    fieldName      string
    xfmrFunc       string
    xfmrField      string
    xfmrPost       string
    validateFunc   string
    rpcFunc        string
    xfmrKey        string
    keyName        *string
    dbIndex        db.DBNum
    keyLevel       int
    isKey          bool
    defVal         string
    tblOwner       *bool
    hasChildSubTree bool
    hasNonTerminalNode bool
    subscribePref      *string
    subscribeOnChg     int
    subscribeMinIntvl  int
    cascadeDel     int
    virtualTbl     *bool
    nameWithMod    *string
	xfmrPre        string
}

type dbInfo  struct {
    dbIndex      db.DBNum
    keyName      *string
    fieldType    string
    rpcFunc      string
    dbEntry      *yang.Entry
    yangXpath    []string
    module       string
    delim        string
	leafRefPath  []string
    listName     []string
    keyList      []string
    xfmrValue    *string
    hasXfmrFn    bool
    cascadeDel   int
}

type sonicTblSeqnInfo struct {
       OrdTbl []string
       DepTbl map[string][]string
}

type mdlInfo struct {
       Org string
       Ver string
}

var xYangSpecMap  map[string]*yangXpathInfo
var xDbSpecMap    map[string]*dbInfo
var xDbSpecOrdTblMap map[string][]string //map of module-name to ordered list of db tables { "sonic-acl" : ["ACL_TABLE", "ACL_RULE"] }
var xDbSpecTblSeqnMap  map[string]*sonicTblSeqnInfo
var xMdlCpbltMap       map[string]*mdlInfo
var sonicOrdTblListMap map[string][]string
var sonicLeafRefMap    map[string][]string

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
	if ((!ok) || (mdlInfoEntry == nil)) {
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

/* update transformer spec with db-node */
func updateDbTableData (xpath string, xpathData *yangXpathInfo, tableName string) {
	if len(tableName) == 0 {
		return
	}
	_, ok := xDbSpecMap[tableName]
	if ok {
		xDbSpecMap[tableName].yangXpath = append(xDbSpecMap[tableName].yangXpath, xpath)
		xpathData.dbEntry = xDbSpecMap[tableName].dbEntry
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

/* Recursive api to fill the map with yang details */
func yangToDbMapFill (keyLevel int, xYangSpecMap map[string]*yangXpathInfo, entry *yang.Entry, xpathPrefix string, xpathFull string) {
	xpath := ""
	curKeyLevel  := 0
	curXpathFull := ""

	if entry != nil && (entry.Kind == yang.CaseEntry || entry.Kind == yang.ChoiceEntry) {
		curXpathFull = xpathFull + "/" + entry.Name
		if _, ok := xYangSpecMap[xpathPrefix]; ok {
			curKeyLevel = xYangSpecMap[xpathPrefix].keyLevel
		}
		curXpathData, ok := xYangSpecMap[curXpathFull]
		if !ok {
			curXpathData = new(yangXpathInfo)
			curXpathData.dbIndex = db.ConfigDB // default value
			xYangSpecMap[curXpathFull] = curXpathData
		}
		curXpathData.yangDataType = strings.ToLower(yang.EntryKindToName[entry.Kind])
		curXpathData.yangEntry    = entry
		if xYangSpecMap[xpathPrefix].subscribePref != nil {
			curXpathData.subscribePref = xYangSpecMap[xpathPrefix].subscribePref
		}
		curXpathData.subscribeOnChg    = xYangSpecMap[xpathPrefix].subscribeOnChg
		curXpathData.subscribeMinIntvl = xYangSpecMap[xpathPrefix].subscribeMinIntvl
		curXpathData.cascadeDel   = xYangSpecMap[xpathPrefix].cascadeDel
		xpath = xpathPrefix
	} else {
	/* create the yang xpath */
	if xYangSpecMap[xpathPrefix] != nil  && xYangSpecMap[xpathPrefix].yangDataType == "module" {
		/* module name is separated from the rest of xpath with ":" */
		xpath = xpathPrefix + ":" + entry.Name
	} else {
		xpath = xpathPrefix + "/" + entry.Name
	}

	updateChoiceCaseXpath := false
	curXpathFull = xpath
	if xpathPrefix != xpathFull {
		curXpathFull = xpathFull + "/" + entry.Name
		if annotNode, ok := xYangSpecMap[curXpathFull]; ok {
			xpathData := new(yangXpathInfo)
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
		xpathData.subscribeOnChg    = XFMR_INVALID
		xpathData.subscribeMinIntvl = XFMR_INVALID
		xpathData.cascadeDel = XFMR_INVALID
	} else {
		if len(xpathData.xfmrFunc) > 0 {
			childSubTreePresenceFlagSet(xpath)
		}
	}

	xpathData.yangDataType = entry.Node.Statement().Keyword
	if (xpathData.tableName != nil && *xpathData.tableName != "") {
		childToUpdateParent(xpath, *xpathData.tableName)
	}

	parentXpathData, ok := xYangSpecMap[xpathPrefix]
	/* init current xpath table data with its parent data, change only if needed. */
	if ok && xpathData.tableName == nil {
		if xpathData.tableName == nil && parentXpathData.tableName != nil && xpathData.xfmrTbl == nil {
			xpathData.tableName = parentXpathData.tableName
		} else if xpathData.xfmrTbl == nil && parentXpathData.xfmrTbl != nil {
			xpathData.xfmrTbl = parentXpathData.xfmrTbl
		}
	}

	if ok && xpathData.dbIndex == db.ConfigDB && parentXpathData.dbIndex != db.ConfigDB {
		// If DB Index is not annotated and parent DB index is annotated inherit the DB Index of the parent
		xpathData.dbIndex = parentXpathData.dbIndex
	}

	if ok && len(parentXpathData.validateFunc) > 0 {
		xpathData.validateFunc = parentXpathData.validateFunc
	}

	if ok && len(parentXpathData.xfmrFunc) > 0 && len(xpathData.xfmrFunc) == 0 {
		xpathData.xfmrFunc = parentXpathData.xfmrFunc
	}

   if ok && (parentXpathData.subscribeMinIntvl == XFMR_INVALID ||
      parentXpathData.subscribeOnChg == XFMR_INVALID) {
       log.Warningf("Susbscribe MinInterval/OnChange flag is set to invalid for(%v) \r\n", xpathPrefix)
       return
   }

   if ok {
	   if xpathData.subscribeOnChg == XFMR_INVALID {
		   xpathData.subscribeOnChg = parentXpathData.subscribeOnChg
	   }

	   if xpathData.subscribeMinIntvl == XFMR_INVALID {
		   xpathData.subscribeMinIntvl = parentXpathData.subscribeMinIntvl
	   }

	   if xpathData.subscribePref == nil && parentXpathData.subscribePref != nil {
		   xpathData.subscribePref = parentXpathData.subscribePref
	   }

	   if xpathData.subscribePref != nil && *xpathData.subscribePref == "NONE" {
		   xpathData.subscribePref = nil
	   }

		if parentXpathData.cascadeDel == XFMR_INVALID {
			/* should not hit this case */
			log.Warningf("Cascade-delete flag is set to invalid for(%v) \r\n", xpathPrefix)
			return
		}

		if xpathData.cascadeDel == XFMR_INVALID && xpathData.dbIndex == db.ConfigDB{
			xpathData.cascadeDel = parentXpathData.cascadeDel
		}

		if entry.Prefix != nil && entry.Prefix.Parent != nil   &&
		   entry.Prefix.Parent.Statement().Keyword == "module" &&
		   parentXpathData.yangEntry.Prefix != nil && parentXpathData.yangEntry.Prefix.Parent != nil {

			if (len(parentXpathData.yangEntry.Prefix.Parent.NName()) > 0) &&
			   (len(parentXpathData.yangEntry.Prefix.Parent.Statement().Keyword) > 0) &&
			   (parentXpathData.yangEntry.Prefix.Parent.NName() != entry.Prefix.Parent.NName()) {
					xpathData.nameWithMod = new(string)
					*xpathData.nameWithMod = entry.Prefix.Parent.NName() + ":" + entry.Name
			}

		}
	}

	if ((xpathData.yangDataType ==  YANG_LEAF || xpathData.yangDataType == YANG_LEAF_LIST) && (len(xpathData.fieldName) == 0)) {

		if len(xpathData.xfmrField) != 0 {
			xpathData.xfmrFunc = ""
		}
		if xpathData.tableName != nil && xDbSpecMap[*xpathData.tableName] != nil {
			if _, ok := xDbSpecMap[*xpathData.tableName + "/" + entry.Name]; ok {
				xpathData.fieldName = entry.Name
			} else {
				if _, ok := xDbSpecMap[*xpathData.tableName + "/" + strings.ToUpper(entry.Name)]; ok {
					xpathData.fieldName = strings.ToUpper(entry.Name)
				}
				if _, ok := xDbSpecMap[*xpathData.tableName + "/" + entry.Name]; ok {
					xpathData.fieldName = entry.Name
				}
			}
		} else if xpathData.xfmrTbl != nil {
			/* table transformer present */
			xpathData.fieldName = entry.Name
		}
	}
	if xpathData.yangDataType == YANG_LEAF && len(entry.Default) > 0 {
		xpathData.defVal = entry.Default
	}

	if (xpathData.yangDataType == YANG_LEAF || xpathData.yangDataType == YANG_LEAF_LIST) && len(xpathData.fieldName) > 0 && xpathData.tableName != nil {
		dbPath := *xpathData.tableName + "/" + xpathData.fieldName
		if xDbSpecMap[dbPath] != nil {
			xDbSpecMap[dbPath].yangXpath = append(xDbSpecMap[dbPath].yangXpath, xpath)
		}
	}

	/* fill table with key data. */
	curKeyLevel := keyLevel
	if len(entry.Key) != 0 {
		parentKeyLen := 0

		/* create list with current keys */
		keyXpath        := make([]string, len(strings.Split(entry.Key, " ")))
		for id, keyName := range(strings.Split(entry.Key, " ")) {
			keyXpath[id] = xpath + "/" + keyName
			if _, ok := xYangSpecMap[xpath + "/" + keyName]; !ok {
				keyXpathData := new(yangXpathInfo)
				keyXpathData.dbIndex = db.ConfigDB // default value
				xYangSpecMap[xpath + "/" + keyName] = keyXpathData
			}
			xYangSpecMap[xpath + "/" + keyName].isKey = true
		}

		xpathData.keyXpath = make(map[int]*[]string, (parentKeyLen + 1))
		k := 0
		for ; k < parentKeyLen; k++ {
			/* copy parent key-list to child key-list*/
			xpathData.keyXpath[k] = parentXpathData.keyXpath[k]
		}
		xpathData.keyXpath[k] = &keyXpath
		xpathData.keyLevel    = curKeyLevel
		curKeyLevel++
	} else if parentXpathData != nil && parentXpathData.keyXpath != nil {
		xpathData.keyXpath = parentXpathData.keyXpath
	}
	xpathData.yangEntry = entry

	if xpathData.subscribeMinIntvl == XFMR_INVALID {
		xpathData.subscribeMinIntvl = 0
	}

	if xpathData.subscribeOnChg == XFMR_INVALID {
		xpathData.subscribeOnChg = XFMR_ENABLE
	}
	if ((xpathData.subscribePref != nil) && (*xpathData.subscribePref == "onchange") && (xpathData.subscribeOnChg == XFMR_DISABLE)) {
		log.Infof("subscribe OnChange is disabled so setting subscribe preference to default/sample from onchange for xpath - %v", xpath)
		xpathData.subscribePref = nil
	}
	if xpathData.cascadeDel == XFMR_INVALID {
		/* set to  default value */
		xpathData.cascadeDel = XFMR_DISABLE
	}

	if updateChoiceCaseXpath {
		copyYangXpathSpecData(xYangSpecMap[curXpathFull], xYangSpecMap[xpath])
	}

	if xpathData.yangDataType == YANG_CONTAINER || xpathData.yangDataType == YANG_LIST {
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

/* Build lookup table based of yang xpath */
func yangToDbMapBuild(entries map[string]*yang.Entry) {
    if entries == nil {
        return
    }

    if xYangSpecMap == nil {
        xYangSpecMap = make(map[string]*yangXpathInfo)
    }

    for module, e := range entries {
        if e == nil || len(e.Dir) == 0 {
            continue
        }

	/* Start to fill xpath based map with yang data */
    keyLevel := 0
    yangToDbMapFill(keyLevel, xYangSpecMap, e, "", "")

	// Fill the ordered map of child tables list for oc yangs
	updateSchemaOrderedMap(module, e)
    }

	//sonicOrdTblListMap = make(map[string][]string)
	//jsonfile := YangPath + TblInfoJsonFile
	//xlateJsonTblInfoLoad(sonicOrdTblListMap, jsonfile)

	mapPrint(xYangSpecMap, "/tmp/fullSpec.txt")
	dbMapPrint("/tmp/dbSpecMap.txt")
	xDbSpecTblSeqnMapPrint("/tmp/dbSpecTblSeqnMap.txt")
	sonicLeafRefDataPrint("/tmp/sonicLeafRef.log")
	sonicLeafRefMap = nil
}

func dbSpecXpathGet(inPath string) (string, error){
	/* This api currently handles only cointainer inside a list for sonic-yang.
	   Should be enhanced to support nested list in future. */
	var err error
	specPath   := ""

	xpath, _, err := XfmrRemoveXPATHPredicates(inPath)
	if err != nil {
		log.Warningf("xpath conversion failed for(%v) \r\n", inPath)
		return specPath, err
	}

	pathList   := strings.Split(xpath, "/")
	if len(pathList) < 3 {
		log.Warningf("Leaf-ref path not valid(%v) \r\n", inPath)
		return specPath, err
	}

	tableName := pathList[2]
	fieldName := pathList[len(pathList)-1]
	specPath = tableName+"/"+fieldName
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
                actDbSpecPath = tableName+"/"+fieldName
        }
        return actDbSpecPath
}

/* Fill the map with db details */
func dbMapFill(tableName string, curPath string, moduleNm string, xDbSpecMap map[string]*dbInfo, entry *yang.Entry) {
	if entry == nil {
		return
	}

	entryType := entry.Node.Statement().Keyword

	if entry.Name != moduleNm {
		if entryType == "container" || entryType == "rpc" {
			tableName = entry.Name
		}
			dbXpath := tableName
			if entryType != "container" && entryType != "rpc" {
				dbXpath = tableName + "/" + entry.Name
			}
			xDbSpecMap[dbXpath] = new(dbInfo)
			xDbSpecMap[dbXpath].dbIndex   = db.MaxDB
			xDbSpecMap[dbXpath].dbEntry   = entry
			xDbSpecMap[dbXpath].fieldType = entryType
			xDbSpecMap[dbXpath].module = moduleNm
			xDbSpecMap[dbXpath].cascadeDel = XFMR_INVALID
			if entryType == "container" {
				xDbSpecMap[dbXpath].dbIndex = db.ConfigDB
				if entry.Exts != nil && len(entry.Exts) > 0 {
					for _, ext := range entry.Exts {
						dataTagArr := strings.Split(ext.Keyword, ":")
						tagType := dataTagArr[len(dataTagArr)-1]
						switch tagType {
						case "key-name" :
							if xDbSpecMap[dbXpath].keyName == nil {
								xDbSpecMap[dbXpath].keyName = new(string)
							}
							*xDbSpecMap[dbXpath].keyName = ext.NName()
						case "rpc-callback" :
							xDbSpecMap[dbXpath].rpcFunc = ext.NName()
						case "db-name" :
							xDbSpecMap[dbXpath].dbIndex = db.GetdbNameToIndex(ext.NName())
						case "key-delim" :
							xDbSpecMap[dbXpath].delim   = ext.NName()
						default :
							log.Infof("Unsupported ext type(%v) for xpath(%v).", tagType, dbXpath)
						}
					}
				}
			} else if tblSpecInfo, ok := xDbSpecMap[tableName]; ok && (entryType == YANG_LIST && len(entry.Key) != 0) {
				tblSpecInfo.listName = append(tblSpecInfo.listName, entry.Name)
				xDbSpecMap[dbXpath].keyList = append(xDbSpecMap[dbXpath].keyList, strings.Split(entry.Key, " ")...)
			} else if entryType == YANG_LEAF || entryType == YANG_LEAF_LIST {
				if entry.Type.Kind == yang.Yleafref {
                                        var lerr error
                                        lrefpath := entry.Type.Path
                                        if (strings.Contains(lrefpath, "..")) {
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
                                                        if (strings.Contains(lrefpath, "..")) {
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
		xDbSpecMap[moduleXpath].dbEntry   = entry
		xDbSpecMap[moduleXpath].fieldType = entryType
		xDbSpecMap[moduleXpath].module = moduleNm
		xDbSpecMap[moduleXpath].cascadeDel = XFMR_INVALID
		for {
			done := true
			sncTblInfo := new(sonicTblSeqnInfo)
			if sncTblInfo == nil {
				log.Warningf("Memory allocation failure for storing Tbl order and dependency info for sonic module %v", moduleNm)
				break
			}
			cvlSess, cvlRetSess := cvl.ValidationSessOpen()
			if cvlRetSess != cvl.CVL_SUCCESS {
				log.Warningf("Failure in creating CVL validation session object required to use CVl API to get Tbl info for module %v - %v", moduleNm, cvlRetSess)
				break
			}
			var cvlRetOrdTbl cvl.CVLRetCode
			sncTblInfo.OrdTbl, cvlRetOrdTbl = cvlSess.GetOrderedTables(moduleNm)
			if cvlRetOrdTbl != cvl.CVL_SUCCESS {
				log.Warningf("Failure in cvlSess.GetOrderedTables(%v) - %v", moduleNm, cvlRetOrdTbl)

			}
			sncTblInfo.DepTbl = make(map[string][]string)
			if sncTblInfo.DepTbl == nil {
				log.Warningf("sncTblInfo.DepTbl is nill , no space to store dependency table list for sonic module %v", moduleNm)
				cvl.ValidationSessClose(cvlSess)
				break
			}
			for _, tbl := range(sncTblInfo.OrdTbl) {
				var cvlRetDepTbl cvl.CVLRetCode
				sncTblInfo.DepTbl[tbl], cvlRetDepTbl = cvlSess.GetDepTables(moduleNm, tbl)
				if cvlRetDepTbl != cvl.CVL_SUCCESS {
					log.Warningf("Failure in cvlSess.GetDepTables(%v, %v) - %v", moduleNm, tbl, cvlRetDepTbl)
				}


			}
			xDbSpecTblSeqnMap[moduleNm] = sncTblInfo
			cvl.ValidationSessClose(cvlSess)
			if done {
				break
			}
		}

	}

	var childList []string
	childList = append(childList, entry.DirOKeys...)

	for _, child := range childList {
		childPath := tableName + "/" + entry.Dir[child].Name
		dbMapFill(tableName, childPath, moduleNm, xDbSpecMap, entry.Dir[child])
	}
}

/* Build redis db lookup map */
func dbMapBuild(entries []*yang.Entry) {
	if entries == nil {
		return
	}
	xDbSpecMap = make(map[string]*dbInfo)
	xDbSpecOrdTblMap = make(map[string][]string)
	xDbSpecTblSeqnMap =  make(map[string]*sonicTblSeqnInfo)
	sonicLeafRefMap   = make(map[string][]string)

	for _, e := range entries {
		if e == nil || len(e.Dir) == 0 {
			continue
		}
		moduleNm := e.Name
		dbMapFill("", "", moduleNm, xDbSpecMap, e)
	}
}

func childToUpdateParent( xpath string, tableName string) {
	var xpathData *yangXpathInfo
	parent := parentXpathGet(xpath)
	if len(parent) == 0  || parent == "/" {
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

       if parentXpathData.yangEntry != nil && parentXpathData.yangEntry.Node.Statement().Keyword == "list" &&
       (parentXpathData.tableName != nil || parentXpathData.xfmrTbl != nil) {
		return
	}
	childToUpdateParent(parent, tableName)
}

/* Build lookup map based on yang xpath */
func annotEntryFill(xYangSpecMap map[string]*yangXpathInfo, xpath string, entry *yang.Entry) {
	xpathData := new(yangXpathInfo)

	xpathData.dbIndex = db.ConfigDB // default value
	xpathData.subscribeOnChg    = XFMR_INVALID
	xpathData.subscribeMinIntvl = XFMR_INVALID
	xpathData.cascadeDel = XFMR_INVALID
	/* fill table with yang extension data. */
	if entry != nil && len(entry.Exts) > 0 {
		for _, ext := range entry.Exts {
			dataTagArr := strings.Split(ext.Keyword, ":")
			tagType := dataTagArr[len(dataTagArr)-1]
			switch tagType {
			case "table-name" :
				if xpathData.tableName == nil {
					xpathData.tableName = new(string)
				}
				*xpathData.tableName = ext.NName()
				updateDbTableData(xpath, xpathData, *xpathData.tableName)
			case "key-name" :
				if xpathData.keyName == nil {
					xpathData.keyName = new(string)
				}
				*xpathData.keyName = ext.NName()
			case "table-transformer" :
				if xpathData.xfmrTbl == nil {
					xpathData.xfmrTbl = new(string)
				}
				*xpathData.xfmrTbl  = ext.NName()
			case "field-name" :
				xpathData.fieldName = ext.NName()
			case "subtree-transformer" :
				xpathData.xfmrFunc  = ext.NName()
			case "key-transformer" :
				xpathData.xfmrKey   = ext.NName()
			case "key-delimiter" :
				xpathData.delim     = ext.NName()
			case "field-transformer" :
				xpathData.xfmrField  = ext.NName()
			case "post-transformer" :
				xpathData.xfmrPost  = ext.NName()
			case "pre-transformer" :
				xpathData.xfmrPre  = ext.NName()
			case "get-validate" :
				xpathData.validateFunc  = ext.NName()
			case "rpc-callback" :
				xpathData.rpcFunc  = ext.NName()
			case "use-self-key" :
				xpathData.keyXpath  = nil
			case "db-name" :
				xpathData.dbIndex = db.GetdbNameToIndex(ext.NName())
			case "table-owner" :
				if xpathData.tblOwner == nil {
					xpathData.tblOwner  = new(bool)
					*xpathData.tblOwner = true
				}
				if strings.EqualFold(ext.NName(), "False") {
					*xpathData.tblOwner = false
				}
			case "subscribe-preference" :
				if xpathData.subscribePref == nil {
					xpathData.subscribePref = new(string)
				}
				*xpathData.subscribePref = ext.NName()
			case "subscribe-on-change" :
				if ext.NName() == "enable" || ext.NName() == "ENABLE" {
					xpathData.subscribeOnChg = XFMR_ENABLE
				} else {
					xpathData.subscribeOnChg = XFMR_DISABLE
				}
			case "subscribe-min-interval" :
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
			case "cascade-delete" :
				if ext.NName() == "ENABLE" ||  ext.NName() == "enable" {
					xpathData.cascadeDel = XFMR_ENABLE
				} else {
					xpathData.cascadeDel = XFMR_DISABLE
				}
			case "virtual-table" :
				if xpathData.virtualTbl == nil {
					xpathData.virtualTbl  = new(bool)
					*xpathData.virtualTbl = false
				}
				if strings.EqualFold(ext.NName(), "True") {
					*xpathData.virtualTbl = true
				}
			}
		}
	}
	xYangSpecMap[xpath] = xpathData
}

/* Build xpath from yang-annotation */
func xpathFromDevCreate(path string) string {
	p := strings.Split(path, "/")
	for i, k := range p {
		if len(k) > 0 { p[i] = strings.Split(k, ":")[1] }
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
    mapPrint(xYangSpecMap, "/tmp/annotSpec.txt")
}

func annotDbSpecMapFill(xDbSpecMap map[string]*dbInfo, dbXpath string, entry *yang.Entry) error {
	var err error
	var dbXpathData *dbInfo
	var ok bool

	pname := strings.Split(dbXpath, "/")
	if len(pname) < 3 {
		// check rpc?
		rpcName := strings.Split(pname[1], ":")
		if len(rpcName) < 2 {
			log.Warningf("DB spec-map data not found(%v) \r\n", rpcName)
			return err
		}
		dbXpathData, ok = xDbSpecMap[rpcName[1]]
		if ok && dbXpathData.fieldType == "rpc" {
			if entry != nil && len(entry.Exts) > 0 {
				for _, ext := range entry.Exts {
					dataTagArr := strings.Split(ext.Keyword, ":")
					tagType := dataTagArr[len(dataTagArr)-1]
					switch tagType {
					case "rpc-callback" :
						dbXpathData.rpcFunc = ext.NName()
					default :
					}
				}
			}
			return err
		} else {
			log.Warningf("DB spec-map data not found(%v) \r\n", dbXpath)
			return err
		}
	}

	tableName  := pname[2]
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
			case "key-name" :
				if dbXpathData.keyName == nil {
					dbXpathData.keyName = new(string)
				}
				*dbXpathData.keyName = ext.NName()
			case "db-name" :
				dbXpathData.dbIndex  = db.GetdbNameToIndex(ext.NName())
			case "value-transformer" :
				fieldName  := pname[len(pname) - 1]
				fieldXpath := tableName+"/"+fieldName
				if fldXpathData, ok := xDbSpecMap[fieldXpath]; ok {
					fldXpathData.xfmrValue  = new(string)
					*fldXpathData.xfmrValue = ext.NName()
					dbXpathData.hasXfmrFn   = true
					if xpathList, ok := sonicLeafRefMap[fieldXpath]; ok {
						for _, curpath := range(xpathList) {
							if curSpecData, ok := xDbSpecMap[curpath]; ok  && curSpecData.xfmrValue == nil {
								curSpecData.xfmrValue = fldXpathData.xfmrValue
								curTableName := strings.Split(curpath, "/")[0]
								if curTblSpecInfo, ok := xDbSpecMap[curTableName]; ok {
									curTblSpecInfo.hasXfmrFn = true
								}
							}
						}
					}
				}
			case "cascade-delete" :
				if ext.NName() == "ENABLE" ||  ext.NName() == "enable" {
					dbXpathData.cascadeDel = XFMR_ENABLE
				} else {
					dbXpathData.cascadeDel = XFMR_DISABLE
				}
			default :
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
	dbMapPrint("/tmp/dbSpecMapFull.txt")
}

/* Debug function to print the yang xpath lookup map */
func mapPrint(inMap map[string]*yangXpathInfo, fileName string) {
    fp, err := os.Create(fileName)
    if err != nil {
        return
    }
    defer fp.Close()

    for k, d := range inMap {
        fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
        fmt.Fprintf(fp, "%v:\r\n", k)
        fmt.Fprintf(fp, "    yangDataType: %v\r\n", d.yangDataType)
        if d.nameWithMod != nil {
            fmt.Fprintf(fp, "    nameWithMod : %v\r\n", *d.nameWithMod)
        }
		fmt.Fprintf(fp, "    cascadeDel  : %v\r\n", d.cascadeDel)
        fmt.Fprintf(fp, "    hasChildSubTree : %v\r\n", d.hasChildSubTree)
        fmt.Fprintf(fp, "    hasNonTerminalNode : %v\r\n", d.hasNonTerminalNode)
		fmt.Fprintf(fp, "    subscribeOnChg     : %v\r\n", d.subscribeOnChg)
		fmt.Fprintf(fp, "    subscribeMinIntvl  : %v\r\n", d.subscribeMinIntvl)
		if d.subscribePref != nil {
			fmt.Fprintf(fp, "    subscribePref      : %v\r\n", *d.subscribePref)
		}
        fmt.Fprintf(fp, "    hasChildSubTree: %v\r\n", d.hasChildSubTree)
        fmt.Fprintf(fp, "    tableName: ")
        if d.tableName != nil {
            fmt.Fprintf(fp, "%v", *d.tableName)
        }
	fmt.Fprintf(fp, "\r\n    postXfmr : %v", d.xfmrPost)
        fmt.Fprintf(fp, "\r\n    tblOwner: ")
        if d.tblOwner != nil {
            fmt.Fprintf(fp, "%v", *d.tblOwner)
        }
        fmt.Fprintf(fp, "\r\n    virtualTbl: ")
		if d.virtualTbl != nil {
			fmt.Fprintf(fp, "%v", *d.virtualTbl)
        }
        fmt.Fprintf(fp, "\r\n    preXfmr  : %v", d.xfmrPre)
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
        fmt.Fprintf(fp, "\r\n    rpcFunc  : %v", d.rpcFunc)
        fmt.Fprintf(fp, "\r\n    yangEntry: ")
        if d.yangEntry != nil {
            fmt.Fprintf(fp, "%v", *d.yangEntry)
        }
        fmt.Fprintf(fp, "\r\n    dbEntry: ")
        if d.dbEntry != nil {
            fmt.Fprintf(fp, "%v", *d.dbEntry)
        }
        fmt.Fprintf(fp, "\r\n    keyXpath: %d\r\n", d.keyXpath)
        for i, kd := range d.keyXpath {
            fmt.Fprintf(fp, "        %d. %#v\r\n", i, kd)
        }
        fmt.Fprintf(fp, "\r\n    isKey   : %v\r\n", d.isKey)
    }
    fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")

}

/* Debug function to print redis db lookup map */
func dbMapPrint( fname string) {
    fp, err := os.Create(fname)
    if err != nil {
        return
    }
    defer fp.Close()
	fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
    for k, v := range xDbSpecMap {
		fmt.Fprintf(fp, "     field:%v: \r\n", k)
        fmt.Fprintf(fp, "     type     :%v \r\n", v.fieldType)
        fmt.Fprintf(fp, "     db-type  :%v \r\n", v.dbIndex)
        fmt.Fprintf(fp, "     rpcFunc  :%v \r\n", v.rpcFunc)
        fmt.Fprintf(fp, "     hasXfmrFn:%v \r\n", v.hasXfmrFn)
        fmt.Fprintf(fp, "     module   :%v \r\n", v.module)
        fmt.Fprintf(fp, "     listName :%v \r\n", v.listName)
        fmt.Fprintf(fp, "     keyList  :%v \r\n", v.keyList)
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
        fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")

    }
}

func xDbSpecTblSeqnMapPrint(fname string) {
        fp, err := os.Create(fname)
        if err != nil {
                return
        }
        defer fp.Close()
        fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
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
                for tblNm, DepTblLst := range mdlTblSeqnDt.DepTbl {
                        fmt.Fprintf(fp, "                                        %v : %v\r\n", tblNm, DepTblLst)
                }
                fmt.Fprintf(fp, "                                }\r\n")
                fmt.Fprintf(fp, "}\r\n")
			}
			fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
			fmt.Fprintf (fp, "OrderedTableList from json: \r\n")

			for tbl, tlist := range sonicOrdTblListMap {
				fmt.Fprintf(fp, "    %v : %v\r\n", tbl, tlist)
			}
			fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")

}

func sonicLeafRefDataPrint(fname string) {
	fp, err := os.Create(fname)
	if err != nil {
		return
	}
	defer fp.Close()
	fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
	if xDbSpecTblSeqnMap == nil {
		return
	}
	for lref, data := range sonicLeafRefMap {
		fmt.Fprintf (fp, "leafref: %v\r\n", lref)
		for i, d := range data {
			fmt.Fprintf (fp, " (%v) %v\r\n", i, d)
		}
	fmt.Fprintf (fp, "-----------------------------------------------------------------\r\n")
	}
}

func updateSchemaOrderedMap(module string, entry *yang.Entry) {
	var children []string
	if entry.Node.Statement().Keyword == "module" {
		for _, dir := range entry.DirOKeys {
			// Gives the yang xpath for the top level container
			xpath := "/" + module + ":" + dir
			_, ok := xYangSpecMap[xpath]
			if ok {
				yentry := xYangSpecMap[xpath].yangEntry
				if yentry.Node.Statement().Keyword == "container" {
					var keyspec = make([]KeySpec, 0)
					keyspec = FillKeySpecs(xpath, "" , &keyspec)
					children = updateChildTable(keyspec, &children)
				}
			}
		}
	}
	xDbSpecOrdTblMap[module] = children
}

func updateChildTable(keyspec []KeySpec, chlist *[]string) ([]string) {
	for _, ks := range keyspec {
		if (ks.Ts.Name != "") {
			if !contains(*chlist, ks.Ts.Name) {
				*chlist = append(*chlist, ks.Ts.Name)
			}
		}
		*chlist = updateChildTable(ks.Child, chlist)
	}
	return *chlist
}
