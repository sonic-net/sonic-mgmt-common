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
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	"github.com/antchfx/jsonquery"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/google/go-cmp/cmp"

	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

// YValidator YANG Validator used for external semantic
// validation including custom/platform validation
type YValidator struct {
	root    *xmlquery.Node //Top evel root for data
	current *xmlquery.Node //Current position
	//operation string     //Edit operation
}

type DepDataCallBack func(ctxt interface{}, redisKeys []string, redisKeyFilter, keyNames, pred, fields, count string) string
type DepDataCountCallBack func(ctxt interface{}, redisKeyFilter, keyNames, pred, field string) float64

var depDataCb DepDataCallBack = func(ctxt interface{}, redisKeys []string, redisKeyFilter, keyNames, pred, fields, count string) string {
	c := ctxt.(*CVL)
	return c.addDepYangData(redisKeys, redisKeyFilter, keyNames, pred, fields, 0)
}

var depDataCountCb DepDataCountCallBack = func(ctxt interface{}, redisKeyFilter, keyNames, pred, field string) float64 {
	if pred != "" {
		pred = "return (" + pred + ")"
	}

	c := ctxt.(*CVL)
	s := cmn.Search{Pattern: redisKeyFilter, Predicate: pred, KeyNames: strings.Split(keyNames, "|"), WithField: field}
	redisEntries, err := c.dbAccess.Count(s).Result()
	count := float64(0)

	if (err == nil) && (redisEntries > 0) {
		count = float64(redisEntries)
	}

	if IsTraceAllowed(TRACE_SEMANTIC) {
		TRACE_LOG(TRACE_SEMANTIC, "depDataCountCb() with redisKeyFilter=%s, keyNames= %s, predicate=%s, fields=%s, returned = %v", redisKeyFilter, keyNames, pred, field, count)
	}

	return count
}

// Generate leaf/leaf-list YANG data
func (c *CVL) generateYangLeafData(tableName string, jsonNode *jsonquery.Node,
	parent *xmlquery.Node) CVLRetCode {

	//Traverse fields
	for jsonFieldNode := jsonNode.FirstChild; jsonFieldNode != nil; jsonFieldNode = jsonFieldNode.NextSibling {
		//Add fields as leaf to the list
		if jsonFieldNode.Type == jsonquery.ElementNode &&
			jsonFieldNode.FirstChild != nil &&
			jsonFieldNode.FirstChild.Type == jsonquery.TextNode {

			if len(modelInfo.tableInfo[tableName].mapLeaf) == 2 { //mapping should have two leaf always
				//Values should be stored inside another list as map table
				listNode := c.addYangNode(tableName, parent, tableName, "") //Add the list to the top node
				c.addYangNode(tableName,
					listNode, modelInfo.tableInfo[tableName].mapLeaf[0],
					jsonFieldNode.Data)

				c.addYangNode(tableName,
					listNode, modelInfo.tableInfo[tableName].mapLeaf[1],
					jsonFieldNode.FirstChild.Data)

			} else {
				//check if it is hash-ref, then need to add only key from "TABLE|k1"
				hashRefMatch := reHashRef.FindStringSubmatch(jsonFieldNode.FirstChild.Data)

				if len(hashRefMatch) == 3 {
					c.addYangNode(tableName,
						parent, jsonFieldNode.Data,
						hashRefMatch[2]) //take hashref key value
				} else {
					c.addYangNode(tableName,
						parent, jsonFieldNode.Data,
						jsonFieldNode.FirstChild.Data)
				}
			}

		} else if jsonFieldNode.Type == jsonquery.ElementNode &&
			jsonFieldNode.FirstChild != nil &&
			jsonFieldNode.FirstChild.Type == jsonquery.ElementNode {
			//Array data e.g. VLAN members@ or 'ports@'
			for arrayNode := jsonFieldNode.FirstChild; arrayNode != nil; arrayNode = arrayNode.NextSibling {

				node := c.addYangNode(tableName,
					parent, jsonFieldNode.Data,
					arrayNode.FirstChild.Data)

				//mark these nodes as leaf-list
				addAttrNode(node, "leaf-list", "")
			}
		}
	}

	//Add all the default nodes required for must and when exps evaluation
	for nodeName, valStr := range modelInfo.tableInfo[tableName].dfltLeafVal {
		//Check if default node is already present in data
		var child *xmlquery.Node
		for child = parent.FirstChild; child != nil; child = child.NextSibling {
			if child.Data == nodeName {
				break
			}
		}

		if child != nil {
			//node is already present, skip adding it
			continue
		}

		valArr := strings.Split(valStr, ",")
		for idx := 0; idx < len(valArr); idx++ {
			node := c.addYangNode(tableName,
				parent, nodeName, valArr[idx])

			//mark these nodes as leaf-list
			if len(valArr) > 1 {
				addAttrNode(node, "leaf-list", "")
			}
		}
	}

	return CVL_SUCCESS
}

// Add attribute YANG node
func addAttrNode(n *xmlquery.Node, key, val string) {
	var attr xml.Attr = xml.Attr{
		Name:  xml.Name{Local: key},
		Value: val,
	}

	n.Attr = append(n.Attr, attr)
}

func getAttrNodeVal(node *xmlquery.Node, name string) string {
	if node == nil {
		return ""
	}

	if len(node.Attr) == 0 {
		return ""
	}

	for idx := 0; idx < len(node.Attr); idx++ {
		if node.Attr[idx].Name.Local == name {
			return node.Attr[idx].Value
		}
	}

	return ""
}

// Add YANG node with or without parent, with or without value
func (c *CVL) addYangNode(tableName string, parent *xmlquery.Node,
	name string, value string) *xmlquery.Node {

	//Create the node
	node := &xmlquery.Node{Parent: parent, Data: name,
		Type: xmlquery.ElementNode}

	//Set prefix from parent
	if parent != nil {
		node.Prefix = parent.Prefix
	}

	if value != "" {
		//Create the value node
		textNode := &xmlquery.Node{Data: value, Type: xmlquery.TextNode}
		node.FirstChild = textNode
		node.LastChild = textNode
	}

	if parent == nil {
		//Creating top node
		return node
	}

	if parent.FirstChild == nil {
		//Create as first child
		parent.FirstChild = node
		parent.LastChild = node

	} else {
		//Append as sibling
		tmp := parent.LastChild
		tmp.NextSibling = node
		node.PrevSibling = tmp
		parent.LastChild = node
	}

	return node
}

