#!/bin/sh
set -e
S=/private/tmp/claude-501/-Users-user-Development-other-project/459f0484-dabf-4eb9-807f-8143c0170df0/scratchpad
W=$S/parity/work
cd $W
node gen.mjs $S/tools/ts $S/sdk-pkgs/v0.3.311/package/sdk.d.ts $S/sdk-pkgs/v0.3.325/package/sdk.d.ts extern.d.ts since.json > types_gen.go
{ cat prelude.go.txt; sed -e '1,/^import "encoding\/json"$/d' types_gen.go; } > $S/parity/types.go
gofmt -w $S/parity/types.go
rm -rf $W/gomod && mkdir -p $W/gomod && cp $S/parity/types.go $W/gomod/
cd $W/gomod && go mod init x >/dev/null 2>&1 && gofmt -l . ; go vet ./...
