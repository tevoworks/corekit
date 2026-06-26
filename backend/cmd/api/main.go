package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tevoworks/corekit/backend/internal/config"
	"github.com/tevoworks/corekit/backend/internal/container"
	appMiddleware "github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/iam"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

var cfg *config.Config

func init() {
	cfg = config.Load()
}

func main() {
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	cont := container.NewContainer(cfg)

	e := echo.New()
	e.HideBanner = true
	e.Validator = validation.NewEchoValidator()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		code := http.StatusInternalServerError
		msg := "An internal error occurred"
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			code = httpErr.Code
			if m, ok := httpErr.Message.(string); ok {
				msg = m
			}
		} else {
			slog.Error("unexpected error", "error", err.Error())
		}
		errCode := httpCodeToErrorCode(code)
		c.JSON(code, map[string]interface{}{
			"error": map[string]string{"code": errCode, "message": msg},
		})
	}

	e.Use(appMiddleware.ObservabilityMiddleware())
	e.Use(echomw.Recover())
	e.Use(echomw.TimeoutWithConfig(echomw.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}))
	e.Use(appMiddleware.SecurityHeadersMiddleware(cfg.AppEnv))
	e.Use(appMiddleware.CSRFMiddleware(cfg.AllowedOrigins, cfg.CSRFEnabled))
	e.Use(echomw.BodyLimit("10M"))

	authMW := appMiddleware.JWTMiddleware(cfg.JWTSecret, cont.DB)
	adminMW := appMiddleware.RequireSuperAdmin()

	apiGroup := e.Group("/api")
	authGroup := e.Group("/api", authMW)

	cont.AuthVerifyH.RegisterRoutes(e, appMiddleware.ServiceAuthMiddleware(), appMiddleware.LimitIP(cfg.IntrospectionRateLimit))
	cont.IAMH.RegisterRoutes(e, authGroup, authMW)
	cont.AuditH.RegisterRoutes(apiGroup, authMW, appMiddleware.RBACMiddleware(cont.RBACSvc, "read:audit_logs"))
	cont.RBACH.RegisterRoutes(apiGroup, authMW)
	cont.QueueH.RegisterRoutes(apiGroup, adminMW)
	cont.PermRegH.RegisterRoutes(authGroup, adminMW)
	cont.SettingsH.RegisterRoutes(apiGroup, authMW)
	cont.APIKeyH.RegisterRoutes(apiGroup, authMW)
	cont.WebhookH.RegisterRoutes(apiGroup, authMW)

	storageAuthGroup := e.Group("/api", authMW)
	publicGroup := e.Group("/api/public")
	cont.StorageH.RegisterRoutes(storageAuthGroup, publicGroup, authMW)

	healthLimit := appMiddleware.LimitIP(60)
	e.GET("/api/health", func(ec echo.Context) error {
		status := "ok"
		dbStatus := "ok"
		if err := cont.DB.PingContext(ec.Request().Context()); err != nil {
			dbStatus = "degraded"
			status = "degraded"
		}
		return httputil.OK(ec, map[string]string{"status": status, "database": dbStatus})
	}, healthLimit)

	e.GET("/health", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/api/health")
	})

	e.GET("/api/about", func(c echo.Context) error {
		return httputil.OK(c, map[string]interface{}{
			"version": "1.0.0",
			"build":   "corekit",
		})
	})

	executors := iam.GetQueueExecutors(cont.DB, cfg)
	workerManager := queue.StartWorker(cont.QueueRepo, cont.DB, executors)

	// Schedule audit log pruning every 24 hours (retention: 365 days)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Run once on startup
		if n, err := cont.AuditSvc.PruneAuditLogs(context.Background(), 365); err != nil {
			slog.Warn("audit log pruning failed on startup", "error", err.Error())
		} else if n > 0 {
			slog.Info("audit logs pruned on startup", "count", n)
		}
		for range ticker.C {
			if n, err := cont.AuditSvc.PruneAuditLogs(context.Background(), 365); err != nil {
				slog.Warn("audit log pruning failed", "error", err.Error())
			} else if n > 0 {
				slog.Info("audit logs pruned", "count", n)
			}
		}
	}()

	if err := cont.PermRegSvc.SyncFromYAML(context.Background(), "permissions.yaml"); err != nil {
		slog.Warn("permission sync failed", "error", err.Error())
	} else {
		slog.Info("Permission registry synced from permissions.yaml")
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("Starting server", "addr", addr, "env", cfg.AppEnv)

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	workerManager.Stop()
	cont.IAMH.Stop()
	cont.RevStore.Close()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	slog.Info("Server stopped")
}

func httpCodeToErrorCode(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	default:
		return "INTERNAL_ERROR"
	}
}
