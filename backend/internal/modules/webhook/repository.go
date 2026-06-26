package webhook

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/lib/pq"
	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, wh *Webhook) error
	Update(ctx context.Context, wh *Webhook) error
	GetByID(ctx context.Context, id int64) (*Webhook, error)
	List(ctx context.Context, limit int, cursor int64) ([]Webhook, error)
	Delete(ctx context.Context, id int64) error
	CreateDelivery(ctx context.Context, d *WebhookDelivery) error
	GetDeliveryByID(ctx context.Context, id, webhookID int64) (*WebhookDelivery, error)
	ListDeliveries(ctx context.Context, webhookID int64, limit int, cursor int64) ([]WebhookDelivery, error)
	UpdateDelivery(ctx context.Context, d *WebhookDelivery) error
	SetDeliveryRetrying(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, wh *Webhook) error {
	query := `
		INSERT INTO webhooks (name, url, events, secret, active, created_by, resolved_ips)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query,
		wh.Name, wh.URL, pq.Array(wh.Events), wh.Secret, wh.Active, wh.CreatedBy, pq.Array(wh.ResolvedIPs),
	).Scan(&wh.ID, &wh.CreatedAt, &wh.UpdatedAt)
}

func (r *pgRepository) Update(ctx context.Context, wh *Webhook) error {
	query := `
		UPDATE webhooks
		SET name = $1, url = $2, events = $3, secret = $4, active = $5, resolved_ips = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
		RETURNING updated_at`
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query,
		wh.Name, wh.URL, pq.Array(wh.Events), wh.Secret, wh.Active, pq.Array(wh.ResolvedIPs), wh.ID,
	).Scan(&wh.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("webhook not found")
		}
		return err
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id int64) (*Webhook, error) {
	query := `
		SELECT id, name, url, events, secret, active, created_by, created_at, updated_at, resolved_ips
		FROM webhooks
		WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	var wh Webhook
	err := q.QueryRowContext(ctx, query, id).Scan(
		&wh.ID, &wh.Name, &wh.URL, pq.Array(&wh.Events), &wh.Secret,
		&wh.Active, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt, pq.Array(&wh.ResolvedIPs),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &wh, nil
}

func (r *pgRepository) List(ctx context.Context, limit int, cursor int64) ([]Webhook, error) {
	query := `
		SELECT id, name, url, events, secret, active, created_by, created_at, updated_at, resolved_ips
		FROM webhooks
		WHERE ($2 = 0 OR id < $2)
		ORDER BY id DESC
		LIMIT $1`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Webhook
	for rows.Next() {
		var wh Webhook
		if err := rows.Scan(
			&wh.ID, &wh.Name, &wh.URL, pq.Array(&wh.Events), &wh.Secret,
			&wh.Active, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt, pq.Array(&wh.ResolvedIPs),
		); err != nil {
			return nil, err
		}
		list = append(list, wh)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("webhook not found")
	}
	return nil
}

func (r *pgRepository) CreateDelivery(ctx context.Context, d *WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (webhook_id, event, status, request_body, response_body, response_code, duration_ms, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query,
		d.WebhookID, d.Event, d.Status, d.RequestBody, d.ResponseBody,
		d.ResponseCode, d.DurationMs, d.ErrorMessage,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *pgRepository) UpdateDelivery(ctx context.Context, d *WebhookDelivery) error {
	query := `
		UPDATE webhook_deliveries
		SET status = $1, response_body = $2, response_code = $3, duration_ms = $4, error_message = $5
		WHERE id = $6`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, d.Status, d.ResponseBody, d.ResponseCode, d.DurationMs, d.ErrorMessage, d.ID)
	return err
}

func (r *pgRepository) ListDeliveries(ctx context.Context, webhookID int64, limit int, cursor int64) ([]WebhookDelivery, error) {
	q := database.GetQueryer(ctx, r.db)

	args := []interface{}{webhookID}
	query := `
		SELECT id, webhook_id, event, status, request_body, response_body, response_code, duration_ms, error_message, created_at
		FROM webhook_deliveries
		WHERE webhook_id = $1`
	if cursor > 0 {
		query += ` AND id < $2`
		args = append(args, cursor)
	}
	query += ` ORDER BY id DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(
			&d.ID, &d.WebhookID, &d.Event, &d.Status, &d.RequestBody, &d.ResponseBody,
			&d.ResponseCode, &d.DurationMs, &d.ErrorMessage, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) GetDeliveryByID(ctx context.Context, id, webhookID int64) (*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, status, request_body, response_body, response_code, duration_ms, error_message, created_at
		FROM webhook_deliveries
		WHERE id = $1 AND webhook_id = $2`
	q := database.GetQueryer(ctx, r.db)
	var d WebhookDelivery
	err := q.QueryRowContext(ctx, query, id, webhookID).Scan(
		&d.ID, &d.WebhookID, &d.Event, &d.Status, &d.RequestBody, &d.ResponseBody,
		&d.ResponseCode, &d.DurationMs, &d.ErrorMessage, &d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *pgRepository) SetDeliveryRetrying(ctx context.Context, id int64) error {
	query := `UPDATE webhook_deliveries SET status = $1 WHERE id = $2`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, DeliveryStatusRetrying, id)
	return err
}
