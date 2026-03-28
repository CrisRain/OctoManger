package httpx

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	DefaultPageSize = 100
	MaxPageSize     = 500
)

type PageRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Limit    int `json:"limit"`
	Offset   int `json:"offset"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func ParsePageRequest(c *app.RequestContext) (PageRequest, error) {
	request := PageRequest{
		Page:     1,
		PageSize: DefaultPageSize,
		Limit:    DefaultPageSize,
		Offset:   0,
	}

	page, hasPage, err := queryInt(c, "page")
	if err != nil {
		return PageRequest{}, err
	}
	pageSize, hasPageSize, err := queryInt(c, "page_size")
	if err != nil {
		return PageRequest{}, err
	}
	limit, hasLimit, err := queryInt(c, "limit")
	if err != nil {
		return PageRequest{}, err
	}
	offset, hasOffset, err := queryInt(c, "offset")
	if err != nil {
		return PageRequest{}, err
	}

	if hasPage || hasPageSize {
		if hasPage {
			if page <= 0 {
				return PageRequest{}, fmt.Errorf("page must be greater than 0")
			}
			request.Page = page
		}
		if hasPageSize {
			if pageSize <= 0 {
				return PageRequest{}, fmt.Errorf("page_size must be greater than 0")
			}
			if pageSize > MaxPageSize {
				pageSize = MaxPageSize
			}
			request.PageSize = pageSize
		}
		request.Limit = request.PageSize
		request.Offset = (request.Page - 1) * request.PageSize
		return request, nil
	}

	if hasLimit {
		if limit <= 0 {
			return PageRequest{}, fmt.Errorf("limit must be greater than 0")
		}
		if limit > MaxPageSize {
			limit = MaxPageSize
		}
		request.Limit = limit
		request.PageSize = limit
	}
	if hasOffset {
		if offset < 0 {
			return PageRequest{}, fmt.Errorf("offset must be greater than or equal to 0")
		}
		request.Offset = offset
	}
	if request.PageSize > 0 {
		request.Page = (request.Offset / request.PageSize) + 1
	}
	return request, nil
}

func BuildPageMeta(request PageRequest, total int64) PageMeta {
	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	page := request.Page
	if page <= 0 {
		page = 1
	}

	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = int64(math.Ceil(float64(total) / float64(pageSize)))
	}

	return PageMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func queryInt(c *app.RequestContext, key string) (int, bool, error) {
	raw, ok := c.GetQuery(key)
	if !ok {
		return 0, false, nil
	}
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, false, fmt.Errorf("invalid %s", key)
	}
	return value, true, nil
}
