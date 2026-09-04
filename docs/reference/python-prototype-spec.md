# DeleGate Modern — Implementation Specification

**Version:** 0.1.0-spec
**Status:** Draft
**Target:** Language-agnostic; primary implementation target is MoonBit
**Source Material:** Original DeleGate proxy (Yutaka Sato, ETL/METI Japan, ~1994–2007), Python prototype in this repo, AGENTIC_VISION.md, knowledge-from-scars.md, next-steps-and-the-steps-after-that.md

---

## 1. Philosophy & Design Mandate

### 1.1 The Original DeleGate Spirit

DeleGate (1994) was a multi-protocol application-level gateway written in C by Yutaka Sato at the Electrotechnical Laboratory, METI, Japan. Its core insight was radical for its era: **a single daemon could speak many protocols on its front face and relay to completely different protocols on its backend**, using a declarative mount table to define the mapping.

A single DeleGate process could simultaneously:
- Accept HTTP on port 8080 and relay to FTP backends
- Accept SOCKS5 on port 1080 and proxy arbitrary TCP
- Accept FTP on port 21 and relay to HTTP APIs
- Apply SSL termination on any protocol pair

The config was expressive and composable. DeleGate instances could be chained. This composability and protocol-agnosticism was its superpower.

### 1.2 The Modern Mandate

The Python prototype in this repo captures a new mandate informed by 14 months of real AI operations:

> **The prompt is not the problem. The interface is the problem.**

When an AI agent fails at infrastructure tasks, it is almost always an **impedance mismatch**: the LLM speaks stateless, tokenized JSON natively, but the infrastructure demands stateful SSH sessions, interactive shells, binary SDKs, and obscure string-parsing utilities.

DeleGate Modern solves this by acting as a **Zero-Trust Agentic Gateway**:
- Exposes simple REST/HTTP endpoints that LLMs inherently understand
- Maps them to complex, stateful backend protocols that infrastructure requires
- Injects credentials at proxy time — the agent never sees secrets
- Enforces RBAC with path whitelisting before a single packet reaches the backend
- Provides deep observability at every layer

### 1.3 Design Principles

1. **Protocol translation is a first-class operation**, not an afterthought
2. **Agents speak REST; infrastructure speaks everything else** — the proxy is the impedance matcher
3. **Zero credential exposure**: secrets are injected at the proxy boundary; never in agent context
4. **Fail closed**: unknown paths, unverified identities, and untranslatable messages are rejected by default
5. **Observability at every layer**: from raw bytes to business operations, everything is structured and queryable
6. **Composable**: proxies can be chained; a DeleGate Modern instance can use another as its backend
7. **Self-describing**: the proxy can emit its own simplified API spec for small/local AI models

---

## 2. Historical DeleGate Reference

This section documents the original DeleGate architecture as a foundation. Modern implementations should understand these concepts even when replacing them.

### 2.1 Core Concepts

**DGROOT**: DeleGate's root directory. Stored config, logs, cache, and temporary files. Equivalent to a modern application's `--data-dir`.

**The `SERVER` Parameter**: Declared what protocol DeleGate listened on.
```
delegated -P8080 SERVER=http        # HTTP proxy on port 8080
delegated -P21   SERVER=ftp         # FTP server on port 21
delegated -P1080 SERVER=socks       # SOCKS proxy on port 1080
```

**The `MOUNT` Directive**: The heart of DeleGate. Mapped virtual paths/URLs to backend resources, with optional protocol translation. This is what made DeleGate unique.
```
# HTTP frontend → FTP backend (protocol translation)
delegated SERVER=http MOUNT="/* ftp://ftpserver.internal/*"

# Mount a sub-path to a specific backend
delegated SERVER=http MOUNT="/files/* ftp://files.host/*"
delegated SERVER=http MOUNT="/mail/* smtp://mail.host/*"

# Daisy-chain through another DeleGate
delegated SERVER=http MOUNT="/* delegate://next-proxy:8081/*"

# Multiple mounts — matched in order
MOUNT="/logs/* ftp://logserver/var/log/*"
MOUNT="/uploads/* ftp://fileserver/incoming/*"
MOUNT="/* http://default-backend/*"
```

**`PERMIT` / `REJECT`**: Access control rules. Format: `protocol:destination:source`
```
PERMIT="*:*:192.168.1.0/24"         # Allow from local network
PERMIT="http:*.internal.com:*"       # Allow HTTP to internal hosts
REJECT="*:*:*"                       # Deny everything else (fail closed)
```

**`STLS`**: SSL/TLS configuration. DeleGate could terminate, originate, or bridge SSL independently on frontend and backend.
```
STLS="-fcl"           # Require SSL from clients (frontend)
STLS="fsv"            # Use SSL to server (backend)
STLS="-fcl,fsv"       # Full SSL bridge (terminate client SSL, re-encrypt to server)
STLS="fcl,-fsv"       # SSL from client, plain to server (SSL terminator)
```

**`FCL` (Filter Control Language)**: Perl-like content transformation scripts applied to request/response bodies and headers. Allowed URL rewriting, content filtering, header injection.

**`RELAY`**: Controls whether DeleGate forwards requests to other hosts.
```
RELAY=no              # Terminating proxy — no relay
RELAY=delegate        # Relay through another DeleGate instance
RELAY=*               # Relay to any host
```

**`CACHE`**: Caching configuration.
```
CACHE=do              # Enable caching
CACHE=no              # Disable caching
CACHE="do,3600"       # Cache with 1-hour TTL
```

### 2.2 Protocols Supported by Original DeleGate

