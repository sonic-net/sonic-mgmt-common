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
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

func tblKeyDataGet(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, cdb db.DBNum) ([]string, bool, error) {
	var err error
	var dbs [db.MaxDB]*db.DB
	var tblList []string
	dbs[cdb] = xlateParams.d
	isVirtualTbl := false

	xfmrLogDebug("Get table data for  (\"%v\")", xlateParams.uri)
	if (xYangSpecMap[xlateParams.xpath].tableName != nil) && (len(*xYangSpecMap[xlateParams.xpath].tableName) > 0) {
		tblList = append(tblList, *xYangSpecMap[xlateParams.xpath].tableName)
	} else if xYangSpecMap[xlateParams.xpath].xfmrTbl != nil {
		xfmrTblFunc := *xYangSpecMap[xlateParams.xpath].xfmrTbl
		if len(xfmrTblFunc) > 0 {
			inParams := formXfmrInputRequest(xlateParams.d, dbs, cdb, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, xlateParams.keyName, dbDataMap, nil, nil, xlateParams.txCache)
			tblList, err = xfmrTblHandlerFunc(xfmrTblFunc, inParams, xlateParams.xfmrDbTblKeyCache)
			if err != nil {
				return tblList, isVirtualTbl, err
			}
			if inParams.isVirtualTbl != nil {
				isVirtualTbl = *(inParams.isVirtualTbl)
			}
		}
	}
	tbl := xlateParams.tableName
	if tbl != "" {
		if !contains(tblList, tbl) {
			tblList = append(tblList, tbl)
		}
	}
	return tblList, isVirtualTbl, err
}

func subTreeXfmrDelDataGet(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, cdb db.DBNum, spec *yangXpathInfo, chldSpec *yangXpathInfo, subTreeResMap *map[string]map[string]db.Value) error {
	var dbs [db.MaxDB]*db.DB
	dbs[cdb] = xlateParams.d

	xfmrLogDebug("Handle subtree for  (\"%v\")", xlateParams.uri)
	if len(chldSpec.xfmrFunc) > 0 {
		if (len(spec.xfmrFunc) == 0) || ((len(spec.xfmrFunc) > 0) &&
			(spec.xfmrFunc != chldSpec.xfmrFunc)) {
			inParams := formXfmrInputRequest(xlateParams.d, dbs, cdb, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, "",
				dbDataMap, xlateParams.subOpDataMap, nil, xlateParams.txCache)
			retMap, err := xfmrHandler(inParams, chldSpec.xfmrFunc)
			if err != nil {
				xfmrLogDebug("Error returned by %v: %v", chldSpec.xfmrFunc, err)
				return err
			}
			mapCopy(*subTreeResMap, retMap)
			if xlateParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
				for _, tblNm := range *inParams.pCascadeDelTbl {
					if !contains(*xlateParams.pCascadeDelTbl, tblNm) {
						*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tblNm)
					}
				}
			}
		}
	}
	return nil
}

