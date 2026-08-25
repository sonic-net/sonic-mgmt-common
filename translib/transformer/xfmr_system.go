package transformer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	ygot "github.com/openconfig/ygot/ygot"
)

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
