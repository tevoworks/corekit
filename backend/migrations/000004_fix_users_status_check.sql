-- Align users.status CHECK constraint with Go code.
-- Go code uses 'HALTED' not 'BANNED'.
-- 'LOCKED' is not a status value (locking uses locked_until column).
-- Existing 'BANNED' rows (if any) are migrated to 'SUSPENDED'.

UPDATE users SET status = 'SUSPENDED' WHERE status = 'BANNED';

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS chk_users_status,
    ADD CONSTRAINT chk_users_status
        CHECK (status IN ('PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'HALTED', 'FORCE_PASSWORD_RESET'));
