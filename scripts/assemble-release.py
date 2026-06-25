#!/usr/bin/env python3

import argparse
import hashlib
import json
import shutil
from pathlib import Path


parser = argparse.ArgumentParser()
parser.add_argument("--input", required=True)
parser.add_argument("--output", required=True)
parser.add_argument("--tag", required=True)
parser.add_argument("--installer")
args = parser.parse_args()

input_dir = Path(args.input)
output_dir = Path(args.output)
output_dir.mkdir(parents=True, exist_ok=True)

artifacts = []
seen_targets = set()
for metadata_path in sorted(input_dir.rglob("build-metadata.json")):
    metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
    target_key = (
        metadata["openwrt_version"],
        metadata["target"],
        metadata["subtarget"],
    )
    if target_key in seen_targets:
        raise SystemExit(f"duplicate build metadata for {target_key}")
    seen_targets.add(target_key)
    prefix = "openwrt-{}-{}-{}".format(*target_key).replace("/", "-")
    release_entry = {
        key: metadata[key]
        for key in (
            "openwrt_version",
            "target",
            "subtarget",
            "pkgarch",
            "package_manager",
        )
    }
    for package_key in ("daemon", "luci", "i18n_ru"):
        source = metadata_path.parent / metadata[package_key]["file"]
        if not source.is_file():
            raise SystemExit(f"missing artifact: {source}")
        actual_sha = hashlib.sha256(source.read_bytes()).hexdigest()
        if actual_sha != metadata[package_key]["sha256"]:
            raise SystemExit(f"checksum mismatch: {source}")
        release_name = f"{prefix}__{source.name}"
        destination = output_dir / release_name
        shutil.copy2(source, destination)
        release_entry[package_key] = {
            "file": release_name,
            "sha256": actual_sha,
        }
    artifacts.append(release_entry)

if not artifacts:
    raise SystemExit("no build metadata found")

if args.installer:
    installer = Path(args.installer)
    if not installer.is_file():
        raise SystemExit(f"missing installer: {installer}")
    shutil.copy2(installer, output_dir / "install.sh")

manifest = {
    "schema": 1,
    "project": "owrt-failsafe-dns-proxy-go",
    "release": args.tag,
    "artifacts": artifacts,
}
(output_dir / "manifest.json").write_text(
    json.dumps(manifest, indent=2) + "\n", encoding="utf-8"
)

checksum_lines = []
for path in sorted(output_dir.iterdir()):
    if path.name == "SHA256SUMS" or not path.is_file():
        continue
    checksum_lines.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}")
(output_dir / "SHA256SUMS").write_text(
    "\n".join(checksum_lines) + "\n", encoding="utf-8"
)
