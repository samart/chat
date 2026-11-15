#!/bin/zsh
# Run the free verification suite (help, hermetic probe, mock scenarios) against one CLI binary.
# usage: [MOCK=http://127.0.0.1:<port>] compare.sh <version> <binary>
# The mock must already be running: node mock-api.mjs --port <port> --log <file>
# Each scenario needs a fresh output directory (run-scenario.mjs refuses a non-empty cwd).
set -u
V=$1; BIN=$2; HERE=${0:A:h}; OUT=$HERE/runs/v$V; MOCK=${MOCK:-http://127.0.0.1:8787}
mkdir -p $OUT; cd $HERE
env -i HOME=$OUT/home PATH=/usr/bin:/bin $BIN --version > $OUT/version.txt 2>&1
env -i HOME=$OUT/home PATH=/usr/bin:/bin $BIN --help > $OUT/help.txt 2>&1
mkdir -p $HERE/../probe-cwd/$V-hermetic
node probe.mjs --bin $BIN --hermetic $MOCK --out $OUT/hermetic --cwd $HERE/../probe-cwd/$V-hermetic > $OUT/probe.log 2>&1; echo "probe exit $?"
node shape.mjs catalog $OUT/hermetic.catalog.json $OUT/hermetic/results.json > /dev/null 2>&1
node shape.mjs infer $OUT/hermetic.shapes.json $OUT/hermetic/transcript.ndjson > /dev/null 2>&1
# The second line onwards needs a mock-api.mjs that has the subagent/queue/compact scripts (see scenarios-2.1.311.md).
for s in text thinking tool_allow tool_deny file_edit plan_mode mcp_tool interrupt resume max_turns error_429 error_529 error_500 error_400 \
         subagent subagent_background subagent_stop subagent_stop_bg subagent_permission subagent_forward_text subagent_nested \
         subagent_parallel subagent_to_background subagent_interrupt subagent_bg_interrupt subagent_bg_interrupt_affordance subagent_failed \
         interrupt_pending_approval ask_user_question compact queued_message queued_interrupt queued_cancel \
         slash_prompt_command slash_prompt_command_replay task_output; do
  node run-scenario.mjs --bin $BIN --target mock --mock-url $MOCK --scenario $s --out $OUT/mock/$s > $OUT/mock-$s.log 2>&1; echo "$s exit $?"
done
node shape.mjs infer $OUT/mock/shapes.json $OUT/mock/*/transcript.ndjson > /dev/null 2>&1
for s in $OUT/mock/*/; do [[ -f $s/transcript.ndjson ]] && node shape.mjs seq $s/transcript.ndjson > $s/seq.txt; done
echo done
