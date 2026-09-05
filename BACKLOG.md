# new-delegate - Feature Backlog

**Purpose:** Capture and prioritize improvement ideas for new-delegate.

**Project:** new-delegate (Go)
**Description:** Modern DeleGate-compatible protocol gateway with fail-closed policy and protocol translation

**Last Updated:** 2026-09-04 (Iteration 104 - own and drain CONNECT sessions)

---

## How to Use This Backlog

- **Adding ideas:** Add new items under appropriate category with [IDEA] tag
- **Prioritizing:** Move items between priority sections based on value/effort
- **Implementing:** Change [IDEA] → [IN PROGRESS] when starting, then [DONE] when complete
- **Reviewing:** Re-prioritize during strategic reviews (every 8 iterations)

---

## Active Priority 1: Review remediation and geographic-link delivery

This queue supersedes the historical next-step guidance below. The owner agreed
with the architecture/code review and requested this handoff for GPT Sol. Do not
resume low-value casing fixtures or add protocol names while these items remain.
Historical DONE entries describe earlier acceptance tests, not proof that the
broader protocol or lifecycle is complete. New items below correct those gaps.

### Handoff baseline and working rules

- Work in `/Volumes/T9/ryan-homedir/devel/new-delegate`, not the Solaris project.
  Read `AGENTS.md`, `CURRENT-STATE.md`, `DESIGN.md`, `COMPATIBILITY.md`, and this
  active queue before implementation. Inspect current git state again.
- At review time HEAD was `1e076e0`; last numbered implementation was iter-83,
  `9e0050e`. `link/`, `docs/design-plans/`, and `docs/implementation-plans/` were
  untracked; `PROJECT_STATUS.md` was modified independently. Preserve all of
  these. Do not blindly stage the whole tree or revert work to get green tests.
- A fresh native Darwin/arm64 `go test -race ./... -count=1 -timeout=30s` passed
  every tested package except `link`, which reported concurrent bytes.Buffer
  access in `TestRunRightPrintsOnePairingAndRedactsTailcatOutput`. This is a
  reproduced failure. Other review findings are source-backed and need focused
  reproductions; do not claim their regression tests already exist.
- No real Tailcat two-host demonstration or current Woodpecker run was verified
  by this review. Untracked code cannot be covered by an existing remote green
  build. HTTP is the only runnable frontend; SOCKS5 has codecs, not a listener.
- Keep Go, canonical configuration, legacy syntax adapters, deterministic mounts,
  policy-before-backend, and separate semantic/relay paths. No rewrite.
- Only Darwin/arm64 and Linux/amd64 are active build targets. Keep normal builds
  CGO-free; no new assembly. Claude owns the site work; leave it alone.
- Execute one ready item at a time, in the order below. Larger items may become
  explicitly named sub-items with their own RED/GREEN checkpoints. Do not mark
  a parent DONE until every acceptance condition is demonstrated.
- For each item: mark IN PROGRESS; write the regression; run it and record the
  expected failure; implement; rerun focused tests; run `./scripts/check.sh`;
  update verified status; commit only intended files with the next available
  `[iter-N]`; push origin and the configured GitHub CI mirror; verify Biggie
  Woodpecker for that exact commit before proceeding. Inspect remote names and
  latest iteration rather than assuming either is unchanged. Record build URL,
  commit, and result. Report a concrete CI blocker instead of claiming success.
- Unit/integration tests must be offline and use loopback/ephemeral ports or
  fakes. A separately invoked real two-host smoke test may use the network.
  The owner explicitly permits short-lived pairing secrets in test process
  arguments; this narrow exception does not permit committing them or logging
  them unnecessarily. The GitHub mirror is now public.

### P1-031: Restore race-clean tests and make Woodpecker enforce them

- [DONE] Files: `link/tailcat_test.go`, `.woodpecker/test.yml`,
  `scripts/check.sh`, `scripts/ci-go.sh` as needed.
