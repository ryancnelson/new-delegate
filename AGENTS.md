# new-delegate agent instructions

This is an iterate-bot project. Work through `BACKLOG.md` in priority order.

## Required loop

1. Read `CURRENT-STATE.md`, `DESIGN.md`, `COMPATIBILITY.md`, and `BACKLOG.md`.
2. Select exactly one ready Priority 1 item.
3. Mark it `[IN PROGRESS]`.
4. Write a focused failing test and run it to prove RED.
5. Implement the smallest coherent behavior that makes the test pass.
6. Run `./scripts/check.sh`.
7. Update `CURRENT-STATE.md`, `COMPATIBILITY.md`, and `BACKLOG.md` only with
   verified facts.
8. Commit with `[iter-N]` in the subject and push `main` to `origin`.
9. Require the Woodpecker pipeline to pass before starting another item.

## Engineering constraints

- Go standard library first. Add dependencies only with a recorded reason.
- Keep the normal build compatible with `CGO_ENABLED=0`.
- Until the owner expands the matrix, build and package only Darwin/arm64 and
  Linux/amd64.
- Do not write assembly.
- Do not contact a backend before mount resolution and policy both succeed.
- Unknown directives, ambiguous mounts, and unsupported translations fail
  closed.
- Wire-protocol types and semantic operations are separate layers.
- Transparent byte relay is separate from semantic protocol translation.
- Tests use loopback listeners, ephemeral ports, fake clocks, and in-memory
  fixtures. They must not depend on the public network.
- Every parser gets table tests, malformed-input tests, and fuzz coverage.
- Never place credentials in configuration values, logs, test fixtures, or
  command arguments. Configuration stores provider references only.
