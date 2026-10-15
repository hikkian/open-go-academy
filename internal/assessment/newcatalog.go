package assessment

import (
	"fmt"
	"sort"

	"academy/internal/grading"
)

// Trusted references are compiled into the assessment policy; learner code is
// always submitted separately to the isolated runner.
var assessmentReferences = map[string]string{
	"junior-exam-count-transitions": `package main
func CountTransitions(values []int) int { count:=0; for i:=1;i<len(values);i++ { if values[i]!=values[i-1] { count++ } }; return count }`,
	"junior-exam-clamp-range": `package main
func ClampRange(value, low, high int) int { if low>high { return low }; if value<low { return low }; if value>high { return high }; return value }`,
	"junior-exam-first-unique-rune": `package main
func FirstUniqueRune(text string) string { runes:=[]rune(text); for i,r:=range runes { found:=false; for j,s:=range runes { if i!=j && r==s { found=true; break } }; if !found { return string(r) } }; return "" }`,
	"juniorplus-exam-safe-redirect": `package main
import ("net/url"; "strings"; "unicode/utf8")
func SafeRedirectPath(path string) bool {
 if !utf8.ValidString(path) || path == "" || path[0] != '/' || strings.ContainsAny(path, "?#") { return false }
 decoded, err := url.PathUnescape(path)
 if err != nil || !utf8.ValidString(decoded) || len(decoded) == 0 || decoded[0] != '/' || strings.HasPrefix(decoded, "//") || strings.ContainsAny(decoded, "%?#") { return false }
 for _, r := range decoded { if r <= 0x20 || r == 0x7f || r == '\\' { return false } }
 for _, segment := range strings.Split(decoded, "/") { if segment == "." || segment == ".." { return false } }
 return true
}`,
	"juniorplus-exam-retry-status": `package main
func RetryableStatus(status int) bool { return status==429 || (status>=500 && status<=599) }`,
	"juniorplus-exam-access-policy": `package main
func CanAccessResource(ownerID, actorID int, role string) bool {
 if ownerID <= 0 || actorID <= 0 { return false }
 return role == "admin" || (role == "user" && actorID == ownerID)
}`,
}

// All returns the immutable assessment versions in learner-path order.
func All() []*Exam { return []*Exam{Load(), loadJunior(), loadJuniorPlus()} }

// Find resolves an exact immutable version, or the current version when empty.
func Find(id, version string) (*Exam, bool) {
	for _, e := range All() {
		if e.ID == id && (version == "" || e.Version == version) {
			return e, true
		}
	}
	return nil, false
}

func FunctionSpecForTask(exam *Exam, taskID string) *grading.Spec { return exam.FunctionSpec(taskID) }

type authoredQuestion struct{ id, prompt, answer, wrong1, wrong2 string }

func buildExam(id, title, rank, required, description string, prerequisites, criteria []string, bank []authoredQuestion, taskDefs []Task, specIDs map[string]string) *Exam {
	e := &Exam{ID: id, Version: "1.0.0", Title: title, Rank: rank, RequiredRank: required, Description: description, EstimatedMinutes: 90, Prerequisites: prerequisites, Criteria: criteria, correctIDs: map[string]string{}, cases: map[string][]Case{}, drivers: map[string]string{}, functionSpecs: map[string]*grading.Spec{}}
	for _, item := range bank {
		q := Question{ID: item.id, Prompt: item.prompt, Options: []Option{{ID: "a", Text: item.answer}, {ID: "b", Text: item.wrong1}, {ID: "c", Text: item.wrong2}}}
		shift := len(e.Questions) % 3
		if shift > 0 {
			q.Options = append(append([]Option(nil), q.Options[shift:]...), q.Options[:shift]...)
		}
		for i := range q.Options {
			q.Options[i].ID = string(rune('a' + i))
			if q.Options[i].Text == item.answer {
				e.correctIDs[q.ID] = q.Options[i].ID
			}
		}
		e.Questions = append(e.Questions, q)
	}
	e.Tasks = taskDefs
	ids := make([]string, 0, len(specIDs))
	taskForSpec := map[string]string{}
	for taskID, specID := range specIDs {
		ids = append(ids, specID)
		taskForSpec[specID] = taskID
	}
	specs, err := grading.LoadFor(ids)
	if err != nil {
		panic(fmt.Errorf("load %s assessment policies: %w", id, err))
	}
	for specID, spec := range specs {
		ref, ok := assessmentReferences[specID]
		if !ok {
			panic("missing trusted assessment reference: " + specID)
		}
		if err = spec.BindReference(ref); err != nil {
			panic(fmt.Errorf("bind %s: %w", specID, err))
		}
		e.functionSpecs[taskForSpec[specID]] = spec
	}
	sort.Slice(e.Tasks, func(i, j int) bool { return e.Tasks[i].ID < e.Tasks[j].ID })
	return e
}

