package store

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrCredentialNotFound      = errors.New("credential not found")
	ErrCredentialConflict      = errors.New("credential conflict")
	ErrCredentialPrerequisites = errors.New("credential prerequisites are not verified")
)

const MaxCredentialEnvelopeBytes = 32 * 1024

type Credential struct {
	ID           string     `json:"id"`
	ProfileID    string     `json:"profile_id"`
	AttemptID    string     `json:"attempt_id"`
	ExamID       string     `json:"exam_id"`
	Version      string     `json:"version"`
	Rank         string     `json:"rank"`
	CriteriaHash string     `json:"criteria_hash"`
	KeyID        string     `json:"key_id"`
	EnvelopeJSON string     `json:"envelope_json"`
	Published    bool       `json:"published"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func validLowerHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	b, e := hex.DecodeString(s)
	return e == nil && len(b)*2 == n && s == strings.ToLower(s)
}
func credentialFromRow(row interface{ Scan(...any) error }) (Credential, error) {
	var c Credential
	var published int
	var created string
	var revoked sql.NullString
	err := row.Scan(&c.ID, &c.ProfileID, &c.AttemptID, &c.ExamID, &c.Version, &c.Rank, &c.CriteriaHash, &c.KeyID, &c.EnvelopeJSON, &published, &created, &revoked)
	if err != nil {
		return Credential{}, err
	}
	c.Published = published == 1
	c.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Credential{}, err
	}
	if revoked.Valid {
		t, e := time.Parse(time.RFC3339Nano, revoked.String)
		if e != nil {
			return Credential{}, e
		}
		c.RevokedAt = &t
	}
	return c, nil
}

const credentialColumns = `id,profile_id,attempt_id,exam_id,version,rank,criteria_hash,key_id,envelope_json,published,created_at,revoked_at`

// IssueCredential stores a caller-signed envelope only when it exactly matches a passed rank award.
// Credentials are always private on issue, regardless of the caller's Published field.
func (s *Store) IssueCredential(profileID string, c Credential) (Credential, error) {
	if profileID != "local" {
		return Credential{}, ErrCredentialPrerequisites
	}
	if !validLowerHex(c.ID, 32) || !validLowerHex(c.KeyID, 64) {
		return Credential{}, errors.New("credential and key IDs must be lowercase hex")
	}
	if c.ProfileID != profileID || c.AttemptID == "" || c.ExamID == "" || c.Version == "" || c.Rank == "" || c.CriteriaHash == "" {
		return Credential{}, ErrCredentialPrerequisites
	}
	if len(c.EnvelopeJSON) == 0 || len(c.EnvelopeJSON) > MaxCredentialEnvelopeBytes {
		return Credential{}, errors.New("credential envelope size is invalid")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(c.EnvelopeJSON), &envelope); err != nil || envelope == nil {
		return Credential{}, errors.New("credential envelope must be a JSON object")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Credential{}, err
	}
	defer tx.Rollback()
	var profile string
	if err = tx.QueryRow(`SELECT id FROM profiles WHERE id=?`, profileID).Scan(&profile); errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrProfileNotFound
	} else if err != nil {
		return Credential{}, err
	}
	var examID, version, rank, hash string
	err = tx.QueryRow(`SELECT a.exam_id,a.version,a.rank,a.criteria_hash FROM rank_awards r JOIN exam_attempts a ON a.id=r.attempt_id JOIN exams e ON e.id=a.exam_id AND e.version=a.version WHERE r.profile_id=? AND r.attempt_id=? AND r.exam_id=a.exam_id AND r.version=a.version AND r.rank=a.rank AND a.status='completed' AND a.passed=1 AND a.score=100 AND e.rank=a.rank AND e.criteria_hash=a.criteria_hash`, profileID, c.AttemptID).Scan(&examID, &version, &rank, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrCredentialPrerequisites
	}
	if err != nil {
		return Credential{}, err
	}
	if c.ExamID != examID || c.Version != version || c.Rank != rank || c.CriteriaHash != hash {
		return Credential{}, ErrCredentialPrerequisites
	}
	now := time.Now().UTC()
	stamp := now.Format(time.RFC3339Nano)
	_, err = tx.Exec(`INSERT INTO credentials(id,profile_id,attempt_id,exam_id,version,rank,criteria_hash,key_id,envelope_json,published,created_at) VALUES(?,?,?,?,?,?,?,?,?,0,?)`, c.ID, profileID, c.AttemptID, c.ExamID, c.Version, c.Rank, c.CriteriaHash, c.KeyID, c.EnvelopeJSON, stamp)
	if err != nil {
		if isConstraint(err) {
			return Credential{}, ErrCredentialConflict
		}
		return Credential{}, err
	}
	if _, err = tx.Exec(`INSERT INTO credential_audit(profile_id,credential_id,action,created_at) VALUES(?,?,'issued',?)`, profileID, c.ID, stamp); err != nil {
		return Credential{}, err
	}
	if err = tx.Commit(); err != nil {
		return Credential{}, err
	}
	return s.Credential(profileID, c.ID)
}

func (s *Store) Credential(profileID, id string) (Credential, error) {
	if profileID != "local" {
		return Credential{}, ErrCredentialNotFound
	}
	c, err := credentialFromRow(s.db.QueryRow(`SELECT `+credentialColumns+` FROM credentials WHERE profile_id=? AND id=?`, profileID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrCredentialNotFound
	}
	return c, err
}

func (s *Store) Credentials(profileID string) ([]Credential, error) {
	if profileID != "local" {
		return nil, ErrProfileNotFound
	}
	rows, err := s.db.Query(`SELECT `+credentialColumns+` FROM credentials WHERE profile_id=? ORDER BY created_at,id`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Credential{}
	for rows.Next() {
		c, e := credentialFromRow(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) PublicCredential(id string) (Credential, error) {
	c, err := credentialFromRow(s.db.QueryRow(`SELECT `+credentialColumns+` FROM credentials WHERE id=? AND published=1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrCredentialNotFound
	}
	return c, err
}

