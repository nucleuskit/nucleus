package execute

const (
	commandUseExecute   = "execute"
	commandShortExecute = "execute allowlisted plan commands and emit evidence"
	defaultDir          = "."
)

const (
	flagPlan         = "plan"
	flagAllowCommand = "allow-command"
	flagJSON         = "json"
	flagPretty       = "pretty"
)

const (
	flagHelpPlan         = "executable plan JSON file"
	flagHelpAllowCommand = "command name allowed for execution; repeat for multiple commands"
	flagHelpJSON         = "emit machine-readable executor evidence"
	flagHelpPretty       = "pretty-print JSON output"
)

const (
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)

const (
	resultKindExecutorEvidence = "nucleus.executor_evidence"
	schemaVersionEvidence      = "evidence.v1"
	schemaRefEvidence          = "contract/schema/evidence.v1.schema.json"
)

const (
	statusPassed  = "passed"
	statusFailed  = "failed"
	statusBlocked = "blocked"
	statusTimeout = "timeout"
)
