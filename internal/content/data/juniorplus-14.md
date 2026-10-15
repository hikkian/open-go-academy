# HTTP и PostgreSQL тесты

## Один тип теста не проверяет всё

Тест (test, автоматическая проверка) — маленькая программа, которая задаёт вход, выполняет код и сравнивает наблюдаемый результат с договорённостью. В HTTP и SQL есть несколько границ, поэтому полезно разделять уровень теста:

| Уровень | Что проверяет | Чего не подтверждает |
|---|---|---|
| Unit test функции | правило или преобразование | HTTP, сеть и PostgreSQL |
| Handler test через `httptest.NewRecorder` | метод, путь, статус, headers, body | TCP listener и реальную БД |
| Handler с fake repository | HTTP и решение о вызове dependency | SQL запрос и ограничения PostgreSQL |
| PostgreSQL integration test | миграции, SQL, constraints, transaction | запуск production окружения целиком |
| End-to-end | компоненты вместе через публичный контракт | все сочетания входов или все production сбои |

Хорошая проверка явно называет, что она доказывает. Если сервис использует fake map, это unit/handler fixture, даже когда тест называется «integration». Внешняя форма теста не повышает доказательную силу внутри.

## Handler test без listener

`httptest.NewRequest` создаёт локальный request, а `httptest.NewRecorder` записывает response в память. Handler вызывается как обычная функция через `ServeHTTP`. Это подходит для status, headers и JSON body. Для JSON декодируйте тело в структуру; whitespace и порядок ключей JSON объекта не должны ломать тест, если контракт их не фиксирует.

Пишите небольшие отдельные тесты для success, отсутствующего ресурса, неправильного method, malformed JSON и oversized body. Таблица случаев (`table-driven test`, тест с таблицей входов) упрощает добавление проверок без копирования одной и той же setup части. В каждом failure печатайте ожидаемое и фактическое значение.

## Fake и PostgreSQL fixture отвечают на разные вопросы

Fake store — контролируемая реализация нужного тесту интерфейса. Он позволяет создать повторяемый сценарий: alias уже занят, lookup выдаёт ошибку или fixture принадлежит другому owner. Fake быстрый и понятный. Он не умеет гарантировать, что уникальный индекс сработает конкурентно, что SQL placeholder действительно параметризирован или что transaction делает rollback.

Integration test запускает PostgreSQL, применяет настоящие migrations на пустую тестовую схему или disposable database, выполняет операции реальным pgx и чистит только собственный fixture. Тест должен отвечать, какой ресурс создал и кому принадлежит. Не указывайте в тесте путь к пользовательской БД и не делайте очистку широким `DELETE FROM links` на общем сервере. Не имитируйте DB драйвер, если вопрос о реальном драйвере.

Изоляция (isolation, независимость тестовых данных) нужна, чтобы два test case не зависели от порядка. Популярные варианты — отдельная БД на тест, schema/table namespace с уникальным ID или rollback transaction для операций, поддерживающих её. Одного `TRUNCATE` недостаточно, если параллельные тесты работают с той же БД.

Если отдельная DB недоступна, честно выполните HTTP suite на fake repository, но обозначьте ограничение. Не перенаправляйте integration URI в личную базу автоматически. Тестовый источник данных может требовать явную метку disposable database и короткий allowlist hostname. Ошибка старта — причина skip с именованным объяснением только там, где suite допускает optional DB; обязательная CI PostgreSQL fixture должна провалить сборку, если не запустилась.

Перед повторным прогоном сопоставьте признаки изоляции: уникальный schema name, owner test, номер миграции и число созданных строк. Если несколько тестов используют `t.Parallel()`, убедитесь, что транзакции независимы и cleanup не удаляет чужие строки. Непараллельный запуск может скрыть гонку, поэтому самый важный конкурентный unique constraint case стоит повторять параллельно.

## Миграции и rollback

Миграция (migration, версия изменения схемы) должна запускаться на пустой базе так же, как используется в проекте. Integration fixture создаёт чистую БД, применяет версии по порядку, выполняет запрос и завершает setup в `t.Cleanup`. Если migration runner использует pool/connection, cleanup обязан закрыть его даже после fatal error.

Транзакционный тест устанавливает наблюдаемый инвариант: если второй шаг изменения fails, первая запись не остаётся одна. Fake может проверить orchestration, но реальный rollback проверяется PostgreSQL, потому что semantics transaction и constraint принадлежат БД. Для уникальности запустите две конкурирующие вставки и убедитесь, что один результат успех, второй — нормализованный conflict, а количество строк равно одному.

