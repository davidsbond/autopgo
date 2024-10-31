// Package delete provides the command for removing a profile from the server.
package delete

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/internal/profile"
	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance used for the delete command.
func Command() *cobra.Command {
	var (
		apiURL string
	)

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a profile",
		GroupID: "utils",
		Args:    cobra.ExactArgs(1),
		Long: "Deletes the profile for an application. This will also delete profiles pending merge for the application.\n\n" +
			"This command will not prevent further profiles from being created if they are still being scraped, so you\n" +
			"should stop all scraping prior to using this command unless you want to start profiling from scratch.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := args[0]
			ctx := cmd.Context()

			if !profile.IsValidAppName(app) {
				return fmt.Errorf("%s is not a valid application name", app)
			}

			return client.New(apiURL).Delete(ctx, app)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server")

	return cmd
}
