package queue

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(g *echo.Group, superAdminMW echo.MiddlewareFunc) {
	g.GET("/jobs", h.ListJobs, superAdminMW)
	g.POST("/jobs/:id/retry", h.RetryJob, superAdminMW)
	g.DELETE("/jobs/:id", h.CancelJob, superAdminMW)
}

func (h *Handler) ListJobs(c echo.Context) error {
	ctx := c.Request().Context()

	limitVal := 50
	cursorVal := int64(0)
	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limitVal = val
		}
	}
	if limitVal > 100 {
		limitVal = 100
	}
	if limitVal < 1 {
		limitVal = 50
	}
	if cs := c.QueryParam("cursor"); cs != "" {
		if val, err := strconv.ParseInt(cs, 10, 64); err == nil {
			cursorVal = val
		}
	}

	statusFilter := c.QueryParam("status")
	typeFilter := c.QueryParam("type")

	jobs, err := h.repo.List(ctx, statusFilter, typeFilter, cursorVal, limitVal)
	if err != nil {
		return httputil.InternalError(c)
	}

	var pendingCount, processingCount, failedCount int
	for _, j := range jobs {
		switch j.Status {
		case StatusPending:
			pendingCount++
		case StatusProcessing:
			processingCount++
		case StatusFailed:
			failedCount++
		}
	}

	nextCursor := int64(0)
	if len(jobs) > 0 {
		nextCursor = jobs[len(jobs)-1].ID
	}

	return httputil.OKWithMeta(c, jobs, map[string]interface{}{
		"count":       len(jobs),
		"limit":       limitVal,
		"cursor":      cursorVal,
		"next_cursor": nextCursor,
		"pending":     pendingCount,
		"processing":  processingCount,
		"failed":      failedCount,
	})
}

func (h *Handler) RetryJob(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid job ID")
	}

	if err := h.repo.Retry(ctx, id); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Job retry scheduled successfully")
}

func (h *Handler) CancelJob(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid job ID")
	}

	if err := h.repo.Cancel(ctx, id); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Job cancelled successfully")
}
