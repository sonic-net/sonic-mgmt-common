//////////////////////////////////////////////////////////////////////////
//
// Copyright (c) 2024 Cisco Systems, Inc.
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
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

// Mapping for severity enumeration oc to sonic
var ocToSonic_severity = map[ocbinds.E_OpenconfigMessages_SyslogSeverity]string{
	ocbinds.OpenconfigMessages_SyslogSeverity_EMERGENCY:     "crit",
	ocbinds.OpenconfigMessages_SyslogSeverity_ALERT:         "crit",
	ocbinds.OpenconfigMessages_SyslogSeverity_CRITICAL:      "crit",
	ocbinds.OpenconfigMessages_SyslogSeverity_ERROR:         "error",
	ocbinds.OpenconfigMessages_SyslogSeverity_WARNING:       "warn",
	ocbinds.OpenconfigMessages_SyslogSeverity_NOTICE:        "notice",
	ocbinds.OpenconfigMessages_SyslogSeverity_INFORMATIONAL: "info",
	ocbinds.OpenconfigMessages_SyslogSeverity_DEBUG:         "debug",
}

// Mapping for severity enumeration oc string to sonic
var ocStrToSonic_severity = map[string]string{
	"EMERGENCY":     "crit",
	"ALERT":         "crit",
	"CRITICAL":      "crit",
	"ERROR":         "error",
	"WARNING":       "warn",
	"NOTICE":        "notice",
	"INFORMATIONAL": "info",
	"DEBUG":         "debug",
}

// Mapping for severity enumeration sonic to oc
var sonicToOc_severity = map[string]string{
	"crit":   "CRITICAL",
	"error":  "ERROR",
	"warn":   "WARNING",
	"notice": "NOTICE",
	"info":   "INFORMATIONAL",
	"debug":  "DEBUG",
	"none":   "DEBUG",
}

var invalid_input_err error = errors.New("Invalid input")
var not_implemented_err error = errors.New("Not implemented")
var invalid_db_err error = errors.New("DB not is proper state")
var aaa_failed_no_method_err error = errors.New("Given AAA methods not found. Valid options include: local, radius, ldap, default and tacacs+")

var intfTblList = []string{"INTERFACE", "LOOPBACK_INTERFACE", "PORTCHANNEL_INTERFACE"}

var aaa_sonicToOc_serverType = map[string]string{
	"RADIUS_SERVER":  "RADIUS",
	"TACPLUS_SERVER": "TACACS",
}

const (
	PATHZ_TBL = "PATHZ_TABLE"
	READS_GET = "get"
	READS_SUB = "subscribe"
	WRITES    = "set"
	GNXI_ID   = "gnxi"

	/** Credential Tables **/
	CREDENTIALS_TBL = "CREDENTIALS"
	CRED_PATHZ_TBL  = "CREDENTIALS|PATHZ_POLICY"
	CRED_AUTHZ_TBL  = "CREDENTIALS|AUTHZ_POLICY"
	CERT_TBL        = "CREDENTIALS|CERT"
	AUTHZ_TBL       = "AUTHZ_TABLE"
	ACCEPTS         = "permitted"
	REJECTS         = "denied"
	cntResult       = "cntResult"
	tsResult        = "tsResult"

	/** System Root paths **/
	SYSTEM_ROOT = "/openconfig-system:system"

	/** Pathz paths **/
	GRPC_OC_SERVERS = SYSTEM_ROOT + "/openconfig-system-grpc:grpc-servers"
	GRPC_SERVERS    = SYSTEM_ROOT + "/grpc-servers"
	GRPC_SERVER     = GRPC_OC_SERVERS + "/grpc-server"

	/** Authz paths **/
	AUTHZ_POLICY_COUNTERS   = GRPC_SERVER + "/authz-policy-counters"
	ALL_AUTHZ               = AUTHZ_POLICY_COUNTERS + "/rpcs"
	SINGLE_AUTHZ            = ALL_AUTHZ + "/rpc"
	AUTHZ_STATE             = SINGLE_AUTHZ + "/state"
	AUTHZ_SUCCESS           = AUTHZ_STATE + "/access-accepts"
	AUTHZ_SUCCESS_TIMESTAMP = AUTHZ_STATE + "/last-access-accept"
	AUTHZ_FAILED            = AUTHZ_STATE + "/access-rejects"
	AUTHZ_FAILED_TIMESTAMP  = AUTHZ_STATE + "/last-access-reject"
	PATHZ_POLICY_COUNTERS   = GRPC_SERVER + "/gnsi-pathz:gnmi-pathz-policy-counters"
	ALL_PATHZ               = PATHZ_POLICY_COUNTERS + "/paths"
	SINGLE_PATHZ            = ALL_PATHZ + "/path"

	PATHZ_STATE  = SINGLE_PATHZ + "/state"
	PATHZ_READS  = PATHZ_STATE + "/reads"
	PATHZ_WRITES = PATHZ_STATE + "/writes"

	PATHZ_READ_SUCCESS            = PATHZ_READS + "/access-accepts"
	PATHZ_READ_SUCCESS_TIMESTAMP  = PATHZ_READS + "/last-access-accept"
	PATHZ_READ_FAILED             = PATHZ_READS + "/access-rejects"
	PATHZ_READ_FAILED_TIMESTAMP   = PATHZ_READS + "/last-access-reject"
	PATHZ_WRITE_SUCCESS           = PATHZ_WRITES + "/access-accepts"
	PATHZ_WRITE_SUCCESS_TIMESTAMP = PATHZ_WRITES + "/last-access-accept"
	PATHZ_WRITE_FAILED            = PATHZ_WRITES + "/access-rejects"
	PATHZ_WRITE_FAILED_TIMESTAMP  = PATHZ_WRITES + "/last-access-reject"
	ACCOUNT_TBL                   = "CREDENTIALS|SSH_ACCOUNT"
	CONSOLE_TBL                   = "CREDENTIALS|CONSOLE_ACCOUNT"
	SSH_TBL                       = "CREDENTIALS|SSH_HOST"
)

type sshState struct {
	caKeys   certData
	hostCert certData
	hostKey  certData
	counters accessCounters
}

type accessCounters struct {
	accessRejects    uint64
	lastAccessReject uint64
	accessAccepts    uint64
	lastAccessAccept uint64
}

type certData struct {
	version string
	created uint64
}

// XfmrCache a sync.Map for storing path values that need to be cached
var XfmrCache sync.Map

var pathzOpers = [][]string{
	[]string{READS_GET, ACCEPTS},
	[]string{READS_GET, REJECTS},
	[]string{READS_SUB, ACCEPTS},
	[]string{READS_SUB, REJECTS},
	[]string{WRITES, ACCEPTS},
	[]string{WRITES, REJECTS}}

var pathzMap = &pathzCounters{
	mu:      sync.Mutex{},
	updated: make(map[string]time.Time),
	data:    make(map[string]map[string]map[string]*uint64),
}

func init() {
	XlateFuncBind("system_post_xfmr", system_post_xfmr)

	/* system/state */
	XlateFuncBind("DbToYang_sys_current_datetime_xfmr", DbToYang_sys_current_datetime_xfmr)
	XlateFuncBind("DbToYang_sys_up_time_xfmr", DbToYang_sys_up_time_xfmr)
	XlateFuncBind("DbToYang_sys_boot_time_xfmr", DbToYang_sys_boot_time_xfmr)
	XlateFuncBind("DbToYang_sys_software_version_xfmr", DbToYang_sys_software_version_xfmr)

	/* system/clock */
	XlateFuncBind("YangToDb_sys_clock_timezone_xfmr", YangToDb_sys_clock_timezone_xfmr)
	XlateFuncBind("DbToYang_sys_clock_timezone_xfmr", DbToYang_sys_clock_timezone_xfmr)

	/* system/processes */
	XlateFuncBind("YangToDb_sys_proc_pid_key_xfmr", YangToDb_sys_proc_pid_key_xfmr)
	XlateFuncBind("DbToYang_sys_proc_pid_key_xfmr", DbToYang_sys_proc_pid_key_xfmr)
	XlateFuncBind("DbToYang_sys_proc_pid_xfmr", DbToYang_sys_proc_pid_xfmr)
	XlateFuncBind("DbToYang_sys_proc_name_xfmr", DbToYang_sys_proc_name_xfmr)
	XlateFuncBind("DbToYang_sys_proc_args_xfmr", DbToYang_sys_proc_args_xfmr)
	XlateFuncBind("DbToYang_sys_process_cpu_utilization_xfmr", DbToYang_sys_process_cpu_utilization_xfmr)
	XlateFuncBind("DbToYang_sys_process_mem_utilization_xfmr", DbToYang_sys_process_mem_utilization_xfmr)

	/* system/ssh-server */
	XlateFuncBind("YangToDb_sys_ssh_timeout_xfmr", YangToDb_sys_ssh_timeout_xfmr)
	XlateFuncBind("DbToYang_sys_ssh_timeout_xfmr", DbToYang_sys_ssh_timeout_xfmr)

	/* Not implemented error */
	XlateFuncBind("YangToDb_sys_not_implemented_leaf_err_xfmr", YangToDb_sys_not_implemented_leaf_err_xfmr)
	XlateFuncBind("DbToYang_sys_not_implemented_leaf_err_xfmr", DbToYang_sys_not_implemented_leaf_err_xfmr)
	XlateFuncBind("YangToDb_sys_not_implemented_container_err_xfmr", YangToDb_sys_not_implemented_container_err_xfmr)
	XlateFuncBind("DbToYang_sys_not_implemented_container_err_xfmr", DbToYang_sys_not_implemented_container_err_xfmr)
	XlateFuncBind("Subscribe_sys_not_implemented_container_err_xfmr", Subscribe_sys_not_implemented_container_err_xfmr)

	/* system/logging */
	XlateFuncBind("YangToDb_sys_logging_remote_server_key_xfmr", YangToDb_sys_logging_remote_server_key_xfmr)
	XlateFuncBind("DbToYang_sys_logging_remote_server_key_xfmr", DbToYang_sys_logging_remote_server_key_xfmr)
	XlateFuncBind("YangToDb_sys_logging_vrf_xfmr", YangToDb_sys_logging_vrf_xfmr)
	XlateFuncBind("DbToYang_sys_logging_vrf_xfmr", DbToYang_sys_logging_vrf_xfmr)
	XlateFuncBind("YangToDb_sys_logging_selector_key_xfmr", YangToDb_sys_logging_selector_key_xfmr)
	XlateFuncBind("DbToYang_sys_logging_selector_key_xfmr", DbToYang_sys_logging_selector_key_xfmr)
	XlateFuncBind("YangToDb_sys_logging_selector_facility_xfmr", YangToDb_sys_logging_selector_facility_xfmr)
	XlateFuncBind("DbToYang_sys_logging_selector_facility_xfmr", DbToYang_sys_logging_selector_facility_xfmr)
	XlateFuncBind("YangToDb_sys_logging_selector_severity_xfmr", YangToDb_sys_logging_selector_severity_xfmr)
	XlateFuncBind("DbToYang_sys_logging_selector_severity_xfmr", DbToYang_sys_logging_selector_severity_xfmr)

	/* system/messages */
	XlateFuncBind("YangToDb_sys_msgs_severity_xfmr", YangToDb_sys_msgs_severity_xfmr)
	XlateFuncBind("DbToYang_sys_msgs_severity_xfmr", DbToYang_sys_msgs_severity_xfmr)

	/* system/ntp */
	XlateFuncBind("YangToDb_sys_ntp_config_enabled_xfmr", YangToDb_sys_ntp_config_enabled_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_config_enabled_xfmr", DbToYang_sys_ntp_config_enabled_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_config_enable_auth_xfmr", YangToDb_sys_ntp_config_enable_auth_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_config_enable_auth_xfmr", DbToYang_sys_ntp_config_enable_auth_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_key_key_xfmr", YangToDb_sys_ntp_key_key_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_key_key_xfmr", DbToYang_sys_ntp_key_key_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_key_type_xfmr", YangToDb_sys_ntp_key_type_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_key_type_xfmr", DbToYang_sys_ntp_key_type_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_server_key_xfmr", YangToDb_sys_ntp_server_key_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_server_key_xfmr", DbToYang_sys_ntp_server_key_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_server_association_type_xfmr", YangToDb_sys_ntp_server_association_type_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_server_association_type_xfmr", DbToYang_sys_ntp_server_association_type_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_server_iburst_xfmr", YangToDb_sys_ntp_server_iburst_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_server_iburst_xfmr", DbToYang_sys_ntp_server_iburst_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_server_vrf_xfmr", YangToDb_sys_ntp_server_vrf_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_server_vrf_xfmr", DbToYang_sys_ntp_server_vrf_xfmr)
	XlateFuncBind("YangToDb_sys_ntp_server_source_address_xfmr", YangToDb_sys_ntp_server_source_address_xfmr)
	XlateFuncBind("DbToYang_sys_ntp_server_source_address_xfmr", DbToYang_sys_ntp_server_source_address_xfmr)

	/* system/dns */
	XlateFuncBind("YangToDb_sys_dns_config_xfmr", YangToDb_sys_dns_config_xfmr)
	XlateFuncBind("DbToYang_sys_dns_config_xfmr", DbToYang_sys_dns_config_xfmr)
	XlateFuncBind("Subscribe_sys_dns_config_xfmr", Subscribe_sys_dns_config_xfmr)

	/* system/aaa */
	XlateFuncBind("YangToDb_sys_aaa_authentication_method_xfmr", YangToDb_sys_aaa_authentication_method_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_authentication_method_xfmr", DbToYang_sys_aaa_authentication_method_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_authorization_method_xfmr", YangToDb_sys_aaa_authorization_method_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_authorization_method_xfmr", DbToYang_sys_aaa_authorization_method_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_accounting_method_xfmr", YangToDb_sys_aaa_accounting_method_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_accounting_method_xfmr", DbToYang_sys_aaa_accounting_method_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_server_group_name_key_xfmr", YangToDb_sys_aaa_server_group_name_key_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_server_group_name_key_xfmr", DbToYang_sys_aaa_server_group_name_key_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_server_group_name_field_xfmr", YangToDb_sys_aaa_server_group_name_field_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_server_group_name_field_xfmr", DbToYang_sys_aaa_server_group_name_field_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_server_group_type_field_xfmr", YangToDb_sys_aaa_server_group_type_field_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_server_group_type_field_xfmr", DbToYang_sys_aaa_server_group_type_field_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_server_groups_address_key_xfmr", YangToDb_sys_aaa_server_groups_address_key_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_server_groups_address_key_xfmr", DbToYang_sys_aaa_server_groups_address_key_xfmr)
	XlateFuncBind("sys_aaa_server_groups_table_xfmr", sys_aaa_server_groups_table_xfmr)
	XlateFuncBind("sys_aaa_server_table_xfmr", sys_aaa_server_table_xfmr)
	XlateFuncBind("YangToDb_aaa_sys_source_address_xfmr", YangToDb_aaa_sys_source_address_xfmr)
	XlateFuncBind("DbToYang_aaa_sys_source_address_xfmr", DbToYang_aaa_sys_source_address_xfmr)
	XlateFuncBind("YangToDb_aaa_server_secret_key_xfmr", YangToDb_aaa_server_secret_key_xfmr)
	XlateFuncBind("DbToYang_aaa_server_secret_key_xfmr", DbToYang_aaa_server_secret_key_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_server_name_field_xfmr", YangToDb_sys_aaa_server_name_field_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_server_name_field_xfmr", DbToYang_sys_aaa_server_name_field_xfmr)
	XlateFuncBind("DbToYang_grpc_server_xfmr", DbToYang_grpc_server_xfmr)
	XlateFuncBind("Subscribe_grpc_server_xfmr", Subscribe_grpc_server_xfmr)
	XlateFuncBind("DbToYang_grpc_server_key_xfmr", DbToYang_grpc_server_key_xfmr)
	XlateFuncBind("DbToYang_ssh_server_state_xfmr", DbToYang_ssh_server_state_xfmr)
	XlateFuncBind("Subscribe_ssh_server_state_xfmr", Subscribe_ssh_server_state_xfmr)
	XlateFuncBind("DbToYang_authz_policy_xfmr", DbToYang_authz_policy_xfmr)
	XlateFuncBind("Subscribe_authz_policy_xfmr", Subscribe_authz_policy_xfmr)
	XlateFuncBind("DbToYang_pathz_policies_xfmr", DbToYang_pathz_policies_xfmr)
	XlateFuncBind("Subscribe_pathz_policies_xfmr", Subscribe_pathz_policies_xfmr)
	XlateFuncBind("DbToYang_pathz_policies_key_xfmr", DbToYang_pathz_policies_key_xfmr)
	XlateFuncBind("DbToYang_console_counters_xfmr", DbToYang_console_counters_xfmr)
	XlateFuncBind("Subscribe_console_counters_xfmr", Subscribe_console_counters_xfmr)
}

var system_post_xfmr PostXfmrFunc = func(inParams XfmrParams) error {
	var err error

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return invalid_db_err
	}

	if inParams.oper == DELETE {
		xpath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)

		switch xpath {
		case "/openconfig-system:system/ntp/servers/server":
			serverKeys, err := inParams.d.GetKeys(&db.TableSpec{Name: "NTP_SERVER"})

			if err == nil && len(serverKeys) == 1 {
				subOpDeleteMap := make(map[db.DBNum]map[string]map[string]db.Value)
				subOpDeleteMap[db.ConfigDB] = make(map[string]map[string]db.Value)
				subOpDeleteMap[db.ConfigDB]["NTP"] = make(map[string]db.Value)
				subOpDeleteMap[db.ConfigDB]["NTP"]["global"] = db.Value{Field: make(map[string]string, 2)}
				subOpDeleteMap[db.ConfigDB]["NTP"]["global"].Field["src_intf"] = ""
				subOpDeleteMap[db.ConfigDB]["NTP"]["global"].Field["vrf"] = ""
				inParams.subOpDataMap[DELETE] = &subOpDeleteMap
				log.Infof("System Post xfmr invoked, return Delete map %v", inParams.subOpDataMap[DELETE])
			}
		}
	}
	return err
}

