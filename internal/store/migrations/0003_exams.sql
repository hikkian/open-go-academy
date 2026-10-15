CREATE TABLE exams (
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    rank TEXT NOT NULL,
    criteria_hash TEXT NOT NULL,
    PRIMARY KEY (id, version)
);

CREATE TABLE exam_attempts (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    exam_id TEXT NOT NULL,
    version TEXT NOT NULL,
    rank TEXT NOT NULL,
    criteria_hash TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('active','grading','completed','interrupted','cancelled')),
    answers_json TEXT NOT NULL DEFAULT '{}',
    codes_json TEXT NOT NULL DEFAULT '{}',
    score INTEGER NOT NULL DEFAULT 0 CHECK(score BETWEEN 0 AND 100),
    passed INTEGER NOT NULL DEFAULT 0 CHECK(passed IN (0,1)),
    created_at TEXT NOT NULL,
    finished_at TEXT,
    checks_json TEXT NOT NULL DEFAULT '[]',
    evidence_json TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY(exam_id, version) REFERENCES exams(id, version)
);
CREATE UNIQUE INDEX one_open_exam_attempt_per_profile ON exam_attempts(profile_id)
    WHERE status IN ('active','grading');
CREATE INDEX exam_attempts_profile_time ON exam_attempts(profile_id, created_at, id);

CREATE TABLE rank_awards (
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    exam_id TEXT NOT NULL,
    version TEXT NOT NULL,
    rank TEXT NOT NULL,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES exam_attempts(id),
    awarded_at TEXT NOT NULL,
    PRIMARY KEY(profile_id, exam_id, version),
    FOREIGN KEY(exam_id, version) REFERENCES exams(id, version)
);
