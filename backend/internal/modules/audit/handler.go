package audit

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc, rbacMiddleware ...echo.MiddlewareFunc) {
	middlewares := []echo.MiddlewareFunc{authMiddleware}
	middlewares = append(middlewares, rbacMiddleware...)
	g.GET("/audit-logs", h.ListLogs, middlewares...)
}

func (h *Handler) ListLogs(c echo.Context) error {
	ctx := c.Request().Context()

	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val >= 1 && val <= 100 {
			limit = val
		}
	}
	var cursor int64
	if cs := c.QueryParam("cursor"); cs != "" {
		if val, err := strconv.ParseInt(cs, 10, 64); err == nil {
			cursor = val
		}
	}

	actorIDStr := c.QueryParam("actor_id")
	action := c.QueryParam("action")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	if dateFrom != "" {
		if _, err := time.Parse(time.RFC3339, dateFrom); err != nil {
			return httputil.BadRequest(c, "Invalid date_from format. Use RFC3339 (e.g., 2024-01-01T00:00:00Z)")
		}
	}
	if dateTo != "" {
		if _, err := time.Parse(time.RFC3339, dateTo); err != nil {
			return httputil.BadRequest(c, "Invalid date_to format. Use RFC3339 (e.g., 2024-01-01T00:00:00Z)")
		}
	}
	if dateFrom != "" && dateTo != "" {
		from, _ := time.Parse(time.RFC3339, dateFrom)
		to, _ := time.Parse(time.RFC3339, dateTo)
		if to.Sub(from) > 30*24*time.Hour {
			return httputil.BadRequest(c, "Date range must not exceed 30 days")
		}
	}

	var logs []AuditLog
	var err error

	if actorIDStr != "" || action != "" || dateFrom != "" || dateTo != "" {
		var actorID *int64
		if val, parseErr := strconv.ParseInt(actorIDStr, 10, 64); parseErr == nil {
			actorID = &val
		}
		logs, err = h.service.GetLogsFiltered(ctx, actorID, action, dateFrom, dateTo, limit, cursor)
	} else {
		logs, err = h.service.GetLogs(ctx, limit, cursor)
	}
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(logs) > 0 {
		nextCursor = logs[len(logs)-1].ID
	}

	return httputil.OKWithMeta(c, logs, map[string]interface{}{
		"count":       len(logs),
		"limit":       limit,
		"cursor":      cursor,
		"next_cursor": nextCursor,
	})
}