var DbToYang_sys_current_datetime_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get the current time.
	now := time.Now()
	// Get the timezone offset.
	_, offset := now.Zone()

	// Format the datetime in YANG format.
	yangFormat := fmt.Sprintf("%s%+03d:%02d", now.Format("2006-01-02T15:04:05Z"), offset/3600, offset%3600/60)

	result["current-datetime"] = yangFormat
	return result, nil
}

var DbToYang_sys_up_time_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	var sysInfo syscall.Sysinfo_t

	err := syscall.Sysinfo(&sysInfo)
	if err != nil {
		return nil, fmt.Errorf("Failed to get system info: %v", err)
	}
	uptimeSeconds := sysInfo.Uptime
	result["up-time"] = strconv.FormatInt(int64(uptimeSeconds*1e9), 10)
	return result, nil
}

var DbToYang_sys_boot_time_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	var uptime syscall.Sysinfo_t

	err := syscall.Sysinfo(&uptime)
	if err != nil {
		return nil, fmt.Errorf("Failed to get system info: %v", err)
	}

	currentTime := time.Now().UnixNano()
	bootTime := currentTime - int64(uptime.Uptime)*int64(time.Second)
	result["boot-time"] = strconv.FormatInt(bootTime, 10)
	return result, nil
}

var DbToYang_sys_software_version_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	yamlFile, err := os.ReadFile("/etc/sonic/sonic_version.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read /etc/sonic/sonic_version.yml: %v", err)
	}

	var versionData map[string]interface{}
	if err := yaml.Unmarshal(yamlFile, &versionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sonic_version.yml: %v", err)
	}

	buildVer, ok := versionData["build_version"].(string)
	if !ok {
		return nil, fmt.Errorf("build_version not found or not a string in sonic_version.yml")
	}

	result["software-version"] = buildVer

	return result, nil
}

var YangToDb_sys_clock_timezone_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var err error

	if inParams.oper == DELETE {
		res_map["timezone"] = ""
		return res_map, nil
	}

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	timezoneNamePtr, ok := inParams.param.(*string)
	if !ok {
		return nil, invalid_input_err
	}
	timezoneName := *timezoneNamePtr

	_, err = time.LoadLocation(timezoneName)
	if err != nil {
		zoneErr := fmt.Errorf("Timezone %s does not conform format", timezoneName)
		return nil, zoneErr
	}

	res_map["timezone"] = timezoneName
	return res_map, err
}

var DbToYang_sys_clock_timezone_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["DEVICE_METADATA"]["localhost"]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["timezone"]) == 0 {
		return nil, nil
	}
	rmap["timezone-name"] = entry.Field["timezone"]
	return rmap, nil
}

var YangToDb_sys_proc_pid_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("pid")
	return ockey, nil
}

var DbToYang_sys_proc_pid_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{}, 1)
	var err error

	_, err = strconv.Atoi(inParams.key)
	if err != nil {
		return nil, nil
	}

	rmap["pid"] = inParams.key
	return rmap, nil
}

var DbToYang_sys_proc_pid_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	rmap["pid"] = inParams.key
	return rmap, nil
}

var DbToYang_sys_proc_name_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["PROCESS_STATS"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["CMD"]) > 0 {
		rmap["name"] = strings.Split(entry.Field["CMD"], " ")[0]

		return rmap, nil
	}
	return nil, nil
}

var DbToYang_sys_proc_args_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["PROCESS_STATS"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["CMD"]) > 0 {
		var args []interface{}
		p_name := strings.Split(entry.Field["CMD"], " ")[1:]
		args = make([]interface{}, 0, len(p_name))
		for _, v := range p_name {
			if len(v) > 0 {
				args = append(args, v)
			}
		}
		rmap["args"] = args
		return rmap, nil
	}
	return nil, nil
}

/* Float to uint8 */
var DbToYang_sys_process_cpu_utilization_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["PROCESS_STATS"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["%CPU"]) > 0 {
		f, _ := strconv.ParseFloat(entry.Field["%CPU"], 32)
		rmap["cpu-utilization"] = uint8(f)

		return rmap, nil
	}
	return nil, nil
}

/* Float to uint8 */
var DbToYang_sys_process_mem_utilization_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["PROCESS_STATS"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["%MEM"]) > 0 {
		f, _ := strconv.ParseFloat(entry.Field["%MEM"], 32)
		rmap["memory-utilization"] = uint8(f)

		return rmap, nil
	}
	return nil, nil
}

var YangToDb_sys_ssh_timeout_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	var err error

	if inParams.oper == DELETE {
		rmap["login_timeout"] = ""
		return rmap, nil
	}

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	timeout, ok := inParams.param.(*uint16)
	if !ok {
		return nil, invalid_input_err
	}
	if timeout != nil {
		rmap["login_timeout"] = fmt.Sprintf("%d", *timeout)
	}
	return rmap, err
}

var DbToYang_sys_ssh_timeout_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	var err error

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["SSH_SERVER"]["POLICIES"]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["login_timeout"]) > 0 {
		timeoutStr, ok := entry.Field["login_timeout"]
		if ok {
			timeoutVal, err := strconv.ParseUint(timeoutStr, 10, 16)
			if err != nil {
				return rmap, err
			}
			rmap["timeout"] = timeoutVal
		}
		return rmap, err
	}
	return rmap, nil
}

var YangToDb_sys_not_implemented_leaf_err_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	if inParams.requestUri == inParams.uri {
		return nil, not_implemented_err
	}
	return nil, nil
}

var DbToYang_sys_not_implemented_leaf_err_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	if inParams.requestUri == inParams.uri {
		return nil, not_implemented_err
	}
	return nil, nil
}

var YangToDb_sys_not_implemented_container_err_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	if inParams.requestUri == inParams.uri {
		return nil, not_implemented_err
	}
	return nil, nil
}

var DbToYang_sys_not_implemented_container_err_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	if inParams.requestUri == inParams.uri {
		return not_implemented_err
	}
	return nil
}

var Subscribe_sys_not_implemented_container_err_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	return result, nil
}

var YangToDb_sys_logging_remote_server_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("host")
	return ockey, nil
}

var DbToYang_sys_logging_remote_server_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{}, 1)

	rmap["host"] = inParams.key
	return rmap, nil
}

var YangToDb_sys_logging_vrf_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	vrf, ok := inParams.param.(*string)
	if !ok {
		return nil, invalid_input_err
	}
	rmap["vrf"] = *vrf
	return rmap, nil
}

var DbToYang_sys_logging_vrf_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["SYSLOG_SERVER"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["vrf"]) > 0 {
		vrf, ok := entry.Field["vrf"]
		if ok {
			rmap["network-instance"] = vrf
			return rmap, nil
		}
		return nil, invalid_input_err
	}
	return nil, nil
}

var YangToDb_sys_logging_selector_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("host")
	return ockey, nil
}

var DbToYang_sys_logging_selector_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{}, 1)

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["SYSLOG_SERVER"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["severity"]) > 0 {
		severity, ok := entry.Field["severity"]
		if ok {
			rmap["severity"] = sonicToOc_severity[severity]
			rmap["facility"] = "ALL"
			return rmap, nil
		}
		return nil, nil
	}
	return nil, nil
}

var YangToDb_sys_logging_selector_facility_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	pathInfo := NewPathInfo(inParams.uri)
	facility := pathInfo.Var("facility")

	if facility != "ALL" {
		return nil, errors.New("Invalid input, only ALL is supported")
	}
	return rmap, nil
}

var DbToYang_sys_logging_selector_facility_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	rmap["facility"] = "ALL"
	return rmap, nil
}

var YangToDb_sys_logging_selector_severity_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("severity")

	if translation, found := ocStrToSonic_severity[ockey]; found {
		rmap["severity"] = translation
		return rmap, nil
	}
	return nil, invalid_input_err
}

var DbToYang_sys_logging_selector_severity_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["SYSLOG_SERVER"][inParams.key]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["severity"]) > 0 {
		severity, ok := entry.Field["severity"]
		if ok {
			rmap["severity"] = sonicToOc_severity[severity]
			return rmap, nil
		}
		return nil, invalid_input_err
	}
	return nil, nil
}

var YangToDb_sys_msgs_severity_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}
	severity, ok := inParams.param.(ocbinds.E_OpenconfigMessages_SyslogSeverity)
	if !ok {
		return nil, invalid_input_err
	}
	if translation, found := ocToSonic_severity[severity]; found {
		rmap["severity"] = translation
		return rmap, nil
	}
	return nil, invalid_input_err
}

var DbToYang_sys_msgs_severity_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry, ok := data["SYSLOG_CONFIG"]["GLOBAL"]
	if !ok {
		return nil, nil
	}

	if len(entry.Field["severity"]) > 0 {
		severity, ok := entry.Field["severity"]
		if ok {
			rmap["severity"] = sonicToOc_severity[severity]
			return rmap, nil
		}
		return nil, errors.New("Invalid data")
	}
	return nil, nil
}

var YangToDb_sys_ntp_config_enabled_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var enStr string

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	enabled, ok := inParams.param.(*bool)
	if !ok {
		return nil, invalid_input_err
	}
	if *enabled {
		enStr = "enabled"
	} else {
		enStr = "disabled"
	}
	res_map["admin_state"] = enStr

	return res_map, nil
}

var DbToYang_sys_ntp_config_enabled_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP"]
	if _, ok := tbl["global"]; !ok {
		return nil, nil
	}

	tblData := tbl["global"]
	dbData, ok := tblData.Field["admin_state"]
	if ok {
		if dbData == "enabled" {
			result["enabled"] = true
		} else {
			result["enabled"] = false
		}
	} else {
		log.Info("Admin state field not found in DB")
	}
	return result, nil
}

var YangToDb_sys_ntp_config_enable_auth_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var enStr string

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	enabled, ok := inParams.param.(*bool)
	if !ok {
		return nil, invalid_input_err
	}
	if *enabled {
		enStr = "enabled"
	} else {
		enStr = "disabled"
	}
	res_map["authentication"] = enStr

	return res_map, nil
}

