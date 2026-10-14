CREATE TABLE project_files (
    project_id TEXT NOT NULL,
    path TEXT NOT NULL,
    source TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (project_id, path)
);
CREATE TABLE project_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    files_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX project_snapshots_project_created ON project_snapshots(project_id, id DESC);
