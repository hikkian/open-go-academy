package projects

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed p2/README.md p2/rubric.md p2/openapi.yaml p2/threat-model.md p2/Dockerfile p2/compose.yaml p2/.env.example p2/.gitignore p2/locked.mod p2/locked.sum p2/starter_readme.md p2/starter_main.go
var p2Assets embed.FS

//go:embed p4bundle/README.md p4bundle/rubric.md p4bundle/go.mod.template p4bundle/starter_files/*
var p4Assets embed.FS

//go:embed p5bundle/README.md p5bundle/rubric.md p5bundle/threat-model.md p5bundle/go.mod.template p5bundle/starter_files/* p5bundle/starter_files/design/*
var p5Assets embed.FS

const (
	P2MaxFiles     = 48
	P2MaxBytes     = 512 * 1024
	P2MaxGoBytes   = 256 * 1024
	P2MaxFileBytes = 128 * 1024
)

// P3 locks are compiled into the server policy. They intentionally match the
// P3 TaskFlow reference dependency graph; client-supplied module files are never
// used to choose the runner cache or image profile.
const p3Module = `module taskflow

go 1.27.2

require github.com/jackc/pgx/v5 v5.11.0

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
`

const p3Sum = `github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/jackc/pgpassfile v1.0.0 h1:/6Hmqy13Ss2zCq62VdNG8tM1wchn8zjSGOBJ6icpsIM=
github.com/jackc/pgpassfile v1.0.0/go.mod h1:CEx0iS5ambNFdcRtxPj5JhEz+xB6uRky5eyVu/W2HEg=
github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 h1:iCEnooe7UlwOQYpKFhBabPMi4aNAfoODPEFNiAnClxo=
github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761/go.mod h1:5TJZWKEWniPve33vlWYSoGYefn3gLQRzjfDlhSJ9ZKM=
github.com/jackc/pgx/v5 v5.11.0 h1:IzBBtyK9AHqf98cctWFifYSci2hgQR/cd56wB4p+ogg=
github.com/jackc/pgx/v5 v5.11.0/go.mod h1:mal1tBGAFfLHvZzaYh77YS/eC6IX9OWbRV1QIIM0Jn4=
github.com/jackc/puddle/v2 v2.2.2 h1:PR8nw+E/1w0GLuRFSmiioY6UooMp6KJv0/61nB7icHo=
github.com/jackc/puddle/v2 v2.2.2/go.mod h1:vriiEXHvEE654aYKXXjOvZM39qJ0q+azkZFrfEOc3H4=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/testify v1.3.0/go.mod h1:M5WIy9Dh21IEIfnGCwXGc5bZfKNJtfHm1UVUgZn+9EI=
github.com/stretchr/testify v1.7.0/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.11.1 h1:7s2iGBzp5EwR7/aIZr8ao5+dra3wiQyKjjFuvgVKu7U=
github.com/stretchr/testify v1.11.1/go.mod h1:wZwfW3scLgRK+23gO65QZefKpKQRnfz6sD981Nm4B6U=
golang.org/x/sync v0.17.0 h1:l60nONMj9l5drqw6jlhIELNv9I0A4OFgRsG9k2oT9Ug=
golang.org/x/sync v0.17.0/go.mod h1:9KTHXmSnoGruLpwFjVSX0lNNA75CykiMECbovNTZqGI=
golang.org/x/text v0.29.0 h1:1neNs90w9YzJ9BocxfsQNHKuAT4pkghyXc4nhZ6sJvk=
golang.org/x/text v0.29.0/go.mod h1:7MhJOA9CD2qZyOKYazxdYMF85OwPdEr9jTtBpO7ydH4=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
`

const p4Module = "module open-go-academy/p4-scalelab/starter\n\ngo 1.27.2\n"
const p5Module = "module p5-capstone-prototype\n\ngo 1.27.2\n"

