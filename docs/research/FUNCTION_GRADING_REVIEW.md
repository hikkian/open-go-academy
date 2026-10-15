# Независимое ревью M10A: function grading

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`. Scope: учебный контракт нового function grader, AST/type/import restrictions, изоляция ожидаемых значений, HTTP dispatch и подавление приватных diagnostics. Reviewer меняет только этот файл. Прямое ограничение задачи: без proof-of-exploit и без запуска враждебного кода; соблюдено.

## Текущий результат

Первичный проход выявил несопоставление пользовательского граничного типа и сериализационных методов с контрактом, некорректную двойную JSON-кодировку части vectors и сериализацию error как конкретной реализации вместо наличия ошибки. **FG-R1/R2/R3 исправлены и CLOSED после повторного чтения и focused suite; открытых High/Medium в проверенной области нет.** Обход выдачи ранга здесь не обнаружен: function grading создаёт exercise evidence/XP, не rank award. Дополнительно прочитан и выполнен отдельный HTTP regression приватного output wipe; ранее отмеченный test gap закрыт.

## Контекст

Прочитаны `internal/grading/grading.go` и tests/specs, `internal/httpapi/function_exercises.go`, dispatch в api.go, private loader `internal/content/content.go`, релевантные reference/условия zero-04, zero-07, zero-08. Полная программа курса повторно не читалась. Использованы ранее изученные audit-context-building/differential-review инструкции; подагенты не запускались.

Авторы specs/reference доверенные. Student files недоверенные. API выбирает spec по exercise ID, генерирует runner request сам; игнорирует присланные клиентом Mode/Args/Assets для function dispatch. Expected vectors остаются на API; в sandbox входят только доверенный driver, student files и inputs. Public Run передаёт только public input; Submit — public+private batch. На submit HTTP сам рассчитывает проверки и перед сохранением/возвратом удаляет raw stdout/stderr/files, поскольку они могут содержать private input. Нового выполнения пользовательского Go на host API нет; существующий runner не изменён.

## FG-R1 — Medium, CLOSED: граничные типы/сериализация не обеспечивали контракт

**Пути:** `internal/grading/grading.go`, ValidateSource/sameSignature; zero-08 drivers.

Исходная проверка сравнивала синтаксический тип имени ScoreSummary в сигнатуре, но не его объявление, и допускала произвольные методы. Драйвер JSON-unmarshal/marshal использует этот student-defined type. Поэтому shape/aliases и пользовательские MarshalJSON/UnmarshalJSON/MarshalText/UnmarshalText могли менять наблюдаемый сериализованный результат независимо от требуемого состояния полей функции. Это дефект учебного контракта; успешное выполнение такого обхода reviewer не заявляет и не запускал.

**Минимум для закрытия:** связать используемые в signature types с доверенными объявлениями reference до обработки конкурентных requests; сравнивать транзитивную форму граничных типов, включая aliases/tags. Запретить JSON/Text serialization hooks во всех student methods/helpers/types, а не только в required function. Обычные положительные fixture с корректным struct и отрицательные с несовпадающим type/method должны проверять AST guards без исполнения student source.

**Повторная проверка:** BindReference строит immutable по назначению map канонических граничных объявлений и их вложенных named types; сравниваются поля/типы/tags, grouped fields нормализуются. Все top-level aliases в student source отвергаются. Шесть JSON/Text serialization method names запрещены независимо от receiver типа и того, является ли функция required; guard проходит по всем файлам. До BindReference ValidateSource возвращает ошибку; Catalog.Load связывает reference до публикации каталога. `TestReferenceBoundaryTypesAndSerializationHooks` принимает обычный корректный struct/helper и отвергает field type/tag/additional field/nested type и все шесть hooks. Эти отрицательные fixtures только парсятся/валидируются, не исполняются. grading package PASS0.152s, повторный PASS0.141s. Для текущих Zero boundary shapes изменение достаточно; CLOSED. Гарантия для будущих generic/qualified boundary types потребует отдельного content/reference review.

## FG-R2 — Medium, CLOSED: двойная JSON-кодировка vectors

**Пути:** часть новых `internal/grading/specs/zero-07-*.json`, `zero-08-*.json`.

В первичном дереве zero-07-hard inputs были JSON strings вида `"{\"Values\":[1,2],\"Value\":3}"`, тогда как driver декодировал []struct. Expected были strings с представлением массивов. Аналогично zero-08-medium использовал strings вместо []int и objects. Это приводит к ошибке декодирования или несовпадению genuine reference output. Данные уже начали преобразовываться к реальным JSON array/object/number/string; статус будет определён итоговыми reference tests.

**Минимум для закрытия:** все vectors имеют тип реальных driver inputs/outputs; reference всех опубликованных specs компилируется и совпадает с public+private cases. Одного json.Valid недостаточно для проверки типа.

**Повторная проверка:** данные приведены к JSON arrays/objects/strings без дополнительного слоя кодирования; newline strings zero-04-hard выровнены с reference. Loader проверяет publicOutput через тот же JSON comparator, поэтому порядок ключей объекта не ломает загрузку. TestEveryPublishedGoExampleAndFunctionReference реально компилирует/запускает только доверенные reference/учебные примеры и сверяет public+private cases; финальный content package PASS4.831s. CLOSED.

## FG-R3 — Medium, CLOSED: error проверяется по JSON-представлению реализации

**Пути:** drivers/specs zero-04-easy/medium/hard.

Условия требуют возвратить ошибку для недопустимого ввода. Драйвер кодирует `[]any{value, err}`; `errors.New` превращается в `{}`, ожидается именно `{}`. Но обычная корректная реализация error с exported fields сериализуется иначе и отклоняется, хотя err!=nil. Запрет JSON hooks сам по себе это не устраняет. Это false negative обычного учебного решения, не доказательство security exploit.

**Минимальная рекомендация:** доверенный driver нормализует наличие ошибки (boolean или фиксированный sentinel), а expected описывает это нормализованное значение; не зависит от текста/структуры error. Проверить обычные nil/nonnil ошибки и отсутствие лишнего stdout.

**Повторная проверка:** итоговые zero-04-medium и zero-04-hard кодируют `[]any{value, err != nil}`, expected содержит boolean, а условия прямо объясняют это представление. Easy был заменён countTo с string result без error. Таким образом конкретный error type/поля больше не участвуют в verdict. Доверенные positive/negative reference cases прошли content suite. CLOSED; особая реализация пользовательского error не исполнялась reviewer согласно ограничению задачи.

## Уже проверенные свойства

- Spec.MarshalJSON выдаёт только exercise ID; private driver/vectors не являются public DTO. PublicOutput строится только из publicExpected.
- Check требует statusok,exit0,пустой stderr, точное число JSON-lines и strict JSON comparison каждой public/private expected. Ошибка public case не скрывается успехом private.
- Source ограничен1..15 Go files, ordinary names без каталогов/test files, reserved academy_driver.go запрещён, student+driver≤64KiB. Combined inputs≤4096bytes, expected encoded output≤32KiB, каждый vector≤4KiB и1..12 пар в группе.
- Imports whitelist и per-spec allowlist; запрет unsafe/os/runtime/reflect/syscall/C/relative, blank/dot imports. Package variables/init/main запрещены; predeclared names защищаются от top-level rebinding. Отдельный go compiler остаётся последней проверкой корректности типов, но не заменяет учебный type contract.
- Submit заменяет все raw stdout/stderr и Files до HTTP ответа и RecordAttempt. Остаются generic check names и verdict; private expected/inputs не возвращаются. Public Run сохраняет реальные compiler diagnostics на public inputs.
- Exercise evidence не создаёт rank_award и не меняет exam policy. На момент первичного чтения `git diff -- internal/assessment/assessment.go` пуст; исходные три exercise dispatch идут по прежней ветке.

## Фактически выполнено и ограничения

```sh
.tools/go/bin/go test -count=1 -timeout=45s ./internal/grading ./internal/httpapi ./internal/content
```

Первичный запуск: grading0.145s, httpapi0.062s, content0.423s PASS. Это состояние до финального BindReference и новых отрицательных fixtures; не доказательство закрытия FG-R1/R3. Тесты выполняли только штатные положительные/отрицательные fixtures и доверенные reference builds, не proof-of-exploit. Reviewer не запускал hostile student code ни на host, ни в sandbox.

Промежуточные повторные запуски честно не считаются финальным gate: после fail-closed BindReference content временно давал21 ошибки «trusted reference has not been bound» до подключения loader; после подключения loader обнаружил несогласованный publicOutput zero-08-medium (JSON object key ordering), из-за чего старый API fixture с проигнорированным content.Load error получил nil Catalog panic. Это зафиксировано и передано интегратору, финальный suite должен пройти после выравнивания публичных результатов.

**Финальный повтор той же команды после изменений:** grading0.150s, content4.831s, httpapi0.077s — все PASS без test cache. Промежуточный zero-04-hard newline mismatch также устранён до этого запуска. Экзаменационный assessment.go и legacy zero-01 данные в diff не менялись; dispatch новой ветки требует Kind=function, исходные hello/repair/two-lines остаются в прежней ветке и существующие HTTP tests прошли.

**Уточнение доказательств:** после сообщения интегратора reviewer прочитал `internal/httpapi/function_exercises_test.go` и выполнил только указанный regression:

```sh
.tools/go/bin/go test -count=1 -run TestFunctionPublicDiagnosticsPrivateOutputAndXP ./internal/httpapi
```

PASS0.010s. Fixture использует обычный fake executor и доверенный reference, ничего пользовательского не исполняет. Marker помещён во stdout/stderr/result.Files; private HTTP response и сохранённый Attempt.Output его не содержат, неудача не даёт XP. Проверены замена клиентских Args/Assets доверенными inputs/driver, сохранность обычных public diagnostics и две успешные submissions с XP10 и persisted evidence. Для заявленной HTTP-границы fixture достаточен; ранее отмеченный test gap закрыт. Полная проверка exam hash неизменности и реальный rootless Run/Submit находятся у интегратора; reviewer не выдаёт их за выполненные им.

Public/hidden expected остаются конечными примерами и не доказывают функцию для всех входов. Private input неизбежно виден коду внутри одного sandbox batch; гарантируется отсутствие ожидаемых ответов в контейнере и подавление raw приватного вывода наружу. Независимые реальные sandbox/browser checks и итоговый полный suite выполняет родитель; reviewer не присваивает себе их результаты.
