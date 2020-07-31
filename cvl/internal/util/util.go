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

package util

import (
	"os"
	"fmt"
	"runtime"
	 "encoding/json"
        "io/ioutil"
        "os/signal"
        "syscall"
	"strings"
	"flag"
	"github.com/go-redis/redis"
	log "github.com/golang/glog"
)

var CVL_SCHEMA string = "/usr/sbin/schema/"
var CVL_CFG_FILE string = "/usr/sbin/cvl_cfg.json"
const SONIC_DB_CONFIG_FILE string = "/var/run/redis/sonic-db/database_config.json"
const ENV_VAR_SONIC_DB_CONFIG_FILE = "DB_CONFIG_PATH"
var sonic_db_config = make(map[string]interface{})

//package init function 
func init() {
	if (os.Getenv("CVL_SCHEMA_PATH") != "") {
		CVL_SCHEMA = os.Getenv("CVL_SCHEMA_PATH") + "/"
	}

	if (os.Getenv("CVL_CFG_FILE") != "") {
		CVL_CFG_FILE = os.Getenv("CVL_CFG_FILE")
	}

	//Initialize DB settings
	dbCfgInit()
}

var cvlCfgMap map[string]string

/* Logging Level for CVL global logging. */
type CVLLogLevel uint8 
const (
        INFO  = 0 + iota
        WARNING
        ERROR
        FATAL
        INFO_API
	INFO_TRACE
	INFO_DEBUG
	INFO_DATA
	INFO_DETAIL
	INFO_ALL
)

var cvlTraceFlags uint32

/* Logging levels for CVL Tracing. */
type CVLTraceLevel uint32 
const (
	TRACE_MIN = 0
	TRACE_MAX = 8 
        TRACE_CACHE  = 1 << TRACE_MIN 
        TRACE_LIBYANG = 1 << 1
        TRACE_YPARSER = 1 << 2
        TRACE_CREATE = 1 << 3
        TRACE_UPDATE = 1 << 4
        TRACE_DELETE = 1 << 5
        TRACE_SEMANTIC = 1 << 6
        TRACE_ONERROR = 1 << 7 
        TRACE_SYNTAX = 1 << TRACE_MAX 

)


var traceLevelMap = map[int]string {
	/* Caching operation traces */
	TRACE_CACHE : "TRACE_CACHE",
	/* Libyang library traces. */
	TRACE_LIBYANG: "TRACE_LIBYANG",
	/* Yang Parser traces. */
	TRACE_YPARSER : "TRACE_YPARSER", 
	/* Create operation traces. */
	TRACE_CREATE : "TRACE_CREATE", 
	/* Update operation traces. */
	TRACE_UPDATE : "TRACE_UPDATE", 
	/* Delete operation traces. */
	TRACE_DELETE : "TRACE_DELETE", 
	/* Semantic Validation traces. */
	TRACE_SEMANTIC : "TRACE_SEMANTIC",
	/* Syntax Validation traces. */
	TRACE_SYNTAX : "TRACE_SYNTAX", 
	/* Trace on Error. */
	TRACE_ONERROR : "TRACE_ONERROR",
}

var Tracing bool = false

var traceFlags uint16 = 0

func SetTrace(on bool) {
	if (on == true) {
		Tracing = true
		traceFlags = 1
	} else {
		Tracing = false 
		traceFlags = 0
	}
}

func IsTraceSet() bool {
	if (traceFlags == 0) {
		return false
	} else {
		return true
	}
}

func TRACE_LEVEL_LOG(level log.Level, tracelevel CVLTraceLevel, fmtStr string, args ...interface{}) {

	if (IsTraceSet() == false) {
		return
	}

	level = (level - INFO_API) + 1;

	traceEnabled := false
		if ((cvlTraceFlags & (uint32)(tracelevel)) != 0) {
			traceEnabled = true
		}

	if IsTraceSet() == true && traceEnabled == true {
		pc := make([]uintptr, 10)
		runtime.Callers(2, pc)
		f := runtime.FuncForPC(pc[0])
		file, line := f.FileLine(pc[0])

		fmt.Printf("%s:%d %s(): ", file, line, f.Name())
		fmt.Printf(fmtStr+"\n", args...)
	} else {
		if (traceEnabled == true) {
			log.V(level).Infof(fmtStr, args...)
		}
	}
}

