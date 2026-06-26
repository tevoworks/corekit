#!/usr/bin/env bash
set -euo pipefail

NAME="${1:-}"
if [ -z "$NAME" ]; then
  echo "Usage: $0 <module-name>"
  echo ""
  echo "Creates a new backend module scaffold with:"
  echo "  backend/internal/modules/<name>/"
  echo "    handler.go"
  echo "    service.go"
  echo "    repository.go"
  echo "    models.go"
  echo ""
  echo "Then prints wiring instructions for container.go, main.go, and check-imports."
  exit 1
fi

# Validate module name: only lowercase alphanum + underscore, start with letter
if ! [[ "$NAME" =~ ^[a-z][a-z0-9_]*$ ]]; then
  echo "Error: Module name must start with a letter and contain only lowercase letters, digits, and underscores."
  exit 1
fi

MODULE_DIR="backend/internal/modules/$NAME"

if [ -d "$MODULE_DIR" ]; then
  echo "Error: Module '$NAME' already exists at $MODULE_DIR"
  exit 1
fi

mkdir -p "$MODULE_DIR"

# ── Models ──────────────────────────────────────────────
cat > "$MODULE_DIR/models.go" << EOF
package $NAME

import "time"

type Item struct {
	ID        int64     \`json:"id"\`
	Name      string    \`json:"name"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
}
EOF

# ── Repository ──────────────────────────────────────────
cat > "$MODULE_DIR/repository.go" << EOF
package $NAME

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, item *Item) error
	GetByID(ctx context.Context, id int64) (*Item, error)
	List(ctx context.Context, limit int, cursor int64) ([]Item, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, item *Item) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		\`INSERT INTO ${NAME}s (name) VALUES (\$1) RETURNING id, created_at, updated_at\`,
		item.Name,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (r *pgRepository) GetByID(ctx context.Context, id int64) (*Item, error) {
	q := database.GetQueryer(ctx, r.db)
	var item Item
	err := q.QueryRowContext(ctx,
		\`SELECT id, name, created_at, updated_at FROM ${NAME}s WHERE id = \$1\`, id,
	).Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *pgRepository) List(ctx context.Context, limit int, cursor int64) ([]Item, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		\`SELECT id, name, created_at, updated_at FROM ${NAME}s WHERE id > \$2 ORDER BY id ASC LIMIT \$1\`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *pgRepository) Update(ctx context.Context, item *Item) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		\`UPDATE ${NAME}s SET name = \$1, updated_at = CURRENT_TIMESTAMP WHERE id = \$2\`,
		item.Name, item.ID,
	)
	return err
}

func (r *pgRepository) Delete(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, \`DELETE FROM ${NAME}s WHERE id = \$1\`, id)
	return err
}
EOF

# ── Service ─────────────────────────────────────────────
cat > "$MODULE_DIR/service.go" << EOF
package $NAME

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type Service interface {
	Create(ctx context.Context, name string, actorID int64) (*Item, error)
	GetByID(ctx context.Context, id int64) (*Item, error)
	List(ctx context.Context, limit int, cursor int64) ([]Item, error)
	Update(ctx context.Context, id int64, name string, actorID int64) (*Item, error)
	Delete(ctx context.Context, id int64, actorID int64) error
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{db: db, repo: repo, auditService: auditService}
}

func (s *service) Create(ctx context.Context, name string, actorID int64) (*Item, error) {
	item := &Item{Name: name}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_${NAME^^}")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Create(txCtx, item)
	})
	if err != nil {
		return nil, fmt.Errorf("create $NAME: %w", err)
	}
	return item, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, limit int, cursor int64) ([]Item, error) {
	return s.repo.List(ctx, limit, cursor)
}

func (s *service) Update(ctx context.Context, id int64, name string, actorID int64) (*Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, database.ErrNotFound
	}
	item.Name = name

	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_${NAME^^}")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Update(txCtx, item)
	})
	if err != nil {
		return nil, fmt.Errorf("update $NAME: %w", err)
	}
	return item, nil
}

func (s *service) Delete(ctx context.Context, id int64, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_${NAME^^}")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, id)
	})
}
EOF

# ── Handler ─────────────────────────────────────────────
cat > "$MODULE_DIR/handler.go" << EOF
package $NAME

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/rbac"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	svc         Service
	rbacService rbac.Service
}

func NewHandler(svc Service, rbacService rbac.Service) *Handler {
	return &Handler{svc: svc, rbacService: rbacService}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMW echo.MiddlewareFunc) {
	g.GET("/${NAME}s", h.List, authMW, middleware.RBACMiddleware(h.rbacService, "read:${NAME}s"))
	g.POST("/${NAME}s", h.Create, authMW, middleware.RBACMiddleware(h.rbacService, "manage:${NAME}s"))
	g.GET("/${NAME}s/:id", h.GetByID, authMW, middleware.RBACMiddleware(h.rbacService, "read:${NAME}s"))
	g.PUT("/${NAME}s/:id", h.Update, authMW, middleware.RBACMiddleware(h.rbacService, "manage:${NAME}s"))
	g.DELETE("/${NAME}s/:id", h.Delete, authMW, middleware.RBACMiddleware(h.rbacService, "manage:${NAME}s"))
}

type CreateRequest struct {
	Name string \`json:"name" validate:"required,nohtml"\`
}

type UpdateRequest struct {
	Name string \`json:"name" validate:"required,nohtml"\`
}

func (h *Handler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	item, err := h.svc.Create(ctx, req.Name, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, item)
}

func (h *Handler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	item, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if item == nil {
		return httputil.NotFound(c, "${NAME^} not found")
	}
	return httputil.OK(c, item)
}

func (h *Handler) List(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)

	items, err := h.svc.List(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}
	return httputil.Paginated(c, items, nextCursor, p.Limit)
}

func (h *Handler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	var req UpdateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	item, err := h.svc.Update(ctx, id, req.Name, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, item)
}

func (h *Handler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	if err := h.svc.Delete(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}
EOF

# ── Migration fragment ─────────────────────────────────
cat > "$MODULE_DIR/../migration_fragment.sql" << EOF
-- ── ${NAME^}s ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ${NAME}s (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed RBAC permissions (replace keys with your actual domain)
INSERT INTO permissions (name, key, module, description)
VALUES ('Manage ${NAME^}s', 'manage:${NAME}s', '${NAME}', 'Create, update, delete ${NAME}s')
ON CONFLICT (key) DO NOTHING;

INSERT INTO permissions (name, key, module, description)
VALUES ('Read ${NAME^}s', 'read:${NAME}s', '${NAME}', 'View ${NAME}s')
ON CONFLICT (key) DO NOTHING;
EOF

echo ""
echo "✅ Module '$NAME' created at $MODULE_DIR"
echo ""
echo "Next steps:"
echo "  1. Wire in container.go:"
echo "       ${NAME}Repo := ${NAME}.NewRepository(db)"
echo "       ${NAME}Svc := ${NAME}.NewService(db, ${NAME}Repo, auditSvc)"
echo "       ${NAME}H := ${NAME}.NewHandler(${NAME}Svc, rbacSvc)"
echo ""
echo "  2. Register routes in main.go:"
echo "       cont.${NAME^}H.RegisterRoutes(authGroup, authMW)"
echo ""
echo "  3. Add to check-imports allowlist (if other modules need to import):"
echo "       \"github.com/tevoworks/corekit/backend/internal/modules/${NAME}\""
echo ""
echo "  4. Create a new migration file from:"
echo "       backend/internal/modules/migration_fragment.sql"
echo ""
echo "  5. Run 'make db-reset && make dev' to test"
echo ""
