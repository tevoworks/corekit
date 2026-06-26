# Building a New Domain Module

This guide walks you through creating a new backend module that follows CoreKit conventions — 4-layer architecture, dependency injection, automatic audit, RBAC integration, keyset pagination, and strict module isolation.

---

## Quick Start (Scaffold)

```sh
make scaffold name=<module_name>
```

This generates 4 files at `backend/internal/modules/<module_name>/`:

| File | Purpose |
|------|---------|
| `models.go` | Data structs |
| `repository.go` | Interface + `pgRepository` (SQL queries) |
| `service.go` | Interface + service struct (business logic) |
| `handler.go` | HTTP handlers + route registration |

It also creates a migration fragment at `backend/internal/modules/migration_fragment.sql`.

After scaffolding, you still need to **manually wire** the module (see below).

---

## Layer-by-Layer Breakdown

### 1. `models.go` — Data Structs

```go
package <module>

import "time"

type Item struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

Rules:
- Every table gets a struct
- Use `int64` for PKs, `time.Time` for timestamps, `*type` for nullable columns
- JSON tags use `snake_case`
- Keep models here; request/response DTOs go in `handler.go`

### 2. `repository.go` — Data Access Layer

```go
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
```

Key patterns:
- **Interface first** — always define an interface so the service depends on abstraction
- **`database.GetQueryer(ctx, r.db)`** — returns either the active `*sql.Tx` or `*sql.DB`. This makes all queries transaction-aware without passing the tx explicitly. **Always use it.**
- **`sql.ErrNoRows` → return `nil, nil`** — service/handler checks for nil and returns `NotFound`. Never propagate raw `sql.ErrNoRows` up.
- **Keyset pagination** — never use `OFFSET`:
  ```sql
  SELECT ... FROM items
  WHERE id > $2
  ORDER BY id ASC
  LIMIT $1
  ```
- **`RETURNING` clause** — use `INSERT ... RETURNING id, created_at, updated_at` to populate auto-generated fields.

### 3. `service.go` — Business Logic

```go
type Service interface {
    Create(ctx context.Context, name string) (*Item, error)
    // ...
}

type service struct {
    db           *sql.DB
    repo         Repository
    auditService audit.Service   // optional
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
    return &service{db: db, repo: repo, auditService: auditService}
}
```

Key patterns:
- **Interface first** — same as repository
- **`db *sql.DB`** is stored in the struct to pass to `RunInTransaction`
- **Audit context** — wrap mutations in `database.WithAuditCtx` + `database.RunInTransaction`:
  ```go
  actx := database.WithAuditCtx(ctx, actorID, "CREATE_ITEM")
  err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
      return s.repo.Create(txCtx, item)
  })
  ```
  This propagates `actor_id` and `action` to the PostgreSQL session so the DB audit trigger captures every mutation automatically.
- **`database.ErrNotFound`** — return this from service when an expected entity is missing; it is a `var` in the `database` package.
- **No cross-module imports** — depend only on `database`, `audit` (interface), and `pkg/event` for async communication. Never import another domain module directly.

### 4. `handler.go` — HTTP Layer

```go
type Handler struct {
    svc Service
}

func NewHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMW echo.MiddlewareFunc) {
    g.GET("/items", h.List, authMW)
    g.POST("/items", h.Create, authMW)
    g.GET("/items/:id", h.GetByID, authMW)
    g.PUT("/items/:id", h.Update, authMW)
    g.DELETE("/items/:id", h.Delete, authMW)
}
```

Key patterns:
- **`validation.BindAndValidate(c, &req)`** — binds and validates request body using Echo's validator with custom rules (`nohtml`, `urlstrict`, `password`, etc.)
- **`middleware.GetUserID(c)`** — extracts authenticated user ID from the JWT context
- **Response envelope** — always use `httputil` helpers:
  - `httputil.OK(c, data)` — 200
  - `httputil.Created(c, data)` — 201
  - `httputil.Deleted(c)` — 200 delete confirmation
  - `httputil.Message(c, msg)` — 200 with message
  - `httputil.Paginated(c, list, nextCursor, limit)` — 200 with keyset pagination meta
  - `httputil.NotFound(c, msg)` — 404
  - `httputil.BadRequest(c, msg)` — 400
  - `httputil.InternalError(c)` — 500
- **RBAC middleware** — attach per-route via `middleware.RBACMiddleware(h.rbacService, "permission:action")`:
  ```go
  g.POST("/items", h.Create, authMW, middleware.RBACMiddleware(h.rbacService, "manage:items"))
  ```
- **Keyset pagination handler pattern**:
  ```go
  func (h *Handler) List(c echo.Context) error {
      p := httputil.ParseCursorPagination(c)
      items, err := h.svc.List(ctx, p.Limit, p.Cursor)
      // ...
      nextCursor := int64(0)
      if len(items) > 0 {
          nextCursor = items[len(items)-1].ID
      }
      return httputil.Paginated(c, items, nextCursor, p.Limit)
  }
  ```

---

## Wiring the Module

### Step 1: Container (`backend/internal/container/container.go`)

Add to the `Container` struct:

```go
type Container struct {
    // ... existing fields
    YourSvc yourmodule.Service
    YourH   *yourmodule.Handler
}
```

In `NewContainer`, instantiate in dependency order:

```go
yourRepo := yourmodule.NewRepository(db)
yourSvc := yourmodule.NewService(db, yourRepo, auditSvc)
yourH := yourmodule.NewHandler(yourSvc) // or NewHandler(yourSvc, rbacSvc)

