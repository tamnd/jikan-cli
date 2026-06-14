package jikan

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes jikan as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/jikan-cli/jikan"
//
// The same Domain also builds the standalone jikan binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the jikan driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "jikan",
		Hosts:  []string{Host, "myanimelist.net"},
		Identity: kit.Identity{
			Binary: "jikan",
			Short:  "MyAnimeList anime and manga data via the Jikan API",
			Long: `jikan fetches anime, manga, characters, and seasonal data from MyAnimeList
via the public Jikan v4 API. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/jikan-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "anime",
		Group:   "read",
		List:    true,
		Summary: "Search anime by title",
	}, animeOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "anime-id",
		Group:   "read",
		Summary: "Get anime by MAL ID",
	}, animeIDOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "top",
		Group:   "read",
		List:    true,
		Summary: "Top anime list",
	}, topOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "manga",
		Group:   "read",
		List:    true,
		Summary: "Search manga by title",
	}, mangaOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "characters",
		Group:   "read",
		List:    true,
		Summary: "Search characters by name",
	}, charactersOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "people",
		Group:   "read",
		List:    true,
		Summary: "Search voice actors and staff",
	}, peopleOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "season",
		Group:   "read",
		List:    true,
		Summary: "Anime for a specific season",
	}, seasonOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "schedule",
		Group:   "read",
		List:    true,
		Summary: "Today's airing schedule",
	}, scheduleOp)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type animeInput struct {
	Query  string  `kit:"arg"          help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type animeIDInput struct {
	ID     int     `kit:"arg"    help:"MAL anime ID"`
	Client *Client `kit:"inject"`
}

type topInput struct {
	Type   string  `kit:"flag"         help:"anime type (tv,movie,ova,special,ona,music)" default:"tv"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type mangaInput struct {
	Query  string  `kit:"arg"          help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type charactersInput struct {
	Query  string  `kit:"arg"          help:"character name"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type peopleInput struct {
	Query  string  `kit:"arg"          help:"person name"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type seasonInput struct {
	Year   int     `kit:"arg"          help:"year (e.g. 2024)"`
	Season string  `kit:"arg"          help:"season (winter, spring, summer, fall)"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type scheduleInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func animeOp(ctx context.Context, in animeInput, emit func(Anime) error) error {
	items, err := in.Client.SearchAnime(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func animeIDOp(ctx context.Context, in animeIDInput, emit func(Anime) error) error {
	item, err := in.Client.GetAnime(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(*item)
}

func topOp(ctx context.Context, in topInput, emit func(Anime) error) error {
	items, err := in.Client.TopAnime(ctx, in.Type, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func mangaOp(ctx context.Context, in mangaInput, emit func(Manga) error) error {
	items, err := in.Client.SearchManga(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func charactersOp(ctx context.Context, in charactersInput, emit func(Character) error) error {
	items, err := in.Client.SearchCharacters(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func peopleOp(ctx context.Context, in peopleInput, emit func(Person) error) error {
	items, err := in.Client.SearchPeople(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func seasonOp(ctx context.Context, in seasonInput, emit func(Anime) error) error {
	items, err := in.Client.Season(ctx, in.Year, in.Season, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func scheduleOp(ctx context.Context, in scheduleInput, emit func(Anime) error) error {
	items, err := in.Client.Schedule(ctx, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver: pure string functions, no network ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty jikan reference")
	}
	return "anime", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "anime":
		return "https://myanimelist.net/anime/" + id, nil
	default:
		return "", errs.Usage("jikan has no resource type %q", uriType)
	}
}
