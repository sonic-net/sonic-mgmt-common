package redis_auth

import (
	"context"
	"strings"
	"testing"
	"time"

	redis7 "github.com/go-redis/redis/v7"
	redis9 "github.com/redis/go-redis/v9"
)

func TestGoRedisV7AndV9Reconnect(t *testing.T) {
	f := setup(t, false)
	address := f.socket
	ctx := context.Background()
	v7 := redis7.NewClient(&redis7.Options{Network: "unix", Addr: address, DB: 6, Dialer: WrapDialer(nil, time.Second, nil)})
	defer v7.Close()
	v9 := redis9.NewClient(&redis9.Options{Network: "unix", Addr: address, DB: 6, Dialer: WrapDialer(nil, time.Second, nil)})
	defer v9.Close()
	if err := v7.Set("version-seven", "value", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := v9.Set(ctx, "version-nine", "value", 0).Err(); err != nil {
		t.Fatal(err)
	}
	pipe := v9.Pipeline()
	pipe.HSet(ctx, "pipeline", "field", "value")
	if _, err := pipe.Exec(ctx); err != nil {
		t.Fatal(err)
	}
	sub := v7.PSubscribe("go-auth-events")
	defer sub.Close()
	if _, err := sub.ReceiveTimeout(time.Second); err != nil {
		t.Fatal(err)
	}
	if err := v9.Publish(ctx, "go-auth-events", "message").Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.ReceiveTimeout(time.Second); err != nil {
		t.Fatal(err)
	}
	control, err := WrapDialer(nil, time.Second, nil)(ctx, "unix", address)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	if reply, err := command(control, "CLIENT", "KILL", "TYPE", "normal", "SKIPME", "yes"); err != nil || !strings.HasPrefix(reply, ":") {
		t.Fatal(reply, err)
	}
	// v7 can return the first dropped request when retries are disabled.
	// A following request must open a new connection and authenticate.
	var value string
	for attempt := 0; attempt < 3; attempt++ {
		value, err = v7.Get("version-nine").Result()
		if err == nil {
			break
		}
	}
	if err != nil || value != "value" {
		t.Fatal(value, err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		value, err = v9.Get(ctx, "version-seven").Result()
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
	v7.Set("forbidden", "value", 0) // Discard the already-closed socket.
	if err := v7.Set("forbidden", "value", 0).Err(); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatal("v7 did not reject new authentication", err)
	}
	v9.Set(ctx, "forbidden", "value", 0)
	if err := v9.Set(ctx, "forbidden", "value", 0).Err(); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatal("v9 did not reject new authentication", err)
	}
}
