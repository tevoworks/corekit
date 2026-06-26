package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type Service interface {
	CreateRole(ctx context.Context, name, description string, actorID int64) (*Role, error)
	ListRoles(ctx context.Context, limit int, cursor int64) ([]Role, error)

	CreatePermission(ctx context.Context, name, description string, actorID int64) (*Permission, error)
	ListPermissions(ctx context.Context, limit int, cursor int64) ([]Permission, error)

	AssignPermission(ctx context.Context, roleID, permissionID int64, actorID int64) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID int64, actorID int64) error

	UpdateRole(ctx context.Context, id int64, name, description string, actorID int64) (*Role, error)
	DeleteRole(ctx context.Context, id int64, actorID int64) error
	UpdatePermission(ctx context.Context, id int64, name, description string, actorID int64) (*Permission, error)
	DeletePermission(ctx context.Context, id int64, actorID int64) error

	CheckAccess(ctx context.Context, userID int64, permissionName string) (bool, error)
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{
		db:           db,
		repo:         repo,
		auditService: auditService,
	}
}

func (s *service) CreateRole(ctx context.Context, name, description string, actorID int64) (*Role, error) {
	r := &Role{
		Name:        name,
		Description: description,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_ROLE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateRole(txCtx, r)
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *service) ListRoles(ctx context.Context, limit int, cursor int64) ([]Role, error) {
	return s.repo.ListRoles(ctx, limit, cursor)
}

func (s *service) CreatePermission(ctx context.Context, name, description string, actorID int64) (*Permission, error) {
	p := &Permission{
		Name:        name,
		Description: description,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_PERMISSION")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreatePermission(txCtx, p)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) ListPermissions(ctx context.Context, limit int, cursor int64) ([]Permission, error) {
	return s.repo.ListPermissions(ctx, limit, cursor)
}

func (s *service) UpdateRole(ctx context.Context, id int64, name, description string, actorID int64) (*Role, error) {
	r := &Role{
		ID:          id,
		Name:        name,
		Description: description,
	}

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_ROLE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdateRole(txCtx, r)
	})
	return r, err
}

func (s *service) DeleteRole(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_ROLE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		q := database.GetQueryer(txCtx, s.db)
		var userCount int
		if err := q.QueryRowContext(txCtx, `SELECT COUNT(*) FROM users WHERE role_id = $1 AND deleted_at IS NULL`, id).Scan(&userCount); err != nil {
			return err
		}
		if userCount > 0 {
			slog.Warn("deleting role with active users", "role_id", id, "user_count", userCount)
		}
		return s.repo.DeleteRole(txCtx, id)
	})
}

func (s *service) UpdatePermission(ctx context.Context, id int64, name, description string, actorID int64) (*Permission, error) {
	p := &Permission{
		ID:          id,
		Name:        name,
		Description: description,
	}

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_PERMISSION")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UpdatePermission(txCtx, p)
	})
	return p, err
}

func (s *service) DeletePermission(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_PERMISSION")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeletePermission(txCtx, id)
	})
}

func (s *service) AssignPermission(ctx context.Context, roleID, permissionID int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "ASSIGN_ROLE_PERMISSION")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		q := database.GetQueryer(txCtx, s.db)

		var exists int
		if err := q.QueryRowContext(txCtx, `SELECT 1 FROM roles WHERE id = $1 FOR UPDATE`, roleID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("role not found")
			}
			return err
		}

		if err := q.QueryRowContext(txCtx, `SELECT 1 FROM permissions WHERE id = $1 FOR UPDATE`, permissionID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("permission not found")
			}
			return err
		}

		return s.repo.AssignPermissionToRole(txCtx, roleID, permissionID)
	})
}

func (s *service) RemovePermissionFromRole(ctx context.Context, roleID, permissionID int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "REMOVE_ROLE_PERMISSION")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.RemovePermissionFromRole(txCtx, roleID, permissionID)
	})
}

func (s *service) CheckAccess(ctx context.Context, userID int64, permissionName string) (bool, error) {
	var isSuper bool
	err := s.db.QueryRowContext(ctx, `SELECT is_super_admin FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&isSuper)
	if err != nil {
		return false, err
	}
	if isSuper {
		return true, nil
	}

	perms, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		if p == permissionName {
			return true, nil
		}
	}

	return false, nil
}
