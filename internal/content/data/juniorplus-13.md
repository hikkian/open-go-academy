# Слои приложения и зависимости

## Зачем разделять HTTP и правило продукта

Когда короткая ссылка создаётся через HTTP, один запрос проходит несколько разных вопросов. Входная граница понимает метод и JSON; доменная логика решает, допустимы ли alias и target; хранилище записывает и читает строки. Если всё это находится в одном большом handler, сложнее проверить правило без HTTP и сложнее заменить PostgreSQL на fake во время теста.

Слой (layer) — группа кода с ясной ответственностью. Для небольшого Go backend достаточно трёх логических частей:

```text
HTTP handler → service/domain → repository → PostgreSQL
     ↑              ↑                ↑
 request/response   правила          SQL и строки
```

Handler переводит протокол в application input и результат обратно в status, headers и JSON. Service (служба приложения) координирует use case: проверяет доменное правило, вызывает repository и сопоставляет стабильные ошибки. Repository (репозиторий) знает конкретный способ хранения и преобразует rows/errors. Это направление зависимости важно: application правила не должны импортировать пакет PostgreSQL или `net/http` только ради хранения.

Маленький проект не нуждается в десятках папок и интерфейсов. Пакеты следует разделять, когда у них появляется самостоятельная ответственность, а не потому, что схема архитектуры обещает по пакету на каждое имя. Первые несколько небольших типов можно держать в `internal/links`; слой обозначить кодом, а не глубокой папочной иерархией.

## Constructor injection — передать зависимость при создании

Dependency injection (внедрение зависимости) означает, что объект получает то, от чего зависит, извне. Вместо глобального `var database *Pool` создайте Service с repository:

```text
type Service struct { links LinkRepository }
func NewService(links LinkRepository) *Service { return &Service{links: links} }
```

HTTP слой создаёт production repository и передаёт его конструктору; тест передаёт fake. У constructor injection несколько практических эффектов: состояние запроса не спрятано в глобальной переменной; одновременно могут жить две независимые конфигурации; тесту не нужно подключать пользовательскую базу; по сигнатуре видно, без чего Service не может работать.

Интерфейс задаётся по потребности вызывающего слоя. Если сервису нужно `FindBySlug`, не принимайте весь большой `*pgxpool.Pool` и не передавайте десяток SQL методов. Маленький интерфейс показывает используемое обещание. Go реализует интерфейсы неявно: repository удовлетворяет типу, если имеет нужные методы; не нужен специальный `implements` блок.

Интерфейс имеет стоимость: он добавляет абстракцию, ещё одно место для изменения и иногда заставляет fake имитировать детали реализации. Начните с concreteness. Вводите интерфейс у границы, когда сервис действительно нужно тестировать отдельно или существует реальная альтернативная реализация. Не делайте интерфейс из единственного метода, если он не помогает вызвать поведение или изолировать ресурс.

### Единый владелец конфигурации

Config (конфигурация) — проверенные параметры среды, например адрес базы, время жизни сессии и лимит body. Считайте их один раз при старте, проверьте ограничения и передайте нужную часть конструктору. Если лимит отрицательный или тайм-аут равен нулю вопреки требованиям, лучше остановиться до начала приёма запросов, чем позволить зависимости вести себя случайно.

Не копируйте secret в каждый слой. Repository получает DSN через pool в конструкторе, но не должен возвращать значение DSN в ошибки API. Печатайте безопасную категорию конфигурационной ошибки; внутреннее объяснение остаётся в защищённой диагностике запуска без секретного значения.

Constructor дерева приложения может владеть созданием pool, проверкой конфигурации и закрытием pool при graceful shutdown. Handler не должен незаметно создавать новый pool на каждый запрос: соединение — управляемый долгоживущий ресурс. На старте явно проверьте ограниченное соединение, затем передайте repository и service в router. Это делает процесс тестируемым и выделяет точку, где ресурс создаётся один раз и закрывается один раз.

## Пример 1. Service использует узкий repository

Путь `main.go`. Реализация и fake находятся в одном маленьком учебном файле, чтобы показать поток запроса без базы или listener.

