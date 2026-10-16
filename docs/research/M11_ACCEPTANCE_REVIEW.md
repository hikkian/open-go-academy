# Независимый review M11 pre-publication gate

Дата: 2026-10-16. Объем: `internal/content/content.go` (выделение `loadLessons`) и `internal/content/m11_acceptance_test.go`; связанные контракты `internal/grading/grading.go`, `internal/runner/runner.go`, `cmd/runner/main.go`, существующие `internal/content/published_test.go` и `content_test.go`. Read-only review реализации: изменен только этот отчет. Файлы разрабатывались параллельно; номера строк относятся к снимку, в котором запуск примеров расположен на строке 78, эталонов — 105.

## Результат финального повторного review

Открытых замечаний по двум целевым файлам после исправлений не осталось. Три первоначальных P2 ниже закрыты по исходникам; runtime fault injection не проводилась. References выполняются одним предварительно проверенным ограниченным Engine. Repository-authored snippets выполняются существующим host helper как доверенные тестовые fixtures; starter только компилируются на host. Этот gate нельзя применять к learner submissions или непроверенным PR как к безопасной песочнице.

## Закрытые замечания первоначального снимка

### P2 / medium: acceptance запускает Engine без обязательной проверки изоляции

**Пути:** `internal/content/m11_acceptance_test.go:78`, `:105`; `internal/runner/runner.go:75`, `:85–123`; для сравнения `cmd/runner/main.go:45–55`.

`runner.New().Execute(...)` не вызывает `Preflight`. Производственный worker до приема заданий проверяет rootless Podman, cgroup v2, фактически примененные memory/swap/PID/CPU ограничения, seccomp, no-new-privileges и версию Go. Новый gate передает аргументы политики контейнеру, но не доказывает, что среда удовлетворяет этим обязательным условиям. Поэтому PASS этого gate не является доказательством проверки в подтвержденной production sandbox. Это не установленный выход из контейнера: граница по-прежнему OCI, исходники не исполняются непосредственно Go-процессом теста.

**Воспроизведение:** read-only trace цепочки `TestM11DraftCatalogAcceptance → runner.New → Execute` показывает отсутствие `Preflight`; `Ready()` нового Engine возвращает true. Безопасный регрессионный тест должен подставлять контролируемый launcher/preflight stub с неподдерживаемым окружением и проверять, что ни один compile/run job не стартует. На реальном неподходящем runtime код запускать не требуется; такого теста в текущем gate нет.

**Минимальная рекомендация:** создать один Engine для всего gate, назначить отдельный уникальный scope теста и выполнить preflight до заданий либо использовать уже проверенный отдельный worker. Не вызывать reconciliation с общим `Scope=default`: `Preflight` удаляет контейнеры того же scope. При недоступной sandbox завершать явно ошибкой, без host fallback.

### P2 / medium: новый Engine на каждое задание отменяет fail-closed после cleanup failure

**Пути:** `internal/content/m11_acceptance_test.go:76–82`, `:104–109`; `internal/runner/runner.go:254–259`.

При невозможности подтвердить `podman rm` Engine устанавливает `disabled=true`. Acceptance записывает `t.Errorf`, продолжает цикл и создает новый Engine с `disabled=false`. Если контейнер не удалился, последующие задания продолжают запускаться: локальный fail-closed протокол production runner утрачен, возможны оставшиеся контейнеры и накопление нагрузки. Параллельных горутин в самом gate нет, но это не предотвращает накопление при неподтвержденной очистке.

**Воспроизведение:** контролируемый launcher stub: первый run завершается, cleanup возвращает ошибку, затем считать последующие run. Ожидается ноль новых run после cleanup failure; текущие два выражения `runner.New()` создают независимые состояния и позволяют следующие вызовы. Fault injection не запускалась; вывод подтвержден исходниками.

**Минимальная рекомендация:** переиспользовать один Engine и немедленно прекращать acceptance при ошибке инфраструктуры/очистки (`t.Fatal`), сохраняя возможность собрать ошибки обычных неверных результатов. Тестировать именно отсутствие последующего запуска.

### P2 / medium: gate не проверяет готовность starter к работе с production driver

**Пути:** `internal/content/m11_acceptance_test.go:88–119`; существующая проверка опубликованного starter — `internal/content/published_test.go`, subtest `starter-compiles`.

Gate вызывает `spec.Request` только с private reference. `loadLessons` требует непустой `StarterCode`, но не проверяет его контракт и компиляцию. Поэтому синтаксически неверный starter или starter с несовместимой сигнатурой/типами может пройти pre-publication gate при правильном эталоне. После публикации пользователь получает неработающий исходный редактор, а обычный опубликованный test уже тогда обнаружит проблему.

