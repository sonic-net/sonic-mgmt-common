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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/openconfig/goyang/pkg/yang"

	log "github.com/golang/glog"
)

type typeMapOfInterface map[string]interface{}

var mapCopyMutex = &sync.Mutex{}

func DbValToInt(dbFldVal string, base int, size int, isUint bool) (interface{}, error) {
	var res interface{}
	var err error
	if isUint {
		if res, err = strconv.ParseUint(dbFldVal, base, size); err != nil {
			log.Warningf("Non Yint%v type for YANG leaf-list item %v", size, dbFldVal)
		}
	} else {
		if res, err = strconv.ParseInt(dbFldVal, base, size); err != nil {
			log.Warningf("Non Yint %v type for YANG leaf-list item %v", size, dbFldVal)
		}
	}
	return res, err
}

func getLeafrefRefdYangType(yngTerminalNdDtType yang.TypeKind, fldXpath string) yang.TypeKind {
	if yngTerminalNdDtType == yang.Yleafref {
		entry := getYangEntryForXPath(fldXpath)
		if entry == nil {
			return yngTerminalNdDtType
		}
		path := entry.Type.Path
		xpath, _, _ := XfmrRemoveXPATHPredicates(path)
		xfmrLogDebug("Received path %v for FieldXpath %v", xpath, fldXpath)
		if strings.HasPrefix(xpath, "/..") {
			if entry != nil && len(xpath) > 0 {
				// Referenced path is relative path
				xpath = xpath[1:]
				pathList := strings.Split(xpath, "/")
				for _, x := range pathList {
					if x == ".." {
						entry = entry.Parent
					} else {
						if _, ok := entry.Dir[x]; ok {
							entry = entry.Dir[x]
						}
					}
				}
				if entry != nil && entry.Type != nil {
					yngTerminalNdDtType = entry.Type.Kind
					xfmrLogDebug("yangLeaf datatype %v", yngTerminalNdDtType)

					if yngTerminalNdDtType == yang.Yleafref {
						leafPath := getXpathFromYangEntry(entry)
						if strings.Contains(leafPath, "sonic") {
							pathList := strings.Split(leafPath, "/")
							leafPath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[SONIC_FIELD_INDEX]
						}
						xfmrLogDebug("xpath for leafref type:%v", leafPath)
						return getLeafrefRefdYangType(yngTerminalNdDtType, leafPath)
					}
				}
			}
		} else if len(xpath) > 0 {
			// Referenced path is absolute path
			// Form xpath based on sonic or non sonic YANG path
			if strings.Contains(xpath, "sonic") {
				pathList := strings.Split(xpath, "/")
				if len(pathList) > SONIC_FIELD_INDEX {
					xpath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[SONIC_FIELD_INDEX]
					if xpath == fldXpath {
						if sonicTblChldInfo, ok := xDbSpecMap[pathList[SONIC_TABLE_INDEX]+"/"+pathList[SONIC_TBL_CHILD_INDEX]]; ok {
							if sonicTblChldInfo.dbEntry != nil {
								entry = sonicTblChldInfo.dbEntry.Dir[pathList[SONIC_FIELD_INDEX]]
								yngTerminalNdDtType = sonicTblChldInfo.dbEntry.Dir[pathList[SONIC_FIELD_INDEX]].Type.Kind
							}
						}
					} else {
						entry = getYangEntryForXPath(xpath)
						if entry == nil {
							xfmrLogDebug("Could not get YANG Entry for %v", xpath)
							return yngTerminalNdDtType
						}
						yngTerminalNdDtType = entry.Type.Kind
					}
				}

			} else {
				if strings.Contains(xpath, ":") {
					modName := ""
					if _, ok := entry.Annotation["modulename"]; ok {
						modName = entry.Annotation["modulename"].(string)
					} else {
						xfmrLogDebug("Could not get modulename for xpath %v", xpath)
					}
					xpath = "/" + modName + ":" + strings.SplitN(xpath, ":", 2)[1]
				}

				entry = getYangEntryForXPath(xpath)
				if entry != nil {
					yngTerminalNdDtType = entry.Type.Kind
				} else {
					xfmrLogDebug("Could not resolve leafref path %v", xpath)
					return yngTerminalNdDtType
				}
			}
			if yngTerminalNdDtType == yang.Yleafref {
				leafPath := getXpathFromYangEntry(entry)
				if strings.Contains(leafPath, "sonic") {
					pathList := strings.Split(leafPath, "/")
					if len(pathList) > SONIC_FIELD_INDEX {
						leafPath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[SONIC_FIELD_INDEX]
					}
				}
				xfmrLogDebug("getLeafrefRefdYangType: xpath for leafref type:%v", leafPath)
				return getLeafrefRefdYangType(yngTerminalNdDtType, leafPath)
			}

		}
		xfmrLogDebug("yangLeaf datatype %v", yngTerminalNdDtType)
	}
	return yngTerminalNdDtType
}

func DbToYangType(yngTerminalNdDtType yang.TypeKind, fldXpath string, dbFldVal string) (interface{}, interface{}, error) {
	xfmrLogDebug("Received FieldXpath %v, yngTerminalNdDtType %v and DB field value %v to be converted to YANG data-type.", fldXpath, yngTerminalNdDtType, dbFldVal)
	var res interface{}
	var resPtr interface{}
	var err error
	const INTBASE = 10

	if yngTerminalNdDtType == yang.Yleafref {
		yngTerminalNdDtType = getLeafrefRefdYangType(yngTerminalNdDtType, fldXpath)
	}

	switch yngTerminalNdDtType {
	case yang.Ynone:
		log.Warning("Yang node data-type is non base YANG type")
		//TODO - enhance to handle non base data types depending on future use case
		err = errors.New("Yang node data-type is non base YANG type")
	case yang.Yint8:
		res, err = DbValToInt(dbFldVal, INTBASE, 8, false)
		var resInt8 int8 = int8(res.(int64))
		resPtr = &resInt8
	case yang.Yint16:
		res, err = DbValToInt(dbFldVal, INTBASE, 16, false)
		var resInt16 int16 = int16(res.(int64))
		resPtr = &resInt16
	case yang.Yint32:
		res, err = DbValToInt(dbFldVal, INTBASE, 32, false)
		var resInt32 int32 = int32(res.(int64))
		resPtr = &resInt32
	case yang.Yuint8:
		res, err = DbValToInt(dbFldVal, INTBASE, 8, true)
		var resUint8 uint8 = uint8(res.(uint64))
		resPtr = &resUint8
	case yang.Yuint16:
		res, err = DbValToInt(dbFldVal, INTBASE, 16, true)
		var resUint16 uint16 = uint16(res.(uint64))
		resPtr = &resUint16
	case yang.Yuint32:
		res, err = DbValToInt(dbFldVal, INTBASE, 32, true)
		var resUint32 uint32 = uint32(res.(uint64))
		resPtr = &resUint32
	case yang.Ybool:
		if res, err = strconv.ParseBool(dbFldVal); err != nil {
			log.Warningf("Non Bool type for YANG leaf-list item %v", dbFldVal)
		}
		var resBool bool = res.(bool)
		resPtr = &resBool
	case yang.Ybinary, yang.Ydecimal64, yang.Yenum, yang.Yidentityref, yang.Yint64, yang.Yuint64, yang.Ystring, yang.Yunion, yang.Yleafref:
		// TODO - handle the union type
		// Make sure to encode as string, expected by util_types.go: ytypes.yangToJSONType
		xfmrLogDebug("Yenum/Ystring/Yunion(having all members as strings) type for yangXpath %v", fldXpath)
		res = dbFldVal
		var resString string = res.(string)
		resPtr = &resString
	case yang.Yempty:
		logStr := fmt.Sprintf("Yang data type for xpath %v is Yempty.", fldXpath)
		log.Warning(logStr)
		err = errors.New(logStr)
	default:
		logStr := fmt.Sprintf("Unrecognized/Unhandled yang-data type(%v) for xpath %v.", fldXpath, yang.TypeKindToName[yngTerminalNdDtType])
		log.Warning(logStr)
		err = errors.New(logStr)
	}
	return res, resPtr, err
}

/*convert leaf-list in DB to leaf-list in yang*/
func processLfLstDbToYang(fieldXpath string, dbFldVal string, yngTerminalNdDtType yang.TypeKind) []interface{} {
	valLst := strings.Split(dbFldVal, ",")
	var resLst []interface{}

	xfmrLogDebug("xpath: %v, dbFldVal: %v", fieldXpath, dbFldVal)
	switch yngTerminalNdDtType {
	case yang.Ybinary, yang.Ydecimal64, yang.Yenum, yang.Yidentityref, yang.Yint64, yang.Yuint64, yang.Ystring, yang.Yunion:
		// TODO - handle the union type.OC YANG should have field xfmr.sonic-yang?
		// Make sure to encode as string, expected by util_types.go: ytypes.yangToJSONType:
		xfmrLogDebug("DB leaf-list and Yang leaf-list are of same data-type")
		for _, fldVal := range valLst {
			resLst = append(resLst, fldVal)
		}
	default:
		for _, fldVal := range valLst {
			resVal, _, err := DbToYangType(yngTerminalNdDtType, fieldXpath, fldVal)
			if err == nil {
				resLst = append(resLst, resVal)
			} else {
				log.Warningf("Failed to convert DB value type to YANG type for xpath %v. Field xfmr recommended if data types differ", fieldXpath)
			}
		}
	}
	return resLst
}

