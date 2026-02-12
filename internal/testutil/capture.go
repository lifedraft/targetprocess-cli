package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RecordingTransport wraps an http.RoundTripper and records all request/response pairs.
type RecordingTransport struct {
	Base  http.RoundTripper
	Pairs []Pair
}

// RoundTrip executes the request and records the exchange.
func (rt *RecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.Base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Buffer the response body so we can read it and still return it.
	respBody, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	// Build query map (excluding access_token and format for cleaner fixtures).
	queryMap := make(map[string]string)
	for key, vals := range req.URL.Query() {
		if key == "access_token" || key == "format" {
			continue
		}
		queryMap[key] = vals[0]
	}

	// Build headers map.
	headers := make(map[string]string)
	for key, vals := range resp.Header {
		if strings.EqualFold(key, "Content-Type") {
			headers["Content-Type"] = vals[0]
		}
	}

	// Store body as raw JSON (either object or string for non-JSON).
	var body json.RawMessage
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "json") || isJSON(respBody) {
		body = json.RawMessage(respBody)
	} else {
		// Non-JSON (XML, etc.) — store as JSON string.
		encoded, err := json.Marshal(string(respBody))
		if err != nil {
			return nil, fmt.Errorf("encoding non-JSON body: %w", err)
		}
		body = json.RawMessage(encoded)
	}

	rt.Pairs = append(rt.Pairs, Pair{
		Request: Request{
			Method: req.Method,
			Path:   req.URL.Path,
			Query:  queryMap,
		},
		Response: Response{
			Status:  resp.StatusCode,
			Headers: headers,
			Body:    body,
		},
	})

	return resp, nil
}

func isJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

// BuildSimulation creates a Simulation from recorded pairs.
func (rt *RecordingTransport) BuildSimulation() *Simulation {
	return &Simulation{Pairs: rt.Pairs}
}
