# AGENT HANDOFF — open-go-academy

Обновлено **2026-10-15**. **M0–M8 DONE в local-only области**, M9 IN PROGRESS, M10–M13 NOT STARTED. Это работающая платформа с первым полным уроком Zero и P1, а не полный курс/публичный hosted продукт.

## Готово

- Verified research: Go1.27.2/экосистема/frontend/Codex/models/sandbox/credentials; ADR и threat model. Codex0.162.0-alpha.17.2 custom agents реально применялись: builder/researcher=gpt-6-luna, reviewer=gpt-6.1-sol, максимум3children.
- Go API/net-http/slog + SQLite modernc1.60.1, migrations1/2, persistent local profile/theme/drafts/evidence/XP, loopback/Host/Origin/session/CSRF, request quotas/retention.
- React19.3.1/TS7.0.2/Vite8.3.4/CodeMirror, safeMarkdown, onboarding/dashboard/course search, six-rank criteria, evidence-linked skills0..4, theme/focus/responsive layouts, retry after save errors.
- Полный Zero Day1,3задания, self-check/explicit solution reveal,2snippets+3references validated. Native Go run/submit/format; server exact+AST grader; XP replay protected, rank remainsZero/unverified.
- Отдельный rootless Podman runner, permission0600 Unix socket. Image pinned digest, uid65532, networknone/read-only/capsdrop/seccomp/noNewPrivs; memory768MiB/swap0/CPU1/PID128; tmp128(noexec)+work192(exec)+cache192(exec)/shm16MiB; source64KiB/JSON64KiB8/output32KiB; deadline45s/cleanup10s/concurrency1. No unsafe fallback; cleanup failure poisons engine; scoped orphan recovery after SIGKILL tested.
- P1 CLI Toolkit starter/reference/rubric, multi-file code/text/JSON, atomic save+snapshots10,restore,ZIP,real Run/GoTests/15blackbox grade. go.mod фиксирован cli-toolkitGo1.27.2/std-only, files16/total128KiB/Go64/JSON64. Student TestMain notsupported, genuineTestX required. Automatic-pass не завершает quality/Git review и не присваиваетrank.

## Последний checkpoint M6

Files changed: internal/projects+content/projects/p1, store/migration0002, httpapi/project routes+tests, runnerArgs/Assets/test mode, web/ProjectWorkspace, tests/projects.spec.ts, Make/CI/docs.
Architecture decisions: сохраняем source как data, только worker компилирует usercode; trusted grader снаружиcontainer; manual rubrics separate. Весь execution локальный, public bind запрещён. API12execution/min120mutations/min; попытки100recent+исходныеverified rows. No paid resources/API/remote publication.
Commands executed (exact):

```sh
make test
ACADEMY_SANDBOX_TEST=1 .tools/go/bin/go test -count=1 -timeout=10m -v ./internal/runner
python3 scripts/restart-check.py
python3 scripts/crash-check.py
ACADEMY_E2E_RUNNER=1 npm run e2e
ACADEMY_E2E_RUNNER=1 npm run e2e -- --grep 'P1 real'
npm run e2e -- --grep 'project edits'
make -C .data/fresh-source bootstrap
make -C .data/fresh-source test build
.tools/go/bin/go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
npm audit
cd web && npm audit
```

