package rbac

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/roles", h.CreateRole, authMiddleware, middleware.RBACMiddleware(h.service, "manage:roles"))
	g.GET("/roles", h.ListRoles, authMiddleware, middleware.RBACMiddleware(h.service, "read:roles"))
	g.PUT("/roles/:id", h.UpdateRole, authMiddleware, middleware.RBACMiddleware(h.service, "manage:roles"))
	g.DELETE("/roles/:id", h.DeleteRole, authMiddleware, middleware.RBACMiddleware(h.service, "manage:roles"))

	g.POST("/permissions", h.CreatePermission, authMiddleware, middleware.RBACMiddleware(h.service, "manage:permissions"))
	g.GET("/permissions", h.ListPermissions, authMiddleware, middleware.RBACMiddleware(h.service, "read:permissions"))
	g.PUT("/permissions/:id", h.UpdatePermission, authMiddleware, middleware.RBACMiddleware(h.service, "manage:permissions"))
	g.DELETE("/permissions/:id", h.DeletePermission, authMiddleware, middleware.RBACMiddleware(h.service, "manage:permissions"))

	g.POST("/roles/:id/permissions", h.AssignPermission, authMiddleware, middleware.RBACMiddleware(h.service, "manage:role_permissions"))
	g.DELETE("/roles/:role_id/permissions/:permission_id", h.RemovePermission, authMiddleware, middleware.RBACMiddleware(h.service, "manage:role_permissions"))
	g.POST("/rbac/check", h.CheckAccess, authMiddleware)
}

type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Description string `json:"description" validate:"nohtml"`
}

type CreatePermissionRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Description string `json:"description" validate:"nohtml"`
}

type AssignPermissionRequest struct {
	PermissionID int64 `json:"permission_id" validate:"gt=0"`
}

type CheckAccessRequest struct {
	PermissionName string                 `json:"permission_name" validate:"required"`
	ContextData    map[string]interface{} `json:"context_data"`
}

func (h *Handler) CreateRole(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreateRoleRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	r, err := h.service.CreateRole(ctx, req.Name, req.Description, actorID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "A role with this name already exists")
		}
		return httputil.InternalError(c)
	}

	return httputil.Created(c, r)
}

func (h *Handler) ListRoles(c echo.Context) error {
	ctx := c.Request().Context()

	p := httputil.ParseCursorPagination(c)
	roles, err := h.service.ListRoles(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(roles) > 0 {
		nextCursor = roles[len(roles)-1].ID
	}

	return httputil.Paginated(c, roles, nextCursor, p.Limit)
}

func (h *Handler) CreatePermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreatePermissionRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	p, err := h.service.CreatePermission(ctx, req.Name, req.Description, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Created(c, p)
}

func (h *Handler) ListPermissions(c echo.Context) error {
	ctx := c.Request().Context()

	p := httputil.ParseCursorPagination(c)
	permissions, err := h.service.ListPermissions(ctx, p.Limit, p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(permissions) > 0 {
		nextCursor = permissions[len(permissions)-1].ID
	}

	return httputil.Paginated(c, permissions, nextCursor, p.Limit)
}

type UpdateRoleRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Description string `json:"description" validate:"nohtml"`
}

type UpdatePermissionRequest struct {
	Name        string `json:"name" validate:"required,nohtml"`
	Description string `json:"description" validate:"nohtml"`
}

func (h *Handler) UpdateRole(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid role ID")
	}

	var req UpdateRoleRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	r, err := h.service.UpdateRole(ctx, id, req.Name, req.Description, actorID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "A role with this name already exists")
		}
		return httputil.InternalError(c)
	}

	return httputil.OK(c, r)
}

func (h *Handler) DeleteRole(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid role ID")
	}

	if err := h.service.DeleteRole(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Deleted(c)
}

func (h *Handler) UpdatePermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid permission ID")
	}

	var req UpdatePermissionRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	p, err := h.service.UpdatePermission(ctx, id, req.Name, req.Description, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, p)
}

func (h *Handler) DeletePermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid permission ID")
	}

	if err := h.service.DeletePermission(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Deleted(c)
}

func (h *Handler) AssignPermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseInt(roleIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid role ID")
	}

	var req AssignPermissionRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	err = h.service.AssignPermission(ctx, roleID, req.PermissionID, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Permission assigned to role successfully")
}

func (h *Handler) RemovePermission(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	roleIDStr := c.Param("role_id")
	roleID, err := strconv.ParseInt(roleIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid role ID")
	}

	permIDStr := c.Param("permission_id")
	permID, err := strconv.ParseInt(permIDStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid permission ID")
	}

	err = h.service.RemovePermissionFromRole(ctx, roleID, permID, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "Permission removed from role successfully")
}

func (h *Handler) CheckAccess(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	var req CheckAccessRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	allowed, err := h.service.CheckAccess(ctx, userID, req.PermissionName)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.OK(c, map[string]interface{}{
		"allowed": allowed,
	})
}
