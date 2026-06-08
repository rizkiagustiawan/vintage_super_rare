package config

import (
	"os"
	"testing"
	"time"
)

func unsetAll(t *testing.T) {
	t.Helper()
	vars := []string{
		"TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID", "WORKERS", "MIN_DELAY",
		"CYCLE_DELAY", "PAGE_TIMEOUT", "BRAND_FILE", "DATABASE_FILE", "BLACKLIST",
	}
	for _, v := range vars {
		old, wasSet := os.LookupEnv(v)
		os.Unsetenv(v)
		if wasSet {
			t.Cleanup(func() { os.Setenv(v, old) })
		}
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	unsetAll(t)

	t.Setenv("TELEGRAM_CHAT_ID", "123456789")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TelegramChatID != 123456789 {
		t.Errorf("TelegramChatID = %d, want 123456789", cfg.TelegramChatID)
	}
	if cfg.Workers != 5 {
		t.Errorf("Workers = %d, want 5", cfg.Workers)
	}
	if cfg.MinDelay != 2*time.Second {
		t.Errorf("MinDelay = %v, want 2s", cfg.MinDelay)
	}
	if cfg.CycleDelay != 15*time.Second {
		t.Errorf("CycleDelay = %v, want 15s", cfg.CycleDelay)
	}
	if cfg.PageTimeout != 45*time.Second {
		t.Errorf("PageTimeout = %v, want 45s", cfg.PageTimeout)
	}
	if cfg.BrandFile != "brand_list.txt" {
		t.Errorf("BrandFile = %s, want brand_list.txt", cfg.BrandFile)
	}
	if cfg.DatabaseFile != "seen_db.json" {
		t.Errorf("DatabaseFile = %s, want seen_db.json", cfg.DatabaseFile)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	unsetAll(t)

	t.Setenv("TELEGRAM_TOKEN", "test-token")
	t.Setenv("TELEGRAM_CHAT_ID", "987654321")
	t.Setenv("WORKERS", "3")
	t.Setenv("MIN_DELAY", "5s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TelegramToken != "test-token" {
		t.Errorf("TelegramToken = %s, want test-token", cfg.TelegramToken)
	}
	if cfg.Workers != 3 {
		t.Errorf("Workers = %d, want 3", cfg.Workers)
	}
	if cfg.MinDelay != 5*time.Second {
		t.Errorf("MinDelay = %v, want 5s", cfg.MinDelay)
	}
}

func TestLoad_MissingChatID(t *testing.T) {
	unsetAll(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TELEGRAM_CHAT_ID")
	}
}
