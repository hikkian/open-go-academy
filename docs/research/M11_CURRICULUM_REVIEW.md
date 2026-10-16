# M11: независимая выборочная проверка учебных материалов

Дата: 2026-10-16. Первичная проверка и повторный короткий audit исправлений выполнены по рабочему дереву, без изменения реализации и контента. Основные разделы P2 ниже сохраняют snapshot обнаружения; актуальные статусы и источники доказательств приведены в конце.

Объём: `middle-03`, `middle-13`, `middle-27`, `middleplus-01`, `middleplus-13`, `senior-01`: все шесть Markdown-глав, соответствующие JSON metadata, 18 grader specs и 18 trusted references. Не проверялись остальные главы, проекты P3–P5, экзамены и runtime. Go-программы, opt-in suites и недоверенный код не запускались. Приведённые ниже результаты — сопоставление исходников и вычисление результатов конечных JSON-таблиц; это не новый build/sandbox evidence.

Итог повторной проверки: P1 в выборке не найдено. Все шесть исходных P2 исправлены в проверенных исходниках; P2-5 дополнительно закрыт после residual-check Easy/Medium. Aggregate M11 acceptance этим отчётом не подтверждается: отдельный глобальный metadata blocker описан в конце. Ограничения доказательства локальных helper-функций в целом названы честно.

## P2-1. Все starter-функции senior-01 имеют несуществующий тип результата

Путь: `internal/content/data/senior-01.json:69`, `:99`, `:129`.

У `RequirementKind`, `CriterionState`, `NextStep` в `starter_code` записано `strin` вместо `string`. Соответствующие `internal/grading/specs/senior-01-{easy,medium,hard}.json` и trusted references используют `string`. Студент, заменивший только `panic("TODO")` правильным телом, сохраняет ошибочную сигнатуру и получает `undefined: strin` либо отказ проверки сигнатуры. Это дефект предоставленного кода, а не решения студента.

Статус: исправлено координатором после обнаружения. По сообщению координатора, все три исправленных starter прошли `go test -run=^$` на Go 1.27.2. Этот запуск не выполнялся рецензентом. Исходные строки и симптомы выше описывают snapshot обнаружения.

Минимальная рекомендация: сохранить проверку компиляции исходных starters отдельно от trusted references, чтобы такой дефект обнаруживался до педагогического review.

## P2-2. Easy-контракт middleplus-01 не определяет вычисляемый результат

Пути: `internal/content/data/middleplus-01.json:47`, `internal/content/data/middleplus-01.md:98`, `internal/content/solutions/middleplus-01-easy/main.go:3`.

Prompt указывает сигнатуру `BenchmarkDelta(before, after int64) int64`, но единственное описание easy — «baseline». Не указаны операция, знак, единицы и диапазон; Markdown отправляет читателя к «точному описанию в starter/spec», хотя starter содержит только `panic`. Эталон вычисляет `before-after`. По одному публичному примеру `(100,90)→10` нельзя различить знаковую дельту, модуль разности и процентное улучшение: все три дают 10. Private `(0,0)→0` этого также не различает. Содержимое server-side spec не заменяет условие, доступное ученику.

Минимальная рекомендация: прямо определить `before-after`, смысл знака и допустимый диапазон в JSON и Markdown; добавить случай `after>before`, не совпадающий с модулем/процентом.

## P2-3. У middle-03 и middle-13 E/M/H являются копиями одного упражнения

Пути: `internal/content/data/middle-03.json` (три поля `prompt` и `starter_code`); `internal/content/data/middle-13.json`; matching trusted references и specs.

В `middle-03` все сложности имеют один контракт `hasValue → value; иначе closed → closed; иначе waiting`, идентичные тела эталона и идентичные public/private inputs. Различаются только имена функций и метки easy/medium/hard. То же сравнение тел у `middle-13` даёт полное совпадение: все три функции классифицируют `rows,matching` по одному порогу. У easy и medium даже private fixture идентичен public `(1000,10)→index`; hard добавляет один отрицательный вход. Три раза сданное одно решение не обеспечивают возрастающую практику. В частности, middle-03 декларирует построение producer/consumer и ownership закрытия, но оценивает только две булевы переменные и не требует канала.

Воспроизводимое свидетельство: чтение трёх `main.go`, удаление только префикса до первой `{` и сравнение оставшихся тел даёт по одному уникальному телу для каждой из этих двух глав. Для middle-03 JSON-таблицы также полностью совпадают.

