package content

import (
	_ "embed"
	"encoding/json"
)

//go:embed manifest.json
var manifestJSON []byte

type CurriculumManifest struct {
	Version            string            `json:"curriculum_version"`
	GoVersion          string            `json:"go_version"`
	DependencyVersions map[string]string `json:"dependency_versions"`
	LastVerifiedAt     string            `json:"last_verified_at"`
	Sources            []string          `json:"source_urls"`
	Compatibility      []string          `json:"compatibility_notes"`
	Published          []string          `json:"published_lessons"`
	Planned            map[string]int    `json:"planned_lessons"`
	Status             string            `json:"status"`
}

func Manifest() (CurriculumManifest, error) {
	var m CurriculumManifest
	err := json.Unmarshal(manifestJSON, &m)
	return m, err
}
