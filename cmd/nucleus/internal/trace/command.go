package trace

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the trace command.
type Config struct {
	Dir *string
}

type options struct {
	json   bool
	pretty bool
}

var ErrTraceFailed = errors.New("trace failed")

// NewCommand creates the trace command group.
func NewCommand(config Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandUseTrace,
		Short:         commandShortTrace,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newSymbolCommand(config))
	cmd.AddCommand(newRouteCommand(config))
	cmd.AddCommand(newCapabilityCommand(config))
	return cmd
}

func newSymbolCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseSymbol,
		Short:         commandShortSymbol,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := traceSymbol(config, args[0])
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrTraceFailed
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}

func newRouteCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseRoute,
		Short:         commandShortRoute,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := traceRoute(config, args[0])
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrTraceFailed
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}

func newCapabilityCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseCapability,
		Short:         commandShortCapability,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := traceCapability(config, args[0])
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrTraceFailed
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
