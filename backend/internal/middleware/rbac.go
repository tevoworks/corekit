package middleware

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type RBACVerifier interface {
	CheckAccess(ctx context.Context, userID int64, permissionName string) (bool, error)
}

func RequireSuperAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !IsSuperAdmin(c) {
				slog.Warn("super admin access denied",
					slog.Int64("user_id", GetUserID(c)),
					slog.String("path", c.Path()),
				)
				return httputil.Forbidden(c, "You do not have permission to perform this action")
			}
			return next(c)
		}
	}
}

func RBACMiddleware(verifier RBACVerifier, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if IsSuperAdmin(c) {
				return next(c)
			}

			userID := GetUserID(c)

			allowed, err := verifier.CheckAccess(c.Request().Context(), userID, permission)
			if err != nil || !allowed {
				slog.Warn("access denied by RBAC middleware",
					slog.Int64("user_id", userID),
					slog.String("permission", permission),
					slog.String("path", c.Path()),
				)
				if err != nil {
					slog.Error("RBAC check error", slog.String("error", err.Error()))
				}
				return httputil.Forbidden(c, "You do not have permission to perform this action")
			}

			return next(c)
		}
	}
}
