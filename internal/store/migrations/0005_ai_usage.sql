CREATE TABLE ai_usage_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    day TEXT NOT NULL,
    reserved_tokens INTEGER NOT NULL CHECK(reserved_tokens > 0 AND reserved_tokens <= 1000000),
    charged_tokens INTEGER NOT NULL DEFAULT 0 CHECK(charged_tokens >= 0),
    input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens >= 0),
    output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens >= 0),
    estimated INTEGER NOT NULL DEFAULT 0 CHECK(estimated IN (0,1)),
    success INTEGER NOT NULL DEFAULT 0 CHECK(success IN (0,1)),
    status TEXT NOT NULL DEFAULT 'reserved' CHECK(status IN ('reserved','finished')),
    created_at TEXT NOT NULL,
    finished_at TEXT
);
CREATE INDEX ai_usage_requests_day ON ai_usage_requests(day,id);