| Protocol | Port | Notes |
|----------|------|-------|
| HTTP/1.0, 1.1 | 80 | Forward proxy, reverse proxy, gateway |
| HTTPS | 443 | Via STLS |
| FTP | 21 | Active and passive modes; data channel complexity |
| FTPS | 990 | FTP over SSL |
| SMTP | 25 | Mail relay |
| ESMTP | 25 | Extended SMTP with AUTH, STARTTLS |
| POP3 | 110 | Mail retrieval |
| POP3S | 995 | POP3 over SSL |
| IMAP4 | 143 | Mail access |
| IMAPS | 993 | IMAP over SSL |
| NNTP | 119 | Usenet news relay and gateway |
| TELNET | 23 | Remote terminal |
| SOCKS4 | 1080 | TCP proxy (no auth) |
| SOCKS4a | 1080 | TCP proxy with DNS delegation |
| SOCKS5 | 1080 | TCP/UDP proxy with auth |
| LDAP | 389 | Directory access |
| Gopher | 70 | Legacy hypertext |
| Finger | 79 | User info |
| Whois | 43 | Domain info |
| DNS | 53 | Resolution and caching |
| SSL/TLS | — | Applied as a layer over any protocol |

### 2.3 Key Limitations of Original DeleGate

- Single-process, fork-per-connection — poor concurrency at scale
- C codebase with minimal memory safety
- Configuration was command-line only — no hot reload
- FCL scripting was powerful but opaque
- No structured logging or metrics
- No concept of identity — access control was IP-based only
- No secrets management — credentials were in config files

---

## 3. Architecture Overview

```
                        ┌─────────────────────────────────────────────┐
                        │              DeleGate Modern                 │
                        │                                              │
  AI Agent / Client     │   ┌──────────────┐   ┌──────────────────┐   │    Backend
  ────────────────  ──► │   │   Frontend   │   │  Policy Engine   │   │
  HTTP REST             │   │   Listener   │──►│  (RBAC/Whitelist)│   │  ┌──────────┐
  WebDAV                │   │              │   └────────┬─────────┘   │  │ SFTP     │
  SOCKS5                │   │  - HTTP/REST │            │             │  │ SSH      │
  FTP                   │   │  - SOCKS5    │   ┌────────▼─────────┐   │  │ Docker   │
                        │   │  - FTP       │   │  Translation     │   │  │ HTTP API │
                        │   └──────────────┘   │  Engine          │──►│  │ FTP      │
                        │                      │  (MOUNT table)   │   │  │ SMTP     │
                        │   ┌──────────────┐   └────────┬─────────┘   │  │ Database │
  Admin / Config  ───►  │   │  Config &    │            │             │  └──────────┘
                        │   │  Mount Table │   ┌────────▼─────────┐   │
                        │   └──────────────┘   │  Auth Injector   │   │
                        │                      │  (Doppler/Vault) │   │
                        │   ┌──────────────┐   └────────┬─────────┘   │
  Monitoring  ◄──────   │   │Observability │            │             │
                        │   │    Bus       │◄───────────┘             │
                        │   └──────────────┘                          │
                        │                                              │
                        │   ┌──────────────┐                          │
  Tailscale MTLS  ───►  │   │  Identity    │                          │
                        │   │  Verifier    │                          │
                        │   └──────────────┘                          │
                        └─────────────────────────────────────────────┘
```

### 3.1 Data Flow

1. **Accept**: Frontend Listener accepts a connection on a configured protocol
2. **Identify**: Identity Verifier checks Tailscale MTLS machine identity (if configured)
3. **Parse**: Protocol Parser converts raw bytes → `InternalMessage`
4. **Policy Check**: RBAC engine evaluates `InternalMessage` against mount table + permit/reject rules
5. **Translate**: Translation Engine maps `InternalMessage` to target protocol's `InternalMessage`
6. **Inject Credentials**: Auth Injector pulls secrets from Doppler/Vault just-in-time
7. **Emit**: Protocol Emitter converts `InternalMessage` → raw bytes for backend connection
8. **Respond**: Backend response traverses the same chain in reverse
9. **Observe**: Every step emits structured events to the Observability Bus

---

## 4. Core Data Model

### 4.1 `InternalMessage`

The Universal Internal Representation. Every protocol parser produces this; every protocol emitter consumes it. This is the lingua franca of the translation engine.

```
InternalMessage {
    // Identity
    id: UUID                        // Unique message ID
    connection_id: UUID             // Owning connection ID
    timestamp: Timestamp

    // Protocol
    protocol: ProtocolId            // "http", "ftp", "smtp", "sftp", "docker", etc.
    message_type: MessageType       // Request | Response | Command | Data | Event

    // Addressing
    source_address: Option<Addr>    // IP:port of sender
    dest_address: Option<Addr>      // IP:port of intended recipient

    // HTTP-style metadata (used by all protocols)
    command: Option<String>         // HTTP method, FTP command, SMTP verb, etc.
    resource_path: Option<String>   // URL path, file path, topic name, etc.
    headers: Map<String, String>    // Protocol headers/metadata

    // Payload
    body: Option<Bytes>             // Message body / file content

    // Response fields
    status_code: Option<Int>        // HTTP status, FTP response code, SMTP reply code
    status_message: Option<String>

    // Resource metadata
    resource_size: Option<Int64>
    resource_permissions: Option<String>
    resource_modified: Option<Timestamp>

    // Protocol-specific extras (escape hatch for fields not in common model)
    protocol_data: Map<String, Value>
}
```

### 4.2 `Connection`

Represents a single client connection through the proxy.

```
Connection {
    id: UUID
    created_at: Timestamp
    state: ConnectionState          // Connecting | Active | Translating | Closing | Closed

    // Identity
    client_addr: Addr
    tailscale_identity: Option<TailscaleIdentity>

    // Matched mount
    matched_mount: Option<MountEntry>

    // Stats
    bytes_in: Int64
    bytes_out: Int64
    messages_in: Int
    messages_out: Int
    translation_count: Int
    error_count: Int
}
```

### 4.3 `TranslationRule`

```
TranslationRule {
    name: String
    source_protocol: ProtocolId
    target_protocol: ProtocolId
    description: String

    // Predicate: does this rule apply to this message?
    condition: (InternalMessage) -> Bool

    // Transform: produce target message from source
    translator: (InternalMessage) -> Result<InternalMessage, TranslationError>

    priority: Int                   // Higher = evaluated first among matching rules
}
```

