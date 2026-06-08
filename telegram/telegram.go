package telegram

import (
	"fmt"
	"html"
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
		strings.ToUpper(html.EscapeString(l.Brand)),
		html.EscapeString(l.Title),
		html.EscapeString(l.Price),
		html.EscapeString(l.Seller),
		html.EscapeString(l.Time),
		html.EscapeString(l.URL),
	)

	msg := tgbotapi.NewMessage(b.chatID, msgText)
	msg.ParseMode = "HTML"

	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}
