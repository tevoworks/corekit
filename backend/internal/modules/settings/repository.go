package settings

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Set(ctx context.Context, s *Setting) error
	Get(ctx context.Context, key string) (*Setting, error)
	List(ctx context.Context) ([]Setting, error)
	Delete(ctx context.Context, key string) error
	CreateFlag(ctx context.Context, f *FeatureFlag) error
	GetFlagByID(ctx context.Context, id int64) (*FeatureFlag, error)
	GetFlagByKey(ctx context.Context, key string) (*FeatureFlag, error)
	UpdateFlag(ctx context.Context, f *FeatureFlag) error
	DeleteFlag(ctx context.Context, id int64) error
	ListFlags(ctx context.Context, limit int, cursor int64) ([]FeatureFlag, error)
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Set(ctx context.Context, s *Setting) error {
	query := `
		INSERT INTO settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
		RETURNING id, updated_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, s.Key, s.Value).Scan(&s.ID, &s.UpdatedAt)
}

func (r *pgRepository) Get(ctx context.Context, key string) (*Setting, error) {
	query := `
		SELECT id, key, value, updated_at
		FROM settings
		WHERE key = $1
		LIMIT 1`

	var s Setting
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, key).
		Scan(&s.ID, &s.Key, &s.Value, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgRepository) List(ctx context.Context) ([]Setting, error) {
	query := `
		SELECT id, key, value, updated_at
		FROM settings
		ORDER BY key ASC`

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *pgRepository) Delete(ctx context.Context, key string) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, key)
	return err
}

func (r *pgRepository) CreateFlag(ctx context.Context, f *FeatureFlag) error {
	query := `
		INSERT INTO feature_flags (name, key, description, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, f.Name, f.Key, f.Description, f.Enabled).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *pgRepository) GetFlagByID(ctx context.Context, id int64) (*FeatureFlag, error) {
	query := `
		SELECT id, name, key, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE id = $1`
	var f FeatureFlag
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.Name, &f.Key, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *pgRepository) GetFlagByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	query := `
		SELECT id, name, key, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE key = $1`
	var f FeatureFlag
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, key).Scan(
		&f.ID, &f.Name, &f.Key, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *pgRepository) UpdateFlag(ctx context.Context, f *FeatureFlag) error {
	query := `
		UPDATE feature_flags
		SET key = $1, description = $2, enabled = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, f.Key, f.Description, f.Enabled, f.ID)
	return err
}

func (r *pgRepository) DeleteFlag(ctx context.Context, id int64) error {
	query := `DELETE FROM feature_flags WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, id)
	return err
}

func (r *pgRepository) ListFlags(ctx context.Context, limit int, cursor int64) ([]FeatureFlag, error) {
	query := `
		SELECT id, name, key, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE id > $2
		ORDER BY id ASC
		LIMIT $1`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		err := rows.Scan(&f.ID, &f.Name, &f.Key, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