func CVL_LEVEL_LOG(level CVLLogLevel, format string, args ...interface{}) {

	switch level {
		case INFO:
		       log.Infof(format, args...)
		case  WARNING:
		       log.Warningf(format, args...)
		case  ERROR:
		       log.Errorf(format, args...)
		case  FATAL:
		       log.Fatalf(format, args...)
		case INFO_API:
			log.V(1).Infof(format, args...)
		case INFO_TRACE:
			log.V(2).Infof(format, args...)
		case INFO_DEBUG:
			log.V(3).Infof(format, args...)
		case INFO_DATA:
			log.V(4).Infof(format, args...)
		case INFO_DETAIL:
			log.V(5).Infof(format, args...)
		case INFO_ALL:
			log.V(6).Infof(format, args...)
	}	

}


func ConfigFileSyncHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	go func() {
		for {
			<-sigs
			cvlCfgMap := ReadConfFile()

			if cvlCfgMap == nil {
				return
			}

			CVL_LEVEL_LOG(INFO ,"Received SIGUSR2. Changed configuration values are %v", cvlCfgMap)


			if (strings.Compare(cvlCfgMap["LOGTOSTDERR"], "true") == 0) {
				SetTrace(true)
				flag.Set("logtostderr", "true")
				flag.Set("stderrthreshold", cvlCfgMap["STDERRTHRESHOLD"])
				flag.Set("v", cvlCfgMap["VERBOSITY"])
			}
		}
	}()

}

func ReadConfFile()  map[string]string{

	/* Return if CVL configuration file is not present. */
	if _, err := os.Stat(CVL_CFG_FILE); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(CVL_CFG_FILE)

	err = json.Unmarshal(data, &cvlCfgMap)

	if err != nil {
		CVL_LEVEL_LOG(INFO ,"Error in reading cvl configuration file %v", err)
		return nil
	}

	CVL_LEVEL_LOG(INFO ,"Current Values of CVL Configuration File %v", cvlCfgMap)
	var index uint32

	for  index = TRACE_MIN ; index < TRACE_MAX ; index++  {
		if (strings.Compare(cvlCfgMap[traceLevelMap[1 << index]], "true") == 0) {
			cvlTraceFlags = cvlTraceFlags |  (1 << index) 
		}
	}

	return cvlCfgMap
}

func SkipValidation() bool {
	val, existing := cvlCfgMap["SKIP_VALIDATION"]
	if (existing == true) && (val == "true") {
		return true
	}

	return false
}

func SkipSemanticValidation() bool {
	val, existing := cvlCfgMap["SKIP_SEMANTIC_VALIDATION"]
	if (existing == true) && (val == "true") {
		return true
	}

	return false
}

//Function to read Redis DB configuration from file.
//In absence of the file, it uses default config for CONFIG_DB
//so that CVL UT will pass in development environment.
func dbCfgInit() {
	defaultDBConfig := `{
		"INSTANCES": {
			"redis":{
				"hostname" : "127.0.0.1",
				"port" : 6379
			}
		},
		"DATABASES" : {
			"CONFIG_DB" : {
				"id" : 4,
				"separator": "|",
				"instance" : "redis"
			},
			"STATE_DB" : {
				"id" : 6,
				"separator": "|",
				"instance" : "redis"
			}
		}
	}`

	dbCfgFile := ""

	//Check if multi-db config file is present
	if _, errF := os.Stat(SONIC_DB_CONFIG_FILE); !os.IsNotExist(errF) {
		dbCfgFile = SONIC_DB_CONFIG_FILE
	} else {
		//Check if multi-db config file is specified in environment
		if fileName := os.Getenv(ENV_VAR_SONIC_DB_CONFIG_FILE); fileName != "" {
			if _, errF := os.Stat(fileName); !os.IsNotExist(errF) {
				dbCfgFile = fileName
			}
		}
	}

	if dbCfgFile != "" {
		//Read from multi-db config file
		data, err := ioutil.ReadFile(dbCfgFile)
		if err != nil {
			panic(err)
		} else {
			err = json.Unmarshal([]byte(data), &sonic_db_config)
			if err != nil {
				panic(err)
			}
		}
	} else {
		//No multi-db config file is present.
		//Use default config for CONFIG_DB setting, this avoids CVL UT failure
		//in absence of at multi-db config file
		err := json.Unmarshal([]byte(defaultDBConfig), &sonic_db_config)
		if err != nil {
			panic(err)
		}
	}
}

