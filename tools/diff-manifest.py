#!/usr/bin/env python3
"""Diff two protocol-surface manifests. Usage: python3 -I diff-manifest.py old.json new.json [--json]

Each change: {section, key, change: added|removed|changed-shape, impact: additive|breaking|neutral, stability, detail}
"""
import json
import sys

FIELD_SECTIONS = ["sdkMessages", "stdoutWire", "controlRequests", "controlResponses", "tools"]
FLAT_FIELD_SECTIONS = ["options", "queryMethods"]  # key -> {optional,type}
SECTION_STABILITY = {  # default stability per section; per-entry 'stability' field overrides
    "sdkMessages": "stable", "stdoutWire": "stable", "controlRequests": "stable", "controlResponses": "stable",
    "options": "stable", "queryMethods": "stable", "exports": "stable", "stringEnums": "stable", "tools": "stable",
    "cliFlags": "stable", "cliHiddenFlags": "observed", "settings": "stable", "observedWire": "observed",
    "binaryStrings": "internal", "slashCommands": "stable",
}


# Sections whose payloads the HOST writes (a new required field there breaks an old host).
HOST_PRODUCED = {"controlRequests", "options", "settings"}


def field_changes(old_f, new_f, host_produced=False):
    out = []
    for k in sorted(set(old_f) | set(new_f)):
        a, b = old_f.get(k), new_f.get(k)
        if a is None:
            out.append(("+", k, b, "breaking" if host_produced and not b.get("optional", True) else "additive"))
        elif b is None:
            out.append(("-", k, a, "breaking"))
        elif a != b:
            impact = "breaking" if (a.get("optional") and not b.get("optional")) else "neutral"
            out.append(("~", k, {"from": a, "to": b}, impact))
    return out


def diff(old, new):
    changes = []

    def add(section, key, change, impact, detail=None, stability=None):
        if (stability or SECTION_STABILITY.get(section)) == "internal":
            impact = "neutral"  # internal strings carry no compatibility promise
        changes.append({"section": section, "key": key, "change": change, "impact": impact,
                        "stability": stability or SECTION_STABILITY.get(section, "stable"), **({"detail": detail} if detail else {})})

    for sec in FIELD_SECTIONS:
        o, n = old.get(sec) or {}, new.get(sec) or {}
        for k in sorted(set(o) | set(n)):
            if k not in o:
                add(sec, k, "added", "additive", {"typeName": n[k].get("typeName"), "fields": sorted(n[k].get("fields", {}))}, n[k].get("stability"))
            elif k not in n:
                add(sec, k, "removed", "breaking", {"typeName": o[k].get("typeName")}, o[k].get("stability"))
            elif o[k].get("shapeHash") != n[k].get("shapeHash"):
                fc = field_changes(o[k].get("fields", {}), n[k].get("fields", {}), sec in HOST_PRODUCED)
                impact = "breaking" if any(c[3] == "breaking" for c in fc) else ("additive" if all(c[0] == "+" for c in fc) else "neutral")
                add(sec, k, "changed-shape", impact, [{"op": c[0], "field": c[1], "impact": c[3], "value": c[2]} for c in fc], n[k].get("stability"))
    for sec in FLAT_FIELD_SECTIONS + ["settings", "cliFlags", "cliHiddenFlags", "slashCommands"]:
        o, n = old.get(sec) or {}, new.get(sec) or {}
        for c in field_changes(o, n, sec in HOST_PRODUCED):
            ch = {"+": "added", "-": "removed", "~": "changed-shape"}[c[0]]
            add(sec, c[1], ch, c[3] if ch != "removed" else "breaking", c[2])
    o, n = old.get("exports") or {}, new.get("exports") or {}
    for k in sorted(set(o) | set(n)):
        if k not in o:
            add("exports", k, "added", "additive", n[k])
        elif k not in n:
            add("exports", k, "removed", "breaking", o[k])
        elif o[k]["shapeHash"] != n[k]["shapeHash"]:
            add("exports", k, "changed-shape", "neutral", {"tags": n[k].get("tags"), "signature": n[k].get("signature")})
    for sec in ["stringEnums", "observedWire", "binaryStrings"]:
        o, n = old.get(sec) or {}, new.get(sec) or {}
        for k in sorted(set(o) | set(n)):
            a, b = set(o.get(k) or []), set(n.get(k) or [])
            if k not in o:
                add(sec, k, "added", "additive", sorted(b))
            elif k not in n:
                add(sec, k, "removed", "breaking", sorted(a))
            elif a != b:
                add(sec, k, "changed-shape", "breaking" if a - b else "additive", {"added": sorted(b - a), "removed": sorted(a - b)})
    return changes


def window(a, b, ctx=40):
    """Show the first point where two type strings diverge."""
    i = next((j for j, (x, y) in enumerate(zip(a, b)) if x != y), min(len(a), len(b)))
    ea, eb = len(a), len(b)
    while ea > i and eb > i and a[ea - 1] == b[eb - 1]:
        ea, eb = ea - 1, eb - 1
    pre = "…" if i > ctx else ""
    return f"`{pre}{a[max(0, i - ctx):ea][:160]}…` -> `{pre}{b[max(0, i - ctx):eb][:200]}…`"


def render_md(old, new, changes):
    ov, nv = old["source"].get("version"), new["source"].get("version")
    lines = [f"# Protocol surface diff {ov} -> {nv}", ""]
    by = {}
    for c in changes:
        by.setdefault(c["section"], []).append(c)
    for sec, cs in by.items():
        lines.append(f"## {sec} ({len(cs)})")
        for c in cs:
            d = c.get("detail")
            s = f"- **{c['change']}** `{c['key']}` [{c['impact']}, {c['stability']}]"
            if c["change"] == "changed-shape" and isinstance(d, list):
                s += ": " + "; ".join(f"{x['op']}{x['field']}" for x in d)
            elif c["change"] == "changed-shape" and isinstance(d, dict) and "from" in d:
                ks = sorted(k for k in set(d["from"]) | set(d["to"]) if d["from"].get(k) != d["to"].get(k))
                s += ": " + "; ".join(f"{k}: {window(str(d['from'].get(k)), str(d['to'].get(k)))}" for k in ks)
            elif c["change"] == "changed-shape" and isinstance(d, dict) and "added" in d:
                s += f": +{d['added']} -{d['removed']}"
            elif c["change"] == "added" and isinstance(d, dict) and "fields" in d:
                s += f" ({d.get('typeName')}: {', '.join(d['fields'])})"
            elif c["change"] in ("added", "removed") and isinstance(d, dict) and "type" in d:
                s += f": `{d['type'][:160]}`"
            lines.append(s)
        lines.append("")
    return "\n".join(lines)


if __name__ == "__main__":
    old, new = (json.load(open(p)) for p in sys.argv[1:3])
    ch = diff(old, new)
    if "--json" in sys.argv:
        json.dump(ch, sys.stdout, indent=2)
    else:
        print(render_md(old, new, ch))
