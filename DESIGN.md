# Design

## Product boundary

`new-delegate` is a modern successor to the original DeleGate application
gateway. It supports ordinary forward and reverse proxying, terminating
gateways, and deliberate cross-protocol operations. Agent-facing endpoints are
one use of the gateway rather than a separate architecture.

## Request path

```text
connection
  -> frontend server
  -> protocol decoder
  -> typed wire message
  -> semantic operation
  -> mount resolver
  -> policy engine
  -> backend connector
  -> semantic result
  -> protocol encoder
  -> connection
```

Transparent forwarding uses a byte-stream relay path after routing and policy
approval. It does not manufacture semantic operations.

## Stable boundaries

- `config`: canonical configuration plus TOML and legacy syntax adapters.
- `server`: listener lifecycle, limits, deadlines, and connection context.
- `protocol`: typed frontend and backend wire codecs.
- `operation`: protocol-neutral capabilities such as Fetch, Store, List,
  SendMail, Connect, and Relay.
- `mount`: deterministic mapping from frontend names to backend targets.
- `policy`: explicit permit/reject decisions with default deny.
- `tlsconfig`: independent, side-effect-free frontend and backend TLS policy.
- `tlsruntime`: explicit certificate loading and standard-library TLS values.
- `connector`: backend implementations.
- `observability`: structured events, metrics, and audit records.

## Invariants

- Configuration is parsed and validated completely before activation.
- Reload failure leaves the previous configuration active.
- Paths are normalized and validated before mount matching.
- A request cannot reach a connector without an affirmative policy decision.
- The direct peer is authoritative unless it is an explicitly trusted proxy;
  malformed client-address chains from trusted proxies fail closed.
- Equal-specificity mounts require distinct priorities or validation fails.
- Protocol decoding is bounded by configured sizes and deadlines.
- Secret values never enter the canonical configuration or observability
  event model.
- Backend TLS verification cannot be disabled by configuration.
- Runtime TLS material is loaded before listeners bind; backend transport
  choice comes from the resolver-selected mount, never request metadata.

## Portability

Darwin/arm64 is the primary development platform. Linux/amd64 and Linux/arm64
are runtime targets. illumos/amd64 and Windows/amd64 are compile-checked. The
default build uses pure Go and no assembly.

## Reference-derived guardrails

Copyparty demonstrates that a broad, multi-protocol service can remain useful
when it is a drop-in artifact with minimal mandatory dependencies. Its volume
permissions, reverse-proxy warnings, optional feature boundaries, and reload
model are useful comparisons for this project:

- Keep the default distribution self-contained; integrations may add optional
  capabilities but must not make the core gateway depend on external services.
- Keep authorization close to the selected mount while retaining one typed,
  protocol-neutral policy engine.
- Treat proxy-supplied client addresses as untrusted unless the immediate peer
  is in an explicitly configured trusted-proxy set.
- Reload the complete validated configuration atomically; never leave a mix of
  old and new configuration sections active.

References: [9001/copyparty](https://github.com/9001/copyparty) and its
[comparison of adjacent tools](https://github.com/9001/copyparty/blob/hovudstraum/docs/versus.md).