var DbToYang_sys_ntp_config_enable_auth_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP"]
	if _, ok := tbl["global"]; !ok {
		return nil, nil
	}

	tblData := tbl["global"]
	dbData, ok := tblData.Field["authentication"]
	if ok {
		if dbData == "enabled" {
			result["enable-ntp-auth"] = true
		} else {
			result["enable-ntp-auth"] = false
		}
	} else {
		log.Info("Authentication field not found in DB")
	}
	return result, nil
}

var YangToDb_sys_ntp_key_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("key-id")
	return ockey, nil
}

var DbToYang_sys_ntp_key_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{}, 1)

	value, _ := strconv.ParseUint(inParams.key, 10, 16)
	rmap["key-id"] = uint16(value)
	return rmap, nil
}

var YangToDb_sys_ntp_key_type_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var typeStr string

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	keyType, ok := inParams.param.(ocbinds.E_OpenconfigSystem_NTP_AUTH_TYPE)
	if !ok {
		return nil, invalid_input_err
	}
	if keyType == ocbinds.OpenconfigSystem_NTP_AUTH_TYPE_NTP_AUTH_MD5 {
		typeStr = "md5"
	} else {
		return nil, invalid_input_err
	}
	res_map["type"] = typeStr

	return res_map, nil
}

var DbToYang_sys_ntp_key_type_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP_KEY"]
	if _, ok := tbl[inParams.key]; !ok {
		return nil, nil
	}

	tblData := tbl[inParams.key]
	dbData, ok := tblData.Field["type"]
	if ok {
		if dbData == "md5" {
			result["key-type"] = "NTP_AUTH_MD5"
		} else {
			return nil, errors.New("Invalid input, only MD5 is supported")
		}
	} else {
		log.Info("Key type field not found in DB")
	}
	return result, nil
}

var YangToDb_sys_ntp_server_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	ockey := pathInfo.Var("address")
	return ockey, nil
}

var DbToYang_sys_ntp_server_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{}, 1)

	rmap["address"] = inParams.key
	return rmap, nil
}

var YangToDb_sys_ntp_server_association_type_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var typeStr string

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	assocType, ok := inParams.param.(ocbinds.E_OpenconfigSystem_System_Ntp_Servers_Server_Config_AssociationType)
	if !ok {
		return nil, invalid_input_err
	}
	if assocType == ocbinds.OpenconfigSystem_System_Ntp_Servers_Server_Config_AssociationType_SERVER {
		typeStr = "server"
	} else if assocType == ocbinds.OpenconfigSystem_System_Ntp_Servers_Server_Config_AssociationType_POOL {
		typeStr = "pool"
	} else if assocType == ocbinds.OpenconfigSystem_System_Ntp_Servers_Server_Config_AssociationType_UNSET {
		return nil, nil
	} else {
		log.Infof("Invalid input %d, only SERVER & POOL are supported", assocType)
		return nil, errors.New("Invalid input, only SERVER & POOL are supported")
	}
	res_map["association_type"] = typeStr

	return res_map, nil
}

var DbToYang_sys_ntp_server_association_type_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP_SERVER"]
	if _, ok := tbl[inParams.key]; !ok {
		return nil, nil
	}

	tblData := tbl[inParams.key]
	dbData, ok := tblData.Field["association_type"]
	if ok {
		if dbData == "server" {
			result["association-type"] = "SERVER"
		} else if dbData == "pool" {
			result["association-type"] = "POOL"
		}
	} else {
		log.Info("Authentication field not found in DB")
	}
	return result, nil
}

var YangToDb_sys_ntp_server_iburst_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var enStr string

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	enabled, ok := inParams.param.(*bool)
	if !ok {
		return nil, invalid_input_err
	}
	if *enabled {
		enStr = "on"
	} else {
		enStr = "off"
	}
	res_map["iburst"] = enStr

	return res_map, nil
}

var DbToYang_sys_ntp_server_iburst_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP_SERVER"]
	if _, ok := tbl[inParams.key]; !ok {
		return nil, nil
	}

	tblData := tbl[inParams.key]
	dbData, ok := tblData.Field["iburst"]
	if ok {
		if dbData == "on" {
			result["iburst"] = true
		} else {
			result["iburst"] = false
		}
	} else {
		log.Info("iburst field not found in DB")
	}
	return result, nil
}

/* Delete will be handled in postXfmr action
 * If user passed vrf -> If should be same as global (if exist)
 */
var YangToDb_sys_ntp_server_vrf_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)

	if inParams.oper == DELETE {
		return nil, errors.New("Delete server instead of network-instance removal")
	}

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	vrf, ok := inParams.param.(*string)
	if !ok {
		return nil, invalid_input_err
	}
	vrfName := *vrf

	if vrfName == "mgmt" {
		mgmtVrfCfgT := &db.TableSpec{Name: "MGMT_VRF_CONFIG"}
		mgmtVrfCfgE, err := inParams.d.GetEntry(mgmtVrfCfgT, db.Key{Comp: []string{"vrf_global"}})
		if err == nil {
			mgmtVrfEnabled, ok := mgmtVrfCfgE.Field["mgmtVrfEnabled"]
			if ok && mgmtVrfEnabled == "false" {
				return nil, errors.New("Mgmt VRF config is not enabled")
			}
		}
	}

	ntpTbl := &db.TableSpec{Name: "NTP"}
	ntpEntry, err := inParams.d.GetEntry(ntpTbl, db.Key{Comp: []string{"global"}})
	if err == nil {
		dbData, ok := ntpEntry.Field["vrf"]
		if ok {
			if dbData != vrfName {
				return nil, errors.New("Given network-instance name is different from already configured one for this/any other server")
			}
		}
	}
	log.Info("vrf field not found in DB")
	res_map["vrf"] = vrfName

	return res_map, nil
}

var DbToYang_sys_ntp_server_vrf_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP"]
	if _, ok := tbl["global"]; !ok {
		return nil, nil
	}

	tblData := tbl["global"]
	dbData, ok := tblData.Field["vrf"]
	if ok && len(dbData) > 0 {
		result["network-instance"] = dbData
	} else {
		log.Info("vrf field not found in DB")
		return nil, nil
	}
	return result, nil
}

// YangToDb - Function to fetch Interface for given source address
func get_src_intf_for_given_addr(src_ip string, inParams XfmrParams) string {
	var ifName string
	var vrfName string
	def_vrf := false
	tblList := make([]string, 0, len(intfTblList)+1)

	// Fetch VRF -> 1) In current inParams 2) In DB
	pathInfo := NewPathInfo(inParams.uri)
	currServer := pathInfo.Var("address")
	obj := (*inParams.ygRoot).(*ocbinds.Device)
	sobj := obj.System
	if sobj != nil && sobj.Ntp != nil && sobj.Ntp.Servers != nil {
		for _, server := range sobj.Ntp.Servers.Server {
			if server.Address != nil && *server.Address == currServer {
				if server.Config.NetworkInstance != nil {
					vrfName = *server.Config.NetworkInstance
				}
				break
			}
		}
	}

	if len(vrfName) == 0 {
		ntpTbl := &db.TableSpec{Name: "NTP"}
		ntpEntry, err := inParams.d.GetEntry(ntpTbl, db.Key{Comp: []string{"global"}})
		if err == nil {
			vrfName, _ = ntpEntry.Field["vrf"]
		}
	}

	log.Infof("Start fetching interface for given ip %s and vrf %s..", src_ip, vrfName)
	if len(vrfName) > 0 {
		if vrfName == "mgmt" {
			tblList = append(tblList, "MGMT_INTERFACE")
		} else if vrfName == "default" {
			def_vrf = true
			tblList = append(tblList, intfTblList...)
		}
	} else {
		tblList = append(tblList, intfTblList...)
		def_vrf = true
	}

	if def_vrf == true {
		mgmtVrfCfgT := &db.TableSpec{Name: "MGMT_VRF_CONFIG"}
		mgmtVrfCfgE, err := inParams.d.GetEntry(mgmtVrfCfgT, db.Key{Comp: []string{"vrf_global"}})
		if err == nil {
			mgmtVrfEnabled, ok := mgmtVrfCfgE.Field["mgmtVrfEnabled"]
			if ok && mgmtVrfEnabled == "false" {
				log.Info("Fetch Interface, Mgmt VRF config is not enabled")
				tblList = append(tblList, "MGMT_INTERFACE")
			}
		} else {
			log.Info("Fetch Interface, Mgmt VRF config is not enabled")
			tblList = append(tblList, "MGMT_INTERFACE")
		}
	}

	log.Infof("Fetch Interface, table list to be looked into %v", tblList)
	for _, tblName := range tblList {
		intfTable := &db.TableSpec{Name: tblName}

		intfKeys, err := inParams.d.GetKeysPattern(intfTable, db.Key{Comp: []string{"*", src_ip}})
		if (err != nil) || len(intfKeys) == 0 {
			src_ip_w_mask := src_ip + "/*"
			intfKeys, err = inParams.d.GetKeysPattern(intfTable, db.Key{Comp: []string{"*", src_ip_w_mask}})
		}

		if (err == nil) && len(intfKeys) > 0 {

			for _, intfKey := range intfKeys {

				if len(intfKey.Comp) != 2 && len(intfKey.Comp[0]) == 0 {
					continue
				}

				ifName = intfKey.Comp[0]
				/* Validate VRF */
				if def_vrf && tblName != "MGMT_INTERFACE" {
					if intfEntry, err := inParams.d.GetEntry(intfTable, db.Key{Comp: []string{ifName}}); err == nil {
						vrfName := (&intfEntry).Get("vrf_name")
						if len(vrfName) > 0 {
							log.Infof("Fetch Interface, fetched vrf for interface %s is %s", ifName, vrfName)
							ifName = ""
							continue
						}
					}
				}
				break
			}
		}
		if len(ifName) > 0 {
			break
		}
	}
	return ifName
}

// DbToYang - Function to fetch IP prefix for given interface and vrf pair
func get_src_addr_for_interface(ifName string, vrfName string, inParams XfmrParams) string {
	var src_addr string
	def_vrf := false
	tblList := make([]string, 0, len(intfTblList)+1)

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return src_addr
	}

	data := (*inParams.dbDataMap)[inParams.curDb]

	log.Infof("Start fetching ip prefix for interface %s and vrf %s..", ifName, vrfName)
	if len(vrfName) > 0 {
		if vrfName == "mgmt" {
			tblList = append(tblList, "MGMT_INTERFACE")
		} else if vrfName == "default" {
			def_vrf = true
			tblList = append(tblList, intfTblList...)
		}
	} else {
		tblList = append(tblList, intfTblList...)
		def_vrf = true
	}

	if def_vrf == true {
		tbl := data["MGMT_VRF_CONFIG"]
		if tblData, ok := tbl["vrf_global"]; ok {
			mgmtVrfEnabled, ok := tblData.Field["mgmtVrfEnabled"]
			if ok && mgmtVrfEnabled == "false" {
				log.Info("Fetch IP Prefix, Mgmt VRF config is not enabled")
				tblList = append(tblList, "MGMT_INTERFACE")
			}
		} else {
			log.Info("Fetch IP Prefix, Mgmt VRF config is not enabled")
			tblList = append(tblList, "MGMT_INTERFACE")
		}
	}

	log.Infof("Fetch IP Prefix, table list to be looked into %v", tblList)
	for _, tblName := range tblList {
		intfTable := &db.TableSpec{Name: tblName}

		/* Validate VRF */
		if def_vrf && tblName != "MGMT_INTERFACE" {
			if intfEntry, err := inParams.d.GetEntry(intfTable, db.Key{Comp: []string{ifName}}); err == nil {
				vrfName := (&intfEntry).Get("vrf_name")
				if len(vrfName) > 0 {
					log.Infof("Fetch IP Prefix, fetched vrf for interface %s is %s", ifName, vrfName)
					return src_addr
				}
			}
		}

		/* Get first ip configured on the given port */
		ipKeys, err := inParams.d.GetKeysPattern(intfTable, db.Key{Comp: []string{ifName, "*"}})
		if err == nil && len(ipKeys) > 0 && len(ipKeys[0].Comp) == 2 {
			idx := strings.Index(ipKeys[0].Comp[1], "/")
			if idx != -1 {
				return ipKeys[0].Comp[1][:idx]
			}
			return ipKeys[0].Comp[1]
		}
	}
	return src_addr
}

var YangToDb_sys_ntp_server_source_address_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	src_ip, ok := inParams.param.(*string)
	if !ok {
		return nil, invalid_input_err
	}
	if len(*src_ip) > 6 {
		ifName := get_src_intf_for_given_addr(*src_ip, inParams)
		if len(ifName) > 0 {

			ntpTbl := &db.TableSpec{Name: "NTP"}
			ntpEntry, err := inParams.d.GetEntry(ntpTbl, db.Key{Comp: []string{"global"}})
			if err == nil {
				dbData, ok := ntpEntry.Field["src_intf"]
				if ok {
					if dbData != ifName {
						return nil, errors.New("Given source address's port doesn't match with already configured src_intf")
					}
				}
			}
			res_map["src_intf"] = ifName
		} else {
			return nil, errors.New("Failed to get source interface for given source address")
		}
	} else {
		return nil, invalid_input_err
	}
	return res_map, nil
}

var DbToYang_sys_ntp_server_source_address_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	tbl := data["NTP"]
	if _, ok := tbl["global"]; !ok {
		return nil, nil
	}

	tblData := tbl["global"]
	intfName, ok := tblData.Field["src_intf"]
	if ok {
		vrfName, _ := tblData.Field["vrf"]
		addr := get_src_addr_for_interface(intfName, vrfName, inParams)
		if len(addr) > 0 {
			result["source-address"] = addr
		} else {
			return nil, errors.New("Source address not configured properly for src_intf " + intfName)
		}
	} else {
		log.Info("src_intf field not found in DB")
	}
	return result, nil
}