// Generate YANG list data along with top container,
// table container.
// If needed, stores the list pointer against each request table/key
// in requestCahce so that YANG data can be reached
// directly on given table/key
func (c *CVL) generateYangListData(jsonNode *jsonquery.Node,
	storeInReqCache bool) (*xmlquery.Node, CVLErrorInfo) {
	var cvlErrObj CVLErrorInfo

	tableName := jsonNode.Data
	origTableName := tableName
	c.batchLeaf = nil
	c.batchLeaf = make([]*yparser.YParserLeafValue, 0)

	//Every Redis table is mapped as list within a container,
	//E.g. ACL_RULE is mapped as
	// container ACL_RULE { list ACL_RULE_LIST {} }
	var topNode *xmlquery.Node
	var listConatinerNode *xmlquery.Node
	topNodesAdded := false

	//Traverse each key instance
	keyPresent := false
	for jsonNode = jsonNode.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {
		//store the redis key
		redisKey := jsonNode.Data

		//Mark at least one key is present
		keyPresent = true

		//For each field check if is key
		//If it is key, create list as child of top container
		// Get all key name/value pairs
		if yangListName := getRedisTblToYangList(origTableName, redisKey); yangListName != "" {
			tableName = yangListName
		}
		if _, exists := modelInfo.tableInfo[tableName]; !exists {
			CVL_LOG(WARNING, "Failed to find schema details for table %s", origTableName)
			cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
			cvlErrObj.TableName = origTableName
			cvlErrObj.Msg = "Schema details not found"
			return nil, cvlErrObj
		}
		if !topNodesAdded {
			// Add top most container e.g. 'container sonic-acl {...}'
			topNode = c.addYangNode(origTableName, nil, modelInfo.tableInfo[tableName].modelName, "")
			//topNode.Prefix = modelInfo.modelNs[modelInfo.tableInfo[tableName].modelName].prefix
			topNode.Prefix = modelInfo.tableInfo[tableName].modelName
			topNode.NamespaceURI = modelInfo.modelNs[modelInfo.tableInfo[tableName].modelName].ns

			//Add the container node for each list
			//e.g. 'container ACL_TABLE { list ACL_TABLE_LIST ...}
			listConatinerNode = c.addYangNode(origTableName, topNode, origTableName, "")
			topNodesAdded = true
		}
		keyValuePair := getRedisToYangKeys(tableName, redisKey)
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

		for ; totalKeyComb > 0; totalKeyComb-- {
			//Get the YANG list name from Redis table name
			//Ideally they are same except when one Redis table is split
			//into multiple YANG lists

			//Add table i.e. create list element
			listNode := c.addYangNode(tableName, listConatinerNode, tableName+"_LIST", "") //Add the list to the top node
			addAttrNode(listNode, "key", redisKey)

			if storeInReqCache {
				//store the list pointer in requestCache against the table/key
				reqCache, exists := c.requestCache[tableName][redisKey]
				if exists {
					//Store same list instance in all requests under same table/key
					for idx := 0; idx < len(reqCache); idx++ {
						//Save the YANG data tree for using it later
						reqCache[idx].YangData = listNode
					}
				}
			}

			//For each key combination
			//Add keys as leaf to the list
			for idx = 0; idx < keyCompCount; idx++ {
				c.addYangNode(tableName, listNode, keyValuePair[idx].key,
					keyValuePair[idx].values[keyIndices[idx]])
			}

			//Get all fields under the key field and add them as children of the list
			c.generateYangLeafData(tableName, jsonNode, listNode)

			//Check which key elements left after current key element
			var next int = keyCompCount - 1
			for (next > 0) && ((keyIndices[next] + 1) >= len(keyValuePair[next].values)) {
				next--
			}
			//No more combination possible
			if next < 0 {
				break
			}

			keyIndices[next]++

			//Reset indices for all other key elements
			for idx = next + 1; idx < keyCompCount; idx++ {
				keyIndices[idx] = 0
			}
		}
	}

	if !keyPresent {
		return nil, cvlErrObj
	}

	return topNode, cvlErrObj
}

// Append given children to destNode
func (c *CVL) appendSubtree(dest, src *xmlquery.Node) CVLRetCode {
	if dest == nil || src == nil {
		return CVL_FAILURE
	}

	var lastSibling *xmlquery.Node = nil

	for srcNode := src; srcNode != nil; srcNode = srcNode.NextSibling {
		//set parent for all nodes
		srcNode.Parent = dest
		lastSibling = srcNode
	}

	if dest.LastChild == nil {
		//No sibling in dest yet
		dest.FirstChild = src
		dest.LastChild = lastSibling
	} else {
		//Append to the last sibling
		dest.LastChild.NextSibling = src
		src.PrevSibling = dest.LastChild
		dest.LastChild = lastSibling
	}

	return CVL_SUCCESS
}

// Return subtree after detaching from parent
func (c *CVL) detachSubtree(parent *xmlquery.Node) *xmlquery.Node {

	child := parent.FirstChild

	if child != nil {
		//set children to nil
		parent.FirstChild = nil
		parent.LastChild = nil
	} else {
		//No children
		return nil
	}

	//Detach all children from parent
	for node := child; node != nil; node = node.NextSibling {
		node.Parent = nil
	}

	return child
}

// Detach a node from its parent
func (c *CVL) detachNode(node *xmlquery.Node) CVLRetCode {
	if node == nil {
		return CVL_FAILURE
	}

	//get the parent node
	parent := node.Parent

	if parent == nil {
		//Already detached node
		return CVL_SUCCESS
	}

	//adjust siblings
	if parent.FirstChild == node && parent.LastChild == node {
		//this is the only node
		parent.FirstChild = nil
		parent.LastChild = nil
	} else if parent.FirstChild == node {
		//first child, set new first child
		parent.FirstChild = node.NextSibling
		node.NextSibling.PrevSibling = nil
	} else {
		node.PrevSibling.NextSibling = node.NextSibling
		if node.NextSibling != nil {
			//if remaining sibling
			node.NextSibling.PrevSibling = node.PrevSibling
		} else {
			//this is last child getting detached,
			//so set lastChild as node's prevSibling
			parent.LastChild = node.PrevSibling
		}
	}

	//detach from parent and siblings
	node.Parent = nil
	node.PrevSibling = nil
	node.NextSibling = nil

	return CVL_SUCCESS
}

// Delete all leaf-list nodes in destination
// Leaf-list should be replaced from source
// destination
func (c *CVL) deleteDestLeafList(dest *xmlquery.Node) {

	TRACE_LOG(TRACE_CACHE, "Updating leaf-list by "+
		"removing and then adding leaf-list")

	//find start and end of dest leaf list
	leafListName := dest.Data
	for node := dest; node != nil; {
		tmpNextNode := node.NextSibling

		if node.Data == leafListName {
			c.detachNode(node)
			node = tmpNextNode
			continue
		} else {
			//no more leaflist node
			break
		}
	}
}

// deleteLeafNodes removes specified child nodes from an xml node topNode
func (c *CVL) deleteLeafNodes(topNode *xmlquery.Node, cfgData map[string]string) {
	for node := topNode.FirstChild; node != nil; {
		if _, found := cfgData[getNodeName(node)]; found {
			tmpNode := node.NextSibling
			c.detachNode(node)
			node = tmpNode
		} else {
			node = node.NextSibling
		}
	}
}

// Check if the given list src node already exists in dest node
func (c *CVL) checkIfListNodeExists(dest, src *xmlquery.Node) *xmlquery.Node {
	if (dest == nil) || (src == nil) {
		return nil
	}

	tableName := getYangListToRedisTbl(src.Data)
	redisKey := getAttrNodeVal(src, "key")

	if tableName == "" || redisKey == "" {
		return nil
	}

	entry, exists := c.requestCache[tableName][redisKey]
	if !exists || (len(entry) == 0) {
		return nil
	}

	//CREATE/UPDATE/DELETE request for same table/key points to
	//same yang list in request cache
	yangList := entry[0].YangData

	if yangList == nil || yangList.Parent == nil {
		//Source node does not exist in destination
		return nil
	}

	if dest.Parent == yangList.Parent {
		//Same parent means yang list already exists in destination tree
		return yangList
	}

	return nil
}

