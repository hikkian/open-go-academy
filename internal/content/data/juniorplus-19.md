## Свести протокол, состояние и доверие

LinkHub — проект коротких ссылок с пользователями, ссылками, тегами, PostgreSQL, миграциями и HTTP handlers. Цель главы — не добавить неописанный endpoint, а проследить существующую функцию через contract, handler, authentication, owner scope, repository, SQL, response, logs и test evidence. Учебные требования собраны в P2 starter README; contract находится в OpenAPI; rubric разделяет automatic checks и ручное ревью; threat model описывает assets и честные ограничения. Сначала реализуйте по контракту, затем изучайте reference, чтобы сравнить выбор дизайна.

Browser отправляет method, path, JSON body, cookie и иногда CSRF header. Сервер проверяет размер, формат и domain rules. Valid session превращается во внутреннего authenticated principal. Owner ID нельзя принимать из request как полномочие: он приходит из этой server-side identity. Service формирует предметную операцию; repository связывает её с параметризованными аргументами; PostgreSQL constraints защищают инвариант при конкурирующих запросах.

## Полный сценарий short link

`POST /api/v1/links` принимает обязательный target URL, optional slug, флаг enabled и список tags. Schema ограничивает длину и неизвестные поля. URL политика разрешает HTTP(S), запрещает userinfo и control characters. Запись ссылок и association tags принадлежит одной транзакции. Slug uniqueness подтверждается constraint даже если приложение сначала пытается удобную проверку: два конкурентных запроса могут одновременно прочитать «свободно».

Ссылка должна принадлежать session owner. То же условие действует в list, read, patch и delete; route ID сам по себе не даёт право. Для чужой записи приложение отвечает общей not-found формой, чтобы не подтверждать существование объекта. Изменение, сохранение ссылки и связей с tag должно либо целиком commit, либо откатиться, если foreign-owner tag или другой insert отклонён.

Публичный GET `/{slug}` перенаправляет только на сохранённый, включённый и валидированный target. Сервер возвращает redirect Location, но не делает fetch этой destination. Если сервер запрашивает произвольный URL, злоумышленник может попытаться обратиться к localhost, cloud metadata или служебной сети. Для API неизменная server-side policy важнее копирования target в автоматический http.Client.

## Аутентификация и CSRF на HTTP boundary

Authentication отвечает на «кто предъявил действующую сессию?» Authorization — «разрешено ли этому пользователю действие над этой записью?». Browser cookie является ambient credential: браузер прикладывает его автоматически в некоторых contexts. Поэтому state-changing request проверяет CSRF token в header и настроенный Origin. SameSite уменьшает класс cross-site запросов, но не заменяет полный серверный контроль.

Референс хранит opaque session и CSRF hash, устанавливает HttpOnly session cookie и читаемый отдельный CSRF cookie, ограничивает возраст и удаляет сессию при logout. Точные flags и порядок проверок ищите в auth/security коде и threat model. Не переносите локальное учебное решение на настоящие пользовательские данные. В P2 нет email verification, recovery, MFA или распределённого rate limiting.

В каждом мутирующем endpoint проверяйте отсутствие сессии, неправильный Origin/CSRF, нормальный владелец, чужой resource ID и поведение logout. Не доверяйте role из request. Для non-owner поведений не пишите разные подробные ошибки, если это раскроет существование private record.

## Интеграция SQL и откат

Fake-store HTTP тест подтверждает обработку ответа handler и помогает быстро проверять ошибку. Он не подтверждает PostgreSQL foreign key, индекс, миграцию, contention или транзакцию. Настоящий disposable DB тест применяет миграции, создаёт две независимые учётные записи и проверяет observed response плюс состояние таблиц. Каждая integration fixture обязана убирать только своё состояние и не трогать пользовательскую базу.

Для tag association проверьте оба направления: одинаковый owner фиксирует новую связь; другой owner её отклоняет и не оставляет полусозданную link. Для slug race убедитесь, что constraint действует при одновременном создании. Для redirect проверяйте Location и отсутствие outbound request. Для schema проверьте запуск с пустой базы и повторный запуск. Для error response и журналов используйте явные secret fixture и утверждайте отсутствие точной строки.

Параметры pgx разделяют values и SQL text. Migrations предоставляют воспроизводимую схему. Transaction охватывает именно согласованные writes, но не browser response или внешние вызовы. После успешного commit соединение может оборваться до того, как браузер увидел ответ; retry API operation поэтому требует продуманного контракта, не бездумной повторной SQL команды.

## Readiness, compose и отдельный project worker

Compose P2 публикует только loopback host ports для local developer. В Academy P2 worker student Go server ограничен и подключён только к свежей PostgreSQL в job. Trusted checker находится отдельно от student source; stdout программы не присваивает acceptance pass. При submit checker проверяет фактические HTTP сценарии, persistence после рестарта процесса и наличие обязательных таблиц. Это реальный database-backed evidence для проекта, отдельный от function grader главы.

Не делайте вывод, что учебная ссылка через проектный worker автоматически является публичным сервисом. LinkHub не компилирует student source; функция платформы выполнения исходников имеет отдельную security boundary. Не открывайте локальный контейнер наружу и не превращайте одноразовую DB в shared production data.

## Нарисовать threat model, а затем проверить

Для каждой проверки запишите asset, actor, trust boundary, защиту и observable evidence. Например: links пользователя A — asset; пользователь B — actor; session principal/repository SQL — boundary; owner predicate — защита; GET/PATCH/DELETE под двумя независимыми cookie и generic 404 — доказательство. Если тест только вызывает pure helper, укажите это, а не формулируйте «ownership защищён end-to-end».

