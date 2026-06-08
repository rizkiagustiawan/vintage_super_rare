package database

import (
	"log/slog"
	"path/filepath"
	"testing"
)

func TestDatabase_New(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db.json")
	logger := slog.Default()

	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if db.Count() != 0 {
		t.Errorf("Count() = %d, want 0", db.Count())
	}
}

func TestDatabase_IsNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db.json")
	logger := slog.Default()

	db, _ := New(dbPath, logger)

	if !db.IsNew("12345") {
		t.Error("IsNew('12345') = false, want true")
	}

	db.MarkSeen("12345")

	if db.IsNew("12345") {
		t.Error("IsNew('12345') = true after MarkSeen, want false")
	}
}

func TestDatabase_Save(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db.json")
	logger := slog.Default()

	db, _ := New(dbPath, logger)
	db.MarkSeen("11111")
	db.MarkSeen("22222")

	err := db.Save()
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load fresh database from file
	db2, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if db2.IsNew("11111") {
		t.Error("loaded db should have '11111'")
	}
	if db2.Count() != 2 {
		t.Errorf("loaded db Count() = %d, want 2", db2.Count())
	}
}
