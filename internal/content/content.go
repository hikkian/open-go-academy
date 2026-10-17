// Package content loads and validates the bundled, versioned course catalog.
package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

//go:embed data/*.json data/*.md
var assets embed.FS

//go:embed solutions/*/main.go
var solutions embed.FS

// Exercise describes a learner-facing task. Reference solutions live outside this package.
type Exercise struct {
	Kind         string      `json:"kind,omitempty"`
	Skills       []string    `json:"skills,omitempty"`
	ID           string      `json:"id"`
	Difficulty   string      `json:"difficulty"`
	Title        string      `json:"title"`
	Prompt       string      `json:"prompt"`
	StarterFile  string      `json:"starter_file"`
	StarterCode  string      `json:"starter_code"`
	PublicOutput string      `json:"public_output"`
	Criteria     []string    `json:"criteria"`
	SelfCheck    []SelfCheck `json:"self_check"`
}
type SelfCheck struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}
type GlossaryEntry struct {
	Term        string `json:"term"`
	Translation string `json:"translation"`
	Definition  string `json:"definition"`
}
type Lesson struct {
	Order             int             `json:"order,omitempty"`
	CurriculumVersion string          `json:"curriculum_version,omitempty"`
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	Rank              string          `json:"rank"`
	Module            string          `json:"module"`
	Prerequisites     []string        `json:"prerequisites"`
	EstimatedMinutes  int             `json:"estimated_minutes"`
	GoVersion         string          `json:"go_version"`
	LearningOutcomes  []string        `json:"learning_outcomes"`
	Glossary          []GlossaryEntry `json:"glossary"`
	SelfCheck         []SelfCheck     `json:"self_check"`
	MarkdownFile      string          `json:"markdown_file"`
	Body              string          `json:"body"`
	Exercises         []Exercise      `json:"exercises"`
}
type Catalog struct {
	Lessons   []Lesson
	exercises map[string]Exercise
	solutions map[string]string
	grading   map[string]*grading.Spec
}

// Load parses and validates all embedded course material.
func Load() (*Catalog, error) {
	manifest, err := Manifest()
	if err != nil {
		return nil, err
	}
	return loadLessons(manifest.Published)
}

