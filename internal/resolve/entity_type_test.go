package resolve

import "testing"

func TestEntityType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Exact match (already canonical)
		{"UserStory", "UserStory"},
		{"Bug", "Bug"},
		{"Task", "Task"},
		{"Feature", "Feature"},

		// Case-insensitive
		{"userstory", "UserStory"},
		{"USERSTORY", "UserStory"},
		{"Userstory", "UserStory"},
		{"bug", "Bug"},
		{"BUG", "Bug"},
		{"testcase", "TestCase"},
		{"TESTCASE", "TestCase"},

		// Plurals
		{"UserStories", "UserStory"},
		{"userstories", "UserStory"},
		{"Bugs", "Bug"},
		{"bugs", "Bug"},
		{"Tasks", "Task"},
		{"Features", "Feature"},
		{"Epics", "Epic"},
		{"stories", "UserStory"},
		{"Stories", "UserStory"},
		{"TestCases", "TestCase"},
		{"testcases", "TestCase"},

		// Aliases
		{"story", "UserStory"},
		{"Story", "UserStory"},
		{"STORY", "UserStory"},
		{"us", "UserStory"},
		{"US", "UserStory"},
		{"issue", "Bug"},
		{"defect", "Bug"},
		{"sprint", "Iteration"},
		{"test", "TestCase"},

		// Unknown types pass through
		{"SomeCustomType", "SomeCustomType"},
		{"Whatever", "Whatever"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EntityType(tt.input)
			if got != tt.want {
				t.Errorf("EntityType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
