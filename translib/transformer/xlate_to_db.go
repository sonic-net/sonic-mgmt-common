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
    "github.com/openconfig/ygot/ygot"
    "os"
    "reflect"
    "strings"
    "github.com/Azure/sonic-mgmt-common/translib/db"
    "github.com/Azure/sonic-mgmt-common/translib/ocbinds"
    "github.com/Azure/sonic-mgmt-common/translib/tlerr"
    "github.com/openconfig/ygot/ytypes"
    "github.com/openconfig/goyang/pkg/yang"
    log "github.com/golang/glog"
)

var ocbSch, _ = ocbinds.Schema()

/* Fill redis-db map with field & value info */
func dataToDBMapAdd(tableName string, dbKey string, result map[string]map[string]db.Value, field string, value string) {
	if len(tableName) > 0 {
		_, ok := result[tableName]
		if !ok {
			result[tableName] = make(map[string]db.Value)
		}

		if len(dbKey) > 0 {
			_, ok = result[tableName][dbKey]
			if !ok {
				result[tableName][dbKey] = db.Value{Field: make(map[string]string)}
			}

			if field == XFMR_NONE_STRING {
				if len(result[tableName][dbKey].Field) == 0 {
					result[tableName][dbKey].Field["NULL"] = "NULL"
				}
				return
			}

			if len(field) > 0 {
				result[tableName][dbKey].Field[field] = value
			}
		}
	}
}

/*use when single table name is expected*/
func tblNameFromTblXfmrGet(xfmrTblFunc string, inParams XfmrParams, xfmrDbTblKeyCache map[string]tblKeyCache) (string, error){
	var err error
	var tblList []string
	tblList, err = xfmrTblHandlerFunc(xfmrTblFunc, inParams, xfmrDbTblKeyCache)
	if err != nil {
		return "", err
	}
	if len(tblList) != 1 {
		xfmrLogInfoAll("Uri (\"%v\") translates to 0 or multiple tables instead of single table - %v", inParams.uri, tblList)
		return "", err
	}
	return tblList[0], err
}

/* Fill the redis-db map with data */
func mapFillData(xlateParams xlateToParams) error {
	var dbs [db.MaxDB]*db.DB
	var err error
    xpath := xlateParams.xpath + "/" + xlateParams.name
    xpathInfo, ok := xYangSpecMap[xpath]
    xfmrLogInfoAll("name: \"%v\", xpathPrefix(\"%v\").", xlateParams.name, xlateParams.xpath)

    if !ok || xpathInfo == nil {
        log.Warningf("Yang path(\"%v\") not found.", xpath)
	return nil
    }

    if xpathInfo.tableName == nil && xpathInfo.xfmrTbl == nil{
        log.Warningf("Table for yang-path(\"%v\") not found.", xpath)
	return nil
    }

    if xpathInfo.tableName != nil && *xpathInfo.tableName == XFMR_NONE_STRING {
        log.Warningf("Table for yang-path(\"%v\") NONE.", xpath)
	return nil
    }

    if len(xlateParams.keyName) == 0 {
        log.Warningf("Table key for yang path(\"%v\") not found.", xpath)
	return nil
    }

    tableName := ""
    if xpathInfo.xfmrTbl != nil {
	    inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, "", xlateParams.txCache)
	    // expecting only one table name from tbl-xfmr
	    tableName, err = tblNameFromTblXfmrGet(*xYangSpecMap[xpath].xfmrTbl, inParams, xlateParams.xfmrDbTblKeyCache)
	    if err != nil {
		    if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
			    *xlateParams.xfmrErr = err
		    }
		    return err
	    }
	    if tableName == "" {
		    log.Warningf("No table name found for uri (\"%v\")", xlateParams.uri)
		    return err
	    }
    } else {
	    tableName = *xpathInfo.tableName
    }

    // tblXpathMap used for default value processing for a given request
    if tblUriMapVal, tblUriMapOk := xlateParams.tblXpathMap[tableName][xlateParams.keyName]; !tblUriMapOk {
	    if _, tblOk := xlateParams.tblXpathMap[tableName]; !tblOk {
		    xlateParams.tblXpathMap[tableName] = make(map[string]map[string]bool)
	    }
	    tblUriMapVal = map[string]bool{xlateParams.uri: true}
	    xlateParams.tblXpathMap[tableName][xlateParams.keyName] = tblUriMapVal
    } else {
	    if tblUriMapVal == nil {
		    tblUriMapVal = map[string]bool{xlateParams.uri: true}
	    } else {
		    tblUriMapVal[xlateParams.uri] = true
	    }
	    xlateParams.tblXpathMap[tableName][xlateParams.keyName] = tblUriMapVal
    }

    curXlateParams := xlateParams
    curXlateParams.tableName = tableName
    curXlateParams.xpath = xpath
    err = mapFillDataUtil(curXlateParams)
    return err
}

