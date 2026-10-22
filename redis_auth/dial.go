// Package redis_auth signs approved local clients in with the boot writer account.
package redis_auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// DialFunc is compatible with the Dialer option in go-redis v7 and v9.
type DialFunc func(context.Context, string, string) (net.Conn, error)

type localPolicy struct {
	Username   string `json:"username"`
	UnixSocket string `json:"unix_socket"`
	TCPHost    string `json:"tcp_host"`
	TCPPort    *int   `json:"tcp_port"`
}

func checkedFile(fd int, directory, secret bool) (*os.File, error) {
	file := os.NewFile(uintptr(fd), "Redis authentication file")
	st, err := file.Stat()
	if err == nil {
		owner, ok := st.Sys().(*syscall.Stat_t)
		mode := st.Mode().Perm()
		if !ok || owner.Uid != 0 || st.IsDir() != directory ||
			(!directory && !st.Mode().IsRegular()) || mode&0022 != 0 ||
			(secret && mode != 0400 && mode != 0600 && mode != 0440 && mode != 0640) {
			err = errors.New("Redis authentication file has unsafe permissions")
		}
	}
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func openAt(directory int, name string, folder, secret bool) (*os.File, error) {
	flags := syscall.O_RDONLY | syscall.O_CLOEXEC | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	if folder {
		flags |= syscall.O_DIRECTORY
	}
	fd, err := syscall.Openat(directory, name, flags, 0)
	if err != nil {
		return nil, err
	}
	return checkedFile(fd, folder, secret)
}

func readFile(file *os.File, maximum int64) ([]byte, error) {
	value, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err == nil && (len(value) == 0 || int64(len(value)) > maximum) {
		err = errors.New("Redis authentication file has invalid size")
	}
	return value, err
}

func selectPolicy(directory int, network, address string) (*localPolicy, error) {
	file, err := openAt(directory, "local.json", false, false)
	if errors.Is(err, syscall.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("Cannot open Redis authentication policy")
	}
	defer file.Close()
	value, err := readFile(file, 4096)
	if err != nil {
		return nil, err
	}
	value = bytes.TrimSpace(value)
	if len(value) > 0 && value[0] == '{' {
		value = append(append([]byte{'['}, value...), ']')
	}
	var entries []json.RawMessage
	if err = json.Unmarshal(value, &entries); err != nil || len(entries) == 0 {
		return nil, errors.New("Redis authentication policy is invalid")
	}
	var selected *localPolicy
	sockets, addresses := map[string]bool{}, map[string]bool{}
	for _, entry := range entries {
		var fields map[string]json.RawMessage
		var policy localPolicy
		decoder := json.NewDecoder(bytes.NewReader(entry))
		decoder.DisallowUnknownFields()
		if json.Unmarshal(entry, &fields) != nil || len(fields) != 4 || decoder.Decode(&policy) != nil ||
			policy.Username == "" || len(policy.Username) > 128 || strings.ContainsRune(policy.Username, 0) ||
			!strings.HasPrefix(policy.UnixSocket, "/") || strings.ContainsRune(policy.UnixSocket, 0) ||
			(policy.TCPHost != "127.0.0.1" && policy.TCPHost != "::1") ||
			policy.TCPPort == nil || *policy.TCPPort < 0 || *policy.TCPPort > 65535 {
			return nil, errors.New("Redis authentication policy is invalid")
		}
		tcp := net.JoinHostPort(policy.TCPHost, strconv.Itoa(*policy.TCPPort))
		if sockets[policy.UnixSocket] || (*policy.TCPPort > 0 && addresses[tcp]) {
			return nil, errors.New("Redis authentication policy has duplicate addresses")
		}
		sockets[policy.UnixSocket] = true
		if *policy.TCPPort > 0 {
			addresses[tcp] = true
		}
		if (network == "unix" && address == policy.UnixSocket) ||
			((network == "tcp" || network == "tcp4" || network == "tcp6") && *policy.TCPPort > 0 && address == tcp) {
			copy := policy
			selected = &copy
		}
	}
	return selected, nil
}

// WrapDialer preserves the requested address and authenticates selected local
// Unix sockets before returning their connection. The existing dialer and TLS
// configuration are retained; selected TCP authentication is unsupported.
// A zero timeout uses go-redis's five-second default; a negative value disables it.
func WrapDialer(existing DialFunc, timeout time.Duration, configuredTLS *tls.Config) DialFunc {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if timeout < 0 {
		timeout = 0
	}
	if existing == nil {
		dialer := &net.Dialer{Timeout: timeout, KeepAlive: 5 * time.Minute}
		existing = dialer.DialContext
		if configuredTLS != nil {
			existing = (&tls.Dialer{NetDialer: dialer, Config: configuredTLS}).DialContext
		}
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		fd, err := syscall.Open("/run/redis-auth", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
		if errors.Is(err, syscall.ENOENT) {
			return existing(ctx, network, address)
		}
		if err != nil {
			return nil, errors.New("Cannot open Redis authentication directory")
		}
		directory, err := checkedFile(fd, true, false)
		if err != nil {
			return nil, err
		}
		defer directory.Close()
		policy, err := selectPolicy(fd, network, address)
		if err != nil {
			return nil, err
		}
		if policy == nil {
			return existing(ctx, network, address)
		}
		if network != "unix" {
			return nil, errors.New("Automatic Redis writer authentication requires a Unix socket")
		}
		deadline, hasDeadline := ctx.Deadline()
		if timeout > 0 {
			end := time.Now().Add(timeout)
			if !hasDeadline || end.Before(deadline) {
				deadline, hasDeadline = end, true
			}
		}
		if hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
		conn, err := existing(ctx, network, address)
		if err != nil {
			return nil, err
		}
		accepted := false
		defer func() {
			if !accepted {
				conn.Close()
			}
		}()
		if hasDeadline {
			if err = conn.SetDeadline(deadline); err != nil {
				return nil, err
			}
		}
		secrets, err := openAt(fd, "local-system", true, false)
		if err != nil {
			return nil, errors.New("Cannot open Redis credential directory")
		}
		secretFile, err := openAt(int(secrets.Fd()), "writer.secret", false, true)
		secrets.Close()
		if errors.Is(err, syscall.EACCES) && os.Geteuid() != 0 {
			if err = conn.SetDeadline(time.Time{}); err != nil {
				return nil, err
			}
			accepted = true
			return conn, nil
		}
		if err != nil {
			return nil, errors.New("Cannot open Redis writer credential")
		}
		secret, err := readFile(secretFile, 128)
		secretFile.Close()
		if err != nil {
			return nil, err
		}
		payload := []byte(fmt.Sprintf("*3\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(policy.Username), policy.Username, len(secret), secret))
		defer func() {
			for i := range secret {
				secret[i] = 0
			}
			for i := range payload {
				payload[i] = 0
			}
		}()
		if _, err = io.Copy(conn, bytes.NewReader(payload)); err != nil {
			return nil, errors.New("Redis authentication transport failure")
		}
		var reply [5]byte
		if _, err = io.ReadFull(conn, reply[:]); err != nil {
			return nil, errors.New("Redis authentication transport failure")
		}
		if string(reply[:]) != "+OK\r\n" {
			return nil, errors.New("Redis writer authentication failed")
		}
		if err = conn.SetDeadline(time.Time{}); err != nil {
			return nil, err
		}
		accepted = true
		return conn, nil
	}
}
