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
