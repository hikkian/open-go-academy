# Независимое ревью M6: project workspace и runner extensions

Дата: 2026-10-15. Область: `internal/httpapi/projects.go`, `internal/projects/projects.go`, `internal/runner/runner.go`, project методы `internal/store/store.go`, bundled P1 criteria. Режим local single-user; открытых HIGH/exploitable command-injection findings не обнаружено. Implementation reviewer не менял. Ниже findings исходного M6 snapshot, проверка исправлений дописывается отдельно.

## Финальное заключение M6 (2026-10-15)

Текущий snapshot повторно inspected: validGoMod нормализует CRLF, отвергает non-ASCII и control chars кроме tab/LF, затем проверяет отдельные module/go directives. `TestModuleSyntaxRejectsNonGoWhitespace` покрывает vertical-tab/form-feed/NBSP/lone-CR; reviewer `.tools/go/bin/go test ./internal/projects ./internal/httpapi`: PASS (cached). **P1, P2/P2a/P2b и P3 полностью resolved по текущему коду и regression evidence; открытых HIGH/MEDIUM findings этого ревью не осталось.**

Ограничения остаются явными: качество student tests требует manual review, `manual_review_required=true`, automatic project XP не rank/credential. Local-only namespaces/rootless containers разделяют host kernel; public execution disabled. Parent предоставил PASS свежей archive-bootstrap/test/build и реальных15vectors+Go tests; reviewer эти длительные checks повторно не запускал. Sections ниже — история discoveries/проверок, а не открытые findings.

## Предыдущее повторное ревью исправлений (2026-10-15)

Последний M6 snapshot: single-line module counterexample P2a отвергается, Go/JSON quotas и genuine Test gate сохранены. Decoder project1MiB покрывает escaped JSON при source cap128KiB; общий submit deadline180s, API WriteTimeout200s. Scratch policy tmp128MiB(noexec)+work192MiB(exec)+cache192MiB(exec), shm16MiB, memory768MiB, CPU1/PID128, per-job45s. Exec cache нужен для Go1.27 cached go run binaries; assets/runtime API не получает host mounts. Parent сообщил full sandbox PASS118.231s, real15-vector project E2E1pass/1mobileSkip и workspace/export/axe2pass; это parent-provided evidence, reviewer повторно запускал только unit suite (PASS cached).

**P2b narrow remaining variant:** per-line `strings.Fields` всё ещё принимает whitespace, не допустимую Go module lexer. `module\vcli-toolkit\n\ngo 1.27.2\n` даёт те же ожидаемые tokens у validator. Reviewer выполнил настоящий Go build temporary exported package: exit1, `go.mod:1:7: unexpected input character '\v'`. Минимально reject non-ASCII/control whitespace кроме ordinary space/tab/newline после CRLF normalization, либо нормативный parser. Поэтому P2 пока нельзя пометить полностью resolved. P1/P3 resolved; открытых HIGH нет.

Current snapshot прочитан. `.tools/go/bin/go test ./internal/projects ./internal/httpapi ./internal/runner ./internal/store`: PASS (projects freshly executed, остальные cached).

- **P1 resolved:** workspace Go64KiB/JSON64KiB/JSONcount8/total128KiB/root-files16 quotas совпадают с runner subset limits. Project encoded body limit384KiB проверен в shared decoder. Required source/README/module gates и regressions присутствуют.
- **P3 resolved:** TestMain отвергается во всех student test files; genuine standalone TestX(*testing.T) signature, импорт testing с alias, отсутствие returns и receivers проверены. Regression `TestProjectHasTestsRejectsOverridesAndHelpers` проходит. UI check теперь честно сообщает только successful go test и необходимость review качества; manual_review_required остаётся true.
- **P2 PARTIAL / still MEDIUM:** missing/extra module directives и отсутствующий README закрыты, но flattened token comparison принимает синтаксически ошибочный go.mod. См. P2a ниже. Real project/browser matrix главный агент ещё выполняет, reviewer не утверждает её PASS.

### P2a — MEDIUM: strings.Fields стирает значимые границы строк go.mod

Current `projects.ValidateFiles` сравнивает `strings.Join(strings.Fields(contents), " ")` с таким же normalized required module. Поэтому single-line `module cli-toolkit go 1.27.2` принимается как правильный exported module, хотя Go parser требует отдельную directive line.

**TESTED temporary harness:** настоящий `projects.ValidateFiles` с main.go/README и указанным go.mod вернул nil. Затем `.tools/go/bin/go build .` в temporary экспортируемом package с таким модулем дал exit1: `go.mod:1: usage: module module/path`. Значит исходная гарантия воспроизводимости ZIP остаётся нарушена в narrower input variant.

Минимально сравнить exact trusted module после CRLF normalization/trim окружающей whitespace, сохранив границы directive lines, либо разобрать go.mod нормативным parser и проверить allowed directives. Regression flattened same-tokens module must reject; accepted exported module должен проходить настоящий Go parse/build.

## Исходный P1 — MEDIUM: project validator принимает файлы, которые runner отвергает

Paths: `internal/projects/projects.go` ValidateFiles; `internal/runner/runner.go` validateFiles/Assets validation; projectJob и run/test routes.

