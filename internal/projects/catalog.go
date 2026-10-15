package projects

import (
	"embed"
	"errors"
	"fmt"
	"path"
	"strings"
)

//go:embed p2/README.md p2/rubric.md p2/openapi.yaml p2/threat-model.md p2/Dockerfile p2/compose.yaml p2/.env.example p2/.gitignore p2/locked.mod p2/locked.sum p2/starter_readme.md p2/starter_main.go
var p2Assets embed.FS

const (
	P2MaxFiles     = 48
	P2MaxBytes     = 512 * 1024
	P2MaxGoBytes   = 256 * 1024
	P2MaxFileBytes = 128 * 1024
)

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

// All returns the bundled project catalog in learning order.
func All() []Project { return []Project{Load(), loadP2()} }

// Find looks up a bundled project by its stable identifier.
func Find(id string) (Project, bool) {
	for _, p := range All() {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

// ModulePolicy returns the immutable Go module files accepted for a project.
func ModulePolicy(id string) (mod, sum string, ok bool) {
	if id != "p2" {
		return "", "", false
	}
	return p2Module, p2Sum, true
}

// ValidateFor applies the project-specific workspace contract.
func ValidateFor(id string, files map[string]string) error {
	switch id {
	case "p1":
		return ValidateFiles(files)
	case "p2":
		return validateP2Files(files)
	default:
		return fmt.Errorf("unknown project %q", id)
	}
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
	return Project{ID: "p2", Title: "LinkHub API", Rank: "Junior+", Description: "REST API коротких ссылок с PostgreSQL, личными ресурсами, сессиями и защитой браузерных запросов.", Requirements: []string{
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
		if name == "go.mod" && contents != p2Module {
			return errors.New("go.mod does not match the pinned LinkHub module policy")
		}
		if name == "go.sum" && contents != p2Sum {
			return errors.New("go.sum does not match the pinned LinkHub dependency lock")
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