func q3(id, prompt, answer, w1, w2 string) authoredQuestion {
	return authoredQuestion{id, prompt, answer, w1, w2}
}

func loadJunior() *Exam {
	return buildExam("junior-core", "Экзамен Junior: основы Go", "Junior", "Zero", "Версия критериев Go Academy 1.0.0 проверяет основы Go, структуры данных, ошибки, пакеты, тестирование и отладку. Автоматический учебный критерий не подтверждает профессию, стаж или ручное качество P1.", juniorPrerequisites(), []string{"Все упражнения Zero и Junior", "Подтверждён предыдущий ранг Zero", "Автоматические проверки проекта P1", "Ручное качество проекта не подтверждается", "Все 20 вопросов экзамена", "Все проверки трёх функций Go"}, juniorQuestions(), juniorTasks(), map[string]string{"transitions": "junior-exam-count-transitions", "range": "junior-exam-clamp-range", "unicode": "junior-exam-first-unique-rune"})
}

func loadJuniorPlus() *Exam {
	return buildExam("juniorplus-api", "Экзамен Junior+: API и данные", "Junior+", "Junior", "Версия критериев Go Academy 1.0.0 проверяет HTTP-контракты, SQL-границы, авторизацию и работу с контекстом. Автоматический учебный критерий не подтверждает профессию, стаж или ручное качество P2.", juniorPlusPrerequisites(), []string{"Все упражнения Junior+", "Подтверждён предыдущий ранг Junior", "Автопроверка P2 с PostgreSQL", "Ручное качество проекта не подтверждается", "Все 20 вопросов экзамена", "Все проверки функций API и доступа"}, juniorPlusQuestions(), juniorPlusTasks(), map[string]string{"redirect": "juniorplus-exam-safe-redirect", "retry": "juniorplus-exam-retry-status", "access": "juniorplus-exam-access-policy"})
}

func juniorTasks() []Task {
	return []Task{
		{ID: "transitions", Title: "Подсчитайте смены значения", Kind: "function", Description: "Реализуйте CountTransitions(values []int) int: посчитайте соседние пары с разными значениями. Пустой и одноэлементный срез дают 0; вход нельзя менять.", StarterCode: "package main\n\nfunc CountTransitions(values []int) int { return 0 }\n", PublicExamples: []Example{{"[]", "0"}, {"[1,2,2,5,1]", "3"}}},
		{ID: "range", Title: "Ограничьте значение диапазоном", Kind: "debugging", Description: "Исправьте ClampRange(value, low, high int) int. При low > high верните low; иначе ограничьте значение включительными границами.", StarterCode: "package main\n\nfunc ClampRange(value, low, high int) int {\n if low > high { return low }; if value < low { return low }; if value > high { return value } // BUG: должно вернуть high\n return value\n}\n", PublicExamples: []Example{{"4,0,9", "4"}, {"-2,0,9", "0"}}},
		{ID: "unicode", Title: "Найдите первую уникальную руну", Kind: "function", Description: "Реализуйте FirstUniqueRune(text string) string: верните первую руну, встречающуюся ровно один раз, или пустую строку. Позиции считаются в рунах, не байтах.", StarterCode: "package main\n\nfunc FirstUniqueRune(text string) string { return \"\" }\n", PublicExamples: []Example{{"aabbc", "c"}, {"🙂a🙂b", "a"}}},
	}
}
func juniorPlusTasks() []Task {
	return []Task{
		{ID: "redirect", Title: "Проверьте локальный путь перенаправления", Kind: "function", Description: "Реализуйте SafeRedirectPath(path string) bool. Разрешайте только валидный UTF-8 локальный абсолютный PATH (корень / допустим), без query и fragment. Однократно декодируйте percent-encoding; запрещайте некорректное или вложенное кодирование, недопустимый UTF-8, ASCII-пробелы и управляющие символы, обратный слеш, ведущий // и сегменты . или ...", StarterCode: "package main\n\nfunc SafeRedirectPath(path string) bool { return false }\n", PublicExamples: []Example{{"/docs", "true"}, {"/", "true"}}},
		{ID: "retry", Title: "Исправьте границы повторяемого HTTP-ответа", Kind: "debugging", Description: "Исправьте RetryableStatus(status int) bool: повтор допустим для 429 и статусов 500..599 включительно; остальные ответы не повторяйте автоматически.", StarterCode: "package main\n\nfunc RetryableStatus(status int) bool { return status >= 500 && status < 599 } // BUG: пропущен 429 и 599\n", PublicExamples: []Example{{"200", "false"}, {"503", "true"}}},
		{ID: "access", Title: "Проверьте право на ресурс", Kind: "function", Description: "Реализуйте CanAccessResource(ownerID, actorID int, role string) bool. Роль приходит из проверенного серверного principal и может быть только user или admin; unknown role запрещена. Сначала требуйте ownerID > 0 и actorID > 0. Затем разрешайте admin либо совпадение actorID с ownerID для user.", StarterCode: "package main\n\nfunc CanAccessResource(ownerID, actorID int, role string) bool { return role == \"admin\" || actorID == ownerID } // BUG: нет проверки положительных ID\n", PublicExamples: []Example{{"4,4,user", "true"}, {"4,9,user", "false"}}},
	}
}

