# Contributing — Adding a New Module

CoreKit is designed to be extended with new backend modules. Each module follows a strict pattern: `handler → service → repository → DB`.

## Quick start

```sh
# Scaffold module files
./scripts/scaffold-module.sh <name>

# Then wire it manually (see below)
```

---

## Module anatomy

Every module lives under `backend/internal/modules/<name>/` with 4 files:

```
internal/modules/<name>/
├── handler.go      # HTTP handlers + route registration
├── service.go      # Business logic
├── repository.go   # Database access
└── models.go       # Data types
```

### 1. Models (`models.go`)

```go
package example

import "time"

type Item struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 2. Repository (`repository.go`)

Must define an **interface** and a **PostgreSQL implementation**.

```go
type Repository interface {
    Create(ctx context.Context, item *Item) error
    GetByID(ctx context.Context, id int64) (*Item, error)
    List(ctx context.Context) ([]Item, error)
}
```

Use `database.GetQueryer(ctx, r.db)` for transaction-safe queries.
Use `database.RunInTransaction(ctx, r.db, func(txCtx) error { ... })` for multi-step ops.

### 3. Service (`service.go`)

Contains business logic. Receives `Repository` via constructor.

```go
type Service interface {
    Create(ctx context.Context, name string) (*Item, error)
}

type service struct {
    repo Repository
}
```

### 4. Handler (`handler.go`)

HTTP layer. Uses `validation.BindAndValidate` for request parsing and `httputil.*` for responses.

---

## Wiring steps

### Step A — Register in container.go

In `backend/internal/container/container.go`:

```go
type Container struct {
    ExampleSvc example.Service
    ExampleH   *example.Handler
    // ...
}

func NewContainer(cfg *config.Config) *Container {
    // ... existing setup ...

    exampleRepo := example.NewRepository(db)
    exampleSvc := example.NewService(exampleRepo)
    exampleH := example.NewHandler(exampleSvc)

    return &Container{
        // ...
        ExampleSvc: exampleSvc,
        ExampleH:   exampleH,
    }
}
```

### Step B — Register routes in main.go

In `backend/cmd/api/main.go`:

```go
c.ExampleH.RegisterRoutes(apiGroup, authMW)
```

### Step C — Add DB migration

Create `backend/migrations/000002_example.sql`:

```sql
CREATE TABLE IF NOT EXISTS examples (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Step D — Update check-imports allowlist

In `backend/cmd/check-imports/main.go`, add to `allowedPrefixes`:

```go
"corekit/backend/internal/modules/example",
```

---

## Rules

| Rule | Why |
|------|-----|
| **No cross-module imports** | Modules communicate via events (`pkg/event`) or shared interfaces |
| **Audit is automatic** | DB triggers capture every mutation via `set_config()` |
| **Keyset pagination** (not `OFFSET`) | For lists with >1000 rows |
| **Errors via `httputil.Error()`** | Consistent `{error: {code, message}}` format |
| **Response envelope** | Always `{data, meta, error}` |
| **Single role per user** | `role_id` on `users` table, no membership table |

---

## Testing

```sh
# Run all tests
make test

# Run single package
cd backend && go test -v ./internal/modules/example/...
```

Add tests under `backend/internal/modules/<name>/<name>_test.go`.
Use the test helpers in `backend/internal/e2e/helper.go` for integration tests.

---

## Architecture

```
Frontend (React/Vite :5173)
    │  HTTP/JSON
Backend (Go/Echo :8080)
    │
    ├── modules/         ← Your domain logic lives here
    │   ├── iam/         Identity & Access Management
    │   ├── rbac/        Roles & Permissions
    │   └── <your>       Your new module
    │
    ├── pkg/             Shared utilities
    ├── internal/        Config, middleware, DI
    └── migrations/      SQL migrations (append-only)
```

See `docs/SYSTEM_DOCUMENTATION.md` for the full system reference.
