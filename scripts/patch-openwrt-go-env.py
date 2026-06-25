#!/usr/bin/env python3

import sys
from pathlib import Path


def patch(path: Path) -> bool:
    lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
    output: list[str] = []
    changed = False
    for index, line in enumerate(lines):
        output.append(line)
        if line.strip() != "GOENV=off \\":
            continue
        next_line = lines[index + 1].strip() if index + 1 < len(lines) else ""
        if next_line == "GOTOOLCHAIN=local \\":
            continue
        indent = line[: len(line) - len(line.lstrip())]
        output.append(f"{indent}GOTOOLCHAIN=local \\\n")
        changed = True
    if changed:
        path.write_text("".join(output), encoding="utf-8")
    return changed


if len(sys.argv) < 2:
    raise SystemExit("usage: patch-openwrt-go-env.py FILE...")

for argument in sys.argv[1:]:
    file_path = Path(argument)
    if patch(file_path):
        print(f"patched Go toolchain environment: {file_path}")
