# Open Go Academy — точка продолжения

Обновлено **2026-10-18 11:29 Asia/Atyrau**. Локальный репозиторий `master`, remote нет; checkpoint `34628a9` содержит M12 responsive fixes/test и M13 cache-only license inventory/docs. Сейчас рабочее дерево меняет только progress JSON/HTML и строгие regression tests для нового cgroup probe; fingerprinted M11 source неизменён. Full gate v3 идёт на `c9c85f0f…af2c0`, лог дошёл до Middle+13, результата ещё нет. **M0–M10 DONE; M11–M13 IN PROGRESS.** Полный проект не завершён.

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

- Полный `make test` после content/manifest/acceptance-harness изменений — **PASS**: curriculum quality + fingerprint regressions, Go unit/integration (local 128 catalog), P1 reference, vet/race, TS typecheck и Vite build. Лог: `artifacts/make-test-m11-current-20261018.log`. Этот прогон был до последней projectrunner guard правки; текущий tree нужно проверить снова после v3.
- Целевой Senior communication Playwright: **2 passed** desktop/mobile; axe, no overflow, human-review label и token privacy. P3 manual rubric UI Playwright: **2 passed** desktop/mobile; keyboard, no API writes, hostile HTML inertness, axe/no overflow.
- `npx playwright test tests/project-review.spec.ts --reporter=list` — **2 passed** desktop/mobile after Russian reviewer-history label update; signed review flow and trust-limit UI pass.
- `npm run test:progress` — PASS desktop light + mobile dark, 6 workstreams, axe clean/no overflow; progress separates local catalog availability from acceptance/public release. Расширенный полный source fingerprint сначала вызвал mobile overflow; rendered text now uses abbreviated fingerprint, exact hash stays in JSON metadata, rerun PASS.
- `python3 scripts/local_load_check.py` повторён на текущем API: **120/120 GET 200**, **119 PUT 200 / 21 PUT 429** с Retry-After, draft bundle atomic, restart persistence, rank/XP baseline, SQLite integrity/FK clean. JSON: `docs/validation/m12-local-load-20261018.json`. Ограниченный local probe, не SLO.
- Root/web `npm audit --audit-level=low` — 0 vulnerabilities. Gitleaks v8.30.1 staged-diff и 27-commit history scans PASS; один false positive про public reviewer-key fingerprint UI text исключён точным fingerprint в `.gitleaksignore`. Нужен повтор scan на final clean archive. Evidence: `docs/validation/secret-scan-20261018.txt`.
- `python3 scripts/recovery-check.py` — PASS on disposable local DB: online backup/restore, API restart, profile/theme/draft/migration persistence. `govulncheck v1.9.0 ./...` — `No vulnerabilities found.`
- Full browser на предыдущем checkpoint: 40 PASS, 10 intentional SKIP; после M11/content/manifest правок полный desktop/mobile suite нужно повторить. `make sandbox-test` **PASS144.479s** после P4/P5 AST-guard fix: 13 rootless runner integration cases + admission/cleanup regressions.
- M12 independent code review нашёл пропущенный `syscall.Exec` и ложное отклонение полей вроде `state.Exit` из-за сравнения только имени селектора. Guard исправлен: связывает запрещённые API с импортированным стандартным пакетом, отказывает dot-import os/runtime/syscall, запрещает process-exit APIs включая `syscall.Exec`, пропускает одноимённые пользовательские поля/методы. AST-only regression tests, focused projectrunner test/vet/race — PASS; полный `make test` после этого последнего изменения ещё нужен. `internal/projectrunner` не входит в M11 fingerprint.
- Независимый M12 audit выявил P2 gap в resource-limit evidence: `internal/runner/runner_test.go` принимает произвольный non-success результат за PASS для memory/PID случаев, даже compile error/timeout; S05 требует фактические cgroup counters. Новый `scripts/m12_sandbox_limit_probe.py` читает точные policy literals из `internal/runner/runner.go`, строго отказывает на изменённые/опасные flags, строит фиксированные host Go fixtures и должен наблюдать actual `memory.events`/`pids.events` с cleanup. Probe и negative/unit tests прошли code review, 10/10 unit tests PASS; интеграцию намеренно не запускали, чтобы не конкурировать с M11 all-88 gate. После gate выполнить `make sandbox-test`, дождаться cgroup report и подтвердить zero orphan containers; затем исправить исходные слабые runner assertions и повторно закрыть M11 fingerprint gate.
- Новый scoped M12 test `npx playwright test tests/m12-responsive-a11y.spec.ts --project=desktop --project=mobile` — PASS2/0 (6.9s): keyboard-only Tab/focus, reduced motion, axe and no-overflow на 768×1024 и 1920×1080. Он поймал и помог исправить 768→986px overflow и `.lesson-icon` contrast; после source CSS изменения и `web/npm run build` обе проверки чистые. Скриншоты visually inspected, evidence `docs/validation/m12-responsive-a11y-20261018.txt`. Полная matrix всех страниц, error/loading, themes и tablet ещё не завершена.
- Clean-source результат M13 от 2026-10-16 относится к старому commit `2b00dfc` и уже не финален. `GETTING_STARTED.md`/README не дают проверенного native Windows/macOS маршрута; можно добавить честный Linux VM/WSL путь и маркировать native варианты непроверенными.
- Cache-only scanner `scripts/m13_license_inventory.py` и 5 regression tests уже вошли в `make test`; полный machine-readable scan — `docs/validation/m13-license-inventory-20261018.json`. Он содержит 339 наблюдений (163 Go, 176 npm), 256 license text files/125 distinct hashes; 78 Go зависимости отсутствуют в local cache, Go SPDX остаётся UNKNOWN, хотя 49 записей имеют license files. npm metadata observed: MIT133/Apache26/MPL14/ISC2/BSD-3. Корневые LICENSE/NOTICE отсутствуют, проект UNLICENSED; legal mapping и выбор лицензии оставлены владельцу.

