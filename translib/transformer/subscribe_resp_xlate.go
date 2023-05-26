////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//     http://www.apache.org/licenses/LICENSE-2.0                             //
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
	"reflect"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

type subscribeNotfRespXlator struct {
	ntfXlateReq   *subscribeNotfXlateReq
	dbYgXlateList []*DbYgXlateInfo
}

type subscribeNotfXlateReq struct {
	path     *gnmi.Path
	dbNum    db.DBNum
	table    *db.TableSpec
	key      *db.Key
	entry    *db.Value
	dbs      [db.MaxDB]*db.DB
	opaque   interface{}
	reqLogId string
}

type DbYgXlateInfo struct {
	pathIdx     int
	ygXpathInfo *yangXpathInfo
	tableName   string
	dbKey       string
	uriPath     string
	xlateReq    *subscribeNotfXlateReq
}

func NewSubscribeNotfRespXlator(ctxID interface{}, gPath *gnmi.Path, dbNum db.DBNum, table *db.TableSpec, key *db.Key,
	entry *db.Value, dbs [db.MaxDB]*db.DB, opaque interface{}) (*subscribeNotfRespXlator, error) {
	reqLogId := "subNotfReq Id:[" + fmt.Sprintf("%v", ctxID) + "] : "

	if log.V(dbLgLvl) {
		log.Infof("%v NewSubscribeNotfRespXlator: table: %v, key: %v, "+
			"dbno: %v, path: %v", reqLogId, table, key, dbNum, gPath)
	}

	if opaque == nil || (reflect.ValueOf(opaque).Kind() == reflect.Ptr && reflect.ValueOf(opaque).IsNil()) {
		opaque = new(sync.Map)
	}

	xlateReq := subscribeNotfXlateReq{gPath, dbNum, table, key, entry, dbs, opaque, reqLogId}
	return &subscribeNotfRespXlator{ntfXlateReq: &xlateReq}, nil
}