func yangListDelData(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, subTreeResMap *map[string]map[string]db.Value, isFirstCall bool) error {
	var err, perr error
	var dbs [db.MaxDB]*db.DB
	var tblList []string
	xfmrLogDebug("yangListDelData Received xlateParams - %v \n dbDataMap - %v\n subTreeResMap - %v\n isFirstCall - %v", xlateParams, dbDataMap, subTreeResMap, isFirstCall)
	fillFields := false
	removedFillFields := false
	virtualTbl := false
	tblOwner := true
	keyName := xlateParams.keyName
	parentTbl := ""
	parentKey := ""

	spec, xpathOk := xYangSpecMap[xlateParams.xpath]
	if !xpathOk || spec.yangEntry == nil {
		xfmrLogDebug("xYangSpecmap is missing xpath or YANG Entry for - %v", xlateParams.xpath)
		return err
	}
	if spec.yangEntry.ReadOnly() {
		xfmrLogDebug("For Uri - %v skip delete processing since its a Read Only node", xlateParams.uri)
		return err
	}
	cdb := spec.dbIndex
	dbs[cdb] = xlateParams.d
	dbOpts := getDBOptions(cdb)
	separator := dbOpts.KeySeparator

	if !isFirstCall {
		xpathKeyExtRet, err := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache)
		if err != nil {
			xfmrLogDebug("Received error from xpathKeyExtract for URI : %v, error: %v", xlateParams.uri, err)
			switch e := err.(type) {
			case tlerr.TranslibXfmrRetError:
				ecode := e.XlateFailDelReq
				log.Warningf("Error received (\"%v\"), ecode :%v", err, ecode)
				if ecode {
					return err
				}
			}
		}
		if xpathKeyExtRet.isVirtualTbl {
			virtualTbl = true
		}
		keyName = xpathKeyExtRet.dbKey
		xlateParams.tableName = xpathKeyExtRet.tableName
		xlateParams.keyName = keyName
	}

	tblList, virtualTbl, err = tblKeyDataGet(xlateParams, dbDataMap, cdb)
	if err != nil {
		return err
	}

	xfmrLogDebug("tblList(%v), tbl(%v), key(%v)  for URI (\"%v\")", tblList, xlateParams.tableName, xlateParams.keyName, xlateParams.uri)
	for _, tbl := range tblList {
		curDbDataMap, ferr := fillDbDataMapForTbl(xlateParams.uri, xlateParams.xpath, tbl, keyName, cdb, dbs, xlateParams.dbTblKeyGetCache)
		if (ferr == nil) && len(curDbDataMap) > 0 {
			mapCopy((*dbDataMap)[cdb], curDbDataMap[cdb])
		}
	}

	// Not required to process parent and current table as subtree is already invoked before we get here
	// We only need to traverse nested subtrees here
	if isFirstCall && len(spec.xfmrFunc) == 0 {
		parentUri := parentUriGet(xlateParams.uri)
		xpathKeyExtRet, xerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, parentUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache)
		parentTbl = xpathKeyExtRet.tableName
		parentKey = xpathKeyExtRet.dbKey
		perr = xerr
		xfmrLogDebug("Parent Uri - %v, ParentTbl - %v, parentKey - %v", parentUri, parentTbl, parentKey)
	}

	if len(tblList) == 0 {
		xfmrLogInfo("Unable to traverse list as no table information available at URI %v. Please check if table mapping available", xlateParams.uri)
	}

	for _, tbl := range tblList {
		tblData, ok := (*dbDataMap)[cdb][tbl]
		xfmrLogDebug("Process Tbl - %v", tbl)
		var curTbl, curKey string
		var cerr error
		if ok {
			for dbKey := range tblData {
				xfmrLogDebug("Process Tbl - %v with dbKey - %v", tbl, dbKey)
				_, curUri, kerr := dbKeyToYangDataConvert(xlateParams.uri, xlateParams.requestUri, xlateParams.xpath, tbl, dbDataMap, dbKey, separator, xlateParams.txCache)
				if kerr != nil {
					continue
				}
				if spec.virtualTbl != nil && *spec.virtualTbl {
					virtualTbl = true
				}

				// Not required to check for table inheritence case here as we have a subtree and subtree is already processed before we get here
				// We only need to traverse nested subtrees here
				if len(spec.xfmrFunc) == 0 {

					xpathKeyExtRet, xerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache)
					curKey = xpathKeyExtRet.dbKey
					curTbl = xpathKeyExtRet.tableName
					cerr = xerr
					xfmrLogDebug("Current Uri - %v, CurrentTbl - %v, CurrentKey - %v", curUri, curTbl, curKey)

					if dbKey != curKey {
						xfmrLogDebug("Mismatch in dbKey and key derived from uri. dbKey : %v, key from uri: %v. Cannot traverse instance at URI %v. Please check the key mapping", dbKey, curKey, curUri)
						continue
					}
					if isFirstCall {
						if perr == nil && cerr == nil {
							if len(curTbl) > 0 && parentTbl != curTbl {
								/* Non-inhertited table case */
								xfmrLogDebug("Non-inhertaed table case, URI - %v", curUri)
								if spec.tblOwner != nil && !*spec.tblOwner {
									xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
									tblOwner = false
									/* Fill only fields */
									fillFields = true
								}
							} else if len(curTbl) > 0 {
								/* Inhertited table case */
								xfmrLogDebug("Inherited table case, URI - %v", curUri)
								if len(parentKey) > 0 {
									if parentKey == curKey { // List within list or List within container, where container map to entire table
										xfmrLogDebug("Parent key is same as current key")
										xfmrLogDebug("Request from NBI is at same level as that of current list - %v", curUri)
										if spec.tblOwner != nil && !*spec.tblOwner {
											xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
											tblOwner = false // since query is at this level, this will make sure to add instance to result
										}
										/* Fill only fields */
										fillFields = true
									} else { /*same table but different keys */
										xfmrLogDebug("Inherited table but parent key is NOT same as current key")
										if spec.tblOwner != nil && !*spec.tblOwner {
											xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
											tblOwner = false
											/* Fill only fields */
											fillFields = true
										}
									}
								} else {
									/*same table but no parent-key exists, parent must be a container wth just tableNm annot with no keyXfmr/Nm */
									xfmrLogDebug("Inherited table but no parent key available")
									if spec.tblOwner != nil && !*spec.tblOwner {
										xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
										tblOwner = false
										/* Fill only fields */
										fillFields = true

									}
								}
							} else { //  len(curTbl) = 0
								log.Warning("No table found for Uri - %v ", curUri)
							}
						}
					} else {
						/* if table instance already filled and there are no feilds present then it's instance level delete.
						If fields present then its fields delet and not instance delete
						*/
						if tblData, tblDataOk := xlateParams.result[curTbl]; tblDataOk {
							if fieldMap, fieldMapOk := tblData[curKey]; fieldMapOk {
								xfmrLogDebug("Found table instance same as that of requestUri")
								if len(fieldMap.Field) > 0 {
									/* Fill only fields */
									xfmrLogDebug("Found table instance same as that of requestUri with fields.")
									fillFields = true
								}
							}
						}
					} // end of if isFirstCall
				} // end if !subtree case
				xfmrLogDebug("For URI - %v , table-owner - %v, fillFields - %v", curUri, tblOwner, fillFields)
				if fillFields || spec.hasChildSubTree || isFirstCall {
					for yangChldName := range spec.yangEntry.Dir {
						chldXpath := xlateParams.xpath + "/" + yangChldName
						chldUri := curUri + "/" + yangChldName
						chldSpec, ok := xYangSpecMap[chldXpath]
						chldSpecYangEntry := spec.yangEntry.Dir[yangChldName]
						if ok && ((chldSpecYangEntry != nil) && (!chldSpecYangEntry.ReadOnly())) {
							chldYangType := chldSpec.yangType
							curXlateParams := xlateParams
							curXlateParams.uri = chldUri
							curXlateParams.xpath = chldXpath
							curXlateParams.tableName = ""
							curXlateParams.keyName = ""

							if (chldSpec.dbIndex == db.ConfigDB) && (len(chldSpec.xfmrFunc) > 0) {
								err = subTreeXfmrDelDataGet(curXlateParams, dbDataMap, cdb, spec, chldSpec, subTreeResMap)
								if err != nil {
									return err
								}
							}
							if chldYangType == YANG_CONTAINER {
								err = yangContainerDelData(curXlateParams, dbDataMap, subTreeResMap, false)
								if err != nil {
									return err
								}
							} else if chldYangType == YANG_LIST {
								err = yangListDelData(curXlateParams, dbDataMap, subTreeResMap, false)
								if err != nil {
									return err
								}
							} else if (chldSpec.dbIndex == db.ConfigDB) && ((chldYangType == YANG_LEAF) || (chldYangType == YANG_LEAF_LIST)) && !virtualTbl {
								if len(curTbl) == 0 {
									continue
								}
								if len(curKey) == 0 { //at list level key should always be there
									xfmrLogDebug("No key available for URI - %v", curUri)
									continue
								}
								if chldYangType == YANG_LEAF && chldSpec.isKey {
									if isFirstCall {
										if !tblOwner { //add dummy field to identify when to fill fields only at children traversal
											dataToDBMapAdd(curTbl, curKey, curXlateParams.result, "FillFields", "true")
										} else {
											dataToDBMapAdd(curTbl, curKey, curXlateParams.result, "", "")
										}
									}
								} else if fillFields {
									//strip off the leaf/leaf-list for mapFillDataUtil takes URI without it
									curXlateParams.uri = xlateParams.uri
									curXlateParams.name = chldSpecYangEntry.Name
									curXlateParams.tableName = curTbl
									curXlateParams.keyName = curKey
									err = mapFillDataUtil(curXlateParams)
									if err != nil {
										xfmrLogDebug("Error received (\"%v\")", err)
										switch e := err.(type) {
										case tlerr.TranslibXfmrRetError:
											ecode := e.XlateFailDelReq
											log.Warningf("Error received (\"%v\"), ecode :%v", err, ecode)
											if ecode {
												return err
											}
										}
									}
									if !removedFillFields {
										if fieldMap, ok := curXlateParams.result[curTbl][curKey]; ok {
											if len(fieldMap.Field) > 1 {
												delete(curXlateParams.result[curTbl][curKey].Field, "FillFields")
												removedFillFields = true
											} else if len(fieldMap.Field) == 1 {
												if _, ok := curXlateParams.result[curTbl][curKey].Field["FillFields"]; !ok {
													removedFillFields = true
												}
											}
										}
									}
								}
							} // end of Leaf case
						} // if rw
					} // end of curUri children traversal loop
				} // Child Subtree or fill fields
			} // end of for dbKey loop
		} // end of tbl in dbDataMap
	} // end of for tbl loop
	return err
}

