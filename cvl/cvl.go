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
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/Azure/sonic-mgmt-common/cvl/internal/yparser"
	"github.com/antchfx/jsonquery"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/go-redis/redis/v7"
	"io/ioutil"
	"path/filepath"
	"sync"
	"unsafe"

	custv "github.com/Azure/sonic-mgmt-common/cvl/custom_validation"
	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

// DB number
const (
	APPL_DB uint8 = 0 + iota
	ASIC_DB
	COUNTERS_DB
	LOGLEVEL_DB
	CONFIG_DB
	PFC_WD_DB
	FLEX_COUNTER_DB = PFC_WD_DB
	STATE_DB
	SNMP_OVERLAY_DB
	INVALID_DB
)

const DEFAULT_CACHE_DURATION uint16 = 300 /* 300 sec */
const MAX_BULK_ENTRIES_IN_PIPELINE int = 50
const MAX_DEVICE_METADATA_FETCH_RETRY = 60
const PLATFORM_SCHEMA_PATH = "platform/"

// var reLeafRef *regexp.Regexp = nil
var reHashRef *regexp.Regexp = nil

//var reSelKeyVal *regexp.Regexp = nil
//var reLeafInXpath *regexp.Regexp = nil

var cvlInitialized bool
var dbNameToDbNum map[string]uint8

// map of lua script loaded
var luaScripts map[string]*redis.Script

type tblFieldPair struct {
	tableName string
	field     string
}

type mustInfo struct {
	expr     string      //must expression
	exprTree *xpath.Expr //compiled expression tree
	errCode  string      //err-app-tag
	errStr   string      //error message
}

type leafRefInfo struct {
	path           string      //leafref path
	exprTree       *xpath.Expr //compiled expression tree
	yangListNames  []string    //all yang list in path
	targetNodeName string      //target node name
}

type whenInfo struct {
	expr          string      //when expression
	exprTree      *xpath.Expr //compiled expression tree
	nodeNames     []string    //list of nodes under when condition
	yangListNames []string    //all yang list in expression
}

// Important schema information to be loaded at bootup time
type modelTableInfo struct {
	dbNum           uint8
	modelName       string
	redisTableName  string //To which Redis table it belongs to, used for 1 Redis to N Yang List
	module          *yparser.YParserModule
	keys            []string
	redisKeyDelim   string
	redisKeyPattern string
	redisTableSize  int
	mapLeaf         []string                  //for 'mapping  list'
	leafRef         map[string][]*leafRefInfo //for storing all leafrefs for a leaf in a table,
	//multiple leafref possible for union
	mustExpr         map[string][]*mustInfo
	whenExpr         map[string][]*whenInfo
	tablesForMustExp map[string]cmn.CVLOperation
	refFromTables    []tblFieldPair      //list of table or table/field referring to this table
	custValidation   map[string][]string // Map for custom validation node and function name
	dfltLeafVal      map[string]string   //map of leaf names and default value
	mandatoryNodes   map[string]bool     //map of leaf names and mandatory flag & leaf-list with min-elements > 0
	dependentOnTable string              // Name of table on which it is dependent
	dependentTables  []string            // list of dependent tables
	has_static_key   bool                // True, if LIST has sonic-extension:tbl-key
}

// CVLErrorInfo Struct for CVL Error Info
type CVLErrorInfo struct {
	TableName        string     /* Table having error */
	ErrCode          CVLRetCode /* CVL Error return Code. */
	CVLErrDetails    string     /* CVL Error Message details. */
	Keys             []string   /* Keys of the Table having error. */
	Value            string     /* Field Value throwing error */
	Field            string     /* Field Name throwing error . */
	Msg              string     /* Detailed error message. */
	ConstraintErrMsg string     /* Constraint error message. */
	ErrAppTag        string
}

// DepDataCacheType cache type for dependent data
type DepDataCacheType map[string]interface{}

// CVL Struct for CVL session
type CVL struct {
	//redisClient *redis.Client
	yp           *yparser.YParser
	tmpDbCache   map[string]interface{}                        //map of table storing map of key-value pair
	requestCache map[string]map[string][]*cmn.RequestCacheType //Cache of validated data,
	//per table, per key. Can be used as dependent data in next request
	maxTableElem map[string]int              //max element count per table
	batchLeaf    []*yparser.YParserLeafValue //field name and value
	depDataCache DepDataCacheType            // Cache dependent data to prevent adding same data multiple times in XML
	//chkLeafRefWithOthCache bool
	yv         *YValidator               //Custom YANG validator for validating external dependencies
	custvCache custv.CustValidationCache //Custom validation cache per session
	dbAccess   cmn.DBAccess              //DB access interface
}

// Struct for model namepsace and prefix
type modelNamespace struct {
	prefix string
	ns     string
}

// Struct for storing all YANG list schema info
type modelDataInfo struct {
	modelNs              map[string]*modelNamespace   //model namespace
	tableInfo            map[string]*modelTableInfo   //redis table to model name and keys
	redisTableToYangList map[string][]string          //Redis table to all YANG lists when it is not 1:1 mapping
	redisTableToListKeys map[string]map[string]string //Redis table to keys(marked using sonic-extension:tbl-key), stores internal map from key to list
	allKeyDelims         map[string]bool
}

