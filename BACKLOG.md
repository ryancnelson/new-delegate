# new-delegate - Feature Backlog

**Purpose:** Capture and prioritize improvement ideas for new-delegate.

**Project:** new-delegate (Go)
**Description:** Modern DeleGate-compatible protocol gateway with fail-closed policy and protocol translation

**Last Updated:** 2026-09-04 (Iteration 28 - HTTP/FTP Fetch and List translation)

---

## How to Use This Backlog

- **Adding ideas:** Add new items under appropriate category with [IDEA] tag
- **Prioritizing:** Move items between priority sections based on value/effort
- **Implementing:** Change [IDEA] → [IN PROGRESS] when starting, then [DONE] when complete
- **Reviewing:** Re-prioritize during strategic reviews (every 8 iterations)

---

## Priority 1: High Impact, Small Scope (30-45 min)

These are ideal next iteration candidates. Each provides clear user value and fits within a single focused session.

### P1-001: Bootstrap and prove remote CI

- [DONE] Create the Go module, project contracts, and Woodpecker gate.
- Acceptance: `./scripts/check.sh` passes locally; a push to `main` produces a
  green Woodpecker pipeline on Biggie; Linux and Darwin/arm64 builds succeed
  with `CGO_ENABLED=0`.

### P1-002: Canonical server configuration

- [DONE] Define typed `Config` and `Server` values with validation.
- Acceptance: table tests prove valid HTTP server configuration is accepted;
  missing name, protocol, or listen address is rejected; duplicate names are
  rejected; inputs are not mutated.

### P1-003: Legacy `SERVER` syntax

- [DONE] Parse `SERVER=http` and `-P8080` into the canonical server model.
- Acceptance: golden tests cover ordering, whitespace, repeated directives,
  defaults, unknown directives, and malformed ports.

### P1-004: Canonical mount configuration

- [DONE] Define and validate `Mount` values.
- Acceptance: path and target are required; paths must be absolute; priorities
  are deterministic; invalid target schemes fail validation.

### P1-005: Legacy `MOUNT` syntax

- [DONE] Parse the practical original `MOUNT="/path/* target/*"` form.
- Acceptance: golden tests cover quoting, wildcard suffixes, options, malformed
  entries, and multiple mounts.

### P1-006: Safe path normalization

- [DONE] Normalize request paths before matching.
- Acceptance: traversal, encoded traversal, NUL bytes, and ambiguous escaping
  are rejected; normal paths have stable idempotent results; fuzz target never
  panics.

### P1-007: Deterministic mount resolution

- [DONE] Resolve by specificity and explicit priority. Server and protocol
  scoping remain for the multi-listener milestone.
- Acceptance: table tests cover exact, wildcard, fallback, no-match, and
  ambiguity; ambiguity fails closed.

### P1-008: Permit/reject policy kernel

- [DONE] Evaluate source, protocol, method, and mount constraints.
- Acceptance: default deny; explicit rejection wins at equal priority; no
  connector method can be invoked before an allow decision.

### P1-009: HTTP-to-HTTP vertical slice

- [DONE] Proxy one request through server, mount, policy, and connector.
- Acceptance: an in-process backend observes the rewritten request; response
  status, headers, and body return to the client; denial produces 403 and zero
  backend connections.

### P1-010: Legacy `PERMIT` / `REJECT` syntax

- [DONE] Parse `protocol:destination:source` selectors into canonical policy
  rules.
- Acceptance: golden and table tests cover quoting, whitespace, CIDR sources,
  destination suffix wildcards, ordered fallback rejection, and malformed
  selectors; matching performs no DNS or network I/O.

### P1-011: Runnable validation-first command

- [DONE] Validate legacy directives before starting the HTTP frontend and add a
  side-effect-free `check` mode.
- Acceptance: tests prove invalid configuration and unsupported frontends never
  invoke serving; valid directives reach the server; `check` emits canonical
  JSON without opening a listener; runtime clients and servers have timeouts.

### P1-012: Graceful listener lifecycle

- [DONE] Stop accepting new connections on cancellation while allowing active
  requests a bounded drain period.
- Acceptance: an integration test holds an in-flight request open, cancels the
  server, proves shutdown waits, releases the request, and verifies the
  listener no longer accepts connections; the command handles interrupt and
  termination signals.

