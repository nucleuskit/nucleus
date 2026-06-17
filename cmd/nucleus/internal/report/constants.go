package report

const (
	commandUseReport   = "report"
	commandShortReport = "summarize AI change quality and release readiness"
	defaultDir         = "."
	defaultAITasksDir  = "artifacts/nucleus/ai-tasks"
)

const (
	flagAITasks  = "ai-tasks"
	flagPlatform = "platform"
	flagJSON     = "json"
	flagPretty   = "pretty"
)

const (
	flagHelpAITasks  = "directory containing AI task result JSON files"
	flagHelpPlatform = "emit platform and release readiness metadata"
	flagHelpJSON     = "emit machine-readable report result"
	flagHelpPretty   = "pretty-print JSON output"
)

const (
	reportModeAIQuality         = "ai_quality"
	reportModePlatformReadiness = "platform_readiness"
)

const (
	resultKindReport    = "nucleus.report_result"
	schemaVersionReport = "report.v1"
	schemaRefReport     = "contract/schema/report.schema.json"
	jsonIndentPrefix    = ""
	jsonIndentValue     = "  "
)

const (
	reportDiagnosticAITasksMissing    = "report.ai_tasks_missing"
	reportDiagnosticAITasksReadFailed = "report.ai_tasks_read_failed"
	reportDiagnosticAITaskParseFailed = "report.ai_task_parse_failed"
	reportDiagnosticInspectFailed     = "report.inspect_failed"
)

const (
	inputStatusLoaded  = "loaded"
	inputStatusMissing = "missing"
	inputStatusFailed  = "failed"
)
