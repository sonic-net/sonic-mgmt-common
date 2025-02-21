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

package yparser

/* Yang parser using libyang library */

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	//lint:ignore ST1001 This is safe to dot import for util package
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
)

/*
#cgo LDFLAGS: -lyang
#include <libyang/libyang.h>
#include <libyang/tree_data.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

size_t golysc_ext_instance_array_count(struct lysc_ext_instance *arr)
{
	return LY_ARRAY_COUNT(arr);
}

struct lysc_ext_instance *golysc_ext_instance_array_idx(struct lysc_ext_instance *arr, size_t idx)
{
	return &arr[idx];
}

size_t golysc_must_array_count(struct lysc_must *arr)
{
	return LY_ARRAY_COUNT(arr);
}

size_t golyd_value_array_count(struct lyd_value **arr)
{
	return LY_ARRAY_COUNT(arr);
}

struct lyd_value *golyd_value_array_idx(struct lyd_value **arr, size_t idx)
{
	return arr[idx];
}

struct ly_ctx *goly_ctx_new(const char *search_dir, uint16_t options)
{
	struct ly_ctx *ctx = NULL;
	if (ly_ctx_new(search_dir, options, &ctx) != LY_SUCCESS) {
		return NULL;
	}
	return ctx;
}

struct lyd_node *golyd_new_inner(struct lyd_node *parent, const struct lys_module *module, const char *name)
{
	struct lyd_node *node = NULL;
	if (lyd_new_inner(parent, module, name, 0, &node) != LY_SUCCESS) {
		return NULL;
	}
	return node;
}

struct lyd_node *golyd_new_list2(struct lyd_node *parent, const struct lys_module *module, const char *name, const char *keylist, uint32_t options)
{
	struct lyd_node *node = NULL;

	if (lyd_new_list2(parent, module, name, keylist, options, &node) != LY_SUCCESS) {
		return NULL;
	}
	return node;
}

size_t golysc_node_list_keys_count(const struct lysc_node *node)
{
	const struct lysc_node *n;
	const struct lysc_node_list *l;
	size_t cnt = 0;

	if (node->nodetype != LYS_LIST) {
		return 0;
	}

	l = (const struct lysc_node_list *)node;

	for (n=l->child; n != NULL; n = n->next) {
		if (n->flags & LYS_KEY) {
			cnt++;
		}
	}
	return cnt;
}

const char *golysc_node_get_when(const struct lysc_node *node)
{
	struct lysc_when **when = NULL;

	switch (node->nodetype) {
	case LYS_CHOICE:
		const struct lysc_node_choice *ch = (const struct lysc_node_choice *)node;
		when = ch->when;
	case LYS_CASE:
		const struct lysc_node_case *ca = (const struct lysc_node_case *)node;
		when = ca->when;
	}

	if (when == NULL || LY_ARRAY_COUNT(when) == 0) {
		return NULL;
	}
	return lyxp_get_expr(when[0]->cond);
}

static ly_bool lysc_node_is_union(const struct lysc_node *node)
{
	struct lysc_type *type;

	if (node == NULL) {
		return 0;
	}
	if (node->nodetype == LYS_LEAF) {
		type = ((struct lysc_node_leaf *)node)->type;
	} else if (node->nodetype == LYS_LEAFLIST) {
		type = ((struct lysc_node_leaflist *)node)->type;
	} else {
		return 0;
	}

	if (type->basetype != LY_TYPE_UNION) {
		return 0;
	}

	return 1;
}

struct leaf_value {
	const char *name;
	const char *value;
};

int lyd_multi_new_leaf(struct lyd_node *parent, const struct lys_module *module,
	struct leaf_value *leafValArr, int size)
{
	const char *name, *val;
	struct lyd_node *leaf;
	struct lysc_type *type = NULL;
	int has_ptr_type = 0;
	int idx = 0;

	for (idx = 0; idx < size; idx++)
	{
		if ((leafValArr[idx].name == NULL) || (leafValArr[idx].value == NULL))
		{
			continue;
		}

		name = leafValArr[idx].name;
		val = leafValArr[idx].value;

		if (lyd_new_term(parent, module, name, val, 0, &leaf) != LY_SUCCESS)
		{
			fprintf(stderr, "lyd_multi_new_leaf(): lyd_new_term(%s, %s) failed\n", name, val);
			return -1;
		}
	}
	return 0;
}

int lyd_node_leafref_match_in_union(const struct lys_module *module, const char *xpath, const char *value)
{
	const struct lysc_node *node = NULL;
	int idx = 0;
	struct ly_set *set = NULL;

	if (module == NULL)
	{
		return -1;
	}

	if (lys_find_xpath(module->ctx, NULL, xpath, 0, &set) != LY_SUCCESS || set == NULL) {
		return -1;
	}

	if (set->count == 0) {
		ly_set_free(set, NULL);
		return -1;
	}

	node = set->snodes[0];
	ly_set_free(set, NULL);

	if (!lysc_node_is_union(node)) {
		return -1;
	}

	if (lysc_node_lref_targets(node, &set) != LY_SUCCESS || set == NULL)
	{
		return -1;
	}

	for (idx = 0; idx < set->count; idx++) {
		if (lyd_value_validate(module->ctx, set->snodes[idx], value, strlen(value), NULL, NULL, NULL) == LY_SUCCESS)
		{
			ly_set_free(set, NULL);
			return 0;
		}
	}

	ly_set_free(set, NULL);
	return -1;
}

struct lysc_xpath_targets {
	const char **xpathlist; // path list
	size_t count; // actual path count
};

void golys_xpath_targets_free(struct lysc_xpath_targets *paths)
{
	size_t i;

	if (paths == NULL) {
		return;
	}

	free(paths->xpathlist);
	free(paths);
}

static struct lysc_xpath_targets *golys_xpath_targets_alloc(size_t cnt)
{
	struct lysc_xpath_targets *paths = malloc(sizeof(*paths));
	paths->xpathlist = malloc(sizeof(*paths->xpathlist) * cnt);
	paths->count = cnt;
	return paths;
}

static const char *nonLeafRef = "non-leafref";
struct lysc_xpath_targets *golys_xpath_targets_get(const struct lysc_node *node)
{
	struct lysc_type *type;
	struct lysc_xpath_targets *paths = NULL;
	LY_ARRAY_COUNT_TYPE u;

	if (node == NULL) {
		return NULL;
	}

	type = ((struct lysc_node_leaf *)node)->type;
	if (type->basetype == LY_TYPE_UNION) {
		// union with possible leafrefs
		struct lysc_type_union *type_un = (struct lysc_type_union *)type;

		paths = golys_xpath_targets_alloc(LY_ARRAY_COUNT(type_un->types));

		LY_ARRAY_FOR(type_un->types, u) {
			struct lysc_type_leafref *lref_type;

			if (type_un->types[u]->basetype != LY_TYPE_LEAFREF) {
				paths->xpathlist[u] = nonLeafRef;
				continue;
			}

			lref_type = (struct lysc_type_leafref *)type_un->types[u];

			if (lref_type->path && lyxp_get_expr(lref_type->path) != NULL) {
				paths->xpathlist[u] = lyxp_get_expr(lref_type->path);
			} else {
				paths->xpathlist[u] = nonLeafRef;
			}
		}
	} else if (type->basetype == LY_TYPE_LEAFREF) {
		struct lysc_type_leafref *lref_type = (struct lysc_type_leafref *)type;

		paths = golys_xpath_targets_alloc(1);

		if (lref_type->path && lyxp_get_expr(lref_type->path) != NULL) {
			paths->xpathlist[0] = lyxp_get_expr(lref_type->path);
		} else {
			paths->xpathlist[0] = nonLeafRef;
		}
	}
	return paths;
}
*/
import "C"

