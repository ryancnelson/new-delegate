# new-delegate

[![Woodpecker CI](http://biggie.lynx-eagle.ts.net:8110/api/badges/5/status.svg)](http://biggie.lynx-eagle.ts.net:8110/repos/5)

`new-delegate` is a portable, DeleGate-compatible application gateway written
in Go. It preserves the original program's strongest concepts: a daemon can
present one protocol on its client side, map virtual resources with `MOUNT`,
apply fail-closed access policy, and reach a different protocol on its server
side.

The first milestone is a tested compatibility kernel for `SERVER`, `MOUNT`,
`PERMIT`, and `REJECT`, followed by an HTTP-to-HTTP vertical slice. Protocol
translation is built around typed semantic operations rather than a bag of
protocol-specific optional fields.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

Validate original-style directives without starting a listener:

```sh
go run ./cmd/delegate check \
  SERVER=http -P8080 \
  'MOUNT=/* http://127.0.0.1:9000/*' \
  'PERMIT=http:127.0.0.1:*'
```

Remove `check` (or replace it with `serve`) to start the current HTTP frontend.
With no matching `PERMIT`, requests are denied by default.

The same canonical model can be written as strict TOML. Unknown keys and
invalid values are fatal before a listener starts:

```sh
go run ./cmd/delegate check --config examples/delegate.toml
go run ./cmd/delegate serve --config examples/delegate.toml
```

`--config=examples/delegate.toml` is also accepted. A TOML file cannot be
combined with legacy directives on the same invocation.

Mounts may be scoped to a named frontend and protocol with `server` and
`protocol`. Unscoped mounts remain shared fallbacks; an equally ranked scoped
mount wins over a shared one.

A canonical file may define multiple HTTP servers. The runtime binds all
configured listeners before serving, then coordinates cancellation and bounded
graceful shutdown across the group.

On Darwin and other Unix-family systems, send `SIGHUP` to atomically reload a
canonical TOML file. Invalid changes are reported and the previous snapshot
stays active. Listener names, protocols, and addresses require a restart;
routing and policy changes reload live. Signal reload is disabled on Windows.

The direct socket peer is the policy source by default. A listener may opt in
to a forwarded client-address header only alongside explicit trusted-proxy
CIDRs:

```toml
client_ip_header = "X-Forwarded-For"
trusted_proxies = ["127.0.0.1/32", "10.0.0.0/8"]
```

Headers from peers outside those CIDRs are ignored. For a trusted peer, the
chain is evaluated from right to left until the first untrusted address;
malformed chains fail with HTTP 400 before routing reaches a backend. These
settings are request policy and may be changed by an atomic `SIGHUP` reload.

Frontend TLS termination and backend TLS verification/client identity have
separate strict configuration records. The shape can be inspected today:

```sh
go run ./cmd/delegate check --config examples/tls-policy.toml
```

Frontend identity requires paired certificate/private-key file references.
Every configured identity is loaded before any listener binds; plaintext and
TLS listeners can coexist under the same bounded shutdown lifecycle. An
unspecified frontend minimum defaults to TLS 1.2, while `1.3` is enforced when
requested. Backend policy may name a CA file, verification name, and paired
client certificate/private-key references. There is intentionally no insecure
verification bypass. Custom backend TLS remains fail-closed at startup and
reload until the per-mount transport adapter lands.

Explain the effective routing and policy result without opening a listener or
contacting the backend:

```sh
go run ./cmd/delegate explain \
  --config examples/delegate.toml \
  --path /docs/index.html \
  --source 127.0.0.1 \
  --method GET
```

The JSON result distinguishes `permit`, `reject`, `no_mount`, `unsafe_path`,
and `ambiguous_mount`, and includes the winning policy rule index when policy
evaluation occurs. Original-style configuration directives can be supplied in
place of `--config`.

Development is organized as small iterate-bot sessions. Each session takes the
highest-priority ready item from `BACKLOG.md`, begins with a failing test, makes
the smallest implementation that passes, updates verified state, and commits.

The canonical private origin is Gitea on Biggie. A private GitHub mirror feeds
Biggie's existing Woodpecker forge integration; it is CI transport, not a
second source of truth.
