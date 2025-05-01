////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Dell, Inc.                                                 //
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

//go:build xfmrtest
// +build xfmrtest

package transformer

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

func init() {
	// Pre and Post transformer functions
	XlateFuncBind("test_pre_xfmr", test_pre_xfmr)
	XlateFuncBind("test_post_xfmr", test_post_xfmr)

	// Table transformer functions
	XlateFuncBind("test_sensor_type_tbl_xfmr", test_sensor_type_tbl_xfmr)
	XlateFuncBind("test_ni_instance_protocol_table_xfmr", test_ni_instance_protocol_table_xfmr)

	// Key transformer functions
	XlateFuncBind("YangToDb_test_sensor_type_key_xfmr", YangToDb_test_sensor_type_key_xfmr)
	XlateFuncBind("DbToYang_test_sensor_type_key_xfmr", DbToYang_test_sensor_type_key_xfmr)
	XlateFuncBind("YangToDb_test_sensor_zone_key_xfmr", YangToDb_test_sensor_zone_key_xfmr)
	XlateFuncBind("DbToYang_test_sensor_zone_key_xfmr", DbToYang_test_sensor_zone_key_xfmr)
	XlateFuncBind("YangToDb_test_set_key_xfmr", YangToDb_test_set_key_xfmr)
	XlateFuncBind("DbToYang_test_set_key_xfmr", DbToYang_test_set_key_xfmr)
	XlateFuncBind("YangToDb_sensor_a_light_sensor_key_xfmr", YangToDb_sensor_a_light_sensor_key_xfmr)
	XlateFuncBind("DbToYang_sensor_a_light_sensor_key_xfmr", DbToYang_sensor_a_light_sensor_key_xfmr)
	//XlateFuncBind("YangToDb_test_ntp_authentication_key_xfmr", YangToDb_test_ntp_authentication_key_xfmr)
	//XlateFuncBind("DbToYang_test_ntp_authentication_key_xfmr", DbToYang_test_ntp_authentication_key_xfmr)
	XlateFuncBind("YangToDb_test_ni_instance_key_xfmr", YangToDb_test_ni_instance_key_xfmr)
	XlateFuncBind("DbToYang_test_ni_instance_key_xfmr", DbToYang_test_ni_instance_key_xfmr)
	XlateFuncBind("YangToDb_test_ni_instance_protocol_key_xfmr", YangToDb_test_ni_instance_protocol_key_xfmr)
	XlateFuncBind("DbToYang_test_ni_instance_protocol_key_xfmr", DbToYang_test_ni_instance_protocol_key_xfmr)
	XlateFuncBind("YangToDb_test_bgp_network_cfg_key_xfmr", YangToDb_test_bgp_network_cfg_key_xfmr)
	XlateFuncBind("DbToYang_test_bgp_network_cfg_key_xfmr", DbToYang_test_bgp_network_cfg_key_xfmr)
	XlateFuncBind("YangToDb_test_ospfv2_router_distribution_key_xfmr", YangToDb_test_ospfv2_router_distribution_key_xfmr)
	XlateFuncBind("DbToYang_test_ospfv2_router_distribution_key_xfmr", DbToYang_test_ospfv2_router_distribution_key_xfmr)
	XlateFuncBind("YangToDb_test_ospfv2_router_key_xfmr", YangToDb_test_ospfv2_router_key_xfmr)

	// Key leafrefed Field transformer functions
	XlateFuncBind("DbToYang_test_sensor_type_field_xfmr", DbToYang_test_sensor_type_field_xfmr)
	XlateFuncBind("DbToYang_test_set_name_field_xfmr", DbToYang_test_set_name_field_xfmr)

	// Field transformer functions
	XlateFuncBind("YangToDb_exclude_filter_field_xfmr", YangToDb_exclude_filter_field_xfmr)
	XlateFuncBind("DbToYang_exclude_filter_field_xfmr", DbToYang_exclude_filter_field_xfmr)
	XlateFuncBind("YangToDb_test_set_type_field_xfmr", YangToDb_test_set_type_field_xfmr)
	XlateFuncBind("DbToYang_test_set_type_field_xfmr", DbToYang_test_set_type_field_xfmr)
	XlateFuncBind("YangToDb_test_set_description_field_xfmr", YangToDb_test_set_description_field_xfmr)
	XlateFuncBind("DbToYang_test_set_description_field_xfmr", DbToYang_test_set_description_field_xfmr)

	//Subtree transformer function
	XlateFuncBind("YangToDb_test_port_bindings_xfmr", YangToDb_test_port_bindings_xfmr)
	XlateFuncBind("DbToYang_test_port_bindings_xfmr", DbToYang_test_port_bindings_xfmr)
	XlateFuncBind("Subscribe_test_port_bindings_xfmr", Subscribe_test_port_bindings_xfmr)

	//validate transformer
	XlateFuncBind("light_sensor_validate", light_sensor_validate)
	XlateFuncBind("validate_bgp_proto", validate_bgp_proto)
	XlateFuncBind("validate_ospfv2_proto", validate_ospfv2_proto)

	// Sonic yang Key transformer functions
	XlateFuncBind("DbToYang_test_sensor_mode_key_xfmr", DbToYang_test_sensor_mode_key_xfmr)
}

const (
	TEST_SET_TABLE = "TEST_SET_TABLE"
	TEST_SET_TYPE  = "type"
	TEST_SET_PORTS = "ports"
	TEST_SET_DESC  = "description"
)

