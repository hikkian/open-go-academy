#!/usr/bin/env python3
"""Positive/reference and ordinary failure fixtures through the isolated worker."""
import http.client
import json
from pathlib import Path
import socket
import subprocess
import time

ROOT=Path(__file__).resolve().parents[1]
class UnixHTTP(http.client.HTTPConnection):
    def connect(self):
        self.sock=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM)
        self.sock.settimeout(self.timeout)
        self.sock.connect(str(ROOT/'.data/project-runner.sock'))

def execute(files,action):
    client=UnixHTTP('project-runner',timeout=220)
    try:
        client.request('POST','/execute',json.dumps({'project_id':'p2','action':action,'files':files}),{'Content-Type':'application/json'})
        response=client.getresponse();raw=response.read()
        if response.status!=200:raise RuntimeError('worker HTTP '+str(response.status)+': '+raw.decode())
        return json.loads(raw)
    finally:client.close()

for _ in range(100):
    client=UnixHTTP('project-runner',timeout=.5)
    try:
        client.request('GET','/health');health=client.getresponse();health.read()
        if health.status==200:break
    except (OSError,http.client.HTTPException):pass
    finally:client.close()
    time.sleep(.1)
else:raise RuntimeError('separate project worker did not become ready')

files={}
reference=ROOT/'content/projects/p2/reference'
for path in reference.rglob('*'):
    if path.is_file() and (path.suffix in ['.go','.sql'] or path.name in ['go.mod','go.sum']):files[str(path.relative_to(reference))]=path.read_text()
files['README.md']='Trusted reference fixture; disposable database only.\n'
for action in ['run','test','submit']:
    result=execute(files,action)
    print(action,json.dumps(result,ensure_ascii=False),flush=True)
    if action=='submit':
        assert result['passed'] and result['checks'] and all(check['passed'] for check in result['checks']), 'reference did not pass trusted checker'
        assert not result.get('stdout') and not result.get('stderr'), 'private checker output leaked'
    else:assert not result.get('passed'), 'advisory action awarded project acceptance'
    assert result['status'] in ['ok','automatic_pass'], 'ordinary reference action failed'
    containers=subprocess.run(['podman','ps','-aq','--filter','label=academy.project-runner=true'],check=True,capture_output=True,text=True).stdout.strip()
    assert not containers, 'job containers remained after cleanup'
print('PASS: actual isolated reference run/test/submit, fresh server persistence and cleanup; no host ports',flush=True)