Workspace разрешал суммарно128KiB и16 файлов без отдельных Go/JSON quotas. Runner принимает Go64KiB, assets64KiB и максимум8 JSON. Поэтому сохранение/ZIP успешны, а запуск корректной по workspace budget программы систематически503. **TESTED** temporary Go harness импортировала настоящие packages: `projects.ValidateFiles` для main.go с70KiB comment вернул nil; `runner.New().Execute` тех же files вернул «исходники превышают64КиБ» до запуска runtime. Девять маленьких JSON — аналогичный counterexample по count (code traced).

Минимально: единые Go/assets/count budgets в workspace и runner, валидировать requests до SaveProjectFiles/execute; encoded JSON body budget учитывать отдельно от decoded source bytes. Regression boundary64KiB±1, JSON64KiB±1, assets8/9; export/save accepted runnable file sets имеют одинаковые quotas.

## Исходный P2 — MEDIUM: automatic release проходит с повреждённым exported go.mod

Paths: projects ValidateFiles; `internal/httpapi/projects.go` projectJob/export/submit; `internal/runner/runner.go` trusted module tar header; P1 spec release automatic criteria.

Execution правильно игнорирует недоверенный go.mod и генерирует фиксированный trusted module. Однако workspace сохранял произвольный go.mod, ZIP экспортировал его, а automatic release criteria заявляют воспроизводимые build/run/test команды. Валидатор не требовал README/module вообще. Поэтому automatic_pass не гарантировал, что пользователь может собрать скачанный ZIP.

**TESTED API wiring:** настоящий Handler/catalog/in-memory store + fake executor, возвращавший корректные trusted functional-vector results, приняли `go.mod: "this is invalid"`, без README, ответ HTTP200 `automatic_pass`, all7 checks true, manual_review_required=true, XP10. Это доказывает отсутствие проверки manifest/export, а не прохождение ошибочного source реальным compiler. Runtime fixed module подтверждён code inspection; реальная сборка ZIP с таким go.mod будет parse error.

Минимально: require/validate fixed stdlib-only exported go.mod и README либо нормализовать trusted module в export; regression invalid/missing module/readme + ZIP Go build. Сохранять manual-review gate; automatic XP не rank.

## Исходный P3 — MEDIUM: TestMain-only маскирует отсутствие Go-тестов

Path: `internal/httpapi/projects.go` hasTests, action test и final submit check.

hasTests принимает любую FuncDecl с prefix Test и одним параметром. Поэтому `func TestMain(m *testing.M){os.Exit(0)}` удовлетворяет gate, хотя Test-функций нет и test runner ничего не проверяет. После trusted functional vectors платформа сообщает «Есть и проходят пользовательские Go-тесты» true. Это ложная проверка tests-presence, а не обход функциональных vectors или manual review.

**TESTED:** настоящий API Handler + fake vector executor и этот sole `_test.go` дали HTTP200, все7 checks true, automatic_pass, XP10. Отдельно `.tools/go/bin/go test ./<temporary-package>` с обычным main.go и только TestMain ниже завершился exit0 за0.002s:

```go
package main
import ("testing"; "os")
func TestMain(m *testing.M) { os.Exit(0) }
```

Минимально: genuine standalone TestX(*testing.T) signature (не method, не TestMain, без return values) и explicit beginner policy против student TestMain, который может пропускать остальные tests. Test gate подтверждает присутствие/исполнение, не качество coverage; последнее не заявлять проверенным только из exit0. Regression TestMain-only/method/wrong-signature/no-op tests; functional vector grade остаётся независимым.

## Проверенные положительные границы

- Args не интерполируются в shell text: фиксированный `sh -c` script, отдельный `$0=--`, argv через `"$@"`; shell metacharacters остаются program arguments. Args count32/bytes4096/NUL reject присутствуют.
- JSON assets root-name allowlist, count8/bytes64KiB, tar regular headers; source не задаёт paths/symlinks/module/runtime options. Generated module trusted, stdlib-only network-off policy сохранена.
- ZIP имена проходят workspace root-name validator; archive filenames не содержат traversal/absolute paths. Store snapshot lookup включает одновременно project_id и id; replacement + snapshots/prune10 в transaction, bind parameters вместо SQL interpolation.
- Rate limiter включает project run/test/submit; engine concurrency1 и latch внутри slot сохранились. Submit общий context100s; Server WriteTimeout120s. Слот per container job, не вся submission: конкурирующий запуск может получить busy; лимит исполнения при этом остаётся1.
- Trusted functional vectors исполняются без student `_test.go`; student test mode никогда не единственный grade. Result всегда manual_review_required=true; RecordAttempt p1:automatic даёт idempotent XP и не присваивает rank.
- `.tools/go/bin/go test ./internal/projects ./internal/httpapi ./internal/runner ./internal/store`: PASS (cached). Existing real M3 isolation matrix здесь повторно не запускалась. Parent добавляет real project Go/browser tests; их результаты не представлены как reviewer-tested.

## Статус исправлений

Главный агент сообщил, что отдельный project worker исправляет P1/P2: согласованные Go64KiB/JSON64KiB/assets8/root16 quotas, обязательные README/main Go и фиксированный exported go.mod, достаточный encoded JSON body limit. **На момент исходного отчёта исправления ещё не inspected/retested.** P3 передан главному агенту для correction. После изменений требуется узкий rereview current snapshot.
