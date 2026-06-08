package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear all env vars
	os.Unsetenv("TELEGRAM_TOKEN")
	os.Unsetenv("TELEGRAM_CHAT_ID")
	os.Unsetenv("WORKERS")
	os.Unsetenv("MIN_DELAY")
	os.Unsetenv("CYCLE_DELAY")
	os.Unsetenv("PAGE_TIMEOUT")
	os.Unsetenv("BRAND_FILE")
	os.Unsetenv("DATABASE_FILE")
	os.Unsetenv("BLACKLIST")

	// Set required vars
	os.Setenv("TELEGRAM_CHAT_ID", "123456789")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

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
	os.Setenv("TELEGRAM_TOKEN", "test-token")
	os.Setenv("TELEGRAM_CHAT_ID", "987654321")
	os.Setenv("WORKERS", "3")
	os.Setenv("MIN_DELAY", "5s")
	defer func() {
		os.Unsetenv("TELEGRAM_TOKEN")
		os.Unsetenv("TELEGRAM_CHAT_ID")
		os.Unsetenv("WORKERS")
		os.Unsetenv("MIN_DELAY")
	}()

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
	os.Unsetenv("TELEGRAM_CHAT_ID")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TELEGRAM_CHAT_ID")
	}
}
