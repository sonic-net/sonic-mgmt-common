//////////////////////////////////////////////////////////////////////////
//
// Copyright 2020 Dell, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//////////////////////////////////////////////////////////////////////////

package transformer

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	/* sFlow tables */
	SFLOW_GLOBAL_TBL = "SFLOW"
	SFLOW_COL_TBL    = "SFLOW_COLLECTOR"
	SFLOW_SESS_TBL   = "SFLOW_SESSION_TABLE" /* Session table in ApplDb */
	SFLOW_INTF_TBL   = "SFLOW_SESSION"       /* Session table in ConfigDb */

	/* sFlow keys */
	SFLOW_GLOBAL_KEY      = "global"
	SFLOW_ADMIN_KEY       = "admin_state"
	SFLOW_POLLING_INT_KEY = "polling_interval"
	SFLOW_SAMPL_RATE_KEY  = "sample_rate"
	SFLOW_AGENT_KEY       = "agent_id"
	SFLOW_INTF_NAME_KEY   = "name"
	SFLOW_COL_IP_KEY      = "collector_ip"
	SFLOW_COL_PORT_KEY    = "collector_port"
	SFLOW_COL_VRF_KEY     = "collector_vrf"

	/* sFlow default values */
	DEFAULT_POLLING_INT = 20
	DEFAULT_AGENT       = "default"
	DEFAULT_VRF_NAME    = "default"
	DEFAULT_COL_PORT    = "6343"

	/* sFlow URIs */
	SAMPLING                                    = "/openconfig-sampling-sflow:sampling"
	SAMPLING_SFLOW                              = "/openconfig-sampling-sflow:sampling/sflow"
	SAMPLING_SFLOW_CONFIG                       = "/openconfig-sampling-sflow:sampling/sflow/config"
	SAMPLING_SFLOW_CONFIG_ENABLED               = "/openconfig-sampling-sflow:sampling/sflow/config/enabled"
	SAMPLING_SFLOW_CONFIG_POLLING_INT           = "/openconfig-sampling-sflow:sampling/sflow/config/polling-interval"
	SAMPLING_SFLOW_CONFIG_AGENT                 = "/openconfig-sampling-sflow:sampling/sflow/config/agent"
	SAMPLING_SFLOW_STATE                        = "/openconfig-sampling-sflow:sampling/sflow/state"
	SAMPLING_SFLOW_STATE_ENABLED                = "/openconfig-sampling-sflow:sampling/sflow/state/enabled"
	SAMPLING_SFLOW_STATE_POLLING_INT            = "/openconfig-sampling-sflow:sampling/sflow/state/polling-interval"
	SAMPLING_SFLOW_STATE_AGENT                  = "/openconfig-sampling-sflow:sampling/sflow/state/agent"
	SAMPLING_SFLOW_COLS                         = "/openconfig-sampling-sflow:sampling/sflow/collectors"
	SAMPLING_SFLOW_COLS_COL                     = "/openconfig-sampling-sflow:sampling/sflow/collectors/collector"
	SAMPLING_SFLOW_COLS_COL_CONFIG              = "/openconfig-sampling-sflow:sampling/sflow/collectors/collector/config"
	SAMPLING_SFLOW_COLS_COL_STATE               = "/openconfig-sampling-sflow:sampling/sflow/collectors/collector/state"
	SAMPLING_SFLOW_INTFS                        = "/openconfig-sampling-sflow:sampling/sflow/interfaces"
	SAMPLING_SFLOW_INTFS_INTF                   = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface"
	SAMPLING_SFLOW_INTFS_INTF_CONFIG            = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface/config"
	SAMPLING_SFLOW_INTFS_INTF_CONFIG_SAMPL_RATE = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface/config/sampling-rate"
	SAMPLING_SFLOW_INTFS_INTF_CONFIG_ENABLED    = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface/config/enabled"
	SAMPLING_SFLOW_INTFS_INTF_STATE             = "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface/state"

	/* IPv4/v6 localhost address */
	IPV4_LOCALHOST = "127.0.0.1"
	IPV6_LOCALHOST = "::1"
)

