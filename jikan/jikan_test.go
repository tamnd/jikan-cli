package jikan_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/jikan-cli/jikan"
)

// --- fake JSON fixtures ---

const fakeAnimeJSON = `{"data":[
  {"mal_id":20,"url":"https://myanimelist.net/anime/20/Naruto",
   "title":"Naruto","title_english":"Naruto","title_japanese":"ナルト",
   "type":"TV","episodes":220,"status":"Finished Airing","airing":false,
   "score":8.02,"rank":713,"popularity":10,"members":2000000,
   "genres":[{"mal_id":1,"name":"Action"},{"mal_id":2,"name":"Adventure"}],
   "studios":[{"mal_id":1,"name":"Pierrot"}]},
  {"mal_id":1,"url":"https://myanimelist.net/anime/1/Cowboy_Bebop",
   "title":"Cowboy Bebop","title_english":"Cowboy Bebop","type":"TV",
   "episodes":26,"status":"Finished Airing","airing":false,
   "score":8.78,"rank":28,"popularity":40,"members":1000000,
   "genres":[{"mal_id":24,"name":"Sci-Fi"}],
   "studios":[{"mal_id":14,"name":"Sunrise"}]}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

const fakeAnimeFullJSON = `{"data":{
  "mal_id":16498,"url":"https://myanimelist.net/anime/16498/Shingeki_no_Kyojin",
  "title":"Shingeki no Kyojin","title_english":"Attack on Titan","type":"TV",
  "episodes":25,"status":"Finished Airing","airing":false,
  "score":8.54,"rank":75,"popularity":5,"members":3500000,
  "synopsis":"Centuries ago, mankind was slaughtered to near extinction.",
  "genres":[{"mal_id":1,"name":"Action"},{"mal_id":8,"name":"Drama"}],
  "studios":[{"mal_id":858,"name":"Wit Studio"}]}}`

const fakeMangaJSON = `{"data":[
  {"mal_id":11,"url":"https://myanimelist.net/manga/11/Naruto",
   "title":"Naruto","title_english":"Naruto",
   "volumes":72,"chapters":700,"status":"Finished","score":8.07,"rank":640,
   "genres":[{"mal_id":1,"name":"Action"},{"mal_id":27,"name":"Shounen"}],
   "synopsis":"Naruto Uzumaki is a young ninja who seeks recognition."}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

const fakeCharactersJSON = `{"data":[
  {"mal_id":17,"url":"https://myanimelist.net/character/17/Naruto_Uzumaki",
   "name":"Naruto Uzumaki","name_kanji":"うずまきナルト",
   "nicknames":["Knucklehead Ninja","Number One Unpredictable Ninja"],
   "favorites":47000,
   "about":"Naruto Uzumaki is a shinobi of Konohagakure."}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

const fakePeopleJSON = `{"data":[
  {"mal_id":185,"url":"https://myanimelist.net/people/185/Junko_Takeuchi",
   "name":"Takeuchi, Junko","given_name":"Junko","family_name":"Takeuchi",
   "birthday":"1972-01-01T00:00:00+00:00","favorites":25000,
   "about":"Junko Takeuchi is a Japanese voice actress."}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

const fakeSeasonJSON = `{"data":[
  {"mal_id":52991,"url":"https://myanimelist.net/anime/52991/Sousou_no_Frieren",
   "title":"Sousou no Frieren","title_english":"Frieren: Beyond Journey's End",
   "type":"TV","episodes":28,"status":"Finished Airing","airing":false,
   "score":9.26,"rank":1,"popularity":3,"members":1500000,
   "season":"fall","year":2023,
   "genres":[{"mal_id":10,"name":"Fantasy"}],
   "studios":[{"mal_id":11,"name":"Madhouse"}]}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

const fakeScheduleJSON = `{"data":[
  {"mal_id":57299,"url":"https://myanimelist.net/anime/57299",
   "title":"Tongari Boushi no Atelier","title_english":"Witch Hat Atelier",
   "type":"TV","episodes":null,"status":"Currently Airing","airing":true,
   "score":8.7,"rank":10,"popularity":50,"members":200000,
   "broadcast":{"day":"Sunday","time":"17:00","timezone":"Asia/Tokyo"},
   "genres":[{"mal_id":2,"name":"Adventure"}],
   "studios":[{"mal_id":1,"name":"Madhouse"}]}
],"pagination":{"last_visible_page":1,"has_next_page":false}}`

// --- helpers ---

func newTestClient(ts *httptest.Server) *jikan.Client {
	cfg := jikan.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	return jikan.NewClient(cfg)
}

// --- tests ---

func TestSearchAnimeSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeAnimeJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.SearchAnime(context.Background(), "naruto", 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent header not sent")
	}
	if !strings.Contains(gotUA, "jikan") {
		t.Errorf("User-Agent = %q, want it to contain 'jikan'", gotUA)
	}
}

func TestSearchAnimeParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeAnimeJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.SearchAnime(context.Background(), "naruto", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	got := items[0]
	if got.MalID != 20 {
		t.Errorf("MalID = %d, want 20", got.MalID)
	}
	if got.Title != "Naruto" {
		t.Errorf("Title = %q, want Naruto", got.Title)
	}
	if got.TitleEnglish != "Naruto" {
		t.Errorf("TitleEnglish = %q, want Naruto", got.TitleEnglish)
	}
	if got.Score != 8.02 {
		t.Errorf("Score = %v, want 8.02", got.Score)
	}
	if got.Episodes != 220 {
		t.Errorf("Episodes = %d, want 220", got.Episodes)
	}
	if len(got.Genres) != 2 {
		t.Errorf("len(Genres) = %d, want 2", len(got.Genres))
	}
	if got.Genres[0].Name != "Action" {
		t.Errorf("Genres[0].Name = %q, want Action", got.Genres[0].Name)
	}
	if len(got.Studios) != 1 || got.Studios[0].Name != "Pierrot" {
		t.Errorf("Studios = %v, want [{1 Pierrot}]", got.Studios)
	}
}

func TestSearchAnimeLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeAnimeJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.SearchAnime(context.Background(), "naruto", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1 (limit=1)", len(items))
	}
}

func TestSearchAnimeRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeAnimeJSON)
	}))
	defer ts.Close()

	cfg := jikan.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := jikan.NewClient(cfg)

	_, err := c.SearchAnime(context.Background(), "naruto", 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestTopAnimeTypePassed(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeAnimeJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.TopAnime(context.Background(), "movie", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "type=movie") {
		t.Errorf("query = %q, want type=movie in it", gotURL)
	}
}

func TestGetAnimeParsesItem(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeAnimeFullJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	item, err := c.GetAnime(context.Background(), 16498)
	if err != nil {
		t.Fatal(err)
	}
	if item.MalID != 16498 {
		t.Errorf("MalID = %d, want 16498", item.MalID)
	}
	if item.Title != "Shingeki no Kyojin" {
		t.Errorf("Title = %q, want Shingeki no Kyojin", item.Title)
	}
	if item.TitleEnglish != "Attack on Titan" {
		t.Errorf("TitleEnglish = %q, want Attack on Titan", item.TitleEnglish)
	}
	if item.Episodes != 25 {
		t.Errorf("Episodes = %d, want 25", item.Episodes)
	}
	if item.Rank != 75 {
		t.Errorf("Rank = %d, want 75", item.Rank)
	}
	if item.Synopsis == "" {
		t.Error("Synopsis should not be empty")
	}
}

func TestSearchMangaParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeMangaJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.SearchManga(context.Background(), "naruto", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.MalID != 11 {
		t.Errorf("MalID = %d, want 11", got.MalID)
	}
	if got.Volumes != 72 {
		t.Errorf("Volumes = %d, want 72", got.Volumes)
	}
	if got.Chapters != 700 {
		t.Errorf("Chapters = %d, want 700", got.Chapters)
	}
	if got.Rank != 640 {
		t.Errorf("Rank = %d, want 640", got.Rank)
	}
}

func TestSearchCharactersParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCharactersJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.SearchCharacters(context.Background(), "Naruto", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.MalID != 17 {
		t.Errorf("MalID = %d, want 17", got.MalID)
	}
	if got.Name != "Naruto Uzumaki" {
		t.Errorf("Name = %q, want Naruto Uzumaki", got.Name)
	}
	if got.NameKanji != "うずまきナルト" {
		t.Errorf("NameKanji = %q, want うずまきナルト", got.NameKanji)
	}
	if got.Favorites != 47000 {
		t.Errorf("Favorites = %d, want 47000", got.Favorites)
	}
	if len(got.Nicknames) != 2 {
		t.Errorf("len(Nicknames) = %d, want 2", len(got.Nicknames))
	}
}

func TestSearchPeopleParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakePeopleJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.SearchPeople(context.Background(), "Takeuchi", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.MalID != 185 {
		t.Errorf("MalID = %d, want 185", got.MalID)
	}
	if got.GivenName != "Junko" {
		t.Errorf("GivenName = %q, want Junko", got.GivenName)
	}
	if got.FamilyName != "Takeuchi" {
		t.Errorf("FamilyName = %q, want Takeuchi", got.FamilyName)
	}
	if got.Birthday != "1972-01-01T00:00:00+00:00" {
		t.Errorf("Birthday = %q, want 1972-01-01T00:00:00+00:00", got.Birthday)
	}
	if got.Favorites != 25000 {
		t.Errorf("Favorites = %d, want 25000", got.Favorites)
	}
}

func TestSeasonURLShape(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = fmt.Fprint(w, fakeSeasonJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Season(context.Background(), 2024, "winter", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "2024") {
		t.Errorf("path = %q, want 2024 in it", gotPath)
	}
	if !strings.Contains(gotPath, "winter") {
		t.Errorf("path = %q, want 'winter' in it", gotPath)
	}
}

func TestSeasonParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSeasonJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Season(context.Background(), 2023, "fall", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.Season != "fall" {
		t.Errorf("Season = %q, want fall", got.Season)
	}
	if got.Year != 2023 {
		t.Errorf("Year = %d, want 2023", got.Year)
	}
}

func TestScheduleParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeScheduleJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Schedule(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	got := items[0]
	if got.Title == "" {
		t.Error("Title should not be empty")
	}
	if got.Airing != true {
		t.Error("Airing should be true for schedule entry")
	}
	if got.Episodes != 0 {
		t.Errorf("Episodes = %d, want 0 (null in JSON)", got.Episodes)
	}
	if got.Broadcast.Day != "Sunday" {
		t.Errorf("Broadcast.Day = %q, want Sunday", got.Broadcast.Day)
	}
}
