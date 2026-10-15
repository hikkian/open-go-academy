## Цель — показать evidence, а не громкий титул

Junior+ assessment — это учебное обсуждение LinkHub. Оно собирает evidence соблюдения критериев курса: запускаемая программа, миграция с пустой схемы, тесты, OpenAPI, threat notes и объяснение trade-off. Это не устройство на работу, оценка личности, профессиональный security audit, независимая аккредитация или доверенный issuer публичных удостоверений. Локальный запуск тем более не подтверждает личность обучающегося.

Начните с чистого checkout или новой копии starter, прочитайте контракт заново и проверьте Go 1.27.2, pgx и PostgreSQL 18.6 P2 reference. Создайте локальный .env из безопасного example, замените placeholder сгенерированным паролем и проверьте gitignore. Убедитесь, что схема стартует пустой, migration применяются в верном порядке, тестовый запуск чистый. Для каждого результата запишите точную команду, версию инструмента и exit status. Сверяйте файл `content/projects/p2/compose.yaml`: PostgreSQL 18.6 pinned digest, приложение собирается на Go 1.27.2, dependency pgx/v5 5.11.0 зафиксирована в reference go.mod. Фраза «на моей машине работало» не является воспроизводимым evidence.

Не удаляйте volume, пока не знаете его имя и владелец не подтвердил, что состояние disposable. Локальный Compose сохраняет DB данные между запусками. Academy P2 worker поднимает отдельную временную базу на job; этот lifecycle не равен локальному `.env` и named volume. Не копируйте worker DSN, credential или checker state в сдачу.

## Seeded failure — сначала наблюдение, потом исправление

Получив seeded mismatch, не редактируйте сразу несколько слоёв. Повторите вход и ожидаемое поведение, запишите наблюдаемое отличие и предположение. Например, если после PATCH связи tag отличаются от представления, проверьте API schema, transaction boundary, tag owner scope и фактическую DB association. Если другой user читает ресурс, подготовьте две независимые сессии и проверьте каждый GET/PATCH/DELETE по owner ID.

Сначала напишите узкий regression test, который падает на дефекте и проходит после одного осмысленного исправления. Затем запускайте соответствующую часть suite, полный suite, migration с пустой базы, vet и build. Если возможно, повторите проектную проверку в fresh disposable DB. Сохраните failure command и pass command; не прикладывайте cookie, пароль, DSN, токен или пользовательский текст. Ответ API проверяется по status/header/разобранному JSON, а не по случайному пробелу.

SQL failure исследуется на проектном PostgreSQL: constraint, transaction, migration order и состояние до/после. HTTP mismatch — точным method и response. OpenAPI mismatch — сопоставлением schema, handler и документации. Без evidence database-state нельзя объявлять, что transaction atomicity подтверждена; stdout learner процесса не должен подменять trusted acceptance checker.

## Threat review по явным границам

Сначала перечислите assets: учетная запись, password hash, server-side session, links/tags и redirect destination. Акторы: anonymous client, authenticated owner, другой пользователь, автор target URL. Границы: browser request, middleware, authenticated principal, handler/service, repository и PostgreSQL, затем Compose network. Для каждой угрозы назовите controls и конкретный test.

Например, actor B угадывает ID объекта A: query должен использовать session owner, оба пользователя создаются в интеграционной фикстуре, wrong-owner операции дают общую not-found форму. Cross-site браузер пытается mutation: проверить настроенный Origin и CSRF header. Владелец вводит private network URL как target: validator отказал либо endpoint отдает только безопасный Location; server не выполняет outbound fetch. Ошибка tag association: обе стороны одной DB транзакции откатываются. Потерянное соединение после commit оставляет неопределённость для браузера; без специальной идемпотентности нельзя слепо повторять неидемпотентную операцию.

Threat model — ограниченный список ожидаемых рисков, не доказательство отсутствия всех проблем. После изменения схемы и кода пересмотрите предположения. Учебный проект честно сообщает, что нет email verification, recovery, MFA, distributed rate limit, audited key rotation и deployment hardening. Эти gaps делают его unsuitable для публичного production. Не обещайте, что дополнительные Docker опции превращают его в сервис для реальных пользователей.

