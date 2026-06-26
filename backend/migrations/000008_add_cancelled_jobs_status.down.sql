ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS chk_jobs_status,
    ADD CONSTRAINT chk_jobs_status
        CHECK (status IN ('pending', 'processing', 'done', 'failed'));
