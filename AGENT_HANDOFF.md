# AGENT HANDOFF — open-go-academy

Обновлено **2026-10-15 12:20 Asia/Atyrau**. **M0–M9 DONE в local-only области; M10 IN PROGRESS; M11–M13 NOT STARTED.** Полный курс ещё не готов. Не пересоздавать существующую платформу.

## Что действительно работает

- M0 checked official Go1.27.2/ecosystem/frontend/Codex/models/credentials/sandbox, ADR/threatmodel/sourceURLs. Codex0.162.0-alpha.17.2 agents applied: builder/researcher=gpt-6-luna, reviewer=gpt-6.1-sol, максимум3children. Исследования повторно использовать, не считать подписку доступом к productAI API.
- Go net/http+slog, SQLite modernc1.60.1, migrations0001–0006; local profile/theme/session/CSRF/Host/Origin/loopback; drafts/evidence/XP/server-derived ranks and skills, persistence/recovery, quotas and retention.
- React19.3.1/TS7.0.2/Vite8.3.4/CodeMirror6, safeMarkdown, onboarding/dashboard/course/path/skills/lesson/editor/projects/exams/Tutor/Passport, responsive/themes/accessibility.
- Zero Day1 complete,3exercises and trustedrefs,2snippets. P1 CLI Toolkit fullstarter/reference/rubric/files/snapshots/restore/ZIP, nativeRun/GoTests/15blackboxcases. P1auto is not manualqualityreview or rank.
- RootlessPodman runner separateUnix0600 socket, pinnedGoimage, uid65532/networknone/read-only/capdrop/seccomp/NNP; CPU1/memory768MiB/swap0/PID128; tmp128MiBnoexec/work192+cache192exec/shm16; source64KiB/JSONassets64KiB8/output32KiB; perjob45s/cleanup10s/concurrency1. Hostproxyenvforwardingfalse; no unsafe fallback, cleanup failure poisons engine; SIGKILL/orphan restore tested. UserGo never onhost. Trustedexamples/ref may compilehost.
- Exam zero-core1.0.0 immutablecriteria/hash:12objectivequestions+2purefunctiontasks/12nativecases; prereqs hello/repair/two-lines; CASdraftrevision+nonce; atomicgrade+Zeroaward; forgedpass/replay/prereqbypassfail. DBinodeflock prevents secondAPI interruptinglivegrade. Keep currentexamhash stable unless versionbump.
- OptionalTutor: no defaultprovider; OpenAI-compatible stdlibHTTP, mode-specificRussianprompts, modelrouting, max3calls/retry/fallback, requesttimeout/bounds, publicIP/DNSpin/redirect/proxyguard; perrequestconsent+separatecodeconsent, minimalpubliccontext/redaction. PersistentUTCbudget reserves4worstcasecalls, retries/unknownusage/failedrequests chargefullreserve, pendingreserves survivecrash. No prompt/response storage/freeformcache, no modelauthority. RealLLM/paidAPI NOT called; actualHTTPtestdouble tested.
- Credentials oga-local-v1: fixedstructJSONmarshal+domainNUL+Ed25519, pinnedpublickey/fingerprint, privateissuer.key0600; missing/mismatchedkey+existingrecords failclosed. ServerRankAwardguard, privatebydefault, publicationconsent/revoke/audit/status; PDF/PNG/QR, JSONdownload, private/publicPassport, actualbadgeeventdates/evidenceactivitygraph/projectoverview. NotJCS/OB/VC/hostedissuer, PDFitselfunsigned (JSONsigned), localQRonly. VendoredDejaVuSans + matchinglicense, gpdf1.0.13/x-image0.47.0. No fakeidentity/accreditation.

## Проверки и checkpoints

