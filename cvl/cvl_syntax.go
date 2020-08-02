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
	"github.com/antchfx/jsonquery"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

//Add child node to a parent node
func(c *CVL) addChildNode(tableName string, parent *yparser.YParserNode, name string) *yparser.YParserNode {

	//return C.lyd_new(parent, modelInfo.tableInfo[tableName].module, C.CString(name))
	return c.yp.AddChildNode(modelInfo.tableInfo[tableName].module, parent, name)
}

func (c *CVL) addChildLeaf(config bool, tableName string, parent *yparser.YParserNode, name string, value string, multileaf *[]*yparser.YParserLeafValue) {

	/* If there is no value then assign default space string. */
	if len(value) == 0 {
                value = " "
        }

	//Batch leaf creation
	*multileaf = append(*multileaf, &yparser.YParserLeafValue{Name: name, Value: value})

	//Check if this leaf has leafref,
	//If so add the add redis key to its table so that those 
	// details can be fetched for dependency validation

	c.addLeafRef(config, tableName, name, value)
}

func (c *CVL) generateTableFieldsData(config bool, tableName string, jsonNode *jsonquery.Node,
parent *yparser.YParserNode, multileaf *[]*yparser.YParserLeafValue) CVLErrorInfo {
	var cvlErrObj CVLErrorInfo

	//Traverse fields
	for jsonFieldNode := jsonNode.FirstChild; jsonFieldNode!= nil;
	jsonFieldNode = jsonFieldNode.NextSibling {
		//Add fields as leaf to the list
		if (jsonFieldNode.Type == jsonquery.ElementNode &&
		jsonFieldNode.FirstChild != nil &&
		jsonFieldNode.FirstChild.Type == jsonquery.TextNode) {

			if (len(modelInfo.tableInfo[tableName].mapLeaf) == 2) {//mapping should have two leaf always
				batchInnerListLeaf := make([]*yparser.YParserLeafValue, 0)
				//Values should be stored inside another list as map table
				listNode := c.addChildNode(tableName, parent, tableName) //Add the list to the top node
				c.addChildLeaf(config, tableName,
				listNode, modelInfo.tableInfo[tableName].mapLeaf[0],
				jsonFieldNode.Data, &batchInnerListLeaf)

				c.addChildLeaf(config, tableName,
				listNode, modelInfo.tableInfo[tableName].mapLeaf[1],
				jsonFieldNode.FirstChild.Data, &batchInnerListLeaf)

				if errObj := c.yp.AddMultiLeafNodes(modelInfo.tableInfo[tableName].module, listNode, batchInnerListLeaf); errObj.ErrCode != yparser.YP_SUCCESS {
					cvlErrObj = createCVLErrObj(errObj)
					CVL_LOG(ERROR, "Failed to create innner list leaf nodes, data = %v", batchInnerListLeaf)
					return cvlErrObj
				}
			} else {
				//check if it is hash-ref, then need to add only key from "TABLE|k1"
				hashRefMatch := reHashRef.FindStringSubmatch(jsonFieldNode.FirstChild.Data)

				if len(hashRefMatch) == 3 {

					c.addChildLeaf(config, tableName,
					parent, jsonFieldNode.Data,
					hashRefMatch[2], multileaf) //take hashref key value
				} else {
					c.addChildLeaf(config, tableName,
					parent, jsonFieldNode.Data,
					jsonFieldNode.FirstChild.Data, multileaf)
				}
			}

		} else if (jsonFieldNode.Type == jsonquery.ElementNode &&
		jsonFieldNode.FirstChild != nil &&
		jsonFieldNode.FirstChild.Type == jsonquery.ElementNode) {
			//Array data e.g. VLAN members
			for  arrayNode:=jsonFieldNode.FirstChild; arrayNode != nil;

			arrayNode = arrayNode.NextSibling {
				c.addChildLeaf(config, tableName,
				parent, jsonFieldNode.Data,
				arrayNode.FirstChild.Data, multileaf)
			}
		}
	}

	cvlErrObj.ErrCode = CVL_SUCCESS
	return cvlErrObj
}