func mapFillDataUtil(xlateParams xlateToParams) error {
	var dbs [db.MaxDB]*db.DB

	xpathInfo, ok := xYangSpecMap[xlateParams.xpath]
	if !ok {
		errStr := fmt.Sprintf("Invalid yang-path(\"%v\").", xlateParams.xpath)
		return tlerr.InternalError{Format: errStr}
	}

	if len(xpathInfo.xfmrField) > 0 {
		xlateParams.uri = xlateParams.uri + "/" + xlateParams.name

		/* field transformer present */
		xfmrLogInfoAll("Transformer function(\"%v\") invoked for yang path(\"%v\"). uri: %v", xpathInfo.xfmrField, xlateParams.xpath, xlateParams.uri)
		curYgotNodeData, nodeErr := yangNodeForUriGet(xlateParams.uri, xlateParams.ygRoot)
		if nodeErr != nil && xlateParams.oper != DELETE {
			return nil
		}
		inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, xlateParams.keyName, nil, xlateParams.subOpDataMap, curYgotNodeData, xlateParams.txCache)
		retData, err := leafXfmrHandler(inParams, xpathInfo.xfmrField)
		if err != nil {
			if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
				*xlateParams.xfmrErr = err
			}
			return err
		}
		if retData != nil {
			xfmrLogInfoAll("Transformer function : %v Xpath : %v retData: %v", xpathInfo.xfmrField, xlateParams.xpath, retData)
			for f, v := range retData {
				dataToDBMapAdd(xlateParams.tableName, xlateParams.keyName, xlateParams.result, f, v)
			}
		}
		return nil
	}

	if len(xpathInfo.fieldName) == 0 {
		xfmrLogInfoAll("Field for yang-path(\"%v\") not found in DB.", xlateParams.xpath)
		errStr := fmt.Sprintf("Field for yang-path(\"%v\") not found in DB.", xlateParams.xpath)
		return tlerr.InternalError{Format: errStr}
	}
	fieldName := xpathInfo.fieldName
	valueStr := ""

        fieldXpath := xlateParams.tableName + "/" + fieldName
        _, ok = xDbSpecMap[fieldXpath]
        if !ok {
                logStr := fmt.Sprintf("Failed to find the xDbSpecMap: xpath(\"%v\").", fieldXpath)
                log.Warning(logStr)
                return nil
        }

	if xpathInfo.yangEntry.IsLeafList() {
		/* Both yang side and Db side('@' suffix field) the data type is leaf-list */
		xfmrLogInfoAll("Yang type and Db type is Leaflist for field  = %v", xlateParams.xpath)
		fieldName += "@"
		if reflect.ValueOf(xlateParams.value).Kind() != reflect.Slice {
			logStr := fmt.Sprintf("Value for yang xpath %v which is a leaf-list should be a slice", xlateParams.xpath)
			log.Warning(logStr)
			return nil
		}
		valData := reflect.ValueOf(xlateParams.value)
		for fidx := 0; fidx < valData.Len(); fidx++ {
			if fidx > 0 {
				valueStr += ","
			}

                        // SNC-3626 - string conversion based on the primitive type
                        fVal, err := unmarshalJsonToDbData(xDbSpecMap[fieldXpath].dbEntry, fieldXpath, fieldName, valData.Index(fidx).Interface())
                        if err == nil {
			      if ((strings.Contains(fVal, ":")) && (strings.HasPrefix(fVal, OC_MDL_PFX) || strings.HasPrefix(fVal, IETF_MDL_PFX) || strings.HasPrefix(fVal, IANA_MDL_PFX))) {
				      // identity-ref/enum has module prefix
				      fVal = strings.SplitN(fVal, ":", 2)[1]
			      }
			      valueStr = valueStr + fVal
                        } else {
                              logStr := fmt.Sprintf("Couldn't unmarshal Json to DbData: table(\"%v\") field(\"%v\") value(\"%v\").", xlateParams.tableName, fieldName, valData.Index(fidx).Interface())
                              log.Warning(logStr)
                              return nil
                        }
		}
		xfmrLogInfoAll("leaf-list value after conversion to DB format %v  :  %v", fieldName, valueStr)

	} else { // xpath is a leaf

                // SNC-3626 - string conversion based on the primitive type
                fVal, err := unmarshalJsonToDbData(xDbSpecMap[fieldXpath].dbEntry, fieldXpath, fieldName, xlateParams.value)
                if err == nil {
                      valueStr = fVal
                } else {
                      logStr := fmt.Sprintf("Couldn't unmarshal Json to DbData: table(\"%v\") field(\"%v\") value(\"%v\").", xlateParams.tableName, fieldName, xlateParams.value)
                      log.Warning(logStr)
                      return nil
                }

		if ((strings.Contains(valueStr, ":")) && (strings.HasPrefix(valueStr, OC_MDL_PFX) || strings.HasPrefix(valueStr, IETF_MDL_PFX) || strings.HasPrefix(valueStr, IANA_MDL_PFX))) {
			// identity-ref/enum might has module prefix
			valueStr = strings.SplitN(valueStr, ":", 2)[1]
		}
	}

	dataToDBMapAdd(xlateParams.tableName, xlateParams.keyName, xlateParams.result, fieldName, valueStr)
	xfmrLogInfoAll("TblName: \"%v\", key: \"%v\", field: \"%v\", valueStr: \"%v\".", xlateParams.tableName, xlateParams.keyName,
	fieldName, valueStr)
	return nil
}

func sonicYangReqToDbMapCreate(xlateParams xlateToParams) error {
    if reflect.ValueOf(xlateParams.jsonData).Kind() == reflect.Map {
        data := reflect.ValueOf(xlateParams.jsonData)
        for _, key := range data.MapKeys() {
            _, ok := xDbSpecMap[key.String()]
            if ok {
                directDbMapData("", key.String(), data.MapIndex(key).Interface(), xlateParams.result)
            } else {
		curXlateParams := xlateParams
		curXlateParams.jsonData = data.MapIndex(key).Interface()
                sonicYangReqToDbMapCreate(curXlateParams)
            }
        }
    }
    return nil
}

func dbMapDataFill(uri string, tableName string, keyName string, d map[string]interface{}, result map[string]map[string]db.Value) {
	result[tableName][keyName] = db.Value{Field: make(map[string]string)}
	for field, value := range d {
		fieldXpath := tableName + "/" + field
		if _, fieldOk := xDbSpecMap[fieldXpath]; (fieldOk  && (xDbSpecMap[fieldXpath].dbEntry != nil)) {
			xfmrLogInfoAll("Found non-nil yang entry in xDbSpecMap for field xpath = %v", fieldXpath)
			if xDbSpecMap[fieldXpath].dbEntry.IsLeafList() {
				xfmrLogInfoAll("Yang type is Leaflist for field  = %v", field)
				field += "@"
				fieldDt := reflect.ValueOf(value)
				fieldValue := ""
				for fidx := 0; fidx < fieldDt.Len(); fidx++ {
					if fidx > 0 {
						fieldValue += ","
					}
					fVal, err := unmarshalJsonToDbData(xDbSpecMap[fieldXpath].dbEntry, fieldXpath, field, fieldDt.Index(fidx).Interface())
					if err != nil {
						log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
					} else {
						fieldValue = fieldValue + fVal
					}
				}
				result[tableName][keyName].Field[field] = fieldValue
				continue
			}
			dbval, err := unmarshalJsonToDbData(xDbSpecMap[fieldXpath].dbEntry, fieldXpath, field, value)
			if err != nil {
				log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
			} else {
				result[tableName][keyName].Field[field] = dbval
			}
		} else {
			// should ideally never happen , just adding for safety
			xfmrLogInfoAll("Did not find entry in xDbSpecMap for field xpath = %v", fieldXpath)
		}
	}
}

func dbMapListDataFill(uri string, tableName string, dbEntry *yang.Entry, jsonData interface{}, result map[string]map[string]db.Value) {
	data := reflect.ValueOf(jsonData)
	tblKeyName := strings.Split(dbEntry.Key, " ")
	for idx := 0; idx < data.Len(); idx++ {
		keyName := ""
		d := data.Index(idx).Interface().(map[string]interface{})
		for i, k := range tblKeyName {
			if i > 0 {
				keyName += "|"
			}
			fieldXpath := tableName + "/" + k
			val, err := unmarshalJsonToDbData(dbEntry.Dir[k], fieldXpath, k, d[k])
			if err != nil {
				log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
			} else {
				keyName += val
			}
			delete(d, k)
		}
		dbMapDataFill(uri, tableName, keyName, d, result)
	}
}

func directDbMapData(uri string, tableName string, jsonData interface{}, result map[string]map[string]db.Value) bool {
	_, ok := xDbSpecMap[tableName]
	if ok && xDbSpecMap[tableName].dbEntry != nil {
		data := reflect.ValueOf(jsonData).Interface().(map[string]interface{})
		key  := ""
		dbSpecData := xDbSpecMap[tableName]
		result[tableName] = make(map[string]db.Value)

		if dbSpecData.keyName != nil {
			key = *dbSpecData.keyName
			xfmrLogInfoAll("Fill data for container uri(%v), key(%v)", uri, key)
			dbMapDataFill(uri, tableName, key, data, result)
			return true
		}

		for k, v := range data {
			xpath := tableName + "/" + k
			curDbSpecData, ok := xDbSpecMap[xpath]
			if ok && curDbSpecData.dbEntry != nil {
				eType := yangTypeGet(curDbSpecData.dbEntry)
				switch eType {
				case "list":
					xfmrLogInfoAll("Fill data for list uri(%v)", uri)
					dbMapListDataFill(uri, tableName, curDbSpecData.dbEntry, v, result)
				default:
					xfmrLogInfoAll("Invalid node type for uri(%v)", uri)
				}
			}
		}
	}
	return true
}

