package targetprocess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer creates a test HTTP server that responds based on the request path.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := NewClient(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c, srv
}

func TestNewClient(t *testing.T) {
	c, err := NewClient("example.tpondemand.com", "my-token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	custom := &http.Client{}
	c, err := NewClient("example.tpondemand.com", "tok", WithHTTPClient(custom), WithDebug(true))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if c.internal.HTTPClient != custom {
		t.Error("custom HTTP client not applied")
	}
	if !c.internal.Debug {
		t.Error("debug not enabled")
	}
}

func TestClientGet(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/UserStorys/123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]any{
			"Id":           123,
			"Name":         "Test Story",
			"ResourceType": "UserStory",
		})
	})
	defer srv.Close()

	entity, err := c.Get(context.Background(), "UserStory", 123)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if entity["Name"] != "Test Story" {
		t.Errorf("Name = %v, want %q", entity["Name"], "Test Story")
	}
}

func TestClientSearch(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v2/Bug") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]any{
			"items": []map[string]any{
				{"id": 1, "name": "Bug One"},
				{"id": 2, "name": "Bug Two"},
			},
		})
	})
	defer srv.Close()

	items, err := c.Search(context.Background(), "Bug", "Severity.Name=='Critical'", WithTake(10))
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestClientCreate(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		writeJSON(w, map[string]any{
			"Id":           999,
			"Name":         "New Task",
			"ResourceType": "Task",
		})
	})
	defer srv.Close()

	entity, err := c.Create(context.Background(), "Task", Entity{"Name": "New Task"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id, ok := entity["Id"].(float64); !ok || int(id) != 999 {
		t.Errorf("Id = %v, want 999", entity["Id"])
	}
}

func TestClientUpdate(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		writeJSON(w, map[string]any{
			"Id":   42,
			"Name": "Updated",
		})
	})
	defer srv.Close()

	entity, err := c.Update(context.Background(), "UserStory", 42, Entity{"Name": "Updated"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if entity["Name"] != "Updated" {
		t.Errorf("Name = %v, want %q", entity["Name"], "Updated")
	}
}

func TestClientDelete(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := c.Delete(context.Background(), "Bug", 55)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestClientQuery(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v2/UserStory") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]any{
			"items": []map[string]any{
				{"id": 1, "name": "Story", "state": "Open"},
			},
			"next": "https://example.com/api/v2/UserStory?skip=1",
		})
	})
	defer srv.Close()

	data, err := c.Query(context.Background(), "UserStory", QueryParams{
		Where:  "EntityState.Name=='Open'",
		Select: "id,name,entityState.name as state",
		Take:   1,
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Query() returned empty response")
	}
}

func TestClientAPIError(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]any{"Message": "Entity not found"})
	})
	defer srv.Close()

	_, err := c.Get(context.Background(), "UserStory", 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	//nolint:errcheck // test helper
	json.NewEncoder(w).Encode(v)
}
