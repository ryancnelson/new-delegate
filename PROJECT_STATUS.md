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
