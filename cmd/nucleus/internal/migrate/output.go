package migrate

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
)

type migrateResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Mode          string                 `json:"mode"`
	Summary       migrateSummary         `json:"summary"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
	Migration     *migrationPlan         `json:"migration,omitempty"`
}

type migrateSummary struct {
	Service          string `json:"service,omitempty"`
	FromVersion      string `json:"from_version"`
	ToVersion        string `json:"to_version"`
	Compatibility    string `json:"compatibility"`
	Steps            int    `json:"steps"`
	RequiredEdits    int    `json:"required_edits"`
	AdvisoryEdits    int    `json:"advisory_edits"`
	Checks           int    `json:"checks"`
	Commands         int    `json:"commands"`
	Errors           int    `json:"errors"`
	Warnings         int    `json:"warnings"`
	GeneratedFresh   bool   `json:"generated_fresh"`
	ContractSurfaces int    `json:"contract_surfaces"`
	CapabilityCount  int    `json:"capability_count"`
}

type migrationPlan struct {
	Service              manifest.Service             `json:"service"`
	Scope                string                       `json:"scope"`
	WritePolicy          string                       `json:"write_policy"`
	FromVersion          versionInfo                  `json:"from_version"`
	ToVersion            versionInfo                  `json:"to_version"`
	Compatibility        string                       `json:"compatibility"`
	ContractFirst        bool                         `json:"contract_first"`
	DescriptionSchema    string                       `json:"description_schema"`
	RequiredEdits        []migrationEdit              `json:"required_edits"`
	AdvisoryEdits        []migrationEdit              `json:"advisory_edits"`
	Checks               []migrationCheck             `json:"checks"`
	Steps                []migrationStep              `json:"steps"`
	Commands             []migrationCommand           `json:"commands"`
	Risks                []migrationRisk              `json:"risks"`
	GeneratedFreshness   []inspect.GeneratedFreshness `json:"generated_freshness"`
	DeclaredCapabilities []string                     `json:"declared_capabilities"`
}

type versionInfo struct {
	Raw        string `json:"raw"`
	Normalized string `json:"normalized"`
	Known      bool   `json:"known"`
}

type migrationEdit struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
}

type migrationCheck struct {
	ID       string `json:"id"`
	Pass     bool   `json:"pass"`
	Severity string `json:"severity"`
	Subject  string `json:"subject,omitempty"`
	Reason   string `json:"reason"`
}

type migrationStep struct {
	ID       string   `json:"id"`
	Sequence int      `json:"sequence"`
	Title    string   `json:"title"`
	Purpose  string   `json:"purpose"`
	Edits    []string `json:"edits,omitempty"`
	Commands []string `json:"commands,omitempty"`
}

type migrationCommand struct {
	Phase      string `json:"phase"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
	Reason     string `json:"reason"`
}

type migrationRisk struct {
	ID         string `json:"id"`
	Severity   string `json:"severity"`
	Reason     string `json:"reason"`
	Mitigation string `json:"mitigation"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, result migrateResult) {
	for _, item := range result.Diagnostics {
		_, _ = fmt.Fprintf(stderr, "%s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
	if result.OK {
		_, _ = fmt.Fprintln(stdout, "OK")
	} else {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	}
	_, _ = fmt.Fprintf(stdout, "mode: %s\n", result.Mode)
	_, _ = fmt.Fprintf(stdout, "migration: %s -> %s\n", result.Summary.FromVersion, result.Summary.ToVersion)
	if result.Summary.Service != "" {
		_, _ = fmt.Fprintf(stdout, "service: %s\n", result.Summary.Service)
	}
	_, _ = fmt.Fprintf(stdout, "compatibility: %s\n", result.Summary.Compatibility)
	_, _ = fmt.Fprintf(stdout, "steps: %d\n", result.Summary.Steps)
	_, _ = fmt.Fprintf(stdout, "edits: %d required, %d advisory\n", result.Summary.RequiredEdits, result.Summary.AdvisoryEdits)
	_, _ = fmt.Fprintf(stdout, "checks: %d\n", result.Summary.Checks)
	if result.Migration != nil && len(result.Migration.Commands) > 0 {
		commands := make([]string, 0, len(result.Migration.Commands))
		for _, command := range result.Migration.Commands {
			commands = append(commands, command.Command)
		}
		_, _ = fmt.Fprintf(stdout, "commands: %s\n", strings.Join(commands, " -> "))
	}
	_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
}

func renderJSON(writer io.Writer, result migrateResult, pretty bool) error {
	result = finalizeResult(result)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}

func finalizeResult(result migrateResult) migrateResult {
	result.ResultKind = resultKindMigrate
	result.SchemaVersion = schemaVersionMigrate
	result.SchemaRef = schemaRefMigrate
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	result.Diagnostics.Sort()
	result.OK = !result.Diagnostics.Failed()
	result.Summary.Errors = result.Diagnostics.Count(diagnostic.SeverityError)
	result.Summary.Warnings = result.Diagnostics.Count(diagnostic.SeverityWarning)
	return result
}

func errorDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}

func warningDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
