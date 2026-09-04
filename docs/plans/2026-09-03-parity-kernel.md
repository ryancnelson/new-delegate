# DeleGate parity kernel implementation plan

> **For Claude:** Use `${SUPERPOWERS_SKILLS_ROOT}/skills/collaboration/executing-plans/SKILL.md` to implement this plan task-by-task.

**Goal:** Build a portable Go gateway that accepts original-style `SERVER`,
`MOUNT`, `PERMIT`, and `REJECT` directives and proxies an authorized HTTP
request to a mounted HTTP backend.

**Architecture:** Legacy and TOML syntax adapters produce one canonical typed
configuration. A frontend decodes typed HTTP messages, maps them to semantic
operations, resolves a mount, obtains an explicit policy decision, and only
then invokes a connector.

**Tech Stack:** Go 1.25 standard library, table-driven tests, native fuzzing,
and Woodpecker CI.

---

### Task 1: Bootstrap remote verification

**Files:**
- Create: `go.mod`
- Create: `.woodpecker/test.yml`
- Create: `scripts/check.sh`

1. Run `./scripts/check.sh` and verify every local gate passes.
2. Commit the bootstrap as `[iter-0]` and push `main`.
3. Verify Biggie's Woodpecker run passes before functional work.

### Task 2: Canonical server model

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

1. Write table tests for required server name, protocol, and listen address;
   duplicate server names; and a valid HTTP server.
2. Run `go test ./config -run TestConfigValidate` and verify RED because the
   model does not exist.
3. Add `Config`, `Server`, and `Validate` with only the tested behavior.
4. Run the focused test, then `./scripts/check.sh`, and verify GREEN.
5. Update the ledgers, commit as `[iter-1]`, push, and require green CI.

### Task 3: Original `SERVER` adapter

**Files:**
- Create: `config/legacy.go`
- Create: `config/legacy_test.go`
- Create: `config/testdata/server.golden.json`

1. Write failing golden/table tests for `SERVER=http`, `-P8080`, whitespace,
   repetitions, invalid ports, and unknown directives.
2. Implement tokenization into canonical `Server` values without runtime side
   effects.
3. Run focused and full gates, update ledgers, commit, push, and require CI.

### Task 4: Mount model and legacy adapter

**Files:**
- Create: `mount/mount.go`
- Create: `mount/mount_test.go`
- Modify: `config/legacy.go`
- Modify: `config/legacy_test.go`

1. Test absolute frontend paths, supported target URLs, wildcard preservation,
   repeated mounts, quoting, and invalid forms.
2. Implement the smallest canonical `Mount` model and parser behavior.
3. Run all gates, update ledgers, commit, push, and require CI.

### Task 5: Path safety and mount resolution

**Files:**
- Create: `mount/path.go`
- Create: `mount/path_test.go`
- Create: `mount/path_fuzz_test.go`
- Create: `mount/resolve.go`
- Create: `mount/resolve_test.go`

1. Test ordinary normalization, encoded traversal, NUL bytes, exact matches,
   wildcard matches, precedence, no-match, and ambiguity.
2. Add a fuzz target asserting normalization never panics and accepted results
   are idempotent.
3. Implement normalization and resolution as pure functions.
4. Run focused tests, a bounded fuzz smoke test, and all gates.
5. Update ledgers, commit, push, and require CI.

### Task 6: Policy kernel

**Files:**
- Create: `policy/policy.go`
- Create: `policy/policy_test.go`

1. Test default deny, explicit permit, explicit reject, and the invariant that
   a denied operation cannot invoke a fake connector.
2. Implement typed decisions with machine-readable reason codes.
3. Run all gates, update ledgers, commit, push, and require CI.

### Task 7: HTTP-to-HTTP acceptance slice

**Files:**
- Create: `operation/operation.go`
- Create: `connector/http.go`
- Create: `server/http.go`
- Create: `server/http_test.go`
- Create: `cmd/delegate/main.go`

1. Start an in-process backend and write a failing test that sends an HTTP
   request through the gateway.
2. Assert mount rewriting, policy authorization, response propagation, and
   zero backend connections for denial.
3. Implement only the required HTTP operation, connector, and handler.
4. Run all gates and a bounded race-enabled integration test.
5. Update ledgers, commit, push, and require CI.