func (respXlator *subscribeNotfRespXlator) Translate() (*gnmi.Path, error) {
	ntfXlateReq := respXlator.ntfXlateReq

	if log.V(dbLgLvl) {
		log.Info(ntfXlateReq.reqLogId, "subscribeNotfRespXlator:Translate: path: ", ntfXlateReq.path)
	}

	pathElem := respXlator.ntfXlateReq.path.Elem

	for idx := len(pathElem) - 1; idx >= 0; idx-- {

		ygPath := respXlator.getYangListPath(idx)
		if log.V(dbLgLvl) {
			log.Info(ntfXlateReq.reqLogId, "subscribeNotfRespXlator:Translate:ygPath: ", ygPath)
		}
		ygXpathInfo, err := respXlator.getYangXpathInfo(ygPath)
		if err != nil {
			return nil, err
		}

		if log.V(dbLgLvl) {
			log.Info(ntfXlateReq.reqLogId, "subscribeNotfRespXlator:Translate: ygXpathInfo: ", ygXpathInfo)
		}

		// for subtree, path transformr can be present at any node level
		if (len(pathElem[idx].Key) == 0 || !path.HasWildcardAtKey(respXlator.ntfXlateReq.path, idx)) && len(ygXpathInfo.xfmrPath) == 0 {
			continue
		}

		if len(ygXpathInfo.xfmrPath) > 0 {
			err = respXlator.handlePathTransformer(ygXpathInfo, idx)
			if err != nil {
				return nil, err
			}
			err = respXlator.processDbToYangKeyXfmrList()
			if err != nil {
				return nil, err
			}
			return respXlator.ntfXlateReq.path, nil
		}
		if ygXpathInfo.virtualTbl != nil && (*ygXpathInfo.virtualTbl) {
			log.Warning(ntfXlateReq.reqLogId, "Translate: virtual table is set to true and path transformer not found list node path: ", *respXlator.ntfXlateReq.path)
			return nil, tlerr.InternalError{Format: ntfXlateReq.reqLogId + "virtual table is set to true and path transformer not found list node path", Path: ygPath}
		}
		if len(ygXpathInfo.xfmrFunc) == 0 && len(ygXpathInfo.xfmrKey) > 0 {
			dbYgXlateInfo := &DbYgXlateInfo{pathIdx: idx, ygXpathInfo: ygXpathInfo, xlateReq: respXlator.ntfXlateReq}
			dbYgXlateInfo.setUriPath()
			respXlator.dbYgXlateList = append(respXlator.dbYgXlateList, dbYgXlateInfo)
			// since there is no path transformer defined in the path, processing the collected db to yang key xfmrs
			if err = respXlator.processDbToYangKeyXfmrList(); err != nil {
				log.Warning(ntfXlateReq.reqLogId, "Translate: Error in processDbToYangKeyXfmrList for the path: ", *respXlator.ntfXlateReq.path)
				return nil, err
			}
		} else {
			if len(ygXpathInfo.xfmrFunc) > 0 {
				if log.V(dbLgLvl) {
					log.Warning(ntfXlateReq.reqLogId, "Translate: Could not find the path transformer for the xpath: ", ygPath)
				}
			} else if log.V(dbLgLvl) {
				log.Warning(ntfXlateReq.reqLogId, "Translate: Could not find the DbToYangKey transformer for the xpath: ", ygPath)
			}

			if log.V(dbLgLvl) {
				log.Warningf("%v Translate: Attempting direct conversion from db key %v to yang key %v directly"+
					" for the path: %v", ntfXlateReq.reqLogId, respXlator.ntfXlateReq.key.Comp, pathElem[idx].Key, ygPath)
			}
			dbKeyComp := respXlator.ntfXlateReq.key.Comp
			tblName := ""
			if ygXpathInfo.tableName != nil {
				tblName = *ygXpathInfo.tableName
			}

			if dbInfo, ok := xDbSpecMap[tblName]; !ok || dbInfo == nil {
				err = fmt.Errorf("error: direct conversion from db key to yang key is not supported for the"+
					" path %v, since there is no sonic yang model for this table %v; need path transformer to"+
					" translate key", ygXpathInfo.dbIndex, ygPath, tblName)
				log.Warning(ntfXlateReq.reqLogId, err)
				return nil, err
			}

			dbKeyRslvr := &DbYangKeyResolver{tableName: tblName, key: respXlator.ntfXlateReq.key,
				dbs: respXlator.ntfXlateReq.dbs, dbIdx: respXlator.ntfXlateReq.dbNum, uriPath: ygPath, reqLogId: respXlator.ntfXlateReq.reqLogId}
			dbKeyComp, err = dbKeyRslvr.resolve(GET)
			if err != nil {
				return nil, tlerr.InternalError{Format: respXlator.ntfXlateReq.reqLogId + "Translate: Error: " + err.Error(), Path: ygPath}
			}
			//yang key can be part of the db key, where db key is from child table db key
			if len(pathElem[idx].Key) > len(dbKeyComp) {
				log.Warning(ntfXlateReq.reqLogId, "Translate: Could not find the path transformer or DbToYangKey transformer for the ygXpathListInfo: ", ygPath)
				return nil, tlerr.InternalError{Format: ntfXlateReq.reqLogId + "Could not find the path transformer or DbToYangKey transformer", Path: ygPath}
			}
			dbYgListKeyNames, err := dbKeyRslvr.getMatchingDbYangListKeyNames(pathElem[idx].Key)
			if err != nil {
				return nil, tlerr.InternalError{Format: "Translate: Error: " + err.Error(), Path: ygPath}
			}
			dbKeyIdx := 0
			for _, dbKeyNm := range dbYgListKeyNames {
				pathElem[idx].Key[dbKeyNm] = dbKeyComp[dbKeyIdx]
				dbKeyIdx++
			}
		}
	}

	log.Info(ntfXlateReq.reqLogId, "subscribeNotfRespXlator: translated path: ", *respXlator.ntfXlateReq.path)
	return respXlator.ntfXlateReq.path, nil
}

