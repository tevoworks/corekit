package contact

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/database"
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

func (h *Handler) RegisterRoutes(g *echo.Group, publicGroup *echo.Group, authMW echo.MiddlewareFunc) {
	publicGroup.POST("/contact", h.SubmitContact)
	publicGroup.POST("/newsletter/subscribe", h.Subscribe)
	publicGroup.POST("/newsletter/unsubscribe", h.Unsubscribe)

	g.GET("/contact/messages", h.ListContacts, authMW, middleware.RBACMiddleware(h.rbacService, "read:contacts"))
	g.GET("/contact/messages/:id", h.GetContact, authMW, middleware.RBACMiddleware(h.rbacService, "read:contacts"))
	g.PATCH("/contact/messages/:id/status", h.UpdateContactStatus, authMW, middleware.RBACMiddleware(h.rbacService, "manage:contacts"))
	g.POST("/contact/messages/:id/assign", h.AssignContact, authMW, middleware.RBACMiddleware(h.rbacService, "manage:contacts"))
	g.DELETE("/contact/messages/:id", h.DeleteContact, authMW, middleware.RBACMiddleware(h.rbacService, "manage:contacts"))
	g.GET("/contact/subscribers", h.ListSubscribers, authMW, middleware.RBACMiddleware(h.rbacService, "read:contacts"))
	g.DELETE("/contact/subscribers/:id", h.DeleteSubscriber, authMW, middleware.RBACMiddleware(h.rbacService, "manage:contacts"))
}

type SubmitContactRequest struct {
	Name    string `json:"name" validate:"required,nohtml"`
	Email   string `json:"email" validate:"required,emailfmt"`
	Phone   string `json:"phone"`
	Subject string `json:"subject" validate:"required,nohtml"`
	Message string `json:"message" validate:"required"`
}

type SubscribeRequest struct {
	Email string `json:"email" validate:"required,emailfmt"`
	Name  string `json:"name"`
}

type UnsubscribeRequest struct {
	Email string `json:"email" validate:"required,emailfmt"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

type AssignRequest struct {
	UserID int64 `json:"user_id" validate:"required"`
}

func (h *Handler) SubmitContact(c echo.Context) error {
	ctx := c.Request().Context()
	var req SubmitContactRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	source := c.QueryParam("source")
	if source == "" {
		source = "website"
	}
	contact, err := h.svc.SubmitContact(ctx, req.Name, req.Email, req.Phone, req.Subject, req.Message, source)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, contact)
}

func (h *Handler) Subscribe(c echo.Context) error {
	ctx := c.Request().Context()
	var req SubscribeRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	source := c.QueryParam("source")
	if source == "" {
		source = "website"
	}
	sub, err := h.svc.Subscribe(ctx, req.Email, req.Name, source)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, sub)
}

func (h *Handler) Unsubscribe(c echo.Context) error {
	ctx := c.Request().Context()
	var req UnsubscribeRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.Unsubscribe(ctx, req.Email); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, map[string]string{"message": "Unsubscribed successfully"})
}

func (h *Handler) ListContacts(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)
	status := c.QueryParam("status")

	items, err := h.svc.ListContacts(ctx, int64(p.Limit), p.Cursor, status)
	if err != nil {
		return httputil.InternalError(c)
	}
	nextCursor := int64(0)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}
	return httputil.Paginated(c, items, nextCursor, p.Limit)
}

func (h *Handler) GetContact(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}
	contact, err := h.svc.GetContact(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if contact == nil {
		return httputil.NotFound(c, "Contact not found")
	}
	return httputil.OK(c, contact)
}

func (h *Handler) UpdateContactStatus(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}
	var req UpdateStatusRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.UpdateContactStatus(ctx, id, actorID, req.Status); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.NotFound(c, "Contact not found")
		}
		return httputil.InternalError(c)
	}
	return httputil.OK(c, map[string]string{"message": "Status updated"})
}

func (h *Handler) AssignContact(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}
	var req AssignRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.AssignContact(ctx, id, req.UserID, actorID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.NotFound(c, "Contact not found")
		}
		return httputil.InternalError(c)
	}
	return httputil.OK(c, map[string]string{"message": "Contact assigned"})
}

func (h *Handler) DeleteContact(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}
	if err := h.svc.DeleteContact(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}

func (h *Handler) ListSubscribers(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)
	items, err := h.svc.ListSubscribers(ctx, int64(p.Limit), p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}
	nextCursor := int64(0)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}
	return httputil.Paginated(c, items, nextCursor, p.Limit)
}

func (h *Handler) DeleteSubscriber(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid ID")
	}
	if err := h.svc.DeleteSubscriber(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}