**Воспроизведение:** в отдельном fixture/overlay заменить только `StarterCode` одного draft на непустой `package main\nfunc broken(`, сохранив reference/spec. Цепочка текущего gate не читает этот source после проверки непустоты, поэтому результат acceptance от такой замены не изменится. Этот mutation не выполнялся: изменение drafts запрещено объемом review.

**Минимальная рекомендация:** до reference execution строить production request для `exercise.StarterFile: exercise.StarterCode` и проверять compilation в ограниченном runner без требования успешных функциональных ответов. Добавить regression с неверным starter, который не меняет reference.

## Что подтверждено

- Выделение `loadLessons(ids)` сохраняет путь production `Load → Manifest → manifest.Published`, существующие проверки метаданных, references, grader binding, public vectors, prerequisites, циклов и markdown navigation. Drafts не публикуются автоматически.
- Reference gate использует настоящие `Catalog.Grading`, `Catalog.Reference`, `Spec.Request(..., true)` и `Spec.Check(..., true)`. Последний проверяет public+private vectors, status, exit code, пустой stderr и строгий JSON. Fake stdout/public-only прохождения здесь не обнаружено.
- В финальном снимке references идут через OCI runner; snippets выполняются через существующий trusted host helper, starter только компилируются. Политика OCI содержит `--network=none`, `--pull=never`, read-only rootfs, ограниченные tmpfs, CPU/memory/PID и output/time budgets. В этих изменениях нет вызова DB, удаления БД, изменения credentials или запуска project Compose.
- Gate последовательный, без `t.Parallel` и разделяемой mutable state в новых циклах. `-race` самого gate не инструментирует пользовательский бинарник внутри sandbox; это не доказательство отсутствия гонок в учебных решениях.
- Обычный test не стартует новые sandbox jobs этим gate без явного `ACADEMY_ACCEPT_M11_DRAFTS=1`.

## Реально выполненная проверка и ограничения

Команда:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off .tools/go/bin/go test -count=1 -timeout=45s ./internal/content -run '^TestM11DraftCatalogAcceptance$' -v
```

Результат: PASS, сам acceptance **SKIP** по штатной env-проверке, 0.003 s package time. Это подтверждает compilation нового test и отсутствие opt-in запуска; не подтверждает полную приемку 88 drafts. Полный opt-in gate, атакующий код, fault injection, реальные sandbox jobs, cgroup/Podman состояние, общий `go test ./...` и race не запускались в этом review. Default test suite уже содержит host execution доверенных опубликованных примеров; этот существующий путь не менялся и не был предметом отдельного security-аудита. Сосуществование acceptance с активным worker имеет отдельные лимиты Engine; глобальная сериализация всех процессов здесь не проверена.

## Подтверждение исправлений в финальном снимке

- Единственный `runner.New()` расположен перед циклами; уникальный scope содержит PID и UnixNano. `Preflight` обязателен и завершается `t.Fatal` до candidate execution при ошибке. Reconciliation использует отдельный scope и не затрагивает scope активного production worker.
- Все reference requests выполняются тем же Engine. Любая `Execute` error завершает gate; `Ready` дополнительно проверяется после задания. Fresh-engine обхода disabled state больше нет.
- Starter проходит production `Spec.Request` и `trustedCommand(..., "build")`; намеренно неверные функциональные ответы starter не требуются.
- Три fenced Go snippets проверяются `trustedCommand(..., "run")` в последовательных subtests. Helper имеет timeout40s на build и20s на process run; GOPROXY/GOSUMDB отключены. Однако host helper наследует environment, не отключает сеть для программы и не ограничивает output/descendants как sandbox. Это допустимо только в заявленной границе доверенных repository fixtures; считать такие запуски изолированными нельзя. Нового пути learner submissions в него не обнаружено.
- Повторно выполнена та же команда без opt-in env: PASS/SKIP, package0.003s. Ни host candidate programs, ни sandbox references при этом не запускались. Полная M11 приемка остается отдельным необходимым runtime evidence.

## Current-tree acceptance follow-up — 2026-10-16

The separate opt-in full gate subsequently ran on the exact current content tree: `ACADEMY_ACCEPT_M11_DRAFTS=1 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/open-go-academy-m11-current-final-20261016 .tools/go/bin/go test -count=1 -timeout=3h ./internal/content -run '^TestM11DraftCatalogAcceptance$' -v` — **PASS**, 2925.611 s; 530 subtests passed, 0 failed. See `docs/validation/m11-current-tree-acceptance-20261016.txt`.

This closes the automatic current-tree acceptance gap noted above. It does not retroactively make the source-level review exhaustive or remove its documented limitations on human pedagogy, trusted host helpers, public sandbox isolation, P3–P5 artifact evidence, or Senior review.