var YangToDb_sys_dns_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	dnsNsTableMap := make(map[string]map[string]db.Value)
	var nsList []string
	var nsListDb []string

	if inParams.ygRoot == nil {
		return nil, invalid_input_err
	}

	/* Set invokeCRUSubtreeOnce flag to invoke subtree once */
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}
	dnsTbl := &db.TableSpec{Name: "DNS_NAMESERVER"}

	switch inParams.oper {
	case DELETE:
		{
			// Note : Value specific delete not supported
			// Get db data
			dnsKeys, err := inParams.d.GetKeysPattern(dnsTbl, db.Key{Comp: []string{"*"}})
			if err == nil && len(dnsKeys) > 0 {
				for _, key := range dnsKeys {
					nsList = append(nsList, key.Comp[0])
				}
			} else {
				// No Data in DB
				return nil, nil
			}
		}
	case REPLACE:
		{
			// Get db data
			dnsKeys, err := inParams.d.GetKeysPattern(dnsTbl, db.Key{Comp: []string{"*"}})
			if err == nil && len(dnsKeys) > 0 {
				for _, key := range dnsKeys {
					nsListDb = append(nsListDb, key.Comp[0])
				}
			}
		}
		fallthrough
	case CREATE:
		fallthrough
	case UPDATE:
		{
			// Get ygRoot
			obj := (*inParams.ygRoot).(*ocbinds.Device)
			dnsObj := obj.System.Dns
			dnsConfigObj := dnsObj.Config
			if dnsConfigObj == nil {
				return nil, invalid_input_err
			}

			nsList = dnsConfigObj.Search
			if len(nsList) == 0 {
				return nil, invalid_input_err
			}
		}
	default:
		return nil, not_implemented_err
	}

	// Dummy db field value for return map
	fVal := make(map[string]string)
	//dbVal["NULL"] = "NULL"
	newVal := db.Value{Field: fVal}
	tblName := "DNS_NAMESERVER"

	// Delete old entries for Replace
	if inParams.oper == REPLACE {
		for _, oldNs := range nsListDb {
			if !contains(nsList, oldNs) {
				dbErr := inParams.d.DeleteEntry(dnsTbl, db.Key{Comp: []string{oldNs}})
				if dbErr != nil {
					return nil, errors.New("Error!!! Failed to remove entry from CONFIG_DB")
				}
				log.Infof("DNS removed %v entry from CONFIG_DB", oldNs)
			}
		}
	}

	for _, ns := range nsList {
		if _, ok := dnsNsTableMap[tblName]; !ok {
			dnsNsTableMap[tblName] = make(map[string]db.Value)
		}
		dnsNsTableMap[tblName][ns] = newVal

	}

	return dnsNsTableMap, nil
}

var DbToYang_sys_dns_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var nameServers []string

	if inParams.ygRoot == nil {
		return nil
	}

	// Get db data
	dnsTbl := &db.TableSpec{Name: "DNS_NAMESERVER"}
	dnsKeys, err := inParams.d.GetKeysPattern(dnsTbl, db.Key{Comp: []string{"*"}})
	if err == nil && len(dnsKeys) > 0 {
		for _, key := range dnsKeys {
			nameServers = append(nameServers, key.Comp[0])
		}
	} else {
		return nil
	}

	// Get ygRoot
	obj := (*inParams.ygRoot).(*ocbinds.Device)
	dnsObj := obj.System.Dns
	dnsConfigObj := dnsObj.Config
	ygot.BuildEmptyTree(dnsConfigObj)

	// Update DB data in ygRoot
	dnsConfigObj.Search = nameServers
	return nil
}

var Subscribe_sys_dns_config_xfmr = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams

	if inParams.subscProc == TRANSLATE_SUBSCRIBE {
		log.V(3).Info("Subscribe system/dns/config : inParams.subscProc: ", inParams.subscProc)

		pathInfo := NewPathInfo(inParams.uri)
		targetUriPath := pathInfo.YangPath

		log.V(3).Infof("Subscribe system/dns/config :- URI:%s pathinfo:%s ", inParams.uri, pathInfo.Path)
		log.V(3).Infof("Subscribe system/dns/config :- Target URI path:%s", targetUriPath)

		// to handle the TRANSLATE_SUBSCRIBE
		result.nOpts = new(notificationOpts)
		result.nOpts.pType = OnChange
		result.nOpts.mInterval = 15
		result.isVirtualTbl = false
		result.needCache = true

		result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {
			"DNS_NAMESERVER": {"*": {}}}}

		log.V(3).Info("Subscribe system/dns/config : result ", result)
	}
	return result, nil
}

var DbToYang_sys_aaa_authentication_method_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry := data["AAA"]["authentication"]
	loginStr, ok := entry.Field["login"]

	if !ok || len(loginStr) == 0 {
		return nil, nil
	}

	authMethods := strings.Split(loginStr, ",")
	var authMethodsList []interface{}

	for _, method := range authMethods {
		methodType, err := openconfig_aaa_translate_DBFormat_To_methodtype(method)
		if err != nil {
			return nil, err
		}
		authMethodsList = append(authMethodsList, methodType)
	}
	rmap["authentication-method"] = authMethodsList
	return rmap, nil
}

var YangToDb_sys_aaa_authentication_method_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}
	authRes, err := openconfig_aaa_process_method_ops(inParams, "authentication")
	if err != nil {
		return nil, err
	}

	authMethods, ok := authRes["authentication-method"]
	if !ok || (len(authMethods) == 0 && inParams.oper != DELETE) {
		return nil, aaa_failed_no_method_err
	}

	rmap["login"] = strings.Join(authMethods, ",")
	return rmap, nil
}

var DbToYang_sys_aaa_authorization_method_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry := data["AAA"]["authorization"]
	loginStr, ok := entry.Field["login"]

	if !ok || len(loginStr) == 0 {
		return nil, nil
	}

	authMethods := strings.Split(loginStr, ",")
	var authMethodsList []interface{}

	for _, method := range authMethods {
		methodType, err := openconfig_aaa_translate_DBFormat_To_methodtype(method)
		if err != nil {
			return nil, err
		}
		authMethodsList = append(authMethodsList, methodType)
	}

	rmap["authorization-method"] = authMethodsList
	return rmap, nil
}

var YangToDb_sys_aaa_authorization_method_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}
	authRes, err := openconfig_aaa_process_method_ops(inParams, "authorization")
	if err != nil {
		return nil, err
	}

	authMethods, ok := authRes["authorization-method"]
	if !ok || (len(authMethods) == 0 && inParams.oper != DELETE) {
		return nil, aaa_failed_no_method_err
	}

	rmap["login"] = strings.Join(authMethods, ",")
	return rmap, nil
}

var DbToYang_sys_aaa_accounting_method_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	data := (*inParams.dbDataMap)[inParams.curDb]
	entry := data["AAA"]["accounting"]
	loginStr, ok := entry.Field["login"]

	if !ok || len(loginStr) == 0 {
		return nil, nil
	}

	authMethods := strings.Split(loginStr, ",")
	var authMethodsList []interface{}

	for _, method := range authMethods {
		methodType, err := openconfig_aaa_translate_DBFormat_To_methodtype(method)
		if err != nil {
			return nil, err
		}
		authMethodsList = append(authMethodsList, methodType)
	}

	rmap["accounting-method"] = authMethodsList
	return rmap, nil
}

var YangToDb_sys_aaa_accounting_method_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	authRes, err := openconfig_aaa_process_method_ops(inParams, "accounting")
	if err != nil {
		return nil, err
	}

	authMethods, ok := authRes["accounting-method"]
	if !ok || (len(authMethods) == 0 && inParams.oper != DELETE) {
		return nil, aaa_failed_no_method_err
	}

	rmap["login"] = strings.Join(authMethods, ",")
	return rmap, nil
}

// openconfig_aaa_process_method_ops manages methods for the openconfig-system module for the AAA and transforms methods (authentication, authorization, accounting)
// based on the type of operation (CREATE, UPDATE, REPLACE). It interacts with the database to fetch existing methods,
// processes the provided parameters, and returns a map of the transformed methods.

func openconfig_aaa_process_method_ops(inParams XfmrParams, key string) (map[string][]string, error) {
	rmap := make(map[string][]string)

	if inParams.oper == DELETE {
		rmap[openconfig_aaa_get_Method_Key(key)] = []string{}
		return rmap, nil
	}

	var methods []string
	syst := &db.TableSpec{Name: "AAA"}
	entry, _ := inParams.d.GetEntry(syst, db.Key{Comp: []string{key}})
	methodsStr, ok := entry.Field["login"]
	var existingMethods []string

	if ok && len(methodsStr) > 0 {
		methods = strings.Split(methodsStr, ",")
		for _, method := range methods {
			if len(method) > 0 {
				existingMethods = append(existingMethods, method)
			}
		}
	}

	var methodsToAdd []string
	switch key {
	case "authentication":
		if v, ok := inParams.param.([]ocbinds.OpenconfigSystem_System_Aaa_Authentication_Config_AuthenticationMethod_Union); ok {
			for _, method := range v {
				methodStr, err := openconfig_aaa_extract_method_string_enum(method)
				if err != nil {
					return nil, err
				}
				methodsToAdd = append(methodsToAdd, methodStr)
			}
		} else {
			return nil, invalid_input_err
		}
	case "authorization":
		if v, ok := inParams.param.([]ocbinds.OpenconfigSystem_System_Aaa_Authorization_Config_AuthorizationMethod_Union); ok {
			for _, method := range v {
				methodStr, err := openconfig_aaa_extract_method_string_enum(method)
				if err != nil {
					return nil, err
				}
				methodsToAdd = append(methodsToAdd, methodStr)
			}
		} else {
			return nil, invalid_input_err
		}
	case "accounting":
		if v, ok := inParams.param.([]ocbinds.OpenconfigSystem_System_Aaa_Accounting_Config_AccountingMethod_Union); ok {
			for _, method := range v {
				methodStr, err := openconfig_aaa_extract_method_string_enum(method)
				if err != nil {
					return nil, err
				}
				methodsToAdd = append(methodsToAdd, methodStr)
			}
		} else {
			return nil, invalid_input_err
		}
	default:
		return nil, fmt.Errorf("Only authentication,authorization and accounting are supported but received %s", key)
	}

	switch inParams.oper {
	case CREATE, UPDATE:
		if len(existingMethods) == 0 {
			rmap[openconfig_aaa_get_Method_Key(key)] = methodsToAdd
		} else {
			for _, method := range methodsToAdd {
				exists := false
				for _, existingMethod := range existingMethods {
					if method == existingMethod {
						exists = true
						break
					}
				}
				if !exists {
					existingMethods = append(existingMethods, method)
				}
			}
			rmap[openconfig_aaa_get_Method_Key(key)] = existingMethods
		}

	case REPLACE:
		rmap[openconfig_aaa_get_Method_Key(key)] = methodsToAdd
	default:
		return nil, fmt.Errorf("Operation type %s is not supported", inParams.oper)
	}
	return rmap, nil
}

// helper function to map openconfig type(Method type) with sonic type
func openconfig_aaa_translate_methodtype_To_dBFormat(methodType ocbinds.E_OpenconfigAaaTypes_AAA_METHOD_TYPE) (string, error) {
	switch methodType {
	case ocbinds.OpenconfigAaaTypes_AAA_METHOD_TYPE_LOCAL:
		return "local", nil
	case ocbinds.OpenconfigAaaTypes_AAA_METHOD_TYPE_RADIUS_ALL:
		return "radius", nil
	case ocbinds.OpenconfigAaaTypes_AAA_METHOD_TYPE_TACACS_ALL:
		return "tacacs+", nil
	default:
		return "", fmt.Errorf("This method type is not allowed,only LOCAL,RADIUS and TACACS method type is allowed but received %v", methodType)
	}
}

// helper function to map sonic type with openconfig type
func openconfig_aaa_translate_DBFormat_To_methodtype(methodStr string) (string, error) {
	switch methodStr {
	case "local":
		return "LOCAL", nil
	case "radius":
		return "RADIUS_ALL", nil
	case "tacacs+":
		return "TACACS_ALL", nil
	case "ldap":
		return "ldap", nil
	case "default":
		return "default", nil
	default:
		return "", fmt.Errorf("Strings apart from local,radius,tacacs+,ldap and default are not allowed but received %s", methodStr)
	}
}

// Helper function to extract method string and identity ref  from method type
// mapping oc values identity ref to sonic
func openconfig_aaa_extract_method_string_enum(method interface{}) (string, error) {
	switch m := method.(type) {
	case *ocbinds.OpenconfigSystem_System_Aaa_Authentication_Config_AuthenticationMethod_Union_E_OpenconfigAaaTypes_AAA_METHOD_TYPE:
		methodStr, err := openconfig_aaa_translate_methodtype_To_dBFormat(m.E_OpenconfigAaaTypes_AAA_METHOD_TYPE) // No dereference needed
		if err != nil {
			return "", err
		}
		return methodStr, nil

	case *ocbinds.OpenconfigSystem_System_Aaa_Authorization_Config_AuthorizationMethod_Union_E_OpenconfigAaaTypes_AAA_METHOD_TYPE:
		methodStr, err := openconfig_aaa_translate_methodtype_To_dBFormat(m.E_OpenconfigAaaTypes_AAA_METHOD_TYPE) // No dereference needed
		if err != nil {
			return "", err
		}
		return methodStr, nil

	case *ocbinds.OpenconfigSystem_System_Aaa_Accounting_Config_AccountingMethod_Union_E_OpenconfigAaaTypes_AAA_METHOD_TYPE:
		methodStr, err := openconfig_aaa_translate_methodtype_To_dBFormat(m.E_OpenconfigAaaTypes_AAA_METHOD_TYPE) // No dereference needed
		if err != nil {
			return "", err
		}
		return methodStr, nil

	case *ocbinds.OpenconfigSystem_System_Aaa_Authentication_Config_AuthenticationMethod_Union_String:
		if m.String != "ldap" && m.String != "default" {
			return "", fmt.Errorf("Invalid method string: %s; only 'ldap' and 'default' are allowed", m.String)
		}
		return m.String, nil
	case *ocbinds.OpenconfigSystem_System_Aaa_Authorization_Config_AuthorizationMethod_Union_String:
		if m.String != "ldap" && m.String != "default" {
			return "", fmt.Errorf("Invalid method string: %s; only 'ldap' and 'default' are allowed", m.String)
		}
		return m.String, nil
	case *ocbinds.OpenconfigSystem_System_Aaa_Accounting_Config_AccountingMethod_Union_String:
		if m.String != "ldap" && m.String != "default" {
			return "", fmt.Errorf("Invalid method string: %s; only 'ldap' and 'default' are allowed", m.String)
		}
		return m.String, nil
	}
	return "", nil
}

