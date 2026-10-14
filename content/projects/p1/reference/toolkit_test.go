package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunExamples(t *testing.T) {
	t.Run("sum", func(t *testing.T) { assertCommand(t, []string{"sum", "4", "-1", "9"}, "12\n", "", 0) })
	t.Run("int64 minimum and maximum cancellation", func(t *testing.T) {
		assertCommand(t, []string{"sum", "-9223372036854775808", "9223372036854775807"}, "-1\n", "", 0)
	})
	file := filepath.Join(t.TempDir(), "names.json")
	if err := os.WriteFile(file, []byte(`["Ada","Lin","Go"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	assertCommand(t, []string{"inspect", file}, "3\n", "", 0)
	empty := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(empty, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertCommand(t, []string{"inspect", empty}, "0\n", "", 0)
}

func TestRunFailures(t *testing.T) {
	for _, args := range [][]string{{}, {"sum"}, {"sum", "nope"}, {"sum", "9223372036854775807", "1"}, {"inspect", "/missing/file"}} {
		if err := run(args); err == nil {
			t.Errorf("run(%q) succeeded", args)
		}
	}
	for _, input := range []string{`{"no":"array"}`, `["ok",42]`, `null`, `[`} {
		file := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(file, []byte(input), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"inspect", file}); err == nil {
			t.Errorf("invalid JSON %q succeeded", input)
		}
	}
}

func assertCommand(t *testing.T, args []string, stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	gotCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			gotCode = ee.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if gotCode != code || out.String() != stdout || errOut.String() != stderr {
		t.Fatalf("got code=%d stdout=%q stderr=%q; want code=%d stdout=%q stderr=%q", gotCode, out.String(), errOut.String(), code, stdout, stderr)
	}
}

func TestCLIExitCode(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "toolkit")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	badJSON := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badJSON, []byte(`{"not":"strings"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args   []string
		stderr string
	}{
		{nil, "usage: toolkit sum N... | inspect path\n"},
		{[]string{"sum", "bad"}, "invalid integer \"bad\"\n"},
		{[]string{"sum", "9223372036854775807", "1"}, "sum overflows int64\n"},
		{[]string{"inspect", "/missing/file"}, "cannot read \"/missing/file\"\n"},
		{[]string{"inspect", badJSON}, "input must be a JSON array of strings\n"},
	}
	for _, tc := range cases {
		cmd := exec.Command(binary, tc.args...)
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		err := cmd.Run()
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
			t.Errorf("args %q exit = %v; want 2", tc.args, err)
		}
		if out.Len() != 0 || errOut.String() != tc.stderr {
			t.Errorf("args %q output stdout=%q stderr=%q; want empty stdout and %q", tc.args, out.String(), errOut.String(), tc.stderr)
		}
	}
}

func TestToolError(t *testing.T) {
	var err error = &ToolError{Message: "expected"}
	if err.Error() != "expected" {
		t.Fatalf("Error() = %q", err.Error())
	}
}
