package plan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the plan command.
type Config struct {
	Dir *string
}

type options struct {
	task       string
	json       bool
	pretty     bool
	executable bool
}

// ErrPlanBlocked is returned when the planned edits exceed allowed edit surfaces.
var ErrPlanBlocked = errors.New("plan blocked by edit surfaces")

// NewCommand creates the plan subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUsePlan,
		Short:         commandShortSummary,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			task := strings.TrimSpace(opts.task)
			if task == "" {
				return fmt.Errorf("--%s is required", flagTask)
			}
			output, err := BuildOutput(OutputOptions{
				Dir:        stringValue(config.Dir, defaultDir),
				Task:       task,
				Executable: opts.executable,
			})
			if err != nil {
				return err
			}
			if opts.json || opts.executable {
				if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), output)
			}
			if blocked := blockedEditsCount(output); blocked > 0 {
				return fmt.Errorf("%w: %d blocked edit(s)", ErrPlanBlocked, blocked)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.task, flagTask, "", flagHelpTask)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	cmd.Flags().BoolVar(&opts.executable, flagExecutable, false, flagHelpExecutable)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
