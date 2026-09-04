# new-delegate — Project Status

Running changelog, updated automatically every 10 minutes.

## 2026-09-04 06:32

- This pass adds `iter-30`, making the compatibility harness runnable as a
  fixture suite with optional external reference execution.
- `./scripts/check.sh` remains green (`gofmt`, `vet`, tests, race, and
  Darwin/arm64 + Linux/amd64 builds) with local verification.
- Current gate in `CURRENT-STATE.md` now: expand compatibility fixture coverage
  while using the reference-executable-capable suite path.
- Working tree is clean and in sync with `origin`/`ci`; scope remains
  Darwin/arm64 and Linux/amd64.

## 2026-09-04 05:53

- New commit `c99b4ca [iter-28] add ftp list translation` adds protocol-neutral
  ftp `LIST` command dispatch through the dedicated `FTP` connector.
- Focused loopback and full-repo verification pass for this slice:
  `gofmt`, `go vet`, `go test`, `go test -race`, and CGO-free `darwin/arm64`
  plus `linux/amd64` builds.
- Biggie Woodpecker repo 5 badge endpoint still returns `success` after the push.
- Working tree is clean and in sync with `origin` and `ci` on `c99b4ca`.
- Current gate in `CURRENT-STATE.md` remains the differential compatibility harness
  against the original DeleGate implementation.

## 2026-09-03 20:48

- Iteration 17 implements the explicit trusted-proxy boundary: the direct peer
  remains authoritative unless the current listener names a client-address
  header and trusted CIDRs. Trusted chains are evaluated right to left;
  untrusted headers are ignored and malformed trusted chains return HTTP 400
  without invoking a backend.
- The complete local gate passes, including race detection and the CGO-free
  portability matrix. Coverage includes spoof rejection, malformed-chain
  rejection before backend invocation, immutable CIDR snapshots, strict TOML
  decoding, IPv4-mapped peer handling, and live trust-policy reload without a
  listener restart. Woodpecker pipeline #18 accepted commit `038da90`.
- Next gate: model independent frontend and backend TLS settings with strict,
  side-effect-free validation.

## 2026-09-03 21:00

- Iteration 18 now has a strict, side-effect-free TLS policy model. Listener
  identity is separate from backend CA/name/client identity; paired references
  and TLS 1.2/1.3 minimums are validated, and no insecure backend verification
  option exists.
- `delegate check` accepts and prints the model, while startup and live reload
  reject TLS configuration before opening listeners or publishing snapshots
  because the runtime adapters are not implemented yet. This prevents a
  plaintext runtime from silently accepting an encryption declaration.
- The full local gate passes with race detection and the CGO-free portability
  matrix. Woodpecker pipeline #19 accepted commit `2431284`. Next gate: frontend TLS termination with
  preloaded certificates and coordinated listener rollback.

## 2026-09-03 21:08

- Iteration 19 implements frontend TLS termination using only Go's standard
  library. Every identity loads before the first bind; configured listeners are
  wrapped individually, so plaintext and TLS coexist under the existing
  coordinated graceful-shutdown lifecycle.
- Generated loopback fixtures prove successful TLS 1.3 service, rejection of a
  TLS 1.2-only client, the secure TLS 1.2 default, zero binds on certificate
  failure, and closure of earlier sockets when a later bind fails.
- The complete local race/static/CGO-free portability gate passes. Woodpecker
  pipeline #20 accepted commit `37ef906`. Next gate: per-mount backend TLS transports with custom roots and
  optional mutual-TLS identity.

## 2026-09-03 21:22

- Iteration 20 implements per-mount backend TLS without adding transport fields
  to protocol-neutral Fetch operations. The handler passes the resolver-selected
  mount to a routed connector, which chooses only among transports preloaded at
  startup.
- System roots remain the default; configured CA files append trust, explicit
  verification names and TLS minimums are applied, and optional client identity
  enables mutual TLS. A generated loopback backend verifies the client
  certificate through the full gateway path.
- Invalid backend material causes zero binds, unknown policies cause zero
  backend calls, idle pools close at shutdown, and changes to the preloaded TLS
  policy set require restart. The full local gate passes; remote CI is pending.
  Woodpecker pipeline #21 accepted commit `8be449e`. Next gate: checksummed
  self-contained release artifacts.

## 2026-09-03 21:31

- Iteration 21 adds a deterministic Go standard-library release packager behind
  `scripts/release.sh`, plus a link-time `delegate version` command.
- Five CGO-free zip archives cover Darwin/arm64, Linux/amd64, Linux/arm64,
  illumos/amd64, and Windows/amd64. Fixed metadata/order, trimmed paths, disabled
  VCS embedding, and an empty build ID make output reproducible. A SHA-256
  manifest covers every archive.
- Two complete real builds produced identical manifests; all five checksums
  verified. The extracted Darwin/arm64 executable reported `v0.0.0-smoke` and
  successfully checked `examples/delegate.toml`. The full local gate passes;
  Woodpecker pipeline #22 accepted commit `ebb1046`. Next gate: URL-authority
  `MOUNT` sources.

