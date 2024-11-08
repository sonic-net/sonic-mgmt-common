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
	"fmt"
	"reflect"

	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	custv "github.com/Azure/sonic-mgmt-common/cvl/custom_validation"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	"github.com/go-redis/redis/v7"
	"github.com/google/go-cmp/cmp"
	toposort "github.com/philopon/go-toposort"

	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
	"github.com/antchfx/xmlquery"
)

var cvlErrorMap = map[CVLRetCode]string{
	CVL_SUCCESS:                         "Config Validation Success",
	CVL_SYNTAX_ERROR:                    "Config Validation Syntax Error",
	CVL_SEMANTIC_ERROR:                  "Config Validation Semantic Error",
	CVL_SYNTAX_MISSING_FIELD:            "Required Field is Missing",
	CVL_SYNTAX_INVALID_FIELD:            "Invalid Field Received",
	CVL_SYNTAX_INVALID_INPUT_DATA:       "Invalid Input Data Received",
	CVL_SYNTAX_MULTIPLE_INSTANCE:        "Multiple Field Instances Received",
	CVL_SYNTAX_DUPLICATE:                "Duplicate Instances Received",
	CVL_SYNTAX_ENUM_INVALID:             "Invalid Enum Value Received",
	CVL_SYNTAX_ENUM_INVALID_NAME:        "Invalid Enum Value Received",
	CVL_SYNTAX_ENUM_WHITESPACE:          "Enum name with leading/trailing whitespaces Received",
	CVL_SYNTAX_OUT_OF_RANGE:             "Value out of range/length/pattern (data)",
	CVL_SYNTAX_MINIMUM_INVALID:          "min-elements constraint not honored",
	CVL_SYNTAX_MAXIMUM_INVALID:          "max-elements constraint not honored",
	CVL_SEMANTIC_DEPENDENT_DATA_MISSING: "Dependent Data is missing",
	CVL_SEMANTIC_MANDATORY_DATA_MISSING: "Mandatory Data is missing",
	CVL_SEMANTIC_KEY_ALREADY_EXIST:      "Key already existing.",
	CVL_SEMANTIC_KEY_NOT_EXIST:          "Key is missing.",
	CVL_SEMANTIC_KEY_DUPLICATE:          "Duplicate key received",
	CVL_SEMANTIC_KEY_INVALID:            "Invalid Key Received",
	CVL_INTERNAL_UNKNOWN:                "Internal Unknown Error",
	CVL_ERROR:                           "Generic Error",
	CVL_NOT_IMPLEMENTED:                 "Error Not Implemented",
	CVL_FAILURE:                         "Generic Failure",
}

// CVLRetCode CVL Error codes
type CVLRetCode int

const (
	CVL_SUCCESS CVLRetCode = iota
	CVL_ERROR
	CVL_NOT_IMPLEMENTED
	CVL_INTERNAL_UNKNOWN
	CVL_FAILURE
	CVL_SYNTAX_ERROR                    = CVLRetCode(yparser.YP_SYNTAX_ERROR)
	CVL_SEMANTIC_ERROR                  = CVLRetCode(yparser.YP_SEMANTIC_ERROR)
	CVL_SYNTAX_MISSING_FIELD            = CVLRetCode(yparser.YP_SYNTAX_MISSING_FIELD)
	CVL_SYNTAX_INVALID_FIELD            = CVLRetCode(yparser.YP_SYNTAX_INVALID_FIELD)            /* Invalid Field  */
	CVL_SYNTAX_INVALID_INPUT_DATA       = CVLRetCode(yparser.YP_SYNTAX_INVALID_INPUT_DATA)       /*Invalid Input Data */
	CVL_SYNTAX_MULTIPLE_INSTANCE        = CVLRetCode(yparser.YP_SYNTAX_MULTIPLE_INSTANCE)        /* Multiple Field Instances */
	CVL_SYNTAX_DUPLICATE                = CVLRetCode(yparser.YP_SYNTAX_DUPLICATE)                /* Duplicate Fields  */
	CVL_SYNTAX_ENUM_INVALID             = CVLRetCode(yparser.YP_SYNTAX_ENUM_INVALID)             /* Invalid enum value */
	CVL_SYNTAX_ENUM_INVALID_NAME        = CVLRetCode(yparser.YP_SYNTAX_ENUM_INVALID_NAME)        /* Invalid enum name  */
	CVL_SYNTAX_ENUM_WHITESPACE          = CVLRetCode(yparser.YP_SYNTAX_ENUM_WHITESPACE)          /* Enum name with leading/trailing whitespaces */
	CVL_SYNTAX_OUT_OF_RANGE             = CVLRetCode(yparser.YP_SYNTAX_OUT_OF_RANGE)             /* Value out of range/length/pattern (data) */
	CVL_SYNTAX_MINIMUM_INVALID          = CVLRetCode(yparser.YP_SYNTAX_MINIMUM_INVALID)          /* min-elements constraint not honored  */
	CVL_SYNTAX_MAXIMUM_INVALID          = CVLRetCode(yparser.YP_SYNTAX_MAXIMUM_INVALID)          /* max-elements constraint not honored */
	CVL_SEMANTIC_DEPENDENT_DATA_MISSING = CVLRetCode(yparser.YP_SEMANTIC_DEPENDENT_DATA_MISSING) /* Dependent Data is missing */
	CVL_SEMANTIC_MANDATORY_DATA_MISSING = CVLRetCode(yparser.YP_SEMANTIC_MANDATORY_DATA_MISSING) /* Mandatory Data is missing */
	CVL_SEMANTIC_KEY_ALREADY_EXIST      = CVLRetCode(yparser.YP_SEMANTIC_KEY_ALREADY_EXIST)      /* Key already existing. */
	CVL_SEMANTIC_KEY_NOT_EXIST          = CVLRetCode(yparser.YP_SEMANTIC_KEY_NOT_EXIST)          /* Key is missing. */
	CVL_SEMANTIC_KEY_DUPLICATE          = CVLRetCode(yparser.YP_SEMANTIC_KEY_DUPLICATE)          /* Duplicate key. */
	CVL_SEMANTIC_KEY_INVALID            = CVLRetCode(yparser.YP_SEMANTIC_KEY_INVALID)
)

