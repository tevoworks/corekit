package middleware

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

const (
	UserIDKey         = "user_id"
	RoleKey           = "role"
	SuperAdminKey     = "is_super_admin"
	TokenIDKey        = "token_id"
	ImpersonatorIDKey = "impersonator_id"
)

type JWTClaims struct {
	UserID         int64  `json:"user_id"`
	Role           string `json:"role"`
	IsSuperAdmin   bool   `json:"is_super_admin"`
	TokenID        string `json:"token_id,omitempty"`
	ImpersonatorID *int64 `json:"impersonator_id,omitempty"`
	jwt.RegisteredClaims
}

func JWTMiddleware(secret string, db *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := ""
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					return httputil.Unauthorized(c, "Invalid Authorization header format")
				}
				tokenStr = parts[1]
			} else if cookie, err := c.Cookie("token"); err == nil && cookie.Value != "" {
				tokenStr = cookie.Value
			}
			if tokenStr == "" {
				return httputil.Unauthorized(c, "Missing Authorization header or token cookie")
			}
			claims := &JWTClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))

			if err != nil || !token.Valid {
				return httputil.Unauthorized(c, "Invalid or expired token")
			}

			if claims.TokenID == "" {
				return httputil.Unauthorized(c, "Invalid or expired token")
			}

			if db != nil {
				var revokedAt *time.Time
				var userStatus string
				err := db.QueryRowContext(c.Request().Context(),
					`SELECT s.revoked_at, u.status
					 FROM sessions s
					 JOIN users u ON u.id = s.user_id
					 WHERE s.token_id = $1 AND s.expires_at > NOW() AND u.deleted_at IS NULL LIMIT 1`,
					claims.TokenID,
				).Scan(&revokedAt, &userStatus)
				if err == sql.ErrNoRows {
					return httputil.Unauthorized(c, "Session not found or expired")
				}
				if err != nil {
					return httputil.Unauthorized(c, "Invalid or expired token")
				}
				if revokedAt != nil {
					return httputil.Unauthorized(c, "Session has been revoked")
				}
				if userStatus != "ACTIVE" && userStatus != "FORCE_PASSWORD_RESET" {
					return httputil.Unauthorized(c, "Account is not active")
				}
			}

			isSuperAdmin := claims.IsSuperAdmin || claims.Role == "super_admin"

			c.Set(UserIDKey, claims.UserID)
			c.Set(RoleKey, claims.Role)
			c.Set(SuperAdminKey, isSuperAdmin)
			c.Set(TokenIDKey, claims.TokenID)
			if claims.ImpersonatorID != nil {
				c.Set(ImpersonatorIDKey, claims.ImpersonatorID)
				req := c.Request()
				c.SetRequest(req.WithContext(WithImpersonatorID(req.Context(), claims.ImpersonatorID)))
			}
			return next(c)
		}
	}
}

// ServiceAuthMiddleware validates incoming requests from other internal services
// using a shared secret (SERVICE_API_KEY env var) sent via the X-Service-API-Key header.
// Used by the auth verification endpoint (/api/auth/introspect) so external services
// (e.g. API gateway, other backends) can validate user tokens without knowing the JWT secret.
// In a single-service deployment this is unused; set SERVICE_API_KEY only when needed.
func ServiceAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := os.Getenv("SERVICE_API_KEY")
			if apiKey == "" {
				return next(c)
			}
			requestKey := c.Request().Header.Get("X-Service-API-Key")
			if requestKey == "" {
				return httputil.Unauthorized(c, "Missing service authentication key")
			}
			if requestKey != apiKey {
				return httputil.Unauthorized(c, "Invalid service authentication key")
			}
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) int64 {
	val := c.Get(UserIDKey)
	if val == nil {
		return 0
	}
	if id, ok := val.(int64); ok {
		return id
	}
	return 0
}

func GetUserRole(c echo.Context) string {
	val := c.Get(RoleKey)
	if val == nil {
		return ""
	}
	if role, ok := val.(string); ok {
		return role
	}
	return ""
}

func IsSuperAdmin(c echo.Context) bool {
	val := c.Get(SuperAdminKey)
	if val == nil {
		return false
	}
	if sa, ok := val.(bool); ok {
		return sa
	}
	return false
}

func GetImpersonatorID(c echo.Context) *int64 {
	val := c.Get(ImpersonatorIDKey)
	if val == nil {
		return nil
	}
	if id, ok := val.(*int64); ok {
		return id
	}
	return nil
}

func GetTokenID(c echo.Context) string {
	val := c.Get(TokenIDKey)
	if val == nil {
		return ""
	}
	if tid, ok := val.(string); ok {
		return tid
	}
	return ""
}

func WithImpersonatorID(ctx context.Context, id *int64) context.Context {
	return database.WithImpersonatorID(ctx, id)
}

func GetImpersonatorIDFromCtx(ctx context.Context) *int64 {
	return database.GetImpersonatorIDFromCtx(ctx)
}
