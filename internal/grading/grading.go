// Package grading loads trusted, private-function exercise specifications.
package grading

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"academy/internal/runner"
)

//go:embed specs/*.json
var specFiles embed.FS

const maxVectorBytes = 4 << 10

var importWhitelist = map[string]bool{
	"fmt": true, "strings": true, "strconv": true, "unicode": true, "unicode/utf8": true,
	"errors": true, "math": true, "sort": true, "slices": true, "cmp": true, "bytes": true,
	"encoding/json": true, "encoding/hex": true, "encoding/base64": true, "time": true,
	"net": true, "net/http": true, "net/url": true, "context": true, "io": true, "sync": true, "sync/atomic": true,
}

type Spec struct {
	id              string
	signature       string
	allowedImports  []string
	driver          string
	publicInput     []json.RawMessage
	publicExpected  []json.RawMessage
	privateInput    []json.RawMessage
	privateExpected []json.RawMessage
	boundTypes      map[string]string
	referenceBound  bool
}

// Fingerprint binds private vectors, the driver and the bound type contract to
// an immutable assessment version. It does not publish those private fields.
func (s *Spec) Fingerprint() string {
	if s == nil || !s.referenceBound {
		return ""
	}
	payload := struct {
		Policy string
		Spec   specJSON
		Types  map[string]string
	}{"function-source-policy-v2", specJSON{s.id, s.signature, s.allowedImports, s.driver, s.publicInput, s.publicExpected, s.privateInput, s.privateExpected}, s.boundTypes}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// MarshalJSON intentionally omits trusted driver code and all answer vectors.
func (s *Spec) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(struct {
		ID string `json:"id"`
	}{ID: s.id})
}

type specJSON struct {
	ID              string            `json:"id"`
	Signature       string            `json:"signature"`
	AllowedImports  []string          `json:"allowed_imports"`
	Driver          string            `json:"driver"`
	PublicInput     []json.RawMessage `json:"public_input"`
	PublicExpected  []json.RawMessage `json:"public_expected"`
	PrivateInput    []json.RawMessage `json:"private_input"`
	PrivateExpected []json.RawMessage `json:"private_expected"`
}

func Load() (map[string]*Spec, error) { return loadSpecs(nil) }

// LoadFor loads only published exercises; draft content stays outside runtime.
func LoadFor(ids []string) (map[string]*Spec, error) {
	selected := map[string]bool{}
	for _, id := range ids {
		if !validID(id) {
			return nil, errors.New("invalid selected exercise id")
		}
		selected[id] = true
	}
	return loadSpecs(selected)
}
func loadSpecs(selected map[string]bool) (map[string]*Spec, error) {
	names, err := specFiles.ReadDir("specs")
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Spec, len(names))
	for _, entry := range names {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if selected != nil && !selected[strings.TrimSuffix(entry.Name(), ".json")] {
			continue
		}
		b, err := specFiles.ReadFile("specs/" + entry.Name())
		if err != nil {
			return nil, err
		}
		var raw specJSON
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return nil, fmt.Errorf("%s: trailing JSON", entry.Name())
		}
		s := &Spec{id: raw.ID, signature: raw.Signature, allowedImports: raw.AllowedImports, driver: raw.Driver, publicInput: raw.PublicInput, publicExpected: raw.PublicExpected, privateInput: raw.PrivateInput, privateExpected: raw.PrivateExpected}
		if err := s.validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if entry.Name() != s.id+".json" {
			return nil, fmt.Errorf("%s: filename must match id", entry.Name())
		}
		if _, exists := out[s.id]; exists {
			return nil, fmt.Errorf("duplicate spec %q", s.id)
		}
		out[s.id] = s
	}
	if selected != nil {
		for id := range selected {
			if out[id] == nil {
				return nil, fmt.Errorf("published exercise %s has no spec", id)
			}
		}
	}
	return out, nil
}

