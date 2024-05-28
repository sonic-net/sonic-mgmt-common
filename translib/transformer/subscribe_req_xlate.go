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

	"strconv"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type subscribeReqXlator struct {
	subReq     *subscribeReq
	pathXlator *subscribePathXlator
	trgtYgNode *ygXpathNode
}

type subscribeReq struct {
	reqLogId         string
	reqUri           string
	ygPath           string
	isTrgtPathWldcrd bool
	gPath            *gnmipb.Path
	txCache          interface{}
	dbs              [db.MaxDB]*db.DB
	tblKeyCache      map[string]tblKeyCache
	subReqMode       NotificationType
	xlateNodeType    xlateNodeType
	subReqXlateInfo  *XfmrSubscribeReqXlateInfo
}

type subscribePathXlator struct {
	gPath              *gnmipb.Path
	pathXlateInfo      *XfmrSubscribePathXlateInfo
	keyXfmrYgXpathInfo *yangXpathInfo
	uriPath            string
	subReq             *subscribeReq
	parentXlateInfo    *XfmrSubscribePathXlateInfo
	xpathYgNode        *ygXpathNode
}

type DbYangKeyResolver struct {
	tableName string
	key       *db.Key
	dbs       [db.MaxDB]*db.DB
	uriPath   string
	reqLogId  string
	dbIdx     db.DBNum
	delim     string
}

var dbLgLvl log.Level = 4 // debug log level

type xlateNodeType int

const (
	TARGET_NODE xlateNodeType = 1 + iota
	CHILD_NODE
)

type dbTableKeyInfo struct {
	Table          *db.TableSpec // table to be subscribed
	Key            *db.Key       // specific key entry of the table to be subscribed
	DbNum          db.DBNum      // database index
	DbFldYgMapList []*DbFldYgPathInfo
	DeleteAction   apis.DeleteActionType // to indicate how db delete needs to be handled w.r.t a path.
	FieldScanPatt  string                // field name match pattern
	KeyGroupComps  []int                 // key component indices that make up the key group (for leaf-list)
}

type DbFldYgPathInfo struct {
	RltvPath       string
	DbFldYgPathMap map[string]string //db field to leaf / rel. path to leaf
}

type XfmrSubscribePathXlateInfo struct {
	Path             *gnmipb.Path // subscribe path
	ygXpathInfo      *yangXpathInfo
	DbKeyXlateInfo   []*dbTableKeyInfo
	MinInterval      int // min interval
	NeedCache        bool
	PType            NotificationType
	OnChange         OnchangeMode
	TrgtNodeChld     bool // to indicate the immediate child level pathXlate info of the target node
	reqLogId         string
	chldXlateInfos   []*XfmrSubscribePathXlateInfo
	HandlerFunc      apis.ProcessOnChange // if its true, stop traversing the ygXpathNode's child nodes further
	IsDataSrcDynamic bool
}

type XfmrSubscribeReqXlateInfo struct {
	TrgtPathInfo    *XfmrSubscribePathXlateInfo
	ChldPathsInfo   []*XfmrSubscribePathXlateInfo
	ygXpathTrgtInfo *yangXpathInfo
}

func (reqXlator *subscribeReqXlator) getSubscribePathXlator(gPath *gnmipb.Path, uriPath string, ygXpathInfo *yangXpathInfo,
	parentXlateInfo *XfmrSubscribePathXlateInfo, xpathYgNode *ygXpathNode) (*subscribePathXlator, error) {
	var err error
	reqXlator.pathXlator.gPath = gPath
	reqXlator.pathXlator.pathXlateInfo = &(XfmrSubscribePathXlateInfo{Path: gPath, ygXpathInfo: ygXpathInfo, reqLogId: reqXlator.subReq.reqLogId})
	reqXlator.pathXlator.pathXlateInfo.chldXlateInfos = make([]*XfmrSubscribePathXlateInfo, 0)
	reqXlator.pathXlator.pathXlateInfo.OnChange = OnchangeDefault
	if ygXpathInfo.subscriptionFlags.Has(subsOnChangeDisable) {
		reqXlator.pathXlator.pathXlateInfo.OnChange = OnchangeDisable
	} else if ygXpathInfo.subscriptionFlags.Has(subsOnChangeEnable) {
		reqXlator.pathXlator.pathXlateInfo.OnChange = OnchangeEnable
	}
	reqXlator.pathXlator.uriPath = uriPath
	reqXlator.pathXlator.keyXfmrYgXpathInfo = ygXpathInfo
	reqXlator.pathXlator.parentXlateInfo = parentXlateInfo
	reqXlator.pathXlator.xpathYgNode = xpathYgNode
	if reqXlator.subReq.xlateNodeType == TARGET_NODE {
		if err = reqXlator.pathXlator.setTrgtYgXpathInfo(); err != nil {
			if log.V(dbLgLvl) {
				log.Warning(reqXlator.subReq.reqLogId, "Error in setting the ygXpathInfo of the last LIST node in the path and the error is :", err)
			}
			return nil, err
		}
	}
	return reqXlator.pathXlator, err
}

func (pathXltr *subscribePathXlator) setTrgtYgXpathInfo() error {
	if log.V(dbLgLvl) {
		log.Infof("%v Entering into the setTrgtListYgXpathInfo: ygPath: %v", pathXltr.subReq.reqLogId, pathXltr.subReq.ygPath)
	}

	ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo
	pathXltr.keyXfmrYgXpathInfo = ygXpathInfo
	ygPathTmp := pathXltr.subReq.ygPath

	for ygXpathInfo != nil && len(ygXpathInfo.xfmrKey) == 0 {
		tIdx := strings.LastIndex(ygPathTmp, "/")
		// -1: not found, and 0: first character in the path
		if tIdx <= 0 {
			break
		}
		ygPathTmp = ygPathTmp[0:tIdx]
		if log.V(dbLgLvl) {
			log.Infof("%v xPathTmp: %v", pathXltr.subReq.reqLogId, ygPathTmp)
		}
		ygXpathInfoTmp, ok := xYangSpecMap[ygPathTmp]
		if !ok || ygXpathInfoTmp == nil {
			log.Warningf("%v xYangSpecMap does not have the yangXpathInfo for the path: %v", pathXltr.subReq.reqLogId, ygPathTmp)
			return tlerr.InternalError{Format: "xYangSpecMap does not have the yangXpathInfo", Path: ygPathTmp}
		}
		if _, ygErr := getYgEntry(pathXltr.subReq.reqLogId, ygXpathInfoTmp, ygPathTmp); ygErr != nil {
			return ygErr
		}
		ygXpathInfo = ygXpathInfoTmp
	}
	if ygXpathInfo != nil && len(ygXpathInfo.xfmrKey) > 0 {
		pathXltr.keyXfmrYgXpathInfo = ygXpathInfo
	}
	return nil
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) addPathXlateInfo(tblSpec *db.TableSpec, dbKey *db.Key, dBNum db.DBNum) *dbTableKeyInfo {
	dbTblInfo := dbTableKeyInfo{Table: tblSpec, Key: dbKey, DbNum: dBNum}
	pathXlateInfo.DbKeyXlateInfo = append(pathXlateInfo.DbKeyXlateInfo, &dbTblInfo)
	if (pathXlateInfo.ygXpathInfo.yangType == YANG_LEAF || pathXlateInfo.ygXpathInfo.yangType == YANG_LEAF_LIST) &&
		pathXlateInfo.ygXpathInfo.subscriptionFlags.Has(subsDelAsUpdate) {
		dbTblInfo.DeleteAction = apis.InspectPathOnDelete
	}
	return &dbTblInfo
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) hasDbTableInfo() bool {
	return len(pathXlateInfo.DbKeyXlateInfo) > 0
}

func NewSubscribeReqXlator(subReqId interface{}, reqUri string, reqMode NotificationType, dbs [db.MaxDB]*db.DB, txCache interface{}) (*subscribeReqXlator, error) {
	reqIdLogStr := "subReq Id:[" + fmt.Sprintf("%v", subReqId) + "] : "
	if log.V(dbLgLvl) {
		log.Infof("%v NewSubscribeReqXlator: for the reqUri: %v; reqMode: %v; txCache: %v", reqIdLogStr, reqUri, reqMode, txCache)
	}
	subReq := subscribeReq{reqLogId: reqIdLogStr, reqUri: reqUri, dbs: dbs, txCache: txCache, isTrgtPathWldcrd: true}
	subReq.tblKeyCache = make(map[string]tblKeyCache)
	subReq.subReqMode = reqMode
	subReq.xlateNodeType = TARGET_NODE
	var err error

	if subReq.ygPath, _, err = XfmrRemoveXPATHPredicates(reqUri); err != nil {
		log.Warningf("%v Received error: %v from the XfmrRemoveXPATHPredicates function: ", subReq.reqLogId, err)
		return nil, err
	}

	if subReq.gPath, err = ygot.StringToPath(reqUri, ygot.StructuredPath); err != nil {
		log.Warningf("%v Error in converting the URI into GNMI path for the URI: %v", subReq.reqLogId, reqUri)
		return nil, tlerr.InternalError{Format: "Error in converting the URI into GNMI path", Path: reqUri}
	}

	subReqXlator := subscribeReqXlator{subReq: &subReq}
	subReqXlator.subReq.subReqXlateInfo = new(XfmrSubscribeReqXlateInfo)
	subReqXlator.pathXlator = &subscribePathXlator{subReq: &subReq}
	return &subReqXlator, nil
}

func (reqXlator *subscribeReqXlator) processSubscribePath() error {
	if log.V(dbLgLvl) {
		log.Infof("%v processSubscribePath: path: %v", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri)
	}
	pathElems := reqXlator.subReq.gPath.Elem
	pathIdx := len(pathElems) - 1
	ygXpathInfoTmp := reqXlator.subReq.subReqXlateInfo.ygXpathTrgtInfo
	ygEntryTmp, ygErr := getYgEntry(reqXlator.subReq.reqLogId, ygXpathInfoTmp, reqXlator.subReq.ygPath)
	if ygErr != nil {
		return ygErr
	}
	ygPathTmp := reqXlator.subReq.ygPath
	isUriMdfd := false
	for {
		if ygEntryTmp.Parent != nil && ygXpathInfoTmp.nameWithMod != nil {
			if log.V(dbLgLvl) {
				log.Infof("%v parent node prefix: %v and curr. node prefix: %v", reqXlator.subReq.reqLogId, ygEntryTmp.Parent.Prefix.Name,
					ygEntryTmp.Prefix.Name)
			}
			if strings.HasSuffix(*ygXpathInfoTmp.nameWithMod, ":"+pathElems[pathIdx].Name) &&
				ygEntryTmp.Parent.Prefix.Name != ygEntryTmp.Prefix.Name {
				if log.V(dbLgLvl) {
					log.Infof("%v module prefix is missing in the path: adding the same: mod prefix: %v, "+
						"input path name: %v", reqXlator.subReq.reqLogId, *ygXpathInfoTmp.nameWithMod, pathElems[pathIdx].Name)
				}
				pathElems[pathIdx].Name = *ygXpathInfoTmp.nameWithMod
				isUriMdfd = true
			}
		}

		if ygEntryTmp.IsList() && len(pathElems[pathIdx].Key) == 0 {
			isUriMdfd = true
			for _, listKey := range strings.Fields(ygEntryTmp.Key) {
				pathElems[pathIdx].Key[listKey] = "*"
			}
			if log.V(dbLgLvl) {
				log.Infof("%v processSubscribePath: list node doesn not have keys in the input path;"+
					" added wildcard to the list node path: %v", reqXlator.subReq.reqLogId, reqXlator.subReq.ygPath)
			}
		} else if reqXlator.subReq.isTrgtPathWldcrd {
			for _, kv := range pathElems[pathIdx].Key {
				if kv == "*" {
					continue
				}
				reqXlator.subReq.isTrgtPathWldcrd = false
				break
			}
		}

		tIdx := strings.LastIndex(ygPathTmp, "/")
		if tIdx <= 0 {
			break
		}
		ygPathTmp = ygPathTmp[0:tIdx]
		pathIdx--
		if log.V(dbLgLvl) {
			log.Infof("%v processSubscribePath: xPathTmp: %v", reqXlator.subReq.reqLogId, ygPathTmp)
		}
		var ok bool
		ygXpathInfoTmp, ok = xYangSpecMap[ygPathTmp]
		if ok && ygXpathInfoTmp != nil {
			var ygErr error
			if ygEntryTmp, ygErr = getYgEntry(reqXlator.subReq.reqLogId, ygXpathInfoTmp, ygPathTmp); ygErr != nil {
				return ygErr
			}
		}
		if !ok || ygXpathInfoTmp == nil || ygEntryTmp == nil {
			log.Warning(reqXlator.subReq.reqLogId, "processSubscribePath: xYangSpecMap does not have the yangXpathInfo for the path:", ygPathTmp)
			return tlerr.InternalError{Format: "xYangSpecMap does not have the yangXpathInfo", Path: ygPathTmp}
		}
	}

	xPathInfoTrgt := reqXlator.subReq.subReqXlateInfo.ygXpathTrgtInfo
	var err error
	if isUriMdfd {
		if reqXlator.subReq.reqUri, err = ygot.PathToString(reqXlator.subReq.gPath); err != nil {
			return tlerr.InvalidArgsError{Format: "Error in converting the GNMI path to URI string.", Path: reqXlator.subReq.gPath.String()}
		}
		if log.V(dbLgLvl) {
			log.Infof("%v GNMI path got modified by adding the missing module name prefix or wildcard keys in the path, and the modified URI: %v", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri)
		}
	}

	if !reqXlator.validateYangPath(reqXlator.subReq.reqUri, xPathInfoTrgt) {
		log.Warningf("%v URI path %v is not valid since validate callback function '%v' returned false for this path: ", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri, xPathInfoTrgt.validateFunc)
		return tlerr.NotSupportedError{Format: "URI path is not valid", Path: reqXlator.subReq.reqUri}
	}

	return nil
}