type Sflow struct {
	Enabled          string
	Polling_Interval string
	Agent            string
}

type SflowCol struct {
	Ip   string
	Port string
	Vrf  string
}

type SflowIntf struct {
	Enabled       string
	Sampling_Rate string
}

func init() {
	XlateFuncBind("DbToYang_sflow_xfmr", DbToYang_sflow_xfmr)
	XlateFuncBind("YangToDb_sflow_xfmr", YangToDb_sflow_xfmr)
	XlateFuncBind("Subscribe_sflow_xfmr", Subscribe_sflow_xfmr)
	XlateFuncBind("DbToYang_sflow_collector_xfmr", DbToYang_sflow_collector_xfmr)
	XlateFuncBind("YangToDb_sflow_collector_xfmr", YangToDb_sflow_collector_xfmr)
	XlateFuncBind("Subscribe_sflow_collector_xfmr", Subscribe_sflow_collector_xfmr)
	XlateFuncBind("DbToYangPath_sflow_collector_path_xfmr", DbToYangPath_sflow_collector_path_xfmr)
	XlateFuncBind("DbToYang_sflow_interface_xfmr", DbToYang_sflow_interface_xfmr)
	XlateFuncBind("YangToDb_sflow_interface_xfmr", YangToDb_sflow_interface_xfmr)
	XlateFuncBind("Subscribe_sflow_interface_xfmr", Subscribe_sflow_interface_xfmr)
	XlateFuncBind("DbToYangPath_sflow_interface_path_xfmr", DbToYangPath_sflow_interface_path_xfmr)
}

var DbToYangPath_sflow_collector_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	log.V(3).Info("DbToYangPath_sflow_collector fmr: tbl:", params.tblName)
	sflowCollRoot := "/openconfig-sampling-sflow:sampling/sflow/collectors/collector"

	if params.tblName != SFLOW_COL_TBL {
		oper_err := errors.New("wrong config DB table sent")
		log.Errorf("Sflow collector Path-xfmr: table name %s not in sflow view", params.tblName)
		return oper_err
	} else {
		if len(params.tblKeyComp) > 0 {
			key_parts := strings.Split(params.tblKeyComp[0], "_")
			if len(key_parts) != 3 {
				oper_err := errors.New("Invalid key " + params.tblKeyComp[0])
				log.Errorf("sflow_collector_path_xfmr: Invalid Key  %s", params.tblKeyComp[0])
				return oper_err
			}
			params.ygPathKeys[sflowCollRoot+"/address"] = key_parts[0]
			params.ygPathKeys[sflowCollRoot+"/port"] = key_parts[1]
			params.ygPathKeys[sflowCollRoot+"/network-instance"] = key_parts[2]
		} else {
			oper_err := errors.New("Missing DB key.")
			log.Errorf("Missing DB Key")
			return oper_err
		}
	}
	log.V(3).Info("DbToYangPath sflow_collector  ygPathKeys: ", params.ygPathKeys)

	return nil
}

var DbToYangPath_sflow_interface_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	log.V(3).Info("DbToYangPath_sflow_interface fmr: tbl:", params.tblName)
	sflowCollRoot := "/openconfig-sampling-sflow:sampling/sflow/interfaces/interface"

	if params.tblName != SFLOW_SESS_TBL {
		oper_err := errors.New("wrong config DB table sent")
		log.Errorf("Sflow interface Path-xfmr: table name %s not in sflow view", params.tblName)
		return oper_err
	} else {
		if len(params.tblKeyComp) > 0 {
			params.ygPathKeys[sflowCollRoot+"/name"] = params.tblKeyComp[0]
		} else {
			oper_err := errors.New("Missing DB key.")
			log.Errorf("Missing DB Key")
			return oper_err
		}
	}
	log.V(3).Info("DbToYangPath sflow_interface  ygPathKeys: ", params.ygPathKeys)

	return nil
}

func getSflowRootObject(s *ygot.GoStruct) *ocbinds.OpenconfigSamplingSflow_Sampling {
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.Sampling
}

