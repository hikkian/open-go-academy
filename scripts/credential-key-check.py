#!/usr/bin/env python3
"""Actual process checks on a copied E2E DB; never changes the user's DB/key."""
import os, pathlib, shutil, sqlite3, subprocess, tempfile, time, urllib.request
root = pathlib.Path(__file__).resolve().parents[1]
source = root / '.data/e2e/academy.db'
binary = root / '.tools/academy-api'
http = urllib.request.build_opener(urllib.request.ProxyHandler({}))
def launch(folder):
    env = dict(os.environ, ACADEMY_ADDR='127.0.0.1:8094', ACADEMY_DB=str(folder/'academy.db'), ACADEMY_WEB=str(root/'web/dist'))
    for name in list(env):
        if name.startswith('ACADEMY_AI_') or name == 'ACADEMY_ISSUER_KEY': env.pop(name)
    return subprocess.Popen([str(binary)], cwd=root, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
def healthy(folder):
    process = launch(folder)
    try:
        for _ in range(100):
            if process.poll() is not None: raise RuntimeError('API startup failed: '+process.communicate()[0].decode())
            try:
                with http.open('http://127.0.0.1:8094/api/health', timeout=1) as response:
                    if response.status == 200: return process
            except OSError: time.sleep(.05)
        raise RuntimeError('API readiness timed out')
    except BaseException:
        process.terminate(); process.communicate(timeout=5); raise

def stop(process):
    process.terminate(); process.communicate(timeout=10)

def rejected(folder, marker):
    process = launch(folder)
    try: output = process.communicate(timeout=10)[0].decode()
    except subprocess.TimeoutExpired:
        process.kill(); process.communicate(); raise RuntimeError('startup did not fail closed')
    assert process.returncode != 0 and marker in output, 'expected fail-closed startup: '+output

with sqlite3.connect(source) as db:
    assert db.execute('SELECT COUNT(*) FROM credentials').fetchone()[0] > 0, 'Run native exam + credential E2E first'
    with tempfile.TemporaryDirectory(prefix='credential-key-check-', dir=root/'.data') as location:
        folder = pathlib.Path(location); original = folder/'issued'; original.mkdir(); empty = folder/'empty'; empty.mkdir()
        with sqlite3.connect(original/'academy.db') as copy: db.backup(copy)
        shutil.copy2(source.parent/'issuer.key', original/'issuer.key'); os.chmod(original/'issuer.key',0o600)
        process = healthy(original); stop(process)
        saved = (original/'issuer.key').read_bytes(); (original/'issuer.key').unlink()
        rejected(original,'issuer key missing')
        process = healthy(empty); stop(process)
        shutil.copy2(empty/'issuer.key', original/'issuer.key'); os.chmod(original/'issuer.key',0o600)
        rejected(original,'issuer key differs')
        (original/'issuer.key').write_bytes(saved); os.chmod(original/'issuer.key',0o600)
        process = healthy(original); stop(process)
print('PASS: actual API restart, missing-key rejection, different-key rejection, original-key recovery; copied E2E DB only')
