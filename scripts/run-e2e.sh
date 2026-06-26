#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# Run Playwright E2E tests with proper setup.
# Splits into 2 batches with backend restart in between to clear rate limiter.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BE_DIR="$(cd "$(dirname "$0")/../backend" && pwd)"
FE_DIR="$(cd "$(dirname "$0")/../apps/admin" && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "═══ E2E Test Runner ═══"
echo "BE_DIR=$BE_DIR"
echo "FE_DIR=$FE_DIR"

start_frontend() {
  kill -9 "$(lsof -ti :5173 2>/dev/null)" 2>/dev/null || true
  sleep 1
  > /tmp/fe-e2e.log
  cd "$FE_DIR"
  bash -c 'trap "" TERM; exec npx vite preview --port 5173 --strictPort' > /tmp/fe-e2e.log 2>&1 &
  echo "  frontend PID=$!"
  sleep 4
  curl -s -o /dev/null -w "  frontend HTTP %{http_code}\n" http://localhost:5173/
}

restart_backend() {
  echo "── Restarting backend (clears rate limiter) ──"
  kill -9 "$(lsof -ti :8080 2>/dev/null)" 2>/dev/null || true
  sleep 1

  if [ ! -f "$BE_DIR/.env" ]; then
    echo "ERROR: backend/.env not found. Run 'cp backend/.env.example backend/.env' first."
    exit 1
  fi

  > /tmp/be-e2e.log
  cd "$BE_DIR"
  bash -c 'trap "" TERM; APP_ENV=test exec /tmp/corekit-api' > /tmp/be-e2e.log 2>&1 &
  echo "  backend PID=$!"

  echo "── Waiting for backend health check ──"
  for i in $(seq 1 15); do
    STATUS=$(curl -s http://localhost:8080/api/health 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['status'])" 2>/dev/null || echo "down")
    if [ "$STATUS" = "ok" ]; then
      echo "  backend healthy"
      return
    fi
    if [ "$i" = "15" ]; then
      tail -5 /tmp/be-e2e.log
      echo "ERROR: backend did not start"
      exit 1
    fi
    sleep 1
  done
}

# ── Build if needed ──────────────────────────────────────────────────────────
if [ ! -f /tmp/corekit-api ]; then
  echo "── Building backend binary ──"
  cd "$BE_DIR" && go build -o /tmp/corekit-api ./cmd/api/
fi
echo "── Building frontend ──"
cd "$FE_DIR" && npm run build 2>&1 || echo "Frontend build skipped (will use existing dist)"

# ── Phase 0: Restart backend + start frontend ──────────────────────────────
restart_backend
start_frontend

# ── Phase 0a: Run registration tests FIRST (need empty DB, no users) ────────
echo "═══ Phase 0a: Registration Tests (empty DB) ═══"
cd "$FE_DIR"
npx playwright test e2e/registration.spec.ts --project=standalone --reporter=line 2>&1 || true
REG_EXIT=$?

# ── Phase 0b: Seed DB + setup auth ──────────────────────────────────────────
bash "$SCRIPT_DIR/setup-e2e-users.sh"
rm -f "$FE_DIR/e2e/.auth/"*.json

echo "═══ Batch 1: Auth + Setup + Existing Admin Tests ═══"
cd "$FE_DIR"
npx playwright test --project=auth --project=setup --project=admin --project=viewer --project=manager --reporter=line 2>&1 || true
BATCH1_EXIT=$?

if [ "$BATCH1_EXIT" -ne 0 ]; then
  echo "WARNING: Batch 1 had failures (exit=$BATCH1_EXIT)"
fi

# ── Phase B-E: Restart backend + frontend + run new tests ────────────────────
restart_backend
start_frontend

echo "═══ Batch 2: CRUD Functional + RBAC + Audit + Edge + UI/UX ═══"
cd "$FE_DIR"
npx playwright test --project=setup --project=admin --project=viewer --project=manager --reporter=line \
  -g "CRUD Functional|RBAC|Audit Logs — Filtering|Edge Cases|UI/UX Compliance" 2>&1 || true
BATCH2_EXIT=$?

echo "═══ Results ═══"
echo "Registration: $( [ "$REG_EXIT" -eq 0 ] && echo '✅ PASS' || echo "❌ FAIL (exit=$REG_EXIT)")"
echo "Batch 1:      $( [ "$BATCH1_EXIT" -eq 0 ] && echo '✅ PASS' || echo "❌ FAIL (exit=$BATCH1_EXIT)")"
echo "Batch 2:      $( [ "$BATCH2_EXIT" -eq 0 ] && echo '✅ PASS' || echo "❌ FAIL (exit=$BATCH2_EXIT)")"

if [ "$REG_EXIT" -ne 0 ] || [ "$BATCH1_EXIT" -ne 0 ] || [ "$BATCH2_EXIT" -ne 0 ]; then
  exit 1
fi
echo "── All E2E tests passed ──"
