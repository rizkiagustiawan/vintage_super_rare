package scraper

import (
	"log/slog"
	"testing"
)

func TestScraper_IsBlacklisted(t *testing.T) {
	logger := slog.Default()
	s, err := New(logger, []string{"repro", "bootleg", "fake"})
	if err != nil {
		t.Fatalf("failed to create scraper: %v", err)
	}
	defer s.Close()

	tests := []struct {
		title string
		want  bool
	}{
		{"Undercover Jacket", false},
		{"Repro Nike Hoodie", true},
		{"Bootleg Vintage Tee", true},
		{"Fake Gucci Bag", true},
		{"Premium Quality Item", false},
		{"Custom Made Ring", false},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := s.isBlacklisted(tt.title); got != tt.want {
				t.Errorf("isBlacklisted(%q) = %v, want %v", tt.title, got, tt.want)
			}
		})
	}
}
