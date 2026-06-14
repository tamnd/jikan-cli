package jikan

import (
	"context"
	"time"

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
			Short:  "Top anime, search, and seasonal charts from MyAnimeList via Jikan",
			Long: `jikan fetches anime data from MyAnimeList through the open Jikan API
(api.jikan.moe). No API key or login required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/jikan-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// top: list the top-ranked anime by MAL score
	kit.Handle(app, kit.OpMeta{
		Name:    "top",
		Group:   "read",
		List:    true,
		Summary: "List top-ranked anime by MAL score",
	}, topOp)

	// search: search anime by title or keyword
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search anime by title or keyword",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchOp)

	// season: list anime from the current airing season
	kit.Handle(app, kit.OpMeta{
		Name:    "season",
		Group:   "read",
		List:    true,
		Summary: "List anime from the current airing season",
	}, seasonOp)
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

type topInput struct {
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type searchInput struct {
	Query  string        `kit:"arg" help:"search query"`
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type seasonInput struct {
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

// --- handlers ---

func topOp(ctx context.Context, in topInput, emit func(Anime) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	items, err := in.Client.Top(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func searchOp(ctx context.Context, in searchInput, emit func(Anime) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	items, err := in.Client.Search(ctx, in.Query, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func seasonOp(ctx context.Context, in seasonInput, emit func(Anime) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	items, err := in.Client.Season(ctx, limit)
	if err != nil {
		return mapErr(err)
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

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
