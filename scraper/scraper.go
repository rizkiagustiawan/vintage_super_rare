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
	Brand  string
	Seller string
	Time   string
	URL    string
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

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	var cancel context.CancelFunc
	if s.pageTimeout > 0 {
		tabCtx, cancel = context.WithTimeout(tabCtx, s.pageTimeout)
		defer cancel()
	}

	go func() {
		select {
		case <-ctx.Done():
			tabCancel()
		case <-tabCtx.Done():
		}
	}()

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
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
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
				Seller: "Unknown", // TODO: extract from HTML when markup is understood
				Time:   "Recent",  // TODO: extract from HTML when markup is understood
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
