# Выборочное педагогическое и техническое ревью M10

Дата: 2026-10-16. Ревьюер: настроенный `gpt-6.1-sol`. Scope — только опубликованные `zero-04.{md,json}` и `juniorplus-15.{md,json}`, их три reference/spec каждый. Для проверки знакомости Sprintf адресно проверено упоминание в prerequisite zero-02; весь курс и полная prerequisite chain не перечитывались. Reviewer меняет только этот отчёт, материалы не переписывал, подагенты не запускались.

## Итог текущего прохода

Две главы содержат содержательные русские объяснения, разобранные примеры, диагностику и практику; оценка не основана на количестве слов. **CR-R1–R4 и CR-L1/L2 исправлены и CLOSED; открытых High/Medium в targeted scope нет.** Матрица cookie policy и KDF boundaries проверены чтением данных; предоставленные родителем aggregate starter-build и targeted native results PASS. Выборочное ревью двух глав не означает педагогическую сертификацию всего117-exercise курса.

## CR-R1 — Medium, CLOSED: неверное обещание32-bit int

**Путь:** zero-04.md, Medium practice и раздел переполнения.

Текст утверждал, что сумма1..1_000_000 помещается в int на32- и64-bit. Арифметическая проверка:500000500000 > MaxInt32(2147483647). Reference использует int accumulator, следовательно, portability claim был ложным, даже если64-bit native grader проходит.

**Исправление:** оба места теперь явно задают64-bit учебный runner, приводят максимальную сумму и предупреждают о необходимости int64/другой сигнатуры для переносимости на32-bit. Исходный диапазон функции не менялся. Перечитано; CLOSED. Reviewer не запускал32-bit binary и не выдаёт вычисление границы за такой тест.

## CR-R2 — Medium, CLOSED: published starter противоречил source parser

**Пути:** zero-04.md/json; связанный контракт function source.

Инструкция говорила не добавлять package main, а три starter были только func declarations. При этом frontend передавал source как есть, Spec.Request использовал parser.ParseFile и требовал package main. Новичок, следовавший инструкции, получал бы422 вместо возможности решать логику цикла. Это не недостающая теория — это противоречие рабочего задания.

**Исправление:** все три targeted starter теперь начинаются с package main; prompt и MD говорят сохранить эту строку и не добавлять func main. Автор исправил также найденные им варианты zero02/03; reviewer их полное повторное ревью не выполнял. В существующем prerequisite zero-02 Sprintf/return/минимальная функция уже объяснены; zero-04 сам кратко вводит error/nil/two returns перед Medium/Hard, не требует обязательного изучения slices. CLOSED по targeted чтению. Полный starter compile gate отдельно отмечен в доказательствах.

## CR-R3 — Medium, CLOSED: KDF upper bound отсутствовал в тексте практики

**Пути:** juniorplus-15.md Easy против JSON/reference/spec.

MD требовал итерации «не ниже600000», но карточка/reference/private vector принимали только600000..1200000 и отвергали1200001. Ученик мог корректно следовать объяснению главы и получить hidden failure.

**Исправление:** текст теперь явно задаёт обе включительные границы и объясняет верхнюю как CPU budget учебной конфигурации, а не ухудшение криптографической стойкости при большем work factor. Согласуется с reference. CLOSED.

## CR-R4 — Medium, CLOSED: cookie-policy fixtures не проверяли независимость сигналов

**Путь:** `internal/grading/specs/juniorplus-15-hard.json`.

На первичном targeted снимке positive cases — POST/PATCH при обоих signals=true. Invalid CSRF проверялся только для DELETE, invalid Origin — только для PUT; другие отрицательные методы GET/TRACE. Поэтому обычная неверная функция, разрешающая только POST/PATCH и игнорирующая оба policy signals, не отличалась этими cases от правильной. Задание заявляет именно успешную проверку обоих signals, поэтому это существенный пробел учебной проверки.

**Минимум:** negative POST с csrfValid=false/originTrusted=true и PATCH с csrfValid=true/originTrusted=false; positive PUT/DELETE с обоимиtrue. Важно различать allowlisted method и independent security signals. Reviewer не писал и не исполнял обходную программу; вывод из таблицы входов/ожиданий. Передано автору для точечного data/test fix.

**Повторная проверка:** hard теперь содержит6 public+12 private cases. Read-only проверка JSON-матрицы подтвердила все16 сочетаний четырёх mutation methods×двух boolean signals: true ровно при csrfValid&&originTrusted; дополнительных GET/TRACE обаfalse. Missing/wrong entries0, private group укладывается в cap12. Предоставленный `.data/m10-auth-expanded-native.log` подтверждает18 actual confined cases PASS. Контракт не менялся, покрытие исправлено; CLOSED.

