# Матрица прослеживаемости требований

Обновлено 2026-10-18 по [актуальному handoff](../AGENT_HANDOFF.md). `PASS` означает только указанный ограниченный gate, а не завершение всего milestone. `PARTIAL` — есть реализация, но обязательные критерии milestone ещё открыты. Ни одна строка не подразумевает независимый security certification или публичную эксплуатацию.

## Срез текущей сессии — 2026-10-18

- `make test` прошёл на текущем локальном каталоге из 128 уроков: curriculum-quality/fingerprint Python tests, Go unit/integration/P1 reference/vet/race, TypeScript/build. `scripts/test_m11_curriculum_quality.py` проверяет уникальность outcomes/self-checks, длины глав и структурные практические материалы.
- All-88 gate v2 завершился PASS на старом fingerprint `sha256:7600aec0…a7911db`. После этого content, manifest и gate harness изменились; текущий полный M11 gate v3 должен вывести `Scope: all 88 advanced lessons`, PASS, exit0 и совпадающие source fingerprints.
- На финальном кандидате исправлены дубли outcomes/self-checks, добавлены P3/Redis labs и Middle+ тематические блоки. PostgreSQL outbox integration и два pinned Redis CLI опыта исполнены локально с подтверждённой очисткой временных ресурсов.
- Current API disposable load/restart probe: 120/120 concurrent reads200, 119 draft writes200 +21 quota429/Retry-After, restart persistence and SQLite integrity/FK PASS; локально, без SLO assertion.
- `python3 scripts/recovery-check.py` — PASS на текущем дереве: disposable SQLite online backup/restore, API restart и профиль/тема/draft/migrations preserved; remote backup custody не проверена.
- Pedagogical review всех 88 M11 chapters, M12 comprehensive E2E/security/a11y matrix, M13 clean-source/recovery and license choice остаются открытыми.

