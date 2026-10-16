package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"academy/internal/projectreview"
)

var ErrProjectReviewNotFound = errors.New("project review not found")

// ProjectReview is the durable form of an operator-key-signed review decision.
// Its digest covers every attestation field, including the signature.
type ProjectReview struct {
	ID                     string                    `json:"id"`
	ProfileID              string                    `json:"profile_id"`
	Attestation            projectreview.Attestation `json:"attestation"`
	AttestationDigest      string                    `json:"attestation_digest"`
	ReviewerKeyFingerprint string                    `json:"reviewer_key_fingerprint"`
	CreatedAt              time.Time                 `json:"created_at"`
}

// RecordProjectReview stores an already verified offline attestation and the
// SHA-256 fingerprint of the exact operator-allowlisted public key used by the
// API to verify it. This records key possession only; it does not establish a
// real-world identity or reviewer independence.
func (s *Store) RecordProjectReview(profileID string, a projectreview.Attestation, reviewerKeyFingerprint string) (ProjectReview, error) {
	if profileID != "local" {
		return ProjectReview{}, ErrProfileNotFound
	}
	if !projectreview.ValidKeyFingerprint(reviewerKeyFingerprint) {
		return ProjectReview{}, errors.New("reviewer key fingerprint must be canonical sha256 hex")
	}
	if a.Version != projectreview.Version || a.ReviewerID == "" || a.SubmissionID == "" || a.EvaluationID == "" ||
		a.ProjectVersion == "" || a.RubricVersion == "" || a.ArtifactHash == "" || a.RubricHash == "" ||
		a.Signature == "" || (a.Decision != "approve" && a.Decision != "changes_requested" && a.Decision != "reject") {
		return ProjectReview{}, errors.New("invalid project review attestation")
	}
	if a.ProjectID != "p3" && a.ProjectID != "p4" && a.ProjectID != "p5" {
		return ProjectReview{}, errors.New("project review requires P3, P4, or P5")
	}
	issued, err := time.Parse(time.RFC3339, a.IssuedAt)
	if err != nil || issued.Format(time.RFC3339) != a.IssuedAt {
		return ProjectReview{}, errors.New("invalid project review issued_at")
	}
	canonical, err := json.Marshal(a)
	if err != nil {
		return ProjectReview{}, err
	}
	sum := sha256.Sum256(append([]byte("open-go-academy/project-review-attestation/v1\x00"), canonical...))
	digest := "sha256:" + hex.EncodeToString(sum[:])

	tx, err := s.db.Begin()
	if err != nil {
		return ProjectReview{}, err
	}
	defer tx.Rollback()
	var subVersion, subArtifact string
	if err := tx.QueryRow(`SELECT project_version,artifact_hash FROM project_submissions
		WHERE id=? AND profile_id=? AND project_id=?`, a.SubmissionID, profileID, a.ProjectID).Scan(&subVersion, &subArtifact); errors.Is(err, sql.ErrNoRows) {
		return ProjectReview{}, ErrProjectEvidenceNotFound
	} else if err != nil {
		return ProjectReview{}, err
	}
	var evVersion, evStatus string
	var passed int
	if err := tx.QueryRow(`SELECT s.project_version,e.evaluation_status,e.passed
		FROM project_evaluations e JOIN project_submissions s ON s.id=e.submission_id AND s.profile_id=e.profile_id AND s.project_id=e.project_id
		WHERE e.id=? AND e.profile_id=? AND e.project_id=? AND e.submission_id=?`,
		a.EvaluationID, profileID, a.ProjectID, a.SubmissionID).Scan(&evVersion, &evStatus, &passed); errors.Is(err, sql.ErrNoRows) {
		return ProjectReview{}, ErrProjectEvidenceNotFound
	} else if err != nil {
		return ProjectReview{}, err
	}
	if subVersion != a.ProjectVersion || evVersion != a.ProjectVersion || strings.TrimPrefix(subArtifact, "sha256:") != a.ArtifactHash {
		return ProjectReview{}, errors.New("project review does not match immutable submission and evaluation artifact")
	}
	if passed != 1 || evStatus != "automatic_pass" {
		return ProjectReview{}, errors.New("project review requires a passing immutable evaluation")
	}

	var existing ProjectReview
	var raw, created string
	err = tx.QueryRow(`SELECT id,profile_id,attestation_json,attestation_digest,reviewer_key_fingerprint,created_at
		FROM project_reviews WHERE attestation_digest=?`, digest).Scan(&existing.ID, &existing.ProfileID, &raw, &existing.AttestationDigest, &existing.ReviewerKeyFingerprint, &created)
	if err == nil {
		if err = json.Unmarshal([]byte(raw), &existing.Attestation); err != nil {
			return ProjectReview{}, err
		}
		if string(canonical) != raw {
			return ProjectReview{}, errors.New("project review digest collision")
		}
		if existing.ReviewerKeyFingerprint != reviewerKeyFingerprint {
			return ProjectReview{}, errors.New("project review replay uses a different or unrecorded reviewer key fingerprint")
		}
		existing.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return ProjectReview{}, err
		}
		if err = tx.Commit(); err != nil {
			return ProjectReview{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ProjectReview{}, err
	}
	id, err := newExamAttemptID()
	if err != nil {
		return ProjectReview{}, err
	}
	now := time.Now().UTC()
	_, err = tx.Exec(`INSERT INTO project_reviews(id,profile_id,submission_id,evaluation_id,project_id,project_version,
		artifact_hash,reviewer_id,attestation_version,rubric_version,rubric_hash,decision,comments,issued_at,signature,
		attestation_digest,attestation_json,reviewer_key_fingerprint,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, profileID, a.SubmissionID, a.EvaluationID, a.ProjectID, a.ProjectVersion, a.ArtifactHash, a.ReviewerID,
		a.Version, a.RubricVersion, a.RubricHash, a.Decision, a.Comments, a.IssuedAt, a.Signature, digest, string(canonical), reviewerKeyFingerprint, now.Format(time.RFC3339Nano))
	if err != nil {
		return ProjectReview{}, fmt.Errorf("persist project review: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return ProjectReview{}, err
	}
	return ProjectReview{ID: id, ProfileID: profileID, Attestation: a, AttestationDigest: digest, ReviewerKeyFingerprint: reviewerKeyFingerprint, CreatedAt: now}, nil
}

// ProjectReview reads one immutable attestation for the local profile.
func (s *Store) ProjectReview(profileID, id string) (ProjectReview, error) {
	if profileID != "local" {
		return ProjectReview{}, ErrProjectReviewNotFound
	}
	var review ProjectReview
	var raw, created string
	err := s.db.QueryRow(`SELECT id,profile_id,attestation_json,attestation_digest,reviewer_key_fingerprint,created_at FROM project_reviews WHERE id=? AND profile_id=?`, id, profileID).
		Scan(&review.ID, &review.ProfileID, &raw, &review.AttestationDigest, &review.ReviewerKeyFingerprint, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectReview{}, ErrProjectReviewNotFound
	}
	if err != nil {
		return ProjectReview{}, err
	}
	if err = json.Unmarshal([]byte(raw), &review.Attestation); err != nil {
		return ProjectReview{}, err
	}
	review.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return review, err
}

// ProjectReviews returns immutable review evidence for one local submission in
// append order. A non-existent submission is an empty result; callers first
// resolve the submission itself when they need to distinguish 404 from none.
func (s *Store) ProjectReviews(profileID, submissionID string) ([]ProjectReview, error) {
	if profileID != "local" {
		return nil, ErrProjectReviewNotFound
	}
	rows, err := s.db.Query(`SELECT id,profile_id,attestation_json,attestation_digest,reviewer_key_fingerprint,created_at
		FROM project_reviews WHERE profile_id=? AND submission_id=? ORDER BY created_at,id`, profileID, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reviews := make([]ProjectReview, 0)
	for rows.Next() {
		var review ProjectReview
		var raw, created string
		if err = rows.Scan(&review.ID, &review.ProfileID, &raw, &review.AttestationDigest, &review.ReviewerKeyFingerprint, &created); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &review.Attestation); err != nil {
			return nil, err
		}
		if review.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}
