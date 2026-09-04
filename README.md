# new-delegate

[![Woodpecker CI](http://biggie.lynx-eagle.ts.net:8110/api/badges/5/status.svg)](http://biggie.lynx-eagle.ts.net:8110/repos/5)

`new-delegate` is a portable, DeleGate-compatible application gateway written
in Go. It preserves the original program's strongest concepts: a daemon can
present one protocol on its client side, map virtual resources with `MOUNT`,
apply fail-closed access policy, and reach a different protocol on its server
side.

The first milestone is a tested compatibility kernel for `SERVER`, `MOUNT`,
`PERMIT`, and `REJECT`, followed by an HTTP-to-HTTP vertical slice. Protocol
translation is built around typed semantic operations rather than a bag of
protocol-specific optional fields.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

Validate original-style directives without starting a listener:

```sh
go run ./cmd/delegate check \
  SERVER=http -P8080 \
  'MOUNT=/* http://127.0.0.1:9000/*' \
  'PERMIT=http:127.0.0.1:*'
```

Remove `check` (or replace it with `serve`) to start the current HTTP frontend.
With no matching `PERMIT`, requests are denied by default.

Development is organized as small iterate-bot sessions. Each session takes the
highest-priority ready item from `BACKLOG.md`, begins with a failing test, makes
the smallest implementation that passes, updates verified state, and commits.

The canonical private origin is Gitea on Biggie. A private GitHub mirror feeds
Biggie's existing Woodpecker forge integration; it is CI transport, not a
second source of truth.
