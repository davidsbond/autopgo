// Package list provides the command-line entrypoint to the list command.
package list

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/davidsbond/autopgo/pkg/client"
)

// Command returns a cobra.Command instance that allows listing and printing profile information from the CLI.
func Command() *cobra.Command {
	var (
		apiURL string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all profiles",
		Long:    "Prints information on all profiles currently stored within the server",
		GroupID: "utils",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			profiles, err := client.New(apiURL).List(ctx)
			if err != nil {
				return err
			}

			writer := tabwriter.NewWriter(os.Stdout, 4, 1, 2, ' ', tabwriter.TabIndent)
			if _, err = fmt.Fprintln(writer, "NAME\tSIZE\tLAST MODIFIED"); err != nil {
				return err
			}

			for _, profile := range profiles {
				lastModified := time.Since(profile.LastModified).Truncate(time.Second)

				if _, err = fmt.Fprintf(writer, "%s\t%d\t%s\n", profile.Key, profile.Size, lastModified); err != nil {
					return err
				}
			}

			return writer.Flush()
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&apiURL, "api-url", "u", "http://localhost:8080", "Base URL of the autopgo server.")

	return cmd
}
