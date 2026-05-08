# -*- coding: utf-8 -*-
"""Replace handleQuery/handleJSON/... with Serve* under internal/handler/ (recursive, exclude serve.go)."""
import pathlib

root = pathlib.Path(__file__).resolve().parents[1] / "internal" / "handler"

repls = [
    ("handleQuery(c,", "ServeQuery(c,"),
    ("handleJSON(c,", "ServeJSON(c,"),
    ("handleJSONCreated(c,", "ServeJSON201(c,"),
    ("handleQueryOK(c,", "ServeQueryOK(c,"),
    ("handleJSONOK(c,", "ServeJSONOK(c,"),
    ("handleQueryWithKind(c,", "ServeQueryWithKind(c,"),
    ("handleQueryWithKindOK(c,", "ServeQueryWithKindOK(c,"),
]

changed = []
for p in sorted(root.rglob("*.go")):
    if p.name == "serve.go":
        continue
    text = p.read_text(encoding="utf-8")
    new = text
    for old, new_s in repls:
        new = new.replace(old, new_s)
    if new != text:
        p.write_text(new, encoding="utf-8")
        changed.append(str(p.relative_to(root)))

print("updated", len(changed), "files")
if changed:
    print("\n".join(changed))
