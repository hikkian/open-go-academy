# Аудит соответствия taxonomy навыков M11

**Дата:** 2026-10-18  
**Объём:** read-only обзор 88 JSON-файлов уроков `middle-*`, `middleplus-*` и `senior-*`, каталога из 13 навыков в `internal/learning/learning.go` и получения evidence в `internal/learning/catalog.go`. Реализация и данные уроков не изменялись; тесты не запускались.

## Результат

`learning.go` определяет ровно такие ID: `program`, `output`, `functions`, `go-core`, `testing`, `http`, `sql`, `concurrency`, `reliability`, `performance`, `distributed`, `architecture`, `communication`. Во всех упражнениях M11 используются 14 различных ID: `go-core`, `testing`, `backend`, `reliability`, `performance`, `sql`, `distributed-systems`, `systems-design`, `requirements`, `observability`, `capacity-planning`, `data-integrity`, `security`, `architecture`. Восемь из них (`backend`, `distributed-systems`, `systems-design`, `requirements`, `observability`, `capacity-planning`, `data-integrity`, `security`) не являются каноническими. `WithCatalog` перебирает ID навыков у упражнений и обновляет `Snapshot.Skills`, только если находит совпадающий ID. Неизвестные значения молча игнорируются и не попадают в evidence. При этом у первых 28 уроков Middle в основном указаны `go-core`/`testing`, хотя темы — конкурентность, SQL, очереди и backend.

`WithCatalog` устанавливает `Level = 2` для подтверждённого навыка, когда упражнение завершено и у него есть серверное evidence. Этот механизм не присваивает ранг; смена ранга происходит через `WithAwards`. Как оговорено в задании, такой уровень следует трактовать как evidence локальной практики. Это позволяет связывать пройденный локальный helper с темой, но не выдавать его за подтверждение компетентности в production-системах. Успешная проверка чистого helper не доказывает умение работать с production-конкурентностью, PostgreSQL, брокером, безопасностью, производительностью или system design. В JSON-промптах часто прямо указано, что это детерминированные проверки helper-функций, которые не запускают и не проверяют внешнюю систему.

## Рекомендуемое соответствие каноническим навыкам

Всем трём упражнениям урока рекомендуется задавать одинаковые тематические навыки, если отдельно не указано исключение. Это метки локальной практики по теме, а не утверждения о работе системы. Удаляйте неканонические ID. Общий `testing` не добавляется, если задание не проверяет проектирование тестов или диагностику ошибок самим учащимся: запуск тестов проверяющей системой сам по себе не подтверждает, что учащийся умеет писать тесты.

