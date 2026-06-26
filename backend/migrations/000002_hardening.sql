-- ─────────────────────────────────────────────────────────────────────────────
-- CoreKit — Hardening Migration (Phase 1-5)
-- Applies fixes on top of 000001_init.sql:
--   1a. impersonator_id capture in audit trigger
--   1b. Missing audit triggers (jobs, dead_letter_jobs, user_identities,
--       user_verifications, user_preferences)
--   1c. Sensitive field exclusion (secret, key_hash, key_lookup_hash, token_hash)
--   5a. FK ON DELETE clauses for api_keys, webhooks
--   5b. CHECK constraints on enum-like VARCHAR columns
--   5c. Missing indexes (revoked_at, composite indexes)
--   5d. FK constraint for dead_letter_jobs.original_job_id
-- ─────────────────────────────────────────────────────────────────────────────

-- ═════════════════════════════════════════════════════════════════════════════
-- 1. AUDIT TRIGGER — impersonator_id + sensitive field stripping
-- ═════════════════════════════════════════════════════════════════════════════

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


-- ═════════════════════════════════════════════════════════════════════════════
-- 2. MISSING AUDIT TRIGGERS (Phase 1b)
-- ═════════════════════════════════════════════════════════════════════════════

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_jobs') THEN
        CREATE TRIGGER trg_audit_jobs
            AFTER INSERT OR UPDATE OR DELETE ON jobs
            FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_dead_letter_jobs') THEN
        CREATE TRIGGER trg_audit_dead_letter_jobs
            AFTER INSERT OR UPDATE OR DELETE ON dead_letter_jobs
            FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_user_identities') THEN
        CREATE TRIGGER trg_audit_user_identities
            AFTER INSERT OR UPDATE OR DELETE ON user_identities
            FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_user_verifications') THEN
        CREATE TRIGGER trg_audit_user_verifications
            AFTER INSERT OR UPDATE OR DELETE ON user_verifications
            FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_audit_user_preferences') THEN
        CREATE TRIGGER trg_audit_user_preferences
            AFTER INSERT OR UPDATE OR DELETE ON user_preferences
            FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    END IF;
END $$;


-- ═════════════════════════════════════════════════════════════════════════════
-- 3. FK CONSTRAINTS — ON DELETE actions (Phase 5a)
-- ═════════════════════════════════════════════════════════════════════════════

-- api_keys.created_by: allow delete user to cascade-delete their keys
ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_created_by_fkey,
    ADD CONSTRAINT api_keys_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;

-- webhooks.created_by: allow delete user to cascade-delete their webhooks
ALTER TABLE webhooks
    DROP CONSTRAINT IF EXISTS webhooks_created_by_fkey,
    ADD CONSTRAINT webhooks_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;

-- dead_letter_jobs.original_job_id: allow delete job without orphaning DLQ entry
ALTER TABLE dead_letter_jobs
    DROP CONSTRAINT IF EXISTS dead_letter_jobs_original_job_id_fkey,
    ADD CONSTRAINT dead_letter_jobs_original_job_id_fkey
        FOREIGN KEY (original_job_id) REFERENCES jobs(id) ON DELETE SET NULL;


-- ═════════════════════════════════════════════════════════════════════════════
-- 4. CHECK CONSTRAINTS — enum-like VARCHAR columns (Phase 5b)
-- ═════════════════════════════════════════════════════════════════════════════

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS chk_users_status,
    ADD CONSTRAINT chk_users_status
        CHECK (status IN ('PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'BANNED', 'FORCE_PASSWORD_RESET'));

ALTER TABLE webhook_deliveries
    DROP CONSTRAINT IF EXISTS chk_webhook_deliveries_status,
    ADD CONSTRAINT chk_webhook_deliveries_status
        CHECK (status IN ('pending', 'delivering', 'delivered', 'failed', 'retrying'));

ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS chk_jobs_status,
    ADD CONSTRAINT chk_jobs_status
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled', 'retrying'));

ALTER TABLE user_identities
    DROP CONSTRAINT IF EXISTS chk_user_identities_provider,
    ADD CONSTRAINT chk_user_identities_provider
        CHECK (provider IN ('google', 'github', 'apple', 'email', 'sso'));

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS chk_notifications_type,
    ADD CONSTRAINT chk_notifications_type
        CHECK (type IN ('system', 'warning', 'info', 'error', 'alert'));

ALTER TABLE user_notification_preferences
    DROP CONSTRAINT IF EXISTS chk_notification_prefs_channel,
    ADD CONSTRAINT chk_notification_prefs_channel
        CHECK (channel IN ('in_app', 'email', 'sms', 'push'));


-- ═════════════════════════════════════════════════════════════════════════════
-- 5. MISSING INDEXES (Phase 5c)
-- ═════════════════════════════════════════════════════════════════════════════

-- sessions.revoked_at — query used by prune_expired_sessions()
CREATE INDEX IF NOT EXISTS idx_sessions_revoked_at
    ON sessions(revoked_at) WHERE revoked_at IS NOT NULL;

-- jobs composite (type, status) — worker polling pattern
CREATE INDEX IF NOT EXISTS idx_jobs_type_status
    ON jobs(type, status);

-- audit_logs composite — most common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_created
    ON audit_logs(target_entity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_created
    ON audit_logs(actor_id, created_at DESC);


-- ═════════════════════════════════════════════════════════════════════════════
-- 6. CLEANUP — remove redundant final_status column from jobs (Phase 5b)
-- ═════════════════════════════════════════════════════════════════════════════
-- The final_status boolean must stay in sync with status — drop and derive.
-- Only run if the column still exists (fresh migrations already omit it).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'jobs' AND column_name = 'final_status') THEN
        ALTER TABLE jobs DROP COLUMN final_status;
    END IF;
END $$;


-- ═════════════════════════════════════════════════════════════════════════════
-- 7. STORAGE_PATH — widen to accommodate long S3 keys (Phase 5e)
-- ═════════════════════════════════════════════════════════════════════════════

ALTER TABLE file_metadata
    ALTER COLUMN storage_path TYPE VARCHAR(1024);

