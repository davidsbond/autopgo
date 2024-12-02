// Package server provides the command-line entrypoint to the profile server.
package server

import (
	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/event"
	"github.com/davidsbond/autopgo/internal/logger"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/server"
)

// Command returns a cobra.Command instance used to run the server.
func Command() *cobra.Command {
	var (
		port           int
		eventWriterURL string
		blobStoreURL   string
		debug          bool
	)

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Run the autopgo server",
		GroupID: "component",
		Long: "Starts the autopgo server on the desired port, publishing events to the configured event bus.\n\n" +
			"The URL based flags follow the semantics based on the individual provider. Supported provides include AWS,\n" +
			"GCP & Azure. See the gocloud.dev documentation for further information on configuring these flags for your\n" +
			"specific provider.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			writer, err := event.NewWriter(ctx, eventWriterURL)
			if err != nil {
				return err
			}
			defer closers.Close(ctx, writer)

			blobs, err := blob.NewBucket(ctx, blobStoreURL)
			if err != nil {
				return err
			}
			defer closers.Close(ctx, blobs)

			return server.Run(ctx, server.Config{
				Debug: debug,
				Port:  port,
				Controllers: []server.Controller{
					profile.NewHTTPController(blobs, writer),
				},
				Middleware: []server.Middleware{
					logger.Middleware(logger.FromContext(ctx)),
				},
			})
		},
	}

	flags := cmd.PersistentFlags()
	flags.IntVarP(&port, "port", "p", 8080, "Port to use for HTTP traffic")
	flags.StringVar(&eventWriterURL, "event-writer-url", "", "The URL to use for writing to the event bus")
	flags.StringVar(&blobStoreURL, "blob-store-url", "", "The URL to use for connecting to blob storage")
	flags.BoolVar(&debug, "debug", false, "Enable debug endpoints")

	cmd.MarkPersistentFlagRequired("blob-store-url")
	cmd.MarkPersistentFlagRequired("event-writer-url")

	return cmd
}
