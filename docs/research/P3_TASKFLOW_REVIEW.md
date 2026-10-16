# P3 TaskFlow: статический обзор reference и учебного контракта

Дата: 2026-10-16. Объём: `content/projects/p3/reference/**`, starter, README, OpenAPI, rubric, threat-model, manifest и фиксированный Compose. Реализация не изменялась. Рецензент **не запускал тесты, runtime, Go-код, контейнеры, PostgreSQL, submissions или сканеры**. Ниже — статически выводимые сценарии; предложенные regression tests ещё требуется выполнить отдельно. Обзор не является подтверждением публичной multi-tenant безопасности.

Итог финального статического recheck: P1 не найдено. Все шесть P2 исправлены в просмотренных исходниках, включая отдельный residual P2-4 terminal cleanup. Разделы ниже сохраняют snapshot обнаружения и историю исправлений; актуальное закрытие residual приведено в конце. Runtime-проверки этим отчётом не подтверждаются; новые PostgreSQL regressions остаются not run.

## P2-1. Worker.Run не дожидается работников: канал завершения используется в обратном направлении

Пути: `reference/worker.go:30`, `:33`, `:35`, `:56`; вызывающий код `reference/main.go:70`.

`Run` сначала кладёт в буфер `concurrency` токенов. Затем и каждая завершающаяся goroutine, и сам `Run` после отмены пытаются **получать** из этого канала. Новых отправок больше нет. Итого на N токенов приходится 2N получений. Это не join.

Контрпример для N=2 после отмены: если оба работника успевают выполнить deferred receive первыми, они съедают оба токена и `Run` навсегда блокируется на первом `<-done`. `main` ждёт `<-workerDone` без тайм-аута и не доходит до 10-секундного Drain. Если оба токена получает `Run`, он возвращается до завершения работников, а работники затем застревают в своих defer. При смешанном порядке также возможна вечная блокировка. Deadline самого Drain не исправляет ожидание, происходящее до него.

Минимальная рекомендация: использовать `sync.WaitGroup` либо пустой completion-channel с отправкой одним токеном при реальном завершении каждого работника; согласовать внешний deadline остановки. Добавить управляемый cancellation/join test, проверяющий отсутствие раннего return и зависания при разных порядках завершения. В поставляемых тестах `Run` вообще не вызывается: `TestWorkerLimitsArePositive` проверяет лишь нормализацию чисел, интеграция вызывает `processOne` и `Drain` напрямую.

## P2-2. CreateTask отделяет проверку membership от изменяющей SQL-операции

Пути: `reference/store.go:157`, `:164`, `:183`; обещание `starter/README.md:14`, `threat-model.md:13`.

Сначала отдельный `SELECT EXISTS` проверяет owner/editor. Позже `INSERT INTO tasks` использует только `project_id`, без membership/actor predicate и без блокировки прочитанного membership. Обычный `db.Begin(ctx)` не задаёт более сильную изоляцию или блокировку строки прав. Сам факт нахождения statements в одной транзакции не делает проверку прав неизменной.

Контрпример: E имеет роль editor; транзакция E читает EXISTS=true; владелец завершает смену E на viewer через `SetMemberRole`; только затем транзакция E выполняет INSERT и фиксирует task+outbox. На момент изменяющей операции E уже viewer, но запрета нет. Остальные записи, например `AddComment`, помещают membership predicate в сам INSERT SELECT; `CreateTask` нарушает явно требуемую same-operation границу.

Минимальная рекомендация: проверять actor/role в самом `INSERT ... SELECT` и корректно разбирать zero rows либо удерживать согласованную блокировку membership до конца операции, определив семантику revocation. Проверить управляемое interleaving «проверка → downgrade commit → insert». Одновременно сохранить атомарность task+event и поведение конкурентного replay.

## P2-3. Глобальный idempotency key связывает независимые tenant и позволяет резервировать чужие ключи

Пути: `reference/migrations/0001_init.up.sql:28`; `reference/store.go:172`, `:188`, `:197`; контракт `starter/README.md:16`.

`outbox.idempotency_key` имеет глобальный UNIQUE, поиск replay выполняется только по key; `project_id` находится внутри payload и проверяется лишь после чтения найденного события. Поэтому ключ не изолирован project/workspace.

