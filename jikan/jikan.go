// Package jikan is the library behind the jikan command line:
// the HTTP client, request shaping, and the typed data models for the Jikan
// v4 API (api.jikan.moe), the unofficial open-source MyAnimeList proxy.
//
// No API key is required. The client paces itself to 2 req/s to stay
// under Jikan's 3 req/s burst limit and 60 req/min sustained limit.
// Transient 429/5xx failures are retried with exponential back-off.
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
// Rate is 500 ms (~2 req/s) to stay well under Jikan's 3 req/s burst limit.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.jikan.moe/v4",
		UserAgent: "jikan-cli/0.1.0 (github.com/tamnd/jikan-cli)",
		Rate:      500 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Jikan v4 API over HTTP.
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

// --- response envelopes ---

type listResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination pagination `json:"pagination"`
}

type pagination struct {
	LastVisiblePage int  `json:"last_visible_page"`
	HasNextPage     bool `json:"has_next_page"`
}

type singleResponse[T any] struct {
	Data T `json:"data"`
}

// --- raw wire types ---

// rawAnime mirrors the JSON shape from the Jikan v4 API.
// Episodes is *int because the API sends null for ongoing series.
type rawAnime struct {
	MalID         int        `json:"mal_id"`
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	TitleEnglish  string     `json:"title_english"`
	TitleJapanese string     `json:"title_japanese"`
	Type          string     `json:"type"`
	Source        string     `json:"source"`
	Episodes      *int       `json:"episodes"`
	Status        string     `json:"status"`
	Airing        bool       `json:"airing"`
	Score         float64    `json:"score"`
	ScoredBy      int        `json:"scored_by"`
	Rank          int        `json:"rank"`
	Popularity    int        `json:"popularity"`
	Members       int        `json:"members"`
	Synopsis      string     `json:"synopsis"`
	Background    string     `json:"background"`
	Premiered     string     `json:"premiered"`
	Season        string     `json:"season"`
	Year          int        `json:"year"`
	Broadcast     rawBcast   `json:"broadcast"`
	Genres        []rawNamed `json:"genres"`
	Studios       []rawNamed `json:"studios"`
}

type rawBcast struct {
	Day      string `json:"day"`
	Time     string `json:"time"`
	Timezone string `json:"timezone"`
}

type rawNamed struct {
	MalID int    `json:"mal_id"`
	Name  string `json:"name"`
}

type rawManga struct {
	MalID        int        `json:"mal_id"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	TitleEnglish string     `json:"title_english"`
	Volumes      int        `json:"volumes"`
	Chapters     int        `json:"chapters"`
	Status       string     `json:"status"`
	Score        float64    `json:"score"`
	Rank         int        `json:"rank"`
	Genres       []rawNamed `json:"genres"`
	Synopsis     string     `json:"synopsis"`
}

type rawCharacter struct {
	MalID     int      `json:"mal_id"`
	URL       string   `json:"url"`
	Name      string   `json:"name"`
	NameKanji string   `json:"name_kanji"`
	Nicknames []string `json:"nicknames"`
	Favorites int      `json:"favorites"`
	About     string   `json:"about"`
}

type rawPerson struct {
	MalID      int    `json:"mal_id"`
	URL        string `json:"url"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Birthday   string `json:"birthday"`
	Favorites  int    `json:"favorites"`
	About      string `json:"about"`
}

// --- public output types ---

