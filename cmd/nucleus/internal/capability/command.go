package capability

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the capability command.
type Config struct {
	Dir *string
}

type options struct {
	provider string
	dryRun   bool
	force    bool
	json     bool
	pretty   bool
}

// ErrCapabilityFailed is returned when a capability scaffold cannot be applied safely.
var ErrCapabilityFailed = errors.New("capability failed")

// NewCommand creates the capability subcommand group.
func NewCommand(config Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandUseCapability,
		Short:         commandShortCapability,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
	}
	cmd.AddCommand(newAddCommand(config))
	return cmd
}

func newAddCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseCapabilityAdd,
		Short:         commandShortCapabilityAdd,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("capability add requires exactly one capability")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config, opts, args[0])
			if opts.json {
				if renderErr := renderJSON(cmd.OutOrStdout(), result, opts.pretty); renderErr != nil {
					return renderErr
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), result)
			}
			if err != nil {
				return err
			}
			if !result.OK {
				return fmt.Errorf("%w: capability diagnostics contain errors", ErrCapabilityFailed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.provider, flagProvider, "", flagHelpProvider)
	cmd.Flags().BoolVar(&opts.dryRun, flagDryRun, false, flagHelpDryRun)
	cmd.Flags().BoolVar(&opts.force, flagForce, false, flagHelpForce)
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