//Struct for storing global DB cache to store DB which are needed frequently like PORT
/*type dbCachedData struct {
	root *yparser.YParserNode //Root of the cached data
	startTime time.Time  //When cache started
	expiry uint16    //How long cache should be maintained in sec
}*/

// Global data cache for redis table
type cvlGlobalSessionType struct {
	//db map[string]dbCachedData
	//pubsub *redis.PubSub
	stopChan chan int //stop channel to stop notification listener
	cv       *CVL
	mutex    *sync.Mutex
}

// Struct for storing key and value pair
type keyValuePairStruct struct {
	key    string
	values []string
}

var cvg cvlGlobalSessionType

// Single redis client for validation
var redisClient *redis.Client

// Stores important model info
var modelInfo modelDataInfo

func TRACE_LOG(tracelevel CVLTraceLevel, fmtStr string, args ...interface{}) {
	TRACE_LEVEL_LOG(tracelevel, fmtStr, args...)
}

func CVL_LOG(level CVLLogLevel, fmtStr string, args ...interface{}) {
	CVL_LEVEL_LOG(level, fmtStr, args...)
}

// package init function
func init() {
	if os.Getenv("CVL_SCHEMA_PATH") != "" {
		CVL_SCHEMA = os.Getenv("CVL_SCHEMA_PATH") + "/"
	}

	if os.Getenv("CVL_DEBUG") != "" {
		SetTrace(true)
	}

	xpath.SetKeyGetClbk(func(listName string) []string {
		if modelInfo.tableInfo[listName] != nil {
			return modelInfo.tableInfo[listName].keys
		}

		return nil
	})

	ConfigFileSyncHandler()

	cvlCfgMap := ReadConfFile()

	if cvlCfgMap != nil {
		CVL_LOG(INFO_DEBUG, "Current Values of CVL Configuration File %v", cvlCfgMap)
	}

	//regular expression for leafref and hashref finding
	reHashRef = regexp.MustCompile(`\[(.*)\|(.*)\]`)

	if Initialize() != CVL_SUCCESS {
		CVL_LOG(FATAL, "CVL initialization failed")
	}

	//Global session keeps the global cache
	cvg.cv, _ = ValidationSessOpen(nil)
	//Create buffer channel of length 1
	cvg.stopChan = make(chan int, 1)
	//Initialize mutex
	cvg.mutex = &sync.Mutex{}
	//Intialize mutex for stats
	statsMutex = &sync.Mutex{}

	_, err := redisClient.ConfigSet("notify-keyspace-events", "AKE").Result()
	if err != nil {
		CVL_LOG(WARNING, "Could not enable notification error %s", err)
	}

	xpath.SetLogCallback(func(fmt string, args ...interface{}) {
		if !IsTraceAllowed(TRACE_SEMANTIC) {
			return
		}

		TRACE_LOG(TRACE_SEMANTIC, "XPATH: "+fmt, args...)
	})
}

func Debug(on bool) {
	yparser.Debug(on)
}

// isLeafListNode checks if the xml node represents a leaf-list field
func isLeafListNode(node *xmlquery.Node) bool {
	return len(node.Attr) != 0 && node.Attr[0].Name.Local == "leaf-list"
}

// getNodeName returns database field name for the xml node.
func getNodeName(node *xmlquery.Node) string {
	if isLeafListNode(node) {
		return node.Data + "@"
	}
	return node.Data
}

// Load all YIN schema files, apply deviation files
func loadSchemaFiles() CVLRetCode {

	platformName := ""
	// Wait to check if CONFIG_DB is populated with DEVICE_METADATA.
	// This is needed to apply deviation file
	retryCnt := 0
	for ; retryCnt < MAX_DEVICE_METADATA_FETCH_RETRY; retryCnt++ {
		deviceMetaDataKey, err := redisClient.Keys("DEVICE_METADATA|localhost").Result()
		if (err != nil) || (len(deviceMetaDataKey) == 0) {
			//Retry for 1 min
			time.Sleep(100 * time.Millisecond) //sleep for 1 sec and then retry
			continue
		}

		//Redis is populated with DEVICE_METADATA
		break
	}

	//Now try to fetch the platform details
	if retryCnt < MAX_DEVICE_METADATA_FETCH_RETRY {
		deviceMetaData, err := redisClient.HGetAll("DEVICE_METADATA|localhost").Result()
		var exists bool
		platformName, exists = deviceMetaData["platform"]
		if !exists || (err != nil) || (platformName == "") {
			CVL_LOG(WARNING, "Could not fetch 'platform' details from CONFIG_DB")
		}
	}

	//Scan schema directory to get all schema files
	modelFiles, err := filepath.Glob(CVL_SCHEMA + "/*.yin")
	if err != nil {
		CVL_LOG(FATAL, "Could not read schema files %v", err)
	}

	moduleMap := map[string]*yparser.YParserModule{}
	// Load all common schema files
	for _, modelFilePath := range modelFiles {
		_, modelFile := filepath.Split(modelFilePath)

		TRACE_LOG(TRACE_LIBYANG, "Parsing schema file %s ...",
			modelFilePath)

		// Now parse each schema file
		var module *yparser.YParserModule
		if module, _ = yparser.ParseSchemaFile(modelFilePath); module == nil {
			CVL_LOG(FATAL, fmt.Sprintf("Unable to parse schema file %s", modelFile))
			return CVL_ERROR
		}

		moduleMap[modelFile] = module
	}

	// Load all platform specific schema files based on platform details
	// present in DEVICE_METADATA
	for {
		if platformName == "" {
			CVL_LOG(INFO, "Skipping parsing of any platform specific YIN schema "+
				"files as platform name can't be determined")
			break
		}

		// Read directory under 'platform' directory
		allDirs, errDir := ioutil.ReadDir(CVL_SCHEMA + "/" + PLATFORM_SCHEMA_PATH)
		if (errDir != nil) || (len(allDirs) == 0) {
			CVL_LOG(INFO, "Could not read platform schema location or no platform "+
				"specific schema exists. %v", err)
			break
		}

		// For matched platform directory parse all schema files
		for _, sDir := range allDirs {

			sDirName := sDir.Name()
			//Check which directory matches
			if !(strings.Contains(platformName, sDirName)) {
				continue
			}

			//Get all platform specific YIN schema file names
			modelFiles, err := filepath.Glob(CVL_SCHEMA + "/" +
				PLATFORM_SCHEMA_PATH + "/" + sDirName + "/*.yin")
			if err != nil {
				CVL_LOG(WARNING, "Could not read platform schema directory %v", err)
				break
			}

			//Now parse platform schema files
			for _, modelFilePath := range modelFiles {
				_, modelFile := filepath.Split(modelFilePath)

				TRACE_LOG(TRACE_YPARSER, "Parsing platform specific schema"+"file %s ...\n", modelFilePath)

				var module *yparser.YParserModule
				if module, _ = yparser.ParseSchemaFile(modelFilePath); module == nil {

					CVL_LOG(ERROR, "Unable to parse platform specific schema file %s", modelFile)
					return CVL_ERROR
				}

				moduleMap[modelFile] = module
			}

			//platform found
			break
		}

		break
	}

	for modelFile, parsedModule := range moduleMap {
		//store schema related info to use in validation
		storeModelInfo(modelFile, parsedModule)
	}

	return CVL_SUCCESS
}

