package gen

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the gen command.
type Config struct {
	Dir *string
}

type options struct {
	json            bool
	pretty          bool
	http            bool
	grpc            bool
	errors          bool
	clients         bool
	clientLanguages []string
	docs            bool
	typeScript      bool
}

// ErrGenFailed is returned when generation cannot produce valid artifacts.
var ErrGenFailed = errors.New("generation failed")

// NewCommand creates the gen subcommand.
func NewCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseGen,
		Short:         commandShortGen,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := run(config, opts)
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
	cmd.Flags().BoolVar(&opts.http, flagHTTP, false, flagHelpHTTP)
	cmd.Flags().BoolVar(&opts.grpc, flagGRPC, false, flagHelpGRPC)
	cmd.Flags().BoolVar(&opts.errors, flagErrors, false, flagHelpErrors)
	cmd.Flags().BoolVar(&opts.clients, flagClients, false, flagHelpClients)
	cmd.Flags().StringSliceVar(&opts.clientLanguages, flagClientLanguage, nil, flagHelpClientLanguage)
	cmd.Flags().BoolVar(&opts.docs, flagDocs, false, flagHelpDocs)
	cmd.Flags().BoolVar(&opts.typeScript, flagTypeScript, false, flagHelpTypeScript)
	return cmd
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