// Anime is one anime entry returned by the Jikan v4 API.
// Episodes is 0 when the API reports null (ongoing with unknown count).
type Anime struct {
	MalID         int       `json:"mal_id"`
	URL           string    `json:"url"`
	Title         string    `json:"title"`
	TitleEnglish  string    `json:"title_english"`
	TitleJapanese string    `json:"title_japanese"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	Episodes      int       `json:"episodes"`
	Status        string    `json:"status"`
	Airing        bool      `json:"airing"`
	Score         float64   `json:"score"`
	ScoredBy      int       `json:"scored_by"`
	Rank          int       `json:"rank"`
	Popularity    int       `json:"popularity"`
	Members       int       `json:"members"`
	Synopsis      string    `json:"synopsis"`
	Background    string    `json:"background"`
	Premiered     string    `json:"premiered"`
	Season        string    `json:"season"`
	Year          int       `json:"year"`
	Broadcast     Broadcast `json:"broadcast"`
	Genres        []Genre   `json:"genres"`
	Studios       []Studio  `json:"studios"`
}

// Broadcast holds the weekly broadcast schedule for a currently-airing anime.
type Broadcast struct {
	Day      string `json:"day"`
	Time     string `json:"time"`
	Timezone string `json:"timezone"`
}

// Genre is a MAL genre tag.
type Genre struct {
	MalID int    `json:"mal_id"`
	Name  string `json:"name"`
}

// Studio is an anime production studio.
type Studio struct {
	MalID int    `json:"mal_id"`
	Name  string `json:"name"`
}

// Manga is one manga entry returned by the Jikan v4 API.
type Manga struct {
	MalID        int     `json:"mal_id"`
	URL          string  `json:"url"`
	Title        string  `json:"title"`
	TitleEnglish string  `json:"title_english"`
	Volumes      int     `json:"volumes"`
	Chapters     int     `json:"chapters"`
	Status       string  `json:"status"`
	Score        float64 `json:"score"`
	Rank         int     `json:"rank"`
	Genres       []Genre `json:"genres"`
	Synopsis     string  `json:"synopsis"`
}

// Character is a MAL character entry.
type Character struct {
	MalID     int      `json:"mal_id"`
	URL       string   `json:"url"`
	Name      string   `json:"name"`
	NameKanji string   `json:"name_kanji"`
	Nicknames []string `json:"nicknames"`
	Favorites int      `json:"favorites"`
	About     string   `json:"about"`
}

// Person is a MAL people entry (voice actor, director, etc.).
type Person struct {
	MalID      int    `json:"mal_id"`
	URL        string `json:"url"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Birthday   string `json:"birthday"`
	Favorites  int    `json:"favorites"`
	About      string `json:"about"`
}

// --- client methods ---

// SearchAnime searches anime by title keyword.
// Limit is clamped to 25 (Jikan's per-page maximum).
func (c *Client) SearchAnime(ctx context.Context, query string, limit int) ([]Anime, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/anime?q=%s&limit=%d", c.cfg.BaseURL, neturl.QueryEscape(query), n)
	return c.fetchAnimeList(ctx, u, limit)
}

// GetAnime fetches a single anime by its MAL ID using the /full endpoint.
func (c *Client) GetAnime(ctx context.Context, id int) (*Anime, error) {
	u := fmt.Sprintf("%s/anime/%d/full", c.cfg.BaseURL, id)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp singleResponse[rawAnime]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode anime: %w", err)
	}
	a := toAnime(resp.Data)
	return &a, nil
}

// TopAnime returns the top-ranked anime of the given type.
// typ may be: tv, movie, ova, special, ona, music. Empty string returns all types.
// Limit is clamped to 25.
func (c *Client) TopAnime(ctx context.Context, typ string, limit int) ([]Anime, error) {
	n := clamp(limit)
	if typ == "" {
		typ = "tv"
	}
	u := fmt.Sprintf("%s/top/anime?type=%s&limit=%d", c.cfg.BaseURL, typ, n)
	return c.fetchAnimeList(ctx, u, limit)
}