func yangContainerDelData(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, subTreeResMap *map[string]map[string]db.Value, isFirstCall bool) error {
	var err error
	var dbs [db.MaxDB]*db.DB
	spec, ok := xYangSpecMap[xlateParams.xpath]
	cdb := spec.dbIndex
	dbs[cdb] = xlateParams.d
	removedFillFields := false
	var curTbl, curKey string

	if !ok {
		return err
	}
	if spec.yangEntry == nil {
		xfmrLogDebug("yang entry is nil for - %v", xlateParams.xpath)
		return err
	}

	if spec.yangEntry.ReadOnly() {
		return err
	}

	xfmrLogDebug("Traverse container for DELETE at URI (\"%v\")", xlateParams.uri)

	fillFields := false

	// Not required to process parent and current table as subtree is already invoked before we get here
	// We only need to traverse nested subtrees here
	if len(spec.xfmrFunc) == 0 {
		xpathKeyExtRet, cerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache)
		curKey = xpathKeyExtRet.dbKey
		curTbl = xpathKeyExtRet.tableName
		if cerr != nil {
			log.Warningf("Received xpathKeyExtract error for uri: %v : err %v", xlateParams.uri, cerr)
			switch e := err.(type) {
			case tlerr.TranslibXfmrRetError:
				ecode := e.XlateFailDelReq
				xfmrLogDebug("Error received (\"%v\"), ecode :%v", cerr, ecode)
				if ecode {
					return cerr
				}
			}
		}

		if isFirstCall {
			parentUri := parentUriGet(xlateParams.uri)
			parentXpathKeyExtRet, perr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, parentUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache)
			parentTbl := parentXpathKeyExtRet.tableName
			parentKey := parentXpathKeyExtRet.dbKey
			if perr == nil && cerr == nil && len(curTbl) > 0 {
				if len(curKey) > 0 {
					xfmrLogDebug("DELETE handling at Container parentTbl %v, curTbl %v, curKey %v", parentTbl, curTbl, curKey)
					if parentTbl != curTbl {
						// Non inhertited table
						if (spec.tblOwner != nil) && !(*spec.tblOwner) {
							// Fill fields only
							xfmrLogDebug("DELETE handling at Container Non inhertited table and not table Owner")
							dataToDBMapAdd(curTbl, curKey, xlateParams.result, "FillFields", "true")
							fillFields = true
						} else if (spec.keyName != nil && len(*spec.keyName) > 0) || len(spec.xfmrKey) > 0 {
							// Table owner && Key transformer present. Fill table instance
							xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner")
							dataToDBMapAdd(curTbl, curKey, xlateParams.result, "", "")
						} else {
							// Fallback case. Ideally should not enter here
							fillFields = true
						}
					} else {
						if curKey != parentKey {
							if (spec.tblOwner != nil) && !(*spec.tblOwner) {
								xfmrLogDebug("DELETE handling at Container inhertited table and not table Owner")
								dataToDBMapAdd(curTbl, curKey, xlateParams.result, "FillFields", "true")
								fillFields = true
							} else {
								// Instance delete
								xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner")
								dataToDBMapAdd(curTbl, curKey, xlateParams.result, "", "")
							}
						} else {
							// if Instance already filled do not fill fields
							xfmrLogDebug("DELETE handling at Container Inherited table")
							//Fill fields only
							if len(curTbl) > 0 && len(curKey) > 0 {
								dataToDBMapAdd(curTbl, curKey, xlateParams.result, "FillFields", "true")
								fillFields = true
							}
						}
					}
				} else {
					if (spec.tblOwner != nil) && !(*spec.tblOwner) {
						// Fill fields only
						xfmrLogDebug("DELETE handling at Container Non inhertited table and not table Owner No Key available. table: %v, key: %v", curTbl, curKey)
					} else {
						// Table owner && Key transformer present. Fill table instance
						xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner. No Key Delete complete TABLE : %v", curTbl)
						dataToDBMapAdd(curTbl, curKey, xlateParams.result, "", "")
					}
				}
			} else {
				xfmrLogDebug("perr: %v cerr: %v curTbl: %v, curKey: %v", perr, cerr, curTbl, curKey)
			}
		} else {
			// Inherited Table. We always expect the curTbl entry in xlateParams.result
			// if Instance already filled do not fill fields
			xfmrLogDebug("DELETE handling at Container Inherited table curTbl: %v, curKey %v", curTbl, curKey)
			if tblMap, ok := xlateParams.result[curTbl]; ok {
				if fieldMap, ok := tblMap[curKey]; ok {
					if len(fieldMap.Field) == 0 {
						xfmrLogDebug("Inhertited table & Instance delete case. Skip fields fill")
					} else {
						xfmrLogDebug("Inhertited table & fields fill for table :%v", curTbl)
						fillFields = true
					}
				}
			}
		}

	}
	xfmrLogDebug("URI %v fillFields %v, hasChildSubtree  %v, isFirstCall %v", xlateParams.uri, fillFields, spec.hasChildSubTree, isFirstCall)

	if fillFields || spec.hasChildSubTree || isFirstCall {
		for yangChldName := range spec.yangEntry.Dir {
			chldXpath := xlateParams.xpath + "/" + yangChldName
			chldUri := xlateParams.uri + "/" + yangChldName
			chldSpec, ok := xYangSpecMap[chldXpath]
			chldSpecYangEntry := spec.yangEntry.Dir[yangChldName]
			if ok && ((chldSpecYangEntry != nil) && (!chldSpecYangEntry.ReadOnly())) {
				chldYangType := chldSpec.yangType
				curXlateParams := xlateParams
				curXlateParams.uri = chldUri
				curXlateParams.xpath = chldXpath
				curXlateParams.tableName = curTbl
				curXlateParams.keyName = curKey

				if (chldSpec.dbIndex == db.ConfigDB) && (len(chldSpec.xfmrFunc) > 0) {
					err = subTreeXfmrDelDataGet(curXlateParams, dbDataMap, cdb, spec, chldSpec, subTreeResMap)
					if err != nil {
						return err
					}
				}
				if chldYangType == YANG_CONTAINER {
					err = yangContainerDelData(curXlateParams, dbDataMap, subTreeResMap, false)
					if err != nil {
						return err
					}
				} else if chldYangType == YANG_LIST {
					err = yangListDelData(curXlateParams, dbDataMap, subTreeResMap, false)
					if err != nil {
						return err
					}
				} else if (chldSpec.dbIndex == db.ConfigDB) && (chldYangType == YANG_LEAF || chldYangType == YANG_LEAF_LIST) && fillFields {
					//strip off the leaf/leaf-list for mapFillDataUtil takes URI without it
					curXlateParams.uri = xlateParams.uri
					curXlateParams.name = chldSpecYangEntry.Name
					err = mapFillDataUtil(curXlateParams)
					if err != nil {
						xfmrLogDebug("Error received during leaf fill (\"%v\")", err)
						switch e := err.(type) {
						case tlerr.TranslibXfmrRetError:
							ecode := e.XlateFailDelReq
							log.Warningf("Error received (\"%v\"), ecode :%v", err, ecode)
							if ecode {
								return err
							}
						}
					}
					if !removedFillFields {
						if fieldMap, ok := curXlateParams.result[curTbl][curKey]; ok {
							if len(fieldMap.Field) > 1 {
								delete(curXlateParams.result[curTbl][curKey].Field, "FillFields")
								removedFillFields = true
							} else if len(fieldMap.Field) == 1 {
								if _, ok := curXlateParams.result[curTbl][curKey].Field["FillFields"]; !ok {
									removedFillFields = true
								}
							}
						}
					}
				} else {
					xfmrLogDebug("%v", "Instance Fill case. Have filled the result table with table and key")
				}
			}
		}
	}
	return err
}