func sonicDbToYangTerminalNodeFill(field string, inParamsForGet xlateFromDbParams, dbEntry *yang.Entry) {
	resField := field
	value := ""

	if len(inParamsForGet.queryParams.fields) > 0 {
		curFldXpath := inParamsForGet.tbl + "/" + field
		if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[curFldXpath]; !ok {
			if !inParamsForGet.queryParams.fieldsFillAll {
				return
			}
		}
	}

	if inParamsForGet.dbDataMap != nil {
		tblInstFields, dbDataExists := (*inParamsForGet.dbDataMap)[inParamsForGet.curDb][inParamsForGet.tbl][inParamsForGet.tblKey]
		if dbDataExists {
			fieldVal, valueExists := tblInstFields.Field[field]
			if !valueExists {
				return
			}
			value = fieldVal
		} else {
			return
		}
	}

	if strings.HasSuffix(field, "@") {
		fldVals := strings.Split(field, "@")
		resField = fldVals[0]
	}
	fieldXpath := inParamsForGet.tbl + "/" + resField
	xDbSpecMapEntry, ok := xDbSpecMap[fieldXpath]
	if !ok {
		log.Warningf("No entry found in xDbSpecMap for xpath %v", fieldXpath)
		return
	}
	if dbEntry == nil {
		log.Warningf("Yang entry is nil for xpath %v", fieldXpath)
		return
	}

	yangType := xDbSpecMapEntry.yangType
	yngTerminalNdDtType := dbEntry.Type.Kind
	if yangType == YANG_LEAF_LIST {
		/* this should never happen but just adding for safetty */
		if !strings.HasSuffix(field, "@") {
			log.Warningf("Leaf-list in Sonic YANG should also be a leaf-list in DB, its not for xpath %v", fieldXpath)
			return
		}
		resLst := processLfLstDbToYang(fieldXpath, value, yngTerminalNdDtType)
		inParamsForGet.resultMap[resField] = resLst
	} else { /* yangType is leaf - there are only 2 types of YANG terminal node leaf and leaf-list */
		resVal, _, err := DbToYangType(yngTerminalNdDtType, fieldXpath, value)
		if err != nil {
			log.Warningf("Failed to convert DB value type to YANG type for xpath %v. Field xfmr recommended if data types differ", fieldXpath)
		} else {
			inParamsForGet.resultMap[resField] = resVal
		}
	}
}