// ValidationTimeStats CVL validations stats
// Maintain time stats for call to ValidateEditConfig().
// Hits : Total number of times ValidateEditConfig() called
// Time : Total time spent in ValidateEditConfig()
// Peak : Highest time spent in ValidateEditConfig()
type ValidationTimeStats struct {
	Hits uint
	Time time.Duration
	Peak time.Duration
}

// CVLDepDataForDelete Structure for dependent entry to be deleted
type CVLDepDataForDelete struct {
	RefKey string                       //Ref Key which is getting deleted
	Entry  map[string]map[string]string //Entry or field which should be deleted as a result
}

// Global data structure for maintaining validation stats
var cfgValidationStats ValidationTimeStats
var statsMutex *sync.Mutex

func Initialize() CVLRetCode {
	if cvlInitialized {
		//CVL has already been initialized
		return CVL_SUCCESS
	}

	//Initialize redis Client
	redisClient = NewDbClient("CONFIG_DB")

	if redisClient == nil {
		CVL_LOG(FATAL, "Unable to connect to Redis Config DB Server")
		return CVL_ERROR
	}

	//Load lua script into redis
	luaScripts = make(map[string]*redis.Script)
	loadLuaScript(luaScripts)

	yparser.Initialize()

	modelInfo.modelNs = make(map[string]*modelNamespace)                //redis table to model name
	modelInfo.tableInfo = make(map[string]*modelTableInfo)              //model namespace
	modelInfo.allKeyDelims = make(map[string]bool)                      //all key delimiter
	modelInfo.redisTableToYangList = make(map[string][]string)          //Redis table to Yang list map
	modelInfo.redisTableToListKeys = make(map[string]map[string]string) //Redis table to Yang List key map
	dbNameToDbNum = map[string]uint8{"APPL_DB": APPL_DB, "CONFIG_DB": CONFIG_DB}

	// Load all YIN schema files
	if retCode := loadSchemaFiles(); retCode != CVL_SUCCESS {
		return retCode
	}

	//Compile leafref path
	compileLeafRefPath()

	//Compile all must exps
	compileMustExps()

	//Compile all when exps
	compileWhenExps()

	//Add all table names to be fetched to validate 'must' expression
	addTableNamesForMustExp()

	//Build reverse leafref info i.e. which table/field uses one table through leafref
	buildRefTableInfo()

	//Build table dependencies based on 'dependent-on' extension
	buildTableDependencies()

	cvlInitialized = true

	return CVL_SUCCESS
}

func Finish() {
	yparser.Finish()
}

func ValidationSessOpen(dbAccess cmn.DBAccess) (*CVL, CVLRetCode) {
	cvl := &CVL{}
	cvl.tmpDbCache = make(map[string]interface{})
	cvl.requestCache = make(map[string]map[string][]*cmn.RequestCacheType)
	cvl.depDataCache = make(map[string]interface{})
	cvl.maxTableElem = make(map[string]int)
	cvl.yp = &yparser.YParser{}
	cvl.yv = &YValidator{}
	cvl.yv.root = &xmlquery.Node{Type: xmlquery.DocumentNode}
	cvl.custvCache.Data = make(map[string]interface{})
	cvl.custvCache.Hint = make(map[string]interface{})
	cvl.dbAccess = dbAccess

	if cvl == nil || cvl.yp == nil {
		return nil, CVL_FAILURE
	}

	return cvl, CVL_SUCCESS
}

func ValidationSessClose(c *CVL) CVLRetCode {
	c.yp.DestroyCache()
	c = nil

	return CVL_SUCCESS
}

func (c *CVL) ValidateStartupConfig(jsonData string) CVLRetCode {
	//Check config data syntax
	//Finally validate
	return CVL_NOT_IMPLEMENTED
}

// ValidateIncrementalConfig Steps:
//
//	Check config data syntax
//	Fetch the depedent data
//	Merge config and dependent data
//	Finally validate
func (c *CVL) ValidateIncrementalConfig(jsonData string) CVLRetCode {
	c.clearTmpDbCache()
	var v interface{}

	b := []byte(jsonData)
	if err := json.Unmarshal(b, &v); err != nil {
		return CVL_SYNTAX_ERROR
	}

	var dataMap map[string]interface{} = v.(map[string]interface{})

	root, _ := c.translateToYang(&dataMap, true)
	defer c.yp.FreeNode(root)
	if root == nil {
		return CVL_SYNTAX_ERROR

	}

	errObj := c.yp.ValidateSyntax(root, nil)
	if yparser.YP_SUCCESS != errObj.ErrCode {
		return CVL_FAILURE
	}

	//Add and fetch entries if already exists in Redis
	for tableName, data := range dataMap {
		for key := range data.(map[string]interface{}) {
			c.addTableEntryToCache(tableName, key)
		}
	}

	existingData := c.fetchDataToTmpCache()

	//Merge existing data for update syntax or checking duplicate entries
	if existingData != nil {
		if _, errObj = c.yp.MergeSubtree(root, existingData); errObj.ErrCode != yparser.YP_SUCCESS {
			return CVL_ERROR
		}
	}

	//Clear cache
	c.clearTmpDbCache()

	//Perform validation
	if cvlErrObj := c.validateCfgSemantics(c.yv.root); cvlErrObj.ErrCode != CVL_SUCCESS {
		return cvlErrObj.ErrCode
	}

	return CVL_SUCCESS
}

// ValidateConfig Validate data for operation
func (c *CVL) ValidateConfig(jsonData string) CVLRetCode {
	c.clearTmpDbCache()
	var v interface{}

	b := []byte(jsonData)
	if err := json.Unmarshal(b, &v); err == nil {
		var value map[string]interface{} = v.(map[string]interface{})
		root, _ := c.translateToYang(&value, true)
		defer c.yp.FreeNode(root)

		if root == nil {
			return CVL_FAILURE

		}

		if c.validate(root) != CVL_SUCCESS {
			return CVL_FAILURE
		}

	}

	return CVL_SUCCESS
}

