package validate

import (
	"errors"

	"github.com/nucleuskit/contract/validation"
	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the validate command.
type Config struct {
	Dir *string
}

type options struct {
	pretty bool
	json   bool
}

var errValidationFailed = errors.New("validation failed")

// NewCommand creates the validate subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   commandUseValidate,
		Short: commandShortSummary,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := stringValue(config.Dir, defaultDir)
			diagnostics := validation.ValidateService(dir)
			summary := buildSummary(dir, diagnostics)
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), diagnostics, summary, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), diagnostics, summary)
			}
			if diagnostics.Failed() {
				return errValidationFailed
			}
			return nil
		},
	}
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
