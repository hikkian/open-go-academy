# Open Go Academy — точка продолжения

Обновлено **2026-10-18 14:47 Asia/Atyrau**. Локальный `master`, remote нет. M11 all-88 v3 остаётся историческим PASS на старом fingerprint; после аудита контента, runner assertions и расширения M+13–23 grading vectors актуальный fingerprint `sha256:e03e7282ceea394a172f8cc4f6204bd20bf58dfb71aa6d546e1f0eda6683d2fb`; v4 gate запускается с отчётом `docs/validation/m11-current-tree-acceptance-20261018-v4.txt`. Текущий `make test` PASS exit0; curriculum quality 6/6, grading package PASS. `make sandbox-test` PASS exit0; real cgroup v2 OOM/PID counters подтверждены, zero scoped orphan containers. Последний usage check 14:46: `ordinaryUsageAllowed=true`, primary15%, weekly34%; reset primary 19:27:57, weekly 2026-10-25 03:38:23 Asia/Atyrau. Сбросные кредиты не использованы. **M0–M10 DONE; M11–M13 IN PROGRESS.** Полный проект не завершён.

## Продукт и доверительные границы

- Реализованы onboarding/profile, курс на русском, редактор/изолированный Go runner/grading, прогресс и drafts, P1–P5 workspaces, rank exams, XP/skills, локальные credentials/passport и optional AI Tutor. No-AI путь полноценный; public multi-tenant runner не включён.
- 128 глав Zero → Senior включены в локальный `internal/content/manifest.json`; это не означает публикацию репозитория наружу. Ранги отличны от XP. Для advanced award нужны course evidence, exact project artifact, human review/quorum и все objective exam parts. Подпись доказывает владение разрешённым ключом, не личность/независимость/аккредитацию.
- P3 manual rubric v2 виден до отправки и связывает M27–40 artifact criteria с version/hash. Автоматические project requirements не подменяются ручным review. Mobile/desktop UI tests проверяют клавиатуру, hostile HTML как текст, отсутствие лишних API writes, axe и overflow.
- Все 13 canonical skill IDs связаны с course/project evidence или exact P5 human-review evidence уже выданного Senior award. Communication не выводится из automatic P5, а Tutor получает только coarse capped level без reviewer details. См. `docs/research/M11_SKILL_MAPPING_20261018.md`.
- M28 Hard теперь проверяет настоящий `TaskService`→`TaskRepository` fake. Full gate v1 обнаружил standalone reference/import и starter type-tag mismatch; исправлены. Узкий gate M28 PASS31.819s: `docs/validation/m11-middle28-current-20261018.txt`.
- Middle+21 Easy теперь показывает `within` и `errors`, а private vectors покрывают 4 invalid boundaries. Lesson/example/spec синхронизированы; scoped gate PASS31.475s: `docs/validation/m11-middleplus21-easy-current-20261018.txt`.
- Панель прогресса `docs/PROJECT_PROGRESS.html` строится reproducibly из `docs/project-progress.json`: 128/128 локальных глав, M11 gate и оставшиеся M12/M13 отображаются отдельно; 11/14 — число контрольных точек, не доля объёма.
- Outcomes всех 88 advanced lessons переведены в естественный русский; точных дублей outcomes и JSON self-check Q/A нет. `scripts/test_m11_curriculum_quality.py` теперь закрепляет уникальность, word count, тематические P3/labs, Middle+ introductions и критические glossary поля, и входит в `make test`.
- Middle01–14 имеют отдельные P3 practice bridges с явными границами helper-моделей; Middle21–37 — шаговые labs с ожидаемыми наблюдениями/очисткой и first-use терминами; Middle+13–23 — тематические объяснения/лабы, по 1853–1921 слов. Непокрытые упражнениями детали честно связываются с отдельными project/integration evidence.
- P3 disposable PostgreSQL `TestPostgresTaskOutboxAndTenantIsolation` — PASS0.388s; Redis 8.4.2 TTL/Streams CLI labs — PASS, временные контейнеры и volume удалены. Evidence: `docs/validation/m11-p3-lab-check-20261018.txt`, `m11-redis-lab-check-20261018.txt`.

## Проверки текущего дерева

