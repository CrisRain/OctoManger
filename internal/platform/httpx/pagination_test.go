package httpx

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestParsePageRequestDefaults(t *testing.T) {
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts")

	page, err := ParsePageRequest(c)
	if err != nil {
		t.Fatalf("parse page request: %v", err)
	}
	if page.Page != 1 || page.PageSize != DefaultPageSize || page.Limit != DefaultPageSize || page.Offset != 0 {
		t.Fatalf("unexpected default page request: %#v", page)
	}
}

func TestParsePageRequestWithPageAndSize(t *testing.T) {
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts?page=3&page_size=20")

	page, err := ParsePageRequest(c)
	if err != nil {
		t.Fatalf("parse page request: %v", err)
	}
	if page.Page != 3 || page.PageSize != 20 || page.Limit != 20 || page.Offset != 40 {
		t.Fatalf("unexpected page request: %#v", page)
	}
}

func TestParsePageRequestWithLimitOffset(t *testing.T) {
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts?limit=999&offset=20")

	page, err := ParsePageRequest(c)
	if err != nil {
		t.Fatalf("parse page request: %v", err)
	}
	if page.Limit != MaxPageSize || page.Offset != 20 {
		t.Fatalf("unexpected limit/offset page: %#v", page)
	}
}

func TestParsePageRequestInvalidValues(t *testing.T) {
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts?page=0")
	if _, err := ParsePageRequest(c); err == nil {
		t.Fatalf("expected error for invalid page")
	}

	c = &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts?offset=-1")
	if _, err := ParsePageRequest(c); err == nil {
		t.Fatalf("expected error for invalid offset")
	}

	c = &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts?limit=bad")
	if _, err := ParsePageRequest(c); err == nil {
		t.Fatalf("expected error for invalid limit")
	}
}

func TestBuildPageMeta(t *testing.T) {
	meta := BuildPageMeta(PageRequest{Page: 2, PageSize: 25}, 101)
	if meta.Page != 2 || meta.PageSize != 25 || meta.Total != 101 || meta.TotalPages != 5 {
		t.Fatalf("unexpected pagination meta: %#v", meta)
	}
}

func TestWriteJSONContextCompatibility(t *testing.T) {
	// Guard to ensure helper can be called with a real context value.
	ctx := context.Background()
	c := &app.RequestContext{}
	WriteJSON(ctx, c, 200, map[string]any{"ok": true})
	if c.Response.StatusCode() != 200 {
		t.Fatalf("unexpected status code: %d", c.Response.StatusCode())
	}
}
