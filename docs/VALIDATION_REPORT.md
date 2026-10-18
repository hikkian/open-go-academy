# Актуальная проверка — 2026-10-18

M0–M10 закрыты; M11/M12/M13 продолжаются. Актуальные исходники, scoped проверки и final gates перечислены в [handoff](../AGENT_HANDOFF.md). Узкие M28 и Middle+21 acceptance gates PASS, а новый full all-88 gate v2 выполняется с fingerprint `sha256:7600aec0f32445b13ff22d2d088c6b402a5a744fca5399930d3112d74a7911db`. Предыдущий all-88 run был остановлен после M28 findings и не принят; прежний full browser suite (40 PASS, 10 intentional SKIP) тоже требует повтора на финальном дереве. Общий M11/M12/M13 PASS не заявляется.

Local load/restart/security-boundary probe — [PASS JSON](validation/m12-local-load-20261018.json):120reads200,119writes200/21quota429 with Retry-After, baseline profile/Zero/no awards/XP0, accepted whole bundle, SQLite integrity/FK and API restart. Current p95 27.473ms GET / 7.084ms PUT on this local host. Проверка локальная, не публичный SLO.

Current M13 recovery probe: `python3 scripts/recovery-check.py` — PASS. A disposable API/SQLite profile/theme/draft was backed up, restored and verified after API restart; evidence in [recovery-check](validation/recovery-check-20261018.txt). Remote backup custody/DR is not implied.

`make test` после M28/M+21 fixes — PASS: Python fingerprint tests, Go unit/integration, P1 reference, vet, race, TypeScript typecheck и Vite build (`artifacts/make-test-final-current-20261018.log`). `npx playwright test tests/project-manual-rubric.spec.ts --reporter=list` — 2 PASS, mobile+desktop; keyboard-only disclosure, hostile HTML inertness, zero API writes, axe и no-overflow. Screenshots лежат в ignored `artifacts/`. Повторные root/web `npm audit --audit-level=low` — 0 vulnerabilities оба; [evidence](validation/npm-audit-root.txt), [evidence](validation/npm-audit-web.txt).

Senior skill evidence regression: exact immutable P5 review quorum now grants communication only through an already-awarded Senior attempt. `/api/learning` and passport retain that evidence after a later review decision; automatic P5 and non-Senior awards do not grant it. A strong read-only reviewer found and verified fixes for current-review-supersession and quorum-size handling; focused store/learning/httpapi tests pass. Mobile skills UI labels the evidence in Russian and explains horizontal table scrolling; targeted browser axe/overflow checks pass.

Codex/Go current recheck: `codex-cli 0.163.0-alpha.10.2`, stable `multi_agent` + `api_key_model_discovery`, current configured Luna builders/researcher and Sol reviewer started successfully; `sh scripts/go.sh version` reports 1.27.2. Official Go release history says 1.27.2 is latest stable on 2026-10-18. Research snapshot: [AI_CODEX](research/AI_CODEX.md), [VERSION_MATRIX](research/VERSION_MATRIX.md).

Gitleaks v8.30.1 — **PASS** on staged changes and 27-commit history (`docs/validation/secret-scan-20261018.txt`). One historical false positive for the public reviewer-key fingerprint UI text is narrowly listed in `.gitleaksignore`; current staged diff has no findings. Re-run against the final clean-source archive before any release.

Current `govulncheck` v1.9.0 against Go 1.27.2 — **No vulnerabilities found**; [evidence](validation/govulncheck.txt). This does not replace application review or a public-hosting threat model.

Ниже сохранён исторический журнал. Старое указание «current tree» относится к дате соответствующей записи, а не к HEAD2026-10-18.

---

# Проверка Open Go Academy — 2026-10-15

## Проверенная область

