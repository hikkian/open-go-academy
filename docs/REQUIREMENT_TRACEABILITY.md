# Матрица прослеживаемости требований

Состояние на 2026-10-16, основано на текущем [handoff](../AGENT_HANDOFF.md), плане и имеющихся validation artifacts. `PASS` означает только указанный ограниченный gate, а не завершение всего milestone. `PARTIAL` — есть реализация, но обязательные критерии milestone ещё открыты. Ни одна строка не подразумевает независимый security certification или публичную эксплуатацию.

| Требование / критерий | Source | Автоматическая проверка | Evidence / статус / оговорка |
|---|---|---|---|
| M0: версии, альтернативы и ограничения | `docs/research/*`, `docs/architecture/ADR-0001-tech-stack.md`, `THREAT_MODEL.md` | Нет единого research test | `docs/research/VERSION_MATRIX.md`; частично завершено в рамках текущего решения, дальнейшая актуализация нужна при upgrade. |
| M1: reproducible local startup/build | `Makefile`, `scripts/go.sh`, `cmd/api`, `.github/workflows/ci.yml` | `make test`; CI foundation job | `docs/validation/fresh-bootstrap.txt`, `foundation.txt`; Linux local profile подтверждён. Fresh M13 clone acceptance ещё не выполнен. |
| M2: content model, safe lesson rendering, first loop | `internal/content`, `internal/httpapi`, `web/src` | `go test ./internal/content ./internal/httpapi`; Playwright | `docs/validation/e2e-first-loop.txt`; 40 ранних глав опубликованы. Полный acceptance для всех рангов — M11. |
| M3: isolated execution and grading | `cmd/runner`, `internal/runner`, `internal/grading` | `make sandbox-test`; `go test -race ./internal/runner` | `docs/validation/sandbox-local.txt`, `e2e-final.txt`; локальный rootless OCI runner проверен ограниченными probes. Это не публичный multi-tenant sandbox. |
| M4: урок → edit → run → submit → persisted progress | `internal/httpapi`, `internal/store`, `web/src/Editor*`, `tests/` | `make e2e` при поднятых сервисах | `docs/validation/e2e-first-loop.txt`; подтверждён для локального сценария. |
| M5: карьерная карта, XP, навыки, шесть рангов | `internal/learning`, `internal/store`, `web/src` | `go test ./internal/learning ./internal/store`; Playwright | Early rank progression работает. Advanced rank policy/evidence ещё неполны; статус milestone partial. |
| M6: project workspace, сохранение и runner | `internal/projects`, `internal/projectrunner`, `internal/store`, `web/src/ProjectWorkspace.tsx` | `go test ./internal/projectrunner ./internal/httpapi`; `scripts/project-runner.py`; `tests/p4-p5-evidence.spec.ts` | P1/P2/P3 и P4/P5 API/workspace интегрированы. P4/P5 rootless Submit + immutable evidence E2E прошли; P5 machine result явно не заменяет human review. См. `docs/validation/p3-*`, `p4-p5-browser-acceptance-20261016.txt`, `p4-p5-mobile-browser-20261016.txt`. Полные rank/manual gates открыты. |
| M7: экзамены и trusted rank policy | `internal/assessment`, `internal/grading/specs/*-exam-*.json`, `internal/httpapi/exams.go`, `internal/store/exams.go` | `go test ./internal/grading ./internal/assessment ./internal/httpapi ./internal/store`; `tests/zz-exam-catalog-ui.spec.ts` | Objective definitions now cover all six ranks. All nine advanced reference implementations run through their actual grading spec drivers and public/private vectors; see `advanced-exam-grade-correction-20261016.txt`. `advanced-exam-browser-20261016.txt` verifies API/UI. Only Zero/Junior/Junior+ awards are configured. Advanced entries are visible but unavailable: no immutable rank policies or persisted human-review gate is registered, and the award guard remains closed. |
| M8: optional AI Tutor, consent, budget, no-AI path | `internal/tutor`, `internal/httpapi/tutor.go`, `web/src/Tutor.tsx` | `go test ./internal/tutor ./internal/httpapi -run Tutor`; `npm run e2e -- --grep tutor` с mock | `docs/validation/tutor-provider-acceptance.txt`; synthetic provider проверен, реальный cloud/API вызов не выполнялся. |
| M9: badges/credentials/passport privacy and verification | `internal/credentials`, `internal/credentialrender`, `internal/httpapi/credentials.go` | `go test ./internal/store ./internal/httpapi -run Credential`; browser tests | `docs/validation/credential-artifacts.txt`, `credential-browser.txt`; локальная self-issued credential, не accredited/trusted public credential. |
| M10: Zero → Junior+ полный курс и P1/P2 | `internal/content/data`, `internal/content/solutions`, `content/projects/p1`, `content/projects/p2` | `go test ./...`; native assessment chain; browser P2 E2E | `docs/validation/m10-complete-technical.txt`, `m10-complete-full-browser.txt`; M10 помечен выполненным текущим handoff. |
| M11: Middle → Senior course, P3–P5, exams, human review | `internal/content/data`, `content/projects/p3`–`p5`, `internal/projects`, `internal/assessment`, `internal/store` | `TestM11DraftCatalogAcceptance`; full `make test`; P3/P4/P5 rootless browser tests | `m11-current-tree-acceptance-20261016.txt` — latest current-tree run PASS: 530 subtests, 0 failures, 2,985.278s. 88 chapters remain draft. P4/P5 rootless Submit and immutable evidence passed desktop/mobile workspace axe/no-overflow. Six objective exam definitions exist and references match grading vectors. A signed offline manual-review workflow persists exact project decisions and the fingerprint of the allowlisted key that verified each new review; the migration does not trust legacy records. It is not transactionally linked to rank policy and key fingerprints cannot prove reviewer identity or independence. Advanced policies, full pedagogy, project/artifact review and trusted Senior review remain open. M11 PARTIAL. |
| M12: security, performance, accessibility, E2E audit | `docs/architecture/THREAT_MODEL.md`, `tests/`, `docs/validation/` | Whole Go race/vet, Playwright/axe per existing suites | Выборочные Go race/vet, rootless probes, screenshot/axe и browser acceptance есть; свежий полный security/load/viewport matrix и перечень закрытых high/critical findings отсутствуют. IN PROGRESS. |
| M13: release preparation, clean install, licenses, traceability | `README.md`, `GETTING_STARTED.md`, `CONTRIBUTING.md`, `SECURITY.md`, `docs/architecture/*`, this file | Clean archive `make bootstrap/build/test`; 48-test `npm run e2e`; P1–P5 Playwright project matrix; `python3 scripts/recovery-check.py`; npm/go vulnerability audits | Exact `2b00dfc` archive bootstrap/build/recovery and full desktop/mobile E2E passed 38 tests / 10 intentional skips / 0 failures; complete test/migration and P1–P5 evidence are in the M13 report. Full Go test/race/vet/TS/build passed on preceding code-identical commit `746bbd5`. Dependency metadata inventory and expanded traceability exist, but legal/SPDX/notices, secret scan, remaining curriculum/governance traceability and owner project-license choice remain open. IN PROGRESS. |

