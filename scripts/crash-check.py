"""Real SIGKILL/restart reconciliation check; touches only this test's scoped runner."""
import hashlib, http.client, json, os, pathlib, socket, subprocess, tempfile, threading, time
root=pathlib.Path(__file__).resolve().parent.parent
class UnixHTTP(http.client.HTTPConnection):
 def __init__(self,path):super().__init__('localhost',timeout=65);self.path=path
 def connect(self):self.sock=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM);self.sock.settimeout(65);self.sock.connect(self.path)
with tempfile.TemporaryDirectory(prefix='crash-',dir=root/'.data') as tmp:
 path=str(pathlib.Path(tmp)/'runner.sock');scope=hashlib.sha256(path.encode()).hexdigest()[:16];env=os.environ.copy();env['ACADEMY_RUNNER_SOCKET']=path
 def ids():return subprocess.check_output(['podman','ps','--all','--filter','label=academy.install='+scope,'--format','{{.ID}}'],text=True).strip()
 def health():
  c=UnixHTTP(path)
  try:c.request('GET','/health');return c.getresponse().status==200
  finally:c.close()
 def start():
  p=subprocess.Popen([str(root/'.tools/academy-runner')],cwd=root,env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
  for _ in range(200):
   if p.poll() is not None:raise RuntimeError('runner exited during startup')
   try:
    if health():return p
   except OSError:pass
   time.sleep(.1)
  p.kill();raise RuntimeError('runner health startup timeout')
 p=start()
 def run():
  c=UnixHTTP(path)
  try:c.request('POST','/execute',json.dumps({'mode':'run','files':{'main.go':'package main\nfunc main(){for{}}'}}),{'Content-Type':'application/json'});c.getresponse().read()
  except (OSError,http.client.HTTPException):pass
  finally:c.close()
 t=threading.Thread(target=run,daemon=True);t.start()
 try:
  for _ in range(100):
   if ids():break
   time.sleep(.05)
  else:raise RuntimeError('job never started')
  p.kill();p.wait(timeout=5);assert pathlib.Path(path).exists(),'expected stale socket after SIGKILL'
  p=start();assert not ids(),'orphan job survived restart'
  print('PASS: real runner SIGKILL -> stale socket removed -> scoped orphan killed -> health restored')
 finally:
  if p.poll() is None:p.terminate();p.wait(timeout=10)
  # Safety cleanup only this fixture's scope in case a test assertion failed.
  for cid in ids().split():subprocess.run(['podman','rm','--force','--time=0',cid],check=True,stdout=subprocess.DEVNULL)
 t.join(timeout=5)
