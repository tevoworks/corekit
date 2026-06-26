package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

type mockRBAC struct {
	allowed bool
	err     error
}

func (m *mockRBAC) CheckAccess(ctx context.Context, userID int64, permissionName string) (bool, error) {
	return m.allowed, m.err
}

func TestRBACMiddlewareSuperAdmin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(SuperAdminKey, true)
	c.Set(UserIDKey, int64(1))

	rbac := &mockRBAC{allowed: false}
	mw := RBACMiddleware(rbac, "test:perm")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("super admin should always pass: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}
}

func TestRBACMiddlewareAllowed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(SuperAdminKey, false)
	c.Set(UserIDKey, int64(1))

	rbac := &mockRBAC{allowed: true}
	mw := RBACMiddleware(rbac, "test:perm")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("allowed should pass: %v", err)
	}
}

func TestRBACMiddlewareDenied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(SuperAdminKey, false)
	c.Set(UserIDKey, int64(1))

	rbac := &mockRBAC{allowed: false}
	mw := RBACMiddleware(rbac, "test:perm")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %d", rec.Code)
	}
}

func TestRBACMiddlewareError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(SuperAdminKey, false)
	c.Set(UserIDKey, int64(1))

	rbac := &mockRBAC{allowed: false, err: context.DeadlineExceeded}
	mw := RBACMiddleware(rbac, "test:perm")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %d", rec.Code)
	}
}
