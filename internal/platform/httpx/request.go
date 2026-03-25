package httpx

import (
	"fmt"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

// PathInt64 parses a path parameter as int64.
func PathInt64(c *app.RequestContext, key string) (int64, error) {
	raw := c.Param(key)
	val, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return val, nil
}