// ValidateEditConfig Validate config data based on edit operation
func (c *CVL) ValidateEditConfig(cfgData []cmn.CVLEditConfigData) (cvlErr CVLErrorInfo, ret CVLRetCode) {
	if c.dbAccess == nil {
		CVL_LOG(FATAL, "ValidateEditConfig: DB Access object is Nil")
		panic(c.dbAccess)
	}

	ts := time.Now()

	defer func() {
		if cvlErr.ErrCode != CVL_SUCCESS {
			CVL_LOG(WARNING, "ValidateEditConfig() failed: %+v", cvlErr)
			// On CVL failure, log config data
			CVL_LOG(WARNING, "Config Data: %v", cfgData)
		}
		//Update validation time stats
		updateValidationTimeStats(time.Since(ts))
		c.clearTmpDbCache()
		c.yv.root = &xmlquery.Node{Type: xmlquery.DocumentNode}
		c.depDataCache = make(DepDataCacheType)
		c.yp.DestroyCache()
	}()

	var cvlErrObj CVLErrorInfo
	checkSyntax := true

	caller := ""
	if IsTraceSet() {
		pc := make([]uintptr, 10)
		runtime.Callers(2, pc)
		f := runtime.FuncForPC(pc[0])
		caller = f.Name()
	}

	CVL_LOG(INFO_DEBUG, "ValidateEditConfig() called from %s() : %v", caller, cfgData)

	if SkipValidation() {
		CVL_LOG(INFO_TRACE, "Skipping CVL validation.")
		return cvlErrObj, CVL_SUCCESS
	}

	//Type cast to custom validation cfg data
	sliceHeader := *(*reflect.SliceHeader)(unsafe.Pointer(&cfgData))
	custvCfg := *(*[]custv.CVLEditConfigData)(unsafe.Pointer(&sliceHeader))

	c.clearTmpDbCache()
	//c.yv.root.FirstChild = nil
	//c.yv.root.LastChild = nil

	//Step 1: Get requested data first
	//add all dependent data to be fetched from Redis
	requestedData := make(map[string]interface{})

	cfgDataLen := len(cfgData)
	for i := 0; i < cfgDataLen; i++ {
		if cmn.VALIDATE_ALL != cfgData[i].VType {
			continue
		}

		//Add config data item to be validated
		tbl, key := c.addCfgDataItem(&requestedData, cfgData[i])

		//Add to request cache
		reqTbl, exists := c.requestCache[tbl]
		if !exists {
			//Create new table key data
			reqTbl = make(map[string][]*cmn.RequestCacheType)
		}
		cfgDataItemArr := reqTbl[key]
		cfgDataItemArr = append(cfgDataItemArr, &cmn.RequestCacheType{cfgData[i], nil})
		reqTbl[key] = cfgDataItemArr
		c.requestCache[tbl] = reqTbl

		//Invalid table name or invalid key separator
		if key == "" {
			cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
			cvlErrObj.Msg = "Invalid table or key for " + cfgData[i].Key
			cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
			return cvlErrObj, CVL_SYNTAX_ERROR
		}

		switch cfgData[i].VOp {
		case cmn.OP_CREATE:
			//Check max-element constraint
			if ret := c.checkMaxElemConstraint(cmn.OP_CREATE, tbl, key); ret != CVL_SUCCESS {
				cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
				cvlErrObj.TableName = tbl
				cvlErrObj.Keys = splitKeyComponents(tbl, key)
				cvlErrObj.ErrAppTag = "too-many-elements"
				cvlErrObj.Msg = "Max elements limit reached"
				cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
				cvlErrObj.ConstraintErrMsg = fmt.Sprintf("Max elements limit %v reached",
					modelInfo.tableInfo[tbl].redisTableSize)

				return cvlErrObj, CVL_SYNTAX_ERROR
			}

		case cmn.OP_UPDATE:
			//Get the existing data from Redis to cache, so that final
			//validation can be done after merging this dependent data
			c.addTableEntryToCache(tbl, key)

		case cmn.OP_DELETE:
			if len(cfgData[i].Data) > 0 {
				//Check constraints for deleting field(s)
				for field := range cfgData[i].Data {
					if c.checkDeleteConstraint(cfgData, cfgData[i], tbl, key, field) != CVL_SUCCESS {
						cvlErrObj.ErrCode = CVL_SEMANTIC_ERROR
						cvlErrObj.TableName = tbl
						cvlErrObj.Keys = splitKeyComponents(tbl, key)
						cvlErrObj.Field = field
						cvlErrObj.Msg = "Validation failed for Delete operation, given instance is in use"
						cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
						cvlErrObj.ErrAppTag = "instance-in-use"
						cvlErrObj.ConstraintErrMsg = cvlErrObj.Msg
						return cvlErrObj, CVL_SEMANTIC_ERROR
					}

					// Cannot delete leaf-list field having min-elements > 0
					if isLeaflistHasMinElems(tbl, field) {
						cvlErrObj.ErrCode = CVL_SYNTAX_MINIMUM_INVALID
						cvlErrObj.Msg = "Mandatory field getting deleted"
						cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
						cvlErrObj.TableName = tbl
						cvlErrObj.Keys = splitKeyComponents(tbl, key)
						cvlErrObj.Field = strings.TrimSuffix(field, "@") // To match libyang error
						return cvlErrObj, cvlErrObj.ErrCode
					}

					//Check mandatory node deletion
					if len(field) != 0 && isMandatoryTrueNode(getRedisTblToYangList(tbl, key), field) {
						cvlErrObj.ErrCode = CVL_SEMANTIC_ERROR
						cvlErrObj.Msg = "Mandatory field getting deleted"
						cvlErrObj.TableName = tbl
						cvlErrObj.Keys = splitKeyComponents(tbl, key)
						cvlErrObj.Field = field
						cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
						cvlErrObj.ErrAppTag = "mandatory-field-delete"
						cvlErrObj.ConstraintErrMsg = cvlErrObj.Msg
						return cvlErrObj, CVL_SEMANTIC_ERROR
					}
				}
			} else {
				//Entire entry to be deleted

				//Update max-elements count
				c.checkMaxElemConstraint(cmn.OP_DELETE, tbl, key)

				//Now check delete constraints
				if c.checkDeleteConstraint(cfgData, cfgData[i], tbl, key, "") != CVL_SUCCESS {
					cvlErrObj.ErrCode = CVL_SEMANTIC_ERROR
					cvlErrObj.TableName = tbl
					cvlErrObj.Keys = splitKeyComponents(tbl, key)
					cvlErrObj.Msg = "Validation failed for Delete operation, given instance is in use"
					cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
					cvlErrObj.ErrAppTag = "instance-in-use"
					cvlErrObj.ConstraintErrMsg = cvlErrObj.Msg
					return cvlErrObj, CVL_SEMANTIC_ERROR
				}
				checkSyntax = false
			}

			c.addTableEntryToCache(tbl, key)
		}
	}

	if IsTraceSet() {
		//Only for tracing
		jsonData := ""

		jsonDataBytes, err := json.Marshal(requestedData)
		if err == nil {
			jsonData = string(jsonDataBytes)
		} else {
			cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
			cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
			return cvlErrObj, CVL_SYNTAX_ERROR
		}

		TRACE_LOG(TRACE_LIBYANG, "Requested JSON Data = [%s]\n", jsonData)
	}

	//Step 2 : Perform syntax validation only
	yang, errN := c.translateToYang(&requestedData, checkSyntax)
	defer c.yp.FreeNode(yang)

	if errN.ErrCode == CVL_SUCCESS {
		//Get dependent data from Redis
		depData := c.fetchDataToTmpCache() //fetch data to temp cache for temporary validation
		if checkSyntax {
			if cvlErrObj, cvlRetCode := c.validateSyntax(yang, depData); cvlRetCode != CVL_SUCCESS {
				return cvlErrObj, cvlRetCode
			}
		}
	} else {
		return errN, errN.ErrCode
	}

	//Step 3 : Check keys and perform semantics validation
	for i := 0; i < cfgDataLen; i++ {

		if cfgData[i].VType != cmn.VALIDATE_ALL && cfgData[i].VType != cmn.VALIDATE_SEMANTICS {
			continue
		}

		tbl, key := splitRedisKey(cfgData[i].Key)

		//Step 3.1 : Check keys
		switch cfgData[i].VOp {
		case cmn.OP_CREATE:
			//Check key should not already exist
			n, err1 := c.dbAccess.Exists(cfgData[i].Key).Result()
			if err1 == nil && n > 0 {
				CVL_LOG(WARNING, "\nValidateEditConfig(): Key = %s already exists", cfgData[i].Key)
				cvlErrObj.ErrCode = CVL_SEMANTIC_KEY_ALREADY_EXIST
				cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
				cvlErrObj.TableName = tbl
				cvlErrObj.Keys = splitKeyComponents(tbl, key)
				return cvlErrObj, CVL_SEMANTIC_KEY_ALREADY_EXIST
			} else if err1 != nil {
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_CREATE - Key = %s ", cfgData[i].Key)
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_CREATE - Err = %v ", err1)
			}

			c.yp.SetOperation("CREATE")

		case cmn.OP_UPDATE:
			n, err1 := c.dbAccess.Exists(cfgData[i].Key).Result()
			if err1 != nil || n == 0 { //key must exists
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_UPDATE - Key = %s does not exist", cfgData[i].Key)
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_UPDATE - Err = %v ", err1)
				cvlErrObj.ErrCode = CVL_SEMANTIC_KEY_NOT_EXIST
				cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
				cvlErrObj.TableName = tbl
				cvlErrObj.Keys = splitKeyComponents(tbl, key)
				return cvlErrObj, CVL_SEMANTIC_KEY_NOT_EXIST
			}

			// Skip validation if UPDATE is received with only NULL field
			if _, exists := cfgData[i].Data["NULL"]; exists && len(cfgData[i].Data) == 1 {
				c.markCfgDataValidated(&cfgData[i], tbl, key)
				continue
			}

			c.yp.SetOperation("UPDATE")

		case cmn.OP_DELETE:
			n, err1 := c.dbAccess.Exists(cfgData[i].Key).Result()
			if err1 != nil || n == 0 { //key must exists
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_DELETE - Key = %s does not exist", cfgData[i].Key)
				CVL_LOG(WARNING, "\nValidateEditConfig(): OP_DELETE - Err = %v ", err1)
				cvlErrObj.ErrCode = CVL_SEMANTIC_KEY_NOT_EXIST
				cvlErrObj.CVLErrDetails = cvlErrorMap[cvlErrObj.ErrCode]
				cvlErrObj.TableName = tbl
				cvlErrObj.Keys = splitKeyComponents(tbl, key)
				return cvlErrObj, CVL_SEMANTIC_KEY_NOT_EXIST
			}

			c.yp.SetOperation("DELETE")
		}

		yangListName := getRedisTblToYangList(tbl, key)

		//Get the YANG validator node
		var node *xmlquery.Node = nil
		if c.requestCache[tbl][key][0].YangData != nil { //get the node for CREATE/UPDATE or DELETE operation
			node = c.requestCache[tbl][key][0].YangData
		} else {
			//Find the node from YANG tree
			node = c.moveToYangList(yangListName, key)
		}

		if node == nil {
			CVL_LOG(WARNING, "Could not find data for semantic validation, "+
				"table %s , key %s", tbl, key)
			continue
		}

		//Step 3.2 : Run all custom validations
		cvlErrObj = c.doCustomValidation(node, custvCfg, &custvCfg[i], yangListName,
			tbl, key)
		if cvlErrObj.ErrCode != CVL_SUCCESS {
			return cvlErrObj, cvlErrObj.ErrCode
		}

		if cfgData[i].ReplaceOp {
			// Fields requested with OP_DELETE are already removed from
			// OP_UPDATE request and validated. Skip further semantic validation.
			if cmn.OP_DELETE == cfgData[i].VOp {
				c.markCfgDataValidated(&cfgData[i], tbl, key)
				continue
			}
			c.updateYangTreeForReplaceOp(node, cfgData)
			if IsTraceAllowed(TRACE_UPDATE) {
				TRACE_LOG(TRACE_UPDATE, "After Replace, YANG data tree: %s", c.yv.root.OutputXML(false))
			}
		}

		//Step 3.3 : Perform semantic validation
		if cvlErrObj = c.validateSemantics(node, yangListName, key, &cfgData[i]); cvlErrObj.ErrCode != CVL_SUCCESS {
			return cvlErrObj, cvlErrObj.ErrCode
		}

		// Setting VType to VALIDATE_NONE for already validated cfgData
		// This can be useful in look-back and look-ahead in case of bulk requests.
		c.markCfgDataValidated(&cfgData[i], tbl, key)
	}

	return cvlErrObj, CVL_SUCCESS
}

