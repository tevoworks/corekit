package settings

import (
	"strconv"

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
	return &Handler{
		service:     s,
		rbacService: rbacService,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/settings", h.SetSetting, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:settings"))
	g.GET("/settings", h.ListSettings, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:settings"))
	g.GET("/settings/:key", h.GetSetting, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:settings"))
	g.DELETE("/settings/:key", h.DeleteSetting, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:settings"))

	g.POST("/feature-flags", h.CreateFlag, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:feature_flags"))
	g.GET("/feature-flags", h.ListFlags, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:feature_flags"))
	g.GET("/feature-flags/:key", h.LookupFlag, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:feature_flags"))
	g.PUT("/feature-flags/:id", h.UpdateFlag, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:feature_flags"))
	g.DELETE("/feature-flags/:id", h.DeleteFlag, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:feature_flags"))
}

type SetSettingRequest struct {
	Key   string `json:"key" validate:"required,min=1,max=255"`
	Value string `json:"value" validate:"required,max=65535"`
}

func (h *Handler) SetSetting(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req SetSettingRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	s, err := h.service.SetSetting(ctx, req.Key, req.Value, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, s)
}

func (h *Handler) GetSetting(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("key")

	s, err := h.service.GetSetting(ctx, key)
	if err != nil {
		return httputil.InternalError(c)
	}

	if s == nil {
		return httputil.NotFound(c, "Setting not found")
	}

	return httputil.OK(c, s)
}

func (h *Handler) ListSettings(c echo.Context) error {
	ctx := c.Request().Context()

	settings, err := h.service.ListSettings(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OKWithMeta(c, settings, map[string]interface{}{"count": len(settings)})
}

func (h *Handler) DeleteSetting(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	key := c.Param("key")
	if key == "" {
		return httputil.BadRequest(c, "Setting key is required")
	}

	if err := h.service.DeleteSetting(ctx, key, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Setting deleted successfully")
}

type CreateFlagRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Key         string `json:"key" validate:"required"`
	Description string `json:"description" validate:"nohtml"`
	Enabled     bool   `json:"enabled"`
}

type UpdateFlagRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Key         string `json:"key" validate:"required"`
	Description string `json:"description" validate:"nohtml"`
	Enabled     bool   `json:"enabled"`
}

func (h *Handler) CreateFlag(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateFlagRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	f, err := h.service.CreateFlag(ctx, req.Name, req.Key, req.Description, req.Enabled, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, f)
}

func (h *Handler) LookupFlag(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("key")

	enabled, err := h.service.LookupFlag(ctx, key)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, map[string]interface{}{"enabled": enabled})
}

func (h *Handler) UpdateFlag(c echo.Context) error {
	ctx := c.Request().Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid flag ID")
	}
	actorID := middleware.GetUserID(c)

	var req UpdateFlagRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	f, err := h.service.UpdateFlag(ctx, id, req.Name, req.Key, req.Description, req.Enabled, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, f)
}

func (h *Handler) DeleteFlag(c echo.Context) error {
	ctx := c.Request().Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid flag ID")
	}
	actorID := middleware.GetUserID(c)

	err = h.service.DeleteFlag(ctx, id, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Feature flag deleted successfully")
}

func (h *Handler) ListFlags(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)

	list, err := h.service.ListFlags(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(list) > 0 {
		nextCursor = list[len(list)-1].ID
	}

	return httputil.Paginated(c, list, nextCursor, p.Limit)
}
