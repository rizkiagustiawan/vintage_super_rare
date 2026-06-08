package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
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
			3,                    // Max retries
			500*time.Millisecond, // Min backoff
			5*time.Second,        // Max backoff
			[]int{429, 503},      // Status codes to retry
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
	encodedBrand := url.PathEscape(brand)
	url := fmt.Sprintf("https://id.carousell.com/search/%s/?sort_by=3", encodedBrand)

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

	linksFound := 0
	doc.Find("a[href*='/p/']").Each(func(i int, sel *goquery.Selection) {
		link, exists := sel.Attr("href")
		if !exists {
			return
		}
		linksFound++

		// Strip query params first
		linkPath := strings.Split(link, "?")[0]
		
		parts := strings.Split(strings.Trim(linkPath, "/"), "-")
		if len(parts) == 0 {
			return
		}

		id := parts[len(parts)-1]

		if len(id) < 5 {
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
		slog.Int("links_found", linksFound),
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