// Get list of YANG list names used in xpath expression
func getYangListNamesInExpr(expr string) []string {
	tbl := []string{}

	//Check with all table names
	for tblName := range modelInfo.tableInfo {

		//Match 1 - Prefix is used in path
		//Match 2 - Prefix is not used in path, it is in same YANG model
		if strings.Contains(expr, ":"+tblName+"_LIST") || strings.Contains(expr, "/"+tblName+"_LIST") {
			tbl = append(tbl, tblName)
		}
	}

	return tbl
}

// Get all YANG lists referred and the target node for leafref
// Ex: leafref { path "../../../ACL_TABLE/ACL_TABLE_LIST[aclname=current()]/aclname";}
// will return [ACL_TABLE] and aclname
func getLeafRefTargetInfo(path string) ([]string, string) {
	target := ""

	//Get list of all YANG list used in the path
	tbl := getYangListNamesInExpr(path)

	//Get the target node name from end of the path
	idx := strings.LastIndex(path, ":") //check with prefix first
	if idx > 0 {
		target = path[idx+1:]
	} else if idx = strings.LastIndex(path, "/"); idx > 0 { //no prefix there
		target = path[idx+1:]
	}
	if len(tbl) != 1 {
		panic("some error")
	}
	return tbl, target
}

func storeModelInfo(modelFile string, module *yparser.YParserModule) {
	//model is derived from file name
	tokens := strings.Split(modelFile, ".")
	modelName := tokens[0]

	//Store namespace and prefix
	ns, prefix := yparser.GetModelNs(module)
	modelInfo.modelNs[modelName] = &modelNamespace{ns: ns, prefix: prefix}

	list := yparser.GetModelListInfo(module)

	if list == nil {
		CVL_LOG(WARNING, "Unable to get schema details for %s", modelFile)
		return
	}

	for _, lInfo := range list {
		TRACE_LOG(TRACE_YPARSER,
			"Storing schema details for list %s", lInfo.ListName)

		tInfo := modelTableInfo{modelName: modelName}

		tInfo.dbNum = dbNameToDbNum[lInfo.DbName]
		tInfo.redisTableName = lInfo.RedisTableName
		tInfo.module = module
		tInfo.redisKeyDelim = lInfo.RedisKeyDelim
		tInfo.redisKeyPattern = lInfo.RedisKeyPattern
		tInfo.redisTableSize = lInfo.RedisTableSize
		tInfo.keys = lInfo.Keys
		tInfo.mapLeaf = lInfo.MapLeaf
		tInfo.custValidation = lInfo.CustValidation
		tInfo.mandatoryNodes = lInfo.MandatoryNodes
		tInfo.dependentOnTable = lInfo.DependentOnTable

		//store default values used in must and when exp
		tInfo.dfltLeafVal = make(map[string]string, len(lInfo.DfltLeafVal))
		for nodeName, val := range lInfo.DfltLeafVal {
			tInfo.dfltLeafVal[nodeName] = val
		}

		//Store leafref details
		tInfo.leafRef = make(map[string][]*leafRefInfo, len(lInfo.LeafRef))
		for nodeName, lpathArr := range lInfo.LeafRef { //for each leaf or leaf-list
			leafRefInfoArr := []*leafRefInfo{}
			for _, lpath := range lpathArr {
				//just store the leafref path
				leafRefData := leafRefInfo{path: lpath}
				leafRefInfoArr = append(leafRefInfoArr, &leafRefData)
			}
			tInfo.leafRef[nodeName] = leafRefInfoArr
		}

		//Store must expression details
		tInfo.mustExpr = make(map[string][]*mustInfo, len(lInfo.XpathExpr))
		for nodeName, xprArr := range lInfo.XpathExpr {
			for _, xpr := range xprArr {
				tInfo.mustExpr[nodeName] = append(tInfo.mustExpr[nodeName],
					&mustInfo{
						expr:    xpr.Expr,
						errCode: xpr.ErrCode,
						errStr:  xpr.ErrStr,
					})
			}
		}

		//Store when expression details
		tInfo.whenExpr = make(map[string][]*whenInfo, len(lInfo.WhenExpr))
		for nodeName, whenExprArr := range lInfo.WhenExpr {
			for _, whenExpr := range whenExprArr {
				tInfo.whenExpr[nodeName] = append(tInfo.whenExpr[nodeName],
					&whenInfo{
						expr:      whenExpr.Expr,
						nodeNames: whenExpr.NodeNames,
					})
			}
		}

		modelInfo.allKeyDelims[tInfo.redisKeyDelim] = true
		yangList := modelInfo.redisTableToYangList[tInfo.redisTableName]
		yangList = append(yangList, lInfo.ListName)
		//Update the map
		modelInfo.redisTableToYangList[tInfo.redisTableName] = yangList
		if lInfo.Key != "" {
			if modelInfo.redisTableToListKeys[tInfo.redisTableName] == nil {
				modelInfo.redisTableToListKeys[tInfo.redisTableName] = make(map[string]string)
			}
			modelInfo.redisTableToListKeys[tInfo.redisTableName][lInfo.Key] = lInfo.ListName
			tInfo.has_static_key = true
		}

		modelInfo.tableInfo[lInfo.ListName] = &tInfo
	}
}

