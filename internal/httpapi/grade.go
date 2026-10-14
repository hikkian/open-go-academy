package httpapi

import (
	"academy/internal/runner"
	"go/ast"
	"go/parser"
	"go/token"
)

// Structural rules only cover these introductory straight-line exercises.
// Requiring direct statements prevents an unused/dead branch from satisfying a criterion.
func structure(id string, files map[string]string) runner.Check {
	var main, answer *ast.FuncDecl
	fmtImport := false
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			return runner.Check{Name: "Обязательная структура задания", Passed: false}
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
				switch fn.Name.Name {
				case "main":
					main = fn
					for _, im := range f.Imports {
						if im.Path.Value == `"fmt"` && (im.Name == nil || im.Name.Name == "fmt") {
							fmtImport = true
						}
					}
				case "answer":
					answer = fn
				}
			}
		}
	}
	ok := main != nil
	if id == "repair" || id == "two-lines" {
		ok = ok && fmtImport
		prints := 0
		answerPrinted := false
		if ok {
			for _, statement := range main.Body.List {
				switch s := statement.(type) {
				case *ast.DeclStmt, *ast.AssignStmt: // Local values before the required direct calls are allowed.
				case *ast.ExprStmt:
					c, isCall := s.X.(*ast.CallExpr)
					if !isCall {
						ok = false
						continue
					}
					sel, isSelector := c.Fun.(*ast.SelectorExpr)
					if !isSelector {
						ok = false
						continue
					}
					p, isPackage := sel.X.(*ast.Ident)
					if !isPackage || p.Name != "fmt" || sel.Sel.Name != "Println" {
						ok = false
						continue
					}
					prints++
					if len(c.Args) == 1 {
						call, isCall := c.Args[0].(*ast.CallExpr)
						if isCall {
							fn, isName := call.Fun.(*ast.Ident)
							answerPrinted = answerPrinted || (isName && fn.Name == "answer" && len(call.Args) == 0)
						}
					}
				default:
					ok = false
				}
			}
		}
		if id == "repair" {
			ok = ok && prints == 1 && answerPrinted && answer != nil && answer.Type.Params.NumFields() == 0 && answer.Type.Results != nil && answer.Type.Results.NumFields() == 1
			if ok {
				i, isInt := answer.Type.Results.List[0].Type.(*ast.Ident)
				ok = isInt && i.Name == "int"
			}
		}
		if id == "two-lines" {
			ok = ok && prints == 2
		}
	}
	return runner.Check{Name: "Обязательная структура задания", Passed: ok}
}
