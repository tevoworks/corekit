-- Add expires_at and rotated_at columns for API key rotation policy.
-- Keys expire 90 days after creation unless rotated.

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '90 days');
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMP;

UPDATE api_keys SET expires_at = created_at + INTERVAL '90 days' WHERE expires_at IS NULL;