- Immediate operational priority: the owner reports Woodpecker has been idle
  for four hours. Investigate before further feature development. Read the
  actual latest pipeline timestamp, commit SHA, event, and status; compare local
  HEAD, Gitea origin, and GitHub mirror branch tips. Inspect which repository
  Woodpecker is connected to, enabled state, event/branch filters, webhook
  deliveries, pending queue, and Biggie agent availability. Distinguish no push,
  missing webhook, rejected event, queued job, and execution failure. Do not
  assume the runner is broken merely because no build was triggered.
- Trigger a manual build of the existing intended remote revision immediately
  if supported, or retry that exact revision through the established UI/API.
  This proves the execution path only; it does not validate uncommitted link
  code or prove push webhooks work. Do not create an empty commit just to hide
  the missing delivery path. After the real remediation commit is pushed,
  require its push-triggered build to demonstrate automatic delivery too.
- Report the concrete idle cause (or the specific check still blocked), build
  URL/number, SHA, and observed state promptly. Follow the build to completion;
  a trigger acknowledgment or queued status is not a successful build. If a
  missing login or runner access prevents this, ask for that exact prerequisite
  rather than spending another session adding code without remote validation.
- RED: run `go test -race ./link -run TestRunRightPrintsOnePairingAndRedactsTailcatOutput -count=1`.
  The producer writes bytes.Buffer while the test polls Len/String.
- Replace timing-based shared-buffer polling with synchronized handoff/observed
  writes; read diagnostics only after synchronization. Use `io.Discard` instead
  of the custom writer returning `(0, nil)`. Give every wait an explicit bound;
  ensure failure cleanup cancels and joins the child before the test exits.
- Acceptance: repeated race-enabled link tests and the full local gate pass.
  Woodpecker runs `go test -race ./...` with a suitable native toolchain, plus
  existing ordinary tests/vet/formatting and CGO-free target builds. Do not
  apply `CGO_ENABLED=0` to the race test. The exact pushed commit is green.
- Reconcile untracked link files deliberately: include their reviewed source
  when committing tests that depend on it. It remains unfinished and unwired;
  passing its tests is not Tailcat feature completion.
- Verified result: the synchronized link tests and full local gate passed on
  Darwin/arm64. Push-triggered Woodpecker pipeline 81 reproduced the missing
  CGO setting; pipeline 82 then proved the agent image lacked a C compiler;
  pipeline 83 confirmed the host compiler was outside the agent filesystem.
  The reproducible Alpine agent image now includes `build-base` (with its prior
  Dockerfile backed up on Biggie). Restarted pipeline 84 passed clone, ordinary
  tests, `go test -race ./...`, and CGO-free Darwin/arm64 and Linux/amd64 builds
  at exact commit `0344b48`.

### P1-032: Prevent HTTP redirect policy bypass

- [DONE] Depends on P1-031. Files: `connector/http.go`, `connector/routes.go`,
  `cmd/delegate/main.go`, `connector/routes_test.go`, `server/http_test.go`.
- Evidence: policy approves only the resolved target; `http.Client.Do` follows
  redirects using a client whose CheckRedirect is unset.
- RED: an allowed loopback backend returns a redirect to a second, denied
  backend. Assert the second backend sees zero requests and the caller receives
  the original redirect/status/body. Cover cross-host and same-host/different
  path redirects, HTTPS-to-HTTP downgrade, and a Store redirect.
- Default decision: proxy redirects unchanged using `http.ErrUseLastResponse`.
  Enforce this for default and per-mount TLS clients; do not accidentally retain
  a caller-supplied redirect callback that bypasses the gateway invariant.
  Following redirects is deferred unless every new destination is reauthorized.
- Acceptance: zero secondary dials, preserved response, and existing TLS and
  header-boundary tests remain green.