// Get YANG list to Redis table name
func getYangListToRedisTbl(yangListName string) string {
	if strings.HasSuffix(yangListName, "_LIST") {
		yangListName = yangListName[0 : len(yangListName)-len("_LIST")]
	}
	tInfo, exists := modelInfo.tableInfo[yangListName]

	if exists && (tInfo.redisTableName != "") {
		return tInfo.redisTableName
	}

	return yangListName
}

// This functions build info of dependent table/fields
// which uses a particular table through leafref
func buildRefTableInfo() {

	CVL_LOG(INFO_API, "Building reverse reference info from leafref")

	for tblName, tblInfo := range modelInfo.tableInfo {
		if len(tblInfo.leafRef) == 0 {
			continue
		}

		//For each leafref update the table used through leafref
		for fieldName, leafRefs := range tblInfo.leafRef {
			for _, leafRef := range leafRefs {

				for _, yangListName := range leafRef.yangListNames {
					refTblInfo := modelInfo.tableInfo[yangListName]

					refFromTables := &refTblInfo.refFromTables
					*refFromTables = append(*refFromTables, tblFieldPair{tblName, fieldName})
					modelInfo.tableInfo[yangListName] = refTblInfo
				}

			}
		}

	}

	//Now sort list 'refFromTables' under each table based on dependency among them
	for tblName, tblInfo := range modelInfo.tableInfo {
		if len(tblInfo.refFromTables) == 0 {
			continue
		}

		depTableList := []string{}
		for i := 0; i < len(tblInfo.refFromTables); i++ {
			depTableList = append(depTableList, tblInfo.refFromTables[i].tableName)
		}

		sortedTableList, _ := cvg.cv.sortDepTables(depTableList)
		if len(sortedTableList) == 0 {
			continue
		}

		newRefFromTables := []tblFieldPair{}

		for i := 0; i < len(sortedTableList); i++ {
			//Find fieldName
			fieldName := ""
			for j := 0; j < len(tblInfo.refFromTables); j++ {
				if sortedTableList[i] == tblInfo.refFromTables[j].tableName {
					fieldName = tblInfo.refFromTables[j].field
					newRefFromTables = append(newRefFromTables, tblFieldPair{sortedTableList[i], fieldName})
				}
			}
		}
		//Update sorted refFromTables
		tblInfo.refFromTables = newRefFromTables
		modelInfo.tableInfo[tblName] = tblInfo
	}

}

// This functions build info of dependent tables based on
// 'dependent-on' extension.
func buildTableDependencies() {
	CVL_LOG(INFO_API, "Building table dependencies based on 'dependent-on' extension")
	for tblName, tblInfo := range modelInfo.tableInfo {
		if len(tblInfo.dependentOnTable) == 0 {
			continue
		}
		//if pTblInfo, ok := modelInfo.tableInfo[tblInfo.dependentOnTable]; ok {
		//	pTblInfo.dependentTables = append(pTblInfo.dependentTables, tblName)
		//} else {
		//	CVL_LOG(ERROR, "Dependent-on Table: %s does not exists", tblInfo.dependentOnTable)
		//	continue
		//}

		idx := strings.LastIndex(tblInfo.dependentOnTable, "_LIST")
		if idx > 0 {
			tn := tblInfo.dependentOnTable[:idx]
			CVL_LOG(INFO_API, "Table: %s is dependent on %s", tblName, tn)

			if pTblInfo, ok := modelInfo.tableInfo[tn]; ok {
				pTblInfo.dependentTables = append(pTblInfo.dependentTables, tblName)
			} else {
				CVL_LOG(ERROR, "Dependent-on Table: %s does not exists", tn)
				continue
			}
		} else {
			CVL_LOG(ERROR, "Dependent-on Table: %s must be suffixed with '_LIST'", tblInfo.dependentOnTable)
			continue
		}
	}
}

