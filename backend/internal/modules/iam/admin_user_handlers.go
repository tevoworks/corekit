package iam

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/validation"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

type AdminCreateUserRequest struct {
	Email        string `json:"email" validate:"required,emailfmt"`
	FullName     string `json:"full_name" validate:"required,nohtml"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

func (h *Handler) AdminCreateUser(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req AdminCreateUserRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	if req.IsSuperAdmin && !middleware.IsSuperAdmin(c) {
		return httputil.Forbidden(c, "Only super administrators can create super admin accounts")
	}

	u, err := h.service.AdminCreateUser(ctx, req.Email, req.FullName, req.IsSuperAdmin, actorID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "already exists"):
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "A user with this email already exists")
		default:
			return httputil.InternalError(c)
		}
	}

	return httputil.Created(c, u)
}

type AdminUpdateUserRequest struct {
	Email    string `json:"email" validate:"required,emailfmt"`
	FullName string `json:"full_name" validate:"required,nohtml"`
}

func (h *Handler) AdminUpdateUser(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid user ID")
	}

	var req AdminUpdateUserRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	u, err := h.service.AdminUpdateUser(ctx, targetUserID, req.Email, req.FullName, actorID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "user not found"):
			return httputil.NotFound(c, "User not found")
		case strings.Contains(err.Error(), "email already taken"):
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "Email is already taken")
		default:
			return httputil.InternalError(c)
		}
	}

	return httputil.OK(c, u)
}

type AdminChangeUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=ACTIVE SUSPENDED HALTED PENDING_VERIFICATION"`
}

func (h *Handler) AdminChangeUserStatus(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid user ID")
	}

	if actorID == targetUserID {
		return httputil.BadRequest(c, "You cannot change your own status")
	}

	var req AdminChangeUserStatusRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	err = h.service.AdminChangeUserStatus(ctx, targetUserID, req.Status, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "User status updated successfully")
}

func (h *Handler) AdminForceResetPassword(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid user ID")
	}

	err = h.service.ForceResetPassword(ctx, targetUserID, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "User password reset successfully and invitation email sent")
}

func (h *Handler) ListUserSessions(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid user ID")
	}

	sessions, err := h.service.ListSessions(ctx, targetUserID)
	if err != nil {
		return httputil.InternalError(c)
	}

	for i := range sessions {
		sessions[i].MaskIP()
	}

	return httputil.OK(c, sessions)
}
