package transformer

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	lvl "github.com/Azure/sonic-mgmt-common/translib/log"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	ygot "github.com/openconfig/ygot/ygot"

	// TODO: replace with crypto/sha512 standard library
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

const (
	HOST_TBL              = "HOST_STATS"
	MEM_TBL               = "MEM_STATS"
	MEMORY_TBL            = "MEMORY_STATS"
	CPU_TBL               = "CPU_STATS"
	FEATURE_LABELS_TBL    = "FEATURE_LABELS"
	PROC_TBL              = "PROCESS_STATS"
	MOUNT_POINTS_TBL      = "MOUNT_POINTS"
	VERIFY_STATE_RESP_TBL = "VERIFY_STATE_RESP_TABLE"
	LOGGING_TBL           = "SYSLOG_SERVER"
	SYSMEM_KEY            = "SYS_MEM"
	HOSTNAME_KEY          = "HOSTNAME"
	HOSTCONFIG_KEY        = "CONFIG"
	CHARSET               = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ALARM_TBL             = "COMPONENT_STATE_TABLE"
	PATHZ_TBL             = "PATHZ_TABLE"
	AUTHZ_TBL             = "AUTHZ_TABLE"
	BOOT_INFO_TBL         = "BOOT_INFO"
	READS_GET             = "get"
	READS_SUB             = "subscribe"
	WRITES                = "set"
	ACCEPTS               = "permitted"
	REJECTS               = "denied"
	GNXI_ID               = "gnxi"
	GNPSI_ID              = "gnpsi"
	cntResult             = "cntResult"
	tsResult              = "tsResult"
	systemKey             = "system"

	/** Credential Tables **/
	CREDENTIALS_TBL = "CREDENTIALS"
	ACCOUNT_TBL     = "CREDENTIALS|SSH_ACCOUNT"
	CRED_AUTHZ_TBL  = "CREDENTIALS|AUTHZ_POLICY"
	CERT_TBL        = "CREDENTIALS|CERT"
	CONSOLE_TBL     = "CREDENTIALS|CONSOLE_ACCOUNT"
	CRED_PATHZ_TBL  = "CREDENTIALS|PATHZ_POLICY"
	SSH_TBL         = "CREDENTIALS|SSH_HOST"

	CHASSIS_TBL    = "CHASSIS_INFO"
	CHASSIS_PREFIX = "chassis"

	/** 01/02/2006 15:04:05 format copied from GO official doc **/
	BASE_TIME_FORMAT = "01/02/2006 15:04:05 -0700 MST"

	/** System Root paths **/
	SYSTEM_ROOT = "/openconfig-system:system"
	/** Supported system alarm state URIs **/
	ALARM_ROOT             = SYSTEM_ROOT + "/alarms/alarm"
	ALRM_STATE             = SYSTEM_ROOT + "/alarms/alarm/state"
	ALRM_STATE_ID          = SYSTEM_ROOT + "/alarms/alarm/state/id"
	ALRM_STATE_RESOURCE    = SYSTEM_ROOT + "/alarms/alarm/state/resource"
	ALRM_STATE_SEVERITY    = SYSTEM_ROOT + "/alarms/alarm/state/severity"
	ALRM_STATE_TEXT        = SYSTEM_ROOT + "/alarms/alarm/state/text"
	ALRM_STATE_TIMECREATED = SYSTEM_ROOT + "/alarms/alarm/state/time-created"
	ALRM_STATE_TYPEID      = SYSTEM_ROOT + "/alarms/alarm/state/type-id"

	/** Pathz paths **/
	GRPC_OC_SERVERS       = SYSTEM_ROOT + "/openconfig-system-grpc:grpc-servers"
	GRPC_SERVERS          = SYSTEM_ROOT + "/grpc-servers"
	GRPC_SERVER           = GRPC_OC_SERVERS + "/grpc-server"
	PATHZ_POLICY_COUNTERS = GRPC_SERVER + "/gnsi-pathz:gnmi-pathz-policy-counters"
	ALL_PATHZ             = PATHZ_POLICY_COUNTERS + "/paths"
	SINGLE_PATHZ          = ALL_PATHZ + "/path"

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

	/** Authz paths **/
	AUTHZ_POLICY_COUNTERS   = GRPC_SERVER + "/authz-policy-counters"
	ALL_AUTHZ               = AUTHZ_POLICY_COUNTERS + "/rpcs"
	SINGLE_AUTHZ            = ALL_AUTHZ + "/rpc"
	AUTHZ_STATE             = SINGLE_AUTHZ + "/state"
	AUTHZ_SUCCESS           = AUTHZ_STATE + "/access-accepts"
	AUTHZ_SUCCESS_TIMESTAMP = AUTHZ_STATE + "/last-access-accept"
	AUTHZ_FAILED            = AUTHZ_STATE + "/access-rejects"
	AUTHZ_FAILED_TIMESTAMP  = AUTHZ_STATE + "/last-access-reject"
)

// XfmrCache a sync.Map for storing path values that need to be cached
var XfmrCache sync.Map

var pathzOpers = [][]string{[]string{READS_GET, ACCEPTS}, []string{READS_GET, REJECTS}, []string{READS_SUB, ACCEPTS}, []string{READS_SUB, REJECTS}, []string{WRITES, ACCEPTS}, []string{WRITES, REJECTS}}
var pathzMap = &pathzCounters{
	mu:      sync.Mutex{},
	updated: make(map[string]time.Time),
	data:    make(map[string]map[string]map[string]*uint64),
}

func init() {
	XlateFuncBind("DbToYang_sys_state_xfmr", DbToYang_sys_state_xfmr)
	XlateFuncBind("Subscribe_sys_state_xfmr", Subscribe_sys_state_xfmr)
	XlateFuncBind("Subscribe_sys_memory_xfmr", Subscribe_sys_memory_xfmr)
	XlateFuncBind("DbToYang_sys_memory_xfmr", DbToYang_sys_memory_xfmr)
	XlateFuncBind("DbToYang_sys_cpus_xfmr", DbToYang_sys_cpus_xfmr)
	XlateFuncBind("Subscribe_sys_cpus_xfmr", Subscribe_sys_cpus_xfmr)
	XlateFuncBind("DbToYang_sys_alarms_xfmr", DbToYang_sys_alarms_xfmr)
	XlateFuncBind("Subscribe_sys_alarms_xfmr", Subscribe_sys_alarms_xfmr)
	XlateFuncBind("DbToYangPath_sys_alarms_path_xfmr", DbToYangPath_sys_alarms_path_xfmr)
	XlateFuncBind("DbToYang_sys_procs_xfmr", DbToYang_sys_procs_xfmr)
	XlateFuncBind("Subscribe_sys_procs_xfmr", Subscribe_sys_procs_xfmr)
	XlateFuncBind("Subscribe_sys_aaa_auth_xfmr", Subscribe_sys_aaa_auth_xfmr)
	XlateFuncBind("DbToYangPath_sys_aaa_auth_path_xfmr", DbToYangPath_sys_aaa_auth_path_xfmr)
	XlateFuncBind("YangToDb_sys_aaa_auth_xfmr", YangToDb_sys_aaa_auth_xfmr)
	XlateFuncBind("DbToYang_sys_aaa_auth_xfmr", DbToYang_sys_aaa_auth_xfmr)
	XlateFuncBind("YangToDb_sys_config_key_xfmr", YangToDb_sys_config_key_xfmr)
	XlateFuncBind("DbToYang_sys_config_key_xfmr", DbToYang_sys_config_key_xfmr)
	XlateFuncBind("DbToYang_grpc_server_xfmr", DbToYang_grpc_server_xfmr)
	XlateFuncBind("YangToDb_grpc_server_xfmr", YangToDb_grpc_server_xfmr)
	XlateFuncBind("Subscribe_grpc_server_xfmr", Subscribe_grpc_server_xfmr)
	XlateFuncBind("DbToYang_grpc_server_key_xfmr", DbToYang_grpc_server_key_xfmr)
	XlateFuncBind("YangToDb_sys_config_xfmr", YangToDb_sys_config_xfmr)
	XlateFuncBind("DbToYang_sys_config_xfmr", DbToYang_sys_config_xfmr)
	XlateFuncBind("Subscribe_sys_config_xfmr", Subscribe_sys_config_xfmr)
	XlateFuncBind("DbToYang_ssh_server_state_xfmr", DbToYang_ssh_server_state_xfmr)
	XlateFuncBind("Subscribe_ssh_server_state_xfmr", Subscribe_ssh_server_state_xfmr)
	XlateFuncBind("DbToYang_pathz_policies_xfmr", DbToYang_pathz_policies_xfmr)
	XlateFuncBind("Subscribe_pathz_policies_xfmr", Subscribe_pathz_policies_xfmr)
	XlateFuncBind("DbToYang_pathz_policies_key_xfmr", DbToYang_pathz_policies_key_xfmr)
	XlateFuncBind("DbToYang_console_counters_xfmr", DbToYang_console_counters_xfmr)
	XlateFuncBind("Subscribe_console_counters_xfmr", Subscribe_console_counters_xfmr)
	XlateFuncBind("DbToYang_authz_policy_xfmr", DbToYang_authz_policy_xfmr)
	XlateFuncBind("Subscribe_authz_policy_xfmr", Subscribe_authz_policy_xfmr)
	XlateFuncBind("YangToDb_syslog_server_ip_fld_xfmr", YangToDb_syslog_server_ip_fld_xfmr)
	XlateFuncBind("DbToYang_syslog_server_ip_fld_xfmr", DbToYang_syslog_server_ip_fld_xfmr)
	XlateFuncBind("DbToYangPath_remote_server_path_xfmr", DbToYangPath_remote_server_path_xfmr)
}

type SysMem struct {
	Total                  uint64
	Used                   uint64
	Free                   uint64
	TotalEccErrors         uint64
	CorrectableEccErrors   uint64
	UncorrectableEccErrors uint64
}

type Cpu struct {
	User   int64
	System int64
	Idle   int64
	Total  timeStat
}

type MountPoint struct {
	Name      string
	Size      uint64
	Available uint64
	Utilized  uint64
	Type      string
}

type Proc struct {
	Cmd      string
	Start    uint64
	User     float64
	System   float64
	Mem      uint64
	Cputil   float64
	Memutil  float64
	MemLimit *uint64
}

type CpuState struct {
	user   uint8
	system uint8
	idle   uint8
}

type timeStat struct {
	avg      uint8
	interval uint64
}

type ProcessState struct {
	Args              []string
	CpuUsageSystem    uint64
	CpuUsageUser      uint64
	CpuUtilization    uint8
	MemoryLimit       *uint64
	MemoryUsage       uint64
	MemoryUtilization uint8
	Name              string
	Pid               uint64
	StartTime         uint64
}