func (c *CVL) markCfgDataValidated(cfgData *cmn.CVLEditConfigData, tbl, key string) {
	if cfgData.VType != cmn.VALIDATE_NONE {
		for k := range c.requestCache[tbl][key] {
			reqcachedCfgData := c.requestCache[tbl][key][k].ReqData
			if cfgData.Key == reqcachedCfgData.Key && cmp.Equal(*cfgData, reqcachedCfgData) {
				c.requestCache[tbl][key][k].ReqData.VType = cmn.VALIDATE_NONE
			}
		}
		cfgData.VType = cmn.VALIDATE_NONE
	}
}

// GetErrorString Fetch the Error Message from CVL Return Code.
func GetErrorString(retCode CVLRetCode) string {

	return cvlErrorMap[retCode]

}

// ValidateKeys Validate key only
func (c *CVL) ValidateKeys(key []string) CVLRetCode {
	return CVL_NOT_IMPLEMENTED
}

// ValidateKeyData Validate key and data
func (c *CVL) ValidateKeyData(key string, data string) CVLRetCode {
	return CVL_NOT_IMPLEMENTED
}

// ValidateFields Validate key, field and value
func (c *CVL) ValidateFields(key string, field string, value string) CVLRetCode {
	return CVL_NOT_IMPLEMENTED
}