## Детальная проверка требований MASTER_SPEC

Эта таблица дополняет milestone matrix для требований, которые легко ошибочно посчитать покрытыми только по наличию файлов. `PARTIAL` означает, что структурные тесты существуют, но ручной критерий или часть требования остаётся недоказанной.

| Требование | Реализация | Автоматические доказательства | Статус и границы |
|---|---|---|---|
| E3 — обязательная структура и полнота урока | `internal/content/content.go`, `internal/content/data/*.json`, `*.md`, эталонные `internal/content/solutions/` | `TestEveryPublishedGoExampleAndFunctionReference`, `TestPublishedOrderAndPrivateGradingBoundary`, `TestM11DraftCatalogAcceptance` | **PARTIAL.** Проверяются metadata, prerequisites, ссылки, упражнения, примеры и grading references. Автотест не подтверждает понятность теории с нуля, перевод English-терминов при первом употреблении, pedagogical completeness всех обязательных критериев и качество заданий после независимого преподавательского review. 88 M11 глав остаются draft и не публикуются автоматически. |
| E5 — обновляемый и воспроизводимый курс | `internal/content/manifest.go`, `internal/content/manifest.json`, lesson version metadata в `internal/content/data/*.json` | `TestCatalogAndReferenceSolutions`, `TestPublishedOrderAndPrivateGradingBoundary`, `TestM11DraftCatalogAcceptance` | **PARTIAL.** Manifest содержит версию curriculum/Go/dependencies, дату, источники и compatibility notes; production загружает только опубликованные lesson IDs. Нет автоматической проверки актуальности даты и доступности source URLs; текущий dependency inventory перечисляет `stdlib`, а воспроизводимость версий всех build/development dependencies не доказывается этим manifest. |
| H5 — governance open-source проекта | `README.md`, `GETTING_STARTED.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `docs/architecture/*`, `.env.example`, `.github/workflows/ci.yml` | CI и команды проверки перечислены в документах; governance acceptance test отсутствует | **OPEN.** Основные guides и CI есть, но нет выбранной owner-ом project license, `LICENSE`/`NOTICE`, issue/PR templates и course-authoring guide. npm manifests остаются `UNLICENSED`; проект нельзя объявлять готовым к open-source распространению до выбора лицензии. |
| D2 — ограниченный изолированный Go execution | `internal/runner/runner.go`, `internal/projectrunner/engine.go`, `p4.go`, `p5.go` | `TestMissingRuntimeFailClosed`, `TestSandboxIntegration`, `TestPreflightCleanupFailureReturnsError`, `TestP4SubmitUsesIsolatedNoDatabaseJobAndTrustedOverlay`, `TestP5SubmitUsesNoDatabaseSandboxAndTrustedPrototypeOracle`; `make sandbox-test`; `go test -race ./internal/runner ./internal/projectrunner` | **PASS для локального режима в пределах проверенных конфигураций; OPEN для публичного multi-tenant использования.** Есть rootless OCI, pinned images, no-network, resource controls and cleanup; evidence не является независимой сертификацией или моделью hardening для публичного общего сервиса. Публичный runner выключен/запрещён до отдельного аудита. |
| Общая Definition of Done | `docs/MASTER_SPEC.md`, `docs/EXECUTION_PLAN.md`, `AGENT_HANDOFF.md` | `make test`, `make sandbox-test`, браузерные и project E2E gates | **OPEN.** Handoff и plan показывают M11/M12/M13 IN PROGRESS; полный курс, advanced rank/manual-review gate, comprehensive security/accessibility audit, final clean-source acceptance, recovery demo и лицензирование не завершены. |

## Сквозные E2E сценарии

| ID | Требование | Source / test | Evidence / состояние |
|---|---|---|---|
| E2E-01 | Onboarding/dashboard/career path | `web/src`, `tests/`; Playwright | `docs/validation/e2e-m5.txt`; ограниченный local path PASS. |
| E2E-02, 03 | Editor Run stdout and compiler error recovery | `internal/runner`, `tests/` | `e2e-first-loop.txt`, `e2e-final.txt`; локальный runner only. |
| E2E-04, 05 | Submit/progress/reload/restart | `internal/store`, `internal/httpapi`; Playwright | `e2e-final.txt`, `api-restart.txt`; local profile. |
| E2E-06 | Multi-file project tests/export | `internal/projects`, `web/src/ProjectWorkspace.tsx`, `tests/p4-p5-evidence.spec.ts` | P1/P2/P3, P4 and P5 project submission evidence verified locally. P4/P5 desktop submit and mobile workspace passed; manual assessment and rank claims stay separate. |
| E2E-07, 08 | Fail denied; qualifying pass awards rank | `internal/store/exams*`, `internal/httpapi/exams*` | `exam-browser.txt`, `m10-native-exam-chain.txt`; early ranks pass. Advanced award intentionally fail-closed pending policy. |
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
