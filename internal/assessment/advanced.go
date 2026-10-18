package assessment

import (
	"fmt"

	"academy/internal/grading"
)

const middleV100BackoffReference = `package main
func BackoffSeconds(baseSeconds, attempt, maxSeconds int) int { if baseSeconds <= 0 || attempt <= 0 || maxSeconds <= 0 { return 0 }; if baseSeconds >= maxSeconds { return maxSeconds }; if attempt > 30 { attempt = 30 }; delay := baseSeconds; for i := 1; i < attempt && delay < maxSeconds; i++ { if delay > maxSeconds/2 { return maxSeconds }; delay *= 2 }; if delay > maxSeconds { return maxSeconds }; return delay }`

// Advanced catalogs are immutable objective exams. Passing their quiz and
// function checks is evidence only; project review and rank policy stay outside
// this package and must independently verify the required artifacts.
var advancedReferences = map[string]string{
	"middle-exam-retry-decision": `package main
func RetryDecision(retryable bool, attemptsRemaining int) string { if retryable && attemptsRemaining > 0 { return "retry" }; return "terminal" }`,
	"middle-exam-backoff-seconds": `package main
func BackoffSeconds(baseSeconds, attempt, maxSeconds int) int { if baseSeconds <= 0 || attempt <= 0 || maxSeconds <= 0 { return 0 }; if baseSeconds >= maxSeconds { return maxSeconds }; delay := baseSeconds; for i := 1; i < attempt && delay < maxSeconds; i++ { if delay > maxSeconds/2 { return maxSeconds }; delay *= 2 }; if delay > maxSeconds { return maxSeconds }; return delay }`,
	"middle-exam-commit-allowed": `package main
func CommitAllowed(validated, conflictFree, contextActive bool) bool { return validated && conflictFree && contextActive }`,
	"middleplus-exam-compare-benchmark": `package main
func CompareBenchmark(baselineRuns int, baselineMedianNS int64, candidateRuns int, candidateMedianNS int64, minimumRuns int) string { if baselineRuns < minimumRuns || candidateRuns < minimumRuns || baselineMedianNS <= 0 || candidateMedianNS <= 0 { return "insufficient" }; if candidateMedianNS < baselineMedianNS { return "improved" }; if candidateMedianNS > baselineMedianNS { return "regressed" }; return "equal" }`,
	"middleplus-exam-slo-budget": `package main
func SLOBudget(targetP95MS, observedP95MS, errorBasisPoints, maxErrorBasisPoints int) string { if targetP95MS <= 0 || observedP95MS < 0 || errorBasisPoints < 0 || maxErrorBasisPoints <= 0 { return "invalid" }; if observedP95MS > targetP95MS || errorBasisPoints > maxErrorBasisPoints { return "breach" }; return "within" }`,
	"middleplus-exam-fault-gate": `package main
func FaultGate(recovered, dataLoss, errorBudgetAvailable bool) string { if dataLoss { return "stop" }; if !recovered { return "hold" }; if !errorBudgetAvailable { return "hold" }; return "proceed" }`,
	"senior-exam-action-priority": `package main
func ActionPriority(impact, likelihood int) string { score := impact * likelihood; if score >= 9 { return "now" }; if score >= 4 { return "plan" }; return "track" }`,
	"senior-exam-decision-review": `package main
func DecisionReview(triggerObserved, evidenceUpdated, ownerAssigned bool) string { if !triggerObserved { return "wait" }; if !evidenceUpdated { return "gather_evidence" }; if !ownerAssigned { return "assign_owner" }; return "revisit_decision" }`,
	"senior-exam-release-gate": `package main
func ReleaseGate(acceptancePassed, rollbackReady, independentReviewApproved, highRiskOpen bool) bool { return acceptancePassed && rollbackReady && independentReviewApproved && !highRiskOpen }`,
}

