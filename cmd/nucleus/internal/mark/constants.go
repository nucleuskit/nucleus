package mark

const (
	commandUseMark         = "mark"
	commandShortMark       = "declare protocol anchors in nucleus.yaml"
	commandUseContract     = "contract <id>"
	commandShortContract   = "declare a contract file"
	commandUseCapability   = "capability <id>"
	commandShortCapability = "declare a capability anchor"
	commandUseVerify       = "verify <command>"
	commandShortVerify     = "declare a project-owned verification command"
	defaultDir             = "."
	resultKindMark         = "nucleus.mark_result"
	schemaVersionMark      = "mark-result.v1"
	schemaRefMark          = "contract/schema/mark-result.v1.schema.json"
	manifestFileName       = "nucleus.yaml"
	jsonIndentPrefix       = ""
	jsonIndentValue        = "  "
	statusResolved         = "resolved"
	statusDeclared         = "declared"
)

const (
	flagKind   = "kind"
	flagPath   = "path"
	flagSymbol = "symbol"
	flagIntent = "intent"
	flagJSON   = "json"
	flagPretty = "pretty"
)

const (
	flagHelpKind   = "semantic kind to record in nucleus.yaml"
	flagHelpPath   = "relative contract path to record in nucleus.yaml"
	flagHelpSymbol = "capability anchor symbol; may be repeated"
	flagHelpIntent = "optional capability intent"
	flagHelpJSON   = "emit machine-readable mark result"
	flagHelpPretty = "pretty-print JSON output"
)
