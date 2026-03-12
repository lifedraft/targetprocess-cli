package targetprocess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTyped(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"Id":           123,
			"Name":         "Login Page Broken",
			"ResourceType": "Bug",
			"EntityState": map[string]any{
				"Id":   10,
				"Name": "Open",
			},
			"Severity": map[string]any{
				"Id":   3,
				"Name": "High",
			},
		})
	})
	defer srv.Close()

	bug, err := Get[Bug](context.Background(), c, 123)
	if err != nil {
		t.Fatalf("Get[Bug]() error = %v", err)
	}
	if bug.ID != 123 {
		t.Errorf("ID = %d, want 123", bug.ID)
	}
	if bug.Name != "Login Page Broken" {
		t.Errorf("Name = %q, want %q", bug.Name, "Login Page Broken")
	}
	if bug.EntityState == nil || bug.EntityState.Name != "Open" {
		t.Errorf("EntityState.Name = %v, want %q", bug.EntityState, "Open")
	}
	if bug.Severity == nil || bug.Severity.Name != "High" {
		t.Errorf("Severity.Name = %v, want %q", bug.Severity, "High")
	}
}

func TestGetTypedTask(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"Id":           456,
			"Name":         "Write tests",
			"ResourceType": "Task",
			"UserStory": map[string]any{
				"Id":   100,
				"Name": "Auth Feature",
			},
		})
	})
	defer srv.Close()

	task, err := Get[Task](context.Background(), c, 456)
	if err != nil {
		t.Fatalf("Get[Task]() error = %v", err)
	}
	if task.UserStory == nil || task.UserStory.Name != "Auth Feature" {
		t.Errorf("UserStory = %v, want Auth Feature", task.UserStory)
	}
}

func TestSearchTyped(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"items": []map[string]any{
				{"id": 1, "name": "Story A", "resourceType": "UserStory"},
				{"id": 2, "name": "Story B", "resourceType": "UserStory"},
			},
		})
	})
	defer srv.Close()

	stories, err := Search[UserStory](context.Background(), c, "Project.Name=='Test'", WithTake(10))
	if err != nil {
		t.Fatalf("Search[UserStory]() error = %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("got %d stories, want 2", len(stories))
	}
}

func TestCreateTyped(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		writeJSON(w, map[string]any{
			"Id":           789,
			"Name":         "New Story",
			"ResourceType": "UserStory",
		})
	})
	defer srv.Close()

	story, err := Create[UserStory](context.Background(), c, Entity{"Name": "New Story"})
	if err != nil {
		t.Fatalf("Create[UserStory]() error = %v", err)
	}
	if story.ID != 789 {
		t.Errorf("ID = %d, want 789", story.ID)
	}
}

func TestUpdateTyped(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"Id":   42,
			"Name": "Updated Feature",
		})
	})
	defer srv.Close()

	feature, err := Update[Feature](context.Background(), c, 42, Entity{"Name": "Updated Feature"})
	if err != nil {
		t.Fatalf("Update[Feature]() error = %v", err)
	}
	if feature.Name != "Updated Feature" {
		t.Errorf("Name = %q, want %q", feature.Name, "Updated Feature")
	}
}

func TestQueryTypedCustomStruct(t *testing.T) {
	type SprintItem struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	}

	c, srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"items": []map[string]any{
				{"id": 1, "name": "Story One", "state": "Open"},
				{"id": 2, "name": "Story Two", "state": "Done"},
			},
			"next": "https://example.com/api/v2/UserStory?skip=2",
		})
	})
	defer srv.Close()

	result, err := Query[SprintItem](context.Background(), c, "UserStory", QueryParams{
		Where:  "TeamIteration.Name=='Sprint 42'",
		Select: "id,name,entityState.name as state",
	})
	if err != nil {
		t.Fatalf("Query[SprintItem]() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(result.Items))
	}
	if result.Items[0].State != "Open" {
		t.Errorf("Items[0].State = %q, want %q", result.Items[0].State, "Open")
	}
	if result.Next == "" {
		t.Error("Next should not be empty")
	}
}

func TestQueryTypedEmptyResult(t *testing.T) {
	type Item struct {
		ID int `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Query[Item](context.Background(), c, "Bug", QueryParams{Where: "id==0"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("got %d items, want 0", len(result.Items))
	}
}
