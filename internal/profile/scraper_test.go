package profile_test

import (
	"testing"

	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/profile/mocks"
)

func TestScraper_Scrape(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name   string
		Config profile.ScrapeConfig
		Setup  func(client *mocks.MockClient)
	}{}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			client := mocks.NewMockClient(t)
			if tc.Setup != nil {
				tc.Setup(client)
			}

			//scraper := profile.NewScraper(client, tc.Config)
		})
	}
}
