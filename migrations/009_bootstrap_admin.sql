-- Bootstrap admin. Password: admin123 (bcrypt below).
-- In production replace the hash with a secret-managed user or an Entra/OIDC
-- group mapping; this is only for local development and testing.
INSERT INTO users (email, name, password_hash, provider)
VALUES ('admin@example.com', 'Bootstrap Admin',
        '$2a$10$t9ULcvj5UFPJLUlrXIBdEuhNlte4IwHGMpm9LeFgqrNIYK0SQ1ozy', 'local');

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.slug = 'admin'
WHERE u.email = 'admin@example.com';