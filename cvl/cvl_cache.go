////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package cvl

import (
	"encoding/json"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	"github.com/antchfx/jsonquery"
	"github.com/antchfx/xmlquery"
	"runtime"
	"time"
)

func (c *CVL) addTableEntryToCache(tableName string, redisKey string) {
	if tableName == "" || redisKey == "" {
		return
	}

	if c.tmpDbCache[tableName] == nil {
		c.tmpDbCache[tableName] = map[string]interface{}{redisKey: nil}
	} else {
		tblMap := c.tmpDbCache[tableName]
		tblMap.(map[string]interface{})[redisKey] = nil
		c.tmpDbCache[tableName] = tblMap
	}
}

//Add the data which are referring this key
/*func (c *CVL) updateDeleteDataToCache(tableName string, redisKey string) {
	if _, existing := c.tmpDbCache[tableName]; !existing {
		return
	} else {
		tblMap := c.tmpDbCache[tableName]
		if _, existing := tblMap.(map[string]interface{})[redisKey]; existing {
			delete(tblMap.(map[string]interface{}), redisKey)
			c.tmpDbCache[tableName] = tblMap
		}
	}
}*/

// Fetch dependent data from validated data cache,
// Returns the data and flag to indicate that if requested data
// is found in update request, the data should be merged with Redis data
func (c *CVL) fetchDataFromRequestCache(tableName string, key string) (d map[string]string, m bool) {
	defer func() {
		pd := &d
		pm := &m

		TRACE_LOG(TRACE_CACHE,
			"Returning data from request cache, data = %v, merge needed = %v",
			*pd, *pm)
	}()

	cfgDataArr := c.requestCache[tableName][key]
	if len(cfgDataArr) != 0 {
		cfgReqData := cfgDataArr[len(cfgDataArr)-1]
		if cfgReqData.ReqData.VOp == cmn.OP_CREATE {
			return cfgReqData.ReqData.Data, false
		} else if cfgReqData.ReqData.VOp == cmn.OP_UPDATE {
			return cfgReqData.ReqData.Data, true
		} else if cfgReqData.ReqData.ReplaceOp && cfgReqData.ReqData.VType == cmn.VALIDATE_ALL && len(cfgDataArr) > 1 {
			prev := cfgDataArr[len(cfgDataArr)-2].ReqData
			if prev.ReplaceOp && prev.VOp == cmn.OP_UPDATE {
				return prev.Data, true // pick the UPDATE entry from replace op pair
			}
		}

		return map[string]string{}, true // caller will reconcile
	}

	return nil, false
}

// Fetch given table entries using pipeline
func (c *CVL) fetchTableDataToTmpCache(tableName string, dbKeys map[string]interface{}) int {

	if IsTraceAllowed(TRACE_CACHE) {
		TRACE_LOG(TRACE_CACHE, "\n%v, Entered fetchTableDataToTmpCache", time.Now())
	}

	totalCount := len(dbKeys)
	if totalCount == 0 {
		//No entry to be fetched
		return 0
	}

	entryFetched := 0
	bulkCount := 0
	bulkKeys := []string{}
	for dbKey, val := range dbKeys { //for all keys

		if val != nil { //skip entry already fetched
			mapTable := c.tmpDbCache[tableName]
			delete(mapTable.(map[string]interface{}), dbKey) //delete entry already fetched
			totalCount = totalCount - 1
			if bulkCount != totalCount {
				//If some entries are remaining go back to 'for' loop
				continue
			}
		} else {
			//Accumulate entries to be fetched
			bulkKeys = append(bulkKeys, dbKey)
			bulkCount = bulkCount + 1
		}

		if (bulkCount != totalCount) && ((bulkCount % MAX_BULK_ENTRIES_IN_PIPELINE) != 0) {
			//If some entries are remaining and bulk bucket is not filled,
			//go back to 'for' loop
			continue
		}

		mCmd := map[string]cmn.StrMapResult{}

		pipe := c.dbAccess.Pipeline()

		for _, dbKey := range bulkKeys {
			yangListName := getRedisTblToYangList(tableName, dbKey)
			redisKey := tableName + modelInfo.tableInfo[yangListName].redisKeyDelim + dbKey
			//Check in validated cache first and add as dependent data
			if entry, mergeNeeded := c.fetchDataFromRequestCache(tableName, dbKey); entry != nil {
				entryFetched = entryFetched + 1
				//Entry found in validated cache, so skip fetching from Redis
				//if merging is not required with Redis DB
				if !mergeNeeded {
					fieldMap := c.checkFieldMap(&entry)
					c.tmpDbCache[tableName].(map[string]interface{})[dbKey] = fieldMap
					continue
				}
				c.tmpDbCache[tableName].(map[string]interface{})[dbKey] = entry
			}

			//Otherwise fetch it from Redis
			mCmd[dbKey] = pipe.HGetAll(redisKey) //write into pipeline
			if mCmd[dbKey] == nil {
				CVL_LOG(WARNING, "Failed pipe.HGetAll('%s')", redisKey)
			}
		}

		err := pipe.Exec()
		defer pipe.Close()
		if err != nil {
			CVL_LOG(WARNING, "Failed to fetch details for table %s", tableName)
			return 0
		}
		bulkKeys = nil

		mapTable := c.tmpDbCache[tableName]

		for key, val := range mCmd {
			res, err := val.Result()

			if mapTable == nil {
				break
			}

			// apply configEdits from requestCache on Db Data
			c.evaluateDbTblNetChange(tableName, key, &res)

			if err != nil || len(res) == 0 {
				//no data found, don't keep blank entry
				delete(mapTable.(map[string]interface{}), key)
				continue
			}
			//exclude table name and delim
			keyOnly := key

			if (len(mapTable.(map[string]interface{})) > 0) && (mapTable.(map[string]interface{})[keyOnly] != nil) {
				tmpFieldMap := (mapTable.(map[string]interface{})[keyOnly]).(map[string]string)
				//merge with validated cache data
				mergeMap(res, tmpFieldMap)
				fieldMap := c.checkFieldMap(&res)
				mapTable.(map[string]interface{})[keyOnly] = fieldMap
			} else {
				fieldMap := c.checkFieldMap(&res)
				mapTable.(map[string]interface{})[keyOnly] = fieldMap
			}

			entryFetched = entryFetched + 1
		}

		runtime.Gosched()
	}

	if IsTraceAllowed(TRACE_CACHE) {
		TRACE_LOG(TRACE_CACHE, "\n%v, Exiting fetchTableDataToTmpCache", time.Now())
	}

	return entryFetched
}

