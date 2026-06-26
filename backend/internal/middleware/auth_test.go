package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

func TestServiceAuthMiddlewareNoKey(t *testing.T) {
	os.Unsetenv("SERVICE_API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Service-API-Key", "some-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ServiceAuthMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (skip auth when SERVICE_API_KEY is empty), got: %d", rec.Code)
	}
}

func TestServiceAuthMiddlewareValid(t *testing.T) {
	os.Setenv("SERVICE_API_KEY", "test-secret-key")
	defer os.Unsetenv("SERVICE_API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Service-API-Key", "test-secret-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ServiceAuthMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("should pass: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}
}

func TestServiceAuthMiddlewareInvalid(t *testing.T) {
	os.Setenv("SERVICE_API_KEY", "test-secret-key")
	defer os.Unsetenv("SERVICE_API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Service-API-Key", "wrong-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ServiceAuthMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", rec.Code)
	}
}

func TestServiceAuthMiddlewareMissingHeader(t *testing.T) {
	os.Setenv("SERVICE_API_KEY", "test-secret-key")
	defer os.Unsetenv("SERVICE_API_KEY")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ServiceAuthMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", rec.Code)
	}
}

func TestGetUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if id := GetUserID(c); id != 0 {
		t.Fatalf("expected 0, got: %d", id)
	}

	c.Set(UserIDKey, int64(42))
	if id := GetUserID(c); id != 42 {
		t.Fatalf("expected 42, got: %d", id)
	}
	_ = rec
}

func TestGetUserRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if role := GetUserRole(c); role != "" {
		t.Fatalf("expected empty, got: %s", role)
	}

	c.Set(RoleKey, "admin")
	if role := GetUserRole(c); role != "admin" {
		t.Fatalf("expected admin, got: %s", role)
	}
	_ = rec
}

func TestIsSuperAdmin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if IsSuperAdmin(c) {
		t.Fatal("expected false by default")
	}

	c.Set(SuperAdminKey, true)
	if !IsSuperAdmin(c) {
		t.Fatal("expected true after setting")
	}
	_ = rec
}

func TestJWTMiddlewareMissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := JWTMiddleware("secret", nil)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", rec.Code)
	}
}

func TestJWTMiddlewareInvalidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := JWTMiddleware("secret", nil)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", rec.Code)
	}
}

func TestJWTMiddlewareValidToken(t *testing.T) {
	claims := &JWTClaims{
		UserID:       1,
		Role:         "super_admin",
		IsSuperAdmin: true,
		TokenID:      "test-token-id",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := JWTMiddleware("test-secret", nil)
	var captured bool
	handler := mw(func(c echo.Context) error {
		captured = true
		uid := GetUserID(c)
		if uid != 1 {
			t.Fatalf("expected user_id=1, got: %d", uid)
		}
		role := GetUserRole(c)
		if role != "super_admin" {
			t.Fatalf("expected role=super_admin, got: %s", role)
		}
		if !IsSuperAdmin(c) {
			t.Fatal("expected super admin")
		}
		return nil
	})

	if err := handler(c); err != nil {
		t.Fatalf("valid token should pass: %v", err)
	}
	if !captured {
		t.Fatal("handler was never called")
	}
	_ = rec
}

func TestJWTMiddlewareWithoutTokenID(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
	})
	tokenStr, _ := token.SignedString([]byte("secret"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := JWTMiddleware("secret", nil)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for token without token_id, got: %d", rec.Code)
	}
}

func TestJWTMiddlewareWithDBRejects(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  1,
		"token_id": "nonexistent",
	})
	tokenStr, _ := token.SignedString([]byte("secret"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db, err := sql.Open("postgres", "postgres://localhost:9999/test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	mw := JWTMiddleware("secret", db)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", rec.Code)
	}
}

func TestGetImpersonatorID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if id := GetImpersonatorID(c); id != nil {
		t.Fatalf("expected nil, got: %v", id)
	}

	val := int64(5)
	c.Set(ImpersonatorIDKey, &val)
	if id := GetImpersonatorID(c); id == nil || *id != 5 {
		t.Fatalf("expected 5, got: %v", id)
	}
	_ = rec
}

func TestGetTokenID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if tid := GetTokenID(c); tid != "" {
		t.Fatalf("expected empty, got: %s", tid)
	}

	c.Set(TokenIDKey, "tok-123")
	if tid := GetTokenID(c); tid != "tok-123" {
		t.Fatalf("expected tok-123, got: %s", tid)
	}
	_ = rec
}