func loadMiddle() *Exam {
	e := buildExam("middle-exam", "Экзамен Middle: надёжность и проектные доказательства", "Middle", "Junior+",
		"Объективная часть проверяет Go, эксплуатационные решения и критерии P3. Допуск требует завершённого курса Middle, подтверждённого Junior+ и проверенного P3. Экзамен не заменяет review проекта и сам не присваивает ранг.", middlePrerequisites(),
		[]string{"Все упражнения Middle и prerequisite Junior+", "Независимо подтверждён ранг Junior+", "P3 TaskFlow: machine checks пройдены и артефакт закреплён за версией", "Все 15 вопросов и проверки трёх функций пройдены", "Авторство и качество проекта требуют review; экзамен не присваивает ранг"}, middleQuestions(), middleTasks(), map[string]string{"retry": "middle-exam-retry-decision", "backoff": "middle-exam-backoff-seconds", "commit": "middle-exam-commit-allowed"})
	e.Version = "1.0.1"
	return e
}

func loadMiddleV100() *Exam {
	e := loadMiddle()
	e.Version = "1.0.0"
	spec, err := grading.LoadArchived("middle-exam-backoff-seconds", "1.0.0")
	if err != nil {
		panic(fmt.Errorf("load immutable Middle 1.0.0 grading policy: %w", err))
	}
	if err := spec.BindReference(middleV100BackoffReference); err != nil {
		panic(fmt.Errorf("bind immutable Middle 1.0.0 reference: %w", err))
	}
	e.functionSpecs["backoff"] = spec
	return e
}

func loadMiddlePlus() *Exam {
	return buildExam("middleplus-exam", "Экзамен Middle+: производительность и отказоустойчивость", "Middle+", "Middle",
		"Объективная часть проверяет измерения, системное поведение при отказах и критерии P4 ScaleLab. Допуск требует завершённого курса Middle+, подтверждённого Middle и независимого review P4; тесты не оценивают качество архитектурного обоснования.", middlePlusPrerequisites(),
		[]string{"Все упражнения Middle+ и prerequisite Middle", "Независимо подтверждён ранг Middle", "P4 ScaleLab: benchmark и SLO evidence проверяются отдельно", "Все 15 вопросов и проверки трёх функций пройдены", "Профилирование и trade-off требуют review; экзамен не присваивает ранг"}, middlePlusQuestions(), middlePlusTasks(), map[string]string{"benchmark": "middleplus-exam-compare-benchmark", "slo": "middleplus-exam-slo-budget", "fault": "middleplus-exam-fault-gate"})
}

func loadSenior() *Exam {
	return buildExam("senior-exam", "Экзамен Senior: архитектурные решения и эксплуатационная готовность", "Senior", "Middle+",
		"Объективная часть проверяет risk-based решения и эксплуатационные данные вместе с P5. Для допуска нужны завершённый курс Senior, подтверждённый Middle+ и независимая аутентифицированная оценка P5. Кодовые проверки не выдают Senior и не заменяют защиту проекта.", seniorPrerequisites(),
		[]string{"Все упражнения Senior и prerequisite Middle+", "Независимо подтверждён ранг Middle+", "P5: capstone, threat model, ADR, operations и postmortem проверяются независимо", "Все 15 вопросов и проверки трёх функций пройдены", "Защита и авторство требуют review; экзамен не присваивает ранг"}, seniorQuestions(), seniorTasks(), map[string]string{"priority": "senior-exam-action-priority", "review": "senior-exam-decision-review", "release": "senior-exam-release-gate"})
}

func registerAdvancedReferences() {
	for id, source := range advancedReferences {
		assessmentReferences[id] = source
	}
}

func init() { registerAdvancedReferences() }

func advancedPrerequisites(prefix string, lessons int) []string {
	out := []string{}
	for n := 1; n <= lessons; n++ {
		out = append(out, fmt.Sprintf("%s-%02d-easy", prefix, n), fmt.Sprintf("%s-%02d-medium", prefix, n), fmt.Sprintf("%s-%02d-hard", prefix, n))
	}
	return out
}

