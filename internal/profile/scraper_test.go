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
	"github.com/davidsbond/autopgo/internal/target"
)

func TestScraper_Scrape(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Config   profile.ScrapeConfig
		Setup    func(client *mocks.MockClient, source *mocks.MockTargetSource)
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
			},
			Setup: func(client *mocks.MockClient, source *mocks.MockTargetSource) {
				source.EXPECT().
					List(mock.Anything).
					Return([]target.Target{
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
					}, nil)

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
			source := mocks.NewMockTargetSource(t)

			if tc.Setup != nil {
				tc.Setup(client, source)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tc.Duration)
			defer cancel()

			err := profile.NewScraper(client, tc.Config).Scrape(ctx, source)
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return
			case err != nil:
				assert.Fail(t, err.Error())
			}
		})
	}
}