```go group=service-injection mode=run
package main

import (
	"context"
	"fmt"
)

type LinkRepository interface {
	Exists(context.Context, string) (bool, error)
}

type Service struct { links LinkRepository }
func NewService(links LinkRepository) *Service { return &Service{links: links} }
func (s *Service) CanCreate(ctx context.Context, alias string) (bool, error) {
	if alias == "" { return false, nil }
	exists, err := s.links.Exists(ctx, alias)
	if err != nil { return false, err }
	return !exists, nil
}

type memoryLinks map[string]bool
func (m memoryLinks) Exists(_ context.Context, alias string) (bool, error) { return m[alias], nil }

func main() {
	service := NewService(memoryLinks{"taken": true})
	free, _ := service.CanCreate(context.Background(), "new")
	taken, _ := service.CanCreate(context.Background(), "taken")
	fmt.Println(free, taken)
}
```

Service не знает, что репозиторий — map. Тот же интерфейс может реализовать pgx repository. Тестовый вывод `true false` проверяет только правило доступности alias на локальном fake.

## Пример 2. Тест подставляет fake с поведением

Тест явно выбирает подстановку и передаёт `context.Context`, как сделал бы запросный слой.

```go group=service-test file=service.go mode=test
package main

import "context"

type Lookup interface { Exists(context.Context, string) (bool, error) }
type Service struct { lookup Lookup }
func NewService(lookup Lookup) Service { return Service{lookup: lookup} }
func (s Service) Available(ctx context.Context, slug string) (bool, error) {
	if slug == "" { return false, nil }
	exists, err := s.lookup.Exists(ctx, slug)
	return !exists && err == nil, err
}

type fixedLookup struct { exists bool; err error }
func (f fixedLookup) Exists(context.Context, string) (bool, error) { return f.exists, f.err }
```

```go group=service-test file=service_test.go mode=test
package main

import (
	"context"
	"errors"
	"testing"
)

func TestAvailableUsesRepositoryOutcome(t *testing.T) {
	for _, tc := range []struct { name string; repo fixedLookup; slug string; want bool; wantErr bool }{
		{"free", fixedLookup{}, "go", true, false},
		{"taken", fixedLookup{exists:true}, "go", false, false},
		{"storage error", fixedLookup{err:errors.New("offline")}, "go", false, true},
		{"empty slug", fixedLookup{exists:true}, "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewService(tc.repo).Available(context.Background(), tc.slug)
			if got != tc.want || (err != nil) != tc.wantErr { t.Fatalf("got (%v, %v)", got, err) }
		})
	}
}
```

Тест не знает SQL и не подтверждает Postgres constraint; он доказывает применение результата зависимости внутри Service. Другой integration test должен отдельно проверить реальное ограничение уникальности.

## Пример 3. Отмена проходит через Service

`context.Context` переносит deadline и отмену. Не заменяйте request context новым фоновым context внутри слоя.

```go group=context-chain mode=run
package main

import (
	"context"
	"fmt"
	"time"
)

type Reader struct{}
func (Reader) Find(ctx context.Context) error {
	select {
	case <-ctx.Done(): return ctx.Err()
	case <-time.After(time.Millisecond): return nil
	}
}

type Service struct { reader Reader }
func (s Service) Load(ctx context.Context) string {
	if err := s.reader.Find(ctx); err != nil { return "cancelled" }
	return "loaded"
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fmt.Println((Service{reader:Reader{}}).Load(ctx))
}
```

В настоящем pgx вызов принимает `ctx`, и deadline от клиента сокращает ожидание запросов. Нельзя игнорировать ошибку и продолжить бизнес-изменение как будто чтение успешно.

## Архитектурные границы без догмы

Data transfer object (DTO, объект передачи данных) может представлять вход или выход HTTP. Он не обязан быть доменной моделью. JSON field `target_url` не должен незаметно определять column name, SQL table и внутреннее правило. Преобразуйте вход в нужный доменный тип на границе, если это упрощает смысл и проверку.

