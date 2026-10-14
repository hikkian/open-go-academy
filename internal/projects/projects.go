// Package projects contains the versioned, bundled project curriculum.
package projects

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// Project is the content contract consumed by the project workspace.
type Project struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Rank           string            `json:"rank"`
	Description    string            `json:"description"`
	Requirements   []string          `json:"requirements"`
	Milestones     []Milestone       `json:"milestones"`
	StarterFiles   map[string]string `json:"starter_files"`
	PublicExamples []Example         `json:"public_examples"`
}

type Milestone struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Criteria []string `json:"criteria"`
}

type Example struct {
	Args   []string `json:"args"`
	Stdout string   `json:"stdout"`
}

//go:embed p1/spec.json p1/starter_files/*
var bundled embed.FS

// Load returns the bundled P1 project definition. Its inputs are build-time
// assets, so loading cannot fail at runtime.
func Load() Project {
	var p Project
	b, _ := bundled.ReadFile("p1/spec.json")
	if err := json.Unmarshal(b, &p); err != nil {
		panic("invalid embedded P1 project: " + err.Error())
	}
	p.StarterFiles = map[string]string{"go.mod": "module cli-toolkit\n\ngo 1.27.2\n"}
	entries, _ := fs.ReadDir(bundled, "p1/starter_files")
	for _, entry := range entries {
		b, _ := bundled.ReadFile("p1/starter_files/" + entry.Name())
		p.StarterFiles[entry.Name()] = string(b)
	}
	return p
}

const maxFiles = 16
const maxBytes = 128 * 1024
const maxGoBytes = 64 * 1024
const maxJSONBytes = 64 * 1024
const maxJSONFiles = 8
const requiredGoMod = "module cli-toolkit\n\ngo 1.27.2\n"

var projectFileName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*\.(go|json)$`)

// ValidateFiles enforces the P1 workspace's path, extension, count, and size
// limits before files are persisted or passed to an execution service.
func ValidateFiles(files map[string]string) error {
	if len(files) == 0 {
		return errors.New("project must contain at least one file")
	}
	if len(files) > maxFiles {
		return fmt.Errorf("project exceeds %d files", maxFiles)
	}
	total := 0
	goBytes, jsonBytes, jsonFiles := 0, 0, 0
	hasSource, hasReadme, hasGoMod := false, false, false
	for name, contents := range files {
		if name == "" || strings.ContainsAny(name, "\\\x00\r\n\t") || path.Clean(name) != name || path.IsAbs(name) || strings.HasPrefix(name, "../") || name == "." {
			return fmt.Errorf("unsafe project path %q", name)
		}
		if strings.Contains(name, "/") || (name != "README.md" && name != "go.mod" && !projectFileName.MatchString(name)) {
			return fmt.Errorf("unsupported project file %q", name)
		}
		total += len(contents)
		if total > maxBytes {
			return fmt.Errorf("project exceeds %d bytes", maxBytes)
		}
		switch {
		case strings.HasSuffix(name, ".go"):
			goBytes += len(contents)
			if !strings.HasSuffix(name, "_test.go") {
				hasSource = true
			}
		case strings.HasSuffix(name, ".json"):
			jsonFiles++
			jsonBytes += len(contents)
		case name == "README.md":
			hasReadme = true
			if strings.TrimSpace(contents) == "" {
				return errors.New("README.md must not be empty")
			}
		case name == "go.mod":
			hasGoMod = true
			if !validGoMod(contents) {
				return errors.New("go.mod must contain only module cli-toolkit and go 1.27.2")
			}
		}
		if goBytes > maxGoBytes {
			return fmt.Errorf("Go source files exceed %d bytes", maxGoBytes)
		}
		if jsonBytes > maxJSONBytes {
			return fmt.Errorf("JSON files exceed %d bytes", maxJSONBytes)
		}
		if jsonFiles > maxJSONFiles {
			return fmt.Errorf("project exceeds %d JSON files", maxJSONFiles)
		}
	}
	if !hasSource {
		return errors.New("project must contain at least one non-test Go source file")
	}
	if !hasReadme {
		return errors.New("project must contain README.md")
	}
	if !hasGoMod {
		return errors.New("project must contain go.mod")
	}
	return nil
}

func validGoMod(contents string) bool {
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	lines := make([][]string, 0, 2)
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			lines = append(lines, fields)
		}
	}
	return len(lines) == 2 &&
		len(lines[0]) == 2 && lines[0][0] == "module" && lines[0][1] == "cli-toolkit" &&
		len(lines[1]) == 2 && lines[1][0] == "go" && lines[1][1] == "1.27.2"
}
