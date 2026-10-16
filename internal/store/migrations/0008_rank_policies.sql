-- Rank policies are server-authored immutable definitions. They are only
-- stored here; exam flow and rank awards do not consume them yet.
CREATE TABLE rank_policies (
    exam_id TEXT NOT NULL,
    exam_version TEXT NOT NULL,
    policy_version INTEGER NOT NULL CHECK(policy_version > 0),
    rank TEXT NOT NULL CHECK(rank IN ('Middle','Middle+','Senior')),
    criteria_hash TEXT NOT NULL,
    policy_hash TEXT NOT NULL,
    definition_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(exam_id, exam_version, policy_version),
    FOREIGN KEY(exam_id, exam_version) REFERENCES exams(id, version) ON DELETE RESTRICT
);

CREATE TRIGGER rank_policies_immutable_update
BEFORE UPDATE ON rank_policies BEGIN
    SELECT RAISE(ABORT, 'rank policies are immutable');
END;
CREATE TRIGGER rank_policies_immutable_delete
BEFORE DELETE ON rank_policies BEGIN
    SELECT RAISE(ABORT, 'rank policies are immutable');
END;