func sonicDbToYangListFill(inParamsForGet xlateFromDbParams) []typeMapOfInterface {
	var mapSlice []typeMapOfInterface
	dbDataMap := inParamsForGet.dbDataMap
	table := inParamsForGet.tbl
	dbIdx := inParamsForGet.curDb
	xpath := inParamsForGet.xpath
	dbTblData := (*dbDataMap)[dbIdx][table]

	delKeyCnt := 0
	for keyStr, dbVal := range dbTblData {
		dbSpecData, ok := xDbSpecMap[table]
		if ok && dbSpecData.keyName == nil && xDbSpecMap[xpath].dbEntry != nil {
			yangKeys := yangKeyFromEntryGet(xDbSpecMap[xpath].dbEntry)
			curMap := make(map[string]interface{})
			sonicKeyDataAdd(dbIdx, yangKeys, table, xDbSpecMap[xpath].dbEntry.Name, keyStr, curMap, false)
			if len(curMap) > 0 {
				linParamsForGet := formXlateFromDbParams(inParamsForGet.dbs[dbIdx], inParamsForGet.dbs, dbIdx, inParamsForGet.ygRoot, inParamsForGet.uri, inParamsForGet.requestUri, xpath, inParamsForGet.oper, table, keyStr, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
				sonicDbToYangDataFill(linParamsForGet)
				curMap = linParamsForGet.resultMap
				dbDataMap = linParamsForGet.dbDataMap
				inParamsForGet.dbDataMap = dbDataMap
				if len(curMap) > 0 {
					mapSlice = append(mapSlice, curMap)
				}
				delKeyCnt++
				dbTblData[keyStr] = db.Value{}
				delete(inParamsForGet.dbTblKeyGetCache[dbIdx][table], keyStr)
			} else if len(dbVal.Field) == 0 {
				delKeyCnt++
			}
		}
	}

	if len(dbTblData) == delKeyCnt {
		delete((*dbDataMap)[dbIdx], table)
		delete(inParamsForGet.dbTblKeyGetCache[dbIdx], table)
	}

	return mapSlice
}

func sonicDbToYangDataFill(inParamsForGet xlateFromDbParams) {
	xpath := inParamsForGet.xpath
	uri := inParamsForGet.uri
	table := inParamsForGet.tbl
	key := inParamsForGet.tblKey
	resultMap := inParamsForGet.resultMap
	dbDataMap := inParamsForGet.dbDataMap
	dbIdx := inParamsForGet.curDb
	yangNode, ok := xDbSpecMap[xpath]

	if ok && yangNode.dbEntry != nil {
		xpathPrefix := table
		if len(table) > 0 {
			xpathPrefix += "/"
		}

		for yangChldName := range yangNode.dbEntry.Dir {
			chldXpath := xpathPrefix + yangChldName
			if xDbSpecMap[chldXpath] != nil && yangNode.dbEntry.Dir[yangChldName] != nil {
				chldYangType := xDbSpecMap[chldXpath].yangType

				if chldYangType == YANG_LEAF || chldYangType == YANG_LEAF_LIST {
					xfmrLogDebug("tbl(%v), k(%v), yc(%v)", table, key, yangChldName)
					fldName := yangChldName
					if chldYangType == YANG_LEAF_LIST {
						fldName = fldName + "@"
					}
					curUri := inParamsForGet.uri + "/" + yangChldName
					linParamsForGet := formXlateFromDbParams(nil, inParamsForGet.dbs, dbIdx, inParamsForGet.ygRoot, curUri, inParamsForGet.requestUri, chldXpath, inParamsForGet.oper, table, key, dbDataMap, inParamsForGet.txCache, resultMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
					dbEntry := yangNode.dbEntry.Dir[yangChldName]
					sonicDbToYangTerminalNodeFill(fldName, linParamsForGet, dbEntry)
					resultMap = linParamsForGet.resultMap
					inParamsForGet.resultMap = resultMap
				} else if chldYangType == YANG_CONTAINER {
					curMap := make(map[string]interface{})
					curUri := uri + "/" + yangChldName
					if len(inParamsForGet.queryParams.fields) > 0 {
						if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[chldXpath]; ok {
							inParamsForGet.queryParams.fieldsFillAll = true
						} else if _, ok := inParamsForGet.queryParams.allowFieldsXpath[chldXpath]; !ok {
							if !inParamsForGet.queryParams.fieldsFillAll {
								for path := range inParamsForGet.queryParams.tgtFieldsXpathMap {
									if strings.HasPrefix(chldXpath, path) {
										inParamsForGet.queryParams.fieldsFillAll = true
									}
								}
								if !inParamsForGet.queryParams.fieldsFillAll {
									continue
								}
							}
						}
					}
					// container can have a static key, so extract key for current container
					_, curKey, curTable := sonicXpathKeyExtract(curUri)
					if _, specmapOk := xDbSpecMap[curTable]; !specmapOk || xDbSpecMap[curTable].dbEntry == nil {
						xfmrLogDebug("Yang entry not found for %v", curTable)
						continue
					}
					if inParamsForGet.queryParams.content != QUERY_CONTENT_ALL {
						processReq, _ := sonicContentQParamYangNodeProcess(curUri, xDbSpecMap[curTable].yangType, xDbSpecMap[curTable].dbEntry.ReadOnly(), inParamsForGet.queryParams)
						if !processReq {
							xfmrLogDebug("Further traversal not needed due to content query param, for of URI - %v", curUri)
							continue
						}
					}
					d := inParamsForGet.dbs[xDbSpecMap[curTable].dbIndex]
					linParamsForGet := formXlateFromDbParams(d, inParamsForGet.dbs, xDbSpecMap[curTable].dbIndex, inParamsForGet.ygRoot, curUri, inParamsForGet.requestUri, chldXpath, inParamsForGet.oper, curTable, curKey, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
					sonicDbToYangDataFill(linParamsForGet)
					curMap = linParamsForGet.resultMap
					dbDataMap = linParamsForGet.dbDataMap
					if _, ok := (*dbDataMap)[xDbSpecMap[curTable].dbIndex][curTable][curKey]; ok {
						delete((*dbDataMap)[xDbSpecMap[curTable].dbIndex][curTable], curKey)
					}
					if len(curMap) > 0 {
						resultMap[yangChldName] = curMap
					} else {
						xfmrLogDebug("Empty container for xpath(%v)", curUri)
					}
					inParamsForGet.queryParams.fieldsFillAll = false
					inParamsForGet.dbDataMap = linParamsForGet.dbDataMap
					inParamsForGet.resultMap = resultMap
				} else if chldYangType == YANG_LIST {
					var mapSlice []typeMapOfInterface
					curUri := uri + "/" + yangChldName
					inParamsForGet.uri = curUri
					inParamsForGet.xpath = chldXpath
					if len(inParamsForGet.queryParams.fields) > 0 {
						if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[chldXpath]; ok {
							inParamsForGet.queryParams.fieldsFillAll = true
						} else if _, ok := inParamsForGet.queryParams.allowFieldsXpath[chldXpath]; !ok {
							if !inParamsForGet.queryParams.fieldsFillAll {
								for path := range inParamsForGet.queryParams.tgtFieldsXpathMap {
									if strings.HasPrefix(chldXpath, path) {
										inParamsForGet.queryParams.fieldsFillAll = true
									}
								}
								if !inParamsForGet.queryParams.fieldsFillAll {
									continue
								}
							}
						}
					}
					mapSlice = sonicDbToYangListFill(inParamsForGet)
					dbDataMap = inParamsForGet.dbDataMap
					if len(key) > 0 && len(mapSlice) == 1 { // Single instance query. Don't return array of maps
						for k, val := range mapSlice[0] {
							resultMap[k] = val
						}

					} else if len(mapSlice) > 0 {
						resultMap[yangChldName] = mapSlice
					} else {
						xfmrLogDebug("Empty list for xpath(%v)", curUri)
					}
					inParamsForGet.queryParams.fieldsFillAll = false
					inParamsForGet.resultMap = resultMap
				} else if chldYangType == YANG_CHOICE || chldYangType == YANG_CASE {
					inParamsForGet.xpath = chldXpath
					sonicDbToYangDataFill(inParamsForGet)
					dbDataMap = inParamsForGet.dbDataMap
					resultMap = inParamsForGet.resultMap
				} else {
					xfmrLogDebug("Not handled case %v", chldXpath)
				}
			} else {
				xfmrLogDebug("Yang entry not found for %v", chldXpath)
			}
		}
	}
}

/* Traverse db map and create json for cvl YANG */
func directDbToYangJsonCreate(inParamsForGet xlateFromDbParams) (string, bool, error) {
	var err error
	uri := inParamsForGet.uri
	jsonData := "{}"
	dbDataMap := inParamsForGet.dbDataMap
	resultMap := inParamsForGet.resultMap
	xpath, key, table := sonicXpathKeyExtract(uri)
	inParamsForGet.xpath = xpath
	inParamsForGet.tbl = table
	inParamsForGet.tblKey = key
	fieldName := ""

	traverse := true
	if inParamsForGet.queryParams.depthEnabled {
		pathList := strings.Split(xpath, "/")
		// Depth at requested URI starts at 1. Hence reduce the reqDepth calculated by 1
		reqDepth := (len(pathList) - 1) + int(inParamsForGet.queryParams.curDepth) - 1
		xfmrLogInfo("xpath: %v ,Sonic Yang len(pathlist) %v, reqDepth %v, sonic Field Index %v", xpath, len(pathList), reqDepth, SONIC_FIELD_INDEX)
		if reqDepth < SONIC_FIELD_INDEX {
			traverse = false
		}
	}

	if len(inParamsForGet.queryParams.fields) > 0 {
		flderr := validateAndFillSonicQpFields(inParamsForGet)
		if flderr != nil {
			return jsonData, true, flderr
		}
	}

	if traverse && len(xpath) > 0 {
		tokens := strings.Split(xpath, "/")
		if len(tokens) > SONIC_FIELD_INDEX {
			// Request is at  levelf/leaflist level
			xpath = table + "/" + tokens[SONIC_FIELD_INDEX]
			fieldName = tokens[SONIC_FIELD_INDEX]
			xfmrLogDebug("Request is at terminal node(leaf/leaf-list) - %v.", uri)
		} else if len(tokens) > SONIC_TBL_CHILD_INDEX {
			// Request is at list level
			xpath = table + "/" + tokens[SONIC_TBL_CHILD_INDEX]
			xfmrLogDebug("Request is at immediate child node of table level container - %v.", uri)
		} else if len(tokens) > SONIC_TABLE_INDEX {
			// Request is at table level
			xpath = table
			xfmrLogDebug("Request is at table level container - %v.", uri)
		} else {
			// Request is at top level container prefixed by module name
			xfmrLogDebug("Request is at top level container - %v.", uri)
		}
		dbNode, ok := xDbSpecMap[xpath]
		if !ok {
			xfmrLogInfo("xDbSpecMap doesn't contain entry for xpath - %v", xpath)
		}
		inParamsForGet.xpath = xpath

		if dbNode != nil {
			cdb := db.ConfigDB
			yangType := dbNode.yangType
			if len(table) > 0 {
				cdb = xDbSpecMap[table].dbIndex
			}
			inParamsForGet.curDb = cdb

			if yangType == YANG_LEAF || yangType == YANG_LEAF_LIST {
				dbEntry := getYangEntryForXPath(inParamsForGet.xpath)
				if yangType == YANG_LEAF_LIST {
					fieldName = fieldName + "@"
				}
				linParamsForGet := formXlateFromDbParams(nil, inParamsForGet.dbs, cdb, inParamsForGet.ygRoot, xpath, inParamsForGet.requestUri, uri, inParamsForGet.oper, table, key, dbDataMap, inParamsForGet.txCache, resultMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
				sonicDbToYangTerminalNodeFill(fieldName, linParamsForGet, dbEntry)
				resultMap = linParamsForGet.resultMap
			} else if yangType == YANG_CONTAINER {
				sonicDbToYangDataFill(inParamsForGet)
				resultMap = inParamsForGet.resultMap
			} else if yangType == YANG_LIST {
				mapSlice := sonicDbToYangListFill(inParamsForGet)
				if len(key) > 0 && len(mapSlice) == 1 { // Single instance query. Don't return array of maps
					for k, val := range mapSlice[0] {
						resultMap[k] = val
					}

				} else if len(mapSlice) > 0 {
					pathl := strings.Split(xpath, "/")
					lname := pathl[len(pathl)-1]
					resultMap[lname] = mapSlice
				}
			}
		}
	}

	jsonMapData, _ := json.Marshal(resultMap)
	isEmptyPayload := isJsonDataEmpty(string(jsonMapData))
	jsonData = fmt.Sprintf("%v", string(jsonMapData))
	if isEmptyPayload {
		log.Warning("No data available")
	}
	return jsonData, isEmptyPayload, err
}

func tableNameAndKeyFromDbMapGet(dbDataMap map[string]map[string]db.Value) (string, string, error) {
	tableName := ""
	tableKey := ""
	for tn, tblData := range dbDataMap {
		tableName = tn
		for kname := range tblData {
			tableKey = kname
		}
	}
	return tableName, tableKey, nil
}

func fillDbDataMapForTbl(uri string, xpath string, tblName string, tblKey string, cdb db.DBNum, dbs [db.MaxDB]*db.DB, dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool) (map[db.DBNum]map[string]map[string]db.Value, error) {
	var err error
	dbresult := make(RedisDbMap)
	dbresult[cdb] = make(map[string]map[string]db.Value)
	dbFormat := KeySpec{}
	dbFormat.Ts.Name = tblName
	dbFormat.DbNum = cdb
	if tblKey != "" {
		if !isSonicYang(uri) {
			// Identify if the dbKey is a partial key
			dbFormat.IsPartialKey = verifyPartialKeyForOc(uri, xpath, tblKey)
		}
		if tblSpecInfo, ok := xDbSpecMap[tblName]; ok && tblSpecInfo.hasXfmrFn {
			/* key from URI should be converted into redis-db key, to read data */
			tblKey, err = dbKeyValueXfmrHandler(CREATE, cdb, tblName, tblKey, dbFormat.IsPartialKey)
			if err != nil {
				log.Warningf("Value-xfmr for table(%v) & key(%v) didn't do conversion.", tblName, tblKey)
				return nil, err
			}
		}

		dbFormat.Key.Comp = append(dbFormat.Key.Comp, tblKey)
	}
	err = TraverseDb(dbs, dbFormat, &dbresult, nil, dbTblKeyGetCache)
	if err != nil {
		xfmrLogInfo("Didn't fetch DB data for tbl(DB num) %v(%v) for xpath %v", tblName, cdb, xpath)
		return nil, err
	}
	if _, ok := dbresult[cdb]; !ok {
		logStr := fmt.Sprintf("TraverseDb() did not populate DB data for tbl(DB num) %v(%v) for xpath %v", tblName, cdb, xpath)
		err = fmt.Errorf("%v", logStr)
		return nil, err
	}
	return dbresult, err

}

// Assumption: All tables are from the same DB
func dbDataFromTblXfmrGet(tbl string, inParams XfmrParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, xpath string) error {
	// skip the query if the table is already visited
	if _, ok := (*dbDataMap)[inParams.curDb][tbl]; ok {
		if len(inParams.key) > 0 {
			if _, ok = (*dbDataMap)[inParams.curDb][tbl][inParams.key]; ok {
				return nil
			}
		} else {
			return nil
		}
	}

	terminalNodeGet := false
	qdbMapHasTblData := false
	qdbMapHasTblKeyData := false
	if !xYangSpecMap[xpath].hasNonTerminalNode && len(inParams.key) > 0 {
		terminalNodeGet = true
	}
	if qdbMap, getOk := dbTblKeyGetCache[inParams.curDb]; getOk {
		if dbTblData, tblPresent := qdbMap[tbl]; tblPresent {
			qdbMapHasTblData = true
			if _, keyPresent := dbTblData[inParams.key]; keyPresent {
				qdbMapHasTblKeyData = true
			}
		}
	}

	if !qdbMapHasTblData || (terminalNodeGet && qdbMapHasTblData && !qdbMapHasTblKeyData) {
		curDbDataMap, err := fillDbDataMapForTbl(inParams.uri, xpath, tbl, inParams.key, inParams.curDb, inParams.dbs, dbTblKeyGetCache)
		if err == nil {
			mapCopy((*dbDataMap)[inParams.curDb], curDbDataMap[inParams.curDb])
		}
	}
	return nil
}

func yangListDataFill(inParamsForGet xlateFromDbParams, isFirstCall bool, isOcMdl bool) error {
	var tblList []string
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	cdb := inParamsForGet.curDb
	resultMap := inParamsForGet.resultMap
	xpath := inParamsForGet.xpath
	tbl := inParamsForGet.tbl
	tblKey := inParamsForGet.tblKey

	_, ok := xYangSpecMap[xpath]
	if ok {
		// Do not handle list if the curdepth is 1
		if inParamsForGet.queryParams.depthEnabled && (inParamsForGet.queryParams.curDepth <= 1) {
			return nil
		}

		if xYangSpecMap[xpath].xfmrTbl != nil {
			xfmrTblFunc := *xYangSpecMap[xpath].xfmrTbl
			if len(xfmrTblFunc) > 0 {
				inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, tblKey, dbDataMap, nil, nil, txCache)
				tblList, _ = xfmrTblHandlerFunc(xfmrTblFunc, inParams, inParamsForGet.xfmrDbTblKeyCache)
				inParamsForGet.dbDataMap = dbDataMap
				inParamsForGet.ygRoot = ygRoot
				if len(tblList) != 0 {
					for _, curTbl := range tblList {
						dbDataFromTblXfmrGet(curTbl, inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, xpath)
						inParamsForGet.dbDataMap = dbDataMap
						inParamsForGet.ygRoot = ygRoot
					}
				}
			}
			if tbl != "" {
				if !contains(tblList, tbl) {
					tblList = append(tblList, tbl)
				}
			}
		} else if tbl != "" && xYangSpecMap[xpath].xfmrTbl == nil {
			tblList = append(tblList, tbl)
		} else if tbl == "" && xYangSpecMap[xpath].xfmrTbl == nil {
			// Handling for case: Parent list is not associated with a tableName but has children containers/lists having tableNames.
			if tblKey != "" {
				var mapSlice []typeMapOfInterface
				instMap, err := yangListInstanceDataFill(inParamsForGet, isFirstCall, isOcMdl)
				dbDataMap = inParamsForGet.dbDataMap
				if err != nil {
					xfmrLogDebug("Error(%v) returned for %v", err, uri)
					// abort GET request if QP Subtree Pruning API returns error
					_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
					if qpSbtPruneErrOk {
						return err
					}
				} else if (instMap != nil) && (len(instMap) > 0) {
					mapSlice = append(mapSlice, instMap)
				}

				if len(mapSlice) > 0 {
					listInstanceGet := false
					// Check if it is a list instance level Get
					if (strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/")) {
						listInstanceGet = true
						for k, v := range mapSlice[0] {
							resultMap[k] = v
						}
					}
					if !listInstanceGet && xYangSpecMap[xpath].yangEntry != nil {
						resultMap[xYangSpecMap[xpath].yangEntry.Name] = mapSlice
					}
					inParamsForGet.resultMap = resultMap
				}
			}
		}
	}

	if len(tblList) == 0 {
		if (strings.HasSuffix(uri, "]") || strings.HasSuffix(uri, "]/")) && (len(xYangSpecMap[xpath].xfmrFunc) > 0 && xYangSpecMap[xpath].hasChildSubTree) {
			err := yangDataFill(inParamsForGet, isOcMdl)
			if err != nil {
				xfmrLogInfo("yangListDataFill: error in its child subtree traversal for the xpath: %v", xpath)
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return err
				}
			}
			return nil
		} else {
			xfmrLogInfo("Unable to traverse list as no table information available at URI %v. Please check if table mapping available", uri)
		}
	}

	for _, tbl = range tblList {
		inParamsForGet.tbl = tbl

		tblData, ok := (*dbDataMap)[cdb][tbl]

		if ok {
			var mapSlice []typeMapOfInterface
			for dbKey := range tblData {
				inParamsForGet.tblKey = dbKey
				instMap, err := yangListInstanceDataFill(inParamsForGet, isFirstCall, isOcMdl)
				dbDataMap = inParamsForGet.dbDataMap
				if err != nil {
					xfmrLogDebug("Error(%v) returned for %v", err, uri)
					// abort GET request if QP Subtree Pruning API returns error
					_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
					if qpSbtPruneErrOk {
						return err
					}
				} else if (instMap != nil) && (len(instMap) > 0) {
					mapSlice = append(mapSlice, instMap)
				}
			}

			if len(mapSlice) > 0 {
				listInstanceGet := false
				/*Check if it is a list instance level Get*/
				if (strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/")) {
					listInstanceGet = true
					for k, v := range mapSlice[0] {
						resultMap[k] = v
					}
				}
				if !listInstanceGet {
					if _, specOk := xYangSpecMap[xpath]; specOk && xYangSpecMap[xpath].yangEntry != nil {
						if _, ok := resultMap[xYangSpecMap[xpath].yangEntry.Name]; ok {
							mlen := len(resultMap[xYangSpecMap[xpath].yangEntry.Name].([]typeMapOfInterface))
							for i := 0; i < mlen; i++ {
								mapSlice = append(mapSlice, resultMap[xYangSpecMap[xpath].yangEntry.Name].([]typeMapOfInterface)[i])
							}
						}
						resultMap[xYangSpecMap[xpath].yangEntry.Name] = mapSlice
						inParamsForGet.resultMap = resultMap
					}
				}
			} else {
				xfmrLogDebug("Empty slice for (\"%v\").\r\n", uri)
			}
		}
	} // end of tblList for

	return nil
}

func yangListInstanceDataFill(inParamsForGet xlateFromDbParams, isFirstCall bool, isOcMdl bool) (typeMapOfInterface, error) {

	var err error
	curMap := make(map[string]interface{})
	err = nil
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	cdb := inParamsForGet.curDb
	xpath := inParamsForGet.xpath
	tbl := inParamsForGet.tbl
	dbKey := inParamsForGet.tblKey

	curKeyMap, curUri, err := dbKeyToYangDataConvert(uri, requestUri, xpath, tbl, dbDataMap, dbKey, dbs[cdb].Opts.KeySeparator, txCache)
	if (err != nil) || (curKeyMap == nil) || (len(curKeyMap) == 0) {
		xfmrLogDebug("Skip filling list instance for URI %v since no YANG  key found corresponding to db-key %v", uri, dbKey)
		return curMap, err
	}
	parentXpath := parentXpathGet(xpath)
	_, ok := xYangSpecMap[xpath]
	if ok && len(xYangSpecMap[xpath].xfmrFunc) > 0 {
		if isFirstCall || (!isFirstCall && (uri != requestUri) && ((len(xYangSpecMap[parentXpath].xfmrFunc) == 0) ||
			(len(xYangSpecMap[parentXpath].xfmrFunc) > 0 && (xYangSpecMap[parentXpath].xfmrFunc != xYangSpecMap[xpath].xfmrFunc)))) {
			xfmrLogDebug("Parent subtree already handled cur uri: %v", xpath)
			inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, GET, dbKey, dbDataMap, nil, nil, txCache)
			inParams.queryParams = inParamsForGet.queryParams
			err := xfmrHandlerFunc(inParams, xYangSpecMap[xpath].xfmrFunc)
			inParamsForGet.ygRoot = ygRoot
			inParamsForGet.dbDataMap = dbDataMap
			if err != nil {
				xfmrLogDebug("Error returned by %v: %v", xYangSpecMap[xpath].xfmrFunc, err)
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return curMap, err
				}
			}
		}
		if xYangSpecMap[xpath].hasChildSubTree {
			linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, xpath, inParamsForGet.oper, tbl, dbKey, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
			linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
			linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
			err := yangDataFill(linParamsForGet, isOcMdl)
			if err != nil {
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return nil, err
				}
			}
			curMap = linParamsForGet.resultMap
			dbDataMap = linParamsForGet.dbDataMap
			ygRoot = linParamsForGet.ygRoot
			inParamsForGet.dbDataMap = dbDataMap
			inParamsForGet.ygRoot = ygRoot
		} else {
			xfmrLogDebug("Has no child subtree at uri: %v. xfmr will not traverse subtree further", uri)
		}
	} else {
		xpathKeyExtRet, _ := xpathKeyExtract(dbs[cdb], ygRoot, GET, curUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache)
		keyFromCurUri := xpathKeyExtRet.dbKey
		inParamsForGet.ygRoot = ygRoot
		var listKeyMap map[string]interface{}
		if dbKey == keyFromCurUri || keyFromCurUri == "" {
			isValid := inParamsForGet.validate
			if len(xYangSpecMap[xpath].validateFunc) > 0 && !inParamsForGet.validate {
				inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
				res := validateHandlerFunc(inParams, xYangSpecMap[xpath].validateFunc)
				if !res {
					xfmrLogDebug("Further traversal not needed. Validate xfmr returns false for URI %v", curUri)
					return nil, nil
				} else {
					isValid = res
				}

			}

			if dbKey == keyFromCurUri {
				listKeyMap = make(map[string]interface{})
				for k, kv := range curKeyMap {
					curMap[k] = kv
					listKeyMap[k] = kv
				}

			}
			linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, xpathKeyExtRet.xpath, inParamsForGet.oper, tbl, dbKey, dbDataMap, inParamsForGet.txCache, curMap, isValid, inParamsForGet.queryParams, listKeyMap)
			linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
			linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
			err := yangDataFill(linParamsForGet, isOcMdl)
			if err != nil {
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return curMap, err
				}
			}
			curMap = linParamsForGet.resultMap
			dbDataMap = linParamsForGet.dbDataMap
			ygRoot = linParamsForGet.ygRoot
			inParamsForGet.dbDataMap = dbDataMap
			inParamsForGet.ygRoot = ygRoot
		} else {
			xfmrLogDebug("Mismatch in dbKey and key derived from uri. dbKey : %v, keyFromCurUri: %v. Cannot traverse instance for URI %v. Please check key mapping", dbKey, keyFromCurUri, curUri)
		}
	}
	return curMap, err
}