## 2026-09-03 21:39

- Iteration 22 extends `MOUNT` with absolute HTTP/HTTPS source URLs in both
  strict TOML and original-style legacy directives while preserving path mounts.
- Resolution compares parsed scheme and authority without DNS, folds host case,
  distinguishes explicit ports, and fails closed on userinfo, query, fragment,
  escaping, traversal, malformed authority, and non-canonical paths. Live
  absolute-form requests and `delegate explain --url` share the resolver.
- All tests, race detection, static checks, and portability builds pass. A
  bounded URL-source fuzz run completed 393,080 executions without a panic.
  Manual Woodpecker pipeline #23 accepted exact commit `9b53249` after the push
  webhook was missed. Next gate: strip hop-by-hop headers and proxy credentials.

## 2026-09-03 21:45

- Iteration 23 enforces a symmetric HTTP metadata boundary. Requests and
  responses remove standard hop-by-hop fields, `Proxy-Connection`, proxy
  credentials/challenges, and every repeated case-insensitive header nominated
  by `Connection`; end-to-end values remain intact.
- A real `http.Client` proxy integration test proves an absolute-form request's
  `Proxy-Authorization` and connection-scoped fields never reach the backend,
  and backend hop-by-hop fields never return to the caller.
- The full local test/race/static/CGO-free portability gate passes. Remote CI is
  pending. Next gate: authorized bounded HTTP `CONNECT` relay.

## 2026-09-03 18:46

- Initial snapshot.
- Repo: `~/devel/new-delegate/` — Go rewrite of DeleGate, iterate-bot methodology.
- Latest commit: `b99ead5 [iter-0] bootstrap Go parity-kernel project`. Working tree clean, in sync with `origin/main`.
- **P1-001** (bootstrap + prove remote CI) is `[IN PROGRESS]` — Woodpecker pipeline green status still unverified.
- P1-002 through P1-009 (config model, legacy SERVER/MOUNT parsing, path normalization, mount resolution, policy kernel, HTTP vertical slice) are `[READY]`, not started.

## 2026-09-03 19:20

- Iterations 1–6 are locally green and pushed through iteration 5.
- Canonical and legacy `SERVER`/`MOUNT` configuration, safe path resolution,
  fail-closed policy, and an authorized HTTP Fetch slice are implemented.
- Gitea remains the canonical origin. Private GitHub mirror
  `ryancnelson/new-delegate` is enabled as Woodpecker repository 5.
- Woodpecker webhook delivery is verified. A checksum-pinned Go bootstrap fix
  is ready to ship with iteration 6 after the minimal local-backend agent was
  found not to include a Go toolchain.

## 2026-09-03 19:22

- Biggie Woodpecker pipeline 4 passed in 31 seconds at `fa799d6`.
- The bootstrap, repository activation, webhook delivery, tests, and portable
  builds are now verified remotely.
- The first compatibility-kernel batch through the HTTP Fetch acceptance slice
  is accepted. Next is canonical and legacy `PERMIT`/`REJECT` configuration.

## 2026-09-03 19:44

- Iteration 7 adds original-style `PERMIT` and `REJECT` parsing with golden and
  table tests.
- Policy evaluation now supports source CIDRs and destination suffix patterns
  without runtime name resolution.
- The complete local gate, including race tests and portable builds, passes.

## 2026-09-03 19:47

- Iteration 8 makes validated legacy directives runnable through the HTTP
  frontend and adds a side-effect-free canonical JSON `check` mode.
- Tests prove parse, validation, and protocol errors cannot reach listener
  startup. Runtime server and backend-client timeouts are bounded.
- The complete local gate passes.

## 2026-09-03 19:54

- Iteration 9 adds context-driven, bounded graceful shutdown with an in-flight
  request integration test.
- The command handles interrupt and termination signals through the tested
  lifecycle primitive.
- Windows/amd64 joins the CGO-free local and Woodpecker compile matrix.

## 2026-09-03 18:54

- Two more iterations landed: `2faa643 [iter-1] add canonical server validation`, `b5bef4f [iter-2] parse legacy SERVER directives`.
- **P1-002** (canonical server config) and **P1-003** (legacy `SERVER`/`-P` syntax) are now `[DONE]`. Known protocols get conventional default ports; conflicting repetitions, malformed ports, and unknown directives are fatal; golden/table tests pass.
- **P1-001** (bootstrap + prove remote CI) remains `[IN PROGRESS]` — Woodpecker pipeline still unverified per CURRENT-STATE.md; current gate is connecting Biggie's Gitea forge to Woodpecker.
- Working tree has uncommitted changes in progress: modified `config/legacy_test.go`, untracked `mount/` directory (work toward P1-004, still `[READY]`) and `PROJECT_STATUS.md` itself.
- Remaining `[READY]` items unchanged: P1-004 (mount config), P1-005 (legacy MOUNT syntax), P1-006 (path normalization), P1-007 (mount resolution), P1-008 (policy kernel), P1-009 (HTTP vertical slice).

