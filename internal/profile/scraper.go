package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/logger"
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
		// Endpoints that can be called to obtain profiles.
		Targets []ScrapeTarget
	}

	// The ScrapeTarget type describes a single pprof endpoint that can be called to obtain a profile.
	ScrapeTarget struct {
		// The target address, should include scheme, host & port.
		Address string `json:"address"`
		// The path to the pprof profile endpoint, including leading slash. Defaults to /debug/pprof/profile if
		// unset.
		Path string `json:"path"`
	}

	// The Scraper type is used to perform periodic sampling of pprof profiles given a selection of valid
	// targets. These profiles are then forwarded to the configured profile server.
	Scraper struct {
		app             string
		sampleSize      uint
		scrapeFrequency time.Duration
		profileDuration time.Duration
		targets         []ScrapeTarget

		client Client
		rand   *rand.Rand
	}
)

// NewScraper returns a new instance of the Scraper type using the provided configuration.
func NewScraper(client Client, config ScrapeConfig) *Scraper {
	return &Scraper{
		sampleSize:      config.SampleSize,
		targets:         config.Targets,
		profileDuration: config.ProfileDuration,
		scrapeFrequency: config.ScrapeFrequency,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		client:          client,
		app:             config.App,
	}
}

// Scrape configured targets. This method blocks until the provided context is cancelled.
func (s *Scraper) Scrape(ctx context.Context) error {
	ticker := time.NewTicker(s.scrapeFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var group sync.WaitGroup

			for target := range s.sample(ctx) {
				group.Add(1)
				go s.forwardProfile(ctx, &group, target)
			}

			group.Wait()
		}
	}
}

func (s *Scraper) sample(ctx context.Context) iter.Seq[ScrapeTarget] {
	size := int(s.sampleSize)
	targets := slices.Clone(s.targets)

	if size > len(targets) {
		size = len(targets)
	}

	s.rand.Shuffle(len(targets), func(i, j int) {
		targets[i], targets[j] = targets[j], targets[i]
	})

	return func(yield func(ScrapeTarget) bool) {
		for _, target := range targets[:size] {
			if ctx.Err() != nil {
				return
			}

			if !yield(target) {
				return
			}
		}
	}
}

func (s *Scraper) forwardProfile(ctx context.Context, group *sync.WaitGroup, target ScrapeTarget) {
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

func (cfg ScrapeConfig) Validate() error {
	if cfg.SampleSize <= 0 {
		return errors.New("sample size must be greater than 0")
	}
	if cfg.ScrapeFrequency <= 0 {
		return errors.New("scrape frequency must be greater than 0")
	}

	if cfg.ProfileDuration <= 0 {
		return errors.New("profile duration must be greater than 0")
	}

	if !IsValidAppName(cfg.App) {
		return errors.New("application name is invalid")
	}

	if len(cfg.Targets) == 0 {
		return errors.New("at least one target must be set")
	}

	for i, target := range cfg.Targets {
		if target.Address == "" {
			return fmt.Errorf("targets[%d].address must be set", i)
		}
	}

	return nil
}

// LoadScrapeConfiguration attempts to parse the file at the specified location and decode it into an array of targets
// that can be scraped. The file is expected to be in JSON encoding.
func LoadScrapeConfiguration(ctx context.Context, location string) ([]ScrapeTarget, error) {
	f, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	defer closers.Close(ctx, f)

	var targets []ScrapeTarget
	if err = json.NewDecoder(f).Decode(&targets); err != nil {
		return nil, err
	}

	return targets, nil
}