func (c *CVL) generateTableData(config bool, jsonNode *jsonquery.Node)(*yparser.YParserNode, CVLErrorInfo) {
	var cvlErrObj CVLErrorInfo

	tableName := jsonNode.Data
	c.batchLeaf = nil
	c.batchLeaf = make([]*yparser.YParserLeafValue, 0)

	//Every Redis table is mapped as list within a container,
	//E.g. ACL_RULE is mapped as 
	// container ACL_RULE { list ACL_RULE_LIST {} }
	var topNode *yparser.YParserNode

	// Add top most conatiner e.g. 'container sonic-acl {...}'
	if _, exists := modelInfo.tableInfo[tableName]; !exists {
		CVL_LOG(ERROR, "Schema details not found for %s", tableName)
		cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
		cvlErrObj.TableName = tableName 
		cvlErrObj.Msg ="Schema details not found"
		return nil, cvlErrObj
	}
	topNode = c.yp.AddChildNode(modelInfo.tableInfo[tableName].module,
	nil, modelInfo.tableInfo[tableName].modelName)

	//Add the container node for each list 
	//e.g. 'container ACL_TABLE { list ACL_TABLE_LIST ...}
	listConatinerNode := c.yp.AddChildNode(modelInfo.tableInfo[tableName].module,
	topNode, tableName)

	//Traverse each key instance
	for jsonNode = jsonNode.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {

		//For each field check if is key 
		//If it is key, create list as child of top container
		// Get all key name/value pairs
		if yangListName := getRedisKeyToYangList(tableName, jsonNode.Data); yangListName!= "" {
			tableName = yangListName
		}
		keyValuePair := getRedisToYangKeys(tableName, jsonNode.Data)
		keyCompCount := len(keyValuePair)
		totalKeyComb := 1
		var keyIndices []int

		//Find number of all key combinations
		//Each key can have one or more key values, which results in nk1 * nk2 * nk2 combinations
		idx := 0
		for i := range keyValuePair {
			totalKeyComb = totalKeyComb * len(keyValuePair[i].values)
			keyIndices = append(keyIndices, 0)
		}

		for  ; totalKeyComb > 0 ; totalKeyComb-- {
			//Get the YANG list name from Redis table name
			//Ideally they are same except when one Redis table is split
			//into multiple YANG lists

			//Add table i.e. create list element
			listNode := c.addChildNode(tableName, listConatinerNode, tableName + "_LIST") //Add the list to the top node

			//For each key combination
			//Add keys as leaf to the list
			for idx = 0; idx < keyCompCount; idx++ {
				c.addChildLeaf(config, tableName,
				listNode, keyValuePair[idx].key,
				keyValuePair[idx].values[keyIndices[idx]], &c.batchLeaf)
			}

			//Get all fields under the key field and add them as children of the list
			if fldDataErrObj := c.generateTableFieldsData(config, tableName, jsonNode, listNode, &c.batchLeaf); fldDataErrObj.ErrCode != CVL_SUCCESS {
				return nil, fldDataErrObj
			}

			//Check which key elements left after current key element
			var next int = keyCompCount - 1
			for  ((next > 0) && ((keyIndices[next] +1) >=  len(keyValuePair[next].values))) {
				next--
			}
			//No more combination possible
			if (next < 0) {
				break
			}

			keyIndices[next]++

			//Reset indices for all other key elements
			for idx = next+1;  idx < keyCompCount; idx++ {
				keyIndices[idx] = 0
			}

			TRACE_LOG(INFO_API, TRACE_CACHE, "Starting batch leaf creation - %v\n", c.batchLeaf)
			//process batch leaf creation
			if errObj := c.yp.AddMultiLeafNodes(modelInfo.tableInfo[tableName].module, listNode, c.batchLeaf); errObj.ErrCode != yparser.YP_SUCCESS {
				cvlErrObj = createCVLErrObj(errObj)
				CVL_LOG(ERROR, "Failed to create leaf nodes, data = %v",
				c.batchLeaf)
				return nil, cvlErrObj
			}

			c.batchLeaf = nil
			c.batchLeaf = make([]*yparser.YParserLeafValue, 0)
		}
	}

	return topNode, cvlErrObj
}

