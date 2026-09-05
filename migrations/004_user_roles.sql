CREATE TABLE user_roles (
    user_id    BIGINT      NOT NULL,
    role_id    BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT pk_user_roles PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_roles_role ON user_roles (role_id) WHERE deleted_at IS NULL;