const p3StarterMain = `package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})
	server := &http.Server{Addr: env("TASKFLOW_ADDR", "127.0.0.1:8098"), Handler: mux, ReadHeaderTimeout: 5*time.Second, ReadTimeout: 10*time.Second, WriteTimeout: 15*time.Second, IdleTimeout: 60*time.Second, MaxHeaderBytes: 16 << 10}
	slog.Info("TaskFlow starter listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed { slog.Error("server failed"); os.Exit(1) }
}
func env(key, fallback string) string { if value := os.Getenv(key); value != "" { return value }; return fallback }
`

const p3StarterReadme = `# TaskFlow — проект Middle

Реализуй локальный Go/PostgreSQL backend: пользователи, workspaces, проекты, задачи, комментарии и durable outbox с ограниченным worker. ` + "`go.mod`" + ` / ` + "`go.sum`" + ` закреплены политикой платформы; меняй остальной код, не dependency lock.

Обязательные этапы: схема и обратимые миграции; server-owned sessions и tenant RBAC; parameterized CRUD со stable bounded pagination; idempotency и транзакционный outbox; lease/fencing/retry/drain; liveness и DB/worker readiness. Тестируй role downgrade race, IDOR, повторный ключ, stale version, повторную доставку, restart persistence, миграции, ограниченную нагрузку и ошибки запуска.

Управляемый runner выполняет только файлы проекта с server-owned Go/DB конфигурацией. Dockerfile/Compose/env примеры являются экспортными файлами и никогда не исполняются. Не добавляй секреты. Проект предназначен для локального обучения: автоматические проверки подтверждают только наблюдаемое поведение и не сертифицируют public multi-tenant безопасность, профессию или производственный опыт. Архитектура, качество кода и компромиссы остаются предметом отдельного ручного review.
`

var p2Module, p2Sum string

func init() {
	b, err := p2Assets.ReadFile("p2/locked.mod")
	if err != nil {
		panic(err)
	}
	p2Module = string(b)
	b, err = p2Assets.ReadFile("p2/locked.sum")
	if err != nil {
		panic(err)
	}
	p2Sum = string(b)
}

// All returns projects whose execution profile is bundled in every installation.
// TaskFlow is opt-in because it requires the separately pinned PostgreSQL checker profile.
func All() []Project { return []Project{Load(), loadP2()} }

// AllWithTaskFlow adds P3 only when the caller has confirmed a TaskFlow-capable runner.
func AllWithTaskFlow(taskFlowAvailable bool) []Project {
	return AllWithProfiles(taskFlowAvailable, false, false)
}

// AllWithProfiles returns projects whose trusted executor profile was reported
// by the currently available local project runner.
func AllWithProfiles(taskFlowAvailable, scaleLabAvailable, capstoneAvailable bool) []Project {
	all := All()
	if taskFlowAvailable {
		all = append(all, loadP3())
	}
	if scaleLabAvailable {
		all = append(all, loadP4())
	}
	if capstoneAvailable {
		all = append(all, loadP5())
	}
	return all
}

