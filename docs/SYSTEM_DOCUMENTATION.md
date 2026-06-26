# Singleapp System Documentation

> **Canonical system reference** — provides complete context for any AI agent or developer to understand, reason about, and extend this codebase without reading source code.

---

## 1. System Overview

### System Purpose
Singleapp is a **Domain-Agnostic Application Boilerplate** that provides battle-ready core infrastructure (identity, RBAC, audit, settings, storage, background jobs, webhooks) for building single-tenant admin panels, ecommerce platforms, SaaS backends, or internal tools.

Singleapp has **no tenant concept**. Every user has a single role, every setting is global, every file belongs to the system.

### Architectural Style
- **Modular Monolith** in Go 1.25 (Echo v4 web framework)
- **Single SPA frontend** (Vite + React 19 + TailwindCSS)
- **9 business domain modules** organized by domain (not technical layer)
- **PostgreSQL 15** primary store, **S3/MinIO** for files, **Redis 7** optional (rate limiting + session revocation)

### Trust Boundaries
1. **Public:** `/api/auth/*` (register, login, verify, OAuth), `/api/public/storage/files/:id`
2. **Authenticated:** `/api/*` (all other routes require valid JWT)
3. **Admin:** Routes requiring specific RBAC permissions (e.g., `manage:users`)

---

## 2. Module Responsibility Map

```
IAM ──> Audit (logs mutations)
IAM ──> Queue (enqueues emails)
RBAC ──> Audit
Settings ──> RBAC (checks permissions)
Settings ──> Audit
Storage ──> RBAC
Storage ──> Audit
API Key ──> Audit
Webhook ──> Audit
Webhook ──> Queue (dispatches via queue)
PermRegistry ──> Audit
PermRegistry ──> Sync from permissions.yaml
```

### Backend Modules

#### IAM (Identity & Access Management)
- **Role:** User registration, login, session management, profile management, email verification, Google OAuth, forgot/reset password, token refresh, notifications, preferences, account deletion, data export
- **Owns:** `users`, `sessions`, `user_identities`, `user_verifications`, `notifications`, `user_notification_preferences`, `user_preferences`
- **Key detail:** Each user has exactly ONE role (`role_id` FK on `users` table). No multi-tenant memberships.
- **Depends on:** `Database`, `Audit`, `RBAC` (for permission checks), `Queue` (email sending), `Redis` (session revocation)

#### RBAC (Role-Based Access Control)
- **Role:** Defines roles, permissions, role-permission mapping. Validates access via `CheckAccess()`.
- **Owns:** `roles`, `permissions`, `role_permissions`
- **Key detail:** Global roles (no tenant isolation). Wildcard permission `*` grants all access.
- **Depends on:** `Database`, `Audit`

#### Audit
- **Role:** Immutable ledger of all state-mutating events. Trigger-based logging via PostgreSQL.
- **Owns:** `audit_logs`
- **Depends on:** `Database`

#### Settings
- **Role:** Global key-value settings store + feature flags toggle.
- **Owns:** `settings`, `feature_flags`
- **Key detail:** Everything is global (no tenant override concept).
- **Depends on:** `Database`, `Audit`, `RBAC`

#### Storage
- **Role:** File upload/download/delete to S3/MinIO. Public file support.
- **Owns:** `file_metadata` (metadata in DB, binary in S3)
- **Depends on:** `Database`, `Audit`, `RBAC`, `Settings` (max upload size)

#### API Key
- **Role:** Manage API keys for programmatic access. SHA-256 hashed + salted.
- **Owns:** `api_keys`
- **Depends on:** `Database`, `Audit`, `RBAC`

#### Queue (Background Job System)
- **Role:** Postgres-backed background job processor. Idempotency keys, retry (max 3), heartbeat claiming, DLQ.
- **Owns:** `jobs`, `dead_letter_jobs`
- **Depends on:** `Database`