| ID уроков | Канонические ID навыков | Причина / граница доказательств |
|---|---|---|
| `middle-01`–`middle-10` | `concurrency` | Темы goroutine, каналов, синхронизации, race, отмены и worker pool. Текущие упражнения — детерминированные модели (например, арифметика статусов и очередей), а не конкурентные Go-программы; уровень 2 означает только практику с моделью. |
| `middle-11`–`middle-12` | `sql`, `concurrency` | PostgreSQL MVCC, уровни изоляции и конкурентные обновления. Текущие задания классифицируют сценарии снимков/версий, но не выполняют транзакции PostgreSQL и не подтверждают поведение базы данных. |
| `middle-13` | `sql`, `performance` | Тема EXPLAIN и стоимости индексов. Проверки используют фиксированный порог селективности, но не изучают реальный план запроса и не проводят benchmark. |
| `middle-14` | `sql`, `reliability` | Политика получения соединения из пула, очереди и deadline. Чистый классификатор состояния; пул или база данных не запускаются. |
| `middle-15`–`middle-20` | `distributed`, `reliability` | Идемпотентность, фоновые задачи, outbox, доставка/подтверждения/повторы/dead-letter и RabbitMQ. Helper-функции не проверяют доставку брокером или надёжную обработку. |
| `middle-21`–`middle-24` | `distributed`, `reliability` | Redis, проекции, кеширование, rate limit, инвалидация и границы очередей. Правила helper-функций не подтверждают гарантии кеша или Redis. |
| `middle-25`–`middle-26` | `go-core`, `architecture` | Go-интерфейсы/DI и границы handler-service-repository, валидация, доменные ошибки и транзакции. Упражнения M25 с fake-объектами проверяют локальную композицию, но не интеграцию и не атомарный rollback. |
| `middle-27` | `http`, `reliability` | Решения об authentication/authorization и границах tenant. Матрица helper-функций не является middleware, SQL-ограничением области доступа или тестом IDOR/безопасности. См. предлагаемое добавление `security` ниже. |
| `middle-28` | `go-core`, `architecture` | Композиция приложения и dependency injection. |
| `middle-29` | `http`, `reliability` | Middleware и request context; точный объём helper-функции определяет, подтверждается ли только практика с контекстом/политикой. |
| `middle-30`–`middle-31` | `reliability` | Shutdown/drain/readiness/health и структурированные логи. Проверки helper-функций не доказывают поведение жизненного цикла процесса или health endpoint. |
| `middle-32` | `reliability` | Метрики, traces и request context; текущие проверки allowlist/счётчиков не создают instrumentation и не отправляют telemetry. См. предлагаемое добавление `observability`. |
| `middle-33`–`middle-35` | `reliability` | SLI/SLO/error budget, повторы и устойчивость, fault injection/postmortem. Политики helper-функций не подтверждают измеренный SLO или восстановление. |
| `middle-36` | `architecture`, `sql` | Границы и схема TaskFlow, ограничения данных. Текущие функции для строк/состояний не проверяют ограничения базы данных или авторизацию tenant. |
| `middle-37` | `http`, `sql`, `reliability` | CRUD/роли/тесты TaskFlow. Оставляйте соответствующую метку, только если упражнение действительно проверяет этот аспект; одна проверка helper-валидации не подтверждает интеграцию API, SQL или авторизации. |
| `middle-38` | `distributed`, `concurrency`, `reliability` | Outbox и ограниченный worker в TaskFlow. Локальные правила не демонстрируют атомарную публикацию outbox или безопасное выполнение worker. |
| `middle-39` | `concurrency`, `reliability` | Race/shutdown/load для TaskFlow. Оставляйте метку, только если упражнение действительно запускает или проверяет это поведение; объявление/чеклист слабее race test, остановки процесса или нагрузочного прогона. |
| `middle-40` | `architecture`, `communication` | Acceptance и защита решений по TaskFlow; присуждение должно зависеть от реального проектного артефакта и review, а не только от выполнения helper-функции. |
| `middleplus-01`–`middleplus-06` | `performance` | Baseline, benchmark, неопределённость, CPU/heap profiles и итерации. Helper-функция, проверяющая счётчики/метки, лишь подготавливает работу; содержательный вывод о производительности требует измерений benchmark/profile. |
| `middleplus-07`–`middleplus-09` | `sql`, `performance` | SQL-планы, селективность, индексы и partitioning. Текущие проверки диапазонов partition не запускают SQL pruning; `performance` здесь означает лишь локальную модель. |
| `middleplus-10`–`middleplus-11` | `sql`, `distributed`, `reliability` | Задержка репликации, read-your-writes и согласованность. Входы — fixture-флаги; реплика и гарантия свежести не проверяются. |
| `middleplus-12` | `distributed`, `reliability` | Корректность кеша и инвалидация; реальное поведение согласованности кеша не проверяется. |
| `middleplus-13`–`middleplus-20` | `performance`, `distributed`, `reliability` | Capacity, backpressure, возраст очереди, retry budget, outbox, admission/load shedding, восстановление и ограничения worker. Задания в основном классифицируют переданные значения/boolean-флаги; не следует делать из них выводы об измеренной ёмкости, доставке или восстановлении. |
| `middleplus-21` | `reliability`, `performance` | SLI/SLO и error budget. Evidence по коммуникации требует review объяснения, а не только результата helper-функции. |
| `middleplus-22` | `distributed`, `reliability` | Политика canary rollout и rollback. Реальный выпуск не выполняется. |
| `middleplus-23` | `reliability`, `communication` | Артефакт postmortem и системные меры; `communication` подходит только при проверке письменного артефакта. |
| `middleplus-24` | `performance`, `distributed`, `reliability`, `communication` | Решение/защита ScaleLab. `communication` подходит только при проверке артефакта решения; helper не проверяет хеши прогонов и фактическое выполнение fault tests. |
| `senior-01` | `architecture`, `communication` | Требования и отделение предположений от evidence, обратимые решения. Текущий helper классифицирует переданные вызывающим метки, но не проверяет качество источников или решения. |
| `senior-02` | `architecture`, `reliability` | Quality attributes и ограниченная деградация. Helper-функции классифицируют счётчики/политику, но не измеряют SLI/SLO и не оценивают trade-offs атрибутов. |
| `senior-03` | `architecture`, `performance`, `reliability` | Capacity planning и admission policy. Арифметика над переданными значениями не измеряет нагрузку или ёмкость. |
| `senior-04` | `architecture`, `distributed`, `reliability` | Владение данными, идемпотентность, optimistic concurrency и задержка проекции. Helper-функции не обеспечивают persistence/atomicity. |
| `senior-05`–`senior-06` | `architecture`, `reliability` | Границы системы/компонентов и эволюция modular monolith. Это темы для проектных решений; выполнение helper-функции не подтверждает качество архитектуры. |
| `senior-07` | `architecture`, `distributed`, `reliability` | Согласованность распределённых операций. Модель решения в упражнении не устанавливает distributed guarantees. |
| `senior-08`–`senior-09` | `reliability`, `distributed` | Домены отказа, деградация и disaster recovery. Для утверждений о восстановлении нужны фактические evidence восстановления. |
| `senior-10`–`senior-12` | `architecture`, `reliability` | Threat model, изоляция tenant, секреты/ротация ключей. Текущие helper-функции не проверяют security controls; добавьте отдельный навык `security`, прежде чем присваивать evidence по безопасности. |
| `senior-13`–`senior-15` | `architecture`, `communication` | Контекст P5/quality attributes, выбор хранилища, контракты/ADR. Helper-функции проверяют наличие полей/статус, а не качество артефакта или совместимость. |
| `senior-16` | `architecture`, `distributed`, `reliability` | Компромиссы scale-out. Классификатор helper-функции не является измеренным сравнением масштабирования. |
| `senior-17` | `architecture`, `reliability`, `distributed` | Migration/cutover/rollback. Boolean helper не выполняет и не проверяет миграцию или rollback. |
| `senior-18` | `reliability`, `communication` | Observability и расследование причин. Добавьте `observability` как отдельный навык, если это критерий. Текущие проверки валидируют allowlist/счётчики, а не реально выданные сигналы или причинный анализ. |
| `senior-19`–`senior-20` | `reliability`, `communication` | Incident command, release gates и координация rollback. Чеклист-helper не демонстрирует руководство инцидентом или безопасную эксплуатацию выпуска. |
| `senior-21` | `architecture`, `performance`, `reliability` | Capacity/cost planning. Арифметика явно синтетическая и не является ценовым предложением, бюджетом или прогнозом ёмкости. |
| `senior-22`–`senior-24` | `architecture`, `communication`, `reliability` | Code review, защита решения, retrospective и изменение архитектуры. Status-helper не оценивает качество review, его независимость или эффективность исправлений. |

