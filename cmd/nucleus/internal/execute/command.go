package execute

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the execute command.
type Config struct {
	Dir *string
}

type options struct {
	planPath      string
	allowCommands []string
	json          bool
	pretty        bool
}

// ErrExecuteFailed is returned when executor evidence does not pass.
var ErrExecuteFailed = errors.New("execute failed")

// NewCommand creates to execute subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseExecute,
		Short:         commandShortExecute,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			evidence, err := run(config, opts)
			if err != nil {
				return err
			}
			if opts.json {
				if renderErr := renderJSON(cmd.OutOrStdout(), evidence, opts.pretty); renderErr != nil {
					return renderErr
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), evidence)
			}
			if !evidencePass(evidence) {
				return fmt.Errorf("%w: evidence did not pass", ErrExecuteFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.planPath, flagPlan, "", flagHelpPlan)
	cmd.Flags().StringArrayVar(&opts.allowCommands, flagAllowCommand, nil, flagHelpAllowCommand)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}
