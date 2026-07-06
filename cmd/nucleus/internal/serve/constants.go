package serve

const (
	commandUseServe   = "serve"
	commandShortServe = "run local Nucleus metadata endpoints"
	defaultDir        = "."
	defaultAddr       = "127.0.0.1:8080"
)

const (
	flagAddr          = "addr"
	flagAllowNonLocal = "allow-non-local"
	flagCheck         = "check"
	flagJSON          = "json"
	flagPretty        = "pretty"
)

const (
	flagHelpAddr          = "local metadata serve address"
	flagHelpAllowNonLocal = "allow serving metadata on a non-loopback address"
	flagHelpCheck         = "inspect manifest and contracts without listening"
	flagHelpJSON          = "emit machine-readable serve result"
	flagHelpPretty        = "pretty-print JSON output"
)

const (
	resultKindServe    = "nucleus.serve_result"
	schemaVersionServe = "serve-result.v1"
	schemaRefServe     = "contract/schema/serve-result.v1.schema.json"
	jsonIndentPrefix   = ""
	jsonIndentValue    = "  "
)

const (
	modeServe = "serve"
	modeCheck = "check"
)

const (
	pathHealthz   = "/healthz"
	pathReadyz    = "/readyz"
	pathWellKnown = "/.well-known/nucleus.json"
)

const (
	diagnosticInspectFailed = "serve.inspect_failed"
	diagnosticAddrInvalid   = "serve.addr_invalid"
	diagnosticNonLocalAddr  = "serve.non_local_addr"
	diagnosticListenFailed  = "serve.listen_failed"
)

const (
	networkScopeLoopback = "loopback"
	networkScopeNonLocal = "non_local"
	networkScopeInvalid  = "invalid"
)