Tests passed (actual evidence): Go unit/vet/race+P1reference/TSbuild; sandbox118.231s12cases; API actualrestart; runneractualSIGKILLrestore; browser10PASS2explicitmobile-runnerduplicates skipped; expandedP1real1PASS1skip15cases+actualGoTests; projectfiles/versions/ZIP/axe2PASS; cleanGitarchive61e418b bootstrap/test/build; govulncheck/npm0vulnerabilities. Latest moduleASCII-whitespace regression+unit+TS/build PASS. Logs and snapshot limits: **docs/VALIDATION_REPORT.md**, docs/validation/*. Independent reviews findings resolved: RUNNER_REVIEW.md, PROJECT_REVIEW.md.
Tests failed/skipped (reasons): исправлены cold compile space, defaultstop grace/cancel cleanup, structure/deadcode grader, contrast/focus, noexec test/cache binaries, module grammar/quotas/TestMain-only; retested. Mobile heavy runtime duplicates skipped, mobile editor/file/snapshot/export проверены. GitHub CI/notLinux/cross-tenant/public security NOT RUN; fullWCAG/perf certification не заявлена.

## Как запустить / повторить

```sh
make bootstrap
make build
# Два процесса:
make runner
make dev
# http://127.0.0.1:8380
```

PinnedGo script uses ignored .tools/go/bin/go если есть, иначе GOTOOLCHAIN=go1.27.2 auto-download через installedGo>=1.21. .data/academy.db и session.secret приватные; migrations auto. README содержит bootstrap/test/export instructions. Не очищать реальную .data/academy.db. E2E использует только .data/e2e/academy.db на8081; fresh archive копия внутри ignored.data/fresh-source, не отдельный remoteclone. Source archived locally in Git, remote absent; никаких публикаций.

## Частично / не готово и риски

- Только Zero Day1 и P1 baseline; полный syllabus/теория шести рангов иP2–P5 не написаны (M10/M11).
- M7 экзамены/Zeroaward и M8 optionalTutor завершены; M9 credentials/passport ещё интегрируются. Нельзя превращать P1:auto или XP в подтверждениегрейда.
- Fullaudit/M12 и release/cross-platform M13 не выполнены; Linux/rootlessPodman проверен, Windows/macOS не проверены.
- Общийkernel/runtime local-only риск; нет microVM/gVisor/publicmultitenant допуска. Teacher/issuer identity/credentials не выданы, compliance не заявлена. README/Git/testquality требуют review.
- License код/контент ещё не выбрана владельцем. CI workflow prepared/notexecuted remotely. Model availability может поменяться; APIaccess не равен подписке.

## Следующий этап

**M9 — local credentials, badges/passport.** Уже написаны stdlibEd25519 core internal/credentials, storemigration0006, HTTPcredentials and private/publicpassport, credentialrenderPDF/PNG/QR (gpdf1.0.13+x/image0.47.0, DejaVuSansTTF+matchinglicense). UI Passport.tsx/CSS и App routes подключены. Ключ issuer.key0600 создаётся при startup; missing/mismatchedkey+existingrecords failsclosed. Keyrestore/backupdocs ещё добавить. Формат oga-local-v1 — не OB/VC. Нужны actualPDF render+QRdecode, nativeexam→issue→download→publicverify→revoke→unpublish browser, finalreview/QA/docscheckpoint. Parent owns HTTP/App/main/tests; agentm7_ui ownscredentialrender; m8_gateway ownsPassportUI; m7_review ownsCREDENTIAL_REVIEW. Не публиковать.

## Финальные уточнения сессии

Defaultport8380 выбран после обнаружения занятого8080 чужим Pythonservice (не изменён). Native API и runner сейчас запущены, livehealth подтвердилavailable=true. Open-in-app browser requestqueued, не заявлять, чтоонфактическиоткрыт. GoMod ASCII+UnicodeTestname/controlledpanic regressions PASS; workerJSONtransport1MiB protectsvalid escapedsource; hostproxy envforwardingfalse+actualsyntheticprobePASS. Все finalmake test/build пройдены. Точные limites/evidence актуализированыVALIDATION_REPORT. Git локальный, безremote; Контрольная точка функционального кода: `7f4cca4`; последующий commit уточняет только документацию. Никакой фоновой разработки/автоматизациинеобещано.

## Продолжение 2026-10-15 — M7/M8

M7 реализован: immutable exam zero-core1.0.0,12вопросов+2чистыефункции/12sandboxcases; prerequisites3упражнения; CASrevision и attemptnonce, atomic claim/grade/award, отказ replay/forgedpass, реальный Zeroaward не зависит отXP. SQLite migrations3/4, inodeDBflock исключает прерывание живойпроверки вторымAPI. Ревью EXAM_REVIEW.md: обаMedium исправлены, открытыхHigh/Medium нет. Go unit/vet/race + TSbuild PASS (`.data/m7-unit.log`). Browser `ACADEMY_E2E_RUNNER=1 npm run e2e -- --grep exam`:3PASS1mobile-runtime-duplicate SKIP, nativepass1.9мин; провал/CAS/reload тоже проверены. `python3 scripts/api-lock-check.py`:PASS. Визуально найдены отсутствующие базовые стили экзамена, добавлены; полный E2E с новым CSS сейчас идёт (`.data/m7-full-e2e.log`). До его результата M7 VALIDATING.

M8 параллельные файлы: internal/tutor/** gateway (m8_gateway); store/ai_usage+migration0005 (m7_store,unit/racePASS); web/src/Tutor.tsx+Tutor.css (m7_ui). Parent интегрирует API/env/settings/consent/budget. Нет настроенной локальной модели/ключа, реальные платные запросы не выполняются; тестовый provider проверяется actualHTTP httptest. Не включать AI по умолчанию.

Автовозобновление существует в Codex heartbeat `open-go-academy-m13` текущегочата; расписание требует актуализации по фактическому limiting reset+1мин, не ежедневно. На07:52местного лимит41%/31%, работа разрешена. Это исправляет предыдущие утверждения об отсутствии механизма. Не обещать выполнение во время выключенного приложения.

## Завершённые M7/M8 и текущая точка

Локальный commit `9e1b794` сохраняет M7/M8код. make test PASS. Nativeexam3PASS1skip; tutor actualHTTPtestdouble3PASS1skip; fullsuite14PASS5skip1staleP1assertionfail -> fixedtest+targetedrealP1rerun1PASS1skip3.1min. Не обозначать исходный fullrun allgreen; новый полныйпрогон нужен после M9. Сохранившийся рангZeroverified после P1 корректен: P1 не меняет прежнийrank. Полныеpaths/logs docs/VALIDATION_REPORT.md и docs/validation/.

Heartbeat `open-go-academy-m13` обновлён реальнымtool: один запуск **2026-10-15 12:26:20 Asia/Atyrau**, сброс5h+1мин. Не ежедневный. Передследующим запуском проверитьобаusagewindows и переноситьэтотжеheartbeat. В текущемтуре работа продолжается; расписание — восстановление после прерывания.

## M9 checkpoint — 2026-10-15 08:16 Asia/Atyrau

**Готово:** credentialscrypto/storage/API/private+publicPassportUI, JSON/PDF/PNG/QR, scopedprivacy/revoke/audit. ActualnativeExam→credentialbrowser3PASS1skip2.0мин (`docs/validation/credential-browser.txt`); Nodecrypto independentlyverifiedJSONsignature, alteredrankrejected; jsQR independentlydecodedQR fromcertificatePNG, PDFpreviewandQRPNG. ActualAPIkeyloss/differentkey/originalrestorePASS (`credential-key-recovery.txt`). make test unit/vet/race/TSbuildPASS; govulncheck0/npmroot+web0. Passportlight/dark/desktop/mobileaxe2PASS4.6s (`passport-layout.txt`). Independent CREDENTIAL_REVIEW closes3Medium; rendererboldDotdoubleprint fixed+visuallyrerendered. APIcanonicalVerifyOrigin fixsupportslocalhostentry whileURLnumeric.

**Частично:** M9 stillINPROGRESS: addexplicitprojectoverviewandrealactivitygraph toPassport, datesforbadges(currentearned_atempty, UIhonestdateunknown). NoHostedissuer/OBVC/JCSclaim; formatoga-local-v1 fixedstructMarshalJSON, Ed25519domain+NUL. PDFitselfnotdigitallysigned; signedJSONauthoritative. Limits localonly. DejaVuTTFmatchinglicensevendored, docs/research/CREDENTIAL_IMPLEMENTATION describesdeps+provenance.

**Следующие конкретные действия:** readcurrentfiles; implement realactivitytimeline fromretainedverifiedevidence and projectoverview/P1manualrubrics inprivatePassport+tests, badgeearnedAt fromactualevidence timestamps; runcombinedfullE2E (previousfull14pass5skip1staleP1assertionfixed,targetP1pass butfullrerunpending), refreshacceptancematrix, markM9DONEonlyaftergate. ThenM10detailedsyllabus+dependencygraph and substantiveZero/Junior/Junior+fullcontent/P2; M11advancedcontent/P3–P5; M12/M13. Neverfakecompletecurriculum.

**Runtime:** API actual`.tools/academy-api`session90782 listening8380, runner`.tools/academy-runner`session9032(older) stillseparate; livehealthok/runnertruechecked. LocaltestHTTPproviderPython8095session42052 remainstestdouble, canstopaftertests (notrealLLM). Don'tkillforeignPython8080. M9keyprobe8094usescopiedDBonlytemporaryfoldercleanup. NativeE2E8081exited; E2EresetDBonlytestdata. UserDB/issuerkeyprivateignoreddonotpublish.