M0–M6: локальная одиночная установка Linux/amd64, Go1.27.2, Node24.18.0, npm11.16.0, rootless Podman5.8.7/cgroupv2/systemd/seccomp. Исходный репозиторий создан локально; remote отсутствует. Публичный multi-tenant execution и trusted credentials не включены. CI-файл подготовлен, удалённый GitHub CI НЕ выполнялся.

| Команда | Фактический результат | Evidence |
|---|---|---|
| `make test` | PASS: Go unit/vet/race, P1 reference tests/vet, TS typecheck/build | [foundation](validation/foundation.txt) |
| `ACADEMY_SANDBOX_TEST=1 .tools/go/bin/go test -count=1 -timeout=10m -v ./internal/runner` | PASS118.231s: 12 confined integration cases + regressions | [sandbox](validation/sandbox-local.txt) |
| `python3 scripts/restart-check.py` | PASS: actual API process stop/restart сохраняет профиль/тему/черновик, migration upgrades покрыты store tests | [restart](validation/api-restart.txt) |
| `python3 scripts/crash-check.py` | PASS: настоящий SIGKILL runner, stale socket и scoped orphan cleanup, health restored | [crash](validation/crash-recovery.txt) |
| `ACADEMY_E2E_RUNNER=1 npm run e2e` | PASS10 / SKIP2: lesson realGo, profile, draft error/retry, search/skills/criteria, project files/snapshots/ZIP, desktop/mobile | [browser](validation/e2e-final.txt) |
| `ACADEMY_E2E_RUNNER=1 npm run e2e -- --grep 'P1 real'` | PASS1 / SKIP1 после расширения до15 black-box cases + actual Go tests | [P1](validation/project-real.txt) |
| `npm run e2e -- --grep 'project edits'` | PASS2: реальные изменения/перезапуск/версии/ZIP; axe desktop/mobile | [files](validation/project-smoke.txt) |
| `make -C .data/fresh-source bootstrap` | PASS на Git archive61e418b: без node_modules/.tools/.data, Go toolchain скачан автоматически | [bootstrap](validation/fresh-bootstrap.txt) |
| `make -C .data/fresh-source test build` | PASS: clean source unit/vet/race/reference/TS/build, оба native binaries | [clean source](validation/fresh-source.txt) |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` через pinned Go | PASS: No vulnerabilities found | [govulncheck](validation/govulncheck.txt) |
| `npm audit` в root и web | PASS: 0 vulnerabilities | [root](validation/npm-audit-root.txt), [web](validation/npm-audit-web.txt) |

Мобильные SKIP — два ресурсных дубликата real lesson/project runtime. Mobile profile/editor/пersistence/search/skills/project files+ZIP проверены отдельно, запуск Go реально проверен desktop и API negative suite. После последнего изменения module-whitespace validator unit regressions и TS/build повторены; clean archive/full browser результаты выше относятся к указанным предшествующим snapshots. Это точная граница проверки, не заявление remote CI/full-course readiness.

## Что действительно выполнялось

- Content tests исполняют 2 Go snippets и 3 уроковых references; проверяются metadata/dependencies/cycles/внутренние Markdown links. P1 reference имеет отдельный Go module; `make test` явно запускает его, root `go test ./...` сам по себе nested modules не включает.
- Sandbox: multi-file main/functions, compile error, host sentinel/read-only root, blocked TCP+mount, output flood, infinite loop, actual OOM137, writable-disk exhaustion, PID bound, formatting, cancellation/descendants+orphan cleanup, literal shell-like Args, JSON asset и Go test mode.
- Preflight фактически подтверждает rootless/cgroup v2, uid65532, memory.max805306368, swap0, pids.max128, cpu.max100000/100000, Seccomp2, NoNewPrivs1, Go1.27.2. Сеть none; root read-only/caps dropped. Scratch tmp128MiB(noexec)+work192MiB(exec)+cache192MiB(exec), shm16MiB. Cache отдельно exec: Go может запускать кэшированный бинарник; это внутри той же ограниченной среды. Deadline45s/job, cleanup10s, source64KiB, JSON64KiB/8, output32KiB, concurrency1, overload rejected. Project evaluation15 fixed cases + unit tests имеет общий180s deadline; API200s write deadline. 12 execution requests/min,120 mutations/min.
- P1 stdout/exit/errors проверяет доверенный API вне sandbox. Пользовательский `go test` не является единственным основанием результата. TestMain не допускается первым проектом, нужны genuine TestX(*testing.T); качество assertions/README/Git требуют отдельного review. `p1:automatic` не завершает ручную часть и не повышает rank.
- XP ledger уникален на exercise/event; concurrent replay, retention100 recent + original evidence, snapshots10/project, transaction rollback/IDOR/upgrade/reopen покрыты. Forged rank/progress/event/unsafe bind/Host/Origin/CSRF rejected в API tests.

## Исправленные провалы

- Cold compile исчерпывал первоначальный scratch: бюджеты калиброваны и negative suite повторён.
- Cleanup после раннего cancel конфликтовал с default Podman stop10s: stop-timeout0/rm time0 и10s cleanup; latch unhealthy, admission recheck и named preflight error покрыты regressions.
- Output-only grader допускал нарушение структуры/недостижимые calls: для начальных задач direct-statement AST rules + regressions.
- Contrast/focus defects исправлены; axe повторно прошёл light/dark/desktop/mobile и project/skills tables. Автоматический axe не равен полной WCAG сертификации.
- Go test/run исполнял бинарники на noexec scratch/cache: выделены ограниченные exec work/cache, `/tmp` остался noexec, все runtime cases повторены.
- Project quotas/exported go.mod/TestMain-only несоответствия устранены. Single-line/неподходящий Unicode/control whitespace в go.mod отвергаются; допустимые ASCII spaces/tabs/CRLF проверены.
- Ошибки тестовых locators (label и banner×) и reused mobile marker исправлены, actual application outcomes проверены повторно.

## Проверка интерфейса

Playwright screenshots/trace доступны в ignored `artifacts/`: desktop/mobile light/dark, skills, project, golden-path и project-passed. Desktop и mobile images визуально просмотрены в сессии; панели читаемы, внешнего horizontal overflow в проверенных viewport нет. Trace сохраняется при failure. Артефакты регенерируются `make e2e`; скриншоты не включены как сторонние assets продукта.

## Частично / не проверено

Полный курс (M10/M11), exams/ranks (M7), Tutor gateway (M8), badges/credentials/passport (M9), performance/full security/accessibility audit (M12), cross-platform release (M13) ещё не реализованы. Доступен один полный Zero Day1 и P1 baseline, а не все главы. Readme/Git quality требует review; local self-hosting не удостоверяет личность/стаж. Нет public deployment, production load tests, exhaustive syscall/UDP/DNS probes, independent microVM/gVisor certification или доказательства отсутствия kernel escape. Kernel/runtime остаются trust boundary; local профиль не разрешает Internet multi-tenant hosting. License ещё не выбрана владельцем. Исследовательские источники и ADR фиксируют выбор, не дают гарантий будущих версий.

## Последняя конфигурация запуска

Порт8080 занят существующим Python-сервисом пользователя (не изменён). Default API/Viteproxy/.env/README перенесены на8380; E2E8081 и restart8090 остаются отдельными. Логready теперь только после успешного bind. После изменений вновь выполнены Go unit/vet/race+typecheck/build, livehealth GET подтверждён отдельно. Source archive61e418b относится кпредыдущейконфигурации; docs не утверждает свежийclone именно с новымпортом.

## Последние регрессии

Дополнительно после fullsuite: Unicode-lowercase Test suffix не принимается как genuineGo test; panic/fatal stack traces не считаются контролируемой CLI-ошибкой. Unit/vet/race+TSbuild PASS. Worker transport1MiB и client result1MiB согласованы с JSONescaping128KiBsource/data: mockedclient regression + realUnixSocket formatter PASS360084 encoded bytes/60035 source bytes ([transport](validation/transport-format.txt)). Native per-job source/output quotas не увеличены. Podman default hostproxy forwarding выключен `--http-proxy=false`; real bounded integration с syntheticHTTP_PROXY/HTTPS_PROXY PASS7.516s, только boolresult без реальных envvalues ([proxy](validation/proxy-env.txt)). Latest live `/api/health` на8380 фактически `status:ok,runner_available:true`.

## M7/M8 — 2026-10-15, дополнительная проверка

M7 native browser: `ACADEMY_E2E_RUNNER=1 npm run e2e -- --grep exam` — 3 PASS, 1 explicit mobile native duplicate SKIP, 1.9min. Проверены actual sandbox cases, провал без award, два CAS draft tabs, retake, Zero award, reload/history. Store/API tests: immutable criteria, replay/forged pass/prerequisite guards, concurrent revision; independent EXAM_REVIEW closes both Medium. DB dual-process probe `python3 scripts/api-lock-check.py` PASS. Визуальный просмотр выявил отсутствующие base exam styles; исправлены и добавлен стабильный путь повторения полностью пройденного единственного урока. Full regression ещё выполняется/повторяется, результат ниже будет актуализирован.

M8 `make test` PASS: Go unit/vet/race, content references, TS typecheck/build. ActualHTTP httptest gateway/API проверяют consent, context minimization, redaction, transient retry/fallback, bounded response/cancellation, SSRF special-ranges/DNS pinning, ledger/concurrent budgets, no rank authority. Реальной LLM/платных запросов нет. Independent TUTOR_REVIEW: special-IP и remote Markdown image flaws закрыты; startup trim/test harness fixes подтверждены. Ответы модели не исполняют инструменты и не могут менять rank. Автоматический кеш свободных запросов отключён из-за приватного контекста; `cached:false` честный. Долларовая стоимость не выдумывается: UI показывает token usage/консервативный резерв, цена зависит от самостоятельно настроенного провайдера.

M7/M8 gate closed: full core run14PASS5SKIP1FAIL — единственный FAIL устаревшее ожидание verified=false в P1 после уже сданного Zero. Исправлено на before/after invariant; native P1 targeted rerun1PASS1SKIP3.1min. [P1 regression](validation/exam-project-regression.txt), [native exam](validation/exam-browser.txt), [actual HTTP tutor](validation/tutor-browser.txt):3PASS1SKIP3.8s. Новый полный combined green run после этого исправления пока не выполнялся. Нельзя представлять test double как реальную LLM; provider billing/remoteTLS не проверены.

## M9 — 2026-10-15, local credential vertical slice

`make test` PASS (unit/vet/race + trustedrefs + TS/build). `ACADEMY_E2E_RUNNER=1 npm run e2e -- --grep exam`:3PASS1explicitmobile-runtime-duplicateSKIP2.0min. Native Zero→earnedaward→UIissueprivatecredential→JSON/PDF/PNGdownloads→independentNodeEd25519verification→tamperedrankrejected→independentjsQRdecode→consentedpublish→publicverifyname/criteria→axe→revoke→revokedstatus→unpublish404→passportoptin/optout passed ([browser](validation/credential-browser.txt)). Browser screenshots: artifacts/passport-tested.png,credential-verify.png; visually inspected. The downloaded credential is an actual local test record, not a third-party attestation.

`python3 scripts/credential-key-check.py`: actualAPI restart with copiedE2EDB, missingkeyrejected, differentvalidkeyrejected, originalkeyrestored PASS ([key recovery](validation/credential-key-recovery.txt)). User .data/academy.db was not mutated by this probe. `govulncheck@v1.9.0 ./...` no vulnerabilities; root/web npm audit0 ([Go](validation/credential-vulnerabilities.txt)). New deps gpdf1.0.13/x/image0.47.0, fontlicense/provenance and QRtestdeps documented CREDENTIAL_IMPLEMENTATION.md. Current schema is a local fixed-struct envelope with domain-separatedEd25519, not JCS/OB/VC. PDF itself is unsigned; JSON is signed. PublicQRlocalhost requires a running accessible installation; no hostedissuertrust.

Renderer actual PDF→PopplerPNG/pdftotext and certificatePNG inspected. Found/fixed doubled boldtext caused by Drawer.Dot mutation; rerendered. Independent jsQR read same verifyURL from QRPNG/fullPNG/PDFpreview. Unit decoder-only tests do not independently scan QR; browser+Node tests do. CREDENTIAL_REVIEW closes3Medium. Fullsuite after M9 not yet rerun. Passport activity graph and explicit project overview are still pending: M9 remains IN PROGRESS despite passing credential vertical slice.

Passport desktop/mobile light/dark axe and documentoverflow:2PASS4.6s, [layout](validation/passport-layout.txt). Screenshots artifacts/{desktop,mobile}-passport-{light,dark}.png. Live default8380 health actual statusok/runner_availabletrue after latest nativeAPI startup. M9 remains incomplete pending explicit project overview, actualactivitygraph and badgeeventdates; fullcombinedregression pending.

## M9 final gate — 2026-10-15

Обзор проектов, фактические badge timestamps и activity graph добавлены и покрыты fixed-date evidence tests; повторы/неверные попытки не считаются подтверждениями, private activity не попадает в publicpassport. Final `make test` PASS. Combined native browser: **17PASS/5explicitSKIP/0FAIL6.0min**, [full output](validation/m9-full-browser.txt). Skips: mobile native exam/lesson/P1 duplicates и2test-providercases, которые выполняются отдельным explicitprovider run. До этого fullrun16PASS/5SKIP/1theme-assumptionFAIL: desktoppassport сохранялdarktheme, mobilelessontest ожидалlight. Test теперь устанавливает свою исходную тему; targeted2PASS10.4s ([regression](validation/theme-regression.txt)); последующий combinedrun выше allgreen. Productpersistedtheme поведение корректно.

Actual browser-downloaded PDF/PNG independentlyscanned and Cyrillicextracted via `node scripts/credential-artifact-check.cjs`:PASS ([artifacts](validation/credential-artifacts.txt)). M9 DONE в local-onlyscope. Hostedissuertrust, OB/VC compliance, fullcourse/M10–M11, publicexecution, crossplatform/fullM12/M13 не заявляются.

## 2026-10-15 — M10 partial:32chapters andP2

`./scripts/go.sh test -count=1 -timeout=5m ./internal/content ./internal/grading ./internal/httpapi ./internal/learning`:32published Goexamples+references PASS27.092scontent. `make test`32basePASS unit/vet/race/TSbuild; finalafterlimitedSQLguardrerunpendinglog`.data/m10-checkpoint-final.log`. Fullnativebasic57refsPASS; new36Junior+nativebatchpending`.data/m10-native-http-sql-36.log`.

Fullbrowser`ACADEMY_E2E_RUNNER=1 npm run e2e`:22PASS6SKIP7.2min (20chapterbinary); P2nativebrowser`ACADEMY_E2E_PROJECT_RUNNER=1 npm run e2e -- --grep 'P2 real isolated'`:1PASS1mobileduplicateskip10.2s. 32chapterHTTP/SQLreaderboththemes:2PASS8.0s afteractualcm-scrollerkeyboardaccessibilityfix+freshwebbuild. InitialP2mobilefixturecollisionandstaledistreaderfailureswerenotaccepted; correctedtargetedrerunsabove. P2desktop/mobile screenshotsactuallyviewed.

`python3 scripts/p2-reference-check.py juniorplus-09 juniorplus-10 juniorplus-11 juniorplus-12`: actualPostgreSQL18.6+4SQLfences+referenceHTTP/ownership/transactions/poolMaxConns1 PASS. `.tools/qa-python/bin/python scripts/p2-schema-check.py`:6actualresponsesPASS OpenAPI3.1/formats. `python3 scripts/p2-worker-check.py`: actualseparateworkerRun/Test/Submit44checks+freshapprestart+SQLpersistedfixture/session+cleanupPASS, latestlimitedrolescopeinlog. `python3 scripts/p2-worker-recovery-check.py`: actualcancel/SIGKILL/stalesocket/scopedorphanPASS. Actualpg_hbaauditSCRAM/nohosttrustPASS. NativeexposedcleanupdependencyorderingandPGtemporaryUnixserverreadiness; bothfixedwithoutrelaxingisolation. IndependentPROJECT_RUNNER_REVIEW tracksfindingsandscope.

Logsunder[validation](validation/) `m10-*`, `p2-worker-*`, `p2-pg-hba-audit.txt`. No hosted/multi-tenant certification, realLLMAPI/payment/publication; fullcurriculum/advancedranksnotyetcomplete.

## 2026-10-16 — M10 complete code, final validation pending

40chapters/120tasks. ContentGoexamples/referencesPASS `.data/m10-40-promotion.log`; all117newrefsactualsandboxPASS in57+36+24batches; actualP2pipeline44checks+SQLandfreshserverPASS (previouslogs). NewJunior/Junior+ nativeAPIgrading→atomicranks→replaydeny→signedcredentials **PASS64.30s**, with explicitseedfixturesforcourse/priorZeroandactualsandboxforsixnewtasks. FocusedAPI/store/assessment/gradingPASS; previousfullmakePASS. PrivateZerooutputleak, inputmutationcheck, predeclaredappend/capguard, unknown-roleowneraccess regressionfixedandreviewed. Newcert6criteriaPDFinitially2pages, QRmetadata3collayoutfix→actual1page; PDF/PNGvisuallyviewed+independentjsQRbothcorrect. Full34browsernewcatalogruninterruptedonduplicateeditorlandmark; fixeddistinctscrollLabels+freshdist, rerun `.data/m10-final-full-browser-fixed.log` stillRUNNING. Latestfinaltechnical`.data/m10-final-technical-complete.log` pendingconfirmation. No claimallfinaltestsPASSyet.

## M10 финальная приёмка — 2026-10-16

**PASS:** `make test`; full`ACADEMY_E2E_RUNNER=1 ACADEMY_E2E_PROJECT_RUNNER=1 npm run e2e` **27passed/7expectedskipped/0failed7.6min**. Skips heavy mobile runtime duplicates andoptionalHTTPprovider-onlyTutor. New117starterbuild gate+allreferences/examplesPASS48.402s. All117refs actualsandboxPASS57+36+24, pluschangedauthpolicy3refs/18matrixcases targetedPASS. NativeJunior→Junior+→credentials64.30sPASS(explicitprerequisitefixtures, realcontainergrading); P2native44checks+actualSQL/SCRAM/restart/cancel/SIGKILLcleanupPASS. Newcert6criteriaPDF+PNG visuallyinspected; PDF1page; independentjsQRmatches. Beginnerstarter/textvariantsfixed, secondfullbrowserfailurescamefromtestfixtureconfirmation/sharedDB/optionalComposeassumptions; separateUIAPI/DB andtarget6PASS26.8s, thenFULLcleanrerun27PASS above. No knownopenHigh/Medium in reviewedscopes; notfullM12audit/crossplatform/publichostingcertification.

Evidence:`docs/validation/m10-complete-full-browser.txt`,`m10-complete-technical.txt`,`m10-all-starters-fixed.txt`,`m10-auth-expanded-native.txt`; independentreviews`CURRICULUM_REVIEW.md`,`STARTER_VARIANTS.md`,`ASSESSMENT_CATALOG_REVIEW.md`,`PROJECT_RUNNER_REVIEW.md`.
