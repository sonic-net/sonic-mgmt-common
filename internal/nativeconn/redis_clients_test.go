package nativeconn_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Azure/sonic-mgmt-common/redisconn"
	redis9 "github.com/redis/go-redis/v9"
)

func TestGoRedisReconnect(t *testing.T) {
	f := setup(t, false)
	address := f.socket
	ctx := context.Background()
	v9 := redis9.NewClient(&redis9.Options{Network: "unix", Addr: address, DB: 6, Dialer: func(ctx context.Context, _ string, address string) (net.Conn, error) {
		return redisconn.DialUnixContext(ctx, address, time.Second, true)
	}})
	defer v9.Close()
	if err := v9.Set(ctx, "reconnect-key", "value", 0).Err(); err != nil {
		t.Fatal(err)
	}
	pipe := v9.Pipeline()
	pipe.HSet(ctx, "pipeline", "field", "value")
	if _, err := pipe.Exec(ctx); err != nil {
		t.Fatal(err)
	}
	sub := v9.PSubscribe(ctx, "go-auth-events")
	defer sub.Close()
	if _, err := sub.ReceiveTimeout(ctx, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := v9.Publish(ctx, "go-auth-events", "message").Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.ReceiveTimeout(ctx, time.Second); err != nil {
		t.Fatal(err)
	}
	control, err := redisconn.DialUnixContext(ctx, address, time.Second, true)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	if reply, err := command(control, "CLIENT", "KILL", "TYPE", "normal", "SKIPME", "yes"); err != nil || !strings.HasPrefix(reply, ":") {
		t.Fatal(reply, err)
	}
	var value string
	for attempt := 0; attempt < 3; attempt++ {
		value, err = v9.Get(ctx, "reconnect-key").Result()
		if err == nil {
			break
		}
	}
	if err != nil || value != "value" {
		t.Fatal(value, err)
	}
	write(t, testSecret, []byte("wrong-password"), 0400)
	if reply, err := command(control, "CLIENT", "KILL", "TYPE", "normal", "SKIPME", "yes"); err != nil || !strings.HasPrefix(reply, ":") {
		t.Fatal(reply, err)
	}
	v9.Set(ctx, "forbidden", "value", 0)
	if err := v9.Set(ctx, "forbidden", "value", 0).Err(); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatal("v9 did not reject new authentication", err)
	}
}
