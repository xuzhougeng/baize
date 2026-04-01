package systemcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"myclaw/internal/toolcontract"
)

const (
	ToolName             = "readonly_system_command"
	defaultTimeout       = 5 * time.Second
	maxTimeout           = 10 * time.Second
	maxOutputPreviewRune = 12000
)

var (
	currentGOOS = func() string { return runtime.GOOS }
	runCommand  = execCommand
)

type ToolInput struct {
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

type ToolResult struct {
	Tool      string   `json:"tool"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	ExitCode  int      `json:"exit_code"`
	Stdout    string   `json:"stdout,omitempty"`
	Stderr    string   `json:"stderr,omitempty"`
	Truncated bool     `json:"truncated,omitempty"`
}

type commandSpec struct {
	Name         string
	ArgVariants  [][]string
	UsageExample string
}

func Definition() toolcontract.Spec {
	return toolcontract.Spec{
		Name:              ToolName,
		Purpose:           "Run a small allowlisted set of read-only local system commands for machine inspection.",
		Description:       "Execute one approved read-only OS command without a shell. Useful for checking host, user, uptime, processes, disk, or basic platform status.",
		InputContract:     "Provide {\"command\":\"...\"} and optionally {\"args\":[...],\"timeout_seconds\":5}. Only allowlisted commands and argument variants are accepted for the current OS, and WeChat does not expose this tool.",
		OutputContract:    "Returns JSON with tool, command, args, exit_code, stdout, stderr, and truncated.",
		InputJSONExample:  `{"command":"hostname"}`,
		OutputJSONExample: `{"tool":"readonly_system_command","command":"hostname","args":[],"exit_code":0,"stdout":"my-workstation\n","stderr":""}`,
		Usage:             UsageText(),
	}
}

func UsageText() string {
	names := availableCommandNames(currentGOOS())
	if len(names) == 0 {
		return "No read-only system commands are enabled on this platform."
	}
	return strings.TrimSpace(fmt.Sprintf(`
Tool: %s
Purpose: inspect the local machine by running one allowlisted read-only OS command without a shell.

Input:
- command: required allowlisted command name for the current platform
- args: optional exact argument variant allowed for that command
- timeout_seconds: optional timeout in seconds, clamped to 1-10 and defaulting to 5

Current platform command allowlist:
- %s

Use this only when the user asks about the current machine state and command output is needed before answering.
`, ToolName, strings.Join(names, "\n- ")))
}

func AllowedForInterface(name string) bool {
	return !strings.EqualFold(strings.TrimSpace(name), "weixin")
}

func SupportedForCurrentPlatform() bool {
	return len(commandCatalog(currentGOOS())) > 0
}

func Execute(ctx context.Context, input ToolInput) (ToolResult, error) {
	input = NormalizeInput(input)

	spec, ok := commandCatalog(currentGOOS())[input.Command]
	if !ok {
		return ToolResult{}, fmt.Errorf("command %q is not allowed on %s", input.Command, currentGOOS())
	}
	if err := validateArgs(spec, input.Args); err != nil {
		return ToolResult{}, err
	}

	timeout := defaultTimeout
	if input.TimeoutSeconds > 0 {
		timeout = time.Duration(input.TimeoutSeconds) * time.Second
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, stderr, exitCode, err := runCommand(execCtx, spec.Name, input.Args...)
	result := ToolResult{
		Tool:     ToolName,
		Command:  spec.Name,
		Args:     append([]string(nil), input.Args...),
		ExitCode: exitCode,
	}
	result.Stdout, result.Truncated = truncateText(stdout, maxOutputPreviewRune)
	stderrText, stderrTruncated := truncateText(stderr, maxOutputPreviewRune)
	result.Stderr, result.Truncated = combineTruncation(result.Truncated, stderrText, stderrTruncated)

	switch {
	case err == nil:
		return result, nil
	case errors.Is(execCtx.Err(), context.DeadlineExceeded):
		return result, fmt.Errorf("command %q timed out after %s", spec.Name, timeout)
	case errors.Is(err, exec.ErrNotFound):
		return result, fmt.Errorf("command %q is not available on this machine", spec.Name)
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return result, nil
		}
		return result, fmt.Errorf("run command %q: %w", spec.Name, err)
	}
}

func FormatResult(result ToolResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func NormalizeInput(raw ToolInput) ToolInput {
	out := ToolInput{
		Command:        strings.ToLower(strings.TrimSpace(raw.Command)),
		TimeoutSeconds: raw.TimeoutSeconds,
	}
	for _, arg := range raw.Args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		out.Args = append(out.Args, arg)
	}
	return out
}

func availableCommandNames(goos string) []string {
	catalog := commandCatalog(goos)
	out := make([]string, 0, len(catalog))
	for _, spec := range catalog {
		label := spec.Name
		if usage := strings.TrimSpace(spec.UsageExample); usage != "" {
			label += " " + usage
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func commandCatalog(goos string) map[string]commandSpec {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return map[string]commandSpec{
			"hostname":   {Name: "hostname", ArgVariants: [][]string{{}}},
			"whoami":     {Name: "whoami", ArgVariants: [][]string{{}}},
			"systeminfo": {Name: "systeminfo", ArgVariants: [][]string{{}}},
			"tasklist":   {Name: "tasklist", ArgVariants: [][]string{{}}},
			"ipconfig":   {Name: "ipconfig", ArgVariants: [][]string{{}, {"/all"}}, UsageExample: "/all"},
		}
	case "darwin":
		return map[string]commandSpec{
			"hostname": {Name: "hostname", ArgVariants: [][]string{{}}},
			"whoami":   {Name: "whoami", ArgVariants: [][]string{{}}},
			"date":     {Name: "date", ArgVariants: [][]string{{}}},
			"uname":    {Name: "uname", ArgVariants: [][]string{{}, {"-a"}}, UsageExample: "-a"},
			"uptime":   {Name: "uptime", ArgVariants: [][]string{{}}},
			"pwd":      {Name: "pwd", ArgVariants: [][]string{{}}},
			"df":       {Name: "df", ArgVariants: [][]string{{}, {"-h"}}, UsageExample: "-h"},
			"ps":       {Name: "ps", ArgVariants: [][]string{{"-ef"}, {"aux"}}, UsageExample: "-ef"},
		}
	case "linux":
		return map[string]commandSpec{
			"hostname": {Name: "hostname", ArgVariants: [][]string{{}}},
			"whoami":   {Name: "whoami", ArgVariants: [][]string{{}}},
			"date":     {Name: "date", ArgVariants: [][]string{{}}},
			"uname":    {Name: "uname", ArgVariants: [][]string{{}, {"-a"}}, UsageExample: "-a"},
			"uptime":   {Name: "uptime", ArgVariants: [][]string{{}}},
			"pwd":      {Name: "pwd", ArgVariants: [][]string{{}}},
			"df":       {Name: "df", ArgVariants: [][]string{{}, {"-h"}}, UsageExample: "-h"},
			"free":     {Name: "free", ArgVariants: [][]string{{}, {"-h"}}, UsageExample: "-h"},
			"ps":       {Name: "ps", ArgVariants: [][]string{{"-ef"}, {"aux"}}, UsageExample: "-ef"},
		}
	default:
		return map[string]commandSpec{}
	}
}

func validateArgs(spec commandSpec, args []string) error {
	if len(spec.ArgVariants) == 0 && len(args) == 0 {
		return nil
	}
	for _, allowed := range spec.ArgVariants {
		if slices.Equal(args, allowed) {
			return nil
		}
	}
	var variants []string
	for _, allowed := range spec.ArgVariants {
		if len(allowed) == 0 {
			variants = append(variants, "(no args)")
			continue
		}
		variants = append(variants, strings.Join(allowed, " "))
	}
	return fmt.Errorf("command %q only allows these args: %s", spec.Name, strings.Join(variants, ", "))
}

func truncateText(value string, limit int) (string, bool) {
	runes := []rune(value)
	if len(runes) <= limit {
		return value, false
	}
	return string(runes[:limit]) + "\n...[truncated]", true
}

func combineTruncation(current bool, value string, truncated bool) (string, bool) {
	return value, current || truncated
}

func execCommand(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}
