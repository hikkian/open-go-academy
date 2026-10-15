// Package assessment defines the versioned, local Zero knowledge exam.
// It only evaluates trusted quiz answers and source structure; user code must
// be sent to the isolated runner by its caller.
package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

const examVersion = "1.0.0"

type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Question struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Options []Option `json:"options"`
}
type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}
type Task struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Kind           string    `json:"kind"`
	Description    string    `json:"description"`
	StarterCode    string    `json:"starter_code"`
	PublicExamples []Example `json:"public_examples"`
}

// Exam is the public DTO. Private keys, hidden vectors and trusted drivers are
// deliberately kept out of its serialized fields.
type Exam struct {
	ID               string     `json:"id"`
	Version          string     `json:"version"`
	Title            string     `json:"title"`
	Rank             string     `json:"rank"`
	Description      string     `json:"description"`
	EstimatedMinutes int        `json:"estimated_minutes"`
	Prerequisites    []string   `json:"prerequisites"`
	Criteria         []string   `json:"criteria"`
	Questions        []Question `json:"questions"`
	Tasks            []Task     `json:"tasks"`
	correctIDs       map[string]string
	cases            map[string][]Case
	drivers          map[string]string
}
type Check struct {
	Detail string `json:"detail,omitempty"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}
type Case struct {
	Args     []string
	Expected string
}

func Load() *Exam {
	e := &Exam{
		ID: "zero-core", Version: examVersion, Title: "Проверка основ Zero",
		Rank: "Zero", EstimatedMinutes: 35,
		Description:   "Экзамен проверяет определённые базовые критерии Zero. Часть учебных материалов после первого дня ещё публикуется; знакомый с Go ученик может подтвердить все перечисленные здесь критерии напрямую после выполнения prerequisite-упражнений. Результат — локальный учебный ранг Zero по criteria v1.0.0, без прокторинга, проверки личности или подтверждения профессионального стажа. Экзамен не присваивает Junior.",
		Prerequisites: []string{"hello", "repair", "two-lines"},
		Criteria:      []string{"Верно ответить на все 12 вопросов", "Решить coding и debugging задачи по всем проверкам"},
		Questions: []Question{
			{"q-computer", "Что делает процессор (CPU — центральный процессор) во время работы программы?", []Option{{"a", "Выполняет команды и вычисления"}, {"b", "Хранит файлы после выключения"}, {"c", "Показывает изображение на мониторе"}}},
			{"q-ram", "Для чего компьютеру нужна оперативная память (RAM)?", []Option{{"a", "Для временных данных работающих программ"}, {"b", "Для постоянного хранения файлов после выключения"}, {"c", "Для подключения к интернету"}}},
			{"q-files", "Что обычно обозначает файл с именем `main.go`?", []Option{{"a", "Файл с исходным кодом Go"}, {"b", "Исполняемый файл Windows"}, {"c", "Папка операционной системы"}}},
			{"q-os", "Какова основная роль операционной системы (OS)?", []Option{{"a", "Управлять ресурсами компьютера и запуском программ"}, {"b", "Переводить любой текст на английский"}, {"c", "Заменять все приложения"}}},
			{"q-terminal", "Что делает команда `cd lessons` в терминале?", []Option{{"a", "Переходит в папку lessons"}, {"b", "Удаляет папку lessons"}, {"c", "Компилирует Go-файл"}}},
			{"q-source", "Что такое исходный код?", []Option{{"a", "Инструкции программы, записанные человеком"}, {"b", "Оперативная память компьютера"}, {"c", "Результат работы монитора"}}},
			{"q-compiler", "Какова задача компилятора Go?", []Option{{"a", "Проверить и перевести исходный код в программу"}, {"b", "Создать резервную копию всех файлов"}, {"c", "Запустить операционную систему"}}},
			{"q-types", "Какой тип в Go подходит для значения `42` без дробной части?", []Option{{"a", "int"}, {"b", "bool"}, {"c", "string"}}},
			{"q-if", "Когда выполняется тело ветки `if score >= 10`?", []Option{{"a", "Когда score не меньше 10"}, {"b", "Всегда ровно десять раз"}, {"c", "Только когда score равен нулю"}}},
			{"q-for-function", "Какое утверждение о `for` и функции в Go верно?", []Option{{"a", "for повторяет действия, функция объединяет команды и может вернуть значение"}, {"b", "for объявляет файл, функция выключает программу"}, {"c", "for и функция являются типами чисел"}}},
			{"q-byte", "Сколько бит обычно составляют один байт?", []Option{{"a", "8 бит"}, {"b", "2 бита"}, {"c", "100 бит"}}},
			{"q-go-command", "Какой командой из папки Go-проекта можно скомпилировать и запустить пакет main?", []Option{{"a", "go run ."}, {"b", "cd ."}, {"c", "fmt.Println(.)"}}},
		},
		correctIDs: map[string]string{},
		cases: map[string][]Case{
			"coding":    {{[]string{"0"}, "0"}, {[]string{"1"}, "0"}, {[]string{"2"}, "2"}, {[]string{"6"}, "12"}, {[]string{"27"}, "182"}, {[]string{"-4"}, "0"}},
			"debugging": {{[]string{"-8"}, "0"}, {[]string{"0"}, "0"}, {[]string{"1"}, "1"}, {[]string{"99"}, "99"}, {[]string{"100"}, "100"}, {[]string{"101"}, "100"}},
		},
		drivers: map[string]string{"coding": driver("sumEven"), "debugging": driver("clampPercent")},
	}
	e.Tasks = []Task{
		{ID: "coding", Title: "Сложи чётные числа", Kind: "coding", Description: "Реализуйте `sumEven(limit int) int`: верните сумму всех чётных целых от 0 до limit включительно. При limit меньше нуля верните 0. Используйте `package main`, функции и при необходимости константы; без import, init, глобальных переменных и типов. Доверенный экзаменационный драйвер сам читает ввод и печатает ответ, поэтому решение остаётся чистой функцией.", StarterCode: "package main\n\nfunc sumEven(limit int) int {\n\t// TODO: sum even numbers from 0 through limit\n\treturn 0\n}\n", PublicExamples: []Example{{"6", "12"}, {"27", "182"}}},
		{ID: "debugging", Title: "Исправь ограничение процента", Kind: "debugging", Description: "Исправьте `clampPercent(value int) int`, чтобы результат всегда был в диапазоне 0..100: отрицательные значения становятся 0, значения выше 100 становятся 100, остальные сохраняются. Используйте `package main`, функции и при необходимости константы; без import, init, глобальных переменных и типов. Доверенный экзаменационный драйвер сам читает ввод и печатает ответ, поэтому решение остаётся чистой функцией.", StarterCode: "package main\n\nfunc clampPercent(value int) int {\n\tif value < 0 {\n\t\treturn value // BUG: negative values must become 0\n\t}\n\tif value > 100 {\n\t\treturn value // BUG: values above 100 must become 100\n\t}\n\treturn value\n}\n", PublicExamples: []Example{{"-4", "0"}, {"101", "100"}}},
	}
	// Rotate the choices and assign IDs by displayed position. Keep the answer
	// key derived from the matching answer text, never from a fixed option ID.
	correctText := map[string]string{
		"q-computer":     "Выполняет команды и вычисления",
		"q-ram":          "Для временных данных работающих программ",
		"q-files":        "Файл с исходным кодом Go",
		"q-os":           "Управлять ресурсами компьютера и запуском программ",
		"q-terminal":     "Переходит в папку lessons",
		"q-source":       "Инструкции программы, записанные человеком",
		"q-compiler":     "Проверить и перевести исходный код в программу",
		"q-types":        "int",
		"q-if":           "Когда score не меньше 10",
		"q-for-function": "for повторяет действия, функция объединяет команды и может вернуть значение",
		"q-byte":         "8 бит",
		"q-go-command":   "go run .",
	}
	for i := range e.Questions {
		shift := i % 3
		q := &e.Questions[i]
		if shift != 0 {
			q.Options = append(append([]Option(nil), q.Options[shift:]...), q.Options[:shift]...)
		}
		for j := range q.Options {
			q.Options[j].ID = string(rune('a' + j))
			if q.Options[j].Text == correctText[q.ID] {
				e.correctIDs[q.ID] = q.Options[j].ID
			}
		}
	}
	return e
}

func driver(fn string) string {
	return "package main\nimport (\"fmt\"; \"os\"; \"strconv\")\nfunc main(){ n,err:=strconv.Atoi(os.Args[1]); if err!=nil { panic(err) }; fmt.Println(" + fn + "(n)) }\n"
}
func (e *Exam) QuizChecks(answers map[string]string) []Check {
	out := make([]Check, 0, len(e.Questions))
	for _, q := range e.Questions {
		out = append(out, Check{ID: q.ID, Name: q.Prompt, Passed: answers[q.ID] == e.correctIDs[q.ID]})
	}
	return out
}
func (e *Exam) TaskCases(taskID string) []Case {
	cs := e.cases[taskID]
	out := make([]Case, len(cs))
	for i, c := range cs {
		out[i] = Case{Args: append([]string(nil), c.Args...), Expected: c.Expected}
	}
	return out
}
func (e *Exam) Driver(taskID string) string { return e.drivers[taskID] }

// ValidateTaskSource applies structural limits only. Compilation and running
// must happen in the dedicated isolated runner, never in this process.
func (e *Exam) ValidateTaskSource(taskID, source string) error {
	if taskID != "coding" && taskID != "debugging" {
		return errors.New("unknown assessment task")
	}
	if len(source)+len(e.Driver(taskID)) > 64*1024 {
		return errors.New("source and trusted driver exceed 64 KiB")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "solution.go", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse Go source: %w", err)
	}
	if file.Name.Name != "main" {
		return errors.New("source must use package main")
	}
	if len(file.Imports) > 0 {
		return errors.New("source must not import packages; the trusted driver handles input and output")
	}
	want := map[string]bool{"sumEven": taskID == "coding", "clampPercent": taskID == "debugging"}
	found := false
	for _, d := range file.Decls {
		switch declaration := d.(type) {
		case *ast.GenDecl:
			if declaration.Tok != token.CONST {
				return errors.New("source may declare constants, but not imports, global variables, or types")
			}
		case *ast.FuncDecl:
			fn := declaration
			if fn.Recv != nil {
				return errors.New("source must not define methods")
			}
			if fn.Name.Name == "main" || fn.Name.Name == "init" {
				return errors.New("source must not define main or init; the trusted driver controls execution")
			}
			if strings.HasPrefix(fn.Name.Name, "Test") {
				return errors.New("source must not define tests")
			}
			if want[fn.Name.Name] {
				if found {
					return errors.New("required function is duplicated")
				}
				found = true
				if err := validSignature(fn, fn.Name.Name); err != nil {
					return err
				}
			}
		default:
			return errors.New("source may contain only functions and constants")
		}
	}
	if !found {
		return errors.New("required function is missing")
	}
	return nil
}
func validSignature(fn *ast.FuncDecl, name string) error {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return fmt.Errorf("%s must have signature func %s(value int) int", name, name)
	}
	param := fn.Type.Params.List[0]
	result := fn.Type.Results.List[0]
	paramType, ok := param.Type.(*ast.Ident)
	if len(param.Names) != 1 || !ok || paramType.Name != "int" {
		return fmt.Errorf("%s must have one int parameter", name)
	}
	r, ok := result.Type.(*ast.Ident)
	if !ok || r.Name != "int" {
		return fmt.Errorf("%s must return int", name)
	}
	return nil
}

// Hash fingerprints all content that can affect a grade, including private
// answers, hidden vectors, trusted driver source, and the criteria version.
func (e *Exam) Hash() string {
	payload := struct {
		ID, Version, Title, Rank, Description string
		EstimatedMinutes                      int
		Prerequisites                         []string
		Criteria                              []string
		Questions                             []Question
		Tasks                                 []Task
		Correct                               map[string]string
		Cases                                 map[string][]Case
		Drivers                               map[string]string
	}{e.ID, e.Version, e.Title, e.Rank, e.Description, e.EstimatedMinutes, e.Prerequisites, e.Criteria, e.Questions, e.Tasks, e.correctIDs, e.cases, e.drivers}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
