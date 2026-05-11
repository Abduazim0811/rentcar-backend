-- +goose Up

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(80) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
    role VARCHAR(20) NOT NULL,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role, permission_id),
    CONSTRAINT role_permissions_role_check CHECK (role IN ('customer', 'admin', 'super_admin'))
);

INSERT INTO permissions (code, description) VALUES
    ('admin:access', 'Access admin area'),
    ('cars:manage', 'Create, update and delete cars'),
    ('rentals:manage', 'Review and update rentals'),
    ('payments:manage', 'Confirm, fail and refund payments'),
    ('maintenance:manage', 'Manage car maintenance records'),
    ('reports:view', 'View reports'),
    ('audit:view', 'View audit logs'),
    ('users:manage', 'Manage users and roles')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT role, id
FROM permissions
CROSS JOIN (VALUES ('admin'), ('super_admin')) AS roles(role)
WHERE code IN ('admin:access', 'cars:manage', 'rentals:manage', 'payments:manage', 'maintenance:manage', 'reports:view')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id
FROM permissions
WHERE code IN ('audit:view', 'users:manage')
ON CONFLICT DO NOTHING;
