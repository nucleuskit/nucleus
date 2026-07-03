package trace

const (
	commandUseTrace        = "trace"
	commandShortTrace      = "query symbol and route chains"
	commandUseSymbol       = "symbol <symbol>"
	commandShortSymbol     = "trace callers and callees for a symbol"
	commandUseRoute        = "route <method path>"
	commandShortRoute      = "trace a route through the flow graph"
	commandUseCapability   = "capability <id>"
	commandShortCapability = "trace callers and callees for capability anchor symbols"
	defaultDir             = "."
	resultKindTrace        = "nucleus.trace_result"
	schemaVersionTrace     = "trace-result.v1"
	schemaRefTrace         = "contract/schema/trace-result.v1.schema.json"
	jsonIndentPrefix       = ""
	jsonIndentValue        = "  "
	routeNodeKind          = "route"
	edgeKindCalls          = "calls"
	flowEdgeKindDispatch   = "dispatch"
)

const (
	flagJSON   = "json"
	flagPretty = "pretty"
)

const (
	flagHelpJSON   = "emit machine-readable trace result"
	flagHelpPretty = "pretty-print JSON output"
)
