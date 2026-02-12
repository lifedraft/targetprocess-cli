package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// Entity represents a generic TP entity as a flexible map.
type Entity = map[string]any

// APIError represents an error response from the TP API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Client is the Targetprocess API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Debug      bool
}

// NewClient creates a new API client with retry support.
func NewClient(baseURL, token string, debug bool) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.Logger = nil

	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: rc.StandardClient(),
		Debug:      debug,
	}
}

func (c *Client) buildURL(path string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", c.Token)
	params.Set("format", "json")
	return fmt.Sprintf("%s%s?%s", c.BaseURL, path, params.Encode())
}

func (c *Client) request(ctx context.Context, method, fullURL string, body io.Reader) ([]byte, error) {
	if c.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: %s %s\n", method, fullURL)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "tp-cli/0.1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if c.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: HTTP %d, %d bytes\n", resp.StatusCode, len(data))
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(data)}
	}
	return data, nil
}

func (c *Client) do(ctx context.Context, method, path string, params url.Values, body io.Reader) ([]byte, error) {
	return c.request(ctx, method, c.buildURL(path, params), body)
}

// SearchEntities searches for entities of the given type.
func (c *Client) SearchEntities(ctx context.Context, entityType, where string, include []string, take int, orderBy []string) ([]Entity, error) {
	params := url.Values{}
	if where != "" {
		params.Set("where", where)
	}
	if len(include) > 0 {
		params.Set("include", "["+strings.Join(include, ",")+"]")
	}
	if take > 0 {
		params.Set("take", strconv.Itoa(take))
	}
	if len(orderBy) > 0 {
		params.Set("orderBy", strings.Join(orderBy, ","))
	}

	path := fmt.Sprintf("/api/v1/%ss", entityType)
	data, err := c.do(ctx, http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []Entity `json:"Items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return resp.Items, nil
}

// GetEntity gets a single entity by type and ID.
func (c *Client) GetEntity(ctx context.Context, entityType string, id int, include []string) (Entity, error) {
	params := url.Values{}
	if len(include) > 0 {
		params.Set("include", "["+strings.Join(include, ",")+"]")
	}

	path := fmt.Sprintf("/api/v1/%ss/%d", entityType, id)
	data, err := c.do(ctx, http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entity, nil
}

// CreateEntity creates a new entity. Fields are sent as the JSON body.
func (c *Client) CreateEntity(ctx context.Context, entityType string, fields map[string]any) (Entity, error) {
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	path := fmt.Sprintf("/api/v1/%ss", entityType)
	data, err := c.do(ctx, http.MethodPost, path, nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entity, nil
}

// UpdateEntity updates an existing entity. TP uses POST for updates.
func (c *Client) UpdateEntity(ctx context.Context, entityType string, id int, fields map[string]any) (Entity, error) {
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	path := fmt.Sprintf("/api/v1/%ss/%d", entityType, id)
	data, err := c.do(ctx, http.MethodPost, path, nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entity, nil
}

// GetMetaIndex fetches the metadata index (list of all entity types) as XML.
func (c *Client) GetMetaIndex(ctx context.Context) ([]byte, error) {
	params := url.Values{}
	params.Set("access_token", c.Token)
	fullURL := fmt.Sprintf("%s/api/v1/Index/meta?%s", c.BaseURL, params.Encode())
	return c.request(ctx, http.MethodGet, fullURL, nil)
}

// GetTypeMeta fetches metadata for a specific entity type as XML.
func (c *Client) GetTypeMeta(ctx context.Context, entityType string) ([]byte, error) {
	params := url.Values{}
	params.Set("access_token", c.Token)
	fullURL := fmt.Sprintf("%s/api/v1/%ss/meta?%s", c.BaseURL, entityType, params.Encode())
	return c.request(ctx, http.MethodGet, fullURL, nil)
}

// Raw makes a raw API request. The path can include query parameters.
func (c *Client) Raw(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	q := u.Query()
	q.Set("access_token", c.Token)
	u.RawQuery = q.Encode()
	return c.request(ctx, method, u.String(), body)
}
