# Open Go Academy — Master Product & Engineering Specification

**Версия ТЗ:** 1.0 · **Дата:** 2026-10-14 · **Рабочее название / репозиторий:** `open-go-academy`  
**Статус документа:** полное исходное задание для автономного coding agent.  
**Операционные правила:** [`../AGENTS.md`](../AGENTS.md) · **План:** [`EXECUTION_PLAN.md`](EXECUTION_PLAN.md) · **Состояние:** [`../AGENT_HANDOFF.md`](../AGENT_HANDOFF.md).

> **Главная цель:** построить настоящую бесплатную open-source интерактивную веб-платформу обучения Go Backend Development **от абсолютного нуля до продвинутых Senior-track навыков**, с редактором и исполнением Go-кода, проверкой заданий, курсом на русском, англо-русской терминологией, проектами, ИИ-наставником, экономной маршрутизацией моделей, карьерными уровнями и проверяемыми учебными сертификатами. Не подменяй продукт списком тем, чатом, макетом или неработающими кнопками.

---

## A. Цели, границы и принципы

### A1. Для кого

Любой новичок, который **не обязан знать вообще ничего**: программирование, устройство компьютера, терминал, Git, язык Go, базы данных или HTTP. Курс не персонализировать по предполагаемому опыту владельца проекта. Теория полностью на русском, английские термины при первом использовании с точным переводом и понятным определением; код и имена API преимущественно на английском. Прогресс и задания подходят и самостоятельному студенту без ИИ, и студенту с AI Tutor.

### A2. Ключевой пользовательский результат

Пользователь открывает сайт → создаёт локальный профиль (без обязательного внешнего аккаунта) → видит карьерную карту Zero → Senior → открывает урок → читает самодостаточную теорию → пишет Go в браузере → исполняет в безопасной песочнице → видит stdout/stderr и ошибки → запускает тесты → получает объяснения/подсказки → сохраняет прогресс → возвращается без потери данных → завершает проекты, экзамены и получает **учебный** подтверждённый ранг. Для продвинутых проектов пользователь может перейти на полноценное локальное окружение, с интеграцией в учебный интерфейс и точными инструкциями.

### A3. Негативные требования (нельзя)

- Не проси пользователя подбирать стек, вручную проверять каждую кнопку, искать обязательные учебники, писать тесты платформы или объединять результаты агентов.
- Не выдавай пустые `TODO`-уроки, моковый вывод терминала или интерфейс без рабочего backend за готовую функциональность.
- Не требуй платные AI API для прохождения курса. AI Tutor опционален; преподавание и проверка базовых задач доступны без LLM.
- Не называй завершение Senior-track или сертификат гарантией Senior-грейда, трудоустройства, аккредитации или стажа.
- Не открывай публичное исполнение недоверенного кода без технически обоснованной изоляции и ограничений.
- Не публикуй репозиторий, не трать деньги и не создавай внешний ресурс без необходимых разрешений.
- Не обещай работу «в фоне» после завершения сессии, если механизм продолжения реально не активен.

### A4. Временной масштаб

Путь Zero → Middle Readiness имеет **ориентировочный интенсивный маршрут на 16 недель по ~20–25 часов/неделю**; это примерная программа, не гарантированный грейд и не принудительный дедлайн. Middle+ и Senior — отдельные, существенно более глубокие треки, их длительность зависит от опыта и практики, часто требует дополнительных месяцев/лет. Не жертвуй необходимой глубиной ради чисел. Поддерживай самостоятельный темп, пропуск знакомых глав и обязательную проверку навыка перед присвоением ранга.

---

## B. PHASE 0: актуальность и выбор технологий (блокирующий этап)

Перед окончательным выбором стека и началом основного контента **фактически проверь** на дату работы следующие сведения по первичным источникам. Не утверждай версию на основании памяти модели. Если сеть недоступна, явно обозначь незавершённую проверку, локально проверь всё доступное и не выдумывай публикационные даты.