//Get list of DB
func getDbList()(map[string]interface{}) {
	db_list, ok := sonic_db_config["DATABASES"].(map[string]interface{})
	if !ok {
		panic(fmt.Errorf("DATABASES' is not valid key in %s!",
		SONIC_DB_CONFIG_FILE))
	}
	return db_list
}

//Get DB instance based on given DB name
func getDbInst(dbName string)(map[string]interface{}) {
	db, ok := sonic_db_config["DATABASES"].(map[string]interface{})[dbName]
	if !ok {
		panic(fmt.Errorf("database name '%v' is not valid in %s !",
		dbName, SONIC_DB_CONFIG_FILE))
	}
	inst_name, ok := db.(map[string]interface{})["instance"]
	if !ok {
		panic(fmt.Errorf("'instance' is not a valid field in %s !",
		SONIC_DB_CONFIG_FILE))
	}
	inst, ok := sonic_db_config["INSTANCES"].(map[string]interface{})[inst_name.(string)]
	if !ok {
		panic(fmt.Errorf("instance name '%v' is not valid in %s !",
		inst_name, SONIC_DB_CONFIG_FILE))
	}
	return inst.(map[string]interface{})
}

//GetDbSeparator Get DB separator based on given DB name
func GetDbSeparator(dbName string)(string) {
	db_list := getDbList()
	separator, ok := db_list[dbName].(map[string]interface{})["separator"]
	if !ok {
		panic(fmt.Errorf("'separator' is not a valid field in %s !",
		SONIC_DB_CONFIG_FILE))
	}
	return separator.(string)
}

//GetDbId Get DB id on given db name
func GetDbId(dbName string)(int) {
	db_list := getDbList()
	id, ok := db_list[dbName].(map[string]interface{})["id"]
	if !ok {
		panic(fmt.Errorf("'id' is not a valid field in %s !",
		SONIC_DB_CONFIG_FILE))
	}
	return int(id.(float64))
}

//GetDbSock Get DB socket path
func GetDbSock(dbName string)(string) {
	inst := getDbInst(dbName)
	unix_socket_path, ok := inst["unix_socket_path"]
	if !ok {
		CVL_LEVEL_LOG(INFO, "'unix_socket_path' is not " +
		"a valid field in %s !", SONIC_DB_CONFIG_FILE)

		return ""
	}

	return unix_socket_path.(string)
}

//GetDbTcpAddr Get DB TCP endpoint
func GetDbTcpAddr(dbName string)(string) {
	inst := getDbInst(dbName)
	hostname, ok := inst["hostname"]
	if !ok {
		panic(fmt.Errorf("'hostname' is not a valid field in %s !",
		SONIC_DB_CONFIG_FILE))
	}

	port, ok1 := inst["port"]
	if !ok1 {
		panic(fmt.Errorf("'port' is not a valid field in %s !",
		SONIC_DB_CONFIG_FILE))
	}

	return fmt.Sprintf("%v:%v", hostname, port)
}

//NewDbClient Get new redis client 
func NewDbClient(dbName string) *redis.Client {
	var redisClient *redis.Client = nil

	//Try unix domain socket first
	if dbSock := GetDbSock(dbName); dbSock != "" {
		redisClient = redis.NewClient(&redis.Options{
			Network:  "unix",
			Addr:     dbSock,
			Password: "",
			DB:       GetDbId(dbName),
		})
	} else {
	//Otherwise, use TCP socket
		redisClient = redis.NewClient(&redis.Options{
			Network:  "tcp",
			Addr:     GetDbTcpAddr(dbName),
			Password: "",
			DB:       GetDbId(dbName),
		})
	}

	if (redisClient == nil) {
		return nil
	}

	//Check the connectivity
	_, err := redisClient.Ping().Result()
	if err != nil {
		CVL_LEVEL_LOG(ERROR, "Failed to connect to Redis server %v", err)
		return nil
	}

	return redisClient
}
