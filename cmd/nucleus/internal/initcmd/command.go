package initcmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the init command.
type Config struct {
	Dir *string
}

type options struct {
	name     string
	module   string
	template string
	agent    string
	json     bool
	human    bool
	pretty   bool
}

// ErrInitFailed is returned when initialization cannot produce a valid project.
var ErrInitFailed = errors.New("init failed")

// NewCommand creates the init subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{template: defaultTemplate}
	cmd := &cobra.Command{
		Use:           commandUseInit,
		Short:         commandShortInit,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config, opts)
			if opts.human && err != nil {
				renderError(cmd.ErrOrStderr(), err)
				return err
			}
			if opts.human {
				renderHuman(cmd.OutOrStdout(), result)
				return err
			}
			if renderErr := renderJSON(cmd.OutOrStdout(), result, opts.pretty); renderErr != nil {
				return renderErr
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.name, flagName, "", flagHelpName)
	cmd.Flags().StringVar(&opts.module, flagModule, "", flagHelpModule)
	cmd.Flags().StringVar(&opts.template, flagTemplate, defaultTemplate, flagHelpTemplate)
	cmd.Flags().StringVar(&opts.agent, flagAgent, "", flagHelpAgent)
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.human, flagHuman, false, flagHelpHuman)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
