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
