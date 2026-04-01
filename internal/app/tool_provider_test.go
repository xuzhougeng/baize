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
