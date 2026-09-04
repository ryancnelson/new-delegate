# Current state

Last verified: 2026-09-03

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
  failure. Custom backend TLS still fails before startup or reload publication.

## Unverified

- Legacy syntax outside the verified `SERVER`, `-P`, practical scoped `MOUNT`,
  `PERMIT`, and `REJECT` subset. The legacy adapter still describes one server
  per process invocation; multiple listeners use canonical configuration.

## Current gate

Build per-mount backend TLS transports without leaking transport policy into
protocol-neutral Fetch operations.