func (s *Spec) validate() error {
	if s == nil || !validID(s.id) || strings.TrimSpace(s.signature) == "" {
		return errors.New("invalid id or signature")
	}
	if _, err := parseSignature(s.signature); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	if strings.TrimSpace(s.driver) == "" || len(s.driver) > runner.MaxSource {
		return errors.New("invalid driver size")
	}
	driver, err := parser.ParseFile(token.NewFileSet(), "academy_driver.go", s.driver, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("driver: %w", err)
	}
	if driver.Name.Name != "main" {
		return errors.New("driver must be package main")
	}
	driverMain := false
	for _, decl := range driver.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "main" {
			driverMain = true
		}
	}
	if !driverMain {
		return errors.New("driver must define main")
	}
	if err := validateVector(s.publicInput, s.publicExpected); err != nil {
		return fmt.Errorf("public vectors: %w", err)
	}
	if err := validateVector(s.privateInput, s.privateExpected); err != nil {
		return fmt.Errorf("private vectors: %w", err)
	}
	inputs, err := marshalInputs(s.publicInput, s.privateInput)
	if err != nil {
		return fmt.Errorf("combined input vectors: %w", err)
	}
	if len(inputs) > 4096 {
		return errors.New("combined input vector exceeds runner argument limit")
	}
	outputSize, err := encodedOutputSize(s.publicExpected, s.privateExpected)
	if err != nil {
		return fmt.Errorf("combined expected output: %w", err)
	}
	if outputSize > runner.MaxOutput {
		return errors.New("combined expected output exceeds runner output limit")
	}
	seen := map[string]bool{}
	for _, p := range s.allowedImports {
		if !importWhitelist[p] || seen[p] {
			return fmt.Errorf("invalid allowed import %q", p)
		}
		seen[p] = true
	}
	return nil
}

func validID(s string) bool {
	if s == "_demo-add" {
		return true
	}
	if len(s) > 80 {
		return false
	}
	for _, prefix := range []string{"zero-", "junior-", "juniorplus-", "middle-", "middleplus-", "senior-"} {
		if strings.HasPrefix(s, prefix) && len(s) > len(prefix) {
			return safeSlug(s[len(prefix):])
		}
	}
	return false
}

func safeSlug(s string) bool {
	for i, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' && i > 0 && i < len(s)-1) {
			return false
		}
	}
	return true
}
func validateVector(in, expected []json.RawMessage) error {
	if len(in) < 1 || len(in) > 12 || len(in) != len(expected) {
		return errors.New("must contain 1..12 paired values")
	}
	for _, v := range append(append([]json.RawMessage{}, in...), expected...) {
		if len(v) == 0 || len(v) > maxVectorBytes {
			return errors.New("JSON value exceeds 4 KiB or is empty")
		}
		var x any
		d := json.NewDecoder(bytes.NewReader(v))
		d.UseNumber()
		if err := d.Decode(&x); err != nil {
			return err
		}
		if d.Decode(new(any)) != io.EOF {
			return errors.New("trailing JSON value")
		}
	}
	// The runner accepts at most 4 KiB of process arguments; reject an unusable
	// vector at load time instead of producing requests that the runner rejects.
	encoded, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if len(encoded) > 4096 {
		return errors.New("encoded input vector exceeds runner argument limit")
	}
	return nil
}

func (s *Spec) ExerciseID() string {
	if s == nil {
		return ""
	}
	return s.id
}

