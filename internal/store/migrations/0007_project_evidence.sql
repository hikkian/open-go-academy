CREATE TABLE project_submissions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    project_id TEXT NOT NULL,
    project_version TEXT NOT NULL,
    criteria_hash TEXT NOT NULL,
    artifact_schema_version INTEGER NOT NULL CHECK(artifact_schema_version > 0),
    artifact_hash TEXT NOT NULL,
    executed_source_hash TEXT NOT NULL,
    files_json TEXT NOT NULL,
    executed_source_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(id, profile_id, project_id),
    UNIQUE(profile_id, project_id, project_version, criteria_hash, artifact_hash, executed_source_hash)
);

CREATE TABLE project_evaluations (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    project_id TEXT NOT NULL,
    submission_id TEXT NOT NULL,
    checker_version TEXT NOT NULL,
    checker_digest TEXT NOT NULL,
    runtime_policy_hash TEXT NOT NULL,
    rubric_version TEXT NOT NULL,
    rubric_hash TEXT NOT NULL,
    evaluation_status TEXT NOT NULL CHECK(evaluation_status IN ('automatic_pass','failed','error')),
    exit_code INTEGER NOT NULL,
    passed INTEGER NOT NULL CHECK(passed IN (0,1)),
    checks_json TEXT NOT NULL,
    checks_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(submission_id, profile_id, project_id)
        REFERENCES project_submissions(id, profile_id, project_id) ON DELETE RESTRICT,
    UNIQUE(submission_id, checker_digest, runtime_policy_hash, rubric_hash)
);

CREATE INDEX project_submissions_profile_project_time
    ON project_submissions(profile_id, project_id, created_at, id);
CREATE INDEX project_evaluations_profile_project_time
    ON project_evaluations(profile_id, project_id, created_at, id);

CREATE TRIGGER project_submissions_immutable_update
BEFORE UPDATE ON project_submissions BEGIN
    SELECT RAISE(ABORT, 'project submissions are immutable');
END;
CREATE TRIGGER project_submissions_immutable_delete
BEFORE DELETE ON project_submissions BEGIN
    SELECT RAISE(ABORT, 'project submissions are immutable');
END;
CREATE TRIGGER project_evaluations_immutable_update
BEFORE UPDATE ON project_evaluations BEGIN
    SELECT RAISE(ABORT, 'project evaluations are immutable');
END;
CREATE TRIGGER project_evaluations_immutable_delete
BEFORE DELETE ON project_evaluations BEGIN
    SELECT RAISE(ABORT, 'project evaluations are immutable');
END;
