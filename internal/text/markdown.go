package text

const markdownPrefix = "<!--markdown-->"

// EnsureMarkdown prepends the <!--markdown--> prefix to enable markdown rendering
// in TargetProcess. Returns empty strings unchanged. Already-prefixed strings are
// returned as-is.
func EnsureMarkdown(desc string) string {
	if desc == "" {
		return desc
	}
	if len(desc) >= len(markdownPrefix) && desc[:len(markdownPrefix)] == markdownPrefix {
		return desc
	}
	return markdownPrefix + desc
}