func middlePrerequisites() []string {
	return advancedPrerequisites("middle", 40)
}
func middlePlusPrerequisites() []string {
	return advancedPrerequisites("middleplus", 24)
}
func seniorPrerequisites() []string {
	return advancedPrerequisites("senior", 24)
}

func middleTasks() []Task {
	return []Task{
		{ID: "retry", Title: "Решите, допустим ли повтор задачи", Kind: "function", Description: "Возвращайте retry только для временной ошибки при оставшихся попытках; постоянная ошибка или исчерпанный бюджет дают terminal.", StarterCode: "package main\nfunc RetryDecision(retryable bool, attemptsRemaining int) string { return \"terminal\" }\n", PublicExamples: []Example{{"true,2", "retry"}, {"false,2", "terminal"}, {"true,0", "terminal"}}},
		{ID: "backoff", Title: "Ограничьте задержку повторов", Kind: "function", Description: "Вычислите экспоненциальный backoff: attempt начинается с 1 и даёт baseSeconds; удваивайте до maxSeconds. Неположительные base, attempt или max дают 0. Сатурируйте безопасно при большом attempt.", StarterCode: "package main\nfunc BackoffSeconds(baseSeconds, attempt, maxSeconds int) int { return 0 }\n", PublicExamples: []Example{{"2,1,10", "2"}, {"2,3,10", "8"}, {"2,8,10", "10"}}},
		{ID: "commit", Title: "Проверьте условия commit", Kind: "function", Description: "Коммит разрешён, только если ввод провалидирован, конфликт отсутствует и контекст ещё активен. Это детерминированное решение, а не замена транзакции базы данных.", StarterCode: "package main\nfunc CommitAllowed(validated, conflictFree, contextActive bool) bool { return false }\n", PublicExamples: []Example{{"true,true,true", "true"}, {"true,false,true", "false"}, {"true,true,false", "false"}}},
	}
}
func middlePlusTasks() []Task {
	return []Task{
		{ID: "benchmark", Title: "Сравните измеримые baseline и candidate", Kind: "function", Description: "При достаточном числе повторов и положительных медиан ns/op сравните измерения: improved, equal или regressed. Число повторов и сама медиана — отдельные входы; одна без другой не доказывает ускорение.", StarterCode: "package main\nfunc CompareBenchmark(baselineRuns int, baselineMedianNS int64, candidateRuns int, candidateMedianNS int64, minimumRuns int) string { return \"insufficient\" }\n", PublicExamples: []Example{{"5,750000,5,600000,3", "improved"}, {"5,600000,5,600000,3", "equal"}, {"2,500000,5,600000,3", "insufficient"}}},
		{ID: "slo", Title: "Проверьте p95 и error-budget SLO", Kind: "function", Description: "Нарушение есть, если observed p95 выше target либо errorBasisPoints выше максимума. Единица ошибки — basis point: 100 bps = 1%. Некорректные пределы возвращают invalid.", StarterCode: "package main\nfunc SLOBudget(targetP95MS, observedP95MS, errorBasisPoints, maxErrorBasisPoints int) string { return \"invalid\" }\n", PublicExamples: []Example{{"40,35,50,100", "within"}, {"40,45,50,100", "breach"}, {"40,35,101,100", "breach"}}},
		{ID: "fault", Title: "Решите, продолжать ли после fault injection", Kind: "function", Description: "Потеря данных всегда требует stop; неполное восстановление или исчерпанный error budget требуют hold; продолжайте только после восстановления без потери данных и при доступном бюджете.", StarterCode: "package main\nfunc FaultGate(recovered, dataLoss, errorBudgetAvailable bool) string { return \"hold\" }\n", PublicExamples: []Example{{"true,false,true", "proceed"}, {"false,false,true", "hold"}, {"true,true,true", "stop"}}},
	}
}
func seniorTasks() []Task {
	return []Task{
		{ID: "priority", Title: "Приоритизируйте архитектурный риск", Kind: "function", Description: "Умножьте impact и likelihood по шкале 1..5: score >= 9 означает now, >= 4 plan, иначе track. На защите обоснуйте шкалы и почему произведение не заменяет качественную оценку.", StarterCode: "package main\nfunc ActionPriority(impact, likelihood int) string { return \"track\" }\n", PublicExamples: []Example{{"3,3", "now"}, {"2,2", "plan"}, {"1,1", "track"}}},
		{ID: "review", Title: "Проведите пересмотр решения", Kind: "function", Description: "Решение пересматривается после наблюдаемого trigger: сначала соберите недостающие данные, затем назначьте владельца и только потом возвращайте revisit_decision.", StarterCode: "package main\nfunc DecisionReview(triggerObserved, evidenceUpdated, ownerAssigned bool) string { return \"wait\" }\n", PublicExamples: []Example{{"true,true,true", "revisit_decision"}, {"true,false,true", "gather_evidence"}, {"false,false,false", "wait"}}},
		{ID: "release", Title: "Примените независимый релизный gate", Kind: "function", Description: "Релиз разрешён только при пройденных критериях, готовом rollback, независимом одобрении и отсутствии открытого высокого риска. Эта функция не подтверждает личность reviewer.", StarterCode: "package main\nfunc ReleaseGate(acceptancePassed, rollbackReady, independentReviewApproved, highRiskOpen bool) bool { return false }\n", PublicExamples: []Example{{"true,true,true,false", "true"}, {"true,true,true,true", "false"}, {"true,false,true,false", "false"}}},
	}
}

