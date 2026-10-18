# Open Go Academy — Autonomous Execution Plan

**Состояние:** см. актуальные статусы и датированные доказательства ниже.  
**Иерархия:** [MASTER_SPEC](MASTER_SPEC.md) задаёт продукт; **этот файл** — последовательность задач/критериев; [AGENT_HANDOFF](../AGENT_HANDOFF.md) — проверенное текущее состояние.  
**Правило:** не меняй цель проекта ради простого `green check`; не помечай этап `DONE` без указанных доказательств.

## 0. Как пользоваться этим планом

Статусы: `NOT STARTED`, `IN PROGRESS`, `BLOCKED`, `VALIDATING`, `DONE`. Замени статус в таблице и добавь датированную заметку по каждому завершённому milestone. При создании нового функционала сначала обозначь проверяемые acceptance criteria. Работай сквозными пользовательскими сценариями. Задачи не обязаны выполняться за одну сессию; сессии могут продолжаться по handoff.

| ID | Milestone | Статус | Доказательства |
|---|---|---|---|
| M0 | Research, stack, threat analysis, Codex verification | DONE | 2026-10-15: docs/research/*, ADR-0001, THREAT_MODEL; official JSON/npm/schema и rootless cgroup probe |
| M1 | Monorepo, dev environment, API/frontend/database startup | DONE | 2026-10-15: go test/vet/race, frontend typecheck/build, actual restart-check.py, desktop/mobile Playwright smoke |
| M2 | Content schema, reader, first complete Zero lesson | DONE | 2026-10-15: embedded catalog validation, 2 Go snippets + 3 reference runs, safe Markdown + axe desktop/mobile/light/dark |
| M3 | Isolated Go runner, public/hidden evaluation | DONE | 2026-10-15 local-only: real rootless suite11 cases PASS94.60s, SIGKILL/restart PASS, independent findings resolved |
| M4 | End-to-end interactive lesson + persistence | DONE | real Go golden-path browser PASS, XP replay/noAI/reload, network write recovery desktop/mobile PASS |
| M5 | Roadmap, rank structure, XP and skills | DONE | six server rank criteria, verified attempt-linked skills, search/review, unit/race + E2E7 PASS1skip |
| M6 | Project workspaces and tests | DONE | P1 starter/reference, files/snapshots/ZIP + realGoRun/Test/15cases; reference/unit/race, browser+axe, independent review resolved |
| M7 | Exam engine, earned-rank logic, negative tests | DONE | 2026-10-15: native exam3PASS1skip; CAS/replay/award unit/race + independent review; full14PASS/5skip, stale P1 assertion corrected+target1PASS1skip |
| M8 | Optional AI Tutor, model routing, usage controls | DONE | 2026-10-15: Go unit/vet/race+TS/build; actualHTTP test-provider browser3PASS1skip, noAI desktop/mobile, consent/fallback/quota/privacy review |
| M9 | Digital badges, certificates, verification, passport | DONE | 2026-10-15: native exam→credential→verify/revoke/privacy; PDF/PNG/QR+independent crypto, actual keyrecovery; full17PASS5skip6min, graph/project/badgedates tested |
| M10 | Full Zero/Junior/Junior+ curriculum and projects | DONE | 2026-10-15: 40chapters/120exercises;117nativefunctionsPASS;P2native44checks/recovery;Junior→Junior+nativeawards/credentialsPASS64.3s;full27PASS7skip7.6min;unit/vet/race/TSbuildPASS |
| M11 | Full Middle/Middle+/Senior curriculum and projects | IN PROGRESS | 2026-10-18: all-88 technical gate v2 **PASS** in 3009.434s on old fingerprint `sha256:7600aec...`; v2 no longer binds the final tree. Pedagogical corrections cover Russian-first outcomes, unique self-checks, P3 practice bridges, executable PostgreSQL/Redis CLI labs, first-use glossary repair, and topic-specific Middle+13–23 explanations. Local content manifest includes all 128 Zero→Senior chapters. `scripts/test_m11_curriculum_quality.py` enforces 88 lesson lengths, unique outcomes/self-checks and course-specific practice scaffolding. Current full `make test` passed before the last projectrunner guard regression fix. Exact-current-tree all-88 gate v3 is **RUNNING** on fingerprint `sha256:c9c85f0f9b6d2622bc29cefad12e2feac1b349950f1b7c3e579a05ed556af2c0`; at 2026-10-18 11:18 Asia/Atyrau its log had reached Middle-35, with no final result yet. All 13 canonical skills have course/project or exact awarded-Senior human-review evidence; communication never comes from automatic P5. Final reviewer/rank/traceability acceptance remains open. See `docs/research/M11_SKILL_MAPPING_20261018.md` and `docs/research/M11_PEDAGOGY_REVIEW_20261018.md`. |
| M12 | Hardening, full E2E, performance, visual and accessibility | IN PROGRESS | 2026-10-18: mixed desktop/mobile suite 40 passed, 10 intentional skips; progress dashboard test passes desktop light/mobile dark with axe/no overflow. New `tests/m12-responsive-a11y.spec.ts` catches tablet overflow/contrast defects; after CSS repair and production rebuild, focused Playwright PASS2 desktop/mobile at 768px + 1920px, axe clean, keyboard visible focus and reduced motion verified (`docs/validation/m12-responsive-a11y-20261018.txt`). Disposable API load/restart probe passes 120 reads/119 writes +21 quota429 and persistence; not an SLO. `make sandbox-test` PASS144.479s. P4/P5 assessment exit guard hardened for process exits; M12 independent audit still requires integration run of new `scripts/m12_sandbox_limit_probe.py` (exact policy, rootless Podman, `memory.events`/`pids.events`) and broader viewport/theme/failure matrix/full final desktop-mobile E2E. Probe has 10/10 unit tests and read-only reviewer approval; no cgroup integration result yet. Existing runner memory/PID false-positive assertions still require correction after current M11 gate and a new fingerprint-bound M11 run. Prior full `make test` predates the last guard change. |
| M13 | Open-source release preparation and final acceptance audit | IN PROGRESS | 2026-10-16: exact `2b00dfc` fresh archive bootstrap/build, runner profiles, SQLite recovery and complete 48-test desktop/mobile E2E passed; 38 passed, 10 intentional skips, 0 failures. Clean-cache full Go tests/vet/race/TS/build passed on prior checkpoint `746bbd5`; `2b00dfc` changed only tests/docs. P1–P5 actual runners and mobile skills-after-P5 state passed. 2026-10-18: cache-only scanner/test suite produced `docs/validation/m13-license-inventory-20261018.json` (339 records, 256 license-text files, 125 distinct hashes); Go SPDX/legal review remains open, project is `UNLICENSED`. Added honest Windows/WSL 2 and macOS/Linux VM installation pathways with official Microsoft/Podman sources; neither non-Linux route is validated and sandbox must remain disabled on failed preflight. Still open: current-tree clean archive acceptance, requirement traceability, pedagogy/governance, secret tooling, actual license/NOTICE review, and owner choice of project license. See `docs/validation/m13-clean-source-acceptance-20261016.txt`, `docs/INSTALL_WINDOWS_WSL.md`, `docs/INSTALL_MACOS.md`. |

### Правила сотрудничества субагентов

- Главный Codex ведёт контракты и архитектуру, интегрирует результаты, отвечает за QA и фиксирует решения.
- Делегируй узкие задачи: verified release research, один UI-компонент, один endpoint/test suite, отдельная учебная глава, изолированное ревью.
- Перед делегированием сообщай входные данные, ровно какие пути агент вправе менять, expected output, acceptance test и что запрещено.
- Не более 3 параллельно; запрещено конкурирующее редактирование одного файла. Высокорисковые security и concurrency области требуют ревью не автором кода.
- После слияния параллельных результатов запускай интеграционные проверки; немедленно устраняй конфликт API/contract.
- Повторное использование уже полученного исследования предпочтительнее нового дорогого вызова. Контролируй рост расхода при parallelization.

---

## M0 — Research, compatibility & safety decisions

**Путь:** `docs/research/*`, `docs/architecture/ADR-0001-tech-stack.md`.

### Работы

1. Зафиксируй состояние Git, OS, Go/Node/Docker/Compose/CI/browser tools, доступ к сети, режим запуска Codex.
2. По официальным источникам проверь актуальный Go, release notes, стандартную библиотеку, поддерживаемые версии.
3. Исследуй альтернативы HTTP, Postgres driver, database tooling, TS frontend, editor, test runner, sandbox, model routing, digital credentials. Не пытайся интегрировать всё.
4. Проверь `.codex/config.toml` и custom agents по действительной локальной версии Codex; выясни доступность `gpt-6-luna` и `gpt-6.1-sol`. При недоступности настрой существующие модели и задокументируй замену. Не подменяй факт доступности текстом в промпте.
5. Создай ADR: архитектурная гипотеза, обоснование, реальные версии, lock strategy, риск-карта и public-hosting restrictions.
6. Создай `docs/research/VERSION_MATRIX.md` с `verified_at`, `source URL`, `selected version`, `why`; `docs/architecture/THREAT_MODEL.md`.

### Gate

- [ ] Источники с датами/URL доступны в репозитории; не выдуманы.
- [ ] Отсутствие сети или неподдерживаемые модели честно задокументированы.
- [ ] Выбран MVP-срез: урок + редактор + тесты + хранение; публичное исполнение кода не включено до sandbox hardening.
- [ ] Выбран стек и зафиксирован ADR.

---

## M1 — Reproducible foundation

**Цель:** реальный воспроизводимый запуск приложения, не демостраница.

### Работы

1. Создай согласованную монорепозиторную структуру, lockfiles, version manifests, `.env.example`, `.gitignore`, конфигурацию форматирования и CI.
2. Настрой frontend, Go API, PostgreSQL или выбранное обоснованное хранилище, миграции, конфигурацию, health endpoints, shutdown, логирование.
3. Создай безопасный локальный профиль и серверное сохранение предпочтений; предусмотри путь добавления multi-user auth.
4. Добавь команды bootstrap/dev/test/build; Docker Compose там, где это уместно.
5. Проверяй readiness и миграции на чистой базе, затем restart и сохранение данных.

### Gate

- [ ] Fresh install запускается по README.
- [ ] Backend/frontend build/tests проходят, ping/health проверены реально.
- [ ] DB migrations на пустой БД и рестарт проходят.
- [ ] Browser smoke test работает, runtime errors/404/500 нет на основном пути.

---

## M2 — Course content engine and first finished lesson

### Работы

1. Составь schema учебного контента (`rank`, `module`, `lesson`, `prerequisites`, `go_version`, `glossary`, `exercise_id`, `tests`, `difficulty`, `estimated_minutes`, `learning_outcomes`).
2. Создай механизмы импортирования, версионирования, поиска/навигации, Markdown rendering с безопасной обработкой HTML и code blocks.
3. Реализуй страницу урока (desktop/mobile), progress marker и ссылки на зависимости.
4. Напиши **полный** Zero/Day 1: «Что такое компьютер, программа и Go», объяснение с нуля, иллюстрации, реальные примеры, упражнения и self-check.
5. Добавь контентные валидации: уникальные ID, отсутствие сломанных внутренних ссылок, dependency cycles, все referenced exercises exist.

### Gate

- [ ] Первый урок — законченная самостоятельная глава, не TODO.
- [ ] Markdown и код читаются на desktop/mobile, рендер безопасен.
- [ ] Все внутренние ссылки и schema validation проходят.

---

## M3 — Secure code execution & grading

### Работы

1. Разработай execution protocol API↔runner, allowlisted Go version/images, job lifecycle, rate/resource limits and timeout.
2. Протестируй выбранный sandbox для угроз local/public. До подтверждения sandbox **не включай public multi-user code execution**.
3. Поддержи `main.go`, функцию/пакет и несколько файлов. stdout/stderr/exit status, errors, timeouts.
4. Добавь grader public/hidden tests с отделением доверенных тестов от source пользователя, корректные языковые ошибки и без утечки эталонных решений.
5. Negative tests: бесконечный цикл, чрезмерная память/вывод, network, forbidden path, process spawn, sandbox cleanup, concurrent requests.

### Gate

- [ ] Реальный Go-код выполняется из браузерного API в изоляции.
- [ ] Корректные программы возвращают реальный вывод, некорректные — реальные ошибки компиляции.
- [ ] Hidden tests недоступны клиенту до выполнения, контрольные неверные решения отклоняются.
- [ ] Ограничения ресурсов протестированы; угрозы и ограничения документированы.

---

## M4 — Golden-path complete learning loop

### Работы

1. Интегрируй Monaco/CodeMirror, multi-file storage, autosave, Run/Submit/Reset, console/test results.
2. Добавь `LessonProgress`, `Submission`, `Evaluation`; прогресс вычисляется из серверных событий.
3. Реализуй student journey первого урока и первого упражнения, прерывание сети, обновление страницы и восстановление кода.
4. Тестируй без LLM и без обязательного внешнего API; решай loading/error/empty states.

### Gate (обязательный E2E)

- [ ] Start → read lesson → edit Go → Run → output → Submit → public/hidden tests → passed status → dashboard progress.
- [ ] Fail submission не присваивает mastery; данные переживают restart.
- [ ] E2E Playwright с screenshot/trace, а не фиктивное сообщение «works».

---

## M5 — Career map, XP & skills matrix

### Работы

1. Создай шесть учебных рангов с компетенциями и dependencies (не выдавай звания автоматически только за наличие контента).
2. Добавь интерактивный Career Path, Course Explorer, Dashboard, реальные completion statistics.
3. XP начисляется за серверно проверяемые события с защитой от повторного начисления.
4. Skills Matrix 0..4 опирается на evaluation evidence; состояние skipped ≠ mastered.
5. Добавь подсказку «следующий шаг» и раздел повторения сложных тем, без фальшивой персонализации.

### Gate

- [ ] Все 6 уровней показываются, даже ещё не наполненные разделы явно помечены как неготовые, не считаются завершёнными.
- [ ] XP невозможно накрутить простым повторным POST того же события.
- [ ] Rank не зависит от XP; direct client tampering отклоняется.

---

## M6 — Projects workspace

### Работы

1. File tree, multi-file editing, starter templates, project steps/acceptance, code execution/test integrations, milestones, ZIP/Git-ready export.
2. Подготовь архитектуру для проектов с PostgreSQL/Docker локально; не запускай произвольный пользовательский Docker Compose на shared server.
3. Создай проверенный starter и baseline reference проекта P1; интегрируй критерии завершения.
4. Сохраняй файлы по пользователю и version snapshots, разумные квоты на объём.

### Gate

- [ ] Создание/изменение/сохранение/экспорт нескольких файлов работает.
- [ ] Проектный Go-код и тесты действительно выполняются.
- [ ] Невыполненная часть проекта не отображается завершённой.

---

## M7 — Exams and trusted rank policy

### Работы

1. Objective quiz, coding, debugging and rubric-driven exam (часть может требовать ручной оценки — обозначь).
2. Server-side rules and evidence linking; exam_attempt, evaluation, retake, grade thresholds with versioned criteria.
3. Незнакомые задачи, публичные примеры и скрытые проверки, ограничения на раскрытие ответов.
4. Политика earned ranks independent of XP; прохождение модулей без экзамена не повышает verified rank.
5. Тесты на negative cases: поддельный запрос pass, обход prerequisites, изменение чужого профиля, replay.

### Gate

- [ ] Прохождение/провал проверяются детерминированно; состояние устойчиво.
- [ ] Клиент не может присвоить себе rank.
- [ ] Есть отчёт о проверенных и субъективных критериях.

---

## M8 — AI Tutor & usage controls

### Работы

1. Provider-neutral AI Gateway, optional keys/localhost models, routing by complexity, retry/fallback/timeout and budget policy.
2. Tutor modes: explain, simpler, deeper, hint, debug, review, quiz, interview, study plan. Russian responses with translated terminology.
3. Контекст только релевантный урок/code/errors/progress — согласие/приватность. Не отправляй секреты, hidden tests или чужие профили.
4. Mock provider tests, model failover, quota exhaustion, redaction, no-AI mode.
5. Модель не принимает серверные решения об экзаменах/ranks/credentials; deterministic tools have priority.

### Gate

- [x] Без моделей весь основной учебный сценарий проходит.
- [x] Mock HTTP provider даёт реальный ответ; local/cloud settings и graceful failure проверены.
- [x] В UI есть usage/budget и объяснение расхода; перед отправкой действует отдельное согласие, контекст раскрыт, ключи не возвращаются.

---

## M9 — Badges, certificates & passport

### Работы

1. Achievement rules основаны на реальных событиях.
2. Выпуск образовательных credential только после verified assessment, уникальный ID, policy version, revocation, audit.
3. PDF/PNG/QR, public `/verify/:id`, контроль статуса криптографической подписи где реализовано.
4. Отдельное разграничение `self-assessed locally` vs `trusted hosted issuer`; никакого придуманного преподавателя и ложной аккредитации.
5. Пользовательский Developer Passport и privacy controls; opt-in public profile.
6. Изучи стандарты, реализуй только реально проверенные части; Open Badges compliance подтверждается тестом схемы/крипто, иначе формулировка «совместимость не подтверждена».

### Gate

- [ ] Невыполнивший критерии не может получить certificate; status/verification работает.
- [ ] Отозванный/поддельный/изменённый credential не проходит проверку.
- [ ] Публичная персональная информация скрыта до opt-in.
- [ ] PDF/PNG и QR реально генерируются и проверены.

---

## M10 — Curriculum production: Zero → Junior+

### Работы

1. Сначала создай dependency graph и detailed syllabus (module/day/lesson IDs) для трёх рангов и сдержанный age-agnostic curriculum design.
2. Последовательно наполняй **все** главы: полноценная оригинальная теория + примеры + упражнения + решения + public/hidden tests + glossary.
3. Реализуй P1 CLI Toolkit, P2 LinkHub API с starter, reference, tests, rubric, documentation.
4. Интегрируй все главы в существующие content API и UI.
5. Автоматически компилируй Go snippets/reference, проверяй links, schema, prerequisites; отдельно выборочно делай педагогическое и техническое review.

### Gate

- [x] Ни одна обязательная глава этих рангов не состоит лишь из заголовка или TBD.
- [x] Референсные решения компилируются/проходят тесты.
- [x] Уроки доступны и корректно отмечают прогресс; все core project workflows работают.

---

## M11 — Curriculum production: Middle → Senior

### Работы

1. Полностью распиши/напиши Middle, Middle+ и Senior как разные компетентностные уровни.
2. Реализуй P3 TaskFlow, P4 ScaleLab, P5 Architecture Capstone с заданиями, starter/reference и objective/rubric checks.
3. Для high-load/production lessons включи измерения, профилирование, before/after, meaningful trade-offs, deployment risk.
4. Для Senior включи design docs, ADR review, incident/mentoring scenarios; не представляй учебную симуляцию как годы опыта в компании.
5. Проверяй совместимость с версией Go/стеком и самодостаточность материалов.
6. Добавь immutable exam/rank policy для Middle, Middle+ и Senior. Каждый award требует prior verified rank, обязательные course competencies, соответствующий P3/P4/P5 project evidence и все objective exam parts; субъективные P5/design-defense критерии должны оставаться pending до настоящего human review и не могут закрываться ответом ИИ или self-review.

### Gate

- [ ] Программа всех шести уровней завершена содержательно.
- [ ] Projects P3–P5 и их проверки/критерии функционируют в документированном сценарии.
- [ ] Версионированные экзамены и серверная rank eligibility покрывают все шесть уровней; award атомарно ссылается на прошлые rank/course/project evidence, а ручная часть видна как pending.
- [ ] Глоссарий/skill mappings complete, нет critical instructional gaps.

### Актуальная трассировка навыков и проверок (2026-10-18)

- Полный source-bound all-88 gate v2 запущен после M28 и Middle+21 Easy fixes; до выхода процесса он не считается PASS для текущего дерева. Exec session `55884`; fingerprint и команда указаны в `AGENT_HANDOFF.md`.
- Полный `make test` после M28/M+21 fixes — PASS: Python fingerprint, Go unit/integration/P1 reference/vet/race, TS typecheck и production build. Scoped M28/M+21 acceptance также PASS. Новый P3 manual-rubric UI test: 2/2 desktop/mobile Playwright, axe violations 0, горизонтального overflow нет.
- `M11_SKILL_MAPPING_20261018.md` обновлён: 13 canonical IDs имеют server-verified course/project evidence либо точный P5 human-review artifact после Senior award. `communication` не выводится из автоматической проверки; latest-review проверки остаются обязательными до выдачи award, а принятый award использует свой неизменяемый snapshot.
- Версия toolchain повторно подтверждена 2026-10-18: Go 1.27.2 остаётся последней стабильной версией по официальной истории релизов; текущая локальная установка та же.

---

## M12 — Security, performance, accessibility & E2E audit

### Работы

1. Прогони полный тестовый набор после развёртывания из чистой установки, миграций и seed контента.
2. Нагрузочные проверки runner, API, persistence, очередей, race/deadlock, resource abuse.
3. Playwright screenshots: мобильный/планшет/desktop/wide, светлая/тёмная темы, длинный текст и код, loading/error states.
4. Accessibility: keyboard/focus, contrast, labels, reduced motion.
5. Security review: sandbox escapes/host access, XSS, CSRF, IDOR, credential forgery, secret scanning, dependencies.
6. Fix-and-retest циклы, regression tests, audit log и открытые риски.

### Gate

- [ ] Golden-path browser tests проходят стабильно, failure cases имеют репродукторы.
- [ ] Нет критических/высоких известных проблем на заявленном режиме эксплуатации.
- [ ] Реальные test commands/output/CI artifacts перечислены в `docs/VALIDATION_REPORT.md`.

---

## M13 — Release preparation & final acceptance

### Работы

1. Подготовь README, GETTING_STARTED, docs для Windows/Linux/macOS, contributor/security guides, architecture/runbook, release notes, samples, `.env.example`.
2. Чистый clone → documented install → data migration → lesson/run/grade/progress → projects/exams/credentials demo.
3. Составь Requirement Traceability Matrix: implementation path + automated test + status + caveat.
4. Проверь секреты, licenses, third-party assets, reproducible builds, offline no-AI mode, version manifest, safe deployment default.
5. Никаких публичных публикаций, paid provisioning и настоящих issuer certificates от имени другого лица без разрешения.

### Gate

- [ ] Документация позволяет воспроизвести запуск.
- [ ] Финальный аудит показывает **реальное** покрытие и known limitations.
- [ ] Full-course completion отмечен только при реально написанных/проверенных материалах всех рангов.
- [ ] Агент оставил точную инструкцию продолжения/публикации, если необходимы внешние разрешения.

---

## Сквозные E2E критерии (обновлять по мере реализации)

| ID | Сценарий | Milestone |
|---|---|---|
| E2E-01 | Onboarding → dashboard → Career Path | M5 |
| E2E-02 | Open lesson → code edit → Run → real stdout | M4 |
| E2E-03 | Compiler error → code correction → Run | M4 |
| E2E-04 | Submit → public/hidden tests → progress | M4 |
| E2E-05 | Reload/restart → preserved code, XP, progress | M4–M5 |
| E2E-06 | Project multiple files → tests → export ZIP | M6 |
| E2E-07 | Exam fail → rank denied | M7 |
| E2E-08 | Exam pass + competency evidence → rank awarded | M7 |
| E2E-09 | Earn badge → PDF credential → QR verify | M9 |
| E2E-10 | Credential tampered/revoked → verification fails | M9 |
| E2E-11 | No AI provider → learning path fully usable | M8 |
| E2E-12 | Mock AI provider failure/quota exhausted → graceful fallback | M8 |
| E2E-13 | Private passport remains private | M9 |
| E2E-14 | Mobile/tablet/desktop & light/dark UI | M12 |
| E2E-15 | Content examples and references pass validation across six ranks | M10–M12 |

---

## Обязательный checkpoint после каждого milestone

Записывай в `AGENT_HANDOFF.md`:

```text
Completed milestone:
Files changed:
Architecture decisions:
Commands executed (exact):
Tests passed (actual evidence):
Tests failed/skipped (reasons):
Open risks:
Next milestone and first concrete task:
```

Если пользователь просит продолжить проект в новой сессии, сначала используй это состояние, затем Git status/текущие файлы, не повторяй все этапы.

## Финальный отчёт Codex

В конце одной агентной сессии (даже если проект ещё не готов) кратко укажи: реально выполненные milestones, запуск/демо, успешные проверки, заблокированные функции, остаточные риски и **один следующий конкретный шаг**, записанный в handoff. Нельзя сообщать, что «готов полностью», пока M0–M13 не имеют проверенных доказательств.

## Checkpoint 2026-10-15 — M0

Документальная проверка первичных источников завершена. Последний stable Go 1.27.2; официальный архив загружен в ignored .tools и SHA256 сверен с go.dev. Exact npm versions/engines сохранены. Codex schema ключи валидны; реальные researcher/reviewer работают на configured моделях; builder проверяется M1. Rootless Podman probe реально показал uid65532, memory.max=536870912, pids.max=64, cpu.max=100000 100000. Не является готовым M3 sandbox audit.

Уточнение плана: выбран SQLite вместо предполагаемого PostgreSQL для local single-user; hosted PostgreSQL требует отдельного этапа auth/migrations. М0 определяет ограничения, а не утверждает безопасность public execution.

## Checkpoint 2026-10-15 — M1 и M2

M1: создан local monorepo, Go API/SQLite numbered migration/profile/server theme/drafts, exact dependency locks, Make commands, loopback Host/Origin/session/CSRF guards, health/shutdown. `go test ./...`, `go vet ./...`, `go test -race ./...` выполнены на .tools/go1.27.2; frontend typecheck/build PASS. `python3 scripts/restart-check.py` реально остановил/перезапустил API, профиль/тема/черновик сохранились. Playwright smoke desktop/mobile PASS; первоначальные contrast/focus defects исправлены и повторный axe прошёл в обеих темах. CI подготовлен (official Actions releases pinned SHA), на GitHub не выполнялся и не публиковался.

M2: полный Zero Day1, 3 упражнений, glossary, beginner types/function diagnostics, self-check с явным раскрытием, reference solution endpoint только по запросу. `go test -count=1 ./internal/store ./internal/content` проверяет schema/dependencies/cycles/links, реально компилирует/исполняет 2 Go snippets и 3 solutions. Browser безопасно отображает Markdown; desktop/mobile screenshots визуально просмотрены. Это не полный Zero-track.

## Checkpoint 2026-10-15 — M3/M4

M3 local-only: отдельный Unix-socket runner, rootless Podman image digest, no network/caps, uid65532/read-only/seccomp/noNewPrivs, память768MiB без swap, CPU1/PID128, tmpfs384+128MiB, deadline45s, output32KiB, source64KiB/files16, concurrency1 (overload rejected). Cold compile требовал больше scratch, первоначальный no-space исправлен калибровкой бюджетов. Default Podman stop10s конфликтовал с cleanup5s при раннем cancel: stop-timeout=0/rm time=0 и cleanup10s, повторный cancel PASS1.08s. Cleanup failure poison+health503, inside-slot admission recheck, scoped orphan reconciliation, stale socket detection, named preflight cleanup error.

`ACADEMY_SANDBOX_TEST=1 .tools/go/bin/go test -count=1 -timeout=10m -v ./internal/runner`: полный PASS94.60s (11 реальных cases + unit regressions). `python3 scripts/crash-check.py`: настоящий SIGKILL/restart/stale socket/orphan cleanup/health PASS. Grader снаружи sandbox проверяет exact output/exit/stderr и начальные AST-критерии; stdout PASS не доверяется, unreachable branches rejected. Это не прокторинг и не public credential evidence. Independent RUNNER_REVIEW все выявленные findings resolved.

M4: `ACADEMY_E2E_RUNNER=1 npm run e2e` PASS3, skip1 (resource-intensive golden run только desktop, mobile UI/editor отдельно); no AI, compiler error/fix/run/submit/format/replay XP/reload. `npm run e2e -- --grep 'failed draft'`: PASS2 desktop/mobile; interrupted PUT → error/retry → preserved source. Первоначальный exact-text locator banner включал ×; исправлен тестовый locator, actual server verdict был passed. Platform source не выполнялся на host.

## Checkpoint 2026-10-15 — M5

Серверная versioned rank-policy0.1: 6 рангов с критериями/компетенциями; опубликован только первый Zero lesson, exams unavailable, verified=false. Skills0..4: фактические базовые упражнения дают2 с exercise+persisted attempt ID; без persisted evidence0. XP не присваивает rank. Course Explorer real server query, skills table/evidence links, review по последней неудачной попытке.

Добавлены request budgets120 mutations/min и6 execution requests/min; retention100 recent attempts плюс исходные verified evidence rows; races/replays не накручиваютXP, исходные ссылки не теряются. API progress не отправляет все saved source snapshots, output preview bounded. `make test` PASS; full `ACADEMY_E2E_RUNNER=1 npm run e2e` PASS7/skip1 (mobile runner duplicate skipped, mobile UI tested), axe skill table+search+rank evidence checks PASS.

## Checkpoint 2026-10-15 — M6

P1 CLI Toolkit: sum/int64 errors/overflow + inspect JSON/files, ToolError struct, starter/reference/tests/readme/release rubric. Multi-file root workspace, JSON/text/Go editors, saved snapshots10 with atomic replacement/restore, ZIP Git-ready export. API+store tests validate migration1→2 preservation, injected rollback, reopen, quota/unsafe names/snapshot IDOR. No symlinks from client: data maps only, worker creates regular tar entries; ZIP filenames root allowlist.

Project evaluation15 trusted black-box vectors+user Go tests actually run in isolated worker; total180s, perjob45s. User TestMain overrides refused, true TestX(*testing.T) required; quality/README/Git review remain pending and are explicitly shown. `p1:automatic` awards at most10XP but never rank/manual completion. Worker /tmp noexec; exec work/cache are bounded because Go test/go run may execute cached binaries. Final scratch128+192+192MiB, memory768MiB/PID128/CPU1/shm16MiB, no host mounts/network/privileges.

`make test` PASS; sandboxfull118.231s PASS12cases+regressions; fullbrowser10PASS/2explicit mobile runtime skips; expandedP1real15vectors+actualGoTests browser1PASS/1mobileduplicate skip3.1m; project files/snapshots/ZIP/axe2PASS. Fresh Git archive61e418b bootstrap/test/build PASS including autoGo1.27.2 download. Latest module Unicode-whitespace validator changes checked unit+TS/build. Independent PROJECT_REVIEW all high/medium findings resolved in reviewed scope; fullM12 audit not implied. Exact commands/logs/caveats: VALIDATION_REPORT.md.

Следующий этап M7: schema exam/attempt/rank evidence + versioned eligibility, mixed objective/coding assessment и negative tests, без fake promotion заXP/P1auto. Учитывать, что полногоZero/Juniorсодержания ещё нет; не выдавать весьrank по однойглаве.

### Финальная регрессия и запуск

APIdefault8380/Viteproxy8380 (8080 занятый чужойсервис не изменён), successfulbind передreadylog. `make test/build` повторён. Module ASCII grammar, UnicodeGoTest name, controlledCLI panic checks, escapedsource transport1MiB и proxyenvironment forwardingfalse regression прошли (смVALIDATION_REPORT). API+runner фактическиlivehealthok на8380; браузерныйpanelrequestqueued. Localcheckpoints only, remoteCI/publication отсутствуют.

Контрольная точка функционального кода и финальных доказательств: `7f4cca4` (локальный Git, без remote). Последующее уточнение handoff/docs не изменяет проверенный код.

### 2026-10-15 — M7/M8 завершены

M7: server-derived local Zero rank after immutable zero-core1.0.0, prerequisites,12objectivequestions+2coding/debuggingfunctions/12nativecases, draftCAS/replay/forgedawardguards; DBflock and recovery. M8: optional stdlibHTTP provider gateway, mode-specificRU prompts, modelrouting,3callsmax, budgetreservation4worstcasecalls, usageSQLite, explicitconsent/redaction/responseimageloadblocked, no modelauthority. Automatic freeform responsecache deliberately disabled: public deterministic glossary remains versionedcoursecontent; private AI requests/responses are not stored. Cloud/private-network restrictions and all limits documented README.

Evidence: make test PASS; native exam3PASS1skip; testprovider tutor3PASS1skip; fullcore14PASS5skip1staleassertionfail (P1 expected unverified after earlierexam) -> P1 assertion now verifies before/after invariant, targeted native rerun1PASS1skip3.1min. No application failure in that P1 run; no claim fullcombinedrerunallgreen yet. Visualbaseexamstyles and completedlessonrepeat fixed. Reports EXAM_REVIEW/TUTOR_REVIEW close openHigh/Medium.

### 2026-10-15 — M9 завершён

Privatebydefault Ed25519 local credentials with serverRankAward guard, PDF/PNG/QR, privacy/revoke/audit, publicverify, private/publicPassport, actualbadgeeventdates/activitygraph/projectoverview. Independent Nodecrypto/jsQR + Poppler/pdftotext checks; actualmissing/different/originalkeyprocessprobes; independentreview3Mediumclosed. Full `ACADEMY_E2E_RUNNER=1 npm run e2e`17PASS5explicitSKIP6.0min after initialthemeassumptionfailure corrected+2targetPASS. Optionaltest-providerseparate3PASS1SKIP. make test,govulncheck/npm auditsPASS0. FullM10course/P2andhigherassessmentstofollow; localformatnotOB/VC/JCS/hostedissuer.

## M10 checkpoint 2026-10-15 19:40 — частичная готовность

Zero8 +Junior12 полных глав опубликованы manifest2026.10; trusted Go examples/57function references passed host-side content gate. Native all57 batch ещё выполняется, не считать завершённым. Reader+P1 files browser4PASS desktop/mobile19.1s+axe. P2reference actualisolatedPostgres18.6 fullHTTP/ownership/transactions/migrations/poolMaxConns1 PASS; actualresponses6schemas PASS OpenAPI3.1. Separateprojectworker staticfixes+unit/vet/race pass; nativeRUN/health+SCRAMaudit pass, test/submit/persistence/cleanupfullgate pending. FullM10 иadvancedM11 НЕ завершены. Подробные команды, незавершённые проверки и ownership в AGENT_HANDOFF.

### M10 checkpoint 2026-10-15 19:55

32полныеглавы published:Zero8/Junior12/Junior+12; 96exercises. ContentGoexamples/referencesPASS;57nativeZero/JuniorPASS,36Junior+batchещёвработе. P2actualPGHTTP+4SQLchapterfences+OpenAPI6responsesPASS. Separateworker44nativechecks+realSQLpersistedfixture/limitedrole+restart+cleanupPASS; cancel/SIGKILLrecoveryPASS. Fullbrowser22PASS6explicitSKIP7.2min plusP2native1PASS1skip10.2s andnewHTTP/SQLreaderthemes2PASS8.0s. Desktop/mobileP2screensvisuallyreviewed. Chapters13..20authoringandnewJunior/Junior+assessmentsstillpending, soM10notDONE. Detailedcommands/remainingjobs/honestfailedprobehistoryinAGENT_HANDOFF/docsvalidation.

### M10 checkpoint 2026-10-16 00:50 — VALIDATING

Все40глав/120упражнений реализованы,117новыхrefsactualsandboxPASS(57+36+24); всеGoexamplespassed. P2native44checks+actualSQL/session/restart/cleanup+recoveryPASS. Junior/Junior+exams20questions+3tasks, allcourse/project/priorranktransactionalevidence; actualnativechain+signedcredentialsPASS64.30s. Zero1.0.0hashfrozen. Policyv1confirmsautomaticacademycriteria, manualprojectqualityNOTconfirmedandremainsseparatemilestone; Seniorhumanreviewunchanged. Independentfunction/projectrunner/assessmentreviewsnoopenHigh/Medium. Full34browserandfinaltechnicalrerunsstillpending; doNOTmarkM10DONEuntilpassed. M11codeNOTstarted; missingecosystemversionsresearchdelegatedread-only.

### M10 завершён 2026-10-16 01:14 Asia/Atyrau

Финальный `ACADEMY_E2E_RUNNER=1 ACADEMY_E2E_PROJECT_RUNNER=1 npm run e2e`: **27PASS7explicitSKIP0FAIL7.6min**, `docs/validation/m10-complete-full-browser.txt`. `make test` PASS unit/vet/race/TSbuild, `m10-complete-technical.txt`; all117starters+references/examples PASS48.402s; all117refs actualsandboxPASS57+36+24, improvedauth3targetednativePASS. Newnativeexamchain64.30sPASS, prerequisites/Zero explicitfixtures; P2nativeactualSQL/restart44checks+recoveryPASS. Allscopedreviewsclosed; pedagogical2chapterreviewandvariantfixesdocumented. Expectedskips: heavyduplicatemobilenative andoptionalprovider-onlyTutorcases(mockproviderpreviouslytestedM8). LocalGit checkpoints9f67b25/8947b0f plusfinaldocscheckpoint. FullMiddle→Senior/P3–P5 NOTdone; nextM11.

## Прогресс и прогноз

Оперативная панель: [PROJECT_PROGRESS.html](PROJECT_PROGRESS.html). Счётчик M0–M13 — доля закрытых milestone gates, не оценка объёма оставшейся работы. Отдельно видны доли опубликованного курса/проектов. Диапазон даты — плановая оценка, обновляется после завершения крупных этапов; он зависит от лимитов Codex и повторных проверок.

### M11 progress — 2026-10-17 (latest update; preceding notes are historical)

Dashboard inputs and generator are `docs/project-progress.json` and `scripts/render-project-progress.py`; rerender after confirmed milestone changes. 11/14 is a count of milestone gates, not weighted completeness. The historical unfiltered M11 PASS (2616.291s) and 2925.611s pass both predate the final Senior-24 corrections. The latest exact-current-tree unfiltered gate passed in 2985.278s with 530 passing subtests and 0 failures; see `docs/validation/m11-current-tree-acceptance-20261016.txt`. Scoped PASS evidence also covers Middle-14…40, Middle+13…24 and selected Senior branches; exact scopes and limits are in `docs/validation/m11-*-remediation.txt`. The stale invocation is excluded. All 88 remain draft bundles and unpublished until pedagogy, rank-policy and human-review gates close.

P3 runtime, browser and immutable submission evidence: `docs/validation/p3-runtime-acceptance.txt`, `p3-browser-acceptance.txt`, and `p3-project-evidence.txt`. Pinned image build passed offline; rootless P3 Run/Test/Submit passed, Submit with 45/45 checks; P2 regression passed 44/44 in combined and P2-only profiles. Desktop browser 2/2 and mobile workspace/axe/no-overflow passed. P4/P5 now have pinned stdlib module locks, independent runner profiles, API/workspace integration and immutable full-artifact/executed-source/checker/rubric evidence. A real local rootless Podman browser Submit passed for both reference projects; P4 deterministic oracle and P5 machine/document checks passed, saved evaluations stay `automatic_pass`, and neither awards a rank. Desktop and mobile screenshots were inspected; axe and horizontal-overflow checks passed. Evidence: `docs/validation/p4-p5-runtime-build-20261016.txt`, `p4-p5-browser-acceptance-20261016.txt`, `p4-p5-mobile-browser-20261016.txt`. These checks do not satisfy manual benchmark/capacity review or authenticated Senior defense, and are not a public multi-tenant certification.

Tutor local and cloud providers are configurable from UI. Local is loopback-only HTTP with no key; cloud is HTTPS/public host with a server-only mode-0600 key. Published course retrieval, diagnostic/progress opt-ins, provider revision and stale-consent rejection are bounded and covered by backend tests. Cloud key is never returned in GET; switching provider clears the prior config; save does not call a model. Desktop+mobile Tutor Playwright privacy/local/cloud/key redaction/revision tests passed 6/6 with a synthetic key and unused public endpoint; axe clean; no LLM/external HTTP request was made. On 2026-10-16, cloud model input also gained current OpenAI model suggestions backed by the official catalog; `gpt-6.1-sol` is presented as balanced, `gpt-6-luna` as cost-sensitive, and `gpt-6-astra` for complex tasks. UI typecheck/build and the updated desktop/mobile Playwright suite pass. Details: `docs/validation/tutor-provider-acceptance.txt`. A ChatGPT/Codex subscription does not provide API billing; the UI explicitly tells users that cloud API usage is billed separately.

Independent M11 audits identified content and rank-evidence gaps across Middle, Middle+ and Senior. Reports include `docs/research/M11_AUDIT_MIDDLE13_26.md`, `M11_AUDIT_MIDDLE27_40.md`, `M11_AUDIT_MIDDLE34_40.md`, `M11_AUDIT_MIDDLEPLUS13_23.md`, `M11_AUDIT_SENIOR01_12.md`, and `M11_AUDIT_SENIOR13_24.md`. Scoped repairs and gates cover the reviewed content groups; the latest current-tree aggregate passed after the Senior-24 remediation (530 subtests, 2,985.278s), while helper-only checks do not prove project/system outcomes. All six objective exam definitions are now authored and exposed in the catalog; advanced attempts stay blocked because no complete server rank policies or independence-verifiable human-review evidence are registered; the new offline signed-review store is groundwork only. Senior-24 Easy/Medium/Hard vectors were expanded: `ActionPriority` 16 boundary/priority pairs (16/16), `DecisionReview` separates evidence availability from verification (16/16), and `RetrospectiveComplete` covers all 8 bool tuples (8/8). Production-path evidence: `docs/validation/m11-senior24-decisionreview-20261016.txt` and `docs/validation/m11-senior24-vector-remediation-20261016.txt`. That edit made the prior full-tree aggregate stale; the newest exact-tree gate has now rerun successfully (530 subtests, 2,985.278s). Any subsequent M11 draft edit requires another full aggregate before release. The 88 lessons remain unpublished pending independent pedagogy review. Keep advanced awards fail-closed until registered policies bind course skills, prior rank, P3/P4/P5 machine evidence, required manual reviews and objective exam criteria; persisted signed P5 manual decisions now exist for exact artifact hashes/rubrics, but independent identity/trust and transactional rank-policy integration remain required before Senior can pass.

Signed project-review flow, 2026-10-16: `project-review-keygen` creates mode-0600 Ed25519 private keys and strict operator registry entries; `project-review-sign` signs the exported attestation template offline. API packet/history/POST and the workspace export/import/reload flow bind decisions to exact current project version, submission, evaluation, artifact and a separate manual rubric. Exact replay is idempotent, review rows are append-only, non-canonical signature base64 is rejected, and persisted history omits raw signatures and shows the fingerprint of the exact allowlisted public key used to verify each new decision; migration 0011 leaves legacy fingerprints empty and untrusted. The localStorage locator is untrusted and revalidated through the API. Full-page axe and no-overflow pass desktop/mobile. Verification: `docs/validation/project-review-flow-20261016.txt` and `docs/validation/project-review-key-fingerprint-20261016.txt`; screenshots `artifacts/desktop-project-review-flow.png`, `desktop-project-review-reload.png`, `mobile-project-review-flow.png`, `mobile-project-review-reload.png`. The signature proves key possession only; identity/independence are not established. Policy integration and evidence-bound awards are documented in `docs/validation/advanced-rank-pipeline-progress-20261017.txt`. A retention regression also preserves attempts referenced by `exam_project_evidence` after 100+ newer terminal attempts. This work is groundwork, not M11 or M12 completion.

Progress UI visual/a11y: `npm run test:progress` passes desktop/light and mobile/dark with no overflow or axe violations. The fresh full `make test` after P4/P5 and process-hook fixes passes Go tests, P1 reference tests/vet, whole-tree vet/race, TypeScript and production build; evidence is `docs/validation/p4-p5-final-technical-20261016.txt`. Real P4/P5 rootless browser Submit, mobile workspace export, axe/no-overflow and process-hook rejection passed; screenshots are in `artifacts/desktop-p4-evidence.png`, `desktop-p5-evidence.png`, `mobile-p4-workspace.png`, `mobile-p5-workspace.png`. Six objective exam definitions appear in API/UI; current policies activate only when trust and project-runner evidence is configured. The M11 draft gate passes, but pedagogy, M12 and M13 remain. No M11 draft may auto-publish.

Latest M11 rank pipeline slice (2026-10-17): `internal/httpapi/rank_policy_registration.go` derives immutable Middle/P3, Middle+/P4 and Senior/P5 policies from the active operator reviewer registry, exact manual rubric and trusted runner profile. Store resolves policy-required course/prior-rank/project/manual-review evidence at exam start, binds exact append-only review rows to the attempt, and transactionally revalidates them at Claim/Finish before award. With no valid configured registry/profile, advanced exams remain unavailable. `docs/validation/advanced-rank-pipeline-progress-20261017.txt` records checks and boundaries. API integration passes project submit, signed reject/approve, quorum, objective failure/no-award, exact successful award and replay rejection for all three ranks; fixture course/prior rank and project checker are explicitly not certified by this API test. Student UI explains exact-artifact review and the identity/independence limit of key fingerprints; desktop/mobile Playwright+axe/no-overflow 2/2, visually inspected. Full `make test` passes all packages, reference/vet, whole-tree race, typecheck and production build. Go version research was refreshed 2026-10-17: official release history still selects Go 1.27.2.

M11 remains **in progress**: 88 lessons are drafts. Full independent pedagogy/competency audit, rank/course mapping, review-rubric governance and release gate must still pass. M12 full security/load/failure/accessibility audit and M13 legal/secret/clean-source final acceptance remain open. The progress dashboard was regenerated and still reports 11/14 closed gates (unweighted; not a time-to-completion percentage).

Middle-24 remediation (2026-10-17): the three previous task contracts could label Redis a durable queue from two boolean inputs. Easy now returns `needs_review` whenever non-loss is required; Medium requires declared persistence, producer-ack semantics, consumer ownership recovery and idempotency; Hard adds failover test declaration, recovery target and loss budget. The lesson explicitly distinguishes declared checklist data from real Redis observation. Scoped `TestM11DraftCatalogAcceptance` passed (30.937s, 3 examples and 3 starter compiles), `go test ./internal/content ./internal/grading` passed, and current full `make test` passed all Go packages, references/vet, whole-tree race, TS typecheck and production build. The prior unfiltered M11 aggregate is stale after this content edit; rerun once the remaining lesson changes are complete. Details: `docs/validation/m11-middle24-redis-remediation-20261017.txt`.

Middle-25 remediation (2026-10-17): replaced three string/boolean role classifiers with a narrow injected `ProfileReader` and fake-backed use cases. Easy distinguishes a valid read, absent row and invalid request/profile; Medium exercises read/validate/write, proves unchanged values perform zero writes and detects storage errors; Hard composes creator+audit interfaces and verifies event order plus the partial state where create succeeds but audit fails. Tutorial text explicitly separates test doubles from SQL/HTTP integration and atomic rollback. Scoped acceptance passed on the final tree in 30.324s; `go test ./internal/content ./internal/grading` passed. Full `make test` passed after the lesson rewrite but before the final test-driver assertions; current scoped gate and affected Go packages passed after them. The 2026-10-16 unfiltered M11 content aggregate is stale after both Middle-24/25 edits and must rerun after remaining content changes. Details: `docs/validation/m11-middle25-di-remediation-20261017.txt`.

### 2026-10-18 — M11 remediation и M12 validation (IN PROGRESS)

M16–18 delivery/outbox/unknown-outcome протоколы, M20 truth table, M22 overflow/refill/invalid-domain и M26 Begin/Apply/Commit/Rollback исправлены и проверены scoped gates, production-mutant regressions, independent review. Каноническая таксономия 13 навыков теперь общая для content/learning; 88 draft mappings не теряют неизвестные ID, automatic P5 не доказывает communication. Общий make test прошёл для этого среза. Педагогическая сверка нашла новые конкретные M28/Senior13–23/M+21/24 gaps; их исправления ещё завершаются, поэтому начатый88 gate прерван намеренно (exit130), новый нужен после окончательного source tree. P3 human rubric v2 явно связывает M27–40 с exact project evidence; автоматические project criteria неизменны. Полные итоги проверок и workers — в AGENT_HANDOFF.md.

Новый локальный API load/restart probe PASS:120reads200,119writes200/21quota429, no mixed bundle/no award/no XP, SQLite integrity и restart persistence. Full50 browser suite ещё выполняется. Progress dashboard различает88подготовленных черновиков и реальную приёмку, актуальность gate основана на source fingerprint, а не mtime. Добавлены course-authoring guide и issue/PR templates; LICENSE/NOTICE/legal/secret final gates остаются открыты. M11/M12/M13 не завершены.

Проверка продолжена 2026-10-18: официальный Go release history и локальный `scripts/go.sh version` подтверждают pin Go 1.27.2; OpenAI Docs и фактические `codex --version`/`codex features list` перепроверили конфигурацию Codex/subagents (подробности `docs/research/AI_CODEX.md`). Текущий all-88 gate v3 длится отдельно, поэтому не изменять файлы M11 fingerprint до завершения. Независимый M12 аудит выявил P2 gap: runner memory/PID regression tests считают любой non-success результат доказательством лимита и не проверяют фактические cgroup `memory.events`/`pids.events`; после gate нужно заменить эти слабые доказательства и повторить rootless тесты. M13 cache-only scan создан без network/install и записан в `docs/validation/m13-license-inventory-20261018.json`: 339 records, 256 license-text digests, 125 distinct hashes. Go metadata оставлен UNKNOWN, где SPDX не установлен; 78 Go modules отсутствуют в local cache. SPDX/NOTICE text-level и legal review открыты; root LICENSE/NOTICE отсутствуют, проект помечен `UNLICENSED`, выбор лицензии остаётся владельцу.
