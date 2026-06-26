-- ─────────────────────────────────────────────────────────────────────────────
-- CoreKit — Initial Schema
-- Single-tenant system: NO tenants table.
-- All tables created with IF NOT EXISTS for idempotency.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 0. Extensions ──────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── 1. Roles ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS roles (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP
);

-- ── 2. Permissions ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS permissions (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 3. Users ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    id                   BIGSERIAL PRIMARY KEY,
    email                VARCHAR(255) NOT NULL,
    password_hash        VARCHAR(255) NOT NULL,
    full_name            VARCHAR(255),
    avatar_url           TEXT,
    status               VARCHAR(50) NOT NULL DEFAULT 'PENDING_VERIFICATION',
    role_id              BIGINT REFERENCES roles(id) ON DELETE SET NULL,
    is_super_admin       BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until         TIMESTAMPTZ,
    last_login_at        TIMESTAMP,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at           TIMESTAMP
);

-- ── 4. Role-Permission Mapping ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS role_permissions (
    id            BIGSERIAL PRIMARY KEY,
    role_id       BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (role_id, permission_id)
);

-- ── 5. Sessions ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sessions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id      VARCHAR(255) NOT NULL,
    ip_address    VARCHAR(45) NOT NULL,
    user_agent    TEXT NOT NULL,
    expires_at    TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at    TIMESTAMP,
    revoked_by    BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- ── 6. Audit Logs ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    actor_id        BIGINT REFERENCES users(id) ON DELETE SET NULL,
    impersonator_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action          VARCHAR(255) NOT NULL,
    target_entity   VARCHAR(255) NOT NULL,
    before_state    JSONB,
    after_state     JSONB,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 7. Settings ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS settings (
    id         BIGSERIAL PRIMARY KEY,
    key        VARCHAR(255) NOT NULL UNIQUE,
    value      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 8. Feature Flags ────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS feature_flags (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    key         VARCHAR(255) NOT NULL UNIQUE,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 9. File Metadata ────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS file_metadata (
    id              BIGSERIAL PRIMARY KEY,
    filename        VARCHAR(255) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    size_bytes      BIGINT NOT NULL,
    storage_path    VARCHAR(500) NOT NULL,
    uploaded_by     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,
    checksum_sha256 VARCHAR(64),
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- ── 10. API Keys ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS api_keys (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    key_hash        VARCHAR(64) NOT NULL,
    key_prefix      VARCHAR(10) NOT NULL,
    key_lookup_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL REFERENCES users(id),
    last_used_at    TIMESTAMP,
    revoked_at      TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- ── 11. Webhooks ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS webhooks (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    url        TEXT NOT NULL,
    secret     TEXT NOT NULL DEFAULT '',
    events     TEXT[] NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ── 12. Webhook Deliveries ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id            BIGSERIAL PRIMARY KEY,
    webhook_id    BIGINT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event         VARCHAR(255) NOT NULL,
    request_body  TEXT,
    response_code INT,
    response_body TEXT,
    duration_ms   INT,
    error_message TEXT,
    status        VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 13. Jobs ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS jobs (
    id              BIGSERIAL PRIMARY KEY,
    type            VARCHAR(255) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(255) UNIQUE,
    max_retries     INT NOT NULL DEFAULT 3,
    retry_count     INT NOT NULL DEFAULT 0,
    run_after       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_at       TIMESTAMP,
    locked_by       VARCHAR(255),
    last_heartbeat_at TIMESTAMP,
    next_retry_at   TIMESTAMP,
    final_status    BOOLEAN NOT NULL DEFAULT FALSE,
    error_message   TEXT,
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    failed_at       TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 14. Dead Letter Jobs ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS dead_letter_jobs (
    id              BIGSERIAL PRIMARY KEY,
    original_job_id BIGINT,
    type            VARCHAR(255) NOT NULL,
    payload         JSONB NOT NULL,
    error_message   TEXT,
    failed_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 15. Permission Registry ─────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS permission_registry (
    id          BIGSERIAL PRIMARY KEY,
    domain      VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 16. Global Templates ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS global_templates (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category    VARCHAR(100) NOT NULL DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    permissions JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 17. User Identities ─────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_identities (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider)
);

-- ── 18. User Verifications ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_verifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 19. Notifications ───────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS notifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(50) NOT NULL DEFAULT 'system',
    title      VARCHAR(255) NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    data       JSONB DEFAULT '{}',
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 20. User Notification Preferences ───────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id         BIGSERIAL PRIMARY KEY,
    user_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type  VARCHAR(50) NOT NULL,
    channel            VARCHAR(50) NOT NULL DEFAULT 'in_app',
    enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, notification_type, channel)
);

-- ── 21. User Preferences ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_preferences (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key        VARCHAR(100) NOT NULL,
    value      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, key)
);


-- ═════════════════════════════════════════════════════════════════════════════
-- INDEXES
-- ═════════════════════════════════════════════════════════════════════════════

-- Users
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active
    ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Roles
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);

-- Permissions
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions(name);

-- Role Permissions
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id);

-- Sessions
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_user_created ON sessions(user_id, created_at DESC);

-- Audit Logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(target_entity);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Feature Flags
CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);

-- File Metadata
CREATE INDEX IF NOT EXISTS idx_file_metadata_uploaded_by ON file_metadata(uploaded_by);

-- API Keys
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_lookup_hash ON api_keys(key_lookup_hash);

-- Webhooks
CREATE INDEX IF NOT EXISTS idx_webhooks_created_by ON webhooks(created_by);

-- Webhook Deliveries
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);

-- Jobs
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_idempotency_key ON jobs(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_jobs_pending ON jobs(run_after, status)
    WHERE status IN ('pending', 'processing');

-- Dead Letter Jobs
CREATE INDEX IF NOT EXISTS idx_dead_letter_jobs_type ON dead_letter_jobs(type);

-- Permission Registry
CREATE INDEX IF NOT EXISTS idx_permission_registry_domain ON permission_registry(domain);

-- Global Templates
CREATE INDEX IF NOT EXISTS idx_global_templates_name ON global_templates(name);

-- User Identities
CREATE INDEX IF NOT EXISTS idx_user_identities_user ON user_identities(user_id);

-- User Verifications
CREATE INDEX IF NOT EXISTS idx_user_verifications_token ON user_verifications(token_hash);
CREATE INDEX IF NOT EXISTS idx_user_verifications_user ON user_verifications(user_id);

-- Notifications
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id, is_read)
    WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC);