#### Webhook
- **Role:** Outgoing webhook management. HMAC-SHA256 signing, delivery tracking, retry.
- **Owns:** `webhooks`, `webhook_deliveries`
- **Depends on:** `Database`, `Queue`, `Audit`, `RBAC`

#### Permission Registry
- **Role:** System-wide permission catalog. Syncs from `permissions.yaml` at startup. Global templates.
- **Owns:** `permission_registry`, `global_templates`
- **Depends on:** `Database`

### Shared Packages

| Package | File | Purpose |
|---------|------|---------|
| `pkg/event` | `dispatcher.go` | Wraps queue repository for event-driven module communication |
| `pkg/httputil` | `response.go` | Response envelope (OK, Created, Paginated, Error), pagination parser |

### Infrastructure Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `config/` | `config.go` | Env-based config via `config.Load()`. Fields: PORT, DATABASE_URL, JWT_SECRET, ALLOWED_ORIGINS, REDIS_URL, S3 config, Google OAuth, SMTP, APP_ENV, TLS |
| `database/` | `database.go, migrate.go` | PostgreSQL connection, transaction helpers (`RunInTransaction`, `WithTx`), audit context propagation via `set_config()`, SQL migration runner |
| `redisstore/` | `store.go` | Redis-backed session revocation. Mode A (Redis ON) / Mode B (fallback to DB query) |
| `validation/` | `validator.go` | Custom Echo validator with rules: nohtml, urlstrict, emailfmt, password (min 8, upper+lower+digit+special) |
| `authverify/` | `service.go, handler.go, cache.go` | JWT introspection service. 10-step pipeline: parse → session validation → cache → user load → response. Rate-limited. |

---

## 3. Business Domain Model

### User
- **Lifecycle:** `Registered` → `Pending Verification` → `Active` → `Super Admin Promoted` → `Deleted` (soft-delete)
- **Role:** Single role FK (`role_id` → `roles.id`). Default role on registration: `viewer`.
- **Constraints:** Email unique globally (partial unique index `WHERE deleted_at IS NULL`). bcrypt passwords. Email verification required.

### Role
- **Lifecycle:** `Created` → `Permissions Assigned` → `Deleted`
- **Default roles (seeded):** `super_admin` (wildcard `*`), `admin`, `manager`, `viewer`

### Permission
- **Lifecycle:** `Created` → `Linked` (via `role_permissions`) → `Deleted`
- **Format:** `<domain>:<verb>` (e.g., `read:users`, `manage:settings`)
- **Wildcard:** Permission named `*` grants all capabilities.

### Session
- **Lifecycle:** `Active` (created on login) → `Revoked` (on logout) → `Expired`
- **Token:** JWT with claims: `user_id`, `role`, `is_super_admin`, `token_id`, `impersonator_id`

### File
- **Lifecycle:** `Created` (metadata + S3 upload) → `Deleted`
- **Access:** Public if `is_public = true`, otherwise requires `read:files` permission.

### API Key
- **Lifecycle:** `Created` (SHA-256 + salt stored, raw key returned once) → `Revoked`
- **Prefix:** `ks_` prefixed for identification.

### Webhook
- **Lifecycle:** `Active` → `Inactive` → `Deleted`
- **Security:** HMAC-SHA256 secret signing. Delivery via background queue.

### Job & Dead Letter Job
- **Lifecycle:** `Pending` → `Processing` → `Completed` / `Failed` (max 3 retries) → `Dead Letter`
- **Idempotency:** Unique `idempotency_key` prevents duplicate dispatch.

### Audit Log
- **Lifecycle:** Written automatically via DB trigger on INSERT/UPDATE/DELETE. Immutable.
- **Fields:** `id`, `actor_id`, `impersonator_id`, `action`, `target_entity`, `before_state` (JSONB), `after_state` (JSONB), `created_at`

---

## 4. API Contract

### Response Envelope
```json
{
  "data": {},
  "meta": {},
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  }
}
```

