package queue

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Enqueue(ctx context.Context, tx *sql.Tx, typeStr string, payload []byte, idempotencyKey *string) error
	Claim(ctx context.Context, workerID string, batchSize int) ([]Job, error)
	Heartbeat(ctx context.Context, jobID int64, workerID string) error
	Complete(ctx context.Context, jobID int64) error
	Fail(ctx context.Context, jobID int64, errMsg string, nextRetry *time.Time) error
	RecoverStuck(ctx context.Context) (int64, error)
	List(ctx context.Context, status, jobType string, cursor int64, limit int) ([]Job, error)
	Retry(ctx context.Context, jobID int64) error
	Cancel(ctx context.Context, jobID int64) error
	PruneExpired(ctx context.Context, age time.Duration) (int64, error)
	Vacuum(ctx context.Context) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Enqueue(ctx context.Context, tx *sql.Tx, typeStr string, payload []byte, idempotencyKey *string) error {
	var key string
	if idempotencyKey == nil || *idempotencyKey == "" {
		key = uuid.New().String()
	} else {
		key = *idempotencyKey
	}

	query := `
		INSERT INTO jobs (type, payload, idempotency_key, status, retry_count, max_retries, run_after)
		VALUES ($1, $2, $3, 'pending', 0, 3, NOW())
		ON CONFLICT (idempotency_key) DO NOTHING`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, typeStr, payload, key)
	} else {
		_, err = r.db.ExecContext(ctx, query, typeStr, payload, key)
	}
	return err
}

func (r *pgRepository) Claim(ctx context.Context, workerID string, batchSize int) ([]Job, error) {
	query := `
		WITH target_jobs AS (
			SELECT id FROM jobs
			WHERE (status = 'pending' AND run_after <= NOW() AND (next_retry_at IS NULL OR next_retry_at <= NOW()))
			   OR (status = 'processing' AND locked_at < NOW() - INTERVAL '120 seconds' AND (last_heartbeat_at IS NULL OR last_heartbeat_at < NOW() - INTERVAL '60 seconds'))
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE jobs
		SET status = 'processing',
			locked_at = NOW(),
			locked_by = $2,
			last_heartbeat_at = NOW(),
			updated_at = NOW()
		WHERE id IN (SELECT id FROM target_jobs)
		RETURNING id, type, payload, retry_count, max_retries, idempotency_key`

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, batchSize, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Job
	for rows.Next() {
		var j Job
		err := rows.Scan(&j.ID, &j.Type, &j.Payload, &j.RetryCount, &j.MaxRetries, &j.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) Heartbeat(ctx context.Context, jobID int64, workerID string) error {
	query := `
		UPDATE jobs
		SET last_heartbeat_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND locked_by = $2 AND status = 'processing'`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, jobID, workerID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("heartbeat failed: job not found or locked by another worker")
	}
	return nil
}

func (r *pgRepository) Complete(ctx context.Context, jobID int64) error {
	query := `
		UPDATE jobs
		SET status = 'done',
			locked_at = NULL,
			locked_by = NULL,
			last_heartbeat_at = NULL,
			updated_at = NOW()
		WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, jobID)
	return err
}

func (r *pgRepository) Fail(ctx context.Context, jobID int64, errMsg string, nextRetry *time.Time) error {
	var query string
	if nextRetry == nil {
		query = `
			UPDATE jobs
			SET status = 'failed',
				locked_at = NULL,
				locked_by = NULL,
				last_heartbeat_at = NULL,
				error_message = $2,
				updated_at = NOW()
			WHERE id = $1`
		q := database.GetQueryer(ctx, r.db)
		_, err := q.ExecContext(ctx, query, jobID, errMsg)
		return err
	}

	query = `
		UPDATE jobs
		SET status = 'pending',
			retry_count = retry_count + 1,
			next_retry_at = $2,
			locked_at = NULL,
			locked_by = NULL,
			last_heartbeat_at = NULL,
			error_message = $3,
			updated_at = NOW()
		WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, jobID, *nextRetry, errMsg)
	return err
}

func (r *pgRepository) RecoverStuck(ctx context.Context) (int64, error) {
	query := `
		UPDATE jobs
		SET status = 'pending',
			locked_at = NULL,
			locked_by = NULL,
			last_heartbeat_at = NULL,
			updated_at = NOW()
		WHERE id IN (
			SELECT id FROM jobs
			WHERE status = 'processing'
			  AND (
				  (locked_at < NOW() - INTERVAL '300 seconds' AND last_heartbeat_at IS NULL)
				  OR (last_heartbeat_at < NOW() - INTERVAL '300 seconds')
			  )
			LIMIT 1000
		)`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *pgRepository) List(ctx context.Context, status, jobType string, cursor int64, limit int) ([]Job, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := "SELECT id, type, payload, status, retry_count, max_retries, run_after, locked_at, locked_by, last_heartbeat_at, next_retry_at, idempotency_key, error_message, created_at, updated_at FROM jobs"
	args := []interface{}{}
	argIdx := 1
	var whereClauses []string

	if status != "" {
		whereClauses = append(whereClauses, "status = $"+strconv.Itoa(argIdx))
		args = append(args, status)
		argIdx++
	}
	if jobType != "" {
		whereClauses = append(whereClauses, "type = $"+strconv.Itoa(argIdx))
		args = append(args, jobType)
		argIdx++
	}
	if cursor > 0 {
		whereClauses = append(whereClauses, "id < $"+strconv.Itoa(argIdx))
		args = append(args, cursor)
		argIdx++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + whereClauses[0]
		for _, c := range whereClauses[1:] {
			query += " AND " + c
		}
	}

	query += " ORDER BY id DESC LIMIT $" + strconv.Itoa(argIdx)
	args = append(args, limit)

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Job
	for rows.Next() {
		var j Job
		err := rows.Scan(
			&j.ID, &j.Type, &j.Payload, &j.Status, &j.RetryCount, &j.MaxRetries,
			&j.RunAfter, &j.LockedAt, &j.LockedBy, &j.LastHeartbeatAt, &j.NextRetryAt,
			&j.IdempotencyKey, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (r *pgRepository) Retry(ctx context.Context, jobID int64) error {
	query := `
		UPDATE jobs
		SET status = 'pending',
		    retry_count = 0,
		    next_retry_at = NULL,
		    error_message = NULL,
		    locked_at = NULL,
		    locked_by = NULL,
		    last_heartbeat_at = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'failed'`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, jobID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("job not found or not in retryable state")
	}
	return nil
}

func (r *pgRepository) Cancel(ctx context.Context, jobID int64) error {
	query := `
		UPDATE jobs
		SET status = 'cancelled',
		    locked_at = NULL,
		    locked_by = NULL,
		    last_heartbeat_at = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('done', 'failed', 'cancelled')`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, jobID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("job not found or already in terminal state")
	}
	return nil
}

func (r *pgRepository) PruneExpired(ctx context.Context, age time.Duration) (int64, error) {
	if age <= 0 {
		return 0, errors.New("prune age must be positive")
	}

	query := `
		DELETE FROM jobs
		WHERE status IN ('done', 'failed')
		  AND updated_at < NOW() - $1 * INTERVAL '1 second'`

	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, int64(age.Seconds()))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *pgRepository) Vacuum(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "ANALYZE jobs")
	return err
}
