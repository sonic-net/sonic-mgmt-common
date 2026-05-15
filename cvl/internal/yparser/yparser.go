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

// Canonical typedefs over libyang's schema node, must, when, ext_instance
// and per-nodetype subtype structs, so the rest of the file refers to each
// type by a single yp_* name.
typedef struct lysc_node yp_snode_t;
typedef struct lysc_node_leaf yp_snode_leaf_t;
typedef struct lysc_node_leaflist yp_snode_leaflist_t;
typedef struct lysc_node_list yp_snode_list_t;
typedef struct lysc_node_container yp_snode_container_t;
typedef struct lysc_must yp_must_t;
typedef struct lysc_when yp_when_t;
typedef struct lysc_ext_instance yp_ext_t;

// YPC_* mirrors of YParser{Ret,Err}Code values for use from C in
// yp_translate_validation_code(). Values must match the Go constants in
// yparser.go.
#define YPC_SUCCESS                         1000
#define YPC_SYNTAX_ERROR                    1001
#define YPC_SEMANTIC_ERROR                  1002
#define YPC_SYNTAX_MISSING_FIELD            1003
#define YPC_SYNTAX_INVALID_FIELD            1004
#define YPC_SYNTAX_INVALID_INPUT_DATA       1005
#define YPC_SYNTAX_MULTIPLE_INSTANCE        1006
#define YPC_SYNTAX_DUPLICATE                1007
#define YPC_SYNTAX_ENUM_INVALID             1008
#define YPC_SYNTAX_ENUM_INVALID_NAME        1009
#define YPC_SYNTAX_ENUM_WHITESPACE          1010
#define YPC_SYNTAX_OUT_OF_RANGE             1011
#define YPC_SYNTAX_MINIMUM_INVALID          1012
#define YPC_SYNTAX_MAXIMUM_INVALID          1013
#define YPC_SEMANTIC_DEPENDENT_DATA_MISSING 1014
#define YPC_SEMANTIC_MANDATORY_DATA_MISSING 1015
#define YPC_SEMANTIC_KEY_ALREADY_EXIST      1016
#define YPC_SEMANTIC_KEY_NOT_EXIST          1017
#define YPC_SEMANTIC_KEY_DUPLICATE          1018
#define YPC_SEMANTIC_KEY_INVALID            1019
#define YPC_INTERNAL_UNKNOWN                1020

struct ly_ctx *goly_ctx_new(const char *search_dir, uint16_t options)
{
	struct ly_ctx *ctx = NULL;
	if (ly_ctx_new(search_dir, options, &ctx) != LY_SUCCESS) {
		return NULL;
	}
	return ctx;
}