// getListNameFromLeafRef - Extracts the yang list name from the leafRefInfo.
// Returns empty string otherwise.
func getListNameFromLeafRef(leafRef *leafRefInfo) string {
	if len(leafRef.yangListNames) == 0 {
		//No-leafref case return empty string
		return ""
	}
	return leafRef.yangListNames[0]
}

func (c *CVL) addDepEdges(graph *toposort.Graph, tableList []string) {
	//Add all the depedency edges for graph nodes
	for ti := 0; ti < len(tableList); ti++ {

		redisTblTo := tableList[ti]

		for tj := 0; tj < len(tableList); tj++ {

			if tableList[ti] == tableList[tj] {
				//same table, continue
				continue
			}

			redisTblFrom := tableList[tj]

			//map for checking duplicate edge
			dupEdgeCheck := map[string]string{}

			for _, leafRefs := range modelInfo.tableInfo[tableList[tj]].leafRef {
				for _, leafRef := range leafRefs {
					listName := getListNameFromLeafRef(leafRef)
					listName = strings.TrimSuffix(listName, "_LIST")
					if listName != redisTblTo {
						continue
					}

					toName, exists := dupEdgeCheck[redisTblFrom]
					if exists && (toName == redisTblTo) {
						//Don't add duplicate edge
						continue
					}

					graph.AddEdge(redisTblFrom, redisTblTo)
					dupEdgeCheck[redisTblFrom] = redisTblTo
					CVL_LOG(INFO_DEBUG, "addDepEdges(): Adding edge %s -> %s", redisTblFrom, redisTblTo)
				}
			}

			for _, depTblName := range modelInfo.tableInfo[redisTblTo].dependentTables {
				if tableList[tj] == depTblName {
					graph.AddEdge(depTblName, redisTblTo)
					dupEdgeCheck[depTblName] = redisTblTo
					CVL_LOG(INFO_DEBUG, "addDepEdges() with sonic-ext: Adding edge %s -> %s", tableList[tj], redisTblTo)
				}
			}
		}
	}
}

// sortDepTables Sort list of given tables as per their dependency - Returns list names
func (c *CVL) sortDepTables(inTableList []string) ([]string, CVLRetCode) {

	tableListMap := make(map[string]bool)

	//Skip all unknown tables
	for ti := 0; ti < len(inTableList); ti++ {
		_, exists := modelInfo.tableInfo[inTableList[ti]]
		if !exists {
			continue
		}

		//Add to map to avoid duplicate nodes
		tableListMap[inTableList[ti]] = true
	}

	tableList := []string{}

	//Add all the table names in graph nodes
	graph := toposort.NewGraph(len(tableListMap))
	for tbl := range tableListMap {
		graph.AddNodes(tbl)
		tableList = append(tableList, tbl)
	}

	//Add all dependency edges
	c.addDepEdges(graph, tableList)

	//Now perform topological sort
	result, ret := graph.Toposort()
	if !ret {
		CVL_LOG(ERROR, "SortDepTables: Toposort failed: %v", result)
		return nil, CVL_ERROR
	}

	return result, CVL_SUCCESS
}

// SortDepList Sort list of given tables as per their dependency, returns LIST names, useful for testing
func (c *CVL) SortDepList(inTables []string) ([]string, CVLRetCode) {
	list := make([]string, 0, len(inTables))
	for _, inTbl := range inTables {
		if tblLists, ok := modelInfo.redisTableToYangList[inTbl]; ok {
			list = append(list, tblLists...)
		}
	}
	return c.sortDepTables(list)
}

// SortDepTables Sort list of given tables as per their dependency
func (c *CVL) SortDepTables(inTableList []string) ([]string, CVLRetCode) {

	list := make([]string, 0, len(inTableList))
	for _, inTbl := range inTableList {
		tblLists, _ := modelInfo.redisTableToYangList[inTbl]
		list = append(list, tblLists...)
	}
	result, status := c.sortDepTables(list)
	if status != CVL_SUCCESS {
		return nil, CVL_ERROR
	}

	return processTopoSortResult(result)
}

