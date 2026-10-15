package grading

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"academy/internal/runner"
)

func demo(t *testing.T) *Spec {
	t.Helper()
	b, err := specFiles.ReadFile("specs/_demo-add.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw specJSON
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	s := &Spec{id: raw.ID, signature: raw.Signature, allowedImports: raw.AllowedImports, driver: raw.Driver, publicInput: raw.PublicInput, publicExpected: raw.PublicExpected, privateInput: raw.PrivateInput, privateExpected: raw.PrivateExpected}
	if err := s.validate(); err != nil {
		t.Fatal(err)
	}
	if err := s.BindReference("package main\nfunc Add(a, b int) int { return a+b }"); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLoadCatalog(t *testing.T) {
	specs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if specs["_demo-add"] == nil {
		t.Fatal("demo spec missing")
	}
}

func TestFingerprintBindsPrivateCasesAndRequiredTypes(t *testing.T) {
	s := demo(t)
	baseline := s.Fingerprint()
	if len(baseline) != 64 || baseline != demo(t).Fingerprint() {
		t.Fatal("fingerprint is not stable")
	}
	s.privateExpected = []json.RawMessage{json.RawMessage(`301`)}
	if s.Fingerprint() == baseline {
		t.Fatal("private grading change did not change fingerprint")
	}
	s = demo(t)
	s.boundTypes = map[string]string{"Input": "struct{N int64}"}
	if s.Fingerprint() == baseline {
		t.Fatal("required type change did not change fingerprint")
	}
	s.referenceBound = false
	if s.Fingerprint() != "" {
		t.Fatal("unbound source policy was fingerprinted")
	}
}

func TestLoadRequestAndPrivateDataIsolation(t *testing.T) {
	s := demo(t)
	files := map[string]string{"main.go": "package main\nfunc Add(left, right int) int { return left+right }"}
	pub, err := s.Request(files, false)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := s.Request(files, true)
	if err != nil {
		t.Fatal(err)
	}
	if pub.Mode != "run" || priv.Mode != "run" || len(pub.Args) != 1 || len(priv.Args) != 1 {
		t.Fatalf("bad runner request: %#v %#v", pub, priv)
	}
	if strings.Contains(pub.Args[0], "300") || strings.Contains(priv.Args[0], "300") {
		t.Fatal("expected values leaked into runner arguments")
	}
	var publicInputs, submissionInputs [][]int
	if err := json.Unmarshal([]byte(pub.Args[0]), &publicInputs); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(priv.Args[0]), &submissionInputs); err != nil {
		t.Fatal(err)
	}
	if len(publicInputs) != 2 || len(submissionInputs) != 3 || submissionInputs[2][0] != 100 || strings.Contains(pub.Args[0], "100") {
		t.Fatal("input vector selection incorrect")
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "300") || strings.Contains(string(encoded), "private_expected") || string(encoded) != "{\"id\":\"_demo-add\"}" {
		t.Fatalf("private spec leaked through JSON: %s", encoded)
	}
	if s.ExerciseID() != "_demo-add" {
		t.Fatal("exercise id mismatch")
	}
}

func TestTrustedDemoReferenceCompiles(t *testing.T) {
	s := demo(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.27.2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "solution.go"), []byte("package main\nfunc Add(a,b int)int{return a+b}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "academy_driver.go"), []byte(s.driver), 0600); err != nil {
		t.Fatal(err)
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goBin, "build", "-o", filepath.Join(dir, "app"), "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("trusted fixture build: %v\n%s", err, out)
	}
}

func TestValidateSourceGuards(t *testing.T) {
	s := demo(t)
	cases := map[string]string{
		"nil source":         "",
		"wrong signature":    "package main\nfunc Add(a int, b string) int{return a}",
		"missing function":   "package main\nfunc helper(){}",
		"reserved main":      "package main\nfunc Add(a,b int)int{return a+b}\nfunc main(){}",
		"init":               "package main\nfunc Add(a,b int)int{return a+b}\nfunc init(){}",
		"top var":            "package main\nvar x int\nfunc Add(a,b int)int{return a+b}",
		"predeclared append": "package main\nfunc append(a,b int)int{return a+b}\nfunc Add(a,b int)int{return a+b}",
		"predeclared cap":    "package main\nfunc cap(a int)int{return a}\nfunc Add(a,b int)int{return a+b}",
		"predeclared alias":  "package main\ntype any = int\nfunc Add(a,b int)int{return a+b}",
		"forbidden import":   "package main\nimport \"os/exec\"\nfunc Add(a,b int)int{return a+b}",
		"not whitelisted":    "package main\nimport \"math/rand\"\nfunc Add(a,b int)int{return a+b}",
		"dot import":         "package main\nimport . \"fmt\"\nfunc Add(a,b int)int{return 0}",
		"reserved driver":    "package main\nfunc Add(a,b int)int{return a+b}",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"solution.go": src}
			if name == "nil source" {
				files = nil
			}
			if name == "reserved driver" {
				files["academy_driver.go"] = src
			}
			if err := s.ValidateSource(files); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	for name, src := range map[string]string{
		"helper type":               "package main\ntype Count int\nfunc Add(a,b int)int{return a+b}",
		"generic helper":            "package main\nfunc identity[T any](v T)T{return v}\nfunc Add(a,b int)int{return identity(a+b)}",
		"different parameter names": "package main\nfunc Add(x,y int)int{return x+y}",
	} {
		t.Run(name, func(t *testing.T) {
			if err := s.ValidateSource(map[string]string{"solution.go": src}); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := s.ValidateSource(map[string]string{}); err == nil {
		t.Fatal("empty file map accepted")
	}
}

func TestIDsAndGroupedSignatureEquivalence(t *testing.T) {
	for _, id := range []string{"zero-intro", "junior-functions", "juniorplus-slices", "middle-http", "middleplus-testing", "senior-runtime"} {
		if !validID(id) {
			t.Errorf("valid future exercise id rejected: %q", id)
		}
	}
	for _, id := range []string{"junior-", "other-functions", "middle-UPPER", "zero-a/../b", strings.Repeat("a", 81)} {
		if validID(id) {
			t.Errorf("invalid exercise id accepted: %q", id)
		}
	}
	s := demo(t)
	s.signature = "func Add(left int, right int) (result int)"
	if err := s.BindReference("package main\nfunc Add(x, y int) (n int) { return x+y }"); err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateSource(map[string]string{"solution.go": "package main\nfunc Add(a, b int) (sum int) { return a+b }"}); err != nil {
		t.Fatalf("grouped parameters or renamed named result were treated as different: %v", err)
	}
}

func TestReferenceBoundaryTypesAndSerializationHooks(t *testing.T) {
	ref := "package main\ntype Detail struct { Code int }\ntype Summary struct { Count int `json:\"count\"`; Label string; Detail Detail }\nfunc Score(s Summary) int { return s.Count+s.Detail.Code }"
	s := demo(t)
	s.signature = "func Score(s Summary) int"
	if err := s.BindReference(ref); err != nil {
		t.Fatal(err)
	}
	valid := "package main\ntype Detail struct { Code int }\ntype Summary struct { Count int `json:\"count\"`; Label string; Detail Detail }\ntype Other struct { Note string }\nfunc helper(o Other) string { return o.Note }\nfunc Score(s Summary) int { return s.Count+s.Detail.Code }"
	if err := s.ValidateSource(map[string]string{"solution.go": valid}); err != nil {
		t.Fatalf("equivalent boundary and unrelated helper type rejected: %v", err)
	}
	for name, changed := range map[string]string{
		"field type":  strings.Replace(valid, "Count int", "Count string", 1),
		"field tag":   strings.Replace(valid, `json:"count"`, `json:"total"`, 1),
		"added field": strings.Replace(valid, "Label string; Detail Detail", "Label string; Extra bool; Detail Detail", 1),
		"nested type": strings.Replace(valid, "Code int", "Code string", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := s.ValidateSource(map[string]string{"solution.go": changed}); err == nil {
				t.Fatal("changed contract type accepted")
			}
		})
	}
	for _, method := range []string{"MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText", "MarshalJSONTo", "UnmarshalJSONFrom"} {
		t.Run(method, func(t *testing.T) {
			src := valid + "\nfunc (s Summary) " + method + "() {}"
			if err := s.ValidateSource(map[string]string{"solution.go": src}); err == nil {
				t.Fatalf("custom serialization method %s accepted", method)
			}
		})
	}
	if err := s.BindReference("package main\ntype Summary struct { Count int }\nfunc Score(s Summary) string { return \"\" }"); err == nil {
		t.Fatal("reference with wrong required signature accepted")
	}
}

func TestCheckUsesStrictJSONAndGenericNames(t *testing.T) {
	s := demo(t)
	good := runner.Result{Stdout: "3\n3\n", ExitCode: 0, Status: "ok"}
	checks := s.Check(good, false)
	if len(checks) != 2 || !checks[0].Passed || !checks[1].Passed {
		t.Fatalf("expected public pass: %#v", checks)
	}
	if got := s.PublicOutput(); got != "3\n3\n" {
		t.Fatalf("public output should match runner lines, got %q", got)
	}
	for _, r := range []runner.Result{
		{Stdout: "3\n3\nextra\n", ExitCode: 0, Status: "ok"},
		{Stdout: "3\n4\n", ExitCode: 0, Status: "ok"},
		{Stdout: "3\n3\n", Stderr: "warning", ExitCode: 0, Status: "ok"},
		{Stdout: "3\n3\n", ExitCode: 1, Status: "error"},
	} {
		allPassed := true
		for _, c := range s.Check(r, false) {
			allPassed = allPassed && c.Passed
		}
		if allPassed {
			t.Fatalf("unexpected all-pass: %#v", r)
		}
	}
	hidden := s.Check(runner.Result{Stdout: "3\n3\n300\n", ExitCode: 0, Status: "ok"}, true)
	if len(hidden) != 3 || !hidden[0].Passed || !hidden[1].Passed || !hidden[2].Passed || strings.Contains(hidden[2].Name, "100") || strings.Contains(hidden[2].Name, "300") {
		t.Fatalf("private check leaked or failed: %#v", hidden)
	}
	// A private case passing cannot hide a public failure during submission.
	failedPublic := s.Check(runner.Result{Stdout: "4\n3\n300\n", ExitCode: 0, Status: "ok"}, true)
	allPassed := true
	for _, c := range failedPublic {
		allPassed = allPassed && c.Passed
	}
	if allPassed || failedPublic[2].Passed == false {
		t.Fatalf("submission did not independently check public and private cases: %#v", failedPublic)
	}
}

func TestRejectUnusableCombinedVectors(t *testing.T) {
	s := demo(t)
	large := json.RawMessage(`"` + strings.Repeat("x", 2500) + `"`)
	s.publicInput = []json.RawMessage{large}
	s.privateInput = []json.RawMessage{large}
	if err := s.validate(); err == nil {
		t.Fatal("combined input over runner argument limit accepted")
	}
	s = demo(t)
	largeExpected := json.RawMessage(`"` + strings.Repeat("x", 4094) + `"`)
	s.publicExpected = make([]json.RawMessage, 12)
	s.privateExpected = make([]json.RawMessage, 12)
	for i := range s.publicExpected {
		s.publicExpected[i] = largeExpected
		s.privateExpected[i] = largeExpected
	}
	if err := s.validate(); err == nil {
		t.Fatal("combined expected output over runner output limit accepted")
	}
}
