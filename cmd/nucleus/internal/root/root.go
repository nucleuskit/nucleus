package root

import (
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/apply"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capability"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/describe"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/execute"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/gen"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/initcmd"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/lint"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/migrate"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/plan"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/repair"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/report"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/scenario"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/serve"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/validate"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/verify"
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
	cmd.AddCommand(lint.NewCommand(lint.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(gen.NewCommand(gen.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(verify.NewCommand(verify.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(plan.NewCommand(plan.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(apply.NewCommand(apply.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(execute.NewCommand(execute.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(repair.NewCommand(repair.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(report.NewCommand(report.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(scenario.NewCommand(scenario.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(initcmd.NewCommand(initcmd.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(capability.NewCommand(capability.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(migrate.NewCommand(migrate.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(serve.NewCommand(serve.Config{
		Dir: &opts.dir,
	}))
	return cmd
}