### Keyset Pagination
- Initial: `offset=0` (functions as cursor)
- Subsequent: `offset=<id_of_last_record>`
- Query: `WHERE id > offset ORDER BY id ASC LIMIT limit`

### Public Endpoints

| Method | Path | Description | Rate Limit |
|--------|------|-------------|-----------|
| POST | `/api/auth/register` | Register user | 3/min/IP |
| POST | `/api/auth/login` | Login, returns JWT | 10/min/IP |
| GET | `/api/auth/verify` | Verify email | 10/min/IP |
| POST | `/api/auth/verify` | Verify email (alt) | 10/min/IP |
| POST | `/api/auth/verify-resend` | Resend verification | 3/min/IP |
| GET | `/api/auth/oauth/google` | Initiate Google OAuth | 10/min/IP |
| GET | `/api/auth/oauth/google/callback` | OAuth callback | 10/min/IP |
| POST | `/api/auth/exchange-code` | Exchange OAuth code for JWT | 5/min/IP |
| POST | `/api/auth/refresh` | Refresh JWT token | 5/min/IP |
| POST | `/api/auth/forgot-password` | Request password reset | 5/min/IP, 3/min/Email |
| POST | `/api/auth/reset-password` | Reset password with token | 5/min/IP, 3/min/Email |
| GET | `/api/public/storage/files/:id` | Download public file | — |

### Authenticated Endpoints (`/api/*`)

**User Profile & Sessions**
| Method | Path | Permission |
|--------|------|-----------|
| GET | `/me` | — |
| POST | `/logout` | — |
| POST | `/logout-all` | — |
| PATCH | `/profile` | — |
| GET | `/sessions` | — |
| DELETE | `/sessions/:token_id` | — |
| GET | `/preferences` | — |
| PUT | `/preferences/:key` | — |
| GET | `/identities` | — |
| DELETE | `/identities/:id` | — |
| POST | `/export-data` | — |
| DELETE | `/account` | — |

**Notifications**
| Method | Path | Permission |
|--------|------|-----------|
| GET | `/notifications/preferences` | — |
| PUT | `/notifications/preferences/:type` | — |
| GET | `/notifications` | — |
| GET | `/notifications/unread-count` | — |
| PATCH | `/notifications/:id/read` | — |
| POST | `/notifications/read-all` | — |
| DELETE | `/notifications/:id` | — |

**User Management (requires `manage:users`)**
| Method | Path | Permission |
|--------|------|-----------|
| GET | `/users` | `read:users` |
| POST | `/users` | `manage:users` |
| PUT | `/users/:id` | `manage:users` |
| PATCH | `/users/:id/status` | `manage:users` |
| DELETE | `/users/:id` | `manage:users` |
| POST | `/users/:id/force-reset` | `manage:users` |
| PUT | `/users/:id/role` | `manage:users` |
| POST | `/impersonate` | `manage:users` |
| POST | `/users/promote` | `manage:users` |
| GET | `/sessions/all` | `read:sessions` |
| DELETE | `/sessions/all/:token_id` | `manage:users` |
| GET | `/users/:id/sessions` | `read:sessions` |

**RBAC**
| Method | Path | Permission |
|--------|------|-----------|
| POST | `/roles` | `manage:roles` |
| GET | `/roles` | `read:roles` |
| PUT | `/roles/:id` | `manage:roles` |
| DELETE | `/roles/:id` | `manage:roles` |
| POST | `/permissions` | `manage:permissions` |
| GET | `/permissions` | `read:permissions` |
| PUT | `/permissions/:id` | `manage:permissions` |
| DELETE | `/permissions/:id` | `manage:permissions` |
| POST | `/roles/:id/permissions` | `manage:role_permissions` |
| DELETE | `/roles/:role_id/permissions/:permission_id` | `manage:role_permissions` |
| POST | `/rbac/check` | — |

