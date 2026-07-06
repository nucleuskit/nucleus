package decision

import (
	"errors"

	"github.com/spf13/cobra"
)

// Config carries root-level flag values used by the decision command.
type Config struct {
	Dir *string
}

type options struct {
	json       bool
	pretty     bool
	acceptedBy string
	acceptedAt string
}

var ErrDecisionInvalid = errors.New("decision validation failed")

// NewCommand creates the decision command group.
func NewCommand(config Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandUseDecision,
		Short:         commandShortDecision,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newValidateCommand(config))
	cmd.AddCommand(newAcceptCommand(config))
	cmd.AddCommand(newSupersedeCommand(config))
	return cmd
}

func newValidateCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseValidate,
		Short:         commandShortValidate,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			output := validate(config, args)
			if opts.json {
				if err := renderJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrDecisionInvalid
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	return cmd
}

func newAcceptCommand(config Config) *cobra.Command {
	opts := &options{acceptedBy: "human"}
	cmd := &cobra.Command{
		Use:           commandUseAccept,
		Short:         commandShortAccept,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := accept(config, args[0], opts.acceptedBy, opts.acceptedAt)
			if opts.json {
				if err := renderActionJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderActionHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrDecisionInvalid
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.json, flagJSON, false, flagHelpJSON)
	cmd.Flags().BoolVar(&opts.pretty, flagPretty, false, flagHelpPretty)
	cmd.Flags().StringVar(&opts.acceptedBy, flagAcceptedBy, "human", flagHelpAcceptedBy)
	cmd.Flags().StringVar(&opts.acceptedAt, flagAcceptedAt, "", flagHelpAcceptedAt)
	return cmd
}

func newSupersedeCommand(config Config) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           commandUseSupersede,
		Short:         commandShortSupersede,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := supersede(config, args[0])
			if opts.json {
				if err := renderActionJSON(cmd.OutOrStdout(), output, opts.pretty); err != nil {
					return err
				}
			} else {
				renderActionHuman(cmd.OutOrStdout(), output)
			}
			if !output.OK {
				return ErrDecisionInvalid
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
