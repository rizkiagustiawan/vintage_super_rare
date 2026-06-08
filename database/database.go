package database

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type Database struct {
	mu       sync.RWMutex
	seen     map[string]bool
	filePath string
	logger   *slog.Logger
}

func New(filePath string, logger *slog.Logger) (*Database, error) {
	db := &Database{
		seen:     make(map[string]bool),
		filePath: filePath,
		logger:   logger,
	}

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read database file: %w", err)
		}

		var ids []string
		if err := json.Unmarshal(data, &ids); err != nil {
			return nil, fmt.Errorf("parse database file: %w", err)
		}

		for _, id := range ids {
			db.seen[id] = true
		}
		logger.Info("loaded database", slog.Int("count", len(ids)))
	}

	return db, nil
}

func (db *Database) IsNew(id string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return !db.seen[id]
}

func (db *Database) MarkSeen(id string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.seen[id] = true
}

func (db *Database) Save() error {
	db.mu.RLock()
	ids := make([]string, 0, len(db.seen))
	for id := range db.seen {
		ids = append(ids, id)
	}
	db.mu.RUnlock()

	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal database: %w", err)
	}

	tmpPath := db.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, db.filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	db.logger.Info("saved database", slog.Int("count", len(ids)))
	return nil
}

func (db *Database) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.seen)
}
