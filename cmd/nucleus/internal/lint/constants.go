package lint

const (
	commandUseLint    = "lint"
	commandShortLint  = "run Nucleus architecture lint rules"
	defaultDir        = "."
	resultKindLint    = "nucleus.lint_result"
	schemaVersionLint = "lint-result.v1"
	schemaRefLint     = "contract/schema/lint-result.v1.schema.json"
	jsonIndentPrefix  = ""
	jsonIndentValue   = "  "
)

const (
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagStrict     = "strict"
	flagHelpJSON   = "emit machine-readable lint result"
	flagHelpPretty = "pretty-print JSON output"
	flagHelpStrict = "enable strict linting"
)