type YParserCtx C.struct_ly_ctx
type YParserNode C.struct_lyd_node
type YParserSNode C.struct_lysc_node
type YParserModule C.struct_lys_module

var ypCtx *YParserCtx

type XpathExpression struct {
	Expr    string
	ErrCode string
	ErrStr  string
}

type WhenExpression struct {
	Expr      string   //when expression
	NodeNames []string //node names under when condition
}

// YParserListInfo Important schema information to be loaded at bootup time
type YParserListInfo struct {
	ListName        string
	Module          *YParserModule
	DbName          string
	ModelName       string
	RedisTableName  string //To which Redis table it belongs to, used for 1 Redis to N Yang List
	Keys            []string
	RedisKeyDelim   string
	RedisKeyPattern string
	RedisTableSize  int
	MapLeaf         []string            //for 'mapping  list'
	LeafRef         map[string][]string //for storing all leafrefs for a leaf in a table,
	//multiple leafref possible for union
	DfltLeafVal      map[string]string //Default value for leaf/leaf-list
	XpathExpr        map[string][]*XpathExpression
	CustValidation   map[string][]string
	WhenExpr         map[string][]*WhenExpression //multiple when expression for choice/case etc
	MandatoryNodes   map[string]bool
	DependentOnTable string //for table on which it is dependent
	Key              string //Static key, value comes from sonic-extension:tbl-key
}

type YParserLeafValue struct {
	Name  string
	Value string
}

type YParser struct {
	// Empty
}

// YParserError YParser Error Structure
type YParserError struct {
	ErrCode   YParserRetCode /* Error Code describing type of error. */
	Msg       string         /* Detailed error message. */
	ErrTxt    string         /* High level error message. */
	TableName string         /* List/Table having error */
	Keys      []string       /* Keys of the Table having error. */
	Field     string         /* Field Name throwing error . */
	Value     string         /* Field Value throwing error */
	ErrAppTag string         /* Error App Tag. */
}

type YParserRetCode int

