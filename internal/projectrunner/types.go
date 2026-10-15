// Package projectrunner runs the P2 LinkHub project in a local, rootless OCI worker.
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
	"strings"
	"unicode"

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
	if req.ProjectID != "p2" {
		return errors.New("unsupported project")
	}
	if req.Action != "run" && req.Action != "test" && req.Action != "submit" {
		return errors.New("unsupported action")
	}
	if len(req.Files) == 0 || len(req.Files) > MaxFiles {
		return errors.New("invalid file count")
	}
	var total, goBytes int
	hasTest := false
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
		if base == "Dockerfile" || base == "compose.yaml" || base == "docker-compose.yml" || strings.HasSuffix(base, ".env") {
			return errors.New("runtime configuration files are not accepted")
		}
		if !allowed {
			return fmt.Errorf("unsupported file type")
		}
		if (name == "go.mod" && content != cfg.ExpectedGoMod) || (name == "go.sum" && content != cfg.ExpectedGoSum) {
			return errors.New("module lock files are immutable")
		}
		if len(content) > MaxOther {
			return errors.New("file too large")
		}
		if ext == ".go" {
			if len(content) > MaxGo {
				return errors.New("Go source too large")
			}
			goBytes += len(content)
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
	if _, ok := req.Files["go.sum"]; !ok {
		return errors.New("go.sum required")
	}
	if goBytes > MaxGo || total > MaxTotal {
		return errors.New("project files too large")
	}
	if req.Action == "test" && !hasTest {
		return errors.New("test action requires a genuine TestX function")
	}
	return nil
}
