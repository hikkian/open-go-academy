#!/usr/bin/env python3
"""Submit repository-owned function references through the real confined API."""
import http.cookiejar,json,pathlib,socket,subprocess,tempfile,time,urllib.request,urllib.error,sys
ROOT=pathlib.Path(__file__).resolve().parents[1]
manifest=json.loads((ROOT/'internal/content/manifest.json').read_text())
if sys.argv[1:]:
 assert all(lesson in manifest['published_lessons'] for lesson in sys.argv[1:]),'only published lesson IDs may be checked'
 manifest['published_lessons']=sys.argv[1:]
with socket.socket() as reservation: reservation.bind(('127.0.0.1',0));port=reservation.getsockname()[1]
with tempfile.TemporaryDirectory(prefix='function-check-',dir=ROOT/'.data') as location:
 import os
 folder=pathlib.Path(location);env=dict(os.environ,ACADEMY_ADDR=f'127.0.0.1:{port}',ACADEMY_DB=str(folder/'academy.db'),ACADEMY_WEB=str(ROOT/'web/dist'))
 for name in list(env):
  if name.startswith('ACADEMY_AI_') or name=='ACADEMY_ISSUER_KEY':env.pop(name)
 log=open(folder/'api.log','w+');process=subprocess.Popen([str(ROOT/'.tools/academy-api')],cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT)
 jar=http.cookiejar.CookieJar();client=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPCookieProcessor(jar));base=f'http://127.0.0.1:{port}'
 csrf=''
 def call(path,body=None,method=None):
  raw=None if body is None else json.dumps(body,ensure_ascii=False).encode();request=urllib.request.Request(base+path,data=raw,method=method,headers={'Content-Type':'application/json','X-CSRF-Token':csrf})
  for retry in range(3):
   try:
    with client.open(request,timeout=70) as response:return json.load(response)
   except urllib.error.HTTPError as error:
    if error.code==429 and retry<2:time.sleep(60);continue
    raise RuntimeError('API request failed '+path+': '+str(error.code)+' '+error.read().decode()) from None
 try:
  for _ in range(100):
   if process.poll() is not None:raise RuntimeError('test API startup failed')
   try:health=call('/api/health');break
   except OSError:time.sleep(.05)
  else:raise RuntimeError('test API did not become ready')
  assert health['runner_available'],'real isolated runner is required'
  csrf=call('/api/profile')['csrf_token'];call('/api/profile',{'name':'Проверка эталонов','theme':'light'},'POST')
  exercises=[]
  for lesson_id in manifest['published_lessons']:
   lesson=json.loads((ROOT/f'internal/content/data/{lesson_id}.json').read_text())
   exercises.extend(e for e in lesson['exercises'] if e.get('kind')=='function')
  for exercise in exercises:
   exercise_id=exercise['id'];source=(ROOT/f'internal/content/solutions/{exercise_id}/main.go').read_text()
   result=call(f'/api/exercises/{exercise_id}/submit',{'files':{'main.go':source}},'POST')
   assert result['passed'] and all(c['passed'] for c in result['checks']),exercise_id+' did not pass all native cases'
   assert result.get('stderr','')=='' and 'academy_driver.go' not in json.dumps(result), 'private adapter leaked'
   print('PASS',exercise_id,len(result['checks']),'actual confined cases',flush=True)
  progress=call('/api/progress');assert progress['xp']==10*len(exercises)
  learning=call('/api/learning');assert learning['current_rank']=='Zero' and not learning['verified'],'exercise XP must not award rank'
  if exercises:
   first=exercises[0]['id'];source=(ROOT/f'internal/content/solutions/{first}/main.go').read_text();call(f'/api/exercises/{first}/submit',{'files':{'main.go':source}},'POST');assert call('/api/progress')['xp']==progress['xp']
  print('PASS: '+str(len(exercises))+' native references, persisted evidence, idempotent XP, no earned rank; temporary DB only',flush=True)
 finally:
  process.terminate();process.wait(timeout=15);log.close()