Эти диапазоны покрывают все 40 уроков Middle, 24 Middle+ и 24 Senior (всего 88). Были просмотрены названия и outcomes всех 88 уроков, сопоставлены exercise metadata во всех уроках и выборочно изучены промпты по конкурентности, SQL/backend, DI, авторизации, observability, проектам, масштабированию и Senior-треку. Проверка набора текущих меток по каждому рангу показывает: во всех трёх упражнениях одного урока повторяются одинаковые ID.

## Узкие задания и возможные новые навыки

Локальную тематическую метку можно сохранять для учёта практики, но многие текущие небольшие helper-функции слишком узки для выводов о продвинутых системных навыках. Особенно это касается `middle-01`–`middle-24` (много простых целочисленных/строковых политик), `middle-27` (матрица helper-функций для auth), `middle-32` (allowlist меток), `middle-36` (helper-функции для ввода/состояния), `middleplus-09`–`middleplus-10` (правила для partition/replica на fixture-данных), `middleplus-13`–`middleplus-24` (в основном переданные вызывающим значения/флаги) и большинства Senior-упражнений (status/boolean helper-функции). Такое evidence следует оставлять на уровне локальной практики, а для критериев реальных систем требовать project-, integration-, measured- или human-review evidence. В M36 Hard прямо сказано, что это warm-up, а не подтверждение компетентности Middle; прохождение этого задания не должно само по себе выдавать сигнал навыка Middle с более строгими требованиями.