// Find the tables names in must expression, these tables data need to be fetched
// during semantic validation
func addTableNamesForMustExp() {

	for tblName, tblInfo := range modelInfo.tableInfo {
		if len(tblInfo.mustExpr) == 0 {
			continue
		}

		tblInfo.tablesForMustExp = make(map[string]cmn.CVLOperation)

		for _, mustExpArr := range tblInfo.mustExpr {
			for _, mustExp := range mustExpArr {
				var op cmn.CVLOperation = cmn.OP_NONE
				//Check if 'must' expression should be executed for a particular operation
				if strings.Contains(mustExp.expr, ":operation != 'CREATE'") {
					op = op | cmn.OP_CREATE
				}
				if strings.Contains(mustExp.expr, ":operation != 'UPDATE'") {
					op = op | cmn.OP_UPDATE
				}
				if strings.Contains(mustExp.expr, ":operation != 'DELETE'") {
					op = op | cmn.OP_DELETE
				}

				//store the current table if aggregate function like count() is used
				//check which table name is present in the must expression
				for tblNameSrch := range modelInfo.tableInfo {
					if tblNameSrch == tblName {
						continue
					}
					//Table name should appear like "../VLAN_MEMBER_LIST/tagging_mode' or '
					// "/prt:PORT/prt:ifname"
					//re := regexp.MustCompile(fmt.Sprintf(".*[/]([-_a-zA-Z]*:)?%s_LIST[\\[/]?", tblNameSrch))
					//matches := re.FindStringSubmatch(mustExp.expr)
					//if len(matches) > 0 {
					listName := tblNameSrch + "_LIST"
					matches := strings.Index(mustExp.expr, listName)
					if matches > 0 {
						//stores the table name
						tblInfo.tablesForMustExp[tblNameSrch] = op
					}
				}
			}
		}

		//update map
		modelInfo.tableInfo[tblName] = tblInfo
	}
}

func SplitRedisKey(key string) (string, string) {
	return splitRedisKey(key)
}

// Split key into table prefix and key
func splitRedisKey(key string) (string, string) {

	var foundIdx int = -1
	//Check with all key delim
	for keyDelim := range modelInfo.allKeyDelims {
		foundIdx = strings.Index(key, keyDelim)
		if foundIdx >= 0 {
			//Matched with key delim
			break
		}
	}

	if foundIdx < 0 {
		//No matches
		CVL_LOG(WARNING, "Could not find any of key delimeter %v in key '%s'",
			modelInfo.allKeyDelims, key)
		return "", ""
	}

	tblName := key[:foundIdx]
	tblKey := key[foundIdx+1:]
	tblListName := getRedisTblToYangList(tblName, tblKey)

	if _, exists := modelInfo.tableInfo[tblListName]; !exists {
		//Wrong table name
		CVL_LOG(WARNING, "Could not find table '%s' in schema", tblName)
		return "", ""
	}

	TRACE_LOG(TRACE_SYNTAX, "Split Redis Key %s into (%s, %s)",
		key, tblName, tblKey)

	return tblName, tblKey
}

func splitKeyComponents(table, keyComps string) []string {
	if m := modelInfo.tableInfo[table]; m != nil {
		return strings.Split(keyComps, m.redisKeyDelim)
	}
	return nil
}

// Get the YANG list name from Redis key and table name
// This just returns same YANG list name as Redis table name
// when 1:1 mapping is there. For one Redis table to
// multiple YANG list, it returns appropriate YANG list name
// INTERFACE:Ethernet12 returns ==> INTERFACE
// INTERFACE:Ethernet12:1.1.1.0/32 ==> INTERFACE_IPADDR
func getRedisTblToYangList(tableName, key string) (yangList string) {
	defer func() {
		pYangList := &yangList
		TRACE_LOG(TRACE_SYNTAX, "Got YANG list '%s' "+
			"from Redis Table '%s', Key '%s'", *pYangList, tableName, key)
	}()

	if keyEntry, keyEntryExists := modelInfo.redisTableToListKeys[tableName]; keyEntryExists {
		if yangList, listExist := keyEntry[key]; listExist {
			return yangList
		}
	}

	mapArr, exists := modelInfo.redisTableToYangList[tableName]

	if !exists || (len(mapArr) == 1) { //no map or only one
		//1:1 mapping case
		return tableName
	}

	//As of now determine the mapping based on number of keys
	var foundIdx int = -1
	numOfKeys := 1 //Assume only one key initially
	for keyDelim := range modelInfo.allKeyDelims {
		foundIdx = strings.Index(key, keyDelim)
		if foundIdx >= 0 {
			//Matched with key delim
			keyComps := strings.Split(key, keyDelim)
			numOfKeys = len(keyComps)
			break
		}
	}

	//Check which list has number of keys as 'numOfKeys'
	for i := 0; i < len(mapArr); i++ {
		tblInfo, exists := modelInfo.tableInfo[mapArr[i]]
		if tblInfo.has_static_key {
			continue // skip the singleton list
		}
		if exists {
			if len(tblInfo.keys) == numOfKeys {
				//Found the YANG list matching the number of keys
				return mapArr[i]
			}
		}
	}

	//No matches
	return tableName
}

