#!/usr/bin/env python3
"""Build immutable local P2 images: trusted dependency policy and checker only."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
BASE = 'docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8'
PG = 'docker.io/library/postgres@sha256:9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d'
OUT = ROOT / '.data/p2-runtime'
OUT.mkdir(mode=0o700, parents=True, exist_ok=True)
os.chmod(OUT, 0o700)

def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)

def image_id(tag):
    value = run(['podman', 'image', 'inspect', tag, '--format', '{{.Id}}'], capture_output=True, text=True).stdout.strip()
    return value if value.startswith('sha256:') else 'sha256:' + value

with tempfile.TemporaryDirectory(prefix='p2-images-', dir=ROOT / '.data') as temp:
    build = Path(temp)
    for name in ['go.mod', 'go.sum']:
        policy = ROOT / 'internal/projects/p2' / ('locked.mod' if name == 'go.mod' else 'locked.sum')
        shutil.copyfile(policy, build / name)
        shutil.copyfile(policy, OUT / name)
        os.chmod(OUT / name, 0o600)
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
COPY checker /opt/academy/checker
USER 65532:65532
ENTRYPOINT ["/opt/academy/checker"]
''')
    run(['podman', 'build', '--pull=never', '--http-proxy=false', '--network=none', '-f', str(build / 'Containerfile.checker'), '-t', 'localhost/academy-p2-checker:local', str(build)])
    config = {'go_image': image_id('localhost/academy-p2-go:local'), 'checker_image': image_id('localhost/academy-p2-checker:local'), 'postgres_image': PG, 'go_version': '1.27.2', 'policy_sha256': {name: hashlib.sha256((OUT / name).read_bytes()).hexdigest() for name in ['go.mod', 'go.sum']}}
    (OUT / 'images.json').write_text(json.dumps(config, indent=2) + '\n')
    os.chmod(OUT / 'images.json', 0o600)
    print('PASS: immutable local images built; IDs and policy saved to .data/p2-runtime')
