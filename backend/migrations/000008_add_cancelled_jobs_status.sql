-- Add 'cancelled' to jobs CHECK constraint.
-- Cancel() now marks jobs as 'cancelled' instead of hard-deleting them,
-- preserving the audit trail.

ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS chk_jobs_status,
    ADD CONSTRAINT chk_jobs_status
        CHECK (status IN ('pending', 'processing', 'done', 'failed', 'cancelled'));
