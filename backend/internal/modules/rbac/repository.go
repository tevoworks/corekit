package rbac

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	CreateRole(ctx context.Context, r *Role) error
	ListRoles(ctx context.Context, limit int, cursor int64) ([]Role, error)
	UpdateRole(ctx context.Context, r *Role) error
	DeleteRole(ctx context.Context, id int64) error

	CreatePermission(ctx context.Context, p *Permission) error
	ListPermissions(ctx context.Context, limit int, cursor int64) ([]Permission, error)
	UpdatePermission(ctx context.Context, p *Permission) error
	DeletePermission(ctx context.Context, id int64) error

	AssignPermissionToRole(ctx context.Context, roleID, permissionID int64) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID int64) error

	GetUserPermissions(ctx context.Context, userID int64) ([]string, error)
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (repo *pgRepository) CreateRole(ctx context.Context, r *Role) error {
	query := `
		INSERT INTO roles (name, description)
		VALUES ($1, $2)
		RETURNING id`
	q := database.GetQueryer(ctx, repo.db)
	return q.QueryRowContext(ctx, query, r.Name, r.Description).Scan(&r.ID)
}

func (repo *pgRepository) ListRoles(ctx context.Context, limit int, cursor int64) ([]Role, error) {
	query := `
		SELECT r.id, r.name, r.description,
		       COALESCE((SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = r.id), 0),
		       COALESCE((SELECT array_agg(p.name ORDER BY p.name) FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id = r.id), ARRAY[]::TEXT[])
		FROM roles r
		WHERE r.deleted_at IS NULL AND r.id > $2
		ORDER BY r.id ASC
		LIMIT $1`
	q := database.GetQueryer(ctx, repo.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.PermissionCount, pq.Array(&r.Permissions)); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (repo *pgRepository) CreatePermission(ctx context.Context, p *Permission) error {
	query := `
		INSERT INTO permissions (name, description)
		VALUES ($1, $2)
		RETURNING id`
	q := database.GetQueryer(ctx, repo.db)
	return q.QueryRowContext(ctx, query, p.Name, p.Description).Scan(&p.ID)
}

func (repo *pgRepository) ListPermissions(ctx context.Context, limit int, cursor int64) ([]Permission, error) {
	query := `
		SELECT id, name, description
		FROM permissions
		WHERE id > $2
		ORDER BY id ASC
		LIMIT $1`
	q := database.GetQueryer(ctx, repo.db)
	rows, err := q.QueryContext(ctx, query, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (repo *pgRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID int64) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING`
	q := database.GetQueryer(ctx, repo.db)
	_, err := q.ExecContext(ctx, query, roleID, permissionID)
	return err
}

func (repo *pgRepository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID int64) error {
	q := database.GetQueryer(ctx, repo.db)
	_, err := q.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`, roleID, permissionID)
	return err
}

func (repo *pgRepository) UpdateRole(ctx context.Context, r *Role) error {
	query := `UPDATE roles SET name = $1, description = $2 WHERE id = $3`
	q := database.GetQueryer(ctx, repo.db)
	res, err := q.ExecContext(ctx, query, r.Name, r.Description, r.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("role not found")
	}
	return nil
}

func (repo *pgRepository) DeleteRole(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE roles SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `UPDATE users SET role_id = NULL WHERE role_id = $1`, id)
	return err
}

func (repo *pgRepository) UpdatePermission(ctx context.Context, p *Permission) error {
	query := `UPDATE permissions SET name = $1, description = $2 WHERE id = $3`
	q := database.GetQueryer(ctx, repo.db)
	res, err := q.ExecContext(ctx, query, p.Name, p.Description, p.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("permission not found")
	}
	return nil
}

func (repo *pgRepository) DeletePermission(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, repo.db)
	_, err := q.ExecContext(ctx, `DELETE FROM permissions WHERE id = $1`, id)
	return err
}

func (repo *pgRepository) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	query := `
		SELECT p.name
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = (SELECT role_id FROM users WHERE id = $1)`

	q := database.GetQueryer(ctx, repo.db)
	rows, err := q.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var permissions []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}
