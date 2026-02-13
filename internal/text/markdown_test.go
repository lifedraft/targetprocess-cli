package text

import "testing"

func TestEnsureMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "plain text",
			input: "hello world",
			want:  "<!--markdown-->hello world",
		},
		{
			name:  "already prefixed",
			input: "<!--markdown-->hello world",
			want:  "<!--markdown-->hello world",
		},
		{
			name:  "prefix in middle of text",
			input: "before <!--markdown--> after",
			want:  "<!--markdown-->before <!--markdown--> after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("EnsureMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
