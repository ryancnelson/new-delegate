# new-delegate — Project Status

Running changelog, updated automatically every 10 minutes.

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
  compile targets. Remote Woodpecker acceptance remains to be run.
- A side review of copyparty added design guardrails for self-contained
  deployment, mount-adjacent authorization, explicit trusted-proxy boundaries,
  and atomic whole-config reloads. It also added trusted-proxy and packaging
  follow-ups without expanding iteration 11's implementation scope.
- Its comparison with adjacent tools reinforced multiple listeners, virtual
  namespaces, reverse-proxy placement, runtime reload, and genuinely portable
  release artifacts as the operational capabilities worth measuring; the
  existing compatibility ledger remains test-backed rather than a feature
  count.