func terminalNodeProcess(inParamsForGet xlateFromDbParams, terminalNodeQuery bool, yangEntry *yang.Entry) (map[string]interface{}, error) {
	xfmrLogDebug("Received xpath - %v, URI - %v, table - %v, table key - %v", inParamsForGet.xpath, inParamsForGet.uri, inParamsForGet.tbl, inParamsForGet.tblKey)
	var err error
	resFldValMap := make(map[string]interface{})
	xpath := inParamsForGet.xpath
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	tbl := inParamsForGet.tbl
	tblKey := inParamsForGet.tblKey
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache

	if uri != requestUri {
		//Chk should be already done in yangDataFill
		if inParamsForGet.queryParams.depthEnabled && inParamsForGet.queryParams.curDepth == 0 {
			return resFldValMap, err
		}
	}

	_, ok := xYangSpecMap[xpath]
	if !ok || yangEntry == nil {
		logStr := fmt.Sprintf("No YANG entry found for xpath %v.", xpath)
		err = fmt.Errorf("%v", logStr)
		return resFldValMap, err
	}

	cdb := xYangSpecMap[xpath].dbIndex
	if len(xYangSpecMap[xpath].xfmrField) > 0 {
		inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, tblKey, dbDataMap, nil, nil, txCache)
		inParams.queryParams = inParamsForGet.queryParams
		fldValMap, err := leafXfmrHandlerFunc(inParams, xYangSpecMap[xpath].xfmrField)
		inParamsForGet.ygRoot = ygRoot
		inParamsForGet.dbDataMap = dbDataMap
		if err != nil {
			xfmrLogDebug("No data from field transformer for %v: %v.", uri, err)
			return resFldValMap, err
		}
		if uri == requestUri {
			yangType := xYangSpecMap[xpath].yangType
			if len(fldValMap) == 0 {
				// field transformer returns empty map when no data in DB
				if (yangType == YANG_LEAF) || ((yangType == YANG_LEAF_LIST) && ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/")))) {
					log.Warningf("Field transformer returned empty data , URI  - %v", requestUri)
					err = tlerr.NotFoundError{Format: "Resource not found"}
					return resFldValMap, err
				}
			} else {
				if (yangType == YANG_LEAF_LIST) && ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {
					return resFldValMap, nil
				}
			}
		}
		for lf, val := range fldValMap {
			resFldValMap[lf] = val
		}
	} else {
		dbFldName := xYangSpecMap[xpath].fieldName
		if dbFldName == XFMR_NONE_STRING {
			return resFldValMap, err
		}
		fillLeafFromUriKey := false
		yangDataType := yangEntry.Type.Kind
		if terminalNodeQuery && xYangSpecMap[xpath].isKey { //GET request for list key-leaf(direct child of list)
			fillLeafFromUriKey = true
		} else if len(dbFldName) > 0 && !xYangSpecMap[xpath].isKey {
			yangType := xYangSpecMap[xpath].yangType
			if yangType == YANG_LEAF_LIST {
				dbFldName += "@"
				val, ok := (*dbDataMap)[cdb][tbl][tblKey].Field[dbFldName]
				leafLstInstGetReq := false

				if terminalNodeQuery && ((strings.HasSuffix(requestUri, "]")) || (strings.HasSuffix(requestUri, "]/"))) {
					xfmrLogDebug("Request URI is leaf-list instance GET - %v", requestUri)
					leafLstInstGetReq = true
				}
				if ok {
					if leafLstInstGetReq {
						leafListInstVal, valErr := extractLeafListInstFromUri(requestUri)
						if valErr != nil {
							return resFldValMap, valErr
						}
						dbSpecField := tbl + "/" + strings.TrimSuffix(dbFldName, "@")
						dbSpecFieldInfo, dbSpecOk := xDbSpecMap[dbSpecField]
						if dbSpecOk && dbSpecFieldInfo.xfmrValue != nil {
							inParams := formXfmrDbInputRequest(CREATE, cdb, tbl, tblKey, dbFldName, leafListInstVal)
							retVal, valXfmrErr := valueXfmrHandler(inParams, *dbSpecFieldInfo.xfmrValue)
							if valXfmrErr != nil {
								log.Warningf("value-xfmr:fldpath(\"%v\") val(\"%v\"):err(\"%v\").", dbSpecField, leafListInstVal, valXfmrErr)
								return resFldValMap, valXfmrErr
							}
							leafListInstVal = retVal
						}
						if !leafListInstExists((*dbDataMap)[cdb][tbl][tblKey].Field[dbFldName], leafListInstVal) {
							log.Warningf("Queried leaf-list instance does not exists, URI  - %v, dbData - %v", requestUri, (*dbDataMap)[cdb][tbl][tblKey].Field[dbFldName])
							err = tlerr.NotFoundError{Format: "Resource not found"}
						}
						if err == nil {
							/* Since translib already fills in ygRoot with queried leaf-list instance, do not
							   fill in resFldValMap or else Unmarshall of payload(resFldValMap) into ygotTgt in
							   app layer will create duplicate instances in result.
							*/
							log.Info("Queried leaf-list instance exists but Since translib already fills in ygRoot with queried leaf-list instance do not populate payload.")
						}
						return resFldValMap, err
					} else {
						resLst := processLfLstDbToYang(xpath, val, yangDataType)
						resFldValMap[yangEntry.Name] = resLst
					}
				} else {
					if leafLstInstGetReq {
						log.Warningf("Queried leaf-list does not exist in DB, URI  - %v", requestUri)
						err = tlerr.NotFoundError{Format: "Resource not found"}
					}
				}
			} else {
				val, ok := (*dbDataMap)[cdb][tbl][tblKey].Field[dbFldName]
				if ok {
					resVal, _, err := DbToYangType(yangDataType, xpath, val)
					if err != nil {
						log.Warning("Conversion of DB value type to YANG type for field didn't happen. Field-xfmr recommended if data types differ. Field xpath", xpath)
					} else {
						resFldValMap[yangEntry.Name] = resVal
					}
				} else {
					resNotFound := true
					if xYangSpecMap[xpath].isRefByKey {
						/*config container leaf is referenced by list-key that exists already before reaching here
						  state container leaf is not referenced by list-key, some state containers are mapped to
						  non-config DB(different from list level DB mapping), so check instance existence before
						  filling from uri*/
						_, stateContainerInstanceOk := (*dbDataMap)[cdb][tbl][tblKey]
						if !yangEntry.ReadOnly() || stateContainerInstanceOk {
							fillLeafFromUriKey = true
							resNotFound = false
						}
					}
					if resNotFound {
						xfmrLogDebug("Field value does not exist in DB for - %v", uri)
						err = tlerr.NotFoundError{Format: "Resource not found"}
					}
				}
			}
		} else if (len(dbFldName) == 0) && (xYangSpecMap[xpath].isRefByKey) {
			_, stateContainerInstanceOk := (*dbDataMap)[cdb][tbl][tblKey]
			if !yangEntry.ReadOnly() || stateContainerInstanceOk {
				fillLeafFromUriKey = true
			}
		}
		if fillLeafFromUriKey {
			extractKeyLeafFromUri := true
			xfmrLogDebug("Inferring isRefByKey Leaf %v value from list instance/keys", xpath)
			if len(inParamsForGet.listKeysMap) > 0 {
				if resVal, keyLeafExists := inParamsForGet.listKeysMap[yangEntry.Name]; keyLeafExists {
					resFldValMap = make(map[string]interface{})
					resFldValMap[yangEntry.Name] = resVal
					xfmrLogDebug("Filled isRefByKey leaf value from list keys map")
					extractKeyLeafFromUri = false
				}
			}
			if extractKeyLeafFromUri {
				xfmrLogDebug("Filling isRefByKey leaf valuea from uri string.")
				val := extractLeafValFromUriKey(uri, yangEntry.Name)
				resVal, _, err := DbToYangType(yangDataType, xpath, val)
				if err != nil {
					log.Warning("Conversion of DB value type to YANG type for field didn't happen. Field-xfmr recommended if data types differ. Field xpath", xpath)
				} else {
					resFldValMap = make(map[string]interface{})
					resFldValMap[yangEntry.Name] = resVal
				}
			}
		}
	}
	return resFldValMap, err
}