## Security acceptance и работа команды

P2 rubric содержит автоматические проверки и manual review. Security-critical случаи служат gate: cross-user доступ, plaintext passwords, client-controlled owner или SSRF не компенсируются красивой документацией. Автотест проверяет перечисленные сценарии и конкретную конфигурацию. Ручная часть читает структуру, документацию и Git history, проверяет объяснение transaction, session/CSRF, owner predicate и компромисса. Обе части требуют наблюдаемого evidence, но отвечают на разные вопросы.

Один helper не подтверждает whole path. Три функции ниже классифицируют safe redirect decision, owner access и транзакционную outcome после того, как предыдущая логика дала значения. Они не смотрят cookie, HTTP, DB или SQL. Проверка в проектном UI использует отдельный DB worker; ручной review остаётся отдельным. Не придумывайте automatic pass по собственной оценке функции.

## Короткая защита проекта

1. Покажите clean setup: версии, local env подготовку без вывода секрета и migration с пустой схемы.
2. Назовите две команды Go test / vet / build и покажите результат.
3. Проследите один запрос browser → middleware/auth → owner scope → repository query → SQL → response.
4. Покажите два negative кейса: чужой ID и невалидный или опасный redirect либо CSRF failure.
5. Объясните один rollback тест и один пункт threat model.
6. Покажите contract в OpenAPI, затем фактический request/response.
7. Перечислите, что проверено автоматикой, что вручную и что остаётся открытым.

Ответьте не только «почему выбрали индекс», но и что он стоит при записи, где план подтверждает гипотезу, на каких данных замер сделан. Если evidence не существует, назовите его непроверенным и предложите небольшую проверку. Уважайте неизвестность: test case доказывает только описанный пример, не universal behavior.

## Итоговый evidence пакет

Соберите ссылки на команды, contract, threat notes, migration и регрессионные tests. Укажите число прохождения rubric, но сохраните разделение между automatic и manual. Review должен быть повторяем другим человеком. Убедитесь, что git diff сфокусирован и секреты не отслеживаются. После исправления заново запускается relevant suite; после изменения handler или SQL нужен новый HTTP/DB test соответственно.

В отчёте отделите локальный reproducibility от запуска Academy worker. Локальная команда использует Docker Compose и volume на компьютере автора; проектная автоматическая проверка использует временную DB в отдельном контейнерном job и trusted checker. Один успешный запуск reference не проверяет student fork, а успешный code build не подтверждает API persistence. Укажите, какая именно среда выполняла каждую команду, где лежит её test source и что осталось ручным. Если helper вернул reviewed_evidence, это лишь выход helper; reviewer должен увидеть actual tests и human review record прежде, чем сделать course conclusion.

Hard exercise возвращает reviewed_evidence только если automatic check, security gate и ручная review уже представлены входными сигналами. Это метка этой маленькой pure функции. Она не ставит реальную оценку, не назначает Junior+ ранг и не делает credential. Платформа и reviewer сами должны подтвердить фактические данные по установленной процедуре. Оставшиеся риски внесите в отчёт, вместо того чтобы объявить полный pass.


В rubric автоматическая часть оценивает наблюдаемые endpoint и DB свойства; ручная часть читает код, API/operations docs, trade-off и Git history. Записывайте отдельно фактический балл, gate result и unresolved review. Если проект набрал число баллов, но обязательный ownership gate упал, нельзя сообщать, что он выполнен. Если все автоматические cases проходят, это не означает, что вся система не имеет уязвимостей.

## Три законченных запускаемых Go-примера

### Evidence

```go group=evidence mode=run
package main

import "fmt"

func stage(auto,clean,docs bool) string {if !auto||!clean||!docs{return "incomplete"};return "manual_review_required"}
func main(){fmt.Println(stage(false,true,true),stage(true,true,true))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

### Gates

```go group=gates mode=run
package main

