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

package cvl_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/cvl"
	. "github.com/Azure/sonic-mgmt-common/cvl/internal/util"
	"github.com/go-redis/redis/v7"
)

// type aliases
type CVLEditConfigData = cvl.CVLEditConfigData
type CVLErrorInfo = cvl.CVLErrorInfo
type CVLRetCode = cvl.CVLRetCode

// enum aliases
const (
	VALIDATE_NONE = cvl.VALIDATE_NONE
	VALIDATE_ALL  = cvl.VALIDATE_ALL
	OP_CREATE     = cvl.OP_CREATE
	OP_UPDATE     = cvl.OP_UPDATE
	OP_DELETE     = cvl.OP_DELETE
)

type testEditCfgData struct {
	filedescription string
	cfgData         string
	depData         string
	retCode         cvl.CVLRetCode
}

var rclient *redis.Client
var port_map map[string]interface{}

var loadDeviceDataMap bool
var deviceDataMap = map[string]interface{} {
	"DEVICE_METADATA" : map[string]interface{} {
		"localhost": map[string] interface{} {
			"hwsku": "Quanta-IX8-54x",
			"hostname": "sonic",
			"platform": "x86_64-quanta_ix8_54x-r0",
			"mac": "4c:76:25:f4:70:82",
			"deployment_id": "1",
		},
	},
}

/* Dependent port channel configuration. */
var depDataMap = map[string]interface{} {
	"PORTCHANNEL" : map[string]interface{} {
		"PortChannel001": map[string] interface{} {
			"admin_status": "up",
			"mtu": "9100",
		},
		"PortChannel002": map[string] interface{} {
			"admin_status": "up",
			"mtu": "9100",
		},
		"PortChannel003": map[string]interface{}{
			"admin_status": "up",
			"mtu":          "9100",
		},
	},
	"PORTCHANNEL_MEMBER": map[string]interface{} {
		"PortChannel001|Ethernet4": map[string] interface{} {
			"NULL": "NULL",
		},
		"PortChannel001|Ethernet8": map[string] interface{} {
			"NULL": "NULL",
		},
		"PortChannel001|Ethernet12": map[string] interface{} {
			"NULL": "NULL",
		},
		"PortChannel002|Ethernet20": map[string] interface{} {
			"NULL": "NULL",
		},
		"PortChannel002|Ethernet24": map[string] interface{} {
			"NULL": "NULL",
		},
	},
}

/* Converts JSON Data in a File to Map. */
func convertJsonFileToMap(t *testing.T, fileName string) map[string]string {
	var mapstr map[string]string

	jsonData := convertJsonFileToString(t, fileName)
	byteData := []byte(jsonData)

	err := json.Unmarshal(byteData, &mapstr)

	if err != nil {
		fmt.Println("Failed to convert Json File to map:", err)
	}

	return mapstr

}

/* Converts JSON Data in a File to Map. */
func convertDataStringToMap(t *testing.T, dataString string) map[string]string {
	var mapstr map[string]string

	byteData := []byte(dataString)

	err := json.Unmarshal(byteData, &mapstr)

	if err != nil {
		fmt.Println("Failed to convert Json Data String to map:", err)
	}

	return mapstr

}

/* Converts JSON Data in a File to String. */
func convertJsonFileToString(t *testing.T, fileName string) string {
	var jsonData string

	data, err := ioutil.ReadFile(fileName)

	if err != nil {
		fmt.Printf("\nFailed to read data file : %v\n", err)
	} else {
		jsonData = string(data)
	}

	return jsonData
}

/* Converts JSON config to map which can be loaded to Redis */
func loadConfig(key string, in []byte) map[string]interface{} {
	var fvp map[string]interface{}

	err := json.Unmarshal(in, &fvp)
	if err != nil {
		fmt.Printf("Failed to Unmarshal %v err: %v", in, err)
	}
	if key != "" {
		kv := map[string]interface{}{}
		kv[key] = fvp
		return kv
	}
	return fvp
}

/* Separator for keys. */
func getSeparator() string {
	return "|"
}

/* Unloads the Config DB based on JSON File. */
func unloadConfigDB(rclient *redis.Client, mpi map[string]interface{}) {
	for key, fv := range mpi {
		switch fv.(type) {
		case map[string]interface{}:
			for subKey, subValue := range fv.(map[string]interface{}) {
				newKey := key + getSeparator() + subKey
				_, err := rclient.Del(newKey).Result()

				if err != nil {
					fmt.Printf("Invalid data for db: %v : %v %v", newKey, subValue, err)
				}

			}
		default:
			fmt.Printf("Invalid data for db: %v : %v", key, fv)
		}
	}

}

/* Loads the Config DB based on JSON File. */
func loadConfigDB(rclient *redis.Client, mpi map[string]interface{}) {
	for key, fv := range mpi {
		switch fv.(type) {
		case map[string]interface{}:
			for subKey, subValue := range fv.(map[string]interface{}) {
				newKey := key + getSeparator() + subKey
				_, err := rclient.HMSet(newKey, subValue.(map[string]interface{})).Result()

				if err != nil {
					fmt.Printf("Invalid data for db: %v : %v %v", newKey, subValue, err)
				}

			}
		default:
			fmt.Printf("Invalid data for db: %v : %v", key, fv)
		}
	}
}

func getConfigDbClient() *redis.Client {
	rclient := NewDbClient("CONFIG_DB")

	if rclient == nil {
		panic("Unable to connect to Redis Config DB Server")
	}

	return rclient
}

/* Prepares the database in Redis Server. */
func prepareDb() {
	rclient = getConfigDbClient()

	fileName := "testdata/port_table.json"
	PortsMapByte, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("read file %v err: %v", fileName, err)
	}

	//Load device data map on which application of deviation files depends
	dm, err:= rclient.Keys("DEVICE_METADATA|localhost").Result()
	if (err != nil) || (len(dm) == 0) {
		loadConfigDB(rclient, deviceDataMap)
		loadDeviceDataMap = true
	}

	port_map = loadConfig("", PortsMapByte)

	portKeys, err:= rclient.Keys("PORT|*").Result()
	//Load only the port config which are not there in Redis
	if err == nil {
		portMapKeys := port_map["PORT"].(map[string]interface{})
		for _, portKey := range portKeys {
			//Delete the port key which is already there in Redis
			delete(portMapKeys, portKey[len("PORTS|") - 1:])
		}
		port_map["PORT"] = portMapKeys
	}

	loadConfigDB(rclient, port_map)
	loadConfigDB(rclient, depDataMap)
}