// loadLessons builds and validates a catalog from explicit versioned lesson
// IDs. Production uses the published manifest; acceptance tests can validate
// a candidate catalog before changing that manifest.
func loadLessons(ids []string) (*Catalog, error) {
	entries := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") || path.Clean(id) != id {
			return nil, errors.New("unsafe manifest lesson id")
		}
		entries = append(entries, "data/"+id+".json")
	}
	c := &Catalog{exercises: map[string]Exercise{}, solutions: map[string]string{}}
	gradeIDs := []string{}
	seen := map[string]bool{}
	for _, name := range entries {
		b, err := assets.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var l Lesson
		if err := json.Unmarshal(b, &l); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if l.ID == "" || seen[l.ID] {
			return nil, fmt.Errorf("empty or duplicate lesson id %q", l.ID)
		}
		seen[l.ID] = true
		if l.Rank == "" || l.Module == "" || l.EstimatedMinutes <= 0 || l.GoVersion == "" || len(l.LearningOutcomes) == 0 || len(l.SelfCheck) == 0 {
			return nil, fmt.Errorf("lesson %s has incomplete metadata", l.ID)
		}
		if !safeAssetPath(l.MarkdownFile) {
			return nil, fmt.Errorf("lesson %s has unsafe markdown path", l.ID)
		}
		body, err := assets.ReadFile(path.Join("data", l.MarkdownFile))
		if err != nil {
			return nil, fmt.Errorf("lesson %s markdown: %w", l.ID, err)
		}
		l.Body = string(body)
		if strings.TrimSpace(l.Body) == "" {
			return nil, fmt.Errorf("lesson %s has empty body", l.ID)
		}
		for _, ex := range l.Exercises {
			if err := validateExerciseSkills(ex); err != nil {
				return nil, err
			}
			if ex.Kind != "" && ex.Kind != "function" {
				return nil, fmt.Errorf("exercise %s has unsupported kind", ex.ID)
			}
			if ex.Kind == "function" {
				gradeIDs = append(gradeIDs, ex.ID)
			}
			if ex.ID == "" || ex.Title == "" || ex.Prompt == "" || ex.StarterFile == "" || ex.StarterCode == "" || len(ex.Criteria) == 0 {
				return nil, fmt.Errorf("lesson %s has incomplete exercise", l.ID)
			}
			if _, ok := c.exercises[ex.ID]; ok {
				return nil, fmt.Errorf("duplicate exercise id %s", ex.ID)
			}
			c.exercises[ex.ID] = ex
			ref, err := solutions.ReadFile("solutions/" + ex.ID + "/main.go")
			if err != nil {
				return nil, fmt.Errorf("exercise %s reference solution: %w", ex.ID, err)
			}
			c.solutions[ex.ID] = string(ref)

		}
		c.Lessons = append(c.Lessons, l)
	}
	checks, err := grading.LoadFor(gradeIDs)
	if err != nil {
		return nil, fmt.Errorf("exercise grading: %w", err)
	}
	c.grading = checks
	for _, ex := range c.exercises {
		if ex.Kind != "function" {
			continue
		}
		spec := c.grading[ex.ID]
		if err := spec.BindReference(c.solutions[ex.ID]); err != nil {
			return nil, fmt.Errorf("exercise %s trusted contract: %w", ex.ID, err)
		}
		for _, check := range spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: ex.PublicOutput}, false) {
			if !check.Passed {
				return nil, fmt.Errorf("exercise %s public result differs from executable public cases", ex.ID)
			}
		}
	}
	for _, l := range c.Lessons {
		for _, dep := range l.Prerequisites {
			if !seen[dep] {
				return nil, fmt.Errorf("lesson %s references missing prerequisite %s", l.ID, dep)
			}
		}
	}
	if err := c.checkCycles(); err != nil {
		return nil, err
	}
	for _, l := range c.Lessons {
		if err := c.validateMarkdownLinks(l); err != nil {
			return nil, err
		}
	}
	if len(c.Lessons) == 0 {
		return nil, errors.New("no lessons found")
	}
	if len(ids) != len(c.Lessons) {
		return nil, errors.New("curriculum list differs from loaded lessons")
	}
	for _, id := range ids {
		if !seen[id] {
			return nil, fmt.Errorf("manifest lesson %s is missing", id)
		}
	}
	ranks := map[string]int{"Zero": 0, "Junior": 1, "Junior+": 2, "Middle": 3, "Middle+": 4, "Senior": 5}
	sort.Slice(c.Lessons, func(i, j int) bool {
		a, b := c.Lessons[i], c.Lessons[j]
		if ranks[a.Rank] != ranks[b.Rank] {
			return ranks[a.Rank] < ranks[b.Rank]
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.ID < b.ID
	})
	return c, nil
}
func safeAssetPath(p string) bool {
	return p != "" && !path.IsAbs(p) && path.Clean(p) == p && !strings.HasPrefix(p, "..") && strings.HasSuffix(p, ".md")
}
func (c *Catalog) checkCycles() error {
	byID := map[string]Lesson{}
	for _, l := range c.Lessons {
		byID[l.ID] = l
	}
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("dependency cycle at %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, d := range byID[id].Prerequisites {
			if err := visit(d); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
func (c *Catalog) AllLessons() []Lesson                    { return append([]Lesson(nil), c.Lessons...) }
func (c *Catalog) Grading(id string) (*grading.Spec, bool) { s, ok := c.grading[id]; return s, ok }
func (c *Catalog) FindLesson(id string) (Lesson, bool) {
	for _, l := range c.Lessons {
		if l.ID == id {
			return l, true
		}
	}
	return Lesson{}, false
}
func (c *Catalog) FindExercise(id string) (Exercise, bool) { e, ok := c.exercises[id]; return e, ok }
func (c *Catalog) Search(q string) []Lesson {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Lesson
	for _, l := range c.Lessons {
		if q == "" || strings.Contains(strings.ToLower(l.Title+" "+l.ID+" "+l.Module+" "+l.Body), q) {
			out = append(out, l)
		}
	}
	return out
}

// Reference returns a private reference solution only for an explicit solution request.
func (c *Catalog) Reference(id string) (string, bool) { s, ok := c.solutions[id]; return s, ok }

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^\s)]+)(?:\s+[^)]*)?\)`)

func (c *Catalog) validateMarkdownLinks(l Lesson) error {
	for _, match := range markdownLinkPattern.FindAllStringSubmatch(markdownLinkProse(l.Body), -1) {
		target := match[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "//") {
			continue
		}
		switch {
		case strings.HasPrefix(target, "lesson:"):
			if _, ok := c.FindLesson(strings.TrimPrefix(target, "lesson:")); !ok {
				return fmt.Errorf("lesson %s links to missing lesson %s", l.ID, target)
			}
		case strings.HasPrefix(target, "exercise:"):
			if _, ok := c.FindExercise(strings.TrimPrefix(target, "exercise:")); !ok {
				return fmt.Errorf("lesson %s links to missing exercise %s", l.ID, target)
			}
		case strings.HasPrefix(target, "#"):
			if !hasHeading(l.Body, strings.TrimPrefix(target, "#")) {
				return fmt.Errorf("lesson %s links to missing heading %s", l.ID, target)
			}
		case strings.HasSuffix(strings.SplitN(target, "#", 2)[0], ".md"):
			parts := strings.SplitN(target, "#", 2)
			found := false
			for _, other := range c.Lessons {
				if other.MarkdownFile == parts[0] || path.Base(other.MarkdownFile) == path.Base(parts[0]) {
					found = len(parts) == 1 || hasHeading(other.Body, parts[1])
					break
				}
			}
			if !found {
				return fmt.Errorf("lesson %s links to missing content %s", l.ID, target)
			}
		default:
			return fmt.Errorf("lesson %s has unsupported or unsafe internal link %s", l.ID, target)
		}
	}
	return nil
}

func hasHeading(markdown, slug string) bool {
	for _, line := range strings.Split(markdown, "\n") {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if headingSlug(title) == slug {
			return true
		}
	}
	return false
}

func headingSlug(title string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if r == ' ' || r == '\t' {
			space = b.Len() > 0
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 0x400 && r <= 0x4ff) || r == '-' || r == '_' {
			if space {
				b.WriteByte('-')
				space = false
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Code examples are not Markdown links: generic calls such as F[T](value)
// must not be interpreted as navigation. Rendering has its own safe URL policy.
func markdownLinkProse(body string) string {
	var out strings.Builder
	fence := ""
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			marker := trim[:3]
			if fence == "" {
				fence = marker
			} else if fence == marker {
				fence = ""
			}
			out.WriteByte('\n')
			continue
		}
		if fence != "" {
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return regexp.MustCompile("`+[^`]*`+").ReplaceAllString(out.String(), "")
}
