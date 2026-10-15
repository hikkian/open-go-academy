# Независимое ревью project runner P2

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`. Scope: ADR0002, `internal/projectrunner/{engine,types,service}`, checker и `cmd/project-runner`. Только статический анализ и безопасные unit fixtures; враждебный код/PoC не запускался. Reviewer владеет только этим отчётом и не меняет production files. Native execution/image build/integration выполняет родитель и не считаются выполненными reviewer.

## Текущий verdict

**PR-R1–R7 и PR-L1 исправлены и CLOSED; открытых High/Medium в проверенной области нет.** Reviewer прочитал native44-check reference run/test/submit/fresh-app persistence/cleanup, SCRAM audit и disconnect/SIGKILL recovery PASS. Финальный limited-role SQL guard принят после статического чтения и focused regression; предоставленный родителем повтор actual44-check также PASS. Quoted psql variables корректны; чтение student данных больше не получает superuser authority.

## Контекст и доверие

API передаёт только bounded source data через приватный Unix socket. Worker — единственный host-процесс, вызывающий Podman; Go source запускается в отдельном rootless container, не на API host. На job создаётся ephemeral PostgreSQL netns, student container входит только в него. Checker — отдельный immutable image/container, собственный stdout и приватный state; source/expected answers checker не копируются в student image. Phase2 использует свежий student container при прежнем DB, поэтому собственная память/tmpfs первого приложения не доказывают persistence.

Спецификации runtime images/module policy и оператор ОС доверенные. User source/files/stdout/HTTP ответы недоверенные. Критичны: ограничение DB роли, доверенность checker output, cleanup до возврата Passed, отсутствие host mounts/ports/Internet, lock до orphan reconciliation.

## PR-R1 — High, CLOSED по исправлению конфигурации: loopback PostgreSQL auth

**Путь:** engine Execute, DB run/bootstrap args.

Первоначально bootstrap и app использовали один password; это уже разделено. Но без явного initdb host auth различие паролей не обеспечивает роль: initdb по умолчанию использует trust, а official image дописывает общий host rule в конец pg_hba.conf, не заменяя ранее созданные loopback rules. Student находится в том же network namespace и использует127.0.0.1. Это противоречит модели limited LOGIN role.

Основание — статическая конфигурация плюс первичные источники: [PostgreSQL18 initdb](https://www.postgresql.org/docs/18/app-initdb.html) и [official18 entrypoint](https://raw.githubusercontent.com/docker-library/postgres/master/18/bookworm/docker-entrypoint.sh). Фактическую возможность входа под superuser reviewer не проверял и не заявляет как исполненный exploit.

**Минимум:** явный POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256 (Unix socket остаётся только DB container; local trust может обслуживать fixed bootstrap). Проверить реальный pg_hba_file_rules обычным read-only конфигурационным запросом; localhost/IPv6 TCP не должны иметь trust. Bootstrap/app passwords разные, app role NOSUPERUSER/NOCREATEDB/NOCREATEROLE/NOREPLICATION. Native gate должен подтвердить эту границу без враждебной программы.

**Повторная проверка:** DB run теперь явно передаёт POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256, bootstrap/app secrets различны; приложению выдаётся только limited DSN. Unix socket не монтируется в app. Fixed bootstrap использует собственный DB-container local socket. Причина конфигурационного дефекта устранена; CLOSED. Реальный pg_hba_file_rules audit reviewer не выполнял; остаётся требованием native gate родителя.

**Позже проверенные evidence:** `.data/p2-pg-hba-audit.log` содержит actual disposable worker DB rules для127.0.0.1,::1 иall — scram-sha-256, host trust не обнаружен. Audit выполнял родитель, reviewer прочитал результат; требование configuration evidence закрыто.

## PR-R2 — Medium, CLOSED по исправлению транспорта: tar stdin

**Путь:** runStudent.

Исходный cmd читает tar из stdin, но серверный режим использует Podman run -d -i. [Официальная документация Podman](https://docs.podman.io/en/latest/markdown/podman-run.1.html#interactive-i) различает attached stdin pipe и detached stdin, ожидающий последующего attach. Следовательно, присвоить exec.Command.Stdin tar недостаточно для гарантированной передачи данных detached container. Fake CLI принимает input bytes и этого поведения runtime не проверяет.

**Минимум:** отделить bounded tar transfer от detached service lifetime: передать архив явным attached exec/cp в существующий sandbox и запускать fixed build/start после успешного переноса. Никаких host source mounts или shell interpolation source. Нужен native positive trusted-reference run/submit и обычный fixture для последовательности transfer→start. Reviewer runtime probe не выполнял.

**Повторная проверка:** detached container запускает только fixed waiter, затем worker последовательно делает exec mkdir/cache-prewarm, attached `exec -i ... tar -xf - -C /work/src` с bounded archive и только после успеха touch READY. Waiter начинает fixed Go build/exec после marker. Test action запускается attached exec отдельно и остаётся advisory. Исходный detached stdin больше не используется; host source mounts не добавлены. CLOSED статически; actual byte transfer подтверждает native gate родителя.

## PR-R3 — Medium, CLOSED: readiness/checker lifecycle

Первоначально checker health при retry имел фиксированное имя и не удалялся; первый not-ready оставлял exited container, второй run сталкивался с именем. Отдельно fresh app2 запускал persistence до окончания build/start.

**Минимум:** readiness обеих app phases; обычные fake fixtures notready→ready с уникальными/reusable checker names, корректным timeout и phase2 ordering. Health Run не присваивает grade: Passed всегдаfalse.

**Повторная проверка:** checker names получают atomic sequence, attached checker run использует --rm. waitReady вызывается для app и app2; phase2 persistence только после второго успеха. Unit readiness требует три health attempts до ready и отвергает trailing checker JSON, submit lifecycle fixture проходит. Run возвращает Passedfalse. CLOSED; реальный cold startup/restart reviewer не запускал.

## PR-R4 — Medium, CLOSED: runtime no-pull policy

Preflight image exists и immutable digest полезны, но не заменяют --pull=never: runtime configuration либо удаление cached image между проверкой/run может вызвать host registry request. ADR прямо обещает unavailable вместо pull. Добавить --pull=never в DB/Go/checker run args; unit fixture должен проверять каждый создаваемый контейнер.

**Повторная проверка:** все три runtime run paths содержат --pull=never. CLOSED. Image build — отдельное явно операторское действие, не user-job fallback.

## PR-R5 — Medium, CLOSED: cmd менял права существующей socket directory

**Путь:** cmd/project-runner, prepare parent.

MkdirAll(...,0700) сопровождался безусловным Chmod(parent,0700). Обычная настройка --socket /tmp/project-runner.sock меняет права существующего /tmp, а socket внутри workspace ограничивает весь workspace. Reviewer такую команду не запускал. Это ошибочная host mutation, не требуемая для приватности собственного socket.

**Минимум:** создавать отдельный новый private parent либо требовать уже private existing parent; существующую nonprivate directory отклонять без изменения. Unit fixture должен сохранять исходные permissions/sentinel. Socket/lock acquisition остаётся до New/preflight/reconcile.

**Повторная проверка:** ensurePrivateSocketDir создаёт только новый parent0700; существующий parent проверяет через Lstat и требует directory+точно0700, не Chmod. TestEnsurePrivateSocketDir проверяет создание и отказ для0755 с неизменёнными правами. cmd suite PASS0.002s. CLOSED.

## PR-R6 — Medium, CLOSED: checker security coverage

В прочитанном initial checker проверял missing CSRF, но не wrong nonempty token/foreign Origin; cookie HttpOnly/SameSite attributes не утверждались. Tags покрывались create/list, не rename/delete ownership; User ID/role валидировались неполно. Это обычные неполные реализации HTTP-контракта, которые автопроверка должна либо отклонять, либо явно оставлять manual-unverified. Отдельный checker container предотвращает подделку его stdout, но сам по себе не доказывает полноту checked criteria.

**Минимум:** добавить требуемые security checks/обычные mock negative fixtures либо исключить эти criteria из automatic claim и явно оставить в rubric для ручного review. Родитель сообщил, что дополняет checker после native phase; до повторного чтения finding pending.

**Повторный узкий проход:** добавлены expected403 для wrong nonempty CSRF и foreign Origin, проверка register user UUID/role=user, session HttpOnly/Path/SameSite и matching readable CSRF cookie. Disposable tag проходит rename/delete и owner-negative update/delete, исходный tag сохраняется для persistence. Все эти assertions изучены; ordinary mock LinkHub fixture обновлён и focused checker unit PASS0.012s. Это корректно устраняет ранее перечисленную отсутствующую проверку, а не доказывает все возможные auth реализации.

**Финальная проверка:** initial теперь вызывает rejectedLogin с wrong password и неизвестным email; только401 считается успешным отрицательным check. TestInvalidLoginMustBeRejected проверяет401/200/500 на обычном HTTP fixture; положительный mock flow сохранён. Focused projectrunner/checker suite PASS0.510/0.013s. Прочитан `.data/p2-worker-native-persistence.log`: оба401 checks и остальные расширенные assertions имеют PASS в actual44-check run. PR-R6 CLOSED. Cookie flags утверждаются на регистрации; HTTPS Secure configuration остаётся ручной проверкой за пределами HTTP-loopback checker. SameSite checker принимает Lax или Strict вместе с CSRF/Origin guards; полнота всех auth реализаций не заявляется.

## PR-R7 — Medium, CLOSED: authority нового SQL persistence guard

**Путь:** engine.persistedFixtureCheck/persistedFixtureSQL.

Новый guard полезно проверяет реальный join users→links→link_tags→tags с matching owner, ID/slug/target/tag/enabled и наличие активной session того же пользователя. Значения передаются отдельными `psql -v name=value` argv и используются только как quoted literals `:'name'` в фиксированном SQL через stdin/-f-. Shell не участвует; checker password не передаётся. Это корректная защита от интерполяции значений.

Однако запрос запускается psql -U postgres и обращается к users/links/... без public qualification. App role владеет этой DB/schema и может менять database search_path и объекты. Доверенная проверка не должна исполнять выбранные студентом views/functions с полномочиями superuser. Это статический trust-boundary finding; reviewer не создавал такие объекты и не запускал попытку эксплуатации.

**Минимальная рекомендация:** выполнять content query ограниченной app/checker-read ролью с фиксированным search_path и public-qualified relations; при утверждении именно таблиц проверить relkind через pg_catalog. Metadata query к pg_catalog можно отделить от чтения student data. Query timeout/read-only предпочтительны как bounds. Ordinary fixtures должны проверять role/SQL/schema flags, native reference остаётся положительным regression.

**Финальная повторная проверка:** persistedFixtureCheck запускается под `student_<job>`, SQL устанавливает search_path=pg_catalog и квалифицирует public.users/links/link_tags/tags/sessions; metadata использует ту же limited роль и pg_catalog.to_regclass с фиксированными public names. Superuser bootstrap остаётся только до user execution. Quoted `:'name'` variables и отсутствие checker password сохранены; -qtA подавляет SET command tag, итог требует ровноt. Regression утверждает limited -U/no postgres/trusted path/public-qualified data. Reviewer выполнил только `go test -count=1 -timeout=30s ./internal/projectrunner -run TestPersistedFixtureQueryRequiresStoredRowsAndDoesNotSendPassword`: PASS0.002s. CLOSED. Metadata доказывает наличие relations, не полный schema/constraints/relkind audit; качество схемы остаётся отдельным review.

## PR-L1 — Low, CLOSED: exact ephemeral credential redaction

Private submit output скрыт, но run/test diagnostics в последнем снимке использовали только bounded(text), без замены точных generated appPass/bootstrapPass/DSN, хотя ADR обещает фильтрацию. Это ephemeral credentials этого же job, которые приложение уже знает; утечки issuer/session/реальной DB здесь нет. Минимум central redaction перед result/HTTP и безопасный fake diagnostic fixture.

**Повторная проверка:** bounded(redact(...)) применяется к success test output, readiness diagnostics и отдельно исправленной failing runStudent/test ветке. TestFailedStudentTestsRedactBothDatabaseCredentials проверяет обычный fake error output с ephemeral credentials; suite PASS. CLOSED.

## Уже исправленные и проверенные части

- Regex genuineTest/TestMain заменены AST parser: block comments не создают TestX, импорт testing/alias и *T signature проверяются, TestMain с комментариями отвергается. Test action advisory и Passedfalse.
- CombinedOutput заменён bounded writer+cancel; mutex защищает общий stdout/stderr buffer. Unit TestBoundedCommandOutputCancelsAtLimit PASS. Go/DB logs имеют max-size32kb, resource/quota policies не берут параметры из браузера.
- Unsupported tmpfs uid/gid/userns flags удалены; PostgreSQL18 path покрывается /var/lib/postgresql tmpfs. Proxy false присутствует у контейнеров, memory-swap==memory, capsdrop/NNP/read-only, fixed nonroot users, CPU/PIDs/tmpfs bounds.
- Cleanup использует независимый20s context, scoped install+job labels, пытается удалить owned containers и повторно проверяет отсутствие. Failure poisons worker и меняет Passedfalse/statuserror до результата. Отменённый caller не отменяет cleanup context.
- Canonical socket directory используется для install hash; cmd получает listener/lock до New и orphan reconciliation. Service строго декодирует request и EOF. Checker output bounded32KiB, unknown/trailing JSON отвергается; state не входит в runner.Result.
- Client response теперь ограничен1MiB+1 для detection overflow и требует EOF после результата; обычный Unix HTTP fixture проверяет trailing/oversized replies. Ранее замеченный Low protocol gap устранён.
- Общий job180s+cleanup20s помещается в client220s/serverwrite225s. Concurrency1 без очереди, busy отказ. Cached immutable images и rootless cgroupv2 обязательны; unsafe host fallback отсутствует.
- Files:48/512KiBtotal/256KiBaggregateGo/128KiBperfile, safeASCII parts≤3, regular tar entries только из map, immutable exact root go.mod/go.sum. Пользовательский Dockerfile не исполняется; fixed shell command не интерполирует source.

## Фактически выполнено

```sh
.tools/go/bin/go test -count=1 -timeout=30s ./internal/projectrunner ./internal/projectrunner/checker ./cmd/project-runner
```

PASS: projectrunner0.505s, checker0.009s; cmd успешно собран, своих tests нет. Fixtures используют fake Podman CLI и mock HTTP contract, не student/native execution. Проверены request/path/module bounds, listener lock/sentinel, strict JSON, cancel/cleanup poisoning, output cap, readiness, private state masking, target allowlist, bounded HTTP response/no redirects и mock initial/persistence.

**Повтор после всех engine/types/service/cmd fixes:** та же команда PASS: projectrunner0.509s, checker0.010s, cmd/project-runner0.002s. cmd теперь содержит безопасный directory unit fixture. Дополнительно read-only просмотрен `scripts/p2-runtime-build.py`: SDK копирует только locked module policy и публичный warm stub/dependency cache, checker image получает отдельный бинарник; reference/student source и checker answers не копируются в SDK. Скрипт reviewer не запускал. Сообщённые builder vet/race не выдаются за самостоятельно выполненные reviewer.

**Повтор только expanded checker**, фактически выполненный reviewer:

```sh
.tools/go/bin/go test -count=1 -timeout=30s ./internal/projectrunner/checker
```

PASS0.012s. Прочитаны новые assertions и ordinary mock fixture; negative target/size/JSON/cookie-auth request cases остаются локальными HTTP fixtures, враждебные программы не запускались.

## Native evidence, предоставленные родителем

Reviewer read-only прочитал `.data/p2-worker-native.log`: actual isolated trusted reference Run — statusok/Passedfalse/healthztrue; Test — Go testsok/Passedfalse; Submit — Passedtrue и33 checks, включая свежий app container, persistence.login/link/redirect, пять SQL tables. Runtime stdout/stderr submit пусты; завершающий log сообщает cleanup и отсутствие host ports. Это подтверждает source transfer/readiness/restart в реальном trusted-reference сценарии после engine fixes, но не является полной negative sandbox/security matrix.

Отдельно прочитан `.data/p2-pg-hba-audit.log`, описанный у PR-R1. Исполнение обоих native сценариев принадлежит родителю. Лог33-check checker предшествует новым PR-R6 assertions; native успех расширенной версии и отрицательных401 сценариев ещё не заявляется. Reviewer завершает узкий проход, не ожидая этот будущий прогон.

**Позднее предоставленные финальные evidence:** `.data/p2-worker-native-persistence.log` содержит actual44-check PASS, включая wrong-password/unknown-user401, cookie/CSRF/Origin/tag assertions, fresh app persistence и новый PostgreSQL join/session check. Run/Test имеют Passedfalse; submit stdout/stderr пусты. `.data/p2-worker-recovery.log` подтверждает disconnect cancellation+owned-container removal и actual SIGKILL/stale socket/scoped orphan recovery PASS. Reviewer прочитал эти файлы, native исполнение не повторял.

Новый TestPersistedFixtureQueryRequiresStoredRowsAndDoesNotSendPassword прочитан: false DB result не проходит, true проходит, invalid/unknown state не достигает psql, query не содержит fixture email value, argv не содержит checker password. Фактически выполнен focused suite projectrunner/checker после этих изменений: PASS0.510/0.013s. Эти результаты подтверждают обычный положительный persistence fixture и протокол; не закрывают PR-R7 authority/search_path.

**Финальный limited-role native repeat:** reviewer прочитал `.data/p2-worker-native-limited-role.log`: actual Run/Test Passedfalse, Submit все44 checks Passedtrue, включая PostgreSQL fixture join/session и свежий app persistence; cleanup/nohostports PASS. Это последующий положительный regression после закрытия PR-R7. waitDB теперь ждёт TCP127.0.0.1 вместо временного init Unix server; код прочитан. Интегратор сообщил о transient bootstrap failure до этой правки, успешный repeat не выдаётся за успех первой попытки. Самостоятельное native выполнение reviewer не заявляется.

Reviewer самостоятельно не запускал actual worker с пользовательским/враждебным кодом, native Podman integration, image dependency build или DB role login probes. Предоставленные родителем native33/44-check, pg_hba и recovery results проверены чтением, их пределы указаны выше. Public multi-tenant/hosted execution по-прежнему запрещён; общий kernel/container runtime остаётся trust boundary.