//Clear all db entries which are used in the test cases.
//The list of such db should be updated here if new
//table is referred in any test case.
//The test case running may fail if tables are not cleared 
//prior to starting execution of test cases.
//"DEVICE_METADATA" should not be cleaned as it is used
//during cvl package init() phase.
func clearDb() {

	tblList := []string {
		"ACL_RULE",
		"ACL_TABLE",
		"BGP_GLOBALS",
		"BUFFER_PG",
		"CABLE_LENGTH",
		"CFG_L2MC_TABLE",
		"INTERFACE",
		"MIRROR_SESSION",
		"PORTCHANNEL",
		"PORTCHANNEL_MEMBER",
		"PORT_QOS_MAP",
		"QUEUE",
		"SCHEDULER",
		"STP",
		"STP_PORT",
		"STP_VLAN",
		"TAM_COLLECTOR_TABLE",
		"TAM_INT_IFA_FLOW_TABLE",
		"VLAN",
		"VLAN_INTERFACE",
		"VLAN_MEMBER",
		"VRF",
		"VXLAN_TUNNEL",
		"VXLAN_TUNNEL_MAP",
		"WRED_PROFILE",
		"DSCP_TO_TC_MAP",
	}

	for _, tbl := range tblList {
		_, err := exec.Command("/bin/sh", "-c",
		"sonic-db-cli CONFIG_DB del `sonic-db-cli CONFIG_DB keys '" +
		tbl + "|*' | cut -d ' ' -f 2`").Output()

		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

/* Setup before starting of test. */
func TestMain(m *testing.M) {

	redisAlreadyRunning := false
	pidOfRedis, err := exec.Command("pidof", "redis-server").Output()
	if err == nil &&  string(pidOfRedis) != "\n" {
		redisAlreadyRunning = true
	}

	if (redisAlreadyRunning == false) {
		//Redis not running, lets start it
		_, err := exec.Command("/bin/sh", "-c", "sudo /etc/init.d/redis-server start").Output()
		if err != nil {
			fmt.Println(err.Error())
		}

	}

	//Clear all tables which are used for testing
	clearDb()

	/* Prepare the Redis database. */
	prepareDb()
	SetTrace(true)
	cvl.Debug(true)
	code := m.Run()
	//os.Exit(m.Run())

	unloadConfigDB(rclient, port_map)
	unloadConfigDB(rclient, depDataMap)
	if (loadDeviceDataMap == true) {
		unloadConfigDB(rclient, deviceDataMap)
	}

	//Clear all tables which were used for testing
	clearDb()

	cvl.Finish()
	rclient.Close()
	rclient.FlushDB()

	if (redisAlreadyRunning == false) {
		//If Redis was not already running, close the instance that we ran
		_, err := exec.Command("/bin/sh", "-c", "sudo /etc/init.d/redis-server stop").Output()
		if err != nil {
			fmt.Println(err.Error())
		}

	}

	os.Exit(code)
}

//Test Initialize() API
func TestInitialize(t *testing.T) {
	ret := cvl.Initialize()
	if (ret != cvl.CVL_SUCCESS) {
		t.Errorf("CVl initialization failed")
	}

	ret = cvl.Initialize()
	if (ret != cvl.CVL_SUCCESS) {
		t.Errorf("CVl re-initialization should not fail")
	}
}

//Test Initialize() API
func TestFinish(t *testing.T) {
	ret := cvl.Initialize()
	if (ret != cvl.CVL_SUCCESS) {
		t.Errorf("CVl initialization failed")
	}

	cvl.Finish()

	//Initialize again for other test cases to run
	cvl.Initialize()
}

func NewTestSession(t *testing.T) *cvl.CVL {
	t.Helper()
    c, status := cvl.ValidationSessOpen()
	if status != CVL_SUCCESS {
		t.Fatalf("ValidationSessOpen failed; err=%v", status)
	}
    t.Cleanup(func() { cvl.ValidationSessClose(c) })
    return c
}

/* ValidateEditConfig with user input in file . */
func TestValidateEditConfig_CfgFile(t *testing.T) {

	tests := []struct {
		filedescription string
		cfgDataFile     string
		depDataFile     string
		retCode         cvl.CVLRetCode
	}{
		{filedescription: "ACL_DATA", cfgDataFile: "testdata/aclrule.json", depDataFile: "testdata/acltable.json", retCode: cvl.CVL_SUCCESS},
	}

	cvSess, _ := cvl.ValidationSessOpen()

	for index, tc := range tests {
		t.Logf("Running Testcase %d with Description %s", index+1, tc.filedescription)

		t.Run(tc.filedescription, func(t *testing.T) {

			jsonEditCfg_Create_DependentMap := convertJsonFileToMap(t, tc.depDataFile)
			jsonEditCfg_Create_ConfigMap := convertJsonFileToMap(t, tc.cfgDataFile)

			cfgData := []cvl.CVLEditConfigData{
				cvl.CVLEditConfigData{cvl.VALIDATE_ALL, cvl.OP_CREATE, "ACL_TABLE|TestACL1", jsonEditCfg_Create_DependentMap},
			}


			cvlErrObj, err := cvSess.ValidateEditConfig(cfgData)

			if err != tc.retCode {
				t.Errorf("Config Validation failed. %v", cvlErrObj)
			}

			cfgData = []cvl.CVLEditConfigData{
				cvl.CVLEditConfigData{cvl.VALIDATE_ALL, cvl.OP_CREATE, "ACL_RULE|TestACL1|Rule1", jsonEditCfg_Create_ConfigMap},
			}


			cvlErrObj, err = cvSess.ValidateEditConfig(cfgData)

			if err != tc.retCode {
				t.Errorf("Config Validation failed. %v", cvlErrObj)
			}
		})
	}

	cvl.ValidationSessClose(cvSess)
}

/* ValidateEditConfig with user input inline. */
func TestValidateEditConfig_CfgStrBuffer(t *testing.T) {

	type testStruct struct {
		filedescription string
		cfgData         string
		depData         string
		retCode         cvl.CVLRetCode
	}

	cvSess, _ := cvl.ValidationSessOpen()

	tests := []testStruct{}

	/* Iterate through data present is separate file. */
	for index, _ := range json_edit_config_create_acl_table_dependent_data {
		tests = append(tests, testStruct{filedescription: "ACL_DATA", cfgData: json_edit_config_create_acl_rule_config_data[index],
			depData: json_edit_config_create_acl_table_dependent_data[index], retCode: cvl.CVL_SUCCESS})
	}

	for index, tc := range tests {
		t.Logf("Running Testcase %d with Description %s", index+1, tc.filedescription)
		t.Run(tc.filedescription, func(t *testing.T) {
			jsonEditCfg_Create_DependentMap := convertDataStringToMap(t, tc.depData)
			jsonEditCfg_Create_ConfigMap := convertDataStringToMap(t, tc.cfgData)

			cfgData := []cvl.CVLEditConfigData{
				cvl.CVLEditConfigData{cvl.VALIDATE_ALL, cvl.OP_CREATE, "ACL_TABLE|TestACL1", jsonEditCfg_Create_DependentMap},
			}


			cvlErrObj, err := cvSess.ValidateEditConfig(cfgData)

			if err != tc.retCode {
				t.Errorf("Config Validation failed. %v", cvlErrObj)
			}

			cfgData = []cvl.CVLEditConfigData{
				cvl.CVLEditConfigData{cvl.VALIDATE_ALL, cvl.OP_CREATE, "ACL_RULE|TestACL1|Rule1", jsonEditCfg_Create_ConfigMap},
			}


			cvlErrObj, err = cvSess.ValidateEditConfig(cfgData)

			if err != tc.retCode {
				t.Errorf("Config Validation failed. %v", cvlErrObj)
			}
		})
	}

	cvl.ValidationSessClose(cvSess)
}
/* API when config is given as string buffer. */
func TestValidateConfig_CfgStrBuffer(t *testing.T) {
	type testStruct struct {
		filedescription string
		jsonString      string
		retCode         cvl.CVLRetCode
	}

	tests := []testStruct{}

	for index, _ := range json_validate_config_data {
		// Fetch the modelName. 
		result := strings.Split(json_validate_config_data[index], "{")
		modelName := strings.Trim(strings.Replace(strings.TrimSpace(result[1]), "\"", "", -1), ":")

		tests = append(tests, testStruct{filedescription: modelName, jsonString: json_validate_config_data[index], retCode: cvl.CVL_SUCCESS})
	}

	cvSess, _ := cvl.ValidationSessOpen()

	for index, tc := range tests {
		t.Logf("Running Testcase %d with Description %s", index+1, tc.filedescription)
		t.Run(fmt.Sprintf("%s [%d]", tc.filedescription, index+1), func(t *testing.T) {
			err := cvSess.ValidateConfig(tc.jsonString)


			if err != tc.retCode {
				t.Errorf("Config Validation failed.")
			}

		})
	}

	 cvl.ValidationSessClose(cvSess)

}


/* API when config is given as json file. */
func TestValidateConfig_CfgFile(t *testing.T) {

	/* Structure containing file information. */
	tests := []struct {
		filedescription string
		fileName        string
		retCode         cvl.CVLRetCode
	}{
		{filedescription: "Config File - VLAN,ACL,PORTCHANNEL", fileName: "testdata/config_db1.json", retCode: cvl.CVL_SUCCESS},
	}

	cvSess, _ := cvl.ValidationSessOpen()

	for index, tc := range tests {

		t.Logf("Running Testcase %d with Description %s", index+1, tc.filedescription)
		t.Run(tc.filedescription, func(t *testing.T) {
			jsonString := convertJsonFileToString(t, tc.fileName)
			err := cvSess.ValidateConfig(jsonString)


			if err != tc.retCode {
				t.Errorf("Config Validation failed.")
			}

		})
	}

	 cvl.ValidationSessClose(cvSess)
}

//Validate invalid json data
func TestValidateConfig_Negative(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()
	jsonData := `{
		"VLANjunk": {
			"Vlan100": {
				"members": [
				"Ethernet4",
				"Ethernet8"
				],
				"vlanid": "100"
			}
		}
	}`

	err := cvSess.ValidateConfig(jsonData)

	if err == cvl.CVL_SUCCESS { //Should return failure
		t.Errorf("Config Validation failed.")
	}

	cvl.ValidationSessClose(cvSess)
}

/* Delete Existing Key.*/
/*
func TestValidateEditConfig_Delete_Semantic_ACLTableReference_Positive(t *testing.T) {

	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1005": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
		"ACL_RULE": map[string]interface{} {
			"TestACL1005|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"L4_SRC_PORT": "1909",
				"IP_PROTOCOL": "103",
				"DST_IP": "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_RULE|TestACL1005|Rule1",
			map[string]string{},
		},
	}

	cvSess, _ := cvl.ValidationSessOpen()

	cvlErrInfo, err := cvSess.ValidateEditConfig(cfgData)

	cvl.ValidationSessClose(cvSess)

	if err != cvl.CVL_SUCCESS {
		t.Errorf("Config Validation failed -- error details %v", cvlErrInfo)
	}

	unloadConfigDB(rclient, depDataMap)
}
*/

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_Valid_FieldValue(t *testing.T) {

	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":           "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

/* API to test edit config with invalid field value. */
func TestValidateEditConfig_Create_Syntax_CableLength(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"CABLE_LENGTH|AZURE",
			map[string]string{
			  "Ethernet8": "5m",
			  "Ethernet12": "5m",
			  "PortChannel16": "5m",
			},
		},
	 }

	 verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SYNTAX_ERROR,
		TableName:        "CABLE_LENGTH",
		Keys:             []string{"AZURE"},
		Field:            "port",
		Value:            "", // BUG: cvl is not filling value "PortChannel16"
		Msg:              invalidValueErrMessage,
		ConstraintErrMsg: "Invalid interface name",
		ErrAppTag:        "interface-name-invalid",
	})
}

