package permregistry

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) RegisterRoutes(g *echo.Group, superAdminMW echo.MiddlewareFunc) {
	g.GET("/permissions/registry", h.ListRegistry, superAdminMW)
	g.GET("/permissions/by-feature", h.ListRegistryByDomain, superAdminMW)
	g.POST("/permissions/registry", h.RegisterPermission, superAdminMW)
	g.PUT("/permissions/registry/:id", h.UpdateRegistryEntry, superAdminMW)
	g.DELETE("/permissions/registry/:id", h.DeleteRegistryEntry, superAdminMW)

	g.GET("/templates", h.ListGlobalTemplates, superAdminMW)
	g.POST("/templates", h.CreateGlobalTemplate, superAdminMW)
	g.PUT("/templates/:id", h.UpdateGlobalTemplate, superAdminMW)
	g.DELETE("/templates/:id", h.DeleteGlobalTemplate, superAdminMW)

	g.POST("/permissions/sync", h.SyncFromYAML, superAdminMW)
	g.GET("/permissions/export", h.ExportToYAML, superAdminMW)
}

func parseID(c echo.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}
	return id, nil
}

func (h *Handler) ListRegistry(c echo.Context) error {
	ctx := c.Request().Context()
	entries, err := h.service.ListRegistry(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}
	if entries == nil {
		entries = []RegistryEntry{}
	}
	return httputil.OKWithMeta(c, entries, map[string]interface{}{"count": len(entries)})
}

func (h *Handler) ListRegistryByDomain(c echo.Context) error {
	ctx := c.Request().Context()
	grouped, err := h.service.ListRegistryByDomain(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}
	if grouped == nil {
		grouped = []ByDomain{}
	}
	return httputil.OK(c, grouped)
}

func (h *Handler) RegisterPermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	var req CreateRegistryEntryRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	entry, err := h.service.RegisterPermission(ctx, req, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, entry)
}

func (h *Handler) UpdateRegistryEntry(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req UpdateRegistryEntryRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	entry, err := h.service.UpdateRegistryEntry(ctx, id, req, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, entry)
}

func (h *Handler) DeleteRegistryEntry(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.service.DeleteRegistryEntry(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}

func (h *Handler) ExportToYAML(c echo.Context) error {
	ctx := c.Request().Context()
	data, err := h.service.ExportToYAML(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, map[string]string{"yaml": string(data)})
}

func (h *Handler) SyncFromYAML(c echo.Context) error {
	ctx := c.Request().Context()
	if err := h.service.SyncFromYAML(ctx, "permissions.yaml"); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Permission registry synced from permissions.yaml")
}

func (h *Handler) ListGlobalTemplates(c echo.Context) error {
	ctx := c.Request().Context()
	templates, err := h.service.ListGlobalTemplates(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}
	if templates == nil {
		templates = []GlobalTemplate{}
	}
	return httputil.OKWithMeta(c, templates, map[string]interface{}{"count": len(templates)})
}

func (h *Handler) CreateGlobalTemplate(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	var req CreateGlobalTemplateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	t, err := h.service.CreateGlobalTemplate(ctx, req, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, t)
}

func (h *Handler) UpdateGlobalTemplate(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var req UpdateGlobalTemplateRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	t, err := h.service.UpdateGlobalTemplate(ctx, id, req, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, t)
}

func (h *Handler) DeleteGlobalTemplate(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.service.DeleteGlobalTemplate(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}