- `make test`: Go unit/vet/race, trustedP1refs, TScheck/build PASS after M9 final graph/project/date changes.
- `ACADEMY_E2E_RUNNER=1 npm run e2e`: **17PASS5explicitSKIP0FAIL6.0min** after theme-assumption correction. Skips mobile heavy native duplicates and explicitHTTPprovider-onlycases. Separate actualHTTPproviderTutor3PASS1skip; nativeexam→issue→downloads→independentNodecrypto→tamperreject→publish→verify→revoke→unpublish/passportprivacy passed.
- `node scripts/credential-artifact-check.cjs`: actualdownloaded PDF Cyrillic text+PDF/PNG QR match signedJSONURL PASS. Poppler renderedPDF and fullPNG visuallyinspected; doubledboldDot fixed/retested.
- `python3 scripts/credential-key-check.py`: actualAPIprocess restart, missingkeyrejection, differentvalidkeyrejection, originalkeyrecovery PASS on copiedE2EDB, no realDBmutation.
- `govulncheck@v1.9.0 ./...` no vulnerabilities, root/web npm audit0. M6 fullsandbox12cases118s+proxy/transportregressions passed; no sandboxcodechangesinM7–M9.
- Independent RUNNER_REVIEW/PROJECT_REVIEW/EXAM_REVIEW/TUTOR_REVIEW/CREDENTIAL_REVIEW findingsclosed in reviewedscopes. Not fullM12certification.
- Exactlogs/limits/failhistory: docs/VALIDATION_REPORT.md, docs/validation/*. Sourcecode localGit no remote: M7/M8 `9e1b794`; M9vertical `c787d0f`; passportactivity/project/date `e091d0e`; subsequentdocs/testfix commit. No publication.

## M10 current task and ownership

Read docs/MASTER_SPEC.md+EXECUTION_PLAN first. New docs/curriculum/ZERO_JUNIOR_PLAN.md details40lessons: Zero8 IDs zero-01..08, Junior12 junior-01..12, Junior+20 juniorplus-01..20. It is a productionplan, not completedcontent. P2 currentprimarysources/versions/digest/security/teststrategy in docs/research/P2_LINKHUB.md (pgx5.11.0/Postgres18.6 checked; no P2implementationyet).

Running agents in this turn:
- m10_grading: ONLY internal/grading/*.go +specs/_demo-add.json. Genericfunctionprivatecases registry; Spec.Load/ValidateSource/Request/Check. Trusteddriver batchesJSONinputcases in one isolatedjob, no expectedvaluesinsidecontainer/API. Public/privateinputs+expected in privateJSONspec, ASTsignature/import/declarationguards. ParentintegratesHTTP/content/tests.
- m10_zero_a: data/zero-02..04.json/md + internal/content/solutions/zero-0{2,3,4}-{easy,medium,hard}/main.go + matchinggrading/specs. Fulloriginalchapters and3tasks each;3complete runnableGoexamples/chapter; validatestrustedrefs. Nozero01changes.
- m10_zero_b: samepaths zero-05..08. Functions/UTF8/tools/diagnostics/slices/mini-project; fullchapters+12tasks. Eachownerusesatomicfilewrites, noenginechanges.

Parent next: add Lesson.order/curriculum_version and Exercise.kind/skills, privategradinglookup; topological/rank ordering instead oflexicaljunior-beforezero; generalizealllessonGoexample/reference validation whilekeepingold3contractregressions. HTTPfunctionRun usespublicdrivercases, Submitprivatecases viaexistingrunner, expectedchecksoutside; Formatnormal. Functionsnohostexecution. Add dynamicmetadata/prereqUI/skills in nextsteps. Reviewsecurity beforegenericgradednativeacceptance. ThenauthorJunior/Junior+allchapters, integrateP2workspace/starter/reference/tests/SQLDockerCompose withsafeisolatedDBtesting(D4), earnedJunior/Junior+assessments. Do not markM10DONE withjustZero, headers, fakebuttons oruncompiledrefs.

## Runtime and reproduction

```sh
make bootstrap
make build
make runner  # separateprocess
make dev     # http://127.0.0.1:8380
make test
ACADEMY_E2E_RUNNER=1 npm run e2e
```

Pinned .tools/go1.27.2 via scripts/go.sh; systemGo1.26.9custom not authoritative. RootlessPodmanimage pinned inrunner/Make. DefaultAPI8380 (foreignPython8080 nottouch); E2E8081 resets ONLY .data/e2e/academy.db, restart8090/keyprobe8094 temporarycopies. `.data/academy.db`, session.secret,issuer.key privateignored; neverdelete/commit/publish. Startuprecover reliesDBlock.

Latest nativeAPI session79797 on8380 and separate runnerprocess stillavailable; inspecthealth/process before restart, don'tassumeallprocessessurvive futureappsession. Test-onlyHTTPprovider8095 stopped. Secrets neverlog. SQLproject newdependency/runtime requires allowlisted offline dependencies/isolatedlocalDB; don'tweaken existingnetworknone/compileuseronhost.

## Limits and continuation

User renewedcontinue. Toolconfirmed **2026-10-15 12:10** actualreset: primary1%,secondary0%, ordinaryUsageAllowedtrue; noresetcreditused/purchase. Newprimaryreset1792066221. Existingheartbeat `open-go-academy-m13` rescheduled via app tool to **2026-10-15 17:11:21 Asia/Atyrau**, one-shot(actual5hreset+1min), notdaily. Whileallowedcontinue now; iflimitlaterchangesinspectbothwindows andupdateSAMEautomation, no duplicates. Mechanismrequiresapp/hostavailable; don'tpromiseworkwhileclosed. DeleteonlyafteractualM13allcriteria.

## Remaining scope / limits

M10fullZero/Junior/Junior+ andP2 notcomplete; M11Middle/Middle+/Senior/P3–P5notstarted. M12fullsecurity/performance/accessibility andM13freshrelease/crossplatform/acceptancematrix remain. Mainlicensecode/contentownerdecisionpending, no remoteCIrun/publicrelease. Localprofile/issuer is notpublicmulti-tenantauth/identityverification. Sharedkernelrisk remains, no microVMcertificate. Preserve honest tested/inferred/not-tested distinctions and updatehandoff aftereachstage.