Возможные дополнительные ID, если продукт хочет показывать эти компетенции отдельно, а не включать их в широкие канонические навыки:

| Предлагаемый ID | Уроки/область | Почему отдельный навык может быть полезен | Какое evidence нужно сверх текущих helper-функций |
|---|---|---|---|
| `security` | M27; Senior-10–12; проектные задания по authorization/security | Безопасность прямо указана в спецификации и существенно отличается от общей надёжности/архитектуры. | Отрицательные integration tests для границ tenant/auth, проверка controls и review threat model. |
| `observability` | M31–33; Middle+-21; Senior-02/18 | Instrumentation, качество сигналов и причинное расследование сейчас скрыты внутри reliability. | Реально созданные metrics/logs/traces, проверки cardinality/privacy и анализ запросов/инцидента. |
| `requirements` | Senior-01–03, Senior-13 | Исследование требований, их источников и acceptance criteria — отдельная важная область Senior-трека. | Проверенные человеком прослеживаемые критерии с владельцами, источниками, измеримой приёмкой и обновлением предположений. |
| `data-integrity` | M11–12, M15–18, M23–24, M36–38; Senior-04/17 | Инварианты данных, транзакции/идемпотентность и корректность миграций не сводятся к общему backend. | Реальные DB transaction/integration tests, replay/reconciliation и evidence миграции/rollback. |
| `capacity-planning` | Middle+-13; Senior-03/21 | Планирование ёмкости и затрат явно присутствует в программе, но отличается от оптимизации/performance. | Измерения на одинаковой нагрузке, указанные единицы/предположения, результаты нагрузочного теста и проверенные оценки. |
| `systems-design` | Senior-трек/P5 | Можно выделить system design как родительскую компетенцию для архитектурных решений, если продукту нужна отдельная метка. Большая часть смысла уже покрывается `architecture`, поэтому добавляйте её только при наличии отдельной rubric/evidence. | Проверенные design-артефакты, альтернативы, ограничения, прототипы и явные trade-offs. |

Не следует добавлять широкие catch-all ID `backend` и `distributed-systems`: первое лучше разложить на конкретные темы HTTP/SQL/architecture/reliability, а второе нормализовать к существующему `distributed`. `systems-design` также можно нормализовать к `architecture`, пока rubric не будет явно отделена. `testing` не следует использовать как метку по умолчанию только потому, что каждое задание проверяется тестами.

## Примечания для владельца реализации

- Обновляйте `skills` у всех трёх упражнений урока согласованно; не оставляйте неизвестные ID, поскольку `WithCatalog` молча их отбрасывает.
- Рассмотрите добавление проверки/диагностики неизвестных ID навыков в каталоге, чтобы опечатки metadata не приводили к незаметной потере evidence.
- Сохраняйте различие между evidence навыка и присвоением ранга. В текущем коде переход ранга уже выполняется через `WithAwards`; тематическая разметка уроков не должна менять эту политику.
- Если rubric требует integration/measured/project evidence, не кодируйте прохождение чистого helper как такое доказательство. Используйте отдельный тип evidence/критерия либо оставляйте навык на уровне локальной практики до появления более сильного evidence.
