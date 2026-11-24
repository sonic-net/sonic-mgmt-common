package transformer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	SYSLOG_SERVER_TABLE_NAME                     = "SYSLOG_SERVER"
	SYSLOG_LOGGING_PATH                          = "/openconfig-system:system/logging"
	SYSLOG_SERVERS_PATH                          = "/openconfig-system:system/logging/remote-servers"
	SYSLOG_SERVER_PATH                           = "/openconfig-system:system/logging/remote-servers/remote-server"
	syslogServerKeyField                         = "host"
	SYSLOG_FIELD_PORT                            = "port"
	SYSLOG_FIELD_PROTOCOL                        = "protocol"
	SYSLOG_FIELD_TRANSPORT_SEC                   = "transport_security"
	SYSLOG_FIELD_TLS_PROFILE_ID                  = "tls_profile_id"
	SYSLOG_FIELD_TLS_PERMITTED_PEERS             = "tls_permitted_peers"
	SYSLOG_FIELD_SOURCE                          = "source"
	SYSLOG_FIELD_VRF                             = "vrf"
	SYSLOG_FIELD_SEVERITY                        = "severity"
	SYSLOG_FIELD_FACILITY                        = "facility"
	SYSLOG_DEFAULT_PROFILE                       = "gnxi"
	syslogErrTransportSecurityRequiredForProfile = "transport-security true is required when tls-profile-id is set"
	syslogErrTlsPermittedPeersRequired           = "tls-permitted-peers is required when transport-security is true"
)

var syslogTlsProfileIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var syslogTlsPermittedPeerRe = regexp.MustCompile(`^(\*\.)?[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

var ocSyslogSeverityToDb = map[ocbinds.E_OpenconfigSystemLogging_SyslogSeverity]string{
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_EMERGENCY:     "crit",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_ALERT:         "crit",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_CRITICAL:      "crit",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_ERROR:         "error",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_WARNING:       "warn",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE:        "notice",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_INFORMATIONAL: "info",
	ocbinds.OpenconfigSystemLogging_SyslogSeverity_DEBUG:         "debug",
}

var ocSeverityNameToDb = map[string]string{
	"EMERGENCY":     "crit",
	"ALERT":         "crit",
	"CRITICAL":      "crit",
	"ERROR":         "error",
	"WARNING":       "warn",
	"NOTICE":        "notice",
	"INFORMATIONAL": "info",
	"DEBUG":         "debug",
}

var dbSyslogSeverityToOc = map[string]ocbinds.E_OpenconfigSystemLogging_SyslogSeverity{
	"crit":    ocbinds.OpenconfigSystemLogging_SyslogSeverity_CRITICAL,
	"error":   ocbinds.OpenconfigSystemLogging_SyslogSeverity_ERROR,
	"err":     ocbinds.OpenconfigSystemLogging_SyslogSeverity_ERROR,
	"warn":    ocbinds.OpenconfigSystemLogging_SyslogSeverity_WARNING,
	"warning": ocbinds.OpenconfigSystemLogging_SyslogSeverity_WARNING,
	"notice":  ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
	"info":    ocbinds.OpenconfigSystemLogging_SyslogSeverity_INFORMATIONAL,
	"debug":   ocbinds.OpenconfigSystemLogging_SyslogSeverity_DEBUG,
}

var ocSyslogFacilityToDb = map[ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY]string{
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL:           "*",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_KERNEL:        "kern",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_USER:          "user",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_MAIL:          "mail",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSTEM_DAEMON: "daemon",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_AUTH:          "auth",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSLOG:        "syslog",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL0:        "local0",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1:        "local1",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL2:        "local2",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL3:        "local3",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL4:        "local4",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL5:        "local5",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL6:        "local6",
	ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL7:        "local7",
}

var ocFacilityNameToDb = map[string]string{
	"ALL":           "*",
	"KERNEL":        "kern",
	"USER":          "user",
	"MAIL":          "mail",
	"SYSTEM_DAEMON": "daemon",
	"DAEMON":        "daemon",
	"AUTH":          "auth",
	"SYSLOG":        "syslog",
	"LOCAL0":        "local0",
	"LOCAL1":        "local1",
	"LOCAL2":        "local2",
	"LOCAL3":        "local3",
	"LOCAL4":        "local4",
	"LOCAL5":        "local5",
	"LOCAL6":        "local6",
	"LOCAL7":        "local7",
}

var dbSyslogFacilityToOc = map[string]ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY{
	"*":      ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL,
	"kern":   ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_KERNEL,
	"user":   ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_USER,
	"mail":   ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_MAIL,
	"daemon": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSTEM_DAEMON,
	"auth":   ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_AUTH,
	"syslog": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSLOG,
	"local0": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL0,
	"local1": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1,
	"local2": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL2,
	"local3": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL3,
	"local4": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL4,
	"local5": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL5,
	"local6": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL6,
	"local7": ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL7,
}

func init() {
	XlateFuncBind("YangToDb_remote_servers_list_xfmr", YangToDb_remote_servers_list_xfmr)
	XlateFuncBind("DbToYang_remote_servers_list_xfmr", DbToYang_remote_servers_list_xfmr)
	XlateFuncBind("Subscribe_remote_servers_list_xfmr", Subscribe_remote_servers_list_xfmr)
	XlateFuncBind("YangToDb_syslog_selector_key_xfmr", YangToDb_syslog_selector_key_xfmr)
	XlateFuncBind("DbToYang_syslog_selector_key_xfmr", DbToYang_syslog_selector_key_xfmr)
	XlateFuncBind("YangToDb_syslog_selector_severity_fld_xfmr", YangToDb_syslog_selector_severity_fld_xfmr)
	XlateFuncBind("DbToYang_syslog_selector_severity_fld_xfmr", DbToYang_syslog_selector_severity_fld_xfmr)
	XlateFuncBind("YangToDb_syslog_selector_facility_fld_xfmr", YangToDb_syslog_selector_facility_fld_xfmr)
	XlateFuncBind("DbToYang_syslog_selector_facility_fld_xfmr", DbToYang_syslog_selector_facility_fld_xfmr)
}

func validateSyslogTlsProfileID(profile string) error {
	if profile == SYSLOG_DEFAULT_PROFILE || profile == "." || profile == ".." ||
		!syslogTlsProfileIDRe.MatchString(profile) {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("tls-profile-id %s is not a valid syslog TLS profile", profile),
		}
	}
	return nil
}

func normalizeSyslogTlsPermittedPeers(peers string) (string, error) {
	peers = strings.TrimSpace(peers)
	if peers == "" {
		return "", nil
	}
	parts := strings.Split(peers, ",")
	out := make([]string, 0, len(parts))
	for _, peer := range parts {
		peer = strings.TrimSpace(peer)
		if peer == "" {
			return "", tlerr.InvalidArgsError{Format: "tls-permitted-peers contains an empty entry"}
		}
		if !syslogTlsPermittedPeerRe.MatchString(peer) {
			return "", tlerr.InvalidArgsError{
				Format: fmt.Sprintf("tls-permitted-peers entry %q is not a valid DNS name", peer),
			}
		}
		out = append(out, peer)
	}
	return strings.Join(out, ","), nil
}

func validateSyslogTlsPermittedPeers(transportSecurity bool, peers *string) error {
	if transportSecurity {
		if peers == nil || strings.TrimSpace(*peers) == "" {
			return tlerr.InvalidArgsError{Format: syslogErrTlsPermittedPeersRequired}
		}
		if _, err := normalizeSyslogTlsPermittedPeers(*peers); err != nil {
			return err
		}
	}
	if peers != nil && strings.TrimSpace(*peers) != "" && !transportSecurity {
		return tlerr.InvalidArgsError{Format: "transport-security true is required when tls-permitted-peers is set"}
	}
	return nil
}

func validateSyslogTlsProfile(transportSecurity bool, profileId *string) error {
	if transportSecurity {
		if profileId == nil || *profileId == "" {
			return tlerr.InvalidArgsError{
				Format: "tls-profile-id is required when transport-security is true",
			}
		}
	}
	if profileId != nil && *profileId != "" {
		if err := validateSyslogTlsProfileID(*profileId); err != nil {
			return err
		}
		if !transportSecurity {
			return tlerr.InvalidArgsError{
				Format: syslogErrTransportSecurityRequiredForProfile,
			}
		}
	}
	return nil
}

func syslogTlsProfileIDPtr(profile string) *string {
	if profile == "" {
		return nil
	}
	return &profile
}

// tls-permitted-peers is augmented into remote-server config/state by
// openconfig-system-logging-ext, so a request URI carries the module prefix
// on that element (.../config/openconfig-system-logging-ext:tls-permitted-peers).
// Accept both the prefixed and the unprefixed form.
func isSyslogTlsPermittedPeersLeafPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	return strings.HasSuffix(path, "/tls-permitted-peers") || strings.HasSuffix(path, ":tls-permitted-peers")
}

func syslogTlsPermittedPeersPtr(peers string) *string {
	if peers == "" {
		return nil
	}
	return &peers
}

func syslogTlsPermittedPeersFromParam(param interface{}) (string, bool) {
	switch v := param.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
	}
	return "", false
}

func validateSyslogServerDbFields(fields map[string]string) error {
	tlsEnabled := isSyslogTlsEnabled(fields)
	if err := validateSyslogTlsProfile(tlsEnabled, syslogTlsProfileIDPtr(fields[SYSLOG_FIELD_TLS_PROFILE_ID])); err != nil {
		return err
	}
	return validateSyslogTlsPermittedPeers(tlsEnabled, syslogTlsPermittedPeersPtr(fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS]))
}

func isSyslogTlsEnabled(fields map[string]string) bool {
	if fields == nil {
		return false
	}
	if fields[SYSLOG_FIELD_PROTOCOL] == "tls" {
		return true
	}
	if ts, ok := fields[SYSLOG_FIELD_TRANSPORT_SEC]; ok {
		b, err := strconv.ParseBool(ts)
		return err == nil && b
	}
	return false
}

func syslogTlsProfileIDFromParam(param interface{}) (string, bool) {
	switch v := param.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
	}
	return "", false
}

func syslogTransportSecurityFromParam(param interface{}) (bool, bool) {
	switch v := param.(type) {
	case bool:
		return v, true
	case *bool:
		if v != nil {
			return *v, true
		}
	}
	return false, false
}

func syslogTrimmedString(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func syslogParseHostFromParam(param interface{}) (string, bool) {
	switch val := param.(type) {
	case string:
		s := strings.TrimSpace(val)
		return s, s != ""
	case *string:
		if val != nil {
			s := strings.TrimSpace(*val)
			return s, s != ""
		}
	}
	return "", false
}

func syslogCheckRemoteServerHostKey(targetKey string, server *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer) error {
	if targetKey == "" || server == nil {
		return nil
	}
	if host := syslogTrimmedString(server.Host); host != "" && host != targetKey {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("remote-server host payload (%s) does not match key (%s)", host, targetKey),
		}
	}
	if server.Config != nil {
		if host := syslogTrimmedString(server.Config.Host); host != "" && host != targetKey {
			return tlerr.InvalidArgsError{
				Format: fmt.Sprintf("remote-server config host payload (%s) does not match key (%s)", host, targetKey),
			}
		}
	}
	return nil
}

func syslogCheckRemoteServerHostLeafParam(targetUriPath, targetKey string, param interface{}) error {
	if targetKey == "" || !strings.HasSuffix(targetUriPath, "/host") {
		return nil
	}
	if host, ok := syslogParseHostFromParam(param); ok && host != targetKey {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("remote-server host payload (%s) does not match key (%s)", host, targetKey),
		}
	}
	return nil
}

// syslogPathInfoForTarget picks uri vs requestUri for path/key resolution.
// Subtree transformers run with uri at the mount point (e.g. /remote-servers).
// requestUri is the client path and may be an ancestor (/logging), the list
// (/remote-servers), or a keyed instance (/remote-server[host=...]).
func syslogPathInfoForTarget(inParams XfmrParams) *PathInfo {
	pathInfo := NewPathInfo(inParams.uri)
	if inParams.requestUri == "" || inParams.requestUri == inParams.uri {
		return pathInfo
	}
	reqPathInfo := NewPathInfo(inParams.requestUri)
	// Ancestor or list-level client path: classify from subtree callback uri.
	if reqPathInfo.Var(syslogServerKeyField) == "" {
		return pathInfo
	}
	// Keyed client path while callback uri has no host key.
	if pathInfo.Var(syslogServerKeyField) == "" {
		return reqPathInfo
	}
	reqPath, err := getYangPathFromUri(inParams.requestUri)
	if err != nil {
		return pathInfo
	}
	uriPath, err := getYangPathFromUri(inParams.uri)
	if err != nil {
		return reqPathInfo
	}
	if isMoreSpecificPath(reqPath, uriPath) {
		return reqPathInfo
	}
	return pathInfo
}

func syslogEffectiveUri(inParams XfmrParams) string {
	return syslogPathInfoForTarget(inParams).Path
}

func syslogTargetFromParams(inParams XfmrParams) (string, string, error) {
	pathInfo := syslogPathInfoForTarget(inParams)
	targetUriPath, err := getYangPathFromUri(pathInfo.Path)
	if err != nil {
		return "", "", err
	}
	targetKey := inParams.key
	if targetKey == "" {
		targetKey = pathInfo.Var(syslogServerKeyField)
	}
	return targetUriPath, targetKey, nil
}

func syslogServerFromLeafUpdate(inParams XfmrParams, targetUriPath, host string, servers map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	if servers != nil && servers[host] != nil {
		return servers[host]
	}
	server := &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
		Host: ygot.String(host),
		Config: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
			Host: ygot.String(host),
		},
	}
	switch {
	case strings.HasSuffix(targetUriPath, "/config/tls-profile-id"):
		if profile, ok := syslogTlsProfileIDFromParam(inParams.param); ok {
			server.Config.TlsProfileId = ygot.String(profile)
		}
	case isSyslogTlsPermittedPeersLeafPath(targetUriPath) && strings.Contains(targetUriPath, "/config/"):
		if peers, ok := syslogTlsPermittedPeersFromParam(inParams.param); ok {
			server.Config.TlsPermittedPeers = ygot.String(peers)
		}
	case strings.HasSuffix(targetUriPath, "/config/transport-security"):
		if ts, ok := syslogTransportSecurityFromParam(inParams.param); ok {
			server.Config.TransportSecurity = &ts
		}
	default:
		return servers[host]
	}
	return server
}

func ocSeverityToDb(severity ocbinds.E_OpenconfigSystemLogging_SyslogSeverity) (string, error) {
	if severity == ocbinds.OpenconfigSystemLogging_SyslogSeverity_UNSET {
		return "", nil
	}
	if dbSev, ok := ocSyslogSeverityToDb[severity]; ok {
		return dbSev, nil
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog severity %v", severity)}
}

func dbSeverityToOc(severity string) (ocbinds.E_OpenconfigSystemLogging_SyslogSeverity, bool) {
	ocSev, ok := dbSyslogSeverityToOc[strings.ToLower(severity)]
	return ocSev, ok
}

func ocFacilityToDb(facility ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY) (string, error) {
	if facility == ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_UNSET {
		return "", nil
	}
	if dbFac, ok := ocSyslogFacilityToDb[facility]; ok {
		return dbFac, nil
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog facility %v", facility)}
}

func dbFacilityToOc(facility string) (ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY, bool) {
	facility = strings.ToLower(strings.TrimSpace(facility))
	if facility == "" {
		return ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL, true
	}
	ocFac, ok := dbSyslogFacilityToOc[facility]
	return ocFac, ok
}

func syslogFacilityValueFromParam(param interface{}) (string, error) {
	if param == nil {
		return "", tlerr.InvalidArgsError{Format: "missing syslog facility value"}
	}
	switch v := param.(type) {
	case ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY:
		return ocFacilityToDb(v)
	case string:
		if dbFac, ok := ocFacilityNameToDb[strings.ToUpper(v)]; ok {
			return dbFac, nil
		}
	case *string:
		if v != nil {
			if dbFac, ok := ocFacilityNameToDb[strings.ToUpper(*v)]; ok {
				return dbFac, nil
			}
		}
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog facility value %v", param)}
}

func syslogFacilityFromURI(uri string) (string, error) {
	pathInfo := NewPathInfo(uri)
	facilityName := strings.ToUpper(pathInfo.Var("facility"))
	if facilityName == "" {
		return "", nil
	}
	if dbFac, ok := ocFacilityNameToDb[facilityName]; ok {
		return dbFac, nil
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog facility %s", facilityName)}
}

func syslogSeverityValueFromParam(param interface{}) (string, error) {
	if param == nil {
		return "", tlerr.InvalidArgsError{Format: "missing syslog severity value"}
	}
	switch v := param.(type) {
	case ocbinds.E_OpenconfigSystemLogging_SyslogSeverity:
		return ocSeverityToDb(v)
	case string:
		if dbSev, ok := ocSeverityNameToDb[strings.ToUpper(v)]; ok {
			return dbSev, nil
		}
	case *string:
		if v != nil {
			if dbSev, ok := ocSeverityNameToDb[strings.ToUpper(*v)]; ok {
				return dbSev, nil
			}
		}
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog severity value %v", param)}
}

func syslogSeverityFromURI(uri string) (string, error) {
	pathInfo := NewPathInfo(uri)
	severityName := strings.ToUpper(pathInfo.Var("severity"))
	if severityName == "" {
		return "", nil
	}
	if dbSev, ok := ocSeverityNameToDb[severityName]; ok {
		return dbSev, nil
	}
	return "", tlerr.InvalidArgsError{Format: fmt.Sprintf("unsupported syslog severity %s", severityName)}
}

func applySelectorToFields(fields map[string]string, server *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer, uri string, mergeMode syslogDbMergeMode) error {
	severity := ""
	facility := ""
	var err error
	if server != nil && server.Selectors != nil {
		severity, facility, err = syslogSelectorFromSelectors(server.Selectors)
		if err != nil {
			return err
		}
	}
	selectorLeafReplace := mergeMode == syslogMergeConfigFieldsOnly && isSyslogSelectorLeafPath(uri)
	selectorContainerReplace := mergeMode == syslogMergeConfigFieldsOnly && isSyslogSelectorContainerReplacePath(uri)
	// Leaf REPLACE on config/facility or config/severity must not pull the
	// other selector field from URI list keys; preserve it from CONFIG_DB.
	// Container/list REPLACE (/selectors, /selector, /selector/config) must
	// not use URI keys or DB for omitted fields; omitted leaves are deleted.
	skipSeverityFromURI := selectorLeafReplace && isSyslogSelectorFacilityLeafPath(uri)
	skipFacilityFromURI := selectorLeafReplace && isSyslogSelectorSeverityLeafPath(uri)
	if selectorContainerReplace {
		skipSeverityFromURI = true
		skipFacilityFromURI = true
	}
	if severity == "" && !skipSeverityFromURI {
		severity, err = syslogSeverityFromURI(uri)
		if err != nil {
			return err
		}
	}
	if facility == "" && !skipFacilityFromURI {
		facility, err = syslogFacilityFromURI(uri)
		if err != nil {
			return err
		}
	}
	if severity != "" {
		fields[SYSLOG_FIELD_SEVERITY] = severity
	}
	if facility != "" {
		fields[SYSLOG_FIELD_FACILITY] = facility
	}
	return nil
}

func syslogSelectorFromSelectors(selectors *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors) (string, string, error) {
	if selectors == nil || len(selectors.Selector) == 0 {
		return "", "", nil
	}
	if len(selectors.Selector) > 1 {
		return "", "", tlerr.NotSupportedError{
			Format: "only one selector per remote syslog server is supported",
		}
	}
	for _, selector := range selectors.Selector {
		if selector == nil || selector.Config == nil {
			continue
		}
		dbSev, err := ocSeverityToDb(selector.Config.Severity)
		if err != nil {
			return "", "", err
		}
		dbFac, err := ocFacilityToDb(selector.Config.Facility)
		if err != nil {
			return "", "", err
		}
		return dbSev, dbFac, nil
	}
	return "", "", nil
}

func populateSyslogServerSelectors(serverObj *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer, dbSeverity, dbFacility string, hasSeverity, hasFacility bool) error {
	if !hasSeverity && !hasFacility {
		return nil
	}
	var ocSeverity ocbinds.E_OpenconfigSystemLogging_SyslogSeverity
	if hasSeverity {
		mapped, ok := dbSeverityToOc(dbSeverity)
		if !ok {
			return nil
		}
		ocSeverity = mapped
	}
	var ocFacility ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY
	if hasFacility {
		mapped, ok := dbFacilityToOc(dbFacility)
		if !ok {
			return nil
		}
		ocFacility = mapped
	}
	if serverObj.Selectors == nil {
		serverObj.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
	}
	keyFacility := ocFacility
	if !hasFacility {
		keyFacility = ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_UNSET
	}
	keySeverity := ocSeverity
	if !hasSeverity {
		keySeverity = ocbinds.OpenconfigSystemLogging_SyslogSeverity_UNSET
	}
	selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
		Facility: keyFacility,
		Severity: keySeverity,
	}
	selector, err := serverObj.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
	if err != nil {
		if existing, ok := serverObj.Selectors.Selector[selectorKey]; ok {
			selector = existing
		} else {
			return err
		}
	}
	ygot.BuildEmptyTree(selector)
	if selector.Config == nil {
		selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{}
	}
	if selector.State == nil {
		selector.State = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_State{}
	}
	if hasFacility {
		selector.Facility = ocFacility
		selector.Config.Facility = ocFacility
		selector.State.Facility = ocFacility
	}
	if hasSeverity {
		selector.Severity = ocSeverity
		selector.Config.Severity = ocSeverity
		selector.State.Severity = ocSeverity
	}
	return nil
}

// syslogServerConfigToDbFields copies only YANG leaves that are present. Protocol
// is applied later by applySyslogTlsSemantics. Port is written only when the
// client sets remote-port; existing DB port is otherwise left unchanged.
func syslogServerConfigToDbFields(cfg *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config) map[string]string {
	fields := map[string]string{nullKey: nullKey}
	if cfg == nil {
		return fields
	}
	if cfg.RemotePort != nil {
		fields[SYSLOG_FIELD_PORT] = strconv.Itoa(int(*cfg.RemotePort))
	}
	if cfg.TransportSecurity != nil {
		fields[SYSLOG_FIELD_TRANSPORT_SEC] = strconv.FormatBool(*cfg.TransportSecurity)
	}
	if cfg.TlsProfileId != nil {
		fields[SYSLOG_FIELD_TLS_PROFILE_ID] = *cfg.TlsProfileId
	}
	if cfg.TlsPermittedPeers != nil {
		fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = *cfg.TlsPermittedPeers
	}
	if cfg.SourceAddress != nil {
		fields[SYSLOG_FIELD_SOURCE] = *cfg.SourceAddress
	}
	if cfg.NetworkInstance != nil {
		fields[SYSLOG_FIELD_VRF] = *cfg.NetworkInstance
	}
	return fields
}

// applySyslogTlsSemantics enforces TLS pairing rules and sets protocol/transport_security.
// When tls_profile_id must be dropped, returns delFields for a REPLACE write (CVL-safe).
func applySyslogTlsSemantics(fields map[string]string, cfg *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config, existing map[string]string) (map[string]string, error) {
	delFields := make(map[string]string)
	var existingProfile string
	var existingPeers string
	if existing != nil {
		existingProfile = existing[SYSLOG_FIELD_TLS_PROFILE_ID]
		existingPeers = existing[SYSLOG_FIELD_TLS_PERMITTED_PEERS]
	}

	yangProfile := cfg != nil && cfg.TlsProfileId != nil && *cfg.TlsProfileId != ""
	yangPeers := cfg != nil && cfg.TlsPermittedPeers != nil && *cfg.TlsPermittedPeers != ""
	tsTrue := cfg != nil && cfg.TransportSecurity != nil && *cfg.TransportSecurity
	tsFalse := cfg != nil && cfg.TransportSecurity != nil && !*cfg.TransportSecurity

	// Like remote-port, profile-only UPDATE merges the existing DB row first.
	// tls-profile-id is allowed when transport-security is already true in the
	// payload, merged fields, or existing row.
	if yangProfile && !tsTrue && !isSyslogTlsEnabled(fields) && !isSyslogTlsEnabled(existing) {
		return nil, tlerr.InvalidArgsError{Format: syslogErrTransportSecurityRequiredForProfile}
	}
	if yangPeers && !tsTrue && !isSyslogTlsEnabled(fields) && !isSyslogTlsEnabled(existing) {
		return nil, tlerr.InvalidArgsError{Format: "transport-security true is required when tls-permitted-peers is set"}
	}

	if tsFalse {
		delete(fields, SYSLOG_FIELD_TLS_PROFILE_ID)
		delete(fields, SYSLOG_FIELD_TLS_PERMITTED_PEERS)
		if existingProfile != "" {
			delFields[SYSLOG_FIELD_TLS_PROFILE_ID] = existingProfile
		}
		if existingPeers != "" {
			delFields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = existingPeers
		}
		fields[SYSLOG_FIELD_TRANSPORT_SEC] = "false"
		fields[SYSLOG_FIELD_PROTOCOL] = "udp"
		return delFields, nil
	}

	tlsEnabled := isSyslogTlsEnabled(fields)
	if !tlsEnabled && yangProfile && existing != nil && isSyslogTlsEnabled(existing) {
		tlsEnabled = true
	}
	if tlsEnabled {
		if err := validateSyslogTlsProfile(true, syslogTlsProfileIDPtr(fields[SYSLOG_FIELD_TLS_PROFILE_ID])); err != nil {
			return nil, err
		}
		if err := validateSyslogTlsPermittedPeers(true, syslogTlsPermittedPeersPtr(fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS])); err != nil {
			return nil, err
		}
		fields[SYSLOG_FIELD_TRANSPORT_SEC] = "true"
		fields[SYSLOG_FIELD_PROTOCOL] = "tls"
		return delFields, nil
	}

	if _, ok := fields[SYSLOG_FIELD_PROTOCOL]; !ok {
		fields[SYSLOG_FIELD_PROTOCOL] = "udp"
	}
	return delFields, nil
}

type syslogDbMergeMode int

const (
	syslogMergeNone syslogDbMergeMode = 0
	syslogMergeAll  syslogDbMergeMode = 1
	// REPLACE on the remote-server config container: replace config leaves from
	// payload and preserve selector fields (severity/facility).
	syslogMergeSelectorsOnly syslogDbMergeMode = 2
	// REPLACE on selectors subtree: replace selector leaves from payload and
	// preserve server config fields (port/source/vrf/TLS).
	syslogMergeConfigFieldsOnly syslogDbMergeMode = 3
)

var syslogConfigDbFields = []string{
	SYSLOG_FIELD_PORT,
	SYSLOG_FIELD_SOURCE,
	SYSLOG_FIELD_VRF,
	SYSLOG_FIELD_TRANSPORT_SEC,
	SYSLOG_FIELD_TLS_PROFILE_ID,
	SYSLOG_FIELD_TLS_PERMITTED_PEERS,
	SYSLOG_FIELD_PROTOCOL,
}

var syslogSelectorDbFields = []string{
	SYSLOG_FIELD_SEVERITY,
	SYSLOG_FIELD_FACILITY,
}

// DB field to YANG leaf names for ON_CHANGE subscribe on remote-servers.
// Matches openconfig-system-annot.yang field-name deviations on remote-server
// config/state leaves.
var syslogServerConfigSubscribeFields = map[string]string{
	SYSLOG_FIELD_PORT:                "remote-port",
	SYSLOG_FIELD_SOURCE:              "source-address",
	SYSLOG_FIELD_VRF:                 "network-instance",
	SYSLOG_FIELD_TRANSPORT_SEC:       "transport-security",
	SYSLOG_FIELD_TLS_PROFILE_ID:      "tls-profile-id",
	SYSLOG_FIELD_TLS_PERMITTED_PEERS: "tls-permitted-peers",
}

func syslogServerSubscribeWatchFields(requestURI string) map[string]string {
	subPath := strings.TrimSuffix(requestURI, "/")
	switch {
	case strings.HasSuffix(subPath, "/severity"):
		return map[string]string{SYSLOG_FIELD_SEVERITY: ""}
	case strings.HasSuffix(subPath, "/facility"):
		return map[string]string{SYSLOG_FIELD_FACILITY: ""}
	case strings.Contains(subPath, "/selectors"):
		return map[string]string{
			SYSLOG_FIELD_SEVERITY: "{severity}",
			SYSLOG_FIELD_FACILITY: "{facility}",
		}
	default:
		fields := make(map[string]string, len(syslogServerConfigSubscribeFields)+2)
		for dbFld, yangLeaf := range syslogServerConfigSubscribeFields {
			fields[dbFld] = yangLeaf
		}
		fields[SYSLOG_FIELD_SEVERITY] = "{severity}"
		fields[SYSLOG_FIELD_FACILITY] = "{facility}"
		return fields
	}
}

func isSyslogConfigContainerPath(targetUriPath string) bool {
	path := strings.TrimSuffix(targetUriPath, "/")
	if !strings.HasSuffix(path, "/config") {
		return false
	}
	if strings.Contains(path, "/config/") {
		return false
	}
	return !isSyslogSelectorSubtreePath(path)
}

func isSyslogSelectorSubtreePath(targetUriPath string) bool {
	return strings.Contains(targetUriPath, "/selectors")
}

func isSyslogSelectorFacilityLeafPath(uri string) bool {
	path := strings.TrimSuffix(uri, "/")
	return strings.HasSuffix(path, "/config/facility")
}

func isSyslogSelectorSeverityLeafPath(uri string) bool {
	path := strings.TrimSuffix(uri, "/")
	return strings.HasSuffix(path, "/config/severity")
}

func isSyslogSelectorLeafPath(uri string) bool {
	return isSyslogSelectorFacilityLeafPath(uri) || isSyslogSelectorSeverityLeafPath(uri)
}

// Container/list REPLACE paths under selectors (not individual leaves).
func isSyslogSelectorContainerReplacePath(uri string) bool {
	return isSyslogSelectorSubtreePath(uri) && !isSyslogSelectorLeafPath(uri)
}

func syslogResolveDbMergeMode(oper Operation, targetUriPath string, isListOp, isServerOp bool, targetKey string) syslogDbMergeMode {
	if isListOp || targetKey == "" {
		return syslogMergeNone
	}
	if oper == UPDATE {
		return syslogMergeAll
	}
	if oper != REPLACE {
		return syslogMergeNone
	}
	if isServerOp {
		return syslogMergeNone
	}
	if isSyslogConfigContainerPath(targetUriPath) {
		return syslogMergeSelectorsOnly
	}
	if isSyslogSelectorSubtreePath(targetUriPath) {
		return syslogMergeConfigFieldsOnly
	}
	return syslogMergeAll
}

func syslogPreserveConfigFields(fields, existing map[string]string) {
	for _, k := range syslogConfigDbFields {
		if v, ok := existing[k]; ok {
			fields[k] = v
		}
	}
}

func syslogPreserveSelectorFields(fields, existing map[string]string) {
	if sev, ok := existing[SYSLOG_FIELD_SEVERITY]; ok {
		fields[SYSLOG_FIELD_SEVERITY] = sev
	}
	if fac, ok := existing[SYSLOG_FIELD_FACILITY]; ok {
		fields[SYSLOG_FIELD_FACILITY] = fac
	}
}

func syslogPreserveUnsetSelectorFields(fields, existing map[string]string) {
	if !fieldsHasKey(fields, SYSLOG_FIELD_SEVERITY) {
		if sev, ok := existing[SYSLOG_FIELD_SEVERITY]; ok && sev != "" {
			fields[SYSLOG_FIELD_SEVERITY] = sev
		}
	}
	if !fieldsHasKey(fields, SYSLOG_FIELD_FACILITY) {
		if fac, ok := existing[SYSLOG_FIELD_FACILITY]; ok && fac != "" {
			fields[SYSLOG_FIELD_FACILITY] = fac
		}
	}
}

func syslogConfigReplaceDelFields(existing, fields map[string]string) map[string]string {
	delFields := make(map[string]string)
	for _, k := range syslogConfigDbFields {
		old, hadOld := existing[k]
		if hadOld && !fieldsHasKey(fields, k) {
			delFields[k] = old
		}
	}
	return delFields
}

func syslogSelectorReplaceDelFields(existing, fields map[string]string) map[string]string {
	delFields := make(map[string]string)
	for _, k := range syslogSelectorDbFields {
		old, hadOld := existing[k]
		if hadOld && !fieldsHasKey(fields, k) {
			delFields[k] = old
		}
	}
	return delFields
}

func fieldsHasKey(fields map[string]string, key string) bool {
	_, ok := fields[key]
	return ok
}

func mergeSyslogDelFields(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func syslogServerDbMutationFromYang(cfgDb *db.DB, host string, server *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer, mergeMode syslogDbMergeMode, uri string) (db.Value, map[string]string, error) {
	fields := map[string]string{nullKey: nullKey}
	existing := map[string]string{}
	if cfgDb != nil {
		if entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{host}}); err == nil {
			for k, v := range entry.Field {
				existing[k] = v
			}
		}
	}
	switch mergeMode {
	case syslogMergeAll:
		for k, v := range existing {
			fields[k] = v
		}
	case syslogMergeSelectorsOnly:
		syslogPreserveSelectorFields(fields, existing)
	case syslogMergeConfigFieldsOnly:
		syslogPreserveConfigFields(fields, existing)
	}
	var cfg *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config
	if server != nil && server.Config != nil {
		cfg = server.Config
		for k, v := range syslogServerConfigToDbFields(cfg) {
			if k != nullKey {
				fields[k] = v
			}
		}
	}
	if err := applySelectorToFields(fields, server, uri, mergeMode); err != nil {
		return db.Value{}, nil, err
	}
	if mergeMode == syslogMergeConfigFieldsOnly && isSyslogSelectorLeafPath(uri) {
		syslogPreserveUnsetSelectorFields(fields, existing)
	}
	delFields, err := applySyslogTlsSemantics(fields, cfg, existing)
	if err != nil {
		return db.Value{}, nil, err
	}
	if mergeMode == syslogMergeSelectorsOnly {
		delFields = mergeSyslogDelFields(delFields, syslogConfigReplaceDelFields(existing, fields))
	}
	if mergeMode == syslogMergeConfigFieldsOnly && isSyslogSelectorContainerReplacePath(uri) {
		delFields = mergeSyslogDelFields(delFields, syslogSelectorReplaceDelFields(existing, fields))
	}
	if peers, ok := fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS]; ok {
		normalized, err := normalizeSyslogTlsPermittedPeers(peers)
		if err != nil {
			return db.Value{}, nil, err
		}
		if normalized != "" {
			fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = normalized
		} else {
			delete(fields, SYSLOG_FIELD_TLS_PERMITTED_PEERS)
		}
	}
	if err := validateSyslogServerDbFields(fields); err != nil {
		return db.Value{}, nil, err
	}
	if err := validateSyslogServerFields(cfgDb, host, fields); err != nil {
		return db.Value{}, nil, err
	}
	return db.Value{Field: fields}, delFields, nil
}

func syslogServerDbValueFromYang(cfgDb *db.DB, host string, server *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer, mergeMode syslogDbMergeMode, uri string) (db.Value, error) {
	value, _, err := syslogServerDbMutationFromYang(cfgDb, host, server, mergeMode, uri)
	return value, err
}

func addSyslogYangAuxFields(inParams XfmrParams, host string, fields map[string]string) {
	if inParams.yangAuxValMap == nil || len(fields) == 0 {
		return
	}
	if inParams.yangAuxValMap[SYSLOG_SERVER_TABLE_NAME] == nil {
		inParams.yangAuxValMap[SYSLOG_SERVER_TABLE_NAME] = make(map[string]db.Value)
	}
	entry := inParams.yangAuxValMap[SYSLOG_SERVER_TABLE_NAME][host]
	if entry.Field == nil {
		entry.Field = make(map[string]string)
	}
	for k, v := range fields {
		entry.Field[k] = v
	}
	inParams.yangAuxValMap[SYSLOG_SERVER_TABLE_NAME][host] = entry
}

func storeSyslogServerDbMutation(inParams XfmrParams, cfgDb *db.DB, syslogTableOps map[string]db.Value, host string, server *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer, mergeMode syslogDbMergeMode) error {
	value, delFields, err := syslogServerDbMutationFromYang(cfgDb, host, server, mergeMode, syslogEffectiveUri(inParams))
	if err != nil {
		return err
	}
	if srcIntf := syslogGlobalSrcIntf(inParams, cfgDb); srcIntf != "" {
		if err := validateSyslogSrcIntfForServers(srcIntf, map[string]string{host: value.Field[SYSLOG_FIELD_SOURCE]}); err != nil {
			return err
		}
	}
	if len(delFields) > 0 && inParams.subOpDataMap != nil {
		// translib processCommon applies DELETE sub-ops before UPDATE, so an
		// HDEL of tls_profile_id fails CVL while protocol is still tls.
		// REPLACE uses SetEntry on the same key: HMSET the UDP row and HDEL
		// tls_profile_id in one write. yangAuxValMap keeps this from being
		// treated as a partial terminal-container replace (which would skip HDEL).
		if inParams.yangAuxValMap == nil {
			inParams.yangAuxValMap = make(map[string]map[string]db.Value)
		}
		addSyslogYangAuxFields(inParams, host, delFields)
		updateSubOpDataMap(map[db.DBNum]map[string]map[string]db.Value{
			db.ConfigDB: {
				SYSLOG_SERVER_TABLE_NAME: {
					host: value,
				},
			},
		}, REPLACE, inParams)
		return nil
	}
	syslogTableOps[host] = value
	return nil
}

// Return nil when there are no DELETE table ops. An empty SYSLOG_SERVER map in
// the DELETE result makes translib treat it as "delete entire table".
func syslogReturnDbMutations(syslogTableOps map[string]db.Value) (map[string]map[string]db.Value, error) {
	if len(syslogTableOps) == 0 {
		return nil, nil
	}
	return map[string]map[string]db.Value{
		SYSLOG_SERVER_TABLE_NAME: syslogTableOps,
	}, nil
}

func storeSyslogTransportSecurityDelete(inParams XfmrParams, syslogTableOps map[string]db.Value, host string, entry db.Value) error {
	tsVal, hasTs := entry.Field[SYSLOG_FIELD_TRANSPORT_SEC]
	proto := entry.Field[SYSLOG_FIELD_PROTOCOL]
	if !hasTs && proto != "tls" {
		return tlerr.NotFoundError{Format: "Resource not found"}
	}
	// yangAuxValMap is not passed on DELETE, so we cannot HDEL tls_profile_id
	// atomically with protocol=tls. HDEL transport_security in the main DELETE
	// result and REPLACE protocol=udp when the row is still TLS via protocol.
	if proto == "tls" {
		row := make(map[string]string)
		for k, v := range entry.Field {
			row[k] = v
		}
		row[SYSLOG_FIELD_PROTOCOL] = "udp"
		delete(row, SYSLOG_FIELD_TRANSPORT_SEC)
		updateSubOpDataMap(map[db.DBNum]map[string]map[string]db.Value{
			db.ConfigDB: {
				SYSLOG_SERVER_TABLE_NAME: {
					host: {Field: row},
				},
			},
		}, REPLACE, inParams)
	}
	if !hasTs {
		return nil
	}
	syslogTableOps[host] = db.Value{Field: map[string]string{
		SYSLOG_FIELD_TRANSPORT_SEC: tsVal,
	}}
	return nil
}

// MODIFIED: added source-address, network-instance, and selector severity population (Group A/B).
func populateSyslogServerFromDbEntry(host string, entry db.Value) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	serverObj := &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
		Host: ygot.String(host),
		Config: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
			Host: ygot.String(host),
		},
		State: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_State{
			Host: ygot.String(host),
		},
	}
	if portStr, ok := entry.Field[SYSLOG_FIELD_PORT]; ok {
		if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
			p := uint16(port)
			serverObj.Config.RemotePort = &p
			serverObj.State.RemotePort = &p
		}
	}
	if proto, ok := entry.Field[SYSLOG_FIELD_PROTOCOL]; ok && proto == "tls" {
		ts := true
		serverObj.Config.TransportSecurity = &ts
		serverObj.State.TransportSecurity = &ts
	} else if tsStr, ok := entry.Field[SYSLOG_FIELD_TRANSPORT_SEC]; ok {
		if ts, err := strconv.ParseBool(tsStr); err == nil {
			serverObj.Config.TransportSecurity = &ts
			serverObj.State.TransportSecurity = &ts
		}
	}
	if profile, ok := entry.Field[SYSLOG_FIELD_TLS_PROFILE_ID]; ok {
		serverObj.Config.TlsProfileId = ygot.String(profile)
		serverObj.State.TlsProfileId = ygot.String(profile)
	}
	if peers, ok := entry.Field[SYSLOG_FIELD_TLS_PERMITTED_PEERS]; ok {
		serverObj.Config.TlsPermittedPeers = ygot.String(peers)
		serverObj.State.TlsPermittedPeers = ygot.String(peers)
	}
	if source, ok := entry.Field[SYSLOG_FIELD_SOURCE]; ok {
		serverObj.Config.SourceAddress = ygot.String(source)
		serverObj.State.SourceAddress = ygot.String(source)
	}
	if vrf, ok := entry.Field[SYSLOG_FIELD_VRF]; ok {
		serverObj.Config.NetworkInstance = ygot.String(vrf)
		serverObj.State.NetworkInstance = ygot.String(vrf)
	}
	dbSeverity, hasSeverity := entry.Field[SYSLOG_FIELD_SEVERITY]
	dbFacility, hasFacility := entry.Field[SYSLOG_FIELD_FACILITY]
	if hasSeverity || hasFacility {
		_ = populateSyslogServerSelectors(serverObj, dbSeverity, dbFacility, hasSeverity, hasFacility)
	}
	return serverObj
}

var DbToYang_remote_servers_list_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	log.V(3).Infof("Inside DbToYang_remote_servers_list_xfmr URI: %s, Key: %s", inParams.uri, inParams.key)
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
	if deviceObj.System.Logging.RemoteServers == nil {
		deviceObj.System.Logging.RemoteServers = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers{}
	}
	remoteServersObj := deviceObj.System.Logging.RemoteServers

	targetKeys := make(map[string]bool)
	targetKey := keyFromInParamsOrUri(inParams, syslogServerKeyField)
	if targetKey != "" {
		if _, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{targetKey}}); err != nil {
			remoteServersObj.RemoteServer = make(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer)
			return tlerr.NotFoundError{Format: "Resource not found"}
		}
		targetKeys[targetKey] = true
	} else {
		keys, err := dbGetKeys(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME})
		if err != nil {
			return err
		}
		for _, key := range keys {
			if len(key.Comp) > 0 {
				targetKeys[key.Comp[0]] = true
			}
		}
	}

	newServerMap := make(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer)
	for host := range targetKeys {
		entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{host}})
		if err != nil {
			log.V(3).Infof("No CONFIG_DB entry for syslog server %s: %v", host, err)
			continue
		}
		newServerMap[host] = populateSyslogServerFromDbEntry(host, entry)
	}
	remoteServersObj.RemoteServer = newServerMap
	return nil
}

var Subscribe_remote_servers_list_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	var result XfmrSubscOutParams
	result.dbDataMap = make(RedisDbSubscribeMap)

	// CRUD parent-existence checks invoke this callback without requestURI.
	// Defer validation to the YangToDb/DbToYang subtree so first-time server
	// creation is not rejected when SYSLOG_SERVER|host does not exist yet.
	if inParams.subscProc == TRANSLATE_EXISTS || inParams.requestURI == "" {
		result.isVirtualTbl = true
		return result, nil
	}

	pathInfo := NewPathInfo(inParams.uri)
	host := pathInfo.Var(syslogServerKeyField)
	if host == "" {
		host = "*"
	}

	log.V(3).Infof("Subscribe_remote_servers_list_xfmr: path=%v host=%v proc=%v requestURI=%v",
		inParams.uri, host, inParams.subscProc, inParams.requestURI)

	result.isVirtualTbl = false
	result.onChange = OnchangeEnable
	result.nOpts = &notificationOpts{pType: OnChange}

	watchFields := syslogServerSubscribeWatchFields(inParams.requestURI)

	result.dbDataMap[db.ConfigDB] = map[string]map[string]map[string]string{
		SYSLOG_SERVER_TABLE_NAME: {host: watchFields},
	}
	return result, nil
}

// MODIFIED: added selector-path handling, invokeCRUSubtreeOnce, and uri forwarding (Group B).
var YangToDb_remote_servers_list_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}
	log.V(3).Infof("Inside YangToDb_remote_servers_list_xfmr: op=%v, uri=%s, requestUri=%s",
		inParams.oper, inParams.uri, inParams.requestUri)
	cfgDb := inParams.dbs[db.ConfigDB]
	if cfgDb == nil {
		cfgDb, err = db.NewDB(getDBOptions(db.ConfigDB))
		if err != nil {
			return nil, tlerr.InvalidArgsError{
				Format: fmt.Sprintf("CONFIG_DB not found for URI %s: %v", inParams.uri, err),
			}
		}
		defer cfgDb.DeleteDB()
	}
	sysObj := getAppRootObject(inParams)
	syslogTableOps := make(map[string]db.Value)
	dbMutations := map[string]map[string]db.Value{
		SYSLOG_SERVER_TABLE_NAME: syslogTableOps,
	}
	targetUriPath, targetKey, err := syslogTargetFromParams(inParams)
	if err != nil {
		return nil, tlerr.InternalError{
			Format: fmt.Sprintf("Failed to get YANG path from URI %s: %v", syslogEffectiveUri(inParams), err),
		}
	}
	isListOp := targetUriPath == SYSLOG_SERVERS_PATH ||
		(targetUriPath == SYSLOG_SERVER_PATH && targetKey == "")
	isServerOp := targetUriPath == SYSLOG_SERVER_PATH && targetKey != ""
	isSelectorOp := strings.Contains(targetUriPath, "/selectors")
	isConfigOp := strings.HasSuffix(targetUriPath, "/config") ||
		strings.HasSuffix(targetUriPath, "/host")
	if inParams.oper == DELETE {
		if isListOp {
			keys, err := dbGetKeys(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME})
			if err != nil {
				return nil, tlerr.InternalError{
					Format: fmt.Sprintf("failed to get keys for clear: %v", err),
				}
			}
			for _, key := range keys {
				syslogTableOps[key.Comp[0]] = db.Value{}
			}
		} else if targetKey != "" && isSelectorOp {
			entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{targetKey}})
			if err != nil {
				return nil, tlerr.NotFoundError{Format: "Resource not found"}
			}
			fields := map[string]string{}
			switch {
			case strings.HasSuffix(targetUriPath, "/severity"):
				if sev, ok := entry.Field[SYSLOG_FIELD_SEVERITY]; ok {
					fields[SYSLOG_FIELD_SEVERITY] = sev
				}
			case strings.HasSuffix(targetUriPath, "/facility"):
				if fac, ok := entry.Field[SYSLOG_FIELD_FACILITY]; ok {
					fields[SYSLOG_FIELD_FACILITY] = fac
				}
			case strings.HasSuffix(targetUriPath, "/selectors"),
				strings.HasSuffix(targetUriPath, "/selector"),
				strings.HasSuffix(targetUriPath, "/selector/config"):
				// DELETE selectors container, selector list instance, or
				// selector/config: clear facility+severity only.
				if sev, ok := entry.Field[SYSLOG_FIELD_SEVERITY]; ok {
					fields[SYSLOG_FIELD_SEVERITY] = sev
				}
				if fac, ok := entry.Field[SYSLOG_FIELD_FACILITY]; ok {
					fields[SYSLOG_FIELD_FACILITY] = fac
				}
			default:
				return nil, tlerr.InvalidArgsError{
					Format: fmt.Sprintf("Unsupported selector DELETE on %s", inParams.uri),
				}
			}
			if len(fields) == 0 {
				return nil, tlerr.NotFoundError{Format: "Resource not found"}
			}
			syslogTableOps[targetKey] = db.Value{Field: fields}
		} else if targetKey != "" && strings.Contains(targetUriPath, "/config/") {
			entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{targetKey}})
			if err != nil {
				return nil, tlerr.NotFoundError{Format: "Resource not found"}
			}
			fields := map[string]string{}
			switch {
			case strings.HasSuffix(targetUriPath, "/source-address"):
				if src, ok := entry.Field[SYSLOG_FIELD_SOURCE]; ok {
					fields[SYSLOG_FIELD_SOURCE] = src
				}
			case strings.HasSuffix(targetUriPath, "/network-instance"):
				if vrf, ok := entry.Field[SYSLOG_FIELD_VRF]; ok {
					fields[SYSLOG_FIELD_VRF] = vrf
				}
			case strings.HasSuffix(targetUriPath, "/remote-port"):
				if port, ok := entry.Field[SYSLOG_FIELD_PORT]; ok {
					fields[SYSLOG_FIELD_PORT] = port
				}
			case strings.HasSuffix(targetUriPath, "/transport-security"):
				if err := storeSyslogTransportSecurityDelete(inParams, syslogTableOps, targetKey, entry); err != nil {
					return nil, err
				}
				return syslogReturnDbMutations(syslogTableOps)
			case strings.HasSuffix(targetUriPath, "/tls-profile-id"):
				profile, ok := entry.Field[SYSLOG_FIELD_TLS_PROFILE_ID]
				if !ok || profile == "" {
					return nil, tlerr.NotFoundError{Format: "Resource not found"}
				}
				if isSyslogTlsEnabled(entry.Field) {
					return nil, tlerr.InvalidArgsError{
						Format: "tls-profile-id can be deleted only when transport-security is false or not set",
					}
				}
				fields[SYSLOG_FIELD_TLS_PROFILE_ID] = profile
			case isSyslogTlsPermittedPeersLeafPath(targetUriPath):
				peers, ok := entry.Field[SYSLOG_FIELD_TLS_PERMITTED_PEERS]
				if !ok || peers == "" {
					return nil, tlerr.NotFoundError{Format: "Resource not found"}
				}
				if isSyslogTlsEnabled(entry.Field) {
					return nil, tlerr.InvalidArgsError{
						Format: "tls-permitted-peers can be deleted only when transport-security is false or not set",
					}
				}
				fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = peers
			default:
				return nil, tlerr.InvalidArgsError{
					Format: fmt.Sprintf("Unsupported config DELETE on %s", inParams.uri),
				}
			}
			if len(fields) == 0 {
				return nil, tlerr.NotFoundError{Format: "Resource not found"}
			}
			syslogTableOps[targetKey] = db.Value{Field: fields}
		} else if targetKey != "" && (isServerOp || isConfigOp) {
			syslogTableOps[targetKey] = db.Value{}
		} else {
			return nil, tlerr.InvalidArgsError{
				Format: fmt.Sprintf("Invalid DELETE on %s, targetKey: %s", inParams.uri, targetKey),
			}
		}
		return syslogReturnDbMutations(syslogTableOps)
	}
	var newServersMap map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer
	if sysObj != nil && sysObj.Logging != nil && sysObj.Logging.RemoteServers != nil {
		newServersMap = sysObj.Logging.RemoteServers.RemoteServer
	} else {
		newServersMap = make(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer)
	}
	if inParams.oper == UPDATE || inParams.oper == REPLACE {
		if err := syslogCheckRemoteServerHostLeafParam(targetUriPath, targetKey, inParams.param); err != nil {
			return nil, err
		}
		mergeMode := syslogResolveDbMergeMode(inParams.oper, targetUriPath, isListOp, isServerOp, targetKey)
		if isListOp {
			listMergeMode := mergeMode
			if inParams.oper == REPLACE {
				listMergeMode = syslogMergeNone
			}
			for host, server := range newServersMap {
				if err := syslogCheckRemoteServerHostKey(host, server); err != nil {
					return nil, err
				}
				if err := storeSyslogServerDbMutation(inParams, cfgDb, syslogTableOps, host, server, listMergeMode); err != nil {
					return nil, err
				}
			}
		} else if targetKey != "" {
			if server := newServersMap[targetKey]; server != nil {
				if err := syslogCheckRemoteServerHostKey(targetKey, server); err != nil {
					return nil, err
				}
			}
			if inParams.oper == REPLACE && isServerOp && newServersMap[targetKey] == nil {
				syslogTableOps[targetKey] = db.Value{}
			} else {
				server := syslogServerFromLeafUpdate(inParams, targetUriPath, targetKey, newServersMap)
				if err := storeSyslogServerDbMutation(inParams, cfgDb, syslogTableOps, targetKey, server, mergeMode); err != nil {
					return nil, err
				}
			}
		} else {
			return nil, tlerr.InvalidArgsError{
				Format: fmt.Sprintf("Missing host key for %v on %s", inParams.oper, inParams.uri),
			}
		}
		if inParams.oper == REPLACE && isListOp {
			keys, err := dbGetKeys(cfgDb, &db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME})
			if err != nil {
				return nil, err
			}
			keysToDelete := make(map[string]db.Value)
			for _, key := range keys {
				existingHost := key.Comp[0]
				if _, present := newServersMap[existingHost]; !present {
					keysToDelete[existingHost] = db.Value{}
				}
			}
			if len(keysToDelete) > 0 {
				subOpMap := map[db.DBNum]map[string]map[string]db.Value{
					db.ConfigDB: {
						SYSLOG_SERVER_TABLE_NAME: keysToDelete,
					},
				}
				updateSubOpDataMap(subOpMap, DELETE, inParams)
			}
		}
		return dbMutations, nil
	}
	return nil, tlerr.NotSupportedError{
		Format: fmt.Sprintf("Unsupported operation %v on %s", inParams.oper, inParams.uri),
	}
}

var YangToDb_syslog_selector_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	return keyFromInParamsOrUri(inParams, syslogServerKeyField), nil
}

func syslogSelectorKeysFromDb(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	host := keyFromInParamsOrUri(inParams, syslogServerKeyField)
	if host == "" {
		host = inParams.key
	}
	if host == "" {
		return result, nil
	}

	var severity, facility string
	var hasSeverity, hasFacility bool

	if inParams.dbDataMap != nil {
		data := (*inParams.dbDataMap)[inParams.curDb]
		if serverData, ok := data[SYSLOG_SERVER_TABLE_NAME]; ok {
			if entry, ok := serverData[host]; ok {
				if sev, ok := entry.Field[SYSLOG_FIELD_SEVERITY]; ok && sev != "" {
					severity = sev
					hasSeverity = true
				}
				if fac, ok := entry.Field[SYSLOG_FIELD_FACILITY]; ok && fac != "" {
					facility = fac
					hasFacility = true
				}
			}
		}
	}

	if inParams.d != nil && (!hasSeverity || !hasFacility) {
		entry, err := inParams.d.GetEntry(&db.TableSpec{Name: SYSLOG_SERVER_TABLE_NAME}, db.Key{Comp: []string{host}})
		if err == nil {
			if !hasSeverity {
				if sev, ok := entry.Field[SYSLOG_FIELD_SEVERITY]; ok && sev != "" {
					severity = sev
					hasSeverity = true
				}
			}
			if !hasFacility {
				if fac, ok := entry.Field[SYSLOG_FIELD_FACILITY]; ok && fac != "" {
					facility = fac
					hasFacility = true
				}
			}
		}
	}

	if hasFacility {
		if ocFac, ok := dbFacilityToOc(facility); ok {
			if name, err := ygot.EnumName(ocFac); err == nil {
				result["facility"] = name
			}
		}
	}
	if hasSeverity {
		if ocSev, ok := dbSeverityToOc(severity); ok {
			if name, err := ygot.EnumName(ocSev); err == nil {
				result["severity"] = name
			}
		}
	}
	return result, nil
}

var DbToYang_syslog_selector_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	return syslogSelectorKeysFromDb(inParams)
}

var YangToDb_syslog_selector_severity_fld_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	dbSev, err := syslogSeverityValueFromParam(inParams.param)
	if err != nil {
		return nil, err
	}
	return map[string]string{SYSLOG_FIELD_SEVERITY: dbSev}, nil
}

var DbToYang_syslog_selector_severity_fld_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	data := (*inParams.dbDataMap)[inParams.curDb]
	serverData, ok := data[SYSLOG_SERVER_TABLE_NAME]
	if !ok {
		return result, nil
	}
	entry, ok := serverData[inParams.key]
	if !ok {
		return result, nil
	}
	severity, ok := entry.Field[SYSLOG_FIELD_SEVERITY]
	if !ok {
		return result, nil
	}
	ocSev, ok := dbSeverityToOc(severity)
	if !ok {
		return result, nil
	}
	result["severity"] = ocSev
	return result, nil
}

var YangToDb_syslog_selector_facility_fld_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	dbFac, err := syslogFacilityValueFromParam(inParams.param)
	if err != nil {
		return nil, err
	}
	return map[string]string{SYSLOG_FIELD_FACILITY: dbFac}, nil
}

var DbToYang_syslog_selector_facility_fld_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	data := (*inParams.dbDataMap)[inParams.curDb]
	serverData, ok := data[SYSLOG_SERVER_TABLE_NAME]
	if !ok {
		return result, nil
	}
	entry, ok := serverData[inParams.key]
	if !ok {
		return result, nil
	}
	facility, ok := entry.Field[SYSLOG_FIELD_FACILITY]
	if !ok {
		return result, nil
	}
	ocFac, ok := dbFacilityToOc(facility)
	if !ok {
		return result, nil
	}
	result["facility"] = ocFac
	return result, nil
}
