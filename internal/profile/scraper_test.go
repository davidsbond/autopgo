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
				ProfileDuration: 30,
				ScrapeFrequency: 1,
				Targets: []profile.ScrapeTarget{
					{
						App:     "test",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
					{
						App:     "test-1",
						Address: "http://localhost:8081/debug/pprof/profile",
					},
					{
						App:     "test-2",
						Address: "http://localhost:8082/debug/pprof/profile",
					},
				},
			},
			Setup: func(client *mocks.MockClient) {
				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test", "http://localhost:8080/debug/pprof/profile", time.Second*30).
					Return(nil)

				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test-1", "http://localhost:8081/debug/pprof/profile", time.Second*30).
					Return(nil)

				client.EXPECT().
					ProfileAndUpload(mock.Anything, "test-2", "http://localhost:8082/debug/pprof/profile", time.Second*30).
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
				ProfileDuration: 10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						App:     "test",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "no targets",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: 10,
				ScrapeFrequency: 30,
			},
		},
		{
			Name:         "invalid scrape frequency",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: 10,
				Targets: []profile.ScrapeTarget{
					{
						App:     "test",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "invalid profile duration",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						App:     "test",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "invalid sample size",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				ProfileDuration: 10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						App:     "test",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "target missing app",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: 10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "target with invalid app",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: 10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						App:     "_/@~",
						Address: "http://localhost:8080/debug/pprof/profile",
					},
				},
			},
		},
		{
			Name:         "target missing address",
			ExpectsError: true,
			Config: profile.ScrapeConfig{
				SampleSize:      10,
				ProfileDuration: 10,
				ScrapeFrequency: 30,
				Targets: []profile.ScrapeTarget{
					{
						App: "test",
					},
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