func (s *Spec) ValidateSource(files map[string]string) error {
	if s == nil {
		return errors.New("nil spec")
	}
	if err := s.validate(); err != nil {
		return err
	}
	if !s.referenceBound {
		return errors.New("trusted reference has not been bound")
	}
	if len(files) == 0 || len(files) > 15 {
		return errors.New("student files must contain 1..15 Go files")
	}
	total := len(s.driver)
	fset := token.NewFileSet()
	found := false
	studentTypes := make(map[string]*ast.TypeSpec)
	for name, src := range files {
		if name == "academy_driver.go" {
			return errors.New("reserved filename academy_driver.go")
		}
		if err := runner.ValidateFiles(map[string]string{name: src}); err != nil {
			return err
		}
		total += len(src)
		f, err := parser.ParseFile(fset, name, src, parser.AllErrors)
		if err != nil {
			return err
		}
		if f.Name.Name != "main" {
			return errors.New("student files must use package main")
		}
		for _, imp := range f.Imports {
			if imp.Name != nil && (imp.Name.Name == "." || imp.Name.Name == "_") {
				return errors.New("dot and blank imports are not allowed")
			}
			if imp.Name != nil && predeclared(imp.Name.Name) {
				return fmt.Errorf("predeclared identifier %q cannot be rebound", imp.Name.Name)
			}
			p, e := strconvUnquote(imp.Path.Value)
			if e != nil {
				return errors.New("invalid import")
			}
			if p == "C" || p == "unsafe" || p == "os" || p == "os/exec" || p == "runtime" || p == "reflect" || p == "syscall" || strings.HasPrefix(p, ".") || strings.HasPrefix(p, "_") || !importWhitelist[p] || !contains(s.allowedImports, p) {
				return fmt.Errorf("import %q is not allowed", p)
			}
		}
		for _, d := range f.Decls {
			switch n := d.(type) {
			case *ast.FuncDecl:
				if n.Recv != nil && serializationMethod(n.Name.Name) {
					return fmt.Errorf("custom serialization method %s is not allowed", n.Name.Name)
				}
				if n.Recv == nil {
					if predeclared(n.Name.Name) {
						return fmt.Errorf("predeclared identifier %q cannot be rebound", n.Name.Name)
					}
					if n.Name.Name == "init" || n.Name.Name == "main" {
						return fmt.Errorf("function %s is reserved", n.Name.Name)
					}
					if n.Name.Name == requiredName(s.signature) {
						if found || !sameSignature(n, s.signature) {
							return errors.New("required function signature mismatch")
						}
						found = true
					}
				}
			case *ast.GenDecl:
				if n.Tok == token.VAR {
					return errors.New("package variables are not allowed")
				}
				if n.Tok == token.TYPE {
					for _, sp := range n.Specs {
						ts := sp.(*ast.TypeSpec)
						if ts.Assign.IsValid() {
							return errors.New("type aliases are not allowed")
						}
						studentTypes[ts.Name.Name] = ts
						if predeclared(ts.Name.Name) {
							return fmt.Errorf("predeclared identifier %q cannot be rebound", ts.Name.Name)
						}
					}
				}
				if n.Tok == token.CONST {
					for _, sp := range n.Specs {
						for _, id := range sp.(*ast.ValueSpec).Names {
							if predeclared(id.Name) {
								return fmt.Errorf("predeclared identifier %q cannot be rebound", id.Name)
							}
						}
					}
				}
			}
		}
	}
	if total > runner.MaxSource {
		return errors.New("student source plus driver exceeds 64 KiB")
	}
	if !found {
		return errors.New("required function is missing")
	}
	for name, expected := range s.boundTypes {
		actual := studentTypes[name]
		if actual == nil || actual.Assign.IsValid() || canonicalTypeSpec(actual) != expected {
			return fmt.Errorf("boundary type %q does not match trusted reference", name)
		}
	}
	return nil
}

// BindReference records the exact shapes of named types exposed by the trusted
// function signature. Call once during startup before accepting source.
func (s *Spec) BindReference(reference string) error {
	if s == nil {
		return errors.New("nil spec")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "reference.go", reference, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("trusted reference: %w", err)
	}
	decls := make(map[string]*ast.TypeSpec)
	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		switch n := decl.(type) {
		case *ast.FuncDecl:
			if n.Recv == nil && n.Name.Name == requiredName(s.signature) {
				if target != nil {
					return errors.New("duplicate reference function")
				}
				target = n
			}
		case *ast.GenDecl:
			if n.Tok == token.TYPE {
				for _, raw := range n.Specs {
					ts := raw.(*ast.TypeSpec)
					decls[ts.Name.Name] = ts
				}
			}
		}
	}
	if target == nil || !sameSignature(target, s.signature) {
		return errors.New("trusted reference function signature mismatch")
	}
	bound := make(map[string]string)
	var visitType func(ast.Expr)
	var visitDecl func(string)
	visitType = func(expr ast.Expr) {
		for name := range referencedTypeNames(expr, decls) {
			visitDecl(name)
		}
	}
	visitDecl = func(name string) {
		if _, ok := bound[name]; ok {
			return
		}
		ts := decls[name]
		if ts == nil {
			return
		}
		if ts.Assign.IsValid() {
			return
		}
		bound[name] = canonicalTypeSpec(ts)
		visitType(ts.Type)
		if ts.TypeParams != nil {
			for _, f := range ts.TypeParams.List {
				visitType(f.Type)
			}
		}
	}
	for _, list := range []*ast.FieldList{target.Type.Params, target.Type.Results} {
		if list != nil {
			for _, f := range list.List {
				visitType(f.Type)
			}
		}
	}
	s.boundTypes = bound
	s.referenceBound = true
	return nil
}

