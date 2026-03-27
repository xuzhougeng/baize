package knowledge

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAddListAndClear(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "entries.json"))
	ctx := context.Background()

	if _, err := store.Add(ctx, Entry{
		Text:       "first",
		RecordedAt: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add first: %v", err)
	}
	if _, err := store.Add(ctx, Entry{
		Text:       "second",
		RecordedAt: time.Date(2026, 3, 27, 11, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add second: %v", err)
	}

	entries, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Text != "first" || entries[1].Text != "second" {
		t.Fatalf("unexpected order: %#v", entries)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	entries, err = store.List(ctx)
	if err != nil {
		t.Fatalf("list after clear: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty store after clear, got %d entries", len(entries))
	}
}

func TestStoreRemoveByPrefix(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "entries.json"))
	ctx := context.Background()

	entry, err := store.Add(ctx, Entry{
		ID:         "0015f908abcd1234",
		Text:       "drink water",
		RecordedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("add entry: %v", err)
	}

	removed, ok, err := store.Remove(ctx, "#0015f908")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !ok {
		t.Fatalf("expected entry to be removed")
	}
	if removed.ID != entry.ID {
		t.Fatalf("unexpected removed entry: %#v", removed)
	}

	entries, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty store after remove, got %d entries", len(entries))
	}
}
