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

	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	log "github.com/golang/glog"
)

func getDbTblCbkName(tableName string) string {
	return tableName + "_cascade_cfg_hdl"
}

func xfmrDbTblCbkHandler(inParams XfmrDbTblCbkParams, tableName string) error {
	xfmrLogDebug("Received inParams %v xfmrDbTblCbkHandler function name %v", inParams, tableName)

	ret, err := XlateFuncCall(getDbTblCbkName(tableName), inParams)
	if err != nil {
		xfmrLogInfo("xfmrDbTblCbkHandler: %v failed.", getDbTblCbkName(tableName))
		return err
	}
	if (ret != nil) && (len(ret) > 0) {
		if ret[0].Interface() != nil {
			err = ret[0].Interface().(error)
			if err != nil {
				log.Warningf("Table callback method i(%v) returned error - %v.", getDbTblCbkName(tableName), err)
			}
		}
	}

	return err
}

func handleCascadeDelete(d *db.DB, dbDataMap map[Operation]map[db.DBNum]map[string]map[string]db.Value, cascadeDelTbl []string) error {
	xfmrLogInfo("handleCascadeDelete : %v, cascadeDelTbl : %v.", dbDataMap, cascadeDelTbl)

	var err error
	cvlSess, cvlRetSess := d.NewValidationSession()
	if cvlRetSess != nil {
		xfmrLogInfo("handleCascadeDelete : cvl.ValidationSessOpen failed.")
		err = fmt.Errorf("%v", "cvl.ValidationSessOpen failed")
		return err
	}
	defer cvl.ValidationSessClose(cvlSess)

	for operIndex, redisMap := range dbDataMap {
		if operIndex != DELETE {
			continue
		}
		for dbIndex, dbMap := range redisMap {
			if dbIndex != db.ConfigDB {
				continue
			}
			for tblIndex, tblMap := range dbMap {
				if !contains(cascadeDelTbl, tblIndex) {
					continue
				}
				for key, entry := range tblMap {
					// need to generate key based on the db type as of now just considering configdb
					// and using "|" as tablename and key seperator
					depKey := tblIndex + "|" + key
					depList := cvlSess.GetDepDataForDelete(depKey)
					xfmrLogInfo("handleCascadeDelete : depKey : %v, depList- %v, entry : %v", depKey, depList, entry)
					for depIndex, depEntry := range depList {
						for depEntkey, depEntkeyInst := range depEntry.Entry {
							depEntkeyList := strings.SplitN(depEntkey, "|", 2)
							cbkHdlName := depEntkeyList[0] + "_cascade_cfg_hdl"
							if IsXlateFuncBinded(cbkHdlName) {
								//handle callback for table call Table Call back method and consolidate the data
								inParams := formXfmrDbTblCbkParams(d, DELETE, depEntry.RefKey, depEntkeyList[0], depEntkeyList[1], depEntkeyInst, dbDataMap[DELETE])
								xfmrLogInfo("handleCascadeDelete CBKHDL present depIndex %v, inParams : %v ", depIndex, inParams)
								err = xfmrDbTblCbkHandler(inParams, depEntkeyList[0])
								if err == nil {
									for operIdx, operMap := range inParams.delDepDataMap {
										if _, ok := dbDataMap[operIdx]; !ok {
											dbDataMap[operIdx] = make(map[db.DBNum]map[string]map[string]db.Value)
										}
										for dbIndx, dbMap := range *operMap {
											if _, ok := dbDataMap[operIdx][dbIndx]; !ok {
												dbDataMap[operIdx][dbIndx] = make(map[string]map[string]db.Value)
											}
											mapMerge(dbDataMap[operIdx][dbIndx], dbMap, operIdx)
										}
									}
								} else {
									xfmrLogInfo("handleCascadeDelete - xfmrDbTblCbkHandler failed.")
									return errors.New("xfmrDbTblCbkHandler failed for table: " + depEntkeyList[0] + ", Key: " + depEntkeyList[1])
								}
							} else {
								if _, ok := dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]]; !ok {
									dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]] = make(map[string]db.Value)
								}
								if _, ok := dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]]; !ok {
									dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]] = db.Value{Field: make(map[string]string)}
								}

								if len(depEntkeyInst) > 0 {
									for depEntAttr, depEntAttrInst := range depEntkeyInst {
										if _, ok := dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]].Field[depEntAttr]; !ok {
											dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]].Field[depEntAttr] = ""
										}

										if len(depEntAttrInst) > 0 {
											val := dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]]
											if strings.HasSuffix(depEntAttr, "@") {
												valList := val.GetList(depEntAttr)
												if !contains(valList, depEntAttrInst) {
													valList = append(valList, depEntAttrInst)
													val.SetList(depEntAttr, valList)
												}
											} else {
												val.Set(depEntAttr, depEntAttrInst)
											}

											dbDataMap[DELETE][db.ConfigDB][depEntkeyList[0]][depEntkeyList[1]] = val
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
