package targetprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lifedraft/targetprocess-cli/internal/api"
)

// Client is the Targetprocess API client.
type Client struct {
	internal *api.Client
}

// Option configures a Client.
type Option func(*clientConfig)

type clientConfig struct {
	httpClient *http.Client
	debug      bool
}

// WithHTTPClient sets a custom HTTP client for the API client.
func WithHTTPClient(c *http.Client) Option {
	return func(cfg *clientConfig) {
		cfg.httpClient = c
	}
}

// WithDebug enables debug logging to stderr.
func WithDebug(debug bool) Option {
	return func(cfg *clientConfig) {
		cfg.debug = debug
	}
}

// NewClient creates a new Targetprocess API client.
// The domain should be your Targetprocess subdomain (e.g., "yourcompany.tpondemand.com").
func NewClient(domain, token string, opts ...Option) (*Client, error) {
	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}

	ic := api.NewClient(domain, token, cfg.debug)
	if cfg.httpClient != nil {
		ic.HTTPClient = cfg.httpClient
	}

	return &Client{internal: ic}, nil
}

// Get retrieves a single entity by type and ID.
// Optional include fields control which nested objects are returned (v1 API).
func (c *Client) Get(ctx context.Context, entityType string, id int, include ...string) (Entity, error) {
	return c.internal.GetEntity(ctx, entityType, id, include)
}

// Search queries entities using the v2 API and returns untyped results.
func (c *Client) Search(ctx context.Context, entityType, where string, opts ...SearchOption) ([]Entity, error) {
	so := resolveSearchOpts(opts)
	params := api.V2Params{
		Where:   where,
		Select:  so.selectExpr,
		OrderBy: so.orderBy,
		Take:    so.take,
	}
	data, err := c.internal.QueryV2(ctx, entityType, params)
	if err != nil {
		return nil, err
	}
	var result Result[Entity]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}
	return result.Items, nil
}

// Create creates a new entity of the given type.
func (c *Client) Create(ctx context.Context, entityType string, fields Entity) (Entity, error) {
	return c.internal.CreateEntity(ctx, entityType, fields)
}

// Update modifies an existing entity.
func (c *Client) Update(ctx context.Context, entityType string, id int, fields Entity) (Entity, error) {
	return c.internal.UpdateEntity(ctx, entityType, id, fields)
}

// Delete removes an entity by type and ID.
func (c *Client) Delete(ctx context.Context, entityType string, id int) error {
	_, err := c.internal.DeleteEntity(ctx, entityType, id)
	return err
}

// Query executes a v2 API query and returns raw JSON bytes.
func (c *Client) Query(ctx context.Context, entityType string, params QueryParams) (json.RawMessage, error) {
	return c.internal.QueryV2(ctx, entityType, api.V2Params{
		Where:   params.Where,
		Select:  params.Select,
		OrderBy: params.OrderBy,
		Take:    params.Take,
		Skip:    params.Skip,
	})
}

// QueryEntity queries a single entity by ID via the v2 API.
func (c *Client) QueryEntity(ctx context.Context, entityType string, id int, selectExpr string) (json.RawMessage, error) {
	return c.internal.QueryV2Entity(ctx, entityType, id, selectExpr)
}

// ResolveType determines the entity type for a given ID via the API.
func (c *Client) ResolveType(ctx context.Context, id int) (string, error) {
	return c.internal.ResolveEntityType(ctx, id)
}

// MetaTypes returns all entity types available in the Targetprocess instance.
func (c *Client) MetaTypes(ctx context.Context) ([]TypeInfo, error) {
	data, err := c.internal.GetMetaIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata index: %w", err)
	}
	return parseMetaIndex(data)
}

// MetaFields returns the fields/properties of an entity type.
func (c *Client) MetaFields(ctx context.Context, entityType string) ([]FieldInfo, error) {
	data, err := c.internal.GetTypeMeta(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("fetching type metadata: %w", err)
	}
	return parseTypeMeta(data)
}

// Raw makes a raw API request. The path should start with / and can include
// query parameters. This is an escape hatch for endpoints not covered by
// other methods.
func (c *Client) Raw(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	return c.internal.Raw(ctx, method, path, body)
}
