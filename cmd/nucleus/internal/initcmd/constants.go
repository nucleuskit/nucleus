package initcmd

const (
	commandUseInit   = "init"
	commandShortInit = "initialize a Nucleus service, worker, or library template"

	defaultDir      = "."
	defaultTemplate = "service"

	flagName     = "name"
	flagModule   = "module"
	flagTemplate = "template"
	flagAgent    = "agent"
	flagJSON     = "json"
	flagHuman    = "human"
	flagPretty   = "pretty"

	flagHelpName     = "service name"
	flagHelpModule   = "Go module path"
	flagHelpTemplate = "template type: service, worker, or library"
	flagHelpAgent    = "agent pack to generate: codex"
	flagHelpJSON     = "emit JSON output (default; kept for compatibility)"
	flagHelpHuman    = "emit human-readable output"
	flagHelpPretty   = "pretty-print JSON output"

	templateService = "service"
	templateWorker  = "worker"
	templateLibrary = "library"

	agentCodex = "codex"

	resultKindInit = "nucleus.init_result"

	nucleusHTTPModule      = "github.com/nucleuskit/http"
	nucleusHTTPVersion     = "v0.1.0-alpha.2"
	nucleusCapModule       = "github.com/nucleuskit/cap"
	nucleusCapVersion      = "v0.1.0-alpha.2"
	nucleusCoreModule      = "github.com/nucleuskit/core"
	nucleusCoreVersion     = "v0.1.0-alpha.2"
	contractGenTarget      = "contract/gen"
	httpAdapterGenTarget   = "internal/adapter/http/gen"
	generatedFreshnessFile = ".nucleus-source.sha256"
	defaultServiceVersion  = "0.1.0"
	defaultGoVersion       = "1.26.3"
	defaultHTTPAddress     = "127.0.0.1:8080"
)
