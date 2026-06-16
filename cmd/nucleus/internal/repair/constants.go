package repair

const (
	commandUseRepair   = "repair"
	commandShortRepair = "inspect failed evidence and report safe repair status"
	defaultDir         = "."
)

const (
	flagFromEvidence = "from-evidence"
	flagMaxRounds    = "max-rounds"
	flagJSON         = "json"
	flagPretty       = "pretty"
)

const (
	flagHelpFromEvidence = "failed evidence JSON file"
	flagHelpMaxRounds    = "maximum repair rounds"
	flagHelpJSON         = "emit machine-readable repair evidence"
	flagHelpPretty       = "pretty-print JSON output"
)

const (
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)