// Merge YANG data recursively from dest to src
// Leaf-list is always replaced and appeneded at
// the end of list's children
func (c *CVL) mergeYangData(dest, src *xmlquery.Node) CVLRetCode {
	if (dest == nil) || (src == nil) {
		return CVL_FAILURE
	}

	TRACE_LOG((TRACE_SYNTAX | TRACE_SEMANTIC),
		"Merging YANG data %s %s", dest.Data, src.Data)

	if (dest.Type == xmlquery.TextNode) && (src.Type == xmlquery.TextNode) {
		//handle leaf node by updating value
		dest.Data = src.Data
		return CVL_SUCCESS
	}

	srcNode := src

	destLeafListDeleted := make(map[string]bool)
	for srcNode != nil {
		//Find all source nodes and attach to the matching destination node
		ret := CVL_FAILURE
		//TRACE_LOG((TRACE_SYNTAX | TRACE_SEMANTIC), "MergeData : src loop\n")
	destLoop:
		destNode := dest
		for ; destNode != nil; destNode = destNode.NextSibling {
			//TRACE_LOG((TRACE_SYNTAX | TRACE_SEMANTIC), "MergeData : dest loop\n")
			if destNode.Data != srcNode.Data {
				//Can proceed to subtree only if current node name matches
				continue
			}

			if strings.HasSuffix(destNode.Data, "_LIST") {
				//Check if src list node already exists in destination
				tmpNode := c.checkIfListNodeExists(destNode, srcNode)
				if tmpNode != nil {
					destNode = tmpNode
				} else {
					destNode = tmpNode
					break
				}
				//find exact match for list instance
				//check with key value, stored in attribute
				/*if (len(destNode.Attr) == 0) || (len(srcNode.Attr) == 0) ||
				(destNode.Attr[0].Value != srcNode.Attr[0].Value) {
					//move to next list
					continue
				}*/
			} else if (len(destNode.Attr) > 0) && (len(srcNode.Attr) > 0) &&
				(destNode.Attr[0].Name.Local == "leaf-list") &&
				(srcNode.Attr[0].Name.Local == "leaf-list") { // attribute has type

				delFlag, exists := destLeafListDeleted[srcNode.Data]

				if !exists || !delFlag {
					//Replace all leaf-list nodes from destination first
					c.deleteDestLeafList(destNode)
					destLeafListDeleted[srcNode.Data] = true
					//Note that 'dest' still points to list keys
					//even though all leaf-list might have been deleted
					//as we never delete key while merging
					goto destLoop
				} else {
					//if all dest leaflist deleted,
					//just break to add all leaflist
					destNode = nil
					break
				}
			}

			//Go to their children
			ret = c.mergeYangData(destNode.FirstChild, srcNode.FirstChild)

			//Node matched break now
			break

		} //dest node loop

		if ret == CVL_FAILURE {
			if destNode == nil {
				//destNode == nil -> node not found
				//detach srcNode and append to dest
				tmpNextSrcNode := srcNode.NextSibling
				if CVL_SUCCESS == c.detachNode(srcNode) {
					if (len(srcNode.Attr) > 0) &&
						(srcNode.Attr[0].Name.Local == "leaf-list") {
						//set the flag so that we don't delete leaf-list
						//from destNode further
						destLeafListDeleted[srcNode.Data] = true
					}
					c.appendSubtree(dest.Parent, srcNode)
				}
				srcNode = tmpNextSrcNode
				continue
			} else {
				//subtree merge failure ,, append subtree
				subTree := c.detachSubtree(srcNode)
				if subTree != nil {
					c.appendSubtree(destNode, subTree)
				}
			}
		}

		srcNode = srcNode.NextSibling
	} //src node loop

	return CVL_SUCCESS
}

func (c *CVL) findYangList(tableName string, redisKey string) *xmlquery.Node {
	origCurrent := c.yv.current
	tmpCurrent := c.moveToYangList(tableName, redisKey)
	c.yv.current = origCurrent

	return tmpCurrent
}

// Locate YANG list instance in root for given table name and key
func (c *CVL) moveToYangList(tableName string, redisKey string) *xmlquery.Node {

	var nodeTbl *xmlquery.Node = nil

	redisTableName := getYangListToRedisTbl(tableName)
	modelName := modelInfo.tableInfo[tableName].modelName

	//move to the model first
	for node := c.yv.root.FirstChild; node != nil; node = node.NextSibling {
		if node.Data != modelName {
			continue
		}

		//Move to container
		for nodeTbl = node.FirstChild; nodeTbl != nil; nodeTbl = nodeTbl.NextSibling {
			if nodeTbl.Data == redisTableName {
				break
			}
		}

		break
	}

	if nodeTbl == nil {
		TRACE_LOG(TRACE_SEMANTIC, "YANG data for table %s, key %s is not present in YANG tree",
			tableName, redisKey)
		return nil
	}

	//Move to list
	listName := tableName + "_LIST"
	for nodeList := nodeTbl.FirstChild; nodeList != nil; nodeList = nodeList.NextSibling {
		if nodeList.Data != listName {
			continue
		}

		c.yv.current = nodeList
		//if no key specified or no other instance exists,
		//just return the first list instance
		if redisKey == "" || nodeList.NextSibling == nil {
			return c.yv.current
		}

		for ; nodeList != nil; nodeList = nodeList.NextSibling {
			if len(nodeList.Attr) > 0 {
				if cmn.KeyMatch(nodeList.Attr[0].Value, redisKey) {
					c.yv.current = nodeList
					return nodeList
				}

			}
		}
		if nodeList == nil {
			break
		}
	}

	CVL_LOG(WARNING, "No list entry matched, unable to find YANG data for table %s, key %s",
		tableName, redisKey)
	return nil
}

// Set operation node value based on operation in request received
func (c *CVL) setOperation(op cmn.CVLOperation) {

	var node *xmlquery.Node

	for node = c.yv.root.FirstChild; node != nil; node = node.NextSibling {
		if node.Data == "operation" {
			break
		}
	}

	//Add the operation container
	if node == nil {
		node = c.addYangNode("", c.yv.root, "operation", "")
		node.Prefix = "sonic-common" //"cmn"
		//modelInfo.modelNs["sonic-common"].prefix
		node.NamespaceURI = modelInfo.modelNs["sonic-common"].ns
	}

	opNode := node.FirstChild
	if opNode == nil {
		node.Prefix = "sonic-common" //"cmn"
		opNode = c.addYangNode("", node, "operation", "NONE")
	}

	switch op {
	case cmn.OP_CREATE:
		opNode.FirstChild.Data = "CREATE"
	case cmn.OP_UPDATE:
		opNode.FirstChild.Data = "UPDATE"
	case cmn.OP_DELETE:
		opNode.FirstChild.Data = "DELETE"
	default:
		opNode.FirstChild.Data = "NONE"
	}
}