func juniorQuestions() []authoredQuestion {
	return []authoredQuestion{
		q3("j-q-slice", "Что хранит заголовок среза?", "Ссылку на массив, длину и ёмкость", "Только копию всех элементов", "Только адрес функции append"),
		q3("j-q-append", "Что гарантирует append?", "Возвращает новый заголовок среза; базовый массив может измениться", "Всегда меняет исходную переменную без присваивания", "Всегда выделяет новый массив"),
		q3("j-q-map", "Как безопасно проверить наличие ключа в map?", "Использовать value, ok := m[key]", "Сравнить m[key] с nil для любого типа", "Перебрать map до нужного ключа"),
		q3("j-q-order", "Каков порядок обхода map в Go?", "Он не гарантирован", "Всегда по возрастанию ключа", "Всегда в порядке вставки"),
		q3("j-q-rune", "Что обычно обозначает rune?", "Кодовую точку Unicode", "Один байт UTF-8", "Размер строки в байтах"),
		q3("j-q-struct-copy", "Что происходит при присваивании struct?", "Копируются поля значения; указатели внутри остаются общими", "Всегда копируется весь граф объектов", "Новая переменная становится алиасом struct"),
		q3("j-q-pointer", "Что означает nil у указателя?", "Он не указывает на значение", "Он указывает на нулевую область памяти", "Он равен любому указателю на zero value"),
		q3("j-q-method", "Как объявляется method для типа?", "Функция с receiver перед именем", "Функция внутри интерфейса", "Функция с обязательным ключевым словом method"),
		q3("j-q-interface", "Когда тип удовлетворяет интерфейсу?", "Когда реализует все его методы; объявлять это отдельно не нужно", "Когда явно перечислен в интерфейсе", "Только через наследование"),
		q3("j-q-error", "Как обычно проверяют ошибку из функции?", "Сравнивают возвращённое значение error с nil", "Проверяют глобальную переменную lastError", "Перехватывают исключение"),
		q3("j-q-wrap", "Зачем используют %w в fmt.Errorf?", "Чтобы обернуть ошибку и сохранить цепочку для errors.Is/As", "Чтобы скрыть ошибку навсегда", "Чтобы сериализовать ошибку в JSON"),
		q3("j-q-package", "Что обычно объединяет Go package?", "Файлы одного пакета в каталоге", "Все файлы репозитория независимо от каталога", "Только один файл с main"),
		q3("j-q-export", "Как экспортируется имя из пакета?", "Начинается с заглавной буквы", "Начинается с подчёркивания", "Объявляется через export"),
		q3("j-q-module", "Что описывает go.mod?", "Модуль и его зависимости", "Только команды сборки", "Список функций пакета"),
		q3("j-q-test", "Где обычно размещают тесты пакета?", "В файлах *_test.go", "Только в каталоге tests обязательным образом", "Внутри go.mod"),
		q3("j-q-tabletest", "Зачем делают table-driven tests?", "Проверять много входов одним повторно используемым сценарием", "Запустить тесты параллельно без синхронизации", "Заменить assertions выводом в консоль"),
		q3("j-q-race", "Что обнаруживает go test -race?", "Некоторые конкурентные обращения к памяти без синхронизации", "Все логические ошибки", "Медленные SQL-запросы"),
		q3("j-q-debug", "Что полезно проверить при ошибке индекса?", "Границы len и индекс перед обращением", "Только capacity после обращения", "Размер исполняемого файла"),
		q3("j-q-complexity", "Какой рост у линейного прохода по n элементам?", "O(n)", "O(1) всегда", "O(n²) всегда"),
		q3("j-q-generic", "Что задаёт constraint параметра типа?", "Множество допустимых типов и доступных операций", "Значение переменной при запуске", "Порядок компиляции пакетов"),
	}
}

