package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSecurityHeadersCSPNonce(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMiddleware("development")
	handler := mw(func(c echo.Context) error {
		nonce := GetCSPNonce(c)
		if nonce == "" {
			t.Fatal("expected non-empty CSP nonce")
		}
		if len(nonce) != 32 {
			t.Fatalf("expected nonce length 32, got: %d", len(nonce))
		}
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected CSP header")
	}
	nonceFromHeader := ""
	for _, part := range strings.Split(csp, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "script-src") {
			for _, val := range strings.Fields(part) {
				if strings.HasPrefix(val, "'nonce-") {
					nonceFromHeader = strings.TrimPrefix(strings.TrimSuffix(val, "'"), "'nonce-")
					break
				}
			}
		}
	}
	if nonceFromHeader == "" {
		t.Fatal("expected nonce in CSP script-src directive")
	}
}

func TestSecurityHeadersCSPNonceUniquePerRequest(t *testing.T) {
	e := echo.New()

	var nonces []string
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := SecurityHeadersMiddleware("development")
		handler := mw(func(c echo.Context) error {
			nonces = append(nonces, GetCSPNonce(c))
			return c.String(http.StatusOK, "ok")
		})
		_ = handler(c)
	}

	if nonces[0] == nonces[1] || nonces[1] == nonces[2] {
		t.Fatal("expected unique nonce per request")
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMiddleware("production")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	h := rec.Header()
	tests := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
		"Permissions-Policy",
	}
	for _, key := range tests {
		if h.Get(key) == "" {
			t.Errorf("header %s should be set", key)
		}
	}

	if h.Get("X-XSS-Protection") != "0" {
		t.Errorf("X-XSS-Protection should be 0, got: %s", h.Get("X-XSS-Protection"))
	}
}

func TestSecurityHeadersHSTS(t *testing.T) {
	tests := []struct {
		env      string
		wantHSTS bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := SecurityHeadersMiddleware(tt.env)
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			_ = handler(c)

			h := rec.Header().Get("Strict-Transport-Security")
			if tt.wantHSTS && h == "" {
				t.Error("expected HSTS header in production")
			}
			if !tt.wantHSTS && h != "" {
				t.Errorf("unexpected HSTS header in %s: %s", tt.env, h)
			}
		})
	}
}

func TestCSRFMiddlewareSafeMethods(t *testing.T) {
	e := echo.New()
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			if err := handler(c); err != nil {
				t.Fatalf("safe method %s should pass: %v", method, err)
			}
			_ = rec
		})
	}
}

func TestCSRFMiddlewareValidOrigin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("valid origin should pass: %v", err)
	}
	_ = rec
}

func TestCSRFMiddlewareInvalidOrigin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %d", rec.Code)
	}
}

func TestCSRFMiddlewareMissingHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %d", rec.Code)
	}
}

func TestCSRFMiddlewareValidReferer(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Referer", "http://localhost:5173/some/path")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("valid referer should pass: %v", err)
	}
	_ = rec
}

func TestCSRFMiddlewareInvalidReferer(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Referer", "https://evil.com")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %d", rec.Code)
	}
}

func TestCSRFMiddlewareRefererWithInvalidOrigin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Referer", "https://evil.com/page")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CSRFMiddleware([]string{"http://localhost:5173"}, false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when origin is invalid, got: %d", rec.Code)
	}
}