type sysState struct {
	Hostname                   string
	LastConfigurationTimestamp string
	MetaData                   string
	BootTime                   uint64
	VerificationStatus         string
	BootType                   string
	WarmbootCount              uint32
	LastColdbootTime           uint64
	LastColdbootVersion        string
}

type alarmState struct {
	state         string
	reason        string
	timeCreated   uint64
	essential     string
	hwErr         string
	debugInfo     string
	debugInfoList string
}

type authUserState struct {
	userName   string
	password   certData
	principals certData
	keys       certData
}

type sshState struct {
	caKeys   certData
	hostCert certData
	hostKey  certData
	counters accessCounters
}

type certData struct {
	version string
	created uint64
}

type accessCounters struct {
	accessRejects    uint64
	lastAccessReject uint64
	accessAccepts    uint64
	lastAccessAccept uint64
}

type pathzCounters struct {
	mu      sync.Mutex
	updated map[string]time.Time
	data    map[string]map[string]map[string]*uint64
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
	pathzVersion   string
	pathzCreated   uint64
	profileId      string
}

type policyState struct {
	instance ocbinds.E_OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance
	version  string
	created  uint64
}

type gnpsiServer struct {
	enable *bool
	port   *uint16
}

var dbToYangPathzInstanceMap = map[string]ocbinds.E_OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance{
	"ACTIVE":  ocbinds.OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance_ACTIVE,
	"SANDBOX": ocbinds.OpenconfigSystem_System_GnmiPathzPolicies_Policies_Policy_State_Instance_SANDBOX,
}

func getAppRootObject(inParams XfmrParams) *ocbinds.OpenconfigSystem_System {
	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	return deviceObj.System
}

func updateResMapFromDB(entry db.Value, attr string, resMap map[string]string) {
	if val := entry.Get(attr); val != "" {
		resMap[attr] = val
	}
}

var YangToDb_sys_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	if inParams.oper == DELETE {
		switch {
		case strings.Contains(inParams.requestUri, "hostname"):
			return map[string]map[string]db.Value{
				"DEVICE_METADATA": map[string]db.Value{
					"localhost": db.Value{
						Field: map[string]string{"hostname": ""},
					},
				},
			}, nil
		case strings.Contains(inParams.requestUri, "config-meta-data"):
			return map[string]map[string]db.Value{
				"DEVICE_METADATA": map[string]db.Value{
					"localhost": db.Value{
						Field: map[string]string{"config-meta-data": ""},
					},
				},
			}, nil
		default:
			return nil, tlerr.InvalidArgs("SET Delete not supported at the subtree level for %v", inParams.requestUri)
		}
	}
	sysObj := getAppRootObject(inParams)
	if sysObj == nil {
		log.V(lvl.DEBUG).Info("YangToDb_sys_config_xfmr: Empty component.")
		return nil, tlerr.NotSupported("YangToDb_sys_config_xfmr: Empty component.")
	}
	if sysObj.Config == nil {
		return nil, nil
	}
	resMap := make(map[string]string)
	if sysObj.Config.Hostname != nil {
		resMap["hostname"] = *sysObj.Config.Hostname
	}
	memMap := map[string]map[string]db.Value{
		"DEVICE_METADATA": map[string]db.Value{
			"localhost": db.Value{
				Field: resMap,
			},
		},
	}

	// In case of a Set Replace for the /system/config subtree, do an Update instead to
	// preserve system-written fields in DEVICE_METADATA|localhost table (Ref:b/199801106)
	if inParams.oper == REPLACE {
		updateSubOpDataMap(map[db.DBNum]map[string]map[string]db.Value{
			db.ConfigDB: memMap,
		}, UPDATE, inParams)
		return nil, nil
	}

	return memMap, nil
}

var DbToYang_sys_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	sysObj := getAppRootObject(inParams)
	var err error
	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		cfgDb, err = db.NewDB(getDBOptions(db.ConfigDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer cfgDb.DeleteDB()
	}

	var sysInfo sysState
	entry, err := cfgDb.GetEntry(&db.TableSpec{Name: "DEVICE_METADATA"}, db.Key{Comp: []string{"localhost"}})
	if err != nil {
		return err
	}
	sysInfo.MetaData = entry.Get("config-meta-data")
	sysInfo.Hostname = entry.Get("hostname")

	ygot.BuildEmptyTree(sysObj)
	sysObj.Config.Hostname = &sysInfo.Hostname
	return nil
}

func getSystemState(sysInfo *sysState, sysstate *ocbinds.OpenconfigSystem_System_State) {
	log.V(lvl.DEBUG).Infof("getSystemState Entry")

	crtime := time.Now().Local().Format(time.RFC3339)

	sysstate.Hostname = &sysInfo.Hostname
	sysstate.CurrentDatetime = &crtime
	if sysInfo.LastConfigurationTimestamp != "" {
		timestamp, err := strconv.ParseUint(sysInfo.LastConfigurationTimestamp, 10, 64)
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to convert last-configuration-timestamp to uint64: %v. Error: %v", sysInfo.LastConfigurationTimestamp, err)
		} else {
			sysstate.LastConfigurationTimestamp = &timestamp
		}
	}

	sysinfo := syscall.Sysinfo_t{}
	sys_err := syscall.Sysinfo(&sysinfo)
	if sys_err != nil {
		log.V(lvl.WARNING).Infof("getSystemState syscall error: %s", sys_err.Error())
		return
	}
	uptime := uint64(sysinfo.Uptime * 1_000_000_000)
	sysstate.UpTime = &uptime

	sysstate.BootTime = &sysInfo.BootTime
	// If boot-time is not present in the database, calculate it
	if sysInfo.BootTime == 0 {
		bt, ok := XfmrCache.Load("boot-time")
		if !ok {
			bt = uint64(time.Now().Local().UnixNano() - sysinfo.Uptime*1_000_000_000)
			XfmrCache.Store("boot-time", bt)
		}
		boot_time := bt.(uint64)
		sysstate.BootTime = &boot_time
	}

}

func hostnameFromDb(d, cfgdb *db.DB) string {
	if hostEntry, err := d.GetEntry(&db.TableSpec{Name: HOST_TBL}, db.Key{Comp: []string{HOSTNAME_KEY}}); err == nil {
		return hostEntry.Get("hostname")
	}
	if entry, _ := d.GetEntry(&db.TableSpec{Name: "DEVICE_METADATA"}, db.Key{Comp: []string{"localhost"}}); entry.Get("hostname") != "" {
		return entry.Get("hostname")
	}

	// TODO(b/383659899): Remove read from config db when statedb option is supported.
	entry, _ := cfgdb.GetEntry(&db.TableSpec{Name: "DEVICE_METADATA"}, db.Key{Comp: []string{"localhost"}})
	return entry.Get("hostname")
}

func getSysStateFromDb(d *db.DB, cfgDb *db.DB, applStateDb *db.DB) (*sysState, error) {
	var sysInfo sysState

	sysInfo.Hostname = hostnameFromDb(d, cfgDb)

	if lastConfigEntry, err := d.GetEntry(&db.TableSpec{Name: HOST_TBL}, db.Key{Comp: []string{HOSTCONFIG_KEY}}); err != nil {
		log.V(tlerr.ErrorSeverity(err)).Info("Can't get entry with key: ", HOSTCONFIG_KEY)
	} else {
		sysInfo.LastConfigurationTimestamp = lastConfigEntry.Get("last-configuration-timestamp")
	}

	bootEntry, err := d.GetEntry(&db.TableSpec{Name: BOOT_INFO_TBL}, db.Key{Comp: []string{systemKey}})
	if err != nil {
		log.V(tlerr.ErrorSeverity(err)).Infof("Can't get entry %v|%v: %v", BOOT_INFO_TBL, systemKey, err)
	}
	sysInfo.BootType = bootEntry.Get("boot-type")
	warmbootCount := bootEntry.Get("warmboot-count")
	if count, err := strconv.ParseUint(warmbootCount, 10, 32); err != nil {
		log.V(lvl.DEBUG).Infof("Failed to convert warmboot count to uint: %v", err)
	} else {
		sysInfo.WarmbootCount = uint32(count)
	}
	lcbt := bootEntry.Get("last-coldboot-timestamp")
	if coldbootTime, err := time.Parse(BASE_TIME_FORMAT, lcbt); err != nil {
		log.V(lvl.DEBUG).Infof("Failed to parse last coldboot timestamp: %v", err)
	} else {
		sysInfo.LastColdbootTime = uint64(coldbootTime.UnixNano())
	}
	sysInfo.LastColdbootVersion = bootEntry.Get("last-coldboot-version")

	// TODO(b/185837247): Remove Config DB lookup post V5 when Backend is ready and use sysEntry instead.
	entry, err := cfgDb.GetEntry(&db.TableSpec{Name: "DEVICE_METADATA"}, db.Key{Comp: []string{"localhost"}})
	if err != nil {
		return nil, err
	}
	sysInfo.MetaData = entry.Get("config-meta-data")

	verificationEntry, err := d.GetEntry(&db.TableSpec{Name: VERIFY_STATE_RESP_TBL}, db.Key{Comp: []string{"all"}})
	if err == nil {
		sysInfo.VerificationStatus = verificationEntry.Get("status")
	}

	chassisEntry, err := d.GetEntry(&db.TableSpec{Name: CHASSIS_TBL}, db.Key{Comp: []string{CHASSIS_PREFIX}})
	if err != nil {
		log.V(tlerr.ErrorSeverity(err)).Infof("Can't get entry %v|%v: err=%v", CHASSIS_TBL, CHASSIS_PREFIX, err)
		return &sysInfo, nil
	}
	bt := chassisEntry.Get("boot-time")
	if bt == "" {
		log.V(lvl.DEBUG).Info("Boot-time missing from STATE_DB")
	} else {
		bt_time, err := time.Parse(BASE_TIME_FORMAT, bt)
		if err != nil {
			log.V(tlerr.ErrorSeverity(err)).Info("Boot-time %v timestamp conversion failed.", bt)
			return &sysInfo, nil
		}
		sysInfo.BootTime = uint64(bt_time.UnixNano())
	}

	return &sysInfo, nil
}

var YangToDb_sys_config_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	log.V(lvl.DEBUG).Info("YangToDb_sys_config_key_xfmr: ", inParams.uri)
	dvKey := "localhost"
	return dvKey, nil
}

var DbToYang_sys_config_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	rmap := make(map[string]interface{})
	log.V(lvl.DEBUG).Info("DbToYang_sys_config_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)
	return rmap, nil
}

var Subscribe_sys_config_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	result := XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.ConfigDB: {"DEVICE_METADATA": {"localhost": {"config-meta-data": "ConfigMetaData"}}}},
		onChange: OnchangeDisable,
	}
	log.V(lvl.DEBUG).Infof("Subscribe_sys_config_xfmr:%s", inParams.requestURI)

	return result, nil
}