var Subscribe_sflow_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var err error
	var result XfmrSubscOutParams
	if inParams.subscProc != TRANSLATE_SUBSCRIBE {
		result.isVirtualTbl = true
		log.V(3).Info("Subscribe_sflow_xfmr :- result.  isVirtualTbl: ", result.isVirtualTbl)
		return result, nil
	}
	result.dbDataMap = make(RedisDbSubscribeMap)

	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.uri)

	log.V(3).Infof("Subscribe_sflow_xfmr: targetUri %v ", targetUriPath)

	if targetUriPath == SAMPLING_SFLOW || targetUriPath == SAMPLING_SFLOW_CONFIG ||
		targetUriPath == SAMPLING_SFLOW_STATE {
		result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {SFLOW_GLOBAL_TBL: {
			SFLOW_GLOBAL_KEY: {"admin_state": "enabled", "polling_interval": "polling-interval", "agent_id": "agent"}}}}
		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{}
		result.nOpts.pType = OnChange
		result.isVirtualTbl = false
	}
	return result, err
}

var DbToYang_sflow_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	log.V(3).Infof("Received GET for sFlow path: %s, vars: %v", pathInfo.Path, pathInfo.Vars)

	log.V(3).Info("inParams.Uri:", inParams.requestUri)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	return getSflow(getSflowRootObject(inParams.ygRoot), targetUriPath, inParams.uri, inParams.dbs[:])
}

var YangToDb_sflow_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	res_map := make(map[string]map[string]db.Value)

	log.V(3).Info("sFlow SubTreeXfmr: ", inParams.uri)
	global_map := make(map[string]db.Value)
	sflowObj := getSflowRootObject(inParams.ygRoot)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)

	global_map[SFLOW_GLOBAL_KEY] = db.Value{Field: make(map[string]string)}

	if inParams.oper == DELETE {
		switch targetUriPath {
		case SAMPLING_SFLOW_CONFIG_AGENT:
			global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_AGENT_KEY] = ""
		case SAMPLING_SFLOW_CONFIG_POLLING_INT:
			global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_POLLING_INT_KEY] = ""
		default:
			return res_map, errors.New("DELETE not supported on attribute")
		}

		res_map[SFLOW_GLOBAL_TBL] = global_map
		return res_map, err
	}

	if sflowObj.Sflow.Config.Enabled != nil {
		if *(sflowObj.Sflow.Config.Enabled) {
			global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_ADMIN_KEY] = "up"
		} else {
			global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_ADMIN_KEY] = "down"
		}
	}

	if sflowObj.Sflow.Config.PollingInterval != nil {
		global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_POLLING_INT_KEY] =
			strconv.FormatUint(uint64(*(sflowObj.Sflow.Config.PollingInterval)), 10)
	}

	if sflowObj.Sflow.Config.Agent != nil {
		global_map[SFLOW_GLOBAL_KEY].Field[SFLOW_AGENT_KEY] = *(sflowObj.Sflow.Config.Agent)
	}

	res_map[SFLOW_GLOBAL_TBL] = global_map
	return res_map, err
}

var DbToYang_sflow_collector_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	log.V(3).Infof("Received GET for sFlow Collector path: %s, vars: %v", pathInfo.Path, pathInfo.Vars)
	log.V(3).Info("inParams.Uri:", inParams.requestUri)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	return getSflowCol(getSflowRootObject(inParams.ygRoot), targetUriPath, inParams.uri, inParams.dbs[db.ConfigDB])
}

func makeColKey(uri string) string {
	ip := NewPathInfo(uri).Var("address")
	port := NewPathInfo(uri).Var("port")
	vrf := NewPathInfo(uri).Var("network-instance")
	name := ""
	if ip != "" {
		name = ip + "_" + port + "_" + vrf
	}
	return name
}

