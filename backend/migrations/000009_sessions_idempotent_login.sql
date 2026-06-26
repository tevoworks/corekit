-- Add UNIQUE constraint on sessions(user_id, token_id) for idempotent login.
-- Combined with INSERT ... ON CONFLICT DO NOTHING in CreateSession,
-- this prevents concurrent login race conditions from creating duplicate sessions.

CREATE UNIQUE INDEX IF NOT EXISTS uq_sessions_user_id_token_id ON sessions(user_id, token_id);
