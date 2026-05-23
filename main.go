package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

var (
	seenListings = make(map[string]bool)
	tgBot        *tgbotapi.BotAPI
	chatID       int64 = 775545807
	tgToken      = "8896705019:AAGgqW4gSilbBcWlNwk6Q3b8KzeNGCSNzso"
)

func initTelegram() {
	var err error
	tgBot, err = tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Fatalf("Failed to init Telegram: %v", err)
	}
	log.Printf("Authorized on account %s", tgBot.Self.UserName)
}

func sendTelegram(l Listing) {
	msgText := fmt.Sprintf(
		"🚨 *SUPER RARE FIND: %s*\n\n"+
			"*Title:* %s\n"+
			"*Price:* %s\n"+
			"*Seller:* %s\n"+
			"*Time:* %s\n\n"+
			"[View Listing](%s)",
		l.Brand, l.Title, l.Price, l.Seller, l.Time, l.URL,
	)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	_, err := tgBot.Send(msg)
	if err != nil {
		log.Printf("Error sending TG: %v", err)
	}
}

func fetchListings(query string) ([]Listing, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var html string
	// query with sort_by=3 (recent)
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", query)

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
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

func readBrands(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read brand file: %v", err)
	}
	raw := string(content)
	// Split by comma and clean up
	parts := strings.Split(raw, ",")
	var brands []string
	for _, p := range parts {
		b := strings.TrimSpace(p)
		if b != "" {
			brands = append(brands, b)
		}
	}
	return brands
}

func main() {
	initTelegram()
	brands := readBrands("brand_list.txt")
	log.Printf("Monitoring %d brands...", len(brands))

	// Initial scan to populate seenListings
	log.Println("Starting initial scan (no notifications)...")
	for _, brand := range brands {
		log.Printf("Scanning %s...", brand)
		listings, _ := fetchListings(brand)
		for _, l := range listings {
			seenListings[l.ID] = true
		}
		time.Sleep(2 * time.Second) // Small delay between brand searches
	}
	log.Println("Initial scan complete. Waiting for new items...")

	for {
		for _, brand := range brands {
			log.Printf("Checking %s...", brand)
			listings, err := fetchListings(brand)
			if err != nil {
				log.Printf("Error checking %s: %v", brand, err)
				continue
			}

			for _, l := range listings {
				if !seenListings[l.ID] {
					log.Printf("!!! NEW ITEM FOUND: %s - %s", l.Brand, l.Title)
					sendTelegram(l)
					seenListings[l.ID] = true
				}
			}
			time.Sleep(5 * time.Second) // Anti-ban delay
		}
		log.Println("Cycle complete. Sleeping for 5 minutes...")
		time.Sleep(5 * time.Minute)
	}
}