var Subscribe_sys_state_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	result := XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.ConfigDB: {"DEVICE_METADATA": {"localhost": {"config-meta-data": "ConfigMetaData"}}}},
		onChange: OnchangeDisable,
	}
	if strings.Contains(inParams.requestURI, "config-meta-data") {
		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{mInterval: 0, pType: OnChange}
	}
	log.V(lvl.DEBUG).Infof("Subscribe_sys_state_xfmr:%s", inParams.requestURI)

	return result, nil
}

var DbToYang_sys_state_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	sysObj := getAppRootObject(inParams)
	var err error
	stateDb := inParams.dbs[db.StateDB]
	if stateDb == nil {
		stateDb, err = db.NewDB(getDBOptions(db.StateDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer stateDb.DeleteDB()
	}
	// TODO(b/185837247): Remove Config DB lookup post V5 when Backend is ready.
	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		cfgDb, err = db.NewDB(getDBOptions(db.ConfigDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer cfgDb.DeleteDB()
	}
	applStateDb := inParams.dbs[db.ApplStateDB]
	if applStateDb == nil {
		applStateDb, err = db.NewDB(getDBOptions(db.ApplStateDB))
		if err != nil {
			return tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer applStateDb.DeleteDB()
	}

	sysInfo, err := getSysStateFromDb(stateDb, cfgDb, applStateDb)
	if err != nil {
		log.V(lvl.DEBUG).Infof("getSysStateFromDb failed")
		return err
	}

	ygot.BuildEmptyTree(sysObj)
	getSystemState(sysInfo, sysObj.State)
	return nil
}

var Subscribe_sys_memory_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	result := XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"MEMORY_STATS": {"system": {}}}},
		onChange: OnchangeDisable,
	}
	log.V(lvl.DEBUG).Infof("Subscribe_sys_memory_xfmr: %s", inParams.requestURI)

	return result, nil
}

func getSysMemFromDb(d *db.DB) (*SysMem, error) {
	var err error

	memInfo := SysMem{}
	memEntry, err := d.GetEntry(&db.TableSpec{Name: MEMORY_TBL}, db.Key{Comp: []string{systemKey}})
	if err != nil {
		log.V(lvl.DEBUG).Infof("Can't get entry with key: %v err: %v", systemKey, err)
		return &memInfo, err
	}
	var errs []string
	if memInfo.Total, err = strconv.ParseUint(memEntry.Get("total"), 10, 64); err != nil {
		msg := fmt.Sprintf("total: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	if memInfo.Used, err = strconv.ParseUint(memEntry.Get("used"), 10, 64); err != nil {
		msg := fmt.Sprintf("used: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	if memInfo.Free, err = strconv.ParseUint(memEntry.Get("free"), 10, 64); err != nil {
		msg := fmt.Sprintf("free: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	if memInfo.TotalEccErrors, err = strconv.ParseUint(memEntry.Get("total-ecc-errors"), 10, 64); err != nil {
		msg := fmt.Sprintf("total-ecc-errors: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	if memInfo.CorrectableEccErrors, err = strconv.ParseUint(memEntry.Get("correctable-ecc-errors"), 10, 64); err != nil {
		msg := fmt.Sprintf("correctable-ecc-errors: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	if memInfo.UncorrectableEccErrors, err = strconv.ParseUint(memEntry.Get("uncorrectable-ecc-errors"), 10, 64); err != nil {
		msg := fmt.Sprintf("uncorrectable-ecc-errors: %s", err.Error())
		log.V(lvl.DEBUG).Info(msg)
		errs = append(errs, msg)
	}
	err = nil
	if len(errs) > 0 {
		err = fmt.Errorf(strings.Join(errs, "; "))
	}
	return &memInfo, err
}

func getSystemMemory(meminfo *SysMem, sysmem *ocbinds.OpenconfigSystem_System_Memory_State) {
	log.V(lvl.DEBUG).Infof("getSystemMemory Entry")
	sysmem.Physical = &meminfo.Total
	sysmem.Reserved = &meminfo.Used
	sysmem.Used = &meminfo.Used
	sysmem.Free = &meminfo.Free
	sysmem.Counters.TotalEccErrors = &meminfo.TotalEccErrors
	sysmem.Counters.CorrectableEccErrors = &meminfo.CorrectableEccErrors
	sysmem.Counters.UncorrectableEccErrors = &meminfo.UncorrectableEccErrors
}

var DbToYang_sys_memory_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error
	log.V(lvl.DEBUG).Infof("DbToYang_sys_memory_xfmr called")
	sysObj := getAppRootObject(inParams)
	meminfo, err := getSysMemFromDb(inParams.dbs[db.StateDB])
	if err != nil {
		log.V(lvl.DEBUG).Infof("getSysMemFromDb failed")
		return err
	}
	ygot.BuildEmptyTree(sysObj)
	if sysObj.Memory == nil {
		ygot.BuildEmptyTree(sysObj.Memory)
	}
	sysObj.Memory.State = &ocbinds.OpenconfigSystem_System_Memory_State{}
	sysObj.Memory.State.Counters = &ocbinds.OpenconfigSystem_System_Memory_State_Counters{}
	getSystemMemory(meminfo, sysObj.Memory.State)
	return err
}

func getSystemCpu(idx int, cpuCnt int, cpu Cpu, syscpu *ocbinds.OpenconfigSystem_System_Cpus_Cpu) {
	log.V(lvl.DEBUG).Info("getSystemCpu Entry idx ", idx)

	sysinfo := syscall.Sysinfo_t{}
	sys_err := syscall.Sysinfo(&sysinfo)
	if sys_err != nil {
		log.V(lvl.DEBUG).Infof("syscall.Sysinfo failed.")
	}
	var cpucur CpuState
	if idx == 0 && cpuCnt > 0 {
		cpucur.user = uint8((cpu.User / int64(cpuCnt)) / sysinfo.Uptime)
		cpucur.system = uint8((cpu.System / int64(cpuCnt)) / sysinfo.Uptime)
		cpucur.idle = uint8((cpu.Idle / int64(cpuCnt)) / sysinfo.Uptime)
	} else {
		cpucur.user = uint8(cpu.User / sysinfo.Uptime)
		cpucur.system = uint8(cpu.System / sysinfo.Uptime)
		cpucur.idle = uint8(cpu.Idle / sysinfo.Uptime)
	}

	ygot.BuildEmptyTree(syscpu.State)
	syscpu.State.User.Instant = &cpucur.user
	syscpu.State.Kernel.Instant = &cpucur.system
	syscpu.State.Idle.Instant = &cpucur.idle
	syscpu.State.Total.Avg = &cpu.Total.avg
	syscpu.State.Total.Interval = &cpu.Total.interval
}

func getSystemCpus(cpus map[int]*Cpu, syscpus *ocbinds.OpenconfigSystem_System_Cpus) {
	log.V(lvl.DEBUG).Info("getSystemCpus Entry")

	sysinfo := syscall.Sysinfo_t{}
	sys_err := syscall.Sysinfo(&sysinfo)
	cpuCnt := len(cpus) - 1
	if sys_err != nil {
		log.V(lvl.DEBUG).Info("syscall.Sysinfo failed.")
	}

	for idx, cpu := range cpus {
		var index ocbinds.OpenconfigSystem_System_Cpus_Cpu_State_Index_Union_Uint32
		index.Uint32 = uint32(idx)
		syscpu, err := syscpus.NewCpu(&index)
		if err != nil {
			log.V(lvl.DEBUG).Info("syscpus.NewCpu failed")
			return
		}
		ygot.BuildEmptyTree(syscpu)
		syscpu.Index = &index
		getSystemCpu(idx, cpuCnt, *cpu, syscpu)
	}
}

func getCpusFromDb(d *db.DB) (map[int]*Cpu, error) {
	var err error

	cpuTbl, err := d.GetTable(&db.TableSpec{Name: CPU_TBL})
	if err != nil {
		log.V(lvl.DEBUG).Info("Can't get table: ", CPU_TBL)
		return nil, err
	}

	keys, err := cpuTbl.GetKeys()
	if err != nil {
		log.V(lvl.DEBUG).Info("Can't get CPU keys from table")
		return nil, err
	}

	cpus := make(map[int]*Cpu)
	for _, key := range keys {
		if len(key.Comp) == 0 {
			continue
		}
		idx, err := strconv.Atoi(key.Comp[0])
		if err != nil {
			log.V(lvl.DEBUG).Infof("Invalid CPU stat key: %v", key)
			continue
		}
		cpuEntry, err := cpuTbl.GetEntry(key)
		if err != nil {
			log.V(lvl.DEBUG).Infof("Can't get entry with key: %v", key)
			return nil, err
		}

		cpu := &Cpu{}
		interval, err := strconv.ParseFloat(cpuEntry.Get("total_interval"), 64)
		if err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty Total.interval for cpu-%d: %v", idx, err)
		}
		cpu.Total.interval = uint64(interval)
		avg, err := strconv.ParseFloat(cpuEntry.Get("total_avg"), 64)
		if err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty Total.avg for cpu-%d: %v", idx, err)
		}
		cpu.Total.avg = uint8(avg)
		cpus[idx] = cpu
	}

	return cpus, err
}

var DbToYang_sys_cpus_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error

	sysObj := getAppRootObject(inParams)

	cpus, err := getCpusFromDb(inParams.dbs[db.StateDB])
	if err != nil {
		log.V(lvl.DEBUG).Info("getCpusFromDb failed")
		return err
	}
	if sysObj.Cpus == nil {
		ygot.BuildEmptyTree(sysObj)
	}

	path := NewPathInfo(inParams.uri)
	if val, ok := path.Vars["index"]; ok {
		idx, err := strconv.Atoi(val)
		if err != nil {
			log.V(lvl.DEBUG).Info("Invalid cpu index ", val)
			return err
		}
		totalCpu := len(cpus)
		if cpu, ok := cpus[idx]; ok {
			//Since key(a pointer) is unknown, there is no way to do a lookup. So looping through a map with only one entry
			for _, value := range sysObj.Cpus.Cpu {
				ygot.BuildEmptyTree(value)
				getSystemCpu(idx, totalCpu-1, *cpu, value)
			}
		} else {
			log.V(lvl.DEBUG).Info("Cpu id: ", cpu, "is invalid, max is ", totalCpu)
		}
	} else {
		getSystemCpus(cpus, sysObj.Cpus)
	}
	return err
}

var Subscribe_sys_cpus_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	key := NewPathInfo(inParams.uri).Var("index")
	log.V(lvl.DEBUG).Infof("+++ Subscribe_sys_cpus_xfmr (%v) +++", inParams.uri)
	if key == "" {
		if inParams.subscProc != TRANSLATE_SUBSCRIBE {
			/* no need to verify dB data if we are requesting ALL cpus */
			return XfmrSubscOutParams{isVirtualTbl: true}, nil
		}
		key = "*"
	}
	return XfmrSubscOutParams{dbDataMap: RedisDbSubscribeMap{db.StateDB: {CPU_TBL: {key: {}}}}}, nil
}