Контрпример: пользователь A создаёт задачу в своём проекте с key `create-task-1`. Независимый пользователь B имеет права в другом проекте и посылает свою первую задачу с тем же key. Вместо независимого создания получает `ErrDenied`/404, поскольку найден чужой project_id. A может заранее занимать предсказуемые клиентские ключи, мешая другим tenant. Чужие данные в ответ не возвращаются, но существует межтенантное влияние на доступность/корректность. Даже два честных клиента из примеров с одинаковыми счётчиками сталкиваются.

Минимальная рекомендация: явно хранить project scope и уникальность `(project_id, idempotency_key)` (при необходимости actor scope), использовать ту же пару в lookup и conflict target. Добавить cross-project same-key test одновременно с existing concurrent same-project replay test. Если глобальная уникальность задумана намеренно, текущий API должен явно определить namespace и ошибку collision; сейчас требование project-local поведения и описание tenant boundaries этого не объясняют.

## P2-4. Устаревший работник может отменить терминальное состояние после передачи lease

Пути: `reference/worker.go:87`, `:94`, `:115`, `:121`, `:127`.

Claim сохраняет `locked_at` и attempts, но не выдаёт поколение/владельца lease. Последующие done/fail обновляют строку только по `id`; ни актуальность lease, ни state, ни попытка в WHERE не проверяются. `fail` решает pending/dead по локальному старому значению attempt.

Контрпример в уже предусмотренном контрактом invalid-event сценарии: maxAttempts=2. A захватывает unsupported kind с attempt=1 и приостанавливается до вызова fail больше чем на 30 секунд. B reclaim-ит ту же строку, получает attempt=2, fail переводит её в dead. После этого A продолжает и `fail(id,1,...)` безусловно возвращает строку в pending, стирая чужой locked_at и возобновляя терминальную работу. Сохранённый attempts=2 этому не препятствует: SELECT claim не ограничивает attempts. Уникальный receipt защищает от дублирования локального успешного эффекта, но не защищает эти переходы состояния.

Минимальная рекомендация: использовать уникальный lease token/поколение claim и условные finish/fail transitions только для актуального владельца; определять terminal bound по актуальной попытке. Stale worker должен получить no-op/lost-lease и не менять новую попытку. Добавить управляемый stale-A/reclaimed-B interleaving и проверить, что dead/done не возвращаются в pending старым работником. Поставляемая интеграция проверяет два последовательных отказа, но не этот сценарий.

## P2-5. Logout подтверждает отзыв сессии даже при ошибке DELETE

Пути: `reference/api.go:177`, `:180`, `:183`; обещание `starter/README.md:13`, `openapi.yaml` DELETE sessions/current.

Ошибка `db.Exec(DELETE sessions...)` полностью игнорируется. Обработчик всегда очищает browser cookie и отвечает 204 «Session revoked».

Контрпример: transient database/network failure во время DELETE. Клиент получает 204, но строка sessions остаётся; сохранённая копия того же bearer token после восстановления БД по-прежнему проходит `identity` до истечения 24 часов. Локальное удаление cookie не равно server-side revocation.

Минимальная рекомендация: очищать локальный cookie, но подтверждать server-side отзыв только после успешного DELETE; на сбой возвращать безопасный retryable 5xx без SQL-диагностики. Отсутствующая/уже удалённая сессия может оставаться идемпотентным 204. Добавить failure-path тест, проверяющий ответ и живучесть токена. В предоставленных тестах нет logout/revocation сценария.

## P2-6. Readiness обещана контрактом, но endpoint всегда сообщает готовность

Пути: `reference/api.go:26`, `openapi.yaml:15`, `starter/README.md:19`.

Единственный health endpoint безусловно возвращает 200 status=ok. Отдельной readiness-проверки нет, а OpenAPI описывает 200 как «Process is ready for local requests». Стартовый Ping не проверяет зависимость после запуска.

Контрпример: PostgreSQL становится недоступен после успешного старта. Register/login/tenant операции неработоспособны, но `/healthz` продолжает 200 без обращения к store. Это пригодно как liveness, но не выполняет требование health/readiness и вводит в заблуждение failure/load evidence.

Минимальная рекомендация: назвать healthz liveness и предоставить bounded readiness (например отдельный `/readyz` с коротким Ping context и safe 503), учитывая draining состояние. Либо явно сузить учебный контракт, не заявляя реализованную readiness. Добавить dependency-down/readiness/drain проверки; не выдавать существующий HTTP 200 за них.