func juniorPlusQuestions() []authoredQuestion {
	return []authoredQuestion{
		q3("jp-q-http", "Что содержит HTTP request?", "Метод, URL, заголовки и при необходимости тело", "Только URL", "Готовую SQL-транзакцию"),
		q3("jp-q-status", "Что сообщает HTTP status code?", "Класс и результат обработки запроса", "Версию JSON схемы", "Количество байтов в ответе"),
		q3("jp-q-method", "Какое свойство обычно ожидают от GET?", "Он читает ресурс и не должен менять его состояние", "Он всегда создаёт новую запись", "Он безопасно передаёт пароль в URL"),
		q3("jp-q-tcp", "Что даёт TCP соединение?", "Упорядоченный надёжный поток байтов между endpoints", "Гарантию успешного HTTP ответа", "Автоматическое шифрование любого трафика"),
		q3("jp-q-dns", "Что делает DNS в типичном запросе?", "Находит сетевой адрес по имени узла", "Шифрует HTTP тело", "Выбирает SQL индекс"),
		q3("jp-q-context", "Зачем handler использует request context?", "Чтобы видеть отмену и deadline исходного запроса", "Чтобы хранить постоянный профиль", "Чтобы продлевать запрос после отключения клиента"),
		q3("jp-q-timeout", "Что полезно задать HTTP клиенту?", "Timeout или context deadline", "Бесконечное ожидание по умолчанию", "Повторять любой запрос без лимита"),
		q3("jp-q-json", "Почему ограничивают размер JSON тела?", "Чтобы ограничить память и объём недоверенного ввода", "Чтобы изменить HTTP метод", "Чтобы включить gzip автоматически"),
		q3("jp-q-validation", "Где валидировать данные запроса?", "На границе API до выполнения побочных действий", "Только в браузере", "После записи в БД"),
		q3("jp-q-sql", "Почему параметризуют SQL?", "Чтобы отделить значения от синтаксиса SQL и снизить риск injection", "Чтобы запрос всегда выполнялся быстрее", "Чтобы избежать транзакций"),
		q3("jp-q-transaction", "Что даёт транзакция?", "Группирует операции с согласованными commit/rollback границами", "Шифрует таблицы", "Гарантирует отсутствие всех блокировок"),
		q3("jp-q-unique", "Как защитить уникальность при конкурирующих запросах?", "Ограничением UNIQUE в базе и обработкой конфликта", "Проверкой SELECT только в приложении", "Блокировкой кнопки в UI"),
		q3("jp-q-authn", "Что отвечает на вопрос authentication?", "Кто выполняет запрос", "Какие действия разрешены", "Сколько строк вернул SQL"),
		q3("jp-q-authz", "Что отвечает на вопрос authorization?", "Разрешено ли субъекту это действие над ресурсом", "Какой пароль ввёл субъект", "Где расположен сервер"),
		q3("jp-q-session", "Как безопаснее обращаться с session token?", "Считать его секретом и не писать в логи", "Добавлять его в URL для удобства", "Хранить без срока действия"),
		q3("jp-q-ownership", "Где надёжнее проверить владельца ресурса?", "В серверном запросе/политике на каждом доступе", "Только скрыть кнопку на клиенте", "Проверить лишь при создании"),
		q3("jp-q-migration", "Что должна делать миграция схемы?", "Версионированно и воспроизводимо менять структуру БД", "Заменять резервное копирование", "Изменять данные без журнала"),
		q3("jp-q-pool", "Зачем ограничивать пул DB соединений?", "Чтобы контролировать конкуренцию и нагрузку на базу", "Чтобы гарантировать один запрос за всё время", "Чтобы хранить HTTP cookies"),
		q3("jp-q-redirect", "Почему опасен redirect на произвольный URL из ввода?", "Можно перенаправить пользователя на недоверенный внешний узел", "Он ломает DNS кэш всегда", "Он превращает GET в SQL"),
		q3("jp-q-retry", "Что учитывать перед повтором запроса?", "Код ответа, идемпотентность и ограничение числа попыток", "Только цвет кнопки", "Повторять любой POST бесконечно"),
	}
}

