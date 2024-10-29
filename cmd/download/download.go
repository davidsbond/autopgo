// Package download provides the command for downloading a profile from the server.
package download

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance used for the download command.
func Command() *cobra.Command {
	var (
		apiURL string
	)

	cmd := &cobra.Command{
		Use:     "download <app>",
		Short:   "Download a profile",
		GroupID: "utils",
		Long:    "Download a combined pprof profile for an application from the autopgo server",
		Example: "autopgo download hello-world >> out.profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app := args[0]

			if !profile.IsValidAppName(app) {
				return fmt.Errorf("%s is not a valid application name", app)
			}

			return client.New(apiURL).Download(ctx, app, os.Stdout)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&apiURL, "api-url", "http://localhost:8080", "Base URL of the autopgo server")

	return cmd
}