/* E_OpenconfigTestXfmr_TEST_SET_TYPE */
var TEST_SET_TYPE_MAP = map[string]string{
	strconv.FormatInt(int64(ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV4), 10): "IPV4",
	strconv.FormatInt(int64(ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV6), 10): "IPV6",
}

var test_pre_xfmr PreXfmrFunc = func(inParams XfmrParams) error {
	var err error
	log.Info("Entering test_pre_xfmr:- Request URI path = ", inParams.requestUri)
	pathInfo := NewPathInfo(inParams.requestUri)
	rejectReplaceNodes := []string{"/openconfig-test-xfmr:test-xfmr/interfaces",
		"/openconfig-test-xfmr:test-xfmr/test-sensor-groups",
		"/openconfig-test-xfmr:test-xfmr/test-sensor-types",
		"/openconfig-test-xfmr:test-xfmr/test-sets",
	}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)

	if inParams.oper == REPLACE {
		for _, rejectNode := range rejectReplaceNodes {
			if targetUriPath == rejectNode {
				err_str := "REPLACE not supported at this node."
				err = tlerr.NotSupportedError{Format: err_str}
				break
			}
		}
	}
	return err
}

var test_post_xfmr PostXfmrFunc = func(inParams XfmrParams) error {

	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")

	retDbDataMap := (*inParams.dbDataMap)[inParams.curDb]
	log.Info("Entering test_post_xfmr Request URI path = ", inParams.requestUri)
	if inParams.oper == UPDATE {
		xpath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
		if xpath == "/openconfig-test-xfmr:test-xfmr/test-sensor-groups/test-sensor-group/config/color-hold-time" {
			holdTime := retDbDataMap["TEST_SENSOR_GROUP"][groupId].Field["color-hold-time"]
			key := groupId + "|" + "sensor_type_a_post" + holdTime
			subOpCreateMap := make(map[db.DBNum]map[string]map[string]db.Value)
			subOpCreateMap[db.ConfigDB] = make(map[string]map[string]db.Value)
			subOpCreateMap[db.ConfigDB]["TEST_SENSOR_A_TABLE"] = make(map[string]db.Value)
			subOpCreateMap[db.ConfigDB]["TEST_SENSOR_A_TABLE"][key] = db.Value{Field: make(map[string]string)}
			subOpCreateMap[db.ConfigDB]["TEST_SENSOR_A_TABLE"][key].Field["description_a"] = "Added instance in post xfmr"
			inParams.subOpDataMap[CREATE] = &subOpCreateMap
		}
	}
	return nil
}

var test_sensor_type_tbl_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string
	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")
	sensorType := pathInfo.Var("type")

	log.Info("test_sensor_type_tbl_xfmr inParams.uri ", inParams.uri)

	if len(groupId) == 0 {
		return tblList, nil
	}
	if len(sensorType) == 0 {
		if inParams.oper == GET || inParams.oper == DELETE {
			tblList = append(tblList, "TEST_SENSOR_A_TABLE")
			tblList = append(tblList, "TEST_SENSOR_B_TABLE")
		}
	} else {
		if strings.HasPrefix(sensorType, "sensora_") {
			tblList = append(tblList, "TEST_SENSOR_A_TABLE")
		} else if strings.HasPrefix(sensorType, "sensorb_") {
			tblList = append(tblList, "TEST_SENSOR_B_TABLE")
		}
	}
	log.Info("test_sensor_type_tbl_xfmr tblList= ", tblList)
	return tblList, nil
}

var YangToDb_test_sensor_type_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var sensor_type_key string
	var err error

	log.Info("YangToDb_test_sensor_type_key_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")
	sensorType := pathInfo.Var("type")
	if groupId == "" {
		return sensor_type_key, err
	}
	if len(groupId) > 0 {
		sensor_type := ""
		if strings.HasPrefix(sensorType, "sensora_") {
			sensor_type = strings.Replace(sensorType, "sensora_", "sensor_type_a_", 1)
			sensor_type_key = groupId + "|" + sensor_type
		} else if strings.HasPrefix(sensorType, "sensorb_") {
			sensor_type = strings.Replace(sensorType, "sensorb_", "sensor_type_b_", 1)
			sensor_type_key = groupId + "|" + sensor_type
		} else if sensorType == "" && (strings.HasSuffix(inParams.uri, "/test-sensor-type")) && (inParams.oper == GET || inParams.oper == DELETE) {
			sensor_type_key = groupId
		} else {
			err_str := "Invalid key. Key not supported."
			err = tlerr.NotSupported(err_str)
		}
	}
	log.Info("YangToDb_test_sensor_type_key_xfmr returns", sensor_type_key)
	return sensor_type_key, err
}

var DbToYang_test_sensor_type_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {

	rmap := make(map[string]interface{})
	var err error
	if log.V(3) {
		log.Info("Entering DbToYang_test_sensor_type_key_xfmr inParams.uri ", inParams.uri)
	}
	var sensorType string

	if strings.Contains(inParams.key, "|") {
		key_split := strings.Split(inParams.key, "|")
		sensorType = key_split[1]
		if strings.HasPrefix(sensorType, "sensor_type_a_") {
			sensorType = strings.Replace(sensorType, "sensor_type_a_", "sensora_", 1)
		} else if strings.HasPrefix(sensorType, "sensor_type_b_") {
			sensorType = strings.Replace(sensorType, "sensor_type_b_", "sensorb_", 1)
		} else {
			sensorType = ""
			err_str := "Invalid key. Key not supported."
			err = tlerr.NotSupported(err_str)
		}
	}

	rmap["type"] = sensorType

	log.Info("DbToYang_test_sensor_type_key_xfmr rmap ", rmap)
	return rmap, err
}