## 2026-09-03 19:04

- Three more iterations landed: `5ba4f7e [iter-3] add canonical and legacy mounts`, `c0ccf3d [iter-4] resolve mounts safely and deterministically`, `85c4bed [iter-5] add fail-closed policy kernel`.
- **P1-004** through **P1-008** are all now `[DONE]`: canonical/legacy mount config, safe path normalization (334,664 fuzz executions with no panic), deterministic mount resolution, and the fail-closed permit/reject policy kernel with stable reason codes and enforcement tests proving denied callbacks never execute.
- Only **P1-009** (HTTP-to-HTTP vertical slice) remains `[READY]` on the Priority 1 list — everything else in the compatibility kernel is done.
- **P1-001** (bootstrap + prove remote CI) is still `[IN PROGRESS]` — Woodpecker pipeline still unverified; current gate is unchanged (connect Biggie's Gitea forge to Woodpecker).
- Untracked in working tree: `server/` (new, likely toward P1-009) and `PROJECT_STATUS.md`. No modified-tracked files this round.
- CURRENT-STATE.md now also flags legacy `PERMIT`/`REJECT` parsing and mount scoping by named server/protocol as deferred, not yet covered by the Priority 1 list.

## 2026-09-03 19:14

- Two new CI commits: `847cbe7 [ci] enable manual Woodpecker runs`, `d2a486a [ci] run Go checks on Biggie local backend` — direct work toward closing P1-001's outstanding gate (Woodpecker activation on Biggie).
- BACKLOG.md and CURRENT-STATE.md content otherwise unchanged from the last entry: P1-002–P1-008 `[DONE]`, P1-009 (HTTP vertical slice) still the only `[READY]` Priority 1 item, P1-001 still `[IN PROGRESS]`.
- Working tree unchanged: `server/` and `PROJECT_STATUS.md` still untracked, nothing modified.
- Woodpecker pipeline result itself remains unverified as of this snapshot — these commits set up the manual-run capability and a local backend for Go checks, but CURRENT-STATE.md hasn't yet recorded a green remote build.

## 2026-09-03 19:24

- Three more commits: `79c396c [iter-6] proxy authorized HTTP fetches`, `fa799d6 [ci] keep tool bootstrap output deterministic`, `f290c05 [iter-6] accept green Biggie CI checkpoint`. Working tree is now clean and in sync with `origin/main`.
- **P1-009** (HTTP-to-HTTP vertical slice) is `[DONE]` — an HTTP frontend maps requests to a protocol-neutral Fetch operation, authorizes it, invokes the HTTP connector, and propagates status/headers/body. Acceptance test verifies path/query rewriting; a denial test verifies HTTP 403 with zero backend calls.
- **P1-001** (bootstrap + prove remote CI) is now also `[DONE]` — Biggie Woodpecker pipeline 4 passed in 31s at `fa799d6`, validating formatting, vet, all tests, and CGO-free builds for Darwin/arm64, Linux/amd64, Linux/arm64, and illumos/amd64.
- **All Priority 1 items (P1-001 through P1-009) are now `[DONE]`.** The entire compatibility-kernel batch through the HTTP Fetch acceptance slice is accepted.
- New current gate per CURRENT-STATE.md: add canonical policy configuration and the original `PERMIT`/`REJECT` syntax, then make the tested HTTP slice runnable from validated configuration. This isn't yet reflected as new Priority 1 backlog items.
- CI note: Biggie Woodpecker repository 5 tracks a private GitHub mirror (`ryancnelson/new-delegate`) for CI while canonical `origin` stays Gitea (`ryan/new-delegate`); the pipeline bootstraps a checksum-pinned Go 1.25.6 toolchain in `/tmp` since the agent image has no Go preinstalled.

## 2026-09-03 19:34

- No new commits since the last entry (`f290c05` still HEAD, in sync with `origin/main`). Only local change is this file itself.
- BACKLOG.md and CURRENT-STATE.md unchanged: all of P1-001–P1-009 still `[DONE]`; current gate is still adding canonical policy configuration plus the original `PERMIT`/`REJECT` syntax, then wiring the tested HTTP slice up to validated configuration.
- Quiet tick — work appears paused since the last iteration landed.

## 2026-09-03 19:44

- New commit: `30d5cf5 [iter-7] parse legacy policy directives`. Working tree clean, in sync with `origin/main`.
- A new backlog item (previously undocumented in earlier snapshots) covering original `PERMIT="protocol:destination:source"`/`REJECT` parsing is `[DONE]`: rules parse with explicit priorities preserving first-match order, and the policy kernel matches IP/CIDR sources and exact or `*.suffix` destinations without DNS or network activity. This matches the iteration-7 work already logged at 19:48 elsewhere in this file (golden/table tests, source CIDRs, destination suffix patterns, race tests, portable builds all passing).
- New current gate per CURRENT-STATE.md: make the tested HTTP slice runnable from validated legacy configuration (i.e. wire a config loader end-to-end) — mount scoping by named server/protocol remains deferred.
- All previously tracked Priority 1 items (P1-001–P1-009) remain `[DONE]`; no regressions.

## 2026-09-03 19:54

- New commit: `2a87d19 [iter-8] run validated legacy configuration`. Matches the iteration-8 work already logged at 19:47 elsewhere in this file: `delegate check DIRECTIVE...` validates and prints canonical JSON without opening sockets; `delegate DIRECTIVE...`/`delegate serve DIRECTIVE...` start the tested HTTP slice only after full parsing/validation; other frontend protocols fail before listener startup; HTTP server and backend-client timeouts are bounded.
- Working tree now has active in-progress work toward iteration 9 (graceful shutdown, also already logged at 19:54 elsewhere in this file): modified `.woodpecker/test.yml`, `BACKLOG.md`, `CURRENT-STATE.md`, `DESIGN.md`, `cmd/delegate/main.go`, `scripts/check.sh`, plus untracked `server/lifecycle.go` and `server/lifecycle_test.go`. Not yet committed as of this snapshot.
- BACKLOG.md now lists a `[DONE]` item at line 94 for the check-mode/runnable-config work; new current gate per CURRENT-STATE.md is adding canonical TOML config loading while keeping the runtime dependency footprint small and portable.
- Windows/amd64 has joined the local and remote CGO-free portability compile matrix.

## 2026-09-03 20:04

- Two new commits: `95c8547 [iter-9] drain listeners gracefully`, `c34092b [status] record iteration 9 acceptance`. This closes out the graceful-shutdown work that was previously only in the uncommitted working tree.
- New work already staged but uncommitted in the working tree: strict canonical **TOML config loading** (`config/toml.go`, `config/toml_test.go`, new `go.sum`, an `examples/` directory) plus modifications across `config/config.go`, `mount/mount.go`, `policy/policy.go`, `cmd/delegate/main.go`, `go.mod`, `README.md`, `COMPATIBILITY.md`, `BACKLOG.md`, `CURRENT-STATE.md`. BACKLOG.md line 111 already reflects this as `[DONE]`: canonical TOML decodes server/mount/policy values through a strict reader-based parser; unknown fields, syntax errors, and invalid values fail before runtime resources open; `--config PATH`/`--config=PATH` is supported but rejects mixing with legacy directives; the pinned decoder is pure Go with no transitive runtime dependencies.
- New current gate per CURRENT-STATE.md: add a side-effect-free explanation command for effective mount and policy decisions (this matches the Priority 2 backlog idea "config check and effective-policy explanation command").
- Remaining unverified: legacy syntax outside `SERVER`/`-P`/practical `MOUNT`/`PERMIT`/`REJECT`, and mount scoping by named server/protocol.

## 2026-09-03 20:14

- Two new commits close out and accept the TOML work: `364497a [iter-10] load strict canonical TOML`, `50a99c8 [status] accept iteration 10 in CI`.
- **New in the working tree (uncommitted):** the `delegate explain` command (`explain/` directory, untracked) — BACKLOG.md line 120/189 marks it `[DONE]`: runs the production mount resolver and policy kernel against a fully specified path/source/method without opening a listener or contacting a backend; JSON output distinguishes permitted, rejected, no-mount, unsafe-path, and ambiguous-mount outcomes, records the rewritten target and winning rule, and works with canonical TOML or the verified legacy directive subset. Modified-but-uncommitted: `BACKLOG.md`, `COMPATIBILITY.md`, `CURRENT-STATE.md`, `README.md`, `cmd/delegate/main.go`, `cmd/delegate/main_test.go`.
- New current gate per CURRENT-STATE.md: scope mounts to named servers and frontend protocols for the multi-listener model.
- Remaining unverified unchanged: legacy syntax outside the verified `SERVER`/`-P`/practical `MOUNT`/`PERMIT`/`REJECT` subset, and mount scoping by named server/protocol (this is now the active gate rather than just a deferred item).

## 2026-09-03 20:24

- Two new commits: `795f81f [iter-11] explain effective gateway decisions`, `1835d58 [iter-12] scope mounts to frontends`. Both close out work previously seen only uncommitted/in-progress.
- **`delegate explain`** is committed and `[DONE]` (BACKLOG.md line 120/199): runs the production mount resolver and policy kernel against a fully specified path/source/method without opening a listener; JSON output distinguishes permitted/rejected/no-mount/unsafe-path/ambiguous-mount outcomes.
- **Mount scoping by named server/protocol** is committed and `[DONE]` (BACKLOG.md line 129/202): mounts may be scoped to a named server, frontend protocol, or both; validation rejects unknown servers and contradictory server/protocol pairs; resolution filters by frontend identity then path specificity, priority, and scope specificity — both the live HTTP frontend and `explain` use the same resolver.
- Working tree has new in-progress work, currently uncommitted: modified `cmd/delegate/main_test.go`, `config/config_test.go`, plus untracked `server/group_test.go` — suggests the next iteration is testing toward the new current gate.
- New current gate per CURRENT-STATE.md: run multiple validated HTTP listeners under one coordinated lifecycle (multi-listener runtime). Remaining unverified: legacy syntax outside the verified `SERVER`/`-P`/practical scoped `MOUNT`/`PERMIT`/`REJECT` subset, and the coordinated multi-listener runtime itself.

## 2026-09-03 20:34

- Two new commits: `53793ef [iter-13] coordinate HTTP listeners`, `c698201 [iter-14] publish atomic config snapshots`. Working tree is clean, in sync with `origin/main`.
- **Multi-listener HTTP coordination** is `[DONE]` (BACKLOG.md line 138/224) — canonical configuration can run multiple named HTTP listeners; duplicate listen addresses fail validation; startup pre-binds every socket and rolls back on partial bind failure; one listener's failure cancels its peers; parent cancellation gives every active listener the same bounded graceful drain window. This closes the gate from the last entry.
- **Atomic config snapshot publication** is `[DONE]` (BACKLOG.md line 148/227) — runtime config is published through an atomic immutable snapshot store; invalid candidates leave the previous snapshot active; inputs/outputs can't mutate stored slices; concurrent replace/read passes the race detector; each HTTP request reads exactly one snapshot for routing and authorization.
- New current gate per CURRENT-STATE.md: reload canonical files into the atomic store while requiring a restart for listener-topology changes (i.e. live config reload, short of changing which ports are bound).
- Remaining unverified unchanged: legacy syntax outside the verified `SERVER`/`-P`/practical scoped `MOUNT`/`PERMIT`/`REJECT` subset. Noted addition: the legacy adapter still describes one server per process invocation — multiple listeners require canonical (TOML) configuration.

## 2026-09-03 20:44

- Two new commits: `f56724b [iter-15] reload canonical config atomically`, `d31c45c [iter-16] reload runtime config on signal`. This closes the "reload canonical files into the atomic store" gate from the last entry.
- **Atomic canonical-file reload** is `[DONE]` (BACKLOG.md line 157/240) and **signal-driven runtime reload** is `[DONE]` (line 169/243): canonical-file runtimes reload routing and policy atomically on `SIGHUP` on Darwin and other Unix-family targets; invalid files and listener-topology changes are reported and leave the prior snapshot active; the watcher stops with the server context; Windows keeps the same portable build with signal reload disabled.
- New current gate per CURRENT-STATE.md: add explicit trusted-proxy CIDRs before honoring forwarded client addresses. Matches new untracked `clientaddr/` directory in the working tree — work on this has already started.
- Working tree has in-progress, uncommitted changes: modified `config/config.go`, `config/config_test.go`, `config/store.go`, plus untracked `clientaddr/`.
- Remaining unverified unchanged: legacy syntax outside the verified subset; the legacy adapter still describes one server per process invocation.

## 2026-09-03 20:54

- New commit: `038da90 [iter-17] trust forwarded clients explicitly`. Working tree is otherwise clean apart from `BACKLOG.md` (modified) and untracked `tlsconfig/`.
- **Explicit forwarded-client trust boundary** is `[DONE]` (BACKLOG.md line 166/191/267) — matches the iteration-17 summary already logged out-of-band at 20:48 in this file: HTTP policy uses the direct socket peer by default; a canonical listener can pair a client-address header with trusted-proxy CIDRs; untrusted headers are ignored; trusted chains walk right to left; malformed trusted chains return HTTP 400 before backend invocation; IPv4-mapped peers match IPv4 CIDRs. Participates in strict TOML decoding, immutable snapshots, and atomic live reload.
- **New `[IN PROGRESS]` item** (BACKLOG.md line 176): model frontend TLS termination separately from backend TLS — this is the new current gate per CURRENT-STATE.md ("define independent frontend and backend TLS configuration with strict validation before adding certificate loading or network side effects"). Matches the untracked `tlsconfig/` directory already appearing in the working tree.
- Remaining unverified unchanged: legacy syntax outside the verified `SERVER`/`-P`/practical scoped `MOUNT`/`PERMIT`/`REJECT` subset; legacy adapter still one server per process invocation.

## 2026-09-03 21:04

- New commit: `2431284 [iter-18] model independent TLS policy`. Matches the iteration-18 summary already logged out-of-band at 21:00 in this file: canonical TOML models frontend TLS termination independently from backend verification/client identity; cert/key references must be paired; minimum versions restricted to TLS 1.2/1.3; backend TLS only legal for HTTPS targets; no insecure-verification bypass exists; validation reads no referenced files; `check` exposes the model while startup/reload still reject configured TLS until runtime adapters land; TLS pointers are deep-copied in immutable snapshots and are part of listener topology.
- New `[IN PROGRESS]` item (BACKLOG.md line 186): implement actual frontend TLS termination from the validated model, preloading every certificate before any listener begins serving — this is the current gate per CURRENT-STATE.md. Working tree has this in progress, uncommitted: modified `tlsconfig/tlsconfig.go`, `tlsconfig/tlsconfig_test.go`, `BACKLOG.md`.
- Remaining unverified unchanged.

## 2026-09-03 21:14

- New commit: `37ef906 [iter-19] terminate frontend TLS atomically`. Matches the iteration-19 summary already logged out-of-band at 21:08 in this file: frontend TLS termination uses only the standard library; every identity loads before the first bind; plaintext and TLS listeners coexist under the existing coordinated shutdown lifecycle; loopback fixtures prove TLS 1.3 service, TLS-1.2-only client rejection, the secure TLS 1.2 default, zero binds on cert failure, and closure of earlier sockets when a later bind fails.
- New `[IN PROGRESS]` item (BACKLOG.md line 195): apply each HTTPS mount's validated backend trust and optional client identity — this is the current gate per CURRENT-STATE.md ("build per-mount backend TLS transports without leaking transport policy into protocol-neutral Fetch operations"). Work has already started uncommitted: modified `tlsruntime/frontend.go`, `BACKLOG.md`; untracked `tlsruntime/backend.go`, `tlsruntime/backend_test.go`, `connector/routes.go`, `connector/routes_test.go`.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 21:24

- New commit: `8be449e [iter-20] route backend TLS per mount`. Working tree clean, in sync with `origin/main`.
- Matches the iteration-20 summary already logged out-of-band at 21:22 in this file: HTTPS mounts select preloaded backend transports after mount resolution/authorization without leaking TLS details into the protocol-neutral Fetch; system roots default, custom CA files append trust, explicit server names/TLS minimums apply, optional client identities support mutual TLS; loopback tests prove private-CA and client-cert verification through the full gateway; invalid backend material blocks all listener binds; unknown transport policies fail before dialing; idle pools close at shutdown; changing the preloaded backend TLS-policy set requires a restart (routing among the existing set may reload).
- **Full TLS story is now `[DONE]`**: frontend termination (iter-19) and per-mount backend TLS/mTLS (iter-20) both closed, joining the earlier independent TLS policy model (iter-18).
- New `[READY]` item (BACKLOG.md line 205): produce checksummed single-binary archives for every supported target. New current gate per CURRENT-STATE.md: package the portable gateway as checksummed single-binary release artifacts and smoke-test the native archive.
- Remaining unverified unchanged: legacy syntax outside the verified `SERVER`/`-P`/practical scoped `MOUNT`/`PERMIT`/`REJECT` subset; legacy adapter still one server per process invocation.

## 2026-09-03 19:56

- The iteration-9 work described as in progress above is now committed as
  `95c8547 [iter-9] drain listeners gracefully` and pushed to both remotes.
- Biggie Woodpecker pipeline 8 passed in 20 seconds, including the expanded
  Windows/amd64 portability build.
- The next ready slice is canonical TOML loading.

## 2026-09-03 20:03

- Iteration 10 adds strict canonical TOML parsing for server, mount, and policy
  configuration, backed by a pure-Go decoder with no transitive dependencies.
- `delegate check` and `delegate serve` accept `--config PATH` or
  `--config=PATH`; mixed legacy/file configuration is rejected before file or
  listener activity.
- The complete local gate passes: formatting, vet, unit tests, race tests, and
  CGO-free builds for Darwin/arm64, Linux/amd64, Linux/arm64, illumos/amd64,
  and Windows/amd64. Biggie Woodpecker pipeline 10 passed in 13 seconds at
  `364497a`.

## 2026-09-03 20:12

- Iteration 11 adds `delegate explain`, a side-effect-free evaluation path
  using the same mount resolver and policy engine as the HTTP frontend.
- Structured results cover permit, reject, no-mount, unsafe-path, and
  ambiguous-mount outcomes; successful routing includes the rewritten target
  and policy rule index.
- The complete local gate passes, including race tests and all five portable
  compile targets. Biggie Woodpecker pipeline 12 passed in 9 seconds at
  `795f81f`.
- A side review of copyparty added design guardrails for self-contained
  deployment, mount-adjacent authorization, explicit trusted-proxy boundaries,
  and atomic whole-config reloads. It also added trusted-proxy and packaging
  follow-ups without expanding iteration 11's implementation scope.
- Its comparison with adjacent tools reinforced multiple listeners, virtual
  namespaces, reverse-proxy placement, runtime reload, and genuinely portable
  release artifacts as the operational capabilities worth measuring; the
  existing compatibility ledger remains test-backed rather than a feature
  count.

## 2026-09-03 20:23

- Iteration 12 adds named-server and protocol scopes to mounts in canonical
  TOML and the legacy `MOUNT` adapter.
- Validation rejects unknown or contradictory scopes. The resolver, decision
  explanation, and live HTTP handler all use the same frontend identity and
  deterministic scoped-fallback behavior.
- The complete local gate passes, including race tests, five portable compile
  targets, and a scoped explanation smoke test. Biggie Woodpecker pipeline 13
  passed in 10 seconds at `1835d58`.

## 2026-09-03 20:25

- Iteration 13 coordinates multiple canonical HTTP frontends under one
  lifecycle. Duplicate addresses fail validation, all sockets are pre-bound,
  partial bind failure rolls back, and cancellation drains every listener.
- The complete local gate passes, including race tests and all five portable
  compile targets. Biggie Woodpecker pipeline 14 passed in 9 seconds at
  `53793ef`.

## 2026-09-03 20:33

- Iteration 14 adds an atomic immutable configuration store with validated
  all-or-nothing replacement and rollback on failure.
- HTTP handlers take one snapshot per request for both routing and policy.
  Mutation-isolation and concurrent replacement tests pass under the race
  detector.
- The complete local gate passes; remote Woodpecker acceptance remains to be
  run. Signal-driven file reload is the next slice.

## 2026-09-03 20:38

- Iteration 15 adds canonical TOML file reload with atomic publication,
  rollback on parse or validation failure, and restart-required rejection for
  listener-topology changes. The complete local gate passes; Biggie
  Woodpecker pipeline 16 passed at `f56724b`.

## 2026-09-03 20:45

- Iteration 16 wires canonical-file reload to `SIGHUP` on Unix-family systems.
  A platform-neutral watcher test covers successful reload, rejected rollback,
  reporting, and cancellation; Windows compiles with signal reload disabled.
- The complete local gate passes. Remote Woodpecker acceptance remains.

## 2026-09-03 21:34

- New commit: `ebb1046 [iter-21] build reproducible release archives`. Matches the iteration-21 summary already logged out-of-band at 21:31 in this file: `scripts/release.sh` produces deterministic CGO-free zip archives for Darwin/arm64, Linux/amd64, Linux/arm64, illumos/amd64, and Windows/amd64 with trimmed paths, no VCS metadata, empty build ID, link-time version stamp, and a stable SHA-256 manifest; two real builds produced identical manifests, all checksums verified, and the extracted Darwin binary reported its version and passed `check` against the canonical example.
- **Release packaging** is `[DONE]` (BACKLOG.md line 205/321). All Priority 1 and TLS-related work through checksummed release archives is now complete.
- New current gate per CURRENT-STATE.md: add fail-closed URL-authority source patterns to `MOUNT` as the routing prerequisite for HTTP forward proxying. Working tree already has in-progress, uncommitted work toward it: modified `BACKLOG.md`, `config/legacy_test.go`, `mount/mount_test.go`, `mount/resolve_test.go`.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 21:44

- New commit: `9b53249 [iter-22] match absolute URL mount sources`. Matches the iteration-22 summary already logged out-of-band at 21:39 in this file: `MOUNT` extended with absolute HTTP/HTTPS source URLs in both strict TOML and legacy directives while preserving path mounts; resolution compares scheme/authority without DNS, folds host case, distinguishes explicit ports, and fails closed on userinfo, query, fragment, escaping, traversal, malformed authority, and non-canonical paths; live absolute-form requests and `delegate explain --url` share the resolver; a bounded URL-source fuzz run completed 393,080 executions without a panic.
- New `[IN PROGRESS]` item (BACKLOG.md line 224): strip proxy credentials and hop-by-hop headers on both sides of the connection — this is the current gate per CURRENT-STATE.md, a prerequisite before expanding forward-proxy behavior. Working tree already has uncommitted work toward it: modified `BACKLOG.md`, `server/http.go`, `server/http_test.go`.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 21:54

- New commit: `02b357f [iter-23] strip HTTP hop-by-hop metadata`. Matches the iteration-23 summary already logged out-of-band at 21:45 in this file: requests and responses now strip standard hop-by-hop fields, `Proxy-Connection`, proxy credentials/challenges, and every repeated case-insensitive header nominated by `Connection`, while end-to-end values remain intact; a real `http.Client` proxy integration test proves `Proxy-Authorization` and connection-scoped fields never reach the backend and backend hop-by-hop fields never return to the caller.
- Working tree is now clean, in sync with `origin/main` — nothing currently `[IN PROGRESS]` in BACKLOG.md.
- New current gate per CURRENT-STATE.md: add authorized, bounded HTTP `CONNECT` as a byte-stream relay separate from semantic Fetch translation.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 22:04

- No new commits since the last entry (`02b357f` still HEAD, in sync with `origin/main`). Only local change is this file itself.
- BACKLOG.md unchanged: nothing currently `[IN PROGRESS]`. CURRENT-STATE.md gate unchanged: authorized, bounded HTTP `CONNECT` as a byte-stream relay separate from semantic Fetch translation.
- Quiet tick — work appears paused since iteration 23 landed.

## 2026-09-03 22:07

- Owner narrowed active build and release work to Darwin/arm64 and Linux/amd64
  until further notice. Local checks, Woodpecker, release packaging, and the
  standing agent constraint now use exactly that matrix; prior broader builds
  remain historical facts only.

## 2026-09-03 22:14

- New commit: `6730dba [iter-24] narrow active build matrix` — the code-level counterpart of the matrix decision already logged out-of-band at 22:07 in this file.
- **New `[IN PROGRESS]` item** (BACKLOG.md line 234): add a bounded byte-stream relay for authorized HTTP `CONNECT`, separate from semantic Fetch translation — this is the current gate per CURRENT-STATE.md. Work has visibly started: untracked `connector/tcp.go`, `connector/tcp_test.go`; modified (uncommitted) `cmd/delegate/http_runtime.go`, `explain/explain_test.go`, `mount/mount.go`, `mount/mount_test.go`, `mount/resolve_test.go`, `mount/source_fuzz_test.go`, `operation/operation.go`, `server/http.go`, `server/http_test.go`, `BACKLOG.md`.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 22:18

- Iteration 25 adds explicit `connect://host:port/` to `tcp://host:port`
  mappings for authorized HTTP CONNECT. Authority validation, mount resolution,
  and policy approval all precede dialing; the byte relay has bounded dial,
  handshake, and rolling idle limits, propagates half-close, and closes both
  hijacked connections. Loopback tests prove bidirectional traffic and denial
  with zero dials; `explain` evaluates the route without network activity.
- The complete local gate passes on the owner-selected Darwin/arm64 and
  Linux/amd64 matrix. A bounded source-validation fuzz run completed 557,516
  executions and a dedicated CONNECT-authority run completed 187,611
  executions without a panic. Remote Woodpecker acceptance remains.

## 2026-09-03 22:27

- Iteration 26 gives authorized HTTP PUT a protocol-neutral Store operation
  instead of representing writes as Fetch. Store uses the same resolved mount,
  policy decision, sanitized metadata, and mount-selected HTTP/TLS transport;
  it preserves known content length and enforces a 32 MiB streaming cap.
- Focused tests prove the full loopback write and that denial or a declared
  oversized body invokes neither Store nor Fetch. The complete local gate
  passes on Darwin/arm64 and Linux/amd64; remote acceptance remains.

## 2026-09-03 22:24

- Two new commits confirm the work already logged out-of-band above: `d7a011d [iter-25] relay authorized HTTP CONNECT` and `a0f398d [iter-26] route HTTP PUT as typed Store`. Working tree is clean, in sync with `origin/main`.
- Nothing currently `[IN PROGRESS]` in BACKLOG.md. New current gate per CURRENT-STATE.md: select the next compatibility slice after the verified HTTP Fetch, Store, and CONNECT foundation. Build/release scope remains limited to Darwin/arm64 and Linux/amd64 per the owner's standing constraint.
- Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-03 22:35

- No new commits since the last entry (`a0f398d` still HEAD, in sync with `origin/main`). Only local change is this file itself.
- BACKLOG.md unchanged: nothing currently `[IN PROGRESS]`. CURRENT-STATE.md gate unchanged: select the next compatibility slice after the verified HTTP Fetch, Store, and CONNECT foundation. Build/release scope still limited to Darwin/arm64 and Linux/amd64.
- Quiet tick — work appears paused since iteration 26 landed.

## 2026-09-03 22:44

- Still no new commits (`a0f398d` remains HEAD, in sync with `origin/main`), but active work has resumed. New `[IN PROGRESS]` item (BACKLOG.md line 265): HTTP/FTP Fetch and Store translation. New current gate per CURRENT-STATE.md: select the next compatibility slice — FTP `LIST` translation and the differential compatibility harness against the original DeleGate implementation.
- Working tree has uncommitted, in-progress changes: modified `connector/routes.go`, `connector/routes_test.go`, `BACKLOG.md`, `COMPATIBILITY.md`, `CURRENT-STATE.md`; untracked `connector/ftp.go`.
- Build/release scope still limited to Darwin/arm64 and Linux/amd64. Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.

## 2026-09-04 05:00

- New commit `e17d7b8 [iter-27] add ftp fetch and store connector routing` adds
  protocol-neutral FTP Fetch and Store for `ftp://` mounts through a dedicated
  `FTP` connector. `routes.FetchForMount` / `StoreForMount` now dispatch by scheme,
  preserving policy and frontend selection behavior while adding URL-targeted FTP routing.
- Focused loopback tests for this iteration pass, and the project verification gate
  now passes end-to-end (formatting/vet/tests/race and CGO-free Darwin/arm64,
  Linux/amd64 builds).
- Working tree is clean and in sync with `origin/main`; current gate per
  `CURRENT-STATE.md` remains the differential compatibility slice: FTP `LIST`
  translation and compatibility harnesses against original DeleGate.

## 2026-09-03 23:14

- New commit: `0a1d631 [iter-29] add compatibility harness scaffolding`. Working tree clean, in sync with `origin/main`.
- FTP `LIST` translation is confirmed committed (`c99b4ca`/`ee9cb4f`, `iter-28`). Differential compatibility harness scaffolding against the original DeleGate implementation is now in place.
- New `[IN PROGRESS]` item (BACKLOG.md line 273): differential compatibility harness for an original DeleGate executable. New current gate per CURRENT-STATE.md: wire the compatibility harness to an actual reference executable.
- Build/release scope still limited to Darwin/arm64 and Linux/amd64. Remaining unverified unchanged: legacy syntax outside the verified subset; legacy adapter still one server per process invocation.