const (
	YP_SUCCESS YParserRetCode = 1000 + iota
	YP_SYNTAX_ERROR
	YP_SEMANTIC_ERROR
	YP_SYNTAX_MISSING_FIELD
	YP_SYNTAX_INVALID_FIELD            /* Invalid Field  */
	YP_SYNTAX_INVALID_INPUT_DATA       /*Invalid Input Data */
	YP_SYNTAX_MULTIPLE_INSTANCE        /* Multiple Field Instances */
	YP_SYNTAX_DUPLICATE                /* Duplicate Fields  */
	YP_SYNTAX_ENUM_INVALID             /* Invalid enum value */
	YP_SYNTAX_ENUM_INVALID_NAME        /* Invalid enum name  */
	YP_SYNTAX_ENUM_WHITESPACE          /* Enum name with leading/trailing whitespaces */
	YP_SYNTAX_OUT_OF_RANGE             /* Value out of range/length/pattern (data) */
	YP_SYNTAX_MINIMUM_INVALID          /* min-elements constraint not honored  */
	YP_SYNTAX_MAXIMUM_INVALID          /* max-elements constraint not honored */
	YP_SEMANTIC_DEPENDENT_DATA_MISSING /* Dependent Data is missing */
	YP_SEMANTIC_MANDATORY_DATA_MISSING /* Mandatory Data is missing */
	YP_SEMANTIC_KEY_ALREADY_EXIST      /* Key already existing */
	YP_SEMANTIC_KEY_NOT_EXIST          /* Key is missing */
	YP_SEMANTIC_KEY_DUPLICATE          /* Duplicate key */
	YP_SEMANTIC_KEY_INVALID            /* Invalid key */
	YP_INTERNAL_UNKNOWN
)

// cvl-yin generator adds this prefix to all user defined error messages.
const customErrorPrefix = "[Error]"

var yparserInitialized bool = false

func TRACE_LOG(tracelevel CVLTraceLevel, fmtStr string, args ...interface{}) {
	TRACE_LEVEL_LOG(tracelevel, fmtStr, args...)
}

func CVL_LOG(level CVLLogLevel, fmtStr string, args ...interface{}) {
	CVL_LEVEL_LOG(level, fmtStr, args...)
}

// package init function
func init() {
	if os.Getenv("CVL_DEBUG") != "" {
		Debug(true)
	}
}

func Debug(on bool) {
	if on {
		C.ly_log_level(C.LY_LLDBG)
	} else {
		C.ly_log_level(C.LY_LLERR)
	}
}

func Initialize() {
	if !yparserInitialized {
		cs := C.CString(CVL_SCHEMA)
		defer C.free(unsafe.Pointer(cs))
		ypCtx = (*YParserCtx)(C.goly_ctx_new(cs, 0))
		C.ly_log_level(C.LY_LLERR)
		//	yparserInitialized = true
	}
}

func Finish() {
	if yparserInitialized {
		C.ly_ctx_destroy((*C.struct_ly_ctx)(ypCtx))
		//	yparserInitialized = false
	}
}

// ParseSchemaFile Parse YIN schema file
func ParseSchemaFile(modelFile string) (*YParserModule, YParserError) {
	var module *C.struct_lys_module
	csModelFile := C.CString(modelFile)
	defer C.free(unsafe.Pointer(csModelFile))
	if C.lys_parse_path((*C.struct_ly_ctx)(ypCtx), csModelFile, C.LYS_IN_YIN, &module) != C.LY_SUCCESS {
		return nil, getErrorDetails()
	}

	return (*YParserModule)(module), YParserError{ErrCode: YP_SUCCESS}
}

func (yp *YParser) AddContainerNode(module *YParserModule, parent *YParserNode, name string) (*YParserNode, YParserError) {
	nameCStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameCStr))
	ret := (*YParserNode)(C.golyd_new_inner((*C.struct_lyd_node)(parent), (*C.struct_lys_module)(module), (*C.char)(nameCStr)))
	if ret == nil {
		TRACE_LOG(TRACE_YPARSER, "Failed parsing node %s", name)
		return ret, getErrorDetails()
	}

	return ret, YParserError{ErrCode: YP_SUCCESS}
}

func (yp *YParser) AddListNode(module *YParserModule, parent *YParserNode, name string, keys []*YParserLeafValue) (*YParserNode, YParserError) {
	var keylist string

	// All key values predicate in the form of "[key1='val1'][key2='val2']...", they do not have to be ordered.
	for index := 0; index < len(keys); index++ {
		if (keys[index] == nil) || (keys[index].Name == "") {
			break
		}

		keylist += "["
		keylist += keys[index].Name
		keylist += "='"
		keylist += keys[index].Value
		keylist += "']"
	}

	nameCStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameCStr))
	keylistCStr := C.CString(keylist)
	defer C.free(unsafe.Pointer(keylistCStr))
	ret := (*YParserNode)(C.golyd_new_list2((*C.struct_lyd_node)(parent), (*C.struct_lys_module)(module), (*C.char)(nameCStr), (*C.char)(keylistCStr), 0))
	if ret == nil {
		TRACE_LOG(TRACE_YPARSER, "Failed parsing node %s", name)
		return ret, getErrorDetails()
	}

	return ret, YParserError{ErrCode: YP_SUCCESS}
}

