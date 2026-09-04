# DeleGate compatibility ledger

This ledger distinguishes implemented behavior from intended compatibility.
Only passing tests may move an item to `verified`.

| Original concept | Intended modern form | State |
| --- | --- | --- |
| `SERVER` | Named frontend listener and protocol | planned |
| `MOUNT` | Ordered virtual-resource mapping with deterministic specificity | planned |
| `PERMIT` / `REJECT` | Fail-closed policy rules | planned |
| `RELAY` | Authorized transparent byte-stream forwarding | planned |
| `STLS` | Independent readable frontend/backend TLS settings | planned |
| `FCL` | Bounded typed request/response transformations | planned |
| `CACHE` | Explicit response cache policy | planned |
| Proxy chaining | Backend connector targeting another delegate | planned |
| Original command-line directives | Compatibility syntax adapter | planned |
| HTTP frontend/backend | Typed codec and connector | planned |
| FTP frontend/backend | Typed codec and Fetch/Store/List connector | planned |
| SOCKS5 | CONNECT and bounded relay | planned |
| SMTP, POP3, IMAP, NNTP, LDAP, DNS | Later protocol packages | research |
| Gopher, Finger, Telnet | Compatibility demand determines priority | research |

Intentional differences:

- Unknown options are fatal.
- No matching mount or permit rule denies the operation.
- Ambiguous mount precedence is a configuration error.
- Secret configuration contains references, never resolved values.
- Modern TOML is canonical; legacy syntax is an adapter onto the same model.

