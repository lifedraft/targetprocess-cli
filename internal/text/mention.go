package text

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lifedraft/targetprocess-cli/internal/api"
)

// mentionRe matches @mentions that are NOT email addresses and NOT already resolved.
// It matches @timo, @timo.litzius, (@name), but not user@email.com or @user:login[Name].
var mentionRe = regexp.MustCompile(`(?:^|[\s(])@([a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)*)`)

// UserResolver resolves @mentions in text to TargetProcess user references.
type UserResolver struct {
	Client *api.Client
}

type v2Response struct {
	Items []struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"items"`
}

// ResolveMentions replaces @mentions in text with @user:login[Full Name] format.
// Unresolvable mentions are left unchanged.
func (r *UserResolver) ResolveMentions(ctx context.Context, text string) (string, error) {
	matches := mentionRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	// Collect unique mentions.
	type mentionInfo struct {
		name    string
		resolved string
	}
	seen := make(map[string]*mentionInfo)
	var unique []*mentionInfo
	for _, m := range matches {
		// Submatch group 1 is the name after @.
		name := text[m[2]:m[3]]
		if _, ok := seen[name]; !ok {
			mi := &mentionInfo{name: name}
			seen[name] = mi
			unique = append(unique, mi)
		}
	}

	// Resolve each unique mention.
	for _, mi := range unique {
		resolved, err := r.lookupUser(ctx, mi.name)
		if err != nil {
			return "", err
		}
		mi.resolved = resolved
	}

	// Build result by replacing mentions from right to left to preserve indices.
	result := text
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		name := text[m[2]:m[3]]
		mi := seen[name]
		if mi.resolved == "" {
			continue
		}
		// m[2]-1 is the @ sign position (submatch start minus 1 for @).
		atPos := m[2] - 1
		endPos := m[3]
		result = result[:atPos] + mi.resolved + result[endPos:]
	}

	return result, nil
}

// lookupUser tries to find a TP user matching the given mention name.
// Strategy: exact login, then login contains, then first name match.
func (r *UserResolver) lookupUser(ctx context.Context, name string) (string, error) {
	strategies := []string{
		fmt.Sprintf("login=='%s'", name),
		fmt.Sprintf("login.contains('%s')", name),
		fmt.Sprintf("firstName.toLower()=='%s'", strings.ToLower(name)),
	}

	for _, where := range strategies {
		data, err := r.Client.QueryV2(ctx, "GeneralUser", api.V2Params{
			Where:  where,
			Select: "id,login,firstName,lastName",
			Take:   1,
		})
		if err != nil {
			return "", fmt.Errorf("looking up user %q: %w", name, err)
		}

		var resp v2Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("parsing user response for %q: %w", name, err)
		}

		if len(resp.Items) > 0 {
			u := resp.Items[0]
			fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
			return fmt.Sprintf("@user:%s[%s]", u.Login, fullName), nil
		}
	}

	return "", nil
}
