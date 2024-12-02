// Package worker provides the command-line entrypoint to the profile worker.
package worker

import (
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/event"
	"github.com/davidsbond/autopgo/internal/logger"
	"github.com/davidsbond/autopgo/internal/operation"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/server"
)

// Command returns a cobra.Command instance used to run the worker.
func Command() *cobra.Command {
	var (
		eventReaderURL string
		eventWriterURL string
		blobStoreURL   string
		prune          string
		port           int
		debug          bool
	)

	cmd := &cobra.Command{
		Use:     "worker",
		Short:   "Run the autopgo worker",
		GroupID: "component",
		Long: "Starts the autopgo worker, a service responsible for handling inbound events for newly uploaded profiles\n" +
			"and merging them with existing profiles.\n\n" +
			"The --prune flag can be optionally provided to parse a JSON-encoded configuration file that describes how\n" +
			"profiles should be pruned as they are merged. See the documentation for more information on configuring\n" +
			"pruning.\n\n" +
			"The URL based flags follow the semantics based on the individual provider. Supported provides include AWS,\n" +
			"GCP & Azure. See the gocloud.dev documentation for further information on configuring these flags for your\n" +
			"specific provider.",
		Example: "autopgo worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			reader, err := event.NewReader(ctx, eventReaderURL)
			if err != nil {
				return err
			}
			defer closers.Close(ctx, reader)

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

			pruning, err := profile.LoadPruneConfig(ctx, prune)
			switch {
			case err != nil:
				return err
			case len(pruning) == 0:
				logger.FromContext(ctx).Warn("worker starting with no prune rules")
			}

			worker := profile.NewWorker(blobs, writer, pruning)

			types := []string{
				profile.EventTypeMerged,
				profile.EventTypeUploaded,
				profile.EventTypeDeleted,
			}

			group, ctx := errgroup.WithContext(ctx)
			group.Go(func() error {
				return reader.Read(ctx, types, worker.HandleEvent)
			})
			group.Go(func() error {
				return server.Run(ctx, server.Config{
					Debug: debug,
					Port:  port,
					Controllers: []server.Controller{
						operation.NewHTTPController([]operation.Checker{
							blobs,
							reader,
							writer,
						}),
					},
					Middleware: []server.Middleware{
						logger.Middleware(logger.FromContext(ctx)),
					},
				})
			})

			return group.Wait()
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&eventReaderURL, "event-reader-url", "", "The URL to use for reading from the event bus")
	flags.StringVar(&eventWriterURL, "event-writer-url", "", "The URL to use for writing to the event bus")
	flags.StringVar(&blobStoreURL, "blob-store-url", "", "The URL to use for connecting to blob storage")
	flags.IntVarP(&port, "port", "p", 8081, "Port to use for HTTP traffic")
	flags.StringVar(&prune, "prune", "", "Location of the configuration file for profile pruning")
	flags.BoolVar(&debug, "debug", false, "Enable debug endpoints")

	cmd.MarkPersistentFlagRequired("blob-store-url")
	cmd.MarkPersistentFlagRequired("event-reader-url")
	cmd.MarkPersistentFlagRequired("event-writer-url")

	return cmd
}