- Полный `make test` после текущих runner/content/spec и M13 scanner изменений — **PASS, exit0**: Python curriculum/fingerprint/license/probe tests, Go unit/integration, P1 reference, vet, race, TypeScript typecheck и Vite build. Лог: `artifacts/make-test-after-m11-audit-fixes-20261018.log`. После добавления узкого pedagogy regression test он отдельно прошёл 6/6; grading package повторно PASS.
- Целевой Senior communication Playwright: **2 passed** desktop/mobile; axe, no overflow, human-review label и token privacy. P3 manual rubric UI Playwright: **2 passed** desktop/mobile; keyboard, no API writes, hostile HTML inertness, axe/no overflow.
- `npx playwright test tests/project-review.spec.ts --reporter=list` — **2 passed** desktop/mobile after Russian reviewer-history label update; signed review flow and trust-limit UI pass.
- `npm run test:progress` — PASS desktop light + mobile dark, 6 workstreams, axe clean/no overflow; progress separates local catalog availability from acceptance/public release. Расширенный полный source fingerprint сначала вызвал mobile overflow; rendered text now uses abbreviated fingerprint, exact hash stays in JSON metadata, rerun PASS.
- `python3 scripts/local_load_check.py` повторён на текущем API: **120/120 GET 200**, **119 PUT 200 / 21 PUT 429** с Retry-After, draft bundle atomic, restart persistence, rank/XP baseline, SQLite integrity/FK clean. JSON: `docs/validation/m12-local-load-20261018.json`. Ограниченный local probe, не SLO.
- Root/web `npm audit --audit-level=low` — 0 vulnerabilities. Gitleaks v8.30.1 staged-diff и 27-commit history scans PASS; один false positive про public reviewer-key fingerprint UI text исключён точным fingerprint в `.gitleaksignore`. Нужен повтор scan на final clean archive. Evidence: `docs/validation/secret-scan-20261018.txt`.
- `python3 scripts/recovery-check.py` — PASS on disposable local DB: online backup/restore, API restart, profile/theme/draft/migration persistence. `govulncheck v1.9.0 ./...` — `No vulnerabilities found.`
- Full browser на предыдущем checkpoint: 40 PASS, 10 intentional SKIP; после M11/content/manifest правок полный desktop/mobile suite нужно повторить. `make sandbox-test` **PASS144.479s** после P4/P5 AST-guard fix: 13 rootless runner integration cases + admission/cleanup regressions.
- M12 independent code review нашёл пропущенный `syscall.Exec` и ложное отклонение полей вроде `state.Exit` из-за сравнения только имени селектора. Guard исправлен: связывает запрещённые API с импортированным стандартным пакетом, отказывает dot-import os/runtime/syscall, запрещает process-exit APIs включая `syscall.Exec`, пропускает одноимённые пользовательские поля/методы. AST-only regression tests, focused projectrunner test/vet/race — PASS; полный `make test` после этого последнего изменения ещё нужен. `internal/projectrunner` не входит в M11 fingerprint.
- Независимый M12 audit выявил P2 gap в resource-limit evidence: `internal/runner/runner_test.go` принимает произвольный non-success результат за PASS для memory/PID случаев, даже compile error/timeout; S05 требует фактические cgroup counters. Новый `scripts/m12_sandbox_limit_probe.py` читает точные policy literals из `internal/runner/runner.go`, строго отказывает на изменённые/опасные flags, строит фиксированные host Go fixtures и должен наблюдать actual `memory.events`/`pids.events` с cleanup. Rootless integration теперь фактически выполнена: `make sandbox-test` PASS exit0; report показывает `memory.events oom_kill=1`/exit137 и `pids.events max=3`/EAGAIN, 33 child cleanup; после выполнения нет scoped probe containers. Runner memory/PID assertions требуют точных результатов. Evidence: `docs/validation/m12-sandbox-cgroup-20261018.txt`.
- Новый scoped M12 test `npx playwright test tests/m12-responsive-a11y.spec.ts --project=desktop --project=mobile` — PASS2/0 (6.9s): keyboard-only Tab/focus, reduced motion, axe/no-overflow and all nav controls fully within viewport на 768×1024 и 1920×1080. Он поймал и помог исправить 768→986px overflow и `.lesson-icon` contrast; после source CSS изменения и `web/npm run build` обе проверки чистые. Скриншоты visually inspected, evidence `docs/validation/m12-responsive-a11y-20261018.txt`. Tablet 768×1024 и wide 1920×1080 проверены. Полная matrix всех страниц, error/loading и themes ещё не завершена.
- Clean-source результат M13 от 2026-10-16 относится к старому commit `2b00dfc` и уже не финален. `GETTING_STARTED.md`/README не дают проверенного native Windows/macOS маршрута; можно добавить честный Linux VM/WSL путь и маркировать native варианты непроверенными.
- Cache-only scanner `scripts/m13_license_inventory.py` и 7 regression tests уже вошли в `make test`; полный machine-readable scan — `docs/validation/m13-license-inventory-20261018.json`. Он содержит 268 наблюдений (92 Go, 176 npm), 192 license-file entries/98 distinct hashes; 40 Go dependencies отсутствуют в local cache, Go SPDX остаётся UNKNOWN. npm metadata observed: MIT133/Apache26/MPL14/ISC2/BSD-3. Корневые LICENSE/NOTICE отсутствуют, проект UNLICENSED; независимый аудит исправления README backup и generated-state/quoted-path scanner пройден, 7 inventory tests PASS. Обновлённый scan будет сгенерирован из финального clean archive; legal mapping и выбор лицензии оставлены владельцу.