// GetOrderedTables Get the order list(parent then child) of tables in a given YANG module
// within a single model this is obtained using leafref relation
func (c *CVL) GetOrderedTables(yangModule string) ([]string, CVLRetCode) {
	tableList := []string{}

	//Get all the table names under this model
	for tblName, tblNameInfo := range modelInfo.tableInfo {
		if tblNameInfo.modelName == yangModule {
			tableList = append(tableList, tblName)
		}
	}

	result, status := c.sortDepTables(tableList)
	if status != CVL_SUCCESS {
		return nil, CVL_ERROR
	}

	return processTopoSortResult(result)
}

func (c *CVL) GetOrderedDepTables(yangModule, tableName string) ([]string, CVLRetCode) {
	var tblLists []string                                    // stores lists related to tableName */
	tableList := make([]string, 0, len(modelInfo.tableInfo)) // stores all tables in the module

	if lists, exists := modelInfo.redisTableToYangList[tableName]; !exists {
		CVL_LOG(INFO_DEBUG, "GetOrderedDepTables(): Unknown table %s\n", tableName)
		return nil, CVL_ERROR
	} else {
		tblLists = lists
	}

	//Get all the table names under this yang module
	for tblName, tblNameInfo := range modelInfo.tableInfo {
		if tblNameInfo.modelName == yangModule {
			tableList = append(tableList, tblName)
		}
	}

	graph := toposort.NewGraph(len(tableList))
	redisTblTo := tableName
	graph.AddNodes(redisTblTo)

	for _, tbl := range tableList {
		if tableName == modelInfo.tableInfo[tbl].redisTableName {
			//same table, continue
			continue
		}
		redisTblFrom := tbl

		//map for checking duplicate edge
		dupEdgeCheck := map[string]string{}

		for _, leafRefs := range modelInfo.tableInfo[tbl].leafRef {
			for _, leafRef := range leafRefs {
				var isSameTbl bool
				var isLeafrefTargetIsKey bool
				listName := getListNameFromLeafRef(leafRef)
				listName = strings.TrimSuffix(listName, "_LIST")
				for _, v := range tblLists {
					if v == listName {
						isSameTbl = true
					}
					for _, key := range modelInfo.tableInfo[v].keys {
						if key == leafRef.targetNodeName {
							isLeafrefTargetIsKey = true
						}
					}
				}
				if !isSameTbl {
					continue
				}

				// if target node of leaf-ref is not key, then skip
				if !(isLeafrefTargetIsKey) {
					continue
				}

				toName, exists := dupEdgeCheck[redisTblFrom]
				if exists && (toName == redisTblTo) {
					//Don't add duplicate edge
					continue
				}

				graph.AddNodes(redisTblFrom)
				graph.AddEdge(redisTblFrom, redisTblTo)
				dupEdgeCheck[redisTblFrom] = redisTblTo
			}
		}
		/* handle dependent-on */
		for _, tn := range tblLists {
			for _, depTblName := range modelInfo.tableInfo[tn].dependentTables {
				if tbl == depTblName {
					graph.AddNodes(depTblName)
					graph.AddEdge(depTblName, redisTblTo)
					dupEdgeCheck[depTblName] = redisTblTo
				}
			}
		}
	}

	//Now perform topological sort
	result, ret := graph.Toposort()
	if !ret {
		return nil, CVL_ERROR
	}

	return processTopoSortResult(result)
}

func (c *CVL) addDepTables(tableMap map[string]bool, listEntry string) {
	tableMap[listEntry] = true
	for _, leafRefs := range modelInfo.tableInfo[listEntry].leafRef {
		for _, leafRef := range leafRefs {
			for _, refTbl := range leafRef.yangListNames {
				c.addDepTables(tableMap, refTbl) //call recursively
			}
		}
	}
	//Now add table on which this table is dependent through 'dependent-on' extension
	if len(modelInfo.tableInfo[listEntry].dependentOnTable) > 0 {
		depTbl := modelInfo.tableInfo[listEntry].dependentOnTable
		depTbl = strings.TrimSuffix(depTbl, "_LIST")
		c.addDepTables(tableMap, depTbl)
	}
}

// processTopoSortResult - Converts List to Table name and placeholder for all future processing
func processTopoSortResult(result []string) ([]string, CVLRetCode) {
	lastOccurrence := make(map[string]int) // This map tracks the last occurrence of each entry
	processedResult := []string{}          // This will store the final processed list

	for i, entry := range result {
		// Replace the entry with the table name from modelInfo, if exists
		if newName, exists := modelInfo.tableInfo[entry]; exists {
			entry = newName.redisTableName
			result[i] = entry // Update the entry in the result slice
		}
		// Always update the last occurrence of this entry
		lastOccurrence[entry] = i
	}

	// Now, filter the results to include only the last occurrence of each entry
	for i, entry := range result {
		if i == lastOccurrence[entry] {
			processedResult = append(processedResult, entry)
		}
	}

	return processedResult, CVL_SUCCESS
}

// GetDepTables Get the list of dependent tables for a given table in a YANG module
func (c *CVL) GetDepTables(yangModule string, tableName string) ([]string, CVLRetCode) {
	tableList := []string{}
	tblMap := make(map[string]bool)

	if _, exists := modelInfo.redisTableToYangList[tableName]; !exists {
		CVL_LOG(INFO_DEBUG, "GetDepTables(): Unknown table %s\n", tableName)
		return []string{}, CVL_ERROR
	}

	for _, listEntry := range modelInfo.redisTableToYangList[tableName] {
		c.addDepTables(tblMap, listEntry)
	}

	for tblName := range tblMap {
		tableList = append(tableList, tblName)
	}

	//Add all the table names in graph nodes
	graph := toposort.NewGraph(len(tableList))
	for ti := 0; ti < len(tableList); ti++ {
		CVL_LOG(INFO_DEBUG, "GetDepTables(): Adding node %s\n", tableList[ti])
		graph.AddNodes(tableList[ti])
	}

	//Add all dependency edges
	c.addDepEdges(graph, tableList)

	//Now perform topological sort
	result, ret := graph.Toposort()
	if !ret {
		return nil, CVL_ERROR
	}

	return processTopoSortResult(result)
}

