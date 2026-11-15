#!/usr/bin/env python3
"""Flatten claude-code-settings.schema.json into dotted setting paths -> {optional,type}.

Usage: python3 -I extract-settings-schema.py claude-code-settings.schema.json VERSION > settings-<ver>.json
"""
import hashlib
import json
import sys

schema = json.load(open(sys.argv[1]))
defs = schema.get("$defs", {}) | schema.get("definitions", {})


def resolve(node):
    seen = 0
    while isinstance(node, dict) and "$ref" in node and seen < 20:
        node = defs.get(node["$ref"].split("/")[-1], {})
        seen += 1
    return node


def tname(n):
    n = resolve(n)
    if "enum" in n:
        return "enum(" + "|".join(sorted(map(str, n["enum"]))) + ")"
    if "const" in n:
        return f"const({n['const']})"
    for k in ("anyOf", "oneOf"):
        if k in n:
            return " | ".join(sorted({tname(x) for x in n[k]}))
    t = n.get("type", "any")
    if t == "array":
        return f"array<{tname(n.get('items', {}))}>"
    return t if isinstance(t, str) else "|".join(t)


def walk(node, prefix, out, required=(), depth=0):
    node = resolve(node)
    for k, v in (node.get("properties") or {}).items():
        p = f"{prefix}.{k}" if prefix else k
        rv = resolve(v)
        out[p] = {"optional": k not in required, "type": tname(v),
                  "descHash": hashlib.sha256(rv.get("description", "").encode()).hexdigest()[:10],
                  **({"deprecated": True} if rv.get("deprecated") or "deprecated" in rv.get("description", "").lower()[:40] else {})}
        if depth < 3 and (rv.get("properties")):
            walk(rv, p, out, rv.get("required", ()), depth + 1)


out = {}
walk(schema, "", out, schema.get("required", ()))
print(json.dumps({"manifestVersion": 1, "source": {"kind": "settings-schema", "version": sys.argv[2], "$id": schema.get("$id")},
                  "settings": dict(sorted(out.items()))}, indent=2))
