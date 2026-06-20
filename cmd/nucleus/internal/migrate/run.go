package migrate

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/validation"
)

type parsedVersion struct {
	raw        string
	normalized string
	major      int
	minor      int
	patch      int
	known      bool
}

type migrationRule struct {
	fromMajor int
	fromMinor int
	toMajor   int
	toMinor   int
}

var knownRules = []migrationRule{
	{fromMajor: 0, fromMinor: 1, toMajor: 0, toMinor: 2},
	{fromMajor: 0, fromMinor: 2, toMajor: 0, toMinor: 3},
	{fromMajor: 1, fromMinor: 0, toMajor: 1, toMinor: 1},
}

func run(config Config, opts *options) (migrateResult, error) {
	if err := validateOptions(opts); err != nil {
		return migrateResult{}, err
	}
	dir := stringValue(config.Dir, defaultDir)
	mode := modePlan
	if opts.check {
		mode = modeCheck
	}
	from, fromErr := parseVersion(opts.fromVersion)
	to, toErr := parseVersion(opts.toVersion)

	description, inspectErr := inspect.Describe(dir)
	if inspectErr != nil {
		result := migrateResult{
			Mode: mode,
			Summary: migrateSummary{
				FromVersion:   strings.TrimSpace(opts.fromVersion),
				ToVersion:     strings.TrimSpace(opts.toVersion),
				Compatibility: compatibilityUnsupported,
			},
			Diagnostics: diagnostic.Diagnostics{
				errorDiagnostic(safeMigratePath(dir), diagnosticInspectFailed, safeErrorMessage(inspectErr)),
			},
		}
		return finalizeResult(result), nil
	}

	validationPassed, diagnostics := validationDiagnostics(dir, mode)
	if fromErr != nil {
		diagnostics = append(diagnostics, errorDiagnostic(flagFromVersion, diagnosticInvalidVersion, fromErr.Error()))
	}
	if toErr != nil {
		diagnostics = append(diagnostics, errorDiagnostic(flagToVersion, diagnosticInvalidVersion, toErr.Error()))
	}
	if fromErr == nil && toErr == nil {
		diagnostics = append(diagnostics, versionDiagnostics(from, to)...)
	}

	compatibility := compatibilityForVersions(from, to, fromErr, toErr)
	plan := buildMigrationPlan(description, from, to, compatibility, mode, validationPassed)
	diagnostics = append(diagnostics, checkDiagnostics(plan.Checks, mode)...)
	result := migrateResult{
		Mode:        mode,
		Summary:     buildSummary(plan, diagnostics),
		Diagnostics: diagnostics,
		Migration:   &plan,
	}
	return finalizeResult(result), nil
}

func validateOptions(opts *options) error {
	if strings.TrimSpace(opts.fromVersion) == "" {
		return fmt.Errorf("--%s is required", flagFromVersion)
	}
	if strings.TrimSpace(opts.toVersion) == "" {
		return fmt.Errorf("--%s is required", flagToVersion)
	}
	return nil
}

func parseVersion(value string) (parsedVersion, error) {
	raw := strings.TrimSpace(value)
	trimmed := strings.TrimPrefix(raw, "v")
	if trimmed == "" {
		return parsedVersion{raw: raw, normalized: raw}, fmt.Errorf("version %q is empty", raw)
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return parsedVersion{raw: raw, normalized: raw}, fmt.Errorf("version %q must look like MAJOR.MINOR or MAJOR.MINOR.PATCH", raw)
	}
	numbers := []int{0, 0, 0}
	for index, part := range parts {
		if part == "" {
			return parsedVersion{raw: raw, normalized: raw}, fmt.Errorf("version %q contains an empty component", raw)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return parsedVersion{raw: raw, normalized: raw}, fmt.Errorf("version %q must contain only non-negative numeric components", raw)
		}
		numbers[index] = number
	}
	normalized := fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2])
	return parsedVersion{
		raw:        raw,
		normalized: normalized,
		major:      numbers[0],
		minor:      numbers[1],
		patch:      numbers[2],
		known:      isKnownVersion(numbers[0], numbers[1]),
	}, nil
}

