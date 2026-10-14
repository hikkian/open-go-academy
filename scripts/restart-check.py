"""Automated real API process restart/persistence check (no runner required)."""
import json, pathlib, subprocess, time, urllib.request, http.cookiejar, tempfile
root=pathlib.Path(__file__).resolve().parent.parent
with tempfile.TemporaryDirectory(prefix='restart-',dir=root/'.data') as tmp:
 env=__import__('os').environ.copy();env.update(ACADEMY_ADDR='127.0.0.1:8090',ACADEMY_DB=str(pathlib.Path(tmp)/'academy.db'))
 client=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
 def request(path,body=None,method=None,token=''):
  data=None if body is None else json.dumps(body).encode()
  r=urllib.request.Request('http://127.0.0.1:8090/api'+path,data=data,method=method,headers={'Content-Type':'application/json','X-CSRF-Token':token})
  return json.load(client.open(r,timeout=3))
 def start():
  p=subprocess.Popen([str(root/'.tools/academy-api')],cwd=root,env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
  for _ in range(100):
   if p.poll() is not None:raise RuntimeError('API exited')
   try:request('/health');return p
   except OSError:time.sleep(.05)
  p.terminate();raise RuntimeError('API startup timed out')
 p=start()
 try:
  token=request('/profile')['csrf_token'];request('/profile',{'name':'Restart learner','theme':'dark'},'POST',token)
  request('/drafts/hello',{'files':{'main.go':'package main\nfunc main(){}'}},'PUT',token)
 finally:p.terminate();p.wait(timeout=5)
 p=start()
 try:
  profile=request('/profile')['profile'];draft=request('/drafts/hello')['files'];assert profile['name']=='Restart learner' and profile['theme']=='dark' and draft['main.go']=='package main\nfunc main(){}'
  print('PASS: actual API stop/restart preserved profile/theme/draft; migrations reapplied safely')
 finally:p.terminate();p.wait(timeout=5)
