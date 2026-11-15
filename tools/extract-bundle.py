#!/usr/bin/env python3
"""Static string-mining of a JS bundle or a Bun-compiled `claude` binary (any platform; never executed).

Usage: python3 -I extract-bundle.py <file> <label> [--version X] > bundle-<label>.json

Sections (all sets of literals, diffed as set add/remove):
  observedWire.subtypeLiterals      every subtype:"x" / subtype==="x" literal
  observedWire.controlRequestsSent  subtypes passed to request({subtype:...}) / sendControlRequest({subtype:...})
  observedWire.typeLiterals         type:"x" literals that look like wire envelopes (control_*, system, keep_alive, ...)
  cliFlags                          commander .option()/new Option() specs, marking hideHelp() ones
  slashCommands                     {type:"local"|"local-jsx"|"prompt",name:"x",description:"..."} command objects
  binaryStrings.envVars             CLAUDE_CODE_* / ANTHROPIC_* / DISABLE_* / ENABLE_* identifiers
"""
import hashlib
import json
import mmap
import re
import sys

path, label = sys.argv[1], sys.argv[2]
version = sys.argv[sys.argv.index("--version") + 1] if "--version" in sys.argv else None
f = open(path, "rb")
m = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_READ)


def uniq(pattern, group=1):
    return sorted({x.decode("utf8", "replace") for x in re.findall(pattern, m)} if group == 1 else set())


out = {"manifestVersion": 1, "source": {"kind": "bundle", "label": label, "file": path, "version": version, "bytes": len(m)},
       "observedWire": {}, "cliFlags": {}, "cliHiddenFlags": {}, "slashCommands": {}, "binaryStrings": {}}

ow = out["observedWire"]
ow["subtypeLiterals"] = uniq(rb'subtype(?::|===?|!==?)"([a-z][a-z0-9_]*)"')
ow["controlRequestsSent"] = uniq(rb'(?:request|sendControlRequest|sendRequest|controlRequest)\(\{subtype:"([a-z][a-z0-9_]*)"')
ow["typeLiterals"] = sorted({t for t in uniq(rb'type(?::|===?)"([a-z][a-z0-9_]*)"')
                             if re.match(r"^(control_|system$|assistant$|user$|result$|stream_event$|keep_alive$|tool_progress$|"
                                         r"tool_use_summary$|auth_status$|rate_limit_event$|prompt_suggestion$|conversation_reset$|"
                                         r"update_environment_variables$|transcript_mirror$|streamlined_|active_goal)", t)})

# commander options: .option("spec","desc" | new X("spec","desc")  followed (before the next option) by .hideHelp()
opt_re = re.compile(rb'(?:\.option|\.requiredOption|new [A-Za-z_$][\w$]{0,3})\("((?:-[A-Za-z0-9][\w]?, )?--[A-Za-z][\w-]*(?:, --[A-Za-z][\w-]*)?(?: [<\[][^"]*[>\]])?)",(?:"((?:[^"\\]|\\.){0,600})"|\'((?:[^\'\\]|\\.){0,600})\'|`([^`]{0,600})`)')
for mt in opt_re.finditer(m):
    spec = mt.group(1).decode()
    tail = m[mt.end(): mt.end() + 600]
    nxt = re.search(rb'\.(?:option|addOption|requiredOption)\(|new [A-Za-z_$][\w$]{0,3}\("-', tail)
    seg = tail[: nxt.start()] if nxt else tail
    hidden = b".hideHelp()" in seg
    choices = re.search(rb'\.choices\(\[([^\]]*)\]\)', seg)
    long_name = re.search(r"--[A-Za-z][\w-]*", spec).group(0)
    entry = {"optional": True, "type": (re.search(r"[<\[].*", spec) or [None])[0] or "boolean",
             "descHash": hashlib.sha256(mt.group(2) or mt.group(3) or mt.group(4) or b"").hexdigest()[:10]}
    if choices:
        entry["choices"] = sorted(re.findall(r'"([^"]*)"', choices.group(1).decode()))
    target = out["cliHiddenFlags"] if hidden else out["cliFlags"]
    target.setdefault(long_name, entry)  # first definition wins (subcommands may reuse names)

for mt in re.finditer(rb'\{type:"(local|local-jsx|prompt)",name:"([\w:-]+)"', m):
    name = mt.group(2).decode()
    seg = m[mt.end(): mt.end() + 800]
    end = seg.find(b'{type:"')
    seg = seg[: end if end > 0 else 500]  # stop at the next command object
    d = re.search(rb'(?:description:|get description\(\)\{return)(?:"((?:[^"\\]|\\.){0,300})"|`([^`]{0,300})`)', seg)
    al = re.search(rb'aliases:\[([^\]]*)\]', seg)
    out["slashCommands"].setdefault("/" + name, {
        "optional": True, "type": mt.group(1).decode(),
        **({"aliases": sorted(re.findall(r'"([^"]+)"', al.group(1).decode()))} if al else {}),
        "descHash": hashlib.sha256((d.group(1) or d.group(2)) if d else b"").hexdigest()[:10]})

out["binaryStrings"]["envVars"] = uniq(rb'\b((?:CLAUDE_CODE|ANTHROPIC|CLAUDE_AGENT_SDK)_[A-Z0-9_]{2,60})\b')
print(json.dumps(out, indent=2, sort_keys=True))
