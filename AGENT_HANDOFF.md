# AGENT HANDOFF — open-go-academy

Обновлено 2026-10-15. M0 DONE; M1 IN PROGRESS. Приложение ещё не реализовано.

Completed milestone: M0 Research & Baseline.
Files changed: docs/research/*, docs/architecture/ADR-0001-tech-stack.md, THREAT_MODEL.md, .codex/agents/builder.toml и researcher.toml (явная модель), .gitignore, EXECUTION_PLAN.md.
Architecture decisions: local-first Go1.27.2 + SQLite modernc v1.60.1, React19.3.1/TS7.0.2/Vite8.3.4/CodeMirror; отдельный runner rootless Podman only Unix socket; public disabled. AI optional.
Commands executed: git status --short (исходно no git), git init (локально, remote отсутствует); go version; node --version; npm --version; codex --version; codex features list; official schema/JSON/npm metadata fetch; podman pull docker.io/library/golang:1.27.2-alpine; rootless podman resource probe (точно описан ADR).
Tests passed: schema ключи verified; configured researcher/reviewer успешно вернули reports; runtime probe uid65532 + cgroup memory/PID/CPU; официальный Go archive SHA256 verified.
Tests failed/skipped: первоначальный bwrap probe не имел /lib loader, исправленный probe прошёл; app build/tests/E2E ещё NOT RUN. Docker rootful не используется. API цены документальны, платных вызовов нет.
Open risks: shared kernel local runner; полный negative audit M3 не проведён; большая часть платформы/курса отсутствует; hosted auth/credentials trust не реализованы.
Next milestone and first concrete task: M1 — создать Go API+SQLite миграции/local session и frontend; test чистой БД/restart и browser smoke. Go toolchain .tools/go/bin/go (ignored, скачан официально) и Node24.18.0 доступны. Detailed specs: docs/MASTER_SPEC.md; порядок docs/EXECUTION_PLAN.md.

## Checkpoint M1/M2 — 2026-10-15
Completed milestone: M1 foundation, M2 content/first lesson.
Files changed: cmd/api, internal/httpapi, internal/store/migrations, internal/content/data+solutions, content/reference, web, Makefile, scripts/go.sh/restart-check.py, package locks, .env.example, CI.
Architecture decisions: SQLite migration version1; deterministic server-only XP; profile cookie+CSRF; lessons bundled Markdown; solutions excluded from lesson JSON and fetched explicitly.
Commands executed: .tools/go/bin/go test ./...; .tools/go/bin/go vet ./...; .tools/go/bin/go test -race ./...; cd web && npm run typecheck && npm run build; python3 scripts/restart-check.py; npm run e2e -- --grep 'profile, roadmap'.
Tests passed: Go unit/race/vet, snippets2/refs3, real API process restart, browser desktop/mobile/light/dark+axe2 tests.
Tests failed/skipped: initial contrast and scroll focus corrected; CI not run remotely; full runner E2E pending.
Open risks: local shared-kernel sandbox, other OS unsupported; M5–M13 untouched; first lesson only.
Next milestone and first concrete task: finish M3 real crash/cancel cleanup suite, M4 browser real Run/Submit/error/reload/noAI/XP replay. Current Makefile+README have startup commands.

## Checkpoint M3/M4 — 2026-10-15
Completed milestone: M3 confined LOCAL runner, M4 golden path.
Files changed: internal/runner, cmd/runner, httpapi grader/API tests, tests/learning.spec.ts, playwright config, web autosave recovery, scripts/crash-check.py.
Architecture decisions: rootless only, server exact+AST verdict, fixed resources after cold compile calibration; cleanup failure disables engine, scoped reconciliation after crash.
Commands executed: ACADEMY_SANDBOX_TEST=1 .tools/go/bin/go test -count=1 -timeout=10m -v ./internal/runner; python3 scripts/crash-check.py; ACADEMY_E2E_RUNNER=1 npm run e2e; npm run e2e -- --grep 'failed draft'; .tools/go/bin/go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...; npm audit --omit=dev.
Tests passed: sandbox suite94.60s11 real cases, SIGKILL recovery, E2E3+skip1, draft recovery2, govulncheck no vulnerabilities, npm audit0.
Tests failed/skipped: earlier resource no-space/cancel grace defect fixed+retested; two negative-test fixtures had unintended unused import (fixed) and mobile retry reused same code marker (fixed); mobile real runtime is skipped to avoid duplicate resource test, editor+layout exercised.
Open risks: shared kernel local-only; no public tenants/auth/credentials. M5–M13 pending.
Next milestone and first concrete task: M5 server rank criteria/skills matrix from real exercise evidence; honest unpublished tracks, XP separate from assessment.

## Checkpoint M5 — 2026-10-15
Completed milestone: M5 career/skills/XP.
Files changed: internal/learning, httpapi learning/limits, store retention/evidence, web course/skills/path UI, tests/learning.spec.ts.
Architecture decisions: rank-policy0.1 criteria from server, level2 only persisted basic exercise evidence; exams unavailable, verified rank false, no XP rank shortcuts. Retention100 recent + original verified attempts;6 executions/min120 mutations/min.
Commands executed: make test; ACADEMY_E2E_RUNNER=1 npm run e2e.
Tests passed: Go unit/vet/race, retention/concurrentXP; E2E7 PASS1 explicit mobile runner skip; skill/course search/criteria + axe desktop/mobile.
Tests failed/skipped: no current failures; CI remote NOT RUN; no attestations/credentials.
Open risks: full course not written, public execution disallowed, M6–M13 pending.
Next milestone and first concrete task: M6 P1 multi-file project persistence/snapshots, rootless test/run, ZIP export.
