package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/tevoworks/corekit/backend/internal/database"
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

func TestPruneExpiredNegativeDuration(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()
	repo := NewRepository(db)

	_, err := repo.PruneExpired(ctx, -1*time.Hour)
	if err == nil {
		t.Fatal("expected error for negative duration")
	}

	_, err = repo.PruneExpired(ctx, 0)
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
}

func TestCancelUsesUpdateNotDelete(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()
	repo := NewRepository(db)

	payload, _ := json.Marshal(map[string]string{"test": "data"})
	err := repo.Enqueue(ctx, nil, JobTypeEmailSend, payload, nil)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	var jobID int64
	_ = db.QueryRow("SELECT id FROM jobs WHERE type = $1 LIMIT 1", JobTypeEmailSend).Scan(&jobID)

	err = repo.Cancel(ctx, jobID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM jobs WHERE id = $1", jobID).Scan(&status)
	if err == sql.ErrNoRows {
		t.Fatal("Cancel() deleted the job instead of updating status to 'cancelled' — no audit trail")
	}
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "cancelled" {
		t.Fatalf("expected status 'cancelled', got: %s", status)
	}
}
