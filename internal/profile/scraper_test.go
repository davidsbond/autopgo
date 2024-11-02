package profile_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/profile/mocks"
)

func TestScraper_Scrape(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Config   profile.ScrapeConfig
		Setup    func(client *mocks.MockClient)
		Duration time.Duration
	}{
		{
			Name:     "successful scrape",
			Duration: 5 * time.Second,
			Config: profile.ScrapeConfig{
				SampleSize:      3,
				ProfileDuration: time.Second * 30,
				App:             "test",
				ScrapeFrequency: time.Second,
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
					{
						Address: "http://localhost:8081",
						Path:    "/debug/pprof/profile",
					},
					{
						Address: "http://localhost:8082",
						Path:    "/debug/pprof/profile",
					},
				},
			},
			Setup: func(client *mocks.MockClient) {
				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test", "http://localhost:8080/debug/pprof/profile", time.Second*30).
					Return(nil)

				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test", "http://localhost:8081/debug/pprof/profile", time.Second*30).
					Return(nil)

				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test", "http://localhost:8082/debug/pprof/profile", time.Second*30).
					Return(nil)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			client := mocks.NewMockClient(t)
			if tc.Setup != nil {
				tc.Setup(client)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tc.Duration)
			defer cancel()

			err := profile.NewScraper(client, tc.Config).Scrape(ctx)
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return
			case err != nil:
				assert.Fail(t, err.Error())
			}
		})
	}
}

func TestScrapeConfig_Validate(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		Config       profile.ScrapeConfig
		ExpectsError bool
	}{
		{
			Name: "valid profile",
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: time.Second * 30,
				ScrapeFrequency: time.Minute,
				App:             "test",
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "no targets",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: time.Second * 30,
				ScrapeFrequency: time.Minute,
			},
		},
		{
			Name:         "invalid scrape frequency",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: time.Second * 30,
				App:             "test",
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "invalid profile duration",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ScrapeFrequency: time.Minute,
				App:             "test",
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "invalid sample size",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				ProfileDuration: time.Second * 30,
				ScrapeFrequency: time.Minute,
				App:             "test",
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "target with invalid app",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: time.Second * 30,
				ScrapeFrequency: time.Minute,
				App:             "_/@~",
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080",
						Path:    "/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "target missing address",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: time.Second * 30,
				ScrapeFrequency: time.Minute,
				App:             "test",
				Targets: []profile.ScrapeTarget{
					{},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Config.Validate()
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
