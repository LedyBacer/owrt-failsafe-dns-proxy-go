#!/usr/bin/env python3

import argparse
import hashlib
import json
from pathlib import Path


def artifact(path: str) -> dict[str, str]:
    file_path = Path(path)
    return {
        "file": file_path.name,
        "sha256": hashlib.sha256(file_path.read_bytes()).hexdigest(),
    }


parser = argparse.ArgumentParser()
parser.add_argument("--output", required=True)
parser.add_argument("--openwrt-version", required=True)
parser.add_argument("--target", required=True)
parser.add_argument("--subtarget", required=True)
parser.add_argument("--pkgarch", required=True)
parser.add_argument("--package-manager", required=True, choices=("opkg", "apk"))
parser.add_argument("--daemon", required=True)
parser.add_argument("--luci", required=True)
parser.add_argument("--i18n-ru", required=True)
args = parser.parse_args()

document = {
    "openwrt_version": args.openwrt_version,
    "target": args.target,
    "subtarget": args.subtarget,
    "pkgarch": args.pkgarch,
    "package_manager": args.package_manager,
    "daemon": artifact(args.daemon),
    "luci": artifact(args.luci),
    "i18n_ru": artifact(args.i18n_ru),
}
Path(args.output).write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
