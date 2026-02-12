package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Simulation holds a set of request/response pairs for replay.
type Simulation struct {
	Pairs []Pair `json:"pairs"`
}

// Pair is a single recorded request/response exchange.
type Pair struct {
	Description string   `json:"description,omitempty"`
	Request     Request  `json:"request"`
	Response    Response `json:"response"`
}

// Request describes the expected HTTP request to match.
type Request struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
}

// Response describes the canned HTTP response to return.
type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body"`
}

// BodyBytes returns the response body as raw bytes.
// If body is a JSON string (e.g. XML content), it unquotes it.
// If body is a JSON object/array, it returns the raw JSON.
func (r Response) BodyBytes() []byte {
	var s string
	if err := json.Unmarshal(r.Body, &s); err == nil {
		return []byte(s)
	}
	return r.Body
}

// LoadSimulation reads a simulation file from disk.
func LoadSimulation(path string) (*Simulation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading simulation %s: %w", path, err)
	}
	var sim Simulation
	if err := json.Unmarshal(data, &sim); err != nil {
		return nil, fmt.Errorf("parsing simulation %s: %w", path, err)
	}
	return &sim, nil
}

// LoadSimulationsFromDir loads all .json files from a directory.
func LoadSimulationsFromDir(dir string) (*Simulation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading simulation directory %s: %w", dir, err)
	}
	combined := &Simulation{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		sim, err := LoadSimulation(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		combined.Pairs = append(combined.Pairs, sim.Pairs...)
	}
	return combined, nil
}

// matches checks whether an HTTP request matches a simulation request.
func matches(r *http.Request, sim Request) bool {
	if r.Method != sim.Method {
		return false
	}
	if r.URL.Path != sim.Path {
		return false
	}
	// If the simulation specifies query params, they must all be present.
	if len(sim.Query) > 0 {
		actual := r.URL.Query()
		for key, expected := range sim.Query {
			if got := actual.Get(key); got != expected {
				return false
			}
		}
	}
	return true
}

// SimulationServer wraps an httptest.Server that replays recorded simulations.
type SimulationServer struct {
	Server *httptest.Server

	mu       sync.Mutex
	sim      *Simulation
	requests []recordedRequest
}

type recordedRequest struct {
	Method string
	Path   string
	Query  url.Values
}

// NewSimulationServer creates and starts a test server from simulation data.
func NewSimulationServer(sim *Simulation) *SimulationServer {
	ss := &SimulationServer{sim: sim}
	ss.Server = httptest.NewServer(http.HandlerFunc(ss.handler))
	return ss
}

func (ss *SimulationServer) handler(w http.ResponseWriter, r *http.Request) {
	ss.mu.Lock()
	ss.requests = append(ss.requests, recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
	})
	ss.mu.Unlock()

	for _, pair := range ss.sim.Pairs {
		if !matches(r, pair.Request) {
			continue
		}
		for k, v := range pair.Response.Headers {
			w.Header().Set(k, v)
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		status := pair.Response.Status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if _, wErr := w.Write(pair.Response.BodyBytes()); wErr != nil {
			return // client disconnected
		}
		return
	}

	// No match found — return detailed 404
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "no matching simulation for %s %s", r.Method, r.URL.String())
}

// URL returns the test server's URL.
func (ss *SimulationServer) URL() string {
	return ss.Server.URL
}

// Close shuts down the test server.
func (ss *SimulationServer) Close() {
	ss.Server.Close()
}

// Requests returns all recorded requests.
func (ss *SimulationServer) Requests() []recordedRequest {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	copied := make([]recordedRequest, len(ss.requests))
	copy(copied, ss.requests)
	return copied
}

// SaveSimulation writes a simulation to a JSON file.
func SaveSimulation(path string, sim *Simulation) error {
	data, err := json.MarshalIndent(sim, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding simulation: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
