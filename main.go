package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db, err := database.New(cfg.DatabaseFile, logger)
	if err != nil {
		logger.Error("failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	tg, err := telegram.New(cfg.TelegramToken, cfg.TelegramChatID, logger)
	if err != nil {
		logger.Error("failed to initialize telegram", slog.String("error", err.Error()))
		os.Exit(1)
	}

	brands, err := loadBrands(cfg.BrandFile)
	if err != nil {
		logger.Error("failed to load brands", slog.String("error", err.Error()))
		os.Exit(1)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	sc := scraper.New(allocCtx, logger, cfg.Blacklist, cfg.PageTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting carousell bot",
		slog.Int("workers", cfg.Workers),
		slog.Int("brands", len(brands)),
	)

	for {
		runCycle(ctx, sc, db, tg, brands, cfg, logger)
		select {
		case <-ctx.Done():
			logger.Info("shutting down gracefully...")
			if err := db.Save(); err != nil {
				logger.Error("failed to save database on shutdown", slog.String("error", err.Error()))
			}
			return
		case <-time.After(cfg.CycleDelay):
		}
	}
}

func runCycle(ctx context.Context, sc *scraper.Scraper, db *database.Database, tg *telegram.Bot, brands []string, cfg *config.Config, logger *slog.Logger) {
	brandChan := make(chan string, len(brands))
	resultChan := make(chan int, len(brands))

	var needsSave atomic.Bool
	var wg sync.WaitGroup
	for i := 1; i <= cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(ctx, workerID, sc, db, tg, brandChan, resultChan, &needsSave, cfg, logger)
		}(i)
	}

	for _, b := range brands {
		brandChan <- b
	}
	close(brandChan)

	wg.Wait()
	close(resultChan)

	if needsSave.Load() {
		if err := db.Save(); err != nil {
			logger.Error("failed to save database", slog.String("error", err.Error()))
		}
	}

	totalNew := 0
	for n := range resultChan {
		totalNew += n
	}

	logger.Info("cycle complete", slog.Int("new_items", totalNew))
}

func worker(ctx context.Context, id int, sc *scraper.Scraper, db *database.Database, tg *telegram.Bot, brands <-chan string, results chan<- int, needsSave *atomic.Bool, cfg *config.Config, logger *slog.Logger) {
	for brand := range brands {
		logger.Info("scanning brand", slog.Int("worker", id), slog.String("brand", brand))

		var listings []scraper.Listing
		var err error

		operation := func() error {
			listings, err = sc.Fetch(ctx, brand)
			return err
		}

		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = 30 * time.Second

		if err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(b, 2), ctx)); err != nil {
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
			needsSave.Store(true)
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
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				brands = append(brands, trimmed)
			}
		}
	}

	return brands, nil
}
