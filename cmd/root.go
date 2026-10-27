// cmd/root.go

// Package cmd is qr2git's command tree.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/messages"
	"smegg.me/qr2g.it/core/datastore"
)

// say renders command text in the operator's language.
var say = messages.Localizer(messages.LanguageFromEnv())

// errUsage exits 2, so a script can tell a misuse from a failure.
var errUsage = errors.New("usage")

// Version is the release in VERSION, set by main.
var Version = "dev"

// Execute runs the command tree and returns the process exit code.
func Execute() int {
	root := newRootCmd()
	err := root.Execute()
	datastore.Close()
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errUsage):
		root.PrintErrln(err)
		return 2
	default:
		root.PrintErrln(err)
		return 1
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "qr2git",
		Short:         messages.CliRoot(say),
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return fmt.Errorf("%w: %v", errUsage, err) })
	root.AddCommand(newServeCmd(), newTokenCmd(), newModerateCmd())
	return root
}

// usage wraps a cobra.PositionalArgs so misuse exits 2.
func usage(check cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := check(cmd, args); err != nil {
			return fmt.Errorf("%w: %v", errUsage, err)
		}
		return nil
	}
}

// loadConfig reads the configuration; a generated default config ends the run.
func loadConfig() error {
	if _, err := config.Initialize(); err != nil {
		if e, ok := config.IsDefaultGenerated(err); ok {
			return errors.New(messages.CliDefaultConfig(say, messages.CliDefaultConfigParams{Path: e.Path}))
		}
		return err
	}
	return logger.ReInitialize(config.Get().Logger)
}

// openStore opens the database for an operator command, logging only warnings so output stays clean.
func openStore(ctx context.Context) error {
	logger.Initialize()
	if err := loadConfig(); err != nil {
		return err
	}
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	return datastore.Initialize(ctx)
}

var headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
var cellStyle = lipgloss.NewStyle().Padding(0, 1)

// printTable renders rows with the shared table style.
func printTable(w io.Writer, headers []string, rows [][]string) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})
	_, _ = fmt.Fprintln(w, t.Render())
}

func stdout(cmd *cobra.Command) io.Writer {
	if w := cmd.OutOrStdout(); w != nil {
		return w
	}
	return os.Stdout
}