## Статически наблюдаемые положительные границы и оговорки

- Actor берётся из server-side session; auth не доверяет user/role полям JSON. Session token создаётся 32 случайными байтами, в БД хранится SHA-256; salted PBKDF2 и constant-time compare видны в store. Cookies HttpOnly/Secure/Strict; отсутствие TLS для classroom HTTP честно описано. Это наблюдение исходника, не криптографическая или browser-runtime сертификация.
- Data SQL использует параметры; строковая конкатенация в integration test строит только имя random schema из hex UUID, не из клиента. Tenant joins и role predicates есть у остальных project/task/comment методов. Во время обзора не найден конкретный прямой SQL injection или чтение чужой строки по подменённому UUID.
- Task и outbox действительно входят в одну транзакцию; replay collision cleanup также происходит до commit. Receipt и done коммитятся одной apply-транзакцией, receipt имеет UNIQUE/PK outbox_id. Это ограниченная локальная демонстрация эффекта, а не exactly-once для внешнего сервиса.
- HTTP ограничивает headers/timeouts; API накладывает MaxBytesReader, строгие JSON-поля, длины полей, bounded lists и per-IP map. Forwarded headers не используются как authority. Прямой SQL/DSN/password из ошибок не возвращается в JSON. Эти контролы не заменяют перечисленные race/failure fixes.
- Compose используется только фиксированный и loopback; API контейнер имеет non-root user, read-only filesystem и cap_drop. DB least privilege, TLS, migration coordination и защита hostile same-site origin прямо исключены threat-model, поэтому отсутствие этих исключённых механизмов не представляется здесь новым finding.

## Starter / reference / test contract

Starter честно содержит только health endpoint и не выдаётся за готовое решение. Проектные требования, роль владельца/редактора/читателя и ограничение claims достаточно явно описаны; README, rubric и OpenAPI не заявляют профессиональный грейд или production security.

Однако маркировка reference как complete должна оставаться ограниченной до исправления findings. C3 требует worker crash/reclaim и shutdown; поставляемая интеграция вызывает processOne конкурентно, Drain, два последовательных отказа и migration rollback/up, но не вызывает Run, не моделирует stale lease owner и не проверяет logout failure/readiness. Отдельный внешний checker в этот обзор не входил: отсутствие сценария в двух test-файлах не означает, что он отсутствует во всём репозитории. Нельзя приписывать этим исходникам или этому статическому review выполненное acceptance/load/failure доказательство.

Перед приёмкой необходимы точечные regression tests по шести сценариям и существующие unit/integration/vet/race/build проверки в явно disposable среде. Все runtime результаты в этом отчёте остаются **not run by reviewer**.

## Повторная проверка шести исправлений

Дата: 2026-10-16. Проверены текущие исходники тех же путей и новые regression-test sources. Никакие тесты, DB/containers, API или Go-программы рецензентом не запускались. Автор сообщил об успешных no-DB unit/race/build проверках; отдельных P3 validation-log файлов в просмотренных `docs/validation`/`.data` не обнаружено. Поэтому ниже `исправлен` означает подтверждённое статическое изменение, а не независимо выполненный runtime PASS.