import "fmt"

func disposition(auto,security,manual bool) string {if !security{return "security_gate_failed"};if !auto{return "automatic_checks_incomplete"};if !manual{return "manual_review_required"};return "reviewed_evidence"}
func main(){fmt.Println(disposition(false,true,false),disposition(true,false,true),disposition(true,true,false),disposition(true,true,true))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

### Failed Checks

```go group=failed-checks mode=run
package main

import "fmt"

type Check struct{Name string;Passed bool}
func failed(checks []Check) []string {out:=[]string{};for _,c:=range checks{if !c.Passed{out=append(out,c.Name)}};return out}
func main(){checks:=[]Check{{"migration",true},{"owner-scope",false}};fmt.Println(failed(checks))}
```

Это полная программа со стандартным импортом, package main и main. Она использует только фиксированные локальные значения: не читает секреты окружения, не соединяется с сетью и не утверждает, что проверила сервис. Запустите пример и измените ровно одну входную величину.

## E / M / H: практика функции

- **Easy: Сгруппируй failure без заявления о pass.** Точные symbolic keys: http_contract → http; auth_session → authentication; owner_scope → ownership; postgres_integrity → database; redirect_boundary → redirect; неизвестное значение → unknown. Это группирует evidence, не запускает acceptance suite.
- **Medium: Подготовь evidence для ручного review.** Если любой сигнал false — incomplete; если все true — manual_review_required. Выход не означает ранг, credential, security certification или профессию.
- **Hard: Раздели security gate и ручное решение.** При securityGatePassed=false → security_gate_failed; иначе если automaticPassed=false → automatic_checks_incomplete; иначе если manualReviewComplete=false → manual_review_required; иначе reviewed_evidence. Последняя категория означает только просмотренный evidence курса, не выдачу ранга, публичного credential или профессиональной квалификации.

Function grader компилирует доверенный код и сравнивает JSON для public/hidden cases. Он запускает только helper и не исполняет PostgreSQL, Compose, HTTP API или review rubric. Названия ready, allow и reviewed_evidence принадлежат малой таблице решений этой функции; не выдавайте их за полный security pass.

## Вопросы для самопроверки

1. Какие команды доказывают clean setup?
2. Почему failure нужно воспроизвести до изменения?
3. Как regression test связывает fix с багом?
4. Почему security gate не складывается с остальными баллами?
5. Что означает manual_review_required?
6. Какие claims нельзя делать по локальной course оценке?

## Типичные ошибки и диагностика

- Run tests прошёл, но не записана команда и версия. В evidence пакете фиксируйте оба значения.
- Несколько файлов меняются до воспроизведения. Сначала минимизируйте failing case.
- Fix не имеет regression test. Сохраните исходное наблюдаемое поведение в тесте.
- Security failure компенсируется баллами других частей. Mandatory gate блокирует вывод.
- Учебное reviewed evidence выдается за credential или должность. Ограничьте claim rubric курса.

## Повторение и глоссарий

Проследите полный путь проекта от публичного HTTP ответа назад к handler, context, repository, transaction и configuration. Для каждого участка назовите файл, команду или test, который даёт evidence. Отдельно назовите неизвестное, требующее человека, и boundary, которая не проверялась. Не расширяйте claim за пределы наблюдаемого результата.

- **assessment (учебная оценка)** — Сбор evidence проекта по критериям учебного курса.
- **evidence (доказательство)** — Повторяемый результат команды, теста, review или объяснения.
- **regression_test (регрессионный тест)** — Проверка, которая сохраняет восстановленное поведение.
- **security_gate (обязательный критерий безопасности)** — Условие, провал которого нельзя компенсировать другими пунктами.
- **manual_review (ручное ревью)** — Оценка решения и объяснения человеком.
- **credential (учётное подтверждение)** — Формальное удостоверение отдельного issuer с явными claims.

Оценка главы означает только самостоятельное evidence по критериям курса. Automatic tests, human review и внешнее профессиональное подтверждение являются отдельными процессами.
