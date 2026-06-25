#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "${repo_dir}/scripts/build-openwrt.sh" 24.10.7 mediatek mt7622 "${FDP_OUTPUT_DIR:-${repo_dir}/dist}"
