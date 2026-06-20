package capability

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
)

type capabilityResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Mode          string                 `json:"mode"`
	Capability    string                 `json:"capability,omitempty"`
	Provider      string                 `json:"provider,omitempty"`
	DryRun        bool                   `json:"dry_run,omitempty"`
	Forced        bool                   `json:"forced,omitempty"`
	Summary       capabilitySummary      `json:"summary"`
	Files         []fileChange           `json:"files"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
	NextSteps     []string               `json:"next_steps,omitempty"`
}

type capabilitySummary struct {
	Changed   int `json:"changed"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Conflicts int `json:"conflicts"`
	Errors    int `json:"errors"`
	Warnings  int `json:"warnings"`
}

type fileChange struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

func renderJSON(writer io.Writer, result capabilityResult, pretty bool) error {
	result = normalizeResult(result)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}

func renderHuman(stdout io.Writer, stderr io.Writer, result capabilityResult) {
	status := "OK"
	if !result.OK {
		status = "FAILED"
	}
	if result.DryRun && result.OK {
		status = "DRY-RUN"
	}
	_, _ = fmt.Fprintf(stdout, "%s capability %s provider=%s\n", status, result.Capability, result.Provider)
	if len(result.Files) > 0 {
		_, _ = fmt.Fprintln(stdout, "files:")
		for _, file := range result.Files {
			_, _ = fmt.Fprintf(stdout, "  - %s %s\n", file.Action, file.Path)
		}
	}
	if len(result.NextSteps) > 0 {
		_, _ = fmt.Fprintln(stdout, "next steps:")
		for _, step := range result.NextSteps {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", step)
		}
	}
	for _, item := range result.Diagnostics {
		if item.Severity == diagnostic.SeverityError {
			_, _ = fmt.Fprintf(stderr, "error: %s\n", item.Message)
		}
	}
}

func normalizeResult(result capabilityResult) capabilityResult {
	if result.ResultKind == "" {
		result.ResultKind = resultKindCapability
	}
	if result.SchemaVersion == "" {
		result.SchemaVersion = schemaVersion
	}
	if result.SchemaRef == "" {
		result.SchemaRef = schemaRefEvidence
	}
	if result.Mode == "" {
		result.Mode = "add"
	}
	if result.Files == nil {
		result.Files = []fileChange{}
	}
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	return result
}

func summarize(files []fileChange, diagnostics diagnostic.Diagnostics) capabilitySummary {
	summary := capabilitySummary{
		Errors:   diagnostics.Count(diagnostic.SeverityError),
		Warnings: diagnostics.Count(diagnostic.SeverityWarning),
	}
	for _, file := range files {
		switch file.Action {
		case actionCreated, actionWouldCreate:
			summary.Created++
			summary.Changed++
		case actionUpdated, actionWouldUpdate:
			summary.Updated++
			summary.Changed++
		case actionUnchanged:
			summary.Unchanged++
		case actionConflict:
			summary.Conflicts++
		}
	}
	return summary
}
