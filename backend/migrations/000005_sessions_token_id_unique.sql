-- Add UNIQUE constraint on sessions.token_id.
-- The auth middleware and authverify both query by token_id — duplicates
-- would cause runtime errors. Clean up any duplicates first (keep most recent).

DELETE FROM sessions a USING sessions b
WHERE a.token_id = b.token_id AND a.created_at < b.created_at;

ALTER TABLE sessions ADD CONSTRAINT uq_sessions_token_id UNIQUE (token_id);