func (reqXlator *subscribeReqXlator) Translate(targetOnly bool) error {

	if reqXlator == nil {
		log.Warningf("Translate: subscribeReqXlator is nil")
		return tlerr.InternalError{Format: "subscribeReqXlator is nil"}
	}

	if log.V(dbLgLvl) {
		log.Infof("%v Translate: reqXlator: ", reqXlator.subReq.reqLogId, *reqXlator.subReq)
	}

	var err error
	ygXpathInfoTrgt, ok := xYangSpecMap[reqXlator.subReq.ygPath]
	if !ok || ygXpathInfoTrgt == nil {
		log.Warningf("%v Translate: ygXpathInfo data not found in the xYangSpecMap for xpath : %v", reqXlator.subReq.reqLogId, reqXlator.subReq.ygPath)
		return tlerr.InternalError{Format: "Error in processing the subscribe path", Path: reqXlator.subReq.reqUri}
	}
	if _, ygErr := getYgEntry(reqXlator.subReq.reqLogId, ygXpathInfoTrgt, reqXlator.subReq.ygPath); ygErr != nil {
		return ygErr
	}
	reqXlator.subReq.subReqXlateInfo.ygXpathTrgtInfo = ygXpathInfoTrgt
	if err = reqXlator.processSubscribePath(); err != nil {
		log.Warningf("%v Translate: Error in processSubscribePath for the path: %v, and the error is %v", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri, err)
		return tlerr.NotSupportedError{Format: "Error in processing the subscribe path", Path: reqXlator.subReq.reqUri}
	}

	if err = reqXlator.translateTargetNodePath(ygXpathInfoTrgt); err != nil {
		log.Warning(reqXlator.subReq.reqLogId, "Translate: Error in translating the target node subscribe path: ", err)
		return err
	}
	if !targetOnly {
		if err = reqXlator.translateChildNodePaths(ygXpathInfoTrgt); err != nil {
			log.Warning(reqXlator.subReq.reqLogId, "Translate: Error in translating the child node for the subscribe path: ", err)
			return err
		}
	}

	trgtPathXlateInfo := reqXlator.subReq.subReqXlateInfo.TrgtPathInfo
	if trgtPathXlateInfo.PType == Sample && trgtPathXlateInfo.hasDbTableInfo() {
		if reqXlator.trgtYgNode != nil && trgtPathXlateInfo.MinInterval < reqXlator.trgtYgNode.subMinIntrvl {
			trgtPathXlateInfo.MinInterval = reqXlator.trgtYgNode.subMinIntrvl
		} else if trgtPathXlateInfo.MinInterval == 0 {
			trgtPathXlateInfo.MinInterval = apis.SAMPLE_NOTIFICATION_MIN_INTERVAL // default value; in seconds
		}
		if log.V(dbLgLvl) {
			log.Infof("%v Translate: requested target path xlate info: Min Interval: %v for the path: %v", reqXlator.subReq.reqLogId, trgtPathXlateInfo.MinInterval, reqXlator.subReq.reqUri)
		}
	}

	return err
}

func (reqXlator *subscribeReqXlator) translateTargetNodePath(trgtYgxPath *yangXpathInfo) error {
	trgtPathXlator, err := reqXlator.getSubscribePathXlator(reqXlator.subReq.gPath, reqXlator.subReq.reqUri,
		trgtYgxPath, nil, &ygXpathNode{ygXpathInfo: trgtYgxPath, ygPath: reqXlator.subReq.ygPath,
			dbFldYgPathMap: make(map[string]string), dbTblFldYgPathMap: make(map[string]map[string]string)})
	if err != nil {
		log.Warning(reqXlator.subReq.reqLogId, "Error in getSubscribePathXlator: error: ", err)
		return err
	}
	subMode := trgtPathXlator.getSubscribeMode(nil)
	if trgtPathXlator.pathXlateInfo.OnChange == OnchangeDisable && subMode == OnChange {
		log.Warningf("%v translateTargetNodePath:Subscribe not supported; on change disabled for the given subscribe path:: %v", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri)
		return tlerr.NotSupportedError{Format: "Subscribe not supported; on change disabled for the given subscribe path: ", Path: reqXlator.subReq.reqUri}
	}

	reqXlator.subReq.subReqXlateInfo.TrgtPathInfo = trgtPathXlator.pathXlateInfo
	if err = trgtPathXlator.translatePath(); err != nil {
		if log.V(dbLgLvl) {
			log.Warning(reqXlator.subReq.reqLogId, "Error: in translateTargetNodePath: error: ", err)
		}
		reqXlator.subReq.subReqXlateInfo.TrgtPathInfo = nil
		return err
	}
	// for leaf node which are mapped to specific tables different than parent
	for _, chldPathXlateInfo := range reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.chldXlateInfos {
		if !reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.hasDbTableInfo() {
			chldPathXlateInfo.TrgtNodeChld = true
		}
		reqXlator.subReq.subReqXlateInfo.ChldPathsInfo = append(reqXlator.subReq.subReqXlateInfo.ChldPathsInfo, chldPathXlateInfo)
	}
	return nil
}

func (pathXltr *subscribePathXlator) getSubscribeMode(xfmrSubsOutParam *XfmrSubscOutParams) NotificationType {
	if pathXltr.subReq.subReqMode != TargetDefined {
		return pathXltr.subReq.subReqMode
	}

	if pathXltr.parentXlateInfo != nil &&
		(pathXltr.parentXlateInfo.PType == Sample && len(pathXltr.parentXlateInfo.DbKeyXlateInfo) > 0) {
		return Sample
	}
	subParamNtfType := pathXltr.pathXlateInfo.PType
	if xfmrSubsOutParam != nil {
		if xfmrSubsOutParam.nOpts != nil {
			subParamNtfType = xfmrSubsOutParam.nOpts.pType
		} else if xfmrSubsOutParam.isVirtualTbl {
			subParamNtfType = Sample
		}
	}

	if pathXltr.pathXlateInfo.ygXpathInfo.subscriptionFlags.Has(subsPrefSample) || subParamNtfType == Sample || pathXltr.pathXlateInfo.OnChange == OnchangeDisable {
		return Sample
	}
	return OnChange
}

func (pathXltr *subscribePathXlator) getKeyCompCnt(tblFld map[string]string) (int, error) {
	if len(tblFld) > 0 {
		ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo
		keyCmpCnt := 0
		if cmpCntVal, ok := tblFld[KEY_COMP_CNT]; ok {
			var err error
			if keyCmpCnt, err = strconv.Atoi(cmpCntVal); err != nil {
				log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:getKeyCompCnt: Invalid value: ", cmpCntVal, " mentioned for the attribute: ", KEY_COMP_CNT, " in the subscribe function: ", ygXpathInfo.xfmrFunc)
				return 0, tlerr.InternalError{Format: "Invalid value: " + cmpCntVal + " mentioned for the attribute: " + KEY_COMP_CNT + " in the subscribe function: " + ygXpathInfo.xfmrFunc, Path: pathXltr.uriPath}
			}
			if log.V(dbLgLvl) {
				log.Infof("%v handleSubtreeNodeXlate:getKeyCompCnt: %v in the subscribe function: %v for the path : %v", pathXltr.subReq.reqLogId, keyCmpCnt, ygXpathInfo.xfmrFunc, pathXltr.uriPath)
			}
			delete(tblFld, KEY_COMP_CNT)
			return keyCmpCnt, nil
		}
	}
	return 0, nil
}

func (pathXltr *subscribePathXlator) getKeyComp(dbKey string, keyCmpCnt int, dbNum db.DBNum) []string {
	var keyComp []string
	if keyCmpCnt > 0 {
		keyComp = strings.SplitN(dbKey, pathXltr.subReq.dbs[dbNum].Opts.KeySeparator, keyCmpCnt)
	} else {
		keyComp = strings.Split(dbKey, pathXltr.subReq.dbs[dbNum].Opts.KeySeparator)
	}
	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "getKeyComp: keyComp: ", keyComp)
	}
	return keyComp
}

