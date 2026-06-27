package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/rbac"
	"github.com/tevoworks/corekit/backend/internal/modules/settings"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

func urlCleanFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "&", "")
	name = strings.ReplaceAll(name, "%", "")
	name = strings.ReplaceAll(name, "+", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	if name == "" {
		return "file"
	}
	return name
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, ";", "")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", "")
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return "untitled"
	}
	return name
}

const defaultMaxUploadMB = 10

type Handler struct {
	service         Service
	rbacService     rbac.Service
	settingsService settings.Service
}

func NewHandler(s Service, rbacService rbac.Service, settingsService settings.Service) *Handler {
	return &Handler{
		service:         s,
		rbacService:     rbacService,
		settingsService: settingsService,
	}
}

func (h *Handler) RegisterRoutes(authGroup *echo.Group, publicGroup *echo.Group, authMiddleware echo.MiddlewareFunc) {
	authGroup.POST("/storage/upload", h.Upload, authMiddleware, middleware.LimitIP(10), middleware.RBACMiddleware(h.rbacService, "write:files"))
	authGroup.GET("/storage/files", h.List, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:files"))
	authGroup.GET("/storage/files/:id", h.Download, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:files"))
	authGroup.GET("/storage/files/:id/:filename", h.Download, authMiddleware, middleware.RBACMiddleware(h.rbacService, "read:files"))
	authGroup.DELETE("/storage/files/:id", h.Delete, authMiddleware, middleware.RBACMiddleware(h.rbacService, "delete:files"))

	publicGroup.GET("/storage/files/:id", h.DownloadPublic)
	publicGroup.GET("/storage/files/:id/:filename", h.DownloadPublic)
}

func (h *Handler) getMaxUploadBytes() int64 {
	maxMB := int64(defaultMaxUploadMB)
	if h.settingsService != nil {
		ctx := context.Background()
		s, err := h.settingsService.GetSetting(ctx, "max_upload_size_mb")
		if err == nil && s != nil && s.Value != "" {
			if val, err := strconv.ParseInt(s.Value, 10, 64); err == nil && val > 0 {
				maxMB = val
			}
		}
	}
	return maxMB << 20
}

func (h *Handler) Upload(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	maxBytes := h.getMaxUploadBytes()
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxBytes)

	file, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || (err != nil && err.Error() == "http: request body too large") {
			maxMB := maxBytes >> 20
			return httputil.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "The uploaded file exceeds the "+strconv.FormatInt(maxMB, 10)+"MB limit")
		}
		return httputil.BadRequest(c, "Missing file in upload request")
	}

	src, err := file.Open()
	if err != nil {
		return httputil.InternalError(c)
	}
	defer src.Close()

	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return httputil.InternalError(c)
	}
	detectedContentType := http.DetectContentType(buffer[:n])
	if !isAllowedMIMEType(detectedContentType) {
		return httputil.BadRequest(c, "File type "+detectedContentType+" is not allowed")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isAllowedExtension(ext) {
		return httputil.BadRequest(c, "File extension "+ext+" is not allowed")
	}

	var reader io.Reader = src
	if seeker, ok := src.(io.Seeker); ok {
		_, err = seeker.Seek(0, io.SeekStart)
		if err != nil {
			return httputil.InternalError(c)
		}
	} else {
		reader = io.MultiReader(bytes.NewReader(buffer[:n]), src)
	}

	isPublic := c.FormValue("is_public") == "true"
	meta, err := h.service.UploadFile(ctx, file.Filename, file.Size, detectedContentType, reader, actorID, isPublic)
	if err != nil {
		slog.Error("upload file failed", "filename", file.Filename, "error", err)
		return httputil.InternalError(c)
	}

	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	meta.URL = fmt.Sprintf("%s://%s/api/storage/files/%d/%s", scheme, c.Request().Host, meta.ID, urlCleanFilename(meta.Filename))

	return httputil.Created(c, meta)
}

