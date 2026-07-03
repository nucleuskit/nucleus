package adopt

import "github.com/spf13/cobra"

// Config carries root-level flag values used by the adopt command.
type Config struct {
	Dir *string
}

type options struct {
	service string
	version string
	intent  string
	agent   string
	force   bool
	json    bool
	pretty  bool
}

// NewCommand creates the adopt subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{version: defaultVersion, intent: defaultIntent}
	cmd := &cobra.Command{
		Use:           commandUseAdopt,
		Short:         commandShortSummary,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := run(config, opts)
			if opts.json {
				if renderErr := renderJSON(cmd.OutOrStdout(), output, opts.pretty); renderErr != nil {
					return renderErr
				}
			} else {
				renderHuman(cmd.OutOrStdout(), output)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.service, flagService, "", flagHelpService)
	cmd.Flags().StringVar(&opts.version, flagVersion, defaultVersion, flagHelpVersion)
	cmd.Flags().StringVar(&opts.intent, flagIntent, defaultIntent, flagHelpIntent)
	cmd.Flags().StringVar(&opts.agent, flagAgent, "", flagHelpAgent)
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
