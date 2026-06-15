package gen

const (
	commandUseGen   = "gen"
	commandShortGen = "generate Nucleus contract artifacts"
	defaultDir      = "."
)

const (
	flagJSON           = "json"
	flagPretty         = "pretty"
	flagHTTP           = "http"
	flagGRPC           = "grpc"
	flagErrors         = "errors"
	flagClients        = "clients"
	flagClientLanguage = "client-language"
	flagDocs           = "docs"
	flagTypeScript     = "typescript"
)

const (
	flagHelpJSON           = "emit machine-readable generation result"
	flagHelpPretty         = "pretty-print JSON output"
	flagHelpHTTP           = "generate HTTP endpoint and adapter metadata"
	flagHelpGRPC           = "generate gRPC service metadata"
	flagHelpErrors         = "generate error catalog metadata"
	flagHelpClients        = "generate client artifacts where supported"
	flagHelpClientLanguage = "client language for --clients; repeat for typescript, dart, java, kotlin"
	flagHelpDocs           = "generate contract markdown documentation"
	flagHelpTypeScript     = "generate TypeScript schema declarations"
)

const (
	resultKindGen    = "nucleus.gen_result"
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)
