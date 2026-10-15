CREATE TABLE credentials (
    id TEXT PRIMARY KEY CHECK(length(id)=32 AND id NOT GLOB '*[^0-9a-f]*'),
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    attempt_id TEXT NOT NULL UNIQUE REFERENCES exam_attempts(id),
    exam_id TEXT NOT NULL,
    version TEXT NOT NULL,
    rank TEXT NOT NULL,
    criteria_hash TEXT NOT NULL,
    key_id TEXT NOT NULL CHECK(length(key_id)=64 AND key_id NOT GLOB '*[^0-9a-f]*'),
    envelope_json TEXT NOT NULL,
    published INTEGER NOT NULL DEFAULT 0 CHECK(published IN (0,1)),
    created_at TEXT NOT NULL,
    revoked_at TEXT,
    FOREIGN KEY(exam_id,version) REFERENCES exams(id,version)
);
CREATE INDEX credentials_profile_created ON credentials(profile_id,created_at,id);

CREATE TABLE credential_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id TEXT NOT NULL REFERENCES profiles(id),
    credential_id TEXT REFERENCES credentials(id),
    action TEXT NOT NULL CHECK(action IN ('issued','published','unpublished','revoked','passport_public','passport_private')),
    created_at TEXT NOT NULL
);
CREATE INDEX credential_audit_profile_time ON credential_audit(profile_id,created_at,id);

CREATE TABLE passport_privacy (
    profile_id TEXT PRIMARY KEY REFERENCES profiles(id),
    is_public INTEGER NOT NULL DEFAULT 0 CHECK(is_public IN (0,1)),
    updated_at TEXT NOT NULL
);
