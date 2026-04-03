CREATE TABLE invites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token       TEXT UNIQUE NOT NULL,
    label       TEXT NOT NULL,
    created_by  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,
    max_uses    INTEGER,
    use_count   INTEGER NOT NULL DEFAULT 0,
    library_ids TEXT[] NOT NULL DEFAULT '{}',
    revoked     BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE registrations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_id     UUID NOT NULL REFERENCES invites(id),
    jf_user_id    TEXT NOT NULL,
    username      TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    username   TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_expires ON sessions(expires_at);