Угрозы для project review: подмена owner ID; CSRF с cookie; race за уникальный alias; cross-owner tag; invalid redirect URL; серверный fetch к target; слишком большой body; фиксация только части транзакции; query/token в журналах; revocation session. У каждой угрозы обозначьте конкретный check. Убедитесь, что negative case действительно пытается нарушение, а не запускает обычный success сценарий.

### Задачи этого урока

Easy вычисляет conjunction для активной проверенной destination. Medium классифицирует доступ после проверки идентичности и скрывает чужой resource category. Hard совмещает owner equality и commit state. Эти функции помогают рассуждать и упражняются на public/hidden vectors; ни одна не исполняет HTTP, pgx или транзакцию. Для реальной гарантии нужны acceptance/integration checks project worker и отдельное ручное review по rubric.

## Разбирать отказ систематически

Сначала воспроизведите одно ожидание по контракту: method, path, status, headers и JSON semantics. Затем отделите parse/validation, auth/CSRF, business decision, repository и DB failure. Один тест должен сделать отказ понятным. После изменения проверьте узкий кейс, suite и новую пустую DB. Если Compose настроен, повторите clean start. Не добавляйте исправление в нескольких слоях сразу; оно затруднит выяснение, какая граница нарушена.

Логи должны сохранить request id и безопасные поля, но не token/cookie/payload. Internal error logging можно анализировать отдельно от публичного ответа. Сверяйте документированный OpenAPI с выходом сервера. Не заменяйте чёрный ящик сверкой private function names: route contract важнее названия внутреннего метода.

Финишный trace должен быть объясним без пропусков: browser → net/http/middleware → session/CSRF/owner check → application operation → parameterized repository query → constraint/transaction → response и ограниченный log. Отметьте context на storage boundary, место commit и тест, который наблюдает эффект.


## Три законченных запускаемых Go-примера

### Redirect

```go group=redirect mode=run
package main

import "fmt"

func redirect(enabled,valid bool) bool{return enabled&&valid}
func main(){fmt.Println(redirect(true,true),redirect(true,false),redirect(false,true))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

### Ownership

```go group=ownership mode=run
package main

import "fmt"

func owner(authenticated bool,principal,row string) string {if !authenticated{return "unauthorized"};if principal==""||row==""||principal!=row{return "not_found"};return "allow"}
func main(){fmt.Println(owner(true,"user-a","user-a"),owner(true,"user-b","user-a"))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

### Transaction

```go group=transaction mode=run
package main

import "fmt"

func linked(same,commit bool) bool{return same&&commit}
func main(){fmt.Println(linked(true,true),linked(true,false),linked(false,true))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

## E / M / H: практика функции

- **Easy: Перенаправляй только валидную активную ссылку.** Разреши redirect только при enabled и ранее выполненной validation URL. Pure helper не проверяет базу или сетевую политику. LinkHub не должен сам вызывать HTTP к target.
- **Medium: Проверь object ownership.** Без сессии верни unauthorized. Пустые либо несовпадающие непустые owner ID → not_found. Только совпадение дает allow. Session owner устанавливается сервером; не читай owner из body/query. Helper не проверяет SQL predicate или server session.
- **Hard: Свяжи ссылку и tag атомарно.** Пустые или разные владельцы дают rejected; совпадающие владельцы и незавершённая transaction дают rolled_back; только совпадающие непустые ID с committed → linked. Pure function не доказывает атомарность. Она требует P2 PostgreSQL integration test.

Function grader компилирует доверенный код и сравнивает JSON для public/hidden cases. Он запускает только helper и не исполняет PostgreSQL, Compose, HTTP API или review rubric. Названия ready, allow и reviewed_evidence принадлежат малой таблице решений этой функции; не выдавайте их за полный security pass.

## Вопросы для самопроверки

1. Откуда сервер берёт owner ID?
2. Как атомарно связать link и tag?
3. Почему сервер не fetch-ит redirect target?
4. Какие проверки требуют настоящую PostgreSQL?
5. Как hidden checker отделён от stdout?
6. Что остаётся предметом ручного review?

## Типичные ошибки и диагностика

- Owner ID принимается из body. Доверяйте principal session.
- Только GET scoped owner predicate, но PATCH/DELETE нет. Проверьте каждый endpoint с чужим ID.
- Link и association не в одной транзакции. Добавьте rollback integration test.
- Server fetch-ит адрес пользователя. Ответ redirect и исходящий запрос — разные операции; fetch не нужен.
- Fake store test выдан за DB integrity evidence. Добавьте disposable PostgreSQL test.

## Повторение и глоссарий

Проследите полный путь проекта от публичного HTTP ответа назад к handler, context, repository, transaction и configuration. Для каждого участка назовите файл, команду или test, который даёт evidence. Отдельно назовите неизвестное, требующее человека, и boundary, которая не проверялась. Не расширяйте claim за пределы наблюдаемого результата.

- **threat_model (модель угроз)** — Активы, actors, границы доверия и заранее очерченные отказы.
- **abuse_case (сценарий злоупотребления)** — Негативный путь, которым пытаются обойти ограничение.
- **owner_scope (область владельца)** — Связь действия и ресурса с principal из server session.
- **acceptance_test (приёмочная проверка)** — Проверка observable HTTP и DB контракта в работающей системе.
- **CSRF (межсайтовая подделка запроса)** — Browser запрос, использующий приложенные cookie для нежелательного действия.
- **SSRF (подделка server request)** — Риск, когда сервер обращается к пользовательскому адресу.

Оценка главы означает только самостоятельное evidence по критериям курса. Automatic tests, human review и внешнее профессиональное подтверждение являются отдельными процессами.
