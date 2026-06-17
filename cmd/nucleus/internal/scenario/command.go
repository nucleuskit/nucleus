package scenario

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the scenario command.
type Config struct {
	Dir *string
}

type options struct {
	json       bool
	pretty     bool
	runHTTP    bool
	baseURL    string
	casesPath  string
	draftCases bool
}

// ErrScenarioFailed is returned when scenario evidence does not pass.
var ErrScenarioFailed = errors.New("scenario failed")

// NewCommand creates the scenario subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseScenario,
		Short:         commandShortScenario,
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
			if evidencePass(result) {
				return nil
			}
			if isEvidence(result) {
				return fmt.Errorf("%w: evidence did not pass", ErrScenarioFailed)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	cmd.Flags().BoolVar(&opts.runHTTP, flagRunHTTP, false, flagHelpRunHTTP)
	cmd.Flags().StringVar(&opts.baseURL, flagBaseURL, "", flagHelpBaseURL)
	cmd.Flags().StringVar(&opts.casesPath, flagCases, "", flagHelpCases)
	cmd.Flags().BoolVar(&opts.draftCases, flagDraftCases, false, flagHelpDraftCases)
	return cmd
}
