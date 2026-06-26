package rbac

import (
	"context"
	"database/sql"
	"os"
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

func TestCreateRole(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	role, err := svc.CreateRole(ctx, "test_role", "Test Role", 0)
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if role.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if role.Name != "test_role" {
		t.Fatalf("expected test_role, got: %s", role.Name)
	}
}

func TestCreateRoleDuplicate(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	_, err := svc.CreateRole(ctx, "dup_role", "", 0)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = svc.CreateRole(ctx, "dup_role", "", 0)
	if err == nil {
		t.Fatal("expected error on duplicate name")
	}
}

func TestListRoles(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	seeded, err := svc.ListRoles(ctx, 100, 0)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	initialCount := len(seeded)

	_, _ = svc.CreateRole(ctx, "list_role_1", "", 0)
	_, _ = svc.CreateRole(ctx, "list_role_2", "", 0)

	roles, err := svc.ListRoles(ctx, 100, 0)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != initialCount+2 {
		t.Fatalf("expected %d roles, got: %d", initialCount+2, len(roles))
	}
}

func TestCreatePermission(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	perm, err := svc.CreatePermission(ctx, "test:perm", "Test Permission", 0)
	if err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if perm.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if perm.Name != "test:perm" {
		t.Fatalf("expected test:perm, got: %s", perm.Name)
	}
}

func TestCreatePermissionDuplicate(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	_, err := svc.CreatePermission(ctx, "dup:perm", "", 0)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = svc.CreatePermission(ctx, "dup:perm", "", 0)
	if err == nil {
		t.Fatal("expected error on duplicate permission name")
	}
}

func TestListPermissions(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	_, _ = svc.CreatePermission(ctx, "perm:one", "", 0)
	_, _ = svc.CreatePermission(ctx, "perm:two", "", 0)

	perms, err := svc.ListPermissions(ctx, 100, 0)
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(perms) < 2 {
		t.Fatalf("expected at least 2 permissions, got: %d", len(perms))
	}
}

func TestAssignAndRemovePermission(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	role, err := svc.CreateRole(ctx, "assign_role", "", 0)
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	perm, err := svc.CreatePermission(ctx, "assign:perm", "", 0)
	if err != nil {
		t.Fatalf("create permission: %v", err)
	}

	if err := svc.AssignPermission(ctx, role.ID, perm.ID, 0); err != nil {
		t.Fatalf("assign permission: %v", err)
	}

	if err := svc.RemovePermissionFromRole(ctx, role.ID, perm.ID, 0); err != nil {
		t.Fatalf("remove permission: %v", err)
	}
}

func TestUpdateRole(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	role, err := svc.CreateRole(ctx, "update_role", "original", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.UpdateRole(ctx, role.ID, "update_role_renamed", "updated desc", 0)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "update_role_renamed" {
		t.Fatalf("expected update_role_renamed, got: %s", updated.Name)
	}
}

func TestDeleteRole(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	role, err := svc.CreateRole(ctx, "delete_role", "", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.DeleteRole(ctx, role.ID, 0); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestUpdatePermission(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	perm, err := svc.CreatePermission(ctx, "update:perm", "original", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.UpdatePermission(ctx, perm.ID, "update:perm_renamed", "updated", 0)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "update:perm_renamed" {
		t.Fatalf("expected update:perm_renamed, got: %s", updated.Name)
	}
}

func TestDeletePermission(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	perm, err := svc.CreatePermission(ctx, "delete:perm", "", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.DeletePermission(ctx, perm.ID, 0); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestCheckAccessSuperAdmin(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	allowed, err := svc.CheckAccess(ctx, 0, "anything")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if allowed {
		t.Fatal("non-existent user should not be allowed")
	}
}

func TestAssignPermissionNotFound(t *testing.T) {
	svc, db := newTestService(t)
	defer db.Close()
	ctx := context.Background()

	err := svc.AssignPermission(ctx, 99999, 99999, 0)
	if err == nil {
		t.Fatal("expected error for non-existent role/permission")
	}
}