func referencedTypeNames(expr ast.Expr, declarations map[string]*ast.TypeSpec) map[string]bool {
	found := make(map[string]bool)
	var visit func(ast.Expr)
	visit = func(e ast.Expr) {
		switch n := e.(type) {
		case *ast.Ident:
			if declarations[n.Name] != nil {
				found[n.Name] = true
			}
		case *ast.StarExpr:
			visit(n.X)
		case *ast.ArrayType:
			visit(n.Elt)
		case *ast.MapType:
			visit(n.Key)
			visit(n.Value)
		case *ast.ChanType:
			visit(n.Value)
		case *ast.Ellipsis:
			visit(n.Elt)
		case *ast.FuncType:
			for _, list := range []*ast.FieldList{n.Params, n.Results} {
				if list != nil {
					for _, f := range list.List {
						visit(f.Type)
					}
				}
			}
		case *ast.InterfaceType:
			for _, f := range n.Methods.List {
				visit(f.Type)
			}
		case *ast.StructType:
			for _, f := range n.Fields.List {
				visit(f.Type)
			}
		case *ast.IndexExpr:
			visit(n.X)
			visit(n.Index)
		case *ast.IndexListExpr:
			visit(n.X)
			for _, idx := range n.Indices {
				visit(idx)
			}
		case *ast.ParenExpr:
			visit(n.X)
		case *ast.SelectorExpr: // imported or qualified type; the package name is not a local type.
		}
	}
	visit(expr)
	return found
}

func canonicalTypeSpec(ts *ast.TypeSpec) string {
	if ts == nil {
		return ""
	}
	canonicalizeStructFields(ts.Type)
	return nodeKey(ts.TypeParams) + " " + nodeKey(ts.Type)
}

func canonicalizeStructFields(expr ast.Expr) {
	ast.Inspect(expr, func(n ast.Node) bool {
		var fields *ast.FieldList
		switch x := n.(type) {
		case *ast.StructType:
			fields = x.Fields
		case *ast.InterfaceType:
			fields = x.Methods
		}
		if fields != nil {
			flat := make([]*ast.Field, 0, len(fields.List))
			for _, field := range fields.List {
				if len(field.Names) == 0 {
					field.Doc = nil
					field.Comment = nil
					flat = append(flat, field)
					continue
				}
				for _, id := range field.Names {
					flat = append(flat, &ast.Field{Names: []*ast.Ident{id}, Type: field.Type, Tag: field.Tag})
				}
			}
			fields.List = flat
		}
		return true
	})
}

func serializationMethod(name string) bool {
	switch name {
	case "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText", "MarshalJSONTo", "UnmarshalJSONFrom":
		return true
	}
	return false
}

// Request uses private=false for public examples. private=true is a submission
// run: it combines public and private inputs into one runner invocation.
func (s *Spec) Request(files map[string]string, private bool) (runner.Request, error) {
	if err := s.ValidateSource(files); err != nil {
		return runner.Request{}, err
	}
	inputs := s.publicInput
	if private {
		inputs = marshalRawList(s.publicInput, s.privateInput)
	}
	encoded, err := marshalInputs(inputs)
	if err != nil {
		return runner.Request{}, err
	}
	merged := make(map[string]string, len(files)+1)
	for k, v := range files {
		merged[k] = v
	}
	merged["academy_driver.go"] = s.driver
	return runner.Request{Files: merged, Mode: "run", Args: []string{string(encoded)}}, nil
}

