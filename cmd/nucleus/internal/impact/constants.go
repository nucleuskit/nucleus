package impact

const (
	commandUseImpact     = "impact"
	commandShortImpact   = "query change impact over the symbol graph"
	commandUseSymbol     = "symbol <symbol>"
	commandShortSymbol   = "show impact for a symbol"
	commandUseFile       = "file <path>"
	commandShortFile     = "show impact for a file"
	commandUseContract   = "contract <path>"
	commandShortContract = "show impact for a contract file"
	defaultDir           = "."
	resultKindImpact     = "nucleus.impact_result"
	schemaVersionImpact  = "impact-result.v1"
	schemaRefImpact      = "contract/schema/impact-result.v1.schema.json"
	jsonIndentPrefix     = ""
	jsonIndentValue      = "  "
	routeNodeKind        = "route"
)

const (
	flagJSON   = "json"
	flagPretty = "pretty"
)

const (
	flagHelpJSON   = "emit machine-readable impact result"
	flagHelpPretty = "pretty-print JSON output"
)
