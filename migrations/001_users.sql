CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL DEFAULT '',
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    provider      VARCHAR(50)  NOT NULL DEFAULT 'local',
    provider_id   VARCHAR(255) NOT NULL DEFAULT '',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ  NULL
);

-- Partial unique index so soft-deleted rows can reuse the email
CREATE UNIQUE INDEX users_email_unique
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_provider ON users (provider, provider_id) WHERE deleted_at IS NULL;
