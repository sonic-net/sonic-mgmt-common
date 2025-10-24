package transformer

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	lvl "github.com/Azure/sonic-mgmt-common/translib/log"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
	ygot "github.com/openconfig/ygot/ygot"
)

const (
	ACCEPTS        = "permitted"
	REJECTS        = "denied"
	GNXI_ID        = "gnxi"
	GNPSI_ID       = "gnpsi"
	cntResult      = "cntResult"
	tsResult       = "tsResult"
	SYSMEM_KEY     = "SYS_MEM"
	HOSTNAME_KEY   = "HOSTNAME"
	HOSTCONFIG_KEY = "CONFIG"

	/** Credential Tables **/
	CREDENTIALS_TBL = "CREDENTIALS"
	CERT_TBL        = "CREDENTIALS|CERT"

	/** System Root paths **/
	SYSTEM_ROOT = "/openconfig-system:system"

	GRPC_OC_SERVERS = SYSTEM_ROOT + "/openconfig-system-grpc:grpc-servers"
	GRPC_SERVERS    = SYSTEM_ROOT + "/grpc-servers"
	GRPC_SERVER     = GRPC_OC_SERVERS + "/grpc-server"
)

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

func init() {
	XlateFuncBind("DbToYang_grpc_server_xfmr", DbToYang_grpc_server_xfmr)
	XlateFuncBind("DbToYang_ssh_server_state_xfmr", DbToYang_ssh_server_state_xfmr)
	XlateFuncBind("Subscribe_ssh_server_state_xfmr", Subscribe_ssh_server_state_xfmr)
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

	}

	return nil
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