### P1-013: Canonical TOML loading

- [DONE] Decode strict TOML into the same validated configuration used by the
  legacy compatibility adapter.
- Acceptance: a reader-based parser has no file or network side effects;
  server, mount, and policy values decode; unknown keys, malformed TOML, and
  invalid canonical values are fatal; `check` and `serve` accept `--config`
  without permitting mixed legacy arguments.

### P1-014: Effective-decision explanation

- [DONE] Explain mount resolution and policy evaluation without starting a
  listener or contacting a backend.
- Acceptance: canonical TOML and legacy configuration use the production
  resolver and policy kernel; JSON distinguishes permit, reject, no-mount,
  unsafe-path, and ambiguous-mount outcomes; a winning policy rule index and
  rewritten target are included when applicable; incomplete requests fail.

### P1-015: Frontend-scoped mounts

- [DONE] Scope canonical and legacy mounts to named servers and frontend
  protocols while retaining unscoped fallback mounts.
- Acceptance: configuration rejects missing named servers and inconsistent
  server/protocol scopes; path specificity and explicit priority remain
  deterministic; equally ranked scoped mounts beat generic fallbacks; the
  explanation and live HTTP paths pass the same frontend identity.

### P1-016: Coordinated HTTP listeners

- [DONE] Run every validated HTTP server under one cancellation and graceful
  drain lifecycle.
- Acceptance: duplicate listener addresses fail validation; all sockets are
  bound before any begins serving and partial bind failure closes earlier
  sockets; a listener failure cancels its peers; parent cancellation drains
  every listener; command validation passes multiple HTTP frontends to the
  runtime and still rejects unsupported protocols.

### P1-017: Atomic configuration snapshots

- [DONE] Publish validated configuration to concurrent request handlers as
  immutable, all-or-nothing snapshots.
- Acceptance: invalid replacement leaves the prior snapshot active; valid
  replacement cannot be changed through caller-owned slices; concurrent reads
  and replacements pass the race detector; each HTTP request uses one snapshot
  for both mount resolution and policy evaluation.

### P1-018: Signal-driven canonical reload

- [DONE] Reload a running canonical-file configuration on `SIGHUP` without
  interrupting listeners or in-flight requests.
- Acceptance: a channel-driven watcher test proves successful publication,
  rejected-file rollback, error reporting, and clean cancellation; Unix-family
  builds register `SIGHUP`; Windows compiles with reload signals disabled;
  listener topology changes remain restart-required.

### P1-019: Explicit trusted proxy boundary

- [DONE] Resolve forwarded client addresses only for explicitly trusted peers.
- Acceptance: direct peers remain authoritative by default; each listener
  couples its chosen header to validated trusted-proxy CIDRs; untrusted peers
  cannot spoof policy identity; trusted chains are walked from right to left;
  malformed chains return HTTP 400 without contacting a backend; immutable
  config snapshots deep-copy the CIDR list; trust settings reload without
  treating them as listener-topology changes.

### P1-020: Independent TLS configuration model

- [DONE] Model frontend TLS termination separately from backend TLS
  verification and client identity.
- Acceptance: strict TOML represents the two directions independently;
  frontend certificate/key and backend client-certificate/key references must
  be paired; minimum versions accept only TLS 1.2 or 1.3; backend verification
  has no insecure bypass; configured-but-not-yet-consumed TLS makes `serve`
  fail before opening listeners while side-effect-free `check` still works.

### P1-021: Frontend TLS runtime

- [DONE] Terminate HTTPS on configured HTTP frontends using the validated
  certificate/key references and minimum TLS version.
- Acceptance: all certificates are loaded before any listener begins serving;
  load or bind failure rolls back every listener; plaintext and TLS listeners
  can coexist; TLS 1.2/1.3 minimums are enforced; cancellation retains the
  bounded multi-listener drain behavior; loopback tests use generated fixtures.

### P1-022: Per-mount backend TLS runtime

- [DONE] Apply each HTTPS mount's validated backend trust and optional client
  identity to the selected Fetch without weakening default verification.
