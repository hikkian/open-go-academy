-- Each attempt snapshots the exact immutable project evaluations used by its
-- registered rank policy. Evidence must outlive attempt pruning and workspace edits.
CREATE UNIQUE INDEX exam_attempt_rank_evidence_identity
    ON exam_attempts(id, profile_id, exam_id, version);
CREATE UNIQUE INDEX project_evaluation_rank_evidence_identity
    ON project_evaluations(id, profile_id, project_id, submission_id);

CREATE TABLE exam_project_evidence (
    attempt_id TEXT NOT NULL,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE RESTRICT,
    exam_id TEXT NOT NULL,
    exam_version TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    requirement_id TEXT NOT NULL,
    submission_id TEXT NOT NULL,
    evaluation_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    project_version TEXT NOT NULL,
    criteria_hash TEXT NOT NULL,
    artifact_schema_version INTEGER NOT NULL,
    artifact_hash TEXT NOT NULL,
    executed_source_hash TEXT NOT NULL,
    checker_version TEXT NOT NULL,
    checker_digest TEXT NOT NULL,
    runtime_policy_hash TEXT NOT NULL,
    rubric_version TEXT NOT NULL,
    rubric_hash TEXT NOT NULL,
    evaluation_status TEXT NOT NULL CHECK(evaluation_status IN ('automatic_pass','failed','error')),
    exit_code INTEGER NOT NULL,
    passed INTEGER NOT NULL CHECK(passed IN (0,1)),
    checks_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(attempt_id, requirement_id),
    FOREIGN KEY(attempt_id, profile_id, exam_id, exam_version)
        REFERENCES exam_attempts(id, profile_id, exam_id, version) ON DELETE RESTRICT,
    FOREIGN KEY(exam_id, exam_version, policy_version)
        REFERENCES rank_policies(exam_id, exam_version, policy_version) ON DELETE RESTRICT,
    FOREIGN KEY(submission_id, profile_id, project_id)
        REFERENCES project_submissions(id, profile_id, project_id) ON DELETE RESTRICT,
    FOREIGN KEY(evaluation_id, profile_id, project_id, submission_id)
        REFERENCES project_evaluations(id, profile_id, project_id, submission_id) ON DELETE RESTRICT
);

CREATE INDEX exam_project_evidence_submission ON exam_project_evidence(submission_id, evaluation_id);
CREATE TRIGGER exam_project_evidence_immutable_update
BEFORE UPDATE ON exam_project_evidence BEGIN
    SELECT RAISE(ABORT, 'exam project evidence is immutable');
END;
CREATE TRIGGER exam_project_evidence_immutable_delete
BEFORE DELETE ON exam_project_evidence BEGIN
    SELECT RAISE(ABORT, 'exam project evidence is immutable');
END;
