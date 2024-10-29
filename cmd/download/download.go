// Package download provides the command for downloading a profile from the server.
package download

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance used for the download command.
func Command() *cobra.Command {
	var (
		apiURL string
		output string
	)

	cmd := &cobra.Command{
		Use:     "download <app>",
		Short:   "Download a profile",
		GroupID: "utils",
		Long:    "Download a combined pprof profile for an application from the autopgo server",
		Example: "autopgo download hello-world",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app := args[0]

			if !profile.IsValidAppName(app) {
				return fmt.Errorf("%s is not a valid application name", app)
			}

			f, err := os.Create(output)
			if err != nil {
				return err
			}

			defer closers.Close(ctx, f)
			return client.New(apiURL).Download(ctx, app, f)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server.")
	flags.StringVarP(&output, "output", "o", "default.pgo", "Where to place the downloaded profile on the local filesystem.")

	return cmd
}
