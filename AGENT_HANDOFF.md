# AGENT HANDOFF — open-go-academy

Обновлено **2026-10-16 01:40 Asia/Atyrau**. **M0–M10 DONE local-only; M11 IN PROGRESS (syllabus/research), код не начат; M12–M13 NOT STARTED.** Полный курс ещё не готов. Не пересоздавать существующую платформу.

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

Read docs/MASTER_SPEC.md+EXECUTION_PLAN first. New docs/curriculum/ZERO_JUNIOR_PLAN.md details40lessons: Zero8 IDs zero-01..08, Junior12 junior-01..12, Junior+20 juniorplus-01..20. It is a productionplan, not completedcontent. P2 currentprimarysources/versions/digest/security/teststrategy in docs/research/P2_LINKHUB.md (pgx5.11.0/Postgres18.6 checked; P2 implemented and tested).

### M10 checkpoint 2026-10-16 01:14 — финальная валидация

**40полныхглав/120упражнений** в manifest2026.10:Zero8,Junior12,Junior+20. Каждыйновыйурок1200–2200слов/Goexamples/3tasks/selfcheck/glossary. AllpublishedGoexamples/multifilegroups/references productiongatePASS `.data/m10-40-promotion.log`. **Все117новыхfunctionrefs actualbasicrunnerPASS**:57Zero/Junior `.data/m10-native-functions-57.log`,36Junior+01..12 `.data/m10-native-http-sql-36.log`,24Junior+13..20 `.data/m10-native-final-24.log`; idempotentXP/persistedevidence/noRankPASS. Исходные3Zero01 проверенынативновfullbrowser. ActualSQLfences09..12PG18.6+P2referenceHTTP/ownership/transactions/MaxConns1PASS `.data/m10-sql-native.log`,6actualHTTP/OpenAPIresponsesPASS.

P2workspace+separateprojectworker allfunctional:48files/nested/snapshots/ZIP, immutablemodsum/export-onlyinfra, isolatedtempPG/nohostports,SDKonlypublicdeps/checkerseparate. Native44checks includesauth401/CSRF/Origin/cookies/tagownerCRUD/SQLfixture+session/freshapprestart/cleanupPASS `.data/p2-worker-native-limited-role.log`; SQLONLYlimitedappRole/search_pathpg_catalog/schemaqualified. ActualSCRAMauditPASS, actualdisconnect/SIGKILLrecoveryPASS. PROJECT_RUNNER_REVIEW noopenHigh/Medium. Sharedkernel/localonly residualrisk; notpublicmulti-tenantcertification.

**Newexams:** internal/assessment All/Find includes immutablezero-core1.0.0(hashbd2cc6db2c954ef3410db69039bcc12cb15085098194686e9883d1e7b70f2d53), junior-core1.0.0, juniorplus-api1.0.0. New20questions+3coding/debuggingtaskseach, privategenericSpecbatch(nohostusercompile). FullcourseexerciseIDs+P1/P2automatic+priorRank required. StoreStartExamvariadicrequiredRanks capturesrank_awards{rank:{exam_id,version,attempt_id}} withVerifiedProgress, rechecksclaim/finishtransaction. HashincludesrequiredRank+Spec.Fingerprint privatevectors/driver/boundtypes/sourcepolicyv2; types.Universepredeclaredguard. PrivateZeroStderrleakfixedwithoutchanginggrade/hash. Mutatingtransitionfixture/wrongroleowner/access/redirectpolicyguardregressionsPASS. ASSESSMENT_CATALOG_REVIEW noopenHigh/Medium.

**Policyexplicit:** Junior/Junior+ are versionedautomaticGoAcademycriteria: allpublishedexercises+priorrank+functionalprojectproof+20quiz+3tasks. They doNOTconfirmmanualprojectquality/professionalgrade. P1/P2manual_review_required remaintrue. Criteria/credential explicitlystate manualqualitynotconfirmed. Seniorhumanreviewrequirementunaffected. PreviousparentphraseaboutfullJuniorrank referredmanualquality; nofakecompletionofmanualreview.

**NativeexamchainPASS64.30s** `.data/m10-native-exam-chain.log`: actual6containergradedtasks Junior→Junior+, atomicawards,replaydeny,completedDraftdeny,signedcredentials. Allcourse/previousZero/prereqproof are EXPLICITtestfixtures only; individualcourse117andP2 separatelyactuallytested. `internal/httpapi/exam_catalog_test.go` testsXP/rankgate,crossExamdraft,wrongnonce409,unknownversion404,changedcriteriahash409,privateoutputZero/genericnoHTTP/Storeleak, credentialformatcompat. Nativegated ACADEMY_EXAM_CATALOG_NATIVE=1.

