# ADR-0002 — локальный исполнитель проектов с PostgreSQL

Дата: 2026-10-15. Статус: решение принято, реализация/проверка M10 в работе. Публичный multi-tenant execution по-прежнему запрещён. Доверенные reference-тесты уже проверены с настоящим PostgreSQL, это ещё не выполнение кода ученика.

## Граница и протокол

Отдельный процесс `cmd/project-runner`, Unix0600 `.data/project-runner.sock`, приватная папка0700. API только сохраняет исходники и вызывает worker: никаких user-source subprocess на host. `Request{project_id:"p2",action:"run"|"test"|"submit",files:map[string]string}`; `runner.Result` обратно. Strict JSON,4MiB wire,48 файлов/512KiBtotal/256KiBGo/128KiBperfile; имена safeASCII,≤3сегментов, без absolute/../backslash/NUL; allowlistedextensions. go.mod/go.sum точно совпадают с bundledpolicy. Dockerfile/Compose/env пользователя никогда не исполняются исполнителем.

Только immutable OCIrefs: registry@sha256digest либо локальныйsha256:imageID. Metadata `.data/p2-runtime/config.json`0700parent/0600file содержит go_image/checker_image/postgres_image, выбранныйGo1.27.2 и hashes policy. Browser не выбирает image/config/path. Операторские go.mod/go.sum рядом. Worker Config включает Socket,GoImage,CheckerImage,PostgresImage,ExpectedGoMod,ExpectedGoSum. Missingimage/policy=>unavailable, neverpull/unsafe fallback.

## Контейнеры на один job

1. Ephemeral rootless Postgres18.6 pinned digest9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d; networknone, UID999, readonlyrootfs, CPU1/memory512MiB=swaplimit/PID128/shm64, /tmp64noexec, PGDATA384MiBtmpfs и socket16MiBtmpfs mode1777. PGDATA18 `/var/lib/postgresql/18/docker`. Файловая система пользователя туда не монтируется.
2. Go server container с `network=container:DB`: только тот же loopback namespace, без host ports/Internet/hostnetwork. UID65532, readonlyrootfs, capdropALL/NNP/http-proxyfalse, CPU1/memory1536MiB=swaplimit/PID128; /tmp128noexec, /work512exec, /cache512exec. Source передаётся tar stdin, без host mounts. GoSDK с predownloaded public pinned pgx dependencies; GOPROXYoff/GOSUMDBoff/CGO0/GOTOOLCHAINlocal, GOMODCACHE=/opt/academy/gomod, GOCACHE=/cache, GOTMPDIR=/work/build. Копировать публичный prewarmcache `/opt/academy/gocache` в scratchcache. Prewarmstub импортирует только зависимости/stdlib, не references/tests/answers. Статический shell исполняет только фиксированные tar/go build/exec; source не shell.
3. Trusted checker — ДРУГОЙ image/container (путь бинарника `/opt/academy/checker`), тот же DBnetns, readonly/UID65532/capsdrop/NNP, CPU1/memory768MiB=swaplimit/PID128; scratch/tmp bounds. Внутри studentimage нет checkerbinary/source/expectedanswers. Checker HTTPclient stdlib, не слушает порты. Его stdout не смешивается с stdout сервера ученика.

Bootstrap DB password randomperjob остаётся у worker/DB. Создать отдельную limited app role LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION, дать нужные DB/schema права. Только limitedDSN передаётся userprocess. Никаких session.secret/issuer.key/реальнойacademy.db/hostmodulescache/runtimeSockets. Credentials ephemeral; точные значения фильтруются из диагностики.

## Проверка и рестарт

`run`: сборка/старт и публичный smoke `/healthz`; actual bounded diagnostics, no XP. `test`: student Go tests внутри sandbox с privateDB, requiregenuineTestX/noTestMain; их результат advisory, не автоматическийgrade.

`submit`: отдельный trustedchecker проверяет HTTPcontract из OpenAPI/reference: регистрация/login/CSRF/me, tags/links, wrongowner404, malformedinput, redirect/nooutboundfetch, logout/revocation. Число auth-запросов учитывает встроенныйratebudget. Phase1 выводит `{checks:[{name,passed}],state:opaque}`. Worker хранит State только внутри job, не возвращает API. Остановить первый usercontainer и создать НОВЫЙ (свежие workspace/tmpfs) при том же DB; Phase2 с сохранённым checkerstate проверяет login и сохранённую ссылку/slug. Так память/файл первого контейнера не подменяют DBpersistence. Дополнительно проверить requiredSQLtables через trustedDBpsql. Checker JSON строго bounded32KiB; verdict только по непустому набору trustedchecks. Studentstdout/selftests не могут присвоитьpass. В privategrade runtimestdout/stderr скрыты, compilerdiagnostics отдельно допустимы; API не получает checkerstate/passwords/privateHTTPbodies.

## Lifecycle и доказательства

Concurrency1, busyfail без очереди, общийdeadline180s, cleanup20s с независимымcontext после отмены. Все owned контейнеры job удаляются, включая checker и оба server; verifycleanup. Failurecleanup latchesworker unavailable, passed=false. Стабильные labels `academy.project-runner=true` +installID SHA256(canonicalsocket) и jobID; orphanreconcile при старте только своегоscope. Сначала получить exclusiveSocketlock/listener и только затем New/preflight/reconcile: второй живой worker не удаляет его контейнеры. Socketstale recovery проверяеттип/lock, не удаляет чужой regularfile. Handler shutdown ограничен.

Нужны unitprotocol/path/quota/lifecycle/cleanup/cancel, actualnative starterfail/referencepass/DBpersistence/compileerror/privateoutputmask и boundednegativecases. Reference-only script `p2-reference-check.py` уже подтвердил DBownership/tagtransaction/migration/single-pool; fullHTTP+DB gate расширяется отдельно. Этот ADR не утверждает готовность нового worker до его actualtests и независимогоreview. Kernel/container runtime остаются trustboundary; publichosting не разрешён.
