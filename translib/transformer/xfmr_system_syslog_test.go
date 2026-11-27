package transformer

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/openconfig/ygot/ygot"
)

const (
	syslogTestHost1          = "10.1.1.1"
	syslogTestHost2          = "10.1.1.2"
	syslogTestHost3          = "100.120.198.176"
	syslogTestProfile        = "siem-east"
	syslogTestPermittedPeers = "syslog.example.com"
)

func newSyslogRemoteServer(host string, transportSecurity *bool, profileID *string, remotePort *uint16, sourceAddress, networkInstance *string) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	server := &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
		Host: ygot.String(host),
		Config: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
			Host: ygot.String(host),
		},
	}
	if transportSecurity != nil {
		server.Config.TransportSecurity = transportSecurity
	}
	if profileID != nil {
		server.Config.TlsProfileId = profileID
	}
	if transportSecurity != nil && *transportSecurity && profileID != nil {
		server.Config.TlsPermittedPeers = ygot.String(syslogTestPermittedPeers)
	}
	if remotePort != nil {
		server.Config.RemotePort = remotePort
	}
	if sourceAddress != nil {
		server.Config.SourceAddress = sourceAddress
	}
	if networkInstance != nil {
		server.Config.NetworkInstance = networkInstance
	}
	return server
}

func newSyslogRemoteServerWithSeverity(host string, severity ocbinds.E_OpenconfigSystemLogging_SyslogSeverity) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	return newSyslogRemoteServerWithSelector(host, ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL, severity)
}

func newSyslogRemoteServerWithFacilityOnly(host string, facility ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	server := newSyslogRemoteServer(host, nil, nil, nil, nil, nil)
	server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
	selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
		Facility: facility,
		Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
	}
	selector, err := server.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
	if err != nil {
		panic(err)
	}
	selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
		Facility: facility,
	}
	return server
}

func newSyslogRemoteServerWithSeverityOnly(host string, severity ocbinds.E_OpenconfigSystemLogging_SyslogSeverity) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	server := newSyslogRemoteServer(host, nil, nil, nil, nil, nil)
	server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
	selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
		Facility: ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL,
		Severity: severity,
	}
	selector, err := server.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
	if err != nil {
		panic(err)
	}
	selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
		Severity: severity,
	}
	return server
}

func newSyslogRemoteServerWithSelector(host string, facility ocbinds.E_OpenconfigSystemLogging_SYSLOG_FACILITY, severity ocbinds.E_OpenconfigSystemLogging_SyslogSeverity) *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer {
	server := newSyslogRemoteServer(host, nil, nil, nil, nil, nil)
	server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
	selector, err := server.Selectors.NewSelector(facility, severity)
	if err != nil {
		panic(err)
	}
	selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
		Facility: facility,
		Severity: severity,
	}
	return server
}

func newSyslogDevice(servers map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer) *ocbinds.Device {
	device := &ocbinds.Device{}
	device.System = &ocbinds.OpenconfigSystem_System{
		Logging: &ocbinds.OpenconfigSystem_System_Logging{
			RemoteServers: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers{
				RemoteServer: servers,
			},
		},
	}
	return device
}

func setupSyslogDbMocks(serverState map[string]db.Value) func() {
	originalDbGetKeys := dbGetKeys
	originalDbGetEntry := dbGetEntry

	dbGetKeys = func(d *db.DB, table *db.TableSpec) ([]db.Key, error) {
		if table.Name == SYSLOG_SERVER_TABLE_NAME {
			keys := make([]db.Key, 0, len(serverState))
			for host := range serverState {
				keys = append(keys, db.Key{Comp: []string{host}})
			}
			return keys, nil
		}
		return nil, errors.New("unexpected table in dbGetKeys")
	}

	dbGetEntry = func(d *db.DB, table *db.TableSpec, key db.Key) (db.Value, error) {
		if len(key.Comp) == 0 {
			return db.Value{}, tlerr.NotFoundError{Format: "empty key"}
		}
		if table.Name == SYSLOG_SERVER_TABLE_NAME {
			if entry, ok := serverState[key.Comp[0]]; ok {
				return entry, nil
			}
		}
		return db.Value{}, tlerr.NotFoundError{Format: "not found"}
	}

	return func() {
		dbGetKeys = originalDbGetKeys
		dbGetEntry = originalDbGetEntry
	}
}

func syslogTlsServerFields(port, protocol, transportSecurity, profileID string) map[string]string {
	fields := map[string]string{
		nullKey:                     nullKey,
		SYSLOG_FIELD_PROTOCOL:       protocol,
		SYSLOG_FIELD_TRANSPORT_SEC:  transportSecurity,
		SYSLOG_FIELD_TLS_PROFILE_ID: profileID,
	}
	if port != "" {
		fields[SYSLOG_FIELD_PORT] = port
	}
	if protocol == "tls" {
		fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = syslogTestPermittedPeers
	}
	return fields
}

