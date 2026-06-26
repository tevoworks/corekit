-- Add missing permissions used by the codebase but not seeded.

INSERT INTO permissions (name, description) VALUES
    ('read:feature_flags', 'View feature flags')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permission_registry (domain, name, description) VALUES
    ('settings', 'read:feature_flags', 'View feature flags')
ON CONFLICT (name) DO NOTHING;
