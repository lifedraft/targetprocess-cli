package targetprocess

import (
	"errors"
	"net/http"
	"testing"
)

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"userstory", "UserStory"},
		{"UserStory", "UserStory"},
		{"stories", "UserStory"},
		{"story", "UserStory"},
		{"us", "UserStory"},
		{"bug", "Bug"},
		{"bugs", "Bug"},
		{"task", "Task"},
		{"feature", "Feature"},
		{"epic", "Epic"},
		{"sprint", "Iteration"},
		{"Unknown", "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeType(tt.input); got != tt.want {
				t.Errorf("NormalizeType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAPIErrorAs(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: "not found"}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should match *APIError")
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
