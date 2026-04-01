package systemcmd

import (
	"context"
	"testing"
)

func TestExecuteUsesAllowlistedCommand(t *testing.T) {
	t.Parallel()

	oldGOOS := currentGOOS
	oldRun := runCommand
	currentGOOS = func() string { return "linux" }
	runCommand = func(_ context.Context, name string, args ...string) (string, string, int, error) {
		if name != "uname" {
			t.Fatalf("unexpected command: %q", name)
		}
		if len(args) != 1 || args[0] != "-a" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return "Linux test-host\n", "", 0, nil
	}
	t.Cleanup(func() {
		currentGOOS = oldGOOS
		runCommand = oldRun
	})

	result, err := Execute(context.Background(), ToolInput{
		Command: "uname",
		Args:    []string{"-a"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Tool != ToolName || result.Command != "uname" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Stdout != "Linux test-host\n" {
		t.Fatalf("unexpected stdout: %#v", result)
	}
}

func TestExecuteRejectsDisallowedArgs(t *testing.T) {
	t.Parallel()

	oldGOOS := currentGOOS
	currentGOOS = func() string { return "linux" }
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	_, err := Execute(context.Background(), ToolInput{
		Command: "uname",
		Args:    []string{"-r"},
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected arg validation error")
	}
}

func TestAllowedForInterface(t *testing.T) {
	t.Parallel()

	if AllowedForInterface("weixin") {
		t.Fatal("expected weixin to be blocked")
	}
	if !AllowedForInterface("desktop") {
		t.Fatal("expected desktop to be allowed")
	}
	if !AllowedForInterface("") {
		t.Fatal("expected empty interface to be allowed")
	}
}

func TestSupportedForCurrentPlatform(t *testing.T) {
	t.Parallel()

	oldGOOS := currentGOOS
	t.Cleanup(func() {
		currentGOOS = oldGOOS
	})

	currentGOOS = func() string { return "linux" }
	if !SupportedForCurrentPlatform() {
		t.Fatal("expected linux platform to be supported")
	}

	currentGOOS = func() string { return "plan9" }
	if SupportedForCurrentPlatform() {
		t.Fatal("expected unsupported platform to be rejected")
	}
}