1. **Go:** последняя стабильная версия + патч, release notes, срок поддержки, изменения языка, стандартной библиотеки, `net/http`, `testing`, компилятора, runtime, GC, scheduler, модулей и security. Источники: [go.dev](https://go.dev/), [release history](https://go.dev/doc/devel/release), [pkg.go.dev](https://pkg.go.dev/), официальный репозиторий Go.
2. **Backend:** стандартный `net/http` и ServeMux; `chi`, `Gin`, `Echo`, `Fiber`; `database/sql`, `pgx`, `sqlc`, `GORM`, `Ent`; Postgres; migration tools; `slog`, `zap`, `zerolog`; `testify`, `testcontainers-go`; Redis; RabbitMQ/Kafka; gRPC/Protobuf; OTel; Docker; профилирование и security tools. Выбери **не всё**, а наименьший подходящий и поддерживаемый набор. Для образовательного трека основы `net/http` объяснить *до* внешних фреймворков.
3. **Frontend:** React/TypeScript, Vite/Next.js или альтернативы, Monaco/CodeMirror, Markdown/MDX, тестирование компонентов, Playwright, accessibility, UI primitives; выбирай поддерживаемые совместимые версии и оптимальную сложность.
4. **AI/Codex:** актуальная поддержка субагентов, проектной `.codex/config.toml`, ролей, лимитов параллелизма, доступных моделей и pricing/usage; API и локальные OpenAI-compatible endpoint; fallback/caching. **Не считай подписку на чат равной отдельному API-ключу**.
5. **Безопасность:** решения для недоверенного выполнения Go, модель угроз для local-only и public multi-tenant, возможности ограничения ядра и сети, риски контейнерных решений.
6. **Digital credentials:** актуальные Open Badges 3.0 / Verifiable Credentials, подписи, revocation/status, приватность и различие issuer trust versus local profile achievement.

**Выходные файлы:** `docs/research/GO_CURRENT_STATE.md`, `ECOSYSTEM.md`, `FRONTEND.md`, `AI_CODEX.md`, `SANDBOX_SECURITY.md`, `CREDENTIALS.md`, `SOURCES.md`, `VERSION_MATRIX.md`; плюс `docs/architecture/ADR-0001-tech-stack.md`. Каждое существенное решение: дата + исходная ссылка + альтернативы + trade-offs + требования к совместимости. Зафиксируй версии/lockfiles и не обновляй автоматически без регрессионных тестов.

**Критерий M0:** не «список библиотек», а документированный воспроизводимый выбор с проверенными версиями и доступными тестами/инструментами.

---

## C. Продуктовые модули и экраны

### C1. Главные экраны

- **Landing:** что за платформа, кому подходит, шесть уровней, пример урока/проекта, бесплатный старт; без фальшивых отзывов/статистики.
- **Onboarding:** local profile/optional account, уровень Zero по умолчанию, настройки языка/темы, AI необязателен.
- **Dashboard:** активный урок, учебный ранг, XP, прогресс, время практики (только измеренное), streak, проекты, слабые темы, недавние попытки, следующий шаг.
- **Career Path:** наглядная карта Zero → Junior → Junior+ → Middle → Middle+ → Senior; состав ранга, критерии экзамена, освоенные и недостающие навыки, прогресс.
- **Course Explorer:** модули, главы, вложенные разделы, зависимости, поиск, статусы, пропуск темы с предупреждением о prerequisite.
- **Lesson Workspace:** теория + навигация + редактор и консоль/тесты + опциональный AI Tutor; изменение ширины панелей, Focus Mode, desktop/mobile UI.
- **Practice / Challenges:** условия, starter code, public examples, ограничения, запуски, попытки, автопроверка, подсказки, hidden tests на сервере.
- **Projects Workspace:** дерево файлов, редактирование нескольких файлов, milestones, тесты, экспорт ZIP/Git-ready, критерии приёмки, code review.
- **Exams:** теория, тесты, исправление багов, Go/SQL, system design, незнакомая кодовая база, проекты и защита архитектуры.
- **Skills Matrix:** уровни 0–4, наблюдаемые доказательства и обязательные пробелы.
- **Achievements / Certificates / Developer Passport:** XP, эмблемы, проверяемые учебные ранги и сертификаты, публичное портфолио только с согласия.
- **AI Tutor / Provider Settings:** объяснения, подсказки, код-ревью, бюджет, локальная/облачная модель, usage/fallback, приватность.
- **Settings, profile, export/delete account, documentation, certificate verification.**

### C2. Дизайн и доступность

Нужен современный аккуратный developer-oriented UI, не набор случайных градиентов и огромных карточек. Система токенов (цвета, типографика, spacing, состояния), светлая/тёмная тема, responsive mobile/tablet/desktop, заметный keyboard focus, accessible names, контраст, reduced motion. В панели с кодом: моноширинный шрифт, горизонтальная прокрутка, удобный размер, переносимые настройки. Никаких «кнопок для красоты»: каждый интерактивный элемент либо действует, либо явно disabled с объяснением. Применяй читабельные схемы/диаграммы, а не декоративные dashboard charts без смысла.

### C3. Прогресс и история

Фиксируй завершения уроков, attempts, решения, результаты тестов, отметки пропуска, навыки, проекты, экзамены, XP, ранги, учебное время и настройки. Состояние сохраняется при перезагрузке и перезапуске сервиса; предусмотрены миграции, backup/restore и export. Расчёты прогресса/ранга **детерминированы и серверные**; LLM не может выдать себе/пользователю ранг. Статистика основана на реальных событиях, а не hardcode.

---

## D. Среда разработки и выполнение Go-кода

### D1. Редактор

Monaco либо подтверждённая исследованием альтернатива: подсветка Go, line numbers, tabs, multi-file workspace, format, undo/redo, поиск, автоотступы, resize, полноэкранный режим, черновики/autosave, run/submit/reset, stdout/stderr, compiler diagnostics, тесты, timeouts. Автодополнение/gopls только если реально реализовано. Учитывай файловые задания (не все решения — единственный `main.go`).

### D2. Исполнитель недоверенного кода

Сделай отдельный защищённый execution boundary. Для публичного многопользовательского запуска потребуется усиленная изоляция (подходящий sandbox/runtime) сверх «запустить Docker по умолчанию». Должны быть реализованы/подтверждены:

- Непривилегированный пользователь, read-only image/rootfs где применимо, временный workspace, запрет доступа к host-секретам и daemon socket, сеть отключена по умолчанию, доступ только к нужным путям.
- CPU/memory/PID/time/output/disk limits, лимиты concurrency, queue, cleanup и защита от бесконечных циклов/злоупотреблений.
- Отключены или строго ограничены Linux capabilities; невозможность монтировать host FS, запускать произвольные privileged containers и использовать системные привилегии сервиса API.
- Контроль stdout/stderr/exit code; параметризованные задания с разрешёнными зависимостями, воспроизводимая версия Go.
- Явное разделение `local safe development mode` и `public hardened deployment`; публичный доступ к runner запрещён по умолчанию, пока не реализованы соответствующие меры.
- Тесты негативных случаев, модель угроз и остаточных рисков, обновление образов/инструментов.

**Запрещено:** `os/exec` над пользовательскими исходниками прямо из API-процесса без isolation. Нельзя представлять обычное process timeout как достаточную защиту.

### D3. Проверка заданий

Deterministic-first: компиляция, запуск, golden output только где подходит, unit/table-driven tests, hidden tests, edge cases, limits, allowed files, security and resource limits. Hidden tests живут на сервере в доверенной зоне; в полностью локальном репозитории пользователь с доступом к исходникам может их просмотреть — не выдавай это за прокторинг. Для сложных решений используй гибрид: тесты + качественные критерии + AI review как дополнение, не как единственный судья.

### D4. Проекты за пределами простого runner

Сложные проекты с БД, Redis, очередью и CI могут потребовать локального project runner / Docker Compose на компьютере студента. В уроке дай команды и интеграцию экспорта/синхронизации там, где возможно, вместо опасного исполнения чужой инфраструктуры на публичном общем сервере. Поддерживай reproducible starter templates.

---

## E. Учебная программа и ранги

### E1. Единая шкала уровней

| Ранг | Практические требования | Основные темы / результат |
|---|---|---|
| **Zero** | Первая самостоятельно запускаемая программа, простые задачи | Устройство ПК, файлы, OS, терминал, бинарные данные, код/компиляция, установки, типы, переменные, условия, циклы, функции |
| **Junior** | CLI Toolkit и контроль Go core | Arrays/slices/maps, Unicode, structs, pointers, methods, interfaces, errors, packages, modules, generics, basic complexity/algorithms, Git, testing, debugging |
| **Junior+** | LinkHub REST API с PostgreSQL и тестами | HTTP/TLS/DNS/TCP basics, REST, net/http, middleware, SQL, pgx, transactions basics, validation, auth, Docker, API architecture |
| **Middle** | TaskFlow backend, эксплуатационные тесты, инженерная защита | Goroutines, channels, mutex, context, race/deadlock/leaks, scheduler, isolation/MVCC/indexes, Redis, queues, resilience, CI/CD, security, profiling/observability |
| **Middle+** | ScaleLab performance/reliability case с измерениями | High load, distributed consistency, advanced SQL, replication, caching, outbox, messaging, OTel, SLI/SLO, fault injection, incident response, capacity |
| **Senior** | Architecture capstone + ADRs/design review/trade-offs | Distributed systems, HA, scalability, resilience, architecture evolution, security design, incident leadership, mentorship, engineering communication |

Ранги — **учебные подтверждения**, не гарантия профессионального стажа. Skills Matrix 0=не изучено, 1=понимает термин, 2=базовые упражнения, 3=самостоятельные типовые задачи, 4=сложные задачи и аргументация trade-offs. XP независимо от Verified Rank.

### E2. Базовый путь до Middle: ориентир 16 недель

| Неделя | Основное содержание | Практический контроль |
|---|---|---|
| 1 | Устройство компьютера, OS, файлы, терминал, код, компиляция, Go Hello World, типы | Запустить и объяснить простую программу |
| 2 | Операторы, условия, циклы, функции, строки, массивы, slices/maps | Самостоятельные базовые задачи |
| 3 | structs, methods, pointers, memory model basics, interfaces, error/defer, packages, generics | Малые модули с тестами |
| 4 | Big O, алгоритмы, Git/GitHub, testing, debugging, CLI project | Проект 1: CLI Toolkit |
| 5 | Клиент/сервер, IP, TCP, DNS, TLS, HTTP, JSON, net/http, REST, middleware | Мини-API |
| 6 | PostgreSQL/SQL с нуля, joins/indexes, transactions, isolation, migrations, pgx | Доступ к данным, SQL-проверки |
| 7 | Go backend structure, handlers/services/repos, DI, validation, config, logs, тестирование | Refactor + архитектурное review |
| 8 | Authentication/authz, sessions/JWT, OWASP, integration tests, Docker | Проект 2: LinkHub API |
| 9 | Concurrency, parallelism, scheduler, goroutines/channels, select, sync, context, race/deadlock/leaks | Worker pool + race tests |
| 10 | Redis, cache invalidation, background jobs, RabbitMQ, retries, idempotency, outbox | Async job pipeline |
| 11 | Linux processes/network basics, CI/CD, containers, secrets, deployments, reliability | Automated pipeline + deploy |
| 12 | GC/escape analysis, pprof, benchmark, load test, latency, logs/metrics/traces | Profiling case |
| 13 | gRPC, protobuf, modular monolith, microservices trade-offs, consistency, scaling | System design exercise |
| 14 | TaskFlow: requirements, schema, auth, CRUD, tests | Project milestone 1 |
| 15 | TaskFlow: resilience, async jobs, monitoring, perf, security, release | Project milestone 2 |
| 16 | Go/SQL/concurrency interviews, debugging, system design, live coding, capstone defense | Verified Middle-track assessment |

Это последовательность, а не обещание, что вся аудитория станет Middle за 16 недель. Создай подробную разбивку на **учебные дни и уроки** с prerequisites, временем, заданием, проверкой и итогом. Для Middle+ и Senior создай сопоставимо полноценные, но более длительные самостоятельные программы, не фиктивные пару страниц.

### E3. Обязательная структура каждого урока

1. Metadata: ID, rank, module, lessons/dependencies, estimated time, learning outcomes, glossary additions.
2. Мотивация «зачем нужна эта проблема» и объяснение с нуля.
3. Полная точная теория (на русском), английские термины с переводом при первом появлении; от простого к механизму.
4. Работающие Go-примеры с файлами/путями/импортами и объяснением исполнения; схемы/таблицы/эксперименты.
5. Типичные ошибки и диагностика.
6. Practice: Easy/Medium/Hard, ясные условия, input/output, критерии, edge cases; **решение скрыто до попытки или явного запроса**.
7. Public tests, server-side hidden tests при уместности, эталонные решения в отдельном каталоге.
8. Knowledge checks, итог, рекомендации по повторению, ссылки на prerequisites, мини-глоссарий.
9. Self-study без AI: текстовые объяснения, проверяемые критерии, troubleshooting и самостоятельная проверка.

Нельзя допускать в заданиях обязательных неизвестных технологий без отдельного объяснения. Не сокращай фундаментальные главы до конспекта или внешней ссылки. Не перенаправляй на видео/курсы/статьи как на обязательный источник. Обязательный контент оригинален, не копирует чужие книги.

### E4. Проекты (минимум)

**P1 Go CLI Toolkit (Junior):** аргументы, файлы, JSON, структуры, errors, tests, README, Git workflow, функциональное ТЗ и release.

**P2 LinkHub API (Junior+):** short links + redirects/CRUD, PostgreSQL, migrations, indexes, auth/roles где нужно, validation, net/http, middleware, logging, OpenAPI, integration tests, Docker Compose, threat basics.

**P3 TaskFlow (Middle):** users/workspaces/projects/tasks/comments, RBAC, transactions, PostgreSQL, background jobs, разумное Redis/queue применение, concurrency-safe jobs, rate limits, observability, CI, Docker, load/security tests. Не добавляй микросервисы без необходимости.

**P4 ScaleLab (Middle+):** performance baseline, profiling, advanced SQL/cache/replication or queue designs, capacity/SLO, fault injection, измеренный before/after, incident postmortem.

**P5 Architecture & Systems Capstone (Senior):** неполные требования, design document, ADRs, отказоустойчивость, компромиссы, критические прототипы, тесты, review, эксплуатационный план. Этот проект проверяет инженерные решения, но не заменяет командный производственный опыт.

У каждого проекта: starter (без готового решения), requirements, acceptance tests, этапы, checkpoints, тесты, quality rubric, полное проверенное reference solution отдельно, экспорт в Git-ready/ZIP и защита.

### E5. Обновляемость курса

Машиночитаемый manifest: `go_version`, `dependency_versions`, `curriculum_version`, `last_verified_at`, `source_urls`, `compatibility_notes`. CI проверяет Go examples и solutions; при новых релизах создаёт audit tasks/PR, но **не совершает автоматически breaking upgrade без тестов**. Контент версии должен быть доступен для повторной сборки.

---

## F. AI Tutor и экономия моделей

### F1. Два независимых слоя

**Development orchestration:** экономное использование моделей Codex при создании приложения. Настрой `.codex/config.toml` (проверь действительность ключей) и `.codex/agents` для ограниченных специализированных ролей. Не более 3 субагентов одновременно по умолчанию. Исследователь/черновик/простое изменение — экономичная модель; архитектура, безопасность, критическое ревью, сложная интеграция — сильная. Если нужная модель отсутствует, найди доступную замену; не обходи ограничения и не включай платные API без разрешения.

**Product AI Gateway:** отдельная система, которой пользуются студенты. На выбор локальные/облачные OpenAI-compatible провайдеры и другие поддерживаемые интеграции; без ключа всё остальное должно работать.

### F2. Возможности Tutor

Режимы: explain / simpler / deeper / example / hint / debug / code review / quiz / interview / architecture review / study plan. Запросы на русском, терминология английская + перевод. AI получает только минимально нужные данные: `lesson_id`, `exercise_id`, исходники задания (с учётом разрешения), compiler diagnostics, результаты тестов, текущие навыки, краткий progress snapshot. Для личных данных — информированное согласие и минимизация отправляемого текста.

Никакого полного решения при запросе «подсказка», никакого уверенного заявления «код выполняется» без запуска. AI не ставит Verified Rank и не подписывает сертификаты напрямую.

### F3. Routing policy

| Запрос | Первичный исполнитель | Эскалация |
|---|---|---|
| Проверка stdout/tests/format | Go compiler/test/gofmt | Без LLM |
| Прогресс, XP, ranking | Серверные детерминированные правила | Без LLM |
| Определение термина | Закэшированный контент → экономичная локальная/облачная модель | Если не помогло |
| Подсказка по уроку | Экономичная модель + ограниченный контекст | Сильнее при низкой уверенности |
| Объяснение compiler error | Результат компилятора + шаблон → экономичная модель | Сложный случай |
| Архитектурное review, concurrency bug, сложное interview | Более сильная модель | При необходимости человек |

Реализуй: бюджеты, лимиты, request timeout, retry/backoff, fallbacks, token usage accounting, approximate cost labeling, cache только для безопасных детерминированных/безличных результатов, provider health, messages when quota exhausted, настройки приватности и запрет неожиданных расходов. Отделяй подписку ChatGPT/Codex от usage конкретного провайдера — они не взаимозаменяемы автоматически. Секреты никогда не попадают в публичные пакеты, логи, prompt dumps или frontend. Режим no-AI — полноценный функционал.

---

## G. XP, ранги, экзамены и проверяемые сертификаты

### G1. Две оси прогресса

- **Learning XP:** мотивация за выполненные действия, повторы, проекты, challenges. Защита от бесконечного накручивания однотипными кликами.
- **Verified Rank:** присвоение только после документированных требований + проверенных компетенций + capstone/exam; отделён от XP, streak и мнений AI.

Ранги: Zero, Junior, Junior+, Middle, Middle+, Senior. Для каждого отдельные критерии, assessment rubrics, fail/retry behavior, traceable evidence. Разрешено пропускать уже знакомые уроки; выдача звания без прохождения итоговой проверки запрещена.

### G2. Экзамены

Смешанные форматы: вопросы с объективным ответом, live coding, debugging, SQL, unfamiliar code review, проектные задачи, архитектурные ADR и устная/письменная защита там, где можно. Для самостоятельной платформы отделяй автоматически проверяемое от требующего человека. Отказ в повышении должен быть объяснимым. Добавь регрессионные тесты на обход требований через API/клиент.

### G3. Сертификаты и badges

Генерация PDF/PNG, уникальный ID, дата/версия criteria, issuer, student display name (либо псевдоним), выбранные verified competencies, QR на `/verify/:credentialId`, понятная страница проверки, expiry/revocation status при применимости, audit trail. Исследуй Open Badges 3.0/Verifiable Credentials и используй стандартную схему подписи с ключами вне репозитория. **Никаких поддельных подписей людей или аккредитаций.** До сертификации соответствия спецификации не заявляй полную совместимость.

### G4. Границы доверия

- **Local self-issued certificate:** достижение локального обучения; защищённая на уровне установки подпись не равна доверенной внешней аттестации.
- **Hosted verified credential:** только доверенный оператор платформы с контролируемыми ключами и серверными аттестациями может выдавать публично проверяемый результат; сам факт валидной подписи не доказывает идентичность человека или отсутствие списывания.
- Privacy by default: публичный `Developer Passport`, имя и сертификаты — только с согласия. Возможны отзыв и удаление персональных данных в рамках выбранного режима.

### G5. Developer Passport / gamification

Страница личного развития: ранги, навыки, проекты, GitHub links, сертификаты, progress graph на реальных данных, достижения. Rank-up animation ненавязчивая, учитывает reduced-motion. Achievement badges конкретны (Hello World, Bug Hunter, API Builder и т.д.) и выдаются за реальные события. Не стимулируй бессмысленный grind; система не должна отвлекать от учёбы.

---

## H. Архитектурные и эксплуатационные требования

### H1. Предпочтительная стартовая форма

После Phase 0 выбери оптимальную архитектуру. Предположительно подходит **modular monolith**: Go API, TypeScript frontend, PostgreSQL, версионированный Markdown/MDX/structured content, отдельный безопасный Go runner, AI gateway. Но это гипотеза, не окончательный стек: обоснуй выбор в ADR. Избегай сети микросервисов, если она мешает обучению и локальному развёртыванию.

### H2. Сущности данных (минимум)

`User/Profile`, `CourseVersion`, `Rank`, `Module`, `Lesson`, `Exercise`, `TestCase` (public/private), `Submission`, `Run`, `Evaluation`, `Project`, `ProjectFile` / archive, `LessonProgress`, `CompetencyEvidence`, `SkillRating`, `Exam`, `ExamAttempt`, `RankAward`, `Achievement`, `XpEvent`, `Credential`, `CredentialRevocation`, `ProviderConfiguration`, `AIUsage`, `AuditEvent`. Миграции обязательны, модель нормализуй осмысленно. Сам student-generated source должен сохраняться без утечек и с лимитами размера.

### H3. API и безопасность

Контракты API (OpenAPI where useful); строгая валидация и ограничения размера, authentication/authorization, защита сессий/cookies/CSRF по реальному способу авторизации, rate limiting, логирование без секретов, дедлайны, тесты на IDOR и эскалацию привилегий. Никогда не доверяй клиентской информации об успешной сдаче.

### H4. Local-first и self-hosting

Стандартный путь запуска без платного облака, с контейнерами/скриптами и документированными командами для Linux/Windows/macOS. Режим без AI ключа. Persistent storage, migrations, backups, health/readiness checks, safe shutdown, sample env без секретов, разумные дефолты. Деплой public multi-user описывай отдельно: TLS, backups, secrets, rate limits, hardened sandbox и инфраструктурные расходы.

### H5. Open-source governance

README на русском, Getting Started, CONTRIBUTING, SECURITY, CHANGELOG, architecture decisions, guides for writing lessons and adding tests, issue/PR templates, .env.example. Предложи лицензии для исходников и оригинального обучающего контента раздельно, но выбор за владельцем проекта; не заявляй уже установленную лицензию до решения. Не включай несвободные материалы или шрифты без соответствующих прав. Уроки и примеры должны работать без внешних учебных сайтов; ссылки — только как необязательные дополнительные ресурсы.

---

## I. Автоматизация качества и тестирование

Пользователь **не должен вручную тестировать сайт**. Автоматизируй максимальное покрытие в доступной среде; тестирование снижает риск, но не даёт абсолютной гарантии отсутствия ошибок.

### I1. Testing layers

- **Go unit/integration:** `go test ./...`, `go test -race ./...` где применимо, `go vet`, static analysis, migrations, repository/API tests, persistence, auth, rank policy, signed credential issuance/revocation.
- **Frontend:** TypeScript typecheck, lint, build, component tests, forms, editor actions, state restoration, routing, keyboard support.
- **Runner:** нормальный запуск, compile error, time/memory/output limit, infinite loop, PID limits, blocked network/host access, sandbox lifecycle/cleanup.
- **Content:** для **каждого** Go example compile/run (в зависимости от типа), reference solutions должны проходить public/hidden tests, явно ошибочные — отклоняться. Markdown links/content schema, glossary/dependency consistency, учебный уровень prerequisites.
- **Playwright E2E:** local profile → roadmap → lesson → edit → run → compiler error → fix → submit → tests → progress persists → project files → exam pass/fail → rank → certificate → verification → privacy controls; no-AI and mocked provider failover.
- **Visual:** screenshots desktop/tablet/mobile, light/dark, editor overflow, long prose, empty/loading/error states, keyboard focus; baseline and regression.
- **Security:** authorization negatives, input sanitization, path traversal, XSS, CSRF where applicable, rate limits, sandbox isolation, credentials trust boundaries, secret leakage and dependency audits.
- **Accessibility:** semantic labels, focus order, contrast, reduced motion, keyboard-only flows; target current relevant WCAG guidance.

### I2. Evidence

Сохраняй `docs/VALIDATION_REPORT.md`: дата, среда, команды, pass/fail/skip, ссылки на логи/screenshots/CI artifacts, непроверенные утверждения, следующие исправления. Нельзя отмечать «passed», если тест не выполнялся. Исправление ⇒ regression test ⇒ повторная проверка. Публикация блокируется известными критическими дефектами.

### I3. Acceptance matrix

Нужна матрица `requirement_id / implemented / tested / evidence / limitations`. Проверяй **не только наличие элемента UI, но и его работу**. Критерий готовности контента — полноценные проверенные главы *всех* шести уровней; название раздела не считается готовым уроком.

---

## J. Операционный порядок работы Codex

1. Прочитай файлы инструкции/ТЗ/плана/handoff, проверь Git status, права и доступные инструменты.
2. M0 Research: официальные версии Go, ecosystem, security, credentials, Codex/model availability, stack ADR.
3. Выполняй `docs/EXECUTION_PLAN.md` milestone-by-milestone, начиная с рабочего end-to-end learning loop, а не с декоративного UI.
4. Делегируй только независимые узкие задачи; не конкурируй за файлы. При сложных решениях или security review используй сильную доступную модель; по обычным задачам — экономную. Не обещай экономию без фактического routing.
5. После каждого milestone запускай тесты, исправляй и сохраняй состояние в `AGENT_HANDOFF.md`; при наличии Git фиксируй разумные локальные checkpoints. Не зацикливайся на перепроектировании без конкретных доказательств.
6. При нехватке доступа к сети или инструментам оставь корректный blocked статус и документируй ограничение; сделай остальные независимые задачи без имитации результатов.
7. Финальное сообщение должно содержать что реально построено, команды запуска, результаты тестов, оставшиеся ограничения/риски и что требуется от владельца перед публичным релизом. Без обоснования не объявляй полностью готовым курс, где написана только первая неделя.

### Definition of Done (общая)

- Приложение запускается с чистой установки, рабочий сквозной путь урок→редактор→безопасный запуск→проверка→progress подтверждён E2E.
- Реализованы и проверены карьерная карта, уровни, упражнения, проекты, экзамены, XP/rank, сертификаты с честной моделью доверия, no-AI mode и optional AI Tutor.
- Полное оригинальное учебное содержимое по шести рангам не placeholder: объяснения + работающие примеры + задания + self-check + tests + projects.
- Выполнены тесты, аудиты ключевых угроз и визуальная проверка; документированы риски публичного запуска.
- Подготовлены README/локальный запуск/CI/версии/лицензирование/принципы участия; код и содержание можно сопровождать.
- Полнота подтверждена фактическими доказательствами, а не утверждением агента.

---

## K. Первая команда агенту

**Прочитай этот MASTER_SPEC вместе с AGENTS.md, EXECUTION_PLAN.md и AGENT_HANDOFF.md. Сначала завершай документированную проверку актуальности технологий. Затем без промежуточных пользовательских согласований реализуй очередные milestones, запуская тесты и обновляя handoff. Не своди продукт к чату, макету или курсу из одних заголовков. Не публикуй проект и не трать деньги без разрешения.**
