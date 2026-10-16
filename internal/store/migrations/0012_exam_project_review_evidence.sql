-- Exam attempts snapshot the exact verified human-review records used to meet
-- an immutable policy quorum. These links are append-only and retain evidence.
CREATE TABLE exam_project_review_evidence (
    attempt_id TEXT NOT NULL,
    profile_id TEXT NOT NULL CHECK(profile_id='local'),
    exam_id TEXT NOT NULL,
    exam_version TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    requirement_id TEXT NOT NULL,
    review_id TEXT NOT NULL REFERENCES project_reviews(id) ON DELETE RESTRICT,
    submission_id TEXT NOT NULL,
    evaluation_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    project_version TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,
    rubric_version TEXT NOT NULL,
    rubric_hash TEXT NOT NULL,
    decision TEXT NOT NULL CHECK(decision='approve'),
    attestation_digest TEXT NOT NULL,
    reviewer_key_fingerprint TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(attempt_id, requirement_id, review_id),
    FOREIGN KEY(attempt_id, profile_id, exam_id, exam_version)
        REFERENCES exam_attempts(id, profile_id, exam_id, version) ON DELETE RESTRICT,
    FOREIGN KEY(exam_id, exam_version, policy_version)
        REFERENCES rank_policies(exam_id, exam_version, policy_version) ON DELETE RESTRICT,
    FOREIGN KEY(attempt_id, requirement_id)
        REFERENCES exam_project_evidence(attempt_id, requirement_id) ON DELETE RESTRICT,
    FOREIGN KEY(submission_id, profile_id, project_id)
        REFERENCES project_submissions(id, profile_id, project_id) ON DELETE RESTRICT
);
CREATE INDEX exam_project_review_evidence_review ON exam_project_review_evidence(review_id);
CREATE TRIGGER exam_project_review_evidence_immutable_update
BEFORE UPDATE ON exam_project_review_evidence BEGIN
    SELECT RAISE(ABORT, 'exam project review evidence is immutable');
END;
CREATE TRIGGER exam_project_review_evidence_immutable_delete
BEFORE DELETE ON exam_project_review_evidence BEGIN
    SELECT RAISE(ABORT, 'exam project review evidence is immutable');
END;