func juniorPrerequisites() []string {
	return []string{
		"hello", "repair", "two-lines", "zero-02-easy",
		"zero-02-medium", "zero-02-hard", "zero-03-easy", "zero-03-medium",
		"zero-03-hard", "zero-04-easy", "zero-04-medium", "zero-04-hard",
		"zero-05-easy", "zero-05-medium", "zero-05-hard", "zero-06-easy",
		"zero-06-medium", "zero-06-hard", "zero-07-easy", "zero-07-medium",
		"zero-07-hard", "zero-08-easy", "zero-08-medium", "zero-08-hard",
		"junior-01-easy", "junior-01-medium", "junior-01-hard", "junior-02-easy",
		"junior-02-medium", "junior-02-hard", "junior-03-easy", "junior-03-medium",
		"junior-03-hard", "junior-04-easy", "junior-04-medium", "junior-04-hard",
		"junior-05-easy", "junior-05-medium", "junior-05-hard", "junior-06-easy",
		"junior-06-medium", "junior-06-hard", "junior-07-easy", "junior-07-medium",
		"junior-07-hard", "junior-08-easy", "junior-08-medium", "junior-08-hard",
		"junior-09-easy", "junior-09-medium", "junior-09-hard", "junior-10-easy",
		"junior-10-medium", "junior-10-hard", "junior-11-easy", "junior-11-medium",
		"junior-11-hard", "junior-12-easy", "junior-12-medium", "junior-12-hard",
		"p1:automatic",
	}
}

func juniorPlusPrerequisites() []string {
	return []string{
		"hello", "repair", "two-lines", "zero-02-easy",
		"zero-02-medium", "zero-02-hard", "zero-03-easy", "zero-03-medium",
		"zero-03-hard", "zero-04-easy", "zero-04-medium", "zero-04-hard",
		"zero-05-easy", "zero-05-medium", "zero-05-hard", "zero-06-easy",
		"zero-06-medium", "zero-06-hard", "zero-07-easy", "zero-07-medium",
		"zero-07-hard", "zero-08-easy", "zero-08-medium", "zero-08-hard",
		"junior-01-easy", "junior-01-medium", "junior-01-hard", "junior-02-easy",
		"junior-02-medium", "junior-02-hard", "junior-03-easy", "junior-03-medium",
		"junior-03-hard", "junior-04-easy", "junior-04-medium", "junior-04-hard",
		"junior-05-easy", "junior-05-medium", "junior-05-hard", "junior-06-easy",
		"junior-06-medium", "junior-06-hard", "junior-07-easy", "junior-07-medium",
		"junior-07-hard", "junior-08-easy", "junior-08-medium", "junior-08-hard",
		"junior-09-easy", "junior-09-medium", "junior-09-hard", "junior-10-easy",
		"junior-10-medium", "junior-10-hard", "junior-11-easy", "junior-11-medium",
		"junior-11-hard", "junior-12-easy", "junior-12-medium", "junior-12-hard",
		"juniorplus-01-easy", "juniorplus-01-medium", "juniorplus-01-hard", "juniorplus-02-easy",
		"juniorplus-02-medium", "juniorplus-02-hard", "juniorplus-03-easy", "juniorplus-03-medium",
		"juniorplus-03-hard", "juniorplus-04-easy", "juniorplus-04-medium", "juniorplus-04-hard",
		"juniorplus-05-easy", "juniorplus-05-medium", "juniorplus-05-hard", "juniorplus-06-easy",
		"juniorplus-06-medium", "juniorplus-06-hard", "juniorplus-07-easy", "juniorplus-07-medium",
		"juniorplus-07-hard", "juniorplus-08-easy", "juniorplus-08-medium", "juniorplus-08-hard",
		"juniorplus-09-easy", "juniorplus-09-medium", "juniorplus-09-hard", "juniorplus-10-easy",
		"juniorplus-10-medium", "juniorplus-10-hard", "juniorplus-11-easy", "juniorplus-11-medium",
		"juniorplus-11-hard", "juniorplus-12-easy", "juniorplus-12-medium", "juniorplus-12-hard",
		"juniorplus-13-easy", "juniorplus-13-medium", "juniorplus-13-hard", "juniorplus-14-easy",
		"juniorplus-14-medium", "juniorplus-14-hard", "juniorplus-15-easy", "juniorplus-15-medium",
		"juniorplus-15-hard", "juniorplus-16-easy", "juniorplus-16-medium", "juniorplus-16-hard",
		"juniorplus-17-easy", "juniorplus-17-medium", "juniorplus-17-hard", "juniorplus-18-easy",
		"juniorplus-18-medium", "juniorplus-18-hard", "juniorplus-19-easy", "juniorplus-19-medium",
		"juniorplus-19-hard", "juniorplus-20-easy", "juniorplus-20-medium", "juniorplus-20-hard",
		"p1:automatic", "p2:automatic",
	}
}
