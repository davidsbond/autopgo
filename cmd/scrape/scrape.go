// Package scrape provides the command-line entrypoint to the profile scraper.
package scrape

import (
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/davidsbond/autopgo/internal/logger"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/server"
	"github.com/davidsbond/autopgo/internal/target"
	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance used to run the scraper.
func Command() *cobra.Command {
	var (
		apiURL     string
		port       int
		sampleSize uint
		duration   time.Duration
		frequency  time.Duration
		app        string
	)

	cmd := &cobra.Command{
		Use:     "scrape <config>",
		Short:   "Run the autopgo scraper",
		GroupID: "component",
		Long: "Starts the profile scraper that will obtain profiles from targets listed within the configuration file,\n" +
			"forwarding those profiles to the configured server.\n\n" +
			"Sample sizes & profiling frequency can be tuned using command-line flags. See the documentation for\n" +
			"more information on the contents of the scraper configuration file.",
		Example: "autopgo scrape",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			source, err := target.NewFileSource(ctx, args[0])
			if err != nil {
				return err
			}

			cl := client.New(apiURL)
			scraper := profile.NewScraper(cl, profile.ScrapeConfig{
				SampleSize:      sampleSize,
				ProfileDuration: duration,
				ScrapeFrequency: frequency,
				App:             app,
			})

			group, ctx := errgroup.WithContext(ctx)
			group.Go(func() error {
				return scraper.Scrape(ctx, source)
			})
			group.Go(func() error {
				return server.Run(ctx, server.Config{
					Port: port,
					Middleware: []server.Middleware{
						logger.Middleware(logger.FromContext(ctx)),
					},
				})
			})

			return group.Wait()
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server")
	flags.IntVarP(&port, "port", "p", 8082, "Port to use for HTTP traffic")
	flags.StringVarP(&app, "app", "a", "", "The name of the application being profiled")
	flags.UintVarP(&sampleSize, "sample-size", "s", 0, "The maximum number of targets to scrape concurrently")
	flags.DurationVarP(&duration, "duration", "d", time.Second*30, "How long to profile targets for")
	flags.DurationVarP(&frequency, "frequency", "f", time.Minute, "Interval between scraping targets")

	cmd.MarkFlagRequired("app")
	cmd.MarkFlagRequired("sample-size")

	return cmd
}
