package cms

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/rbac"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	svc         Service
	rbacService rbac.Service
}

func NewHandler(svc Service, rbacService rbac.Service) *Handler {
	return &Handler{svc: svc, rbacService: rbacService}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMW echo.MiddlewareFunc) {
	g.GET("/cmss", h.List, authMW, middleware.RBACMiddleware(h.rbacService, "read:cmss"))
	g.POST("/cmss", h.Create, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cmss"))
	g.GET("/cmss/:id", h.GetByID, authMW, middleware.RBACMiddleware(h.rbacService, "read:cmss"))
	g.PUT("/cmss/:id", h.Update, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cmss"))
	g.DELETE("/cmss/:id", h.Delete, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cmss"))
}

type CreateRequest struct {
	Name string `json:"name" validate:"required,nohtml"`
}

type UpdateRequest struct {
	Name string `json:"name" validate:"required,nohtml"`
}

func (h *Handler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	item, err := h.svc.Create(ctx, req.Name, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, item)
}

func (h *Handler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	item, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if item == nil {
		return httputil.NotFound(c, "Cms not found")
	}
	return httputil.OK(c, item)
}

func (h *Handler) List(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)

	items, err := h.svc.List(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}
	return httputil.Paginated(c, items, nextCursor, p.Limit)
}

func (h *Handler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	var req UpdateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	item, err := h.svc.Update(ctx, id, req.Name, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, item)
}

func (h *Handler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}

	if err := h.svc.Delete(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}