-- Notification Preferences
CREATE INDEX IF NOT EXISTS idx_notification_prefs_user ON user_notification_preferences(user_id);

-- User Preferences
CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON user_preferences(user_id);



-- ═════════════════════════════════════════════════════════════════════════════
-- SEED DATA
-- ═════════════════════════════════════════════════════════════════════════════

-- Default roles
INSERT INTO roles (name, description) VALUES
    ('super_admin', 'Unrestricted system access — all permissions granted'),
    ('admin',       'Full system access — manage users, roles, settings'),
    ('manager',     'Can manage users, roles, and view most resources'),
    ('viewer',      'Read-only access to system resources')
ON CONFLICT (name) DO NOTHING;

-- Seed core permission registry entries grouped by domain
INSERT INTO permission_registry (domain, name, description) VALUES
    ('identity',    'read:users',               'View system users'),
    ('identity',    'manage:users',             'Invite / remove system users'),
    ('roles',       'read:roles',               'View roles'),
    ('roles',       'manage:roles',             'Create, edit, delete roles'),
    ('permissions', 'read:permissions',         'View permissions'),
    ('permissions', 'manage:permissions',       'Create, edit, delete permissions'),
    ('permissions', 'manage:role_permissions',  'Assign/revoke permissions on roles'),
    ('audit',       'read:audit_logs',          'View audit log entries'),
    ('settings',    'read:settings',            'View system settings'),
    ('settings',    'manage:settings',          'Change system settings'),
    ('settings',    'manage:feature_flags',     'Toggle feature flags'),
    ('storage',     'read:files',               'View and download files'),
    ('storage',     'write:files',              'Upload files'),
    ('storage',     'manage:files',             'Upload and modify files'),
    ('storage',     'delete:files',             'Permanently delete files'),
    ('api-keys',    'read:api_keys',            'View API keys'),
    ('api-keys',    'manage:api_keys',          'Create and revoke API keys'),
    ('webhooks',    'read:webhooks',            'View webhooks'),
    ('webhooks',    'manage:webhooks',          'Create, edit, delete webhooks'),
    ('sessions',    'read:sessions',            'View active sessions')
