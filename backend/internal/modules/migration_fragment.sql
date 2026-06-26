-- ── Cmss ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS cmss (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed RBAC permissions (replace keys with your actual domain)
INSERT INTO permissions (name, key, module, description)
VALUES ('Manage Cmss', 'manage:cmss', 'cms', 'Create, update, delete cmss')
ON CONFLICT (key) DO NOTHING;

INSERT INTO permissions (name, key, module, description)
VALUES ('Read Cmss', 'read:cmss', 'cms', 'View cmss')
ON CONFLICT (key) DO NOTHING;
