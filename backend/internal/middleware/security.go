package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

const CSPNonceKey = "csp_nonce"

func generateCSPNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func SecurityHeadersMiddleware(appEnv string, allowedOrigins []string) echo.MiddlewareFunc {
	imgSrc := "'self' data:"
	connectSrc := "'self'"
	for _, origin := range allowedOrigins {
		origin = strings.TrimRight(origin, "/")
		if origin != "" {
			imgSrc += " " + origin
			connectSrc += " " + origin
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nonce := generateCSPNonce()
			c.Set(CSPNonceKey, nonce)

			// Add the API server's own origin to img/connect-src for admin SPA compatibility
			scheme := "http"
			if c.Request().TLS != nil {
				scheme = "https"
			}
			apiOrigin := fmt.Sprintf("%s://%s", scheme, c.Request().Host)

			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "0")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			csp := fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src %s %s; font-src 'self' https://fonts.gstatic.com; connect-src %s %s; frame-ancestors 'none'",
				nonce, imgSrc, apiOrigin, connectSrc, apiOrigin)
			c.Response().Header().Set("Content-Security-Policy", csp)
			c.Response().Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			if appEnv == "production" {
				c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			return next(c)
		}
	}
}

func GetCSPNonce(c echo.Context) string {
	if val, ok := c.Get(CSPNonceKey).(string); ok {
		return val
	}
	return ""
}

const csrfCookieName = "csrf_token"

// generateCSRFToken returns a random 32-byte hex string.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CSRFMiddleware implements Double Submit Cookie pattern:
//   - Sets a csrf_token cookie on every response (if missing)
//   - Validates X-CSRF-Token header matches the cookie on state-changing requests
//   - Also validates Origin/Referer header as defense-in-depth
//
// Can be disabled via CSRFEnabled config flag (e.g., for same-origin deployments).
// When disabled, only the lightweight Origin/Referer check runs.
func CSRFMiddleware(allowedOrigins []string, enabled bool) echo.MiddlewareFunc {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[strings.TrimRight(o, "/")] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isSecure := c.Request().TLS != nil
			// Ensure CSRF token cookie exists (generate if missing)
			if enabled {
				cookie, err := c.Cookie(csrfCookieName)
				if err != nil || cookie.Value == "" {
					token, err := generateCSRFToken()
					if err != nil {
						return httputil.InternalError(c)
					}
					c.SetCookie(&http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false,
					Secure:   isSecure,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   86400,
				})
				}
			}

			// Only validate state-changing requests
			if c.Request().Method == http.MethodGet ||
				c.Request().Method == http.MethodHead ||
				c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Origin/Referer check (always active, defense-in-depth)
			origin := c.Request().Header.Get("Origin")
			referer := c.Request().Header.Get("Referer")

			if origin == "" && referer == "" {
				return httputil.Forbidden(c, "CSRF validation failed: missing origin or referer header")
			}

			if origin != "" {
				origin = strings.TrimRight(origin, "/")
				if !originSet[origin] {
					return httputil.Forbidden(c, "CSRF validation failed: request origin not allowed")
				}
			} else if referer != "" {
				refURL, err := url.Parse(referer)
				if err == nil && refURL.IsAbs() {
					refOrigin := refURL.Scheme + "://" + refURL.Host
					if !originSet[refOrigin] {
						return httputil.Forbidden(c, "CSRF validation failed: request origin not allowed")
					}
				}
			}

			// Token check (only when CSRF is enabled)
			if enabled {
				cookie, err := c.Cookie(csrfCookieName)
				if err != nil || cookie.Value == "" {
					return httputil.Forbidden(c, "CSRF validation failed: missing token cookie")
				}
				headerToken := c.Request().Header.Get("X-CSRF-Token")
				if headerToken == "" || headerToken != cookie.Value {
					return httputil.Forbidden(c, "CSRF validation failed: invalid or missing token")
				}
			}

			return next(c)
		}
	}
}