## Пример 1. HTTP contract через recorder

Путь `example_test.go`; внешний сервер не открывается.

```go group=handler-recorder file=handler_test.go mode=test
package academy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func health(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/health" { http.NotFound(w,r); return }
	if r.Method != http.MethodGet { w.Header().Set("Allow","GET"); http.Error(w,"method not allowed",405); return }
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func TestHealthResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,"/health",nil)
	health(w,r)
	if w.Code != 200 { t.Fatalf("status=%d",w.Code) }
	if w.Header().Get("Content-Type") != "application/json" { t.Fatal("wrong content type") }
	if w.Body.String() != `{"status":"ok"}` { t.Fatalf("body=%q",w.Body.String()) }
}
```

Тест осознанно фиксирует конкретные пробелы этого примера через строку; production JSON test обычно декодировал бы body и сравнивал поле `status`. Не переносите ненужное текстовое сравнение в договор.

## Пример 2. Таблица method/path/status

```go group=route-table-test file=route_test.go mode=test
package academy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func healthTable(w http.ResponseWriter,r *http.Request) {
	if r.URL.Path != "/health" { http.NotFound(w,r); return }
	if r.Method != "GET" { w.Header().Set("Allow","GET"); http.Error(w,"method not allowed",405); return }
	w.WriteHeader(http.StatusNoContent)
}

func TestHealthRouteCases(t *testing.T) {
	cases := []struct{name,method,path string; want int; allow string}{
		{"ok","GET","/health",204,""},
		{"wrong method","POST","/health",405,"GET"},
		{"unknown path","GET","/other",404,""},
	}
	for _,tc := range cases { t.Run(tc.name,func(t *testing.T) {
		w:=httptest.NewRecorder();healthTable(w,httptest.NewRequest(tc.method,tc.path,nil))
		if w.Code!=tc.want { t.Fatalf("status=%d want=%d",w.Code,tc.want) }
		if got:=w.Header().Get("Allow");got!=tc.allow{t.Fatalf("Allow=%q want=%q",got,tc.allow)}
	}) }
}
```

Имя каждого case объясняет failure. Такой тест можно выполнить в любом порядке: все локальные переменные принадлежат subtest.

## Пример 3. Cleanup локального fixture

`httptest.NewServer` открывает временный локальный listener для проверки HTTP client и server вместе. Он не использует DNS или внешний адрес; `Close` нужен обязательно.

```go group=local-server-test file=client_test.go mode=test
package academy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalHTTPClient(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(204)}))
	t.Cleanup(server.Close)
	response,err:=server.Client().Get(server.URL)
	if err!=nil{t.Fatal(err)}
	defer response.Body.Close()
	_,_=io.Copy(io.Discard,response.Body)
	if response.StatusCode!=204{t.Fatalf("status=%d",response.StatusCode)}
}
```

Здесь нет PostgreSQL и нет реального DNS: этот integration test соединяет только локальный HTTP client с временным handler. БД тестируется отдельной disposable fixture.

## Часы, случайность и стабильный тест

Время и случайность создают недетерминированность. Тест на expiry не должен ждать десять минут: передайте `now` через clock, например интерфейс `Now() time.Time` или функцию. Для случайного ID проверяйте свойства на границе — непустой допустимый размер, уникальность в наборе — и инъецируйте reader/generator там, где это действительно делает тест воспроизводимым. Не ослабляйте реальную генерацию до фиксированного токена в production.

Старайтесь не обращаться к реальному DNS, внешнему HTTP, публичной PostgreSQL или платному service из unit теста. При недоступной disposable БД честно отмечайте integration suite unavailable; fake store не становится эквивалентом.

### Время и порядок не должны быть случайными

Не сравнивайте в тесте `time.Now()` с точностью до наносекунды, если контракт гарантирует только секунды. Внедрите фиксированную отметку времени, сравните диапазон или протестируйте нормализованное значение. Так вы сможете воспроизвести expiry ровно на границе. Аналогично, random ID хранится как случайный результат, но тесту не нужно требовать буквальное значение; проверяйте форму, длину и отличие двух токенов, если это уместно.

