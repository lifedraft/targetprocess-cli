package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Precompiled regexes for error pattern matching and select validation.
var (
	regexNow           = regexp.MustCompile(`\bNow\b`)
	regexColonSubfield = regexp.MustCompile(`\{[a-zA-Z]+:\{`)
	regexTokenPattern  = regexp.MustCompile(`\b([a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)+)\b`)
	regexAsPattern     = regexp.MustCompile(`\b([a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)+)\s+as\b`)
	regexParenPattern  = regexp.MustCompile(`\([^)]*\)`)
)

// errorPattern defines a known API error pattern and its suggested fix.
type errorPattern struct {
	// Name is a short identifier for the pattern (for debugging/logging).
	Name string

	// Match returns true if this pattern applies to the given error context.
	Match func(apiErr *APIError, path string, params map[string]string) bool

	// Hint is the suggestion shown to the user.
	Hint string
}

// knownPatterns is the list of known API error patterns with fix suggestions.
// Order matters: first match wins.
var knownPatterns = []errorPattern{
	{
		Name: "is-null",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode != http.StatusBadRequest {
				return false
			}
			if !strings.Contains(apiErr.Body, "mismatched input 'is'") {
				return false
			}
			where := params["where"]
			if strings.Contains(strings.ToLower(where), "is not null") {
				return false // handled by next pattern
			}
			return strings.Contains(strings.ToLower(where), "is null")
		},
		Hint: "Use ==null instead of 'is null'. Example: description==null",
	},
	{
		Name: "is-not-null",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode != http.StatusBadRequest {
				return false
			}
			if !strings.Contains(apiErr.Body, "mismatched input 'is'") {
				return false
			}
			return strings.Contains(strings.ToLower(params["where"]), "is not null")
		},
		Hint: "Use !=null instead of 'is not null'. Example: description!=null",
	},
	{
		Name: "is-mismatched-generic",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			return apiErr.StatusCode == http.StatusBadRequest && strings.Contains(apiErr.Body, "mismatched input 'is'")
		},
		Hint: "Use ==null instead of 'is null', or !=null instead of 'is not null'.",
	},
	{
		Name: "datetime-minus-int",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			return apiErr.StatusCode == http.StatusBadRequest &&
				strings.Contains(apiErr.Body, "incompatible types") &&
				strings.Contains(apiErr.Body, "DateTime") &&
				strings.Contains(apiErr.Body, "Int32")
		},
		Hint: "Date arithmetic like 'Today - 7' is not supported. Use Today.AddDays(-7) instead.",
	},
	{
		Name: "datetime-vs-string",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			return apiErr.StatusCode == http.StatusBadRequest &&
				strings.Contains(apiErr.Body, "incompatible types") &&
				strings.Contains(apiErr.Body, "DateTime") &&
				strings.Contains(apiErr.Body, "String")
		},
		Hint: "Date string literals don't work in v2. Use Today.AddDays(-N) for relative dates. Example: createDate > Today.AddDays(-30)",
	},
	{
		Name: "now-not-recognized",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode != http.StatusBadRequest {
				return false
			}
			where := params["where"]
			// Check if the where clause uses "Now" (case-sensitive, as a word boundary).
			return regexNow.MatchString(where)
		},
		Hint: "Use 'Today' instead of 'Now'. The v2 API does not recognize 'Now'.",
	},
	{
		Name: "collection-all",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode < 400 {
				return false
			}
			allParams := params["where"] + " " + params["select"] + " " + params["orderBy"]
			return strings.Contains(apiErr.Body, "issues with generated report") &&
				strings.Contains(allParams, ".all(")
		},
		Hint: ".all() is not supported. Use .where(condition).count == .count instead.",
	},
	{
		Name: "collection-none",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			return apiErr.StatusCode >= 400 &&
				strings.Contains(apiErr.Body, "'none'") &&
				strings.Contains(apiErr.Body, "does not exist")
		},
		Hint: ".none() is not supported. Use .where(condition).count == 0 instead.",
	},
	{
		Name: "collection-first",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			return apiErr.StatusCode >= 400 &&
				strings.Contains(apiErr.Body, "'first'") &&
				strings.Contains(apiErr.Body, "does not exist")
		},
		Hint: ".first() is not supported. Use .select({...}) and take the first result client-side.",
	},
	{
		Name: "include-in-v2",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			_, hasInclude := params["include"]
			return hasInclude && strings.Contains(path, "/api/v2/")
		},
		Hint: "'include' is a v1 parameter. In v2, use 'select' instead: select={id,name,field}",
	},
	{
		Name: "wrong-entity-plural-404",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode != http.StatusNotFound {
				return false
			}
			return strings.Contains(path, "/api/v2/") && endsWithPlural(path)
		},
		Hint: "v2 uses singular entity names. Example: /api/v2/UserStory (not /api/v2/UserStorys or /api/v2/UserStories).",
	},
	{
		Name: "orderby-aggregate",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			if apiErr.StatusCode < 400 {
				return false
			}
			if !strings.Contains(apiErr.Body, "issues with generated report") {
				return false
			}
			orderBy := params["orderBy"]
			return strings.Contains(orderBy, "count") || strings.Contains(orderBy, "sum") ||
				strings.Contains(orderBy, "avg") || strings.Contains(orderBy, "Count") ||
				strings.Contains(orderBy, "Sum") || strings.Contains(orderBy, "Avg")
		},
		Hint: "Ordering by aggregate fields is not supported in v2. Sort results client-side.",
	},
	{
		Name: "groupby-count",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			_, hasGroupBy := params["groupBy"]
			if !hasGroupBy {
				return false
			}
			selectParam := params["select"]
			return strings.Contains(selectParam, "count") || strings.Contains(selectParam, "Count")
		},
		Hint: "Top-level groupBy with count is not well supported. Query from the parent entity using a collection instead.",
	},
	{
		Name: "colon-subfield-syntax",
		Match: func(apiErr *APIError, path string, params map[string]string) bool {
			// The {field:{subfield}} syntax returns wrong data rather than an error,
			// but if they somehow got an error and we see this pattern, warn about it.
			selectParam := params["select"]
			return apiErr.StatusCode >= 400 &&
				regexColonSubfield.MatchString(selectParam)
		},
		Hint: "The {field:{subfield}} syntax can return incorrect data. Use field.subfield as alias instead. Example: entityState.name as state",
	},
}

