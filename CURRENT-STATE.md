# Current state

Last verified: 2026-09-04

## Verified

- The private Gitea repository `ryan/new-delegate` exists on Biggie.
- The local working copy uses Go 1.25 on Darwin/arm64.
- The design, compatibility ledger, and iterate-bot workflow are recorded.
- A minimal `delegate` command exists so portability builds exercise a real Go
  package. It has no gateway behavior yet.
- Canonical `Config` and `Server` values reject missing fields and duplicate
  names without mutating caller-owned input.
- The iteration-1 focused test and complete local gate pass on Darwin/arm64.
- `SERVER=http` and compact or separated `-P` arguments parse into canonical
  server configuration. Known protocols receive their conventional default
  ports; conflicting repetitions, malformed ports, and unknown directives are
  fatal. Golden and table tests pass.
- Canonical mounts validate absolute frontend patterns and HTTP, HTTPS, FTP,
  or delegate backend URLs. Legacy quoted `MOUNT="path target"` directives,
  multiple ordered mappings, and explicit `priority=N` parse without runtime
  side effects. Invalid wildcards, schemes, hosts, options, and priorities are
  fatal.
- Request paths are decoded once and normalized before matching. Traversal,
  encoded separators, double encoding, backslashes, invalid UTF-8, and control
  bytes fail closed. A bounded fuzz run completed 334,664 executions without a
  panic, and accepted paths are tested for idempotence.
- Mount resolution chooses longest specificity, then explicit priority;
  no-match and equal-winner ambiguity are typed failures.
- The policy kernel defaults to deny, matches source/protocol/method/mount
  constraints, chooses highest priority, and gives rejection precedence at an
  equal priority. Decisions carry stable reason codes, and enforcement tests
  prove a denied callback cannot execute.
- An HTTP frontend now maps a request to a protocol-neutral Fetch operation,
  authorizes it, invokes the HTTP connector, and propagates the backend status,
  end-to-end headers, and body. An in-process acceptance test verifies path and
  query rewriting. A denial test verifies HTTP 403 and zero backend calls.
- Biggie Woodpecker repository 5 tracks a private GitHub CI mirror while the
  canonical `origin` remains `ryan/new-delegate` on Gitea. The local-backend
  pipeline bootstraps a checksum-pinned Go 1.25.6 toolchain in `/tmp` because
  the intentionally small agent image does not contain Go.
- Biggie Woodpecker pipeline 4 passed in 31 seconds at commit `fa799d6`,
  validating formatting, vet, all tests, and CGO-free Darwin/arm64,
  Linux/amd64, Linux/arm64, and illumos/amd64 builds.
- Original `PERMIT="protocol:destination:source"` and `REJECT=...` directives
  parse into canonical rules with explicit priorities preserving first-match
  order. The policy kernel matches IP/CIDR sources and exact or `*.suffix`
  destinations without DNS or network activity.
- `delegate check DIRECTIVE...` validates and prints canonical JSON without
  opening sockets. `delegate DIRECTIVE...` and `delegate serve DIRECTIVE...`
  start the tested HTTP slice only after full parsing and validation. Other
  frontend protocols fail before listener startup. HTTP server and backend
  client timeouts are bounded.
- Listener cancellation stops new accepts and drains active requests for a
  bounded period. The command converts interrupt and termination signals into
  graceful cancellation. The local and remote portability matrices now also
  compile Windows/amd64 with `CGO_ENABLED=0`.
- Canonical TOML decodes server, mount, and policy values through a strict,
  reader-based parser. Unknown fields, syntax errors, and invalid canonical
  values fail before runtime resources are opened. The command accepts
  `--config PATH` and `--config=PATH`, but rejects mixing either with legacy
  directives. The pinned decoder is pure Go and has no transitive runtime
  dependencies.
- `delegate explain` runs the production mount resolver and policy kernel
  against a fully specified path, source, and method without opening a
  listener or contacting a backend. Its JSON output distinguishes permitted,
  rejected, no-mount, unsafe-path, and ambiguous-mount outcomes and records
  the rewritten target and winning policy rule when applicable. It works with
  canonical TOML or the verified legacy directive subset.
- Mounts may be scoped to a named server, frontend protocol, or both. Config
  validation rejects unknown servers and contradictory server/protocol pairs.
  Resolution filters by frontend identity, then uses path specificity,
  explicit priority, and scope specificity deterministically. Both the live
  HTTP frontend and `explain` supply that identity to the same resolver.
