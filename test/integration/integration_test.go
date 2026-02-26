//go:build integration

package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"

	"github.com/lifedraft/targetprocess-cli/internal/testutil"
)

var testBinary string

func TestMain(m *testing.M) {
	// Build the tp binary once for all tests.
	dir, err := os.MkdirTemp("", "tp-integration-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	bin := filepath.Join(dir, "tp")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	build := exec.Command("go", "build", "-o", bin, projectRoot()+"/cmd/tp")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to build tp binary: " + err.Error())
	}
	testBinary = bin

	os.Exit(m.Run())
}

// runTP executes the tp binary against the simulation server and returns stdout.
func runTP(t *testing.T, serverURL string, args ...string) string {
	t.Helper()

	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(),
		"TP_DOMAIN="+serverURL,
		"TP_TOKEN=test-token",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("tp %s failed: %v\nstderr: %s\nstdout: %s",
			strings.Join(args, " "), err, stderr.String(), stdout.String())
	}
	return stdout.String()
}

// runTPExpectError executes the tp binary and expects a non-zero exit code.
func runTPExpectError(t *testing.T, serverURL string, args ...string) string {
	t.Helper()

	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(),
		"TP_DOMAIN="+serverURL,
		"TP_TOKEN=test-token",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to fail but it succeeded")
	}
	// Combine stdout and stderr for error snapshot
	return stderr.String()
}

func startServer(t *testing.T, simFiles ...string) *testutil.SimulationServer {
	t.Helper()

	combined := &testutil.Simulation{}
	for _, f := range simFiles {
		path := filepath.Join(projectRoot(), "testdata", "simulations", f)
		sim, err := testutil.LoadSimulation(path)
		if err != nil {
			t.Fatalf("loading simulation %s: %v", f, err)
		}
		combined.Pairs = append(combined.Pairs, sim.Pairs...)
	}

	ss := testutil.NewSimulationServer(combined)
	t.Cleanup(ss.Close)
	return ss
}

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// --- Query command tests ---

func TestQueryCollection(t *testing.T) {
	ss := startServer(t, "query_collection.json")
	out := runTP(t, ss.URL(),
		"query", "UserStory",
		"-s", "id,name,entityState.name as state",
		"-w", "entityState.isFinal!=true",
		"--take", "3",
	)
	cupaloy.SnapshotT(t, out)
}

func TestQueryCollectionJSON(t *testing.T) {
	ss := startServer(t, "query_collection.json")
	out := runTP(t, ss.URL(),
		"query", "UserStory",
		"-s", "id,name,entityState.name as state",
		"-w", "entityState.isFinal!=true",
		"--take", "3",
		"--output", "json",
	)
	cupaloy.SnapshotT(t, out)
}

func TestQuerySingleEntity(t *testing.T) {
	ss := startServer(t, "query_single.json")
	out := runTP(t, ss.URL(),
		"query", "UserStory/342348",
		"-s", "id,name,entityState.name as state",
	)
	cupaloy.SnapshotT(t, out)
}

func TestQueryDryRun(t *testing.T) {
	ss := startServer(t, "query_collection.json")
	out := runTP(t, ss.URL(),
		"query", "UserStory",
		"-s", "id,name",
		"-w", "entityState.isFinal!=true",
		"--dry-run",
	)
	// Replace the dynamic server URL for a stable snapshot.
	out = strings.ReplaceAll(out, ss.URL(), "http://test.tpondemand.com")
	cupaloy.SnapshotT(t, out)
}

// --- Entity command tests ---

func TestShow(t *testing.T) {
	ss := startServer(t, "entity_get.json")
	out := runTP(t, ss.URL(),
		"show", "342348",
		"--type", "UserStory",
	)
	cupaloy.SnapshotT(t, out)
}

func TestShowJSON(t *testing.T) {
	ss := startServer(t, "entity_get.json")
	out := runTP(t, ss.URL(),
		"show", "342348",
		"--type", "UserStory",
		"--output", "json",
	)
	cupaloy.SnapshotT(t, out)
}

func TestSearch(t *testing.T) {
	ss := startServer(t, "entity_search.json")
	out := runTP(t, ss.URL(),
		"search", "UserStory",
		"-s", "id,name,entityState.name as state",
		"-w", "entityState.isFinal!=true",
		"--take", "3",
	)
	cupaloy.SnapshotT(t, out)
}

func TestSearchJSON(t *testing.T) {
	ss := startServer(t, "entity_search.json")
	out := runTP(t, ss.URL(),
		"search", "UserStory",
		"-s", "id,name,entityState.name as state",
		"-w", "entityState.isFinal!=true",
		"--take", "3",
		"--output", "json",
	)
	cupaloy.SnapshotT(t, out)
}

// --- Inspect command tests ---

func TestInspectTypes(t *testing.T) {
	ss := startServer(t, "inspect_types.json")
	out := runTP(t, ss.URL(),
		"inspect", "types",
	)
	cupaloy.SnapshotT(t, out)
}

func TestInspectTypesJSON(t *testing.T) {
	ss := startServer(t, "inspect_types.json")
	out := runTP(t, ss.URL(),
		"inspect", "types",
		"--output", "json",
	)
	cupaloy.SnapshotT(t, out)
}

func TestInspectProperties(t *testing.T) {
	ss := startServer(t, "inspect_properties.json")
	out := runTP(t, ss.URL(),
		"inspect", "properties",
		"--type", "UserStory",
	)
	cupaloy.SnapshotT(t, out)
}

// --- Comment command tests ---

func TestCommentList(t *testing.T) {
	ss := startServer(t, "comment_list.json")
	out := runTP(t, ss.URL(), "comment", "list", "342236")
	cupaloy.SnapshotT(t, out)
}

func TestCommentAdd(t *testing.T) {
	ss := startServer(t, "comment_add.json")
	out := runTP(t, ss.URL(), "comment", "add", "342236", "New comment added")
	cupaloy.SnapshotT(t, out)
}

func TestCommentDelete(t *testing.T) {
	ss := startServer(t, "comment_delete.json")
	out := runTP(t, ss.URL(), "comment", "delete", "1001")
	cupaloy.SnapshotT(t, out)
}

// --- Error scenario tests ---

func TestQueryMissingEntityType(t *testing.T) {
	ss := startServer(t, "query_collection.json")
	out := runTPExpectError(t, ss.URL(),
		"query",
	)
	cupaloy.SnapshotT(t, out)
}

func TestSearchNoMatch(t *testing.T) {
	// Use empty simulation to trigger "no matching simulation" 404
	ss := testutil.NewSimulationServer(&testutil.Simulation{})
	t.Cleanup(ss.Close)

	out := runTPExpectError(t, ss.URL(),
		"search", "NonExistent",
		"-s", "id,name",
	)
	// Stabilize dynamic URL in error output
	out = strings.ReplaceAll(out, ss.URL(), "http://test.tpondemand.com")
	cupaloy.SnapshotT(t, out)
}
