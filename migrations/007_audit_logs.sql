CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,

    user_id     BIGINT NULL,

    action      VARCHAR(100) NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),

    method      VARCHAR(10),
    path        TEXT,

    ip_address  INET,
    user_agent  TEXT,

    request_id  VARCHAR(100),

    old_data    JSONB,
    new_data    JSONB,

    metadata    JSONB,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id);

ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;
