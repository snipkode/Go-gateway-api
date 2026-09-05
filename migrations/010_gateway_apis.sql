-- API Gateway registry: self-service registration of APIs the Nginx gateway
-- exposes dynamically. Each enabled/active row becomes a generated nginx
-- location (shared volume) that the gateway picks up after its hot-reload.
CREATE TABLE gateway_apis (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT        NOT NULL,
    base_path        TEXT        NOT NULL,
    upstream         TEXT        NOT NULL,
    methods          TEXT[]      NOT NULL DEFAULT '{GET}',
    requires_auth    BOOLEAN     NOT NULL DEFAULT TRUE,
    rate_limit_rpm   INT         NOT NULL DEFAULT 60 CHECK (rate_limit_rpm BETWEEN 1 AND 100000),
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    status           TEXT        NOT NULL DEFAULT 'unknown',
    last_checked_at  TIMESTAMPTZ NULL,
    note             TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX uq_gateway_apis_base_path ON gateway_apis (base_path) WHERE deleted_at IS NULL;

-- Management console permissions. Grant both to admin; viewer keeps read-only.
INSERT INTO permissions (name, slug, description) VALUES
    ('Gateway API Read',   'apigateway:read',   'Read registered gateway APIs and their stats'),
    ('Gateway API Manage', 'apigateway:manage', 'Register/update/delete gateway APIs and publish config');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'admin' AND p.slug IN ('apigateway:read', 'apigateway:manage');