- Verified result: default and per-mount HTTP clients now stop at the first
  redirect and return that response unchanged. Regression tests cover
  cross-host Fetch and Store redirects, a same-host path redirect, and an
  HTTPS-to-HTTP downgrade; none reaches the secondary destination or invokes a
  caller-provided redirect callback. The full local gate passed, including the
  race suite and CGO-free Darwin/arm64 and Linux/amd64 builds. Woodpecker
  pipeline 86 passed all stages at exact commit `9639b33`.

### P1-033: Constrain FTP data-channel destinations

- [DONE] Depends on P1-032. Files: `connector/ftp.go`; create
  `connector/ftp_test.go`; extend `connector/routes_test.go`.
- Evidence: the PASV response is converted directly into a DialContext target.
- RED: a fake approved control server advertises a different host in PASV;
  assert no dial to that host. Cover malformed octets, zero/out-of-range ports,
  and IPv6 control connections.
- Prefer EPSV, using the actual control peer IP for data connections. If PASV
  fallback is supported, constrain it to that same peer, validate all fields,
  and document the fallback rule. Do not trust an arbitrary server-supplied
  hostname or perform a fresh unpinned hostname resolution for the data peer.
- Acceptance: normal retrieval/storage/listing still work; hostile passive
  replies cannot redirect the gateway to another host. Keep data-port handling
  an explicit part of the authorized FTP operation.
- Verified locally: the focused RED tests observed arbitrary PASV hosts and
  invalid ports reaching the dialer. The implementation now prefers EPSV,
  joins its validated port to the control peer, and falls back only for
  explicit capability responses. PASV validates every byte and the nonzero
  port but ignores the advertised host. IPv6 EPSV, hostile-host fallback,
  malformed-field, invalid-port, no-fallback, and RETR/STOR/LIST tests pass;
  the complete local gate is green. Push-triggered Woodpecker pipeline 88
  passed all stages at exact commit `f20c3c8`.

### P1-034: Bound FTP I/O and implement transfer lifecycle

- [DONE] Depends on P1-033. Files: `connector/ftp.go`, `connector/ftp_test.go`,
  `connector/routes_test.go`; semantic result changes only as needed.
- RED cases: stalled greeting, unterminated/oversized multiline reply, stalled
  data stream, client cancellation mid-transfer, and large RETR/LIST payload.
- Replace unbounded reply reading with a bounded FTP reply decoder including
  multiline handling. Add dial/control/data deadlines and cancellation-driven
  socket closure. Stream payloads with bounded memory; do not replace ReadAll
  with another whole-file buffer. A returned body must own data/control cleanup
  and transfer completion validation, including when the caller closes early.
- Acceptance: cancellation and timeout release all owned sockets/goroutines;
  first response bytes do not require the complete download; final FTP failures
  are surfaced. Document how a failure after HTTP headers are sent is handled
  (abort/report transfer failure, never pretend the already-sent status changed).
- Sub-checkpoints: (a) bounded multiline control replies plus dial/control
  deadlines and cancellation; (b) streaming RETR/LIST body ownership and STOR
  completion cleanup; (c) frontend propagation that aborts a partially written
  HTTP response when a late FTP completion fails. Keep this parent in progress
  until all three pass locally and in Woodpecker.
- Sub-checkpoint (a) is locally verified: stalled greetings time out, canceled
  contexts close the control socket, dialers receive a bounded context, and the
  aggregate-bounded decoder accepts valid multiline replies while rejecting
  oversized, unterminated, mismatched, and malformed records. Table, fuzz-seed,
  focused, race, and full portability gates pass. Push-triggered Woodpecker
  pipeline 90 passed all stages at exact commit `d6363e2`; sub-checkpoints (b)
  and (c) remain.
- Sub-checkpoint (b) is locally verified: RETR/LIST return a live body before
  data completion, stream a multi-megabyte staged payload without connector
  buffering, validate the final 2xx reply at EOF, surface late failure through
  Body.Read, and close both sockets on timeout, cancellation, or early Close.
  STOR rejects a failed completion reply. Focused, existing connector, race, and
  full portability gates pass. Push-triggered Woodpecker pipeline 92 passed all
  stages at exact commit `ad9b9e8`; sub-checkpoint (c) remains.
