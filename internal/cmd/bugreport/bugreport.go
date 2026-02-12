package bugreport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/config"
)

// Info holds the diagnostic information collected by the bug-report command.
type Info struct {
	CLIVersion  string `json:"cli_version"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	ConfigPath  string `json:"config_path"`
	ConfigFound bool   `json:"config_found"`
	Domain      string `json:"domain,omitempty"`
	APIStatus   string `json:"api_status,omitempty"`
}

// Collect gathers diagnostic information from the environment.
func Collect(f *cmdutil.Factory, version string) Info {
	info := Info{
		CLIVersion: version,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		ConfigPath: config.DefaultPath(),
	}

	if f.ConfigPath != "" {
		info.ConfigPath = f.ConfigPath
	}

	_, err := os.Stat(info.ConfigPath)
	info.ConfigFound = err == nil

	cfg, err := f.Config()
	if err == nil && cfg.Domain != "" {
		info.Domain = cfg.Domain
		info.APIStatus = checkAPI(cfg.Domain)
	}

	return info
}

func checkAPI(domain string) string {
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, domain, http.NoBody)
	if err != nil {
		return "unreachable"
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "unreachable"
	}
	resp.Body.Close()
	return "reachable"
}

// FormatText formats the diagnostic info as a human-readable string.
func FormatText(info Info) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CLI version:  %s\n", info.CLIVersion)
	fmt.Fprintf(&b, "Go version:   %s\n", info.GoVersion)
	fmt.Fprintf(&b, "OS/Arch:      %s/%s\n", info.OS, info.Arch)
	fmt.Fprintf(&b, "Config path:  %s\n", info.ConfigPath)
	fmt.Fprintf(&b, "Config found: %t\n", info.ConfigFound)
	if info.Domain != "" {
		fmt.Fprintf(&b, "TP domain:    %s\n", info.Domain)
		fmt.Fprintf(&b, "API status:   %s\n", info.APIStatus)
	}
	return b.String()
}

// BuildIssueURL constructs a pre-filled GitHub issue URL.
func BuildIssueURL(info Info) string {
	body := fmt.Sprintf(`## Environment

%s
## Description

<!-- Describe the bug -->

## Steps to Reproduce

1.

## Expected Behavior

<!-- What did you expect to happen? -->
`, "```\n"+FormatText(info)+"```")

	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "/lifedraft/targetprocess-cli/issues/new",
	}
	q := u.Query()
	q.Set("title", "Bug: [describe]")
	q.Set("labels", "bug")
	q.Set("template", "bug_report.yml")
	q.Set("body", body)
	u.RawQuery = q.Encode()

	result := u.String()
	if len(result) > 8000 {
		// Truncate body to stay under GitHub's URL limit.
		q.Set("body", "Environment info too long. Please paste from `tp bug-report`.")
		u.RawQuery = q.Encode()
		result = u.String()
	}
	return result
}

func openBrowser(ctx context.Context, rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", rawURL)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", rawURL)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Run()
}

func copyToClipboard(ctx context.Context, text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.CommandContext(ctx, "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// HandlePanic prints a friendly crash message and a pre-filled GitHub issue URL.
// Call this from a deferred recover() in main.
func HandlePanic(f *cmdutil.Factory, version string, recovered any) {
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)
	stack = stack[:n]

	fmt.Fprintf(os.Stderr, "\nWell, this is embarrassing. tp crashed.\n\n")
	fmt.Fprintf(os.Stderr, "panic: %v\n\n%s\n", recovered, stack)

	info := Collect(f, version)
	issueURL := BuildPanicIssueURL(info, recovered, stack)

	fmt.Fprintf(os.Stderr, "Please report this issue:\n  %s\n\n", issueURL)
	fmt.Fprintf(os.Stderr, "Or run: tp bug-report --mode open\n")
}

// BuildPanicIssueURL constructs a GitHub issue URL pre-filled with crash details.
func BuildPanicIssueURL(info Info, panicVal any, stack []byte) string {
	// Truncate stack to keep URL under limits.
	stackStr := string(stack)
	const maxStack = 1500
	if len(stackStr) > maxStack {
		stackStr = stackStr[:maxStack] + "\n... (truncated)"
	}

	body := fmt.Sprintf(`## Crash Report

tp panicked unexpectedly. Here's what happened:

### Panic
%s
%s
%s

### Environment

%s
%s
%s

### What were you doing?

<!-- What command did you run? Any other context? -->
`,
		"```", fmt.Sprintf("panic: %v", panicVal), "```",
		"```", FormatText(info), "```",
	)

	// Only include stack in body if URL won't exceed limit.
	bodyWithStack := fmt.Sprintf(`## Crash Report

tp panicked unexpectedly. Here's what happened:

### Panic
%s
panic: %v

%s
%s

### Environment

%s
%s
%s

### What were you doing?

<!-- What command did you run? Any other context? -->
`,
		"```", panicVal, stackStr, "```",
		"```", FormatText(info), "```",
	)

	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "/lifedraft/targetprocess-cli/issues/new",
	}
	q := u.Query()
	q.Set("title", fmt.Sprintf("Crash: %v", panicVal))
	q.Set("labels", "bug,crash")

	// Try with stack first, fall back to without if too long.
	q.Set("body", bodyWithStack)
	u.RawQuery = q.Encode()
	result := u.String()

	if len(result) > 8000 {
		q.Set("body", body)
		u.RawQuery = q.Encode()
		result = u.String()
	}

	return result
}

// NewCmd creates the bug-report command.
func NewCmd(f *cmdutil.Factory, version string) *cli.Command {
	return &cli.Command{
		Name:  "bug-report",
		Usage: "File a bug report with pre-filled environment info",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "mode",
				Usage: "Output mode: terminal, open, clipboard, json",
				Value: "terminal",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			info := Collect(f, version)
			mode := cmd.String("mode")

			switch mode {
			case "terminal":
				fmt.Print(FormatText(info))
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			case "open":
				issueURL := BuildIssueURL(info)
				fmt.Fprintln(os.Stderr, "Opening GitHub issue form in your browser...")
				return openBrowser(ctx, issueURL)
			case "clipboard":
				text := FormatText(info)
				if err := copyToClipboard(ctx, text); err != nil {
					return fmt.Errorf("copying to clipboard: %w", err)
				}
				fmt.Fprintln(os.Stderr, "Diagnostic info copied to clipboard.")
			default:
				return fmt.Errorf("unknown mode %q (valid: terminal, open, clipboard, json)", mode)
			}
			return nil
		},
	}
}
