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

## Unverified

- Legacy syntax other than the verified `SERVER`, `-P`, and practical `MOUNT`
  subset, plus gateway runtime behavior. Legacy `PERMIT`/`REJECT` parsing and
  mount scoping by named server/protocol remain deferred.

## Current gate

Add canonical policy configuration and the original `PERMIT`/`REJECT` syntax,
then make the tested HTTP slice runnable from validated configuration.
