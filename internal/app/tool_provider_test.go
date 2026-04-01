package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"myclaw/internal/dirlist"
	"myclaw/internal/filesearch"
	"myclaw/internal/knowledge"
)

func TestLocalToolSideEffectLabels(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "app.db"))
	service := NewService(store, nil, nil)
	ctx := context.Background()
	mc := MessageContext{}

	defs, err := service.toolProviders.Definitions(ctx, mc)
	if err != nil {
		t.Fatalf("Definitions() failed: %v", err)
	}

	// Strip provider prefix and build a lookup map.
	levelByTool := make(map[string]string, len(defs))
	for _, def := range defs {
		_, name, ok := strings.Cut(def.Name, "::")
		if !ok {
			name = def.Name
		}
		levelByTool[name] = def.SideEffectLevel
	}

	want := map[string]string{
		"knowledge_search":        string(ToolSideEffectReadOnly),
		dirlist.ToolName:          string(ToolSideEffectReadOnly),
		filesearch.ToolName:       string(ToolSideEffectReadOnly),
		"readonly_system_command": string(ToolSideEffectReadOnly),
		"reminder_list":           string(ToolSideEffectReadOnly),
		"remember":                string(ToolSideEffectSoftWrite),
		"append_knowledge":        string(ToolSideEffectSoftWrite),
		"reminder_add":            string(ToolSideEffectSoftWrite),
		"forget_knowledge":        string(ToolSideEffectDestructive),
		"reminder_remove":         string(ToolSideEffectDestructive),
	}

	for tool, wantLevel := range want {
		got, ok := levelByTool[tool]
		if !ok {
			t.Errorf("tool %q not found in definitions", tool)
			continue
		}
		if got != wantLevel {
			t.Errorf("tool %q SideEffectLevel = %q, want %q", tool, got, wantLevel)
		}
	}
}

func TestReadonlySystemCommandNotExposedOnWeixin(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "app.db"))
	service := NewService(store, nil, nil)

	defs, err := service.toolProviders.Definitions(context.Background(), MessageContext{Interface: "weixin"})
	if err != nil {
		t.Fatalf("Definitions() failed: %v", err)
	}
	for _, def := range defs {
		if strings.HasSuffix(def.Name, "::readonly_system_command") {
			t.Fatalf("unexpected readonly system tool in weixin definitions: %#v", def)
		}
		if strings.HasSuffix(def.Name, "::list_directory") {
			t.Fatalf("unexpected directory listing tool in weixin definitions: %#v", def)
		}
	}
}

func TestDisabledAgentToolIsFilteredFromDefinitions(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "app.db"))
	service := NewService(store, nil, nil)
	service.SetDisabledAgentTools([]string{"local::everything_file_search"})

	allDefs, err := service.ListAllAgentToolDefinitions(context.Background(), MessageContext{})
	if err != nil {
		t.Fatalf("ListAllAgentToolDefinitions() failed: %v", err)
	}
	filteredDefs, err := service.ListAgentToolDefinitions(context.Background(), MessageContext{})
	if err != nil {
		t.Fatalf("ListAgentToolDefinitions() failed: %v", err)
	}

	var allHasFileSearch bool
	for _, def := range allDefs {
		if def.Name == "local::everything_file_search" {
			allHasFileSearch = true
			break
		}
	}
	if !allHasFileSearch {
		t.Fatalf("expected disabled tool to remain visible in all definitions: %#v", allDefs)
	}

	for _, def := range filteredDefs {
		if def.Name == "local::everything_file_search" {
			t.Fatalf("expected disabled tool to be filtered out: %#v", filteredDefs)
		}
	}
}

func TestExecuteAgentToolRejectsDisabledTool(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "app.db"))
	service := NewService(store, nil, nil)
	service.SetDisabledAgentTools([]string{"local::everything_file_search"})

	_, err := service.ExecuteAgentTool(context.Background(), MessageContext{}, "local::everything_file_search", `{"query":"report.csv"}`)
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled tool error, got %v", err)
	}
}
