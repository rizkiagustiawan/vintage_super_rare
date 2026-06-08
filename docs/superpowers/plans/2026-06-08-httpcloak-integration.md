# httpcloak Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace net/http with httpcloak for Cloudflare bypass using browser-identical TLS/HTTP2 fingerprinting.

**Architecture:** Replace standard net/http client with httpcloak client that mimics Chrome's TLS fingerprint, bypassing Cloudflare bot detection.

**Tech Stack:** httpcloak (github.com/sardanioss/httpcloak/client), goquery

---

## File Structure

```
carousell_bot/
├── scraper/
│   ├── scraper.go       # Rewrite: httpcloak client instead of net/http
│   └── scraper_test.go  # Update tests
├── go.mod               # Add httpcloak dependency
└── go.sum
```

---

## Task 1: Rewrite Scraper with httpcloak

**Files:**
- Modify: `scraper/scraper.go`
- Modify: `scraper/scraper_test.go`

- [ ] **Step 1: Add httpcloak dependency**

```bash
cd /home/awan/Documents/carousell_bot
go get github.com/sardanioss/httpcloak/client
```

- [ ] **Step 2: Update scraper.go with httpcloak client**

Replace `scraper/scraper.go` with:

```go
package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sardanioss/httpcloak/client"
)

type Listing struct {
	ID     string
	Title  string
	Price  string
	Brand  string
	Seller string
	Time   string
	URL    string
}

type Scraper struct {
	httpClient *client.Client
	logger     *slog.Logger
	blacklist  []string
}

func New(logger *slog.Logger, blacklist []string) (*Scraper, error) {
	c := client.NewClient("chrome-latest",
		client.WithTimeout(30*time.Second),
		client.WithRetry(2),
		client.WithRetryConfig(
			3,                      // Max retries
			500*time.Millisecond,   // Min backoff
			5*time.Second,          // Max backoff
			[]int{429, 503},        // Status codes to retry
		),
	)

	return &Scraper{
		httpClient: c,
		logger:     logger,
		blacklist:  blacklist,
	}, nil
}

func (s *Scraper) Close() {
	if s.httpClient != nil {
		s.httpClient.Close()
	}
}

func (s *Scraper) Fetch(ctx context.Context, brand string) ([]Listing, error) {
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", brand)

	resp, err := s.httpClient.Get(ctx, url, map[string][]string{
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language": {"en-US,en;q=0.5"},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := resp.Text()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var listings []Listing
	seenInPage := make(map[string]bool)

	doc.Find("a[href*='/p/']").Each(func(i int, sel *goquery.Selection) {
		link, exists := sel.Attr("href")
		if !exists {
			return
		}

		parts := strings.Split(strings.Trim(link, "/"), "-")
		if len(parts) == 0 {
			return
		}

		rawID := parts[len(parts)-1]
		idParts := strings.Split(rawID, "?")
		id := idParts[0]

		if strings.Contains(id, "tap_index") || len(id) < 5 {
			return
		}

		if seenInPage[id] {
			return
		}

		title := ""
		sel.Find("p").Each(func(_ int, p *goquery.Selection) {
			t := strings.TrimSpace(p.Text())
			if len(t) > 3 && !strings.Contains(t, "Rp") && title == "" {
				title = t
			}
		})

		if title == "" || s.isBlacklisted(title) {
			return
		}

		price := ""
		sel.Find("p, div, span").Each(func(_ int, node *goquery.Selection) {
			t := strings.TrimSpace(node.Text())
			if strings.Contains(t, "Rp") && price == "" {
				price = t
			}
		})

		if price != "" {
			seenInPage[id] = true
			listings = append(listings, Listing{
				ID:     id,
				Title:  title,
				Price:  price,
				Seller: "Unknown",
				Time:   "Recent",
				URL:    "https://id.carousell.com" + link,
				Brand:  brand,
			})
		}
	})

	s.logger.Info("fetched listings",
		slog.String("brand", brand),
		slog.Int("count", len(listings)),
	)

	return listings, nil
}

func (s *Scraper) isBlacklisted(title string) bool {
	t := strings.ToLower(title)
	for _, word := range s.blacklist {
		if strings.Contains(t, word) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Update scraper_test.go**

```go
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./scraper/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add scraper/ go.mod go.sum
git commit -m "feat: integrate httpcloak for Cloudflare bypass"
```

---

## Task 2: Update Main for httpcloak

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update main.go scraper initialization**

Change scraper initialization to handle error:

```go
// Before:
// sc := scraper.New(logger, cfg.Blacklist)

// After:
sc, err := scraper.New(logger, cfg.Blacklist)
if err != nil {
    logger.Error("failed to initialize scraper", slog.String("error", err.Error()))
    os.Exit(1)
}
defer sc.Close()
```

- [ ] **Step 2: Run all tests**

```bash
go test ./... -v
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "refactor: update main for httpcloak scraper"
```

---

## Task 3: Test with Real Carousell

**Files:**
- None (manual testing)

- [ ] **Step 1: Run bot for 30 seconds**

```bash
timeout 30 ./bot 2>&1 | head -100
```

Expected: Bot starts, fetches listings, finds items (count > 0)

- [ ] **Step 2: Verify Cloudflare bypass**

Check for:
- No "Just a moment..." in responses
- Listings found (count > 0)
- No 403/503 errors

- [ ] **Step 3: Final commit if needed**

```bash
git add -A
git commit -m "fix: adjustments after real-world testing"
```

---

## Verification Checklist

- [ ] All tests pass: `go test ./... -v`
- [ ] No vet warnings: `go vet ./...`
- [ ] Binary builds: `go build -o bot .`
- [ ] httpcloak in go.mod
- [ ] Bot runs without Chrome installed
- [ ] Listings fetched successfully (count > 0)
- [ ] Cloudflare bypass works