func middleQuestions() []authoredQuestion {
	return []authoredQuestion{
		q3("m-q-context", "Почему передают context.Context через цепочку запросов?", "Чтобы распространять deadline, отмену и request-scoped значения", "Чтобы хранить глобальную конфигурацию сервиса", "Чтобы автоматически повторять неудачные запросы"),
		q3("m-q-cancel", "Что должен сделать worker после отмены своего context?", "Как можно раньше прекратить работу и освободить ресурсы", "Игнорировать отмену до завершения процесса", "Создать новый context без deadline"),
		q3("m-q-error", "Когда уместно оборачивать ошибку через %w?", "Когда вызывающая сторона должна сохранить возможность errors.Is/As", "Чтобы превратить любую ошибку в успешный результат", "Чтобы добавить stack trace, который Go создаёт автоматически"),
		q3("m-q-race", "Как устранить data race между конкурентными goroutine?", "Задать общую синхронизацию или передавать владение сообщениями", "Добавить больше goroutine", "Запустить gofmt перед сборкой"),
		q3("m-q-lock", "Почему нельзя вызывать неизвестный callback под mutex без причины?", "Callback может блокироваться или повторно войти и создать deadlock", "Mutex запрещает вызывать функции вообще", "Callback всегда запускается на другом CPU"),
		q3("m-q-pool", "Что даёт ограниченный worker pool?", "Ограничивает параллелизм и давление на downstream ресурс", "Гарантирует отсутствие ошибок", "Автоматически обеспечивает FIFO для всех запросов"),
		q3("m-q-backpressure", "Что означает backpressure в очереди?", "Производитель замедляется или получает отказ при исчерпании ёмкости", "Очередь безлимитно накапливает работу", "Consumer завершает все запросы синхронно"),
		q3("m-q-idempotency", "Зачем нужна idempotency key для повторяемой операции?", "Чтобы распознать повтор доставки и не применить эффект дважды", "Чтобы зашифровать payload", "Чтобы гарантировать exactly-once сетью"),
		q3("m-q-db-lock", "Что делать при временной ошибке сериализации транзакции?", "Повторить целиком ограниченное число раз с учётом идемпотентности", "Повторить только последний SQL statement без проверки состояния", "Отключить транзакции"),
		q3("m-q-index", "Как проверить пользу нового индекса?", "Сопоставить план и измерения на репрезентативной нагрузке", "Добавить индекс на каждый столбец", "Судить только по размеру миграции"),
		q3("m-q-slo", "Что выражает SLO?", "Целевой уровень качества сервиса за заданное окно", "Гарантию, что инцидентов не будет", "Список всех возможных ошибок"),
		q3("m-q-budget", "Для чего используют error budget?", "Балансируют скорость изменений и допустимую ненадёжность по SLO", "Распределяют зарплаты команды", "Заменяют мониторинг"),
		q3("m-q-observe", "Какой набор наиболее полезен для диагностики сервиса?", "Метрики, структурированные логи и трассировки с общими идентификаторами", "Только fmt.Println без контекста", "Снимок исходного кода в каждом ответе"),
		q3("m-q-rollback", "Когда rollback безопаснее включения feature flag?", "Когда схема и состояние совместимы с предыдущей версией и rollback проверен", "Всегда после любого deploy", "Только если удалены все данные"),
		q3("m-q-p3", "Что подтверждает машинный результат P3 проекта?", "Только конкретные автоматические проверки зафиксированной версии артефакта", "Авторство и самостоятельность ученика", "Полную готовность к должности Middle"),
	}
}

