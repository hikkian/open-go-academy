-- Offline review attestations are append-only and bound to exact immutable
-- local project submissions and passing evaluator results.
CREATE UNIQUE INDEX project_evaluation_review_identity
    ON project_evaluations(id, profile_id, project_id, submission_id);

CREATE TABLE project_reviews (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE RESTRICT CHECK(profile_id='local'),
    submission_id TEXT NOT NULL,
    evaluation_id TEXT NOT NULL,
    project_id TEXT NOT NULL CHECK(project_id IN ('p3','p4','p5')),
    project_version TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    attestation_version INTEGER NOT NULL CHECK(attestation_version > 0),
    rubric_version TEXT NOT NULL,
    rubric_hash TEXT NOT NULL,
    decision TEXT NOT NULL CHECK(decision IN ('approve','changes_requested','reject')),
    comments TEXT NOT NULL,
    issued_at TEXT NOT NULL,
    signature TEXT NOT NULL,
    attestation_digest TEXT NOT NULL UNIQUE,
    attestation_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(submission_id, profile_id, project_id)
        REFERENCES project_submissions(id, profile_id, project_id) ON DELETE RESTRICT,
    FOREIGN KEY(evaluation_id, profile_id, project_id, submission_id)
        REFERENCES project_evaluations(id, profile_id, project_id, submission_id) ON DELETE RESTRICT
);

CREATE INDEX project_reviews_submission_time ON project_reviews(submission_id, created_at, id);
CREATE TRIGGER project_reviews_immutable_update
BEFORE UPDATE ON project_reviews BEGIN
    SELECT RAISE(ABORT, 'project reviews are immutable');
END;
CREATE TRIGGER project_reviews_immutable_delete
BEFORE DELETE ON project_reviews BEGIN
    SELECT RAISE(ABORT, 'project reviews are immutable');
END;
