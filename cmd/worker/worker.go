// Package worker provides the command-line entrypoint to the profile worker.
package worker

import (
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/event"
	"github.com/davidsbond/autopgo/internal/logger"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/internal/server"
)

// Command returns a cobra.Command instance used to run the worker.
func Command() *cobra.Command {
	var (
		eventReaderURL string
		eventWriterURL string
		blobStoreURL   string
		port           int
	)

	cmd := &cobra.Command{
		Use:     "worker",
		Short:   "Run the autopgo worker",
		GroupID: "component",
		Long: "Starts the autopgo worker, a service responsible for handling inbound events for newly uploaded profiles " +
			"and merging them with existing profiles.\n\n" +
			"The URL based flags follow the semantics based on the individual provider. Supported provides include AWS, " +
			"GCP & Azure. See the gocloud.dev documentation for further information on configuring these flags for your " +
			"specific provider.",
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

			worker := profile.NewWorker(blobs, writer)

			types := []string{
				profile.EventTypeMerged,
				profile.EventTypeUploaded,
			}

			group, ctx := errgroup.WithContext(ctx)
			group.Go(func() error {
				return reader.Read(ctx, types, worker.HandleEvent)
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
	flags.StringVar(&eventReaderURL, "event-reader-url", "", "The URL to use for reading from the event bus")
	flags.StringVar(&eventWriterURL, "event-writer-url", "", "The URL to use for writing to the event bus")
	flags.StringVar(&blobStoreURL, "blob-store-url", "", "The URL to use for connecting to blob storage")
	flags.IntVarP(&port, "port", "p", 8081, "Port to use for HTTP traffic")

	cmd.MarkPersistentFlagRequired("blob-store-url")
	cmd.MarkPersistentFlagRequired("event-reader-url")
	cmd.MarkPersistentFlagRequired("event-writer-url")

	return cmd
}