**Settings & Feature Flags**
| Method | Path | Permission |
|--------|------|-----------|
| POST | `/settings` | `manage:settings` |
| GET | `/settings` | `read:settings` |
| GET | `/settings/:key` | `read:settings` |
| DELETE | `/settings/:key` | `manage:settings` |
| POST | `/feature-flags` | `manage:feature_flags` |
| GET | `/feature-flags` | `read:feature_flags` |
| GET | `/feature-flags/:key` | `read:feature_flags` |
| PUT | `/feature-flags/:id` | `manage:feature_flags` |
| DELETE | `/feature-flags/:id` | `manage:feature_flags` |

**Storage**
| Method | Path | Permission |
|--------|------|-----------|
| POST | `/storage/upload` | `write:files` |
| GET | `/storage/files` | `read:files` |
| GET | `/storage/files/:id` | `read:files` |
| DELETE | `/storage/files/:id` | `delete:files` |

**API Keys**
| Method | Path | Permission |
|--------|------|-----------|
| POST | `/api-keys` | `manage:api_keys` |
| GET | `/api-keys` | `read:api_keys` |
| DELETE | `/api-keys/:id` | `manage:api_keys` |
| POST | `/api-keys/:id/rotate` | `manage:api_keys` |

**Webhooks**
| Method | Path | Permission |
|--------|------|-----------|
| POST | `/webhooks` | `manage:webhooks` |
| GET | `/webhooks` | `read:webhooks` |
| GET | `/webhooks/:id` | `read:webhooks` |
| PUT | `/webhooks/:id` | `manage:webhooks` |
| DELETE | `/webhooks/:id` | `manage:webhooks` |
| POST | `/webhooks/:id/test` | `manage:webhooks` |
| GET | `/webhooks/:id/deliveries` | `read:webhooks` |
| GET | `/webhooks/:id/deliveries/:deliveryId` | `read:webhooks` |
| POST | `/webhooks/:id/deliveries/:deliveryId/retry` | `manage:webhooks` |

**Audit & Monitoring**
| Method | Path | Permission |
|--------|------|-----------|
| GET | `/audit-logs` | `read:audit_logs` |
| GET | `/jobs` | `*` (admin) |
| POST | `/jobs/:id/retry` | `*` (admin) |
| DELETE | `/jobs/:id` | `*` (admin) |

**Permission Registry**
| Method | Path | Permission |
|--------|------|-----------|
| GET | `/permissions/registry` | `*` (admin) |
| GET | `/permissions/by-feature` | `*` (admin) |
| POST | `/permissions/registry` | `*` (admin) |
| PUT | `/permissions/registry/:id` | `*` (admin) |
| DELETE | `/permissions/registry/:id` | `*` (admin) |
| POST | `/permissions/sync` | `*` (admin) |
| GET | `/permissions/export` | `*` (admin) |
| GET | `/templates` | `*` (admin) |
| POST | `/templates` | `*` (admin) |
| PUT | `/templates/:id` | `*` (admin) |
| DELETE | `/templates/:id` | `*` (admin) |

**Health**
| Method | Path |
|--------|------|
| GET | `/health` |

| GET | `/api/about` |

---

## 5. Security Model

### Authentication
- **JWT** signed with HMAC-SHA256 (`JWT_SECRET`)
- **Claims:** `user_id`, `role`, `is_super_admin`, `token_id`, `impersonator_id`
- **Session validation:** Token ID checked against database + optional Redis revocation store
- **Introspection:** `POST /api/auth/introspect` with rate-limited per-service-key access
- **Fail-closed:** Any DB/Redis error returns `active: false`

### Authorization
- **RBAC:** Role → Permission mapping via `role_permissions` table
- **Wildcard:** Permission `*` grants all actions
- **Middleware checks:** `RBACMiddleware(verifier, permission)` applied per-route
- **Fail-closed:** Query error or missing permission → deny

### Rate Limiting
- **IP-based:** Auth endpoints (register: 3/min, login: 10/min, verify: 10/min)
- **Architecture:** Token bucket (Redis Lua script or in-memory fallback)
- **Memory fallback:** `sync.Map`, max 50,000 entries, periodic purge

