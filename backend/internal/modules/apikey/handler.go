package apikey

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
	return &Handler{service: s, rbacService: rbacService}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/api-keys", h.CreateKey, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:api_keys"))
	g.GET("/api-keys", h.ListKeys, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:api_keys"))
	g.DELETE("/api-keys/:id", h.RevokeKey, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:api_keys"))
	g.POST("/api-keys/:id/rotate", h.RotateKey, authMiddleware, middleware.RBACMiddleware(h.rbacService, "manage:api_keys"))
}

type CreateKeyRequest struct {
	Name string `json:"name" validate:"required,nohtml"`
}

func (h *Handler) CreateKey(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateKeyRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	k, rawKey, err := h.service.CreateKey(ctx, actorID, req.Name)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Created(c, map[string]interface{}{
		"id":         k.ID,
		"name":       k.Name,
		"key_prefix": k.KeyPrefix,
		"raw_key":    rawKey,
		"created_at": k.CreatedAt,
	})
}

func (h *Handler) ListKeys(c echo.Context) error {
	ctx := c.Request().Context()

	keys, err := h.service.ListKeys(ctx)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OKWithMeta(c, keys, map[string]interface{}{
		"count": len(keys),
	})
}

func (h *Handler) RotateKey(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid key ID")
	}

	k, rawKey, err := h.service.RotateKey(ctx, id, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, map[string]interface{}{
		"id":         k.ID,
		"name":       k.Name,
		"key_prefix": k.KeyPrefix,
		"raw_key":    rawKey,
		"expires_at": k.ExpiresAt,
	})
}

func (h *Handler) RevokeKey(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid key ID")
	}

	err = h.service.RevokeKey(ctx, id, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "API key revoked successfully")
}