func TestIsSyslogTlsPermittedPeersLeafPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{SYSLOG_SERVER_PATH + "/config/tls-permitted-peers", true},
		{SYSLOG_SERVER_PATH + "/config/openconfig-system-logging-ext:tls-permitted-peers", true},
		{SYSLOG_SERVER_PATH + "/config/openconfig-system-logging-ext:tls-permitted-peers/", true},
		{SYSLOG_SERVER_PATH + "/config/tls-profile-id", false},
		{SYSLOG_SERVER_PATH + "/config", false},
	}
	for _, tt := range tests {
		if got := isSyslogTlsPermittedPeersLeafPath(tt.path); got != tt.want {
			t.Errorf("isSyslogTlsPermittedPeersLeafPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeSyslogTlsPermittedPeers(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		want        string
		wantErr     bool
		errContains string
	}{
		{name: "single name", in: "syslog.example.com", want: "syslog.example.com"},
		{name: "list is trimmed", in: " a.example.com , b.example.com ", want: "a.example.com,b.example.com"},
		{name: "leading wildcard label", in: "*.example.com", want: "*.example.com"},
		{name: "empty input", in: "  ", want: ""},
		{name: "empty entry", in: "a.example.com,,b.example.com", wantErr: true, errContains: "empty entry"},
		{name: "space inside name", in: "bad peer", wantErr: true, errContains: "not a valid DNS name"},
		{name: "bare wildcard", in: "*", wantErr: true, errContains: "not a valid DNS name"},
		{name: "leading hyphen", in: "-bad.example.com", wantErr: true, errContains: "not a valid DNS name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeSyslogTlsPermittedPeers(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeSyslogTlsPermittedPeers(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %v, want substring %q", err, tt.errContains)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("normalizeSyslogTlsPermittedPeers(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSyslogServerConfigToDbFields(t *testing.T) {
	tsFalse := false
	tsTrue := true
	port200 := uint16(200)

	tests := []struct {
		name string
		cfg  *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config
		want map[string]string
	}{
		{
			name: "Plain syslog defaults",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				Host:              ygot.String(syslogTestHost1),
				TransportSecurity: &tsFalse,
			},
			want: map[string]string{
				nullKey:                    nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC: "false",
			},
		},
		{
			name: "TLS syslog defaults",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				Host:              ygot.String(syslogTestHost3),
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
			},
			want: syslogTlsServerFields("", "tls", "true", syslogTestProfile),
		},
		{
			name: "TLS explicit port",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				Host:              ygot.String(syslogTestHost3),
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
				RemotePort:        &port200,
			},
			want: syslogTlsServerFields("200", "tls", "true", syslogTestProfile),
		},
		{
			name: "Host only uses plain defaults",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				Host: ygot.String(syslogTestHost1),
			},
			want: map[string]string{
				nullKey: nullKey,
			},
		},
		{
			name: "Syslog with source and vrf",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				Host:            ygot.String(syslogTestHost1),
				SourceAddress:   ygot.String("1.1.1.1"),
				NetworkInstance: ygot.String("mgmt"),
			},
			want: map[string]string{
				nullKey:             nullKey,
				SYSLOG_FIELD_SOURCE: "1.1.1.1",
				SYSLOG_FIELD_VRF:    "mgmt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syslogServerConfigToDbFields(tt.cfg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("syslogServerConfigToDbFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPopulateSyslogServerFromDbEntry(t *testing.T) {
	entry := db.Value{Field: map[string]string{
		SYSLOG_FIELD_PORT:   "6514",
		SYSLOG_FIELD_SOURCE: "10.0.0.1",
		SYSLOG_FIELD_VRF:    "default",
	}}
	server := populateSyslogServerFromDbEntry(syslogTestHost1, entry)

	if server.Config.SourceAddress == nil || *server.Config.SourceAddress != "10.0.0.1" {
		t.Fatalf("unexpected source-address: %v", server.Config.SourceAddress)
	}
	if server.Config.NetworkInstance == nil || *server.Config.NetworkInstance != "default" {
		t.Fatalf("unexpected network-instance: %v", server.Config.NetworkInstance)
	}
	if server.Config.RemotePort == nil || *server.Config.RemotePort != 6514 {
		t.Fatalf("unexpected remote-port: %v", server.Config.RemotePort)
	}
}

func TestPopulateSyslogServerFromDbEntry_Severity(t *testing.T) {
	entry := db.Value{Field: map[string]string{
		SYSLOG_FIELD_SEVERITY: "notice",
	}}
	server := populateSyslogServerFromDbEntry(syslogTestHost1, entry)
	if server.Selectors == nil || len(server.Selectors.Selector) != 1 {
		t.Fatalf("expected one selector, got %#v", server.Selectors)
	}
	for _, selector := range server.Selectors.Selector {
		if selector.Config == nil || selector.Config.Severity != ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE {
			t.Fatalf("unexpected selector severity: %#v", selector.Config)
		}
		if selector.Config.Facility != ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_UNSET {
			t.Fatalf("unexpected selector facility: %v", selector.Config.Facility)
		}
	}
}

func TestPopulateSyslogServerFromDbEntry_FacilityOnly(t *testing.T) {
	entry := db.Value{Field: map[string]string{
		SYSLOG_FIELD_FACILITY: "local4",
	}}
	server := populateSyslogServerFromDbEntry(syslogTestHost1, entry)
	if server.Selectors == nil || len(server.Selectors.Selector) != 1 {
		t.Fatalf("expected one selector, got %#v", server.Selectors)
	}
	for _, selector := range server.Selectors.Selector {
		if selector.Config == nil || selector.Config.Facility != ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL4 {
			t.Fatalf("unexpected selector facility: %#v", selector.Config)
		}
		if selector.Config.Severity != ocbinds.OpenconfigSystemLogging_SyslogSeverity_UNSET {
			t.Fatalf("unexpected selector severity: %v", selector.Config.Severity)
		}
	}
}

func TestPopulateSyslogServerFromDbEntry_Facility(t *testing.T) {
	entry := db.Value{Field: map[string]string{
		SYSLOG_FIELD_SEVERITY: "notice",
		SYSLOG_FIELD_FACILITY: "local1",
	}}
	server := populateSyslogServerFromDbEntry(syslogTestHost1, entry)
	if server.Selectors == nil || len(server.Selectors.Selector) != 1 {
		t.Fatalf("expected one selector, got %#v", server.Selectors)
	}
	for _, selector := range server.Selectors.Selector {
		if selector.Config == nil || selector.Config.Facility != ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1 {
			t.Fatalf("unexpected selector facility: %#v", selector.Config)
		}
	}
}

func TestSyslogSourceValidation(t *testing.T) {
	restore := setSyslogLinuxIPViewForTest(&syslogLinuxIPView{
		vrfList: []string{"mgmt"},
		vrfMembers: map[string][]string{
			"mgmt": {"eth0"},
		},
		intfIPs: map[string][]string{
			"eth0": {"10.10.10.2"},
			"eth1": {"1.1.1.1"},
		},
	})
	defer restore()

	if err := validateSyslogSourceAddress("100.120.198.169", "127.0.0.1"); err == nil {
		t.Fatalf("expected loopback source validation failure")
	}
	if err := validateSyslogServerFields(nil, "100.120.198.169", map[string]string{
		SYSLOG_FIELD_SOURCE: "10.10.10.2",
	}); err == nil {
		t.Fatalf("expected default VRF source validation failure")
	}
	if err := validateSyslogServerFields(nil, "100.120.198.169", map[string]string{
		SYSLOG_FIELD_SOURCE: "1.1.1.1",
	}); err != nil {
		t.Fatalf("expected default VRF source validation success, got %v", err)
	}
	if err := validateSyslogServerFields(nil, "100.120.198.169", map[string]string{
		SYSLOG_FIELD_SOURCE: "10.10.10.2",
		SYSLOG_FIELD_VRF:    "mgmt",
	}); err != nil {
		t.Fatalf("expected mgmt VRF source validation success, got %v", err)
	}
}

func TestSyslogSelectorFieldTransformers(t *testing.T) {
	uri := SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity"
	hostKey, err := YangToDb_syslog_selector_key_xfmr(XfmrParams{uri: uri, key: syslogTestHost1})
	if err != nil || hostKey != syslogTestHost1 {
		t.Fatalf("YangToDb_syslog_selector_key_xfmr() = %q, %v; want %s", hostKey, err, syslogTestHost1)
	}

	severityFields, err := YangToDb_syslog_selector_severity_fld_xfmr(XfmrParams{
		uri:   uri,
		param: "NOTICE",
	})
	if err != nil || severityFields[SYSLOG_FIELD_SEVERITY] != "notice" {
		t.Fatalf("YangToDb_syslog_selector_severity_fld_xfmr() = %v, %v; want notice", severityFields, err)
	}

	facilityURI := SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/facility"
	facilityFields, err := YangToDb_syslog_selector_facility_fld_xfmr(XfmrParams{
		uri:   facilityURI,
		param: "ALL",
	})
	if err != nil || facilityFields[SYSLOG_FIELD_FACILITY] != "*" {
		t.Fatalf("YangToDb_syslog_selector_facility_fld_xfmr() = %v, %v; want facility *", facilityFields, err)
	}

	facilityFields, err = YangToDb_syslog_selector_facility_fld_xfmr(XfmrParams{
		uri:   facilityURI,
		param: "LOCAL1",
	})
	if err != nil || facilityFields[SYSLOG_FIELD_FACILITY] != "local1" {
		t.Fatalf("YangToDb_syslog_selector_facility_fld_xfmr() = %v, %v; want facility local1", facilityFields, err)
	}

	local1URI := SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=NOTICE]/config/severity"
	severityFields, err = YangToDb_syslog_selector_severity_fld_xfmr(XfmrParams{
		uri:   local1URI,
		param: "NOTICE",
	})
	if err != nil || severityFields[SYSLOG_FIELD_SEVERITY] != "notice" {
		t.Fatalf("YangToDb_syslog_selector_severity_fld_xfmr() = %v, %v; want notice", severityFields, err)
	}

	dbDataMap := RedisDbMap{
		db.ConfigDB: map[string]map[string]db.Value{
			SYSLOG_SERVER_TABLE_NAME: {
				syslogTestHost1: {Field: map[string]string{
					SYSLOG_FIELD_FACILITY: "daemon",
				}},
			},
		},
	}
	ocFields, err := DbToYang_syslog_selector_facility_fld_xfmr(XfmrParams{
		curDb:     db.ConfigDB,
		dbDataMap: &dbDataMap,
		key:       syslogTestHost1,
	})
	if err != nil {
		t.Fatalf("DbToYang_syslog_selector_facility_fld_xfmr() error = %v", err)
	}
	if ocFields["facility"] != ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSTEM_DAEMON {
		t.Fatalf("DbToYang_syslog_selector_facility_fld_xfmr() = %v; want DAEMON", ocFields["facility"])
	}

	selectorDbDataMap := RedisDbMap{
		db.ConfigDB: map[string]map[string]db.Value{
			SYSLOG_SERVER_TABLE_NAME: {
				syslogTestHost1: {Field: map[string]string{
					SYSLOG_FIELD_FACILITY: "local4",
					SYSLOG_FIELD_SEVERITY: "warn",
				}},
			},
		},
	}
	selectorKeys, err := DbToYang_syslog_selector_key_xfmr(XfmrParams{
		curDb:     db.ConfigDB,
		dbDataMap: &selectorDbDataMap,
		key:       syslogTestHost1,
	})
	if err != nil {
		t.Fatalf("DbToYang_syslog_selector_key_xfmr() error = %v", err)
	}
	if selectorKeys["facility"] != "LOCAL4" {
		t.Fatalf("DbToYang_syslog_selector_key_xfmr() facility = %v; want LOCAL4", selectorKeys["facility"])
	}
	if selectorKeys["severity"] != "WARNING" {
		t.Fatalf("DbToYang_syslog_selector_key_xfmr() severity = %v; want WARNING", selectorKeys["severity"])
	}
}

func TestSyslogSeverityMapping(t *testing.T) {
	dbSev, err := ocSeverityToDb(ocbinds.OpenconfigSystemLogging_SyslogSeverity_INFORMATIONAL)
	if err != nil || dbSev != "info" {
		t.Fatalf("ocSeverityToDb() = %q, %v; want info", dbSev, err)
	}
	ocSev, ok := dbSeverityToOc("notice")
	if !ok || ocSev != ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE {
		t.Fatalf("dbSeverityToOc() = %v, %v; want NOTICE", ocSev, ok)
	}
	dbSev, err = syslogSeverityFromURI(SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity")
	if err != nil || dbSev != "notice" {
		t.Fatalf("syslogSeverityFromURI() = %q, %v; want notice", dbSev, err)
	}
	dbFac, err := syslogFacilityFromURI(SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=NOTICE]/config/facility")
	if err != nil || dbFac != "local1" {
		t.Fatalf("syslogFacilityFromURI() = %q, %v; want local1", dbFac, err)
	}
	dbFac, err = ocFacilityToDb(ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSLOG)
	if err != nil || dbFac != "syslog" {
		t.Fatalf("ocFacilityToDb() = %q, %v; want syslog", dbFac, err)
	}
	ocFac, ok := dbFacilityToOc("auth")
	if !ok || ocFac != ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_AUTH {
		t.Fatalf("dbFacilityToOc() = %v, %v; want AUTH", ocFac, ok)
	}
}

func TestSyslogServerDbValueFromURI(t *testing.T) {
	uri := SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity"
	value, err := syslogServerDbValueFromYang(nil, syslogTestHost1, nil, syslogMergeNone, uri)
	if err != nil {
		t.Fatalf("syslogServerDbValueFromYang() error = %v", err)
	}
	if value.Field[SYSLOG_FIELD_SEVERITY] != "notice" {
		t.Fatalf("severity = %q, want notice", value.Field[SYSLOG_FIELD_SEVERITY])
	}
	if value.Field[SYSLOG_FIELD_FACILITY] != "*" {
		t.Fatalf("facility = %q, want *", value.Field[SYSLOG_FIELD_FACILITY])
	}

	local1URI := SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=NOTICE]/config/severity"
	value, err = syslogServerDbValueFromYang(nil, syslogTestHost1, nil, syslogMergeNone, local1URI)
	if err != nil {
		t.Fatalf("syslogServerDbValueFromYang() error = %v", err)
	}
	if value.Field[SYSLOG_FIELD_SEVERITY] != "notice" {
		t.Fatalf("severity = %q, want notice", value.Field[SYSLOG_FIELD_SEVERITY])
	}
	if value.Field[SYSLOG_FIELD_FACILITY] != "local1" {
		t.Fatalf("facility = %q, want local1", value.Field[SYSLOG_FIELD_FACILITY])
	}
}

func TestValidateSyslogTlsProfile(t *testing.T) {
	profileID := syslogTestProfile
	gnxi := "gnxi"
	badID := "bad/id"

	tests := []struct {
		name              string
		transportSecurity bool
		profileID         *string
		wantErr           bool
		errContains       string
	}{
		{
			name:              "TLS requires profile id",
			transportSecurity: true,
			profileID:         nil,
			wantErr:           true,
			errContains:       "tls-profile-id is required",
		},
		{
			name:              "TLS rejects gnxi",
			transportSecurity: true,
			profileID:         &gnxi,
			wantErr:           true,
			errContains:       "not a valid syslog TLS profile",
		},
		{
			name:              "TLS rejects bad id",
			transportSecurity: true,
			profileID:         &badID,
			wantErr:           true,
			errContains:       "not a valid syslog TLS profile",
		},
		{
			name:              "TLS accepts Certz profile id",
			transportSecurity: true,
			profileID:         &profileID,
			wantErr:           false,
		},
		{
			name:              "Plain syslog without profile",
			transportSecurity: false,
			profileID:         nil,
			wantErr:           false,
		},
		{
			name:              "Profile without transport-security is rejected",
			transportSecurity: false,
			profileID:         &profileID,
			wantErr:           true,
			errContains:       "transport-security true is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSyslogTlsProfile(tt.transportSecurity, tt.profileID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSyslogTlsProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("validateSyslogTlsProfile() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestApplySyslogTlsSemantics(t *testing.T) {
	tsFalse := false
	tsTrue := true
	port200 := uint16(200)
	newProfile := "siem-west"

	tests := []struct {
		name        string
		cfg         *ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config
		existing    map[string]string
		fields      map[string]string
		wantFields  map[string]string
		wantDel     map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "TLS create leaves port unset",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
			},
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC:       "true",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: syslogTlsServerFields("", "tls", "true", syslogTestProfile),
		},
		{
			name: "TLS create keeps explicit port",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
				RemotePort:        &port200,
			},
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PORT:                "200",
				SYSLOG_FIELD_TRANSPORT_SEC:       "true",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: syslogTlsServerFields("200", "tls", "true", syslogTestProfile),
		},
		{
			name: "TLS enable keeps previously configured port",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
			},
			existing: map[string]string{
				nullKey:               nullKey,
				SYSLOG_FIELD_PORT:     "514",
				SYSLOG_FIELD_PROTOCOL: "udp",
			},
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PORT:                "514",
				SYSLOG_FIELD_PROTOCOL:            "udp",
				SYSLOG_FIELD_TRANSPORT_SEC:       "true",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: syslogTlsServerFields("514", "tls", "true", syslogTestProfile),
		},
		{
			name: "Change profile-id on existing TLS server",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TlsProfileId: ygot.String(newProfile),
			},
			existing: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile),
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PORT:                "6514",
				SYSLOG_FIELD_PROTOCOL:            "tls",
				SYSLOG_FIELD_TRANSPORT_SEC:       "true",
				SYSLOG_FIELD_TLS_PROFILE_ID:      newProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: syslogTlsServerFields("6514", "tls", "true", newProfile),
		},
		{
			name: "Profile-id without transport-security is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TlsProfileId: ygot.String(syslogTestProfile),
			},
			existing: map[string]string{
				nullKey:               nullKey,
				SYSLOG_FIELD_PORT:     "514",
				SYSLOG_FIELD_PROTOCOL: "udp",
			},
			fields: map[string]string{
				nullKey:                     nullKey,
				SYSLOG_FIELD_PORT:           "514",
				SYSLOG_FIELD_PROTOCOL:       "udp",
				SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
			},
			wantErr:     true,
			errContains: "transport-security true is required",
		},
		{
			name: "Transport-security true without profile is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
			},
			fields: map[string]string{
				nullKey:                    nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC: "true",
			},
			wantErr:     true,
			errContains: "tls-profile-id is required",
		},
		{
			name: "Disable TLS unsets profile and leaves port unset",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsFalse,
			},
			existing: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile),
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PROTOCOL:            "tls",
				SYSLOG_FIELD_TRANSPORT_SEC:       "false",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: map[string]string{
				nullKey:                    nullKey,
				SYSLOG_FIELD_PROTOCOL:      "udp",
				SYSLOG_FIELD_TRANSPORT_SEC: "false",
			},
			wantDel: map[string]string{
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
		},
		{
			name: "Transport-security false with profile in payload is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsFalse,
				TlsProfileId:      ygot.String(syslogTestProfile),
			},
			fields: map[string]string{
				nullKey:                     nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC:  "false",
				SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
			},
			wantErr:     true,
			errContains: "transport-security true is required",
		},
		{
			name: "Disable TLS keeps previously configured port",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsFalse,
			},
			existing: syslogTlsServerFields("200", "tls", "true", syslogTestProfile),
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PORT:                "200",
				SYSLOG_FIELD_PROTOCOL:            "tls",
				SYSLOG_FIELD_TRANSPORT_SEC:       "false",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantFields: map[string]string{
				nullKey:                    nullKey,
				SYSLOG_FIELD_PORT:          "200",
				SYSLOG_FIELD_PROTOCOL:      "udp",
				SYSLOG_FIELD_TRANSPORT_SEC: "false",
			},
			wantDel: map[string]string{
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
		},
		{
			name: "TLS without permitted peers is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
			},
			fields: map[string]string{
				nullKey:                     nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC:  "true",
				SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
			},
			wantErr:     true,
			errContains: "tls-permitted-peers is required",
		},
		{
			name: "Permitted peers without transport-security is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TlsPermittedPeers: ygot.String(syslogTestPermittedPeers),
			},
			existing: map[string]string{
				nullKey:               nullKey,
				SYSLOG_FIELD_PORT:     "514",
				SYSLOG_FIELD_PROTOCOL: "udp",
			},
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_PORT:                "514",
				SYSLOG_FIELD_PROTOCOL:            "udp",
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
			},
			wantErr:     true,
			errContains: "transport-security true is required when tls-permitted-peers is set",
		},
		{
			name: "Invalid permitted peer is rejected",
			cfg: &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Config{
				TransportSecurity: &tsTrue,
				TlsProfileId:      ygot.String(syslogTestProfile),
				TlsPermittedPeers: ygot.String("bad peer"),
			},
			fields: map[string]string{
				nullKey:                          nullKey,
				SYSLOG_FIELD_TRANSPORT_SEC:       "true",
				SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
				SYSLOG_FIELD_TLS_PERMITTED_PEERS: "bad peer",
			},
			wantErr:     true,
			errContains: "is not a valid DNS name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]string{}
			for k, v := range tt.fields {
				fields[k] = v
			}
			gotDel, err := applySyslogTlsSemantics(fields, tt.cfg, tt.existing)
			if (err != nil) != tt.wantErr {
				t.Fatalf("applySyslogTlsSemantics() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
					t.Fatalf("applySyslogTlsSemantics() error = %v, want substring %q", err, tt.errContains)
				}
				return
			}
			if !reflect.DeepEqual(fields, tt.wantFields) {
				t.Fatalf("fields = %v, want %v", fields, tt.wantFields)
			}
			if tt.wantDel == nil {
				tt.wantDel = map[string]string{}
			}
			if !reflect.DeepEqual(gotDel, tt.wantDel) {
				t.Fatalf("delFields = %v, want %v", gotDel, tt.wantDel)
			}
		})
	}
}

