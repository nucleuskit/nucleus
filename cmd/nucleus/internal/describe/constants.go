package describe

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
	resultKindDescribe    = "nucleus.describe_result"
	schemaVersionDescribe = "describe-result.v1"
	schemaRefDescribe     = "contract/schema/describe-result.v1.schema.json"
)

const (
	outputFieldSchemaVersion = "schema_version"
	outputFieldResultKind    = "result_kind"
	outputFieldSchemaRef     = "schema_ref"
	outputFieldOK            = "ok"
	outputFieldVerification  = "verification"
	outputFieldDiagnostics   = "diagnostics"
)

const (
	verificationFieldCommands       = "commands"
	verificationFieldPipeline       = "pipeline"
	verificationFieldProjectSource  = "project_commands_source"
	verificationFieldResultKind     = "result_kind"
	verificationFieldEvidenceSchema = "evidence_schema"
	verificationFieldOptional       = "optional_evidence"
)

const (
	verificationResultKind           = "nucleus.verify_result"
	verificationEvidenceSchema       = "contract/schema/evidence.v1.schema.json"
	verificationProjectCommandSource = "nucleus.yaml verify.commands"
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
	commandValidate         = "nucleus validate --dir ."
	commandLintStrict       = "nucleus lint --dir . --strict"
	commandDecisionValidate = "nucleus decision validate --dir ."
	commandVerifyJSON       = "nucleus verify --dir . --json"
	commandDescribeJSON     = "nucleus describe --dir . --json"
	commandScenarioPlanJSON = "nucleus scenario --dir . --json"
	commandScenarioRunJSON  = "nucleus scenario --dir . --run-http --base-url <base-url> --json"
)

const (
	phaseValidate           = "validate"
	phaseLint               = "lint"
	phaseDecision           = "decision"
	phaseGeneratedFreshness = "generated_freshness"
	phaseScenario           = "scenario"
)

const (
	evidenceKindHTTPScenario = "nucleus.http_scenario_evidence"
)
