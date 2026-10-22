package redis_auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const testRoot = "/run/redis-auth"
const testSecret = testRoot + "/local-system/writer.secret"
const testPassword = "local-test-password"

type fixture struct {
	t           *testing.T
	dir, socket string
	port        int
	policy      map[string]interface{}
}

func write(t *testing.T, name string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(name, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(name, mode); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) setPolicy(value interface{}) {
	f.t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		f.t.Fatal(err)
	}
	write(f.t, testRoot+"/local.json", data, 0644)
}

func makeCertificate(t *testing.T, cert, key string, ip string) {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Redis test"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true,
		IPAddresses: []net.IP{net.ParseIP(ip)}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		t.Fatal(err)
	}
	write(t, cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)
	write(t, key, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)}), 0600)
}

func startRedis(t *testing.T, directory, name, username string, port int) string {
	t.Helper()
	socket := filepath.Join(directory, name+".sock")
	acl := filepath.Join(directory, name+".acl")
	write(t, acl, []byte("user default on nopass ~* +@read +select +ping\nuser "+username+" on >"+testPassword+" ~* &* +@all\n"), 0600)
	args := []string{"--port", "0", "--unixsocket", socket, "--unixsocketperm", "777", "--save", "", "--appendonly", "no", "--aclfile", acl}
	if port > 0 {
		args = append(args, "--bind", "127.0.0.1", "--tls-port", fmt.Sprint(port), "--tls-cert-file", filepath.Join(directory, "legacy.crt"), "--tls-key-file", filepath.Join(directory, "server.key"), "--tls-ca-cert-file", filepath.Join(directory, "legacy.crt"), "--tls-auth-clients", "no")
	}
	server := exec.Command("redis-server", args...)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Process.Kill(); server.Wait() })
	for i := 0; i < 200; i++ {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			conn.Close()
			return socket
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Redis did not start")
	return ""
}

func setup(t *testing.T, tls bool) *fixture {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("run in an isolated root container")
	}
	if err := os.Mkdir(testRoot, 0755); err != nil {
		t.Fatal("refusing to replace runtime files:", err)
	}
	t.Cleanup(func() { os.RemoveAll(testRoot) })
	if err := os.Mkdir(testRoot+"/local-system", 0755); err != nil {
		t.Fatal(err)
	}
	write(t, testSecret, []byte(testPassword), 0400)
	dir, err := os.MkdirTemp("/tmp", "redis-go-auth-")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	f := &fixture{t: t, dir: dir}
	if tls {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		f.port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		makeCertificate(t, filepath.Join(dir, "legacy.crt"), filepath.Join(dir, "server.key"), "127.0.0.1")
	}
	f.socket = startRedis(t, dir, "primary", "primary-writer", f.port)
	f.policy = map[string]interface{}{"username": "primary-writer", "unix_socket": f.socket, "tcp_host": "127.0.0.1", "tcp_port": 0}
	f.setPolicy(f.policy)
	return f
}

func command(conn net.Conn, args ...string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, arg := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(arg), arg)
	}
	if _, err := fmt.Fprint(conn, b.String()); err != nil {
		return "", err
	}
	return bufio.NewReader(conn).ReadString('\n')
}

func assertWriter(t *testing.T, network, address string) {
	t.Helper()
	conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), network, address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reply, err := command(conn, "SET", "go-auth-test", "works")
	if err != nil || reply != "+OK\r\n" {
		t.Fatalf("write failed: %q %v", reply, err)
	}
}

func TestUnixWriter(t *testing.T) {
	f := setup(t, false)
	assertWriter(t, "unix", f.socket)
	assertWriter(t, "unix", f.socket)
}
func TestBadAndMissingCredentials(t *testing.T) {
	f := setup(t, false)
	for _, missing := range []bool{false, true} {
		if missing {
			os.Remove(testSecret)
		} else {
			write(t, testSecret, []byte("wrong-password"), 0400)
		}
		conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket)
		if err == nil {
			conn.Close()
			t.Fatal("credential failure allowed a connection")
		}
	}
}

func TestPolicyValidation(t *testing.T) {
	f := setup(t, false)
	duplicate := map[string]interface{}{}
	for k, v := range f.policy {
		duplicate[k] = v
	}
	duplicate["unix_socket"] = f.socket + ".other"
	duplicate["tcp_port"] = 1234
	primary := map[string]interface{}{}
	for k, v := range f.policy {
		primary[k] = v
	}
	primary["tcp_port"] = 1234
	invalid := map[string]interface{}{}
	for k, v := range f.policy {
		invalid[k] = v
	}
	invalid["unix_socket"] = "relative.sock"
	for name, value := range map[string]interface{}{
		"empty-array": []interface{}{}, "null": nil, "missing-fields": map[string]interface{}{},
		"duplicate-unix": []interface{}{f.policy, f.policy}, "duplicate-tcp": []interface{}{primary, duplicate},
		"invalid-unselected": []interface{}{f.policy, invalid}, "null-port": map[string]interface{}{"username": "writer", "unix_socket": f.socket, "tcp_host": "127.0.0.1", "tcp_port": nil},
	} {
		t.Run(name, func(t *testing.T) {
			f.setPolicy(value)
			conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket)
			if err == nil {
				conn.Close()
				t.Fatal("invalid policy accepted")
			}
		})
	}
}

func TestUnselectedEndpointNeverReadsSecret(t *testing.T) {
	f := setup(t, false)
	f.policy["unix_socket"] = f.socket + ".other"
	f.setPolicy(f.policy)
	os.Remove(testSecret)
	conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reply, err := command(conn, "PING")
	if err != nil || reply != "+PONG\r\n" {
		t.Fatal(reply, err)
	}
	reply, err = command(conn, "SET", "forbidden", "value")
	if err != nil || !strings.Contains(reply, "NOPERM") {
		t.Fatal(reply, err)
	}
}

