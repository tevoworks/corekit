package webhook

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
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
	queueRepo := queue.NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	svc := NewService(db, repo, queueRepo, auditSvc, "")
	return svc, db
}

func cleanupWebhooks(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM webhook_deliveries`)
	_, _ = db.Exec(`DELETE FROM webhooks`)
}

func TestCreateWebhook(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	wh, err := svc.Create(ctx, 1, "Test Webhook", "https://example.com/hook", []string{"user.created"}, "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if wh.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if wh.Name != "Test Webhook" {
		t.Fatalf("expected 'Test Webhook', got: %s", wh.Name)
	}
	if wh.Secret == "" {
		t.Fatal("expected auto-generated secret")
	}
}

func TestCreateWebhookWithCustomSecret(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	wh, err := svc.Create(ctx, 1, "Secret Webhook", "https://example.com/hook", []string{"event.occurred"}, "this-is-a-32-char-custom-secret!!", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if wh.Secret == "" || wh.Secret == "****" {
		t.Fatal("expected custom secret to be set and accessible via RawSecret")
	}
}

func TestCreateWebhookInvalidURL(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, 1, "Bad URL", "http://example.com/hook", []string{"test"}, "", true)
	if err == nil {
		t.Fatal("expected error for non-HTTPS URL")
	}

	_, err = svc.Create(ctx, 1, "Private IP", "https://127.0.0.1/hook", []string{"test"}, "", true)
	if err == nil {
		t.Fatal("expected error for private IP URL")
	}

	_, err = svc.Create(ctx, 1, "Private IP", "https://169.254.169.254/latest/meta-data/", []string{"test"}, "", true)
	if err == nil {
		t.Fatal("expected error for link-local IP (SSRF target)")
	}
}

func TestGetWebhookByID(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "Get Test", "https://example.com/get", []string{"test"}, "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	fetched, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected non-nil webhook")
	}
	if fetched.Name != "Get Test" {
		t.Fatalf("expected 'Get Test', got: %s", fetched.Name)
	}

	notFound, err := svc.GetByID(ctx, 999999)
	if err != nil {
		t.Fatalf("get non-existent: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent webhook")
	}
}

func TestListWebhooks(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	_, _ = svc.Create(ctx, 1, "WH1", "https://example.com/1", []string{"a"}, "", true)
	_, _ = svc.Create(ctx, 1, "WH2", "https://example.com/2", []string{"b"}, "", true)

	list, err := svc.List(ctx, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("expected at least 2 webhooks, got: %d", len(list))
	}
}

func TestUpdateWebhook(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	wh, err := svc.Create(ctx, 1, "Original", "https://example.com/orig", []string{"a"}, "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(ctx, wh.ID, 1, "Updated", "https://example.com/upd", []string{"a", "b"}, "", false)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected 'Updated', got: %s", updated.Name)
	}
	if updated.Active != false {
		t.Fatal("expected active=false")
	}

	fetched, _ := svc.GetByID(ctx, wh.ID)
	if fetched.URL != "https://example.com/upd" {
		t.Fatalf("expected updated URL, got: %s", fetched.URL)
	}
}

func TestDeleteWebhook(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	wh, err := svc.Create(ctx, 1, "Delete Me", "https://example.com/del", []string{"x"}, "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = svc.Delete(ctx, wh.ID, 1)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	fetched, _ := svc.GetByID(ctx, wh.ID)
	if fetched != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDeleteNonExistentWebhook(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	err := svc.Delete(ctx, 999999, 1)
	if err == nil {
		t.Fatal("expected error deleting non-existent webhook")
	}
}

func TestWebhookDeliveryLifecycle(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	cleanupWebhooks(t, db)
	ctx := context.Background()

	wh, err := svc.Create(ctx, 1, "Delivery Test", "https://example.com/dlv", []string{"test.event"}, "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := svc.ListDeliveries(ctx, wh.ID, 20, 0)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if list == nil {
		t.Fatal("expected empty list, not nil")
	}

	err = svc.TestWebhook(ctx, wh.ID, 1)
	if err != nil {
		t.Fatalf("test webhook: %v", err)
	}
}

func TestWebhookMaskSecret(t *testing.T) {
	wh := &Webhook{Secret: "abcdef1234567890abcdef1234567890"}
	wh.MaskSecret()
	if wh.Secret[:8] != "abcdef12" {
		t.Fatalf("expected prefix 'abcdef12', got: %s", wh.Secret[:8])
	}
	if len(wh.Secret) != 8+3+4 {
		t.Fatalf("unexpected masked length: %d", len(wh.Secret))
	}

	short := &Webhook{Secret: "short"}
	short.MaskSecret()
	if short.Secret != "****" {
		t.Fatalf("expected '****', got: %s", short.Secret)
	}

	empty := &Webhook{Secret: ""}
	empty.MaskSecret()
	if empty.Secret != "" {
		t.Fatalf("expected '', got: %s", empty.Secret)
	}
}
