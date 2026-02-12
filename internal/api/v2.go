package api //nolint:revive // package name "api" is intentional

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// V2Params holds the query parameters for a v2 API request.
type V2Params struct {
	Where   string
	Select  string
	OrderBy string
	Take    int
	Skip    int
}

// BuildV2URL constructs the full v2 URL without executing the request.
// Useful for --dry-run to inspect the URL that would be called.
func (c *Client) BuildV2URL(entityType string, params V2Params) string {
	path := fmt.Sprintf("/api/v2/%s", entityType)

	q := url.Values{}
	q.Set("access_token", c.Token)
	if params.Where != "" {
		q.Set("where", params.Where)
	}
	if params.Select != "" {
		q.Set("select", "{"+params.Select+"}")
	}
	if params.OrderBy != "" {
		q.Set("orderBy", params.OrderBy)
	}
	if params.Take > 0 {
		q.Set("take", strconv.Itoa(params.Take))
	}
	if params.Skip > 0 {
		q.Set("skip", strconv.Itoa(params.Skip))
	}

	return fmt.Sprintf("%s%s?%s", c.BaseURL, path, q.Encode())
}

// QueryV2 executes a v2 API query and returns raw JSON bytes.
// entityType is singular (e.g., "UserStory", "Assignable").
func (c *Client) QueryV2(ctx context.Context, entityType string, params V2Params) ([]byte, error) {
	fullURL := c.BuildV2URL(entityType, params)
	return c.request(ctx, http.MethodGet, fullURL, nil)
}

// BuildV2EntityURL constructs the full v2 URL for a single entity by ID.
func (c *Client) BuildV2EntityURL(entityType string, id int, selectExpr string) string {
	path := fmt.Sprintf("/api/v2/%s/%d", entityType, id)

	q := url.Values{}
	q.Set("access_token", c.Token)
	if selectExpr != "" {
		q.Set("select", "{"+selectExpr+"}")
	}

	return fmt.Sprintf("%s%s?%s", c.BaseURL, path, q.Encode())
}

// QueryV2Entity gets a single entity by ID via v2.
func (c *Client) QueryV2Entity(ctx context.Context, entityType string, id int, selectExpr string) ([]byte, error) {
	fullURL := c.BuildV2EntityURL(entityType, id, selectExpr)
	return c.request(ctx, http.MethodGet, fullURL, nil)
}
