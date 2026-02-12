package bugreport

import (
	"context"
	"encoding/json"
	"net/url"
	"runtime"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
)

func testInfo() Info {
	return Info{
		CLIVersion:  "1.2.3",
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		ConfigPath:  "/tmp/nonexistent/config.yaml",
		ConfigFound: false,
		Domain:      "example.tpondemand.com",
		APIStatus:   "reachable",
	}
}

func TestCollectInfo(t *testing.T) {
	f := &cmdutil.Factory{}
	info := Collect(f, "1.2.3")

	if info.CLIVersion != "1.2.3" {
		t.Errorf("CLIVersion = %q, want %q", info.CLIVersion, "1.2.3")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
	if info.OS == "" {
		t.Error("OS is empty")
	}
	if info.Arch == "" {
		t.Error("Arch is empty")
	}
	if info.ConfigPath == "" {
		t.Error("ConfigPath is empty")
	}
}

func TestFormatText(t *testing.T) {
	info := testInfo()
	text := FormatText(info)

	checks := []struct {
		label    string
		contains string
	}{
		{"CLI version", "CLI version:  1.2.3"},
		{"Go version", "Go version:"},
		{"OS/Arch", "OS/Arch:"},
		{"Config path", "Config path:"},
		{"Config found", "Config found: false"},
		{"TP domain", "TP domain:    example.tpondemand.com"},
		{"API status", "API status:   reachable"},
	}

	for _, c := range checks {
		if !strings.Contains(text, c.contains) {
			t.Errorf("FormatText missing %s: expected %q in output:\n%s", c.label, c.contains, text)
		}
	}
}

func TestFormatTextNoDomain(t *testing.T) {
	info := testInfo()
	info.Domain = ""
	info.APIStatus = ""
	text := FormatText(info)

	if strings.Contains(text, "TP domain") {
		t.Error("FormatText should not include TP domain when empty")
	}
	if strings.Contains(text, "API status") {
		t.Error("FormatText should not include API status when domain is empty")
	}
}

func TestFormatJSON(t *testing.T) {
	info := testInfo()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	expectedKeys := []string{"cli_version", "go_version", "os", "arch", "config_path", "config_found"}
	for _, key := range expectedKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON output missing key %q", key)
		}
	}

	if parsed["cli_version"] != "1.2.3" {
		t.Errorf("cli_version = %v, want 1.2.3", parsed["cli_version"])
	}
}

func TestBuildIssueURL(t *testing.T) {
	info := testInfo()
	issueURL := BuildIssueURL(info)

	if !strings.HasPrefix(issueURL, "https://github.com/lifedraft/targetprocess-cli/issues/new") {
		t.Errorf("URL has wrong prefix: %s", issueURL)
	}

	parsed, err := url.Parse(issueURL)
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}

	q := parsed.Query()
	if q.Get("labels") != "bug" {
		t.Errorf("labels = %q, want %q", q.Get("labels"), "bug")
	}
	if q.Get("template") != "bug_report.yml" {
		t.Errorf("template = %q, want %q", q.Get("template"), "bug_report.yml")
	}

	body := q.Get("body")
	if !strings.Contains(body, "CLI version:  1.2.3") {
		t.Error("body should contain environment info")
	}
	if !strings.Contains(body, "## Environment") {
		t.Error("body should contain Environment heading")
	}
	if !strings.Contains(body, "## Steps to Reproduce") {
		t.Error("body should contain Steps to Reproduce heading")
	}

	if len(issueURL) > 8000 {
		t.Errorf("URL length %d exceeds 8000 char limit", len(issueURL))
	}
}

func TestInvalidMode(t *testing.T) {
	f := &cmdutil.Factory{}
	cmd := NewCmd(f, "1.0.0")

	app := &cli.Command{
		Name:     "tp",
		Commands: []*cli.Command{cmd},
	}

	err := app.Run(context.Background(), []string{"tp", "bug-report", "--mode", "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown mode")
	}
}

func TestBuildPanicIssueURL(t *testing.T) {
	info := testInfo()
	stack := []byte("goroutine 1 [running]:\nmain.main()\n\t/cmd/tp/main.go:30")
	issueURL := BuildPanicIssueURL(info, "runtime error: index out of range", stack)

	parsed, err := url.Parse(issueURL)
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}

	q := parsed.Query()
	if !strings.Contains(q.Get("title"), "runtime error: index out of range") {
		t.Error("title should contain the panic message")
	}
	if !strings.Contains(q.Get("labels"), "crash") {
		t.Errorf("labels = %q, want it to contain 'crash'", q.Get("labels"))
	}

	body := q.Get("body")
	if !strings.Contains(body, "runtime error: index out of range") {
		t.Error("body should contain the panic message")
	}
	if !strings.Contains(body, "goroutine 1") {
		t.Error("body should contain the stack trace")
	}
	if !strings.Contains(body, "CLI version:  1.2.3") {
		t.Error("body should contain environment info")
	}
	if len(issueURL) > 8000 {
		t.Errorf("URL length %d exceeds 8000 char limit", len(issueURL))
	}
}

func TestBuildPanicIssueURLLongStack(t *testing.T) {
	info := testInfo()
	// Generate a stack trace that's way too long.
	stack := []byte(strings.Repeat("goroutine 1 [running]:\nsome/deep/call\n", 200))
	issueURL := BuildPanicIssueURL(info, "boom", stack)

	if len(issueURL) > 8000 {
		t.Errorf("URL length %d exceeds 8000 char limit even with long stack", len(issueURL))
	}
}

func TestTokenNeverInOutput(t *testing.T) {
	fakeToken := "super-secret-token-12345"
	info := testInfo()

	text := FormatText(info)
	if strings.Contains(text, fakeToken) {
		t.Error("FormatText should never contain token values")
	}

	issueURL := BuildIssueURL(info)
	if strings.Contains(issueURL, fakeToken) {
		t.Error("BuildIssueURL should never contain token values")
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "token") {
		t.Error("JSON output should not contain a token field")
	}
}
