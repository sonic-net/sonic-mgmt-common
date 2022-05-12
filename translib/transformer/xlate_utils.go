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
    "errors"
    "strings"
    "regexp"
    "runtime"
    "github.com/Azure/sonic-mgmt-common/translib/db"
    "github.com/Azure/sonic-mgmt-common/translib/tlerr"
    "github.com/openconfig/goyang/pkg/yang"
    "github.com/openconfig/gnmi/proto/gnmi"
    "github.com/openconfig/ygot/ygot"
    "github.com/openconfig/ygot/ytypes"
    "github.com/Azure/sonic-mgmt-common/translib/ocbinds"
    log "github.com/golang/glog"
    "sync"
)

func initRegex() {
	rgpKeyExtract = regexp.MustCompile(`\[([^\[\]]*)\]`)
	rgpIpv6 = regexp.MustCompile(`(([^:]+:){6}(([^:]+:[^:]+)|(.*\..*)))|((([^:]+:)*[^:]+)?::(([^:]+:)*[^:]+)?)(%.+)?`)
	rgpMac = regexp.MustCompile(`([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	rgpIsMac = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	rgpSncKeyExtract = regexp.MustCompile(`\[([^\[\]]*)\]`)

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
				xfmrLogInfoAll("key concatenater(\"%v\") found for xpath %v ", delim, xpath)
			}

			if len(keyPrefix) > 0 { keyPrefix += delim }
			keyVal := ""
			for i, k := range (strings.Split(yangEntry.Key, " ")) {
				if i > 0 { keyVal = keyVal + delim }
				fieldXpath :=  xpath + "/" + k
				fVal, err := unmarshalJsonToDbData(yangEntry.Dir[k], fieldXpath, k, data.(map[string]interface{})[k])
				if err != nil {
					log.Warningf("Couldn't unmarshal Json to DbData: path(\"%v\") error (\"%v\").", fieldXpath, err)
				}

				if ((strings.Contains(fVal, ":")) &&
				    (strings.HasPrefix(fVal, OC_MDL_PFX) || strings.HasPrefix(fVal, IETF_MDL_PFX) || strings.HasPrefix(fVal, IANA_MDL_PFX))) {
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
func mapMerge(destnMap map[string]map[string]db.Value, srcMap map[string]map[string]db.Value, oper int) {
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
                if (oper == DELETE) {
                    if ((len(destnMap[table][rule].Field) == 0) && (len(ruleData.Field) > 0)) {
                        continue;
                    }
                    if ((len(destnMap[table][rule].Field) > 0) && (len(ruleData.Field) == 0)) {
                        destnMap[table][rule] = db.Value{Field: make(map[string]string)};
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

func yangTypeGet(entry *yang.Entry) string {
    if entry != nil && entry.Node != nil {
        return entry.Node.Statement().Keyword
    }
    return ""
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
	id          := xYangSpecMap[xpath].keyLevel
	keyDataList := strings.SplitN(dbKey, dbKeySep, id)
	uriWithKey  := fmt.Sprintf("%v", xpath)
	uriWithKeyCreate := true
	if len(keyDataList) == 0 {
		keyDataList = append(keyDataList, dbKey)
	}

	/* if uri contins key, use it else use xpath */
	if strings.Contains(uri, "[") {
		if strings.HasSuffix(uri, "]") || strings.HasSuffix(uri, "]/") {
			uriXpath, _, _ := XfmrRemoveXPATHPredicates(uri)
			if uriXpath == xpath {
				uriWithKeyCreate = false
			}
		}
		uriWithKey  = fmt.Sprintf("%v", uri)
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
		log.Warningf("No key transformer found for multi element yang key mapping to a single redis key string, for uri %v", uri)
                errStr := fmt.Sprintf("Error processing key for list %v", uri)
                err = fmt.Errorf("%v", errStr)
                return rmap, uriWithKey, err
	}
	keyXpath := xpath + "/" + keyNameList[0]
	xyangSpecInfo, ok := xYangSpecMap[keyXpath]
	if !ok  || xyangSpecInfo == nil {
		errStr := fmt.Sprintf("Failed to find key xpath %v in xYangSpecMap or is nil, needed to fetch the yangEntry data-type", keyXpath)
		err = fmt.Errorf("%v", errStr)
		return rmap, uriWithKey, err
	}
	yngTerminalNdDtType := xyangSpecInfo.yangEntry.Type.Kind
	resVal, _, err := DbToYangType(yngTerminalNdDtType, keyXpath, keyDataList[0])
	if err != nil {
		err = fmt.Errorf("Failure in converting Db value type to yang type for field %v",      keyXpath)
		return rmap, uriWithKey, err
	} else {
		 rmap[keyNameList[0]] = resVal
	}
	if uriWithKeyCreate {
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
        log.Warningf("Error in uri to path conversion: %v", err)
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
    keyList = append(keyList, strings.Split(entry.Key, " ")...)
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
        xfmrLogInfoAll("getYangTerminalNodeTypeName keyXpath: %v ", keyXpath)
        dbInfo, ok := xDbSpecMap[keyXpath]
        if ok {
                yngTerminalNdTyName := dbInfo.dbEntry.Type.Name
                xfmrLogInfoAll("yngTerminalNdTyName: %v", yngTerminalNdTyName)
                return yngTerminalNdTyName
        }
        return ""
}


func sonicKeyDataAdd(dbIndex db.DBNum, keyNameList []string, xpathPrefix string, keyStr string, resultMap map[string]interface{}) {
        var dbOpts db.Options
        var keyValList []string
        xfmrLogInfoAll("sonicKeyDataAdd keyNameList:%v, keyStr:%v", keyNameList, keyStr)

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
			xfmrLogInfoAll("Single key non ipv6/mac address for : separator")
                }
        } else if (strings.Count(keyStr, keySeparator) == len(keyNameList)-1) {
                /* number of keys will match number of key values */
                keyValList = strings.SplitN(keyStr, keySeparator, len(keyNameList))
        } else {
                /* number of dbKey values more than number of keys */
                if keySeparator == ":" && (hasIpv6AddString(keyStr)|| hasMacAddString(keyStr)) {
			xfmrLogInfoAll("Key Str has ipv6/mac address with : separator")
                        if len(keyNameList) == 2 {
                                valList := strings.SplitN(keyStr, keySeparator, -1)
                                /* IPV6 address is first entry */
                                yngTerminalNdTyName := getYangTerminalNodeTypeName(xpathPrefix, keyNameList[0])
                                if yngTerminalNdTyName == "ip-address" || yngTerminalNdTyName == "ip-prefix" || yngTerminalNdTyName == "ipv6-prefix" || yngTerminalNdTyName == "ipv6-address" || yngTerminalNdTyName == "mac-address"{
                                        keyValList = append(keyValList, strings.Join(valList[:len(valList)-2], keySeparator))
                                        keyValList = append(keyValList, valList[len(valList)-1])
                                } else {
                                        yngTerminalNdTyName := getYangTerminalNodeTypeName(xpathPrefix, keyNameList[1])
                                        if yngTerminalNdTyName == "ip-address" || yngTerminalNdTyName == "ip-prefix" || yngTerminalNdTyName == "ipv6-prefix" || yngTerminalNdTyName == "ipv6-address" || yngTerminalNdTyName == "mac-address" {
                                                keyValList = append(keyValList, valList[0])
                                                keyValList = append(keyValList, strings.Join(valList[1:], keySeparator))
                                        } else {
                                                xfmrLogInfoAll("No ipv6 or mac address found in value. Cannot split value ")
                                        }
                                }
				xfmrLogInfoAll("KeyValList has %v", keyValList)
                        } else {
                                xfmrLogInfoAll("Number of keys : %v", len(keyNameList))
                        }
                } else {
                        keyValList = strings.SplitN(keyStr, keySeparator, -1)
			xfmrLogInfoAll("Split all keys KeyValList has %v", keyValList)
                }
        }
        xfmrLogInfoAll("yang keys list - %v, xpathprefix - %v, DB-key string - %v, DB-key list after db key separator split - %v, dbIndex - %v", keyNameList, xpathPrefix, keyStr, keyValList, dbIndex)

        if len(keyNameList) != len(keyValList) {
                return
        }

    for i, keyName := range keyNameList {
            keyXpath := xpathPrefix + "/" + keyName
            dbInfo, ok := xDbSpecMap[keyXpath]
            var resVal interface{}
            resVal = keyValList[i]
            if !ok || dbInfo == nil {
                    log.Warningf("xDbSpecMap entry not found or is nil for xpath %v, hence data-type conversion cannot happen", keyXpath)
            } else {
                    yngTerminalNdDtType := dbInfo.dbEntry.Type.Kind
                    var err error
                    resVal, _, err = DbToYangType(yngTerminalNdDtType, keyXpath, keyValList[i])
                    if err != nil {
                            log.Warningf("Data-type conversion unsuccessfull for xpath %v", keyXpath)
                            resVal = keyValList[i]
                    }
            }

        resultMap[keyName] = resVal
    }
}

func yangToDbXfmrFunc(funcName string) string {
    return ("YangToDb_" + funcName)
}

func uriWithKeyCreate (uri string, xpathTmplt string, data interface{}) (string, error) {
    var err error
    if _, ok := xYangSpecMap[xpathTmplt]; ok {
         yangEntry := xYangSpecMap[xpathTmplt].yangEntry
         if yangEntry != nil {
              for _, k := range (strings.Split(yangEntry.Key, " ")) {
		      keyXpath := xpathTmplt + "/" + k
		      if _, keyXpathEntryOk := xYangSpecMap[keyXpath]; !keyXpathEntryOk {
			      log.Warningf("No entry found in xYangSpec map for xapth %v", keyXpath)
                              err = fmt.Errorf("No entry found in xYangSpec map for xapth %v", keyXpath)
                              break
		      }
		      keyYangEntry := xYangSpecMap[keyXpath].yangEntry
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
		      if ((strings.Contains(keyVal, ":")) && (strings.HasPrefix(keyVal, OC_MDL_PFX) || strings.HasPrefix(keyVal, IETF_MDL_PFX) || strings.HasPrefix(keyVal, IANA_MDL_PFX))) {
			      // identity-ref/enum has module prefix
			      keyVal = strings.SplitN(keyVal, ":", 2)[1]
		      }
                      uri += fmt.Sprintf("[%v=%v]", k, keyVal)
              }
	 } else {
            err = fmt.Errorf("Yang Entry not available for xpath %v", xpathTmplt)
	 }
    } else {
        err = fmt.Errorf("No entry in xYangSpecMap for xpath %v", xpathTmplt)
    }
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
		log.Warning("Empty uri string supplied")
                err = fmt.Errorf("Empty uri string supplied")
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
		err = fmt.Errorf("No module name found in uri %s", uri)
        }
	xfmrLogInfo("module name = %v", result)
	return result, err
}

func formXfmrInputRequest(d *db.DB, dbs [db.MaxDB]*db.DB, cdb db.DBNum, ygRoot *ygot.GoStruct, uri string, requestUri string, oper int, key string, dbDataMap *RedisDbMap, subOpDataMap map[int]*RedisDbMap, param interface{}, txCache interface{}) XfmrParams {
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
        case db.ApplDB, db.CountersDB:
                opt = getDBOptionsWithSeparator(dbNo, "", ":", ":")
        case db.FlexCounterDB, db.AsicDB, db.LogLevelDB, db.ConfigDB, db.StateDB, db.ErrorDB, db.UserDB, db.EventDB:
                opt = getDBOptionsWithSeparator(dbNo, "", "|", "|")
        }

        return opt
}

func getDBOptionsWithSeparator(dbNo db.DBNum, initIndicator string, tableSeparator string, keySeparator string) db.Options {
        return(db.Options {
                    DBNo              : dbNo,
                    InitIndicator     : initIndicator,
                    TableNameSeparator: tableSeparator,
                    KeySeparator      : keySeparator,
                      })
}

func getXpathFromYangEntry(entry *yang.Entry) string {
        xpath := ""
        if entry != nil {
                xpath = entry.Name
                entry = entry.Parent
                for {
                        if entry.Parent != nil {
                                xpath = entry.Name + "/" + xpath
                                entry = entry.Parent
                        } else {
                                // This is the module entry case
                                xpath = "/" + entry.Name + ":" + xpath
                                break
                        }
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
                        pvar = strings.Split(pvar,":")[1]
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

func replacePrefixWithModuleName(xpath string) (string) {
	//Input xpath is after removing the xpath Predicates
	var moduleNm string
	if _, ok := xYangSpecMap[xpath]; ok {
		moduleNm = xYangSpecMap[xpath].dbEntry.Prefix.Parent.NName()
		pathList := strings.Split(xpath, ":")
		if len(moduleNm) > 0 && len(pathList) == 2 {
			xpath = "/" + moduleNm + ":" + pathList[1]
		}
	}
	return xpath
}


/* Extract key vars, create db key and xpath */
func xpathKeyExtract(d *db.DB, ygRoot *ygot.GoStruct, oper int, path string, requestUri string, dbDataMap *map[db.DBNum]map[string]map[string]db.Value, subOpDataMap map[int]*RedisDbMap, txCache interface{}, xfmrTblKeyCache map[string]tblKeyCache) (xpathTblKeyExtractRet, error) {
	 xfmrLogInfoAll("In uri(%v), reqUri(%v), oper(%v)", path, requestUri, oper)
	 var retData xpathTblKeyExtractRet
	 keyStr    := ""
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
	 requestUriYangType := yangTypeGet(xpathInfo.yangEntry)
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
	 xfmrLogInfoAll("path elements are : %v", pathList)
	 for i, k := range pathList {
		 curPathWithKey += "/" + k
		 callKeyXfmr := true
		 yangXpath += "/" + xpathList[i]
		 xpathInfo, ok := xYangSpecMap[yangXpath]
		 if ok {
			 yangType := yangTypeGet(xpathInfo.yangEntry)
			 /* when deleting a specific element from leaf-list query uri is of the form
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
							 retData.dbKey,retData.tableName,retData.xpath = "","",""
							 return retData, err
						 }
						 if ret != nil {
							 keyStr = ret[0].Interface().(string)
						 }
					 } else {
						 ret, err := keyXfmrHandler(inParams, xYangSpecMap[yangXpath].xfmrKey)
						 if err != nil {
							 retData.dbKey,retData.tableName,retData.xpath = "","",""
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
					 /* multi-leaf yang key together forms a single key-string in redis.
					 There should be key-transformer, if not then the yang key leaves
					 will be concatenated with respective default DB type key-delimiter
					 */
					 for idx, kname := range rgpKeyExtract.FindAllString(k, -1) {
						 if idx > 0 { keyStr += keySeparator }
						 keyl := strings.TrimRight(strings.TrimLeft(kname, "["), "]")
						 keys := strings.Split(keyl, "=")
						 keyStr += keys[1]
					 }
				 }
			 } else if len(xYangSpecMap[yangXpath].xfmrKey) > 0  {
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
						retData.dbKey,retData.tableName,retData.xpath = "","",""
						return retData, err
					 }
					 if ret != nil {
						 keyStr = ret[0].Interface().(string)
					 }
				 } else {
					 ret, err := keyXfmrHandler(inParams, xYangSpecMap[yangXpath].xfmrKey)
					 if ((yangType != YANG_LIST) && (err != nil)) {
						retData.dbKey,retData.tableName,retData.xpath = "","",""
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
	 tblPtr     := xpathInfo.tableName
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
			 return retData, err
		 }
	 }
	 if ((oper == SUBSCRIBE) && (strings.TrimSpace(keyStr) == "") && (requestUriYangType == YANG_LIST) && (!isUriForListInstance)) {
		 keyStr="*"
	 }
	 retData.dbKey = keyStr
	 xfmrLogInfoAll("Return uri(%v), xpath(%v), key(%v), tableName(%v), isVirtualTbl:%v", path, retData.xpath, keyStr, retData.tableName, retData.isVirtualTbl)
	return retData, err
}

func dbTableFromUriGet(d *db.DB, ygRoot *ygot.GoStruct, oper int, uri string, requestUri string, subOpDataMap map[int]*RedisDbMap, txCache interface{}, xfmrTblKeyCache map[string]tblKeyCache) (string, error) {
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
	 xfmrLogInfoAll("In uri(%v)", path)
	 xpath, keyStr, tableName, fldNm := "", "", "", ""
	 var err error
	 lpath := path
	 xpath, _, err = XfmrRemoveXPATHPredicates(path)
	 if err != nil {
		 return xpath, keyStr, tableName
	 }
	 if xpath != "" {
		 fldPth := strings.Split(xpath, "/")
		 if len(fldPth) > SONIC_FIELD_INDEX {
			 fldNm = fldPth[SONIC_FIELD_INDEX]
			 xfmrLogInfoAll("Field Name : %v", fldNm)
		 }
	 }
	 pathsubStr := strings.Split(path , "/")
	 if len(pathsubStr) > SONIC_TABLE_INDEX  {
		 if strings.Contains(pathsubStr[2], "[") {
			 tableName = strings.Split(pathsubStr[SONIC_TABLE_INDEX], "[")[0]
		 } else {
			 tableName = pathsubStr[SONIC_TABLE_INDEX]
		 }
		 dbInfo, ok := xDbSpecMap[tableName]
		 cdb := db.ConfigDB
		 if !ok {
			 xfmrLogInfoAll("No entry in xDbSpecMap for xpath %v in order to fetch DB index", tableName)
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
				 xfmrLogInfoAll("pathList after uri split %v", pathLst)
				 lpath = "/" + strings.Join(pathLst[:SONIC_FIELD_INDEX-1], "/")
				 xfmrLogInfoAll("path after removing the field portion %v", lpath)
			 }
			 for i, kname := range rgpSncKeyExtract.FindAllString(lpath, -1) {
				 if i > 0 {
					 keyStr += dbOpts.KeySeparator
				 }
				 val := strings.Split(kname, "=")[1]
				 keyStr += strings.TrimRight(val, "]")
			 }
		 }
	 }
	 xfmrLogInfoAll("Return uri(%v), xpath(%v), key(%v), tableName(%v)", path, xpath, keyStr, tableName)
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
                for sncMdl := range(xDbSpecTblSeqnMap) {
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
	_, AbsfileName, lineNum, _ := runtime.Caller(2)
	fileNmElems := strings.Split(AbsfileName, "/")
	fileNm := fileNmElems[len(fileNmElems)-1]
	fNmLnoStr := fmt.Sprintf("[%v:%v]", fileNm, lineNum)
	return fNmLnoStr
}

func xfmrLogInfo(format string, args ...interface{}) {
	fNmLnoStr := getFileNmLineNumStr()
	log.Infof(fNmLnoStr + format, args...)
}

func xfmrLogInfoAll(format string, args ...interface{}) {
	if log.V(5) {
		fNmLnoStr := getFileNmLineNumStr()
		log.Infof(fNmLnoStr + format, args...)
	}
}

func formXfmrDbInputRequest(oper int, d db.DBNum, tableName string, key string, field string, value string) XfmrDbParams {
	var inParams XfmrDbParams
	inParams.oper       = oper
	inParams.dbNum      = d
	inParams.tableName  = tableName
	inParams.key        = key
	inParams.fieldName  = field
	inParams.value      = value
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

func dbKeyValueXfmrHandler(oper int, dbNum db.DBNum, tblName string, dbKey string) (string, error) {
	var err error
	var keyValList []string

    xfmrLogInfoAll("dbKeyValueXfmrHandler: oper(%v), db(%v), tbl(%v), dbKey(%v)",
    oper, dbNum, tblName, dbKey)
	if specTblInfo, ok := xDbSpecMap[tblName]; ok {
		for _, lname := range specTblInfo.listName {
			listXpath := tblName + "/" + lname
			keyMap    := make(map[string]interface{})

			if specListInfo, ok := xDbSpecMap[listXpath]; ok && len(specListInfo.keyList) > 0 {
				sonicKeyDataAdd(dbNum, specListInfo.keyList, tblName, dbKey, keyMap)

				if len(keyMap) == len(specListInfo.keyList) {
					for _, kname := range specListInfo.keyList {
						keyXpath  := tblName + "/" + kname
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
    xfmrLogInfoAll("dbKeyValueXfmrHandler: tbl(%v), dbKey(%v), retKey(%v), keyValList(%v)",
    tblName, dbKey, retKey, keyValList)

	return retKey, nil
}

func dbDataXfmrHandler(resultMap map[int]map[db.DBNum]map[string]map[string]db.Value) error {
	xfmrLogInfoAll("Received  resultMap(%v)", resultMap)
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
	xfmrLogInfoAll("Transformed resultMap(%v)", resultMap)
	return nil
}

func formXlateFromDbParams(d *db.DB, dbs [db.MaxDB]*db.DB, cdb db.DBNum, ygRoot *ygot.GoStruct, uri string, requestUri string, xpath string, oper int, tbl string, tblKey string, dbDataMap *RedisDbMap, txCache interface{}, resultMap map[string]interface{}, validate bool) xlateFromDbParams {
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

func formXlateToDbParam(d *db.DB, ygRoot *ygot.GoStruct, oper int, uri string, requestUri string, xpathPrefix string, keyName string, jsonData interface{}, resultMap map[int]RedisDbMap, result map[string]map[string]db.Value, txCache interface{}, tblXpathMap map[string]map[string]map[string]bool, subOpDataMap map[int]*RedisDbMap, pCascadeDelTbl *[]string, xfmrErr *error, name string, value interface{}, tableName string) xlateToParams {
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

func splitUri(uri string) []string {
	pathList := SplitPath(uri)
	xfmrLogInfoAll("uri: %v ", uri)
	xfmrLogInfoAll("uri path elems: %v", pathList)
	return pathList
}

func dbTableExists(d *db.DB, tableName string, dbKey string, oper int) (bool, error) {
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
                        xfmrLogInfoAll("dbKeyValueXfmrHandler() returned db key %v", retKey)
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
			xfmrLogInfoAll("keys for table %v are %v", tableName, keys)
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
	xfmrLogInfoAll("received Db no - %v, table - %v, dbkey - %v", dbNo, table, dbKey)
	if _, exists := dbData[dbNo][table][dbKey]; exists {
		return true
	} else {
		return false
	}
}

func leafListInstExists(leafListInDbVal string, checkLeafListInstVal string) bool {
	/*function to check if leaf-list DB value contains the given instance*/
	exists := false
	xfmrLogInfoAll("received value of leaf-list in DB - %v,  Value to be checked if exists in leaf-list - %v", leafListInDbVal, checkLeafListInstVal)
	leafListItemLst := strings.Split(leafListInDbVal, ",")
	for idx := range(leafListItemLst) {
		if leafListItemLst[idx] == checkLeafListInstVal {
			exists = true
			xfmrLogInfoAll("Leaf-list instance exists")
			break
		}
	}
	return exists
}

func extractLeafListInstFromUri(uri string) (string, error) {
	/*function to extract leaf-list instance value coming as part of uri
	Handling [ ] in value*/
	xfmrLogInfoAll("received uri - %v", uri)
	var leafListInstVal string
	yangType := ""
	err := fmt.Errorf("Unable to extract leaf-list instance value for uri - %v", uri)

	xpath, _, xerr := XfmrRemoveXPATHPredicates(uri)
        if !isSonicYang(uri) {
                specInfo, ok := xYangSpecMap[xpath]
                if !ok {
                        return leafListInstVal, xerr
                }
                yangType = yangTypeGet(specInfo.yangEntry)
                if !(yangType == YANG_LEAF_LIST) {
                        return leafListInstVal, err
                }
        } else {
                tokens:= strings.Split(xpath, "/")
                fieldName := ""
                tableName := ""
                if len(tokens) > SONIC_FIELD_INDEX {
                        fieldName = tokens[SONIC_FIELD_INDEX]
                        tableName = tokens[SONIC_TABLE_INDEX]
                }
                dbSpecField := tableName + "/" + fieldName
                _, ok := xDbSpecMap[dbSpecField]
                if ok {
                        yangType := xDbSpecMap[dbSpecField].fieldType
                        // terminal node case
                        if !(yangType == YANG_LEAF_LIST) {
                                return leafListInstVal, err
                        }
                }
        }

	//Check if uri has Leaf-list value
	if ((strings.HasSuffix(uri, "]")) || (strings.HasSuffix(uri, "]/"))) {
		xpathList := strings.Split(xpath, "/")
		ll_name := xpathList[len(xpathList)-1]
		ll_inx := strings.LastIndex(uri,ll_name)
		if ll_inx != -1 {
			ll_value := uri[ll_inx:]
			ll_value = strings.TrimSuffix(ll_value, "]")
			valueLst := strings.SplitN(ll_value, "=", 2)
			leafListInstVal = valueLst[1]

			if ((strings.Contains(leafListInstVal, ":")) && (strings.HasPrefix(leafListInstVal, OC_MDL_PFX) || strings.HasPrefix(leafListInstVal, IETF_MDL_PFX) || strings.HasPrefix(leafListInstVal, IANA_MDL_PFX))) {
				// identity-ref/enum has module prefix
				leafListInstVal = strings.SplitN(leafListInstVal, ":", 2)[1]
				xfmrLogInfoAll("Leaf-list instance value after removing identityref prefix - %v", leafListInstVal)
			}
			xfmrLogInfoAll("Leaf-list instance value to be returned - %v", leafListInstVal)

			return leafListInstVal, nil
		}
	}
	return leafListInstVal, err
}