// Convert Redis key to Yang keys, if multiple key components are there,
// they are separated based on Yang schema
func getRedisToYangKeys(tableName string, redisKey string) []keyValuePairStruct {
	keyNames := modelInfo.tableInfo[tableName].keys
	//First split all the keys components
	keyVals := strings.Split(redisKey, modelInfo.tableInfo[tableName].redisKeyDelim) //split by DB separator
	//Store patterns for each key components by splitting using key delim
	keyPatterns := strings.Split(modelInfo.tableInfo[tableName].redisKeyPattern,
		modelInfo.tableInfo[tableName].redisKeyDelim) //split by DB separator

	if len(keyNames) != len(keyVals) {
		return nil //number key names and values does not match
	}

	mkeys := []keyValuePairStruct{}
	//For each key check the pattern and store key/value pair accordingly
	for idx, keyName := range keyNames {

		//check if key-pattern contains specific key pattern
		if keyPatterns[idx+1] == ("{" + keyName + "}") { //no specific key pattern - just "{key}"
			//Store key/value mapping
			mkeys = append(mkeys, keyValuePairStruct{keyName, []string{keyVals[idx]}})
		} else if keyPatterns[idx+1] == ("({" + keyName + "},)*") { // key pattern is "({key},)*" i.e. repeating keys seperated by ','
			repeatedKeys := strings.Split(keyVals[idx], ",")
			mkeys = append(mkeys, keyValuePairStruct{keyName, repeatedKeys})
		}
	}

	TRACE_LOG(TRACE_SYNTAX, "getRedisToYangKeys() returns %v "+
		"from Redis Table '%s', Key '%s'", mkeys, tableName, redisKey)

	return mkeys
}

// Checks field map values and removes "NULL" entry, create array for leaf-list
func (c *CVL) checkFieldMap(fieldMap *map[string]string) map[string]interface{} {
	fieldMapNew := map[string]interface{}{}

	for field, value := range *fieldMap {
		if field == "NULL" {
			continue
		} else if field[len(field)-1:] == "@" {
			//last char @ means it is a leaf-list/array of fields
			field = field[:len(field)-1] //strip @
			//split the values seprated using ','
			strArr := strings.Split(value, ",")
			//fieldMapNew[field] = strings.Split(value, ",")
			arrMap := make([]interface{}, 0) //len(strArr))
			for _, strArrItem := range strArr {
				arrMap = append(arrMap, strArrItem)
			}
			fieldMapNew[field] = arrMap //make([]interface{}, len(strArr))
		} else {
			fieldMapNew[field] = value
		}
	}

	return fieldMapNew
}

// Merge 'src' map to 'dest' map of map[string]string type
func mergeMap(dest map[string]string, src map[string]string) {
	TRACE_LOG(TRACE_SEMANTIC,
		"Merging map %v into %v", src, dest)

	for key, data := range src {
		dest[key] = data
	}
}

func (c *CVL) generateYangParserData(jsonNode *jsonquery.Node, root **yparser.YParserNode) CVLErrorInfo {

	var errObj yparser.YParserError
	topNode, cvlErrObj := c.generateTableData(true, jsonNode)

	if topNode == nil {
		cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
		CVL_LOG(WARNING, "Unable to translate request data to YANG format")
		return cvlErrObj
	}

	if *root == nil {
		*root = topNode
	} else {
		if *root, errObj = c.yp.MergeSubtree(*root, topNode); errObj.ErrCode != yparser.YP_SUCCESS {
			CVL_LOG(WARNING, "Unable to merge translated YANG data(libyang) "+
				"while translating from request data to YANG format")
			defer c.yp.FreeNode(topNode)
			cvlErrObj.ErrCode = CVL_SYNTAX_ERROR
			return cvlErrObj
		}
	}
	return cvlErrObj

}

func (c *CVL) generateYangValidatorData(jsonNode *jsonquery.Node) CVLErrorInfo {
	var cvlErrObj CVLErrorInfo
	topYangNode, cvlYErrObj := c.generateYangListData(jsonNode, true)
	if topYangNode == nil {
		cvlYErrObj.ErrCode = CVL_SYNTAX_ERROR
		CVL_LOG(WARNING, "Unable to translate request data to YANG format")
		return cvlYErrObj
	}

	//Create a full document and merge with main YANG data
	doc := &xmlquery.Node{Type: xmlquery.DocumentNode}
	doc.FirstChild = topYangNode
	doc.LastChild = topYangNode
	topYangNode.Parent = doc

	if IsTraceAllowed(TRACE_CACHE) {
		TRACE_LOG(TRACE_CACHE, "Before merge, YANG data tree = %s, source = %s",
			c.yv.root.OutputXML(false),
			doc.OutputXML(false))
	}

	if c.mergeYangData(c.yv.root, doc) != CVL_SUCCESS {
		CVL_LOG(WARNING, "Unable to merge translated YANG data while "+
			"translating from request data to YANG format")
		cvlYErrObj.ErrCode = CVL_SYNTAX_ERROR
		return cvlErrObj
	}
	if IsTraceAllowed(TRACE_CACHE) {
		TRACE_LOG(TRACE_CACHE, "After merge, YANG data tree = %s",
			c.yv.root.OutputXML(false))
	}
	return cvlErrObj
}

