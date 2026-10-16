# Open Go Academy — точка продолжения

Обновлено **2026-10-16 06:44 Asia/Atyrau**. **M0–M10 завершены в локальном режиме; M11 начат; M12–M13 не начаты.** Последние контрольные коммиты: `9f67b25`, `8947b0f`, `193e32b`, `c9207de`, `1c3e37f`. Ветка `master`, remote нет.

Открой [панель прогресса](docs/PROJECT_PROGRESS.html). Данные и генератор: [project-progress.json](docs/project-progress.json), `python3 scripts/render-project-progress.py`. Она показывает 11/14 закрытых milestone gates (79%, не оценка объёма), 40/128 уроков (31%), 3/6 экзаменных рангов и 2/5 проектов. Оценка полного срока 3–6 недель активной работы, 6–27 ноября 2026 календарно; уверенность низкая, это плановый диапазон, не срок поставки. Он пересматривается по фактическому темпу M11 и лимитам.

## Что уже готово

- Zero/Junior/Junior+: **40 глав, 120 упражнений**, русский самостоятельный учебный материал. Все 117 функций и их стартовый код проходят production-grader contract; trusted Go-примеры и решения компилируются.
- Локальный путь курса: урок, CodeMirror-редактор, runner, проверки, сохранение прогресса, roadmap/skills/exam catalog. Опциональный Tutor работает без платного API по умолчанию. Self-issued локальные Ed25519 credentials, PDF/PNG/QR и приватный паспорт.
- P1 CLI Toolkit и P2 LinkHub. P2 использует отдельный rootless Podman project-runner, offline immutable images и временные PostgreSQL 18.6 jobs. Пользовательский код никогда не запускается в API или на host. На runner/review scope пройдены HTTP/auth/CSRF/ownership/SQL-проверки, рестарт приложения, cleanup, SIGKILL recovery. Это локальная общая kernel/container изоляция, **не сертификация публичного multi-tenant запуска**.
- Версионированные экзамены Zero/Junior/Junior+; версия Zero 1.0.0 и её hash заморожены. Полные курсовые prerequisites, предыдущий ранг и project evidence проверяются сервером; XP не выдаёт rank. Junior/Junior+ утверждают только автоматические критерии Go Academy. Качество P1/P2, которое требует review человека, явно остаётся непроверенным; Senior требует human review.
- Research стека P3/P4: [M11_ECOSYSTEM.md](docs/research/M11_ECOSYSTEM.md). Это version snapshot; новые библиотеки и образы ещё не устанавливались/не интегрировались. Go 1.27.2, pgx 5.11.0, PostgreSQL 18.6 pins и checker images M10 уже проверены.

## Проверки M10

- Полный browser suite: **27 passed, 7 expected skipped, 0 failed, 7.6 min** — `docs/validation/m10-complete-full-browser.txt`.
- `make test`: Go tests/vet/race, TypeScript и production build PASS — `docs/validation/m10-complete-technical.txt`.
- Все 117 заготовок/решений и исполняемых Go-примеров проверяются `internal/content/published_test.go`; последний прогон **48.402 s PASS** — `docs/validation/m10-all-starters-fixed.txt`.
- P2 native browser submit, замена сервера с сохранением временной БД и checker: `docs/validation/p2-worker-native-limited-role.txt`. P2 sandbox recovery: `docs/validation/p2-worker-recovery.txt`.
- Новый native Junior→Junior+ grading/atomic award/replay rejection/credentials: **64.30 s PASS**, с явно обозначенными тестовыми prerequisite fixtures — текущий источник `.data/m10-native-exam-chain.log`.
- Progress page: axe и mobile/desktop layout PASS; последние screenshot previews `.data/project-progress-desktop.png` и `...mobile.png`.

## M11 статус и следующая работа

Подробная разбивка: [MIDDLE_SENIOR_PLAN.md](docs/curriculum/MIDDLE_SENIOR_PLAN.md). План включает 40 занятий Middle на недели 9–16, по 24 углублённых занятия Middle+ и Senior, P3 TaskFlow, P4 ScaleLab и P5 Architecture Capstone. План не считается созданным учебным материалом: сейчас **0/88 M11 глав опубликовано, P3/P4/P5 не реализованы**.

Три экономичных builder-агента владеют непересекающимися черновыми путями:

- `/root/middle_01_12`: Middle 01–12;
- `/root/middle_13_26`: Middle 13–26;
- `/root/middleplus_01_12`: Middle+ 01–12.

Их файлы не добавлять в published manifest автоматически. Проверить каждую главу, prompts/starter/source contracts, all private vectors, JSON-number precision, Go examples и prerequisites через production `LoadFor → BindReference → Request → Check`; только после исправлений двигать progress lesson count. Освободившиеся agent slots можно использовать следующим раздельным batch Middle 27–40/Middle+ 13–24/Senior 01–12, не больше трёх параллельно.

После покрытия контента реализовать реальные P3/P4/P5 вертикальные сценарии в ограниченной отдельной среде; существующий runner проверен только на P2. Для race checks, брокера, Redis, benchmark/profiling и deployment отдельно доказать инструментальную поддержку и лимиты. Документация Docker Compose сама по себе не исполняет проект. Нужны server-owned assessment versions, rank prerequisites и человеческий Senior-review workflow без фальшивого профессионального статуса. M12: полный threat/performance/accessibility audit; M13: чистый reproducible install, CI/release/governance preparation. Не публиковать код и не расходовать деньги.

## Запуск и ограничения

```sh
make build
make runner
make project-runtime
make project-runner
make dev
make test
ACADEMY_E2E_RUNNER=1 ACADEMY_E2E_PROJECT_RUNNER=1 npm run e2e
```

Основной API на `127.0.0.1:8380`, basic runner и project-runner — отдельные процессы; чужой Python-сервис на `8080` не трогать. P2 runtime требует заранее доступный rootless Podman, скрипт `make project-runtime` создаёт/фиксирует локальные images. `.data/academy.db`, `session.secret`, `issuer.key` — личные данные/ключи: не удалять, не логировать, не коммитить. E2E scripts должны использовать только отдельную `.data/e2e` или выделенную temp-папку.

Последнее фактическое usage check (около 06:40 local): primary2%, weekly47%, usage разрешён; следующий primary reset11:37:58 local, weekly reset Oct22. Имеющийся heartbeat `open-go-academy-m13` следует оставить одноразовым и привязать к reset+1min; не создавать daily automation/дубликаты и не расходовать reset credits. Проверять текущие лимиты заново до большого batch. Рабочая среда/host должны оставаться активными; фон после закрытия Codex не обещать.