// Add given YANG data buffer to Yang Validator
// redisKeys - Set of redis keys
// redisKeyFilter - Redis key filter in glob style pattern
// keyNames - Names of all keys separated by "|"
// predicate - Condition on keys/fields
// fields - Fields to retrieve, separated by "|"
// Return "," separated list of leaf nodes if only one leaf is requested
// One leaf is used as xpath query result in other nested xpath
func (c *CVL) addDepYangData(redisKeys []string, redisKeyFilter,
	keyNames, predicate, fields string, count int) string {

	var v interface{}
	tmpPredicate := ""

	//Get filtered Redis data based on lua script
	//filter derived from Xpath predicate
	if predicate != "" {
		tmpPredicate = "return (" + predicate + ")"
	}

	s := cmn.Search{Pattern: redisKeyFilter, Predicate: tmpPredicate, KeyNames: strings.Split(keyNames, "|"), WithField: fields, Limit: count}
	cfgData, err := c.dbAccess.Lookup(s).Result()

	singleLeaf := "" //leaf data for single leaf

	if IsTraceAllowed(TRACE_SEMANTIC) {
		TRACE_LOG(TRACE_SEMANTIC, "addDepYangData() with redisKeyFilter=%s, "+
			"predicate=%s, fields=%s, returned cfgData = %s, err=%v",
			redisKeyFilter, predicate, fields, cfgData, err)
	}

	if len(cfgData) == 0 {
		return ""
	}

	// If dependent data is already being added for semantic evaluation, don't
	// add again. This prevents adding same data multiple times and reduce size
	// of xml generated for semantic evaluation by xpath engine.
	if _, ok := c.depDataCache[cfgData]; ok {
		return ""
	}
	c.depDataCache[cfgData] = nil

	//Parse the JSON map received from lua script
	b := []byte(cfgData)
	if err := json.Unmarshal(b, &v); err != nil {
		return ""
	}

	var dataMap map[string]interface{} = v.(map[string]interface{})

	dataTop, _ := jsonquery.ParseJsonMap(&dataMap)

	for jsonNode := dataTop.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {
		//Generate YANG data for Yang Validator from Redis JSON
		topYangNode, _ := c.generateYangListData(jsonNode, false)

		if topYangNode == nil {
			continue
		}

		if (topYangNode.FirstChild != nil) &&
			(topYangNode.FirstChild.FirstChild != nil) {
			//Add attribute mentioning that data is from db
			addAttrNode(topYangNode.FirstChild.FirstChild, "db", "")
		}

		//Build single leaf data requested
		singleLeaf = ""
		for redisKey := topYangNode.FirstChild.FirstChild; redisKey != nil; redisKey = redisKey.NextSibling {

			for field := redisKey.FirstChild; field != nil; field = field.NextSibling {
				if field.Data == fields && field.FirstChild != nil {
					//Single field requested
					singleLeaf = singleLeaf + field.FirstChild.Data + ","
					break
				}
			}
		}

		//Merge with main YANG data cache
		doc := &xmlquery.Node{Type: xmlquery.DocumentNode}
		doc.FirstChild = topYangNode
		doc.LastChild = topYangNode
		topYangNode.Parent = doc
		if c.mergeYangData(c.yv.root, doc) != CVL_SUCCESS {
			continue
		}
	}

	//remove last comma in case mulitple values returned
	if singleLeaf != "" {
		return singleLeaf[:len(singleLeaf)-1]
	}

	return ""
}

// Add all other table data for validating all 'must' exp for tableName
// One entry is needed for incremental loading of must tables
func (c *CVL) addYangDataForMustExp(op cmn.CVLOperation, tableName string, oneEntry bool) CVLRetCode {
	if modelInfo.tableInfo[tableName].mustExpr == nil {
		return CVL_SUCCESS
	}
	defer c.clearTmpDbCache()

	for mustTblName, mustOp := range modelInfo.tableInfo[tableName].tablesForMustExp {
		//First check if must expression should be executed for the given operation
		if (mustOp != cmn.OP_NONE) && ((mustOp & op) == cmn.OP_NONE) {
			//must to be excuted for particular operation, but current operation
			//is not the same one
			continue
		}

		//If one entry is needed and it is already availale in c.yv.root cache
		//just ignore and continue
		if oneEntry {
			node := c.moveToYangList(mustTblName, "")
			if node != nil {
				//One entry exists, continue
				continue
			}
		}

		redisTblName := getYangListToRedisTbl(mustTblName) //1 yang to N Redis table case
		tableKeys, err := c.dbAccess.Keys(redisTblName +
			modelInfo.tableInfo[mustTblName].redisKeyDelim + "*").Result()

		if err != nil {
			return CVL_FAILURE
		}

		if len(tableKeys) == 0 {
			//No dependent data for mustTable available
			continue
		}

		c.clearTmpDbCache()

		//fill all keys; TBD Optimize based on predicate in Xpath
		tablePrefixLen := len(redisTblName + modelInfo.tableInfo[mustTblName].redisKeyDelim)
		for _, tableKey := range tableKeys {
			tableKey = tableKey[tablePrefixLen:] //remove table prefix

			tmpKeyArr := strings.Split(tableKey, modelInfo.tableInfo[mustTblName].redisKeyDelim)
			if len(tmpKeyArr) != len(modelInfo.tableInfo[mustTblName].keys) {
				//Number of keys should be same as in YANG list keys
				//Need to check this for one Redis table to many YANG list case
				continue
			}

			if c.tmpDbCache[redisTblName] == nil {
				c.tmpDbCache[redisTblName] = map[string]interface{}{tableKey: nil}
			} else {
				tblMap := c.tmpDbCache[redisTblName]
				tblMap.(map[string]interface{})[tableKey] = nil
				c.tmpDbCache[redisTblName] = tblMap
			}
			//Load only one entry
			if oneEntry {
				TRACE_LOG(TRACE_SEMANTIC, "addYangDataForMustExp(): Adding one entry table %s, key %s",
					redisTblName, tableKey)
				break
			}
		}

		if c.tmpDbCache[redisTblName] == nil {
			//No entry present in DB
			continue
		}

		//fetch using pipeline
		c.fetchTableDataToTmpCache(redisTblName, c.tmpDbCache[redisTblName].(map[string]interface{}))
		data, err := jsonquery.ParseJsonMap(&c.tmpDbCache)

		if err != nil {
			return CVL_FAILURE
		}

		//Build yang tree for each table and cache it
		for jsonNode := data.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {
			//Visit each top level list in a loop for creating table data
			topYangNode, _ := c.generateYangListData(jsonNode, false)

			if topYangNode == nil {
				//No entry found, check next entry
				continue
			}

			if (topYangNode.FirstChild != nil) &&
				(topYangNode.FirstChild.FirstChild != nil) {
				//Add attribute mentioning that data is from db
				addAttrNode(topYangNode.FirstChild.FirstChild, "db", "")
			}

			//Create full document by adding document node
			doc := &xmlquery.Node{Type: xmlquery.DocumentNode}
			doc.FirstChild = topYangNode
			doc.LastChild = topYangNode
			topYangNode.Parent = doc
			if c.mergeYangData(c.yv.root, doc) != CVL_SUCCESS {
				return CVL_INTERNAL_UNKNOWN
			}
		}
	}

	return CVL_SUCCESS
}

// Compile all must expression and save the expression tree
func compileMustExps() {
	reMultiPred := regexp.MustCompile(`\][ ]*\[`)

	for _, tInfo := range modelInfo.tableInfo {
		if tInfo.mustExpr == nil {
			continue
		}

		// Replace multiple predicate using 'and' expressiona
		// xpath engine not accepting multiple predicates
		for _, mustExprArr := range tInfo.mustExpr {
			for _, mustExpr := range mustExprArr {
				mustExpr.exprTree = xpath.MustCompile(
					reMultiPred.ReplaceAllString(mustExpr.expr, " and "))
			}
		}
	}
}

// Compile all when expression and save the expression tree
func compileWhenExps() {
	reMultiPred := regexp.MustCompile(`\][ ]*\[`)

	for _, tInfo := range modelInfo.tableInfo {
		if tInfo.whenExpr == nil {
			continue
		}

		// Replace multiple predicate using 'and' expressiona
		// xpath engine not accepting multiple predicates
		for _, whenExprArr := range tInfo.whenExpr {
			for _, whenExpr := range whenExprArr {
				whenExpr.exprTree = xpath.MustCompile(
					reMultiPred.ReplaceAllString(whenExpr.expr, " and "))
				//Store all YANG list used in the expression
				whenExpr.yangListNames = getYangListNamesInExpr(whenExpr.expr)
			}
		}
	}
}

