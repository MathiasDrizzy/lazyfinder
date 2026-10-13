#!/usr/bin/env python3
"""Refresh internal/i18n/testdata/lazymark.json from a lazymark checkout (read only).

usage: scripts/snapshot-lazymark-strings.py [path-to-lazymark]

The snapshot holds every i18n.T pair of lazymark (English -> Spanish), its catalogs for the other languages and the
git revision it was taken from. TestStringsShared compares it with lazyfinder's own texts, so the two apps word the
same thing the same way. The snapshot is committed so the test needs no lazymark checkout.
"""
import json, os, re, subprocess, sys

root = os.path.expanduser(sys.argv[1] if len(sys.argv) > 1 else "~/Documents/proyectos/lazymark")
pat = re.compile(r'i18n\.(?:T|E)\(\s*("(?:[^"\\]|\\.)*"|`[^`]*`)\s*,\s*("(?:[^"\\]|\\.)*"|`[^`]*`)\s*[,)]', re.S)
unq = lambda x: x[1:-1] if x[0] == "`" else json.loads(x)
es = {}
for d, _, files in os.walk(os.path.join(root, "internal")):
    for f in files:
        if f.endswith(".go") and not f.endswith("_test.go"):
            for m in pat.finditer(open(os.path.join(d, f)).read()):
                es[unq(m.group(2))] = unq(m.group(1))
catalog = {}
cdir = os.path.join(root, "internal", "i18n", "catalog")
for f in sorted(os.listdir(cdir)):
    if f.endswith(".json"):
        catalog[f[:-5]] = json.load(open(os.path.join(cdir, f)))
rev = subprocess.run(["git", "-C", root, "rev-parse", "--short", "HEAD"], capture_output=True, text=True).stdout.strip()
out = {"source": "lazymark " + rev, "es": es, "catalog": catalog}
path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "internal", "i18n", "testdata", "lazymark.json")
open(path, "w").write(json.dumps(out, indent=1, ensure_ascii=False, sort_keys=True) + "\n")
print("wrote", path, len(es), "texts")
