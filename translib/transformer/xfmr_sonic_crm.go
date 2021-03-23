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

package transformer

import (
    "fmt"
    log "github.com/golang/glog"
    "github.com/Azure/sonic-mgmt-common/translib/db"
    "encoding/json"
)

func init() {
    XlateFuncBind("rpc_crm_stats", rpc_crm_stats)
    XlateFuncBind("rpc_crm_acl_group_stats", rpc_crm_acl_group_stats)
    XlateFuncBind("rpc_crm_acl_table_stats", rpc_crm_acl_table_stats)
}

/* RPC CRM_STATS */
var rpc_crm_stats RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

    var str string
    var idx int

    log.Info("+++ RPC: rpc_crm_stats +++")

    d := dbs[db.CountersDB]
    d.Opts.KeySeparator = ":"
    d.Opts.TableNameSeparator = ":"

    tbl := db.TableSpec { Name: "CRM" }
    key := db.Key { Comp : [] string { "STATS" } }

    val, err := d.GetEntry(&tbl, key)

    if err == nil {
        idx = 0
        str += "{\n"
        str += "  \"sonic-system-crm:output\": {\n"
        for k, v := range val.Field {
            str += fmt.Sprintf("    \"%s\": %s", k, v)
            if (idx == len(val.Field) - 1) {
                str += "\n"
            } else {
                str += ",\n"
            }
            idx = idx + 1
        }
        str += "  }\n"
        str += "}"
    } else {
        str = "{}"
    }

    return []byte(str), err
}

/* RPC CRM_ACL_GROUP_STATS */
var rpc_crm_acl_group_stats RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

    var str string
    var idx int
    var err error
    var val db.Value
    var mapData map[string]interface{}

    log.Info("+++ RPC: rpc_crm_acl_group_stats +++")

    err = json.Unmarshal(body, &mapData)
    if err != nil {
        log.Error("Failed to unmarshall given input data")
        return nil, err
    }
    input := mapData["sonic-system-crm:input"]
    mapData = input.(map[string]interface{})

    rule := ""
    if value, ok := mapData["rule"].(string) ; ok {
        rule = value
    }

    proto := ""
    if value, ok := mapData["type"].(string) ; ok {
        proto = value
    }

    d := dbs[db.CountersDB]
    d.Opts.KeySeparator = ":"
    d.Opts.TableNameSeparator = ":"

    tbl := db.TableSpec { Name: "CRM" }
    key := db.Key { Comp : [] string { "ACL_STATS", rule, proto } }

    val, err = d.GetEntry(&tbl, key)

    if err == nil {
        idx = 0
        str += "{\n"
        str += "  \"sonic-system-crm:output\": {\n"
        for k, v := range val.Field {
            str += fmt.Sprintf("    \"%s\": %s", k, v)
            if (idx == len(val.Field) - 1) {
                str += "\n"
            } else {
                str += ",\n"
            }
            idx = idx + 1
        }
        str += "  }\n"
        str += "}"
    } else {
        str = "{}"
    }

    return []byte(str), err
}

/* RPC CRM_ACL_TABLE_STATS */
var rpc_crm_acl_table_stats RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
    var str string

    log.Info("+++ RPC: rpc_crm_acl_table_stats +++")

    tbl := db.TableSpec { Name: "CRM" }
    key := db.Key { Comp : [] string { "ACL_TABLE_STATS", "*" } }

    d := dbs[db.CountersDB]
    d.Opts.KeySeparator = ":"
    d.Opts.TableNameSeparator = ":"

    keys, err := d.GetKeysPattern(&tbl, key)
    if err == nil {
        str += "{\n"
        str += "  \"sonic-system-crm:output\": {\n"
        str += "    \"crm-acl-table-stats-list\": [\n"

        for i := 0; i < len(keys); i++ {
            var j int
            var val db.Value

            str += "      {\n"
            str += fmt.Sprintf("        \"id\": \"%s\",\n", keys[i].Comp[1])

            val, err = d.GetEntry(&tbl, keys[i])
            if err != nil {
                continue
            }
            j = 0
            for k, v := range val.Field {
                str += fmt.Sprintf("        \"%s\": %s", k, v)
                if (j == len(val.Field) - 1) {
                    str += "\n"
                } else {
                    str += ",\n"
                }
                j += 1
            }

            if (i == len(keys) - 1) {
                str += "      }\n"
            } else {
                str += "      },\n"
            }
        }

        str += "    ]\n"
        str += "  }\n"
        str += "}"
    } else {
        str = "{}"
    }

    return []byte(str), err
}