/* Get the data from incoming update/replace request, create map and fill with dbValue(ie. field:value to write into redis-db */
func dbMapUpdate(d *db.DB, ygRoot *ygot.GoStruct, oper int, path string, requestUri string, jsonData interface{}, result map[int]map[db.DBNum]map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value, yangAuxValMap map[string]map[string]db.Value, txCache interface{}) error {
    xfmrLogInfo("Update/replace req: path(\"%v\").", path)
    err := dbMapCreate(d, ygRoot, oper, path, requestUri, jsonData, result, yangDefValMap, yangAuxValMap, txCache)
    xfmrLogInfo("Update/replace req: path(\"%v\") result(\"%v\").", path, result)
    printDbData(result, nil, "/tmp/yangToDbDataUpRe.txt")
    return err
}

func dbMapDefaultFieldValFill(xlateParams xlateToParams, tblUriList []string) error {
	tblData := xlateParams.result[xlateParams.tableName]
	var dbs [db.MaxDB]*db.DB
	tblName := xlateParams.tableName
	dbKey := xlateParams.keyName
	for _, tblUri := range tblUriList {
		xfmrLogInfoAll("Processing uri %v for default value filling(Table - %v, dbKey - %v)", tblUri, tblName, dbKey)
		yangXpath, _, prdErr := XfmrRemoveXPATHPredicates(tblUri)
		if prdErr != nil {
			continue
		}
		yangNode, ok := xYangSpecMap[yangXpath]
		if ok {
			for childName  := range yangNode.yangEntry.Dir {
				childXpath := yangXpath + "/" + childName
				childUri :=  tblUri + "/" + childName
				childNode, ok := xYangSpecMap[childXpath]
				if ok {
					if (len(childNode.xfmrFunc) > 0) {
						xfmrLogInfoAll("Skip default filling since a subtree Xfmr found for path - %v", childXpath)
						continue
					}

					if (childNode.yangDataType == YANG_LIST || childNode.yangDataType == YANG_CONTAINER) && !childNode.yangEntry.ReadOnly() {
						var tblList []string
						tblList = append(tblList, childUri)
						err := dbMapDefaultFieldValFill(xlateParams, tblList)
						if err != nil {
							return err
						}
					}
					if (childNode.tableName != nil && *childNode.tableName == tblName) || (childNode.xfmrTbl != nil) {
						tblXfmrPresent := false
						var inParamsTblXfmr XfmrParams
						if childNode.xfmrTbl != nil {
							if len(*childNode.xfmrTbl) > 0 {
								inParamsTblXfmr = formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, childUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, "", xlateParams.txCache)
								//performance optimization - call table transformer only for default val leaf and avoid for other leaves unless its REPLACE operi(aux-map filling)
								tblXfmrPresent = true

							}
						}
						_, ok := tblData[dbKey].Field[childName]
						if !ok {
							if len(childNode.xfmrField) > 0 {
								childYangType := childNode.yangEntry.Type.Kind
								var param interface{}
								oper := xlateParams.oper
								if len(childNode.defVal) > 0 {
									if tblXfmrPresent {
										chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
										xfmrLogInfoAll("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
										if ctErr != nil || chldTblNm != tblName {
											continue
										}
									}
									xfmrLogInfoAll("Update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
									childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
									_, defValPtr, err := DbToYangType(childYangType, childXpath, childNode.defVal)
									if err == nil && defValPtr != nil {
										param = defValPtr
									} else {
										xfmrLogInfoAll("Failed to update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
										childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
									}
								} else {
									if xlateParams.oper != REPLACE {
										continue
									}
									oper = DELETE
									if tblXfmrPresent {
										chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
										xfmrLogInfoAll("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
										if ctErr != nil || chldTblNm != tblName {
											continue
										}
									}
								}
								inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, tblUri+"/"+childName, xlateParams.requestUri, oper, "", nil, xlateParams.subOpDataMap, param, xlateParams.txCache)
								retData, err := leafXfmrHandler(inParams, childNode.xfmrField)
								if err != nil {
									log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.xfmrField)
								}
								if retData != nil {
									xfmrLogInfoAll("Transformer function : %v Xpath: %v retData: %v", childNode.xfmrField, childXpath, retData)
									for f, v := range retData {
										// Fill default value only if value is not available in result Map
										// else we overwrite the value filled in resultMap with default value
										_, ok := xlateParams.result[tblName][dbKey].Field[f]
										if !ok {
											if len(childNode.defVal) > 0 {
												dataToDBMapAdd(tblName, dbKey, xlateParams.yangDefValMap, f, v)
											} else {
												// Fill the yangAuxValMap with all fields that are not in either resultMap or defaultValue Map
												dataToDBMapAdd(tblName, dbKey, xlateParams.yangAuxValMap, f, "")
											}
										}
									}
								}
							} else if len(childNode.fieldName) > 0 {
								var xfmrErr error
								if _, ok := xDbSpecMap[tblName+"/"+childNode.fieldName]; ok {
									if tblXfmrPresent {
										chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
										xfmrLogInfoAll("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
										if ctErr != nil || chldTblNm != tblName {
											continue
										}
									}
									// Fill default value only if value is not available in result Map
									// else we overwrite the value filled in resultMap with default value
									_, ok = xlateParams.result[tblName][dbKey].Field[childNode.fieldName]
									if !ok {
										if len(childNode.defVal) > 0 {
											curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, childXpath, dbKey, xlateParams.jsonData, xlateParams.resultMap, xlateParams.yangDefValMap, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, &xfmrErr, childName, childNode.defVal, tblName)
											err := mapFillDataUtil(curXlateParams)
											if err != nil {
												log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.fieldName)
											}
										} else {
											if xlateParams.oper != REPLACE {
												continue
											}
											dataToDBMapAdd(tblName, dbKey, xlateParams.yangAuxValMap, childNode.fieldName, "")
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func dbMapDefaultValFill(xlateParams xlateToParams) error {
	for tbl, tblData := range xlateParams.result {
		for dbKey := range tblData {
			var yxpathList []string //contains all uris(with keys) that were traversed for a table while processing the incoming request
			if tblUriMapVal, ok := xlateParams.tblXpathMap[tbl][dbKey]; ok {
				for tblUri := range tblUriMapVal {
						yxpathList = append(yxpathList, tblUri)
				}
			}
			curXlateParams := xlateParams
			curXlateParams.tableName = tbl
			curXlateParams.keyName = dbKey
			if len(yxpathList) > 0 {
				err := dbMapDefaultFieldValFill(curXlateParams, yxpathList)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}


/* Get the data from incoming create request, create map and fill with dbValue(ie. field:value to write into redis-db */
func dbMapCreate(d *db.DB, ygRoot *ygot.GoStruct, oper int, uri string, requestUri string, jsonData interface{}, resultMap map[int]RedisDbMap, yangDefValMap map[string]map[string]db.Value, yangAuxValMap map[string]map[string]db.Value, txCache interface{}) error {
	var err, xfmrErr error
    var cascadeDelTbl []string
	var result        = make(map[string]map[string]db.Value)
	tblXpathMap  := make(map[string]map[string]map[string]bool)
	subOpDataMap := make(map[int]*RedisDbMap)
	root         := xpathRootNameGet(uri)

	/* Check if the parent table exists for RFC compliance */
	var exists bool
	var dbs [db.MaxDB]*db.DB
	subOpMapDiscard := make(map[int]*RedisDbMap)
	exists, err = verifyParentTable(d, dbs, ygRoot, oper, uri, nil, txCache, subOpMapDiscard)
	xfmrLogInfoAll("verifyParentTable() returned - exists - %v, err - %v", exists, err)
	if err != nil {
		log.Warningf("Cannot perform Operation %v on uri %v due to - %v", oper, uri, err)
		return err
	}
	if !exists {
		errStr := fmt.Sprintf("Parent table does not exist for uri(%v)", uri)
		return tlerr.NotFoundError{Format: errStr}
	}

	xlateToData := formXlateToDbParam(d, ygRoot, oper, root, uri, "", "", jsonData, resultMap, result, txCache, tblXpathMap, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "")

	moduleNm := "/" + strings.Split(uri, "/")[1]
	xfmrLogInfo("Module name for uri %s is %s", uri, moduleNm)

	if isSonicYang(uri) {
		err = sonicYangReqToDbMapCreate(xlateToData)
		xpathPrefix, keyName, tableName := sonicXpathKeyExtract(uri)
		xfmrLogInfoAll("xpath - %v, keyName - %v, tableName - %v , for uri - %v", xpathPrefix, keyName, tableName, uri)
		fldPth := strings.Split(xpathPrefix, "/")
		if len(fldPth) > SONIC_FIELD_INDEX {
			fldNm := fldPth[SONIC_FIELD_INDEX]
			xfmrLogInfoAll("Field Name : %v", fldNm)
			if fldNm != "" {
				_, ok := xDbSpecMap[tableName]
				if ok {
					dbSpecField := tableName + "/" + fldNm
					_, dbFldok := xDbSpecMap[dbSpecField]
					if dbFldok {
						/* RFC compliance - REPLACE on leaf/leaf-list becomes UPDATE/merge */
						resultMap[UPDATE] = make(RedisDbMap)
						resultMap[UPDATE][db.ConfigDB] = result
					} else {
						log.Warningf("For uri - %v, no entry found in xDbSpecMap for table(%v)/field(%v)", uri, tableName, fldNm)
					}
				} else {
					log.Warningf("For uri - %v, no entry found in xDbSpecMap with tableName - %v", uri, tableName)
				}
			}
		} else {
			resultMap[oper] = make(RedisDbMap)
			resultMap[oper][db.ConfigDB] = result
		}
	} else {
		/* Invoke pre-xfmr is present for the yang module */
		if modSpecInfo, specOk := xYangSpecMap[moduleNm]; specOk && (len(modSpecInfo.xfmrPre) > 0) {
			var dbs [db.MaxDB]*db.DB
			inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, "", nil, xlateToData.subOpDataMap, nil, txCache)
			err = preXfmrHandlerFunc(modSpecInfo.xfmrPre, inParams)
			xfmrLogInfo("Invoked pre-transformer: %v, oper: %v, subOpDataMap: %v ",
			modSpecInfo.xfmrPre, oper, subOpDataMap)
			if err != nil {
				log.Warningf("Pre-transformer: %v failed.(err:%v)", modSpecInfo.xfmrPre, err)
				return err
			}
		}

		err = yangReqToDbMapCreate(xlateToData)
		if xfmrErr != nil {
			return xfmrErr
		}
		if err != nil {
			return err
		}
	}
	if err == nil {
		if !isSonicYang(uri) {
			xpath, _, _ := XfmrRemoveXPATHPredicates(uri)
			yangNode, ok := xYangSpecMap[xpath]
			defSubOpDataMap := make(map[int]*RedisDbMap)
			if ok {
				xfmrLogInfo("Fill default value for %v, oper(%v)\r\n", uri, oper)
				curXlateToParams := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpath, "", jsonData, resultMap, result, txCache, tblXpathMap, defSubOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "")
				curXlateToParams.yangDefValMap = yangDefValMap
				curXlateToParams.yangAuxValMap = yangAuxValMap
				err = dbMapDefaultValFill(curXlateToParams)
				if err != nil {
					return err
				}
			}

			if ok && oper == REPLACE {
				if yangNode.yangDataType == YANG_LEAF {
					xfmrLogInfo("Change leaf oper to UPDATE for %v, oper(%v)\r\n", uri, oper)
					resultMap[UPDATE] = make(RedisDbMap)
					resultMap[UPDATE][db.ConfigDB] = result
					result = make(map[string]map[string]db.Value)
				} else if yangNode.yangDataType == YANG_LEAF_LIST {
					/* RFC compliance - REPLACE on leaf-list becomes UPDATE/merge */
					xfmrLogInfo("Change leaflist oper to UPDATE for %v, oper(%v)\r\n", uri, oper)
					resultMap[UPDATE] = make(RedisDbMap)
					resultMap[UPDATE][db.ConfigDB] = result
					result = make(map[string]map[string]db.Value)
				}
			}

			moduleNm := "/" + strings.Split(uri, "/")[1]
			xfmrLogInfo("Module name for uri %s is %s", uri, moduleNm)
			if _, ok := xYangSpecMap[moduleNm]; ok {
				if xYangSpecMap[moduleNm].yangDataType == "container" && len(xYangSpecMap[moduleNm].xfmrPost) > 0 {
					xfmrLogInfo("Invoke post transformer: %v, result map: %v ", xYangSpecMap[moduleNm].xfmrPost, result)
					var dbDataMap = make(RedisDbMap)
					dbDataMap[db.ConfigDB] = result
					var dbs [db.MaxDB]*db.DB
					inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, "", &dbDataMap, subOpDataMap, nil, txCache)
					inParams.yangDefValMap = yangDefValMap
					result, err = postXfmrHandlerFunc(xYangSpecMap[moduleNm].xfmrPost, inParams)
					if err != nil {
						return err
					}
                                        if inParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
                                            for _, tblNm :=  range *inParams.pCascadeDelTbl {
                                            if !contains(cascadeDelTbl, tblNm) {
                                                cascadeDelTbl = append(cascadeDelTbl, tblNm)
                                            }
                                        }
                                    }
				}
			} else {
				log.Warningf("No Entry exists for module %s in xYangSpecMap. Unable to process post xfmr (\"%v\") uri(\"%v\") error (\"%v\").", oper, uri, err)
			}
                        if len(result) > 0 || len(subOpDataMap) > 0 {
                                  resultMap[oper] = make(RedisDbMap)
                                  resultMap[oper][db.ConfigDB] = result
                                  for op, redisMapPtr := range subOpDataMap {
                                         if redisMapPtr != nil {
                                                 if _,ok := resultMap[op]; !ok {
                                                       resultMap[op] = make(RedisDbMap)
                                               }
                                               for dbNum, dbMap := range *redisMapPtr {
                                                       if _,ok := resultMap[op][dbNum]; !ok {
                                                               resultMap[op][dbNum] = make(map[string]map[string]db.Value)
                                                       }
                                                       mapCopy(resultMap[op][dbNum],dbMap)
                                               }
                                         }
                                  }
                        }
		}

		err = dbDataXfmrHandler(resultMap)
		if err != nil {
			log.Warningf("Failed in dbdata-xfmr for %v", resultMap)
			return err
		}

                if (len(cascadeDelTbl) > 0) {
		    cdErr := handleCascadeDelete(d, resultMap, cascadeDelTbl)
		    if cdErr != nil {
			xfmrLogInfo("Cascade Delete Failed for cascadeDelTbl (%v), Error (%v).", cascadeDelTbl, cdErr)
			return cdErr
		    }
                }

		printDbData(resultMap, yangDefValMap, "/tmp/yangToDbDataCreate.txt")
	} else {
		log.Warningf("DBMapCreate req failed for oper (\"%v\") uri(\"%v\") error (\"%v\").", oper, uri, err)
	}
	return err
}

func yangNodeForUriGet(uri string, ygRoot *ygot.GoStruct) (interface{}, error) {
	path, err := ygot.StringToPath(uri, ygot.StructuredPath, ygot.StringSlicePath)
	if path == nil || err != nil {
		log.Warningf("For uri %v - StringToPath failure", uri)
		errStr := fmt.Sprintf("Ygot stringTopath failed for uri(%v)", uri)
		return nil, tlerr.InternalError{Format: errStr}
	}

	for _, p := range path.Elem {
		pathSlice := strings.Split(p.Name, ":")
		p.Name = pathSlice[len(pathSlice)-1]
		if len(p.Key) > 0 {
			for ekey, ent := range p.Key {
				// SNC-2126: check the occurrence of ":"
				if ((strings.Contains(ent, ":")) && (strings.HasPrefix(ent, OC_MDL_PFX) || strings.HasPrefix(ent, IETF_MDL_PFX) || strings.HasPrefix(ent, IANA_MDL_PFX))) {
					// identity-ref/enum has module prefix
					eslice := strings.SplitN(ent, ":", 2)
					// TODO - exclude the prexix by checking enum type
					p.Key[ekey] = eslice[len(eslice)-1]
				} else {
					p.Key[ekey] = ent
				}
			}
		}
	}
	schRoot := ocbSch.RootSchema()
	node, nErr := ytypes.GetNode(schRoot, (*ygRoot).(*ocbinds.Device), path)
	if nErr != nil {
		log.Warningf("For uri %v - GetNode failure - %v", uri, nErr)
		errStr := fmt.Sprintf("%v", nErr)
		return nil, tlerr.InternalError{Format: errStr}
	}
	if ((node == nil) || (len(node) == 0) || (node[0].Data == nil)) {
		log.Warningf("GetNode returned nil for uri %v", uri)
		errStr := "GetNode returned nil for the given uri."
		return nil, tlerr.InternalError{Format: errStr}
	}
	xfmrLogInfoAll("GetNode data: %v", node[0].Data)
	return node[0].Data, nil
}

func yangReqToDbMapCreate(xlateParams xlateToParams) error {
	xfmrLogInfoAll("key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
	var dbs [db.MaxDB]*db.DB
	var retErr error

	if reflect.ValueOf(xlateParams.jsonData).Kind() == reflect.Slice {
		xfmrLogInfoAll("slice data: key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
		jData := reflect.ValueOf(xlateParams.jsonData)
		dataMap := make([]interface{}, jData.Len())
		for idx := 0; idx < jData.Len(); idx++ {
			dataMap[idx] = jData.Index(idx).Interface()
		}
		for _, data := range dataMap {
			curKey := ""
			curUri, _ := uriWithKeyCreate(xlateParams.uri, xlateParams.xpath, data)
			_, ok := xYangSpecMap[xlateParams.xpath]
			if ok && len(xYangSpecMap[xlateParams.xpath].xfmrKey) > 0 {
				/* key transformer present */
				curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
				if nodeErr != nil {
					curYgotNode = nil
				}
				inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)

				ktRetData, err := keyXfmrHandler(inParams, xYangSpecMap[xlateParams.xpath].xfmrKey)
				//if key transformer is called without key values in curUri ignore the error
				if err != nil  && strings.HasSuffix(curUri, "]") {
					if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
						*xlateParams.xfmrErr = err
					}
					return nil
				}
				curKey = ktRetData
			} else if ok && xYangSpecMap[xlateParams.xpath].keyName != nil {
				curKey = *xYangSpecMap[xlateParams.xpath].keyName
			} else {
				curKey = keyCreate(xlateParams.keyName, xlateParams.xpath, data, xlateParams.d.Opts.KeySeparator)
			}
			curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, xlateParams.xpath, curKey, data, xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, "", "", "")
			retErr = yangReqToDbMapCreate(curXlateParams)
		}
	} else {
		if reflect.ValueOf(xlateParams.jsonData).Kind() == reflect.Map {
			jData := reflect.ValueOf(xlateParams.jsonData)
			for _, key := range jData.MapKeys() {
				typeOfValue := reflect.TypeOf(jData.MapIndex(key).Interface()).Kind()

				xfmrLogInfoAll("slice/map data: key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
				xpath    := xlateParams.uri
				curUri   := xlateParams.uri
				curKey   := xlateParams.keyName
				pathAttr := key.String()
				if len(xlateParams.xpath) > 0 {
					curUri = xlateParams.uri + "/" + pathAttr
					if strings.Contains(pathAttr, ":") {
						pathAttr = strings.Split(pathAttr, ":")[1]
					}
					xpath  = xlateParams.xpath + "/" + pathAttr
				}
				_, ok := xYangSpecMap[xpath]
				xfmrLogInfoAll("slice/map data: curKey(\"%v\"), xpath(\"%v\"), curUri(\"%v\").",
				curKey, xpath, curUri)
				if ok && xYangSpecMap[xpath] != nil && len(xYangSpecMap[xpath].xfmrKey) > 0 {
					specYangType := yangTypeGet(xYangSpecMap[xpath].yangEntry)
					curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
					if nodeErr != nil {
						curYgotNode = nil
					}
					inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
					ktRetData, err := keyXfmrHandler(inParams, xYangSpecMap[xpath].xfmrKey)
					if ((err != nil) && (specYangType != YANG_LIST || strings.HasSuffix(curUri, "]"))) {
						if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
							*xlateParams.xfmrErr = err
						}
						return nil
					}
					curKey = ktRetData
				} else if ok && xYangSpecMap[xpath].keyName != nil {
					curKey = *xYangSpecMap[xpath].keyName
				}

				if ok && (typeOfValue == reflect.Map || typeOfValue == reflect.Slice) && xYangSpecMap[xpath].yangDataType != "leaf-list" {
					// Call subtree only if start processing for the requestUri. Skip for parent uri traversal
					curXpath, _, _ := XfmrRemoveXPATHPredicates(curUri)
					reqXpath, _, _ := XfmrRemoveXPATHPredicates(xlateParams.requestUri)
					xfmrLogInfoAll("CurUri: %v, requestUri: %v\r\n", curUri, xlateParams.requestUri)
					xfmrLogInfoAll("curxpath: %v, requestxpath: %v\r\n", curXpath, reqXpath)
					if strings.HasPrefix(curXpath, reqXpath) {
						if xYangSpecMap[xpath] != nil && len(xYangSpecMap[xpath].xfmrFunc) > 0 &&
                        (xYangSpecMap[xlateParams.xpath] != xYangSpecMap[xpath]) {
							/* subtree transformer present */
							curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
							if nodeErr != nil {
								curYgotNode = nil
							}
							inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
							stRetData, err := xfmrHandler(inParams, xYangSpecMap[xpath].xfmrFunc)
							if err != nil {
								if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
									*xlateParams.xfmrErr = err
                                                                }
								return nil
							}
							if stRetData != nil {
								mapCopy(xlateParams.result, stRetData)
							}
                                                        if xlateParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
                                                            for _, tblNm :=  range *inParams.pCascadeDelTbl {
                                                                if !contains(*xlateParams.pCascadeDelTbl, tblNm) {
                                                                    *xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tblNm)
                                                                }
                                                            }
                                                        }
						}
					}
					curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, xpath, curKey, jData.MapIndex(key).Interface(), xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, "", "", "")
					retErr = yangReqToDbMapCreate(curXlateParams)
				} else {
					pathAttr := key.String()
					if strings.Contains(pathAttr, ":") {
						pathAttr = strings.Split(pathAttr, ":")[1]
					}
					xpath := xlateParams.xpath + "/" + pathAttr
					xfmrLogInfoAll("LEAF Case: xpath: %v, xpathPrefix: %v, pathAttr: %v", xpath, xlateParams.xpath, pathAttr)
					/* skip processing for list key-leaf outside of config container(OC yang) directly under the list.
					   Inside full-spec isKey is set to true for list key-leaf dierctly under the list(outside of config container) 
					   For ietf yang(eg.ietf-ptp) list key-leaf might have a field transformer.
					 */
					 _, ok := xYangSpecMap[xpath]
					if ok && ((!xYangSpecMap[xpath].isKey) || (len(xYangSpecMap[xpath].xfmrField) > 0)) {
						if len(xYangSpecMap[xpath].xfmrFunc) == 0 {
							value := jData.MapIndex(key).Interface()
							xfmrLogInfoAll("data field: key(\"%v\"), value(\"%v\").", key, value)
							curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, xlateParams.xpath, curKey, xlateParams.jsonData, xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, pathAttr, value, "")
							retErr = mapFillData(curXlateParams)
							if retErr != nil {
								log.Warningf("Failed constructing data for db write: key(\"%v\"), value(\"%v\"), path(\"%v\").",
								pathAttr, value, xlateParams.xpath)
								return retErr
							}
						} else {
							xfmrLogInfoAll("write: key(\"%v\"), xpath(\"%v\"), uri(%v).",key, xpath, xlateParams.uri)
							curYgotNode, nodeErr := yangNodeForUriGet(xlateParams.uri, xlateParams.ygRoot)
							if nodeErr != nil {
								curYgotNode = nil
							}
							inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, curKey, nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
							stRetData, err := xfmrHandler(inParams, xYangSpecMap[xpath].xfmrFunc)
							if err != nil {
								if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
                                                                        *xlateParams.xfmrErr = err
                                                                }
								return nil
							}
							if stRetData != nil {
                                                                mapCopy(xlateParams.result, stRetData)
							}
                                                        if xlateParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
                                                            for _, tblNm :=  range *inParams.pCascadeDelTbl {
                                                                if !contains(*xlateParams.pCascadeDelTbl, tblNm) {
                                                                    *xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tblNm)
                                                                }
                                                            }
                                                        }
						}
					}
				}
			}
		}
	}

	return retErr
}

func verifyParentTableSonic(d *db.DB, dbs [db.MaxDB]*db.DB, oper int, uri string, dbData RedisDbMap) (bool, error) {
        var err error
        pathList := splitUri(uri)

        xpath, dbKey, table := sonicXpathKeyExtract(uri)
        xfmrLogInfoAll("uri: %v xpath: %v table: %v, key: %v", uri, xpath, table, dbKey)

        if (len(table) > 0) && (len(dbKey) > 0) {
		tableExists := false
		var derr error
		if oper == GET {
			var cdb db.DBNum = db.ConfigDB
			dbInfo, ok := xDbSpecMap[table]
			if !ok {
				log.Warningf("No entry in xDbSpecMap for xpath %v, for uri - %v", table, uri)
			} else {
				cdb =  dbInfo.dbIndex
			}
			tableExists = dbTableExistsInDbData(cdb, table, dbKey, dbData)
			derr = tlerr.NotFoundError{Format:"Resource not found"}
		} else {
			// Valid table mapping exists. Read the table entry from DB
			tableExists, derr = dbTableExists(d, table, dbKey, oper)
			if derr != nil {
				return false, derr
			}
		}
		if len(pathList) == SONIC_LIST_INDEX && (oper == UPDATE || oper == CREATE || oper == DELETE || oper == GET) && !tableExists {
                        // Uri is at /sonic-module:sonic-module/container-table/list
                        // PATCH opertion permitted only if table exists in DB.
                        // POST case since the uri is the parent, the parent needs to exist
                        // PUT case allow operation(Irrespective of table existence update the DB either through CREATE or REPLACE)
                        // DELETE case Table instance should be available to perform delete else, CVL may throw error
                        log.Warningf("Parent table %v with key %v does not exist for oper %v in DB", table, dbKey, oper)
                        err = tlerr.NotFound("Resource not found")
                        return false, err
                } else if len(pathList) > SONIC_LIST_INDEX  && !tableExists {
                        // Uri is at /sonic-module/container-table/list or /sonic-module/container-table/list/leaf
                        // Parent table should exist for all CRUD cases
                        log.Warningf("Parent table %v with key %v does not exist in DB", table, dbKey)
                        err = tlerr.NotFound("Resource not found")
                        return false, err
                } else {
                        // Allow all other operations
                        return true, err
                }
        } else {
                // Request is at module level. No need to check for parent table. Hence return true always or 
                // Request at /sonic-module:sonic-module/container-table level
                return true, err
        }
}

/* This function checks the existence of Parent tables in DB for the given URI request
   and returns a boolean indicating if the operation is permitted based on the operation type*/
func verifyParentTable(d *db.DB, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct, oper int, uri string, dbData RedisDbMap, txCache interface{}, subOpDataMap map[int]*RedisDbMap) (bool, error) {
	xfmrLogInfoAll("Checking for Parent table existence for uri: %v", uri)
        if isSonicYang(uri) {
                return verifyParentTableSonic(d, dbs, oper, uri, dbData)
        } else {
                return verifyParentTableOc(d, dbs, ygRoot, oper, uri, dbData, txCache, subOpDataMap)
        }
}

func verifyParentTblSubtree(dbs [db.MaxDB]*db.DB, uri string, xfmrFuncNm string, oper int, dbData RedisDbMap) (bool, error) {
	var inParams XfmrSubscInParams
	inParams.uri = uri
	inParams.dbDataMap = make(RedisDbMap)
	inParams.dbs = dbs
	inParams.subscProc = TRANSLATE_SUBSCRIBE
	parentTblExists := true
	var err error

	st_result, st_err := xfmrSubscSubtreeHandler(inParams, xfmrFuncNm)
	if st_result.isVirtualTbl {
		xfmrLogInfoAll("Subtree returned Virtual table true.")
		goto Exit
	}
	if st_err != nil {
		err = st_err
		parentTblExists = false
		goto Exit
	} else if st_result.dbDataMap != nil && len(st_result.dbDataMap) > 0 {
		xfmrLogInfoAll("Subtree subcribe dbData %v", st_result.dbDataMap)
		for dbNo, dbMap := range st_result.dbDataMap {
			xfmrLogInfoAll("processing Db no - %v", dbNo)
			for table, keyInstance := range dbMap {
				xfmrLogInfoAll("processing Db table - %v", table)
				for dbKey := range keyInstance {
					xfmrLogInfoAll("processing Db key - %v", dbKey)
					exists := false
					if oper != GET {
						dptr, derr := db.NewDB(getDBOptions(dbNo))
						if derr != nil {
							log.Warningf("Couldn't allocate NewDb/DbOptions for db - %v, while processing uri - %v", dbNo, uri)
							err = derr
							parentTblExists = false
							goto Exit
						}
						defer dptr.DeleteDB()
						exists, err = dbTableExists(dptr, table, dbKey, oper)
					} else {
						d := dbs[dbNo]
						if dbKey == "*" { //dbKey is "*" for GET on entire list
							xfmrLogInfoAll("Found table instance in dbData")
							goto Exit
						}
						// GET case - attempt to find in dbData before doing a dbGet in dbTableExists()
						exists = dbTableExistsInDbData(dbNo, table, dbKey, dbData)
						if exists {
							xfmrLogInfoAll("Found table instance in dbData")
							goto Exit
						}
						exists, err = dbTableExists(d, table, dbKey, oper)
					}
					if !exists || err != nil {
						log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for uri %v", table, dbKey, uri)
						err = tlerr.NotFound("Resource not found")
						parentTblExists = false
						goto Exit
					}
				}
			}

		}
	} else {
		log.Warningf("No Table information retrieved from subtree for uri %v", uri)
		err = tlerr.NotFound("Resource not found")
		parentTblExists = false
		goto Exit
	}
	Exit:
	xfmrLogInfoAll("For subtree at uri - %v, returning ,parentTblExists - %v, err - %v", parentTblExists, err)
	return parentTblExists, err
}

func verifyParentTableOc(d *db.DB, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct, oper int, uri string, dbData RedisDbMap, txCache interface{}, subOpDataMap map[int]*RedisDbMap) (bool, error) {
	var err error
	var cdb db.DBNum
        uriList := splitUri(uri)
        parentTblExists := true
        curUri := "/"
        yangType := ""
	xpath, _, _ := XfmrRemoveXPATHPredicates(uri)
	xpathInfo, ok := xYangSpecMap[xpath]
        if !ok {
		errStr := fmt.Sprintf("No entry found in xYangSpecMap for uri - %v", uri)
		err = tlerr.InternalError{Format:errStr}
		return false, err
	}
	yangType = yangTypeGet(xpathInfo.yangEntry)
	if yangType == YANG_LEAF_LIST {
		/*query is for leaf-list instance, hence remove that from uriList to avoid list-key like processing*/
		if ((strings.HasSuffix(uriList[len(uriList)-1], "]")) || (strings.HasSuffix(uriList[len(uriList)-1], "]/"))) { //splitUri chops off the leaf-list value having square brackets
			uriList[len(uriList)-1] = strings.SplitN(uriList[len(uriList)-1], "[", 2)[0]
			xfmrLogInfoAll("Uri list after removing leaf-list instance portion - %v", uriList)
		}
	}

	parentUriList := uriList[:len(uriList)-1]
	xfmrLogInfoAll("Parent uri list - %v", parentUriList)

	// Loop for the parent uri to check parent table existence
        for idx, path := range parentUriList {
		curUri += uriList[idx]

		/* Check for parent table for oc- yang lists*/
                keyList := rgpKeyExtract.FindAllString(path, -1)
		if len(keyList) > 0 {

			//Check for subtree existence
			curXpath, _, _ := XfmrRemoveXPATHPredicates(curUri)
			curXpathInfo, ok := xYangSpecMap[curXpath]
			if !ok {
				errStr := fmt.Sprintf("No entry found in xYangSpecMap for uri - %v", curUri)
				err = tlerr.InternalError{Format:errStr}
				parentTblExists = false
				break
			}
			if oper == GET {
				cdb = curXpathInfo.dbIndex
				xfmrLogInfoAll("db index for curXpath - %v is %v", curXpath, cdb)
				d = dbs[cdb]
			}
			if curXpathInfo.virtualTbl != nil && *curXpathInfo.virtualTbl {
				curUri += "/"
				continue
			}
			// Check for subtree case and invoke subscribe xfmr
			if len(curXpathInfo.xfmrFunc) > 0 {
				xfmrLogInfoAll("Found subtree for uri - %v", curUri)
				stParentTblExists := false
				stParentTblExists, err = verifyParentTblSubtree(dbs, curUri, curXpathInfo.xfmrFunc, oper, dbData)
				if err != nil {
					parentTblExists = false
					break
				}
				if !stParentTblExists {
					log.Warningf("Parent Table does not exist for uri %v", uri)
                                        err = tlerr.NotFound("Resource not found")
					parentTblExists = false
					break
				}
			} else {

				xfmrLogInfoAll("Check parent table for uri: %v", curUri)
				// Get Table and Key only for yang list instances
				xpathKeyExtRet, xerr := xpathKeyExtract(d, ygRoot, oper, curUri, uri, nil, subOpDataMap, txCache, nil)
				if xerr != nil {
					log.Warningf("Failed to get table and key for uri: %v err: %v", curUri, xerr)
					err = xerr
					parentTblExists = false
					break
				}
				if (xpathKeyExtRet.isVirtualTbl) {
					curUri += "/"
					continue
				}

				if len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) > 0 {
					// Check for Table existence
					xfmrLogInfoAll("DB Entry Check for uri: %v table: %v, key: %v", uri, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
					existsInDbData := false
					if oper == GET {
                                                // GET case - attempt to find in dbData before doing a dbGet in dbTableExists()
                                                existsInDbData = dbTableExistsInDbData(cdb, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, dbData)
					}
					// Read the table entry from DB
					if !existsInDbData {
						exists, derr := dbTableExists(d, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
						if derr != nil {
							return false, derr
						}
						if !exists {
							parentTblExists = false
							log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for uri %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, uri)
							err = tlerr.NotFound("Resource not found")
							break
						}
					}
				} else {
					// We always expect a valid table and key to be returned. Else we cannot validate parent check
					parentTblExists = false
					log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for uri %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, curUri)
					err = tlerr.NotFound("Resource not found")
					break
				}
			}
		}
		curUri += "/"
        }
        if !parentTblExists {
                // For all operations Parent Table has to exist
                return false, err
        }

        if yangType == YANG_LIST && (oper == UPDATE || oper == CREATE || oper == DELETE || oper == GET) {
                // For PATCH request the current table instance should exist for the operation to go through
                // For POST since the target URI is the parent URI, it should exist.
                // For DELETE we handle the table verification here to avoid any CVL error thrown for delete on non existent table
		xfmrLogInfoAll("Check last parent table for uri: %v", uri)
		xpath, _, xpathErr := XfmrRemoveXPATHPredicates(uri)
		if xpathErr != nil {
			log.Warningf("Xpath conversion didn't happen for Uri - %v, due to - %v", uri, xpathErr)
			return false, xpathErr
		}
		xpathInfo, ok := xYangSpecMap[xpath]
		if !ok {
			err = fmt.Errorf("xYangSpecMap does not contain xpath - %v", xpath)
			return false, err
		}
		virtualTbl := false
		if xpathInfo.virtualTbl != nil {
			virtualTbl = *xpathInfo.virtualTbl
		}
		if virtualTbl {
			xfmrLogInfoAll("Virtual table at uri - %v", uri)
			return true, nil
		}
		// Check for subtree case and invoke subscribe xfmr
		if len(xpathInfo.xfmrFunc) > 0 {
			if !((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {//uri points to entire list
			        xfmrLogInfoAll("subtree , whole list case for uri - %v", uri)
				return true, nil
			}
			xfmrLogInfoAll("Found subtree for uri - %v", uri)
			parentTblExists, err = verifyParentTblSubtree(dbs, uri, xpathInfo.xfmrFunc, oper, dbData)
			if err != nil {
				return false, err
			}
			if !parentTblExists {
				log.Warningf("Parent Table does not exist for uri %v", uri)
				err = tlerr.NotFound("Resource not found")
				return false, err
			}
			return true, nil
		}

		xpathKeyExtRet, xerr := xpathKeyExtract(d, ygRoot, oper, uri, uri, nil, subOpDataMap, txCache, nil)
		if xerr != nil {
			log.Warningf("xpathKeyExtract failed err: %v, table %v, key %v", xerr, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
			return false, xerr
		}
		if xpathKeyExtRet.isVirtualTbl {
			return true, nil
		}
		if len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) > 0 {
			// Read the table entry from DB
			exists := false
			var derr error
			if oper == GET {
				// GET case - find in dbData instead of doing a dbGet in dbTableExists()
				cdb = xpathInfo.dbIndex
				xfmrLogInfoAll("db index for xpath - %v is %v", xpath, cdb)
				exists = dbTableExistsInDbData(cdb, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, dbData)
				if !exists {
					exists, derr = dbTableExists(dbs[cdb], xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
					if derr != nil {
						return false, derr
					}
					if !exists {
						parentTblExists = false
						log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for uri %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, uri)
						err = tlerr.NotFound("Resource not found")
					}
				}
			} else {
				exists, derr = dbTableExists(d, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
			}
			if derr != nil {
				log.Warningf("GetEntry failed for table: %v, key: %v err: %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, derr)
				return false, derr
			}
			if !exists {
				log.Warningf("GetEntry failed for table: %v, key: %v err: %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, derr)
				err = tlerr.NotFound("Resource not found")
				return false, err
			} else {
				return true, nil
			}
		} else if !((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {//uri points to entire list
			return true, nil
		} else {
			log.Warningf("Unable to get valid table and key err: %v, table %v, key %v", xerr, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
			return false, xerr
		}
	} else if (yangType == YANG_CONTAINER && oper == DELETE && ((xpathInfo.keyName != nil && len(*xpathInfo.keyName) > 0) || len(xpathInfo.xfmrKey) > 0)) {
		// If the delete is at container level and the container is mapped to a unique table, then check for table existence to avoid CVL throwing error
		parentUri := ""
		if len(parentUriList) > 0 {
			parentUri = strings.Join(parentUriList, "/")
			parentUri = "/" + parentUri
		}
		// Get table for parent xpath
		parentTable, perr := dbTableFromUriGet(d, ygRoot, oper, parentUri, uri, nil, txCache, nil)
		// Get table for current xpath
		xpathKeyExtRet, cerr := xpathKeyExtract(d, ygRoot, oper, uri, uri, nil, subOpDataMap, txCache, nil)
		curKey := xpathKeyExtRet.dbKey
		curTable := xpathKeyExtRet.tableName
		if len(curTable) > 0 {
			if perr == nil && cerr == nil && (curTable != parentTable) && len(curKey) > 0 {
				exists, derr := dbTableExists(d, curTable, curKey, oper)
				if !exists {
					return true, derr
				} else {
					return true, nil
				}
			} else {
				return true, nil
			}
		} else {
			return true, nil
		}
	} else {
                // PUT at list is allowed to do a create if table does not exist else replace OR
                // This is a container or leaf at the end of the URI. Parent check already done and hence all operations are allowed
                return true, err
        }
}

/* Debug function to print the map data into file */
func printDbData(resMap map[int]map[db.DBNum]map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value, fileName string) {
	fp, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer fp.Close()
	for oper, dbRes := range resMap {
		fmt.Fprintf(fp, "-------------------------- REQ DATA -----------------------------\r\n")
		fmt.Fprintf(fp, "Oper Type : %v\r\n", oper)
		for d, dbMap := range dbRes {
			fmt.Fprintf(fp, "DB num : %v\r\n", d)
			for k, v := range dbMap {
				fmt.Fprintf(fp, "table name : %v\r\n", k)
				for ik, iv := range v {
					fmt.Fprintf(fp, "  key : %v\r\n", ik)
					for k, d := range iv.Field {
						fmt.Fprintf(fp, "    %v :%v\r\n", k, d)
					}
				}
			}
		}
	}
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
	fmt.Fprintf(fp, "-------------------------- YANG DEFAULT DATA --------------------\r\n")
	for k, v := range yangDefValMap {
		fmt.Fprintf(fp, "table name : %v\r\n", k)
		for ik, iv := range v {
			fmt.Fprintf(fp, "  key : %v\r\n", ik)
			for k, d := range iv.Field {
				fmt.Fprintf(fp, "    %v :%v\r\n", k, d)
			}
		}
	}
	fmt.Fprintf(fp, "-----------------------------------------------------------------\r\n")
}
