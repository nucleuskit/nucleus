package apply

const (
	commandUseApply   = "apply"
	commandShortApply = "apply an executable Nucleus plan and emit evidence"
	defaultDir        = "."
)

const (
	flagPlan   = "plan"
	flagDryRun = "dry-run"
	flagJSON   = "json"
	flagPretty = "pretty"
)

const (
	flagHelpPlan   = "executable plan JSON file"
	flagHelpDryRun = "validate plan without writing files"
	flagHelpJSON   = "emit machine-readable apply evidence"
	flagHelpPretty = "pretty-print JSON output"
)

const (
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)
