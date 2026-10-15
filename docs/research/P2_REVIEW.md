# Независимое ревью reference P2 LinkHub

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`. Scope: доверенный учебный reference `content/projects/p2`; только статический анализ и обычные положительные/отрицательные unit fixtures. Враждебные программы, эксплуатационные proof и внешние/платные сервисы не запускались. Reviewer меняет только этот отчёт, production code не редактировал. Подагенты не запускались.

## Текущий результат

High/Critical нарушения auth/ownership в прочитанном коде не обнаружены. **P2-R1/R2/R3 и P2-L1–L4 исправлены и CLOSED; открытых High/Medium в проверенной области нет.** Reviewer дополнительно прочитал фактический PostgreSQL18.6 log и isolation script: integration fixture с MaxConns1 прошёл. Это unit HTTP tests с fake Store и реальные PGStore tests, а не полный HTTP+PostgreSQL end-to-end.

## Контекст и границы

Прочитаны P2_LINKHUB research, E4 MASTER_SPEC, P2 README/threat-model/OpenAPI/Compose/Dockerfile, reference config/main/model/security/auth/http/postgres/migrations и оба набора tests. Новые P2 файлы не заменяют прежние security guards. Типовая граница: HTTP → проверка Origin/CSRF/session → owner из server session → параметризованный pgx SQL → атомарная link/tag транзакция. Redirect только возвращает сохранённый Location, target сервером не запрашивается. API LinkHub не принимает код на выполнение и не имеет зависимости от academy runner/socket.

Доверенные стороны — оператор local DB/config и reference author. Пароль/session/CSRF, IDs и тела запросов недоверенные. DB constraints и owner predicates составляют существенную часть гарантии: fake store tests не заменяют PostgreSQL. Public deployment не разрешён этим ревью.

## P2-R1 — Medium, CLOSED: exhausted generated-slug retries возвращали ложный201

**Путь:** `content/projects/p2/reference/http.go`, createLink.

После каждого errConflict исходный loop выполнял `_, e = rand.Read(...)`. На четвёртом конфликте успешный rand.Read заменял errConflict на nil, а цикл завершался; handler возвращал201 с zero-value Link, хотя запись не создана. Это обычный граничный сценарий коллизий, не auth exploit.

**Исправление и evidence:** на последней попытке loop сохраняет conflict и выходит; ошибка генератора использует отдельную randomErr. `TestGeneratedSlugExhaustionReturnsConflict` задаёт fake conflictAlways, ожидает409, ровно4 store calls и0 созданных записей. Reviewer прочитал fixture и запустил focused unit group; PASS. Позитивный retry после2 конфликтов также PASS. CLOSED.

## P2-R2 — Medium, CLOSED: ListLinks удерживал два pool connections

**Путь:** `reference/postgres.go`, ListLinks.

Исходный for rows.Next удерживал connection основного SELECT и вызывал readTags через pool для каждой строки. При ограниченном pool один обычный ListLinks с MaxConns1 не мог завершиться; при MaxConns8 такое же ожидание возможно у8 одновременных lists. Запросы завершались только по operation timeout. Этот вывод основан на статическом анализе lifecycle rows, reviewer не запускал нагрузочный сценарий.

**Исправление:** сначала собираются bounded link rows, вызывается rows.Close, затем выполняется один bulk tags SELECT. Tags инициализируются пустыми slices для корректного JSON array. Integration fixture создаёт limitedPool MaxConns1, две ссылки с тегами и проверяет корректный list. Код/fixture прочитаны. Дополнительно reviewer прочитал `.data/p2-postgres.log`: PostgreSQL18.6, TestPostgresOwnershipAndAtomicTags PASS0.02s, весь reference suite PASS0.373s. Прочитан выполнявший проверку `scripts/p2-reference-check.py`. Regression проверяет исходное ожидание второго соединения реальным PGStore; CLOSED. Исполнение скрипта выполнял родитель, reviewer проверил предоставленные evidence.

## P2-R3 — Medium, CLOSED: OpenAPI Tag отвергает реальные{id,name}

**Путь:** `content/projects/p2/openapi.yaml`, components.schemas.Tag.

Tag задан через allOf: TagInput с additionalProperties:false и только name, затем объект с id. JSON Schema проверяет каждый allOf independently; TagInput отвергает id. Таким образом фактические Tag responses и Link.tags не соответствуют опубликованному контракту. Это мешает schema validation/generated clients.

**Минимум:** явные id/name properties у Tag с additionalProperties:false либо корректный outer unevaluatedProperties:false. Проверить обычный `{id:<uuid>,name:"go"}` schema validator и неизвестное поле в negative fixture. Статический вывод; validator reviewer не запускал.

**Повторная проверка:** Tag теперь является одним object с required[id,name], явными properties и additionalProperties:false. HTTP fixture проверяет обязательные response fields. Исходное противоречие allOf устранено; CLOSED по статическому чтению и unit PASS. Попытка локальной schema validation не выполнилась: Python jsonschema отсутствует; ничего не устанавливалось, semantic schema validation не заявляется.

## Low замечания

- **P2-L1, CLOSED — HTTPS cookie flags/config normalization.** Config.LoadConfig вычислял SecureCookies по case-sensitive префиксу сырого env до url.Parse. Стандартный net/url нормализует scheme lowercase, поэтому `HTTPS://...` проходил проверку scheme, но SecureCookies=false; origin string оставалась ненормализованной и не совпадала с браузерным Origin. Исправлено: normalizeOrigin строит lower scheme/host, удаляет стандартный порт и вычисляет SecureCookies после parse. Прочитано повторно; полученный позже PostgreSQL log также содержит PASS TestLoadConfigNormalizesOriginAndCookieSecurity для uppercase HTTPS/default ports и PASS TestLoadConfigRejectsNonOriginURL. Это ошибка operator configuration, не доказанная удалённая эксплуатация.
- **P2-L2, CLOSED — logout contract.** OpenAPI logout не объявлял required X-CSRF-Token parameter, хотя authenticated logout проверяет его. Добавлен общий CSRF parameter; повторно прочитано.
- **P2-L3, CLOSED — malformed ID delete даёт500.** DeleteLink/DeleteTag возвращали raw pgx error, тогда как Get/Update используют translate. Обычный malformed UUID давал SQLSTATE22P02 и500 вместо consistent404; DB details в ответ не выходили. Обе delete ветки теперь вызывают translate(e). Прочитано повторно; reviewer не обращался к DB для этого отрицательного случая.
- **P2-L4, CLOSED — URL без hostname.** validTarget проверял u.Host!=empty, поэтому `https://:443/path` проходил с Host=:443 и пустым Hostname. Добавлен hostname!=empty; обычный negative helper fixture с этим URL и positive IPv6 URL проходит. Сервер не делает outbound запрос: это некорректный сохранённый redirect, не SSRF.

