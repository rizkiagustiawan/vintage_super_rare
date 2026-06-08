# Carousell Bot Modernization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Modernize the Carousell bot from a single-file Go script to a well-structured, modern Go application with proper error handling, structured logging, graceful shutdown, and comprehensive testing.

**Architecture:** Split into 4 packages (config, database, scraper, telegram) with clean interfaces, dependency injection, and context propagation throughout.

**Tech Stack:** Go 1.26+, log/slog, godotenv, go-envconfig, cenkalti/backoff/v4, chromedp, goquery, telegram-bot-api/v5

---

## File Structure

```
carousell_bot/
├── main.go              # Entry point, signal handling, orchestration
├── config/
│   ├── config.go        # Config struct, Load() from env/.env
│   └── config_test.go   # Unit tests
├── scraper/
│   ├── scraper.go       # Carousell scraper logic
│   └── scraper_test.go  # Unit tests (mock HTML)
├── telegram/
│   ├── telegram.go      # Telegram bot init, notify
│   └── telegram_test.go # Unit tests
├── database/
│   ├── database.go      # JSON persistence, seen listings
│   └── database_test.go # Unit tests
├── brand_list.txt       # Existing file (unchanged)
├── .env.example         # Template untuk env vars
├── .gitignore
├── go.mod
└── go.sum
```

---

## Task 1: Setup Project Structure & Dependencies

**Files:**
- Modify: `go.mod`
- Create: `.env.example`
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: Update go.mod with new dependencies**

```bash
cd /home/awan/Documents/carousell_bot
go get github.com/joho/godotenv@v1.5.1
go get github.com/sethvargo/go-envconfig@v1.1.0
go get github.com/cenkalti/backoff/v4@v4.3.0
```

- [ ] **Step 2: Create .env.example**

```env
# Telegram Configuration
TELEGRAM_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here

# Scraper Configuration
WORKERS=5
MIN_DELAY=2s
CYCLE_DELAY=15s
PAGE_TIMEOUT=45s

# File Paths
BRAND_FILE=brand_list.txt
DATABASE_FILE=seen_db.json

# Blacklist (comma-separated)
BLACKLIST=repro,bootleg,kaos,custom,premium,fake
```

- [ ] **Step 3: Write failing test for config loading**

```go
// config/config_test.go
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
```

- [ ] **Step 4: Run test to verify it fails**

```bash
go test ./config/... -v
```

Expected: FAIL with "package config is not in GOROOT"

- [ ] **Step 5: Implement config package**

```go
// config/config.go
package config

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	TelegramToken  string        `env:"TELEGRAM_TOKEN"`
	TelegramChatID int64         `env:"TELEGRAM_CHAT_ID,required"`
	BrandFile      string        `env:"BRAND_FILE,default=brand_list.txt"`
	DatabaseFile   string        `env:"DATABASE_FILE,default=seen_db.json"`
	Workers        int           `env:"WORKERS,default=5"`
	MinDelay       time.Duration `env:"MIN_DELAY,default=2s"`
	CycleDelay     time.Duration `env:"CYCLE_DELAY,default=15s"`
	PageTimeout    time.Duration `env:"PAGE_TIMEOUT,default=45s"`
	Blacklist      []string      `env:"BLACKLIST,default=repro,bootleg,kaos,custom,premium,fake"`
}

func Load() (*Config, error) {
	// Load .env file (ignore error if not exists)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &cfg, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
go test ./config/... -v
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add config/ .env.example go.mod go.sum
git commit -m "feat: add config package with env vars support"
```

---

## Task 2: Database Package

**Files:**
- Create: `database/database.go`
- Create: `database/database_test.go`

- [ ] **Step 1: Write failing test for database**

```go
// database/database_test.go
package database

import (
	"log/slog"
	"os"
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

	if !db2.IsNew("11111") {
		t.Error("loaded db should have '11111'")
	}
	if db2.Count() != 2 {
		t.Errorf("loaded db Count() = %d, want 2", db2.Count())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./database/... -v
```

Expected: FAIL with "package database is not in GOROOT"

- [ ] **Step 3: Implement database package**

```go
// database/database.go
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

	// Atomic write: write to temp file, then rename
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./database/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add database/
git commit -m "feat: add database package with atomic writes"
```

---

## Task 3: Telegram Package

**Files:**
- Create: `telegram/telegram.go`
- Create: `telegram/telegram_test.go`

- [ ] **Step 1: Write failing test for telegram**

```go
// telegram/telegram_test.go
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

	// Should not panic
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./telegram/... -v
```

