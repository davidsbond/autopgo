// Package scrape provides the command-line entrypoint to the profile scraper.
package scrape

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/profile"
)

// Command returns a cobra.Command instance used to run the scraper.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scrape <config>",
		Short:   "Run the autopgo scraper",
		GroupID: "component",
		Long: "Starts the profile scraper that will obtain profiles from targets listed within the configuration file,\n" +
			"forwarding those profiles to the configured server.\n\n" +
			"Sample sizes & profiling frequency can be tuned using this configuration file. See the documentation for\n" +
			"more information on the contents of the scraper configuration file.",
		Example: "autopgo scrape ./config.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			configLocation := args[0]
			f, err := os.Open(configLocation)
			if err != nil {
				return err
			}
			defer closers.Close(ctx, f)

			var config profile.ScrapeConfig
			if err = json.NewDecoder(f).Decode(&config); err != nil {
				return err
			}

			if err = config.Validate(); err != nil {
				return err
			}

			return profile.NewScraper(config).Scrape(ctx)
		},
	}

	return cmd
}
