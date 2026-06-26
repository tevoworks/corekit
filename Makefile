.PHONY: dev dev-be dev-fe dev-mkt stop restart lint test test-e2e test-e2e-auth build docker-up docker-down \
        logs db-reset migration-up migration-down migration-reset clean scaffold docker-build setup help

BE_DIR    = backend
FE_DIR    = apps/admin
MK_DIR    = apps/marketing
BE_PORT   = 8080
FE_PORT   = 5173
MK_PORT   = 3000

dev: docker-up
	@echo "── Starting all services ──"
	@trap 'kill 0' EXIT; \
		$(MAKE) dev-be & \
		$(MAKE) dev-fe & \
		$(MAKE) dev-mkt & \
		wait

dev-be:
	@echo "Starting backend (port $(BE_PORT))..."
	@cd $(BE_DIR) && go run ./cmd/api

dev-fe:
	@echo "Starting admin (port $(FE_PORT))..."
	@cd $(FE_DIR) && npm run dev

dev-mkt:
	@echo "Starting marketing (port $(MK_PORT))..."
	@cd $(MK_DIR) && npm run dev

stop:
	@echo "── Stopping all services ──"
	@-lsof -ti :$(BE_PORT) 2>/dev/null | xargs kill -15 2>/dev/null; sleep 10; lsof -ti :$(BE_PORT) 2>/dev/null | xargs kill -15 2>/dev/null || true
	@-lsof -ti :$(FE_PORT) 2>/dev/null | xargs kill -15 2>/dev/null; sleep 10; lsof -ti :$(FE_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@-lsof -ti :$(MK_PORT) 2>/dev/null | xargs kill -15 2>/dev/null; sleep 10; lsof -ti :$(MK_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@-docker compose down 2>/dev/null || true
	@echo "Done."

restart: stop dev

lint:
	@echo "── Checking module boundaries ──"
	cd $(BE_DIR) && go run ./cmd/check-imports/ -root $(CURDIR)/$(BE_DIR)
	@echo "── Running go vet ──"
	cd $(BE_DIR) && go vet ./...
	@echo "── Running gosec ──"
	cd $(BE_DIR) && which gosec 2>/dev/null && gosec -quiet -exclude-dir=vendor ./... || echo "gosec not found — install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"

test:
	@echo "── Running backend tests ──"
	cd $(BE_DIR) && go test -v -count=1 ./...

test-e2e:
	@echo "── Running full E2E test suite ──"
	./scripts/run-e2e.sh

test-e2e-auth:
	@echo "── Running full E2E tests (auth project) ──"
	@kill -9 $$(lsof -ti :8080 2>/dev/null) 2>/dev/null || true
	@kill -9 $$(lsof -ti :5173 2>/dev/null) 2>/dev/null || true
	@sleep 1
	@cp backend/bin/api /tmp/corekit-api 2>/dev/null || true
	@cd backend && bash -c 'trap "" TERM; exec nohup /tmp/corekit-api' > /tmp/be-e2e.log 2>&1 &
	@cd $(FE_DIR) && bash -c 'trap "" TERM; exec nohup npx vite preview --port 5173 --strictPort' > /tmp/fe-e2e.log 2>&1 &
	@sleep 6
	@PGPASSWORD=postgres psql -h localhost -p 5434 -U postgres -d corekit -c "DELETE FROM users WHERE email NOT IN ('admin@corekit.com','viewer@test.corekit','manager@test.corekit');" 2>/dev/null || true
	@bash scripts/setup-e2e-users.sh
	@rm -f $(FE_DIR)/e2e/.auth/*.json
	@cd $(FE_DIR) && npx playwright test --project=auth --project=setup --reporter=line || true
	@kill -9 $$(lsof -ti :5173 2>/dev/null) 2>/dev/null || true
	@kill -9 $$(lsof -ti :8080 2>/dev/null) 2>/dev/null || true

test-e2e-quick: build
	@echo "── Running existing E2E tests only ──"
	@kill -9 $$(lsof -ti :8080 2>/dev/null) 2>/dev/null || true
	@sleep 1
	@cd backend && bash -c 'trap "" TERM; exec nohup /tmp/corekit-api' > /tmp/be-e2e.log 2>&1 &
	@sleep 6
	@bash scripts/setup-e2e-users.sh
	@rm -f $(FE_DIR)/e2e/.auth/*.json
	@cd $(FE_DIR) && npx playwright test --project=auth --project=setup --project=admin --project=viewer --project=manager --reporter=line || true
	@kill -9 $$(lsof -ti :5173 2>/dev/null) 2>/dev/null || true
	@kill -9 $$(lsof -ti :8080 2>/dev/null) 2>/dev/null || true

build:
	@echo "── Building backend ──"
	cd $(BE_DIR) && go build -o bin/api ./cmd/api
	@echo "── Building admin ──"
	cd $(FE_DIR) && npm run build
	@echo "── Building marketing ──"
	cd $(MK_DIR) && npm run build
	@echo "Build complete."

docker-build:
	@echo "── Building Docker image ──"
	docker build -t corekit .
	@echo "Done. Run: docker compose -f docker-compose.prod.yml up"

docker-up:
	@docker compose up -d --no-recreate db redis minio 2>/dev/null || true

docker-down:
	docker compose down

logs:
	docker compose logs -f

db-reset:
	@if [ "${APP_ENV}" = "production" ]; then \
		echo "ERROR: Refusing to reset database in production (APP_ENV=production)"; \
		exit 1; \
	fi
	@echo "── Resetting database ──"
	@docker compose exec -T db psql -U $${POSTGRES_USER:-postgres} -c "DROP DATABASE IF EXISTS corekit;" 2>/dev/null || true
	@docker compose exec -T db psql -U $${POSTGRES_USER:-postgres} -c "CREATE DATABASE corekit;" 2>/dev/null || true
	@echo "Database reset. Run 'make dev' to re-apply migrations."

migration-up:
	@echo "── Running pending migrations ──"
	cd $(BE_DIR) && go run ./cmd/migrate up

migration-down:
	@echo "── Rolling back last migration ──"
	cd $(BE_DIR) && go run ./cmd/migrate down

migration-reset:
	@echo "── Resetting all migrations ──"
	cd $(BE_DIR) && go run ./cmd/migrate reset

scaffold:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make scaffold name=<module-name>"; \
		exit 1; \
	fi
	./scripts/scaffold-module.sh $(name)

setup:
	./scripts/setup.sh

clean:
	@echo "── Cleaning ──"
	rm -rf $(BE_DIR)/bin $(BE_DIR)/tmp
	rm -rf $(FE_DIR)/dist $(FE_DIR)/node_modules $(MK_DIR)/.next $(MK_DIR)/node_modules .turbo node_modules
	@echo "Done."

help:
	@echo "Usage:"
	@echo "  make dev              Start backend + admin + marketing + infra (hot-reload)"
	@echo "  make stop             Stop all services (backend, admin, marketing, docker)"
	@echo "  make restart          Stop then start"
	@echo "  make dev-be           Start backend only (port 8080)"
	@echo "  make dev-fe           Start admin only (port 5173)"
	@echo "  make dev-mkt          Start marketing only (port 3000)"
	@echo "  make build            Build production assets (all apps)"
	@echo "  make docker-build     Build Docker image"
	@echo "  make lint             Check imports + go vet"
	@echo "  make test             Run backend unit tests"
	@echo "  make test-e2e         Run full Playwright E2E suite (87 tests, 2 batches with BE restart)"
	@echo "  make test-e2e-quick   Run existing E2E tests only (81 tests, 1 batch)"
	@echo "  make scaffold name=X  Generate a new backend module"
	@echo "  make setup            Bootstrap project (docker + deps)"
	@echo "  make logs             Follow docker logs"
	@echo "  make docker-up        Start infra containers"
	@echo "  make docker-down      Stop infra containers"
	@echo "  make migration-up     Run all pending migrations"
	@echo "  make migration-down   Roll back the last migration"
	@echo "  make migration-reset  Roll back all, then re-apply"
	@echo "  make db-reset         Drop and recreate database"
	@echo "  make clean            Remove build artifacts"