Expected: FAIL with "package telegram is not in GOROOT"

- [ ] **Step 3: Implement telegram package**

```go
// telegram/telegram.go
package telegram

import (
	"fmt"
	"log/slog"
	"strings"

	"carousell-bot/scraper"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	chatID   int64
	logger   *slog.Logger
	disabled bool
}

func New(token string, chatID int64, logger *slog.Logger) (*Bot, error) {
	if token == "" {
		logger.Warn("TELEGRAM_TOKEN not set, running in LOG ONLY mode")
		return &Bot{chatID: chatID, logger: logger, disabled: true}, nil
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("init telegram bot: %w", err)
	}

	logger.Info("authorized on telegram account", slog.String("username", api.Self.UserName))
	return &Bot{api: api, chatID: chatID, logger: logger}, nil
}

func (b *Bot) Notify(l scraper.Listing) error {
	if b.disabled {
		b.logger.Info("LISTING FOUND (log only)",
			slog.String("id", l.ID),
			slog.String("brand", l.Brand),
			slog.String("title", l.Title),
			slog.String("price", l.Price),
		)
		return nil
	}

	msgText := fmt.Sprintf(
		"<b>PERFECT FIND: %s</b>\n\n"+
			"<b>Title:</b> %s\n"+
			"<b>Price:</b> %s\n"+
			"<b>Seller:</b> %s\n"+
			"<b>Time:</b> %s\n\n"+
			"<a href=\"%s\">Open Listing</a>",
		strings.ToUpper(l.Brand), l.Title, l.Price, l.Seller, l.Time, l.URL,
	)

	msg := tgbotapi.NewMessage(b.chatID, msgText)
	msg.ParseMode = "HTML"

	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./telegram/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add telegram/
git commit -m "feat: add telegram package with HTML format"
```

---

## Task 4: Scraper Package

**Files:**
- Create: `scraper/scraper.go`
- Create: `scraper/scraper_test.go`

- [ ] **Step 1: Write failing test for scraper**

```go
// scraper/scraper_test.go
package scraper

import (
	"log/slog"
	"testing"
)

func TestScraper_IsBlacklisted(t *testing.T) {
	logger := slog.Default()
	s := New(nil, logger, []string{"repro", "bootleg", "fake"}, 0)

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

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./scraper/... -v
```

Expected: FAIL with "package scraper is not in GOROOT"

- [ ] **Step 3: Implement scraper package**

```go
// scraper/scraper.go
package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type Listing struct {
	ID     string
	Title  string
	Price  string
	Seller string
	Time   string
	URL    string
	Brand  string
}

type Scraper struct {
	allocCtx    context.Context
	logger      *slog.Logger
	blacklist   []string
	pageTimeout time.Duration
}

func New(allocCtx context.Context, logger *slog.Logger, blacklist []string, pageTimeout time.Duration) *Scraper {
	return &Scraper{
		allocCtx:    allocCtx,
		logger:      logger,
		blacklist:   blacklist,
		pageTimeout: pageTimeout,
	}
}

func (s *Scraper) Fetch(ctx context.Context, brand string) ([]Listing, error) {
	var html string
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", brand)

	tctx, cancel := context.WithTimeout(ctx, s.pageTimeout)
	defer cancel()

	// Create new tab
	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	err := chromedp.Run(tabCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 3; i++ {
				if err := chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil).Do(ctx); err != nil {
					return err
				}
				chromedp.Sleep(500 * time.Millisecond).Do(ctx)
			}
			return nil
		}),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
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

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./scraper/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add scraper/
git commit -m "feat: add scraper package with context support"
```

---

## Task 5: Main Orchestrator

**Files:**
- Create: `main.go` (rewrite)

- [ ] **Step 1: Write main.go with graceful shutdown**

