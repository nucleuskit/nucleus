package validate

const (
	commandUseValidate  = "validate"
	commandShortSummary = "validate manifest and contract files"
	defaultDir          = "."
)

const (
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagHelpJSON   = "emit machine-readable validation result"
	flagHelpPretty = "pretty-print JSON output"
)

const (
	resultKindValidate    = "nucleus.validate_result"
	schemaVersionValidate = "validate-result.v1"
	schemaRefValidate     = "contract/schema/validate-result.v1.schema.json"
	jsonIndentPrefix      = ""
	jsonIndentValue       = "  "
)
