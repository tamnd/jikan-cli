package jikan

// Anime is one anime entry returned by the Jikan API.
// Episodes is 0 when the API reports null (ongoing series with unknown count).
type Anime struct {
	Rank     int      `json:"rank"`
	MalID    int      `json:"mal_id"`
	Title    string   `json:"title"`
	TitleEn  string   `json:"title_english"`
	Score    float64  `json:"score"`
	Episodes int      `json:"episodes"` // 0 when unknown/ongoing
	Type     string   `json:"type"`     // "TV", "Movie", "OVA", "ONA", "Special", "Music"
	Season   string   `json:"season"`   // "spring", "summer", "fall", "winter"
	Year     int      `json:"year"`
	Status   string   `json:"status"` // "Finished Airing", "Currently Airing", "Not yet aired"
	Genres   []string `json:"genres"`
	URL      string   `json:"url"` // myanimelist.net URL
}

// unexported: used only inside the package for JSON decoding.

type apiResponse struct {
	Data []apiAnime `json:"data"`
}

// apiAnime is the raw JSON shape from api.jikan.moe/v4.
// Episodes is *int because the API sends null for ongoing series.
type apiAnime struct {
	Rank     int        `json:"rank"`
	MalID    int        `json:"mal_id"`
	Title    string     `json:"title"`
	TitleEn  string     `json:"title_english"`
	Score    float64    `json:"score"`
	Episodes *int       `json:"episodes"`
	Type     string     `json:"type"`
	Season   string     `json:"season"`
	Year     int        `json:"year"`
	Status   string     `json:"status"`
	Genres   []apiGenre `json:"genres"`
	URL      string     `json:"url"`
}

type apiGenre struct {
	Name string `json:"name"`
}

// toAnime converts a wire apiAnime into the output Anime type.
// null episodes become 0.
func toAnime(a apiAnime) Anime {
	eps := 0
	if a.Episodes != nil {
		eps = *a.Episodes
	}
	genres := make([]string, len(a.Genres))
	for i, g := range a.Genres {
		genres[i] = g.Name
	}
	return Anime{
		Rank:     a.Rank,
		MalID:    a.MalID,
		Title:    a.Title,
		TitleEn:  a.TitleEn,
		Score:    a.Score,
		Episodes: eps,
		Type:     a.Type,
		Season:   a.Season,
		Year:     a.Year,
		Status:   a.Status,
		Genres:   genres,
		URL:      a.URL,
	}
}