var YangToDb_test_sensor_zone_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var sensor_zone_key string
	var err error

	log.Info("YangToDb_test_sensor_zone_key_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")
	sensorZone := pathInfo.Var("zone")
	if groupId == "" {
		return sensor_zone_key, err
	}
	if sensorZone == "" && (inParams.oper == DELETE || inParams.oper == GET) {
		return groupId, err
	}
	if len(groupId) > 0 {
		sensor_zone_key = groupId + "|" + sensorZone
	}
	log.Info("YangToDb_test_sensor_zone_key_xfmr returns", sensor_zone_key)
	return sensor_zone_key, err
}

var DbToYang_test_sensor_zone_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {

	rmap := make(map[string]interface{})
	var err error
	if log.V(3) {
		log.Info("Entering DbToYang_test_sensor_zone_key_xfmr inParams.uri ", inParams.uri)
	}
	var sensorZone string

	if strings.Contains(inParams.key, "|") {
		key_split := strings.Split(inParams.key, "|")
		sensorZone = key_split[1]
	}

	rmap["zone"] = sensorZone

	log.Info("DbToYang_test_sensor_zone_key_xfmr rmap ", rmap)
	return rmap, err
}

var YangToDb_test_set_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {

	testSetKey := ""
	if log.V(3) {
		log.Info("Entering DbToYang_testsensor_type_key_xfmr inParams.uri ", inParams.uri)
	}

	pathInfo := NewPathInfo(inParams.uri)
	testSetName := pathInfo.Var("name")
	testSetType := pathInfo.Var("type")

	if len(testSetName) > 0 && len(testSetType) > 0 {
		testSetKey = testSetName + "_" + testSetType
	}
	log.Info(" YangToDb_test_set_key_xfmr returns ", testSetKey)
	return testSetKey, nil

}

var DbToYang_test_set_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	var err error
	if log.V(3) {
		log.Info("DbToYang_test_set_key_xfmr invoked for uri: ", inParams.uri)
	}
	var testSetName string
	var testSetType string

	if len(inParams.key) == 0 {
		return rmap, errors.New("Incorrect dbKey : " + inParams.key)
	}

	if strings.HasSuffix(inParams.key, "TEST_SET_IPV4") {
		testSetType = "TEST_SET_IPV4"
	} else if strings.HasSuffix(inParams.key, "TEST_SET_IPV6") {
		testSetType = "TEST_SET_IPV6"
	}
	testSetName = getTestSetNameCompFromDbKey(inParams.key, testSetType)

	rmap["name"] = testSetName
	rmap["type"] = testSetType

	log.Info("DbToYang_testsensor_type_key_xfmr rmap ", rmap)
	return rmap, err

}

var DbToYang_test_sensor_type_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	log.Info("DbToYang_test_sensor_type_field_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")
	sensorType := pathInfo.Var("type")
	if groupId == "" || sensorType == "" {
		return result, err
	}
	if strings.HasPrefix(sensorType, "sensor") {
		result["type"] = sensorType
	} else {
		errStr := "Invalid Key in uri."
		return result, tlerr.InvalidArgsError{Format: errStr}
	}

	log.Info("DbToYang_test_sensor_type_field_xfmr returns ", result)

	return result, err
}

var DbToYang_test_set_name_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	log.Info("DbToYang_test_set_name_field_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	setName := pathInfo.Var("name")
	setType := pathInfo.Var("type")
	if setName == "" || setType == "" {
		return result, err
	}
	result["name"] = setName

	log.Info("DbToYang_test_set_name_field_xfmrreturns ", result)

	return result, err
}

func getTestSetRoot(s *ygot.GoStruct) *ocbinds.OpenconfigTestXfmr_TestXfmr {
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.TestXfmr
}

func getTestSetKeyStrFromOCKey(setname string, settype ocbinds.E_OpenconfigTestXfmr_TEST_SET_TYPE) string {
	setT := ""
	if settype == ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV4 {
		setT = "TEST_SET_IPV4"
	} else {
		setT = "TEST_SET_IPV6"
	}
	return setname + "_" + setT
}

func getTestSetNameCompFromDbKey(testSetDbKey string, testSetType string) string {
	return testSetDbKey[:strings.LastIndex(testSetDbKey, "_"+testSetType)]
}

var YangToDb_exclude_filter_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var err error
	if inParams.param == nil {
		res_map["exclude_filter"] = ""
		return res_map, err
	}
	exflt, _ := inParams.param.(*string)
	if exflt != nil {
		res_map["exclude_filter"] = "filter_" + *exflt
		log.Info("YangToDb_exclude_filter_field_xfmr ", res_map["exclude_filter"])
	}
	return res_map, err

}

var DbToYang_exclude_filter_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})
	data := (*inParams.dbDataMap)[inParams.curDb]
	log.Info("DbToYang_exclude_filter_field_xfmr", data, inParams.ygRoot)
	pathInfo := NewPathInfo(inParams.uri)
	sensor_type := pathInfo.Var("type")
	tblNm := ""
	if strings.HasPrefix(sensor_type, "sensora_") {
		tblNm = "TEST_SENSOR_A_TABLE"
	} else if strings.HasPrefix(sensor_type, "sensorb_") {
		tblNm = "TEST_SENSOR_B_TABLE"
	}

	sensorData, ok := data[tblNm]
	if ok {
		sensorInst, instOk := sensorData[inParams.key]
		if instOk {
			exFlt, fldOk := sensorInst.Field["exclude_filter"]
			if fldOk {
				result["exclude-filter"] = strings.Split(exFlt, "filter_")[1]
				log.Info("DbToYang_exclude_filter_field_xfmr - returning %v", result["exclude-filter"])
			} else {
				return nil, tlerr.NotFound("Resource Not Found")
			}
		} else {
			log.Info("DbToYang_exclude_filter_field_xfmr - sensor instance %v doesn't exist", inParams.key)
		}
	} else {
		log.Info("DbToYang_exclude_filter_field_xfmr - Table %v does not exist in db Data", tblNm)
	}

	return result, err
}

