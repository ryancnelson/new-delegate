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

## Unverified

- Woodpecker repository activation and pipeline execution.
- Legacy syntax other than the verified `SERVER`, `-P`, and practical `MOUNT`
  subset, plus gateway runtime behavior. Legacy `PERMIT`/`REJECT` parsing and
  mount scoping by named server/protocol remain deferred.

## Current gate

Connect Biggie's Gitea forge to Woodpecker and obtain a green remote build.
Local red/green iterations may proceed, but are not accepted until that remote
gate validates the accumulated commits.
