package targetprocess

import "testing"

// Compile-time interface compliance checks.
var (
	_ Typed = UserStory{}
	_ Typed = Bug{}
	_ Typed = Task{}
	_ Typed = Feature{}
	_ Typed = Epic{}
	_ Typed = Request{}
	_ Typed = Comment{}

	_ Identifiable = BaseEntity{}
)

func TestTPResourceType(t *testing.T) {
	tests := []struct {
		entity Typed
		want   string
	}{
		{UserStory{}, "UserStory"},
		{Bug{}, "Bug"},
		{Task{}, "Task"},
		{Feature{}, "Feature"},
		{Epic{}, "Epic"},
		{Request{}, "Request"},
		{Comment{}, "Comment"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.entity.TPResourceType(); got != tt.want {
				t.Errorf("TPResourceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBaseEntityGetID(t *testing.T) {
	e := BaseEntity{ID: 42}
	if got := e.GetID(); got != 42 {
		t.Errorf("GetID() = %d, want 42", got)
	}
}

func TestCommentGetID(t *testing.T) {
	c := Comment{ID: 99}
	if got := c.GetID(); got != 99 {
		t.Errorf("GetID() = %d, want 99", got)
	}
}