func TestTwoLocalProcesses(t *testing.T) {
	f := setup(t, false)
	second := startRedis(t, f.dir, "secondary", "secondary-writer", 0)
	f.setPolicy([]interface{}{f.policy, map[string]interface{}{"username": "secondary-writer", "unix_socket": second, "tcp_host": "127.0.0.1", "tcp_port": 0}})
	assertWriter(t, "unix", f.socket)
	assertWriter(t, "unix", second)
}

func TestUnsafeFiles(t *testing.T) {
	f := setup(t, false)
	for _, name := range []string{testRoot, testRoot + "/local.json", testRoot + "/local-system", testSecret} {
		st, _ := os.Stat(name)
		os.Chmod(name, 0777)
		conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket)
		os.Chmod(name, st.Mode().Perm())
		if err == nil {
			conn.Close()
			t.Fatalf("accepted writable file %s", name)
		}
	}
	write(t, testSecret, []byte(testPassword), 0644)
	if conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket); err == nil {
		conn.Close()
		t.Fatal("world-readable secret accepted")
	}
}

func TestSecretSymlinkRejected(t *testing.T) {
	f := setup(t, false)
	os.Rename(testSecret, testRoot+"/decoy")
	os.Symlink(testRoot+"/decoy", testSecret)
	if conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), "unix", f.socket); err == nil {
		conn.Close()
		t.Fatal("symlink secret accepted")
	}
}

func TestExistingDialerPreserved(t *testing.T) {
	f := setup(t, false)
	os.Remove(testRoot + "/local.json")
	left, right := net.Pipe()
	defer right.Close()
	called := false
	existing := func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "unix" || address != f.socket {
			t.Fatal("address changed")
		}
		called = true
		return left, nil
	}
	conn, err := WrapDialer(existing, time.Second, nil)(context.Background(), "unix", f.socket)
	if err != nil || !called || conn != left {
		t.Fatal("existing dialer was replaced", err)
	}
	conn.Close()
}

func TestSelectedTCPRejectedBeforeDialAndCredentialAccess(t *testing.T) {
	f := setup(t, false)
	f.policy["tcp_port"] = 6379
	f.setPolicy(f.policy)
	if err := os.Remove(testSecret); err != nil {
		t.Fatal(err)
	}
	called := false
	existing := func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, fmt.Errorf("unexpected TCP connection")
	}
	for _, network := range []string{"tcp", "tcp4", "tcp6"} {
		_, err := WrapDialer(existing, time.Second, nil)(context.Background(), network, "127.0.0.1:6379")
		if called || err == nil || !strings.Contains(err.Error(), "requires a Unix socket") {
			t.Fatalf("selected TCP reached the dialer or credential file: %v", err)
		}
	}
}

func TestAuthenticationDeadline(t *testing.T) {
	f := setup(t, false)
	listener, err := net.Listen("unix", filepath.Join(f.dir, "stalled.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	f.policy["unix_socket"] = listener.Addr().String()
	f.setPolicy(f.policy)
	done := make(chan struct{})
	defer close(done)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			<-done
		}
	}()
	start := time.Now()
	_, err = WrapDialer(nil, 100*time.Millisecond, nil)(context.Background(), "unix", listener.Addr().String())
	if err == nil || time.Since(start) > time.Second {
		t.Fatal("authentication timeout not preserved", err)
	}
}

func TestOrdinaryReaders(t *testing.T) {
	f := setup(t, false)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	publicBinary := filepath.Join(f.dir, "reader.test")
	write(t, publicBinary, data, 0755)
	child := exec.Command(publicBinary, "-test.run=^TestOrdinaryChild$")
	child.Env = append(os.Environ(), "REDIS_AUTH_TEST_NETWORK=unix", "REDIS_AUTH_TEST_ADDRESS="+f.socket)
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("reader failed: %s %v", output, err)
	}
}

func TestApprovedGroupWriter(t *testing.T) {
	f := setup(t, false)
	if err := os.Chown(testSecret, 0, 65534); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(testSecret, 0440); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	publicBinary := filepath.Join(f.dir, "writer.test")
	write(t, publicBinary, data, 0755)
	child := exec.Command(publicBinary, "-test.run=^TestOrdinaryChild$")
	child.Env = append(os.Environ(), "REDIS_AUTH_TEST_NETWORK=unix", "REDIS_AUTH_TEST_ADDRESS="+f.socket, "REDIS_AUTH_TEST_WRITER=1")
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("approved writer failed: %s %v", output, err)
	}
}

func TestOrdinaryChild(t *testing.T) {
	address := os.Getenv("REDIS_AUTH_TEST_ADDRESS")
	if address == "" {
		t.Skip("subprocess test")
	}
	if os.Geteuid() == 0 {
		t.Fatal("reader did not drop root")
	}
	conn, err := WrapDialer(nil, time.Second, nil)(context.Background(), os.Getenv("REDIS_AUTH_TEST_NETWORK"), address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if reply, err := command(conn, "PING"); err != nil || reply != "+PONG\r\n" {
		t.Fatal(reply, err)
	}
	reply, err := command(conn, "SET", "forbidden", "value")
	if os.Getenv("REDIS_AUTH_TEST_WRITER") == "1" {
		if err != nil || reply != "+OK\r\n" {
			t.Fatal(reply, err)
		}
	} else if err != nil || !strings.Contains(reply, "NOPERM") {
		t.Fatal(reply, err)
	}
}