var YangToDb_sflow_collector_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	res_map := make(map[string]map[string]db.Value)

	log.V(3).Info("sFlow Collector YangToDBSubTreeXfmr: ", inParams.uri)
	col_map := make(map[string]db.Value)
	sflowObj := getSflowRootObject(inParams.ygRoot)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)

	key := makeColKey(inParams.uri)
	if inParams.oper == DELETE {
		if strings.HasPrefix(targetUriPath, SAMPLING_SFLOW_COLS_COL_CONFIG) {
			return res_map, errors.New("Delete operation not supported for this xpath")
		}

		if key != "" {
			col_map[key] = db.Value{Field: make(map[string]string)}
		}
		res_map[SFLOW_COL_TBL] = col_map
		return res_map, err
	}

	if key != "" {
		ip := NewPathInfo(inParams.uri).Var("address")
		port := NewPathInfo(inParams.uri).Var("port")
		vrf := NewPathInfo(inParams.uri).Var("network-instance")
		col_map[key] = db.Value{Field: make(map[string]string)}
		col_map[key].Field[SFLOW_COL_IP_KEY] = ip
		col_map[key].Field[SFLOW_COL_PORT_KEY] = port
		col_map[key].Field[SFLOW_COL_VRF_KEY] = vrf
	} else {
		for col := range sflowObj.Sflow.Collectors.Collector {
			port := strconv.FormatUint(uint64(col.Port), 10)
			key = col.Address + "_" + port + "_" + col.NetworkInstance
			col_map[key] = db.Value{Field: make(map[string]string)}
			col_map[key].Field[SFLOW_COL_IP_KEY] = col.Address
			col_map[key].Field[SFLOW_COL_PORT_KEY] = port
			col_map[key].Field[SFLOW_COL_VRF_KEY] = col.NetworkInstance
		}
	}

	res_map[SFLOW_COL_TBL] = col_map
	return res_map, err
}

var Subscribe_sflow_collector_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {

	var err error
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)

	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.uri)

	log.V(3).Infof("Subscribe_sflow_collector_xfmr: pathInfo %v targetUri %v ", pathInfo, targetUriPath)

	ip := pathInfo.Var("address")
	if ip == "" {
		ip = "*"
	}

	port := pathInfo.Var("port")
	if port == "" {
		port = "*"
	}
	vrf := pathInfo.Var("network-instance")
	if vrf == "" {
		vrf = "*"
	}
	var name string
	if ip == "*" && port == "*" && vrf == "*" {
		name = "*"
	} else {
		name = ip + "_" + port + "_" + vrf
	}
	log.V(3).Infof("Subscribe_sflow_collector_xfmr: key %s", name)
	result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {SFLOW_COL_TBL: {name: {
		"collector_ip": "address", "collector_port": "port", "collector_vrf": "network-instance"}}}}

	return result, err
}

var DbToYang_sflow_interface_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	log.V(3).Infof("Received GET for sFlow Interface path: %s, vars: %v", pathInfo.Path, pathInfo.Vars)
	log.V(3).Info("inParams.Uri:", inParams.requestUri)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	return getSflowIntf(getSflowRootObject(inParams.ygRoot), targetUriPath, inParams.uri, inParams.dbs[db.ApplDB])
}

var Subscribe_sflow_interface_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var err error
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	key := NewPathInfo(inParams.uri).Var("name")

	if key == "" {
		key = "*"
	}
	log.V(3).Infof("XfmrSubscribe_sflow_interface_xfmr key %s ", key)
	result.dbDataMap = RedisDbSubscribeMap{db.ApplDB: {SFLOW_SESS_TBL: {key: {
		"ifname": "name", "admin_state": "enabled", "sample_rate": "sampling-rate"}}}}

	return result, err
}

