-- 000012: Enable public registration
-- Adds setting toggle + customer role + CMS/contact permissions

INSERT INTO settings (key, value) VALUES ('public_registration', 'false')
ON CONFLICT (key) DO NOTHING;

INSERT INTO roles (name, description) VALUES ('customer', 'End-user customer — self-registered through marketing site')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permission_registry (domain, name, description) VALUES
    ('cms', 'read:cms', 'View CMS content'),
    ('cms', 'manage:cms', 'Create, edit, publish, delete CMS content'),
    ('contact', 'read:contacts', 'View contact messages and subscribers'),
    ('contact', 'manage:contacts', 'Manage contact messages and subscribers')
ON CONFLICT (name) DO NOTHING;
