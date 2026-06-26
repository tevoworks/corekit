#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# Create/ensure E2E test users in the corekit database.
# Run after migrations are applied (roles + permissions exist).
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5434}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-postgres}"
DB_NAME="${DB_NAME:-corekit}"

PSQL="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
export PGPASSWORD="$DB_PASS"

echo "── Seeding E2E test users ──"

# Viewer role
VIEWER_ROLE_ID=$($PSQL -t -c "SELECT id FROM roles WHERE name = 'viewer' LIMIT 1;" | tr -d ' ')
MANAGER_ROLE_ID=$($PSQL -t -c "SELECT id FROM roles WHERE name = 'manager' LIMIT 1;" | tr -d ' ')

echo "  viewer role id=$VIEWER_ROLE_ID"
echo "  manager role id=$MANAGER_ROLE_ID"

# viewer@test.corekit / ViewerPass1!
$PSQL -c "
INSERT INTO users (email, password_hash, full_name, is_super_admin, role_id, status)
SELECT 'viewer@test.corekit', '\$2a\$10\$ydmNhzCvIFkUq2YOHV8lNuL3.XuJtrVO7RMNPKozeFsQjdgUWMEOa', 'Test Viewer', false, $VIEWER_ROLE_ID, 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'viewer@test.corekit');
" 2>/dev/null || echo "  viewer user already exists"

# manager@test.corekit / ManagerPass1!
$PSQL -c "
INSERT INTO users (email, password_hash, full_name, is_super_admin, role_id, status)
SELECT 'manager@test.corekit', '\$2a\$10\$21c0JDk2aNsc24XksWsdaeU1Q0Hq.DggKwASmyCioLXTi5ybhmPFe', 'Test Manager', false, $MANAGER_ROLE_ID, 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'manager@test.corekit');
 " 2>/dev/null || echo "  manager user already exists"

# admin@corekit.com / Admin123!
$PSQL -c "
INSERT INTO users (email, password_hash, full_name, is_super_admin, status)
SELECT 'admin@corekit.com', '\$2a\$10\$EflZ0VuvCq7otUimJUXCBePVnxjhqLM9Pte/dnSSaP8jJMTvzGIo.', 'Admin User', true, 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@corekit.com');
" 2>/dev/null || echo "  admin user already exists"

echo "── Users:"
$PSQL -c "SELECT id, email, role_id, status FROM users ORDER BY id;" 2>/dev/null
echo "── Done ──"
