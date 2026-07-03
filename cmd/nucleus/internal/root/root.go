package root

import (
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/adopt"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/apply"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/describe"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/execute"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/gen"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/impact"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/lint"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/mark"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/mcp"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/plan"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/repair"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/report"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/scenario"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/serve"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/trace"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/validate"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/verify"
	"github.com/spf13/cobra"
)

// options holds the command line options for the root command.
type options struct {
	dir     string // service root directory
	verbose bool   // verbose output
}

// New creates the root nucleus command and wires top-level subcommands.
func New() *cobra.Command {
	opts := &options{dir: "."}
	cmd := &cobra.Command{
		Use:           "nucleus",
		Short:         "Nucleus agent-native protocol CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&opts.dir, "dir", ".", "service root directory")
	cmd.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "enable verbose output")
	cmd.AddCommand(adopt.NewCommand(adopt.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(decision.NewCommand(decision.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(describe.NewCommand(describe.Config{
		Dir: &opts.dir,
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
	cmd.AddCommand(trace.NewCommand(trace.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(impact.NewCommand(impact.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(mark.NewCommand(mark.Config{
		Dir: &opts.dir,
	}))
	cmd.AddCommand(mcp.NewCommand(mcp.Config{
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
	cmd.AddCommand(serve.NewCommand(serve.Config{
		Dir: &opts.dir,
	}))
	return cmd
}
