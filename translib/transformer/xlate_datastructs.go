////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Dell, Inc.                                                 //
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
	"context"
	"regexp"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

var rgpIpv6, rgpMac *regexp.Regexp

type yangElementType uint8

type tblKeyCache struct {
	dbKey                string
	dbTblList            []string
	ygXpathInfo          *yangXpathInfo
	pathIdx              int // path index of the key transformer defined in the path
	childListNodePathIdx int // child list node path index
}

type KeySpec struct {
	DbNum           db.DBNum
	Ts              db.TableSpec
	Key             db.Key
	Child           []KeySpec
	IgnoreParentKey bool
	IsPartialKey    bool
}

type NotificationType int

const (
	TargetDefined NotificationType = iota
	Sample
	OnChange
)

// KEY_COMP_CNT - To specify the number of key components for the given key in the RedisDbSubscribeMap map
const KEY_COMP_CNT = "@KEY_COMP_CNT"

const DEL_AS_UPDATE = "@DEL_AS_UPDATE"

const FIELD_CURSOR = "@FIELD_CURSOR"

type XfmrTranslateSubscribeInfo struct {
	DbDataMap   RedisDbMap
	MinInterval int
	NeedCache   bool
	PType       NotificationType
	OnChange    bool
}

type xpathTblKeyExtractRet struct {
	xpath         string
	tableName     string
	dbKey         string
	isVirtualTbl  bool
	isNotTblOwner bool
}

type xlateFromDbParams struct {
	d          *db.DB //current db
	dbs        [db.MaxDB]*db.DB
	curDb      db.DBNum
	ygRoot     *ygot.GoStruct
	uri        string
	requestUri string //original uri using which a curl/NBI request is made
	oper       Operation
	dbDataMap  *map[db.DBNum]map[string]map[string]db.Value
	// subOpDataMap map[int]*RedisDbMap // used to add an in-flight data with a sub-op
	// param interface{}
	txCache interface{}
	//  skipOrdTblChk *bool
	//  pCascadeDelTbl *[] string //used to populate list of tables needed cascade delete by subtree overloaded methods
	xpath             string //curr uri xpath
	tbl               string
	tblKey            string
	resultMap         map[string]interface{}
	xfmrDbTblKeyCache map[string]tblKeyCache
	queryParams       QueryParams
	dbTblKeyGetCache  map[db.DBNum]map[string]map[string]bool
	reqCtxt           context.Context
	listKeysMap       map[string]interface{}
	ygParentObj       *ygot.GoStruct
	relUri            string
	ygSchema          *yang.Entry
}

type xlateToParams struct {
	d                       *db.DB
	ygRoot                  *ygot.GoStruct
	oper                    Operation
	uri                     string
	requestUri              string
	xpath                   string
	parentXpath             string
	keyName                 string
	jsonData                interface{}
	resultMap               map[Operation]RedisDbMap
	result                  map[string]map[string]db.Value
	txCache                 interface{}
	tblXpathMap             map[string]map[string]map[string]bool
	subOpDataMap            map[Operation]*RedisDbMap
	pCascadeDelTbl          *[]string
	xfmrErr                 *error
	name                    string
	value                   interface{}
	tableName               string
	isNotTblOwner           bool
	yangDefValMap           map[string]map[string]db.Value
	yangAuxValMap           map[string]map[string]db.Value
	xfmrDbTblKeyCache       map[string]tblKeyCache
	dbTblKeyGetCache        map[db.DBNum]map[string]map[string]bool
	invokeCRUSubtreeOnceMap map[string]map[string]bool
}

type contentQPSpecMapInfo struct {
	yangType              yangElementType
	yangName              string
	isReadOnly            bool
	isOperationalNd       bool
	hasNonTerminalNd      bool
	hasChildOperationalNd bool
	isOcMdl               bool
}

type qpSubtreePruningErr struct {
	subtreePath string
}

type Operation int
type subOpDataMapType map[Operation]*RedisDbMap

type ContentType uint8

type QueryParams struct {
	depthEnabled      bool
	curDepth          uint
	content           ContentType
	fields            []string
	fieldsFillAll     bool
	allowFieldsXpath  map[string]bool
	tgtFieldsXpathMap map[string][]string
}

type ygotUnMarshalCtx struct {
	ygParentObj  *ygot.GoStruct
	relUri       string
	ygSchema     *yang.Entry
	trgtYgObj    *ygot.GoStruct
	trgtYgSchema *yang.Entry
	err          error
}

type ygotXlator struct {
	ygotCtx *ygotUnMarshalCtx
}
