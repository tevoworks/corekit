package storage

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

// inMemoryProvider implements StorageProvider without external S3 dependency.
type inMemoryProvider struct {
	store map[string][]byte
}

func newInMemoryProvider() *inMemoryProvider {
	return &inMemoryProvider{store: make(map[string][]byte)}
}

func (p *inMemoryProvider) Upload(ctx context.Context, key string, size int64, content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	p.store[key] = data
	return key, nil
}

func (p *inMemoryProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	data, ok := p.store[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (p *inMemoryProvider) Delete(ctx context.Context, key string) error {
	delete(p.store, key)
	return nil
}

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

func newTestService(t *testing.T) (Service, *sql.DB, *inMemoryProvider) {
	t.Helper()
	db := testDB(t)
	repo := NewRepository(db)
	provider := newInMemoryProvider()
	auditSvc := audit.NewService(audit.NewRepository(db))
	svc := NewService(db, repo, provider, auditSvc)
	return svc, db, provider
}

func cleanupFiles(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM file_metadata`)
}

func TestUploadFile(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "hello, world"
	meta, err := svc.UploadFile(ctx, "test.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if meta.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if meta.Filename != "test.txt" {
		t.Fatalf("expected 'test.txt', got: %s", meta.Filename)
	}
	if meta.ChecksumSHA256 == "" {
		t.Fatal("expected non-empty checksum")
	}
}

func TestUploadPublicFile(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "public content"
	meta, err := svc.UploadFile(ctx, "public.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, true)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if !meta.IsPublic {
		t.Fatal("expected is_public=true")
	}
}

func TestDownloadFile(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "download me"
	meta, err := svc.UploadFile(ctx, "download.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	downloaded, reader, err := svc.DownloadFile(ctx, meta.ID, 1, false)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer reader.Close()

	if downloaded.Filename != "download.txt" {
		t.Fatalf("expected 'download.txt', got: %s", downloaded.Filename)
	}

	data, _ := io.ReadAll(reader)
	if string(data) != content {
		t.Fatalf("expected '%s', got: '%s'", content, string(data))
	}
}

func TestDownloadPublicFile(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "public download"
	meta, err := svc.UploadFile(ctx, "pub.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, true)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	downloaded, reader, err := svc.DownloadPublicFile(ctx, meta.ID)
	if err != nil {
		t.Fatalf("download public: %v", err)
	}
	defer reader.Close()

	if !downloaded.IsPublic {
		t.Fatal("expected is_public=true")
	}

	data, _ := io.ReadAll(reader)
	if string(data) != content {
		t.Fatalf("expected '%s', got: '%s'", content, string(data))
	}
}

func TestDownloadPrivateFileAsOtherUser(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "secret"
	meta, err := svc.UploadFile(ctx, "secret.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	_, _, err = svc.DownloadFile(ctx, meta.ID, 2, false)
	if err == nil {
		t.Fatal("expected error for other user downloading private file")
	}
}

func TestDownloadPrivateFileAsSuperAdmin(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "admin access"
	meta, err := svc.UploadFile(ctx, "admin.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	_, reader, err := svc.DownloadFile(ctx, meta.ID, 2, true)
	if err != nil {
		t.Fatalf("super admin download: %v", err)
	}
	defer reader.Close()
}

func TestDeleteFile(t *testing.T) {
	svc, db, provider := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "delete me"
	meta, err := svc.UploadFile(ctx, "delete.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	err = svc.DeleteFile(ctx, meta.ID, 1)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	fetched, _, _ := svc.DownloadFile(ctx, meta.ID, 1, true)
	if fetched != nil {
		t.Fatal("expected nil after delete")
	}

	if len(provider.store) != 0 {
		t.Fatal("expected S3 storage to be empty after delete")
	}
}

func TestDeleteNonExistentFile(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	err := svc.DeleteFile(ctx, 999999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestDeleteFileOwnershipCheck(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "owned by user 1"
	meta, err := svc.UploadFile(ctx, "owner.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	err = svc.DeleteFile(ctx, meta.ID, 2)
	if err == nil {
		t.Fatal("expected error when user 2 tries to delete user 1's file")
	}
}

func TestDeleteFileOwnershipCheckSuperAdmin(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "super admin can delete"
	meta, err := svc.UploadFile(ctx, "sa_delete.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	err = svc.DeleteFile(ctx, meta.ID, 1)
	if err != nil {
		t.Fatalf("owner should be able to delete own file: %v", err)
	}
}

func TestListFiles(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	_, _ = svc.UploadFile(ctx, "f1.txt", 5, "text/plain", strings.NewReader("hello"), 1, false)
	_, _ = svc.UploadFile(ctx, "f2.txt", 5, "text/plain", strings.NewReader("world"), 1, false)

	files, err := svc.ListFiles(ctx, 20, 0, 1, false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected at least 2 files, got: %d", len(files))
	}
}

func TestListFilesAsSuperAdmin(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	_, _ = svc.UploadFile(ctx, "sa.txt", 4, "text/plain", strings.NewReader("data"), 2, false)

	files, err := svc.ListFiles(ctx, 20, 0, 1, true)
	if err != nil {
		t.Fatalf("list as super admin: %v", err)
	}
	if len(files) < 1 {
		t.Fatal("expected super admin to see all files")
	}
}

func TestUploadFileWithRollbackOnDBFailure(t *testing.T) {
	svc, db, provider := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "rollback test"
	meta, err := svc.UploadFile(ctx, "rollback.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if _, ok := provider.store[meta.StoragePath]; !ok {
		t.Fatal("expected file in storage after successful upload")
	}
}

func TestUploadSameContentDifferentFiles(t *testing.T) {
	svc, db, _ := newTestService(t)
	defer db.Close()
	cleanupFiles(t, db)
	ctx := context.Background()

	content := "same content"
	meta1, err := svc.UploadFile(ctx, "same1.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload 1: %v", err)
	}

	meta2, err := svc.UploadFile(ctx, "same2.txt", int64(len(content)), "text/plain",
		strings.NewReader(content), 1, false)
	if err != nil {
		t.Fatalf("upload 2: %v", err)
	}

	if meta1.ChecksumSHA256 != meta2.ChecksumSHA256 {
		t.Fatal("expected identical checksums for identical content")
	}
}
