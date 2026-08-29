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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

type typeMapOfInterface map[string]interface{}

type DbToYangXfmrInputArgs struct {
	InParamsForGet *xlateFromDbParams
	CurUri         string
	DbKey          string
	XfmrFuncName   string
}

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
		if entry == nil || entry.Type == nil {
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
							leafPath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[len(pathList)-1]
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
					xpath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[len(pathList)-1]
					if xpath == fldXpath {
						if sonicTblChldInfo, ok := xDbSpecMap[pathList[SONIC_TABLE_INDEX]+"/"+pathList[SONIC_TBL_CHILD_INDEX]]; ok {
							if sonicTblChldInfo.dbEntry != nil {
								entry = sonicTblChldInfo.dbEntry.Dir[pathList[len(pathList)-1]]
								if entry == nil && len(sonicTblChldInfo.listName) > 0 {
									// Leaf belongs to sonic inner list case
									innerListDbEntry := sonicTblChldInfo.dbEntry.Dir[pathList[SONIC_FIELD_INDEX]]
									if innerListDbEntry != nil && innerListDbEntry.IsList() {
										entry = innerListDbEntry.Dir[pathList[len(pathList)-1]]
									}
								}
								// Leaf in a regular list/singleton container case
								if entry != nil {
									yngTerminalNdDtType = entry.Type.Kind
								}
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
						leafPath = pathList[SONIC_TABLE_INDEX] + "/" + pathList[len(pathList)-1]
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

func DbToYangType(yngTerminalNdDtType yang.TypeKind, fldXpath string, dbFldVal string, oper Operation) (interface{}, interface{}, error) {
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
		if oper == GET {
			var resFloat64 float64 = float64(res.(uint64))
			resPtr = &resFloat64
			res = resFloat64
		} else {
			var resUint8 uint8 = uint8(res.(uint64))
			resPtr = &resUint8
		}
	case yang.Yuint16:
		res, err = DbValToInt(dbFldVal, INTBASE, 16, true)
		if oper == GET {
			var resFloat64 float64 = float64(res.(uint64))
			resPtr = &resFloat64
			res = resFloat64
		} else {
			var resUint16 uint16 = uint16(res.(uint64))
			resPtr = &resUint16
		}
	case yang.Yuint32:
		res, err = DbValToInt(dbFldVal, INTBASE, 32, true)
		if oper == GET {
			var resFloat64 float64 = float64(res.(uint64))
			resPtr = &resFloat64
			res = resFloat64
		} else {
			var resUint32 uint32 = uint32(res.(uint64))
			resPtr = &resUint32
		}
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
func processLfLstDbToYang(fieldXpath string, dbFldVal string, yngTerminalNdDtType yang.TypeKind, oper Operation) []interface{} {
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
			resVal, _, err := DbToYangType(yngTerminalNdDtType, fieldXpath, fldVal, oper)
			if err == nil {
				resLst = append(resLst, resVal)
			} else {
				log.Warningf("Failed to convert DB value type to YANG type for xpath %v. Field xfmr recommended if data types differ", fieldXpath)
			}
		}
	}
	return resLst
}

func sonicDbToYangTerminalNodeFill(field string, inParamsForGet xlateFromDbParams, dbEntry *yang.Entry, isNestedListEntry bool, isKeyLeaf bool) {
	resField := field
	value := ""

	if dbEntry == nil {
		log.Warningf("Yang entry is nil for xpath %v", inParamsForGet.xpath)
		return
	}

	if len(inParamsForGet.queryParams.fields) > 0 {
		curFldXpath := inParamsForGet.tbl + "/" + field
		if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[curFldXpath]; !ok {
			if !inParamsForGet.queryParams.fieldsFillAll {
				return
			}
		}
	}

	// Check if the terminal node is in the nested list case
	if isNestedListEntry {
		var fieldVal string
		valueExists := true
		innerListInstance := extractLeafValFromUriKey(inParamsForGet.uri, dbEntry.Parent.Key)
		tblInstFields, dbDataExists := (*inParamsForGet.dbDataMap)[inParamsForGet.curDb][inParamsForGet.tbl][inParamsForGet.tblKey]
		if dbDataExists {
			fieldVal, valueExists = tblInstFields.Field[innerListInstance]
			if !valueExists {
				xfmrLogInfo("Instance %v doesn't exist in table - %v, instance - %v", innerListInstance, inParamsForGet.tbl, inParamsForGet.tblKey)
				return
			}
		}
		if isKeyLeaf {
			value = innerListInstance
		} else {
			value = fieldVal
		}
	} else {
		if inParamsForGet.dbDataMap != nil {
			tblInstFields, dbDataExists := (*inParamsForGet.dbDataMap)[inParamsForGet.curDb][inParamsForGet.tbl][inParamsForGet.tblKey]
			if dbDataExists {
				if dbEntry.IsLeafList() {
					// The leaflist is stored with @ suffix in DB
					field = field + "@"
				}
				fieldVal, valueExists := tblInstFields.Field[field]
				if !valueExists {
					return
				}
				value = fieldVal
			} else {
				return
			}
		}
	}

	fieldXpath := inParamsForGet.tbl + "/" + resField
	yngTerminalNdDtType := dbEntry.Type.Kind
	if dbEntry.IsLeafList() {
		resLst := processLfLstDbToYang(fieldXpath, value, yngTerminalNdDtType, inParamsForGet.oper)
		inParamsForGet.resultMap[resField] = resLst
	} else { /* yangType is leaf - there are only 2 types of YANG terminal node leaf and leaf-list */
		resVal, _, err := DbToYangType(yngTerminalNdDtType, fieldXpath, value, inParamsForGet.oper)
		if err != nil {
			log.Warningf("Failed to convert DB value type to YANG type for xpath %v. Field xfmr recommended if data types differ", fieldXpath)
		} else {
			inParamsForGet.resultMap[resField] = resVal
		}
	}
}

func sonicDbToYangListFill(inParamsForGet xlateFromDbParams) ([]typeMapOfInterface, error) {
	var mapSlice []typeMapOfInterface
	var err error
	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err = tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return mapSlice, err
	}
	dbDataMap := inParamsForGet.dbDataMap
	table := inParamsForGet.tbl
	dbIdx := inParamsForGet.curDb
	xpath := inParamsForGet.xpath
	dbTblData := (*dbDataMap)[dbIdx][table]
	delKeyCnt := 0
	nestedListName := ""
	nestedListXpath := ""
	curUri := inParamsForGet.uri
	traverseNestedList := false

	if xDbSpecMap[xpath] != nil && xDbSpecMap[xpath].dbEntry != nil && xDbSpecMap[xpath].dbEntry.IsList() && len(xDbSpecMap[xpath].listName) > 0 {
		traverseNestedList = true
		// The xpath is already at table list level. Hence form inner list xpath before we start processing nested list
		// We only have one nested list entry for a given list. Hence accesing from index 0 entry
		nestedListName = xDbSpecMap[xpath].listName[0]
		nestedListXpath = xpath + "/" + nestedListName
		curUri = curUri + "/" + nestedListName

		if len(inParamsForGet.queryParams.fields) > 0 {
			if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[nestedListXpath]; ok {
				inParamsForGet.queryParams.fieldsFillAll = true
			} else if _, ok := inParamsForGet.queryParams.allowFieldsXpath[nestedListXpath]; !ok {
				if !inParamsForGet.queryParams.fieldsFillAll {
					for path := range inParamsForGet.queryParams.tgtFieldsXpathMap {
						if strings.HasPrefix(nestedListXpath, path) {
							inParamsForGet.queryParams.fieldsFillAll = true
						}
					}
					if !inParamsForGet.queryParams.fieldsFillAll {
						traverseNestedList = false
					}
				}
			}
		}

	}

	for keyStr, dbVal := range dbTblData {
		dbSpecData, ok := xDbSpecMap[table]
		if ok && dbSpecData != nil && dbSpecData.dbEntry != nil {
			if _, isSingletonContainer := dbSpecData.dbEntry.Dir[keyStr]; isSingletonContainer {
				xfmrLogDebug("DB key %v is a singleton container hence skip processing it during list processing.", keyStr)
				continue
			}
		}
		if ok && dbSpecData != nil && dbSpecData.keyName == nil && xDbSpecMap[xpath] != nil && xDbSpecMap[xpath].dbEntry != nil {
			curMap := make(map[string]interface{})
			yangKeys := yangKeyFromEntryGet(xDbSpecMap[xpath].dbEntry)
			sonicKeyDataAdd(dbIdx, yangKeys, table, xDbSpecMap[xpath].dbEntry.Name, keyStr, curMap, inParamsForGet.oper, false)
			if len(curMap) > 0 {
				linParamsForGet := formXlateFromDbParams(inParamsForGet.dbs[dbIdx], inParamsForGet.dbs, dbIdx, inParamsForGet.ygRoot, curUri, inParamsForGet.requestUri, xpath, inParamsForGet.oper, table, keyStr, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
				var nestedMapSlice []typeMapOfInterface
				if traverseNestedList {
					linParamsForGet.xpath = nestedListXpath
					if nestedMapSlice, err = sonicDbToYangNestedListDataFill(linParamsForGet); err != nil {
						return mapSlice, err
					}
					curMap[nestedListName] = nestedMapSlice
				} else {
					if err = sonicDbToYangDataFill(linParamsForGet); err != nil {
						return mapSlice, err
					}
					curMap = linParamsForGet.resultMap
				}

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
	return mapSlice, err
}

func sonicDbToYangNestedListDataFill(inParamsForGet xlateFromDbParams) ([]typeMapOfInterface, error) {
	var mapSlice []typeMapOfInterface
	var err error
	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err = tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return mapSlice, err
	}

	if inParamsForGet.queryParams.depthEnabled && (inParamsForGet.queryParams.curDepth < SONIC_NESTEDLIST_FIELD_INDEX) {
		return mapSlice, nil
	}

	dbDataMap := inParamsForGet.dbDataMap
	table := inParamsForGet.tbl
	dbKey := inParamsForGet.tblKey
	dbIdx := inParamsForGet.curDb
	xpath := inParamsForGet.xpath // Xpath is in the form of /Tbl/Outer-list/Inner-list
	dbTblData := (*dbDataMap)[dbIdx][table][dbKey]

	// Inner list always has one key
	// Key is a comman seperated strings for multi key case. For single key case it will be single string
	keyLeafYangName := xDbSpecMap[xpath].dbEntry.Key

	var requestedKey string
	// Check for resource existence if request is at nested list instance
	if (inParamsForGet.uri == inParamsForGet.requestUri) && (strings.HasSuffix(inParamsForGet.requestUri, "]") || strings.HasSuffix(inParamsForGet.requestUri, "]/")) {
		requestedKey = extractLeafValFromUriKey(inParamsForGet.requestUri, keyLeafYangName)
		if _, ok := dbTblData.Field[requestedKey]; !ok {
			xfmrLogInfo("Instance %v doesn't exist in table - %v, instance - %v", requestedKey, table, dbKey)
			return mapSlice, tlerr.NotFoundError{Format: "Resource not found"}
		}
	}

	nonKeyLeafYangName := ""
	// For Inner list case we always have 2 leafs
	for leaf := range xDbSpecMap[xpath].dbEntry.Dir {
		if leaf != keyLeafYangName {
			nonKeyLeafYangName = leaf
			break
		}
	}

	// Get the data type of the key and non key leaf
	yngTerminalKeyNdDtType := xDbSpecMap[xpath].dbEntry.Dir[keyLeafYangName].Type.Kind
	yngTerminalNonKeyNdDtType := xDbSpecMap[xpath].dbEntry.Dir[nonKeyLeafYangName].Type.Kind
	keyLeafXpath := table + "/" + keyLeafYangName
	nonKeyLeafXpath := table + "/" + nonKeyLeafYangName

	// If fields query parameter is enabled, key leaf is always filled and non key leaf can be checked if it needs t be filtered out.
	skipNonKeyLeafAdd := false
	if len(inParamsForGet.queryParams.fields) > 0 {
		if _, ok := inParamsForGet.queryParams.tgtFieldsXpathMap[nonKeyLeafXpath]; !ok {
			if !inParamsForGet.queryParams.fieldsFillAll {
				skipNonKeyLeafAdd = true
			}
		}
	}

	for field, value := range dbTblData.Field {
		// Skip NULL fields. We do not expect NULL fields in nested list.
		// This can happen only when the outer list instance is available in DB.
		if field == "NULL" {
			continue
		}
		if requestedKey != "" && field != requestedKey {
			continue
		}
		curMap := make(map[string]interface{})
		keyResVal, _, err := DbToYangType(yngTerminalKeyNdDtType, keyLeafXpath, field, inParamsForGet.oper)
		if err != nil {
			log.Warning("Conversion of DB value type to YANG type for field didn't happen. Field xpath", keyLeafXpath)
			curMap[keyLeafYangName] = field
		} else {
			curMap[keyLeafYangName] = keyResVal
		}
		if !skipNonKeyLeafAdd {
			nonKeyResVal, _, err := DbToYangType(yngTerminalNonKeyNdDtType, nonKeyLeafXpath, value, inParamsForGet.oper)
			if err != nil {
				log.Warning("Conversion of DB value type to YANG type for field didn't happen. Field xpath", nonKeyLeafXpath)
				curMap[keyLeafYangName] = value
			} else {
				curMap[nonKeyLeafYangName] = nonKeyResVal
			}
		}

		mapSlice = append(mapSlice, curMap)
	}
	return mapSlice, err
}

func sonicDbToYangDataFill(inParamsForGet xlateFromDbParams) error {
	var err error
	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err = tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return err
	}
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
					curUri := inParamsForGet.uri + "/" + yangChldName
					linParamsForGet := formXlateFromDbParams(nil, inParamsForGet.dbs, dbIdx, inParamsForGet.ygRoot, curUri, inParamsForGet.requestUri, chldXpath, inParamsForGet.oper, table, key, dbDataMap, inParamsForGet.txCache, resultMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
					dbEntry := yangNode.dbEntry.Dir[yangChldName]
					sonicDbToYangTerminalNodeFill(fldName, linParamsForGet, dbEntry, false, xDbSpecMap[chldXpath].isKey)
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
					// use table-name as xpath from now on
					d := inParamsForGet.dbs[xDbSpecMap[curTable].dbIndex]
					linParamsForGet := formXlateFromDbParams(d, inParamsForGet.dbs, xDbSpecMap[curTable].dbIndex, inParamsForGet.ygRoot, curUri, inParamsForGet.requestUri, chldXpath, inParamsForGet.oper, curTable, curKey, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
					if err = sonicDbToYangDataFill(linParamsForGet); err != nil {
						return err
					}
					curMap = linParamsForGet.resultMap
					dbDataMap = linParamsForGet.dbDataMap
					if _, ok := (*dbDataMap)[xDbSpecMap[curTable].dbIndex][curTable][curKey]; ok {
						delete((*dbDataMap)[xDbSpecMap[curTable].dbIndex][curTable], curKey)
						delete(inParamsForGet.dbTblKeyGetCache[dbIdx][table], curKey)
					}
					if curTable == chldXpath { // table-level container processing complete
						delete((*dbDataMap)[dbIdx], table)
						delete(inParamsForGet.dbTblKeyGetCache[dbIdx], table)
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

					if mapSlice, err = sonicDbToYangListFill(inParamsForGet); err != nil {
						return err
					}
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
					if err = sonicDbToYangDataFill(inParamsForGet); err != nil {
						return err
					}
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
	return nil
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
	isNestedListCase := false

	traverse := true
	if inParamsForGet.queryParams.depthEnabled {
		pathList := strings.Split(xpath, "/")
		// Depth at requested URI starts at 1. Hence reduce the reqDepth calculated by 1
		// The requested depth is the depth calculated for the whole yang tree from root.
		// Current depth is the depth of the requested URI.
		reqDepth := (len(pathList) - 1) + int(inParamsForGet.queryParams.curDepth) - 1
		xfmrLogInfo("xpath: %v ,Sonic Yang len(pathlist) %v, reqDepth %v, sonic Field Index %v", xpath, len(pathList), reqDepth, SONIC_FIELD_INDEX)
		if reqDepth < SONIC_FIELD_INDEX {
			traverse = false
		}
		// Reset the curDepth to the requested depth which gives the depth from root to be processed.
		// Used for nested list case only. Other cases should be identified through the traverse flag.
		inParamsForGet.queryParams.curDepth = uint(reqDepth)
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
			// Check if request is at nested list
			dbTblChldNode, ok := xDbSpecMap[table+"/"+tokens[SONIC_TBL_CHILD_INDEX]]
			if ok && dbTblChldNode.dbEntry != nil && dbTblChldNode.dbEntry.IsList() && len(dbTblChldNode.listName) > 0 {
				if dbTblChldNode.dbEntry.Dir[tokens[SONIC_FIELD_INDEX]].IsList() {
					isNestedListCase = true
				}
			}
			if len(tokens) > SONIC_NESTEDLIST_FIELD_INDEX {
				// For nestedList request at leaf/leaflist level set the xpath accordingly
				xpath = table + "/" + tokens[SONIC_NESTEDLIST_FIELD_INDEX]
				fieldName = tokens[SONIC_NESTEDLIST_FIELD_INDEX]
				xfmrLogDebug("Request is at terminal node (leaf/leaf-list) for nested list - %v.", uri)
			} else if isNestedListCase {
				// Request is at  nested list level
				xpath = table + "/" + tokens[SONIC_TBL_CHILD_INDEX] + "/" + tokens[SONIC_FIELD_INDEX]
			} else {
				// Request is at  level leaf/leaflist level
				xpath = table + "/" + tokens[SONIC_FIELD_INDEX]
				fieldName = tokens[SONIC_FIELD_INDEX]
			}
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

		if ok && dbNode != nil {
			cdb := db.ConfigDB
			yangType := dbNode.yangType
			if len(table) > 0 {
				cdb = xDbSpecMap[table].dbIndex
			}
			inParamsForGet.curDb = cdb

			if yangType == YANG_LEAF || yangType == YANG_LEAF_LIST {
				dbEntry := getYangEntryForXPath(inParamsForGet.xpath)
				if dbEntry == nil {
					return "", true, tlerr.InternalError{Format: "yangEntry not found. Unable to process", Path: inParamsForGet.xpath}
				}
				linParamsForGet := formXlateFromDbParams(nil, inParamsForGet.dbs, cdb, inParamsForGet.ygRoot, uri, inParamsForGet.requestUri, xpath, inParamsForGet.oper, table, key, dbDataMap, inParamsForGet.txCache, resultMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
				sonicDbToYangTerminalNodeFill(fieldName, linParamsForGet, dbEntry, isNestedListCase, dbNode.isKey)
				if isNestedListCase && len(linParamsForGet.resultMap) == 0 {
					return "", true, tlerr.NotFoundError{Format: "Resource not found"}
				}
				resultMap = linParamsForGet.resultMap
			} else if yangType == YANG_CONTAINER {
				if err = sonicDbToYangDataFill(inParamsForGet); err != nil {
					return "", true, err
				}
				resultMap = inParamsForGet.resultMap
			} else if yangType == YANG_LIST {
				var mapSlice []typeMapOfInterface
				if isNestedListCase {
					mapSlice, err = sonicDbToYangNestedListDataFill(inParamsForGet)
				} else {
					mapSlice, err = sonicDbToYangListFill(inParamsForGet)
				}
				if err != nil {
					return "", true, err
				}
				if len(key) > 0 && len(mapSlice) == 1 && strings.HasSuffix(inParamsForGet.requestUri, "]") {
					// Single instance query at target Uri (either outer list or inner list). Don't return array of maps
					for k, val := range mapSlice[0] {
						resultMap[k] = val
					}

				} else {
					if len(mapSlice) > 0 {
						pathl := strings.Split(xpath, "/")
						lname := pathl[len(pathl)-1]
						resultMap[lname] = mapSlice
					}
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

func tableNameAndKeyFromDbMapGet(dbDataMap map[string]map[string]db.Value, tblName string) (string, string, error) {
	tableName := ""
	tableKey := ""
	for tn, tblData := range dbDataMap {
		tableName = tn
		for kname := range tblData {
			tableKey = kname
		}
		if tblName == tableName {
			break
		}
	}
	return tableName, tableKey, nil
}

func fillDbDataMapForTbl(uri string, xpath string, tblName string, tblKey string, cdb db.DBNum, dbs [db.MaxDB]*db.DB,
	dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, reqCtxt context.Context) (map[db.DBNum]map[string]map[string]db.Value, error) {
	var err error
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
	dbresult := make(RedisDbMap)
	dbresult[cdb] = make(map[string]map[string]db.Value)
	err = TraverseDb(dbs, dbFormat, &dbresult, nil, dbTblKeyGetCache, reqCtxt)
	if err != nil {
		xfmrLogDebug("Didn't fetch DB data for tbl(DB num) %v(%v) for xpath %v", tblName, cdb, xpath)
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
func dbDataFromTblXfmrGet(tbl string, inParams XfmrParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value,
	dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, xpath string, reqCtxt context.Context) error {
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
		curDbDataMap, err := fillDbDataMapForTbl(inParams.uri, xpath, tbl, inParams.key,
			inParams.curDb, inParams.dbs, dbTblKeyGetCache, reqCtxt)
		if err == nil {
			mapCopy((*dbDataMap)[inParams.curDb], curDbDataMap[inParams.curDb])
		}
	}
	return nil
}

func yangListDataFill(inParamsForGet xlateFromDbParams, isFirstCall bool, isOcMdl bool) error {
	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err := tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return err
	}
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
	tblDelList := make(map[string]bool)
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
						if _, ok := (*dbDataMap)[cdb][curTbl]; !ok {
							tblDelList[curTbl] = true
						}
						dbDataFromTblXfmrGet(curTbl, inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, xpath, inParamsForGet.reqCtxt)
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
					if isReqContextCancelledError(err) {
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

	uriPathList := SplitPath(inParamsForGet.relUri)
	parentUriPath := ""
	if len(uriPathList) > 0 {
		parentUriPath = strings.Join(uriPathList[:len(uriPathList)-1], "/")
	}
	ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygParentObj, relUri: parentUriPath, ygSchema: inParamsForGet.ygSchema}
	if len(parentUriPath) > 0 {
		err := ygotXlator{&ygotCtx}.translate()
		if err != nil {
			log.Warningf("yangListDataFill: error in unmarshaling the parent uri: %v; error: %v; "+
				"parent obj: %v", parentUriPath, err, reflect.TypeOf(*inParamsForGet.ygParentObj))
			return err
		} else {
			inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
			inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
			inParamsForGet.relUri = uriPathList[len(uriPathList)-1]
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
				if isReqContextCancelledError(err) {
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
			delKeyCnt := 0
			for dbKey, dbVal := range tblData {
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
					if isReqContextCancelledError(err) {
						return err
					}
				} else if (instMap != nil) && (len(instMap) > 0) {
					mapSlice = append(mapSlice, instMap)
					if tblDelList[tbl] {
						tblData[dbKey] = db.Value{}
						delete(inParamsForGet.dbTblKeyGetCache[cdb][tbl], dbKey)
						delKeyCnt++
					}
				} else if len(dbVal.Field) == 0 {
					delKeyCnt++
				}
			}
			if tblDelList[tbl] && len(tblData) == delKeyCnt {
				delete((*dbDataMap)[cdb], tbl)
				delete(inParamsForGet.dbTblKeyGetCache[cdb], tbl)
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
	var curMap typeMapOfInterface

	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err := tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return curMap, err
	}

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

	curKeyMap, curUri, keyUriPath, err := dbKeyToYangDataConvert(uri, requestUri, xpath, tbl, dbs[cdb], dbs, dbDataMap, dbKey, dbs[cdb].Opts.KeySeparator, txCache, inParamsForGet.oper)
	inParamsForGet.relUri += keyUriPath

	if (err != nil) || (curKeyMap == nil) || (len(curKeyMap) == 0) {
		xfmrLogDebug("Skip filling list instance for URI %v since no YANG  key found corresponding to db-key %v", uri, dbKey)
		return curMap, err
	}
	parentXpath := parentXpathGet(xpath)
	_, ok := xYangSpecMap[xpath]
	_, parentOk := xYangSpecMap[parentXpath]
	if ok && len(xYangSpecMap[xpath].xfmrFunc) > 0 {
		if isFirstCall || (!isFirstCall && (uri != requestUri) && ((len(xYangSpecMap[parentXpath].xfmrFunc) == 0) ||
			(len(xYangSpecMap[parentXpath].xfmrFunc) > 0 && (xYangSpecMap[parentXpath].xfmrFunc != xYangSpecMap[xpath].xfmrFunc)))) {
			xfmrLogDebug("Parent subtree already handled cur uri: %v", xpath)
			ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygParentObj, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
			inArgs := DbToYangXfmrInputArgs{&inParamsForGet, curUri, dbKey, xYangSpecMap[xpath].xfmrFunc}
			err := executeDbToYangHandler(&inArgs, &ygotCtx)
			if err != nil {
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return curMap, err
				}
				if isReqContextCancelledError(err) {
					return curMap, err
				}
				if ygotCtx.err != nil {
					log.Warningf("yangListInstanceDataFill: error in unmarshalling the URI: %v, relUri: %v, "+
						"ygNode: %v, ygot parent obj: %v; error: %v", curUri, inParamsForGet.relUri, inParamsForGet.ygSchema.Name,
						reflect.TypeOf(*inParamsForGet.ygParentObj), ygotCtx.err)
					return curMap, ygotCtx.err
				}
			}
			if ygotCtx.trgtYgObj != nil {
				inParamsForGet.relUri = ""
				inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
				inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
			}
		}
		if xYangSpecMap[xpath].hasChildSubTree {
			curMap = make(map[string]interface{})
			linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, xpath, inParamsForGet.oper,
				tbl, dbKey, dbDataMap, inParamsForGet.txCache, curMap,
				inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
			linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
			linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
			linParamsForGet.ygParentObj = inParamsForGet.ygParentObj
			linParamsForGet.ygSchema = inParamsForGet.ygSchema
			linParamsForGet.relUri = inParamsForGet.relUri
			if len(linParamsForGet.relUri) > 0 {
				ygotCtx := ygotUnMarshalCtx{ygParentObj: linParamsForGet.ygParentObj, relUri: linParamsForGet.relUri, ygSchema: linParamsForGet.ygSchema}
				err := ygotXlator{&ygotCtx}.translate()
				if err != nil {
					log.Warningf("yangListInstanceDataFill: error in unmarshalling the URI: %v, relUri: %v, "+
						"ygNode: %v, ygot parent obj: %v; error: %v", curUri, linParamsForGet.relUri, linParamsForGet.ygSchema.Name,
						reflect.TypeOf(*linParamsForGet.ygParentObj), err)
					return nil, err
				}
				if ygotCtx.trgtYgObj != nil {
					linParamsForGet.relUri = ""
					linParamsForGet.ygParentObj = ygotCtx.trgtYgObj
					linParamsForGet.ygSchema = ygotCtx.trgtYgSchema
				}
			}
			err := yangDataFill(linParamsForGet, isOcMdl)
			if err != nil {
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return nil, err
				}
				if isReqContextCancelledError(err) {
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
		xpathKeyExtRet, _ := xpathKeyExtractForGet(dbs[cdb], ygRoot, GET, curUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache, dbs)
		keyFromCurUri := xpathKeyExtRet.dbKey
		inParamsForGet.ygRoot = ygRoot
		var listKeyMap map[string]interface{}
		if dbKey == keyFromCurUri || keyFromCurUri == "" {
			curMap = make(map[string]interface{})
			if ok && parentOk && (len(xYangSpecMap[xpath].validateFunc) > 0) && (xYangSpecMap[xpath].validateFunc != xYangSpecMap[parentXpath].validateFunc) {
				inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
				res := validateHandlerFunc(inParams, xYangSpecMap[xpath].validateFunc)
				if !res {
					xfmrLogDebug("Further traversal not needed. Validate xfmr returns false for URI %v", curUri)
					return nil, nil
				}
			}

			if dbKey == keyFromCurUri {
				listKeyMap = make(map[string]interface{})
				for k, kv := range curKeyMap {
					curMap[k] = kv
					listKeyMap[k] = kv
				}
			}
			linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, xpathKeyExtRet.xpath, inParamsForGet.oper, tbl, dbKey, dbDataMap, inParamsForGet.txCache, curMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, listKeyMap)
			linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
			linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
			linParamsForGet.ygParentObj = inParamsForGet.ygParentObj
			linParamsForGet.ygSchema = inParamsForGet.ygSchema
			linParamsForGet.relUri = inParamsForGet.relUri
			if len(linParamsForGet.relUri) > 0 {
				ygotCtx := ygotUnMarshalCtx{ygParentObj: linParamsForGet.ygParentObj, relUri: linParamsForGet.relUri, ygSchema: linParamsForGet.ygSchema}
				err := ygotXlator{&ygotCtx}.translate()
				if err != nil {
					log.Warningf("yangListInstanceDataFill: error in unmarshalling the URI: %v, relUri: %v, "+
						"ygNode: %v, ygot parent obj: %v; error: %v", curUri, linParamsForGet.relUri, linParamsForGet.ygSchema.Name,
						reflect.TypeOf(*linParamsForGet.ygParentObj), err)
					return curMap, err
				}
				if ygotCtx.trgtYgObj != nil {
					linParamsForGet.relUri = ""
					linParamsForGet.ygParentObj = ygotCtx.trgtYgObj
					linParamsForGet.ygSchema = ygotCtx.trgtYgSchema
				}
			}
			err := yangDataFill(linParamsForGet, isOcMdl)
			if err != nil {
				// abort GET request if QP Subtree Pruning API returns error
				_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
				if qpSbtPruneErrOk {
					return curMap, err
				}
				if isReqContextCancelledError(err) {
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
	var resFldValMap map[string]interface{}
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
		if len(fldValMap) > 0 {
			resFldValMap = make(map[string]interface{})
			for lf, val := range fldValMap {
				resFldValMap[lf] = val
			}
		}
	} else {
		dbFldName := xYangSpecMap[xpath].fieldName
		if dbFldName == XFMR_NONE_STRING {
			return resFldValMap, err
		}
		fillLeafFromUriKey := false
		yangDataType := yangEntry.Type.Kind
		_, dbKeyExist := (*dbDataMap)[cdb][tbl][tblKey]
		if (terminalNodeQuery || dbKeyExist) && xYangSpecMap[xpath].isKey { //GET request for list key-leaf(direct child of list)
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
						resLst := processLfLstDbToYang(xpath, val, yangDataType, inParamsForGet.oper)
						resFldValMap = make(map[string]interface{})
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
					resVal, _, err := DbToYangType(yangDataType, xpath, val, inParamsForGet.oper)
					if err != nil {
						log.Warning("Conversion of DB value type to YANG type for field didn't happen. Field-xfmr recommended if data types differ. Field xpath", xpath)
					} else {
						resFldValMap = make(map[string]interface{})
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
				resVal, _, err := DbToYangType(yangDataType, xpath, val, inParamsForGet.oper)
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
	if isReqContextCancelled(inParamsForGet.reqCtxt) {
		err := tlerr.RequestContextCancelled("Client request's context cancelled.", inParamsForGet.reqCtxt.Err())
		log.Warningf(err.Error())
		return err
	}
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	uri := inParamsForGet.uri
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	resultMap := inParamsForGet.resultMap
	xpath := inParamsForGet.xpath
	var chldUri string

	log.V(5).Infof("yangDataFill: uri: %v; parent obj: %v; inParamsForGet.relUri: %v; xpath: %v",
		inParamsForGet.uri, reflect.TypeOf(*inParamsForGet.ygParentObj), inParamsForGet.relUri, xpath)

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

		if len(inParamsForGet.relUri) > 0 {
			ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygParentObj, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
			err := ygotXlator{&ygotCtx}.translate()
			if err != nil {
				log.Warningf("error in unmarshalling the uri: %v; error: %v; parent obj: %v; schema: %v",
					inParamsForGet.relUri, err, reflect.TypeOf(*inParamsForGet.ygParentObj), inParamsForGet.ygSchema.Name)
				return err
			} else if ygotCtx.trgtYgObj != nil {
				inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
				inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
				inParamsForGet.relUri = ""
			}
			log.V(5).Infof("yangDataFill: after uri marshalling: uri: %v; parent obj: %v", inParamsForGet.uri, reflect.TypeOf(*inParamsForGet.ygParentObj))
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
			inParamsForGet.relUri = ""
			if xYangSpecMap[chldXpath] != nil && yangNode.yangEntry.Dir[yangChldName] != nil {
				chldYangType := xYangSpecMap[chldXpath].yangType
				if chldYangType == YANG_CONTAINER || chldYangType == YANG_LIST {
					inParamsForGet.relUri = "/" + yangChldName
					log.V(5).Infof("yangDataFill: About to process URI : %v, chldYangType: %v; inParamsForGet.relUri: %v, validate-handler-name: %v",
						chldUri, getYangTypeStrId(chldYangType), inParamsForGet.relUri, xYangSpecMap[chldXpath].validateFunc)
				}
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
				if (len(xYangSpecMap[chldXpath].validateFunc) > 0) && (xYangSpecMap[chldXpath].validateFunc != xYangSpecMap[xpath].validateFunc) && (chldYangType != YANG_LIST) {
					xpathKeyExtRet, _ := xpathKeyExtractForGet(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache, dbs)
					inParamsForGet.ygRoot = ygRoot
					// TODO - handle non CONFIG-DB
					inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
					inParams.queryParams = inParamsForGet.queryParams
					res := validateHandlerFunc(inParams, xYangSpecMap[chldXpath].validateFunc)
					if !res {
						xfmrLogDebug("Further traversal not needed. Validate xfmr returns false for URI %v", chldUri)
						continue
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
					var jsonDataMap map[string]interface{}
					if len(fldValMap) > 0 {
						jsonDataTmp, _ := json.Marshal(fldValMap)
						errJs := json.Unmarshal(jsonDataTmp, &jsonDataMap)
						if errJs != nil {
							log.Warningf("yangDataFill: Failed to unmarshall json data: %v for the uri : %v; error: %v", jsonDataTmp, chldUri, errJs)
						} else {
							fldValMap = jsonDataMap
						}
					}
					for lf, val := range fldValMap {
						resultMap[lf] = val
					}
					inParamsForGet.resultMap = resultMap
					if len(fldValMap) > 0 {
						if err := ytypes.Unmarshal(inParamsForGet.ygSchema, *inParamsForGet.ygParentObj, fldValMap); err != nil {
							log.Warningf("yangDataFill: error in object unmarshalling: %v; schema: %v; parent obj: %v, "+
								"resFldValMap: %v", err, inParamsForGet.ygSchema.Name, reflect.TypeOf(*inParamsForGet.ygParentObj), fldValMap)
							return err
						}
					}
				} else if chldYangType == YANG_CONTAINER {
					xpathKeyExtRet, _ := xpathKeyExtractForGet(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache, dbs)
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
							curDbDataMap, err := fillDbDataMapForTbl(chldUri, chldXpath, chtbl, tblKey, cdb, dbs,
								inParamsForGet.dbTblKeyGetCache, inParamsForGet.reqCtxt)
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
							dbDataFromTblXfmrGet(tblList[0], inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, chldXpath, inParamsForGet.reqCtxt)
							inParamsForGet.dbDataMap = dbDataMap
							inParamsForGet.ygRoot = ygRoot
							chtbl = tblList[0]
						}
					}
					ygTrgtParentObj := inParamsForGet.ygParentObj
					ygTrgtSchema := inParamsForGet.ygSchema
					chldNodeUri := inParamsForGet.relUri
					if len(xYangSpecMap[chldXpath].xfmrFunc) > 0 {
						if (len(xYangSpecMap[xpath].xfmrFunc) == 0) ||
							(len(xYangSpecMap[xpath].xfmrFunc) > 0 &&
								(xYangSpecMap[xpath].xfmrFunc != xYangSpecMap[chldXpath].xfmrFunc)) {
							ygotCtx := ygotUnMarshalCtx{ygParentObj: ygTrgtParentObj, relUri: chldNodeUri, ygSchema: ygTrgtSchema}
							inArgs := DbToYangXfmrInputArgs{&inParamsForGet, chldUri, "", xYangSpecMap[chldXpath].xfmrFunc}
							err := executeDbToYangHandler(&inArgs, &ygotCtx)
							if err != nil {
								// abort GET request if QP Subtree Pruning API returns error
								_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
								if qpSbtPruneErrOk {
									return err
								}
								if isReqContextCancelledError(err) {
									return err
								}
								if ygotCtx.err != nil {
									return ygotCtx.err
								}
							}
							if ygotCtx.trgtYgObj != nil {
								ygTrgtParentObj = ygotCtx.trgtYgObj
								ygTrgtSchema = ygotCtx.trgtYgSchema
								chldNodeUri = ""
							}
						}
						if !xYangSpecMap[chldXpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. xfmr will not traverse subtree further", chldUri)
							continue
						}
					}
					cmap2 := make(map[string]interface{})
					linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, chldXpath, inParamsForGet.oper, chtbl, tblKey, dbDataMap, inParamsForGet.txCache, cmap2, inParamsForGet.queryParams, inParamsForGet.reqCtxt, inParamsForGet.listKeysMap)
					linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
					linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
					linParamsForGet.ygParentObj = ygTrgtParentObj
					linParamsForGet.relUri = chldNodeUri
					linParamsForGet.ygSchema = ygTrgtSchema
					linParamsForGet.queryParams.fieldsFillAll = chFieldsFillAll
					err = yangDataFill(linParamsForGet, isOcMdl)
					if err != nil {
						// abort GET request if QP Subtree Pruning API returns error
						_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
						if qpSbtPruneErrOk {
							return err
						}
						if isReqContextCancelledError(err) {
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
					xpathKeyExtRet, xerr := xpathKeyExtractForGet(dbs[cdb], ygRoot, GET, chldUri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache, dbs)
					if xerr != nil && len(xpathKeyExtRet.dbKey) == 0 {
						log.Warningf("yangDataFill: could not convert yang key into db key, error: %v; chldUri: %v", xerr.Error(), chldUri)
						continue
					}
					inParamsForGet.ygRoot = ygRoot
					cdb = xYangSpecMap[chldXpath].dbIndex
					inParamsForGet.curDb = cdb
					ygTrgtParentObj := inParamsForGet.ygParentObj
					ygTrgtSchema := inParamsForGet.ygSchema
					chldNodeUri := inParamsForGet.relUri
					if len(xYangSpecMap[chldXpath].xfmrFunc) > 0 {
						if (len(xYangSpecMap[xpath].xfmrFunc) == 0) ||
							(len(xYangSpecMap[xpath].xfmrFunc) > 0 &&
								(xYangSpecMap[xpath].xfmrFunc != xYangSpecMap[chldXpath].xfmrFunc)) {
							ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygParentObj, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
							inArgs := DbToYangXfmrInputArgs{&inParamsForGet, chldUri, "", xYangSpecMap[chldXpath].xfmrFunc}
							err := executeDbToYangHandler(&inArgs, &ygotCtx)
							if err != nil {
								// abort GET request if QP Subtree Pruning API returns error
								_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
								if qpSbtPruneErrOk {
									return err
								}
								if isReqContextCancelledError(err) {
									return err
								}
								if ygotCtx.err != nil {
									return ygotCtx.err
								}
							}
							if ygotCtx.trgtYgObj != nil {
								ygTrgtParentObj = ygotCtx.trgtYgObj
								ygTrgtSchema = ygotCtx.trgtYgSchema
								chldNodeUri = ""
							}
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
						curDbDataMap, err := fillDbDataMapForTbl(chldUri, chldXpath, lTblName, "", cdb, dbs,
							inParamsForGet.dbTblKeyGetCache, inParamsForGet.reqCtxt)
						if err == nil {
							mapCopy((*dbDataMap)[cdb], curDbDataMap[cdb])
							inParamsForGet.dbDataMap = dbDataMap
						}
					}
					linParamsForGet := formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, chldUri, requestUri, chldXpath,
						inParamsForGet.oper, lTblName, xpathKeyExtRet.dbKey, dbDataMap, inParamsForGet.txCache,
						resultMap, inParamsForGet.queryParams, inParamsForGet.reqCtxt, nil)
					linParamsForGet.xfmrDbTblKeyCache = inParamsForGet.xfmrDbTblKeyCache
					linParamsForGet.dbTblKeyGetCache = inParamsForGet.dbTblKeyGetCache
					linParamsForGet.ygParentObj = ygTrgtParentObj
					linParamsForGet.ygSchema = ygTrgtSchema
					linParamsForGet.relUri = chldNodeUri
					err := yangListDataFill(linParamsForGet, false, isOcMdl)
					if err != nil {
						// abort GET request if QP Subtree Pruning API returns error
						_, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
						if qpSbtPruneErrOk {
							return err
						}
						if isReqContextCancelledError(err) {
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
						if isReqContextCancelledError(err) {
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
	} else {
		log.Warningf("yangDataFill: xpath info or yang.Entry not found for the xpath: %v; uri: %v", xpath, uri)
	}
	return err
}

func executeDbToYangHandler(inputArgs *DbToYangXfmrInputArgs, ygotCtxt *ygotUnMarshalCtx) error {
	inParamsForGet := inputArgs.InParamsForGet
	curUri := inputArgs.CurUri
	dbKey := inputArgs.DbKey
	xfmrFuncName := inputArgs.XfmrFuncName
	dbs := inParamsForGet.dbs
	ygRoot := inParamsForGet.ygRoot
	requestUri := inParamsForGet.requestUri
	dbDataMap := inParamsForGet.dbDataMap
	txCache := inParamsForGet.txCache
	cdb := inParamsForGet.curDb

	inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, curUri, requestUri, GET, dbKey, dbDataMap, nil, nil, txCache)
	inParams.queryParams = inParamsForGet.queryParams
	inParams.ctxt = inParamsForGet.reqCtxt
	err := xfmrHandlerFunc(inParams, xfmrFuncName, ygotCtxt)

	inParamsForGet.dbDataMap = dbDataMap
	inParamsForGet.ygRoot = ygRoot

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
	inParamsForGet.ygSchema = ocbSch.RootSchema()
	inParamsForGet.relUri = inParamsForGet.uri
	inParamsForGet.ygParentObj = ygRoot

	if isSonicYang(uri) {
		return directDbToYangJsonCreate(inParamsForGet)
	} else {
		xpathKeyExtRet, _ := xpathKeyExtractForGet(d, ygRoot, GET, uri, requestUri, dbDataMap, nil, txCache, inParamsForGet.xfmrDbTblKeyCache, dbs)

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

			if len(xYangSpecMap[xpathKeyExtRet.xpath].validateFunc) > 0 {
				inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, xpathKeyExtRet.dbKey, dbDataMap, nil, nil, txCache)
				inParams.queryParams = inParamsForGet.queryParams
				res := validateHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].validateFunc)
				inParamsForGet.dbDataMap = dbDataMap
				inParamsForGet.ygRoot = ygRoot
				if !res {
					validateHandlerFlag = true
					xfmrLogDebug("Further traversal not required for this node since validate-handler evaluated to false - %v", uri)
					/* cannot immediately return from here since reXpath yangtype decides the return type */
				}
			}
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
								dbDataFromTblXfmrGet(tbl, inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, xpathKeyExtRet.xpath, inParamsForGet.reqCtxt)
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
						ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygRoot, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
						inArgs := DbToYangXfmrInputArgs{&inParamsForGet, uri, "", xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc}
						xfmrFuncErr := executeDbToYangHandler(&inArgs, &ygotCtx)
						if xfmrFuncErr != nil {
							/*For request Uri pointing to leaf/leaf-list having subtree, error will be propagated
							  to handle check of leaf/leaf-list-instance existence in DB , which will be performed
							  by subtree
							  Error will also be propagated if QP Pruning API for subtree returns error
							*/
							_, qpSbtPruneErrOk := xfmrFuncErr.(*qpSubtreePruningErr)

							if qpSbtPruneErrOk || isReqContextCancelledError(xfmrFuncErr) || ygotCtx.err != nil {
								err = xfmrFuncErr
							} else {
								// propagate err from subtree callback
								fldSbtErr = xfmrFuncErr
							}
							inParamsForGet.ygRoot = ygRoot
							break
						} else if (yangType == YANG_LEAF_LIST) && ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {
							/*GET on leaf-list instance with subtree annotatiion.
							  Translib pre-populates ygRoot with leaf-list instance specified in GET URL
							*/
							jsonMapData, _ := json.Marshal(resultMap)
							jsonData = fmt.Sprintf("%v", string(jsonMapData))
							return jsonData, false, nil
						}
					} else {
						tbl, key, _ := tableNameAndKeyFromDbMapGet((*dbDataMap)[cdb], xpathKeyExtRet.tableName)
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
						var jsonDataMap map[string]interface{}
						if len(fldValMap) > 0 {
							jsonDataTmp, _ := json.Marshal(fldValMap)
							errJs := json.Unmarshal(jsonDataTmp, &jsonDataMap)
							if errJs != nil {
								log.Warningf("dbDataToYangJsonCreate: Failed to unmarshall json data: %v for the uri : %v; error: %v", jsonDataTmp, uri, errJs)
								return string(jsonDataTmp), false, errJs
							} else {
								fldValMap = jsonDataMap
							}
						}
						resultMap = fldValMap
						if len(fldValMap) > 0 {
							uriPathList := SplitPath(inParamsForGet.relUri)
							parentUriPath := ""
							if len(uriPathList) > 0 {
								parentUriPath = strings.Join(uriPathList[:len(uriPathList)-1], "/")
							}
							ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygParentObj, relUri: parentUriPath, ygSchema: inParamsForGet.ygSchema}
							if len(parentUriPath) > 0 {
								err := ygotXlator{&ygotCtx}.translate()
								if err != nil {
									log.Warningf("dbDataToYangJsonCreate: error in unmarshalling the URI: %v, relUri: %v, "+
										"ygNode: %v, ygot parent obj: %v; inParamsForGet.relUri: %v; error: %v", inParamsForGet.uri, parentUriPath, inParamsForGet.ygSchema.Name,
										reflect.TypeOf(*inParamsForGet.ygParentObj), inParamsForGet.relUri, err)
									return "", true, err
								} else if ygotCtx.trgtYgObj != nil {
									inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
									inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
									inParamsForGet.relUri = uriPathList[len(uriPathList)-1]
								}
							}
							if err := ytypes.Unmarshal(inParamsForGet.ygSchema, *inParamsForGet.ygParentObj, fldValMap); err != nil {
								log.Warningf("yangDataFill: error in object unmarshalling: %v; schema: %v; parent obj: %v, "+
									"resFldValMap: %v", inParamsForGet.ygSchema.Name, reflect.TypeOf(*inParamsForGet.ygParentObj), fldValMap)
								return "", true, err
							}
						}
					}
					break

				} else if yangType == YANG_CONTAINER {
					cmap := make(map[string]interface{})
					resultMap = cmap
					if validateHandlerFlag || tableXfmrFlag {
						break
					}
					if len(xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc) > 0 {
						ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygRoot, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
						inArgs := DbToYangXfmrInputArgs{&inParamsForGet, uri, "", xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc}
						err := executeDbToYangHandler(&inArgs, &ygotCtx)
						if err != nil {
							qpSbtPruneErr, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
							if qpSbtPruneErrOk {
								err = tlerr.InternalError{Format: QUERY_PARAMETER_SBT_PRUNING_ERR, Path: qpSbtPruneErr.subtreePath}
							}
							return jsonData, true, err
						}
						if !xYangSpecMap[xpathKeyExtRet.xpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. Transfomer will not traverse subtree further", uri)
							break
						}
						if ygotCtx.trgtYgObj != nil {
							inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
							inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
							inParamsForGet.relUri = ""
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
						ygotCtx := ygotUnMarshalCtx{ygParentObj: inParamsForGet.ygRoot, relUri: inParamsForGet.relUri, ygSchema: inParamsForGet.ygSchema}
						inArgs := DbToYangXfmrInputArgs{&inParamsForGet, uri, "", xYangSpecMap[xpathKeyExtRet.xpath].xfmrFunc}
						err := executeDbToYangHandler(&inArgs, &ygotCtx)
						if err != nil {
							qpSbtPruneErr, qpSbtPruneErrOk := err.(*qpSubtreePruningErr)
							if qpSbtPruneErrOk {
								err = tlerr.InternalError{Format: QUERY_PARAMETER_SBT_PRUNING_ERR, Path: qpSbtPruneErr.subtreePath}
								return jsonData, true, err
							}
							if isReqContextCancelledError(err) {
								return jsonData, true, err
							}
							if ygotCtx.err != nil {
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
						if !xYangSpecMap[xpathKeyExtRet.xpath].hasChildSubTree {
							xfmrLogDebug("Has no child subtree at uri: %v. Tranformer will not traverse subtree further", uri)
							break
						}
						if ygotCtx.trgtYgObj != nil {
							inParamsForGet.ygParentObj = ygotCtx.trgtYgObj
							inParamsForGet.relUri = ""
							inParamsForGet.ygSchema = ygotCtx.trgtYgSchema
						}
					}
					inParamsForGet.resultMap = make(map[string]interface{})
					if strings.HasSuffix(uri, "]") || strings.HasSuffix(uri, "]/") && (len(inParamsForGet.tbl) > 0 && len(inParamsForGet.tblKey) > 0) {
						inParams := formXfmrInputRequest(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, GET, inParamsForGet.tblKey, dbDataMap, nil, nil, txCache)
						err = dbDataFromTblXfmrGet(inParamsForGet.tbl, inParams, dbDataMap, inParamsForGet.dbTblKeyGetCache, inParamsForGet.xpath, inParamsForGet.reqCtxt)
						if err != nil {
							xfmrLogInfo("yangDataFill failed for list instance case(\"%v\").\r\n", uri)
						}
						inParamsForGet.dbDataMap = dbDataMap
						_, dbKeyExist := (*dbDataMap)[cdb][inParamsForGet.tbl][inParamsForGet.tblKey]
						if dbKeyExist || len(xYangSpecMap[inParamsForGet.xpath].xfmrFunc) > 0 {
							err = yangDataFill(inParamsForGet, isOcMdl)
							if err != nil {
								xfmrLogInfo("yangDataFill failed for list instance case(\"%v\").\r\n", uri)
							}
						}
					} else {
						err = yangListDataFill(inParamsForGet, isFirstCall, isOcMdl)
						if err != nil {
							xfmrLogInfo("yangListDataFill failed for list case(\"%v\").\r\n", uri)
						}
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
		if isReqContextCancelledError(err) {
			return jsonData, isEmptyPayload, err
		}
	}

	return jsonData, isEmptyPayload, nil
}