func getMountPointsFromDb(d *db.DB) (map[string]MountPoint, error) {
	var err error
	var curMountPoint MountPoint

	mountPointTbl, err := d.GetTable(&db.TableSpec{Name: MOUNT_POINTS_TBL})
	if err != nil {
		log.V(lvl.ERROR).Infof("Can't get table: %v, err: %s", MOUNT_POINTS_TBL, err.Error())
		return nil, err
	}

	keys, err := mountPointTbl.GetKeys()
	if err != nil {
		log.V(lvl.ERROR).Infof("Can't get mount point keys from table err: %s", err.Error())
		return nil, err
	}

	mount_points := make(map[string]MountPoint)
	for _, key := range keys {
		mountPointStr := key.Get(0)
		//Adding the filter here to filter out name=LastUpdateTime
		if mountPointStr == "LastUpdateTime" {
			continue
		}

		mountPointEntry, err := mountPointTbl.GetEntry(key)
		if err != nil {
			log.V(lvl.ERROR).Infof("Can't get entry with key %v. err: %s", mountPointStr, err.Error())
			return nil, err
		}
		curMountPoint.Name = mountPointStr
		if curMountPoint.Size, err = strconv.ParseUint(mountPointEntry.Get("size"), 10, 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid size for mount point - %d: %v", curMountPoint.Name, err)
		}
		if curMountPoint.Available, err = strconv.ParseUint(mountPointEntry.Get("available"), 10, 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid available space for mount point - %d: %v", curMountPoint.Name, err)
		}
		if curMountPoint.Utilized, err = strconv.ParseUint(mountPointEntry.Get("used"), 10, 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid utilized space for mount point - %d: %v", curMountPoint.Name, err)
		}
		curMountPoint.Type = mountPointEntry.Get("filesystem-type")
		mount_points[mountPointStr] = curMountPoint
	}

	return mount_points, nil
}

func translateDBKeyToAlarmID(entry *db.Value, tblKey string) (string, error) {
	alarmInfo, alarmInfoErr := alarmStateFromDb(entry)
	if alarmInfoErr != nil {
		log.V(lvl.DEBUG).Infof("Can't parse entry %s with err: %v", tblKey, alarmInfoErr)
		return "", alarmInfoErr
	}
	return tblKey + "_" + strconv.FormatUint(alarmInfo.timeCreated, 10), nil
}

/* Given a DB entry representing an alarm, populate a type alarmState with the
 * DB fields.  Returns an error for failed conversions, e.g. invalid timestamp
 * strings. */
func alarmStateFromDb(entry *db.Value) (alarmState, error) {
	var rv alarmState
	rv.state = entry.Get("state")
	rv.reason = entry.Get("reason")
	rv.essential = entry.Get("essential")
	rv.hwErr = entry.Get("hw-err")
	rv.debugInfo = entry.Get("debug_info")
	rv.debugInfoList = entry.Get("debug_info_list")

	timeSec, tsErr := strconv.ParseUint(entry.Get("timestamp-seconds"), 10, 64)
	if tsErr != nil {
		log.V(lvl.DEBUG).Infof("Can't parse timestamp-seconds entry with err: %v", tsErr)
		return rv, tsErr
	}

	timeNanoSec, tnsErr := strconv.ParseUint(entry.Get("timestamp-nanoseconds"), 10, 64)
	if tnsErr != nil {
		log.V(lvl.DEBUG).Infof("Can't parse timestamp-nanoseconds entry with err: %v", tnsErr)
		return rv, tnsErr
	}

	rv.timeCreated = (timeSec * 1000000000) + timeNanoSec
	return rv, nil
}

/* The state DB may contain alarms we do not want to report, this function returns true
 * for such cases.
 * Alarms which have Debug Info or the following states will NOT be suppressed: ERROR, MINOR, INACTIVE.
 * Otherwise the alarm is suppressed. */
func alarmEntryIsFiltered(alarmInfo alarmState) bool {
	if alarmInfo.state != "ERROR" && alarmInfo.state != "INACTIVE" && alarmInfo.state != "MINOR" && alarmInfo.debugInfo != "true" {
		log.V(lvl.DEBUG).Infof("alarm entry filtered: %v", alarmInfo)
		return true
	}
	return false
}

/* Convert the string fields from STATE_DB into an openconfig alarm severity
 * enum according to go/crash-artifact-framework */
func alarmSeverityFromState(alarmInfo alarmState) ocbinds.E_OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY {
	switch alarmInfo.state {
	case "INACTIVE":
		fallthrough
	case "ERROR":
		if essential, err := strconv.ParseBool(alarmInfo.essential); err == nil && essential {
			return ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_CRITICAL
		}
		return ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_MAJOR
	case "MINOR":
		return ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_WARNING
	default:
		return ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_SEVERITY_MINOR
	}
}

func fillAlarmDBInfo(sysAlarm *ocbinds.OpenconfigSystem_System_Alarms_Alarm, alarmInfo *alarmState, id, path, tblKey string) (err error) {
	switch path {
	case ALRM_STATE_RESOURCE:
		sysAlarm.State.Resource = &tblKey
	case ALRM_STATE_ID:
		sysAlarm.State.Id = &id
	case ALRM_STATE_TIMECREATED:
		sysAlarm.State.TimeCreated = &alarmInfo.timeCreated
	case ALRM_STATE_TYPEID:
		if alarmInfo.hwErr == "true" {
			if sysAlarm.State.TypeId, err = sysAlarm.State.To_OpenconfigSystem_System_Alarms_Alarm_State_TypeId_Union(ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_TYPE_ID_EQPT); err != nil {
				return errors.New("error in setting type-id: " + err.Error())
			}
			return nil
		}
		if sysAlarm.State.TypeId, err = sysAlarm.State.To_OpenconfigSystem_System_Alarms_Alarm_State_TypeId_Union("SOFTWARE"); err != nil {
			return errors.New("error in setting type-id: " + err.Error())
		}
	case ALRM_STATE_TEXT:
		if alarmInfo.state == "" {
			return errors.New("state field not found in DB.")
		}
		text := alarmInfo.state + ": " + alarmInfo.reason
		sysAlarm.State.Text = &text
	case ALRM_STATE_SEVERITY:
		if alarmInfo.state == "" {
			return errors.New("state field not found in DB.")
		}
		sysAlarm.State.Severity = alarmSeverityFromState(*alarmInfo)
	default:
		return errors.New("path not supported for alarm state.")
	}
	return nil
}

func getAlarmState(sysAlarms *ocbinds.OpenconfigSystem_System_Alarms, alarmTbl *db.Table, all bool, id, targetUriPath string) error {
	log.V(lvl.DEBUG).Infof("getAlarmState Entry: %s allPaths=%v", id, all)

	// Example id = syncd:syncd_1611693908000044444, container_monitor_1611693908000044444
	tblKey := id
	if keyIdx := strings.LastIndex(id, "_"); keyIdx != -1 {
		tblKey = id[:keyIdx]
	}
	entry, err := alarmTbl.GetEntry(db.Key{Comp: []string{tblKey}})
	if err != nil {
		return errors.New("Can't get entry with key: " + tblKey)
	}

	alarmInfo, alarmInfoErr := alarmStateFromDb(&entry)
	if alarmInfoErr != nil {
		return errors.New("Can't parse " + tblKey + " with err: " + alarmInfoErr.Error())
	}
	if (alarmInfo.state == "INITIALIZING" || alarmInfo.state == "UP") && alarmInfo.debugInfo == "true" {
		alarmInfo.reason = "Crash artifact detected: " + alarmInfo.debugInfoList
	}

	if alarmEntryIsFiltered(alarmInfo) {
		return tlerr.NotFoundErr("", targetUriPath, "Alarm Filtered: %s", id)
	}
	sysAlarm, ok := sysAlarms.Alarm[id]
	if !ok || sysAlarm == nil {
		sysAlarm, err = sysAlarms.NewAlarm(id)
		if err != nil {
			return errors.New("Cannot create alarm object for: " + err.Error())
		}
	}
	ygot.BuildEmptyTree(sysAlarm)
	ygot.BuildEmptyTree(sysAlarm.State)
	if !all {
		return fillAlarmDBInfo(sysAlarm, &alarmInfo, id, targetUriPath, tblKey)
	}

	// Ignore errors for subtree level request
	if alarmInfoErr == nil {
		sysAlarm.State.TimeCreated = &alarmInfo.timeCreated
	}
	sysAlarm.State.Id = &id
	text := alarmInfo.state + ": " + alarmInfo.reason
	sysAlarm.State.Text = &text
	sysAlarm.State.Resource = &tblKey
	sysAlarm.State.TypeId, _ = sysAlarm.State.To_OpenconfigSystem_System_Alarms_Alarm_State_TypeId_Union("SOFTWARE")
	if alarmInfo.hwErr == "true" {
		sysAlarm.State.TypeId, _ = sysAlarm.State.To_OpenconfigSystem_System_Alarms_Alarm_State_TypeId_Union(ocbinds.OpenconfigAlarmTypes_OPENCONFIG_ALARM_TYPE_ID_EQPT)
	}
	sysAlarm.State.Severity = alarmSeverityFromState(alarmInfo)

	return nil
}

func getAllAlarmsState(sysAlarms *ocbinds.OpenconfigSystem_System_Alarms, alarmTbl *db.Table, targetUriPath string) error {
	alarmIDKeys, err := alarmTbl.GetKeys()
	if err != nil || len(alarmIDKeys) < 1 {
		return errors.New("Failed to get keys from: " + ALARM_TBL)
	}
	for _, id := range alarmIDKeys {
		if id.Len() < 1 {
			continue
		}
		tblKey := id.Get(0)
		entry, err := alarmTbl.GetEntry(db.Key{Comp: []string{tblKey}})
		if err != nil {
			log.V(lvl.DEBUG).Infof("Can't get DB entry %v with err: %v", tblKey, err)
			continue
		}
		alarmID, err := translateDBKeyToAlarmID(&entry, tblKey)
		if err != nil {
			continue
		}
		getAlarmState(sysAlarms, alarmTbl, true, alarmID, targetUriPath)
	}
	return nil
}