var YangToDb_sflow_interface_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	res_map := make(map[string]map[string]db.Value)

	log.V(3).Info("sFlow Interface YangToDBSubTreeXfmr: ", inParams.uri)
	intf_map := make(map[string]db.Value)
	sflowObj := getSflowRootObject(inParams.ygRoot)
	targetUriPath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	log.V(3).Infof("YangToDb_sflow_interface_xfmr: targetUri %v ", targetUriPath)

	if inParams.oper == DELETE {
		if !strings.Contains(targetUriPath, SAMPLING_SFLOW_INTFS_INTF) {
			return res_map, tlerr.NotSupportedError{Format: "DELETE not supported", Path: targetUriPath}
		}

		name := NewPathInfo(inParams.uri).Var("name")
		if name == "" {
			return res_map, tlerr.InvalidArgs("Missing interface name")
		}

		intf_map[name] = db.Value{Field: make(map[string]string)}
		switch targetUriPath {
		case SAMPLING_SFLOW_INTFS_INTF_CONFIG_SAMPL_RATE:
			intf_map[name].Field[SFLOW_SAMPL_RATE_KEY] = ""
		case SAMPLING_SFLOW_INTFS_INTF_CONFIG_ENABLED:
			intf_map[name].Field[SFLOW_ADMIN_KEY] = ""
		case SAMPLING_SFLOW_INTFS_INTF:
			/* Delete all interface configurations */
		default:
			return res_map, errors.New("DELETE not supported on attribute or container")
		}
	} else {
		for _, intf := range sflowObj.Sflow.Interfaces.Interface {

			if intf.Name == nil {
				return res_map, errors.New("sFlow Interface: No interface name")
			}

			if intf.Config == nil {
				log.V(3).Infof("sFlow Inteface: No configuration")
				continue
			}

			name := *(intf.Name)
			intf_map[name] = db.Value{Field: make(map[string]string)}

			if intf.Config.Enabled != nil {
				if *(intf.Config.Enabled) {
					intf_map[name].Field[SFLOW_ADMIN_KEY] = "up"
				} else {
					intf_map[name].Field[SFLOW_ADMIN_KEY] = "down"
				}
			}

			if intf.Config.SamplingRate != nil {
				intf_map[name].Field[SFLOW_SAMPL_RATE_KEY] =
					strconv.FormatUint(uint64(*(intf.Config.SamplingRate)), 10)
			}
		}
	}

	res_map[SFLOW_INTF_TBL] = intf_map
	return res_map, err
}

func getSflowInfoFromDb(d *db.DB) (Sflow, error) {
	var sfInfo Sflow
	var err error

	sflowEntry, err := d.GetEntry(&db.TableSpec{Name: SFLOW_GLOBAL_TBL}, db.Key{Comp: []string{SFLOW_GLOBAL_KEY}})
	if err != nil {
		return sfInfo, err
	}

	sfInfo.Enabled = sflowEntry.Get(SFLOW_ADMIN_KEY)
	sfInfo.Polling_Interval = sflowEntry.Get(SFLOW_POLLING_INT_KEY)
	sfInfo.Agent = sflowEntry.Get(SFLOW_AGENT_KEY)

	return sfInfo, err
}

func fillSflowInfo(sflow *ocbinds.OpenconfigSamplingSflow_Sampling_Sflow,
	targetUriPath string, d *db.DB) error {
	var err error
	enabled := false
	sfInfo, err := getSflowInfoFromDb(d)

	if err != nil {
		if !strings.Contains(err.Error(), "Entry does not exist") {
			log.V(3).Info("Cant get entry: ", SFLOW_GLOBAL_TBL)
			return err
		}
		err = nil
		log.V(3).Info("sFlow not enabled")
	}

	config := sflow.Config
	state := sflow.State

	state.Enabled = &enabled
	config.Enabled = &enabled

	if sfInfo.Enabled != "" {
		enabled = sfInfo.Enabled == "up"
	}

	if sfInfo.Polling_Interval != "" {
		tmp, _ := strconv.ParseUint(sfInfo.Polling_Interval, 10, 16)
		pollingInt := uint16(tmp)
		state.PollingInterval = &pollingInt
		config.PollingInterval = state.PollingInterval
	}

	if sfInfo.Agent != "" {
		state.Agent = &sfInfo.Agent
		config.Agent = state.Agent
	}

	return err
}

func getSflow(sflow_tr *ocbinds.OpenconfigSamplingSflow_Sampling, targetUriPath string,
	uri string, d []*db.DB) error {
	log.V(3).Infof("Getting sFlow information")
	var err error

	ygot.BuildEmptyTree(sflow_tr)
	ygot.BuildEmptyTree(sflow_tr.Sflow)
	ygot.BuildEmptyTree(sflow_tr.Sflow.Config)
	ygot.BuildEmptyTree(sflow_tr.Sflow.State)

	switch targetUriPath {
	case SAMPLING_SFLOW:
		ygot.BuildEmptyTree(sflow_tr.Sflow.Collectors)
		ygot.BuildEmptyTree(sflow_tr.Sflow.Interfaces)
		err = fillSflowCollectorInfo(sflow_tr.Sflow.Collectors, "", targetUriPath, d[db.ConfigDB])
		if err != nil {
			return err
		}
		err = fillSflowInterfaceInfo(sflow_tr.Sflow.Interfaces, "", targetUriPath, d[db.ApplDB])
		if err != nil {
			return err
		}
		fallthrough
	default:
		err = fillSflowInfo(sflow_tr.Sflow, targetUriPath, d[db.ConfigDB])
	}
	return err
}

