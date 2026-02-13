package text

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/testutil"
)

func TestMentionRegex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		want    []string // expected captured names
	}{
		{
			name:    "simple name",
			input:   "@timo",
			wantLen: 1,
			want:    []string{"timo"},
		},
		{
			name:    "dotted name",
			input:   "@timo.litzius",
			wantLen: 1,
			want:    []string{"timo.litzius"},
		},
		{
			name:    "email should not match",
			input:   "user@email.com",
			wantLen: 0,
		},
		{
			name:    "already resolved should not match",
			input:   "@user:timo[Timo Litzius]",
			wantLen: 1,
			want:    []string{"user"}, // only matches @user, not the full resolved format
		},
		{
			name:    "mention in parens",
			input:   "(@name)",
			wantLen: 1,
			want:    []string{"name"},
		},
		{
			name:    "mention after space",
			input:   "hello @world",
			wantLen: 1,
			want:    []string{"world"},
		},
		{
			name:    "multiple mentions",
			input:   "@alice and @bob",
			wantLen: 2,
			want:    []string{"alice", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := mentionRe.FindAllStringSubmatch(tt.input, -1)
			if len(matches) != tt.wantLen {
				t.Fatalf("expected %d matches, got %d: %v", tt.wantLen, len(matches), matches)
			}
			for i, m := range matches {
				if i < len(tt.want) && m[1] != tt.want[i] {
					t.Errorf("match[%d] = %q, want %q", i, m[1], tt.want[i])
				}
			}
		})
	}
}

func TestResolveMentions(t *testing.T) {
	userResponse, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{
				"id":        1,
				"login":     "timo.litzius",
				"firstName": "Timo",
				"lastName":  "Litzius",
			},
		},
	})

	emptyResponse, _ := json.Marshal(map[string]any{
		"items": []any{},
	})

	sim := &testutil.Simulation{
		Pairs: []testutil.Pair{
			{
				Description: "exact login match for timo.litzius",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "login=='timo.litzius'",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   userResponse,
				},
			},
			{
				Description: "exact login match for timo",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "login=='timo'",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   emptyResponse,
				},
			},
			{
				Description: "login contains timo",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "login.contains('timo')",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   userResponse,
				},
			},
			{
				Description: "exact login match for unknown",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "login=='unknown'",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   emptyResponse,
				},
			},
			{
				Description: "login contains unknown",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "login.contains('unknown')",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   emptyResponse,
				},
			},
			{
				Description: "firstName match for unknown",
				Request: testutil.Request{
					Method: "GET",
					Path:   "/api/v2/GeneralUser",
					Query: map[string]string{
						"where":  "firstName.toLower()=='unknown'",
						"select": "{id,login,firstName,lastName}",
						"take":   "1",
					},
				},
				Response: testutil.Response{
					Status: 200,
					Body:   emptyResponse,
				},
			},
		},
	}

	ss := testutil.NewSimulationServer(sim)
	defer ss.Close()

	client := api.NewClient(ss.URL(), "test-token", false)
	resolver := &UserResolver{Client: client}
	ctx := context.Background()

	t.Run("resolve dotted mention", func(t *testing.T) {
		got, err := resolver.ResolveMentions(ctx, "hello @timo.litzius")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "hello @user:timo.litzius[Timo Litzius]"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("resolve short mention via contains", func(t *testing.T) {
		got, err := resolver.ResolveMentions(ctx, "hi @timo!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "hi @user:timo.litzius[Timo Litzius]!"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unresolvable mention left unchanged", func(t *testing.T) {
		got, err := resolver.ResolveMentions(ctx, "cc @unknown")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "cc @unknown"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no mentions returns text unchanged", func(t *testing.T) {
		got, err := resolver.ResolveMentions(ctx, "plain text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "plain text" {
			t.Errorf("got %q, want %q", got, "plain text")
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		got, err := resolver.ResolveMentions(ctx, "@timo.litzius and @timo.litzius again")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "@user:timo.litzius[Timo Litzius] and @user:timo.litzius[Timo Litzius] again"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