Минимальная рекомендация: оставить классификатор как easy, задать два действительно разных последующих контракта, соответствующих теме и prerequisite. Для каналов отдельно проверять завершение/закрытие управляемого producer/consumer, если текущий тип grader поддерживает такое доказательство; иначе явно вынести это в обязательную проверяемую лабораторию и не считать три копии полноценной E/M/H-практикой.

## P2-4. Глава об EXPLAIN не обучает чтению EXPLAIN и содержит объяснение чужого кода

Путь: `internal/content/data/middle-13.md:23`, `:37`, `:59`, `:71`.

Все три запускаемых примера — одна функция `ChoosePlanHard` с разными числами. В главе нет SQL-запроса `EXPLAIN`, фрагмента плана и разбора scan node, estimated/actual rows или стоимости; студенту предлагается понять планы без самодостаточного примера по основной теме. Раздел «Табличный тест» содержит единственный `fmt.Println`, без таблицы, ожиданий и проверки. После третьего примера текст утверждает «conflict важнее found», хотя в программе нет ни `conflict`, ни `found`: она возвращает `invalid/index/sequential`.

Минимальная рекомендация: заменить шаблонные разделы на небольшой закреплённый SQL fixture, пример плана и объяснение его полей; показать реальный табличный тест либо честно назвать фрагмент демонстрацией; исправить объяснение ветвей на фактические категории. Эвристику четверти строк можно оставить только как явно отдельную игрушечную модель, как уже оговорено в prompt.

## P2-5. Скрытые проверки authn/authz не обнаруживают пропуск основных проверок доступа

Путь: `internal/grading/specs/middle-27-hard.json:9` и `:19`.

Всего два входа. Public имеет одинаковые tenant и ожидает true; private имеет разные tenant и ожидает false. В обоих authentication, canWrite и owner совпадают/разрешены. Поэтому неправильный ответ

```go
return principalTenant == taskTenant
```

проходит все указанные vectors, хотя игнорирует authentication, write permission, непустой tenant и owner. Свидетельство проверено вычислением этой логики над исходными JSON-массивами: public `[true]` и private `[false]` точно совпадают с expected. Это не запуск Go-кода. В `middle-27-medium` также не проверены unauthenticated и пустые ID, в easy — пустой tenant.

Минимальная рекомендация: добавить отдельный отрицательный случай на каждую обязательную предпосылку, два пустых tenant, неверного owner при верном tenant и сочетания, проверяющие приоритеты. Для hard простые известные mutant-реализации должны отклоняться.

## P2-6. Числовые trusted references нарушают свой контракт при допустимых больших входах

Пути: `internal/content/solutions/middle-13-{easy,medium,hard}/main.go:7`; `internal/content/solutions/middleplus-13-{medium,hard}/main.go:4`; соответствующие JSON prompts и Markdown-примеры.

В `ChoosePlan*` допустимы любые неотрицательные int64 с `matching<=rows`, но сравнение реализовано как `matching*4<rows`. При `rows=matching=9223372036854775807` произведение int64 переполняется в `-4`, функция выбирает `index`; по контракту совпадают все строки, поэтому требуется `sequential`.

У `CapacityDecision/CapacityFits` допустимые диапазоны не ограничивают `workers` сверху, но до деления вычисляется `workers*(100-reservePercent)`. На 64-битной платформе `workers=9223372036854775807, arrival=0, reservePercent=0` даёт переполненное произведение `-100`, затем budget `-1`: medium возвращает `over`, hard — false вместо `within`/true. Аналогичная проблема существует на 32-битном int с соответствующим MaxInt. Вывод получен разбором арифметики; новый Go-запуск не выполнялся. Текущие маленькие vectors этого не обнаруживают.

Минимальная рекомендация: сравнивать доли без переполнения с учётом точного округления либо явно ограничить диапазон входа и согласовать validation, prompt, примеры и vectors. Для выбора плана безопасно сравнивать `matching < rows/4 + indicator(rows%4!=0)` после проверки диапазонов. Для процента можно вычислять `workers/100*p + (workers%100)*p/100`, где `p=100-reservePercent`. Добавить MaxInt/MaxInt64, ноль и значения по обе стороны порога.

## Что проверено без отдельных замечаний P1/P2