Не отдавайте `pgx.ErrNoRows`, `*pgconn.PgError` или внутреннюю ошибку middleware прямо в UI. Repository распознаёт инфраструктурную категорию, Service сопоставляет стабильное доменное имя, Handler устанавливает HTTP status и безопасный body. Текст ошибки SQL зависит от версии и окружения, поэтому не делайте `strings.Contains(err.Error(), "duplicate")` архитектурным контрактом.

Одна небольшая монолитная Go программа часто удобнее ранних микросервисов. Все слои собраны в одном модуле и одной deployable unit; запрос вызывает функцию, не перепрыгивая в другой сетевой сервис. Такую структуру называют modular monolith (модульный монолит): логические границы существуют, но HTTP сервис остаётся цельным процессом. Разделение в отдельный сервис потребовало бы сетевых отказов, версионирования контрактов, трассировки и дополнительных deployment решений, которые пока не решают текущую потребность.

## Типичные ошибки и диагностика

- **Глобальная база.** Тесты начинают влиять друг на друга, а конфигурацию нельзя проверить отдельно. Передайте pool через constructor.
- **Зависимость от net/http в доменном правиле.** Сервис начинает говорить кодами статуса вместо результата продукта. Пусть handler переводит результат в HTTP.
- **Интерфейс на каждую структуру.** Спроси, какая реальная реализация или test seam нужна. Если никто не использует интерфейс, он может быть лишним.
- **SQL внутри handler.** Сопоставление протокола и чтение rows смешаны. Вынесите repository operation, если handler перестаёт быть понятным.
- **Обернуть всякую ошибку в строку.** Клиент не может понять `not found` против `conflict`, а тест зависим от текста. Используйте стабильные категории.
- **Потерять context.** `context.Background()` в середине запроса игнорирует отмену клиента и deadline.
- **Лишний слой, который просто переименовывает вызов.** Абстракция полезна, если она задаёт границу или логику, а не добавляет переход для каждой функции.

Для расследования пройдите один `POST /links` по цепочке и назовите данные на каждом переходе: Request DTO, валидированные alias/target, Service command, repository parameters, persisted link, response DTO. Затем найдите, кто переводит storage conflict в `409`. Если ответа два или ни одного, скорее всего контракт размыт.

## Практика

- **Easy:** извлеките проверку alias/target из HTTP callback в функцию доменного слоя. У неё не должно быть аргумента `ResponseWriter`, а результат — стабильная категория или отсутствие ошибки.
- **Medium:** опишите узкий `SlugRepository.Exists(ctx, slug)` и передайте fake в Service через constructor. Проверьте свободный slug, повтор и error repository; помните, уникальный индекс в БД всё равно нужен против гонки.
- **Hard:** передайте запросный context через handler, Service и repository callback. Отмена должна дать категорию `cancelled`; ошибка хранения — `repository_error`; успех — `ok`. Не заменяйте входной context `Background()`.

## Проверка понимания

1. За что отвечает handler и за что Service?
2. Какой кодовый слой должен знать SQL placeholder?
3. Что означает constructor injection?
4. Когда маленький интерфейс оправдан?
5. Почему fake не проверяет уникальное ограничение реальной БД?
6. Как context cancellation доходит от запроса к repository?

Перед следующей главой вспомните `httptest` из juniorplus-05: он проверяет HTTP границу, а fake отдельно подменяет storage. В главе 14 эти два тестовых уровня будут разнесены, чтобы не выдавать успешный unit test за проверку миграции.

## Мини-глоссарий

- **handler (обработчик)** — адаптер между HTTP и внутренним use case.
- **service (служба приложения)** — координирует сценарий и доменные решения.
- **repository (репозиторий)** — изолирует чтение и запись в хранилище.
- **dependency injection (внедрение зависимости)** — передача зависимости снаружи при создании компонента.
- **DTO** — тип для транспортной формы входных/выходных данных.
- **modular monolith (модульный монолит)** — один deployable процесс с явными внутренними границами.
- **context cancellation (отмена контекста)** — сигнал прекратить работу после deadline или отмены.
- **test seam (граница тестовой подстановки)** — место, где реализацию можно заменить управляемым fake.