---

## 5. The Mount Table

The mount table is the central configuration concept, directly inspired by the original DeleGate MOUNT directive. It defines how virtual frontend paths map to backend resources, optionally with protocol translation.

### 5.1 Mount Entry

```
MountEntry {
    // Matching
    frontend_protocol: Option<ProtocolId>   // Which frontend protocol triggers this (null = any)
    path_pattern: GlobPattern               // e.g. "/files/*", "/*"

    // Backend
    backend_url: BackendUrl                 // e.g. "sftp://blackbarn.internal/var/www/*"
                                            //      "docker://gastown/var/app/*"
                                            //      "http://api.internal/*"
                                            //      "ftp://legacy-server/*"

    // Access control
    readonly: Bool                          // If true, reject write operations at policy stage
    allowed_methods: Option<Set<String>>    // e.g. ["GET", "HEAD"] — null means all

    // Auth
    auth_provider: Option<AuthProvider>     // "doppler:SECRET_NAME", "vault:path", "none"

    // Observability
    name: String                            // Human-readable name for logs/metrics
}
```

### 5.2 Mount Resolution Algorithm

1. Collect all mounts whose `frontend_protocol` matches (or is null)
2. Filter to those whose `path_pattern` glob matches the request path
3. Sort by specificity (longer/more-specific patterns win over shorter/wildcards)
4. Take the first match
5. If no match: return `403 Forbidden` — fail closed

### 5.3 Path Traversal Protection

Before mount resolution, normalize the request path:
1. URL-decode percent-encoding
2. Resolve `.` and `..` segments
3. Collapse double slashes
4. Reject paths that, after normalization, escape the mount root (path traversal)
5. Log and reject with `403` if traversal detected — never forward

This is the core security guarantee: **traversal attacks are blocked mathematically before any packet reaches the backend.**

### 5.4 Example Mount Configurations

```toml
# SFTP gateway: agent does HTTP PUT → proxy does SFTP upload
[[mount]]
name = "dev-files"
path_pattern = "/dev/*"
backend_url = "sftp://blackbarn.internal/var/www/dev/*"
auth_provider = "doppler:BLACKBARN_SSH_KEY"

# Logs: readonly SFTP mount
[[mount]]
name = "app-logs"
path_pattern = "/logs/*"
backend_url = "sftp://blackbarn.internal/var/log/app/*"
readonly = true
auth_provider = "doppler:BLACKBARN_SSH_KEY"

# Docker file extraction
[[mount]]
name = "container-files"
path_pattern = "/docker/:container/*"
backend_url = "docker:///var/run/docker.sock"
allowed_methods = ["GET"]

# Legacy FTP server presented as HTTP
[[mount]]
name = "legacy-ftp"
path_pattern = "/archive/*"
backend_url = "ftp://legacy.internal/pub/*"
readonly = true

# Default: HTTP passthrough proxy
[[mount]]
name = "default"
path_pattern = "/*"
backend_url = "http://$HOST$PATH"
```

---

## 6. Policy Engine (RBAC)

The policy engine is a mandatory stage between parsing and translation. It evaluates every `InternalMessage` before any backend contact.

### 6.1 Policy Rule

```
PolicyRule {
    name: String
    priority: Int                       // Higher = evaluated first

    // Conditions (all must match — AND semantics)
    source_identity: Option<IdentityMatcher>    // Tailscale node name, IP CIDR
    frontend_protocol: Option<ProtocolId>
    method_pattern: Option<Glob>                // "GET", "PUT", "RETR", "*"
    path_pattern: Option<Glob>                  // "/dev/*", "/*"

    // Action
    action: PolicyAction                        // Allow | Deny | RateLimit | Log
    reason: String                              // Included in logs/403 response
}
```

### 6.2 Evaluation Order

1. Evaluate rules in priority order (highest first)
2. First matching rule's action applies
3. If no rule matches: **DENY** (fail closed)
4. Log all deny decisions with reason and identity

### 6.3 Built-in Safety Rules (Always Active)

These cannot be overridden by configuration:

1. **Path traversal block**: Any normalized path containing `..` after mount-relative resolution → `403`
2. **Null byte block**: Any path or parameter containing `\x00` → `400`
3. **Oversized header block**: Any single header > 8KB → `431`
4. **Max header count**: More than 100 headers → `431`

### 6.4 Example Policy Configuration

```toml
# Only seattle-ai-worker node can write to /dev/
[[policy]]
name = "dev-write-restricted"
priority = 100
source_identity = { tailscale_node = "seattle-ai-worker" }
path_pattern = "/dev/*"
method_pattern = "PUT"
action = "allow"

# Anyone on tailnet can read logs
[[policy]]
name = "logs-readonly"
priority = 90
path_pattern = "/logs/*"
method_pattern = "GET"
action = "allow"

# Block all writes to logs (redundant with mount readonly, defense in depth)
[[policy]]
name = "logs-no-write"
priority = 95
path_pattern = "/logs/*"
method_pattern = "*"
action = "deny"
reason = "logs mount is readonly"

# Default deny
[[policy]]
name = "default-deny"
priority = 0
path_pattern = "/*"
method_pattern = "*"
action = "deny"
reason = "no matching allow rule"
```

---

## 7. Protocol Specifications

Each protocol requires a **Parser** (raw bytes → `InternalMessage`), an **Emitter** (`InternalMessage` → raw bytes), and where applicable, a **Backend Connector** (manages the stateful backend connection).

### 7.1 HTTP (Frontend and Backend)

**Frontend role**: Primary interface for AI agents. Accept HTTP/1.1 requests.

**Parser requirements**:
- Parse request line: method, path, HTTP version
- Parse headers (case-insensitive keys, trimmed values)
- Respect `Content-Length` for body framing
- Support chunked transfer encoding
- Detect and parse `Host` header for virtual hosting
- Produce `InternalMessage` with `message_type=Request`, `command=method`, `resource_path=path`

