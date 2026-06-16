package apply

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the apply command.
type Config struct {
	Dir *string
}

type options struct {
	planPath string
	dryRun   bool
	json     bool
	pretty   bool
}

// ErrApplyFailed is returned when apply evidence does not pass.
var ErrApplyFailed = errors.New("apply failed")

// NewCommand creates to apply subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{dryRun: true}
	cmd := &cobra.Command{
		Use:           commandUseApply,
		Short:         commandShortApply,
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
				return fmt.Errorf("%w: evidence did not pass", ErrApplyFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.planPath, flagPlan, "", flagHelpPlan)
	cmd.Flags().BoolVar(&opts.dryRun, flagDryRun, true, flagHelpDryRun)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}
