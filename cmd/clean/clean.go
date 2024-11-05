// Package clean provides the command for removing old or large profiles from the server.
package clean

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance that runs the clean command.
func Command() *cobra.Command {
	var (
		apiURL     string
		olderThan  time.Duration
		largerThan int64
	)

	cmd := &cobra.Command{
		Use:     "clean",
		Short:   "Clean up profiles",
		GroupID: "utils",
		Long: "Deletes profiles that have exceeded a specified size or have not been modified for a specified amount of\n" +
			"time.\n\n" +
			"This command is destructive and will not prevent further profiles from being uploaded and merged for\n" +
			"any of the applications whose profiles are deleted.",
		Example: "autopgo clean --older-than 48h\n" +
			"autopgo clean --larger-than 1000000",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if largerThan == 0 && olderThan == 0 {
				return errors.New("one of --older-than or --larger-than must be set")
			}

			cleaned, err := client.New(apiURL).Clean(ctx, olderThan, largerThan)
			if err != nil {
				return err
			}

			for _, profile := range cleaned {
				if _, err = fmt.Fprintf(os.Stdout, "Deleted profile '%s'\n", profile); err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server")
	flags.DurationVarP(&olderThan, "older-than", "d", 0, "The duration a profile must have remained static for")
	flags.Int64VarP(&largerThan, "larger-than", "s", 0, "The minimum size a profile must be")

	return cmd
}
