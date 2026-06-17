package scenario

const (
	commandUseScenario   = "scenario"
	commandShortScenario = "generate and run contract-derived scenario checks"
	defaultDir           = "."
)

const (
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagRunHTTP    = "run-http"
	flagBaseURL    = "base-url"
	flagCases      = "cases"
	flagDraftCases = "draft-cases"
)

const (
	flagHelpJSON       = "emit machine-readable scenario output"
	flagHelpPretty     = "pretty-print JSON output"
	flagHelpRunHTTP    = "execute generated HTTP success scenarios and emit evidence"
	flagHelpBaseURL    = "base URL for HTTP scenario execution"
	flagHelpCases      = "execute explicit HTTP scenario cases JSON"
	flagHelpDraftCases = "emit executable HTTP case drafts derived from scenario suggestions"
)

const (
	jsonIndentPrefix = ""
	jsonIndentValue  = "  "
)

const (
	resultKindScenarioPlan   = "nucleus.scenario_plan_result"
	resultKindHTTPCaseDrafts = "nucleus.http_case_drafts_result"
	planKind                 = "nucleus.scenario_plan"
	httpCaseDraftsKind       = "nucleus.http_case_drafts"
	scenarioSchemaVersion    = "scenario.v1"
)
