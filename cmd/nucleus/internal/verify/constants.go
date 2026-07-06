package verify

const (
	commandUseVerify   = "verify"
	commandShortVerify = "run Nucleus verification checks"
	defaultDir         = "."
)

const (
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagHelpJSON   = "emit machine-readable verification result"
	flagHelpPretty = "pretty-print JSON output"
)

const (
	resultKindVerify = "nucleus.verify_result"
	schemaVersion    = "evidence.v1"
	schemaRef        = "contract/schema/evidence.v1.schema.json"
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)

const (
	phaseValidate           = "validate"
	phaseLint               = "lint"
	phaseDecision           = "decision"
	phaseGeneratedFreshness = "generated_freshness"
	phaseVerifyCommand      = "verify_command"
)

const (
	commandValidate             = "nucleus validate --dir ."
	commandLintStrict           = "nucleus lint --dir . --strict"
	commandDecisionValidate     = "nucleus decision validate --dir ."
	commandGeneratedFreshness   = "nucleus describe --dir . --json"
	commandWorkingDir           = "."
	statusPassed                = "passed"
	statusFailed                = "failed"
	redactedValue               = "[REDACTED]"
	truncatedOutputNotice       = "[output truncated]"
	maxCommandOutputRunes       = 32768
	projectVerifyTimeoutSeconds = 120
)
