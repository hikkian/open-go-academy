#!/usr/bin/env python3
"""Start the isolated P2 worker and optionally expose a verified P3 profile."""
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
P2_RUNTIME = ROOT / '.data/p2-runtime'
PG_IMAGE = 'docker.io/library/postgres@sha256:9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d'
IMAGE_RE = re.compile(r'(?:sha256:[0-9a-f]{64}|[A-Za-z0-9][A-Za-z0-9._/:+-]*@sha256:[0-9a-f]{64})\Z')


def _read_expected(root, name):
    return (root / name).read_bytes()


def validate_policy(policy, policy_dir, root):
    """Validate P2 and every optional immutable P3/P4/P5 profile present."""
    if policy.get('go_version') != '1.27.2' or policy.get('postgres_image') != PG_IMAGE:
        raise ValueError('P2 immutable Go/PostgreSQL image policy mismatch')
    for key in ('go_image', 'checker_image'):
        if not IMAGE_RE.fullmatch(policy.get(key, '')):
            raise ValueError(f'{key} must be an immutable cached image reference')
    hashes = policy.get('policy_sha256', {})
    for name, source in (
        ('go.mod', root / 'internal/projects/p2/locked.mod'),
        ('go.sum', root / 'internal/projects/p2/locked.sum'),
    ):
        data = (policy_dir / name).read_bytes()
        if data != source.read_bytes() or hashlib.sha256(data).hexdigest() != hashes.get(name):
            raise ValueError(f'locked P2 module policy mismatch: {name}')

    enabled = False
    if policy.get('p3_checker') is not None:
        if policy['p3_checker'] != 'taskflow-v1':
            raise ValueError('unsupported P3 trusted checker profile')
        for name, starter, reference in (
            ('p3.mod', root / 'content/projects/p3/starter/go.mod', root / 'content/projects/p3/reference/go.mod'),
            ('p3.sum', root / 'content/projects/p3/starter/go.sum', root / 'content/projects/p3/reference/go.sum'),
        ):
            data = (policy_dir / name).read_bytes()
            if data != starter.read_bytes() or data != reference.read_bytes() or hashlib.sha256(data).hexdigest() != hashes.get(name):
                raise ValueError(f'locked P3 module policy mismatch: {name}')
        enabled = True

    for project_id, key, profile, source in (
        ('P4', 'p4_checker', 'scalelab-deterministic-v1', root / 'internal/projects/p4bundle/go.mod.template'),
        ('P5', 'p5_checker', 'capstone-deterministic-v1', root / 'internal/projects/p5bundle/go.mod.template'),
    ):
        if policy.get(key) is None:
            continue
        if policy[key] != profile:
            raise ValueError(f'unsupported {project_id} trusted checker profile')
        name = project_id.lower() + '.mod'
        data = (policy_dir / name).read_bytes()
        if data != source.read_bytes() or hashlib.sha256(data).hexdigest() != hashes.get(name):
            raise ValueError(f'locked {project_id} module policy mismatch: {name}')
        enabled = True
    return enabled


def self_test():
    with tempfile.TemporaryDirectory(prefix='project-runner-policy-') as temp:
        policy_dir = Path(temp)
        p2_files = (
            ('go.mod', ROOT / 'internal/projects/p2/locked.mod'),
            ('go.sum', ROOT / 'internal/projects/p2/locked.sum'),
        )
        for name, source in p2_files:
            (policy_dir / name).write_bytes(source.read_bytes())
        base = {
            'go_image': 'sha256:' + 'a' * 64,
            'checker_image': 'sha256:' + 'b' * 64,
            'postgres_image': PG_IMAGE,
            'go_version': '1.27.2',
            'policy_sha256': {name: hashlib.sha256((policy_dir / name).read_bytes()).hexdigest() for name, _ in p2_files},
        }
        if validate_policy(base, policy_dir, ROOT):
            raise AssertionError('legacy P2-only profile unexpectedly enabled P3')
        combined = dict(base)
        combined['p3_checker'] = 'taskflow-v1'
        combined['policy_sha256'] = dict(base['policy_sha256'])
        for name, source in (
            ('p3.mod', ROOT / 'content/projects/p3/starter/go.mod'),
            ('p3.sum', ROOT / 'content/projects/p3/starter/go.sum'),
        ):
            (policy_dir / name).write_bytes(source.read_bytes())
            combined['policy_sha256'][name] = hashlib.sha256((policy_dir / name).read_bytes()).hexdigest()
        if not validate_policy(combined, policy_dir, ROOT):
            raise AssertionError('complete P3 profile was not enabled')
        combined['p4_checker'] = 'scalelab-deterministic-v1'
        (policy_dir / 'p4.mod').write_bytes((ROOT / 'internal/projects/p4bundle/go.mod.template').read_bytes())
        combined['policy_sha256']['p4.mod'] = hashlib.sha256((policy_dir / 'p4.mod').read_bytes()).hexdigest()
        if not validate_policy(combined, policy_dir, ROOT):
            raise AssertionError('complete P3/P4 profile was not enabled')
        (policy_dir / 'p4.mod').write_bytes(b'module forged\n\ngo 1.27.2\n')
        if hashlib.sha256((policy_dir / 'p4.mod').read_bytes()).hexdigest() == combined['policy_sha256']['p4.mod']:
            raise AssertionError('forged test setup unexpectedly retained the valid digest')
        try:
            validate_policy(combined, policy_dir, ROOT)
        except ValueError:
            pass
        else:
            raise AssertionError('forged P4 module policy was accepted')
        (policy_dir / 'p4.mod').write_bytes((ROOT / 'internal/projects/p4bundle/go.mod.template').read_bytes())
        combined['p5_checker'] = 'capstone-deterministic-v1'
        (policy_dir / 'p5.mod').write_bytes((ROOT / 'internal/projects/p5bundle/go.mod.template').read_bytes())
        combined['policy_sha256']['p5.mod'] = hashlib.sha256((policy_dir / 'p5.mod').read_bytes()).hexdigest()
        if not validate_policy(combined, policy_dir, ROOT):
            raise AssertionError('complete P3/P4/P5 profile was not enabled')
        (policy_dir / 'p5.mod').write_bytes(b'module forged\n\ngo 1.27.2\n')
        try:
            validate_policy(combined, policy_dir, ROOT)
        except ValueError:
            pass
        else:
            raise AssertionError('forged P5 module policy was accepted')
    print('PASS: P2-only cache remains valid; P3/P4/P5 immutable lock profiles validate')


def main():
    if sys.argv[1:] == ['--self-test']:
        self_test()
        return
    if sys.argv[1:]:
        raise SystemExit('unsupported project-runner.py option')
    policy = json.loads((P2_RUNTIME / 'images.json').read_text())
    try:
        validate_policy(policy, P2_RUNTIME, ROOT)
    except (OSError, ValueError) as err:
        raise SystemExit(str(err)) from err
    os.chdir(ROOT)
    binary = ROOT / '.tools/academy-project-runner'
    socket = os.environ.get('ACADEMY_PROJECT_RUNNER_SOCKET', '.data/project-runner.sock')
    os.execv(str(binary), [str(binary), '-socket', socket, '-go-image', policy['go_image'], '-checker-image', policy['checker_image']])


if __name__ == '__main__':
    main()
