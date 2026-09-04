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

## Unverified

- Woodpecker repository activation and pipeline execution.
- Legacy syntax other than the verified `SERVER`, `-P`, and practical `MOUNT`
  subset, plus path resolution, policy, and gateway runtime behavior.

## Current gate

Connect Biggie's Gitea forge to Woodpecker and obtain a green remote build.
Local red/green iterations may proceed, but are not accepted until that remote
gate validates the accumulated commits.