**Emitter requirements**:
- Emit valid HTTP/1.1 request or response
- Add `Content-Length` header for bodies
- Proper CRLF line endings

**Fields mapping**:
| InternalMessage field | HTTP mapping |
|---|---|
| `command` | Method (GET, POST, PUT, DELETE, etc.) |
| `resource_path` | Request path (without query string) |
| `headers` | HTTP headers |
| `body` | Request/response body |
| `status_code` | HTTP status code |
| `status_message` | HTTP reason phrase |
| `protocol_data["query"]` | Query string parameters |
| `protocol_data["version"]` | HTTP version string |

### 7.2 FTP (Frontend and Backend)

FTP is a stateful protocol with a **control channel** (commands) and a separate **data channel** (file transfers). This is the primary complexity.

**Control channel commands to support**:

| Command | Operation | InternalMessage mapping |
|---|---|---|
| `USER <name>` | Begin authentication | `command=USER`, `protocol_data["username"]` |
| `PASS <password>` | Complete authentication | `command=PASS` — **never log this** |
| `CWD <path>` | Change directory | `command=CWD`, `resource_path` |
| `PWD` | Get current directory | `command=PWD` |
| `LIST [path]` | List directory (detailed) | `command=LIST`, `resource_path` |
| `NLST [path]` | List names only | `command=NLST`, `resource_path` |
| `RETR <file>` | Download file | `command=RETR`, `resource_path` |
| `STOR <file>` | Upload file | `command=STOR`, `resource_path` |
| `DELE <file>` | Delete file | `command=DELE`, `resource_path` |
| `MKD <dir>` | Create directory | `command=MKD`, `resource_path` |
| `RMD <dir>` | Remove directory | `command=RMD`, `resource_path` |
| `RNFR <file>` | Rename from | `command=RNFR`, `resource_path` |
| `RNTO <file>` | Rename to | `command=RNTO`, `resource_path` |
| `PASV` | Passive mode | `command=PASV` |
| `PORT h,h,h,h,p,p` | Active mode | `command=PORT` |
| `TYPE A\|I` | Set transfer type | `command=TYPE` |
| `QUIT` | Disconnect | `command=QUIT` |
| `NOOP` | Keep-alive | `command=NOOP` |

**FTP Response codes** (map to status_code):
- 1xx: Positive Preliminary
- 2xx: Positive Completion
- 3xx: Positive Intermediate (need more info)
- 4xx: Transient Negative (retry possible)
- 5xx: Permanent Negative

**Data channel handling**:
- PASV mode: server opens listening port, client connects to it
- PORT mode: client specifies address, server connects to client
- Both modes transfer the actual file bytes
- Implementation must manage the data channel lifecycle independently of control channel

### 7.3 SFTP (Backend Connector Only)

SFTP runs over SSH. The proxy accepts HTTP/WebDAV on its frontend, and translates to SFTP on the backend. The agent never sees SSH.

**Backend connector requirements**:
- Establish SSH connection using key from auth injector (never from agent context)
- Support key-based auth via `ssh-agent` socket or in-memory key
- Support password auth (from Doppler, never from agent)
- SFTP subsystem negotiation over SSH channel
- Operations to support:

| HTTP method + path | SFTP operation |
|---|---|
| `GET /path/to/file` | `open` + `read` + `close` |
| `PUT /path/to/file` (with body) | `open(write)` + `write` + `close` |
| `DELETE /path/to/file` | `remove` |
| `GET /path/to/dir/` (directory) | `opendir` + `readdir` + `closedir` |
| `MKCOL /path/to/dir` (WebDAV) | `mkdir` |

**Security**: Never store SSH private keys in memory longer than the connection duration. Use `ssh-agent` protocol where possible so the private key bytes never enter the process.

### 7.4 SOCKS5 (Frontend)

SOCKS5 (RFC 1928) is the any-to-any TCP proxy frontend. When an agent sets `ALL_PROXY=socks5://delegate:1080`, all TCP connections from that agent flow through this listener.

**Handshake sequence**:
1. Client → Server: version (0x05), num auth methods, auth method list
2. Server → Client: version (0x05), chosen method
3. Auth negotiation (if method != NO_AUTH)
4. Client → Server: CONNECT request (version, command, address type, dest addr, dest port)
5. Server → Client: reply (version, status, bound addr, bound port)
6. Bidirectional data relay begins

**Commands to support**:
- `CONNECT` (0x01): TCP stream relay — required
- `BIND` (0x02): Accept incoming — optional
- `UDP ASSOCIATE` (0x03): UDP relay — optional

**Auth methods**:
- No authentication (0x00)
- Username/password (0x02) — for future machine identity binding

**RBAC integration**: After CONNECT request, extract `(dest_addr, dest_port)` and evaluate against policy rules before establishing backend connection. If policy denies: send SOCKS5 error reply and close.

**Blocked targets** (always, regardless of policy):
- RFC 1918 addresses not explicitly whitelisted: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
- Loopback: 127.0.0.0/8, ::1
- Link-local: 169.254.0.0/16

### 7.5 SMTP (Backend Connector)

Used for translating agent HTTP requests into email sends. The proxy accepts `POST /mail/send` with a JSON body and translates to SMTP.

**HTTP → SMTP mapping**:
```
POST /mail/send
Content-Type: application/json

{
  "to": ["user@example.com"],
  "from": "agent@internal.company.com",
  "subject": "Deployment report",
  "body": "Deploy completed at 14:32 UTC"
}
```

Translates to an SMTP session: `EHLO`, `MAIL FROM`, `RCPT TO`, `DATA`, `QUIT`.

### 7.6 Docker Archive API (Backend Connector)

Translates HTTP `GET` requests into Docker Archive API calls to extract files from running containers without shell access.

**Endpoint pattern**: `GET /docker/:container/:path`

