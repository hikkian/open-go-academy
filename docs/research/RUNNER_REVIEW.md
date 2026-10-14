# Независимое ревью local runner

Дата: 2026-10-15. Область: `internal/runner/runner.go`, `cmd/runner/main.go`, `internal/httpapi/api.go`, `cmd/api/main.go`, `internal/store/store.go`; учебные критерии `internal/content/data/zero-01.json`. Reviewed snapshot — до исправлений главным агентом; ссылки на строки относятся к нему. Код не менялся. Public mode явно отсутствует и не является finding.

## Финальное повторное ревью (2026-10-15)

Последний snapshot прочитан после второго цикла исправлений. **Все конкретные findings R1/R1a, R2/R2a и R3/R3a закрыты на уровне code review текущего local-only профиля; открытых HIGH findings не осталось.** `.tools/go/bin/go test ./internal/httpapi ./internal/runner` выполнен reviewer повторно: PASS (cached).

- R1a: Ready повторно проверяется внутри успешно захваченного slot; stale admission interleaving больше не запускает container. Atomic latch и health503 сохранены.
- R2a: Preflight имеет named result `preflightErr`; deferred cleanup failure устанавливает result error и disabled. Startup прекращается до создания listener.
- R2: scoped reconciliation и stale socket handling присутствуют; отдельные реальные crash/cancel tests главный агент ещё добавляет. Это оставшийся evidence gap, а не утверждение о доказанном дефекте текущего кода.
- R3a: grader принимает только required прямые statements в main, проверяет fmt import и Println; controls/defer/dead branches отвергаются. `TestStructureRejectsDeadBranches` закрывает подтверждённый counterexample. Правило ограничено этими вводными straight-line упражнениями и не является универсальным доказательством runtime semantics.

Sandbox policy в этом цикле не менялась, поэтому 93.33s integration matrix не запускалась повторно. Финальная проверка crash recovery/cancellation/descendant cleanup остаётся за отдельными real integration evidence. Historical sections ниже описывают предыдущие snapshots и не являются открытыми findings.

## Предыдущее повторное ревью после исправлений (2026-10-15)

Исходные findings ниже сохранены как история. Повторно прочитаны runner, startup, grade.go и regression tests. Reviewer выполнил `.tools/go/bin/go test ./internal/runner ./internal/httpapi` и `.tools/go/bin/go test -race ./internal/runner ./internal/httpapi`: PASS (cached). В предоставленном `.data/sandbox-validation.log` real Podman integration завершён PASS за 93.33 s: multi-file, compile error, isolation, output/timeout/memory/disk/PID limits и format. Reviewer не запускал этот integration повторно. Актуальные бюджеты: memory768MiB, tmp384MiB + work128MiB = scratch512MiB, CPU1, PID128, timeout45s; исходные численные примеры findings относятся к прошлому snapshot.

| Finding | После исправлений | Остаток |
|---|---|---|
| R1 | Atomic disabled latch + health503 + sequential regression исправлены | Race между Ready-check и slot acquisition; детали ниже |
| R2 | Scoped installation label (hash socket), startup orphan reconciliation, stale socket detection и deferred preflight cleanup реализованы | Crash/restart и simultaneous-start integration evidence отсутствуют; deferred error не возвращается вызывающему коду |
| R3 | AST checks reject исходные counterexamples; replay test XP10 проходит | Dead-code bypass подтверждён настоящим API handler + fake executor |

### R1a — MEDIUM: Ready проверяется до получения слота

`internal/runner/runner.go:150`–`:165`: B читает Ready=true и приостанавливается перед select; A получает cleanup error, ставит disabled=true и освобождает slot; B затем получает slot и запускает новый container без повторной Ready-check. Atomic устраняет data race, но не эту logical interleaving. **INFERRED schedule; controlled concurrent reproducer NOT RUN**. Минимально повторно проверить Ready после захвата slot; cleanup и release выполняются владельцем единственного slot, поэтому такая проверка блокирует stale admission. Regression с controlled barrier на промежутке admission.

### R2a — LOW: deferred cleanup error не меняет возвращаемый preflight result

`internal/runner/runner.go:101`–`:116`: Ready проверяется перед defer execution. Если проверки прошли, но deferred rm failed, Preflight возвращает nil и устанавливает disabled; main открывает socket и пишет «ready», хотя health503 и Execute denied. **INFERRED**, это incorrect readiness/startup reporting, а не fail-open execution. Минимально named return error или explicit cleanup до return; startup должен завершаться ошибкой и не создавать socket. Нужен fake regression: successful info/ps/run + failing preflight rm → Preflight error.

### R3a — MEDIUM: AST считает недостижимые вызовы

`internal/httpapi/grade.go` инспектирует всё main.Body, включая `if false`. **TESTED** временная Go harness использовала настоящий API Handler, in-memory store, catalog и fake executor (status ok/stdout42/exit0), отправив:

```go
package main
import "fmt"
func answer() int { return 0 }
func main() {
    if false { fmt.Println(answer()) }
    fmt.Println(42)
}
```

Результат HTTP200, `passed:true`, все четыре checks true, persisted XP10. Программа действительно печатает42, но значение answer=0 никогда не вызывается и не выводится. Аналог two-lines: два dead Println и фактический fmt.Print с multiline. Минимально для этих вводных задач проверять форму обязательных прямых top-level statements в main, reject controls/deferred/dead calls в обязательном шаблоне, добавить этот counterexample. Для произвольного кода AST alone не является доказательством runtime semantics; избегать заявления полной проверки компетенции.