// helper function to get key for processing the function openconfig_aaa_process_method_ops
func openconfig_aaa_get_Method_Key(key string) string {
	switch key {
	case "authentication":
		return "authentication-method"
	case "authorization":
		return "authorization-method"
	case "accounting":
		return "accounting-method"
	default:
		return key
	}
}

var YangToDb_sys_aaa_server_group_name_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)
	name := pathInfo.Var("name")
	if name == "" {
		return name, nil
	}
	if name != "tacacs+" && name != "radius" {
		log.Error("Invalid server group name:", name)
		return "", fmt.Errorf("Invalid server group name: %s; must be either 'tacacs+' or 'radius'", name)
	}

	return name, nil
}

var DbToYang_sys_aaa_server_group_name_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	pathInfo := NewPathInfo(inParams.uri)
	reqpathInfo := NewPathInfo(inParams.requestUri)
	requestUriPath := reqpathInfo.YangPath

	log.Infof("DbToYang_sys_aaa_server_group_name_key_xfmr: inParams.uri: %s, pathInfo: %s, inParams.requestUri: %s", inParams.uri, pathInfo, requestUriPath)
	srvGrpName := reqpathInfo.Var("name")

	if srvGrpName == "" {
		log.Infof("DbToYang_sys_aaa_server_group_name_key_xfmr: inParams.table: %s", inParams.table)
		if inParams.table == "TACPLUS_SERVER" {
			rmap["name"] = "tacacs+"
			log.Info("DbToYang_sys_aaa_server_group_name_key_xfmr - Mapped TACPLUS_SERVER to name: tacacs")
			return rmap, nil
		} else if inParams.table == "RADIUS_SERVER" {
			rmap["name"] = "radius"
			log.Info("DbToYang_sys_aaa_server_group_name_key_xfmr - Mapped RADIUS_SERVER to name: radius")
			return rmap, nil
		}
	}

	// Use inParams.key directly for mapping
	serverName := inParams.key
	log.Info("DbToYang_sys_aaa_server_group_name_key_xfmr - Received server group name: ", serverName)
	if serverName != "tacacs+" && serverName != "radius" {
		log.Error("DbToYang_sys_aaa_server_group_name_key_xfmr - Unknown server group name: ", serverName)
		return nil, fmt.Errorf("unknown server group name: %s", serverName)
	}
	rmap["name"] = serverName
	log.Info("DbToYang_sys_aaa_server_group_name_key_xfmr returns: ", rmap)
	return rmap, nil
}

var YangToDb_sys_aaa_server_group_name_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	var err error

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	log.Info("YangToDb_sys_aaa_server_group_name_field_xfmr - Received input parameter:", inParams.param)

	// Attempt to cast inParams.param to a string
	name, ok := inParams.param.(*string)
	if !ok {
		return rmap, fmt.Errorf("Expected a string, got %T", inParams.param)
	}
	log.Info("YangToDb_sys_aaa_server_group_name_field_xfmr - Validating server group name:", *name)
	// Validate the server group name
	if *name != "tacacs+" && *name != "radius" {
		log.Error("Invalid server group name:", *name)
		return rmap, fmt.Errorf("Invalid server group name: %s; must be either 'tacacs' or 'radius'", name)
	}
	return rmap, err
}

var DbToYang_sys_aaa_server_group_name_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	pathInfo := NewPathInfo(inParams.uri)
	serverName := pathInfo.Var("name")

	log.Info("DbToYang_sys_aaa_server_group_name_field_xfmr - inParams.uri ", inParams.uri)

	log.Info("DbToYang_sys_aaa_server_group_name_field_xfmr - Received server group name:", serverName)
	// Check if the server group name is empty
	if serverName == "" {
		log.Error("Error: server group name is empty")
		return nil, fmt.Errorf("server group name is empty")
	}

	rmap["name"] = serverName

	log.Info("DbToYang_sys_aaa_server_group_name_field_xfmr returns ", rmap)
	return rmap, nil // Return nil for error if everything is fine
}

var YangToDb_sys_aaa_server_name_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	var err error
	// Do nothing for server name as it is not stored in DB.
	return rmap, err
}

var DbToYang_sys_aaa_server_name_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	//Server name is not stored in DB. So, it is not possible to get the server name, retun nil
	log.Info("DbToYang_sys_aaa_server_name_field_xfmr - inParams.uri ", inParams.uri)

	return rmap, nil // Return nil for error if everything is fine
}

var YangToDb_sys_aaa_server_group_type_field_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	var err error

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	serverType, ok := inParams.param.(*ocbinds.E_OpenconfigAaaTypes_AAA_SERVER_TYPE)
	if !ok {
		return nil, fmt.Errorf("expected string pointer for server type, got %T", inParams.param)
	}

	name := inParams.key
	if (name == "tacacs+" && *serverType != ocbinds.OpenconfigAaaTypes_AAA_SERVER_TYPE_TACACS) ||
		(name == "radius" && *serverType != ocbinds.OpenconfigAaaTypes_AAA_SERVER_TYPE_RADIUS) {
		return nil, fmt.Errorf("invalid combination: name '%s' cannot be paired with type '%s'", name, *serverType)
	}
	return rmap, err
}

var DbToYang_sys_aaa_server_group_type_field_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	log.Info("DbToYang_sys_aaa_server_group_type_field_xfmr - inParams.uri ", inParams.uri)

	// Extracting path information
	pathInfo := NewPathInfo(inParams.uri)
	serverType := pathInfo.Var("type")

	if serverType == "" {
		return rmap, fmt.Errorf("server type is required")
	}

	// Use the mapping to find the corresponding OpenConfig type
	openConfigType, exists := aaa_sonicToOc_serverType[serverType]
	if !exists {
		return nil, fmt.Errorf("unknown server configuration for key: %s", serverType)
	}

	rmap["type"] = openConfigType
	log.Info("DbToYang_sys_aaa_server_group_type_field_xfmr returns ", rmap)

	return rmap, nil
}

var YangToDb_sys_aaa_server_groups_address_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	pathInfo := NewPathInfo(inParams.uri)

	log.Info("YangToDb_sys_aaa_server_groups_address_key_xfmr, inParams.uri  ", inParams.uri)

	address := pathInfo.Var("address")
	srvGrpName := pathInfo.Var("name")
	if inParams.oper == DELETE {
		tblName := ""
		if srvGrpName == "tacacs+" {
			tblName = "TACPLUS_SERVER"
		} else if srvGrpName == "radius" {
			tblName = "RADIUS_SERVER"
		}
		if address != "" {
			log.Info("YangToDb_sys_aaa_server_groups_address_key_xfmr, table, address  ", tblName, address)
			Tbl := &db.TableSpec{Name: tblName}
			entry, dbErr := inParams.d.GetEntry(Tbl, db.Key{Comp: []string{address}})
			if dbErr != nil || !entry.IsPopulated() {
				// Not returning error from here since mgmt infra will return "Resource not found" error in case of non-existence entries
				return "", tlerr.InvalidArgsError{Format: "Entry not found in table " + tblName + " with key " + address}
			}
			if inParams.subOpDataMap[DELETE] == nil {
				subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
				subIntfmap_del := make(map[string]map[string]db.Value)
				subIntfmap_del[tblName] = make(map[string]db.Value)
				subIntfmap_del[tblName][address] = db.Value{Field: map[string]string{}}
				subOpMap[db.ConfigDB] = subIntfmap_del
				inParams.subOpDataMap[DELETE] = &subOpMap
			} else {
				subOpMap := *(inParams.subOpDataMap[DELETE])
				_, ok := subOpMap[db.ConfigDB]
				if !ok {
					subIntfmap_del := make(map[string]map[string]db.Value)
					subOpMap[db.ConfigDB] = subIntfmap_del
				}
				subIntfmap_del := subOpMap[db.ConfigDB]
				//Check if the the table entry is present in subIntfmap_del
				_, ok2 := subIntfmap_del[tblName]
				if !ok2 {
					subIntfmap_del[tblName] = make(map[string]db.Value)
				}
				subIntfmap_del[tblName][address] = db.Value{Field: map[string]string{}}
			}
		}
	}

	log.Info("YangToDb_sys_aaa_server_groups_address_key_xfmr ", address)

	return address, nil
}

var DbToYang_sys_aaa_server_groups_address_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})

	serverAddress := inParams.key
	log.Info("DbToYang_sys_aaa_server_groups_address_key_xfmr ", serverAddress)
	if serverAddress == "" {
		return nil, fmt.Errorf("Ipaddress field is missing or not a string in DB data")
	}

	rmap["address"] = serverAddress
	return rmap, nil
}

var YangToDb_aaa_sys_source_address_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)

	if inParams.param == nil && inParams.oper == DELETE {
		pathInfo := NewPathInfo(inParams.uri)
		address := pathInfo.Var("address")
		name := pathInfo.Var("name")
		table := ""
		key := ""
		if name == "tacacs+" {
			table = "TACPLUS"
			key = "global"
		} else if name == "radius" {
			table = "RADIUS_SERVER"
			key = address
		}
		ServerObj, err := inParams.d.GetEntry(&db.TableSpec{Name: table}, db.Key{Comp: []string{key}})
		log.Infof("YangToDb_aaa_sys_source_address_xfmr, key: %v, table: %v", key, table)
		if err == nil || ServerObj.IsPopulated() {
			ServerMap := ServerObj.Field
			val, fieldExists := ServerMap["src_intf"]
			if fieldExists {
				log.Info("YangToDb_aaa_sys_source_address_xfmr, src_intf exists")
				res_map["src_intf"] = val
				return res_map, nil
			}
		}
		return nil, nil
	}

	if inParams.param == nil {
		return nil, invalid_input_err
	}

	pathInfo := NewPathInfo(inParams.uri)
	name := pathInfo.Var("name")
	src_ip, ok := inParams.param.(*string)
	if !ok {
		return nil, invalid_input_err
	}
	// IP address as string should have length greater than 6.
	// For ex: 1.1.1.1, the length of the string is greater than 6.
	if len(*src_ip) > 6 {
		ifName, vrfName := aaa_server_fetchVrfName_InterfaceName_From_SrcIP(*src_ip, inParams)
		if len(ifName) > 0 {
			res_map["vrf"] = vrfName
			if name == "radius" {
				res_map["src_intf"] = ifName
			} else if name == "tacacs+" {
				tacPlusTbl := &db.TableSpec{Name: "TACPLUS"}
				tacplusEntry, err := inParams.d.GetEntry(tacPlusTbl, db.Key{Comp: []string{"global"}})
				if err == nil {
					src_intf := (&tacplusEntry).Get("src_intf")
					if len(src_intf) != 0 {
						if ifName != src_intf {
							return nil, errors.New("The Entry src_intf is already set in TACPLUS for another ipaddress")
						} else { // Existing interface name is same as ifName
							return nil, nil
						}
					}
				}
				/* Save the src_intf to TACPLUS Table */
				key := db.Key{Comp: []string{"global"}}
				value := db.Value{map[string]string{"src_intf": ifName}}
				e := inParams.d.SetEntry(tacPlusTbl, key, value)
				if e != nil {
					log.Infof("The Entry src_intf is not set in TACPLUS")
					return nil, errors.New("The Entry src_intf is not set in TACPLUS")
				}
			}
		}
	} else {
		return nil, invalid_input_err
	}

	return res_map, nil
}

var DbToYang_aaa_sys_source_address_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath := pathInfo.YangPath

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}

	name := pathInfo.Var("name")
	address := pathInfo.Var("address")
	data := (*inParams.dbDataMap)[inParams.curDb]

	log.Infof("DbToYang_aaa_sys_source_address_xfmr, name: %v, inParams.table: %v, targetUriPath: %v", name, inParams.table, targetUriPath)

	if name == "tacacs+" && strings.HasPrefix(targetUriPath, "/openconfig-system:system/aaa/server-groups/server-group/servers/server/tacacs/config/source-address") {
		tacPlusTable := &db.TableSpec{Name: "TACPLUS"}
		tacPlusEntry, _ := inParams.d.GetEntry(tacPlusTable, db.Key{Comp: []string{"global"}})
		intfName := (&tacPlusEntry).Get("src_intf")
		if len(intfName) > 0 {
			tacplusTbl := data["TACPLUS_SERVER"]
			intfData := tacplusTbl[address]
			vrfName, _ := intfData.Field["vrf"]
			if len(vrfName) == 0 {
				vrfName = "default"
			}
			src_ip := get_src_addr_for_interface(intfName, vrfName, inParams)
			if len(src_ip) > 0 {
				result["source-address"] = src_ip
			} else {
				return nil, errors.New("Source address not configured properly for src_intf " + intfName)
			}
		} else {
			return nil, errors.New("Interface is not configured  " + intfName)
		}
	} else if name == "radius" && strings.HasPrefix(targetUriPath, "/openconfig-system:system/aaa/server-groups/server-group/servers/server/radius/config/source-address") {
		tbl := data["RADIUS_SERVER"]
		intfData := tbl[address]
		intfName, ok := intfData.Field["src_intf"]
		if ok && len(intfName) > 0 {
			vrfName, _ := intfData.Field["vrf"]
			if len(vrfName) == 0 {
				vrfName = "default"
			}
			src_ip := get_src_addr_for_interface(intfName, vrfName, inParams)
			if len(src_ip) > 0 {
				result["source-address"] = src_ip
			} else {
				return nil, errors.New("Source address not configured properly for src_intf " + intfName)
			}
		} else {
			return nil, errors.New("Interface is not configured  " + intfName)
		}
	}
	return result, nil
}

