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
- `connector`: backend implementations.
- `observability`: structured events, metrics, and audit records.

## Invariants

- Configuration is parsed and validated completely before activation.
- Reload failure leaves the previous configuration active.
- Paths are normalized and validated before mount matching.
- A request cannot reach a connector without an affirmative policy decision.
- Equal-specificity mounts require distinct priorities or validation fails.
- Protocol decoding is bounded by configured sizes and deadlines.
- Secret values never enter the canonical configuration or observability
  event model.

## Portability

Darwin/arm64 is the primary development platform. Linux/amd64 and Linux/arm64
are runtime targets. illumos/amd64 and Windows remain compile-checked where the
standard library permits. The default build uses pure Go and no assembly.

