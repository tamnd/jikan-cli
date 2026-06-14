package jikan_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/jikan-cli/jikan"
)

const fakeTopJSON = `{
  "data": [
    {
      "mal_id": 52991,
      "rank": 1,
      "title": "Sousou no Frieren",
      "title_english": "Frieren: Beyond Journey's End",
      "score": 9.26,
      "episodes": 28,
      "type": "TV",
      "season": "fall",
      "year": 2023,
      "status": "Finished Airing",
      "genres": [
        {"mal_id": 1, "type": "anime", "name": "Action"},
        {"mal_id": 8, "type": "anime", "name": "Drama"},
        {"mal_id": 10, "type": "anime", "name": "Fantasy"}
      ],
      "url": "https://myanimelist.net/anime/52991/Sousou_no_Frieren"
    },
    {
      "mal_id": 9253,
      "rank": 2,
      "title": "Steins;Gate",
      "title_english": "Steins;Gate",
      "score": 9.07,
      "episodes": 24,
      "type": "TV",
      "season": "spring",
      "year": 2011,
      "status": "Finished Airing",
      "genres": [
        {"mal_id": 4, "type": "anime", "name": "Comedy"},
        {"mal_id": 40, "type": "anime", "name": "Psychological"},
        {"mal_id": 24, "type": "anime", "name": "Sci-Fi"}
      ],
      "url": "https://myanimelist.net/anime/9253/Steins_Gate"
    }
  ],
  "pagination": {
    "last_visible_page": 428,
    "has_next_page": true,
    "items": {"count": 2, "total": 10000, "per_page": 25}
  }
}`

const fakeSeasonJSON = `{
  "data": [
    {
      "mal_id": 57299,
      "rank": 1,
      "title": "Tongari Boushi no Atelier",
      "title_english": "Witch Hat Atelier",
      "score": 8.7,
      "episodes": null,
      "type": "TV",
      "season": "spring",
      "year": 2026,
      "status": "Currently Airing",
      "genres": [
        {"mal_id": 2, "type": "anime", "name": "Adventure"},
        {"mal_id": 24, "type": "anime", "name": "Sci-Fi"}
      ],
      "url": "https://myanimelist.net/anime/57299/Tongari_Boushi_no_Atelier"
    }
  ],
  "pagination": {"items": {"total": 194}}
}`

const fakeSearchJSON = `{
  "data": [
    {
      "mal_id": 16498,
      "rank": 1,
      "title": "Shingeki no Kyojin",
      "title_english": "Attack on Titan",
      "score": 8.54,
      "episodes": 25,
      "type": "TV",
      "season": "spring",
      "year": 2013,
      "status": "Finished Airing",
      "genres": [
        {"mal_id": 1, "type": "anime", "name": "Action"},
        {"mal_id": 8, "type": "anime", "name": "Drama"}
      ],
      "url": "https://myanimelist.net/anime/16498/Shingeki_no_Kyojin"
    }
  ],
  "pagination": {"items": {"count": 1, "total": 23, "per_page": 25}}
}`

func newTestClient(ts *httptest.Server) *jikan.Client {
	cfg := jikan.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return jikan.NewClient(cfg)
}

func TestTopSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeTopJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Top(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestTopParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeTopJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Top(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	got := items[0]
	if got.MalID != 52991 {
		t.Errorf("MalID = %d, want 52991", got.MalID)
	}
	if got.Title != "Sousou no Frieren" {
		t.Errorf("Title = %q, want Sousou no Frieren", got.Title)
	}
	if got.Score != 9.26 {
		t.Errorf("Score = %v, want 9.26", got.Score)
	}
	if got.Episodes != 28 {
		t.Errorf("Episodes = %d, want 28", got.Episodes)
	}

	wantGenres := map[string]bool{"Action": true, "Drama": true, "Fantasy": true}
	for _, g := range got.Genres {
		delete(wantGenres, g)
	}
	if len(wantGenres) > 0 {
		t.Errorf("missing genres: %v", wantGenres)
	}

	const wantPrefix = "https://myanimelist.net/"
	if len(got.URL) < len(wantPrefix) || got.URL[:len(wantPrefix)] != wantPrefix {
		t.Errorf("URL = %q, want prefix %q", got.URL, wantPrefix)
	}
}

func TestTopLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeTopJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Top(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestTopRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeTopJSON)
	}))
	defer ts.Close()

	cfg := jikan.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := jikan.NewClient(cfg)

	_, err := c.Top(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestSearchParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Search(context.Background(), "attack on titan", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.MalID != 16498 {
		t.Errorf("MalID = %d, want 16498", got.MalID)
	}
	if got.TitleEn != "Attack on Titan" {
		t.Errorf("TitleEn = %q, want Attack on Titan", got.TitleEn)
	}
}

func TestSeasonParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSeasonJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Season(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.MalID != 57299 {
		t.Errorf("MalID = %d, want 57299", got.MalID)
	}
	if got.Season != "spring" {
		t.Errorf("Season = %q, want spring", got.Season)
	}
	if got.Year != 2026 {
		t.Errorf("Year = %d, want 2026", got.Year)
	}
	if got.Episodes != 0 {
		t.Errorf("Episodes = %d, want 0 (null in JSON)", got.Episodes)
	}
	if got.Status != "Currently Airing" {
		t.Errorf("Status = %q, want Currently Airing", got.Status)
	}
}