func getSflowColInfoFromDb(d *db.DB) (map[string]SflowCol, error) {
	var sfInfo map[string]SflowCol
	var col SflowCol
	var err error

	sflowColTbl, err := d.GetTable(&db.TableSpec{Name: SFLOW_COL_TBL})

	if err != nil {
		return sfInfo, err
	}

	keys, err := sflowColTbl.GetKeys()
	if err != nil {
		log.V(3).Info("No collectors configured")
		return sfInfo, nil
	}

	sfInfo = make(map[string]SflowCol)
	for _, key := range keys {
		name := key.Get(0)
		colEntry, err := sflowColTbl.GetEntry(key)
		if err != nil {
			log.Errorf("Can't get entry with key: ", name)
			return sfInfo, err
		}
		col.Ip = colEntry.Get(SFLOW_COL_IP_KEY)
		col.Port = colEntry.Get(SFLOW_COL_PORT_KEY)
		col.Vrf = colEntry.Get(SFLOW_COL_VRF_KEY)
		sfInfo[name] = col
	}

	return sfInfo, err
}

func appendColToYang(sflowCols *ocbinds.OpenconfigSamplingSflow_Sampling_Sflow_Collectors,
	ip string, port uint16, vrf string) error {
	var err error
	colKey := ocbinds.OpenconfigSamplingSflow_Sampling_Sflow_Collectors_Collector_Key{ip, port, vrf}

	sfc, found := sflowCols.Collector[colKey]
	if !found {
		sfc, err = sflowCols.NewCollector(ip, port, vrf)
		if err != nil {
			log.Errorf("Error creating Collector component")
			return err
		}
	}

	ygot.BuildEmptyTree(sfc)
	ygot.BuildEmptyTree(sfc.Config)
	ygot.BuildEmptyTree(sfc.State)
	sfc.Config.Address = &ip
	sfc.Config.Port = &port
	sfc.Config.NetworkInstance = &vrf

	sfc.State.Address = &ip
	sfc.State.Port = &port
	sfc.State.NetworkInstance = &vrf

	return err
}

func fillSflowCollectorInfo(sflowCols *ocbinds.OpenconfigSamplingSflow_Sampling_Sflow_Collectors,
	name string, targetUriPath string, d *db.DB) error {
	var err error
	var port uint16

	sfInfo, err := getSflowColInfoFromDb(d)
	if err != nil {
		return err
	}

	if name == "" {
		for _, v := range sfInfo {
			if v.Ip == "" {
				log.Errorf("No collector IP")
				break
			}
			if v.Port == "" {
				v.Port = DEFAULT_COL_PORT
			}
			if v.Vrf == "" {
				v.Vrf = DEFAULT_VRF_NAME
			}
			tmp, _ := strconv.ParseUint(v.Port, 10, 16)
			port = uint16(tmp)
			err = appendColToYang(sflowCols, v.Ip, port, v.Vrf)
		}
		return err
	}

	if v, ok := sfInfo[name]; ok {
		if v.Ip == "" {
			log.Errorf("No collector IP")
			return err
		}
		if v.Port == "" {
			v.Port = DEFAULT_COL_PORT
		}
		if v.Vrf == "" {
			v.Vrf = DEFAULT_VRF_NAME
		}
		tmp, _ := strconv.ParseUint(v.Port, 10, 16)
		port = uint16(tmp)
		err = appendColToYang(sflowCols, v.Ip, port, v.Vrf)
		return err
	}

	return errors.New("Collector entry not found")
}