## Low замечания

- **CR-L1, CLOSED:** глава ошибочно называла SameSite=Lax выбором P2 reference, где используется Strict. Текст теперь различает Strict в reference и допустимый Lax в самостоятельном recorder example. Оба не объявляются полной CSRF защитой.
- **CR-L2, CLOSED:** добавлены exact1200000 accept при salt16/output32 и output33 reject при iterations600000/salt16. Данные перечитаны; Easy теперь8 cases, предоставленный native log PASS. Reference уже реализовывал правильное включение/равенство; исправлено покрытие критериев.

## Педагогические выводы в пределах двух глав

**Zero-04:** цепочка «повторение→counter/condition/step→accumulator→boundary→termination→simple invariant» объяснена своими связными абзацами; три полных runnable примера имеют предсказанный результат и объяснение. Есть конкретная диагностика off-by-one/reset/infinite loop. Errors/two returns — предварительное знакомство, но минимальное объяснение дано до заданий; Hard использует ранее введённый Sprintf/конкатенацию, а не требует ещё не представленных slices. Исправленные starter/архитектурная граница необходимы, иначе хороший текст не обеспечивал бы проходимость урока.

**Juniorplus-15:** объяснены session lookup/opaque token, verifier/salt/version, expiry/rotation/revoke, cookies и отдельные CSRF/Origin boundaries. Примеры работают с настоящими stdlib token/KDF/cookie APIs, но не выдаются за полноценную authentication систему. Practice — validators/policy predicates и time comparison; текст и JSON criteria честно ограничивают доказательство конечным наблюдаемым поведением. Они не доказывают реальную SQL session storage, качество password store, защищённость браузерного login либо отсутствие ошибок авторизации в P2. Отдельные HTTP+PostgreSQL acceptance checks проекта нужны именно для этих системных свойств.

Объяснение KDF корректно: PBKDF2 CPU-bound и не memory-hard; Argon2id использует память/время, требует зависимости и калибровки.600000 — явно учебный baseline, не вечная гарантия или измеренная cracking resistance. Это согласуется с [RFC9106 об Argon2](https://www.rfc-editor.org/info/rfc9106/) и контекстом [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), где PBKDF2-HMAC-SHA256600000 описан как конкретный вариант, а не универсальная замена оценке угроз. Проверена локальная Go1.27.2 stdlib сигнатура crypto/pbkdf2.Key; пример использует её правильно, salt16/key32 и constant-time derived comparison. Секреты и derived bytes в примерах не печатаются.

## Доказательства и пределы

Фактически выполнена только простая арифметическая проверка суммы средствами Python и read-only просмотр кода/metadata/stdlib; новые native/длинные course suites reviewer не запускал. Полные examples/117 native grade/PostgreSQL/P2 PASS сообщил интегратор; эти результаты не выданы за собственное выполнение этого педагогического ревью.

Прочитан добавленный `published_test.go` guard: каждый starter проходит production Spec.Request(public) и trusted build, а не только reference solution. Первоначальный `.data/m10-all-starters.log` завершился FAIL54.728s с проблемами за пределами targeted двух глав. Родитель исправил дополнительные metadata starters (JSON tags/generic signature/import usage); reviewer не расширял педагогический scope на эти главы. После исправлений прочитан `.data/m10-all-starters-fixed.log`: internal/content PASS48.402s. По предоставленному интегратором запуску это aggregate117 productionRequest+starterBuild вместе с reference/examples; успешный повтор не выдаётся за успех первой попытки.

Дополнительно прочитан `.data/m10-auth-expanded-native.log`: easy8/medium6/hard18 actual confined cases PASS, persisted evidence/idempotent XP/no earned rank и temporary DB. Эти native executions выполнил родитель; reviewer только сверил log и новые JSON cases, не запускал дополнительные программы. Новая проверка native cookie-policy не означает сертификацию настоящего authentication/SQL сервиса.

Проверка исходности текста здесь означает содержательный read-through русских объяснений и отсутствие зависимости от обязательного внешнего учебника; plagiarism/авторство/лицензионное происхождение всего корпуса этим проходом не исследовались. Не проверялись classroom usability, полная prerequisite chain Junior+, визуальная верстка и усвоение материала реальными новичками. Эти ограничения не подменяют найденные конкретные исправимые контрактные дефекты.
