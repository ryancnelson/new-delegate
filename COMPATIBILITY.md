# DeleGate compatibility ledger

This ledger distinguishes implemented behavior from intended compatibility.
Only passing tests may move an item to `verified`.

| Original concept | Intended modern form | State |
| --- | --- | --- |
| `SERVER` | Named frontend listener and protocol | verified: basic protocol, port, and coordinated HTTP listeners |
| `MOUNT` | Ordered virtual-resource mapping with deterministic specificity | verified: path/absolute-HTTP/CONNECT sources, legacy syntax, frontend scoping, resolver kernel |
| `PERMIT` / `REJECT` | Fail-closed policy rules | verified: basic legacy selectors and canonical kernel |
| `RELAY` | Authorized transparent byte-stream forwarding | partial: HTTP CONNECT over TCP verified |
| `STLS` | Independent readable frontend/backend TLS settings | partial: HTTP frontend/backend and mutual TLS verified |
| `FCL` | Bounded typed request/response transformations | planned |
| `CACHE` | Explicit response cache policy | planned |
| Proxy chaining | Backend connector targeting another delegate | planned |
| Original command-line directives | Compatibility syntax adapter | partial: `SERVER`, `-P`, `MOUNT`, `PERMIT`, `REJECT`; differential fixture suite supports baseline and optional reference execution |
| Modern TOML | Strict canonical configuration file | verified: server, mount, policy, trusted proxy, and TLS models |
| Decision inspection | Side-effect-free effective routing/policy explanation | verified: JSON `explain` command |
| Forwarded client identity | Per-listener header and trusted-proxy CIDRs | verified: HTTP, right-to-left trust chain |
| Portable distribution | Checksummed self-contained executable archives | verified: Darwin/arm64 and Linux/amd64 |
| HTTP frontend/backend | Typed codec and connector | partial: authorized Fetch, Store (PUT), CONNECT, and FTP Fetch/Store/List; differential harness suite is runnable against baselines |
| HTTP proxy metadata boundary | Hop-by-hop and proxy-credential stripping | verified: requests and responses |
| FTP frontend/backend | Typed codec and Fetch/Store/List connector | partial: LIST added and differential harness suite is runnable |
| SOCKS5 | CONNECT and bounded relay | planned |
| SMTP, POP3, IMAP, NNTP, LDAP, DNS | Later protocol packages | research |
| Gopher, Finger, Telnet | Compatibility demand determines priority | research |

Intentional differences:

- Unknown options are fatal.
- Identical repeated scalar directives are idempotent; conflicting repetitions
  are fatal rather than silently taking the last value.
- No matching mount or permit rule denies the operation.
- Legacy fixture parity now covers scoped legacy `MOUNT` options (`server=`,
  `protocol=`, and `priority=`) alongside existing path and absolute-URL forms.
- Compatibility fixtures now also include legacy `CONNECT` mount translation metadata
  with protocol-scoped rule coverage.
- Ambiguous mount precedence is a configuration error.
- Secret configuration contains references, never resolved values.
- Modern TOML is canonical; legacy syntax is an adapter onto the same model.
- Unknown TOML keys are fatal and configuration is fully validated before
  runtime resources are opened.
- Forwarded client addresses are ignored unless the immediate peer is within a
  trusted CIDR configured on that listener.
- Legacy compatibility fixtures now include default-port `SERVER=ftp` parsing with
  ftp mount and protocol-specific policy metadata.
- Legacy compatibility fixtures now include default-port `SERVER=gopher` parsing with
  path mount and policy metadata.
- Legacy compatibility fixtures now include default-port `SERVER=https` with HTTPS
  mount and policy parity.
- Legacy compatibility fixtures now include default-port `SERVER=socks` with TCP
  mount and protocol metadata.
- Legacy compatibility fixtures now include protocol-case normalization for
  uppercase `SERVER=` and uppercase protocol selectors in `PERMIT`/`REJECT`.
- Legacy compatibility fixtures now include protocol-case normalization for
  uppercase `SERVER=FTP` and matching `PERMIT`/`REJECT` selectors.
- Legacy compatibility fixtures now include protocol-case normalization for
  `SERVER=HTTPS` and matching protocol selectors.
- Legacy compatibility fixtures now include protocol-case normalization for
  `SERVER=GOPHER` and matching protocol selectors.
- Legacy compatibility fixtures now include protocol-case normalization for
  `SERVER=SOCKS` and matching protocol selectors.
- Legacy compatibility fixtures now include CONNECT metadata and selector parity for
  protocol normalization.
- Legacy compatibility fixtures now include protocol-case normalization for HTTPS
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for FTP
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for Gopher
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for SOCKS
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for URL-source
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include mixed-case option-key normalization
  for URL-source legacy `MOUNT` directives.
- Legacy compatibility fixtures now include mixed-case option-key
  normalization for HTTPS URL-source legacy `MOUNT` directives.
- Legacy compatibility fixtures now include protocol-case normalization for HTTPS
  `CONNECT` legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for HTTP
  `CONNECT` legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for Gopher
  `CONNECT` legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for FTP
  `CONNECT` legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for SOCKS
  legacy `MOUNT` protocol options.
- Legacy compatibility fixtures now include protocol-case normalization for CONNECT
  `MOUNT` protocol and `server=` options on uppercase server flows.
- Legacy compatibility fixtures now include protocol- and server-option
  case-normalization for CONNECT `MOUNT`s across HTTPS, Gopher, FTP, and SOCKS
  when `SERVER=` is uppercase.
- Legacy compatibility fixtures now include mixed-case `MOUNT` option keys
  (`server=`, `protocol=`, `priority=`) on legacy directives.
- Legacy compatibility fixtures now include mixed-case `MOUNT` option keys for a
  full HTTPS legacy mount with matching policies.
- Legacy compatibility fixtures now include mixed-case `MOUNT` option keys on
  legacy FTP, Gopher, and SOCKS directives.
- Legacy compatibility fixtures now include mixed-case `MOUNT` option keys in
  CONNECT legacy directives.
- Legacy compatibility fixtures now include uppercase `SERVER=HTTPS` `CONNECT`
  metadata and selector normalization.
- Legacy compatibility fixtures now include uppercase `SERVER=SOCKS` `CONNECT`
  metadata and selector normalization.
- Legacy compatibility fixtures now include uppercase `SERVER=FTP` `CONNECT`
  metadata and selector normalization.
- Legacy compatibility fixtures now include uppercase `SERVER=GOPHER` `CONNECT`
  metadata and selector normalization.
- Legacy compatibility fixtures now include mount protocol-scope normalization in
  `MOUNT` options.