// IsLeafrefMatchedInUnion Check if value matches with leafref node in union
func (yp *YParser) IsLeafrefMatchedInUnion(module *YParserModule, xpath, value string) bool {
	xpathCStr := C.CString(xpath)
	valCStr := C.CString(value)
	defer func() {
		C.free(unsafe.Pointer(xpathCStr))
		C.free(unsafe.Pointer(valCStr))
	}()
	return C.lyd_node_leafref_match_in_union((*C.struct_lys_module)(module), (*C.char)(xpathCStr), (*C.char)(valCStr)) == 0
}

// AddMultiLeafNodes dd child node to a parent node
func (yp *YParser) AddMultiLeafNodes(module *YParserModule, parent *YParserNode, multiLeaf []*YParserLeafValue) YParserError {
	if len(multiLeaf) == 0 {
		return YParserError{ErrCode: YP_SUCCESS}
	}

	leafValArr := make([]C.struct_leaf_value, len(multiLeaf))
	tmpArr := make([]*C.char, len(multiLeaf)*2)

	size := C.int(0)
	for index := 0; index < len(multiLeaf); index++ {
		if (multiLeaf[index] == nil) || (multiLeaf[index].Name == "") {
			break
		}
		//Accumulate all name/value in array to be passed in lyd_multi_new_leaf()
		nameCStr := C.CString(multiLeaf[index].Name)
		valCStr := C.CString(multiLeaf[index].Value)
		leafValArr[index].name = (*C.char)(nameCStr)
		leafValArr[index].value = (*C.char)(valCStr)
		size++

		tmpArr = append(tmpArr, (*C.char)(nameCStr))
		tmpArr = append(tmpArr, (*C.char)(valCStr))
	}

	defer func() {
		for _, cStr := range tmpArr {
			C.free(unsafe.Pointer(cStr))
		}
	}()

	if C.lyd_multi_new_leaf((*C.struct_lyd_node)(parent), (*C.struct_lys_module)(module), (*C.struct_leaf_value)(unsafe.Pointer(&leafValArr[0])), size) != 0 {
		if IsTraceAllowed(TRACE_ONERROR) {
			TRACE_LOG(TRACE_ONERROR, "Failed to create Multi Leaf Data = %v", multiLeaf)
		}
		return getErrorDetails()
	}

	return YParserError{ErrCode: YP_SUCCESS}

}

// NodeDump Return entire subtree in XML format in string
func (yp *YParser) NodeDump(root *YParserNode) string {
	if root == nil {
		return ""
	} else {
		var outBuf *C.char
		C.lyd_print_mem(&outBuf, (*C.struct_lyd_node)(root), C.LYD_JSON, C.LYD_PRINT_WITHSIBLINGS)
		defer C.free(unsafe.Pointer(outBuf))
		return C.GoString(outBuf)
	}
}

// MergeSubtree Merge source with destination
func (yp *YParser) MergeSubtree(root, node *YParserNode) (*YParserNode, YParserError) {
	rootTmp := (*C.struct_lyd_node)(root)

	if root == nil || node == nil {
		return root, YParserError{ErrCode: YP_SUCCESS}
	}

	if IsTraceAllowed(TRACE_YPARSER) {
		rootdumpStr := yp.NodeDump((*YParserNode)(rootTmp))
		TRACE_LOG(TRACE_YPARSER, "Root subtree = %v\n", rootdumpStr)
	}

	if C.lyd_merge_siblings(&rootTmp, (*C.struct_lyd_node)(node), C.LYD_MERGE_DESTRUCT) != C.LY_SUCCESS {
		return (*YParserNode)(rootTmp), getErrorDetails()
	}

	if IsTraceAllowed(TRACE_YPARSER) {
		dumpStr := yp.NodeDump((*YParserNode)(rootTmp))
		TRACE_LOG(TRACE_YPARSER, "Merged subtree = %v\n", dumpStr)
	}

	return (*YParserNode)(rootTmp), YParserError{ErrCode: YP_SUCCESS}
}

// createTempDepData merge depdata and data to create temp data. used in syntax, semantic and custom validation
func (yp *YParser) mergeDepData(data *(*C.struct_lyd_node), depData *YParserNode, destruct bool) YParserError {
	var flags C.uint16_t

	flags = 0
	if destruct {
		flags |= C.LYD_MERGE_DESTRUCT
	}

	if C.lyd_merge_siblings(data, (*C.struct_lyd_node)(depData), flags) != C.LY_SUCCESS {
		TRACE_LOG((TRACE_SYNTAX | TRACE_LIBYANG), "Unable to merge dependent data\n")
		return getErrorDetails()
	}
	return YParserError{ErrCode: YP_SUCCESS}
}

