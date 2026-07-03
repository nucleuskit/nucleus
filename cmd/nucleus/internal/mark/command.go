package mark

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the mark command.
type Config struct {
	Dir *string
}

type options struct {
	kind    string
	path    string
	symbols []string
	intent  string
	json    bool
	pretty  bool
}

var ErrMarkFailed = errors.New("mark failed")

// NewCommand creates the mark command group.
func NewCommand(config Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandUseMark,
		Short:         commandShortMark,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newContractCommand(config))
	cmd.AddCommand(newCapabilityCommand(config))
	cmd.AddCommand(newVerifyCommand(config))
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
			output := markContract(config, args[0], opts)
			return renderAndReturn(cmd, output, opts)
		},
	}
	cmd.Flags().StringVar(&opts.kind, flagKind, "", flagHelpKind)
	cmd.Flags().StringVar(&opts.path, flagPath, "", flagHelpPath)
	addOutputFlags(cmd, opts)
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
			output := markCapability(config, args[0], opts)
			return renderAndReturn(cmd, output, opts)
		},
	}
	cmd.Flags().StringVar(&opts.kind, flagKind, "", flagHelpKind)
	cmd.Flags().StringArrayVar(&opts.symbols, flagSymbol, nil, flagHelpSymbol)
	cmd.Flags().StringVar(&opts.intent, flagIntent, "", flagHelpIntent)
	addOutputFlags(cmd, opts)
	return cmd
}

func newVerifyCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseVerify,
		Short:         commandShortVerify,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := markVerify(config, args[0])
			return renderAndReturn(cmd, output, opts)
		},
	}
	addOutputFlags(cmd, opts)
	return cmd
}

func addOutputFlags(cmd *cobra.Command, opts *options) {
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
		return ErrMarkFailed
	}
	return nil
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
