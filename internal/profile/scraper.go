package profile

import (
	"context"
	"iter"
	"log/slog"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/davidsbond/autopgo/internal/logger"
	"github.com/davidsbond/autopgo/internal/target"
)

type (
	// The ScrapeConfig type describes the configuration used by the Scraper to sample pprof profiles from
	// specified targets.
	ScrapeConfig struct {
		// How many profiles to obtain after the ProfileDuration has passed.
		SampleSize uint
		// How long targets should be profiled for, in seconds.
		ProfileDuration time.Duration
		// How frequently profiles are sampled, in seconds.
		ScrapeFrequency time.Duration
		// The application this scraper instance is collecting profiles for.
		App string
	}

	// The Scraper type is used to perform periodic sampling of pprof profiles given a selection of valid
	// targets. These profiles are then forwarded to the configured profile server.
	Scraper struct {
		app             string
		sampleSize      uint
		scrapeFrequency time.Duration
		profileDuration time.Duration

		client Client
		rand   *rand.Rand
	}

	// The TargetSource interface describes types that can list scraping targets.
	TargetSource interface {
		// List should return all targets that are available to be scraped.
		List(ctx context.Context) ([]target.Target, error)
	}
)

// NewScraper returns a new instance of the Scraper type using the provided configuration.
func NewScraper(client Client, config ScrapeConfig) *Scraper {
	return &Scraper{
		sampleSize:      config.SampleSize,
		profileDuration: config.ProfileDuration,
		scrapeFrequency: config.ScrapeFrequency,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		client:          client,
		app:             config.App,
	}
}

// Scrape configured targets. This method blocks until the provided context is cancelled.
func (s *Scraper) Scrape(ctx context.Context, source TargetSource) error {
	ticker := time.NewTicker(s.scrapeFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			targets, err := source.List(ctx)
			if err != nil {
				return err
			}

			var group sync.WaitGroup
			for t := range s.sample(ctx, targets) {
				group.Add(1)
				go s.forwardProfile(ctx, &group, t)
			}

			group.Wait()
		}
	}
}

func (s *Scraper) sample(ctx context.Context, targets []target.Target) iter.Seq[target.Target] {
	size := int(s.sampleSize)

	if size > len(targets) {
		size = len(targets)
	}

	s.rand.Shuffle(len(targets), func(i, j int) {
		targets[i], targets[j] = targets[j], targets[i]
	})

	return func(yield func(target.Target) bool) {
		for _, t := range targets[:size] {
			if ctx.Err() != nil {
				return
			}

			if !yield(t) {
				return
			}
		}
	}
}

func (s *Scraper) forwardProfile(ctx context.Context, group *sync.WaitGroup, target target.Target) {
	defer group.Done()

	log := logger.FromContext(ctx).With(
		slog.String("target.address", target.Address),
		slog.String("target.app", s.app),
	)

	u, err := url.Parse(target.Address)
	if err != nil {
		log.With(slog.String("error", err.Error())).
			ErrorContext(ctx, "failed to parse target address")
		return
	}

	u.Path = "/debug/pprof/profile"
	if target.Path != "" {
		u.Path = target.Path
	}

	log.DebugContext(ctx, "profiling target")
	if err = s.client.ProfileAndUpload(ctx, s.app, u.String(), s.profileDuration); err != nil {
		log.With(slog.String("error", err.Error())).
			ErrorContext(ctx, "failed to profile target")
		return
	}

	log.DebugContext(ctx, "uploaded profile")
}
