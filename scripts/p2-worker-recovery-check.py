#!/usr/bin/env python3
"""Cancel/SIGKILL recovery using a dedicated worker and disposable ordinary job."""
import hashlib
import http.client
import json
from pathlib import Path
import socket
import subprocess
import tempfile
import threading
import time

ROOT=Path(__file__).resolve().parents[1]
config=json.loads((ROOT/'.data/p2-runtime/images.json').read_text())
files={name:(ROOT/'.data/p2-runtime'/name).read_text() for name in ['go.mod','go.sum']}
files.update({'README.md':'Disposable ordinary recovery fixture.', 'main.go':'package main\nimport "net/http"\nfunc main(){ http.HandleFunc("/healthz",func(w http.ResponseWriter,r *http.Request){w.Write([]byte(`{"status":"ok"}`))}); http.ListenAndServe(":8097",nil) }\n'})
with tempfile.TemporaryDirectory(prefix='p2-recovery-',dir=ROOT/'.data') as temp:
    folder=Path(temp);sock=folder/'worker.sock'
    install=hashlib.sha256(str(sock.resolve()).encode()).hexdigest()
    def containers():
        return subprocess.run(['podman','ps','-aq','--filter','label=academy.project-runner=true','--filter','label=academy.install='+install],check=True,capture_output=True,text=True).stdout.split()
    class Client(http.client.HTTPConnection):
        def connect(self):
            self.sock=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM);self.sock.settimeout(220);self.sock.connect(str(sock))
    log=open(folder/'worker.log','w')
    def start():
        process=subprocess.Popen([str(ROOT/'.tools/academy-project-runner'),'-socket',str(sock),'-go-image',config['go_image'],'-checker-image',config['checker_image'],'-expected-go-mod',str(ROOT/'.data/p2-runtime/go.mod'),'-expected-go-sum',str(ROOT/'.data/p2-runtime/go.sum')],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
        for _ in range(150):
            if process.poll() is not None:raise RuntimeError('dedicated recovery worker failed startup')
            c=Client('worker')
            try:
                c.request('GET','/health');r=c.getresponse();r.read()
                if r.status==200:return process
            except OSError:pass
            finally:c.close()
            time.sleep(.1)
        raise RuntimeError('dedicated recovery worker startup timed out')
    def begin_job():
        client=Client('worker')
        client.request('POST','/execute',json.dumps({'project_id':'p2','action':'run','files':files}),{'Content-Type':'application/json'})
        def receive():
            try:r=client.getresponse();r.read()
            except (OSError,http.client.HTTPException):pass
        thread=threading.Thread(target=receive,daemon=True);thread.start()
        for _ in range(100):
            if containers():return client,thread
            time.sleep(.05)
        raise RuntimeError('ordinary disposable job did not start')
    process=start()
    try:
        client,thread=begin_job()
        # The handler observes cancellation through its request context.
        client.sock.shutdown(socket.SHUT_RDWR);client.close();thread.join(timeout=3)
        for _ in range(150):
            if not containers():break
            time.sleep(.1)
        else:raise RuntimeError('canceled job was not cleaned')
        print('PASS: disconnected request cancels its job and removes owned containers',flush=True)
        client,thread=begin_job();process.kill();process.wait(timeout=5);client.close();thread.join(timeout=3)
        assert containers(),'SIGKILL fixture did not leave an orphan to reconcile'
        process=start()
        assert not containers(),'restarted dedicated worker did not reconcile its orphans'
        print('PASS: actual SIGKILL, stale socket recovery and scoped orphan cleanup',flush=True)
    finally:
        if process.poll() is None:process.terminate();process.wait(timeout=25)
        log.close()
        # Only this test installation can be removed, including failure paths.
        owned=containers()
        if owned:subprocess.run(['podman','rm','-fv',*owned],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
