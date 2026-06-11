package describe

const defaultSchemaVersion = "1.1"

const (
	commandUseDescribe  = "describe"
	commandShortSummary = "describe service metadata as JSON"
	defaultDir          = "."
	jsonIndentPrefix    = ""
	jsonIndentValue     = "  "
)

const (
	flagJSON   = "json"
	flagPretty = "pretty"
	flagFlow   = "flow"
)

const (
	flagHelpJSON   = "emit JSON"
	flagHelpPretty = "pretty-print JSON"
	flagHelpFlow   = "include conservative flow graph"
)

const (
	outputFieldSchemaVersion = "schema_version"
	outputFieldVerification  = "verification"
)

const (
	verificationFieldCommands       = "commands"
	verificationFieldPipeline       = "pipeline"
	verificationFieldResultKind     = "result_kind"
	verificationFieldEvidenceSchema = "evidence_schema"
)

const (
	verificationResultKind     = "nucleus.verify_result"
	verificationEvidenceSchema = "contract/schema/evidence.schema.json"
)

const (
	pipelineFieldID        = "id"
	pipelineFieldSequence  = "sequence"
	pipelineFieldPhase     = "phase"
	pipelineFieldCommand   = "command"
	pipelineFieldSchemaRef = "schema_ref"
	pipelineFieldProduces  = "produces"
)

const (
	commandValidate          = "nucleus validate --dir ."
	commandLintStrict        = "nucleus lint --dir . --strict"
	commandVerifyJSON        = "nucleus verify --dir . --json"
	commandDescribeJSON      = "nucleus describe --dir . --json"
	commandGoModTidy         = "go mod tidy"
	commandGoListAll         = "go list ./..."
	commandGoTestCompileOnly = "go test ./... -run ^$"
	commandGoTestAll         = "go test ./..."
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