/* API to test edit config with invalid field value. */
func TestValidateEditConfig_Create_Syntax_Invalid_FieldValue(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{cvl.VALIDATE_ALL, cvl.OP_CREATE, "ACL_TABLE|TestACL1", map[string]string{
			"stage": "INGRESS",
			"type":  "junk",
		},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_TABLE",
		Keys:      []string{"TestACL1"},
		Field:     "type",
		Value:     "junk",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_Invalid_PacketAction_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD777",
				"IP_TYPE": "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":    "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "PACKET_ACTION",
		Value:     "FORWARD777",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_Invalid_SrcPrefix_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE": "IPV4",
				"SRC_IP":            "10.1.1.1/3288888",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "SRC_IP",
		Value:     "10.1.1.1/3288888",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_InvalidIPAddress_Negative(t *testing.T) {

	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1a.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "SRC_IP",
		Value:     "10.1a.1.1/32",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_OutofBound_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "19099090909090",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "L4_SRC_PORT",
		Value:     "19099090909090",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_InvalidProtocol_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "10388888",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "IP_PROTOCOL",
		Value:     "10388888",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
//Note: Syntax check is done first before dependency check
//hence ACL_TABLE is not required here
func TestValidateEditConfig_Create_Syntax_InvalidRange_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "777779000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "L4_DST_PORT_RANGE",
		Value:     "777779000-12000",
		Msg:       invalidValueErrMessage,
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Create_Syntax_InvalidCharNEw_Negative(t *testing.T) {


	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1jjjj|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1jjjj", "Rule1"},
		// Field:     "aclname",  /* BUG: cvl is not filling Field & Value */
		// Value:     "TestACL1jjjj",
		ConstraintErrMsg: "No instance found for 'TestACL1jjjj'",
		ErrAppTag:        "instance-required",
	})
}

func TestValidateEditConfig_Create_Syntax_SpecialChar_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule@##",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Syntax_InvalidKeyName_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"AC&&***L_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SYNTAX_ERROR,
		Msg:     "Invalid table or key for AC&&***L_RULE|TestACL1|Rule1",
	})
}

func TestValidateEditConfig_Create_Semantic_AdditionalInvalidNode_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL1": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
				"extra":             "shhs",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
		Field:     "extra",
		Msg:       unknownFieldErrMessage,
	})
}

func TestValidateEditConfig_Create_Semantic_MissingMandatoryNode_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VXLAN_TUNNEL|Tunnel1",
			map[string]string{
				"NULL": "NULL",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SYNTAX_MISSING_FIELD,
		TableName: "VXLAN_TUNNEL",
		Keys:      []string{"Tunnel1"},
		Field:     "src_ip",
		Msg:       invalidValueErrMessage,
	})
}

func TestValidateEditConfig_Create_Syntax_Invalid_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULERule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SYNTAX_ERROR,
		Msg:     "Invalid table or key for ACL_RULERule1",
	})
}

func TestValidateEditConfig_Create_Syntax_IncompleteKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SYNTAX_MISSING_FIELD,
		TableName: "ACL_RULE",
		Field:     "aclname",
		Msg:       invalidValueErrMessage,
	})
}

func TestValidateEditConfig_Create_Syntax_InvalidKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SYNTAX_ERROR,
		Msg:     "Invalid table or key for |Rule1",
	})
}

/*
func TestValidateEditConfig_Update_Syntax_DependentData_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"MIRROR_SESSION|everflow",
			map[string]string{
				"src_ip": "10.1.0.32",
				"dst_ip": "2.2.2.2",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|MyACL11_ACL_IPV4|RULE_1",
			map[string]string{
				"MIRROR_ACTION": "everflow",
			},
		},
	}

	cvSess, _ := cvl.ValidationSessOpen()

	cvlErrObj, err := cvSess.ValidateEditConfig(cfgData)

	cvl.ValidationSessClose(cvSess)

	WriteToFile(fmt.Sprintf("\nCVL Error Info is  %v\n", cvlErrObj))

	if err == cvl.CVL_SUCCESS {
		t.Errorf("Config Validation failed -- error details %v", cvlErrObj)
	}

}

func TestValidateEditConfig_Create_Syntax_DependentData_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL|ch1",
			map[string]string{
				"admin_status": "up",
				"mtu":          "9100",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL|ch2",
			map[string]string{
				"admin_status": "up",
				"mtu":          "9100",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL_MEMBER|ch1|Ethernet4",
			map[string]string{},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL_MEMBER|ch1|Ethernet8",
			map[string]string{},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL_MEMBER|ch2|Ethernet12",
			map[string]string{},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL_MEMBER|ch2|Ethernet16",
			map[string]string{},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_NONE,
			"PORTCHANNEL_MEMBER|ch2|Ethernet20",
			map[string]string{},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "102",
				"members@": "Ethernet24,ch1,Ethernet8",
			},
		},
	}

	cvSess, _ := cvl.ValidationSessOpen()

	cvlErrInfo, err := cvSess.ValidateEditConfig(cfgData)

	cvl.ValidationSessClose(cvSess)

	WriteToFile(fmt.Sprintf("\nCVL Error Info is  %v\n", cvlErrInfo))

	if err == cvl.CVL_SUCCESS {
		t.Errorf("Config Validation failed -- error details %v", cvlErrInfo)
	}

}
*/

func TestValidateEditConfig_Delete_Syntax_InvalidKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SYNTAX_ERROR,
		Msg:     "Invalid table or key for |Rule1",
	})
}

func TestValidateEditConfig_Update_Syntax_InvalidKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode: CVL_SYNTAX_ERROR,
		Msg:     "Invalid table or key for |Rule1",
	})
}

func TestValidateEditConfig_Delete_InvalidKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_RULE|TestACL1:Rule1",
			map[string]string{
				"PACKET_ACTION": "",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SYNTAX_MISSING_FIELD,
		TableName: "ACL_RULE",
		Field:     "aclname",
		Msg:       invalidValueErrMessage,
	})
}

func TestValidateEditConfig_Update_Semantic_Invalid_Key_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|TestACL1Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103uuuu",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SYNTAX_MISSING_FIELD,
		TableName: "ACL_RULE",
		Field:     "aclname",
		Msg:       invalidValueErrMessage,
	})
}

func TestValidateEditConfig_Delete_Semantic_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"MIRROR_SESSION": map[string]interface{}{
			"everflow": map[string]interface{}{
				"src_ip": "10.1.0.32",
				"dst_ip": "2.2.2.2",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"MIRROR_SESSION|everflow",
			map[string]string{},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Delete_Semantic_KeyNotExisting_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"MIRROR_SESSION|everflow0",
			map[string]string{},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_NOT_EXIST,
		TableName: "MIRROR_SESSION",
		Keys:      []string{"everflow0"},
	})
}

