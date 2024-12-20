// Package upload provides the command for uploading a profile to the server.
package upload

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance used for the upload command.
func Command() *cobra.Command {
	var (
		apiURL string
		app    string
	)

	cmd := &cobra.Command{
		Use:     "upload <file>",
		Short:   "Upload a profile",
		GroupID: "utils",
		Long:    "Upload a pprof profile from a Go application to the autopgo server",
		Example: "autopgo upload -a hello-world example.profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !profile.IsValidAppName(app) {
				return fmt.Errorf("%s is not a valid application name", app)
			}

			ctx := cmd.Context()
			location := args[0]

			cl := client.New(apiURL)

			file, err := os.Open(location)
			switch {
			case errors.Is(err, os.ErrNotExist):
				return fmt.Errorf("file %s does not exist", location)
			case err != nil:
				return err
			}

			return cl.Upload(ctx, app, file)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server")
	flags.StringVarP(&app, "app", "a", "", "The name of the application")

	cmd.MarkPersistentFlagRequired("app")

	return cmd
}