var DbToYang_sys_alarms_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	alarmTbl, err := inParams.d.GetTable(&db.TableSpec{Name: ALARM_TBL})
	if err != nil {
		return errors.New("Can't get table: " + ALARM_TBL + " Err: " + err.Error())
	}
	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.Alarms)
	pathInfo := NewPathInfo(inParams.uri)
	alarmID, ok := pathInfo.Vars["id"]
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)

	log.V(lvl.DEBUG).Infof("+++ DbToYang_sys_alarms_xfmr (%v) targetUriPath=%s Id=%s +++", inParams.uri, targetUriPath, alarmID)

	// For paths /system/alarms and /system/alarms/alarm
	if !ok || len(alarmID) == 0 {
		getAllAlarmsState(sysObj.Alarms, &alarmTbl, targetUriPath)
		return nil
	}

	allPaths := false
	if targetUriPath == ALARM_ROOT || targetUriPath == ALRM_STATE {
		allPaths = true
	}
	return getAlarmState(sysObj.Alarms, &alarmTbl, allPaths, alarmID, targetUriPath)
}

var Subscribe_sys_alarms_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	key := NewPathInfo(inParams.uri).Var("id")
	log.V(lvl.DEBUG).Infof("+++ Subscribe_sys_alarms_xfmr (%v) op %v key %s +++", inParams.uri, inParams.subscProc, key)
	if key == "" {
		if inParams.subscProc != TRANSLATE_SUBSCRIBE {
			/* no need to verify dB data if we are requesting ALL alarms */
			return XfmrSubscOutParams{isVirtualTbl: true}, nil
		}
		key = "*"
	}
	// Example key = syncd:syncd_1611693908000044444, container_monitor_1611693908000044444
	if keyIdx := strings.LastIndex(key, "_"); keyIdx != -1 {
		key = key[:keyIdx]
	}
	return XfmrSubscOutParams{
		needCache: true,
		onChange:  OnchangeEnable,
		nOpts:     &notificationOpts{mInterval: 0, pType: OnChange},
		/* DB field "state" maps to "severity" leaf */
		dbDataMap: RedisDbSubscribeMap{db.StateDB: {ALARM_TBL: {key: {"state": "severity"}}}}}, nil
}

var DbToYangPath_sys_alarms_path_xfmr PathXfmrDbToYangFunc = func(inParams XfmrDbToYgPathParams) error {
	log.V(lvl.DEBUG).Infof("DbToYangPath_sys_alarms_path_xfmr: yangPath %v tblKeyComp %v", inParams.yangPath, inParams.tblKeyComp)

	if len(inParams.tblKeyComp) != 1 || inParams.tblEntry == nil {
		return fmt.Errorf("DbToYangPath_sys_alarms_path_xfmr: Invalid tblKey %v or tblEntry %v", inParams.tblKeyComp, inParams.tblEntry)
	}

	alarmID, err := translateDBKeyToAlarmID(inParams.tblEntry, inParams.tblKeyComp[0])
	if err != nil {
		return err
	}
	inParams.ygPathKeys[ALARM_ROOT+"/id"] = alarmID
	log.V(lvl.DEBUG).Infof("DbToYangPath_sys_alarms_path_xfmr: inParams.ygPathKeys: %v", inParams.ygPathKeys)

	return nil
}

func getSystemProcess(proc *Proc, sysproc *ocbinds.OpenconfigSystem_System_Processes_Process, pid uint64) {

	var procstate ProcessState

	ygot.BuildEmptyTree(sysproc)
	procstate.CpuUsageUser = uint64(proc.User * 1000000000)     // ns
	procstate.CpuUsageSystem = uint64(proc.System * 1000000000) // ns
	procstate.MemoryLimit = proc.MemLimit                       // The memory available to the container the process is running in.
	procstate.MemoryUsage = proc.Mem
	procstate.MemoryUtilization = uint8(proc.Memutil)
	procstate.CpuUtilization = uint8(proc.Cputil)
	procstate.Name = proc.Cmd
	procstate.Pid = pid
	procstate.StartTime = proc.Start // ns

	sysproc.Pid = &procstate.Pid
	sysproc.State.CpuUsageSystem = &procstate.CpuUsageSystem
	sysproc.State.CpuUsageUser = &procstate.CpuUsageUser
	sysproc.State.CpuUtilization = &procstate.CpuUtilization
	sysproc.State.MemoryUsage = &procstate.MemoryUsage
	sysproc.State.MemoryUtilization = &procstate.MemoryUtilization
	sysproc.State.Name = &procstate.Name
	sysproc.State.Pid = &procstate.Pid
	sysproc.State.StartTime = &procstate.StartTime
}

func getSystemProcesses(procs *map[string]Proc, sysprocs *ocbinds.OpenconfigSystem_System_Processes, pid uint64) (err error) {
	log.V(lvl.DEBUG).Infof("getSystemProcesses Entry")

	if pid != 0 {
		proc := (*procs)[strconv.Itoa(int(pid))]
		sysproc, ok := sysprocs.Process[pid]
		if !ok || sysproc == nil {
			sysproc, err = sysprocs.NewProcess(pid)
			if err != nil {
				return errors.New("sysprocs.NewProcess failed: " + err.Error())
			}
		}

		getSystemProcess(&proc, sysproc, pid)
	} else {

		for pidstr, proc := range *procs {
			idx, _ := strconv.Atoi(pidstr)
			sysproc, err := sysprocs.NewProcess(uint64(idx))
			if err != nil {
				return errors.New("sysprocs.NewProcess failed: " + err.Error())
			}

			getSystemProcess(&proc, sysproc, uint64(idx))
		}
	}
	return nil
}

func getProcsFromDb(d *db.DB) (map[string]Proc, error) {
	var err error
	var procs map[string]Proc
	var curProc Proc

	procTbl, err := d.GetTable(&db.TableSpec{Name: PROC_TBL})
	if err != nil {
		log.V(lvl.INFO).Infof("Can't get %v table: %v", PROC_TBL, err)
		return procs, err
	}

	keys, err := procTbl.GetKeys()
	if err != nil {
		log.V(lvl.DEBUG).Info("Can't get proc keys from table")
		return procs, err
	}

	memEntry, err := d.GetEntry(&db.TableSpec{Name: MEMORY_TBL}, db.Key{Comp: []string{systemKey}})
	if err != nil {
		log.V(lvl.DEBUG).Info("Can't get entry with key: ", systemKey)
		return nil, err
	}
	totalMem, _ := strconv.ParseUint(memEntry.Get("total"), 10, 64)

	procs = make(map[string]Proc)
	for _, key := range keys {
		pidstr := key.Get(0)
		procEntry, err := procTbl.GetEntry(key)
		if err != nil {
			log.V(lvl.DEBUG).Info("Can't get entry with key: ", pidstr)
			return procs, err
		}

		curProc.Cmd = procEntry.Get("CMD")
		if curProc.Cmd == "" {
			log.V(lvl.DEBUG).Infof("CMD empty for pid=%s, ignoring entry", key)
			continue
		}
		if t, err := time.ParseInLocation("Jan 2 2006 15:04:05 MST", procEntry.Get("STIME"), time.Local); err == nil {
			// "The timeticks64 represents the time, modulo 2^64 in nanoseconds
			//  between two epochs. The leaf using this type must define
			//  the epochs that tests are relative to."
			// It is converted to local time-zone to be consistent with 'boot-time'.
			curProc.Start = uint64(t.Local().UnixNano())
		} else {
			log.V(lvl.DEBUG).Infof("Invalid or empty STIME for process - %s: %v", curProc.Cmd, err)
		}
		if curProc.User, err = strconv.ParseFloat(procEntry.Get("USER_TIME"), 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty User for process - %v: %v", curProc.Cmd, err)
		}
		if curProc.System, err = strconv.ParseFloat(procEntry.Get("SYS_TIME"), 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty System for process - %v: %v", curProc.Cmd, err)
		}
		/* For memory-usage, commenting out the lookup for attribute "VSZ" to align
		 * with SONiC implementation. Instead, using the %MEM attribute along with
		 * the total memory to calculate the memory-usage. */
		// curProc.Mem, _ = strconv.ParseUint(procEntry.Get("VSZ"), 10, 64) * 1024
		if curProc.Cputil, err = strconv.ParseFloat(procEntry.Get("%CPU"), 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty Cputil for process - %v: %v", curProc.Cmd, err)
		}
		if curProc.Memutil, err = strconv.ParseFloat(procEntry.Get("%MEM"), 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty Memutil for process - %v: %v", curProc.Cmd, err)
		}
		if memLimit, err := strconv.ParseUint(procEntry.Get("MEM_LIMIT"), 10, 64); err != nil {
			log.V(lvl.DEBUG).Infof("Invalid or empty MemLimit for process - %v: %v", curProc.Cmd, err)
			curProc.MemLimit = nil
		} else {
			curProc.MemLimit = &memLimit
		}
		log.V(lvl.DEBUG).Infof("curProc.MemLimit: %v", curProc.MemLimit)
		curProc.Mem = uint64((curProc.Memutil / 100) * float64(totalMem))
		procs[pidstr] = curProc
	}

	/* Delete the one non-pid key procdockerstatsd deamon uses to store the last
	 * update time in the PROCCESSSTATS table */
	delete(procs, "LastUpdateTime")

	return procs, err
}

var DbToYang_sys_procs_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error

	sysObj := getAppRootObject(inParams)

	procs, err := getProcsFromDb(inParams.dbs[db.StateDB])
	if err != nil {
		log.V(lvl.DEBUG).Infof("getProcsFromDb failed")
		return err
	}

	ygot.BuildEmptyTree(sysObj)
	path := NewPathInfo(inParams.uri)
	val := path.Vars["pid"]
	pid := 0
	if len(val) != 0 {
		pid, _ = strconv.Atoi(val)
	}
	return getSystemProcesses(&procs, sysObj.Processes, uint64(pid))
}

var Subscribe_sys_procs_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	if key := NewPathInfo(inParams.uri).Var("pid"); key != "" {
		return XfmrSubscOutParams{dbDataMap: RedisDbSubscribeMap{db.StateDB: {PROC_TBL: {key: {}}}}}, nil
	}
	/* no need to verify dB data if we are requesting ALL processes */
	return XfmrSubscOutParams{isVirtualTbl: true}, nil
}

var DbToYangPath_sys_aaa_auth_path_xfmr PathXfmrDbToYangFunc = func(inParams XfmrDbToYgPathParams) error {
	rootPath := "/openconfig-system:system/aaa/authentication/users/user"

	log.V(lvl.DEBUG).Info("DbToYangPath_sys_aaa_auth_path_xfmr: inParams: ", inParams)

	switch len(inParams.tblKeyComp) {
	case 2:
		inParams.ygPathKeys[rootPath+"/username"] = inParams.tblKeyComp[1]
	default:
		return fmt.Errorf("Invalid tblKeyCom for intf path xmfr:%v", inParams.tblKeyComp)
	}

	log.V(lvl.DEBUG).Info("DbToYangPath_sys_aaa_auth_path_xfmr:- params.ygPathKeys: ", inParams.ygPathKeys)

	return nil
}

