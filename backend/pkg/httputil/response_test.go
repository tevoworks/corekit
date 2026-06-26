package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	data := map[string]string{"hello": "world"}
	err := OK(c, data)
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != nil {
		t.Fatalf("expected no error, got: %+v", resp["error"])
	}
	d, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data type unexpected: %T", resp["data"])
	}
	if d["hello"] != "world" {
		t.Fatalf("expected hello=world, got: %v", d["hello"])
	}
	// meta may be omitted by omitempty when empty map is passed
}

func TestOKWithMeta(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	data := []int{1, 2, 3}
	err := OKWithMeta(c, data, map[string]interface{}{"count": 3})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	meta, ok := resp["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta type unexpected: %T", resp["meta"])
	}
	if meta["count"].(float64) != 3 {
		t.Fatalf("expected count=3, got: %v", meta["count"])
	}
}

func TestCreated(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Created(c, map[string]string{"id": "42"}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got: %d", rec.Code)
	}
}

func TestErrorResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = Error(c, http.StatusBadRequest, "BAD_REQUEST", "something wrong")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	errBody, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errBody["code"] != "BAD_REQUEST" || errBody["message"] != "something wrong" {
		t.Fatalf("unexpected error: %+v", errBody)
	}
}

func TestConvenienceErrors(t *testing.T) {
	e := echo.New()
	tests := []struct {
		name     string
		fn       func(echo.Context) error
		wantCode int
	}{
		{"BadRequest", func(c echo.Context) error { return BadRequest(c, "bad") }, 400},
		{"ValidationError", func(c echo.Context) error { return ValidationError(c, "invalid") }, 400},
		{"Unauthorized", func(c echo.Context) error { return Unauthorized(c, "unauth") }, 401},
		{"Forbidden", func(c echo.Context) error { return Forbidden(c, "forbid") }, 403},
		{"NotFound", func(c echo.Context) error { return NotFound(c, "missing") }, 404},
		{"TooManyRequests", func(c echo.Context) error { return TooManyRequests(c, "slow") }, 429},
		{"InternalError", func(c echo.Context) error { return InternalError(c) }, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			_ = tt.fn(c)
			if rec.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d", tt.wantCode, rec.Code)
			}
		})
	}
}

func TestParseCursorPaginationDefaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := ParseCursorPagination(c)
	if p.Limit != 50 {
		t.Fatalf("default limit should be 50, got: %d", p.Limit)
	}
	if p.Cursor != 0 {
		t.Fatalf("default cursor should be 0, got: %d", p.Cursor)
	}
	_ = rec
}

func TestParseCursorPaginationCustom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?limit=10&cursor=100", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := ParseCursorPagination(c)
	if p.Limit != 10 {
		t.Fatalf("expected limit=10, got: %d", p.Limit)
	}
	if p.Cursor != 100 {
		t.Fatalf("expected cursor=100, got: %d", p.Cursor)
	}
	_ = rec
}

func TestParseCursorPaginationClamp(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?limit=500", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := ParseCursorPagination(c)
	if p.Limit > 100 {
		t.Fatalf("limit should be clamped to 100, got: %d", p.Limit)
	}
	_ = rec
}

func TestPaginated(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	data := []string{"a", "b"}
	if err := Paginated(c, data, 42, 10); err != nil {
		t.Fatal(err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	meta, ok := resp["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta type unexpected: %T", resp["meta"])
	}
	if meta["count"].(float64) != 2 {
		t.Fatalf("expected count=2, got: %v", meta["count"])
	}
	if meta["limit"].(float64) != 10 {
		t.Fatalf("expected limit=10, got: %v", meta["limit"])
	}
	if meta["next_cursor"].(float64) != 42 {
		t.Fatalf("expected next_cursor=42, got: %v", meta["next_cursor"])
	}
}

func TestMessage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Message(c, "hello"); err != nil {
		t.Fatal(err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	d, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data type unexpected: %T", resp["data"])
	}
	if d["message"] != "hello" {
		t.Fatalf("expected message=hello, got: %v", d["message"])
	}
}

func TestDeleted(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Deleted(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", rec.Code)
	}
}
