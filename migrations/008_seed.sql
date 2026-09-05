-- Seed defaults: roles and permissions
INSERT INTO permissions (name, slug, description) VALUES
    ('User Read',   'user:read',   'Read user records'),
    ('User Create', 'user:create', 'Create users'),
    ('User Update', 'user:update', 'Update users'),
    ('User Delete', 'user:delete', 'Soft delete users'),
    ('User Restore','user:restore','Restore soft-deleted users'),
    ('Role Read',   'role:read',   'Read roles'),
    ('Role Write',  'role:write',  'Create/update/delete roles'),
    ('Permission Assign', 'permission:assign', 'Assign permissions to roles'),
    ('Rate Limit Read', 'ratelimit:read', 'Read rate limit rules'),
    ('Rate Limit Write', 'ratelimit:write', 'Create/update/delete rate limit rules'),
    ('Audit Read',  'audit:read',  'Read audit logs');

INSERT INTO roles (name, slug, description) VALUES
    ('Admin',  'admin',  'Full administrative access'),
    ('Viewer', 'viewer', 'Read-only access');

-- github.com-based default: admin gets all permissions, viewer gets read-only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'admin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.slug IN ('user:read', 'role:read', 'ratelimit:read', 'audit:read')
WHERE r.slug = 'viewer';

-- Default rate limit rules (dynamic, overridable at runtime)
INSERT INTO rate_limit_rules (name, scope, identifier, requests, window_seconds, priority) VALUES
    ('API Default',     'global', '',                         1000, 60, 0),
    ('Login Route',     'route',  'POST:/api/v1/auth/login',    10, 60, 10),
    ('Admin Role',      'role',   'admin',                     5000, 60, 5),
    ('Viewer Role',     'role',   'viewer',                     500, 60, 5);
