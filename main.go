//go:generate mockery

// Package main contains the entrypoint to autopgo.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/davidsbond/autopgo/cmd/download"
	"github.com/davidsbond/autopgo/cmd/list"
	"github.com/davidsbond/autopgo/cmd/scrape"
	"github.com/davidsbond/autopgo/cmd/server"
	"github.com/davidsbond/autopgo/cmd/target"
	"github.com/davidsbond/autopgo/cmd/upload"
	"github.com/davidsbond/autopgo/cmd/worker"
	"github.com/davidsbond/autopgo/internal/logger"
)

var (
	version = "dev"
	commit  = "HEAD"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	var (
		logLevel string
	)

	cmd := &cobra.Command{
		Use:     "autopgo",
		Version: fmt.Sprintf("%s (%s)", version, commit),
		Short:   "autopgo is a collection of services used to automating profile-guided optimization in Go",
		Long: "The autopgo CLI is a combination of individual services used to manage the upload, merging & retrieval of\n" +
			"pprof profiles generated by Go applications for performing profile-guided optimization.\n\n" +
			"It consists of 3 main components, the server, worker & scraper.\n\n" +
			"The server is used for uploading & downloading profiles to be passed into the go build command. The worker\n" +
			"is responsible for handling newly uploaded profiles and merging them into a single profile to be used for\n" +
			"profile-guided optimization. Finally, the scraper is used to sample profiles from a fleet of applications\n" +
			"exposing pprof endpoints.\n\n" +
			"Configuration relies heavily on URLs to configure access to event buses & blob storage for individual\n" +
			"cloud providers. Supported providers include AWS, GCP & Azure. Please see the gocloud.dev documentation\n" +
			"for semantics on individual cloud providers.\n\n" +
			"All command-line flags can also be set via environment variables, to use environment variables simply set\n" +
			"them using the following format:\n\n" +
			"export autopgo_<flag-name>\n\n" +
			"Replacing <flag-name> with the name of the command-line flag replacing dashes for underscores and using all\n" +
			"capital letters. For example, autopgo_LOG_LEVEL=error will map to the --log-level command-line flag.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx = logger.ToContext(ctx, slog.New(
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logger.LevelFromString(logLevel)}),
			))

			cmd.SetContext(ctx)
		},
	}

	cmd.AddGroup(
		&cobra.Group{ID: "component", Title: "Components:"},
		&cobra.Group{ID: "utils", Title: "Utilities:"},
	)

	cmd.AddCommand(
		worker.Command(),
		upload.Command(),
		download.Command(),
		server.Command(),
		scrape.Command(),
		target.Command(),
		list.Command(),
	)

	flags := cmd.PersistentFlags()
	flags.StringVarP(&logLevel, "log-level", "l", "info", "Sets the minimum log level (debug, info, warn or error)")

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix(strings.ToUpper(cmd.Use))

	if err := bindFlags(v, cmd); err != nil {
		log.Fatal(err)
	}

	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func bindFlags(v *viper.Viper, root *cobra.Command) error {
	if err := bindCommandFlags(v, root); err != nil {
		return err
	}

	for _, command := range root.Commands() {
		if len(command.Commands()) > 0 {
			if err := bindFlags(v, command); err != nil {
				return err
			}
		}

		if err := bindCommandFlags(v, command); err != nil {
			return err
		}
	}

	return nil
}

func bindCommandFlags(v *viper.Viper, command *cobra.Command) error {
	var err error
	command.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if err != nil {
			return
		}

		configName := f.Name
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			err = command.PersistentFlags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	return err
}
