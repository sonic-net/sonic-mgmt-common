package transformer

import (
	"errors"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
)

const (
	SYSLOG_CONFIG_TABLE_NAME = "SYSLOG_CONFIG"
	SYSLOG_GLOBAL_KEY        = "GLOBAL"
	syslogSrcIntfField       = "src_intf"
)

func init() {
	XlateFuncBind("YangToDb_logging_config_xfmr", YangToDb_logging_config_xfmr)
	XlateFuncBind("DbToYang_logging_config_xfmr", DbToYang_logging_config_xfmr)
	XlateFuncBind("Subscribe_logging_config_xfmr", Subscribe_logging_config_xfmr)
}

// YangToDb_logging_config_xfmr maps the openconfig-system-ext logging
// source-interface leaf to CONFIG_DB SYSLOG_CONFIG|GLOBAL:src_intf. The backend
// (rsyslog-config) resolves the interface to an IP and uses it as the source
// address for remote syslog forwarding (EOS 'logging source-interface').
var YangToDb_logging_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	res := make(map[string]map[string]db.Value)
	log.V(3).Infof("Inside YangToDb_logging_config_xfmr: op=%v, uri=%s", inParams.oper, inParams.uri)

	if inParams.oper == DELETE {
		// Clear only the field this transformer owns; leave the rest of the
		// SYSLOG_CONFIG|GLOBAL row (rate-limit, format, severity) untouched.
		val := db.Value{Field: map[string]string{syslogSrcIntfField: ""}}
		res[SYSLOG_CONFIG_TABLE_NAME] = map[string]db.Value{SYSLOG_GLOBAL_KEY: val}
		return res, nil
	}

	sysObj := getAppRootObject(inParams)
	if sysObj == nil || sysObj.Logging == nil || sysObj.Logging.Config == nil {
		return res, nil
	}
	cfg := sysObj.Logging.Config

	if cfg.SourceInterface == nil {
		return res, nil
	}
	if err := validateSyslogSrcIntfForServers(*cfg.SourceInterface, syslogServersForSrcIntfCheck(inParams, sysObj)); err != nil {
		return nil, err
	}
	val := db.Value{Field: map[string]string{syslogSrcIntfField: *cfg.SourceInterface}}
	res[SYSLOG_CONFIG_TABLE_NAME] = map[string]db.Value{SYSLOG_GLOBAL_KEY: val}
	return res, nil
}

// syslogServersForSrcIntfCheck returns host -> per-server source-address for
// the remote servers in CONFIG_DB, overlaid with any servers in the request.
func syslogServersForSrcIntfCheck(inParams XfmrParams, sysObj *ocbinds.OpenconfigSystem_System) map[string]string {
	servers := make(map[string]string)
	if cfgDb := inParams.dbs[db.ConfigDB]; cfgDb != nil {
		keys, _ := dbGetKeys(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME})
		for _, key := range keys {
			if len(key.Comp) == 0 {
				continue
			}
			host := key.Comp[0]
			servers[host] = ""
			if entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, key); err == nil {
				servers[host] = entry.Get(SYSLOG_FIELD_SOURCE)
			}
		}
	}
	if sysObj != nil && sysObj.Logging != nil && sysObj.Logging.RemoteServers != nil {
		for host, server := range sysObj.Logging.RemoteServers.RemoteServer {
			if _, ok := servers[host]; !ok {
				servers[host] = ""
			}
			if server != nil && server.Config != nil && server.Config.SourceAddress != nil {
				servers[host] = *server.Config.SourceAddress
			}
		}
	}
	return servers
}

// syslogGlobalSrcIntf returns the global logging source-interface, preferring
// a value carried in the same request over the one in CONFIG_DB.
func syslogGlobalSrcIntf(inParams XfmrParams, cfgDb *db.DB) string {
	if inParams.ygRoot != nil {
		if sysObj := getAppRootObject(inParams); sysObj != nil && sysObj.Logging != nil &&
			sysObj.Logging.Config != nil && sysObj.Logging.Config.SourceInterface != nil {
			return *sysObj.Logging.Config.SourceInterface
		}
	}
	if cfgDb == nil {
		return ""
	}
	entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_CONFIG_TABLE_NAME}, db.Key{Comp: []string{SYSLOG_GLOBAL_KEY}})
	if err != nil {
		return ""
	}
	return entry.Get(syslogSrcIntfField)
}

// DbToYang_logging_config_xfmr populates the logging source-interface from
// CONFIG_DB SYSLOG_CONFIG|GLOBAL.
var DbToYang_logging_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	log.V(3).Infof("Inside DbToYang_logging_config_xfmr URI: %s", inParams.uri)
	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		return errors.New("CONFIG_DB not found")
	}

	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	if deviceObj.System == nil {
		deviceObj.System = &ocbinds.OpenconfigSystem_System{}
	}
	if deviceObj.System.Logging == nil {
		deviceObj.System.Logging = &ocbinds.OpenconfigSystem_System_Logging{}
	}
	if deviceObj.System.Logging.Config == nil {
		deviceObj.System.Logging.Config = &ocbinds.OpenconfigSystem_System_Logging_Config{}
	}
	cfg := deviceObj.System.Logging.Config

	entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_CONFIG_TABLE_NAME}, db.Key{Comp: []string{SYSLOG_GLOBAL_KEY}})
	if err != nil {
		// No SYSLOG_CONFIG|GLOBAL row configured yet; nothing to return.
		return nil
	}

	if v := entry.Get(syslogSrcIntfField); v != "" {
		cfg.SourceInterface = &v
	}
	return nil
}

// Subscribe_logging_config_xfmr maps the config subtree to SYSLOG_CONFIG|GLOBAL.
var Subscribe_logging_config_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)
	result.dbDataMap[db.ConfigDB] = map[string]map[string]map[string]string{
		SYSLOG_CONFIG_TABLE_NAME: {SYSLOG_GLOBAL_KEY: {}},
	}
	return result, nil
}
