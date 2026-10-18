# M11: текущая сверка Middle и advanced assessment

**Обновлено:** 2026-10-18. Эта заметка заменяет прежнюю read-only сверку, где P2 по M28, Middle27–40 artifacts и отдельным test vectors описывались до remediation. Она не является полной педагогической оценкой курса.

## Текущий evidence flow

- Middle27–40 helper checks сами по себе остаются локальными проверками функции, а не доказательством безопасности, runtime resilience или эксплуатации.
- Для P3 TaskFlow перед submission UI показывает versioned manual rubric v2 с конкретными M27–40 artifact criteria: actor/tenant boundaries и IDOR negatives; DI graph/cleanup; logs/metrics/traces; measured SLI/SLO/error budget; fault/recovery; version/rollback/release evidence.
- Advanced award связывает course evidence, exact immutable project submission/evaluation, objective exam attempt, manual review version/hash и установленный reviewer-key quorum. `/api/learning` и passport derive rank только из сохранённых awards; подписанный key не подтверждает личность или независимость reviewer.
- M28 Hard теперь вызывает реальный `TaskService` через `TaskRepository` interface с injected fake и grader сверяет error propagation, переданный ID и вызовы. Lesson/reference/starter/grader согласованы; дополнительно исправлен standalone reference, который прежде зависел от grader driver.
- Для P5/Senior человек подтверждает реальные design/ADR/release артефакты через exact-project snapshot; automatic P5 checks остаются только техническим evidence. Senior `communication` skill теперь получает evidence только после awarded Senior attempt с P5 review quorum.

## Остаточные ограничения

- Project/manual review — обязательный workflow; локальная установка не доказывает reviewer identity или независимость, а само наличие rubric не означает, что студент качественно выполнил критерии.
- Не каждый Middle chapter работает с настоящей БД, брокером, Redis или concurrent server. Это ограничение обозначено в уроках; заявленный проектный outcome должен подтверждаться P3 evidence и ручным review, а не helper pass.
- Педагогическая вычитка 88 draft bundles, factual accuracy, prerequisite flow и novice usability ещё не завершены. M11 full automatic gate сейчас выполняется для текущего source fingerprint.
- Мобильный интерфейс навыков сохраняет семантическую таблицу с горизонтальной прокруткой; UI показывает явную подсказку, axe и viewport overflow проверяются Playwright.

## Evidence

- `docs/research/M11_AUDIT_MIDDLE27_40.md`, `M11_AUDIT_MIDDLE34_40.md`, `M11_MIDDLE14_23_STATUS_20261018.md`.
- `docs/validation/m11-middle27-32-remediation.txt`, `m11-middle34-40-remediation.txt`, `m11-middle19-26-remediation.txt`.
- P3 rubric source/API: `internal/projects/`, `internal/httpapi/project_review_outcomes_test.go`, `tests/project-manual-rubric.spec.ts`.
- Текущий automatic all-88 PASS можно заявлять только после проверки report `Result`, source fingerprints и exit status в handoff.
