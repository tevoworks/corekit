# CoreKit — Agent Guide

## Quick start
```sh
docker compose up -d db redis minio
cp backend/.env.example backend/.env   # fill secrets
cd backend && go run ./cmd/api/
```
Or use `make setup` to bootstrap everything.

## Key commands
`make dev` — Start backend + frontend (hot-reload), auto-starts docker infra
`make dev-be` — Start backend only
`make dev-fe` — Start frontend only (Vite :5173)
`make stop` — Kill backend, frontend, and docker compose down
`make restart` — Stop then start
`make build` — Build backend binary + frontend dist
`make docker-build` — Build Docker image
`make docker-up` / `make docker-down` — Infra (db, redis, minio)
`make lint` — check-imports → go vet
`make test` — Run all backend tests (`go test -v -count=1 ./...`)
`make scaffold name=X` — Generate a new backend module
`make setup` — Bootstrap project (docker + deps)
`make logs` — Follow docker logs
`make migration-up` / `make migration-down` / `make migration-reset` — Manage DB migrations
`make db-reset` — Drop and recreate database (dev only)
`make clean` — Remove build artifacts
`make-admin <email>` — Promote user to super_admin (`go run ./cmd/cli/ make-admin user@example.com`)
`create-admin <email> <password>` — Create new super admin (`go run ./cmd/cli/ create-admin user@example.com pass123`)

## Architecture
Modular monolith (Go/Echo :8080) + single SPA frontend (Vite/React 19, TailwindCSS, TanStack Query).
9 backend modules under `backend/internal/modules/`.
DI via `backend/internal/container/container.go`.
Single-tenant — no tenant_id anywhere.
DB port: 5434 (to avoid local PG conflict).

### Modules
| Module | Routes | Tables |
|--------|--------|--------|
| IAM | `/api/auth/*`, `/api/me`, `/api/users/*`, `/api/sessions/*`, `/api/notifications/*`, `/api/preferences/*`, `/api/identities/*` | `users`, `sessions`, `user_identities`, `user_verifications`, `notifications`, `user_notification_preferences`, `user_preferences` |
| RBAC | `/api/roles/*`, `/api/permissions/*`, `/api/rbac/check` | `roles`, `permissions`, `role_permissions` |
| Audit | `/api/audit-logs` | `audit_logs` |
| Settings | `/api/settings/*`, `/api/feature-flags/*` | `settings`, `feature_flags` |
| Storage | `/api/storage/upload`, `/api/storage/files/*`, `/api/public/storage/files/:id` | `file_metadata` + S3/MinIO |
| API Key | `/api/api-keys/*` | `api_keys` |
| Webhook | `/api/webhooks/*` | `webhooks`, `webhook_deliveries` |
| Queue | `/api/jobs/*` | `jobs`, `dead_letter_jobs` |
| Permission Registry | `/api/permissions/registry/*`, `/api/templates/*` | `permission_registry`, `global_templates` |

## Critical rules
- **No cross-module imports** — use interfaces and events (`pkg/event`). Enforced by `cmd/check-imports` via `make lint`.
- **Audit is automatic via DB triggers** — no Go-level audit method needed.
- **Keyset pagination only** — never use `OFFSET`.
- **Errors** — return standard error types; use httputil error helpers (BadRequest, Unauthorized, Forbidden, NotFound, InternalError).
- **Response envelope** — always `{data, meta, error}` via `pkg/httputil`.
- **Single role per user** — `role_id` on users table, no separate membership table.
- **CSRF** — Double Submit Cookie pattern via `CSRFMiddleware`. Set `CSRF_ENABLED=false` in env for same-origin deployments (frontend reverse-proxied under the same domain). Always `true` for cross-origin setups.
- **Migration order** — `000001_init.sql` is for fresh installs. `000002_hardening.sql` applies FK constraints, CHECK constraints, indexes, and audit trigger improvements on top. Always add new `.sql` files with higher prefix numbers — never edit existing migrations. There are currently 11 migration files (`000001`–`000011`).

## For AI agents
Read `docs/SYSTEM_DOCUMENTATION.md` (in project root) for the full system reference, including:
- Module responsibilities and owned tables
- API contract summary
- Security model
- Middleware stack


