package nativeconn_test

import (
	"bufio"
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
	"testing"
	"time"
)

const testRoot = "/run/redis-auth"

var testSecret = testRoot + "/local-system/writer.secret"

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
			reply, pingErr := command(conn, "PING")
			conn.Close()
			if pingErr == nil && strings.TrimSpace(reply) == "+PONG" {
				return socket
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Redis did not start")
	return ""
}

func setup(t *testing.T, tls bool) *fixture {
	t.Helper()
	if os.Geteuid() != 0 || os.Getenv("SWSS_AUTH_TEST_CONTAINER") != "1" {
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