func (respXlator *subscribeNotfRespXlator) handlePathTransformer(ygXpathInfo *yangXpathInfo, pathIdx int) error {
	var currPath gnmi.Path
	pathElems := respXlator.ntfXlateReq.path.Elem
	ygSchemPath := "/" + pathElems[0].Name
	currPath.Elem = append(currPath.Elem, pathElems[0])

	for idx := 1; idx <= pathIdx; idx++ {
		ygSchemPath = ygSchemPath + "/" + pathElems[idx].Name
		currPath.Elem = append(currPath.Elem, pathElems[idx])
	}

	inParam := XfmrDbToYgPathParams{
		yangPath:      &currPath,
		subscribePath: respXlator.ntfXlateReq.path,
		ygSchemaPath:  ygSchemPath,
		tblName:       respXlator.ntfXlateReq.table.Name,
		tblKeyComp:    respXlator.ntfXlateReq.key.Comp,
		tblEntry:      respXlator.ntfXlateReq.entry,
		dbNum:         respXlator.ntfXlateReq.dbNum,
		dbs:           respXlator.ntfXlateReq.dbs,
		db:            respXlator.ntfXlateReq.dbs[respXlator.ntfXlateReq.dbNum],
		ygPathKeys:    make(map[string]string),
	}

	if err := respXlator.xfmrPathHandlerFunc("DbToYangPath_"+ygXpathInfo.xfmrPath, inParam); err != nil {
		return fmt.Errorf(respXlator.ntfXlateReq.reqLogId+"Error in path transformer callback : %v for"+
			" the gnmi path: %v, and the error: %v", ygXpathInfo.xfmrPath, respXlator.ntfXlateReq.path, err)
	}

	if log.V(dbLgLvl) {
		log.Info(respXlator.ntfXlateReq.reqLogId, "handlePathTransformer: uriPathKeysMap: ", inParam.ygPathKeys)
	}
	ygpath := "/" + respXlator.ntfXlateReq.path.Elem[0].Name

	for idx := 1; idx <= pathIdx; idx++ {
		ygpath = ygpath + "/" + respXlator.ntfXlateReq.path.Elem[idx].Name

		if log.V(dbLgLvl) {
			log.Info(respXlator.ntfXlateReq.reqLogId, "handlePathTransformer: yang map keys: yang path:", ygpath)
		}

		for keyName, keyVal := range respXlator.ntfXlateReq.path.Elem[idx].Key {
			if keyVal != "*" {
				continue
			}
			if log.V(dbLgLvl) {
				log.Info(respXlator.ntfXlateReq.reqLogId, "handlePathTransformer: yang map keys: yang key path:", ygpath, "/", keyName)
			}
			ygKeyVal, ok := inParam.ygPathKeys[ygpath+"/"+keyName]
			if !ok {
				return fmt.Errorf(respXlator.ntfXlateReq.reqLogId+"Error: path transformer callback (%v)"+
					" response yang key map does not have the yang key value for the yang key: %v ",
					ygXpathInfo.xfmrPath, ygpath+"/"+keyName)
			}
			respXlator.ntfXlateReq.path.Elem[idx].Key[keyName] = ygKeyVal
		}
	}

	return nil
}

func (respXlator *subscribeNotfRespXlator) xfmrPathHandlerFunc(xfmrPathFunc string, inParam XfmrDbToYgPathParams) error {
	if log.V(dbLgLvl) {
		log.Infof(respXlator.ntfXlateReq.reqLogId+"Received inParam %v, Path transformer function name %v", inParam, xfmrPathFunc)
	}

	retVals, err := XlateFuncCall(xfmrPathFunc, inParam)
	if err != nil {
		return err
	}
	if retVals == nil || len(retVals) != PATH_XFMR_RET_ARGS {
		return tlerr.InternalError{Format: "incorrect return type in the transformer call back function", Path: inParam.yangPath.String()}
	}
	if retVals[PATH_XFMR_RET_ERR_INDX].Interface() != nil {
		if err = retVals[PATH_XFMR_RET_ERR_INDX].Interface().(error); err != nil {
			return err
		}
	}
	return nil
}

