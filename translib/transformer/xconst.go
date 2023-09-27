////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Dell, Inc.                                                 //
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

const (
	XPATH_SEP_FWD_SLASH    = "/"
	XFMR_EMPTY_STRING      = ""
	XFMR_NONE_STRING       = "NONE"
	SONIC_TABLE_INDEX      = 2
	SONIC_TBL_CHILD_INDEX  = 3
	SONIC_FIELD_INDEX      = 4
	SONIC_TOPCONTR_INDEX   = 1
	SONIC_MDL_PFX          = "sonic"
	OC_MDL_PFX             = "openconfig-"
	IETF_MDL_PFX           = "ietf-"
	IANA_MDL_PFX           = "iana-"
	PATH_XFMR_RET_ARGS     = 1
	PATH_XFMR_RET_ERR_INDX = 0

	YANG_CONTAINER_NM_CONFIG  = "config"
	CONFIG_CNT_SUFFIXED_XPATH = "/config"
	STATE_CNT_SUFFIXED_XPATH  = "/state"
	CONFIG_CNT_WITHIN_XPATH   = "/config/"
	STATE_CNT_WITHIN_XPATH    = "/state/"
)

const (
	YANG_MODULE yangElementType = iota + 1
	YANG_LIST
	YANG_CONTAINER
	YANG_LEAF
	YANG_LEAF_LIST
	YANG_CHOICE
	YANG_CASE
	YANG_RPC
	YANG_NOTIF
)
const (
	XFMR_INVALID = iota - 1
	XFMR_DISABLE
	XFMR_ENABLE
	XFMR_DEFAULT_ENABLE
)

const (
	QUERY_CONTENT_ALL ContentType = iota
	QUERY_CONTENT_CONFIG
	QUERY_CONTENT_NONCONFIG
	QUERY_CONTENT_OPERATIONAL
)

const (
	QUERY_CONTENT_MISMATCH_ERR      = "Query Parameter Content mismatch"
	QUERY_PARAMETER_SBT_PRUNING_ERR = "Query Parameter processing unsuccessful"
)

const (
	GET Operation = iota + 1
	CREATE
	REPLACE
	UPDATE
	DELETE
	SUBSCRIBE
	MAXOPER
)