- Sub-checkpoint (c) is locally verified: a backend body that fails after
  streaming 32 KiB causes the HTTP handler to abort the response instead of
  returning a clean EOF or attempting to replace the committed status. The
  client observes a request/body-read failure, and the focused, server, race,
  and full Darwin/arm64 plus Linux/amd64 portability gates pass. Push-triggered
  Woodpecker pipeline 94 passed all stages at exact commit `4e483ac`. All three
  sub-checkpoints are accepted.

### P1-035: Make FTP-to-HTTP translation semantic and fail closed

- [DONE] Depends on P1-034. Files: `operation/operation.go`,
  `connector/ftp.go`, `server/http.go`, related connector/server tests.
- Evidence: FTP completion codes such as 226 are returned as HTTP status codes;
  unknown methods silently become RETR. Operation fields remain HTTP-shaped.
- RED: a real HTTP frontend against a fake FTP backend returns sensible HTTP
  success/not-found/permission/upstream-failure outcomes; unsupported methods
  execute no FTP command or backend dial. Test HEAD deliberately: either
  implement its semantics or reject it, rather than silently retrieving a file.
- Define a small semantic outcome/capability contract for Fetch, Store, List and
  their failures. Map FTP wire codes into it and let the HTTP frontend encode
  HTTP statuses. Preserve deliberate HTTP passthrough behavior without making
  every future connector manufacture HTTP codes. Avoid a speculative RPC layer.
- Acceptance: no raw FTP status crosses as an HTTP status; translation errors
  fail closed; existing HTTP behavior is regression-tested. Record the chosen
  mapping and any unsupported operations in COMPATIBILITY.md.
- Sub-checkpoint (a) is locally verified: `operation.Result` distinguishes
  native HTTP passthrough from semantic success, not-found, permission-denied,
  and upstream-failure outcomes. The HTTP frontend maps them to 200/204, 404,
  403, and 502, while invalid outcomes/statuses fail closed. FTP success no
  longer exposes 226, rejected reply classes map semantically, and GET/LIST or
  PUT method validation happens before dialing. Focused connector/server tests
  and the full race/portability gate pass. Push-triggered Woodpecker pipeline
  96 passed all stages at exact commit `4e9efe1`. The real HTTP-to-FTP outcome
  matrix remains before P1-035 acceptance.
- Sub-checkpoint (b) is locally verified: LIST is a distinct semantic operation
  through the frontend, route selector, and HTTP/FTP connectors rather than a
  Fetch method string. A real loopback HTTP-to-FTP test proves GET 200, LIST
  200, PUT 204, missing 404, permission 403, and upstream 502 behavior. HEAD,
  POST, and DELETE return 405 without opening an FTP control connection. The
  focused tests and full race plus Darwin/arm64 and Linux/amd64 gate pass.
  Push-triggered Woodpecker pipeline 98 passed all stages at exact commit
  `075c7a2`, completing P1-035.

### P1-036: Own and drain CONNECT and bridge sessions

- [IN PROGRESS] Depends on P1-035. Files: `server/lifecycle.go`, `server/group.go`,
  `server/http.go`, `link/bridge.go`, and lifecycle/relay tests.
- Evidence: HTTP Shutdown does not manage hijacked CONNECT sessions. The bridge
  registers connections only after dialing; Close snapshots allow later Add;
  the drain-timeout branch then waits without a bound.
- RED: cancel with an active CONNECT, a pending dial, a relay completing during
  drain, and a relay held past the deadline. Deterministically coordinate the
  registration-versus-close race with channels, not sleeps. Cover listener
  failure as well as parent cancellation.
- Use explicit session ownership: register accepted clients before dialing,
  reject/close late registrations after shutdown, cancel owned dials, stop
  accepting, allow a bounded drain, then force-close remaining sessions.
  Define the Dialer contract to honor cancellation; Go cannot forcibly kill an
  arbitrary non-cooperative callback. Do not promise bounds the API cannot meet.