**Browser/visual:** previousfull22PASS6skip7.2min plusP2native1PASS1skip10.2s andHTTP/SQLreaderbothlight/dark2PASS8s; P2desktop/mobileactuallyviewed. Newfull34cases run`.data/m10-final-full-browser-fixed.log` session37793 withbothnativeflags, STILLRUNNING: checkfinalstatusbeforeM10DONE. Earliernewfullruninterrupted afterduplicateGoEditorregionlabel axe failure; fixedper-taskscrollLabel, rebuiltweb. Lastfullmake `.data/m10-final-technical-complete.log` session73094 (checkexit). PreviousfullmakePASS `.data/m10-all-technical.log`.

Certificate6criteriapreview exposed2-pagePDF, fixedQRmetadata3columnlayout. ActualnewJunior+PDF1page andPNG visuallyinspected, independentjsQRdecodedbothsameURL. Previewfixtures `.data/m10-certificates`, scriptassessment-credential-preview.go. Renderer/APIunitPASS `.data/m10-credential-layout.log`. NoLLMpaidcalls/remote/publication. Tutorlongchapterboundedpublicoverview16KiB/nohiddenrefs tested; lessonOrder/prereqlinks/Markdownanchors/mappedskills+actualprojectevidence2levels available; newexamselector/historyactualIDs.

**Next:** waitfull34browserfixpass+latesttechnical; fixanyfailure beforeM10DONE. `tests/zz-exam-catalog-ui.spec.ts` runsLAST, usesexplicitdisposable`.data/e2e`seedfixture (`scripts/seed-exam-ui.go` guardedenv, noAPIbackdoor), testsnewexamselection/draft/cancel/history/axe/mobile. Do notrunUIseedagainstrealDB. Afterpassupdateplan/handoff/validation, commitM10, restartliveAPIwithnew40catalog+newexams (currentlyliveold32). ThenM11fullMiddle/Middle+/Senior/P3–P5, notDONEfromoutline. Researcher m11_ecosystem_research ownsdocs/research/M11_ECOSYSTEM.mdonly, verifiesmissingRedis/RabbitMQ/gRPC/OTelversionsprimarysources; noM11codeyet. AllM10authors/reviewercompleted. RetainimmutableallcurrentexamcriteriaafterliveDBregistration; newhashchangesrequireversionbump+retainedoldlookup.

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

LiveAPI session97991 on8380 started19:54 with32chapterbinary; basicrunner session33202, projectworker session64869 (latestrebuild). Inspecthealth/processbeforerestart; do notassumesurvival. Test-onlyHTTPprovider8095 stopped. Secrets neverlog. SQLproject newdependency/runtime requires allowlisted offline dependencies/isolatedlocalDB; don'tweaken existingnetworknone/compileuseronhost.

## Limits and continuation

2026-10-16 00:29 actualresetconfirmed primary0%,weekly31%,ordinaryUsageAllowedtrue. Newprimaryreset1792110557(05:29:17local),weekly1792653021. Noresetcredit/purchase. SAMEheartbeat open-go-academy-m13 rescheduledviatoolone-shot **2026-10-16 05:30:17 Asia/Atyrau**. ContinueNOWwhileavailable; adapttoactualbothwindows. App/hostrequired; nofalsebackgroundpromises.

## Remaining scope / limits

M10fullZero/Junior/Junior+ andP2 implemented, finalbrowservalidationpending; M11Middle/Middle+/Senior/P3–P5notstarted. M12fullsecurity/performance/accessibility andM13freshrelease/crossplatform/acceptancematrix remain. Mainlicensecode/contentownerdecisionpending, no remoteCIrun/publicrelease. Localprofile/issuer is notpublicmulti-tenantauth/identityverification. Sharedkernelrisk remains, no microVMcertificate. Preserve honest tested/inferred/not-tested distinctions and updatehandoff aftereachstage.

## Финальное продолжение 2026-10-16 01:14 — M10 DONE

