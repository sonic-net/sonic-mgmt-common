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

import (
    "github.com/Azure/sonic-mgmt-common/translib/db"
)

type XfmrDbTblCbkParams struct {
    d                       *db.DB        //Config DB handler
    oper                    int
    delDepRefKey            string
    tblName                 string
    dbKey                   string
    delDepEntry             map[string]string 
    dbDataMap               map[db.DBNum]map[string]map[string]db.Value
    delDepDataMap           map[int]*RedisDbMap   // Call back methods can add the data  
}

func formXfmrDbTblCbkParams (d *db.DB, oper int, delDepRefKey string, tblName string, dbKey string, delDepEntry map[string]string, dbDataMap RedisDbMap) XfmrDbTblCbkParams {

    var inParams XfmrDbTblCbkParams

    inParams.d = d
    inParams.oper = oper
    inParams.delDepRefKey = delDepRefKey
    inParams.tblName = tblName
    inParams.dbKey = dbKey
    inParams.delDepEntry = delDepEntry
    inParams.dbDataMap = dbDataMap
    inParams.delDepDataMap =  make(map[int]*RedisDbMap)

    return inParams
}

type  XfmrDbTblCbkMethod func (inParams XfmrDbTblCbkParams) error 

