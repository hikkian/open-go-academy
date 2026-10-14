# ADR-0001: local-first modular monolith

Дата: 2026-10-15. Статус: принято для локальной одиночной установки; public hosting не разрешён.

## Решение и основания

Go 1.27.2 API на std net/http/slog; SQLite через database/sql + modernc.org/sqlite v1.60.1. React 19.3.1 / TypeScript 7.0.2 / Vite 8.3.4, CodeMirror 6, react-markdown без raw HTML, Playwright 1.64.1. Пакеты pin exact + npm lock/go.sum. Точные sources/engines в ../research/evidence/npm-versions.json и VERSION_MATRIX.md. Чистая сборка ещё обязательна в M1.

SQLite выбран вместо предполагаемого PostgreSQL: реальный single-user local-first, миграции и transactions, без обязательной службы БД. Hosted multi-user потребует PostgreSQL+pgx, отдельных auth и migration tests. JSON-файл не выбран: слабее транзакции, integrity и migrations. Продуктовая цель неизменна.

Vite SPA вместо Next.js: Go API уже сервер приложения, SSR не нужен для первого learning loop. CodeMirror вместо Monaco: модульность и меньшая IDE-поверхность, реальная Go подсветка; gopls не обещается. Plain Markdown вместо MDX: учебный текст не исполняет JS. Никаких UI-библиотек до потребности.

Runner отдельный процесс `cmd/runner`, transport Unix socket 0600. Только rootless Podman и pinned image Go 1.27.2 по digest; network none, uid65532, rootfs read-only, caps drop, no-new-privileges, seccomp default, cgroup memory/PID/CPU, bounded tmpfs/output/concurrency/deadline. API не получает docker socket и не запускает user source на host. Недоступный hardened local runtime → run disabled, без fallback. Проверенный M0 probe: memory.max 536870912, pids.max 64, cpu.max 100000 100000, uid65532, go1.27.2. Docker rootful имеется, но не используется runner.

Контейнер делит ядро с host: это локальный учебный режим с остаточным kernel/runtime escape риском. Public multi-tenant требует независимого hardening audit и gVisor/microVM boundary, multi-user authorization, queue quotas, monitoring; пока API принудительно loopback, проверяет Host/Origin, profile session secret. Self-issued achievements не публичные trusted credentials.

Источники и альтернативы: ../research/GO_CURRENT_STATE.md, ECOSYSTEM.md, FRONTEND.md, SANDBOX_SECURITY.md, CREDENTIALS.md, AI_CODEX.md; THREAT_MODEL.md. Платные облака, API и публикация не нужны и не выполняются.

## Калибровка реализации M3–M6

Фактически tested local limits: memory768MiB/swap0, CPU1/PID128, source64KiB/files16, JSON64KiB/8, output32KiB, deadline45s/job+cleanup10s, scratch tmp128(noexec)+work192(exec)+cache192(exec), shm16MiB. 512MiB scratch сумма не изменена после перераспределения дляGo test/cached executables. Main API/read-only content не выполняет user code; subprocess Podman толькоworker. Project files128KiB total, Go64KiB/JSON64KiB, fixed Go module для stdlib-only P1; saved versions/ZIP validated. Private evaluation15 blackbox cases+unit overall180s; 12exec/min120mutations/min (M5 initial6 увеличен дляproject workflow), no queue/unbounded workers. Quality/manual rubric не приравнивается к automatic-pass.
