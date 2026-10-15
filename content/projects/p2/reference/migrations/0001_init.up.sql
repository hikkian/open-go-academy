CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email_normalized text NOT NULL UNIQUE CHECK (length(email_normalized) BETWEEN 3 AND 254),
    password_hash bytea NOT NULL,
    role text NOT NULL DEFAULT 'user' CHECK (role IN ('user','admin')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
    token_hash char(64) PRIMARY KEY,
    csrf_hash char(64) NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);
CREATE TABLE links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{1,62}[a-z0-9])$'),
    target_url text NOT NULL CHECK (length(target_url) BETWEEN 1 AND 2048),
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX links_owner_created_idx ON links(owner_id,created_at DESC,id DESC);
CREATE TABLE tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 40),
    name_normalized text NOT NULL,
    UNIQUE(owner_id,name_normalized)
);
CREATE TABLE link_tags (
    link_id uuid NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY(link_id,tag_id)
);
CREATE INDEX link_tags_tag_idx ON link_tags(tag_id,link_id);
