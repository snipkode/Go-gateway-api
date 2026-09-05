CREATE TABLE role_permissions (
    role_id       BIGINT      NOT NULL,
    permission_id BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ NULL,

    CONSTRAINT pk_role_permissions PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_perm FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

CREATE INDEX idx_role_permissions_role ON role_permissions (role_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_role_permissions_perm ON role_permissions (permission_id) WHERE deleted_at IS NULL;
