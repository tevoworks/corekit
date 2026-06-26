package rbac

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/validation"
)

func testDBE2E(t *testing.T) *sql.DB {
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

func setupHandler(t *testing.T) (*Handler, *sql.DB, *echo.Echo) {
	t.Helper()
	db := testDBE2E(t)
	repo := NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	svc := NewService(db, repo, auditSvc)
	h := NewHandler(svc)
	e := echo.New()
	e.Validator = validation.NewEchoValidator()
	return h, db, e
}

func authContext(e *echo.Echo, req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.UserIDKey, int64(1))
	c.Set(middleware.SuperAdminKey, true)
	return c, rec
}

func TestCreateRoleE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()

	body := `{"name":"e2e_role","description":"E2E Test Role"}`
	req := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := authContext(e, req)

	if err := h.CreateRole(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got: %d", rec.Code)
	}
}

func TestListRolesE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/roles?limit=10", nil)
	c, _ := authContext(e, req)

	if err := h.ListRoles(c); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePermissionE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()

	body := `{"name":"e2e:perm","description":"E2E Permission"}`
	req := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := authContext(e, req)

	if err := h.CreatePermission(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got: %d", rec.Code)
	}
}

func TestAssignPermissionE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()
	ctx := context.Background()

	role, err := h.service.CreateRole(ctx, "e2e_assign_role", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	perm, err := h.service.CreatePermission(ctx, "e2e:assign", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"permission_id":%d}`, perm.ID)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/roles/%d/permissions", role.ID), bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := authContext(e, req)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", role.ID))

	if err := h.AssignPermission(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}
}

func TestCheckAccessE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()

	body := `{"permission_name":"test:perm"}`
	req := httptest.NewRequest(http.MethodPost, "/rbac/check", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := authContext(e, req)

	if err := h.CheckAccess(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRoleE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()
	ctx := context.Background()

	role, err := h.service.CreateRole(ctx, "e2e_upd_role", "original", 0)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"name":"e2e_upd_role_renamed","description":"updated"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/roles/%d", role.ID), bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, _ := authContext(e, req)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", role.ID))

	if err := h.UpdateRole(c); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRoleE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()
	ctx := context.Background()

	role, err := h.service.CreateRole(ctx, "e2e_del_role", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/roles/%d", role.ID), nil)
	c, _ := authContext(e, req)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", role.ID))

	if err := h.DeleteRole(c); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRoleValidationE2E(t *testing.T) {
	h, db, e := setupHandler(t)
	defer db.Close()

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"description":"test"}`},
		{"html in name", `{"name":"<script>"}`},
		{"empty body", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c, rec := authContext(e, req)
			_ = h.CreateRole(c)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got: %d", rec.Code)
			}
		})
	}
}
