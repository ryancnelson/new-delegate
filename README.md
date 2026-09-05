# new-delegate

[![Woodpecker CI](http://biggie.lynx-eagle.ts.net:8110/api/badges/5/status.svg)](http://biggie.lynx-eagle.ts.net:8110/repos/5)

`new-delegate` is a portable application gateway written in Go. It preserves
the original DeleGate program's strongest concepts: a daemon can present one
protocol on its client side, map virtual resources, apply fail-closed access
policy, and reach a different protocol on its server side. General command-line
compatibility with the original is not a goal.

The primary one-route CLI is moving toward a strict, socat-familiar pair of
typed addresses. `SERVER`, `MOUNT`, `PERMIT`, and `REJECT` remain a curated
migration dialect, while canonical TOML describes multi-route gateways and
protocol translation. Translation uses typed semantic operations rather than a
bag of protocol-specific optional fields.

The new two-address grammar is parsed and tested but not yet wired into the
executable. Its first intended forms are:

```text
delegate TCP-LISTEN:127.0.0.1:18080 TCP-CONNECT:127.0.0.1:8080
delegate TAILCAT-LISTEN:8080 TCP-CONNECT:127.0.0.1:8080
delegate TCP-LISTEN:127.0.0.1:18080 TAILCAT-CONNECT:@stdin
```

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

Mount sources may also be absolute HTTP URLs, matching the original tool's
distinctive form:

```sh
go run ./cmd/delegate check \
  SERVER=http -P8080 \
  'MOUNT="http://example.com:8080/docs/* http://docs.internal/*"' \
  'PERMIT="http:docs.internal:*"'
```

Strict TOML uses `source = "http://example.com:8080/docs/*"` instead of
`path`. Scheme and authority match without DNS, host case is ignored, and an
explicit port is distinct from an omitted port. URL sources reject userinfo,
queries, fragments, escaping, traversal, and non-canonical paths.

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
verification bypass. Each distinct backend policy is loaded before listeners
bind and selected only from the resolver's winning mount; the semantic Fetch
operation contains no TLS fields. System roots remain the default, custom roots
are appended when configured, and mutual TLS identities are supported. Changing
the set of backend TLS policies requires a restart; routing among already
loaded policies may still reload atomically.

Explain the effective routing and policy result without opening a listener or
contacting the backend:

```sh
go run ./cmd/delegate explain \
  --config examples/delegate.toml \
  --path /docs/index.html \
  --source 127.0.0.1 \
  --method GET
```

For an absolute URL source, replace `--path` with
`--url http://example.com:8080/docs/index.html`.

At the HTTP boundary, proxy credentials and hop-by-hop metadata are never
forwarded. Requests and responses drop `Connection`, every header it names,
`Proxy-Authorization`, `Proxy-Authenticate`, `Proxy-Connection`, `Keep-Alive`,
`TE`, `Trailer`, `Transfer-Encoding`, and `Upgrade`; end-to-end metadata is
preserved.

HTTP `CONNECT` is an explicit, fail-closed byte-stream route. It does not dial
the client-supplied authority directly: a `connect://host:port/` source must
select a `tcp://host:port` target, and policy must permit method `CONNECT`
before the TCP connector runs. For example:

```toml
[[mounts]]
source = "connect://origin.example:443/"
target = "tcp://127.0.0.1:9443"

[[policies]]
effect = "permit"
protocol = "http"
destination = "127.0.0.1"
source = "127.0.0.1"
method = "CONNECT"
```

The relay does not intercept TLS. It validates explicit ports without DNS,
uses bounded handshakes and rolling idle deadlines, supports TCP half-close,
and always closes both sides when the tunnel ends. The same route can be
inspected without dialing:

```sh
go run ./cmd/delegate explain --config examples/connect.toml \
  --url connect://origin.example:443/ --source 127.0.0.1 \
  --method CONNECT
```

Authorized HTTP `PUT` requests use a distinct protocol-neutral Store operation
while sharing the same routing, policy, header-scrubbing, and per-mount backend
TLS selection as Fetch. Store bodies are streamed with a 32 MiB cap; a declared
oversized body is rejected before the backend connector runs.

The JSON result distinguishes `permit`, `reject`, `no_mount`, `unsafe_path`,
and `ambiguous_mount`, and includes the winning policy rule index when policy
evaluation occurs. Original-style configuration directives can be supplied in
place of `--config`.

Development is organized as small iterate-bot sessions. Each session takes the
highest-priority ready item from `BACKLOG.md`, begins with a failing test, makes
the smallest implementation that passes, updates verified state, and commits.

Create reproducible single-binary release archives for the active targets:

```sh
./scripts/release.sh v0.1.0 dist
shasum -a 256 -c dist/SHA256SUMS
```

The current matrix is Darwin/arm64 and Linux/amd64, both with `CGO_ENABLED=0`.
Each deterministic zip contains only `delegate` and `README.md`; `delegate
version` reports the link-time stamp. The native archive is smoke-tested with
`delegate check`.

The canonical private origin is Gitea on Biggie. A private GitHub mirror feeds
Biggie's existing Woodpecker forge integration; it is CI transport, not a
second source of truth.
