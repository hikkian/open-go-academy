#!/usr/bin/env python3
"""Trusted reference tests in rootless containers sharing an isolated DB netns.
No host ports, user source, host module cache, paid services or persisted DB.
"""
import os, pathlib, secrets, shutil, subprocess, tempfile, time
ROOT=pathlib.Path(__file__).resolve().parents[1]
PG='docker.io/library/postgres@sha256:9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d'
GO='docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8'
cache=ROOT/'.data/p2-module-cache';cache.mkdir(exist_ok=True)
env=dict(os.environ,GOMODCACHE=str(cache))
subprocess.run([str(ROOT/'scripts/go.sh'),'-C',str(ROOT/'content/projects/p2/reference'),'mod','download'],env=env,check=True)
name='oga-p2-check-'+secrets.token_hex(4);password=secrets.token_urlsafe(24)
policy=['--rm','--pull=never','--http-proxy=false','--read-only','--cap-drop=ALL','--security-opt=no-new-privileges','--cpus=1','--pids-limit=128','--stop-timeout=0','--label=academy.project-check=true']
def run(arguments,**options):
    result=subprocess.run(arguments,check=False,**options)
    if result.returncode!=0: raise RuntimeError('isolated container step failed with exit '+str(result.returncode))
    return result
try:
    run(['podman','run','-d','--name',name,*policy,'--network=none','--user=999:999','--memory=512m','--memory-swap=512m','--shm-size=64m','--tmpfs=/tmp:rw,noexec,nosuid,nodev,size=64m','--tmpfs=/var/run/postgresql:rw,nosuid,nodev,size=16m,mode=1777','--tmpfs=/var/lib/postgresql:rw,nosuid,nodev,size=384m,mode=1777','-e','POSTGRES_USER=academy','-e','POSTGRES_DB=linkhub','-e','POSTGRES_PASSWORD='+password,'-e','POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256',PG,'postgres','-c','shared_buffers=32MB','-c','max_connections=20'],stdout=subprocess.DEVNULL)
    for _ in range(80):
        ready=subprocess.run(['podman','exec','-e','PGPASSWORD='+password,name,'psql','-h','127.0.0.1','-U','academy','-d','linkhub','-Atc','SELECT 1'],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
        if ready.returncode==0 and ready.stdout.strip()==b'1':break
        time.sleep(.25)
    else:
        logs=subprocess.run(['podman','logs',name],capture_output=True,text=True).stdout
        raise RuntimeError('isolated PostgreSQL did not become ready: '+logs.replace(password,'[redacted]'))
    version=run(['podman','exec','-e','PGPASSWORD='+password,name,'psql','-h','127.0.0.1','-U','academy','-d','linkhub','-Atc','SHOW server_version'],stdout=subprocess.PIPE,text=True).stdout.strip();print('PostgreSQL:',version,flush=True)
    import re, sys, json
    manifest=json.loads((ROOT/'internal/content/manifest.json').read_text())
    lesson_ids=manifest['published_lessons']+sys.argv[1:]
    checked=set()
    for lesson_id in lesson_ids:
        if lesson_id in checked:continue
        if not re.fullmatch(r'(?:zero|junior|juniorplus)-[0-9]{2}',lesson_id):raise RuntimeError('invalid SQL lesson ID')
        checked.add(lesson_id)
        chapter=(ROOT/'internal/content/data'/str(lesson_id+'.md')).read_text()
        for index,sql in enumerate(re.findall(r'^```sql[^\n]*\n(.*?)^```',chapter,re.MULTILINE|re.DOTALL),1):
            run(['podman','exec','-i',name,'psql','-U','academy','-d','linkhub','-v','ON_ERROR_STOP=1','-c','SET statement_timeout=5000','-f','-'],input=sql,text=True,stdout=subprocess.DEVNULL,timeout=25)
            print('PASS actual PostgreSQL SQL example:',lesson_id,index,flush=True)

    with tempfile.TemporaryDirectory(prefix='p2-reference-',dir=ROOT/'.data') as location:
        workspace=pathlib.Path(location);shutil.copytree(ROOT/'content/projects/p2/reference',workspace/'src');os.chmod(workspace,0o755)
        fixtures=ROOT/'.data/p2-http-fixtures';fixtures.mkdir(exist_ok=True)
        fixture_names=['user.json','tag.json','link.json','untagged-link.json','links.json','not-found.json']
        for fixture in fixture_names:
            target=fixtures/fixture;target.write_text('');os.chmod(target,0o666)
        dsn='postgres://academy:'+password+'@127.0.0.1:5432/linkhub?sslmode=disable'
        run(['podman','run',*policy,'--network=container:'+name,'--user=65532:65532','--memory=1536m','--memory-swap=1536m','--tmpfs=/tmp:rw,noexec,nosuid,nodev,size=128m','--tmpfs=/work:rw,nosuid,nodev,size=512m','--tmpfs=/cache:rw,nosuid,nodev,size=512m','--workdir=/work','-v',str(workspace/'src')+':/source:ro,Z','-v',str(cache)+':/gomod:ro,Z','-v',str(fixtures)+':/fixtures:rw,Z','-e','LINKHUB_RESPONSE_FIXTURE_DIR=/fixtures','-e','GOMODCACHE=/gomod','-e','GOCACHE=/cache','-e','GOTMPDIR=/work/build','-e','HOME=/tmp','-e','GOMAXPROCS=2','-e','GOPROXY=off','-e','GOSUMDB=off','-e','GOTOOLCHAIN=local','-e','CGO_ENABLED=0','-e','LINKHUB_TEST_DATABASE_URL='+dsn,GO,'sh','-c','cp -r /source/. /work/ && mkdir -p /work/build && go test -mod=readonly -count=1 -timeout=120s -v ./...'],timeout=180)
    for fixture in fixture_names:os.chmod(fixtures/fixture,0o600)
    print('PASS: trusted reference + actual PostgreSQL ownership/transactions/migrations/single-connection pool tests; no host network/ports',flush=True)
except Exception as failure:
    print('FAIL:',str(failure).replace(password,'[redacted]'),flush=True)
    raise SystemExit(1) from None
finally:
    subprocess.run(['podman','rm','--force',name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
