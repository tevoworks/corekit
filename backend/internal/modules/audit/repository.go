package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, limit int, cursor int64) ([]AuditLog, error)
	ListFiltered(ctx context.Context, actorID *int64, action, dateFrom, dateTo string, limit int, cursor int64) ([]AuditLog, error)
	AnonymizeAuditPII(ctx context.Context, userID int64, email string) error
	PruneAuditLogs(ctx context.Context, retentionDays int) (int, error)
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (actor_id, impersonator_id, action, target_entity, before_state, after_state)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query,
		log.ActorID,
		log.ImpersonatorID,
		log.Action,
		log.TargetEntity,
		log.BeforeState,
		log.AfterState,
	).Scan(&log.ID, &log.CreatedAt)
	return err
}

const auditLogColumns = `al.id, al.actor_id, al.impersonator_id, al.action, al.target_entity, al.before_state, al.after_state, al.created_at`
const auditLogJoin = `LEFT JOIN users u ON al.actor_id = u.id`

func (r *pgRepository) scanAuditLogs(rows *sql.Rows) ([]AuditLog, error) {
	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		var beforeBytes, afterBytes []byte
		err := rows.Scan(
			&l.ID,
			&l.ActorID,
			&l.ImpersonatorID,
			&l.Action,
			&l.TargetEntity,
			&beforeBytes,
			&afterBytes,
			&l.CreatedAt,
			&l.ActorName,
			&l.ActorEmail,
		)
		if err != nil {
			return nil, err
		}
		l.BeforeState = beforeBytes
		l.AfterState = afterBytes
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *pgRepository) List(ctx context.Context, limit int, cursor int64) ([]AuditLog, error) {
	args := []interface{}{}
	whereClause := ""
	if cursor > 0 {
		whereClause = "WHERE al.id < $1"
		args = append(args, cursor)
	}
	query := fmt.Sprintf(`
		SELECT %s, u.full_name AS actor_name, u.email AS actor_email
		FROM audit_logs al
		%s
		%s
		ORDER BY al.id DESC
		LIMIT $%d`, auditLogColumns, auditLogJoin, whereClause, len(args)+1)
	args = append(args, limit)

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAuditLogs(rows)
}

func (r *pgRepository) ListFiltered(ctx context.Context, actorID *int64, action, dateFrom, dateTo string, limit int, cursor int64) ([]AuditLog, error) {
	var conditions []string
	args := []interface{}{}
	paramIdx := 1

	if cursor > 0 {
		conditions = append(conditions, fmt.Sprintf("al.id < $%d", paramIdx))
		args = append(args, cursor)
		paramIdx++
	}
	if actorID != nil {
		conditions = append(conditions, fmt.Sprintf("al.actor_id = $%d", paramIdx))
		args = append(args, *actorID)
		paramIdx++
	}
	if action != "" {
		conditions = append(conditions, fmt.Sprintf("al.action = $%d", paramIdx))
		args = append(args, action)
		paramIdx++
	}
	if dateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("al.created_at >= $%d", paramIdx))
		args = append(args, dateFrom)
		paramIdx++
	}
	if dateTo != "" {
		conditions = append(conditions, fmt.Sprintf("al.created_at <= $%d", paramIdx))
		args = append(args, dateTo)
		paramIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT %s, u.full_name AS actor_name, u.email AS actor_email
		FROM audit_logs al
		%s
		%s
		ORDER BY al.id DESC
		LIMIT $%d`,
		auditLogColumns,
		auditLogJoin,
		whereClause,
		paramIdx,
	)

	allArgs := append(args, limit)

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAuditLogs(rows)
}

func (r *pgRepository) AnonymizeAuditPII(ctx context.Context, userID int64, email string) error {
	q := database.GetQueryer(ctx, r.db)

	redact := func(col string) error {
		query := fmt.Sprintf(`
			UPDATE audit_logs
			SET %s = jsonb_set(
				COALESCE(%s, '{}'::jsonb),
				'{email}',
				'"REDACTED"',
				true
			) #- '{full_name}' #- '{password_hash}' #- '{avatar_url}'
			WHERE %s IS NOT NULL
			  AND (actor_id = $1 OR (%s @> jsonb_build_object('email', $2::text)))`, col, col, col, col)
		_, err := q.ExecContext(ctx, query, userID, email)
		return err
	}

	if err := redact("before_state"); err != nil {
		return err
	}
	if err := redact("after_state"); err != nil {
		return err
	}
	return nil
}

func (r *pgRepository) PruneAuditLogs(ctx context.Context, retentionDays int) (int, error) {
	var total int
	for {
		res, err := r.db.ExecContext(ctx,
			`DELETE FROM audit_logs
			 WHERE id IN (
			     SELECT id FROM audit_logs
			     WHERE created_at < NOW() - ($1 || ' days')::INTERVAL
			     ORDER BY id
			     LIMIT 10000
			 )`, retentionDays)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += int(n)
		if n == 0 {
			break
		}
	}
	return total, nil
}
