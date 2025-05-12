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
	if xYangSpecMap[xlateParams.xpath].xfmrTbl != nil {
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
	return tblList, isVirtualTbl, err
}

func subTreeXfmrDelDataGet(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, cdb db.DBNum, spec *yangXpathInfo, chldSpec *yangXpathInfo, subTreeResMap *map[string]map[string]db.Value) (bool, error) {
	var dbs [db.MaxDB]*db.DB
	dbs[cdb] = xlateParams.d
	validate := true

	xfmrLogDebug("Handle subtree for  (\"%v\")", xlateParams.uri)

	// If Delete traversal is being done for REPLACE check if the subtree is already invoked during REPLACE flow
	if xlateParams.replaceInfo != nil && xlateParams.replaceInfo.isDeleteForReplace {
		// The xlateParams.uri is always at the container or whole list level. The entries in subtreeVisitedCache is also made for containers or whole list uris. Hence no instance check will be required here.
		_, entryExists := xlateParams.replaceInfo.subtreeVisitedCache[xlateParams.uri]
		if entryExists {
			// return validate as true to allow yang tree traversal for child subtrees
			return validate, nil
		}
	}

	// Evaluate validate xfmr if available for subtree
	if (len(chldSpec.validateFunc) > 0) && (chldSpec.validateFunc != spec.validateFunc) {
		xfmrLogDebug("Invoke validate Xfmr function %v for uri %v", spec.validateFunc, xlateParams.uri)
		xpathKeyExtRet, err := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
		if err == nil {
			inParams := formXfmrInputRequest(xlateParams.d, dbs, db.ConfigDB, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, xpathKeyExtRet.dbKey, dbDataMap, xlateParams.subOpDataMap, nil, xlateParams.txCache)
			res := validateHandlerFunc(inParams, chldSpec.validateFunc)
			if !res {
				// Return the validation status to caller to indicates further traversal not required
				validate = res
				return validate, err
			}
		}
	}

	inParams := formXfmrInputRequest(xlateParams.d, dbs, cdb, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, "",
		dbDataMap, xlateParams.subOpDataMap, nil, xlateParams.txCache)
	retMap, err := xfmrHandler(inParams, chldSpec.xfmrFunc)
	if err != nil {
		xfmrLogDebug("Error returned by %v: %v", chldSpec.xfmrFunc, err)
		return validate, err
	}
	mapCopy(*subTreeResMap, retMap)
	if xlateParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
		for _, tblNm := range *inParams.pCascadeDelTbl {
			if !contains(*xlateParams.pCascadeDelTbl, tblNm) {
				*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tblNm)
			}
		}
	}
	return validate, nil
}

