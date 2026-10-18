# Open Go Academy — точка продолжения

Обновлено **2026-10-18 09:46 Asia/Atyrau**. Локальный репозиторий `master`, remote нет. Текущее состояние сохранено в локальном Git `HEAD`; remote нет. **M0–M10 DONE; M11–M13 IN PROGRESS.** Полный проект не завершён.

## Продукт и доверительные границы

- Реализованы onboarding/profile, курс на русском, редактор/изолированный Go runner/grading, прогресс и drafts, P1–P5 workspaces, rank exams, XP/skills, локальные credentials/passport и optional AI Tutor. No-AI путь полноценный; public multi-tenant runner не включён.
- 40 глав Zero/Junior/Junior+ опубликованы; 88 Middle/Middle+/Senior существуют как черновики. Ранги отличны от XP. Для advanced award нужны course evidence, exact project artifact, human review/quorum и все objective exam parts. Подпись доказывает владение разрешённым ключом, не личность/независимость/аккредитацию.
- P3 manual rubric v2 виден до отправки и связывает M27–40 artifact criteria с version/hash. Автоматические project requirements не подменяются ручным review. Mobile/desktop UI tests проверяют клавиатуру, hostile HTML как текст, отсутствие лишних API writes, axe и overflow.
- Все 13 canonical skill IDs связаны с course/project evidence или exact P5 human-review evidence уже выданного Senior award. Communication не выводится из automatic P5, а Tutor получает только coarse capped level без reviewer details. См. `docs/research/M11_SKILL_MAPPING_20261018.md`.
- M28 Hard теперь проверяет настоящий `TaskService`→`TaskRepository` fake. Full gate v1 обнаружил standalone reference/import и starter type-tag mismatch; исправлены. Узкий gate M28 PASS31.819s: `docs/validation/m11-middle28-current-20261018.txt`.
- Middle+21 Easy теперь показывает `within` и `errors`, а private vectors покрывают 4 invalid boundaries. Lesson/example/spec синхронизированы; scoped gate PASS31.475s: `docs/validation/m11-middleplus21-easy-current-20261018.txt`.
- Панель прогресса `docs/PROJECT_PROGRESS.html` строится reproducibly из `docs/project-progress.json`. 88/88 — подготовка draft files, не принятие/публикация. Текущий fingerprint-bound gate показывает running. 11/14 — счётчик этапов, не процент объёма работ.
- Независимая read-only педагогическая вычитка и повторный подсчёт одинаковых JSON-строк выявили P2 пробелы: Middle01–14 упражнения не проверяют часть заявленных concurrency/SQL механизмов; Middle21–37 интеграционные лабораторные не дают воспроизводимых шагов; термины Middle01–02/21–37 не всегда переведены при первом употреблении; generic outcomes/self-check повторяются в Middle01–12, 13–24/26, 27–33 и Middle+01–12; Middle+13–23 повторяет общий материал, Middle+18–24 имеет недостаточно специфичные outcomes; Senior06–12 повторяют JSON self-check, Senior13–24 — generic outcomes. Детали: `docs/research/M11_PEDAGOGY_REVIEW_20261018.md`. Не менять fingerprint-входы до завершения gate v2.

## Проверки текущего дерева

- Полный `make test` после M28/M+21 edits — **PASS**: Python fingerprint tests, Go unit/integration, P1 reference, vet/race, frontend typecheck и production build. Лог: `artifacts/make-test-final-current-20261018.log`.
- Целевой Senior communication Playwright: **2 passed** desktop/mobile; axe, no overflow, human-review label и token privacy. P3 manual rubric UI Playwright: **2 passed** desktop/mobile; keyboard, no API writes, hostile HTML inertness, axe/no overflow.
- `npx playwright test tests/project-review.spec.ts --reporter=list` — **2 passed** desktop/mobile after Russian reviewer-history label update; signed review flow and trust-limit UI pass.
- `npm run test:progress` — PASS desktop light + mobile dark, 6 workstreams, axe clean/no overflow. Renderer/test теперь явно показывают running state только для совпадающего fingerprint.
- `python3 scripts/local_load_check.py` повторён на текущем API: **120/120 GET 200**, **119 PUT 200 / 21 PUT 429** с Retry-After, draft bundle atomic, restart persistence, rank/XP baseline, SQLite integrity/FK clean. JSON: `docs/validation/m12-local-load-20261018.json`. Ограниченный local probe, не SLO.
- Root/web `npm audit --audit-level=low` — 0 vulnerabilities. Gitleaks v8.30.1 staged-diff и 27-commit history scans PASS; один false positive про public reviewer-key fingerprint UI text исключён точным fingerprint в `.gitleaksignore`. Нужен повтор scan на final clean archive. Evidence: `docs/validation/secret-scan-20261018.txt`.
- `python3 scripts/recovery-check.py` — PASS on disposable local DB: online backup/restore, API restart, profile/theme/draft/migration persistence. `govulncheck v1.9.0 ./...` — `No vulnerabilities found.`
- Full browser на предыдущем checkpoint: 40 PASS, 10 intentional SKIP; после M28/M+21 fixes full browser suite нужно повторить. Полный runner sandbox test ещё запустить.