var sys_aaa_server_groups_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string
	pathInfo := NewPathInfo(inParams.uri)

	srvGrpName := pathInfo.Var("name")
	log.Info("sys_aaa_server_groups_table_xfmr srvGrpName ", srvGrpName)
	tacsPlusServEntries, err1 := areEntriesPresntInTable("TACPLUS_SERVER", inParams)
	radServEntries, err2 := areEntriesPresntInTable("RADIUS_SERVER", inParams)

	if srvGrpName == "" {
		if inParams.oper == GET || inParams.oper == DELETE {
			if inParams.dbDataMap != nil {
				// Traverse server-groups tacacs+ only when TACPLUS_SERVER table has entries
				// created TACPLUS_TBL temporary holder table to traverse through server-groups yang tree.
				if tacsPlusServEntries && err1 == nil {
					(*inParams.dbDataMap)[db.ConfigDB]["TACPLUS_TBL"] = make(map[string]db.Value)
					(*inParams.dbDataMap)[db.ConfigDB]["TACPLUS_TBL"]["tacacs+"] = db.Value{Field: make(map[string]string)}
					tblList = append(tblList, "TACPLUS_TBL")
				}
				// Traverse server-groups radius only when RADIUS_SERVER table has entries
				// created RADIUS_TBL temporary holder table to traverse through server-groups yang tree.
				if radServEntries && err2 == nil {
					(*inParams.dbDataMap)[db.ConfigDB]["RADIUS_TBL"] = make(map[string]db.Value)
					(*inParams.dbDataMap)[db.ConfigDB]["RADIUS_TBL"]["radius"] = db.Value{Field: make(map[string]string)}
					tblList = append(tblList, "RADIUS_TBL")
				}
			}
		}
		log.Info("sys_aaa_server_groups_table_xfmr - Server groups get operation ")
		return tblList, nil
	}

	if srvGrpName == "tacacs+" {
		if inParams.dbDataMap != nil {
			// created TACPLUS_TBL temporary holder table to traverse through server-groups yang tree.
			(*inParams.dbDataMap)[db.ConfigDB]["TACPLUS_TBL"] = make(map[string]db.Value)
			(*inParams.dbDataMap)[db.ConfigDB]["TACPLUS_TBL"]["tacacs+"] = db.Value{Field: make(map[string]string)}
			if inParams.oper != DELETE {
				(*inParams.dbDataMap)[db.ConfigDB]["TACPLUS_TBL"]["tacacs+"].Field["NULL"] = "NULL"
			}
		}
		tblList = append(tblList, "TACPLUS_TBL")
	} else if srvGrpName == "radius" {
		if inParams.dbDataMap != nil {
			// created RADIUS_TBL temporary holder table to traverse through server-groups yang tree.
			(*inParams.dbDataMap)[db.ConfigDB]["RADIUS_TBL"] = make(map[string]db.Value)
			(*inParams.dbDataMap)[db.ConfigDB]["RADIUS_TBL"]["radius"] = db.Value{Field: make(map[string]string)}
			if inParams.oper != DELETE {
				(*inParams.dbDataMap)[db.ConfigDB]["RADIUS_TBL"]["radius"].Field["NULL"] = "NULL"
			}
		}
		tblList = append(tblList, "RADIUS_TBL")
	} else {
		return tblList, fmt.Errorf("Invalid server group name: %s; must be either 'tacacs+' or 'radius'", srvGrpName)
	}
	log.Info("sys_aaa_server_groups_table_xfmr Table ", tblList)
	return tblList, nil
}

func aaa_server_fetchVrfName_InterfaceName_From_SrcIP(src_ip string, inParams XfmrParams) (string, string) {
	var ifName string
	var vrfName string
	tblList := make([]string, 0, len(intfTblList)+1)

	tblList = append(tblList, "MGMT_INTERFACE")
	tblList = append(tblList, intfTblList...)

	log.Infof("Start fetching interface for given radius/tacacs source-address ip %s ...", src_ip)
	log.Infof("Fetch Interface, table list to be looked into %v", tblList)
	for _, tblName := range tblList {
		intfTable := &db.TableSpec{Name: tblName}

		intfKeys, err := inParams.d.GetKeysPattern(intfTable, db.Key{Comp: []string{"*", src_ip}})
		if (err != nil) || len(intfKeys) == 0 {
			src_ip_w_mask := src_ip + "/*"
			intfKeys, err = inParams.d.GetKeysPattern(intfTable,
				db.Key{Comp: []string{"*", src_ip_w_mask}})
		}
		if err == nil && len(intfKeys) > 0 {
			for _, intfKey := range intfKeys {
				if len(intfKey.Comp) != 2 && len(intfKey.Comp[0]) == 0 {
					continue
				}
				ifName = intfKey.Comp[0]
				ipPrefix := intfKey.Comp[1]
				ipAddress := strings.Split(ipPrefix, "/")
				ip := strings.Split(src_ip, "/")
				if ipAddress[0] == ip[0] {
					intfEntry, err := inParams.d.GetEntry(intfTable, db.Key{Comp: []string{ifName}})

					if tblName != "MGMT_INTERFACE" {
						if err == nil {
							vrfName = (&intfEntry).Get("vrf_name")
							log.Infof("Fetch Interface, fetched vrf for interface %s is %s", ifName, vrfName)
							if vrfName == "" {
								vrfName = "default"
								break
							} else {
								/* SONiC radius and tacacs Yang models expects interface name with VRF either mgmt or default */
								vrfName = ""
								ifName = ""
							}

						}
					} else {
						mgmtVrfCfgT := &db.TableSpec{Name: "MGMT_VRF_CONFIG"}
						mgmtVrfCfgE, err := inParams.d.GetEntry(mgmtVrfCfgT, db.Key{Comp: []string{"vrf_global"}})
						vrfName = "default"
						if err == nil {
							mgmtVrfEnabled, ok := mgmtVrfCfgE.Field["mgmtVrfEnabled"]
							if ok && mgmtVrfEnabled == "true" {
								vrfName = "mgmt"
							}
						}
						break
					}
				}
			}
		}
		if ifName != "" {
			break
		}
	}
	if ifName == "" {
		log.Infof("Interface for the source-address %T not configured", inParams.param)
		return "", ""
	}
	log.Infof("Interface for the source-address %T is %s, and vrf: %s", inParams.param, ifName, vrfName)
	return ifName, vrfName
}

var sys_aaa_server_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string
	pathInfo := NewPathInfo(inParams.uri)

	srvGrpName := pathInfo.Var("name")
	log.Info("sys_aaa_server_table_xfmr srvGrpName ", srvGrpName)
	if srvGrpName == "" {
		tblList = append(tblList, "TACPLUS_SERVER")
		tblList = append(tblList, "RADIUS_SERVER")
		return tblList, nil
	}

	if srvGrpName == "tacacs+" {
		tblList = append(tblList, "TACPLUS_SERVER")
	} else if srvGrpName == "radius" {
		tblList = append(tblList, "RADIUS_SERVER")
	} else {
		return tblList, fmt.Errorf("Invalid server group name: %s; must be either 'tacacs+' or 'radius'", srvGrpName)
	}
	log.Info("sys_aaa_server_table_xfmr Table ", tblList)
	return tblList, nil
}

var YangToDb_aaa_server_secret_key_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	rmap := make(map[string]string)
	var err error

	if inParams.param == nil && inParams.oper == DELETE {
		pathInfo := NewPathInfo(inParams.uri)
		address := pathInfo.Var("address")
		name := pathInfo.Var("name")
		table := ""
		if name == "tacacs+" {
			table = "TACPLUS_SERVER"
		} else if name == "radius" {
			table = "RADIUS_SERVER"
		}
		ServerObj, err := inParams.d.GetEntry(&db.TableSpec{Name: table}, db.Key{Comp: []string{address}})
		log.Infof("YangToDb_aaa_server_secret_key_xfmr, address: %v, table: %v", address, table)
		if err == nil || ServerObj.IsPopulated() {
			ServerMap := ServerObj.Field
			val, fieldExists := ServerMap["passkey"]
			if fieldExists {
				log.Info("YangToDb_aaa_server_secret_key_xfmr, secret key exists")
				rmap["passkey"] = val
				return rmap, nil
			}
		}
		return nil, nil
	}

	if inParams.param == nil {
		log.Info("YangToDb_aaa_server_secret_key_xfmr, inParams.param nil")
		return nil, invalid_input_err
	}

	secretkey, ok := inParams.param.(*string)
	if !ok {
		log.Info("YangToDb_aaa_server_secret_key_xfmr, secretkey nil")
		return nil, invalid_input_err
	}
	rmap["passkey"] = *secretkey
	return rmap, err
}

var DbToYang_aaa_server_secret_key_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath := pathInfo.YangPath

	if inParams.dbDataMap == nil || (*inParams.dbDataMap)[inParams.curDb] == nil {
		return nil, invalid_db_err
	}
	name := pathInfo.Var("name")
	address := pathInfo.Var("address")
	data := (*inParams.dbDataMap)[inParams.curDb]
	log.Infof("DbToYang_aaa_server_secret_key_xfmr, name: %v, inParams.table: %v, targetUriPath: %v", name, inParams.table, targetUriPath)

	if name == "tacacs+" && strings.HasPrefix(targetUriPath, "/openconfig-system:system/aaa/server-groups/server-group/servers/server/tacacs/config/secret-key") {
		TacServerTbl := data["TACPLUS_SERVER"]
		entry := TacServerTbl[address]
		secretkey, ok := entry.Field["passkey"]
		if !ok {
			return nil, nil
		}
		rmap["secret-key"] = secretkey
		return rmap, nil
	}

	if name == "radius" && strings.HasPrefix(targetUriPath, "/openconfig-system:system/aaa/server-groups/server-group/servers/server/radius/config/secret-key") {
		RadiusServerTbl := data["RADIUS_SERVER"]
		entry := RadiusServerTbl[address]
		secretkey, ok := entry.Field["passkey"]
		if !ok {
			return nil, nil
		}
		rmap["secret-key"] = secretkey
		return rmap, nil
	}
	return rmap, nil
}

func areEntriesPresntInTable(tableName string, inParams XfmrParams) (bool, error) {
	tblTs := db.TableSpec{Name: tableName}
	table, err := inParams.d.GetTable(&tblTs)
	if err == nil {
		keys, err2 := table.GetKeys()
		if err2 == nil {
			if len(keys) == 0 {
				return false, nil
			} else { // Keys are present in the table
				return true, nil
			}
		} else {
			return false, err2
		}
	}
	return false, err
}

type grpcState struct {
	name           string
	certVersion    string
	certCreated    uint64
	caVersion      string
	caCreated      uint64
	crlVersion     string
	crlCreated     uint64
	authPolVersion string
	authPolCreated uint64
	profileId      string
	pathzVersion   string
	pathzCreated   uint64
}

type pathzCounters struct {
	mu      sync.Mutex
	updated map[string]time.Time
	data    map[string]map[string]map[string]*uint64
}

type policyState struct {
	instance ocbinds.E_OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance
	version  string
	created  uint64
}

var dbToYangPathzInstanceMap = map[string]ocbinds.E_OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance{
	"ACTIVE":  ocbinds.OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance_ACTIVE,
	"SANDBOX": ocbinds.OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance_SANDBOX,
}

func getAppRootObject(inParams XfmrParams) *ocbinds.OpenconfigSystem_System {
	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	return deviceObj.System
}

func getAllKeys(sdb *db.DB, tblName string) ([]string, error) {
	tbl, err := sdb.GetTable(&db.TableSpec{Name: tblName})
	if err != nil {
		return nil, fmt.Errorf("Can't get table: %v, err: %v", tblName, err)
	}
	log.V(3).Infof("tbl: %v", tbl)
	keys, err := tbl.GetKeys()
	if err != nil {
		return nil, fmt.Errorf("Can't get keys from %v, err: %v", tblName, err)
	}
	log.V(3).Infof("tbl keys: %v", keys)
	ret := []string{}
	for _, key := range keys {
		if len(key.Comp) != 3 {
			// This is a phantom key. Ignore it.
			continue
		}
		ret = append(ret, key.Comp[2])
	}
	log.V(3).Infof("keys: %v", ret)
	return ret, nil
}

var Subscribe_ssh_server_state_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	log.V(3).Infof("Subscribe_ssh_server_state_xfmr:%s", inParams.requestURI)

	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"SSH_HOST": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}
var Subscribe_authz_policy_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	log.V(3).Infof("Subscribe_authz_policy_xfmr:%s", inParams.requestURI)
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"AUTHZ_POLICY|gnxi": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

