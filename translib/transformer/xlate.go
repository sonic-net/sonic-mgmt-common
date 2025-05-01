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
	"reflect"
	"sort"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

var XlateFuncs = make(map[string]reflect.Value)

var (
	ErrParamsNotAdapted = errors.New("The number of params is not adapted.")
)

func XlateFuncBind(name string, fn interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(name + " is not valid Xfmr function.")
		}
	}()

	if _, ok := XlateFuncs[name]; !ok {
		v := reflect.ValueOf(fn)
		v.Type().NumIn()
		XlateFuncs[name] = v
	} else {
		xfmrLogInfo("Duplicate entry found in the XlateFunc map ", name)
	}
	return
}
func IsXlateFuncBinded(name string) bool {
	if _, ok := XlateFuncs[name]; !ok {
		return false
	} else {
		return true
	}
}
func XlateFuncCall(name string, params ...interface{}) (result []reflect.Value, err error) {
	if _, ok := XlateFuncs[name]; !ok {
		log.Warning("Xfmr function does not exist: ", name)
		return nil, nil
	}
	if len(params) != XlateFuncs[name].Type().NumIn() {
		log.Warning("Error parameters not adapted")
		return nil, nil
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	result = XlateFuncs[name].Call(in)
	return result, nil
}

func TraverseDb(dbs [db.MaxDB]*db.DB, spec KeySpec, result *map[db.DBNum]map[string]map[string]db.Value,
	parentKey *db.Key, dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, reqCtxt context.Context) error {
	var dataMap = make(RedisDbMap)

	for i := db.ApplDB; i < db.MaxDB; i++ {
		dataMap[i] = make(map[string]map[string]db.Value)
	}

	err := traverseDbHelper(dbs, &spec, &dataMap, parentKey, dbTblKeyGetCache, reqCtxt)
	if err != nil {
		xfmrLogDebug("Didn't get all data from traverseDbHelper")
		return err
	}
	/* db data processing */
	curMap := make(map[Operation]map[db.DBNum]map[string]map[string]db.Value)
	curMap[GET] = dataMap
	err = dbDataXfmrHandler(curMap)
	if err != nil {
		log.Warning("No conversion in dbdata-xfmr")
		return err
	}

	for oper, dbData := range curMap {
		if oper == GET {
			for dbNum, tblData := range dbData {
				mapCopy((*result)[dbNum], tblData)
			}
		}
	}
	return nil
}

func traverseDbHelper(dbs [db.MaxDB]*db.DB, spec *KeySpec, result *map[db.DBNum]map[string]map[string]db.Value,
	parentKey *db.Key, dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, reqCtxt context.Context) error {
	var err error
	if isReqContextCancelled(reqCtxt) {
		err = tlerr.RequestContextCancelled("Client request's context cancelled.", reqCtxt.Err())
		log.Warningf(err.Error())
		return err
	}

	var dbOpts db.Options = getDBOptions(spec.DbNum)

	separator := dbOpts.KeySeparator

	if spec.Key.Len() > 0 && !spec.IsPartialKey {
		// get an entry with a specific key
		if spec.Ts.Name != XFMR_NONE_STRING { // Do not traverse for NONE table
			data, err := dbs[spec.DbNum].GetEntry(&spec.Ts, spec.Key)
			dbKeyStr := strings.Join(spec.Key.Comp, separator)
			if err != nil {
				updateDbDataMapAndKeyCache(dbKeyStr, &data, spec, result, dbTblKeyGetCache, false)
				if log.V(5) {
					log.Warningf("Didn't get data for tbl(%v), key(%v) in traverseDbHelper", spec.Ts.Name, spec.Key)
				}
				return err
			}
			updateDbDataMapAndKeyCache(dbKeyStr, &data, spec, result, dbTblKeyGetCache, true)
		}
		if len(spec.Child) > 0 {
			for _, ch := range spec.Child {
				if err = traverseDbHelper(dbs, &ch, result, &spec.Key, dbTblKeyGetCache, reqCtxt); err != nil &&
					isReqContextCancelledError(err) {
					break
				}
			}
		}
	} else {
		spec.Key.Comp = append(spec.Key.Comp, "*")
		// TODO - GetEntry support with regex patten, 'abc*' for optimization
		if spec.Ts.Name != XFMR_NONE_STRING { //Do not traverse for NONE table
			tblObj, err := dbs[spec.DbNum].GetTablePattern(&spec.Ts, *db.NewKey("*"))
			if err != nil {
				log.Warningf("GetTablePattern returned error %v for tbl(%v) in traverseDbHelper", err, spec.Ts.Name)
				return err
			}
			dbKeys, err := tblObj.GetKeys()
			if err != nil {
				log.Warningf("Table.GetKeys returned error %v for tbl(%v) in traverseDbHelper", err, spec.Ts.Name)
				return err
			}
			xfmrLogDebug("keys for table %v in DB %v are %v", spec.Ts.Name, spec.DbNum, dbKeys)
			parentDbKeyStr := ""
			if parentKey != nil && !spec.IgnoreParentKey {
				parentDbKeyStr = strings.Join((*parentKey).Comp, separator)
			}
			for _, dbKey := range dbKeys {
				dbKeyStr := strings.Join(dbKey.Comp, separator)
				if len(parentDbKeyStr) > 0 {
					// TODO - multi-depth with a custom delimiter
					if !strings.Contains(dbKeyStr, parentDbKeyStr) {
						continue
					}
				}
				data, err := tblObj.GetEntry(dbKey)
				if err != nil {
					log.Warningf("Table.GetEntry returned error %v for tbl(%v), and the key %v in traverseDbHelper", err, spec.Ts.Name, dbKey)
					updateDbDataMapAndKeyCache(dbKeyStr, &data, spec, result, dbTblKeyGetCache, false)
				} else if data.IsPopulated() {
					updateDbDataMapAndKeyCache(dbKeyStr, &data, spec, result, dbTblKeyGetCache, true)
				}
				if len(spec.Child) > 0 {
					for _, ch := range spec.Child {
						if err = traverseDbHelper(dbs, &ch, result, &dbKey, dbTblKeyGetCache, reqCtxt); err != nil &&
							isReqContextCancelledError(err) {
							break
						}
					}
				}
			}
		} else if len(spec.Child) > 0 {
			for _, ch := range spec.Child {
				err = traverseDbHelper(dbs, &ch, result, &spec.Key, dbTblKeyGetCache, reqCtxt)
				if err != nil && isReqContextCancelledError(err) {
					break
				}
			}
		}
	}
	return err
}

func updateDbDataMapAndKeyCache(dbKeyStr string, data *db.Value, spec *KeySpec,
	result *map[db.DBNum]map[string]map[string]db.Value, dbTblKeyGetCache map[db.DBNum]map[string]map[string]bool, readOk bool) {
	if (*result)[spec.DbNum][spec.Ts.Name] == nil {
		(*result)[spec.DbNum][spec.Ts.Name] = map[string]db.Value{dbKeyStr: *data}
	} else {
		(*result)[spec.DbNum][spec.Ts.Name][dbKeyStr] = *data
	}
	if dbTblKeyGetCache == nil {
		dbTblKeyGetCache = make(map[db.DBNum]map[string]map[string]bool)
	}
	if dbTblKeyGetCache[spec.DbNum] == nil {
		dbTblKeyGetCache[spec.DbNum] = make(map[string]map[string]bool)
	}
	if dbTblKeyGetCache[spec.DbNum][spec.Ts.Name] == nil {
		dbTblKeyGetCache[spec.DbNum][spec.Ts.Name] = make(map[string]bool)
	}
	dbTblKeyGetCache[spec.DbNum][spec.Ts.Name][dbKeyStr] = readOk
}

func XlateUriToKeySpec(uri string, requestUri string, ygRoot *ygot.GoStruct, t *interface{}, txCache interface{},
	qParams QueryParams, dbs [db.MaxDB]*db.DB, dbTblKeyCache map[string]tblKeyCache, dbDataMap RedisDbMap) (*[]KeySpec, error) {

	var err error
	var retdbFormat = make([]KeySpec, 0)
	// In case of SONIC yang, the tablename and key info is available in the xpath
	if isSonicYang(uri) {
		/* Extract the xpath and key from input xpath */
		xpath, keyStr, tableName := sonicXpathKeyExtract(uri)
		if tblSpecInfo, ok := xDbSpecMap[tableName]; ok && keyStr != "" && hasKeyValueXfmr(tableName) {
			/* key from URI should be converted into redis-db key, to read data */
			keyStr, err = dbKeyValueXfmrHandler(CREATE, tblSpecInfo.dbIndex, tableName, keyStr, false)
			if err != nil {
				log.Warningf("Value-xfmr for table(%v) & key(%v) didn't do conversion.", tableName, keyStr)
				return &retdbFormat, err
			}
		}

		retdbFormat = fillSonicKeySpec(xpath, tableName, keyStr, qParams.content)
	} else {
		var reqUriXpath string
		/* Extract the xpath and key from input xpath */
		xpath, _, _ := XfmrRemoveXPATHPredicates(uri)
		d := dbs[db.ConfigDB]
		if xpathInfo, ok := xYangSpecMap[xpath]; ok {
			d = dbs[xpathInfo.dbIndex]
		}
		retData, _ := xpathKeyExtractForGet(d, ygRoot, GET, uri, requestUri, &dbDataMap, nil, txCache, dbTblKeyCache, dbs)
		if requestUri == uri {
			reqUriXpath = retData.xpath
		} else {
			reqUriXpath, _, _ = XfmrRemoveXPATHPredicates(requestUri)
		}
		retdbFormat = fillKeySpecs(uri, reqUriXpath, &qParams, retData.xpath, retData.tableName, retData.dbKey, &retdbFormat, "")
		log.V(5).Infof("filled keyspec: %v; retData: %v; reqUriXpath: %v", retdbFormat, retData, reqUriXpath)
	}

	return &retdbFormat, err
}

func fillKeySpecs(uri string, reqUriXpath string, qParams *QueryParams, yangXpath string, tableName string, keyStr string, retdbFormat *[]KeySpec, parentKey string) []KeySpec {
	var err error
	if xYangSpecMap == nil {
		return *retdbFormat
	}
	_, ok := xYangSpecMap[yangXpath]
	if ok {
		xpathInfo := xYangSpecMap[yangXpath]
		if len(tableName) > 0 {
			dbFormat := KeySpec{}
			dbFormat.Ts.Name = tableName
			dbFormat.DbNum = xpathInfo.dbIndex
			if (len(parentKey) == 0 && len(xYangSpecMap[yangXpath].xfmrKey) > 0) || xYangSpecMap[yangXpath].keyName != nil {
				dbFormat.IgnoreParentKey = true
			} else {
				dbFormat.IgnoreParentKey = false
			}
			if keyStr != "" {
				if tblSpecInfo, ok := xDbSpecMap[dbFormat.Ts.Name]; ok && tblSpecInfo.hasXfmrFn {
					/* key from URI should be converted into redis-db key, to read data */
					isPartialKey := verifyPartialKeyForOc(uri, reqUriXpath, keyStr)
					keyStr, err = dbKeyValueXfmrHandler(CREATE, dbFormat.DbNum, dbFormat.Ts.Name, keyStr, isPartialKey)
					if err != nil {
						log.Warningf("Value-xfmr for table(%v) & key(%v) didn't do conversion.", dbFormat.Ts.Name, keyStr)
					}
				}
				dbFormat.Key.Comp = append(dbFormat.Key.Comp, keyStr)
			}
			for _, child := range xpathInfo.childTable {
				if child == dbFormat.Ts.Name {
					continue
				}
				if xDbSpecMap != nil {
					if _, ok := xDbSpecMap[child]; ok {
						chlen := len(xDbSpecMap[child].yangXpath)
						if chlen > 0 {
							children := make([]KeySpec, 0)
							ygXpathTbl := make(map[string]bool)
							for _, childXpath := range xDbSpecMap[child].yangXpath {
								if _, xOk := ygXpathTbl[childXpath]; !xOk {
									ygXpathTbl[childXpath] = true
								} else {
									continue
								}
								if isChildTraversalRequired(reqUriXpath, qParams, childXpath) {
									if _, ok := xYangSpecMap[childXpath]; ok {
										tableNm := ""
										if xYangSpecMap[childXpath].tableName != nil {
											tableNm = *xYangSpecMap[childXpath].tableName
										}
										children = fillKeySpecs(childXpath, reqUriXpath, qParams, childXpath, tableNm, "", &children, keyStr)
										dbFormat.Child = append(dbFormat.Child, children...)
									}
								}
							}
						}
					}
				}
			}
			*retdbFormat = append(*retdbFormat, dbFormat)
		} else {
			for _, child := range xpathInfo.childTable {
				if xDbSpecMap != nil {
					if _, ok := xDbSpecMap[child]; ok {
						chlen := len(xDbSpecMap[child].yangXpath)
						if chlen > 0 {
							ygXpathTbl := make(map[string]bool)
							for _, childXpath := range xDbSpecMap[child].yangXpath {
								if _, xOk := ygXpathTbl[childXpath]; !xOk {
									ygXpathTbl[childXpath] = true
								} else {
									continue
								}
								if isChildTraversalRequired(reqUriXpath, qParams, childXpath) {
									if _, ok := xYangSpecMap[childXpath]; ok {
										tableNm := ""
										if xYangSpecMap[childXpath].tableName != nil {
											tableNm = *xYangSpecMap[childXpath].tableName
										}
										*retdbFormat = fillKeySpecs(childXpath, reqUriXpath, qParams, childXpath, tableNm, "", retdbFormat, keyStr)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return *retdbFormat
}

func fillSonicKeySpec(xpath string, tableName string, keyStr string, content ContentType) []KeySpec {

	var retdbFormat = make([]KeySpec, 0)

	if tableName != "" {
		dbFormat := KeySpec{}
		dbFormat.Ts.Name = tableName
		cdb := db.ConfigDB
		if _, ok := xDbSpecMap[tableName]; ok {
			if (xDbSpecMap[tableName].dbEntry == nil) || ((content == QUERY_CONTENT_CONFIG) && (xDbSpecMap[tableName].dbEntry.ReadOnly())) || ((content == QUERY_CONTENT_NONCONFIG) && (!xDbSpecMap[tableName].dbEntry.ReadOnly())) {
				return retdbFormat
			}
			cdb = xDbSpecMap[tableName].dbIndex
		}
		dbFormat.DbNum = cdb
		if keyStr != "" {
			dbFormat.Key.Comp = append(dbFormat.Key.Comp, keyStr)
		}
		retdbFormat = append(retdbFormat, dbFormat)
	} else {
		// If table name not available in xpath get top container name
		container := xpath
		if xDbSpecMap != nil {
			if _, ok := xDbSpecMap[container]; ok {
				dbInfo := xDbSpecMap[container]
				if dbInfo.yangType == YANG_CONTAINER {
					for dir := range dbInfo.dbEntry.Dir {
						_, ok := xDbSpecMap[dir]
						if ok && xDbSpecMap[dir].yangType == YANG_CONTAINER {
							if (xDbSpecMap[dir].dbEntry == nil) || ((content == QUERY_CONTENT_CONFIG) && (xDbSpecMap[dir].dbEntry.ReadOnly())) || ((content == QUERY_CONTENT_NONCONFIG) && (!xDbSpecMap[dir].dbEntry.ReadOnly())) {
								continue
							}
							cdb := xDbSpecMap[dir].dbIndex
							dbFormat := KeySpec{}
							dbFormat.Ts.Name = dir
							dbFormat.DbNum = cdb
							retdbFormat = append(retdbFormat, dbFormat)
						}
					}
				}
			}
		}
	}
	return retdbFormat
}

func XlateToDb(path string, oper int, d *db.DB, yg *ygot.GoStruct, yt *interface{}, jsonPayload []byte, txCache interface{}, skipOrdTbl *bool) (map[Operation]RedisDbMap, map[string]map[string]db.Value, map[string]map[string]db.Value, error) {

	requestUri := path
	jsonData := make(map[string]interface{})
	opcode := Operation(oper)

	device := (*yg).(*ocbinds.Device)
	jsonBytes, err := ocbinds.EmitJSON(device, nil)
	if err == nil {
		err = json.Unmarshal(jsonBytes, &jsonData)
	}
	if err != nil {
		errStr := "Error: failed to unmarshal json."
		err = tlerr.InternalError{Format: errStr}
		return nil, nil, nil, err
	}

	// Map contains table.key.fields
	var result = make(map[Operation]RedisDbMap)
	var yangDefValMap = make(map[string]map[string]db.Value)
	var yangAuxValMap = make(map[string]map[string]db.Value)
	switch opcode {
	case CREATE:
		xfmrLogInfo("CREATE case")
		err = dbMapCreate(d, yg, opcode, path, requestUri, jsonData, result, yangDefValMap, yangAuxValMap, txCache)
		if err != nil {
			log.Warning("Data translation from YANG to db failed for create request.")
		}

	case UPDATE:
		xfmrLogInfo("UPDATE case")
		err = dbMapUpdate(d, yg, opcode, path, requestUri, jsonData, result, yangDefValMap, yangAuxValMap, txCache)
		if err != nil {
			log.Warning("Data translation from YANG to db failed for update request.")
		}

	case REPLACE:
		xfmrLogInfo("REPLACE case")
		err = dbMapUpdate(d, yg, opcode, path, requestUri, jsonData, result, yangDefValMap, yangAuxValMap, txCache)
		if err != nil {
			log.Warning("Data translation from YANG to db failed for replace request.")
		}

	case DELETE:
		xfmrLogInfo("DELETE case")
		err = dbMapDelete(d, yg, opcode, path, requestUri, jsonData, result, txCache, skipOrdTbl)
		if err != nil {
			log.Warning("Data translation from YANG to db failed for delete request.")
		}
	}
	return result, yangDefValMap, yangAuxValMap, err
}

func GetAndXlateFromDB(uri string, ygRoot *ygot.GoStruct, dbs [db.MaxDB]*db.DB,
	txCache interface{}, qParams QueryParams, reqCtxt context.Context, ygSchema *yang.Entry) ([]byte, bool, error) {
	var err error
	var payload []byte
	var inParamsForGet xlateFromDbParams
	var processReq bool
	inParamsForGet.queryParams = qParams
	inParamsForGet.reqCtxt = reqCtxt
	inParamsForGet.ygSchema = ygSchema
	xfmrLogInfo("received xpath = %v", uri)
	requestUri := uri

	if len(qParams.fields) > 0 {
		yngNdType, nd_err := getYangNodeTypeFromUri(uri)
		if nd_err != nil {
			return []byte("{}"), false, nd_err
		} else if (yngNdType == YANG_LEAF) || (yngNdType == YANG_LEAF_LIST) {
			err = tlerr.InvalidArgsError{Format: "Bad Request - fields query parameter specified on a terminal node uri."}
			return []byte("{}"), false, err
		}
	} else {
		processReq, err = contentQParamTgtEval(uri, qParams)
		if err != nil {
			return []byte("{}"), false, err
		}
		if !processReq {
			xfmrLogInfo("further processing of request not needed due to content query param.")
			/* translib fills requested list-instance into ygot, but when there is content-mismatch
			   we have to send empty payload response.So distinguish this case in common_app we send this err
			*/
			if IsListNode(uri) {
				return []byte("{}"), true, tlerr.InternalError{Format: QUERY_CONTENT_MISMATCH_ERR}
			}
			return []byte("{}"), true, err
		}
	}
	dbTblKeyCache := make(map[string]tblKeyCache)
	var dbresult = make(RedisDbMap)
	for i := db.ApplDB; i < db.MaxDB; i++ {
		dbresult[i] = make(map[string]map[string]db.Value)
	}
	keySpec, _ := XlateUriToKeySpec(uri, requestUri, ygRoot, nil, txCache, qParams, dbs, dbTblKeyCache, dbresult)

	inParamsForGet.dbTblKeyGetCache = make(map[db.DBNum]map[string]map[string]bool)
	for _, spec := range *keySpec {
		err := TraverseDb(dbs, spec, &dbresult, nil, inParamsForGet.dbTblKeyGetCache, inParamsForGet.reqCtxt)
		if err != nil {
			xfmrLogDebug("TraverseDb() didn't fetch data.")
			if isReqContextCancelledError(err) {
				return payload, true, err
			}
		}
	}

	isEmptyPayload := false
	inParamsForGet.xfmrDbTblKeyCache = dbTblKeyCache
	payload, isEmptyPayload, err = XlateFromDb(uri, ygRoot, dbs, dbresult, txCache, inParamsForGet)
	if err != nil {
		return payload, true, err
	}

	return payload, isEmptyPayload, err
}

func XlateFromDb(uri string, ygRoot *ygot.GoStruct, dbs [db.MaxDB]*db.DB, data RedisDbMap, txCache interface{}, inParamsForGet xlateFromDbParams) ([]byte, bool, error) {

	var err error
	var result []byte
	var dbData = make(RedisDbMap)
	var cdb db.DBNum = db.ConfigDB
	var xpath string

	dbData = data
	requestUri := uri
	/* Check if the parent table exists for RFC compliance */
	var exists bool
	subOpMapDiscard := make(map[Operation]*RedisDbMap)
	exists, err = verifyParentTable(nil, dbs, ygRoot, GET, uri, dbData, txCache, subOpMapDiscard, inParamsForGet.xfmrDbTblKeyCache)
	xfmrLogDebug("verifyParentTable() returned - exists - %v, err - %v", exists, err)
	if err != nil {
		log.Warningf("Cannot perform GET Operation on URI %v due to - %v", uri, err)
		return []byte(""), true, err
	}
	if !exists {
		err = tlerr.NotFoundError{Format: "Resource Not found"}
		return []byte(""), true, err
	}

	if isSonicYang(uri) {
		lxpath, keyStr, tableName := sonicXpathKeyExtract(uri)
		xpath = lxpath
		if tableName != "" {
			dbInfo, ok := xDbSpecMap[tableName]
			if !ok {
				log.Warningf("No entry in xDbSpecMap for xpath %v", tableName)
			} else {
				cdb = dbInfo.dbIndex
			}
			tokens := strings.Split(xpath, "/")
			// Format /module:container/tableName/listname[key]/fieldName
			if tokens[SONIC_TABLE_INDEX] == tableName {
				fieldName := ""
				if len(tokens) > SONIC_FIELD_INDEX {
					fieldName = tokens[SONIC_FIELD_INDEX]
					dbSpecField := tableName + "/" + fieldName
					dbSpecFieldInfo, ok := xDbSpecMap[dbSpecField]
					if ok && fieldName != "" {
						yangNodeType := xDbSpecMap[dbSpecField].yangType
						if yangNodeType == YANG_LEAF_LIST {
							fieldName = fieldName + "@"
						}
						if (yangNodeType == YANG_LEAF_LIST) || (yangNodeType == YANG_LEAF) {
							dbData[cdb], err = extractFieldFromDb(tableName, keyStr, fieldName, data[cdb])
							// return resource not found when the leaf/leaf-list instance(not entire leaf-list GET) not found
							if (err != nil) && ((yangNodeType == YANG_LEAF) || ((yangNodeType == YANG_LEAF_LIST) && (strings.HasSuffix(uri, "]") || strings.HasSuffix(uri, "]/")))) {
								return []byte(""), true, err
							}
							if (yangNodeType == YANG_LEAF_LIST) && ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {
								leafListInstVal, valErr := extractLeafListInstFromUri(uri)
								if valErr != nil {
									return []byte(""), true, valErr
								}
								if dbSpecFieldInfo.xfmrValue != nil {
									inParams := formXfmrDbInputRequest(CREATE, cdb, tableName, keyStr, fieldName, leafListInstVal)
									retVal, err := valueXfmrHandler(inParams, *dbSpecFieldInfo.xfmrValue)
									if err != nil {
										log.Warningf("value-xfmr:fldpath(\"%v\") val(\"%v\"):err(\"%v\").", dbSpecField, leafListInstVal, err)
										return []byte(""), true, err
									}
									leafListInstVal = retVal
								}
								if leafListInstExists(dbData[cdb][tableName][keyStr].Field[fieldName], leafListInstVal) {
									/* Since translib already fills in ygRoot with queried leaf-list instance, do not
									   fill in resFldValMap or else Unmarshall of payload(resFldValMap) into ygotTgt in
									   app layer will create duplicate instances in result.
									*/
									log.Info("Queried leaf-list instance exists.")
									return []byte("{}"), false, nil
								} else {
									xfmrLogDebug("Queried leaf-list instance does not exist - %v", uri)
									return []byte(""), true, tlerr.NotFoundError{Format: "Resource not found"}
								}
							}
						}
					}
				}
			}
		}
	} else {
		lxpath, _, _ := XfmrRemoveXPATHPredicates(uri)
		xpath = lxpath
		if _, ok := xYangSpecMap[xpath]; ok {
			cdb = xYangSpecMap[xpath].dbIndex
		}
	}
	dbTblKeyGetCache := inParamsForGet.dbTblKeyGetCache
	tblXfmrCache := inParamsForGet.xfmrDbTblKeyCache
	qparams := inParamsForGet.queryParams
	ygSchema := inParamsForGet.ygSchema
	inParamsForGet = formXlateFromDbParams(dbs[cdb], dbs, cdb, ygRoot, uri, requestUri, xpath, GET, "", "",
		&dbData, txCache, nil, qparams, inParamsForGet.reqCtxt, nil)
	inParamsForGet.dbTblKeyGetCache = dbTblKeyGetCache
	inParamsForGet.xfmrDbTblKeyCache = tblXfmrCache
	inParamsForGet.ygSchema = ygSchema
	payload, isEmptyPayload, err := dbDataToYangJsonCreate(inParamsForGet)
	xfmrLogDebug("Payload generated : ", payload)

	if err != nil {
		log.Warning("Couldn't create json response from DB data.")
		return nil, isEmptyPayload, err
	}
	xfmrLogInfo("Created json response from DB data.")

	result = []byte(payload)
	return result, isEmptyPayload, err
}

func extractFieldFromDb(tableName string, keyStr string, fieldName string, data map[string]map[string]db.Value) (map[string]map[string]db.Value, error) {

	var dbVal db.Value
	var dbData = make(map[string]map[string]db.Value)
	var err error

	if tableName != "" && keyStr != "" && fieldName != "" {
		if data[tableName][keyStr].Field != nil {
			fldVal, fldValExists := data[tableName][keyStr].Field[fieldName]
			if fldValExists {
				dbData[tableName] = make(map[string]db.Value)
				dbVal.Field = make(map[string]string)
				dbVal.Field[fieldName] = fldVal
				dbData[tableName][keyStr] = dbVal
			} else {
				log.Warningf("Field %v doesn't exist in table - %v, instance - %v", fieldName, tableName, keyStr)
				err = tlerr.NotFoundError{Format: "Resource not found"}
			}
		}
	}
	return dbData, err
}

func GetModuleNmFromPath(uri string) (string, error) {
	xfmrLogDebug("received URI %s to extract module name from ", uri)
	moduleNm, err := uriModuleNameGet(uri)
	return moduleNm, err
}

func GetOrdTblList(xfmrTbl string, uriModuleNm string) []string {
	var ordTblList []string
	processedTbl := false
	var sncMdlList []string = getYangMdlToSonicMdlList(uriModuleNm)

	for _, sonicMdlNm := range sncMdlList {
		sonicMdlTblInfo := xDbSpecTblSeqnMap[sonicMdlNm]
		for _, ordTblNm := range sonicMdlTblInfo.OrdTbl {
			if xfmrTbl == ordTblNm {
				xfmrLogInfo("Found sonic module(%v) whose ordered table list contains table %v", sonicMdlNm, xfmrTbl)
				ordTblList = sonicMdlTblInfo.DepTbl[xfmrTbl].DepTblWithinMdl
				processedTbl = true
				break
			}
		}
		if processedTbl {
			break
		}
	}
	return ordTblList
}

func GetXfmrOrdTblList(xfmrTbl string) []string {
	/* get the table hierarchy read from json file */
	var ordTblList []string
	if _, ok := sonicOrdTblListMap[xfmrTbl]; ok {
		ordTblList = sonicOrdTblListMap[xfmrTbl]
	}
	return ordTblList
}

func GetTablesToWatch(xfmrTblList []string, uriModuleNm string) []string {
	var depTblList []string
	depTblMap := make(map[string]bool) //create to avoid duplicates in depTblList, serves as a Set
	processedTbl := false
	var sncMdlList []string
	var lXfmrTblList []string

	sncMdlList = getYangMdlToSonicMdlList(uriModuleNm)

	// remove duplicates from incoming list of tables
	xfmrTblMap := make(map[string]bool) //create to avoid duplicates in xfmrTblList
	for _, xfmrTblNm := range xfmrTblList {
		xfmrTblMap[xfmrTblNm] = true
	}
	for xfmrTblNm := range xfmrTblMap {
		lXfmrTblList = append(lXfmrTblList, xfmrTblNm)
	}

	for _, xfmrTbl := range lXfmrTblList {
		processedTbl = false
		//can be optimized if there is a way to know all sonic modules, a given OC-Yang spans over
		for _, sonicMdlNm := range sncMdlList {
			sonicMdlTblInfo := xDbSpecTblSeqnMap[sonicMdlNm]
			for _, ordTblNm := range sonicMdlTblInfo.OrdTbl {
				if xfmrTbl == ordTblNm {
					xfmrLogInfo("Found sonic module(%v) whose ordered table list contains table %v", sonicMdlNm, xfmrTbl)
					ldepTblList := sonicMdlTblInfo.DepTbl[xfmrTbl].DepTblAcrossMdl
					for _, depTblNm := range ldepTblList {
						depTblMap[depTblNm] = true
					}
					//assumption that a table belongs to only one sonic module
					processedTbl = true
					break
				}
			}
			if processedTbl {
				break
			}
		}
		if !processedTbl {
			depTblMap[xfmrTbl] = false
		}
	}
	for depTbl := range depTblMap {
		depTblList = append(depTblList, depTbl)
	}
	return depTblList
}

func CallRpcMethod(path string, body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	const (
		RPC_XFMR_RET_ARGS     = 2
		RPC_XFMR_RET_VAL_INDX = 0
		RPC_XFMR_RET_ERR_INDX = 1
	)
	var err error
	var ret []byte
	var data []reflect.Value
	var rpcFunc = ""

	// TODO - check module name
	if isSonicYang(path) {
		if rpcFuncNm, ok := xDbRpcSpecMap[path]; ok {
			rpcFunc = rpcFuncNm
		}
	} else {
		if rpcFuncNm, ok := xYangRpcSpecMap[path]; ok {
			rpcFunc = rpcFuncNm
		}
	}

	if rpcFunc != "" {
		xfmrLogInfo("RPC callback invoked (%v) \r\n", rpcFunc)
		data, err = XlateFuncCall(rpcFunc, body, dbs)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			if len(data) == RPC_XFMR_RET_ARGS {
				// rpc xfmr callback returns err as second value in return data list from <xfmr_func>.Call()
				if data[RPC_XFMR_RET_ERR_INDX].Interface() != nil {
					err = data[RPC_XFMR_RET_ERR_INDX].Interface().(error)
					if err != nil {
						log.Warningf("Transformer function(\"%v\") returned error - %v.", rpcFunc, err)
					}
				}
			}

			if data[RPC_XFMR_RET_VAL_INDX].Interface() != nil {
				retVal, retOk := data[RPC_XFMR_RET_VAL_INDX].Interface().([]byte)
				if retOk {
					ret = retVal
				}
			}
		}
	} else {
		log.Warning("Not supported RPC", path)
		err = tlerr.NotSupported("Not supported RPC")
	}
	return ret, err
}

func AddModelCpbltInfo() map[string]*mdlInfo {
	return xMdlCpbltMap
}

func IsTerminalNode(uri string) (bool, error) {
	xpath, _, err := XfmrRemoveXPATHPredicates(uri)
	if xpathData, ok := xYangSpecMap[xpath]; ok {
		if !xpathData.hasNonTerminalNode {
			return true, nil
		}
	} else {
		log.Warningf("xYangSpecMap data not found for xpath : %v", xpath)
		errStr := "xYangSpecMap data not found for xpath."
		err = tlerr.InternalError{Format: errStr}
	}

	return false, err
}

func IsLeafNode(uri string) bool {
	result := false
	yngNdType, err := getYangNodeTypeFromUri(uri)
	if (err == nil) && (yngNdType == YANG_LEAF) {
		result = true
	}
	return result
}

func IsLeafListNode(uri string) bool {
	result := false
	yngNdType, err := getYangNodeTypeFromUri(uri)
	if (err == nil) && (yngNdType == YANG_LEAF_LIST) {
		result = true
	}
	return result
}

func IsListNode(uri string) bool {
	result := false
	yngNdType, err := getYangNodeTypeFromUri(uri)
	if (err == nil) && (yngNdType == YANG_LIST) {
		result = true
	}
	return result
}

func tableKeysToBeSorted(tblNm string) bool {
	/* function to decide whether to sort table keys.
	Required when a sonic table has more than 1 lists
	with keys having leaf-refs to each other, i.e table has primary and secondary keys
	*/
	areTblKeysToBeSorted := false
	TBL_LST_CNT_NO_SEC_KEY := 1 //Tables having primary and secondary keys have more than one lists defined in sonic yang
	if dbSpecInfo, ok := xDbSpecMap[tblNm]; ok {
		if len(dbSpecInfo.listName) > TBL_LST_CNT_NO_SEC_KEY {
			areTblKeysToBeSorted = true
		}
	} else {
		log.Warning("xDbSpecMap data not found for ", tblNm)
	}
	xfmrLogInfo("Table %v keys should be sorted - %v", tblNm, areTblKeysToBeSorted)
	return areTblKeysToBeSorted
}

func SortSncTableDbKeys(tableName string, dbKeyMap map[string]db.Value) []string {
	var ordDbKey []string

	if tableKeysToBeSorted(tableName) {

		m := make(map[string]int)
		for tblKey := range dbKeyMap {
			keyList := strings.Split(tblKey, "|")
			m[tblKey] = len(keyList)
		}

		type kv struct {
			Key   string
			Value int
		}

		var ss []kv
		for k, v := range m {
			ss = append(ss, kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		for _, kv := range ss {
			ordDbKey = append(ordDbKey, kv.Key)
		}

	} else {

		// Restore the order as in the original map in case of single list in table case and error case
		if len(ordDbKey) == 0 {
			for tblKey := range dbKeyMap {
				ordDbKey = append(ordDbKey, tblKey)
			}
		}
	}

	return ordDbKey
}
