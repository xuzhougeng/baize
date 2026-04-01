package sqliteutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbMu sync.Mutex
	dbs  = make(map[string]*sql.DB)
)

func Open(path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("sqlite path is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	if db := dbs[absPath]; db != nil {
		return db, nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	dsn := absPath + "?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	// Keep no idle connections so temporary databases are not left locked on Windows
	// after a test has finished using them.
	db.SetMaxIdleConns(0)

	if err := configure(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	dbs[absPath] = db
	return db, nil
}

func Close(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	dbMu.Lock()
	db := dbs[absPath]
	delete(dbs, absPath)
	dbMu.Unlock()

	if db == nil {
		return nil
	}
	return db.Close()
}

func CloseAll() error {
	dbMu.Lock()
	items := make([]*sql.DB, 0, len(dbs))
	for path, db := range dbs {
		delete(dbs, path)
		items = append(items, db)
	}
	dbMu.Unlock()

	var firstErr error
	for _, db := range items {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func configure(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
	}
	for _, query := range pragmas {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
