package text

import (
	"context"

	"github.com/lifedraft/targetprocess-cli/internal/api"
)

// PrepareFields processes text fields in a TP entity field map before submission.
// For Description fields, it resolves @mentions and prepends the markdown prefix.
func PrepareFields(ctx context.Context, client *api.Client, fields map[string]any) error {
	v, ok := fields["Description"]
	if !ok {
		return nil
	}
	desc, ok := v.(string)
	if !ok {
		return nil
	}

	resolver := &UserResolver{Client: client}
	resolved, err := resolver.ResolveMentions(ctx, desc)
	if err != nil {
		return err
	}
	fields["Description"] = EnsureMarkdown(resolved)
	return nil
}