func TestYangToDb_Syslog_Servers(t *testing.T) {
	tsTrue := true
	tsFalse := false
	source := "1.1.1.1"
	vrf := "mgmt"

	tests := []struct {
		name           string
		op             Operation
		uri            string
		requestUri     string
		key            string
		ygRoot         func() *ocbinds.Device
		serverState    map[string]db.Value
		wantMap        map[string]map[string]db.Value
		wantSubDel     map[string]map[string]db.Value
		wantSubReplace map[string]map[string]db.Value
		wantAux        map[string]map[string]db.Value
		wantErr        bool
	}{
		{
			name: "Server: REPLACE updates keyed server only",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH,
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				port := uint16(518)
				tsTrue := true
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, &tsTrue, ygot.String(syslogTestProfile), &port, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{nullKey: nullKey}},
				syslogTestHost2: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: syslogTlsServerFields("518", "tls", "true", syslogTestProfile)},
				},
			},
		},
		{
			name: "Server: REPLACE rejects host payload mismatch",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH,
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost2, nil, nil, nil, nil, nil),
				})
			},
			wantErr: true,
		},
		{
			name:       "Logging container: REPLACE via requestUri replaces remote-servers list",
			op:         REPLACE,
			uri:        SYSLOG_SERVERS_PATH,
			requestUri: SYSLOG_LOGGING_PATH,
			ygRoot: func() *ocbinds.Device {
				port := uint16(518)
				tsTrue := true
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, &tsTrue, ygot.String(syslogTestProfile), &port, nil, nil),
					syslogTestHost2: newSyslogRemoteServer(syslogTestHost2, &tsTrue, ygot.String(syslogTestProfile), &port, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: syslogTlsServerFields("518", "tls", "true", syslogTestProfile)},
					syslogTestHost2: {Field: syslogTlsServerFields("518", "tls", "true", syslogTestProfile)},
				},
			},
			wantSubDel: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {},
				},
			},
		},
		{
			name:       "Server: REPLACE via requestUri does not prune other servers",
			op:         REPLACE,
			uri:        SYSLOG_SERVERS_PATH,
			requestUri: SYSLOG_SERVER_PATH + "[host=" + syslogTestHost1 + "]",
			ygRoot: func() *ocbinds.Device {
				port := uint16(518)
				tsTrue := true
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, &tsTrue, ygot.String(syslogTestProfile), &port, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{nullKey: nullKey}},
				syslogTestHost2: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: syslogTlsServerFields("518", "tls", "true", syslogTestProfile)},
				},
			},
		},
		{
			name: "List: REPLACE deletes missing servers",
			op:   REPLACE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, &tsTrue, ygot.String(syslogTestProfile), nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{nullKey: nullKey}},
				syslogTestHost2: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: syslogTlsServerFields("", "tls", "true", syslogTestProfile)},
				},
			},
			wantSubDel: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {},
					syslogTestHost2: {},
				},
			},
		},
		{
			name: "List: REPLACE empty payload clears all",
			op:   REPLACE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{nullKey: nullKey}},
				syslogTestHost2: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubDel: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {},
					syslogTestHost2: {},
				},
			},
		},
		{
			name: "List: UPDATE adds without deleting existing",
			op:   UPDATE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, nil, nil, nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost2: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PROTOCOL: "udp",
					}},
				},
			},
		},
		{
			name: "List: REPLACE rejects TLS without profile id",
			op:   REPLACE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, &tsTrue, nil, nil, nil, nil),
				})
			},
			wantErr: true,
		},
		{
			name: "List: REPLACE rejects gnxi profile",
			op:   REPLACE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, &tsTrue, ygot.String("gnxi"), nil, nil, nil),
				})
			},
			wantErr: true,
		},
		{
			name: "UPDATE transport-security false replaces row and drops profile",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, &tsFalse, nil, nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:                    nullKey,
						SYSLOG_FIELD_PORT:          "6514",
						SYSLOG_FIELD_PROTOCOL:      "udp",
						SYSLOG_FIELD_TRANSPORT_SEC: "false",
					}},
				},
			},
			wantAux: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name:   "List: DELETE clears all",
			op:     DELETE,
			uri:    SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device { return &ocbinds.Device{} },
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{nullKey: nullKey}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {},
				},
			},
		},
		{
			name: "Server config with source and vrf",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, nil, nil, nil, &source, &vrf),
				})
			},
			serverState: map[string]db.Value{},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SOURCE:   source,
						SYSLOG_FIELD_VRF:      vrf,
					}},
				},
			},
		},
		{
			name: "Server config rejects source not in default VRF",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				badSource := "10.10.10.2"
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, nil, nil, nil, &badSource, nil),
				})
			},
			serverState: map[string]db.Value{},
			wantErr:     true,
		},
		{
			name: "Selector severity from URI when YANG selectors are absent",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "notice",
						SYSLOG_FIELD_FACILITY: "*",
					}},
				},
			},
		},
		{
			name: "Selector severity maps to per-server SYSLOG_SERVER severity",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServerWithSeverity(syslogTestHost1, ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "notice",
						SYSLOG_FIELD_FACILITY: "*",
					}},
				},
			},
		},
		{
			name: "REPLACE config container clears unset config fields and keeps selectors",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				port := uint16(524)
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, nil, nil, &port, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:                          nullKey,
					SYSLOG_FIELD_PORT:                "520",
					SYSLOG_FIELD_PROTOCOL:            "tls",
					SYSLOG_FIELD_TRANSPORT_SEC:       "true",
					SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
					SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					SYSLOG_FIELD_SOURCE:              "1.1.1.1",
					SYSLOG_FIELD_VRF:                 "mgmt",
					SYSLOG_FIELD_SEVERITY:            "notice",
					SYSLOG_FIELD_FACILITY:            "*",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "524",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "notice",
						SYSLOG_FIELD_FACILITY: "*",
					}},
				},
			},
			wantAux: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_SOURCE:              "1.1.1.1",
						SYSLOG_FIELD_VRF:                 "mgmt",
						SYSLOG_FIELD_TRANSPORT_SEC:       "true",
						SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name: "REPLACE selector instance deletes omitted facility",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				server := newSyslogRemoteServer(syslogTestHost3, nil, nil, nil, nil, nil)
				server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
				selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
					Facility: ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_ALL,
					Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
				}
				selector, err := server.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
				if err != nil {
					panic(err)
				}
				selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
					Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
				}
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: server,
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "6514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "notice",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "6514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "notice",
					}},
				},
			},
			wantAux: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
		},
		{
			name: "REPLACE selector facility deletes omitted severity",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=DEBUG]",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				server := newSyslogRemoteServer(syslogTestHost3, nil, nil, nil, nil, nil)
				server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
				selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
					Facility: ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1,
					Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
				}
				selector, err := server.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
				if err != nil {
					panic(err)
				}
				selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
					Facility: ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1,
				}
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: server,
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "6514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "debug",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "6514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
			wantAux: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_SEVERITY: "debug",
					}},
				},
			},
		},
		{
			name: "REPLACE selector config container deletes omitted facility",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=DEBUG]/config",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				server := newSyslogRemoteServer(syslogTestHost3, nil, nil, nil, nil, nil)
				server.Selectors = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors{}
				selectorKey := ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Key{
					Facility: ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL1,
					Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_DEBUG,
				}
				selector, err := server.Selectors.NewSelector(selectorKey.Facility, selectorKey.Severity)
				if err != nil {
					panic(err)
				}
				selector.Config = &ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer_Selectors_Selector_Config{
					Severity: ocbinds.OpenconfigSystemLogging_SyslogSeverity_ERROR,
				}
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: server,
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "6514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "debug",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "6514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "error",
					}},
				},
			},
			wantAux: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
		},
		{
			name: "REPLACE selector preserves existing server config fields",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServerWithSeverity(syslogTestHost1, ocbinds.OpenconfigSystemLogging_SyslogSeverity_WARNING),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:                          nullKey,
					SYSLOG_FIELD_PORT:                "6514",
					SYSLOG_FIELD_PROTOCOL:            "tls",
					SYSLOG_FIELD_TRANSPORT_SEC:       "true",
					SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
					SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					SYSLOG_FIELD_SOURCE:              "1.1.1.1",
					SYSLOG_FIELD_VRF:                 "mgmt",
					SYSLOG_FIELD_SEVERITY:            "notice",
					SYSLOG_FIELD_FACILITY:            "*",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:                          nullKey,
						SYSLOG_FIELD_PORT:                "6514",
						SYSLOG_FIELD_PROTOCOL:            "tls",
						SYSLOG_FIELD_TRANSPORT_SEC:       "true",
						SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
						SYSLOG_FIELD_SOURCE:              "1.1.1.1",
						SYSLOG_FIELD_VRF:                 "mgmt",
						SYSLOG_FIELD_SEVERITY:            "warn",
						SYSLOG_FIELD_FACILITY:            "*",
					}},
				},
			},
		},
		{
			name: "REPLACE selector config/facility leaf preserves existing severity",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/facility",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServerWithFacilityOnly(
						syslogTestHost3,
						ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_LOCAL2,
					),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "6514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "warn",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "6514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "warn",
						SYSLOG_FIELD_FACILITY: "local2",
					}},
				},
			},
		},
		{
			name: "REPLACE selector config/severity leaf preserves existing facility",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServerWithSeverityOnly(
						syslogTestHost3,
						ocbinds.OpenconfigSystemLogging_SyslogSeverity_WARNING,
					),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "6514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "notice",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "6514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "warn",
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
		},
		{
			name: "Selector maps non-ALL facility to CONFIG_DB",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=SYSLOG][severity=NOTICE]/config/severity",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServerWithSelector(
						syslogTestHost1,
						ocbinds.OpenconfigSystemLogging_SYSLOG_FACILITY_SYSLOG,
						ocbinds.OpenconfigSystemLogging_SyslogSeverity_NOTICE,
					),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						nullKey:               nullKey,
						SYSLOG_FIELD_PORT:     "514",
						SYSLOG_FIELD_PROTOCOL: "udp",
						SYSLOG_FIELD_SEVERITY: "notice",
						SYSLOG_FIELD_FACILITY: "syslog",
					}},
				},
			},
		},
		{
			name: "DELETE selector list instance clears facility and severity only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=DEBUG]",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SOURCE:   "10.0.0.15",
					SYSLOG_FIELD_SEVERITY: "debug",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_SEVERITY: "debug",
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
		},
		{
			name: "DELETE selector state is unsupported",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=DEBUG]/state",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_SEVERITY: "debug",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantErr: true,
		},
		{
			name: "DELETE selector facility clears per-server facility only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=LOCAL1][severity=NOTICE]/config/facility",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "notice",
					SYSLOG_FIELD_FACILITY: "local1",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						SYSLOG_FIELD_FACILITY: "local1",
					}},
				},
			},
		},
		{
			name: "DELETE config source-address clears per-server source only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/source-address",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SOURCE:   "10.0.0.15",
					SYSLOG_FIELD_VRF:      "mgmt",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_SOURCE: "10.0.0.15",
					}},
				},
			},
		},
		{
			name: "DELETE config remote-port clears per-server port only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/remote-port",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SOURCE:   "10.0.0.15",
					SYSLOG_FIELD_VRF:      "mgmt",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_PORT: "514",
					}},
				},
			},
		},
		{
			name: "DELETE config network-instance clears per-server vrf only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/network-instance",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SOURCE:   "10.0.0.15",
					SYSLOG_FIELD_VRF:      "mgmt",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_VRF: "mgmt",
					}},
				},
			},
		},
		{
			name: "DELETE selector severity clears per-server severity only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
					SYSLOG_FIELD_SEVERITY: "notice",
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						SYSLOG_FIELD_SEVERITY: "notice",
					}},
				},
			},
		},
		{
			name: "DELETE transport-security sets protocol udp and keeps profile",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/transport-security",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_TRANSPORT_SEC: "true",
					}},
				},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:                          nullKey,
						SYSLOG_FIELD_PORT:                "6514",
						SYSLOG_FIELD_PROTOCOL:            "udp",
						SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name: "DELETE transport-security on one TLS server leaves siblings untouched",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/transport-security",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_TRANSPORT_SEC: "true",
					}},
				},
			},
			wantSubReplace: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						nullKey:                          nullKey,
						SYSLOG_FIELD_PORT:                "6514",
						SYSLOG_FIELD_PROTOCOL:            "udp",
						SYSLOG_FIELD_TLS_PROFILE_ID:      syslogTestProfile,
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name: "DELETE transport-security on UDP server clears leaf only",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/transport-security",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:                     nullKey,
					SYSLOG_FIELD_PORT:           "514",
					SYSLOG_FIELD_PROTOCOL:       "udp",
					SYSLOG_FIELD_TRANSPORT_SEC:  "false",
					SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: map[string]string{
						SYSLOG_FIELD_TRANSPORT_SEC: "false",
					}},
				},
			},
		},
		{
			name: "DELETE transport-security not found when unset",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/transport-security",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
			},
			wantErr: true,
		},
		{
			name: "DELETE tls-profile-id allowed when transport-security is false",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-profile-id",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:                     nullKey,
					SYSLOG_FIELD_PORT:           "514",
					SYSLOG_FIELD_PROTOCOL:       "udp",
					SYSLOG_FIELD_TRANSPORT_SEC:  "false",
					SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						SYSLOG_FIELD_TLS_PROFILE_ID: syslogTestProfile,
					}},
				},
			},
		},
		{
			name: "DELETE tls-profile-id rejected when transport-security is true",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-profile-id",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantErr: true,
		},
		{
			name: "DELETE tls-permitted-peers allowed when transport-security is false",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-permitted-peers",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:                          nullKey,
					SYSLOG_FIELD_PORT:                "514",
					SYSLOG_FIELD_PROTOCOL:            "udp",
					SYSLOG_FIELD_TRANSPORT_SEC:       "false",
					SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name: "DELETE module-prefixed tls-permitted-peers allowed when transport-security is false",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/openconfig-system-logging-ext:tls-permitted-peers",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost1: {Field: map[string]string{
					nullKey:                          nullKey,
					SYSLOG_FIELD_PORT:                "514",
					SYSLOG_FIELD_PROTOCOL:            "udp",
					SYSLOG_FIELD_TRANSPORT_SEC:       "false",
					SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
				}},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost1: {Field: map[string]string{
						SYSLOG_FIELD_TLS_PERMITTED_PEERS: syslogTestPermittedPeers,
					}},
				},
			},
		},
		{
			name: "DELETE tls-permitted-peers rejected when transport-security is true",
			op:   DELETE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-permitted-peers",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantErr: true,
		},
		{
			name: "UPDATE list profile-only on existing TLS server",
			op:   UPDATE,
			uri:  SYSLOG_SERVERS_PATH,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, nil, ygot.String("asdfghklm"), nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", "asdfghklm")},
				},
			},
		},
		{
			name: "UPDATE tls-profile-id leaf on existing TLS server",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-profile-id",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", "siem-west")},
				},
			},
		},
		{
			name: "UPDATE tls-permitted-peers leaf on existing TLS server",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config/tls-permitted-peers",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: func() map[string]string {
						fields := syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)
						fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = "syslog2.example.com,syslog3.example.com"
						return fields
					}()},
				},
			},
		},
		{
			name: "UPDATE module-prefixed tls-permitted-peers leaf on existing TLS server",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config/openconfig-system-logging-ext:tls-permitted-peers",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(nil)
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)},
			},
			wantMap: map[string]map[string]db.Value{
				SYSLOG_SERVER_TABLE_NAME: {
					syslogTestHost3: {Field: func() map[string]string {
						fields := syslogTlsServerFields("6514", "tls", "true", syslogTestProfile)
						fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] = "syslog2.example.com,syslog3.example.com"
						return fields
					}()},
				},
			},
		},
		{
			name: "CREATE rejects tls-permitted-peers without transport-security",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				server := newSyslogRemoteServer(syslogTestHost1, nil, nil, nil, nil, nil)
				server.Config.TlsPermittedPeers = ygot.String(syslogTestPermittedPeers)
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: server,
				})
			},
			serverState: map[string]db.Value{},
			wantErr:     true,
		},
		{
			name: "REPLACE config container enabling TLS without permitted peers is rejected",
			op:   REPLACE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				tsTrue := true
				server := newSyslogRemoteServer(syslogTestHost3, &tsTrue, ygot.String(syslogTestProfile), nil, nil, nil)
				server.Config.TlsPermittedPeers = nil
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: server,
				})
			},
			serverState: map[string]db.Value{
				syslogTestHost3: {Field: map[string]string{
					nullKey:               nullKey,
					SYSLOG_FIELD_PORT:     "514",
					SYSLOG_FIELD_PROTOCOL: "udp",
				}},
			},
			wantErr: true,
		},
		{
			name: "CREATE rejects tls-profile-id without transport-security",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost1,
			ygRoot: func() *ocbinds.Device {
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost1: newSyslogRemoteServer(syslogTestHost1, nil, ygot.String(syslogTestProfile), nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{},
			wantErr:     true,
		},
		{
			name: "UPDATE rejects transport-security false with tls-profile-id",
			op:   UPDATE,
			uri:  SYSLOG_SERVER_PATH + "/config",
			key:  syslogTestHost3,
			ygRoot: func() *ocbinds.Device {
				tsFalse := false
				return newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
					syslogTestHost3: newSyslogRemoteServer(syslogTestHost3, &tsFalse, ygot.String(syslogTestProfile), nil, nil, nil),
				})
			},
			serverState: map[string]db.Value{},
			wantErr:     true,
		},
	}

	restoreIPView := setSyslogLinuxIPViewForTest(&syslogLinuxIPView{
		vrfList: []string{"mgmt"},
		vrfMembers: map[string][]string{
			"mgmt": {"eth0"},
		},
		intfIPs: map[string][]string{
			"eth1": {"1.1.1.1"},
		},
	})
	defer restoreIPView()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := setupSyslogDbMocks(tt.serverState)
			defer restore()

			dbs := [db.MaxDB]*db.DB{db.ConfigDB: &db.DB{}}
			ygRoot := ygot.GoStruct(tt.ygRoot())
			inParams := XfmrParams{
				dbs:           dbs,
				ygRoot:        &ygRoot,
				oper:          tt.op,
				uri:           tt.uri,
				requestUri:    tt.requestUri,
				key:           tt.key,
				subOpDataMap:  make(map[Operation]*map[db.DBNum]map[string]map[string]db.Value),
				yangAuxValMap: make(map[string]map[string]db.Value),
			}
			switch tt.name {
			case "UPDATE tls-profile-id leaf on existing TLS server":
				inParams.param = "siem-west"
			case "UPDATE tls-permitted-peers leaf on existing TLS server",
				"UPDATE module-prefixed tls-permitted-peers leaf on existing TLS server":
				inParams.param = "syslog2.example.com,syslog3.example.com"
			}

			gotMap, err := YangToDb_remote_servers_list_xfmr(inParams)
			if (err != nil) != tt.wantErr {
				t.Fatalf("YangToDb_remote_servers_list_xfmr() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(gotMap, tt.wantMap) {
				t.Fatalf("YangToDb_remote_servers_list_xfmr() gotMap = %v, wantMap %v", gotMap, tt.wantMap)
			}
			if tt.wantSubDel != nil {
				delOps := inParams.subOpDataMap[DELETE]
				if delOps == nil || (*delOps)[db.ConfigDB] == nil {
					t.Fatalf("Expected DELETE sub operations, got none")
				}
				gotSubDel := (*delOps)[db.ConfigDB]
				if !reflect.DeepEqual(gotSubDel, tt.wantSubDel) {
					t.Fatalf("SubOpMap DELETE got = %v, want %v", gotSubDel, tt.wantSubDel)
				}
			}
			if tt.wantSubReplace != nil {
				replOps := inParams.subOpDataMap[REPLACE]
				if replOps == nil || (*replOps)[db.ConfigDB] == nil {
					t.Fatalf("Expected REPLACE sub operations, got none")
				}
				gotSubReplace := (*replOps)[db.ConfigDB]
				if !reflect.DeepEqual(gotSubReplace, tt.wantSubReplace) {
					t.Fatalf("SubOpMap REPLACE got = %v, want %v", gotSubReplace, tt.wantSubReplace)
				}
			}
			if tt.wantAux != nil {
				if !reflect.DeepEqual(inParams.yangAuxValMap, tt.wantAux) {
					t.Fatalf("yangAuxValMap got = %v, want %v", inParams.yangAuxValMap, tt.wantAux)
				}
			}
		})
	}
}