func TestValidateEditConfig_Update_Semantic_MissingKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|TestACL177|Rule1",
			map[string]string{
				"MIRROR_ACTION": "everflow",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_NOT_EXIST,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL177", "Rule1"},
	})
}

func TestValidateEditConfig_Create_Duplicate_Key_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{} {
			"TestACL100": map[string]interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	//Load same key in DB
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL100",
			map[string]string{
				"stage": "INGRESS",
				"type":  "L3",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_ALREADY_EXIST,
		TableName: "ACL_TABLE",
		Keys:      []string{"TestACL100"},
	})
}

/* API to test edit config with valid syntax. */
func TestValidateEditConfig_Update_Semantic_Positive(t *testing.T) {

	// Create ACL Table.
	fileName := "testdata/create_acl_table.json"
	aclTableMapByte, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("read file %v err: %v", fileName, err)
	}

	mpi_acl_table_map := loadConfig("", aclTableMapByte)
	loadConfigDB(rclient, mpi_acl_table_map)
	defer unloadConfigDB(rclient, mpi_acl_table_map)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_TABLE|TestACL1",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

/* API to test edit config with valid syntax. */
func TestValidateConfig_Semantic_Vlan_Negative(t *testing.T) {

	cvSess, _ := cvl.ValidationSessOpen()

	jsonData := `{
                        "VLAN": {
                                "Vlan100": {
                                        "members": [
                                        "Ethernet44",
                                        "Ethernet64"
                                        ],
                                        "vlanid": "107"
                                }
                        }
                }`

	err := cvSess.ValidateConfig(jsonData)

	if err == cvl.CVL_SUCCESS { //Expected semantic failure
		t.Errorf("Config Validation failed -- error details.")
	}

	cvl.ValidationSessClose(cvSess)
}

func TestValidateEditConfig_Update_Syntax_DependentData_Redis_Positive(t *testing.T) {

	// Create ACL Table.
	fileName := "testdata/create_acl_table13.json"
	aclTableMapByte, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("read file %v err: %v", fileName, err)
	}

	mpi_acl_table_map := loadConfig("", aclTableMapByte)
	loadConfigDB(rclient, mpi_acl_table_map)
	defer unloadConfigDB(rclient, mpi_acl_table_map)

	// Create ACL Rule.
	fileName = "testdata/acl_rule.json"
	aclTableMapRule, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("read file %v err: %v", fileName, err)
	}

	mpi_acl_table_rule := loadConfig("", aclTableMapRule)
	loadConfigDB(rclient, mpi_acl_table_rule)
	defer unloadConfigDB(rclient, mpi_acl_table_rule)

	depDataMap := map[string]interface{}{
		"MIRROR_SESSION": map[string]interface{}{
			"everflow2": map[string]interface{}{
				"src_ip": "10.1.0.32",
				"dst_ip": "2.2.2.2",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	/* ACL and Rule name pre-created . */
	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|TestACL13|Rule1",
			map[string]string{
				"MIRROR_ACTION": "everflow2",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Update_Syntax_DependentData_Invalid_Op_Seq(t *testing.T) {

	/* ACL and Rule name pre-created . */
	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL1",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_NONE,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION": "DROP",
				"L4_SRC_PORT":   "781",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_NOT_EXIST,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
	})
}

func TestValidateEditConfig_Update_Syntax_DependentData_Redis_Negative(t *testing.T) {

	/* ACL does not exist.*/
	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"MIRROR_ACTION": "everflow0",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_NOT_EXIST,
		TableName: "ACL_RULE",
		Keys:      []string{"TestACL1", "Rule1"},
	})
}

/* Create with User provided dependent data. */
func TestValidateEditConfig_Create_Syntax_DependentData_Redis_Positive(t *testing.T) {

	/* ACL and Rule name pre-created . */
	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL22",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	cvSess := NewTestSession(t)

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)

	cfgData = []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL22|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)
}

/* Delete Non-Existing Key.*/
func TestValidateEditConfig_Delete_Semantic_ACLTableReference_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_RULE|MyACLTest_ACL_IPV4|Test_1",
			map[string]string{},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   cvl.CVL_SEMANTIC_KEY_NOT_EXIST,
		TableName: "ACL_RULE",
		Keys:      []string{"MyACLTest_ACL_IPV4", "Test_1"},
	})
}

func TestValidateEditConfig_Create_Dependent_CacheData(t *testing.T) {

	cvSess := NewTestSession(t)

	//Create ACL rule
	cfgDataAcl := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL14",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgDataAcl)
	verifyErr(t, cvlErrInfo, Success)

	//Create ACL rule
	cfgDataRule := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL14|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgDataRule)
	verifyErr(t, cvlErrInfo, Success)
}

func TestValidateEditConfig_Create_DepData_In_MultiSess(t *testing.T) {

	//Create ACL rule - Session 1
	cvSess, _ := cvl.ValidationSessOpen()
	cfgDataAcl := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL16",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	cvlErrInfo, err1 := cvSess.ValidateEditConfig(cfgDataAcl)

	cvl.ValidationSessClose(cvSess)

	//Create ACL rule - Session 2, validation should fail
	cvSess, _ = cvl.ValidationSessOpen()
	cfgDataRule := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL16|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	_, err2 := cvSess.ValidateEditConfig(cfgDataRule)


	cvl.ValidationSessClose(cvSess)

	if err1 != cvl.CVL_SUCCESS || err2 == cvl.CVL_SUCCESS {
		t.Errorf("Config Validation failed -- error details %v", cvlErrInfo)
	}

}

func TestValidateEditConfig_Create_DepData_From_Redis_Negative11(t *testing.T) {

	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataRule := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL188|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataRule, CVLErrorInfo{
		ErrCode:          cvl.CVL_SEMANTIC_DEPENDENT_DATA_MISSING,
		TableName:        "ACL_RULE",
		Keys:             []string{"TestACL188", "Rule1"},
		ConstraintErrMsg: "No instance found for 'TestACL188'",
		ErrAppTag:        "instance-required",
	})
}


func TestValidateEditConfig_Create_DepData_From_Redis(t *testing.T) {

	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataRule := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataRule, Success)
}

func TestValidateEditConfig_Create_Syntax_ErrAppTag_In_Range_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan701",
			map[string]string{
				"vlanid":   "7001",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SYNTAX_ERROR,
		TableName:        "VLAN",
		Keys:             []string{"Vlan701"},
		Field:            "vlanid",
		Msg:              invalidValueErrMessage,
		ConstraintErrMsg: "Vlan ID out of range",
		ErrAppTag:        "vlanid-invalid",
	})
}

func TestValidateEditConfig_Create_Syntax_ErrAppTag_In_Length_Negative(t *testing.T) {
	longText := "A12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL1",
			map[string]string{
				"stage": "INGRESS",
				"type":  "MIRROR",
				"policy_desc": longText,
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SYNTAX_ERROR,
		TableName: "ACL_TABLE",
		Keys:      []string{"TestACL1"},
		Field:     "policy_desc",
		Value:     longText,
		Msg:       invalidValueErrMessage,
		ErrAppTag: "policy-desc-invalid-length",
	})
}

func TestValidateEditConfig_Create_Syntax_ErrAppTag_In_Pattern_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan5001",
			map[string]string{
				"vlanid":   "102",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SYNTAX_ERROR,
		TableName:        "VLAN",
		Keys:             []string{"Vlan5001"},
		Field:            "name",
		Msg:              invalidValueErrMessage,
		ConstraintErrMsg: "Invalid Vlan name pattern",
		ErrAppTag:        "vlan-name-invalid",
	})
}

//EditConfig(Delete) deleting entry already used by other table as leafref
func TestValidateEditConfig_Delete_Dep_Leafref_Negative(t *testing.T) {
	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
		"ACL_RULE": map[string]interface{} {
			"TestACL1|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP": "10.1.1.1/32",
				"L4_SRC_PORT": "1909",
				"IP_PROTOCOL": "103",
				"DST_IP": "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataVlan := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_TABLE|TestACL1",
			map[string]string {
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataVlan, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "ACL_TABLE",
		Keys:      []string{"TestACL1"},
		Msg:       instanceInUseErrMessage,
		ErrAppTag: "instance-in-use",
	})
}

