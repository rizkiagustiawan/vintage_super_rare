package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// --- CONFIGURATION & TYPES ---

type Listing struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Price  string `json:"price"`
	Seller string `json:"seller"`
	Time   string `json:"time"`
	URL    string `json:"url"`
	Brand  string `json:"brand"`
}

type Config struct {
	TelegramToken  string
	TelegramChatID int64
	BrandFile      string
	DatabaseFile   string
	Blacklist      []string
	MinDelay       time.Duration
	CycleDelay     time.Duration
}

var (
	seenListings = make(map[string]bool)
	tgBot        *tgbotapi.BotAPI
	config       Config
)

// --- DATABASE LOGIC (PERSISTENCE) ---

func loadDatabase() {
	if _, err := os.Stat(config.DatabaseFile); os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile(config.DatabaseFile)
	if err != nil {
		log.Printf("Warning: Could not read database: %v", err)
		return
	}
	var saved []string
	json.Unmarshal(data, &saved)
	for _, id := range saved {
		seenListings[id] = true
	}
	log.Printf("Loaded %d items from database.", len(seenListings))
}

func saveDatabase() {
	var ids []string
	for id := range seenListings {
		ids = append(ids, id)
	}
	data, _ := json.Marshal(ids)
	os.WriteFile(config.DatabaseFile, data, 0644)
}

// --- TELEGRAM LOGIC ---

func initTelegram() {
	config.TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	idStr := os.Getenv("TELEGRAM_CHAT_ID")

	if config.TelegramToken == "" || idStr == "" {
		log.Fatalf("Critical: TELEGRAM_TOKEN or TELEGRAM_CHAT_ID not set")
	}

	fmt.Sscanf(idStr, "%d", &config.TelegramChatID)

	var err error
	tgBot, err = tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to init Telegram: %v", err)
	}
	log.Printf("Authorized on account %s", tgBot.Self.UserName)
}

func notify(l Listing) {
	msgText := fmt.Sprintf(
		"💎 *PERFECT FIND: %s*\n\n"+
			"🏷️ *Title:* %s\n"+
			"💰 *Price:* %s\n"+
			"👤 *Seller:* %s\n"+
			"🕒 *Time:* %s\n\n"+
			"🔗 [Open Listing](%s)",
		strings.ToUpper(l.Brand), l.Title, l.Price, l.Seller, l.Time, l.URL,
	)

	msg := tgbotapi.NewMessage(config.TelegramChatID, msgText)
	msg.ParseMode = "Markdown"
	_, err := tgBot.Send(msg)
	if err != nil {
		log.Printf("Telegram Error: %v", err)
	}
}

// --- SCRAPER LOGIC ---

func isBlacklisted(title string) bool {
	t := strings.ToLower(title)
	for _, word := range config.Blacklist {
		if strings.Contains(t, word) {
			return true
		}
	}
	return false
}

func fetchListings(query string) ([]Listing, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)
	
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var html string
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", query)

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(4*time.Second),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var listings []Listing
	doc.Find("div.D_aih").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("p.D_ait").Text())
		
		// Perfect Filter: Blacklist check
		if isBlacklisted(title) {
			return
		}

		price := strings.TrimSpace(s.Find("p.D_aiu").Text())
		seller := strings.TrimSpace(s.Find("p.D_aio").Text())
		timeStr := strings.TrimSpace(s.Find("p.D_aiy").Text())
		link, _ := s.Find("a.D_aik").Attr("href")

		if title != "" && price != "" && link != "" {
			parts := strings.Split(strings.Trim(link, "/"), "-")
			id := parts[len(parts)-1]

			listings = append(listings, Listing{
				ID:     id,
				Title:  title,
				Price:  price,
				Seller: seller,
				Time:   timeStr,
				URL:    "https://id.carousell.com" + link,
				Brand:  query,
			})
		}
	})

	return listings, nil
}

// --- MAIN ENGINE ---

func main() {
	config = Config{
		BrandFile:    "brand_list.txt",
		DatabaseFile: "seen_db.json",
		Blacklist:    []string{"repro", "bootleg", "kaos gambar", "custom", "premium high", "fake"},
		MinDelay:     7 * time.Second,  // Delay between brands
		CycleDelay:   10 * time.Minute, // Delay between full cycles
	}

	initTelegram()
	loadDatabase()

	content, err := os.ReadFile(config.BrandFile)
	if err != nil {
		log.Fatalf("Failed to read brand list: %v", err)
	}
	brands := strings.Split(string(content), ",")

	log.Printf("Starting Perfect Hunter. Brands: %d", len(brands))

	for {
		newFoundInCycle := 0
		for _, b := range brands {
			brand := strings.TrimSpace(b)
			if brand == "" {
				continue
			}

			log.Printf("🔍 Scanning: %s", brand)
			listings, err := fetchListings(brand)
			if err != nil {
				log.Printf("❌ Error [%s]: %v", brand, err)
				continue
			}

			for _, l := range listings {
				if !seenListings[l.ID] {
					log.Printf("🎯 FOUND: %s", l.Title)
					
					// Only notify if we already have a database (don't spam on first ever run)
					if len(seenListings) > 0 {
						notify(l)
					}
					
					seenListings[l.ID] = true
					newFoundInCycle++
				}
			}
			
			// Save after each brand to be safe
			if newFoundInCycle > 0 {
				saveDatabase()
			}
			
			time.Sleep(config.MinDelay)
		}

		log.Printf("✅ Cycle complete. Found %d new items. Sleeping %v...", newFoundInCycle, config.CycleDelay)
		time.Sleep(config.CycleDelay)
	}
}
