package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type BFfSession struct {
	JWT       string `json:"jwt"`
	UserID    int64  `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func generateBFfSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "bff_" + hex.EncodeToString(b)
}

func BFFSessionMiddleware(redisClient *redis.Client, appEnv string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if redisClient == nil {
				return next(c)
			}

			path := c.Request().URL.Path
			// Skip BFF session check for public endpoints
			if strings.HasPrefix(path, "/api/auth/") ||
				strings.HasPrefix(path, "/api/public/") ||
				strings.HasPrefix(path, "/api/bff/") {
				return next(c)
			}

			cookie, err := c.Cookie("bff_session")
			if err != nil || cookie == nil || cookie.Value == "" {
				return next(c)
			}

			data, err := redisClient.Get(c.Request().Context(), "bff:"+cookie.Value).Bytes()
			if err != nil {
				slog.Debug("BFF session not found, falling back to cookie auth", "session", cookie.Value)
				return next(c)
			}

			var session BFfSession
			if err := json.Unmarshal(data, &session); err != nil {
				return next(c)
			}

			if time.Now().Unix() > session.ExpiresAt {
				redisClient.Del(c.Request().Context(), "bff:"+cookie.Value)
				c.SetCookie(&http.Cookie{
					Name:    "bff_session",
					Value:   "",
					Expires: time.Unix(0, 0),
					Path:    "/",
				})
				return next(c)
			}

			c.Request().Header.Set("Authorization", "Bearer "+session.JWT)
			return next(c)
		}
	}
}
