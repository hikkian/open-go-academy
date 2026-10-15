package content

import (
	"fmt"
	"path"
	"strings"
)

type documentationExample struct {
	ID, Mode string
	Files    map[string]string
}

func documentationExamples(markdown string) ([]documentationExample, error) {
	groups := []documentationExample{}
	positions := map[string]int{}
	var code strings.Builder
	var active *documentationExample
	filename := ""
	block := 0
	for _, line := range strings.Split(markdown, "\n") {
		if active == nil {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) == 0 || fields[0] != "```go" {
				continue
			}
			block++
			id := fmt.Sprintf("example-%02d", block)
			mode := "run"
			filename = ""
			for _, attribute := range fields[1:] {
				switch {
				case strings.HasPrefix(attribute, "group="):
					id = strings.TrimPrefix(attribute, "group=")
				case strings.HasPrefix(attribute, "file="):
					filename = strings.TrimPrefix(attribute, "file=")
				case strings.HasPrefix(attribute, "mode="):
					mode = strings.TrimPrefix(attribute, "mode=")
				default:
					return nil, fmt.Errorf("unknown Go example attribute %q", attribute)
				}
			}
			if id == "" || (mode != "run" && mode != "test" && mode != "compile") {
				return nil, fmt.Errorf("invalid Go example group or mode")
			}
			if filename == "" {
				filename = "main.go"
				if mode == "test" {
					filename = "example_test.go"
				}
			}
			if path.IsAbs(filename) || path.Clean(filename) != filename || strings.HasPrefix(filename, "..") || !strings.HasSuffix(filename, ".go") {
				return nil, fmt.Errorf("unsafe Go example path")
			}
			active = &documentationExample{ID: id, Mode: mode, Files: map[string]string{}}
			code.Reset()
			continue
		}
		if strings.TrimSpace(line) == "```" {
			if i, ok := positions[active.ID]; ok {
				if groups[i].Mode != active.Mode || groups[i].Files[filename] != "" {
					return nil, fmt.Errorf("inconsistent or duplicate example group %s", active.ID)
				}
				groups[i].Files[filename] = code.String()
			} else {
				active.Files[filename] = code.String()
				positions[active.ID] = len(groups)
				groups = append(groups, *active)
			}
			active = nil
			continue
		}
		code.WriteString(line)
		code.WriteByte('\n')
	}
	if active != nil {
		return nil, fmt.Errorf("unterminated Go example")
	}
	return groups, nil
}