func (pathXltr *subscribePathXlator) handleSubtreeNodeXlate() error {
	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: reqUri: ", pathXltr.uriPath)
	}
	ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo

	uriSubtree := pathXltr.uriPath
	ygEntry, ygErr := getYgEntry(pathXltr.subReq.reqLogId, ygXpathInfo, pathXltr.getYgPath())
	if ygErr != nil {
		return ygErr
	}
	if ygEntry.IsLeaf() || ygEntry.IsLeafList() {
		idxS := strings.LastIndex(uriSubtree, "/")
		uriSubtree = uriSubtree[:idxS]
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: trimmed uriSubtree: ", uriSubtree)
		}
	}

	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: handleSubtreeNodeXlate: uriSubtree: ", uriSubtree)
	}

	subInParam := XfmrSubscInParams{uriSubtree, pathXltr.subReq.dbs, make(RedisDbMap), TRANSLATE_SUBSCRIBE}
	subOutPram, subErr := xfmrSubscSubtreeHandler(subInParam, ygXpathInfo.xfmrFunc)

	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: subOutPram:  ", subOutPram)
	}
	if subErr != nil {
		if log.V(dbLgLvl) {
			log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: Got error form the Subscribe transformer callback ", subErr)
		}
		return subErr
	}

	if subOutPram.onChange != OnchangeDefault {
		pathXltr.pathXlateInfo.OnChange = subOutPram.onChange
	}

	ntfType := NotificationType(-1)

	if subOutPram.nOpts != nil {
		ntfType = subOutPram.nOpts.pType
	}

	subNotfMode := pathXltr.getSubscribeMode(&subOutPram)
	pathXltr.pathXlateInfo.PType = subNotfMode

	if !subOutPram.isVirtualTbl && pathXltr.pathXlateInfo.OnChange == OnchangeDisable && subNotfMode == OnChange {
		log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: Onchange subscription is not supported; onChange flag set to"+
			" false in the XfmrSubscOutParams for the Subscribe transformer callback: ", ygXpathInfo.xfmrFunc)
		return tlerr.InternalError{Format: "Onchange subscription is not supported; onChange flag set to false in the" +
			" XfmrSubscOutParams for the Subscribe transformer callback", Path: pathXltr.uriPath}
	}

	if subNotfMode == Sample {
		pathXltr.pathXlateInfo.OnChange = OnchangeDisable
		if subOutPram.nOpts != nil && pathXltr.pathXlateInfo.MinInterval < subOutPram.nOpts.mInterval {
			pathXltr.pathXlateInfo.MinInterval = subOutPram.nOpts.mInterval
		}
	}

	if subOutPram.isVirtualTbl {
		log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: isVirtualTbl flag is set to true for the path: ", pathXltr.uriPath)
		return nil
	}

	isTrgtNodeLeaf := (ygEntry.IsLeaf() || ygEntry.IsLeafList())
	isLeafTblFound := false
	ygLeafNodePathPrefix := pathXltr.subReq.ygPath // default target node path
	if pathXltr.xpathYgNode != nil {
		ygLeafNodePathPrefix = pathXltr.xpathYgNode.ygPath
	}
	ygLeafNodeSecDbMap := make(map[string]bool) // yang leaf/leaf-list node name as key
	for dbNum, tblKeyInfo := range subOutPram.secDbDataMap {
		if isLeafTblFound {
			break
		} // do not process any more entry if the target node is leaf and it is processed
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: secDbDataMap: dbNum: ", dbNum)
		}
		if dbNum == db.CountersDB && (pathXltr.subReq.subReqMode == OnChange && pathXltr.pathXlateInfo.OnChange != OnchangeEnable) {
			log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: Onchange subscription is not supported for COUNTERS_DB by default for the path: ", pathXltr.uriPath)
			return tlerr.InternalError{Format: "Onchange subscription is not supported for COUNTERS_DB by default.", Path: pathXltr.uriPath}
		}
		for tblName, tblFieldInfo := range tblKeyInfo {
			if isLeafTblFound {
				break
			} // do not process any more entry if the target node is leaf and it is processed
			if log.V(dbLgLvl) {
				log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: secDbDataMap: tblName: ", tblName)
			}
			tblSpec := &db.TableSpec{Name: tblName}
			for dBKey, nodeIntf := range tblFieldInfo {
				if isLeafTblFound {
					break
				} // do not process any more entry if the target node is leaf and it is processed

				keyComp := pathXltr.getKeyComp(dBKey, tblSpec.CompCt, dbNum)
				if log.V(dbLgLvl) {
					log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: secDbDataMap: keyComp: ", keyComp, "; dBKey: ", dBKey)
				}

				dbFldYgNameMap := make(map[string]string) // map of yang leaf/leaf-list name as key and db table field name as value
				var keyGroupComps []int

				switch intfVal := nodeIntf.(type) {
				case string:
					// yang leaf/leaf-list node mapped to db table key, not db table field
					// because of this, the db field is empty, and keeping yang name as db field name in this map.
					dbFldYgNameMap[intfVal] = intfVal
				case map[string]string:
					dbFldYgNameMap = intfVal
					if keyCmpCnt, err := pathXltr.getKeyCompCnt(dbFldYgNameMap); err != nil {
						return err
					} else {
						tblSpec.CompCt = keyCmpCnt
					}
				case DBKeyYgNodeInfo:
					if len(intfVal.nodeName) != 0 {
						dbFldYgNameMap[intfVal.nodeName] = intfVal.nodeName
					}
					if len(intfVal.keyGroup) != 0 {
						keyGroupComps = intfVal.keyGroup
					}
					if intfVal.keyCompCt > 0 {
						tblSpec.CompCt = intfVal.keyCompCt
					}
					if intfVal.onChangeFunc != nil {
						pathXltr.pathXlateInfo.HandlerFunc = intfVal.onChangeFunc
						tkInfo := pathXltr.pathXlateInfo.addPathXlateInfo(tblSpec, &db.Key{keyComp}, dbNum)
						tkInfo.KeyGroupComps = keyGroupComps
						continue
					}
				case apis.ProcessOnChange:
					pathXltr.pathXlateInfo.HandlerFunc = intfVal
					pathXltr.pathXlateInfo.addPathXlateInfo(tblSpec, &db.Key{keyComp}, dbNum)
					continue
				default:
					log.Warningf("%s handleSubtreeNodeXlate: Subscribe callback '%v' returned incorrect type: %T",
						pathXltr.subReq.reqLogId, ygXpathInfo.xfmrFunc, intfVal)
					return tlerr.InternalError{Format: "Onchange subscription: Incorrect type received from the Subscribe transformer callback", Path: pathXltr.uriPath}
				}

				isDelUpdate := false
				if v, ok := dbFldYgNameMap[DEL_AS_UPDATE]; ok {
					isDelUpdate = (v == "true")
					delete(dbFldYgNameMap, DEL_AS_UPDATE)
				}

				for dbField, yangNodeName := range dbFldYgNameMap {
					if isLeafTblFound {
						break
					} // do not process any more entry if the target node is leaf and it is processed
					ygLeafNodePath := ygLeafNodePathPrefix
					if isTrgtNodeLeaf {
						if yangNodeName != ygEntry.Name {
							continue
						}
						isLeafTblFound = true // only if the subscribe path target is leaf/leaf-list
					} else {
						ygLeafNodePath = ygLeafNodePathPrefix + "/" + yangNodeName
					}

					if log.V(dbLgLvl) {
						log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: ygLeafNodePath: ", ygLeafNodePath)
					}
					yangNodeNameWithMod := yangNodeName

					ygLeafXpathInfo, okLeaf := xYangSpecMap[ygLeafNodePath]
					if ygLeafXpathInfo == nil {
						log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: XpathInfo not found for the leaf path: ", ygLeafNodePath)
						return tlerr.InternalError{Format: "XpathInfo not found for the leaf path", Path: ygLeafNodePath}
					}
					if okLeaf && ygLeafXpathInfo.nameWithMod != nil {
						yangNodeNameWithMod = *(ygLeafXpathInfo.nameWithMod)
					}
					ygLeafEntry, ygErr := getYgEntry(pathXltr.subReq.reqLogId, ygLeafXpathInfo, ygLeafNodePath)
					if ygErr != nil {
						return ygErr
					}
					if log.V(dbLgLvl) {
						log.Infof("%v handleSubtreeNodeXlate: secDbDataMap: uripath: %v; "+
							"KeySeparator: %v ", pathXltr.subReq.reqLogId, pathXltr.uriPath+"/"+yangNodeNameWithMod, pathXltr.subReq.dbs[dbNum].Opts.KeySeparator)
					}
					leafPath, err := ygot.StringToPath(pathXltr.uriPath+"/"+yangNodeNameWithMod, ygot.StructuredPath)
					if err != nil {
						log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: error in StringToPath: err: ", err)
						return err
					}
					if log.V(dbLgLvl) {
						log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:secDbDataMap: leafPath", leafPath)
					}
					ygLeafNodeSecDbMap[yangNodeName] = true
					if isLeafTblFound {
						// target node
						pathXltr.pathXlateInfo.PType = subNotfMode
						dbTblInfo := pathXltr.pathXlateInfo.addPathXlateInfo(tblSpec, &db.Key{keyComp}, dbNum)
						if log.V(dbLgLvl) {
							log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:secDbDataMap: path is leaf / leaf-list", pathXltr.uriPath)
						}
						dbYgPath := DbFldYgPathInfo{"", make(map[string]string)}
						dbYgPath.DbFldYgPathMap[dbField] = ""
						dbTblInfo.DbFldYgMapList = append(dbTblInfo.DbFldYgMapList, &dbYgPath)
						dbTblInfo.KeyGroupComps = keyGroupComps
						if ygEntry.IsLeafList() || isDelUpdate {
							dbTblInfo.DeleteAction = apis.InspectLeafOnDelete
						}

						if log.V(dbLgLvl) {
							log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: secDbDataMap: target node: leaf: "+
								"Adding special entry for leaf node mapped to table for the uri path: ", pathXltr.uriPath+"/"+yangNodeNameWithMod)
						}
					} else {
						leafPathXlateInfo := &XfmrSubscribePathXlateInfo{Path: leafPath, PType: ntfType, OnChange: pathXltr.pathXlateInfo.OnChange, ygXpathInfo: ygLeafXpathInfo}
						leafPathXlateInfo.chldXlateInfos = make([]*XfmrSubscribePathXlateInfo, 0)
						dbTblInfo := leafPathXlateInfo.addPathXlateInfo(tblSpec, &db.Key{keyComp}, dbNum)
						if (ygLeafEntry != nil && ygLeafEntry.IsLeafList()) || isDelUpdate {
							dbTblInfo.DeleteAction = apis.InspectLeafOnDelete
						}
						if log.V(dbLgLvl) {
							log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:secDbDataMap: mapped to leaf / leaf-list", pathXltr.uriPath)
						}
						dbYgPath := DbFldYgPathInfo{"", make(map[string]string)}
						dbYgPath.DbFldYgPathMap[dbField] = ""
						dbTblInfo.DbFldYgMapList = append(dbTblInfo.DbFldYgMapList, &dbYgPath)
						dbTblInfo.KeyGroupComps = keyGroupComps
						leafPathXlateInfo.PType = subNotfMode
						if subNotfMode == Sample {
							leafPathXlateInfo.MinInterval = ygLeafXpathInfo.subscribeMinIntvl
							if leafPathXlateInfo.MinInterval < pathXltr.pathXlateInfo.MinInterval {
								leafPathXlateInfo.MinInterval = pathXltr.pathXlateInfo.MinInterval
							} else if leafPathXlateInfo.MinInterval == 0 {
								leafPathXlateInfo.MinInterval = apis.SAMPLE_NOTIFICATION_MIN_INTERVAL
							}
						}
						if log.V(dbLgLvl) {
							log.Infof("%v handleSubtreeNodeXlate: secDbDataMap: Adding child pathXlateInfo %v in the current pathXlateInfo %v for"+
								" leaf node mapped to table for the uri path: %v", pathXltr.subReq.reqLogId, *leafPathXlateInfo,
								pathXltr.pathXlateInfo.Path, pathXltr.uriPath+"/"+yangNodeNameWithMod)
						}
						pathXltr.pathXlateInfo.chldXlateInfos = append(pathXltr.pathXlateInfo.chldXlateInfos, leafPathXlateInfo)
					}
				}
			}
		}
	}

	if !isLeafTblFound {
		for dbNum, tblKeyInfo := range subOutPram.dbDataMap {
			if log.V(dbLgLvl) {
				log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:  dbNum: ", dbNum)
			}
			if dbNum == db.CountersDB && (pathXltr.subReq.subReqMode == OnChange && pathXltr.pathXlateInfo.OnChange != OnchangeEnable) {
				log.Warning(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: Onchange subscription is not supported for COUNTERS_DB by default for the path: ", pathXltr.uriPath)
				return tlerr.InternalError{Format: "Onchange subscription is not supported for COUNTERS_DB by default.", Path: pathXltr.uriPath}
			}
			for tblName, tblFieldInfo := range tblKeyInfo {
				if log.V(dbLgLvl) {
					log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: tblName: ", tblName)
				}
				tblSpec := &db.TableSpec{Name: tblName}
				for dBKey, tblFld := range tblFieldInfo {
					if log.V(dbLgLvl) {
						log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: pathXltr.subReq.dbs[dbNum].Opts.KeySeparator: ", pathXltr.subReq.dbs[dbNum].Opts.KeySeparator)
					}
					keyCmpCnt, err := pathXltr.getKeyCompCnt(tblFld)
					if err != nil {
						return err
					}
					tblSpec.CompCt = keyCmpCnt
					var keyComp []string
					if len(dBKey) > 0 {
						keyComp = pathXltr.getKeyComp(dBKey, tblSpec.CompCt, dbNum)
					}
					if log.V(dbLgLvl) {
						log.Infof("%v handleSubtreeNodeXlate: keyComp: %v ; dBKey %v", pathXltr.subReq.reqLogId, keyComp, dBKey)
					}

					dbTblInfo := pathXltr.pathXlateInfo.addPathXlateInfo(tblSpec, &db.Key{keyComp}, dbNum)

					if fldNamePattern, ok := tblFld[FIELD_CURSOR]; ok {
						dbTblInfo.FieldScanPatt = fldNamePattern
						log.Infof("%v handleSubtreeNodeXlate: fldNamePattern %v", pathXltr.subReq.reqLogId, fldNamePattern)
						delete(tblFld, FIELD_CURSOR)
					}
					if v, ok := tblFld[DEL_AS_UPDATE]; ok {
						if v == "true" {
							dbTblInfo.DeleteAction = apis.InspectPathOnDelete
						}
						delete(tblFld, DEL_AS_UPDATE)
					}
					dbYgPath := DbFldYgPathInfo{"", make(map[string]string)}
					yangNodes := make(map[string]bool)
					// copy the leaf nodes form the secDbMap to skip those.
					for ygNameSecDbMap := range ygLeafNodeSecDbMap {
						yangNodes[ygNameSecDbMap] = true
					}
					for dbFld, ygLeafNames := range tblFld {
						nodeNames := strings.Split(ygLeafNames, ",")
						for _, ygNodeName := range nodeNames {
							ygNodeName = strings.Trim(ygNodeName, " ")
							if log.V(dbLgLvl) {
								log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: subOutPram.dbDataMap: leaf node name: ", ygNodeName)
							}
							// isDbFldYgKey - to identify if the yang node name has tagged with {,} in the subscribe
							// db map - to represent the mapped db field is actually yang key
							isDbFldYgKey := false
							nodeName := ygNodeName
							if strings.HasPrefix(nodeName, "{") {
								isDbFldYgKey = true
								nodeName = strings.TrimPrefix(nodeName, "{")
								nodeName = strings.TrimSuffix(nodeName, "}")
								log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: dbFld map - yang node after trimming: ", nodeName)
								dbYgPath.DbFldYgPathMap[dbFld] = ygNodeName
							}
							if !isTrgtNodeLeaf {
								if !isDbFldYgKey {
									ygNodeNameWithPrefix := ygNodeName
									ygLeafNodePath := ygLeafNodePathPrefix + "/" + ygNodeName
									if log.V(dbLgLvl) {
										log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: subOutPram.dbDataMap: ygLeafNodePath: ", ygLeafNodePath)
									}
									if ygLeafXpathInfo, okLeaf := xYangSpecMap[ygLeafNodePath]; okLeaf && ygLeafXpathInfo.nameWithMod != nil {
										ygNodeNameWithPrefix = *(ygLeafXpathInfo.nameWithMod)
										if log.V(dbLgLvl) {
											log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: subOutPram.dbDataMap: leaf node name with prefix: ", ygNodeNameWithPrefix)
										}
									}
									if ygLeafTmp, ok := dbYgPath.DbFldYgPathMap[dbFld]; ok {
										ygLeafTmp = ygLeafTmp + "," + ygNodeNameWithPrefix
										if log.V(dbLgLvl) {
											log.Infof("%v handleSubtreeNodeXlate: same DB field %v is mapped to multiple yang leaf nodes %v", pathXltr.subReq.reqLogId, dbFld, ygLeafTmp)
										}
										dbYgPath.DbFldYgPathMap[dbFld] = dbYgPath.DbFldYgPathMap[dbFld] + "," + ygNodeNameWithPrefix
									} else {
										dbYgPath.DbFldYgPathMap[dbFld] = ygNodeNameWithPrefix
									}
								}
								yangNodes[nodeName] = true
							} else if nodeName == ygEntry.Name {
								// for the target node - leaf / leaf-list
								if log.V(dbLgLvl) {
									log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate:dbDataMap: path is leaf / leaf-list", pathXltr.uriPath)
								}
								if !isDbFldYgKey {
									dbYgPath.DbFldYgPathMap[dbFld] = ""
								}
								dbTblInfo.DbFldYgMapList = append(dbTblInfo.DbFldYgMapList, &dbYgPath)
								yangNodes[nodeName] = true
								break
							}
						}
					}
					// to add the db field which are same as yang leaf/leaf-list nodes
					if !isTrgtNodeLeaf {
						for ygNodeName, ygLeafEntry := range ygEntry.Dir {
							if log.V(dbLgLvl) {
								log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: traversing yang node: ", ygNodeName)
							}
							if !(yangNodes[ygNodeName]) && (ygLeafEntry.IsLeaf() || ygLeafEntry.IsLeafList()) {
								if log.V(dbLgLvl) {
									log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: adding default leaf node: ", ygNodeName)
								}
								ygNodeNameWithPrefix := ygNodeName
								ygLeafNodePath := ygLeafNodePathPrefix + "/" + ygNodeName
								if ygLeafXpathInfo, okLeaf := xYangSpecMap[ygLeafNodePath]; okLeaf && ygLeafXpathInfo.nameWithMod != nil {
									ygNodeNameWithPrefix = *(ygLeafXpathInfo.nameWithMod)
									if log.V(dbLgLvl) {
										log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: subOutPram.dbDataMap: leaf node name: ", ygNodeName)
									}
								}
								dbYgPath.DbFldYgPathMap[ygNodeName] = ygNodeNameWithPrefix
							}
						}
						dbTblInfo.DbFldYgMapList = append(dbTblInfo.DbFldYgMapList, &dbYgPath)
						if log.V(dbLgLvl) {
							log.Infof("%v handleSubtreeNodeXlate:Db field and yang node mapping: "+
								"dbYgPath: %v and for the ygpath: %v", pathXltr.subReq.reqLogId, dbYgPath, pathXltr.uriPath)
						}
						if pathXltr.xpathYgNode != nil {
							// only one db key entry per table for the given subscribe path, so dbTblFldYgPathMap or dbFldYgPathMap won't get overridden
							if len(tblKeyInfo) > 1 {
								pathXltr.xpathYgNode.dbTblFldYgPathMap[tblName] = dbYgPath.DbFldYgPathMap
							} else {
								pathXltr.xpathYgNode.dbFldYgPathMap = dbYgPath.DbFldYgPathMap
							}
						}
					} else if !(yangNodes[ygEntry.Name]) {
						// for the target node - leaf / leaf-list
						if log.V(dbLgLvl) {
							log.Info(pathXltr.subReq.reqLogId, "handleSubtreeNodeXlate: target: LEAF: adding default leaf node: ", ygEntry.Name)
						}
						dbYgPath.DbFldYgPathMap[ygEntry.Name] = ""
						dbTblInfo.DbFldYgMapList = append(dbTblInfo.DbFldYgMapList, &dbYgPath)
					}
				}
			}
		}
	}

	return nil
}