func (c *CVL) translateToYang(jsonMap *map[string]interface{}, buildSyntaxDOM bool) (*yparser.YParserNode, CVLErrorInfo) {

	var cvlErrObj CVLErrorInfo
	//Parse the map data to json tree
	data, _ := jsonquery.ParseJsonMap(jsonMap)
	var root *yparser.YParserNode = nil

	for jsonNode := data.FirstChild; jsonNode != nil; jsonNode = jsonNode.NextSibling {
		TRACE_LOG(TRACE_LIBYANG, "Translating, Top Node=%v\n", jsonNode.Data)
		///Generate YANG data for Yang Validator
		cvlError := c.generateYangValidatorData(jsonNode)
		if cvlError.ErrCode != CVL_SUCCESS {
			return nil, cvlError
		}
		if buildSyntaxDOM {
			//Visit each top level list in a loop for creating table data
			cvlError = c.generateYangParserData(jsonNode, &root)
			if cvlError.ErrCode != CVL_SUCCESS {
				defer c.yp.FreeNode(root)
				return nil, cvlError
			}
		}
	}

	return root, cvlErrObj
}

// Validate config - syntax and semantics
func (c *CVL) validate(data *yparser.YParserNode) CVLRetCode {

	depData := c.fetchDataToTmpCache()

	if IsTraceAllowed(TRACE_LIBYANG) {
		TRACE_LOG(TRACE_LIBYANG, "\nValidate1 data=%v\n", c.yp.NodeDump(data))
	}

	errObj, _ := c.validateSyntax(data, depData)
	if CVL_SUCCESS != errObj.ErrCode {
		return errObj.ErrCode
	}

	cvlErrObj := c.validateCfgSemantics(c.yv.root)
	if CVL_SUCCESS != cvlErrObj.ErrCode {
		return cvlErrObj.ErrCode
	}

	return CVL_SUCCESS
}

func createCVLErrObj(errObj yparser.YParserError, srcNode *jsonquery.Node) CVLErrorInfo {
	errCode := CVLRetCode(errObj.ErrCode)
	if errObj.ErrCode == yparser.YP_INTERNAL_UNKNOWN {
		errCode = CVL_INTERNAL_UNKNOWN
	}

	// YParserError may not contain table or key info when creating a field (libyang issue)
	// Fill the missing info from source json data tree
	if srcNode != nil && srcNode.Parent != nil {
		if len(errObj.TableName) == 0 {
			errObj.TableName = srcNode.Parent.Data
			errObj.Keys = splitKeyComponents(errObj.TableName, srcNode.Data)
		} else if len(errObj.Keys) == 0 && errObj.TableName == srcNode.Parent.Data {
			errObj.Keys = splitKeyComponents(errObj.TableName, srcNode.Data)
		}
	}

	cvlErrObj := CVLErrorInfo{
		TableName:        errObj.TableName,
		ErrCode:          errCode,
		CVLErrDetails:    cvlErrorMap[errCode],
		Keys:             errObj.Keys,
		Value:            errObj.Value,
		Field:            errObj.Field,
		Msg:              errObj.Msg,
		ConstraintErrMsg: errObj.ErrTxt,
		ErrAppTag:        errObj.ErrAppTag,
	}

	return cvlErrObj
}

// Perform syntax checks
func (c *CVL) validateSyntax(data, depData *yparser.YParserNode) (CVLErrorInfo, CVLRetCode) {
	var cvlErrObj CVLErrorInfo
	TRACE_LOG(TRACE_YPARSER, "Validating syntax ....")

	if errObj := c.yp.ValidateSyntax(data, depData); errObj.ErrCode != yparser.YP_SUCCESS {

		retCode := CVLRetCode(errObj.ErrCode)

		cvlErrObj = CVLErrorInfo{
			TableName:        errObj.TableName,
			ErrCode:          CVLRetCode(errObj.ErrCode),
			CVLErrDetails:    cvlErrorMap[retCode],
			Keys:             errObj.Keys,
			Value:            errObj.Value,
			Field:            errObj.Field,
			Msg:              errObj.Msg,
			ConstraintErrMsg: errObj.ErrTxt,
			ErrAppTag:        errObj.ErrAppTag,
		}

		CVL_LOG(WARNING, "Syntax validation failed. Error - %v", cvlErrObj)

		return cvlErrObj, retCode
	}

	return cvlErrObj, CVL_SUCCESS
}

// Add config data item to accumulate per table
func (c *CVL) addCfgDataItem(configData *map[string]interface{},
	cfgDataItem cmn.CVLEditConfigData) (string, string) {
	var cfgData map[string]interface{} = *configData

	tblName, key := splitRedisKey(cfgDataItem.Key)
	CVL_LOG(WARNING, "addCfgDataItem(): table: %s, key: %s", tblName, key)
	if tblName == "" || key == "" {
		//Bad redis key
		return "", ""
	}

	if _, existing := cfgData[tblName]; existing {
		fieldsMap := cfgData[tblName].(map[string]interface{})
		if cfgDataItem.VOp == cmn.OP_DELETE {
			return tblName, key
		}
		fieldsMap[key] = c.checkFieldMap(&cfgDataItem.Data)
	} else {
		fieldsMap := make(map[string]interface{})
		if cfgDataItem.VOp == cmn.OP_DELETE {
			fieldsMap[key] = nil
		} else {
			fieldsMap[key] = c.checkFieldMap(&cfgDataItem.Data)
		}
		cfgData[tblName] = fieldsMap
	}

	return tblName, key
}

