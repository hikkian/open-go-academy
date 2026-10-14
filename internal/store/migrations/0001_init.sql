CREATE TABLE profiles (id TEXT PRIMARY KEY CHECK(id='local'), name TEXT NOT NULL, theme TEXT NOT NULL, rank TEXT NOT NULL DEFAULT 'Zero', xp INTEGER NOT NULL DEFAULT 0 CHECK(xp>=0));
CREATE TABLE drafts (exercise_id TEXT PRIMARY KEY, files_json TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE attempts (id INTEGER PRIMARY KEY AUTOINCREMENT, exercise_id TEXT NOT NULL, files_json TEXT NOT NULL, passed INTEGER NOT NULL CHECK(passed IN (0,1)), output TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE verified_progress (exercise_id TEXT PRIMARY KEY, attempt_id INTEGER NOT NULL UNIQUE REFERENCES attempts(id), xp_awarded INTEGER NOT NULL CHECK(xp_awarded>=0));