func compileLeafRefPath() {
	reMultiPred := regexp.MustCompile(`\][ ]*\[`)

	for _, tInfo := range modelInfo.tableInfo {
		if len(tInfo.leafRef) == 0 { //no leafref
			continue
		}

		//for  nodeName, leafRefArr := range tInfo.leafRef {
		for _, leafRefArr := range tInfo.leafRef {
			for _, leafRefArrItem := range leafRefArr {
				if leafRefArrItem.path == "non-leafref" {
					//Leaf type has at-least one non-learef data type
					continue
				}

				//first store the referred table and target node
				leafRefArrItem.yangListNames, leafRefArrItem.targetNodeName =
					getLeafRefTargetInfo(leafRefArrItem.path)
				//check if predicate is used in path
				//for complex expression, xpath engine is
				//used for evaluation,
				//else don't build expression tree,
				//it is handled by just checking redis entry
				if strings.Contains(leafRefArrItem.path, "[") &&
					strings.Contains(leafRefArrItem.path, "]") {
					//Compile the xpath in leafref
					tmpExp := reMultiPred.ReplaceAllString(leafRefArrItem.path, " and ")
					//tmpExp = nodeName + " = " + tmpExp
					tmpExp = "current() = " + tmpExp
					leafRefArrItem.exprTree = xpath.MustCompile(tmpExp)
				}
			}
		}
	}
}

// Validate must expression
func (c *CVL) validateMustExp(node *xmlquery.Node,
	tableName, key string, op cmn.CVLOperation) (r CVLErrorInfo) {
	defer func() {
		ret := &r
		CVL_LOG(INFO_API, "validateMustExp(): table name = %s, "+
			"return value = %v", tableName, *ret)
	}()

	c.setOperation(op)

	//Set xpath callback for retreiving dependent data
	xpath.SetDepDataClbk(c, depDataCb)

	//Set xpath callback for retriving dependent data count
	xpath.SetDepDataCntClbk(c, depDataCountCb)

	if node == nil || node.FirstChild == nil {
		return CVLErrorInfo{
			TableName:     tableName,
			ErrCode:       CVL_SEMANTIC_ERROR,
			CVLErrDetails: cvlErrorMap[CVL_SEMANTIC_ERROR],
			Msg:           "Failed to find YANG data for must expression validation",
		}
	}

	//Load all table's any one entry for 'must' expression execution.
	//This helps building full YANG tree for tables needed.
	//Other instances/entries would be fetched as per xpath predicate execution
	//during expression evaluation
	c.addYangDataForMustExp(op, tableName, true)

	//Find the node where must expression is attached
	for nodeName, mustExpArr := range modelInfo.tableInfo[tableName].mustExpr {
		for _, mustExp := range mustExpArr {
			ctxNode := node
			if ctxNode.Data != nodeName { //must expression at list level
				ctxNode = ctxNode.FirstChild
				for (ctxNode != nil) && (ctxNode.Data != nodeName) {
					ctxNode = ctxNode.NextSibling //must expression at leaf level
				}
				if ctxNode != nil && op == cmn.OP_UPDATE {
					addAttrNode(ctxNode, "db", "")
				}
			}

			//Check leafref for each leaf-list node
			/*for ;(ctxNode != nil) && (ctxNode.Data == nodeName);
			ctxNode = ctxNode.NextSibling {
				//Load first data for each referred table.
				//c.yv.root has all requested data merged and any depdendent
				//data needed for leafref validation should be available from this.

				leafRefSuccess := false*/

			if ctxNode != nil {
				CVL_LOG(INFO_DEBUG, "Eval must \"%s\"; ctxNode=%s",
					mustExp.expr, ctxNode.Data)

				if !xmlquery.Eval(c.yv.root, ctxNode, mustExp.exprTree) {
					keys := []string{}
					if len(ctxNode.Parent.Attr) > 0 {
						keys = strings.Split(ctxNode.Parent.Attr[0].Value,
							modelInfo.tableInfo[tableName].redisKeyDelim)
					}

					return CVLErrorInfo{
						TableName:        tableName,
						ErrCode:          CVL_SEMANTIC_ERROR,
						CVLErrDetails:    cvlErrorMap[CVL_SEMANTIC_ERROR],
						Keys:             keys,
						Value:            ctxNode.FirstChild.Data,
						Field:            nodeName,
						Msg:              "Must expression validation failed",
						ConstraintErrMsg: mustExp.errStr,
						ErrAppTag:        mustExp.errCode,
					}
				}
			}
		} //for each must exp
	} //all must exp under one node

	return CVLErrorInfo{ErrCode: CVL_SUCCESS}
}

// Currently supports when expression with current table only
func (c *CVL) validateWhenExp(node *xmlquery.Node,
	tableName, key string, op cmn.CVLOperation) (r CVLErrorInfo) {

	defer func() {
		ret := &r
		CVL_LOG(INFO_API, "validateWhenExp(): table name = %s, "+
			"return value = %v", tableName, *ret)
	}()

	if op == cmn.OP_DELETE {
		//No new node getting added so skip when validation
		return CVLErrorInfo{ErrCode: CVL_SUCCESS}
	}

	//Set xpath callback for retreiving dependent data
	xpath.SetDepDataClbk(c, depDataCb)

	if node == nil || node.FirstChild == nil {
		return CVLErrorInfo{
			TableName:     tableName,
			ErrCode:       CVL_SEMANTIC_ERROR,
			CVLErrDetails: cvlErrorMap[CVL_SEMANTIC_ERROR],
			Msg:           "Failed to find YANG data for must expression validation",
		}
	}

	//Find the node where when expression is attached
	for nodeName, whenExpArr := range modelInfo.tableInfo[tableName].whenExpr {
		for _, whenExp := range whenExpArr { //for each when expression
			ctxNode := node
			if ctxNode.Data != nodeName { //when expression not at list level
				ctxNode = ctxNode.FirstChild
				for (ctxNode != nil) && (ctxNode.Data != nodeName) {
					ctxNode = ctxNode.NextSibling //whent expression at leaf level
				}
			}

			//Add data for dependent table in when expression
			//Add one entry only
			for _, refListName := range whenExp.yangListNames {
				refRedisTableName := getYangListToRedisTbl(refListName)

				filter := refRedisTableName +
					modelInfo.tableInfo[refListName].redisKeyDelim + "*"

				c.addDepYangData([]string{}, filter,
					strings.Join(modelInfo.tableInfo[refListName].keys, "|"),
					"true", "", 1) //fetch one entry only
			}

			//Validate the when expression
			if (ctxNode != nil) && !(xmlquery.Eval(c.yv.root, ctxNode, whenExp.exprTree)) {
				keys := []string{}
				if len(ctxNode.Parent.Attr) > 0 {
					keys = strings.Split(ctxNode.Parent.Attr[0].Value,
						modelInfo.tableInfo[tableName].redisKeyDelim)
				}

				if (len(whenExp.nodeNames) == 1) && //when in leaf
					(nodeName == whenExp.nodeNames[0]) {
					return CVLErrorInfo{
						TableName:     tableName,
						ErrCode:       CVL_SEMANTIC_ERROR,
						CVLErrDetails: cvlErrorMap[CVL_SEMANTIC_ERROR],
						Keys:          keys,
						Value:         ctxNode.FirstChild.Data,
						Field:         nodeName,
						Msg:           "When expression validation failed",
					}
				} else {
					//check if any nodes in whenExp.nodeNames
					//present in request data, when at list level
					whenNodeList := strings.Join(whenExp.nodeNames, ",") + ","
					for cNode := node.FirstChild; cNode != nil; cNode = cNode.NextSibling {
						if strings.Contains(whenNodeList, (cNode.Data + ",")) {
							return CVLErrorInfo{
								TableName:     tableName,
								ErrCode:       CVL_SEMANTIC_ERROR,
								CVLErrDetails: cvlErrorMap[CVL_SEMANTIC_ERROR],
								Keys:          keys,
								Value:         cNode.FirstChild.Data,
								Field:         cNode.Data,
								Msg:           "When expression validation failed",
							}
						}
					}
				}
			}
		}
	}

	return CVLErrorInfo{ErrCode: CVL_SUCCESS}
}