func (pathXltr *subscribePathXlator) addDbFldYangMapInfo() error {
	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: target subscribe path is leaf/leaf-list node: ", pathXltr.uriPath)
	}
	fieldName, err := pathXltr.getDbFieldName()
	if err != nil {
		return err
	}
	if len(pathXltr.pathXlateInfo.ygXpathInfo.compositeFields) > 0 {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: adding composite db field names in the dbFldYgPath map: ", pathXltr.pathXlateInfo.ygXpathInfo.compositeFields)
		}
		for _, dbTblFldName := range pathXltr.pathXlateInfo.ygXpathInfo.compositeFields {
			tblField := strings.Split(dbTblFldName, ":")
			if len(tblField) <= 1 {
				log.Warning(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: Table name is missing in the composite-db-fields annoation for the leaf node path:", pathXltr.uriPath)
				return tlerr.InternalError{Format: "Table name is missing in the composite-db-fields annoation for the leaf node path", Path: pathXltr.uriPath}
			}
			tblName := strings.TrimSpace(tblField[0])
			var dbKeyInfo *dbTableKeyInfo
			for _, dbKeyInfo = range pathXltr.pathXlateInfo.DbKeyXlateInfo {
				if dbKeyInfo.Table.Name == tblName {
					dbFldYgPath := DbFldYgPathInfo{DbFldYgPathMap: make(map[string]string)}
					dbFldYgPath.DbFldYgPathMap[strings.TrimSpace(tblField[1])] = ""
					dbKeyInfo.DbFldYgMapList = append(dbKeyInfo.DbFldYgMapList, &dbFldYgPath)
					if log.V(dbLgLvl) {
						log.Info(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: target subscribe leaf/leaf-list path dbygpathmap list for composite field names: ", dbKeyInfo.DbFldYgMapList)
					}
					break
				}
			}
			if dbKeyInfo == nil || len(dbKeyInfo.DbFldYgMapList) == 0 {
				log.Warningf("%v subscribePathXlator: addDbFldYangMapInfo: Table name %v is not mapped to this path:", pathXltr.subReq.reqLogId, tblName, pathXltr.uriPath)
				intfStrArr := []interface{}{tblName}
				return tlerr.InternalError{Format: "Table name %v is not mapped for the leaf node path", Path: pathXltr.uriPath, Args: intfStrArr}
			}
		}
	} else if len(fieldName) > 0 {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: adding db field name in the dbFldYgPath map: ", fieldName)
		}
		dbFldYgPath := DbFldYgPathInfo{DbFldYgPathMap: make(map[string]string)}
		dbFldYgPath.DbFldYgPathMap[fieldName] = ""
		for _, dbKeyInfo := range pathXltr.pathXlateInfo.DbKeyXlateInfo {
			dbKeyInfo.DbFldYgMapList = append(dbKeyInfo.DbFldYgMapList, &dbFldYgPath)
			if log.V(dbLgLvl) {
				log.Info(pathXltr.subReq.reqLogId, "subscribePathXlator: addDbFldYangMapInfo: target subscribe leaf/leaf-list path dbygpathmap list: ", dbKeyInfo.DbFldYgMapList)
			}
		}
	}

	return nil
}

