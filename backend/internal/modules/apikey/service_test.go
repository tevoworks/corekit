package apikey

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := database.RunMigrations(db, "../../../migrations/"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	db := testDB(t)
	repo := NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	svc := NewService(db, repo, auditSvc)
	return svc, db
}

func cleanupKeys(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM api_keys`)
}

func TestCreateAPIKey(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	key, rawKey, err := svc.CreateKey(ctx, 1, "Test Key")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if key.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if key.Name != "Test Key" {
		t.Fatalf("expected 'Test Key', got: %s", key.Name)
	}
	if !strings.HasPrefix(rawKey, "ca_") {
		t.Fatalf("expected 'ca_' prefix, got: %s", rawKey[:3])
	}
	if key.KeyPrefix == "" {
		t.Fatal("expected non-empty prefix")
	}
	if key.KeyHash == "" {
		t.Fatal("expected non-empty hash")
	}
	if key.RevokedAt != nil {
		t.Fatal("expected nil revoked_at for new key")
	}
}

func TestListAPIKeys(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	_, _, _ = svc.CreateKey(ctx, 1, "Key 1")
	_, _, _ = svc.CreateKey(ctx, 1, "Key 2")

	keys, err := svc.ListKeys(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got: %d", len(keys))
	}
}

func TestValidateAPIKey(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	_, rawKey, err := svc.CreateKey(ctx, 1, "Validate Test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	validated, err := svc.ValidateKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("validate with raw key: %v", err)
	}
	if validated == nil {
		t.Fatal("expected non-nil key")
	}
	if validated.Name != "Validate Test" {
		t.Fatalf("expected 'Validate Test', got: %s", validated.Name)
	}
}

func TestValidateAPIKeyInvalid(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	_, err := svc.ValidateKey(ctx, "invalid-key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestValidateAPIKeyWrongKey(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	_, rawKey, err := svc.CreateKey(ctx, 1, "Wrong Key Test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	tamperedKey := rawKey[:len(rawKey)-1] + "0"
	_, err = svc.ValidateKey(ctx, tamperedKey)
	if err == nil {
		t.Fatal("expected error for tampered key")
	}
}

func TestRevokeAPIKey(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	key, _, err := svc.CreateKey(ctx, 1, "Revoke Test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = svc.RevokeKey(ctx, key.ID, 1)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}

	keys, _ := svc.ListKeys(ctx)
	for _, k := range keys {
		if k.ID == key.ID {
			if k.RevokedAt == nil {
				t.Fatal("expected revoked_at to be set")
			}
		}
	}
}

func TestRevokeAPIKeyInvalid(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	err := svc.RevokeKey(ctx, 999999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestCreateMultipleAPIKeys(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	keys := make(map[string]bool)
	for i := 0; i < 5; i++ {
		_, rawKey, err := svc.CreateKey(ctx, 1, "Multi Test")
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		if keys[rawKey] {
			t.Fatal("duplicate key generated")
		}
		keys[rawKey] = true
	}
}

func TestAPIKeyNilKeyHashValidation(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupKeys(t, db)
	ctx := context.Background()

	_, _, err := svc.CreateKey(ctx, 1, "Hash Check")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	keys, _ := svc.ListKeys(ctx)
	for _, k := range keys {
		if k.KeyHash == "" {
			t.Fatal("expected non-empty key_hash in DB")
		}
	}
}
