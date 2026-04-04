////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2024 Dell, Inc.                                                 //
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
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
)

func processTargetUriForReplace(xlateParams xlateToParams, skipDelete *bool) error {
	/* This function identifies if the sonic table mapped to the request URI is owned by the node or not and decide if
	   the DB table key entry should be in UPDATE map or REPLACE map so that related payload(leaf/leaf-list) can go in correct
	   operation map.This function also decides if DELETE processing is to be skipped following a REPLACE when request URI
	   is Leaf or Leaf-list or list instance that  doesn't exist in DB or a subtree-xfmr is present at request URI with
	   no nested child subtree-xfmr.
	*/

	isTblOwner := true
	var dbs [db.MaxDB]*db.DB

	// Check for subtree at request URI
	xpath, _, _ := XfmrRemoveXPATHPredicates(xlateParams.requestUri)
	spec, ok := xYangSpecMap[xpath]
	if !ok {
		errStr := fmt.Sprintf("Invalid yang-path(\"%v\").", xpath)
		return tlerr.InternalError{Format: errStr}
	}
	if len(spec.xfmrFunc) > 0 {
		if !spec.hasChildSubTree {
			if skipDelete != nil {
				*skipDelete = true
				xfmrLogDebug("Request URI has subtree annotated with no child subtree hence skip DELETE after REPLACE.")
			}
		}
		return nil
	}

	/* skip Delete and follow regular REPLACE flow for request URI thats a :
	   a) leaf
	   b) leaf-list
	   c) terminal container(having only leaves/leaf-list(s) children).Infra handles partial-relace
	      at terminal container using Aux map & common_app flow appropriately
	*/
	if spec.yangType == YANG_LEAF || spec.yangType == YANG_LEAF_LIST {
		xlateParams.replaceInfo.targetHasNonTerminalNode = false
		if skipDelete != nil {
			*skipDelete = true
			xfmrLogDebug("Request URI is leaf/leaf-list hence skip DELETE after REPLACE.")
		}
		return nil
	}
	if spec.yangType == YANG_CONTAINER && !spec.hasNonTerminalNode {
		xlateParams.replaceInfo.targetHasNonTerminalNode = false
		if skipDelete != nil {
			*skipDelete = true
		}
		xfmrLogDebug("Request URI is terminal container hence skip DELETE after REPLACE.")
		return nil
	}

	/* whole list case at request URI,so no table-instance.*/
	if spec.yangType == YANG_LIST {
		if !spec.hasNonTerminalNode {
			xlateParams.replaceInfo.targetHasNonTerminalNode = false
		}
		if !(strings.HasSuffix(xlateParams.requestUri, "]") || strings.HasSuffix(xlateParams.requestUri, "]/")) {
			return nil
		} else if !spec.hasNonTerminalNode {
			/*if the request URI list-instance belongs to a terminal list
			  then the aux-map/partial-replace handling in common_app will do
			  the needful so DELETE traversal can be skipped.
			*/
			if skipDelete != nil {
				*skipDelete = true
			}
			xfmrLogDebug("Request URI is terminal list instance hence skip DELETE after REPLACE.")
			return nil
		}
	}

	/*skip further processing if virtual table annotated*/
	if spec.virtualTbl != nil && *spec.virtualTbl {
		xfmrLogDebug("Request URI has virtual table annotated.")
		return nil
	}

	// Check the table owner flag if annotated
	checkTblOwnerThruInheritance := true
	if spec.tblOwner != nil {
		if !*spec.tblOwner {
			isTblOwner = false
		}
		checkTblOwnerThruInheritance = false
		xfmrLogDebug("Target uri contains table owner annoatation with value %v.", *spec.tblOwner)
	}

	xpathKeyExtRet, cerr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap,
		xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
	if cerr != nil {
		return cerr
	}
	curKey := xpathKeyExtRet.dbKey
	curTbl := xpathKeyExtRet.tableName
	if len(curTbl) == 0 || len(curKey) == 0 {
		xfmrLogDebug("Target uri didn't translate to a table or key so proceed to payload processing.")
		return nil
	}
	if xpathKeyExtRet.isVirtualTbl {
		xfmrLogDebug("Target uri table transformer reported virtual table.")
		return nil
	}
	if xpathKeyExtRet.isNotTblOwner {
		checkTblOwnerThruInheritance = false
		xfmrLogDebug("Target uri table transformer reported not table owner.")
	}

	if checkTblOwnerThruInheritance {
		xfmrLogDebug("Checking table ownership through inheritance.")
		parentUri := parentUriGet(xlateParams.requestUri)
		parentXpathKeyExtRet, perr := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, parentUri, xlateParams.requestUri, nil, xlateParams.subOpDataMap,
			xlateParams.txCache, xlateParams.xfmrDbTblKeyCache, dbs)
		parentTbl := parentXpathKeyExtRet.tableName
		parentKey := parentXpathKeyExtRet.dbKey

		if perr != nil {
			return perr
		}

		if parentTbl == curTbl && curKey == parentKey {
			// Parent and current node mapped to same instance
			isTblOwner = false
			xfmrLogDebug("Target uri inherits table and key from parent so its not table owner.")
		}
	}

	/*check for target URI resource existance */
	_, derr := dbTableExists(xlateParams.d, curTbl, curKey, xlateParams.oper)

	/* dbTableExists() returns Resource Not found(NotFoundError) error along with other error.
	   In order to skip Delete after Replace for list instance target URI, it should be
	   correctly known whether Tbl instance doesn't exists, which is when dbTableExists()
	   returns tlerr.NotFoundError error.Also if target URI is container that has complex child
	   nodes then that resource needs get created if it doesn't exist, in case there is payload for it.
	*/
	resourceExist := false
	if derr != nil {
		if _, ok := derr.(tlerr.NotFoundError); ok && strings.Contains(derr.Error(), "Resource not found") {
			xfmrLogDebug("Resource doesn't exist in DB.(%v)", derr.Error())
		} else {
			xfmrLogDebug("Resource existance in DB couldn't be determined.(%v)", derr)
		}
	} else {
		resourceExist = true
	}

	/* Handle list instance case, whole list already handled above. */
	if spec.yangType == YANG_LIST && !resourceExist && skipDelete != nil {
		/*skip DELETE after REPLACE processing if instance doesn't exist in DB*/
		*skipDelete = true
		xfmrLogDebug("List instance doesn't exist in DB hence skip DELETE after REPLACE.")
	}

	if isTblOwner || ((spec.yangType == YANG_LIST) && (!resourceExist)) {
		/* Table owner can REPLACE the instance.So add it to result(map where REPLACE result is
		   added when payload processing).For NON table-owner if resource doesn't exist, the instance
		   can still be added to result(REPLACE), so that cases where applications use the presence of that
		   instance in global default value map(inParams.yangDefValMap - filled for instances in result) to
		   add all defaults for a SONIC table-instance.eg.BGP_GLOBALS post-xfmr.
		*/
		xlateParams.result[curTbl] = map[string]db.Value{curKey: {Field: map[string]string{"NULL": "NULL"}}}
	} else {
		/*If resource exists then Non table owner should update the instance not replace/swap it hence put it in UPDATE map*/
		allocateSubOpDataMapForOper(xlateParams.replaceInfo.subOpDataMap, UPDATE)
		subOpUpdateMap, updateOk := xlateParams.replaceInfo.subOpDataMap[UPDATE]
		if updateOk {
			if _, curTblOk := (*subOpUpdateMap)[db.ConfigDB][curTbl]; !curTblOk {
				(*subOpUpdateMap)[db.ConfigDB][curTbl] = make(map[string]db.Value)
			}
			if _, curKeyOk := (*subOpUpdateMap)[db.ConfigDB][curTbl][curKey]; !curKeyOk {
				(*subOpUpdateMap)[db.ConfigDB][curTbl][curKey] = db.Value{Field: map[string]string{"NULL": "NULL"}}
			}
		}
	}

	//Add the uri to default value cacahe to have the defaults filled once payload processing is done
	// tblXpathMap used for default value processing for a given request
	if tblUriMapVal, tblUriMapOk := xlateParams.tblXpathMap[curTbl][curKey]; !tblUriMapOk {
		if _, tblOk := xlateParams.tblXpathMap[curTbl]; !tblOk {
			xlateParams.tblXpathMap[curTbl] = make(map[string]map[string]bool)
		}
		tblUriMapVal = map[string]bool{xlateParams.requestUri: true}
		xlateParams.tblXpathMap[curTbl][curKey] = tblUriMapVal
	} else {
		if tblUriMapVal == nil {
			tblUriMapVal = map[string]bool{xlateParams.requestUri: true}
		} else {
			tblUriMapVal[xlateParams.requestUri] = true
		}
		xlateParams.tblXpathMap[curTbl][curKey] = tblUriMapVal
	}

	return nil
}