## M11 acceptance checkpoints

Попытка v1 на fingerprint `sha256:6ba40b...` прервана после двух M28 findings; её отчёт — `docs/validation/m11-current-tree-acceptance-20261018.txt`, статус `INTERRUPTED`. Это не PASS.

Gate v2 завершился **PASS** за 3009.434s на прежнем fingerprint `sha256:7600aec0…a7911db` (`docs/validation/m11-current-tree-acceptance-20261018-v2.txt`), не подтверждал поздние педагогические/manifest правки. **Gate v3 завершился PASS** на fingerprint `sha256:c9c85f0f9b6d2622bc29cefad12e2feac1b349950f1b7c3e579a05ed556af2c0`: all 88 advanced lessons, exit0, 3097.044s, matching start/finish fingerprints, report `docs/validation/m11-current-tree-acceptance-20261018-v3.txt`, log `artifacts/m11-full-final-v3-20261018.log`. Это закрывает технический all-88 gate, но не самостоятельную педагогику/governance, M12/M13.

Предыдущий gate v2 воспроизводится командой:

```sh
python3 scripts/check_m11_acceptance.py \
  --report docs/validation/m11-current-tree-acceptance-20261018-v2.txt \
  --log artifacts/m11-full-final-v2-20261018.log \
  --cache-dir /tmp/oga-m26-20261018
```

Для v3 проверить `Result`, exit code, `Scope: all 88 advanced lessons`, и равенство source fingerprints. Изменения `internal/content`, `internal/grading`, `internal/runner`, `internal/competency`, `go.mod`, `go.sum` или M11 gate scripts делают результат STALE. Report: `docs/validation/m11-current-tree-acceptance-20261018-v3.txt`; log: `artifacts/m11-full-final-v3-20261018.log`; cache: `/tmp/oga-m11-v3-20261018`.

## Следующие шаги

1. Дождаться all-88 M11 gate v4; проверить status, scope, exit0 и равенство начала/конца fingerprint.
2. Запустить full desktop/mobile E2E с `ACADEMY_E2E_RUNNER=1 ACADEMY_E2E_PROJECT_RUNNER=1 ACADEMY_E2E_P3_RUNNER=1` (без P3 флага suite пропускает реальную интеграцию P3).
3. Закрыть независимую педагогику/ранговую traceability M11; далее полный M12 security/accessibility/load audit и M13 clean-source/bootstrap/recovery/NOTICE. Не публиковать и не выбирать лицензию за владельца.
4. Завершить M11 traceability, M12 угрозы/resource-abuse/restart/responsive/theme/a11y, затем M13 clean archive/bootstrap/recovery, requirement matrix, Gitleaks, dependency notices и runbook. Не публиковать и не выбирать лицензию за владельца.
5. Cache-only report готов: `docs/validation/m13-license-inventory-20261018.json`; сканер и tests добавлены в `make test`. Дальше выполнить text-level SPDX/legal review для найденных текстов и подготовить NOTICE только после верификации; неизвестные/uncached Go модули не выдавать за подтверждённые. Root LICENSE/NOTICE отсутствуют, manifests UNLICENSED; проектную лицензию выбирает владелец.
6. Обновить план/progress/handoff, `git diff --check`, локальный checkpoint. M11–M13 не закрывать без фактического gate evidence.

## Лимиты и безопасность

Последний `get_usage_limits` **2026-10-18 14:46 Asia/Atyrau**: `ordinaryUsageAllowed=true`, primary **15%**, weekly **34%**; бесплатный reset credit доступен, но не использован. Primary reset **2026-10-18 19:27:57**, weekly reset **2026-10-25 03:38:23 Asia/Atyrau**; кредитов не покупать и лимиты не обходить. При следующем запуске проверить лимиты до любой большой задачи; продолжать после reset без покупки credits. Единственный heartbeat `open-go-academy-m13` должен оставаться one-shot на следующий limiting reset +1 минута; не создавать daily repeat/duplicate.

Работать только в репозитории. Не читать/удалять/коммитить `.data/academy.db`, `session.secret`, `issuer.key`; не останавливать пользовательские сервисы на 8380/8080; QA DB/keys только disposable. Не публиковать, не тратить деньги, не включать public multi-tenant execution. Чат оставить открытым.
