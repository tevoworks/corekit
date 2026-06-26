ALTER TABLE file_metadata ALTER COLUMN storage_path TYPE VARCHAR(500);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'jobs' AND column_name = 'final_status') THEN
        ALTER TABLE jobs DROP COLUMN final_status;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_audit_logs_actor_created;
DROP INDEX IF EXISTS idx_audit_logs_entity_created;
DROP INDEX IF EXISTS idx_jobs_type_status;
DROP INDEX IF EXISTS idx_sessions_revoked_at;

ALTER TABLE user_notification_preferences DROP CONSTRAINT IF EXISTS chk_notification_prefs_channel;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS chk_notifications_type;
ALTER TABLE user_identities DROP CONSTRAINT IF EXISTS chk_user_identities_provider;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS chk_jobs_status;
ALTER TABLE webhook_deliveries DROP CONSTRAINT IF EXISTS chk_webhook_deliveries_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;

ALTER TABLE dead_letter_jobs DROP CONSTRAINT IF EXISTS dead_letter_jobs_original_job_id_fkey;
ALTER TABLE webhooks DROP CONSTRAINT IF EXISTS webhooks_created_by_fkey;
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_created_by_fkey;

DROP TRIGGER IF EXISTS trg_audit_user_preferences ON user_preferences;
DROP TRIGGER IF EXISTS trg_audit_user_verifications ON user_verifications;
DROP TRIGGER IF EXISTS trg_audit_user_identities ON user_identities;
DROP TRIGGER IF EXISTS trg_audit_dead_letter_jobs ON dead_letter_jobs;
DROP TRIGGER IF EXISTS trg_audit_jobs ON jobs;

CREATE OR REPLACE FUNCTION audit_trigger_func()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_action          text;
    v_actor_id        bigint;
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

    IF TG_OP = 'DELETE' THEN
        v_before_json := to_jsonb(OLD) - 'password_hash';
    ELSIF TG_OP = 'UPDATE' THEN
        v_before_json := to_jsonb(OLD) - 'password_hash';
        v_after_json  := to_jsonb(NEW) - 'password_hash';
    ELSIF TG_OP = 'INSERT' THEN
        v_after_json := to_jsonb(NEW) - 'password_hash';
    END IF;

    INSERT INTO audit_logs (actor_id, action, target_entity, before_state, after_state)
    VALUES (
        v_actor_id,
        v_action,
        TG_TABLE_NAME,
        v_before_json,
        v_after_json
    );

    RETURN COALESCE(NEW, OLD);
END;
$$;