// Perform user defined custom validation
func (c *CVL) doCustomValidation(node *xmlquery.Node,
	custvCfg []custv.CVLEditConfigData,
	curCustvCfg *custv.CVLEditConfigData, yangListName,
	tbl, key string) CVLErrorInfo {

	cvlErrObj := CVLErrorInfo{ErrCode: CVL_SUCCESS}

	// yangListName provides the correct table name defined in sonic-yang
	// For ex. VLAN_INTERFACE_LIST and VLAN_INTERFACE_IPADDR_LIST are in same container
	for nodeName, custFuncs := range modelInfo.tableInfo[yangListName].custValidation {
		//find the node value
		//node value is empty for custom validation function at list level
		//If key does not exist, then no need for custom validation
		nodeVal := ""
		fieldFound := false
		if !strings.HasSuffix(nodeName, "_LIST") {
			for nodeLeaf := node.FirstChild; nodeLeaf != nil; nodeLeaf = nodeLeaf.NextSibling {
				if nodeName != nodeLeaf.Data {
					continue
				}
				if (len(nodeLeaf.Attr) > 0) &&
					(nodeLeaf.Attr[0].Name.Local == "leaf-list") {
					nodeVal = curCustvCfg.Data[nodeName]
				} else {
					nodeVal = nodeLeaf.FirstChild.Data
				}
				fieldFound = true
				break
			}
			if !fieldFound {
				CVL_LOG(INFO_DEBUG, "Non list with empty entry, no need of custom validation for %s", nodeName)
				continue
			}

		}

		//Call custom validation functions
		pCustv := &custv.CustValidationCtxt{
			ReqData:   custvCfg,
			CurCfg:    curCustvCfg,
			YNodeName: nodeName,
			YNodeVal:  nodeVal,
			YCur:      node,
			SessCache: &(c.custvCache),
			RClient:   c.dbAccess}
		for _, custFunction := range custFuncs {
			CVL_LOG(INFO_TRACE, "Calling custom validation function %s", custFunction)
			errObj := custv.InvokeCustomValidation(&custv.CustomValidation{}, custFunction, pCustv)
			cvlErrObj = *(*CVLErrorInfo)(unsafe.Pointer(&errObj))
			if cvlErrObj.ErrCode != CVL_SUCCESS {
				CVL_LOG(WARNING, "Custom validation function %s - failed, Error = %v", custFunction, cvlErrObj)
				return cvlErrObj
			}
		}
	}

	return cvlErrObj
}

// getLeafRefInfo This function returns leafrefInfo structure based on table name,
// target table name and leaf node name where leafRef is present
func getLeafRefInfo(tblName, fldName, targetTblName string) *leafRefInfo {
	for _, refTblLeafRef := range modelInfo.tableInfo[tblName].leafRef[fldName] {
		if refTblLeafRef.path == "non-leafref" {
			continue
		}

		for k := range refTblLeafRef.yangListNames {
			if refTblLeafRef.yangListNames[k] == targetTblName {
				return refTblLeafRef
			}
		}
	}
	return nil
}

func isMandatoryTrueNode(tblName, field string) bool {
	if flag, exists := modelInfo.tableInfo[tblName].mandatoryNodes[field]; exists {
		return flag
	}

	return false
}

// isLeaflistHasMinElems returns true if given field is a leaf-list node with min-elements > 0.
// Field name passed MUST have the '@' suffix. Otherwise, it will be treated as a leaf node.
func isLeaflistHasMinElems(table, field string) bool {
	if n := len(field) - 1; field[n] == '@' {
		field = field[:n]
		return modelInfo.tableInfo[table].mandatoryNodes[field]
	}
	return false
}

func AddFormatterFunc(s string, f Formatter) error {
	return AddToFormatterFuncsMap(s, f)
}

func (err CVLErrorInfo) String() string {
	var s string
	if CVL_SUCCESS == err.ErrCode {
		s = "Success"
	} else {
		s = fmt.Sprintf("ErrCode[%v]: ErrDetails[%s], Msg[%s], ConstraintErrMsg[%s], Table[%s:%v], Field[%s:%s]", err.ErrCode, err.CVLErrDetails, err.Msg, err.ConstraintErrMsg, err.TableName, err.Keys, err.Field, err.Value)
	}

	return s
}

// Message returns the user defined error message if it exists; an auto formatted message otherwise.
// It may not be suitable for end user consumption always.
func (err CVLErrorInfo) Message() string {
	if err.ErrCode == CVL_SUCCESS {
		return ""
	}
	if len(err.ConstraintErrMsg) != 0 {
		return err.ConstraintErrMsg
	}
	s := err.Msg
	if len(s) == 0 {
		return "Internal error"
	}
	if len(err.TableName) != 0 {
		s += " in " + err.TableName + " table"
		if len(err.Keys) != 0 {
			s += " entry [" + strings.Join(err.Keys, " ") + "]"
		}
	}
	return s
}