**Backend operation**:
1. Connect to `/var/run/docker.sock` (Unix socket)
2. Call `GET /containers/:container/archive?path=:path`
3. Docker returns a tar archive
4. Extract the requested file from the tar in memory
5. Stream file bytes back to agent as HTTP response body

**Error cases**:
- Container not found → `404 Not Found`
- Path not found in container → `404 Not Found`
- Container not running → `503 Service Unavailable`
- Path is a directory → Return tar listing as JSON, or `Content-Type: application/x-tar`

**Security**: Never pass unsanitized container names or paths to the Docker socket. Validate container name against allowlist in mount config. Normalize path and reject traversal before calling Docker API.

### 7.7 Additional Protocols (Future)

These are specified for roadmap completeness:

**LDAP**: `GET /ldap/search?base=dc=example,dc=com&filter=(uid=*)` → LDAP search request. Return JSON array of entries.

**Kafka / MQTT**: `GET /telemetry/latest` returns buffered messages as JSON. `POST /telemetry` publishes a message. Proxy maintains the stateful consumer/producer connection.

**PostgreSQL / MySQL**: `POST /db/query` with `{"sql": "SELECT ..."}` — proxy validates the SQL (reject DDL, DML only if configured), executes against backend, returns JSON result set.

**Kubernetes API**: `GET /k8s/pods?namespace=prod&status=failing` — translates to complex kubectl-equivalent API call, returns simplified JSON.

---

## 8. Translation Rules

Translation rules map `InternalMessage` from one protocol to another. Rules are evaluated in priority order; first match wins.

### 8.1 Built-in HTTP ↔ FTP Rules

| Rule name | Source | Target | Condition | Translation |
|---|---|---|---|---|
| `http_get_to_ftp_retr` | http | ftp | `message_type=request AND command=GET` | `RETR <path>` |
| `http_put_to_ftp_stor` | http | ftp | `message_type=request AND command=PUT` | `STOR <path>` with body |
| `http_delete_to_ftp_dele` | http | ftp | `message_type=request AND command=DELETE` | `DELE <path>` |
| `http_get_dir_to_ftp_list` | http | ftp | `message_type=request AND command=GET AND path ends /` | `LIST <path>` |
| `ftp_retr_to_http_get` | ftp | http | `command=RETR` | `GET /<path>` |
| `ftp_stor_to_http_put` | ftp | http | `command=STOR` | `PUT /<path>` with body |
| `ftp_2xx_to_http_200` | ftp | http | `message_type=response AND 200≤code<300` | `200 OK` |
| `ftp_4xx_to_http_5xx` | ftp | http | `message_type=response AND 400≤code<500` | `503 Service Unavailable` |
| `ftp_5xx_to_http_4xx` | ftp | http | `message_type=response AND 500≤code<600` | `400 Bad Request` |

### 8.2 HTTP → SFTP Rules

| Rule name | Source | Condition | SFTP operation |
|---|---|---|---|
| `http_get_to_sftp_read` | http | `GET`, non-directory path | `open` + `read` + `close` |
| `http_put_to_sftp_write` | http | `PUT` with body | `open(write)` + `write` + `close` |
| `http_delete_to_sftp_remove` | http | `DELETE` | `remove` |
| `http_get_to_sftp_list` | http | `GET`, path ends `/` | `opendir` + `readdir` → JSON |

### 8.3 HTTP → Docker Archive Rules

| Rule name | Source | Condition | Docker operation |
|---|---|---|---|
| `http_get_to_docker_archive` | http | `GET /docker/:container/*` | Archive API + tar extract |
| `http_get_to_docker_list` | http | `GET /docker/:container/*/` | Archive API + tar list → JSON |

### 8.4 Rule Registration API

Implementations must provide a way to register custom rules at runtime (for plugins and testing):

```
register_rule(rule: TranslationRule) -> Result<(), RegistrationError>
unregister_rule(name: String) -> Result<(), NotFoundError>
list_rules() -> List<TranslationRuleSummary>
```

---

## 9. Authentication & Secret Injection

### 9.1 Auth Provider Interface

```
AuthProvider {
    name: String
    provider_type: ProviderType     // Doppler | Vault | EnvVar | SshAgent | None

    // Retrieve a credential by logical name
    get_secret(name: String) -> Result<SecretValue, AuthError>

    // For SSH: inject key into an in-memory ssh-agent
    get_ssh_agent(key_name: String) -> Result<SshAgentHandle, AuthError>
}
```

### 9.2 Doppler Integration

