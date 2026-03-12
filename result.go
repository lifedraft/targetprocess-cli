package targetprocess

// Result wraps a v2 API response containing a paginated collection of items.
type Result[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next,omitempty"`
}
