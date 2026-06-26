package authverify

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/validation"
)

// Handler exposes the introspection API for external service consumption.
type Handler struct {
	svc Service
}

// NewHandler constructs the introspection handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the introspection endpoint.
func (h *Handler) RegisterRoutes(e *echo.Echo, serviceAuthMW, ipLimitMW echo.MiddlewareFunc) {
	e.POST("/api/auth/introspect", h.Introspect, serviceAuthMW, ipLimitMW)
}

// introspectRequest is the inbound JSON body from external services.
type introspectRequest struct {
	Token      string     `json:"token" validate:"required"`
	ActionType ActionType `json:"action_type"`
	Permission string     `json:"permission,omitempty"`
}

// Introspect handles POST /api/auth/introspect.
//
// Security invariants:
//   - Fail-closed: any internal error → active:false (never 5xx that leaks info)
//   - CRITICAL responses are never served from cache
func (h *Handler) Introspect(c echo.Context) error {
	// ── Parse request ─────────────────────────────────────────────────────
	var req introspectRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	// Default action type
	if req.ActionType == "" {
		req.ActionType = ActionREAD
	}

	// ── Execute introspection ─────────────────────────────────────────────
	introspectReq := IntrospectRequest{
		Token:      req.Token,
		ActionType: req.ActionType,
		Permission: req.Permission,
	}
	resp := h.svc.Introspect(c.Request().Context(), introspectReq)

	// Always return 200 — active:false is the fail-closed signal.
	// Never return 5xx for introspection failures (leaks internal state).
	return c.JSON(http.StatusOK, resp)
}