// ValidateSyntax Perform syntax checks
func (yp *YParser) ValidateSyntax(data *YParserNode, depData *YParserNode) YParserError {
	if data == nil {
		return YParserError{ErrCode: YP_INTERNAL_UNKNOWN}
	}

	dataPtr := (*C.struct_lyd_node)(data)

	if depData != nil {
		// merge dependent data for syntax validation - Update/Delete case
		// This is a destructive merge, depData is no longer valid.
		err := yp.mergeDepData(&dataPtr, depData, true)
		depData = nil
		if err.ErrCode != YP_SUCCESS {
			return err
		}
	}

	//Just validate syntax
	if C.lyd_validate_all(&dataPtr, (*C.struct_ly_ctx)(ypCtx), C.LYD_VALIDATE_PRESENT|C.LYD_VALIDATE_NO_STATE|C.LYD_VALIDATE_NOEXTDEPS, nil) != C.LY_SUCCESS {
		if IsTraceAllowed(TRACE_ONERROR) {
			strData := yp.NodeDump((*YParserNode)(dataPtr))
			TRACE_LOG(TRACE_ONERROR, "Failed to validate Syntax, data = %v", strData)
		}
		return getErrorDetails()
	}

	return YParserError{ErrCode: YP_SUCCESS}
}

func (yp *YParser) FreeNode(node *YParserNode) YParserError {
	if node != nil {
		C.lyd_free_all((*C.struct_lyd_node)(node))
		node = nil
	}

	return YParserError{ErrCode: YP_SUCCESS}
}

/* This function translates LIBYANG error code to valid YPARSER error code. */
func translateLYErrToYParserErr(LYErrcode int, apptag string, msg string) YParserRetCode {
	var ypErrCode YParserRetCode

	// YP_SYNTAX_MISSING_FIELD
	// YP_SYNTAX_INVALID_FIELD            /* Invalid Field  */
	// YP_SYNTAX_INVALID_INPUT_DATA       /* Invalid Input Data */
	// YP_SYNTAX_MULTIPLE_INSTANCE        /* Multiple Field Instances */
	// YP_SYNTAX_DUPLICATE                /* Duplicate Fields  */
	// YP_SYNTAX_ENUM_INVALID             /* Invalid enum value */
	// YP_SYNTAX_ENUM_INVALID_NAME        /* Invalid enum name  */
	// YP_SYNTAX_ENUM_WHITESPACE          /* Enum name with leading/trailing whitespaces */
	// YP_SYNTAX_OUT_OF_RANGE             /* Value out of range/length/pattern (data) */
	// YP_SYNTAX_MINIMUM_INVALID          /* min-elements constraint not honored  */
	// YP_SYNTAX_MAXIMUM_INVALID          /* max-elements constraint not honored */
	// YP_SEMANTIC_DEPENDENT_DATA_MISSING /* Dependent Data is missing */
	// YP_SEMANTIC_MANDATORY_DATA_MISSING /* Mandatory Data is missing */
	// YP_SEMANTIC_KEY_ALREADY_EXIST      /* Key already existing */
	// YP_SEMANTIC_KEY_NOT_EXIST          /* Key is missing */
	// YP_SEMANTIC_KEY_DUPLICATE          /* Duplicate key */
	// YP_SEMANTIC_KEY_INVALID            /* Invalid key */
	// YP_INTERNAL_UNKNOWN

	switch LYErrcode {
	case C.LYVE_SUCCESS: /**< no error */
		ypErrCode = YP_SUCCESS
	case C.LYVE_SYNTAX: /**< generic syntax error */
		ypErrCode = YP_SYNTAX_INVALID_INPUT_DATA
	case C.LYVE_SYNTAX_YANG: /**< YANG-related syntax error */
		ypErrCode = YP_SYNTAX_INVALID_INPUT_DATA
	case C.LYVE_SYNTAX_YIN: /**< YIN-related syntax error */
		ypErrCode = YP_SYNTAX_INVALID_INPUT_DATA
	case C.LYVE_REFERENCE: /**< invalid referencing or using an item */
		ypErrCode = YP_SEMANTIC_DEPENDENT_DATA_MISSING
	case C.LYVE_XPATH: /**< invalid XPath expression */
		ypErrCode = YP_SEMANTIC_KEY_NOT_EXIST
	case C.LYVE_SEMANTICS: /**< generic semantic error */
		ypErrCode = YP_SEMANTIC_KEY_INVALID
	case C.LYVE_SYNTAX_XML: /**< XML-related syntax error */
		ypErrCode = YP_SYNTAX_INVALID_FIELD
	case C.LYVE_SYNTAX_JSON: /**< JSON-related syntax error */
		ypErrCode = YP_SYNTAX_INVALID_FIELD
	case C.LYVE_DATA: /**< YANG data does not reflect some of the module restrictions */
		if apptag == "too-few-elements" {
			ypErrCode = YP_SYNTAX_MINIMUM_INVALID
		} else if apptag == "too-many-elements" {
			ypErrCode = YP_SYNTAX_MAXIMUM_INVALID
		} else if strings.HasPrefix(msg, "Invalid enumeration value") {
			ypErrCode = YP_SYNTAX_ENUM_INVALID
		} else if strings.HasPrefix(msg, "Unsatisfied") {
			ypErrCode = YP_SYNTAX_OUT_OF_RANGE
		} else if strings.HasPrefix(msg, "Mandatory") {
			ypErrCode = YP_SYNTAX_MISSING_FIELD
		} else {
			ypErrCode = YP_SYNTAX_INVALID_INPUT_DATA
		}
	case C.LYVE_OTHER:
		ypErrCode = YP_INTERNAL_UNKNOWN
	default:
		ypErrCode = YP_INTERNAL_UNKNOWN
	}
	return ypErrCode
}