Doppler provides just-in-time secret retrieval. The proxy uses the Doppler service token (set in the proxy's own environment, never in agent context) to fetch secrets at connection time.

```
DopplerProvider {
    service_token: String       // From proxy's own env: DOPPLER_TOKEN
    project: String
    config: String              // e.g. "production", "development"

    // Cache secrets with short TTL (e.g. 30 seconds)
    cache_ttl: Duration
}
```

**Secret resolution**: When a mount entry specifies `auth_provider = "doppler:SECRET_NAME"`, the proxy calls Doppler to retrieve the value. The secret value is used for the backend connection and then discarded. It is never written to disk, never included in logs, and never returned in responses.

### 9.3 Tailscale MTLS Identity

When the proxy is running on a Tailscale network, it can cryptographically verify the identity of connecting clients via the Tailscale LocalAPI.

```
TailscaleIdentity {
    node_name: String           // e.g. "seattle-ai-worker"
    node_id: String
    ip_addresses: List<Addr>
    tags: List<String>
    is_authenticated: Bool
}

verify_identity(client_addr: Addr) -> Result<TailscaleIdentity, IdentityError>
```

**Integration point**: After accepting a TCP connection, before parsing any protocol data, query the Tailscale LocalAPI at `/var/run/tailscale/tailscaled.sock` to identify the connecting client. Attach the resulting `TailscaleIdentity` to the `Connection` object.

**RBAC usage**: Policy rules can specify `source_identity = { tailscale_node = "seattle-ai-worker" }` to restrict operations to specific Tailscale nodes. This provides cryptographically verified access control — no passwords, no bearer tokens.

---

## 10. Observability

Observability is a first-class requirement, not an afterthought. The system must be instrumented at every layer.

### 10.1 Observability Levels

```
ObservabilityLevel {
    Silent,     // No output
    Error,      // Errors and panics only
    Info,       // Connection lifecycle, translation results
    Debug,      // Detailed operation tracing
    Protocol,   // Protocol message content (headers, commands)
    Bytes,      // Raw bytes (hex-dumped, size-limited)
    Trace,      // Everything including body content
}
```

### 10.2 Event Types

**NetworkEvent**: Emitted when bytes are received or sent.
```
NetworkEvent {
    event_id, timestamp, connection_id,
    direction: Inbound | Outbound,
    protocol: ProtocolId,
    bytes_count: Int,
    raw_bytes: Option<Bytes>    // Only at Bytes level; truncated at >1KB
}
```

**ProtocolEvent**: Emitted when a message is parsed or emitted.
```
ProtocolEvent {
    event_id, timestamp, connection_id,
    protocol: ProtocolId,
    direction: Parsed | Emitted,
    message_type: String,
    command: Option<String>,
    resource_path: Option<String>,
    status_code: Option<Int>,
    headers: Option<Map<String,String>>,    // Only at Protocol level
    parsing_duration_ms: Float
}
```

**TranslationEvent**: Emitted when a translation is attempted.
```
TranslationEvent {
    event_id, timestamp, connection_id,
    rule_name: String,
    source_protocol: ProtocolId,
    target_protocol: ProtocolId,
    success: Bool,
    duration_ms: Float,
    error: Option<String>
}
```

**PolicyEvent**: Emitted when a policy decision is made.
```
PolicyEvent {
    event_id, timestamp, connection_id,
    rule_name: String,
    action: Allow | Deny,
    reason: String,
    source_identity: Option<String>,
    path: String,
    method: String
}
```

**AuthEvent**: Emitted when secrets are accessed. **Never log secret values.**
```
AuthEvent {
    event_id, timestamp, connection_id,
    provider: String,
    secret_name: String,        // Logical name only, never the value
    success: Bool,
    duration_ms: Float,
    error: Option<String>
}
```

### 10.3 Structured Log Format

All events must be emitted as structured JSON. Minimum fields for every log line:
```json
{
  "ts": "2026-03-30T14:23:01.456Z",
  "level": "info",
  "event": "translation_successful",
  "connection_id": "01J7...",
  "rule": "http_get_to_sftp_read",
  "source": "http",
  "target": "sftp",
  "duration_ms": 12.3
}
```

### 10.4 Metrics

The proxy must expose the following metrics (Prometheus-compatible format preferred):

```
# Counters
delegate_connections_total{frontend_protocol, status}
delegate_messages_total{protocol, message_type}
delegate_translations_total{rule, success}
delegate_policy_decisions_total{action, rule}
delegate_auth_requests_total{provider, success}
delegate_bytes_total{direction}

# Histograms
delegate_translation_duration_ms{rule}
delegate_connection_duration_ms{frontend_protocol}
delegate_auth_duration_ms{provider}

# Gauges
delegate_active_connections{frontend_protocol}
delegate_event_buffer_size
```

Metrics endpoint: `GET /metrics` (if admin HTTP server is enabled)

### 10.5 Health Check

`GET /health` returns:
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "active_connections": 5,
  "protocols": ["http", "ftp", "sftp", "socks5"],
  "auth_providers": [{"name": "doppler", "status": "ok"}],
  "tailscale": {"connected": true, "node": "delegate-gateway"}
}
```

---

## 11. Configuration Format

Configuration is TOML-based. Supports hot reload on SIGHUP.

### 11.1 Top-Level Structure

```toml
[server]
admin_port = 9090               # Admin HTTP server (metrics, health, config inspect)
log_level = "info"              # ObservabilityLevel value
log_format = "json"             # "json" | "text"

[tailscale]
enabled = true
socket = "/var/run/tailscale/tailscaled.sock"
require_auth = false            # If true, reject connections from non-Tailscale clients

[doppler]
enabled = true
# Token comes from environment: DOPPLER_TOKEN
project = "delegate-modern"
config = "production"
cache_ttl_seconds = 30

[[listener]]
protocol = "http"
port = 8080
# Optional: bind to specific address
address = "0.0.0.0"

[[listener]]
protocol = "socks5"
port = 1080
address = "127.0.0.1"          # SOCKS only on localhost by default

[[listener]]
protocol = "ftp"
port = 2121

[[mount]]
name = "dev-files"
path_pattern = "/dev/*"
backend_url = "sftp://blackbarn.internal/var/www/dev/*"
auth_provider = "doppler:BLACKBARN_SSH_KEY"
readonly = false

[[mount]]
name = "logs"
path_pattern = "/logs/*"
backend_url = "sftp://blackbarn.internal/var/log/app/*"
auth_provider = "doppler:BLACKBARN_SSH_KEY"
readonly = true

[[mount]]
name = "container-config"
path_pattern = "/docker/:container/*"
backend_url = "docker:///var/run/docker.sock"
allowed_methods = ["GET"]

[[policy]]
name = "dev-write-seattle-only"
priority = 100
source_identity = { tailscale_node = "seattle-ai-worker" }
path_pattern = "/dev/*"
method_pattern = "*"
action = "allow"

[[policy]]
name = "logs-read-all"
priority = 90
path_pattern = "/logs/*"
method_pattern = "GET"
action = "allow"