- Preserve bidirectional half-close; terminate the opposite copy on fatal I/O
  errors, handle connections without CloseWrite, and retain idle deadlines.
- Acceptance: Serve returns within the documented deadline plus test tolerance,
  no tracked sessions survive, half-close works, race tests pass. Share lifecycle
  logic where useful without conflating HTTP parsing and transparent relay.
- Sub-checkpoint (a) is locally verified for the geographic bridge: accepted
  clients register before remote dialing, tracker closure permanently rejects
  and closes later registrations, and the drain deadline closes an accepted
  client even while a deliberately non-cooperative dial remains pending. Tests
  coordinate with channels rather than sleeps and pass under the full race plus
  Darwin/arm64 and Linux/amd64 gate. Push-triggered Woodpecker pipeline 100
  passed all stages at exact commit `deaa9a6`; CONNECT/server lifecycle work
  remains.
- Sub-checkpoint (b) is locally verified for HTTP CONNECT: the handler owns
  backend and hijacked client streams, begins a permanent drain on parent
  cancellation or listener failure, cancels pending context-aware dials,
  rejects late registrations, allows active relays to finish within the shared
  shutdown budget, and force-closes sessions held at its deadline. Existing
  bidirectional half-close behavior remains covered. Focused tests pass across
  repeated runs and the full race plus Darwin/arm64 and Linux/amd64 gate passes.
  Exact-commit Woodpecker acceptance remains before P1-036 completion.

### P1-037: Replace fragile Tailcat orchestration with a tested transport adapter

- [IDEA] Depends on P1-036. Files: `link/tailcat.go`, `link/tailcat_test.go`,
  `link/bridge.go`, `go.mod`, `go.sum` if embedding; existing Tailcat plans.
- Prefer a pinned Tailcat Go-library adapter behind the narrow connection
  interface and shared lifecycle. Inspect the actual chosen upstream version,
  toolchain requirements, license, and Darwin/arm64 + Linux/amd64 build results;
  record why the dependency is justified. Do not depend on moving latest.
  A CLI adapter is acceptable only if its lifecycle is equally tested.
- Required regressions: saved default key cannot defeat one-run pairing;
  malformed token including `tcp` is rejected; one newline completes handoff
  without waiting for EOF; oversized/trailing input has a defined fail-closed
  rule; cancellation during input/startup returns; output failure stops/reaps
  child resources; selected loopback host is preserved or explicitly rejected.
- Use upstream address parsing rather than scraping arbitrary `tc` words from
  human logs. With CLI serving explicitly request `--key=new`; drain output
  readers correctly, check scanner errors, prevent blocked sends/early returns,
  and define process cleanup. Do not run a second independent forwarding loop
  that silently bypasses the shared drain contract.
- Acceptance: offline adapter tests cover failures and cleanup, no secret is
  persisted by default, and one live pairing supports multiple TCP streams.
  Upstream reference: https://github.com/tailscale/tailcat#key-management and
  https://github.com/tailscale/tailcat#go-library . Recheck pinned-version facts.

### P1-038: Wire the geographic link into the executable and prove it on Biggie

- [IDEA] Depends on P1-037. Files: `cmd/delegate/main.go`, command/runtime tests,
  `examples/`, `scripts/` smoke-test entrypoint, README and compatibility docs.
- Right starts first and emits a one-run handoff; left consumes it and exposes
  a loopback endpoint. Keep the existing ordinary serve/check/explain behavior.
  Define and document exact flags before writing command tests. Unknown or
  invalid link arguments fail before creating processes, listeners, or backends.
- First complete slice: HTTP between the two gateways, with any backend protocol
  translation performed on the destination side. Carry approved traffic only;
  do not implicitly trust forwarded client identity just because it arrived
  over Tailcat. Document whether policy sees the left gateway or an explicitly
  authenticated end-client identity.
