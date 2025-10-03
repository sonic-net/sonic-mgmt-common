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
	"os"
	"reflect"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

var ocbSch, _ = ocbinds.GetSchema()

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

func processDataToResultMap(xlateParams xlateToParams, tblField string, tblFieldVal string, isDefaultValueProcessingFlow bool) {
	/* isDefaultValueProcessingFlow is set to true only from Replace default value processing flow to avoid invocation of
	   dataToDBMapForReplace() since there is no need to make table-ownership decision for an instance whose ownership is
	   already decided and the instance belongs in appropriate oper(REPLACE/UPDATE) map(passed as xlateParams.result here).
	*/
	if xlateParams.oper == REPLACE && (!isDefaultValueProcessingFlow) {
		dataToDBMapForReplace(xlateParams, tblField, tblFieldVal)
	} else if xlateParams.oper == DELETE && xlateParams.replaceInfo != nil && xlateParams.replaceInfo.isDeleteForReplace {
		addToMap := false
		if xlateParams.replaceInfo.skipFieldSiblingTraversalForDelete != nil {
			if *xlateParams.replaceInfo.skipFieldSiblingTraversalForDelete {
				// This can happen if the field xfmr returns more fields for a single oc field. If this flag is already set it indicates that it has been evaluated for the earlier field and we need to skip sibling processing. Hence do not process this field.
				return
			}
			addToMap, *xlateParams.replaceInfo.skipFieldSiblingTraversalForDelete = addToDeleteForReplaceMap(xlateParams.tableName, xlateParams.keyName, tblField, xlateParams.resultMap)
			if addToMap {
				dataToDBMapAdd(xlateParams.tableName, xlateParams.keyName, xlateParams.result, tblField, tblFieldVal)
			}
		}
	} else {
		dataToDBMapAdd(xlateParams.tableName, xlateParams.keyName, xlateParams.result, tblField, tblFieldVal)
	}
}

