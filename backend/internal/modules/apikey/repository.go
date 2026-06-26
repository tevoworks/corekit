package apikey

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	List(ctx context.Context) ([]APIKey, error)
	Revoke(ctx context.Context, id int64) error
	Rotate(ctx context.Context, id int64, newHash, newLookupHash, newPrefix string, expiresAt time.Time) error
	ListExpiring(ctx context.Context, withinDays int) ([]APIKey, error)
	TouchLastUsed(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

const apiKeyColumns = `id, name, key_prefix, key_hash, key_lookup_hash, created_by, revoked_at, last_used_at, expires_at, rotated_at, created_at`

func scanAPIKey(k *APIKey, sc interface{ Scan(dest ...interface{}) error }) error {
	return sc.Scan(
		&k.ID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.KeyLookupHash, &k.CreatedBy,
		&k.RevokedAt, &k.LastUsedAt, &k.ExpiresAt, &k.RotatedAt, &k.CreatedAt,
	)
}

func (r *pgRepository) Create(ctx context.Context, key *APIKey) error {
	query := `
		INSERT INTO api_keys (name, key_prefix, key_hash, key_lookup_hash, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, key.Name, key.KeyPrefix, key.KeyHash, key.KeyLookupHash, key.CreatedBy, key.ExpiresAt).
		Scan(&key.ID, &key.CreatedAt)
}

func (r *pgRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `SELECT ` + apiKeyColumns + ` FROM api_keys WHERE key_lookup_hash = $1`
	var k APIKey
	q := database.GetQueryer(ctx, r.db)
	err := scanAPIKey(&k, q.QueryRowContext(ctx, query, keyHash))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

func (r *pgRepository) List(ctx context.Context) ([]APIKey, error) {
	query := `SELECT ` + apiKeyColumns + ` FROM api_keys ORDER BY created_at DESC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []APIKey
	for rows.Next() {
		var k APIKey
		if err := scanAPIKey(&k, rows); err != nil {
			return nil, err
		}
		list = append(list, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) Revoke(ctx context.Context, id int64) error {
	query := `
		UPDATE api_keys
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND revoked_at IS NULL`
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
		return errors.New("API key not found or already revoked")
	}
	return nil
}

func (r *pgRepository) Rotate(ctx context.Context, id int64, newHash, newLookupHash, newPrefix string, expiresAt time.Time) error {
	query := `
		UPDATE api_keys
		SET key_hash = $1, key_lookup_hash = $2, key_prefix = $3, expires_at = $4, rotated_at = CURRENT_TIMESTAMP, revoked_at = NULL, last_used_at = NULL
		WHERE id = $5 AND revoked_at IS NULL`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, newHash, newLookupHash, newPrefix, expiresAt, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("API key not found or was revoked")
	}
	return nil
}

func (r *pgRepository) ListExpiring(ctx context.Context, withinDays int) ([]APIKey, error) {
	query := `SELECT ` + apiKeyColumns + ` FROM api_keys WHERE revoked_at IS NULL AND expires_at <= CURRENT_TIMESTAMP + $1::INTERVAL ORDER BY expires_at ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, fmt.Sprintf("%d days", withinDays))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []APIKey
	for rows.Next() {
		var k APIKey
		if err := scanAPIKey(&k, rows); err != nil {
			return nil, err
		}
		list = append(list, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) TouchLastUsed(ctx context.Context, id int64) error {
	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, id)
	return err
}
