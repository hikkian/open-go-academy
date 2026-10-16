package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"sort"
	"strings"
	"time"

	"academy/internal/projects"
	"academy/internal/runner"
)

const ProjectArtifactSchemaVersion = 1

var ErrProjectEvidenceNotFound = errors.New("project evidence not found")

type ProjectSubmission struct {
	ID                 string            `json:"id"`
	ProfileID          string            `json:"profile_id"`
	ProjectID          string            `json:"project_id"`
	ProjectVersion     string            `json:"project_version"`
	CriteriaHash       string            `json:"criteria_hash"`
	ArtifactSchema     int               `json:"artifact_schema_version"`
	ArtifactHash       string            `json:"artifact_hash"`
	ExecutedSourceHash string            `json:"executed_source_hash"`
	Files              map[string]string `json:"-"`
	ExecutedSource     map[string]string `json:"-"`
	CreatedAt          time.Time         `json:"created_at"`
}

type ProjectEvaluationProfile struct {
	ProjectVersion    string
	CriteriaHash      string
	CheckerVersion    string
	CheckerDigest     string
	RuntimePolicyHash string
	RubricVersion     string
	RubricHash        string
}

type ProjectEvaluation struct {
	ID                string    `json:"id"`
	ProfileID         string    `json:"profile_id"`
	ProjectID         string    `json:"project_id"`
	SubmissionID      string    `json:"submission_id"`
	CheckerVersion    string    `json:"checker_version"`
	CheckerDigest     string    `json:"checker_digest"`
	RuntimePolicyHash string    `json:"runtime_policy_hash"`
	RubricVersion     string    `json:"rubric_version"`
	RubricHash        string    `json:"rubric_hash"`
	EvaluationStatus  string    `json:"evaluation_status"`
	ExitCode          int       `json:"exit_code"`
	Passed            bool      `json:"passed"`
	ChecksJSON        string    `json:"checks_json"`
	ChecksHash        string    `json:"checks_hash"`
	CreatedAt         time.Time `json:"created_at"`
}

