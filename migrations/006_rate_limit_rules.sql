CREATE TABLE rate_limit_rules (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL,
    scope          VARCHAR(20)  NOT NULL,      -- global | ip | user | role | route | api_key
    identifier     VARCHAR(255) NOT NULL DEFAULT '',
    requests       BIGINT       NOT NULL,
    window_seconds BIGINT       NOT NULL,
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    priority       INT          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ  NULL
);

CREATE INDEX idx_rate_limit_scope ON rate_limit_rules (scope, identifier) WHERE deleted_at IS NULL AND enabled = TRUE;
CREATE INDEX idx_rate_limit_deleted_at ON rate_limit_rules (deleted_at);
