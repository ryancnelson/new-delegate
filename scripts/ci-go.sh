#!/bin/sh
set -eu

go_version="1.25.6"
go_sha256="f022b6aad78e362bcba9b0b94d09ad58c5a70c6ba3b7582905fababf5fe0181a"
tool_root="/tmp/new-delegate-toolchains/go${go_version}"

if [ ! -x "${tool_root}/bin/go" ]; then
	stage="$(mktemp -d "/tmp/new-delegate-go${go_version}.XXXXXX")"
	archive="${stage}/go.tar.gz"
	trap 'rm -rf "$stage"' EXIT HUP INT TERM
	wget -q -O "$archive" "https://go.dev/dl/go${go_version}.linux-amd64.tar.gz"
	printf '%s  %s\n' "$go_sha256" "$archive" | sha256sum -c - >/dev/null
	tar -C "$stage" -xzf "$archive"
	mkdir -p "$(dirname "$tool_root")"
	if ! mv "${stage}/go" "$tool_root" 2>/dev/null; then
		test -x "${tool_root}/bin/go"
	fi
	rm -rf "$stage"
	trap - EXIT HUP INT TERM
fi

export GOROOT="$tool_root"
export GOCACHE="/tmp/new-delegate-go-build-cache"
export GOPATH="/tmp/new-delegate-go-path"
export PATH="${GOROOT}/bin:${PATH}"

exec "$@"