func isKnownVersion(major int, minor int) bool {
	for _, rule := range knownRules {
		if rule.fromMajor == major && rule.fromMinor == minor {
			return true
		}
		if rule.toMajor == major && rule.toMinor == minor {
			return true
		}
	}
	return false
}

func compareVersions(from parsedVersion, to parsedVersion) int {
	for _, pair := range [][2]int{{from.major, to.major}, {from.minor, to.minor}, {from.patch, to.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

func compatibilityForVersions(from parsedVersion, to parsedVersion, fromErr error, toErr error) string {
	if fromErr != nil || toErr != nil {
		return compatibilityUnsupported
	}
	switch compareVersions(from, to) {
	case 0:
		return compatibilityNoChange
	case 1:
		return compatibilityUnsupported
	}
	if hasExactRule(from, to) {
		return compatibilitySupported
	}
	return compatibilityGenericForward
}

func hasExactRule(from parsedVersion, to parsedVersion) bool {
	for _, rule := range knownRules {
		if rule.fromMajor == from.major && rule.fromMinor == from.minor && rule.toMajor == to.major && rule.toMinor == to.minor {
			return true
		}
	}
	return false
}

func versionDiagnostics(from parsedVersion, to parsedVersion) diagnostic.Diagnostics {
	switch compareVersions(from, to) {
	case 1:
		return diagnostic.Diagnostics{
			errorDiagnostic(flagToVersion, diagnosticDowngrade, "downgrade migrations are not supported"),
		}
	case 0:
		return nil
	}
	if hasExactRule(from, to) {
		return nil
	}
	return diagnostic.Diagnostics{
		warningDiagnostic(flagToVersion, diagnosticGenericTransition, "no exact migration rule is registered; using the generic forward migration checklist"),
	}
}

func validationDiagnostics(dir string, mode string) (bool, diagnostic.Diagnostics) {
	diagnostics := validation.ValidateService(dir)
	passed := !diagnostics.Failed()
	var mapped diagnostic.Diagnostics
	for _, item := range diagnostics {
		severity := item.Severity
		if mode == modePlan && severity == diagnostic.SeverityError {
			severity = diagnostic.SeverityWarning
		}
		mapped = append(mapped, diagnostic.Diagnostic{
			Severity: severity,
			Code:     diagnosticValidationFailed,
			Path:     item.Path,
			Message:  item.Code + ": " + item.Message,
		})
	}
	return passed, mapped
}

func buildMigrationPlan(description inspect.Description, from parsedVersion, to parsedVersion, compatibility string, mode string, validationPassed bool) migrationPlan {
	requiredEdits, advisoryEdits := migrationEdits(description, compatibility)
	commands := migrationCommands(description)
	checks := migrationChecks(description, compatibility, mode, validationPassed)
	steps := migrationSteps(requiredEdits, advisoryEdits, commands, compatibility)
	return migrationPlan{
		Service:              description.Service,
		Scope:                scopeLocalService,
		WritePolicy:          writePolicyReport,
		FromVersion:          toVersionInfo(from),
		ToVersion:            toVersionInfo(to),
		Compatibility:        compatibility,
		ContractFirst:        true,
		DescriptionSchema:    description.SchemaVersion,
		RequiredEdits:        nonNilEdits(requiredEdits),
		AdvisoryEdits:        nonNilEdits(advisoryEdits),
		Checks:               nonNilChecks(checks),
		Steps:                nonNilSteps(steps),
		Commands:             nonNilCommands(commands),
		Risks:                nonNilRisks(migrationRisks(description, compatibility)),
		GeneratedFreshness:   append([]inspect.GeneratedFreshness{}, description.GeneratedFreshness...),
		DeclaredCapabilities: append([]string{}, description.Capabilities...),
	}
}

func toVersionInfo(version parsedVersion) versionInfo {
	return versionInfo{
		Raw:        version.raw,
		Normalized: version.normalized,
		Known:      version.known,
	}
}

func migrationEdits(description inspect.Description, compatibility string) ([]migrationEdit, []migrationEdit) {
	if compatibility == compatibilityNoChange {
		return nil, []migrationEdit{
			{Path: "nucleus.yaml", Kind: "manifest", Required: false, Reason: "confirm service metadata already matches the target version"},
		}
	}
	required := []migrationEdit{
		{Path: "nucleus.yaml", Kind: "manifest", Required: true, Reason: "record target Nucleus compatibility and service metadata changes"},
		{Path: "AGENTS.md", Kind: "agent_contract", Required: true, Reason: "keep AI edit boundaries and verification instructions aligned with the target version"},
	}
	if len(description.Endpoints) > 0 {
		required = append(required, migrationEdit{Path: "api/openapi.yaml", Kind: "http_contract", Required: true, Reason: "review HTTP contract compatibility before regenerating service code"})
	}
	for _, service := range description.GRPCServices {
		required = append(required, migrationEdit{Path: service.Source, Kind: "grpc_contract", Required: true, Reason: "review gRPC contract compatibility before regenerating service code"})
	}
	if len(description.ErrorCodes) > 0 {
		required = append(required, migrationEdit{Path: "api/errors.yaml", Kind: "error_contract", Required: true, Reason: "preserve stable external error mapping during migration"})
	}
	for _, item := range description.GeneratedFreshness {
		required = append(required, migrationEdit{Path: item.Target, Kind: "generated", Required: true, Reason: "regenerate after contract or manifest changes"})
	}
	advisory := []migrationEdit{
		{Path: "docs/**", Kind: "documentation", Required: false, Reason: "document behavior or compatibility changes visible to service owners"},
	}
	if len(description.Capabilities) > 0 {
		advisory = append(advisory,
			migrationEdit{Path: "go.mod", Kind: "dependency", Required: false, Reason: "confirm module requirements match capability adapters used by the target version"},
			migrationEdit{Path: "configs/**", Kind: "configuration", Required: false, Reason: "review provider configuration keys and defaults"},
		)
	}
	return uniqueEdits(required), uniqueEdits(advisory)
}

func migrationChecks(description inspect.Description, compatibility string, mode string, validationPassed bool) []migrationCheck {
	generatedFresh := generatedFreshnessPass(description.GeneratedFreshness)
	commandsDeclared := len(description.Verification.Commands) > 0
	rulePass := compatibility == compatibilitySupported || compatibility == compatibilityNoChange
	return []migrationCheck{
		{ID: checkContractInspection, Pass: true, Severity: severityInfo, Subject: "contract/inspect", Reason: "service manifest and contract facts loaded"},
		{ID: checkVersionOrder, Pass: compatibility != compatibilityUnsupported, Severity: checkSeverity(compatibility != compatibilityUnsupported, severityError), Subject: "version", Reason: chooseCheckReason(compatibility != compatibilityUnsupported, "target version is not older than source version", "target version is older than source version or invalid")},
		{ID: checkRuleRegistry, Pass: rulePass, Severity: checkSeverity(rulePass, severityWarning), Subject: "migration_rules", Reason: chooseCheckReason(rulePass, "exact migration rule is registered", "using generic forward migration checklist")},
		{ID: checkManifestValidation, Pass: validationPassed, Severity: checkSeverity(validationPassed, chooseModeSeverity(mode)), Subject: "nucleus.yaml", Reason: chooseCheckReason(validationPassed, "manifest and contract validation passed", "manifest or contract validation diagnostics are emitted separately")},
		{ID: checkGeneratedFreshness, Pass: generatedFresh, Severity: checkSeverity(generatedFresh, chooseModeSeverity(mode)), Subject: "ai.generated", Reason: chooseCheckReason(generatedFresh, "generated targets are fresh or not declared", "generated targets are stale or missing freshness markers")},
		{ID: checkVerificationCommands, Pass: commandsDeclared, Severity: checkSeverity(commandsDeclared, chooseModeSeverity(mode)), Subject: "verification", Reason: chooseCheckReason(commandsDeclared, "verification commands are declared", "verification commands are missing")},
	}
}

func migrationSteps(requiredEdits []migrationEdit, advisoryEdits []migrationEdit, commands []migrationCommand, compatibility string) []migrationStep {
	if compatibility == compatibilityNoChange {
		return []migrationStep{
			{ID: stepNoChange, Sequence: 1, Title: "Confirm no-op migration", Purpose: "Verify source and target versions already match.", Commands: commandStrings(commands)},
		}
	}
	contractEdits := editPathsByKind(requiredEdits, "http_contract", "grpc_contract", "error_contract")
	generatedEdits := editPathsByKind(requiredEdits, "generated")
	capabilityEdits := editPathsByKind(advisoryEdits, "dependency", "configuration")
	return []migrationStep{
		{ID: stepInventory, Sequence: 1, Title: "Inventory current service facts", Purpose: "Load manifest, contracts, capabilities, generated freshness, and verification metadata before editing.", Commands: []string{"nucleus describe --dir . --json"}},
		{ID: stepManifest, Sequence: 2, Title: "Update manifest and agent contract", Purpose: "Move version metadata and AI-safe edit boundaries first so subsequent edits remain auditable.", Edits: editPathsByKind(requiredEdits, "manifest", "agent_contract")},
		{ID: stepContracts, Sequence: 3, Title: "Review contract surfaces", Purpose: "Apply HTTP, gRPC, and error catalog compatibility changes before generated code refresh.", Edits: contractEdits},
		{ID: stepGenerated, Sequence: 4, Title: "Refresh generated artifacts", Purpose: "Regenerate derived code and metadata after contract-first changes.", Edits: generatedEdits, Commands: []string{"nucleus gen --dir ."}},
		{ID: stepCapabilities, Sequence: 5, Title: "Review capability wiring", Purpose: "Confirm optional provider and configuration wiring without leaking provider SDKs into kernel layers.", Edits: capabilityEdits},
		{ID: stepVerification, Sequence: 6, Title: "Run AI-safe verification loop", Purpose: "Validate, lint, and verify the migrated service with stable evidence.", Commands: commandStrings(commands)},
	}
}

func migrationCommands(description inspect.Description) []migrationCommand {
	commands := []migrationCommand{
		{Phase: "validate", Command: "nucleus validate --dir . --json", WorkingDir: commandWorkingDir, Reason: "validate manifest and contract sources"},
		{Phase: "lint", Command: "nucleus lint --dir . --strict --json", WorkingDir: commandWorkingDir, Reason: "enforce strict contract and capability rules"},
		{Phase: "verify", Command: "nucleus verify --dir . --json", WorkingDir: commandWorkingDir, Reason: "run the full verification gate and emit evidence"},
	}
	for _, command := range description.Verification.Commands {
		if !containsCommand(commands, command) {
			commands = append(commands, migrationCommand{Phase: "declared", Command: command, WorkingDir: commandWorkingDir, Reason: "declared by service inspection metadata"})
		}
	}
	return commands
}

func migrationRisks(description inspect.Description, compatibility string) []migrationRisk {
	risks := []migrationRisk{
		{ID: "manual_code_changes", Severity: severityInfo, Reason: "migrate is report-only and does not rewrite service code", Mitigation: "apply edits through describe -> plan -> gen -> lint -> verify"},
		{ID: "version_rule_coverage", Severity: chooseRiskSeverity(compatibility == compatibilityGenericForward), Reason: chooseCheckReason(compatibility == compatibilityGenericForward, "no exact version rule is registered", "exact or no-op migration path is available"), Mitigation: "review generic checklist before treating the result as release evidence"},
	}
	if !generatedFreshnessPass(description.GeneratedFreshness) {
		risks = append(risks, migrationRisk{ID: "stale_generated_artifacts", Severity: severityWarning, Reason: "one or more generated targets are stale or missing freshness metadata", Mitigation: "run nucleus gen and nucleus verify before migration sign-off"})
	}
	if len(description.Capabilities) > 0 {
		risks = append(risks, migrationRisk{ID: "capability_provider_wiring", Severity: severityInfo, Reason: "capabilities may involve optional provider adapters outside core contracts", Mitigation: "keep provider-specific options in bridge or service wiring"})
	}
	return risks
}

func buildSummary(plan migrationPlan, diagnostics diagnostic.Diagnostics) migrateSummary {
	contractSurfaces := 0
	for _, edit := range plan.RequiredEdits {
		if strings.Contains(edit.Kind, "contract") {
			contractSurfaces++
		}
	}
	return migrateSummary{
		Service:          plan.Service.Name,
		FromVersion:      plan.FromVersion.Normalized,
		ToVersion:        plan.ToVersion.Normalized,
		Compatibility:    plan.Compatibility,
		Steps:            len(plan.Steps),
		RequiredEdits:    len(plan.RequiredEdits),
		AdvisoryEdits:    len(plan.AdvisoryEdits),
		Checks:           len(plan.Checks),
		Commands:         len(plan.Commands),
		Errors:           diagnostics.Count(diagnostic.SeverityError),
		Warnings:         diagnostics.Count(diagnostic.SeverityWarning),
		GeneratedFresh:   generatedFreshnessPass(plan.GeneratedFreshness),
		ContractSurfaces: contractSurfaces,
		CapabilityCount:  len(plan.DeclaredCapabilities),
	}
}

func checkDiagnostics(checks []migrationCheck, mode string) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, check := range checks {
		if check.Pass {
			continue
		}
		switch check.ID {
		case checkGeneratedFreshness:
			if mode == modeCheck {
				diagnostics = append(diagnostics, errorDiagnostic(check.Subject, diagnosticGeneratedStale, check.Reason))
			} else {
				diagnostics = append(diagnostics, warningDiagnostic(check.Subject, diagnosticGeneratedStale, check.Reason))
			}
		case checkVerificationCommands:
			if mode == modeCheck {
				diagnostics = append(diagnostics, errorDiagnostic(check.Subject, diagnosticVerificationMissing, check.Reason))
			} else {
				diagnostics = append(diagnostics, warningDiagnostic(check.Subject, diagnosticVerificationMissing, check.Reason))
			}
		}
	}
	return diagnostics
}

func generatedFreshnessPass(items []inspect.GeneratedFreshness) bool {
	for _, item := range items {
		if !item.Fresh {
			return false
		}
	}
	return true
}

func chooseModeSeverity(mode string) string {
	if mode == modeCheck {
		return severityError
	}
	return severityWarning
}

func checkSeverity(pass bool, failureSeverity string) string {
	if pass {
		return severityInfo
	}
	return failureSeverity
}

func chooseRiskSeverity(condition bool) string {
	if condition {
		return severityWarning
	}
	return severityInfo
}

func chooseCheckReason(ok bool, pass string, fail string) string {
	if ok {
		return pass
	}
	return fail
}

func uniqueEdits(values []migrationEdit) []migrationEdit {
	seen := map[string]struct{}{}
	var unique []migrationEdit
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		if path == "" {
			continue
		}
		path = filepath.ToSlash(path)
		key := value.Kind + "\x00" + path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value.Path = path
		unique = append(unique, value)
	}
	sort.SliceStable(unique, func(i, j int) bool {
		if unique[i].Kind == unique[j].Kind {
			return unique[i].Path < unique[j].Path
		}
		return unique[i].Kind < unique[j].Kind
	})
	return unique
}

func editPathsByKind(edits []migrationEdit, kinds ...string) []string {
	allowed := map[string]struct{}{}
	for _, kind := range kinds {
		allowed[kind] = struct{}{}
	}
	var paths []string
	for _, edit := range edits {
		if _, ok := allowed[edit.Kind]; ok {
			paths = append(paths, edit.Path)
		}
	}
	return uniqueStrings(paths)
}

func commandStrings(commands []migrationCommand) []string {
	values := make([]string, 0, len(commands))
	for _, command := range commands {
		values = append(values, command.Command)
	}
	return uniqueStrings(values)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var unique []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func containsCommand(commands []migrationCommand, command string) bool {
	for _, existing := range commands {
		if existing.Command == command {
			return true
		}
	}
	return false
}

func nonNilEdits(values []migrationEdit) []migrationEdit {
	if values == nil {
		return []migrationEdit{}
	}
	return values
}

func nonNilChecks(values []migrationCheck) []migrationCheck {
	if values == nil {
		return []migrationCheck{}
	}
	return values
}

func nonNilSteps(values []migrationStep) []migrationStep {
	if values == nil {
		return []migrationStep{}
	}
	return values
}

func nonNilCommands(values []migrationCommand) []migrationCommand {
	if values == nil {
		return []migrationCommand{}
	}
	return values
}

func nonNilRisks(values []migrationRisk) []migrationRisk {
	if values == nil {
		return []migrationRisk{}
	}
	return values
}