func (h *Handler) Download(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid file ID")
	}

	isSuperAdmin := middleware.IsSuperAdmin(c)
	meta, fileReader, err := h.service.DownloadFile(ctx, id, actorID, isSuperAdmin)
	if err != nil {
		return httputil.NotFound(c, "File not found or access denied")
	}
	defer fileReader.Close()

	safeName := sanitizeFilename(meta.Filename)
	isDangerous := forceAttachment(meta.MIMEType)
	contentType := meta.MIMEType
	contentDisposition := "inline; filename=" + safeName
	if isDangerous {
		contentDisposition = "attachment; filename=" + safeName
		contentType = "application/octet-stream"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, contentDisposition)
	c.Response().Header().Set(echo.HeaderContentType, contentType)
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(meta.SizeBytes, 10))
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	if meta.ChecksumSHA256 != "" {
		c.Response().Header().Set("ETag", "\""+meta.ChecksumSHA256+"\"")
	}

	return c.Stream(http.StatusOK, contentType, fileReader)
}

func (h *Handler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid file ID")
	}

	err = h.service.DeleteFile(ctx, id, actorID)
	if err != nil {
		return httputil.InternalError(c)
	}

	return httputil.Message(c, "File successfully deleted")
}

func (h *Handler) List(c echo.Context) error {
	ctx := c.Request().Context()
	actorID := middleware.GetUserID(c)

	limit := 50
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

	isSuperAdmin := middleware.IsSuperAdmin(c)
	files, err := h.service.ListFiles(ctx, limit, cursor, actorID, isSuperAdmin)
	if err != nil {
		return httputil.InternalError(c)
	}

	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request().Host)
	for i := range files {
		files[i].URL = fmt.Sprintf("%s/api/storage/files/%d/%s", baseURL, files[i].ID, urlCleanFilename(files[i].Filename))
	}

	nextCursor := int64(0)
	if len(files) > 0 {
		nextCursor = files[len(files)-1].ID
	}

	return httputil.OKWithMeta(c, files, map[string]interface{}{
		"count":       len(files),
		"limit":       limit,
		"cursor":      cursor,
		"next_cursor": nextCursor,
	})
}

var dangerousMIMETypes = map[string]bool{
	"text/html":                   true,
	"image/svg+xml":               true,
	"text/xml":                    true,
	"application/xml":             true,
	"application/xhtml+xml":       true,
	"application/javascript":      true,
	"text/javascript":             true,
	"text/vnd.wap.wml":            true,
	"application/x-msdownload":    true,
	"application/x-msdos-program": true,
	"application/x-executable":    true,
	"application/x-sh":            true,
	"application/x-csh":           true,
}

func forceAttachment(mimeType string) bool {
	return dangerousMIMETypes[mimeType]
}

func (h *Handler) DownloadPublic(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return httputil.BadRequest(c, "Invalid file ID")
	}

	meta, fileReader, err := h.service.DownloadPublicFile(ctx, id)
	if err != nil {
		return httputil.NotFound(c, "File not found or access denied")
	}
	defer fileReader.Close()

	safeName := sanitizeFilename(meta.Filename)
	isDangerous := forceAttachment(meta.MIMEType)
	contentType := meta.MIMEType
	contentDisposition := "inline; filename=" + safeName
	if isDangerous {
		contentDisposition = "attachment; filename=" + safeName
		contentType = "application/octet-stream"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, contentDisposition)
	c.Response().Header().Set(echo.HeaderContentType, contentType)
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(meta.SizeBytes, 10))
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	if meta.ChecksumSHA256 != "" {
		c.Response().Header().Set("ETag", "\""+meta.ChecksumSHA256+"\"")
	}

	return c.Stream(http.StatusOK, contentType, fileReader)
}

var allowedMIMETypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"image/webp":         true,
	"application/pdf":    true,
	"text/plain":         true,
	"application/zip":    true,
	"application/gzip":   true,
	"application/json":   true,
	"text/csv":           true,
	"application/x-yaml": true,
}

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".pdf":  true,
	".txt":  true,
	".zip":  true,
	".gz":   true,
	".tgz":  true,
	".json": true,
	".csv":  true,
	".yaml": true,
	".yml":  true,
}

func isAllowedExtension(ext string) bool {
	return allowedExtensions[ext]
}

func isAllowedMIMEType(mime string) bool {
	if allowedMIMETypes[mime] {
		return true
	}
	if idx := strings.Index(mime, ";"); idx > 0 {
		base := strings.TrimSpace(mime[:idx])
		return allowedMIMETypes[base]
	}
	return false
}