func yangDataFill(inParamsForGet xlateFromDbParams, isOcMdl bool) error {
	var err error
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	resultMap := inParamsForGet.resultMap
	xpath := inParamsForGet.xpath
	var chldUri string

	yangNode, ok := xYangSpecMap[xpath]

	if ok && yangNode.yangEntry != nil {
		if inParamsForGet.queryParams.depthEnabled {
			// If fields are available, we need to evaluate the depth after fields level is met
			// The depth upto the fields level is considered at level 1
			inParamsForGet.queryParams.curDepth = inParamsForGet.queryParams.curDepth - 1
			log.Infof("yangDataFill curdepth: %v, Path : %v", inParamsForGet.queryParams.curDepth, xpath)
			if inParamsForGet.queryParams.curDepth == 0 {
				return err
			}
		}

		for yangChldName := range yangNode.yangEntry.Dir {
			chldXpath := xpath + "/" + yangChldName
			chFieldsFillAll := inParamsForGet.queryParams.fieldsFillAll
			if xYangSpecMap[chldXpath] != nil && xYangSpecMap[chldXpath].nameWithMod != nil {
				chldUri = uri + "/" + *(xYangSpecMap[chldXpath].nameWithMod)
			} else {
				chldUri = uri + "/" + yangChldName
			}
			inParamsForGet.xpath = chldXpath
			inParamsForGet.uri = chldUri
			isValid := inParamsForGet.validate
			if xYangSpecMap[chldXpath] != nil && yangNode.yangEntry.Dir[yangChldName] != nil {
				chldYangType := xYangSpecMap[chldXpath].yangType
				if inParamsForGet.queryParams.content != QUERY_CONTENT_ALL {
					yangNdInfo := contentQPSpecMapInfo{
						yangType:              chldYangType,
						yangName:              yangChldName,
						isReadOnly:            yangNode.yangEntry.Dir[yangChldName].ReadOnly(),
						isOperationalNd:       xYangSpecMap[chldXpath].operationalQP,
						hasNonTerminalNd:      xYangSpecMap[chldXpath].hasNonTerminalNode,
						hasChildOperationalNd: xYangSpecMap[chldXpath].hasChildOpertnlNd,
						isOcMdl:               isOcMdl,
					}
					processReq, _ := contentQParamYangNodeProcess(chldUri, yangNdInfo, inParamsForGet.queryParams)
					if !processReq {
						xfmrLogDebug("Further traversal not needed due to content query param, for of URI - %v", chldUri)
						continue
					}
				}

				if inParamsForGet.queryParams.depthEnabled && inParamsForGet.queryParams.curDepth == 1 {
					if (chldYangType == YANG_CONTAINER) || (chldYangType == YANG_LIST) {
						continue
					}
				}

				cdb := xYangSpecMap[chldXpath].dbIndex
				inParamsForGet.curDb = cdb
				/* For list validate handler is evaluated at each instance */
				if len(xYangSpecMap[chldXpath].validateFunc) > 0 && !inParamsForGet.validate && chldYangType != YANG_LIST {
					xpathKeyExtRet, _ := xpathKeyExtract(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache)
					inParamsForGet.ygRoot = ygRoot
					// TODO - handle non CONFIG-DB
					inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
					inParams.queryParams = inParamsForGet.queryParams
					res := validateHandlerFunc(inParams, xYangSpecMap[chldXpath].validateFunc)
					if !res {
						xfmrLogDebug("Further traversal not needed. Validate xfmr returns false for URI %v", chldUri)
						continue
					} else {
						isValid = res
					}
					inParamsForGet.dbDataMap = dbDataMap
					inParamsForGet.ygRoot = ygRoot
				}
				if chldYangType == YANG_LEAF || chldYangType == YANG_LEAF_LIST {
					if len(xYangSpecMap[xpath].xfmrFunc) > 0 {
						continue
					}
					if !xYangSpecMap[chldXpath].isKey && len(inParamsForGet.queryParams.fields) > 0 {
						if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[chldXpath]; !ok {
							if !inParamsForGet.queryParams.fieldsFillAll {
								xfmrLogDebug("Skip processing URI due to fields QP procesing - %v", chldUri)
								continue
							}
						}
					}
					yangEntry := yangNode.yangEntry.Dir[yangChldName]
					fldValMap, err := terminalNodeProcess(inParamsForGet, false, yangEntry)
					dbDataMap = inParamsForGet.dbDataMap
					ygRoot = inParamsForGet.ygRoot
					if err != nil {
						xfmrLogDebug("Failed to get data(\"%v\").", chldUri)
					}
					for lf, val := range fldValMap {
						resultMap[lf] = val
					}
					inParamsForGet.resultMap = resultMap
				} else if chldYangType == YANG_CONTAINER {
					xpathKeyExtRet, _ := xpathKeyExtract(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache)
					tblKey := xpathKeyExtRet.dbKey
					chtbl := xpathKeyExtRet.tableName
					inParamsForGet.ygRoot = ygRoot

					if len(inParamsForGet.queryParams.fields) > 0 {
						if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[chldXpath]; ok {
							chFieldsFillAll = true
						} else if _, ok := inParamsForGet.queryParams.allowFieldsXpath[chldXpath]; !ok {
							if !inParamsForGet.queryParams.fieldsFillAll {
								for path := range inParamsForGet.queryParams.tgtFieldsXpathMap {
									if strings.HasPrefix(chldXpath, path) {
										chFieldsFillAll = true
									}
								}
								if !chFieldsFillAll {
									continue
								}
							}
						}
					}

					if _, ok := (*dbDataMap)[cdb][chtbl][tblKey]; !ok && len(chtbl) > 0 {
						qdbMapHasTblData := false
						qdbMapHasTblKeyData := false
						if qdbMap, getOk := inParamsForGet.dbTblKeyGetCache[cdb]; getOk {
							if dbTblData, tblPresent := qdbMap[chtbl]; tblPresent {
								qdbMapHasTblData = true
								if _, keyPresent := dbTblData[tblKey]; keyPresent {
									qdbMapHasTblKeyData = true
								}
							}
						}

						if !qdbMapHasTblData || (qdbMapHasTblData && !qdbMapHasTblKeyData) {
							curDbDataMap, err := fillDbDataMapForTbl(chldUri, chldXpath, chtbl, tblKey, cdb, dbs, inParamsForGet.dbTblKeyGetCache)
							if err == nil {
								mapCopy((*dbDataMap)[cdb], curDbDataMap[cdb])
								inParamsForGet.dbDataMap = dbDataMap
							}
						}
					}
					if xYangSpecMap[chldXpath].xfmrTbl != nil {
						xfmrTblFunc := *xYangSpecMap[chldXpath].xfmrTbl
						if len(xfmrTblFunc) > 0 {
							inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, GET, tblKey, dbDataMap, nil, nil, txCache)
							tblList, _ := xfmrTblHandlerFunc(xfmrTblFunc, inParams, inParamsForGet.xfmrDbTblKeyCache)
							inParamsForGet.dbDataMap = dbDataMap
							inParamsForGet.ygRoot = ygRoot
							if len(tblList) > 1 {
								log.Warningf("Table transformer returned more than one table for container %v", chldXpath)
							}
							if len(tblList) == 0 {
								continue
							}
							dbDataFromTblXfmrGet(tblList[0], inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, chldXpath)
							inParamsForGet.dbDataMap = dbDataMap
							inParamsForGet.ygRoot = ygRoot
							chtbl = tblList[0]
						}
					}
					if len(xYangSpecMap[chldXpath].xfmrFunc) > 0 {
						if (len(xYangSpecMap[xpath].xfmrFunc) == 0) ||
							(len(xYangSpecMap[xpath].xfmrFunc) > 0 &&
								(xYangSpecMap[xpath].xfmrFunc != xYangSpecMap[chldXpath].xfmrFunc)) {
							inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
							inParams.queryParams = inParamsForGet.queryParams
							err := xfmrHandlerFunc(inParams, xYangSpecMap[chldXpath].xfmrFunc)
							inParamsForGet.dbDataMap = dbDataMap
							inParamsForGet.ygRoot = ygRoot
							if err != nil {
								xfmrLogDebug("Error returned by %v: %v", xYangSpecMap[xpath].xfmrFunc, err)
								// abort GET request if QP Subtree Pruning API returns error
								_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
								if qpSbtPruneErrOk {
									return err
								}
							}
						}
						if !xYangSpecMap[chldXpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. xfmr will not traverse subtree further", chldUri)
							continue
						}
					}
					cmap2 := make(map[string]interface{})
					linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, chldXpath, inParamsForGet.oper, chtbl, tblKey, dbDataMap, inParamsForGet.txCache, cmap2, isValid, inParamsForGet.queryParams, inParamsForGet.listKeysMap)
					linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
					linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
					linParamsForGet.queryParams.fieldsFillAll = chFieldsFillAll
					err = yangDataFill(linParamsForGet, isOcMdl)
					if err != nil {
						// abort GET request if QP Subtree Pruning API returns error
						_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
						if qpSbtPruneErrOk {
							return err
						}
					}
					cmap2 = linParamsForGet.resultMap
					dbDataMap = linParamsForGet.dbDataMap
					ygRoot = linParamsForGet.ygRoot
					if err != nil && len(cmap2) == 0 {
						xfmrLogDebug("Empty container.(\"%v\").\r\n", chldUri)
					} else {
						if len(cmap2) > 0 {
							resultMap[yangChldName] = cmap2
						}
						inParamsForGet.resultMap = resultMap
					}
					inParamsForGet.dbDataMap = dbDataMap
					inParamsForGet.ygRoot = ygRoot
				} else if chldYangType == YANG_LIST {
					xpathKeyExtRet, _ := xpathKeyExtract(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache)
					inParamsForGet.ygRoot = ygRoot
					cdb = xYangSpecMap[chldXpath].dbIndex
					inParamsForGet.curDb = cdb
					if len(xYangSpecMap[chldXpath].xfmrFunc) > 0 {
						if (len(xYangSpecMap[xpath].xfmrFunc) == 0) ||
							(len(xYangSpecMap[xpath].xfmrFunc) > 0 &&
								(xYangSpecMap[xpath].xfmrFunc != xYangSpecMap[chldXpath].xfmrFunc)) {
							inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
							inParams.queryParams = inParamsForGet.queryParams
							err := xfmrHandlerFunc(inParams, xYangSpecMap[chldXpath].xfmrFunc)
							if err != nil {
								xfmrLogDebug("Error returned by %v: %v", xYangSpecMap[chldXpath].xfmrFunc, err)
								// abort GET request if QP Subtree Pruning API returns error
								_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
								if qpSbtPruneErrOk {
									return err
								}
							}
							inParamsForGet.dbDataMap = dbDataMap
							inParamsForGet.ygRoot = ygRoot
						}
						if !xYangSpecMap[chldXpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. Do not traverse further", chldUri)
							continue
						}
					}
					ynode, ok := xYangSpecMap[chldXpath]
					lTblName := ""
					if ok && ynode.tableName != nil {
						lTblName = *ynode.tableName
					}
					if _, ok := (*dbDataMap)[cdb][lTblName]; !ok && len(lTblName) > 0 {
						curDbDataMap, err := fillDbDataMapForTbl(chldUri, chldXpath, lTblName, "", cdb, dbs, inParamsForGet.dbTblKeyGetCache)
						if err == nil {
							mapCopy((*dbDataMap)[cdb], curDbDataMap[cdb])
							inParamsForGet.dbDataMap = dbDataMap
						}
					}
					linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, chldXpath, inParamsForGet.oper, lTblName, xpathKeyExtRet.dbKey, dbDataMap, inParamsForGet.txCache, resultMap, inParamsForGet.validate, inParamsForGet.queryParams, nil)
					linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
					linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
					err := yangListDataFill(linParamsForGet, false, isOcMdl)
					if err != nil {
						// abort GET request if QP Subtree Pruning API returns error
						_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
						if qpSbtPruneErrOk {
							return err
						}
					}
					resultMap = linParamsForGet.resultMap
					dbDataMap = linParamsForGet.dbDataMap
					ygRoot = linParamsForGet.ygRoot
					inParamsForGet.dbDataMap = dbDataMap
					inParamsForGet.resultMap = resultMap
					inParamsForGet.ygRoot = ygRoot

				} else if chldYangType == YANG_CHOICE || chldYangType == YANG_CASE {
					err := yangDataFill(inParamsForGet, isOcMdl)
					if err != nil {
						// abort GET request if QP Subtree Pruning API returns error
						_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
						if qpSbtPruneErrOk {
							return err
						}
					}
					resultMap = inParamsForGet.resultMap
					dbDataMap = inParamsForGet.dbDataMap
				} else {
					return err
				}
			}
		}
	}
	return err
}