var YangToDb_test_set_type_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var err error
	if inParams.param == nil {
		return res_map, err
	}

	testSetType, _ := inParams.param.(ocbinds.E_OpenconfigTestXfmr_TEST_SET_TYPE)
	log.Info("YangToDb_test_set_type_field_xfmr: ", inParams.ygRoot, " Xpath: ", inParams.uri, " Type: ", testSetType)
	res_map[TEST_SET_TYPE] = findInMap(TEST_SET_TYPE_MAP, strconv.FormatInt(int64(testSetType), 10))
	return res_map, err
}

var DbToYang_test_set_type_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})
	data := (*inParams.dbDataMap)[inParams.curDb]
	log.Info("DbToYang_test_set_type_field_xfmr", data, inParams.ygRoot)
	oc_testSetType := findInMap(TEST_SET_TYPE_MAP, data[TEST_SET_TABLE][inParams.key].Field[TEST_SET_TYPE])
	n, err := strconv.ParseInt(oc_testSetType, 10, 64)
	if n == int64(ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV4) {
		result[TEST_SET_TYPE] = "TEST_SET_IPV4"
	} else {
		result[TEST_SET_TYPE] = "TEST_SET_IPV6"
	}
	return result, err
}

var DbToYang_test_set_description_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	var err error
	data := (*inParams.dbDataMap)[inParams.curDb]
	log.Info("DbToYang_test_set_description_field_xfmr", data, inParams.ygRoot)
	oc_testSetDesc := strings.SplitN(data[TEST_SET_TABLE][inParams.key].Field[TEST_SET_DESC], "Description : ", 2)[1]
	result[TEST_SET_DESC] = oc_testSetDesc
	log.Info("DbToYang_test_set_description_field_xfmr returning :", result)
	return result, err
}

var YangToDb_test_set_description_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	var err error
	var testSetDesc string
	res_map := make(map[string]string)
	log.Info("YangToDb_test_set_description_field_xfmr called")
	if inParams.param == nil {
		return res_map, err
	}

	testSetDescPtr, _ := inParams.param.(*string)
	if testSetDescPtr != nil {
		testSetDesc = *testSetDescPtr
	}
	res_map[TEST_SET_DESC] = "Description : " + testSetDesc
	log.Info("YangToDb_test_set_description_field_xfmr: ", res_map)
	return res_map, err
}

var YangToDb_test_port_bindings_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	res_map := make(map[string]map[string]db.Value)
	testSetTableMap := make(map[string]db.Value)
	testSetTableMapNew := make(map[string]db.Value)
	log.Info("YangToDb_test_port_bindings_xfmr: ", inParams.ygRoot, inParams.uri)

	testXfmrObj := getTestSetRoot(inParams.ygRoot)
	if testXfmrObj.Interfaces == nil {
		return res_map, err
	}

	testSetTs := &db.TableSpec{Name: TEST_SET_TABLE}
	testSetKeys, err := inParams.d.GetKeys(testSetTs)
	if err != nil {
		return res_map, err
	}

	for key := range testSetKeys {
		testSetEntry, err := inParams.d.GetEntry(testSetTs, testSetKeys[key])
		if err != nil {
			return res_map, err
		}
		testSetTableMap[(testSetKeys[key].Get(0))] = testSetEntry
	}

	testSetInterfacesMap := make(map[string][]string)
	for intfId, _ := range testXfmrObj.Interfaces.Interface {
		intf := testXfmrObj.Interfaces.Interface[intfId]
		if intf == nil {
			continue
		}
		if intf.IngressTestSets != nil && len(intf.IngressTestSets.IngressTestSet) > 0 {
			for inTestSetKey, _ := range intf.IngressTestSets.IngressTestSet {
				testSetName := getTestSetKeyStrFromOCKey(inTestSetKey.SetName, inTestSetKey.Type)
				testSetInterfacesMap[testSetName] = append(testSetInterfacesMap[testSetName], *intf.Id)
				_, ok := testSetTableMap[testSetName]
				if !ok && inParams.oper == DELETE {
					return res_map, tlerr.NotFound("Binding not found for test set  %v on %v", inTestSetKey.SetName, *intf.Id)
				}
				if inParams.oper == DELETE {
					testSetTableMapNew[testSetName] = db.Value{Field: make(map[string]string)}
				} else {
					testSetType := findInMap(TEST_SET_TYPE_MAP, strconv.FormatInt(int64(inTestSetKey.Type), 10))
					testSetTableMapNew[testSetName] = db.Value{Field: map[string]string{"type": testSetType}}
				}
			}
		} else {
			for testSetKey, testSetData := range testSetTableMap {
				ports := testSetData.GetList(TEST_SET_PORTS)
				if contains(ports, *intf.Id) {
					testSetInterfacesMap[testSetKey] = append(testSetInterfacesMap[testSetKey], *intf.Id)
					testSetTableMapNew[testSetKey] = db.Value{Field: make(map[string]string)}
				}

			}

		}
	}
	for k, _ := range testSetInterfacesMap {
		val := testSetTableMapNew[k]
		(&val).SetList(TEST_SET_PORTS+"@", testSetInterfacesMap[k])
	}
	res_map[TEST_SET_TABLE] = testSetTableMapNew
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}
	return res_map, err
}