## Проверенные защитные свойства

- Password — versioned1+16byte random salt+32byte PBKDF2-SHA256,600000 iterations. Constant-time derived key comparison, dummy derivation для неизвестного пользователя,128byte max перед KDF;4 concurrent KDF slots и12 auth attempts/IP/min с bounded10000 entries. Точная calibration/offline cracking resistance не сертифицированы.
- Session и CSRF credentials по32random bytes; в PostgreSQL только SHA-256 hashes. Session HttpOnly, SameSiteStrict, Secure в обычной lowercase HTTPS config; readable CSRF cookie сверяется с сохранённым hash. Logout удаляет server session, expiry проверяется в SQL и приложении. Роль регистрации берётся из DB default'user', user_id/role body не принимаются.
- Mutations cookie-authenticated пользователя требуют header CSRF, exact configured Origin при наличии browser Origin; cross-site fetch отвергается. Отсутствующий Origin разрешён для CLI, а не описан как публичная browser-auth гарантия. GET/HEAD не мутируют записи.
- JSON body≤16KiB, unknown/trailing fields rejected; link list limit≤100/offset≤100000; server/pool/statement/lock timeouts ограничены. Auth errors и SQL errors в HTTP generic; request log содержит method/status/latency/request ID, не password/token/DSN/target URL.
- Owner в link/tag CRUD берётся из session и включён в SQL predicate. Link/tag association выбирает tags только с этим owner; link mutation и replacement links/tags в одной транзакции. Неподходящий tag вызывает rollback. Values параметризованы; SQL identifiers/order constants.
- Migration transaction+фиксированный advisory lock на одном dedicated connection; version record и schema атомарны. UUID defaults, unique slug/normalized owner tag, FKs/cascade и индексы заданы SQL. Down — отдельная явная destructive operator command, README требует disposable DB и остановить API.
- Target допускает только http/https без userinfo/control chars и≤2048bytes. Redirect использует saved enabled row и только http.Redirect; исходящие HTTP calls отсутствуют. Open redirect — сам продукт short-link, не непредусмотренный SSRF.
- Compose публикует API/DB только loopback, API user65532, read-only/capsdrop/no-new-privileges; DB credentials из локального env. Модель явно локальная учебная, не production identity system.

