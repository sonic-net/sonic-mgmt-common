# Local Redis writer login for Go

Use the dialer wrapper with go-redis v7 and v9, forwarding any existing TLS
configuration:

```go
opts.Dialer = redis_auth.WrapDialer(opts.Dialer, opts.DialTimeout, opts.TLSConfig)
```

It keeps the requested transport and address. At each new connection it checks
`/run/redis-auth/local.json`, using the same object or array format as swss-common:

```json
{"username":"sonic-trusted-writer","unix_socket":"/var/run/redis/redis.sock","tcp_host":"127.0.0.1","tcp_port":0}
```

Only explicitly selected Unix sockets receive automatic writer login. Selected
processes share `/run/redis-auth/local-system/writer.secret` for one host boot.
Linux file permissions grant approved callers access to this password. An ordinary
caller denied file access connects without authentication. Redis must separately
configure its passwordless default account to deny writes.

Startup owns these fixed files and must deliver them to each client. Directories
must be root-owned and must not allow group or other users to write. The password
accepts modes `0400`, `0600`, `0440` or `0640`, with root ownership. Ordinary readers
must see the protected file and receive a permission error; a missing selected
password is an error.

A selected TCP address is rejected before connecting or opening the password.
Unselected connections retain the caller's existing transport and authentication
behavior. A custom dialer takes precedence; otherwise the wrapper retains the
supplied TLS configuration without modifying it. Callers without TLS settings
pass `nil` as the last argument.

No startup policy means existing connection behavior. Invalid or duplicate
entries, unsafe files and rejected credentials stop the connection. The original
dialer remains in use. The connection timeout covers connection and
authentication; go-redis retains its command timeout handling.

Run `go test ./redis_auth` as root in an isolated Linux container with `redis-server`
installed. The tests refuse to replace an existing `/run/redis-auth` directory.
They include real Redis, ordinary-user subprocess checks and compatibility with
existing TLS callers. Installing this package alone does not change clients;
their connection setup must use the wrapper, and startup must supply the matching
Redis account and runtime files.
