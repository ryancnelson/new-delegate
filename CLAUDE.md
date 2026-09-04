# new-delegate — Iterate-Bot Project

This project uses the iterate-bot methodology: small, focused, tested improvements done consistently.

## Current Iteration

Check iteration count: `git log --oneline --grep='\[iter-' | wc -l`
Iteration branch naming: `iter-N-feature-name`

## How to Run a Session

1. Read `BACKLOG.md` — focus on Priority 1 items
2. Select the highest ready Priority 1 item; do not pause for confirmation
3. Run `./scripts/next-iteration.sh feature-name` to set up the branch
4. Implement, run tests, commit with `[iter-N]` prefix in the message
5. Mark item done in `BACKLOG.md`, move to Completed section
6. Every 8 iterations: read `STRATEGIC-REVIEW-CHECKLIST.md`

## Test Command

go test ./...

## Constraints

- One improvement per iteration
- All tests must pass before commit
- Update BACKLOG.md after each iteration
- Strategic review every 8 iterations
- Use RED-GREEN-REFACTOR: demonstrate the focused test failing before implementation
- Push each completed iteration to `origin/main`
- Require its Woodpecker pipeline to pass before beginning another iteration
- Keep the default build pure Go with `CGO_ENABLED=0`; do not write assembly
- Read `DESIGN.md`, `COMPATIBILITY.md`, and `CURRENT-STATE.md` before editing
- Unknown configuration and unsupported operations fail closed


## Project Notes

**Language:** Go
**Description:** Modern DeleGate-compatible protocol gateway with fail-closed policy and protocol translation