- Canonical configuration can run multiple named HTTP listeners. Duplicate
  listen addresses fail validation. Runtime startup pre-binds every socket and
  rolls back partial bind success; one listener failure cancels its peers, and
  parent cancellation gives every active listener the same bounded graceful
  drain window.
- Runtime configuration is published through an atomic immutable snapshot
  store. Invalid candidates leave the previous snapshot active, inputs and
  outputs cannot mutate stored slices, and concurrent replacements/readers
  pass the race detector. Each HTTP request reads exactly one snapshot for
  routing and authorization.
- Canonical-file runtimes reload routing and policy atomically on `SIGHUP` on
  Darwin and other Unix-family targets. Invalid files and listener-topology
  changes are reported and leave the prior snapshot active. The watcher stops
  with the server context; Windows keeps the same portable build with signal
  reload disabled.
- HTTP policy uses the direct socket peer as its source by default. A canonical
  listener can explicitly pair a client-address header with trusted-proxy
  CIDRs. Headers from untrusted peers are ignored; trusted chains are walked
  right to left, malformed chains return HTTP 400 before backend invocation,
  and IPv4-mapped peers match IPv4 CIDRs. These settings participate in strict
  TOML decoding, immutable configuration snapshots, and atomic live reload.
- Canonical TOML models frontend TLS termination independently from backend
  verification and optional client identity. Certificate/key references must
  be paired, minimum versions are restricted to TLS 1.2 or 1.3, backend TLS is
  legal only for HTTPS targets, and no insecure verification bypass exists.
  Validation reads no referenced files and `check` exposes the canonical
  model. TLS pointers are deep copied in immutable snapshots, and frontend TLS
  is part of listener topology.
- The HTTP runtime loads every configured frontend certificate/key before it
  binds any socket, then wraps only the matching listeners with standard-library
  TLS. Plaintext and TLS listeners coexist in one bounded shutdown group. The
  default minimum is TLS 1.2 and a configured TLS 1.3 minimum is enforced.
  Generated-certificate loopback tests prove both protocols, version rejection,
  zero binds on identity failure, and closure of earlier sockets on later bind
  failure.
- HTTPS mounts select preloaded backend transports after mount resolution and
  authorization without putting TLS details into the protocol-neutral Fetch.
  System roots are the default; custom CA files are appended, explicit server
  names and TLS minimums are applied, and optional client identities support
  mutual TLS. Generated loopback tests prove private-CA and client-certificate
  verification through the full gateway. Invalid backend material causes zero
  listener binds, unknown transport policies fail before dialing, idle pools
  close at shutdown, and changing the preloaded backend TLS-policy set requires
  a restart while routing among the existing set may reload.
- `scripts/release.sh VERSION OUTPUT_DIRECTORY` produces deterministic zip
  archives for the active Darwin/arm64 and Linux/amd64 matrix with CGO
  disabled, trimmed paths, no VCS metadata, an empty Go
  build ID, and a link-time version stamp. Archives contain the executable and
  README, preserve executable mode, and have a stable SHA-256 manifest. Two
  complete real builds produced identical manifests; all checksums verified,
  and the extracted Darwin binary reported its version and passed `check`
  against the canonical example. Archive construction also has isolated,
  network-free determinism tests.
- Mounts may use either the existing path source or an absolute HTTP/HTTPS URL
  source. Legacy `MOUNT="http://host/path/* target/*"` and strict TOML
  `source` decode to the same model. Resolution matches scheme and authority
  without DNS, folds host case, keeps explicit ports distinct, normalizes the
  request path, and uses URL authority as a deterministic tie-breaker. Source
  userinfo, query, fragment, percent escaping, traversal, duplicate separators,
  and unsupported schemes fail closed. Live absolute-form HTTP requests and
  `delegate explain --url` use this same resolver. A bounded fuzz run completed
  393,080 executions without a panic.
- HTTP request and response metadata is sanitized symmetrically. Standard
  hop-by-hop headers, the non-standard `Proxy-Connection`, proxy credentials
  and challenges, and every repeated/mixed-case `Connection` nominee are
  removed; end-to-end fields retain all values. A real absolute-form request
  sent through an `http.Client` proxy proves credentials and connection-scoped
  fields do not reach the backend or return to the caller.
