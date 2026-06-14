package lint

import (
	"errors"
	"fmt"

	"github.com/nucleuskit/contract/lint"
	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the lint command.
type Config struct {
	Dir *string
}

type options struct {
	json   bool
	pretty bool
	strict bool
}

// ErrLintFailed is returned when lint findings are present.
var ErrLintFailed = errors.New("lint failed")

// NewCommand creates the lint subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseLint,
		Short:         commandShortLint,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := stringValue(config.Dir, defaultDir)
			findings := lint.Run(dir, opts.strict)
			summary := buildSummary(opts.strict, findings)
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), findings, summary, opts.strict, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), findings, summary)
			}
			if len(findings) > 0 {
				return fmt.Errorf("%w: %d finding(s)", ErrLintFailed, len(findings))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	cmd.Flags().BoolVar(&opts.strict, flagStrict, false, flagHelpStrict)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