func TestSubscribe_Syslog_RemoteServers(t *testing.T) {
	assertOnChange := func(t *testing.T, out XfmrSubscOutParams) {
		if out.isVirtualTbl {
			t.Fatalf("isVirtualTbl=true, want false for ON_CHANGE subscribe")
		}
		if out.onChange != OnchangeEnable {
			t.Fatalf("onChange=%v, want OnchangeEnable", out.onChange)
		}
		if out.nOpts == nil || out.nOpts.pType != OnChange {
			t.Fatalf("nOpts=%v, want OnChange notification type", out.nOpts)
		}
	}

	t.Run("CRUD parent check is virtual", func(t *testing.T) {
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        SYSLOG_SERVER_PATH + "/selectors",
			requestURI: "",
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		if !out.isVirtualTbl {
			t.Fatalf("expected virtual table when requestURI is empty")
		}
	})

	t.Run("Subscribe all servers", func(t *testing.T) {
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        SYSLOG_SERVERS_PATH,
			requestURI: SYSLOG_SERVERS_PATH,
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		assertOnChange(t, out)
		fields, ok := out.dbDataMap[db.ConfigDB][SYSLOG_SERVER_TABLE_NAME]["*"]
		if !ok {
			t.Fatalf("missing wildcard SYSLOG_SERVER watch: %v", out.dbDataMap)
		}
		for dbFld, yangLeaf := range syslogServerConfigSubscribeFields {
			if fields[dbFld] != yangLeaf {
				t.Fatalf("field map[%s]=%q, want %q; got %v", dbFld, fields[dbFld], yangLeaf, fields)
			}
		}
		if fields[SYSLOG_FIELD_TLS_PERMITTED_PEERS] != "tls-permitted-peers" {
			t.Fatalf("missing tls-permitted-peers watch: %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_SEVERITY]; !ok {
			t.Fatalf("missing severity watch: %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_FACILITY]; !ok {
			t.Fatalf("missing facility watch: %v", fields)
		}
	})

	t.Run("Subscribe specific server selectors container", func(t *testing.T) {
		uri := SYSLOG_SERVER_PATH + "[host=" + syslogTestHost1 + "]/selectors"
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        uri,
			requestURI: uri,
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		assertOnChange(t, out)
		if _, ok := out.dbDataMap[db.ConfigDB][SYSLOG_SERVER_TABLE_NAME][syslogTestHost1]; !ok {
			t.Fatalf("missing keyed SYSLOG_SERVER watch: %v", out.dbDataMap)
		}
	})

	t.Run("Subscribe selectors with wildcard list keys", func(t *testing.T) {
		uri := SYSLOG_SERVER_PATH + "[host=" + syslogTestHost1 + "]/selectors/selector"
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        uri,
			requestURI: uri,
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		assertOnChange(t, out)
		fields := out.dbDataMap[db.ConfigDB][SYSLOG_SERVER_TABLE_NAME][syslogTestHost1]
		if fields[SYSLOG_FIELD_SEVERITY] != "{severity}" || fields[SYSLOG_FIELD_FACILITY] != "{facility}" {
			t.Fatalf("expected selector key placeholders, got %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_SEVERITY]; !ok {
			t.Fatalf("missing severity watch on selector subscribe: %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_FACILITY]; !ok {
			t.Fatalf("missing facility watch on selector subscribe: %v", fields)
		}
	})

	t.Run("Subscribe selector severity leaf", func(t *testing.T) {
		uri := SYSLOG_SERVER_PATH + "[host=" + syslogTestHost1 + "]/selectors"
		requestURI := SYSLOG_SERVER_PATH + "/selectors/selector[facility=ALL][severity=NOTICE]/config/severity"
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        uri,
			requestURI: requestURI,
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		assertOnChange(t, out)
		fields := out.dbDataMap[db.ConfigDB][SYSLOG_SERVER_TABLE_NAME][syslogTestHost1]
		if _, ok := fields[SYSLOG_FIELD_SEVERITY]; !ok {
			t.Fatalf("missing severity watch: %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_FACILITY]; ok {
			t.Fatalf("did not expect facility watch on severity leaf: %v", fields)
		}
	})

	t.Run("Subscribe wildcard host selectors", func(t *testing.T) {
		uri := SYSLOG_SERVER_PATH + "/selectors/selector"
		out, err := Subscribe_remote_servers_list_xfmr(XfmrSubscInParams{
			uri:        uri,
			requestURI: uri,
			subscProc:  TRANSLATE_SUBSCRIBE,
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		assertOnChange(t, out)
		fields := out.dbDataMap[db.ConfigDB][SYSLOG_SERVER_TABLE_NAME]["*"]
		if _, ok := fields[SYSLOG_FIELD_SEVERITY]; !ok {
			t.Fatalf("missing severity watch: %v", fields)
		}
		if _, ok := fields[SYSLOG_FIELD_FACILITY]; !ok {
			t.Fatalf("missing facility watch: %v", fields)
		}
	})
}
