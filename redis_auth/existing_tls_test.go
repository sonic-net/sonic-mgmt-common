package redis_auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	redis7 "github.com/go-redis/redis/v7"
	redis9 "github.com/redis/go-redis/v9"
)

func TestUnselectedTLSConfigPreserved(t *testing.T) {
	for _, customDialer := range []bool{false, true} {
		t.Run(fmt.Sprint(customDialer), func(t *testing.T) {
			f := setup(t, true)
			// The TLS listener is outside the Unix-only policy. Neither local
			// writer credentials nor a local authentication CA are available.
			if err := os.Remove(testSecret); err != nil {
				t.Fatal(err)
			}
			certificate, err := os.ReadFile(filepath.Join(f.dir, "legacy.crt"))
			if err != nil {
				t.Fatal(err)
			}
			roots := x509.NewCertPool()
			if !roots.AppendCertsFromPEM(certificate) {
				t.Fatal("cannot load the existing caller's test certificate")
			}
			configuration := &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12, SessionTicketsDisabled: true}
			forwarded := configuration
			var existing DialFunc
			calls := 0
			if customDialer {
				dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: time.Second}, Config: configuration}
				existing = func(ctx context.Context, network, address string) (net.Conn, error) {
					calls++
					return dialer.DialContext(ctx, network, address)
				}
				forwarded = &tls.Config{ServerName: "unused-with-custom-dialer"}
			}
			address := net.JoinHostPort("127.0.0.1", fmt.Sprint(f.port))
			v7Options := &redis7.Options{Addr: address, DB: 6, TLSConfig: forwarded, DialTimeout: time.Second}
			v7Options.Dialer = WrapDialer(existing, v7Options.DialTimeout, v7Options.TLSConfig)
			v7 := redis7.NewClient(v7Options)
			defer v7.Close()
			if err := v7.Ping().Err(); err != nil {
				t.Fatal("v7 lost its existing TLS configuration", err)
			}
			if err := v7.Set("unselected-forbidden", "value", 0).Err(); err == nil || !strings.Contains(err.Error(), "NOPERM") {
				t.Fatal("unselected v7 reader received writer access", err)
			}
			ctx := context.Background()
			v9Options := &redis9.Options{Addr: address, DB: 6, TLSConfig: forwarded, DialTimeout: time.Second}
			v9Options.Dialer = WrapDialer(existing, v9Options.DialTimeout, v9Options.TLSConfig)
			v9 := redis9.NewClient(v9Options)
			defer v9.Close()
			if err := v9.Ping(ctx).Err(); err != nil {
				t.Fatal("v9 lost its existing TLS configuration", err)
			}
			if err := v9.Set(ctx, "unselected-forbidden", "value", 0).Err(); err == nil || !strings.Contains(err.Error(), "NOPERM") {
				t.Fatal("unselected v9 reader received writer access", err)
			}
			if configuration.ServerName != "" || configuration.RootCAs != roots ||
				configuration.MinVersion != tls.VersionTLS12 || !configuration.SessionTicketsDisabled {
				t.Fatal("caller TLS configuration was modified")
			}
			if customDialer && calls != 2 {
				t.Fatal("custom dialer was not retained", calls)
			}
		})
	}
}
