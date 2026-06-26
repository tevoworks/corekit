package permregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/tevoworks/corekit/backend/internal/database"
)

// Repository defines data access for the permission registry and global templates.
type Repository interface {
	// Registry entries
	ListRegistry(ctx context.Context) ([]RegistryEntry, error)
	ListRegistryByDomain(ctx context.Context) ([]ByDomain, error)
	GetRegistryEntry(ctx context.Context, id int64) (*RegistryEntry, error)
	CreateRegistryEntry(ctx context.Context, e *RegistryEntry) error
	UpdateRegistryEntry(ctx context.Context, e *RegistryEntry) error
	UpsertRegistryEntry(ctx context.Context, name, description, domain string) error
	DeleteRegistryEntry(ctx context.Context, id int64) error

	// Global templates
	ListGlobalTemplates(ctx context.Context) ([]GlobalTemplate, error)
	GetGlobalTemplate(ctx context.Context, id int64) (*GlobalTemplate, error)
	CreateGlobalTemplate(ctx context.Context, t *GlobalTemplate) error
	UpdateGlobalTemplate(ctx context.Context, t *GlobalTemplate) error
	DeleteGlobalTemplate(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

// NewRepository creates a new postgres-backed Repository.
func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

// ── Registry Entries ──────────────────────────────────────────────────────────

func (r *pgRepository) ListRegistry(ctx context.Context) ([]RegistryEntry, error) {
	query := `
		SELECT id, name, description, domain, is_active, created_at, updated_at
		FROM permission_registry
		ORDER BY domain ASC, name ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var list []RegistryEntry
	for rows.Next() {
		var e RegistryEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Domain, &e.IsActive, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) ListRegistryByDomain(ctx context.Context) ([]ByDomain, error) {
	all, err := r.ListRegistry(ctx)
	if err != nil {
		return nil, err
	}
	// Group by domain
	order := []string{}
	grouped := map[string][]RegistryEntry{}
	for _, e := range all {
		if _, ok := grouped[e.Domain]; !ok {
			order = append(order, e.Domain)
		}
		grouped[e.Domain] = append(grouped[e.Domain], e)
	}
	out := make([]ByDomain, 0, len(order))
	for _, d := range order {
		out = append(out, ByDomain{Domain: d, Permissions: grouped[d]})
	}
	return out, nil
}

func (r *pgRepository) GetRegistryEntry(ctx context.Context, id int64) (*RegistryEntry, error) {
	query := `SELECT id, name, description, domain, is_active, created_at, updated_at FROM permission_registry WHERE id=$1`
	q := database.GetQueryer(ctx, r.db)
	var e RegistryEntry
	err := q.QueryRowContext(ctx, query, id).Scan(&e.ID, &e.Name, &e.Description, &e.Domain, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *pgRepository) CreateRegistryEntry(ctx context.Context, e *RegistryEntry) error {
	query := `
		INSERT INTO permission_registry (name, description, domain, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, e.Name, e.Description, e.Domain, e.IsActive).
		Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (r *pgRepository) UpdateRegistryEntry(ctx context.Context, e *RegistryEntry) error {
	query := `
		UPDATE permission_registry
		SET description=$1, domain=$2, is_active=$3, updated_at=NOW()
		WHERE id=$4`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, e.Description, e.Domain, e.IsActive, e.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("registry entry not found")
	}
	return nil
}

func (r *pgRepository) UpsertRegistryEntry(ctx context.Context, name, description, domain string) error {
	query := `
		INSERT INTO permission_registry (name, description, domain, is_active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (name) DO UPDATE
		SET description = EXCLUDED.description,
		    domain      = EXCLUDED.domain,
		    updated_at  = NOW()`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, name, description, domain)
	return err
}

func (r *pgRepository) DeleteRegistryEntry(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM permission_registry WHERE id=$1`, id)
	return err
}

// ── Global Templates ──────────────────────────────────────────────────────────

func (r *pgRepository) ListGlobalTemplates(ctx context.Context) ([]GlobalTemplate, error) {
	query := `
		SELECT id, name, description, permissions, category, is_active, created_at, updated_at
		FROM global_templates
		ORDER BY name ASC`
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var list []GlobalTemplate
	for rows.Next() {
		var t GlobalTemplate
		var permsJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &permsJSON, &t.Category, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
			t.Permissions = []string{}
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pgRepository) GetGlobalTemplate(ctx context.Context, id int64) (*GlobalTemplate, error) {
	query := `SELECT id, name, description, permissions, category, is_active, created_at, updated_at FROM global_templates WHERE id=$1`
	q := database.GetQueryer(ctx, r.db)
	var t GlobalTemplate
	var permsJSON []byte
	err := q.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.Name, &t.Description, &permsJSON, &t.Category, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
		t.Permissions = []string{}
	}
	return &t, nil
}

func (r *pgRepository) CreateGlobalTemplate(ctx context.Context, t *GlobalTemplate) error {
	permsJSON, err := json.Marshal(t.Permissions)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO global_templates (name, description, permissions, category, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query, t.Name, t.Description, permsJSON, t.Category, t.IsActive).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *pgRepository) UpdateGlobalTemplate(ctx context.Context, t *GlobalTemplate) error {
	permsJSON, err := json.Marshal(t.Permissions)
	if err != nil {
		return err
	}
	query := `
		UPDATE global_templates
		SET name=$1, description=$2, permissions=$3, category=$4, is_active=$5, updated_at=NOW()
		WHERE id=$6`
	q := database.GetQueryer(ctx, r.db)
	res, err := q.ExecContext(ctx, query, t.Name, t.Description, permsJSON, t.Category, t.IsActive, t.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("global template not found")
	}
	return nil
}

func (r *pgRepository) DeleteGlobalTemplate(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM global_templates WHERE id=$1`, id)
	return err
}
