package webhook

import (
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/rbac"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	service     Service
	rbacService rbac.Service
}

func NewHandler(s Service, rbacService rbac.Service) *Handler {
	return &Handler{service: s, rbacService: rbacService}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/webhooks", h.Create, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:webhooks"))
	g.GET("/webhooks", h.List, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:webhooks"))
	g.GET("/webhooks/:id", h.GetByID, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:webhooks"))
	g.PUT("/webhooks/:id", h.Update, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:webhooks"))
	g.DELETE("/webhooks/:id", h.Delete, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:webhooks"))
	g.POST("/webhooks/:id/test", h.Test, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:webhooks"))
	g.GET("/webhooks/:id/deliveries", h.ListDeliveries, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:webhooks"))
	g.GET("/webhooks/:id/deliveries/:deliveryId", h.GetDelivery, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:webhooks"))
	g.POST("/webhooks/:id/deliveries/:deliveryId/retry", h.RetryDelivery, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:webhooks"))
}

type CreateWebhookRequest struct {
	Name   string   `json:"name" validate:"required,nohtml"`
	URL    string   `json:"url" validate:"required,urlstrict"`
	Events []string `json:"events" validate:"required"`
	Secret string   `json:"secret" validate:"omitempty,min=16"`
	Active *bool    `json:"active"`
}

type UpdateWebhookRequest struct {
	Name   string   `json:"name" validate:"required,nohtml"`
	URL    string   `json:"url" validate:"required,urlstrict"`
	Events []string `json:"events" validate:"required"`
	Secret string   `json:"secret" validate:"omitempty,min=16"`
	Active *bool    `json:"active"`
}

func (h *Handler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateWebhookRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(req.URL), "https://") {
		return httputil.BadRequest(c, "Webhook URL must use HTTPS")
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	wh, err := h.service.Create(ctx, actorID, req.Name, req.URL, req.Events, req.Secret, active)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Created(c, wh)
}

func (h *Handler) List(c echo.Context) error {
	ctx := c.Request().Context()

	p := httputil.ParseCursorPagination(c)
	list, err := h.service.List(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(list) > 0 {
		nextCursor = list[len(list)-1].ID
	}

	return httputil.Paginated(c, list, nextCursor, p.Limit)
}

func (h *Handler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	wh, err := h.service.GetByID(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if wh == nil {
		return httputil.NotFound(c, "Webhook not found")
	}

	return httputil.OK(c, wh)
}

func (h *Handler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	var req UpdateWebhookRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(req.URL), "https://") {
		return httputil.BadRequest(c, "Webhook URL must use HTTPS")
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	wh, err := h.service.Update(ctx, id, actorID, req.Name, req.URL, req.Events, req.Secret, active)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, wh)
}

func (h *Handler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	if err := h.service.Delete(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Webhook deleted successfully")
}

func (h *Handler) Test(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	if err := h.service.TestWebhook(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Test event queued")
}

func (h *Handler) ListDeliveries(c echo.Context) error {
	ctx := c.Request().Context()

	whIDStr := c.Param("id")
	whID, err := strconv.ParseInt(whIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	wh, err := h.service.GetByID(ctx, whID)
	if err != nil || wh == nil {
		return httputil.NotFound(c, "Webhook not found")
	}

	limit := 20
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

	deliveries, err := h.service.ListDeliveries(ctx, whID, limit, cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(deliveries) > 0 {
		nextCursor = deliveries[len(deliveries)-1].ID
	}

	return httputil.OKWithMeta(c, deliveries, map[string]interface{}{
		"count":       len(deliveries),
		"limit":       limit,
		"cursor":      cursor,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) GetDelivery(c echo.Context) error {
	ctx := c.Request().Context()

	whIDStr := c.Param("id")
	whID, err := strconv.ParseInt(whIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	dIDStr := c.Param("deliveryId")
	dID, err := strconv.ParseInt(dIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid delivery ID")
	}

	wh, err := h.service.GetByID(ctx, whID)
	if err != nil || wh == nil {
		return httputil.NotFound(c, "Webhook not found")
	}

	d, err := h.service.GetDeliveryByID(ctx, whID, dID)
	if err != nil {
		return httputil.InternalError(c)
	}
	if d == nil {
		return httputil.NotFound(c, "Delivery not found")
	}

	return httputil.OK(c, d)
}

func (h *Handler) RetryDelivery(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	whIDStr := c.Param("id")
	whID, err := strconv.ParseInt(whIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid webhook ID")
	}

	dIDStr := c.Param("deliveryId")
	dID, err := strconv.ParseInt(dIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid delivery ID")
	}

	if err := h.service.RetryDelivery(ctx, whID, dID, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Delivery queued for retry")
}
