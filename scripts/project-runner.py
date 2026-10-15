#!/usr/bin/env python3
"""Start the separate local P2 worker with the last built immutable images."""
import json
import os
from pathlib import Path

root = Path(__file__).resolve().parents[1]
policy = json.loads((root / '.data/p2-runtime/images.json').read_text())
os.chdir(root)
os.execv(str(root / '.tools/academy-project-runner'), [str(root / '.tools/academy-project-runner'), '-go-image', policy['go_image'], '-checker-image', policy['checker_image']])