/*use when single table name is expected*/
func tblNameFromTblXfmrGet(xfmrTblFunc string, inParams XfmrParams, xfmrDbTblKeyCache map[string]tblKeyCache) (string, error) {
	var err error
	var tblList []string
	tblList, err = xfmrTblHandlerFunc(xfmrTblFunc, inParams, xfmrDbTblKeyCache)
	if err != nil {
		return "", err
	}
	if len(tblList) != 1 {
		xfmrLogDebug("Uri (\"%v\") translates to 0 or multiple tables instead of single table - %v", inParams.uri, tblList)
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
	xfmrLogDebug("name: \"%v\", xpathPrefix(\"%v\").", xlateParams.name, xlateParams.xpath)

	if !ok || xpathInfo == nil {
		log.Warningf("Yang path(\"%v\") not found.", xpath)
		return nil
	}

	if xpathInfo.tableName == nil && xpathInfo.xfmrTbl == nil {
		log.Warningf("Table for yang-path(\"%v\") not found.", xpath)
		return nil
	}

	if xpathInfo.tableName != nil && *xpathInfo.tableName == XFMR_NONE_STRING {
		log.Warningf("Table for yang-path(\"%v\") NONE.", xpath)
		return nil
	}

	if len(xlateParams.keyName) == 0 {
		log.Warningf("Table key for YANG path(\"%v\") not found.", xpath)
		return nil
	}

	tableName := ""
	isNotTblOwner := false
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
			log.Warningf("No table name found for URI (\"%v\")", xlateParams.uri)
			return err
		}
		if inParams.isNotTblOwner != nil {
			isNotTblOwner = *inParams.isNotTblOwner
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
	curXlateParams.isNotTblOwner = isNotTblOwner
	curXlateParams.xpath = xpath
	err = mapFillDataUtil(curXlateParams, false)
	return err
}

func mapFillDataUtil(xlateParams xlateToParams, isDefaultValueProcessingFlow bool) error {
	var dbs [db.MaxDB]*db.DB

	xpathInfo, ok := xYangSpecMap[xlateParams.xpath]
	if !ok {
		errStr := fmt.Sprintf("Invalid yang-path(\"%v\").", xlateParams.xpath)
		return tlerr.InternalError{Format: errStr}
	}

	if len(xpathInfo.xfmrField) > 0 {
		xlateParams.uri = xlateParams.uri + "/" + xlateParams.name

		/* field transformer present */
		xfmrLogDebug("xfmr function(\"%v\") invoked for YANG path(\"%v\"). uri: %v", xpathInfo.xfmrField, xlateParams.xpath, xlateParams.uri)
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
			xfmrLogDebug("xfmr function : %v Xpath : %v retData: %v", xpathInfo.xfmrField, xlateParams.xpath, retData)
			for f, v := range retData {
				// Ideally the translated field should be single field
				processDataToResultMap(xlateParams, f, v, isDefaultValueProcessingFlow)
			}
		}
		return nil
	}

	if len(xpathInfo.fieldName) == 0 {
		if xpathInfo.isRefByKey || (xpathInfo.isKey && (strings.HasPrefix(xlateParams.uri, "/"+IETF_MDL_PFX) || xlateParams.oper == REPLACE)) {
			processDataToResultMap(xlateParams, "NULL", "NULL", isDefaultValueProcessingFlow)
			xfmrLogDebug("%v - Either an OC YANG leaf path referenced by list key or IETF YANG list key, ", xlateParams.xpath,
				"maps to no actual field in sonic yang so dummy row added to create instance in DB.")
		} else {
			xfmrLogInfo("Field for YANG path(\"%v\") not found in DB.", xlateParams.xpath)
		}
		return nil
	}
	fieldName := xpathInfo.fieldName
	valueStr := ""

	fieldXpath := xlateParams.tableName + "/" + fieldName
	xDbSpecInfo, ok := xDbSpecMap[fieldXpath]
	dbEntry := getYangEntryForXPath(fieldXpath)
	if !ok || (xDbSpecInfo == nil) || (dbEntry == nil) {
		logStr := fmt.Sprintf("Failed to find the xDbSpecMap: xpath(\"%v\").", fieldXpath)
		log.Warning(logStr)
		return nil
	}
	if xDbSpecInfo.isKey {
		if xpathInfo.isRefByKey || (xpathInfo.isKey && strings.HasPrefix(xlateParams.uri, "/"+IETF_MDL_PFX)) {
			// apps use this leaf in payload to create an instance, redis needs atleast one field:val to create an instance
			processDataToResultMap(xlateParams, "NULL", "NULL", isDefaultValueProcessingFlow)
			xfmrLogDebug("%v - Either an OC YANG leaf path referenced by list key or IETF YANG list key, ", xlateParams.xpath,
				"maps to key in sonic YANG - %v so dummy row added to create instance in DB.", fieldXpath)
		} else {
			xfmrLogInfo("OC YANG leaf path(%v), maps to key in sonic YANG - %v which cannot be filled as a field in DB instance", xlateParams.xpath, fieldXpath)
		}
		return nil
	}
	if xpathInfo.yangType == YANG_LEAF_LIST {
		/* Both YANG side and DB side('@' suffix field) the data type is leaf-list */
		xfmrLogDebug("Yang type and DB type is Leaflist for field  = %v", xlateParams.xpath)
		fieldName += "@"
		if reflect.ValueOf(xlateParams.value).Kind() != reflect.Slice {
			logStr := fmt.Sprintf("Value for YANG xpath %v which is a leaf-list should be a slice", xlateParams.xpath)
			log.Warning(logStr)
			return nil
		}
		valData := reflect.ValueOf(xlateParams.value)
		for fidx := 0; fidx < valData.Len(); fidx++ {
			if fidx > 0 {
				valueStr += ","
			}

			// SNC-3626 - string conversion based on the primitive type
			fVal, err := unmarshalJsonToDbData(dbEntry, fieldXpath, fieldName, valData.Index(fidx).Interface())
			if err == nil {
				if (strings.Contains(fVal, ":")) && (strings.HasPrefix(fVal, OC_MDL_PFX) || strings.HasPrefix(fVal, IETF_MDL_PFX) || strings.HasPrefix(fVal, IANA_MDL_PFX)) {
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
		xfmrLogDebug("leaf-list value after conversion to DB format %v  :  %v", fieldName, valueStr)

	} else { // xpath is a leaf

		// SNC-3626 - string conversion based on the primitive type
		fVal, err := unmarshalJsonToDbData(dbEntry, fieldXpath, fieldName, xlateParams.value)
		if err == nil {
			valueStr = fVal
		} else {
			logStr := fmt.Sprintf("Couldn't unmarshal Json to DbData: table(\"%v\") field(\"%v\").", xlateParams.tableName, fieldName, xlateParams.value)
			log.Warning(logStr)
			return nil
		}

		if (strings.Contains(valueStr, ":")) && (strings.HasPrefix(valueStr, OC_MDL_PFX) || strings.HasPrefix(valueStr, IETF_MDL_PFX) || strings.HasPrefix(valueStr, IANA_MDL_PFX)) {
			// identity-ref/enum might has module prefix
			valueStr = strings.SplitN(valueStr, ":", 2)[1]
		}
	}

	processDataToResultMap(xlateParams, fieldName, valueStr, isDefaultValueProcessingFlow)
	xfmrLogDebug("TblName: \"%v\", key: \"%v\", field: \"%v\".", xlateParams.tableName, xlateParams.keyName, fieldName)
	return nil
}

func sonicYangReqToDbMapCreate(xlateParams xlateToParams) error {
	/* This function processeses soic yang CRU request and payload to DB format */
	xfmrLogDebug("About to process uri: \"%v\".", xlateParams.requestUri)
	topLevelContainerMap, contOk := xlateParams.jsonData.(map[string]interface{})
	if !contOk {
		errStr := fmt.Sprintf("Unexpected JSON format for URI \"%v\".", xlateParams.requestUri)
		xfmrLogInfo("%v", errStr)
		return tlerr.InternalError{Format: errStr}
	}
	moduleNm, err := uriModuleNameGet(xlateParams.requestUri)
	if err != nil {
		return tlerr.InternalError{Format: err.Error()}
	}
	topContNdNmInJson := moduleNm + ":" + moduleNm
	topLevelContainerInterface, contIntfOk := topLevelContainerMap[topContNdNmInJson]
	if !contIntfOk {
		errStr := fmt.Sprintf("Module %v not found in JSON for URI \"%v\".", topContNdNmInJson, xlateParams.requestUri)
		xfmrLogInfo("%v", errStr)
		return tlerr.InternalError{Format: errStr}
	}

	topLevelContainerData, contDataOk := topLevelContainerInterface.(map[string]interface{})
	if !contDataOk {
		errStr := fmt.Sprintf("Unexpected JSON format value of module %v for URI \"%v\".", moduleNm, xlateParams.requestUri)
		xfmrLogInfo("%v", errStr)
		return tlerr.InternalError{Format: errStr}
	}

	for tableLevelContainerName, valueInterface := range topLevelContainerData {
		if _, ok := xDbSpecMap[tableLevelContainerName]; ok {
			directDbMapData(xlateParams.requestUri, tableLevelContainerName, valueInterface, xlateParams.result, xlateParams.yangDefValMap)
		} else {
			xfmrLogInfo("table \"%v\" under %v is not in transformer sonic yang spec map.", tableLevelContainerName, moduleNm)
		}
	}

	return nil
}

func dbMapDataFill(uri string, tableName string, keyName string, d map[string]interface{}, result map[string]map[string]db.Value) {

	for field, value := range d {
		fieldXpath := tableName + "/" + field

		dbEntry := getYangEntryForXPath(fieldXpath)

		if _, fieldOk := xDbSpecMap[fieldXpath]; fieldOk && dbEntry != nil {
			xfmrLogDebug("Found non-nil YANG entry in xDbSpecMap for field xpath = %v", fieldXpath)
			if dbEntry.IsLeafList() {
				xfmrLogDebug("Yang type is Leaflist for field  = %v", field)
				field += "@"
				fieldDt := reflect.ValueOf(value)
				fieldValue := ""
				for fidx := 0; fidx < fieldDt.Len(); fidx++ {
					if fidx > 0 {
						fieldValue += ","
					}
					fVal, err := unmarshalJsonToDbData(dbEntry, fieldXpath, field, fieldDt.Index(fidx).Interface())
					if err != nil {
						log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
					} else {
						fieldValue = fieldValue + fVal
					}
				}
				dataToDBMapAdd(tableName, keyName, result, field, fieldValue)
				continue
			}
			dbval, err := unmarshalJsonToDbData(dbEntry, fieldXpath, field, value)
			if err != nil {
				log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
			} else {
				dataToDBMapAdd(tableName, keyName, result, field, dbval)
			}
		} else {
			// should ideally never happen , just adding for safety
			xfmrLogDebug("Did not find entry in xDbSpecMap for field xpath = %v", fieldXpath)
		}
	}
}

func dbMapDataFillForNestedList(tableName string, key string, nestedListDbEntry *yang.Entry, jsonData interface{}, result map[string]map[string]db.Value) {
	/*function to process CRU request for nested/child list of list under table level container in sonic yang.*/

	/*As per current sonic yang structure in community nested list has only one key leaf that
	  corresponds to dynamic field-name case.
	*/
	nestedListYangKeyName := strings.Split(nestedListDbEntry.Key, " ")[0]
	nestedListData, ok := jsonData.([]interface{}) //nested list slice/array containing its instances
	if !ok {
		log.Warningf("Invalid nested list data.")
		return
	}
	for _, instance := range nestedListData {
		nestedListInstance, ok := instance.(map[string]interface{}) //child/nested list instance
		if !ok {
			xfmrLogInfo("Invalid nested list instance type")
			continue
		}
		fieldXpath := tableName + "/" + nestedListYangKeyName
		fieldDbEntry := nestedListDbEntry.Dir[nestedListYangKeyName]
		fieldName, err := unmarshalJsonToDbData(fieldDbEntry, fieldXpath, nestedListYangKeyName, nestedListInstance[nestedListYangKeyName])
		if err != nil {
			log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\"), value %v,  error (\"%v\").", fieldXpath, nestedListInstance[nestedListYangKeyName], err)
			continue
		}
		for field, value := range nestedListInstance {
			if field == nestedListYangKeyName {
				continue
			}
			fieldValXpath := tableName + "/" + field
			fieldDbEntry := nestedListDbEntry.Dir[field]
			fieldValue, err := unmarshalJsonToDbData(fieldDbEntry, fieldValXpath, field, value)
			if err != nil {
				log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\"), value %v error (\"%v\").", fieldValXpath, value, err)
				break
			} else {
				dataToDBMapAdd(tableName, key, result, fieldName, fieldValue)
			}
		}

	}

}
func dbMapTableChildListDataFill(uri string, tableName string, childListNames []string, dbEntry *yang.Entry, jsonData interface{}, result map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value) {
	data := reflect.ValueOf(jsonData)
	tblKeyName := strings.Split(dbEntry.Key, " ")
	hasNestedList := len(childListNames) > 0

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

		if hasNestedList {
			for _, nestedListNm := range childListNames {
				if nestedListData, nestedListPresentInPayload := d[nestedListNm]; nestedListPresentInPayload {
					dbMapDataFillForNestedList(tableName, keyName, dbEntry.Dir[nestedListNm], nestedListData, result)
					delete(d, nestedListNm)
				}

			}
		}
		dbMapDataFill(uri, tableName, keyName, d, result)
		sonicDefaultFieldValFill(dbEntry, tableName, keyName, result, yangDefValMap)

		if len(result[tableName][keyName].Field) == 0 {
			dataToDBMapAdd(tableName, keyName, result, "NULL", "NULL") // redis needs atleast one field:val to create an instance
		}
	}
}

func dbMapTableChildContainerDataFill(uri string, tableName string, dbEntry *yang.Entry, jsonData interface{}, result map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value) {
	data := reflect.ValueOf(jsonData).Interface().(map[string]interface{})
	keyName := dbEntry.Name
	xfmrLogDebug("Container name %v will become table key.", keyName)
	dbMapDataFill(uri, tableName, keyName, data, result)
	sonicDefaultFieldValFill(dbEntry, tableName, keyName, result, yangDefValMap)
	if len(result[tableName][keyName].Field) == 0 {
		dataToDBMapAdd(tableName, keyName, result, "NULL", "NULL") // redis needs atleast one field:val to create an instance
	}
}

