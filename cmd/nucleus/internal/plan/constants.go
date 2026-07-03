package plan

const (
	commandUsePlan      = "plan"
	commandShortSummary = "suggest safe edit surfaces and verification commands for a task"
	defaultDir          = "."
)

const (
	flagTask       = "task"
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagExecutable = "executable"
)

const (
	flagHelpTask       = "natural language task"
	flagHelpJSON       = "emit machine-readable plan result"
	flagHelpPretty     = "pretty-print JSON output"
	flagHelpExecutable = "emit executable plan JSON"
)

const (
	resultKindPlan           = "nucleus.plan_result"
	resultKindExecutablePlan = "nucleus.executable_plan_result"
	planKind                 = "nucleus.plan"
	executablePlanKind       = "nucleus.executable_plan"
	schemaVersionPlan        = "plan-result.v1"
	schemaVersionExecutable  = "plan-executable.v1"
	schemaRefPlan            = "contract/schema/plan-result.v1.schema.json"
	schemaRefPlanExecutable  = "contract/schema/plan-executable.v1.schema.json"
	schemaRefEvidence        = "contract/schema/evidence.v1.schema.json"
	evidenceKindApply        = "nucleus.apply_evidence"
	evidenceKindExecutor     = "nucleus.executor_evidence"
	evidenceKindHTTPScenario = "nucleus.http_scenario_evidence"
	evidenceKindVerify       = "nucleus.verify_result"
	jsonIndentPrefix         = ""
	jsonIndentValue          = "  "
)

const (
	taskTypeCapability   = "capability"
	taskTypeErrorCatalog = "error_catalog"
	taskTypeGeneral      = "general"
	taskTypeGRPCService  = "grpc_service"
	taskTypeHTTPEndpoint = "http_endpoint"
)

const (
	commandValidate     = "nucleus validate --dir ."
	commandLintStrict   = "nucleus lint --dir . --strict"
	commandVerifyJSON   = "nucleus verify --dir . --json"
	commandDescribeFlow = "nucleus describe --dir . --json --flow"
)

const (
	commandPhaseGenerate = "generate"
	commandPhaseLint     = "lint"
	commandPhaseTest     = "test"
	commandPhaseValidate = "validate"
	commandPhaseVerify   = "verify"
)

const (
	commandProducesExit = "command_exit"
)

const decisionSchemaRef = "contract/schema/decision.v1.schema.json"
const recipeSchemaRef = "contract/schema/recipe.v1.schema.json"
