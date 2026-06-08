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
