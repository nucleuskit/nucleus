package migrate

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the migrate command.
type Config struct {
	Dir *string
}

type options struct {
	fromVersion string
	toVersion   string
	check       bool
	reportPath  string
	json        bool
	pretty      bool
}

// ErrMigrateFailed is returned when migration planning or checks fail.
var ErrMigrateFailed = errors.New("migrate failed")

// NewCommand creates to migrate subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseMigrate,
		Short:         commandShortMigrate,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config, opts)
			if err != nil {
				return err
			}
			if opts.reportPath != "" {
				if err := writeReport(stringValue(config.Dir, defaultDir), opts.reportPath, result); err != nil {
					return err
				}
			}
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), result, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), result)
			}
			if !result.OK {
				return fmt.Errorf("%w: migration diagnostics contain errors", ErrMigrateFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.fromVersion, flagFromVersion, "", flagHelpFromVersion)
	cmd.Flags().StringVar(&opts.toVersion, flagToVersion, "", flagHelpToVersion)
	cmd.Flags().BoolVar(&opts.check, flagCheck, false, flagHelpCheck)
	cmd.Flags().StringVar(&opts.reportPath, flagReport, "", flagHelpReport)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