| Требование / критерий | Source | Автоматическая проверка | Evidence / статус / оговорка |
|---|---|---|---|
| M0: версии, альтернативы и ограничения | `docs/research/*`, `docs/architecture/ADR-0001-tech-stack.md`, `THREAT_MODEL.md` | Нет единого research test | `docs/research/VERSION_MATRIX.md`; частично завершено в рамках текущего решения, дальнейшая актуализация нужна при upgrade. |
| M1: reproducible local startup/build | `Makefile`, `scripts/go.sh`, `cmd/api`, `.github/workflows/ci.yml` | `make test`; CI foundation job | `docs/validation/fresh-bootstrap.txt`, `foundation.txt`; Linux local profile подтверждён. Fresh M13 clone acceptance ещё не выполнен. |
| M2: content model, safe lesson rendering, first loop | `internal/content`, `internal/httpapi`, `web/src` | `go test ./internal/content ./internal/httpapi`; Playwright | `docs/validation/e2e-first-loop.txt`; 128 глав включены в локальный manifest. Финальный M11 acceptance и responsive browser test после local catalog expansion нужны до общего завершения. |
| M3: isolated execution and grading | `cmd/runner`, `internal/runner`, `internal/grading` | `make sandbox-test`; `go test -race ./internal/runner` | `docs/validation/sandbox-local.txt`, `e2e-final.txt`; локальный rootless OCI runner проверен ограниченными probes. Это не публичный multi-tenant sandbox. |
| M4: урок → edit → run → submit → persisted progress | `internal/httpapi`, `internal/store`, `web/src/Editor*`, `tests/` | `make e2e` при поднятых сервисах | `docs/validation/e2e-first-loop.txt`; подтверждён для локального сценария. |
| M5: карьерная карта, XP, навыки, шесть рангов | `internal/learning`, `internal/store`, `web/src` | `go test ./internal/learning ./internal/store`; Playwright | Early rank progression работает. Advanced rank policy/evidence ещё неполны; статус milestone partial. |
| M6: project workspace, сохранение и runner | `internal/projects`, `internal/projectrunner`, `internal/store`, `web/src/ProjectWorkspace.tsx` | `go test ./internal/projectrunner ./internal/httpapi`; `scripts/project-runner.py`; `tests/p4-p5-evidence.spec.ts` | P1/P2/P3 и P4/P5 API/workspace интегрированы. P4/P5 rootless Submit + immutable evidence E2E прошли; P5 machine result явно не заменяет human review. См. `docs/validation/p3-*`, `p4-p5-browser-acceptance-20261016.txt`, `p4-p5-mobile-browser-20261016.txt`. Полные rank/manual gates открыты. |
| M7: экзамены и trusted rank policy | `internal/assessment`, `internal/grading/specs/*-exam-*.json`, `internal/httpapi/exams.go`, `internal/httpapi/rank_policy_registration.go`, `internal/store/exams.go` | `go test ./internal/grading ./internal/assessment ./internal/httpapi ./internal/store`; `advanced_rank_pipeline_test.go`; `tests/zz-exam-catalog-ui.spec.ts`, `tests/manual-review-prerequisite.spec.ts` | Objective definitions cover all six ranks; advanced policies are created from the exact configured reviewer allowlist, rubric hashes and trusted runner profiles. `docs/validation/advanced-rank-pipeline-progress-20261017.txt` records full API flow for Middle/Middle+/Senior: project submit, reject/approval quorum, failed attempt with no award, successful evidence-bound award and replay denial. Default configuration without operator keys/profile stays fail-closed. Key fingerprints prove key possession only. Advanced rank award is implemented conditionally, not a public trust/identity attestation. |
| M8: optional AI Tutor, consent, budget, no-AI path | `internal/tutor`, `internal/httpapi/tutor.go`, `web/src/Tutor.tsx` | `go test ./internal/tutor ./internal/httpapi -run Tutor`; `npm run e2e -- --grep tutor` с mock | `docs/validation/tutor-provider-acceptance.txt`; synthetic provider проверен, реальный cloud/API вызов не выполнялся. |
| M9: badges/credentials/passport privacy and verification | `internal/credentials`, `internal/credentialrender`, `internal/httpapi/credentials.go` | `go test ./internal/store ./internal/httpapi -run Credential`; browser tests | `docs/validation/credential-artifacts.txt`, `credential-browser.txt`; локальная self-issued credential, не accredited/trusted public credential. |
| M10: Zero → Junior+ полный курс и P1/P2 | `internal/content/data`, `internal/content/solutions`, `content/projects/p1`, `content/projects/p2` | `go test ./...`; native assessment chain; browser P2 E2E | `docs/validation/m10-complete-technical.txt`, `m10-complete-full-browser.txt`; M10 помечен выполненным текущим handoff. |
| M11: Middle → Senior course, P3–P5, exams, human review | `internal/content/data`, `content/projects/p3`–`p5`, `internal/projects`, `internal/assessment`, `internal/httpapi/rank_policy_registration.go`, `internal/store` | `TestM11AdvancedCurriculumAcceptance`; `scripts/test_m11_curriculum_quality.py`; full `make test`; all-88 source-fingerprinted gate v3; P3/P4/P5 rootless browser tests; exact review/quorum/supersession store tests | v2 all-88 gate PASS on old fingerprint; current local manifest contains all 128 lessons; current `make test` PASS, including advanced content and uniqueness/length guard. Current all-88 gate v3 and full browser suite remain. Advanced policies pin course/prior-rank/project runner/manual rubric/reviewer quorum; exact review snapshots revalidated at Claim/Finish. Communication derives only from stored Senior P5 review; fingerprints prove key possession, not identity. M11 PARTIAL until v3/E2E/traceability pass. |
| M12: security, performance, accessibility, E2E audit | `docs/architecture/THREAT_MODEL.md`, `tests/`, `docs/validation/` | Whole Go race/vet, Playwright/axe per existing suites | Выборочные Go race/vet, rootless probes, screenshot/axe и browser acceptance есть; свежий полный security/load/viewport matrix и перечень закрытых high/critical findings отсутствуют. IN PROGRESS. |
| M13: release preparation, clean install, licenses, traceability | `README.md`, `GETTING_STARTED.md`, `CONTRIBUTING.md`, `SECURITY.md`, `docs/architecture/*`, this file | Clean archive `make bootstrap/build/test`; 48-test `npm run e2e`; P1–P5 Playwright project matrix; `python3 scripts/recovery-check.py`; npm/go vulnerability audits | Historical exact-commit clean archive/bootstrap/recovery and full desktop/mobile E2E passed 38 tests / 10 intentional skips / 0 failures. Current changes need a final clean-source run. Current Gitleaks staged/history scan reports no leaks with one documented UI false-positive fingerprint. Dependency metadata inventory and expanded traceability exist, but legal/SPDX/notices, final archive scan, remaining curriculum/governance traceability and owner project-license choice remain open. IN PROGRESS. |

## Детальная проверка требований MASTER_SPEC

Эта таблица дополняет milestone matrix для требований, которые легко ошибочно посчитать покрытыми только по наличию файлов. `PARTIAL` означает, что структурные тесты существуют, но ручной критерий или часть требования остаётся недоказанной.

