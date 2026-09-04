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

## 2026-09-03 19:48

- Iteration 7 adds original-style `PERMIT` and `REJECT` parsing with golden and
  table tests.
- Policy evaluation now supports source CIDRs and destination suffix patterns
  without runtime name resolution.
- The complete local gate, including race tests and portable builds, passes.

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