ON CONFLICT (name) DO NOTHING;

-- Seed RBAC permissions (used by middleware for access control)
INSERT INTO permissions (name, description) VALUES
    ('read:users',               'View system users'),
    ('manage:users',             'Invite / remove system users'),
    ('read:roles',               'View roles'),
    ('manage:roles',             'Create, edit, delete roles'),
    ('read:permissions',         'View permissions'),
    ('manage:permissions',       'Create, edit, delete permissions'),
    ('manage:role_permissions',  'Assign/revoke permissions on roles'),
    ('read:audit_logs',          'View audit log entries'),
    ('read:settings',           'View system settings'),
    ('manage:settings',          'Change system settings'),
    ('manage:feature_flags',     'Toggle feature flags'),
    ('read:files',               'View and download files'),
    ('write:files',              'Upload files'),
    ('manage:files',             'Upload and modify files'),
    ('delete:files',             'Permanently delete files'),
    ('read:api_keys',            'View API keys'),
    ('manage:api_keys',          'Create and revoke API keys'),
    ('read:webhooks',            'View webhooks'),
    ('manage:webhooks',          'Create, edit, delete webhooks'),
    ('read:sessions',            'View active sessions')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to super_admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- Seed system global templates
INSERT INTO global_templates (name, description, permissions) VALUES
    ('Admin',   'Full system access — all permissions granted',
     '["read:users","manage:users","read:roles","manage:roles","read:permissions","manage:permissions","manage:role_permissions","read:audit_logs","manage:settings","read:files","manage:files","delete:files","read:api_keys","manage:api_keys"]'),
    ('Manager', 'Can manage users, roles and view most resources',
     '["read:users","read:roles","manage:roles","read:permissions","manage:role_permissions","read:audit_logs","read:files","read:api_keys"]'),
    ('Viewer',  'Read-only access to system content',
     '["read:users","read:roles","read:permissions","read:files"]'),
    ('Custom',  'Starter template — customize as needed',
     '[]')
ON CONFLICT (name) DO NOTHING;


-- ═════════════════════════════════════════════════════════════════════════════
-- AUDIT TRIGGER FUNCTIONS
-- ═════════════════════════════════════════════════════════════════════════════

-- Main audit trigger function
-- Captures mutations to audit_logs. Actor identity and action are passed
-- via session-level settings: SET app.actor_id = '123', SET app.action = 'action_name'.
CREATE OR REPLACE FUNCTION audit_trigger_func()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_action          text;
    v_actor_id        bigint;
    v_impersonator_id bigint;
    v_before_json     jsonb;
    v_after_json      jsonb;