// Parses the JSON string buffer and returns
// array of dependent fields to be deleted
func (c *CVL) getDepDeleteField(refKey, hField, hValue, jsonBuf string) []CVLDepDataForDelete {
	//Parse the JSON map received from lua script
	var v interface{}
	b := []byte(jsonBuf)
	if err := json.Unmarshal(b, &v); err != nil {
		return []CVLDepDataForDelete{}
	}

	depEntries := []CVLDepDataForDelete{}

	var dataMap map[string]interface{} = v.(map[string]interface{})

	for tbl, keys := range dataMap {
		for key, fields := range keys.(map[string]interface{}) {
			yangList := getRedisTblToYangList(tbl, key)
			tblKey := tbl + modelInfo.tableInfo[yangList].redisKeyDelim + key
			entryMap := make(map[string]map[string]string)
			entryMap[tblKey] = make(map[string]string)

			for field := range fields.(map[string]interface{}) {
				if (field != hField) && (field != (hField + "@")) {
					continue
				}

				if field == (hField + "@") {
					//leaf-list - specific value to be deleted
					entryMap[tblKey][field] = hValue
				} else {
					// If mandatory field is getting deleted, then instead of
					// specific field deletion, entire entry to be deleted.
					// So need to delete dependent entries also when this entire
					// entry is deleted. Find all dependent entries and mark
					// for deletion
					if isMandatoryTrueNode(yangList, field) {
						retDepEntries := c.GetDepDataForDelete(tblKey)
						depEntries = append(depEntries, retDepEntries...)
					} else {
						//leaf - specific field to be deleted
						entryMap[tblKey][field] = ""
					}
				}
			}
			depEntries = append(depEntries, CVLDepDataForDelete{
				RefKey: refKey,
				Entry:  entryMap,
			})
		}
	}

	return depEntries
}

// GetDepDataForDelete Get the dependent (Redis keys) to be deleted or modified
// for a given entry getting deleted
func (c *CVL) GetDepDataForDelete(redisKey string) []CVLDepDataForDelete {

	type filterScript struct {
		script string
		field  string
		value  string
	}

	tableName, key := splitRedisKey(redisKey)
	// Determine the correct redis table name
	// For ex. LOOPBACK_INTERFACE_LIST and LOOPBACK_INTERFACE_IPADDR_LIST are
	// present in same container LOOPBACK_INTERFACE
	tableName = getRedisTblToYangList(tableName, key)

	if (tableName == "") || (key == "") {
		CVL_LOG(INFO_DEBUG, "GetDepDataForDelete(): Unknown or invalid table %s\n",
			tableName)
	}

	if _, exists := modelInfo.tableInfo[tableName]; !exists {
		CVL_LOG(INFO_DEBUG, "GetDepDataForDelete(): Unknown table %s\n", tableName)
		return []CVLDepDataForDelete{}
	}

	redisKeySep := modelInfo.tableInfo[tableName].redisKeyDelim
	redisMultiKeys := strings.Split(key, redisKeySep)

	// There can be multiple leaf in Reference table with leaf-ref to same target field
	// Hence using array of filterScript and redis.StringSliceCmd
	mCmd := map[string][]cmn.StrSliceResult{}
	mFilterScripts := map[string][]filterScript{}
	pipe := c.dbAccess.Pipeline()

	for _, refTbl := range modelInfo.tableInfo[tableName].refFromTables {

		//check if ref field is a key
		numKeys := len(modelInfo.tableInfo[refTbl.tableName].keys)
		refRedisTblName := getYangListToRedisTbl(refTbl.tableName)
		idx := 0

		if refRedisTblName == "" {
			continue
		}

		// Find the targetnode from leaf-refs on refTbl.field
		var refTblTargetNodeName string
		for _, refTblLeafRef := range modelInfo.tableInfo[refTbl.tableName].leafRef[refTbl.field] {
			if (refTblLeafRef.path != "non-leafref") && (len(refTblLeafRef.yangListNames) > 0) {
				var isTargetNodeFound bool
				for k := range refTblLeafRef.yangListNames {
					if refTblLeafRef.yangListNames[k] == tableName {
						refTblTargetNodeName = refTblLeafRef.targetNodeName
						isTargetNodeFound = true
						break
					}
				}
				if isTargetNodeFound {
					break
				}
			}
		}

		// Determine the correct value of key in case of composite key
		if len(redisMultiKeys) > 1 {
			rediskeyTblKeyPatterns := strings.Split(modelInfo.tableInfo[tableName].redisKeyPattern, redisKeySep)
			for z := 1; z < len(rediskeyTblKeyPatterns); z++ { // Skipping 0th position, as it is a tableName
				if rediskeyTblKeyPatterns[z] == fmt.Sprintf("{%s}", refTblTargetNodeName) {
					key = redisMultiKeys[z-1]
					break
				}
			}
		}

		if _, exists := mCmd[refTbl.tableName]; !exists {
			mCmd[refTbl.tableName] = make([]cmn.StrSliceResult, 0)
		}
		mCmdArr := mCmd[refTbl.tableName]

		for ; idx < numKeys; idx++ {
			if modelInfo.tableInfo[refTbl.tableName].keys[idx] != refTbl.field {
				continue
			}

			expr := CreateFindKeyExpression(refTbl.tableName, map[string]string{refTbl.field: key})
			CVL_LOG(INFO_DEBUG, "GetDepDataForDelete()->CreateFindKeyExpression: %s\n", expr)

			mCmdArr = append(mCmdArr, pipe.Keys(expr))
			break
		}
		mCmd[refTbl.tableName] = mCmdArr

		if idx == numKeys {
			//field is hash-set field, not a key, match with hash-set field
			//prepare the lua filter script
			// ex: (h['members'] == 'Ethernet4' or h['members@'] == 'Ethernet4' or
			//(string.find(h['members@'], 'Ethernet4,') != nil)
			//',' to include leaf-list case
			if _, exists := mFilterScripts[refTbl.tableName]; !exists {
				mFilterScripts[refTbl.tableName] = make([]filterScript, 0)
			}
			fltScrs := mFilterScripts[refTbl.tableName]
			fltScrs = append(fltScrs, filterScript{
				script: fmt.Sprintf("return (h['%s'] ~= nil and (h['%s'] == '%s' or h['%s'] == '[%s|%s]')) or "+
					"(h['%s@'] ~= nil and ((h['%s@'] == '%s') or "+
					"(string.find(h['%s@']..',', '%s,') ~= nil)))",
					refTbl.field, refTbl.field, key, refTbl.field, tableName, key,
					refTbl.field, refTbl.field, key,
					refTbl.field, key),
				field: refTbl.field,
				value: key,
			})
			mFilterScripts[refTbl.tableName] = fltScrs
		}
	}

	err := pipe.Exec()
	if err != nil {
		CVL_LOG(WARNING, "Failed to fetch dependent key details for table %s", tableName)
	}
	pipe.Close()

	depEntries := []CVLDepDataForDelete{}

	//Add dependent keys which should be modified
	for tableName, mFilterScriptArr := range mFilterScripts {
		for _, mFilterScript := range mFilterScriptArr {
			s := cmn.Search{Pattern: tableName + "|*", Predicate: mFilterScript.script, KeyNames: modelInfo.tableInfo[tableName].keys, WithField: mFilterScript.field}
			refEntriesJson, err := c.dbAccess.Lookup(s).Result()

			if err != nil {
				CVL_LOG(INFO_DEBUG, "Lua script status: (%v)", err)
			}

			if refEntriesJson != "" {
				//Add all keys whose fields to be deleted
				depEntries = append(depEntries, c.getDepDeleteField(redisKey, mFilterScript.field, mFilterScript.value, refEntriesJson)...)
			}
		}
	}

	keysArr := []string{}
	for tblName, mCmdArr := range mCmd {
		for idx := range mCmdArr {
			keys := mCmdArr[idx]
			res, err := keys.Result()
			if err != nil {
				CVL_LOG(WARNING, "Failed to fetch dependent key details for table %s", tblName)
				continue
			}

			//Add keys found
			for _, k := range res {
				entryMap := make(map[string]map[string]string)
				entryMap[k] = make(map[string]string)
				depEntries = append(depEntries, CVLDepDataForDelete{
					RefKey: redisKey,
					Entry:  entryMap,
				})
			}

			keysArr = append(keysArr, res...)
		}
	}

	TRACE_LOG(INFO_TRACE, "GetDepDataForDelete() : input key %s, "+
		"entries to be deleted : %v", redisKey, depEntries)

	//For each key, find dependent data for delete recursively
	for i := 0; i < len(keysArr); i++ {
		retDepEntries := c.GetDepDataForDelete(keysArr[i])
		depEntries = append(depEntries, retDepEntries...)
	}

	return depEntries
}