/* This function performs parsing and processing of LIBYANG error messages. */
func getErrorDetails() YParserError {
	var key []string
	var errtableName string
	var ElemVal string
	var errMessage string
	var ElemName string
	var errText string
	var ypErrCode YParserRetCode = YP_INTERNAL_UNKNOWN
	var errMsg, errPath, errAppTag string

	ctx := (*C.struct_ly_ctx)(ypCtx)
	ypErrLast := C.ly_err_last(ctx)

	if ypErrLast == nil {
		return YParserError{
			ErrCode: ypErrCode,
		}
	}

	if ypErrLast.err == C.LY_SUCCESS {
		return YParserError{
			ErrCode: YP_SUCCESS,
		}
	}

	errMsg = C.GoString(ypErrLast.msg)
	if ypErrLast.data_path != nil {
		errPath = C.GoString(ypErrLast.data_path)
	} else {
		errPath = C.GoString(ypErrLast.schema_path)
	}
	errAppTag = C.GoString(ypErrLast.apptag)

	// Try to resolve table, keys and field name from the error path.
	errtableName, key, ElemName = parseLyPath(errPath)

	if !strings.HasPrefix(errMsg, customErrorPrefix) {
		// libyang generated error message.. try to extract the field value & name
		ElemVal = parseLyMessage(errMsg, lyBadValue, lyUnsatisfied)
		if len(ElemName) == 0 { // if not resolved from path
			ElemName = parseLyMessage(errMsg, lyMandatory)
		}
	} else {
		/* Custom contraint error message like in must statement.
		This can be used by App to display to user.
		*/
		errText = errMsg[len(customErrorPrefix):]
	}

	switch ypErrLast.err {
	case C.LY_EVALID:
		// validation failure
		ypErrCode = translateLYErrToYParserErr(int(ypErrLast.vecode), errAppTag, errMsg)
		if len(ElemName) != 0 {
			errMessage = "Field \"" + ElemName + "\" has invalid value"
			if len(ElemVal) != 0 {
				errMessage += " " + strconv.Quote(ElemVal)
			}
		} else {
			errMessage = "Data validation failed"
		}

	case C.LY_EINVAL:
		// invalid node. With our usage it will be the field name.
		ypErrCode = YP_SYNTAX_ERROR
		if field := parseLyMessage(errMsg, lyUnknownElem); len(field) != 0 {
			ElemName = field
			errMessage = "Unknown field \"" + field + "\""
		} else {
			errMessage = "Invalid value"
		}

	case C.LY_EMEM:
		errMessage = "Resources exhausted"

	default:
		errMessage = "Internal error"
	}

	errObj := YParserError{
		TableName: errtableName,
		ErrCode:   ypErrCode,
		Keys:      key,
		Value:     ElemVal,
		Field:     ElemName,
		Msg:       errMessage,
		ErrTxt:    errText,
		ErrAppTag: errAppTag,
	}

	TRACE_LOG(TRACE_YPARSER, "YParser error details: %v...", errObj)

	return errObj
}

func GetModelNs(module *YParserModule) (ns, prefix string) {
	return C.GoString(((*C.struct_lys_module)(module)).ns),
		C.GoString(((*C.struct_lys_module)(module)).prefix)
}

