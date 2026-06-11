package describe

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the describe command.
type Config struct {
	Dir            *string
	SchemaOverride *string
}

type options struct {
	pretty bool
	json   bool
	flow   bool
}

// NewCommand creates the describe subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "describe service metadata as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := BuildOutput(buildOptions(config, opts))
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			if opts.pretty {
				encoder.SetIndent("", "  ")
			}
			return encoder.Encode(output)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "emit JSON")
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "pretty-print JSON")
	cmd.Flags().BoolVar(&opts.flow, "flow", false, "include conservative flow graph")
	return cmd
}

// buildOptions builds an OutputOptions struct from the root command's flag values.
func buildOptions(config Config, opts *options) OutputOptions {
	return OutputOptions{
		Dir:            stringValue(config.Dir, "."),
		SchemaOverride: stringValue(config.SchemaOverride, ""),
		IncludeFlow:    opts.flow,
	}
}

// stringValue returns the string value of a pointer, or a fallback value if the pointer is nil.
func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
