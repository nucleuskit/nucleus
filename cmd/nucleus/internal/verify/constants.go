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
	schemaVersion    = "verify.v1"
	schemaRef        = "contract/schema/evidence.schema.json"
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)

const (
	phaseValidate           = "validate"
	phaseLint               = "lint"
	phaseGeneratedFreshness = "generated_freshness"
	phaseTidy               = "tidy"
	phaseImport             = "import"
	phaseBuild              = "build"
	phaseTest               = "test"
)

const (
	commandValidate           = "nucleus validate --dir ."
	commandLintStrict         = "nucleus lint --dir . --strict"
	commandGeneratedFreshness = "nucleus describe --dir . --json"
	commandWorkingDir         = "."
	statusPassed              = "passed"
	statusFailed              = "failed"
)
