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

package cvl
import (
	"encoding/json"
	"github.com/go-redis/redis"
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	"time"
	"runtime"
	"github.com/antchfx/jsonquery"
)

// Fetch dependent data from validated data cache,
// Returns the data and flag to indicate that if requested data 
// is found in update request, the data should be merged with Redis data
func (c *CVL) fetchDataFromRequestCache(tableName string, key string) (d map[string]string, m bool) {
	defer func() {
		pd := &d
		pm := &m

		TRACE_LOG(INFO_API, TRACE_CACHE,
		"Returning data from request cache, data = %v, merge needed = %v",
		*pd, *pm)
	}()

	cfgDataArr := c.requestCache[tableName][key]
	if (cfgDataArr != nil) {
		for _, cfgReqData := range cfgDataArr {
			//Delete request doesn't have depedent data
			if (cfgReqData.VOp == OP_CREATE) {
				return cfgReqData.Data, false
			} else	if (cfgReqData.VOp == OP_UPDATE) {
				return cfgReqData.Data, true
			}
		}
	}

	return nil, false
}

//Fetch given table entries using pipeline
func (c *CVL) fetchTableDataToTmpCache(tableName string, dbKeys map[string]interface{}) int {

	TRACE_LOG(INFO_API, TRACE_CACHE, "\n%v, Entered fetchTableDataToTmpCache", time.Now())

	totalCount := len(dbKeys)
	if (totalCount == 0) {
		//No entry to be fetched
		return 0
	}

	entryFetched := 0
	bulkCount := 0
	bulkKeys := []string{}
	for dbKey, val := range dbKeys { //for all keys

		 if (val != nil) { //skip entry already fetched
                        mapTable := c.tmpDbCache[tableName]
                        delete(mapTable.(map[string]interface{}), dbKey) //delete entry already fetched
                        totalCount = totalCount - 1
                        if(bulkCount != totalCount) {
                                //If some entries are remaining go back to 'for' loop
                                continue
                        }
                } else {
                        //Accumulate entries to be fetched
                        bulkKeys = append(bulkKeys, dbKey)
                        bulkCount = bulkCount + 1
                }

                if(bulkCount != totalCount) && ((bulkCount % MAX_BULK_ENTRIES_IN_PIPELINE) != 0) {
                        //If some entries are remaining and bulk bucket is not filled,
                        //go back to 'for' loop
                        continue
                }

		mCmd := map[string]*redis.StringStringMapCmd{}

		pipe := redisClient.Pipeline()

		for _, dbKey := range bulkKeys {

			redisKey := tableName + modelInfo.tableInfo[tableName].redisKeyDelim + dbKey
			//Check in validated cache first and add as dependent data
			if entry, mergeNeeded := c.fetchDataFromRequestCache(tableName, dbKey); (entry != nil) {
				 c.tmpDbCache[tableName].(map[string]interface{})[dbKey] = entry 
				 entryFetched = entryFetched + 1
				 //Entry found in validated cache, so skip fetching from Redis
				 //if merging is not required with Redis DB
				 if (mergeNeeded == false) {
					 continue
				 }
			}

			//Otherwise fetch it from Redis
			mCmd[dbKey] = pipe.HGetAll(redisKey) //write into pipeline
			if mCmd[dbKey] == nil {
				CVL_LOG(ERROR, "Failed pipe.HGetAll('%s')", redisKey)
			}
		}

		_, err := pipe.Exec()
		if err != nil {
			CVL_LOG(ERROR, "Failed to fetch details for table %s", tableName)
			return 0
		}
		pipe.Close()
		bulkKeys = nil

		mapTable := c.tmpDbCache[tableName]

		for key, val := range mCmd {
			res, err := val.Result()
			if (err != nil || len(res) == 0) {
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

	TRACE_LOG(INFO_API, TRACE_CACHE,"\n%v, Exiting fetchTableDataToTmpCache", time.Now())

	return entryFetched
}

//populate redis data to cache
func (c *CVL) fetchDataToTmpCache() *yparser.YParserNode {
	TRACE_LOG(INFO_API, TRACE_CACHE, "\n%v, Entered fetchToTmpCache", time.Now())

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
			if (len(dbKeys.(map[string]interface{}))  == 0) {
				 delete(c.tmpDbCache, tableName)
				 continue
			}
		}

		if (entryToFetch == 0) {
			//No more entry to fetch
			break
		}

		if Tracing {
			jsonDataBytes, _ := json.Marshal(c.tmpDbCache)
			jsonData := string(jsonDataBytes)
			TRACE_LOG(INFO_API, TRACE_CACHE, "Top Node=%v\n", jsonData)
		}

		data, err := jsonquery.ParseJsonMap(&c.tmpDbCache)

		if (err != nil) {
			return nil
		}

		//Build yang tree for each table and cache it
		for jsonNode := data.FirstChild; jsonNode != nil; jsonNode=jsonNode.NextSibling {
			TRACE_LOG(INFO_API, TRACE_CACHE, "Top Node=%v\n", jsonNode.Data)
			//Visit each top level list in a loop for creating table data
			topNode, _ := c.generateTableData(true, jsonNode)
			if (root == nil) {
				root = topNode
			} else {
				if root, errObj = c.yp.MergeSubtree(root, topNode); errObj.ErrCode != yparser.YP_SUCCESS {
					return nil
				}
			}
		}
	} // until all dependent data is fetched

	if root != nil && Tracing {
		dumpStr := c.yp.NodeDump(root)
		TRACE_LOG(INFO_API, TRACE_CACHE, "Dependent Data = %v\n", dumpStr)
	}

	TRACE_LOG(INFO_API, TRACE_CACHE, "\n%v, Exiting fetchToTmpCache", time.Now())
	return root
}


func (c *CVL) clearTmpDbCache() {
	for key := range c.tmpDbCache {
		delete(c.tmpDbCache, key)
	}
}


