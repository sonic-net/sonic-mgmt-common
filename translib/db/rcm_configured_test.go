package db

import (
	"context"
	"encoding/json"
	"net"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRedisClientManagerInitializesOnlyConfiguredDatabases(t *testing.T) {
	start := func(name string) string {
		path := filepath.Join(t.TempDir(), name+".sock")
		server := exec.Command("redis-server", "--port", "0", "--unixsocket", path,
			"--save", "", "--appendonly", "no")
		if err := server.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = server.Process.Kill(); _ = server.Wait() })
		for i := 0; i < 200; i++ {
			connection, err := net.DialTimeout("unix", path, time.Second)
			if err == nil {
				_ = connection.Close()
				return path
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("test Redis did not start")
		return ""
	}
	primary, bmp := start("primary"), start("bmp")
	previousConfig, previousManager := dbConfigMap, rcm
	previousOptions, previousReconfigure := dbRedisOptsConfig, reconfigureRedisOptsConfig
	defer func() {
		for _, client := range rcm.clients {
			if client != nil {
				_ = client.Close()
			}
		}
		dbConfigMap, rcm = previousConfig, previousManager
		dbRedisOptsConfig, reconfigureRedisOptsConfig = previousOptions, previousReconfigure
	}()
	configuration, err := json.Marshal(map[string]interface{}{
		"INSTANCES": map[string]interface{}{
			"redis":     map[string]interface{}{"hostname": "127.0.0.1", "port": 0, "unix_socket_path": primary},
			"redis_bmp": map[string]interface{}{"hostname": "127.0.0.1", "port": 0, "unix_socket_path": bmp},
		},
		"DATABASES": map[string]interface{}{
			"CONFIG_DB": map[string]interface{}{"id": 4, "separator": "|", "instance": "redis"},
			"STATE_DB":  map[string]interface{}{"id": 6, "separator": "|", "instance": "redis_bmp"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dbConfigMap = make(map[string]interface{})
	if err := json.Unmarshal(configuration, &dbConfigMap); err != nil {
		t.Fatal(err)
	}
	rcm, dbRedisOptsConfig, reconfigureRedisOptsConfig = nil, &_DBRedisOptsConfig{}, false
	initializeRedisClientManager()
	for index, client := range rcm.clients {
		if DBNum(index) != ConfigDB && DBNum(index) != StateDB {
			if client != nil {
				t.Errorf("unconfigured database %d received an eager client", index)
			}
			continue
		}
		if client == nil {
			t.Fatalf("configured database %d has no client", index)
		}
		wanted := primary
		if DBNum(index) == StateDB {
			wanted = bmp
		}
		if options := client.Options(); options.Network != "unix" || options.Addr != wanted {
			t.Errorf("database %d selected %s %s", index, options.Network, options.Addr)
		}
		if err := client.Ping(context.Background()).Err(); err != nil {
			t.Fatal(err)
		}
	}
	// An explicit request for an unconfigured database retains its old options.
	if options := adjustRedisOpts(&Options{DBNo: ErrorDB}); options.Addr != DefaultRedisLocalTCPEP || options.DB != int(ErrorDB) {
		t.Fatal("explicit unconfigured-database behavior changed")
	}
}
