package targetprocess

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lifedraft/targetprocess-cli/internal/api"
)

// resourceType extracts the TP resource type from the zero value of a Typed type.
func resourceType[T Typed]() string {
	var zero T
	return zero.TPResourceType()
}

// unmarshalEntity converts an Entity (map[string]any) to a typed struct.
// Go's encoding/json does case-insensitive key matching, so PascalCase keys
// from the v1 API match struct tags regardless of casing.
func unmarshalEntity[T any](raw Entity) (*T, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("re-encoding entity: %w", err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding into %T: %w", result, err)
	}
	return &result, nil
}

// Get retrieves a single entity by ID with full type safety.
// The entity type is inferred from T (e.g., Get[UserStory] hits /api/v1/UserStorys/{id}).
// Optional include fields control which nested objects are returned.
func Get[T Typed](ctx context.Context, c *Client, id int, include ...string) (*T, error) {
	rt := resourceType[T]()
	raw, err := c.internal.GetEntity(ctx, rt, id, include)
	if err != nil {
		return nil, err
	}
	return unmarshalEntity[T](raw)
}

// Search queries entities of type T using the v2 API.
func Search[T Typed](ctx context.Context, c *Client, where string, opts ...SearchOption) ([]T, error) {
	rt := resourceType[T]()
	so := resolveSearchOpts(opts)
	params := api.V2Params{
		Where:   where,
		Select:  so.selectExpr,
		OrderBy: so.orderBy,
		Take:    so.take,
	}
	data, err := c.internal.QueryV2(ctx, rt, params)
	if err != nil {
		return nil, err
	}
	var result Result[T]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}
	return result.Items, nil
}

// Create creates a new entity of type T and returns the typed result.
func Create[T Typed](ctx context.Context, c *Client, fields Entity) (*T, error) {
	rt := resourceType[T]()
	raw, err := c.internal.CreateEntity(ctx, rt, fields)
	if err != nil {
		return nil, err
	}
	return unmarshalEntity[T](raw)
}

// Update modifies an existing entity of type T and returns the typed result.
func Update[T Typed](ctx context.Context, c *Client, id int, fields Entity) (*T, error) {
	rt := resourceType[T]()
	raw, err := c.internal.UpdateEntity(ctx, rt, id, fields)
	if err != nil {
		return nil, err
	}
	return unmarshalEntity[T](raw)
}

// Query executes a v2 query and deserializes items into the provided type T.
// T can be any struct — use struct tags matching the v2 select expression.
// Unlike other generic functions, entityType must be passed explicitly because
// T may be a user-defined struct that does not implement Typed.
//
// Example:
//
//	type SprintItem struct {
//	    ID    int    `json:"id"`
//	    Name  string `json:"name"`
//	    State string `json:"state"`
//	}
//	result, err := targetprocess.Query[SprintItem](ctx, c, "UserStory", targetprocess.QueryParams{
//	    Where:  "TeamIteration.Name=='Sprint 42'",
//	    Select: "id,name,entityState.name as state",
//	})
func Query[T any](ctx context.Context, c *Client, entityType string, params QueryParams) (*Result[T], error) {
	data, err := c.internal.QueryV2(ctx, entityType, api.V2Params{
		Where:   params.Where,
		Select:  params.Select,
		OrderBy: params.OrderBy,
		Take:    params.Take,
		Skip:    params.Skip,
	})
	if err != nil {
		return nil, err
	}
	var result Result[T]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing v2 response: %w", err)
	}
	return &result, nil
}
