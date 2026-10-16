CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE users (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), email text NOT NULL UNIQUE CHECK (length(email) BETWEEN 3 AND 254), password_hash text NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_expiry ON sessions(user_id, expires_at);
CREATE TABLE workspaces (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL CHECK (length(name) BETWEEN 1 AND 100), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE memberships (
 workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE, user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, role text NOT NULL CHECK(role IN ('owner','editor','viewer')), PRIMARY KEY(workspace_id,user_id)
);
CREATE TABLE projects (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE, name text NOT NULL CHECK(length(name) BETWEEN 1 AND 120), created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(workspace_id,id)
);
CREATE INDEX projects_workspace_created ON projects(workspace_id,created_at,id);
CREATE TABLE tasks (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE, title text NOT NULL CHECK(length(title) BETWEEN 1 AND 200), description text NOT NULL DEFAULT '' CHECK(length(description)<=4000), state text NOT NULL DEFAULT 'open' CHECK(state IN ('open','doing','done')), version integer NOT NULL DEFAULT 1 CHECK(version>0), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tasks_project_updated ON tasks(project_id,updated_at DESC,id);
CREATE TABLE comments (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), task_id uuid NOT NULL REFERENCES tasks ON DELETE CASCADE, author_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, body text NOT NULL CHECK(length(body) BETWEEN 1 AND 2000), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comments_task_created ON comments(task_id,created_at,id);
CREATE TABLE outbox (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE, idempotency_key text NOT NULL CHECK(length(idempotency_key) BETWEEN 1 AND 128), kind text NOT NULL CHECK(length(kind) BETWEEN 1 AND 80), payload jsonb NOT NULL, state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','processing','done','dead')), attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0), available_at timestamptz NOT NULL DEFAULT now(), locked_at timestamptz, lease_token uuid, last_error text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), completed_at timestamptz, UNIQUE(project_id,idempotency_key)
);
CREATE INDEX outbox_ready ON outbox(available_at,created_at) WHERE state IN ('pending','processing');
CREATE TABLE job_receipts (
 outbox_id uuid PRIMARY KEY REFERENCES outbox ON DELETE CASCADE, kind text NOT NULL, payload jsonb NOT NULL, processed_at timestamptz NOT NULL DEFAULT now()
);
