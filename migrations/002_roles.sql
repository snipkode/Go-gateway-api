CREATE TABLE roles (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  NULL
);

CREATE UNIQUE INDEX roles_slug_unique
    ON roles (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_roles_deleted_at ON roles (deleted_at);
