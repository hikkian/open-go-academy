package store

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrExamNotFound        = errors.New("exam not found")
	ErrExamAttemptNotFound = errors.New("exam attempt not found")
	ErrExamAttemptConflict = errors.New("exam attempt state conflict")
	ErrExamPrerequisites   = errors.New("exam prerequisites are not verified")
	ErrExamToken           = errors.New("invalid exam attempt token")
)

const MaxExamPayloadBytes = 1024 * 1024

type ExamAttempt struct {
	ID           string     `json:"id"`
	ProfileID    string     `json:"profile_id"`
	ExamID       string     `json:"exam_id"`
	Version      string     `json:"version"`
	Rank         string     `json:"rank"`
	CriteriaHash string     `json:"criteria_hash"`
	Status       string     `json:"status"`
	Revision     int64      `json:"revision"`
	AnswersJSON  string     `json:"answers_json"`
	CodesJSON    string     `json:"codes_json"`
	Score        int        `json:"score"`
	Passed       bool       `json:"passed"`
	CreatedAt    time.Time  `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	ChecksJSON   string     `json:"checks_json"`
	EvidenceJSON string     `json:"evidence_json"`
}

type RankAward struct {
	ProfileID string    `json:"profile_id"`
	ExamID    string    `json:"exam_id"`
	Version   string    `json:"version"`
	Rank      string    `json:"rank"`
	AttemptID string    `json:"attempt_id"`
	AwardedAt time.Time `json:"awarded_at"`
}

func validExamText(s string) bool { return strings.TrimSpace(s) != "" && len(s) <= 256 }
func validRank(rank string) bool {
	switch rank {
	case "Zero", "Junior", "Junior+", "Middle", "Middle+", "Senior":
		return true
	}
	return false
}

// RegisterExam records immutable criteria for a particular exam version.
func (s *Store) RegisterExam(id, version, rank, criteriaHash string) error {
	if !validExamText(id) || !validExamText(version) || !validRank(rank) || !validExamText(criteriaHash) {
		return errors.New("invalid exam registration")
	}
	_, err := s.db.Exec(`INSERT INTO exams(id,version,rank,criteria_hash) VALUES(?,?,?,?) ON CONFLICT(id,version) DO NOTHING`, id, version, rank, criteriaHash)
	if err != nil {
		return err
	}
	var gotRank, gotHash string
	if err = s.db.QueryRow(`SELECT rank,criteria_hash FROM exams WHERE id=? AND version=?`, id, version).Scan(&gotRank, &gotHash); err != nil {
		return err
	}
	if gotRank != rank || gotHash != criteriaHash {
		return errors.New("exam version is immutable")
	}
	return nil
}

func jsonPayload(raw string) (string, error) {
	if len(raw) > MaxExamPayloadBytes {
		return "", errors.New("exam payload too large")
	}
	if !json.Valid([]byte(raw)) {
		return "", errors.New("exam payload must be valid JSON")
	}
	return raw, nil
}
func newExamAttemptID() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
func attemptFromRow(row interface{ Scan(...any) error }) (ExamAttempt, error) {
	var a ExamAttempt
	var passed int
	var created string
	var finished sql.NullString
	err := row.Scan(&a.ID, &a.ProfileID, &a.ExamID, &a.Version, &a.Rank, &a.CriteriaHash, &a.Status, &a.AnswersJSON, &a.CodesJSON, &a.Score, &passed, &created, &finished, &a.ChecksJSON, &a.EvidenceJSON, &a.Revision)
	if err != nil {
		return ExamAttempt{}, err
	}
	a.Passed = passed == 1
	a.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return ExamAttempt{}, err
	}
	if finished.Valid {
		t, e := time.Parse(time.RFC3339Nano, finished.String)
		if e != nil {
			return ExamAttempt{}, e
		}
		a.FinishedAt = &t
	}
	return a, nil
}

const examAttemptColumns = `id,profile_id,exam_id,version,rank,criteria_hash,status,answers_json,codes_json,score,passed,created_at,finished_at,checks_json,evidence_json,revision`

func (s *Store) StartExam(profileID, examID, version, tokenHash string, prerequisites []string, requiredRanks ...string) (ExamAttempt, error) {
	if profileID != "local" {
		return ExamAttempt{}, ErrProfileNotFound
	}
	if !validExamText(examID) || !validExamText(version) || !validExamText(tokenHash) {
		return ExamAttempt{}, errors.New("invalid exam attempt request")
	}
	rankEvidence := make(map[string]rankAwardEvidence, len(requiredRanks))
	for _, required := range requiredRanks {
		if !validRank(required) {
			return ExamAttempt{}, fmt.Errorf("%w: unknown rank %q", ErrExamPrerequisites, required)
		}
		if _, duplicate := rankEvidence[required]; duplicate {
			return ExamAttempt{}, fmt.Errorf("%w: duplicate rank %q", ErrExamPrerequisites, required)
		}
		rankEvidence[required] = rankAwardEvidence{}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ExamAttempt{}, err
	}
	defer tx.Rollback()
	var profile string
	if err = tx.QueryRow(`SELECT id FROM profiles WHERE id=?`, profileID).Scan(&profile); errors.Is(err, sql.ErrNoRows) {
		return ExamAttempt{}, ErrProfileNotFound
	} else if err != nil {
		return ExamAttempt{}, err
	}
	var rank, hash string
	if err = tx.QueryRow(`SELECT rank,criteria_hash FROM exams WHERE id=? AND version=?`, examID, version).Scan(&rank, &hash); errors.Is(err, sql.ErrNoRows) {
		return ExamAttempt{}, ErrExamNotFound
	} else if err != nil {
		return ExamAttempt{}, err
	}
	verified := make(map[string]int64, len(prerequisites))
	for _, prereq := range prerequisites {
		var attemptID int64
		if err = tx.QueryRow(`SELECT attempt_id FROM verified_progress WHERE exercise_id=?`, prereq).Scan(&attemptID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ExamAttempt{}, err
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ExamAttempt{}, fmt.Errorf("%w: %s", ErrExamPrerequisites, prereq)
		}
		verified[prereq] = attemptID
	}
	for required := range rankEvidence {
		var proof rankAwardEvidence
		err = tx.QueryRow(`SELECT exam_id,version,attempt_id FROM rank_awards WHERE profile_id=? AND rank=? ORDER BY awarded_at,exam_id,version LIMIT 1`, profileID, required).Scan(&proof.ExamID, &proof.Version, &proof.AttemptID)
		if errors.Is(err, sql.ErrNoRows) {
			return ExamAttempt{}, fmt.Errorf("%w: rank %s", ErrExamPrerequisites, required)
		}
		if err != nil {
			return ExamAttempt{}, err
		}
		rankEvidence[required] = proof
	}
	id, err := newExamAttemptID()
	if err != nil {
		return ExamAttempt{}, err
	}
	now := time.Now().UTC()
	evidence, _ := json.Marshal(map[string]any{"verified_progress": verified, "rank_awards": rankEvidence})
	_, err = tx.Exec(`INSERT INTO exam_attempts(id,profile_id,exam_id,version,rank,criteria_hash,token_hash,status,created_at,evidence_json) VALUES(?,?,?,?,?,?,?,'active',?,?)`, id, profileID, examID, version, rank, hash, tokenHash, now.Format(time.RFC3339Nano), string(evidence))
	if err != nil {
		if isConstraint(err) {
			return ExamAttempt{}, ErrExamAttemptConflict
		}
		return ExamAttempt{}, err
	}
	if err = tx.Commit(); err != nil {
		return ExamAttempt{}, err
	}
	return s.ExamAttempt(profileID, id)
}

func (s *Store) ClaimExamAttempt(profileID, examID, attemptID, tokenHash string, expectedRevision int64, answersJSON, codesJSON string) (ExamAttempt, error) {
	if profileID != "local" {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	answers, err := jsonPayload(answersJSON)
	if err != nil {
		return ExamAttempt{}, err
	}
	codes, err := jsonPayload(codesJSON)
	if err != nil {
		return ExamAttempt{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ExamAttempt{}, err
	}
	defer tx.Rollback()
	var storedToken, storedExam, status, evidenceJSON string
	var revision int64
	err = tx.QueryRow(`SELECT token_hash,exam_id,status,revision,evidence_json FROM exam_attempts WHERE profile_id=? AND id=?`, profileID, attemptID).Scan(&storedToken, &storedExam, &status, &revision, &evidenceJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	if err != nil {
		return ExamAttempt{}, err
	}
	if subtle.ConstantTimeCompare([]byte(storedToken), []byte(tokenHash)) != 1 {
		return ExamAttempt{}, ErrExamToken
	}
	if storedExam != examID || status != "active" || revision != expectedRevision {
		return ExamAttempt{}, ErrExamAttemptConflict
	}
	if err = validateExamEvidenceTx(tx, profileID, evidenceJSON); err != nil {
		return ExamAttempt{}, err
	}
	r, err := tx.Exec(`UPDATE exam_attempts SET status='grading',answers_json=?,codes_json=?,revision=revision+1 WHERE id=? AND profile_id=? AND status='active' AND revision=?`, answers, codes, attemptID, profileID, expectedRevision)
	if err != nil {
		return ExamAttempt{}, err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ExamAttempt{}, ErrExamAttemptConflict
	}
	if err = tx.Commit(); err != nil {
		return ExamAttempt{}, err
	}
	return s.ExamAttempt(profileID, attemptID)
}

func (s *Store) SaveExamDraft(profileID, attemptID, tokenHash string, expectedRevision int64, answersJSON, codesJSON string) (ExamAttempt, error) {
	if profileID != "local" {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	answers, err := jsonPayload(answersJSON)
	if err != nil {
		return ExamAttempt{}, err
	}
	codes, err := jsonPayload(codesJSON)
	if err != nil {
		return ExamAttempt{}, err
	}
	res, err := s.db.Exec(`UPDATE exam_attempts SET answers_json=?,codes_json=?,revision=revision+1 WHERE profile_id=? AND id=? AND status='active' AND token_hash=? AND revision=?`, answers, codes, profileID, attemptID, tokenHash, expectedRevision)
	if err != nil {
		return ExamAttempt{}, err
	}
	n, _ := res.RowsAffected()
	if n == 1 {
		return s.ExamAttempt(profileID, attemptID)
	}
	var exists int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM exam_attempts WHERE profile_id=? AND id=?`, profileID, attemptID).Scan(&exists); err != nil {
		return ExamAttempt{}, err
	}
	if exists == 0 {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	return ExamAttempt{}, ErrExamAttemptConflict
}