var DbToYang_ssh_server_state_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var state sshState

	table, err := inParams.dbs[inParams.curDb].GetEntry(&db.TableSpec{Name: "CREDENTIALS"}, db.Key{Comp: []string{"SSH_HOST"}})
	if err != nil {
		log.V(3).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	state.caKeys.version = table.Get("ca_keys_version")
	time := table.Get("ca_keys_created_on")
	if state.caKeys.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(0).Infof("Couldn't find ca_keys_created_on: %v", err)
	}
	state.hostKey.version = table.Get("host_key_version")
	time = table.Get("host_key_created_on")
	if state.hostKey.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(0).Infof("Couldn't find host_key_created_on: %v", err)
	}
	state.hostCert.version = table.Get("host_cert_version")
	time = table.Get("host_cert_created_on")
	if state.hostCert.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(0).Infof("Couldn't find host_cert_created_on: %v", err)
	}
	accepts := table.Get("access_accepts")
	if state.counters.accessAccepts, err = strconv.ParseUint(accepts, 10, 64); err != nil && accepts != "" {
		log.V(0).Infof("Couldn't find access_accepts: %v", err)
	}
	lastAccept := table.Get("last_access_accept")
	if state.counters.lastAccessAccept, err = strconv.ParseUint(lastAccept, 10, 64); err != nil && lastAccept != "" {
		log.V(0).Infof("Couldn't find last_access_accept: %v", err)
	}
	rejects := table.Get("access_rejects")
	if state.counters.accessRejects, err = strconv.ParseUint(rejects, 10, 64); err != nil && rejects != "" {
		log.V(0).Infof("Couldn't find access_rejects: %v", err)
	}
	lastReject := table.Get("last_access_reject")
	if state.counters.lastAccessReject, err = strconv.ParseUint(lastReject, 10, 64); err != nil && lastReject != "" {
		log.V(0).Infof("Couldn't find last_access_reject: %v", err)
	}

	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj.SshServer.State)

	sysObj.SshServer.State.ActiveTrustedUserCaKeysCreatedOn = &state.caKeys.created
	sysObj.SshServer.State.ActiveTrustedUserCaKeysVersion = &state.caKeys.version
	sysObj.SshServer.State.ActiveHostCertificateCreatedOn = &state.hostKey.created
	sysObj.SshServer.State.ActiveHostCertificateVersion = &state.hostKey.version
	sysObj.SshServer.State.ActiveHostKeyCreatedOn = &state.hostCert.created
	sysObj.SshServer.State.ActiveHostKeyVersion = &state.hostCert.version
	sysObj.SshServer.State.Counters.AccessAccepts = &state.counters.accessAccepts
	sysObj.SshServer.State.Counters.AccessRejects = &state.counters.accessRejects
	sysObj.SshServer.State.Counters.LastAccessAccept = &state.counters.lastAccessAccept
	sysObj.SshServer.State.Counters.LastAccessReject = &state.counters.lastAccessReject
	return nil
}
var DbToYang_authz_policy_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var state certData

	table, err := inParams.dbs[inParams.curDb].GetEntry(&db.TableSpec{Name: CRED_AUTHZ_TBL}, db.Key{Comp: []string{GNXI_ID}})
	if err != nil {
		log.V(3).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	state.version = table.Get("authz_version")
	time := table.Get("authz_created_on")
	if state.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(3).Infof("Couldn't find authz_created_on: %v", err)
	}

	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj.Aaa.Authorization.State)

	sysObj.Aaa.Authorization.State.GrpcAuthzPolicyCreatedOn = &state.created
	sysObj.Aaa.Authorization.State.GrpcAuthzPolicyVersion = &state.version

	return nil
}

func (m *pathzCounters) getCounters(pathzTables db.Table, xpath string) map[string]map[string]*uint64 {
	result := make(map[string]map[string]*uint64)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updated == nil || m.data == nil {
		m.updated = make(map[string]time.Time)
		m.data = make(map[string]map[string]map[string]*uint64)
	}

	// Update the map if necessary
	updateTime, ok := m.updated[xpath]
	if !ok {
		result = GetPathzPolicyCounter(pathzTables, xpath)
		if len(m.data) < 50 {
			m.data[xpath] = result
			m.updated[xpath] = time.Now()
		}
	} else if time.Now().After(updateTime.Add(30 * time.Second)) {
		m.data[xpath] = GetPathzPolicyCounter(pathzTables, xpath)
		m.updated[xpath] = time.Now()
	}

	// Fetch the result or return the previously calculated result
	if data, ok := m.data[xpath]; ok {
		result = data
	}
	return result
}

func GetPathzPolicyCounter(pathzTables db.Table, path string) map[string]map[string]*uint64 {
	cntMap := make(map[string]*uint64)
	tsMap := make(map[string]*uint64)

	for _, tmp := range pathzOpers {
		pattern := PatternGenerator(tmp, path)
		if pattern == "" {
			log.V(3).Infof("Invalid pathz counter key pattern.")
			continue
		}
		key := db.NewKey(tmp[0], path, tmp[1])

		// Sum the data collected
		value, err := pathzTables.GetEntry(*key)
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("Cannot get value from %v table for %v, err: %v", PATHZ_TBL, key, err)
			continue
		}

		c := value.Get("count")
		if c == "" {
			continue
		}
		dbCnt, err := strconv.ParseUint(c, 10, 64)
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("Failed to convert counters from DB for pathz, err: %v", err)
			continue
		}
		tsval := value.Get("timestamp")
		if tsval == "" {
			continue
		}
		dbTs, err := strconv.ParseUint(tsval, 10, 64)
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("Failed to convert timestamp for counters from DB for pathz, err: %v", err)
			continue
		}

		cnt, cntExists := cntMap[pattern]
		if cntExists && cnt != nil {
			cntUpdate, err := strconv.ParseUint(strconv.FormatUint((*cnt+dbCnt), 10), 10, 64)
			if err != nil {
				log.V(tlerr.ErrorSeverity(err)).Infof("Failed to convert counters for pathz, err: %v", err)
				continue
			}
			cntMap[pattern] = &cntUpdate
		} else {
			cntMap[pattern] = &dbCnt
		}

		ts, tsExists := tsMap[pattern]
		if !tsExists || ts == nil || *ts < dbTs {
			tsMap[pattern] = &dbTs
		}
	}
	return map[string]map[string]*uint64{cntResult: cntMap, tsResult: tsMap}
}

func getAllXpaths(pathzTables db.Table) ([]string, error) {
	var res []string
	check := make(map[string]bool)
	pathzTableKeys, err := pathzTables.GetKeys()
	if err != nil {
		log.V(tlerr.ErrorSeverity(err)).Infof("Cannot get all keys from %v table, err: %v", PATHZ_TBL, err)
		return []string{}, err
	}
	for _, pathzTableKey := range pathzTableKeys {
		if len(pathzTableKey.Comp) != 3 {
			log.V(3).Infof("invalid number of Comps for pathzTableKey %v.", pathzTableKey)
			continue
		}
		if pathzTableKey.Comp[1] != "" {
			key := pathzTableKey.Comp[1]
			if val, ok := check[key]; !ok || !val {
				res = append(res, key)
				check[key] = true
			}
		}
	}

	return res, nil
}

var pathToPatternKeysMap = map[string][]string{
	PATHZ_READ_SUCCESS:            []string{"reads", ACCEPTS},
	PATHZ_READ_SUCCESS_TIMESTAMP:  []string{"reads", ACCEPTS},
	PATHZ_READ_FAILED:             []string{"reads", REJECTS},
	PATHZ_READ_FAILED_TIMESTAMP:   []string{"reads", REJECTS},
	PATHZ_WRITE_SUCCESS:           []string{"writes", ACCEPTS},
	PATHZ_WRITE_SUCCESS_TIMESTAMP: []string{"writes", ACCEPTS},
	PATHZ_WRITE_FAILED:            []string{"writes", REJECTS},
	PATHZ_WRITE_FAILED_TIMESTAMP:  []string{"writes", REJECTS},
}

func PatternGenerator(params []string, xpath string) string {
	if len(params) != 2 {
		log.V(3).Infof("Invalid params for patternGenerator %#v", params)
		return ""
	}

	if params[0] == READS_GET || params[0] == READS_SUB || params[0] == "reads" {
		return "*|reads|" + xpath + "|" + params[1]
	}

	if params[0] == WRITES || params[0] == "writes" {
		return "*|writes|" + xpath + "|" + params[1]
	}

	log.V(3).Infof("Invalid operation %v", params[0])
	return ""
}

var Subscribe_grpc_server_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	serverName := pathInfo.Var("name")
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return XfmrSubscOutParams{}, err
	}
	log.V(3).Infof("Subscribe_grpc_server_xfmr: targetUriPath: %s name: %s", targetUriPath, serverName)

	var result XfmrSubscOutParams
	if serverName == "" {
		result.dbDataMap = RedisDbSubscribeMap{
			db.StateDB: map[string]map[string]map[string]string{
				CREDENTIALS_TBL: {
					"CERT|gnxi":           {},
					"PATHZ_POLICY|ACTIVE": {}},
			},
		}
	} else {
		result = XfmrSubscOutParams{
			dbDataMap: RedisDbSubscribeMap{
				db.StateDB: map[string]map[string]map[string]string{
					CREDENTIALS_TBL: {
						"CERT|gnxi":           {},
						"PATHZ_POLICY|ACTIVE": {}},
				}},
		}
	}

	if !strings.HasPrefix(targetUriPath, "/openconfig-system:system/grpc-servers/grpc-server/gnsi-pathz:gnmi-pathz-policy-counters") {
		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{mInterval: 0, pType: OnChange}
	} else {

		// For counters, configure nOpts to enable sampling on path.
		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{mInterval: 60, pType: Sample}
	}

	return result, nil
}
var DbToYang_pathz_policies_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	instances := []string{pathInfo.Var("instance")}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(3).Infof("DbToYang_pathz_policies_xfmr: targetUriPath: %s instances: %v", targetUriPath, instances)

	stateDb := inParams.dbs[db.StateDB]
	if len(instances) == 0 || len(instances[0]) == 0 {
		var err error
		if instances, err = getAllKeys(stateDb, CRED_PATHZ_TBL); err != nil {
			return err
		}
	}
	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.GnmiPathzPolicies)
	ygot.BuildEmptyTree(sysObj.GnmiPathzPolicies.Policies)

	for _, instance := range instances {
		log.V(3).Infof("instance: %v", instance)
		i, ok := dbToYangPathzInstanceMap[instance]
		if !ok {
			log.V(0).Infof("Pathz Policy Instance not found: %v", instance)
			continue
		}
		policyObj, ok := sysObj.GnmiPathzPolicies.Policies.Policy[i]
		if !ok {
			var err error
			policyObj, err = sysObj.GnmiPathzPolicies.Policies.NewPolicy(i)
			if err != nil {
				log.V(0).Infof("sysObj.GnmiPathzPolicies.Policies.NewPolicy failed: %v", err)
				continue
			}
		}
		table, err := stateDb.GetEntry(&db.TableSpec{Name: CRED_PATHZ_TBL}, db.Key{Comp: []string{instance}})
		if err != nil {
			log.V(0).Infof("Failed to read from StateDB %v, id: %v, err: %v", inParams.table, instance, err)
			return err
		}
		var state policyState

		state.instance = i
		state.version = table.Get("pathz_version")
		time := table.Get("pathz_created_on")
		if state.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
			return err
		}
		ygot.BuildEmptyTree(policyObj)
		policyObj.State.Instance = state.instance
		policyObj.State.CreatedOn = &state.created
		policyObj.State.Version = &state.version
	}
	return nil
}
var DbToYang_grpc_server_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(3).Info("DbToYang_grpc_server_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)

	return map[string]interface{}{"name": NewPathInfo(inParams.uri).Var("name")}, nil
}

