package profile

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
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
		SampleSize int `json:"sampleSize"`
		// How long targets should be profiled for, in seconds.
		ProfileDuration int `json:"profileDuration"`
		// How frequently profiles are sampled, in seconds.
		ScrapeFrequency int `json:"scrapeFrequency"`
		// Endpoints that can be called to obtain profiles.
		Targets []ScrapeTarget `json:"targets"`
	}

	// The ScrapeTarget type describes a single pprof endpoint that can be called to obtain a profile.
	ScrapeTarget struct {
		// The application associated with the profile.
		App string `json:"app"`
		// The full target address, including the path for pprof.
		Address string `json:"address"`
	}

	// The Scraper type is used to perform periodic sampling of pprof profiles given a selection of valid
	// targets. These profiles are then forwarded to the configured profile server.
	Scraper struct {
		sampleSize      int
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
		profileDuration: time.Duration(config.ProfileDuration) * time.Second,
		scrapeFrequency: time.Duration(config.ScrapeFrequency) * time.Second,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		client:          client,
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
	size := s.sampleSize
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
		slog.String("target.app", target.App),
	)

	log.DebugContext(ctx, "profiling target")
	data, err := s.client.Profile(ctx, target.Address, s.profileDuration)
	if err != nil {
		log.With(slog.String("error", err.Error())).
			ErrorContext(ctx, "failed to profile target")
		return
	}

	defer closers.Close(ctx, data)
	if err = s.client.Upload(ctx, target.App, data); err != nil {
		log.With(slog.String("error", err.Error())).
			ErrorContext(ctx, "failed to upload profile")
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

	if len(cfg.Targets) == 0 {
		return errors.New("at least one target must be set")
	}

	for i, target := range cfg.Targets {
		if target.App == "" {
			return fmt.Errorf("targets[%d].app must be set", i)
		}
		if target.Address == "" {
			return fmt.Errorf("targets[%d].address must be set", i)
		}
		if !IsValidAppName(target.App) {
			return fmt.Errorf("targets[%d].app is invalid", i)
		}
	}

	return nil
}
