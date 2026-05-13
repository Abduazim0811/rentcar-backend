-- +goose Up

INSERT INTO permissions (code, description)
VALUES ('users:view', 'View users and contact details')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT role, id
FROM permissions
CROSS JOIN (VALUES ('admin'), ('super_admin')) AS roles(role)
WHERE code = 'users:view'
ON CONFLICT DO NOTHING;

-- +goose Down

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'users:view');

DELETE FROM permissions WHERE code = 'users:view';