// populate redis data to cache
func (c *CVL) fetchDataToTmpCache() *yparser.YParserNode {
	if IsTraceAllowed(TRACE_CACHE) {
		TRACE_LOG(TRACE_CACHE, "\n%v, Entered fetchToTmpCache", time.Now())
	}

	entryToFetch := 0
	var root *yparser.YParserNode = nil
	var errObj yparser.YParserError

	for entryToFetch = 1; entryToFetch > 0; { //Force to enter the loop for first time
		//Repeat until all entries are fetched
		entryToFetch = 0
		for tableName, dbKeys := range c.tmpDbCache { //for each table
			entryToFetch = entryToFetch + c.fetchTableDataToTmpCache(tableName, dbKeys.(map[string]interface{}))
		} //for each table

		//If no table entry delete the table  itself
		for tableName, dbKeys := range c.tmpDbCache { //for each table
			if len(dbKeys.(map[string]interface{})) == 0 {
				delete(c.tmpDbCache, tableName)
				continue
			}
		}

		if entryToFetch == 0 {
			//No more entry to fetch
			break
		}

		if IsTraceAllowed(TRACE_CACHE) {
			jsonDataBytes, _ := json.Marshal(c.tmpDbCache)
			jsonData := string(jsonDataBytes)
			TRACE_LOG(TRACE_CACHE, "Top Node=%v\n", jsonData)
		}

		data, err := jsonquery.ParseJsonMap(&c.tmpDbCache)

		if err != nil {
			return nil
		}

		//Build yang tree for each table and cache it
		for jsonNode := data.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {
			TRACE_LOG(TRACE_CACHE, "Top Node=%v\n", jsonNode.Data)
			//Visit each top level list in a loop for creating table data
			topNode, _ := c.generateTableData(true, jsonNode)
			if root == nil {
				root = topNode
			} else {
				if root, errObj = c.yp.MergeSubtree(root, topNode); errObj.ErrCode != yparser.YP_SUCCESS {
					return nil
				}
			}

			//Generate YANG data for Yang Validator
			topYangNode, cvlYErrObj := c.generateYangListData(jsonNode, true)
			if topYangNode == nil {
				cvlYErrObj.ErrCode = CVL_SYNTAX_ERROR
				CVL_LOG(WARNING, "Unable to translate cache data to YANG format")
				return nil
			}

			//Create a full document and merge with main YANG data
			doc := &xmlquery.Node{Type: xmlquery.DocumentNode}
			doc.FirstChild = topYangNode
			doc.LastChild = topYangNode
			topYangNode.Parent = doc

			if IsTraceAllowed(TRACE_CACHE) {
				TRACE_LOG(TRACE_CACHE, "Before cache merge = %s, source = %s",
					c.yv.root.OutputXML(false),
					doc.OutputXML(false))
			}

			if c.mergeYangData(c.yv.root, doc) != CVL_SUCCESS {
				CVL_LOG(WARNING, "Unable to merge translated YANG data while "+
					"translating from cache data to YANG format")
				cvlYErrObj.ErrCode = CVL_SYNTAX_ERROR
				return nil
			}
			if IsTraceAllowed(TRACE_CACHE) {
				TRACE_LOG(TRACE_CACHE, "After cache merge = %s",
					c.yv.root.OutputXML(false))
			}
		}
	} // until all dependent data is fetched

	if IsTraceAllowed(TRACE_CACHE) {
		if root != nil {
			dumpStr := c.yp.NodeDump(root)
			TRACE_LOG(TRACE_CACHE, "Dependent Data = %v\n", dumpStr)
		}
		TRACE_LOG(TRACE_CACHE, "\n%v, Exiting fetchToTmpCache", time.Now())
	}
	return root
}

func (c *CVL) clearTmpDbCache() {
	for key := range c.tmpDbCache {
		delete(c.tmpDbCache, key)
	}
}

// This function applies configEdits already validated on top of
// DB data as these are still not committed to DB
func (c *CVL) evaluateDbTblNetChange(tableName, key string, dbData *map[string]string) {
	cfgDataArr := c.requestCache[tableName][key]
	for i := 0; i < len(cfgDataArr); i++ {
		if cfgDataArr[i].ReqData.VType == cmn.VALIDATE_NONE {
			if len(cfgDataArr[i].ReqData.Data) == 0 {
				if cfgDataArr[i].ReqData.VOp == cmn.OP_DELETE {
					newMap := make(map[string]string)
					dbData = &newMap
				}
			} else {
				for k, v := range cfgDataArr[i].ReqData.Data {
					if cfgDataArr[i].ReqData.VOp == cmn.OP_DELETE {
						delete(*dbData, k)
					} else {
						(*dbData)[k] = v
					}
				}
			}
		}
	}
}
