package transformer

import (
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

const (
	syslogTestHostV6  = "2001:db8::10"
	syslogTestSrcIntf = "Loopback0"
)

func setupSyslogSrcIntfMocks(serverState map[string]db.Value, srcIntf string) func() {
	restoreServers := setupSyslogDbMocks(serverState)
	serverGetEntry := dbGetEntry
	dbGetEntry = func(d *db.DB, table *db.TableSpec, key db.Key) (db.Value, error) {
		if table.Name == SYSLOG_CONFIG_TABLE_NAME && srcIntf != "" &&
			len(key.Comp) == 1 && key.Comp[0] == SYSLOG_GLOBAL_KEY {
			return db.Value{Field: map[string]string{syslogSrcIntfField: srcIntf}}, nil
		}
		return serverGetEntry(d, table, key)
	}
	return restoreServers
}

func setSyslogTestSrcIntfAddrs(addrs ...string) func() {
	return setSyslogLinuxIPViewForTest(&syslogLinuxIPView{
		intfIPs: map[string][]string{syslogTestSrcIntf: addrs},
	})
}

func newSyslogSrcIntfDevice(srcIntf string) *ocbinds.Device {
	device := &ocbinds.Device{}
	device.System = &ocbinds.OpenconfigSystem_System{
		Logging: &ocbinds.OpenconfigSystem_System_Logging{
			Config: &ocbinds.OpenconfigSystem_System_Logging_Config{
				SourceInterface: ygot.String(srcIntf),
			},
		},
	}
	return device
}

func runLoggingConfigUpdate(srcIntf string) error {
	ygRoot := ygot.GoStruct(newSyslogSrcIntfDevice(srcIntf))
	_, err := YangToDb_logging_config_xfmr(XfmrParams{
		dbs:    [db.MaxDB]*db.DB{db.ConfigDB: &db.DB{}},
		ygRoot: &ygRoot,
		oper:   UPDATE,
		uri:    SYSLOG_LOGGING_PATH + "/openconfig-system-ext:config/source-interface",
	})
	return err
}

func TestValidateSyslogSrcIntfForServer(t *testing.T) {
	view := &syslogLinuxIPView{intfIPs: map[string][]string{
		"Loopback0": {"10.1.0.1", "fe80::1"},
		"Loopback1": {"2001:db8::1"},
	}}
	tests := []struct {
		name    string
		host    string
		source  string
		srcIntf string
		wantErr bool
	}{
		{"IPv4 server with IPv4 interface", syslogTestHost1, "", "Loopback0", false},
		{"IPv6 server with only link-local IPv6", syslogTestHostV6, "", "Loopback0", true},
		{"IPv6 server with global IPv6", syslogTestHostV6, "", "Loopback1", false},
		{"IPv4 server with IPv6-only interface", syslogTestHost1, "", "Loopback1", true},
		{"per-server source-address takes precedence", syslogTestHostV6, "2001:db8::99", "Loopback0", false},
		{"hostname server is not checked", "syslog.example.com", "", "Loopback1", false},
		{"no source-interface", syslogTestHostV6, "", "", false},
		{"unknown interface", syslogTestHost1, "", "Loopback9", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSyslogSrcIntfForServer(tt.host, tt.source, tt.srcIntf, view)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSyslogSrcIntfForServer(%s, %q, %s) error = %v, wantErr %v",
					tt.host, tt.source, tt.srcIntf, err, tt.wantErr)
			}
		})
	}
}

func TestYangToDb_logging_config_xfmr_FamilyMismatch(t *testing.T) {
	servers := map[string]db.Value{syslogTestHostV6: {Field: map[string]string{nullKey: nullKey}}}

	t.Run("rejects interface without IPv6 for IPv6 server", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(servers, "")()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1")()
		if err := runLoggingConfigUpdate(syslogTestSrcIntf); err == nil {
			t.Fatalf("expected IPv6/IPv4 family mismatch error")
		}
	})

	t.Run("accepts interface with IPv6 for IPv6 server", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(servers, "")()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1", "2001:db8::1")()
		if err := runLoggingConfigUpdate(syslogTestSrcIntf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts any interface with no servers", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(map[string]db.Value{}, "")()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1")()
		if err := runLoggingConfigUpdate(syslogTestSrcIntf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestYangToDb_remote_servers_SrcIntfFamilyMismatch(t *testing.T) {
	run := func() error {
		ygRoot := ygot.GoStruct(newSyslogDevice(map[string]*ocbinds.OpenconfigSystem_System_Logging_RemoteServers_RemoteServer{
			syslogTestHostV6: newSyslogRemoteServer(syslogTestHostV6, nil, nil, nil, nil, nil),
		}))
		_, err := YangToDb_remote_servers_list_xfmr(XfmrParams{
			dbs:           [db.MaxDB]*db.DB{db.ConfigDB: &db.DB{}},
			ygRoot:        &ygRoot,
			oper:          UPDATE,
			uri:           SYSLOG_SERVER_PATH + "/config/host",
			key:           syslogTestHostV6,
			param:         syslogTestHostV6,
			subOpDataMap:  make(map[Operation]*map[db.DBNum]map[string]map[string]db.Value),
			yangAuxValMap: make(map[string]map[string]db.Value),
		})
		return err
	}

	t.Run("rejects IPv6 server when source-interface has no IPv6", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(map[string]db.Value{}, syslogTestSrcIntf)()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1")()
		if err := run(); err == nil {
			t.Fatalf("expected IPv6/IPv4 family mismatch error")
		}
	})

	t.Run("accepts IPv6 server when source-interface has IPv6", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(map[string]db.Value{}, syslogTestSrcIntf)()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1", "2001:db8::1")()
		if err := run(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts IPv6 server with no source-interface", func(t *testing.T) {
		defer setupSyslogSrcIntfMocks(map[string]db.Value{}, "")()
		defer setSyslogTestSrcIntfAddrs("10.1.0.1")()
		if err := run(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
