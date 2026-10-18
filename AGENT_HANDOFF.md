# Open Go Academy — точка продолжения

Обновлено **2026-10-18 10:55 Asia/Atyrau**. Локальный репозиторий `master`, remote нет. Working tree содержит финальный M11 candidate; `make test` прошёл, полный source-fingerprinted gate v3 ещё не запускался. **M0–M10 DONE; M11–M13 IN PROGRESS.** Полный проект не завершён.

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

- Полный `make test` после content/manifest/acceptance-harness изменений — **PASS**: curriculum quality + fingerprint regressions, Go unit/integration (published 128 catalog), P1 reference, vet/race, TS typecheck и Vite build. Лог: `artifacts/make-test-m11-current-20261018.log`.
- Целевой Senior communication Playwright: **2 passed** desktop/mobile; axe, no overflow, human-review label и token privacy. P3 manual rubric UI Playwright: **2 passed** desktop/mobile; keyboard, no API writes, hostile HTML inertness, axe/no overflow.
- `npx playwright test tests/project-review.spec.ts --reporter=list` — **2 passed** desktop/mobile after Russian reviewer-history label update; signed review flow and trust-limit UI pass.
- `npm run test:progress` — PASS desktop light + mobile dark, 6 workstreams, axe clean/no overflow; progress now distinguishes local catalog availability from M11 acceptance and public release.
- `python3 scripts/local_load_check.py` повторён на текущем API: **120/120 GET 200**, **119 PUT 200 / 21 PUT 429** с Retry-After, draft bundle atomic, restart persistence, rank/XP baseline, SQLite integrity/FK clean. JSON: `docs/validation/m12-local-load-20261018.json`. Ограниченный local probe, не SLO.
- Root/web `npm audit --audit-level=low` — 0 vulnerabilities. Gitleaks v8.30.1 staged-diff и 27-commit history scans PASS; один false positive про public reviewer-key fingerprint UI text исключён точным fingerprint в `.gitleaksignore`. Нужен повтор scan на final clean archive. Evidence: `docs/validation/secret-scan-20261018.txt`.
- `python3 scripts/recovery-check.py` — PASS on disposable local DB: online backup/restore, API restart, profile/theme/draft/migration persistence. `govulncheck v1.9.0 ./...` — `No vulnerabilities found.`
- Full browser на предыдущем checkpoint: 40 PASS, 10 intentional SKIP; после M11/content/manifest правок полный desktop/mobile suite нужно повторить. `make sandbox-test` **PASS144.479s** после P4/P5 AST-guard fix: 13 rootless runner integration cases + admission/cleanup regressions.
- M12 source review выявил P2 узкое место: AST-guard доверенного P4/P5 assessment тестирования не замечал bare `Exit`/`Goexit` через dot-import и сохранял исключение для пользовательской `main()`. Исправлено: запрещены process-exit hooks, dot-import и стандартные `syscall`-вызовы выхода, исключения для `main` нет. Regression cases лишь парсятся. Unit/vet/race по `internal/projectrunner` и полный `make test` после исправления — PASS.

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

Для v3 проверить `Result`, exit code, `Scope: all 88 advanced lessons`, и равенство source fingerprints. Изменения `internal/content`, `internal/grading`, `internal/runner`, `internal/competency`, `go.mod`, `go.sum` или M11 gate scripts делают результат STALE.

## Следующие шаги

1. Проверить diff финального кандидата, включая outcomes/glossary/labs, и синхронизировать `docs/MASTER_SPEC` traceability с P3/P4/P5 review/exam evidence.
2. Закоммитить финальный M11 source; запустить all-88 `check_m11_acceptance.py` v3 на этом commit, подтвердить PASS/exit0/scope/equal fingerprints.
3. Пересобрать progress renderer после v3, повторить desktop/mobile E2E, актуальный `govulncheck`, визуальные screenshots/themes/axe; `make test` уже прошёл, `make sandbox-test` повторно использовать, если runner-код не менялся.
4. Закрыть M11 traceability course→skill→project→exam→rank; M12 — оставшийся threat/resource-abuse, persistence/restart, responsive/theme/accessibility audit; M13 — clean archive/bootstrap/recovery, license/NOTICE inventory and runbook. Не публиковать внешне и не выбирать лицензию за владельца.
5. M12: завершить threat/failure/resource-abuse, persistence/restart, viewport/theme, accessibility/dependency/security audit и остаточные ограничения в `docs/VALIDATION_REPORT.md`.
6. M13: clean archive/bootstrap/recovery на окончательном commit, requirement matrix, повтор Gitleaks для clean archive, dependency license/third-party notice review, runbook. Проектная LICENSE не выбрана владельцем; не выдумывать лицензию и не публиковать без разрешения.
7. Обновить план/progress/handoff, выполнить `git diff --check`, сохранить локальный checkpoint. M11–M13 не закрывать до фактических gate criteria.

## Лимиты и безопасность

Последний `get_usage_limits` **2026-10-18 10:55 Asia/Atyrau**: `ordinaryUsageAllowed=true`, primary **63%**, weekly **26%**, reset credit не использован. Primary reset **2026-10-18 13:55:42**, weekly reset **2026-10-25 03:38:23 Asia/Atyrau**. Единственный heartbeat `open-go-academy-m13` должен оставаться one-shot на следующий limiting reset +1 минута; не создавать daily repeat/duplicate.

Работать только в репозитории. Не читать/удалять/коммитить `.data/academy.db`, `session.secret`, `issuer.key`; не останавливать пользовательские сервисы на 8380/8080; QA DB/keys только disposable. Не публиковать, не тратить деньги, не включать public multi-tenant execution. Чат оставить открытым.