- Offline acceptance: executable-level tests cover startup, repeated clients,
  concurrent streams, backend failure, half-close, cancellation and cleanup.
- Real smoke acceptance: right on Linux/amd64 Biggie, left on Darwin/arm64;
  repeated and concurrent curl requests reuse one pairing; teardown closes
  listeners; restart creates a fresh pairing and the old one no longer works.
  Capture sanitized evidence and exact binary commit IDs. Run this network test
  separately from deterministic offline CI, with bounded waits and cleanup.
- Woodpecker must build/test the exact pushed checkpoint. Cross-compilation is
  not a two-host execution test, and a local smoke is not a Woodpecker result.
- Document a working one-machine command and its exact right-first, two-host
  equivalent, plus direct same-tailnet guidance that needs no Tailcat pairing.
  Do not claim general FTP-client support through a single forwarded control
  port: FTP data connections need separate handling. Do not claim arbitrary
  semantic splitting of every original DeleGate command in this first slice.

### P1-039: Make compatibility and progress claims evidence-based

- [IDEA] Depends on P1-038; correct misleading claims opportunistically with
  each earlier checkpoint. Files: `compatibility/harness.go`, related tests,
  `CURRENT-STATE.md`, `COMPATIBILITY.md`, `DESIGN.md`, this backlog.
- Current fixtures compare canonical config, and the reference executable path
  invokes `check` expecting this project's JSON schema. Label this parser/config
  regression coverage, not verified original DeleGate wire parity.
- Specify a real behavioral comparison: controlled original server and new
  server, same requests/backends, observed response and backend-side effects.
  If the original binary needs an adapter, specify it explicitly. Availability
  of a reference binary is a prerequisite, not permission to invent evidence.
- Acceptance: ledger distinguishes parsed configuration, codec-only support,
  runnable integration, actual two-host tests, and reference comparisons.
  Current-state docs summarize capabilities/limits; preserve historical records
  without promoting old acceptance tests into broader completion claims.
- Resume broader protocol work only after the above delivery checkpoint, using
  user-visible vertical slices rather than another series of casing fixtures.

---

## Historical Priority 1: Completed acceptance slices

These records are retained for history. Select new work from the active queue above.

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

### P1-029: Cancelable compatibility fixture suites

- [DONE] Stop a fixture suite before it parses fixtures or invokes a reference
  executable when its context has been canceled.
- Acceptance: an already-canceled suite returns the context error and makes no
  reference-runner call; cancellation is also checked between fixture phases.

### P1-030: SOCKS5 CONNECT wire framing

- [DONE] Decode the no-auth SOCKS5 greeting and CONNECT request, then
  encode method-selection and reply frames without opening a backend socket.
- Acceptance: table and malformed-input tests plus fuzz coverage prove bounded,
  fail-closed parsing for IPv4, IPv6, and domain authorities; unsupported
  methods and commands have deterministic SOCKS5 replies.

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
- [DONE] Differential compatibility harness for an original DeleGate executable.

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

### Iteration 29 (2026-09-04)
- [DONE] ✅ Added compatibility harness scaffolding for deterministic fixture loading

### Iteration 30 (2026-09-04)
- [DONE] ✅ Added a runnable compatibility fixture suite with optional reference-binary execution

### Iteration 31 (2026-09-04)
- [DONE] ✅ Expanded compatibility fixture coverage and verified multi-fixture suite behavior

### Iteration 32 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for scoped legacy `MOUNT` options (`server=`, `protocol=`, `priority=`)

### Iteration 33 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for legacy `CONNECT` mount mapping with fallback policy

### Iteration 34 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for default-port `SERVER=ftp` and ftp mount translation metadata

### Iteration 35 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for default-port `SERVER=https` and HTTPS mount translation metadata

### Iteration 36 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for default-port `SERVER=gopher` and path mount translation metadata

### Iteration 37 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for default-port `SERVER=socks` and TCP mount translation metadata