// Find looks up a bundled project by its stable identifier.
func Find(id string) (Project, bool) {
	if id == "p3" {
		return loadP3(), true
	}
	if id == "p4" {
		return loadP4(), true
	}
	if id == "p5" {
		return loadP5(), true
	}
	for _, p := range All() {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

// ModulePolicy returns the immutable Go module files accepted for a project.
func ModulePolicy(id string) (mod, sum string, ok bool) {
	switch id {
	case "p2":
		return p2Module, p2Sum, true
	case "p3":
		return p3Module, p3Sum, true
	case "p4":
		return p4Module, "", true
	case "p5":
		return p5Module, "", true
	default:
		return "", "", false
	}
}

// ValidateFor applies the project-specific workspace contract.
func ValidateFor(id string, files map[string]string) error {
	switch id {
	case "p1":
		return ValidateFiles(files)
	case "p2":
		return validateP2Files(files)
	case "p3":
		return validateP3Files(files)
	case "p4":
		return validateP4Files(files)
	case "p5":
		return validateP5Files(files)
	default:
		return fmt.Errorf("unknown project %q", id)
	}
}

func loadP4() Project {
	read := func(name string) string {
		b, err := p4Assets.ReadFile(name)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
	starter := map[string]string{"go.mod": p4Module}
	entries, err := p4Assets.ReadDir("p4bundle/starter_files")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		starter[entry.Name()] = read("p4bundle/starter_files/" + entry.Name())
	}
	starter["go.mod"] = read("p4bundle/go.mod.template")
	starter["docs/project.md"] = read("p4bundle/README.md")
	starter["docs/rubric.md"] = read("p4bundle/rubric.md")
	return Project{ID: "p4", Version: "p4-scalelab-v1", Title: "ScaleLab: измеряемая оптимизация", Rank: "Middle+",
		Description: "Профилируй измеряемый bottleneck, оптимизируй обработку событий без изменения контракта и представь воспроизводимый evidence package.",
		Requirements: []string{
			"Зафиксируй baseline, одинаковый набор данных, версию Go, среду и сырые измерения.",
			"Сравни корректность и стоимость baseline/optimized; объясни выводы CPU/heap профилей.",
			"Проведи capacity/SLO расчёт с допущениями и проверь граничные/отказные сценарии.",
			"Раздели автоматические проверки и ручную интерпретацию; микробенчмарк не подтверждает production SLO.",
		},
		Milestones: []Milestone{
			{ID: "baseline", Title: "Baseline и гипотеза", Criteria: []string{"Формулировка bottleneck до изменений.", "Повторяемый benchmark и сохранённые сырые измерения."}},
			{ID: "profile", Title: "Профиль и оптимизация", Criteria: []string{"Профиль подтверждает выбранную гипотезу.", "Изменение сохраняет контракт и имеет измеримое объяснение."}},
			{ID: "capacity", Title: "Capacity и эксплуатация", Criteria: []string{"Размерностно корректный SLO/capacity расчёт.", "Edge/fault review и ограничения переноса в production."}},
		}, StarterFiles: starter, PublicExamples: []Example{}}
}

func validateP4Files(files map[string]string) error {
	if len(files) == 0 || len(files) > 24 {
		return errors.New("invalid ScaleLab file count")
	}
	total, goBytes := 0, 0
	hasSource, hasReadme := false, false
	for name, contents := range files {
		if !safeP2Path(name) || !(strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md") || name == "go.mod") {
			return fmt.Errorf("unsupported ScaleLab file %q", name)
		}
		if len(contents) > 128*1024 {
			return fmt.Errorf("ScaleLab file %q exceeds size limit", name)
		}
		total += len(contents)
		if total > 512*1024 {
			return errors.New("ScaleLab workspace exceeds total size limit")
		}
		if strings.HasSuffix(name, ".go") {
			goBytes += len(contents)
			if !strings.HasSuffix(name, "_test.go") {
				hasSource = true
			}
		}
		if name == "README.md" {
			hasReadme = strings.TrimSpace(contents) != ""
		}
		if name == "go.mod" && contents != p4Module {
			return errors.New("ScaleLab go.mod is immutable")
		}
	}
	if goBytes > 256*1024 {
		return errors.New("ScaleLab Go source exceeds size limit")
	}
	if !hasSource || !hasReadme || files["go.mod"] != p4Module {
		return errors.New("ScaleLab requires Go source, README.md and the pinned go.mod")
	}
	for _, name := range []string{"main.go", "docs/project.md", "docs/rubric.md", "aggregate.go"} {
		if _, ok := files[name]; !ok {
			return fmt.Errorf("ScaleLab requires %s", name)
		}
	}
	return nil
}

func loadP5() Project {
	read := func(name string) string {
		b, err := p5Assets.ReadFile(name)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
	starter := map[string]string{
		"go.mod": read("p5bundle/go.mod.template"), "README.md": read("p5bundle/starter_files/README.md"),
		"docs/project.md": read("p5bundle/README.md"), "docs/rubric.md": read("p5bundle/rubric.md"),
		"docs/threat-model.md": read("p5bundle/threat-model.md"),
	}
	err := fs.WalkDir(p5Assets, "p5bundle/starter_files", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || name == "p5bundle/starter_files/README.md" {
			return nil
		}
		contents, err := p5Assets.ReadFile(name)
		if err != nil {
			return err
		}
		starter[strings.TrimPrefix(name, "p5bundle/starter_files/")] = string(contents)
		return nil
	})
	if err != nil {
		panic("invalid embedded P5 starter: " + err.Error())
	}
	return Project{ID: "p5", Version: "p5-architecture-capstone-v1", Title: "Architecture & Systems Capstone", Rank: "Senior",
		Description: "Системное проектирование при неполных требованиях, проверка выбранного риска и отдельная защита артефактов независимым человеком.",
		Requirements: []string{
			"Раздели требования на подтверждённые и временные допущения с владельцами и планом проверки.",
			"Подготовь architecture package, модель отказов/угроз, API/data invariants и не менее трёх ADR.",
			"Собери capacity/SLO расчёт, rollout/rollback, backup/restore, runbooks и incident postmortem.",
			"Проверь небольшой Go prototype для выбранного риска; его тесты не доказывают свойства production-инфраструктуры.",
			"Пройди независимую аутентифицированную защиту точного artifact hash по версии rubric; AI и самооценка не могут одобрить Senior gate.",
		},
		Milestones: []Milestone{
			{ID: "requirements", Title: "Требования и допущения", Criteria: []string{"Неизвестные требования явно записаны.", "У каждого временного числового порога есть владелец и способ верификации."}},
			{ID: "architecture", Title: "Архитектура и ADR", Criteria: []string{"Показаны доверительные границы и пути отказа.", "ADR сравнивают жизнеспособные альтернативы."}},
			{ID: "prototype", Title: "Проверка выбранного риска", Criteria: []string{"Прототип воспроизводим и имеет детерминированные тесты.", "Выводы ограничены свойствами самого прототипа."}},
			{ID: "operations", Title: "Эксплуатация и восстановление", Criteria: []string{"Есть backup/restore, rollout/rollback, runbooks и учебный postmortem."}},
			{ID: "defense", Title: "Независимая защита", Criteria: []string{"Аутентифицированный независимый reviewer оценил точный immutable artifact hash.", "AI, automation, self-review и отсутствие reviewer не закрывают этот gate."}},
		}, StarterFiles: starter, PublicExamples: []Example{}}
}

func validateP5Files(files map[string]string) error {
	if len(files) == 0 || len(files) > 48 {
		return errors.New("invalid Capstone file count")
	}
	total, goBytes := 0, 0
	hasSource, hasReadme := false, false
	for name, contents := range files {
		if !safeP2Path(name) || !(strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json") || name == "go.mod") {
			return fmt.Errorf("unsupported Capstone file %q", name)
		}
		if len(contents) > P2MaxFileBytes {
			return fmt.Errorf("Capstone file %q exceeds size limit", name)
		}
		total += len(contents)
		if total > P2MaxBytes {
			return errors.New("Capstone workspace exceeds total size limit")
		}
		if strings.HasSuffix(name, ".go") {
			goBytes += len(contents)
			if !strings.HasSuffix(name, "_test.go") {
				hasSource = true
			}
		}
		if name == "README.md" {
			hasReadme = strings.TrimSpace(contents) != ""
		}
		if name == "go.mod" && contents != p5Module {
			return errors.New("Capstone go.mod is immutable")
		}
	}
	if goBytes > P2MaxGoBytes {
		return errors.New("Capstone Go source exceeds size limit")
	}
	if !hasSource || !hasReadme || files["go.mod"] != p5Module {
		return errors.New("Capstone requires Go source, README.md and the pinned go.mod")
	}
	for _, name := range []string{"main.go", "queue.go", "docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/adr-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md"} {
		if _, ok := files[name]; !ok {
			return fmt.Errorf("Capstone requires %s", name)
		}
	}
	return nil
}

func loadP3() Project {
	starter := map[string]string{"main.go": p3StarterMain, "go.mod": p3Module, "go.sum": p3Sum, "README.md": p3StarterReadme}
	return Project{ID: "p3", Version: "p3-taskflow-v1", Title: "TaskFlow backend", Rank: "Middle", Description: "Локальный Go/PostgreSQL backend с tenant RBAC, транзакционным outbox и bounded worker.", Requirements: []string{
		"Реализуй server-owned identity, workspaces/projects/tasks/comments и owner/editor/viewer RBAC.",
		"Проверь IDOR, атомарное управление ролью, параметризованные SQL и транзакции.",
		"Реализуй project-scoped idempotency, version checks, outbox leases/fencing/retry/drain.",
		"Сохраняй данные после перезапуска; проверь `/livez` и зависимостную `/readyz`.",
		"Не исполняй пользовательские Compose/Dockerfile/env; все проверки проходят только внутри изолированного runner.",
	}, Milestones: []Milestone{
		{ID: "schema", Title: "Схема и границы", Criteria: []string{"Спроектируй tenant-модель, RBAC и обратимые миграции.", "Зафиксируй API, индексы и угрозы до реализации."}},
		{ID: "api", Title: "TaskFlow API", Criteria: []string{"Реализуй сессии, workspaces, проекты, задачи и комментарии.", "Докажи запрет IDOR и конкурентного downgrade роли."}},
		{ID: "jobs", Title: "Outbox и worker", Criteria: []string{"Фиксируй задачу и событие одной транзакцией.", "Проверь fenced lease, retries, terminal failure и bounded drain."}},
		{ID: "evidence", Title: "Проверка и review", Criteria: []string{"Пройди trusted HTTP/DB acceptance на disposable PostgreSQL.", "Архитектура, качество решений и эксплуатационные компромиссы требуют ручного review."}},
	}, StarterFiles: starter, PublicExamples: []Example{}}
}

func validateP3Files(files map[string]string) error {
	return validateLockedProject(files, p3Module, p3Sum, "TaskFlow")
}

func loadP2() Project {
	read := func(name string) string {
		b, err := p2Assets.ReadFile(name)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
	starter := map[string]string{
		"main.go": read("p2/starter_main.go"), "go.mod": p2Module, "go.sum": p2Sum,
		"README.md": read("p2/starter_readme.md"), "Dockerfile": read("p2/Dockerfile"),
		".env.example": read("p2/.env.example"), ".gitignore": read("p2/.gitignore"),
		"compose.yaml": read("p2/compose.yaml"), "docs/project.md": read("p2/README.md"),
		"docs/rubric.md": read("p2/rubric.md"), "docs/openapi.yaml": read("p2/openapi.yaml"),
		"docs/threat-model.md":          read("p2/threat-model.md"),
		"migrations/0001_init.up.sql":   "-- TODO: create tables, constraints, and indexes from docs/project.md\n",
		"migrations/0001_init.down.sql": "-- TODO: reverse 0001_init.up.sql safely\n",
	}
	return Project{ID: "p2", Version: "p2-linkhub-v1", Title: "LinkHub API", Rank: "Junior+", Description: "REST API коротких ссылок с PostgreSQL, личными ресурсами, сессиями и защитой браузерных запросов.", Requirements: []string{
		"Реализуй регистрацию, вход, выход, текущего пользователя, ссылки и теги по контракту OpenAPI.",
		"Храни пароли через KDF с солью; используй непрозрачные отзываемые сессии, CSRF и проверку Origin.",
		"Ограничивай доступ владельцем; обновляй ссылки и связи тегов атомарно параметризованными SQL-запросами.",
		"Проверяй адрес перенаправления без сетевого запроса; ограничивай ввод и возвращай безопасные ошибки.",
		"Добавь обратимые миграции, HTTP-тесты, отдельные тесты PostgreSQL и инструкции эксплуатации.",
	}, Milestones: []Milestone{
		{ID: "schema", Title: "Схема и хранилище", Criteria: []string{"Создай users, sessions, links, tags и связи.", "Добавь ограничения, индексы и обратимые миграции.", "Примени миграции к пустой БД."}},
		{ID: "auth", Title: "Аутентификация и HTTP", Criteria: []string{"Ограничь JSON и проверь ввод.", "Реализуй хеширование паролей и отзываемые сессии.", "Защити изменения через cookie проверками CSRF и Origin."}},
		{ID: "resources", Title: "Ссылки и теги владельца", Criteria: []string{"Проверяй владельца каждой операции с ресурсом.", "Сохраняй ссылку и связи тегов в одной транзакции.", "Отклоняй привязку чужих тегов."}},
		{ID: "evidence", Title: "Доказательства готовности", Criteria: []string{"Добавь HTTP-тесты с тестовым хранилищем и отдельные тесты PostgreSQL.", "Пройди тесты, vet и сборку.", "Опиши локальный запуск Compose и обоснуй архитектурное решение."}},
	}, StarterFiles: starter, PublicExamples: []Example{}}

}

func validateP2Files(files map[string]string) error {
	return validateLockedProject(files, p2Module, p2Sum, "LinkHub")
}

func validateLockedProject(files map[string]string, module, sum, title string) error {
	if len(files) == 0 {
		return errors.New("project must contain at least one file")
	}
	if len(files) > P2MaxFiles {
		return fmt.Errorf("project exceeds %d files", P2MaxFiles)
	}
	total, goBytes := 0, 0
	hasSource, hasReadme := false, false
	for name, contents := range files {
		if !safeP2Path(name) {
			return fmt.Errorf("unsafe project path %q", name)
		}
		if len(contents) > P2MaxFileBytes {
			return fmt.Errorf("project file %q exceeds %d bytes", name, P2MaxFileBytes)
		}
		total += len(contents)
		if strings.HasSuffix(name, ".go") {
			goBytes += len(contents)
			if !strings.HasSuffix(name, "_test.go") {
				hasSource = true
			}
		}
		if name == "README.md" {
			hasReadme = true
			if strings.TrimSpace(contents) == "" {
				return errors.New("README.md must not be empty")
			}
		}
		if name == "go.mod" && contents != module {
			return fmt.Errorf("go.mod does not match the pinned %s module policy", title)
		}
		if name == "go.sum" && contents != sum {
			return fmt.Errorf("go.sum does not match the pinned %s dependency lock", title)
		}
	}
	if total > P2MaxBytes {
		return fmt.Errorf("project exceeds %d bytes", P2MaxBytes)
	}
	if goBytes > P2MaxGoBytes {
		return fmt.Errorf("Go source files exceed %d bytes", P2MaxGoBytes)
	}
	if !hasSource {
		return errors.New("project must contain at least one non-test Go source file")
	}
	if !hasReadme {
		return errors.New("project must contain README.md")
	}
	if _, ok := files["go.mod"]; !ok {
		return errors.New("project must contain go.mod")
	}
	if _, ok := files["go.sum"]; !ok {
		return errors.New("project must contain go.sum")
	}
	return nil
}

func safeP2Path(name string) bool {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\\x00\r\n\t") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, r := range part {
			if r < 0x21 || r > 0x7e {
				return false
			}
		}
	}
	base := parts[len(parts)-1]
	if base == "go.mod" || base == "go.sum" || base == "Dockerfile" || base == ".env.example" || base == ".gitignore" {
		return true
	}
	for _, ext := range []string{".go", ".json", ".sql", ".md", ".yaml", ".yml"} {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	return false
}
