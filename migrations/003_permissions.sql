CREATE TABLE permissions (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  NULL
);

CREATE UNIQUE INDEX permissions_slug_unique
    ON permissions (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_permissions_deleted_at ON permissions (deleted_at);