var Subscribe_sys_aaa_auth_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	userName := pathInfo.Var("username")
	log.V(lvl.DEBUG).Infof("Subscribe_sys_aaa_auth_xfmr: pathInfo=%s username=%s", pathInfo, userName)

	accountKey := strings.Join([]string{"SSH_ACCOUNT", userName}, "|")
	consoleKey := strings.Join([]string{"CONSOLE_ACCOUNT", userName}, "|")
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {
				CREDENTIALS_TBL: {
					accountKey: {},
					consoleKey: {}},
			}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

func getSalt(seed []byte) []byte {
	saltRes := "$6$" + string(seed)
	return []byte(saltRes)
}

func getSeed() []byte {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	seed := make([]byte, 16)
	for i := range seed {
		seed[i] = CHARSET[seededRand.Intn(len(CHARSET))]
	}
	return seed
}
func getHashedPassword(userPassword string) (string, error) {
	seededRand := getSeed()
	salt := getSalt(seededRand)
	// use salt to hash user-supplied password
	c := sha512_crypt.New()
	hash, err := c.Generate([]byte(userPassword), salt)
	if err != nil {
		log.V(lvl.DEBUG).Infof("error hashing user's supplied password: %s\n", err)
		return "", err
	}
	return string(hash), nil
}

var DbToYang_sys_aaa_auth_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	userNames := []string{pathInfo.Var("username")}
	log.V(lvl.DEBUG).Infof("DbToYang_sys_aaa_auth_xfmr - uri=%s uername=%v", inParams.uri, userNames)
	stateDb := inParams.dbs[db.StateDB]
	if len(userNames) == 0 || len(userNames[0]) == 0 {
		var err error
		if userNames, err = getAllKeys(stateDb, ACCOUNT_TBL); err != nil {
			return err
		}
	}
	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.Aaa)
	ygot.BuildEmptyTree(sysObj.Aaa.Authentication)
	ygot.BuildEmptyTree(sysObj.Aaa.Authentication.Users)

	for _, userName := range userNames {
		log.V(lvl.DEBUG).Info("userName: ", userName)
		sshEntry, err := stateDb.GetEntry(&db.TableSpec{Name: ACCOUNT_TBL}, db.Key{Comp: []string{userName}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from StateDB %v, username: %v, err: %v", ACCOUNT_TBL, userName, err)
			continue
		}
		var state authUserState
		state.userName = userName
		state.keys.version = sshEntry.Get("keys_version")
		time := sshEntry.Get("keys_created_on")
		if state.keys.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
			log.V(lvl.ERROR).Infof("`keys_created_on` for user:`%v` failed: %v", userName, err)
		}
		state.principals.version = sshEntry.Get("principals_version")
		time = sshEntry.Get("principals_created_on")
		if state.principals.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
			log.V(lvl.ERROR).Infof("`users_created_on` for user:`%v` failed: %v", userName, err)
		}

		userObj, ok := sysObj.Aaa.Authentication.Users.User[userName]
		if !ok {
			userObj, err = sysObj.Aaa.Authentication.Users.NewUser(userName)
			if err != nil {
				log.V(lvl.ERROR).Infof("sysObj.Aaa.Authentication.Users.NewUser(%v) failed: %v", userName, err)
				continue
			}
		}
		ygot.BuildEmptyTree(userObj)
		userObj.State.Username = &state.userName
		userObj.State.AuthorizedKeysListCreatedOn = &state.keys.created
		userObj.State.AuthorizedKeysListVersion = &state.keys.version
		userObj.State.AuthorizedPrincipalsListCreatedOn = &state.principals.created
		userObj.State.AuthorizedPrincipalsListVersion = &state.principals.version

		console, err := stateDb.GetEntry(&db.TableSpec{Name: CONSOLE_TBL}, db.Key{Comp: []string{userName}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from StateDB %v, err: %v", CONSOLE_TBL, err)
			continue
		}
		state.password.version = console.Get("password_version")
		time = console.Get("password_created_on")
		if state.password.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
			log.V(lvl.ERROR).Infof("`password_created_on` for user:`%v` failed: %v", userName, err)
		}
		userObj.State.PasswordCreatedOn = &state.password.created
		userObj.State.PasswordVersion = &state.password.version
	}
	return nil
}

var YangToDb_sys_aaa_auth_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	log.V(lvl.DEBUG).Info("SubtreeXfmrFunc - Uri SYS AUTH: ", inParams.uri)
	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(lvl.DEBUG).Info("TARGET URI PATH SYS AUTH:", targetUriPath)
	sysObj := getAppRootObject(inParams)
	usersObj := sysObj.Aaa.Authentication.Users
	userName := pathInfo.Var("username")
	log.V(lvl.DEBUG).Info("username:", userName)
	if len(userName) == 0 {
		return nil, nil
	}
	var status bool
	var err_str string
	var err error
	if _, _ok := inParams.txCache.Load(userName); !_ok {
		inParams.txCache.Store(userName, userName)
	} else {
		if val, present := inParams.txCache.Load("tx_err"); present {
			return nil, fmt.Errorf("%s", val)
		}
		return nil, nil
	}
	d := inParams.dbs[db.ConfigDB]
	if d == nil {
		d, err = db.NewDB(getDBOptions(db.ConfigDB))
		if err != nil {
			return nil, tlerr.InvalidArgsError{Format: err.Error()}
		}
		defer d.DeleteDB()
	}
	if inParams.oper == DELETE {
		status, err_str = hostAccountUserDel(userName)
		if status {
			var USER_TABLE = "USER"
			userTable := &db.TableSpec{Name: USER_TABLE}
			key := db.Key{Comp: []string{userName}}
			err = d.DeleteEntry(userTable, key)
			if err != nil {
				log.V(lvl.DEBUG).Infof("YangToDb_sys_aaa_auth_xfmr, delete entry error %v", err)
				return nil, err
			}
		}
	} else {
		if value, present := usersObj.User[userName]; present {
			hashedPwd := *(value.Config.PasswordHashed)
			clearPwd := *(value.Config.Password)
			if (len(clearPwd) != 0) && (len(hashedPwd) != 0) {
				errStr := "Clear text password and Hashed password entered for user " + userName
				log.V(lvl.ERROR).Info(errStr)
				return nil, tlerr.InvalidArgsError{Format: errStr}
			}
			if len(clearPwd) != 0 {
				hashedPwd, err = getHashedPassword(clearPwd)
				if err != nil {
					return nil, err
				}
			}
			temp := value.Config.Role.(*ocbinds.OpenconfigSystem_System_Aaa_Authentication_Users_User_Config_Role_Union_String)
			log.V(lvl.DEBUG).Info("Role:", temp.String)
			status, err_str = hostAccountUserMod(*(value.Config.Username), temp.String, hashedPwd)
			if status {
				var USER_TABLE = "USER"
				userTable := &db.TableSpec{Name: USER_TABLE}
				key := db.Key{Comp: []string{*(value.Config.Username)}}
				userInfo := db.Value{Field: map[string]string{}}
				(&userInfo).Set("password", hashedPwd)
				(&userInfo).Set("role@", temp.String)
				err = d.CreateEntry(userTable, key, userInfo)
				if err != nil {
					log.V(lvl.DEBUG).Infof("YangToDb_sys_aaa_auth_xfmr, create entry error %v", err)
					return nil, err
				}
			}
		}
	}
	if !status {
		if _, present := inParams.txCache.Load("tx_err"); !present {
			log.V(lvl.DEBUG).Info("Error in operation:", err_str)
			inParams.txCache.Store("tx_err", err_str)
			return nil, fmt.Errorf("%s", err_str)
		}
	} else {
		return nil, nil
	}
	return nil, nil
}

var Subscribe_ssh_server_state_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	log.V(lvl.INFO).Infof("Subscribe_ssh_server_state_xfmr:%s", inParams.requestURI)

	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"SSH_HOST": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

var DbToYang_ssh_server_state_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var state sshState

	table, err := inParams.dbs[inParams.curDb].GetEntry(&db.TableSpec{Name: "CREDENTIALS"}, db.Key{Comp: []string{"SSH_HOST"}})
	if err != nil {
		log.V(lvl.DEBUG).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	state.caKeys.version = table.Get("ca_keys_version")
	time := table.Get("ca_keys_created_on")
	if state.caKeys.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find ca_keys_created_on: %v", err)
	}
	state.hostKey.version = table.Get("host_key_version")
	time = table.Get("host_key_created_on")
	if state.hostKey.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find host_key_created_on: %v", err)
	}
	state.hostCert.version = table.Get("host_cert_version")
	time = table.Get("host_cert_created_on")
	if state.hostCert.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find host_cert_created_on: %v", err)
	}
	accepts := table.Get("access_accepts")
	if state.counters.accessAccepts, err = strconv.ParseUint(accepts, 10, 64); err != nil && accepts != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find access_accepts: %v", err)
	}
	lastAccept := table.Get("last_access_accept")
	if state.counters.lastAccessAccept, err = strconv.ParseUint(lastAccept, 10, 64); err != nil && lastAccept != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find last_access_accept: %v", err)
	}
	rejects := table.Get("access_rejects")
	if state.counters.accessRejects, err = strconv.ParseUint(rejects, 10, 64); err != nil && rejects != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find access_rejects: %v", err)
	}
	lastReject := table.Get("last_access_reject")
	if state.counters.lastAccessReject, err = strconv.ParseUint(lastReject, 10, 64); err != nil && lastReject != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find last_access_reject: %v", err)
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

var Subscribe_authz_policy_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	log.V(lvl.DEBUG).Infof("Subscribe_authz_policy_xfmr:%s", inParams.requestURI)
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"AUTHZ_POLICY|gnxi": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

var DbToYang_authz_policy_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var state certData

	table, err := inParams.dbs[inParams.curDb].GetEntry(&db.TableSpec{Name: CRED_AUTHZ_TBL}, db.Key{Comp: []string{GNXI_ID}})
	if err != nil {
		log.V(lvl.DEBUG).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	state.version = table.Get("authz_version")
	time := table.Get("authz_created_on")
	if state.created, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find authz_created_on: %v", err)
	}

	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj.Aaa.Authorization.State)

	sysObj.Aaa.Authorization.State.GrpcAuthzPolicyCreatedOn = &state.created
	sysObj.Aaa.Authorization.State.GrpcAuthzPolicyVersion = &state.version

	return nil
}

var DbToYang_grpc_server_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(lvl.DEBUG).Info("DbToYang_grpc_server_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)

	return map[string]interface{}{"name": NewPathInfo(inParams.uri).Var("name")}, nil
}

