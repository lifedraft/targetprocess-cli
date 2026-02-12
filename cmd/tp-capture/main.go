// Command tp-capture records real Targetprocess API responses for use as test fixtures.
//
// Usage:
//
//	TP_DOMAIN=your.tpondemand.com TP_TOKEN=yourtoken go run ./cmd/tp-capture
//
// It will make a set of representative API calls, record the responses,
// redact all sensitive data, and write simulation files to testdata/simulations/.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/config"
	"github.com/lifedraft/targetprocess-cli/internal/testutil"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w (set TP_DOMAIN and TP_TOKEN env vars)", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	outDir := filepath.Join(projectRoot(), "testdata", "simulations")
	fmt.Printf("Capturing test data from %s\n", cfg.Domain)
	fmt.Printf("Output directory: %s\n\n", outDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := testutil.DefaultRedactOptions(cfg.Domain)

	scenarios := []struct {
		name    string
		capture func(ctx context.Context, client *api.Client, rt *testutil.RecordingTransport) error
	}{
		{"query_collection", captureQueryCollection},
		{"query_single", captureQuerySingle},
		{"entity_get", captureEntityGet},
		{"entity_search", captureEntitySearch},
		{"inspect_types", captureInspectTypes},
		{"inspect_properties", captureInspectProperties},
	}

	for _, sc := range scenarios {
		fmt.Printf("Capturing %s... ", sc.name)

		rt := &testutil.RecordingTransport{
			Base: http.DefaultTransport,
		}
		client := &api.Client{
			BaseURL:    "https://" + cfg.Domain,
			Token:      cfg.Token,
			HTTPClient: &http.Client{Transport: rt},
		}

		if err := sc.capture(ctx, client, rt); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			continue
		}

		sim := rt.BuildSimulation()

		// Add descriptions
		for i := range sim.Pairs {
			sim.Pairs[i].Description = sc.name
		}

		// Redact sensitive data
		testutil.ResetCounters()
		testutil.RedactSimulation(sim, opts)

		// Save
		path := filepath.Join(outDir, sc.name+".json")
		if err := testutil.SaveSimulation(path, sim); err != nil {
			return fmt.Errorf("saving %s: %w", sc.name, err)
		}
		fmt.Printf("OK (%d pairs)\n", len(sim.Pairs))
	}

	fmt.Println("\nCapture complete. Review testdata/simulations/ for redacted fixtures.")
	return nil
}

func captureQueryCollection(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	_, err := client.QueryV2(ctx, "UserStory", api.V2Params{
		Select: "id,name,entityState.name as state",
		Where:  "entityState.isFinal!=true",
		Take:   3,
	})
	return err
}

func captureQuerySingle(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	id, err := findFirstUserStoryID(ctx, client)
	if err != nil {
		return err
	}
	_, err = client.QueryV2Entity(ctx, "UserStory", id, "id,name,entityState.name as state")
	return err
}

func captureEntityGet(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	id, err := findFirstUserStoryID(ctx, client)
	if err != nil {
		return err
	}
	_, err = client.GetEntity(ctx, "UserStory", id, nil)
	return err
}

func captureEntitySearch(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	_, err := client.QueryV2(ctx, "UserStory", api.V2Params{
		Select: "id,name,entityState.name as state",
		Where:  "entityState.isFinal!=true",
		Take:   3,
	})
	return err
}

func captureInspectTypes(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	_, err := client.GetMetaIndex(ctx)
	return err
}

func captureInspectProperties(ctx context.Context, client *api.Client, _ *testutil.RecordingTransport) error {
	_, err := client.GetTypeMeta(ctx, "UserStory")
	return err
}

func findFirstUserStoryID(ctx context.Context, client *api.Client) (int, error) {
	data, err := client.QueryV2(ctx, "UserStory", api.V2Params{
		Select: "id",
		Take:   1,
	})
	if err != nil {
		return 0, err
	}
	type v2Resp struct {
		Items []struct {
			ID float64 `json:"id"`
		} `json:"items"`
	}
	var resp v2Resp
	if err := jsonUnmarshal(data, &resp); err != nil {
		return 0, err
	}
	if len(resp.Items) == 0 {
		return 0, errors.New("no user stories found")
	}
	return int(resp.Items[0].ID), nil
}

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v) //nolint:musttag // generic test helper
}