## M11 acceptance checkpoints

Попытка v1 на fingerprint `sha256:6ba40b...` прервана после двух M28 findings; её отчёт — `docs/validation/m11-current-tree-acceptance-20261018.txt`, статус `INTERRUPTED`. Это не PASS.

Gate v2 завершился **PASS на all-88 candidate set** за 3009.434s на прежнем fingerprint `sha256:7600aec0f32445b13ff22d2d088c6b402a5a744fca5399930d3112d74a7911db` (`docs/validation/m11-current-tree-acceptance-20261018-v2.txt`). Он не подтверждает финальный кандидат после педагогических исправлений и локального включения 128 lessons. Текущий проверяемый fingerprint — `sha256:c9c85f0f9b6d2622bc29cefad12e2feac1b349950f1b7c3e579a05ed556af2c0`; `make test` уже прошёл на этом tree. Следующий M11 step — commit и полный all-88 gate v3 с новым scope `all 88 advanced lessons`.

Предыдущий gate v2 воспроизводится командой:

```sh
python3 scripts/check_m11_acceptance.py \
  --report docs/validation/m11-current-tree-acceptance-20261018-v2.txt \
  --log artifacts/m11-full-final-v2-20261018.log \
  --cache-dir /tmp/oga-m26-20261018
```

Для v3 проверить `Result`, exit code, `Scope: all 88 advanced lessons`, и равенство source fingerprints. Изменения `internal/content`, `internal/grading`, `internal/runner`, `internal/competency`, `go.mod`, `go.sum` или M11 gate scripts делают результат STALE. Report: `docs/validation/m11-current-tree-acceptance-20261018-v3.txt`; log: `artifacts/m11-full-final-v3-20261018.log`; cache: `/tmp/oga-m11-v3-20261018`.

## Следующие шаги

1. Дождаться session `56253` и проверить v3: PASS, exit0, `Scope: all 88 advanced lessons`, равные fingerprints `c9c85f0f…af2c0`. Не менять fingerprinted M11 paths пока gate идёт.
2. После результата обновить renderer и прогнать `npm run test:progress`.
3. Запустить полный `make test` после последней guard и Tutor retrieval правок; затем full desktop/mobile E2E с `ACADEMY_E2E_RUNNER=1 ACADEMY_E2E_PROJECT_RUNNER=1 ACADEMY_E2E_P3_RUNNER=1` (без P3 флага suite пропускает реальный P3).
4. Исправить M12 P2 слабые memory/PID assertions после gate, наблюдать реальные `memory.events`/`pids.events` и cleanup, повторить `make sandbox-test` и failure recovery.
5. Завершить M11 traceability, M12 угрозы/resource-abuse/restart/responsive/theme/a11y, затем M13 clean archive/bootstrap/recovery, requirement matrix, Gitleaks, dependency notices и runbook. Не публиковать и не выбирать лицензию за владельца.
6. Cache-only report готов: `docs/validation/m13-license-inventory-20261018.json`; сканер и tests добавлены в `make test`. Дальше выполнить text-level SPDX/legal review для найденных текстов и подготовить NOTICE только после верификации; неизвестные/uncached Go модули не выдавать за подтверждённые. Root LICENSE/NOTICE отсутствуют, manifests UNLICENSED; проектную лицензию выбирает владелец.
7. Обновить план/progress/handoff, `git diff --check`, локальный checkpoint. M11–M13 не закрывать без фактического gate evidence.

## Лимиты и безопасность

Последний `get_usage_limits` **2026-10-18 11:28 Asia/Atyrau**: `ordinaryUsageAllowed=true`, primary **88%**, weekly **29%**; бесплатный reset credit доступен, но не использован, покупок/сбросов не делать. Primary reset **2026-10-18 13:55:42**, weekly reset **2026-10-25 03:38:23 Asia/Atyrau**. Единственный heartbeat `open-go-academy-m13` должен оставаться one-shot на следующий limiting reset +1 минута; не создавать daily repeat/duplicate.

Работать только в репозитории. Не читать/удалять/коммитить `.data/academy.db`, `session.secret`, `issuer.key`; не останавливать пользовательские сервисы на 8380/8080; QA DB/keys только disposable. Не публиковать, не тратить деньги, не включать public multi-tenant execution. Чат оставить открытым.
