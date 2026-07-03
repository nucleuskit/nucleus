package mcp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the mcp command.
type Config struct {
	Dir *string
}

type options struct {
	stdio bool
}

// NewCommand creates the mcp subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseMCP,
		Short:         commandShortMCP,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := NewServer(stringValue(config.Dir, defaultDir))
			if opts.stdio {
				return server.Serve(cmd.InOrStdin(), cmd.OutOrStdout())
			}
			return fmt.Errorf("--%s is required", flagStdio)
		},
	}
	cmd.Flags().BoolVar(&opts.stdio, flagStdio, false, flagHelpStdio)
	cmd.SetIn(os.Stdin)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
