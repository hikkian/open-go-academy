package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// VerifiedSeniorCommunicationEvidence returns a stable opaque identifier only
// when a persisted Senior award is backed by its exact, validated P5 review
// snapshot. The identifier contains no reviewer identity or key material.
func (s *Store) VerifiedSeniorCommunicationEvidence(profileID string) (string, error) {
	rows, err := s.db.Query(`SELECT profile_id,exam_id,version,rank,attempt_id FROM rank_awards WHERE profile_id=? AND rank='Senior' ORDER BY awarded_at,exam_id,version`, profileID)
	if err != nil {
		return "", err
	}
	type award struct{ profile, exam, version, rank, attempt string }
	var awards []award
	for rows.Next() {
		var a award
		if err = rows.Scan(&a.profile, &a.exam, &a.version, &a.rank, &a.attempt); err != nil {
			rows.Close()
			return "", err
		}
		awards = append(awards, a)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	if err = rows.Close(); err != nil {
		return "", err
	}
	for _, a := range awards {
		var attempt ExamAttempt
		var passed int
		err = s.db.QueryRow(`SELECT id,profile_id,exam_id,version,rank,criteria_hash,status,score,passed,evidence_json
			FROM exam_attempts WHERE id=?`, a.attempt).Scan(&attempt.ID, &attempt.ProfileID, &attempt.ExamID, &attempt.Version,
			&attempt.Rank, &attempt.CriteriaHash, &attempt.Status, &attempt.Score, &passed, &attempt.EvidenceJSON)
		if err != nil {
			return "", fmt.Errorf("Senior award references missing attempt: %w", err)
		}
		attempt.Passed = passed == 1
		if a.profile != attempt.ProfileID || a.exam != attempt.ExamID || a.version != attempt.Version || a.rank != attempt.Rank ||
			attempt.ProfileID != profileID || attempt.Rank != "Senior" || attempt.Status != "completed" || !attempt.Passed || attempt.Score != 100 {
			return "", errors.New("Senior award and attempt binding is corrupt")
		}
		var examRank, criteriaHash string
		if err = s.db.QueryRow(`SELECT rank,criteria_hash FROM exams WHERE id=? AND version=?`, attempt.ExamID, attempt.Version).Scan(&examRank, &criteriaHash); err != nil {
			return "", err
		}
		if examRank != attempt.Rank || criteriaHash != attempt.CriteriaHash {
			return "", errors.New("Senior attempt exam binding is corrupt")
		}
		var evidence examEvidenceSnapshot
		if err = json.Unmarshal([]byte(attempt.EvidenceJSON), &evidence); err != nil {
			return "", fmt.Errorf("invalid Senior exam evidence: %w", err)
		}
		if err = validateAwardedExamEvidenceTxMustDB(s.db, profileID, attempt.ID, attempt.ExamID, attempt.Version, attempt.Rank, attempt.CriteriaHash, attempt.EvidenceJSON); err != nil {
			return "", err
		}
		if evidence.RankPolicyVersion < 1 {
			continue
		}
		policy, policyErr := s.RankPolicy(attempt.ExamID, attempt.Version, evidence.RankPolicyVersion)
		if policyErr != nil {
			return "", policyErr
		}
		if policy.Rank != "Senior" || policy.PolicyHash != evidence.RankPolicyHash {
			return "", errors.New("Senior policy snapshot is corrupt")
		}
		for _, req := range policy.ProjectEvidenceRequirements {
			if req.ProjectID != "p5" || req.MinimumDistinctReviewerKeyFingerprints < 2 {
				continue
			}
			proof, ok := evidence.ProjectEvidence[req.RequirementID]
			if !ok || proof.ProjectID != "p5" || len(proof.ManualReviews) != req.MinimumDistinctReviewerKeyFingerprints {
				continue
			}
			// Derive the ID only from the award/attempt/policy and project proof
			// bindings; reviewer keys and identities are deliberately excluded.
			material := "open-go-academy/senior-communication-evidence/v1\x00" + profileID + "\x00" + a.exam + "\x00" + a.version + "\x00" + a.attempt + "\x00" + policy.PolicyHash + "\x00" + proof.SubmissionID + "\x00" + proof.EvaluationID
			sum := sha256.Sum256([]byte(material))
			return "sha256:" + hex.EncodeToString(sum[:]), nil
		}
	}
	return "", nil
}

func validateAwardedExamEvidenceTxMustDB(db *sql.DB, profileID, attemptID, examID, examVersion, rank, criteriaHash, evidenceJSON string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateAwardedExamEvidenceTx(tx, profileID, attemptID, examID, examVersion, rank, criteriaHash, evidenceJSON); err != nil {
		return err
	}
	return tx.Commit()
}