func (respXlator *subscribeNotfRespXlator) processDbToYangKeyXfmrList() error {
	for idx := len(respXlator.dbYgXlateList) - 1; idx >= 0; idx-- {
		if err := respXlator.dbYgXlateList[idx].handleDbToYangKeyXlate(); err != nil {
			log.Warningf(respXlator.ntfXlateReq.reqLogId+"handleDbToYangKeyXlate: Error: %v for the  ygPathTmp: %v ",
				err, respXlator.dbYgXlateList[idx].uriPath)
		}
	}
	return nil
}

func (respXlator *subscribeNotfRespXlator) getYangListPath(listIdx int) string {
	ygPathTmp := ""
	for idx := 0; idx <= listIdx; idx++ {
		pathName := respXlator.ntfXlateReq.path.Elem[idx].Name
		if idx > 0 {
			pathNames := strings.Split(respXlator.ntfXlateReq.path.Elem[idx].Name, ":")
			if len(pathNames) > 1 {
				pathName = pathNames[1]
			}
		}
		ygPathTmp = ygPathTmp + "/" + pathName
	}
	if log.V(dbLgLvl) {
		log.Infof(respXlator.ntfXlateReq.reqLogId+"getYangListPath: listIdx: %v, ygPathTmp: %v ", listIdx, ygPathTmp)
	}
	return ygPathTmp
}

func (dbYgXlateInfo *DbYgXlateInfo) setUriPath() {
	for idx := 0; idx <= dbYgXlateInfo.pathIdx; idx++ {
		dbYgXlateInfo.uriPath = dbYgXlateInfo.uriPath + "/" + dbYgXlateInfo.xlateReq.path.Elem[idx].Name
		for kn, kv := range dbYgXlateInfo.xlateReq.path.Elem[idx].Key {
			// not including the wildcard in the path; since it will be sent to db to yang key xfmr
			if kv == "*" {
				continue
			}
			dbYgXlateInfo.uriPath = dbYgXlateInfo.uriPath + "[" + kn + "=" + kv + "]"
		}
	}
}

func (respXlator *subscribeNotfRespXlator) getYangXpathInfo(ygPath string) (*yangXpathInfo, error) {
	ygXpathListInfo, ok := xYangSpecMap[ygPath]

	if !ok || ygXpathListInfo == nil {
		log.Warning(respXlator.ntfXlateReq.reqLogId, "ygXpathInfo data not found in the xYangSpecMap for xpath : ", ygPath)
		return nil, tlerr.InternalError{Format: respXlator.ntfXlateReq.reqLogId + "Error in processing the subscribe path", Path: ygPath}
	}
	if _, ygErr := getYgEntry(respXlator.ntfXlateReq.reqLogId, ygXpathListInfo, ygPath); ygErr != nil {
		return nil, tlerr.NotSupportedError{Format: respXlator.ntfXlateReq.reqLogId + "Subscribe not supported", Path: ygPath}
	}
	return ygXpathListInfo, nil
}