// FinishExamAttempt persists a trusted server-side evaluation. A pass must have a perfect score.
func (s *Store) FinishExamAttempt(profileID, attemptID string, score int, passed bool, checksJSON string) (ExamAttempt, error) {
	if profileID != "local" {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	if score < 0 || score > 100 || passed != (score == 100) {
		return ExamAttempt{}, errors.New("inconsistent exam result")
	}
	checks, err := jsonPayload(checksJSON)
	if err != nil {
		return ExamAttempt{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ExamAttempt{}, err
	}
	defer tx.Rollback()
	a, err := attemptFromRow(tx.QueryRow(`SELECT `+examAttemptColumns+` FROM exam_attempts WHERE profile_id=? AND id=?`, profileID, attemptID))
	if errors.Is(err, sql.ErrNoRows) {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	if err != nil {
		return ExamAttempt{}, err
	}
	if a.Status != "grading" {
		return ExamAttempt{}, ErrExamAttemptConflict
	}
	var registeredRank, registeredHash string
	if err = tx.QueryRow(`SELECT rank,criteria_hash FROM exams WHERE id=? AND version=?`, a.ExamID, a.Version).Scan(&registeredRank, &registeredHash); err != nil {
		return ExamAttempt{}, err
	}
	if registeredRank != a.Rank || registeredHash != a.CriteriaHash {
		return ExamAttempt{}, errors.New("exam criteria snapshot mismatch")
	}
	if passed {
		if err = validateExamEvidenceTx(tx, profileID, a.EvidenceJSON); err != nil {
			return ExamAttempt{}, err
		}
	}
	now := time.Now().UTC()
	_, err = tx.Exec(`UPDATE exam_attempts SET status='completed',score=?,passed=?,checks_json=?,finished_at=? WHERE id=? AND status='grading'`, score, boolInt(passed), checks, now.Format(time.RFC3339Nano), attemptID)
	if err != nil {
		return ExamAttempt{}, err
	}
	if passed {
		_, err = tx.Exec(`INSERT OR IGNORE INTO rank_awards(profile_id,exam_id,version,rank,attempt_id,awarded_at) VALUES(?,?,?,?,?,?)`, profileID, a.ExamID, a.Version, a.Rank, attemptID, now.Format(time.RFC3339Nano))
		if err != nil {
			return ExamAttempt{}, err
		}
		if err = updateProfileRankTx(tx, profileID, a.Rank); err != nil {
			return ExamAttempt{}, err
		}
	}
	if err = pruneExamAttemptsTx(tx, profileID); err != nil {
		return ExamAttempt{}, err
	}
	if err = tx.Commit(); err != nil {
		return ExamAttempt{}, err
	}
	return s.ExamAttempt(profileID, attemptID)
}
func rankLevel(rank string) int {
	for i, r := range []string{"Zero", "Junior", "Junior+", "Middle", "Middle+", "Senior"} {
		if rank == r {
			return i
		}
	}
	return -1
}
func updateProfileRankTx(tx *sql.Tx, profileID, rank string) error {
	var old string
	if e := tx.QueryRow(`SELECT rank FROM profiles WHERE id=?`, profileID).Scan(&old); e != nil {
		return e
	}
	if rankLevel(rank) > rankLevel(old) {
		_, e := tx.Exec(`UPDATE profiles SET rank=? WHERE id=?`, rank, profileID)
		return e
	}
	return nil
}

func validateExamEvidenceTx(tx *sql.Tx, profileID, evidenceJSON string) error {
	var evidence struct {
		VerifiedProgress map[string]int64             `json:"verified_progress"`
		RankAwards       map[string]rankAwardEvidence `json:"rank_awards"`
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &evidence); err != nil {
		return fmt.Errorf("invalid exam evidence: %w", err)
	}
	for exerciseID, attemptID := range evidence.VerifiedProgress {
		var current int64
		if err := tx.QueryRow(`SELECT attempt_id FROM verified_progress WHERE exercise_id=?`, exerciseID).Scan(&current); err != nil {
			return fmt.Errorf("exam prerequisite evidence missing: %w", err)
		}
		if current != attemptID {
			return errors.New("exam prerequisite evidence changed")
		}
	}
	for rank, proof := range evidence.RankAwards {
		var currentExam, currentVersion, currentAttempt string
		if err := tx.QueryRow(`SELECT exam_id,version,attempt_id FROM rank_awards WHERE profile_id=? AND rank=? AND exam_id=? AND version=?`, profileID, rank, proof.ExamID, proof.Version).Scan(&currentExam, &currentVersion, &currentAttempt); err != nil {
			return fmt.Errorf("exam rank prerequisite evidence missing: %w", err)
		}
		if currentExam != proof.ExamID || currentVersion != proof.Version || currentAttempt != proof.AttemptID {
			return errors.New("exam rank prerequisite evidence changed")
		}
	}
	return nil
}

type rankAwardEvidence struct {
	ExamID    string `json:"exam_id"`
	Version   string `json:"version"`
	AttemptID string `json:"attempt_id"`
}

func pruneExamAttemptsTx(tx *sql.Tx, profileID string) error {
	_, err := tx.Exec(`DELETE FROM exam_attempts
		WHERE profile_id=? AND status IN ('completed','interrupted','cancelled')
		AND id NOT IN (SELECT id FROM exam_attempts WHERE profile_id=? AND status IN ('completed','interrupted','cancelled') ORDER BY created_at DESC,id DESC LIMIT 100)
		AND id NOT IN (SELECT attempt_id FROM rank_awards)`, profileID, profileID)
	return err
}

func (s *Store) InterruptExamAttempt(profileID, attemptID string) error {
	if profileID != "local" {
		return ErrExamAttemptNotFound
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	r, e := tx.Exec(`UPDATE exam_attempts SET status='interrupted' WHERE profile_id=? AND id=? AND status='grading'`, profileID, attemptID)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 1 {
		if e = pruneExamAttemptsTx(tx, profileID); e != nil {
			return e
		}
		return tx.Commit()
	}
	return ErrExamAttemptConflict
}
func (s *Store) CancelExamAttempt(profileID, attemptID string) error {
	if profileID != "local" {
		return ErrExamAttemptNotFound
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	r, e := tx.Exec(`UPDATE exam_attempts SET status='cancelled' WHERE profile_id=? AND id=? AND status='active'`, profileID, attemptID)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 1 {
		if e = pruneExamAttemptsTx(tx, profileID); e != nil {
			return e
		}
		return tx.Commit()
	}
	return ErrExamAttemptConflict
}
func (s *Store) RecoverExamAttempts() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE exam_attempts SET status='interrupted' WHERE status='grading'`); e != nil {
		return e
	}
	if e = pruneExamAttemptsTx(tx, "local"); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) ExamAttempt(profileID, id string) (ExamAttempt, error) {
	if profileID != "local" {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	a, e := attemptFromRow(s.db.QueryRow(`SELECT `+examAttemptColumns+` FROM exam_attempts WHERE profile_id=? AND id=?`, profileID, id))
	if errors.Is(e, sql.ErrNoRows) {
		return ExamAttempt{}, ErrExamAttemptNotFound
	}
	return a, e
}
func (s *Store) ExamAttempts(profileID string) ([]ExamAttempt, error) {
	if profileID != "local" {
		return nil, ErrProfileNotFound
	}
	rows, e := s.db.Query(`SELECT `+examAttemptColumns+` FROM (SELECT * FROM exam_attempts WHERE profile_id=? ORDER BY created_at DESC,id DESC LIMIT 100) ORDER BY created_at,id`, profileID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ExamAttempt{}
	for rows.Next() {
		a, e := attemptFromRow(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) RankAwards(profileID string) ([]RankAward, error) {
	if profileID != "local" {
		return nil, ErrProfileNotFound
	}
	rows, e := s.db.Query(`SELECT profile_id,exam_id,version,rank,attempt_id,awarded_at FROM rank_awards WHERE profile_id=? ORDER BY awarded_at,exam_id,version`, profileID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []RankAward{}
	for rows.Next() {
		var a RankAward
		var t string
		if e = rows.Scan(&a.ProfileID, &a.ExamID, &a.Version, &a.Rank, &a.AttemptID, &t); e != nil {
			return nil, e
		}
		a.AwardedAt, e = time.Parse(time.RFC3339Nano, t)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
