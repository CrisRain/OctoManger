package httpx

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"

	"octomanger/internal/testutil"
)

func TestDecodeJSONInvalid(t *testing.T) {
	ctx := &app.RequestContext{}
	ctx.Request.SetBody([]byte("not-json"))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	var payload map[string]any
	if err := DecodeJSON(ctx, &payload); err != ErrInvalidJSONBody {
		t.Fatalf("expected invalid json error, got %v", err)
	}
}

func TestDecodeJSONValid(t *testing.T) {
	ctx := testutil.NewJSONRequestContext("POST", "/", map[string]any{"a": 1})
	var payload struct {
		A int `json:"a"`
	}
	if err := DecodeJSON(ctx, &payload); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}
	if payload.A != 1 {
		t.Fatalf("unexpected payload %+v", payload)
	}
}

func TestPathInt64(t *testing.T) {
	ctx := &app.RequestContext{}
	testutil.SetPathParam(ctx, "id", "42")
	id, err := PathInt64(ctx, "id")
	if err != nil || id != 42 {
		t.Fatalf("unexpected id %d err=%v", id, err)
	}

	ctx = &app.RequestContext{}
	testutil.SetPathParam(ctx, "id", "bad")
	if _, err := PathInt64(ctx, "id"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestSanitizeClientDetailExtraCases(t *testing.T) {
	if got := sanitizeClientDetail(""); got != "invalid request" {
		t.Fatalf("expected default detail, got %q", got)
	}
	if got := sanitizeClientDetail("SELECT * FROM users"); got != "invalid request" {
		t.Fatalf("expected sql detail to be sanitized")
	}
	long := bytes.Repeat([]byte("a"), 300)
	if got := sanitizeClientDetail(string(long)); got != "invalid request" {
		t.Fatalf("expected long detail to be sanitized")
	}
}

func TestStreamWriterAndPrepareStream(t *testing.T) {
	ctx := &app.RequestContext{}

	PrepareStream(ctx, func(w *StreamWriter) {
		if err := w.WriteEvent("tick", map[string]any{"value": 1}); err != nil {
			t.Fatalf("write event: %v", err)
		}
	})

	body, err := ctx.Response.BodyE()
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(body), []byte("\n"))
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var payload map[string]any
	if err := json.Unmarshal(lines[0], &payload); err != nil {
		t.Fatalf("decode stream payload: %v", err)
	}
	if payload["event"] != "tick" {
		t.Fatalf("unexpected event payload %#v", payload)
	}

	ctx = &app.RequestContext{}
	PrepareSSE(ctx, func(w *SSEWriter) {
		_ = w.WriteEvent("pong", map[string]any{"ok": true})
	})
	if _, err := ctx.Response.BodyE(); err != nil {
		t.Fatalf("read sse body: %v", err)
	}
}
