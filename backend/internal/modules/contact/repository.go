package contact

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	CreateContact(ctx context.Context, c *Contact) error
	GetContactByID(ctx context.Context, id int64) (*Contact, error)
	ListContacts(ctx context.Context, limit, cursor int64, status string) ([]Contact, error)
	UpdateContactStatus(ctx context.Context, id int64, status string) error
	AssignContact(ctx context.Context, id int64, userID int64) error
	DeleteContact(ctx context.Context, id int64) error

	CreateSubscriber(ctx context.Context, s *NewsletterSubscriber) error
	GetSubscriberByEmail(ctx context.Context, email string) (*NewsletterSubscriber, error)
	ListSubscribers(ctx context.Context, limit, cursor int64) ([]NewsletterSubscriber, error)
	Unsubscribe(ctx context.Context, email string) error
	DeleteSubscriber(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) CreateContact(ctx context.Context, c *Contact) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO contacts (name, email, phone, subject, message, source)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		c.Name, c.Email, c.Phone, c.Subject, c.Message, c.Source,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *pgRepository) GetContactByID(ctx context.Context, id int64) (*Contact, error) {
	q := database.GetQueryer(ctx, r.db)
	var c Contact
	err := q.QueryRowContext(ctx,
		`SELECT id, name, email, phone, subject, message, source, status, assigned_to, created_at, updated_at
		 FROM contacts WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Subject, &c.Message, &c.Source, &c.Status, &c.AssignedTo, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *pgRepository) ListContacts(ctx context.Context, limit, cursor int64, status string) ([]Contact, error) {
	q := database.GetQueryer(ctx, r.db)
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = q.QueryContext(ctx,
			`SELECT id, name, email, phone, subject, message, source, status, assigned_to, created_at, updated_at
			 FROM contacts WHERE id > $2 AND status = $3 ORDER BY id ASC LIMIT $1`,
			limit, cursor, status,
		)
	} else {
		rows, err = q.QueryContext(ctx,
			`SELECT id, name, email, phone, subject, message, source, status, assigned_to, created_at, updated_at
			 FROM contacts WHERE id > $2 ORDER BY id ASC LIMIT $1`,
			limit, cursor,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Subject, &c.Message, &c.Source, &c.Status, &c.AssignedTo, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *pgRepository) UpdateContactStatus(ctx context.Context, id int64, status string) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE contacts SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *pgRepository) AssignContact(ctx context.Context, id int64, userID int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE contacts SET assigned_to = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		userID, id,
	)
	return err
}

func (r *pgRepository) DeleteContact(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM contacts WHERE id = $1`, id)
	return err
}

func (r *pgRepository) CreateSubscriber(ctx context.Context, s *NewsletterSubscriber) error {
	q := database.GetQueryer(ctx, r.db)
	metadata := []byte("{}")
	if len(s.Metadata) > 0 {
		metadata = s.Metadata
	}
	return q.QueryRowContext(ctx,
		`INSERT INTO newsletter_subscribers (email, name, source, metadata)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (email) DO UPDATE SET
		   unsubscribed_at = NULL,
		   name = EXCLUDED.name,
		   source = EXCLUDED.source,
		   metadata = EXCLUDED.metadata
		 RETURNING id, subscribed_at`,
		s.Email, s.Name, s.Source, metadata,
	).Scan(&s.ID, &s.SubscribedAt)
}

func (r *pgRepository) GetSubscriberByEmail(ctx context.Context, email string) (*NewsletterSubscriber, error) {
	q := database.GetQueryer(ctx, r.db)
	var s NewsletterSubscriber
	err := q.QueryRowContext(ctx,
		`SELECT id, email, name, source, metadata, subscribed_at, unsubscribed_at
		 FROM newsletter_subscribers WHERE email = $1`, email,
	).Scan(&s.ID, &s.Email, &s.Name, &s.Source, &s.Metadata, &s.SubscribedAt, &s.UnsubscribedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgRepository) ListSubscribers(ctx context.Context, limit, cursor int64) ([]NewsletterSubscriber, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT id, email, name, source, metadata, subscribed_at, unsubscribed_at
		 FROM newsletter_subscribers WHERE id > $2 ORDER BY id ASC LIMIT $1`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NewsletterSubscriber
	for rows.Next() {
		var s NewsletterSubscriber
		if err := rows.Scan(&s.ID, &s.Email, &s.Name, &s.Source, &s.Metadata, &s.SubscribedAt, &s.UnsubscribedAt); err != nil {
			return nil, err
		}
		if s.Metadata == nil {
			s.Metadata = json.RawMessage("{}")
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *pgRepository) Unsubscribe(ctx context.Context, email string) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE newsletter_subscribers SET unsubscribed_at = NOW() WHERE email = $1 AND unsubscribed_at IS NULL`,
		email,
	)
	return err
}

func (r *pgRepository) DeleteSubscriber(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM newsletter_subscribers WHERE id = $1`, id)
	return err
}