В `senior-01.md` три локальных примера согласуются с их ограниченными контрактами и не выдают helper за систему архитектурных решений; проблема находится в starter JSON. В `middle-27.md` правильно выделены доверенный principal, tenant, owner и пределы локальной модели; дефект — недостаточная проверка этих предпосылок grader. У `middleplus-13` обычные маленькие входы соответствуют описанной целочисленной модели, но это не снимает переполнение на неограниченном типом диапазоне. Остальная выборка не является свидетельством качества всех 88 глав M11.

## Повторная независимая проверка исправлений (2026-10-16)

Рецензент повторно прочитал JSON/Markdown/spec/reference тех же шести chapter IDs, а также пять validation-файлов ниже. Продукт, учебный контент и тесты не менялись. Никакие Go-программы, opt-in acceptance и недоверенные программы рецензентом не запускались. Дополнительно выполнен read-only Python-разбор metadata/spec: все **18 starter signatures совпадают с grader signatures**. Вычисление tenant-only mutant над текущими JSON-входами даёт **3/6 несовпадений public и 5/6 private**; прежняя ошибочная реализация теперь отклоняется. Проверка 15 capacity vectors средствами Python с неограниченными целыми подтвердила соответствие declared математическому контракту, включая MaxInt64. Это source/vector evidence, не sandbox run.

| Finding | Текущий статус | Что рецензент непосредственно видел |
|---|---|---|
| P2-1 Senior starters | Исправлен | Все три `starter_code` возвращают `string`; сигнатуры совпадают со spec/reference. В рабочем дереве есть журнал проверки всех трёх starters. |
| P2-2 M+01 Easy | Исправлен | JSON prompt и Markdown теперь явно задают `before-after`, единицы, знак и диапазон `[0,MaxInt64]`. Public `(100,110)→-10` и private `(90,110)→-20` отличают результат от модуля/процента. |
| P2-3 E/M/H duplication | Исправлен в выборке | M03: `ChannelState`, `ReceiveResult`, `BufferedSend` имеют разные сигнатуры/семантику и соответствующие drivers/vectors. M13: классификация selectivity, строгая граница индекса, read/write trade-off. Тела больше не являются копиями одного задания. |
| P2-4 EXPLAIN teaching | Исправлен | M13 содержит bounded temporary SQL fixture на 10 000 строк, индекс, `ANALYZE`, `EXPLAIN`, иллюстративный план с `cost/rows/width` и ограничения модели. Чужие категории `conflict/found` удалены. Раздел больше не выдаёт `fmt.Println` за табличный тест. SQL здесь прочитан, но не выполнен рецензентом. |
| P2-5 authn/authz vectors | **Исправлен, residual-check завершён** | Hard: JSON/MD/reference/driver согласованы на новой role/operation matrix; 6 public + 6 private cases обнаруживают tenant-only mutant. Easy теперь имеет 2 public + 2 private cases, включая отдельно пустой principalTenant и resourceTenant. Medium имеет 2 public + 3 private cases: allow, unauthenticated, пустой principalID, пустой ownerID, foreign owner. |
| P2-6 overflow | Исправлен в source/vector scope | M13 использует `(total-1)/4` после `total>0`, остальные функции не умножают произвольные int64. M+13 Medium/Hard и MD используют `workers/100*p+(workers%100)*p/100` после валидации; vectors содержат MaxInt64, запас 0/1%, границу 80/81 и invalid. Совпадение 15 vectors с математическим контрактом проверено рецензентом; отдельного production-log M+13 в дереве пока нет. |

### Видимые validation-файлы и точные пределы их свидетельства