### CSRF Protection
- **Pattern:** Double Submit Cookie
- **Cookie:** `csrf_token` — random 32-byte hex, `HttpOnly=false`, `Secure=true`, `SameSite=None`
- **Validation:** `X-CSRF-Token` header must match cookie value on all POST/PUT/PATCH/DELETE
- **Config:** `CSRF_ENABLED=true` (default) — set to `false` for same-origin deployments (frontend reverse-proxied under same domain)
- **Defense-in-depth:** Origin/Referer header validation always active regardless of CSRF_ENABLED flag
- **Frontend:** Axios request interceptor reads cookie and attaches header automatically

### Impersonation Tracking
- Admins with `manage:users` can impersonate any user via `POST /api/impersonate`
- JWT claim `impersonator_id` tracks the real admin
- In-context propagation: `JWTMiddleware` stores impersonator_id on request context
- Database layer: `RunInTransaction` reads impersonator_id from context and sets `app.impersonator_id` via `set_config()`
- Audit trigger captures both `actor_id` (target user) and `impersonator_id` (real admin) in every log entry

### Database-Level Hardening
- **CHECK constraints** on all enum-like VARCHAR columns:
  - `users.status` — `PENDING_VERIFICATION`, `ACTIVE`, `SUSPENDED`, `HALTED`, `FORCE_PASSWORD_RESET`
  - `webhook_deliveries.status` — `pending`, `delivering`, `delivered`, `failed`, `retrying`
  - `jobs.status` — `pending`, `processing`, `done`, `failed`, `cancelled`
  - `user_identities.provider` — `google`, `github`, `apple`, `email`, `sso`
  - `notifications.type` — `system`, `warning`, `info`, `error`, `alert`
  - `user_notification_preferences.channel` — `in_app`, `email`, `sms`, `push`
- **FK ON DELETE actions:** `api_keys.created_by` → CASCADE, `webhooks.created_by` → CASCADE, `dead_letter_jobs.original_job_id` → SET NULL
- **Partial unique index:** `users(email) WHERE deleted_at IS NULL` — allows soft-delete email reuse
- **Sensitive fields stripped from audit logs:** `password_hash`, `secret` (webhooks), `key_hash`/`key_lookup_hash` (API keys), `token_hash` (verification)

### Additional Protections
- Security headers (CSP, HSTS, X-Frame-Options, X-Content-Type-Options)
- CORS with explicit allowlist
- Body limit (10MB)
- Request timeout (30s)
- Registration email leak prevention (uniform 201 response)
- Password complexity (8+ chars, upper+lower+digit+special)
- Account lockout after failed attempts
- First-user registration serialized via `pg_advisory_xact_lock`
- Soft-deleted users cannot verify email or be promoted to super_admin

---

## 6. Database Schema

### Migrations
- `000001_init.sql` — Fresh install schema (tables, indexes, seeds, audit triggers)
- `000002_hardening.sql` — Hardening layer: `audit_trigger_func` v2 (impersonator_id + sensitive field stripping), FK ON DELETE, CHECK constraints, missing indexes, 5 missing audit triggers
- **Rule:** Always create new `.sql` files with higher prefix — never edit existing migrations

### Tables (21 total)