Последняя истина перекрывает interim-предупреждения выше: full34browser **27PASS7SKIP0FAIL7.6min** `.data/m10-complete-full-browser.log`; `make test` **PASS** `.data/m10-complete-technical.log`, подтверждёнexit0. Все117starter_code теперь проверяются permanentproductionRequest+GoBuild, **PASS48.402s**; найденные9missingpackage+6tag/generic/importdefects исправлены. Zero02..04 contradictoryinstructions/32bitoverflowclaims исправлены; Jr+15KDFcap/StrictvsLaxtext+full18cookiematrix+exactKDFbounds исправлены/actualnative3PASS. ScopedCURRICULUM_REVIEW closed, variantreportSTARTER_VARIANTS.

NewexamUI fixture now uses **SEPARATE disposable API/DB inside .data/e2e-exam-ui**, envguard+pathboundary in seed-exam-ui.go. It does notseedmainE2EDB (previousfixturecontaminatedmobileprereqstate), acceptsconfirmdialog, fullypassesbothviewports. P2ZIPtest checksmandatoryREADMEinsteadoptionalComposeafterreferencereplacement. Fullcleanrerunabovepassed. Codecheckpoints9f67b25/8947b0f; finaldocs/statuscommitnext. ManifestM10-complete,lastverified2026-10-16.

Next: M11 fulladvancedcurriculum/P3–P5+realvalidation, thenM12/M13. docs/research/M11_ECOSYSTEM.md currentverifiedversions/go directives/protoc36.2 publishedchecksum; images/downloads/integrationNOTtested. Selectsmallstack: PostgreSQLoutbox+workerpool first, RabbitMQeducationalintegration, Rediscacheoptional, gRPC/OTel pinned. Mustwrite detailedadvanceddependency/daysyllabus and separateprojects/executionADR, notclaimdonefromoutline. NoM11codeyet. Broadcourseplan recommendation: Middle40lessons(weeks9–16), deeperMiddle+/SeniorseparateselfpacedtrackswithP4/P5; exactcounts/depsmustjustifyinplan. StudentGo remainscontainer-only; externalmoduleexamplesneedtrustedlockedfixturemodules andactualrace/svcintegrationevidence. Senior requiresseparatehumanreviewworkflow, automaticJuniorpolicy doesnotrelaxthat.

Beforeendingcheckactualusage(01:07primary92weekly46allowedtrue); heartbeatSAMEone-shot05:30:17local. Preserveproviderlimits, don'tbuyreset. Currentuseraskedcontinue; workwhileallowed, savecleancheckpointswhenexhausted. No backgroundpromisebeyondworkingheartbeat.

### Состояние перед сбросом лимита — 2026-10-16 01:15

M10 final acceptance commit **193e32b**, предыдущие9f67b25/8947b0f. Actualusage primary99%,weekly47%,ordinaryUsageAllowedtrue; следующийreset05:29:17local, SAMEheartbeat05:30:17ACTIVE. БольшойM11код приостатке1%не запускался; primaryresearchготово. LiveAPIпересобран и реально перезапущен с40главами/новымиэкзаменами/исправленнымистартерами, session58638, healthrunner_availabletrue. ТеперьJunior/Junior+1.0.0зарегистрированы вреальнойлокальнойБД: **не меняйтеихhash/criteriapayloadбезversionbumpссохранениемстарыхверсий**. РеальнаяБДнеочищалась, пользовательскиедостижениянеподменялисьтестовымиfixtures. Gitчистыйпослеэтогодокcheckpoint; продолжатьM11послеactualusagecheck, не повторятьM10native117илиfullE2Eбезновыхпричин.

### M11 старт — 2026-10-16 01:40

Появилась отдельная [панель прогресса](docs/PROJECT_PROGRESS.html), источник [JSON](docs/project-progress.json), генератор `python3 scripts/render-project-progress.py`. На 2026-10-16 закрыто11/14milestone gates (79% gate-count; явно не трудоёмкость); курс40/128уроков, проекты2/5, экзаменные ранги3/6, M12/M13 0/2. Плановыйсрок полного завершения6–27ноября2026 при регулярной доступности лимитов; confidence low, дата не гарантируется. Пересчитывать после каждого крупного gate.

M11 detailed syllabus [MIDDLE_SENIOR_PLAN.md](docs/curriculum/MIDDLE_SENIOR_PLAN.md): Middle40 lessons weeks9–16; Middle+24/Senior24; P3/P4/P5 criteria, times, prerequisites. Ровно88 запланированных advancedlessons — не опубликовано. Ecosystem primaryversion report [M11_ECOSYSTEM.md](docs/research/M11_ECOSYSTEM.md); Redis/RabbitMQ/gRPC/OTel modules/cross-version compatibility requireimplementation-time verification, no images pulled/no modules installed.
