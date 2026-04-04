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

package db

import (
	"encoding/json"
	"fmt"
	io "io/ioutil"
	"os"
	"strconv"

	"github.com/golang/glog"
)

var dbConfigMap = make(map[string]interface{})

func dbConfigInit() {
	dbConfigPath := "/var/run/redis/sonic-db/database_config.json"
	if path, ok := os.LookupEnv("DB_CONFIG_PATH"); ok {
		dbConfigPath = path
	}

	// If the path does not exist, it could be a go lang jenkins test with
	// an uninitialized/missing DB_CONFIG_PATH. Use the path
	// ${PWD}/../../../tools/test/database_config.json if it exists
	if _, e := os.Stat(dbConfigPath); e != nil {
		cwd, e := os.Getwd()
		goTestDBConfigPath := cwd + "/../../../tools/test/database_config.json"
		if _, e = os.Stat(goTestDBConfigPath); e == nil {
			dbConfigPath = goTestDBConfigPath
		}
	}

	data, err := io.ReadFile(dbConfigPath)
	if err != nil {
		assert(err)
	} else {
		err = json.Unmarshal([]byte(data), &dbConfigMap)
		if err != nil {
			assert(err)
		}
	}
}

func assert(msg error) {
	panic(msg)
}

func getDbList() map[string]interface{} {
	dbEntries, ok := dbConfigMap["DATABASES"].(map[string]interface{})
	if !ok {
		assert(fmt.Errorf("DATABASES is invalid key."))
	}
	return dbEntries
}

func isDbInstPresent(dbName string) bool {
	_, ok := getDbList()[dbName]
	return ok
}

func getDbInst(dbName string) map[string]interface{} {
	db, ok := dbConfigMap["DATABASES"].(map[string]interface{})[dbName]
	if !ok {
		assert(fmt.Errorf("database name '%v' is not found", dbName))
	}
	instName, ok := db.(map[string]interface{})["instance"]
	if !ok {
		assert(fmt.Errorf("'instance' is not a valid field"))
	}
	inst, ok := dbConfigMap["INSTANCES"].(map[string]interface{})[instName.(string)]
	if !ok {
		assert(fmt.Errorf("instance name '%v' is not found", instName))
	}
	return inst.(map[string]interface{})
}

func getDbSeparator(dbName string) string {
	dbEntries := getDbList()
	separator, ok := dbEntries[dbName].(map[string]interface{})["separator"]
	if !ok {
		assert(fmt.Errorf("'separator' is not a valid field"))
	}
	return separator.(string)
}

func getDbId(dbName string) int {
	dbEntries := getDbList()
	id, ok := dbEntries[dbName].(map[string]interface{})["id"]
	if !ok {
		assert(fmt.Errorf("'id' is not a valid field"))
	}
	return int(id.(float64))
}

func getDbHostName(dbName string) string {
	inst := getDbInst(dbName)
	hostname, ok := inst["hostname"]
	if !ok {
		assert(fmt.Errorf("'hostname' is not a valid field"))
	}
	return hostname.(string)
}

func getDbPort(dbName string) int {
	inst := getDbInst(dbName)
	port, ok := inst["port"]
	if !ok {
		assert(fmt.Errorf("'port' is not a valid field"))
	}
	return int(port.(float64))
}

func getDbTcpAddr(dbName string) string {
	hostname := getDbHostName(dbName)
	port := getDbPort(dbName)
	return hostname + ":" + strconv.Itoa(port)
}

func getDbSock(dbName string) string {
	inst := getDbInst(dbName)
	if unix_socket_path, ok := inst["unix_socket_path"]; ok {
		return unix_socket_path.(string)
	} else {
		glog.V(4).Info("getDbSock: 'unix_socket_path' is not a valid field")
		return ""
	}
}

func getDbPassword(dbName string) string {
	inst := getDbInst(dbName)
	password := ""
	password_path, ok := inst["password_path"]
	if !ok {
		return password
	}
	data, er := io.ReadFile(password_path.(string))
	if er != nil {
		//
	} else {
		password = (string(data))
	}
	return password
}

func GetDbConfigMap() map[string]interface{} {
	return dbConfigMap
}
