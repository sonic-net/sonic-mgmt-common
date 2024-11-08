////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package common

import (
	"github.com/antchfx/xmlquery"
)

// CVLEditConfigData Strcture for key and data in API
type CVLEditConfigData struct {
	VType     CVLValidateType   //Validation type
	VOp       CVLOperation      //Operation type
	Key       string            //Key format : "PORT|Ethernet4"
	Data      map[string]string //Value :  {"alias": "40GE0/28", "mtu" : 9100,  "admin_status":  down}
	ReplaceOp bool
}

type CVLValidateType uint

const (
	VALIDATE_NONE      CVLValidateType = iota //Data is used as dependent data
	VALIDATE_SYNTAX                           //Syntax is checked and data is used as dependent data
	VALIDATE_SEMANTICS                        //Semantics is checked
	VALIDATE_ALL                              //Syntax and Semantics are checked
)

type CVLOperation uint

const (
	OP_NONE   CVLOperation = 0      //Used to just validate the config without any operation
	OP_CREATE              = 1 << 0 //For Create operation
	OP_UPDATE              = 1 << 1 //For Update operation
	OP_DELETE              = 1 << 2 //For Delete operation
)

// RequestCacheType Struct for request data and YANG data
type RequestCacheType struct {
	ReqData  CVLEditConfigData
	YangData *xmlquery.Node
}