### Iteration 38 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for legacy protocol case normalization in `SERVER` and policy selectors

### Iteration 39 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase `SERVER=FTP` and uppercase policy selectors

### Iteration 40 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase `SERVER=HTTPS` and matching protocol selectors

### Iteration 41 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase `SERVER=GOPHER` and matching protocol selectors

### Iteration 42 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase `SERVER=SOCKS` and matching protocol selectors

### Iteration 43 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for protocol normalization in `CONNECT` mount metadata and selectors

### Iteration 44 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase `MOUNT` protocol scope values

### Iteration 45 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase HTTPS `MOUNT` protocol scope values

### Iteration 46 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase FTP `MOUNT` protocol scope values

### Iteration 47 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase Gopher `MOUNT` protocol scope values

### Iteration 48 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase SOCKS `MOUNT` protocol scope values

### Iteration 49 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for URL-source `MOUNT` protocol scope case normalization

### Iteration 50 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for HTTPS `CONNECT` `MOUNT` protocol scope case normalization

### Iteration 51 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for HTTP `CONNECT` `MOUNT` protocol scope case normalization

### Iteration 52 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for HTTPS `CONNECT` metadata and selector case-normalization

### Iteration 53 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=SOCKS` `CONNECT` metadata and selector case-normalization

### Iteration 54 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=FTP` `CONNECT` metadata and selector case-normalization

### Iteration 55 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=GOPHER` `CONNECT` metadata and selector normalization

### Iteration 56 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=GOPHER` `CONNECT` `MOUNT` protocol-scope normalization

### Iteration 57 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=FTP` `CONNECT` `MOUNT` protocol-scope normalization

### Iteration 58 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=SOCKS` `CONNECT` `MOUNT` protocol-scope normalization

### Iteration 59 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for `SERVER=HTTP` `CONNECT` `MOUNT` protocol and `server=` option case normalization

### Iteration 60 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case CONNECT `MOUNT`
  `protocol=` and explicit `server=` values across `SERVER=HTTPS`, `SERVER=GOPHER`, `SERVER=FTP`, and
  `SERVER=SOCKS`.

### Iteration 61 (2026-09-04)
- [DONE] ✅ Added mixed-case parsing support for legacy `MOUNT` option keys and
  compatibility fixture coverage for case-mixed `priority`, `server`, and `protocol`
  options.

### Iteration 62 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case `MOUNT` option
  keys in an HTTPS legacy mount flow with `PERMIT`/`REJECT` metadata.

### Iteration 63 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case `MOUNT` option
  keys across FTP, Gopher, and SOCKS legacy directives.

### Iteration 64 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case `MOUNT` option
  keys in CONNECT legacy mount directives across HTTP, FTP, Gopher, and SOCKS.

### Iteration 65 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case option key casing in legacy URL-source `MOUNT` directives.

### Iteration 67 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case option-key casing in HTTPS URL-source legacy `MOUNT` directives.

### Iteration 68 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for mixed-case option-key casing in FTP URL-source legacy `MOUNT` directives.

### Iteration 69 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase protocol scoping on HTTPS URL-source legacy `MOUNT` directives.

### Iteration 70 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase protocol scoping on Gopher URL-source legacy `MOUNT` directives.

### Iteration 71 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase protocol scoping on SOCKS URL-source legacy `MOUNT` directives.

### Iteration 72 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for uppercase protocol scoping on FTP URL-source legacy `MOUNT` directives.

### Iteration 73 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for option-key case normalization in Gopher URL-source legacy `MOUNT` directives.

### Iteration 74 (2026-09-04)
- [DONE] ✅ Added compatibility fixture coverage for option-key case normalization in SOCKS URL-source legacy `MOUNT` directives.

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

**Next step:** Start P1-034 in the active review-remediation queue: bound FTP
control/data I/O and make transfer completion own its full lifecycle. Continue
in dependency order toward the two-host geographic-link demonstration.
