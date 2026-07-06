package report

const (
	commandUseReport   = "report"
	commandShortReport = "summarize local AI change quality"
	defaultDir         = "."
	defaultAITasksDir  = "artifacts/nucleus/ai-tasks"
)

const (
	flagAITasks = "ai-tasks"
	flagJSON    = "json"
	flagPretty  = "pretty"
)

const (
	flagHelpAITasks = "directory containing AI task result JSON files"
	flagHelpJSON    = "emit machine-readable report result"
	flagHelpPretty  = "pretty-print JSON output"
)

const (
	reportModeAIQuality = "ai_quality"
)

const (
	resultKindReport    = "nucleus.report_result"
	schemaVersionReport = "report.v1"
	schemaRefReport     = "contract/schema/report.v1.schema.json"
	jsonIndentPrefix    = ""
	jsonIndentValue     = "  "
)

const (
	reportDiagnosticAITasksMissing    = "report.ai_tasks_missing"
	reportDiagnosticAITasksReadFailed = "report.ai_tasks_read_failed"
	reportDiagnosticAITaskParseFailed = "report.ai_task_parse_failed"
)

const (
	inputStatusLoaded  = "loaded"
	inputStatusMissing = "missing"
	inputStatusFailed  = "failed"
)
