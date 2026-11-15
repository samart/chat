#!/usr/bin/env python3
"""Slice a Keep-a-Changelog-ish CHANGELOG.md to the versions in (old, new] and classify bullets.

Usage: python3 -I extract-changelog.py CHANGELOG.md OLD NEW [--diff diff.json]
With --diff (output of diff-manifest.py --json), reports which manifest changes are mentioned in the
changelog (explained) and which are not (silent changes worth a closer look).
"""
import json
import re
import sys

text = open(sys.argv[1], encoding="utf8").read()
old, new = sys.argv[2], sys.argv[3]
vkey = lambda v: tuple(int(x) for x in v.split("."))
PROTOCOL_HINTS = re.compile(r"`[^`]+`|\bsystem/\w+|\bSDK\b|stream-json|control[_ ]request|--[a-z][\w-]+|\bsetting\b|\bhook\b|\bQuery\.|\boption\b", re.I)

releases = []
for m in re.finditer(r"^## (\d+\.\d+\.\d+)\s*\n(.*?)(?=^## |\Z)", text, re.M | re.S):
    v = m.group(1)
    if vkey(old) < vkey(v) <= vkey(new):
        bullets = [b.strip() for b in re.findall(r"^- (.*)$", m.group(2), re.M)]
        releases.append({"version": v, "bullets": [
            {"verb": (re.match(r"(\w+)", b).group(1) if re.match(r"(\w+)", b) else ""),
             "protocol": bool(PROTOCOL_HINTS.search(b)),
             "idents": sorted(set(re.findall(r"`([^`]+)`", b))), "text": b} for b in bullets]})

out = {"range": [old, new], "releases": releases,
       "counts": {"releases": len(releases), "bullets": sum(len(r["bullets"]) for r in releases),
                  "protocolBullets": sum(b["protocol"] for r in releases for b in r["bullets"])}}

if "--diff" in sys.argv:
    changes = json.load(open(sys.argv[sys.argv.index("--diff") + 1]))
    corpus = " ".join(b["text"] for r in releases for b in r["bullets"]).lower()
    norm = lambda s: re.sub(r"[^a-z0-9]", "", s.lower())
    ncorpus = norm(corpus)
    expl, silent = [], []
    for c in changes:
        cands = [c["key"].split(":")[-1].split("#")[0].lstrip("/-")]
        if isinstance(c.get("detail"), list):
            cands += [d["field"] if isinstance(d, dict) else str(d) for d in c["detail"]]
        if isinstance(c.get("detail"), dict) and "added" in c["detail"]:
            cands += c["detail"]["added"] + c["detail"]["removed"]
        hit = any(len(x) > 3 and norm(x) in ncorpus for x in cands)
        (expl if hit else silent).append(f'{c["section"]}:{c["key"]} {c["change"]}')
    out["crossCheck"] = {"explained": expl, "silent": silent}

print(json.dumps(out, indent=2))