| Требование | Реализация | Автоматические доказательства | Статус и границы |
|---|---|---|---|
| E3 — обязательная структура и полнота урока | `internal/content/content.go`, `internal/content/data/*.json`, `*.md`, эталонные `internal/content/solutions/` | `TestEveryPublishedGoExampleAndFunctionReference`, `TestPublishedOrderAndPrivateGradingBoundary`, `TestM11DraftCatalogAcceptance` | **PARTIAL.** Проверяются metadata, prerequisites, ссылки, упражнения, примеры и grading references. Автотест не подтверждает понятность теории с нуля, перевод English-терминов при первом употреблении, pedagogical completeness всех обязательных критериев и качество заданий после независимого преподавательского review. 88 M11 глав остаются draft и не публикуются автоматически. |
| E5 — обновляемый и воспроизводимый курс | `internal/content/manifest.go`, `internal/content/manifest.json`, lesson version metadata в `internal/content/data/*.json` | `TestCatalogAndReferenceSolutions`, `TestPublishedOrderAndPrivateGradingBoundary`, `TestM11DraftCatalogAcceptance` | **PARTIAL.** Manifest содержит версию curriculum/Go/dependencies, дату, источники и compatibility notes; production загружает только опубликованные lesson IDs. Нет автоматической проверки актуальности даты и доступности source URLs; текущий dependency inventory перечисляет `stdlib`, а воспроизводимость версий всех build/development dependencies не доказывается этим manifest. |
| H5 — governance open-source проекта | `README.md`, `GETTING_STARTED.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `docs/architecture/*`, `.env.example`, `.github/workflows/ci.yml` | CI и команды проверки перечислены в документах; governance acceptance test отсутствует | **OPEN.** Основные guides, CI, issue/PR templates и course-authoring guide есть; владелец ещё не выбрал project license, `LICENSE`/`NOTICE` и юридическую атрибуцию зависимостей. npm manifests остаются `UNLICENSED`; проект нельзя объявлять готовым к open-source распространению до решения владельца. |
| D2 — ограниченный изолированный Go execution | `internal/runner/runner.go`, `internal/projectrunner/engine.go`, `p4.go`, `p5.go` | `TestMissingRuntimeFailClosed`, `TestSandboxIntegration`, `TestPreflightCleanupFailureReturnsError`, `TestP4SubmitUsesIsolatedNoDatabaseJobAndTrustedOverlay`, `TestP5SubmitUsesNoDatabaseSandboxAndTrustedPrototypeOracle`; `make sandbox-test`; `go test -race ./internal/runner ./internal/projectrunner` | **PASS для локального режима в пределах проверенных конфигураций; OPEN для публичного multi-tenant использования.** Есть rootless OCI, pinned images, no-network, resource controls and cleanup; evidence не является независимой сертификацией или моделью hardening для публичного общего сервиса. Публичный runner выключен/запрещён до отдельного аудита. |
| Общая Definition of Done | `docs/MASTER_SPEC.md`, `docs/EXECUTION_PLAN.md`, `AGENT_HANDOFF.md` | `make test`, `make sandbox-test`, браузерные и project E2E gates | **OPEN.** Handoff и plan показывают M11/M12/M13 IN PROGRESS; полный курс, advanced rank/manual-review gate, comprehensive security/accessibility audit, final clean-source acceptance, recovery demo и лицензирование не завершены. |

## Сквозные E2E сценарии

| ID | Требование | Source / test | Evidence / состояние |
|---|---|---|---|
| E2E-01 | Onboarding/dashboard/career path | `web/src`, `tests/`; Playwright | `docs/validation/e2e-m5.txt`; ограниченный local path PASS. |
| E2E-02, 03 | Editor Run stdout and compiler error recovery | `internal/runner`, `tests/` | `e2e-first-loop.txt`, `e2e-final.txt`; локальный runner only. |
| E2E-04, 05 | Submit/progress/reload/restart | `internal/store`, `internal/httpapi`; Playwright | `e2e-final.txt`, `api-restart.txt`; local profile. |
| E2E-06 | Multi-file project tests/export | `internal/projects`, `web/src/ProjectWorkspace.tsx`, `tests/p4-p5-evidence.spec.ts` | P1/P2/P3, P4 and P5 project submission evidence verified locally. P4/P5 desktop submit and mobile workspace passed; manual assessment and rank claims stay separate. |
| E2E-07, 08 | Fail denied; qualifying pass awards rank | `internal/store/exams*`, `internal/httpapi/exams*`, `internal/httpapi/advanced_rank_pipeline_test.go` | Early-rank browser/native chain passes. Advanced API test exercises evidence-bound Middle/Middle+/Senior awards and denial cases. It uses fixed trusted exam references plus a deterministic project-checker fixture; separate P3/P4/P5 rootless E2E covers actual OCI integration. |
| E2E-09, 10, 13 | Private credential, QR/PDF, tamper/revoke | `internal/credentials`, browser tests | `credential-artifacts.txt`, `credential-browser.txt`; self-issued local trust only. |
| E2E-11, 12 | No AI path and mock failure/quota | `internal/tutor`, `tests/tutor.spec.ts` | `tutor-provider-acceptance.txt`; mock transport only, no real model required. |
| E2E-14 | Responsive themes, accessibility | `tests/`, `web/src/styles.css` | Selective screenshot/axe evidence; M12 comprehensive matrix still open. |

## Воспроизводимые базовые команды

Из корня репозитория, на Linux с необходимыми зависимостями:

```sh
make test
make sandbox-test
make build
```

Для browser E2E нужны уже запущенные API и соответствующий runner:

```sh
make e2e
```

Отдельная fresh clone / install / migrate / golden-path / release acceptance пока не пройдена. Не интерпретируйте перечисление команды как утверждение, что она запускалась в текущей проверке.