Итог повторного ревью: исходные sequential cleanup и простые structural bypass исправлены, но R1a/R3a требуют доработки; R2a улучшает честность startup. Отдельного public execution не требуется для текущего local-only scope.

## Исходный R1 — HIGH: ошибка cleanup не выключает исполнение

Paths: `internal/runner/runner.go:123` (slot освобождается defer), `:164`–`:167` (rm failure возвращает error без permanent state); `cmd/runner/main.go:49` (health всегда ok).

После failed/timed-out `podman rm --force` нельзя подтвердить остановку job. Однако следующий запрос снова получает слот и создаёт контейнер. При сохранении старого workload global concurrency=1 обходится: несколько orphan jobs могут расходовать N×512 MiB и CPU, хотя каждый остаётся в своём cgroup. Сам текст «исполнитель должен быть перезапущен» ничего не блокирует.

**TESTED controlled reproduction:** временный fake `podman` журналировал argv; `info` вернул `true v2`, `run` прочёл stdin и exit 0, `rm` exit 1. Собран настоящий runner `.tools/go/bin/go build -o <temp>/runner ./cmd/runner`, выбран private temporary socket и PATH с fake. Два последовательных HTTP POST `/execute` с `{"mode":"run","files":{"main.go":"package main; func main() {}"}}` оба дали 422 cleanup error. `/health` после первого отказа вернул `ok`; в логе 3 вызова `run` (preflight + оба jobs). Это доказывает повторный admission, а не наличие реального orphan: настоящий ресурсный вред выводится условно из failed cleanup.

Минимально: permanent unhealthy latch после неподтверждённой очистки, health=503 и отказ новых Execute до reconciliation. Regression: fake rm failure → второй Execute не вызывает run; recovery только после подтверждённого удаления.

## Исходный R2 — MEDIUM: restart после crash не сверяет оставшиеся jobs

Paths: `cmd/runner/main.go:26`–`:38`, `internal/runner/runner.go:73`–`:82`, `:144`–`:165`.

Job cleanup находится только в штатном Execute. После SIGKILL/power interruption slot state теряется; startup проверяет preflight, но не проверяет контейнеры с label `academy.runner=true`. Удаление stale socket, требуемое сообщением startup, не убивает контейнер. Podman run CLI, привязанный к socket connection, не является доказанной host process lifetime boundary; оставшийся job может пережить worker и новый запуск нарушает global concurrency.

**INFERRED from code; destructive SIGKILL integration NOT RUN.** Reproduction для отдельного bounded harness: запустить ограниченный long-running job; записать container ID; SIGKILL worker; проверить `podman ps --filter label=academy.runner=true`; удалить stale socket и перезапустить worker. Если старый container running, новый worker не должен стать ready. Аналогичный cleanup отсутствует у cancelled preflight.

Минимально: installation-specific label/ownership, startup reconciliation и ready только после подтверждённой очистки own stale containers; не удалять чужие контейнеры. Negative test crash/restart + preflight timeout + health admission.

## Исходный R3 — MEDIUM: grading не проверяет заявленные критерии

Paths: `internal/httpapi/api.go:218`–`:225`; `internal/content/data/zero-01.json` exercises `repair`, `two-lines`.

`repair` требует отдельную `answer() int` и её вызов; `two-lines` — два отдельных fmt.Println. Grader проверяет только успешный exit и точный stdout/stderr. Программа ниже получает те же три true checks и persisted completion/10 XP, хотя не выполняет ключевой критерий `repair`:

```go
package main
import "fmt"
func main() { fmt.Println(42) }
```

Аналог `fmt.Print("Имя: Ada\nЯзык: Go\n")` обходит два required calls. **INFERRED deterministic code trace; actual API/container reproduction NOT RUN.** Это incorrect learning verification, не credential forgery: ranks/credentials пока не выдаются.

Минимально: доверенные deterministic AST checks для явно структурных критериев и negative tests этих counterexamples; либо согласовать формулировку задания с фактически проверяемым результатом без заявления полной проверки критериев.

## Проверено и открытые evidence gaps

- `.tools/go/bin/go build ... ./cmd/runner`: PASS. Системный `go build` был отклонён из-за Go 1.26.9 vs required 1.27.2; использован закреплённый локальный toolchain.
- `.tools/go/bin/go test ./internal/store ./internal/content`: PASS (store cached). Полный suite/race/container negative matrix этим reviewer не запускались: главный агент добавлял tests одновременно.
- Защитные механизмы по code inspection: numeric loopback bind, exact Host/Origin, cross-site Fetch-Metadata reject, session + CSRF на mutations; strict files без путей/tests; digest-pinned rootless preflight с actual cgroup values и Seccomp=2; network none, RO rootfs, cap-drop, bounded output, concurrency=1; grade вне sandbox; transactional idempotent XP. Не выявлен конкретный Host/Origin/CSRF bypass в данном local scope.
- Нужны реальные negative evidence: descendant pipe retention/cancel, output flood, scratch exhaustion, network/host sentinel, preflight unavailable, orphan recovery. Проверка Seccomp=2 доказывает режим filter, но не конкретную policy effectiveness. Зелёный preflight не заменяет эту матрицу.

Результат ревью: устранить R1 до утверждения fail-closed execution; подтвердить crash cleanup для R2; исправить R3 до заявления, что все criteria lesson проверяются. High/medium findings основаны на конкретном snapshot, после изменения требуется retest.
