"""Verify actual second API process cannot recover an active exam in the same DB."""
import os,pathlib,sqlite3,subprocess,tempfile,time,urllib.request
root=pathlib.Path(__file__).resolve().parent.parent
with tempfile.TemporaryDirectory(prefix='api-lock-',dir=root/'.data') as tmp:
 db=str(pathlib.Path(tmp)/'academy.db');env=os.environ.copy();env.update(ACADEMY_DB=db,ACADEMY_ADDR='127.0.0.1:8092')
 first=subprocess.Popen([str(root/'.tools/academy-api')],cwd=root,env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 try:
  for _ in range(100):
   if first.poll() is not None:raise RuntimeError('first API exited')
   try:
    urllib.request.urlopen('http://127.0.0.1:8092/api/health',timeout=2).close();break
   except OSError:time.sleep(.05)
  with sqlite3.connect(db) as con:
   con.execute("INSERT INTO profiles(id,name,theme,rank,xp) VALUES('local','Fixture','light','Zero',0)")
   con.execute("INSERT INTO exam_attempts(id,profile_id,exam_id,version,rank,criteria_hash,token_hash,status,created_at) SELECT 'live','local',id,version,rank,criteria_hash,'fixture','grading','2026-10-15T00:00:00Z' FROM exams LIMIT 1")
  env['ACADEMY_ADDR']='127.0.0.1:8093';second=subprocess.run([str(root/'.tools/academy-api')],cwd=root,env=env,capture_output=True,timeout=10)
  assert second.returncode!=0 and b'database already used' in second.stderr,second.stderr
  with sqlite3.connect(db) as con:assert con.execute("SELECT status FROM exam_attempts WHERE id='live'").fetchone()[0]=='grading'
  print('PASS: second API/different port blocked before recovery; live grading preserved')
 finally:first.terminate();first.wait(timeout=10)