func TestValidateEditConfig_Create_Syntax_RangeValidation(t *testing.T) {
	t.Run("success", func(tt *testing.T) {
		data := []CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "PORTCHANNEL|PortChannel100",
			Data:  map[string]string{"mtu": "5555", "admin_status": "up"},
		}}
		verifyValidateEditConfig(tt, data, Success)
	})

	t.Run("failure_with_errmsg", func(tt *testing.T) {
		data := []CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "PORTCHANNEL|PortChannel100",
			Data:  map[string]string{"mtu": "1", "admin_status": "up"},
		}}
		verifyValidateEditConfig(tt, data, CVLErrorInfo{
			ErrCode:          CVL_SYNTAX_ERROR,
			TableName:        "PORTCHANNEL",
			Keys:             []string{"PortChannel100"},
			Field:            "mtu",
			Msg:              invalidValueErrMessage,
			ConstraintErrMsg: "Invalid MTU value",
			ErrAppTag:        "mtu-invalid",
		})
	})

	t.Run("failure_no_errmsg", func(tt *testing.T) {
		data := []CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "ACL_RULE|ONE|rule100",
			Data:  map[string]string{"PRIORITY": "65535", "IP_PROTOCOL": "4444"},
		}}
		verifyValidateEditConfig(tt, data, CVLErrorInfo{
			ErrCode:   CVL_SYNTAX_ERROR,
			TableName: "ACL_RULE",
			Keys:      []string{"ONE", "rule100"},
			Field:     "IP_PROTOCOL",
			Value:     "4444",
			Msg:       invalidValueErrMessage,
		})
	})

	t.Run("failure_datatype_err", func(tt *testing.T) {
		data := []CVLEditConfigData{{
			VType: VALIDATE_ALL,
			VOp:   OP_CREATE,
			Key:   "PORTCHANNEL|PortChannel100",
			Data:  map[string]string{"mtu": "xyz"}, // mtu is not a number
		}}
		verifyValidateEditConfig(tt, data, CVLErrorInfo{
			// Range will not be evaluated if the value is not a number.. hence generic error
			ErrCode:   CVL_SYNTAX_ERROR,
			TableName: "PORTCHANNEL",
			Keys:      []string{"PortChannel100"},
			Field:     "mtu",
			Value:     "xyz",
			Msg:       invalidValueErrMessage,
		})
	})
}

//Test Initialize() API
func TestLogging(t *testing.T) {
        ret := cvl.Initialize()
        str := "Testing"
        cvl.CVL_LOG(INFO ,"This is Info Log %s", str)
        cvl.CVL_LOG(WARNING,"This is Warning Log %s", str)
        cvl.CVL_LOG(ERROR ,"This is Error Log %s", str)
        cvl.CVL_LOG(INFO_API ,"This is Info API %s", str)
        cvl.CVL_LOG(INFO_TRACE ,"This is Info Trace %s", str)
        cvl.CVL_LOG(INFO_DEBUG ,"This is Info Debug %s", str)
        cvl.CVL_LOG(INFO_DATA ,"This is Info Data %s", str)
        cvl.CVL_LOG(INFO_DETAIL ,"This is Info Detail %s", str)
        cvl.CVL_LOG(INFO_ALL ,"This is Info all %s", str)

        if (ret != cvl.CVL_SUCCESS) {
                t.Errorf("CVl initialization failed")
        }

        cvl.Finish()

	//Initialize again for other test cases to run
	cvl.Initialize()
}

func TestValidateEditConfig_DepData_Through_Cache(t *testing.T) {
	depDataMap := map[string]interface{} {
		"PORT" : map[string]interface{} {
			"Ethernet3" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
				"mtu": "9100",
			},
			"Ethernet5" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "85,86,87,89",
				"mtu": "9100",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	//Modify entry
	modDepDataMap := map[string]interface{} {
		"PORT" : map[string]interface{} {
			"Ethernet3" : map[string]interface{} {
				"mtu": "9200",
			},
		},
	}

	loadConfigDB(rclient, modDepDataMap)
	defer unloadConfigDB(rclient, modDepDataMap)

	cfgDataAclRule :=  []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_TABLE|TestACL1",
			map[string]string {
				"stage": "INGRESS",
				"type": "L3",
				"ports@":"Ethernet3,Ethernet5",
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataAclRule, Success)
}

/* Delete field for an existing key.*/
func TestValidateEditConfig_Delete_Single_Field_Positive(t *testing.T) {

	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
				"policy_desc":"Test ACL desc",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"ACL_TABLE|TestACL1",
			map[string]string{
				"policy_desc":"Test ACL desc",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Dscp_To_Tc_Map(t *testing.T) {
	cfgData := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"DSCP_TO_TC_MAP|AZURE",
			map[string]string {
				"1": "7",
				"2": "8",
				"3": "9",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateConfig_Repeated_Keys_Positive(t *testing.T) {
	jsonData := `{
		"WRED_PROFILE": {
			"AZURE_LOSSLESS": {
				"red_max_threshold": "312000",
				"wred_green_enable": "true",
				"ecn": "ecn_all",
				"green_min_threshold": "104000",
				"red_min_threshold": "104000",
				"wred_yellow_enable": "true",
				"yellow_min_threshold": "104000",
				"wred_red_enable": "true",
				"yellow_max_threshold": "312000",
				"green_max_threshold": "312000"
			}
		},
		"SCHEDULER": {
			"scheduler.0": {
				"type": "DWRR",
				"weight": "25"
			},
			"scheduler.1": {
				"type": "DWRR",
				"weight": "30"
			},
			"scheduler.2": {
				"type": "DWRR",
				"weight": "20"
			}
		},
		"QUEUE": {
			"Ethernet0,Ethernet4,Ethernet8,Ethernet12,Ethernet16,Ethernet20,Ethernet24,Ethernet28,Ethernet32,Ethernet36,Ethernet40,Ethernet44,Ethernet48,Ethernet52,Ethernet56,Ethernet60,Ethernet64,Ethernet68,Ethernet72,Ethernet76,Ethernet80,Ethernet84,Ethernet88,Ethernet92,Ethernet96,Ethernet100,Ethernet104,Ethernet108,Ethernet112,Ethernet116,Ethernet120,Ethernet124|0": {
				"scheduler": "[SCHEDULER|scheduler.1]"
			},
			"Ethernet0,Ethernet4,Ethernet8,Ethernet12,Ethernet16,Ethernet20,Ethernet24,Ethernet28,Ethernet32,Ethernet36,Ethernet40,Ethernet44,Ethernet48,Ethernet52,Ethernet56,Ethernet60,Ethernet64,Ethernet68,Ethernet72,Ethernet76,Ethernet80,Ethernet84,Ethernet88,Ethernet92,Ethernet96,Ethernet100,Ethernet104,Ethernet108,Ethernet112,Ethernet116,Ethernet120,Ethernet124|1": {
				"scheduler": "[SCHEDULER|scheduler.2]"
			},
			"Ethernet0,Ethernet4,Ethernet8,Ethernet12,Ethernet16,Ethernet20,Ethernet24,Ethernet28,Ethernet32,Ethernet36,Ethernet40,Ethernet44,Ethernet48,Ethernet52,Ethernet56,Ethernet60,Ethernet64,Ethernet68,Ethernet72,Ethernet76,Ethernet80,Ethernet84,Ethernet88,Ethernet92,Ethernet96,Ethernet100,Ethernet104,Ethernet108,Ethernet112,Ethernet116,Ethernet120,Ethernet124|3-4": {
				"wred_profile": "[WRED_PROFILE|AZURE_LOSSLESS]",
				"scheduler": "[SCHEDULER|scheduler.0]"
			}
		}
	}`

	cvSess, _ := cvl.ValidationSessOpen()
	err := cvSess.ValidateConfig(jsonData)

	if err != cvl.CVL_SUCCESS {
		t.Errorf("Config Validation failed -- error details.")
	}

	cvl.ValidationSessClose(cvSess)
}

func TestValidateEditConfig_Delete_Entry_Then_Dep_Leafref_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan20": map[string] interface{} {
				"vlanid": "20",
			},
		},
		"VLAN_MEMBER": map[string]interface{} {
			"Vlan20|Ethernet4": map[string] interface{} {
				"tagging_mode": "tagged",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	cfgDataAcl := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"VLAN_MEMBER|Vlan20|Ethernet4",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgDataAcl)
	verifyErr(t, cvlErrInfo, Success)

	cfgDataAcl = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_NONE,
			cvl.OP_DELETE,
			"VLAN_MEMBER|Vlan20|Ethernet4",
			map[string]string {
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"VLAN|Vlan20",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgDataAcl)
	verifyErr(t, cvlErrInfo, Success)
}

/*
func TestBadSchema(t *testing.T) {
	env := os.Environ()
	env[0] = env[0] + " "

	if _, err := os.Stat("/usr/sbin/schema"); os.IsNotExist(err) {
		//Corrupt some schema file 
		exec.Command("/bin/sh", "-c", "/bin/cp testdata/schema/sonic-port.yin testdata/schema/sonic-port.yin.bad" + 
		" && /bin/sed -i '1 a <junk>' testdata/schema/sonic-port.yin.bad").Output()

		//Parse bad schema file
		if module, _ := yparser.ParseSchemaFile("testdata/schema/sonic-port.yin.bad"); module != nil { //should fail
			t.Errorf("Bad schema parsing should fail.")
		}

		//Revert to 
		exec.Command("/bin/sh",  "-c", "/bin/rm testdata/schema/sonic-port.yin.bad").Output()
	} else {
		//Corrupt some schema file 
		exec.Command("/bin/sh", "-c", "/bin/cp /usr/sbin/schema/sonic-port.yin /usr/sbin/schema/sonic-port.yin.bad" + 
		" && /bin/sed -i '1 a <junk>' /usr/sbin/schema/sonic-port.yin.bad").Output()

		//Parse bad schema file
		if module, _ := yparser.ParseSchemaFile("/usr/sbin/schema/sonic-port.yin.bad"); module != nil { //should fail
			t.Errorf("Bad schema parsing should fail.")
		}

		//Revert to 
		exec.Command("/bin/sh",  "-c", "/bin/rm /usr/sbin/schema/sonic-port.yin.bad").Output()
	}

}
*/

/*
func TestServicability_Debug_Trace(t *testing.T) {

	cvl.Debug(false)
	SetTrace(false)

	//Reload the config file by sending SIGUSR2 to ourself
	p, err := os.FindProcess(os.Getpid())
	if (err == nil) {
		p.Signal(syscall.SIGUSR2)
	}


	depDataMap := map[string]interface{}{
		"ACL_TABLE": map[string]interface{}{
			"TestACL1": map[string]interface{}{
				"stage": "INGRESS",
				"type":  "MIRROR",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)

	//Create ACL rule - Session 2
	cvSess, _ := cvl.ValidationSessOpen()
	cfgDataRule := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"ACL_RULE|TestACL1|Rule1",
			map[string]string{
				"PACKET_ACTION":     "FORWARD",
				"IP_TYPE":	     "IPV4",
				"SRC_IP":            "10.1.1.1/32",
				"L4_SRC_PORT":       "1909",
				"IP_PROTOCOL":       "103",
				"DST_IP":            "20.2.2.2/32",
				"L4_DST_PORT_RANGE": "9000-12000",
			},
		},
	}


	cvSess.ValidateEditConfig(cfgDataRule)

	unloadConfigDB(rclient, depDataMap)

	SetTrace(true)
	cvl.Debug(true)

	cvl.ValidationSessClose(cvSess)

	//Reload the  bad config file by sending SIGUSR2 to ourself
	exec.Command("/bin/sh", "-c", "/bin/cp conf/cvl_cfg.json conf/cvl_cfg.json.orig" + 
	" && /bin/echo 'junk' >> conf/cvl_cfg.json").Output()
	p, err = os.FindProcess(os.Getpid())
	if (err == nil) {
		p.Signal(syscall.SIGUSR2)
	}
	exec.Command("/bin/sh",  "-c", "/bin/mv conf/cvl_cfg.json.orig conf/cvl_cfg.json").Output()
	p.Signal(syscall.SIGUSR2)
}*/

// EditConfig(Create) with chained leafref from redis
func TestValidateEditConfig_Delete_Create_Same_Entry_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan100": map[string]interface{} {
				"members@": "Ethernet1",
				"vlanid": "100",
			},
		},
		"PORT" : map[string]interface{} {
			"Ethernet1" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
				"mtu": "9100",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	cfgDataVlan := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"VLAN|Vlan100",
			map[string]string {
			},
		},
	}

	res, _ := cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, res, Success)

	//Same entry getting created again
	cfgDataVlan = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan100",
			map[string]string {
				"vlanid": "100",
			},
		},
	}

	res, _ = cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, res, Success)
}

