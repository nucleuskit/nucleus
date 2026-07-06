package decision

const (
	commandUseDecision      = "decision"
	commandShortDecision    = "validate structured Nucleus decisions"
	commandUseValidate      = "validate [path ...]"
	commandShortValidate    = "validate decision evidence files"
	commandUseAccept        = "accept <path>"
	commandShortAccept      = "accept and lock one decision file"
	commandUseSupersede     = "supersede <path>"
	commandShortSupersede   = "fill supersedes_hash for one supersede decision"
	defaultDir              = "."
	defaultDecisionDir      = ".nucleus/decisions"
	resultKindDecision      = "nucleus.decision_validate_result"
	resultKindAccept        = "nucleus.decision_accept_result"
	resultKindSupersede     = "nucleus.decision_supersede_result"
	schemaVersionDecision   = "decision-result.v1"
	schemaVersionAction     = "decision-result.v1"
	schemaRefDecision       = "contract/schema/decision-result.v1.schema.json"
	jsonIndentPrefix        = ""
	jsonIndentValue         = "  "
	decisionSchemaVersion   = "decision.v1"
	decisionHashAlgorithm   = "sha256"
	diagnosticPathField     = "path"
	decisionFileKind        = "decision"
	decisionStatusProposed  = "proposed"
	decisionStatusAccepted  = "accepted"
	decisionStatusSupersede = "superseded"
)

const (
	flagJSON       = "json"
	flagPretty     = "pretty"
	flagAcceptedBy = "by"
	flagAcceptedAt = "accepted-at"
)

const (
	flagHelpJSON       = "emit machine-readable decision result"
	flagHelpPretty     = "pretty-print JSON output"
	flagHelpAcceptedBy = "accepted-by value for locked decisions"
	flagHelpAcceptedAt = "RFC3339 accepted_at timestamp; defaults to current UTC time"
)