- HTTP `CONNECT` uses explicit `connect://host:port/` mount sources and
  `tcp://host:port` targets. Authority and port syntax validate without DNS;
  mount resolution and an affirmative HTTP/CONNECT policy decision occur
  before dialing. The transparent relay is separate from Fetch, applies a
  bounded dial and handshake plus rolling idle deadlines, propagates TCP
  half-closes, and closes both hijacked connections. Loopback tests prove
  bidirectional traffic, client half-close, denial with zero dials, malformed
  authority rejection, and idle expiry. `explain` evaluates the same route
  without opening a socket. A bounded source-validation fuzz run completed
  557,516 executions and a dedicated CONNECT-authority run completed 187,611
  executions without a panic.
- Authorized HTTP `PUT` requests now become protocol-neutral Store operations
  rather than Fetch operations. They reuse the winning mount's backend TLS
  transport, preserve end-to-end metadata and known content length, and cap
  request bodies at 32 MiB. Denied and declared-oversized writes invoke no
  connector; loopback tests prove the complete HTTP Store path while existing
  Fetch and CONNECT coverage remains green.
- `ftp://` mounts now execute protocol-neutral Fetch and Store operations through
  a dedicated FTP connector and preserve loopback behavior for `RETR`, `STOR`, and
  `LIST` operations.
- A differential compatibility harness now executes fixture suites over
  canonical baseline JSON and supports an optional reference executable/path for
  actual behavior comparisons against original DeleGate output.
- The compatibility fixture corpus now includes mount/policy parity coverage with a
  MOUNT + PERMIT/REJECT legacy fixture.
- The compatibility fixture corpus now includes scoped legacy `MOUNT` option
  coverage for `server=`, `protocol=`, and `priority=` metadata.
- The compatibility fixture corpus now includes a legacy `CONNECT` mount fixture
  with protocol-scoped mapping and explicit legacy policy.
- The compatibility fixture corpus now includes default-port `SERVER=ftp` legacy
  coverage with ftp mount + mixed protocol policy rules.
- The compatibility fixture corpus now includes default-port `SERVER=gopher`
  coverage with a path mount and protocol-aware policy metadata.
- The compatibility fixture corpus now includes default-port `SERVER=https`
  coverage with an HTTPS mount and protocol-specific permit/reject parity.
- The compatibility fixture corpus now includes default-port `SERVER=socks`
  coverage with TCP target support and protocol-aware permit/reject metadata.
- The compatibility fixture corpus now includes protocol case normalization checks
  for `SERVER=HTTP` and uppercase `PERMIT` selectors.
- The compatibility fixture corpus now includes protocol-case normalization checks
  for `SERVER=FTP` and uppercase `PERMIT` selectors with FTP mounts.
- The compatibility fixture corpus now includes protocol-case normalization checks
  for `SERVER=HTTPS` and uppercase protocol selectors.
- The compatibility fixture corpus now includes protocol-case normalization checks
  for `SERVER=GOPHER` and uppercase protocol selectors.
- The compatibility fixture corpus now includes protocol-case normalization checks
  for `SERVER=SOCKS` and uppercase protocol selectors.
- The compatibility fixture corpus now includes protocol-case normalization checks
  for CONNECT metadata and `PERMIT`/`REJECT` selectors.
- The compatibility fixture corpus now includes protocol-option scope
  normalization for legacy `MOUNT` directives.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for HTTPS legacy `MOUNT` directives with `protocol=HTTPS`.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for FTP legacy `MOUNT` directives with `protocol=FTP`.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for Gopher legacy `MOUNT` directives with `protocol=GOPHER`.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for SOCKS legacy `MOUNT` directives with `protocol=SOCKS`.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for URL-source legacy `MOUNT` directives with mixed-case protocol scope.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for HTTPS `CONNECT` `MOUNT` metadata with mixed-case protocol.
- The compatibility fixture corpus now includes protocol-option case-normalization
  for HTTP `CONNECT` `MOUNT` metadata with mixed-case protocol.

## Unverified

- Legacy syntax outside the verified `SERVER`, `-P`, practical scoped `MOUNT`,
  `PERMIT`, and `REJECT` subset. The legacy adapter still describes one server
  per process invocation; multiple listeners use canonical configuration.

## Current gate

Active gate is now expanding the compatibility fixture set for additional legacy
directives while keeping comparison behavior wired through a reference-executable
path.

Active verification and release builds target only Darwin/arm64 and Linux/amd64
until the owner expands the matrix.
