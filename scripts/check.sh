#!/usr/bin/env sh
set -eu

GOCACHE="${TMPDIR:-/tmp}/new-delegate-go-build"
export GOCACHE

test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -race ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o "$GOCACHE/delegate-darwin-arm64" ./cmd/delegate
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$GOCACHE/delegate-linux-amd64" ./cmd/delegate
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$GOCACHE/delegate-linux-arm64" ./cmd/delegate
CGO_ENABLED=0 GOOS=illumos GOARCH=amd64 go build -o "$GOCACHE/delegate-illumos-amd64" ./cmd/delegate
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$GOCACHE/delegate-windows-amd64.exe" ./cmd/delegate
