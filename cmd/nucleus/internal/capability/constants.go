package capability

const (
	commandUseCapability      = "capability"
	commandUseCapabilityAdd   = "add <capability>"
	commandShortCapability    = "manage service capability scaffolds"
	commandShortCapabilityAdd = "add a capability declaration and service-side scaffold"
	defaultDir                = "."
)

const (
	flagProvider = "provider"
	flagDryRun   = "dry-run"
	flagForce    = "force"
	flagJSON     = "json"
	flagPretty   = "pretty"
)

const (
	flagHelpProvider = "capability provider scaffold"
	flagHelpDryRun   = "preview changes without writing files"
	flagHelpForce    = "overwrite generated scaffold files when their content differs"
	flagHelpJSON     = "emit machine-readable capability result"
	flagHelpPretty   = "pretty-print JSON output"
)

const (
	resultKindCapability = "nucleus.capability_result"
	schemaVersion        = "capability.v1"
	schemaRefEvidence    = "contract/schema/evidence.schema.json"
)

const (
	actionCreated     = "created"
	actionUpdated     = "updated"
	actionUnchanged   = "unchanged"
	actionWouldCreate = "would_create"
	actionWouldUpdate = "would_update"
	actionConflict    = "conflict"
)

const (
	postgresDriverModule  = "github.com/lib/pq"
	postgresDriverVersion = "v1.10.9"
)
