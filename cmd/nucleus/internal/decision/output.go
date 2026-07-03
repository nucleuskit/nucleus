package decision

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
)

type result struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Summary       summary                `json:"summary"`
	Decisions     []fileSummary          `json:"decisions"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

type summary struct {
	Files    int `json:"files"`
	Valid    int `json:"valid"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// QualitySummary is a compact, report/verify friendly decision health view.
type QualitySummary struct {
	Files          int                    `json:"files"`
	Valid          int                    `json:"valid"`
	Errors         int                    `json:"errors"`
	Warnings       int                    `json:"warnings"`
	AcceptedLocked int                    `json:"accepted_locked"`
	Supersedes     int                    `json:"supersedes"`
	Drift          int                    `json:"drift"`
	Diagnostics    diagnostic.Diagnostics `json:"diagnostics,omitempty"`
}

type fileSummary struct {
	Path       string `json:"path"`
	ID         string `json:"id,omitempty"`
	Capability string `json:"capability,omitempty"`
	Status     string `json:"status,omitempty"`
	Locked     bool   `json:"locked"`
	Hash       string `json:"hash,omitempty"`
}

type actionResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Action        string                 `json:"action"`
	Path          string                 `json:"path,omitempty"`
	Changed       bool                   `json:"changed"`
	Decision      fileSummary            `json:"decision,omitempty"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

func renderJSON(writer io.Writer, output result, pretty bool) error {
	output = normalizeResult(output)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(output)
}

func renderActionJSON(writer io.Writer, output actionResult, pretty bool) error {
	output = normalizeActionResult(output)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(output)
}

func renderActionHuman(writer io.Writer, output actionResult) {
	output = normalizeActionResult(output)
	if output.OK {
		_, _ = fmt.Fprintf(writer, "OK decision %s\n", output.Action)
	} else {
		_, _ = fmt.Fprintf(writer, "FAILED decision %s\n", output.Action)
	}
	if output.Path != "" {
		_, _ = fmt.Fprintf(writer, "path: %s\n", output.Path)
	}
	if output.Decision.Hash != "" {
		_, _ = fmt.Fprintf(writer, "hash: %s\n", output.Decision.Hash)
	}
	for _, item := range output.Diagnostics {
		_, _ = fmt.Fprintf(writer, "  - %s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
}

func renderHuman(writer io.Writer, output result) {
	output = normalizeResult(output)
	if output.OK {
		_, _ = fmt.Fprintln(writer, "OK decisions")
	} else {
		_, _ = fmt.Fprintln(writer, "FAILED decisions")
	}
	_, _ = fmt.Fprintf(writer, "files: %d\n", output.Summary.Files)
	_, _ = fmt.Fprintf(writer, "diagnostics: %d errors, %d warnings\n", output.Summary.Errors, output.Summary.Warnings)
	for _, item := range output.Diagnostics {
		_, _ = fmt.Fprintf(writer, "  - %s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
}

func normalizeResult(output result) result {
	if output.Decisions == nil {
		output.Decisions = []fileSummary{}
	}
	if output.Diagnostics == nil {
		output.Diagnostics = diagnostic.Diagnostics{}
	}
	output.ResultKind = resultKindDecision
	output.SchemaVersion = schemaVersionDecision
	output.SchemaRef = schemaRefDecision
	output.OK = !output.Diagnostics.Failed()
	output.Summary = summary{
		Files:    len(output.Decisions),
		Valid:    validDecisionCount(output.Decisions, output.Diagnostics),
		Errors:   output.Diagnostics.Count(diagnostic.SeverityError),
		Warnings: output.Diagnostics.Count(diagnostic.SeverityWarning),
	}
	return output
}

func normalizeActionResult(output actionResult) actionResult {
	if output.Diagnostics == nil {
		output.Diagnostics = diagnostic.Diagnostics{}
	}
	if output.ResultKind == "" {
		switch output.Action {
		case "accept":
			output.ResultKind = resultKindAccept
		case "supersede":
			output.ResultKind = resultKindSupersede
		}
	}
	output.SchemaVersion = schemaVersionAction
	output.SchemaRef = schemaRefDecision
	output.OK = !output.Diagnostics.Failed()
	return output
}

func validDecisionCount(decisions []fileSummary, diagnostics diagnostic.Diagnostics) int {
	invalid := map[string]bool{}
	for _, item := range diagnostics {
		if item.Severity == diagnostic.SeverityError && item.Path != "" {
			invalid[item.Path] = true
		}
	}
	count := 0
	for _, decision := range decisions {
		if !invalid[decision.Path] {
			count++
		}
	}
	return count
}