func TestValidateStartupConfig_Positive(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()
	if cvl.CVL_NOT_IMPLEMENTED != cvSess.ValidateStartupConfig("") {
		t.Errorf("Not implemented yet.")
	}
	cvl.ValidationSessClose(cvSess)
}

func TestValidateIncrementalConfig_Positive(t *testing.T) {
	existingDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan800": map[string]interface{} {
				"members@": "Ethernet1",
				"vlanid": "800",
			},
			"Vlan801": map[string]interface{} {
				"members@": "Ethernet2",
				"vlanid": "801",
			},
		},
		"VLAN_MEMBER": map[string]interface{} {
			"Vlan800|Ethernet1": map[string] interface{} {
				"tagging_mode": "tagged",
			},
		},
		"PORT" : map[string]interface{} {
			"Ethernet1" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "81,82,83,84",
				"mtu": "9100",
			},
			"Ethernet2" : map[string]interface{} {
				"alias":"hundredGigE1",
				"lanes": "85,86,87,89",
				"mtu": "9100",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, existingDataMap)

	cvSess, _ := cvl.ValidationSessOpen()

	jsonData := `{
		"VLAN": {
			"Vlan800": {
				"members": [
				"Ethernet1",
				"Ethernet2"
				],
				"vlanid": "800"
			}
		},
		"VLAN_MEMBER": {
			"Vlan800|Ethernet1": {
				"tagging_mode": "untagged"
			},
			"Vlan801|Ethernet2": {
				"tagging_mode": "tagged"
			}
		}
	}`

	ret := cvSess.ValidateIncrementalConfig(jsonData)

	cvl.ValidationSessClose(cvSess)

	unloadConfigDB(rclient, existingDataMap)

	if ret != cvl.CVL_SUCCESS { //should succeed
		t.Errorf("Config Validation failed.")
		return
	}
}

//Validate key only
func TestValidateKeys(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()
	if cvl.CVL_NOT_IMPLEMENTED != cvSess.ValidateKeys([]string{}) {
		t.Errorf("Not implemented yet.")
	}
	cvl.ValidationSessClose(cvSess)
}

//Validate key and data
func TestValidateKeyData(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()
	if cvl.CVL_NOT_IMPLEMENTED != cvSess.ValidateKeyData("", "") {
		t.Errorf("Not implemented yet.")
	}
	cvl.ValidationSessClose(cvSess)
}

//Validate key, field and value
func TestValidateFields(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()
	if cvl.CVL_NOT_IMPLEMENTED != cvSess.ValidateFields("", "", "") {
		t.Errorf("Not implemented yet.")
	}
	cvl.ValidationSessClose(cvSess)
}

func TestValidateEditConfig_Two_Updates_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgDataAcl := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_TABLE|TestACL1",
			map[string]string {
				"policy_desc": "Test ACL",
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"ACL_TABLE|TestACL1",
			map[string]string {
				"type": "MIRROR",
			},
		},
	}

	verifyValidateEditConfig(t, cfgDataAcl, Success)
}

func TestValidateEditConfig_Create_Syntax_DependentData_PositivePortChannel(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "1001",
				"members@": "Ethernet28,PortChannel002",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}


func TestValidateEditConfig_Create_Syntax_DependentData_PositivePortChannelIfName(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "1001",
				"members@": "Ethernet24,PortChannel001",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Syntax_DependentData_NegativePortChannelEthernet(t *testing.T) {
	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "1001",
				"members@": "PortChannel001,Ethernet4",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SEMANTIC_ERROR,
		TableName:        "VLAN",
		Keys:             []string{"Vlan1001"},
		Field:            "members",
		Value:            "PortChannel001",
		Msg:              mustExpressionErrMessage,
		ConstraintErrMsg: "A vlan interface member cannot be part of portchannel which is already a vlan member",
	})
}