func middlePlusQuestions() []authoredQuestion {
	return []authoredQuestion{
		q3("mp-q-benchmark", "Почему benchmark нужно запускать несколько раз?", "Чтобы увидеть разброс и снизить влияние случайного шума", "Чтобы каждый запуск автоматически ускорял код", "Чтобы заменить correctness tests"),
		q3("mp-q-baseline", "Как обеспечить честное сравнение candidate с baseline?", "Одинаковыми данными, окружением, нагрузкой и критерием корректности", "Запустить каждый вариант в разных условиях", "Измерять только лучший candidate запуск"),
		q3("mp-q-alloc", "Что показывает число allocs/op?", "Количество heap allocations на операцию в benchmark", "Число запросов в секунду", "Размер бинарного файла"),
		q3("mp-q-profile", "Как использовать CPU profile?", "Найти горячие участки под репрезентативной нагрузкой и проверить гипотезу", "Доказать отсутствие всех bottleneck", "Выбрать функцию с самым длинным именем"),
		q3("mp-q-pareto", "Что означает trade-off latency/throughput?", "Улучшение одного показателя может ухудшить другой при заданной нагрузке", "Оба показателя всегда растут вместе", "Latency и throughput — одно и то же"),
		q3("mp-q-tail", "Почему p99 latency важна для пользователя?", "Среднее скрывает медленный хвост, затрагивающий часть запросов", "p99 всегда равно максимальному времени", "p99 измеряет только ошибки компиляции"),
		q3("mp-q-load", "Что нужно зафиксировать в нагрузочном тесте?", "Профиль нагрузки, длительность, окружение и критерий успешности", "Только название ветки Git", "Только среднее число goroutine"),
		q3("mp-q-cache", "Какой риск добавляет cache-aside?", "Устаревшие данные и сложность согласования инвалидации", "Исчезновение всех сетевых отказов", "Гарантированное отсутствие гонок"),
		q3("mp-q-queue", "Что ограниченная очередь делает при перегрузке?", "Позволяет применить backpressure или явный отказ", "Неограниченно потребляет память", "Гарантирует обработку до перезапуска"),
		q3("mp-q-retry", "Как избежать retry amplification?", "Ограничить попытки, применять backoff/jitter и учитывать общий бюджет", "Повторять запрос немедленно без лимита", "Повторять все ошибки одинаково"),
		q3("mp-q-circuit", "Что делает circuit breaker в open state?", "Временно отклоняет вызовы к нездоровой зависимости", "Повторяет каждую операцию быстрее", "Удаляет зависимость из архитектуры"),
		q3("mp-q-chaos", "Как безопасно провести fault injection?", "Задать ограниченный blast radius, критерии остановки и наблюдаемость", "Отключить production без уведомления", "Скрыть отказ от операторов"),
		q3("mp-q-capacity", "Как оценить запас capacity?", "Проверить ожидаемый пик и деградацию при отказе компонента", "Считать текущую среднюю загрузку достаточной", "Умножить CPU на число разработчиков"),
		q3("mp-q-p4", "Что машинные проверки P4 способны подтвердить?", "Корректность заданных тестов и воспроизводимых измерений", "Что выбранная оптимизация лучше для всех нагрузок", "Что автор лично проводил benchmark"),
		q3("mp-q-evidence", "Почему сохраняют raw benchmark output и версию окружения?", "Чтобы сравнение было проверяемым и воспроизводимым", "Чтобы заменить описание методики", "Чтобы скрыть неудачные запуски"),
	}
}

