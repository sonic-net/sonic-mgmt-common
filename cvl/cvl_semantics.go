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
	"strings"
	"github.com/antchfx/xmlquery"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

//YValidator YANG Validator used for external semantic
//validation including custom/platform validation
type YValidator struct {
	root *xmlquery.Node    //Top evel root for data
	current *xmlquery.Node //Current position
}

//Check delete constraint for leafref if key/field is deleted
func (c *CVL) checkDeleteConstraint(cfgData []CVLEditConfigData,
			tableName, keyVal, field string) CVLRetCode {
	var leafRefs []tblFieldPair
	if (field != "") {
		//Leaf or field is getting deleted
		leafRefs = c.findUsedAsLeafRef(tableName, field)
	} else {
		//Entire entry is getting deleted
		leafRefs = c.findUsedAsLeafRef(tableName, modelInfo.tableInfo[tableName].keys[0])
	}

	//The entry getting deleted might have been referred from multiple tables
	//Return failure if at-least one table is using this entry
	for _, leafRef := range leafRefs {
		TRACE_LOG(INFO_API, (TRACE_DELETE | TRACE_SEMANTIC), "Checking delete constraint for leafRef %s/%s", leafRef.tableName, leafRef.field)
		//Check in dependent data first, if the referred entry is already deleted
		leafRefDeleted := false
		for _, cfgDataItem := range cfgData {
			if (cfgDataItem.VType == VALIDATE_NONE) &&
			(cfgDataItem.VOp == OP_DELETE ) &&
			(strings.HasPrefix(cfgDataItem.Key, (leafRef.tableName + modelInfo.tableInfo[leafRef.tableName].redisKeyDelim + keyVal + modelInfo.tableInfo[leafRef.tableName].redisKeyDelim))) {
				//Currently, checking for one entry is being deleted in same session
				//We should check for all entries
				leafRefDeleted = true
				break
			}
		}

		if (leafRefDeleted == true) {
			continue //check next leafref
		}

		//Else, check if any referred enrty is present in DB
		var nokey []string
		refKeyVal, err := luaScripts["find_key"].Run(redisClient, nokey, leafRef.tableName,
		modelInfo.tableInfo[leafRef.tableName].redisKeyDelim, leafRef.field, keyVal).Result()
		if (err == nil &&  refKeyVal != "") {
			CVL_LOG(ERROR, "Delete will violate the constraint as entry %s is referred in %s", tableName, refKeyVal)

			return CVL_SEMANTIC_ERROR
		}
	}


	return CVL_SUCCESS
}

//Perform semantic checks 
func (c *CVL) validateSemantics(data *yparser.YParserNode, appDepData *yparser.YParserNode) (CVLErrorInfo, CVLRetCode) {
	var cvlErrObj CVLErrorInfo
	
	if (SkipSemanticValidation() == true) {
		return cvlErrObj, CVL_SUCCESS
	}

	//Get dependent data from 
	depData := c.fetchDataToTmpCache() //fetch data to temp cache for temporary validation

	if (Tracing == true) {
		TRACE_LOG(INFO_API, TRACE_SEMANTIC, "Validating semantics data=%s\n depData =%s\n, appDepData=%s\n....", c.yp.NodeDump(data), c.yp.NodeDump(depData), c.yp.NodeDump(appDepData))
	}

	if errObj := c.yp.ValidateSemantics(data, depData, appDepData); errObj.ErrCode != yparser.YP_SUCCESS {

		retCode := CVLRetCode(errObj.ErrCode)

		cvlErrObj =  CVLErrorInfo {
			TableName : errObj.TableName,
		        ErrCode   : CVLRetCode(errObj.ErrCode),		
			CVLErrDetails : cvlErrorMap[retCode], 
			Keys      : errObj.Keys,
			Value     : errObj.Value,
			Field     : errObj.Field,
			Msg       : errObj.Msg,
			ConstraintErrMsg : errObj.ErrTxt,
			ErrAppTag	: errObj.ErrAppTag,
		}



		return  cvlErrObj, retCode
	}

	return cvlErrObj ,CVL_SUCCESS
}