func TestValidateEditConfig_Create_Syntax_DependentData_NegativePortChannelNew(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan1001",
			map[string]string{
				"vlanid":   "1001",
				"members@": "PortChannel003,Ethernet12,PortChannel001",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:   CVL_SEMANTIC_ERROR,
		TableName: "VLAN",
		Keys:      []string{"Vlan1001"},
		Field:     "members",
		//Value:            "Ethernet12", <<< BUG: cvl always fills 1st instance, even thought it was ok
		Msg:              mustExpressionErrMessage,
		ConstraintErrMsg: "A vlan interface member cannot be part of portchannel which is already a vlan member",
	})
}

func TestValidateEditConfig_Use_Updated_Data_As_Create_DependentData_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan201": map[string] interface{} {
				"vlanid":   "201",
				"mtu": "1700",
				"members@": "Ethernet8",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"VLAN|Vlan201",
			map[string]string{
				"mtu": "1900",
				"members@": "Ethernet8,Ethernet12",
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)

	cfgData = []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN_MEMBER|Vlan201|Ethernet8",
			map[string]string{
				"tagging_mode": "tagged",
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)
}

func TestValidateEditConfig_Use_Updated_Data_As_Create_DependentData_Single_Call_Positive(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN" : map[string]interface{} {
			"Vlan201": map[string] interface{} {
				"vlanid":   "201",
				"mtu": "1700",
				"members@": "Ethernet8",
			},
		},
	}

	//Prepare data in Redis
	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"VLAN|Vlan201",
			map[string]string{
				"mtu": "1900",
				"members@": "Ethernet8,Ethernet12",
			},
		},
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN_MEMBER|Vlan201|Ethernet8",
			map[string]string{
				"tagging_mode": "tagged",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Syntax_Interface_AllKeys_Positive(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"INTERFACE|Ethernet24|10.0.0.0/31",
			map[string]string{
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Syntax_Interface_OptionalKey_Positive(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"INTERFACE|Ethernet24",
			map[string]string{
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, Success)
}

func TestValidateEditConfig_Create_Syntax_Interface_IncorrectKey_Negative(t *testing.T) {

	cfgData := []cvl.CVLEditConfigData{
		cvl.CVLEditConfigData{
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"INTERFACE|10.0.0.0/31",
			map[string]string{
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:          CVL_SYNTAX_ERROR,
		TableName:        "INTERFACE",
		Keys:             []string{"10.0.0.0/31"},
		Field:            "portname",
		Msg:              invalidValueErrMessage,
		ConstraintErrMsg: "Invalid interface name",
		ErrAppTag:        "interface-name-invalid",
	})
}

func TestValidateEditConfig_EmptyNode_Positive(t *testing.T) {
        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_UPDATE,
                        "PORT|Ethernet0",
                        map[string]string{
                                "description": "",
                                "index": "3",
                        },
                },
        }

		verifyValidateEditConfig(t, cfgData, Success)
}

func TestSortDepTables(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()

	result, _ := cvSess.SortDepTables([]string{"PORT", "ACL_RULE", "ACL_TABLE"})

	expectedResult := []string{"ACL_RULE", "ACL_TABLE", "PORT"}

	if len(expectedResult) != len(result) {
		t.Errorf("Validation failed, returned value = %v", result)
		return
	}

	for i := 0; i < len(expectedResult) ; i++ {
		if result[i] != expectedResult[i] {
			t.Errorf("Validation failed, returned value = %v", result)
			break
		}
	}

	cvl.ValidationSessClose(cvSess)
}

func TestGetOrderedTables(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()

	result, _ := cvSess.GetOrderedTables("sonic-vlan")

	expectedResult := []string{"VLAN_MEMBER", "VLAN"}

	if len(expectedResult) != len(result) {
		t.Errorf("Validation failed, returned value = %v", result)
		return
	}

	for i := 0; i < len(expectedResult) ; i++ {
		if result[i] != expectedResult[i] {
			t.Errorf("Validation failed, returned value = %v", result)
			break
		}
	}

	cvl.ValidationSessClose(cvSess)
}

func TestGetOrderedDepTables(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()

	result, _ := cvSess.GetOrderedDepTables("sonic-vlan", "VLAN")

	expectedResult := []string{"VLAN_MEMBER", "VLAN"}

	if len(expectedResult) != len(result) {
		t.Errorf("Validation failed, returned value = %v", result)
		return
	}

	for i := 0; i < len(expectedResult) ; i++ {
		if result[i] != expectedResult[i] {
			t.Errorf("Validation failed, returned value = %v", result)
			break
		}
	}

	cvl.ValidationSessClose(cvSess)
}

func TestGetDepTables(t *testing.T) {
	cvSess, _ := cvl.ValidationSessOpen()

	result, _ := cvSess.GetDepTables("sonic-acl", "ACL_RULE")

	expectedResult := []string{"ACL_RULE", "ACL_TABLE", "MIRROR_SESSION", "PORT", "PORTCHANNEL"}

	sort.Strings(result)
	sort.Strings(expectedResult)
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Validation failed, returned value = %v", result)
	}

	cvl.ValidationSessClose(cvSess)
}


func TestGetDepDataForDelete(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VLAN_MEMBER" : map[string]interface{} {
			"Vlan21|Ethernet7": map[string] interface{} {
				"tagging_mode":   "tagged",
			},
			"Vlan22|Ethernet7": map[string] interface{} {
				"tagging_mode":   "tagged",
			},
			"Vlan22|Ethernet72": map[string] interface{} {
				"tagging_mode":   "tagged",
			},
		},
		"PORTCHANNEL_MEMBER" : map[string]interface{} {
			"Ch47|Ethernet7": map[string] interface{} {
				"NULL": "NULL",
			},
			"Ch47|Ethernet75": map[string] interface{} {
				"NULL": "NULL",
			},
		},
		"ACL_TABLE" : map[string]interface{} {
			"TestACL1": map[string] interface{} {
				"stage": "INGRESS",
				"type": "L3",
				"ports@": "Ethernet3,Ethernet76,Ethernet7",
			},
		},
		"CFG_L2MC_STATIC_MEMBER_TABLE" : map[string]interface{} {
			"Vlan24|10.1.1.1|Ethernet7": map[string] interface{} {
				"NULL": "NULL",
			},
			"Vlan25|10.1.1.2|Ethernet78": map[string] interface{} {
				"NULL": "NULL",
			},
		},
		"CFG_L2MC_MROUTER_TABLE" : map[string]interface{} {
			"Vlan21|Ethernet7": map[string] interface{} {
				"NULL": "NULL",
			},
		},
		"MIRROR_SESSION": map[string]interface{}{
			"sess1": map[string]interface{}{
				"src_ip": "10.1.0.32",
				"dst_ip": "2.2.2.2",
			},
		},
		"ACL_RULE" : map[string]interface{} {
			"TestACL1|Rule1": map[string] interface{} {
				"PACKET_ACTION": "FORWARD",
				"MIRROR_ACTION": "sess1",
			},
		},
		"INTERFACE" : map[string]interface{} {
			"Ethernet7": map[string] interface{} {
				"vrf_name": "Vrf1",
			},
			"Ethernet7|10.2.1.1/16": map[string] interface{} {
				"NULL": "NULL",
			},
			"Ethernet7|10.2.1.2/16": map[string] interface{} {
				"NULL": "NULL",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)

        cvSess, _ := cvl.ValidationSessOpen()

	depEntries := cvSess.GetDepDataForDelete("PORT|Ethernet7")

        if (len(depEntries) != 9) { //9 entries to be deleted
                t.Errorf("GetDepDataForDelete() failed")
        }

	depEntries1 := cvSess.GetDepDataForDelete("MIRROR_SESSION|sess1")

        if (len(depEntries1) != 1) { //1 entry to be deleted
                t.Errorf("GetDepDataForDelete() failed")
        }
        cvl.ValidationSessClose(cvSess)

	unloadConfigDB(rclient, depDataMap)
}

func TestMaxElements_All_Entries_In_Request(t *testing.T) {
	cvSess := NewTestSession(t)

        cfgData := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VXLAN_TUNNEL|tun1",
			map[string]string{
				"src_ip": "20.1.1.1",
			},
                },
        }

	//Check addition of first element
	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)

        cfgData1 := []cvl.CVLEditConfigData{
                cvl.CVLEditConfigData{
                        cvl.VALIDATE_ALL,
                        cvl.OP_CREATE,
                        "VXLAN_TUNNEL|tun2",
			map[string]string{
				"src_ip": "30.1.1.1",
			},
                },
        }

	//Try to validate addition of second element
        cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData1)
	verifyErr(t, cvlErrInfo, CVLErrorInfo{
		ErrCode:          CVL_SYNTAX_ERROR,
		TableName:        "VXLAN_TUNNEL",
		Keys:             []string{"tun2"},
		Msg:              "Max elements limit reached",
		ConstraintErrMsg: "Max elements limit 1 reached",
		ErrAppTag:        "too-many-elements",
	})
}