// Thin wrapper around lys_parse_path so the caller does not have to deal
// with libyang's return-value-vs-out-param convention directly.
int goly_parse_path(struct ly_ctx *ctx, const char *path, struct lys_module **module)
{
	return lys_parse_path(ctx, path, LYS_IN_YIN, module);
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

struct leaf_value {
	const char *name;
	const char *value;
};

int lyd_multi_new_leaf(struct lyd_node *parent, const struct lys_module *module,
	struct leaf_value *leafValArr, int size)
{
	const char *name, *val;
	struct lyd_node *leaf;
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

// Result type for golys_xpath_targets_get. The struct itself is not
// libyang-specific so it lives outside the #if; the path-extraction logic
// however differs and is gated below.
struct lysc_xpath_targets {
	const char **xpathlist; // path list
	size_t count; // actual path count
};

void golys_xpath_targets_free(struct lysc_xpath_targets *paths)
{
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

// ----- yp_* accessors ----------------------------------------------------
// Thin accessors over libyang's schema structs. They keep Go from poking at
// libyang struct fields directly; signatures use the yp_* typedefs defined
// above.

// Get first child of a schema node (container/list/choice/case).
const yp_snode_t *yp_node_child(const yp_snode_t *n)
{
	return lysc_node_child(n);
}

// Is this leaf node a key of its parent list?
int yp_node_is_key(const yp_snode_t *n)
{
	return (n != NULL && (n->flags & LYS_KEY)) ? 1 : 0;
}

// Extensions on a schema node.
size_t yp_node_exts_count(const yp_snode_t *n)
{
	if (n == NULL || n->exts == NULL) {
		return 0;
	}
	return LY_ARRAY_COUNT(n->exts);
}
const char *yp_node_ext_def_name(const yp_snode_t *n, size_t idx)
{
	if (n == NULL || n->exts == NULL) {
		return NULL;
	}
	return n->exts[idx].def->name;
}
const char *yp_node_ext_argument(const yp_snode_t *n, size_t idx)
{
	if (n == NULL || n->exts == NULL) {
		return NULL;
	}
	return n->exts[idx].argument;
}

// Must arrays on leaf / leaflist / list / container nodes.
yp_must_t *yp_node_musts(const yp_snode_t *n)
{
	if (n == NULL) {
		return NULL;
	}
	switch (n->nodetype) {
	case LYS_LEAF:
		return ((const yp_snode_leaf_t *)n)->musts;
	case LYS_LEAFLIST:
		return ((const yp_snode_leaflist_t *)n)->musts;
	case LYS_LIST:
		return ((const yp_snode_list_t *)n)->musts;
	case LYS_CONTAINER:
		return ((const yp_snode_container_t *)n)->musts;
	}
	return NULL;
}
size_t yp_node_musts_count(const yp_snode_t *n)
{
	yp_must_t *m = yp_node_musts(n);
	return m ? LY_ARRAY_COUNT(m) : 0;
}
const char *yp_node_must_cond_at(yp_must_t *musts, size_t idx)
{
	return lyxp_get_expr(musts[idx].cond);
}
const char *yp_node_must_apptag_at(yp_must_t *musts, size_t idx)
{
	return musts[idx].eapptag;
}
const char *yp_node_must_emsg_at(yp_must_t *musts, size_t idx)
{
	return musts[idx].emsg;
}

// Default value of a leaf as the canonical string (NULL if none).
const char *yp_leaf_dflt(struct ly_ctx *ctx, const yp_snode_t *n)
{
	const yp_snode_leaf_t *leaf;
	if (n == NULL || n->nodetype != LYS_LEAF) {
		return NULL;
	}
	leaf = (const yp_snode_leaf_t *)n;
	if (leaf->dflt == NULL) {
		return NULL;
	}
	return lyd_value_get_canonical(ctx, leaf->dflt);
}
size_t yp_leaflist_dflts_count(const yp_snode_t *n)
{
	const yp_snode_leaflist_t *ll;
	if (n == NULL || n->nodetype != LYS_LEAFLIST) {
		return 0;
	}
	ll = (const yp_snode_leaflist_t *)n;
	if (ll->dflts == NULL) {
		return 0;
	}
	return LY_ARRAY_COUNT(ll->dflts);
}
const char *yp_leaflist_dflt_at(struct ly_ctx *ctx, const yp_snode_t *n, size_t idx)
{
	const yp_snode_leaflist_t *ll;
	if (n == NULL || n->nodetype != LYS_LEAFLIST) {
		return NULL;
	}
	ll = (const yp_snode_leaflist_t *)n;
	return lyd_value_get_canonical(ctx, ll->dflts[idx]);
}

// min-elements and max-elements field names are stable across libyang
// versions; just need the typedef-aware cast.
uint32_t yp_leaflist_min(const yp_snode_t *n)
{
	if (n == NULL || n->nodetype != LYS_LEAFLIST) {
		return 0;
	}
	return ((const yp_snode_leaflist_t *)n)->min;
}

uint32_t yp_list_max(const yp_snode_t *n)
{
	if (n == NULL || n->nodetype != LYS_LIST) {
		return 0;
	}
	return ((const yp_snode_list_t *)n)->max;
}

// Returns the first "when" expression on a leaf/leaflist (NULL if none).
const char *yp_leaf_when_cond(const yp_snode_t *n)
{
	yp_when_t **when = NULL;
	if (n == NULL) {
		return NULL;
	}
	switch (n->nodetype) {
	case LYS_LEAF:
		when = ((const yp_snode_leaf_t *)n)->when;
		break;
	case LYS_LEAFLIST:
		when = ((const yp_snode_leaflist_t *)n)->when;
		break;
	}
	if (when == NULL || LY_ARRAY_COUNT(when) == 0) {
		return NULL;
	}
	return lyxp_get_expr(when[0]->cond);
}

// Returns the top-level container under module that holds the data lists,
// or NULL if the module's top isn't a container.
const yp_snode_t *yp_module_top_container(const struct lys_module *mod)
{
	if (mod == NULL || mod->compiled == NULL || mod->compiled->data == NULL) {
		return NULL;
	}
	if (mod->compiled->data->nodetype != LYS_CONTAINER) {
		return NULL;
	}
	return mod->compiled->data;
}

// Aggregated error info filled in by yp_get_last_error.
struct yp_error_info {
	uint32_t err;     // overall error code (LY_EVALID, LY_EINVAL, LY_EMEM, ...)
	uint32_t vecode;  // LYVE_* validation sub-code
	const char *msg;
	const char *path;
	const char *apptag;
};

// Fills *info with details of the last error on ctx. Return values:
//   0  no error item available
//   1  last error item is LY_SUCCESS (no real error)
//   2  real error, *info populated
int yp_get_last_error(struct ly_ctx *ctx, struct yp_error_info *info)
{
	const struct ly_err_item *err = ly_err_last(ctx);
	if (err == NULL) {
		return 0;
	}
	if (err->err == LY_SUCCESS) {
		return 1;
	}
	info->err = err->err;
	info->vecode = err->vecode;
	info->msg = err->msg;
	info->path = err->data_path ? err->data_path : err->schema_path;
	info->apptag = err->apptag;
	return 2;
}

// ---- yp_* wrappers for the data-tree functions used by Go ----
// Wrap the libyang functions whose flag enums and out-param conventions are
// awkward to spell directly from Go, so the Go-side call sites stay short.

void yp_ly_set_loglevel(int level)
{
	ly_log_level(level);
}

void yp_ly_ctx_destroy(struct ly_ctx *ctx)
{
	ly_ctx_destroy(ctx);
}

void yp_lyd_free(struct lyd_node *node)
{
	if (node != NULL) {
		lyd_free_all(node);
	}
}
char *yp_lyd_print_mem(struct lyd_node *node)
{
	char *out = NULL;
	lyd_print_mem(&out, node, LYD_JSON, LYD_PRINT_WITHSIBLINGS);
	return out;
}
int yp_lyd_merge(struct lyd_node **dst, struct lyd_node *src, int destruct, struct ly_ctx *ctx)
{
	uint16_t flags = destruct ? LYD_MERGE_DESTRUCT : 0;
	(void)ctx;
	return lyd_merge_siblings(dst, src, flags);
}
int yp_lyd_validate_edit(struct ly_ctx *ctx, struct lyd_node **data)
{
	return lyd_validate_all(data, ctx,
		LYD_VALIDATE_PRESENT | LYD_VALIDATE_NO_STATE | LYD_VALIDATE_NOEXTDEPS,
		NULL);
}

// Map a libyang validation error code to one of the YPC_* mirror values.
// The Go side casts the result to YParserRetCode directly. Lives in C so
// the switch on libyang's LYVE_* enumeration is self-contained.
int yp_translate_validation_code(int vecode, const char *apptag, const char *msg)
{
	switch (vecode) {
	case LYVE_SUCCESS:
		return YPC_SUCCESS;
	case LYVE_SYNTAX:
	case LYVE_SYNTAX_YANG:
	case LYVE_SYNTAX_YIN:
		return YPC_SYNTAX_INVALID_INPUT_DATA;
	case LYVE_REFERENCE:
		return YPC_SEMANTIC_DEPENDENT_DATA_MISSING;
	case LYVE_XPATH:
		return YPC_SEMANTIC_KEY_NOT_EXIST;
	case LYVE_SEMANTICS:
		return YPC_SEMANTIC_KEY_INVALID;
	case LYVE_SYNTAX_XML:
	case LYVE_SYNTAX_JSON:
		return YPC_SYNTAX_INVALID_FIELD;
	case LYVE_DATA:
		if (apptag != NULL && strcmp(apptag, "too-few-elements") == 0) {
			return YPC_SYNTAX_MINIMUM_INVALID;
		}
		if (apptag != NULL && strcmp(apptag, "too-many-elements") == 0) {
			return YPC_SYNTAX_MAXIMUM_INVALID;
		}
		if (msg != NULL && strncmp(msg, "Invalid enumeration value", 25) == 0) {
			return YPC_SYNTAX_ENUM_INVALID;
		}
		if (msg != NULL && strncmp(msg, "Unsatisfied", 11) == 0) {
			return YPC_SYNTAX_OUT_OF_RANGE;
		}
		if (msg != NULL && strncmp(msg, "Mandatory", 9) == 0) {
			return YPC_SYNTAX_MISSING_FIELD;
		}
		return YPC_SYNTAX_INVALID_INPUT_DATA;
	}
	return YPC_INTERNAL_UNKNOWN;
}

*/
import "C"

type YParserCtx C.struct_ly_ctx
type YParserNode C.struct_lyd_node
type YParserSNode C.yp_snode_t
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
		C.yp_ly_set_loglevel(C.LY_LLDBG)
	} else {
		C.yp_ly_set_loglevel(C.LY_LLERR)
	}
}

func Initialize() {
	if !yparserInitialized {
		cs := C.CString(CVL_SCHEMA)
		defer C.free(unsafe.Pointer(cs))
		ypCtx = (*YParserCtx)(C.goly_ctx_new(cs, 0))
		C.yp_ly_set_loglevel(C.LY_LLERR)
		//	yparserInitialized = true
	}
}

func Finish() {
	if yparserInitialized {
		C.yp_ly_ctx_destroy((*C.struct_ly_ctx)(ypCtx))
		//	yparserInitialized = false
	}
}

// ParseSchemaFile Parse YIN schema file
func ParseSchemaFile(modelFile string) (*YParserModule, YParserError) {
	var module *C.struct_lys_module
	csModelFile := C.CString(modelFile)
	defer C.free(unsafe.Pointer(csModelFile))
	if C.goly_parse_path((*C.struct_ly_ctx)(ypCtx), csModelFile, &module) != 0 {
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

// NodeDump Return entire subtree as a serialized string.
func (yp *YParser) NodeDump(root *YParserNode) string {
	if root == nil {
		return ""
	}
	outBuf := C.yp_lyd_print_mem((*C.struct_lyd_node)(root))
	defer C.free(unsafe.Pointer(outBuf))
	return C.GoString(outBuf)
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

	if C.yp_lyd_merge(&rootTmp, (*C.struct_lyd_node)(node), 1, (*C.struct_ly_ctx)(ypCtx)) != C.LY_SUCCESS {
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
	d := C.int(0)
	if destruct {
		d = 1
	}
	if C.yp_lyd_merge(data, (*C.struct_lyd_node)(depData), d, (*C.struct_ly_ctx)(ypCtx)) != C.LY_SUCCESS {
		TRACE_LOG((TRACE_SYNTAX | TRACE_LIBYANG), "Unable to merge dependent data\n")
		return getErrorDetails()
	}
	return YParserError{ErrCode: YP_SUCCESS}
}

// ValidateSyntax Perform syntax checks
func (yp *YParser) ValidateSyntax(data *YParserNode, depData *YParserNode) YParserError {
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
	if C.yp_lyd_validate_edit((*C.struct_ly_ctx)(ypCtx), &dataPtr) != C.LY_SUCCESS {
		if IsTraceAllowed(TRACE_ONERROR) {
			strData := yp.NodeDump((*YParserNode)(dataPtr))
			TRACE_LOG(TRACE_ONERROR, "Failed to validate Syntax, data = %v", strData)
		}
		return getErrorDetails()
	}

	return YParserError{ErrCode: YP_SUCCESS}
}

func (yp *YParser) FreeNode(node *YParserNode) YParserError {
	C.yp_lyd_free((*C.struct_lyd_node)(node))
	return YParserError{ErrCode: YP_SUCCESS}
}

/* This function translates LIBYANG error code to valid YPARSER error code.
 * The translation table itself lives in C (yp_translate_validation_code)
 * so the switch over libyang's LYVE_* enum stays in one place. */
func translateLYErrToYParserErr(LYErrcode int, apptag string, msg string) YParserRetCode {
	var apptagCstr, msgCstr *C.char
	if apptag != "" {
		apptagCstr = C.CString(apptag)
		defer C.free(unsafe.Pointer(apptagCstr))
	}
	if msg != "" {
		msgCstr = C.CString(msg)
		defer C.free(unsafe.Pointer(msgCstr))
	}
	return YParserRetCode(C.yp_translate_validation_code(C.int(LYErrcode), apptagCstr, msgCstr))
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

	var errInfo C.struct_yp_error_info
	switch C.yp_get_last_error((*C.struct_ly_ctx)(ypCtx), &errInfo) {
	case 0:
		return YParserError{ErrCode: ypErrCode}
	case 1:
		return YParserError{ErrCode: YP_SUCCESS}
	}

	errMsg = C.GoString(errInfo.msg)
	errPath = C.GoString(errInfo.path)
	errAppTag = C.GoString(errInfo.apptag)

	// Try to resolve table, keys and field name from the error path.
	errtableName, key, ElemName = parseLyPath(errPath)

	if !strings.HasPrefix(errMsg, customErrorPrefix) {
		// libyang generated error message.. try to extract the field value & name
		ElemVal = parseLyMessage(errMsg, lyBadValue, lyUnsatisfied)
		if len(ElemName) == 0 { // if not resolved from path
			ElemName = parseLyMessage(errMsg, lyMandatory, lyElemPrefix, lyElemSuffix)
		}
	} else {
		/* Custom contraint error message like in must statement.
		This can be used by App to display to user.
		*/
		errText = errMsg[len(customErrorPrefix):]
	}

	switch errInfo.err {
	case C.LY_EVALID:
		// validation failure
		ypErrCode = translateLYErrToYParserErr(int(errInfo.vecode), errAppTag, errMsg)
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
func getModelChildInfo(l *YParserListInfo, node *C.yp_snode_t, module *YParserModule,
	underWhen bool, whenExpr *WhenExpression) {

	for sChild := C.yp_node_child(node); sChild != nil; sChild = sChild.next {
		switch sChild.nodetype {
		case C.LYS_LIST:
			keysCnt := C.golysc_node_list_keys_count(sChild)
			if keysCnt == 1 {
				// fetch key leaf
				for sChildInner := C.yp_node_child(sChild); sChildInner != nil; sChildInner = sChildInner.next {
					if sChildInner.nodetype == C.LYS_LEAF && C.yp_node_is_key(sChildInner) != 0 {
						keyName := C.GoString(sChildInner.name)
						l.MapLeaf = append(l.MapLeaf, keyName)
						break
					}
				}
				// Now, find and add the first non-key leaf.
				for sChildInner := C.yp_node_child(sChild); sChildInner != nil; sChildInner = sChildInner.next {
					if sChildInner.nodetype == C.LYS_LEAF && C.yp_node_is_key(sChildInner) == 0 {
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
			if sChild.nodetype == C.LYS_LEAF {
				if dflt := C.yp_leaf_dflt((*C.struct_ly_ctx)(ypCtx), sChild); dflt != nil {
					l.DfltLeafVal[leafName] = C.GoString(dflt)
				}
			} else {
				if dfltCnt := C.yp_leaflist_dflts_count(sChild); dfltCnt > 0 {
					tmpValStr := ""
					for idx := C.size_t(0); idx < dfltCnt; idx++ {
						if idx > 0 {
							//Separate multiple values by ,
							tmpValStr = tmpValStr + ","
						}
						tmpValStr = tmpValStr + C.GoString(C.yp_leaflist_dflt_at((*C.struct_ly_ctx)(ypCtx), sChild, idx))
					}
					l.DfltLeafVal[leafName] = tmpValStr
				}

				// leaf-list with min-elements > 0 should be treated as a mandatory node.
				// Reusing MandatoryNodes map itself to store this info.. Different error codes
				// are needed for min-elements and mandatory true violations. Cvl will have to
				// rely on the "@" field name suffix in db dataMap to differentiate.
				if C.yp_leaflist_min(sChild) > 0 {
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
			if musts := C.yp_node_musts(sChild); musts != nil {
				for idx := C.size_t(0); idx < C.yp_node_musts_count(sChild); idx++ {
					mustexpr := rewriteXPathPrefix(module, C.GoString(C.yp_node_must_cond_at(musts, idx)))
					exp := XpathExpression{Expr: mustexpr}
					if apptag := C.yp_node_must_apptag_at(musts, idx); apptag != nil {
						exp.ErrCode = C.GoString(apptag)
					}
					if emsg := C.yp_node_must_emsg_at(musts, idx); emsg != nil {
						exp.ErrStr = strings.TrimPrefix(C.GoString(emsg), customErrorPrefix)
					}

					l.XpathExpr[leafName] = append(l.XpathExpr[leafName],
						&exp)
				}
			}

			//Check for when expression
			if whenCond := C.yp_leaf_when_cond(sChild); whenCond != nil {
				whenexpr := rewriteXPathPrefix(module, C.GoString(whenCond))
				l.WhenExpr[leafName] = append(l.WhenExpr[leafName],
					&WhenExpression{
						Expr:      whenexpr,
						NodeNames: []string{leafName},
					})
			}

			//Check for custom extension
			for idx := C.size_t(0); idx < C.yp_node_exts_count(sChild); idx++ {
				if C.GoString(C.yp_node_ext_def_name(sChild, idx)) == "custom-validation" {
					if argVal := C.GoString(C.yp_node_ext_argument(sChild, idx)); argVal != "" {
						l.CustValidation[leafName] = append(l.CustValidation[leafName], argVal)
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
	// Each model has a base container at the top, with the per-table
	// containers under it. Skip the base container.
	topContainer := C.yp_module_top_container(mod)
	if topContainer == nil {
		return nil
	}

	for snode := C.yp_node_child(topContainer); snode != nil; snode = snode.next { //for each container
		if snode.nodetype != C.LYS_CONTAINER {
			continue
		}

		//for each list
		for n := C.yp_node_child(snode); n != nil; n = n.next {
			var l YParserListInfo
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
			if listMax := C.yp_list_max(n); listMax > 0 {
				l.RedisTableSize = int(listMax)
			}

			l.LeafRef = make(map[string][]string)
			l.XpathExpr = make(map[string][]*XpathExpression)
			l.CustValidation = make(map[string][]string)
			l.WhenExpr = make(map[string][]*WhenExpression)
			l.DfltLeafVal = make(map[string]string)
			l.MandatoryNodes = make(map[string]bool)

			//Add keys
			for child := C.yp_node_child(n); child != nil; child = child.next {
				if C.yp_node_is_key(child) != 0 {
					l.Keys = append(l.Keys, C.GoString(child.name))
				}
			}

			//Check for must expression
			if musts := C.yp_node_musts(n); musts != nil {
				for idx := C.size_t(0); idx < C.yp_node_musts_count(n); idx++ {
					mustexp := rewriteXPathPrefix(module, C.GoString(C.yp_node_must_cond_at(musts, idx)))
					exp := XpathExpression{Expr: mustexp}
					if apptag := C.yp_node_must_apptag_at(musts, idx); apptag != nil {
						exp.ErrCode = C.GoString(apptag)
					}
					if emsg := C.yp_node_must_emsg_at(musts, idx); emsg != nil {
						exp.ErrStr = strings.TrimPrefix(C.GoString(emsg), customErrorPrefix)
					}

					l.XpathExpr[listName] = append(l.XpathExpr[listName],
						&exp)
				}
			}

			//Check for custom extension
			for idx := C.size_t(0); idx < C.yp_node_exts_count(n); idx++ {
				extName := C.GoString(C.yp_node_ext_def_name(n, idx))
				argVal := C.GoString(C.yp_node_ext_argument(n, idx))
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

			//Add default key pattern
			if l.RedisKeyPattern == "" {
				keyPattern := []string{tableName}
				for idx := 0; idx < len(l.Keys); idx++ {
					keyPattern = append(keyPattern, fmt.Sprintf("{%s}", l.Keys[idx]))
				}
				l.RedisKeyPattern = strings.Join(keyPattern, l.RedisKeyDelim)
			}

			getModelChildInfo(&l, n, module, false, nil)

			list = append(list, &l)
		} //each list inside a container
	} //each container
	return list
}
