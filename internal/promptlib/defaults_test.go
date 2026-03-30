package promptlib

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedDefaultPromptsAddsBuiltinPromptOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(filepath.Join(root, "prompts.json"))
	marker := filepath.Join(root, ".seeded-v1")

	if err := SeedDefaultPrompts(context.Background(), store, marker); err != nil {
		t.Fatalf("seed default prompts: %v", err)
	}

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 builtin prompt, got %d", len(items))
	}
	if items[0].Title != "5题联想式心理测试" {
		t.Fatalf("unexpected builtin prompt title: %q", items[0].Title)
	}

	if err := SeedDefaultPrompts(context.Background(), store, marker); err != nil {
		t.Fatalf("seed default prompts again: %v", err)
	}

	items, err = store.List(context.Background())
	if err != nil {
		t.Fatalf("list prompts after reseed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected builtin prompt to be seeded once, got %d prompts", len(items))
	}
}

func TestSeedDefaultPromptsMergesWithoutDuplicatingTitles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(filepath.Join(root, "prompts.json"))
	marker := filepath.Join(root, ".seeded-v1")

	if _, err := store.Add(context.Background(), Prompt{
		Title:   "5题联想式心理测试",
		Content: "custom",
	}); err != nil {
		t.Fatalf("add custom prompt: %v", err)
	}
	if _, err := store.Add(context.Background(), Prompt{
		Title:   "日报整理",
		Content: "整理日报",
	}); err != nil {
		t.Fatalf("add second prompt: %v", err)
	}

	if err := SeedDefaultPrompts(context.Background(), store, marker); err != nil {
		t.Fatalf("seed default prompts: %v", err)
	}

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected existing prompts to stay without duplicates, got %d", len(items))
	}

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected marker file to be created: %v", err)
	}
}