func TestMaxElements_Entries_In_Redis(t *testing.T) {
	depDataMap := map[string]interface{} {
		"VXLAN_TUNNEL" : map[string]interface{} {
			"tun1" : map[string]interface{} {
				"src_ip": "20.1.1.1",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

    t.Run("create_new", func(t *testing.T) {
        cfgData := []CVLEditConfigData{{
            VType: VALIDATE_ALL,
            VOp:   OP_CREATE,
            Key:   "VXLAN_TUNNEL|tun2",
            Data: map[string]string{
                "src_ip": "30.1.1.1",
            },
        }}

        verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
            ErrCode:          CVL_SYNTAX_ERROR,
            TableName:        "VXLAN_TUNNEL",
            Keys:             []string{"tun2"},
            Msg:              "Max elements limit reached",
            ConstraintErrMsg: "Max elements limit 1 reached",
            ErrAppTag:        "too-many-elements",
        })
    })

    t.Run("delete_and_create", func(t *testing.T) {
        cvSess := NewTestSession(t)

        cfgData1 := []CVLEditConfigData{{
            VType: VALIDATE_ALL,
            VOp:   OP_DELETE,
            Key:   "VXLAN_TUNNEL|tun1",
            Data:  map[string]string{},
        }}

        //Delete the existing entry, should succeed
        cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData1)
        verifyErr(t, cvlErrInfo, Success)

        cfgData1 = []CVLEditConfigData{{
            VType: VALIDATE_NONE,
            VOp:   OP_DELETE,
            Key:   "VXLAN_TUNNEL|tun1",
            Data:  map[string]string{},
        }, {
            VType: VALIDATE_ALL,
            VOp: OP_CREATE,
            Key: "VXLAN_TUNNEL|tun2",
            Data: map[string]string{
                "src_ip": "30.1.1.1",
            },
        }}

        //Check validation of new entry, should succeed now
        cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData1)
        verifyErr(t, cvlErrInfo, Success)
	})
}

func TestValidateEditConfig_Two_Create_Requests_Positive(t *testing.T) {
	cvSess := NewTestSession(t)

	cfgDataVlan := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VLAN|Vlan21",
			map[string]string {
				"vlanid": "21",
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, cvlErrInfo, Success)

	cfgDataVlan = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_NONE,
			cvl.OP_CREATE,
			"VLAN|Vlan21",
			map[string]string {
				"vlanid": "21",
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"STP_VLAN|Vlan21",
			map[string]string {
				"enabled": "true",
				"forward_delay": "15",
				"hello_time": "2",
				"max_age" : "20",
				"priority": "327",
				"vlanid": "21",
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, cvlErrInfo, Success)
}

func TestValidateEditConfig_Two_Delete_Requests_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan51": map[string]interface{}{
				"vlanid": "51",
			},
		}, 
		"STP_VLAN": map[string]interface{}{
			"Vlan51": map[string]interface{}{
				"enabled": "true",
				"forward_delay": "15",
				"hello_time": "2",
				"max_age" : "20",
				"priority": "327",
				"vlanid": "51",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cvSess := NewTestSession(t)

	cfgDataVlan := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"STP_VLAN|Vlan51",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, cvlErrInfo, Success)

	cfgDataVlan = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_NONE,
			cvl.OP_DELETE,
			"STP_VLAN|Vlan51",
			map[string]string {
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"VLAN|Vlan51",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgDataVlan)
	verifyErr(t, cvlErrInfo, Success)
}

//Check delete constraing with table having multiple keys
func TestValidateEditConfig_Multi_Delete_MultiKey_Same_Session_Positive(t *testing.T) {
	depDataMap := map[string]interface{}{
		"VLAN": map[string]interface{}{
			"Vlan511": map[string]interface{}{
				"vlanid": "511",
			},
		},
		"VLAN_MEMBER": map[string]interface{}{
			"Vlan511|Ethernet16": map[string]interface{}{
				"tagging_mode": "untagged",
			},
		},
		"STP_VLAN_PORT": map[string]interface{}{
			"Vlan511|Ethernet16": map[string]interface{}{
				"path_cost": "200",
				"priority": "128",
			},
		},
		"STP_PORT": map[string]interface{}{
			"Ethernet16": map[string]interface{}{
				"bpdu_filter": "global",
				"enabled": "true",
				"portfast": "true",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)
	cvSess := NewTestSession(t)

	cfgData := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"STP_VLAN_PORT|Vlan511|Ethernet16",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ := cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)

	cfgData = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"VLAN_MEMBER|Vlan511|Ethernet16",
			map[string]string {
				"tagging_mode": "untagged",
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)

	cfgData = []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_NONE,
			cvl.OP_DELETE,
			"STP_VLAN_PORT|Vlan511|Ethernet16",
			map[string]string {
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_NONE,
			cvl.OP_DELETE,
			"VLAN_MEMBER|Vlan511|Ethernet16",
			map[string]string {
				"tagging_mode": "untagged",
			},
		},
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_DELETE,
			"STP_PORT|Ethernet16",
			map[string]string {
			},
		},
	}

	cvlErrInfo, _ = cvSess.ValidateEditConfig(cfgData)
	verifyErr(t, cvlErrInfo, Success)
}

func TestValidateEditConfig_Update_Leaf_List_Max_Elements_Negative(t *testing.T) {
	depDataMap := map[string]interface{}{
		"VLAN": map[string]interface{} {
			"Vlan801": map[string]interface{} {
				"vlanid": "801",
			},
		},
		"CFG_L2MC_STATIC_GROUP_TABLE": map[string]interface{} {
			"Vlan801|16.2.2.1": map[string]interface{} {
				"out-intf@": "Ethernet4,Ethernet8,Ethernet16",
			},
		},
	}

	loadConfigDB(rclient, depDataMap)
	defer unloadConfigDB(rclient, depDataMap)

	cfgData := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_UPDATE,
			"CFG_L2MC_STATIC_GROUP_TABLE|Vlan801|16.2.2.1",
			map[string]string {
				"out-intf@": "Ethernet4,Ethernet8,Ethernet16,Ethernet20",
			},
		},
	}

	verifyValidateEditConfig(t, cfgData, CVLErrorInfo{
		ErrCode:       CVL_SYNTAX_MAXIMUM_INVALID,
		TableName:     "CFG_L2MC_STATIC_GROUP_TABLE",
		Keys:          []string{"Vlan801", "16.2.2.1"},
		Field:         "out-intf",
		CVLErrDetails: "max-elements constraint not honored",
	})
}

func TestValidationTimeStats(t *testing.T) {
	cvl.ClearValidationTimeStats()

	stats := cvl.GetValidationTimeStats()

	if (stats.Hits != 0 || stats.Time != 0 || stats.Peak != 0) {
		t.Errorf("TestValidationTimeStats : clearing stats failed")
		return
	}

	cvSess, _ := cvl.ValidationSessOpen()

	cfgData := []cvl.CVLEditConfigData {
		cvl.CVLEditConfigData {
			cvl.VALIDATE_ALL,
			cvl.OP_CREATE,
			"VRF|VrfTest",
			map[string]string {
				"fallback": "true",
			},
		},
	}

	cvSess.ValidateEditConfig(cfgData)

	cvl.ValidationSessClose(cvSess)

	stats = cvl.GetValidationTimeStats()

	if (stats.Hits == 0 || stats.Time == 0 || stats.Peak == 0) {
		t.Errorf("TestValidationTimeStats : getting stats failed")
		return
	}

	//Clear stats again and check
	cvl.ClearValidationTimeStats()

	stats = cvl.GetValidationTimeStats()

	if (stats.Hits != 0 || stats.Time != 0 || stats.Peak != 0) {
		t.Errorf("TestValidationTimeStats : clearing stats failed")
	}
}