// Check verifies public examples, or both public and private cases for a
// submission when private is true.
func (s *Spec) Check(result runner.Result, private bool) []runner.Check {
	expected := s.publicExpected
	if private {
		expected = marshalRawList(s.publicExpected, s.privateExpected)
	}
	checks := make([]runner.Check, 0, len(expected))
	publicCount := len(s.publicExpected)
	pass := result.ExitCode == 0 && result.Status == "ok" && result.Stderr == ""
	lines := strings.Split(strings.TrimSuffix(result.Stdout, "\n"), "\n")
	if result.Stdout == "" {
		lines = nil
	}
	if len(lines) != len(expected) {
		pass = false
	}
	for i, e := range expected {
		ok := pass && i < len(lines) && strictJSONEqual([]byte(lines[i]), e)
		name := "пример"
		if private && i >= publicCount {
			name = "закрытый пример"
		}
		checks = append(checks, runner.Check{Name: fmt.Sprintf("%s %d", name, i+1), Passed: ok})
	}
	return checks
}

func marshalRawList(groups ...[]json.RawMessage) []json.RawMessage {
	var out []json.RawMessage
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

func marshalInputs(groups ...[]json.RawMessage) ([]byte, error) {
	raw := marshalRawList(groups...)
	values := make([]any, len(raw))
	for i, b := range raw {
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if err := d.Decode(&values[i]); err != nil {
			return nil, err
		}
	}
	return json.Marshal(values)
}

func encodedOutputSize(groups ...[]json.RawMessage) (int, error) {
	raw := marshalRawList(groups...)
	total := 0
	for _, b := range raw {
		var value any
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if err := d.Decode(&value); err != nil {
			return 0, err
		}
		line, err := json.Marshal(value)
		if err != nil {
			return 0, err
		}
		total += len(line) + 1 // json.Encoder appends one newline per value.
	}
	return total, nil
}

func strictJSONEqual(a, b []byte) bool {
	var x, y any
	da := json.NewDecoder(bytes.NewReader(a))
	da.UseNumber()
	db := json.NewDecoder(bytes.NewReader(b))
	db.UseNumber()
	if da.Decode(&x) != nil || db.Decode(&y) != nil {
		return false
	}
	if da.Decode(new(any)) != io.EOF || db.Decode(new(any)) != io.EOF {
		return false
	}
	aa, _ := json.Marshal(x)
	bb, _ := json.Marshal(y)
	return bytes.Equal(aa, bb)
}

func (s *Spec) PublicOutput() string {
	if s == nil {
		return ""
	}
	var out strings.Builder
	for _, raw := range s.publicExpected {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return ""
		}
		line, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		out.Write(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func parseSignature(sig string) (*ast.FuncType, error) {
	f, e := parser.ParseFile(token.NewFileSet(), "signature.go", "package p\n"+sig, parser.AllErrors)
	if e != nil {
		return nil, e
	}
	if len(f.Decls) != 1 {
		return nil, errors.New("exactly one function declaration required")
	}
	x, ok := f.Decls[0].(*ast.FuncDecl)
	if !ok || x.Recv != nil || x.Body != nil {
		return nil, errors.New("top-level function signature without a body required")
	}
	return x.Type, nil
}
func requiredName(sig string) string {
	f, _ := parser.ParseFile(token.NewFileSet(), "signature.go", "package p\n"+sig, 0)
	for _, d := range f.Decls {
		if x, ok := d.(*ast.FuncDecl); ok {
			return x.Name.Name
		}
	}
	return ""
}
func sameSignature(fn *ast.FuncDecl, sig string) bool {
	want, e := parseSignature(sig)
	if e != nil {
		return false
	}
	return fieldsKey(fn.Type.Params) == fieldsKey(want.Params) && fieldsKey(fn.Type.Results) == fieldsKey(want.Results) && nodeKey(fn.Type.TypeParams) == nodeKey(want.TypeParams)
}
func fieldsKey(f *ast.FieldList) string {
	if f == nil {
		return ""
	}
	var out []string
	for _, x := range f.List {
		key := nodeKey(x.Type)
		for range x.Names {
			out = append(out, key)
		}
		if len(x.Names) == 0 {
			out = append(out, key)
		}
	}
	return strings.Join(out, ";")
}
func nodeKey(n ast.Node) string {
	if n == nil {
		return ""
	}
	var b bytes.Buffer
	if format.Node(&b, token.NewFileSet(), n) != nil {
		return "!"
	}
	return b.String()
}
func strconvUnquote(s string) (string, error) { return strconv.Unquote(s) }
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
func predeclared(s string) bool {
	return types.Universe.Lookup(s) != nil
}
