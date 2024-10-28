// Package target provides the command-line entrypoint to the sample target server.
package target

import (
	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/server"
	"github.com/davidsbond/autopgo/internal/target"
)

func Command() *cobra.Command {
	var (
		port int
	)

	cmd := &cobra.Command{
		Use:   "target",
		Short: "Run an example scraping target",
		Long: "Starts a basic HTTP application that exposes pprof endpoints. This can be used to test the scraper\n" +
			"component.",
		GroupID: "utils",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			return server.Run(ctx, server.Config{
				Port: port,
				Controllers: []server.Controller{
					target.NewHTTPController(),
				},
			})
		},
	}

	flags := cmd.PersistentFlags()
	flags.IntVar(&port, "port", 8081, "Port to use for HTTP traffic")

	return cmd
}
