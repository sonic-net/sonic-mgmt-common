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
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func initRegex() {
	rgpIpv6 = regexp.MustCompile(`(([^:]+:){6}(([^:]+:[^:]+)|(.*\..*)))|((([^:]+:)*[^:]+)?::(([^:]+:)*[^:]+)?)(%.+)?`)
	rgpMac = regexp.MustCompile(`([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	rgpIsMac = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)

}

/* Create db key from data xpath(request) */
func keyCreate(keyPrefix string, xpath string, data interface{}, dbKeySep string) string {
	_, ok := xYangSpecMap[xpath]
	if ok {
		if xYangSpecMap[xpath].yangEntry != nil {
			yangEntry := xYangSpecMap[xpath].yangEntry
			delim := dbKeySep
			if len(xYangSpecMap[xpath].delim) > 0 {
				delim = xYangSpecMap[xpath].delim
				xfmrLogDebug("key concatenater(\"%v\") found for xpath %v ", delim, xpath)
			}

			if len(keyPrefix) > 0 {
				keyPrefix += delim
			}
			keyVal := ""
			for i, k := range strings.Split(yangEntry.Key, " ") {
				if i > 0 {
					keyVal = keyVal + delim
				}
				fieldXpath := xpath + "/" + k
				fVal, err := unmarshalJsonToDbData(yangEntry.Dir[k], fieldXpath, k, data.(map[string]interface{})[k])
				if err != nil {
					log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
				}

				if (strings.Contains(fVal, ":")) &&
					(strings.HasPrefix(fVal, OC_MDL_PFX) || strings.HasPrefix(fVal, IETF_MDL_PFX) || strings.HasPrefix(fVal, IANA_MDL_PFX)) {
					// identity-ref/enum has module prefix
					fVal = strings.SplitN(fVal, ":", 2)[1]
				}
				keyVal += fVal
			}
			keyPrefix += string(keyVal)
		}
	}
	return keyPrefix
}

/* Copy redis-db source to destn map */
func mapCopy(destnMap map[string]map[string]db.Value, srcMap map[string]map[string]db.Value) {
	mapCopyMutex.Lock()
	for table, tableData := range srcMap {
		_, ok := destnMap[table]
		if !ok {
			destnMap[table] = make(map[string]db.Value)
		}
		for rule, ruleData := range tableData {
			_, ok = destnMap[table][rule]
			if !ok {
				destnMap[table][rule] = db.Value{Field: make(map[string]string)}
			}
			for field, value := range ruleData.Field {
				destnMap[table][rule].Field[field] = value
			}
		}
	}
	mapCopyMutex.Unlock()
}

var mapMergeMutex = &sync.Mutex{}

/* Merge redis-db source to destn map */
func mapMerge(destnMap map[string]map[string]db.Value, srcMap map[string]map[string]db.Value, oper Operation) {
	mapMergeMutex.Lock()
	for table, tableData := range srcMap {
		_, ok := destnMap[table]
		if !ok {
			destnMap[table] = make(map[string]db.Value)
		}
		for rule, ruleData := range tableData {
			_, ok = destnMap[table][rule]
			if !ok {
				destnMap[table][rule] = db.Value{Field: make(map[string]string)}
			} else {
				if oper == DELETE {
					if (len(destnMap[table][rule].Field) == 0) && (len(ruleData.Field) > 0) {
						continue
					}
					if (len(destnMap[table][rule].Field) > 0) && (len(ruleData.Field) == 0) {
						destnMap[table][rule] = db.Value{Field: make(map[string]string)}
					}
				}
			}
			for field, value := range ruleData.Field {
				dval := destnMap[table][rule]
				if dval.IsPopulated() && dval.Has(field) && strings.HasSuffix(field, "@") {
					attrList := dval.GetList(field)
					attrList = append(attrList, value)
					dval.SetList(field, attrList)
				} else {
					destnMap[table][rule].Field[field] = value
				}
			}
		}
	}
	mapMergeMutex.Unlock()
}

func parentXpathGet(xpath string) string {
	path := ""
	if len(xpath) > 0 {
		p := strings.Split(xpath, "/")
		path = strings.Join(p[:len(p)-1], "/")
	}
	return path
}

func parentUriGet(uri string) string {
	parentUri := ""
	if len(uri) > 0 {
		uriList := splitUri(uri)
		if len(uriList) > 2 {
			parentUriList := uriList[:len(uriList)-1]
			parentUri = strings.Join(parentUriList, "/")
			parentUri = "/" + parentUri
		}
	}
	return parentUri
}

func dbKeyToYangDataConvert(uri string, requestUri string, xpath string, tableName string, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, dbKey string, dbKeySep string, txCache interface{}) (map[string]interface{}, string, error) {
	var err error
	if len(uri) == 0 && len(xpath) == 0 && len(dbKey) == 0 {
		err = fmt.Errorf("Insufficient input")
		return nil, "", err
	}

	if _, ok := xYangSpecMap[xpath]; ok {
		if xYangSpecMap[xpath].yangEntry == nil {
			log.Warningf("Yang Entry not available for xpath %v", xpath)
			return nil, "", nil
		}
	}

	keyNameList := yangKeyFromEntryGet(xYangSpecMap[xpath].yangEntry)
	id := xYangSpecMap[xpath].keyLevel
	keyDataList := strings.SplitN(dbKey, dbKeySep, int(id))
	uriWithKey := fmt.Sprintf("%v", xpath)
	uriWithKeyCreate := true
	if len(keyDataList) == 0 {
		keyDataList = append(keyDataList, dbKey)
	}

	/* if URI contins key, use it else use xpath */
	if strings.Contains(uri, "[") {
		if strings.HasSuffix(uri, "]") || strings.HasSuffix(uri, "]/") {
			uriXpath, _, _ := XfmrRemoveXPATHPredicates(uri)
			if uriXpath == xpath {
				uriWithKeyCreate = false
			}
		}
		uriWithKey = fmt.Sprintf("%v", uri)
	}

	if len(xYangSpecMap[xpath].xfmrKey) > 0 {
		var dbs [db.MaxDB]*db.DB
		inParams := formXfmrInputRequest(nil, dbs, db.MaxDB, nil, uri, requestUri, GET, dbKey, dbDataMap, nil, nil, txCache)
		inParams.table = tableName
		rmap, err := keyXfmrHandlerFunc(inParams, xYangSpecMap[xpath].xfmrKey)
		if err != nil {
			return nil, "", err
		}
		if uriWithKeyCreate {
			for k, v := range rmap {
				if reflect.TypeOf(v).Kind() == reflect.String {
					v = escapeKeyValForSplitPathAndNewPathInfo(v.(string))
				}
				uriWithKey += fmt.Sprintf("[%v=%v]", k, v)
			}
		}
		return rmap, uriWithKey, nil
	}

	if len(keyNameList) == 0 {
		return nil, "", nil
	}

	rmap := make(map[string]interface{})
	if len(keyNameList) > 1 {
		log.Warningf("No key transformer found for multi element YANG key mapping to a single redis key string, for URI %v", uri)
		errStr := fmt.Sprintf("Error processing key for list %v", uri)
		err = fmt.Errorf("%v", errStr)
		return rmap, uriWithKey, err
	}
	keyXpath := xpath + "/" + keyNameList[0]
	yangEntry, ok := xYangSpecMap[xpath].yangEntry.Dir[keyNameList[0]]
	if !ok || yangEntry == nil {
		errStr := fmt.Sprintf("Failed to find key xpath %v in xYangSpecMap or is nil, needed to fetch the yangEntry data-type", keyXpath)
		err = fmt.Errorf("%v", errStr)
		return rmap, uriWithKey, err
	}
	yngTerminalNdDtType := yangEntry.Type.Kind
	resVal, _, err := DbToYangType(yngTerminalNdDtType, keyXpath, keyDataList[0])
	if err != nil {
		err = fmt.Errorf("Failed in convert DB value type to YANG type for field %v. Key-xfmr recommended if data types differ", keyXpath)
		return rmap, uriWithKey, err
	} else {
		rmap[keyNameList[0]] = resVal
	}
	if uriWithKeyCreate {
		if reflect.TypeOf(resVal).Kind() == reflect.String {
			resVal = escapeKeyValForSplitPathAndNewPathInfo(resVal.(string))
		}
		uriWithKey += fmt.Sprintf("[%v=%v]", keyNameList[0], resVal)
	}

	return rmap, uriWithKey, nil
}

func contains(sl []string, str string) bool {
	for _, v := range sl {
		if v == str {
			return true
		}
	}
	return false
}

func isSubtreeRequest(targetUriPath string, nodePath string) bool {
	if len(targetUriPath) > 0 && len(nodePath) > 0 {
		return strings.HasPrefix(targetUriPath, nodePath)
	}
	return false
}

func getYangPathFromUri(uri string) (string, error) {
	var path *gnmi.Path
	var err error

	path, err = ygot.StringToPath(uri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		log.Warningf("Error in URI to path conversion: %v", err)
		return "", err
	}

	yangPath, yperr := ygot.PathToSchemaPath(path)
	if yperr != nil {
		log.Warningf("Error in Gnmi path to Yang path conversion: %v", yperr)
		return "", yperr
	}

	return yangPath, err
}

func yangKeyFromEntryGet(entry *yang.Entry) []string {
	var keyList []string
	if entry != nil {
		keyList = append(keyList, strings.Split(entry.Key, " ")...)
	}
	return keyList
}

func isSonicYang(path string) bool {
	return strings.HasPrefix(path, "/sonic")
}

func hasIpv6AddString(val string) bool {
	return rgpIpv6.MatchString(val)
}

func hasMacAddString(val string) bool {
	return rgpMac.MatchString(val)
}

func isMacAddString(val string) bool {
	return rgpIsMac.MatchString(val)
}

func getYangTerminalNodeTypeName(xpathPrefix string, keyName string) string {
	keyXpath := xpathPrefix + "/" + keyName
	dbEntry := getYangEntryForXPath(keyXpath)
	xfmrLogDebug("getYangTerminalNodeTypeName keyXpath: %v ", keyXpath)
	_, ok := xDbSpecMap[keyXpath]
	if ok && (dbEntry != nil) {
		yngTerminalNdTyName := dbEntry.Type.Name
		xfmrLogDebug("yngTerminalNdTyName: %v", yngTerminalNdTyName)
		return yngTerminalNdTyName
	}
	return ""
}

func sonicKeyDataAdd(dbIndex db.DBNum, keyNameList []string, xpathPrefix string, listNm string, keyStr string, resultMap map[string]interface{}) {
	var dbOpts db.Options
	var keyValList []string
	xfmrLogDebug("sonicKeyDataAdd keyNameList:%v, keyStr:%v", keyNameList, keyStr)

	if xDbSpecInfo, ok := xDbSpecMap[xpathPrefix+"/"+listNm]; ok {
		if xDbSpecInfo.xfmrKey != "" {
			inParams := formSonicXfmrInputRequest(dbIndex, xpathPrefix, keyStr, xpathPrefix+"/"+listNm)
			ret, err := sonicKeyXfmrHandlerFunc(inParams, xDbSpecInfo.xfmrKey)
			if err != nil {
				return
			}
			if len(ret) > 0 {
				if resultMap == nil {
					resultMap = make(map[string]interface{})
				}
				for keyName, keyVal := range ret {
					resultMap[keyName] = keyVal
				}
			}
			return
		}
	} else {
		xfmrLogDebug("xDbSpecmap doesn't have xpath - %v", xpathPrefix, "/", listNm)
	}

	dbOpts = getDBOptions(dbIndex)
	keySeparator := dbOpts.KeySeparator
	/* num of key separators will be less than number of keys */
	if len(keyNameList) == 1 && keySeparator == ":" {
		yngTerminalNdTyName := getYangTerminalNodeTypeName(xpathPrefix, keyNameList[0])
		if yngTerminalNdTyName == "mac-address" && isMacAddString(keyStr) {
			keyValList = strings.SplitN(keyStr, keySeparator, len(keyNameList))
		} else if (yngTerminalNdTyName == "ip-address" || yngTerminalNdTyName == "ip-prefix" || yngTerminalNdTyName == "ipv6-prefix" || yngTerminalNdTyName == "ipv6-address") && hasIpv6AddString(keyStr) {
			keyValList = strings.SplitN(keyStr, keySeparator, len(keyNameList))
		} else {
			keyValList = strings.SplitN(keyStr, keySeparator, -1)
			xfmrLogDebug("Single key non ipv6/mac address for : separator")
		}
	} else if strings.Count(keyStr, keySeparator) == len(keyNameList)-1 {
		/* number of keys will match number of key values */
		keyValList = strings.SplitN(keyStr, keySeparator, len(keyNameList))
	} else {
		/* number of dbKey values more than number of keys */
		if keySeparator == ":" && (hasIpv6AddString(keyStr) || hasMacAddString(keyStr)) {
			xfmrLogDebug("Key Str has ipv6/mac address with : separator")
			if len(keyNameList) == 2 {
				valList := strings.SplitN(keyStr, keySeparator, -1)
				/* IPV6 address is first entry */
				yngTerminalNdTyName := getYangTerminalNodeTypeName(xpathPrefix, keyNameList[0])
				if yngTerminalNdTyName == "ip-address" || yngTerminalNdTyName == "ip-prefix" || yngTerminalNdTyName == "ipv6-prefix" || yngTerminalNdTyName == "ipv6-address" || yngTerminalNdTyName == "mac-address" {
					keyValList = append(keyValList, strings.Join(valList[:len(valList)-2], keySeparator))
					keyValList = append(keyValList, valList[len(valList)-1])
				} else {
					yngTerminalNdTyName := getYangTerminalNodeTypeName(xpathPrefix, keyNameList[1])
					if yngTerminalNdTyName == "ip-address" || yngTerminalNdTyName == "ip-prefix" || yngTerminalNdTyName == "ipv6-prefix" || yngTerminalNdTyName == "ipv6-address" || yngTerminalNdTyName == "mac-address" {
						keyValList = append(keyValList, valList[0])
						keyValList = append(keyValList, strings.Join(valList[1:], keySeparator))
					} else {
						xfmrLogDebug("No ipv6 or mac address found in value. Cannot split value ")
					}
				}
				xfmrLogDebug("KeyValList has %v", keyValList)
			} else {
				xfmrLogDebug("Number of keys : %v", len(keyNameList))
			}
		} else {
			keyValList = strings.SplitN(keyStr, keySeparator, -1)
			xfmrLogDebug("Split all keys KeyValList has %v", keyValList)
		}
	}
	xfmrLogDebug("yang keys list - %v, xpathprefix - %v, DB-key string - %v, DB-key list after db key separator split - %v, dbIndex - %v", keyNameList, xpathPrefix, keyStr, keyValList, dbIndex)

	if len(keyNameList) != len(keyValList) {
		return
	}

	for i, keyName := range keyNameList {
		keyXpath := xpathPrefix + "/" + keyName
		dbEntry := getYangEntryForXPath(keyXpath)
		var resVal interface{}
		resVal = keyValList[i]
		if dbEntry == nil {
			log.Warningf("xDbSpecMap entry not found or is nil for xpath %v, hence data-type conversion cannot happen", keyXpath)
		} else {
			yngTerminalNdDtType := dbEntry.Type.Kind
			var err error
			resVal, _, err = DbToYangType(yngTerminalNdDtType, keyXpath, keyValList[i])
			if err != nil {
				log.Warningf("Failed to convert data-type for xpath %v. Key-xfmr recommended if data types differ", keyXpath)
				resVal = keyValList[i]
			}
		}

		resultMap[keyName] = resVal
	}
}

func yangToDbXfmrFunc(funcName string) string {
	return ("YangToDb_" + funcName)
}

func uriWithKeyCreate(uri string, xpathTmplt string, data interface{}) (string, error) {
	var err error
	if _, ok := xYangSpecMap[xpathTmplt]; ok {
		yangEntry := xYangSpecMap[xpathTmplt].yangEntry
		if yangEntry != nil {
			for _, k := range strings.Split(yangEntry.Key, " ") {
				keyXpath := xpathTmplt + "/" + k
				if _, keyXpathEntryOk := xYangSpecMap[keyXpath]; !keyXpathEntryOk {
					log.Warningf("No entry found in xYangSpec map for xapth %v", keyXpath)
					err = fmt.Errorf("No entry found in xYangSpec map for xapth %v", keyXpath)
					break
				}
				keyYangEntry := yangEntry.Dir[k]
				if keyYangEntry == nil {
					log.Warningf("Yang Entry not available for xpath %v", keyXpath)
					err = fmt.Errorf("Yang Entry not available for xpath %v", keyXpath)
					break
				}
				keyVal, keyValErr := unmarshalJsonToDbData(keyYangEntry, keyXpath, k, data.(map[string]interface{})[k])
				if keyValErr != nil {
					log.Warningf("unmarshalJsonToDbData() didn't unmarshal for key %v with xpath %v", k, keyXpath)
					err = keyValErr
					break
				}
				if (strings.Contains(keyVal, ":")) && (strings.HasPrefix(keyVal, OC_MDL_PFX) || strings.HasPrefix(keyVal, IETF_MDL_PFX) || strings.HasPrefix(keyVal, IANA_MDL_PFX)) {
					// identity-ref/enum has module prefix
					keyVal = strings.SplitN(keyVal, ":", 2)[1]
				}
				uri += fmt.Sprintf("[%v=%v]", k, escapeKeyVal(keyVal))
			}
		} else {
			err = fmt.Errorf("Yang Entry not available for xpath %v", xpathTmplt)
		}
	} else {
		err = fmt.Errorf("No entry in xYangSpecMap for xpath %v", xpathTmplt)
	}
	xfmrLogDebug("returning URI - %v", uri)
	return uri, err
}

func xpathRootNameGet(path string) string {
	if len(path) > 0 {
		pathl := strings.Split(path, "/")
		return ("/" + pathl[1])
	}
	return ""
}

func dbToYangXfmrFunc(funcName string) string {
	return ("DbToYang_" + funcName)
}

func uriModuleNameGet(uri string) (string, error) {
	var err error
	result := ""
	if len(uri) == 0 {
		log.Warning("Empty URI string supplied")
		err = fmt.Errorf("Empty URI string supplied")
		return result, err
	}
	urislice := strings.Split(uri, ":")
	if len(urislice) == 1 {
		log.Warningf("uri string %s does not have module name", uri)
		err = fmt.Errorf("uri string does not have module name: %v", uri)
		return result, err
	}
	moduleNm := strings.Split(urislice[0], "/")
	result = moduleNm[1]
	if len(strings.Trim(result, " ")) == 0 {
		log.Warning("Empty module name")
		err = fmt.Errorf("No module name found in URI %s", uri)
	}
	xfmrLogDebug("module name = %v", result)
	return result, err
}

func formXfmrInputRequest(d *db.DB, dbs [db.MaxDB]*db.DB, cdb db.DBNum, ygRoot *ygot.GoStruct, uri string, requestUri string, oper Operation, key string, dbDataMap *RedisDbMap, subOpDataMap map[Operation]*RedisDbMap, param interface{}, txCache interface{}) XfmrParams {
	var inParams XfmrParams
	inParams.d = d
	inParams.dbs = dbs
	inParams.curDb = cdb
	inParams.ygRoot = ygRoot
	inParams.uri = uri
	inParams.requestUri = requestUri
	inParams.oper = oper
	inParams.key = key
	inParams.dbDataMap = dbDataMap
	inParams.subOpDataMap = subOpDataMap
	inParams.param = param // generic param
	inParams.txCache = txCache.(*sync.Map)
	inParams.skipOrdTblChk = new(bool)
	inParams.isVirtualTbl = new(bool)
	inParams.pCascadeDelTbl = new([]string)
	// By default invoke the subtree at list and its child container level for CRU.
	// If the application wants optimization in subtree invocation set this flag to true
	// to not invoke subtree at child container level for CRU
	inParams.invokeCRUSubtreeOnce = new(bool)

	return inParams
}

func findByValue(m map[string]string, value string) string {
	for key, val := range m {
		if val == value {
			return key
		}
	}
	return ""
}
func findByKey(m map[string]string, key string) string {
	if val, found := m[key]; found {
		return val
	}
	return ""
}
func findInMap(m map[string]string, str string) string {
	// Check if str exists as a value in map m.
	if val := findByKey(m, str); val != "" {
		return val
	}

	// Check if str exists as a key in map m.
	if val := findByValue(m, str); val != "" {
		return val
	}

	// str doesn't exist in map m.
	return ""
}

func getDBOptions(dbNo db.DBNum) db.Options {
	var opt db.Options

	switch dbNo {
	case db.ApplDB, db.CountersDB, db.FlexCounterDB, db.AsicDB:
		opt = getDBOptionsWithSeparator(dbNo, "", ":", ":")
	case db.ConfigDB, db.StateDB:
		opt = getDBOptionsWithSeparator(dbNo, "", "|", "|")
	}

	return opt
}

func getDBOptionsWithSeparator(dbNo db.DBNum, initIndicator string, tableSeparator string, keySeparator string) db.Options {
	return (db.Options{
		DBNo:               dbNo,
		InitIndicator:      initIndicator,
		TableNameSeparator: tableSeparator,
		KeySeparator:       keySeparator,
	})
}

func getXpathFromYangEntry(entry *yang.Entry) string {
	xpath := ""
	if entry != nil {
		if _, ok := entry.Annotation["schemapath"]; ok {
			xpath = entry.Annotation["schemapath"].(string)
			/* module name is delimetered from the rest of schema path with ":" */
			xpath = string('/') + strings.Replace(xpath[1:], "/", ":", 1)

		} else {
			// Parent of terminal node is guratnteed to have a schemapath annotaion by ygot
			xpath = entry.Parent.Annotation["schemapath"].(string)
			xpath = string('/') + strings.Replace(xpath[1:], "/", ":", 1)
			xpath = xpath + "/" + entry.Name
		}
	}
	return xpath
}

func stripAugmentedModuleNames(xpath string) string {
	if !strings.HasPrefix(xpath, "/") {
		xpath = "/" + xpath
	}
	pathList := strings.Split(xpath, "/")
	pathList = pathList[1:]
	for i, pvar := range pathList {
		if (i > 0) && strings.Contains(pvar, ":") {
			pvar = strings.Split(pvar, ":")[1]
			pathList[i] = pvar
		}
	}
	path := "/" + strings.Join(pathList, "/")
	return path
}

func XfmrRemoveXPATHPredicates(uri string) (string, []string, error) {
	var uriList []string
	var pathList []string
	uriList = SplitPath(uri)

	// Strip keys for xpath creation
	for _, path := range uriList {
		si := strings.Index(path, "[")
		if si != -1 {
			pathList = append(pathList, path[:si])
		} else {
			pathList = append(pathList, path)
		}
	}

	inPath := strings.Join(pathList, "/")
	if !strings.HasPrefix(uri, "..") {
		inPath = "/" + inPath
	}

	xpath := stripAugmentedModuleNames(inPath)
	return xpath, uriList, nil
}

/* Extract key vars, create db key and xpath */
func xpathKeyExtract(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, path string, requestUri string, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, subOpDataMap map[Operation]*RedisDbMap, txCache interface{}, xfmrTblKeyCache map[string]tblKeyCache) (xpathTblKeyExtractRet, error) {
	xfmrLogDebug("In uri(%v), reqUri(%v), oper(%v)", path, requestUri, oper)
	var retData xpathTblKeyExtractRet
	keyStr := ""
	curPathWithKey := ""
	cdb := db.ConfigDB
	var dbs [db.MaxDB]*db.DB
	var err error
	var isUriForListInstance bool
	var pathList []string

	retData.xpath = ""
	retData.tableName = ""
	retData.dbKey = ""
	retData.isVirtualTbl = false

	isUriForListInstance = false
	retData.xpath, pathList, _ = XfmrRemoveXPATHPredicates(path)
	xpathInfo, ok := xYangSpecMap[retData.xpath]
	if !ok {
		log.Warningf("No entry found in xYangSpecMap for xpath %v.", retData.xpath)
		return retData, err
	}
	// for SUBSCRIBE reuestUri = path
	requestUriYangType := xpathInfo.yangType
	if requestUriYangType == YANG_LIST {
		if strings.HasSuffix(path, "]") { //uri is for list instance
			isUriForListInstance = true
		}
	}
	cdb = xpathInfo.dbIndex
	dbOpts := getDBOptions(cdb)
	keySeparator := dbOpts.KeySeparator
	if len(xpathInfo.delim) > 0 {
		keySeparator = xpathInfo.delim
	}
	xpathList := strings.Split(retData.xpath, "/")
	xpathList = xpathList[1:]
	yangXpath := ""
	xfmrLogDebug("path elements are : %v", pathList)
	for i, k := range pathList {
		curPathWithKey += "/" + k
		callKeyXfmr := true
		yangXpath += "/" + xpathList[i]
		xpathInfo, ok := xYangSpecMap[yangXpath]
		if ok {
			yangType := xpathInfo.yangType
			/* when deleting a specific element from leaf-list query URI is of the form
			   /prefix-path/leafList-field-name[leafList-field-name=value].
			   Here the syntax is like a list-key instance enclosed in square
			   brackets .So avoid list key instance like processing for such a case
			*/
			if yangType == YANG_LEAF_LIST {
				break
			}
			if strings.Contains(k, "[") {
				if len(keyStr) > 0 {
					keyStr += keySeparator
				}
				if len(xYangSpecMap[yangXpath].xfmrKey) > 0 {
					if xfmrTblKeyCache != nil {
						if tkCache, _ok := xfmrTblKeyCache[curPathWithKey]; _ok {
							if len(tkCache.dbKey) != 0 {
								keyStr = tkCache.dbKey
								callKeyXfmr = false
							}
						}
					}
					if callKeyXfmr {
						xfmrFuncName := yangToDbXfmrFunc(xYangSpecMap[yangXpath].xfmrKey)
						inParams := formXfmrInputRequest(d, dbs, cdb, ygRoot, curPathWithKey, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
						if oper == GET {
							ret, err := XlateFuncCall(xfmrFuncName, inParams)
							if err != nil {
								retData.dbKey, retData.tableName, retData.xpath = "", "", ""
								xfmrLogDebug("keyXfmr %v failed at path %v: %v. Please check key-xfmr", xfmrFuncName, curPathWithKey, err)
								xfmrLogDebug("Return at uri(%v) - xpath(%v), key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, keyStr, retData.tableName, retData.isVirtualTbl)

								return retData, err
							}
							if ret != nil {
								keyStr = ret[0].Interface().(string)
							}
						} else {
							ret, err := keyXfmrHandler(inParams, xYangSpecMap[yangXpath].xfmrKey)
							if err != nil {
								retData.dbKey, retData.tableName, retData.xpath = "", "", ""
								xfmrLogDebug("Return at uri(%v) - xpath(%v), curPathWithKey(%v) key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, curPathWithKey, keyStr, retData.tableName, retData.isVirtualTbl)
								return retData, err
							}
							keyStr = ret
						}
						if xfmrTblKeyCache != nil {
							if _, _ok := xfmrTblKeyCache[curPathWithKey]; !_ok {
								xfmrTblKeyCache[curPathWithKey] = tblKeyCache{}
							}
							tkCache := xfmrTblKeyCache[curPathWithKey]
							tkCache.dbKey = keyStr
							xfmrTblKeyCache[curPathWithKey] = tkCache
						}
					}
				} else if xYangSpecMap[yangXpath].keyName != nil {
					keyStr += *xYangSpecMap[yangXpath].keyName
				} else {
					/* multi-leaf YANG key together forms a single key-string in redis.
					There should be key-transformer, if not then the YANG key leaves
					will be concatenated with respective default DB type key-delimiter
					*/
					if (yangType == YANG_LIST) && (xpathInfo.yangEntry != nil) {
						xfmrLogDebug("No key-xfmr at list %v, 1:1 mapping case. Concatenating YANG keys", curPathWithKey)
						keyNmList := strings.Split(xpathInfo.yangEntry.Key, " ")
						pathInfo := NewPathInfo("/" + k)
						if (pathInfo != nil) && (len(pathInfo.Vars) > 0) {
							for _, keyNm := range keyNmList {
								if len(keyStr) > 0 {
									keyStr += keySeparator
								}
								if pathInfo.HasVar(keyNm) {
									keyStr += pathInfo.Var(keyNm)
								}
							}
						}
					}
				}
			} else if len(xYangSpecMap[yangXpath].xfmrKey) > 0 {
				if xfmrTblKeyCache != nil {
					if tkCache, _ok := xfmrTblKeyCache[curPathWithKey]; _ok {
						if len(tkCache.dbKey) != 0 {
							keyStr = tkCache.dbKey
							callKeyXfmr = false
						}
					}
				}
				if callKeyXfmr {
					xfmrFuncName := yangToDbXfmrFunc(xYangSpecMap[yangXpath].xfmrKey)
					inParams := formXfmrInputRequest(d, dbs, cdb, ygRoot, curPathWithKey, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
					if oper == GET {
						ret, err := XlateFuncCall(xfmrFuncName, inParams)
						if err != nil {
							retData.dbKey, retData.tableName, retData.xpath = "", "", ""
							xfmrLogDebug("Error from keyXfmr %v at path %v: %v", xfmrFuncName, curPathWithKey, err)
							xfmrLogDebug("Return at uri(%v) - xpath(%v), curPathWithKey(%v) key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, curPathWithKey, keyStr, retData.tableName, retData.isVirtualTbl)
							return retData, err
						}
						if ret != nil {
							keyStr = ret[0].Interface().(string)
						}
					} else {
						ret, err := keyXfmrHandler(inParams, xYangSpecMap[yangXpath].xfmrKey)
						if (yangType != YANG_LIST) && (err != nil) {
							retData.dbKey, retData.tableName, retData.xpath = "", "", ""
							xfmrLogDebug("Return at uri(%v) - xpath(%v), curPathWithKey(%v) key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, curPathWithKey, keyStr, retData.tableName, retData.isVirtualTbl)
							return retData, err
						}
						keyStr = ret
					}
					if xfmrTblKeyCache != nil {
						if _, _ok := xfmrTblKeyCache[curPathWithKey]; !_ok {
							xfmrTblKeyCache[curPathWithKey] = tblKeyCache{}
						}
						tkCache := xfmrTblKeyCache[curPathWithKey]
						tkCache.dbKey = keyStr
						xfmrTblKeyCache[curPathWithKey] = tkCache
					}
				}
			} else if xYangSpecMap[yangXpath].keyName != nil {
				keyStr += *xYangSpecMap[yangXpath].keyName
			}
		}
	}
	curPathWithKey = strings.TrimSuffix(curPathWithKey, "/")
	if !strings.HasPrefix(curPathWithKey, "/") {
		curPathWithKey = "/" + curPathWithKey
	}
	retData.dbKey = keyStr
	tblPtr := xpathInfo.tableName
	if tblPtr != nil && *tblPtr != XFMR_NONE_STRING {
		retData.tableName = *tblPtr
	} else if xpathInfo.xfmrTbl != nil {
		inParams := formXfmrInputRequest(d, dbs, cdb, ygRoot, curPathWithKey, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
		if oper == GET {
			inParams.dbDataMap = dbDataMap
		}
		retData.tableName, err = tblNameFromTblXfmrGet(*xpathInfo.xfmrTbl, inParams, xfmrTblKeyCache)
		if inParams.isVirtualTbl != nil {
			retData.isVirtualTbl = *(inParams.isVirtualTbl)
		}
		if err != nil && oper != GET {
			xfmrLogDebug("Return at uri(%v) - xpath(%v), key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, keyStr, retData.tableName, retData.isVirtualTbl)
			return retData, err
		}
	}
	if (oper == SUBSCRIBE) && (strings.TrimSpace(keyStr) == "") && (requestUriYangType == YANG_LIST) && (!isUriForListInstance) {
		keyStr = "*"
	}
	retData.dbKey = keyStr
	xfmrLogDebug("Return at uri(%v) - xpath(%v), key(%v), tableName(%v), isVirtualTbl(%v)", path, retData.xpath, keyStr, retData.tableName, retData.isVirtualTbl)
	return retData, err
}

func dbTableFromUriGet(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, requestUri string, subOpDataMap map[Operation]*RedisDbMap, txCache interface{}, xfmrTblKeyCache map[string]tblKeyCache) (string, error) {
	tableName := ""
	var err error
	cdb := db.ConfigDB
	var dbs [db.MaxDB]*db.DB

	xPath, _, _ := XfmrRemoveXPATHPredicates(uri)
	xpathInfo, ok := xYangSpecMap[xPath]
	if !ok {
		log.Warningf("No entry found in xYangSpecMap for xpath %v.", xPath)
		return tableName, err
	}

	tblPtr := xpathInfo.tableName
	if tblPtr != nil && *tblPtr != XFMR_NONE_STRING {
		tableName = *tblPtr
	} else if xpathInfo.xfmrTbl != nil {
		inParams := formXfmrInputRequest(d, dbs, cdb, ygRoot, uri, requestUri, oper, "", nil, subOpDataMap, nil, txCache)
		tableName, err = tblNameFromTblXfmrGet(*xpathInfo.xfmrTbl, inParams, xfmrTblKeyCache)
	}
	return tableName, err
}

func sonicXpathKeyExtract(path string) (string, string, string) {
	xfmrLogDebug("In uri(%v)", path)
	xpath, keyStr, tableName, fldNm := "", "", "", ""
	var err error
	lpath := path
	xpath, _, err = XfmrRemoveXPATHPredicates(path)
	if err != nil || len(xpath) == 0 {
		return xpath, keyStr, tableName
	}

	pathsubStr := strings.Split(xpath, "/")
	if len(pathsubStr) > SONIC_FIELD_INDEX {
		fldNm = pathsubStr[SONIC_FIELD_INDEX]
		xfmrLogDebug("Field Name : %v", fldNm)
	}
	if len(pathsubStr) > SONIC_TABLE_INDEX {
		if strings.Contains(pathsubStr[2], "[") {
			tableName = strings.Split(pathsubStr[SONIC_TABLE_INDEX], "[")[0]
		} else {
			tableName = pathsubStr[SONIC_TABLE_INDEX]
		}
		dbInfo, ok := xDbSpecMap[tableName]
		cdb := db.ConfigDB
		if !ok {
			xfmrLogDebug("No entry in xDbSpecMap for xpath %v in order to fetch DB index", tableName)
			return xpath, keyStr, tableName
		}
		cdb = dbInfo.dbIndex
		dbOpts := getDBOptions(cdb)
		if dbInfo.keyName != nil {
			keyStr = *dbInfo.keyName
		} else {
			/* chomp off the field portion to avoid processing specific item delete in leaf-list
			   eg. /sonic-acl:sonic-acl/ACL_TABLE/ACL_TABLE_LIST[aclname=MyACL2_ACL_IPV4]/ports[ports=Ethernet12]
			*/
			if fldNm != "" {
				pathLst := splitUri(path)
				xfmrLogDebug("pathList after URI split %v", pathLst)
				lpath = "/" + strings.Join(pathLst[:SONIC_FIELD_INDEX-1], "/")
				xfmrLogDebug("path after removing the field portion %v", lpath)
			}
			if len(pathsubStr) > SONIC_TBL_CHILD_INDEX {
				tblChldNm := pathsubStr[SONIC_TBL_CHILD_INDEX]
				xfmrLogDebug("Table Child Name : %v", tblChldNm)
				tblChldXpath := tableName + "/" + tblChldNm
				if specTblChldInfo, ok := xDbSpecMap[tblChldXpath]; ok {
					if specTblChldInfo.yangType == YANG_CONTAINER {
						keyStr = tblChldNm
					} else if specTblChldInfo.yangType == YANG_LIST {
						pathInfo := NewPathInfo(lpath)
						if (pathInfo != nil) && (len(pathInfo.Vars) > 0) {
							for idx, keyNm := range specTblChldInfo.keyList {
								if idx > 0 {
									keyStr += dbOpts.KeySeparator
								}
								if pathInfo.HasVar(keyNm) {
									keyStr += pathInfo.Var(keyNm)
								}
							}
						}
					}
				}
			}
		}
	}
	xfmrLogDebug("Return uri(%v), xpath(%v), key(%v), tableName(%v)", path, xpath, keyStr, tableName)
	return xpath, keyStr, tableName
}

func getYangMdlToSonicMdlList(moduleNm string) []string {
	var sncMdlList []string
	if len(xDbSpecTblSeqnMap) == 0 {
		xfmrLogInfo("xDbSpecTblSeqnMap is empty.")
		return sncMdlList
	}
	if strings.HasPrefix(moduleNm, SONIC_MDL_PFX) {
		sncMdlList = append(sncMdlList, moduleNm)
	} else {
		//can be optimized if there is a way to know sonic modules, a given OC-Yang spans over
		for sncMdl := range xDbSpecTblSeqnMap {
			sncMdlList = append(sncMdlList, sncMdl)
		}
	}
	return sncMdlList
}

func yangFloatIntToGoType(t yang.TypeKind, v float64) (interface{}, error) {
	switch t {
	case yang.Yint8:
		return int8(v), nil
	case yang.Yint16:
		return int16(v), nil
	case yang.Yint32:
		return int32(v), nil
	case yang.Yuint8:
		return uint8(v), nil
	case yang.Yuint16:
		return uint16(v), nil
	case yang.Yuint32:
		return uint32(v), nil
	}
	return nil, fmt.Errorf("unexpected YANG type %v", t)
}

func unmarshalJsonToDbData(schema *yang.Entry, fieldXpath string, fieldName string, value interface{}) (string, error) {
	var data string

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%v", v), nil
	}

	ykind := schema.Type.Kind
	if ykind == yang.Yleafref {
		ykind = getLeafrefRefdYangType(ykind, fieldXpath)
	}

	switch ykind {
	case yang.Ystring, yang.Ydecimal64, yang.Yint64, yang.Yuint64,
		yang.Yenum, yang.Ybool, yang.Ybinary, yang.Yidentityref, yang.Yunion:
		data = fmt.Sprintf("%v", value)

	case yang.Yint8, yang.Yint16, yang.Yint32,
		yang.Yuint8, yang.Yuint16, yang.Yuint32:
		pv, err := yangFloatIntToGoType(ykind, value.(float64))
		if err != nil {
			errStr := fmt.Sprintf("error parsing %v for schema %s: %v", value, schema.Name, err)
			return "", tlerr.InternalError{Format: errStr}
		}
		data = fmt.Sprintf("%v", pv)
	default:
		// TODO - bitset, empty
		data = fmt.Sprintf("%v", value)
	}

	return data, nil
}

func copyYangXpathSpecData(dstNode *yangXpathInfo, srcNode *yangXpathInfo) {
	if dstNode != nil && srcNode != nil {
		*dstNode = *srcNode
	}
}

func isJsonDataEmpty(jsonData string) bool {
	return string(jsonData) == "{}"
}

func getFileNmLineNumStr() string {
	pc, AbsfileName, lineNum, _ := runtime.Caller(2)
	fileNmElems := strings.Split(AbsfileName, "/")
	fileNm := fileNmElems[len(fileNmElems)-1]
	fn := runtime.FuncForPC(pc)
	funcName := ""
	if fn != nil {
		funcNameElems := strings.Split(fn.Name(), "/")
		funcName = strings.Replace(funcNameElems[len(funcNameElems)-1], "transformer.", "", 1)
	}
	fNmLnoStr := fmt.Sprintf("[%v:%v]%v ", fileNm, lineNum, funcName)
	return fNmLnoStr
}

func xfmrLogInfo(format string, args ...interface{}) {
	fNmLnoStr := getFileNmLineNumStr()
	log.Infof(fNmLnoStr+format, args...)
}

func xfmrLogDebug(format string, args ...interface{}) {
	if log.V(5) {
		fNmLnoStr := getFileNmLineNumStr()
		log.Infof(fNmLnoStr+format, args...)
	}
}

func formXfmrDbInputRequest(oper Operation, d db.DBNum, tableName string, key string, field string, value string) XfmrDbParams {
	var inParams XfmrDbParams
	inParams.oper = oper
	inParams.dbNum = d
	inParams.tableName = tableName
	inParams.key = key
	inParams.fieldName = field
	inParams.value = value
	return inParams
}

func hasKeyValueXfmr(tblName string) bool {
	if specTblInfo, ok := xDbSpecMap[tblName]; ok {
		for _, lname := range specTblInfo.listName {
			listXpath := tblName + "/" + lname
			if specListInfo, ok := xDbSpecMap[listXpath]; ok {
				for _, key := range specListInfo.keyList {
					keyXpath := tblName + "/" + key
					if specKeyInfo, ok := xDbSpecMap[keyXpath]; ok {
						if specKeyInfo.xfmrValue != nil {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func dbKeyValueXfmrHandler(oper Operation, dbNum db.DBNum, tblName string, dbKey string) (string, error) {
	var err error
	var keyValList []string

	xfmrLogDebug("dbKeyValueXfmrHandler: oper(%v), db(%v), tbl(%v), dbKey(%v)",
		oper, dbNum, tblName, dbKey)
	if specTblInfo, ok := xDbSpecMap[tblName]; ok {
		for _, lname := range specTblInfo.listName {
			listXpath := tblName + "/" + lname
			keyMap := make(map[string]interface{})

			if specListInfo, ok := xDbSpecMap[listXpath]; ok && len(specListInfo.keyList) > 0 {
				sonicKeyDataAdd(dbNum, specListInfo.keyList, tblName, lname, dbKey, keyMap)

				if len(keyMap) == len(specListInfo.keyList) {
					for _, kname := range specListInfo.keyList {
						keyXpath := tblName + "/" + kname
						curKeyVal := fmt.Sprintf("%v", keyMap[kname])
						if kInfo, ok := xDbSpecMap[keyXpath]; ok && xDbSpecMap[keyXpath].xfmrValue != nil {
							inParams := formXfmrDbInputRequest(oper, dbNum, tblName, dbKey, kname, curKeyVal)
							curKeyVal, err = valueXfmrHandler(inParams, *kInfo.xfmrValue)
							if err != nil {
								log.Warningf("value-xfmr: keypath(\"%v\") value (\"%v\"):err(%v).",
									keyXpath, curKeyVal, err)
								return "", err
							}
						}
						keyValList = append(keyValList, curKeyVal)
					}
				}
			}
		}
	}

	dbOpts := getDBOptions(dbNum)
	retKey := strings.Join(keyValList, dbOpts.KeySeparator)
	xfmrLogDebug("dbKeyValueXfmrHandler: tbl(%v), dbKey(%v), retKey(%v), keyValList(%v)",
		tblName, dbKey, retKey, keyValList)

	return retKey, nil
}

func dbDataXfmrHandler(resultMap map[Operation]map[db.DBNum]map[string]map[string]db.Value) error {
	xfmrLogDebug("Received  resultMap(%v)", resultMap)
	for oper, dbDataMap := range resultMap {
		for dbNum, tblData := range dbDataMap {
			for tblName, data := range tblData {
				if specTblInfo, ok := xDbSpecMap[tblName]; ok && specTblInfo.hasXfmrFn {
					skipKeySet := make(map[string]bool)
					for dbKey, fldData := range data {
						if _, ok := skipKeySet[dbKey]; !ok {
							for fld, val := range fldData.Field {
								fldName := fld
								if strings.HasSuffix(fld, "@") {
									fldName = strings.Split(fld, "@")[0]
								}
								/* check & invoke value-xfmr */
								fldXpath := tblName + "/" + fldName
								if fInfo, ok := xDbSpecMap[fldXpath]; ok && fInfo.xfmrValue != nil {
									inParams := formXfmrDbInputRequest(oper, dbNum, tblName, dbKey, fld, val)
									retVal, err := valueXfmrHandler(inParams, *fInfo.xfmrValue)
									if err != nil {
										log.Warningf("value-xfmr:fldpath(\"%v\") val(\"%v\"):err(\"%v\").",
											fldXpath, val, err)
										return err
									}
									resultMap[oper][dbNum][tblName][dbKey].Field[fld] = retVal
								}
							}

							/* split tblkey and invoke value-xfmr if present */
							if hasKeyValueXfmr(tblName) {
								retKey, err := dbKeyValueXfmrHandler(oper, dbNum, tblName, dbKey)
								if err != nil {
									return err
								}
								/* cache processed keys */
								skipKeySet[retKey] = true
								if dbKey != retKey {
									resultMap[oper][dbNum][tblName][retKey] = resultMap[oper][dbNum][tblName][dbKey]
									delete(resultMap[oper][dbNum][tblName], dbKey)
								}
							}
						}
					}
				}
			}
		}
	}
	xfmrLogDebug("Transformed resultMap(%v)", resultMap)
	return nil
}

func formXlateFromDbParams(d *db.DB, dbs [db.MaxDB]*db.DB, cdb db.DBNum, ygRoot *ygot.GoStruct, uri string, requestUri string, xpath string, oper Operation, tbl string, tblKey string, dbDataMap *RedisDbMap, txCache interface{}, resultMap map[string]interface{}, validate bool) xlateFromDbParams {
	var inParamsForGet xlateFromDbParams
	inParamsForGet.d = d
	inParamsForGet.dbs = dbs
	inParamsForGet.curDb = cdb
	inParamsForGet.ygRoot = ygRoot
	inParamsForGet.uri = uri
	inParamsForGet.requestUri = requestUri
	inParamsForGet.xpath = xpath
	inParamsForGet.oper = oper
	inParamsForGet.tbl = tbl
	inParamsForGet.tblKey = tblKey
	inParamsForGet.dbDataMap = dbDataMap
	inParamsForGet.txCache = txCache
	inParamsForGet.resultMap = resultMap
	inParamsForGet.validate = validate

	return inParamsForGet
}

func formXlateToDbParam(d *db.DB, ygRoot *ygot.GoStruct, oper Operation, uri string, requestUri string, xpathPrefix string, keyName string, jsonData interface{}, resultMap map[Operation]RedisDbMap, result map[string]map[string]db.Value, txCache interface{}, tblXpathMap map[string]map[string]map[string]bool, subOpDataMap map[Operation]*RedisDbMap, pCascadeDelTbl *[]string, xfmrErr *error, name string, value interface{}, tableName string, invokeSubtreeOnceMap map[string]map[string]bool) xlateToParams {
	var inParamsForSet xlateToParams
	inParamsForSet.d = d
	inParamsForSet.ygRoot = ygRoot
	inParamsForSet.oper = oper
	inParamsForSet.uri = uri
	inParamsForSet.requestUri = requestUri
	inParamsForSet.xpath = xpathPrefix
	inParamsForSet.keyName = keyName
	inParamsForSet.jsonData = jsonData
	inParamsForSet.resultMap = resultMap
	inParamsForSet.result = result
	inParamsForSet.txCache = txCache.(*sync.Map)
	inParamsForSet.tblXpathMap = tblXpathMap
	inParamsForSet.subOpDataMap = subOpDataMap
	inParamsForSet.pCascadeDelTbl = pCascadeDelTbl
	inParamsForSet.xfmrErr = xfmrErr
	inParamsForSet.name = name
	inParamsForSet.value = value
	inParamsForSet.tableName = tableName
	inParamsForSet.invokeCRUSubtreeOnceMap = invokeSubtreeOnceMap

	return inParamsForSet
}

func xlateUnMarshallUri(ygRoot *ygot.GoStruct, uri string) (*interface{}, error) {
	if len(uri) == 0 {
		errMsg := errors.New("Error: URI is empty")
		log.Warning(errMsg)
		return nil, errMsg
	}

	path, err := ygot.StringToPath(uri, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		return nil, err
	}

	for _, p := range path.Elem {
		if strings.Contains(p.Name, ":") {
			pathSlice := strings.Split(p.Name, ":")
			p.Name = pathSlice[len(pathSlice)-1]
		}
	}

	deviceObj := (*ygRoot).(*ocbinds.Device)
	ygNode, _, errYg := ytypes.GetOrCreateNode(ocbSch.RootSchema(), deviceObj, path)

	if errYg != nil {
		log.Warning("Error in creating the target object: ", errYg)
		return nil, errYg
	}

	return &ygNode, nil
}

func (ygtXltr *ygotXlator) validate() error {
	uri := ygtXltr.ygotCtx.relUri

	if len(uri) == 0 {
		ygtXltr.ygotCtx.err = errors.New("Error: URI is empty")
		log.Warning("ygotXlator: validate: " + ygtXltr.ygotCtx.err.Error())
		return ygtXltr.ygotCtx.err
	}

	if ygtXltr.ygotCtx.ygParentObj == nil {
		ygtXltr.ygotCtx.err = fmt.Errorf("ygot object is nil for the URI %v", uri)
		log.Warning("ygotXlator: validate:: " + ygtXltr.ygotCtx.err.Error())
		return ygtXltr.ygotCtx.err
	}

	if ygtXltr.ygotCtx.ygSchema == nil {
		ygtXltr.ygotCtx.err = fmt.Errorf("ygSchema is nil for the URI %v; for the ygot object: %v", uri, ygtXltr.getParentObjName())
		log.Warning("ygotXlator: validate:: " + ygtXltr.ygotCtx.err.Error())
		return ygtXltr.ygotCtx.err
	}
	return nil
}

func (ygtXltr ygotXlator) getParentObjName() string {
	return fmt.Sprintf("%v", reflect.ValueOf(*ygtXltr.ygotCtx.ygParentObj))
}

func (ygtXltr ygotXlator) uriToPath() (*gnmi.Path, error) {
	path, err := ygot.StringToPath(ygtXltr.ygotCtx.relUri, ygot.StructuredPath)
	if err != nil {
		ygtXltr.ygotCtx.err = fmt.Errorf("error: %v for the ygSchema: %v; ygot obj: %v; URI: %v",
			err.Error(), ygtXltr.ygotCtx.ygSchema.Name, ygtXltr.getParentObjName(), ygtXltr.ygotCtx.relUri)
		log.Warning(ygtXltr.ygotCtx.err)
		return nil, ygtXltr.ygotCtx.err
	}
	ygtXltr.stripModulePrefix(path)
	return path, nil
}

func (ygtXltr ygotXlator) stripModulePrefix(path *gnmi.Path) {
	for _, p := range path.Elem {
		if strings.Contains(p.Name, ":") {
			pathSlice := strings.Split(p.Name, ":")
			p.Name = pathSlice[len(pathSlice)-1]
		}
	}
}

func (ygtXltr ygotXlator) getListPath(path *gnmi.Path) *gnmi.Path {
	listPath := &gnmi.Path{}
	listPath.Elem = append(listPath.Elem, &gnmi.PathElem{Name: path.Elem[0].Name})
	return listPath
}

func (ygtXltr ygotXlator) unmarshalListKey(path *gnmi.Path) error {

	listPath := ygtXltr.getListPath(path)
	objIntf, ygListSchema, err := ytypes.GetOrCreateTargetNode(ygtXltr.ygotCtx.ygSchema, *ygtXltr.ygotCtx.ygParentObj, listPath)
	if err != nil {
		objName := fmt.Sprintf("%v", reflect.ValueOf(*ygtXltr.ygotCtx.ygParentObj))
		return fmt.Errorf("error in getting the target object: %v; for the ygSchema: %v; "+
			"ygot obj: %v for the given URI: %v",
			err.Error(), ygtXltr.ygotCtx.ygSchema.Name, objName, listPath)
	}

	listObj, err := ytypes.UnmarshalListKey(ygListSchema, objIntf, path.Elem[0].Key)
	if err != nil {
		objName := fmt.Sprintf("%v", reflect.ValueOf(objIntf))
		return fmt.Errorf("error in creating the target ygot list struct with key: %v; for the ygSchema: %v; "+
			"ygot obj: %v for the given path: %v",
			err.Error(), ygListSchema.Name, objName, path)
	} else if listObj == nil {
		objName := fmt.Sprintf("%v", reflect.ValueOf(objIntf))
		return fmt.Errorf("error in creating the target ygot list struct with key for the ygSchema: %v; "+
			"ygot obj: %v for the given path: %v", ygListSchema.Name, objName, path)
	}

	var ygStructPtr *ygot.GoStruct
	if ygStruct, ok := listObj.(ygot.GoStruct); ok {
		ygStructPtr = &ygStruct
	} else {
		objName := fmt.Sprintf("%v", reflect.ValueOf(objIntf))
		return fmt.Errorf("error in casting the target ygot list struct with key for the ygSchema: %v; "+
			"ygot obj: %v for the given path: %v", ygListSchema.Name, objName, path)
	}

	retPath := util.PopGNMIPath(path)
	path.Elem = retPath.Elem
	if len(path.GetElem()) == 0 {
		ygtXltr.ygotCtx.trgtYgSchema = ygListSchema
		ygtXltr.ygotCtx.trgtYgObj = ygStructPtr
	} else {
		ygtXltr.ygotCtx.ygParentObj = ygStructPtr
		ygtXltr.ygotCtx.ygSchema = ygListSchema
	}

	return nil
}

func (ygtXltr ygotXlator) translate() error {
	if err := ygtXltr.validate(); err != nil {
		return err
	}

	path, err := ygtXltr.uriToPath()
	if err != nil {
		return err
	}

	if len(path.Elem) == 0 {
		ygtXltr.ygotCtx.err = fmt.Errorf("path is empty for the given uri %v; for the ygot object: %v", ygtXltr.ygotCtx.relUri, ygtXltr.getParentObjName())
		return ygtXltr.ygotCtx.err
	}

	if len(path.Elem[0].Key) > 0 {
		if err := ygtXltr.unmarshalListKey(path); err != nil {
			ygtXltr.ygotCtx.err = err
			log.Warning(ygtXltr.ygotCtx.err)
			return ygtXltr.ygotCtx.err
		}
		if len(path.GetElem()) == 0 {
			return nil
		}
	}

	ygNode, ygEntry, err := ytypes.GetOrCreateTargetNode(ygtXltr.ygotCtx.ygSchema, *ygtXltr.ygotCtx.ygParentObj, path)
	if err != nil {
		ygtXltr.ygotCtx.err = fmt.Errorf("error in getting the target object: %v; for the ygSchema: %v; "+
			"ygot obj: %v for the given URI: %v",
			err.Error(), ygtXltr.ygotCtx.ygSchema.Name, ygtXltr.getParentObjName(), ygtXltr.ygotCtx.relUri)
		log.Warning(ygtXltr.ygotCtx.err)
		return ygtXltr.ygotCtx.err
	}

	ygtXltr.ygotCtx.trgtYgSchema = ygEntry
	if ygStruct, ok := ygNode.(ygot.GoStruct); ok {
		ygtXltr.ygotCtx.trgtYgObj = &ygStruct
	}
	return nil
}

func splitUri(uri string) []string {
	pathList := SplitPath(uri)
	xfmrLogDebug("uri: %v ", uri)
	xfmrLogDebug("uri path elems: %v", pathList)
	return pathList
}

func dbTableExists(d *db.DB, tableName string, dbKey string, oper Operation) (bool, error) {
	var err error
	// Read the table entry from DB
	if len(tableName) > 0 {
		if hasKeyValueXfmr(tableName) {
			if oper == GET { //value tranformer callback decides based on oper type
				oper = CREATE
			}
			retKey, err := dbKeyValueXfmrHandler(oper, d.Opts.DBNo, tableName, dbKey)
			if err != nil {
				return false, err
			}
			xfmrLogDebug("dbKeyValueXfmrHandler() returned db key %v", retKey)
			dbKey = retKey
		}

		dbTblSpec := &db.TableSpec{Name: tableName}

		if strings.Contains(dbKey, "*") {
			keys, derr := d.GetKeysByPattern(dbTblSpec, dbKey)
			if derr != nil {
				log.Warningf("Failed to get keys for tbl(%v) dbKey pattern %v error: %v", tableName, dbKey, derr)
				err = tlerr.NotFound("Resource not found")
				return false, err
			}
			xfmrLogDebug("keys for table %v are %v", tableName, keys)
			if len(keys) > 0 {
				return true, nil
			} else {
				log.Warningf("dbKey %v does not exist in DB for table %v", dbKey, tableName)
				err = tlerr.NotFound("Resource not found")
				return false, err
			}
		} else {

			existingEntry, derr := d.GetEntry(dbTblSpec, db.Key{Comp: []string{dbKey}})
			if derr != nil {
				log.Warningf("GetEntry failed for table: %v, key: %v err: %v", tableName, dbKey, derr)
				err = tlerr.NotFound("Resource not found")
				return false, err
			}
			return existingEntry.IsPopulated(), err
		}
	} else {
		log.Warning("Empty table name received")
		return false, nil
	}
}

func dbTableExistsInDbData(dbNo db.DBNum, table string, dbKey string, dbData RedisDbMap) bool {
	xfmrLogDebug("received DB no - %v, table - %v, dbkey - %v", dbNo, table, dbKey)
	if _, exists := dbData[dbNo][table][dbKey]; exists {
		return true
	} else {
		return false
	}
}

func leafListInstExists(leafListInDbVal string, checkLeafListInstVal string) bool {
	/*function to check if leaf-list DB value contains the given instance*/
	exists := false
	xfmrLogDebug("received value of leaf-list in DB - %v,  Value to be checked if exists in leaf-list - %v", leafListInDbVal, checkLeafListInstVal)
	leafListItemLst := strings.Split(leafListInDbVal, ",")
	for idx := range leafListItemLst {
		if leafListItemLst[idx] == checkLeafListInstVal {
			exists = true
			xfmrLogDebug("Leaf-list instance exists")
			break
		}
	}
	return exists
}

func extractLeafListInstFromUri(uri string) (string, error) {
	/*function to extract leaf-list instance value coming as part of uri
	Handling [ ] in value*/
	xfmrLogDebug("received URI - %v", uri)
	var leafListInstVal string
	var yangType yangElementType
	err := fmt.Errorf("Unable to extract leaf-list instance value for URI - %v", uri)

	xpath, _, xerr := XfmrRemoveXPATHPredicates(uri)
	if !isSonicYang(uri) {
		specInfo, ok := xYangSpecMap[xpath]
		if !ok {
			return leafListInstVal, xerr
		}
		yangType = specInfo.yangType
		if !(yangType == YANG_LEAF_LIST) {
			return leafListInstVal, err
		}
	} else {
		tokens := strings.Split(xpath, "/")
		fieldName := ""
		tableName := ""
		if len(tokens) > SONIC_FIELD_INDEX {
			fieldName = tokens[SONIC_FIELD_INDEX]
			tableName = tokens[SONIC_TABLE_INDEX]
		}
		dbSpecField := tableName + "/" + fieldName
		_, ok := xDbSpecMap[dbSpecField]
		if ok {
			yangType := xDbSpecMap[dbSpecField].yangType
			// terminal node case
			if !(yangType == YANG_LEAF_LIST) {
				return leafListInstVal, err
			}
		}
	}

	//Check if URI has Leaf-list value
	if (strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/")) {
		xpathList := strings.Split(xpath, "/")
		ll_name := xpathList[len(xpathList)-1]
		ll_inx := strings.LastIndex(uri, ll_name)
		if ll_inx != -1 {
			ll_value := uri[ll_inx:]
			ll_value = strings.TrimSuffix(ll_value, "]")
			valueLst := strings.SplitN(ll_value, "=", 2)
			leafListInstVal = valueLst[1]

			if (strings.Contains(leafListInstVal, ":")) && (strings.HasPrefix(leafListInstVal, OC_MDL_PFX) || strings.HasPrefix(leafListInstVal, IETF_MDL_PFX) || strings.HasPrefix(leafListInstVal, IANA_MDL_PFX)) {
				// identity-ref/enum has module prefix
				leafListInstVal = strings.SplitN(leafListInstVal, ":", 2)[1]
				xfmrLogDebug("Leaf-list instance value after removing identityref prefix - %v", leafListInstVal)
			}
			xfmrLogDebug("Leaf-list instance value to be returned - %v", leafListInstVal)

			return leafListInstVal, nil
		}
	}
	return leafListInstVal, err
}

func formSonicXfmrInputRequest(dbNum db.DBNum, table string, key string, xpath string) SonicXfmrParams {
	var inParams SonicXfmrParams
	inParams.dbNum = dbNum
	inParams.tableName = table
	inParams.key = key
	inParams.xpath = table
	return inParams
}

func getYangNodeTypeFromUri(uri string) (yangElementType, error) {
	// function to get YANG node type(leaf/leaf-lit/list/container) from uri
	var yangNodeType yangElementType
	var err error
	var specInfoOk bool
	var dbSpecInfo *dbInfo
	var xYangSpecInfo *yangXpathInfo

	specInfo, spec_err := getXfmrSpecInfoFromUri(uri)
	if spec_err != nil {
		return yangNodeType, spec_err
	}

	if isSonicYang(uri) {
		dbSpecInfo, specInfoOk = specInfo.(*dbInfo)
		if specInfoOk {
			yangNodeType = dbSpecInfo.yangType
		}
	} else {
		xYangSpecInfo, specInfoOk = specInfo.(*yangXpathInfo)
		if specInfoOk {
			yangNodeType = xYangSpecInfo.yangType
		}
	}
	if !specInfoOk {
		err = fmt.Errorf("no YANG meta-data for translation for URI - %v", uri)
		xfmrLogInfo("%v", err)
		return yangNodeType, err
	}
	xfmrLogDebug("For URI %v , yangNodeType is %v", uri, yangNodeType)
	return yangNodeType, nil
}

func getXfmrSpecInfoFromUri(uri string) (interface{}, error) {
	// function to extract xfmr spec info for a given sonic uri/path
	var err error
	var specInfo interface{}
	var xpathInSpecMapOk bool

	xpath, _, xpathErr := XfmrRemoveXPATHPredicates(uri)
	if xpathErr != nil {
		xfmrLogInfo("For URI - %v, couldn't convert to xpath - %v", uri, xpathErr)
		return specInfo, xpathErr
	}
	xfmrLogDebug("For URI %v , xpath is - %v", uri, xpath)
	if isSonicYang(uri) {
		tokens := strings.Split(xpath, "/")
		fieldName := ""
		tableName := ""
		tblChldName := ""
		dbSpecXpath := ""
		if len(tokens) > SONIC_FIELD_INDEX {
			fieldName = tokens[SONIC_FIELD_INDEX]
			tableName = tokens[SONIC_TABLE_INDEX]
			dbSpecXpath = tableName + "/" + fieldName
			specInfo, xpathInSpecMapOk = xDbSpecMap[dbSpecXpath]
		} else if len(tokens) > SONIC_TBL_CHILD_INDEX {
			tableName = tokens[SONIC_TABLE_INDEX]
			tblChldName = tokens[SONIC_TBL_CHILD_INDEX]
			dbSpecXpath = tableName + "/" + tblChldName
			specInfo, xpathInSpecMapOk = xDbSpecMap[dbSpecXpath]
		} else if len(tokens) > SONIC_TABLE_INDEX {
			tableName = tokens[SONIC_TABLE_INDEX]
			dbSpecXpath = tableName
			specInfo, xpathInSpecMapOk = xDbSpecMap[dbSpecXpath]
		} else {
			//top-most level container
			topContainer := tokens[SONIC_TOPCONTR_INDEX]
			dbSpecXpath = "/" + topContainer
			specInfo, xpathInSpecMapOk = xDbSpecMap[dbSpecXpath]
		}
	} else {
		specInfo, xpathInSpecMapOk = xYangSpecMap[xpath]
	}
	if !xpathInSpecMapOk {
		msg := fmt.Sprintf("yang spec map doesn't contain xpath - %v", xpath)
		xfmrLogInfo(msg)
		err = fmt.Errorf("%v", msg)
	}
	if specInfo == nil {
		msg := fmt.Sprintf("spec map info couldn't be filled for  - %v", xpath)
		xfmrLogInfo(msg)
		err = fmt.Errorf("%v", msg)
	}

	return specInfo, err
}

func escapeKeyVal(val string) string {
	val = strings.Replace(val, "]", "\\]", -1)
	val = strings.Replace(val, "/", "\\/", -1)
	return val
}

// escapeKeyValForSplitPathAndNewPathInfo function escapes a path key's value as per SplitPathApi() Api
// which treats unescaped ] as the key. The ] character in key value should be escaped.
// '\' stored in redis DB Key is unescaped. Escape '\' in redisKey so the NewPathInfo can extract the key correctly
// Used in GET code flow
func escapeKeyValForSplitPathAndNewPathInfo(val string) string {
	if strings.Contains(val, "\\") {
		val = strings.Replace(val, "\\", "\\\\", -1)
	}
	val = strings.Replace(val, "]", "\\]", -1)
	return val
}

func getYangTypeStrId(yangTypeInt yangElementType) string {
	var yangTypeStr string
	switch yangTypeInt {
	case YANG_MODULE:
		yangTypeStr = "module"
	case YANG_LIST:
		yangTypeStr = "list"
	case YANG_CONTAINER:
		yangTypeStr = "container"
	case YANG_LEAF:
		yangTypeStr = "leaf"
	case YANG_LEAF_LIST:
		yangTypeStr = "leaf-list"
	case YANG_CHOICE:
		yangTypeStr = "choice"
	case YANG_CASE:
		yangTypeStr = "case"
	case YANG_RPC:
		yangTypeStr = "rpc"
	case YANG_NOTIF:
		yangTypeStr = "notification"
	default:
		yangTypeStr = "other"
	}
	return yangTypeStr
}

func getYangTypeIntId(e *yang.Entry) yangElementType {
	if e.IsCase() {
		return YANG_CASE
	} else if e.IsChoice() {
		return YANG_CHOICE
	} else if e.IsContainer() {
		return YANG_CONTAINER
	} else if e.IsLeaf() {
		return YANG_LEAF
	} else if e.IsLeafList() {
		return YANG_LEAF_LIST
	} else if e.IsList() {
		return YANG_LIST
	} else if e.Kind == yang.NotificationEntry {
		return YANG_NOTIF
	} else {
		return YANG_RPC
	}
}

func getYangEntryForXPath(xpath string) *yang.Entry {
	var entry *yang.Entry

	if len(xpath) > 0 {
		if yNode, ok := xDbSpecMap[xpath]; ok {
			if (yNode.yangType == YANG_LEAF) || (yNode.yangType == YANG_LEAF_LIST) {
				dbPathList := strings.Split(xpath, "/")
				table := dbPathList[0]
				field := dbPathList[1]
				yNd, tok := xDbSpecMap[table]
				if tok && yNd.yangType == YANG_CONTAINER && yNd.dbEntry != nil {
					for _, chldNode := range yNd.dbEntry.Dir {
						if chldNode == nil {
							continue
						}
						if chEntry, cok := chldNode.Dir[field]; cok {
							entry = chEntry
							break
						}
					}
					if entry == nil {
						for _, tblChldNode := range yNd.dbEntry.Dir {
							if tblChldNode == nil {
								continue
							}
							for _, chNode := range tblChldNode.Dir {
								if chNode == nil {
									continue
								}
								if chNode.Kind == yang.ChoiceEntry {
									for _, caNode := range chNode.Dir {
										if caNode == nil {
											continue
										}
										if flEntry, fok := caNode.Dir[field]; fok {
											entry = flEntry
											goto done
										}
									}
								}
							}
						}
					}
				}
			} else {
				entry = yNode.dbEntry
				goto done
			}
		} else if yNode, ok := xYangSpecMap[xpath]; ok {
			if (yNode.yangType == YANG_LEAF) || (yNode.yangType == YANG_LEAF_LIST) {
				pathList := strings.Split(xpath, "/")
				pxpath := strings.Join(pathList[:len(pathList)-1], "/")
				if yNd, pok := xYangSpecMap[pxpath]; pok {
					parentEntry := yNd.yangEntry
					childName := pathList[len(pathList)-1]
					if parentEntry != nil && childName != "" {
						if _, ok := parentEntry.Dir[childName]; ok {
							entry = parentEntry.Dir[childName]
							goto done
						}
						if entry == nil {
							for _, chNode := range yNd.yangEntry.Dir {
								if chNode == nil {
									continue
								}
								if chNode.Kind == yang.ChoiceEntry {
									for _, caNode := range chNode.Dir {
										if caNode == nil {
											continue
										}
										if flEntry, fok := caNode.Dir[childName]; fok {
											entry = flEntry
											goto done
										}
									}
								}
							}
						}
					}
				}
			} else {
				entry = yNode.yangEntry
				goto done
			}

		}
	}
done:
	return entry
}

func (inPm XfmrParams) String() string {
	return fmt.Sprintf("{oper: %v, uri: %v, requestUri: %v, "+
		"DB Name at current node: %v, table: %v, key: %v, dbDataMap: %v, subOpDataMap: %v, yangDefValMap: %v "+
		"skipOrdTblChk: %v, isVirtualTbl: %v, pCascadeDelTbl: %v, "+
		"invokeCRUSubtreeOnce: %v}",
		inPm.oper, inPm.uri, inPm.requestUri,
		inPm.curDb.Name(), inPm.table, inPm.key, inPm.dbDataMap, inPm.subOpDataMap, inPm.yangDefValMap,
		boolPtrToString(inPm.skipOrdTblChk), boolPtrToString(inPm.isVirtualTbl), inPm.pCascadeDelTbl,
		boolPtrToString(inPm.invokeCRUSubtreeOnce))
}

func boolPtrToString(boolPtr *bool) string {
	if boolPtr == nil {
		return "<nil>"
	} else {
		return fmt.Sprintf("%v", *boolPtr)
	}
}

func (inPm XfmrDbParams) String() string {
	return fmt.Sprintf("{oper: %v, dbNum: %v, tableName: %v, key: %v, fieldName: %v, value: %v", inPm.oper, inPm.dbNum, inPm.tableName,
		inPm.key, inPm.fieldName, inPm.value)
}

func (inPm SonicXfmrParams) String() string {
	return fmt.Sprintf("{dbNum: %v, tableName: %v, key: %v, xpath: %v", inPm.dbNum, inPm.tableName, inPm.key, inPm.xpath)
}

func (oper Operation) String() string {
	ret := "Unknown"
	switch oper {
	case GET:
		ret = "GET"
	case CREATE:
		ret = "CREATE"
	case REPLACE:
		ret = "REPLACE"
	case UPDATE:
		ret = "UPDATE"
	case DELETE:
		ret = "DELETE"
	case SUBSCRIBE:
		ret = "SUBSCRIBE"
	}
	return ret
}

func SonicUriHasSingletonContainer(uri string) bool {
	hasSingletonContainer := false
	if !strings.HasPrefix(uri, "/sonic") {
		return hasSingletonContainer
	}

	xpath, _, err := XfmrRemoveXPATHPredicates(uri)
        if err != nil || len(xpath) == 0 {
                return hasSingletonContainer
        }

        pathList := strings.Split(xpath, "/")

	if len(pathList) > SONIC_TBL_CHILD_INDEX {
		tblChldXpath := pathList[SONIC_TABLE_INDEX] + "/" + pathList[SONIC_TBL_CHILD_INDEX]
		if specTblChldInfo, ok := xDbSpecMap[tblChldXpath]; ok {
			if specTblChldInfo.yangType == YANG_CONTAINER {
				hasSingletonContainer = true
			}
		}
	}
	return hasSingletonContainer
}