```go
// main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"carousell-bot/config"
	"carousell-bot/database"
	"carousell-bot/scraper"
	"carousell-bot/telegram"

	"github.com/cenkalti/backoff/v4"
	"github.com/chromedp/chromedp"
)

func main() {
	// Setup structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseFile, logger)
	if err != nil {
		logger.Error("failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize telegram
	tg, err := telegram.New(cfg.TelegramToken, cfg.TelegramChatID, logger)
	if err != nil {
		logger.Error("failed to initialize telegram", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Load brands
	brands, err := loadBrands(cfg.BrandFile)
	if err != nil {
		logger.Error("failed to load brands", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Setup Chrome allocator
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Initialize scraper
	sc := scraper.New(allocCtx, logger, cfg.Blacklist, cfg.PageTimeout)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting carousell bot",
		slog.Int("workers", cfg.Workers),
		slog.Int("brands", len(brands)),
	)

	// Orchestrator loop
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down gracefully...")
			if err := db.Save(); err != nil {
				logger.Error("failed to save database on shutdown", slog.String("error", err.Error()))
			}
			return
		default:
			runCycle(ctx, sc, db, tg, brands, cfg, logger)
			time.Sleep(cfg.CycleDelay)
		}
	}
}

func runCycle(ctx context.Context, sc *scraper.Scraper, db *database.Database, tg *telegram.Bot, brands []string, cfg *config.Config, logger *slog.Logger) {
	brandChan := make(chan string, len(brands))
	resultChan := make(chan int, len(brands))

	// Start workers
	var wg sync.WaitGroup
	for i := 1; i <= cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(ctx, workerID, sc, db, tg, brandChan, resultChan, cfg, logger)
		}(i)
	}

	// Push jobs
	for _, b := range brands {
		brandChan <- b
	}
	close(brandChan)

	// Wait for cycle
	wg.Wait()
	close(resultChan)

	totalNew := 0
	for n := range resultChan {
		totalNew += n
	}

	logger.Info("cycle complete", slog.Int("new_items", totalNew))
}

func worker(ctx context.Context, id int, sc *scraper.Scraper, db *database.Database, tg *telegram.Bot, brands <-chan string, results chan<- int, cfg *config.Config, logger *slog.Logger) {
	for brand := range brands {
		logger.Info("scanning brand", slog.Int("worker", id), slog.String("brand", brand))

		var listings []scraper.Listing
		var err error

		// Exponential backoff retry
		operation := func() error {
			listings, err = sc.Fetch(ctx, brand)
			if err != nil {
				return err
			}
			return nil
		}

		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = 30 * time.Second

		if err := backoff.Retry(operation, backoff.WithMaxRetries(b, 2)); err != nil {
			logger.Error("failed to fetch brand",
				slog.Int("worker", id),
				slog.String("brand", brand),
				slog.String("error", err.Error()),
			)
			results <- 0
			continue
		}

		newFound := 0
		for _, l := range listings {
			if db.IsNew(l.ID) {
				logger.Info("found new listing",
					slog.Int("worker", id),
					slog.String("title", l.Title),
					slog.String("price", l.Price),
				)
				if err := tg.Notify(l); err != nil {
					logger.Error("failed to send notification",
						slog.String("error", err.Error()),
					)
				}
				db.MarkSeen(l.ID)
				newFound++
			}
		}

		if newFound > 0 {
			if err := db.Save(); err != nil {
				logger.Error("failed to save database", slog.String("error", err.Error()))
			}
		}

		logger.Info("brand scan complete",
			slog.Int("worker", id),
			slog.String("brand", brand),
			slog.Int("new", newFound),
		)

		results <- newFound
		time.Sleep(cfg.MinDelay)
	}
}

func loadBrands(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var brands []string
	for _, line := range splitLines(string(data)) {
		for _, brand := range splitComma(line) {
			trimmed := trimSpace(brand)
			if trimmed != "" {
				brands = append(brands, trimmed)
			}
		}
	}

	return brands, nil
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range splitBy(s, "\n") {
		lines = append(lines, line)
	}
	return lines
}

func splitComma(s string) []string {
	var parts []string
	for _, part := range splitBy(s, ",") {
		parts = append(parts, part)
	}
	return parts
}

func splitBy(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
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

Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: rewrite main.go with graceful shutdown and structured logging"
```

---

## Task 6: Final Cleanup

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Update .gitignore**

```gitignore
# Binary
bot

# Environment
.env

# Database
seen_db.json

# Go
*.exe
*.test
*.out
```

- [ ] **Step 2: Run final verification**

```bash
go vet ./...
go test ./...
go build -o bot .
```

Expected: All pass

- [ ] **Step 3: Final commit**

```bash
git add .gitignore
git commit -m "chore: update gitignore for new structure"
```

---

## Verification Checklist

- [ ] All tests pass: `go test ./... -v`
- [ ] No vet warnings: `go vet ./...`
- [ ] Binary builds: `go build -o bot .`
- [ ] `.env.example` exists with all vars
- [ ] `seen_db.json` format unchanged
- [ ] `brand_list.txt` format unchanged
- [ ] Graceful shutdown works (Ctrl+C)
- [ ] Structured JSON logging works
