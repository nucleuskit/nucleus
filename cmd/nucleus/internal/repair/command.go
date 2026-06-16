package repair

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the repair command.
type Config struct {
	Dir *string
}

type options struct {
	evidencePath string
	maxRounds    int
	json         bool
	pretty       bool
}

// ErrRepairFailed is returned when repair evidence does not pass.
var ErrRepairFailed = errors.New("repair failed")

// NewCommand creates the repair subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{maxRounds: 1}
	cmd := &cobra.Command{
		Use:           commandUseRepair,
		Short:         commandShortRepair,
		SilenceUsage:  true,
		SilenceErrors: true,
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
			if !evidencePass(result) {
				return fmt.Errorf("%w: evidence did not pass", ErrRepairFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.evidencePath, flagFromEvidence, "", flagHelpFromEvidence)
	cmd.Flags().IntVar(&opts.maxRounds, flagMaxRounds, 1, flagHelpMaxRounds)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}