BEGIN
    BEGIN
        v_action := current_setting('app.action', true);
        IF v_action IS NULL OR v_action = '' THEN
            v_action := TG_OP || '_' || UPPER(TG_TABLE_NAME);
        END IF;
    EXCEPTION WHEN OTHERS THEN
        v_action := TG_OP || '_' || UPPER(TG_TABLE_NAME);
    END;

    BEGIN
        v_actor_id := current_setting('app.actor_id', true)::bigint;
    EXCEPTION WHEN OTHERS THEN
        v_actor_id := NULL;
    END;

    BEGIN
        v_impersonator_id := current_setting('app.impersonator_id', true)::bigint;
    EXCEPTION WHEN OTHERS THEN
        v_impersonator_id := NULL;
    END;

    IF TG_OP = 'DELETE' THEN
        v_before_json := to_jsonb(OLD) - 'password_hash'
            - 'secret' - 'key_hash' - 'key_lookup_hash' - 'token_hash';
    ELSIF TG_OP = 'UPDATE' THEN
        v_before_json := to_jsonb(OLD) - 'password_hash'
            - 'secret' - 'key_hash' - 'key_lookup_hash' - 'token_hash';
        v_after_json  := to_jsonb(NEW) - 'password_hash'
            - 'secret' - 'key_hash' - 'key_lookup_hash' - 'token_hash';
    ELSIF TG_OP = 'INSERT' THEN
        v_after_json := to_jsonb(NEW) - 'password_hash'
            - 'secret' - 'key_hash' - 'key_lookup_hash' - 'token_hash';
    END IF;

    INSERT INTO audit_logs (actor_id, impersonator_id, action, target_entity, before_state, after_state)
    VALUES (
        v_actor_id,
        v_impersonator_id,
        v_action,
        TG_TABLE_NAME,
        v_before_json,
        v_after_json
    );

    RETURN COALESCE(NEW, OLD);
END;
$$;

-- Session cleanup function
CREATE OR REPLACE FUNCTION prune_expired_sessions()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    pruned INTEGER;
BEGIN
    DELETE FROM sessions
    WHERE expires_at < NOW()
       OR revoked_at IS NOT NULL;

    GET DIAGNOSTICS pruned = ROW_COUNT;
    RETURN pruned;
END;
$$;

-- Audit log pruning function
CREATE OR REPLACE FUNCTION prune_audit_logs(retention_days INTEGER DEFAULT 365)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    pruned INTEGER;
BEGIN
    DELETE FROM audit_logs
    WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS pruned = ROW_COUNT;
    RETURN pruned;
END;
$$;

-- Apply audit triggers to core tables
CREATE TRIGGER trg_audit_users
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_roles
    AFTER INSERT OR UPDATE OR DELETE ON roles
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_permissions
    AFTER INSERT OR UPDATE OR DELETE ON permissions
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_role_permissions
    AFTER INSERT OR DELETE ON role_permissions
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_sessions
    AFTER INSERT OR UPDATE OR DELETE ON sessions
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_settings
    AFTER INSERT OR UPDATE OR DELETE ON settings
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_feature_flags
    AFTER INSERT OR UPDATE OR DELETE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_file_metadata
    AFTER INSERT OR UPDATE OR DELETE ON file_metadata
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_api_keys
    AFTER INSERT OR UPDATE OR DELETE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_webhooks
    AFTER INSERT OR UPDATE OR DELETE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_webhook_deliveries
    AFTER INSERT OR UPDATE OR DELETE ON webhook_deliveries
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_permission_registry
    AFTER INSERT OR UPDATE OR DELETE ON permission_registry
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_global_templates
    AFTER INSERT OR UPDATE OR DELETE ON global_templates
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_notifications
    AFTER INSERT OR UPDATE OR DELETE ON notifications
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_user_notification_preferences
    AFTER INSERT OR UPDATE OR DELETE ON user_notification_preferences
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

-- Missing audit triggers (Phase 1b) — add audit coverage for remaining tables
CREATE TRIGGER trg_audit_jobs
    AFTER INSERT OR UPDATE OR DELETE ON jobs
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_dead_letter_jobs
    AFTER INSERT OR UPDATE OR DELETE ON dead_letter_jobs
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_user_identities
    AFTER INSERT OR UPDATE OR DELETE ON user_identities
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_user_verifications
    AFTER INSERT OR UPDATE OR DELETE ON user_verifications
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_user_preferences
    AFTER INSERT OR UPDATE OR DELETE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
