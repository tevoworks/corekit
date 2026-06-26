package cms

import (
	"encoding/json"
	"errors"
	"net/http"
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
	// Public routes
	publicGroup.GET("/pages", h.ListPublishedPages)
	publicGroup.GET("/pages/:slug", h.GetPublishedPage)
	publicGroup.GET("/posts", h.ListPublishedPosts)
	publicGroup.GET("/posts/:slug", h.GetPublishedPost)

	// Admin routes with /cms/ prefix
	g.GET("/cms/pages", h.ListPages, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.POST("/cms/pages", h.CreatePage, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.GET("/cms/pages/:id", h.GetPage, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.PUT("/cms/pages/:id", h.UpdatePage, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.DELETE("/cms/pages/:id", h.DeletePage, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.POST("/cms/pages/:id/publish", h.PublishPage, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.POST("/cms/pages/:id/unpublish", h.UnpublishPage, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))

	g.GET("/cms/posts", h.ListPosts, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.POST("/cms/posts", h.CreatePost, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.GET("/cms/posts/:id", h.GetPost, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.PUT("/cms/posts/:id", h.UpdatePost, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.DELETE("/cms/posts/:id", h.DeletePost, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.POST("/cms/posts/:id/publish", h.PublishPost, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.POST("/cms/posts/:id/unpublish", h.UnpublishPost, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))

	g.GET("/cms/pages/:pageID/sections", h.ListSectionsByPage, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.POST("/cms/pages/:pageID/sections", h.CreateSection, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.GET("/cms/sections/:id", h.GetSection, authMW, middleware.RBACMiddleware(h.rbacService, "read:cms"))
	g.PUT("/cms/sections/:id", h.UpdateSection, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))
	g.DELETE("/cms/sections/:id", h.DeleteSection, authMW, middleware.RBACMiddleware(h.rbacService, "manage:cms"))

	// Slug check
	g.GET("/cms/check-slug", h.CheckSlug, authMW)
}

// Request types

type CreatePageRequest struct {
	Title           string `json:"title" validate:"required,nohtml"`
	Slug            string `json:"slug" validate:"required"`
	Content         string `json:"content"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	OgImage         string `json:"og_image"`
	FeaturedImageID *int64 `json:"featured_image_id"`
}

type UpdatePageRequest struct {
	Title           string `json:"title" validate:"required,nohtml"`
	Slug            string `json:"slug" validate:"required"`
	Content         string `json:"content"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	OgImage         string `json:"og_image"`
	FeaturedImageID *int64 `json:"featured_image_id"`
}

type CreatePostRequest struct {
	Title           string   `json:"title" validate:"required,nohtml"`
	Slug            string   `json:"slug" validate:"required"`
	Content         string   `json:"content"`
	Excerpt         string   `json:"excerpt"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	OgImage         string   `json:"og_image"`
	FeaturedImageID *int64   `json:"featured_image_id"`
	Tags            []string `json:"tags"`
}

type UpdatePostRequest struct {
	Title           string   `json:"title" validate:"required,nohtml"`
	Slug            string   `json:"slug" validate:"required"`
	Content         string   `json:"content"`
	Excerpt         string   `json:"excerpt"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	OgImage         string   `json:"og_image"`
	FeaturedImageID *int64   `json:"featured_image_id"`
	Tags            []string `json:"tags"`
}

type CreateSectionRequest struct {
	Type      string          `json:"type" validate:"required"`
	Title     string          `json:"title"`
	Content   json.RawMessage `json:"content"`
	SortOrder int             `json:"sort_order"`
}

type UpdateSectionRequest struct {
	Type      string          `json:"type" validate:"required"`
	Title     string          `json:"title"`
	Content   json.RawMessage `json:"content"`
	SortOrder int             `json:"sort_order"`
}

// --- Page Handlers ---

func (h *Handler) CreatePage(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreatePageRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	page, err := h.svc.CreatePage(ctx, req.Title, req.Slug, req.Content, req.MetaTitle, req.MetaDescription, req.OgImage, req.FeaturedImageID, actorID)
	if err != nil {
		if errors.Is(err, ErrSlugConflict) {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "Slug already exists")
		}
		return httputil.InternalError(c)
	}
	return httputil.Created(c, page)
}

func (h *Handler) GetPage(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	page, err := h.svc.GetPage(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if page == nil {
		return httputil.NotFound(c, "Page not found")
	}
	return httputil.OK(c, page)
}

func (h *Handler) ListPages(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)
	status := c.QueryParam("status")

	pages, err := h.svc.ListPages(ctx, int64(p.Limit), p.Cursor, status)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(pages) > 0 {
		nextCursor = pages[len(pages)-1].ID
	}
	return httputil.Paginated(c, pages, nextCursor, p.Limit)
}

func (h *Handler) UpdatePage(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	var req UpdatePageRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	page, err := h.svc.UpdatePage(ctx, id, req.Title, req.Slug, req.Content, req.MetaTitle, req.MetaDescription, req.OgImage, req.FeaturedImageID, actorID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.NotFound(c, "Page not found")
		}
		if errors.Is(err, ErrSlugConflict) {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "Slug already exists")
		}
		return httputil.InternalError(c)
	}
	return httputil.OK(c, page)
}

func (h *Handler) DeletePage(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	if err := h.svc.DeletePage(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}

func (h *Handler) PublishPage(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	if err := h.svc.PublishPage(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Page published")
}

func (h *Handler) UnpublishPage(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	if err := h.svc.UnpublishPage(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Page unpublished")
}

// --- Public Page Handlers ---

func (h *Handler) ListPublishedPages(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)

	pages, err := h.svc.ListPublishedPages(ctx, int64(p.Limit), p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(pages) > 0 {
		nextCursor = pages[len(pages)-1].ID
	}
	return httputil.Paginated(c, pages, nextCursor, p.Limit)
}

func (h *Handler) GetPublishedPage(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")

	page, err := h.svc.GetPageBySlug(ctx, slug)
	if err != nil {
		return httputil.InternalError(c)
	}
	if page == nil || page.Status != "published" {
		return httputil.NotFound(c, "Page not found")
	}
	return httputil.OK(c, page)
}

// --- Blog Post Handlers ---

func (h *Handler) CreatePost(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	var req CreatePostRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	post, err := h.svc.CreatePost(ctx, req.Title, req.Slug, req.Content, req.Excerpt, req.MetaTitle, req.MetaDescription, req.OgImage, req.FeaturedImageID, req.Tags, actorID)
	if err != nil {
		if errors.Is(err, ErrSlugConflict) {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "Slug already exists")
		}
		return httputil.InternalError(c)
	}
	return httputil.Created(c, post)
}

func (h *Handler) GetPost(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid post ID")
	}

	post, err := h.svc.GetPost(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if post == nil {
		return httputil.NotFound(c, "Post not found")
	}
	return httputil.OK(c, post)
}

func (h *Handler) ListPosts(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)
	status := c.QueryParam("status")

	posts, err := h.svc.ListPosts(ctx, int64(p.Limit), p.Cursor, status)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(posts) > 0 {
		nextCursor = posts[len(posts)-1].ID
	}
	return httputil.Paginated(c, posts, nextCursor, p.Limit)
}

func (h *Handler) UpdatePost(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid post ID")
	}

	var req UpdatePostRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	post, err := h.svc.UpdatePost(ctx, id, req.Title, req.Slug, req.Content, req.Excerpt, req.MetaTitle, req.MetaDescription, req.OgImage, req.FeaturedImageID, req.Tags, actorID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.NotFound(c, "Post not found")
		}
		if errors.Is(err, ErrSlugConflict) {
			return httputil.Error(c, http.StatusConflict, "CONFLICT", "Slug already exists")
		}
		return httputil.InternalError(c)
	}
	return httputil.OK(c, post)
}

func (h *Handler) DeletePost(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid post ID")
	}

	if err := h.svc.DeletePost(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}

func (h *Handler) PublishPost(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid post ID")
	}

	if err := h.svc.PublishPost(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Post published")
}

func (h *Handler) UnpublishPost(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid post ID")
	}

	if err := h.svc.UnpublishPost(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Message(c, "Post unpublished")
}

// --- Public Post Handlers ---

func (h *Handler) ListPublishedPosts(c echo.Context) error {
	ctx := c.Request().Context()
	p := httputil.ParseCursorPagination(c)

	posts, err := h.svc.ListPublishedPosts(ctx, int64(p.Limit), p.Cursor)
	if err != nil {
		return httputil.InternalError(c)
	}

	nextCursor := int64(0)
	if len(posts) > 0 {
		nextCursor = posts[len(posts)-1].ID
	}
	return httputil.Paginated(c, posts, nextCursor, p.Limit)
}

func (h *Handler) GetPublishedPost(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")

	post, err := h.svc.GetPostBySlug(ctx, slug)
	if err != nil {
		return httputil.InternalError(c)
	}
	if post == nil || post.Status != "published" {
		return httputil.NotFound(c, "Post not found")
	}
	return httputil.OK(c, post)
}

// --- Section Handlers ---

func (h *Handler) CreateSection(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	pageID, err := strconv.ParseInt(c.Param("pageID"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	var req CreateSectionRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	section, err := h.svc.CreateSection(ctx, pageID, req.Type, req.Title, req.Content, req.SortOrder, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Created(c, section)
}

func (h *Handler) GetSection(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid section ID")
	}

	section, err := h.svc.GetSection(ctx, id)
	if err != nil {
		return httputil.InternalError(c)
	}
	if section == nil {
		return httputil.NotFound(c, "Section not found")
	}
	return httputil.OK(c, section)
}

func (h *Handler) ListSectionsByPage(c echo.Context) error {
	ctx := c.Request().Context()
	pageID, err := strconv.ParseInt(c.Param("pageID"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid page ID")
	}

	sections, err := h.svc.ListSectionsByPage(ctx, pageID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, sections)
}

func (h *Handler) UpdateSection(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid section ID")
	}

	var req UpdateSectionRequest
	if err := validation.BindAndValidate(c, &req); err != nil {
		return err
	}

	section, err := h.svc.UpdateSection(ctx, id, 0, req.Type, req.Title, req.Content, req.SortOrder, actorID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.NotFound(c, "Section not found")
		}
		return httputil.InternalError(c)
	}
	return httputil.OK(c, section)
}

func (h *Handler) DeleteSection(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid section ID")
	}

	if err := h.svc.DeleteSection(ctx, id, actorID); err != nil {
		return httputil.InternalError(c)
	}
	return httputil.Deleted(c)
}

// --- Slug Check ---

func (h *Handler) CheckSlug(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.QueryParam("slug")
	excludeID, _ := strconv.ParseInt(c.QueryParam("exclude_id"), 10, 64)

	if slug == "" {
		return httputil.BadRequest(c, "slug parameter is required")
	}

	exists, err := h.svc.CheckSlugExists(ctx, slug, excludeID)
	if err != nil {
		return httputil.InternalError(c)
	}
	return httputil.OK(c, SlugCheckResult{Available: !exists, Slug: slug})
}