// Get model details for child under list/choice/case
func getModelChildInfo(l *YParserListInfo, node *C.struct_lysc_node, module *YParserModule,
	underWhen bool, whenExpr *WhenExpression) {

	for sChild := C.lysc_node_child(node); sChild != nil; sChild = sChild.next {
		switch sChild.nodetype {
		case C.LYS_LIST:
			keysCnt := C.golysc_node_list_keys_count(sChild)
			if keysCnt == 1 {
				// fetch key leaf
				for sChildInner := C.lysc_node_child(sChild); sChildInner != nil; sChildInner = sChildInner.next {
					if sChildInner.nodetype == C.LYS_LEAF && (sChildInner.flags&C.LYS_KEY) != 0 {
						keyName := C.GoString(sChildInner.name)
						l.MapLeaf = append(l.MapLeaf, keyName)
						break
					}
				}
				// Now, find and add the first non-key leaf.
				for sChildInner := C.lysc_node_child(sChild); sChildInner != nil; sChildInner = sChildInner.next {
					if sChildInner.nodetype == C.LYS_LEAF && (sChildInner.flags&C.LYS_KEY) == 0 {
						name := C.GoString(sChildInner.name)
						l.MapLeaf = append(l.MapLeaf, name)
						break
					}
				}
			} else { // should never hit here, as linter does the validation
				listName := C.GoString(sChild.name)
				TRACE_LOG(TRACE_YPARSER, "Inner List %s for Dynamic fields has %d keys", listName, keysCnt)
			}
		case C.LYS_CHOICE, C.LYS_CASE:
			when := C.golysc_node_get_when(sChild)
			if when != nil {
				cWhenExp := WhenExpression{
					Expr: C.GoString(when),
				}
				listName := l.ListName + "_LIST"
				l.WhenExpr[listName] = append(l.WhenExpr[listName],
					&cWhenExp)
				getModelChildInfo(l, sChild, module, true, &cWhenExp)
			} else {
				if underWhen && sChild.nodetype == C.LYS_CASE {
					// Why only nodetype == C.LYS_CASE? old code was like that
					getModelChildInfo(l, sChild, module, underWhen, whenExpr)
				} else {
					getModelChildInfo(l, sChild, module, false, nil)
				}
			}
		case C.LYS_LEAF, C.LYS_LEAFLIST:
			leafName := C.GoString(sChild.name)
			var nodeMusts (*C.struct_lysc_must)
			var nodeWhen (**C.struct_lysc_when)
			if sChild.nodetype == C.LYS_LEAF {
				sleaf := (*C.struct_lysc_node_leaf)(unsafe.Pointer(sChild))
				nodeMusts = sleaf.musts
				nodeWhen = sleaf.when
				if sleaf.dflt != nil {
					l.DfltLeafVal[leafName] = C.GoString(C.lyd_value_get_canonical((*C.struct_ly_ctx)(ypCtx), sleaf.dflt))
				}
			} else {
				sLeafList := (*C.struct_lysc_node_leaflist)(unsafe.Pointer(sChild))
				nodeMusts = sLeafList.musts
				nodeWhen = sLeafList.when
				if sLeafList.dflts != nil {
					tmpValStr := ""
					for idx := 0; idx < int(C.golyd_value_array_count(sLeafList.dflts)); idx++ {
						if idx > 0 {
							//Separate multiple values by ,
							tmpValStr = tmpValStr + ","
						}

						tmpValStr = tmpValStr + C.GoString(C.lyd_value_get_canonical((*C.struct_ly_ctx)(ypCtx), C.golyd_value_array_idx(sLeafList.dflts, (C.size_t)(idx))))
					}

					//Remove last ','
					l.DfltLeafVal[leafName] = tmpValStr
				}

				// leaf-list with min-elements > 0 should be treated as a mandatory node.
				// Reusing MandatoryNodes map itself to store this info.. Different error codes
				// are needed for min-elements and mandatory true violations. Cvl will have to
				// rely on the "@" field name suffix in db dataMap to differentiate.
				if sLeafList.min > 0 {
					l.MandatoryNodes[leafName] = true
				}
			}

			//If parent has when expression,
			//just add leaf to when expression node list
			if underWhen {
				whenExpr.NodeNames = append(whenExpr.NodeNames, leafName)
			}

			//Check for leafref expression
			leafRefs := C.golys_xpath_targets_get(sChild)
			defer C.golys_xpath_targets_free(leafRefs)
			if leafRefs != nil {
				leafRefPaths := (*[256]*C.char)(unsafe.Pointer(leafRefs.xpathlist))
				for idx := 0; idx < int(leafRefs.count); idx++ {
					path := rewriteXPathPrefix(module, C.GoString(leafRefPaths[idx]))
					l.LeafRef[leafName] = append(l.LeafRef[leafName], path)
				}
			}

			//Check for must expression; one must expession only per leaf
			if nodeMusts != nil {
				must := (*[20]C.struct_lysc_must)(unsafe.Pointer(nodeMusts))
				for idx := 0; idx < int(C.golysc_must_array_count(nodeMusts)); idx++ {
					mustexpr := rewriteXPathPrefix(module, C.GoString(C.lyxp_get_expr(must[idx].cond)))
					exp := XpathExpression{Expr: mustexpr}
					if must[idx].eapptag != nil {
						exp.ErrCode = C.GoString(must[idx].eapptag)
					}
					if must[idx].emsg != nil {
						exp.ErrStr = strings.TrimPrefix(C.GoString(must[idx].emsg), customErrorPrefix)
					}

					l.XpathExpr[leafName] = append(l.XpathExpr[leafName],
						&exp)
				}
			}

			//Check for when expression
			if nodeWhen != nil {
				when := (*[20]*C.struct_lysc_when)(unsafe.Pointer(nodeWhen))
				whenexpr := rewriteXPathPrefix(module, C.GoString(C.lyxp_get_expr(when[0].cond)))
				l.WhenExpr[leafName] = append(l.WhenExpr[leafName],
					&WhenExpression{
						Expr:      whenexpr,
						NodeNames: []string{leafName},
					})
			}

			//Check for custom extension
			if sChild.exts != nil {
				for idx := 0; idx < int(C.golysc_ext_instance_array_count(sChild.exts)); idx++ {
					ext := C.golysc_ext_instance_array_idx(sChild.exts, (C.size_t)(idx))
					if C.GoString(ext.def.name) == "custom-validation" {
						argVal := C.GoString(ext.argument)
						if argVal != "" {
							l.CustValidation[leafName] = append(l.CustValidation[leafName], argVal)
						}
					}
				}
			}

			// check for mandatory flag
			if (sChild.flags & C.LYS_MAND_MASK) == C.LYS_MAND_TRUE {
				l.MandatoryNodes[leafName] = true
			} else if (sChild.flags & C.LYS_MAND_MASK) == C.LYS_MAND_FALSE {
				l.MandatoryNodes[leafName] = false
			}
		}
	}
}