| Table | Key Columns | Notes |
|-------|------------|-------|
| `users` | id, email, password_hash, full_name, role_id (FK), status, is_super_admin, failed_login_attempts, locked_until, last_login_at, deleted_at | Partial unique email index WHERE deleted_at IS NULL |
| `roles` | id, name, description | Seeded: super_admin, admin, manager, viewer |
| `permissions` | id, name, description | Format: `<domain>:<verb>` |
| `role_permissions` | id, role_id (FK), permission_id (FK) | Maps roles to permissions |
| `sessions` | id, user_id (FK), token_id, expires_at, revoked_at | |
| `audit_logs` | id, actor_id, impersonator_id, action, target_entity, before_state (JSONB), after_state (JSONB), created_at | Written via DB trigger — all 21 tables covered |
| `settings` | id, key (UNIQUE), value | Global key-value store |
| `feature_flags` | id, name, key (UNIQUE), enabled | Global feature flags |
| `file_metadata` | id, filename, mime_type, size_bytes, storage_path, uploaded_by (FK), is_public, checksum_sha256, deleted_at | S3 storage |
| `api_keys` | id, name, key_hash, key_prefix, key_lookup_hash, created_by (FK), expires_at, rotated_at, revoked_at | Key rotation support via 000011 migration |
| `webhooks` | id, name, url, secret, events (TEXT[]), active, created_by (FK), resolved_ips (TEXT[]) | DNS rebinding protection via resolved_ips |
| `webhook_deliveries` | id, webhook_id (FK), event, payload (JSONB), response_code, status | |
| `jobs` | id, type, payload (JSONB), status, idempotency_key (UNIQUE), max_retries, retry_count | Background queue |
| `dead_letter_jobs` | id, original_job_id, type, payload, error_message | Renamed from extension_dlq |
| `permission_registry` | id, domain, name, description | Synced from permissions.yaml |
| `global_templates` | id, name, description, category, is_active, permissions (JSONB) | Permission set blueprints |
| `user_identities` | id, user_id (FK), provider, provider_id | OAuth identities |
| `user_verifications` | id, user_id (FK), token_hash (UNIQUE), expires_at | Email verification — stores SHA-256 hash of token |
| `notifications` | id, user_id (FK), type, title, body, data (JSONB), is_read | |
| `user_notification_preferences` | id, user_id (FK), notification_type, channel, enabled | |
| `user_preferences` | id, user_id (FK), key, value | |

---

## 7. Middleware Stack

| Order | Middleware | File | Purpose |
|-------|-----------|------|---------|
| 1 | Observability | `observability.go` | Structured JSON logging via slog, X-Request-ID injection |
| 2 | Recover | (Echo built-in) | Panic recovery |
| 3 | Timeout | (Echo built-in) | 30s request timeout |
| 4 | CORS | (Echo built-in) | Cross-origin with explicit allowlist |
| 5 | Security Headers | `security.go` | CSP, HSTS, X-Frame-Options, X-Content-Type-Options |
| 6 | CSRF | `security.go` | Double Submit Cookie pattern (token in cookie + header). Configurable via `CSRF_ENABLED` env var. Origin/Referer defense-in-depth always active. |
| 7 | Body Limit | (Echo built-in) | 10MB max body |

**Per-route middleware:**
- `JWTMiddleware` (`auth.go`) — Validates JWT, extracts claims to context (includes `impersonator_id`)
- `RBACMiddleware` (`rbac.go`) — Checks user has required permission
- `LimitIP` (`limiter.go`) — IP-based rate limiting for auth endpoints

---

## 8. Infrastructure Dependencies

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL 15 | ✅ Required | Primary database |
| S3/MinIO | ✅ Required | File storage (fails to start without S3 config) |
| Redis 7 | ❌ Optional | Session revocation (Mode A), rate limiting (falls back to in-memory) |
| SMTP | ❌ Optional | Email sending (queue retries if unavailable) |
| Google OAuth | ❌ Optional | Social login |

---

## 9. Background Job Queue

### Job Types
| Type | Executor | Purpose |
|------|----------|---------|
| `EMAIL_SEND` | `EmailSendExecutor` | Sends emails via SMTP |
| `SECURITY_EVENT_LOG` | `SecurityEventLogExecutor` | Logs security events |
| `WEBHOOK_DISPATCH` | `WebhookDispatcher` | Dispatches webhook HTTP calls |
| `NOTIFICATION_CREATE` | `NotificationCreateExecutor` | Creates in-app notifications |