func (pathXltr *subscribePathXlator) translatePath() error {
	ygXpathInfoTrgt := pathXltr.pathXlateInfo.ygXpathInfo
	ygEntry, ygErr := getYgEntry(pathXltr.subReq.reqLogId, ygXpathInfoTrgt, pathXltr.getYgPath())
	if ygErr != nil {
		return ygErr
	}
	if log.V(dbLgLvl) {
		log.Infof("%v translatePath: path: %v; ygXpathInfoTrgt.Name: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath, ygEntry.Name)
	}

	if ygXpathInfoTrgt.isDataSrcDynamic != nil {
		pathXltr.pathXlateInfo.IsDataSrcDynamic = *ygXpathInfoTrgt.isDataSrcDynamic
	}

	if log.V(dbLgLvl) {
		log.Infof("%v translatePath: path: %v; isDataSrcDynamic: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath, pathXltr.pathXlateInfo.IsDataSrcDynamic)
	}

	pathXltr.pathXlateInfo.PType = pathXltr.getSubscribeMode(nil)
	if log.V(dbLgLvl) {
		log.Infof("%v translatePath: path: %v; subMode: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath, pathXltr.pathXlateInfo.PType)
	}
	ygNodeSubMinIntrvl := 0
	if pathXltr.subReq.xlateNodeType == TARGET_NODE {
		ygNodeSubMinIntrvl = ygXpathInfoTrgt.subscribeMinIntvl
	} else if pathXltr.xpathYgNode != nil {
		if log.V(dbLgLvl) {
			log.Infof("%v translatePath: ygXpathInfoTrgt.subscribeMinIntvl: %v", pathXltr.subReq.reqLogId, ygXpathInfoTrgt.subscribeMinIntvl)
		}
		ygNodeSubMinIntrvl = pathXltr.xpathYgNode.subMinIntrvl
	} else {
		log.Warningf("%v translatePath: pathXltr.xpathYgNode not found for the path: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath)
	}

	if pathXltr.pathXlateInfo.PType == Sample || pathXltr.subReq.subReqMode == TargetDefined {
		if pathXltr.pathXlateInfo.MinInterval < ygNodeSubMinIntrvl {
			pathXltr.pathXlateInfo.MinInterval = ygNodeSubMinIntrvl
		} else if pathXltr.pathXlateInfo.MinInterval == 0 {
			pathXltr.pathXlateInfo.MinInterval = apis.SAMPLE_NOTIFICATION_MIN_INTERVAL
		}
		if log.V(dbLgLvl) {
			log.Infof("%v translatePath: path: %v; pathXltr.pathXlateInfo.MinInterval: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath, pathXltr.pathXlateInfo.MinInterval)
		}
	}

	if len(ygXpathInfoTrgt.xfmrFunc) > 0 {
		if err := pathXltr.handleSubtreeNodeXlate(); err != nil {
			return err
		}
	} else {
		if pathXltr.subReq.subReqMode == OnChange && (ygXpathInfoTrgt.dbIndex == db.CountersDB && !ygXpathInfoTrgt.subscriptionFlags.Has(subsOnChangeEnable)) {
			log.Warningf("%v translatePath:Subscribe not supported for COUNTERS_DB by default for the path:: %v", pathXltr.subReq.reqLogId, pathXltr.uriPath)
			return tlerr.NotSupportedError{Format: "Onchange subscription is not supported for COUNTERS_DB by default", Path: pathXltr.uriPath}
		}
		if err := pathXltr.handleNonSubtreeNodeXlate(); err != nil {
			return err
		}
	}

	if pathXltr.pathXlateInfo.PType == Sample || pathXltr.subReq.subReqMode == TargetDefined {
		// setting the interval after the translation
		if pathXltr.pathXlateInfo.MinInterval < ygNodeSubMinIntrvl {
			pathXltr.pathXlateInfo.MinInterval = ygNodeSubMinIntrvl
		} else if pathXltr.pathXlateInfo.MinInterval == 0 {
			pathXltr.pathXlateInfo.MinInterval = apis.SAMPLE_NOTIFICATION_MIN_INTERVAL
		}
		if log.V(dbLgLvl) {
			log.Infof("%v translatePath: Min Interval: %v; for the path: %v", pathXltr.subReq.reqLogId, pathXltr.pathXlateInfo.MinInterval, pathXltr.uriPath)
		}
		if pathXltr.subReq.subReqXlateInfo.TrgtPathInfo.hasDbTableInfo() && pathXltr.subReq.subReqXlateInfo.TrgtPathInfo.MinInterval < pathXltr.pathXlateInfo.MinInterval {
			pathXltr.subReq.subReqXlateInfo.TrgtPathInfo.MinInterval = pathXltr.pathXlateInfo.MinInterval
		}
	}
	return nil
}

func (pathXltr *subscribePathXlator) handleYangToDbKeyXfmr() (string, error) {
	if log.V(dbLgLvl) {
		log.Infof("%v handleYangToDbKeyXfmr.. pathXltr.uriPath: %v; isTrgtPathWldcrd: %v",
			pathXltr.subReq.reqLogId, pathXltr.uriPath, pathXltr.subReq.isTrgtPathWldcrd)
	}

	if pathXltr.keyXfmrYgXpathInfo == nil {
		log.Warning(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: yangXpathInfo is nil in the xYangSpecMap for the path: ", pathXltr.uriPath)
		return "", tlerr.InternalError{Format: pathXltr.subReq.reqLogId + "yangXpathInfo is nil in the xYangSpecMap for the path", Path: pathXltr.uriPath}
	}

	if len(pathXltr.keyXfmrYgXpathInfo.xfmrKey) > 0 {
		ygXpathInfo := pathXltr.keyXfmrYgXpathInfo
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: key transformer name:", ygXpathInfo.xfmrKey)
		}
		ygotRoot, err := newYgotRootObj()
		if err != nil {
			log.Warning(pathXltr.subReq.reqLogId, "Error: newYgotRootObj error: ", err)
			return "", err
		}

		currDbNum := ygXpathInfo.dbIndex
		inParams := formXfmrInputRequest(pathXltr.subReq.dbs[ygXpathInfo.dbIndex], pathXltr.subReq.dbs, currDbNum, ygotRoot, pathXltr.uriPath,
			pathXltr.subReq.reqUri, SUBSCRIBE, "", nil, nil, nil, pathXltr.subReq.txCache)
		dBTblKey, errKey := keyXfmrHandler(inParams, ygXpathInfo.xfmrKey)
		if errKey == nil {
			if log.V(dbLgLvl) {
				log.Infof("%v handleYangToDbKeyXfmr: key transformer: %v; dBTblKey: %v", pathXltr.subReq.reqLogId, ygXpathInfo.xfmrKey, dBTblKey)
			}
		} else {
			log.Warning(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: keyXfmrHandler callback error:", errKey)
		}
		return dBTblKey, errKey
	} else {
		if log.V(dbLgLvl) {
			log.Infof("%v handleYangToDbKeyXfmr: default db key translation uri path: %v; ygListXpathInfo.dbIndex: %v ",
				pathXltr.subReq.reqLogId, pathXltr.uriPath, pathXltr.keyXfmrYgXpathInfo.dbIndex)
		}

		dbKey := "*"
		isKeyEmpty := true

		keyDelm := pathXltr.subReq.dbs[pathXltr.keyXfmrYgXpathInfo.dbIndex].Opts.KeySeparator
		log.Info(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: keyDelm: ", keyDelm)

		pathElems := pathXltr.gPath.Elem
		ygPath := "/" + pathElems[0].Name

		for idx := 1; idx < len(pathElems); idx++ {
			ygNames := strings.Split(pathElems[idx].Name, ":")
			if len(ygNames) == 1 {
				ygPath = ygPath + "/" + ygNames[0]
			} else {
				ygPath = ygPath + "/" + ygNames[1]
			}
			if log.V(dbLgLvl) {
				log.Info(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: ygPath: ", ygPath)
			}
			if len(pathElems[idx].Key) > 0 {
				if ygXpathInfo, ok := xYangSpecMap[ygPath]; ok {
					if ygXpathInfo.virtualTbl == nil || !(*ygXpathInfo.virtualTbl) {
						/* Build the database key string in the form of:
						 * <keyVal1><Seperator><KeyVal2>...<Seperator><KeyValN>
						 * or just <keyVal1> in the case of a single key.
						 * Note the yangEntry.Key field contains the key field names in order and
						 * seperated by ' ' characters while the pathElems.Key variable maps the
						 * field names to their values. */
						for _, keyName := range strings.Split(ygXpathInfo.yangEntry.Key, " ") {
							kv := pathElems[idx].Key[keyName]
							if isKeyEmpty {
								// For the first key there is no need to append and add the seperator.
								dbKey = kv
								isKeyEmpty = false
							} else {
								dbKey = dbKey + keyDelm + kv
							}
						}
					}
				} else {
					log.Warning(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: xpathinfo not found for the ygpath: ", ygPath)
				}
			}
		}
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleYangToDbKeyXfmr: default translation: dbKey: ", dbKey)
		}
		return dbKey, nil
	}
}

func (pathXltr *subscribePathXlator) handleNonSubtreeNodeXlate() error {
	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "handleNonSubtreeNodeXlate: uriPath: ", pathXltr.uriPath)
	}
	var keyComp []string
	tblNameMap := make(map[string]bool)

	ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo
	if pTblName := pathXltr.getXpathInfoTableName(); pTblName != nil {
		if log.V(dbLgLvl) {
			log.Infof("%v handleNonSubtreeNodeXlate: mapped table name: %v for the path: %v", pathXltr.subReq.reqLogId, *pTblName, pathXltr.uriPath)
		}
		tblNameMap[*pTblName] = true
	} else if ygXpathInfo.xfmrTbl != nil && len(*ygXpathInfo.xfmrTbl) > 0 {
		if tblNames, err := pathXltr.handleTableXfmrCallback(); err != nil {
			return err
		} else {
			if log.V(dbLgLvl) {
				log.Infof("%v handleNonSubtreeNodeXlate: table transfoerm: tblNames: %v for the path: %v", pathXltr.subReq.reqLogId, tblNames, pathXltr.uriPath)
			}
			for _, tblName := range tblNames {
				tblNameMap[tblName] = true
			}
		}
	}

	tblCnt := 0
	if pathXltr.parentXlateInfo != nil && !ygXpathInfo.subscriptionFlags.Has(subsDelAsUpdate) {
		for _, dbTblKeyInfo := range pathXltr.parentXlateInfo.DbKeyXlateInfo {
			if dbTblKeyInfo.DbNum == ygXpathInfo.dbIndex && tblNameMap[dbTblKeyInfo.Table.Name] {
				tblCnt++
				break
			}
		}
	}

	if tblCnt == len(tblNameMap) {
		if log.V(dbLgLvl) {
			log.Infof("%v handleNonSubtreeNodeXlate: tables are actually mapped its parent node; table count: %v for the path $v", pathXltr.subReq.reqLogId, tblCnt, pathXltr.uriPath)
		}
		return nil
	}

	dBTblKey, err := pathXltr.handleYangToDbKeyXfmr()
	if err != nil {
		return err
	}
	if len(dBTblKey) > 0 {
		if log.V(dbLgLvl) {
			log.Infof("%v handleNonSubtreeNodeXlate: dBTblKey: %v for the path %v", pathXltr.subReq.reqLogId, dBTblKey, pathXltr.uriPath)
		}
		keyComp = pathXltr.getKeyComp(dBTblKey, ygXpathInfo.dbKeyCompCnt, ygXpathInfo.dbIndex)
	}

	dbKey := &db.Key{Comp: keyComp}

	if len(keyComp) > 0 {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleNonSubtreeNodeXlate: keyComp: ", keyComp)
		}
	}

	for tblName := range tblNameMap {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleNonSubtreeNodeXlate: Adding tablename: ", tblName)
		}
		if log.V(dbLgLvl) {
			log.Infof("%v handleNonSubtreeNodeXlate: Alias mode; resolving the db key %v for the tablename: ", pathXltr.subReq.reqLogId, dbKey, tblName)
		}
		dbKeyRslvr := &DbYangKeyResolver{tableName: tblName, key: dbKey,
			dbs: pathXltr.subReq.dbs, dbIdx: ygXpathInfo.dbIndex, uriPath: pathXltr.uriPath, reqLogId: pathXltr.subReq.reqLogId}
		dbKeyComp, err := dbKeyRslvr.resolve(SUBSCRIBE)
		if err != nil {
			if log.V(dbLgLvl) {
				log.Warningf("%v handleNonSubtreeNodeXlate: Error in resolving the Db key %v for the table: %v and request path: %v",
					pathXltr.subReq.reqLogId, dbKey, tblName, pathXltr.uriPath)
			}
			return tlerr.InternalError{Format: pathXltr.subReq.reqLogId + "Translate: Error: " + err.Error(), Path: pathXltr.uriPath}
		}
		dbKey.Comp = dbKeyComp
		pathXltr.pathXlateInfo.addPathXlateInfo(&db.TableSpec{Name: tblName, CompCt: ygXpathInfo.dbKeyCompCnt}, dbKey, ygXpathInfo.dbIndex)
	}

	ygEntry, ygErr := getYgEntry(pathXltr.subReq.reqLogId, pathXltr.pathXlateInfo.ygXpathInfo, pathXltr.xpathYgNode.ygPath)
	if ygErr != nil {
		return ygErr
	}
	if pathXltr.subReq.xlateNodeType == TARGET_NODE && (ygEntry.IsLeafList() || ygEntry.IsLeaf()) {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleNonSubtreeNodeXlate: leaf/leaf-list target node for the path: ", pathXltr.uriPath)
		}
		if err := pathXltr.addDbFldYangMapInfo(); err != nil {
			return err
		}
	}

	return nil
}

func (pathXltr *subscribePathXlator) handleTableXfmrCallback() ([]string, error) {
	if log.V(dbLgLvl) {
		log.Info(pathXltr.subReq.reqLogId, "handleTableXfmrCallback:", pathXltr.uriPath)
	}
	ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo

	currDbNum := ygXpathInfo.dbIndex
	dbDataMap := make(RedisDbMap)
	for i := db.ApplDB; i < db.MaxDB; i++ {
		dbDataMap[i] = make(map[string]map[string]db.Value)
	}
	ygotObj, err := newYgotRootObj()
	if err != nil {
		log.Warningf(pathXltr.subReq.reqLogId, "newYgotRootObj: error: %v", err)
		return nil, err
	}
	inParams := formXfmrInputRequest(pathXltr.subReq.dbs[ygXpathInfo.dbIndex], pathXltr.subReq.dbs, currDbNum, ygotObj, pathXltr.uriPath,
		pathXltr.subReq.reqUri, SUBSCRIBE, "", &dbDataMap, nil, nil, pathXltr.subReq.txCache)
	tblList, tblXfmrErr := xfmrTblHandlerFunc(*ygXpathInfo.xfmrTbl, inParams, pathXltr.subReq.tblKeyCache)
	if tblXfmrErr != nil {
		if log.V(dbLgLvl) {
			log.Warningf("%v handleTableXfmrCallback: table transformer callback returns error: %v and table transformer callback: %v", pathXltr.subReq.reqLogId, tblXfmrErr, *ygXpathInfo.xfmrTbl)
		}
		return nil, tblXfmrErr
	}
	if inParams.isVirtualTbl != nil && *inParams.isVirtualTbl {
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "handleTableXfmrCallback: isVirtualTbl is set to true for the table transformer callback: ", *ygXpathInfo.xfmrTbl)
		}
		return nil, nil
	}
	if log.V(dbLgLvl) {
		log.Infof("%v handleTableXfmrCallback: table names from table transformer callback: %v for the transformer name: %v", pathXltr.subReq.reqLogId, tblList, *ygXpathInfo.xfmrTbl)
	}
	return tblList, nil
}

func (pathXltr *subscribePathXlator) getXpathInfoTableName() *string {
	ygXpathInfo := pathXltr.pathXlateInfo.ygXpathInfo
	if ygXpathInfo.tableName != nil && *ygXpathInfo.tableName != "NONE" {
		return ygXpathInfo.tableName
	}
	return nil
}

func (pathXltr *subscribePathXlator) getYgPath() string {
	if pathXltr.xpathYgNode != nil {
		return pathXltr.xpathYgNode.ygPath
	}
	return ""
}

func (reqXlator *subscribeReqXlator) translateChildNodePaths(ygXpathInfo *yangXpathInfo) error {
	var err error
	ygNode, ygErr := getYgEntry(reqXlator.subReq.reqLogId, ygXpathInfo, reqXlator.subReq.ygPath)
	if ygErr != nil {
		return ygErr
	}
	if log.V(dbLgLvl) {
		log.Infof("%v translateChildNodePaths: yangEntry.Name: %v for the request uri path: %v", reqXlator.subReq.reqLogId, ygNode.Name, reqXlator.subReq.reqUri)
	}
	if !ygNode.IsList() && !ygNode.IsContainer() {
		return nil
	}

	if reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.HandlerFunc != nil {
		if log.V(dbLgLvl) {
			log.Infof("%v translateChildNodePaths: custmCallback is true for the request uri path: %v", reqXlator.subReq.reqLogId, reqXlator.subReq.reqUri)
		}
		return nil
	}

	rltvUriPath := ""
	reqXlator.subReq.xlateNodeType = CHILD_NODE

	reqXlator.trgtYgNode = &(ygXpathNode{relUriPath: rltvUriPath, ygXpathInfo: ygXpathInfo, dbFldYgPathMap: make(map[string]string),
		dbTblFldYgPathMap: make(map[string]map[string]string), subMinIntrvl: ygXpathInfo.subscribeMinIntvl, ygPath: reqXlator.subReq.ygPath})

	if ygNode.IsList() {
		reqXlator.trgtYgNode.listKeyMap = make(map[string]bool)
		if log.V(dbLgLvl) {
			log.Info(reqXlator.subReq.reqLogId, "collectChldYgXPathInfo: yangEntry.Key: ", ygNode.Key)
		}
		keyElemNames := strings.Fields(ygNode.Key)
		for _, keyName := range keyElemNames {
			reqXlator.trgtYgNode.listKeyMap[keyName] = true
		}
	}

	reqXlator.trgtYgNode.pathXlateInfo = reqXlator.subReq.subReqXlateInfo.TrgtPathInfo

	if err = reqXlator.collectChldYgXPathInfo(ygNode, reqXlator.subReq.ygPath, rltvUriPath, ygXpathInfo, reqXlator.trgtYgNode); err != nil {
		log.Warning(reqXlator.subReq.reqLogId, "translateChildNodePaths: Error in collectChldYgXPathInfo; error: ", err)
		return err
	}

	if err := reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.addDbFldYgPathMap("", reqXlator.trgtYgNode); err != nil {
		if log.V(dbLgLvl) {
			log.Warning(reqXlator.subReq.reqLogId, "translateChildNodePaths: Error in addDbFldYgPathMap; error: ", err)
		}
		return err
	}

	if err = reqXlator.traverseYgXpathAndTranslate(reqXlator.trgtYgNode, "", reqXlator.subReq.subReqXlateInfo.TrgtPathInfo); err != nil {
		if log.V(dbLgLvl) {
			log.Warning(reqXlator.subReq.reqLogId, "translateChildNodePaths: Error in traverseYgXpathAndTranslate; error: ", err)
		}
	}

	return err
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) isSamePathXlateInfo(parentPathXlateInfo *XfmrSubscribePathXlateInfo) bool {
	if (pathXlateInfo.ygXpathInfo.yangType == YANG_LEAF || pathXlateInfo.ygXpathInfo.yangType == YANG_LEAF_LIST) &&
		pathXlateInfo.ygXpathInfo.subscriptionFlags.Has(subsDelAsUpdate) {
		return false
	}
	if pathXlateInfo.PType == Sample {
		if parentPathXlateInfo.PType != pathXlateInfo.PType || parentPathXlateInfo.MinInterval != pathXlateInfo.MinInterval {
			return false
		}
	}
	return pathXlateInfo.isDbTablePresentInParent(parentPathXlateInfo.DbKeyXlateInfo)
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) isDbTablePresentInParent(parentDbKeyXlateInfo []*dbTableKeyInfo) bool {
	if log.V(dbLgLvl) {
		log.Info(pathXlateInfo.reqLogId, "isDbTablePresentInParent: path: ", pathXlateInfo.Path)
	}
	if len(parentDbKeyXlateInfo) == 0 {
		if log.V(dbLgLvl) {
			log.Info(pathXlateInfo.reqLogId, "isDbTablePresentInParent: parentDbKeyXlateInfo is empty for the path: ", pathXlateInfo.Path)
		}
		return false
	}
	if log.V(dbLgLvl) {
		log.Info(pathXlateInfo.reqLogId, "isDbTablePresentInParent: pathXlateInfo.DbKeyXlateInfo: ", pathXlateInfo.DbKeyXlateInfo)
	}
	for _, dbXlateInfo := range pathXlateInfo.DbKeyXlateInfo {
		isPresent := false
		for _, parentDbInfo := range parentDbKeyXlateInfo {
			if log.V(dbLgLvl) {
				log.Infof("%v isDbTablePresentInParent: parentDbInfo: %v for the path: %v", pathXlateInfo.reqLogId, parentDbInfo, pathXlateInfo.Path)
			}
			if parentDbInfo.DbNum != dbXlateInfo.DbNum {
				continue
			}
			if parentDbInfo.Table.Name != dbXlateInfo.Table.Name {
				continue
			}
			isPresent = true
			break
		}
		if !isPresent {
			return false
		}
	}
	return true
}

func (reqXlator *subscribeReqXlator) traverseYgXpathAndTranslate(ygXpNode *ygXpathNode, parentRelUri string, parentPathXlateInfo *XfmrSubscribePathXlateInfo) error {
	log.Infof("%v traverseYgXpathAndTranslate: ygXpNode path:%v; parentRelUri: %v; parentPathXlateInfo path: %v ",
		reqXlator.subReq.reqLogId, ygXpNode.ygPath, parentRelUri, parentPathXlateInfo.Path)
	var err error

	if log.V(dbLgLvl) {
		log.Info(reqXlator.subReq.reqLogId, "traverseYgXpathAndTranslate: parentPathXlateInfo: ", *parentPathXlateInfo)
	}

	for _, chldNode := range ygXpNode.chldNodes {

		if log.V(dbLgLvl) {
			log.Infof("%v traverseYgXpathAndTranslate: child path: %v; isParentTbl: %v; relUriPath: %v ",
				reqXlator.subReq.reqLogId, chldNode.ygPath, chldNode.isParentTbl, chldNode.relUriPath)
		}

		var pathXlateInfo *XfmrSubscribePathXlateInfo
		relUri := parentRelUri

		if chldNode.isParentTbl {
			pathXlateInfo = parentPathXlateInfo
			if log.V(dbLgLvl) {
				log.Info(reqXlator.subReq.reqLogId, "traverseYgXpathAndTranslate: isParentTbl: true")
			}
			if len(chldNode.dbFldYgPathMap) > 0 || len(chldNode.dbTblFldYgPathMap) > 0 {
				pathXlateInfo.copyDbFldYgPathMap(relUri, chldNode)
			} else {
				if log.V(dbLgLvl) {
					log.Info(reqXlator.subReq.reqLogId, "traverseYgXpathAndTranslate: isParentTbl: no db field yang map found for the path: ", reqXlator.subReq.reqUri, chldNode.relUriPath)
				}
			}
			if parentPathXlateInfo.PType == Sample && parentPathXlateInfo.MinInterval < chldNode.subMinIntrvl {
				parentPathXlateInfo.MinInterval = chldNode.subMinIntrvl
			}
		} else {
			var gPathCurr *gnmipb.Path
			if gPathCurr, err = reqXlator.uriToAbsolutePath(chldNode.relUriPath); err != nil {
				return err
			}

			uriPath := reqXlator.subReq.reqUri + chldNode.relUriPath
			if log.V(dbLgLvl) {
				log.Info(reqXlator.subReq.reqLogId, "next child node URI Path: ", uriPath)
			}

			pathXlator, err := reqXlator.getSubscribePathXlator(gPathCurr, uriPath, chldNode.ygXpathInfo, parentPathXlateInfo, chldNode)
			if err != nil {
				if log.V(dbLgLvl) {
					log.Warningf("%v traverseYgXpathAndTranslate: Error in getSubscribePathXlator: %v for the path: %v", reqXlator.subReq.reqLogId, err, uriPath)
				}
				return err
			}

			var forceSeparateEntry bool

			err = pathXlator.translatePath()
			// Some apps are not yet ready for subscription. Ignore such child paths during SAMPLE (and ONCE & POLL)
			if err != nil && reqXlator.subReq.subReqMode == Sample {
				log.Warningf("%v traverseYgXpathAndTranslate: Skip child node '%s' due to error in translatePath: %v",
					reqXlator.subReq.reqLogId, uriPath, err)
				continue
			}
			if err != nil {
				log.Warningf("%v traverseYgXpathAndTranslate: Error in translatePath: %v for the path %v", reqXlator.subReq.reqLogId, err, uriPath)
				return err
			}
			chldNode.pathXlateInfo = pathXlator.pathXlateInfo
			pathXlateInfo = chldNode.pathXlateInfo
			if pathXlateInfo.HandlerFunc != nil {
				forceSeparateEntry = true
				for key := range chldNode.dbFldYgPathMap {
					delete(chldNode.dbFldYgPathMap, key)
				}
				for key := range chldNode.dbTblFldYgPathMap {
					delete(chldNode.dbTblFldYgPathMap, key)
				}
			}

			hasDbTblInfo := chldNode.pathXlateInfo.hasDbTableInfo()

			// resolve and add xlate info applicable for onchange, or if the target node db info is empty, or current parent node xlate info's ptype is onchange
			if !forceSeparateEntry && chldNode.pathXlateInfo.isSamePathXlateInfo(parentPathXlateInfo) {
				pathXlateInfo = parentPathXlateInfo
				if log.V(dbLgLvl) {
					log.Infof("%v traverseYgXpathAndTranslate: isDbTablePresentInParent is true for the path: %v for the parent path: %v", reqXlator.subReq.reqLogId, uriPath, parentPathXlateInfo.Path)
				}
				parentPathXlateInfo.copyDbFldYgPathMap(relUri, chldNode)
				if chldNode.pathXlateInfo.PType == Sample && parentPathXlateInfo.MinInterval < chldNode.pathXlateInfo.MinInterval {
					parentPathXlateInfo.MinInterval = chldNode.pathXlateInfo.MinInterval
				}
			} else {
				relUri = chldNode.relUriPath

				if len(chldNode.ygXpathInfo.xfmrFunc) == 0 {
					// only for non sub tree - for subtree, got added by handleSubtreeNodeXlate
					if err := chldNode.pathXlateInfo.addDbFldYgPathMap("", chldNode); err != nil {
						return err
					}
				}

				if chldNode.ygXpathInfo.yangType != YANG_LIST && parentPathXlateInfo.hasDbTableInfo() {
					// other than list node, that is for the container / leaf / leaf-list node
					// the db key entry of the parent list node's table db key will be used as the table
					// key for the container/leaf/leaf-list node's table
					// this is needed to subscribe to the table for the particular key entry
					// if we need to add support to handle if the container table key is different
					// than it parent table key, if so then the feature team needs to write the
					// yang to db key transformer for the given path and its associated table
					for _, dbKeyInfo := range chldNode.pathXlateInfo.DbKeyXlateInfo {
						if dbKeyInfo.Key == nil {
							// since the yang key is same for the all mapped tables, so assigning
							// the first key.. which will be same for all the tables
							dbKeyInfo.Key = parentPathXlateInfo.DbKeyXlateInfo[0].Key
						}
					}
				}

				hasDbTblInfo = chldNode.pathXlateInfo.hasDbTableInfo()

				if len(reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.DbKeyXlateInfo) == 0 &&
					(!parentPathXlateInfo.TrgtNodeChld && hasDbTblInfo) {
					if log.V(dbLgLvl) {
						log.Info("traverseYgXpathAndTranslate: target info path is empty; setting TrgtNodeChld flag to true for the path: ", chldNode.pathXlateInfo.Path)
					}
					chldNode.pathXlateInfo.TrgtNodeChld = true
				}

				if log.V(dbLgLvl) {
					log.Info("traverseYgXpathAndTranslate: chldNode.pathXlateInfo: ", chldNode.pathXlateInfo)
				}
				reqXlator.subReq.subReqXlateInfo.ChldPathsInfo = append(reqXlator.subReq.subReqXlateInfo.ChldPathsInfo, chldNode.pathXlateInfo)

				if !hasDbTblInfo {
					// for list node and the length of DbKeyXlateInfo is 0
					log.Warning(reqXlator.subReq.reqLogId, "traverseYgXpathAndTranslate: Db table information is not found for the node path : %v; traversing the child nodes with grand parent", uriPath)
					pathXlateInfo = parentPathXlateInfo
				}
			}

			// for leaf node which are mapped to specific tables different than parent
			for _, chldPathXlateInfo := range chldNode.pathXlateInfo.chldXlateInfos {
				if len(reqXlator.subReq.subReqXlateInfo.TrgtPathInfo.DbKeyXlateInfo) == 0 && (!hasDbTblInfo && !chldNode.pathXlateInfo.TrgtNodeChld) {
					chldPathXlateInfo.TrgtNodeChld = true
				}
				reqXlator.subReq.subReqXlateInfo.ChldPathsInfo = append(reqXlator.subReq.subReqXlateInfo.ChldPathsInfo, chldPathXlateInfo)
			}
		}

		if pathXlateInfo.HandlerFunc != nil {
			if log.V(dbLgLvl) {
				log.Infof("%v traverseYgXpathAndTranslate: HandlerFunc is set for the gnmi path: %v", reqXlator.subReq.reqLogId, chldNode.pathXlateInfo.Path)
			}
			continue
		}

		if err = reqXlator.traverseYgXpathAndTranslate(chldNode, relUri, pathXlateInfo); err != nil {
			return err
		}
	}
	return err
}

func (reqXlator *subscribeReqXlator) debugTrvsalCtxt(ygEntry *yang.Entry, ygPath string, rltvUriPath string, ygXpathInfo *yangXpathInfo) {
	if log.V(dbLgLvl) {
		log.Infof("%v debugTrvsalCtxt ygPath: %v; rltvUriPath: %v; ygXpathInfo: %v; ygEntry: %v", reqXlator.subReq.reqLogId, ygPath, rltvUriPath, *ygXpathInfo, ygEntry)
	}
}

type ygXpathNode struct {
	relUriPath        string
	ygXpathInfo       *yangXpathInfo
	chldNodes         []*ygXpathNode
	dbFldYgPathMap    map[string]string
	dbTblFldYgPathMap map[string]map[string]string
	pathXlateInfo     *XfmrSubscribePathXlateInfo
	isParentTbl       bool
	listKeyMap        map[string]bool
	ygPath            string
	subMinIntrvl      int
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) copyDbFldYgPathMap(parentRelUri string, ygXpNode *ygXpathNode) {
	if log.V(dbLgLvl) {
		log.Infof("%v copyDbFldYgPathMap: parentRelUri: %v; ygXpNode.relUriPath: %v", pathXlateInfo.reqLogId, parentRelUri, ygXpNode.relUriPath)
	}
	if sIdx := strings.Index(ygXpNode.relUriPath, parentRelUri); sIdx == -1 {
		log.Warning(pathXlateInfo.reqLogId, "copyDbFldYgPathMap: Not able to get the relative path of the node for the relUriPath: ", ygXpNode.relUriPath)
	} else {
		if log.V(dbLgLvl) {
			log.Info(pathXlateInfo.reqLogId, "copyDbFldYgPathMap: sIdx: ", sIdx)
		}
		relPath := ygXpNode.relUriPath[sIdx+len(parentRelUri):]
		if log.V(dbLgLvl) {
			log.Info(pathXlateInfo.reqLogId, "copyDbFldYgPathMap: relPath: ", relPath)
		}
		if err := pathXlateInfo.addDbFldYgPathMap(relPath, ygXpNode); err != nil {
			if log.V(dbLgLvl) {
				log.Warning(pathXlateInfo.reqLogId, "copyDbFldYgPathMap: error in addDbFldYgPathMap; for the relUriPath: ", ygXpNode.relUriPath)
			}
		}
	}
}

func (pathXlateInfo *XfmrSubscribePathXlateInfo) addDbFldYgPathMap(relPath string, ygXpNode *ygXpathNode) error {

	if len(pathXlateInfo.DbKeyXlateInfo) == 0 && len(ygXpNode.dbFldYgPathMap) > 0 {
		log.Warning(pathXlateInfo.reqLogId, "addDbFldYgPathMap: pathXlateInfo.DbKeyXlateInfo is empty for the path ", ygXpNode.ygPath)
		return tlerr.InternalError{Format: "DbKeyXlateInfo is empty: ", Path: ygXpNode.ygPath}
	}
	if len(ygXpNode.dbTblFldYgPathMap) > 0 {
		// multi table field mapped to same yang node
		for _, dbKeyInfo := range pathXlateInfo.DbKeyXlateInfo {
			if dbFldYgMap, ok := ygXpNode.dbTblFldYgPathMap[dbKeyInfo.Table.Name]; ok {
				dbFldInfo := DbFldYgPathInfo{relPath, make(map[string]string)}
				dbFldInfo.DbFldYgPathMap = dbFldYgMap
				dbKeyInfo.DbFldYgMapList = append(dbKeyInfo.DbFldYgMapList, &dbFldInfo)
				if log.V(dbLgLvl) {
					log.Infof("%v addDbFldYgPathMap: multi table field nodes: dbFldInfo: %v for the table name: %v", pathXlateInfo.reqLogId, dbFldInfo, dbKeyInfo.Table.Name)
				}
			} else {
				log.Warningf("%v addDbFldYgPathMap: Not able to find the db field mapping of the node: %v for the table %v", pathXlateInfo.reqLogId, ygXpNode.ygPath, dbKeyInfo.Table.Name)
				return tlerr.InternalError{Format: "Not able to find the db field mapping for the path ", Path: ygXpNode.ygPath}
			}
		}
	} else if len(ygXpNode.dbFldYgPathMap) > 0 {
		if log.V(dbLgLvl) {
			log.Info(pathXlateInfo.reqLogId, "addDbFldYgPathMap: adding the direct leaf nodes: ygXpNode.dbFldYgPathMap: ", ygXpNode.dbFldYgPathMap)
		}
		dbFldYgPathInfo := &DbFldYgPathInfo{relPath, ygXpNode.dbFldYgPathMap}

		for _, dbKeyInfo := range pathXlateInfo.DbKeyXlateInfo {
			if log.V(dbLgLvl) {
				log.Info(pathXlateInfo.reqLogId, "addDbFldYgPathMap: adding the direct leaf node to the table : ", dbKeyInfo.Table.Name)
			}
			dbKeyInfo.DbFldYgMapList = append(dbKeyInfo.DbFldYgMapList, dbFldYgPathInfo)
		}
	}

	return nil
}

func (ygXpNode *ygXpathNode) addDbFldNames(ygNodeName string, dbFldNames []string) error {
	for _, dbTblFldName := range dbFldNames {
		tblField := strings.Split(dbTblFldName, ":")
		if len(tblField) <= 1 {
			log.Warning("addDbFldNames: Table name is missing in the composite-db-fields annoation for the leaf node path:", ygXpNode.relUriPath, "/", ygNodeName)
			return tlerr.InternalError{Format: "Table name is missing in the composite-db-fields annoation for the leaf node path", Path: ygXpNode.relUriPath + "/" + ygNodeName}
		}
		tblName := strings.TrimSpace(tblField[0])
		if _, ok := ygXpNode.dbTblFldYgPathMap[tblName]; !ok {
			ygXpNode.dbTblFldYgPathMap[tblName] = make(map[string]string)
			ygXpNode.dbTblFldYgPathMap[tblName][strings.TrimSpace(tblField[1])] = ygNodeName
		} else {
			ygXpNode.dbTblFldYgPathMap[tblName][strings.TrimSpace(tblField[1])] = ygNodeName
		}
	}
	return nil
}

func (ygXpNode *ygXpathNode) addDbFldName(ygNodeName string, dbFldName string) {
	ygXpNode.dbFldYgPathMap[dbFldName] = ygNodeName
}

func (ygXpNode *ygXpathNode) addChildNode(rltUri string, ygXpathInfo *yangXpathInfo, ygPath string) *ygXpathNode {
	chldNode := ygXpathNode{relUriPath: rltUri, ygXpathInfo: ygXpathInfo, ygPath: ygPath}
	chldNode.dbFldYgPathMap = make(map[string]string)
	chldNode.dbTblFldYgPathMap = make(map[string]map[string]string)
	chldNode.listKeyMap = make(map[string]bool)
	chldNode.subMinIntrvl = chldNode.ygXpathInfo.subscribeMinIntvl
	ygXpNode.chldNodes = append(ygXpNode.chldNodes, &chldNode)
	return &chldNode
}

func (pathXltr *subscribePathXlator) getDbFieldName() (string, error) {
	xpathInfo := pathXltr.pathXlateInfo.ygXpathInfo
	ygEntry, ygErr := getYgEntry(pathXltr.subReq.reqLogId, xpathInfo, pathXltr.xpathYgNode.ygPath)
	if ygErr != nil {
		return "", ygErr
	}
	if log.V(dbLgLvl) {
		log.Infof("%v getDbFieldName: fieldName: %v; yangEntry: %v", pathXltr.subReq.reqLogId, xpathInfo.fieldName, ygEntry)
	}
	if ygEntry.IsLeafList() || ygEntry.IsLeaf() {
		fldName := xpathInfo.fieldName
		if len(fldName) == 0 {
			fldName = ygEntry.Name
		}
		if log.V(dbLgLvl) {
			log.Info(pathXltr.subReq.reqLogId, "getDbFieldName: fldName: ", fldName)
		}
		return fldName, nil
	}
	return "", nil
}

func (reqXlator *subscribeReqXlator) validateYangPath(uriPath string, ygXpathInfo *yangXpathInfo) bool {
	if len(ygXpathInfo.validateFunc) > 0 {
		if log.V(dbLgLvl) {
			log.Infof("%v validateYangPath: calbback name: %v, uriPath: %v", reqXlator.subReq.reqLogId, ygXpathInfo.validateFunc, uriPath)
		}
		currDbNum := ygXpathInfo.dbIndex
		inParams := formXfmrInputRequest(reqXlator.subReq.dbs[ygXpathInfo.dbIndex], reqXlator.subReq.dbs, currDbNum, nil, uriPath,
			reqXlator.subReq.reqUri, SUBSCRIBE, "", nil, nil, nil, reqXlator.subReq.txCache)
		return validateHandlerFunc(inParams, ygXpathInfo.validateFunc)
	}
	return true
}

func (reqXlator *subscribeReqXlator) collectChldYgXPathInfo(ygEntry *yang.Entry, ygPath string,
	rltvUriPath string, ygXpathInfo *yangXpathInfo, ygXpNode *ygXpathNode) error {

	log.Infof("%v collectChldYgXPathInfo: ygEntry: %v, ygPath: %v, rltvUriPath: %v; table name: %v; parent node path: %v",
		reqXlator.subReq.reqLogId, ygEntry, ygPath, rltvUriPath, ygXpathInfo.tableName, ygXpNode.ygPath)

	reqXlator.debugTrvsalCtxt(ygEntry, ygPath, rltvUriPath, ygXpathInfo)

	for _, childYgEntry := range ygEntry.Dir {
		childYgPath := ygPath + "/" + childYgEntry.Name
		if log.V(dbLgLvl) {
			log.Info(reqXlator.subReq.reqLogId, "collectChldYgXPathInfo: childYgPath:", childYgPath)
		}
		if chYgXpathInfo, ok := xYangSpecMap[childYgPath]; ok {

			rltvChldUriPath := rltvUriPath

			if chYgXpathInfo.nameWithMod != nil {
				rltvChldUriPath = rltvChldUriPath + "/" + *(chYgXpathInfo.nameWithMod)
			} else {
				rltvChldUriPath = rltvChldUriPath + "/" + childYgEntry.Name
			}

			var keyListMap map[string]bool

			if childYgEntry.IsList() {
				keyListMap = make(map[string]bool)
				if log.V(dbLgLvl) {
					log.Info(reqXlator.subReq.reqLogId, "collectChldYgXPathInfo: childYgEntry.Key: ", childYgEntry.Key)
				}
				keyElemNames := strings.Fields(childYgEntry.Key)

				for _, keyName := range keyElemNames {
					rltvChldUriPath = rltvChldUriPath + "[" + keyName + "=*]"
					keyListMap[keyName] = true
				}

				if log.V(dbLgLvl) {
					log.Infof("%v collectChldYgXPathInfo: keyListMap: %v, for the path: %v ",
						reqXlator.subReq.reqLogId, keyListMap, childYgPath)
				}

				chldPathUri := reqXlator.pathXlator.uriPath + rltvChldUriPath
				if !reqXlator.validateYangPath(chldPathUri, chYgXpathInfo) {
					log.Warningf("%v URI path %v is not valid since validate callback function '%v' returned false",
						reqXlator.subReq.reqLogId, chldPathUri, chYgXpathInfo.validateFunc)
					continue
				}
			}

			if chYgXpathInfo.dbIndex == db.CountersDB && !chYgXpathInfo.subscriptionFlags.Has(subsOnChangeEnable) && reqXlator.subReq.subReqMode == OnChange {
				log.Warning(reqXlator.subReq.reqLogId, "CountersDB mapped in the path: ", childYgPath)
				return tlerr.NotSupportedError{Format: "Subscribe not supported; one of its child path is mapped to COUNTERS DB and its not enabled explicitly", Path: childYgPath}
			}
			if chYgXpathInfo.subscriptionFlags.Has(subsOnChangeDisable) && reqXlator.subReq.subReqMode == OnChange {
				if log.V(dbLgLvl) {
					log.Warning(reqXlator.subReq.reqLogId, "Subscribe not supported; one of the child path's on_change subscription is disabled: ", childYgPath)
					debugPrintXPathInfo(chYgXpathInfo)
				}
				return tlerr.NotSupportedError{Format: "Subscribe not supported; one of the child path's on_change subscription is disabled", Path: childYgPath}
			}

			tblName := ""
			if (chYgXpathInfo.tableName != nil && *chYgXpathInfo.tableName != "NONE") && (chYgXpathInfo.virtualTbl == nil || !*chYgXpathInfo.virtualTbl) {
				if ygXpathInfo.tableName == nil || *ygXpathInfo.tableName != *chYgXpathInfo.tableName {
					tblName = *chYgXpathInfo.tableName
				}
			}

			if childYgEntry.IsLeaf() || childYgEntry.IsLeafList() {
				if ygXpNode.subMinIntrvl < chYgXpathInfo.subscribeMinIntvl {
					ygXpNode.subMinIntrvl = chYgXpathInfo.subscribeMinIntvl
				}
				ygEntry, ygErr := getYgEntry(reqXlator.subReq.reqLogId, ygXpNode.ygXpathInfo, ygXpNode.ygPath)
				if ygErr != nil {
					return ygErr
				}
				if ygEntry.IsList() {
					ygChldEntry, ygChldErr := getYgEntry(reqXlator.subReq.reqLogId, chYgXpathInfo, childYgPath)
					if ygChldErr != nil {
						return ygChldErr
					}
					if _, ok := ygXpNode.listKeyMap[ygChldEntry.Name]; ok {
						// for key leaf - there is no need to collect the info
						if log.V(dbLgLvl) {
							log.Info(reqXlator.subReq.reqLogId, "List key leaf node.. not collecting the info.. key leaf name: ", ygChldEntry.Name)
						}
						continue
					}
				}
				tmpXpathNode := ygXpNode
				childNodeName := childYgEntry.Name
				if tblName != "" || chYgXpathInfo.subscriptionFlags.Has(subsDelAsUpdate) {
					if log.V(dbLgLvl) {
						log.Info(reqXlator.subReq.reqLogId, "adding child ygXpNode for the table name: ", tblName, " for the leaf node for the path: ", childYgPath)
					}
					tmpXpathNode = ygXpNode.addChildNode(rltvChldUriPath, chYgXpathInfo, childYgPath)
					childNodeName = "" // leaf/leaf-list name will already part of child path
				}
				if len(chYgXpathInfo.compositeFields) > 0 && len(chYgXpathInfo.xfmrFunc) == 0 {
					if log.V(dbLgLvl) {
						log.Info(reqXlator.subReq.reqLogId, "adding composite field names: ", chYgXpathInfo.compositeFields, " for the leaf node for the path: ", childYgPath)
					}
					if err := tmpXpathNode.addDbFldNames(childNodeName, chYgXpathInfo.compositeFields); err != nil {
						return err
					}
				} else if len(chYgXpathInfo.fieldName) > 0 && len(chYgXpathInfo.xfmrFunc) == 0 {
					if log.V(dbLgLvl) {
						log.Info(reqXlator.subReq.reqLogId, "collectChldYgXPathInfo: adding field name: ", chYgXpathInfo.fieldName, " for the leaf node for the path: ", childYgPath)
					}
					ygXpNode.addDbFldName(childNodeName, chYgXpathInfo.fieldName)
				} else if len(chYgXpathInfo.xfmrFunc) == 0 {
					if log.V(dbLgLvl) {
						log.Warning(reqXlator.subReq.reqLogId, "collectChldYgXPathInfo: Adding yang node namae as db field name by default since there is no db field name mapping for the yang leaf-name: ", childYgPath)
					}
					ygXpNode.addDbFldName(childNodeName, childYgEntry.Name)
				}
			} else if childYgEntry.IsList() || childYgEntry.IsContainer() {
				chldNode := ygXpNode
				isVirtualTbl := (chYgXpathInfo.virtualTbl != nil && *chYgXpathInfo.virtualTbl)

				if len(chYgXpathInfo.xfmrFunc) > 0 {
					if log.V(dbLgLvl) {
						log.Infof("%v adding subtree xfmr func %v for the path %v ", reqXlator.subReq.reqLogId, chYgXpathInfo.xfmrFunc, childYgPath)
					}
					chldNode = ygXpNode.addChildNode(rltvChldUriPath, chYgXpathInfo, childYgPath)
				} else if tblName != "" {
					if log.V(dbLgLvl) {
						log.Infof("%v adding table name %v for the path %v ", reqXlator.subReq.reqLogId, tblName, childYgPath)
					}
					chldNode = ygXpNode.addChildNode(rltvChldUriPath, chYgXpathInfo, childYgPath)
				} else if chYgXpathInfo.xfmrTbl != nil && !isVirtualTbl {
					if log.V(dbLgLvl) {
						log.Infof("%v adding table transformer %v for the path %v ", reqXlator.subReq.reqLogId, *chYgXpathInfo.xfmrTbl, childYgPath)
					}
					chldNode = ygXpNode.addChildNode(rltvChldUriPath, chYgXpathInfo, childYgPath)
				} else {
					if childYgEntry.IsList() && !isVirtualTbl {
						if log.V(dbLgLvl) {
							log.Warning(reqXlator.subReq.reqLogId, "No table related information for the LIST yang node path: ", childYgPath)
						}
						return tlerr.InternalError{Format: "No yangXpathInfo found for the LIST / Container yang node path", Path: childYgPath}
					}
					if log.V(dbLgLvl) {
						log.Info(reqXlator.subReq.reqLogId, "Adding ygXpNode for the list node(with virtual table) / container with no tables mapped and the path: ", childYgPath)
					}
					chldNode = ygXpNode.addChildNode(rltvChldUriPath, chYgXpathInfo, childYgPath)
					chldNode.isParentTbl = true
				}

				if childYgEntry.IsList() {
					chldNode.listKeyMap = keyListMap
				}

				chldNode.ygPath = childYgPath
				if err := reqXlator.collectChldYgXPathInfo(childYgEntry, childYgPath, rltvChldUriPath, chYgXpathInfo, chldNode); err != nil {
					log.Warning(reqXlator.subReq.reqLogId, "Error in collecting the ygXpath Info for the yang path: ", childYgPath, " and the error: ", err)
					return err
				}
			}
		} else if childYgEntry.IsList() || childYgEntry.IsContainer() {
			if log.V(dbLgLvl) {
				log.Warning(reqXlator.subReq.reqLogId, "No yangXpathInfo found for the LIST / Container yang node path: ", childYgPath)
			}
			return tlerr.InternalError{Format: "No yangXpathInfo found for the LIST / Container yang node path", Path: childYgPath}
		} else {
			if log.V(dbLgLvl) {
				log.Warning(reqXlator.subReq.reqLogId, "No yangXpathInfo found for the leaf / leaf-list node yang node path: ", childYgPath)
			}
		}
	}

	return nil
}

func newYgotRootObj() (*ygot.GoStruct, error) {
	deviceObj := ocbinds.Device{}
	rootIntf := reflect.ValueOf(&deviceObj).Interface()
	ygotObj := rootIntf.(ygot.GoStruct)
	return &ygotObj, nil
}

func (reqXlator *subscribeReqXlator) GetSubscribeReqXlateInfo() (*XfmrSubscribeReqXlateInfo, error) {
	if reqXlator == nil {
		log.Errorf("error: GetSubscribeReqXlateInfo: subscribeReqXlator is nil")
		return nil, tlerr.InternalError{Format: "error: subscribeReqXlator is nil"}
	}
	return reqXlator.subReq.subReqXlateInfo, nil
}

func (reqXlator *subscribeReqXlator) uriToAbsolutePath(rltvUri string) (*gnmipb.Path, error) {
	if log.V(dbLgLvl) {
		log.Info(reqXlator.subReq.reqLogId, "uriToAbsolutePath: rltvUri: ", rltvUri)
	}
	gRelPath, err := ygot.StringToPath(rltvUri, ygot.StructuredPath)
	if err != nil {
		log.Warning(reqXlator.subReq.reqLogId, "Error in converting the URI into GNMI path for the URI: ", rltvUri)
		return nil, tlerr.InternalError{Format: "Error in converting the URI into GNMI path", Path: rltvUri}
	}
	gPath := gnmipb.Path{}
	gPath.Elem = append(gPath.Elem, reqXlator.subReq.gPath.Elem...)
	gPath.Elem = append(gPath.Elem, gRelPath.Elem...)
	if log.V(dbLgLvl) {
		log.Info(reqXlator.subReq.reqLogId, "uriToAbsolutePath: gPath: ", gPath)
	}
	return &gPath, nil
}

func debugPrintXPathInfo(xpathInfo *yangXpathInfo) {
	log.Infof("    yangType: %v\r\n", getYangTypeStrId(xpathInfo.yangType))
	log.Info("      fieldName: ", xpathInfo.fieldName)
	if xpathInfo.nameWithMod != nil {
		log.Infof("    nameWithMod : %v\r\n", *xpathInfo.nameWithMod)
	} else {
		log.Info("      nameWithMod: ", xpathInfo.nameWithMod)
	}
	log.Infof("    hasChildSubTree : %v\r\n", xpathInfo.hasChildSubTree)
	log.Infof("    hasNonTerminalNode : %v\r\n", xpathInfo.hasNonTerminalNode)
	log.Infof("    subscribeOnChg Disable  : %v\r\n", xpathInfo.subscriptionFlags.Has(subsOnChangeDisable))
	log.Infof("    subscribeMinIntvl  : %v\r\n", xpathInfo.subscribeMinIntvl)
	log.Infof("    subscribePref Sample  : %v\r\n", xpathInfo.subscriptionFlags.Has(subsPrefSample))

	log.Infof("    tableName: ")
	if xpathInfo.tableName != nil {
		log.Infof("%v", *xpathInfo.tableName)
	} else {
		log.Infof("%v", xpathInfo.tableName)
	}
	log.Infof("\r\n    virtualTbl: ")
	if xpathInfo.virtualTbl != nil {
		log.Infof("%v", *xpathInfo.virtualTbl)
	} else {
		log.Infof("%v", xpathInfo.virtualTbl)
	}
	log.Infof("\r\n    xfmrTbl  : ")
	if xpathInfo.xfmrTbl != nil {
		log.Infof("%v", *xpathInfo.xfmrTbl)
	} else {
		log.Infof("%v", xpathInfo.xfmrTbl)
	}
	log.Infof("\r\n    keyName  : ")
	if xpathInfo.keyName != nil {
		log.Infof("%v", *xpathInfo.keyName)
	} else {
		log.Infof("%v", xpathInfo.keyName)
	}
	if len(xpathInfo.childTable) > 0 {
		log.Infof("\r\n    childTbl : %v", xpathInfo.childTable)
	}
	if len(xpathInfo.fieldName) > 0 {
		log.Infof("\r\n    FieldName: %v", xpathInfo.fieldName)
	}
	log.Infof("\r\n    keyLevel : %v", xpathInfo.keyLevel)
	if len(xpathInfo.xfmrKey) > 0 {
		log.Infof("\r\n    xfmrKeyFn: %v", xpathInfo.xfmrKey)
	}
	if len(xpathInfo.xfmrFunc) > 0 {
		log.Infof("\r\n    xfmrFunc : %v", xpathInfo.xfmrFunc)
	}
	if len(xpathInfo.xfmrField) > 0 {
		log.Infof("\r\n    xfmrField :%v", xpathInfo.xfmrField)
	}
	if len(xpathInfo.xfmrPath) > 0 {
		log.Infof("\r\n    xfmrPath :%v", xpathInfo.xfmrPath)
	}
	log.Infof("\r\n    dbIndex  : %v", xpathInfo.dbIndex)

	log.Infof("\r\n    yangEntry: ")
	if xpathInfo.yangEntry != nil {
		log.Infof("%v", *xpathInfo.yangEntry)
	} else {
		log.Infof("%v", xpathInfo.yangEntry)
	}
	log.Infof("\r\n    keyXpath: %d\r\n", xpathInfo.keyXpath)
	for i, kd := range xpathInfo.keyXpath {
		log.Infof("        %d : xpathInfo. %#v\r\n", i, kd)
	}
	log.Infof("\r\n    isKey   : %v\r\n", xpathInfo.isKey)
	log.Info("      delim: ", xpathInfo.delim)
}

func getYgEntry(reqLogId string, ygXpath *yangXpathInfo, ygPath string) (*yang.Entry, error) {
	ygEntry := ygXpath.yangEntry
	if ygEntry == nil && (ygXpath.yangType == YANG_LEAF || ygXpath.yangType == YANG_LEAF_LIST) {
		ygEntry = getYangEntryForXPath(ygPath)
	}
	if ygEntry == nil {
		if log.V(dbLgLvl) {
			log.Warningf("%v : yangEntry is nil in the yangXpathInfo for the path:", reqLogId, ygPath)
		}
		return nil, tlerr.InternalError{Format: "yangXpathInfo has nil value for its yangEntry field", Path: ygPath}
	}
	return ygEntry, nil
}

func (keyRslvr *DbYangKeyResolver) handleValueXfmr(xfmrName string, operation Operation, keyName string, keyVal string) (keyLeafVal string, err error) {
	if log.V(dbLgLvl) {
		log.Info(keyRslvr.reqLogId, "resolveDbKey: keyLeafRefNode xfmrValue; ", xfmrName)
	}
	inParams := formXfmrDbInputRequest(operation, keyRslvr.dbIdx, keyRslvr.tableName, strings.Join(keyRslvr.key.Comp, keyRslvr.delim), keyName, keyVal)
	keyLeafVal, err = valueXfmrHandler(inParams, xfmrName)
	if err != nil {
		return
	}
	if log.V(dbLgLvl) {
		log.Info(keyRslvr.reqLogId, "resolveDbKey: resolved Db key value; ", keyLeafVal)
	}
	return
}

func (keyRslvr *DbYangKeyResolver) resolveDbKey(keyList []string, operation Operation) ([]string, error) {
	var keyComp []string
	for idx := 0; idx < len(keyList); idx++ {
		keyPath := keyRslvr.tableName + "/" + keyList[idx]
		keyYgNode, ok := xDbSpecMap[keyPath]
		if !ok || keyYgNode == nil {
			log.Warningf("%v resolveDbKey: Db yang node not found in the xDbSpecMap for the path: %v", keyRslvr.reqLogId, keyPath)
			continue
		}
		if log.V(dbLgLvl) {
			log.Info(keyRslvr.reqLogId, "resolveDbKey: keyYgNode; ", *keyYgNode)
		}
		keyLeafVal := keyRslvr.key.Comp[idx]
		var err error
		if keyYgNode.xfmrValue != nil && len(*keyYgNode.xfmrValue) > 0 {
			keyLeafVal, err = keyRslvr.handleValueXfmr(*keyYgNode.xfmrValue, operation, keyList[idx], keyRslvr.key.Comp[idx])
			if err != nil {
				return keyComp, fmt.Errorf("%v resolveDbKey: Error in valueXfmrHandler: %v; given keyPath: %v; keyLeafVal: %v for the "+
					"uri path %v", keyRslvr.reqLogId, err, keyPath, keyLeafVal, keyRslvr.uriPath)
			}
		} else {
			for _, leafRefPath := range keyYgNode.leafRefPath {
				keyLeafRefNode, ok := xDbSpecMap[leafRefPath]
				if !ok || keyLeafRefNode == nil {
					log.Warningf("%v resolveDbKey: Db yang key leafref node not found in the xDbSpecMap for the path: ", keyRslvr.reqLogId, leafRefPath)
					continue
				}
				if keyLeafRefNode.xfmrValue != nil && len(*keyLeafRefNode.xfmrValue) > 0 {
					keyLeafVal, err = keyRslvr.handleValueXfmr(*keyLeafRefNode.xfmrValue, operation, keyList[idx], keyRslvr.key.Comp[idx])
					if len(keyLeafVal) == 0 || err != nil {
						log.Warningf("%v resolveDbKey: valueXfmrHandler error: %v; given keyPath: %v; keyLeafVal: %v for the "+
							"uri path %v", keyRslvr.reqLogId, err, keyPath, keyLeafVal, keyRslvr.uriPath)
						continue
					} else {
						break
					}
				}
			}
		}
		keyComp = append(keyComp, keyLeafVal)
	}
	if len(keyComp) > 0 {
		return keyComp, nil
	}
	log.Warningf("%v resolveDbKey: Could not resolve the db key %v; yang schema not found for the table %v "+
		"for the path: %v", keyRslvr.reqLogId, keyRslvr.key.Comp, keyRslvr.tableName, keyRslvr.uriPath)
	return keyRslvr.key.Comp, nil
}

func (keyRslvr *DbYangKeyResolver) resolve(operation Operation) ([]string, error) {
	ygDbInfo, dbYgEntry, err := keyRslvr.getDbYangNode()
	if err != nil {
		if log.V(dbLgLvl) {
			log.Warningf("%v DbYangKeyResolver: xDbSpecMap does not have the dbInfo entry for the table: %v", keyRslvr.reqLogId, keyRslvr.tableName)
		}
		if keyRslvr.dbIdx < db.MaxDB {
			keyRslvr.delim = keyRslvr.dbs[keyRslvr.dbIdx].Opts.KeySeparator
		}
		return keyRslvr.key.Comp, nil
	}

	keyRslvr.dbIdx = ygDbInfo.dbIndex
	keyRslvr.delim = ygDbInfo.delim
	if len(keyRslvr.delim) == 0 && keyRslvr.dbIdx < db.MaxDB {
		keyRslvr.delim = keyRslvr.dbs[keyRslvr.dbIdx].Opts.KeySeparator
	}

	if !ygDbInfo.hasXfmrFn && !hasKeyValueXfmr(keyRslvr.tableName) {
		return keyRslvr.key.Comp, nil
	}

	for _, listName := range ygDbInfo.listName {
		dbYgListInfo, ygDbListNode, err := keyRslvr.getDbYangListInfo(listName)
		if err != nil {
			return nil, err
		}
		if !ygDbListNode.IsList() {
			log.Warningf("%v DbYangKeyResolver: resolve: list name %v is not found in the xDbSpecMap as yang list node for the table: %v", keyRslvr.reqLogId, listName, keyRslvr.tableName)
			continue
		}
		keyList := strings.Fields(ygDbListNode.Key)
		if log.V(dbLgLvl) {
			log.Info("keyList: ", keyList)
		}
		if len(keyList) != keyRslvr.key.Len() {
			log.Warningf("%v DbYangKeyResolver: db table %v key count %v and yang model list %v key count %v does not match",
				keyRslvr.reqLogId, keyRslvr.tableName, keyRslvr.key.Len(), ygDbListNode.Name, len(keyList))
			continue
		}
		if len(dbYgListInfo.delim) > 0 {
			keyRslvr.delim = dbYgListInfo.delim
		} else if dbYgListInfo.dbIndex < db.MaxDB {
			keyRslvr.delim = keyRslvr.dbs[dbYgListInfo.dbIndex].Opts.KeySeparator
		} else if len(keyRslvr.delim) == 0 && dbYgEntry.Config != yang.TSFalse {
			keyRslvr.delim = "|"
		}
		if len(keyRslvr.delim) == 0 {
			return nil, tlerr.InternalError{Format: keyRslvr.reqLogId + "Could not form db key, since key-delim or " +
				"db-name annotation is missing from the sonic yang model container", Path: dbYgEntry.Name}
		}
		return keyRslvr.resolveDbKey(keyList, operation)
	}
	log.Warningf("%v DbYangKeyResolver: Db yang matching list node not found for the table %v for the path: %v", keyRslvr.reqLogId, keyRslvr.tableName, keyRslvr.uriPath)
	return keyRslvr.key.Comp, nil
}

func (keyRslvr *DbYangKeyResolver) getDbYangNode() (*dbInfo, *yang.Entry, error) {
	dbInfo, ok := xDbSpecMap[keyRslvr.tableName]
	if !ok || dbInfo == nil {
		if log.V(dbLgLvl) {
			log.Warning(keyRslvr.reqLogId, "xDbSpecMap does not have the dbInfo entry for the table:", keyRslvr.tableName)
		}
		return nil, nil, tlerr.InternalError{Format: keyRslvr.reqLogId + "xDbSpecMap does not have the dbInfo entry for the table " + keyRslvr.tableName, Path: keyRslvr.uriPath}
	}
	ygEntry, _ := getYgDbEntry(keyRslvr.reqLogId, dbInfo, keyRslvr.tableName)
	if ygEntry == nil {
		if log.V(dbLgLvl) {
			log.Warning(keyRslvr.reqLogId, "dbInfo has nil value for its yangEntry field for the table:", keyRslvr.tableName)
		}
		return nil, ygEntry, tlerr.InternalError{Format: keyRslvr.reqLogId + "dbInfo has nil value for its yangEntry field for the table " + keyRslvr.tableName, Path: keyRslvr.uriPath}
	}
	return dbInfo, ygEntry, nil
}

func (keyRslvr *DbYangKeyResolver) getDbYangListInfo(listName string) (*dbInfo, *yang.Entry, error) {
	dbListkey := keyRslvr.tableName + "/" + listName
	if log.V(dbLgLvl) {
		log.Info(keyRslvr.reqLogId, ": DbYangKeyResolver: getDbYangListInfo: dbListkey: ", dbListkey)
	}
	dbListInfo, ok := xDbSpecMap[dbListkey]
	if !ok {
		return nil, nil, tlerr.InternalError{Format: keyRslvr.reqLogId + "xDbSpecMap does not have the dbInfo entry for the table " + keyRslvr.tableName, Path: keyRslvr.uriPath}
	}
	ygEntry, _ := getYgDbEntry(keyRslvr.reqLogId, dbListInfo, dbListkey)
	if ygEntry == nil {
		return nil, nil, tlerr.InternalError{Format: keyRslvr.reqLogId + "dbInfo has nil value for its yangEntry field for the table " + keyRslvr.tableName, Path: keyRslvr.uriPath}
	}
	if ygEntry.IsList() {
		return dbListInfo, ygEntry, nil
	}
	return nil, ygEntry, tlerr.InternalError{Format: keyRslvr.reqLogId + "dbListInfo is not a Db yang LIST node for the listName " + listName}
}

func getYgDbEntry(reqLogId string, ygDbInfo *dbInfo, ygPath string) (*yang.Entry, error) {
	ygEntry := ygDbInfo.dbEntry
	if ygEntry == nil && (ygDbInfo.yangType == YANG_LEAF || ygDbInfo.yangType == YANG_LEAF_LIST) {
		ygEntry = getYangEntryForXPath(ygPath)
	}
	if ygEntry == nil {
		if log.V(dbLgLvl) {
			log.Warningf("%v : yangEntry is nil in the yangXpathInfo for the path:", reqLogId, ygPath)
		}
		return nil, tlerr.InternalError{Format: "yangXpathInfo has nil value for its yangEntry field", Path: ygPath}
	}
	return ygEntry, nil
}