func allChildTblGetToDelete(xlateParams xlateToParams) (map[string]map[string]db.Value, error) {
	var err error
	subTreeResMap := make(map[string]map[string]db.Value)
	xpath, _, _ := XfmrRemoveXPATHPredicates(xlateParams.requestUri)
	spec, ok := xYangSpecMap[xpath]
	isFirstCall := true
	if !ok {
		errStr := "Xpath not found in spec-map:" + xpath
		return subTreeResMap, errors.New(errStr)
	}

	dbDataMap := make(RedisDbMap)
	for i := db.ApplDB; i < db.MaxDB; i++ {
		dbDataMap[i] = make(map[string]map[string]db.Value)
	}

	if len(spec.xfmrFunc) > 0 {
		// Subtree is already invoked before we get here
		// Not required to process parent and current tables
		isFirstCall = false
	}

	xfmrLogDebug("Request URI (\"%v\") to traverse for delete", xlateParams.requestUri)
	if ok && spec.yangEntry != nil {
		xlateParams.uri = xlateParams.requestUri
		xlateParams.xpath = xpath
		yangType := spec.yangType
		if yangType == YANG_LIST {
			err = yangListDelData(xlateParams, &dbDataMap, &subTreeResMap, isFirstCall)
			return subTreeResMap, err
		} else if yangType == YANG_CONTAINER {
			err = yangContainerDelData(xlateParams, &dbDataMap, &subTreeResMap, isFirstCall)
		}
	}
	return subTreeResMap, err
}

