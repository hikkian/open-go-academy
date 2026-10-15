ALTER TABLE exam_attempts ADD COLUMN revision INTEGER NOT NULL DEFAULT 0 CHECK(revision >= 0);