func dataToDBMapForReplace(xlateParams xlateToParams, field string, value string) {
	var updateOk, sendSubOpMapUpdate bool
	var subOpUpdateMap *RedisDbMap

	curTbl := xlateParams.tableName
	curKey := xlateParams.keyName

	if xlateParams.replaceInfo == nil { //ideally will never happen, just added for safety
		xfmrLogInfo("replace processing info not set.")
		return
	}

	xfmrLogDebug("Received uri: %v, Table: %v, Key: %v, Field: %v, Value: %v, Resultmap(REPLACE oper): %v, "+
		"replaceInfo.subOpDataMap: %v", xlateParams.uri, curTbl, curKey, field, value,
		xlateParams.result, subOpDataMapType(xlateParams.replaceInfo.subOpDataMap))

	/* This function processes payload and translates it to appropriate operation DB result map.*/
	subOpUpdateMap, updateOk = xlateParams.replaceInfo.subOpDataMap[UPDATE]
	if updateOk && subOpUpdateMap != nil {
		if tblRw, ok := (*subOpUpdateMap)[db.ConfigDB][curTbl][curKey]; ok {
			for fld := range tblRw.Field {
				if fld == field {
					xfmrLogDebug("Field %v already present in subOpMapData under UPDATE operation so no need to refill.", field)
					return
				}
			}
			sendSubOpMapUpdate = true
		}
	}
	xpathInfo, ok := xYangSpecMap[xlateParams.xpath]
	if !ok {
		xfmrLogInfo("Invalid yang-path(\"%v\").", xlateParams.xpath)
		return
	}

	tblOwner := !xlateParams.isNotTblOwner
	if xpathInfo.tblOwner != nil {
		tblOwner = *xpathInfo.tblOwner
	}

	/*if the request URI(identified using replaceInfo.targetHasNonTerminalNode) is
	  leaf/leaf-list/terminal container/terminal list then the aux-map/partial-replace
	  handling in common_app will do the needful for non-table owner cases too
	  so no need to add to subOpData UPDATE.
	*/
	if !sendSubOpMapUpdate && xlateParams.replaceInfo.targetHasNonTerminalNode && !tblOwner {
		configDbOk := false
		if updateOk && subOpUpdateMap != nil {
			_, configDbOk = (*subOpUpdateMap)[db.ConfigDB]
		}
		if !updateOk || subOpUpdateMap == nil || !configDbOk {
			allocateSubOpDataMapForOper(xlateParams.replaceInfo.subOpDataMap, UPDATE)
			subOpUpdateMap = xlateParams.replaceInfo.subOpDataMap[UPDATE]
		}
		sendSubOpMapUpdate = true
	}

	if sendSubOpMapUpdate {
		dataToDBMapAdd(curTbl, curKey, (*subOpUpdateMap)[db.ConfigDB], field, value)
	} else {
		dataToDBMapAdd(curTbl, curKey, xlateParams.result, field, value)
	}
	xfmrLogDebug("After processing uri: %v, Table: %v, Key: %v, Field: %v, Value: %v, "+
		"Resultmap(REPLACE oper): %v, replaceInfo.subOpDataMap: %v", xlateParams.uri, curTbl, curKey, field, value,
		xlateParams.result, subOpDataMapType(xlateParams.replaceInfo.subOpDataMap))

}

