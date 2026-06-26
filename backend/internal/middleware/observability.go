package middleware

import (
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var sensitivePattern = regexp.MustCompile(`(?i)(password|token|secret|authorization|key|signature)=[^&\s"]+`)

func sanitizeError(msg string) string {
	return sensitivePattern.ReplaceAllString(msg, "$1=[REDACTED]")
}

const CorrelationIDHeader = "X-Correlation-ID"

var logger *slog.Logger

func init() {
	level := slog.LevelInfo
	if os.Getenv("APP_ENV") == "development" {
		level = slog.LevelDebug
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func ObservabilityMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			reqID := c.Request().Header.Get(echo.HeaderXRequestID)
			if reqID == "" {
				reqID = uuid.New().String()
				c.Response().Header().Set(echo.HeaderXRequestID, reqID)
			}

			corrID := c.Request().Header.Get(CorrelationIDHeader)
			if corrID == "" {
				corrID = uuid.New().String()
				c.Response().Header().Set(CorrelationIDHeader, corrID)
			}

			c.Set("request_id", reqID)
			c.Set("correlation_id", corrID)

			err := next(c)

			userID := GetUserID(c)
			path := c.Path()
			method := c.Request().Method
			status := c.Response().Status
			latency := time.Since(start)

			attrs := []slog.Attr{
				slog.String("request_id", reqID),
				slog.String("correlation_id", corrID),
				slog.Int64("user_id", userID),
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.Duration("latency", latency),
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", sanitizeError(err.Error())))
				logger.LogAttrs(c.Request().Context(), slog.LevelError, "request failed", attrs...)
			} else if status >= 400 {
				logger.LogAttrs(c.Request().Context(), slog.LevelWarn, "request warning", attrs...)
			} else {
				logger.LogAttrs(c.Request().Context(), slog.LevelInfo, "request completed", attrs...)
			}

			return err
		}
	}
}
