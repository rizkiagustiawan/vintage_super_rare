# Carousell Bot Modernization Design

**Date:** 2026-06-08  
**Status:** Approved  
**Approach:** Clean Rewrite (Approach A)

## Overview

Modernize the Carousell "Perfect Hunter" bot from a single-file Go script to a well-structured, modern Go application with proper error handling, structured logging, graceful shutdown, and comprehensive testing.

## Current State

- Single file: `main.go` (377 lines)
- Logging: `log.Printf`
- Config: Hardcoded values
- Error handling: Inconsistent
- Shutdown: None
- Retry: Simple loop
- Tests: None

## Target State

- Multi-file: 4 packages × 2 files each
- Logging: `log/slog` JSON structured
- Config: Env vars + `.env` file
- Error handling: Wrapped errors with `%w`
- Shutdown: Graceful via `signal.NotifyContext`
- Retry: Exponential backoff
- Tests: Comprehensive unit tests

## Architecture

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
├── brand_list.txt
├── .env.example         # Template untuk env vars
├── .gitignore
├── go.mod
└── go.sum
```

## Data Flow

```
main.go
  ├─→ config.Load() → Config struct
  ├─→ database.New() → seenListings map
  ├─→ telegram.New() → BotAPI client
  ├─→ scraper.New() → Carousell client
  └─→ orchestrator loop:
       ├─→ scraper.Fetch(brand) → []Listing
       ├─→ database.IsNew(id) → bool
       ├─→ telegram.Notify(listing)
       └─→ database.Save()
```

## Package Designs

### Config Package

```go
// config/config.go
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

func Load() (*Config, error)
```

**Dependencies:**
- `github.com/joho/godotenv` - .env file loading
- `github.com/sethvargo/go-envconfig` - env var parsing with defaults

### Database Package

```go
// database/database.go
type Database struct {
    mu       sync.RWMutex
    seen     map[string]bool
    filePath string
    logger   *slog.Logger
}

func New(filePath string, logger *slog.Logger) (*Database, error)
func (db *Database) IsNew(id string) bool
func (db *Database) MarkSeen(id string)
func (db *Database) Save() error
func (db *Database) Count() int
```

**Key Features:**
- Atomic file writes (write temp → rename)
- Thread-safe dengan `sync.RWMutex`
- Structured logging

### Scraper Package

```go
// scraper/scraper.go
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

func New(allocCtx context.Context, logger *slog.Logger, blacklist []string, pageTimeout time.Duration) *Scraper
func (s *Scraper) Fetch(ctx context.Context, brand string) ([]Listing, error)
```

**Retry Strategy:**
- Exponential backoff with `github.com/cenkalti/backoff/v4`
- Max 2 retries per brand
- Connection errors trigger tab rebuild

### Telegram Package

```go
// telegram/telegram.go
type Bot struct {
    api      *tgbotapi.BotAPI
    chatID   int64
    logger   *slog.Logger
    disabled bool
}

func New(token string, chatID int64, logger *slog.Logger) (*Bot, error)
func (b *Bot) Notify(l scraper.Listing) error
```

**Key Features:**
- HTML format (lebih reliable dari Markdown)
- Graceful degradation (LOG ONLY mode tanpa token)
- Structured logging

### Main Orchestrator

```go
// main.go
func main() {
    // 1. Setup structured logger (slog JSON)
    // 2. Load config
    // 3. Initialize components (db, telegram, scraper)
    // 4. Setup Chrome allocator
    // 5. Graceful shutdown with signal.NotifyContext
    // 6. Orchestrator loop with select
}

func runCycle(ctx context.Context, ...) {
    // Worker pool, distribute brands, collect results
}
```

**Graceful Shutdown Flow:**
```
SIGINT/SIGTERM
  ↓
signal.NotifyContext cancels ctx
  ↓
select case <-ctx.Done() triggers
  ↓
db.Save() // persist state
  ↓
logger.Info("shutting down gracefully...")
  ↓
defer cancel() // cleanup Chrome
  ↓
exit
```

## New Dependencies

```go
require (
    github.com/joho/godotenv v1.5.1
    github.com/sethvargo/go-envconfig v1.1.0
    github.com/cenkalti/backoff/v4 v4.3.0
    // existing:
    github.com/PuerkitoBio/goquery v1.12.0
    github.com/chromedp/chromedp v0.15.1
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
)
```

## Changes Summary

| Area | Before | After |
|------|--------|-------|
| Logging | `log.Printf` | `log/slog` JSON structured |
| Config | Hardcoded | Env vars + .env file |
| Structure | Single file (377 lines) | 4 packages (8 files) |
| Error handling | Ignore/log | Wrapped errors with `%w` |
| Shutdown | None | Graceful via `signal.NotifyContext` |
| Retry | Simple loop | Exponential backoff |
| Telegram | Markdown | HTML format |
| Database | Unsafe writes | Atomic writes (temp → rename) |
| Tests | None | Comprehensive unit tests |

## Backward Compatibility

- `.env` file optional (env vars still work)
- `seen_db.json` format unchanged
- `brand_list.txt` format unchanged
- Telegram notifications same content, different format (HTML vs Markdown)

## Testing Strategy

- **config:** Test Load() with various env combinations
- **database:** Test IsNew/MarkSeen/Save with temp files
- **scraper:** Test isBlacklisted, mock HTML parsing
- **telegram:** Test Notify in disabled mode
