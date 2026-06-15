package root

import (
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/describe"
	nucleuslint "github.com/nucleuskit/nucleus/cmd/nucleus/internal/lint"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/plan"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/validate"
	"github.com/spf13/cobra"
)

// options holds the command line options for the root command.
type options struct {
	dir     string // service root directory
	verbose bool   // verbose output
	schema  string // schema version override
}

// New creates the root nucleus command and wires top-level subcommands.
func New() *cobra.Command {
	opts := &options{dir: "."}
	cmd := &cobra.Command{
		Use:           "nucleus",
		Short:         "Nucleus service kernel CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&opts.dir, "dir", ".", "service root directory")
	cmd.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "enable verbose output")
	cmd.PersistentFlags().StringVar(&opts.schema, "schema", "", "override describe schema version")

	cmd.AddCommand(describe.NewCommand(describe.Config{
		Dir:            &opts.dir,
		SchemaOverride: &opts.schema,
	}))
	cmd.AddCommand(validate.NewCommand(validate.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(nucleuslint.NewCommand(nucleuslint.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(plan.NewCommand(plan.Config{
		Dir: &opts.dir,
	}))
	return cmd
}