func getSflowCol(sflow_tr *ocbinds.OpenconfigSamplingSflow_Sampling, targetUriPath string,
	uri string, d *db.DB) error {
	log.V(3).Infof("Getting sFlow collector information")
	ygot.BuildEmptyTree(sflow_tr.Sflow)
	ygot.BuildEmptyTree(sflow_tr.Sflow.Collectors)
	key := makeColKey(uri)
	return fillSflowCollectorInfo(sflow_tr.Sflow.Collectors, key, targetUriPath, d)
}

func getSflowIntfInfoFromDb(d *db.DB) (map[string]SflowIntf, error) {
	var sfInfo map[string]SflowIntf
	var intf SflowIntf
	var err error

	sflowIntfTbl, err := d.GetTable(&db.TableSpec{Name: SFLOW_SESS_TBL})

	if err != nil {
		return sfInfo, err
	}

	keys, err := sflowIntfTbl.GetKeys()
	if err != nil {
		log.V(3).Info("No interface configured, sFlow not enabled")
		return sfInfo, nil
	}

	sfInfo = make(map[string]SflowIntf)
	for _, key := range keys {
		name := key.Get(0)
		intfEntry, err := sflowIntfTbl.GetEntry(key)
		if err != nil {
			log.Errorf("Can't get entry with key: ", name)
			return sfInfo, err
		}
		intf.Enabled = intfEntry.Get(SFLOW_ADMIN_KEY)
		intf.Sampling_Rate = intfEntry.Get(SFLOW_SAMPL_RATE_KEY)
		sfInfo[name] = intf
	}

	return sfInfo, err
}

func fillSflowInterfaceInfo(sflowIntfs *ocbinds.OpenconfigSamplingSflow_Sampling_Sflow_Interfaces,
	name string, targetUriPath string, d *db.DB) error {
	var err error
	var enabled bool
	var samplingRate uint32

	sfInfo, err := getSflowIntfInfoFromDb(d)
	if err != nil {
		return err
	}

	if name == "" {
		for name, v := range sfInfo {
			tmp, _ := strconv.ParseUint(v.Sampling_Rate, 10, 32)
			samplingRate = uint32(tmp)
			enabled = v.Enabled == "up"
			err = appendIntfToYang(sflowIntfs, name, enabled, samplingRate)
			if err != nil {
				break
			}
		}
		return err
	}

	if v, ok := sfInfo[name]; ok {
		tmp, _ := strconv.ParseUint(v.Sampling_Rate, 10, 32)
		samplingRate = uint32(tmp)
		enabled = v.Enabled == "up"
		err = appendIntfToYang(sflowIntfs, name, enabled, samplingRate)
		return err
	}

	return errors.New("sFlow Interface entry not found")
}

func appendIntfToYang(sflowIntf *ocbinds.OpenconfigSamplingSflow_Sampling_Sflow_Interfaces,
	name string, enabled bool, samplingRate uint32) error {
	var err error
	sfc, found := sflowIntf.Interface[name]
	if !found {
		sfc, err = sflowIntf.NewInterface(name)
		if err != nil {
			log.Errorf("Error creating sFlow Interface")
			return err
		}
	}

	ygot.BuildEmptyTree(sfc)
	ygot.BuildEmptyTree(sfc.Config)
	ygot.BuildEmptyTree(sfc.State)

	sfc.Config.Enabled = &enabled
	sfc.Config.Name = &name
	sfc.Config.SamplingRate = &samplingRate

	sfc.State.Enabled = &enabled
	sfc.State.Name = &name
	sfc.State.SamplingRate = &samplingRate

	return err
}

func getSflowIntf(sflow_tr *ocbinds.OpenconfigSamplingSflow_Sampling, targetUriPath string,
	uri string, d *db.DB) error {
	log.V(3).Infof("Getting sFlow interface information")

	name := NewPathInfo(uri).Var("name")
	ygot.BuildEmptyTree(sflow_tr.Sflow)
	ygot.BuildEmptyTree(sflow_tr.Sflow.Interfaces)
	err := fillSflowInterfaceInfo(sflow_tr.Sflow.Interfaces, name, targetUriPath, d)
	return err
}