var DbToYang_grpc_server_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	serverNames := []string{pathInfo.Var("name")}
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		log.V(0).Infof("Error Parsing Uri Path, err: %v", err)
	}
	if log.V(3) {
		log.Info("SubtreeXfmrFunc - Uri SYS AUTH: ", inParams.uri)
		log.Info("TARGET URI PATH SYS AUTH:", targetUriPath)
		log.Info("names:", serverNames)
	}
	stateDb := inParams.dbs[db.StateDB]
	if stateDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr stateDb is nil!")
	}
	if len(serverNames) == 0 || len(serverNames[0]) == 0 {
		var err error
		if serverNames, err = getAllKeys(stateDb, CERT_TBL); err != nil {
			return err
		}
	}
	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.GrpcServers)

	for _, serverName := range serverNames {
		log.V(3).Info("serverName: ", serverName)
		var state grpcState
		state.name = serverName

		certzID := GNXI_ID
		certTable, err := stateDb.GetEntry(&db.TableSpec{Name: CERT_TBL}, db.Key{Comp: []string{certzID}})
		if err != nil {
			log.V(0).Infof("Failed to read from StateDB %v | %v err: %v", CERT_TBL, certzID, err)
		} else {
			state.certVersion = certTable.Get("certificate_version")
			state.caVersion = certTable.Get("ca_trust_bundle_version")
			state.crlVersion = certTable.Get("certificate_revocation_list_bundle_version")
			state.authPolVersion = certTable.Get("authentication_policy_version")
			state.profileId = certTable.Get("ssl_profile_id")
			time := certTable.Get("certificate_created_on")
			if state.certCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(0).Infof("Cannot convert `certificate_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("ca_trust_bundle_created_on")
			if state.caCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(0).Infof("Cannot convert `ca_trust_bundle_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("certificate_revocation_list_bundle_created_on")
			if state.crlCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(0).Infof("Cannot convert `certificate_revocation_list_bundle_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("authentication_policy_created_on")
			if state.authPolCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(0).Infof("Cannot convert `authentication_policy_created_on` for %v, err: %v", certzID, err)
			}
		}

		pathzTable, err := stateDb.GetEntry(&db.TableSpec{Name: CRED_PATHZ_TBL}, db.Key{Comp: []string{"ACTIVE"}})
		if err != nil {
			log.V(0).Infof("Failed to read from StateDB %v, err: %v", CRED_PATHZ_TBL, err)
		} else {
			state.pathzVersion = pathzTable.Get("pathz_version")
			if timeStr := pathzTable.Get("pathz_created_on"); timeStr != "" {
				if state.pathzCreated, err = strconv.ParseUint(timeStr, 10, 64); err != nil {
					log.V(0).Infof("Cannot convert `pathz_created_on` for %v, err: %v", serverName, err)
				}
			}
		}
		serverObj, ok := sysObj.GrpcServers.GrpcServer[serverName]
		if !ok {
			serverObj, err = sysObj.GrpcServers.NewGrpcServer(serverName)
			if err != nil {
				log.V(0).Infof("sysObj.GrpcServers.NewGrpcServer(%v) failed: %v", serverName, err)
				continue
			}
		}
		ygot.BuildEmptyTree(serverObj)
		serverObj.State.Name = &state.name
		serverObj.State.CaTrustBundleVersion = &state.caVersion
		serverObj.State.CaTrustBundleCreatedOn = &state.caCreated
		serverObj.State.CertificateVersion = &state.certVersion
		serverObj.State.CertificateCreatedOn = &state.certCreated
		serverObj.State.CertificateRevocationListBundleCreatedOn = &state.crlCreated
		serverObj.State.CertificateRevocationListBundleVersion = &state.crlVersion
		serverObj.State.AuthenticationPolicyVersion = &state.authPolVersion
		serverObj.State.SslProfileId = &state.profileId
		serverObj.State.AuthenticationPolicyCreatedOn = &state.authPolCreated
		serverObj.State.GnmiPathzPolicyCreatedOn = &state.pathzCreated
		serverObj.State.GnmiPathzPolicyVersion = &state.pathzVersion

		// Authz counter
		authzTables, err := stateDb.GetTable(&db.TableSpec{Name: AUTHZ_TBL})
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("getAuthzPolicyCounter failed to get AUTHZ_TBL, err: %v", err)
			return err
		}

		rpcString := pathInfo.Var("name#2")
		rpcStrings := []string{rpcString}

		if rpcString == "" || rpcString == "*" {
			rpcStrings = []string{}
			rpcStrings, err = getAllRpcs(authzTables, serverName)
			if err != nil {
				log.V(tlerr.ErrorSeverity(err)).Infof("Failed get all authz rpcs, err: %v", err)
				return err
			}
		}

		ygot.BuildEmptyTree(serverObj.AuthzPolicyCounters)
		for _, rpcString := range rpcStrings {
			service, rpc, err := getServiceRpc(rpcString)
			if err != nil {
				log.V(0).Infof("invalid RPC method %s", rpcString)
				continue
			}

			authzPolicyData := getAuthzPolicyCounter(authzTables, serverName, rpcString)
			rpcObj, ok := serverObj.AuthzPolicyCounters.Rpcs.Rpc[rpcString]
			if !ok {
				rpcObj, err = serverObj.AuthzPolicyCounters.Rpcs.NewRpc(rpcString)
				if err != nil {
					log.V(0).Infof("serverObj.AuthzPolicyCounters.Rpcs.NewRpc(%v) failed: %v", rpcString, err)
					continue
				}
			}
			ygot.BuildEmptyTree(rpcObj)

			// If targetUriPath is a parent AUTHZ_STATE, i.e.root path, all counters and timestamps should be returned
			allAuthzCounter := strings.HasPrefix(AUTHZ_STATE, targetUriPath) || targetUriPath == GRPC_OC_SERVERS

			tmpCnt := make(map[string]*uint64)
			tmpTs := make(map[string]*uint64)
			if cnt, ok := authzPolicyData[cntResult]; ok {
				tmpCnt = cnt
			}
			if ts, ok := authzPolicyData[tsResult]; ok {
				tmpTs = ts
			}
			// Handle root paths here.
			if allAuthzCounter {
				ygot.BuildEmptyTree(rpcObj.State)
				rpcObj.State.AccessAccepts = tmpCnt["*|"+serverName+"|"+service+"|"+rpc+"|"+ACCEPTS]
				rpcObj.State.LastAccessAccept = tmpTs["*|"+serverName+"|"+service+"|"+rpc+"|"+ACCEPTS]
				rpcObj.State.AccessRejects = tmpCnt["*|"+serverName+"|"+service+"|"+rpc+"|"+REJECTS]
				rpcObj.State.LastAccessReject = tmpTs["*|"+serverName+"|"+service+"|"+rpc+"|"+REJECTS]

			} else {
				// Handle leaf paths here.
				switch targetUriPath {
				case AUTHZ_SUCCESS:
					rpcObj.State.AccessAccepts = tmpCnt["*|"+serverName+"|"+service+"|"+rpc+"|"+ACCEPTS]
				case AUTHZ_SUCCESS_TIMESTAMP:
					rpcObj.State.LastAccessAccept = tmpTs["*|"+serverName+"|"+service+"|"+rpc+"|"+ACCEPTS]
				case AUTHZ_FAILED:
					rpcObj.State.AccessRejects = tmpCnt["*|"+serverName+"|"+service+"|"+rpc+"|"+REJECTS]
				case AUTHZ_FAILED_TIMESTAMP:
					rpcObj.State.LastAccessReject = tmpTs["*|"+serverName+"|"+service+"|"+rpc+"|"+REJECTS]
				}
			}
		}
		// Pathz counter is for GNXI_ID only
		if serverName != GNXI_ID {
			continue
		}

		// Pathz counter
		pathzTables, err := stateDb.GetTable(&db.TableSpec{Name: PATHZ_TBL})
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("getPathzPolicyCounter failed to get PATHZ_TBL, err: %v", err)
		}

		xpath := pathInfo.Var("xpath")
		xpaths := []string{xpath}

		if xpath == "" || xpath == "*" {
			xpaths = []string{}
			xpaths, err = getAllXpaths(pathzTables)
			if err != nil {
				log.V(tlerr.ErrorSeverity(err)).Infof("Failed get all paths, err: %v", err)
			}
		}

		ygot.BuildEmptyTree(serverObj.GnmiPathzPolicyCounters)
		for _, xpath := range xpaths {
			// Processing these counters is hard on the CPU. We will only update these counters every 30 seconds.
			pathzPolicyData := pathzMap.getCounters(pathzTables, xpath)

			pathObj, ok := serverObj.GnmiPathzPolicyCounters.Paths.Path[xpath]
			if !ok {
				pathObj, err = serverObj.GnmiPathzPolicyCounters.Paths.NewPath(xpath)
				if err != nil {
					log.V(0).Infof("serverObj.GnmiPathzPolicyCounters.NewPath(%v) failed: %v", xpath, err)
					continue
				}
			}
			ygot.BuildEmptyTree(pathObj)

			// If targetUriPath is a parent PATHZ_STATE, i.e.root path, all counters and timestamps should be returned
			allPathzCounter := strings.HasPrefix(PATHZ_STATE, targetUriPath) || targetUriPath == GRPC_OC_SERVERS

			tmpCnt := make(map[string]*uint64)
			tmpTs := make(map[string]*uint64)
			if cnt, ok := pathzPolicyData[cntResult]; ok {
				tmpCnt = cnt
			}
			if ts, ok := pathzPolicyData[tsResult]; ok {
				tmpTs = ts
			}

			// Handle root paths here.
			if allPathzCounter || targetUriPath == PATHZ_READS || targetUriPath == PATHZ_WRITES {
				ygot.BuildEmptyTree(pathObj.State)
				if allPathzCounter || targetUriPath == PATHZ_READS {
					pathObj.State.Reads.AccessAccepts = tmpCnt[PatternGenerator(pathToPatternKeysMap[PATHZ_READ_SUCCESS], xpath)]
					pathObj.State.Reads.LastAccessAccept = tmpTs[PatternGenerator(pathToPatternKeysMap[PATHZ_READ_SUCCESS_TIMESTAMP], xpath)]
					pathObj.State.Reads.AccessRejects = tmpCnt[PatternGenerator(pathToPatternKeysMap[PATHZ_READ_FAILED], xpath)]
					pathObj.State.Reads.LastAccessReject = tmpTs[PatternGenerator(pathToPatternKeysMap[PATHZ_READ_FAILED_TIMESTAMP], xpath)]
				}
				if allPathzCounter || targetUriPath == PATHZ_WRITES {
					pathObj.State.Writes.AccessAccepts = tmpCnt[PatternGenerator(pathToPatternKeysMap[PATHZ_WRITE_SUCCESS], xpath)]
					pathObj.State.Writes.LastAccessAccept = tmpTs[PatternGenerator(pathToPatternKeysMap[PATHZ_WRITE_SUCCESS_TIMESTAMP], xpath)]
					pathObj.State.Writes.AccessRejects = tmpCnt[PatternGenerator(pathToPatternKeysMap[PATHZ_WRITE_FAILED], xpath)]
					pathObj.State.Writes.LastAccessReject = tmpTs[PatternGenerator(pathToPatternKeysMap[PATHZ_WRITE_FAILED_TIMESTAMP], xpath)]
				}
			} else {
				// Handle leaf paths here.
				patternKeys := pathToPatternKeysMap[targetUriPath]
				if patternKeys == nil {
					log.V(0).Infof("Invalid pathz table key: %#v", targetUriPath)
					continue
				}
				pattern := PatternGenerator([]string{patternKeys[0], patternKeys[1]}, xpath)

				switch targetUriPath {
				case PATHZ_READ_SUCCESS:
					pathObj.State.Reads.AccessAccepts = tmpCnt[pattern]
				case PATHZ_READ_SUCCESS_TIMESTAMP:
					pathObj.State.Reads.LastAccessAccept = tmpTs[pattern]
				case PATHZ_READ_FAILED:
					pathObj.State.Reads.AccessRejects = tmpCnt[pattern]
				case PATHZ_READ_FAILED_TIMESTAMP:
					pathObj.State.Reads.LastAccessReject = tmpTs[pattern]
				case PATHZ_WRITE_SUCCESS:
					pathObj.State.Writes.AccessAccepts = tmpCnt[pattern]
				case PATHZ_WRITE_SUCCESS_TIMESTAMP:
					pathObj.State.Writes.LastAccessAccept = tmpTs[pattern]
				case PATHZ_WRITE_FAILED:
					pathObj.State.Writes.AccessRejects = tmpCnt[pattern]
				case PATHZ_WRITE_FAILED_TIMESTAMP:
					pathObj.State.Writes.LastAccessReject = tmpTs[pattern]
				}
			}
		}
	}
	return nil
}
var DbToYang_pathz_policies_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(3).Info("DbToYang_pathz_policies_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)
	return map[string]interface{}{"instance": NewPathInfo(inParams.uri).Var("instance")}, nil
}

func getAuthzPolicyCounter(authzTables db.Table, server string, rpcString string) map[string]map[string]*uint64 {
	cntMap := make(map[string]*uint64)
	tsMap := make(map[string]*uint64)

	for _, oper := range []string{ACCEPTS, REJECTS} {
		var service string
		var rpc string
		service, rpc, err := getServiceRpc(rpcString)
		if err != nil {
			log.V(0).Infof("invalid RPC method %s", rpcString)
			continue
		}

		pattern := "*|" + server + "|" + service + "|" + rpc + "|" + oper
		key := db.NewKey(server, service, rpc, oper)

		// Sum the data collected
		value, err := authzTables.GetEntry(*key)
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Infof("Cannot get value from %v table for %v, err: %v", AUTHZ_TBL, key, err)
			continue
		}

		c := value.Get("count")
		if c != "" {
			if dbCnt, err := strconv.ParseUint(c, 10, 64); err == nil {
				cntMap[pattern] = &dbCnt
			} else {
				log.V(tlerr.ErrorSeverity(err)).Infof("Failed to convert counters from DB for authz, err: %v", err)
			}
		}

		ts := value.Get("timestamp")
		if ts != "" {
			if dbTs, err := strconv.ParseUint(ts, 10, 64); err == nil {
				tsMap[pattern] = &dbTs
			} else {
				log.V(tlerr.ErrorSeverity(err)).Infof("Failed to convert timestamp for counters from DB for authz, err: %v", err)
			}
		}
	}
	return map[string]map[string]*uint64{cntResult: cntMap, tsResult: tsMap}
}

func getServiceRpc(rpcString string) (string, string, error) {
	strs := strings.Split(rpcString, "/")
	if len(strs) == 3 {
		return strs[1], strs[2], nil
	}

	return "", "", errors.New("invalid RPC method " + rpcString)
}

func getAllRpcs(authzTables db.Table, server string) ([]string, error) {
	var res []string
	check := make(map[string]bool)
	authzTableKeys, err := authzTables.GetKeys()
	if err != nil {
		log.V(tlerr.ErrorSeverity(err)).Infof("Cannot get all keys from %v table, err: %v", AUTHZ_TBL, err)
		return []string{}, err
	}
	for _, authzTableKey := range authzTableKeys {
		if len(authzTableKey.Comp) != 4 {
			log.V(3).Infof("invalid number of Comps for authzTableKey %v.", authzTableKey)
			continue
		}
		if authzTableKey.Comp[0] != server {
			continue
		}
		key := "/" + authzTableKey.Comp[1] + "/" + authzTableKey.Comp[2]
		if val, ok := check[key]; !ok || !val {
			res = append(res, key)
			check[key] = true
		}
	}

	return res, nil
}

var Subscribe_pathz_policies_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	instance := pathInfo.Var("instance")
	if instance == "" {
		instance = "*"
	}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(3).Infof("Subscribe_pathz_policies_xfmr: targetUriPath: %s instance: %s", targetUriPath, instance)

	key := strings.Join([]string{"PATHZ_POLICY", instance}, "|")
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {CREDENTIALS_TBL: {key: {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

var DbToYang_console_counters_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var counters accessCounters

	table, err := inParams.dbs[inParams.curDb].GetEntry(&db.TableSpec{Name: "CREDENTIALS"}, db.Key{Comp: []string{"CONSOLE_METRICS"}})
	if err != nil {
		log.V(0).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	accepts := table.Get("access_accepts")
	if counters.accessAccepts, err = strconv.ParseUint(accepts, 10, 64); err != nil && accepts != "" {
		log.V(0).Infof("Couldn't find access_accepts: %v", err)
	}
	lastAccept := table.Get("last_access_accept")
	if counters.lastAccessAccept, err = strconv.ParseUint(lastAccept, 10, 64); err != nil && lastAccept != "" {
		log.V(0).Infof("Couldn't find last_access_accept: %v", err)
	}
	rejects := table.Get("access_rejects")
	if counters.accessRejects, err = strconv.ParseUint(rejects, 10, 64); err != nil && rejects != "" {
		log.V(0).Infof("Couldn't find access_rejects: %v", err)
	}
	lastReject := table.Get("last_access_reject")
	if counters.lastAccessReject, err = strconv.ParseUint(lastReject, 10, 64); err != nil && lastReject != "" {
		log.V(0).Infof("Couldn't find last_access_reject: %v", err)
	}

	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.Console)
	ygot.BuildEmptyTree(sysObj.Console.State)

	sysObj.Console.State.Counters.AccessAccepts = &counters.accessAccepts
	sysObj.Console.State.Counters.AccessRejects = &counters.accessRejects
	sysObj.Console.State.Counters.LastAccessAccept = &counters.lastAccessAccept
	sysObj.Console.State.Counters.LastAccessReject = &counters.lastAccessReject

	return nil
}

var Subscribe_console_counters_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	log.V(0).Infof("Subscribe_console_counters_xfmr:%s", inParams.requestURI)

	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"CONSOLE_METRICS": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}