### Worker Configuration
- Max 3 retries before DLQ
- Heartbeat-based claiming (prevents split-brain)
- Stop via `Manager.Stop()` waits for in-flight jobs
- Channel-per-type buffering (EMAIL_SEND: 5, SECURITY_EVENT_LOG: 20, WEBHOOK_DISPATCH: 10, NOTIFICATION_CREATE: 20)

---

## 10. Observability

- **Structured logging** via `slog` with X-Request-ID per request
- **Health check:** `GET /health`
- **Audit trail:** Every mutation captures `actor_id`, `action`, `before/after` JSON snapshots
- **No built-in metrics/prometheus** — gap to be filled as needed

---

## 11. Permission Registry (`permissions.yaml`)

| Domain | Permissions |
|--------|------------|
| `identity` | `read:users`, `manage:users` |
| `roles` | `read:roles`, `manage:roles` |
| `permissions` | `read:permissions`, `manage:permissions`, `manage:role_permissions` |
| `audit` | `read:audit_logs` |
| `settings` | `read:settings`, `manage:settings` |
| `feature-flags` | `read:feature_flags`, `manage:feature_flags` |
| `storage` | `read:files`, `write:files`, `manage:files`, `delete:files` |
| `api-keys` | `read:api_keys`, `manage:api_keys` |
| `webhooks` | `read:webhooks`, `manage:webhooks` |
| `sessions` | `read:sessions` |

---

## 12. System Invariants

- **Mutations must be logged:** All tables have audit triggers — no DB insert/update/delete bypasses audit.
- **Audit logs capture impersonation:** `audit_logs` has both `actor_id` (target user) and `impersonator_id` (real admin) columns.
- **Sensitive data excluded from audit:** `password_hash`, webhook `secret`, `key_hash`/`key_lookup_hash`, `token_hash` are stripped by the trigger function.
- **Authentication is fail-closed:** Missing/invalid token → 401.
- **DB down degrades gracefully:** `JWTMiddleware` treats DB errors as session revoked (not cache-broken).
- **No orphans on rollback:** Failed file DB transactions clean up S3 objects.
- **Job idempotency:** Duplicate jobs rejected by unique `idempotency_key`.
- **DLQ for exhausted retries:** Jobs failing 3× move to `dead_letter_jobs`.
- **S3 config is mandatory:** Server fails to start without it.
- **No cross-module imports:** Modules communicate via injected interfaces and events only.
- **Registration email leak prevention:** Duplicate registrations return uniform 201.
- **Single role per user:** `role_id` on `users` table (no membership table).
- **CSRF is configurable:** `CSRF_ENABLED=true` for cross-origin (Double Submit Cookie), `false` for same-origin.
- **First-user registration is serialized:** `pg_advisory_xact_lock` prevents race → multiple super_admins.
- **Soft-deleted users can't escalate:** Verification tokens and super_admin promotion checks `deleted_at IS NULL`.
- **OAuth ID tokens are signature-verified:** Google tokeninfo endpoint is NOT used; verification uses local JWKS (RS256) only.

---

## 13. Automated Enforcement (CI)

### Module Boundary Enforcer
`cmd/check-imports/main.go` — Rejects cross-module imports.
```sh
go run ./cmd/check-imports/ -root internal/
```
**Allowlist:** `audit`, `queue`, `rbac`, `settings`, `permregistry`, `database`, `middleware`, `config`, `redisstore`, `authverify`, `container`, `validation`, `pkg/`

### Environment Variables Added
| Variable | Default | Description |
|----------|---------|-------------|
| `CSRF_ENABLED` | `true` | Enable Double Submit Cookie CSRF protection. Set to `false` only for same-origin deployments. |

---

## 14. Default Roles (Seeded)

| Role | Permissions | Intended For |
|------|-------------|-------------|
| `super_admin` | `*` (wildcard — all access) | Platform owners |
| `admin` | Manage users, roles, settings, all read | System administrators |
| `manager` | Read users, manage content | Operations team |
| `viewer` | Read-only access to permitted resources | Read-only users |
