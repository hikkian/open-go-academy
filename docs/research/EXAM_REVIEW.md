# Независимое ревью M7: экзамен и подтверждение Zero

Дата: 2026-10-15. Ревьюер: настроенный `gpt-6.1-sol`, независимый проход по рабочему дереву поверх `011517d`. Реализация в ходе ревью не изменялась. Область — локальная установка с одним профилем; внешняя аттестация личности и публичный multi-tenant режим не рассматриваются.

## Результат

При первичном проходе найдены два воспроизводимых дефекта уровня **Medium**: потеря свежего черновика при позднем сохранении и прерывание живой проверки вторым процессом API. **Оба закрыты после повторного ревью исправлений 2026-10-15.** Открытых High/Medium замечаний в проверенной области нет. Блокирующих High/Critical путей присвоения ранга через клиентский результат в прочитанном коде не найдено. Это результат ограниченного ревью, а не доказательство отсутствия уязвимостей.

## Контекст и границы доверия

Прочитаны AGENTS.md, AGENT_HANDOFF.md, раздел G MASTER_SPEC, раздел M7 EXECUTION_PLAN, THREAT_MODEL, `internal/httpapi/exams.go`, `internal/assessment/assessment.go`, `internal/store/exams.go`, migration0003, изменения learning/API/startup и соответствующие текущие unit tests. Дополнительно прочитан `web/src/Exams.tsx` для анализа сохранения. Применены инструкции audit-context-building и differential-review; подагенты не запускались согласно прямому ограничению задачи.

Поток данных: браузер передаёт ответы/исходник и nonce → API проверяет DTO и версию → store атомарно переводит active→grading → API передаёт чистую функцию и доверенный драйвер отдельному runner → API сравнивает результат с приватными векторами → store в одной транзакции сохраняет verdict и award. DB/оператор установки и серверный grade caller доверенные; SQL-редактирование локальным владельцем вне модели угроз.

Blast radius: `ClaimExamAttempt` и `FinishExamAttempt` имеют по одному production caller в HTTP submit; `SaveExamDraft` — один HTTP caller; `RecoverExamAttempts` — один startup caller. `WithAwards` применяется при GET learning snapshot. Новые exams/assessment файлы отсутствуют в baseline, удалённых исторических security guards в них нет; история startup сверена через git blame.

## M7-R1 — Medium, CLOSED: поздний черновик молча затирает актуальные ответы

**Пути:** `internal/store/exams.go:221` (UPDATE на момент проверки — строка233); `internal/httpapi/exams.go`, draft endpoint; `web/src/Exams.tsx:55`.

`SaveExamDraft` проверяет token и active, но не проверяет версию сохранённого черновика. Две вкладки используют один token из localStorage. При открытии попытки каждая вкладка автоматически сохраняет загруженный snapshot через650мс; очередной поздний save старой вкладки полностью заменяет `answers_json` и `codes_json` новой вкладки и получает успешный ответ. Похожая перестановка возможна для уже отправленных запросов одной вкладки. После перезагрузки работа потеряна; фраза «Черновик сохранён» не сообщает о конфликте.

**Воспроизведение:** открыть одну active попытку в двух вкладках, во второй сохранить новые ответы/код, затем изменить один ответ в первой со старой копией кода. Перезагрузка второй показывает старый код. Исполненный store probe на отдельной временной БД выполнил `SaveExamDraft(...,{"q":"fresh"},{})`, затем `SaveExamDraft(...,{"q":"stale"},{})`: оба вызова вернули nil; чтение показало `{"q":"stale"}`. Probe не затрагивал `.data/academy.db` и удалён после выполнения.

**Минимальная рекомендация:** добавить revision в attempt DTO и draft запрос; атомарный `UPDATE ... WHERE revision=?`, увеличивать revision при успехе; устаревшему клиенту409. UI должен сохранять текущий текст и предлагать восстановление/повтор после конфликта. Добавить тест двух сохранений с одной revision и тест сохранения после claim: последнее не должно менять grading snapshot.

**Повторная проверка:** migration0004 добавляет revision; SaveExamDraft и ClaimExamAttempt атомарно сравнивают/увеличивают её. API требует непустую неотрицательную revision. HTTP regression подтверждает409 для устаревшего draft/submit и запрет изменения completed. UI сериализует saves, сбрасывает их перед submit, сохраняет текст на экране при409 и предлагает явную загрузку серверной версии. `TestExamDraftRevisionRejectsStaleTabAndSubmit` и `TestExamBoundariesDraftCASPassFailAndReplay` PASS без test cache. Отдельный повторный probe подтвердил, что оба старых запроса теперь возвращают conflict. Исходный сценарий устранён; CLOSED. Браузерное поведение повторно статически прочитано, фактический E2E выполняет родительский агент.

## M7-R2 — Medium, CLOSED: второй API на другом порту прерывает действующий экзамен

**Пути:** `cmd/api/main.go:31–46`; `internal/store/exams.go:403`, UPDATE на момент проверки — строка409.

