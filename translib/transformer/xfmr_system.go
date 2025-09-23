package transformer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	ygot "github.com/openconfig/ygot/ygot"
)

const (
	AUTHZ_TBL = "AUTHZ_TABLE"
	ACCEPTS   = "permitted"
	REJECTS   = "denied"
	GNXI_ID   = "gnxi"
	cntResult = "cntResult"
	tsResult  = "tsResult"

	/** Credential Tables **/
	CREDENTIALS_TBL = "CREDENTIALS"
	CRED_AUTHZ_TBL  = "CREDENTIALS|AUTHZ_POLICY"
	CERT_TBL        = "CREDENTIALS|CERT"

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
)

func init() {
	XlateFuncBind("DbToYang_grpc_server_xfmr", DbToYang_grpc_server_xfmr)
	XlateFuncBind("Subscribe_grpc_server_xfmr", Subscribe_grpc_server_xfmr)
	XlateFuncBind("DbToYang_grpc_server_key_xfmr", DbToYang_grpc_server_key_xfmr)
	XlateFuncBind("DbToYang_authz_policy_xfmr", DbToYang_authz_policy_xfmr)
	XlateFuncBind("Subscribe_authz_policy_xfmr", Subscribe_authz_policy_xfmr)
}

type certData struct {
	version string
	created uint64
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

func getAppRootObject(inParams XfmrParams) *ocbinds.OpenconfigSystem_System {
	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	return deviceObj.System
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

var DbToYang_grpc_server_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	log.V(3).Info("DbToYang_grpc_server_key_xfmr root, uri: ", inParams.ygRoot, inParams.uri)

	return map[string]interface{}{"name": NewPathInfo(inParams.uri).Var("name")}, nil
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
					"CERT|gnxi": {},
				},
			},
		}
	} else {
		result = XfmrSubscOutParams{
			dbDataMap: RedisDbSubscribeMap{
				db.StateDB: map[string]map[string]map[string]string{
					CREDENTIALS_TBL: {
						"CERT|gnxi": {},
					},
				}},
		}
	}

	if !strings.HasPrefix(targetUriPath, "/openconfig-system:system/grpc-servers/grpc-server/gnsi-pathz:gnmi-pathz-policy-counters") {
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
	}
	return nil
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
			// This is a fanthom key. Ignore it.
			continue
		}
		ret = append(ret, key.Comp[2])
	}
	log.V(3).Infof("keys: %v", ret)
	return ret, nil
}