// endsWithPlural checks if the path's last segment looks like a naive plural (ends with 's').
func endsWithPlural(path string) bool {
	// Strip query string if present.
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	// Skip if it's a numeric ID segment.
	if last != "" && last[0] >= '0' && last[0] <= '9' {
		return false
	}
	// Common entity types that end in 's' but are not plurals
	lower := strings.ToLower(last)
	notPlural := []string{"process", "status", "address", "access", "progress", "class", "analysis"}
	for _, np := range notPlural {
		if lower == np {
			return false
		}
	}
	return strings.HasSuffix(last, "s") || strings.HasSuffix(last, "S")
}

// EnhanceError checks if an API error matches known patterns and returns
// an enhanced error message with fix suggestions. If no pattern matches,
// returns the original error unchanged.
func EnhanceError(err error, path string, params map[string]string) error {
	if err == nil {
		return nil
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	if params == nil {
		params = map[string]string{}
	}

	for _, p := range knownPatterns {
		if p.Match(apiErr, path, params) {
			return fmt.Errorf("%w\n\nHint: %s", err, p.Hint)
		}
	}

	return err
}

// WarnSelectDotPaths checks for dot-path fields in a select expression
// that are missing 'as' aliases. These fields are silently dropped by the API.
// Returns a warning message or empty string.
func WarnSelectDotPaths(selectExpr string) string {
	if selectExpr == "" {
		return ""
	}

	allDotPaths := regexTokenPattern.FindAllString(selectExpr, -1)
	if len(allDotPaths) == 0 {
		return ""
	}

	aliased := make(map[string]bool)
	for _, match := range regexAsPattern.FindAllStringSubmatch(selectExpr, -1) {
		aliased[match[1]] = true
	}

	// Find dot-paths that appear inside parentheses (e.g., .where(entityState.isFinal==true))
	// These are collection sub-expressions and should not trigger warnings.
	insideParens := make(map[string]bool)
	for _, match := range regexParenPattern.FindAllString(selectExpr, -1) {
		for _, dp := range regexTokenPattern.FindAllString(match, -1) {
			insideParens[dp] = true
		}
	}

	var missing []string
	seen := make(map[string]bool)
	for _, dp := range allDotPaths {
		// Skip things that look like method calls (e.g., Today.AddDays).
		if strings.Contains(selectExpr, dp+"(") {
			continue
		}
		// Skip dot-paths inside parentheses (collection sub-expressions).
		if insideParens[dp] {
			continue
		}
		if !aliased[dp] && !seen[dp] {
			missing = append(missing, dp)
			seen[dp] = true
		}
	}

	if len(missing) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Warning: These dot-path fields in select are missing 'as' aliases and will be silently dropped by the API:\n")
	for _, m := range missing {
		sb.WriteString(fmt.Sprintf("  - %s  (add: %s as %s)\n", m, m, suggestAlias(m)))
	}
	return sb.String()
}

// suggestAlias generates a simple alias from a dot-path by taking the last segment.
func suggestAlias(dotPath string) string {
	parts := strings.Split(dotPath, ".")
	return parts[len(parts)-1]
}
