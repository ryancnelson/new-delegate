#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 VERSION OUTPUT_DIRECTORY" >&2
  exit 2
fi

exec go run ./tools/release "$1" "$2"