// Validate leafref
// Convert leafref to must expression
// type leafref { path "../../../ACL_TABLE/ACL_TABLE_LIST/aclname";} converts to
// "current() = ../../../ACL_TABLE/ACL_TABLE_LIST[aclname=current()]/aclname"
func (c *CVL) validateLeafRef(node *xmlquery.Node,
	tableName, key string, op cmn.CVLOperation) (r CVLErrorInfo) {
	defer func() {
		ret := &r
		CVL_LOG(INFO_API, "validateLeafRef(): table name = %s, "+
			"return value = %v", tableName, *ret)
	}()
	var errMsg string = ""

	tblName := getYangListToRedisTbl(tableName)
	if op == cmn.OP_DELETE {
		//No new node getting added so skip leafref validation
		return CVLErrorInfo{ErrCode: CVL_SUCCESS}
	}

	//Set xpath callback for retreiving dependent data
	xpath.SetDepDataClbk(c, depDataCb)

	listNode := node
	if listNode == nil || listNode.FirstChild == nil {
		return CVLErrorInfo{
			TableName:     tblName,
			Keys:          strings.Split(key, modelInfo.tableInfo[tableName].redisKeyDelim),
			ErrCode:       CVL_SEMANTIC_ERROR,
			CVLErrDetails: cvlErrorMap[CVL_SEMANTIC_ERROR],
			Msg:           "Failed to find YANG data for leafref expression validation",
		}
	}

	tblInfo := modelInfo.tableInfo[tableName]

	for nodeName, leafRefs := range tblInfo.leafRef { //for each leafref node

		//Reach to the node where leafref is present
		ctxNode := listNode.FirstChild
		for ; (ctxNode != nil) && (ctxNode.Data != nodeName); ctxNode = ctxNode.NextSibling {
		}

		if ctxNode == nil {
			//No leafref instance present, proceed to next leafref
			continue
		}

		//Check leafref for each leaf-list node
		for ; (ctxNode != nil) && (ctxNode.Data == nodeName); ctxNode = ctxNode.NextSibling {
			//Load first data for each referred table.
			//c.yv.root has all requested data merged and any depdendent
			//data needed for leafref validation should be available from this.

			leafRefSuccess := false
			nonLeafRefPresent := false //If leaf has non-leafref data type due to union
			nodeValMatchedWithLeafref := false

			ctxtVal := ""
			//Get the leaf value
			if ctxNode.FirstChild != nil {
				ctxtVal = ctxNode.FirstChild.Data
			}

			//Excute all leafref checks, multiple leafref for unions
		leafRefLoop:
			for _, leafRefPath := range leafRefs {
				if leafRefPath.path == "non-leafref" {
					//Leaf has at-least one non-leaferf data type in union
					nonLeafRefPresent = true
					continue
				}

				//Add dependent data for all referred tables
				for _, refListName := range leafRefPath.yangListNames {
					refRedisTableName := getYangListToRedisTbl(refListName)

					numKeys := len(modelInfo.tableInfo[refListName].keys)
					filter := ""
					var err error
					var tableKeys []string
					if leafRefPath.exprTree == nil { //no predicate, single key case
						//Context node used for leafref
						//Keys -> ACL_TABLE|TestACL1
						if numKeys == 1 {
							filter = refRedisTableName +
								modelInfo.tableInfo[refListName].redisKeyDelim + ctxtVal
						} else if numKeys > 1 {
							filter = CreateFindKeyExpression(refListName, map[string]string{leafRefPath.targetNodeName: ctxtVal})
						}
						tableKeys, err = c.dbAccess.Keys(filter).Result()
					} else {
						//Keys -> ACL_TABLE|*
						filter = refRedisTableName +
							modelInfo.tableInfo[refListName].redisKeyDelim + "*"
						//tableKeys, _, err = redisClient.Scan(0, filter, 1).Result()
						keysFromDb, err1 := c.dbAccess.Keys(filter).Result()
						err = err1
						// keysFromDb can be of type like "INTERFACE|Ethernet0" or
						// "INTERFACE|Ethernet0|1.1.1.1/24". So need to filter out
						// only those keys which are related to table used in leaf-ref.
						for _, keyFrmDb := range keysFromDb {
							keyStrArr := strings.SplitN(keyFrmDb, "|", 2)
							yangListName := getRedisTblToYangList(keyStrArr[0], keyStrArr[1])
							if yangListName == refListName {
								tableKeys = append(tableKeys, keyFrmDb)
							}
						}
					}

					if (err != nil) || (len(tableKeys) == 0) {
						//There must be at least one entry in the ref table
						TRACE_LOG(TRACE_SEMANTIC, "Leafref dependent data "+
							"table %s, key %s not found in Redis", refRedisTableName,
							ctxtVal)

						if leafRefPath.exprTree == nil {
							_, keyFilter := splitRedisKey(filter)
							//Check the key in request cache also
							if _, exists := c.requestCache[refRedisTableName][ctxtVal]; exists {
								//no predicate and single key is referred
								leafRefSuccess = true
								break leafRefLoop
							} else if node := c.findYangList(refListName, keyFilter); node != nil {
								leafRefSuccess = true
								break leafRefLoop
							}
						}
						continue
					} else {
						if leafRefPath.exprTree == nil {
							//Check the ref key in request cache also if it is getting deleted
							if _, exists := c.requestCache[refRedisTableName][ctxtVal]; exists {
								isReferDeletedInReqData := c.hasDeletedInReqCache(refRedisTableName, ctxtVal)

								if isReferDeletedInReqData {
									errMsg = "No instance found for '" + refRedisTableName + "[" + ctxtVal + "]'"
									leafRefSuccess = false
									break
								}
							}
							//no predicate and single key is referred
							leafRefSuccess = true
							break leafRefLoop
						}
					}

					//Now add the first data
					c.addDepYangData([]string{}, tableKeys[0],
						strings.Join(modelInfo.tableInfo[refListName].keys, "|"),
						"true", "", 0)
				}

				//Excute xpath expression for complex leafref path
				if xmlquery.Eval(c.yv.root, ctxNode, leafRefPath.exprTree) {
					leafRefSuccess = true
					break leafRefLoop
				}
			} //for loop for all leafref check for a leaf - union case

			if !leafRefSuccess && nonLeafRefPresent && (len(leafRefs) > 1) {
				//If union has mixed type with base and leafref type,
				//check if node value matched with any leafref.
				//If so non-existence of leafref in DB will be treated as failure.
				if ctxtVal != "" {
					nodeValMatchedWithLeafref = c.yp.IsLeafrefMatchedInUnion(tblInfo.module,
						fmt.Sprintf("/%s:%s/%s/%s_LIST/%s", tblInfo.modelName,
							tblInfo.modelName, tblInfo.redisTableName,
							tableName, nodeName),
						ctxtVal)
				}
			}

			if !leafRefSuccess && (!nonLeafRefPresent || nodeValMatchedWithLeafref) {
				if len(errMsg) == 0 {
					errMsg = "No instance found for '" + ctxtVal + "'"
				}
				//Return failure if none of the leafref exists
				return CVLErrorInfo{
					TableName: tblName,
					Keys: strings.Split(key,
						modelInfo.tableInfo[tableName].redisKeyDelim),
					ErrCode:          CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
					CVLErrDetails:    cvlErrorMap[CVL_SEMANTIC_DEPENDENT_DATA_MISSING],
					ErrAppTag:        "instance-required",
					ConstraintErrMsg: errMsg,
				}
			} else if !leafRefSuccess {
				TRACE_LOG(TRACE_SEMANTIC, "validateLeafRef(): "+
					"Leafref dependent data not found but leaf has "+
					"other data type in union, returning success.")
			}
		} //for each leaf-list node
	}

	return CVLErrorInfo{ErrCode: CVL_SUCCESS}
}

