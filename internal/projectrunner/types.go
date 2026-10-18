// Package projectrunner runs approved P2/P3 student projects in a local, rootless OCI worker.
package projectrunner

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"academy/internal/projects"
	"academy/internal/runner"
)

const (
	MaxBody       = 4 << 20
	MaxFiles      = 48
	MaxTotal      = 512 << 10
	MaxGo         = 256 << 10
	MaxOther      = 128 << 10
	PostgresImage = "docker.io/library/postgres@sha256:9551dd5bf356409a21b31248f49b33bba9bb1877365f3a7e933aecd9af3cf52d"
)

type Request struct {
	ProjectID string            `json:"project_id"`
	Action    string            `json:"action"`
	Files     map[string]string `json:"files"`
}

// EvaluationProfile identifies the immutable project/checker/runtime policy that
// produced an automatic evaluation. It is served only by the local trusted runner.
type EvaluationProfile struct {
	ProjectVersion    string `json:"project_version"`
	CriteriaHash      string `json:"criteria_hash"`
	CheckerVersion    string `json:"checker_version"`
	CheckerDigest     string `json:"checker_digest"`
	RuntimePolicyHash string `json:"runtime_policy_hash"`
	RubricVersion     string `json:"rubric_version"`
	RubricHash        string `json:"rubric_hash"`
}

type Config struct {
	Socket        string
	GoImage       string
	CheckerImage  string
	PostgresImage string
	ExpectedGoMod string
	ExpectedGoSum string
}

type Engine interface {
	Execute(context.Context, Request) (runner.Result, error)
	Available(context.Context) bool
}

var safePart = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func isGenuineTestSource(name, source string) (bool, bool) {
	f, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		return false, false
	}
	aliases := map[string]bool{}
	for _, im := range f.Imports {
		if im.Path.Value == `"testing"` {
			alias := "testing"
			if im.Name != nil {
				alias = im.Name.Name
			}
			aliases[alias] = true
		}
	}
	found := false
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "TestMain" {
			return false, true
		}
		n := fn.Name.Name
		if fn.Recv != nil || len(n) <= 4 || !strings.HasPrefix(n, "Test") || unicode.IsLower([]rune(n[4:])[0]) || fn.Type.Results != nil || fn.Type.Params.NumFields() != 1 {
			continue
		}
		star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "T" {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if ok && aliases[pkg.Name] {
			found = true
		}
	}
	return found, false
}

func validate(req Request, cfg Config) error {
	if req.ProjectID != "p2" && req.ProjectID != "p3" && req.ProjectID != "p4" && req.ProjectID != "p5" {
		return errors.New("unsupported project")
	}
	if req.Action != "run" && req.Action != "test" && req.Action != "submit" {
		return errors.New("unsupported action")
	}
	if (req.ProjectID == "p4" || req.ProjectID == "p5") && (req.Action == "test" || req.Action == "submit") {
		if err := ValidateAssessmentProcessHooks(req.Files); err != nil {
			return err
		}
	}
	if len(req.Files) == 0 || len(req.Files) > MaxFiles {
		return errors.New("invalid file count")
	}
	var total, goBytes int
	hasTest := false
	hasSource, hasReadme := false, false
	modPolicy, sumPolicy := cfg.ExpectedGoMod, cfg.ExpectedGoSum
	if req.ProjectID == "p3" || req.ProjectID == "p4" || req.ProjectID == "p5" {
		var ok bool
		modPolicy, sumPolicy, ok = projects.ModulePolicy(req.ProjectID)
		if !ok {
			return errors.New("project module policy unavailable")
		}
	}
	for name, content := range req.Files {
		if name == "" || strings.ContainsAny(name, "\\\x00") || path.IsAbs(name) || strings.Contains(name, "//") {
			return errors.New("invalid file path")
		}
		parts := strings.Split(name, "/")
		if len(parts) > 3 {
			return errors.New("invalid file path")
		}
		for _, p := range parts {
			if p == "" || p == "." || p == ".." || !safePart.MatchString(p) {
				return errors.New("invalid file path")
			}
		}
		ext := path.Ext(name)
		allowed := ext == ".go" || ext == ".json" || ext == ".sql" || ext == ".md" || ext == ".yaml" || ext == ".yml" || name == "go.mod" || name == "go.sum" || name == "Dockerfile" || name == ".env.example" || name == ".gitignore"
		base := path.Base(name)
		if base == "Dockerfile" || base == "Containerfile" || base == "compose.yaml" || base == "compose.yml" || base == "docker-compose.yml" || base == "docker-compose.yaml" || strings.HasSuffix(base, ".env") {
			return errors.New("runtime configuration files are not accepted")
		}
		if !allowed {
			return fmt.Errorf("unsupported file type")
		}
		if (name == "go.mod" && content != modPolicy) || (name == "go.sum" && content != sumPolicy) {
			return errors.New("module lock files are immutable")
		}
		if name == "README.md" && strings.TrimSpace(content) == "" {
			return errors.New("README.md must not be empty")
		}
		if name == "README.md" {
			hasReadme = true
		}
		if len(content) > MaxOther {
			return errors.New("file too large")
		}
		if ext == ".go" {
			if len(content) > MaxGo {
				return errors.New("Go source too large")
			}
			goBytes += len(content)
			if !strings.HasSuffix(name, "_test.go") {
				hasSource = true
			}
			if strings.HasSuffix(name, "_test.go") {
				genuine, hasTestMain := isGenuineTestSource(name, content)
				hasTest = hasTest || genuine
				if hasTestMain {
					return errors.New("TestMain is not allowed")
				}
			}
		}
		total += len(content)
	}
	if _, ok := req.Files["go.mod"]; !ok {
		return errors.New("go.mod required")
	}
	if _, ok := req.Files["go.sum"]; !ok && req.ProjectID != "p4" && req.ProjectID != "p5" {
		return errors.New("go.sum required")
	}
	if req.ProjectID == "p3" && (!hasSource || !hasReadme) {
		return errors.New("TaskFlow requires a non-test Go source and README.md")
	}
	if goBytes > MaxGo || total > MaxTotal {
		return errors.New("project files too large")
	}
	if (req.Action == "test" || req.ProjectID == "p3" && req.Action == "submit") && !hasTest {
		return errors.New("test action requires a genuine TestX function")
	}
	if req.ProjectID == "p4" {
		if !hasSource || !hasReadme {
			return errors.New("ScaleLab requires non-test Go source and README.md")
		}
		if req.Action == "test" && !hasTest {
			return errors.New("test action requires a genuine TestX function")
		}
		if err := projects.ValidateFor("p4", req.Files); err != nil {
			return err
		}
		for _, required := range []string{"go.mod", "README.md", "docs/project.md", "docs/rubric.md", "main.go", "aggregate.go"} {
			if strings.TrimSpace(req.Files[required]) == "" {
				return fmt.Errorf("ScaleLab required project file missing or empty: %s", required)
			}
		}
	}
	if req.ProjectID == "p5" {
		if !hasSource || !hasReadme {
			return errors.New("Capstone requires non-test Go source and README.md")
		}
		if req.Action == "test" && !hasTest {
			return errors.New("test action requires a genuine TestX function")
		}
		if err := projects.ValidateFor("p5", req.Files); err != nil {
			return err
		}
		for _, required := range []string{"go.mod", "README.md", "main.go", "queue.go", "docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md", "design/adr-template.md"} {
			if strings.TrimSpace(req.Files[required]) == "" {
				return fmt.Errorf("Capstone required project file missing or empty: %s", required)
			}
		}
	}
	return nil
}