func (s *Store) SetCredentialPublished(profileID, id string, published bool) error {
	if profileID != "local" {
		return ErrCredentialNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current int
	var revoked sql.NullString
	err = tx.QueryRow(`SELECT published,revoked_at FROM credentials WHERE profile_id=? AND id=?`, profileID, id).Scan(&current, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCredentialNotFound
	}
	if err != nil {
		return err
	}
	if published && revoked.Valid {
		return ErrCredentialConflict
	}
	if current != boolInt(published) {
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err = tx.Exec(`UPDATE credentials SET published=? WHERE profile_id=? AND id=?`, boolInt(published), profileID, id); err != nil {
			return err
		}
		action := "unpublished"
		if published {
			action = "published"
		}
		if _, err = tx.Exec(`INSERT INTO credential_audit(profile_id,credential_id,action,created_at) VALUES(?,?,?,?)`, profileID, id, action, stamp); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RevokeCredential is idempotent. The free-form reason is validated but never persisted or audited.
func (s *Store) RevokeCredential(profileID, id, reason string) error {
	if profileID != "local" {
		return ErrCredentialNotFound
	}
	if len(reason) > 256 {
		return errors.New("revocation reason is too long")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var revoked sql.NullString
	err = tx.QueryRow(`SELECT revoked_at FROM credentials WHERE profile_id=? AND id=?`, profileID, id).Scan(&revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCredentialNotFound
	}
	if err != nil {
		return err
	}
	if revoked.Valid {
		return tx.Commit()
	}
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.Exec(`UPDATE credentials SET revoked_at=? WHERE profile_id=? AND id=?`, stamp, profileID, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO credential_audit(profile_id,credential_id,action,created_at) VALUES(?,?,'revoked',?)`, profileID, id, stamp); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) PassportPublic(profileID string) (bool, error) {
	if profileID != "local" {
		return false, ErrProfileNotFound
	}
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE id=?`, profileID).Scan(&exists); err != nil {
		return false, err
	}
	if exists == 0 {
		return false, ErrProfileNotFound
	}
	var public int
	err := s.db.QueryRow(`SELECT is_public FROM passport_privacy WHERE profile_id=?`, profileID).Scan(&public)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return public == 1, err
}

func (s *Store) SetPassportPublic(profileID string, published bool) error {
	if profileID != "local" {
		return ErrProfileNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM profiles WHERE id=?`, profileID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrProfileNotFound
	}
	var current int
	err = tx.QueryRow(`SELECT is_public FROM passport_privacy WHERE profile_id=?`, profileID).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) && !published {
		return tx.Commit()
	}
	if err == nil && current == boolInt(published) {
		return tx.Commit()
	}
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.Exec(`INSERT INTO passport_privacy(profile_id,is_public,updated_at) VALUES(?,?,?) ON CONFLICT(profile_id) DO UPDATE SET is_public=excluded.is_public,updated_at=excluded.updated_at`, profileID, boolInt(published), stamp); err != nil {
		return err
	}
	action := "passport_private"
	if published {
		action = "passport_public"
	}
	if _, err = tx.Exec(`INSERT INTO credential_audit(profile_id,action,created_at) VALUES(?,?,?)`, profileID, action, stamp); err != nil {
		return fmt.Errorf("record passport privacy event: %w", err)
	}
	return tx.Commit()
}
