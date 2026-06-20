package migrate

const (
	commandUseMigrate   = "migrate"
	commandShortMigrate = "plan Nucleus version migrations and readiness checks"
	defaultDir          = "."
)

const (
	flagFromVersion = "from-version"
	flagToVersion   = "to-version"
	flagCheck       = "check"
	flagReport      = "report"
	flagJSON        = "json"
	flagPretty      = "pretty"
)

const (
	flagHelpFromVersion = "source Nucleus or manifest schema version"
	flagHelpToVersion   = "target Nucleus or manifest schema version"
	flagHelpCheck       = "fail when migration readiness checks do not pass"
	flagHelpReport      = "write the migration result JSON to this path"
	flagHelpJSON        = "emit machine-readable migration result"
	flagHelpPretty      = "pretty-print JSON output"
)

const (
	resultKindMigrate    = "nucleus.migrate_result"
	schemaVersionMigrate = "migrate.v1"
	schemaRefMigrate     = "contract/schema/migrate.schema.json"
	jsonIndentPrefix     = ""
	jsonIndentValue      = "  "
)

const (
	modePlan  = "plan"
	modeCheck = "check"
)

const (
	compatibilityNoChange       = "no_change"
	compatibilitySupported      = "supported"
	compatibilityGenericForward = "generic_forward"
	compatibilityUnsupported    = "unsupported"
)

const (
	diagnosticInspectFailed       = "migrate.inspect_failed"
	diagnosticInvalidVersion      = "migrate.version_invalid"
	diagnosticDowngrade           = "migrate.downgrade_unsupported"
	diagnosticGenericTransition   = "migrate.generic_transition"
	diagnosticGeneratedStale      = "migrate.generated_stale"
	diagnosticVerificationMissing = "migrate.verification_missing"
	diagnosticValidationFailed    = "migrate.validation_failed"
)

const (
	checkVersionOrder         = "version_order"
	checkRuleRegistry         = "rule_registry"
	checkManifestValidation   = "manifest_validation"
	checkContractInspection   = "contract_inspection"
	checkGeneratedFreshness   = "generated_freshness"
	checkVerificationCommands = "verification_commands"
)

const (
	stepInventory     = "inventory"
	stepManifest      = "manifest"
	stepContracts     = "contracts"
	stepGenerated     = "generated"
	stepCapabilities  = "capabilities"
	stepVerification  = "verification"
	stepNoChange      = "no_change"
	commandWorkingDir = "."
	writePolicyReport = "report_only"
	scopeLocalService = "local_service"
	severityInfo      = "info"
	severityWarning   = "warning"
	severityError     = "error"
)