func (keyRslvr *DbYangKeyResolver) getMatchingDbYangListKeyNames(ygListKey map[string]string) ([]string, error) {
	ygDbInfo, _, err := keyRslvr.getDbYangNode()
	if err != nil {
		log.Warning(err)
		return nil, err
	}
	for _, listName := range ygDbInfo.listName {
		_, ygDbListNode, err := keyRslvr.getDbYangListInfo(listName)
		if err != nil {
			log.Warning(err)
			return nil, err
		}
		if !ygDbListNode.IsList() {
			if log.V(dbLgLvl) {
				log.Infof("%v DbYangKeyResolver: resolve: list name %v is not found in the xDbSpecMap as yang list node for the table: %v", keyRslvr.reqLogId, listName, keyRslvr.tableName)
			}
			continue
		}
		keyList := strings.Fields(ygDbListNode.Key)
		if log.V(dbLgLvl) {
			log.Info("keyList: ", keyList)
		}
		isMatch := true
		for _, kn := range keyList {
			if _, ok := ygListKey[kn]; !ok {
				isMatch = false
				break
			}
		}
		if !isMatch {
			continue
		}
		return keyList, nil
	}
	err = fmt.Errorf("DbYangKeyResolver: Db yang matching list node not found for the table %v for the path: %v", keyRslvr.tableName, keyRslvr.uriPath)
	log.Warning(err)
	return nil, err
}

func (dbYgXlateInfo *DbYgXlateInfo) handleDbToYangKeyXlate() error {
	if dbYgXlateInfo.ygXpathInfo.tableName != nil && *dbYgXlateInfo.ygXpathInfo.tableName != "NONE" {
		dbYgXlateInfo.tableName = *dbYgXlateInfo.ygXpathInfo.tableName
	} else if dbYgXlateInfo.ygXpathInfo.xfmrTbl == nil {
		return tlerr.InternalError{Format: dbYgXlateInfo.xlateReq.reqLogId + "Could not find the table information for the path", Path: dbYgXlateInfo.uriPath}
	}
	tblLst, err := dbYgXlateInfo.handleTableXfmrCallback()
	if err != nil {
		return fmt.Errorf("%v : Error: %v - in handleDbToYangKeyXlate; table name: %v",
			dbYgXlateInfo.xlateReq.reqLogId, err, *dbYgXlateInfo.ygXpathInfo.tableName)
	}
	if len(tblLst) == 0 {
		return fmt.Errorf("%v handleDbToYangKeyXlate: Error: No tables are returned by the table "+
			"transformer: for the path: %v", dbYgXlateInfo.xlateReq.reqLogId, dbYgXlateInfo.uriPath)
	}
	// taking the first table, since number of keys should be same between the tables returned by table transformer
	dbYgXlateInfo.tableName = tblLst[0]
	if log.V(dbLgLvl) {
		log.Info(dbYgXlateInfo.xlateReq.reqLogId, "handleDbToYangKeyXlate: Found table from the table transformer: table name: ", dbYgXlateInfo.tableName)
	}
	dbKeyRslvr := &DbYangKeyResolver{tableName: dbYgXlateInfo.tableName, key: dbYgXlateInfo.xlateReq.key,
		dbs: dbYgXlateInfo.xlateReq.dbs, dbIdx: dbYgXlateInfo.xlateReq.dbNum, uriPath: dbYgXlateInfo.uriPath, reqLogId: dbYgXlateInfo.xlateReq.reqLogId}
	dbTableKeyComp, err := dbKeyRslvr.resolve(GET)
	if err != nil {
		return tlerr.InternalError{Format: dbYgXlateInfo.xlateReq.reqLogId + "handleDbToYangKeyXlate: Error: " + err.Error() +
			"; tableName: " + dbYgXlateInfo.tableName, Path: dbYgXlateInfo.uriPath}
	}
	if len(dbTableKeyComp) > 0 {
		dbYgXlateInfo.dbKey = dbTableKeyComp[0]
		for idx := 1; idx < len(dbTableKeyComp); idx++ {
			dbYgXlateInfo.dbKey = dbYgXlateInfo.dbKey + dbKeyRslvr.delim + dbTableKeyComp[idx]
		}
	}
	return dbYgXlateInfo.handleDbToYangKeyXfmr()
}

