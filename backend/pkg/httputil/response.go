package httputil

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/labstack/echo/v4"
)

type envelope struct {
	Data  interface{}            `json:"data,omitempty"`
	Meta  map[string]interface{} `json:"meta,omitempty"`
	Error *errorBody             `json:"error,omitempty"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, envelope{Data: data, Meta: map[string]interface{}{}})
}

func OKWithMeta(c echo.Context, data interface{}, meta map[string]interface{}) error {
	if meta == nil {
		meta = map[string]interface{}{}
	}
	return c.JSON(http.StatusOK, envelope{Data: data, Meta: meta})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, envelope{Data: data, Meta: map[string]interface{}{}})
}

func Deleted(c echo.Context) error {
	return c.JSON(http.StatusOK, envelope{
		Data: map[string]string{"message": "Deleted"},
		Meta: map[string]interface{}{},
	})
}

func Message(c echo.Context, msg string) error {
	return c.JSON(http.StatusOK, envelope{
		Data: map[string]string{"message": msg},
		Meta: map[string]interface{}{},
	})
}

func Error(c echo.Context, httpStatus int, code, message string) error {
	return c.JSON(httpStatus, envelope{
		Error: &errorBody{Code: code, Message: message},
	})
}

func BadRequest(c echo.Context, message string) error {
	return Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

func ValidationError(c echo.Context, message string) error {
	return Error(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func Unauthorized(c echo.Context, message string) error {
	return Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(c echo.Context, message string) error {
	return Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(c echo.Context, message string) error {
	return Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

func TooManyRequests(c echo.Context, message string) error {
	return Error(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", message)
}

func InternalError(c echo.Context) error {
	return Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
}

type CursorPagination struct {
	Limit  int
	Cursor int64
}

func ParseCursorPagination(c echo.Context) CursorPagination {
	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val >= 1 && val <= 100 {
			limit = val
		}
	}
	cursor := int64(0)
	if cs := c.QueryParam("cursor"); cs != "" {
		if val, err := strconv.ParseInt(cs, 10, 64); err == nil && val >= 0 {
			cursor = val
		}
	}
	return CursorPagination{Limit: limit, Cursor: cursor}
}

func Paginated(c echo.Context, data interface{}, nextCursor int64, limit int) error {
	count := 0
	if v := reflect.ValueOf(data); v.Kind() == reflect.Slice {
		count = v.Len()
	}
	return c.JSON(http.StatusOK, envelope{
		Data: data,
		Meta: map[string]interface{}{
			"count":       count,
			"limit":       limit,
			"next_cursor": nextCursor,
		},
	})
}