Startup безусловно переводит все grading попытки в interrupted. Ранний `net.Listen` защищает только конкретный TCP address. Второй процесс с другим `ACADEMY_ADDR` и тем же `ACADEMY_DB` успешно открывает БД и ошибочно считает grading первого процесса оставшейся после сбоя. Исходный submit затем не может завершить попытку; пользователь теряет действующую проверку и должен пересдавать. Это эксплуатационная гонка локального запуска, а не удалённый путь повышения ранга.

**Воспроизведение:** запустить API на8380 с отдельной тестовой БД, начать submit с медленной sandbox-функцией; пока status=grading, запустить другой API на8381 с той же БД. Статус становится interrupted; первый submit получает conflict при Finish. Исполненный эквивалентный store probe открыл два Store на одной временной БД: первый сделал Claim, второй — Recover; первый прочитал interrupted, а `FinishExamAttempt(...,100,true,[])` вернул `exam attempt state conflict`.

**Минимальная рекомендация:** удерживать эксклюзивный process lock по каноническому пути БД до открытия/recovery и до выхода процесса. Отказывать второму процессу с понятной ошибкой независимо от порта. Альтернатива — owner/lease для grading с восстановлением только заведомо умершего владельца, но это сложнее локального lock. Добавить regression второго процесса на другом порту и существующей живой grading попытки; отдельно сохранить проверку recovery после crash.

**Повторная проверка:** `internal/apilock/lock_unix.go` удерживает nonblocking OS flock на открытом inode файла БД; main получает lock до Store.Open и Recover. FD живёт до завершения API. У другого port остаётся тот же lock; path alias не создаёт отдельную блокировку. Unit подтверждает отказ второму владельцу и освобождение; отдельный probe с symlink также получил отказ. Неподдерживаемые ОС явно отказывают в запуске. Прямой Store API по-прежнему доверенный; production startup не запускает recovery без lock. Исходный сценарий устранён; CLOSED. Отдельный фактический запуск двух процессов reviewer не выполнял.

## Проверенные защитные свойства

- JSON decoder отвергает неизвестные поля; клиент передаёт только token/answers/codes, `passed`, score и rank рассчитывает API.
- Public Exam DTO не сериализует correctIDs/cases/drivers. Attempt не содержит token hash; история возвращает ответы ученика, а не answer key.
- 32 случайных байта nonce; сохранён SHA-256; claim проверяет nonce и active внутри транзакции. Partial unique index ограничивает открытые попытки; повторный claim отклоняется.
- Submit проверяет ID/version/hash до исполнения. Хеш включает приватные ответы, векторы и драйвер. Регистрация версии отвергает замену критериев.
- Source policy отвергает import/init/main/global vars/types; драйвер контролирует ввод/вывод. Поддельный stdout/PASS не читается как verdict; API требует точное stdout, пустой stderr и успешный exit для каждого случая. Все12 objective и12 practical проверок обязательны.
- Выдача award и completed verdict атомарны; score100 обязателен; snapshot criteria и prerequisite evidence проверяются перед award. XP и P1:auto не используются как основание подтверждённого ранга.
- `WithAwards` получает server awards, выбирает высший известный ранг. Retention сохраняет award-linked attempt; active/grading не удаляются pruning.

## Запущено и ограничения доказательств

Фактически выполнено:

```sh
.tools/go/bin/go test ./internal/assessment ./internal/store ./internal/learning
```

Результат: все три пакета PASS (store/learning использовали Go test cache). Дополнительно выполнен описанный временный Go probe на отдельной БД; exit0, оба дефекта воспроизведены. Штатная БД не менялась.

На момент первичного прохода exam-specific HTTP tests на диске отсутствовали; в повторном проходе прочитан и выполнен `internal/httpapi/exams_test.go`. Fake runner тестирует HTTP state machine и выдачу award, но не является доказательством работы Go sandbox. Unit tests покрывают DTO/hash/структуру source, reference vectors, immutability, prerequisites, replay, concurrent start, recovery, score consistency, CAS и retention. Ревьюер не запускал реальные sandbox/E2E, полный race-suite и crash процесса для M7; результаты родителя должны фиксироваться отдельно. Native reference test запускает только доверенный reference source, не пользовательский код.

Повторно фактически выполнено без кеша:

```sh
.tools/go/bin/go test -count=1 ./internal/apilock ./internal/store ./internal/httpapi ./internal/assessment ./internal/learning
```

Все пять пакетов PASS. Повторный временный Go probe (отдельная БД, удалён, exit0) подтвердил: second-owner lock через symlink отклонён; stale draft и stale submit отклонены; удаление prerequisite между claim и finish блокирует award, award count0 и status остаётся grading после rollback; несовпадение зарегистрированного criteria_hash также блокирует finish. SQL-мутации служили только проверкой guards на изолированном fixture, а не моделью HTTP-атакующего. Критерии/статус/award сохраняются в одной транзакции; production API передаёт доверенную grade100 только после всех24 checks. Предпосылки задаёт доверенный Exam DTO в StartExam, а не входящий HTTP запрос.

Ограничения: конечные детерминированные vectors проверяют конкретные примеры, а не доказывают функцию для всех int; локальный читатель исходников может узнать cases. Отсутствие прокторинга обозначено явно. Эти свойства сами по себе не считаются дефектами локальной версии. Новые существенные изменения API/source policy/award после этого прохода потребуют дополнительного ревью.