func dbMapDefaultFieldValFillForReplace(xlateParams xlateToParams, tblUriList []string) error {
	var result, yangDefValMap, yangAuxValMap map[string]map[string]db.Value

	/* In REPLACE flow non-table-owner nodes' payload data is added to subOpDataMap[UPDATE] */
	if xlateParams.replaceInfo != nil && xlateParams.replaceInfo.isNonTblOwnerDefaultValProcess {
		/* Defaults for leaves inside non-table-owner nodes must be reset even if instance exists
		   in DB so transfer them to result.Non-default leaves inside non-table-owner nodes
		   which were not present in payload should be deleted, so transfer them from auxValmap to
		   original/global DELETE subOpDataMap.
		*/
		result = (*xlateParams.replaceInfo.subOpDataMap[UPDATE])[db.ConfigDB]
		yangDefValMap = (*xlateParams.replaceInfo.subOpDataMap[UPDATE])[db.ConfigDB]
		yangAuxValMap = (*xlateParams.replaceInfo.subOpDataMap[DELETE])[db.ConfigDB]
	} else {
		result = xlateParams.result
		yangDefValMap = xlateParams.yangDefValMap
		yangAuxValMap = xlateParams.yangAuxValMap
	}

	tblData := result[xlateParams.tableName]
	var dbs [db.MaxDB]*db.DB
	tblName := xlateParams.tableName
	isNotTblOwner := xlateParams.isNotTblOwner
	dbKey := xlateParams.keyName
	defSubOpDataMap := make(map[Operation]*RedisDbMap)
	for _, tblUri := range tblUriList {
		xfmrLogDebug("Processing URI %v for default value filling(Table - %v, dbKey - %v)", tblUri, tblName, dbKey)
		yangXpath, _, prdErr := XfmrRemoveXPATHPredicates(tblUri)
		if prdErr != nil {
			continue
		}
		yangNode, ok := xYangSpecMap[yangXpath]
		if ok && yangNode.yangEntry != nil {
			for childName := range yangNode.yangEntry.Dir {
				childXpath := yangXpath + "/" + childName
				childUri := tblUri + "/" + childName
				childNode, ok := xYangSpecMap[childXpath]
				if ok {
					if len(childNode.xfmrFunc) > 0 {
						xfmrLogDebug("Skip default filling since a subtree Xfmr found for path - %v", childXpath)
						continue
					}
					yangType := childNode.yangType
					childNodeYangEntry := yangNode.yangEntry.Dir[childName]
					if childNodeYangEntry == nil {
						xfmrLogDebug("yang entry is nil for xpath %v", childXpath)
						continue
					}
					if ((yangType == YANG_LIST) || (yangType == YANG_CONTAINER)) && (!childNodeYangEntry.ReadOnly()) {
						var tblList []string
						if childNode.tableName != nil && *childNode.tableName != tblName {
							continue
						}
						if childNode.xfmrTbl != nil {
							if len(*childNode.xfmrTbl) > 0 {
								inParamsTblXfmr := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, childUri, xlateParams.requestUri, xlateParams.oper, "", nil, defSubOpDataMap, "", xlateParams.txCache)
								chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
								xfmrLogDebug("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
								if ctErr != nil || chldTblNm != tblName {
									continue
								}
							}
						}
						tblList = append(tblList, childUri)
						err := dbMapDefaultFieldValFillForReplace(xlateParams, tblList)
						if err != nil {
							return err
						}
					}
					/*terminal-node/leaf case*/
					_, ok := tblData[dbKey].Field[childName]
					if !ok {
						if len(childNode.xfmrField) > 0 {
							childYangDataType := childNodeYangEntry.Type.Kind
							var param interface{}
							oper := xlateParams.oper
							if len(childNode.defVal) > 0 {
								xfmrLogDebug("Update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
									childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
								_, defValPtr, err := DbToYangType(childYangDataType, childXpath, childNode.defVal, xlateParams.oper)
								if err == nil && defValPtr != nil {
									param = defValPtr
								} else {
									xfmrLogDebug("Failed to update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
										childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
								}
							} else {
								// non-default field not part of translated result should be deleted so add it to auxValMap
								oper = DELETE
							}
							inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, tblUri+"/"+childName, xlateParams.requestUri, oper, "", nil, defSubOpDataMap, param, xlateParams.txCache)
							retData, err := leafXfmrHandler(inParams, childNode.xfmrField)
							if err != nil {
								log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.xfmrField)
							}
							if retData != nil {
								xfmrLogDebug("xfmr function : %v Xpath: %v retData: %v", childNode.xfmrField, childXpath, retData)
								for f, v := range retData {
									// Fill default value only if value is not available in result Map
									// else we overwrite the value filled in resultMap with default value
									_, ok := result[tblName][dbKey].Field[f]
									if !ok {
										if len(childNode.defVal) > 0 {
											dataToDBMapAdd(tblName, dbKey, yangDefValMap, f, v)
										} else {
											// Fill the yangAuxValMap with all fields that are not in either resultMap or defaultValue Map
											dataToDBMapAdd(tblName, dbKey, yangAuxValMap, f, "")
										}
									}
								}
							}
						} else if len(childNode.fieldName) > 0 {
							var xfmrErr error
							if xDbSpecInfo, ok := xDbSpecMap[tblName+"/"+childNode.fieldName]; ok && (xDbSpecInfo != nil) && (!xDbSpecInfo.isKey) {
								// Fill default value only if value is not available in result Map
								// else we overwrite the value filled in resultMap with default value
								dbFieldNm := childNode.fieldName
								if xDbSpecInfo.yangType == YANG_LEAF_LIST {
									dbFieldNm += "@"
								}
								_, ok = result[tblName][dbKey].Field[dbFieldNm]
								if !ok {
									if len(childNode.defVal) > 0 {
										curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, childXpath, dbKey, xlateParams.jsonData, xlateParams.resultMap, yangDefValMap, xlateParams.txCache, xlateParams.tblXpathMap, defSubOpDataMap, xlateParams.pCascadeDelTbl, &xfmrErr, childName, childNode.defVal, tblName, isNotTblOwner, xlateParams.invokeCRUSubtreeOnceMap, nil, nil, xlateParams.replaceInfo)
										err := mapFillDataUtil(curXlateParams, true)
										if err != nil {
											log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.fieldName)
										}
									} else {
										dataToDBMapAdd(tblName, dbKey, yangAuxValMap, dbFieldNm, "")
									}
								}
							}
						} else if childNode.isRefByKey {
							/* this case occurs only for static table-name, table-xfmr always has field-name
							   assigned by infra when there no explicit annotation from user.
							   Also key-leaf in config container has no default value, so here we handle only
							   REPLACE yangAuxValMap filling */
							_, ok := result[tblName][dbKey].Field["NULL"]
							if !ok {
								dataToDBMapAdd(tblName, dbKey, yangAuxValMap, "NULL", "")
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func dbMapDefaultValFillForReplace(xlateParams xlateToParams) error {

	xfmrLogDebug("Before default value processing of result for REPLACE xlateParams.result %v, xlateParams.subOpDataMap %v, xlateParams.yangDefValMap %v", xlateParams.result, subOpDataMapType(xlateParams.subOpDataMap), xlateParams.yangDefValMap)
	for tbl, tblData := range xlateParams.result {
		if _, tblOk := xlateParams.tblXpathMap[tbl]; !tblOk {
			continue
		}
		for dbKey := range tblData {
			var yxpathList []string //contains all uris(with keys) that were traversed for a table while processing the incoming request
			if tblUriMapVal, ok := xlateParams.tblXpathMap[tbl][dbKey]; !ok {
				continue
			} else {
				for tblUri := range tblUriMapVal {
					yxpathList = append(yxpathList, tblUri)
				}
			}
			if len(yxpathList) == 0 {
				continue
			}

			curXlateParams := xlateParams
			curXlateParams.tableName = tbl
			curXlateParams.keyName = dbKey
			err := dbMapDefaultFieldValFillForReplace(curXlateParams, yxpathList)
			if err != nil {
				return err
			}
			tblInstDefaultFields, deflvalOk := xlateParams.yangDefValMap[tbl][dbKey]
			if !deflvalOk {
				tblInstDefaultFields = db.Value{}
			}
			removeTblInstMappedtoContainer(xlateParams.result, tbl, dbKey, tblInstDefaultFields, yxpathList)
		}
	}
	xfmrLogDebug("After default value processing of result for REPLACE xlateParams.result %v, xlateParams.subOpDataMap %v,"+
		" xlateParams.yangDefValMap %v, xlateParams.yangAuxValMap %v", xlateParams.result, subOpDataMapType(xlateParams.subOpDataMap),
		xlateParams.yangDefValMap, xlateParams.yangAuxValMap)

	// non-table owner infra handled cases are filled in erplaceInfo.subOpDataMap.Perform default value processing for them.
	if (xlateParams.replaceInfo == nil) || (!xlateParams.replaceInfo.targetHasNonTerminalNode) {
		return nil
	}

	var subOpUpdateMap *RedisDbMap
	var updateDbDataMap map[string]map[string]db.Value
	var updateOk, cfgOk, subOpDelMapAllocated bool
	if subOpUpdateMap, updateOk = xlateParams.replaceInfo.subOpDataMap[UPDATE]; !updateOk || subOpUpdateMap == nil {
		return nil
	}
	if updateDbDataMap, cfgOk = (*subOpUpdateMap)[db.ConfigDB]; !cfgOk || (len(updateDbDataMap) == 0) {
		return nil
	}
	xfmrLogDebug("Before default value processing of result for UPDATE(non table owners filled by infra) replaceInfo.subOpDataMap %v", subOpDataMapType(xlateParams.replaceInfo.subOpDataMap))
	xlateParams.replaceInfo.isNonTblOwnerDefaultValProcess = true
	for tbl, tblData := range updateDbDataMap {
		if _, tblOk := xlateParams.tblXpathMap[tbl]; !tblOk {
			continue
		}
		for dbKey := range tblData {
			var yxpathList []string
			if tblUriMapVal, ok := xlateParams.tblXpathMap[tbl][dbKey]; !ok {
				continue
			} else {
				for tblUri := range tblUriMapVal {
					yxpathList = append(yxpathList, tblUri)
				}
			}

			if len(yxpathList) == 0 {
				continue
			}

			if !subOpDelMapAllocated {
				allocateSubOpDataMapForOper(xlateParams.replaceInfo.subOpDataMap, DELETE)
				subOpDelMapAllocated = true
			}
			/* check if the same instance has been processed above as part of REPLACE result map instance processing.
			   Same instance might get added into result of REPLACE by subtree, since during payload processing immediately
			   after every subtree call the subtree returned result is consolidated into infra replace result map.
			   The default value cache(xlateParams.tblXpathMap) is indexed based on table-key,so all infra-handled yang-paths
			   mapped against it are already processed in above default value processing and the global yangDefValMap and yangAuxValMap filled accordingly.
			   So transfer data from global yangDefValMap for the instance into replaceInfo.subOpDataMap[UPDATE]
			   after cross checking against what was filled from infra-handled payload into replaceInfo.subOpDataMap[UPDATE].
			   replaceInfo.subOpDataMap[UPDATE] will be consolidated with Replace result map later.
			   Also since the instance is going to be replaced clear off the instance from global yangAuxvalMap.
			*/

			if _, instanceFoundInReplaceResultMap := xlateParams.result[tbl][dbKey]; instanceFoundInReplaceResultMap {
				if defValMapInstanceFields, instanceYangDefValMapOk := xlateParams.yangDefValMap[tbl][dbKey]; instanceYangDefValMapOk {
					for field, val := range defValMapInstanceFields.Field {
						if _, fldOk := updateDbDataMap[tbl][dbKey].Field[field]; !fldOk {
							updateDbDataMap[tbl][dbKey].Field[field] = val
						}
					}
					delete(xlateParams.yangDefValMap[tbl], dbKey)
					if len(xlateParams.yangDefValMap[tbl]) == 0 {
						delete(xlateParams.yangDefValMap, tbl)
					}
				}
				if _, instanceYangAuxValMapOk := xlateParams.yangAuxValMap[tbl][dbKey]; instanceYangAuxValMapOk {
					delete(xlateParams.yangAuxValMap[tbl], dbKey)
					if len(xlateParams.yangAuxValMap[tbl]) == 0 {
						delete(xlateParams.yangAuxValMap, tbl)
					}
				}
				xfmrLogDebug("Skipping default value processing for instance [%v][%v] as it is already processed in REPLACE result map."+
					" xlateParams.replaceInfo.subOpDataMap %v, xlateParams.yangDefValMap %v, xlateParams.yangAuxValMap %v",
					tbl, dbKey, subOpDataMapType(xlateParams.replaceInfo.subOpDataMap), xlateParams.yangDefValMap, xlateParams.yangAuxValMap)
				removeTblInstMappedtoContainer(updateDbDataMap, tbl, dbKey, db.Value{}, yxpathList)
				continue
			}

			curXlateParams := xlateParams
			curXlateParams.tableName = tbl
			curXlateParams.isNotTblOwner = true
			curXlateParams.keyName = dbKey

			err := dbMapDefaultFieldValFillForReplace(curXlateParams, yxpathList)
			if err != nil {
				return err
			}
			removeTblInstMappedtoContainer(updateDbDataMap, tbl, dbKey, db.Value{}, yxpathList)
		}
	}

	xfmrLogDebug("Returning from default value processing for REPLACE xlateParams.result(Replace oper result) %v, xlateParams.subOpDataMap %v,"+
		"xlateParams.yangDefValMap %v, xlateParams.yangAuxValMap %v, xlateParams.replaceInfo.subOpDataMap %v", xlateParams.result,
		subOpDataMapType(xlateParams.subOpDataMap), xlateParams.yangDefValMap, xlateParams.yangAuxValMap, subOpDataMapType(xlateParams.replaceInfo.subOpDataMap))
	return nil
}

func addToDeleteForReplaceMap(tableName string, dbKey string, field string, replaceResultMap map[Operation]RedisDbMap) (bool, bool) {
	// This function is called only for isDeleteForReplace case and checks if the table,dbKey and field is already present in replaceResultMap
	//Inputs - table, dbKey and fieldName that needs to be verified if available in replaceResultMap, resultMap (having consolidated result and subOpMap from REPLACE
	// Output - bool indicating add to delete resultMap required or not, bool to indicate skipSiblingTraversal for the field in input param
	addToMap := true
	skipSiblingTraversal := false

	// Check if table instance exists in replace resultMap
	for op, redisMap := range replaceResultMap {
		for _, tblMap := range redisMap {
			for tbl, instMap := range tblMap {
				if tbl != tableName {
					// If table not found continue to loop to chk if tableName exists in replaceResultMap
					continue
				} else {
					for key, fieldVal := range instMap {
						if key != dbKey {
							// If instance not found continue to loop to check if dbKey exists in replaceResultMap
							continue
						} else {
							// Instance found in replaceResultMap
							// If no field and value is found in input args it is a table instance, mark instance to be not added to delete resultMap
							if field == "" {
								addToMap = false
							} else if field == "FillFields" {
								// For non table owners add FillFields to identify fields to be deleted from the yang tree
								addToMap = true
							} else {
								// Check for field availability
								// If the instance is found in REPLACE resultMap for field level fill, then do not add to delete result as its a complete instance replace
								if op == REPLACE {
									addToMap = false
									skipSiblingTraversal = true
								}
								for fld := range fieldVal.Field {
									if fld == field {
										// If field found in replaceResultMap for non REPLACE operations, do not add field to deleteMap
										addToMap = false
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return addToMap, skipSiblingTraversal
}

func processDeleteForReplace(xlateParams xlateToParams) error {
	var dbs [db.MaxDB]*db.DB
	var xfmrErr error
	jsonData := make(map[string]interface{})

	xfmrLogDebug("Before merging REPLACE before going to DELETE xlateParams.resultMap - %v xlateParams.result %v xlateParams.subOpDataMap %v", xlateParams.resultMap, xlateParams.result, subOpDataMapType(xlateParams.subOpDataMap))
	/*merge REPLACE result and subOpMap into resultMap to pass to DELETE flow*/
	mergeSubOpMapWithResultForReplace(xlateParams.resultMap, xlateParams.result, xlateParams.subOpDataMap)

	xfmrLogDebug("After merging REPLACE before going to DELETE xlateParams.resultMap - %v ", xlateParams.resultMap)

	/*pass empty result and subOpDataMap as it would be for normal DELETE procesing*/
	result := make(map[string]map[string]db.Value)
	subOpDataMap := make(map[Operation]*RedisDbMap)
	xpathKeyExtRet, err := xpathKeyExtract(xlateParams.d, xlateParams.ygRoot, DELETE, xlateParams.uri, xlateParams.requestUri, nil, xlateParams.subOpDataMap, xlateParams.txCache, nil, dbs)
	if err != nil {
		return err
	}
	xfmrLogInfo("Delete req for Replace: uri(\"%v\"), key(\"%v\"), xpath(\"%v\"), tableName(\"%v\").", xlateParams.uri, xpathKeyExtRet.dbKey, xpathKeyExtRet.xpath, xpathKeyExtRet.tableName)

	xlateParamsForDelete := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, DELETE, xlateParams.uri, xlateParams.requestUri, xpathKeyExtRet.xpath, xpathKeyExtRet.dbKey, jsonData,
		xlateParams.resultMap, result, xlateParams.txCache, xlateParams.tblXpathMap, subOpDataMap, xlateParams.pCascadeDelTbl, &xfmrErr, "", "", xpathKeyExtRet.tableName,
		xlateParams.isNotTblOwner, nil, nil, nil, xlateParams.replaceInfo)
	curResult, cerr := allChildTblGetToDelete(xlateParamsForDelete)
	if cerr != nil {
		err = cerr
		return err
	} else {
		xfmrLogDebug("allChildTblGetToDelete Before subtree result merge result: %v  subtree curResult: %v subtree subOpDataMap: %v", result, curResult, subOpDataMapType(subOpDataMap))
		// merge the DELETE translated maps by infra and subtrees
		mapCopy(result, curResult)
		xfmrLogDebug("allChildTblGetToDelete DELETE merged result: %v ", result)
	}

	// Merge the xlateParams.result and xlateParams.subOpDataMap having result of delete traversal for Replace into the resultMap generated for replace payload.
	mergeDeleteResultMaptoReplaceResultMap(xlateParamsForDelete)

	xfmrLogDebug("After merging DELETE result and subop with REPLACE xlateParams.resultMap - %v \n", xlateParamsForDelete.resultMap)

	/* Reset the result and subOpDataMap with data from resultMap for REPLACE flow since post-xfmr and regular REPLACE flow expects that way*/
	for op, redisMap := range xlateParamsForDelete.resultMap {
		// Reset the subopMap to empty Map as it's current data has already been merged to xlateParams.resultMap
		// Copy into subOpMap data from merged xlateParams.resultMap
		allocateSubOpDataMapForOper(xlateParams.subOpDataMap, op)
		operMap := make(RedisDbMap)
		operMap = redisMap
		xlateParams.subOpDataMap[op] = &operMap

		if op == REPLACE {
			xlateParams.result = redisMap[db.ConfigDB]
			continue
		}

		if op == DELETE {
			// Add Delete tables in resultMap to cascade delete table list
			for tbNm, dbKeyMap := range redisMap[db.ConfigDB] {
				for _, fieldVal := range dbKeyMap {
					// Cascade delete supported for instance level delete only
					if len(fieldVal.Field) == 0 {
						if tblSpecInfo, ok := xDbSpecMap[tbNm]; ok && tblSpecInfo.cascadeDel == XFMR_ENABLE {
							if !contains(*xlateParams.pCascadeDelTbl, tbNm) {
								*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tbNm)
							}
						}
					}
				}
			}
		}
	}

	xfmrLogDebug("After rearraging resultMap for post xfmr xlateParams.resultMap - %v xlateParams.result - %v, xlateParams.subOpDataMap - %v", xlateParamsForDelete.resultMap, xlateParams.result, subOpDataMapType(xlateParams.subOpDataMap))

	return nil
}

func mergeDeleteResultMaptoReplaceResultMap(xlateParams xlateToParams) {
	/* Cross Check & Consolidate REPLACE and DELETE data at end of DELETE for internal REPLACEDELETE:
	   Cross-check and consolidate data into resultMap["DELETE"] from xlateParams.result(includes DELETE data from infra as well as subtree returned result for DELETE).
	   Priority is given to resultMap data since it was created from REPLACE payload and should be retained.*/

	if _, ok := xlateParams.resultMap[DELETE]; ok {
		if _, ok := xlateParams.resultMap[DELETE][db.ConfigDB]; !ok {
			xlateParams.resultMap[DELETE][db.ConfigDB] = make(map[string]map[string]db.Value)
		}
	} else {
		xlateParams.resultMap[DELETE] = make(RedisDbMap)
		xlateParams.resultMap[DELETE][db.ConfigDB] = make(map[string]map[string]db.Value)
	}

	mapMergeAcrossOperations(xlateParams.resultMap, xlateParams.result, DELETE)

	/*Cross-check and consolidate into resultMap data from subOpMap built at the end of DELETE for each oper,tbl,key,field against each
	  oper,tbl,key,field in resultMap. Priority to be given to REPLACE resultMap in case of conflicts.*/

	operList := []Operation{REPLACE, UPDATE, CREATE, DELETE}
	for _, op := range operList {
		if redisMapPtr, ok := xlateParams.subOpDataMap[op]; ok && redisMapPtr != nil {
			for dbNum, dbMap := range *redisMapPtr {
				if dbNum != db.ConfigDB {
					continue
				}
				if _, ok := xlateParams.resultMap[op]; !ok {
					xlateParams.resultMap[op] = make(RedisDbMap)
				}
				if _, ok := xlateParams.resultMap[op][dbNum]; !ok {
					xlateParams.resultMap[op][dbNum] = make(map[string]map[string]db.Value)
				}
				mapMergeAcrossOperations(xlateParams.resultMap, dbMap, op)
			}
		}
	}
}

func mapMergeAcrossOperations(resultMap map[Operation]RedisDbMap, dbMap map[string]map[string]db.Value, op Operation) {
	for tbl, instMap := range dbMap {
		for key, fieldVal := range instMap {
			instFound := false
			var operFound Operation
			instFound, operFound = instanceExistsinResultMap(tbl, key, resultMap)
			if !instFound {
				// If the instance is not found in resultMap, add them to resultMap
				if _, ok := resultMap[op][db.ConfigDB][tbl]; !ok {
					resultMap[op][db.ConfigDB][tbl] = make(map[string]db.Value)
				}
				resultMap[op][db.ConfigDB][tbl][key] = db.Value{Field: make(map[string]string)}
			}
			// If instance is found check if the fields are present in resultMap for this instance
			for fld, val := range fieldVal.Field {
				_, fieldOk := fieldExistsinResultMap(tbl, key, fld, resultMap, instFound, operFound)
				if op == DELETE {
					// For DELETE oper if whole instance found in SubOpMap do not merge that instance.
					// If the field for this instance exists in other oper other than DELETE, then too merge is not required. This field can be ignored.
					// If instance is found in resultMap and found for REPLACE Operation no merge is required as the whole instance is being replaced
					// If the instance is found in other opers (CREATE/UPDATE/DELETE) in ReplaceResultMap and field for this instance is not available in other oper in resultMap,
					// then it can be merged as it may target a specific field delete.
					if !instFound {
						resultMap[op][db.ConfigDB][tbl][key].Field[fld] = val
					} else if instFound && operFound != REPLACE && !fieldOk {
						// Merge Leaves into the resultMap for Delete operation
						if _, ok := resultMap[op][db.ConfigDB][tbl]; !ok {
							resultMap[op][db.ConfigDB][tbl] = make(map[string]db.Value)
						}
						if _, ok := resultMap[op][db.ConfigDB][tbl][key]; !ok {
							resultMap[op][db.ConfigDB][tbl][key] = db.Value{Field: make(map[string]string)}
						}
						resultMap[op][db.ConfigDB][tbl][key].Field[fld] = val
					}
				} else {
					if instFound && !fieldOk {
						// Merge Leaves into the resultMap for Update/Create operation
						if _, ok := resultMap[operFound][db.ConfigDB][tbl][key]; !ok {
							// Allocate memory for fields if empty
							resultMap[operFound][db.ConfigDB][tbl][key] = db.Value{Field: make(map[string]string)}
						}
						resultMap[operFound][db.ConfigDB][tbl][key].Field[fld] = val
					} else if !instFound {
						resultMap[op][db.ConfigDB][tbl][key].Field[fld] = val
					}
				}
			}
		}
	}
}

func replcePayloadContainerProcessing(xlateParams xlateToParams) {

	// tblXpathMap used for default value processing for a given request
	if tblUriMapVal, tblUriMapOk := xlateParams.tblXpathMap[xlateParams.tableName][xlateParams.keyName]; !tblUriMapOk {
		if _, tblOk := xlateParams.tblXpathMap[xlateParams.tableName]; !tblOk {
			xlateParams.tblXpathMap[xlateParams.tableName] = make(map[string]map[string]bool)
		}
		tblUriMapVal = map[string]bool{xlateParams.uri: true}
		xlateParams.tblXpathMap[xlateParams.tableName][xlateParams.keyName] = tblUriMapVal
	} else {
		if tblUriMapVal == nil {
			tblUriMapVal = map[string]bool{xlateParams.uri: true}
		} else {
			tblUriMapVal[xlateParams.uri] = true
		}
		xlateParams.tblXpathMap[xlateParams.tableName][xlateParams.keyName] = tblUriMapVal
	}

	//Add table-instance to translated result
	dataToDBMapForReplace(xlateParams, "NULL", "NULL")
}

func combineGlobalSubOpMapWithReplaceInfoSubOpMap(subOpDataMapGlobal map[Operation]*RedisDbMap, subOpDataMapFromReplaceInfo map[Operation]*RedisDbMap) {
	/* This funtion will transfer UPDATE and DELETE oper data, generated by infra-handled non table owner cases during payload and
		   default value processing, into global subOpMapData UPDATE and DELETE oper data respectively, filled by app callbacks during Replace payload processing.
	       If there is same field then app callback field is taken just like mapCopy() does
	*/
	var operMapGlobal RedisDbMap
	var subOpMapPtr *RedisDbMap
	var operOkGlobal bool
	replaceInfoSubOper := []Operation{DELETE, UPDATE}

	xfmrLogDebug("Received subOpDataMapGlobal: %v, subOpDataMapFromReplaceInfo : %v", subOpDataMapType(subOpDataMapGlobal), subOpDataMapType(subOpDataMapFromReplaceInfo))
	for _, oper := range replaceInfoSubOper {
		if operMap, operOk := subOpDataMapFromReplaceInfo[oper]; operOk && operMap != nil {
			if operDbData, cfgOk := (*operMap)[db.ConfigDB]; cfgOk && (len(operDbData) != 0) {
				for tbl, tblData := range operDbData {
					if subOpMapPtr, operOkGlobal = subOpDataMapGlobal[oper]; !operOkGlobal {
						operMapGlobal = make(RedisDbMap)
						subOpMapPtr = &operMapGlobal
						subOpDataMapGlobal[oper] = subOpMapPtr
					}
					if _, configDbOk := (*subOpMapPtr)[db.ConfigDB]; !configDbOk {
						(*subOpMapPtr)[db.ConfigDB] = make(map[string]map[string]db.Value)
					}

					_, ok := (*subOpMapPtr)[db.ConfigDB][tbl]
					if !ok {
						(*subOpMapPtr)[db.ConfigDB][tbl] = make(map[string]db.Value)
					}
					for rule, ruleData := range tblData {
						_, ok = (*subOpMapPtr)[db.ConfigDB][tbl][rule]
						if !ok || (len((*subOpMapPtr)[db.ConfigDB][tbl][rule].Field) == 0) {
							(*subOpMapPtr)[db.ConfigDB][tbl][rule] = db.Value{Field: make(map[string]string)}
						}
						for field, value := range ruleData.Field {
							/*Do not overwrite field into global subOpMapData if it already exists.Give preference
							  to app callback filled data like mapCopy()
							*/
							if _, fldOk := (*subOpMapPtr)[db.ConfigDB][tbl][rule].Field[field]; !fldOk {
								(*subOpMapPtr)[db.ConfigDB][tbl][rule].Field[field] = value
							}
						}
					}
				}
			}
		}
	}
	subOpDataMapFromReplaceInfo = nil
	xfmrLogDebug("Returning subOpDataMapGlobal: %v", subOpDataMapType(subOpDataMapGlobal))

}

func removeTblInstMappedtoContainer(tblInstMap map[string]map[string]db.Value, tblNm string, dbkey string, tblInstDefaultFields db.Value, yangXpathListMappedFortable []string) {
	/* Enclosing Containers in payload mapping to a unique table instance should not get created or replaced with NULL:NULL entry
	   if no default values are found as well as no actual payload/leaf.So this function will remove such  table instance from infra
	   translated result(both table owners and non table owners) during REPLACE processing.yangXpathListMappedFortable contains all
	   yang complex node paths that were found mapped to a table instance during replace payload processing that need to be traversed
	   during default value processing.
	*/
	xfmrLogDebug("Received table-instance map %v, table name %v, key %v, tblInstDefFields %v, yangXpathList %v", tblInstMap, tblNm, dbkey, tblInstDefaultFields, yangXpathListMappedFortable)
	if tblInstMap == nil || tblInstMap[tblNm] == nil || len(yangXpathListMappedFortable) == 0 {
		xfmrLogDebug("Table %v, key %v not found in tblInstMap or yangXpathList is empty %v", tblNm, dbkey, len(yangXpathListMappedFortable))
		return
	}
	tblInstFields := tblInstMap[tblNm][dbkey]
	if len(tblInstFields.Field) == 1 && tblInstFields.Has("NULL") {
		if len(tblInstDefaultFields.Field) == 0 {
			removeInstMappedToContainer := true
			for _, path := range yangXpathListMappedFortable {
				if strings.HasSuffix(path, "]") { // Check to make sure only containers are mapped to this instance
					removeInstMappedToContainer = false
					break
				}
			}
			if removeInstMappedToContainer {
				if len(tblInstMap[tblNm]) == 1 {
					delete(tblInstMap, tblNm)
				} else {
					delete(tblInstMap[tblNm], dbkey)
				}
			}
		}
	}
	xfmrLogDebug("Returning tblInstMap after processing %v", tblInstMap)
}