func seniorQuestions() []authoredQuestion {
	return []authoredQuestion{
		q3("s-q-boundary", "Как выбрать service boundary?", "По связности изменений, владению данными и ясным контрактам", "По числу каталогов в репозитории", "Разбить каждый пакет в отдельный сервис"),
		q3("s-q-consistency", "Когда eventual consistency допустима?", "Когда продукт явно допускает задержку и определены reconciliation и пользовательские последствия", "Всегда при использовании микросервисов", "Когда нет тестов"),
		q3("s-q-outbox", "Как transactional outbox уменьшает dual-write риск?", "Записывает бизнес-изменение и событие в одной транзакции", "Гарантирует доставку ровно один раз без дедупликации", "Заменяет брокер сообщений"),
		q3("s-q-saga", "Что такое compensation в saga?", "Явное действие, компенсирующее эффект ранее завершённого шага", "Глобальный rollback распределённой транзакции", "Повтор запроса без знания результата"),
		q3("s-q-threat", "С чего начинается threat model?", "С активов, границ доверия, акторов и правдоподобных угроз", "С выбора библиотеки шифрования", "С заполнения списка CVE наугад"),
		q3("s-q-least", "Что означает least privilege?", "Субъект получает только необходимые полномочия на ограниченное время", "Все сервисы используют один admin token", "Доступ не пересматривают после выдачи"),
		q3("s-q-compat", "Как безопасно менять публичный API?", "Сначала обеспечить совместимость или согласованный период миграции", "Удалить старое поле сразу после локальной сборки", "Менять контракт без версии и наблюдения"),
		q3("s-q-migration", "Как снизить риск миграции большой таблицы?", "Использовать expand-migrate-contract с проверяемыми этапами", "Заблокировать таблицу на неизвестный срок", "Удалить старый столбец до переноса данных"),
		q3("s-q-slo", "Что должен задавать service-level objective?", "Измеримый target, окно и событие, которое считается хорошим", "Общее пожелание быть быстрыми", "Абсолютное обещание нулевых отказов"),
		q3("s-q-incident", "Как вести инцидент с несколькими командами?", "Назначить incident lead, вести timeline и явные владельцы действий", "Параллельно менять production без координации", "Искать виновного до восстановления"),
		q3("s-q-blameless", "Что означает blameless postmortem?", "Исследовать системные условия и улучшения, сохраняя ответственность за действия", "Не записывать ошибки и последствия", "Не назначать владельцев исправлений"),
		q3("s-q-adr", "Что фиксирует ADR?", "Контекст, решение, альтернативы и последствия архитектурного выбора", "Полную документацию каждого файла", "Результат любого автоматического теста"),
		q3("s-q-degrade", "Что такое graceful degradation?", "Сервис сохраняет ключевую функцию при ограниченной потере второстепенных возможностей", "Сервис скрывает любой отказ", "Все зависимости объявляются необязательными"),
		q3("s-q-p5", "Что доказывает успешный запуск P5 prototype?", "Только прохождение машинных прототипных проверок", "Независимое архитектурное одобрение и авторство", "Присвоение ранга Senior"),
		q3("s-q-review", "Какой review считается независимым для capstone?", "Аутентифицированный reviewer оценивает точный hash артефакта и оставляет решение", "Автор сам утверждает собственную работу", "LLM выдаёт pass без сохранения evidence"),
	}
}
