-- Add absolute_expires_at to sessions for absolute session lifetime enforcement.
-- A session can be refreshed up to expires_at, but never beyond absolute_expires_at.
-- Default: 7 days from creation (grace period for existing sessions).

ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS absolute_expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '7 days');

-- Update existing rows to have a sensible absolute_expires_at based on their created_at.
UPDATE sessions
SET absolute_expires_at = created_at + INTERVAL '7 days'
WHERE absolute_expires_at IS NULL
   OR absolute_expires_at = created_at + INTERVAL '7 days';
