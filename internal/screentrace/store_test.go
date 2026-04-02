package screentrace

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRecordDigestAndCleanup(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "app.db"))
	ctx := context.Background()
	base := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)

	first, err := store.AddRecord(ctx, Record{
		CapturedAt:   base,
		ImagePath:    "/tmp/shot-1.jpg",
		ImageHash:    "0000000000000000",
		Width:        1200,
		Height:       800,
		DisplayIndex: 0,
		SceneSummary: "current_user 在编辑代码",
		Apps:         []string{"VS Code"},
		Keywords:     []string{"coding"},
	})
	if err != nil {
		t.Fatalf("add first record: %v", err)
	}
	if first.ID == "" {
		t.Fatal("expected record id")
	}

	second, err := store.AddRecord(ctx, Record{
		CapturedAt:   base.Add(5 * time.Minute),
		ImagePath:    "/tmp/shot-2.jpg",
		ImageHash:    "ffffffffffffffff",
		Width:        1200,
		Height:       800,
		DisplayIndex: 0,
		SceneSummary: "current_user 在看浏览器文档",
		Apps:         []string{"Chrome"},
		Keywords:     []string{"docs"},
	})
	if err != nil {
		t.Fatalf("add second record: %v", err)
	}

	latest, ok, err := store.LatestRecord(ctx)
	if err != nil {
		t.Fatalf("latest record: %v", err)
	}
	if !ok || latest.ID != second.ID {
		t.Fatalf("unexpected latest record: %#v ok=%t", latest, ok)
	}

	records, err := store.ListRecordsBetween(ctx, base.Add(-time.Minute), base.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("list between: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	digest, err := store.UpsertDigest(ctx, Digest{
		BucketStart:      base,
		BucketEnd:        base.Add(15 * time.Minute),
		RecordCount:      2,
		Summary:          "这一段时间主要在编码和查文档。",
		Keywords:         []string{"coding", "docs"},
		DominantApps:     []string{"VS Code", "Chrome"},
		DominantTasks:    []string{"写代码", "查资料"},
		WrittenToKB:      true,
		KnowledgeEntryID: "abcd1234",
	})
	if err != nil {
		t.Fatalf("upsert digest: %v", err)
	}
	if digest.ID == "" {
		t.Fatal("expected digest id")
	}

	loadedDigest, ok, err := store.GetDigestByBucket(ctx, base)
	if err != nil {
		t.Fatalf("get digest: %v", err)
	}
	if !ok || loadedDigest.ID != digest.ID || !loadedDigest.WrittenToKB {
		t.Fatalf("unexpected loaded digest: %#v ok=%t", loadedDigest, ok)
	}

	paths, err := store.DeleteRecordsBefore(ctx, base.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if len(paths) != 1 || paths[0] != first.ImagePath {
		t.Fatalf("unexpected deleted paths: %#v", paths)
	}

	count, err := store.CountRecords(ctx)
	if err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 remaining record, got %d", count)
	}
}
