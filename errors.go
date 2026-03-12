package targetprocess

import (
	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/resolve"
)

// APIError represents an HTTP error response from the Targetprocess API.
// Use errors.As to extract it from returned errors:
//
//	var apiErr *targetprocess.APIError
//	if errors.As(err, &apiErr) {
//	    fmt.Println(apiErr.StatusCode, apiErr.Body)
//	}
type APIError = api.APIError

// NormalizeType resolves a user-provided entity type string to its canonical
// Targetprocess API form. It handles case-insensitive matching, plural stripping,
// and common aliases (e.g., "stories" -> "UserStory", "bug" -> "Bug").
//
// Unknown types pass through unchanged.
func NormalizeType(input string) string {
	return resolve.EntityType(input)
}
