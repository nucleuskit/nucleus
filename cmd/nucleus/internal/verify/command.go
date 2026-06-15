package verify

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the verify command.
type Config struct {
	Dir *string
}

type options struct {
	json   bool
	pretty bool
}

// ErrVerifyFailed is returned when one or more verification checks fail.
var ErrVerifyFailed = errors.New("verification failed")

// NewCommand creates to verify subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseVerify,
		Short:         commandShortVerify,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config)
			if opts.json {
				if renderErr := renderJSON(cmd.OutOrStdout(), result, opts.pretty); renderErr != nil {
					return renderErr
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), result)
			}
			return err
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
