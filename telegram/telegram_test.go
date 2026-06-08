package telegram

import (
	"log/slog"
	"testing"

	"carousell-bot/scraper"
)

func TestBot_DisabledMode(t *testing.T) {
	logger := slog.Default()

	bot, err := New("", 123456789, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bot.disabled {
		t.Error("bot should be disabled when token is empty")
	}

	err = bot.Notify(scraper.Listing{
		ID:    "12345",
		Title: "Test Item",
		Price: "Rp 100.000",
		Brand: "test-brand",
	})
	if err != nil {
		t.Fatalf("Notify() error: %v", err)
	}
}