| Finding | Статус | Непосредственно просмотренное свидетельство |
|---|---|---|
| P2-1 worker join/drain | Исправлен статически | `worker.go:53` использует WaitGroup: Add до spawn, deferred Done, Wait после ctx cancellation. `main.go:73` задаёт 5-секундный join; Drain начинается только при joined=true и также имеет 5-секундный context. `TestRunWorkersJoinsInflightWork` удерживает in-flight callback до release и запрещает ранний return; `TestWaitForWorkerHonorsShutdownDeadline` проверяет deadline. |
| P2-2 role revocation | Исправлен статически | `store.go:174` читает membership `FOR SHARE OF m` в той же tx, которая затем INSERT/commit; до commit lock не освобождается. В `:196` сам INSERT SELECT также включает actor/owner-editor predicate. Downgrade должен сериализоваться с lock. Integration-test source содержит управляемый afterAuthorization и попытку UPDATE роли с lock_timeout; запуск не выполнен рецензентом. |
| P2-3 idempotency scope | Исправлен статически | Migration объявляет `project_id` FK и UNIQUE(project_id,idempotency_key). Lookup, replay после conflict и conflict target в `store.go:188`, `:204`, `:213` используют одну project/key пару. Integration source создаёт задачи в двух проектах с одинаковым key. |
| P2-4 lease/retry | **Исправлен статически, включая residual** | Каждое claim получает свежий lease UUID. Done и fail условны по актуальному token+processing; zero-row done откатывает receipt. Terminal decision использует persisted attempts. В финальном no-row branch claimOne теперь Commit сохраняет expired-final cleanup; ошибка Commit возвращается вызывающему. Добавлен PG regression с единственной expired final-attempt строкой. Запуск PG не выполнен. |
| P2-5 logout | Исправлен статически | `Store.RevokeSession` возвращает DELETE error. `api.go:192` при nil revoke/error очищает browser cookie, но отвечает safe 503; 204 только после успешного revoke либо отсутствия cookie. OpenAPI включает 503 и объясняет, что cookie clear не означает revocation. Новый unit source проверяет 503/cookie/error hygiene; integration source различает failure и успешный отзыв токена. |
| P2-6 readiness | Исправлен статически | `/livez` отделён от `/readyz`. `api.go:212` создаёт 500-ms context для DB Ping и требует Worker.Ready; running/draining — atomic флаги. `main.go` передаёт worker в API. README/OpenAPI/starter contract согласованы с новыми маршрутами. Unit source содержит DB-down, no-worker, ready и draining случаи. |

### История residual P2-4: откат cleanup при пустой очереди — исправлен

Путь: `content/projects/p3/reference/worker.go:117`, `:122`, `:125`, `:130`.

В начале `claimOne` открыта tx с deferred Rollback. UPDATE переводит expired processing с attempts>=maxAttempts в dead. Затем SELECT ищет только jobs с attempts<maxAttempts. Если SELECT не находит строки, ветвь `pgx.ErrNoRows` возвращает `(nil,false,nil)` **без Commit**. Deferred Rollback отменяет только что сделанный terminal UPDATE.

Конкретный контрпример: в outbox есть одна строка `state='processing', attempts=2, locked_at=now()-interval '31 seconds'`; worker.maxAttempts=2; других ready jobs нет. Cleanup временно меняет её на dead; SELECT ничего не находит; rollback возвращает processing. Каждый следующий poll повторяет это. Drain также возвращает nil как при отсутствии работы, хотя обещанный terminal state не сохранён. Финальная попытка, worker которой погиб, остаётся processing бессрочно до появления другой claim-able строки, позволяющей всей tx дойти до Commit.

Минимальное исправление: commit cleanup перед успешным no-row return либо выполнять cleanup в отдельной корректно фиксируемой операции; propagating commit error обязателен. Добавить DB regression с **единственной expired final-attempt строкой и пустым остальным backlog**: claimOne возвращает no job, но отдельное чтение видит persisted dead и cleared lease. Текущий новый stale-failure test моделирует attempt1→reclaim attempt2→fail и не покрывает crash после claim последней попытки.

Описанный выше дефект обнаружен статически во втором snapshot; финальный branch проверен повторно после отдельного исправления. В текущем `worker.go` ветвь `pgx.ErrNoRows` сначала вызывает `tx.Commit(ctx)`, возвращает ошибку Commit при её наличии и только затем возвращает `(nil,false,nil)`. Deferred Rollback после успешного Commit не отменяет зафиксированный cleanup. **Residual P2-4 закрыт статически.**

В `postgres_integration_test.go:363` добавлена processing-строка `only-expired-final` с attempts=2, locked_at на 31 секунду в прошлом и UUID lease. Последующий `claimOne` обязан вернуть nil/no job, отдельный DB SELECT проверяет persisted state=`dead`, повторный claim снова обязан вернуть no job. Это точный regression для обнаруженной no-row/rollback ветви. При просмотре исходника рецензент не запускал его; автор также прямо отметил, что PostgreSQL test ещё не выполнен. Ни наличие regression source, ни авторские no-DB checks не являются DB-backed PASS.

Для приёмки остаются отдельный запуск PostgreSQL regressions и соответствующее evidence. Открытых статических P1/P2 из этих шести findings после финального recheck не осталось. Старые исходные counterexamples выше относятся к snapshot до исправлений, а не к текущим исправленным строкам.