[[policy]]
name = "default-deny"
priority = 0
path_pattern = "/*"
method_pattern = "*"
action = "deny"
reason = "no matching allow rule"
```

### 11.2 Environment Variable Overrides

Any config key can be overridden with env vars using the pattern `DELEGATE_SECTION_KEY`:
```
DELEGATE_SERVER_LOG_LEVEL=debug
DELEGATE_TAILSCALE_ENABLED=false
DOPPLER_TOKEN=dp.st.xxxxx
```

### 11.3 Configuration Validation

On startup (and on SIGHUP reload), the configuration must be validated:
1. All `backend_url` schemes are recognized protocols
2. All `auth_provider` references point to configured providers
3. Mount path patterns are valid globs
4. Policy rules have valid priorities (no duplicates)
5. At least one `[[listener]]` is defined
6. `default-deny` policy exists (warn if absent)

Validation failures on startup: fatal exit.
Validation failures on reload: log error, keep running with old config.

---

## 12. CLI Interface

```
delegate-modern [OPTIONS] SUBCOMMAND

SUBCOMMANDS:
    serve           Start the proxy server
    translate       Translate a single message (stdin → stdout, for testing)
    analyze         Parse and display protocol analysis of input
    check           Validate configuration file
    version         Print version

SERVE OPTIONS:
    --config <path>     Config file (default: ./delegate.toml)
    --port <n>          Override listener port (for quick testing)
    --protocol <proto>  Override listener protocol
    --log-level <level> Override log level
    --admin-port <n>    Admin HTTP server port

TRANSLATE OPTIONS:
    --source <proto>    Source protocol
    --target <proto>    Target protocol
    (reads InternalMessage JSON from stdin, writes translated JSON to stdout)

ANALYZE OPTIONS:
    --protocol <proto>  Protocol to parse as (or auto-detect)
    (reads raw bytes from stdin, writes parsed InternalMessage JSON to stdout)

EXAMPLES:
    # Start with config file
    delegate-modern serve --config /etc/delegate/delegate.toml

    # Quick HTTP proxy on port 8080
    delegate-modern serve --port 8080 --protocol http

    # Translate HTTP to FTP (for testing rules)
    echo '{"protocol":"http","command":"GET","resource_path":"/file.txt"}' \
      | delegate-modern translate --source http --target ftp

    # Analyze FTP traffic
    printf "RETR document.pdf\r\n" | delegate-modern analyze --protocol ftp

    # Validate config
    delegate-modern check --config delegate.toml
```

---

## 13. Self-Describing Endpoint

The proxy can emit a simplified API description for small/local AI models.

### 13.1 `GET /v1/endpoints.json`

Returns a plain-English, ultra-simplified OpenAPI-like spec of all active mounts. Designed to fit in a small model's context window.

```json
{
  "version": "1",
  "description": "DeleGate Modern proxy. Use these endpoints to access files and infrastructure. All operations use simple HTTP.",
  "endpoints": [
    {
      "name": "dev-files",
      "description": "Read and write files in the development web directory on blackbarn server.",
      "examples": [
        "GET /dev/app.js — read a file",
        "PUT /dev/app.js — write a file (send file content as request body)",
        "DELETE /dev/old.js — delete a file",
        "GET /dev/ — list directory"
      ],
      "allowed_methods": ["GET", "PUT", "DELETE"],
      "readonly": false
    },
    {
      "name": "logs",
      "description": "Read application log files. Read-only.",
      "examples": [
        "GET /logs/app.log — read log file",
        "GET /logs/ — list available logs"
      ],
      "allowed_methods": ["GET"],
      "readonly": true
    },
    {
      "name": "container-config",
      "description": "Extract files from running Docker containers. Read-only.",
      "examples": [
        "GET /docker/gastown/var/app/config.json — read config from gastown container",
        "GET /docker/nginx/etc/nginx/nginx.conf — read nginx config"
      ],
      "allowed_methods": ["GET"],
      "readonly": true
    }
  ]
}
```

---

## 14. Implementation Phases

### Phase 1: Core Engine (MVP)

**Goal**: A working proxy that accepts HTTP, applies mount/policy rules, and can translate to HTTP backends. Proves the architecture.

**Deliverables**:
- [ ] `InternalMessage` data model
- [ ] `ProtocolRegistry` with plugin registration
- [ ] HTTP parser and emitter (frontend + backend)
- [ ] Mount table with glob matching
- [ ] Path traversal protection (always active)
- [ ] Basic policy engine (path + method rules)
- [ ] Translation engine with rule registration
- [ ] HTTP ↔ FTP translation rules (from Python prototype)
- [ ] Structured JSON logging (ObservabilityBus)
- [ ] CLI: `serve`, `translate`, `analyze`, `check`
- [ ] TOML config loading + validation
- [ ] `GET /health` admin endpoint
- [ ] Basic test suite: each translator, path traversal, policy evaluation

### Phase 2: Zero-Trust Security

**Goal**: Proxy is safe for an AI agent to touch.

**Deliverables**:
- [ ] SFTP backend connector (with in-memory key handling)
- [ ] Doppler secret injector
- [ ] Tailscale MTLS identity verification
- [ ] RBAC rules with identity-aware matching
- [ ] AuthEvent observability (without logging secret values)
- [ ] `GET /v1/endpoints.json` self-describing endpoint
- [ ] PolicyEvent observability
- [ ] Test: path traversal attacks blocked
- [ ] Test: Tailscale identity enforcement
- [ ] Test: credential injection (verify value never appears in logs)

### Phase 3: Protocol Connectors

**Goal**: High-value connectors that solve specific AI agent friction cases.

**Deliverables**:
- [ ] Docker Archive API connector
- [ ] SOCKS5 frontend listener
- [ ] FTP backend connector (with data channel management)
- [ ] SMTP backend connector
- [ ] Prometheus `/metrics` endpoint
- [ ] Hot reload on SIGHUP
- [ ] Test: SOCKS5 handshake + relay
- [ ] Test: Docker file extraction
- [ ] Test: SMTP send via HTTP POST

### Phase 4: AI-Native Control Plane

**Goal**: Advanced features for AI orchestration scenarios.

**Deliverables**:
- [ ] PostgreSQL backend connector (query firewall: allow SELECT, block DDL)
- [ ] Kafka/MQTT bridge (stateful consumer with REST polling interface)
- [ ] Config generator endpoint (English → policy YAML via Claude API)
- [ ] Enhanced `GET /v1/endpoints.json` with token counts and context hints
- [ ] Distributed tracing (OpenTelemetry spans)
- [ ] Benchmarks: throughput and latency under AI agent workload patterns

---

## 15. Test Requirements

### 15.1 Unit Tests

- Every `TranslationRule`: input message → expected output message
- Path normalization: 20+ traversal attack patterns must produce `403`
- Mount resolution: specificity ordering, no-match → deny
- Policy evaluation: allow/deny, ordering, fail-closed
- Config validation: valid config, each invalid config type

### 15.2 Integration Tests

- HTTP → FTP translation end-to-end (against a real FTP server or mock)
- HTTP → SFTP (against a test sshd)
- SOCKS5 handshake and relay
- Docker Archive API extraction
- Tailscale identity verification (mock tailscaled socket)
- Doppler secret injection (mock Doppler API)

### 15.3 Security Tests (Must All Pass)

| Test | Expected result |
|---|---|
| `GET /../../../etc/passwd` | `403 Forbidden` |
| `GET /dev/%2e%2e%2f%2e%2e%2fetc/passwd` (encoded traversal) | `403 Forbidden` |
| `GET /dev/\x00null` (null byte) | `400 Bad Request` |
| 200 headers in one request | `431 Request Header Fields Too Large` |
| Request from non-whitelisted Tailscale node to restricted path | `403 Forbidden` |
| Credential value appears in any log line | Test FAILS |
| SOCKS5 CONNECT to 192.168.1.1 (not whitelisted) | SOCKS5 error reply |

### 15.4 Observability Tests

- Verify every successful translation emits a `TranslationEvent`
- Verify every policy denial emits a `PolicyEvent` with reason
- Verify auth access emits `AuthEvent` without secret value
- Verify `bytes` log level emits hex dump for packets ≤ 1KB
- Verify `silent` level produces zero output

---

## 16. MoonBit-Specific Implementation Notes

When implementing in MoonBit, the following patterns apply based on the language's characteristics (see `~/.moon/AGENTS.md` for full reference):

### 16.1 Module Structure

```
delegate_modern/
├── moon.mod.json
├── moon.pkg.json                   # Root package (re-exports)
├── types.mbt                       # InternalMessage, Connection, etc.
├── cmd/
│   └── main/
│       ├── moon.pkg.json           # {"is_main": true}
│       └── main.mbt               # CLI entry point
├── core/
│   ├── moon.pkg.json
│   ├── registry.mbt               # ProtocolRegistry
│   ├── engine.mbt                 # TranslationEngine
│   ├── policy.mbt                 # PolicyEngine
│   └── mount.mbt                  # MountTable
├── protocols/
│   ├── moon.pkg.json
│   ├── http.mbt                   # HTTP parser + emitter
│   ├── ftp.mbt                    # FTP parser + emitter
│   └── socks5.mbt                 # SOCKS5 frontend
├── connectors/
│   ├── moon.pkg.json
│   ├── sftp.mbt                   # SFTP backend
│   └── docker.mbt                 # Docker Archive API
├── auth/
│   ├── moon.pkg.json
│   ├── doppler.mbt
│   └── tailscale.mbt
└── observability/
    ├── moon.pkg.json
    └── bus.mbt