Полезное failure сообщение отвечает: какое условие проверялось, что получилось и какое значение ожидалось. Не выводите весь приватный database row или access token ради контекста. Например, `TestUpdateOtherOwner/response` и `TestUpdateOtherOwner/persisted_state` разделяют отказ API и побочный эффект. Именованные subtests позволяют быстро запустить одно доказательство и затем полный набор.

Проверяйте, что test cleanup выполняется и после раннего `t.Fatal`: `t.Cleanup` для этого надёжнее кода в конце тела. Порядок cleanup обратный регистрации, поэтому ресурсно зависимую схему запишите в понятной последовательности: handler stop, repository pool close, database/schema remove. Если ошибка cleanup важна для безопасности, CI не должна молча игнорировать её и объявлять тест успешным.

### Что не нужно смешивать в один тест

Один end-to-end сценарий полезен как acceptance proof, но плохая первая точка для поиска ошибки. Если он провалился, неизвестно, сломан ли router, auth middleware, repository query, migration или DB readiness. Быстрый handler test локализует protocol contract; repository integration test локализует SQL; один сквозной сценарий подтверждает их совместимость. Наиболее эффективная test suite содержит все три масштаба, а не только самые крупные сценарии.

Для PostgreSQL race test дайте каждому конкуренту независимое действие и дождитесь обеих завершений через channel или WaitGroup. Не `Sleep` в надежде попасть в нужный interleaving. Уникальное ограничение БД принимает финальное решение; Service переводит ожидаемый конфликт в стабильный результат. Fake test может проверить нормализацию ошибки, но constraint race остаётся реальной DB responsibility.

## Типичные ошибки и диагностика

- **Один тест проверяет только status.** Успешный status ещё не подтверждает Content-Type, Location и body.
- **Строковое сравнение JSON.** Порядок ключей и whitespace обычно незначимы. Декодируйте JSON.
- **Запуск теста на общей user базе.** Fixture может удалить данные другого участника. Создавайте отдельный disposable database/schema.
- **Утечка `httptest.Server`.** Включайте `t.Cleanup(server.Close)`; response body тоже закрывайте.
- **Объявить fake proof PostgreSQL.** Fake не выполняет SQL, миграции, constraints или настоящий rollback.
- **Параллельные тесты с общим static state.** Убирайте process globals, используйте уникальные IDs или изолированные схемы.
- **Сравнивать текст SQL ошибки.** Нормализуйте результат в repository и тестируйте стабильную категорию; текст может измениться.
- **Тестировать clock через sleep.** Подставьте значение времени напрямую, это быстрее и точнее.

Если тест проходит отдельно, но падает в наборе, ищите общую БД, текущую папку, глобальную регистрацию маршрута, shared map и незакрытый listener. Запустите один case, затем весь пакет с повтором; тест, зависящий от порядка, содержит скрытое состояние.

## Практика

- **Easy:** запишите `httptest` response assertion для status, Content-Type и body. Для JSON декодируйте значение, не сравнивайте случайное форматирование.
- **Medium:** протестируйте CRUD plus redirect через fake store: create меняет fake, GET отображает запись, redirect возвращает ожидаемый Location, неизвестный id — `404`. Назовите, что этот тест не проверяет SQL.
- **Hard:** спроектируйте PostgreSQL fixture с migration с нуля, двумя users, rollback при конфликте и cleanup только созданных данных. Добавьте invalid body и conflict cases. Фикстура должна получать временную тестовую DSN явно; не подключайтесь к пользовательской DB.

## Проверка понимания

1. Что доказывает вызов handler через recorder?
2. Какое свойство fake не может подтвердить?
3. Для чего нужен `t.Cleanup`?
4. Почему migration проверяют на пустой схеме?
5. Как избавиться от `time.Sleep` в expiry test?
6. Как изолировать параллельные PostgreSQL tests?

Перед следующей главой повторите различие unit/handler/integration. В следующем уроке проверка password/session будет развита threat-driven тестами, а не фактом, что какой-либо fake метод вернул `true`.

## Мини-глоссарий

- **unit test** — проверка одного правила или компонента с управляемыми зависимостями.
- **integration test** — проверка связи с реальной зависимостью или внешней границей.
- **fixture** — данные и ресурсы, подготовленные для одного теста.
- **fake repository** — контролируемая тестовая реализация интерфейса хранения.
- **migration** — версионированное изменение схемы БД.
- **rollback** — отмена всех изменений транзакции.
- **isolation (изоляция)** — независимость результата теста от соседних тестов.
- **deterministic test** — тест, который повторяемо выдаёт одинаковый результат.
