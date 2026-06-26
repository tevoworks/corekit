package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	return db
}

func TestConnect(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	_ = db
}

func TestRunMigrations(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	if err := RunMigrations(db, "../migrations/"); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one migration record")
	}

	var tableCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount == 0 {
		t.Fatal("expected tables to exist after migration")
	}
}

func TestRunMigrationsIdempotent(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	if err := RunMigrations(db, "../migrations/"); err != nil {
		t.Fatalf("first migration run: %v", err)
	}
	if err := RunMigrations(db, "../migrations/"); err != nil {
		t.Fatalf("second migration run should be idempotent: %v", err)
	}
}

func TestWithTx(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := WithTx(nil, nil)
	if GetTx(ctx) != nil {
		t.Fatal("expected nil tx")
	}
}

func TestWithAuditAction(t *testing.T) {
	ctx := WithAuditAction(context.Background(), "TEST_ACTION")
	if a := GetAuditAction(ctx); a != "TEST_ACTION" {
		t.Fatalf("expected TEST_ACTION, got: %s", a)
	}

	ctx2 := WithAuditAction(context.Background(), "")
	if a := GetAuditAction(ctx2); a != "" {
		t.Fatalf("expected empty, got: %s", a)
	}
}

func TestWithCtxUserID(t *testing.T) {
	ctx := WithCtxUserID(context.Background(), 42)
	id := GetCtxUserID(ctx)
	if id == nil || *id != 42 {
		t.Fatalf("expected 42, got: %v", id)
	}

	ctx2 := WithCtxUserID(context.Background(), 0)
	id2 := GetCtxUserID(ctx2)
	if id2 != nil {
		t.Fatalf("expected nil for user_id=0, got: %v", id2)
	}
}

func TestWithAuditCtx(t *testing.T) {
	ctx := WithAuditCtx(context.Background(), 100, "DO_SOMETHING")
	if a := GetAuditAction(ctx); a != "DO_SOMETHING" {
		t.Fatalf("expected DO_SOMETHING, got: %s", a)
	}
	id := GetCtxUserID(ctx)
	if id == nil || *id != 100 {
		t.Fatalf("expected 100, got: %v", id)
	}
}

func TestGetQueryer(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	q := GetQueryer(nil, db)
	if q == nil {
		t.Fatal("expected non-nil queryer")
	}
}
