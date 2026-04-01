package sqliteutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDoesNotKeepDatabaseFileLockedWhenIdle(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "app.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := Close(path); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS demo (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove idle sqlite db: %v", err)
	}
}
