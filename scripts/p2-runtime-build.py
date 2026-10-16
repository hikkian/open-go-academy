#!/usr/bin/env python3
"""Build immutable local P2 images: trusted dependency policy and checker only."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
BASE = 'docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8'
PG = 'docker.io/library/postgres@sha256:9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d'
P3 = ROOT / 'content/projects/p3'
OUT = ROOT / '.data/p2-runtime'
OUT.mkdir(mode=0o700, parents=True, exist_ok=True)
os.chmod(OUT, 0o700)

def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)

def image_id(tag):
    value = run(['podman', 'image', 'inspect', tag, '--format', '{{.Id}}'], capture_output=True, text=True).stdout.strip()
    return value if value.startswith('sha256:') else 'sha256:' + value

def normalized_go_mod(raw):
    lines = raw.decode().splitlines()
    if not lines or not lines[0].startswith('module '):
        raise RuntimeError('invalid project module policy')
    return '\n'.join(['module <project>'] + lines[1:]) + '\n'

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

# Validate the compiled P3 catalog lock against its checked-in starter/reference
# before building the immutable offline images. This runs only deterministic Go
# unit tests and does not start a database, Podman, or a user job.
run([str(ROOT / 'scripts/go.sh'), 'test', './internal/projects', './internal/projectrunner', './internal/projectrunner/checker'], cwd=ROOT)

with tempfile.TemporaryDirectory(prefix='p2-images-', dir=ROOT / '.data') as temp:
    build = Path(temp)
    p3_go_mod = P3 / 'starter/go.mod'
    p3_go_sum = P3 / 'starter/go.sum'
    p3_ref_mod = P3 / 'reference/go.mod'
    p3_ref_sum = P3 / 'reference/go.sum'
    if p3_go_mod.read_bytes() != p3_ref_mod.read_bytes() or p3_go_sum.read_bytes() != p3_ref_sum.read_bytes():
        raise RuntimeError('P3 starter module lock must exactly match its trusted reference')
    for name in ['go.mod', 'go.sum']:
        policy = ROOT / 'internal/projects/p2' / ('locked.mod' if name == 'go.mod' else 'locked.sum')
        shutil.copyfile(policy, build / name)
        shutil.copyfile(policy, OUT / name)
        os.chmod(OUT / name, 0o600)
    p2_module=(OUT/'go.mod').read_bytes()
    p2_sum=(OUT/'go.sum').read_bytes()
    if normalized_go_mod(p3_go_mod.read_bytes()) != normalized_go_mod(p2_module) or p3_go_sum.read_bytes() != p2_sum:
        raise RuntimeError('P3 module graph is not covered by the prewarmed offline Go cache')
    shutil.copyfile(p3_go_mod, OUT/'p3.mod')
    shutil.copyfile(p3_go_sum, OUT/'p3.sum')
    os.chmod(OUT/'p3.mod', 0o600)
    os.chmod(OUT/'p3.sum', 0o600)
    catalog = (ROOT / 'internal/projects/catalog.go').read_text()
    p4_match = re.search(r'^const p4Module = ("(?:[^"\\]|\\.)*")$', catalog, re.MULTILINE)
    if not p4_match:
        raise RuntimeError('cannot read the server P4 module policy')
    p4_go_mod = json.loads(p4_match.group(1))
    if not p4_go_mod.startswith('module ') or '\ngo 1.27.2\n' not in p4_go_mod:
        raise RuntimeError('P4 module lock differs from the pinned Go version')
    (OUT/'p4.mod').write_text(p4_go_mod)
    os.chmod(OUT/'p4.mod', 0o600)
    p5_match = re.search(r'^const p5Module = ("(?:[^"\\]|\\.)*")$', catalog, re.MULTILINE)
    if not p5_match:
        raise RuntimeError('cannot read the server P5 module policy')
    p5_go_mod = json.loads(p5_match.group(1))
    if not p5_go_mod.startswith('module ') or '\ngo 1.27.2\n' not in p5_go_mod:
        raise RuntimeError('P5 module lock differs from the pinned Go version')
    (OUT/'p5.mod').write_text(p5_go_mod)
    os.chmod(OUT/'p5.mod', 0o600)
    (build / 'warm.go').write_text('''package main
import (
 _ "github.com/jackc/pgx/v5/pgxpool"
 _ "net/http"
 _ "crypto/pbkdf2"
 _ "crypto/sha256"
 _ "encoding/json"
 _ "testing"
)
func main() {}
''')
    (build / 'Containerfile.go').write_text(f'''FROM {BASE}
ENV GOMODCACHE=/opt/academy/gomod GOCACHE=/opt/academy/gocache GOTOOLCHAIN=local CGO_ENABLED=0
WORKDIR /opt/academy/warm
COPY go.mod go.sum warm.go ./
RUN go mod download && go build -mod=readonly -o /tmp/warm . && rm -f /tmp/warm && chmod -R a+rX /opt/academy
WORKDIR /work
''')
    run(['podman', 'build', '--pull=never', '--http-proxy=false', '-f', str(build / 'Containerfile.go'), '-t', 'localhost/academy-p2-go:local', str(build)])
    checker = build / 'checker'
    run([str(ROOT / 'scripts/go.sh'), 'build', '-trimpath', '-ldflags=-s -w', '-o', str(checker), './internal/projectrunner/checker'], cwd=ROOT, env=dict(os.environ, CGO_ENABLED='0'))
    (build / 'Containerfile.checker').write_text(f'''FROM {BASE}
LABEL academy.checker.p3="taskflow-v1" \\
      academy.checker.p4="scalelab-deterministic-v1" \\
      academy.checker.p5="capstone-deterministic-v1"
COPY checker /opt/academy/checker
USER 65532:65532
ENTRYPOINT ["/opt/academy/checker"]
''')
    run(['podman', 'build', '--pull=never', '--http-proxy=false', '--network=none', '-f', str(build / 'Containerfile.checker'), '-t', 'localhost/academy-p2-checker:local', str(build)])
    config = {'go_image': image_id('localhost/academy-p2-go:local'), 'checker_image': image_id('localhost/academy-p2-checker:local'), 'postgres_image': PG, 'go_version': '1.27.2', 'p3_checker': 'taskflow-v1', 'p4_checker': 'scalelab-deterministic-v1', 'p5_checker': 'capstone-deterministic-v1', 'policy_sha256': {name: digest(OUT / name) for name in ['go.mod', 'go.sum', 'p3.mod', 'p3.sum', 'p4.mod', 'p5.mod']}}
    (OUT / 'images.json').write_text(json.dumps(config, indent=2) + '\n')
    os.chmod(OUT / 'images.json', 0o600)
    print('PASS: immutable offline P2/P3/P4/P5 Go profile and trusted checker identities built; IDs and policy saved to .data/p2-runtime')
