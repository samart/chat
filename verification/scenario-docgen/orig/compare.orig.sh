#!/bin/zsh
# Run the free verification suite (help, hermetic probe, mock scenarios) against one CLI binary.
# usage: compare.sh <version> <binary>
set -u
V=$1; BIN=$2; HERE=${0:A:h}; OUT=$HERE/runs/v$V; MOCK=http://127.0.0.1:8787
mkdir -p $OUT; cd $HERE
env -i HOME=$OUT/home PATH=/usr/bin:/bin $BIN --version > $OUT/version.txt 2>&1
env -i HOME=$OUT/home PATH=/usr/bin:/bin $BIN --help > $OUT/help.txt 2>&1
mkdir -p $HERE/../probe-cwd/$V-hermetic
node probe.mjs --bin $BIN --hermetic $MOCK --out $OUT/hermetic --cwd $HERE/../probe-cwd/$V-hermetic > $OUT/probe.log 2>&1; echo "probe exit $?"
node shape.mjs catalog $OUT/hermetic.catalog.json $OUT/hermetic/results.json > /dev/null 2>&1
node shape.mjs infer $OUT/hermetic.shapes.json $OUT/hermetic/transcript.ndjson > /dev/null 2>&1
for s in text thinking tool_allow tool_deny file_edit plan_mode mcp_tool interrupt resume max_turns error_429 error_529 error_500 error_400; do
  node run-scenario.mjs --bin $BIN --target mock --mock-url $MOCK --scenario $s --out $OUT/mock/$s > $OUT/mock-$s.log 2>&1; echo "$s exit $?"
done
node shape.mjs infer $OUT/mock/shapes.json $OUT/mock/*/transcript.ndjson > /dev/null 2>&1
for s in $OUT/mock/*/; do n=${s:h:t}; done
echo done