var Subscribe_grpc_server_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	serverName := pathInfo.Var("name")
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return XfmrSubscOutParams{}, err
	}
	log.V(lvl.DEBUG).Infof("Subscribe_grpc_server_xfmr: targetUriPath: %s name: %s", targetUriPath, serverName)

	var result XfmrSubscOutParams
	if serverName == "" {
		result.dbDataMap = RedisDbSubscribeMap{
			db.ApplStateDB: map[string]map[string]map[string]string{
				"GNPSI": {"global": {}},
			},
			db.StateDB: map[string]map[string]map[string]string{
				CREDENTIALS_TBL: {
					"CERT|gnxi":           {},
					"PATHZ_POLICY|ACTIVE": {}},
			},
		}
	} else if serverName == "gnpsi" {
		result.dbDataMap = RedisDbSubscribeMap{
			db.ApplStateDB: map[string]map[string]map[string]string{
				"GNPSI": {
					"global": {},
				},
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

	if !strings.HasPrefix(targetUriPath, "/openconfig-system:system/grpc-servers/grpc-server/gnmi-pathz-policy-counters") {
		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{mInterval: 0, pType: OnChange}
	}

	return result, nil
}

var DbToYang_grpc_server_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	serverNames := []string{pathInfo.Var("name")}
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		log.V(lvl.ERROR).Infof("Error Parsing Uri Path, err: %v", err)
	}
	if log.V(lvl.DEBUG) {
		log.Info("SubtreeXfmrFunc - Uri SYS AUTH: ", inParams.uri)
		log.Info("TARGET URI PATH SYS AUTH:", targetUriPath)
		log.Info("names:", serverNames)
	}
	stateDb := inParams.dbs[db.StateDB]
	if stateDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr stateDb is nil!")
	}
	applStateDb := inParams.dbs[db.ApplStateDB]
	if applStateDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr applStateDb is nil!")
	}
	if len(serverNames) == 0 || len(serverNames[0]) == 0 {
		var err error
		if serverNames, err = getAllKeys(stateDb, CERT_TBL); err != nil {
			return err
		}
		// Check if GNPSI is configured in APPL_STATE_DB
		// TODO b/347066081: If GNPSI writes to CREDENTIALS|CERT, remove the check of GNPSI table in APPL_STATE_DB
		_, err = applStateDb.GetEntry(&db.TableSpec{Name: "GNPSI"}, db.Key{Comp: []string{"global"}})
		if err == nil {
			serverNames = append(serverNames, GNPSI_ID)
		}

	}
	sysObj := getAppRootObject(inParams)
	ygot.BuildEmptyTree(sysObj)
	ygot.BuildEmptyTree(sysObj.GrpcServers)

	for _, serverName := range serverNames {
		log.V(lvl.DEBUG).Info("serverName: ", serverName)
		var state grpcState
		state.name = serverName

		certzID := GNXI_ID
		certTable, err := stateDb.GetEntry(&db.TableSpec{Name: CERT_TBL}, db.Key{Comp: []string{certzID}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from StateDB %v | %v err: %v", CERT_TBL, certzID, err)
		} else {
			state.certVersion = certTable.Get("certificate_version")
			state.caVersion = certTable.Get("ca_trust_bundle_version")
			state.crlVersion = certTable.Get("certificate_revocation_list_bundle_version")
			state.authPolVersion = certTable.Get("authentication_policy_version")
			state.profileId = certTable.Get("ssl_profile_id")
			time := certTable.Get("certificate_created_on")
			if state.certCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(lvl.ERROR).Infof("Cannot convert `certificate_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("ca_trust_bundle_created_on")
			if state.caCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(lvl.ERROR).Infof("Cannot convert `ca_trust_bundle_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("certificate_revocation_list_bundle_created_on")
			if state.crlCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(lvl.ERROR).Infof("Cannot convert `certificate_revocation_list_bundle_created_on` for %v, err: %v", certzID, err)
			}
			time = certTable.Get("authentication_policy_created_on")
			if state.authPolCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(lvl.ERROR).Infof("Cannot convert `authentication_policy_created_on` for %v, err: %v", certzID, err)
			}
		}

		pathzTable, err := stateDb.GetEntry(&db.TableSpec{Name: CRED_PATHZ_TBL}, db.Key{Comp: []string{"ACTIVE"}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from StateDB %v, err: %v", CRED_PATHZ_TBL, err)
		} else {
			state.pathzVersion = pathzTable.Get("pathz_version")
			time := pathzTable.Get("pathz_created_on")
			if state.pathzCreated, err = strconv.ParseUint(time, 10, 64); err != nil && time != "" {
				log.V(lvl.ERROR).Infof("Cannot convert `pathz_created_on` for %v, err: %v", serverName, err)
			}
		}

		serverObj, ok := sysObj.GrpcServers.GrpcServer[serverName]
		if !ok {
			serverObj, err = sysObj.GrpcServers.NewGrpcServer(serverName)
			if err != nil {
				log.V(lvl.ERROR).Infof("sysObj.GrpcServers.NewGrpcServer(%v) failed: %v", serverName, err)
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

		if serverName == GNPSI_ID {
			if err := processGnpsiPaths(inParams, serverObj); err != nil {
				return err
			}
		}

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
				log.V(lvl.ERROR).Infof("invalid RPC method %s", rpcString)
				continue
			}

			authzPolicyData := getAuthzPolicyCounter(authzTables, serverName, rpcString)
			rpcObj, ok := serverObj.AuthzPolicyCounters.Rpcs.Rpc[rpcString]
			if !ok {
				rpcObj, err = serverObj.AuthzPolicyCounters.Rpcs.NewRpc(rpcString)
				if err != nil {
					log.V(lvl.ERROR).Infof("serverObj.AuthzPolicyCounters.Rpcs.NewRpc(%v) failed: %v", rpcString, err)
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
					log.V(lvl.ERROR).Infof("serverObj.GnmiPathzPolicyCounters.NewPath(%v) failed: %v", xpath, err)
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
					log.V(lvl.ERROR).Infof("Invalid pathz table key: %#v", targetUriPath)
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

func processGnpsiPaths(inParams XfmrParams, serverObj *ocbinds.OpenconfigSystem_System_GrpcServers_GrpcServer) error {
	applStateDb := inParams.dbs[db.ApplStateDB]
	if applStateDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr applStateDb is nil!")
	}
	countersDb := inParams.dbs[db.CountersDB]
	if countersDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr countersDb is nil!")
	}
	configDb := inParams.dbs[db.ConfigDB]
	if configDb == nil {
		return errors.New("DbToYang_grpc_server_xfmr configDb is nil!")
	}
	// Global Config/State session
	gnpsiConfigData := getGnpsiServerData(configDb)
	serverObj.Config.Enable = gnpsiConfigData.enable
	serverObj.Config.Port = gnpsiConfigData.port

	gnpsiStateData := getGnpsiServerData(applStateDb)
	serverObj.State.Enable = gnpsiStateData.enable
	serverObj.State.Port = gnpsiStateData.port

	// Connection session
	addressInPath := NewPathInfo(inParams.uri).Var("address")
	portInPath := NewPathInfo(inParams.uri).Var("port")
	if addressInPath == "" {
		addressInPath = "*"
	}
	if portInPath == "" {
		portInPath = "*"
	}
	if addressInPath == "*" || portInPath == "*" {
		gnpsiCountersKeys, err := countersDb.GetKeysPattern(&db.TableSpec{Name: "COUNTERS"}, db.Key{Comp: []string{"GNPSI", addressInPath + "/" + portInPath}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from GNPSI Counters table err: %v", err)
		}
		for _, key := range gnpsiCountersKeys {
			// Check valid key
			if key.Len() < 1 {
				log.V(lvl.DEBUG).Info("Not a valid GNPSI Counter table")
				continue
			}

			// Ipv4 example: GNPSI:0.0.0.1/4343
			// Ipv6 example: GNPSI:2001:0db8:0000:ff00:0042:7879::1/4343
			var address, port string
			keyLen := key.Len()
			ipAndPort := strings.Split(key.Get(keyLen-1), "/")
			if len(ipAndPort) != 2 {
				log.V(lvl.ERROR).Infof("Invalid address/port format: %v", key.Comp)
				continue
			}
			if keyLen == 2 {
				// Handle Ipv4 key.
				address = ipAndPort[0]
				port = ipAndPort[1]
			} else if keyLen > 2 {
				// Handle Ipv6 key.
				address = strings.Join(key.Comp[1:keyLen-1], ":") + ":" + ipAndPort[0]
				port = ipAndPort[1]
			}

			// Get data from counters DB
			gnpsiCounters, err := countersDb.GetEntry(&db.TableSpec{Name: "COUNTERS"}, key)
			if err != nil {
				return err
			}

			// Construct gnpsi oc tree
			getGrpcConnectionState(serverObj, GNPSI_ID, address, port, gnpsiCounters)
		}
	} else {
		gnpsiCounters, err := countersDb.GetEntry(&db.TableSpec{Name: "COUNTERS"}, db.Key{Comp: []string{"GNPSI", addressInPath + "/" + portInPath}})
		if err != nil {
			return err
		}

		// Construct gnpsi oc tree
		getGrpcConnectionState(serverObj, GNPSI_ID, addressInPath, portInPath, gnpsiCounters)
	}
	return nil
}

func getGnpsiServerData(database *db.DB) gnpsiServer {
	serverData := gnpsiServer{}
	gnpsiTable, err := database.GetEntry(&db.TableSpec{Name: "GNPSI"}, db.Key{Comp: []string{"global"}})
	if err != nil {
		log.V(lvl.DEBUG).Infof("Failed to read from %v GNPSI global table. err: %v", database, err)
		return serverData
	}

	gnpsiEnabled := false
	if gnpsiTable.Get("admin_state") == "ENABLE" {
		gnpsiEnabled = true

	}
	serverData.enable = &gnpsiEnabled

	if portNum, err := strconv.ParseUint(gnpsiTable.Get("port"), 10, 32); err == nil {
		u16Port := uint16(portNum)
		serverData.port = &u16Port
	}

	return serverData
}

func getGrpcConnectionState(serverObj *ocbinds.OpenconfigSystem_System_GrpcServers_GrpcServer, grpcName, address, port string, grpcCounters db.Value) error {
	portNum, err := strconv.ParseUint(port, 10, 32)
	if err != nil {
		return err
	}
	connectionKey := ocbinds.OpenconfigSystem_System_GrpcServers_GrpcServer_Connections_Connection_Key{address, uint16(portNum)}
	ygot.BuildEmptyTree(serverObj.Connections)
	grpcCon, found := serverObj.Connections.Connection[connectionKey]
	if !found {
		grpcCon, err = serverObj.Connections.NewConnection(address, uint16(portNum))
		if err != nil {
			log.V(lvl.DEBUG).Infof("Error creating %s connection", grpcName)
			return err
		}
	}
	ygot.BuildEmptyTree(grpcCon)
	ygot.BuildEmptyTree(grpcCon.State)

	grpcCon.State.Address = &address
	u16Port := uint16(portNum)
	grpcCon.State.Port = &u16Port

	ygot.BuildEmptyTree(grpcCon.State.Counters)
	fieldLeafPairs := []fieldU64LeafPair{
		{"bytes_sent", &grpcCon.State.Counters.BytesSent},
		{"packets_sent", &grpcCon.State.Counters.PacketsSent},
		{"packets_error", &grpcCon.State.Counters.DataSendError},
	}
	return processFieldLeafPairs(&grpcCounters, fieldLeafPairs)
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
			log.V(lvl.DEBUG).Infof("Invalid pathz counter key pattern.")
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

func getAuthzPolicyCounter(authzTables db.Table, server string, rpcString string) map[string]map[string]*uint64 {
	cntMap := make(map[string]*uint64)
	tsMap := make(map[string]*uint64)

	for _, oper := range []string{ACCEPTS, REJECTS} {
		var service string
		var rpc string
		service, rpc, err := getServiceRpc(rpcString)
		if err != nil {
			log.V(lvl.ERROR).Infof("invalid RPC method %s", rpcString)
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
			log.V(lvl.DEBUG).Infof("invalid number of Comps for pathzTableKey %v.", pathzTableKey)
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
			log.V(lvl.DEBUG).Infof("invalid number of Comps for authzTableKey %v.", authzTableKey)
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
		log.V(lvl.DEBUG).Infof("Invalid params for patternGenerator %#v", params)
		return ""
	}

	if params[0] == READS_GET || params[0] == READS_SUB || params[0] == "reads" {
		return "*|reads|" + xpath + "|" + params[1]
	}

	if params[0] == WRITES || params[0] == "writes" {
		return "*|writes|" + xpath + "|" + params[1]
	}

	log.V(lvl.DEBUG).Infof("Invalid operation %v", params[0])
	return ""
}

var DbToYang_pathz_policies_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	pathInfo := NewPathInfo(inParams.uri)
	instances := []string{pathInfo.Var("instance")}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(lvl.DEBUG).Infof("DbToYang_pathz_policies_xfmr: targetUriPath: %s instances: %v", targetUriPath, instances)

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
		log.V(lvl.DEBUG).Infof("instance: %v", instance)
		i, ok := dbToYangPathzInstanceMap[instance]
		if !ok {
			log.V(lvl.ERROR).Infof("Pathz Policy Instance not found: %v", instance)
			continue
		}
		policyObj, ok := sysObj.GnmiPathzPolicies.Policies.Policy[i]
		if !ok {
			var err error
			policyObj, err = sysObj.GnmiPathzPolicies.Policies.NewPolicy(i)
			if err != nil {
				log.V(lvl.ERROR).Infof("sysObj.GnmiPathzPolicies.Policies.NewPolicy failed: %v", err)
				continue
			}
		}
		table, err := stateDb.GetEntry(&db.TableSpec{Name: CRED_PATHZ_TBL}, db.Key{Comp: []string{instance}})
		if err != nil {
			log.V(lvl.ERROR).Infof("Failed to read from StateDB %v, id: %v, err: %v", inParams.table, instance, err)
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

var DbToYang_pathz_policies_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(lvl.DEBUG).Info("DbToYang_pathz_policies_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)

	return map[string]interface{}{"instance": NewPathInfo(inParams.uri).Var("instance")}, nil
}

var Subscribe_pathz_policies_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	pathInfo := NewPathInfo(inParams.uri)
	instance := pathInfo.Var("instance")
	if instance == "" {
		instance = "*"
	}
	targetUriPath, _ := getYangPathFromUri(pathInfo.Path)
	log.V(lvl.DEBUG).Infof("Subscribe_pathz_policies_xfmr: targetUriPath: %s instance: %s", targetUriPath, instance)

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
		log.V(lvl.DEBUG).Infof("Failed to read from StateDB: %v", inParams.table)
		return err
	}

	accepts := table.Get("access_accepts")
	if counters.accessAccepts, err = strconv.ParseUint(accepts, 10, 64); err != nil && accepts != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find access_accepts: %v", err)
	}
	lastAccept := table.Get("last_access_accept")
	if counters.lastAccessAccept, err = strconv.ParseUint(lastAccept, 10, 64); err != nil && lastAccept != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find last_access_accept: %v", err)
	}
	rejects := table.Get("access_rejects")
	if counters.accessRejects, err = strconv.ParseUint(rejects, 10, 64); err != nil && rejects != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find access_rejects: %v", err)
	}
	lastReject := table.Get("last_access_reject")
	if counters.lastAccessReject, err = strconv.ParseUint(lastReject, 10, 64); err != nil && lastReject != "" {
		log.V(lvl.DEBUG).Infof("Couldn't find last_access_reject: %v", err)
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
	log.V(lvl.INFO).Infof("Subscribe_console_counters_xfmr:%s", inParams.requestURI)

	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.StateDB: {"CREDENTIALS": {"CONSOLE_METRICS": {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}

func getAllKeys(sdb *db.DB, tblName string) ([]string, error) {
	tbl, err := sdb.GetTable(&db.TableSpec{Name: tblName})
	if err != nil {
		return nil, fmt.Errorf("Can't get table: %v, err: %v", tblName, err)
	}
	log.V(lvl.DEBUG).Infof("tbl: %v", tbl)
	keys, err := tbl.GetKeys()
	if err != nil {
		return nil, fmt.Errorf("Can't get keys from %v, err: %v", tblName, err)
	}
	log.V(lvl.DEBUG).Infof("tbl keys: %v", keys)
	ret := []string{}
	for _, key := range keys {
		if len(key.Comp) != 3 {
			// This is a fanthom key. Ignore it.
			continue
		}
		ret = append(ret, key.Comp[2])
	}
	log.V(lvl.DEBUG).Infof("keys: %v", ret)
	return ret, nil
}

// 1000 = 100% -> "1.0"
// 10   =   1% -> "0.01"
// 1    =  .1% -> "0.001"
func tenthPercentToDecimalStr(v uint64) string {
	d := v / 1000
	r := (v % 1000)
	if r == 0 {
		return strconv.FormatUint(d, 10)
	}

	padding := ""
	if r < 10 {
		padding = "00"
	} else if r < 100 {
		padding = "0"
	}
	return strings.TrimRight(fmt.Sprintf("%d.%s%d", d, padding, r), "0")
}

// "1"     = 100% -> 1000
// "0.01"  =   1% -> 10
// "0.001" =  .1% -> 1
func percentDecimalStrToTenthPercent(s string) (uint64, error) {
	tokens := strings.Split(s, ".")
	switch len(tokens) {
	case 1:
		u, e := strconv.ParseUint(tokens[0], 10, 64)
		if e != nil {
			return 0, errors.New("Error converting to int: " + s)
		} else {
			return u * 1000, nil
		}
	case 2:
		x := uint64(0)
		switch len(tokens[1]) {
		case 1:
			x = 100
		case 2:
			x = 10
		case 3:
			x = 1
		default:
			return 0, errors.New("Unsupported precision, more than 3 decimal places: " + s)
		}
		u, e := strconv.ParseUint(tokens[0], 10, 64)
		u2, e2 := strconv.ParseUint(tokens[1], 10, 64)
		if e != nil || e2 != nil {
			return 0, errors.New("Error converting to int: " + s)
		}
		return (u*1000 + (u2 * x)), nil
	}
	return 0, errors.New("Unexpected format, multiple dots: " + s)
}

type fl_list struct {
	field string
	leaf  **uint32
}

func doFieldLeafMapping(entry db.Value, flList []fl_list) error {
	for _, fl := range flList {
		if v, ok := entry.Field[fl.field]; ok {
			i, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			u := uint32(i)
			*fl.leaf = &u
		}
	}
	return nil
}

func YangToDb_syslog_server_ip_fld_xfmr(inParams XfmrParams) (map[string]string, error) {
	return make(map[string]string), nil
}

func DbToYang_syslog_server_ip_fld_xfmr(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(lvl.DEBUG).Infof("DbToYang_syslog_server_ip_fld_xfmr: key=\"%s\"", inParams.key)
	result := make(map[string]interface{})
	if len((*inParams.dbDataMap)[inParams.curDb]) > 0 {
		result["host"] = inParams.key
	}
	return result, nil
}

var DbToYangPath_remote_server_path_xfmr PathXfmrDbToYangFunc = func(inParams XfmrDbToYgPathParams) error {
	rootPath := "/openconfig-system:system/logging/remote-servers/remote-server/host"

	log.V(lvl.DEBUG).Infof("DbToYangPath_remote_server_path_xfmr: inParams: %#v", inParams)
	if len(inParams.tblKeyComp) != 1 {
		return fmt.Errorf("Invalid tblKeyCom for DbToYangPath_remote_server_path_xmfr:%v", inParams.tblKeyComp)
	}
	if LOGGING_TBL != inParams.tblName {
		return nil
	}
	inParams.ygPathKeys[rootPath] = inParams.tblKeyComp[0]
	log.V(lvl.DEBUG).Info("DbToYangPath_remote_server_path_xfmr: returned params.ygPathKeys: ", inParams.ygPathKeys)
	return nil
}

var YangToDb_grpc_server_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	sysObj := getAppRootObject(inParams)
	pathInfo := NewPathInfo(inParams.uri)
	grpcServerName := pathInfo.Var("name")

	if strings.Compare(grpcServerName, GNPSI_ID) != 0 {
		return nil, nil
	}

	resMap := make(map[string]map[string]db.Value)

	if inParams.oper == DELETE {
		resMap["GNPSI"] = make(map[string]db.Value)
		return resMap, nil
	}

	gnpsiRes := db.Value{Field: map[string]string{}}
	if sysObj.GrpcServers.GrpcServer[grpcServerName].Config.Enable != nil {
		if *sysObj.GrpcServers.GrpcServer[grpcServerName].Config.Enable {
			gnpsiRes.Field["admin_state"] = "ENABLE"
		} else {
			gnpsiRes.Field["admin_state"] = "DISABLE"
		}
	}
	if sysObj.GrpcServers.GrpcServer[grpcServerName].Config.Port != nil {
		portNum := *sysObj.GrpcServers.GrpcServer[grpcServerName].Config.Port
		gnpsiRes.Field["port"] = strconv.FormatUint(uint64(portNum), 10)
	}

	if len(gnpsiRes.Field) != 0 {
		resMap["GNPSI"] = make(map[string]db.Value)
		resMap["GNPSI"]["global"] = gnpsiRes
	}

	return resMap, nil
}
