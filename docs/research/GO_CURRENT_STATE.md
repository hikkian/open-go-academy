# Go: состояние и рекомендации

Проверено 2026-10-17. Повторная проверка официальной [истории релизов](https://go.dev/doc/devel/release) подтверждает, что последний стабильный patch остаётся `go1.27.2` (выпущен 2026-10-08); официальная [Go 1.27 release note](https://go.dev/doc/go1.27) также идентифицирует 1.27 как актуальный major. Релиз содержит security fixes в `go` command и `crypto/tls`, `html/template`, `net/http`, `net/textproto`, `os`, плюс исправления компилятора, linker, runtime, vet и библиотек. Локально закреплён `go1.27.2`; версия не менялась.

## Поддержка и совместимость

Политика Go поддерживает major-релиз до появления двух более новых major-релизов и выпускает необходимые исправления безопасности в поддерживаемые версии ([release policy](https://go.dev/doc/devel/release), [security](https://go.dev/doc/security/)). На 2026-10-17 это Go 1.26 и 1.27; Go 1.25 уже вне обычного окна поддержки. Source compatibility Go 1 не означает совместимость скомпилированных package binaries между релизами: их нужно пересобирать ([compatibility](https://go.dev/doc/go1compat)).

`go` directive в `go.mod` — минимальная требуемая версия; начиная с Go 1.21 это обязательное требование, а не подсказка. Фиксировать точную версию toolchain в CI отдельно от минимальной языковой версии, сохранять `go.sum`, проверять обновления зависимостей и не полагаться на незакреплённое `@latest` ([module reference](https://go.dev/ref/mod), [toolchains](https://go.dev/doc/toolchain)).

## Что важно в 1.27 для курса

- Язык: добавлены generic methods; generic-методы интерфейсов по-прежнему не поддерживаются. Это материал продвинутого уровня, а не предпосылка для начального backend ([Go 1.27 notes](https://go.dev/doc/go1.27)).
- Compiler/runtime: size-specialized allocation вызовы снижают стоимость некоторых аллокаций менее 80 байт до 30%; ожидаемый суммарный эффект около 1% на allocation-heavy программах, размер бинарника +~60 KB. Фактический выигрыш зависит от нагрузки. Эти цифры не заменяют измерение своего профиля.
- GC/runtime: стандартным стал `goroutineleak` profile (включая `/debug/pprof/goroutineleak`); обнаруживается существенный, но не весь класс утечек — достижимые через globals/активные goroutines блокировки могут не обнаруживаться. Это дополнительная диагностика, не доказательство отсутствия утечек.
- Scheduler/concurrency: release notes не заявляют общего пересмотра scheduler. В `testing/synctest` добавлен `Sleep`; объяснять scheduler по документации runtime и измерениям, не приписывать 1.27 неподтверждённые изменения.
- `net/http`: 1.27 добавил HTTP/2 client-priority handling, `Server.MaxHeaderValueCount`, поддержку TLS ALPN для пользовательских `net.Conn` с `ConnectionState()`, и ограниченный drain незачитанного HTTP/1 body при Close. Новое поведение влияет на connection reuse; читать notes при диагностике редких деградаций. `ServeMux` поддерживает method-aware и wildcard patterns с Go 1.22; начинать курс следует с него и `http.Handler`, затем показывать router как trade-off ([1.27 notes](https://go.dev/doc/go1.27), [net/http 1.27.2 API](https://pkg.go.dev/net/http@go1.27.2)).
- Testing/tooling: `go test` запускает `stdversion` vet check по умолчанию; `httptest.NewTestServer` создаёт in-memory fake network для `testing/synctest`. Обычные unit/integration tests, race detector, fuzzing и профилирование остаются отдельными проверками, каждая ловит свой класс проблем ([notes](https://go.dev/doc/go1.27), [testing API](https://pkg.go.dev/testing@go1.27.2)).
- Modules: Go 1.27 `go mod tidy` приводит разрозненные require-блоки к поддерживаемому двухблочному формату; semver и major suffix `/vN` сохраняют значение. `go` line, lock/checksum и регрессионная проверка остаются обязательны.

## Базовая политика курса

Содержание писать на Go 1.27, но формулировать концепции отдельно от новых возможностей одного выпуска. Для каждого executable example/reference solution хранить версию toolchain и проверять `gofmt`, `go vet`, `go test`; конкурентный код дополнительно проверять с `-race`, а парсеры/обработчики внешнего ввода — fuzz-тестами. `pprof` применять после фиксации benchmark baseline. Для зависимостей выполнять `govulncheck` и пересматривать его результат при обновлении; он не доказывает безопасность приложения и не заменяет threat model. Официальные источники: [Go security best practices](https://go.dev/doc/security/best-practices), [fuzzing](https://go.dev/doc/security/fuzz/), [pprof](https://go.dev/doc/diagnostics#profiling), [govulncheck](https://go.dev/doc/security/vuln/).

Проверка здесь документальная и сетевой: официальный JSON release endpoint, release history, release notes и pkg.go.dev были доступны. Локальную установку Go, сборку кода, тесты, benchmarks и аудит зависимостей проекта в рамках этого исследования не запускал; доступность версии в API сама по себе не подтверждает совместимость будущего кода.