func (c *CVL) hasDeletedInReqCache(tblName, keyVal string) bool {
	isDeletedInReqData := false
	for i := range c.requestCache[tblName][keyVal] {
		reqData := c.requestCache[tblName][keyVal][i].ReqData
		// Only when complete delete entry is recorded in cache
		if reqData.VOp == cmn.OP_DELETE && reqData.VType == cmn.VALIDATE_NONE && len(reqData.Data) == 0 {
			isDeletedInReqData = true
			continue
		}
		// In same transaction, CREATE request also can be there
		if reqData.VOp == cmn.OP_CREATE {
			isDeletedInReqData = false
		}
	}

	return isDeletedInReqData
}

//Find which all tables (and which field) is using given (tableName/field)
// as leafref
//Use LUA script to find if table has any entry for this leafref
/*func (c *CVL) findUsedAsLeafRef(tableName, field string) []tblFieldPair {

	var tblFieldPairArr []tblFieldPair

	for tblName, tblInfo := range  modelInfo.tableInfo {
		if (tableName == tblName) {
			continue
		}
		if (len(tblInfo.leafRef) == 0) {
			continue
		}

		for fieldName, leafRefs  := range tblInfo.leafRef {
			found := false
			//Find leafref by searching table and field name
			for _, leafRef := range leafRefs {
				if (strings.Contains(leafRef.path, tableName) && strings.Contains(leafRef.path, field)) {
					tblFieldPairArr = append(tblFieldPairArr,
					tblFieldPair{tblName, fieldName})
					//Found as leafref, no need to search further
					found = true
					break
				}
			}

			if found {
				break
			}
		}
	}

	return tblFieldPairArr
}*/

// This function returns true if any entry
// in request cache is using the given entry
// getting deleted. The given entry can be found
// either in key or in hash-field.
// Example : If T1|K1 is getting deleted,
// check if T2*|K1 or T2|*:{H1: K1}
// was using T1|K1 and getting deleted
// in same session also.
func (c *CVL) checkDeleteInRequestCache(cfgData []cmn.CVLEditConfigData, currCfgData cmn.CVLEditConfigData,
	leafRef *tblFieldPair, depDataKey, keyVal string) bool {

	for _, cfgDataItem := range cfgData {
		// All cfgDataItems which have VType as VALIDATE_NONE should be
		// checked in cache
		//if cfgDataItem.VType != cmn.VALIDATE_NONE {
		//	continue
		//}
		// Skip the current cfgData
		if cfgDataItem.Key == currCfgData.Key && cmp.Equal(cfgDataItem, currCfgData) {
			continue
		}

		//If any entry is using the given entry
		//getting deleted, break immediately

		//Find in request key, case - T2*|K1
		if cfgDataItem.Key == depDataKey && cfgDataItem.VOp == cmn.OP_DELETE && len(cfgDataItem.Data) == 0 {
			return true
		}

		//Find in request hash-field, case - T2*|K2:{H1: K1}
		isLeafList := false
		val, exists := cfgDataItem.Data[leafRef.field]
		if !exists {
			// Leaf-lists field names are suffixed by "@".
			val, exists = cfgDataItem.Data[leafRef.field+"@"]
			isLeafList = exists
		}
		// For delete cases, val sent is empty.
		// For update cases, val will be different from keyVal
		if exists {
			if !isLeafList && ((cfgDataItem.VOp == cmn.OP_DELETE && val == "") || (cfgDataItem.VOp != cmn.OP_DELETE && val != keyVal)) {
				return true
			}
			if isLeafList {
				entryFound := false
				for _, v := range strings.Split(val, ",") {
					if v == keyVal {
						entryFound = true
						break
					}
				}
				if !entryFound {
					return true
				}
			}
		}
	}

	return false
}

// checkDepDataCompatible This function evaluates relationship between two table
// keys based on leafref
func (c *CVL) checkDepDataCompatible(tblName, key, reftblName, refTblKey, leafRefField string, depEntry map[string]string) bool {
	CVL_LOG(INFO_DEBUG, "checkDepDataCompatible--> TargetTbl: %s[%s] referred by refTbl: %s[%s] through field: %s", tblName, key, reftblName, refTblKey, leafRefField)

	// Key compatibility to be checked only if no. of keys are more than 1
	if len(modelInfo.tableInfo[tblName].keys) <= 1 {
		return true
	}

	// Fetch key value pair for both current table and ref table
	tblKeysKVP := getRedisToYangKeys(tblName, key)
	refTblKeysKVP := getRedisToYangKeys(reftblName, refTblKey)

	// Determine the leafref
	leafRef := getLeafRefInfo(reftblName, leafRefField, tblName)

	// If depEntry map is Empty, it means leafRefField is one of key of RefTable.
	// So RefTblKey should equal to targetTbl key plus additional 1 or more key
	// For ex. BGP_NEIGHBOR|Vrf1|2114::2 referred by BGP_NEIGHBOR_AF|Vrf1|2114::2|ipv4_unicast
	if len(depEntry) == 0 {
		// TODO Need to revisit. For now assumed that orderof keys are same.
		return strings.Contains(refTblKey, key)
	} else {
		// leafRefField is NOT key but a normal leaf node
		if _, exists := depEntry[leafRefField]; exists {
			// Compare keys of table and refTable
			//LeafRef: /sonic-bgp-peergroup:sonic-bgp-peergroup/sonic-bgp-peergroup:BGP_PEER_GROUP/sonic-bgp-peergroup:BGP_PEER_GROUP_LIST[sonic-bgp-peergroup:vrf_name=current()/../vrf_name]/sonic-bgp-peergroup:peer_group_name
			if leafRef != nil && leafRef.exprTree != nil {
				leafrefExpr := leafRef.exprTree.String()
				if len(leafrefExpr) > 0 {
					// TODO Assumed that predicate will have one expression without any 'and'/'or' operators
					// predicate looks like [sonic-bgp-peergroup:vrf_name=current()/../vrf_name]
					predicate := leafrefExpr[strings.Index(leafrefExpr, "[")+1 : strings.Index(leafrefExpr, "]")]
					if strings.Contains(predicate, "=") {
						// target tbl key is left of '='
						leafrefTargetTblkey := predicate[:strings.Index(predicate, "=")]
						if strings.Contains(leafrefTargetTblkey, ":") {
							leafrefTargetTblkey = leafrefTargetTblkey[strings.Index(leafrefTargetTblkey, ":")+1:]
						}
						// current tbl key is right of '=' after last '/'('current()/../vrf_name")
						leafrefCurrentTblkey := predicate[strings.LastIndex(predicate, "/")+1:]

						var leafrefTargetKeyVal, leafrefCurrentKeyVal string
						for _, kvp := range refTblKeysKVP {
							if kvp.key == leafrefTargetTblkey {
								leafrefTargetKeyVal = kvp.values[0]
							}
						}

						for _, kvp := range tblKeysKVP {
							if kvp.key == leafrefCurrentTblkey {
								leafrefCurrentKeyVal = kvp.values[0]
							}
						}

						return leafrefTargetKeyVal == leafrefCurrentKeyVal
					}
				}
			}
		}
	}

	return true
}