// RecordProjectSubmission freezes both the complete accepted artifact and the
// exact source subset sent to the isolated executor. Hashes are server-computed.
func (s *Store) RecordProjectSubmission(profileID, projectID string, files, executedSource map[string]string) (ProjectSubmission, error) {
	if profileID != "local" {
		return ProjectSubmission{}, ErrProfileNotFound
	}
	project, ok := projects.Find(projectID)
	if !ok || project.Version == "" || len(files) == 0 || len(executedSource) == 0 {
		return ProjectSubmission{}, errors.New("invalid project submission")
	}
	if err := projects.ValidateFor(projectID, files); err != nil {
		return ProjectSubmission{}, err
	}
	if err := validateExecutedSubset(files, executedSource); err != nil {
		return ProjectSubmission{}, err
	}
	criteriaHash, ok := projects.CriteriaHash(projectID)
	if !ok {
		return ProjectSubmission{}, errors.New("project criteria are unavailable")
	}
	artifactHash := projectArtifactHash(projectID, project.Version, files)
	executedHash := projectArtifactHash(projectID, project.Version, executedSource)
	filesJSON, err := encodeFiles(files)
	if err != nil {
		return ProjectSubmission{}, err
	}
	executedJSON, err := encodeFiles(executedSource)
	if err != nil {
		return ProjectSubmission{}, err
	}
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return ProjectSubmission{}, err
	}
	defer tx.Rollback()
	var existing ProjectSubmission
	var storedFiles, storedExecuted, storedCreated string
	err = tx.QueryRow(`SELECT id,profile_id,project_id,project_version,criteria_hash,artifact_schema_version,
		artifact_hash,executed_source_hash,files_json,executed_source_json,created_at
		FROM project_submissions WHERE profile_id=? AND project_id=? AND project_version=? AND criteria_hash=?
		AND artifact_hash=? AND executed_source_hash=?`, profileID, projectID, project.Version, criteriaHash,
		artifactHash, executedHash).Scan(&existing.ID, &existing.ProfileID, &existing.ProjectID,
		&existing.ProjectVersion, &existing.CriteriaHash, &existing.ArtifactSchema, &existing.ArtifactHash,
		&existing.ExecutedSourceHash, &storedFiles, &storedExecuted, &storedCreated)
	if err == nil {
		existing.Files, err = decodeFiles(storedFiles)
		if err == nil {
			existing.ExecutedSource, err = decodeFiles(storedExecuted)
		}
		if err == nil && (string(filesJSON) != storedFiles || string(executedJSON) != storedExecuted) {
			err = errors.New("project artifact hash collision")
		}
		if err != nil {
			return ProjectSubmission{}, err
		}
		existing.CreatedAt, err = time.Parse(time.RFC3339Nano, storedCreated)
		if err != nil {
			return ProjectSubmission{}, err
		}
		if err = tx.Commit(); err != nil {
			return ProjectSubmission{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ProjectSubmission{}, err
	}
	id, err := newExamAttemptID()
	if err != nil {
		return ProjectSubmission{}, err
	}
	_, err = tx.Exec(`INSERT INTO project_submissions(
		id,profile_id,project_id,project_version,criteria_hash,artifact_schema_version,
		artifact_hash,executed_source_hash,files_json,executed_source_json,created_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, profileID, projectID, project.Version, criteriaHash,
		ProjectArtifactSchemaVersion, artifactHash, executedHash, string(filesJSON), string(executedJSON), now.Format(time.RFC3339Nano))
	if err != nil {
		return ProjectSubmission{}, err
	}
	if err = tx.Commit(); err != nil {
		return ProjectSubmission{}, err
	}
	return ProjectSubmission{ID: id, ProfileID: profileID, ProjectID: projectID, ProjectVersion: project.Version,
		CriteriaHash: criteriaHash, ArtifactSchema: ProjectArtifactSchemaVersion, ArtifactHash: artifactHash,
		ExecutedSourceHash: executedHash, Files: cloneFiles(files), ExecutedSource: cloneFiles(executedSource), CreatedAt: now}, nil
}

// RecordProjectEvaluation persists an immutable result from the trusted local
// runner. Replaying the exact evaluation is idempotent; conflicting replays fail.
func (s *Store) RecordProjectEvaluation(profileID, submissionID string, profile ProjectEvaluationProfile, status string, exitCode int, passed bool, checksJSON string) (ProjectEvaluation, error) {
	if profileID != "local" {
		return ProjectEvaluation{}, ErrProfileNotFound
	}
	if !validProjectEvaluationProfileShape(profile) {
		return ProjectEvaluation{}, errors.New("invalid trusted project evaluation profile")
	}
	if status != "automatic_pass" && status != "failed" && status != "error" {
		return ProjectEvaluation{}, errors.New("invalid project evaluation status")
	}
	checks, err := projectChecks(checksJSON)
	if err != nil {
		return ProjectEvaluation{}, err
	}
	allPassed := len(checks) > 0
	for _, check := range checks {
		allPassed = allPassed && check.Passed
	}
	if allPassed != passed || passed != (status == "automatic_pass" && exitCode == 0) {
		return ProjectEvaluation{}, errors.New("project evaluation result does not match checks")
	}
	checksHash := hashBytes("oga-project-checks-v1", []byte(checksJSON))
	tx, err := s.db.Begin()
	if err != nil {
		return ProjectEvaluation{}, err
	}
	defer tx.Rollback()
	var projectID, projectVersion, criteriaHash string
	if err = tx.QueryRow(`SELECT project_id,project_version,criteria_hash FROM project_submissions WHERE id=? AND profile_id=?`, submissionID, profileID).Scan(&projectID, &projectVersion, &criteriaHash); errors.Is(err, sql.ErrNoRows) {
		return ProjectEvaluation{}, ErrProjectEvidenceNotFound
	} else if err != nil {
		return ProjectEvaluation{}, err
	}
	if !validProjectEvaluationProfile(projectID, profile) || profile.ProjectVersion != projectVersion || profile.CriteriaHash != criteriaHash {
		return ProjectEvaluation{}, errors.New("project evaluation policy does not match submission")
	}
	var existing ProjectEvaluation
	var existingPassed int
	var existingExitCode int
	var existingCreated string
	err = tx.QueryRow(`SELECT id,profile_id,project_id,submission_id,checker_version,checker_digest,
		runtime_policy_hash,rubric_version,rubric_hash,evaluation_status,exit_code,passed,checks_json,checks_hash,created_at
		FROM project_evaluations WHERE submission_id=? AND checker_digest=? AND runtime_policy_hash=? AND rubric_hash=?`,
		submissionID, profile.CheckerDigest, profile.RuntimePolicyHash, profile.RubricHash).Scan(
		&existing.ID, &existing.ProfileID, &existing.ProjectID, &existing.SubmissionID, &existing.CheckerVersion,
		&existing.CheckerDigest, &existing.RuntimePolicyHash, &existing.RubricVersion, &existing.RubricHash,
		&existing.EvaluationStatus, &existingExitCode, &existingPassed, &existing.ChecksJSON, &existing.ChecksHash, &existingCreated)
	if err == nil {
		if existing.CheckerVersion != profile.CheckerVersion || existing.RubricVersion != profile.RubricVersion || existing.ChecksHash != checksHash || (existingPassed == 1) != passed || existing.EvaluationStatus != status || existingExitCode != exitCode {
			return ProjectEvaluation{}, errors.New("conflicting replay for immutable project evaluation")
		}
		existing.Passed = existingPassed == 1
		existing.ExitCode = existingExitCode
		existing.CreatedAt, err = time.Parse(time.RFC3339Nano, existingCreated)
		if err != nil {
			return ProjectEvaluation{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ProjectEvaluation{}, err
	}
	evaluationID, err := newExamAttemptID()
	if err != nil {
		return ProjectEvaluation{}, err
	}
	now := time.Now().UTC()
	_, err = tx.Exec(`INSERT INTO project_evaluations(
		id,profile_id,project_id,submission_id,checker_version,checker_digest,runtime_policy_hash,
		rubric_version,rubric_hash,evaluation_status,exit_code,passed,checks_json,checks_hash,created_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, evaluationID, profileID, projectID, submissionID,
		profile.CheckerVersion, profile.CheckerDigest, profile.RuntimePolicyHash, profile.RubricVersion,
		profile.RubricHash, status, exitCode, boolInt(passed), checksJSON, checksHash, now.Format(time.RFC3339Nano))
	if err != nil {
		return ProjectEvaluation{}, err
	}
	if err = tx.Commit(); err != nil {
		return ProjectEvaluation{}, err
	}
	return ProjectEvaluation{ID: evaluationID, ProfileID: profileID, ProjectID: projectID, SubmissionID: submissionID,
		CheckerVersion: profile.CheckerVersion, CheckerDigest: profile.CheckerDigest,
		RuntimePolicyHash: profile.RuntimePolicyHash, RubricVersion: profile.RubricVersion,
		RubricHash: profile.RubricHash, EvaluationStatus: status, ExitCode: exitCode, Passed: passed,
		ChecksJSON: checksJSON, ChecksHash: checksHash, CreatedAt: now}, nil
}

func (s *Store) ProjectEvaluation(profileID, evaluationID string) (ProjectEvaluation, error) {
	if profileID != "local" {
		return ProjectEvaluation{}, ErrProjectEvidenceNotFound
	}
	var evaluation ProjectEvaluation
	var passed int
	var exitCode int
	var created string
	err := s.db.QueryRow(`SELECT id,profile_id,project_id,submission_id,checker_version,checker_digest,
		runtime_policy_hash,rubric_version,rubric_hash,evaluation_status,exit_code,passed,checks_json,checks_hash,created_at
		FROM project_evaluations WHERE id=? AND profile_id=?`, evaluationID, profileID).Scan(
		&evaluation.ID, &evaluation.ProfileID, &evaluation.ProjectID, &evaluation.SubmissionID,
		&evaluation.CheckerVersion, &evaluation.CheckerDigest, &evaluation.RuntimePolicyHash,
		&evaluation.RubricVersion, &evaluation.RubricHash, &evaluation.EvaluationStatus, &exitCode, &passed, &evaluation.ChecksJSON,
		&evaluation.ChecksHash, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectEvaluation{}, ErrProjectEvidenceNotFound
	}
	if err != nil {
		return ProjectEvaluation{}, err
	}
	evaluation.Passed = passed == 1
	evaluation.ExitCode = exitCode
	evaluation.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return evaluation, err
}

func (s *Store) ProjectSubmission(profileID, submissionID string) (ProjectSubmission, error) {
	if profileID != "local" {
		return ProjectSubmission{}, ErrProjectEvidenceNotFound
	}
	var submission ProjectSubmission
	var filesJSON, executedJSON, created string
	err := s.db.QueryRow(`SELECT id,profile_id,project_id,project_version,criteria_hash,
		artifact_schema_version,artifact_hash,executed_source_hash,files_json,executed_source_json,created_at
		FROM project_submissions WHERE id=? AND profile_id=?`, submissionID, profileID).Scan(
		&submission.ID, &submission.ProfileID, &submission.ProjectID, &submission.ProjectVersion,
		&submission.CriteriaHash, &submission.ArtifactSchema, &submission.ArtifactHash,
		&submission.ExecutedSourceHash, &filesJSON, &executedJSON, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectSubmission{}, ErrProjectEvidenceNotFound
	}
	if err != nil {
		return ProjectSubmission{}, err
	}
	submission.Files, err = decodeFiles(filesJSON)
	if err != nil {
		return ProjectSubmission{}, err
	}
	submission.ExecutedSource, err = decodeFiles(executedJSON)
	if err != nil {
		return ProjectSubmission{}, err
	}
	submission.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return submission, err
}

func ProjectArtifactHash(projectID, projectVersion string, files map[string]string) string {
	return projectArtifactHash(projectID, projectVersion, files)
}

func projectArtifactHash(projectID, projectVersion string, files map[string]string) string {
	h := sha256.New()
	writeHashField(h, []byte("open-go-academy-project-artifact-v1"))
	writeHashField(h, []byte(projectID))
	writeHashField(h, []byte(projectVersion))
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(names)))
	_, _ = h.Write(count[:])
	for _, name := range names {
		writeHashField(h, []byte(name))
		writeHashField(h, []byte(files[name]))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func writeHashField(h hash.Hash, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = h.Write(size[:])
	_, _ = h.Write(value)
}

func hashBytes(domain string, value []byte) string {
	h := sha256.New()
	writeHashField(h, []byte(domain))
	writeHashField(h, value)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func validateExecutedSubset(files, executed map[string]string) error {
	for name, contents := range executed {
		if original, ok := files[name]; !ok || original != contents {
			return fmt.Errorf("executed source %q does not match the submitted artifact", name)
		}
	}
	if len(executed) == 0 {
		return errors.New("empty executed project source")
	}
	return nil
}

func validProjectEvaluationProfileShape(profile ProjectEvaluationProfile) bool {
	return profile.ProjectVersion != "" && validSHA256(profile.CriteriaHash) &&
		validExamText(profile.CheckerVersion) && validImmutableDigest(profile.CheckerDigest) &&
		validSHA256(profile.RuntimePolicyHash) && validExamText(profile.RubricVersion) && validSHA256(profile.RubricHash)
}

func validProjectEvaluationProfile(projectID string, profile ProjectEvaluationProfile) bool {
	if !validProjectEvaluationProfileShape(profile) {
		return false
	}
	switch projectID {
	case "p3":
		return profile.CheckerVersion == "taskflow-v1" && profile.RubricVersion == "taskflow-automatic-v1"
	case "p4":
		return profile.CheckerVersion == "scalelab-deterministic-v1" && profile.RubricVersion == "scalelab-deterministic-v1"
	case "p5":
		return profile.CheckerVersion == "capstone-deterministic-v1" && profile.RubricVersion == "capstone-deterministic-v1"
	default:
		return false
	}
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validImmutableDigest(value string) bool {
	index := strings.LastIndex(value, "@sha256:")
	if index >= 0 {
		return validSHA256(value[index+1:]) && index > 0
	}
	return validSHA256(value)
}

func projectChecks(raw string) ([]runner.Check, error) {
	if len(raw) > 64*1024 || !json.Valid([]byte(raw)) {
		return nil, errors.New("invalid project evaluation checks")
	}
	var checks []runner.Check
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&checks); err != nil {
		return nil, errors.New("invalid project evaluation checks")
	}
	return checks, nil
}

func cloneFiles(files map[string]string) map[string]string {
	clone := make(map[string]string, len(files))
	for name, contents := range files {
		clone[name] = contents
	}
	return clone
}