var DbToYang_test_port_bindings_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error
	var testSetTs *db.TableSpec
	var testSetTblMap map[string]db.Value
	var trustIntf, trustTestSet bool

	log.Info("DbToYang_test_port_bindings_xfmr")

	pathInfo := NewPathInfo(inParams.uri)

	testXfmrObj := getTestSetRoot(inParams.ygRoot)
	cdb := inParams.dbs[db.ConfigDB]
	testSetTs = &db.TableSpec{Name: TEST_SET_TABLE}
	testSetKeys, err := cdb.GetKeys(testSetTs)
	if err != nil {
		return err
	}
	testSetTblMap = make(map[string]db.Value)

	for key := range testSetKeys {
		testSetEntry, err := cdb.GetEntry(testSetTs, testSetKeys[key])
		if err != nil {
			return err
		}
		testSetTblMap[(testSetKeys[key]).Get(0)] = testSetEntry
	}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	interfaces := make(map[string]bool)
	if isSubtreeRequest(targetUriPath, "/openconfig-test-xfmr:test-xfmr/interfaces") || isSubtreeRequest(targetUriPath, "/openconfig-test-xfmr:test-xfmr/interfaces/interface") {
		intfSbt := testXfmrObj.Interfaces
		if nil == intfSbt.Interface || (nil != intfSbt.Interface && len(intfSbt.Interface) == 0) {
			log.Info("Get request for all interfaces")
			for testSetKey := range testSetTblMap {
				testSetData := testSetTblMap[testSetKey]
				if len(testSetData.GetList(TEST_SET_PORTS)) > 0 {
					testSetIntfs := testSetData.GetList(TEST_SET_PORTS)
					for intf := range testSetIntfs {
						interfaces[testSetIntfs[intf]] = true
					}
				}
			}

			// No interface bindings present. Return. This is general Query ie No interface specified
			// by the user. We should return no error
			if len(interfaces) == 0 {
				return nil
			}
			trustIntf = true
			// For each binding present, create Ygot tree to process next level.
			ygot.BuildEmptyTree(intfSbt)
			for intfId := range interfaces {
				ptr, _ := intfSbt.NewInterface(intfId)
				ygot.BuildEmptyTree(ptr)
			}
		} else {
			log.Info("Get request for specific interface")
		}

		// For each interface present, Process it. The interface present could be created as part of
		// of the URI or created above
		for ifName, ocIntfPtr := range intfSbt.Interface {
			log.Infof("Processing get request for %s", *ocIntfPtr.Id)
			if !trustIntf {
				if targetUriPath == "/openconfig-test-xfmr:test-xfmr/interfaces/interface" && strings.HasSuffix(inParams.requestUri, "]") {
					ygot.BuildEmptyTree(ocIntfPtr)
				}
			}

			if nil != ocIntfPtr.Config {
				ocIntfPtr.Config.Id = ocIntfPtr.Id
			}
			if nil != ocIntfPtr.State {
				ocIntfPtr.State.Id = ocIntfPtr.Id
			}

			intfValPtr := reflect.ValueOf(ocIntfPtr)
			intfValElem := intfValPtr.Elem()

			testSets := intfValElem.FieldByName("IngressTestSets")
			if !testSets.IsNil() {
				testSet := testSets.Elem().FieldByName("IngressTestSet")
				if testSet.IsNil() || (!testSet.IsNil() && testSet.Len() == 0) {
					log.Infof("Get all Ingress Test Sets for %s", ifName)

					// Check if any Test Set is applied
					for testSet, testSetData := range testSetTblMap {
						trustTestSet = true
						ports := testSetData.GetList(TEST_SET_PORTS)
						if contains(ports, ifName) {
							testSetType := findInMap(TEST_SET_TYPE_MAP, testSetData.Get(TEST_SET_TYPE))
							n, _ := strconv.ParseInt(testSetType, 10, 64)
							testSetTypeDbkeyComp := "TEST_SET_IPV4"
							if n == int64(ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV6) {
								testSetTypeDbkeyComp = "TEST_SET_IPV6"
							}
							testSetName := getTestSetNameCompFromDbKey(testSet, testSetTypeDbkeyComp)
							log.Infof("Port:%v TestSetName:%v TestSetype:%v ", ifName, testSetName, testSetTypeDbkeyComp)
							testSetOrigType := convertSonicTestSetTypeToOC(testSetData.Get(TEST_SET_TYPE))
							testSet := testSets.MethodByName("NewIngressTestSet").Call([]reflect.Value{reflect.ValueOf(testSetName), reflect.ValueOf(testSetOrigType)})
							ygot.BuildEmptyTree(testSet[0].Interface().(ygot.ValidatedGoStruct))
						}
					}

				} else {
					log.Info("Get for specific Test Set")
				}

				testSetMap := testSets.Elem().FieldByName("IngressTestSet")
				testSetMapIter := testSetMap.MapRange()
				for testSetMapIter.Next() {
					testSetKey := testSetMapIter.Key()
					testSetPtr := testSetMapIter.Value()

					if !trustTestSet {
						if targetUriPath == "/openconfig-test-xfmr:test-xfmr/interfaces/interface/ingress-test-sets/ingress-test-set" && strings.HasSuffix(inParams.requestUri, "]") {
							ygot.BuildEmptyTree(testSetPtr.Interface().(ygot.ValidatedGoStruct))
						}
					}

					testSetName := testSetKey.FieldByName("SetName")
					testSetType := testSetKey.FieldByName("Type")
					testSetKeyStr := getTestSetKeyStrFromOCKey(testSetName.String(), testSetType.Interface().(ocbinds.E_OpenconfigTestXfmr_TEST_SET_TYPE))
					testSetData, found := testSetTblMap[testSetKeyStr]
					if found && contains(testSetData.GetList(TEST_SET_PORTS), ifName) {
						testSetCfg := testSetPtr.Elem().FieldByName("Config")
						if !testSetCfg.IsNil() {
							testSetCfg.Elem().FieldByName("SetName").Set(testSetPtr.Elem().FieldByName("SetName"))
							testSetCfg.Elem().FieldByName("Type").Set(testSetPtr.Elem().FieldByName("Type"))
						}
						testSetState := testSetPtr.Elem().FieldByName("State")
						if !testSetState.IsNil() {
							testSetState.Elem().FieldByName("SetName").Set(testSetPtr.Elem().FieldByName("SetName"))
							testSetState.Elem().FieldByName("Type").Set(testSetPtr.Elem().FieldByName("Type"))
						}

					}
				}
			}
		}

	}
	return err
}