- Acceptance: all referenced CA and client identities load before listeners
  bind; system roots remain the default when no CA file is set; server-name and
  TLS minimum settings are enforced per selected mount; mutual TLS is covered
  with generated loopback fixtures; invalid material causes zero backend and
  frontend connections; transports have bounded idle pools and close cleanly.

### P1-023: Self-contained release artifacts

- [DONE] Produce checksummed single-binary archives for every active target.
- Acceptance: one deterministic script builds Darwin/arm64 and Linux/amd64
  with `CGO_ENABLED=0`; archives
  contain only the executable plus essential notices; SHA-256 checksums are
  emitted; the native artifact passes `delegate check` against the example;
  tests use a temporary output directory and never depend on public networks.

### P1-024: URL-authority MOUNT sources

- [DONE] Extend mount sources beyond path-only patterns to original-style
  absolute HTTP URLs while preserving path mounts as reverse-gateway defaults.
- Acceptance: canonical and legacy forms distinguish scheme, host, port, and
  normalized path without DNS; host comparison is case-insensitive and ports
  are explicit; userinfo, fragments, ambiguous escaping, and malformed
  authorities fail closed; resolution remains deterministic; `explain` covers
  both source forms. This becomes the routing basis for HTTP forward proxying.

### P1-025: HTTP hop-by-hop boundary

- [DONE] Strip proxy credentials and hop-by-hop headers on both sides of the
  HTTP Fetch boundary before expanding forward-proxy behavior.
- Acceptance: standard hop-by-hop fields and every header nominated by
  `Connection` are removed from backend requests and frontend responses;
  `Proxy-Authorization` and `Proxy-Authenticate` never cross the gateway;
  end-to-end headers remain intact; casing and repeated values are covered;
  an absolute-form proxy-client integration test proves no credential leakage.

### P1-026: Authorized HTTP CONNECT relay

- [DONE] Add a bounded byte-stream relay for authorized HTTP `CONNECT`
  requests without conflating it with semantic Fetch translation.
- Acceptance: authority syntax and ports validate before dialing; mount and
  policy approval precede all network activity; denial makes zero dials;
  successful loopback tunnels support bidirectional traffic and half-close;
  handshake/idle limits are bounded; hijacked connections are always closed;
  `explain` can report the CONNECT route without opening a socket.

### P1-027: Typed HTTP Store operation

- [DONE] Route authorized HTTP `PUT` through a protocol-neutral Store
  operation instead of representing writes as Fetch requests.
- Acceptance: routing and policy remain shared with Fetch; the selected mount
  rewrites the destination deterministically; metadata and bounded request
  bodies reach the HTTP connector; denial invokes no connector; existing Fetch
  and CONNECT behavior remains unchanged; loopback tests prove the full slice.

### P1-028: Typed FTP List operation

- [DONE] Route authorized ftp list fetches into the dedicated FTP connector.
- Acceptance: `LIST` over `ftp://` targets returns directory listing text and
  status through the semantic `Fetch` operation while preserving mount
  selection and policy behavior.

---

## Priority 2: High Impact, Medium Scope (60-90 min)

- [DONE] Independent frontend/backend TLS configuration model; runtime
  certificate loading, frontend termination, and custom backend transports are
  tracked separately.
- [DONE] Frontend TLS termination from validated file references.
- [DONE] Per-mount backend TLS verification and optional client identity.
- [DONE] Atomic configuration reload for canonical files, with rollback and an
  explicit restart requirement for listener-topology changes.
- [DONE] Trusted-proxy CIDRs for accepting forwarded client addresses; direct
  peer address remains the default and untrusted forwarding headers are
  ignored.
- [DONE] HTTP/FTP Fetch, Store, and List translation.
- Differential compatibility harness for an original DeleGate executable.

---

## Priority 3: Medium Impact (Nice to Have)

Features that improve the experience but aren't critical. Consider these after core functionality is polished.

---

## Priority 4: Low Priority / Future / Research Needed

Ideas that need more thought, research, or are lower value. Park here for consideration during strategic reviews.

---

## Completed (Archive)

Track completed items here to celebrate progress and inform future strategic reviews.

### Iteration 0 (2026-09-03)
- [DONE] ✅ Initialized project with iterate-bot methodology and proved Biggie CI

