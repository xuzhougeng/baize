package windowsautomationtool

import (
	"context"
	"strings"
	"testing"
)

func TestDefinitionHasFamilyMetadata(t *testing.T) {
	t.Parallel()

	spec := Definition().Normalized()
	if spec.Name != ToolName {
		t.Fatalf("unexpected tool name: %#v", spec)
	}
	if spec.FamilyKey != ToolFamilyKey || spec.FamilyTitle != ToolFamilyTitle {
		t.Fatalf("unexpected family metadata: %#v", spec)
	}
	if spec.DisplayTitle == "" || spec.OutputJSONExample == "" {
		t.Fatalf("expected display title and output example: %#v", spec)
	}
}

func TestNormalizeInputClampsLimit(t *testing.T) {
	t.Parallel()

	got := NormalizeInput(ToolInput{Action: " list_windows ", Limit: 999})
	if got.Action != "list_windows" {
		t.Fatalf("NormalizeInput().Action = %q", got.Action)
	}
	if got.Limit != maxWindowLimit {
		t.Fatalf("NormalizeInput().Limit = %d, want %d", got.Limit, maxWindowLimit)
	}

	got = NormalizeInput(ToolInput{})
	if got.Limit != defaultWindowLimit {
		t.Fatalf("NormalizeInput().Limit = %d, want %d", got.Limit, defaultWindowLimit)
	}
}

func TestExecuteUsesPowerShellForFrontmostWindow(t *testing.T) {
	oldGOOS := currentGOOS
	oldRun := runCommand
	oldProgram := powerShellProgram
	currentGOOS = func() string { return "windows" }
	powerShellProgram = func() string { return "powershell" }
	runCommand = func(_ context.Context, name string, args ...string) (string, string, int, error) {
		if name != "powershell" {
			t.Fatalf("unexpected runner: %q", name)
		}
		if len(args) != 4 || args[0] != "-NoProfile" || args[1] != "-NonInteractive" || args[2] != "-Command" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if !strings.Contains(args[3], "Get-MyClawForegroundWindowInfo") {
			t.Fatalf("expected frontmost window script, got %q", args[3])
		}
		return `{"process_name":"Code","process_id":123,"title":"main.go"}`, "", 0, nil
	}
	t.Cleanup(func() {
		currentGOOS = oldGOOS
		runCommand = oldRun
		powerShellProgram = oldProgram
	})

	result, err := Execute(context.Background(), ToolInput{Action: "frontmost_window"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Tool != ToolName || result.Action != "frontmost_window" || result.Shell != "powershell" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Stdout, `"process_name":"Code"`) {
		t.Fatalf("unexpected stdout: %#v", result)
	}
}

func TestExecuteFocusWindowEscapesTitle(t *testing.T) {
	oldGOOS := currentGOOS
	oldRun := runCommand
	oldProgram := powerShellProgram
	currentGOOS = func() string { return "windows" }
	powerShellProgram = func() string { return "powershell" }
	runCommand = func(_ context.Context, _ string, args ...string) (string, string, int, error) {
		script := args[3]
		if !strings.Contains(script, "$needle = 'Bob''s Editor'") {
			t.Fatalf("expected escaped title in script, got %q", script)
		}
		return `{"success":true}`, "", 0, nil
	}
	t.Cleanup(func() {
		currentGOOS = oldGOOS
		runCommand = oldRun
		powerShellProgram = oldProgram
	})

	_, err := Execute(context.Background(), ToolInput{
		Action:        "focus_window",
		TitleContains: "Bob's Editor",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestExecuteRejectsMissingActionFields(t *testing.T) {
	oldGOOS := currentGOOS
	currentGOOS = func() string { return "windows" }
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	_, err := Execute(context.Background(), ToolInput{Action: "focus_app"})
	if err == nil || !strings.Contains(err.Error(), "process_name") {
		t.Fatalf("expected process_name validation error, got %v", err)
	}

	_, err = Execute(context.Background(), ToolInput{Action: "focus_window"})
	if err == nil || !strings.Contains(err.Error(), "title_contains") {
		t.Fatalf("expected title_contains validation error, got %v", err)
	}

	_, err = Execute(context.Background(), ToolInput{Action: "launch_or_focus_app"})
	if err == nil || !strings.Contains(err.Error(), "app_name") {
		t.Fatalf("expected app_name validation error, got %v", err)
	}
}

func TestSupportedForCurrentPlatform(t *testing.T) {
	oldGOOS := currentGOOS
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	currentGOOS = func() string { return "windows" }
	if !SupportedForCurrentPlatform() {
		t.Fatal("expected windows platform to be supported")
	}

	currentGOOS = func() string { return "linux" }
	if SupportedForCurrentPlatform() {
		t.Fatal("expected linux to be unsupported")
	}
}