func (dbYgXlateInfo *DbYgXlateInfo) handleDbToYangKeyXfmr() error {
	dbDataMap := make(RedisDbMap)
	for i := db.ApplDB; i < db.MaxDB; i++ {
		dbDataMap[i] = make(map[string]map[string]db.Value)
	}
	inParams := formXfmrInputRequest(dbYgXlateInfo.xlateReq.dbs[dbYgXlateInfo.xlateReq.dbNum], dbYgXlateInfo.xlateReq.dbs, dbYgXlateInfo.xlateReq.dbNum,
		nil, dbYgXlateInfo.uriPath, dbYgXlateInfo.uriPath, GET, dbYgXlateInfo.dbKey, &dbDataMap, nil, nil, dbYgXlateInfo.xlateReq.opaque)

	inParams.table = dbYgXlateInfo.tableName
	rmap, err := keyXfmrHandlerFunc(inParams, dbYgXlateInfo.ygXpathInfo.xfmrKey)
	if err != nil {
		return fmt.Errorf("%v handleDbToYangKeyXfmr: error in keyXfmrHandlerFunc: %v", dbYgXlateInfo.xlateReq.reqLogId, err)
	}
	if log.V(dbLgLvl) {
		log.Info(dbYgXlateInfo.xlateReq.reqLogId, "handleDbToYangKeyXfmr: res map: ", rmap)
	}
	for k, v := range rmap {
		//Assuming that always the string to be passed as the value in the DbtoYang key transformer response map
		dbYgXlateInfo.xlateReq.path.Elem[dbYgXlateInfo.pathIdx].Key[k] = fmt.Sprintf("%v", v)
	}

	return nil
}

func (dbYgXlateInfo *DbYgXlateInfo) handleTableXfmrCallback() ([]string, error) {
	ygXpathInfo := dbYgXlateInfo.ygXpathInfo
	uriPath := dbYgXlateInfo.uriPath

	if log.V(dbLgLvl) {
		log.Info(dbYgXlateInfo.xlateReq.reqLogId, "handleTableXfmrCallback: ", uriPath)
	}
	dbs := dbYgXlateInfo.xlateReq.dbs
	txCache := new(sync.Map)
	currDbNum := ygXpathInfo.dbIndex
	xfmrDbTblKeyCache := make(map[string]tblKeyCache)
	dbDataMap := make(RedisDbMap)
	for i := db.ApplDB; i < db.MaxDB; i++ {
		dbDataMap[i] = make(map[string]map[string]db.Value)
	}
	deviceObj := ocbinds.Device{}
	rootIntf := reflect.ValueOf(&deviceObj).Interface()
	ygotObj := rootIntf.(ygot.GoStruct)
	inParams := formXfmrInputRequest(dbs[ygXpathInfo.dbIndex], dbs, currDbNum, &ygotObj, uriPath,
		uriPath, SUBSCRIBE, "", &dbDataMap, nil, nil, txCache)
	tblList, tblXfmrErr := xfmrTblHandlerFunc(*ygXpathInfo.xfmrTbl, inParams, xfmrDbTblKeyCache)
	if tblXfmrErr != nil {
		log.Warningf("%v handleTableXfmrCallback: table transformer callback returns"+
			" error: %v for the callback %v", dbYgXlateInfo.xlateReq.reqLogId, tblXfmrErr, *ygXpathInfo.xfmrTbl)
	} else if inParams.isVirtualTbl != nil && *inParams.isVirtualTbl {
		if log.V(dbLgLvl) {
			log.Info(dbYgXlateInfo.xlateReq.reqLogId, "handleTableXfmrCallback: virtualTbl is SET to TRUE for this table transformer callback: ", *ygXpathInfo.xfmrTbl)
		}
	} else {
		if log.V(dbLgLvl) {
			log.Infof(dbYgXlateInfo.xlateReq.reqLogId+"handleTableXfmrCallback: table list %v returned by table transformer callback: %v ", tblList, *ygXpathInfo.xfmrTbl)
		}
		return tblList, nil
	}

	return nil, nil
}