var Subscribe_test_port_bindings_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var err error
	var result XfmrSubscOutParams

	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.Info("Subscribe_test_port_bindings_xfmr targetUriPath:", targetUriPath)
	result.isVirtualTbl = true
	log.Info("Returning Subscribe_test_port_bindings_xfmr")
	return result, err
}

func convertSonicTestSetTypeToOC(testSetType string) ocbinds.E_OpenconfigTestXfmr_TEST_SET_TYPE {
	var testSetOrigType ocbinds.E_OpenconfigTestXfmr_TEST_SET_TYPE

	if "IPV4" == testSetType {
		testSetOrigType = ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV4
	} else if "IPV6" == testSetType {
		testSetOrigType = ocbinds.OpenconfigTestXfmr_TEST_SET_TYPE_TEST_SET_IPV6
	} else {
		log.Infof("Unknown type %v", testSetType)
	}

	return testSetOrigType
}

// Sonic yang key transformer functions
var DbToYang_test_sensor_mode_key_xfmr SonicKeyXfmrDbToYang = func(inParams SonicXfmrParams) (map[string]interface{}, error) {
	res_map := make(map[string]interface{})
	/* from DB-key string(inParams.key) extract mode and id to fill into the res_map
	* db key contains the separator as well eg: "mode:test123:3545"
	 */
	log.Info("DbToYang_test_sensor_mode_key_xfmr: key", inParams.key)
	if len(inParams.key) > 0 {
		/*split id and mode */
		temp := strings.SplitN(inParams.key, ":", 3)
		if len(temp) >= 3 {
			res_map["mode"] = temp[0] + ":" + temp[1]
			id := temp[2]
			i64, _ := strconv.ParseUint(id, 10, 32)
			i32 := uint32(i64)
			res_map["id"] = i32
		} else if len(temp) == 2 {
			res_map["mode"] = temp[0]
			i64, _ := strconv.ParseUint(temp[1], 10, 32)
			i32 := uint32(i64)
			res_map["id"] = i32
		} else {
			errStr := "Invalid Key in uri."
			return res_map, tlerr.InvalidArgsError{Format: errStr}
		}
	}
	log.Info("DbToYang_test_sensor_mode_key_xfmr: res_map - ", res_map)
	return res_map, nil
}

var YangToDb_sensor_a_light_sensor_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var light_sensor_key string
	var err error

	log.Info("YangToDb_sensor_a_light_sensor_key_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	groupId := pathInfo.Var("id")
	sensorType := pathInfo.Var("type")
	lightSensorTag := pathInfo.Var("tag")
	if groupId == "" || sensorType == "" {
		err_str := "Invalid key. Key not supported."
		err = tlerr.NotSupported(err_str)
		return light_sensor_key, err
	}
	sensor_type := ""
	if strings.HasPrefix(sensorType, "sensora_") {
		sensor_type = strings.Replace(sensorType, "sensora_", "sensor_type_a_", 1)
		light_sensor_key = groupId + "|" + sensor_type
	} else if strings.HasPrefix(sensorType, "sensorb_") {
		err_str := "light sensor not supported for sensor type b."
		err = tlerr.NotSupported(err_str)
	} else {
		err_str := "Invalid key. Key not supported."
		err = tlerr.NotSupported(err_str)
	}
	if err == nil && lightSensorTag != "" {
		sensor_tag := strings.Replace(lightSensorTag, "lightsensor_", "light_sensor_", 1)
		light_sensor_key += "|" + sensor_tag
	}
	log.Info("YangToDb_sensor_a_light_sensor_key_xfmr returns", light_sensor_key)
	return light_sensor_key, err
}

var DbToYang_sensor_a_light_sensor_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {

	rmap := make(map[string]interface{})
	var err error
	if log.V(3) {
		log.Info("Entering DbToYang_sensor_a_light_sensor_key_xfmr inParams.uri ", inParams.uri)
	}
	var lightSensor string

	if strings.Contains(inParams.key, "|") {
		key_split := strings.Split(inParams.key, "|")
		if len(key_split) == 3 {
			lightSensor = key_split[2]
		}
		lightSensor = strings.Replace(lightSensor, "light_sensor_", "lightsensor_", 1)
	}

	rmap["tag"] = lightSensor

	log.Info("DbToYang_sensor_a_light_sensor_key_xfmr rmap ", rmap)
	return rmap, err
}