// In the return statement:
return &Container{
    // ...
    YourSvc: yourSvc,
    YourH:   yourH,
}
```

### Step 2: Routes (`backend/cmd/api/main.go`)

Import your module, then register routes:

```go
cont.YourH.RegisterRoutes(apiGroup, authMW)
```

Choose the right route group:

| Group | Middleware | When to use |
|-------|-----------|-------------|
| `apiGroup` (`/api`) | Auth optional | Public endpoints or mixed |
| `authGroup` (`/api` with `authMW`) | JWT required | Authenticated endpoints |
| `adminMW` | Super admin only | Admin-only endpoints |

Add per-route RBAC inside your `RegisterRoutes` method (see handler section above).

### Step 3: Import Allowlist (`backend/cmd/check-imports/main.go`)

Add your module's base package to `allowedPrefixes`:

```go
var allowedPrefixes = []string{
    // ...
    "github.com/tevoworks/corekit/backend/internal/modules/yourmodule",
}
```

This allows other allowed modules to import your module's interface. Without this, `make lint` will fail.

### Step 4: Database Migration

Create a new migration file at `backend/migrations/` with a higher prefix number:

```sql
-- 000007_create_items.sql
CREATE TABLE IF NOT EXISTS items (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add a permission for RBAC
INSERT INTO permissions (name, key, module, description)
VALUES ('Manage Items', 'manage:items', 'items', 'Create, update, delete items')
ON CONFLICT (key) DO NOTHING;

INSERT INTO permissions (name, key, module, description)
VALUES ('Read Items', 'read:items', 'items', 'View items')
ON CONFLICT (key) DO NOTHING;
```

**Rules:**
- **Never edit existing migration files** — always add new `.sql` files with a higher prefix number
- Post-migration hardening (FKs, CHECK constraints, indexes) is optional but follows `000002_hardening.sql` style
- `ON CONFLICT DO NOTHING` for seed data to make migrations idempotent

Run migrations with: `make migration-up` (or restart `make dev` to auto-migrate).

---

## Cross-Module Communication

**Never import another domain module directly.** Instead:

| Pattern | How |
|---------|-----|
| **Interface injection** | Inject the required service interface into your handler constructor (e.g., `rbac.Service`) |
| **Event dispatching** | Use `pkg/event.EventDispatcher.Dispatch(ctx, tx, "event_type", payload)` inside a transaction to enqueue async jobs. This writes to the `jobs` table, and the queue worker processes it. |

The `queue` module is the only cross-cutting concern — it is exempt from import checking.

---

## RBAC Integration

1. Add permissions in your migration (see Step 4 above)
2. Seed via `permissions.yaml` (optional, used for runtime sync) or let the migration do it
3. In your handler, accept `rbac.Service` and use `middleware.RBACMiddleware(h.rbacService, "permission:key")` per route:

```go
func NewHandler(svc Service, rbacSvc rbac.Service) *Handler {
    return &Handler{svc: svc, rbacSvc: rbacSvc}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMW echo.MiddlewareFunc) {
    g.GET("/items", h.List, authMW, middleware.RBACMiddleware(h.rbacService, "read:items"))
    g.POST("/items", h.Create, authMW, middleware.RBACMiddleware(h.rbacService, "manage:items"))
}
```

Super admins automatically bypass all RBAC checks.

---

## Audit Trail

**Audit is fully automatic** — no Go-level audit calls needed. Just follow the pattern in the service layer:

```go
actx := database.WithAuditCtx(ctx, actorID, "YOUR_ACTION_NAME")
database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
    return s.repo.Create(txCtx, item)
})
```

This calls `set_config('app.actor_id', ...)` and `set_config('app.action', ...)` on the DB session, which the PostgreSQL audit trigger reads. Every INSERT/UPDATE/DELETE on tables with the trigger is automatically logged to `audit_logs`.

Sensitive fields (`password_hash`, `secret`, `key_hash`, etc.) are automatically stripped from audit records by the trigger function.

---

## Key Constraints & Conventions

| Rule | Detail |
|------|--------|
| **No cross-module imports** | Enforced by `cmd/check-imports`. Add your module to `allowedPrefixes` if other modules need to import its interface. |
| **Interface-first** | Repository and Service layers must define interfaces. |
| **Keyset pagination only** | `WHERE id > $cursor ORDER BY id ASC LIMIT $limit` — never use `OFFSET`. |
| **Response envelope** | Every response uses `{data, meta, error}` via `httputil`. |
| **Transaction-aware queries** | Always use `database.GetQueryer(ctx, db)` in repository methods — it transparently handles both `*sql.DB` and `*sql.Tx`. |
| **Errors** | Return `httputil.BadRequest`, `httputil.NotFound`, `httputil.InternalError` etc. from handlers. Never expose raw DB errors. |
| **Migration immutability** | Never edit existing migration files. Add new ones with higher prefixes. |
| **Single role per user** | If you need user-role relationships, use `role_id` on the `users` table (no separate membership table). |
| **No `tenant_id`** | This is single-tenant. Do not introduce tenant columns. |

---

## Verification Checklist

- [ ] `make lint` passes (import boundary check + `go vet`)
- [ ] `make test` passes (`go test -v -count=1 ./...`)
- [ ] `make dev` starts without error (auto-runs migrations)
- [ ] Endpoints return the correct response envelope (`{data, meta, error}`)
- [ ] Keyset pagination works (test with `?limit=X&cursor=Y`)
- [ ] RBAC permissions are seeded and enforced
- [ ] Audit logs are generated on mutations (check `audit_logs` table)
- [ ] The scaffold script is up-to-date if you changed conventions

Happy building!