// Check delete constraint for leafref if key/field is deleted
func (c *CVL) checkDeleteConstraint(cfgData []cmn.CVLEditConfigData, currCfgData cmn.CVLEditConfigData,
	tableName, keyVal, field string) CVLRetCode {

	yangTblName := getRedisTblToYangList(tableName, keyVal)
	// Creating a map of leaf-ref referred tableName and associated fields array
	refTableFieldsMap := map[string][]string{}
	for _, leafRef := range modelInfo.tableInfo[yangTblName].refFromTables {
		// If field is getting deleted, then collect only those leaf-refs that
		// refers that field
		if (field != "") && ((field != leafRef.field) || (field != leafRef.field+"@")) {
			continue
		}

		if _, ok := refTableFieldsMap[leafRef.tableName]; !ok {
			refTableFieldsMap[leafRef.tableName] = make([]string, 0)
		}
		refFieldsArr := refTableFieldsMap[leafRef.tableName]
		refFieldsArr = append(refFieldsArr, leafRef.field)
		refTableFieldsMap[leafRef.tableName] = refFieldsArr
	}

	if len(refTableFieldsMap) == 0 {
		return CVL_SUCCESS
	}

	//The entry getting deleted might have been referred from multiple tables
	//Return failure if at-least one table is using this entry

	// Retrieve all dependent DB entries referring the given entry tableName and keyVal
	redisKeyForDepData := tableName + "|" + keyVal
	depDataArr := c.GetDepDataForDelete(redisKeyForDepData)
	TRACE_LOG(TRACE_SEMANTIC, "checkDeleteConstraint--> All Data for deletion: %v", depDataArr)

	// Iterate through dependent DB entries and check if it already present in delete request cache
	for _, depData := range depDataArr {
		if len(depData.Entry) > 0 && depData.RefKey == redisKeyForDepData {
			TRACE_LOG(TRACE_SEMANTIC, "checkDeleteConstraint--> DepData: %v", depData.Entry)
			for depEntkey := range depData.Entry {
				depEntkeyList := strings.SplitN(depEntkey, "|", 2)
				refTblName := depEntkeyList[0]
				refTblKey := depEntkeyList[1]
				refYangTblName := getRedisTblToYangList(refTblName, refTblKey)

				var isRefTblKeyNotCompatible bool
				var isEntryInRequestCache bool
				leafRefFieldsArr := refTableFieldsMap[refYangTblName]
				// Verify each dependent data with help of its associated leaf-ref
				for _, leafRefField := range leafRefFieldsArr {
					TRACE_LOG(TRACE_SEMANTIC, "checkDeleteConstraint--> Checking delete constraint for leafRef %s/%s", refYangTblName, leafRefField)
					// Key compatibility to be checked only if no. of keys are more than 1
					// because dep data for key like "BGP_PEER_GROUP|Vrf1|PG1" can be returned as
					// BGP_NEIGHBOR|Vrf1|11.1.1.1 or BGP_NEIGHBOR|default|11.1.1.1 or BGP_NEIGHBOR|Vrf2|11.1.1.1
					// So we have to discard imcompatible dep data
					if !c.checkDepDataCompatible(yangTblName, keyVal, refYangTblName, refTblKey, leafRefField, depData.Entry[depEntkey]) {
						isRefTblKeyNotCompatible = true
						TRACE_LOG(TRACE_SEMANTIC, "checkDeleteConstraint--> %s is NOT compatible with %s", redisKeyForDepData, depEntkey)
						break
					}
					tempLeafRef := tblFieldPair{refTblName, leafRefField}
					if c.checkDeleteInRequestCache(cfgData, currCfgData, &tempLeafRef, depEntkey, keyVal) {
						isEntryInRequestCache = true
						break
					}
				}

				if isRefTblKeyNotCompatible {
					continue
				}

				if isEntryInRequestCache {
					// Entry already in delete request cache, proceed for next dep data
					continue
				} else {
					CVL_LOG(WARNING, "Delete will violate the constraint as entry %s is referred in %v", redisKeyForDepData, depEntkey)
					return CVL_SEMANTIC_ERROR
				}
			}
		}
	}

	return CVL_SUCCESS
}

// Validate external dependency using leafref, must and when expression
func (c *CVL) validateSemantics(node *xmlquery.Node,
	yangListName, key string,
	cfgData *cmn.CVLEditConfigData) (r CVLErrorInfo) {

	//Mark the list entries from DB if OP_DELETE operation when complete list delete requested
	if (node != nil) && (cfgData.VOp == cmn.OP_DELETE) && (len(cfgData.Data) == 0) {
		addAttrNode(node, "db", "")
	}

	//Check all leafref
	if errObj := c.validateLeafRef(node, yangListName, key, cfgData.VOp); errObj.ErrCode != CVL_SUCCESS {
		return errObj
	}

	//Validate when expression
	if errObj := c.validateWhenExp(node, yangListName, key, cfgData.VOp); errObj.ErrCode != CVL_SUCCESS {
		return errObj
	}

	//Validate must expression
	if cfgData.VOp == cmn.OP_DELETE {
		if len(cfgData.Data) > 0 {
			// Delete leaf nodes from tree. This ensures validateMustExp will
			// skip all must expressions defined for deleted nodes; and other
			// must expressions get correct context.
			c.deleteLeafNodes(node, cfgData.Data)
		}
	}

	if errObj := c.validateMustExp(node, yangListName, key, cfgData.VOp); errObj.ErrCode != CVL_SUCCESS {
		return errObj
	}

	if cfgData.VOp == cmn.OP_DELETE && len(cfgData.Data) == 0 {
		for _, r := range c.requestCache[yangListName][key] {
			r.YangData = nil
		}
		listNodeParent := node.Parent
		if listNodeParent != nil {
			for childNode := listNodeParent.FirstChild; childNode != nil; {
				tmpNextNode := childNode.NextSibling
				if len(childNode.Attr) > 0 {
					if cmn.KeyMatch(childNode.Attr[0].Value, key) {
						c.detachNode(childNode)
					}
				}
				childNode = tmpNextNode
			}
			if listNodeParent.FirstChild == nil {
				c.detachNode(listNodeParent)
			}
		}
	}

	return CVLErrorInfo{ErrCode: CVL_SUCCESS}
}

// Validate external dependency using leafref, must and when expression
func (c *CVL) validateCfgSemantics(root *xmlquery.Node) (r CVLErrorInfo) {
	if strings.HasSuffix(root.Data, "_LIST") {
		if len(root.Attr) == 0 {
			return CVLErrorInfo{ErrCode: CVL_SUCCESS}
		}
		yangListName := root.Data[:len(root.Data)-5]
		return c.validateSemantics(root, yangListName, root.Attr[0].Value,
			&cmn.CVLEditConfigData{VType: cmn.VALIDATE_NONE, VOp: cmn.OP_NONE})
	}

	//Traverse through all list instances and validate
	ret := CVLErrorInfo{}
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		ret = c.validateCfgSemantics(node)
		if ret.ErrCode != CVL_SUCCESS {
			break
		}
	}

	return ret
}

// For Replace operation DB layer sends update request and delete fields request
// For semantic validation, remove fields provided in delete request from
// update request.
func (c *CVL) updateYangTreeForReplaceOp(node *xmlquery.Node, cfgData []cmn.CVLEditConfigData) {
	for _, cfgDataItem := range cfgData {
		if cmn.VALIDATE_ALL != cfgDataItem.VType {
			continue
		}

		if !cfgDataItem.ReplaceOp {
			return
		}

		if cmn.OP_DELETE == cfgDataItem.VOp && len(cfgDataItem.Data) > 0 {
			c.deleteLeafNodes(node, cfgDataItem.Data)
		}
	}
}