func light_sensor_validate(inParams XfmrParams) bool {
	var traversal_valid bool
	pathInfo := NewPathInfo(inParams.uri)
	sensorType := pathInfo.Var("type")
	if strings.HasPrefix(sensorType, "sensora_") {
		traversal_valid = true
	}
	log.Info("light_sensor_validate returning ", traversal_valid)
	return traversal_valid
}

var YangToDb_test_ni_instance_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var db_ni_name string
	var err error
	log.Info("YangToDb_test_ni_instance_key_xfmr ", inParams.uri)
	pathInfo := NewPathInfo(inParams.uri)
	ni_name := pathInfo.Var("ni-name")

	if strings.HasPrefix(ni_name, "vrf-") {
		db_ni_name = strings.Replace(ni_name, "vrf-", "Vrf_", 1)
	} else if strings.HasPrefix(ni_name, "default") {
		db_ni_name = ni_name
	} else if ni_name != "" {
		err_str := "Invalid key. Key not supported."
		err = tlerr.NotSupported(err_str)
	}
	log.Info("YangToDb_test_ni_instance_key_xfmr returning db_ni_name ", db_ni_name, " error ", err)
	return db_ni_name, err
}

var DbToYang_test_ni_instance_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.Info("DbToYang_test_ni_instance_key_xfmr ", inParams.uri)
	var rmap map[string]interface{}
	var ni_name string

	if inParams.key == "default" {
		ni_name = "default"
	} else {
		ni_name = strings.Replace(inParams.key, "Vrf_", "vrf-", 1)
	}
	rmap = make(map[string]interface{})
	rmap["ni-name"] = ni_name

	log.Info("DbToYang_test_ni_instance_key_xfmr returning ", rmap)
	return rmap, nil
}

var test_ni_instance_protocol_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string
	var err error

	log.Info("test_ni_instance_protocol_table_xfmr", inParams.uri)
	if inParams.oper == GET || inParams.oper == DELETE {
		pathInfo := NewPathInfo(inParams.uri)
		ni_name := pathInfo.Var("ni-name")
		proto_name := pathInfo.Var("name")
		log.Info("test_ni_instance_protocol_table_xfmr ni-inatnce ", ni_name, " proto name ", proto_name)
		cfg_tbl_updated := false
		if inParams.dbDataMap != nil {
			log.Info("test_ni_instance_protocol_table_xfmr  tblList dbDataMap ", (*inParams.dbDataMap)[db.ConfigDB])
			if (ni_name == "default") || (strings.HasPrefix(ni_name, "vrf-")) {
				if proto_name == "" { // inParams.uri at whole list level, hence add all child instances to be traversed
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"] = make(map[string]db.Value)
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["bgp"] = db.Value{Field: make(map[string]string)}
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["bgp"].Field["NULL"] = "NULL"
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["ospfv2"] = db.Value{Field: make(map[string]string)}
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["ospfv2"].Field["NULL"] = "NULL"
					cfg_tbl_updated = true
					log.Info("test_ni_instance_protocol_table_xfmr returning (*inParams.dbDataMap)[db.ConfigDB] ", (*inParams.dbDataMap)[db.ConfigDB])
				} else if proto_name == "bgp" {
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"] = make(map[string]db.Value)
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["bgp"] = db.Value{Field: make(map[string]string)}
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["bgp"].Field["NULL"] = "NULL"
					cfg_tbl_updated = true
					log.Info("test_ni_instance_protocol_table_xfmr returning (*inParams.dbDataMap)[db.ConfigDB] ", (*inParams.dbDataMap)[db.ConfigDB])
				} else if proto_name == "ospfv2" {
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"] = make(map[string]db.Value)
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["ospfv2"] = db.Value{Field: make(map[string]string)}
					(*inParams.dbDataMap)[db.ConfigDB]["TEST_CFG_PROTO_TBL"]["ospfv2"].Field["NULL"] = "NULL"
					cfg_tbl_updated = true
					log.Info("test_ni_instance_protocol_table_xfmr returning (*inParams.dbDataMap)[db.ConfigDB] ", (*inParams.dbDataMap)[db.ConfigDB])
				} else {
					err_str := "Invalid protocol key. Key not supported."
					err = tlerr.NotSupported(err_str)
				}
			} else {
				err_str := "Invalid ni-instance key. Key not supported."
				err = tlerr.NotSupported(err_str)
			}
		}
		if cfg_tbl_updated {
			tblList = append(tblList, "TEST_CFG_PROTO_TBL")
		}
	}
	*inParams.isVirtualTbl = true
	log.Info("test_ni_instance_protocol_table_xfmr returning tblList ", tblList, " error ", err)
	return tblList, err
}

var YangToDb_test_ni_instance_protocol_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var key string
	var err error

	if inParams.oper == GET || inParams.oper == DELETE {
		pathInfo := NewPathInfo(inParams.uri)
		ni_name := pathInfo.Var("ni-name")
		proto_name := pathInfo.Var("name")
		log.Info("YangToDb_test_ni_instance_protocol_key_xfmr received ni-instance ", ni_name, " protocol name ", proto_name)
		if (ni_name == "default") || (strings.HasPrefix(ni_name, "vrf-")) {
			if proto_name == "bgp" {
				key = "bgp"
			} else if proto_name == "ospfv2" {
				key = "ospfv2"
			} else if proto_name != "" {
				err_str := "Invalid protocol key. Key not supported."
				err = tlerr.NotSupported(err_str)
			}
		} else {
			err_str := "Invalid ni-instance key. Key not supported."
			err = tlerr.NotSupported(err_str)
		}
	}
	log.Info("YangToDb_test_ni_instance_protocol_key_xfmr returning key ", key, " error ", err)
	return key, err
}