/* Traverse linear db-map data and add to nested json data */
func dbDataToYangJsonCreate(inParamsForGet xlateFromDbParams) (string, bool, error) {
	var err error
	var fldSbtErr error // used only when direct query on leaf/leaf-list having subtree
	var fldErr error    //used only when direct query on leaf/leaf-list having field transformer
	var isOcMdl bool
	jsonData := "{}"
	resultMap := make(map[string]interface{})
	d := inParamsForGet.d
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	cdb := inParamsForGet.curDb
	inParamsForGet.resultMap = resultMap

	if isSonicYang(uri) {
		return directDbToYangJsonCreate(inParamsForGet)
	} else {
		xpathKeyExtRet, _ := xpathKeyExtract(d, ygRoot, GET, uri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache)

		inParamsForGet.xpath = xpathKeyExtRet.xpath
		inParamsForGet.tbl = xpathKeyExtRet.tableName
		inParamsForGet.tblKey = xpathKeyExtRet.dbKey
		inParamsForGet.ygRoot = ygRoot
		yangNode, ok := xYangSpecMap[xpathKeyExtRet.xpath]
		if ok {
			yangType := yangNode.yangType
			//Check if fields are valid
			if len(inParamsForGet.queryParams.fields) > 0 {
				flderr := validateAndFillQpFields(inParamsForGet)
				if flderr != nil {
					return jsonData, true, flderr
				}
			}

			//Check if the request depth is 1
			if inParamsForGet.queryParams.depthEnabled && inParamsForGet.queryParams.curDepth == 1 && (yangType == YANG_CONTAINER || yangType == YANG_LIST || yangType == YANG_MODULE) {
				return jsonData, true, err
			}

			/* Invoke pre-xfmr is present for the YANG module */
			moduleName := "/" + strings.Split(uri, "/")[1]
			xfmrLogInfo("Module name for URI %s is %s", uri, moduleName)
			if xYangModSpecMap != nil {
				if modSpecInfo, specOk := xYangModSpecMap[moduleName]; specOk && (len(modSpecInfo.xfmrPre) > 0) {
					inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
					inParams.queryParams = inParamsForGet.queryParams
					err = preXfmrHandlerFunc(modSpecInfo.xfmrPre, inParams)
					xfmrLogInfo("Invoked pre transformer: %v, dbDataMap: %v ", modSpecInfo.xfmrPre, dbDataMap)
					if err != nil {
						log.Warningf("Pre-transformer: %v failed.(err:%v)", modSpecInfo.xfmrPre, err)
						return jsonData, true, err
					}
					inParamsForGet.dbDataMap = dbDataMap
					inParamsForGet.ygRoot = ygRoot
				}
			}

			validateHandlerFlag := false
			tableXfmrFlag := false
			IsValidate := false

			if len(xYangSpecMap[xpathKeyExtRet.xpath].validateFunc) > 0 {
				inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
				inParams.queryParams = inParamsForGet.queryParams
				res := validateHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].validateFunc)
				inParamsForGet.dbDataMap = dbDataMap
				inParamsForGet.ygRoot = ygRoot
				if !res {
					validateHandlerFlag = true
					/* cannot immediately return from here since reXpath yangtype decides the return type */
				} else {
					IsValidate = res
				}
			}
			inParamsForGet.validate = IsValidate
			isList := false
			if strings.HasPrefix(requestUri, "/"+OC_MDL_PFX) {
				isOcMdl = true
			}
			switch yangType {
			case YANG_LIST:
				isList = true
			case YANG_LEAF, YANG_LEAF_LIST, YANG_CONTAINER:
				isList = false
			default:
				xfmrLogInfo("Unknown YANG object type for path %v", xpathKeyExtRet.xpath)
				isList = true //do not want non-list processing to happen
			}
			/*If yangtype is a list separate code path is to be taken in case of table transformer
			since that code path already handles the calling of table transformer and subsequent processing
			*/
			if (!validateHandlerFlag) && (!isList) {
				if xYangSpecMap[xpathKeyExtRet.xpath].xfmrTbl != nil {
					xfmrTblFunc := *xYangSpecMap[xpathKeyExtRet.xpath].xfmrTbl
					if len(xfmrTblFunc) > 0 {
						inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
						tblList, _ := xfmrTblHandlerFunc(xfmrTblFunc, inParams, inParamsForGet.xfmrDbTblKeyCache)
						inParamsForGet.dbDataMap = dbDataMap
						inParamsForGet.ygRoot = ygRoot
						if len(tblList) > 1 {
							log.Warningf("Table transformer returned more than one table for container %v", xpathKeyExtRet.xpath)
						}
						if len(tblList) == 0 {
							log.Warningf("Table transformer returned no table for conatiner %v", xpathKeyExtRet.xpath)
							tableXfmrFlag = true
						}
						if !tableXfmrFlag {
							for _, tbl := range tblList {
								dbDataFromTblXfmrGet(tbl, inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, xpathKeyExtRet.xpath)
								inParamsForGet.dbDataMap = dbDataMap
								inParamsForGet.ygRoot = ygRoot
							}

						}
					} else {
						log.Warningf("empty table transformer function name for xpath - %v", xpathKeyExtRet.xpath)
						tableXfmrFlag = true
					}
				}
			}

			for {
				done := true
				if yangType == YANG_LEAF || yangType == YANG_LEAF_LIST {
					yangEntry := getYangEntryForXPath(inParamsForGet.xpath)
					yangName := ""
					if yangEntry != nil {
						yangName = yangEntry.Name
						if validateHandlerFlag || tableXfmrFlag {
							resultMap[yangName] = ""
							break
						}
					}
					if len(xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc) > 0 {
						inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
						inParams.queryParams = inParamsForGet.queryParams
						xfmrFuncErr := xfmrHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc)
						if xfmrFuncErr != nil {
							/*For request Uri pointing to leaf/leaf-list having subtree, error will be propagated
														  to handle check of leaf/leaf-list-instance existence in DB , which will be performed
														  by subtree
							                                                          Error will also be propagated if QP Pruning API for subtree returns error
							*/
							_, qpSbtPruneErrOk := xfmrFuncErr.(*qpSubtreePruningErr)

							if qpSbtPruneErrOk {
								err = xfmrFuncErr
							} else {
								// propagate err from subtree callback
								fldSbtErr = xfmrFuncErr
							}
							inParamsForGet.ygRoot = ygRoot
							break
						}
						inParamsForGet.dbDataMap = dbDataMap
						inParamsForGet.ygRoot = ygRoot
					} else {
						tbl, key, _ := tableNameAndKeyFromDbMapGet((*dbDataMap)[cdb])
						inParamsForGet.tbl = tbl
						inParamsForGet.tblKey = key
						var fldValMap map[string]interface{}
						fldValMap, fldErr = terminalNodeProcess(inParamsForGet, true, yangEntry)
						if (fldErr != nil) || (len(fldValMap) == 0) {
							if fldErr == nil {
								if yangType == YANG_LEAF {
									xfmrLogInfo("Empty terminal node (\"%v\").", uri)
									fldErr = tlerr.NotFoundError{Format: "Resource Not found"}
								} else if (yangType == YANG_LEAF_LIST) && ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {
									jsonMapData, _ := json.Marshal(resultMap)
									jsonData = fmt.Sprintf("%v", string(jsonMapData))
									return jsonData, false, nil
								}
							}
						}
						resultMap = fldValMap
					}
					break

				} else if yangType == YANG_CONTAINER {
					cmap := make(map[string]interface{})
					resultMap = cmap
					if validateHandlerFlag || tableXfmrFlag {
						break
					}
					if len(xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc) > 0 {
						inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
						inParams.queryParams = inParamsForGet.queryParams
						err := xfmrHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc)
						if err != nil {
							qpSbtPruneErr, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
							if qpSbtPruneErrOk {
								err = tlerr.InternalError{Format: QUERY_PARAMETER_SBT_PRUNING_ERR, Path: qpSbtPruneErr.subtreePath}
							}
							return jsonData, true, err
						}
						inParamsForGet.dbDataMap = dbDataMap
						inParamsForGet.ygRoot = ygRoot
						if !xYangSpecMap[xpathKeyExtRet.xpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. Transfomer will not traverse subtree further", uri)
							break
						}
					}
					inParamsForGet.resultMap = make(map[string]interface{})
					err = yangDataFill(inParamsForGet, isOcMdl)
					if err != nil {
						xfmrLogInfo("Empty container(\"%v\").\r\n", uri)
					}
					resultMap = inParamsForGet.resultMap
					break
				} else if yangType == YANG_LIST {
					isFirstCall := true
					if len(xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc) > 0 {
						inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, "", dbDataMap, nil, nil, txCache)
						inParams.queryParams = inParamsForGet.queryParams
						err := xfmrHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc)
						if err != nil {
							qpSbtPruneErr, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
							if qpSbtPruneErrOk {
								err = tlerr.InternalError{Format: QUERY_PARAMETER_SBT_PRUNING_ERR, Path: qpSbtPruneErr.subtreePath}
								return jsonData, true, err
							}
							if ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) && (uri == requestUri) {
								// The error handling here is for the deferred resource check error being handled by the subtree for virtual table cases.
								log.Warningf("Subtree at list instance level returns error %v for  URI  - %v", err, uri)
								return jsonData, true, err

							} else {

								xfmrLogInfo("Error returned by %v: %v", xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc, err)
							}
						}
						isFirstCall = false
						inParamsForGet.dbDataMap = dbDataMap
						inParamsForGet.ygRoot = ygRoot
						if !xYangSpecMap[xpathKeyExtRet.xpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. Tranformer will not traverse subtree further", uri)
							break
						}
					}
					inParamsForGet.resultMap = make(map[string]interface{})
					err = yangListDataFill(inParamsForGet, isFirstCall, isOcMdl)
					if err != nil {
						xfmrLogInfo("yangListDataFill failed for list case(\"%v\").\r\n", uri)
					}
					resultMap = inParamsForGet.resultMap
					break
				} else {
					log.Warningf("Unknown YANG object type for path %v", xpathKeyExtRet.xpath)
					break
				}
				if done {
					break
				}
			} //end of for
		}
	}
	jsonMapData, _ := json.Marshal(resultMap)
	isEmptyPayload := isJsonDataEmpty(string(jsonMapData))
	jsonData = fmt.Sprintf("%v", string(jsonMapData))
	if fldSbtErr != nil {
		/*error should be propagated only when request Uri points to leaf/leaf-list-instance having subtree,
				  This is to handle check of leaf/leaf-list-instance existence in DB , which will be performed
		                  by subtree, and depending whether queried node exists or not subtree should return error
		*/
		return jsonData, isEmptyPayload, fldSbtErr
	}
	if fldErr != nil {
		/* error should be propagated only when request Uri points to leaf/leaf-list-instance and the data
		   is not available(via field-xfmr or field name)
		*/
		return jsonData, isEmptyPayload, fldErr
	}
	if err != nil {
		qpSbtPruneErr, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
		if qpSbtPruneErrOk {
			return jsonData, isEmptyPayload, tlerr.InternalError{Format: QUERY_PARAMETER_SBT_PRUNING_ERR, Path: qpSbtPruneErr.subtreePath}
		}
	}

	return jsonData, isEmptyPayload, nil
}
