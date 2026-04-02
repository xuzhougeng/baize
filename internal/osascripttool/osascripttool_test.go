package osascripttool

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

func TestExecuteUsesOsaScriptForFrontmostApp(t *testing.T) {
	oldGOOS := currentGOOS
	oldRun := runCommand
	oldBinary := osascriptBinary
	currentGOOS = func() string { return "darwin" }
	osascriptBinary = func() string { return "osascript" }
	runCommand = func(_ context.Context, name string, args ...string) (string, string, int, error) {
		if name != "osascript" {
			t.Fatalf("unexpected runner: %q", name)
		}
		if len(args) != 2 || args[0] != "-e" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if !strings.Contains(args[1], `first application process whose frontmost is true`) {
			t.Fatalf("expected frontmost app script, got %q", args[1])
		}
		return "Visual Studio Code\n", "", 0, nil
	}
	t.Cleanup(func() {
		currentGOOS = oldGOOS
		runCommand = oldRun
		osascriptBinary = oldBinary
	})

	result, err := Execute(context.Background(), ToolInput{Action: "frontmost_app"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Tool != ToolName || result.Action != "frontmost_app" || result.Shell != "osascript" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Stdout != "Visual Studio Code\n" {
		t.Fatalf("unexpected stdout: %#v", result)
	}
}

func TestExecuteActivateAppBuildsScript(t *testing.T) {
	oldGOOS := currentGOOS
	oldRun := runCommand
	oldBinary := osascriptBinary
	currentGOOS = func() string { return "darwin" }
	osascriptBinary = func() string { return "osascript" }
	runCommand = func(_ context.Context, _ string, args ...string) (string, string, int, error) {
		script := args[1]
		if !strings.Contains(script, `set appName to "Safari"`) {
			t.Fatalf("expected app name in script, got %q", script)
		}
		if !strings.Contains(script, `tell application appName to activate`) {
			t.Fatalf("expected activate script, got %q", script)
		}
		return "activated:Safari\n", "", 0, nil
	}
	t.Cleanup(func() {
		currentGOOS = oldGOOS
		runCommand = oldRun
		osascriptBinary = oldBinary
	})

	result, err := Execute(context.Background(), ToolInput{
		Action:  "activate_app",
		AppName: "Safari",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(result.Stdout, "activated:Safari") {
		t.Fatalf("unexpected stdout: %#v", result)
	}
}

func TestExecuteRejectsMissingAppName(t *testing.T) {
	oldGOOS := currentGOOS
	currentGOOS = func() string { return "darwin" }
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	_, err := Execute(context.Background(), ToolInput{Action: "activate_app"})
	if err == nil || !strings.Contains(err.Error(), "app_name") {
		t.Fatalf("expected app_name validation error, got %v", err)
	}

	_, err = Execute(context.Background(), ToolInput{Action: "open_or_activate_app"})
	if err == nil || !strings.Contains(err.Error(), "app_name") {
		t.Fatalf("expected app_name validation error, got %v", err)
	}
}

func TestSupportedForCurrentPlatform(t *testing.T) {
	oldGOOS := currentGOOS
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	currentGOOS = func() string { return "darwin" }
	if !SupportedForCurrentPlatform() {
		t.Fatal("expected darwin platform to be supported")
	}

	currentGOOS = func() string { return "linux" }
	if SupportedForCurrentPlatform() {
		t.Fatal("expected linux to be unsupported")
	}
}