// GetModelListInfo Get model info for YANG list and its subtree
func GetModelListInfo(module *YParserModule) []*YParserListInfo {
	var list []*YParserListInfo

	mod := (*C.struct_lys_module)(module)
	if mod == nil || mod.compiled == nil || mod.compiled.data == nil {
		return nil
	}

	// Each container has a base container, then a set of containers under that.
	// We need to skip over the base container
	if mod.compiled.data.nodetype != C.LYS_CONTAINER {
		return nil
	}
	cbase := (*C.struct_lysc_node_container)(unsafe.Pointer(mod.compiled.data))

	for snode := cbase.child; snode != nil; snode = snode.next { //for each container
		if snode.nodetype != C.LYS_CONTAINER {
			continue
		}
		snodec := (*C.struct_lysc_node_container)(unsafe.Pointer(snode))

		//for each list
		for n := snodec.child; n != nil; n = n.next {
			var l YParserListInfo
			slist := (*C.struct_lysc_node_list)(unsafe.Pointer(n))
			listName := C.GoString(n.name)
			l.RedisTableName = C.GoString(snode.name)

			tableName := listName
			if strings.HasSuffix(tableName, "_LIST") {
				tableName = tableName[0 : len(tableName)-len("_LIST")]
			}
			l.ListName = tableName
			l.ModelName = C.GoString(mod.name)
			//Default database is CONFIG_DB since CVL works with config db mainly
			l.Module = module
			l.DbName = "CONFIG_DB"
			//default delim '|'
			l.RedisKeyDelim = "|"
			//Default table size is -1 i.e. size limit
			l.RedisTableSize = -1
			if slist.max > 0 {
				l.RedisTableSize = int(slist.max)
			}

			l.LeafRef = make(map[string][]string)
			l.XpathExpr = make(map[string][]*XpathExpression)
			l.CustValidation = make(map[string][]string)
			l.WhenExpr = make(map[string][]*WhenExpression)
			l.DfltLeafVal = make(map[string]string)
			l.MandatoryNodes = make(map[string]bool)

			//Add keys
			for child := slist.child; child != nil; child = child.next {
				if (child.flags & C.LYS_KEY) != 0 {
					l.Keys = append(l.Keys, C.GoString(child.name))
				}
			}

			//Check for must expression
			if slist.musts != nil {
				must := (*[10]C.struct_lysc_must)(unsafe.Pointer(slist.musts))
				for idx := 0; idx < int(C.golysc_must_array_count(slist.musts)); idx++ {
					mustexp := rewriteXPathPrefix(module, C.GoString(C.lyxp_get_expr(must[idx].cond)))
					exp := XpathExpression{Expr: mustexp}
					if must[idx].eapptag != nil {
						exp.ErrCode = C.GoString(must[idx].eapptag)
					}
					if must[idx].emsg != nil {
						exp.ErrStr = strings.TrimPrefix(C.GoString(must[idx].emsg), customErrorPrefix)
					}

					l.XpathExpr[listName] = append(l.XpathExpr[listName],
						&exp)
				}
			}

			//Check for custom extension
			if n.exts != nil {
				for idx := 0; idx < int(C.golysc_ext_instance_array_count(n.exts)); idx++ {
					ext := C.golysc_ext_instance_array_idx(n.exts, (C.size_t)(idx))
					extName := C.GoString(ext.def.name)
					argVal := C.GoString(ext.argument)
					switch extName {
					case "custom-validation":
						if argVal != "" {
							l.CustValidation[listName] = append(l.CustValidation[listName], argVal)
						}
					case "db-name":
						l.DbName = argVal
					case "key-delim":
						l.RedisKeyDelim = argVal
					case "key-pattern":
						l.RedisKeyPattern = argVal
					case "dependent-on":
						l.DependentOnTable = argVal
					case "tbl-key":
						l.Key = argVal
					}
				}

			}

			//Add default key pattern
			if l.RedisKeyPattern == "" {
				keyPattern := []string{tableName}
				for idx := 0; idx < len(l.Keys); idx++ {
					keyPattern = append(keyPattern, fmt.Sprintf("{%s}", l.Keys[idx]))
				}
				l.RedisKeyPattern = strings.Join(keyPattern, l.RedisKeyDelim)
			}

			getModelChildInfo(&l,
				(*C.struct_lysc_node)(unsafe.Pointer(slist)), module, false, nil)

			list = append(list, &l)
		} //each list inside a container
	} //each container
	return list
}