/* Get the db table, key and field name for the incoming delete request */
func dbMapDelete(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, requestUri string, jsonData interface{}, resultMap map[Operation]map[db.DBNum]map[string]map[string]db.Value, txCache interface{}, skipOrdTbl *bool) error {
	var err error
	var result = make(map[string]map[string]db.Value)
	subOpDataMap := make(map[Operation]*RedisDbMap)
	var xfmrErr error
	*skipOrdTbl = false
	var cascadeDelTbl []string

	/* Check if the parent table exists for RFC compliance */
	var exists bool
	var dbs [db.MaxDB]*db.DB
	subOpMapDiscard := make(map[Operation]*RedisDbMap)
	exists, err = verifyParentTable(d, dbs, ygRoot, oper, uri, nil, txCache, subOpMapDiscard)
	xfmrLogDebug("verifyParentTable() returned - exists - %v, err - %v", exists, err)
	if err != nil {
		if exists {
			// Special case when we delete at container that does exist. Not required to do translation. Do not return error either.
			return nil
		}
		log.Warningf("Cannot perform Operation %v on URI %v due to - %v", oper, uri, err)
		return err
	}
	if !exists {
		errStr := fmt.Sprintf("Parent table does not exist for uri(%v)", uri)
		return tlerr.InternalError{Format: errStr}
	}
	oper_crud_list := []Operation{CREATE, REPLACE, UPDATE, DELETE}
	for _, oper := range oper_crud_list {
		resultMap[oper] = make(map[db.DBNum]map[string]map[string]db.Value)
	}

	if isSonicYang(uri) {
		xpathPrefix, keyName, tableName := sonicXpathKeyExtract(uri)
		xfmrLogInfo("Delete req: uri(\"%v\"), key(\"%v\"), xpathPrefix(\"%v\"), tableName(\"%v\").", uri, keyName, xpathPrefix, tableName)
		resultMap[oper][db.ConfigDB] = result
		xlateToData := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpathPrefix, keyName, jsonData, resultMap, result, txCache, nil, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", tableName, nil)
		err = sonicYangReqToDbMapDelete(xlateToData)
		if err != nil {
			return err
		}
	} else {
		xpathKeyExtRet, err := xpathKeyExtract(d, ygRoot, oper, uri, requestUri, nil, subOpDataMap, txCache, nil)
		if err != nil {
			return err
		}
		xfmrLogInfo("Delete req: uri(\"%v\"), key(\"%v\"), xpath(\"%v\"), tableName(\"%v\").", uri, xpathKeyExtRet.dbKey, xpathKeyExtRet.xpath, xpathKeyExtRet.tableName)
		spec, ok := xYangSpecMap[xpathKeyExtRet.xpath]
		if ok {
			specYangType := spec.yangType
			moduleNm := "/" + strings.Split(uri, "/")[1]
			xfmrLogInfo("Module name for URI %s is %s", uri, moduleNm)
			if xYangModSpecMap != nil {
				if modSpecInfo, specOk := xYangModSpecMap[moduleNm]; specOk && (len(modSpecInfo.xfmrPre) > 0) {
					var dbs [db.MaxDB]*db.DB
					inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
					err = preXfmrHandlerFunc(modSpecInfo.xfmrPre, inParams)
					xfmrLogInfo("Invoked pre-transformer: %v, oper: %v, subOpDataMap: %v ",
						modSpecInfo.xfmrPre, oper, subOpDataMap)
					if err != nil {
						log.Warningf("Pre-transformer: %v failed.(err:%v)", modSpecInfo.xfmrPre, err)
						return err
					}
				}
			}

			if spec.cascadeDel == XFMR_ENABLE && xpathKeyExtRet.tableName != "" && xpathKeyExtRet.tableName != XFMR_NONE_STRING {
				if !contains(cascadeDelTbl, xpathKeyExtRet.tableName) {
					cascadeDelTbl = append(cascadeDelTbl, xpathKeyExtRet.tableName)
				}
			}
			curXlateParams := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpathKeyExtRet.xpath, xpathKeyExtRet.dbKey, jsonData, resultMap, result, txCache, nil, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", xpathKeyExtRet.tableName, nil)
			curXlateParams.xfmrDbTblKeyCache = make(map[string]tblKeyCache)
			if len(spec.xfmrFunc) > 0 {
				var dbs [db.MaxDB]*db.DB
				cdb := spec.dbIndex
				inParams := formXfmrInputRequest(d, dbs, cdb, ygRoot, uri, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
				stRetData, err := xfmrHandler(inParams, spec.xfmrFunc)
				if err == nil {
					mapCopy(result, stRetData)
				} else {
					return err
				}
				// TODO: Nested subtree invoke
				curResult, cerr := allChildTblGetToDelete(curXlateParams)
				if cerr != nil {
					return cerr
				} else {
					mapCopy(result, curResult)
				}

				if inParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
					for _, tblNm := range *inParams.pCascadeDelTbl {
						if !contains(cascadeDelTbl, tblNm) {
							cascadeDelTbl = append(cascadeDelTbl, tblNm)
						}
					}
				}
			} else if specYangType == YANG_LEAF || specYangType == YANG_LEAF_LIST {
				if len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) > 0 {
					dataToDBMapAdd(xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, result, "", "")
					xpath := xpathKeyExtRet.xpath
					pathList := strings.Split(xpath, "/")
					uriItemList := splitUri(strings.TrimSuffix(uri, "/"))
					uriItemListLen := len(uriItemList)
					var luri string
					if uriItemListLen > 0 {
						luri = strings.Join(uriItemList[:uriItemListLen-1], "/") //strip off the leaf/leaf-list for mapFillDataUtil takes URI without it

					}

					if specYangType == YANG_LEAF {
						_, ok := xYangSpecMap[xpath]
						if ok && len(xYangSpecMap[xpath].defVal) > 0 {
							// Do not fill def value if leaf does not map to any redis field
							dbSpecXpath := xpathKeyExtRet.tableName + "/" + xYangSpecMap[xpath].fieldName
							_, mapped := xDbSpecMap[dbSpecXpath]
							if mapped || len(xYangSpecMap[xpath].xfmrField) > 0 {
								curXlateParams.uri = luri
								curXlateParams.name = pathList[len(pathList)-1]
								curXlateParams.value = xYangSpecMap[xpath].defVal
								err = mapFillDataUtil(curXlateParams)
								if xfmrErr != nil {
									return xfmrErr
								}
								if err != nil {
									return err
								}
								if len(subOpDataMap) > 0 && subOpDataMap[UPDATE] != nil {
									subOperMap := subOpDataMap[UPDATE]
									mapCopy((*subOperMap)[db.ConfigDB], result)
								} else {
									var redisMap = new(RedisDbMap)
									var dbresult = make(RedisDbMap)
									for i := db.ApplDB; i < db.MaxDB; i++ {
										dbresult[i] = make(map[string]map[string]db.Value)
									}
									redisMap = &dbresult
									(*redisMap)[db.ConfigDB] = result
									subOpDataMap[UPDATE] = redisMap
								}
							}
							result = make(map[string]map[string]db.Value)
						} else {
							curXlateParams.uri = luri
							curXlateParams.name = pathList[len(pathList)-1]
							err = mapFillDataUtil(curXlateParams)
							if xfmrErr != nil {
								return xfmrErr
							}
							if err != nil {
								return err
							}
						}
					} else if specYangType == YANG_LEAF_LIST {
						var fieldVal []interface{}
						leafListInstVal, valErr := extractLeafListInstFromUri(uri)
						if valErr == nil && leafListInstVal != "" {
							fieldVal = append(fieldVal, leafListInstVal)
						}
						curXlateParams.uri = luri
						curXlateParams.name = pathList[len(pathList)-1]
						curXlateParams.value = fieldVal
						err = mapFillDataUtil(curXlateParams)

						if xfmrErr != nil {
							return xfmrErr
						}
						if err != nil {
							return err
						}
					}
				} else {
					log.Warningf("No proper table and key information to fill result map for URI %v, table: %v, key %v", uri, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
				}
			} else {
				xfmrLogDebug("Before calling allChildTblGetToDelete result: %v", curXlateParams.result)
				curResult, cerr := allChildTblGetToDelete(curXlateParams)
				if cerr != nil {
					err = cerr
					return err
				} else {
					mapCopy(result, curResult)
				}
				xfmrLogDebug("allChildTblGetToDelete result: %v  subtree curResult: %v", result, curResult)
				// Add the child tables to delete when table at request URI is not available or its complete table delete request (not specific instance)
				chResult := make(map[string]map[string]db.Value)
				if (len(xpathKeyExtRet.tableName) == 0 || (len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) == 0)) && len(spec.childTable) > 0 {
					for _, child := range spec.childTable {
						chResult[child] = make(map[string]db.Value)
					}
					xfmrLogDebug("Before adding children result: %v  result with child tables: %v", result, chResult)
				}
				mapCopy(result, chResult)
			}

			if xYangModSpecMap != nil {
				xfmrPost := ""
				if modSpec, ok := xYangModSpecMap[moduleNm]; ok {
					xfmrPost = modSpec.xfmrPost
				}
				if len(xfmrPost) > 0 {
					xfmrLogInfo("Invoke post transformer: %v", xfmrPost)
					var dbs [db.MaxDB]*db.DB
					var dbresult = make(RedisDbMap)
					dbresult[db.ConfigDB] = result
					inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, "", &dbresult, subOpDataMap, nil, txCache)
					result, err = postXfmrHandlerFunc(xfmrPost, inParams)
					if err != nil {
						return err
					}
					if inParams.skipOrdTblChk != nil {
						*skipOrdTbl = *(inParams.skipOrdTblChk)
						xfmrLogInfo("skipOrdTbl flag: %v", *skipOrdTbl)
					}
					if inParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
						for _, tblNm := range *inParams.pCascadeDelTbl {
							if !contains(cascadeDelTbl, tblNm) {
								cascadeDelTbl = append(cascadeDelTbl, tblNm)
							}
						}
					}
				}
			}

			if len(result) > 0 {
				resultMap[oper][db.ConfigDB] = result
			}

			if len(subOpDataMap) > 0 {
				for op, data := range subOpDataMap {
					if len(*data) > 0 {
						for dbType, dbData := range *data {
							if len(dbData) > 0 {
								if _, ok := resultMap[op][dbType]; !ok {
									resultMap[op][dbType] = make(map[string]map[string]db.Value)
								}
								mapCopy(resultMap[op][dbType], (*subOpDataMap[op])[dbType])
							}
						}
					}
				}

			}
			/* for container/list delete req , it should go through, even if there are any leaf default-yang-values */
		}
	} // End OC YANG handling

	err = dbDataXfmrHandler(resultMap)
	if err != nil {
		log.Warningf("Failed in dbdata-xfmr for %v", resultMap)
		return err
	}
	if len(cascadeDelTbl) > 0 {
		cdErr := handleCascadeDelete(d, resultMap, cascadeDelTbl)
		if cdErr != nil {
			xfmrLogInfo("Cascade Delete Failed for cascadeDelTbl (%v), Error: (%v)", cascadeDelTbl, cdErr)
			return cdErr
		}
	}

	printDbData(resultMap, nil, "/tmp/yangToDbDataDel.txt")
	xfmrLogInfo("Delete req: uri(\"%v\") resultMap(\"%v\").", uri, resultMap)
	return err
}