- `docs/validation/m11-senior01-starter-fix.txt`: указана команда `go test -run=^$ .` для каждого starter в свежем временном модуле, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`; pinned Go `go1.27.2`. Записано `PASS` для easy/medium/hard. Рецензент видел этот сохранённый summary, но не полный stdout compiler и не повторял запуск.
- `docs/validation/m11-middleplus01-easy.txt`: `scripts/go.sh test -overlay=/tmp/middleplus_01_easy_overlay.json ./internal/content -run '^TestMiddlePlus01EasyScopedOverlay$' -count=1` — записан `PASS`; summary сообщает production `LoadFor`, `BindReference`, `Request`, `Check`, starter compilation и reference execution. Временный overlay в репозитории отсутствует; точный набор assertions независимо не воспроизведён.
- `docs/validation/m11-middle13-fix.txt`: `go test -count=1 -overlay=<temporary-overlay.json> ./internal/grading -run '^TestMiddle13ProductionValidation$' -v` — записан `PASS`: easy 10.39 s, medium 11.99 s, hard 11.90 s, total 34.288 s. Указан pipeline `LoadFor → BindReference → Request(private=true) → isolated runner Execute → Check`. Это сохранённый summary; placeholder/удалённый overlay не позволяет повторить его дословно из файла. Там же сообщается компиляция/запуск трёх Markdown-примеров, но отдельных raw outputs нет.
- `docs/validation/m11-content-suite.txt`: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off .tools/go/bin/go test -count=1 ./internal/content` — сохранено `ok academy/internal/content 58.193s`. Сам файл явно отмечает: проверен опубликованный курс, **M11 draft acceptance пропущен**, потому что `ACADEMY_ACCEPT_M11_DRAFTS` не задан. Поэтому данный PASS не доказывает приемку новых глав M11.
- `docs/validation/m11-grading-unit.txt`: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off .tools/go/bin/go test -count=1 ./internal/grading` — сохранено `ok academy/internal/grading 0.194s`. Файл явно отличает загрузку specs/unit suite от isolated 264-reference acceptance.

### Что остаётся только сообщением автора / координатора

Для M03 передан результат production refs 3/3 и Go fences 4/4; для M27 Hard — overlay PASS 21.286 s. Координатор сообщил, что M+13 арифметика исправлена. На момент повторного чтения в `docs/validation` и `.data` не найдено отдельного per-scope журнала M03, M27 Hard или M+13, поэтому это **author-reported**, а не непосредственно просмотренное run evidence. Исходники исправлений во всех трёх случаях рецензент прочитал сам; их логическую согласованность можно подтвердить без утверждения, что рецензент запускал sandbox.

### Оставшиеся пробелы

1. **P2-5 закрыт по residual-check.** Теперь в Easy пары `("", "t1")` и `("t1", "")` ожидают `invalid`, поэтому equality-only реализация больше не проходит. Medium содержит `(u1,u1,false)→unauthenticated`, `("",u1,true)→invalid`, `(u1,"",true)→invalid`, `(u1,u2,true)→forbidden` наряду с allow. Комбинация unauthenticated+empty ID пока не проверяет приоритет одновременно истинных условий; это оставшаяся возможность усилить покрытие, не повтор прежнего отсутствия проверок authn/ID. Рецензент прочитал vectors и expected, ничего не исполнял. По сообщению автора, `TestMiddle27EasyMediumVectorsOverlay` прошёл за 21.062 s через `LoadFor → BindReference → Request → runner Execute → Check`. Отдельного журнала этого запуска в `docs/validation` при residual-check не обнаружено: результат остаётся author-reported.
2. Для самостоятельного воспроизведения scoped evidence желательно сохранить исходник overlay/test harness или стабильную команду; summary PASS с удалённым `/tmp` overlay фиксирует сообщение о запуске, но не полный повторяемый артефакт.
3. Новая M03 практика теперь честно проверяет разные состояния, однако helpers по-прежнему не исполняют настоящий concurrent producer/consumer. Эту границу текст называет явно; завершение/ownership требуют отдельного практического доказательства в лаборатории/проекте, а не вывода из helper PASS.
4. Полный candidate catalog, isolated 264-reference gate, все Markdown-примеры и проектные проверки M11 этим review не выполнены. Нельзя объявлять весь M11 готовым по шести главам или default package tests.

Новых P1 или иных новых P2 в этих шести chapter IDs при короткой повторной проверке не выявлено. Отчёт не является аудитом остальных материалов M11.

### Глобальный blocker и актуальный статус

В первоначальном снимке координатор сообщил, что acceptance остановился на `middle-27`: у 50 JSON отсутствовал обязательный `self_check`. Это было историческое coordinator-reported состояние и не доказывало объём дефекта во всём каталоге. **На 2026-10-16 blocker снят по актуальному автоматическому evidence:** fresh unfiltered gate на текущем дереве прошёл за 2925.611 s, 530 под-тестов PASS, 0 FAIL. Отчёт: `docs/validation/m11-current-tree-acceptance-20261016.txt`.

Этот gate подтверждает автоматические schema/reference/example checks для текущего draft-каталога. Независимый педагогический review остаётся выборочным; он не покрывает все 88 глав, достоверность фактических утверждений, реалистичность системных доказательств и полные проектные артефакты. Поэтому PASS не означает готовность M11 к публикации или присвоению всех рангов.
