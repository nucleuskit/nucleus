package impact

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the impact command.
type Config struct {
	Dir *string
}

type options struct {
	json   bool
	pretty bool
}

var ErrImpactFailed = errors.New("impact failed")

// NewCommand creates the impact command group.
func NewCommand(config Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandUseImpact,
		Short:         commandShortImpact,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newSymbolCommand(config))
	cmd.AddCommand(newFileCommand(config))
	cmd.AddCommand(newContractCommand(config))
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
			output := impactSymbol(config, args[0])
			return renderAndReturn(cmd, output, opts)
		},
	}
	addFlags(cmd, opts)
	return cmd
}

func newFileCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseFile,
		Short:         commandShortFile,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := impactFile(config, args[0])
			return renderAndReturn(cmd, output, opts)
		},
	}
	addFlags(cmd, opts)
	return cmd
}

func newContractCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseContract,
		Short:         commandShortContract,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := impactContract(config, args[0])
			return renderAndReturn(cmd, output, opts)
		},
	}
	addFlags(cmd, opts)
	return cmd
}

func addFlags(cmd *cobra.Command, opts *options) {
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
}

func renderAndReturn(cmd *cobra.Command, output result, opts *options) error {
	if opts.json {
		if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
			return err
		}
	} else {
		renderHuman(cmd.OutOrStdout(), output)
	}
	if !output.OK {
		return ErrImpactFailed
	}
	return nil
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