## Фактические проверки

Из `content/projects/p2/reference` выполнено:

```sh
/home/alisher/Desktop/open-go-academy/.tools/go/bin/go test -count=1 -timeout=45s ./...
/home/alisher/Desktop/open-go-academy/.tools/go/bin/go test -count=1 -timeout=45s -run 'TestGeneratedSlug|TestMutation|TestRegister|TestHelpers' ./...
```

Результат: unit suite PASS0.336s; focused group после slug fix PASS0.349s. Интеграционный PostgreSQL test opt-in без LINKHUB_TEST_DATABASE_URL пропущен; PASS unit команды не означает проверку SQL на реальной БД. Первоначальная попытка команды с относительным `.tools/go` из вложенного module не запустила тесты (path not found); выше записан исправленный фактически исполненный путь.

Финальный повтор unit suite после config/validation/OpenAPI/delete fixes: PASS0.329s. Read-only команды в этом вызове сначала использовали неверный относительный путь вложенного module и были повторены с корректным путём; это не повлияло на самостоятельно выполненную test команду и исходники не меняло. Отмеченный ранее config normalization table regression впоследствии добавлен и имеет PASS в изолированном reference suite.

## Дополнительные проверенные evidence PostgreSQL

Reviewer прочитал, но не запускал повторно:

- `.data/p2-postgres.log`: PostgreSQL18.6 Debian; TestPostgresOwnershipAndAtomicTags PASS0.02s; все unit/reference tests PASS, package0.373s. Есть реальные проверки owner predicates, rollback failed tag association, unique slug, migration version и двух tagged links при MaxConns1.
- `scripts/p2-reference-check.py`: отдельный случайно именованный rootless Podman PostgreSQL container с network=none, без host ports, ephemeral tmpfs; SDK container присоединяется только к тому же DB network namespace. Runtime proxy disabled, cap-dropALL/no-new-privileges/read-only, source и отдельный module cache монтируются read-only; тестируется только доверенный reference. Зависимости предварительно загружаются отдельным trusted `go mod download`, runtime Go использует GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local.
- Проверочный класс P2 имеет собственные лимиты: PostgreSQL512MiB и отдельные tmpfs; Go1536MiB, work512MiB/cache512MiB, tmp128MiB noexec, GOTMPDIR=/work/build на exec work; memory-swap равен memory, что запрещает дополнительный swap. Timeout Go120s/контейнера180s, lifecycle удаляет только именованную fixture и временную копию reference. Прежняя политика academy runner этим скриптом не меняется.

По данным интегратора первый cold build упёрся в tmp quota/noexec и был повторён после настройки этого bounded P2 class. Reviewer не прочитал отдельный первоначальный failure log и не выдаёт исправленный PASS за успех первой попытки. Прочитанный окончательный log подтверждает успешный повтор.

Граница доказательств: HTTP handlers проверяются fake Store unit fixtures; PGStore/миграции/транзакции — настоящим PostgreSQL18.6. Полный HTTP registration/login/CRUD с реальной PostgreSQL через браузер/сетевой API этим тестом не выполняется и не заявляется.

Reviewer самостоятельно не запускал PostgreSQL/Compose, KDF calibration benchmark, schema validator, browser cookies flow или полный race suite. Предоставленный родителем успешный PostgreSQL run и его конфигурация проверены чтением, как указано выше. Результаты не доказывают готовность публичного сервиса, защиту от распределённого злоупотребления или отсутствие ошибок во всех concurrency interleavings.
