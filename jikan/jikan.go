// Package jikan is the library behind the jikan command line:
// the HTTP client, request shaping, and the typed data models for the Jikan
// API (api.jikan.moe), the unofficial open-source MyAnimeList proxy.
//
// No API key is required. The client paces itself to ~2.5 req/s to stay
// under Jikan's 3 req/s rate limit, and retries transient 429/5xx failures
// with exponential backoff.
package jikan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"
)

// Host is the Jikan API host.
const Host = "api.jikan.moe"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
// Rate is set to 400 ms (~2.5 req/s) to stay under Jikan's 3 req/s limit.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.jikan.moe/v4",
		UserAgent: "Mozilla/5.0 (compatible; jikan-cli/dev; +https://github.com/tamnd/jikan-cli)",
		Rate:      400 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Jikan API over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Top returns the top-ranked anime by MAL score.
// Pass limit <= 0 to use the API default of 25.
func (c *Client) Top(ctx context.Context, limit int) ([]Anime, error) {
	u := c.cfg.BaseURL + "/top/anime"
	if limit > 0 {
		u += fmt.Sprintf("?limit=%d", limit)
	}
	return c.fetchAnime(ctx, u, limit)
}

// Search returns anime matching the query string.
// Pass limit <= 0 to use the API default of 25.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Anime, error) {
	u := c.cfg.BaseURL + "/anime?q=" + neturl.QueryEscape(query)
	if limit > 0 {
		u += fmt.Sprintf("&limit=%d", limit)
	}
	return c.fetchAnime(ctx, u, limit)
}

// Season returns anime from the currently airing season.
// Pass limit <= 0 to use the API default of 25.
func (c *Client) Season(ctx context.Context, limit int) ([]Anime, error) {
	u := c.cfg.BaseURL + "/seasons/now"
	if limit > 0 {
		u += fmt.Sprintf("?limit=%d", limit)
	}
	return c.fetchAnime(ctx, u, limit)
}

func (c *Client) fetchAnime(ctx context.Context, u string, limit int) ([]Anime, error) {
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode anime: %w", err)
	}
	items := make([]Anime, 0, len(resp.Data))
	for _, a := range resp.Data {
		items = append(items, toAnime(a))
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