// Update global stats for all sessions
func updateValidationTimeStats(td time.Duration) {
	statsMutex.Lock()

	cfgValidationStats.Hits++
	if td > cfgValidationStats.Peak {
		cfgValidationStats.Peak = td
	}

	cfgValidationStats.Time += td

	statsMutex.Unlock()
}

// GetValidationTimeStats Retrieve global stats
func GetValidationTimeStats() ValidationTimeStats {
	return cfgValidationStats
}

// ClearValidationTimeStats Clear global stats
func ClearValidationTimeStats() {
	statsMutex.Lock()

	cfgValidationStats.Hits = 0
	cfgValidationStats.Peak = 0
	cfgValidationStats.Time = 0

	statsMutex.Unlock()
}

// CreateFindKeyExpression Create expression for searching DB entries based on given key fields and values.
// Expressions created will be like CFG_L2MC_STATIC_MEMBER_TABLE|*|*|Ethernet0
func CreateFindKeyExpression(tableName string, keyFldValPair map[string]string) string {
	var expr string

	refRedisTblName := getYangListToRedisTbl(tableName)
	tempSlice := []string{refRedisTblName}
	sep := modelInfo.tableInfo[tableName].redisKeyDelim

	tblKeyPatterns := strings.Split(modelInfo.tableInfo[tableName].redisKeyPattern, sep)
	for z := 1; z < len(tblKeyPatterns); z++ {
		fldFromPattern := tblKeyPatterns[z][1 : len(tblKeyPatterns[z])-1] //remove "{" and "}"
		if val, exists := keyFldValPair[fldFromPattern]; exists {
			tempSlice = append(tempSlice, val)
		} else {
			tempSlice = append(tempSlice, "*")
		}
	}

	expr = strings.Join(tempSlice, sep)

	return expr
}

// GetAllReferringTables Returns list of all tables and fields which has leaf-ref
// to given table. For ex. tableName="PORT" will return all tables and fields
// which has leaf-ref to "PORT" table.
func (c *CVL) GetAllReferringTables(tableName string) map[string][]string {
	var refTbls = make(map[string][]string)
	if tblInfo, exists := modelInfo.tableInfo[tableName]; exists {
		for _, refTbl := range tblInfo.refFromTables {
			fldArr := refTbls[refTbl.tableName]
			fldArr = append(fldArr, refTbl.field)
			refTbls[refTbl.tableName] = fldArr
		}
	}

	return refTbls
}

func ReconfigureRedisOptions(opt redis.Options) {
	UpdateRedisOptions(&opt)

	newRedisClient := NewDbClient("CONFIG_DB")
	if newRedisClient != nil && redisClient != nil {
		redisClient.Close()
		redisClient = newRedisClient
		CVL_LEVEL_LOG(INFO, "ReconfigureRedisOptions -- Updated redisClient. Options [%v]", redisClient.Options())
	} else if newRedisClient == nil {
		CVL_LEVEL_LOG(INFO, "ReconfigureRedisOptions -- Failed to update redisClient")
	}
}