## Активный M11 acceptance v2

Попытка v1 на fingerprint `sha256:6ba40b...` прервана после двух M28 findings; её отчёт — `docs/validation/m11-current-tree-acceptance-20261018.txt`, статус `INTERRUPTED`. Это не PASS.

Текущий full gate v2 — **exec session 55884**, исходный fingerprint `sha256:7600aec0f32445b13ff22d2d088c6b402a5a744fca5399930d3112d74a7911db`, лог дошёл до Middle-27. Команда:

```sh
python3 scripts/check_m11_acceptance.py \
  --report docs/validation/m11-current-tree-acceptance-20261018-v2.txt \
  --log artifacts/m11-full-final-v2-20261018.log \
  --cache-dir /tmp/oga-m26-20261018
```

Дождаться завершения; проверить `Result`, exit code, `Scope: all 88 drafts`, и равенство source fingerprints. Любое изменение `internal/content`, `internal/grading`, `internal/runner`, `internal/competency`, `go.mod`, `go.sum` или двух M11 gate scripts делает результат STALE. Не прерывать без нужды.

## Следующие шаги

1. Дождаться M11 gate v2; проверить PASS/exit0 и совпадающие исходный/конечный fingerprint. Gate ещё выполняется; не прерывать и не менять fingerprint-входы.
2. Исправить документированные педагогические замечания, сверяя упражнения/JSON с реально исполняемыми возможностями runner; добавить конкретные наблюдаемые навыки и terminology. Запустить полный gate заново на итоговом содержимом и обновить outcome/evidence traceability.
3. После окончательного M11 дерева выполнить полный desktop/mobile Playwright E2E с актуальным dist, затем `make sandbox-test`, актуальный `govulncheck`, repeat load/restart only if runtime code changes, screenshots/themes/a11y inspection. Сохранить доказательства.
4. Закрыть содержательную/педагогическую вычитку всех 88 draft bundles и traceability course→skill→project→exam→rank; не публиковать автоматически.
4. M12: завершить threat/failure/resource-abuse, persistence/restart, viewport/theme, accessibility/dependency/security audit и остаточные ограничения в `docs/VALIDATION_REPORT.md`.
5. M13: clean archive/bootstrap/recovery на окончательном commit, requirement matrix, повтор Gitleaks для clean archive, dependency license/third-party notice review, runbook. Проектная LICENSE не выбрана владельцем; не выдумывать лицензию и не публиковать без разрешения.
6. Обновить план/progress/handoff, выполнить `git diff --check`, сохранить локальный checkpoint. M11–M13 не закрывать до фактических gate criteria.

## Лимиты и безопасность

Последний `get_usage_limits` **2026-10-18 09:44 Asia/Atyrau**: `ordinaryUsageAllowed=true`, primary **40%**, weekly **22%**, reset credit не использован. Primary reset **2026-10-18 13:55:42**, weekly reset **2026-10-25 03:38:23 Asia/Atyrau**. Единственный heartbeat `open-go-academy-m13` должен оставаться one-shot на следующий limiting reset +1 минута; не создавать daily repeat/duplicate.

Работать только в репозитории. Не читать/удалять/коммитить `.data/academy.db`, `session.secret`, `issuer.key`; не останавливать пользовательские сервисы на 8380/8080; QA DB/keys только disposable. Не публиковать, не тратить деньги, не включать public multi-tenant execution. Чат оставить открытым.
