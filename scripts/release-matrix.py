#!/usr/bin/env python3

import argparse
import json
from pathlib import Path


parser = argparse.ArgumentParser()
parser.add_argument("--targets", default="build/release-targets.json")
args = parser.parse_args()

document = json.loads(Path(args.targets).read_text(encoding="utf-8"))
print(json.dumps({"include": document["targets"]}, separators=(",", ":")))
