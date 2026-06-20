package serve

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the serve command.
type Config struct {
	Dir *string
}

type options struct {
	addr          string
	allowNonLocal bool
	check         bool
	json          bool
	pretty        bool
	mode          string
}

// ErrServeFailed is returned when local metadata serving cannot start safely.
var ErrServeFailed = errors.New("serve failed")

// NewCommand creates the serve subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{addr: defaultAddr}
	cmd := &cobra.Command{
		Use:           commandUseServe,
		Short:         commandShortServe,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result := buildResult(config, opts)
			var listener listener
			if !opts.check && result.OK {
				opened, err := listen(result.Summary.Addr)
				if err != nil {
					result.Diagnostics = append(result.Diagnostics, errorDiagnostic("", diagnosticListenFailed, fmt.Sprintf("listen on %s: %v", result.Summary.Addr, err)))
					result = finalizeResult(result)
				} else {
					listener = opened
					result.Summary.Addr = listener.Addr().String()
					result.Server.Listening = true
				}
			}
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), result, opts.pretty); err != nil {
					closeListener(listener)
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), cmd.ErrOrStderr(), result)
			}
			if !result.OK {
				closeListener(listener)
				return fmt.Errorf("%w: serve diagnostics contain errors", ErrServeFailed)
			}
			if opts.check {
				return nil
			}
			return serveListener(cmd.Context(), listener, newHandler(result.Description))
		},
	}
	cmd.Flags().StringVar(&opts.addr, flagAddr, defaultAddr, flagHelpAddr)
	cmd.Flags().BoolVar(&opts.allowNonLocal, flagAllowNonLocal, false, flagHelpAllowNonLocal)
	cmd.Flags().BoolVar(&opts.check, flagCheck, false, flagHelpCheck)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}
