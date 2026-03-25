package httpx

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(ctx context.Context, c *app.RequestContext, status int, payload any) {
	c.JSON(status, payload)
}

// DecodeJSON decodes the request body as JSON into target.
func DecodeJSON(c *app.RequestContext, target any) error {
	if err := c.BindJSON(target); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}
	return nil
}