// SearchManga searches manga by title keyword.
// Limit is clamped to 25.
func (c *Client) SearchManga(ctx context.Context, query string, limit int) ([]Manga, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/manga?q=%s&limit=%d", c.cfg.BaseURL, neturl.QueryEscape(query), n)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp listResponse[rawManga]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode manga: %w", err)
	}
	out := make([]Manga, 0, len(resp.Data))
	for _, m := range resp.Data {
		out = append(out, toManga(m))
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// SearchCharacters searches MAL characters by name.
// Limit is clamped to 25.
func (c *Client) SearchCharacters(ctx context.Context, query string, limit int) ([]Character, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/characters?q=%s&limit=%d", c.cfg.BaseURL, neturl.QueryEscape(query), n)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp listResponse[rawCharacter]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode characters: %w", err)
	}
	out := make([]Character, 0, len(resp.Data))
	for _, ch := range resp.Data {
		out = append(out, toCharacter(ch))
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// SearchPeople searches MAL people (voice actors, staff) by name.
// Limit is clamped to 25.
func (c *Client) SearchPeople(ctx context.Context, query string, limit int) ([]Person, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/people?q=%s&limit=%d", c.cfg.BaseURL, neturl.QueryEscape(query), n)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp listResponse[rawPerson]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode people: %w", err)
	}
	out := make([]Person, 0, len(resp.Data))
	for _, p := range resp.Data {
		out = append(out, toPerson(p))
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// Season returns anime from a specific year/season (winter, spring, summer, fall).
// Limit is clamped to 25.
func (c *Client) Season(ctx context.Context, year int, season string, limit int) ([]Anime, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/seasons/%d/%s?limit=%d", c.cfg.BaseURL, year, season, n)
	return c.fetchAnimeList(ctx, u, limit)
}

// Schedule returns today's airing anime schedule.
// Limit is clamped to 25.
func (c *Client) Schedule(ctx context.Context, limit int) ([]Anime, error) {
	n := clamp(limit)
	u := fmt.Sprintf("%s/schedules?limit=%d", c.cfg.BaseURL, n)
	return c.fetchAnimeList(ctx, u, limit)
}

// --- internal helpers ---

func (c *Client) fetchAnimeList(ctx context.Context, u string, limit int) ([]Anime, error) {
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp listResponse[rawAnime]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode anime list: %w", err)
	}
	out := make([]Anime, 0, len(resp.Data))
	for _, a := range resp.Data {
		out = append(out, toAnime(a))
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
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

// clamp limits n to [1, 25]; 0 or negative becomes 25.
func clamp(n int) int {
	if n <= 0 || n > 25 {
		return 25
	}
	return n
}

// --- converters ---

func toAnime(a rawAnime) Anime {
	eps := 0
	if a.Episodes != nil {
		eps = *a.Episodes
	}
	genres := make([]Genre, len(a.Genres))
	for i, g := range a.Genres {
		genres[i] = Genre{MalID: g.MalID, Name: g.Name}
	}
	studios := make([]Studio, len(a.Studios))
	for i, s := range a.Studios {
		studios[i] = Studio{MalID: s.MalID, Name: s.Name}
	}
	return Anime{
		MalID:         a.MalID,
		URL:           a.URL,
		Title:         a.Title,
		TitleEnglish:  a.TitleEnglish,
		TitleJapanese: a.TitleJapanese,
		Type:          a.Type,
		Source:        a.Source,
		Episodes:      eps,
		Status:        a.Status,
		Airing:        a.Airing,
		Score:         a.Score,
		ScoredBy:      a.ScoredBy,
		Rank:          a.Rank,
		Popularity:    a.Popularity,
		Members:       a.Members,
		Synopsis:      a.Synopsis,
		Background:    a.Background,
		Premiered:     a.Premiered,
		Season:        a.Season,
		Year:          a.Year,
		Broadcast:     Broadcast{Day: a.Broadcast.Day, Time: a.Broadcast.Time, Timezone: a.Broadcast.Timezone},
		Genres:        genres,
		Studios:       studios,
	}
}

func toManga(m rawManga) Manga {
	genres := make([]Genre, len(m.Genres))
	for i, g := range m.Genres {
		genres[i] = Genre{MalID: g.MalID, Name: g.Name}
	}
	return Manga{
		MalID:        m.MalID,
		URL:          m.URL,
		Title:        m.Title,
		TitleEnglish: m.TitleEnglish,
		Volumes:      m.Volumes,
		Chapters:     m.Chapters,
		Status:       m.Status,
		Score:        m.Score,
		Rank:         m.Rank,
		Genres:       genres,
		Synopsis:     m.Synopsis,
	}
}

func toCharacter(ch rawCharacter) Character {
	nicks := ch.Nicknames
	if nicks == nil {
		nicks = []string{}
	}
	return Character{
		MalID:     ch.MalID,
		URL:       ch.URL,
		Name:      ch.Name,
		NameKanji: ch.NameKanji,
		Nicknames: nicks,
		Favorites: ch.Favorites,
		About:     ch.About,
	}
}

func toPerson(p rawPerson) Person {
	return Person{
		MalID:      p.MalID,
		URL:        p.URL,
		Name:       p.Name,
		GivenName:  p.GivenName,
		FamilyName: p.FamilyName,
		Birthday:   p.Birthday,
		Favorites:  p.Favorites,
		About:      p.About,
	}
}
