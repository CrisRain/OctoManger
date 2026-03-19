package httpx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/app"
)

// SSEWriter writes Server-Sent Events to an underlying writer.
type SSEWriter struct {
	w *bufio.Writer
}

// WriteEvent marshals payload and sends a named SSE event.
func (s *SSEWriter) WriteEvent(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if event != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	return s.w.Flush()
}

// PrepareSSE sets SSE headers on the Hertz response and calls fn in a goroutine.
// The response is streamed via io.Pipe so Hertz can serve it as a body stream.
func PrepareSSE(c *app.RequestContext, fn func(w *SSEWriter)) {
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	pr, pw := io.Pipe()

	go func() {
		bw := bufio.NewWriter(pw)
		fn(&SSEWriter{w: bw})
		_ = bw.Flush()
		pw.Close()
	}()

	c.SetBodyStream(pr, -1)
}
