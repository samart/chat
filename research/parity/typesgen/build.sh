#!/bin/sh
set -e
# Paths are relative to the repo root. SDK packages are not committed; unpack them
# into reference/sdk-pkgs/v<version>/package/ first (see README).
W=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$W/../../.." && pwd)
P=$ROOT/research/parity
PKGS=$ROOT/reference/sdk-pkgs
cd "$W"
node gen.mjs "$ROOT/tools/ts" "$PKGS/v0.3.311/package/sdk.d.ts" "$PKGS/v0.3.325/package/sdk.d.ts" extern.d.ts since.json > types_gen.go
{ cat prelude.go.txt; sed -e '1,/^import "encoding\/json"$/d' types_gen.go; } > "$P/types.go"
gofmt -w "$P/types.go"
rm -rf "$W/gomod" && mkdir -p "$W/gomod" && cp "$P/types.go" "$W/gomod/"
cd $W/gomod && go mod init x >/dev/null 2>&1 && gofmt -l . ; go vet ./...
