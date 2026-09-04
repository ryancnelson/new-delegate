# DeleGate compatibility ledger

This ledger distinguishes implemented behavior from intended compatibility.
Only passing tests may move an item to `verified`.

| Original concept | Intended modern form | State |
| --- | --- | --- |
| `SERVER` | Named frontend listener and protocol | verified: basic protocol, port, and coordinated HTTP listeners |
| `MOUNT` | Ordered virtual-resource mapping with deterministic specificity | verified: path/absolute-HTTP sources, legacy syntax, frontend scoping, resolver kernel |
| `PERMIT` / `REJECT` | Fail-closed policy rules | verified: basic legacy selectors and canonical kernel |
| `RELAY` | Authorized transparent byte-stream forwarding | planned |
| `STLS` | Independent readable frontend/backend TLS settings | partial: HTTP frontend/backend and mutual TLS verified |
| `FCL` | Bounded typed request/response transformations | planned |
| `CACHE` | Explicit response cache policy | planned |
| Proxy chaining | Backend connector targeting another delegate | planned |
| Original command-line directives | Compatibility syntax adapter | partial: `SERVER`, `-P`, `MOUNT`, `PERMIT`, `REJECT` |
| Modern TOML | Strict canonical configuration file | verified: server, mount, policy, trusted proxy, and TLS models |
| Decision inspection | Side-effect-free effective routing/policy explanation | verified: JSON `explain` command |
| Forwarded client identity | Per-listener header and trusted-proxy CIDRs | verified: HTTP, right-to-left trust chain |
| Portable distribution | Checksummed self-contained executable archives | verified: five CGO-free targets |
| HTTP frontend/backend | Typed codec and connector | partial: authorized Fetch slice |
| HTTP proxy metadata boundary | Hop-by-hop and proxy-credential stripping | verified: requests and responses |
| FTP frontend/backend | Typed codec and Fetch/Store/List connector | planned |
| SOCKS5 | CONNECT and bounded relay | planned |
| SMTP, POP3, IMAP, NNTP, LDAP, DNS | Later protocol packages | research |
| Gopher, Finger, Telnet | Compatibility demand determines priority | research |

Intentional differences:

- Unknown options are fatal.
- Identical repeated scalar directives are idempotent; conflicting repetitions
  are fatal rather than silently taking the last value.
- No matching mount or permit rule denies the operation.
- Ambiguous mount precedence is a configuration error.
- Secret configuration contains references, never resolved values.
- Modern TOML is canonical; legacy syntax is an adapter onto the same model.
- Unknown TOML keys are fatal and configuration is fully validated before
  runtime resources are opened.
- Forwarded client addresses are ignored unless the immediate peer is within a
  trusted CIDR configured on that listener.