func sonicDefaultFieldValFill(parentDbEntry *yang.Entry, tableName string, tableKey string,
	result map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value) {
	if parentDbEntry == nil {
		xfmrLogDebug("dbEntry whose children with default values are to be found is nil")
		return
	}
	for field := range parentDbEntry.Dir {
		fieldXpath := tableName + "/" + field
		dbEntry := parentDbEntry.Dir[field]
		if _, fieldOk := xDbSpecMap[fieldXpath]; fieldOk && (len(dbEntry.Default) > 0) {
			if _, ok := result[tableName][tableKey].Field[field]; !ok {
				fVal, err := unmarshalJsonToDbData(dbEntry, fieldXpath, field, dbEntry.Default)
				if err != nil {
					xfmrLogDebug("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
				} else {
					dataToDBMapAdd(tableName, tableKey, yangDefValMap, field, fVal)
				}
			}
		}
	}

}

func directDbMapData(uri string, tableName string, jsonData interface{}, result map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value) bool {
	dbSpecData, ok := xDbSpecMap[tableName]
	if ok && dbSpecData.dbEntry != nil {
		data := reflect.ValueOf(jsonData).Interface().(map[string]interface{})
		key := ""
		dbSpecData := xDbSpecMap[tableName]
		result[tableName] = make(map[string]db.Value)

		if dbSpecData.keyName != nil {
			key = *dbSpecData.keyName
			xfmrLogDebug("Fill data for container uri(%v), key(%v)", uri, key)
			dbMapDataFill(uri, tableName, key, data, result)
			sonicDefaultFieldValFill(dbSpecData.dbEntry, tableName, key, result, yangDefValMap)
			return true
		}

		for k, v := range data {
			xpath := tableName + "/" + k
			curDbSpecData, ok := xDbSpecMap[xpath]
			if ok && curDbSpecData.dbEntry != nil {
				eType := curDbSpecData.yangType
				switch eType {
				case YANG_LIST:
					xfmrLogDebug("Fill data for list %v child of table level node %v", k, tableName)
					dbMapTableChildListDataFill(uri, tableName, curDbSpecData.listName, curDbSpecData.dbEntry, v, result, yangDefValMap)
				case YANG_CONTAINER:
					xfmrLogDebug("Fill data for container %v child of table level node %v", k, tableName)
					dbMapTableChildContainerDataFill(uri, tableName, curDbSpecData.dbEntry, v, result, yangDefValMap)
				default:
					xfmrLogDebug("Invalid node type for uri(%v)", uri)
				}
			}
		}
	}
	return true
}

/* Get the data from incoming update/replace request, create map and fill with dbValue(ie. field:value to write into redis-db */
func dbMapUpdate(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, path string, requestUri string, jsonData interface{}, result map[Operation]map[db.DBNum]map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value, yangAuxValMap map[string]map[string]db.Value, txCache interface{}) error {
	xfmrLogInfo("Update/replace req: path(\"%v\").", path)

	err := dbMapCreate(d, ygRoot, oper, path, requestUri, jsonData, result, yangDefValMap, yangAuxValMap, txCache)
	printDbData(result, nil, "/tmp/yangToDbDataUpRe.txt")
	return err
}

func dbMapDefaultFieldValFill(xlateParams xlateToParams, tblUriList []string) error {
	tblData := xlateParams.result[xlateParams.tableName]
	var dbs [db.MaxDB]*db.DB
	tblName := xlateParams.tableName
	dbKey := xlateParams.keyName
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
								childYangDataType := childNodeYangEntry.Type.Kind
								var param interface{}
								oper := xlateParams.oper
								if len(childNode.defVal) > 0 {
									if tblXfmrPresent {
										chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
										xfmrLogDebug("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
										if ctErr != nil || chldTblNm != tblName {
											continue
										}
									}
									xfmrLogDebug("Update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
										childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
									_, defValPtr, err := DbToYangType(childYangDataType, childXpath, childNode.defVal, xlateParams.oper)
									if err == nil && defValPtr != nil {
										param = defValPtr
									} else {
										xfmrLogDebug("Failed to update(\"%v\") default: tbl[\"%v\"]key[\"%v\"]fld[\"%v\"] = val(\"%v\").",
											childXpath, tblName, dbKey, childNode.fieldName, childNode.defVal)
									}

									inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, tblUri+"/"+childName, xlateParams.requestUri, oper, "", nil, xlateParams.subOpDataMap, param, xlateParams.txCache)
									retData, err := leafXfmrHandler(inParams, childNode.xfmrField)
									if err != nil {
										log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.xfmrField)
									}
									if retData != nil {
										xfmrLogDebug("xfmr function : %v Xpath: %v retData: %v", childNode.xfmrField, childXpath, retData)
										for f, v := range retData {
											// Fill default value only if value is not available in result Map
											// else we overwrite the value filled in resultMap with default value
											_, ok := xlateParams.result[tblName][dbKey].Field[f]
											if !ok {
												if len(childNode.defVal) > 0 {
													dataToDBMapAdd(tblName, dbKey, xlateParams.yangDefValMap, f, v)
												}
											}
										}
									}
								}
							} else if len(childNode.fieldName) > 0 {
								var xfmrErr error
								if xDbSpecInfo, ok := xDbSpecMap[tblName+"/"+childNode.fieldName]; ok && (xDbSpecInfo != nil) && (!xDbSpecInfo.isKey) {
									if tblXfmrPresent {
										chldTblNm, ctErr := tblNameFromTblXfmrGet(*childNode.xfmrTbl, inParamsTblXfmr, xlateParams.xfmrDbTblKeyCache)
										xfmrLogDebug("Table transformer %v for xpath %v returned table %v", *childNode.xfmrTbl, childXpath, chldTblNm)
										if ctErr != nil || chldTblNm != tblName {
											continue
										}
									}
									// Fill default value only if value is not available in result Map
									// else we overwrite the value filled in resultMap with default value
									_, ok = xlateParams.result[tblName][dbKey].Field[childNode.fieldName]
									if !ok {
										if len(childNode.defVal) > 0 {
											curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, childXpath, dbKey, xlateParams.jsonData, xlateParams.resultMap, xlateParams.yangDefValMap, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, &xfmrErr, childName, childNode.defVal, tblName, xlateParams.isNotTblOwner, xlateParams.invokeCRUSubtreeOnceMap, nil, nil, xlateParams.replaceInfo)
											err := mapFillDataUtil(curXlateParams, false)
											if err != nil {
												log.Warningf("Default/AuxMap Value filling. Received error %v from %v", err, childNode.fieldName)
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
	}
	return nil
}

func dbMapDefaultValFill(xlateParams xlateToParams) error {
	for tbl, tblData := range xlateParams.result {
		if _, tblOk := xlateParams.tblXpathMap[tbl]; !tblOk {
			continue
		}
		for dbKey := range tblData {
			var yxpathList []string //contains all uris(with keys) that were traversed for a table while processing the incoming request
			if tblUriMapVal, ok := xlateParams.tblXpathMap[tbl][dbKey]; ok {
				for tblUri := range tblUriMapVal {
					yxpathList = append(yxpathList, tblUri)
				}
			}
			if len(yxpathList) > 0 {
				curXlateParams := xlateParams
				curXlateParams.tableName = tbl
				curXlateParams.keyName = dbKey
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
func dbMapCreate(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, requestUri string, jsonData interface{}, resultMap map[Operation]RedisDbMap, yangDefValMap map[string]map[string]db.Value, yangAuxValMap map[string]map[string]db.Value, txCache interface{}) error {
	var err, xfmrErr error
	var cascadeDelTbl []string
	var result = make(map[string]map[string]db.Value)
	var skipDelete *bool
	var isDeleteForReplace, isNonTblOwnerDefaultValProcess bool
	tblXpathMap := make(map[string]map[string]map[string]bool)
	subOpDataMap := make(map[Operation]*RedisDbMap)
	root := xpathRootNameGet(uri)
	yangAuxValOper := oper
	invokeSubtreeOnceMap := make(map[string]map[string]bool)
	replaceSubtreeMap := make(map[string]bool)
	subOpDataMapForReplace := make(map[Operation]*RedisDbMap)
	targetHasNonTerminalNode := true

	/* Check if the parent table exists for RFC compliance */
	var exists bool
	var dbs [db.MaxDB]*db.DB
	var replaceInfo replaceProcessingInfo
	subOpMapDiscard := make(map[Operation]*RedisDbMap)
	exists, err = verifyParentTable(d, dbs, ygRoot, oper, uri, nil, txCache, subOpMapDiscard, nil)
	xfmrLogDebug("verifyParentTable() returned - exists - %v, err - %v", exists, err)
	if err != nil {
		log.Warningf("Cannot perform Operation %v on URI %v due to - %v", oper, uri, err)
		return err
	}
	if !exists {
		errStr := fmt.Sprintf("Parent table does not exist for uri(%v)", uri)
		return tlerr.NotFoundError{Format: errStr}
	}

	moduleNm := "/" + strings.Split(uri, "/")[1]
	xfmrLogInfo("Module name for URI %s is %s", uri, moduleNm)

	if isSonicYang(uri) {
		xlateToData := formXlateToDbParam(d, ygRoot, oper, root, uri, "", "", jsonData, resultMap, result, txCache, tblXpathMap, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "", false, nil, yangDefValMap, nil, nil)
		err = sonicYangReqToDbMapCreate(xlateToData)
		xpathPrefix, keyName, tableName := sonicXpathKeyExtract(uri)
		xfmrLogDebug("xpath - %v, keyName - %v, tableName - %v , for URI - %v", xpathPrefix, keyName, tableName, uri)
		fldPth := strings.Split(xpathPrefix, "/")
		if (len(fldPth) > SONIC_FIELD_INDEX) && (oper == REPLACE) {
			fldNm := fldPth[SONIC_FIELD_INDEX]
			xfmrLogDebug("Field Name : %v", fldNm)
			if fldNm != "" {
				_, ok := xDbSpecMap[tableName]
				if ok {
					dbSpecField := tableName + "/" + fldNm
					dbSpecInfo, dbFldok := xDbSpecMap[dbSpecField]
					if dbFldok && dbSpecInfo != nil {
						if dbSpecInfo.yangType == YANG_LEAF {
							resultMap[UPDATE] = make(RedisDbMap)
							resultMap[UPDATE][db.ConfigDB] = result
						} else if dbSpecInfo.yangType == YANG_LEAF_LIST {
							resultMap[REPLACE] = make(RedisDbMap)
							resultMap[REPLACE][db.ConfigDB] = result
						} else {
							log.Warningf("For URI - %v, unrecognized terminal YANG node type", uri)
						}
					} else if !dbFldok { //check for nested list case
						nestedChildName := fldNm
						dbSpecPath := tableName + "/" + fldPth[SONIC_TBL_CHILD_INDEX] + "/" + nestedChildName
						dbSpecNestedChildInfo, ok := xDbSpecMap[dbSpecPath]
						if ok && dbSpecNestedChildInfo != nil {
							if dbSpecNestedChildInfo.yangType == YANG_LIST && dbSpecNestedChildInfo.dbEntry.Parent.IsList() { //nested list case
								if strings.HasSuffix(uri, nestedChildName) || strings.HasSuffix(uri, nestedChildName+"/") { //whole list case
									resultMap[REPLACE] = make(RedisDbMap)
									resultMap[REPLACE][db.ConfigDB] = result
								} else { // target URI is at nested list-instance or nested-list-instance/leaf
									resultMap[UPDATE] = make(RedisDbMap)
									resultMap[UPDATE][db.ConfigDB] = result
								}
							} else {
								log.Warningf("For URI - %v, only nested list supported, other type of yang node not supported - %v", uri, dbSpecPath)
							}
						} else {
							log.Warningf("For URI - %v, no entry found in xDbSpecMap for table(%v)/field(%v)", uri, tableName, fldNm)
						}
					} else {
						log.Warningf("For URI - %v, no data found in xDbSpecMap for path - %v", uri, dbSpecField)
					}
				} else {
					log.Warningf("For URI - %v, no entry found in xDbSpecMap with tableName - %v", uri, tableName)
				}
			}
		} else {
			resultMap[oper] = make(RedisDbMap)
			resultMap[oper][db.ConfigDB] = result
		}
	} else {
		replaceInfo = replaceProcessingInfo{isDeleteForReplace, replaceSubtreeMap, subOpDataMapForReplace, targetHasNonTerminalNode, nil, isNonTblOwnerDefaultValProcess}
		xlateToData := formXlateToDbParam(d, ygRoot, oper, root, uri, "", "", jsonData, resultMap, result, txCache, tblXpathMap, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "", false, invokeSubtreeOnceMap, nil, nil, &replaceInfo)
		/* Invoke pre-xfmr is present for the YANG module */
		if xYangModSpecMap != nil {
			if modSpecInfo, specOk := xYangModSpecMap[moduleNm]; specOk && (len(modSpecInfo.xfmrPre) > 0) {
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
		}
		if oper == REPLACE {
			skipDelete = new(bool)
			xlateToData.replaceInfo.isDeleteForReplace = true
			allocateSubOpDataMapForOper(xlateToData.replaceInfo.subOpDataMap, UPDATE)
			xlateToData.uri = uri
			err = processTargetUriForReplace(xlateToData, skipDelete)
			xlateToData.uri = root //payload processing begins from module root
			xfmrLogDebug("processTargetUriForReplace returned skipDelete - %v, replace result map - %v, subOpDataMap - %v, xlateToData.replaceInfo.subOpDataMap - %v",
				*skipDelete, xlateToData.result, subOpDataMapType(xlateToData.subOpDataMap), subOpDataMapType(xlateToData.replaceInfo.subOpDataMap))
			if err != nil {
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
			defSubOpDataMap := make(map[Operation]*RedisDbMap)
			if ok {
				xfmrLogInfo("Fill default value for %v, oper(%v)\r\n", uri, oper)
				curXlateToParams := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, xpath, "", jsonData, resultMap, result, txCache, tblXpathMap, defSubOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "", false, invokeSubtreeOnceMap, yangDefValMap, yangAuxValMap, nil)
				if oper != REPLACE {
					err = dbMapDefaultValFill(curXlateToParams)
				} else {
					curXlateToParams.subOpDataMap = subOpDataMap
					curXlateToParams.replaceInfo = &replaceInfo
					err = dbMapDefaultValFillForReplace(curXlateToParams)
				}
				if err != nil {
					return err
				}
			}

			if ok && oper == REPLACE {
				combineGlobalSubOpMapWithReplaceInfoSubOpMap(subOpDataMap, replaceInfo.subOpDataMap)
				if (skipDelete != nil) && !(*skipDelete) {
					xlateToData := formXlateToDbParam(d, ygRoot, oper, uri, requestUri, "", "", jsonData, resultMap, result, txCache, tblXpathMap, subOpDataMap, &cascadeDelTbl, &xfmrErr, "", "", "", false, invokeSubtreeOnceMap, nil, nil, &replaceInfo)
					if err = processDeleteForReplace(xlateToData); err != nil {
						return err
					}
				} else {
					if yangNode.yangType == YANG_LEAF {
						xfmrLogInfo("Change leaf oper to UPDATE for %v, oper(%v)\r\n", uri, oper)
						resultMap[UPDATE] = make(RedisDbMap)
						resultMap[UPDATE][db.ConfigDB] = result
						result = make(map[string]map[string]db.Value)
					}
				}
			}

			/* Invoke post-xfmr is present for the YANG module */
			moduleNm := "/" + strings.Split(uri, "/")[1]
			xfmrLogDebug("Module name for URI %s is %s", uri, moduleNm)
			if _, ok := xYangSpecMap[moduleNm]; ok {
				xfmrPost := ""
				if xYangModSpecMap != nil {
					if _, ok = xYangModSpecMap[moduleNm]; ok {
						xfmrPost = xYangModSpecMap[moduleNm].xfmrPost
					}
				}
				if xYangSpecMap[moduleNm].yangType == YANG_CONTAINER && len(xfmrPost) > 0 {
					var dbDataMap = make(RedisDbMap)
					dbDataMap[db.ConfigDB] = result
					var dbs [db.MaxDB]*db.DB
					inParams := formXfmrInputRequest(d, dbs, db.ConfigDB, ygRoot, uri, requestUri, oper, "", &dbDataMap, subOpDataMap, nil, txCache)
					inParams.yangDefValMap = yangDefValMap
					err = postXfmrHandlerFunc(xfmrPost, inParams)
					if err != nil {
						return err
					}
					if inParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
						for _, tblNm := range *inParams.pCascadeDelTbl {
							if !contains(cascadeDelTbl, tblNm) {
								cascadeDelTbl = append(cascadeDelTbl, tblNm)
							}
						}
					}
				}
			} else {
				log.Warningf("No Entry exists for module %s in xYangSpecMap. Unable to process post xfmr (\"%v\") uri(\"%v\") error (\"%v\").", moduleNm, oper, uri, err)
			}

			if (oper != REPLACE) && (len(result) > 0 || len(subOpDataMap) > 0) {
				resultMap[oper] = make(RedisDbMap)
				resultMap[oper][db.ConfigDB] = result
				for op, redisMapPtr := range subOpDataMap {
					if redisMapPtr != nil {
						if _, ok := resultMap[op]; !ok {
							resultMap[op] = make(RedisDbMap)
						}
						for dbNum, dbMap := range *redisMapPtr {
							if _, ok := resultMap[op][dbNum]; !ok {
								resultMap[op][dbNum] = make(map[string]map[string]db.Value)
							}
							mapCopy(resultMap[op][dbNum], dbMap)
						}
					}
				}
			} else if (len(result) > 0 || len(subOpDataMap) > 0) && (oper == REPLACE && (skipDelete != nil && *skipDelete)) {
				//Merge/Consolidate, only for non processDeleteForReplace cases, across operations giving priority to Replace since its the north bound operation
				xfmrLogDebug("Before merging resultMap %v , result(replace result) %v, subOpDataMap %v", resultMap, result, subOpDataMapType(subOpDataMap))
				mergeSubOpMapWithResultForReplace(resultMap, result, subOpDataMap)
				xfmrLogDebug("After merging resultMap %v", resultMap)
			}
		}

		err = dbDataXfmrHandler(resultMap)
		if err != nil {
			log.Warningf("Failed in dbdata-xfmr for %v", resultMap)
			return err
		}
		yangDefValDbDataXfmrMap := make(map[Operation]RedisDbMap)
		yangDefValDbDataXfmrMap[oper] = RedisDbMap{db.ConfigDB: yangDefValMap}
		err = dbDataXfmrHandler(yangDefValDbDataXfmrMap)
		if err != nil {
			log.Warningf("Failed in dbdata-xfmr for %v", yangDefValDbDataXfmrMap)
			return err
		}
		if yangAuxValOper == REPLACE {
			/* yangAuxValMap is used only for terminal-container REPLACE case
			   and has fields to be deleted, so send DELETE to valueXfmr
			*/
			yangAuxValOper = DELETE
		}
		yangAuxValDbDataXfmrMap := make(map[Operation]RedisDbMap)
		yangAuxValDbDataXfmrMap[yangAuxValOper] = RedisDbMap{db.ConfigDB: yangAuxValMap}
		err = dbDataXfmrHandler(yangAuxValDbDataXfmrMap)
		if err != nil {
			log.Warningf("Failed in dbdata-xfmr for %v", yangAuxValDbDataXfmrMap)
			return err
		}

		if len(cascadeDelTbl) > 0 {
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
		log.Warningf("For URI %v - StringToPath failure", uri)
		errStr := fmt.Sprintf("Ygot stringTopath failed for uri(%v)", uri)
		return nil, tlerr.InternalError{Format: errStr}
	}

	for _, p := range path.Elem {
		pathSlice := strings.Split(p.Name, ":")
		p.Name = pathSlice[len(pathSlice)-1]
		if len(p.Key) > 0 {
			for ekey, ent := range p.Key {
				// SNC-2126: check the occurrence of ":"
				if (strings.Contains(ent, ":")) && (strings.HasPrefix(ent, OC_MDL_PFX) || strings.HasPrefix(ent, IETF_MDL_PFX) || strings.HasPrefix(ent, IANA_MDL_PFX)) {
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
		xfmrLogDebug("For URI %v - Unable to GetNode - %v", uri, nErr)
		errStr := fmt.Sprintf("%v", nErr)
		return nil, tlerr.InternalError{Format: errStr}
	}
	if (node == nil) || (len(node) == 0) || (node[0].Data == nil) {
		log.Warningf("GetNode returned nil for URI %v", uri)
		errStr := "GetNode returned nil for the given uri."
		return nil, tlerr.InternalError{Format: errStr}
	}
	xfmrLogDebug("GetNode data: %v", node[0].Data)
	return node[0].Data, nil
}

func yangReqToDbMapCreate(xlateParams xlateToParams) error {
	xfmrLogDebug("key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
	var dbs [db.MaxDB]*db.DB
	var retErr error

	if reflect.ValueOf(xlateParams.jsonData).Kind() == reflect.Slice {
		xfmrLogDebug("slice data: key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
		jData := reflect.ValueOf(xlateParams.jsonData)
		dataMap := make([]interface{}, jData.Len())
		for idx := 0; idx < jData.Len(); idx++ {
			dataMap[idx] = jData.Index(idx).Interface()
		}
		for _, data := range dataMap {
			curKey := ""
			curUri, _ := uriWithKeyCreate(xlateParams.uri, xlateParams.xpath, data)
			_, ok := xYangSpecMap[xlateParams.xpath]
			if ok && len(xYangSpecMap[xlateParams.xpath].validateFunc) > 0 {
				inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, nil, xlateParams.txCache)
				res := validateHandlerFunc(inParams, xYangSpecMap[xlateParams.xpath].validateFunc)
				if !res {
					if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
						*xlateParams.xfmrErr = tlerr.InternalError{Format: "Invalid request data", Path: curUri}
					}
					xfmrLogDebug("Validate handler returned false so don't traverse further for URI - %v", curUri)
					return nil
				}
			}
			if ok && len(xYangSpecMap[xlateParams.xpath].xfmrKey) > 0 {
				// key transformer present
				curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
				if nodeErr != nil {
					curYgotNode = nil
				}
				inParams := formXfmrInputRequest(xlateParams.d, dbs, db.ConfigDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)

				ktRetData, err := keyXfmrHandler(inParams, xYangSpecMap[xlateParams.xpath].xfmrKey)
				// if key transformer is called without key values in curUri ignore the error
				if err != nil && strings.HasSuffix(curUri, "]") {
					if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
						*xlateParams.xfmrErr = err
					}
					return nil
				}
				curKey = ktRetData
			} else if ok && xYangSpecMap[xlateParams.xpath].keyName != nil {
				curKey = *xYangSpecMap[xlateParams.xpath].keyName
			} else {
				curKey = keyCreate(xlateParams, curUri, data)
			}

			curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, xlateParams.xpath, curKey, data, xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, "", "", "", false, xlateParams.invokeCRUSubtreeOnceMap, nil, nil, xlateParams.replaceInfo)

			if xlateParams.oper == REPLACE { //propagate table-name to children
				curTbl := xlateParams.tableName
				var tblErr error
				curXpath := xlateParams.xpath
				reqXpath, _, _ := XfmrRemoveXPATHPredicates(xlateParams.requestUri)
				if strings.HasPrefix(curXpath, reqXpath) {
					curTbl, tblErr = dbTableFromUriGet(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curXpath, curUri, xlateParams.requestUri, xlateParams.subOpDataMap, xlateParams.txCache, nil)
					if tblErr != nil {
						if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
							*xlateParams.xfmrErr = tblErr
						}
						return nil
					}
					curXlateParams.tableName = curTbl
				}
			}
			xfmrLogDebug("Before yangReqToDbMapCreate(List instance case) uri - %v, result map - %v, subOpDataMap - %v", curUri, xlateParams.result, xlateParams.subOpDataMap)
			retErr = yangReqToDbMapCreate(curXlateParams)
			xfmrLogDebug("After yangReqToDbMapCreate(List instance case) uri - %v, result map - %v, subOpDataMap - %v", xlateParams.uri, xlateParams.result, xlateParams.subOpDataMap)
		}
	} else {
		if reflect.ValueOf(xlateParams.jsonData).Kind() == reflect.Map {
			jData := reflect.ValueOf(xlateParams.jsonData)
			for _, key := range jData.MapKeys() {
				typeOfValue := reflect.TypeOf(jData.MapIndex(key).Interface()).Kind()

				xfmrLogDebug("slice/map data: key(\"%v\"), xpathPrefix(\"%v\").", xlateParams.keyName, xlateParams.xpath)
				xpath := xlateParams.uri
				curUri := xlateParams.uri
				curKey := xlateParams.keyName
				pathAttr := key.String()
				if len(xlateParams.xpath) > 0 {
					curUri = xlateParams.uri + "/" + pathAttr
					if strings.Contains(pathAttr, ":") {
						pathAttr = strings.Split(pathAttr, ":")[1]
					}
					xpath = xlateParams.xpath + "/" + pathAttr
				}
				curXpath, _, _ := XfmrRemoveXPATHPredicates(curUri)
				reqXpath, _, _ := XfmrRemoveXPATHPredicates(xlateParams.requestUri)
				_, ok := xYangSpecMap[xpath]
				xfmrLogDebug("slice/map data: curKey(\"%v\"), xpath(\"%v\"), curUri(\"%v\").",
					curKey, xpath, curUri)
				/*for list case validate handler will be called per instance so don't call at whole list level*/
				if ok && (xYangSpecMap[xpath] != nil) && (len(xYangSpecMap[xpath].validateFunc) > 0) && (xYangSpecMap[xpath].validateFunc != xYangSpecMap[xlateParams.xpath].validateFunc) && (xYangSpecMap[xpath].yangType != YANG_LIST) {
					inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, nil, xlateParams.txCache)
					res := validateHandlerFunc(inParams, xYangSpecMap[xpath].validateFunc)
					if !res {
						if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
							*xlateParams.xfmrErr = tlerr.InternalError{Format: "Invalid request data", Path: curUri}
						}
						xfmrLogDebug("Validate handler returned false so don't traverse further for URI - %v", curUri)
						return nil
					}

				}
				if ok && xYangSpecMap[xpath] != nil && len(xYangSpecMap[xpath].xfmrKey) > 0 {
					specYangType := xYangSpecMap[xpath].yangType
					curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
					if nodeErr != nil {
						curYgotNode = nil
					}
					inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
					ktRetData, err := keyXfmrHandler(inParams, xYangSpecMap[xpath].xfmrKey)
					if (err != nil) && (specYangType != YANG_LIST || strings.HasSuffix(curUri, "]")) {
						if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
							*xlateParams.xfmrErr = err
						}
						return nil
					}
					curKey = ktRetData
				} else if ok && xYangSpecMap[xpath].keyName != nil {
					curKey = *xYangSpecMap[xpath].keyName
				}
				if ok && (typeOfValue == reflect.Map || typeOfValue == reflect.Slice) && xYangSpecMap[xpath].yangType != YANG_LEAF_LIST {
					// Call subtree only if start processing for the requestUri. Skip for parent URI traversal
					xfmrLogDebug("CurUri: %v, requestUri: %v\r\n", curUri, xlateParams.requestUri)
					xfmrLogDebug("curxpath: %v, requestxpath: %v\r\n", curXpath, reqXpath)
					if strings.HasPrefix(curXpath, reqXpath) {
						if xYangSpecMap[xpath] != nil && len(xYangSpecMap[xpath].xfmrFunc) > 0 {
							xfmrFunc := xYangSpecMap[xpath].xfmrFunc
							callSubtree := true
							uriMap, subEntryOk := xlateParams.invokeCRUSubtreeOnceMap[xfmrFunc]
							if subEntryOk {
								for subtreeUri, stOnceFlag := range uriMap {
									if strings.HasPrefix(curUri, subtreeUri) {
										if stOnceFlag {
											callSubtree = false
										}
										break
									}
								}
							}
							/* Mark subtree visited while REPLACE/PUT payload processing, to be used in delete flow */
							if xlateParams.oper == REPLACE && xlateParams.replaceInfo != nil {
								if xlateParams.replaceInfo.subtreeVisitedCache == nil {
									xlateParams.replaceInfo.subtreeVisitedCache = make(map[string]bool)
								}
								subtreeUri := curUri
								if (curXpath == reqXpath) && xYangSpecMap[xpath].hasChildSubTree {
									if xYangSpecMap[xpath].yangType == YANG_LIST && (strings.HasSuffix(xlateParams.requestUri, "]") || strings.HasSuffix(xlateParams.requestUri, "]/")) {
										/* payload processing begins at whole list level, so curUri will be pointing to whole list
										   and not actual list instance in requestUri which will be encountered when reaching child subtree
										   in Delete flow.So mark parent subtree as visited
										*/
										subtreeUri = xlateParams.requestUri
									}
									xlateParams.replaceInfo.subtreeVisitedCache[subtreeUri] = true
								} else if len(curXpath) > len(reqXpath) {
									parentSubtreeFunc := xYangSpecMap[xlateParams.xpath].xfmrFunc
									if (len(parentSubtreeFunc) == 0) || ((len(xfmrFunc) > 0) && (parentSubtreeFunc != xfmrFunc)) {
										xlateParams.replaceInfo.subtreeVisitedCache[subtreeUri] = true
									}
								}
							}
							if callSubtree {
								// TODO: Call subtree for child node if same subtree invoked for parent
								/* subtree transformer present */
								curYgotNode, nodeErr := yangNodeForUriGet(curUri, xlateParams.ygRoot)
								if nodeErr != nil {
									curYgotNode = nil
								}
								inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, "", nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
								stRetData, err := xfmrHandler(inParams, xfmrFunc)
								if err != nil {
									if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
										*xlateParams.xfmrErr = err
									}
									return nil
								}
								if stRetData != nil {
									mapCopy(xlateParams.result, stRetData)
								}
								if *inParams.invokeCRUSubtreeOnce {
									xfmrLogDebug("Invoke subtree %v only once at  %v. xfmr will not invoke subtree for children", xfmrFunc, curUri)
									_, subEntryOk := xlateParams.invokeCRUSubtreeOnceMap[xfmrFunc]
									if !subEntryOk {
										xlateParams.invokeCRUSubtreeOnceMap[xfmrFunc] = make(map[string]bool)
										xlateParams.invokeCRUSubtreeOnceMap[xfmrFunc][curUri] = *inParams.invokeCRUSubtreeOnce

									} else {
										xlateParams.invokeCRUSubtreeOnceMap[xfmrFunc][curUri] = *inParams.invokeCRUSubtreeOnce
									}
								} else {
									xfmrLogDebug("Tranformer will invoke subtree %v at child nodes(container/list) for URI %v. Set inParams.invokeCRUSubtreeOnce to true for transformer to invoke subtree only once", xfmrFunc, curUri)
								}
								if xlateParams.pCascadeDelTbl != nil && len(*inParams.pCascadeDelTbl) > 0 {
									for _, tblNm := range *inParams.pCascadeDelTbl {
										if !contains(*xlateParams.pCascadeDelTbl, tblNm) {
											*xlateParams.pCascadeDelTbl = append(*xlateParams.pCascadeDelTbl, tblNm)
										}
									}
								}
							}
						}
					}
					xfmrLogDebug("Before yangReqToDbMapCreate() uri - %v, result map - %v, subOpDataMap - %v", curUri, xlateParams.result, xlateParams.subOpDataMap)
					curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, curUri, xlateParams.requestUri, xpath, curKey, jData.MapIndex(key).Interface(), xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, "", "", "", false, xlateParams.invokeCRUSubtreeOnceMap, nil, nil, xlateParams.replaceInfo)
					if ok && (xYangSpecMap[xpath] != nil) && (len(xYangSpecMap[xpath].xfmrFunc) == 0) && (xlateParams.oper == REPLACE) && (len(curXpath) > len(reqXpath)) && (xYangSpecMap[xpath].yangType == YANG_CONTAINER) {
						/*propagate table-name to children.Also add table-instance, corresponding to the container,
						  if different from parent, to translated result */
						curTbl := xlateParams.tableName
						var tblErr error
						curTbl, tblErr = dbTableFromUriGet(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xpath, curUri, xlateParams.requestUri, xlateParams.subOpDataMap, xlateParams.txCache, nil)
						if tblErr != nil {
							if xlateParams.xfmrErr != nil && *xlateParams.xfmrErr == nil {
								*xlateParams.xfmrErr = tblErr
							}
							return nil //stop taversal, error already propagated through xlateParams.xfmrErr
						}
						curXlateParams.tableName = curTbl
						if (curTbl != xlateParams.tableName) && (curKey != "") {
							replcePayloadContainerProcessing(curXlateParams)
						}
					}
					retErr = yangReqToDbMapCreate(curXlateParams)
					xfmrLogDebug("After yangReqToDbMapCreate() uri - %v, result map - %v, subOpDataMap - %v", xlateParams.uri, xlateParams.result, xlateParams.subOpDataMap)
				} else {
					pathAttr := key.String()
					if strings.Contains(pathAttr, ":") {
						pathAttr = strings.Split(pathAttr, ":")[1]
					}
					xpath := xlateParams.xpath + "/" + pathAttr
					xfmrLogDebug("TerminalNode(LEAF/LEAF-LIST) Case: xpath: %v, xpathPrefix: %v, pathAttr: %v", xpath, xlateParams.xpath, pathAttr)
					/* skip processing for list key-leaf outside of config container(OC yang) directly under the list.
					   Inside full-spec isKey is set to true for list key-leaf dierctly under the list(outside of config container)
					   For ietf yang(eg.ietf-ptp) list key-leaf might have a field transformer.
					*/
					_, ok := xYangSpecMap[xpath]
					// Process the terminal node only if the targetUri is at terminal Node or if the leaf is not at parent level
					if ok && strings.HasPrefix(curXpath, reqXpath) {
						if (!xYangSpecMap[xpath].isKey) || (len(xYangSpecMap[xpath].xfmrField) > 0) || (xYangSpecMap[xpath].isKey && (strings.HasPrefix(xlateParams.uri, "/"+IETF_MDL_PFX) || xlateParams.oper == REPLACE)) {
							if len(xYangSpecMap[xpath].xfmrFunc) == 0 {
								value := jData.MapIndex(key).Interface()
								xfmrLogDebug("Processing data field: key(\"%v\").", key)
								xfmrLogDebug("Before mapFillData uri - %v, node - %v, result map - %v, subOpDataMap - %v", xlateParams.uri, pathAttr, xlateParams.result, xlateParams.subOpDataMap)
								curXlateParams := formXlateToDbParam(xlateParams.d, xlateParams.ygRoot, xlateParams.oper, xlateParams.uri, xlateParams.requestUri, xlateParams.xpath, curKey, xlateParams.jsonData, xlateParams.resultMap, xlateParams.result, xlateParams.txCache, xlateParams.tblXpathMap, xlateParams.subOpDataMap, xlateParams.pCascadeDelTbl, xlateParams.xfmrErr, pathAttr, value, "", false, xlateParams.invokeCRUSubtreeOnceMap, nil, nil, xlateParams.replaceInfo)
								retErr = mapFillData(curXlateParams)
								xfmrLogDebug("After mapFillData uri - %v, node - %v, result map - %v, subOpDataMap - %v", xlateParams.uri, pathAttr, xlateParams.result, xlateParams.subOpDataMap)
								if retErr != nil {
									log.Warningf("Failed constructing data for db write: key(\"%v\"), path(\"%v\").",
										pathAttr, xlateParams.xpath)
									return retErr
								}
							} else if (curXpath == reqXpath) || (xYangSpecMap[xlateParams.xpath].xfmrFunc != xYangSpecMap[xpath].xfmrFunc) {
								xfmrLogDebug("write: key(\"%v\"), xpath(\"%v\"), uri(%v).", key, xpath, xlateParams.uri)
								curYgotNode, nodeErr := yangNodeForUriGet(xlateParams.uri, xlateParams.ygRoot)
								if nodeErr != nil {
									curYgotNode = nil
								}
								inParams := formXfmrInputRequest(xlateParams.d, dbs, db.MaxDB, xlateParams.ygRoot, curUri, xlateParams.requestUri, xlateParams.oper, curKey, nil, xlateParams.subOpDataMap, curYgotNode, xlateParams.txCache)
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
									for _, tblNm := range *inParams.pCascadeDelTbl {
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
	}

	return retErr
}

func verifyParentTableSonic(d *db.DB, dbs [db.MaxDB]*db.DB, oper Operation, uri string, dbData RedisDbMap) (bool, error) {
	var err error

	xpath, dbKey, table := sonicXpathKeyExtract(uri)
	xfmrLogDebug("uri: %v xpath: %v table: %v, key: %v", uri, xpath, table, dbKey)

	if (len(table) > 0) && (len(dbKey) > 0) {
		tableExists := false
		var derr error
		pathList := strings.Split(xpath, "/")[1:]
		hasSingletonContainer := SonicUriHasSingletonContainer(uri)
		if hasSingletonContainer && oper != DELETE {
			// No resource check required for singleton container for CRU cases
			return true, err
		}
		if oper == GET {
			var cdb db.DBNum = db.ConfigDB
			dbInfo, ok := xDbSpecMap[table]
			if !ok {
				log.Warningf("No entry in xDbSpecMap for xpath %v, for URI - %v", table, uri)
			} else {
				cdb = dbInfo.dbIndex
			}
			tableExists = dbTableExistsInDbData(cdb, table, dbKey, dbData)

		} else {
			// Valid table mapping exists. Read the table entry from DB
			tableExists, derr = dbTableExists(d, table, dbKey, oper)
			if hasSingletonContainer && oper == DELETE {
				// Special case when we delete at container that does'nt exist. Return true to skip translation.
				if !tableExists {
					return true, derr
				} else {
					return true, nil
				}
			}
		}
		if len(pathList) == SONIC_TBL_CHILD_INDEX && (oper == UPDATE || oper == CREATE || oper == DELETE || oper == GET) && !tableExists {
			// Uri is at /sonic-module:sonic-module/container-table/list
			// PATCH opertion permitted only if table exists in DB.
			// POST case since the URI is the parent, the parent needs to exist
			// PUT case allow operation(Irrespective of table existence update the DB either through CREATE or REPLACE)
			// DELETE case Table instance should be available to perform delete else, CVL may throw error
			log.Warningf("Parent table %v with key %v does not exist for oper %v in DB", table, dbKey, oper)
			err = tlerr.NotFound("Resource not found")
			return false, err
		} else if len(pathList) > SONIC_TBL_CHILD_INDEX && !tableExists {
			// Uri is at /sonic-module/container-table/list/leaf
			// Parent table should exist for all CRUD cases
			log.Warningf("Parent table %v with key %v does not exist in DB", table, dbKey)
			err = tlerr.NotFound("Resource not found")
			return false, err
		} else {
			/*For CRUD operations on a nested list-intance or nested-list[intance]/leaf query
			  check if nested-list instance exists in DB. For GET operation the resource check for
			  nested list instance will be done before populating data into yang from DBdata to
			  optimize processing, since DBdata will be referenced there anyways using nested list
			  instance key-value */
			if len(pathList) > SONIC_TBL_CHILD_INDEX && oper != GET {
				//extract nested-list name from pathList
				pathElement := pathList[SONIC_FIELD_INDEX-1] //Empty element at index 0 is removed when xpath split on slash
				dbSpecField := table + "/" + pathElement
				_, ok := xDbSpecMap[dbSpecField]
				if !ok && pathElement != "" {
					if nestedListErr := sonicNestedListRequestResourceCheck(uri, table, dbKey, pathList[SONIC_TBL_CHILD_INDEX-1], pathElement, d, oper); nestedListErr != nil {
						return false, nestedListErr
					}
				}
			}
			// Allow all other operations
			return true, err
		}
	} else {
		// Request is at module level. No need to check for parent table. Hence return true always or
		// Request at /sonic-module:sonic-module/container-table or container-table/whole list level
		return true, err
	}
}

/*
This function checks the existence of Parent tables in DB for the given URI request

	and returns a boolean indicating if the operation is permitted based on the operation type
*/
func verifyParentTable(d *db.DB, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, dbData RedisDbMap, txCache interface{}, subOpDataMap map[Operation]*RedisDbMap, dbTblKeyCache map[string]tblKeyCache) (bool, error) {
	xfmrLogDebug("Checking for Parent table existence for uri: %v", uri)
	if d != nil && dbs[d.Opts.DBNo] == nil {
		dbs[d.Opts.DBNo] = d
	}
	if isSonicYang(uri) {
		return verifyParentTableSonic(d, dbs, oper, uri, dbData)
	} else {
		return verifyParentTableOc(d, dbs, ygRoot, oper, uri, dbData, txCache, subOpDataMap, dbTblKeyCache)
	}
}

func verifyParentTblSubtree(dbs [db.MaxDB]*db.DB, uri string, xfmrFuncNm string, oper Operation, dbData RedisDbMap) (bool, error) {
	var inParams XfmrSubscInParams
	inParams.uri = uri
	inParams.dbDataMap = make(RedisDbMap)
	inParams.dbs = dbs
	inParams.subscProc = TRANSLATE_SUBSCRIBE
	parentTblExists := true
	var err error

	st_result, st_err := xfmrSubscSubtreeHandler(inParams, xfmrFuncNm)
	if st_result.isVirtualTbl {
		xfmrLogDebug("Subtree returned Virtual table true.")
		goto Exit
	}
	if st_err != nil {
		err = st_err
		parentTblExists = false
		goto Exit
	} else if st_result.dbDataMap != nil && len(st_result.dbDataMap) > 0 {
		xfmrLogDebug("Subtree subcribe dbData %v", st_result.dbDataMap)
		for dbNo, dbMap := range st_result.dbDataMap {
			xfmrLogDebug("processing DB no - %v", dbNo)
			for table, keyInstance := range dbMap {
				xfmrLogDebug("processing DB table - %v", table)
				for dbKey := range keyInstance {
					xfmrLogDebug("processing DB key - %v", dbKey)
					exists := false
					if oper != GET {
						var dptr *db.DB
						if d := dbs[dbNo]; d != nil {
							dptr = d
						} else if dbNo == db.ConfigDB {
							// Infra MUST always pass ConfigDB handle (for bulk & config session usecase)
							err = tlerr.New("DB access failure")
						} else {
							dptr, err = db.NewDB(getDBOptions(dbNo))
							defer dptr.DeleteDB()
						}
						if err != nil {
							log.Warningf("Couldn't allocate NewDb/DbOptions for db - %v, while processing URI - %v", dbNo, uri)
							parentTblExists = false
							goto Exit
						}
						exists, err = dbTableExists(dptr, table, dbKey, oper)
					} else {
						d := dbs[dbNo]
						if dbKey == "*" { //dbKey is "*" for GET on entire list
							xfmrLogDebug("Found table instance in dbData")
							goto Exit
						}
						// GET case - attempt to find in dbData before doing a dbGet in dbTableExists()
						exists = dbTableExistsInDbData(dbNo, table, dbKey, dbData)
						if exists {
							xfmrLogDebug("Found table instance in dbData")
							goto Exit
						}
						exists, err = dbTableExists(d, table, dbKey, oper)
					}
					if !exists || err != nil {
						log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for URI %v", table, dbKey, uri)
						err = tlerr.NotFound("Resource not found")
						parentTblExists = false
						goto Exit
					}
				}
			}

		}
	} else {
		log.Warningf("No Table information retrieved from subtree for URI %v", uri)
		err = tlerr.NotFound("Resource not found")
		parentTblExists = false
		goto Exit
	}
Exit:
	xfmrLogDebug("For subtree at URI - %v, returning ,parentTblExists - %v, err - %v", parentTblExists, err)
	return parentTblExists, err
}

func verifyParentTableOc(d *db.DB, dbs [db.MaxDB]*db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, dbData RedisDbMap, txCache interface{}, subOpDataMap map[Operation]*RedisDbMap, dbTblKeyCache map[string]tblKeyCache) (bool, error) {
	var err error
	var cdb db.DBNum
	var parentTable string
	uriList := splitUri(uri)
	parentTblExists := true
	curUri := "/"
	var yangType yangElementType
	xpath, _, _ := XfmrRemoveXPATHPredicates(uri)
	xpathInfo, ok := xYangSpecMap[xpath]
	if !ok {
		errStr := fmt.Sprintf("No entry found in xYangSpecMap for URI - %v", uri)
		err = tlerr.InternalError{Format: errStr}
		return false, err
	}
	yangType = xpathInfo.yangType
	if yangType == YANG_LEAF_LIST {
		/*query is for leaf-list instance, hence remove that from uriList to avoid list-key like processing*/
		if (strings.HasSuffix(uriList[len(uriList)-1], "]")) || (strings.HasSuffix(uriList[len(uriList)-1], "]/")) { //splitUri chops off the leaf-list value having square brackets
			uriList[len(uriList)-1] = strings.SplitN(uriList[len(uriList)-1], "[", 2)[0]
			xfmrLogDebug("Uri list after removing leaf-list instance portion - %v", uriList)
		}
	}

	parentUriList := uriList[:len(uriList)-1]
	xfmrLogDebug("Parent URI list - %v", parentUriList)

	// Loop for the parent URI to check parent table existence
	for idx, path := range parentUriList {
		curUri += uriList[idx]

		/* Check for parent table for oc- YANG lists*/
		pathInfo := NewPathInfo("/" + path)
		if (pathInfo != nil) && (len(pathInfo.Vars) > 0) {

			//Check for subtree existence
			curXpath, _, _ := XfmrRemoveXPATHPredicates(curUri)
			curXpathInfo, ok := xYangSpecMap[curXpath]
			if !ok {
				errStr := fmt.Sprintf("No entry found in xYangSpecMap for URI - %v", curUri)
				err = tlerr.InternalError{Format: errStr}
				parentTblExists = false
				break
			}
			if oper == GET {
				cdb = curXpathInfo.dbIndex
				xfmrLogDebug("db index for curXpath - %v is %v", curXpath, cdb)
				d = dbs[cdb]
			}
			if curXpathInfo.virtualTbl != nil && *curXpathInfo.virtualTbl {
				curUri += "/"
				continue
			}
			// Check for subtree case and invoke subscribe xfmr
			if len(curXpathInfo.xfmrFunc) > 0 {
				xfmrLogDebug("Found subtree for URI - %v", curUri)
				stParentTblExists := false
				stParentTblExists, err = verifyParentTblSubtree(dbs, curUri, curXpathInfo.xfmrFunc, oper, dbData)
				if err != nil {
					parentTblExists = false
					break
				}
				if !stParentTblExists {
					log.Warningf("Parent Table does not exist for URI %v", uri)
					err = tlerr.NotFound("Resource not found")
					parentTblExists = false
					break
				}
			} else {

				xfmrLogDebug("Check parent table for uri: %v", curUri)
				// Get Table and Key only for YANG list instances
				var xpathKeyExtRet xpathTblKeyExtractRet
				var xerr error
				if oper == GET {
					xpathKeyExtRet, xerr = xpathKeyExtractForGet(d, ygRoot, oper, curUri, uri, &dbData, subOpDataMap, txCache, dbTblKeyCache, dbs)
				} else {
					xpathKeyExtRet, xerr = xpathKeyExtract(d, ygRoot, oper, curUri, uri, nil, subOpDataMap, txCache, nil, dbs)
				}
				if xerr != nil {
					log.Warningf("Failed to get table and key for uri: %v err: %v", curUri, xerr)
					err = xerr
					parentTblExists = false
					break
				}
				if xpathKeyExtRet.isVirtualTbl {
					curUri += "/"
					continue
				}

				if len(xpathKeyExtRet.tableName) > 0 && len(xpathKeyExtRet.dbKey) > 0 {
					// Check for Table existence
					xfmrLogDebug("DB Entry Check for uri: %v table: %v, key: %v", uri, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
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
							log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for URI %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, uri)
							err = tlerr.NotFound("Resource not found")
							break
						}
					}
					parentTable = xpathKeyExtRet.tableName
				} else {
					// We always expect a valid table and key to be returned. Else we cannot validate parent check
					parentTblExists = false
					log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for URI %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, curUri)
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
		xfmrLogDebug("Check last parent table for uri: %v", uri)
		virtualTbl := false
		if xpathInfo.virtualTbl != nil {
			virtualTbl = *xpathInfo.virtualTbl
		}
		if virtualTbl {
			xfmrLogDebug("Virtual table at URI - %v", uri)
			return true, nil
		}
		// Check for subtree case and invoke subscribe xfmr
		if len(xpathInfo.xfmrFunc) > 0 {
			if !((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) { //uri points to entire list
				xfmrLogDebug("subtree , whole list case for URI - %v", uri)
				return true, nil
			}
			xfmrLogDebug("Found subtree for URI - %v", uri)
			parentTblExists, err = verifyParentTblSubtree(dbs, uri, xpathInfo.xfmrFunc, oper, dbData)
			if err != nil {
				return false, err
			}
			if !parentTblExists {
				log.Warningf("Parent Table does not exist for URI %v", uri)
				err = tlerr.NotFound("Resource not found")
				return false, err
			}
			return true, nil
		}
		if !((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) { //uri points to entire list
			return true, nil
		}
		d = dbs[xpathInfo.dbIndex]
		var xpathKeyExtRet xpathTblKeyExtractRet
		var xerr error
		if oper == GET {
			xpathKeyExtRet, xerr = xpathKeyExtractForGet(d, ygRoot, oper, uri, uri, nil, subOpDataMap, txCache, dbTblKeyCache, dbs)
		} else {
			xpathKeyExtRet, xerr = xpathKeyExtract(d, ygRoot, oper, uri, uri, nil, subOpDataMap, txCache, nil, dbs)
		}
		if xerr != nil {
			log.Warningf("key extract failed err: %v, table %v, key %v, operation: %v", xerr, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
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
				xfmrLogDebug("db index for xpath - %v is %v", xpath, cdb)
				exists = dbTableExistsInDbData(cdb, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, dbData)
				if !exists {
					exists, derr = dbTableExists(dbs[cdb], xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
					if derr != nil {
						return false, derr
					}
					if !exists {
						parentTblExists = false
						log.Warningf("Parent Tbl :%v, dbKey: %v does not exist for URI %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, uri)
						err = tlerr.NotFound("Resource not found")
					}
				}
			} else {
				exists, derr = dbTableExists(d, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, oper)
			}
			if derr != nil {
				log.Warningf("ParentTable GetEntry failed for table: %v, key: %v err: %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, derr)
				return false, derr
			}
			if !exists {
				log.Warningf("ParentTable GetEntry failed for table: %v, key: %v err: %v", xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey, derr)
				err = tlerr.NotFound("Resource not found")
				return false, err
			} else {
				return true, nil
			}
		} else {
			log.Warningf("Parent table check: Unable to get valid table and key err: %v, table %v, key %v. Please verify table and key mapping", xerr, xpathKeyExtRet.tableName, xpathKeyExtRet.dbKey)
			return false, xerr
		}
	} else if oper == DELETE && (len(xpathInfo.xfmrFunc) == 0) && ((yangType == YANG_LEAF && len(xpathInfo.defVal) > 0) || (yangType == YANG_CONTAINER && ((xpathInfo.keyName != nil && len(*xpathInfo.keyName) > 0) || len(xpathInfo.xfmrKey) > 0))) {
		// If the delete is at container/leaf(having default value) that is mapped to a unique table, then check for table existence to avoid CVL throwing error or reset default value at the leaf when table does not exist.

		// Check for virtual table case at curUri
		if xpathInfo.virtualTbl != nil && *xpathInfo.virtualTbl {
			xfmrLogDebug("virtual table case for uri - %v", uri)
			return true, nil
		}
		var perr error
		if yangType == YANG_CONTAINER {
			parentUri := ""
			if len(parentUriList) > 0 {
				parentUri = strings.Join(parentUriList, "/")
				parentUri = "/" + parentUri
			}
			// Get table for parent xpath
			parentTable, perr = dbTableFromUriGet(d, ygRoot, oper, "", parentUri, uri, nil, txCache, nil)
		}
		// Get table for current xpath
		xpathKeyExtRet, cerr := xpathKeyExtract(d, ygRoot, oper, uri, uri, nil, subOpDataMap, txCache, nil, dbs)
		if xpathKeyExtRet.isVirtualTbl {
			return true, nil
		}
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
func printDbData(resMap map[Operation]map[db.DBNum]map[string]map[string]db.Value, yangDefValMap map[string]map[string]db.Value, fileName string) {
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
