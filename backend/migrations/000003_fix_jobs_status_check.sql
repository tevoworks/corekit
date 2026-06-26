-- Fix jobs status CHECK constraint to match Go code constants.
-- Go code uses 'done' (StatusDone) not 'completed'.
-- 'cancelled' and 'retrying' are never set by the code.

ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS chk_jobs_status,
    ADD CONSTRAINT chk_jobs_status
        CHECK (status IN ('pending', 'processing', 'done', 'failed'));