### Iteration 1 (2026-09-03)
- [DONE] ✅ Added immutable validation for canonical server configuration

### Iteration 2 (2026-09-03)
- [DONE] ✅ Added a side-effect-free `SERVER`/`-P` compatibility adapter

### Iteration 3 (2026-09-03)
- [DONE] ✅ Added canonical mounts and the practical original `MOUNT` form

### Iteration 4 (2026-09-03)
- [DONE] ✅ Added fail-closed path normalization and deterministic mount resolution

### Iteration 5 (2026-09-03)
- [DONE] ✅ Added typed default-deny permit/reject decisions and enforcement

### Iteration 6 (2026-09-03)
- [DONE] ✅ Proxied an authorized HTTP Fetch operation end to end

### Iteration 7 (2026-09-03)
- [DONE] ✅ Added original-style `PERMIT`/`REJECT` parsing and selector matching

### Iteration 8 (2026-09-03)
- [DONE] ✅ Made validated legacy directives runnable with a safe check mode

### Iteration 9 (2026-09-03)
- [DONE] ✅ Added bounded graceful shutdown and Windows compile verification

### Iteration 10 (2026-09-03)
- [DONE] ✅ Added strict canonical TOML parsing and command file loading

### Iteration 11 (2026-09-03)
- [DONE] ✅ Added side-effect-free effective routing and policy explanation

### Iteration 12 (2026-09-03)
- [DONE] ✅ Added validated named-server and protocol mount scoping

### Iteration 13 (2026-09-03)
- [DONE] ✅ Added coordinated, bounded multi-listener HTTP lifecycle

### Iteration 14 (2026-09-03)
- [DONE] ✅ Added race-tested atomic configuration snapshot publication

### Iteration 15 (2026-09-03)
- [DONE] ✅ Added atomic canonical-file reload with topology protection

### Iteration 16 (2026-09-03)
- [DONE] ✅ Wired portable signal-driven runtime reload

### Iteration 17 (2026-09-03)
- [DONE] ✅ Added explicit forwarded-client trust boundaries

### Iteration 18 (2026-09-03)
- [DONE] ✅ Modeled independent fail-closed TLS policy

### Iteration 19 (2026-09-03)
- [DONE] ✅ Terminated frontend TLS with atomic startup

### Iteration 20 (2026-09-03)
- [DONE] ✅ Routed per-mount backend TLS and mutual TLS

### Iteration 21 (2026-09-03)
- [DONE] ✅ Built reproducible checksummed release archives

### Iteration 22 (2026-09-03)
- [DONE] ✅ Added original-style URL-authority mount sources

### Iteration 23 (2026-09-03)
- [DONE] ✅ Enforced the HTTP hop-by-hop metadata boundary

### Iteration 24 (2026-09-03)
- [DONE] ✅ Narrowed active builds to Darwin/arm64 and Linux/amd64

### Iteration 25 (2026-09-03)
- [DONE] ✅ Added authorized, bounded HTTP CONNECT relaying

### Iteration 26 (2026-09-03)
- [DONE] ✅ Routed bounded HTTP PUT requests through typed Store operations

### Iteration 27 (2026-09-04)
- [DONE] ✅ Added FTP Fetch and Store connector translation

### Iteration 28 (2026-09-04)
- [DONE] ✅ Added FTP LIST translation on the dedicated ftp connector

---

## Ideas Inbox (Unsorted)

New ideas get added here during iterations. Sort into priority sections during strategic reviews.

- Unix-domain listener support where the host platform provides it.
- Release packaging that preserves the one-self-contained-artifact deployment
  experience across supported targets, with smoke tests against the packaged
  artifacts rather than only cross-compilation.

---

## Notes on This Project

**Initial State:**
- Fresh Go rewrite; the Python prototype is retained in its original repository.
- Test coverage begins with iteration P1-002.
- Gitea origin: `ssh://git@biggie:2222/ryan/new-delegate.git`

**Goals:**
- Preserve DeleGate's multi-protocol gateway model and configuration concepts.
- Make routing and authorization deterministic and fail closed.
- Build every behavior through small TDD iterations validated by Woodpecker.

---

**Next step:** Select the next compatibility slice after the HTTP Fetch, Store,
and CONNECT foundation.
