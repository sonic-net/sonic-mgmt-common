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

var aaa_ocToSonic_serverType = map[ocbinds.E_OpenconfigAaaTypes_AAA_SERVER_TYPE]string{
	ocbinds.OpenconfigAaaTypes_AAA_SERVER_TYPE_RADIUS: "RADIUS_SERVER",
	ocbinds.OpenconfigAaaTypes_AAA_SERVER_TYPE_TACACS: "TACPLUS_SERVER",
}

var aaa_ocStrToSonic_serverType = map[string]string{
	"RADIUS": "RADIUS_SERVER",
	"TACACS": "TACPLUS_SERVER",
}

var aaa_sonicToOc_serverType = map[string]string{
	"RADIUS_SERVER":  "RADIUS",
	"TACPLUS_SERVER": "TACACS",
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
	tblList := make([]string, len(intfTblList)+1)

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
			copy(tblList, intfTblList)
		}
	} else {
		copy(tblList, intfTblList)
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
	tblList := make([]string, len(intfTblList)+1)

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
			copy(tblList, intfTblList)
		}
	} else {
		copy(tblList, intfTblList)
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

// modified but where should the name come frome??
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
		log.Infof("YangToDb_aaa_sys_source_address_xfmr, key, table", key, table)
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

	log.Infof("DbToYang_aaa_sys_source_address_xfmr, name, inParams.table, targetUriPath", name, inParams.table, targetUriPath)

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
	//	}
	log.Info("sys_aaa_server_groups_table_xfmr Table ", tblList)
	return tblList, nil
}

func aaa_server_fetchVrfName_InterfaceName_From_SrcIP(src_ip string, inParams XfmrParams) (string, string) {
	var ifName string
	var vrfName string
	tblList := make([]string, len(intfTblList)+1)

	tblList = append(tblList, "MGMT_INTERFACE")
	copy(tblList, intfTblList)

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
		log.Infof("YangToDb_aaa_server_secret_key_xfmr, address, table", address, table)
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
	log.Infof("DbToYang_aaa_server_secret_key_xfmr, name, inParams.table, targetUriPath", name, inParams.table, targetUriPath)

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
