# CoreKit

> **Single-tenant modular monolith boilerplate** — Go/Echo backend + React/Vite frontend.
> Clean foundation for building admin panels, internal tools, or ecommerce platforms.

## Setup from scratch

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) — for Postgres, Redis, MinIO
- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [openssl](https://www.openssl.org/) (for JWT secret generation)

### Step-by-step

```sh
# 1. Clone the repository
git clone https://github.com/tevoworks/corekit && cd corekit

# 2. Copy environment file and generate JWT secret
cp backend/.env.example backend/.env
# (optional) generate a strong JWT secret:
#   openssl rand -hex 64
# then replace the value in backend/.env

# 3. Start infrastructure (Postgres on :5434, Redis on :6380, MinIO on :9000)
docker compose up -d db redis minio

# 4. Run database migrations (creates tables, seed roles & permissions)
make migration-up

# 5. Create a super admin user
cd backend && go run ./cmd/cli/ create-admin admin@example.com Admin123!
# You should see: Super admin created: admin@example.com (id=...)

# 6. Start both backend (:8080) and frontend (:5173) with hot-reload
make dev
```

### Login

1. Open **http://localhost:5173** in your browser.
2. Enter the credentials you created above:
   - Email: `admin@example.com`
   - Password: `Admin123!`
3. You will be redirected to the dashboard.

> Or use `make setup` to automate steps 2–3 (copies `backend/.env`, generates JWT secret, starts Docker, and installs frontend npm deps).

---

## Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start backend + frontend (hot-reload) |
| `make dev-be` | Start backend only |
| `make dev-fe` | Start frontend only |
| `make stop` | Stop all services (backend, frontend, docker) |
| `make test` | Run all backend tests |
| `make lint` | Module boundary check + go vet |
| `make scaffold name=<module>` | Generate new backend module |
| `make build` | Build production assets |
| `make docker-build` | Build Docker image |
| `make docker-up` | Start infra containers (db, redis, minio) |
| `make docker-down` | Stop infra containers |
| `make setup` | Bootstrap project (env, docker, deps) |
| `make logs` | Follow docker logs |
| `make db-reset` | Drop and recreate database |
| `make migration-up` | Run all pending migrations |
| `make migration-down` | Roll back the last migration |
| `make migration-reset` | Roll back all, then re-apply |
| `make clean` | Remove build artifacts |

### Admin CLI (run from `backend/`)

```sh
cd backend

# Create a new super admin user
go run ./cmd/cli/ create-admin <email> <password>

# Promote an existing user to super_admin
go run ./cmd/cli/ make-admin <email>
```

---

## Architecture

```
Frontend (Vite/React :5173)
    │  HTTP/JSON
Backend (Go/Echo :8080)
    │
    ├── modules/         ← Domain logic (9 modules)
    │   ├── iam/         Auth, users, sessions, notifications
    │   ├── rbac/        Roles & permissions
    │   ├── audit/       Immutable audit log
    │   ├── settings/    Key-value store + feature flags
    │   ├── storage/     File upload/download (S3/MinIO)
    │   ├── apikey/      Programmatic API access
    │   ├── webhook/     Outgoing event-driven webhooks
    │   ├── queue/       Background job processor
    │   └── permregistry/ Permission catalog + templates
    │
    ├── pkg/             Shared utilities (httputil, errors, event, crypto)
    ├── internal/        Config, middleware, DI container
    └── migrations/      SQL migrations (append-only)

Infra (Docker)
    ├── PostgreSQL 15    Database (:5434)
    ├── Redis 7          Session cache + rate limiting (:6380)
    └── MinIO            S3-compatible storage (:9000 / :9001)
```

## Module creation

```sh
make scaffold name=products
```

This generates `backend/internal/modules/products/` with handler, service, repository, and models.
Then manually wire in `container.go` and register routes in `main.go` (see `CONTRIBUTING.md`).

---

## API

All responses follow the envelope format:

```json
{
  "data": { ... },
  "meta": { "count": 10 },
  "error": { "code": "BAD_REQUEST", "message": "..." }
}
```

| Module | Routes | Auth |
|--------|--------|------|
| Auth | `POST /api/auth/register`, `/login`, `/verify`, `/oauth/google` | Public (rate-limited) |
| Users | `GET/POST /api/users`, `PUT/DELETE /api/users/:id` | JWT + Super Admin |
| Profile | `GET /api/me`, `PATCH /api/profile`, `POST /api/logout` | JWT |
| Roles | `CRUD /api/roles`, `POST /api/roles/:id/permissions` | JWT |
| Permissions | `CRUD /api/permissions/registry`, `/api/permissions/sync` | JWT |
| Settings | `GET/POST /api/settings`, `CRUD /api/feature-flags` | JWT + RBAC |
| Audit | `GET /api/audit-logs` | JWT |
| API Keys | `POST /api/api-keys`, `GET /api/api-keys`, `DELETE /api/api-keys/:id` | JWT + RBAC |
| Webhooks | `CRUD /api/webhooks`, `GET /api/webhooks/:id/deliveries` | JWT + RBAC |
| Storage | `POST /api/storage/upload`, `GET/DELETE /api/storage/files/:id` | JWT + RBAC |
| Jobs | `GET /api/jobs`, `POST /api/jobs/:id/retry` | Super Admin |
| Notifications | `GET /api/notifications`, `PATCH /:id/read` | JWT |

See `docs/SYSTEM_DOCUMENTATION.md` for the full reference.

---

## Security

| Feature | Implementation |
|---------|---------------|
| Authentication | JWT (HMAC-SHA256) with session revocation |
| Authorization | RBAC per route (super_admin or explicit permission check) |
| CSRF | Double Submit Cookie pattern (`CSRF_ENABLED=true` for cross-origin, `false` for same-origin) |
| OAuth | Google ID token signature verified via JWKS (RFC 7517) |
| Audit Trail | DB triggers capture all mutations — `actor_id` + `impersonator_id` on every change |
| Rate Limiting | Token bucket per IP (login: 10/min, register: 3/min) |
| Input Validation | Strict: nohtml, urlstrict, emailfmt, password complexity |
| Database | Parameterized queries (no SQL injection), CHECK constraints on enum columns |
| File Upload | MIME whitelist, public downloads forced to `attachment` disposition |
| Password Storage | bcrypt cost 12 |
| Impersonation | Admin impersonation tracked via separate `impersonator_id` in audit trail |

## Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25 / Echo v4 |
| Frontend | React 19 / Vite 8 / TailwindCSS 3 |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Storage | MinIO / S3 |
| Auth | JWT + OAuth 2.0 (Google) |
| Background Jobs | Postgres-backed queue |

## License

MIT