func yangListDelData(xlateParams xlateToParams, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, subTreeResMap *map[string]map[string]db.Value, isFirstCall bool) error {
	var err, perr error
	var dbs [db.MaxDB]*db.DB
	var tblList []string
	xfmrLogDebug("yangListDelData Received xlateParams - %v \n dbDataMap - %v\n subTreeResMap - %v\n isFirstCall - %v", xlateParams, dbDataMap, subTreeResMap, isFirstCall)
	fillFields := false
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
		var dbs [db.MaxDB]*db.DB
		xpathKeyExtRet, err := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
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

	if xYangSpecMap[xlateParams.xpath].xfmrTbl != nil {
		tblList, virtualTbl, err = tblKeyDataGet(xlateParams, dbDataMap, cdb)
		if err != nil {
			return err
		}
	} else if len(xlateParams.tableName) > 0 {
		tblList = append(tblList, xlateParams.tableName)
	}

	if len(tblList) == 0 {
		xfmrLogInfo("Unable to traverse list as no table information available at URI %v. Please check if table mapping available", xlateParams.uri)
		return err
	}

	xfmrLogDebug("tblList(%v), tbl(%v), key(%v)  for URI (\"%v\")", tblList, xlateParams.tableName, xlateParams.keyName, xlateParams.uri)
	for _, tbl := range tblList {
		curDbDataMap, ferr := fillDbDataMapForTbl(xlateParams.uri, xlateParams.xpath, tbl, keyName, cdb, dbs, xlateParams.dbTblKeyGetCache, nil)
		if (ferr == nil) && len(curDbDataMap) > 0 {
			mapCopy((*dbDataMap)[cdb], curDbDataMap[cdb])
		}
	}

	// Not required to process parent and current table as subtree is already invoked before we get here
	// We only need to traverse nested subtrees here
	if isFirstCall && len(spec.xfmrFunc) == 0 {
		parentUri := parentUriGet(xlateParams.uri)
		var dbs [db.MaxDB]*db.DB
		xpathKeyExtRet, xerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, parentUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
		parentTbl = xpathKeyExtRet.tableName
		parentKey = xpathKeyExtRet.dbKey
		perr = xerr
		xfmrLogDebug("Parent Uri - %v, ParentTbl - %v, parentKey - %v", parentUri, parentTbl, parentKey)
	}

	for _, tbl := range tblList {
		tblData, ok := (*dbDataMap)[cdb][tbl]
		xfmrLogDebug("Process Tbl - %v", tbl)
		removedFillFields := false
		var curTbl, curKey string
		var cerr error
		if ok {
			for dbKey := range tblData {
				xfmrLogDebug("Process Tbl - %v with dbKey - %v", tbl, dbKey)
				rmap, curUri, _, kerr := dbKeyToYangDataConvert(xlateParams.uri, xlateParams.requestUri, xlateParams.xpath, tbl, xlateParams.d, dbs, dbDataMap, dbKey, separator, xlateParams.txCache, xlateParams.oper)
				if kerr != nil || len(rmap) == 0 {
					continue
				}
				if spec.virtualTbl != nil {
					virtualTbl = *spec.virtualTbl
				}

				var dbs [db.MaxDB]*db.DB
				xpathKeyExtRet, xerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
				curKey = xpathKeyExtRet.dbKey
				curTbl = xpathKeyExtRet.tableName
				if dbKey != curKey {
					xfmrLogDebug("Mismatch in dbKey and key derived from uri. dbKey : %v, key from uri: %v. Cannot traverse instance at URI %v. Please check the key mapping", dbKey, curKey, curUri)
					continue
				}

				// Not required to check for table inheritence case here as we have a subtree and subtree is already processed before we get here
				// We only need to traverse nested subtrees here
				if len(spec.xfmrFunc) == 0 {
					if spec.virtualTbl == nil && xpathKeyExtRet.isVirtualTbl {
						virtualTbl = xpathKeyExtRet.isVirtualTbl
					}
					if spec.tblOwner != nil {
						tblOwner = *spec.tblOwner
					} else {
						tblOwner = !xpathKeyExtRet.isNotTblOwner
					}

					cerr = xerr
					xfmrLogDebug("Current Uri - %v, CurrentTbl - %v, CurrentKey - %v", curUri, curTbl, curKey)
					// Invoke the validate Xfmr function to evaluate if further processing is required
					parentSpec, parentOk := xYangSpecMap[xlateParams.parentXpath]

					if len(spec.validateFunc) > 0 && (isFirstCall ||
						(parentOk && (spec.validateFunc != parentSpec.validateFunc))) {
						xfmrLogDebug("Invoke validate Xfmr function %v for uri %v", spec.validateFunc, curUri)
						inParams := formXfmrInputRequest(xlateParams.d, dbs, db.ConfigDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, curKey, dbDataMap, xlateParams.subOpDataMap, nil, xlateParams.txCache)
						res := validateHandlerFunc(inParams, spec.validateFunc)
						if !res {
							continue
						}
					}

					if isFirstCall {
						if perr == nil && cerr == nil {
							if len(curTbl) > 0 && parentTbl != curTbl {
								/* Non-inhertited table case */
								xfmrLogDebug("Non-inhertaed table case, URI - %v", curUri)
								if !tblOwner {
									xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
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
										if !tblOwner {
											xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, tblOwner)
										}
										/* Fill only fields */
										fillFields = true
									} else { /*same table but different keys */
										xfmrLogDebug("Inherited table but parent key is NOT same as current key")
										if !tblOwner {
											xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
											/* Fill only fields */
											fillFields = true
										}
									}
								} else {
									/*same table but no parent-key exists, parent must be a container wth just tableNm annot with no keyXfmr/Nm */
									xfmrLogDebug("Inherited table but no parent key available")
									if !tblOwner {
										xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, *spec.tblOwner)
										/* Fill only fields */
										fillFields = true

									}
								}
							} else { //  len(curTbl) = 0
								log.Warning("No table found for Uri -  ", curUri)
							}
						}
					} else {
						/* if table instance already filled and there are no feilds present then it's instance level delete.
						If fields present then its fields delet and not instance delete
						*/
						if !tblOwner {
							xfmrLogDebug("For URI - %v, table owner - %v", xlateParams.uri, tblOwner)
							/* Fill only fields */
							fillFields = true
						} else {
							if tblData, tblDataOk := xlateParams.result[curTbl]; tblDataOk {
								if fieldMap, fieldMapOk := tblData[curKey]; fieldMapOk {
									xfmrLogDebug("Found table instance same as that of requestUri")
									if len(fieldMap.Field) > 0 {
										//* Fill only fields
										xfmrLogDebug("Found table instance same as that of requestUri with fields.")
										fillFields = true
									}
								}
							}
						}

					} // end of if isFirstCall
				} else { // end if !subtree case
					if !spec.hasChildSubTree {
						return err
					}
				}

				skipSibling := false
				isDeleteForReplace := false
				if xlateParams.replaceInfo != nil {
					isDeleteForReplace = xlateParams.replaceInfo.isDeleteForReplace
				}
				// For deleteForReplace case and for nodes having terminal nodes only and no complex nodes we can skip the delete processing if the uri is available in xlateParams.tblXpathMap for the curTbl and curKey identified. This is encountered if the REPLACE payload is avaiable for the current xpath (instance as we also condider the curKey) being processed.
				if isDeleteForReplace && !spec.hasNonTerminalNode {
					if tblUriMapVal, ok := xlateParams.tblXpathMap[curTbl][curKey]; ok {
						if _, found := tblUriMapVal[curUri]; found {
							continue
						}
					}
				}

				xfmrLogDebug("For URI - %v , table-owner - %v, fillFields - %v", curUri, tblOwner, fillFields)
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
						curXlateParams.parentXpath = xlateParams.xpath
						if curXlateParams.replaceInfo != nil {
							if curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete == nil {
								curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete = new(bool)
							} else {
								*curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete = false
							}
						}

						if (chldSpec.dbIndex == db.ConfigDB) && (len(chldSpec.xfmrFunc) > 0) {
							if (len(spec.xfmrFunc) == 0) ||
								((len(spec.xfmrFunc) > 0) && (spec.xfmrFunc != chldSpec.xfmrFunc)) {
								validate, serr := subTreeXfmrDelDataGet(curXlateParams, dbDataMap, cdb, spec, chldSpec, subTreeResMap)
								if serr != nil {
									// Subtree returned error
									return serr
								} else if !validate {
									// No error from subtree but validates to false
									continue
								}
							}
							if !chldSpec.hasChildSubTree {
								continue
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
						} else if (chldSpec.dbIndex == db.ConfigDB) && ((chldYangType == YANG_LEAF) || (chldYangType == YANG_LEAF_LIST)) && !virtualTbl && (len(chldSpec.xfmrFunc) == 0) {
							/* Do not fill table in resultMap for virtual table and subtree cases */
							if len(curTbl) == 0 {
								continue
							}
							if len(curKey) == 0 { //at list level key should always be there
								xfmrLogDebug("No key avaialble for URI - %v", curUri)
								continue
							}
							if chldYangType == YANG_LEAF && chldSpec.isKey {
								if _, tblDataOk := xlateParams.result[curTbl][curKey]; !tblDataOk {
									if !tblOwner { //add dummy field to identify when to fill fields only at children traversal
										addInstanceToDeleteMap(curTbl, curKey, curXlateParams.result, "FillFields", "true", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
									} else {
										addInstanceToDeleteMap(curTbl, curKey, curXlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
									}
								}
							} else if fillFields && !skipSibling {
								//strip off the leaf/leaf-list for mapFillDataUtil takes URI without it
								curXlateParams.uri = xlateParams.uri
								curXlateParams.name = chldSpecYangEntry.Name
								curXlateParams.tableName = curTbl
								curXlateParams.keyName = curKey
								if chldYangType == YANG_LEAF_LIST {
									curXlateParams.value = []interface{}{}
								}

								err = mapFillDataUtil(curXlateParams, false)
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
								if curXlateParams.replaceInfo != nil && curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete != nil {
									skipSibling = *curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete
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
	virtualTbl := false
	tblOwner := true

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
		var dbs [db.MaxDB]*db.DB
		xpathKeyExtRet, cerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
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
		if spec.virtualTbl != nil {
			virtualTbl = *spec.virtualTbl
		} else if xpathKeyExtRet.isVirtualTbl {
			virtualTbl = xpathKeyExtRet.isVirtualTbl
		}

		if spec.tblOwner != nil {
			tblOwner = *spec.tblOwner
		} else {
			tblOwner = !xpathKeyExtRet.isNotTblOwner
		}

		isDeleteForReplace := false
		if xlateParams.replaceInfo != nil {
			isDeleteForReplace = xlateParams.replaceInfo.isDeleteForReplace
		}
		// For deleteForReplace case and for containers having terminal nodes only and no complex nodes we can skip the delete processing if the uri is available in xlateParams.tblXpathMap for the curTbl and curKey identified. This is encountered if the REPLACE payload is avaiable for the current xpath being processed. For non table owners the auxValMap will be used to identify fields to be deleted.
		if isDeleteForReplace && !spec.hasNonTerminalNode {
			if tblUriMapVal, ok := xlateParams.tblXpathMap[curTbl][curKey]; ok {
				if _, found := tblUriMapVal[xlateParams.uri]; found {
					return err
				}
			}
		}

		if isFirstCall && !virtualTbl {
			parentUri := parentUriGet(xlateParams.uri)
			var dbs [db.MaxDB]*db.DB
			parentXpathKeyExtRet, perr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, parentUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
			parentTbl := parentXpathKeyExtRet.tableName
			parentKey := parentXpathKeyExtRet.dbKey
			if perr == nil && cerr == nil && len(curTbl) > 0 {
				if len(curKey) > 0 {
					xfmrLogDebug("DELETE handling at Container parentTbl %v, curTbl %v, curKey %v", parentTbl, curTbl, curKey)
					if parentTbl != curTbl {
						// Non inhertited table
						if !tblOwner {
							// Fill fields only
							xfmrLogDebug("DELETE handling at Container Non inhertited table and not table Owner")
							if addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "FillFields", "true", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl) {
								fillFields = true
							}
						} else {
							// Table owner and valid key present (either through inheritence or annotation). Hence mark the instance for delete
							xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner")
							addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
						}
					} else {
						if curKey != parentKey {
							if !tblOwner {
								xfmrLogDebug("DELETE handling at Container inhertited table and not table Owner")
								if addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "FillFields", "true", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl) {
									fillFields = true
								}
							} else {
								// Instance delete
								xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner")
								addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
							}
						} else {
							xfmrLogDebug("DELETE handling at Container Inherited table")
							//Fill fields only
							if addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "FillFields", "true", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl) {
								fillFields = true
							}
						}
					}
				} else {
					if !tblOwner {
						// Fill fields only
						xfmrLogDebug("DELETE handling at Container Non inhertited table and not table Owner No Key available. table: %v, key: %v", curTbl, curKey)
					} else {
						// Table owner && Key transformer present. Fill table instance
						xfmrLogDebug("DELETE handling at Container Non inhertited table & table Owner. No Key Delete complete TABLE : %v", curTbl)
						addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
					}
				}
			} else {
				// Unable to determine table key entry mapped at this level. traverse to children.
				xfmrLogDebug("perr: %v cerr: %v curTbl: %v, curKey: %v", perr, cerr, curTbl, curKey)
			}
		} else if !isFirstCall {
			// Invoke the validate transformer if available. Not required for firstCall as its called at entry point
			parentSpec, parentOk := xYangSpecMap[xlateParams.parentXpath]
			if (len(spec.validateFunc) > 0) && (parentOk && (spec.validateFunc != parentSpec.validateFunc)) {
				xfmrLogDebug("Invoke validate Xfmr function %v fo uri %v", spec.validateFunc, xlateParams.uri)
				inParams := formXfmrInputRequest(xlateParams.d, dbs, db.ConfigDB, xlateParams.ygRoot, xlateParams.uri, xlateParams.requestUri, xlateParams.oper, curKey, nil, xlateParams.subOpDataMap, nil, xlateParams.txCache)
				res := validateHandlerFunc(inParams, spec.validateFunc)
				if !res {
					return err
				}
			}
			if !virtualTbl {
				xfmrLogDebug("DELETE handling at Container Inherited table curTbl: %v, curKey %v", curTbl, curKey)
				// We always expect table mapped to container to have a valid key.
				if !tblOwner {
					xfmrLogDebug("Non table owner child node handling. Fields fill for table :%v", curTbl)
					if len(curTbl) > 0 && len(curKey) > 0 {
						if fldMap, ok := xlateParams.result[curTbl][curKey]; !ok {
							if addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "FillFields", "true", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl) {
								fillFields = true
							}
						} else {
							if len(fldMap.Field) > 0 {
								fillFields = true
							}
						}
					}
				} else if len(curTbl) > 0 {
					xfmrLogDebug("DELETE handling at child container mapped table curTbl: %v, curKey %v", curTbl, curKey)
					// Mark the instance for delete if its a table owner. For derived tables from parent, entry should already exist during parent traversal.
					if len(curKey) > 0 {
						if fieldMap, ok := xlateParams.result[curTbl][curKey]; ok {
							// Identify if DB instance existence check is required. If exists add to resultMap else ignore and continue traversal.
							if len(fieldMap.Field) > 0 {
								fillFields = true
							}
						} else {
							addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
						}
					} else {
						// We expect a valid key always. If key not present it will be considered as complete table delete
						// TODO: Identify if DB instance existence check is required. If exists add to resultMap else ignore and continue traversal.
						addInstanceToDeleteMap(curTbl, curKey, xlateParams.result, "", "", isDeleteForReplace, xlateParams.resultMap, xlateParams.pCascadeDelTbl)
					}
				}
			}
		}
	} else {
		if !spec.hasChildSubTree {
			return err
		}
	}

	xfmrLogDebug("URI %v fillFields %v, hasChildSubtree  %v, isFirstCall %v", xlateParams.uri, fillFields, spec.hasChildSubTree, isFirstCall)

	skipSibling := false
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
			curXlateParams.parentXpath = xlateParams.xpath
			if curXlateParams.replaceInfo != nil {
				if curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete == nil {
					curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete = new(bool)
				} else {
					*curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete = false
				}
			}

			if (chldSpec.dbIndex == db.ConfigDB) && (len(chldSpec.xfmrFunc) > 0) {
				if (len(spec.xfmrFunc) == 0) || ((len(spec.xfmrFunc) > 0) &&
					(spec.xfmrFunc != chldSpec.xfmrFunc)) {

					validate, serr := subTreeXfmrDelDataGet(curXlateParams, dbDataMap, cdb, spec, chldSpec, subTreeResMap)
					if serr != nil {
						// Return the subtree error
						return serr
					} else if !validate {
						// No error from subtree but validates to false
						return nil
					}
				}
				if !chldSpec.hasChildSubTree {
					continue
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
			} else if (chldSpec.dbIndex == db.ConfigDB) && (chldYangType == YANG_LEAF || chldYangType == YANG_LEAF_LIST) && fillFields && !skipSibling {
				//strip off the leaf/leaf-list for mapFillDataUtil takes URI without it
				curXlateParams.uri = xlateParams.uri
				curXlateParams.name = chldSpecYangEntry.Name
				if chldYangType == YANG_LEAF_LIST {
					curXlateParams.value = []interface{}{}
				}
				err = mapFillDataUtil(curXlateParams, false)
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
				if curXlateParams.replaceInfo != nil && curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete != nil {
					skipSibling = *curXlateParams.replaceInfo.skipFieldSiblingTraversalForDelete
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
		xlateParams.parentXpath = parentXpathGet(xpath)
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
	var dbs [db.MaxDB]*db.DB
	subOpMapDiscard := make(map[Operation]*RedisDbMap)
	exists, parChkErr := verifyParentTable(d, dbs, ygRoot, oper, uri, nil, txCache, subOpMapDiscard, nil)
	xfmrLogDebug("verifyParentTable() returned - exists - %v, err - %v", exists, parChkErr)
	if parChkErr != nil && !exists {
		log.Warningf("Cannot perform Operation %v on URI %v due to - %v", oper, uri, parChkErr)
		return parChkErr
	}
	// Special case: If Resource not found in DB when we delete at container node(requestUri - parChkErr != nil && exists), continue to traverse to get child tables. Also identify a leaf case having default value when table does not exist to skip default value setting for non existent entry.
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
		xlateToData := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpathPrefix, keyName, jsonData, resultMap, result, txCache, nil, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", tableName, false, nil, nil, nil, nil)
		err = sonicYangReqToDbMapDelete(xlateToData)
		if err != nil {
			return err
		}
	} else {
		var dbs [db.MaxDB]*db.DB
		xpathKeyExtRet, err := xpathKeyExtract(d, ygRoot, oper, uri, requestUri, nil, subOpDataMap, txCache, nil, dbs)
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

			if len(xYangSpecMap[xpathKeyExtRet.xpath].validateFunc) > 0 && specYangType != YANG_LIST {
				// For list cases evaluate for every instance to make sure the validation is done for the current instance
				inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, xpathKeyExtRet.dbKey, nil, subOpDataMap, nil, txCache)
				res := validateHandlerFunc(inParams, xYangSpecMap[xpathKeyExtRet.xpath].validateFunc)
				if !res {
					// Validate xfmr returns not valid hence, no further traversal required. return here
					return nil
				}
			}

			curXlateParams := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpathKeyExtRet.xpath, xpathKeyExtRet.dbKey, jsonData, resultMap, result, txCache, nil, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", xpathKeyExtRet.tableName, false, nil, nil, nil, nil)
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
				// Traverse the tree only if the yang tree has a nested subtree
				if spec.hasChildSubTree {
					curResult, cerr := allChildTblGetToDelete(curXlateParams)
					if cerr != nil {
						return cerr
					} else {
						mapCopy(result, curResult)
					}
				}
				if inParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
					for _, tblNm := range *inParams.pCascadeDelTbl {
						if !contains(cascadeDelTbl, tblNm) {
							cascadeDelTbl = append(cascadeDelTbl, tblNm)
						}
					}
				}
			} else if specYangType == YANG_LEAF || specYangType == YANG_LEAF_LIST {
				xpath := xpathKeyExtRet.xpath
				_, ok := xYangSpecMap[xpath]
				if parChkErr != nil && exists && specYangType == YANG_LEAF && ok && len(xYangSpecMap[xpath].defVal) > 0 {
					xfmrLogDebug("Do not reset to default for DELETE on leaf having default but maps to a table annotated at container that does not exist.")
				} else if len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) > 0 {
					dataToDBMapAdd(xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, result, "", "")
					pathList := strings.Split(xpath, "/")
					uriItemList := splitUri(strings.TrimSuffix(uri, "/"))
					uriItemListLen := len(uriItemList)
					var luri string
					if uriItemListLen > 0 {
						luri = "/" + strings.Join(uriItemList[:uriItemListLen-1], "/") //strip off the leaf/leaf-list for mapFillDataUtil takes URI without it

					}

					if specYangType == YANG_LEAF {
						if ok && len(xYangSpecMap[xpath].defVal) > 0 {
							// Do not fill def value if leaf does not map to any redis field
							dbSpecXpath := xpathKeyExtRet.tableName + "/" + xYangSpecMap[xpath].fieldName
							_, mapped := xDbSpecMap[dbSpecXpath]
							if mapped || len(xYangSpecMap[xpath].xfmrField) > 0 {
								curXlateParams.uri = luri
								curXlateParams.name = pathList[len(pathList)-1]
								curXlateParams.value = xYangSpecMap[xpath].defVal
								err = mapFillDataUtil(curXlateParams, false)
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
							err = mapFillDataUtil(curXlateParams, false)
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
						err = mapFillDataUtil(curXlateParams, false)

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
					err = postXfmrHandlerFunc(xfmrPost, inParams)
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
	oper := xlateParams.oper
	if xlateParams.tableName != "" {
		// Specific table entry case
		xlateParams.result[xlateParams.tableName] = make(map[string]db.Value)
		tokens := strings.Split(xlateParams.xpath, "/")
		isFieldReq := false
		var dbVal db.Value
		if xlateParams.keyName != "" {
			// Specific key case
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
							dbFldVal := ""
							if len(dbEntry.Default) > 0 {
								dbFldVal, err = unmarshalJsonToDbData(dbEntry, dbSpecField, fieldName, dbEntry.Default)
								if err != nil {
									log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", dbSpecField, err)
									return err
								}
								oper = UPDATE
							}
							dbVal.Field[fieldName] = dbFldVal
						}
					} else if !ok { //check if its nested list DELETE and process it
						return checkAndProcessSonicYangNesetedListDelete(xlateParams, tokens[SONIC_TBL_CHILD_INDEX], fieldName)
					}
				}
			}
			xlateParams.result[xlateParams.tableName][xlateParams.keyName] = dbVal
		} else {
			/* handle delete request at whole list level which is a sibling list to singleton container.
			   For such delete request, we need to get make sure we delete only list keys and not the keys that
			   correspond to singleton container(s).
			*/
			if len(tokens) > SONIC_TBL_CHILD_INDEX {
				tblChldNm := tokens[SONIC_TBL_CHILD_INDEX]
				xfmrLogDebug("Table Child Name : %v", tblChldNm)
				tblChldXpath := xlateParams.tableName + "/" + tblChldNm
				if specTblChldInfo, ok := xDbSpecMap[tblChldXpath]; ok && specTblChldInfo != nil {
					if specTblChldInfo.yangType == YANG_LIST && (!strings.HasSuffix(xlateParams.requestUri, "]") || !strings.HasSuffix(xlateParams.requestUri, "]/")) {
						if tblSpecInfo, ok := xDbSpecMap[xlateParams.tableName]; ok && tblSpecInfo != nil {
							if len(tblSpecInfo.dbEntry.Dir) > len(tblSpecInfo.listName) { // table level container has singleton container and list as siblings.
								singletonContainers := make(map[string]bool)
								for childName, child := range tblSpecInfo.dbEntry.Dir {
									if child.IsContainer() {
										singletonContainers[childName] = true
									}
								}
								if len(singletonContainers) > 0 {
									dbTblSpec := &db.TableSpec{Name: xlateParams.tableName}
									allKeys, err := xlateParams.d.GetKeys(dbTblSpec)
									if err != nil {
										xfmrLogInfo("Failed to get keys for table (%v), Error: (%v)", xlateParams.tableName, err)
										delete(xlateParams.result, xlateParams.tableName)
									}
									separator := xlateParams.d.Opts.KeySeparator
									for _, key := range allKeys {
										keyStr := strings.Join(key.Comp, separator)
										if _, ok := singletonContainers[keyStr]; ok {
											xfmrLogDebug("Skipping %v since it matches singleton container", keyStr)
											continue
										}
										xlateParams.result[xlateParams.tableName][keyStr] = dbVal
									}
								}
							}
						}
					}
				}

			}
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
	xlateParams.resultMap[oper][db.ConfigDB] = xlateParams.result
	return nil
}

func addInstanceToDeleteMap(tableName string, dbKey string, result map[string]map[string]db.Value, field string, value string, isDeleteForReplace bool, replaceResultMap map[Operation]RedisDbMap, pCascadeDelTbl *[]string) bool {
	if len(tableName) == 0 || len(dbKey) == 0 {
		return false
	}
	addToMap := true
	if isDeleteForReplace {
		addToMap, _ = addToDeleteForReplaceMap(tableName, dbKey, field, replaceResultMap)
	}
	if !isDeleteForReplace || (isDeleteForReplace && addToMap) {
		dataToDBMapAdd(tableName, dbKey, result, field, value)
	}
	/* Add table to cascade delete list if annotation is available only for complete instance Delete case. Delete for replace is handled after merging the result and subOp maps */
	if len(field) == 0 && !isDeleteForReplace {
		if tblSpecInfo, ok := xDbSpecMap[tableName]; ok && tblSpecInfo.cascadeDel == XFMR_ENABLE {
			if pCascadeDelTbl != nil && !contains(*pCascadeDelTbl, tableName) {
				*pCascadeDelTbl = append(*pCascadeDelTbl, tableName)
			}
		}
	}
	return addToMap
}

func checkAndProcessSonicYangNesetedListDelete(xlateParams xlateToParams, parentListNm string, nestedChildNm string) error {
	var fieldNm, fieldVal string

	oper := xlateParams.oper //DELETE
	dbSpecPath := xlateParams.tableName + "/" + parentListNm + "/" + nestedChildNm
	dbSpecNestedChildInfo, ok := xDbSpecMap[dbSpecPath]

	if !ok || dbSpecNestedChildInfo == nil {
		errStr := fmt.Sprintf("Sonic yang path %v not found in spec so cannot be processed.", xlateParams.requestUri)
		return tlerr.InternalError{Format: errStr, Path: xlateParams.requestUri}
	}

	if dbSpecNestedChildInfo.dbEntry == nil {
		errStr := fmt.Sprintf("Yang entry not found for Sonic yang path in spec so cannot be processed.", xlateParams.requestUri)
		return tlerr.InternalError{Format: errStr, Path: xlateParams.requestUri}
	}

	if dbSpecNestedChildInfo.dbEntry.Parent == nil {
		errStr := fmt.Sprintf("Parent node yang entry not found for Sonic yang path in spec so cannot be processed.", xlateParams.requestUri)
		return tlerr.InternalError{Format: errStr, Path: xlateParams.requestUri}
	}

	if dbSpecNestedChildInfo.yangType == YANG_LIST && dbSpecNestedChildInfo.dbEntry.Parent.IsList() { //nested list case
		if strings.HasSuffix(xlateParams.requestUri, "]") || strings.HasSuffix(xlateParams.requestUri, "]/") { // target URI is at nested list-instance
			nestedListInstanceValue := extractLeafValFromUriKey(xlateParams.requestUri, dbSpecNestedChildInfo.keyList[0])
			xfmrLogDebug("Nested List Instance Value : %v", nestedListInstanceValue)
			if nestedListInstanceValue != "" {
				//nested list instance becomes field-name
				fieldNm = nestedListInstanceValue
				fieldVal = ""
			} else {
				errStr := fmt.Sprintf("List instance couldn't be extracted from URI - %v", xlateParams.requestUri)
				return tlerr.InternalError{Format: errStr, Path: xlateParams.requestUri}
			}
		} else if strings.HasSuffix(xlateParams.requestUri, nestedChildNm) || strings.HasSuffix(xlateParams.requestUri, nestedChildNm+"/") {
			//target URI is at nested whole list level hence all fields in the table-instance should be replaced with NULL/NULL
			xfmrLogDebug("Target URI is at nested whole list level hence all fields in the table-instance should be replaced with NULL/NULL")
			oper = REPLACE
			fieldNm, fieldVal = "NULL", "NULL"
		} else {
			xfmrLogDebug("Target URI is at nested list non-key leaf, reject it.")
			return tlerr.NotSupportedError{Format: "DELETE not supported", Path: xlateParams.requestUri}
		}
	} else { //non-nested list case
		errStr := fmt.Sprintf("For sonic yang only nested list under list is supported, other type of yang node not supported - %v", xlateParams.requestUri)
		return tlerr.NotSupportedError{Format: errStr, Path: xlateParams.requestUri}
	}

	xlateParams.result[xlateParams.tableName][xlateParams.keyName] = db.Value{Field: map[string]string{fieldNm: fieldVal}}
	xlateParams.resultMap[oper][db.ConfigDB] = xlateParams.result
	return nil
}