func sonicYangReqToDbMapDelete(xlateParams xlateToParams) error {
	var err error
	if xlateParams.tableName != "" {
		// Specific table entry case
		xlateParams.result[xlateParams.tableName] = make(map[string]db.Value)
		isFieldReq := false
		if xlateParams.keyName != "" {
			// Specific key case
			var dbVal db.Value
			tokens := strings.Split(xlateParams.xpath, "/")
			if tokens[SONIC_TABLE_INDEX] == xlateParams.tableName {
				fieldName := ""
				if len(tokens) > SONIC_FIELD_INDEX {
					fieldName = tokens[SONIC_FIELD_INDEX]
				}

				if fieldName != "" {
					isFieldReq = true
					dbSpecField := xlateParams.tableName + "/" + fieldName
					dbEntry := getYangEntryForXPath(dbSpecField)
					_, ok := xDbSpecMap[dbSpecField]
					if ok && dbEntry != nil {
						yangType := xDbSpecMap[dbSpecField].yangType
						// terminal node case
						if yangType == YANG_LEAF_LIST {
							dbVal.Field = make(map[string]string)
							dbFldVal := ""
							//check if it is a specific item in leaf-list delete
							leafListInstVal, valErr := extractLeafListInstFromUri(xlateParams.requestUri)
							if valErr == nil {
								dbFldVal, err = unmarshalJsonToDbData(dbEntry, dbSpecField, fieldName, leafListInstVal)
								if err != nil {
									log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", dbSpecField, err)
									return err
								}
							}
							fieldName = fieldName + "@"
							dbVal.Field[fieldName] = dbFldVal
						}
						if yangType == YANG_LEAF {
							dbVal.Field = make(map[string]string)
							dbVal.Field[fieldName] = ""
						}
					}
				}
			}
			xlateParams.result[xlateParams.tableName][xlateParams.keyName] = dbVal
		}
		if !isFieldReq {
			if tblSpecInfo, ok := xDbSpecMap[xlateParams.tableName]; ok && (tblSpecInfo.cascadeDel == XFMR_ENABLE) {
				*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, xlateParams.tableName)
			}
		}
	} else {
		// Get all table entries
		// If table name not available in xpath get top container name
		_, ok := xDbSpecMap[xlateParams.xpath]
		if ok && xDbSpecMap[xlateParams.xpath] != nil {
			dbInfo := xDbSpecMap[xlateParams.xpath]
			if dbInfo.yangType == YANG_CONTAINER && dbInfo.dbEntry != nil {
				for dir := range dbInfo.dbEntry.Dir {
					if tblSpecInfo, ok := xDbSpecMap[dir]; ok && tblSpecInfo.cascadeDel == XFMR_ENABLE {
						*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, dir)
					}
					if dbInfo.dbEntry.Dir[dir].Config != yang.TSFalse {
						xlateParams.result[dir] = make(map[string]db.Value)
					}
				}
			}
		}
	}
	return nil
}