// The P4/P5 trusted acceptance overlay is executed by `go test` in the
// isolated project container. Package init functions run before the test
// harness and test code can call os.Exit/Goexit, so either could falsely make
// the Go command exit successfully without running the trusted checks. These
// projects do not require package initialization or process control in tests;
// reject those hooks on machine-assessed actions while leaving normal `run`
// behavior available.
// ValidateAssessmentProcessHooks rejects process termination primitives that
// could skip trusted P4/P5 acceptance tests before they execute.
func ValidateAssessmentProcessHooks(files map[string]string) error {
	for name, source := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, source, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("invalid Go source %s", name)
		}
		importAliases := map[string]string{}
		for _, im := range file.Imports {
			importPath, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				continue
			}
			alias := path.Base(importPath)
			if im.Name != nil {
				alias = im.Name.Name
			}
			if (importPath == "os" || importPath == "runtime" || importPath == "syscall") && alias == "." {
				return fmt.Errorf("%s не может использовать dot-import %s при проверке P4/P5", name, importPath)
			}
			if importPath == "unsafe" {
				return fmt.Errorf("%s не может импортировать unsafe при проверке P4/P5", name)
			}
			if alias != "_" && alias != "." {
				importAliases[alias] = importPath
			}
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if strings.HasPrefix(strings.TrimSpace(comment.Text), "//go:linkname") {
					return fmt.Errorf("%s не может использовать директиву go:linkname при проверке P4/P5", name)
				}
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "init" || fn.Name.Name == "TestMain" {
				return fmt.Errorf("%s не может определять init или TestMain в проверяемом проекте", name)
			}
			// Bodyless top-level functions are only needed for assembly/compiler
			// declarations. P4/P5 assessments have no such requirement, and a
			// declaration can bind a linkname symbol to an otherwise innocent call.
			if fn.Recv == nil && fn.Body == nil {
				return fmt.Errorf("%s не может объявлять функцию без тела при проверке P4/P5", name)
			}
		}
		var processExit bool
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath, importedPackage := importAliases[ident.Name]
			if !importedPackage {
				return true
			}
			switch importPath {
			case "os":
				if selector.Sel.Name == "Exit" {
					processExit = true
				}
			case "runtime":
				if selector.Sel.Name == "Goexit" {
					processExit = true
				}
			case "syscall":
				switch selector.Sel.Name {
				case "Exit", "Exec", "Syscall", "Syscall6", "RawSyscall", "RawSyscall6":
					processExit = true
				}
			}
			return true
		})
		if processExit {
			return fmt.Errorf("%s не может завершать процесс из пользовательских тестов", name)
		}
	}
	return nil
}