```

### 16.2 Error Handling

MoonBit uses checked exceptions. Functions that can fail should declare `raise`:

```moonbit
pub fn translate(msg : InternalMessage, target : String) -> InternalMessage raise TranslationError {
  match find_rule(msg, target) {
    None => raise TranslationError::NoRuleFound(msg.protocol, target)
    Some(rule) => rule.translator(msg)
  }
}
```

### 16.3 Async I/O

Use `moonbitlang/async` for network I/O (file system, process, socket, HTTP support). TCP listeners, SFTP connections, and Docker socket calls all require async.

### 16.4 Pattern Matching

Prefer exhaustive pattern matching for protocol dispatch:

```moonbit
fn message_type_to_string(t : MessageType) -> String {
  match t {
    Request => "request"
    Response => "response"
    Command => "command"
    Data => "data"
  }
}
```

### 16.5 The `///|` Block Delimiter

Separate all top-level items with `///|` comments. Keep files focused and small. Split large protocol implementations into multiple files (e.g., `http_parser.mbt`, `http_emitter.mbt`).

---

## 17. Glossary

| Term | Definition |
|---|---|
| **DeleGate** | Original multi-protocol proxy by Yutaka Sato (1994–2007), ETL/METI Japan |
| **DGROOT** | Original DeleGate root directory; analog to `--data-dir` |
| **MOUNT** | Original DeleGate directive mapping virtual paths to backend URLs |
| **InternalMessage** | Universal internal representation used as translation lingua franca |
| **Frontend** | The protocol face the client (AI agent) sees |
| **Backend** | The protocol face the real infrastructure sees |
| **Translation Rule** | A named function that maps one InternalMessage to another across protocols |
| **Mount Entry** | A config record mapping a path pattern to a backend URL |
| **Policy Rule** | An RBAC rule that allows or denies requests |
| **Auth Injector** | Component that retrieves credentials from Doppler/Vault and injects them into backend connections |
| **Observability Bus** | Central event bus; all components emit events to it |
| **Tailscale MTLS** | Cryptographic machine identity verification via Tailscale LocalAPI |
| **Impedance Mismatch** | The mismatch between an LLM's native interface (stateless JSON) and infrastructure interfaces (stateful, binary, complex) |
| **Path Traversal** | Attack using `../` sequences to escape the intended path — blocked unconditionally |
| **FCL** | Filter Control Language — original DeleGate's content transformation scripting layer |
| **STLS** | Original DeleGate SSL/TLS configuration parameter |
