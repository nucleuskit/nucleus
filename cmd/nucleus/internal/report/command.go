package report

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the report command.
type Config struct {
	Dir *string
}

type options struct {
	aiTasksPath string
	json        bool
	pretty      bool
}

// ErrReportFailed is returned when report inputs cannot be loaded safely.
var ErrReportFailed = errors.New("report failed")

// NewCommand creates the report subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseReport,
		Short:         commandShortReport,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config, opts)
			if err != nil {
				return err
			}
			if opts.json {
				if renderErr := renderJSON(cmd.OutOrStdout(), result, opts.pretty); renderErr != nil {
					return renderErr
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), result)
			}
			if !result.OK {
				return fmt.Errorf("%w: report diagnostics contain errors", ErrReportFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.aiTasksPath, flagAITasks, "", flagHelpAITasks)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}