var DbToYang_test_ni_instance_protocol_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	rmap["name"] = inParams.key
	log.Info("DbToYang_test_ni_instance_protocol_key_xfmr returning ", rmap)
	return rmap, nil
}

func checkNwInstanceProtocol(inParams XfmrParams, protoNm string) bool {
	pathInfo := NewPathInfo(inParams.uri)
	proto_name := pathInfo.Var("name")
	log.Info("checkNwInstanceProtocol() through validate handler ", proto_name)
	return proto_name == protoNm
}

func validate_bgp_proto(inParams XfmrParams) bool {
	return checkNwInstanceProtocol(inParams, "bgp")
}

func validate_ospfv2_proto(inParams XfmrParams) bool {
	return checkNwInstanceProtocol(inParams, "ospfv2")
}

var YangToDb_test_ospfv2_router_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var ospfv2_db_key string
	var err error
	log.Info("YangToDb_test_ospfv2_router_key_xfmr ", inParams.uri)
	pathInfo := NewPathInfo(inParams.uri)
	ni_name := pathInfo.Var("ni-name")

	if strings.HasPrefix(ni_name, "vrf-") {
		ospfv2_db_key = strings.Replace(ni_name, "vrf-", "Vrf_", 1)
	} else if strings.HasPrefix(ni_name, "default") {
		ospfv2_db_key = ni_name
	} else if ni_name != "" {
		err_str := "Invalid network-instance key. Key not supported."
		err = tlerr.NotSupported(err_str)
	}
	log.Info("YangToDb_test_ospfv2_router_key_xfmr returning ", ospfv2_db_key, " error ", err)
	return ospfv2_db_key, err
}

var YangToDb_test_ospfv2_router_distribution_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var db_ni_name, ospfv2_db_key string
	var err error
	log.Info("YangToDb_test_ospfv2_router_distribution_key_xfmr ", inParams.uri)
	pathInfo := NewPathInfo(inParams.uri)
	ni_name := pathInfo.Var("ni-name")
	distribution_id := pathInfo.Var("distribution-id")

	if strings.HasPrefix(ni_name, "vrf-") {
		db_ni_name = strings.Replace(ni_name, "vrf-", "Vrf_", 1)
	} else if strings.HasPrefix(ni_name, "default") {
		db_ni_name = ni_name
	} else if ni_name != "" {
		err_str := "Invalid key. Key not supported."
		err = tlerr.NotSupported(err_str)
	}

	if distribution_id != "" {
		ospfv2_db_key = db_ni_name + "|" + distribution_id
	} else {
		ospfv2_db_key = db_ni_name
	}

	log.Info("YangToDb_test_ospfv2_router_distribution_key_xfmr returning ", ospfv2_db_key, " error ", err)
	return ospfv2_db_key, err
}

var DbToYang_test_ospfv2_router_distribution_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.Info("DbToYang_test_ospfv2_router_distribution_key_xfmr ", inParams.uri)
	var rmap map[string]interface{}
	var distribution_id string

	if strings.Contains(inParams.key, "|") {
		key_split := strings.SplitN(inParams.key, "|", 2)
		if len(key_split) == 2 {
			distribution_id = key_split[1]
			rmap = make(map[string]interface{})
			rmap["distribution-id"] = distribution_id
		}
	}
	log.Info("DbToYang_test_ospfv2_router_distribution_key_xfmr returning ", rmap)
	return rmap, nil
}

var YangToDb_test_bgp_network_cfg_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var db_ni_name, bgp_cfg_db_key string
	var err error
	log.Info("YangToDb_test_bgp_network_cfg_key_xfmr ", inParams.uri)
	pathInfo := NewPathInfo(inParams.uri)
	ni_name := pathInfo.Var("ni-name")
	network_id := pathInfo.Var("network-id")

	if strings.HasPrefix(ni_name, "vrf-") {
		db_ni_name = strings.Replace(ni_name, "vrf-", "Vrf_", 1)
	} else if strings.HasPrefix(ni_name, "default") {
		db_ni_name = ni_name
	} else if ni_name != "" {
		err_str := "Invalid key. Key not supported."
		err = tlerr.NotSupported(err_str)
	}

	if network_id != "" {
		bgp_cfg_db_key = db_ni_name + "|" + network_id
	} else {
		bgp_cfg_db_key = db_ni_name
	}

	log.Info("YangToDb_test_bgp_network_cfg_key_xfmr returning ", bgp_cfg_db_key, " error ", err)
	return bgp_cfg_db_key, err
}

var DbToYang_test_bgp_network_cfg_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.Info("DbToYang_test_bgp_network_cfg_key_xfmr ", inParams.uri)
	var rmap map[string]interface{}
	var network_id string

	if strings.Contains(inParams.key, "|") {
		key_split := strings.SplitN(inParams.key, "|", 2)
		if len(key_split) == 2 {
			network_id = key_split[1]
			rmap = make(map[string]interface{})
			rmap["network-id"] = network_id
		}
	}
	log.Info("DbToYang_test_bgp_network_cfg_key_xfmr returning ", rmap)
	return rmap, nil
}
