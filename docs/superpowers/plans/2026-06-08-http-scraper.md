# HTTP Scraper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace chromedp (Chrome automation) with net/http + goquery for faster, lighter scraping without Chrome dependency.

**Architecture:** Replace browser-based scraping with direct HTTP requests. Carousell returns HTML directly, no JavaScript rendering needed. Use standard net/http client with goquery for HTML parsing.

**Tech Stack:** Go standard library net/http, goquery (already in project)

---

## File Structure

```
carousell_bot/
├── scraper/
│   ├── scraper.go       # Rewrite: HTTP client instead of chromedp
│   └── scraper_test.go  # Update tests
├── main.go              # Remove chromedp allocator setup
├── go.mod               # Remove chromedp dependency
└── go.sum
```

---

## Task 1: Rewrite Scraper with HTTP Client

**Files:**
- Modify: `scraper/scraper.go`
- Modify: `scraper/scraper_test.go`

- [ ] **Step 1: Update scraper.go with HTTP client**

Replace the entire `scraper/scraper.go` with:

```go
package scraper

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	httpClient *http.Client
	logger     *slog.Logger
	blacklist  []string
}

func New(logger *slog.Logger, blacklist []string) *Scraper {
	return &Scraper{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		blacklist: blacklist,
	}
}

func (s *Scraper) Fetch(ctx context.Context, brand string) ([]Listing, error) {
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", brand)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
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

- [ ] **Step 2: Update scraper_test.go**

```go
package scraper

import (
	"log/slog"
	"testing"
)

func TestScraper_IsBlacklisted(t *testing.T) {
	logger := slog.Default()
	s := New(logger, []string{"repro", "bootleg", "fake"})

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

- [ ] **Step 3: Run tests**

```bash
go test ./scraper/... -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add scraper/
git commit -m "feat: replace chromedp with net/http for lighter scraping"
```

---

## Task 2: Update Main to Remove chromedp

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update main.go**

Remove chromedp imports and allocator setup. Change scraper initialization:

```go
// Remove these imports:
// "github.com/chromedp/chromedp"

// Remove Chrome allocator setup (lines 51-59):
// opts := append(chromedp.DefaultExecAllocatorOptions[:], ...)
// allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
// defer allocCancel()

// Change scraper initialization:
// Before: sc := scraper.New(allocCtx, logger, cfg.Blacklist, cfg.PageTimeout)
// After:
sc := scraper.New(logger, cfg.Blacklist)
```

- [ ] **Step 2: Update config to remove PageTimeout**

In `config/config.go`, remove `PageTimeout` field:
```go
type Config struct {
	TelegramToken  string        `env:"TELEGRAM_TOKEN"`
	TelegramChatID int64         `env:"TELEGRAM_CHAT_ID,required"`
	BrandFile      string        `env:"BRAND_FILE,default=brand_list.txt"`
	DatabaseFile   string        `env:"DATABASE_FILE,default=seen_db.json"`
	Workers        int           `env:"WORKERS,default=5"`
	MinDelay       time.Duration `env:"MIN_DELAY,default=2s"`
	CycleDelay     time.Duration `env:"CYCLE_DELAY,default=15s"`
	Blacklist      []string      `env:"BLACKLIST,default=repro,bootleg,kaos,custom,premium,fake"`
}
```

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add main.go config/
git commit -m "refactor: remove chromedp dependency from main"
```

---

## Task 3: Clean Up Dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Remove chromedp dependency**

```bash
go mod tidy
```

- [ ] **Step 2: Verify build**

```bash
go build -o bot .
```

Expected: Build successful

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: remove chromedp dependency"
```

---

## Task 4: Test with Real Carousell

**Files:**
- None (manual testing)

- [ ] **Step 1: Run bot for 30 seconds**

```bash
timeout 30 ./bot 2>&1 | head -100
```

Expected: Bot starts, fetches listings, logs results

- [ ] **Step 2: Verify output**

Check for:
- No "context canceled" errors
- Listings found (count > 0)
- Structured JSON logging works

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
- [ ] chromedp removed from go.mod
- [ ] Bot runs without Chrome installed
- [ ] Listings fetched successfully
