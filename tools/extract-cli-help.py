#!/usr/bin/env python3
"""Extract CLI flags + subcommands from `claude --help` (recursively) into manifest sections.

Usage: python3 -I extract-cli-help.py /path/to/claude > cli-<ver>.json
Only runs `<bin> --version` and `<bin> [sub...] --help`; never starts a session.
"""
import json
import os
import re
import subprocess
import sys

BIN = sys.argv[1]
ENV = {**os.environ, "DISABLE_AUTOUPDATER": "1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1", "NO_COLOR": "1"}


def run(args):
    r = subprocess.run([BIN, *args], capture_output=True, text=True, env=ENV, timeout=60, stdin=subprocess.DEVNULL)
    return r.stdout + r.stderr


def parse_help(text):
    sections, cur, entries = {}, None, []
    for line in text.splitlines():
        m = re.match(r"^(\w[\w ]*):\s*$", line)
        if m:
            cur = m.group(1)
            sections[cur] = []
            continue
        if cur is None or not line.strip():
            continue
        if re.match(r"^  \S", line):  # new entry (2-space indent)
            parts = re.split(r"\s{2,}", line.strip(), maxsplit=1)
            sections[cur].append({"spec": parts[0], "desc": parts[1] if len(parts) > 1 else ""})
        elif sections[cur]:  # continuation
            sections[cur][-1]["desc"] = (sections[cur][-1]["desc"] + " " + line.strip()).strip()
    return sections


def option_entry(spec, desc):
    names = re.findall(r"(--?[\w-]+)", spec.split("<")[0].split("[")[0])
    arg = re.search(r"([<\[].*[>\]])", spec)
    choices = re.search(r"\(choices: ([^)]*)\)", desc)
    default = re.search(r"\(default: ([^)]*)\)", desc)
    long_name = next((n for n in names if n.startswith("--")), names[0] if names else spec)
    return long_name, {
        "aliases": sorted(n for n in names if n != long_name),
        "arg": arg.group(1) if arg else None,
        "choices": sorted(re.findall(r'"([^"]*)"', choices.group(1))) if choices else None,
        "default": default.group(1) if default else None,
        "descHash": __import__("hashlib").sha256(desc.encode()).hexdigest()[:10],
        "desc": desc[:200],
    }


def walk(path, out, depth=0):
    sec = parse_help(run([*path, "--help"]))
    key = " ".join(path) or "claude"
    for e in sec.get("Options", []):
        name, info = option_entry(e["spec"], e["desc"])
        out["cliFlags"][f"{key} {name}"] = {"optional": True, "type": info["arg"] or "boolean", **{k: v for k, v in info.items() if v is not None and k != "desc"}}
    for e in sec.get("Commands", []):
        cname = e["spec"].split()[0]
        if not re.match(r"^[a-z][\w-]*(\|[\w-]+)*$", cname):
            continue  # e.g. an "Examples:" block embedded in a command description
        primary = cname.split("|")[0]
        if primary == "help":
            continue
        out["cliCommands"][" ".join([*path, primary])] = {"optional": True, "type": e["spec"], "aliases": cname.split("|")[1:]}
        if depth < 2:
            walk([*path, primary], out, depth + 1)


if __name__ == "__main__":
    ver = run(["--version"]).strip().split()[0]
    out = {"manifestVersion": 1, "source": {"kind": "cli-help", "version": ver, "binary": BIN}, "cliFlags": {}, "cliCommands": {}}
    walk([], out)
    print(json.dumps(out, indent=2, sort_keys=True))
