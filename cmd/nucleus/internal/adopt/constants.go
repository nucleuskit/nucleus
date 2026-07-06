package adopt

const (
	commandUseAdopt      = "adopt"
	commandShortSummary  = "add a minimal Nucleus protocol index to an existing Go project"
	defaultDir           = "."
	defaultVersion       = "0.0.0"
	defaultIntent        = "Agent-readable protocol index for an existing Go service."
	resultKindAdopt      = "nucleus.adopt_result"
	schemaVersionAdopt   = "adopt-result.v1"
	schemaRefAdopt       = "contract/schema/adopt-result.v1.schema.json"
	manifestFileName     = "nucleus.yaml"
	decisionsKeepFile    = ".nucleus/decisions/.gitkeep"
	nucleusReadmeFile    = ".nucleus/README.md"
	codexInstructionFile = ".nucleus/agents/codex.md"
	agentCodex           = "codex"
)

const (
	flagService = "service"
	flagVersion = "version"
	flagIntent  = "intent"
	flagAgent   = "agent"
	flagForce   = "force"
	flagJSON    = "json"
	flagPretty  = "pretty"
)

const (
	flagHelpService = "service name to write into nucleus.yaml; inferred from go.mod or directory when omitted"
	flagHelpVersion = "service version to write into nucleus.yaml"
	flagHelpIntent  = "AI intent to write into nucleus.yaml"
	flagHelpAgent   = "optional agent instruction pack to write under .nucleus/agents; supported: codex"
	flagHelpForce   = "overwrite nucleus.yaml when it already exists"
	flagHelpJSON    = "emit JSON output"
	flagHelpPretty  = "pretty-print JSON output"
)
